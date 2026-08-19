//nolint:goconst // repeated assertion literals are intentional in this integration test file.
package iam

import (
	"errors"
	"strings"
	"testing"
	"time"

	"pantheon-base/pkg/common"
	"pantheon-base/pkg/database"
	"pantheon-base/pkg/testmysql"

	"gorm.io/gorm"
)

type permissionTestRole struct {
	ID        uint64         `gorm:"primaryKey;autoIncrement"`
	RoleName  string         `gorm:"size:64;not null"`
	RoleKey   string         `gorm:"size:64;not null;uniqueIndex"`
	Sort      int            `gorm:"default:0"`
	Status    int            `gorm:"default:1"`
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

func (permissionTestRole) TableName() string {
	return "system_role"
}

type permissionTestRoleMenu struct {
	RoleID uint64 `gorm:"primaryKey;autoIncrement:false"`
	MenuID uint64 `gorm:"primaryKey;autoIncrement:false"`
}

func (permissionTestRoleMenu) TableName() string {
	return "system_role_menu"
}

type permissionTestRolePermission struct {
	ID            uint64 `gorm:"primaryKey;autoIncrement"`
	RoleID        uint64 `gorm:"not null"`
	PermissionKey string `gorm:"size:128;not null"`
}

func (permissionTestRolePermission) TableName() string {
	return "system_role_permission"
}

type permissionTestMenu struct {
	ID       uint64 `gorm:"primaryKey;autoIncrement"`
	TitleKey string `gorm:"size:64;not null"`
	Path     string `gorm:"size:255;default:''"`
	PagePerm string `gorm:"size:128;default:''"`
	Perms    string `gorm:"size:128;default:''"`
	Type     string `gorm:"size:1;default:'M'"`
	Module   string `gorm:"size:64;default:'system'"`
	Sort     int    `gorm:"default:0"`
}

func (permissionTestMenu) TableName() string {
	return "system_menu"
}

func setupPermissionTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db := testmysql.Open(t)

	if err := db.AutoMigrate(
		&permissionTestRole{},
		&permissionTestRoleMenu{},
		&permissionTestRolePermission{},
		&permissionTestMenu{},
		&database.CasbinRule{},
		&PermissionRoleDataScopePolicy{},
	); err != nil {
		t.Fatalf("migrate permission fixtures: %v", err)
	}

	return db
}

func TestPermissionService_GetWorkbench(t *testing.T) {
	db := setupPermissionTestDB(t)
	service := NewPermissionService(db)

	if err := db.Create(&permissionTestRole{ID: 1, RoleName: "管理员", RoleKey: "admin", Status: 1, Sort: 1}).Error; err != nil {
		t.Fatalf("seed role1: %v", err)
	}
	if err := db.Create(&permissionTestRole{ID: 2, RoleName: "编辑", RoleKey: "editor", Status: 1, Sort: 2}).Error; err != nil {
		t.Fatalf("seed role2: %v", err)
	}
	menus := []permissionTestMenu{
		{ID: 10, TitleKey: "system.menu.user", Path: "/system/user", Module: "system", Type: "C", PagePerm: "system:user:list"},
		{ID: 11, TitleKey: "system.permission.user.create", Path: "/system/user", Module: "system", Type: "F", Perms: "system:user:create"},
		{ID: 12, TitleKey: "system.menu.role", Path: "/system/role", Module: "system", Type: "C", PagePerm: "system:role:list"},
	}
	if err := db.Create(&menus).Error; err != nil {
		t.Fatalf("seed menus: %v", err)
	}
	if err := db.Create(&[]permissionTestRoleMenu{
		{RoleID: 2, MenuID: 10},
		{RoleID: 2, MenuID: 12},
	}).Error; err != nil {
		t.Fatalf("seed role menus: %v", err)
	}
	if err := db.Create(&[]permissionTestRolePermission{
		{RoleID: 2, PermissionKey: "system:user:list"},
		{RoleID: 2, PermissionKey: "system:user:create"},
	}).Error; err != nil {
		t.Fatalf("seed role permissions: %v", err)
	}
	if err := db.Create(&database.CasbinRule{
		PType: "p",
		V0:    "editor",
		V1:    "/api/v1/system/user/list",
		V2:    "GET",
	}).Error; err != nil {
		t.Fatalf("seed casbin rule: %v", err)
	}

	workbench, err := service.GetWorkbench(&PermissionWorkbenchQuery{RoleKey: "editor"})
	if err != nil {
		t.Fatalf("get workbench: %v", err)
	}
	if workbench.Overview.RoleCount != 1 {
		t.Fatalf("expected 1 role, got %d", workbench.Overview.RoleCount)
	}
	if len(workbench.Roles) != 1 {
		t.Fatalf("expected 1 role row, got %d", len(workbench.Roles))
	}

	role := workbench.Roles[0]
	if role.RoleKey != "editor" {
		t.Fatalf("expected role editor, got %s", role.RoleKey)
	}
	if role.MenuCount != 2 {
		t.Fatalf("expected 2 menus, got %d", role.MenuCount)
	}
	if role.PagePermissionCount != 1 {
		t.Fatalf("expected 1 page permission, got %d", role.PagePermissionCount)
	}
	if role.ActionPermissionCount != 1 {
		t.Fatalf("expected 1 action permission, got %d", role.ActionPermissionCount)
	}
	if role.APIPolicyCount != 1 {
		t.Fatalf("expected 1 api policy, got %d", role.APIPolicyCount)
	}
	if role.UnknownPermissionCount != 0 {
		t.Fatalf("expected 0 unknown permissions, got %d", role.UnknownPermissionCount)
	}
}

