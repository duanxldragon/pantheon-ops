package observability

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/database"

	"github.com/gin-gonic/gin"
)

// ObservabilityHandler handles observability API endpoints.
type ObservabilityHandler struct {
	datasourceService   *DatasourceService
	alertRuleService    *AlertRuleService
	notificationService *NotificationService
	alertEvaluator      *AlertEvaluator
}

// NewObservabilityHandler creates a new ObservabilityHandler.
func NewObservabilityHandler(
	datasourceService *DatasourceService,
	alertRuleService *AlertRuleService,
	notificationService *NotificationService,
	alertEvaluator *AlertEvaluator,
) *ObservabilityHandler {
	return &ObservabilityHandler{
		datasourceService:   datasourceService,
		alertRuleService:    alertRuleService,
		notificationService: notificationService,
		alertEvaluator:      alertEvaluator,
	}
}

// RegisterRoutes registers all observability routes. Both groups must be
// DataScopedGroups (token + Casbin + data-scope middleware); the handler never
// registers on a raw router group.
func (h *ObservabilityHandler) RegisterRoutes(canonical, alias gin.IRoutes) {
	h.registerRoutes(canonical)
	// The business-prefixed path is the canonical overlay contract used by the
	// frontend. Keep the original path as a compatibility alias for V1 clients.
	h.registerRoutes(alias)
}

func (h *ObservabilityHandler) registerRoutes(obs gin.IRoutes) {
	// Datasources
	obs.POST("/datasources", h.CreateDatasource)
	obs.GET("/datasources", h.ListDatasources)
	obs.GET("/datasources/:id", h.GetDatasource)
	obs.PUT("/datasources/:id", h.UpdateDatasource)
	obs.DELETE("/datasources/:id", h.DeleteDatasource)
	obs.POST("/datasources/:id/test", h.TestDatasourceConnection)
	obs.GET("/datasources/:id/query", h.QueryDatasource)
	obs.GET("/datasources/:id/logs", h.QueryLogs)
	obs.GET("/datasources/:id/traces", h.SearchTraces)
	obs.GET("/datasources/:id/traces/:traceId", h.GetTrace)

	// Alert Rules
	obs.POST("/alert-rules", h.CreateAlertRule)
	obs.GET("/alert-rules", h.ListAlertRules)
	obs.GET("/alert-rules/:id", h.GetAlertRule)
	obs.PUT("/alert-rules/:id", h.UpdateAlertRule)
	obs.DELETE("/alert-rules/:id", h.DeleteAlertRule)
	obs.PATCH("/alert-rules/:id/toggle", h.ToggleAlertRule)

	// Alert Events
	obs.GET("/alert-events", h.ListAlertEvents)
	obs.GET("/alert-events/:id", h.GetAlertEvent)

	// Notification Channels
	obs.POST("/notification-channels", h.CreateNotificationChannel)
	obs.GET("/notification-channels", h.ListNotificationChannels)
	obs.GET("/notification-channels/:id", h.GetNotificationChannel)
	obs.PUT("/notification-channels/:id", h.UpdateNotificationChannel)
	obs.DELETE("/notification-channels/:id", h.DeleteNotificationChannel)
}

// Datasource handlers

func (h *ObservabilityHandler) CreateDatasource(c *gin.Context) {
	common.SetAuditMetadata(c, "observability.datasource.audit.create", common.BusinessInsert)
	var dto DatasourceDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		common.FailWithCode(c, http.StatusBadRequest, "invalid_request")
		return
	}

	createdBy := c.GetString("username")
	resp, err := h.datasourceService.Create(dto, createdBy, common.GetDataScope(c))
	if err != nil {
		common.FailWithCode(c, http.StatusInternalServerError, "create_failed")
		return
	}

	common.Success(c, resp)
}

func (h *ObservabilityHandler) ListDatasources(c *gin.Context) {
	resp, err := h.datasourceService.List(common.GetDataScope(c))
	if err != nil {
		common.FailWithCode(c, http.StatusInternalServerError, "list_failed")
		return
	}

	common.Success(c, resp)
}

func (h *ObservabilityHandler) GetDatasource(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	resp, err := h.datasourceService.GetByID(id, common.GetDataScope(c))
	if err != nil {
		common.FailWithCode(c, http.StatusNotFound, "not_found")
		return
	}

	common.Success(c, resp)
}

func (h *ObservabilityHandler) UpdateDatasource(c *gin.Context) {
	common.SetAuditMetadata(c, "observability.datasource.audit.update", common.BusinessUpdate)
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	var dto DatasourceDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		common.FailWithCode(c, http.StatusBadRequest, "invalid_request")
		return
	}

	if err := h.datasourceService.Update(id, dto, common.GetDataScope(c)); err != nil {
		common.FailWithCode(c, http.StatusInternalServerError, "update_failed")
		return
	}

	common.Success(c, gin.H{"message": "updated"})
}

