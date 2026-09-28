package system

import (
	"strconv"

	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/impexp"
	"github.com/duanxldragon/pantheon-base/backend/pkg/tenant"

	"github.com/gin-gonic/gin"
)

const errRequestFailed = "request.failed"
const errParamInvalid = "param.invalid"

type AuditHandler struct {
	service *AuditService
}

func NewAuditHandler(s *AuditService) *AuditHandler {
	return &AuditHandler{service: s}
}

// requestTenantContext returns the resolved tenant context for the request
// (queue-5 audit slice). Under compat it is nil — the service then applies no
// filter, preserving single-tenant behavior byte-for-byte.
func (h *AuditHandler) requestTenantContext(c *gin.Context) *tenant.Context {
	return tenant.FromGin(c)
}

// enforceTenantQueryBoundary rejects tenantIdFilter from subjects that are not
// platform-global (contract §7): a tenant subject must never be able to widen
// its own scope, and a forged filter value must not leak another tenant's
// audit trail. Platform-global subjects keep cross-tenant query capability.
func (h *AuditHandler) enforceTenantQueryBoundary(c *gin.Context, query *OperationLogQuery) bool {
	if query == nil || query.TenantIDFilter == 0 {
		return true
	}
	ctx := h.requestTenantContext(c)
	if ctx != nil && ctx.IsMulti() && ctx.TenantID != tenant.PlatformGlobalTenantID {
		common.Fail(c, common.CodeForbidden, "tenant.forbidden")
		return false
	}
	return true
}

func (h *AuditHandler) GetOperationLogList(c *gin.Context) {
	var query OperationLogQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return
	}

	if !h.enforceTenantQueryBoundary(c, &query) {
		return
	}

	page, err := h.service.ListOperationLogs(&query, h.requestTenantContext(c))
	if err != nil {
		common.Fail(c, common.CodeError, "audit.operation_log.list.error")
		return
	}
	common.Success(c, page)
}

func (h *AuditHandler) GetOperationLog(c *gin.Context) {
	logID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return
	}

	resp, err := h.service.GetOperationLog(logID, h.requestTenantContext(c))
	if err != nil {
		common.Fail(c, common.CodeError, "audit.operation_log.detail.error")
		return
	}
	common.Success(c, resp)
}

func (h *AuditHandler) DeleteOperationLog(c *gin.Context) {
	common.SetAuditMetadata(c, "audit.operation_log.delete.title", common.BusinessDelete)
	logID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return
	}

	if err := h.service.DeleteOperationLog(logID, h.requestTenantContext(c)); err != nil {
		common.FailWithError(c, common.CodeError, err, errRequestFailed)
		return
	}
	common.Success(c, gin.H{"deleted": true})
}

func (h *AuditHandler) CleanupOperationLogs(c *gin.Context) {
	common.SetAuditMetadata(c, "audit.operation_log.cleanup.title", common.BusinessClean)

	var req OperationLogCleanupReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return
	}

	clearedCount, err := h.service.CleanupOperationLogs(req.RetentionDays, req.StartedAt, req.EndedAt, h.requestTenantContext(c))
	if err != nil {
		common.FailWithError(c, common.CodeError, err, errRequestFailed)
		return
	}
	common.Success(c, gin.H{"clearedCount": clearedCount})
}

func (h *AuditHandler) BatchDeleteOperationLogs(c *gin.Context) {
	common.SetAuditMetadata(c, "audit.operation_log.batch_delete.title", common.BusinessDelete)

	var req OperationLogBatchDeleteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return
	}

	deletedCount, err := h.service.BatchDeleteOperationLogs(req.IDs, h.requestTenantContext(c))
	if err != nil {
		common.FailWithError(c, common.CodeError, err, errRequestFailed)
		return
	}
	common.Success(c, gin.H{"deletedCount": deletedCount})
}

func (h *AuditHandler) ExportOperationLogs(c *gin.Context) {
	common.SetAuditMetadata(c, "audit.operation_log.export.title", common.BusinessExport)

	var query OperationLogQuery
	if err := c.ShouldBindJSON(&query); err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return
	}
	if !h.enforceTenantQueryBoundary(c, &query) {
		return
	}
	file, err := h.service.ExportOperationLogs(&query, h.requestTenantContext(c))
	if err != nil {
		common.Fail(c, common.CodeError, "audit.operation_log.export.error")
		return
	}
	if err := impexp.WriteCSV(c, *file); err != nil {
		common.Fail(c, common.CodeError, "audit.operation_log.export.error")
	}
}
