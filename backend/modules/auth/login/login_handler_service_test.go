package login

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	settingmod "github.com/duanxldragon/pantheon-base/backend/modules/system/config/setting"
	rolemod "github.com/duanxldragon/pantheon-base/backend/modules/system/iam/role"
	user "github.com/duanxldragon/pantheon-base/backend/modules/system/iam/user"
	"github.com/duanxldragon/pantheon-base/backend/pkg/authtoken"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/testmysql"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

func setupHandlerTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db := testmysql.Open(t)
	_ = db.AutoMigrate(
		&user.SystemUser{},
		&user.SystemUserRole{},
		&rolemod.SystemRole{},
		&rolemod.SystemRolePermission{},
		&settingmod.SystemSetting{},
		&SystemUserSession{},
		&SystemLogLogin{},
		&SystemLoginThrottle{},
		&SystemAuthSecurityEvent{},
		&SystemUserPasswordHistory{},
	)
	return db
}

func seedHandlerUser(t *testing.T, db *gorm.DB, username, password string) user.SystemUser {
	t.Helper()
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		t.Fatalf("hash test password: %v", err)
	}
	u := user.SystemUser{Username: username, Password: string(hash), Status: common.StatusEnabled}
	if err := db.Create(&u).Error; err != nil {
		t.Fatalf("seed user: %v", err)
	}
	return u
}

func newHandlerTestContext(t *testing.T) (*gin.Context, *httptest.ResponseRecorder) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)
	c.Request = httptest.NewRequest("POST", "/", nil)
	return c, recorder
}

