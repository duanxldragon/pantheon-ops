package observability

import (
	"strconv"

	"pantheon-base/pkg/common"

	"github.com/gin-gonic/gin"
)

// Handler handles HTTP requests for observability module.
type Handler struct {
	service *Service
}

// NewHandler creates a new observability handler.
func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

// PaginatedResponse represents a paginated response
type PaginatedResponse struct {
	Data     interface{} `json:"data"`
	Total    int64       `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"pageSize"`
}

// CreateMetricSource handles POST /api/v1/observability/metrics/sources
func (h *Handler) CreateMetricSource(c *gin.Context) {
	var source MetricSource
	if err := c.ShouldBindJSON(&source); err != nil {
		common.Fail(c, common.CodeParamInvalid, err.Error())
		return
	}
	if err := h.service.CreateMetricSource(&source); err != nil {
		common.Fail(c, common.CodeError, err.Error())
		return
	}
	common.Success(c, source)
}

// GetMetricSource handles GET /api/v1/observability/metrics/sources/:id
func (h *Handler) GetMetricSource(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, "Invalid ID")
		return
	}
	source, err := h.service.GetMetricSource(id)
	if err != nil {
		common.Fail(c, common.CodeNotFound, "Not found")
		return
	}
	common.Success(c, source)
}

// ListMetricSources handles GET /api/v1/observability/metrics/sources
func (h *Handler) ListMetricSources(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	filters := make(map[string]interface{})
	if id := c.Query("businessScopeId"); id != "" {
		if v, err := strconv.ParseUint(id, 10, 64); err == nil {
			filters["business_scope_id"] = v
		}
	}
	if v := c.Query("status"); v != "" {
		filters["status"] = v
	}
	if v := c.Query("type"); v != "" {
		filters["type"] = v
	}

	sources, total, err := h.service.ListMetricSources(page, pageSize, filters)
	if err != nil {
		common.Fail(c, common.CodeError, err.Error())
		return
	}
	common.Success(c, PaginatedResponse{Data: sources, Total: total, Page: page, PageSize: pageSize})
}

// UpdateMetricSource handles PUT /api/v1/observability/metrics/sources/:id
func (h *Handler) UpdateMetricSource(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, "Invalid ID")
		return
	}
	var source MetricSource
	if err := c.ShouldBindJSON(&source); err != nil {
		common.Fail(c, common.CodeParamInvalid, err.Error())
		return
	}
	source.ID = id
	if err := h.service.UpdateMetricSource(&source); err != nil {
		common.Fail(c, common.CodeError, err.Error())
		return
	}
	common.Success(c, source)
}

// DeleteMetricSource handles DELETE /api/v1/observability/metrics/sources/:id
func (h *Handler) DeleteMetricSource(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, "Invalid ID")
		return
	}
	if err := h.service.DeleteMetricSource(id); err != nil {
		common.Fail(c, common.CodeError, err.Error())
		return
	}
	common.Success(c, gin.H{"message": "Deleted successfully"})
}

// CreateAlertRule handles POST /api/v1/observability/alerts/rules
func (h *Handler) CreateAlertRule(c *gin.Context) {
	var rule AlertRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		common.Fail(c, common.CodeParamInvalid, err.Error())
		return
	}
	if err := h.service.CreateAlertRule(&rule); err != nil {
		common.Fail(c, common.CodeError, err.Error())
		return
	}
	common.Success(c, rule)
}

// GetAlertRule handles GET /api/v1/observability/alerts/rules/:id
func (h *Handler) GetAlertRule(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, "Invalid ID")
		return
	}
	rule, err := h.service.GetAlertRule(id)
	if err != nil {
		common.Fail(c, common.CodeNotFound, "Not found")
		return
	}
	common.Success(c, rule)
}

// ListAlertRules handles GET /api/v1/observability/alerts/rules
func (h *Handler) ListAlertRules(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	filters := make(map[string]interface{})
	if id := c.Query("businessScopeId"); id != "" {
		if v, err := strconv.ParseUint(id, 10, 64); err == nil {
			filters["business_scope_id"] = v
		}
	}
	if v := c.Query("environment"); v != "" {
		filters["environment"] = v
	}
	if v := c.Query("severity"); v != "" {
		filters["severity"] = v
	}
	if v := c.Query("status"); v != "" {
		filters["status"] = v
	}

	rules, total, err := h.service.ListAlertRules(page, pageSize, filters)
	if err != nil {
		common.Fail(c, common.CodeError, err.Error())
		return
	}
	common.Success(c, PaginatedResponse{Data: rules, Total: total, Page: page, PageSize: pageSize})
}

// UpdateAlertRule handles PUT /api/v1/observability/alerts/rules/:id
func (h *Handler) UpdateAlertRule(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, "Invalid ID")
		return
	}
	var rule AlertRule
	if err := c.ShouldBindJSON(&rule); err != nil {
		common.Fail(c, common.CodeParamInvalid, err.Error())
		return
	}
	rule.ID = id
	if err := h.service.UpdateAlertRule(&rule); err != nil {
		common.Fail(c, common.CodeError, err.Error())
		return
	}
	common.Success(c, rule)
}

