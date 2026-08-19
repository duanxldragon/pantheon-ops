package deploy

import (
	"errors"
	"mime/multipart"
	"strconv"
	"strings"

	"pantheon-base/pkg/common"
	"pantheon-base/pkg/impexp"

	"github.com/gin-gonic/gin"
)

const (
	msgParamInvalid = "common.param_invalid"
	packageIDRoute  = "/packages/:id"
	templateIDRoute = "/templates/:id"
	taskIDRoute     = "/tasks/:id"
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
	r.GET(packageIDRoute, h.GetPackage)
	r.PUT(packageIDRoute, h.UpdatePackage)
	r.DELETE(packageIDRoute, h.DeletePackage)
	r.GET("/packages/export", h.ExportPackages)
	r.POST("/packages/import", h.ImportPackages)
	r.GET("/templates", h.ListTemplates)
	r.POST("/templates", h.CreateTemplate)
	r.GET(templateIDRoute, h.GetTemplate)
	r.PUT(templateIDRoute, h.UpdateTemplate)
	r.DELETE(templateIDRoute, h.DeleteTemplate)
	r.GET("/templates/export", h.ExportTemplates)
	r.POST("/templates/import", h.ImportTemplates)
	r.GET("/credentials", h.ListCredentials)
	r.POST("/credentials", h.CreateCredential)
	r.PUT("/credentials/:id", h.UpdateCredential)
	r.DELETE("/credentials/:id", h.DeleteCredential)
	r.GET("/tasks", h.ListTasks)
	r.POST("/tasks", h.CreateTask)
	r.GET(taskIDRoute, h.GetTask)
	r.PUT(taskIDRoute, h.UpdateTask)
	r.DELETE(taskIDRoute, h.DeleteTask)
	r.POST("/tasks/:id/start", h.StartTask)
	r.POST("/tasks/:id/cancel", h.CancelTask)
	r.GET("/tasks/export", h.ExportTasks)
	r.POST("/task-hosts/:id/result", h.MarkHostResult)
	r.POST("/task-hosts/:id/report", h.MarkHostResult)
}

// ListCredentials handles the credential reference metadata list endpoint.
func (h *DeployHandler) ListCredentials(c *gin.Context) {
	items, err := h.svc.ListCredentials()
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "deploycredential.list_failed")
		return
	}
	common.Success(c, items)
}

// CreateCredential handles credential reference creation.
func (h *DeployHandler) CreateCredential(c *gin.Context) {
	var req CreateDeployCredentialRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	item, err := h.svc.CreateCredential(req, currentActor(c))
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "deploycredential.create_failed")
		return
	}
	common.Success(c, item)
}

// UpdateCredential handles credential reference metadata updates and rotations.
func (h *DeployHandler) UpdateCredential(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	var req UpdateDeployCredentialRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	item, err := h.svc.UpdateCredential(id, req, currentActor(c))
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "deploycredential.update_failed")
		return
	}
	common.Success(c, item)
}

// DeleteCredential handles credential reference deletion.
func (h *DeployHandler) DeleteCredential(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	if err := h.svc.DeleteCredential(id); err != nil {
		common.FailWithError(c, common.CodeError, err, "deploycredential.delete_failed")
		return
	}
	common.Success(c, nil)
}

// ExportPackages handles CSV export for deploy packages.
func (h *DeployHandler) ExportPackages(c *gin.Context) {
	var query PackageQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	file, err := h.svc.ExportPackages(query)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "deploypackage.export_failed")
		return
	}
	if err := impexp.WriteCSV(c, *file); err != nil {
		common.FailWithError(c, common.CodeError, err, "deploypackage.export_failed")
	}
}

