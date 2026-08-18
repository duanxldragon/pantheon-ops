package label

import (
	"strconv"

	"pantheon-base/pkg/common"
	"pantheon-base/pkg/impexp"

	"github.com/gin-gonic/gin"
)

const msgParamInvalid = "common.param_invalid"

type LabelHandler struct {
	svc *LabelService
}

func NewLabelHandler(svc *LabelService) *LabelHandler {
	return &LabelHandler{svc: svc}
}

func (h *LabelHandler) RegisterRoutes(r gin.IRoutes) {
	r.GET("/labels", h.List)
	r.GET("/labels/options", h.ListOptions)
	r.GET("/labels/export", h.Export)
	r.GET("/labels/import-template", h.DownloadImportTemplate)
	r.POST("/labels/import", h.Import)
	r.POST("/labels", h.Create)
	r.PUT("/labels/:id", h.Update)
	r.DELETE("/labels/:id", h.Delete)
}

func (h *LabelHandler) List(c *gin.Context) {
	var query LabelSchemaQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	items, err := h.svc.List(query)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "cmdblabel.list_failed")
		return
	}
	common.Success(c, items)
}

func (h *LabelHandler) ListOptions(c *gin.Context) {
	var query LabelSchemaQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	items, err := h.svc.ListOptions(query)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "cmdblabel.list_failed")
		return
	}
	common.Success(c, items)
}

func (h *LabelHandler) Create(c *gin.Context) {
	var req CreateLabelSchemaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	resp, err := h.svc.Create(req)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "cmdblabel.create_failed")
		return
	}
	common.Success(c, resp)
}

func (h *LabelHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	var req UpdateLabelSchemaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	resp, err := h.svc.Update(id, req)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "cmdblabel.update_failed")
		return
	}
	common.Success(c, resp)
}

func (h *LabelHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	if err := h.svc.Delete(id); err != nil {
		common.FailWithError(c, common.CodeError, err, "cmdblabel.delete_failed")
		return
	}
	common.Success(c, nil)
}

// Export writes visible CMDB label schemas as CSV.
func (h *LabelHandler) Export(c *gin.Context) {
	common.SetAuditMetadata(c, "cmdb.label.export.title", common.BusinessExport)

	var query LabelSchemaQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}

	file, err := h.svc.Export(query, common.GetDataScope(c))
	if err != nil {
		common.Fail(c, common.CodeError, "cmdblabel.export_failed")
		return
	}
	_ = impexp.WriteCSV(c, *file)
}

// DownloadImportTemplate writes the CMDB label import template.
func (h *LabelHandler) DownloadImportTemplate(c *gin.Context) {
	file := h.svc.BuildImportTemplate()
	_ = impexp.WriteCSV(c, *file)
}

// Import creates or updates CMDB label schemas from an uploaded CSV.
func (h *LabelHandler) Import(c *gin.Context) {
	common.SetAuditMetadata(c, "cmdb.label.import.title", common.BusinessImport)

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
	result, err := h.svc.Import(records, createdBy)
	if err != nil {
		common.Fail(c, common.CodeError, "cmdblabel.import_failed")
		return
	}
	common.Success(c, result)
}
