package system

import (
	"strconv"

	"pantheon-ops/backend/pkg/common"
	"pantheon-ops/backend/pkg/impexp"

	"github.com/gin-gonic/gin"
)

type AuditHandler struct {
	service *AuditService
}

func NewAuditHandler(s *AuditService) *AuditHandler {
	return &AuditHandler{service: s}
}

func (h *AuditHandler) GetOperationLogList(c *gin.Context) {
	var query OperationLogQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}

	page, err := h.service.ListOperationLogs(&query)
	if err != nil {
		common.Fail(c, common.CodeError, "audit.operation_log.list.error")
		return
	}
	common.Success(c, page)
}

func (h *AuditHandler) GetOperationLog(c *gin.Context) {
	logID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}

	resp, err := h.service.GetOperationLog(logID)
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
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}

	if err := h.service.DeleteOperationLog(logID); err != nil {
		common.FailWithError(c, common.CodeError, err, "request.failed")
		return
	}
	common.Success(c, gin.H{"deleted": true})
}

func (h *AuditHandler) CleanupOperationLogs(c *gin.Context) {
	common.SetAuditMetadata(c, "audit.operation_log.cleanup.title", common.BusinessClean)

	var req OperationLogCleanupReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}

	clearedCount, err := h.service.CleanupOperationLogs(req.RetentionDays, req.StartedAt, req.EndedAt)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "request.failed")
		return
	}
	common.Success(c, gin.H{"clearedCount": clearedCount})
}

func (h *AuditHandler) BatchDeleteOperationLogs(c *gin.Context) {
	common.SetAuditMetadata(c, "audit.operation_log.batch_delete.title", common.BusinessDelete)

	var req OperationLogBatchDeleteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}

	deletedCount, err := h.service.BatchDeleteOperationLogs(req.IDs)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "request.failed")
		return
	}
	common.Success(c, gin.H{"deletedCount": deletedCount})
}

func (h *AuditHandler) ExportOperationLogs(c *gin.Context) {
	common.SetAuditMetadata(c, "audit.operation_log.export.title", common.BusinessExport)

	var query OperationLogQuery
	if err := c.ShouldBindJSON(&query); err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}
	file, err := h.service.ExportOperationLogs(&query)
	if err != nil {
		common.Fail(c, common.CodeError, "audit.operation_log.export.error")
		return
	}
	if err := impexp.WriteCSV(c, *file); err != nil {
		common.Fail(c, common.CodeError, "audit.operation_log.export.error")
	}
}
