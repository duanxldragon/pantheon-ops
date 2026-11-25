package login

import (
	"context"
	"errors"
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
	db := testmysql.Open(t)
	// No session schema or Redis exists. A grant cannot depend on a saved token
	// or session, including after recreating the adapter on a cold restart.
	require.NoError(t, db.AutoMigrate(&user.SystemUser{}, &role.SystemRole{}, &user.SystemUserRole{},
		&role.SystemRolePermission{}, &setting.SystemSetting{}, &database.CasbinRule{}, &middleware.SystemRoleDataScope{}))
	seedSettings(t, db, map[string]string{tenant.FeatureFlagSettingKey: tenant.ModeCompat})
	owner := user.SystemUser{Username: "job-owner", Password: "unused", Status: common.StatusEnabled, DeptID: 8}
	grantedRole := role.SystemRole{RoleKey: "job_dispatch", RoleName: "job dispatch", Status: common.StatusEnabled}
	require.NoError(t, db.Create(&owner).Error)
	require.NoError(t, db.Create(&grantedRole).Error)
	require.NoError(t, db.Create(&user.SystemUserRole{UserID: owner.ID, RoleID: grantedRole.ID}).Error)
	action := businessauth.Action{Permission: "business:jobfixture:dispatch", Method: "POST",
		Path: "/api/v1/business/jobfixture/jobs/7/start", RouteTemplate: "/api/v1/business/jobfixture/jobs/:id/start"}
	require.NoError(t, permissionroutes.Register("business:jobfixture", []permissionroutes.Binding{{
		Permission: action.Permission, Routes: []permissionroutes.Route{{Method: action.Method, Path: action.RouteTemplate}},
	}}))
	require.NoError(t, db.Create(&role.SystemRolePermission{RoleID: grantedRole.ID, PermissionKey: action.Permission}).Error)
	// A broad API policy exercises the stricter registered job action check,
	// including literal siblings of the numeric resource route.
	require.NoError(t, db.Create(&database.CasbinRule{PType: "p", V0: grantedRole.RoleKey, V1: "/api/v1/business/jobfixture/*", V2: "POST"}).Error)
	require.NoError(t, db.Create(&middleware.SystemRoleDataScope{RoleKey: grantedRole.RoleKey, Mode: common.DataScopeModeDept}).Error)
	mode := tenant.NewStrictSettingModeReader(setting.NewSettingService(db))
	reader, ok := NewBusinessAuthorizationReader(&Runtime{db: db, userRepo: testCredentialRepo(db)}, nil, mode).(businessauth.JobAuthorizationReader)
	require.True(t, ok)
	return businessJobFixture{db: db, reader: reader, input: businessauth.JobInput{ActorID: owner.ID, Action: action}, owner: owner, role: grantedRole, mode: mode}
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
