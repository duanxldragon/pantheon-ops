package host

import (
	"strconv"

	"pantheon-base/pkg/common"
	"pantheon-base/pkg/impexp"

	"github.com/gin-gonic/gin"
)

const (
	msgParamInvalid = "common.param_invalid"
	hostIDRoute     = "/hosts/:id"
)

type HostHandler struct {
	svc *HostService
}

func NewHostHandler(svc *HostService) *HostHandler {
	return &HostHandler{svc: svc}
}

func (h *HostHandler) RegisterRoutes(r gin.IRoutes) {
	r.GET("/hosts", h.List)
	r.GET("/hosts/export", h.Export)
	r.GET("/hosts/import-template", h.DownloadImportTemplate)
	r.GET(hostIDRoute, h.GetByID)
	r.POST("/hosts", h.Create)
	r.POST("/hosts/import", h.Import)
	r.PUT(hostIDRoute, h.Update)
	r.DELETE(hostIDRoute, h.Delete)
	r.POST("/hosts/:id/collect", h.Collect)
	r.PATCH("/hosts/:id/status", h.UpdateStatus)
	r.PATCH("/hosts/:id/state", h.TransitionState)
}

func (h *HostHandler) List(c *gin.Context) {
	var query HostListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	resp, err := h.svc.List(query, common.GetDataScope(c))
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "cmdbhost.list_failed")
		return
	}
	common.Success(c, resp)
}

func (h *HostHandler) Export(c *gin.Context) {
	common.SetAuditMetadata(c, "cmdb.host.export.title", common.BusinessExport)

	var query HostListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	file, err := h.svc.Export(query, common.GetDataScope(c))
	if err != nil {
		common.Fail(c, common.CodeError, "cmdbhost.export_failed")
		return
	}
	_ = impexp.WriteCSV(c, *file)
}

func (h *HostHandler) DownloadImportTemplate(c *gin.Context) {
	file := h.svc.BuildImportTemplate()
	_ = impexp.WriteCSV(c, *file)
}

func (h *HostHandler) Import(c *gin.Context) {
	common.SetAuditMetadata(c, "cmdb.host.import.title", common.BusinessImport)

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
	records, err := impexp.ReadCSV(file)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, "import.file.invalid_csv")
		return
	}
	createdBy := strconv.FormatUint(common.GetUserID(c), 10)
	result, err := h.svc.Import(records, common.GetDataScope(c), createdBy)
	if err != nil {
		common.Fail(c, common.CodeError, "cmdbhost.import_failed")
		return
	}
	common.Success(c, result)
}

func (h *HostHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
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
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
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
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	var req UpdateHostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
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
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
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
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	var req CollectRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
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
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	var req UpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	if err := h.svc.UpdateStatus(id, req.Status, common.GetDataScope(c)); err != nil {
		common.FailWithError(c, common.CodeError, err, "cmdbhost.status_failed")
		return
	}
	common.Success(c, nil)
}

func (h *HostHandler) TransitionState(c *gin.Context) {
	common.SetAuditMetadata(c, "cmdb.host.state.update.title", common.BusinessUpdate)

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	var req HostStateTransitionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	actor := strconv.FormatUint(common.GetUserID(c), 10)
	resp, err := h.svc.TransitionState(id, req, actor, common.GetDataScope(c))
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "cmdbhost.state.update_failed")
		return
	}
	common.Success(c, resp)
}
