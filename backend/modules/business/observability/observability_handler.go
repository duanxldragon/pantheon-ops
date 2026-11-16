package observability

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/database"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

// ObservabilityHandler handles observability API endpoints.
//
//nolint:revive // the exported handler name keeps the observability domain prefix for API clarity.
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
	h.RegisterDatasourceRoutes(obs)
	h.RegisterLegacyRoutes(obs)
}

// RegisterDatasourceRoutes wires the datasource management endpoints onto obs.
func (h *ObservabilityHandler) RegisterDatasourceRoutes(obs gin.IRoutes) {
	// Datasources
	obs.POST("/datasources", h.CreateDatasource)
	obs.GET("/datasources", h.ListDatasources)
	obs.GET("/datasources/:id", h.GetDatasource)
	obs.PUT("/datasources/:id", h.UpdateDatasource)
	obs.DELETE("/datasources/:id", h.DeleteDatasource)
	obs.POST("/datasources/:id/test", h.TestDatasourceConnection)
	obs.POST("/datasources/:id/query", h.RunQuery)
	obs.POST("/datasources/:id/logs", h.RunQuery)
	obs.POST("/datasources/:id/traces", h.RunQuery)
	obs.GET("/datasources/:id/query", h.DeprecatedQuery)
	obs.GET("/datasources/:id/logs", h.DeprecatedQuery)
	obs.GET("/datasources/:id/traces", h.DeprecatedQuery)
	obs.GET("/datasources/:id/query-fields", h.QueryFields)
	obs.GET("/datasources/:id/traces/:traceId", h.GetTrace)
}

// RegisterWorkspaceRoutes wires the query history and saved-view endpoints onto obs.
func (h *ObservabilityHandler) RegisterWorkspaceRoutes(obs gin.IRoutes) {
	obs.GET("/query-history", h.QueryHistory)
	obs.GET("/query-history/:id", h.QueryHistory)
	obs.DELETE("/query-history", h.DeleteHistory)
	obs.DELETE("/query-history/:id", h.DeleteHistory)
	obs.POST("/query-history/:id/restore", h.QueryHistory)
	obs.GET("/saved-views", h.SavedViews)
	obs.GET("/saved-views/:id", h.SavedViews)
	obs.POST("/saved-views", h.SaveView)
	obs.PUT("/saved-views/:id", h.SaveView)
	obs.DELETE("/saved-views/:id", h.DeleteView)
	obs.POST("/saved-views/:id/restore", h.SavedViews)
}

