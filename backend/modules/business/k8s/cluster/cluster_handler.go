package cluster

import (
	"strconv"

	"pantheon-base/pkg/common"

	"github.com/gin-gonic/gin"
)

const (
	msgParamInvalid = "common.param_invalid"
	clusterIDRoute  = "/clusters/:id"
)

// ClusterHandler exposes Kubernetes cluster HTTP endpoints.
//
//nolint:revive // retained as the public handler name for this package.
type ClusterHandler struct {
	svc *ClusterService
}

// NewClusterHandler creates the Kubernetes cluster HTTP handler.
func NewClusterHandler(svc *ClusterService) *ClusterHandler {
	return &ClusterHandler{svc: svc}
}

// RegisterRoutes registers Kubernetes cluster endpoints.
func (h *ClusterHandler) RegisterRoutes(r gin.IRoutes) {
	r.GET("/clusters", h.List)
	r.POST("/clusters", h.Create)
	r.GET(clusterIDRoute, h.GetByID)
	r.PUT(clusterIDRoute, h.Update)
	r.DELETE(clusterIDRoute, h.Delete)
	r.POST("/clusters/:id/sync", h.Sync)
	r.GET("/clusters/:id/nodes", h.GetNodes)
}

// List returns Kubernetes clusters visible in the request scope.
func (h *ClusterHandler) List(c *gin.Context) {
	var query ClusterListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	resp, err := h.svc.List(query, common.GetDataScope(c))
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "k8s.cluster.list_failed")
		return
	}
	common.Success(c, resp)
}

// GetByID returns one Kubernetes cluster.
func (h *ClusterHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	resp, err := h.svc.GetByID(id, common.GetDataScope(c))
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "k8s.cluster.not_found")
		return
	}
	common.Success(c, resp)
}

// Create registers a Kubernetes cluster.
func (h *ClusterHandler) Create(c *gin.Context) {
	common.SetAuditMetadata(c, "k8s.cluster.audit.create", common.BusinessInsert)

	var req CreateClusterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	deptID := uint64(0)
	if scope := common.GetDataScope(c); scope != nil {
		deptID = scope.DeptID
	}
	createdBy := strconv.FormatUint(common.GetUserID(c), 10)
	resp, err := h.svc.Create(req, createdBy, deptID)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "k8s.cluster.create_failed")
		return
	}
	common.Success(c, resp)
}

// Update changes a Kubernetes cluster registration.
func (h *ClusterHandler) Update(c *gin.Context) {
	common.SetAuditMetadata(c, "k8s.cluster.audit.update", common.BusinessUpdate)

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	var req UpdateClusterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	updatedBy := strconv.FormatUint(common.GetUserID(c), 10)
	resp, err := h.svc.Update(id, req, updatedBy, common.GetDataScope(c))
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "k8s.cluster.update_failed")
		return
	}
	common.Success(c, resp)
}

// Delete removes a Kubernetes cluster registration.
func (h *ClusterHandler) Delete(c *gin.Context) {
	common.SetAuditMetadata(c, "k8s.cluster.audit.delete", common.BusinessDelete)

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	if err := h.svc.Delete(id, common.GetDataScope(c)); err != nil {
		common.FailWithError(c, common.CodeError, err, "k8s.cluster.delete_failed")
		return
	}
	common.Success(c, nil)
}

// Sync refreshes Kubernetes cluster metadata.
func (h *ClusterHandler) Sync(c *gin.Context) {
	common.SetAuditMetadata(c, "k8s.cluster.audit.sync", common.BusinessUpdate)

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	resp, err := h.svc.Sync(id, common.GetDataScope(c))
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "k8s.cluster.sync_failed")
		return
	}
	common.Success(c, resp)
}

// GetNodes returns nodes discovered from a Kubernetes cluster.
func (h *ClusterHandler) GetNodes(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	resp, err := h.svc.GetNodes(id, common.GetDataScope(c))
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "k8s.cluster.nodes_failed")
		return
	}
	common.Success(c, resp)
}
