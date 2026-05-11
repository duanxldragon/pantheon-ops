package system

import (
	"encoding/json"
	"errors"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"pantheon-ops/backend/pkg/common"
	"pantheon-ops/backend/pkg/impexp"
)

type I18nHandler struct {
	service *I18nService
}

func NewI18nHandler(s *I18nService) *I18nHandler {
	return &I18nHandler{service: s}
}

// GetLangPack 获取语言包接口 (供前端初始化使用)
func (h *I18nHandler) GetLangPack(c *gin.Context) {
	locale := c.DefaultQuery("locale", "zh-CN")
	pack, err := h.service.GetLangPack(locale)
	if err != nil {
		common.Fail(c, common.CodeError, "i18n.fetch.error")
		return
	}
	common.Success(c, pack)
}

func (h *I18nHandler) GetOverview(c *gin.Context) {
	resp, err := h.service.GetOverview()
	if err != nil {
		common.Fail(c, common.CodeError, "i18n.overview.error")
		return
	}
	common.Success(c, resp)
}

func (h *I18nHandler) GetAudit(c *gin.Context) {
	resp, err := h.service.GetAudit()
	if err != nil {
		common.Fail(c, common.CodeError, "i18n.audit.error")
		return
	}
	common.Success(c, resp)
}

func (h *I18nHandler) CleanupUnusedKeys(c *gin.Context) {
	common.SetAuditMetadata(c, "清理未使用国际化键", common.BusinessDelete)
	resp, err := h.service.CleanupUnusedKeys(c.Query("module"))
	if err != nil {
		common.Fail(c, common.CodeError, "i18n.cleanup_unused.error")
		return
	}
	common.Success(c, resp)
}

func (h *I18nHandler) StartUnusedObservation(c *gin.Context) {
	common.SetAuditMetadata(c, "标记国际化键进入观察期", common.BusinessUpdate)
	module := c.Query("module")
	setI18nLifecycleAuditParam(c, "observe", strings.TrimSpace(module), I18nLifecycleStatusActive, I18nLifecycleStatusObserving, false)
	resp, err := h.service.StartUnusedObservation(module)
	if err != nil {
		common.Fail(c, common.CodeError, "i18n.lifecycle.observe.error")
		return
	}
	setI18nLifecycleAuditResult(c, "observe", I18nLifecycleStatusActive, I18nLifecycleStatusObserving, resp)
	common.Success(c, resp)
}

func (h *I18nHandler) ArchiveObservedUnusedKeys(c *gin.Context) {
	common.SetAuditMetadata(c, "归档观察期国际化键", common.BusinessUpdate)
	module := c.Query("module")
	setI18nLifecycleAuditParam(c, "archive", strings.TrimSpace(module), I18nLifecycleStatusObserving, I18nLifecycleStatusArchived, false)
	resp, err := h.service.ArchiveObservedUnusedKeys(module)
	if err != nil {
		common.Fail(c, common.CodeError, "i18n.lifecycle.archive.error")
		return
	}
	setI18nLifecycleAuditResult(c, "archive", I18nLifecycleStatusObserving, I18nLifecycleStatusArchived, resp)
	common.Success(c, resp)
}

func (h *I18nHandler) DeleteArchivedUnusedKeys(c *gin.Context) {
	common.SetAuditMetadata(c, "删除已归档国际化键", common.BusinessDelete)
	var req I18nUnusedLifecycleReq
	if err := c.ShouldBindJSON(&req); err != nil && err.Error() != "EOF" {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}
	if req.Module == "" {
		req.Module = c.Query("module")
	}
	req.Module = strings.TrimSpace(req.Module)
	setI18nLifecycleAuditParam(c, "delete", req.Module, I18nLifecycleStatusArchived, "deleted", req.ConfirmArchived)
	resp, err := h.service.DeleteArchivedUnusedKeys(req.Module, req.ConfirmArchived)
	if err != nil {
		if err.Error() == "i18n.lifecycle.delete.confirm_required" {
			common.FailWithError(c, common.CodeParamInvalid, err, "param.invalid")
			return
		}
		common.Fail(c, common.CodeError, "i18n.lifecycle.delete.error")
		return
	}
	setI18nLifecycleAuditResult(c, "delete", I18nLifecycleStatusArchived, "deleted", resp)
	common.Success(c, resp)
}

