package common

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTokenCookiesAlwaysSecure(t *testing.T) {
	recorder := httptest.NewRecorder()
	SetAccessTokenCookie(recorder, "token")

	cookie := recorder.Result().Cookies()[0]
	if !cookie.Secure {
		t.Fatal("expected secure cookie to always be enabled")
	}
}

func TestCSRFCookieAlwaysUsesSecureFlag(t *testing.T) {
	recorder := httptest.NewRecorder()
	_, err := SetCSRFCookie(recorder)
	if err != nil {
		t.Fatalf("set csrf cookie: %v", err)
	}

	cookie := recorder.Result().Cookies()[0]
	if cookie.Name != CookieCSRFToken {
		t.Fatalf("expected csrf cookie, got %s", cookie.Name)
	}
	if !cookie.Secure {
		t.Fatal("expected csrf cookie to always use secure flag")
	}
}

func TestCSRFCookieKeepsHttpOnlyDisabled(t *testing.T) {
	recorder := httptest.NewRecorder()
	token, err := SetCSRFCookie(recorder)
	if err != nil {
		t.Fatalf("set csrf cookie: %v", err)
	}
	if token == "" {
		t.Fatal("expected csrf token to be generated")
	}

	cookie := recorder.Result().Cookies()[0]
	if cookie.Name != CookieCSRFToken {
		t.Fatalf("expected csrf cookie, got %s", cookie.Name)
	}
	if cookie.HttpOnly {
		t.Fatal("expected csrf cookie to remain readable by the browser")
	}
	if !strings.Contains(recorder.Header().Get("Set-Cookie"), "SameSite=Strict") {
		t.Fatal("expected strict same-site cookie")
	}
}

func TestGenerateCSRFTokenReturnsValue(t *testing.T) {
	token, err := generateCSRFToken()
	if err != nil {
		t.Fatalf("generate csrf token: %v", err)
	}
	if token == "" {
		t.Fatal("expected csrf token to be generated")
	}
}

func TestClearTokenCookiesUsesDeletionMarkers(t *testing.T) {
	recorder := httptest.NewRecorder()
	ClearTokenCookies(recorder)

	headers := recorder.Header().Values("Set-Cookie")
	if len(headers) != 3 {
		t.Fatalf("expected 3 cookies to be cleared, got %d", len(headers))
	}
	for _, raw := range headers {
		if !strings.Contains(raw, "Max-Age=0") {
			t.Fatalf("expected deletion marker, got %s", raw)
		}
		if !strings.Contains(raw, "=;") {
			t.Fatalf("expected cleared cookie value, got %s", raw)
		}
	}
}
