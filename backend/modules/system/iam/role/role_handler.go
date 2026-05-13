package iam

import (
	"strconv"

	"pantheon-ops/backend/pkg/common"
	"pantheon-ops/backend/pkg/impexp"

	"github.com/gin-gonic/gin"
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
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
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
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}

	role, err := h.service.CreateRole(&req)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "request.failed")
		return
	}
	common.Success(c, role)
}

func (h *RoleHandler) ExportRoles(c *gin.Context) {
	common.SetAuditMetadata(c, "导出角色", common.BusinessExport)

	var query RoleListQuery
	if err := c.ShouldBindJSON(&query); err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}

	file, err := h.service.ExportRoles(&query)
	if err != nil {
		common.Fail(c, common.CodeError, "role.export.error")
		return
	}
	if err := impexp.WriteCSV(c, *file); err != nil {
		common.Fail(c, common.CodeError, "role.export.error")
	}
}

// UpdateRole 更新角色。
func (h *RoleHandler) UpdateRole(c *gin.Context) {
	common.SetAuditMetadata(c, "role.update.title", common.BusinessUpdate)

	var req RoleUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}

	roleID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}

	role, err := h.service.UpdateRole(roleID, &req)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "request.failed")
		return
	}
	common.Success(c, role)
}

func (h *RoleHandler) BatchUpdateRoleStatus(c *gin.Context) {
	common.SetAuditMetadata(c, "批量更新角色状态", common.BusinessUpdate)

	var req RoleBatchStatusReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}

	updatedCount, err := h.service.BatchUpdateRoleStatus(req.RoleIDs, req.Status)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "request.failed")
		return
	}
	common.Success(c, gin.H{"updatedCount": updatedCount})
}

func (h *RoleHandler) BatchDeleteRoles(c *gin.Context) {
	common.SetAuditMetadata(c, "批量删除角色", common.BusinessDelete)

	var req common.BatchDeleteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}
	resp := common.BatchDelete(req.IDs, h.service.DeleteRole)
	common.Success(c, resp)
}

// DeleteRole 删除角色。
func (h *RoleHandler) DeleteRole(c *gin.Context) {
	common.SetAuditMetadata(c, "role.delete.title", common.BusinessDelete)

	roleID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}

	if err := h.service.DeleteRole(roleID); err != nil {
		common.FailWithError(c, common.CodeError, err, "request.failed")
		return
	}
	common.Success(c, gin.H{"deleted": true})
}