func TestPermissionService_DataScopePolicies(t *testing.T) {
	db := setupPermissionTestDB(t)
	service := NewPermissionService(db)

	if err := db.Create(&permissionTestRole{ID: 1, RoleName: "编辑", RoleKey: "editor", Status: 1, Sort: 1}).Error; err != nil {
		t.Fatalf("seed role: %v", err)
	}

	initial, err := service.ListDataScopePolicies(nil)
	if err != nil {
		t.Fatalf("list initial data scope policies: %v", err)
	}
	if initial.Total != 1 || len(initial.Items) != 1 {
		t.Fatalf("expected one role data scope row, got %+v", initial)
	}
	if initial.Items[0].Mode != "all" || initial.Items[0].PolicyExists {
		t.Fatalf("expected default all mode without explicit policy, got %+v", initial.Items[0])
	}

	updated, err := service.UpdateDataScopePolicy("editor", &PermissionDataScopePolicyUpdateReq{
		Mode:    "custom",
		DeptIDs: []uint64{20, 10, 20, 0},
	})
	if err != nil {
		t.Fatalf("update data scope policy: %v", err)
	}
	if updated.Mode != "custom" || !updated.PolicyExists {
		t.Fatalf("expected custom explicit policy, got %+v", updated)
	}
	if len(updated.DeptIDs) != 2 || updated.DeptIDs[0] != 10 || updated.DeptIDs[1] != 20 {
		t.Fatalf("expected normalized dept ids [10 20], got %+v", updated.DeptIDs)
	}

	updated, err = service.UpdateDataScopePolicy("editor", &PermissionDataScopePolicyUpdateReq{Mode: "dept"})
	if err != nil {
		t.Fatalf("update data scope policy to dept: %v", err)
	}
	if updated.Mode != "dept" || len(updated.DeptIDs) != 0 {
		t.Fatalf("expected dept mode without custom dept ids, got %+v", updated)
	}
}

func TestPermissionService_DataScopePolicyRequiresCustomDeptIDs(t *testing.T) {
	db := setupPermissionTestDB(t)
	service := NewPermissionService(db)

	if err := db.Create(&permissionTestRole{ID: 1, RoleName: "编辑", RoleKey: "editor", Status: 1, Sort: 1}).Error; err != nil {
		t.Fatalf("seed role: %v", err)
	}

	_, err := service.UpdateDataScopePolicy("editor", &PermissionDataScopePolicyUpdateReq{Mode: "custom"})
	if err == nil || !strings.Contains(err.Error(), "permission.data_scope.dept_required") {
		t.Fatalf("expected custom dept required error, got %v", err)
	}
}

func TestPermissionService_GetWorkbenchIntegrityFilter(t *testing.T) {
	db := setupPermissionTestDB(t)
	service := NewPermissionService(db)

	roles := []permissionTestRole{
		{ID: 1, RoleName: "Clean Role", RoleKey: "clean_role", Status: 1, Sort: 1},
		{ID: 2, RoleName: "Dirty Role", RoleKey: "dirty_role", Status: 2, Sort: 2},
	}
	if err := db.Create(&roles).Error; err != nil {
		t.Fatalf("seed roles: %v", err)
	}

	menus := []permissionTestMenu{
		{ID: 10, TitleKey: "system.menu.user", Path: "/system/user", Module: "system", Type: "C", PagePerm: "system:user:list"},
		{ID: 11, TitleKey: "system.permission.user.create", Path: "/system/user", Module: "system", Type: "F", Perms: "system:user:create"},
	}
	if err := db.Create(&menus).Error; err != nil {
		t.Fatalf("seed menus: %v", err)
	}

	if err := db.Create(&[]permissionTestRolePermission{
		{RoleID: 1, PermissionKey: "system:user:list"},
		{RoleID: 2, PermissionKey: "system:user:create"},
		{RoleID: 2, PermissionKey: "system:unknown:manage"},
	}).Error; err != nil {
		t.Fatalf("seed permissions: %v", err)
	}

	if err := db.Create(&[]database.CasbinRule{
		{PType: "p", V0: "clean_role", V1: "/api/v1/system/user/list", V2: "GET"},
		{PType: "p", V0: "dirty_role", V1: "/api/v1/system/user/create", V2: "POST"},
	}).Error; err != nil {
		t.Fatalf("seed policies: %v", err)
	}

	allWorkbench, err := service.GetWorkbench(nil)
	if err != nil {
		t.Fatalf("get workbench: %v", err)
	}
	if allWorkbench.Overview.RoleCount != 2 {
		t.Fatalf("expected 2 roles, got %d", allWorkbench.Overview.RoleCount)
	}
	if allWorkbench.Overview.UnknownPermissionAssignmentCount != 1 {
		t.Fatalf("expected 1 unknown assignment, got %d", allWorkbench.Overview.UnknownPermissionAssignmentCount)
	}

	assertWorkbenchIntegrityFilter(t, service, "unknown", "dirty_role", 1)
	assertWorkbenchIntegrityFilter(t, service, "clean", "clean_role", 0)
}

