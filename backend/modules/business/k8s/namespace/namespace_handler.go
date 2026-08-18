package namespace

import (
	"strconv"

	"pantheon-base/pkg/common"

	"github.com/gin-gonic/gin"
)

const msgParamInvalid = "common.param_invalid"

type NamespaceHandler struct {
	svc *NamespaceService
}

func NewNamespaceHandler(svc *NamespaceService) *NamespaceHandler {
	return &NamespaceHandler{svc: svc}
}

func (h *NamespaceHandler) RegisterRoutes(r gin.IRoutes) {
	r.GET("/clusters/:clusterId/namespaces", h.List)
	r.POST("/clusters/:clusterId/namespaces", h.Create)
	r.DELETE("/clusters/:clusterId/namespaces/:name", h.Delete)
}

func (h *NamespaceHandler) List(c *gin.Context) {
	clusterID, err := strconv.ParseUint(c.Param("clusterId"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	resp, err := h.svc.List(clusterID, common.GetDataScope(c))
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "k8s.namespace.list_failed")
		return
	}
	common.Success(c, resp)
}

func (h *NamespaceHandler) Create(c *gin.Context) {
	common.SetAuditMetadata(c, "k8s.namespace.audit.create", common.BusinessInsert)

	clusterID, err := strconv.ParseUint(c.Param("clusterId"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	var req CreateNamespaceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	resp, err := h.svc.Create(clusterID, req, common.GetDataScope(c))
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "k8s.namespace.create_failed")
		return
	}
	common.Success(c, resp)
}

func (h *NamespaceHandler) Delete(c *gin.Context) {
	common.SetAuditMetadata(c, "k8s.namespace.audit.delete", common.BusinessDelete)

	clusterID, err := strconv.ParseUint(c.Param("clusterId"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	name := c.Param("name")
	if err := h.svc.Delete(clusterID, name, common.GetDataScope(c)); err != nil {
		common.FailWithError(c, common.CodeError, err, "k8s.namespace.delete_failed")
		return
	}
	common.Success(c, nil)
}
