package bizscope

import (
	"strconv"

	"pantheon-ops/backend/pkg/common"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) List(c *gin.Context) {
	var query BizScopeListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}
	items, err := h.service.List(&query, common.GetDataScope(c))
	if err != nil {
		common.Fail(c, common.CodeError, "bizscope.list.error")
		return
	}
	common.Success(c, items)
}

func (h *Handler) Options(c *gin.Context) {
	items, err := h.service.ListOptions()
	if err != nil {
		common.Fail(c, common.CodeError, "bizscope.options.error")
		return
	}
	common.Success(c, items)
}

func (h *Handler) Detail(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}
	item, serviceErr := h.service.Get(id)
	if serviceErr != nil {
		common.Fail(c, common.CodeError, "bizscope.detail.error")
		return
	}
	common.Success(c, item)
}

func (h *Handler) Hosts(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}
	items, serviceErr := h.service.ListBoundHosts(id)
	if serviceErr != nil {
		common.Fail(c, common.CodeError, "bizscope.hosts.error")
		return
	}
	common.Success(c, items)
}

func (h *Handler) AvailableHosts(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}
	items, serviceErr := h.service.ListAvailableHosts(id)
	if serviceErr != nil {
		common.Fail(c, common.CodeError, "bizscope.availableHosts.error")
		return
	}
	common.Success(c, items)
}

func (h *Handler) BindHosts(c *gin.Context) {
	common.SetAuditMetadata(c, "business.bizscope.audit.update", common.BusinessUpdate)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}
	var req BindBizScopeHostsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}
	if serviceErr := h.service.BindHosts(id, req.HostIDs); serviceErr != nil {
		common.Fail(c, common.CodeError, "bizscope.bindHosts.error")
		return
	}
	common.Success(c, gin.H{"bound": true})
}

func (h *Handler) UnbindHost(c *gin.Context) {
	common.SetAuditMetadata(c, "business.bizscope.audit.update", common.BusinessUpdate)
	scopeID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}
	hostID, err := strconv.ParseUint(c.Param("hostId"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}
	if serviceErr := h.service.UnbindHost(scopeID, hostID); serviceErr != nil {
		common.Fail(c, common.CodeError, "bizscope.unbindHost.error")
		return
	}
	common.Success(c, gin.H{"unbound": true})
}

func (h *Handler) Create(c *gin.Context) {
	common.SetAuditMetadata(c, "business.bizscope.audit.create", common.BusinessInsert)
	var req CreateBizScopeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}
	item, serviceErr := h.service.Create(&req)
	if serviceErr != nil {
		common.Fail(c, common.CodeError, "bizscope.create.error")
		return
	}
	common.Success(c, item)
}

func (h *Handler) Update(c *gin.Context) {
	common.SetAuditMetadata(c, "business.bizscope.audit.update", common.BusinessUpdate)
	var req UpdateBizScopeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}
	item, serviceErr := h.service.Update(id, &req)
	if serviceErr != nil {
		common.Fail(c, common.CodeError, "bizscope.update.error")
		return
	}
	common.Success(c, item)
}

func (h *Handler) Delete(c *gin.Context) {
	common.SetAuditMetadata(c, "business.bizscope.audit.delete", common.BusinessDelete)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}
	if serviceErr := h.service.Delete(id); serviceErr != nil {
		common.Fail(c, common.CodeError, "bizscope.delete.error")
		return
	}
	common.Success(c, gin.H{"deleted": true})
}
