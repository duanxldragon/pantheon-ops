package common

import (
	"net/http"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
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

func IsI18nMessageKey(message string) bool {
	return i18nKeyPattern.MatchString(strings.TrimSpace(message))
}

func ResolveErrorMessageKey(err error, fallback string) string {
	if err == nil {
		if strings.TrimSpace(fallback) != "" {
			return fallback
		}
		return "request.failed"
	}
	message := strings.TrimSpace(err.Error())
	if IsI18nMessageKey(message) {
		return message
	}
	if strings.TrimSpace(fallback) != "" {
		return fallback
	}
	return "request.failed"
}

func FailWithError(c *gin.Context, code int, err error, fallback string) {
	Fail(c, code, ResolveErrorMessageKey(err, fallback))
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
