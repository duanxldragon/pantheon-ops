package service

import (
	"errors"
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
	resp, err := h.svc.ListWithContext(c.Request.Context(), clusterID, query, common.GetDataScope(c))
	if err != nil {
		failServiceError(c, err, "k8s.service.list_failed")
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
	resp, err := h.svc.GetWithContext(c.Request.Context(), clusterID, c.Query("namespace"), c.Param("name"), common.GetDataScope(c))
	if err != nil {
		failServiceError(c, err, "request.failed")
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
	resp, err := h.svc.CreateWithContext(c.Request.Context(), clusterID, namespace, req, common.GetDataScope(c))
	if err != nil {
		failServiceError(c, err, "k8s.service.create_failed")
		return
	}
	common.Success(c, resp)
}

// Update handles revision-checked structured Service routing changes.
func (h *ServiceHandler) Update(c *gin.Context) {
	common.SetAuditMetadata(c, "k8s.service.audit.update", common.BusinessUpdate)
	clusterID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || clusterID == 0 || c.Query("namespace") == "" {
		common.FailWithCode(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	var req UpdateServiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.FailWithCode(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	namespace := c.Query("namespace")
	if h.rbacSvc != nil && !h.rbacSvc.CheckHandlerPermission(c, clusterID, namespace, rbac.ResourceTypeService, rbac.ActionUpdate) {
		common.FailWithCode(c, common.CodeForbidden, "k8s.rbac.permission_denied")
		return
	}
	resp, err := h.svc.UpdateWithContext(c.Request.Context(), clusterID, namespace, c.Param("name"), req, common.GetDataScope(c))
	if err != nil {
		failServiceError(c, err, "k8s.service.update_failed")
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
	if err := h.svc.DeleteWithResourceVersionAndContext(c.Request.Context(), clusterID, namespace, c.Param("name"), c.Query("resourceVersion"), c.Query("uid"), common.GetDataScope(c)); err != nil {
		failServiceError(c, err, "k8s.service.delete_failed")
		return
	}
	common.Success(c, nil)
}

func failServiceError(c *gin.Context, err error, fallback string) {
	code := common.CodeForError(err)
	if errors.Is(err, common.ErrConflict) {
		code = http.StatusConflict
	}
	// Existing owner keys predate common's typed errors. Classify only their
	// exact contracts, never an arbitrary provider failure's fallback text.
	if code == common.CodeError && err != nil {
		switch err.Error() {
		case "k8s.namespace.binding_required", "k8s.namespace.binding_invalid", "k8s.namespace.action_forbidden":
			code = common.CodeForbidden
		case "k8s.cluster.not_found":
			code = common.CodeNotFound
		}
	}
	if code == common.CodeError {
		common.FailWithError(c, code, err, fallback)
		return
	}
	common.FailWithCode(c, code, common.ResolveErrorMessageKey(err, fallback))
}
