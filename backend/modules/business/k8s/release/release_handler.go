package release

import (
	"errors"
	"io"
	"strconv"

	"pantheon-base/pkg/common"

	"github.com/gin-gonic/gin"
)

const msgParamInvalid = "common.param_invalid"

// ReleaseHandler exposes Kubernetes release HTTP endpoints.
//
//nolint:revive // retained as the public handler name for this package.
type ReleaseHandler struct {
	svc *ReleaseService
}

// NewReleaseHandler creates the Kubernetes release HTTP handler.
func NewReleaseHandler(svc *ReleaseService) *ReleaseHandler {
	return &ReleaseHandler{svc: svc}
}

// RegisterRoutes registers release endpoints.
func (h *ReleaseHandler) RegisterRoutes(r gin.IRoutes) {
	r.GET("/releases", h.List)
	r.POST("/releases", h.Create)
	r.POST("/releases/:id/rollback", h.Rollback)
	r.POST("/releases/:id/reconcile", h.Reconcile)
}

// List handles release listing.
func (h *ReleaseHandler) List(c *gin.Context) {
	var query ReleaseListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	resp, err := h.svc.List(query, common.GetDataScope(c))
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "k8s.release.list_failed")
		return
	}
	common.Success(c, resp)
}

// Create handles release creation.
func (h *ReleaseHandler) Create(c *gin.Context) {
	common.SetAuditMetadata(c, "k8s.release.audit.create", common.BusinessUpdate)

	var req CreateReleaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	createdBy := strconv.FormatUint(common.GetUserID(c), 10)
	resp, err := h.svc.Create(req, createdBy, common.GetDataScope(c))
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "k8s.release.create_failed")
		return
	}
	common.Success(c, resp)
}

// Rollback handles release rollback.
func (h *ReleaseHandler) Rollback(c *gin.Context) {
	common.SetAuditMetadata(c, "k8s.release.audit.rollback", common.BusinessUpdate)

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	var req RollbackReleaseRequest
	if err := c.ShouldBindJSON(&req); err != nil && !errors.Is(err, io.EOF) {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	createdBy := strconv.FormatUint(common.GetUserID(c), 10)
	resp, err := h.svc.Rollback(id, req, createdBy, common.GetDataScope(c))
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "k8s.release.rollback_failed")
		return
	}
	common.Success(c, resp)
}

// Reconcile handles release observation reconciliation.
func (h *ReleaseHandler) Reconcile(c *gin.Context) {
	common.SetAuditMetadata(c, "k8s.release.audit.reconcile", common.BusinessUpdate)

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	resp, err := h.svc.Reconcile(id, common.GetDataScope(c))
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "k8s.release.reconcile_failed")
		return
	}
	common.Success(c, resp)
}