func assertWorkbenchIntegrityFilter(t *testing.T, service *PermissionService, integrity, wantRoleKey string, wantUnknown int) {
	t.Helper()
	workbench, err := service.GetWorkbench(&PermissionWorkbenchQuery{Integrity: integrity})
	if err != nil {
		t.Fatalf("get %s workbench: %v", integrity, err)
	}
	if len(workbench.Roles) != 1 || workbench.Roles[0].RoleKey != wantRoleKey {
		t.Fatalf("expected only %s in %s filter, got %+v", wantRoleKey, integrity, workbench.Roles)
	}
	if workbench.Overview.RoleCount != 1 || workbench.Overview.UnknownPermissionAssignmentCount != wantUnknown {
		t.Fatalf("unexpected %s overview: %+v", integrity, workbench.Overview)
	}
}

func TestPermissionService_GetWorkbenchCoverageFilter(t *testing.T) {
	db := setupPermissionTestDB(t)
	service := NewPermissionService(db)

	roles := []permissionTestRole{
		{ID: 1, RoleName: "Page Gap", RoleKey: "page_gap", Status: 1, Sort: 1},
		{ID: 2, RoleName: "API Gap", RoleKey: "api_gap", Status: 1, Sort: 2},
		{ID: 3, RoleName: "Complete", RoleKey: "complete_role", Status: 1, Sort: 3},
	}
	if err := db.Create(&roles).Error; err != nil {
		t.Fatalf("seed roles: %v", err)
	}

	menus := []permissionTestMenu{
		{ID: 10, TitleKey: "system.menu.user", Path: "/system/user", Module: "system", Type: "C", PagePerm: "system:user:list"},
		{ID: 11, TitleKey: "system.permission.user.create", Path: "/system/user", Module: "system", Type: "F", Perms: "system:user:create"},
	}
	if err := db.Create(&menus).Error; err != nil {
		t.Fatalf("seed menus: %v", err)
	}
	if err := db.Create(&[]permissionTestRoleMenu{
		{RoleID: 1, MenuID: 10},
		{RoleID: 2, MenuID: 10},
		{RoleID: 3, MenuID: 10},
	}).Error; err != nil {
		t.Fatalf("seed role menus: %v", err)
	}
	if err := db.Create(&[]permissionTestRolePermission{
		{RoleID: 2, PermissionKey: "system:user:list"},
		{RoleID: 2, PermissionKey: "system:user:create"},
		{RoleID: 3, PermissionKey: "system:user:list"},
		{RoleID: 3, PermissionKey: "system:user:create"},
	}).Error; err != nil {
		t.Fatalf("seed role permissions: %v", err)
	}
	if err := db.Create(&[]database.CasbinRule{
		{PType: "p", V0: "complete_role", V1: "/api/v1/system/user/list", V2: "GET"},
		{PType: "p", V0: "complete_role", V1: "/api/v1/system/user/create", V2: "POST"},
	}).Error; err != nil {
		t.Fatalf("seed casbin rules: %v", err)
	}

	workbench, err := service.GetWorkbench(nil)
	if err != nil {
		t.Fatalf("get workbench: %v", err)
	}
	if workbench.Overview.PageGapRoleCount != 1 {
		t.Fatalf("expected 1 page gap role, got %d", workbench.Overview.PageGapRoleCount)
	}
	if workbench.Overview.APIGapRoleCount != 1 {
		t.Fatalf("expected 1 api gap role, got %d", workbench.Overview.APIGapRoleCount)
	}

	assertWorkbenchCoverageFilter(t, service, "page-gap", "page_gap")
	assertWorkbenchCoverageFilter(t, service, "api-gap", "api_gap")
	assertWorkbenchCoverageFilter(t, service, "complete", "complete_role")
}

func assertWorkbenchCoverageFilter(t *testing.T, service *PermissionService, coverage, wantRoleKey string) {
	t.Helper()
	workbench, err := service.GetWorkbench(&PermissionWorkbenchQuery{Coverage: coverage})
	if err != nil {
		t.Fatalf("get %s workbench: %v", coverage, err)
	}
	if len(workbench.Roles) != 1 || workbench.Roles[0].RoleKey != wantRoleKey {
		t.Fatalf("expected only %s role, got %+v", wantRoleKey, workbench.Roles)
	}
	role := workbench.Roles[0]
	if coverage == "page-gap" && !role.HasPageGap {
		t.Fatalf("expected %s to have page gap", wantRoleKey)
	}
	if coverage == "api-gap" && !role.HasAPIGap {
		t.Fatalf("expected %s to have api gap", wantRoleKey)
	}
}

