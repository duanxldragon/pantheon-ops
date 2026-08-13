package config

import (
	"strconv"
	"strings"

	"pantheon-base/pkg/common"
	"pantheon-base/pkg/impexp"

	"github.com/gin-gonic/gin"
)

const errRequestFailed = "request.failed"
const errParamInvalid = "param.invalid"

type DictHandler struct {
	service *DictService
}

func NewDictHandler(service *DictService) *DictHandler {
	return &DictHandler{service: service}
}

func (h *DictHandler) GetDictTypeList(c *gin.Context) {
	var query DictTypeListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return
	}
	rows, err := h.service.ListDictTypes(&query)
	if err != nil {
		common.Fail(c, common.CodeError, "dict.type.list.error")
		return
	}
	common.Success(c, rows)
}

func (h *DictHandler) CreateDictType(c *gin.Context) {
	common.SetAuditMetadata(c, "dict.type.create.title", common.BusinessInsert)
	var req DictTypeCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return
	}
	row, err := h.service.CreateDictType(&req)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, errRequestFailed)
		return
	}
	common.Success(c, row)
}

func (h *DictHandler) UpdateDictType(c *gin.Context) {
	common.SetAuditMetadata(c, "dict.type.update.title", common.BusinessUpdate)
	var req DictTypeUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return
	}
	typeID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return
	}
	row, err := h.service.UpdateDictType(typeID, &req)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, errRequestFailed)
		return
	}
	common.Success(c, row)
}

func (h *DictHandler) DeleteDictType(c *gin.Context) {
	common.SetAuditMetadata(c, "dict.type.delete.title", common.BusinessDelete)
	typeID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return
	}
	if err := h.service.DeleteDictType(typeID); err != nil {
		common.FailWithError(c, common.CodeError, err, errRequestFailed)
		return
	}
	common.Success(c, gin.H{"deleted": true})
}

func (h *DictHandler) BatchUpdateDictTypeStatus(c *gin.Context) {
	common.SetAuditMetadata(c, "dict.type.batch_status.title", common.BusinessUpdate)
	var req DictTypeBatchStatusReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return
	}
	updatedCount, err := h.service.BatchUpdateDictTypeStatus(req.TypeIDs, req.Status)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, errRequestFailed)
		return
	}
	common.Success(c, gin.H{"updatedCount": updatedCount})
}

func (h *DictHandler) BatchDeleteDictTypes(c *gin.Context) {
	common.SetAuditMetadata(c, "dict.type.batch_delete.title", common.BusinessDelete)
	var req common.BatchDeleteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return
	}
	resp := common.BatchDelete(req.IDs, h.service.DeleteDictType)
	common.Success(c, resp)
}

func (h *DictHandler) GetDictItemList(c *gin.Context) {
	var query DictItemListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return
	}
	rows, err := h.service.ListDictItems(&query)
	if err != nil {
		common.Fail(c, common.CodeError, "dict.item.list.error")
		return
	}
	common.Success(c, rows)
}

func (h *DictHandler) AnalyzeDictUsage(c *gin.Context) {
	dictCode := strings.TrimSpace(c.Query("dictCode"))
	if dictCode == "" {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return
	}
	resp, err := h.service.AnalyzeDictUsage(dictCode)
	if err != nil {
		common.Fail(c, common.CodeError, "dict.usage.error")
		return
	}
	common.Success(c, resp)
}

func (h *DictHandler) CreateDictItem(c *gin.Context) {
	common.SetAuditMetadata(c, "dict.item.create.title", common.BusinessInsert)
	var req DictItemCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return
	}
	row, err := h.service.CreateDictItem(&req)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, errRequestFailed)
		return
	}
	common.Success(c, row)
}

func (h *DictHandler) UpdateDictItem(c *gin.Context) {
	common.SetAuditMetadata(c, "dict.item.update.title", common.BusinessUpdate)
	var req DictItemUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return
	}
	itemID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return
	}
	row, err := h.service.UpdateDictItem(itemID, &req)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, errRequestFailed)
		return
	}
	common.Success(c, row)
}

func (h *DictHandler) DeleteDictItem(c *gin.Context) {
	common.SetAuditMetadata(c, "dict.item.delete.title", common.BusinessDelete)
	itemID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return
	}
	if err := h.service.DeleteDictItem(itemID); err != nil {
		common.FailWithError(c, common.CodeError, err, errRequestFailed)
		return
	}
	common.Success(c, gin.H{"deleted": true})
}

