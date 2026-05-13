package host

import (
	"strconv"

	"pantheon-ops/backend/pkg/common"

	"github.com/gin-gonic/gin"
)

type HostHandler struct {
	svc *HostService
}

func NewHostHandler(svc *HostService) *HostHandler {
	return &HostHandler{svc: svc}
}

func (h *HostHandler) RegisterRoutes(r gin.IRoutes) {
	r.GET("/hosts", h.List)
	r.GET("/hosts/:id", h.GetByID)
	r.POST("/hosts", h.Create)
	r.PUT("/hosts/:id", h.Update)
	r.DELETE("/hosts/:id", h.Delete)
	r.POST("/hosts/:id/collect", h.Collect)
	r.PATCH("/hosts/:id/status", h.UpdateStatus)
}

func (h *HostHandler) List(c *gin.Context) {
	var query HostListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.Fail(c, common.CodeParamInvalid, "common.param_invalid")
		return
	}
	resp, err := h.svc.List(query, common.GetDataScope(c))
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "cmdbhost.list_failed")
		return
	}
	common.Success(c, resp)
}

func (h *HostHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, "common.param_invalid")
		return
	}
	resp, err := h.svc.GetByID(id, common.GetDataScope(c))
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "cmdbhost.not_found")
		return
	}
	common.Success(c, resp)
}

func (h *HostHandler) Create(c *gin.Context) {
	var req CreateHostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, "common.param_invalid")
		return
	}
	if req.DeptID == 0 {
		if scope := common.GetDataScope(c); scope != nil && scope.DeptID > 0 {
			req.DeptID = scope.DeptID
		}
	}
	createdBy := strconv.FormatUint(common.GetUserID(c), 10)
	resp, err := h.svc.Create(req, createdBy)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "cmdbhost.create_failed")
		return
	}
	common.Success(c, resp)
}

func (h *HostHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, "common.param_invalid")
		return
	}
	var req UpdateHostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, "common.param_invalid")
		return
	}
	updatedBy := strconv.FormatUint(common.GetUserID(c), 10)
	resp, err := h.svc.Update(id, req, updatedBy, common.GetDataScope(c))
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "cmdbhost.update_failed")
		return
	}
	common.Success(c, resp)
}

func (h *HostHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, "common.param_invalid")
		return
	}
	if err := h.svc.Delete(id, common.GetDataScope(c)); err != nil {
		common.FailWithError(c, common.CodeError, err, "cmdbhost.delete_failed")
		return
	}
	common.Success(c, nil)
}

func (h *HostHandler) Collect(c *gin.Context) {
	common.SetAuditMetadata(c, "cmdb.host.collect.title", common.BusinessUpdate)

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, "common.param_invalid")
		return
	}
	var req CollectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, "common.param_invalid")
		return
	}
	resp, err := h.svc.Collect(id, req, common.GetDataScope(c))
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "cmdbhost.collect_failed")
		return
	}
	common.Success(c, resp)
}

func (h *HostHandler) UpdateStatus(c *gin.Context) {
	common.SetAuditMetadata(c, "cmdb.host.status.update.title", common.BusinessUpdate)

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, "common.param_invalid")
		return
	}
	var req UpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, "common.param_invalid")
		return
	}
	if err := h.svc.UpdateStatus(id, req.Status, common.GetDataScope(c)); err != nil {
		common.FailWithError(c, common.CodeError, err, "cmdbhost.status_failed")
		return
	}
	common.Success(c, nil)
}
