package tenant

import (
	"errors"
	"fmt"
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/pkg/testmysql"

	"gorm.io/gorm"
)

// ─────────────────────────────────────────────────────────────
// Casbin domain subject expansion (contract §4)
// ─────────────────────────────────────────────────────────────

func TestRoleSubjectFormat(t *testing.T) {
	got := RoleSubject("editor", 101)
	want := "role:editor@tenant:101"
	if got != want {
		t.Fatalf("RoleSubject = %q, want %q", got, want)
	}
}

func TestGlobalRoleSubjectTrims(t *testing.T) {
	if got := GlobalRoleSubject("  " + MembershipRoleAdmin + "  "); got != MembershipRoleAdmin {
		t.Fatalf("GlobalRoleSubject = %q, want admin", got)
	}
}

func TestCasbinDomainPolicySubjectsCompat(t *testing.T) {
	ctx := &Context{TenantID: PlatformGlobalTenantID, Mode: ModeCompat}
	got := CasbinDomainPolicySubjects(MembershipRoleAdmin, ctx)
	if len(got) != 1 || got[0] != MembershipRoleAdmin {
		t.Fatalf("compat subjects = %v, want [admin]", got)
	}
}

func TestCasbinDomainPolicySubjectsNilContext(t *testing.T) {
	got := CasbinDomainPolicySubjects(MembershipRoleAdmin, nil)
	if len(got) != 1 || got[0] != MembershipRoleAdmin {
		t.Fatalf("nil-context subjects = %v, want [admin]", got)
	}
}

func TestCasbinDomainPolicySubjectsMulti(t *testing.T) {
	ctx := &Context{TenantID: 202, Mode: ModeMulti, ResolvedBy: "subject"}
	got := CasbinDomainPolicySubjects("editor", ctx)
	if len(got) != 2 || got[0] != "editor" || got[1] != "role:editor@tenant:202" {
		t.Fatalf("multi subjects = %v, want [editor role:editor@tenant:202]", got)
	}
}

// Contract §4: global subject first, domain subject second — never merged.
func TestCasbinDomainPolicySubjectsOrder(t *testing.T) {
	ctx := &Context{TenantID: 7, Mode: ModeMulti}
	got := CasbinDomainPolicySubjects("ops", ctx)
	if got[0] != "ops" || got[1] != "role:ops@tenant:7" {
		t.Fatalf("subject order = %v, want global first", got)
	}
}

// ─────────────────────────────────────────────────────────────
// Session claim normalization
// ─────────────────────────────────────────────────────────────

func TestSessionTenantClaim(t *testing.T) {
	if got := SessionTenantClaim(PlatformGlobalTenantID); got != 0 {
		t.Fatalf("SessionTenantClaim(0) = %d, want 0", got)
	}
	if got := SessionTenantClaim(101); got != 101 {
		t.Fatalf("SessionTenantClaim(101) = %d, want 101", got)
	}
}

// ─────────────────────────────────────────────────────────────
// Tenant gate error classification (contract §5)
// ─────────────────────────────────────────────────────────────

func TestIsTenantGateError(t *testing.T) {
	cases := map[error]bool{
		ErrTenantForbidden:      true,
		ErrTenantSuspended:      true,
		ErrTenantArchived:       true,
		ErrTenantContextMissing: true,
		errors.New("other"):     false,
		nil:                     false,
	}
	for err, want := range cases {
		if got := IsTenantGateError(err); got != want {
			t.Fatalf("IsTenantGateError(%v) = %v, want %v", err, got, want)
		}
	}
}

// Wrapped sentinels stay classified (errors.Is chain).
func TestIsTenantGateErrorWrapped(t *testing.T) {
	wrapped := errors.Join(ErrTenantForbidden)
	if !IsTenantGateError(wrapped) {
		t.Fatal("wrapped ErrTenantForbidden not classified")
	}
}

// ─────────────────────────────────────────────────────────────
// W4 (W3 decision A): the role-presence predicate
// ─────────────────────────────────────────────────────────────

// gateAuthzFixture migrates the membership/tenant tables plus a raw
// system_user_role table. pkg/tenant must not import the IAM user module
// (import direction), and the predicate reads the table by name anyway, so the
// test schema carries only the columns the predicate queries.
func gateAuthzFixture(t *testing.T) *gorm.DB {
	t.Helper()
	db := testmysql.Open(t)
	if err := db.AutoMigrate(&Membership{}, &Tenant{}); err != nil {
		t.Fatalf("migrate tenant tables: %v", err)
	}
	if err := db.Exec("CREATE TABLE IF NOT EXISTS system_user_role (" +
		"tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0, " +
		"user_id BIGINT UNSIGNED NOT NULL, " +
		"role_id BIGINT UNSIGNED NOT NULL, " +
		"PRIMARY KEY (tenant_id, user_id, role_id))").Error; err != nil {
		t.Fatalf("create system_user_role: %v", err)
	}
	return db
}

func gateAuthzSeedTenant(t *testing.T, db *gorm.DB, id uint64, status string) {
	t.Helper()
	row := Tenant{ID: id, Code: fmt.Sprintf("t%d", id), Name: fmt.Sprintf("t%d", id), Status: status}
	if err := db.Create(&row).Error; err != nil {
		t.Fatalf("seed tenant %d: %v", id, err)
	}
}

func gateAuthzSeedMembership(t *testing.T, db *gorm.DB, tenantID, userID uint64, status string) {
	t.Helper()
	row := Membership{TenantID: tenantID, UserID: userID, Role: MembershipRoleMember, Status: status}
	if err := db.Create(&row).Error; err != nil {
		t.Fatalf("seed membership t%d u%d: %v", tenantID, userID, err)
	}
}

