package login

import (
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/modules/system/config/setting"
	"github.com/duanxldragon/pantheon-base/backend/modules/system/iam/user"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/tenant"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func seedTenantLoginFixture(t *testing.T, db *gorm.DB, tenantID uint64, username, password string) user.SystemUser {
	t.Helper()
	if err := db.AutoMigrate(&tenant.Membership{}); err != nil {
		t.Fatalf("migrate memberships: %v", err)
	}
	if err := db.Create(&tenant.Tenant{
		ID:     tenantID,
		Code:   fmt.Sprintf("tenant-%d", tenantID),
		Name:   fmt.Sprintf("Tenant %d", tenantID),
		Status: tenant.TenantStatusActive,
	}).Error; err != nil {
		t.Fatalf("seed tenant %d: %v", tenantID, err)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	account := user.SystemUser{
		TenantID: tenantID,
		Username: username,
		Password: string(hash),
		Status:   common.StatusEnabled,
	}
	if err := db.Create(&account).Error; err != nil {
		t.Fatalf("seed tenant user: %v", err)
	}
	if err := db.Create(&tenant.Membership{
		TenantID: tenantID,
		UserID:   account.ID,
		Role:     tenant.MembershipRoleOwner,
		Status:   tenant.MembershipActive,
	}).Error; err != nil {
		t.Fatalf("seed membership: %v", err)
	}
	return account
}

func seedTenantMode(t *testing.T, db *gorm.DB, mode string) {
	t.Helper()
	if err := db.Create(&setting.SystemSetting{
		SettingKey:   tenant.FeatureFlagSettingKey,
		SettingValue: mode,
		ValueType:    "string",
		GroupKey:     "platform",
		Module:       "platform",
	}).Error; err != nil {
		t.Fatalf("seed tenant mode: %v", err)
	}
}

func postLoginRequest(t *testing.T, handler *AuthHandler, body string) (int, map[string]any) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	c, recorder := newHandlerTestContext(t)
	c.Request = httptest.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(body))
	c.Request.Header.Set("Content-Type", "application/json")
	handler.LoginHandler(c)
	return decodeHandlerResponse(t, recorder)
}

