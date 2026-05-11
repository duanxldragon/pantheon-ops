package dashboard

import (
	"testing"
	"time"

	"pantheon-ops/backend/internal/middleware"
	auth "pantheon-ops/backend/modules/auth"
	dict "pantheon-ops/backend/modules/system/config/dict"
	setting "pantheon-ops/backend/modules/system/config/setting"
	menu "pantheon-ops/backend/modules/system/iam/menu"
	role "pantheon-ops/backend/modules/system/iam/role"
	user "pantheon-ops/backend/modules/system/iam/user"
	dept "pantheon-ops/backend/modules/system/org/dept"
	post "pantheon-ops/backend/modules/system/org/post"
	"pantheon-ops/backend/pkg/testmysql"

	"gorm.io/gorm"
)

func setupDashboardTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db := testmysql.Open(t)

	if err := db.AutoMigrate(
		&user.SystemUser{},
		&role.SystemRole{},
		&dept.SystemDept{},
		&post.SystemPost{},
		&dict.SystemDictType{},
		&setting.SystemSetting{},
		&menu.SystemMenu{},
		&auth.SystemUserSession{},
		&auth.SystemLogLogin{},
		&middleware.SystemLogOper{},
	); err != nil {
		t.Fatalf("migrate dashboard fixtures: %v", err)
	}

	return db
}

