package tenant

import (
	"errors"
	"testing"
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
