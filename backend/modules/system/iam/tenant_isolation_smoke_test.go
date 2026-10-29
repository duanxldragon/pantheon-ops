// Package iam_test hosts the cross-module two-tenant isolation smoke that
// guards the shared tenant-scope implementation (pkg/tenant Scoper). Every
// service converted onto the shared helper gets one subtest here: a subject
// bound to tenant A must see and mutate only tenant A's rows, while the same
// operations against tenant B's ids return not-found with no existence leak.
// The compat context keeps reading exactly the tenant_id=0 population.
package iam_test

import (
	"testing"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/modules/auth/login"
	authsecurity "github.com/duanxldragon/pantheon-base/backend/modules/auth/security"
	authsession "github.com/duanxldragon/pantheon-base/backend/modules/auth/session"
	"github.com/duanxldragon/pantheon-base/backend/modules/platform"
	"github.com/duanxldragon/pantheon-base/backend/modules/system/config/dict"
	"github.com/duanxldragon/pantheon-base/backend/modules/system/config/setting"
	"github.com/duanxldragon/pantheon-base/backend/modules/system/iam/permission"
	"github.com/duanxldragon/pantheon-base/backend/modules/system/iam/role"
	"github.com/duanxldragon/pantheon-base/backend/modules/system/iam/user"
	"github.com/duanxldragon/pantheon-base/backend/modules/system/org/dept"
	"github.com/duanxldragon/pantheon-base/backend/modules/system/org/post"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/database"
	"github.com/duanxldragon/pantheon-base/backend/pkg/tenant"
	"github.com/duanxldragon/pantheon-base/backend/pkg/testmysql"
	"github.com/duanxldragon/pantheon-base/backend/pkg/testredis"

	"gorm.io/gorm"
)

const (
	smokeTenantA = uint64(101)
	smokeTenantB = uint64(202)
)

// smokeMulti returns a multi-mode request context for the given tenant.
func smokeMulti(id uint64) *tenant.Context {
	return &tenant.Context{TenantID: id, Mode: tenant.ModeMulti}
}

// smokeCompat returns a compat-mode request context (always tenant 0).
func smokeCompat() *tenant.Context {
	return &tenant.Context{TenantID: tenant.PlatformGlobalTenantID, Mode: tenant.ModeCompat}
}

// smokeDB builds the isolated MySQL database with every table the smoke
// touches. Optional collaborators (menus, casbin) are seeded minimally so
// validation paths behave like production.
func smokeDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := testmysql.Open(t)
	if err := db.AutoMigrate(
		&role.SystemRole{}, &role.SystemRoleMenu{}, &role.SystemRolePermission{},
		&database.CasbinRule{},
		&post.SystemPost{},
		&dept.SystemDept{},
		&user.SystemUser{}, &user.SystemUserRole{},
		&dict.SystemDictType{}, &dict.SystemDictItem{},
		&setting.SystemSetting{},
		&permission.PermissionRoleDataScopePolicy{},
		&login.SystemLogLogin{},
		&authsecurity.SystemAuthSecurityEvent{},
		&authsession.SystemUserSession{},
	); err != nil {
		t.Fatalf("migrate smoke tables: %v", err)
	}
	// role update validates menu ids against system_menu (id 1 seeded), and
	// session list joins system_user for the username column. The dashboard
	// counts visible menus and today's operation logs, so is_visible and
	// system_log_oper must exist even though no subtest writes them.
	if err := db.Exec("CREATE TABLE IF NOT EXISTS system_menu (id BIGINT UNSIGNED PRIMARY KEY, page_perm TEXT, perms TEXT, type TEXT, module TEXT, path TEXT, is_visible INT NOT NULL DEFAULT 1)").Error; err != nil {
		t.Fatalf("create system_menu: %v", err)
	}
	if err := db.Exec("INSERT INTO system_menu (id, page_perm, perms, type) VALUES (1, 'sys:user:list', '', 'C')").Error; err != nil {
		t.Fatalf("seed system_menu: %v", err)
	}
	if err := db.Exec("CREATE TABLE IF NOT EXISTS system_log_oper (id BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT, tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0, oper_time DATETIME)").Error; err != nil {
		t.Fatalf("create system_log_oper: %v", err)
	}
	return db
}

