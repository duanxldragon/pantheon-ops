package org

import (
	"strconv"

	"pantheon-base/pkg/common"
	"pantheon-base/pkg/impexp"

	"github.com/gin-gonic/gin"
)

const errRequestFailed = "request.failed"
const errParamInvalid = "param.invalid"

type DeptHandler struct {
	service *DeptService
}

func NewDeptHandler(s *DeptService) *DeptHandler {
	return &DeptHandler{service: s}
}

func (h *DeptHandler) GetDeptTree(c *gin.Context) {
	var query DeptListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return
	}

	tree, err := h.service.GetDeptTree(&query)
	if err != nil {
		common.Fail(c, common.CodeError, "dept.list.error")
		return
	}
	common.Success(c, tree)
}

func (h *DeptHandler) GetDeptOverview(c *gin.Context) {
	overview, err := h.service.GetOverview()
	if err != nil {
		common.Fail(c, common.CodeError, "dept.overview.error")
		return
	}
	common.Success(c, overview)
}

func (h *DeptHandler) GetGovernanceTasks(c *gin.Context) {
	var query DeptGovernanceTaskQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return
	}

	items, err := h.service.ListGovernanceTasks(&query)
	if err != nil {
		common.Fail(c, common.CodeError, "dept.governance.task.error")
		return
	}
	common.Success(c, items)
}

func (h *DeptHandler) GetDeptLeaderCandidates(c *gin.Context) {
	deptID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return
	}

	items, err := h.service.ListLeaderCandidates(deptID)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, errRequestFailed)
		return
	}
	common.Success(c, items)
}

func (h *DeptHandler) CreateDept(c *gin.Context) {
	common.SetAuditMetadata(c, "dept.create.title", common.BusinessInsert)
	var req DeptCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return
	}

	dept, err := h.service.CreateDept(&req)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, errRequestFailed)
		return
	}
	common.Success(c, dept)
}

func (h *DeptHandler) UpdateDept(c *gin.Context) {
	common.SetAuditMetadata(c, "dept.update.title", common.BusinessUpdate)
	var req DeptUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return
	}

	deptID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return
	}

	dept, err := h.service.UpdateDept(deptID, &req)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, errRequestFailed)
		return
	}
	common.Success(c, dept)
}

func (h *DeptHandler) BatchUpdateDeptStatus(c *gin.Context) {
	common.SetAuditMetadata(c, "dept.batch_status.title", common.BusinessUpdate)

	var req DeptBatchStatusReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return
	}

	updatedCount, err := h.service.BatchUpdateDeptStatus(req.DeptIDs, req.Status)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, errRequestFailed)
		return
	}
	common.Success(c, gin.H{"updatedCount": updatedCount})
}

func (h *DeptHandler) BatchUpdateDeptLeader(c *gin.Context) {
	common.SetAuditMetadata(c, "dept.batch_leader.title", common.BusinessUpdate)

	var req DeptBatchLeaderReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return
	}

	updatedCount, err := h.service.BatchUpdateDeptLeader(req.Items)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, errRequestFailed)
		return
	}
	common.Success(c, gin.H{"updatedCount": updatedCount})
}

func (h *DeptHandler) DeleteDept(c *gin.Context) {
	common.SetAuditMetadata(c, "dept.delete.title", common.BusinessDelete)
	deptID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return
	}

	if err := h.service.DeleteDept(deptID); err != nil {
		common.FailWithError(c, common.CodeError, err, errRequestFailed)
		return
	}
	common.Success(c, gin.H{"deleted": true})
}

func (h *DeptHandler) BatchDeleteDepts(c *gin.Context) {
	common.SetAuditMetadata(c, "dept.batch_delete.title", common.BusinessDelete)

	var req common.BatchDeleteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return
	}
	resp := common.BatchDelete(req.IDs, h.service.DeleteDept)
	common.Success(c, resp)
}

func (h *DeptHandler) ExportDepts(c *gin.Context) {
	common.SetAuditMetadata(c, "dept.export.title", common.BusinessExport)

	var query DeptListQuery
	if err := c.ShouldBindJSON(&query); err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return
	}
	file, err := h.service.ExportDepts(c.Request.Context(), &query)
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
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return
	}
	file, err := h.service.ExportGovernanceTasks(&query)
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

	result, err := h.service.ImportDepts(records)
	if err != nil {
		common.Fail(c, common.CodeError, "dept.import.error")
		return
	}
	common.Success(c, result)
}
