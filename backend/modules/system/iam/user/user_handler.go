package iam

import (
	"strconv"

	"pantheon-ops/backend/pkg/common"
	"pantheon-ops/backend/pkg/impexp"

	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	service *UserService
}

func NewUserHandler(s *UserService) *UserHandler {
	return &UserHandler{service: s}
}

// GetProfile 获取个人中心信息。
func (h *UserHandler) GetProfile(c *gin.Context) {
	userID, ok := getUserIDFromContext(c)
	if !ok {
		common.Fail(c, common.CodeUnauthorized, "token.invalid")
		return
	}

	profile, err := h.service.GetProfile(userID)
	if err != nil {
		common.Fail(c, common.CodeError, "user.profile.error")
		return
	}
	common.Success(c, profile)
}

// UpdateProfile 更新个人资料。
func (h *UserHandler) UpdateProfile(c *gin.Context) {
	common.SetAuditMetadata(c, "更新个人资料", common.BusinessUpdate)
	userID, ok := getUserIDFromContext(c)
	if !ok {
		common.Fail(c, common.CodeUnauthorized, "token.invalid")
		return
	}

	var req UserProfileUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}

	profile, err := h.service.UpdateProfile(userID, &req)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "user.profile.update.error")
		return
	}
	common.Success(c, profile)
}

// GetUserList 获取用户列表。
func (h *UserHandler) GetUserList(c *gin.Context) {
	var query UserListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}

	dataScope := common.GetDataScope(c)
	list, err := h.service.ListUsers(&query, dataScope)
	if err != nil {
		common.Fail(c, common.CodeError, "user.list.error")
		return
	}
	common.Success(c, list)
}

func (h *UserHandler) ExportUsers(c *gin.Context) {
	common.SetAuditMetadata(c, "导出用户", common.BusinessExport)

	var query UserListQuery
	if err := c.ShouldBindJSON(&query); err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}
	file, err := h.service.ExportUsers(&query)
	if err != nil {
		common.Fail(c, common.CodeError, "user.export.error")
		return
	}
	if err := impexp.WriteCSV(c, *file); err != nil {
		common.Fail(c, common.CodeError, "user.export.error")
	}
}

func (h *UserHandler) DownloadImportTemplate(c *gin.Context) {
	file := h.service.BuildUserImportTemplate()
	if err := impexp.WriteCSV(c, *file); err != nil {
		common.Fail(c, common.CodeError, "user.import.template.error")
	}
}

func (h *UserHandler) ImportUsers(c *gin.Context) {
	common.SetAuditMetadata(c, "导入用户", common.BusinessImport)

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

	result, err := h.service.ImportUsers(records)
	if err != nil {
		common.Fail(c, common.CodeError, "user.import.error")
		return
	}
	common.Success(c, result)
}

// GetUserDetail 获取用户详情。
func (h *UserHandler) GetUserDetail(c *gin.Context) {
	userID, err := parseUintParam(c, "id")
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}

	detail, err := h.service.GetUserDetail(userID)
	if err != nil {
		common.Fail(c, common.CodeError, "user.detail.error")
		return
	}
	common.Success(c, detail)
}

// CreateUser 创建用户。
func (h *UserHandler) CreateUser(c *gin.Context) {
	common.SetAuditMetadata(c, "新增用户", common.BusinessInsert)
	var req UserCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}

	user, err := h.service.CreateUser(&req)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "user.create.error")
		return
	}
	common.Success(c, user)
}

// UpdateUser 更新用户。
func (h *UserHandler) UpdateUser(c *gin.Context) {
	common.SetAuditMetadata(c, "编辑用户", common.BusinessUpdate)
	var req UserUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}

	userID, err := parseUintParam(c, "id")
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}

	user, err := h.service.UpdateUser(userID, &req)
	if err != nil {
		common.Fail(c, common.CodeError, "user.update.error")
		return
	}
	common.Success(c, user)
}

// ResetPassword 重置用户密码。
func (h *UserHandler) ResetPassword(c *gin.Context) {
	common.SetAuditMetadata(c, "重置用户密码", common.BusinessUpdate)

	userID, err := parseUintParam(c, "id")
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}

	var req UserResetPasswordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}

	revokedSessionCount, err := h.service.ResetPassword(userID, req.NewPassword)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "user.password.reset.error")
		return
	}
	common.Success(c, gin.H{
		"passwordReset":       true,
		"revokedSessionCount": revokedSessionCount,
	})
}

func (h *UserHandler) BatchUpdateUserStatus(c *gin.Context) {
	common.SetAuditMetadata(c, "批量更新用户状态", common.BusinessUpdate)

	var req UserBatchStatusReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}

	updatedCount, err := h.service.BatchUpdateUserStatus(req.UserIDs, req.Status)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "user.status.batch_update.error")
		return
	}
	common.Success(c, gin.H{"updatedCount": updatedCount})
}

func (h *UserHandler) BatchDeleteUsers(c *gin.Context) {
	common.SetAuditMetadata(c, "批量删除用户", common.BusinessDelete)

	var req common.BatchDeleteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}
	resp := common.BatchDelete(req.IDs, h.service.DeleteUser)
	common.Success(c, resp)
}

// DeleteUser 删除用户。
func (h *UserHandler) DeleteUser(c *gin.Context) {
	common.SetAuditMetadata(c, "删除用户", common.BusinessDelete)
	userID, err := parseUintParam(c, "id")
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}

	if err := h.service.DeleteUser(userID); err != nil {
		common.FailWithError(c, common.CodeError, err, "user.delete.error")
		return
	}
	common.Success(c, gin.H{"deleted": true})
}

func getUserIDFromContext(c *gin.Context) (uint64, bool) {
	userIDValue, ok := c.Get("userId")
	if !ok {
		return 0, false
	}
	userID, ok := userIDValue.(uint64)
	return userID, ok
}

func parseUintParam(c *gin.Context, key string) (uint64, error) {
	value := c.Param(key)
	return strconv.ParseUint(value, 10, 64)
}