func (h *DictHandler) BatchUpdateDictItemStatus(c *gin.Context) {
	common.SetAuditMetadata(c, "dict.item.batch_status.title", common.BusinessUpdate)
	var req DictItemBatchStatusReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return
	}
	updatedCount, err := h.service.BatchUpdateDictItemStatus(req.ItemIDs, req.Status)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, errRequestFailed)
		return
	}
	common.Success(c, gin.H{"updatedCount": updatedCount})
}

func (h *DictHandler) BatchDeleteDictItems(c *gin.Context) {
	common.SetAuditMetadata(c, "dict.item.batch_delete.title", common.BusinessDelete)
	var req common.BatchDeleteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return
	}
	resp := common.BatchDelete(req.IDs, h.service.DeleteDictItem)
	common.Success(c, resp)
}

func (h *DictHandler) ReorderDictItem(c *gin.Context) {
	common.SetAuditMetadata(c, "dict.item.sort.title", common.BusinessUpdate)
	itemID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return
	}
	var req DictItemReorderReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return
	}
	row, err := h.service.ReorderDictItem(itemID, req.Direction)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, errRequestFailed)
		return
	}
	common.Success(c, row)
}

func (h *DictHandler) GetDictOptions(c *gin.Context) {
	rawCodes := strings.TrimSpace(c.Query("codes"))
	if rawCodes == "" {
		common.Success(c, DictOptionMapResp{})
		return
	}
	rows, err := h.service.GetDictOptions(strings.Split(rawCodes, ","))
	if err != nil {
		common.Fail(c, common.CodeError, "dict.options.error")
		return
	}
	common.Success(c, rows)
}

func (h *DictHandler) RefreshDictOptionsCache(c *gin.Context) {
	common.SetAuditMetadata(c, "dict.cache.refresh.title", common.BusinessUpdate)
	var req DictCacheRefreshReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return
	}
	resp, err := h.service.RefreshDictOptionsCache(req.Codes)
	if err != nil {
		common.Fail(c, common.CodeError, "dict.cache.refresh.error")
		return
	}
	common.Success(c, resp)
}

func (h *DictHandler) ExportDictTypes(c *gin.Context) {
	common.SetAuditMetadata(c, "dict.type.export.title", common.BusinessExport)

	var query DictTypeListQuery
	if err := c.ShouldBindJSON(&query); err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return
	}
	file, err := h.service.ExportDictTypes(&query)
	if err != nil {
		common.Fail(c, common.CodeError, "dict.type.export.error")
		return
	}
	if err := impexp.WriteCSV(c, *file); err != nil {
		common.Fail(c, common.CodeError, "dict.type.export.error")
	}
}

func (h *DictHandler) DownloadDictTypeImportTemplate(c *gin.Context) {
	file := h.service.BuildDictTypeImportTemplate()
	if err := impexp.WriteCSV(c, *file); err != nil {
		common.Fail(c, common.CodeError, "dict.type.import_template.error")
	}
}

func (h *DictHandler) ImportDictTypes(c *gin.Context) {
	common.SetAuditMetadata(c, "dict.type.import.title", common.BusinessImport)

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
	result, err := h.service.ImportDictTypes(records)
	if err != nil {
		common.Fail(c, common.CodeError, "dict.type.import.error")
		return
	}
	common.Success(c, result)
}

func (h *DictHandler) ExportDictItems(c *gin.Context) {
	common.SetAuditMetadata(c, "dict.item.export.title", common.BusinessExport)

	var query DictItemListQuery
	if err := c.ShouldBindJSON(&query); err != nil {
		common.Fail(c, common.CodeParamInvalid, errParamInvalid)
		return
	}
	file, err := h.service.ExportDictItems(&query)
	if err != nil {
		common.Fail(c, common.CodeError, "dict.item.export.error")
		return
	}
	if err := impexp.WriteCSV(c, *file); err != nil {
		common.Fail(c, common.CodeError, "dict.item.export.error")
	}
}

func (h *DictHandler) DownloadDictItemImportTemplate(c *gin.Context) {
	file := h.service.BuildDictItemImportTemplate()
	if err := impexp.WriteCSV(c, *file); err != nil {
		common.Fail(c, common.CodeError, "dict.item.import_template.error")
	}
}

func (h *DictHandler) ImportDictItems(c *gin.Context) {
	common.SetAuditMetadata(c, "dict.item.import.title", common.BusinessImport)

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
	result, err := h.service.ImportDictItems(records)
	if err != nil {
		common.Fail(c, common.CodeError, "dict.item.import.error")
		return
	}
	common.Success(c, result)
}
