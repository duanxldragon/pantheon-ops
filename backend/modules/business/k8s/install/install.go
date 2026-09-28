// Package install records Kubernetes cluster installation wizard intents.
// Rows describe an ongoing or finished k3s/k8s install on a target host.
package install

import (
	"errors"
	"strings"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/modules/business/cmdb/host"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const msgParamInvalid = "common.param_invalid"

// Install is a Kubernetes installation wizard record.
type Install struct {
	ID          uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	ClusterCode string    `gorm:"column:cluster_code;size:128;not null;index" json:"clusterCode"`
	HostIP      string    `gorm:"column:host_ip;size:64;not null;index" json:"hostIP"`
	InstallType string    `gorm:"column:install_type;size:16;not null;index" json:"installType"`
	Version     string    `gorm:"size:64" json:"version"`
	Status      string    `gorm:"size:32;default:pending;index" json:"status"`
	Message     string    `gorm:"type:text" json:"message"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

// TableName returns the install record table name.
func (Install) TableName() string { return "biz_k8s_install" }

// ListQuery filters install records.
type ListQuery struct {
	ClusterCode string `form:"clusterCode"`
	HostIP      string `form:"hostIP"`
	InstallType string `form:"installType"`
	Status      string `form:"status"`
}

// ListResponse contains install records.
type ListResponse struct {
	Items []Install `json:"items"`
	Total int64     `json:"total"`
}

// CreateInstallRequest contains install wizard fields.
type CreateInstallRequest struct {
	ClusterCode string `json:"clusterCode" binding:"required"`
	HostIP      string `json:"hostIP" binding:"required"`
	InstallType string `json:"installType" binding:"required"`
	Version     string `json:"version"`
}

// InstallService manages Kubernetes install wizard records.
//
//nolint:revive // retained as the public service name for this package.
type InstallService struct {
	db          *gorm.DB
	hostService *host.HostService
}

// NewInstallService creates an install record service.
func NewInstallService(db *gorm.DB, hostService *host.HostService) *InstallService {
	return &InstallService{db: db, hostService: hostService}
}

// InitInstallModule initialises the install module and registers its routes.
func InitInstallModule(r *gin.RouterGroup, db *gorm.DB, hostService *host.HostService) *InstallService {
	svc := NewInstallService(db, hostService)
	r.GET("/install", svc.List)
	r.POST("/install", svc.Create)
	return svc
}

// Migrate creates the install record table.
func (s *InstallService) Migrate() error {
	if s.db == nil {
		return errors.New("database.not_initialized")
	}
	return s.db.AutoMigrate(&Install{})
}

// List returns install records matching the filters.
func (s *InstallService) List(c *gin.Context) {
	var query ListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	if s.db == nil {
		common.FailWithError(c, common.CodeError, errors.New("database.not_initialized"), "k8s.install.list_failed")
		return
	}
	db := s.db.Model(&Install{})
	if query.ClusterCode != "" {
		db = db.Where("cluster_code = ?", query.ClusterCode)
	}
	if query.HostIP != "" {
		db = db.Where("host_ip = ?", query.HostIP)
	}
	if query.InstallType != "" {
		db = db.Where("install_type = ?", query.InstallType)
	}
	if query.Status != "" {
		db = db.Where("status = ?", query.Status)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		common.FailWithError(c, common.CodeError, err, "k8s.install.list_failed")
		return
	}
	var items []Install
	if err := db.Order("id DESC").Find(&items).Error; err != nil {
		common.FailWithError(c, common.CodeError, err, "k8s.install.list_failed")
		return
	}
	common.Success(c, ListResponse{Items: items, Total: total})
}

// Create records an install wizard intent after validating the target host.
func (s *InstallService) Create(c *gin.Context) {
	common.SetAuditMetadata(c, "k8s.install.audit.create", common.BusinessInsert)

	var req CreateInstallRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	installType := strings.ToLower(strings.TrimSpace(req.InstallType))
	if installType != "k3s" && installType != "k8s" {
		common.Fail(c, common.CodeParamInvalid, "k8s.install.invalid_type")
		return
	}
	if s.db == nil {
		common.FailWithError(c, common.CodeError, errors.New("database.not_initialized"), "k8s.install.create_failed")
		return
	}
	if s.hostService != nil {
		hostList, err := s.hostService.List(host.HostListQuery{Keyword: strings.TrimSpace(req.HostIP), Page: 1, PageSize: 1}, common.GetDataScope(c))
		if err != nil || hostList == nil || hostList.Total == 0 {
			common.Fail(c, common.CodeParamInvalid, "k8s.install.host_not_found")
			return
		}
	}
	record := Install{
		ClusterCode: strings.TrimSpace(req.ClusterCode),
		HostIP:      strings.TrimSpace(req.HostIP),
		InstallType: installType,
		Version:     req.Version,
		Status:      "pending",
	}
	if err := s.db.Create(&record).Error; err != nil {
		common.FailWithError(c, common.CodeError, err, "k8s.install.create_failed")
		return
	}
	common.Success(c, record)
}
