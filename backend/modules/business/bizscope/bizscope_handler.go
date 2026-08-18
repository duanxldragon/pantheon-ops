package bizscope

import (
	"strconv"

	"pantheon-base/pkg/common"
	"pantheon-base/pkg/impexp"

	"github.com/gin-gonic/gin"
)

const (
	msgParamInvalid        = "param.invalid"
	bizScopeAuditUpdateKey = "business.bizscope.audit.update"
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
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	items, err := h.service.List(&query, common.GetDataScope(c))
	if err != nil {
		failBizScopeError(c, err)
		return
	}
	common.Success(c, items)
}

func (h *Handler) Options(c *gin.Context) {
	items, err := h.service.ListOptions(common.GetDataScope(c))
	if err != nil {
		failBizScopeError(c, err)
		return
	}
	common.Success(c, items)
}

func (h *Handler) Detail(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	item, serviceErr := h.service.Get(id, common.GetDataScope(c))
	if serviceErr != nil {
		failBizScopeError(c, serviceErr)
		return
	}
	common.Success(c, item)
}

func (h *Handler) Hosts(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	items, serviceErr := h.service.ListBoundHosts(id, common.GetDataScope(c))
	if serviceErr != nil {
		failBizScopeError(c, serviceErr)
		return
	}
	common.Success(c, items)
}

func (h *Handler) AvailableHosts(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	items, serviceErr := h.service.ListAvailableHosts(id, common.GetDataScope(c))
	if serviceErr != nil {
		failBizScopeError(c, serviceErr)
		return
	}
	common.Success(c, items)
}

func (h *Handler) BindHosts(c *gin.Context) {
	common.SetAuditMetadata(c, bizScopeAuditUpdateKey, common.BusinessUpdate)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	var req BindBizScopeHostsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	if serviceErr := h.service.BindHosts(id, req.HostIDs, common.GetDataScope(c)); serviceErr != nil {
		failBizScopeError(c, serviceErr)
		return
	}
	common.Success(c, gin.H{"bound": true})
}

func (h *Handler) UnbindHost(c *gin.Context) {
	common.SetAuditMetadata(c, bizScopeAuditUpdateKey, common.BusinessUpdate)
	scopeID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	hostID, err := strconv.ParseUint(c.Param("hostId"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	if serviceErr := h.service.UnbindHost(scopeID, hostID, common.GetDataScope(c)); serviceErr != nil {
		failBizScopeError(c, serviceErr)
		return
	}
	common.Success(c, gin.H{"unbound": true})
}

func (h *Handler) Create(c *gin.Context) {
	common.SetAuditMetadata(c, "business.bizscope.audit.create", common.BusinessInsert)
	var req CreateBizScopeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	item, serviceErr := h.service.Create(&req, common.GetDataScope(c))
	if serviceErr != nil {
		failBizScopeError(c, serviceErr)
		return
	}
	common.Success(c, item)
}

func (h *Handler) Update(c *gin.Context) {
	common.SetAuditMetadata(c, bizScopeAuditUpdateKey, common.BusinessUpdate)
	var req UpdateBizScopeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	item, serviceErr := h.service.Update(id, &req, common.GetDataScope(c))
	if serviceErr != nil {
		failBizScopeError(c, serviceErr)
		return
	}
	common.Success(c, item)
}

func (h *Handler) Delete(c *gin.Context) {
	common.SetAuditMetadata(c, "business.bizscope.audit.delete", common.BusinessDelete)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	if serviceErr := h.service.Delete(id, common.GetDataScope(c)); serviceErr != nil {
		failBizScopeError(c, serviceErr)
		return
	}
	common.Success(c, gin.H{"deleted": true})
}

func (h *Handler) Export(c *gin.Context) {
	common.SetAuditMetadata(c, "bizscope.export.title", common.BusinessExport)

	var query BizScopeListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	file, err := h.service.Export(query, common.GetDataScope(c))
	if err != nil {
		common.Fail(c, common.CodeError, "bizscope.export_failed")
		return
	}
	_ = impexp.WriteCSV(c, *file)
}

func (h *Handler) DownloadImportTemplate(c *gin.Context) {
	file := h.service.BuildImportTemplate()
	_ = impexp.WriteCSV(c, *file)
}

func (h *Handler) Import(c *gin.Context) {
	common.SetAuditMetadata(c, "bizscope.import.title", common.BusinessImport)

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
	result, err := h.service.Import(records, createdBy)
	if err != nil {
		common.Fail(c, common.CodeError, "bizscope.import_failed")
		return
	}
	common.Success(c, result)
}

func failBizScopeError(c *gin.Context, err error) {
	common.Fail(c, common.CodeError, resolveBizScopeErrorKey(err))
}

func resolveBizScopeErrorKey(err error) string {
	switch message := common.ErrMessage(err); message {
	case bizScopeCodeExistsKey, bizScopeInUseKey, bizScopeNotFoundKey, msgParamInvalid:
		return message
	default:
		return "request.failed"
	}
}
