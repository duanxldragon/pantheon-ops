package label

import (
	"strconv"

	"pantheon-ops/backend/pkg/common"

	"github.com/gin-gonic/gin"
)

type LabelHandler struct {
	svc *LabelService
}

func NewLabelHandler(svc *LabelService) *LabelHandler {
	return &LabelHandler{svc: svc}
}

func (h *LabelHandler) RegisterRoutes(r gin.IRoutes) {
	r.GET("/labels", h.List)
	r.POST("/labels", h.Create)
	r.PUT("/labels/:id", h.Update)
	r.DELETE("/labels/:id", h.Delete)
}

func (h *LabelHandler) List(c *gin.Context) {
	var query LabelSchemaQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.Fail(c, common.CodeParamInvalid, "common.param_invalid")
		return
	}
	items, err := h.svc.List(query)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "cmdblabel.list_failed")
		return
	}
	common.Success(c, items)
}

func (h *LabelHandler) Create(c *gin.Context) {
	var req CreateLabelSchemaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, "common.param_invalid")
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
		common.Fail(c, common.CodeParamInvalid, "common.param_invalid")
		return
	}
	var req UpdateLabelSchemaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, "common.param_invalid")
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
		common.Fail(c, common.CodeParamInvalid, "common.param_invalid")
		return
	}
	if err := h.svc.Delete(id); err != nil {
		common.FailWithError(c, common.CodeError, err, "cmdblabel.delete_failed")
		return
	}
	common.Success(c, nil)
}
