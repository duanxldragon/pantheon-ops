package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"pantheon-ops/backend/pkg/common"

	"gorm.io/gorm"

	"github.com/gin-gonic/gin"
)

type operationLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

const (
	operationLogTitleKey        = "operationLog.title"
	operationLogBusinessTypeKey = "operationLog.businessType"
	operationLogParamKey        = "operationLog.param"
	operationLogResultKey       = "operationLog.result"
	operationLogStatusKey       = "operationLog.status"
	operationLogErrorMsgKey     = "operationLog.errorMsg"
)

func (w operationLogWriter) Write(data []byte) (int, error) {
	w.body.Write(data)
	return w.ResponseWriter.Write(data)
}

type SystemLogOper struct {
	ID              uint64 `gorm:"primaryKey;autoIncrement"`
	RequestID       string `gorm:"size:64;index:idx_system_log_oper_request_id"`
	Title           string `gorm:"size:64"`
	BusinessType    int    `gorm:"default:0"`
	Method          string `gorm:"size:128"`
	OperName        string `gorm:"size:64"`
	OperURL         string `gorm:"size:255"`
	OperIP          string `gorm:"size:128"`
	SourceDomain    string `gorm:"size:32;index:idx_system_log_oper_source_domain_page,priority:1"`
	SourcePage      string `gorm:"size:32;index:idx_system_log_oper_source_domain_page,priority:2;index:idx_system_log_oper_source_page"`
	OperParam       string `gorm:"type:text"`
	JsonResult      string `gorm:"type:text"`
	Status          int    `gorm:"default:1"`
	FailureCategory string `gorm:"size:32;index:idx_system_log_oper_failure_category"`
	ErrorMsg        string `gorm:"type:text"`
	OperTime        time.Time
	CostTime        int64
}

func (SystemLogOper) TableName() string {
	return "system_log_oper"
}

// OperationLogMiddleware 异步记录操作日志。
func OperationLogMiddleware(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		if db == nil || c.Request.Method == http.MethodGet {
			c.Next()
			return
		}

		start := time.Now()
		requestBody := readAndRestoreBody(c)
		responseBody := &bytes.Buffer{}
		c.Writer = operationLogWriter{ResponseWriter: c.Writer, body: responseBody}

		c.Next()

		username := ""
		if value, ok := c.Get("username"); ok {
			username, _ = value.(string)
		}

		status := 1
		errorMessage := ""
		if c.Writer.Status() >= http.StatusBadRequest {
			status = 2
			errorMessage = http.StatusText(c.Writer.Status())
		}
		if code, message := parseBusinessResult(responseBody.String()); code != 0 && code != 200 {
			status = 2
			errorMessage = message
		}
		if overrideStatus, ok := readOperationLogStatus(c); ok {
			status = overrideStatus
		}
		if overrideErrorMessage := readOperationLogErrorMsg(c); overrideErrorMessage != "" {
			errorMessage = overrideErrorMessage
		}

		log := SystemLogOper{
			RequestID:       strings.TrimSpace(common.GetRequestID(c)),
			Title:           readOperationLogTitle(c),
			BusinessType:    readOperationLogBusinessType(c),
			Method:          c.Request.Method,
			OperName:        username,
			OperURL:         c.Request.URL.Path,
			OperIP:          c.ClientIP(),
			SourceDomain:    DetectOperationLogSourceDomain(c.Request.URL.Path),
			SourcePage:      DetectOperationLogSourcePage(c.Request.URL.Path),
			OperParam:       readOperationLogParam(c, requestBody),
			JsonResult:      readOperationLogResult(c, responseBody.String()),
			Status:          status,
			FailureCategory: DetectOperationLogFailureCategory(status, errorMessage, readOperationLogResult(c, responseBody.String())),
			ErrorMsg:        errorMessage,
			OperTime:        start,
			CostTime:        time.Since(start).Milliseconds(),
		}

		go db.Create(&log)
	}
}

func readOperationLogTitle(c *gin.Context) string {
	if value, ok := c.Get(operationLogTitleKey); ok {
		if text, ok := value.(string); ok && strings.TrimSpace(text) != "" {
			return strings.TrimSpace(text)
		}
	}
	return c.FullPath()
}

func readOperationLogBusinessType(c *gin.Context) int {
	if value, ok := c.Get(operationLogBusinessTypeKey); ok {
		switch typed := value.(type) {
		case int:
			return typed
		case int64:
			return int(typed)
		case float64:
			return int(typed)
		case string:
			if parsed, err := strconv.Atoi(strings.TrimSpace(typed)); err == nil {
				return parsed
			}
		}
	}
	return 0
}

func readOperationLogParam(c *gin.Context, fallback string) string {
	if value, ok := c.Get(operationLogParamKey); ok {
		if text, ok := value.(string); ok {
			return text
		}
	}
	return sanitizeJSON(fallback)
}

func readOperationLogResult(c *gin.Context, fallback string) string {
	if value, ok := c.Get(operationLogResultKey); ok {
		if text, ok := value.(string); ok {
			return sanitizeJSON(text)
		}
	}
	return sanitizeJSON(fallback)
}