func TestPermissionService_GetWorkbenchIncludesRemediationGovernanceSummary(t *testing.T) {
	db := setupPermissionTestDB(t)
	service := NewPermissionService(db)
	if err := service.Migrate(); err != nil {
		t.Fatalf("migrate permission service: %v", err)
	}

	roles := []permissionTestRole{
		{ID: 1, RoleName: "Pending Role", RoleKey: "pending_role", Status: 1, Sort: 1},
		{ID: 2, RoleName: "Remediated Role", RoleKey: "remediated_role", Status: 1, Sort: 2},
		{ID: 3, RoleName: "Clean Role", RoleKey: "clean_role", Status: 1, Sort: 3},
	}
	if err := db.Create(&roles).Error; err != nil {
		t.Fatalf("seed roles: %v", err)
	}

	menus := []permissionTestMenu{
		{ID: 10, TitleKey: "system.menu.generator", Path: "/system/generator", Module: "system.lowcode", Type: "C", PagePerm: "system:generator:use"},
		{ID: 11, TitleKey: "system.permission.module.generate", Path: "/system/generator", Module: "system.lowcode", Type: "F", Perms: "system:module:generate"},
	}
	if err := db.Create(&menus).Error; err != nil {
		t.Fatalf("seed menus: %v", err)
	}
	if err := db.Create(&[]permissionTestRolePermission{
		{RoleID: 1, PermissionKey: "system:generator:use"},
		{RoleID: 1, PermissionKey: "system:module:generate"},
		{RoleID: 2, PermissionKey: "system:generator:use"},
		{RoleID: 2, PermissionKey: "system:module:generate"},
	}).Error; err != nil {
		t.Fatalf("seed role permissions: %v", err)
	}
	if err := db.Create(&database.CasbinRule{
		PType: "p",
		V0:    "remediated_role",
		V1:    "/api/v1/lowcode/dynamic-modules/generate",
		V2:    "POST",
	}).Error; err != nil {
		t.Fatalf("seed casbin rule: %v", err)
	}

	now := time.Now()
	if err := db.Create(&[]PermissionWorkbenchRemediationEvent{
		{
			RoleKey:      "pending_role",
			IssueType:    "api-gap",
			IssueKey:     "POST /api/v1/lowcode/dynamic-modules/generate",
			BeforeState:  "api-gap",
			AfterState:   "complete",
			Action:       "remediated",
			CreatedCount: 1,
			SkippedCount: 0,
			CreatedAt:    now.Add(-time.Hour),
		},
		{
			RoleKey:      "remediated_role",
			IssueType:    "api-gap",
			IssueKey:     "POST /api/v1/lowcode/dynamic-modules/generate",
			BeforeState:  "api-gap",
			AfterState:   "complete",
			Action:       "remediated",
			CreatedCount: 1,
			SkippedCount: 0,
			CreatedAt:    now,
		},
	}).Error; err != nil {
		t.Fatalf("seed remediation events: %v", err)
	}

	workbench, err := service.GetWorkbench(nil)
	if err != nil {
		t.Fatalf("get workbench: %v", err)
	}

	assertRemediationGovernanceOverview(t, workbench)
	roleByKey := permissionWorkbenchRolesByKey(workbench.Roles)
	cases := []struct {
		roleKey       string
		status        string
		action        string
		wantTimestamp bool
	}{
		{roleKey: "pending_role", status: "pending", action: "remediated", wantTimestamp: true},
		{roleKey: "remediated_role", status: "remediated", action: "remediated", wantTimestamp: true},
		{roleKey: "clean_role", status: "clean"},
	}
	for _, tc := range cases {
		assertRemediationGovernanceRole(t, roleByKey, tc.roleKey, tc.status, tc.action, tc.wantTimestamp)
	}
}

func assertRemediationGovernanceOverview(t *testing.T, workbench *PermissionWorkbenchResp) {
	t.Helper()
	if workbench.Overview.PendingRemediationRoleCount != 1 ||
		workbench.Overview.RemediatedRoleCount != 1 ||
		workbench.Overview.RecentRemediationCount != 2 {
		t.Fatalf("unexpected remediation governance overview: %+v", workbench.Overview)
	}
}

func permissionWorkbenchRolesByKey(roles []PermissionWorkbenchRoleResp) map[string]PermissionWorkbenchRoleResp {
	roleByKey := make(map[string]PermissionWorkbenchRoleResp, len(roles))
	for _, item := range roles {
		roleByKey[item.RoleKey] = item
	}
	return roleByKey
}

func assertRemediationGovernanceRole(
	t *testing.T,
	roleByKey map[string]PermissionWorkbenchRoleResp,
	roleKey, wantStatus, wantAction string,
	wantTimestamp bool,
) {
	t.Helper()
	role, ok := roleByKey[roleKey]
	if !ok {
		t.Fatalf("missing %s in workbench", roleKey)
	}
	if role.GovernanceStatus != wantStatus || role.LastRemediationAction != wantAction {
		t.Fatalf("unexpected governance state for %s: %+v", roleKey, role)
	}
	if (role.LastRemediationAt != "") != wantTimestamp {
		t.Fatalf("unexpected remediation timestamp for %s: %q", roleKey, role.LastRemediationAt)
	}
}

