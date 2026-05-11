package generator

import (
	"pantheon-ops/backend/pkg/common"
	"strings"

	"github.com/gin-gonic/gin"
)

type GeneratorHandler struct {
	service *GeneratorService
}

func NewGeneratorHandler(service *GeneratorService) *GeneratorHandler {
	return &GeneratorHandler{service: service}
}

func (h *GeneratorHandler) ListDatasources(c *gin.Context) {
	items, err := h.service.ListDatasources()
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "generator.datasource.list.error")
		return
	}
	common.Success(c, items)
}

func (h *GeneratorHandler) CreateDatasource(c *gin.Context) {
	common.SetAuditMetadata(c, "新增生成器数据源", common.BusinessInsert)

	var req UpsertGeneratorDatasourceReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}
	item, err := h.service.CreateDatasource(&req)
	if err != nil {
		common.FailWithError(c, mapGeneratorErrorCode(err), err, "generator.datasource.save.error")
		return
	}
	common.Success(c, item)
}

func (h *GeneratorHandler) UpdateDatasource(c *gin.Context) {
	common.SetAuditMetadata(c, "更新生成器数据源", common.BusinessUpdate)

	var req UpsertGeneratorDatasourceReq
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, "param.invalid")
		return
	}
	item, err := h.service.UpdateDatasource(c.Param("id"), &req)
	if err != nil {
		common.FailWithError(c, mapGeneratorErrorCode(err), err, "generator.datasource.save.error")
		return
	}
	common.Success(c, item)
}

func (h *GeneratorHandler) DeleteDatasource(c *gin.Context) {
	common.SetAuditMetadata(c, "删除生成器数据源", common.BusinessDelete)

	if err := h.service.DeleteDatasource(c.Param("id")); err != nil {
		common.FailWithError(c, mapGeneratorErrorCode(err), err, "generator.datasource.delete.error")
		return
	}
	common.Success(c, gin.H{"deleted": true})
}

func (h *GeneratorHandler) TestDatasource(c *gin.Context) {
	common.SetAuditMetadata(c, "测试生成器数据源", common.BusinessOther)

	item, err := h.service.TestDatasource(c.Param("id"))
	if err != nil {
		common.FailWithError(c, mapGeneratorErrorCode(err), err, "generator.datasource.test.error")
		return
	}
	common.Success(c, item)
}

func (h *GeneratorHandler) ListTables(c *gin.Context) {
	items, err := h.service.ListTables(c.Query("datasourceId"), c.Query("keyword"))
	if err != nil {
		common.FailWithError(c, mapGeneratorErrorCode(err), err, "generator.table.list.error")
		return
	}
	common.Success(c, items)
}

func (h *GeneratorHandler) PreviewTable(c *gin.Context) {
	tableName := strings.TrimSpace(c.Query("tableName"))
	if tableName == "" {
		common.Fail(c, common.CodeParamInvalid, "generator.table.required")
		return
	}
	preview, err := h.service.PreviewTable(c.Query("datasourceId"), tableName)
	if err != nil {
		common.FailWithError(c, mapGeneratorErrorCode(err), err, "generator.table.preview.error")
		return
	}
	common.Success(c, preview)
}

func mapGeneratorErrorCode(err error) int {
	switch err.Error() {
	case "generator.table.required",
		"generator.table.invalid",
		"generator.table.not_found",
		"generator.datasource.required",
		"generator.datasource.host_invalid",
		"generator.datasource.host_private_disabled",
		"generator.datasource.password_required",
		"generator.datasource.port_invalid",
		"generator.datasource.not_found",
		"generator.datasource.disabled",
		"generator.datasource.driver_unsupported":
		return common.CodeParamInvalid
	default:
		return common.CodeError
	}
}