// RegisterLegacyRoutes wires the legacy alert and notification endpoints onto obs.
func (h *ObservabilityHandler) RegisterLegacyRoutes(obs gin.IRoutes) {
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

// CreateDatasource registers a Prometheus datasource from the request body.
func (h *ObservabilityHandler) CreateDatasource(c *gin.Context) {
	state, bindErr := h.bindQueryRequest(c)
	if bindErr != nil {
		queryFailure(c, bindErr)
		return
	}
	common.SetAuditMetadata(c, "observability.datasource.audit.create", common.BusinessInsert)
	var dto DatasourceDTO
	if err := strictQueryJSON(c.Request.Body, &dto); err != nil {
		common.FailWithCode(c, http.StatusBadRequest, "invalid_request")
		return
	}
	if binding.Validator.ValidateStruct(dto) != nil {
		queryFailure(c, ErrQueryIntent)
		return
	}

	createdBy := strconv.FormatUint(state.owner.ActorID, 10)
	resp, err := state.datasource.Create(dto, createdBy, common.GetDataScope(c))
	if err != nil {
		common.FailWithCode(c, http.StatusInternalServerError, "create_failed")
		return
	}

	if err := state.refresh(); err != nil {
		queryFailure(c, err)
		return
	}
	common.Success(c, resp)
}

// ListDatasources returns the configured Prometheus datasources.
func (h *ObservabilityHandler) ListDatasources(c *gin.Context) {
	state, bindErr := h.bindQueryRequest(c)
	if bindErr != nil {
		queryFailure(c, bindErr)
		return
	}
	if c.Query("dashboard") == "true" {
		h.dashboardSummary(state)
		return
	}
	_, err := state.datasource.List(common.GetDataScope(c))
	if err != nil {
		common.FailWithCode(c, http.StatusInternalServerError, "list_failed")
		return
	}

	if err := state.refresh(); err != nil {
		queryFailure(c, err)
		return
	}
	// A fresh DataScope may have changed during the first read.
	resp, err := state.datasource.List(common.GetDataScope(c))
	if err != nil {
		queryFailure(c, ErrWorkspaceDenied)
		return
	}
	common.Success(c, resp)
}

// GetDatasource returns one Prometheus datasource by id.
func (h *ObservabilityHandler) GetDatasource(c *gin.Context) {
	state, bindErr := h.bindQueryRequest(c)
	if bindErr != nil {
		queryFailure(c, bindErr)
		return
	}
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	_, err := state.datasource.GetByID(id, common.GetDataScope(c))
	if err != nil {
		common.FailWithCode(c, http.StatusNotFound, "not_found")
		return
	}

	if err := state.refresh(); err != nil {
		queryFailure(c, err)
		return
	}
	resp, err := state.datasource.GetByID(id, common.GetDataScope(c))
	if err != nil {
		queryFailure(c, ErrWorkspaceDenied)
		return
	}
	common.Success(c, resp)
}

// UpdateDatasource applies the request body to the datasource identified by the path id.
func (h *ObservabilityHandler) UpdateDatasource(c *gin.Context) {
	state, bindErr := h.bindQueryRequest(c)
	if bindErr != nil {
		queryFailure(c, bindErr)
		return
	}
	common.SetAuditMetadata(c, "observability.datasource.audit.update", common.BusinessUpdate)
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	var dto DatasourceDTO
	if err := strictQueryJSON(c.Request.Body, &dto); err != nil {
		common.FailWithCode(c, http.StatusBadRequest, "invalid_request")
		return
	}
	if binding.Validator.ValidateStruct(dto) != nil {
		queryFailure(c, ErrQueryIntent)
		return
	}

	if err := state.datasource.Update(id, dto, common.GetDataScope(c)); err != nil {
		common.FailWithCode(c, http.StatusInternalServerError, "update_failed")
		return
	}

	if err := state.refresh(); err != nil {
		queryFailure(c, err)
		return
	}
	common.Success(c, gin.H{"message": "updated"})
}

// DeleteDatasource removes the datasource identified by the path id.
func (h *ObservabilityHandler) DeleteDatasource(c *gin.Context) {
	state, bindErr := h.bindQueryRequest(c)
	if bindErr != nil {
		queryFailure(c, bindErr)
		return
	}
	common.SetAuditMetadata(c, "observability.datasource.audit.delete", common.BusinessDelete)
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	if err := state.datasource.Delete(id, common.GetDataScope(c)); err != nil {
		common.FailWithCode(c, http.StatusInternalServerError, "delete_failed")
		return
	}

	if err := state.refresh(); err != nil {
		queryFailure(c, err)
		return
	}
	common.Success(c, gin.H{"message": "deleted"})
}

// TestDatasourceConnection probes the datasource identified by the path id.
func (h *ObservabilityHandler) TestDatasourceConnection(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()
	c.Request = c.Request.WithContext(ctx)
	state, bindErr := h.bindQueryRequest(c)
	if bindErr != nil {
		queryFailure(c, bindErr)
		return
	}
	common.SetAuditMetadata(c, "observability.datasource.audit.test", common.BusinessUpdate)
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	healthy, err := state.datasource.TestConnection(id, common.GetDataScope(c))
	if err != nil {
		common.FailWithCode(c, http.StatusInternalServerError, "test_failed")
		return
	}

	if err := state.refresh(); err != nil {
		queryFailure(c, err)
		return
	}
	common.Success(c, gin.H{"healthy": healthy})
}

// QueryDatasource proxies a PromQL instant query to the data source.
func (h *ObservabilityHandler) QueryDatasource(c *gin.Context) {
	h.DeprecatedQuery(c)
}

// QueryLogs proxies a Loki LogQL range query.
func (h *ObservabilityHandler) QueryLogs(c *gin.Context) {
	h.DeprecatedQuery(c)
}

// SearchTraces proxies a Tempo TraceQL search.
func (h *ObservabilityHandler) SearchTraces(c *gin.Context) {
	h.DeprecatedQuery(c)
}

// GetTrace proxies one Tempo trace lookup.
func (h *ObservabilityHandler) GetTrace(c *gin.Context) {
	state, bindErr := h.bindQueryRequest(c)
	if bindErr != nil {
		queryFailure(c, bindErr)
		return
	}
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)
	if id == 0 {
		common.FailWithCode(c, http.StatusBadRequest, "invalid_request")
		return
	}
	source, checkErr := state.source(id)
	if checkErr != nil {
		queryFailure(c, checkErr)
		return
	}
	result, err := state.datasource.GetTraceContext(c.Request.Context(), id, c.Param("traceId"), common.GetDataScope(c))
	if err != nil {
		common.FailWithCode(c, http.StatusBadGateway, "query_failed")
		return
	}
	if checkErr = state.checkSource(source); checkErr != nil {
		queryFailure(c, checkErr)
		return
	}
	common.Success(c, result)
}

