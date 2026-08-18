package namespace

import (
	"strconv"

	"pantheon-base/pkg/common"

	"github.com/gin-gonic/gin"
)

const msgParamInvalid = "common.param_invalid"

// NamespaceHandler exposes Kubernetes namespace HTTP endpoints.
//
//nolint:revive // retained as the public handler name for this package.
type NamespaceHandler struct {
	svc *NamespaceService
}

// NewNamespaceHandler creates the namespace HTTP handler.
func NewNamespaceHandler(svc *NamespaceService) *NamespaceHandler {
	return &NamespaceHandler{svc: svc}
}

// RegisterRoutes registers namespace endpoints.
func (h *NamespaceHandler) RegisterRoutes(r gin.IRoutes) {
	r.GET("/clusters/:id/namespaces", h.List)
	r.POST("/clusters/:id/namespaces", h.Create)
	r.DELETE("/clusters/:id/namespaces/:name", h.Delete)
}

// List handles namespace listing.
func (h *NamespaceHandler) List(c *gin.Context) {
	clusterID, err := strconv.ParseUint(c.Param("id"), 10, 64)
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

// Create handles namespace creation.
func (h *NamespaceHandler) Create(c *gin.Context) {
	common.SetAuditMetadata(c, "k8s.namespace.audit.create", common.BusinessInsert)

	clusterID, err := strconv.ParseUint(c.Param("id"), 10, 64)
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

// Delete handles namespace deletion.
func (h *NamespaceHandler) Delete(c *gin.Context) {
	common.SetAuditMetadata(c, "k8s.namespace.audit.delete", common.BusinessDelete)

	clusterID, err := strconv.ParseUint(c.Param("id"), 10, 64)
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
