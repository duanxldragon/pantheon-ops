// Package observability provides the basic observability module: monitoring
// target registration and a lightweight metric view. The V1 scope is a
// scaffolding module that records monitoring targets per business scope and
// exposes a health endpoint; Prometheus scraping configuration and PromQL
// proxying are planned for v1.1 (see BUSINESS_OBSERVABILITY_MODULE_DESIGN.md).
package observability

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"pantheon-base/modules/business/bizscope"
	"pantheon-base/pkg/common"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// MonitoringTargetType enumerates the target kinds the module can track.
const (
	TargetTypeHost  = "host"
	TargetTypeK8s   = "k8s_cluster"
	TargetTypeOther = "other"
)

// MonitoringTarget is a DB-recorded monitoring target bound to a business
// scope. It is the seed for file_sd generation in v1.1.
type MonitoringTarget struct {
	ID               uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	BusinessScopeID  uint64    `gorm:"column:business_scope_id;index" json:"businessScopeId"`
	TargetType       string    `gorm:"size:32;not null" json:"targetType"`
	TargetName       string    `gorm:"size:128;not null" json:"targetName"`
	Endpoint         string    `gorm:"size:255" json:"endpoint"`
	MetricsPath      string    `gorm:"size:255;default:/metrics" json:"metricsPath"`
	Enabled          bool      `gorm:"not null;default:true" json:"enabled"`
	Status           string    `gorm:"size:32;default:pending" json:"status"`
	LastScrapedAt    *time.Time `gorm:"column:last_scraped_at" json:"lastScrapedAt,omitempty"`
	Remark           string    `gorm:"size:512" json:"remark"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
	CreatedBy        string    `gorm:"size:64" json:"createdBy"`
}

// TableName returns the monitoring target table name.
func (MonitoringTarget) TableName() string { return "biz_observability_target" }

// TargetListQuery contains monitoring target list filters.
type TargetListQuery struct {
	BusinessScopeID uint64 `form:"businessScopeId"`
	TargetType      string `form:"targetType"`
	Page            int    `form:"page"`
	PageSize        int    `form:"pageSize"`
}

// TargetListResponse is the paginated monitoring target list.
type TargetListResponse struct {
	Items    []MonitoringTarget `json:"items"`
	Total    int64              `json:"total"`
	Page     int                `json:"page"`
	PageSize int                `json:"pageSize"`
}

// CreateTargetRequest contains monitoring target creation fields.
type CreateTargetRequest struct {
	BusinessScopeID uint64 `json:"businessScopeId" binding:"required"`
	TargetType      string `json:"targetType" binding:"required"`
	TargetName      string `json:"targetName" binding:"required"`
	Endpoint        string `json:"endpoint"`
	MetricsPath     string `json:"metricsPath"`
	Remark          string `json:"remark"`
}

// ObservabilityModule is the module facade exposed to the overlay registry.
type ObservabilityModule struct {
	db            *gorm.DB
	bizScopeSvc   *bizscope.Service
}

// InitObservabilityModule creates the observability module.
func InitObservabilityModule(r *gin.RouterGroup, db *gorm.DB, bizScopeSvc *bizscope.Service) *ObservabilityModule {
	return &ObservabilityModule{db: db, bizScopeSvc: bizScopeSvc}
}

// Migrate creates the monitoring target table.
func (m *ObservabilityModule) Migrate(db *gorm.DB) error {
	return db.AutoMigrate(&MonitoringTarget{})
}

// RegisterRoutes registers observability endpoints on the given router group.
func (m *ObservabilityModule) RegisterRoutes(r *gin.RouterGroup) {
	group := r.Group("/business/observability")
	group.GET("/targets", m.ListTargets)
	group.POST("/targets", m.CreateTarget)
	group.DELETE("/targets/:id", m.DeleteTarget)
	group.GET("/health", func(c *gin.Context) {
		common.Success(c, gin.H{"status": "ok"})
	})
}

// ListTargets returns monitoring targets visible in the caller's data scope.
func (m *ObservabilityModule) ListTargets(c *gin.Context) {
	var query TargetListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.Fail(c, common.CodeParamInvalid, "common.param_invalid")
		return
	}
	page := query.Page
	if page <= 0 {
		page = 1
	}
	pageSize := query.PageSize
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	db := m.db.Model(&MonitoringTarget{})
	if query.BusinessScopeID > 0 {
		db = db.Where("business_scope_id = ?", query.BusinessScopeID)
	}
	if strings.TrimSpace(query.TargetType) != "" {
		db = db.Where("target_type = ?", strings.TrimSpace(query.TargetType))
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		common.FailWithError(c, common.CodeError, err, "observability.list_failed")
		return
	}
	var items []MonitoringTarget
	if err := db.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&items).Error; err != nil {
		common.FailWithError(c, common.CodeError, err, "observability.list_failed")
		return
	}
	common.Success(c, TargetListResponse{Items: items, Total: total, Page: page, PageSize: pageSize})
}

// CreateTarget creates a monitoring target bound to a business scope.
func (m *ObservabilityModule) CreateTarget(c *gin.Context) {
	common.SetAuditMetadata(c, "observability.target.create", common.BusinessInsert)
	var req CreateTargetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, "common.param_invalid")
		return
	}
	req.TargetType = strings.TrimSpace(req.TargetType)
	if req.TargetType != TargetTypeHost && req.TargetType != TargetTypeK8s {
		req.TargetType = TargetTypeOther
	}
	req.MetricsPath = strings.TrimSpace(req.MetricsPath)
	if req.MetricsPath == "" {
		req.MetricsPath = "/metrics"
	}
	createdBy := strconv.FormatUint(common.GetUserID(c), 10)
	target := MonitoringTarget{
		BusinessScopeID: req.BusinessScopeID,
		TargetType:      req.TargetType,
		TargetName:      strings.TrimSpace(req.TargetName),
		Endpoint:        strings.TrimSpace(req.Endpoint),
		MetricsPath:     req.MetricsPath,
		Enabled:         true,
		Status:          "pending",
		Remark:          req.Remark,
		CreatedBy:       createdBy,
	}
	if err := m.db.Create(&target).Error; err != nil {
		common.FailWithError(c, common.CodeError, err, "observability.create_failed")
		return
	}
	common.Success(c, target)
}

// DeleteTarget deletes a monitoring target.
func (m *ObservabilityModule) DeleteTarget(c *gin.Context) {
	common.SetAuditMetadata(c, "observability.target.delete", common.BusinessDelete)
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		common.Fail(c, common.CodeParamInvalid, "common.param_invalid")
		return
	}
	result := m.db.Where("id = ?", id).Delete(&MonitoringTarget{})
	if result.Error != nil {
		common.FailWithError(c, common.CodeError, result.Error, "observability.delete_failed")
		return
	}
	if result.RowsAffected == 0 {
		common.FailWithError(c, common.CodeError, errors.New("observability.target_not_found"), "observability.target_not_found")
		return
	}
	common.Success(c, gin.H{"deleted": true})
}
