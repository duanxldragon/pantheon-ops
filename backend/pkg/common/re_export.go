package common

import (
	"net/http"

	"github.com/gin-gonic/gin"

	commonhttp "pantheon-ops/backend/pkg/common/http"
	"pantheon-ops/backend/pkg/common/security"
)

// ── Re-exports from http package ─────────────────────────────────────

type Response = commonhttp.Response

// Response codes (re-exported from http)
const (
	CodeSuccess      = commonhttp.CodeSuccess
	CodeError        = commonhttp.CodeError
	CodeParamInvalid = commonhttp.CodeParamInvalid
	CodeUnauthorized = commonhttp.CodeUnauthorized
	CodeForbidden    = commonhttp.CodeForbidden
	CodeNotFound     = commonhttp.CodeNotFound
)

// Cookie constants
const (
	CookieAccessToken  = commonhttp.CookieAccessToken
	CookieRefreshToken = commonhttp.CookieRefreshToken
	CookieCSRFToken    = commonhttp.CookieCSRFToken
)

// Request context constants
const (
	HeaderRequestID     = commonhttp.HeaderRequestID
	HeaderTraceID       = commonhttp.HeaderTraceID
	ContextKeyRequestID = commonhttp.ContextKeyRequestID
	ContextKeyTraceID   = commonhttp.ContextKeyTraceID
)

// Response functions
func Success(c *gin.Context, data interface{}) {
	commonhttp.Success(c, data)
}

func SuccessWithStatus(c *gin.Context, httpStatus int, data interface{}) {
	commonhttp.SuccessWithStatus(c, httpStatus, data)
}

func Fail(c *gin.Context, code int, message string) {
	commonhttp.Fail(c, code, message)
}

func FailWithError(c *gin.Context, code int, err error, fallback string) {
	commonhttp.FailWithError(c, code, err, fallback)
}

func FailWithCode(c *gin.Context, code int, message string) {
	commonhttp.FailWithCode(c, code, message)
}

// Cookie functions
func SetAccessTokenCookie(w http.ResponseWriter, token string) {
	commonhttp.SetAccessTokenCookie(w, token)
}

func SetRefreshTokenCookie(w http.ResponseWriter, token string) {
	commonhttp.SetRefreshTokenCookie(w, token)
}

func ClearTokenCookies(w http.ResponseWriter) {
	commonhttp.ClearTokenCookies(w)
}

func SetCSRFCookie(w http.ResponseWriter) (string, error) {
	return commonhttp.SetCSRFCookie(w)
}

// Request context functions
func GetRequestID(c *gin.Context) string {
	return commonhttp.GetRequestID(c)
}

// ── Re-exports from security package ──────────────────────────────────

var DefaultDevSecrets = security.DefaultDevSecrets

func IsProductionEnv() bool {
	return security.IsProductionEnv()
}

func ResolveSecret(name, fallback string) string {
	return security.ResolveSecret(name, fallback)
}

func ValidateRequiredProductionSecret(name, fallback string) error {
	return security.ValidateRequiredProductionSecret(name, fallback)
}

func GenerateRandomSecret(byteLen int) (string, error) {
	return security.GenerateRandomSecret(byteLen)
}

func InitSecurityConfig() error {
	return security.InitSecurityConfig()
}
