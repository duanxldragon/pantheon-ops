package http

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

	// AccessTokenTTL is the effective TTL for access tokens, overridable via SetTokenTTL.
	AccessTokenTTL = 15 * time.Minute
	// RefreshTokenTTL is the effective TTL for refresh tokens, overridable via SetTokenTTL.
	RefreshTokenTTL = 7 * 24 * time.Hour
)

var accessTokenTTL = AccessTokenTTL
var refreshTokenTTL = RefreshTokenTTL

func setCookie(w http.ResponseWriter, name, value string, maxAge int, sameSite http.SameSite) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   true,
		SameSite: sameSite,
	})
}

func SetAccessTokenCookie(w http.ResponseWriter, token string) {
	ttl := int(accessTokenTTL.Seconds())
	setCookie(w, CookieAccessToken, token, ttl, http.SameSiteStrictMode)
}

func SetRefreshTokenCookie(w http.ResponseWriter, token string) {
	ttl := int(refreshTokenTTL.Seconds())
	setCookie(w, CookieRefreshToken, token, ttl, http.SameSiteStrictMode)
}

func ClearTokenCookies(w http.ResponseWriter) {
	setCookie(w, CookieAccessToken, "", -1, http.SameSiteStrictMode)
	setCookie(w, CookieRefreshToken, "", -1, http.SameSiteStrictMode)
	setCookie(w, CookieCSRFToken, "", -1, http.SameSiteStrictMode)
	w.Header().Del("X-CSRF-Token")
}

func SetCSRFCookie(w http.ResponseWriter) (string, error) {
	token, err := generateCSRFToken()
	if err != nil {
		return "", err
	}
	setCookie(w, CookieCSRFToken, token, int((24 * time.Hour).Seconds()), http.SameSiteStrictMode)
	w.Header().Set("X-CSRF-Token", token)
	return token, nil
}

func generateCSRFToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// SetTokenTTL overrides the effective token TTLs. Pass zero to use defaults.
func SetTokenTTL(accessTTL, refreshTTL time.Duration) {
	if accessTTL > 0 {
		accessTokenTTL = accessTTL
	}
	if refreshTTL > 0 {
		refreshTokenTTL = refreshTTL
	}
}
