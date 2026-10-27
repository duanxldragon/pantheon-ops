package user

import (
	"errors"
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/pkg/tenant"
	"gorm.io/gorm"
)

const (
	membershipTestTenantID = uint64(282)
	membershipTestPassword = "password123"
)

// newTenantUserService binds a user service to a tenant context, exactly the
// way the HTTP handler does (WithTenantContext(tenant.FromGin(c))). Tests that
// do not call this keep the historical platform/global behaviour.
func newTenantUserService(db *gorm.DB, tenantID uint64) *UserService {
	return NewUserService(db).WithTenantContext(&tenant.Context{
		TenantID:   tenantID,
		Mode:       tenant.ModeMulti,
		ResolvedBy: tenant.ResolvedBySubject,
	})
}

func seedActiveTenant(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := db.Exec(
		"INSERT INTO tenants (id, code, name, status) VALUES (?, 'acme', 'Acme', 'active')",
		membershipTestTenantID,
	).Error; err != nil {
		t.Fatalf("seed tenant: %v", err)
	}
}

type membershipRow struct {
	Role   string
	Status string
}

func loadMembership(t *testing.T, db *gorm.DB, tenantID, userID uint64) (membershipRow, bool) {
	t.Helper()
	var row membershipRow
	err := db.Table(tenantMembershipTable).
		Select("role, status").
		Where("tenant_id = ? AND user_id = ?", tenantID, userID).
		Scan(&row).Error
	if err != nil {
		t.Fatalf("load membership: %v", err)
	}
	return row, row.Role != ""
}

func countMembershipsForUser(t *testing.T, db *gorm.DB, userID uint64) int64 {
	t.Helper()
	var count int64
	if err := db.Table(tenantMembershipTable).Where("user_id = ?", userID).Count(&count).Error; err != nil {
		t.Fatalf("count memberships: %v", err)
	}
	return count
}

// F1: CreateUser used to write system_user only, so the row appeared in the
// list but GateSessionIssuance rejected the first login with 403
// tenant.forbidden. The membership must be written in the same transaction.
func TestCreateUserWritesTenantMembershipForFirstLogin(t *testing.T) {
	db := setupUserTestDB(t)
	seedActiveTenant(t, db)
	s := newTenantUserService(db, membershipTestTenantID)

	created, err := s.CreateUser(&UserCreateReq{
		Username: "acme_member",
		Password: membershipTestPassword,
		Nickname: "Acme Member",
		Status:   1,
	})
	if err != nil {
		t.Fatalf("create user in tenant: %v", err)
	}

	var ownerTenantID uint64
	if err := db.Table("system_user").Select("tenant_id").Where("id = ?", created.ID).Scan(&ownerTenantID).Error; err != nil {
		t.Fatalf("read user tenant: %v", err)
	}
	if ownerTenantID != membershipTestTenantID {
		t.Fatalf("user tenant = %d, want %d", ownerTenantID, membershipTestTenantID)
	}

	row, ok := loadMembership(t, db, membershipTestTenantID, created.ID)
	if !ok {
		t.Fatal("expected a tenant_memberships row written by CreateUser")
	}
	if row.Status != tenant.MembershipActive {
		t.Fatalf("membership status = %q, want %q", row.Status, tenant.MembershipActive)
	}
	if row.Role != tenant.MembershipRoleMember {
		t.Fatalf("membership role = %q, want %q", row.Role, tenant.MembershipRoleMember)
	}

	// This is the exact check login performs at issuance time.
	claim, err := tenant.GateSessionIssuance(db, tenant.IssuanceCheckInput{
		Mode:     tenant.ModeMulti,
		UserID:   created.ID,
		TenantID: membershipTestTenantID,
	})
	if err != nil {
		t.Fatalf("first login is gated: %v", err)
	}
	if claim != membershipTestTenantID {
		t.Fatalf("issued tenant claim = %d, want %d", claim, membershipTestTenantID)
	}
}

