package iam

import (
	"strconv"

	"pantheon-base/pkg/common"
	"pantheon-base/pkg/impexp"

	"github.com/gin-gonic/gin"
)

const (
	errParamInvalid  = "param.invalid"
	errRequestFailed = "request.failed"
)

type RoleHandler struct {
	service *RoleService
}

func NewRoleHandler(s *RoleService) *RoleHandler {
	return &RoleHandler{service: s}
}

// GetRoleList 获取角色列表。
func (h *RoleHandler) GetRoleList(c *gin.Context) {
	var query RoleListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return
	}

	list, err := h.service.ListRoles(&query)
	if err != nil {
		common.Fail(c, common.CodeError, "role.list.error")
		return
	}
	common.Success(c, list)
}

// CreateRole 创建角色。
func (h *RoleHandler) CreateRole(c *gin.Context) {
	common.SetAuditMetadata(c, "role.create.title", common.BusinessInsert)

	var req RoleCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return
	}

	role, err := h.service.CreateRole(&req)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, errRequestFailed)
		return
	}
	common.Success(c, role)
}

func (h *RoleHandler) ExportRoles(c *gin.Context) {
	common.SetAuditMetadata(c, "role.export.title", common.BusinessExport)

	var query RoleListQuery
	if err := c.ShouldBindJSON(&query); err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return
	}

	file, err := h.service.ExportRoles(c.Request.Context(), &query)
	if err != nil {
		common.Fail(c, common.CodeError, "role.export.error")
		return
	}
	if err := impexp.WriteCSV(c, *file); err != nil {
		common.Fail(c, common.CodeError, "role.export.error")
	}
}

func (h *RoleHandler) DownloadImportTemplate(c *gin.Context) {
	file := h.service.BuildRoleImportTemplate()
	if err := impexp.WriteCSV(c, *file); err != nil {
		common.Fail(c, common.CodeError, "role.import.template.error")
	}
}

func (h *RoleHandler) ImportRoles(c *gin.Context) {
	common.SetAuditMetadata(c, "role.import.title", common.BusinessImport)

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

	result, err := h.service.ImportRoles(records)
	if err != nil {
		common.Fail(c, common.CodeError, "role.import.error")
		return
	}
	common.Success(c, result)
}

// UpdateRole 更新角色。
func (h *RoleHandler) UpdateRole(c *gin.Context) {
	common.SetAuditMetadata(c, "role.update.title", common.BusinessUpdate)

	var req RoleUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return
	}

	roleID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return
	}

	role, err := h.service.UpdateRole(roleID, &req)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, errRequestFailed)
		return
	}
	common.Success(c, role)
}

func (h *RoleHandler) BatchUpdateRoleStatus(c *gin.Context) {
	common.SetAuditMetadata(c, "role.batch_status.title", common.BusinessUpdate)

	var req RoleBatchStatusReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return
	}

	updatedCount, err := h.service.BatchUpdateRoleStatus(req.RoleIDs, req.Status)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, errRequestFailed)
		return
	}
	common.Success(c, gin.H{"updatedCount": updatedCount})
}

func (h *RoleHandler) BatchDeleteRoles(c *gin.Context) {
	common.SetAuditMetadata(c, "role.batch_delete.title", common.BusinessDelete)

	var req common.BatchDeleteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return
	}
	resp := common.BatchDelete(req.IDs, h.service.DeleteRole)
	common.Success(c, resp)
}

func (h *RoleHandler) GetRoleMembers(c *gin.Context) {
	roleID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return
	}

	var query RoleMemberQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return
	}

	resp, err := h.service.ListRoleMembers(roleID, &query)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, errRequestFailed)
		return
	}
	common.Success(c, resp)
}

func (h *RoleHandler) GetRoleMemberCandidates(c *gin.Context) {
	roleID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return
	}

	var query RoleMemberQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return
	}

	resp, err := h.service.ListAssignableUsers(roleID, &query)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, errRequestFailed)
		return
	}
	common.Success(c, resp)
}

func (h *RoleHandler) AddRoleMembers(c *gin.Context) {
	common.SetAuditMetadata(c, "role.members.update.title", common.BusinessUpdate)

	roleID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return
	}

	var req RoleMemberAssignReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return
	}

	addedCount, err := h.service.AddRoleMembers(roleID, req.UserIDs)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, errRequestFailed)
		return
	}
	common.Success(c, gin.H{"addedCount": addedCount})
}

func (h *RoleHandler) RemoveRoleMembers(c *gin.Context) {
	common.SetAuditMetadata(c, "role.members.remove.title", common.BusinessUpdate)

	roleID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return
	}

	var req RoleMemberAssignReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return
	}

	removedCount, err := h.service.RemoveRoleMembers(roleID, req.UserIDs)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, errRequestFailed)
		return
	}
	common.Success(c, gin.H{"removedCount": removedCount})
}

// DeleteRole 删除角色。
func (h *RoleHandler) DeleteRole(c *gin.Context) {
	common.SetAuditMetadata(c, "role.delete.title", common.BusinessDelete)

	roleID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return
	}

	if err := h.service.DeleteRole(roleID); err != nil {
		common.FailWithError(c, common.CodeError, err, errRequestFailed)
		return
	}
	common.Success(c, gin.H{"deleted": true})
}