func TestPermissionService_GetWorkbenchUsesVersionedRemediationTableName(t *testing.T) {
	db := setupPermissionTestDB(t)
	service := NewPermissionService(db)

	if err := db.Create(&permissionTestRole{
		ID:       1,
		RoleName: "Versioned Remediated",
		RoleKey:  "versioned_remediated",
		Status:   1,
		Sort:     1,
	}).Error; err != nil {
		t.Fatalf("seed role: %v", err)
	}
	if err := db.Create(&permissionTestMenu{
		ID:       10,
		TitleKey: "system.menu.generator",
		Path:     "/system/generator",
		Module:   "system.lowcode",
		Type:     "C",
		PagePerm: "system:generator:use",
	}).Error; err != nil {
		t.Fatalf("seed menu: %v", err)
	}
	if err := db.Create(&permissionTestRolePermission{
		RoleID:        1,
		PermissionKey: "system:generator:use",
	}).Error; err != nil {
		t.Fatalf("seed role permission: %v", err)
	}
	if err := db.Create(&database.CasbinRule{
		PType: "p",
		V0:    "versioned_remediated",
		V1:    "/api/v1/lowcode/dynamic-modules/generate",
		V2:    "POST",
	}).Error; err != nil {
		t.Fatalf("seed api policy: %v", err)
	}
	if err := db.Exec(`
CREATE TABLE IF NOT EXISTS permission_workbench_remediation_event (
  id BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  role_key VARCHAR(64) NOT NULL,
  issue_type VARCHAR(32) NOT NULL,
  issue_key VARCHAR(255) NOT NULL,
  before_state VARCHAR(32) NOT NULL,
  after_state VARCHAR(32) NOT NULL,
  action VARCHAR(32) NOT NULL,
  created_count INT DEFAULT 0,
  skipped_count INT DEFAULT 0,
  created_at DATETIME(3) DEFAULT NULL,
  PRIMARY KEY (id),
  INDEX idx_permission_remediation_role_created (role_key, created_at),
  INDEX idx_permission_remediation_issue_type (issue_type)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
`).Error; err != nil {
		t.Fatalf("create versioned remediation table: %v", err)
	}
	if err := db.Exec(`
INSERT INTO permission_workbench_remediation_event
  (role_key, issue_type, issue_key, before_state, after_state, action, created_count, skipped_count, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
`, "versioned_remediated", "api-gap", "POST /api/v1/lowcode/dynamic-modules/generate", "api-gap", "complete", "remediated", 1, 0, time.Now()).Error; err != nil {
		t.Fatalf("seed versioned remediation event: %v", err)
	}

	workbench, err := service.GetWorkbench(&PermissionWorkbenchQuery{RoleKey: "versioned_remediated"})
	if err != nil {
		t.Fatalf("get workbench: %v", err)
	}
	if len(workbench.Roles) != 1 {
		t.Fatalf("expected 1 role, got %+v", workbench.Roles)
	}
	if workbench.Roles[0].GovernanceStatus != "remediated" {
		t.Fatalf("expected governance status remediated, got %+v", workbench.Roles[0])
	}
	if workbench.Overview.RemediatedRoleCount != 1 {
		t.Fatalf("expected remediated role count 1, got %+v", workbench.Overview)
	}
}

func TestPermissionService_GetWorkbenchDetectsSpecificRequiredAPIPolicyGap(t *testing.T) {
	db := setupPermissionTestDB(t)
	service := NewPermissionService(db)

	roles := []permissionTestRole{
		{ID: 1, RoleName: "Generate Gap", RoleKey: "generate_gap", Status: 1, Sort: 1},
		{ID: 2, RoleName: "Generate Ready", RoleKey: "generate_ready", Status: 1, Sort: 2},
	}
	if err := db.Create(&roles).Error; err != nil {
		t.Fatalf("seed roles: %v", err)
	}

	menus := []permissionTestMenu{
		{ID: 10, TitleKey: "system.menu.generator", Path: "/system/generator", Module: "system.lowcode", Type: "C", PagePerm: "system:generator:use"},
		{ID: 11, TitleKey: "system.permission.module.generate", Path: "/system/generator", Module: "system.lowcode", Type: "F", Perms: "system:module:generate"},
	}
	if err := db.Create(&menus).Error; err != nil {
		t.Fatalf("seed menus: %v", err)
	}
	if err := db.Create(&[]permissionTestRolePermission{
		{RoleID: 1, PermissionKey: "system:generator:use"},
		{RoleID: 1, PermissionKey: "system:module:generate"},
		{RoleID: 2, PermissionKey: "system:generator:use"},
		{RoleID: 2, PermissionKey: "system:module:generate"},
	}).Error; err != nil {
		t.Fatalf("seed role permissions: %v", err)
	}
	if err := db.Create(&[]database.CasbinRule{
		{PType: "p", V0: "generate_gap", V1: "/api/v1/lowcode/dynamic-modules", V2: "GET"},
		{PType: "p", V0: "generate_ready", V1: "/api/v1/lowcode/dynamic-modules/generate", V2: "POST"},
	}).Error; err != nil {
		t.Fatalf("seed casbin rules: %v", err)
	}

	workbench, err := service.GetWorkbench(nil)
	if err != nil {
		t.Fatalf("get workbench: %v", err)
	}
	if len(workbench.Roles) != 2 {
		t.Fatalf("expected 2 roles, got %d", len(workbench.Roles))
	}

	var gapRole PermissionWorkbenchRoleResp
	var readyRole PermissionWorkbenchRoleResp
	for _, item := range workbench.Roles {
		switch item.RoleKey {
		case "generate_gap":
			gapRole = item
		case "generate_ready":
			readyRole = item
		}
	}

	if !gapRole.HasAPIGap {
		t.Fatalf("expected generate_gap to have api gap, got %+v", gapRole)
	}
	if gapRole.MissingAPIPolicyCount != 1 {
		t.Fatalf("expected 1 missing api policy, got %+v", gapRole.MissingAPIPolicies)
	}
	if len(gapRole.MissingAPIPolicies) != 1 || gapRole.MissingAPIPolicies[0].Path != "/api/v1/lowcode/dynamic-modules/generate" || gapRole.MissingAPIPolicies[0].Method != "POST" {
		t.Fatalf("unexpected missing policies: %+v", gapRole.MissingAPIPolicies)
	}
	if readyRole.HasAPIGap {
		t.Fatalf("expected generate_ready to be api-complete, got %+v", readyRole)
	}
	if readyRole.MissingAPIPolicyCount != 0 {
		t.Fatalf("expected 0 missing api policies, got %+v", readyRole.MissingAPIPolicies)
	}
}

