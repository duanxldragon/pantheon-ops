package service

import (
	"net/http"
	"strconv"

	"github.com/duanxldragon/pantheon-base/backend/modules/business/k8s/rbac"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"

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
	r.PUT("/clusters/:id/services/:name", h.Update)
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
	resp, err := h.svc.ListContext(c.Request.Context(), clusterID, query, common.GetDataScope(c))
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
	resp, err := h.svc.GetContext(c.Request.Context(), clusterID, c.Query("namespace"), c.Param("name"), common.GetDataScope(c))
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
	resp, err := h.svc.CreateContext(c.Request.Context(), clusterID, namespace, req, common.GetDataScope(c))
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
	if namespace == "" || c.Query("uid") == "" || c.Query("resourceVersion") == "" {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	if h.rbacSvc != nil && !h.rbacSvc.CheckHandlerPermission(c, clusterID, namespace, rbac.ResourceTypeService, rbac.ActionDelete) {
		common.Fail(c, common.CodeForbidden, "k8s.rbac.permission_denied")
		return
	}
	if err := h.svc.DeleteContext(c.Request.Context(), clusterID, namespace, c.Param("name"), c.Query("uid"), c.Query("resourceVersion"), common.GetDataScope(c)); err != nil {
		failServiceMutation(c, err, "k8s.service.delete_failed")
		return
	}
	common.Success(c, nil)
}

// Update handles a version-bound change to Service routing.
func (h *ServiceHandler) Update(c *gin.Context) {
	common.SetAuditMetadata(c, "business.k8s.service.audit.update", common.BusinessUpdate)
	clusterID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || clusterID == 0 || c.Query("namespace") == "" {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	var req UpdateServiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	if err := validateServiceUpdate(req); err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	ns := c.Query("namespace")
	if h.rbacSvc != nil && !h.rbacSvc.CheckHandlerPermission(c, clusterID, ns, rbac.ResourceTypeService, rbac.ActionUpdate) {
		common.Fail(c, common.CodeForbidden, "k8s.rbac.permission_denied")
		return
	}
	item, err := h.svc.UpdateContext(c.Request.Context(), clusterID, ns, c.Param("name"), req, common.GetDataScope(c))
	if err != nil {
		failServiceMutation(c, err, "business.k8s.service.updateFailed")
		return
	}
	common.Success(c, item)
}

func failServiceMutation(c *gin.Context, err error, fallback string) {
	switch err.Error() {
	case msgServiceConflict:
		common.FailWithCode(c, http.StatusConflict, msgServiceConflict)
	case msgParamInvalid:
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
	case "k8s.rbac.permission_denied", "k8s.namespace.action_forbidden":
		common.Fail(c, common.CodeForbidden, err.Error())
	case "k8s.service.not_found", "k8s.cluster.not_found":
		common.Fail(c, common.CodeNotFound, err.Error())
	default:
		common.FailWithError(c, common.CodeError, err, fallback)
	}
}
