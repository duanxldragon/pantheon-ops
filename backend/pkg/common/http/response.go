package http

import (
	"net/http"
	"regexp"
	"strings"

	"pantheon-base/pkg/logging"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type Response struct {
	Code    int         `json:"code"`
	Data    interface{} `json:"data"`
	Message string      `json:"message"`
}

const (
	CodeSuccess      = 200
	CodeError        = 500
	CodeParamInvalid = 400
	CodeUnauthorized = 401
	CodeForbidden    = 403
	CodeNotFound     = 404
)

var i18nKeyPattern = regexp.MustCompile(`^[a-z0-9_]+(?:\.[a-z0-9_]+)+$`)

func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    CodeSuccess,
		Data:    data,
		Message: "success",
	})
}

func SuccessWithStatus(c *gin.Context, httpStatus int, data interface{}) {
	c.JSON(httpStatus, Response{
		Code:    CodeSuccess,
		Data:    data,
		Message: "success",
	})
}

func Fail(c *gin.Context, code int, message string) {
	c.JSON(http.StatusOK, Response{
		Code:    code,
		Data:    nil,
		Message: message,
	})
}

func isI18nMessageKey(message string) bool {
	return i18nKeyPattern.MatchString(strings.TrimSpace(message))
}

// lastMessageSegment returns the final ": "-delimited segment of a message,
// which is where the sentinel-wrapped i18n key lives (see pkg/common/error.go).
func lastMessageSegment(message string) string {
	message = strings.TrimSpace(message)
	if idx := strings.LastIndex(message, ": "); idx != -1 {
		message = message[idx+2:]
	}
	return message
}

func resolveErrorMessageKey(err error, fallback string) string {
	if err == nil {
		if strings.TrimSpace(fallback) != "" {
			return fallback
		}
		return "request.failed"
	}
	message := lastMessageSegment(err.Error())
	if isI18nMessageKey(message) {
		return message
	}
	if strings.TrimSpace(fallback) != "" {
		return fallback
	}
	return "request.failed"
}

func FailWithError(c *gin.Context, code int, err error, fallback string) {
	// When err carries no i18n message key, resolveErrorMessageKey replaces it
	// with the generic fallback ("request.failed"), discarding the underlying
	// cause (e.g. a raw GORM/driver error). Log the original here so the detail
	// is not silently swallowed at the HTTP boundary.
	if err != nil && !isI18nMessageKey(lastMessageSegment(err.Error())) {
		logging.Error("request failed with unclassified error",
			zap.Int("code", code),
			zap.String("requestId", GetRequestID(c)),
			zap.String("error", err.Error()),
		)
	}
	Fail(c, code, resolveErrorMessageKey(err, fallback))
}

func FailWithCode(c *gin.Context, code int, message string) {
	c.JSON(code, Response{
		Code:    code,
		Data:    nil,
		Message: message,
	})
}

func GetUserID(c *gin.Context) uint64 {
	for _, key := range []string{"userID", "userId"} {
		val, ok := c.Get(key)
		if !ok {
			continue
		}
		userID, ok := val.(uint64)
		if ok {
			return userID
		}
	}
	return 0
}
