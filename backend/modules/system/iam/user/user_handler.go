package user

import (
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/impexp"
	"github.com/duanxldragon/pantheon-base/backend/pkg/tenant"

	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	service *UserService
}

func NewUserHandler(s *UserService) *UserHandler {
	return &UserHandler{service: s}
}

// svc returns the request-scoped service view bound to the resolved tenant
// context (contract §3.3). A nil context returns the shared service; compat
// binds the platform-global scope (tenant_id=0). Admin operations must use this
// instead of h.service directly, otherwise they read and write platform-global
// rows.
func (h *UserHandler) svc(c *gin.Context) *UserService {
	return h.service.WithTenantContext(tenant.FromGin(c))
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
	common.SetAuditMetadata(c, "user.profile.update.title", common.BusinessUpdate)
	userID, ok := getUserIDFromContext(c)
	if !ok {
		common.Fail(c, common.CodeUnauthorized, "token.invalid")
		return
	}

	var req UserProfileUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, common.MessageParamInvalid)
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
		common.Fail(c, common.CodeParamInvalid, common.MessageParamInvalid)
		return
	}

	dataScope := common.GetDataScope(c)
	list, err := h.svc(c).ListUsers(&query, dataScope)
	if err != nil {
		common.Fail(c, common.CodeError, "user.list.error")
		return
	}
	common.Success(c, list)
}

func (h *UserHandler) ExportUsers(c *gin.Context) {
	common.SetAuditMetadata(c, "user.export.title", common.BusinessExport)

	var query UserListQuery
	if err := c.ShouldBindJSON(&query); err != nil {
		common.Fail(c, common.CodeParamInvalid, common.MessageParamInvalid)
		return
	}
	dataScope := common.GetDataScope(c)
	file, err := h.svc(c).ExportUsers(c.Request.Context(), &query, dataScope)
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
	common.SetAuditMetadata(c, "user.import.title", common.BusinessImport)

	records, ok := impexp.ReadImportCSV(c)
	if !ok {
		return
	}

	result, err := h.svc(c).ImportUsers(records)
	if err != nil {
		common.Fail(c, common.CodeError, "user.import.error")
		return
	}
	common.Success(c, result)
}

// GetUserDetail 获取用户详情。
func (h *UserHandler) GetUserDetail(c *gin.Context) {
	userID, ok := common.ParseUintParam(c, "id")
	if !ok {
		return
	}

	detail, err := h.svc(c).GetUserDetail(userID)
	if err != nil {
		// CodeForError keeps a missing/foreign user at 404 while every other
		// cause stays a server fault (contract §5: do not distinguish the two).
		common.FailWithError(c, common.CodeForError(err), err, "user.detail.error")
		return
	}
	common.Success(c, detail)
}

// CreateUser 创建用户。
func (h *UserHandler) CreateUser(c *gin.Context) {
	common.SetAuditMetadata(c, "user.create.title", common.BusinessInsert)
	var req UserCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, common.MessageParamInvalid)
		return
	}

	user, err := h.svc(c).CreateUser(&req)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "user.create.error")
		return
	}
	common.Success(c, user)
}

// UpdateUser 更新用户。
func (h *UserHandler) UpdateUser(c *gin.Context) {
	common.SetAuditMetadata(c, "user.update.title", common.BusinessUpdate)
	var req UserUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, common.MessageParamInvalid)
		return
	}

	userID, ok := common.ParseUintParam(c, "id")
	if !ok {
		return
	}

	user, err := h.svc(c).UpdateUser(userID, &req)
	if err != nil {
		common.FailWithError(c, common.CodeForError(err), err, "user.update.error")
		return
	}
	common.Success(c, user)
}

// ResetPassword 重置用户密码。
func (h *UserHandler) ResetPassword(c *gin.Context) {
	common.SetAuditMetadata(c, "user.password.reset.title", common.BusinessUpdate)

	userID, ok := common.ParseUintParam(c, "id")
	if !ok {
		return
	}

	var req UserResetPasswordReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, common.MessageParamInvalid)
		return
	}

	revokedSessionCount, err := h.svc(c).ResetPassword(userID, req.NewPassword)
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
	common.SetAuditMetadata(c, "user.batch_status.title", common.BusinessUpdate)

	var req UserBatchStatusReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, common.MessageParamInvalid)
		return
	}

	updatedCount, err := h.svc(c).BatchUpdateUserStatus(req.UserIDs, req.Status)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "user.status.batch_update.error")
		return
	}
	common.Success(c, gin.H{"updatedCount": updatedCount})
}

func (h *UserHandler) BatchDeleteUsers(c *gin.Context) {
	common.SetAuditMetadata(c, "user.batch_delete.title", common.BusinessDelete)

	var req common.BatchDeleteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, common.MessageParamInvalid)
		return
	}
	resp := common.BatchDelete(req.IDs, h.svc(c).DeleteUser)
	common.Success(c, resp)
}

// DeleteUser 删除用户。
func (h *UserHandler) DeleteUser(c *gin.Context) {
	common.SetAuditMetadata(c, "user.delete.title", common.BusinessDelete)
	userID, ok := common.ParseUintParam(c, "id")
	if !ok {
		return
	}

	if err := h.svc(c).DeleteUser(userID); err != nil {
		common.FailWithError(c, common.CodeError, err, "user.delete.error")
		return
	}
	common.Success(c, gin.H{"deleted": true})
}

// getUserIDFromContext keeps the (uint64, bool) shape the handler call sites
// expect while delegating the key lookup to the shared implementation; the
// token middleware stores the id as uint64 under "userId" and real ids are
// nonzero, so a zero result means absent or mistyped.
func getUserIDFromContext(c *gin.Context) (uint64, bool) {
	userID := common.GetUserID(c)
	return userID, userID != 0
}
