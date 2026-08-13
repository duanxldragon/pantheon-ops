package platform

import (
	"testing"
	"time"

	"pantheon-base/internal/middleware"
	login "pantheon-base/modules/auth/login"
	security "pantheon-base/modules/auth/security"
	session "pantheon-base/modules/auth/session"
	dictcfg "pantheon-base/modules/system/config/dict"
	settingcfg "pantheon-base/modules/system/config/setting"
	menu "pantheon-base/modules/system/iam/menu"
	role "pantheon-base/modules/system/iam/role"
	user "pantheon-base/modules/system/iam/user"
	dept "pantheon-base/modules/system/org/dept"
	post "pantheon-base/modules/system/org/post"
	"pantheon-base/pkg/testmysql"

	"gorm.io/gorm"
)

func migrateDashboardRuntimeSchema(t *testing.T, db *gorm.DB) {
	t.Helper()
	if err := db.AutoMigrate(
		&user.SystemUser{},
		&role.SystemRole{},
		&dept.SystemDept{},
		&post.SystemPost{},
		&dictcfg.SystemDictType{},
		&settingcfg.SystemSetting{},
		&menu.SystemMenu{},
		&session.SystemUserSession{},
		&login.SystemLogLogin{},
		&middleware.SystemLogOper{},
		&security.SystemAuthSecurityEvent{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}
}

func TestDashboardServiceGetSummaryUsesCurrentRuntimeSchema(t *testing.T) {
	db := testmysql.Open(t)
	migrateDashboardRuntimeSchema(t, db)

	now := time.Now()
	if err := db.Create(&user.SystemUser{Username: "admin", Status: 1}).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	if err := db.Create(&role.SystemRole{RoleKey: "admin", Status: 1}).Error; err != nil {
		t.Fatalf("seed role: %v", err)
	}
	if err := db.Create(&dept.SystemDept{DeptName: "Root", IsRoot: 1, Status: 1}).Error; err != nil {
		t.Fatalf("seed dept: %v", err)
	}
	if err := db.Create(&dictcfg.SystemDictType{DictName: "yes/no", DictCode: "system_yes_no", Module: "system", Status: 1}).Error; err != nil {
		t.Fatalf("seed dict type: %v", err)
	}
	if err := db.Create(&settingcfg.SystemSetting{SettingKey: "login.session_idle_minutes", SettingValue: "30", GroupKey: "login", Module: "system"}).Error; err != nil {
		t.Fatalf("seed setting: %v", err)
	}
	if err := db.Create(&middleware.SystemLogOper{
		Title:    "dashboard smoke",
		Status:   1,
		OperTime: now,
	}).Error; err != nil {
		t.Fatalf("seed operation log: %v", err)
	}
	if err := db.Create(&security.SystemAuthSecurityEvent{
		Username:   "admin",
		EventType:  "source_locked",
		Severity:   "high",
		MessageKey: "auth.security_event.list.error",
		CreatedAt:  now,
	}).Error; err != nil {
		t.Fatalf("seed pending security event: %v", err)
	}
	ackAt := now
	if err := db.Create(&security.SystemAuthSecurityEvent{
		Username:       "admin",
		EventType:      "account_locked",
		Severity:       "medium",
		MessageKey:     "auth.security_event.list.error",
		AcknowledgedAt: &ackAt,
		CreatedAt:      now,
	}).Error; err != nil {
		t.Fatalf("seed acknowledged security event: %v", err)
	}

	svc := NewDashboardService(db)
	summary, err := svc.GetSummary()
	if err != nil {
		t.Fatalf("get summary: %v", err)
	}

	if summary.TodayOperationCount != 1 {
		t.Fatalf("expected 1 operation log today, got %d", summary.TodayOperationCount)
	}
	if summary.TotalSecurityEventCount != 2 {
		t.Fatalf("expected 2 security events, got %d", summary.TotalSecurityEventCount)
	}
	if summary.PendingSecurityEventCount != 1 {
		t.Fatalf("expected 1 pending security event, got %d", summary.PendingSecurityEventCount)
	}
}

type staticGovernanceLoader struct {
	tasks []OrgGovernanceTask
}

func (l staticGovernanceLoader) ListOrgGovernanceTasks() ([]OrgGovernanceTask, error) {
	return l.tasks, nil
}

func TestDashboardServiceGetSummaryMapsGovernanceTodoFields(t *testing.T) {
	db := testmysql.Open(t)
	migrateDashboardRuntimeSchema(t, db)
	svc := NewDashboardService(db, WithOrgGovernanceTaskLoader(staticGovernanceLoader{
		tasks: []OrgGovernanceTask{
			{
				TaskKey:          "dept:8:assign-leader",
				GovernanceScope:  "dept",
				GovernanceTag:    "leaderless",
				GovernanceAction: "assign-leader",
				DeptID:           8,
				DeptName:         "研发中心",
			},
			{
				TaskKey:          "post:15:reassign-users",
				GovernanceScope:  "post",
				GovernanceTag:    "in-use",
				GovernanceAction: "reassign-users",
				DeptID:           6,
				DeptName:         "产品部",
				PostName:         "产品经理",
				RelatedUserCount: 3,
			},
		},
	}))

	summary, err := svc.GetSummary()
	if err != nil {
		t.Fatalf("get summary: %v", err)
	}
	if summary.OrgGovernanceTaskCount != 2 {
		t.Fatalf("expected 2 governance tasks, got %d", summary.OrgGovernanceTaskCount)
	}
	if len(summary.OrgGovernanceTasks) != 2 {
		t.Fatalf("expected 2 governance todo items, got %d", len(summary.OrgGovernanceTasks))
	}

	deptTask := summary.OrgGovernanceTasks[0]
	if deptTask.Issue != "leaderless" {
		t.Fatalf("expected dept issue leaderless, got %q", deptTask.Issue)
	}
	if deptTask.Action != "assign-leader" {
		t.Fatalf("expected dept action assign-leader, got %q", deptTask.Action)
	}
	if deptTask.ResourceLabel != "研发中心" {
		t.Fatalf("expected dept resource label 研发中心, got %q", deptTask.ResourceLabel)
	}
	if deptTask.RoutePath != "/system/dept" {
		t.Fatalf("expected dept route path /system/dept, got %q", deptTask.RoutePath)
	}
	if deptTask.RouteStateDeptID != 8 {
		t.Fatalf("expected dept route state dept id 8, got %d", deptTask.RouteStateDeptID)
	}

	postTask := summary.OrgGovernanceTasks[1]
	if postTask.Issue != "in-use" {
		t.Fatalf("expected post issue in-use, got %q", postTask.Issue)
	}
	if postTask.Action != "reassign-users" {
		t.Fatalf("expected post action reassign-users, got %q", postTask.Action)
	}
	if postTask.ResourceLabel != "产品经理 / 产品部" {
		t.Fatalf("expected post resource label 产品经理 / 产品部, got %q", postTask.ResourceLabel)
	}
	if postTask.RelatedUserCount != 3 {
		t.Fatalf("expected post related user count 3, got %d", postTask.RelatedUserCount)
	}
	if postTask.RoutePath != "/system/dept" {
		t.Fatalf("expected post route path /system/dept, got %q", postTask.RoutePath)
	}
	if postTask.RouteStateDeptID != 6 {
		t.Fatalf("expected post route state dept id 6, got %d", postTask.RouteStateDeptID)
	}
}