func setI18nLifecycleAuditParam(c *gin.Context, action string, module string, fromStatus string, toStatus string, confirmArchived bool) {
	payload := map[string]interface{}{
		"action":                   action,
		"module":                   module,
		"fromStatus":               fromStatus,
		"toStatus":                 toStatus,
		"observationThresholdDays": I18nUnusedObservationThresholdDays,
	}
	if action == "delete" {
		payload["confirmArchived"] = confirmArchived
	}
	writeI18nLifecycleAuditJSON(common.SetAuditParam, c, payload)
}

func setI18nLifecycleAuditResult(c *gin.Context, action string, fromStatus string, toStatus string, resp *I18nUnusedLifecycleResp) {
	if resp == nil {
		return
	}
	payload := map[string]interface{}{
		"code":    common.CodeSuccess,
		"message": "success",
		"data": map[string]interface{}{
			"action":       action,
			"module":       resp.Module,
			"fromStatus":   fromStatus,
			"toStatus":     toStatus,
			"affectedRows": resp.AffectedRows,
			"affectedKeys": resp.AffectedKeys,
		},
	}
	writeI18nLifecycleAuditJSON(common.SetAuditResult, c, payload)
}

func writeI18nLifecycleAuditJSON(writer func(*gin.Context, string), c *gin.Context, payload map[string]interface{}) {
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}
	writer(c, string(data))
}

func (h *I18nHandler) PreviewRenameKey(c *gin.Context) {
	var req I18nRenamePreviewReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}
	resp, err := h.service.PreviewRenameKey(&req)
	if err != nil {
		switch err.Error() {
		case "i18n.rename.invalid":
			common.FailWithError(c, common.CodeParamInvalid, err, "param.invalid")
		case "i18n.rename.source_not_found":
			common.FailWithError(c, common.CodeError, err, "i18n.rename.preview.error")
		default:
			common.Fail(c, common.CodeError, "i18n.rename.preview.error")
		}
		return
	}
	common.Success(c, resp)
}

func (h *I18nHandler) RenameKey(c *gin.Context) {
	common.SetAuditMetadata(c, "重命名国际化键", common.BusinessUpdate)
	var req I18nRenameExecuteReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}
	resp, err := h.service.RenameKey(&req)
	if err != nil {
		switch err.Error() {
		case "i18n.rename.invalid", "i18n.rename.source_not_found":
			common.FailWithError(c, common.CodeParamInvalid, err, "param.invalid")
		case "i18n.rename.target_exists", "i18n.rename.source_not_confirmed":
			common.FailWithError(c, common.CodeError, err, "i18n.rename.execute.error")
		default:
			common.Fail(c, common.CodeError, "i18n.rename.execute.error")
		}
		return
	}
	common.Success(c, resp)
}

func (h *I18nHandler) GetMissingLocales(c *gin.Context) {
	resp, err := h.service.ListMissingLocales(c.Query("module"))
	if err != nil {
		common.Fail(c, common.CodeError, "i18n.missing_locales.error")
		return
	}
	common.Success(c, resp)
}

func (h *I18nHandler) FillMissingLocales(c *gin.Context) {
	common.SetAuditMetadata(c, "补齐缺失国际化语言", common.BusinessInsert)
	resp, err := h.service.FillMissingLocales(c.Query("module"))
	if err != nil {
		common.Fail(c, common.CodeError, "i18n.fill_missing_locales.error")
		return
	}
	common.Success(c, resp)
}

func (h *I18nHandler) HydrateBuiltinLocales(c *gin.Context) {
	common.SetAuditMetadata(c, "回填内置国际化翻译", common.BusinessUpdate)
	resp, err := h.service.HydrateBuiltinLocales(c.Query("module"))
	if err != nil {
		common.Fail(c, common.CodeError, "i18n.hydrate_builtin.error")
		return
	}
	common.Success(c, resp)
}

// List 翻译列表 (管理端)
func (h *I18nHandler) List(c *gin.Context) {
	var query I18nQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}
	resp, err := h.service.List(&query)
	if err != nil {
		common.Fail(c, common.CodeError, "i18n.list.error")
		return
	}
	common.Success(c, resp)
}

func (h *I18nHandler) Get(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		common.Fail(c, common.CodeParamInvalid, "id.invalid")
		return
	}
	resp, err := h.service.Get(id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			common.Fail(c, common.CodeError, "i18n.not_found")
			return
		}
		common.Fail(c, common.CodeError, "i18n.detail.error")
		return
	}
	common.Success(c, resp)
}

