package bizscope

import (
	"strconv"

	"github.com/duanxldragon/pantheon-base/backend/pkg/common"

	"github.com/gin-gonic/gin"
)

func parseScopeID(c *gin.Context) (uint64, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return 0, false
	}
	return id, true
}

func parseEnvID(c *gin.Context) (uint64, bool) {
	id, err := strconv.ParseUint(c.Param("envId"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return 0, false
	}
	return id, true
}

// ListEnvironments returns a paged list of environments for a business scope.
func (h *Handler) ListEnvironments(c *gin.Context) {
	scopeID, ok := parseScopeID(c)
	if !ok {
		return
	}
	var query EnvironmentListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	items, err := h.service.ListEnvironments(scopeID, &query)
	if err != nil {
		failBizScopeError(c, err)
		return
	}
	common.Success(c, items)
}

// EnvironmentOptions returns environment options (code + name) for a scope.
func (h *Handler) EnvironmentOptions(c *gin.Context) {
	scopeID, ok := parseScopeID(c)
	if !ok {
		return
	}
	activeOnly := c.Query("activeOnly") == "true"
	items, err := h.service.ListEnvironmentOptions(scopeID, activeOnly)
	if err != nil {
		failBizScopeError(c, err)
		return
	}
	common.Success(c, items)
}

// CreateEnvironment creates a new environment for a business scope.
func (h *Handler) CreateEnvironment(c *gin.Context) {
	common.SetAuditMetadata(c, "business.bizscope.audit.environment_create", common.BusinessInsert)
	scopeID, ok := parseScopeID(c)
	if !ok {
		return
	}
	var req CreateEnvironmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	item, serviceErr := h.service.CreateEnvironment(scopeID, &req)
	if serviceErr != nil {
		failBizScopeError(c, serviceErr)
		return
	}
	common.Success(c, item)
}

// UpdateEnvironment updates an environment for a business scope.
func (h *Handler) UpdateEnvironment(c *gin.Context) {
	common.SetAuditMetadata(c, "business.bizscope.audit.environment_update", common.BusinessUpdate)
	scopeID, ok := parseScopeID(c)
	if !ok {
		return
	}
	envID, ok := parseEnvID(c)
	if !ok {
		return
	}
	var req UpdateEnvironmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	item, serviceErr := h.service.UpdateEnvironment(scopeID, envID, &req)
	if serviceErr != nil {
		failBizScopeError(c, serviceErr)
		return
	}
	common.Success(c, item)
}

// DeleteEnvironment deletes an environment for a business scope.
func (h *Handler) DeleteEnvironment(c *gin.Context) {
	common.SetAuditMetadata(c, "business.bizscope.audit.environment_delete", common.BusinessDelete)
	scopeID, ok := parseScopeID(c)
	if !ok {
		return
	}
	envID, ok := parseEnvID(c)
	if !ok {
		return
	}
	if serviceErr := h.service.DeleteEnvironment(scopeID, envID); serviceErr != nil {
		failBizScopeError(c, serviceErr)
		return
	}
	common.Success(c, gin.H{"deleted": true})
}

// GetEnvironment returns a single environment for a business scope.
func (h *Handler) GetEnvironment(c *gin.Context) {
	scopeID, ok := parseScopeID(c)
	if !ok {
		return
	}
	envID, ok := parseEnvID(c)
	if !ok {
		return
	}
	item, serviceErr := h.service.GetEnvironment(scopeID, envID)
	if serviceErr != nil {
		failBizScopeError(c, serviceErr)
		return
	}
	common.Success(c, item)
}
