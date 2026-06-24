package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	settingmod "pantheon-ops/backend/modules/system/config/setting"
	user "pantheon-ops/backend/modules/system/iam/user"
	"pantheon-ops/backend/pkg/common"
	"pantheon-ops/backend/pkg/testmysql"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type authResponseEnvelope[T any] struct {
	Code int `json:"code"`
	Data T   `json:"data"`
}

func setupSmokeTestRouter(t *testing.T) (*gin.Engine, *gorm.DB) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	db := testmysql.Open(t)

	// 清理旧表
	_ = db.Exec("DROP TABLE IF EXISTS system_user")
	_ = db.Exec("DROP TABLE IF EXISTS system_user_session")
	_ = db.Exec("DROP TABLE IF EXISTS system_log_login")
	_ = db.Exec("DROP TABLE IF EXISTS system_login_throttle")
	_ = db.Exec("DROP TABLE IF EXISTS system_auth_factor")
	_ = db.Exec("DROP TABLE IF EXISTS system_auth_mfa_challenge")
	_ = db.Exec("DROP TABLE IF EXISTS system_role")
	_ = db.Exec("DROP TABLE IF EXISTS system_user_role")
	_ = db.Exec("DROP TABLE IF EXISTS system_role_permission")
	_ = db.Exec("DROP TABLE IF EXISTS system_setting")

	// 迁移所有核心表
	_ = db.AutoMigrate(&user.SystemUser{}, &SystemUserSession{}, &SystemLogLogin{}, &SystemLoginThrottle{}, &SystemAuthFactor{}, &SystemAuthMFAChallenge{}, &settingmod.SystemSetting{})
	_ = db.Exec("CREATE TABLE IF NOT EXISTS system_role (id BIGINT PRIMARY KEY, role_key VARCHAR(64), status INT)")
	_ = db.Exec("CREATE TABLE IF NOT EXISTS system_user_role (user_id BIGINT, role_id BIGINT)")
	_ = db.Exec("CREATE TABLE IF NOT EXISTS system_role_permission (role_id BIGINT, permission_key VARCHAR(128))")

	// 创建初始管理员
	hash, _ := bcrypt.GenerateFromPassword([]byte("123456"), bcrypt.DefaultCost)
	db.Create(&user.SystemUser{
		ID:       1,
		Username: "admin",
		Password: string(hash),
		Status:   1,
	})
	_ = db.Exec("INSERT INTO system_role (id, role_key, status) VALUES (1, 'admin', 1)")
	_ = db.Exec("INSERT INTO system_user_role (user_id, role_id) VALUES (1, 1)")
	_ = db.Exec("INSERT INTO system_role_permission (role_id, permission_key) VALUES (1, 'sys:dashboard:view')")

	authSvc := NewAuthService(db)
	handler := NewAuthHandler(authSvc)

	r := gin.Default()
	v1 := r.Group("/api/v1")
	{
		v1.POST("/auth/login", handler.LoginHandler)
		v1.POST("/auth/refresh", handler.RefreshTokenHandler)
	}

	return r, db
}

func performJSONRequest(t *testing.T, router *gin.Engine, method, url string, payload any) *httptest.ResponseRecorder {
	t.Helper()

	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal %s %s payload: %v", method, url, err)
	}

	req, err := http.NewRequest(method, url, bytes.NewBuffer(body))
	if err != nil {
		t.Fatalf("build %s %s request: %v", method, url, err)
	}
	req.Header.Set("Content-Type", "application/json")

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, req)
	return recorder
}

func decodeAuthResponse[T any](t *testing.T, recorder *httptest.ResponseRecorder) authResponseEnvelope[T] {
	t.Helper()

	var resp authResponseEnvelope[T]
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	return resp
}

func assertCSRFCookieAndHeader(t *testing.T, recorder *httptest.ResponseRecorder, cookieMessage, headerMessage string) {
	t.Helper()

	var csrfCookieFound bool
	for _, cookie := range recorder.Result().Cookies() {
		if cookie.Name != common.CookieCSRFToken {
			continue
		}
		csrfCookieFound = true
		if !cookie.HttpOnly {
			t.Fatal(cookieMessage)
		}
		headerValue := recorder.Header().Get("X-CSRF-Token")
		if headerValue == "" {
			t.Fatal(headerMessage)
		}
		if headerValue != cookie.Value {
			t.Fatalf("expected csrf header and cookie to match, got header=%q cookie=%q", headerValue, cookie.Value)
		}
	}
	if !csrfCookieFound {
		t.Fatal("expected response to set csrf cookie")
	}
}

