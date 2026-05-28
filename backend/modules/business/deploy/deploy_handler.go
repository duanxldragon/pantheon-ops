package deploy

import (
	"strconv"

	"pantheon-ops/backend/pkg/common"

	"github.com/gin-gonic/gin"
)

type DeployHandler struct {
	svc *DeployService
}

func NewDeployHandler(svc *DeployService) *DeployHandler {
	return &DeployHandler{svc: svc}
}

func (h *DeployHandler) RegisterRoutes(r gin.IRoutes) {
	r.GET("/packages", h.ListPackages)
	r.POST("/packages", h.CreatePackage)
	r.GET("/packages/:id", h.GetPackage)
	r.PUT("/packages/:id", h.UpdatePackage)
	r.DELETE("/packages/:id", h.DeletePackage)
	r.GET("/templates", h.ListTemplates)
	r.POST("/templates", h.CreateTemplate)
	r.GET("/templates/:id", h.GetTemplate)
	r.PUT("/templates/:id", h.UpdateTemplate)
	r.DELETE("/templates/:id", h.DeleteTemplate)
	r.GET("/tasks", h.ListTasks)
	r.POST("/tasks", h.CreateTask)
	r.GET("/tasks/:id", h.GetTask)
	r.PUT("/tasks/:id", h.UpdateTask)
	r.POST("/tasks/:id/start", h.StartTask)
	r.POST("/tasks/:id/cancel", h.CancelTask)
	r.POST("/task-hosts/:id/result", h.MarkHostResult)
	r.POST("/task-hosts/:id/report", h.MarkHostResult)
}

func (h *DeployHandler) ListTemplates(c *gin.Context) {
	var query TemplateQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.Fail(c, common.CodeParamInvalid, "common.param_invalid")
		return
	}
	resp, err := h.svc.ListTemplates(query)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "deploytemplate.list_failed")
		return
	}
	common.Success(c, resp)
}

func (h *DeployHandler) CreateTemplate(c *gin.Context) {
	common.SetAuditMetadata(c, "新增任务模板", common.BusinessInsert)
	var req CreateTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, "common.param_invalid")
		return
	}
	resp, err := h.svc.CreateTemplate(req, currentActor(c))
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "deploytemplate.create_failed")
		return
	}
	common.Success(c, resp)
}

func (h *DeployHandler) GetTemplate(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	resp, err := h.svc.GetTemplate(id)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "deploytemplate.not_found")
		return
	}
	common.Success(c, resp)
}

func (h *DeployHandler) UpdateTemplate(c *gin.Context) {
	common.SetAuditMetadata(c, "编辑任务模板", common.BusinessUpdate)
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	var req UpdateTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, "common.param_invalid")
		return
	}
	resp, err := h.svc.UpdateTemplate(id, req, currentActor(c))
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "deploytemplate.update_failed")
		return
	}
	common.Success(c, resp)
}

func (h *DeployHandler) DeleteTemplate(c *gin.Context) {
	common.SetAuditMetadata(c, "删除任务模板", common.BusinessDelete)
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	if err := h.svc.DeleteTemplate(id); err != nil {
		common.FailWithError(c, common.CodeError, err, "deploytemplate.delete_failed")
		return
	}
	common.Success(c, nil)
}

func (h *DeployHandler) ListPackages(c *gin.Context) {
	var query PackageQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.Fail(c, common.CodeParamInvalid, "common.param_invalid")
		return
	}
	resp, err := h.svc.ListPackages(query)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "deploypackage.list_failed")
		return
	}
	common.Success(c, resp)
}

func (h *DeployHandler) CreatePackage(c *gin.Context) {
	common.SetAuditMetadata(c, "新增软件组件", common.BusinessInsert)
	var req CreatePackageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, "common.param_invalid")
		return
	}
	resp, err := h.svc.CreatePackage(req, currentActor(c))
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "deploypackage.create_failed")
		return
	}
	common.Success(c, resp)
}

func (h *DeployHandler) GetPackage(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	resp, err := h.svc.GetPackage(id)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "deploypackage.not_found")
		return
	}
	common.Success(c, resp)
}

func (h *DeployHandler) UpdatePackage(c *gin.Context) {
	common.SetAuditMetadata(c, "编辑软件组件", common.BusinessUpdate)
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	var req UpdatePackageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, "common.param_invalid")
		return
	}
	resp, err := h.svc.UpdatePackage(id, req, currentActor(c))
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "deploypackage.update_failed")
		return
	}
	common.Success(c, resp)
}

func (h *DeployHandler) DeletePackage(c *gin.Context) {
	common.SetAuditMetadata(c, "删除软件组件", common.BusinessDelete)
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	if err := h.svc.DeletePackage(id); err != nil {
		common.FailWithError(c, common.CodeError, err, "deploypackage.delete_failed")
		return
	}
	common.Success(c, nil)
}

func (h *DeployHandler) ListTasks(c *gin.Context) {
	var query TaskQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.Fail(c, common.CodeParamInvalid, "common.param_invalid")
		return
	}
	resp, err := h.svc.ListTasks(query)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "deploytask.list_failed")
		return
	}
	common.Success(c, resp)
}

func (h *DeployHandler) CreateTask(c *gin.Context) {
	common.SetAuditMetadata(c, "新增部署任务", common.BusinessInsert)
	var req CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, "common.param_invalid")
		return
	}
	resp, err := h.svc.CreateTask(req, currentActor(c), common.GetDataScope(c))
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "deploytask.create_failed")
		return
	}
	common.Success(c, resp)
}

func (h *DeployHandler) GetTask(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	resp, err := h.svc.GetTask(id)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "deploytask.not_found")
		return
	}
	common.Success(c, resp)
}

func (h *DeployHandler) UpdateTask(c *gin.Context) {
	common.SetAuditMetadata(c, "编辑部署任务", common.BusinessUpdate)
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	var req UpdateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, "common.param_invalid")
		return
	}
	resp, err := h.svc.UpdateTask(id, req, currentActor(c))
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "deploytask.update_failed")
		return
	}
	common.Success(c, resp)
}

func (h *DeployHandler) StartTask(c *gin.Context) {
	common.SetAuditMetadata(c, "启动部署任务", common.BusinessUpdate)
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	var req StartTaskRequest
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			common.Fail(c, common.CodeParamInvalid, "common.param_invalid")
			return
		}
	}
	resp, err := h.svc.StartTask(id, req, currentActor(c), common.GetDataScope(c))
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "deploytask.start_failed")
		return
	}
	common.Success(c, resp)
}

func (h *DeployHandler) CancelTask(c *gin.Context) {
	common.SetAuditMetadata(c, "取消部署任务", common.BusinessUpdate)
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	resp, err := h.svc.CancelTask(id, currentActor(c))
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "deploytask.cancel_failed")
		return
	}
	common.Success(c, resp)
}

func (h *DeployHandler) MarkHostResult(c *gin.Context) {
	common.SetAuditMetadata(c, "标记部署结果", common.BusinessUpdate)
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	var req MarkHostResultRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, "common.param_invalid")
		return
	}
	resp, err := h.svc.MarkHostResult(id, req, currentActor(c))
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "deploytask.result_failed")
		return
	}
	common.Success(c, resp)
}

func parseIDParam(c *gin.Context) (uint64, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, "common.param_invalid")
		return 0, false
	}
	return id, true
}

func currentActor(c *gin.Context) string {
	return strconv.FormatUint(common.GetUserID(c), 10)
}