func TestTwoTenantIsolationSmoke(t *testing.T) {
	db := smokeDB(t)

	seedSmokeData(t, db)

	subtests := []struct {
		name string
		run  func(t *testing.T, db *gorm.DB)
	}{
		{"role list and mutate stay inside the caller's tenant", smokeRoleIsolation},
		{"post list and mutate stay inside the caller's tenant", smokePostIsolation},
		{"dept tree and mutate stay inside the caller's tenant", smokeDeptIsolation},
		{"user list and mutate stay inside the caller's tenant", smokeUserIsolation},
		{"dict list, mutate and write-owner stay inside the caller's tenant", smokeDictIsolation},
		{"setting reads resolve the caller's override, never another tenant's", smokeSettingIsolation},
		{"permission data-scope console lists only the caller's tenant roles", smokeDataScopeIsolation},
		{"login logs list and delete stay inside the caller's tenant", smokeLoginLogIsolation},
		{"security events list stays inside the caller's tenant", smokeSecurityEventIsolation},
		{"admin session list stays inside the caller's tenant", smokeAdminSessionIsolation},
		{"platform summary counts only the caller's tenant rows", smokeDashboardIsolation},
	}
	for _, st := range subtests {
		t.Run(st.name, func(t *testing.T) {
			st.run(t, db)
		})
	}
}

func smokeRoleIsolation(t *testing.T, db *gorm.DB) {
	t.Helper()
	svc := role.NewRoleService(db).WithTenantContext(smokeMulti(smokeTenantA))

	page, err := svc.ListRoles(nil)
	if err != nil {
		t.Fatalf("list roles: %v", err)
	}
	if page.Total != 1 || len(page.Items) != 1 || page.Items[0].RoleKey != "ops-a" {
		t.Fatalf("tenant A must see only its own role, got total=%d items=%+v", page.Total, page.Items)
	}

	// Same update shape succeeds on A's row and fails on B's row. The request
	// must carry A's own data scope: a role update rewrites the policy row, and
	// an omitted scope falls back to "all".
	aUpdate := &role.RoleUpdateReq{RoleName: "A 运营改", RoleKey: "ops-a", MenuIDs: []uint64{1}, DataScope: common.DataScopeModeSelf}
	if _, err := svc.UpdateRole(page.Items[0].ID, aUpdate); err != nil {
		t.Fatalf("tenant A updates its own role: %v", err)
	}
	if _, err := svc.UpdateRole(smokeRoleBID, aUpdate); err == nil {
		t.Fatalf("tenant A must not update tenant B's role")
	}
	if err := svc.DeleteRole(smokeRoleBID); err == nil {
		t.Fatalf("tenant A must not delete tenant B's role")
	}
	if countSmokeRows(t, db, "system_role", smokeTenantB) != 1 {
		t.Fatalf("tenant B's role must be untouched")
	}
}

func smokePostIsolation(t *testing.T, db *gorm.DB) {
	t.Helper()
	svc := post.NewPostService(db).WithTenantContext(smokeMulti(smokeTenantA))

	page, err := svc.ListPosts(nil)
	if err != nil {
		t.Fatalf("list posts: %v", err)
	}
	if page.Total != 1 || len(page.Items) != 1 {
		t.Fatalf("tenant A must see only its own post, got total=%d items=%+v", page.Total, page.Items)
	}

	aUpdate := &post.PostUpdateReq{DeptID: smokeDeptAChildID, PostCode: "p-a", PostName: "A 岗位", Sort: 1, Status: 1}
	if _, err := svc.UpdatePost(page.Items[0].ID, aUpdate); err != nil {
		t.Fatalf("tenant A updates its own post: %v", err)
	}
	if _, err := svc.UpdatePost(smokePostBID, aUpdate); err == nil {
		t.Fatalf("tenant A must not update tenant B's post")
	}
	if err := svc.DeletePost(smokePostBID); err == nil {
		t.Fatalf("tenant A must not delete tenant B's post")
	}
	if countSmokeRows(t, db, "system_post", smokeTenantB) != 1 {
		t.Fatalf("tenant B's post must be untouched")
	}
}