// The old half-state (user row committed, membership missing) is invisible to
// the UI but fatal at login. A membership write failure must roll the user
// back instead of leaving a user that can never sign in.
func TestCreateUserRollsBackWhenMembershipWriteFails(t *testing.T) {
	db := setupUserTestDB(t)
	seedActiveTenant(t, db)
	s := newTenantUserService(db, membershipTestTenantID)

	const callbackName = "test:fail_tenant_membership_insert"
	forced := errors.New("forced membership insert failure")
	if err := db.Callback().Create().Before("gorm:create").Register(callbackName, func(tx *gorm.DB) {
		if tx.Statement != nil && tx.Statement.Table == tenantMembershipTable {
			_ = tx.AddError(forced)
		}
	}); err != nil {
		t.Fatalf("register create callback: %v", err)
	}

	if _, err := s.CreateUser(&UserCreateReq{
		Username: "acme_orphan",
		Password: membershipTestPassword,
		Status:   1,
	}); err == nil {
		t.Fatal("expected CreateUser to fail when the membership cannot be written")
	}

	var users int64
	if err := db.Table("system_user").Where("username = ?", "acme_orphan").Count(&users).Error; err != nil {
		t.Fatalf("count users: %v", err)
	}
	if users != 0 {
		t.Fatalf("user row survived the rolled-back transaction: %d", users)
	}
}

// F4: deleting a user must deactivate the membership (audit record kept) so the
// member list stops showing __deleted_user_* ghosts and the login gate denies.
func TestDeleteUserDeactivatesTenantMembership(t *testing.T) {
	db := setupUserTestDB(t)
	seedActiveTenant(t, db)
	// The builtin admin (id 1) is delete-protected; occupy that id first so the
	// user under test is an ordinary row, like it is in every real install.
	if err := db.Exec("INSERT INTO system_user (username, password, status) VALUES ('__builtin_seed', 'x', 1)").Error; err != nil {
		t.Fatalf("seed builtin admin row: %v", err)
	}
	s := newTenantUserService(db, membershipTestTenantID)

	created, err := s.CreateUser(&UserCreateReq{
		Username: "acme_leaver",
		Password: membershipTestPassword,
		Status:   1,
	})
	if err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := s.DeleteUser(created.ID); err != nil {
		t.Fatalf("delete user: %v", err)
	}

	row, ok := loadMembership(t, db, membershipTestTenantID, created.ID)
	if !ok {
		t.Fatal("membership row was deleted; it is an audit record and must only be deactivated")
	}
	if row.Status != tenant.MembershipDisabled {
		t.Fatalf("membership status = %q, want %q", row.Status, tenant.MembershipDisabled)
	}

	if _, err := tenant.GateSessionIssuance(db, tenant.IssuanceCheckInput{
		Mode:     tenant.ModeMulti,
		UserID:   created.ID,
		TenantID: membershipTestTenantID,
	}); !errors.Is(err, tenant.ErrTenantForbidden) {
		t.Fatalf("login after delete = %v, want %v", err, tenant.ErrTenantForbidden)
	}

	// The user row itself is soft-deleted and renamed; only the membership row
	// remains visible to the member list, and it is no longer active.
	var surviving int64
	if err := db.Table("system_user").Where("id = ? AND deleted_at IS NULL", created.ID).Count(&surviving).Error; err != nil {
		t.Fatalf("count live user: %v", err)
	}
	if surviving != 0 {
		t.Fatalf("soft-deleted user still live: %d", surviving)
	}
}

// Platform-scope users have no tenant and never get a membership; the login
// gate lets tenant_id = 0 through. This is the compat path that must not
// regress.
func TestCreatePlatformUserSkipsTenantMembership(t *testing.T) {
	db := setupUserTestDB(t)
	s := NewUserService(db)

	created, err := s.CreateUser(&UserCreateReq{
		Username: "platform_user",
		Password: membershipTestPassword,
		Status:   1,
	})
	if err != nil {
		t.Fatalf("create platform user: %v", err)
	}

	var ownerTenantID uint64
	if err := db.Table("system_user").Select("tenant_id").Where("id = ?", created.ID).Scan(&ownerTenantID).Error; err != nil {
		t.Fatalf("read user tenant: %v", err)
	}
	if ownerTenantID != tenant.PlatformGlobalTenantID {
		t.Fatalf("platform user tenant = %d, want %d", ownerTenantID, tenant.PlatformGlobalTenantID)
	}
	if got := countMembershipsForUser(t, db, created.ID); got != 0 {
		t.Fatalf("platform user has %d membership rows, want 0", got)
	}

	claim, err := tenant.GateSessionIssuance(db, tenant.IssuanceCheckInput{
		Mode:     tenant.ModeMulti,
		UserID:   created.ID,
		TenantID: tenant.PlatformGlobalTenantID,
	})
	if err != nil || claim != 0 {
		t.Fatalf("platform issuance = (%d, %v), want (0, nil)", claim, err)
	}
}