func readOperationLogStatus(c *gin.Context) (int, bool) {
	if value, ok := c.Get(operationLogStatusKey); ok {
		switch typed := value.(type) {
		case int:
			return typed, true
		case int64:
			return int(typed), true
		case float64:
			return int(typed), true
		case string:
			if parsed, err := strconv.Atoi(strings.TrimSpace(typed)); err == nil {
				return parsed, true
			}
		}
	}
	return 0, false
}

func readOperationLogErrorMsg(c *gin.Context) string {
	if value, ok := c.Get(operationLogErrorMsgKey); ok {
		if text, ok := value.(string); ok {
			return strings.TrimSpace(text)
		}
	}
	return ""
}

func readAndRestoreBody(c *gin.Context) string {
	if c.Request.Body == nil {
		return ""
	}
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return ""
	}
	c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
	return string(body)
}

func sanitizeJSON(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return raw
	}

	var payload map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return raw
	}

	payload = maskSensitivePayload(payload).(map[string]interface{})

	data, err := json.Marshal(payload)
	if err != nil {
		return raw
	}
	return string(data)
}

func maskSensitivePayload(value interface{}) interface{} {
	switch typed := value.(type) {
	case map[string]interface{}:
		for key, item := range typed {
			if isSensitiveLogKey(key) {
				typed[key] = "***"
				continue
			}
			typed[key] = maskSensitivePayload(item)
		}
		return typed
	case []interface{}:
		for index, item := range typed {
			typed[index] = maskSensitivePayload(item)
		}
		return typed
	default:
		return typed
	}
}

func isSensitiveLogKey(key string) bool {
	lowerKey := strings.ToLower(strings.ReplaceAll(key, "_", ""))
	sensitiveTokens := []string{"password", "token", "secret", "accesskey", "apikey", "credential"}
	for _, token := range sensitiveTokens {
		if strings.Contains(lowerKey, token) {
			return true
		}
	}
	return false
}

func parseBusinessResult(raw string) (int, string) {
	var payload struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return 0, ""
	}
	return payload.Code, payload.Message
}

func DetectOperationLogSourceDomain(operURL string) string {
	path := strings.TrimSpace(operURL)
	switch {
	case strings.Contains(path, "/system/setting"), strings.Contains(path, "/system/upload"), strings.Contains(path, "/system/i18n"):
		return "config"
	case strings.Contains(path, "/system/operation-log"):
		return "audit"
	case strings.Contains(path, "/system/login-log"), strings.Contains(path, "/system/session"), strings.Contains(path, "/auth/"):
		return "auth"
	case strings.Contains(path, "/system/user"), strings.Contains(path, "/system/role"), strings.Contains(path, "/system/menu"), strings.Contains(path, "/system/permission"):
		return "iam"
	case strings.Contains(path, "/system/dept"), strings.Contains(path, "/system/post"):
		return "org"
	case strings.Contains(path, "/dashboard"):
		return "platform"
	default:
		return "other"
	}
}

func DetectOperationLogSourcePage(operURL string) string {
	path := strings.TrimSpace(operURL)
	switch {
	case strings.Contains(path, "/system/setting"):
		return "setting"
	case strings.Contains(path, "/system/upload"):
		return "upload"
	case strings.Contains(path, "/system/i18n"):
		return "i18n"
	case strings.Contains(path, "/system/operation-log"):
		return "operationLog"
	case strings.Contains(path, "/system/login-log"):
		return "loginLog"
	case strings.Contains(path, "/system/session"), strings.Contains(path, "/auth/sessions"):
		return "session"
	case strings.Contains(path, "/system/user"):
		return "user"
	case strings.Contains(path, "/system/role"):
		return "role"
	case strings.Contains(path, "/system/menu"):
		return "menu"
	case strings.Contains(path, "/system/permission"):
		return "permission"
	case strings.Contains(path, "/system/dept"):
		return "dept"
	case strings.Contains(path, "/system/post"):
		return "post"
	case strings.Contains(path, "/dashboard"):
		return "dashboard"
	default:
		return "other"
	}
}

func DetectOperationLogFailureCategory(status int, errorMsg string, jsonResult string) string {
	if status != 2 {
		return ""
	}
	errorText := strings.ToLower(strings.TrimSpace(errorMsg) + " " + strings.TrimSpace(jsonResult))
	switch {
	case strings.Contains(errorText, "param.invalid"),
		strings.Contains(errorText, "setting.value."),
		strings.Contains(errorText, "upload.file."),
		strings.Contains(errorText, "\"code\":400"),
		strings.Contains(errorText, `"code": 400`):
		return "validation"
	case strings.Contains(errorText, "permission.denied"),
		strings.Contains(errorText, "\"code\":403"),
		strings.Contains(errorText, `"code": 403`):
		return "permission"
	case strings.Contains(errorText, "refresh_token"),
		strings.Contains(errorText, "auth."),
		strings.Contains(errorText, "login.error"),
		strings.Contains(errorText, "\"code\":401"),
		strings.Contains(errorText, `"code": 401`):
		return "auth"
	case strings.Contains(errorText, "database."),
		strings.Contains(errorText, "\"code\":500"),
		strings.Contains(errorText, `"code": 500`):
		return "server"
	default:
		return "business"
	}
}