func (h *ObservabilityHandler) DeleteDatasource(c *gin.Context) {
	common.SetAuditMetadata(c, "observability.datasource.audit.delete", common.BusinessDelete)
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	if err := h.datasourceService.Delete(id, common.GetDataScope(c)); err != nil {
		common.FailWithCode(c, http.StatusInternalServerError, "delete_failed")
		return
	}

	common.Success(c, gin.H{"message": "deleted"})
}

func (h *ObservabilityHandler) TestDatasourceConnection(c *gin.Context) {
	common.SetAuditMetadata(c, "observability.datasource.audit.test", common.BusinessUpdate)
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	healthy, err := h.datasourceService.TestConnection(id, common.GetDataScope(c))
	if err != nil {
		common.FailWithCode(c, http.StatusInternalServerError, "test_failed")
		return
	}

	common.Success(c, gin.H{"healthy": healthy})
}

// QueryDatasource proxies a PromQL instant query to the data source.
func (h *ObservabilityHandler) QueryDatasource(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	query := c.Query("query")
	if query == "" {
		common.FailWithCode(c, http.StatusBadRequest, "invalid_request")
		return
	}

	result, err := h.datasourceService.Query(id, query, common.GetDataScope(c))
	if err != nil {
		common.FailWithCode(c, http.StatusInternalServerError, "query_failed")
		return
	}

	common.Success(c, result)
}

// QueryLogs proxies a Loki LogQL range query.
func (h *ObservabilityHandler) QueryLogs(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	query := strings.TrimSpace(c.Query("query"))
	if id == 0 || query == "" {
		common.FailWithCode(c, http.StatusBadRequest, "invalid_request")
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "500"))
	result, err := h.datasourceService.QueryLogs(id, query, c.Query("start"), c.Query("end"), limit, common.GetDataScope(c))
	if err != nil {
		common.FailWithCode(c, http.StatusBadGateway, "query_failed")
		return
	}
	common.Success(c, result)
}

// SearchTraces proxies a Tempo TraceQL search.
func (h *ObservabilityHandler) SearchTraces(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	query := strings.TrimSpace(c.Query("query"))
	if id == 0 || query == "" {
		common.FailWithCode(c, http.StatusBadRequest, "invalid_request")
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	result, err := h.datasourceService.SearchTraces(id, query, c.Query("start"), c.Query("end"), limit, common.GetDataScope(c))
	if err != nil {
		common.FailWithCode(c, http.StatusBadGateway, "query_failed")
		return
	}
	common.Success(c, result)
}

// GetTrace proxies one Tempo trace lookup.
func (h *ObservabilityHandler) GetTrace(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if id == 0 {
		common.FailWithCode(c, http.StatusBadRequest, "invalid_request")
		return
	}
	result, err := h.datasourceService.GetTrace(id, c.Param("traceId"), common.GetDataScope(c))
	if err != nil {
		common.FailWithCode(c, http.StatusBadGateway, "query_failed")
		return
	}
	common.Success(c, result)
}

// Alert Rule handlers

func (h *ObservabilityHandler) CreateAlertRule(c *gin.Context) {
	common.SetAuditMetadata(c, "observability.alert_rule.audit.create", common.BusinessInsert)
	var dto AlertRuleDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		common.FailWithCode(c, http.StatusBadRequest, "invalid_request")
		return
	}

	createdBy := c.GetString("username")
	resp, err := h.alertRuleService.Create(dto, createdBy, common.GetDataScope(c))
	if err != nil {
		common.FailWithCode(c, http.StatusInternalServerError, "create_failed")
		return
	}

	// Add rule to evaluator
	var rule AlertRule
	if err := h.alertRuleService.db.First(&rule, resp.ID).Error; err == nil {
		h.alertEvaluator.AddRule(&rule)
	}

	common.Success(c, resp)
}

func (h *ObservabilityHandler) ListAlertRules(c *gin.Context) {
	var query AlertRuleListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.FailWithCode(c, http.StatusBadRequest, "invalid_request")
		return
	}

	resp, err := h.alertRuleService.List(query, common.GetDataScope(c))
	if err != nil {
		common.FailWithCode(c, http.StatusInternalServerError, "list_failed")
		return
	}

	common.Success(c, resp)
}

func (h *ObservabilityHandler) GetAlertRule(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	resp, err := h.alertRuleService.GetByID(id, common.GetDataScope(c))
	if err != nil {
		common.FailWithCode(c, http.StatusNotFound, "not_found")
		return
	}

	common.Success(c, resp)
}

func (h *ObservabilityHandler) UpdateAlertRule(c *gin.Context) {
	common.SetAuditMetadata(c, "observability.alert_rule.audit.update", common.BusinessUpdate)
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	var dto AlertRuleDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		common.FailWithCode(c, http.StatusBadRequest, "invalid_request")
		return
	}

	if err := h.alertRuleService.Update(id, dto, common.GetDataScope(c)); err != nil {
		common.FailWithCode(c, http.StatusInternalServerError, "update_failed")
		return
	}

	// Reload rule in evaluator
	var rule AlertRule
	if err := h.alertRuleService.db.First(&rule, id).Error; err == nil {
		h.alertEvaluator.RemoveRule(id)
		if rule.Enabled {
			h.alertEvaluator.AddRule(&rule)
		}
	}

	common.Success(c, gin.H{"message": "updated"})
}