func (h *I18nHandler) Create(c *gin.Context) {
	common.SetAuditMetadata(c, "新增翻译", common.BusinessInsert)
	var req I18nCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}
	resp, err := h.service.Create(&req)
	if err != nil {
		switch err.Error() {
		case "i18n.create.invalid":
			common.FailWithError(c, common.CodeParamInvalid, err, "param.invalid")
		case "i18n.key.duplicate":
			common.FailWithError(c, common.CodeError, err, "i18n.create.error")
		default:
			common.Fail(c, common.CodeError, "i18n.create.error")
		}
		return
	}
	common.Success(c, resp)
}

// Update 更新翻译 (管理端)
func (h *I18nHandler) Update(c *gin.Context) {
	common.SetAuditMetadata(c, "编辑翻译", common.BusinessUpdate)
	idStr := c.Param("id")
	id, _ := strconv.ParseUint(idStr, 10, 64)
	if id == 0 {
		common.Fail(c, common.CodeParamInvalid, "id.invalid")
		return
	}

	var req I18nUpdateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}

	if err := h.service.Update(id, &req); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			common.Fail(c, common.CodeError, "i18n.not_found")
			return
		}
		if err.Error() == "i18n.value.required" {
			common.FailWithError(c, common.CodeParamInvalid, err, "param.invalid")
			return
		}
		common.Fail(c, common.CodeError, "i18n.update.error")
		return
	}
	common.Success(c, nil)
}

func (h *I18nHandler) Delete(c *gin.Context) {
	common.SetAuditMetadata(c, "删除翻译", common.BusinessDelete)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		common.Fail(c, common.CodeParamInvalid, "id.invalid")
		return
	}
	if err := h.service.Delete(id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			common.Fail(c, common.CodeError, "i18n.not_found")
			return
		}
		common.Fail(c, common.CodeError, "i18n.delete.error")
		return
	}
	common.Success(c, gin.H{"deleted": true})
}

func (h *I18nHandler) DeleteBatch(c *gin.Context) {
	common.SetAuditMetadata(c, "批量删除翻译", common.BusinessDelete)
	var req I18nBatchDeleteReq
	if err := c.ShouldBindJSON(&req); err != nil || len(req.IDs) == 0 {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}
	if err := h.service.DeleteBatch(req.IDs); err != nil {
		common.Fail(c, common.CodeError, "i18n.delete.error")
		return
	}
	common.Success(c, gin.H{"deleted": true, "count": len(req.IDs)})
}

func (h *I18nHandler) Export(c *gin.Context) {
	common.SetAuditMetadata(c, "导出翻译", common.BusinessExport)
	var query I18nQuery
	if err := c.ShouldBindJSON(&query); err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}
	file, err := h.service.Export(&query)
	if err != nil {
		common.Fail(c, common.CodeError, "i18n.export.error")
		return
	}
	if err := impexp.WriteCSV(c, *file); err != nil {
		common.Fail(c, common.CodeError, "i18n.export.error")
	}
}

func (h *I18nHandler) DownloadImportTemplate(c *gin.Context) {
	file := h.service.BuildImportTemplate()
	if err := impexp.WriteCSV(c, *file); err != nil {
		common.Fail(c, common.CodeError, "i18n.import.template.error")
	}
}

func (h *I18nHandler) Import(c *gin.Context) {
	common.SetAuditMetadata(c, "导入翻译", common.BusinessImport)

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
	result, err := h.service.Import(records)
	if err != nil {
		common.Fail(c, common.CodeError, "i18n.import.error")
		return
	}
	common.Success(c, result)
}

// SyncMissingKeys 一键同步缺失的错误码 Key
func (h *I18nHandler) SyncMissingKeys(c *gin.Context) {
	resp, err := h.service.SyncMissingKeys()
	if err != nil {
		common.Fail(c, common.CodeError, "i18n.sync.error")
		return
	}
	common.Success(c, resp)
}

// ReloadCache 手动刷新缓存 (管理端)
func (h *I18nHandler) ReloadCache(c *gin.Context) {
	var req I18nCacheRefreshReq
	if err := c.ShouldBindJSON(&req); err != nil && err.Error() != "EOF" {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}
	if err := h.service.ReloadLocales(req.Locales); err != nil {
		common.Fail(c, common.CodeError, "i18n.refresh.error")
		return
	}
	common.Success(c, nil)
}
