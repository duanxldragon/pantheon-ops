// Package ingress provides DB-snapshot management of Kubernetes Ingress
// resources. Rows mirror the cluster state at the last sync.
package ingress

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/modules/business/k8s/cluster"
	"github.com/duanxldragon/pantheon-base/backend/modules/business/k8s/namespace"
	"github.com/duanxldragon/pantheon-base/backend/modules/business/k8s/rbac"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/database"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	msgParamInvalid     = "common.param_invalid"
	opTimeout           = 30 * time.Second
	ingressStatusActive = "active"
)

// Ingress is the DB snapshot of a Kubernetes Ingress.
type Ingress struct {
	ID              uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	Name            string         `gorm:"size:255;not null;index" json:"name"`
	Namespace       string         `gorm:"size:255;not null;index" json:"namespace"`
	ClusterID       uint64         `gorm:"column:cluster_id;index" json:"clusterId"`
	BusinessScopeID uint64         `gorm:"column:business_scope_id;index" json:"-"`
	DeptID          uint64         `gorm:"column:dept_id;index" json:"-"`
	IngressClass    string         `gorm:"column:ingress_class;size:128" json:"ingressClass"`
	Rules           datatypes.JSON `gorm:"type:json" json:"rules"`
	TLS             datatypes.JSON `gorm:"type:json" json:"tls"`
	Annotations     datatypes.JSON `gorm:"type:json" json:"annotations"`
	Status          string         `gorm:"size:50;default:active" json:"status"`
	LoadBalancerIP  string         `gorm:"column:load_balancer_ip;size:255" json:"loadBalancerIp"`
	Hosts           datatypes.JSON `gorm:"type:json" json:"hosts"`
	CreatedBy       string         `gorm:"size:64" json:"createdBy"`
	CreatedAt       time.Time      `json:"createdAt"`
	UpdatedAt       time.Time      `json:"updatedAt"`
}

// TableName returns the ingress snapshot table name.
func (Ingress) TableName() string {
	return "biz_k8s_ingress"
}

// IngressRule mirrors the frontend K8sIngressRule contract.
type IngressRule struct {
	Host  string            `json:"host"`
	Paths []IngressRulePath `json:"paths"`
}

// IngressRulePath mirrors the frontend K8sIngressRulePath contract.
type IngressRulePath struct {
	Path        string `json:"path"`
	PathType    string `json:"pathType"`
	ServiceName string `json:"serviceName"`
	ServicePort int32  `json:"servicePort"`
}

// IngressTLS mirrors the frontend K8sIngressTLS contract.
type IngressTLS struct {
	Hosts      []string `json:"hosts"`
	SecretName string   `json:"secretName"`
}

// IngressListQuery filters ingress snapshots.
type IngressListQuery struct {
	ClusterID uint64 `form:"clusterId"`
	Namespace string `form:"namespace"`
	Name      string `form:"name"`
	Host      string `form:"host"`
}

// IngressListResponse contains ingress snapshots.
type IngressListResponse struct {
	Items []Ingress `json:"items"`
	Total int64     `json:"total"`
}

// IngressService manages Kubernetes Ingress snapshots.
//
//nolint:revive // retained as the public service name for this package.
type IngressService struct {
	db           *gorm.DB
	clusterSvc   *cluster.ClusterService
	namespaceSvc *namespace.NamespaceService
}

// NewIngressService creates an Ingress service.
func NewIngressService(db *gorm.DB, clusterSvc *cluster.ClusterService, namespaceSvc *namespace.NamespaceService) *IngressService {
	return &IngressService{db: db, clusterSvc: clusterSvc, namespaceSvc: namespaceSvc}
}

// Migrate creates the ingress snapshot table.
func (s *IngressService) Migrate() error {
	if s.db == nil {
		return errors.New("database.not_initialized")
	}
	return s.db.AutoMigrate(&Ingress{})
}

// List returns ingress snapshots matching the filters.
func (s *IngressService) List(query IngressListQuery, dataScope *common.DataScopeReq) (*IngressListResponse, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}
	db := s.db.Model(&Ingress{}).Scopes(database.WithDataScope(dataScope))
	if query.ClusterID > 0 {
		db = db.Where("cluster_id = ?", query.ClusterID)
	}
	if query.Namespace != "" {
		db = db.Where("namespace = ?", query.Namespace)
	}
	if query.Name != "" {
		db = db.Where("name = ?", query.Name)
	}
	if query.Host != "" {
		db = db.Where("hosts LIKE ?", "%"+strings.TrimSpace(query.Host)+"%")
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}
	var items []Ingress
	if err := db.Order("id DESC").Find(&items).Error; err != nil {
		return nil, err
	}
	return &IngressListResponse{Items: items, Total: total}, nil
}

// Sync refreshes one ingress snapshot from the live cluster.
func (s *IngressService) Sync(id uint64, dataScope *common.DataScopeReq) (*Ingress, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}
	var record Ingress
	if err := s.db.Scopes(database.WithDataScope(dataScope)).First(&record, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("k8s.ingress.not_found")
		}
		return nil, err
	}
	clientset, err := s.clusterSvc.GetClientset(record.ClusterID, dataScope)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), opTimeout)
	defer cancel()

	item, err := clientset.NetworkingV1().Ingresses(record.Namespace).Get(ctx, record.Name, metav1.GetOptions{})
	if err != nil {
		return nil, errors.New("k8s.ingress.sync_failed")
	}
	updated := fromK8s(item, record.ClusterID, record.BusinessScopeID, record.DeptID)
	updated.ID = record.ID
	updated.CreatedAt = record.CreatedAt
	updated.CreatedBy = record.CreatedBy
	if err := s.db.Save(&updated).Error; err != nil {
		return nil, err
	}
	return &updated, nil
}