func gateAuthzSeedBinding(t *testing.T, db *gorm.DB, tenantID, userID uint64) {
	t.Helper()
	if err := db.Exec("INSERT INTO system_user_role (tenant_id, user_id, role_id) VALUES (?, ?, 1)", tenantID, userID).Error; err != nil {
		t.Fatalf("seed role binding t%d u%d: %v", tenantID, userID, err)
	}
}

// HasRoleBindingInTenant is the single role-presence predicate behind login
// discovery, the candidate list and the switchable list. Every guard fails
// closed: a nil handle, a zero identifier, a missing table and a query error
// all read as "no binding".
func TestHasRoleBindingInTenantFailClosed(t *testing.T) {
	if HasRoleBindingInTenant(nil, 42, 101) {
		t.Fatal("nil db must read as no binding")
	}
	db := gateAuthzFixture(t)
	gateAuthzSeedTenant(t, db, 101, TenantStatusActive)
	gateAuthzSeedMembership(t, db, 101, 42, MembershipActive)
	gateAuthzSeedBinding(t, db, 101, 42)

	if HasRoleBindingInTenant(db, 0, 101) || HasRoleBindingInTenant(db, 42, 0) {
		t.Fatal("zero identifiers must read as no binding")
	}
	if !HasRoleBindingInTenant(db, 42, 101) {
		t.Fatal("existing binding not found")
	}
	if HasRoleBindingInTenant(db, 42, 202) {
		t.Fatal("unbound tenant read as bound")
	}

	// A schema without the table (pre-migration deployment) fails closed too.
	bare := testmysql.Open(t)
	if HasRoleBindingInTenant(bare, 42, 101) {
		t.Fatal("missing system_user_role table must read as no binding")
	}
}

// A membership without a role binding must not resolve at discovery (TC-01:
// it used to hand the subject a signed-in console with zero roles). The filter
// runs before the ambiguity logic, so one bound + one unbound membership
// resolves the bound one instead of denying; two bound memberships deny.
func TestDiscoverDefaultTenantFiltersMembershipWithoutRoles(t *testing.T) {
	db := gateAuthzFixture(t)
	gateAuthzSeedTenant(t, db, 101, TenantStatusActive)
	gateAuthzSeedTenant(t, db, 202, TenantStatusActive)
	gateAuthzSeedMembership(t, db, 101, 42, MembershipActive)
	gateAuthzSeedMembership(t, db, 202, 42, MembershipActive)
	gateAuthzSeedBinding(t, db, 101, 42)

	discovery, err := DiscoverDefaultTenant(db, 42)
	if err != nil {
		t.Fatalf("discovery with one bound membership: %v", err)
	}
	if !discovery.Resolved || discovery.TenantID != 101 {
		t.Fatalf("discovery = %+v, want tenant 101 (unbound 202 filtered before ambiguity)", discovery)
	}

	if err := db.Exec("DELETE FROM system_user_role").Error; err != nil {
		t.Fatalf("clear bindings: %v", err)
	}
	discovery, err = DiscoverDefaultTenant(db, 42)
	if err != nil {
		t.Fatalf("discovery with no bindings: %v", err)
	}
	if discovery.Resolved {
		t.Fatalf("discovery = %+v, want unresolved without bindings", discovery)
	}

	gateAuthzSeedBinding(t, db, 101, 42)
	gateAuthzSeedBinding(t, db, 202, 42)
	if _, err := DiscoverDefaultTenant(db, 42); !errors.Is(err, ErrTenantForbidden) {
		t.Fatalf("two bound memberships err = %v, want ErrTenantForbidden", err)
	}
}

// The issuance gate applies the same predicate to an explicit tenant choice:
// active membership ∧ active tenant is not sufficient without a binding.
func TestGateSessionIssuanceRequiresRoleBinding(t *testing.T) {
	db := gateAuthzFixture(t)
	gateAuthzSeedTenant(t, db, 101, TenantStatusActive)
	gateAuthzSeedMembership(t, db, 101, 42, MembershipActive)

	if _, err := GateSessionIssuance(db, IssuanceCheckInput{Mode: ModeMulti, UserID: 42, TenantID: 101}); !errors.Is(err, ErrTenantForbidden) {
		t.Fatalf("issuance without binding err = %v, want ErrTenantForbidden", err)
	}

	gateAuthzSeedBinding(t, db, 101, 42)
	claim, err := GateSessionIssuance(db, IssuanceCheckInput{Mode: ModeMulti, UserID: 42, TenantID: 101})
	if err != nil || claim != 101 {
		t.Fatalf("issuance with binding = %d, %v; want 101, nil", claim, err)
	}
}

// The candidate list applies the same predicate: a membership in an active
// tenant without a role binding is withheld from the picker.
func TestListActiveMembershipsForUserExcludesUnboundMembership(t *testing.T) {
	db := gateAuthzFixture(t)
	gateAuthzSeedTenant(t, db, 101, TenantStatusActive)
	gateAuthzSeedTenant(t, db, 202, TenantStatusActive)
	gateAuthzSeedMembership(t, db, 101, 42, MembershipActive)
	gateAuthzSeedMembership(t, db, 202, 42, MembershipActive)
	gateAuthzSeedBinding(t, db, 101, 42)

	candidates, err := ListActiveMembershipsForUser(db, 42)
	if err != nil {
		t.Fatalf("candidates: %v", err)
	}
	if len(candidates) != 1 || candidates[0].TenantID != 101 {
		t.Fatalf("candidates = %+v, want only 101 (202 has no binding)", candidates)
	}
}
