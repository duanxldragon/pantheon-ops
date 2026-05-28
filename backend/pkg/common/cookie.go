package common

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	CookieAccessToken  = "pantheon_access_token"
	CookieRefreshToken = "pantheon_refresh_token"
	CookieCSRFToken    = "pantheon_csrf_token"
)

func shouldUseSecureCookies() bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv("PANTHEON_COOKIE_SECURE")))
	if value == "0" || value == "false" || value == "off" {
		return false
	}
	return true
}

func setCookie(w http.ResponseWriter, name, value string, maxAge int, httpOnly bool, sameSite http.SameSite) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: httpOnly,
		Secure:   shouldUseSecureCookies(),
		SameSite: sameSite,
	})
}

func SetAccessTokenCookie(w http.ResponseWriter, token string) {
	ttl := int((15 * time.Minute).Seconds())
	setCookie(w, CookieAccessToken, token, ttl, true, http.SameSiteStrictMode)
}

func SetRefreshTokenCookie(w http.ResponseWriter, token string) {
	ttl := int((7 * 24 * time.Hour).Seconds())
	setCookie(w, CookieRefreshToken, token, ttl, true, http.SameSiteStrictMode)
}

func ClearTokenCookies(w http.ResponseWriter) {
	setCookie(w, CookieAccessToken, "", -1, true, http.SameSiteStrictMode)
	setCookie(w, CookieRefreshToken, "", -1, true, http.SameSiteStrictMode)
	setCookie(w, CookieCSRFToken, "", -1, false, http.SameSiteStrictMode)
}

func SetCSRFCookie(w http.ResponseWriter) (string, error) {
	token := generateCSRFToken()
	setCookie(w, CookieCSRFToken, token, int((24 * time.Hour).Seconds()), false, http.SameSiteStrictMode)
	return token, nil
}

func generateCSRFToken() string {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return hex.EncodeToString([]byte("fallback-csrf-token-32bytes!!"))
	}
	return hex.EncodeToString(b)
}
