package bizscope

import (
	"strconv"

	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/gin-gonic/gin"
)

func (h *Handler) ListEnvironments(c *gin.Context) {
	var query EnvironmentListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	page, err := h.service.ListEnvironments(c.Request.Context(), query, common.GetDataScope(c))
	if err != nil {
		failEnvironmentError(c, err)
		return
	}
	common.Success(c, page)
}

func environmentID(c *gin.Context) (uint64, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return 0, false
	}
	return id, true
}

func (h *Handler) GetEnvironment(c *gin.Context) {
	id, ok := environmentID(c)
	if !ok {
		return
	}
	item, err := h.service.GetEnvironment(c.Request.Context(), id, common.GetDataScope(c))
	if err != nil {
		failEnvironmentError(c, err)
		return
	}
	common.Success(c, item)
}

func (h *Handler) CreateEnvironment(c *gin.Context) {
	common.SetAuditMetadata(c, "business.bizscope.environment.audit.create", common.BusinessInsert)
	var req CreateEnvironmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	item, err := h.service.CreateEnvironment(c.Request.Context(), req, common.GetDataScope(c))
	if err != nil {
		failEnvironmentError(c, err)
		return
	}
	common.Success(c, item)
}

func (h *Handler) UpdateEnvironment(c *gin.Context) {
	common.SetAuditMetadata(c, "business.bizscope.environment.audit.update", common.BusinessUpdate)
	id, ok := environmentID(c)
	if !ok {
		return
	}
	var req UpdateEnvironmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	item, err := h.service.UpdateEnvironment(c.Request.Context(), id, req, common.GetDataScope(c))
	if err != nil {
		failEnvironmentError(c, err)
		return
	}
	common.Success(c, item)
}

func (h *Handler) DeleteEnvironment(c *gin.Context) {
	common.SetAuditMetadata(c, "business.bizscope.environment.audit.delete", common.BusinessDelete)
	id, ok := environmentID(c)
	if !ok {
		return
	}
	if err := h.service.DeleteEnvironment(c.Request.Context(), id, common.GetDataScope(c)); err != nil {
		failEnvironmentError(c, err)
		return
	}
	common.Success(c, gin.H{"deleted": true})
}

func failEnvironmentError(c *gin.Context, err error) {
	switch key := common.ErrMessage(err); key {
	case environmentNotFoundKey, environmentCodeExistsKey, environmentCodeRequiredKey, environmentNameRequiredKey, environmentScopeRequiredKey, environmentTypeInvalidKey, environmentStatusInvalidKey, bizScopeNotFoundKey, msgParamInvalid, "permission.denied":
		common.Fail(c, common.CodeError, key)
	default:
		common.FailWithError(c, common.CodeError, err, "request.failed")
	}
}
