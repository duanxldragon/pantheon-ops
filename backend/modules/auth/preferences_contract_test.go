package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"pantheon-ops/backend/internal/middleware"
	settingmod "pantheon-ops/backend/modules/system/config/setting"
	user "pantheon-ops/backend/modules/system/iam/user"
	"pantheon-ops/backend/pkg/common"
	"pantheon-ops/backend/pkg/testmysql"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func setupPreferenceContractRouter(t *testing.T) (*gin.Engine, *gorm.DB) {
	t.Helper()
	gin.SetMode(gin.TestMode)

	db := testmysql.Open(t)
	if err := db.AutoMigrate(
		&user.SystemUser{},
		&SystemUserSession{},
		&SystemLogLogin{},
		&SystemLoginThrottle{},
		&middleware.SystemLogOper{},
		&settingmod.SystemSetting{},
	); err != nil {
		t.Fatalf("migrate auth preference contract tables: %v", err)
	}
	if err := db.Exec("CREATE TABLE IF NOT EXISTS system_role (id BIGINT PRIMARY KEY, role_key VARCHAR(64), status INT)").Error; err != nil {
		t.Fatalf("create role table: %v", err)
	}
	if err := db.Exec("CREATE TABLE IF NOT EXISTS system_user_role (user_id BIGINT, role_id BIGINT)").Error; err != nil {
		t.Fatalf("create user role table: %v", err)
	}
	if err := db.Exec("CREATE TABLE IF NOT EXISTS system_role_permission (role_id BIGINT, permission_key VARCHAR(128))").Error; err != nil {
		t.Fatalf("create role permission table: %v", err)
	}

	hash, _ := bcrypt.GenerateFromPassword([]byte("123456"), bcrypt.DefaultCost)
	if err := db.Create(&user.SystemUser{
		ID:             1,
		Username:       "admin",
		Password:       string(hash),
		Status:         1,
		PreferenceJSON: `{"theme":"emerald","layout":"vertical","lang":"zh-CN"}`,
	}).Error; err != nil {
		t.Fatalf("seed admin user: %v", err)
	}
	if err := db.Exec("INSERT INTO system_role (id, role_key, status) VALUES (1, 'admin', 1)").Error; err != nil {
		t.Fatalf("seed admin role: %v", err)
	}
	if err := db.Exec("INSERT INTO system_user_role (user_id, role_id) VALUES (1, 1)").Error; err != nil {
		t.Fatalf("seed admin binding: %v", err)
	}

	service := NewAuthService(db)
	handler := NewAuthHandler(service)

	engine := gin.New()
	engine.Use(middleware.OperationLogMiddleware(db))
	engine.Use(func(c *gin.Context) {
		c.Set("userId", uint64(1))
		c.Set("username", "admin")
		c.Set("sessionId", "session-contract")
		c.Next()
	})
	engine.GET("/api/v1/auth/me", handler.GetCurrentUserInfo)
	engine.PUT("/api/v1/auth/me/preferences", handler.UpdateCurrentUserPreferences)
	return engine, db
}