func TestSmoke_LoginFlow(t *testing.T) {
	r, db := setupSmokeTestRouter(t)

	tests := []struct {
		name            string
		username        string
		password        string
		expectedCode    int
		expectedBizCode int
	}{
		{
			name:            "Valid Admin Login",
			username:        "admin",
			password:        "123456",
			expectedCode:    http.StatusOK,
			expectedBizCode: 200,
		},
		{
			name:            "Wrong Password",
			username:        "admin",
			password:        "wrongpass",
			expectedCode:    http.StatusOK,
			expectedBizCode: 401,
		},
		{
			name:            "Non-existent User",
			username:        "nobody",
			password:        "123456",
			expectedCode:    http.StatusOK,
			expectedBizCode: 401,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := performJSONRequest(t, r, http.MethodPost, "/api/v1/auth/login", LoginReq{
				Username: tt.username,
				Password: tt.password,
			})

			if recorder.Code != tt.expectedCode {
				t.Errorf("expected http code %d, got %d", tt.expectedCode, recorder.Code)
			}

			var resp common.Response
			_ = json.Unmarshal(recorder.Body.Bytes(), &resp)
			if resp.Code != tt.expectedBizCode {
				t.Errorf("expected biz code %d, got %d. Msg: %s", tt.expectedBizCode, resp.Code, resp.Message)
			}
		})
	}

	var loginLogCount int64
	if err := db.Model(&SystemLogLogin{}).Count(&loginLogCount).Error; err != nil {
		t.Fatalf("count login logs: %v", err)
	}
	if loginLogCount != int64(len(tests)) {
		t.Fatalf("expected %d login logs, got %d", len(tests), loginLogCount)
	}
}

func TestSmoke_LoginFlowSetsHttpOnlyCSRFCookieAndHeader(t *testing.T) {
	r, _ := setupSmokeTestRouter(t)

	recorder := performJSONRequest(t, r, http.MethodPost, "/api/v1/auth/login", LoginReq{
		Username: "admin",
		Password: "123456",
	})

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}

	resp := decodeAuthResponse[map[string]any](t, recorder)
	if resp.Code != common.CodeSuccess {
		t.Fatalf("expected business success code, got %d", resp.Code)
	}
	if _, ok := resp.Data["token"]; ok {
		t.Fatalf("expected login response to omit token body, got %#v", resp.Data)
	}
	if _, ok := resp.Data["accessToken"]; ok {
		t.Fatalf("expected login response to omit access token body, got %#v", resp.Data)
	}
	if _, ok := resp.Data["refreshToken"]; ok {
		t.Fatalf("expected login response to omit refresh token body, got %#v", resp.Data)
	}

	assertCSRFCookieAndHeader(
		t,
		recorder,
		"expected csrf cookie to be httpOnly after login",
		"expected login response to expose csrf token header",
	)
}

func TestSmoke_RefreshFlowReissuesCSRFCookieAndHeader(t *testing.T) {
	r, _ := setupSmokeTestRouter(t)

	loginRecorder := performJSONRequest(t, r, http.MethodPost, "/api/v1/auth/login", LoginReq{
		Username: "admin",
		Password: "123456",
	})

	if loginRecorder.Code != http.StatusOK {
		t.Fatalf("expected login status 200, got %d", loginRecorder.Code)
	}

	loginResp := decodeAuthResponse[map[string]any](t, loginRecorder)
	if loginResp.Code != common.CodeSuccess {
		t.Fatalf("expected login business success code, got %d", loginResp.Code)
	}
	if _, ok := loginResp.Data["refreshToken"]; ok {
		t.Fatalf("expected login response to omit refresh token body, got %#v", loginResp.Data)
	}

	loginResult := loginRecorder.Result()
	var refreshCookie *http.Cookie
	for _, cookie := range loginResult.Cookies() {
		if cookie.Name == common.CookieRefreshToken {
			refreshCookie = cookie
			break
		}
	}
	if refreshCookie == nil || strings.TrimSpace(refreshCookie.Value) == "" {
		t.Fatal("expected login response to set refresh token cookie")
	}

	refreshReq, err := http.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewBufferString(`{}`))
	if err != nil {
		t.Fatalf("build refresh request: %v", err)
	}
	refreshReq.Header.Set("Content-Type", "application/json")
	refreshReq.AddCookie(refreshCookie)
	refreshRecorder := httptest.NewRecorder()
	r.ServeHTTP(refreshRecorder, refreshReq)

	if refreshRecorder.Code != http.StatusOK {
		t.Fatalf("expected refresh status 200, got %d", refreshRecorder.Code)
	}

	refreshResp := decodeAuthResponse[map[string]any](t, refreshRecorder)
	if refreshResp.Code != common.CodeSuccess {
		t.Fatalf("expected refresh business success code, got %d", refreshResp.Code)
	}
	if _, ok := refreshResp.Data["token"]; ok {
		t.Fatalf("expected refresh response to omit token body, got %#v", refreshResp.Data)
	}
	if _, ok := refreshResp.Data["accessToken"]; ok {
		t.Fatalf("expected refresh response to omit access token body, got %#v", refreshResp.Data)
	}
	if _, ok := refreshResp.Data["refreshToken"]; ok {
		t.Fatalf("expected refresh response to omit refresh token body, got %#v", refreshResp.Data)
	}

	assertCSRFCookieAndHeader(
		t,
		refreshRecorder,
		"expected refreshed csrf cookie to be httpOnly",
		"expected refresh response to expose csrf token header",
	)
}
