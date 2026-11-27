package deploy

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"strconv"
	"strings"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/modules/business/cmdb"
	"github.com/duanxldragon/pantheon-base/backend/modules/business/shared/dashboard"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/impexp"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	msgParamInvalid = "common.param_invalid"
	// msgTaskEventsFailed is the visible error key for event-log retrieval.
	msgTaskEventsFailed = "deploytask.events_failed"
	packageIDRoute      = "/packages/:id"
	templateIDRoute     = "/templates/:id"
	taskIDRoute         = "/tasks/:id"
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
	r.GET("/credentials/scope-options", h.ListCredentialScopeOptions)
	r.POST("/credentials", h.CreateCredential)
	r.PUT("/credentials/:id", h.UpdateCredential)
	r.DELETE("/credentials/:id", h.DeleteCredential)
	r.GET("/tasks", h.ListTasks)
	r.POST("/tasks", h.CreateTask)
	r.POST("/targets/resolve", h.ResolveTargetSelection)
	r.POST("/tasks/:id/preview", h.PreviewTask)
	r.GET(taskIDRoute, h.GetTask)
	r.PUT(taskIDRoute, h.UpdateTask)
	r.DELETE(taskIDRoute, h.DeleteTask)
	r.POST("/tasks/:id/start", h.StartTask)
	r.POST("/tasks/:id/pause", h.PauseTask)
	r.POST("/tasks/:id/resume", h.ResumeTask)
	r.POST("/tasks/:id/retry", h.RetryTask)
	r.POST("/tasks/:id/cancel", h.CancelTask)
	r.POST("/tasks/:id/rollback", h.RollbackTask)
	r.GET("/tasks/:id/rollback", h.GetRollbackInfo)
	r.GET("/tasks/export", h.ExportTasks)
	r.POST("/task-hosts/:id/result", h.MarkHostResult)
	r.POST("/task-hosts/:id/report", h.MarkHostResult)
	r.GET("/tasks/:id/events", h.StreamTaskEvents)
	r.GET("/tasks/:id/events/history", h.ListTaskEvents)
	r.GET("/tasks/:id/logs", h.ListTaskLogs)
}

// ListCredentials handles the credential reference metadata list endpoint.
func (h *DeployHandler) ListCredentials(c *gin.Context) {
	items, err := h.svc.ListCredentials(common.GetDataScope(c))
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "deploycredential.list_failed")
		return
	}
	common.Success(c, items)
}

