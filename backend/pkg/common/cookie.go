package common

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"
)

const (
	CookieAccessToken  = "pantheon_access_token"
	CookieRefreshToken = "pantheon_refresh_token"
	CookieCSRFToken    = "pantheon_csrf_token"
)

func setTokenCookie(w http.ResponseWriter, name, value string, maxAge int, httpOnly bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: httpOnly,
		Secure:   false,
		SameSite: http.SameSiteStrictMode,
	})
}

func SetAccessTokenCookie(w http.ResponseWriter, token string) {
	ttl := int((15 * time.Minute).Seconds())
	setTokenCookie(w, CookieAccessToken, token, ttl, true)
}

func SetRefreshTokenCookie(w http.ResponseWriter, token string) {
	ttl := int((7 * 24 * time.Hour).Seconds())
	setTokenCookie(w, CookieRefreshToken, token, ttl, true)
}

func ClearTokenCookies(w http.ResponseWriter) {
	setTokenCookie(w, CookieAccessToken, "", -1, true)
	setTokenCookie(w, CookieRefreshToken, "", -1, true)
	setTokenCookie(w, CookieCSRFToken, "", -1, false)
}

func SetCSRFCookie(w http.ResponseWriter) (string, error) {
	token := generateCSRFToken()
	setTokenCookie(w, CookieCSRFToken, token, int((24*time.Hour).Seconds()), false)
	return token, nil
}

func generateCSRFToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return hex.EncodeToString([]byte("fallback-csrf-token-32bytes!!"))
	}
	return hex.EncodeToString(b)
}
