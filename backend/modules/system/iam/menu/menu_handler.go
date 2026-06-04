package iam

import (
	"strconv"

	"pantheon-ops/backend/pkg/common"

	"github.com/gin-gonic/gin"
)

type MenuHandler struct {
	service *MenuService
}

func NewMenuHandler(s *MenuService) *MenuHandler {
	return &MenuHandler{service: s}
}

// GetMenuTree 获取菜单树 API
func (h *MenuHandler) GetMenuTree(c *gin.Context) {
	var query MenuListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}

	roleKeys := getRoleKeysFromContext(c)
	if normalizeMenuScope(&query) == "manage" {
		allowed, err := h.service.HasManageAccess(roleKeys)
		if err != nil {
			common.Fail(c, common.CodeError, "menu.fetch.error")
			return
		}
		if !allowed {
			common.Fail(c, common.CodeForbidden, "permission.denied")
			return
		}
	}
	tree, err := h.service.GetMenuTree(&query, roleKeys)
	if err != nil {
		common.Fail(c, common.CodeError, "menu.fetch.error")
		return
	}
	common.Success(c, tree)
}

// CreateMenu 创建菜单。
func (h *MenuHandler) CreateMenu(c *gin.Context) {
	common.SetAuditMetadata(c, "menu.create.title", common.BusinessInsert)

	var req MenuCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}

	menu, err := h.service.CreateMenu(&req)
	if err != nil {
		common.Fail(c, menuErrorCode(err), menuErrorMessage(err, "menu.create.error"))
		return
	}
	common.Success(c, menu)
}

// UpdateMenu 更新菜单。
func (h *MenuHandler) UpdateMenu(c *gin.Context) {
	common.SetAuditMetadata(c, "menu.update.title", common.BusinessUpdate)

	var req MenuUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}

	menuID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}

	menu, err := h.service.UpdateMenu(menuID, &req)
	if err != nil {
		common.Fail(c, menuErrorCode(err), menuErrorMessage(err, "menu.update.error"))
		return
	}
	common.Success(c, menu)
}

// DeleteMenu 删除菜单。
func (h *MenuHandler) DeleteMenu(c *gin.Context) {
	common.SetAuditMetadata(c, "menu.delete.title", common.BusinessDelete)

	menuID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}

	if err := h.service.DeleteMenu(menuID); err != nil {
		common.FailWithError(c, common.CodeError, err, "request.failed")
		return
	}
	common.Success(c, gin.H{"deleted": true})
}

func getRoleKeysFromContext(c *gin.Context) []string {
	roleKeysValue, ok := c.Get("roleKeys")
	if ok {
		if roleKeys, ok := roleKeysValue.([]string); ok {
			return roleKeys
		}
	}
	roleKeyValue, ok := c.Get("roleKey")
	if ok {
		if roleKey, ok := roleKeyValue.(string); ok && roleKey != "" {
			return []string{roleKey}
		}
	}
	return nil
}

func menuErrorCode(err error) int {
	if isMenuValidationError(err) {
		return common.CodeParamInvalid
	}
	return common.CodeError
}

func menuErrorMessage(err error, fallback string) string {
	if isMenuValidationError(err) {
		return err.Error()
	}
	return fallback
}

func isMenuValidationError(err error) bool {
	if err == nil {
		return false
	}
	switch err.Error() {
	case "menu.update.error.parent_self",
		"menu.route_name.required",
		"menu.page_perm.required",
		"menu.perms.required",
		"menu.path.invalid_external",
		"menu.component.required",
		"menu.component.invalid",
		"menu.parent.not_found",
		"menu.path.exists",
		"menu.route_name.exists":
		return true
	default:
		return false
	}
}
