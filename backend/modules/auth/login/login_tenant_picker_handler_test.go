package login

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	settingmod "github.com/duanxldragon/pantheon-base/backend/modules/system/config/setting"
	user "github.com/duanxldragon/pantheon-base/backend/modules/system/iam/user"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/tenant"
	"github.com/duanxldragon/pantheon-base/backend/pkg/testmysql"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

func TestAuthHandler_LoginReturnsTenantPickerAfterPasswordVerification(t *testing.T) {
	db := testmysql.Open(t)
	if err := db.AutoMigrate(
		&user.SystemUser{},
		&SystemUserSession{},
		&SystemLogLogin{},
		&SystemLoginThrottle{},
		&settingmod.SystemSetting{},
		&tenant.Tenant{},
		&tenant.Membership{},
	); err != nil {
		t.Fatalf("migrate picker fixture: %v", err)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte("picker-pass"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	account := user.SystemUser{Username: "picker_user", Password: string(hash), Status: common.StatusEnabled}
	if err := db.Create(&account).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	for _, row := range []tenant.Tenant{
		{ID: 101, Code: "alpha", Name: "Alpha", Status: tenant.TenantStatusActive},
		{ID: 202, Code: "beta", Name: "Beta", Status: tenant.TenantStatusActive},
	} {
		if err := db.Create(&row).Error; err != nil {
			t.Fatalf("create tenant %d: %v", row.ID, err)
		}
	}
	for _, row := range []tenant.Membership{
		{TenantID: 101, UserID: account.ID, Role: tenant.MembershipRoleOwner, Status: tenant.MembershipActive},
		{TenantID: 202, UserID: account.ID, Role: tenant.MembershipRoleMember, Status: tenant.MembershipActive},
	} {
		if err := db.Create(&row).Error; err != nil {
			t.Fatalf("create membership: %v", err)
		}
	}
	if err := db.Create(&settingmod.SystemSetting{
		SettingKey:   tenant.FeatureFlagSettingKey,
		SettingValue: tenant.ModeMulti,
		ValueType:    "string",
		GroupKey:     "platform",
		Module:       "platform",
	}).Error; err != nil {
		t.Fatalf("create tenant mode setting: %v", err)
	}

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(
		`{"username":"picker_user","password":"picker-pass"}`,
	))
	c.Request.Header.Set("Content-Type", "application/json")

	NewAuthHandler(NewRuntime(db, testCredentialRepo(db))).LoginHandler(c)

	var envelope struct {
		Code int `json:"code"`
		Data struct {
			TenantSelectionRequired bool                          `json:"tenantSelectionRequired"`
			TenantCandidates        []tenant.LoginTenantCandidate `json:"tenantCandidates"`
		} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode picker response: %v", err)
	}
	if envelope.Code != common.CodeSuccess || !envelope.Data.TenantSelectionRequired {
		t.Fatalf("expected picker success response, got %s", recorder.Body.String())
	}
	if len(envelope.Data.TenantCandidates) != 2 ||
		envelope.Data.TenantCandidates[0].TenantID != 101 ||
		envelope.Data.TenantCandidates[1].TenantID != 202 {
		t.Fatalf("unexpected tenant candidates: %+v", envelope.Data.TenantCandidates)
	}
	var sessions int64
	if err := db.Model(&SystemUserSession{}).Where("user_id = ?", account.ID).Count(&sessions).Error; err != nil {
		t.Fatalf("count sessions: %v", err)
	}
	if sessions != 0 {
		t.Fatalf("picker response must not create a session, got %d", sessions)
	}
}

func TestAuthHandler_LoginDoesNotExposeTenantPickerOnPasswordFailure(t *testing.T) {
	db := testmysql.Open(t)
	if err := db.AutoMigrate(
		&user.SystemUser{},
		&SystemLogLogin{},
		&SystemLoginThrottle{},
		&settingmod.SystemSetting{},
		&tenant.Tenant{},
		&tenant.Membership{},
	); err != nil {
		t.Fatalf("migrate picker failure fixture: %v", err)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte("picker-pass"), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	account := user.SystemUser{Username: "picker_failure_user", Password: string(hash), Status: common.StatusEnabled}
	if err := db.Create(&account).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	for _, row := range []tenant.Tenant{
		{ID: 301, Code: "gamma", Name: "Gamma", Status: tenant.TenantStatusActive},
		{ID: 302, Code: "delta", Name: "Delta", Status: tenant.TenantStatusActive},
	} {
		if err := db.Create(&row).Error; err != nil {
			t.Fatalf("create tenant %d: %v", row.ID, err)
		}
	}
	for _, row := range []tenant.Membership{
		{TenantID: 301, UserID: account.ID, Role: tenant.MembershipRoleMember, Status: tenant.MembershipActive},
		{TenantID: 302, UserID: account.ID, Role: tenant.MembershipRoleMember, Status: tenant.MembershipActive},
	} {
		if err := db.Create(&row).Error; err != nil {
			t.Fatalf("create membership: %v", err)
		}
	}
	if err := db.Create(&settingmod.SystemSetting{
		SettingKey:   tenant.FeatureFlagSettingKey,
		SettingValue: tenant.ModeMulti,
		ValueType:    "string",
		GroupKey:     "platform",
		Module:       "platform",
	}).Error; err != nil {
		t.Fatalf("create tenant mode setting: %v", err)
	}

	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(
		`{"username":"picker_failure_user","password":"wrong-pass"}`,
	))
	c.Request.Header.Set("Content-Type", "application/json")

	NewAuthHandler(NewRuntime(db, testCredentialRepo(db))).LoginHandler(c)

	var envelope struct {
		Code int                    `json:"code"`
		Data map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode failure response: %v", err)
	}
	if envelope.Code != common.CodeUnauthorized {
		t.Fatalf("expected unauthorized response, got %s", recorder.Body.String())
	}
	if _, ok := envelope.Data["tenantCandidates"]; ok {
		t.Fatalf("password failure must not expose tenant candidates: %s", recorder.Body.String())
	}
}