// Delete removes an ingress snapshot after verifying the namespace binding.
func (s *IngressService) Delete(id uint64, dataScope *common.DataScopeReq) error {
	if s.db == nil {
		return errors.New("database.not_initialized")
	}
	var record Ingress
	if err := s.db.Scopes(database.WithDataScope(dataScope)).First(&record, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("k8s.ingress.not_found")
		}
		return err
	}
	if s.namespaceSvc != nil {
		if err := s.namespaceSvc.RequireWrite(record.ClusterID, record.Namespace, "delete"); err != nil {
			return err
		}
	}
	return s.db.Delete(&Ingress{}, id).Error
}

// IngressHandler exposes ingress snapshot endpoints.
//
//nolint:revive // retained as the public handler name for this package.
type IngressHandler struct {
	svc     *IngressService
	rbacSvc *rbac.K8sRBACService
}

// NewIngressHandler creates the Ingress HTTP handler.
func NewIngressHandler(svc *IngressService, rbacSvc *rbac.K8sRBACService) *IngressHandler {
	return &IngressHandler{svc: svc, rbacSvc: rbacSvc}
}

// RegisterRoutes registers ingress endpoints.
func (h *IngressHandler) RegisterRoutes(r gin.IRoutes) {
	r.GET("/ingresses", h.List)
	r.POST("/ingresses/:id/sync", h.Sync)
	r.DELETE("/ingresses/:id", h.Delete)
}

// List handles ingress listing.
func (h *IngressHandler) List(c *gin.Context) {
	var query IngressListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	resp, err := h.svc.List(query, common.GetDataScope(c))
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "k8s.ingress.list_failed")
		return
	}
	common.Success(c, resp)
}

// Sync handles single ingress refresh.
func (h *IngressHandler) Sync(c *gin.Context) {
	common.SetAuditMetadata(c, "k8s.ingress.audit.sync", common.BusinessUpdate)

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	if h.rbacSvc != nil && !h.rbacSvc.CheckHandlerPermission(c, 0, "", rbac.ResourceTypeIngress, rbac.ActionUpdate) {
		common.Fail(c, common.CodeForbidden, "k8s.rbac.permission_denied")
		return
	}
	record, err := h.svc.Sync(id, common.GetDataScope(c))
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "k8s.ingress.sync_failed")
		return
	}
	common.Success(c, gin.H{"synced": true, "item": record})
}

// Delete handles ingress snapshot deletion.
func (h *IngressHandler) Delete(c *gin.Context) {
	common.SetAuditMetadata(c, "k8s.ingress.audit.delete", common.BusinessDelete)

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	if h.rbacSvc != nil && !h.rbacSvc.CheckHandlerPermission(c, 0, "", rbac.ResourceTypeIngress, rbac.ActionDelete) {
		common.Fail(c, common.CodeForbidden, "k8s.rbac.permission_denied")
		return
	}
	if err := h.svc.Delete(id, common.GetDataScope(c)); err != nil {
		common.FailWithError(c, common.CodeError, err, "k8s.ingress.delete_failed")
		return
	}
	common.Success(c, nil)
}

func fromK8s(item *networkingv1.Ingress, clusterID, businessScopeID, deptID uint64) Ingress {
	ingressClass := ""
	if item.Spec.IngressClassName != nil {
		ingressClass = *item.Spec.IngressClassName
	}
	rules := make([]IngressRule, 0, len(item.Spec.Rules))
	hosts := make([]string, 0, len(item.Spec.Rules))
	for _, rule := range item.Spec.Rules {
		paths := make([]IngressRulePath, 0, len(rule.HTTP.Paths))
		for _, p := range rule.HTTP.Paths {
			pathType := ""
			if p.PathType != nil {
				pathType = string(*p.PathType)
			}
			port := int32(0)
			if p.Backend.Service != nil {
				port = p.Backend.Service.Port.Number
			}
			serviceName := ""
			if p.Backend.Service != nil {
				serviceName = p.Backend.Service.Name
			}
			paths = append(paths, IngressRulePath{
				Path:        p.Path,
				PathType:    pathType,
				ServiceName: serviceName,
				ServicePort: port,
			})
		}
		if rule.Host != "" {
			hosts = append(hosts, rule.Host)
		}
		rules = append(rules, IngressRule{Host: rule.Host, Paths: paths})
	}
	tls := make([]IngressTLS, 0, len(item.Spec.TLS))
	for _, t := range item.Spec.TLS {
		tls = append(tls, IngressTLS{Hosts: t.Hosts, SecretName: t.SecretName})
	}
	annotations := item.Annotations
	if annotations == nil {
		annotations = map[string]string{}
	}
	lbIP := ""
	if len(item.Status.LoadBalancer.Ingress) > 0 {
		lbIP = item.Status.LoadBalancer.Ingress[0].IP
	}
	return Ingress{
		Name:            item.Name,
		Namespace:       item.Namespace,
		ClusterID:       clusterID,
		BusinessScopeID: businessScopeID,
		DeptID:          deptID,
		IngressClass:    ingressClass,
		Rules:           jsonBytes(rules),
		TLS:             jsonBytes(tls),
		Annotations:     jsonBytes(annotations),
		Status:          ingressStatusActive,
		LoadBalancerIP:  lbIP,
		Hosts:           jsonBytes(hosts),
	}
}

func jsonBytes(v interface{}) datatypes.JSON {
	b, _ := json.Marshal(v)
	return datatypes.JSON(b)
}