// ImportPackages handles CSV import for deploy packages.
func (h *DeployHandler) ImportPackages(c *gin.Context) {
	file, err := multipartFile(c)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, err.Error())
		return
	}
	records, err := impexp.ReadCSV(file)
	if err != nil {
		common.FailWithError(c, common.CodeParamInvalid, err, "deploypackage.import_failed")
		return
	}
	result, err := h.svc.ImportPackages(records, currentActor(c))
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "deploypackage.import_failed")
		return
	}
	common.Success(c, result)
}

// ExportTemplates handles CSV export for deploy templates.
func (h *DeployHandler) ExportTemplates(c *gin.Context) {
	var query TemplateQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	file, err := h.svc.ExportTemplates(query)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "deploytemplate.export_failed")
		return
	}
	if err := impexp.WriteCSV(c, *file); err != nil {
		common.FailWithError(c, common.CodeError, err, "deploytemplate.export_failed")
	}
}
// ImportTemplates handles CSV import for deploy templates.
func (h *DeployHandler) ImportTemplates(c *gin.Context) {
	file, err := multipartFile(c)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, err.Error())
		return
	}
	records, err := impexp.ReadCSV(file)
	if err != nil {
		common.FailWithError(c, common.CodeParamInvalid, err, "deploytemplate.import_failed")
		return
	}
	result, err := h.svc.ImportTemplates(records, currentActor(c))
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "deploytemplate.import_failed")
		return
	}
	common.Success(c, result)
}
// ExportTasks handles CSV export for deploy tasks.
func (h *DeployHandler) ExportTasks(c *gin.Context) {
	var query TaskQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	file, err := h.svc.ExportTasks(query, common.GetDataScope(c))
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "deploytask.export_failed")
		return
	}
	if err := impexp.WriteCSV(c, *file); err != nil {
		common.FailWithError(c, common.CodeError, err, "deploytask.export_failed")
	}
}

func multipartFile(c *gin.Context) (multipart.File, error) {
	file, _, err := c.Request.FormFile("file")
	if err != nil {
		return nil, errors.New("import.file.required")
	}
	return file, nil
}

func (h *DeployHandler) ListTemplates(c *gin.Context) {
	var query TemplateQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
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
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
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
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
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
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
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
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
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
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
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
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	resp, err := h.svc.ListTasks(query, common.GetDataScope(c))
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
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	resp, err := h.svc.CreateTask(req, currentActor(c), common.GetDataScope(c))
	if err != nil {
		failDeployTaskError(c, err, "deploytask.create_failed")
		return
	}
	common.Success(c, resp)
}