func decodeHandlerResponse(t *testing.T, recorder *httptest.ResponseRecorder) (int, map[string]any) {
	t.Helper()
	var envelope struct {
		Code    int            `json:"code"`
		Message string         `json:"message"`
		Data    map[string]any `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode response %q: %v", recorder.Body.String(), err)
	}
	return envelope.Code, envelope.Data
}

func TestAuthHandler_GetCurrentUserInfo(t *testing.T) {
	db := setupHandlerTestDB(t)
	u := seedHandlerUser(t, db, "info_user", "pass123")
	h := NewAuthHandler(NewRuntime(db, testCredentialRepo(db)))

	c, recorder := newHandlerTestContext(t)
	c.Set("userId", u.ID)
	h.GetCurrentUserInfo(c)

	code, data := decodeHandlerResponse(t, recorder)
	if code != common.CodeSuccess {
		t.Fatalf("expected success, got code=%d body=%s", code, recorder.Body.String())
	}
	if data["username"] != "info_user" {
		t.Fatalf("expected username info_user, got %v", data["username"])
	}
}

func TestAuthHandler_GetCurrentUserInfoUnknownUser(t *testing.T) {
	db := setupHandlerTestDB(t)
	h := NewAuthHandler(NewRuntime(db, testCredentialRepo(db)))

	c, recorder := newHandlerTestContext(t)
	c.Set("userId", uint64(99999))
	h.GetCurrentUserInfo(c)

	code, _ := decodeHandlerResponse(t, recorder)
	if code == common.CodeSuccess {
		t.Fatalf("expected error for unknown user, got body=%s", recorder.Body.String())
	}
}

func TestAuthHandler_UpdatePassword(t *testing.T) {
	db := setupHandlerTestDB(t)
	u := seedHandlerUser(t, db, "pwd_user", "oldpass123")
	h := NewAuthHandler(NewRuntime(db, testCredentialRepo(db)))

	c, recorder := newHandlerTestContext(t)
	c.Set("userId", u.ID)
	c.Request = httptest.NewRequest("POST", "/", strings.NewReader(`{"oldPassword":"oldpass123","newPassword":"newpass456"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	h.UpdatePassword(c)

	code, data := decodeHandlerResponse(t, recorder)
	if code != common.CodeSuccess || data["passwordUpdated"] != true {
		t.Fatalf("expected passwordUpdated=true, got code=%d body=%s", code, recorder.Body.String())
	}
}

func TestAuthHandler_UpdatePasswordInvalidBody(t *testing.T) {
	db := setupHandlerTestDB(t)
	h := NewAuthHandler(NewRuntime(db, testCredentialRepo(db)))

	c, recorder := newHandlerTestContext(t)
	c.Request = httptest.NewRequest("POST", "/", strings.NewReader(`{invalid`))
	c.Request.Header.Set("Content-Type", "application/json")
	h.UpdatePassword(c)

	code, _ := decodeHandlerResponse(t, recorder)
	if code != common.CodeParamInvalid {
		t.Fatalf("expected param invalid, got code=%d body=%s", code, recorder.Body.String())
	}
}

func TestAuthHandler_GetLoginLogList(t *testing.T) {
	db := setupHandlerTestDB(t)
	h := NewAuthHandler(NewRuntime(db, testCredentialRepo(db)))

	if err := db.Create(&SystemLogLogin{Username: "log_user", Status: 1, LoginTime: time.Now()}).Error; err != nil {
		t.Fatalf("seed login log: %v", err)
	}

	c, recorder := newHandlerTestContext(t)
	c.Request = httptest.NewRequest("GET", "/?page=1&pageSize=10", nil)
	h.GetLoginLogList(c)

	code, data := decodeHandlerResponse(t, recorder)
	if code != common.CodeSuccess {
		t.Fatalf("expected success, got code=%d body=%s", code, recorder.Body.String())
	}
	if data["total"] == float64(0) {
		t.Fatalf("expected at least one login log, got %v", data["total"])
	}
}

func TestAuthHandler_GetLoginLogListInvalidQuery(t *testing.T) {
	db := setupHandlerTestDB(t)
	h := NewAuthHandler(NewRuntime(db, testCredentialRepo(db)))

	c, recorder := newHandlerTestContext(t)
	c.Request = httptest.NewRequest("GET", "/?status=notanumber", nil)
	h.GetLoginLogList(c)

	code, _ := decodeHandlerResponse(t, recorder)
	if code != common.CodeParamInvalid {
		t.Fatalf("expected param invalid, got code=%d body=%s", code, recorder.Body.String())
	}
}

func TestAuthHandler_GetOwnLoginLogs(t *testing.T) {
	db := setupHandlerTestDB(t)
	h := NewAuthHandler(NewRuntime(db, testCredentialRepo(db)))

	if err := db.Create(&SystemLogLogin{Username: "own_user", Status: 1, LoginTime: time.Now()}).Error; err != nil {
		t.Fatalf("seed login log: %v", err)
	}

	c, recorder := newHandlerTestContext(t)
	c.Set("username", "own_user")
	c.Request = httptest.NewRequest("GET", "/?page=1&pageSize=10", nil)
	h.GetOwnLoginLogs(c)

	code, data := decodeHandlerResponse(t, recorder)
	if code != common.CodeSuccess {
		t.Fatalf("expected success, got code=%d body=%s", code, recorder.Body.String())
	}
	if data["total"] == float64(0) {
		t.Fatalf("expected own logs, got %v", data["total"])
	}
}

func TestAuthHandler_GetOwnLoginLogsEmptyUsername(t *testing.T) {
	db := setupHandlerTestDB(t)
	h := NewAuthHandler(NewRuntime(db, testCredentialRepo(db)))

	c, recorder := newHandlerTestContext(t)
	c.Request = httptest.NewRequest("GET", "/", nil)
	h.GetOwnLoginLogs(c)

	code, _ := decodeHandlerResponse(t, recorder)
	if code == common.CodeSuccess {
		t.Fatalf("expected error for empty username, got body=%s", recorder.Body.String())
	}
}

func TestAuthHandler_GetSecurityEventList(t *testing.T) {
	db := setupHandlerTestDB(t)
	h := NewAuthHandler(NewRuntime(db, testCredentialRepo(db)))

	if err := db.Create(&SystemAuthSecurityEvent{
		UserID:     1,
		Username:   "event_user",
		EventType:  "password_wrong",
		Severity:   "medium",
		MessageKey: "auth.security.event.password_wrong",
	}).Error; err != nil {
		t.Fatalf("seed security event: %v", err)
	}

	c, recorder := newHandlerTestContext(t)
	c.Request = httptest.NewRequest("GET", "/?page=1&pageSize=10", nil)
	h.GetSecurityEventList(c)

	code, data := decodeHandlerResponse(t, recorder)
	if code != common.CodeSuccess {
		t.Fatalf("expected success, got code=%d body=%s", code, recorder.Body.String())
	}
	if data["total"] == float64(0) {
		t.Fatalf("expected at least one event, got %v", data["total"])
	}
}

func TestAuthHandler_AcknowledgeSecurityEvent(t *testing.T) {
	db := setupHandlerTestDB(t)
	h := NewAuthHandler(NewRuntime(db, testCredentialRepo(db)))

	event := SystemAuthSecurityEvent{
		UserID:     1,
		Username:   "ack_user",
		EventType:  "password_wrong",
		Severity:   "medium",
		MessageKey: "auth.security.event.password_wrong",
	}
	if err := db.Create(&event).Error; err != nil {
		t.Fatalf("seed security event: %v", err)
	}

	c, recorder := newHandlerTestContext(t)
	c.Params = gin.Params{{Key: "id", Value: "1"}}
	c.Set("userId", uint64(7))
	c.Set("username", "auditor")
	c.Request = httptest.NewRequest("POST", "/", strings.NewReader(`{"acknowledgementNote":"reviewed"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	h.AcknowledgeSecurityEvent(c)

	code, data := decodeHandlerResponse(t, recorder)
	if code != common.CodeSuccess || data["acknowledged"] != true {
		t.Fatalf("expected acknowledged=true, got code=%d body=%s", code, recorder.Body.String())
	}
}

func TestAuthHandler_AcknowledgeSecurityEventInvalidID(t *testing.T) {
	db := setupHandlerTestDB(t)
	h := NewAuthHandler(NewRuntime(db, testCredentialRepo(db)))

	c, recorder := newHandlerTestContext(t)
	c.Params = gin.Params{{Key: "id", Value: "abc"}}
	c.Request = httptest.NewRequest("POST", "/", strings.NewReader(`{}`))
	c.Request.Header.Set("Content-Type", "application/json")
	h.AcknowledgeSecurityEvent(c)

	code, _ := decodeHandlerResponse(t, recorder)
	if code != common.CodeParamInvalid {
		t.Fatalf("expected param invalid, got code=%d body=%s", code, recorder.Body.String())
	}
}

func TestAuthHandler_BatchAcknowledgeSecurityEvents(t *testing.T) {
	db := setupHandlerTestDB(t)
	h := NewAuthHandler(NewRuntime(db, testCredentialRepo(db)))

	events := []SystemAuthSecurityEvent{
		{UserID: 1, Username: "batch_user", EventType: "password_wrong", Severity: "medium", MessageKey: "auth.security.event.password_wrong"},
		{UserID: 1, Username: "batch_user", EventType: "password_wrong", Severity: "medium", MessageKey: "auth.security.event.password_wrong"},
	}
	if err := db.Create(&events).Error; err != nil {
		t.Fatalf("seed security events: %v", err)
	}

	c, recorder := newHandlerTestContext(t)
	c.Set("userId", uint64(7))
	c.Set("username", "auditor")
	c.Request = httptest.NewRequest("POST", "/", strings.NewReader(`{"ids":[1,2],"acknowledgementNote":"batch"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	h.BatchAcknowledgeSecurityEvents(c)

	code, data := decodeHandlerResponse(t, recorder)
	if code != common.CodeSuccess {
		t.Fatalf("expected success, got code=%d body=%s", code, recorder.Body.String())
	}
	if data["acknowledgedCount"] == float64(0) {
		t.Fatalf("expected acknowledgedCount > 0, got %v", data["acknowledgedCount"])
	}
}

func TestAuthHandler_CleanupSecurityEvents(t *testing.T) {
	db := setupHandlerTestDB(t)
	h := NewAuthHandler(NewRuntime(db, testCredentialRepo(db)))

	oldTime := time.Now().AddDate(0, 0, -60)
	now := time.Now()
	ackAt := now.AddDate(0, 0, -30)
	if err := db.Create(&SystemAuthSecurityEvent{
		UserID: 1, Username: "old_user", EventType: "password_wrong", Severity: "medium",
		MessageKey: "auth.security.event.password_wrong", CreatedAt: oldTime,
		AcknowledgedAt:     &ackAt,
		AcknowledgedByUser: "auditor",
	}).Error; err != nil {
		t.Fatalf("seed acknowledged old event: %v", err)
	}

	c, recorder := newHandlerTestContext(t)
	c.Request = httptest.NewRequest("POST", "/", strings.NewReader(`{"retentionDays":30}`))
	c.Request.Header.Set("Content-Type", "application/json")
	h.CleanupSecurityEvents(c)

	code, data := decodeHandlerResponse(t, recorder)
	if code != common.CodeSuccess {
		t.Fatalf("expected success, got code=%d body=%s", code, recorder.Body.String())
	}
	if data["clearedCount"] == float64(0) {
		t.Fatalf("expected clearedCount > 0, got %v", data["clearedCount"])
	}
}

func TestAuthHandler_CleanupLoginLogs(t *testing.T) {
	db := setupHandlerTestDB(t)
	h := NewAuthHandler(NewRuntime(db, testCredentialRepo(db)))

	oldTime := time.Now().AddDate(0, 0, -60)
	if err := db.Create(&SystemLogLogin{Username: "old_log", Status: 1, LoginTime: oldTime}).Error; err != nil {
		t.Fatalf("seed old log: %v", err)
	}

	c, recorder := newHandlerTestContext(t)
	c.Request = httptest.NewRequest("POST", "/", strings.NewReader(`{"retentionDays":30}`))
	c.Request.Header.Set("Content-Type", "application/json")
	h.CleanupLoginLogs(c)

	code, data := decodeHandlerResponse(t, recorder)
	if code != common.CodeSuccess {
		t.Fatalf("expected success, got code=%d body=%s", code, recorder.Body.String())
	}
	if data["clearedCount"] == float64(0) {
		t.Fatalf("expected clearedCount > 0, got %v", data["clearedCount"])
	}
}

func TestAuthHandler_CleanupLoginLogsInvalidBody(t *testing.T) {
	db := setupHandlerTestDB(t)
	h := NewAuthHandler(NewRuntime(db, testCredentialRepo(db)))

	c, recorder := newHandlerTestContext(t)
	c.Request = httptest.NewRequest("POST", "/", strings.NewReader(`{`))
	c.Request.Header.Set("Content-Type", "application/json")
	h.CleanupLoginLogs(c)

	code, _ := decodeHandlerResponse(t, recorder)
	if code != common.CodeParamInvalid {
		t.Fatalf("expected param invalid, got code=%d body=%s", code, recorder.Body.String())
	}
}

func TestAuthHandler_CleanupHistoricSessions(t *testing.T) {
	db := setupHandlerTestDB(t)
	h := NewAuthHandler(NewRuntime(db, testCredentialRepo(db)))

	u := seedHandlerUser(t, db, "cleanup_sessions_user", "pass123")
	oldRevokedAt := time.Now().AddDate(0, 0, -60)
	if err := db.Create(&SystemUserSession{
		SessionID:        "old-revoked-session",
		UserID:           u.ID,
		RefreshJTI:       "old-jti",
		RefreshExpiresAt: time.Now().AddDate(0, 0, -60),
		RevokedAt:        &oldRevokedAt,
		CreatedAt:        time.Now().AddDate(0, 0, -60),
	}).Error; err != nil {
		t.Fatalf("seed old session: %v", err)
	}

	c, recorder := newHandlerTestContext(t)
	c.Request = httptest.NewRequest("POST", "/", strings.NewReader(`{"retentionDays":30}`))
	c.Request.Header.Set("Content-Type", "application/json")
	h.CleanupHistoricSessions(c)

	code, data := decodeHandlerResponse(t, recorder)
	if code != common.CodeSuccess {
		t.Fatalf("expected success, got code=%d body=%s", code, recorder.Body.String())
	}
	if data["clearedCount"] == float64(0) {
		t.Fatalf("expected clearedCount > 0, got %v", data["clearedCount"])
	}
}

func TestAuthHandler_BatchRevokeSessions(t *testing.T) {
	db := setupHandlerTestDB(t)
	h := NewAuthHandler(NewRuntime(db, testCredentialRepo(db)))

	u := seedHandlerUser(t, db, "revoke_user", "pass123")
	sessions := []SystemUserSession{
		{SessionID: "revoke-a", UserID: u.ID, RefreshJTI: "jti-a", RefreshExpiresAt: time.Now().Add(24 * time.Hour), CreatedAt: time.Now().Add(-2 * time.Hour)},
		{SessionID: "revoke-b", UserID: u.ID, RefreshJTI: "jti-b", RefreshExpiresAt: time.Now().Add(24 * time.Hour), CreatedAt: time.Now().Add(-time.Hour)},
	}
	if err := db.Create(&sessions).Error; err != nil {
		t.Fatalf("seed sessions: %v", err)
	}

	c, recorder := newHandlerTestContext(t)
	c.Set("sessionId", "current-session")
	c.Request = httptest.NewRequest("POST", "/", strings.NewReader(`{"sessionIds":["revoke-a","revoke-b"]}`))
	c.Request.Header.Set("Content-Type", "application/json")
	h.BatchRevokeSessions(c)

	code, data := decodeHandlerResponse(t, recorder)
	if code != common.CodeSuccess {
		t.Fatalf("expected success, got code=%d body=%s", code, recorder.Body.String())
	}
	if data["revokedCount"] != float64(2) {
		t.Fatalf("expected revokedCount=2, got %v", data["revokedCount"])
	}
}

func TestAuthHandler_BatchDeleteLoginLogs(t *testing.T) {
	db := setupHandlerTestDB(t)
	h := NewAuthHandler(NewRuntime(db, testCredentialRepo(db)))

	logs := []SystemLogLogin{
		{Username: "del_log_1", Status: 1, LoginTime: time.Now()},
		{Username: "del_log_2", Status: 1, LoginTime: time.Now()},
	}
	if err := db.Create(&logs).Error; err != nil {
		t.Fatalf("seed logs: %v", err)
	}

	c, recorder := newHandlerTestContext(t)
	c.Request = httptest.NewRequest("POST", "/", strings.NewReader(`{"ids":[`+
		strconv.FormatUint(logs[0].ID, 10)+`,`+strconv.FormatUint(logs[1].ID, 10)+`]}`))
	c.Request.Header.Set("Content-Type", "application/json")
	h.BatchDeleteLoginLogs(c)

	code, data := decodeHandlerResponse(t, recorder)
	if code != common.CodeSuccess {
		t.Fatalf("expected success, got code=%d body=%s", code, recorder.Body.String())
	}
	if data["deletedCount"] != float64(2) {
		t.Fatalf("expected deletedCount=2, got %v", data["deletedCount"])
	}
}

func TestAuthHandler_GetSessionList(t *testing.T) {
	db := setupHandlerTestDB(t)
	h := NewAuthHandler(NewRuntime(db, testCredentialRepo(db)))

	u := seedHandlerUser(t, db, "session_list_user", "pass123")
	if err := db.Create(&SystemUserSession{
		SessionID:        "list-session",
		UserID:           u.ID,
		RefreshJTI:       "list-jti",
		RefreshExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt:        time.Now(),
	}).Error; err != nil {
		t.Fatalf("seed session: %v", err)
	}

	c, recorder := newHandlerTestContext(t)
	c.Request = httptest.NewRequest("GET", "/?page=1&pageSize=10", nil)
	h.GetSessionList(c)

	code, data := decodeHandlerResponse(t, recorder)
	if code != common.CodeSuccess {
		t.Fatalf("expected success, got code=%d body=%s", code, recorder.Body.String())
	}
	if data["total"] == float64(0) {
		t.Fatalf("expected sessions, got %v", data["total"])
	}
}

func TestAuthHandler_RevokeAnySession(t *testing.T) {
	db := setupHandlerTestDB(t)
	h := NewAuthHandler(NewRuntime(db, testCredentialRepo(db)))

	u := seedHandlerUser(t, db, "any_revoke_user", "pass123")
	if err := db.Create(&SystemUserSession{
		SessionID:        "any-session",
		UserID:           u.ID,
		RefreshJTI:       "any-jti",
		RefreshExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt:        time.Now(),
	}).Error; err != nil {
		t.Fatalf("seed session: %v", err)
	}

	c, recorder := newHandlerTestContext(t)
	c.Params = gin.Params{{Key: "id", Value: "any-session"}}
	c.Set("sessionId", "admin-current")
	c.Request = httptest.NewRequest("POST", "/", nil)
	h.RevokeAnySession(c)

	code, data := decodeHandlerResponse(t, recorder)
	if code != common.CodeSuccess || data["revoked"] != true {
		t.Fatalf("expected revoked=true, got code=%d body=%s", code, recorder.Body.String())
	}
}

func TestAuthHandler_RevokeAnySessionEmptyID(t *testing.T) {
	db := setupHandlerTestDB(t)
	h := NewAuthHandler(NewRuntime(db, testCredentialRepo(db)))

	c, recorder := newHandlerTestContext(t)
	c.Params = gin.Params{{Key: "id", Value: " "}}
	c.Set("sessionId", "admin-current")
	h.RevokeAnySession(c)

	code, _ := decodeHandlerResponse(t, recorder)
	if code == common.CodeSuccess {
		t.Fatalf("expected error for empty session id, got body=%s", recorder.Body.String())
	}
}

func TestAuthHandler_RevokeSession(t *testing.T) {
	db := setupHandlerTestDB(t)
	h := NewAuthHandler(NewRuntime(db, testCredentialRepo(db)))

	u := seedHandlerUser(t, db, "self_revoke_user", "pass123")
	if err := db.Create(&SystemUserSession{
		SessionID:        "self-session",
		UserID:           u.ID,
		RefreshJTI:       "self-jti",
		RefreshExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt:        time.Now(),
	}).Error; err != nil {
		t.Fatalf("seed session: %v", err)
	}

	c, recorder := newHandlerTestContext(t)
	c.Params = gin.Params{{Key: "id", Value: "self-session"}}
	h.RevokeSession(c)

	code, data := decodeHandlerResponse(t, recorder)
	if code != common.CodeSuccess || data["revoked"] != true {
		t.Fatalf("expected revoked=true, got code=%d body=%s", code, recorder.Body.String())
	}
}

func TestAuthHandler_GetSessions(t *testing.T) {
	db := setupHandlerTestDB(t)
	h := NewAuthHandler(NewRuntime(db, testCredentialRepo(db)))

	u := seedHandlerUser(t, db, "own_sessions_user", "pass123")
	if err := db.Create(&SystemUserSession{
		SessionID:        "own-session",
		UserID:           u.ID,
		RefreshJTI:       "own-jti",
		RefreshExpiresAt: time.Now().Add(24 * time.Hour),
		LastActivityAt:   timePtr(time.Now().Add(-time.Minute)),
		CreatedAt:        time.Now().Add(-time.Hour),
	}).Error; err != nil {
		t.Fatalf("seed session: %v", err)
	}

	c, recorder := newHandlerTestContext(t)
	c.Set("userId", u.ID)
	c.Set("sessionId", "own-session")
	h.GetSessions(c)

	// GetSessions returns a top-level data array (not an object).
	var envelope struct {
		Code int                      `json:"code"`
		Data []map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("decode array response %q: %v", recorder.Body.String(), err)
	}
	if envelope.Code != common.CodeSuccess {
		t.Fatalf("expected success, got code=%d body=%s", envelope.Code, recorder.Body.String())
	}
	if len(envelope.Data) != 1 || envelope.Data[0]["sessionId"] != "own-session" {
		t.Fatalf("expected own-session in response, got %+v", envelope.Data)
	}
}

func TestAuthHandler_LogoutHandler(t *testing.T) {
	db := setupHandlerTestDB(t)
	var invalidatedToken string
	h := NewAuthHandler(NewRuntime(db, testCredentialRepo(db)), func(token string) {
		invalidatedToken = token
	})
	rdb := setupTestRedis(t)

	u := seedHandlerUser(t, db, "logout_user", "pass123")
	if err := db.Create(&SystemUserSession{
		SessionID:        "logout-session",
		UserID:           u.ID,
		RefreshJTI:       "logout-jti",
		RefreshExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt:        time.Now(),
	}).Error; err != nil {
		t.Fatalf("seed session: %v", err)
	}

	c, recorder := newHandlerTestContext(t)
	c.Set("sessionId", "logout-session")
	accessToken := "logout-access-token"
	if err := authtoken.StoreSession(c, rdb, accessToken, &authtoken.SessionData{
		UserID:    u.ID,
		SessionID: "logout-session",
	}, time.Hour); err != nil {
		t.Fatalf("seed access token: %v", err)
	}
	c.Request.Header.Set("Authorization", "Bearer "+accessToken)
	h.LogoutHandler(c)

	code, data := decodeHandlerResponse(t, recorder)
	if code != common.CodeSuccess || data["loggedOut"] != true {
		t.Fatalf("expected loggedOut=true, got code=%d body=%s", code, recorder.Body.String())
	}
	if _, err := authtoken.ValidateSession(c, rdb, accessToken); err == nil {
		t.Fatal("expected logout to delete the current access token")
	}
	if invalidatedToken != accessToken {
		t.Fatalf("expected logout to invalidate token cache for %q, got %q", accessToken, invalidatedToken)
	}
}

func TestAuthHandler_TouchActivity(t *testing.T) {
	db := setupHandlerTestDB(t)
	h := NewAuthHandler(NewRuntime(db, testCredentialRepo(db)))

	u := seedHandlerUser(t, db, "touch_user", "pass123")
	if err := db.Create(&SystemUserSession{
		SessionID:        "touch-session-2",
		UserID:           u.ID,
		RefreshJTI:       "touch-jti-2",
		RefreshExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt:        time.Now().Add(-time.Hour),
	}).Error; err != nil {
		t.Fatalf("seed session: %v", err)
	}

	c, recorder := newHandlerTestContext(t)
	c.Set("sessionId", "touch-session-2")
	c.Set("userId", u.ID)
	c.Request = httptest.NewRequest("POST", "/", nil)
	c.Request.Header.Set("User-Agent", "Test Browser")
	h.TouchActivity(c)

	code, data := decodeHandlerResponse(t, recorder)
	if code != common.CodeSuccess || data["touched"] != true {
		t.Fatalf("expected touched=true, got code=%d body=%s", code, recorder.Body.String())
	}
}

func TestAuthHandler_GetSecurityOverview(t *testing.T) {
	db := setupHandlerTestDB(t)
	h := NewAuthHandler(NewRuntime(db, testCredentialRepo(db)))

	u := seedHandlerUser(t, db, "overview_user", "pass12345")
	if err := db.Create(&SystemUserSession{
		SessionID:        "overview-session",
		UserID:           u.ID,
		RefreshJTI:       "overview-jti",
		RefreshExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt:        time.Now().Add(-time.Hour),
	}).Error; err != nil {
		t.Fatalf("seed session: %v", err)
	}

	c, recorder := newHandlerTestContext(t)
	c.Set("userId", u.ID)
	c.Set("username", u.Username)
	c.Set("sessionId", "overview-session")
	h.GetSecurityOverview(c)

	code, _ := decodeHandlerResponse(t, recorder)
	if code != common.CodeSuccess {
		t.Fatalf("expected success, got body=%s", recorder.Body.String())
	}
}

func TestAuthHandler_buildLoginSourceKey(t *testing.T) {
	if got := buildLoginSourceKey(" 10.0.0.1 "); got != "ip:10.0.0.1" {
		t.Fatalf("expected ip:10.0.0.1, got %q", got)
	}
	if got := buildLoginSourceKey(""); got != loginSourceKeyUnknownIP {
		t.Fatalf("expected ip:unknown, got %q", got)
	}
	if got := buildLoginSourceKey("   "); got != loginSourceKeyUnknownIP {
		t.Fatalf("expected ip:unknown for whitespace, got %q", got)
	}
}

func TestAuthHandler_parseRefreshTokenWithContextEmpty(t *testing.T) {
	if _, err := parseRefreshTokenWithContext(context.TODO(), ""); err == nil {
		t.Fatal("expected error for empty refresh token")
	}
}

func TestAuthHandler_LoginHandlerInvalidBody(t *testing.T) {
	db := setupHandlerTestDB(t)
	h := NewAuthHandler(NewRuntime(db, testCredentialRepo(db)))

	c, recorder := newHandlerTestContext(t)
	c.Request = httptest.NewRequest("POST", "/", strings.NewReader(`{`))
	c.Request.Header.Set("Content-Type", "application/json")
	h.LoginHandler(c)

	code, _ := decodeHandlerResponse(t, recorder)
	if code != common.CodeParamInvalid {
		t.Fatalf("expected param invalid, got code=%d body=%s", code, recorder.Body.String())
	}
}

func TestAuthHandler_LoginHandlerUnknownUserRecordsFailureLog(t *testing.T) {
	db := setupHandlerTestDB(t)
	h := NewAuthHandler(NewRuntime(db, testCredentialRepo(db)))

	c, recorder := newHandlerTestContext(t)
	c.Request = httptest.NewRequest("POST", "/", strings.NewReader(`{"username":"ghost_user","password":"whatever"}`))
	c.Request.Header.Set("Content-Type", "application/json")
	h.LoginHandler(c)

	code, _ := decodeHandlerResponse(t, recorder)
	if code != common.CodeUnauthorized {
		t.Fatalf("expected unauthorized, got code=%d body=%s", code, recorder.Body.String())
	}

	var logCount int64
	if err := db.Model(&SystemLogLogin{}).Where("username = ?", "ghost_user").Count(&logCount).Error; err != nil {
		t.Fatalf("count login logs: %v", err)
	}
	if logCount != 1 {
		t.Fatalf("expected failure login log recorded, got %d", logCount)
	}
}

func TestAuthHandler_VerifyMFAHandlerInvalidBody(t *testing.T) {
	db := setupHandlerTestDB(t)
	h := NewAuthHandler(NewRuntime(db, testCredentialRepo(db)))

	c, recorder := newHandlerTestContext(t)
	c.Request = httptest.NewRequest("POST", "/", strings.NewReader(`{`))
	c.Request.Header.Set("Content-Type", "application/json")
	h.VerifyMFAHandler(c)

	code, _ := decodeHandlerResponse(t, recorder)
	if code != common.CodeParamInvalid {
		t.Fatalf("expected param invalid, got code=%d body=%s", code, recorder.Body.String())
	}
}

func TestAuthHandler_RefreshTokenHandlerEmptyBody(t *testing.T) {
	db := setupHandlerTestDB(t)
	h := NewAuthHandler(NewRuntime(db, testCredentialRepo(db)))

	c, recorder := newHandlerTestContext(t)
	c.Request = httptest.NewRequest("POST", "/", strings.NewReader(`{}`))
	c.Request.Header.Set("Content-Type", "application/json")
	h.RefreshTokenHandler(c)

	// RefreshTokenReq has binding:"required" — an empty/missing refreshToken
	// fails binding and returns param.invalid (400) before the token check.
	code, _ := decodeHandlerResponse(t, recorder)
	if code != common.CodeParamInvalid {
		t.Fatalf("expected param invalid, got code=%d body=%s", code, recorder.Body.String())
	}
}

func TestAuthHandler_VerifyOperationPasswordInvalidBody(t *testing.T) {
	db := setupHandlerTestDB(t)
	h := NewAuthHandler(NewRuntime(db, testCredentialRepo(db)))

	c, recorder := newHandlerTestContext(t)
	c.Request = httptest.NewRequest("POST", "/", strings.NewReader(`{}`))
	c.Request.Header.Set("Content-Type", "application/json")
	h.VerifyOperationPassword(c)

	code, _ := decodeHandlerResponse(t, recorder)
	if code != common.CodeParamInvalid {
		t.Fatalf("expected param invalid, got code=%d body=%s", code, recorder.Body.String())
	}
}

func TestAuthHandler_ExportLoginLogsInvalidBody(t *testing.T) {
	db := setupHandlerTestDB(t)
	h := NewAuthHandler(NewRuntime(db, testCredentialRepo(db)))

	c, recorder := newHandlerTestContext(t)
	c.Request = httptest.NewRequest("POST", "/", strings.NewReader(`{`))
	c.Request.Header.Set("Content-Type", "application/json")
	h.ExportLoginLogs(c)

	code, _ := decodeHandlerResponse(t, recorder)
	if code != common.CodeParamInvalid {
		t.Fatalf("expected param invalid, got code=%d body=%s", code, recorder.Body.String())
	}
}

func TestAuthHandler_ExportLoginLogsWritesCSV(t *testing.T) {
	db := setupHandlerTestDB(t)
	h := NewAuthHandler(NewRuntime(db, testCredentialRepo(db)))

	if err := db.Create(&SystemLogLogin{Username: "export_user", Status: 1, LoginTime: time.Now()}).Error; err != nil {
		t.Fatalf("seed log: %v", err)
	}

	c, recorder := newHandlerTestContext(t)
	c.Request = httptest.NewRequest("POST", "/", strings.NewReader(`{}`))
	c.Request.Header.Set("Content-Type", "application/json")
	h.ExportLoginLogs(c)

	if recorder.Code != 200 {
		t.Fatalf("expected CSV 200 response, got %d body=%s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Header().Get("Content-Type"), "text/csv") {
		t.Fatalf("expected text/csv content type, got %q", recorder.Header().Get("Content-Type"))
	}
}
