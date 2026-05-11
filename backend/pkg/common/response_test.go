package common

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestResolveErrorMessageKeyPrefersI18nKey(t *testing.T) {
	message := ResolveErrorMessageKey(assertErr("user.create.error.username_exists"), "request.failed")
	if message != "user.create.error.username_exists" {
		t.Fatalf("expected original message key, got %s", message)
	}
}

func TestResolveErrorMessageKeyFallsBackForNaturalLanguage(t *testing.T) {
	message := ResolveErrorMessageKey(assertErr("sql: no rows in result set"), "user.list.error")
	if message != "user.list.error" {
		t.Fatalf("expected fallback key, got %s", message)
	}
}

func TestResolveErrorMessageKeyUsesDefaultFallback(t *testing.T) {
	message := ResolveErrorMessageKey(assertErr("unexpected failure"), "")
	if message != "request.failed" {
		t.Fatalf("expected default fallback key, got %s", message)
	}
}

func TestFailWithErrorPreservesI18nKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)

	FailWithError(c, CodeError, assertErr("dept.delete.error.has_users"), "request.failed")

	var resp Response
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Message != "dept.delete.error.has_users" {
		t.Fatalf("expected original i18n key, got %s", resp.Message)
	}
}

func TestFailWithErrorMasksNaturalLanguage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	recorder := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(recorder)

	FailWithError(c, CodeError, assertErr("sql: no rows in result set"), "request.failed")

	var resp Response
	if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Message != "request.failed" {
		t.Fatalf("expected fallback key, got %s", resp.Message)
	}
}

type assertErr string

func (e assertErr) Error() string {
	return string(e)
}
