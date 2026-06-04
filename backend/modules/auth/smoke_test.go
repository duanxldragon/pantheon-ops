package auth

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	user "pantheon-ops/backend/modules/system/iam/user"
	"pantheon-ops/backend/pkg/common"
	"pantheon-ops/backend/pkg/testmysql"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

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

	// 迁移所有核心表
	_ = db.AutoMigrate(&user.SystemUser{}, &SystemUserSession{}, &SystemLogLogin{}, &SystemLoginThrottle{}, &SystemAuthFactor{}, &SystemAuthMFAChallenge{})
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
	}

	return r, db
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
			body, _ := json.Marshal(LoginReq{
				Username: tt.username,
				Password: tt.password,
			})
			req, _ := http.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != tt.expectedCode {
				t.Errorf("expected http code %d, got %d", tt.expectedCode, w.Code)
			}

			var resp common.Response
			_ = json.Unmarshal(w.Body.Bytes(), &resp)
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