func TestDashboardService_GetSummary(t *testing.T) {
	db := setupDashboardTestDB(t)
	service := NewDashboardService(db)

	now := time.Now()
	successLoginTime := now.Add(-2 * time.Hour)

	if err := db.Create(&user.SystemUser{Username: "admin", Status: 1}).Error; err != nil {
		t.Fatalf("seed user1: %v", err)
	}
	if err := db.Create(&user.SystemUser{Username: "disabled", Status: 2}).Error; err != nil {
		t.Fatalf("seed user2: %v", err)
	}
	if err := db.Create(&role.SystemRole{RoleName: "Admin", RoleKey: "admin", Status: 1}).Error; err != nil {
		t.Fatalf("seed role: %v", err)
	}
	if err := db.Create(&dept.SystemDept{DeptName: "研发中心", Status: 1}).Error; err != nil {
		t.Fatalf("seed dept: %v", err)
	}
	if err := db.Create(&post.SystemPost{PostCode: "cto", PostName: "CTO", Status: 1}).Error; err != nil {
		t.Fatalf("seed post: %v", err)
	}
	if err := db.Create(&dict.SystemDictType{DictCode: "system_status", DictName: "system.dict.seed.status", Module: "system", Status: 1}).Error; err != nil {
		t.Fatalf("seed dict type: %v", err)
	}
	if err := db.Create(&setting.SystemSetting{SettingKey: "site.name", SettingValue: "Pantheon", ValueType: "string", GroupKey: "basic", Module: "system"}).Error; err != nil {
		t.Fatalf("seed setting: %v", err)
	}
	if err := db.Create(&setting.SystemSetting{SettingKey: "login.session_idle_minutes", SettingValue: "30", ValueType: "number", GroupKey: "login", Module: "system"}).Error; err != nil {
		t.Fatalf("seed idle setting: %v", err)
	}
	if err := db.Create(&menu.SystemMenu{TitleKey: "system.menu.dashboard", Path: "/dashboard", Type: "C", Module: "platform", IsVisible: 1}).Error; err != nil {
		t.Fatalf("seed menu: %v", err)
	}
	if err := db.Create(&auth.SystemUserSession{
		SessionID:        "session-1",
		UserID:           1,
		RefreshJTI:       "jti-1",
		RefreshExpiresAt: now.Add(24 * time.Hour),
		LastRefreshAt:    &now,
		LastIP:           "127.0.0.1",
		UserAgent:        "test",
		CreatedAt:        now.Add(-3 * time.Hour),
		UpdatedAt:        now,
	}).Error; err != nil {
		t.Fatalf("seed session: %v", err)
	}
	if err := db.Create(&auth.SystemUserSession{
		SessionID:        "session-expired",
		UserID:           1,
		RefreshJTI:       "jti-expired",
		RefreshExpiresAt: now.Add(-5 * time.Minute),
		LastRefreshAt:    timePtr(now.Add(-10 * time.Minute)),
		LastActivityAt:   timePtr(now.Add(-5 * time.Minute)),
		LastIP:           "127.0.0.2",
		UserAgent:        "expired",
		CreatedAt:        now.Add(-4 * time.Hour),
		UpdatedAt:        now,
	}).Error; err != nil {
		t.Fatalf("seed expired session: %v", err)
	}
	if err := db.Create(&auth.SystemUserSession{
		SessionID:        "session-idle",
		UserID:           1,
		RefreshJTI:       "jti-idle",
		RefreshExpiresAt: now.Add(24 * time.Hour),
		LastRefreshAt:    timePtr(now.Add(-90 * time.Minute)),
		LastActivityAt:   timePtr(now.Add(-45 * time.Minute)),
		LastIP:           "127.0.0.3",
		UserAgent:        "idle",
		CreatedAt:        now.Add(-5 * time.Hour),
		UpdatedAt:        now,
	}).Error; err != nil {
		t.Fatalf("seed idle session: %v", err)
	}
	if err := db.Create(&auth.SystemLogLogin{
		Username:  "admin",
		Ipaddr:    "127.0.0.1",
		Browser:   "Chrome",
		Os:        "Windows",
		Status:    1,
		Msg:       "auth.loginSuccess",
		LoginTime: successLoginTime,
	}).Error; err != nil {
		t.Fatalf("seed success login: %v", err)
	}
	if err := db.Model(&auth.SystemLogLogin{}).Create(map[string]any{
		"username":   "disabled",
		"ipaddr":     "127.0.0.2",
		"browser":    "Chrome",
		"os":         "Windows",
		"status":     0,
		"msg":        "user.login.error.disabled",
		"login_time": now.Add(-1 * time.Hour),
	}).Error; err != nil {
		t.Fatalf("seed failure login: %v", err)
	}
	if err := db.Create(&middleware.SystemLogOper{
		Title:    "system.user.create",
		OperName: "admin",
		OperURL:  "/api/v1/system/user",
		Status:   1,
		OperTime: now.Add(-30 * time.Minute),
	}).Error; err != nil {
		t.Fatalf("seed operation log: %v", err)
	}

	summary, err := service.GetSummary()
	if err != nil {
		t.Fatalf("get summary: %v", err)
	}

	if summary.TotalUsers != 2 {
		t.Fatalf("expected 2 users, got %d", summary.TotalUsers)
	}
	if summary.EnabledUsers != 1 {
		t.Fatalf("expected 1 enabled user, got %d", summary.EnabledUsers)
	}
	if summary.TotalRoles != 1 || summary.TotalDepts != 1 || summary.TotalPosts != 1 {
		t.Fatalf("expected role/dept/post counts to be 1, got roles=%d depts=%d posts=%d", summary.TotalRoles, summary.TotalDepts, summary.TotalPosts)
	}
	if summary.TotalDictTypes != 1 || summary.TotalSettings != 1 {
		t.Fatalf("expected dict/settings counts to be 1, got dictTypes=%d settings=%d", summary.TotalDictTypes, summary.TotalSettings)
	}
	if summary.VisibleMenuCount != 1 {
		t.Fatalf("expected 1 visible menu, got %d", summary.VisibleMenuCount)
	}
	if summary.ActiveSessionCount != 1 {
		t.Fatalf("expected 1 active session, got %d", summary.ActiveSessionCount)
	}
	if summary.LoginSuccessCount != 1 || summary.LoginFailureCount != 1 {
		t.Fatalf("expected login success/failure counts to be 1, got success=%d failure=%d", summary.LoginSuccessCount, summary.LoginFailureCount)
	}
	if summary.TodayOperationCount != 1 {
		t.Fatalf("expected 1 operation today, got %d", summary.TodayOperationCount)
	}
	if summary.LastSuccessfulLoginAt == "" {
		t.Fatal("expected last successful login timestamp")
	}
	if len(summary.RecentLogins) != 2 {
		t.Fatalf("expected 2 recent logins, got %d", len(summary.RecentLogins))
	}
}

func timePtr(value time.Time) *time.Time {
	return &value
}
