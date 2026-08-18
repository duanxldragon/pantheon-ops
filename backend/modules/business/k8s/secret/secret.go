// Package secret provides live Secret management for a k8s cluster. Secret
// values are never returned to the client; only key names are exposed, so
// sensitive data stays in the cluster.
package secret

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

type SecretItem struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Type      string `json:"type"`
	KeyCount  int    `json:"keyCount"`
}

type SecretListResponse struct {
	Items []SecretItem `json:"items"`
	Total int          `json:"total"`
}

type SecretDetail struct {
	Name      string   `json:"name"`
	Namespace string   `json:"namespace"`
	Type      string   `json:"type"`
	Keys      []string `json:"keys"`
}

type CreateSecretRequest struct {
	Name string            `json:"name" binding:"required"`
	Type string            `json:"type"`
	Data map[string]string `json:"data"`
}

type SecretService struct {
	clusterSvc *cluster.ClusterService
}

func NewSecretService(clusterSvc *cluster.ClusterService) *SecretService {
	return &SecretService{clusterSvc: clusterSvc}
}

func (s *SecretService) List(clusterID uint64, namespace string, dataScope *common.DataScopeReq) (*SecretListResponse, error) {
	clientset, err := s.clusterSvc.GetClientset(clusterID, dataScope)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), opTimeout)
	defer cancel()

	list, err := clientset.CoreV1().Secrets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, errors.New("k8s.secret.list_failed")
	}
	items := make([]SecretItem, 0, len(list.Items))
	for i := range list.Items {
		sc := &list.Items[i]
		items = append(items, SecretItem{Name: sc.Name, Namespace: sc.Namespace, Type: string(sc.Type), KeyCount: len(sc.Data)})
	}
	return &SecretListResponse{Items: items, Total: len(items)}, nil
}

func (s *SecretService) Get(clusterID uint64, namespace, name string, dataScope *common.DataScopeReq) (*SecretDetail, error) {
	clientset, err := s.clusterSvc.GetClientset(clusterID, dataScope)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), opTimeout)
	defer cancel()

	sc, err := clientset.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, errors.New("k8s.secret.not_found")
	}
	keys := make([]string, 0, len(sc.Data))
	for k := range sc.Data {
		keys = append(keys, k)
	}
	return &SecretDetail{Name: sc.Name, Namespace: sc.Namespace, Type: string(sc.Type), Keys: keys}, nil
}

func (s *SecretService) Create(clusterID uint64, namespace string, req CreateSecretRequest, dataScope *common.DataScopeReq) (*SecretDetail, error) {
	clientset, err := s.clusterSvc.GetClientset(clusterID, dataScope)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), opTimeout)
	defer cancel()

	secretType := corev1.SecretType(req.Type)
	if secretType == "" {
		secretType = corev1.SecretTypeOpaque
	}
	data := make(map[string][]byte, len(req.Data))
	for k, v := range req.Data {
		data[k] = []byte(v)
	}

	sc := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: req.Name, Namespace: namespace},
		Type:       secretType,
		Data:       data,
	}
	created, err := clientset.CoreV1().Secrets(namespace).Create(ctx, sc, metav1.CreateOptions{})
	if err != nil {
		return nil, errors.New("k8s.secret.create_failed")
	}
	keys := make([]string, 0, len(created.Data))
	for k := range created.Data {
		keys = append(keys, k)
	}
	return &SecretDetail{Name: created.Name, Namespace: created.Namespace, Type: string(created.Type), Keys: keys}, nil
}

func (s *SecretService) Delete(clusterID uint64, namespace, name string, dataScope *common.DataScopeReq) error {
	clientset, err := s.clusterSvc.GetClientset(clusterID, dataScope)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), opTimeout)
	defer cancel()

	if err := clientset.CoreV1().Secrets(namespace).Delete(ctx, name, metav1.DeleteOptions{}); err != nil {
		return errors.New("k8s.secret.delete_failed")
	}
	return nil
}

type SecretHandler struct {
	svc *SecretService
}

func NewSecretHandler(svc *SecretService) *SecretHandler {
	return &SecretHandler{svc: svc}
}

func (h *SecretHandler) RegisterRoutes(r gin.IRoutes) {
	r.GET("/clusters/:clusterId/secrets", h.List)
	r.POST("/clusters/:clusterId/secrets", h.Create)
	r.GET("/clusters/:clusterId/secrets/:name", h.Get)
	r.DELETE("/clusters/:clusterId/secrets/:name", h.Delete)
}

func (h *SecretHandler) List(c *gin.Context) {
	clusterID, err := strconv.ParseUint(c.Param("clusterId"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	resp, err := h.svc.List(clusterID, c.Query("namespace"), common.GetDataScope(c))
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "k8s.secret.list_failed")
		return
	}
	common.Success(c, resp)
}

func (h *SecretHandler) Get(c *gin.Context) {
	clusterID, err := strconv.ParseUint(c.Param("clusterId"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	resp, err := h.svc.Get(clusterID, c.Query("namespace"), c.Param("name"), common.GetDataScope(c))
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "k8s.secret.not_found")
		return
	}
	common.Success(c, resp)
}

func (h *SecretHandler) Create(c *gin.Context) {
	common.SetAuditMetadata(c, "k8s.secret.audit.create", common.BusinessInsert)

	clusterID, err := strconv.ParseUint(c.Param("clusterId"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	var req CreateSecretRequest
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
		common.FailWithError(c, common.CodeError, err, "k8s.secret.create_failed")
		return
	}
	common.Success(c, resp)
}

func (h *SecretHandler) Delete(c *gin.Context) {
	common.SetAuditMetadata(c, "k8s.secret.audit.delete", common.BusinessDelete)

	clusterID, err := strconv.ParseUint(c.Param("clusterId"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	if err := h.svc.Delete(clusterID, c.Query("namespace"), c.Param("name"), common.GetDataScope(c)); err != nil {
		common.FailWithError(c, common.CodeError, err, "k8s.secret.delete_failed")
		return
	}
	common.Success(c, nil)
}
