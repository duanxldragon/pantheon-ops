package dynamicmodule

import (
	"bytes"
	"encoding/json"
	"io"
	"pantheon-ops/backend/internal/scaffold"
	"pantheon-ops/backend/pkg/common"

	"github.com/gin-gonic/gin"
)

type DynamicModuleHandler struct {
	service *DynamicModuleService
}

func NewDynamicModuleHandler(s *DynamicModuleService) *DynamicModuleHandler {
	return &DynamicModuleHandler{service: s}
}

// RegisterModule 注册模块
func (h *DynamicModuleHandler) RegisterModule(c *gin.Context) {
	common.SetAuditMetadata(c, "注册动态模块", common.BusinessInsert)

	var req struct {
		Name string `json:"name" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}

	registration, err := h.service.RegisterManagedModule(req.Name)
	if err != nil {
		code := common.CodeError
		switch err.Error() {
		case "module.invalid_name", "module.register.source_missing", "module.register.schema_invalid":
			code = common.CodeParamInvalid
		}
		common.FailWithError(c, code, err, "module.register.error")
		return
	}

	common.Success(c, gin.H{
		"registered": true,
		"module":     registration,
		"message":    "module.register.success",
	})
}

func (h *DynamicModuleHandler) GenerateAndRegisterModule(c *gin.Context) {
	common.SetAuditMetadata(c, "一键生成并注册模块", common.BusinessInsert)

	rawBody, err := io.ReadAll(c.Request.Body)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}
	c.Request.Body = io.NopCloser(bytes.NewReader(rawBody))

	var input struct {
		Schema    scaffold.ModuleSchema `json:"schema"`
		Overwrite bool                  `json:"overwrite"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}
	applyGenerateSchemaRawMetadata(rawBody, &input.Schema)

	req := scaffold.RegisterGeneratedModuleRequest{
		Schema:    input.Schema,
		Overwrite: input.Overwrite,
	}

	registration, writtenFiles, summary, err := h.service.RegisterGeneratedModule(&req)
	if err != nil {
		code := common.CodeError
		if isGenerateValidationError(err) {
			code = common.CodeParamInvalid
		}
		common.FailWithError(c, code, err, "module.generate.error")
		return
	}

	common.Success(c, gin.H{
		"module":                registration,
		"summary":               summary,
		"writtenFiles":          writtenFiles,
		"requiresRestart":       true,
		"requiresFrontendBuild": true,
		"message":               "module.generate.success",
	})
}

// UnregisterModule 卸载模块
func (h *DynamicModuleHandler) UnregisterModule(c *gin.Context) {
	common.SetAuditMetadata(c, "卸载动态模块", common.BusinessDelete)

	moduleName := c.Param("name")
	dropTable := c.Query("dropTable") == "true"
	purgeSource := c.Query("purgeSource") == "true"

	if err := h.service.UnregisterModule(moduleName, dropTable, purgeSource); err != nil {
		common.FailWithError(c, common.CodeError, err, "module.unregister.error")
		return
	}

	common.Success(c, gin.H{
		"unregistered": true,
		"message":      "module.unregistered",
	})
}

func (h *DynamicModuleHandler) DeleteModuleRecord(c *gin.Context) {
	common.SetAuditMetadata(c, "删除动态模块记录", common.BusinessDelete)

	if err := h.service.DeleteModuleRecord(c.Param("name")); err != nil {
		common.FailWithError(c, common.CodeError, err, "module.delete_record.error")
		return
	}
	common.Success(c, gin.H{
		"deleted": true,
		"message": "module.record.deleted",
	})
}

func (h *DynamicModuleHandler) PurgeModule(c *gin.Context) {
	common.SetAuditMetadata(c, "彻底删除动态模块", common.BusinessDelete)

	moduleName := c.Param("name")
	dropTable := c.Query("dropTable") == "true"
	purgeSource := c.Query("purgeSource") != "false"

	if err := h.service.PurgeModule(moduleName, dropTable, purgeSource); err != nil {
		common.FailWithError(c, common.CodeError, err, "module.purge.error")
		return
	}
	common.Success(c, gin.H{
		"deleted": true,
		"message": "module.deleted",
	})
}

func (h *DynamicModuleHandler) RepairRegistries(c *gin.Context) {
	common.SetAuditMetadata(c, "执行模块注册表自检修复", common.BusinessUpdate)

	summary, err := h.service.AuditAndRepairGeneratedRegistries()
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "module.registry.repair.error")
		return
	}
	common.Success(c, gin.H{
		"repaired": true,
		"summary":  summary,
		"message":  "module.registry.repaired",
	})
}

func (h *DynamicModuleHandler) AuditPendingActivations(c *gin.Context) {
	common.SetAuditMetadata(c, "执行模块激活检查", common.BusinessOther)

	summary, err := h.service.AuditPendingGeneratedModuleActivations()
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "module.activation.audit.error")
		return
	}
	common.Success(c, gin.H{
		"audited": true,
		"summary": summary,
		"message": "module.activation.audit.success",
	})
}

func (h *DynamicModuleHandler) GetModuleSchema(c *gin.Context) {
	moduleName := c.Query("module")
	if moduleName == "" {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}

	schema, err := h.service.GetManagedModuleSchema(moduleName)
	if err != nil {
		code := common.CodeError
		switch err.Error() {
		case "module.invalid_name", "module.register.source_missing", "module.register.schema_invalid":
			code = common.CodeParamInvalid
		}
		common.FailWithError(c, code, err, "module.schema.error")
		return
	}

	common.Success(c, schema)
}

// ListModules 获取模块列表
func (h *DynamicModuleHandler) ListModules(c *gin.Context) {
	modules, err := h.service.ListRegisteredModules()
	if err != nil {
		common.Fail(c, common.CodeError, "module.list.error")
		return
	}

	common.Success(c, modules)
}

// GetModuleStatus 获取模块状态
func (h *DynamicModuleHandler) GetModuleStatus(c *gin.Context) {
	moduleName := c.Param("name")

	module, err := h.service.GetModuleStatus(moduleName)
	if err != nil {
		common.Fail(c, common.CodeError, "module.status.error")
		return
	}

	common.Success(c, module)
}

func applyGenerateSchemaRawMetadata(rawBody []byte, schema *scaffold.ModuleSchema) {
	if schema == nil || len(rawBody) == 0 {
		return
	}

	var raw struct {
		Schema struct {
			Metadata map[string]json.RawMessage `json:"metadata"`
		} `json:"schema"`
	}
	if err := json.Unmarshal(rawBody, &raw); err != nil {
		return
	}

	if encoded, ok := raw.Schema.Metadata["autoRecycle"]; ok {
		var autoRecycle bool
		if err := json.Unmarshal(encoded, &autoRecycle); err == nil {
			schema.Metadata.AutoRecycle = autoRecycle
		}
	}
}

func isGenerateValidationError(err error) bool {
	switch err.Error() {
	case "module.generate.invalid_payload",
		"module.generate.invalid_name",
		"module.generate.invalid_scope",
		"module.generate.display_name_required",
		"module.generate.table_name_required",
		"module.generate.invalid_table_name",
		"module.generate.empty_files",
		"module.generate.invalid_path",
		"module.generate.duplicate_file",
		"module.generate.file_exists",
		"module.generate.already_exists",
		"module.generate.business_only",
		"module.invalid_name":
		return true
	default:
		return false
	}
}