func TestPermissionWorkbenchRequiresSecurityEventListPolicy(t *testing.T) {
	policies := requiredAPIPoliciesByPermissionKey("system:security-event:list")
	if len(policies) != 1 {
		t.Fatalf("expected one required policy, got %+v", policies)
	}
	if policies[0].Path != "/api/v1/system/security-event/list" || policies[0].Method != "GET" {
		t.Fatalf("unexpected required policy: %+v", policies[0])
	}
}

func TestPermissionWorkbenchRequiresSecurityEventActionPolicies(t *testing.T) {
	acknowledge := requiredAPIPoliciesByPermissionKey("system:security-event:acknowledge")
	if len(acknowledge) != 2 {
		t.Fatalf("expected two required acknowledge policies, got %+v", acknowledge)
	}
	if acknowledge[0].Path != "/api/v1/system/security-event/:id/acknowledge" || acknowledge[0].Method != "POST" {
		t.Fatalf("unexpected acknowledge policy: %+v", acknowledge[0])
	}
	if acknowledge[1].Path != "/api/v1/system/security-event/batch-acknowledge" || acknowledge[1].Method != "POST" {
		t.Fatalf("unexpected batch acknowledge policy: %+v", acknowledge[1])
	}

	clear := requiredAPIPoliciesByPermissionKey("system:security-event:clear")
	if len(clear) != 1 {
		t.Fatalf("expected one required clear policy, got %+v", clear)
	}
	if clear[0].Path != "/api/v1/system/security-event/cleanup" || clear[0].Method != "POST" {
		t.Fatalf("unexpected clear policy: %+v", clear[0])
	}
}

func TestProtectedManagementPolicyGuard(t *testing.T) {
	cases := []struct {
		name     string
		roleKeys []string
		path     string
		want     bool
	}{
		{name: "admin can write protected", roleKeys: []string{"admin"}, path: "/api/v1/system/permission/list", want: true},
		{name: "non-admin blocked on permission", roleKeys: []string{"editor"}, path: "/api/v1/system/permission/list", want: false},
		{name: "non-admin blocked on role", roleKeys: []string{"editor"}, path: "/api/v1/system/role/list", want: false},
		{name: "non-admin can write business policy", roleKeys: []string{"editor"}, path: "/api/v1/system/user/list", want: true},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			if got := canWritePolicy(tt.roleKeys, tt.path); got != tt.want {
				t.Fatalf("canWritePolicy() = %v, want %v", got, tt.want)
			}
		})
	}

	if !isProtectedManagementPolicy("/api/v1/system/permission/list") {
		t.Fatal("expected permission list to be protected")
	}
	if !isProtectedManagementPolicy("/api/v1/system/role/list") {
		t.Fatal("expected role list to be protected")
	}
	if isProtectedManagementPolicy("/api/v1/system/user/list") {
		t.Fatal("expected user list to remain writable")
	}
}

func TestPermissionService_PolicyWriteProtection(t *testing.T) {
	db := setupPermissionTestDB(t)
	service := NewPermissionService(db)

	roles := []permissionTestRole{
		{ID: 1, RoleName: "管理员", RoleKey: "admin", Status: 1, Sort: 1},
		{ID: 2, RoleName: "编辑", RoleKey: "editor", Status: 1, Sort: 2},
	}
	if err := db.Create(&roles).Error; err != nil {
		t.Fatalf("seed roles: %v", err)
	}

	protectedReq := &PermissionPolicyCreateReq{
		RoleKey: "editor",
		Path:    "/api/v1/system/permission/list",
		Method:  "GET",
	}
	adminProtected, err := service.CreatePolicy([]string{"admin"}, protectedReq)
	if err != nil {
		t.Fatalf("admin create protected policy: %v", err)
	}
	if adminProtected.Path != protectedReq.Path {
		t.Fatalf("unexpected protected policy path: %+v", adminProtected)
	}

	if _, err := service.CreatePolicy([]string{"editor"}, protectedReq); err == nil || !errors.Is(err, common.ErrForbidden) || common.ErrMessage(err) != "permission.escalation.forbidden" {
		t.Fatalf("expected forbidden create for non-admin, got %v", err)
	}

	normalResp, err := service.CreatePolicy([]string{"editor"}, &PermissionPolicyCreateReq{
		RoleKey: "editor",
		Path:    "/api/v1/system/user/list",
		Method:  "GET",
	})
	if err != nil {
		t.Fatalf("editor create normal policy: %v", err)
	}

	if _, err := service.UpdatePolicy([]string{"editor"}, normalResp.ID, &PermissionPolicyUpdateReq{
		RoleKey: "editor",
		Path:    "/api/v1/system/role/list",
		Method:  "GET",
	}); err == nil || !errors.Is(err, common.ErrForbidden) || common.ErrMessage(err) != "permission.escalation.forbidden" {
		t.Fatalf("expected forbidden update for non-admin, got %v", err)
	}

	if err := service.DeletePolicy([]string{"editor"}, adminProtected.ID); err == nil || !errors.Is(err, common.ErrForbidden) || common.ErrMessage(err) != "permission.escalation.forbidden" {
		t.Fatalf("expected forbidden delete for non-admin, got %v", err)
	}

	importResult, err := service.ImportPolicies([]string{"editor"}, [][]string{
		{"roleKey", "path", "method"},
		{"editor", "/api/v1/system/permission/list", "GET"},
	})
	if err == nil || !errors.Is(err, common.ErrForbidden) || common.ErrMessage(err) != "permission.escalation.forbidden" {
		t.Fatalf("expected forbidden import for non-admin, got result=%+v err=%v", importResult, err)
	}
}