func (h *ObservabilityHandler) DeleteAlertRule(c *gin.Context) {
	common.SetAuditMetadata(c, "observability.alert_rule.audit.delete", common.BusinessDelete)
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	if err := h.alertRuleService.Delete(id, common.GetDataScope(c)); err != nil {
		common.FailWithCode(c, http.StatusInternalServerError, "delete_failed")
		return
	}
	h.alertEvaluator.RemoveRule(id)

	common.Success(c, gin.H{"message": "deleted"})
}

func (h *ObservabilityHandler) ToggleAlertRule(c *gin.Context) {
	common.SetAuditMetadata(c, "observability.alert_rule.audit.toggle", common.BusinessUpdate)
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	var body struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		common.FailWithCode(c, http.StatusBadRequest, "invalid_request")
		return
	}

	if err := h.alertRuleService.ToggleEnabled(id, body.Enabled, common.GetDataScope(c)); err != nil {
		common.FailWithCode(c, http.StatusInternalServerError, "update_failed")
		return
	}

	// Update evaluator
	var rule AlertRule
	if err := h.alertRuleService.db.First(&rule, id).Error; err == nil {
		h.alertEvaluator.RemoveRule(id)
		if body.Enabled {
			h.alertEvaluator.AddRule(&rule)
		}
	}

	common.Success(c, gin.H{"message": "updated"})
}

// Alert Event handlers

func (h *ObservabilityHandler) ListAlertEvents(c *gin.Context) {
	var query AlertEventListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.FailWithCode(c, http.StatusBadRequest, "invalid_request")
		return
	}

	resp, err := h.alertRuleService.ListEvents(query, common.GetDataScope(c))
	if err != nil {
		common.FailWithCode(c, http.StatusInternalServerError, "list_failed")
		return
	}

	common.Success(c, resp)
}

func (h *ObservabilityHandler) GetAlertEvent(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	var event AlertEvent
	if err := h.alertRuleService.db.Scopes(database.WithDataScope(common.GetDataScope(c))).First(&event, id).Error; err != nil {
		common.FailWithCode(c, http.StatusNotFound, "not_found")
		return
	}

	resp := h.alertRuleService.toEventResponse(&event)
	common.Success(c, resp)
}

// Notification Channel handlers

func (h *ObservabilityHandler) CreateNotificationChannel(c *gin.Context) {
	common.SetAuditMetadata(c, "observability.notification.audit.create", common.BusinessInsert)
	var dto NotificationChannelDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		common.FailWithCode(c, http.StatusBadRequest, "invalid_request")
		return
	}

	createdBy := c.GetString("username")
	resp, err := h.notificationService.CreateChannel(dto, createdBy, common.GetDataScope(c))
	if err != nil {
		common.FailWithCode(c, http.StatusInternalServerError, "create_failed")
		return
	}

	common.Success(c, resp)
}

func (h *ObservabilityHandler) ListNotificationChannels(c *gin.Context) {
	resp, err := h.notificationService.ListChannels(common.GetDataScope(c))
	if err != nil {
		common.FailWithCode(c, http.StatusInternalServerError, "list_failed")
		return
	}

	common.Success(c, resp)
}

func (h *ObservabilityHandler) GetNotificationChannel(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	resp, err := h.notificationService.GetChannelByID(id, common.GetDataScope(c))
	if err != nil {
		common.FailWithCode(c, http.StatusNotFound, "not_found")
		return
	}

	common.Success(c, resp)
}

func (h *ObservabilityHandler) UpdateNotificationChannel(c *gin.Context) {
	common.SetAuditMetadata(c, "observability.notification.audit.update", common.BusinessUpdate)
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	var dto NotificationChannelDTO
	if err := c.ShouldBindJSON(&dto); err != nil {
		common.FailWithCode(c, http.StatusBadRequest, "invalid_request")
		return
	}

	if err := h.notificationService.UpdateChannel(id, dto, common.GetDataScope(c)); err != nil {
		common.FailWithCode(c, http.StatusInternalServerError, "update_failed")
		return
	}

	common.Success(c, gin.H{"message": "updated"})
}

func (h *ObservabilityHandler) DeleteNotificationChannel(c *gin.Context) {
	common.SetAuditMetadata(c, "observability.notification.audit.delete", common.BusinessDelete)
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	if err := h.notificationService.DeleteChannel(id, common.GetDataScope(c)); err != nil {
		common.FailWithCode(c, http.StatusInternalServerError, "delete_failed")
		return
	}

	common.Success(c, gin.H{"message": "deleted"})
}
