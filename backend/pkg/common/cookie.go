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

func setCookie(w http.ResponseWriter, name, value string, maxAge int, httpOnly bool, sameSite http.SameSite) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		MaxAge:   maxAge,
		HttpOnly: httpOnly,
		Secure:   true,
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
	token, err := generateCSRFToken()
	if err != nil {
		return "", err
	}
	setCookie(w, CookieCSRFToken, token, int((24 * time.Hour).Seconds()), false, http.SameSiteStrictMode)
	return token, nil
}

func generateCSRFToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