func TestLoginHandlerAuditsImplicitTenantLoginInTenant(t *testing.T) {
	db := setupHandlerTestDB(t)
	setupTestRedis(t)
	account := seedTenantLoginFixture(t, db, 101, "tenant_success_user", "tenant-pass")
	seedTenantMode(t, db, tenant.ModeMulti)
	// W4 (W3 decision A): a membership without a role binding is not a usable
	// context — the issuance gate refuses it. Seed the binding so the test
	// isolates the implicit-login audit path.
	if err := db.Create(&user.SystemUserRole{TenantID: 101, UserID: account.ID, RoleID: 1}).Error; err != nil {
		t.Fatalf("seed role binding: %v", err)
	}

	handler := NewAuthHandler(NewRuntime(db, testCredentialRepo(db)))
	code, _ := postLoginRequest(t, handler, `{"username":"tenant_success_user","password":"tenant-pass"}`)
	if code != common.CodeSuccess {
		t.Fatalf("expected implicit tenant login to succeed, got code=%d", code)
	}

	var logRow SystemLogLogin
	if err := db.Where("username = ?", account.Username).First(&logRow).Error; err != nil {
		t.Fatalf("load login log: %v", err)
	}
	if logRow.TenantID != 101 || logRow.Status != common.LoginStatusSuccess {
		t.Fatalf("expected success log in tenant 101, got tenant=%d status=%d", logRow.TenantID, logRow.Status)
	}

	var sess SystemUserSession
	if err := db.Where("user_id = ?", account.ID).First(&sess).Error; err != nil {
		t.Fatalf("load session: %v", err)
	}
	if sess.TenantID != 101 {
		t.Fatalf("expected session claim 101, got %d", sess.TenantID)
	}

	page, err := NewRuntime(db, testCredentialRepo(db)).
		WithTenantContext(&tenant.Context{TenantID: 101, Mode: tenant.ModeMulti}).
		ListLoginLogs(&LoginLogQuery{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("list tenant login logs: %v", err)
	}
	if page.Total != 1 || page.Items[0].Username != account.Username {
		t.Fatalf("expected the login in the tenant audit view, got %+v", page)
	}
}

func TestLoginHandlerAuditsFailedImplicitLoginInOwningTenant(t *testing.T) {
	db := setupHandlerTestDB(t)
	account := seedTenantLoginFixture(t, db, 202, "tenant_fail_user", "tenant-pass")
	seedTenantMode(t, db, tenant.ModeMulti)
	seedSettings(t, db, map[string]string{"login.security_event_enabled": "true"})

	handler := NewAuthHandler(NewRuntime(db, testCredentialRepo(db)))
	code, _ := postLoginRequest(t, handler, `{"username":"tenant_fail_user","password":"wrong-pass"}`)
	if code != common.CodeUnauthorized {
		t.Fatalf("expected unauthorized, got code=%d", code)
	}

	var logRow SystemLogLogin
	if err := db.Where("username = ?", account.Username).First(&logRow).Error; err != nil {
		t.Fatalf("load login log: %v", err)
	}
	if logRow.TenantID != 202 || logRow.Status != common.LoginStatusFailure {
		t.Fatalf("expected failure log in tenant 202, got tenant=%d status=%d", logRow.TenantID, logRow.Status)
	}

	var event SystemAuthSecurityEvent
	if err := db.Where("username = ? AND event_type = ?", account.Username, "password_wrong").First(&event).Error; err != nil {
		t.Fatalf("load security event: %v", err)
	}
	if event.TenantID != 202 {
		t.Fatalf("expected security event in tenant 202, got %d", event.TenantID)
	}

	scoped := NewRuntime(db, testCredentialRepo(db)).
		WithTenantContext(&tenant.Context{TenantID: 202, Mode: tenant.ModeMulti})
	page, err := scoped.ListLoginLogs(&LoginLogQuery{Page: 1, PageSize: 10})
	if err != nil || page.Total != 1 {
		t.Fatalf("expected failure log in tenant audit view, page=%+v err=%v", page, err)
	}
	events, err := scoped.ListSecurityEvents(&SecurityEventQuery{Username: account.Username, Page: 1, PageSize: 10})
	if err != nil || events.Total != 1 {
		t.Fatalf("expected security event in tenant audit view, events=%+v err=%v", events, err)
	}
}

func TestLoginHandlerCompatKeepsAuditAtPlatformScope(t *testing.T) {
	db := setupHandlerTestDB(t)
	account := seedTenantLoginFixture(t, db, 303, "compat_fail_user", "tenant-pass")

	handler := NewAuthHandler(NewRuntime(db, testCredentialRepo(db)))
	code, _ := postLoginRequest(t, handler, `{"username":"compat_fail_user","password":"wrong-pass"}`)
	if code != common.CodeUnauthorized {
		t.Fatalf("expected unauthorized, got code=%d", code)
	}

	var logRow SystemLogLogin
	if err := db.Where("username = ?", account.Username).First(&logRow).Error; err != nil {
		t.Fatalf("load login log: %v", err)
	}
	if logRow.TenantID != tenant.PlatformGlobalTenantID {
		t.Fatalf("compat login log must stay platform-global, got tenant=%d", logRow.TenantID)
	}

	page, err := NewRuntime(db, testCredentialRepo(db)).
		WithTenantContext(&tenant.Context{TenantID: tenant.PlatformGlobalTenantID, Mode: tenant.ModeCompat}).
		ListLoginLogs(&LoginLogQuery{Page: 1, PageSize: 10})
	if err != nil || page.Total != 1 {
		t.Fatalf("expected the compat log in the platform audit view, page=%+v err=%v", page, err)
	}
}
