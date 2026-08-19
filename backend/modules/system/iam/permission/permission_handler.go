package iam

import (
	"errors"
	"strconv"

	"pantheon-base/pkg/common"
	"pantheon-base/pkg/impexp"

	"github.com/gin-gonic/gin"
)

const errRequestFailed = "request.failed"
const errParamInvalid = "param.invalid"

func permissionServiceErrorCode(err error) int {
	if errors.Is(err, common.ErrForbidden) {
		return common.CodeForbidden
	}
	return common.CodeError
}

func failPermissionServiceError(c *gin.Context, err error) {
	code := permissionServiceErrorCode(err)
	common.FailWithError(c, code, err, errRequestFailed)
}

type PermissionHandler struct {
	service *PermissionService
}

func NewPermissionHandler(s *PermissionService) *PermissionHandler {
	return &PermissionHandler{service: s}
}

func (h *PermissionHandler) GetWorkbench(c *gin.Context) {
	var query PermissionWorkbenchQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
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
	common.SetAuditMetadata(c, "permission.workbench.export.title", common.BusinessExport)

	var query PermissionWorkbenchQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
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
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
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
	common.SetAuditMetadata(c, "permission.policy.sync_recommended.title", common.BusinessInsert)

	var req PermissionWorkbenchRemediateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return
	}

	resp, err := h.service.RemediateWorkbenchPolicies(&req)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, errRequestFailed)
		return
	}
	common.Success(c, resp)
}

func (h *PermissionHandler) ListDataScopePolicies(c *gin.Context) {
	var query PermissionDataScopeQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
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
	common.SetAuditMetadata(c, "permission.data_policy.update.title", common.BusinessUpdate)

	var req PermissionDataScopePolicyUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return
	}

	resp, err := h.service.UpdateDataScopePolicy(c.Param("roleKey"), &req)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, errRequestFailed)
		return
	}
	common.Success(c, resp)
}

func (h *PermissionHandler) GetPolicyList(c *gin.Context) {
	var query PermissionPolicyQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
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
	common.SetAuditMetadata(c, "permission.policy.create.title", common.BusinessInsert)
	var req PermissionPolicyCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return
	}

	policy, err := h.service.CreatePolicy(common.GetRoleKeys(c), &req)
	if err != nil {
		failPermissionServiceError(c, err)
		return
	}
	common.Success(c, policy)
}

func (h *PermissionHandler) UpdatePolicy(c *gin.Context) {
	common.SetAuditMetadata(c, "permission.policy.update.title", common.BusinessUpdate)
	var req PermissionPolicyUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return
	}

	policyID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return
	}

	policy, err := h.service.UpdatePolicy(common.GetRoleKeys(c), policyID, &req)
	if err != nil {
		failPermissionServiceError(c, err)
		return
	}
	common.Success(c, policy)
}

func (h *PermissionHandler) DeletePolicy(c *gin.Context) {
	common.SetAuditMetadata(c, "permission.policy.delete.title", common.BusinessDelete)
	policyID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return
	}

	if err := h.service.DeletePolicy(common.GetRoleKeys(c), policyID); err != nil {
		failPermissionServiceError(c, err)
		return
	}
	common.Success(c, gin.H{"deleted": true})
}

func (h *PermissionHandler) BatchDeletePolicies(c *gin.Context) {
	common.SetAuditMetadata(c, "permission.policy.batch_delete.title", common.BusinessDelete)

	var req common.BatchDeleteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return
	}
	operatorRoleKeys := common.GetRoleKeys(c)
	resp := common.BatchDelete(req.IDs, func(id uint64) error {
		return h.service.DeletePolicy(operatorRoleKeys, id)
	})
	common.Success(c, resp)
}

func (h *PermissionHandler) ExportPolicies(c *gin.Context) {
	common.SetAuditMetadata(c, "permission.policy.export.title", common.BusinessExport)

	var query PermissionPolicyQuery
	if err := c.ShouldBindJSON(&query); err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
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
	common.SetAuditMetadata(c, "permission.policy.import.title", common.BusinessImport)

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

	result, err := h.service.ImportPolicies(common.GetRoleKeys(c), records)
	if err != nil {
		failPermissionServiceError(c, err)
		return
	}
	common.Success(c, result)
}
