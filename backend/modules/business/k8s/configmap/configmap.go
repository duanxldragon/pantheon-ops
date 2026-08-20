// Package configmap provides live ConfigMap management for a k8s cluster.
package configmap

import (
	"context"
	"errors"
	"strconv"
	"time"

	"pantheon-base/modules/business/k8s/cluster"
	"pantheon-base/modules/business/k8s/namespace"
	"pantheon-base/pkg/common"

	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	msgParamInvalid = "common.param_invalid"
	opTimeout       = 30 * time.Second
)

func normalizeListLimit(limit int64) int64 {
	if limit <= 0 {
		return 100
	}
	if limit > 500 {
		return 500
	}
	return limit
}

// ConfigMapItem summarizes a Kubernetes ConfigMap.
//
//nolint:revive // retained as the public ConfigMap DTO name.
type ConfigMapItem struct {
	Name            string `json:"name"`
	Namespace       string `json:"namespace"`
	KeyCount        int    `json:"keyCount"`
	ResourceVersion string `json:"resourceVersion"`
}

// ConfigMapListResponse contains ConfigMap summaries.
//
//nolint:revive // retained as the public ConfigMap DTO name.
type ConfigMapListResponse struct {
	Items         []ConfigMapItem `json:"items"`
	Total         int             `json:"total"`
	ContinueToken string          `json:"continueToken"`
}

// ConfigMapDetail contains a Kubernetes ConfigMap payload.
//
//nolint:revive // retained as the public ConfigMap DTO name.
type ConfigMapDetail struct {
	Name            string            `json:"name"`
	Namespace       string            `json:"namespace"`
	Data            map[string]string `json:"data"`
	ResourceVersion string            `json:"resourceVersion"`
}

// ConfigMapListQuery contains pagination and namespace filters.
//
//nolint:revive // Public DTO name is retained for compatibility.
type ConfigMapListQuery struct {
	Namespace     string `form:"namespace"`
	Limit         int64  `form:"limit"`
	ContinueToken string `form:"continue"`
}

// CreateConfigMapRequest contains ConfigMap creation fields.
type CreateConfigMapRequest struct {
	Name string            `json:"name" binding:"required"`
	Data map[string]string `json:"data"`
}

// ConfigMapService manages Kubernetes ConfigMaps through client-go.
//
//nolint:revive // retained as the public service name for this package.
type ConfigMapService struct {
	clusterSvc   *cluster.ClusterService
	namespaceSvc *namespace.NamespaceService
}

// NewConfigMapService creates a ConfigMap service.
func NewConfigMapService(clusterSvc *cluster.ClusterService, namespaceSvc *namespace.NamespaceService) *ConfigMapService {
	return &ConfigMapService{clusterSvc: clusterSvc, namespaceSvc: namespaceSvc}
}

// List returns ConfigMaps in a namespace.
func (s *ConfigMapService) List(clusterID uint64, query ConfigMapListQuery, dataScope *common.DataScopeReq) (*ConfigMapListResponse, error) {
	clientset, err := s.clusterSvc.GetClientset(clusterID, dataScope)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), opTimeout)
	defer cancel()

	list, err := clientset.CoreV1().ConfigMaps(query.Namespace).List(ctx, metav1.ListOptions{Limit: normalizeListLimit(query.Limit), Continue: query.ContinueToken})
	if err != nil {
		return nil, errors.New("k8s.configmap.list_failed")
	}
	items := make([]ConfigMapItem, 0, len(list.Items))
	for i := range list.Items {
		cm := &list.Items[i]
		items = append(items, ConfigMapItem{Name: cm.Name, Namespace: cm.Namespace, KeyCount: len(cm.Data), ResourceVersion: cm.ResourceVersion})
	}
	return &ConfigMapListResponse{Items: items, Total: len(items), ContinueToken: list.Continue}, nil
}

// Get returns one ConfigMap.
func (s *ConfigMapService) Get(clusterID uint64, namespace, name string, dataScope *common.DataScopeReq) (*ConfigMapDetail, error) {
	clientset, err := s.clusterSvc.GetClientset(clusterID, dataScope)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), opTimeout)
	defer cancel()

	cm, err := clientset.CoreV1().ConfigMaps(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, errors.New("k8s.configmap.not_found")
	}
	return &ConfigMapDetail{Name: cm.Name, Namespace: cm.Namespace, Data: cm.Data, ResourceVersion: cm.ResourceVersion}, nil
}

