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