// Alert Rule handlers

// CreateAlertRule registers an alert rule from the request body.
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

// ListAlertRules returns alert rules matching the query.
func (h *ObservabilityHandler) ListAlertRules(c *gin.Context) {
	if c.Query("dashboard") == "true" {
		h.dashboardAlertRules(c)
		return
	}
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

// GetAlertRule returns one alert rule by id.
func (h *ObservabilityHandler) GetAlertRule(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	resp, err := h.alertRuleService.GetByID(id, common.GetDataScope(c))
	if err != nil {
		common.FailWithCode(c, http.StatusNotFound, "not_found")
		return
	}

	common.Success(c, resp)
}

// UpdateAlertRule applies the request body to the alert rule identified by the path id.
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

// DeleteAlertRule removes the alert rule identified by the path id.
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

// ToggleAlertRule flips the enabled state of the alert rule identified by the path id.
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

// ListAlertEvents returns fired alerts matching the query.
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

// GetAlertEvent returns one fired alert by id.
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

// CreateNotificationChannel registers a notification channel from the request body.
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

// ListNotificationChannels returns the configured notification channels.
func (h *ObservabilityHandler) ListNotificationChannels(c *gin.Context) {
	resp, err := h.notificationService.ListChannels(common.GetDataScope(c))
	if err != nil {
		common.FailWithCode(c, http.StatusInternalServerError, "list_failed")
		return
	}

	common.Success(c, resp)
}

// GetNotificationChannel returns one notification channel by id.
func (h *ObservabilityHandler) GetNotificationChannel(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	resp, err := h.notificationService.GetChannelByID(id, common.GetDataScope(c))
	if err != nil {
		common.FailWithCode(c, http.StatusNotFound, "not_found")
		return
	}

	common.Success(c, resp)
}

// UpdateNotificationChannel applies the request body to the channel identified by the path id.
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

// DeleteNotificationChannel removes the channel identified by the path id.
func (h *ObservabilityHandler) DeleteNotificationChannel(c *gin.Context) {
	common.SetAuditMetadata(c, "observability.notification.audit.delete", common.BusinessDelete)
	id, _ := strconv.ParseUint(c.Param("id"), 10, 64)

	if err := h.notificationService.DeleteChannel(id, common.GetDataScope(c)); err != nil {
		common.FailWithCode(c, http.StatusInternalServerError, "delete_failed")
		return
	}

	common.Success(c, gin.H{"message": "deleted"})
}
