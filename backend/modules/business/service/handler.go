package service

import (
	"strconv"

	"pantheon-base/pkg/common"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	manager *Manager
}

func (h *Handler) reconcileInstanceState(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req ReconcileInstanceRequest
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			common.Fail(c, common.CodeParamInvalid, "common.param_invalid")
			return
		}
	}
	ref, err := h.manager.ReconcileInstanceState(c.Request.Context(), id, req, actor(c), common.GetDataScope(c))
	if err != nil {
		fail(c, err, "business.service.instance_reconcile_failed")
		return
	}
	common.Success(c, ref)
}

func NewHandler(manager *Manager) *Handler { return &Handler{manager: manager} }

func (h *Handler) RegisterRoutes(r gin.IRoutes) {
	r.GET("/applications", h.listApplications)
	r.GET("/applications/options", h.applicationOptions)
	r.POST("/applications", h.createApplication)
	r.PUT("/applications/:id", h.updateApplication)
	r.DELETE("/applications/:id", h.deleteApplication)
	r.GET("/services", h.listServices)
	r.GET("/services/options", h.serviceOptions)
	r.POST("/services", h.createService)
	r.PUT("/services/:id", h.updateService)
	r.DELETE("/services/:id", h.deleteService)
	r.GET("/instances", h.listInstances)
	r.GET("/instances/:id", h.getInstance)
	r.POST("/instances", h.createInstance)
	r.PUT("/instances/:id", h.updateInstance)
	r.DELETE("/instances/:id", h.deleteInstance)
	r.PATCH("/instances/:id/state", h.transitionInstanceState)
	r.POST("/instances/:id/reconcile", h.reconcileInstanceState)
}

func (h *Handler) listApplications(c *gin.Context) {
	var q ApplicationQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		common.Fail(c, common.CodeParamInvalid, "common.param_invalid")
		return
	}
	resp, err := h.manager.ListApplications(q, common.GetDataScope(c))
	if err != nil {
		fail(c, err, "business.service.application_list_failed")
		return
	}
	common.Success(c, resp)
}

func (h *Handler) applicationOptions(c *gin.Context) {
	resp, err := h.manager.ListApplicationOptions(common.GetDataScope(c))
	if err != nil {
		fail(c, err, "business.service.application_options_failed")
		return
	}
	common.Success(c, resp)
}

func (h *Handler) createApplication(c *gin.Context) {
	var req CreateApplicationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, "common.param_invalid")
		return
	}
	resp, err := h.manager.CreateApplication(req, actor(c), common.GetDataScope(c))
	if err != nil {
		fail(c, err, "business.service.application_create_failed")
		return
	}
	common.Success(c, resp)
}

func (h *Handler) updateApplication(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req UpdateApplicationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, "common.param_invalid")
		return
	}
	resp, err := h.manager.UpdateApplication(id, req, actor(c), common.GetDataScope(c))
	if err != nil {
		fail(c, err, "business.service.application_update_failed")
		return
	}
	common.Success(c, resp)
}

func (h *Handler) deleteApplication(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.manager.DeleteApplication(id, common.GetDataScope(c)); err != nil {
		fail(c, err, "business.service.application_delete_failed")
		return
	}
	common.Success(c, gin.H{"deleted": true})
}

func (h *Handler) transitionInstanceState(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req InstanceStateTransitionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, "common.param_invalid")
		return
	}
	req.InstanceID = id
	ref, err := h.manager.TransitionState(c.Request.Context(), req, actor(c), common.GetDataScope(c))
	if err != nil {
		fail(c, err, "business.service.instance_state_update_failed")
		return
	}
	common.Success(c, ref)
}

func (h *Handler) listServices(c *gin.Context) {
	var q ServiceQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		common.Fail(c, common.CodeParamInvalid, "common.param_invalid")
		return
	}
	resp, err := h.manager.ListServices(q, common.GetDataScope(c))
	if err != nil {
		fail(c, err, "business.service.service_list_failed")
		return
	}
	common.Success(c, resp)
}

func (h *Handler) serviceOptions(c *gin.Context) {
	var q struct {
		ApplicationID uint64 `form:"applicationId"`
	}
	if err := c.ShouldBindQuery(&q); err != nil {
		common.Fail(c, common.CodeParamInvalid, "common.param_invalid")
		return
	}
	resp, err := h.manager.ListServiceOptions(q.ApplicationID, common.GetDataScope(c))
	if err != nil {
		fail(c, err, "business.service.service_options_failed")
		return
	}
	common.Success(c, resp)
}

func (h *Handler) createService(c *gin.Context) {
	var req CreateServiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, "common.param_invalid")
		return
	}
	resp, err := h.manager.CreateService(req, actor(c), common.GetDataScope(c))
	if err != nil {
		fail(c, err, "business.service.service_create_failed")
		return
	}
	common.Success(c, resp)
}

func (h *Handler) updateService(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req UpdateServiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, "common.param_invalid")
		return
	}
	resp, err := h.manager.UpdateService(id, req, actor(c), common.GetDataScope(c))
	if err != nil {
		fail(c, err, "business.service.service_update_failed")
		return
	}
	common.Success(c, resp)
}

func (h *Handler) deleteService(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.manager.DeleteService(id, common.GetDataScope(c)); err != nil {
		fail(c, err, "business.service.service_delete_failed")
		return
	}
	common.Success(c, gin.H{"deleted": true})
}

func (h *Handler) listInstances(c *gin.Context) {
	var q InstanceQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		common.Fail(c, common.CodeParamInvalid, "common.param_invalid")
		return
	}
	resp, err := h.manager.ListInstances(q, common.GetDataScope(c))
	if err != nil {
		fail(c, err, "business.service.instance_list_failed")
		return
	}
	common.Success(c, resp)
}

func (h *Handler) getInstance(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	ref, err := h.manager.GetInstance(c.Request.Context(), id, common.GetDataScope(c))
	if err != nil {
		fail(c, err, "business.service.instance_not_found")
		return
	}
	common.Success(c, ref)
}

func (h *Handler) createInstance(c *gin.Context) {
	var req CreateInstanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, "common.param_invalid")
		return
	}
	ref, err := h.manager.CreateInstance(c.Request.Context(), req, actor(c), common.GetDataScope(c))
	if err != nil {
		fail(c, err, "business.service.instance_create_failed")
		return
	}
	common.Success(c, ref)
}

func (h *Handler) updateInstance(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	var req UpdateInstanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, "common.param_invalid")
		return
	}
	resp, err := h.manager.UpdateInstance(id, req, actor(c), common.GetDataScope(c))
	if err != nil {
		fail(c, err, "business.service.instance_update_failed")
		return
	}
	common.Success(c, resp)
}

func (h *Handler) deleteInstance(c *gin.Context) {
	id, ok := parseID(c)
	if !ok {
		return
	}
	if err := h.manager.DeleteInstance(id, common.GetDataScope(c)); err != nil {
		fail(c, err, "business.service.instance_delete_failed")
		return
	}
	common.Success(c, gin.H{"deleted": true})
}

func parseID(c *gin.Context) (uint64, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, "common.param_invalid")
		return 0, false
	}
	return id, true
}

func actor(c *gin.Context) string {
	return strconv.FormatUint(common.GetUserID(c), 10)
}

func fail(c *gin.Context, err error, fallback string) {
	common.FailWithError(c, common.CodeError, err, fallback)
}
