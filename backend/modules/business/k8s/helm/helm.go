// Package helm provides DB-snapshot records of Helm releases. Kubernetes
// remains the runtime source of truth; these rows track releases registered
// through the platform.
package helm

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/database"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const msgParamInvalid = "common.param_invalid"

// HelmRelease is the DB record of a Helm release.
type HelmRelease struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	Name      string    `gorm:"size:255;not null;index" json:"name"`
	ClusterID uint64    `gorm:"column:cluster_id;index" json:"clusterId"`
	Namespace string    `gorm:"size:255;not null;index" json:"namespace"`
	Chart     string    `gorm:"size:255" json:"chart"`
	Version   string    `gorm:"size:128" json:"version"`
	Status    string    `gorm:"size:64;default:deployed;index" json:"status"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// TableName returns the Helm release table name.
func (HelmRelease) TableName() string { return "biz_k8s_helm_release" }

// CreateHelmReleaseRequest contains Helm release creation fields.
type CreateHelmReleaseRequest struct {
	Name      string `json:"name" binding:"required"`
	ClusterID uint64 `json:"clusterId" binding:"required"`
	Namespace string `json:"namespace" binding:"required"`
	Chart     string `json:"chart"`
	Version   string `json:"version"`
	Status    string `json:"status"`
}

// ListQuery filters Helm release records.
type ListQuery struct {
	ClusterID uint64 `form:"clusterId"`
	Namespace string `form:"namespace"`
	Name      string `form:"name"`
}

// ListResponse contains Helm release records.
type ListResponse struct {
	Items []HelmRelease `json:"items"`
	Total int64         `json:"total"`
}

// HelmService manages Helm release snapshot records.
//
//nolint:revive // retained as the public service name for this package.
type HelmService struct {
	db *gorm.DB
}

// NewHelmService creates a Helm release service.
func NewHelmService(db *gorm.DB) *HelmService {
	return &HelmService{db: db}
}

// Migrate creates the Helm release table.
func (s *HelmService) Migrate() error {
	if s.db == nil {
		return errors.New("database.not_initialized")
	}
	return s.db.AutoMigrate(&HelmRelease{})
}

// List returns Helm release records.
func (s *HelmService) List(query ListQuery, dataScope *common.DataScopeReq) (*ListResponse, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}
	db := s.db.Model(&HelmRelease{}).Scopes(database.WithDataScope(dataScope))
	if query.ClusterID > 0 {
		db = db.Where("cluster_id = ?", query.ClusterID)
	}
	if query.Namespace != "" {
		db = db.Where("namespace = ?", query.Namespace)
	}
	if query.Name != "" {
		db = db.Where("name = ?", query.Name)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}
	var items []HelmRelease
	if err := db.Order("id DESC").Find(&items).Error; err != nil {
		return nil, err
	}
	return &ListResponse{Items: items, Total: total}, nil
}

// Create records a Helm release.
func (s *HelmService) Create(req CreateHelmReleaseRequest, createdBy string) (*HelmRelease, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}
	if strings.TrimSpace(req.Name) == "" || req.ClusterID == 0 || strings.TrimSpace(req.Namespace) == "" {
		return nil, errors.New("common.param_invalid")
	}
	record := HelmRelease{
		Name:      strings.TrimSpace(req.Name),
		ClusterID: req.ClusterID,
		Namespace: strings.TrimSpace(req.Namespace),
		Chart:     req.Chart,
		Version:   req.Version,
		Status:    req.Status,
	}
	if record.Status == "" {
		record.Status = "deployed"
	}
	_ = createdBy // retained for audit symmetry with other k8s modules
	if err := s.db.Create(&record).Error; err != nil {
		return nil, err
	}
	return &record, nil
}

// Delete removes a Helm release record.
func (s *HelmService) Delete(id uint64, dataScope *common.DataScopeReq) error {
	if s.db == nil {
		return errors.New("database.not_initialized")
	}
	return s.db.Scopes(database.WithDataScope(dataScope)).Delete(&HelmRelease{}, id).Error
}

// HelmHandler exposes Helm release endpoints.
//
//nolint:revive // retained as the public handler name for this package.
type HelmHandler struct {
	svc *HelmService
}

// NewHelmHandler creates the Helm HTTP handler.
func NewHelmHandler(svc *HelmService) *HelmHandler {
	return &HelmHandler{svc: svc}
}

// RegisterRoutes registers Helm release endpoints.
func (h *HelmHandler) RegisterRoutes(r gin.IRoutes) {
	r.GET("/helm/releases", h.List)
	r.POST("/helm/releases", h.Create)
	r.DELETE("/helm/releases/:id", h.Delete)
}

// List handles Helm release listing.
func (h *HelmHandler) List(c *gin.Context) {
	var query ListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	resp, err := h.svc.List(query, common.GetDataScope(c))
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "k8s.helm.list_failed")
		return
	}
	common.Success(c, resp)
}

// Create handles Helm release creation.
func (h *HelmHandler) Create(c *gin.Context) {
	common.SetAuditMetadata(c, "k8s.helm.audit.create", common.BusinessInsert)

	var req CreateHelmReleaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	createdBy := strconv.FormatUint(common.GetUserID(c), 10)
	record, err := h.svc.Create(req, createdBy)
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "k8s.helm.create_failed")
		return
	}
	common.Success(c, record)
}

// Delete handles Helm release deletion.
func (h *HelmHandler) Delete(c *gin.Context) {
	common.SetAuditMetadata(c, "k8s.helm.audit.delete", common.BusinessDelete)

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	if err := h.svc.Delete(id, common.GetDataScope(c)); err != nil {
		common.FailWithError(c, common.CodeError, err, "k8s.helm.delete_failed")
		return
	}
	common.Success(c, nil)
}
