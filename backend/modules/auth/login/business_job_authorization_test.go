package login

import (
	"context"
	"errors"
	"strconv"
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/internal/middleware"
	"github.com/duanxldragon/pantheon-base/backend/modules/system/config/setting"
	"github.com/duanxldragon/pantheon-base/backend/modules/system/iam/role"
	"github.com/duanxldragon/pantheon-base/backend/modules/system/iam/user"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/contracts/businessauth"
	"github.com/duanxldragon/pantheon-base/backend/pkg/database"
	"github.com/duanxldragon/pantheon-base/backend/pkg/permissionroutes"
	"github.com/duanxldragon/pantheon-base/backend/pkg/tenant"
	"github.com/duanxldragon/pantheon-base/backend/pkg/testmysql"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestBusinessJobAuthorizationCapabilityAvailable(t *testing.T) {
	if _, ok := NewBusinessAuthorizationReader(nil, nil, nil).(businessauth.JobAuthorizationReader); !ok {
		t.Fatal("auth-owned reader has no durable job authorization capability")
	}
}

type businessJobFixture struct {
	db     *gorm.DB
	reader businessauth.JobAuthorizationReader
	input  businessauth.JobInput
	owner  user.SystemUser
	role   role.SystemRole
	mode   tenant.ModeReader
}

func newBusinessJobFixture(t *testing.T) businessJobFixture {
	t.Helper()
	return newBusinessJobFixtureInTenant(t, tenant.PlatformGlobalTenantID)
}

func newBusinessJobFixtureInTenant(t *testing.T, tenantID uint64) businessJobFixture {
	t.Helper()
	db := testmysql.Open(t)
	// No session schema or Redis exists. A grant cannot depend on a saved token
	// or session, including after recreating the adapter on a cold restart.
	require.NoError(t, db.AutoMigrate(&user.SystemUser{}, &role.SystemRole{}, &user.SystemUserRole{},
		&role.SystemRolePermission{}, &setting.SystemSetting{}, &database.CasbinRule{}, &middleware.SystemRoleDataScope{}))
	modeValue := tenant.ModeCompat
	if tenantID != tenant.PlatformGlobalTenantID {
		require.NoError(t, db.AutoMigrate(&tenant.Tenant{}, &tenant.Membership{}))
		modeValue = tenant.ModeMulti
	}
	seedSettings(t, db, map[string]string{tenant.FeatureFlagSettingKey: modeValue})
	action := businessauth.Action{Permission: "business:jobfixture:dispatch", Method: "POST",
		Path: "/api/v1/business/jobfixture/jobs/7/start", RouteTemplate: "/api/v1/business/jobfixture/jobs/:id/start"}
	require.NoError(t, permissionroutes.Register("business:jobfixture", []permissionroutes.Binding{{
		Permission: action.Permission, Routes: []permissionroutes.Route{{Method: action.Method, Path: action.RouteTemplate}},
	}}))
	owner, grantedRole := seedBusinessJobGrant(t, db, tenantID, action)
	mode := tenant.NewStrictSettingModeReader(setting.NewSettingService(db))
	reader, ok := NewBusinessAuthorizationReader(&Runtime{db: db, userRepo: testCredentialRepo(db)}, nil, mode).(businessauth.JobAuthorizationReader)
	require.True(t, ok)
	return businessJobFixture{db: db, reader: reader,
		input: businessauth.JobInput{ActorID: owner.ID, ResourceTenantID: tenantID, Action: action},
		owner: owner, role: grantedRole, mode: mode}
}

func seedBusinessJobGrant(t *testing.T, db *gorm.DB, tenantID uint64, action businessauth.Action) (user.SystemUser, role.SystemRole) {
	t.Helper()
	code := strconv.FormatUint(tenantID, 10)
	owner := user.SystemUser{Username: "job-owner-" + code, Password: "unused", Status: common.StatusEnabled, DeptID: 8, TenantID: tenantID}
	grantedRole := role.SystemRole{RoleKey: "job_dispatch", RoleName: "job dispatch", Status: common.StatusEnabled, TenantID: tenantID}
	require.NoError(t, db.Create(&owner).Error)
	require.NoError(t, db.Create(&grantedRole).Error)
	subject := tenant.GlobalRoleSubject(grantedRole.RoleKey)
	if tenantID != tenant.PlatformGlobalTenantID {
		require.NoError(t, db.Create(&tenant.Tenant{ID: tenantID, Code: code, Name: "Job tenant", Status: tenant.TenantStatusActive}).Error)
		require.NoError(t, db.Create(&tenant.Membership{TenantID: tenantID, UserID: owner.ID, Status: tenant.MembershipActive}).Error)
		subject = tenant.RoleSubject(grantedRole.RoleKey, tenantID)
	}
	require.NoError(t, db.Create(&user.SystemUserRole{UserID: owner.ID, RoleID: grantedRole.ID, TenantID: tenantID}).Error)
	require.NoError(t, db.Create(&role.SystemRolePermission{RoleID: grantedRole.ID, PermissionKey: action.Permission, TenantID: tenantID}).Error)
	// A broad API policy exercises the stricter registered job action check,
	// including literal siblings of the numeric resource route.
	require.NoError(t, db.Create(&database.CasbinRule{PType: "p", V0: subject, V1: "/api/v1/business/jobfixture/*", V2: "POST"}).Error)
	require.NoError(t, db.Create(&middleware.SystemRoleDataScope{TenantID: tenantID, RoleKey: grantedRole.RoleKey, Mode: common.DataScopeModeDept}).Error)
	return owner, grantedRole
}

func TestBusinessJobAuthorizationFreshScopeWithoutSession(t *testing.T) {
	f := newBusinessJobFixture(t)
	ctx := context.Background()
	auth, err := f.reader.AuthorizeJob(ctx, f.input)
	require.NoError(t, err)
	require.NotNil(t, auth.DataScope)
	require.Equal(t, common.DataScopeModeDept, auth.DataScope.Mode)
	require.Equal(t, uint64(8), auth.DataScope.DeptID)
	require.Equal(t, f.owner.ID, auth.DataScope.UserID)
	require.False(t, auth.DataScope.IsAdmin)
	require.Equal(t, []businessauth.RoleBinding{{RoleKey: f.role.RoleKey}}, auth.Bindings)
	require.False(t, f.db.Migrator().HasTable("system_user_session"))
	require.NoError(t, f.db.Model(&f.owner).Update("dept_id", 99).Error)
	auth, err = f.reader.AuthorizeJob(ctx, f.input)
	require.NoError(t, err)
	require.Equal(t, uint64(99), auth.DataScope.DeptID, "creation-time scope must not survive a department change")
	restarted := NewBusinessAuthorizationReader(&Runtime{db: f.db, userRepo: testCredentialRepo(f.db)}, nil, f.mode).(businessauth.JobAuthorizationReader)
	auth, err = restarted.AuthorizeJob(ctx, f.input)
	require.NoError(t, err)
	require.Equal(t, uint64(99), auth.DataScope.DeptID)
}

func TestBusinessJobAuthorizationSupportsTenantJobs(t *testing.T) {
	const tenantA, tenantB = 101, 202
	f := newBusinessJobFixtureInTenant(t, tenantA)
	otherOwner, otherRole := seedBusinessJobGrant(t, f.db, tenantB, f.input.Action)
	otherJob := businessauth.JobInput{ActorID: otherOwner.ID, ResourceTenantID: tenantB, Action: f.input.Action}
	ctx := context.Background()
	for _, input := range []businessauth.JobInput{f.input, otherJob} {
		auth, err := f.reader.AuthorizeJob(ctx, input)
		require.NoError(t, err)
		require.Equal(t, []businessauth.RoleBinding{{TenantID: input.ResourceTenantID, RoleKey: f.role.RoleKey}}, auth.Bindings)
		require.NotNil(t, auth.DataScope)
		require.Equal(t, input.ActorID, auth.DataScope.UserID)
		require.Equal(t, common.DataScopeModeDept, auth.DataScope.Mode)
		require.False(t, auth.DataScope.IsAdmin)
	}
	require.False(t, f.db.Migrator().HasTable("system_user_session"))

	// Historical cross-tenant membership and inconsistent binding rows must
	// not authorize a job whose resource tenant differs from the owner.
	require.NoError(t, f.db.Create(&tenant.Membership{TenantID: tenantB, UserID: f.owner.ID}).Error)
	require.NoError(t, f.db.Create(&user.SystemUserRole{TenantID: tenantB, UserID: f.owner.ID, RoleID: otherRole.ID}).Error)
	otherJob.ActorID = f.owner.ID
	auth, err := f.reader.AuthorizeJob(ctx, otherJob)
	require.ErrorIs(t, err, businessauth.ErrDenied)
	require.Empty(t, auth)

	require.NoError(t, f.db.Model(&f.owner).Update("dept_id", 99).Error)
	restarted := NewBusinessAuthorizationReader(&Runtime{db: f.db, userRepo: testCredentialRepo(f.db)}, nil, f.mode).(businessauth.JobAuthorizationReader)
	auth, err = restarted.AuthorizeJob(ctx, f.input)
	require.NoError(t, err)
	require.Equal(t, uint64(99), auth.DataScope.DeptID)

	_, globalRole := seedBusinessJobGrant(t, f.db, tenant.PlatformGlobalTenantID, f.input.Action)
	require.NoError(t, f.db.Create(&user.SystemUserRole{UserID: f.owner.ID, RoleID: globalRole.ID}).Error)
	require.NoError(t, f.db.Delete(&user.SystemUserRole{TenantID: tenantA, UserID: f.owner.ID, RoleID: f.role.ID}).Error)
	auth, err = restarted.AuthorizeJob(ctx, f.input)
	require.ErrorIs(t, err, businessauth.ErrDenied)
	require.Empty(t, auth)
}

func TestBusinessJobAuthorizationRechecksTenantState(t *testing.T) {
	for _, tc := range []struct {
		name   string
		revoke func(f businessJobFixture) error
	}{
		{"disabled-membership", func(f businessJobFixture) error {
			return f.db.Model(&tenant.Membership{}).Where("tenant_id = ? AND user_id = ?", f.input.ResourceTenantID, f.owner.ID).
				Update("status", tenant.MembershipDisabled).Error
		}},
		{"suspended-tenant", func(f businessJobFixture) error {
			return f.db.Model(&tenant.Tenant{ID: f.input.ResourceTenantID}).Update("status", tenant.TenantStatusSuspended).Error
		}},
		{"compat-mode", func(f businessJobFixture) error {
			return f.db.Model(&setting.SystemSetting{}).Where("setting_key = ?", tenant.FeatureFlagSettingKey).
				Update("setting_value", tenant.ModeCompat).Error
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newBusinessJobFixtureInTenant(t, 101)
			_, err := f.reader.AuthorizeJob(context.Background(), f.input)
			require.NoError(t, err)
			require.NoError(t, tc.revoke(f))
			auth, err := f.reader.AuthorizeJob(context.Background(), f.input)
			require.ErrorIs(t, err, businessauth.ErrDenied)
			require.Empty(t, auth)
		})
	}
}

func TestBusinessJobAuthorizationCurrentRevocationMatrix(t *testing.T) {
	for _, tc := range []struct {
		name   string
		revoke func(*testing.T, businessJobFixture)
	}{
		{"disabled-owner", func(t *testing.T, f businessJobFixture) {
			require.NoError(t, f.db.Model(&f.owner).Update("status", 2).Error)
		}},
		{"deleted-owner", func(t *testing.T, f businessJobFixture) { require.NoError(t, f.db.Delete(&f.owner).Error) }},
		{"tenant-local-owner", func(t *testing.T, f businessJobFixture) {
			require.NoError(t, f.db.Model(&f.owner).Update("tenant_id", 101).Error)
		}},
		{"disabled-role", func(t *testing.T, f businessJobFixture) {
			require.NoError(t, f.db.Model(&f.role).Update("status", 2).Error)
		}},
		{"deleted-role", func(t *testing.T, f businessJobFixture) { require.NoError(t, f.db.Delete(&f.role).Error) }},
		{"removed-binding", func(t *testing.T, f businessJobFixture) {
			require.NoError(t, f.db.Where("user_id = ?", f.owner.ID).Delete(&user.SystemUserRole{}).Error)
		}},
		{"removed-key", func(t *testing.T, f businessJobFixture) {
			require.NoError(t, f.db.Where("role_id = ?", f.role.ID).Delete(&role.SystemRolePermission{}).Error)
		}},
		{"removed-policy", func(t *testing.T, f businessJobFixture) {
			require.NoError(t, f.db.Where("v0 = ?", f.role.RoleKey).Delete(&database.CasbinRule{}).Error)
		}},
		{"missing-scope", func(t *testing.T, f businessJobFixture) {
			require.NoError(t, f.db.Where("role_key = ?", f.role.RoleKey).Delete(&middleware.SystemRoleDataScope{}).Error)
		}},
		{"invalid-scope", func(t *testing.T, f businessJobFixture) {
			require.NoError(t, f.db.Model(&middleware.SystemRoleDataScope{}).Where("role_key = ?", f.role.RoleKey).Update("mode", "unknown").Error)
		}},
		{"multi-mode", func(t *testing.T, f businessJobFixture) {
			require.NoError(t, f.db.Model(&setting.SystemSetting{}).Where("setting_key = ?", tenant.FeatureFlagSettingKey).Update("setting_value", tenant.ModeMulti).Error)
		}},
		{"unknown-mode", func(t *testing.T, f businessJobFixture) {
			require.NoError(t, f.db.Model(&setting.SystemSetting{}).Where("setting_key = ?", tenant.FeatureFlagSettingKey).Update("setting_value", "unknown").Error)
		}},
		{"missing-mode", func(t *testing.T, f businessJobFixture) {
			require.NoError(t, f.db.Where("setting_key = ?", tenant.FeatureFlagSettingKey).Delete(&setting.SystemSetting{}).Error)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newBusinessJobFixture(t)
			_, err := f.reader.AuthorizeJob(context.Background(), f.input)
			require.NoError(t, err, "precondition must be an actual granted job")
			tc.revoke(t, f)
			auth, err := f.reader.AuthorizeJob(context.Background(), f.input)
			require.ErrorIs(t, err, businessauth.ErrDenied)
			require.Equal(t, businessauth.Authorization{}, auth)
		})
	}
}

func TestBusinessJobAuthorizationRejectsUnregisteredOrAmbiguousActions(t *testing.T) {
	f := newBusinessJobFixture(t)
	for _, tc := range []struct {
		name   string
		change func(*businessauth.JobInput)
	}{
		{"zero-owner", func(in *businessauth.JobInput) { in.ActorID = 0 }},
		{"nonexistent-owner", func(in *businessauth.JobInput) { in.ActorID = 99999 }},
		{"foreign-resource", func(in *businessauth.JobInput) { in.ResourceTenantID = 101 }},
		{"missing-key", func(in *businessauth.JobInput) { in.Action.Permission = "" }},
		{"unknown-key", func(in *businessauth.JobInput) { in.Action.Permission = "business:jobfixture:unknown" }},
		{"wrong-method", func(in *businessauth.JobInput) { in.Action.Method = "GET" }},
		{"wrong-template", func(in *businessauth.JobInput) {
			in.Action.RouteTemplate = "/api/v1/business/jobfixture/jobs/:id/delete"
		}},
		{"literal-sibling", func(in *businessauth.JobInput) { in.Action.Path = "/api/v1/business/jobfixture/jobs/export/start" }},
		{"other-static-action", func(in *businessauth.JobInput) { in.Action.Path = "/api/v1/business/jobfixture/jobs/7/delete" }},
		{"zero-id", func(in *businessauth.JobInput) { in.Action.Path = "/api/v1/business/jobfixture/jobs/0/start" }},
		{"overflow-id", func(in *businessauth.JobInput) {
			in.Action.Path = "/api/v1/business/jobfixture/jobs/18446744073709551616/start"
		}},
		{"encoded-id", func(in *businessauth.JobInput) { in.Action.Path = "/api/v1/business/jobfixture/jobs/%37/start" }},
		{"query", func(in *businessauth.JobInput) { in.Action.Path += "?id=7" }},
		{"extra-segment", func(in *businessauth.JobInput) { in.Action.Path += "/later" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input := f.input
			tc.change(&input)
			auth, err := f.reader.AuthorizeJob(context.Background(), input)
			require.ErrorIs(t, err, businessauth.ErrDenied)
			require.Equal(t, businessauth.Authorization{}, auth)
		})
	}
}

type jobModeReaderFunc func(context.Context) (string, error)

func (f jobModeReaderFunc) ReadMode(ctx context.Context) (string, error) { return f(ctx) }

func TestBusinessJobAuthorizationUnavailableStateAndCancellation(t *testing.T) {
	f := newBusinessJobFixture(t)
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	for _, ctx := range []context.Context{nil, cancelled} {
		auth, err := f.reader.AuthorizeJob(ctx, f.input)
		require.ErrorIs(t, err, businessauth.ErrDenied)
		require.Equal(t, businessauth.Authorization{}, auth)
	}
	for _, reader := range []*businessAuthorizationReader{nil, {}, {runtime: &Runtime{}}, {runtime: &Runtime{db: f.db}}} {
		auth, err := reader.AuthorizeJob(context.Background(), f.input)
		require.ErrorIs(t, err, businessauth.ErrDenied)
		require.Equal(t, businessauth.Authorization{}, auth)
	}
	private := errors.New("synthetic private storage diagnostic")
	failingMode := NewBusinessAuthorizationReader(&Runtime{db: f.db, userRepo: testCredentialRepo(f.db)}, nil,
		jobModeReaderFunc(func(context.Context) (string, error) { return "", private })).(businessauth.JobAuthorizationReader)
	_, err := failingMode.AuthorizeJob(context.Background(), f.input)
	require.ErrorIs(t, err, businessauth.ErrDenied)
	require.NotContains(t, err.Error(), private.Error())
	reads := 0
	changingMode := NewBusinessAuthorizationReader(&Runtime{db: f.db, userRepo: testCredentialRepo(f.db)}, nil,
		jobModeReaderFunc(func(context.Context) (string, error) {
			reads++
			if reads == 1 {
				return tenant.ModeCompat, nil
			}
			return tenant.ModeMulti, nil
		})).(businessauth.JobAuthorizationReader)
	auth, err := changingMode.AuthorizeJob(context.Background(), f.input)
	require.ErrorIs(t, err, businessauth.ErrDenied)
	require.Equal(t, businessauth.Authorization{}, auth)
	require.Equal(t, 2, reads)
	require.NoError(t, f.db.Callback().Query().Before("gorm:query").Register("test:job-storage-failure", func(tx *gorm.DB) { tx.AddError(private) }))
	t.Cleanup(func() { _ = f.db.Callback().Query().Remove("test:job-storage-failure") })
	auth, err = f.reader.AuthorizeJob(context.Background(), f.input)
	require.ErrorIs(t, err, businessauth.ErrDenied)
	require.Equal(t, businessauth.Authorization{}, auth)
	require.NotContains(t, err.Error(), private.Error())
}
