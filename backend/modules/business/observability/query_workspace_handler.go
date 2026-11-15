package observability

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/pkg/auditpolicy"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/contracts/businessauth"
	"github.com/duanxldragon/pantheon-base/backend/pkg/database"
	"github.com/duanxldragon/pantheon-base/backend/pkg/tenant"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type queryRequestState struct {
	c          *gin.Context
	input      businessauth.Input
	owner      QueryOwner
	datasource *DatasourceService
	workspace  *QueryWorkspaceService
}

func (h *ObservabilityHandler) bindQueryRequest(c *gin.Context) (*queryRequestState, error) {
	input, err := businessauth.FromGin(c)
	tc := tenant.FromGin(c)
	if err != nil || tc == nil || input.Identity.ActorID == 0 || input.Identity.ActorID != common.GetUserID(c) || input.ResourceTenantID != tc.TenantID || input.Resolution != *tc {
		return nil, ErrWorkspaceDenied
	}
	authorized, err := businessauth.Reauthorize(c, nil)
	if err != nil || authorized.DataScope == nil {
		return nil, ErrWorkspaceDenied
	}
	ds, err := h.datasourceService.WithTenantContext(tc)
	if err != nil {
		return nil, err
	}
	ds.db = ds.db.WithContext(c.Request.Context()).Session(&gorm.Session{Logger: logger.Default.LogMode(logger.Silent)})
	owner := QueryOwner{TenantID: tc.TenantID, ActorID: input.Identity.ActorID}
	workspace, err := NewQueryWorkspaceService(ds.db, owner, authorized.DataScope)
	if err != nil {
		return nil, err
	}
	state := &queryRequestState{c: c, input: input, owner: owner, datasource: ds, workspace: workspace}
	ds.checkpoint = state.checkSource
	auditpolicy.SetMetadata(c, auditpolicy.SafeMetadata{TenantID: owner.TenantID, ActorID: owner.ActorID, CatalogVersion: QueryCatalogVersion})
	return state, nil
}
func (s *queryRequestState) refresh() error {
	authorized, err := businessauth.Reauthorize(s.c, nil)
	if err != nil || authorized.DataScope == nil {
		return ErrWorkspaceDenied
	}
	s.workspace.scope = authorized.DataScope
	return nil
}
func sourceReadAction(kind string, id uint64) (businessauth.Action, error) {
	key, path := "", ""
	switch kind {
	case "prometheus", "victoriametrics":
		key = "metric_source:view"
		path = "/query"
	case "loki":
		key = "logs:view"
		path = "/logs"
	case "tempo":
		key = "traces:view"
		path = "/traces"
	default:
		return businessauth.Action{}, ErrQueryIntent
	}
	return businessauth.Action{Permission: "business:observability:" + key, Method: "GET", Path: observabilityAPIPrefix + "/datasources/" + strconv.FormatUint(id, 10) + path, RouteTemplate: observabilityAPIPrefix + "/datasources/:id" + path}, nil
}
func (s *queryRequestState) checkSource(record *PrometheusDatasource) error {
	if err := s.refresh(); err != nil {
		return err
	}
	if record == nil || record.TenantID != s.owner.TenantID {
		return ErrWorkspaceDenied
	}
	action, err := sourceReadAction(record.Type, record.ID)
	if err != nil {
		return err
	}
	authorized, err := businessauth.Reauthorize(s.c, &action)
	if err != nil || authorized.DataScope == nil {
		return ErrWorkspaceDenied
	}
	s.workspace.scope = authorized.DataScope
	var current PrometheusDatasource
	if err := s.datasource.db.Where("tenant_id = ?", s.owner.TenantID).Scopes(database.WithDataScope(authorized.DataScope)).First(&current, record.ID).Error; err != nil {
		return ErrWorkspaceDenied
	}
	if current.Revision != record.Revision || current.Type != record.Type {
		return ErrWorkspaceConflict
	}
	return nil
}
func queryFailure(c *gin.Context, err error) {
	status, key, outcome := http.StatusInternalServerError, "observability.workspace_unavailable", auditpolicy.OutcomeStorage
	switch {
	case errors.Is(err, ErrWorkspaceDenied), errors.Is(err, gorm.ErrRecordNotFound), errors.Is(err, businessauth.ErrDenied):
		status, key, outcome = http.StatusForbidden, "permission.denied", auditpolicy.OutcomeDenied
	case errors.Is(err, ErrQueryIntent):
		status, key, outcome = http.StatusBadRequest, "param.invalid", auditpolicy.OutcomeInvalid
	case errors.Is(err, ErrWorkspaceConflict):
		status, key, outcome = http.StatusConflict, "observability.workspace_conflict", auditpolicy.OutcomeConflict
	case errors.Is(err, ErrWorkspaceQuota), errors.Is(err, ErrQueryBusy):
		status, key, outcome = http.StatusTooManyRequests, err.Error(), auditpolicy.OutcomeDenied
	}
	auditpolicy.SetOutcome(c, outcome)
	common.FailWithCode(c, status, key)
}
func queryPathID(c *gin.Context) (uint64, error) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		return 0, ErrQueryIntent
	}
	return id, nil
}
func (s *queryRequestState) source(id uint64) (*PrometheusDatasource, error) {
	record, err := s.datasource.datasourceForTypeContext(s.c.Request.Context(), id, s.workspace.scope, "prometheus", "victoriametrics", "loki", "tempo")
	if err != nil {
		return nil, ErrWorkspaceDenied
	}
	if err := s.checkSource(record); err != nil {
		return nil, err
	}
	auditpolicy.SetMetadata(s.c, auditpolicy.SafeMetadata{TenantID: s.owner.TenantID, ActorID: s.owner.ActorID, DatasourceID: id, CatalogVersion: QueryCatalogVersion})
	return record, nil
}
func (h *ObservabilityHandler) QueryFields(c *gin.Context) {
	state, err := h.bindQueryRequest(c)
	if err != nil {
		queryFailure(c, err)
		return
	}
	id, err := queryPathID(c)
	if err != nil {
		queryFailure(c, err)
		return
	}
	source, err := state.source(id)
	if err != nil {
		queryFailure(c, err)
		return
	}
	catalog, err := queryCatalog(source.Type)
	if err == nil {
		err = state.checkSource(source)
	}
	if err != nil {
		queryFailure(c, err)
		return
	}
	common.Success(c, catalog)
}
func (h *ObservabilityHandler) DeprecatedQuery(c *gin.Context) {
	if _, err := h.bindQueryRequest(c); err != nil {
		queryFailure(c, err)
		return
	}
	auditpolicy.SetOutcome(c, auditpolicy.OutcomeInvalid)
	common.FailWithCode(c, http.StatusUpgradeRequired, "observability.query_upgrade_required")
}
func (h *ObservabilityHandler) RunQuery(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second)
	defer cancel()
	c.Request = c.Request.WithContext(ctx)
	state, err := h.bindQueryRequest(c)
	if err != nil {
		queryFailure(c, err)
		return
	}
	id, err := queryPathID(c)
	if err != nil {
		queryFailure(c, err)
		return
	}
	source, err := state.source(id)
	if err != nil {
		queryFailure(c, err)
		return
	}
	// Prevent the metric/log/trace action from executing a different source kind.
	suffix := c.FullPath()
	if (strings.HasSuffix(suffix, "/query") && source.Type != "prometheus" && source.Type != "victoriametrics") ||
		(strings.HasSuffix(suffix, "/logs") && source.Type != "loki") ||
		(strings.HasSuffix(suffix, "/traces") && source.Type != "tempo") {
		queryFailure(c, ErrQueryIntent)
		return
	}
	request, compiled, intent, limit, mode, err := decodeRunQuery(c.Request.Body, source.Type, time.Now().UTC())
	if err != nil {
		queryFailure(c, err)
		return
	}
	permit, err := AcquireQueryPermit(ctx, database.RDB, state.owner)
	if err != nil {
		queryFailure(c, err)
		return
	}
	defer func() { _ = permit.Release() }()
	if err := state.checkSource(source); err != nil {
		queryFailure(c, err)
		return
	}
	started := time.Now()
	var result map[string]interface{}
	switch source.Type {
	case "prometheus", "victoriametrics":
		result, err = state.datasource.QueryContext(ctx, id, compiled.Expression, state.workspace.scope)
	case "loki":
		result, err = state.datasource.QueryLogsContext(ctx, id, compiled.Expression, compiled.Start.Format(time.RFC3339Nano), compiled.End.Format(time.RFC3339Nano), limit, state.workspace.scope)
	case "tempo":
		result, err = state.datasource.SearchTracesContext(ctx, id, compiled.Expression, compiled.Start.Format(time.RFC3339Nano), compiled.End.Format(time.RFC3339Nano), limit, state.workspace.scope)
	}
	if releaseErr := state.checkSource(source); releaseErr != nil {
		queryFailure(c, releaseErr)
		return
	}
	outcome := "success"
	if err != nil {
		outcome = "upstream"
		if ctx.Err() != nil {
			outcome = "cancelled"
		}
	}
	duration := time.Since(started).Milliseconds()
	if duration > 30000 {
		duration = 30000
	}
	history, storageErr := state.workspace.AppendHistory(ctx, HistoryInput{HistoryMode: request.HistoryMode, HistoryID: request.HistoryID, DatasourceID: id, SourceRevision: source.Revision, SourceKind: source.Type, Time: intent, Limit: limit, ResultMode: mode, Outcome: outcome, DurationMS: uint32(duration), Recipe: request.Recipe})
	// Every slow storage operation has a fresh response-release checkpoint too.
	if releaseErr := state.checkSource(source); releaseErr != nil {
		queryFailure(c, releaseErr)
		return
	}
	if err != nil {
		auditpolicy.SetOutcome(c, auditpolicy.OutcomeUpstream)
		common.FailWithCode(c, http.StatusBadGateway, "observability.query_unavailable")
		return
	}
	capQueryRows(result, limit)
	response := gin.H{"result": result, "historyUnavailable": storageErr != nil}
	if request.HistoryMode == "coalesced" && storageErr == nil {
		response["historyId"] = history.ID
	}
	common.Success(c, response)
}
func capQueryRows(result map[string]interface{}, limit int) {
	if data, ok := result["data"].(map[string]interface{}); ok {
		if rows, ok := data["result"].([]interface{}); ok {
			if len(rows) > limit {
				rows = rows[:limit]
				data["result"] = rows
			}
			left := limit
			for _, row := range rows {
				if stream, ok := row.(map[string]interface{}); ok {
					if values, ok := stream["values"].([]interface{}); ok {
						n := len(values)
						if n > left {
							n = left
						}
						stream["values"] = values[:n]
						left -= n
					}
				}
			}
		}
	}
	if rows, ok := result["traces"].([]interface{}); ok && len(rows) > limit {
		result["traces"] = rows[:limit]
	}
}
func (h *ObservabilityHandler) QueryHistory(c *gin.Context) { h.readWorkspace(c, false) }
func (h *ObservabilityHandler) SavedViews(c *gin.Context)   { h.readWorkspace(c, true) }
func (h *ObservabilityHandler) readWorkspace(c *gin.Context, view bool) {
	state, err := h.bindQueryRequest(c)
	if err != nil {
		queryFailure(c, err)
		return
	}
	if c.Request.Method == http.MethodPost && c.Request.ContentLength != 0 {
		var empty struct{}
		if err := strictQueryJSON(c.Request.Body, &empty); err != nil {
			queryFailure(c, err)
			return
		}
	}
	var id uint64
	if c.Param("id") != "" {
		id, err = queryPathID(c)
		if err != nil {
			queryFailure(c, err)
			return
		}
	}
	page, _ := strconv.Atoi(c.Query("page"))
	size, _ := strconv.Atoi(c.Query("pageSize"))
	var items any
	var total int64
	sources := map[uint64]*PrometheusDatasource{}
	if view {
		var rows []SavedView
		rows, total, err = state.workspace.Views(c.Request.Context(), id, page, size)
		dtos := make([]SavedViewDTO, 0, len(rows))
		if err == nil {
			for _, row := range rows {
				source, sourceErr := state.source(row.DatasourceID)
				if sourceErr != nil {
					err = sourceErr
					break
				}
				sources[source.ID] = source
				recipe, decodeErr := DecodeQueryRecipe(bytes.NewReader(row.Recipe))
				if decodeErr == nil {
					_, decodeErr = CompileQueryRecipe(recipe, source.Type, time.Now().UTC())
				}
				if decodeErr != nil {
					err = decodeErr
					break
				}
				dtos = append(dtos, SavedViewDTO{ID: row.ID, DatasourceID: row.DatasourceID, Revision: row.Revision, Recipe: recipe, UpdatedAt: row.UpdatedAt})
			}
		}
		items = dtos
	} else {
		var rows []QueryHistory
		rows, total, err = state.workspace.History(c.Request.Context(), id, page, size)
		dtos := make([]HistoryDTO, 0, len(rows))
		if err == nil {
			for _, row := range rows {
				source, sourceErr := state.source(row.DatasourceID)
				if sourceErr != nil {
					err = sourceErr
					break
				}
				sources[source.ID] = source
				dto := HistoryDTO{ID: row.ID, DatasourceID: row.DatasourceID, SourceKind: row.SourceKind, Limit: row.Limit, ResultMode: row.ResultMode, Outcome: row.Outcome, DurationMS: row.DurationMS, CreatedAt: row.CreatedAt, ExpressionUnavailable: len(row.Recipe) == 0}
				if json.Unmarshal(row.Time, &dto.Time) != nil {
					err = ErrQueryIntent
					break
				}
				if _, _, timeErr := queryTimeBounds(dto.Time, time.Now().UTC()); timeErr != nil {
					err = timeErr
					break
				}
				if len(row.Recipe) > 0 {
					recipe, decodeErr := DecodeQueryRecipe(bytes.NewReader(row.Recipe))
					if decodeErr == nil {
						_, decodeErr = CompileQueryRecipe(recipe, source.Type, time.Now().UTC())
					}
					if decodeErr != nil {
						err = decodeErr
						break
					}
					dto.Recipe = &recipe
				}
				dtos = append(dtos, dto)
			}
		}
		items = dtos
	}
	if err == nil {
		err = state.refresh()
	}
	if err == nil {
		for _, source := range sources {
			if err = state.checkSource(source); err != nil {
				break
			}
		}
	}
	if err != nil {
		queryFailure(c, err)
		return
	}
	// Restore deliberately returns intent only; it never touches the provider.
	common.Success(c, gin.H{"items": items, "total": total, "catalogVersion": QueryCatalogVersion})
}
func (h *ObservabilityHandler) SaveView(c *gin.Context) {
	state, err := h.bindQueryRequest(c)
	if err != nil {
		queryFailure(c, err)
		return
	}
	var request SaveViewRequest
	if err := strictQueryJSON(c.Request.Body, &request); err != nil {
		queryFailure(c, err)
		return
	}
	var id uint64
	if c.Param("id") != "" {
		id, err = queryPathID(c)
		if err != nil {
			queryFailure(c, err)
			return
		}
	}
	if request.DatasourceID == 0 || (id == 0 && request.ExpectedRevision != 0) || (id != 0 && request.ExpectedRevision == 0) {
		queryFailure(c, ErrQueryIntent)
		return
	}
	source, err := state.source(request.DatasourceID)
	if err != nil {
		queryFailure(c, err)
		return
	}
	row, err := state.workspace.Save(c.Request.Context(), source.ID, id, request.ExpectedRevision, request.Recipe)
	if err == nil {
		err = state.checkSource(source)
	}
	if err != nil {
		queryFailure(c, err)
		return
	}
	recipe, err := DecodeQueryRecipe(bytes.NewReader(row.Recipe))
	if err != nil {
		queryFailure(c, err)
		return
	}
	auditpolicy.SetMetadata(c, auditpolicy.SafeMetadata{TenantID: state.owner.TenantID, ActorID: state.owner.ActorID, DatasourceID: source.ID, ViewID: row.ID, CatalogVersion: QueryCatalogVersion})
	common.Success(c, SavedViewDTO{ID: row.ID, DatasourceID: row.DatasourceID, Revision: row.Revision, Recipe: recipe, UpdatedAt: row.UpdatedAt})
}
func (h *ObservabilityHandler) DeleteHistory(c *gin.Context) { h.deleteWorkspace(c, false) }
func (h *ObservabilityHandler) DeleteView(c *gin.Context)    { h.deleteWorkspace(c, true) }
func (h *ObservabilityHandler) deleteWorkspace(c *gin.Context, view bool) {
	state, err := h.bindQueryRequest(c)
	if err != nil {
		queryFailure(c, err)
		return
	}
	var id uint64
	if c.Param("id") != "" {
		id, err = queryPathID(c)
		if err != nil {
			queryFailure(c, err)
			return
		}
	}
	err = state.workspace.Delete(c.Request.Context(), view, id)
	if err == nil {
		err = state.refresh()
	}
	if err != nil {
		queryFailure(c, err)
		return
	}
	common.Success(c, gin.H{"deleted": true})
}