func smokeDeptIsolation(t *testing.T, db *gorm.DB) {
	t.Helper()
	svc := dept.NewDeptService(db).WithTenantContext(smokeMulti(smokeTenantA))

	tree, err := svc.GetDeptTree(nil)
	if err != nil {
		t.Fatalf("get dept tree: %v", err)
	}
	if len(tree) != 1 || tree[0].DeptName != "A 部门" {
		t.Fatalf("tenant A must see only its own dept, got %+v", tree)
	}

	aUpdate := &dept.DeptUpdateReq{ParentID: 0, DeptName: "A 部门改", Sort: 1, Status: 1}
	if _, err := svc.UpdateDept(tree[0].ID, aUpdate); err != nil {
		t.Fatalf("tenant A updates its own dept: %v", err)
	}
	if _, err := svc.UpdateDept(smokeDeptBID, aUpdate); err == nil {
		t.Fatalf("tenant A must not update tenant B's dept")
	}
	if err := svc.DeleteDept(smokeDeptBChildID); err == nil {
		t.Fatalf("tenant A must not delete tenant B's dept")
	}
	if countSmokeRows(t, db, "system_dept", smokeTenantB) != 2 {
		t.Fatalf("tenant B's dept tree must be untouched")
	}
}

func smokeUserIsolation(t *testing.T, db *gorm.DB) {
	t.Helper()
	svc := user.NewUserService(db).WithTenantContext(smokeMulti(smokeTenantA))

	page, err := svc.ListUsers(nil, nil)
	if err != nil {
		t.Fatalf("list users: %v", err)
	}
	if page.Total != 1 || len(page.Items) != 1 || page.Items[0].Username != "user-a" {
		t.Fatalf("tenant A must see only its own user, got total=%d items=%+v", page.Total, page.Items)
	}

	if _, err := svc.GetUserDetail(smokeUserBID); err == nil {
		t.Fatalf("tenant A must not read tenant B's user detail")
	}
	aUpdate := &user.UserUpdateReq{Nickname: "A 用户改", Status: 1}
	if _, err := svc.UpdateUser(page.Items[0].ID, aUpdate); err != nil {
		t.Fatalf("tenant A updates its own user: %v", err)
	}
	if _, err := svc.UpdateUser(smokeUserBID, aUpdate); err == nil {
		t.Fatalf("tenant A must not update tenant B's user")
	}
	if err := svc.DeleteUser(smokeUserBID); err == nil {
		t.Fatalf("tenant A must not delete tenant B's user")
	}
	if countSmokeRows(t, db, "system_user", smokeTenantB) != 1 {
		t.Fatalf("tenant B's user must be untouched")
	}
}

