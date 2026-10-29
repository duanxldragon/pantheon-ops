package common

import (
	"github.com/gin-gonic/gin"

	commonhttp "github.com/duanxldragon/pantheon-base/backend/pkg/common/http"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common/security"
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

// Request context functions
func GetRequestID(c *gin.Context) string {
	return commonhttp.GetRequestID(c)
}

// ── Re-exports from security package ──────────────────────────────────

func InitSecurityConfig() error {
	return security.InitSecurityConfig()
}