// ListCredentialScopeOptions returns the caller's eligible business scope owners.
func (h *DeployHandler) ListCredentialScopeOptions(c *gin.Context) {
	items, err := h.svc.listCredentialScopeOptions(c.Request.Context(), common.GetDataScope(c))
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
	item, err := h.svc.CreateCredential(req, currentActor(c), common.GetDataScope(c))
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
	item, err := h.svc.UpdateCredential(id, req, currentActor(c), common.GetDataScope(c))
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
	if err := h.svc.DeleteCredential(id, common.GetDataScope(c)); err != nil {
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
	if c.Query("dashboard") == "true" {
		resp, err := h.svc.DashboardSummary(c.Request.Context(), dashboard.Limit(c.Query("limit")), c.Query("status"), common.GetDataScope(c))
		if err != nil {
			common.FailWithError(c, common.CodeError, err, "deploytask.list_failed")
			return
		}
		common.Success(c, resp)
		return
	}
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
	resp, err := h.svc.CreateTaskContext(c.Request.Context(), req, currentActor(c), common.GetDataScope(c))
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
	resp, err := h.svc.UpdateTaskContext(c.Request.Context(), id, req, currentActor(c), common.GetDataScope(c))
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
	req.IdempotencyKey = strings.TrimSpace(c.GetHeader("X-Idempotency-Key"))
	req.RequirePreview = true
	resp, err := h.svc.StartTaskContext(c.Request.Context(), id, req, currentActor(c), common.GetDataScope(c))
	if err != nil {
		failDeployTaskError(c, err, "deploytask.start_failed")
		return
	}
	common.Success(c, resp)
}

// PauseTask suspends the deployment task identified by the path id.
func (h *DeployHandler) PauseTask(c *gin.Context) {
	common.SetAuditMetadata(c, "暂停部署任务", common.BusinessUpdate)
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	var req TaskCommandRequest
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	resp, err := h.svc.ControlTask(c.Request.Context(), id, currentActor(c), common.GetDataScope(c), "pause", req)
	if err != nil {
		failDeployTaskError(c, err, "deploytask.pause_failed")
		return
	}
	common.Success(c, resp)
}

// ResumeTask continues the deployment task identified by the path id.
func (h *DeployHandler) ResumeTask(c *gin.Context) {
	common.SetAuditMetadata(c, "恢复部署任务", common.BusinessUpdate)
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	var req TaskCommandRequest
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	resp, err := h.svc.ControlTask(c.Request.Context(), id, currentActor(c), common.GetDataScope(c), "resume", req)
	if err != nil {
		failDeployTaskError(c, err, "deploytask.resume_failed")
		return
	}
	common.Success(c, resp)
}

// RetryTask re-runs the deployment task identified by the path id.
func (h *DeployHandler) RetryTask(c *gin.Context) {
	common.SetAuditMetadata(c, "重试部署任务", common.BusinessUpdate)
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	var req TaskCommandRequest
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	resp, err := h.svc.ControlTask(c.Request.Context(), id, currentActor(c), common.GetDataScope(c), "retry", req)
	if err != nil {
		failDeployTaskError(c, err, "deploytask.retry_failed")
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
	var req TaskCommandRequest
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	resp, err := h.svc.ControlTask(c.Request.Context(), id, currentActor(c), common.GetDataScope(c), "cancel", req)
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
	resp, err := h.svc.targetRequestService(c.Request.Context()).MarkHostResult(id, req, currentActor(c), common.GetDataScope(c))
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
	case ErrTaskCommandConflict.Error(), ErrTaskPreviewConflict.Error(), cmdb.ErrTargetSelectionConflict.Error(),
		stateStaleKey,
		"business.deploy.task.invalidStartState",
		"business.deploy.task.invalidCancelState",
		errDeployTaskInvalidUpdateState,
		errDeployTaskInvalidDeleteState,
		errDeployTaskAlreadyRunning,
		errDeployTaskLeaseConflict,
		errDeployTaskHostStaleReport,
		errDeployPackageImmutable,
		errDeployTemplateImmutable,
		errDeployTaskInvalidPauseState,
		errDeployTaskInvalidResumeState,
		errDeployTaskInvalidRetryState,
		msgDeployAttemptExhausted:
		return true
	default:
		return false
	}
}

func deployTaskValidationError(err error) bool {
	switch strings.TrimSpace(errorText(err)) {
	case ErrTaskCommandVersionRequired.Error(), ErrTaskPreviewRequired.Error(), cmdb.ErrTargetSelectionInvalid.Error(),
		errDeployServiceLinkInvalid,
		"business.deploy.task.nameRequired",
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
		errDeployTaskSSHPrivateKeyRequired,
		"business.deploy.credential.required",
		"business.deploy.credential.not_found",
		"business.deploy.credential.inactive",
		"business.deploy.credential.inline_forbidden",
		"business.deploy.credential.version_changed",
		"business.deploy.credential.auth_mode_invalid",
		"business.deploy.credential.key_missing",
		"business.deploy.credential.key_invalid",
		"business.deploy.credential.malformed",
		"business.deploy.credential.tampered":
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

// --- FZ-04: Event and log endpoints ---

// TaskEventItem is a JSON-serializable event record.
type TaskEventItem struct {
	ID         uint64 `json:"id"`
	Sequence   uint64 `json:"sequence"`
	EventType  string `json:"eventType"`
	FromStatus string `json:"fromStatus"`
	ToStatus   string `json:"toStatus"`
	Actor      string `json:"actor"`
	RequestID  string `json:"requestId"`
	Reason     string `json:"reason"`
	CreatedAt  string `json:"createdAt"`
}

// TaskLogItem is a JSON-serializable log chunk record.
type TaskLogItem struct {
	TaskHostID uint64 `json:"taskHostId"`
	TaskStepID uint64 `json:"taskStepId"`
	ID         uint64 `json:"id"`
	Sequence   uint64 `json:"sequence"`
	Stream     string `json:"stream"`
	Content    string `json:"content"`
	CreatedAt  string `json:"createdAt"`
	AttemptNo  int    `json:"attemptNo"`
}

// ListTaskEvents handles event log retrieval for a task.
func (h *DeployHandler) ListTaskEvents(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	reads := h.svc.targetRequestService(c.Request.Context())
	if _, err := reads.loadVisibleTask(id, common.GetDataScope(c)); err != nil {
		failDeployTaskError(c, err, msgTaskEventsFailed)
		return
	}
	var after uint64
	if v := c.Query("after"); v != "" {
		parsed, err := strconv.ParseUint(v, 10, 64)
		if err != nil {
			common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
			return
		}
		after = parsed
	}
	events, err := reads.loadScopedTaskEvents(id, after, common.GetDataScope(c))
	if err != nil {
		failDeployTaskError(c, err, msgTaskEventsFailed)
		return
	}
	items := make([]TaskEventItem, 0, len(events))
	for _, e := range events {
		items = append(items, TaskEventItem{
			ID: e.ID, Sequence: e.Sequence, EventType: e.EventType,
			FromStatus: e.FromStatus, ToStatus: e.ToStatus,
			Actor: e.Actor, RequestID: e.RequestID, Reason: e.Reason,
			CreatedAt: e.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		})
	}
	common.Success(c, map[string]any{"items": items, "total": len(items)})
}

// ListTaskLogs handles log chunk retrieval for a task.
func (h *DeployHandler) ListTaskLogs(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	reads := h.svc.targetRequestService(c.Request.Context())
	if _, err := reads.loadVisibleTask(id, common.GetDataScope(c)); err != nil {
		failDeployTaskError(c, err, "deploytask.logs_failed")
		return
	}
	after, filter, err := parseTaskLogQuery(c)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	chunks, err := reads.loadScopedLogChunks(id, after, filter, common.GetDataScope(c))
	if err != nil {
		failDeployTaskError(c, err, "deploytask.logs_failed")
		return
	}
	items, clipped, err := reads.safeTaskLogItems(id, chunks)
	if err != nil {
		failDeployTaskError(c, err, "deploytask.logs_failed")
		return
	}
	c.Header("Cache-Control", "no-store")
	common.Success(c, map[string]any{"items": items, "total": len(items), "truncated": clipped, "redaction": "known-task-secrets"})
}

func parseTaskLogQuery(c *gin.Context) (uint64, taskLogFilter, error) {
	var after uint64
	var filter taskLogFilter
	for _, field := range []struct {
		key       string
		target    *uint64
		bits      int
		allowZero bool
	}{
		{"after", &after, 64, true}, {"hostId", &filter.hostID, 64, false},
		{"stepId", &filter.stepID, 64, false}, {"attemptNo", &filter.attemptNo, 31, false},
	} {
		values, present := c.Request.URL.Query()[field.key]
		if !present {
			continue
		}
		if len(values) != 1 {
			return 0, filter, errors.New(msgParamInvalid)
		}
		parsed, err := strconv.ParseUint(values[0], 10, field.bits)
		if err != nil || (!field.allowZero && parsed == 0) {
			return 0, filter, errors.New(msgParamInvalid)
		}
		*field.target = parsed
	}
	if values, present := c.Request.URL.Query()["firstError"]; present {
		if len(values) != 1 || values[0] != "1" || after != 0 {
			return 0, filter, errors.New(msgParamInvalid)
		}
		filter.firstError = true
	}
	return after, filter, nil
}

// StreamTaskEvents streams authenticated, data-scoped task events with replay.
func (h *DeployHandler) StreamTaskEvents(c *gin.Context) {
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	reads := h.svc.targetRequestService(c.Request.Context())
	if _, err := reads.loadVisibleTask(id, common.GetDataScope(c)); err != nil {
		failDeployTaskError(c, err, msgTaskEventsFailed)
		return
	}
	after, err := parseEventSequence(c)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")

	pollTicker := time.NewTicker(time.Second)
	heartbeatTicker := time.NewTicker(15 * time.Second)
	defer pollTicker.Stop()
	defer heartbeatTicker.Stop()

	writeEvents := func() error {
		events, loadErr := reads.loadScopedTaskEvents(id, after, common.GetDataScope(c))
		if loadErr != nil {
			if errors.Is(loadErr, errDeployTaskForbidden) {
				_, _ = fmt.Fprint(c.Writer, "event: forbidden\ndata: {}\n\n")
				c.Writer.Flush()
			}
			return loadErr
		}
		for _, event := range events {
			payload, marshalErr := json.Marshal(event)
			if marshalErr != nil {
				return marshalErr
			}
			if _, writeErr := fmt.Fprintf(c.Writer, "id: %d\nevent: task\ndata: %s\n\n", event.Sequence, payload); writeErr != nil {
				return writeErr
			}
			after = event.Sequence
		}
		c.Writer.Flush()
		return nil
	}

	if err := writeEvents(); err != nil {
		return
	}
	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-heartbeatTicker.C:
			if _, err := fmt.Fprintf(c.Writer, ": heartbeat %d\n\n", time.Now().Unix()); err != nil {
				return
			}
			c.Writer.Flush()
		case <-pollTicker.C:
			if err := writeEvents(); err != nil {
				return
			}
		}
	}
}

func parseEventSequence(c *gin.Context) (uint64, error) {
	value := strings.TrimSpace(c.Query("after"))
	if value == "" {
		value = strings.TrimSpace(c.GetHeader("Last-Event-ID"))
	}
	if value == "" {
		return 0, nil
	}
	return strconv.ParseUint(value, 10, 64)
}

// RollbackTask 创建回滚任务
func (h *DeployHandler) RollbackTask(c *gin.Context) {
	taskIDStr := c.Param("id")
	taskID, err := strconv.ParseUint(taskIDStr, 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}

	common.SetAuditMetadata(c, "创建回滚任务", common.BusinessInsert)

	userID := strconv.FormatUint(common.GetUserID(c), 10)

	rollbackTask, err := h.svc.RollbackTask(c.Request.Context(), taskID, userID, common.GetDataScope(c))
	if err != nil {
		failDeployTaskError(c, err, "business.deploy.task.rollbackFailed")
		return
	}

	common.Success(c, rollbackTask)
}

// GetRollbackInfo 获取任务的回滚信息
func (h *DeployHandler) GetRollbackInfo(c *gin.Context) {
	taskIDStr := c.Param("id")
	taskID, err := strconv.ParseUint(taskIDStr, 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}

	rollbackTask, err := h.svc.GetRollbackTask(c.Request.Context(), taskID, common.GetDataScope(c))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			common.Success(c, nil)
			return
		}
		failDeployTaskError(c, err, "business.deploy.task.rollbackLookupFailed")
		return
	}

	common.Success(c, rollbackTask)
}

// ResolveTargetSelection is a read-only provider resolution; audit stores only
// bounded counts through the response, never secret execution parameters.
func (h *DeployHandler) ResolveTargetSelection(c *gin.Context) {
	common.SetAuditMetadata(c, "解析部署目标", common.BusinessOther)
	var req cmdb.TargetSelection
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	result, err := h.svc.ResolveTargetSelection(c.Request.Context(), req, common.GetDataScope(c))
	if err != nil {
		failDeployTaskError(c, err, "business.cmdb.selection.invalid")
		return
	}
	common.Success(c, result)
}

// PreviewTask handles the deployment-change preview endpoint.
func (h *DeployHandler) PreviewTask(c *gin.Context) {
	common.SetAuditMetadata(c, "预览部署变更", common.BusinessOther)
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
	result, err := h.svc.PreviewTaskContext(c.Request.Context(), id, req, currentActor(c), common.GetDataScope(c))
	if err != nil {
		failDeployTaskError(c, err, "business.deploy.preview.conflict")
		return
	}
	common.Success(c, result)
}
