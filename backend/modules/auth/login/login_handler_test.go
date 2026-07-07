package login

import (
	"errors"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"pantheon-ops/backend/pkg/authtoken"
	"pantheon-ops/backend/pkg/common"

	"github.com/gin-gonic/gin"
)

func TestFailOnCSRFCookieErrorReturnsFalseWhenErrorIsNil(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)

	if failOnCSRFCookieError(context, nil) {
		t.Fatal("expected nil csrf error to be ignored")
	}
}

func TestFailOnCSRFCookieErrorWritesErrorResponse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)

	if !failOnCSRFCookieError(context, errors.New("csrf unavailable")) {
		t.Fatal("expected csrf error to be handled")
	}
	if recorder.Code != 200 {
		t.Fatalf("expected common failure payload status 200, got %d", recorder.Code)
	}
	if recorder.Body.Len() == 0 {
		t.Fatal("expected failure response body to be written")
	}
	if got := recorder.Header().Get("Content-Type"); got == "" {
		t.Fatal("expected failure response content type to be set")
	}
	if body := recorder.Body.String(); !strings.Contains(body, `"code":`+strconv.Itoa(common.CodeError)) {
		t.Fatalf("expected failure response to include code %d, got %q", common.CodeError, body)
	}
}

func TestWriteLoginSuccessResponseSetsCookiesAndPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)

	expiresAt := time.Date(2026, time.June, 5, 10, 0, 0, 0, time.UTC)
	tokenPair := &authtoken.Pair{
		AccessToken:      "access-token",
		RefreshToken:     "refresh-token",
		TokenType:        "Bearer",
		AccessExpiresAt:  expiresAt,
		RefreshExpiresAt: expiresAt.Add(24 * time.Hour),
		SessionID:        "session-1",
	}
	userInfo := &UserInfoResp{ID: 7, Username: "admin"}

	if !writeLoginSuccessResponse(context, tokenPair, userInfo) {
		t.Fatal("expected login response helper to succeed")
	}

	resp := decodeAuthResponse[AuthTokenResp](t, recorder)
	if resp.Code != common.CodeSuccess {
		t.Fatalf("expected success code, got %d", resp.Code)
	}
	if resp.Data.Token != "" || resp.Data.AccessToken != "" || resp.Data.RefreshToken != "" {
		t.Fatalf("expected login payload to omit raw tokens, got %+v", resp.Data)
	}
	if resp.Data.SessionID != tokenPair.SessionID {
		t.Fatalf("expected session id %q, got %q", tokenPair.SessionID, resp.Data.SessionID)
	}
	if resp.Data.TokenType != tokenPair.TokenType {
		t.Fatalf("expected token type %q, got %q", tokenPair.TokenType, resp.Data.TokenType)
	}
	if resp.Data.User == nil || resp.Data.User.Username != userInfo.Username {
		t.Fatalf("expected user payload %+v, got %+v", userInfo, resp.Data.User)
	}

	assertCSRFCookieAndHeader(
		t,
		recorder,
		"expected login helper to set an httpOnly csrf cookie",
		"expected login helper to expose csrf header",
	)
}

func TestWriteMFASuccessResponseSetsCookiesAndPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)

	respPayload := &AuthTokenResp{
		Token:            "mfa-access",
		AccessToken:      "mfa-access",
		RefreshToken:     "mfa-refresh",
		TokenType:        "Bearer",
		AccessExpiresAt:  "2026-06-05 10:00:00",
		RefreshExpiresAt: "2026-06-06 10:00:00",
		SessionID:        "session-mfa",
		User:             &UserInfoResp{ID: 8, Username: "mfa-user"},
	}

	if !writeMFASuccessResponse(context, respPayload) {
		t.Fatal("expected mfa response helper to succeed")
	}

	resp := decodeAuthResponse[AuthTokenResp](t, recorder)
	if resp.Code != common.CodeSuccess {
		t.Fatalf("expected success code, got %d", resp.Code)
	}
	if resp.Data.Token != "" || resp.Data.AccessToken != "" || resp.Data.RefreshToken != "" {
		t.Fatalf("expected mfa payload to omit raw tokens, got %+v", resp.Data)
	}
	if resp.Data.User == nil || resp.Data.User.Username != respPayload.User.Username {
		t.Fatalf("expected user payload %+v, got %+v", respPayload.User, resp.Data.User)
	}
	if resp.Data.SessionID != respPayload.SessionID {
		t.Fatalf("expected session id %q, got %q", respPayload.SessionID, resp.Data.SessionID)
	}

	assertCSRFCookieAndHeader(
		t,
		recorder,
		"expected mfa helper to set an httpOnly csrf cookie",
		"expected mfa helper to expose csrf header",
	)
}

func TestWriteRefreshSuccessResponseSetsCookiesAndPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)

	expiresAt := time.Date(2026, time.June, 5, 10, 0, 0, 0, time.UTC)
	tokenPair := &authtoken.Pair{
		AccessToken:      "refresh-access",
		RefreshToken:     "refresh-refresh",
		TokenType:        "Bearer",
		AccessExpiresAt:  expiresAt,
		RefreshExpiresAt: expiresAt.Add(24 * time.Hour),
		SessionID:        "session-refresh",
	}

	if !writeRefreshSuccessResponse(context, tokenPair) {
		t.Fatal("expected refresh response helper to succeed")
	}

	resp := decodeAuthResponse[map[string]any](t, recorder)
	if resp.Code != common.CodeSuccess {
		t.Fatalf("expected success code, got %d", resp.Code)
	}
	if _, ok := resp.Data["token"]; ok {
		t.Fatalf("expected refresh payload to omit token, got %#v", resp.Data)
	}
	if _, ok := resp.Data["accessToken"]; ok {
		t.Fatalf("expected refresh payload to omit access token, got %#v", resp.Data)
	}
	if _, ok := resp.Data["refreshToken"]; ok {
		t.Fatalf("expected refresh payload to omit refresh token, got %#v", resp.Data)
	}
	if resp.Data["sessionId"] != tokenPair.SessionID {
		t.Fatalf("expected session id %q, got %#v", tokenPair.SessionID, resp.Data["sessionId"])
	}
	if resp.Data["tokenType"] != tokenPair.TokenType {
		t.Fatalf("expected token type %q, got %#v", tokenPair.TokenType, resp.Data["tokenType"])
	}

	assertCSRFCookieAndHeader(
		t,
		recorder,
		"expected refresh helper to set an httpOnly csrf cookie",
		"expected refresh helper to expose csrf header",
	)
}