func smokeDictIsolation(t *testing.T, db *gorm.DB) {
	t.Helper()
	svc := dict.NewDictService(db).WithTenantContext(smokeMulti(smokeTenantA))

	types, err := svc.ListDictTypes(nil)
	if err != nil {
		t.Fatalf("list dict types: %v", err)
	}
	if len(types) != 1 || types[0].DictCode != "smoke-a" {
		t.Fatalf("tenant A must see only its own dict type, got %+v", types)
	}

	created, err := svc.CreateDictType(&dict.DictTypeCreateReq{DictCode: "smoke-created", DictName: "A 新建", Status: 1})
	if err != nil {
		t.Fatalf("tenant A creates a dict type: %v", err)
	}
	var createdRow struct {
		TenantID uint64
	}
	if err := db.Table("system_dict_type").Select("tenant_id").Where("id = ?", created.ID).Scan(&createdRow).Error; err != nil {
		t.Fatalf("load created dict type: %v", err)
	}
	if createdRow.TenantID != smokeTenantA {
		t.Fatalf("write owner must come from the context (tenant %d), got %d", smokeTenantA, createdRow.TenantID)
	}

	aUpdate := &dict.DictTypeUpdateReq{DictCode: "smoke-a", DictName: "A 字典改", Status: 1}
	if _, err := svc.UpdateDictType(types[0].ID, aUpdate); err != nil {
		t.Fatalf("tenant A updates its own dict type: %v", err)
	}
	if _, err := svc.UpdateDictType(smokeDictBID, aUpdate); err == nil {
		t.Fatalf("tenant A must not update tenant B's dict type")
	}
	if err := svc.DeleteDictType(smokeDictBID); err == nil {
		t.Fatalf("tenant A must not delete tenant B's dict type")
	}
	if countSmokeRows(t, db, "system_dict_type", smokeTenantB) != 1 {
		t.Fatalf("tenant B's dict type must be untouched")
	}
}

func smokeSettingIsolation(t *testing.T, db *gorm.DB) {
	t.Helper()
	svc := setting.NewSettingService(db).WithTenantContext(smokeMulti(smokeTenantA))

	value, err := svc.GetByKey("site.name")
	if err != nil {
		t.Fatalf("get by key: %v", err)
	}
	if value != "override-a" {
		t.Fatalf("tenant A must resolve its own override, got %q", value)
	}

	compatSvc := setting.NewSettingService(db).WithTenantContext(smokeCompat())
	compatValue, err := compatSvc.GetByKey("site.name")
	if err != nil {
		t.Fatalf("compat get by key: %v", err)
	}
	if compatValue != "platform-global" {
		t.Fatalf("compat must resolve the platform-global value, got %q", compatValue)
	}
}

func smokeDataScopeIsolation(t *testing.T, db *gorm.DB) {
	t.Helper()
	svc := permission.NewPermissionService(db).WithTenantContext(smokeMulti(smokeTenantA))

	page, err := svc.ListDataScopePolicies(nil)
	if err != nil {
		t.Fatalf("list data scope policies: %v", err)
	}
	if page.Total != 1 || len(page.Items) != 1 || page.Items[0].RoleKey != "ops-a" {
		t.Fatalf("tenant A must list only its own role row, got total=%d items=%+v", page.Total, page.Items)
	}
	if page.Items[0].Mode != common.DataScopeModeSelf {
		t.Fatalf("tenant A must echo its own policy mode, got %q", page.Items[0].Mode)
	}
}