// DeleteAlertRule handles DELETE /api/v1/observability/alerts/rules/:id
func (h *Handler) DeleteAlertRule(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, "Invalid ID")
		return
	}
	if err := h.service.DeleteAlertRule(id); err != nil {
		common.Fail(c, common.CodeError, err.Error())
		return
	}
	common.Success(c, gin.H{"message": "Deleted successfully"})
}

// ValidatePromQL handles POST /api/v1/observability/alerts/rules/validate
func (h *Handler) ValidatePromQL(c *gin.Context) {
	var req struct {
		PromQL string `json:"promql" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, err.Error())
		return
	}
	if err := h.service.ValidatePromQL(req.PromQL); err != nil {
		common.Fail(c, common.CodeParamInvalid, "Invalid PromQL")
		return
	}
	common.Success(c, gin.H{"valid": true})
}

// CreateNotificationChannel handles POST /api/v1/observability/alerts/channels
func (h *Handler) CreateNotificationChannel(c *gin.Context) {
	var channel NotificationChannel
	if err := c.ShouldBindJSON(&channel); err != nil {
		common.Fail(c, common.CodeParamInvalid, err.Error())
		return
	}
	if err := h.service.CreateNotificationChannel(&channel); err != nil {
		common.Fail(c, common.CodeError, err.Error())
		return
	}
	common.Success(c, channel)
}

// GetNotificationChannel handles GET /api/v1/observability/alerts/channels/:id
func (h *Handler) GetNotificationChannel(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, "Invalid ID")
		return
	}
	channel, err := h.service.GetNotificationChannel(id)
	if err != nil {
		common.Fail(c, common.CodeNotFound, "Not found")
		return
	}
	common.Success(c, channel)
}

// ListNotificationChannels handles GET /api/v1/observability/alerts/channels
func (h *Handler) ListNotificationChannels(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	filters := make(map[string]interface{})
	if id := c.Query("businessScopeId"); id != "" {
		if v, err := strconv.ParseUint(id, 10, 64); err == nil {
			filters["business_scope_id"] = v
		}
	}
	if v := c.Query("type"); v != "" {
		filters["type"] = v
	}
	if v := c.Query("status"); v != "" {
		filters["status"] = v
	}

	channels, total, err := h.service.ListNotificationChannels(page, pageSize, filters)
	if err != nil {
		common.Fail(c, common.CodeError, err.Error())
		return
	}
	common.Success(c, PaginatedResponse{Data: channels, Total: total, Page: page, PageSize: pageSize})
}

// UpdateNotificationChannel handles PUT /api/v1/observability/alerts/channels/:id
func (h *Handler) UpdateNotificationChannel(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, "Invalid ID")
		return
	}
	var channel NotificationChannel
	if err := c.ShouldBindJSON(&channel); err != nil {
		common.Fail(c, common.CodeParamInvalid, err.Error())
		return
	}
	channel.ID = id
	if err := h.service.UpdateNotificationChannel(&channel); err != nil {
		common.Fail(c, common.CodeError, err.Error())
		return
	}
	common.Success(c, channel)
}

// DeleteNotificationChannel handles DELETE /api/v1/observability/alerts/channels/:id
func (h *Handler) DeleteNotificationChannel(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, "Invalid ID")
		return
	}
	if err := h.service.DeleteNotificationChannel(id); err != nil {
		common.Fail(c, common.CodeError, err.Error())
		return
	}
	common.Success(c, gin.H{"message": "Deleted successfully"})
}

// TestNotificationChannel handles POST /api/v1/observability/alerts/channels/:id/test
func (h *Handler) TestNotificationChannel(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, "Invalid ID")
		return
	}
	if err := h.service.TestNotificationChannel(id); err != nil {
		common.Fail(c, common.CodeError, err.Error())
		return
	}
	common.Success(c, gin.H{"message": "Test sent"})
}

// ListAlertRecords handles GET /api/v1/observability/alerts/records
func (h *Handler) ListAlertRecords(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	filters := make(map[string]interface{})
	if id := c.Query("alertRuleId"); id != "" {
		if v, err := strconv.ParseUint(id, 10, 64); err == nil {
			filters["alert_rule_id"] = v
		}
	}
	if v := c.Query("severity"); v != "" {
		filters["severity"] = v
	}
	if v := c.Query("resolved"); v != "" {
		filters["resolved"] = v == "true"
	}

	records, total, err := h.service.repo.ListAlertRecords(page, pageSize, filters)
	if err != nil {
		common.Fail(c, common.CodeError, err.Error())
		return
	}
	common.Success(c, PaginatedResponse{Data: records, Total: total, Page: page, PageSize: pageSize})
}

// GetActiveAlerts handles GET /api/v1/observability/alerts/active
func (h *Handler) GetActiveAlerts(c *gin.Context) {
	filters := map[string]interface{}{"resolved": false}
	records, total, err := h.service.repo.ListAlertRecords(1, 100, filters)
	if err != nil {
		common.Fail(c, common.CodeError, err.Error())
		return
	}
	common.Success(c, PaginatedResponse{Data: records, Total: total, Page: 1, PageSize: 100})
}