// Create creates a ConfigMap.
func (s *ConfigMapService) Create(clusterID uint64, namespace string, req CreateConfigMapRequest, dataScope *common.DataScopeReq) (*ConfigMapDetail, error) {
	if s.namespaceSvc == nil {
		return nil, errors.New("k8s.namespace.binding_required")
	}
	if err := s.namespaceSvc.RequireWrite(clusterID, namespace, "configmap:create"); err != nil {
		return nil, err
	}
	clientset, err := s.clusterSvc.GetClientset(clusterID, dataScope)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), opTimeout)
	defer cancel()

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: req.Name, Namespace: namespace},
		Data:       req.Data,
	}
	created, err := clientset.CoreV1().ConfigMaps(namespace).Create(ctx, cm, metav1.CreateOptions{})
	if err != nil {
		return nil, errors.New("k8s.configmap.create_failed")
	}
	return &ConfigMapDetail{Name: created.Name, Namespace: created.Namespace, Data: created.Data, ResourceVersion: created.ResourceVersion}, nil
}

// Delete removes a ConfigMap.
func (s *ConfigMapService) Delete(clusterID uint64, namespace, name string, dataScope *common.DataScopeReq) error {
	return s.DeleteWithResourceVersion(clusterID, namespace, name, "", dataScope)
}

// DeleteWithResourceVersion deletes a ConfigMap when its resource version matches.
func (s *ConfigMapService) DeleteWithResourceVersion(clusterID uint64, namespace, name, expectedResourceVersion string, dataScope *common.DataScopeReq) error {
	if s.namespaceSvc == nil {
		return errors.New("k8s.namespace.binding_required")
	}
	if err := s.namespaceSvc.RequireWrite(clusterID, namespace, "configmap:delete"); err != nil {
		return err
	}
	clientset, err := s.clusterSvc.GetClientset(clusterID, dataScope)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), opTimeout)
	defer cancel()

	options := metav1.DeleteOptions{}
	if expectedResourceVersion != "" {
		options.Preconditions = &metav1.Preconditions{ResourceVersion: &expectedResourceVersion}
	}
	if err := clientset.CoreV1().ConfigMaps(namespace).Delete(ctx, name, options); err != nil {
		if apierrors.IsConflict(err) || apierrors.IsInvalid(err) {
			return common.NewConflict("k8s.configmap.resource_version_conflict")
		}
		return errors.New("k8s.configmap.delete_failed")
	}
	return nil
}

// ConfigMapHandler exposes ConfigMap HTTP endpoints.
//
//nolint:revive // retained as the public handler name for this package.
type ConfigMapHandler struct {
	svc *ConfigMapService
}

// NewConfigMapHandler creates the ConfigMap HTTP handler.
func NewConfigMapHandler(svc *ConfigMapService) *ConfigMapHandler {
	return &ConfigMapHandler{svc: svc}
}

// RegisterRoutes registers ConfigMap endpoints.
func (h *ConfigMapHandler) RegisterRoutes(r gin.IRoutes) {
	r.GET("/clusters/:id/configmaps", h.List)
	r.POST("/clusters/:id/configmaps", h.Create)
	r.GET("/clusters/:id/configmaps/:name", h.Get)
	r.DELETE("/clusters/:id/configmaps/:name", h.Delete)
}

// List handles ConfigMap listing.
func (h *ConfigMapHandler) List(c *gin.Context) {
	clusterID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	var query ConfigMapListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	resp, err := h.svc.List(clusterID, query, common.GetDataScope(c))
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "k8s.configmap.list_failed")
		return
	}
	common.Success(c, resp)
}

// Get handles ConfigMap retrieval.
func (h *ConfigMapHandler) Get(c *gin.Context) {
	clusterID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	resp, err := h.svc.Get(clusterID, c.Query("namespace"), c.Param("name"), common.GetDataScope(c))
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "k8s.configmap.not_found")
		return
	}
	common.Success(c, resp)
}

// Create handles ConfigMap creation.
func (h *ConfigMapHandler) Create(c *gin.Context) {
	common.SetAuditMetadata(c, "k8s.configmap.audit.create", common.BusinessInsert)

	clusterID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	var req CreateConfigMapRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	namespace := c.Query("namespace")
	if namespace == "" {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	resp, err := h.svc.Create(clusterID, namespace, req, common.GetDataScope(c))
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "k8s.configmap.create_failed")
		return
	}
	common.Success(c, resp)
}

// Delete handles ConfigMap deletion.
func (h *ConfigMapHandler) Delete(c *gin.Context) {
	common.SetAuditMetadata(c, "k8s.configmap.audit.delete", common.BusinessDelete)

	clusterID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	if err := h.svc.DeleteWithResourceVersion(clusterID, c.Query("namespace"), c.Param("name"), c.Query("resourceVersion"), common.GetDataScope(c)); err != nil {
		common.FailWithError(c, common.CodeError, err, "k8s.configmap.delete_failed")
		return
	}
	common.Success(c, nil)
}