func smokeLoginLogIsolation(t *testing.T, db *gorm.DB) {
	t.Helper()
	svc := login.NewLoginService(db, nil, nil, nil).WithTenantContext(smokeMulti(smokeTenantA))

	page, err := svc.ListLoginLogs(&login.LoginLogQuery{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("list login logs: %v", err)
	}
	if page.Total != 1 || len(page.Items) != 1 {
		t.Fatalf("tenant A must see only its own login log, got total=%d items=%+v", page.Total, page.Items)
	}

	if affected, err := svc.BatchDeleteLoginLogs([]uint64{smokeLoginLogBID}); err != nil || affected != 0 {
		t.Fatalf("tenant A must not delete tenant B's login log (affected=%d, err=%v)", affected, err)
	}
	if countSmokeRows(t, db, "system_log_login", smokeTenantB) != 1 {
		t.Fatalf("tenant B's login log must be untouched")
	}
}

func smokeSecurityEventIsolation(t *testing.T, db *gorm.DB) {
	t.Helper()
	svc := authsecurity.NewService(db, nil, nil).WithTenantContext(smokeMulti(smokeTenantA))

	page, err := svc.ListSecurityEvents(&authsecurity.SecurityEventQuery{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("list security events: %v", err)
	}
	if page.Total != 1 || len(page.Items) != 1 {
		t.Fatalf("tenant A must see only its own security event, got total=%d items=%+v", page.Total, page.Items)
	}
}

func smokeAdminSessionIsolation(t *testing.T, db *gorm.DB) {
	t.Helper()
	rdb := testredis.OpenDB(t, testredis.DBSession)
	oldRDB := database.RDB
	database.RDB = rdb
	t.Cleanup(func() { database.RDB = oldRDB })

	svc := authsession.NewService(db, sessionPolicyStub{}, nil, nil).WithTenantContext(smokeMulti(smokeTenantA))

	page, err := svc.ListAllSessions(&authsession.AdminSessionQuery{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("list admin sessions: %v", err)
	}
	if page.Total != 1 || len(page.Items) != 1 {
		t.Fatalf("tenant A must see only its own session, got total=%d items=%+v", page.Total, page.Items)
	}
}

func smokeDashboardIsolation(t *testing.T, db *gorm.DB) {
	t.Helper()
	svc := platform.NewDashboardService(db).WithTenantContext(smokeMulti(smokeTenantA))

	summary, err := svc.GetSummary()
	if err != nil {
		t.Fatalf("get summary: %v", err)
	}
	if summary.TotalRoles != 1 || summary.TotalUsers != 1 {
		t.Fatalf("tenant A summary must count only its own rows, got roles=%d users=%d", summary.TotalRoles, summary.TotalUsers)
	}
}

// sessionPolicyStub satisfies ListAllSessions' policy read.
type sessionPolicyStub struct{}

func (sessionPolicyStub) GetSessionPolicy() authsession.AuthRuntimePolicy {
	return authsession.AuthRuntimePolicy{SessionIdleMinutes: 30, SessionRetentionDays: 90}
}

// Fixed seed ids so mutation subtests can address tenant B's rows explicitly.
const (
	smokeDeptAID = uint64(1101)
	smokeDeptBID = uint64(2202)
	// Posts may not hang off a root department (post.dept.root_forbidden), so
	// each tenant gets one child department and the seeded posts use it.
	smokeDeptAChildID = uint64(1102)
	smokeDeptBChildID = uint64(2203)
	smokeUserAID      = uint64(1101)
	smokeUserBID      = uint64(2202)
	smokePostAID      = uint64(1101)
	smokePostBID      = uint64(2202)
	smokeRoleAID      = uint64(1101)
	smokeRoleBID      = uint64(2202)
	smokeDictBID      = uint64(2202)
	smokeLoginLogBID  = uint64(2202)
)

// seedSmokeData plants one row per tenant for every converted module. The
// rows are addressed by the fixed ids above so the same operation can be
// proven to succeed on A and fail on B.
func seedSmokeData(t *testing.T, db *gorm.DB) {
	t.Helper()

	rows := []any{
		&role.SystemRole{ID: smokeRoleAID, TenantID: smokeTenantA, RoleName: "A 运营", RoleKey: "ops-a", Status: 1, Sort: 1},
		&role.SystemRole{ID: smokeRoleBID, TenantID: smokeTenantB, RoleName: "B 运营", RoleKey: "ops-b", Status: 1, Sort: 1},
		&post.SystemPost{ID: smokePostAID, TenantID: smokeTenantA, DeptID: smokeDeptAChildID, PostCode: "p-a", PostName: "A 岗位", Sort: 1, Status: 1},
		&post.SystemPost{ID: smokePostBID, TenantID: smokeTenantB, DeptID: smokeDeptBChildID, PostCode: "p-b", PostName: "B 岗位", Sort: 1, Status: 1},
		&dept.SystemDept{ID: smokeDeptAID, TenantID: smokeTenantA, DeptName: "A 部门", ParentID: 0, IsRoot: common.StatusFlagYes, Sort: 1, Status: 1},
		&dept.SystemDept{ID: smokeDeptBID, TenantID: smokeTenantB, DeptName: "B 部门", ParentID: 0, IsRoot: common.StatusFlagYes, Sort: 1, Status: 1},
		&dept.SystemDept{ID: smokeDeptAChildID, TenantID: smokeTenantA, DeptName: "A 子部门", ParentID: smokeDeptAID, Sort: 2, Status: 1},
		&dept.SystemDept{ID: smokeDeptBChildID, TenantID: smokeTenantB, DeptName: "B 子部门", ParentID: smokeDeptBID, Sort: 2, Status: 1},
		&user.SystemUser{ID: smokeUserAID, TenantID: smokeTenantA, Username: "user-a", Password: "seed-hash", Nickname: "A 用户", Status: 1},
		&user.SystemUser{ID: smokeUserBID, TenantID: smokeTenantB, Username: "user-b", Password: "seed-hash", Nickname: "B 用户", Status: 1},
		&dict.SystemDictType{ID: 1101, TenantID: smokeTenantA, DictCode: "smoke-a", DictName: "A 字典", Module: "system", Status: 1},
		&dict.SystemDictType{ID: smokeDictBID, TenantID: smokeTenantB, DictCode: "smoke-b", DictName: "B 字典", Module: "system", Status: 1},
		&setting.SystemSetting{TenantID: tenant.PlatformGlobalTenantID, SettingKey: "site.name", SettingValue: "platform-global", ValueType: "string", GroupKey: "basic", Module: "system"},
		&setting.SystemSetting{TenantID: smokeTenantA, SettingKey: "site.name", SettingValue: "override-a", ValueType: "string", GroupKey: "basic", Module: "system"},
		&setting.SystemSetting{TenantID: smokeTenantB, SettingKey: "site.name", SettingValue: "override-b", ValueType: "string", GroupKey: "basic", Module: "system"},
		&permission.PermissionRoleDataScopePolicy{TenantID: smokeTenantA, RoleKey: "ops-a", Mode: common.DataScopeModeSelf},
		&permission.PermissionRoleDataScopePolicy{TenantID: smokeTenantB, RoleKey: "ops-b", Mode: common.DataScopeModeAll},
		&login.SystemLogLogin{ID: 1101, TenantID: smokeTenantA, Username: "user-a", Status: 1, LoginTime: time.Now()},
		&login.SystemLogLogin{ID: smokeLoginLogBID, TenantID: smokeTenantB, Username: "user-b", Status: 1, LoginTime: time.Now()},
		&authsecurity.SystemAuthSecurityEvent{ID: 1101, TenantID: smokeTenantA, UserID: smokeUserAID, Username: "user-a", EventType: "login.password_wrong", Severity: "warn"},
		&authsecurity.SystemAuthSecurityEvent{ID: 2202, TenantID: smokeTenantB, UserID: smokeUserBID, Username: "user-b", EventType: "login.password_wrong", Severity: "warn"},
		&authsession.SystemUserSession{SessionID: "sess-a", UserID: smokeUserAID, TenantID: smokeTenantA, RefreshJTI: "jti-a", RefreshExpiresAt: time.Now().Add(time.Hour)},
		&authsession.SystemUserSession{SessionID: "sess-b", UserID: smokeUserBID, TenantID: smokeTenantB, RefreshJTI: "jti-b", RefreshExpiresAt: time.Now().Add(time.Hour)},
	}
	for _, row := range rows {
		if err := db.Create(row).Error; err != nil {
			t.Fatalf("seed %T: %v", row, err)
		}
	}
}

// countSmokeRows counts a table's rows for one tenant directly, bypassing the
// services under test so mutation assertions cannot be fooled by a service
// that returns success without doing anything.
func countSmokeRows(t *testing.T, db *gorm.DB, table string, tenantID uint64) int64 {
	t.Helper()
	var n int64
	if err := db.Table(table).Where("tenant_id = ?", tenantID).Count(&n).Error; err != nil {
		t.Fatalf("count %s rows for tenant %d: %v", table, tenantID, err)
	}
	return n
}
