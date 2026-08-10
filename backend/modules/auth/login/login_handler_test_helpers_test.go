package login

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	commonhttp "pantheon-ops/backend/pkg/common/http"
)

type authResponseEnvelope[T any] struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    T      `json:"data"`
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
		if cookie.Name != commonhttp.CookieCSRFToken {
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
