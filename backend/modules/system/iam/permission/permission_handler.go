package permission

import (
	"strconv"

	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/impexp"
	"github.com/duanxldragon/pantheon-base/backend/pkg/tenant"

	"github.com/gin-gonic/gin"
)

const errRequestFailed = "request.failed"
const errParamInvalid = "param.invalid"

// permissionServiceErrorCode keeps the HTTP class of a rejected policy write:
// a cross-tenant or out-of-scope target is 403, an unknown row is 404, an
// invalid payload is 400. Collapsing them to 500 hid acceptance F8/F14 behind a
// generic "request failed".
func permissionServiceErrorCode(err error) int {
	return common.CodeForError(err)
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

// svc binds the service to the request tenant context. Every surface that reads
// or writes tenant-owned rows (roles, data-scope policies) must go through it;
// otherwise a tenant administrator operates on the whole installation
// (acceptance F7/F8/F9/F10).
func (h *PermissionHandler) svc(c *gin.Context) *PermissionService {
	return h.service.WithTenantContext(tenant.FromGin(c))
}

func (h *PermissionHandler) GetWorkbench(c *gin.Context) {
	var query PermissionWorkbenchQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return
	}

	workbench, err := h.svc(c).GetWorkbench(&query)
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

	file, err := h.svc(c).ExportWorkbench(&query)
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

	events, err := h.svc(c).ListWorkbenchRemediationEvents(&query)
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

	resp, err := h.svc(c).RemediateWorkbenchPolicies(&req)
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

	resp, err := h.svc(c).ListDataScopePolicies(&query)
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

	resp, err := h.svc(c).UpdateDataScopePolicy(c.Param("roleKey"), &req)
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

	list, err := h.svc(c).ListPolicies(&query)
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

	policy, err := h.svc(c).CreatePolicy(common.GetRoleKeys(c), &req)
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

	policy, err := h.svc(c).UpdatePolicy(common.GetRoleKeys(c), policyID, &req)
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

	if err := h.svc(c).DeletePolicy(common.GetRoleKeys(c), policyID); err != nil {
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
		return h.svc(c).DeletePolicy(operatorRoleKeys, id)
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
	file, err := h.svc(c).ExportPolicies(&query)
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
	// 显式接管 multipart part 的生命周期：ReadCSV 内部也会关闭，
	// 这里 defer 保证所有权清晰，且覆盖未来提前 return 的路径。
	defer func() { _ = file.Close() }()
	records, err := impexp.ReadCSV(file)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, "import.file.invalid_csv")
		return
	}

	result, err := h.svc(c).ImportPolicies(common.GetRoleKeys(c), records)
	if err != nil {
		failPermissionServiceError(c, err)
		return
	}
	common.Success(c, result)
}
