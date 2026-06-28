package platform

import (
	"testing"
	"time"

	"pantheon-ops/backend/internal/middleware"
	security "pantheon-ops/backend/modules/auth/security"
	dictcfg "pantheon-ops/backend/modules/system/config/dict"
	settingcfg "pantheon-ops/backend/modules/system/config/setting"
	role "pantheon-ops/backend/modules/system/iam/role"
	user "pantheon-ops/backend/modules/system/iam/user"
	dept "pantheon-ops/backend/modules/system/org/dept"
	post "pantheon-ops/backend/modules/system/org/post"
	"pantheon-ops/backend/pkg/testmysql"
)

func TestDashboardServiceGetSummaryUsesCurrentRuntimeSchema(t *testing.T) {
	db := testmysql.Open(t)

	if err := db.AutoMigrate(
		&user.SystemUser{},
		&role.SystemRole{},
		&dept.SystemDept{},
		&post.SystemPost{},
		&dictcfg.SystemDictType{},
		&settingcfg.SystemSetting{},
		&middleware.SystemLogOper{},
		&security.SystemAuthSecurityEvent{},
	); err != nil {
		t.Fatalf("auto migrate: %v", err)
	}

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

type noopGovernanceLoader struct{}

func (noopGovernanceLoader) ListOrgGovernanceTasks() ([]OrgGovernanceTask, error) {
	return nil, nil
}