func TestPermissionService_RemediateWorkbenchPolicies(t *testing.T) {
	db := setupPermissionTestDB(t)
	service := NewPermissionService(db)
	if err := service.Migrate(); err != nil {
		t.Fatalf("migrate permission service: %v", err)
	}

	if err := db.Create(&permissionTestRole{ID: 1, RoleName: "Generate Gap", RoleKey: "generate_gap", Status: 1, Sort: 1}).Error; err != nil {
		t.Fatalf("seed role: %v", err)
	}
	menus := []permissionTestMenu{
		{ID: 10, TitleKey: "system.menu.generator", Path: "/system/generator", Module: "system.lowcode", Type: "C", PagePerm: "system:generator:use"},
		{ID: 11, TitleKey: "system.permission.module.generate", Path: "/system/generator", Module: "system.lowcode", Type: "F", Perms: "system:module:generate"},
	}
	if err := db.Create(&menus).Error; err != nil {
		t.Fatalf("seed menus: %v", err)
	}
	if err := db.Create(&[]permissionTestRolePermission{
		{RoleID: 1, PermissionKey: "system:generator:use"},
		{RoleID: 1, PermissionKey: "system:module:generate"},
	}).Error; err != nil {
		t.Fatalf("seed role permissions: %v", err)
	}

	resp, err := service.RemediateWorkbenchPolicies(&PermissionWorkbenchRemediateReq{RoleKey: "generate_gap"})
	if err != nil {
		t.Fatalf("remediate workbench policies: %v", err)
	}
	assertCreatedRemediationPolicy(t, resp)
	assertPersistedRemediationPolicy(t, db)

	secondResp, err := service.RemediateWorkbenchPolicies(&PermissionWorkbenchRemediateReq{RoleKey: "generate_gap"})
	if err != nil {
		t.Fatalf("second remediate workbench policies: %v", err)
	}
	if secondResp.CreatedCount != 0 || len(secondResp.CreatedPolicies) != 0 || secondResp.SkippedCount != 1 {
		t.Fatalf("expected skipped count 1 after remediation, got %+v", secondResp)
	}

	events, err := service.ListWorkbenchRemediationEvents(&PermissionWorkbenchRemediationQuery{RoleKey: "generate_gap"})
	if err != nil {
		t.Fatalf("list remediation events: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 remediation events, got %+v", events)
	}
	assertRemediationEvents(t, events)
}

func assertCreatedRemediationPolicy(t *testing.T, resp *PermissionWorkbenchRemediateResp) {
	t.Helper()
	if resp.RoleKey != "generate_gap" || resp.CreatedCount != 1 || len(resp.CreatedPolicies) != 1 || resp.SkippedCount != 0 {
		t.Fatalf("unexpected remediation response: %+v", resp)
	}
	if resp.CreatedPolicies[0].Path != "/api/v1/lowcode/dynamic-modules/generate" || resp.CreatedPolicies[0].Method != "POST" {
		t.Fatalf("unexpected created policy: %+v", resp.CreatedPolicies[0])
	}
}

func assertPersistedRemediationPolicy(t *testing.T, db *gorm.DB) {
	t.Helper()
	var count int64
	if err := db.Model(&database.CasbinRule{}).
		Where("ptype = ? AND v0 = ? AND v1 = ? AND v2 = ?", "p", "generate_gap", "/api/v1/lowcode/dynamic-modules/generate", "POST").
		Count(&count).Error; err != nil {
		t.Fatalf("count created policy: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 persisted policy, got %d", count)
	}
}

func assertRemediationEvents(t *testing.T, events []PermissionWorkbenchRemediationResp) {
	t.Helper()
	if events[0].RoleKey != "generate_gap" || events[0].IssueType != "api-gap" || events[0].Action != "remediated" || events[0].CreatedCount != 1 || events[0].AfterState != "complete" {
		t.Fatalf("unexpected first remediation event: %+v", events[0])
	}
	if events[1].Action != "noop" || events[1].CreatedCount != 0 || events[1].SkippedCount != 1 {
		t.Fatalf("unexpected second remediation event: %+v", events[1])
	}
}

func TestPermissionService_GetWorkbenchExcludesDeletedRoles(t *testing.T) {
	db := setupPermissionTestDB(t)
	service := NewPermissionService(db)

	deletedAt := gorm.DeletedAt{Time: time.Now(), Valid: true}
	if err := db.Create(&permissionTestRole{ID: 1, RoleName: "Deleted Role", RoleKey: "__deleted_role_1", Status: 1, DeletedAt: deletedAt}).Error; err != nil {
		t.Fatalf("seed deleted role: %v", err)
	}
	if err := db.Create(&permissionTestRole{ID: 2, RoleName: "Active Role", RoleKey: "active_role", Status: 1}).Error; err != nil {
		t.Fatalf("seed active role: %v", err)
	}

	workbench, err := service.GetWorkbench(nil)
	if err != nil {
		t.Fatalf("get workbench: %v", err)
	}
	if workbench.Overview.RoleCount != 1 {
		t.Fatalf("expected 1 active role, got %d", workbench.Overview.RoleCount)
	}
	if len(workbench.Roles) != 1 || workbench.Roles[0].RoleKey != "active_role" {
		t.Fatalf("expected only active_role, got %+v", workbench.Roles)
	}
}

