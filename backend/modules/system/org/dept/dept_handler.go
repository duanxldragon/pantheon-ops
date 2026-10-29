package dept

import (
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/impexp"
	"github.com/duanxldragon/pantheon-base/backend/pkg/tenant"

	"github.com/gin-gonic/gin"
)

type DeptHandler struct {
	service *DeptService
}

func NewDeptHandler(s *DeptService) *DeptHandler {
	return &DeptHandler{service: s}
}

// svc returns the request-scoped service view bound to the resolved tenant
// context (contract §3.3). A nil context returns the shared service; compat
// binds the platform-global scope (tenant_id=0). Admin operations must use this
// instead of h.service directly, otherwise they read and write the
// platform-global tree.
func (h *DeptHandler) svc(c *gin.Context) *DeptService {
	return h.service.WithTenantContext(tenant.FromGin(c))
}

func (h *DeptHandler) GetDeptTree(c *gin.Context) {
	var query DeptListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.Fail(c, common.CodeParamInvalid, common.MessageParamInvalid)
		return
	}

	tree, err := h.svc(c).GetDeptTree(&query)
	if err != nil {
		common.Fail(c, common.CodeError, "dept.list.error")
		return
	}
	common.Success(c, tree)
}

func (h *DeptHandler) GetDeptOverview(c *gin.Context) {
	overview, err := h.svc(c).GetOverview()
	if err != nil {
		common.Fail(c, common.CodeError, "dept.overview.error")
		return
	}
	common.Success(c, overview)
}

func (h *DeptHandler) GetGovernanceTasks(c *gin.Context) {
	var query DeptGovernanceTaskQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.Fail(c, common.CodeParamInvalid, common.MessageParamInvalid)
		return
	}

	items, err := h.svc(c).ListGovernanceTasks(&query)
	if err != nil {
		common.Fail(c, common.CodeError, "dept.governance.task.error")
		return
	}
	common.Success(c, items)
}

func (h *DeptHandler) GetDeptLeaderCandidates(c *gin.Context) {
	deptID, ok := common.ParseUintParam(c, "id")
	if !ok {
		return
	}

	items, err := h.svc(c).ListLeaderCandidates(deptID)
	if err != nil {
		common.FailWithError(c, common.CodeForError(err), err, common.MessageRequestFailed)
		return
	}
	common.Success(c, items)
}

func (h *DeptHandler) CreateDept(c *gin.Context) {
	common.SetAuditMetadata(c, "dept.create.title", common.BusinessInsert)
	var req DeptCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, common.MessageParamInvalid)
		return
	}

	dept, err := h.svc(c).CreateDept(&req)
	if err != nil {
		common.FailWithError(c, common.CodeForError(err), err, common.MessageRequestFailed)
		return
	}
	common.Success(c, dept)
}

func (h *DeptHandler) UpdateDept(c *gin.Context) {
	common.SetAuditMetadata(c, "dept.update.title", common.BusinessUpdate)
	var req DeptUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, common.MessageParamInvalid)
		return
	}

	deptID, ok := common.ParseUintParam(c, "id")
	if !ok {
		return
	}

	dept, err := h.svc(c).UpdateDept(deptID, &req)
	if err != nil {
		common.FailWithError(c, common.CodeForError(err), err, common.MessageRequestFailed)
		return
	}
	common.Success(c, dept)
}

func (h *DeptHandler) BatchUpdateDeptStatus(c *gin.Context) {
	common.SetAuditMetadata(c, "dept.batch_status.title", common.BusinessUpdate)

	var req DeptBatchStatusReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, common.MessageParamInvalid)
		return
	}

	updatedCount, err := h.svc(c).BatchUpdateDeptStatus(req.DeptIDs, req.Status)
	if err != nil {
		common.FailWithError(c, common.CodeForError(err), err, common.MessageRequestFailed)
		return
	}
	common.Success(c, gin.H{"updatedCount": updatedCount})
}

func (h *DeptHandler) BatchUpdateDeptLeader(c *gin.Context) {
	common.SetAuditMetadata(c, "dept.batch_leader.title", common.BusinessUpdate)

	var req DeptBatchLeaderReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, common.MessageParamInvalid)
		return
	}

	updatedCount, err := h.svc(c).BatchUpdateDeptLeader(req.Items)
	if err != nil {
		common.FailWithError(c, common.CodeForError(err), err, common.MessageRequestFailed)
		return
	}
	common.Success(c, gin.H{"updatedCount": updatedCount})
}

func (h *DeptHandler) DeleteDept(c *gin.Context) {
	common.SetAuditMetadata(c, "dept.delete.title", common.BusinessDelete)
	deptID, ok := common.ParseUintParam(c, "id")
	if !ok {
		return
	}

	if err := h.svc(c).DeleteDept(deptID); err != nil {
		common.FailWithError(c, common.CodeForError(err), err, common.MessageRequestFailed)
		return
	}
	common.Success(c, gin.H{"deleted": true})
}

func (h *DeptHandler) BatchDeleteDepts(c *gin.Context) {
	common.SetAuditMetadata(c, "dept.batch_delete.title", common.BusinessDelete)

	var req common.BatchDeleteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, common.MessageParamInvalid)
		return
	}
	resp := common.BatchDelete(req.IDs, h.svc(c).DeleteDept)
	common.Success(c, resp)
}

func (h *DeptHandler) ExportDepts(c *gin.Context) {
	common.SetAuditMetadata(c, "dept.export.title", common.BusinessExport)

	var query DeptListQuery
	if err := c.ShouldBindJSON(&query); err != nil {
		common.Fail(c, common.CodeParamInvalid, common.MessageParamInvalid)
		return
	}
	file, err := h.svc(c).ExportDepts(c.Request.Context(), &query)
	if err != nil {
		common.Fail(c, common.CodeError, "dept.export.error")
		return
	}
	if err := impexp.WriteCSV(c, *file); err != nil {
		common.Fail(c, common.CodeError, "dept.export.error")
	}
}

func (h *DeptHandler) ExportGovernanceTasks(c *gin.Context) {
	common.SetAuditMetadata(c, "dept.governance.export.title", common.BusinessExport)

	var query DeptGovernanceTaskQuery
	if err := c.ShouldBindJSON(&query); err != nil {
		common.Fail(c, common.CodeParamInvalid, common.MessageParamInvalid)
		return
	}
	file, err := h.svc(c).ExportGovernanceTasks(&query)
	if err != nil {
		common.Fail(c, common.CodeError, "dept.governance.task.export.error")
		return
	}
	if err := impexp.WriteCSV(c, *file); err != nil {
		common.Fail(c, common.CodeError, "dept.governance.task.export.error")
	}
}

func (h *DeptHandler) DownloadImportTemplate(c *gin.Context) {
	file := h.service.BuildDeptImportTemplate()
	if err := impexp.WriteCSV(c, *file); err != nil {
		common.Fail(c, common.CodeError, "dept.import.template.error")
	}
}

func (h *DeptHandler) ImportDepts(c *gin.Context) {
	common.SetAuditMetadata(c, "dept.import.title", common.BusinessImport)

	records, ok := impexp.ReadImportCSV(c)
	if !ok {
		return
	}

	result, err := h.svc(c).ImportDepts(records)
	if err != nil {
		common.Fail(c, common.CodeError, "dept.import.error")
		return
	}
	common.Success(c, result)
}