func waitPreferenceAuditLog(t *testing.T, db *gorm.DB) middleware.SystemLogOper {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		var log middleware.SystemLogOper
		err := db.Order("id desc").First(&log).Error
		if err == nil {
			return log
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("preference audit log not written in time")
	return middleware.SystemLogOper{}
}

func TestAuthPreferencesContract_GetMeIncludesNormalizedPreferences(t *testing.T) {
	engine, _ := setupPreferenceContractRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/auth/me", nil)
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status code: %d", recorder.Code)
	}

	var resp struct {
		Code int          `json:"code"`
		Data UserInfoResp `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Code != common.CodeSuccess {
		t.Fatalf("unexpected business code: %d", resp.Code)
	}
	if resp.Data.Preferences == nil {
		t.Fatalf("expected preferences in response")
	}
	if resp.Data.Preferences.Theme != "emerald" || resp.Data.Preferences.LayoutMode != "vertical" || resp.Data.Preferences.Language != "zh-CN" {
		t.Fatalf("unexpected normalized preferences: %+v", resp.Data.Preferences)
	}
}

func TestAuthPreferencesContract_UpdatePersistsAndAuditsPreferenceChange(t *testing.T) {
	engine, db := setupPreferenceContractRouter(t)

	body := []byte(`{"theme":"slate","language":"en-US","layoutMode":"horizontal","densityMode":"compact"}`)
	req := httptest.NewRequest(http.MethodPut, "/api/v1/auth/me/preferences", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("unexpected status code: %d", recorder.Code)
	}

	resp := decodePreferenceUserInfoResp(t, recorder.Body.Bytes())
	assertUpdatedPreferenceResponse(t, resp)
	assertPersistedPreferenceJSON(t, db)
	assertPreferenceAuditLogUpdated(t, waitPreferenceAuditLog(t, db))
}

func decodePreferenceUserInfoResp(t *testing.T, body []byte) struct {
	Code int          `json:"code"`
	Data UserInfoResp `json:"data"`
} {
	t.Helper()

	var resp struct {
		Code int          `json:"code"`
		Data UserInfoResp `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Code != common.CodeSuccess {
		t.Fatalf("unexpected business code: %d", resp.Code)
	}
	return resp
}

func assertUpdatedPreferenceResponse(t *testing.T, resp struct {
	Code int          `json:"code"`
	Data UserInfoResp `json:"data"`
}) {
	t.Helper()

	if resp.Data.Preferences == nil || resp.Data.Preferences.Theme != "slate" || resp.Data.Preferences.Language != "en-US" || resp.Data.Preferences.LayoutMode != "horizontal" || resp.Data.Preferences.DensityMode != "compact" {
		t.Fatalf("unexpected response preferences: %+v", resp.Data.Preferences)
	}
}

func assertPersistedPreferenceJSON(t *testing.T, db *gorm.DB) {
	t.Helper()

	var updated user.SystemUser
	if err := db.First(&updated, uint64(1)).Error; err != nil {
		t.Fatalf("reload updated user: %v", err)
	}
	if updated.PreferenceJSON != `{"theme":"slate","language":"en-US","layoutMode":"horizontal","densityMode":"compact"}` {
		t.Fatalf("unexpected persisted preference json: %s", updated.PreferenceJSON)
	}
}

func assertPreferenceAuditLogUpdated(t *testing.T, log middleware.SystemLogOper) {
	t.Helper()

	if log.Title != "更新平台偏好" {
		t.Fatalf("unexpected audit title: %s", log.Title)
	}
	if log.BusinessType != common.BusinessUpdate {
		t.Fatalf("unexpected audit business type: %d", log.BusinessType)
	}

	var auditParam map[string]any
	if err := json.Unmarshal([]byte(log.OperParam), &auditParam); err != nil {
		t.Fatalf("unmarshal audit param: %v", err)
	}
	before, _ := auditParam["before"].(map[string]any)
	after, _ := auditParam["after"].(map[string]any)
	if before["theme"] != "emerald" || before["layoutMode"] != "vertical" || before["language"] != "zh-CN" {
		t.Fatalf("unexpected audit before payload: %#v", before)
	}
	if after["theme"] != "slate" || after["layoutMode"] != "horizontal" || after["language"] != "en-US" || after["densityMode"] != "compact" {
		t.Fatalf("unexpected audit after payload: %#v", after)
	}

	var auditResult map[string]any
	if err := json.Unmarshal([]byte(log.JsonResult), &auditResult); err != nil {
		t.Fatalf("unmarshal audit result: %v", err)
	}
	if changed, _ := auditResult["changed"].(bool); !changed {
		t.Fatalf("expected changed=true in audit result, got %#v", auditResult["changed"])
	}
}
