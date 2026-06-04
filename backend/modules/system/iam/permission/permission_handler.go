package iam

import (
	"strconv"

	"pantheon-ops/backend/pkg/common"
	"pantheon-ops/backend/pkg/impexp"

	"github.com/gin-gonic/gin"
)

type PermissionHandler struct {
	service *PermissionService
}

func NewPermissionHandler(s *PermissionService) *PermissionHandler {
	return &PermissionHandler{service: s}
}

func (h *PermissionHandler) GetWorkbench(c *gin.Context) {
	var query PermissionWorkbenchQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}

	workbench, err := h.service.GetWorkbench(&query)
	if err != nil {
		common.Fail(c, common.CodeError, "permission.workbench.error")
		return
	}
	common.Success(c, workbench)
}

func (h *PermissionHandler) ExportWorkbench(c *gin.Context) {
	common.SetAuditMetadata(c, "导出权限工作台", common.BusinessExport)

	var query PermissionWorkbenchQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}

	file, err := h.service.ExportWorkbench(&query)
	if err != nil {
		common.Fail(c, common.CodeError, "permission.workbench.export.error")
		return
	}
	if err := impexp.WriteCSV(c, *file); err != nil {
		common.Fail(c, common.CodeError, "permission.workbench.export.error")
	}
}

func (h *PermissionHandler) ListWorkbenchRemediationEvents(c *gin.Context) {
	var query PermissionWorkbenchRemediationQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}

	events, err := h.service.ListWorkbenchRemediationEvents(&query)
	if err != nil {
		common.Fail(c, common.CodeError, "permission.workbench.remediation.error")
		return
	}
	common.Success(c, events)
}

func (h *PermissionHandler) RemediateWorkbenchPolicies(c *gin.Context) {
	common.SetAuditMetadata(c, "补齐推荐接口策略", common.BusinessInsert)

	var req PermissionWorkbenchRemediateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}

	resp, err := h.service.RemediateWorkbenchPolicies(&req)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "request.failed")
		return
	}
	common.Success(c, resp)
}

func (h *PermissionHandler) ListDataScopePolicies(c *gin.Context) {
	var query PermissionDataScopeQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}

	resp, err := h.service.ListDataScopePolicies(&query)
	if err != nil {
		common.Fail(c, common.CodeError, "permission.data_scope.list.error")
		return
	}
	common.Success(c, resp)
}

func (h *PermissionHandler) UpdateDataScopePolicy(c *gin.Context) {
	common.SetAuditMetadata(c, "更新数据权限策略", common.BusinessUpdate)

	var req PermissionDataScopePolicyUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}

	resp, err := h.service.UpdateDataScopePolicy(c.Param("roleKey"), &req)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "request.failed")
		return
	}
	common.Success(c, resp)
}

func (h *PermissionHandler) GetPolicyList(c *gin.Context) {
	var query PermissionPolicyQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}

	list, err := h.service.ListPolicies(&query)
	if err != nil {
		common.Fail(c, common.CodeError, "permission.policy.list.error")
		return
	}
	common.Success(c, list)
}

func (h *PermissionHandler) CreatePolicy(c *gin.Context) {
	common.SetAuditMetadata(c, "新增权限策略", common.BusinessInsert)
	var req PermissionPolicyCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}

	policy, err := h.service.CreatePolicy(&req)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "request.failed")
		return
	}
	common.Success(c, policy)
}

func (h *PermissionHandler) UpdatePolicy(c *gin.Context) {
	common.SetAuditMetadata(c, "编辑权限策略", common.BusinessUpdate)
	var req PermissionPolicyUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}

	policyID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}

	policy, err := h.service.UpdatePolicy(policyID, &req)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "request.failed")
		return
	}
	common.Success(c, policy)
}

func (h *PermissionHandler) DeletePolicy(c *gin.Context) {
	common.SetAuditMetadata(c, "删除权限策略", common.BusinessDelete)
	policyID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}

	if err := h.service.DeletePolicy(policyID); err != nil {
		common.FailWithError(c, common.CodeError, err, "request.failed")
		return
	}
	common.Success(c, gin.H{"deleted": true})
}

func (h *PermissionHandler) BatchDeletePolicies(c *gin.Context) {
	common.SetAuditMetadata(c, "批量删除权限策略", common.BusinessDelete)

	var req common.BatchDeleteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}
	resp := common.BatchDelete(req.IDs, h.service.DeletePolicy)
	common.Success(c, resp)
}

func (h *PermissionHandler) ExportPolicies(c *gin.Context) {
	common.SetAuditMetadata(c, "导出权限策略", common.BusinessExport)

	var query PermissionPolicyQuery
	if err := c.ShouldBindJSON(&query); err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}
	file, err := h.service.ExportPolicies(&query)
	if err != nil {
		common.Fail(c, common.CodeError, "permission.policy.export.error")
		return
	}
	if err := impexp.WriteCSV(c, *file); err != nil {
		common.Fail(c, common.CodeError, "permission.policy.export.error")
	}
}

func (h *PermissionHandler) DownloadImportTemplate(c *gin.Context) {
	file := h.service.BuildImportTemplate()
	if err := impexp.WriteCSV(c, *file); err != nil {
		common.Fail(c, common.CodeError, "permission.policy.import_template.error")
	}
}

func (h *PermissionHandler) ImportPolicies(c *gin.Context) {
	common.SetAuditMetadata(c, "导入权限策略", common.BusinessImport)

	fileHeader, err := c.FormFile("file")
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, "import.file.required")
		return
	}
	file, err := fileHeader.Open()
	if err != nil {
		common.Fail(c, common.CodeError, "import.file.read.error")
		return
	}
	records, err := impexp.ReadCSV(file)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, "import.file.invalid_csv")
		return
	}

	result, err := h.service.ImportPolicies(records)
	if err != nil {
		common.Fail(c, common.CodeError, "permission.policy.import.error")
		return
	}
	common.Success(c, result)
}
