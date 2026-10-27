package role

import (
	"errors"
	"strconv"

	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/impexp"
	"github.com/duanxldragon/pantheon-base/backend/pkg/tenant"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	errParamInvalid  = "param.invalid"
	errRequestFailed = "request.failed"
	errRoleNotFound  = "role.not_found"
)

type RoleHandler struct {
	service *RoleService
}

func NewRoleHandler(s *RoleService) *RoleHandler {
	return &RoleHandler{service: s}
}

// svc returns the request-scoped service view bound to the resolved tenant
// context (contract §3.3/§4). A nil context returns the shared service; compat
// binds the platform-global scope (tenant_id=0). Admin operations must use this
// instead of h.service directly, otherwise they read and write platform-global
// roles.
func (h *RoleHandler) svc(c *gin.Context) *RoleService {
	return h.service.WithTenantContext(tenant.FromGin(c))
}

// roleErrorResponse maps a role-service error to the right envelope shape.
//
// The service resolves a role with a tenant-scoped query (getRole), so a
// cross-tenant or stale id surfaces as gorm.ErrRecordNotFound. Answering that
// with request.failed turned an isolation miss into a server fault — the
// caller addressed a role that simply does not exist in its own tenant
// (acceptance F14). Map the not-found sentinels to 404 role.not_found and
// keep everything else on the generic 500 path.
func roleErrorResponse(c *gin.Context, err error) {
	if errors.Is(err, gorm.ErrRecordNotFound) || errors.Is(err, common.ErrNotFound) {
		common.Fail(c, common.CodeNotFound, errRoleNotFound)
		return
	}
	// A rejected role key is the caller's input to fix, not a server fault:
	// CodeForError maps ErrConflict/ErrBadRequest to the parameter-invalid code so
	// the console can surface the message instead of a generic 500.
	common.FailWithError(c, common.CodeForError(err), err, errRequestFailed)
}

// GetRoleList 获取角色列表。
func (h *RoleHandler) GetRoleList(c *gin.Context) {
	var query RoleListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return
	}

	list, err := h.svc(c).ListRoles(&query)
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

	role, err := h.svc(c).CreateRole(&req)
	if err != nil {
		roleErrorResponse(c, err)
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

	file, err := h.svc(c).ExportRoles(c.Request.Context(), &query)
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
	// 显式接管 multipart part 的生命周期：ReadCSV 内部也会关闭，
	// 这里 defer 保证所有权清晰，且覆盖未来提前 return 的路径。
	defer func() { _ = file.Close() }()
	records, err := impexp.ReadCSV(file)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, "import.file.invalid_csv")
		return
	}

	result, err := h.svc(c).ImportRoles(records)
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

	role, err := h.svc(c).UpdateRole(roleID, &req)
	if err != nil {
		roleErrorResponse(c, err)
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

	updatedCount, err := h.svc(c).BatchUpdateRoleStatus(req.RoleIDs, req.Status)
	if err != nil {
		roleErrorResponse(c, err)
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
	resp := common.BatchDelete(req.IDs, h.svc(c).DeleteRole)
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

	resp, err := h.svc(c).ListRoleMembers(roleID, &query)
	if err != nil {
		roleErrorResponse(c, err)
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

	resp, err := h.svc(c).ListAssignableUsers(roleID, &query)
	if err != nil {
		roleErrorResponse(c, err)
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

	addedCount, err := h.svc(c).AddRoleMembers(roleID, req.UserIDs)
	if err != nil {
		roleErrorResponse(c, err)
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

	removedCount, err := h.svc(c).RemoveRoleMembers(roleID, req.UserIDs)
	if err != nil {
		roleErrorResponse(c, err)
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

	if err := h.svc(c).DeleteRole(roleID); err != nil {
		roleErrorResponse(c, err)
		return
	}
	common.Success(c, gin.H{"deleted": true})
}
