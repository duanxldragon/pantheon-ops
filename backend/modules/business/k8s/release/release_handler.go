package release

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/duanxldragon/pantheon-base/backend/pkg/common"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
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
	r.POST("/releases/preview", h.PreviewCreate)
	r.POST("/releases/:id/rollback/preview", h.PreviewRollback)
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
	if err := bindReviewedRelease(c, &req); err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	createdBy := strconv.FormatUint(common.GetUserID(c), 10)
	resp, err := h.svc.CreateReviewed(c.Request.Context(), req, createdBy, common.GetDataScope(c))
	if err != nil {
		failReleaseReview(c, err, "k8s.release.create_failed")
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
	if err := bindReviewedRelease(c, &req); err != nil && !errors.Is(err, io.EOF) {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	createdBy := strconv.FormatUint(common.GetUserID(c), 10)
	resp, err := h.svc.RollbackReviewed(c.Request.Context(), id, req, createdBy, common.GetDataScope(c))
	if err != nil {
		failReleaseReview(c, err, "k8s.release.rollback_failed")
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

func failReleaseReview(c *gin.Context, err error, fallback string) {
	if errors.Is(err, common.ErrConflict) {
		common.FailWithCode(c, 409, common.ErrMessage(err))
		return
	}
	common.FailWithError(c, common.CodeForError(err), err, fallback)
}

// PreviewCreate audits the read-only image review.
func (h *ReleaseHandler) PreviewCreate(c *gin.Context) {
	common.SetAuditMetadata(c, "k8s.release.audit.preview", common.BusinessOther)
	var req CreateReleaseRequest
	if err := bindReviewedRelease(c, &req); err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	result, err := h.svc.PreviewCreate(c.Request.Context(), req, strconv.FormatUint(common.GetUserID(c), 10), common.GetDataScope(c))
	if err != nil {
		failReleaseReview(c, err, "k8s.release.preview_failed")
		return
	}
	common.Success(c, result)
}

// PreviewRollback reviews restoring the recorded image without executing it.
func (h *ReleaseHandler) PreviewRollback(c *gin.Context) {
	common.SetAuditMetadata(c, "k8s.release.audit.preview", common.BusinessOther)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	var req RollbackReleaseRequest
	if err := bindReviewedRelease(c, &req); err != nil && !errors.Is(err, io.EOF) {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	result, err := h.svc.PreviewRollback(c.Request.Context(), id, req, strconv.FormatUint(common.GetUserID(c), 10), common.GetDataScope(c))
	if err != nil {
		failReleaseReview(c, err, "k8s.release.preview_failed")
		return
	}
	common.Success(c, result)
}

// Unsupported manifest/config/resource writes must fail explicitly, rather than
// be silently discarded while executing an image-only intent.
func bindReviewedRelease(c *gin.Context, target interface{}) error {
	decoder := json.NewDecoder(http.MaxBytesReader(c.Writer, c.Request.Body, 64*1024))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	var trailing interface{}
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return errors.New(msgParamInvalid)
	}
	return binding.Validator.ValidateStruct(target)
}