func (h *DeployHandler) GetTask(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	resp, err := h.svc.GetTask(id, common.GetDataScope(c))
	if err != nil {
		switch {
		case isDeployTaskForbidden(err):
			common.FailWithCode(c, common.CodeForbidden, err.Error())
		case isDeployTaskNotFound(err):
			common.FailWithCode(c, common.CodeNotFound, err.Error())
		default:
			common.FailWithError(c, common.CodeError, err, "deploytask.not_found")
		}
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
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	resp, err := h.svc.UpdateTask(id, req, currentActor(c), common.GetDataScope(c))
	if err != nil {
		failDeployTaskError(c, err, "deploytask.update_failed")
		return
	}
	common.Success(c, resp)
}

func (h *DeployHandler) DeleteTask(c *gin.Context) {
	common.SetAuditMetadata(c, "删除部署任务", common.BusinessDelete)
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	if err := h.svc.DeleteTask(id, currentActor(c), common.GetDataScope(c)); err != nil {
		failDeployTaskError(c, err, "deploytask.delete_failed")
		return
	}
	common.Success(c, nil)
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
			common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
			return
		}
	}
	resp, err := h.svc.StartTask(id, req, currentActor(c), common.GetDataScope(c))
	if err != nil {
		failDeployTaskError(c, err, "deploytask.start_failed")
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
	resp, err := h.svc.CancelTask(id, currentActor(c), common.GetDataScope(c))
	if err != nil {
		failDeployTaskError(c, err, "deploytask.cancel_failed")
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
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	resp, err := h.svc.MarkHostResult(id, req, currentActor(c), common.GetDataScope(c))
	if err != nil {
		failDeployTaskHostError(c, err, "deploytask.result_failed")
		return
	}
	common.Success(c, resp)
}

func failDeployTaskError(c *gin.Context, err error, fallback string) {
	switch {
	case isDeployTaskForbidden(err):
		common.FailWithCode(c, common.CodeForbidden, err.Error())
	case isDeployTaskNotFound(err):
		common.FailWithCode(c, common.CodeNotFound, err.Error())
	case deployTaskConflictError(err):
		common.FailWithCode(c, 409, err.Error())
	case deployTaskValidationError(err):
		common.FailWithCode(c, common.CodeParamInvalid, err.Error())
	default:
		common.FailWithError(c, common.CodeError, err, fallback)
	}
}

func failDeployTaskHostError(c *gin.Context, err error, fallback string) {
	switch {
	case strings.TrimSpace(errorText(err)) == "business.deploy.taskHost.notFound":
		common.FailWithCode(c, common.CodeNotFound, err.Error())
	case strings.TrimSpace(errorText(err)) == "business.deploy.taskHost.invalidResultState":
		common.FailWithCode(c, 409, err.Error())
	case strings.TrimSpace(errorText(err)) == errDeployTaskHostStaleReport:
		common.FailWithCode(c, 409, err.Error())
	case strings.TrimSpace(errorText(err)) == "business.deploy.taskHost.markFailed.reasonRequired":
		common.FailWithCode(c, common.CodeParamInvalid, err.Error())
	default:
		failDeployTaskError(c, err, fallback)
	}
}

func deployTaskConflictError(err error) bool {
	switch strings.TrimSpace(errorText(err)) {
	case "business.deploy.task.invalidStartState",
		"business.deploy.task.invalidCancelState",
		errDeployTaskInvalidUpdateState,
		errDeployTaskInvalidDeleteState,
		errDeployTaskAlreadyRunning,
		errDeployTaskLeaseConflict,
		errDeployTaskHostStaleReport,
		errDeployPackageImmutable,
		errDeployTemplateImmutable:
		return true
	default:
		return false
	}
}

func deployTaskValidationError(err error) bool {
	switch strings.TrimSpace(errorText(err)) {
	case "business.deploy.task.nameRequired",
		"business.deploy.task.packageRequired",
		"business.deploy.task.packageDisabled",
		"business.deploy.task.scopeRequired",
		"business.deploy.task.scopeInvalid",
		"business.deploy.task.targetRequired",
		"business.deploy.task.invalidTargetType",
		"business.deploy.task.targetOutOfScope",
		"business.deploy.task.invalidExecutorType",
		"business.deploy.task.invalidAction",
		"business.deploy.task.targetStatusMismatch",
		errDeployTaskTemplateNotFound,
		errDeployTaskTemplateDisabled,
		errDeployTaskPackageNotFound,
		"business.deploy.task.emptyResolvedTargets",
		errDeployTaskSnapshotMissing,
		errDeployTaskTemplateParamsInvalid,
		errDeployTaskTemplateInvalid,
		errDeployTaskInstallCommandRequired,
		errDeployTaskUninstallCommandRequired,
		errDeployTaskPackageSourceMissing,
		errDeployTaskSSHHostKeyRequired,
		errDeployTaskSSHUserRequired,
		errDeployTaskSSHPasswordRequired,
		errDeployTaskSSHPrivateKeyRequired:
		return true
	case errDeployTaskSSHHostKeyMismatch,
		errDeployTaskSSHAuthFailed,
		errDeployTaskSSHConnectFailed:
		return true
	default:
		return false
	}
}

func parseIDParam(c *gin.Context) (uint64, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return 0, false
	}
	return id, true
}

func currentActor(c *gin.Context) string {
	return strconv.FormatUint(common.GetUserID(c), 10)
}