func TestPermissionService_MigrateRemovesOrphanPolicies(t *testing.T) {
	db := setupPermissionTestDB(t)
	service := NewPermissionService(db)

	deletedAt := gorm.DeletedAt{Time: time.Now(), Valid: true}
	if err := db.Create(&permissionTestRole{ID: 1, RoleName: "Active Role", RoleKey: "active_role", Status: 1}).Error; err != nil {
		t.Fatalf("seed active role: %v", err)
	}
	if err := db.Create(&permissionTestRole{ID: 2, RoleName: "Deleted Role", RoleKey: "__deleted_role_2", Status: 1, DeletedAt: deletedAt}).Error; err != nil {
		t.Fatalf("seed deleted role: %v", err)
	}
	if err := db.Create(&[]database.CasbinRule{
		{PType: "p", V0: "active_role", V1: "/api/v1/system/active", V2: "GET"},
		{PType: "p", V0: "missing_role", V1: "/api/v1/system/missing", V2: "GET"},
		{PType: "p", V0: "__deleted_role_2", V1: "/api/v1/system/deleted", V2: "GET"},
	}).Error; err != nil {
		t.Fatalf("seed casbin policies: %v", err)
	}

	if err := service.Migrate(); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	var policies []database.CasbinRule
	if err := db.Model(&database.CasbinRule{}).Order("v0 asc").Find(&policies).Error; err != nil {
		t.Fatalf("list policies: %v", err)
	}
	if len(policies) != 1 || policies[0].V0 != "active_role" {
		t.Fatalf("expected only active role policy, got %+v", policies)
	}
}

func TestPermissionService_ImportTemplateAndExport(t *testing.T) {
	db := setupPermissionTestDB(t)
	service := NewPermissionService(db)

	if err := db.Create(&permissionTestRole{ID: 1, RoleName: "管理员", RoleKey: "admin", Status: 1}).Error; err != nil {
		t.Fatalf("seed role: %v", err)
	}

	template := service.BuildImportTemplate()
	if len(template.Rows) == 0 || !strings.HasPrefix(template.Rows[0][0], "#") {
		t.Fatalf("expected template to include ignored instruction rows, got %+v", template.Rows)
	}
	templateResult, err := service.ImportPolicies(nil, append([][]string{template.Headers}, template.Rows...))
	if err != nil {
		t.Fatalf("import template comments: %v", err)
	}
	if !templateResult.Applied || templateResult.Created != 0 || templateResult.Failed != 0 {
		t.Fatalf("expected template comments to be ignored, got %+v", templateResult)
	}

	result, err := service.ImportPolicies(nil, [][]string{
		template.Headers,
		{"admin", "/api/v1/system/user/list", "GET"},
	})
	if err != nil {
		t.Fatalf("import policy: %v", err)
	}
	if !result.Applied || result.Created != 1 || result.Failed != 0 {
		t.Fatalf("unexpected import result: %+v", result)
	}

	exported, err := service.ExportPolicies(&PermissionPolicyQuery{RoleKey: "admin"})
	if err != nil {
		t.Fatalf("export policy: %v", err)
	}
	if len(exported.Rows) != 1 || exported.Rows[0][0] != "admin" || exported.Rows[0][2] != "GET" {
		t.Fatalf("unexpected export rows: %+v", exported.Rows)
	}
}

func TestPermissionService_ExportWorkbench(t *testing.T) {
	db := setupPermissionTestDB(t)
	service := NewPermissionService(db)

	if err := db.Create(&[]permissionTestRole{
		{ID: 1, RoleName: "Page Gap", RoleKey: "page_gap", Status: 1, Sort: 1},
		{ID: 2, RoleName: "API Gap", RoleKey: "api_gap", Status: 2, Sort: 2},
	}).Error; err != nil {
		t.Fatalf("seed roles: %v", err)
	}
	if err := db.Create(&[]permissionTestMenu{
		{ID: 10, TitleKey: "system.menu.user", Path: "/system/user", Module: "system", Type: "C", PagePerm: "system:user:list"},
		{ID: 11, TitleKey: "system.permission.user.create", Path: "/system/user", Module: "system", Type: "F", Perms: "system:user:create"},
	}).Error; err != nil {
		t.Fatalf("seed menus: %v", err)
	}
	if err := db.Create(&[]permissionTestRoleMenu{
		{RoleID: 1, MenuID: 10},
		{RoleID: 2, MenuID: 10},
	}).Error; err != nil {
		t.Fatalf("seed role menus: %v", err)
	}
	if err := db.Create(&[]permissionTestRolePermission{
		{RoleID: 2, PermissionKey: "system:user:list"},
		{RoleID: 2, PermissionKey: "system:unknown:manage"},
	}).Error; err != nil {
		t.Fatalf("seed role permissions: %v", err)
	}

	file, err := service.ExportWorkbench(&PermissionWorkbenchQuery{})
	if err != nil {
		t.Fatalf("export workbench: %v", err)
	}
	if file.Filename != "system-permission-workbench-export.csv" {
		t.Fatalf("unexpected filename: %s", file.Filename)
	}
	if len(file.Rows) != 2 {
		t.Fatalf("expected 2 rows, got %+v", file.Rows)
	}
	if file.Rows[0][1] != "page_gap" || file.Rows[0][8] != "true" || file.Rows[0][10] != "page-gap" {
		t.Fatalf("unexpected page gap row: %+v", file.Rows[0])
	}
	if file.Rows[1][1] != "api_gap" || file.Rows[1][9] != "true" || file.Rows[1][10] != "api-gap" || file.Rows[1][11] != "system:unknown:manage" {
		t.Fatalf("unexpected api gap row: %+v", file.Rows[1])
	}
}
