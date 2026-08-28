package service

import (
	"strconv"

	"pantheon-base/modules/business/k8s/rbac"
	"pantheon-base/pkg/common"

	"github.com/gin-gonic/gin"
)

// ServiceHandler exposes Kubernetes Service HTTP endpoints.
//
//nolint:revive // retained as the public handler name for this package.
type ServiceHandler struct {
	svc     *ServiceService
	rbacSvc *rbac.K8sRBACService
}

// NewServiceHandler creates the Service HTTP handler.
func NewServiceHandler(svc *ServiceService, rbacSvc *rbac.K8sRBACService) *ServiceHandler {
	return &ServiceHandler{svc: svc, rbacSvc: rbacSvc}
}

// RegisterRoutes registers Service endpoints.
func (h *ServiceHandler) RegisterRoutes(r gin.IRoutes) {
	r.GET("/clusters/:id/services", h.List)
	r.POST("/clusters/:id/services", h.Create)
	r.GET("/clusters/:id/services/:name", h.Get)
	r.DELETE("/clusters/:id/services/:name", h.Delete)
}

// List handles Service listing.
func (h *ServiceHandler) List(c *gin.Context) {
	clusterID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	var query ServiceListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	resp, err := h.svc.List(clusterID, query, common.GetDataScope(c))
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "k8s.service.list_failed")
		return
	}
	common.Success(c, resp)
}

// Get handles Service retrieval.
func (h *ServiceHandler) Get(c *gin.Context) {
	clusterID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	resp, err := h.svc.Get(clusterID, c.Query("namespace"), c.Param("name"), common.GetDataScope(c))
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "k8s.service.not_found")
		return
	}
	common.Success(c, resp)
}

// Create handles Service creation.
func (h *ServiceHandler) Create(c *gin.Context) {
	common.SetAuditMetadata(c, "k8s.service.audit.create", common.BusinessInsert)

	clusterID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	var req CreateServiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	namespace := c.Query("namespace")
	if namespace == "" {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	if h.rbacSvc != nil && !h.rbacSvc.CheckHandlerPermission(c, clusterID, namespace, rbac.ResourceTypeService, rbac.ActionCreate) {
		common.Fail(c, common.CodeForbidden, "k8s.rbac.permission_denied")
		return
	}
	resp, err := h.svc.Create(clusterID, namespace, req, common.GetDataScope(c))
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "k8s.service.create_failed")
		return
	}
	common.Success(c, resp)
}

// Delete handles Service deletion.
func (h *ServiceHandler) Delete(c *gin.Context) {
	common.SetAuditMetadata(c, "k8s.service.audit.delete", common.BusinessDelete)

	clusterID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	namespace := c.Query("namespace")
	if namespace == "" {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	if h.rbacSvc != nil && !h.rbacSvc.CheckHandlerPermission(c, clusterID, namespace, rbac.ResourceTypeService, rbac.ActionDelete) {
		common.Fail(c, common.CodeForbidden, "k8s.rbac.permission_denied")
		return
	}
	if err := h.svc.Delete(clusterID, namespace, c.Param("name"), common.GetDataScope(c)); err != nil {
		common.FailWithError(c, common.CodeError, err, "k8s.service.delete_failed")
		return
	}
	common.Success(c, nil)
}
