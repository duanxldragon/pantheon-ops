// Package configmap provides live ConfigMap management for a k8s cluster.
package configmap

import (
	"context"
	"errors"
	"strconv"
	"time"

	"pantheon-base/modules/business/k8s/cluster"
	"pantheon-base/pkg/common"

	"github.com/gin-gonic/gin"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	msgParamInvalid = "common.param_invalid"
	opTimeout       = 30 * time.Second
)

type ConfigMapItem struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	KeyCount  int    `json:"keyCount"`
}

type ConfigMapListResponse struct {
	Items []ConfigMapItem `json:"items"`
	Total int             `json:"total"`
}

type ConfigMapDetail struct {
	Name      string            `json:"name"`
	Namespace string            `json:"namespace"`
	Data      map[string]string `json:"data"`
}

type CreateConfigMapRequest struct {
	Name string            `json:"name" binding:"required"`
	Data map[string]string `json:"data"`
}

type ConfigMapService struct {
	clusterSvc *cluster.ClusterService
}

func NewConfigMapService(clusterSvc *cluster.ClusterService) *ConfigMapService {
	return &ConfigMapService{clusterSvc: clusterSvc}
}

func (s *ConfigMapService) List(clusterID uint64, namespace string, dataScope *common.DataScopeReq) (*ConfigMapListResponse, error) {
	clientset, err := s.clusterSvc.GetClientset(clusterID, dataScope)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), opTimeout)
	defer cancel()

	list, err := clientset.CoreV1().ConfigMaps(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, errors.New("k8s.configmap.list_failed")
	}
	items := make([]ConfigMapItem, 0, len(list.Items))
	for i := range list.Items {
		cm := &list.Items[i]
		items = append(items, ConfigMapItem{Name: cm.Name, Namespace: cm.Namespace, KeyCount: len(cm.Data)})
	}
	return &ConfigMapListResponse{Items: items, Total: len(items)}, nil
}

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
	return &ConfigMapDetail{Name: cm.Name, Namespace: cm.Namespace, Data: cm.Data}, nil
}

func (s *ConfigMapService) Create(clusterID uint64, namespace string, req CreateConfigMapRequest, dataScope *common.DataScopeReq) (*ConfigMapDetail, error) {
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
	return &ConfigMapDetail{Name: created.Name, Namespace: created.Namespace, Data: created.Data}, nil
}

func (s *ConfigMapService) Delete(clusterID uint64, namespace, name string, dataScope *common.DataScopeReq) error {
	clientset, err := s.clusterSvc.GetClientset(clusterID, dataScope)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), opTimeout)
	defer cancel()

	if err := clientset.CoreV1().ConfigMaps(namespace).Delete(ctx, name, metav1.DeleteOptions{}); err != nil {
		return errors.New("k8s.configmap.delete_failed")
	}
	return nil
}

type ConfigMapHandler struct {
	svc *ConfigMapService
}

func NewConfigMapHandler(svc *ConfigMapService) *ConfigMapHandler {
	return &ConfigMapHandler{svc: svc}
}

func (h *ConfigMapHandler) RegisterRoutes(r gin.IRoutes) {
	r.GET("/clusters/:clusterId/configmaps", h.List)
	r.POST("/clusters/:clusterId/configmaps", h.Create)
	r.GET("/clusters/:clusterId/configmaps/:name", h.Get)
	r.DELETE("/clusters/:clusterId/configmaps/:name", h.Delete)
}

func (h *ConfigMapHandler) List(c *gin.Context) {
	clusterID, err := strconv.ParseUint(c.Param("clusterId"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	resp, err := h.svc.List(clusterID, c.Query("namespace"), common.GetDataScope(c))
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "k8s.configmap.list_failed")
		return
	}
	common.Success(c, resp)
}

func (h *ConfigMapHandler) Get(c *gin.Context) {
	clusterID, err := strconv.ParseUint(c.Param("clusterId"), 10, 64)
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

func (h *ConfigMapHandler) Create(c *gin.Context) {
	common.SetAuditMetadata(c, "k8s.configmap.audit.create", common.BusinessInsert)

	clusterID, err := strconv.ParseUint(c.Param("clusterId"), 10, 64)
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

func (h *ConfigMapHandler) Delete(c *gin.Context) {
	common.SetAuditMetadata(c, "k8s.configmap.audit.delete", common.BusinessDelete)

	clusterID, err := strconv.ParseUint(c.Param("clusterId"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	if err := h.svc.Delete(clusterID, c.Query("namespace"), c.Param("name"), common.GetDataScope(c)); err != nil {
		common.FailWithError(c, common.CodeError, err, "k8s.configmap.delete_failed")
		return
	}
	common.Success(c, nil)
}
