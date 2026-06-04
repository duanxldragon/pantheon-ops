package config

import (
	"strconv"
	"strings"

	"pantheon-ops/backend/pkg/common"
	"pantheon-ops/backend/pkg/impexp"

	"github.com/gin-gonic/gin"
)

type DictHandler struct {
	service *DictService
}

func NewDictHandler(service *DictService) *DictHandler {
	return &DictHandler{service: service}
}

func (h *DictHandler) GetDictTypeList(c *gin.Context) {
	var query DictTypeListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
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
	common.SetAuditMetadata(c, "新增字典类型", common.BusinessInsert)
	var req DictTypeCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}
	row, err := h.service.CreateDictType(&req)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "request.failed")
		return
	}
	common.Success(c, row)
}

func (h *DictHandler) UpdateDictType(c *gin.Context) {
	common.SetAuditMetadata(c, "编辑字典类型", common.BusinessUpdate)
	var req DictTypeUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}
	typeID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}
	row, err := h.service.UpdateDictType(typeID, &req)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "request.failed")
		return
	}
	common.Success(c, row)
}

func (h *DictHandler) DeleteDictType(c *gin.Context) {
	common.SetAuditMetadata(c, "删除字典类型", common.BusinessDelete)
	typeID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}
	if err := h.service.DeleteDictType(typeID); err != nil {
		common.FailWithError(c, common.CodeError, err, "request.failed")
		return
	}
	common.Success(c, gin.H{"deleted": true})
}

func (h *DictHandler) BatchUpdateDictTypeStatus(c *gin.Context) {
	common.SetAuditMetadata(c, "批量更新字典类型状态", common.BusinessUpdate)
	var req DictTypeBatchStatusReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}
	updatedCount, err := h.service.BatchUpdateDictTypeStatus(req.TypeIDs, req.Status)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "request.failed")
		return
	}
	common.Success(c, gin.H{"updatedCount": updatedCount})
}

func (h *DictHandler) BatchDeleteDictTypes(c *gin.Context) {
	common.SetAuditMetadata(c, "批量删除字典类型", common.BusinessDelete)
	var req common.BatchDeleteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}
	resp := common.BatchDelete(req.IDs, h.service.DeleteDictType)
	common.Success(c, resp)
}

func (h *DictHandler) GetDictItemList(c *gin.Context) {
	var query DictItemListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
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
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
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
	common.SetAuditMetadata(c, "新增字典项", common.BusinessInsert)
	var req DictItemCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}
	row, err := h.service.CreateDictItem(&req)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "request.failed")
		return
	}
	common.Success(c, row)
}

func (h *DictHandler) UpdateDictItem(c *gin.Context) {
	common.SetAuditMetadata(c, "编辑字典项", common.BusinessUpdate)
	var req DictItemUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}
	itemID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}
	row, err := h.service.UpdateDictItem(itemID, &req)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "request.failed")
		return
	}
	common.Success(c, row)
}

func (h *DictHandler) DeleteDictItem(c *gin.Context) {
	common.SetAuditMetadata(c, "删除字典项", common.BusinessDelete)
	itemID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}
	if err := h.service.DeleteDictItem(itemID); err != nil {
		common.FailWithError(c, common.CodeError, err, "request.failed")
		return
	}
	common.Success(c, gin.H{"deleted": true})
}

func (h *DictHandler) BatchUpdateDictItemStatus(c *gin.Context) {
	common.SetAuditMetadata(c, "批量更新字典项状态", common.BusinessUpdate)
	var req DictItemBatchStatusReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}
	updatedCount, err := h.service.BatchUpdateDictItemStatus(req.ItemIDs, req.Status)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "request.failed")
		return
	}
	common.Success(c, gin.H{"updatedCount": updatedCount})
}

func (h *DictHandler) BatchDeleteDictItems(c *gin.Context) {
	common.SetAuditMetadata(c, "批量删除字典项", common.BusinessDelete)
	var req common.BatchDeleteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}
	resp := common.BatchDelete(req.IDs, h.service.DeleteDictItem)
	common.Success(c, resp)
}

func (h *DictHandler) ReorderDictItem(c *gin.Context) {
	common.SetAuditMetadata(c, "调整字典项排序", common.BusinessUpdate)
	itemID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}
	var req DictItemReorderReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}
	row, err := h.service.ReorderDictItem(itemID, req.Direction)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "request.failed")
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
	common.SetAuditMetadata(c, "刷新字典缓存", common.BusinessUpdate)
	var req DictCacheRefreshReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
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
	common.SetAuditMetadata(c, "导出字典类型", common.BusinessExport)

	var query DictTypeListQuery
	if err := c.ShouldBindJSON(&query); err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
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
	common.SetAuditMetadata(c, "导入字典类型", common.BusinessImport)

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
	common.SetAuditMetadata(c, "导出字典项", common.BusinessExport)

	var query DictItemListQuery
	if err := c.ShouldBindJSON(&query); err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
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
	common.SetAuditMetadata(c, "导入字典项", common.BusinessImport)

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
