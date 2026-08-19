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

// SecretItem summarizes a Kubernetes Secret without exposing values.
//
//nolint:revive // DTO names retain the secret domain prefix for generated API clarity.
type SecretItem struct {
	Name            string `json:"name"`
	Namespace       string `json:"namespace"`
	Type            string `json:"type"`
	KeyCount        int    `json:"keyCount"`
	ResourceVersion string `json:"resourceVersion"`
}

// SecretListResponse contains Kubernetes Secret summaries.
//
//nolint:revive // DTO names retain the secret domain prefix for generated API clarity.
type SecretListResponse struct {
	Items         []SecretItem `json:"items"`
	Total         int          `json:"total"`
	ContinueToken string       `json:"continueToken"`
}

// SecretDetail contains Secret metadata and key names, never values.
//
//nolint:revive // DTO names retain the secret domain prefix for generated API clarity.
type SecretDetail struct {
	Name            string   `json:"name"`
	Namespace       string   `json:"namespace"`
	Type            string   `json:"type"`
	Keys            []string `json:"keys"`
	ResourceVersion string   `json:"resourceVersion"`
}

type SecretListQuery struct {
	Namespace     string `form:"namespace"`
	Limit         int64  `form:"limit"`
	ContinueToken string `form:"continue"`
}

// CreateSecretRequest contains fields accepted when creating a Secret.
type CreateSecretRequest struct {
	Name string            `json:"name" binding:"required"`
	Type string            `json:"type"`
	Data map[string]string `json:"data"`
}

// SecretService manages live Kubernetes Secrets through client-go.
//
//nolint:revive // Service names retain the secret domain prefix for module wiring clarity.
type SecretService struct {
	clusterSvc   *cluster.ClusterService
	namespaceSvc *namespace.NamespaceService
}

// NewSecretService creates a Kubernetes Secret service.
func NewSecretService(clusterSvc *cluster.ClusterService, namespaceSvc *namespace.NamespaceService) *SecretService {
	return &SecretService{clusterSvc: clusterSvc, namespaceSvc: namespaceSvc}
}

// List returns Secret metadata visible in a namespace.
func (s *SecretService) List(clusterID uint64, query SecretListQuery, dataScope *common.DataScopeReq) (*SecretListResponse, error) {
	clientset, err := s.clusterSvc.GetClientset(clusterID, dataScope)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), opTimeout)
	defer cancel()

	list, err := clientset.CoreV1().Secrets(query.Namespace).List(ctx, metav1.ListOptions{Limit: normalizeListLimit(query.Limit), Continue: query.ContinueToken})
	if err != nil {
		return nil, errors.New("k8s.secret.list_failed")
	}
	items := make([]SecretItem, 0, len(list.Items))
	for i := range list.Items {
		sc := &list.Items[i]
		items = append(items, SecretItem{Name: sc.Name, Namespace: sc.Namespace, Type: string(sc.Type), KeyCount: len(sc.Data), ResourceVersion: sc.ResourceVersion})
	}
	return &SecretListResponse{Items: items, Total: len(items), ContinueToken: list.Continue}, nil
}

// Get returns Secret metadata and key names without values.
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
	return &SecretDetail{Name: sc.Name, Namespace: sc.Namespace, Type: string(sc.Type), Keys: keys, ResourceVersion: sc.ResourceVersion}, nil
}

// Create creates a Secret and returns metadata plus key names.
func (s *SecretService) Create(clusterID uint64, namespace string, req CreateSecretRequest, dataScope *common.DataScopeReq) (*SecretDetail, error) {
	if s.namespaceSvc == nil {
		return nil, errors.New("k8s.namespace.binding_required")
	}
	if err := s.namespaceSvc.RequireWrite(clusterID, namespace, "secret:create"); err != nil {
		return nil, err
	}
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
	return &SecretDetail{Name: created.Name, Namespace: created.Namespace, Type: string(created.Type), Keys: keys, ResourceVersion: created.ResourceVersion}, nil
}

// Delete removes a Secret from a namespace.
func (s *SecretService) Delete(clusterID uint64, namespace, name string, dataScope *common.DataScopeReq) error {
	return s.DeleteWithResourceVersion(clusterID, namespace, name, "", dataScope)
}

func (s *SecretService) DeleteWithResourceVersion(clusterID uint64, namespace, name, expectedResourceVersion string, dataScope *common.DataScopeReq) error {
	if s.namespaceSvc == nil {
		return errors.New("k8s.namespace.binding_required")
	}
	if err := s.namespaceSvc.RequireWrite(clusterID, namespace, "secret:delete"); err != nil {
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
	if err := clientset.CoreV1().Secrets(namespace).Delete(ctx, name, options); err != nil {
		if apierrors.IsConflict(err) || apierrors.IsInvalid(err) {
			return common.NewConflict("k8s.secret.resource_version_conflict")
		}
		return errors.New("k8s.secret.delete_failed")
	}
	return nil
}

// SecretHandler exposes Kubernetes Secret routes.
//
//nolint:revive // Handler names retain the secret domain prefix for module wiring clarity.
type SecretHandler struct {
	svc *SecretService
}

// NewSecretHandler creates a Secret route handler.
func NewSecretHandler(svc *SecretService) *SecretHandler {
	return &SecretHandler{svc: svc}
}

// RegisterRoutes registers Secret endpoints.
func (h *SecretHandler) RegisterRoutes(r gin.IRoutes) {
	r.GET("/clusters/:id/secrets", h.List)
	r.POST("/clusters/:id/secrets", h.Create)
	r.GET("/clusters/:id/secrets/:name", h.Get)
	r.DELETE("/clusters/:id/secrets/:name", h.Delete)
}

// List handles Secret metadata listing.
func (h *SecretHandler) List(c *gin.Context) {
	clusterID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	var query SecretListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	resp, err := h.svc.List(clusterID, query, common.GetDataScope(c))
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "k8s.secret.list_failed")
		return
	}
	common.Success(c, resp)
}

// Get handles Secret metadata lookup.
func (h *SecretHandler) Get(c *gin.Context) {
	clusterID, err := strconv.ParseUint(c.Param("id"), 10, 64)
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

// Create handles Secret creation.
func (h *SecretHandler) Create(c *gin.Context) {
	common.SetAuditMetadata(c, "k8s.secret.audit.create", common.BusinessInsert)

	clusterID, err := strconv.ParseUint(c.Param("id"), 10, 64)
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

// Delete handles Secret deletion.
func (h *SecretHandler) Delete(c *gin.Context) {
	common.SetAuditMetadata(c, "k8s.secret.audit.delete", common.BusinessDelete)

	clusterID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	if err := h.svc.DeleteWithResourceVersion(clusterID, c.Query("namespace"), c.Param("name"), c.Query("resourceVersion"), common.GetDataScope(c)); err != nil {
		common.FailWithError(c, common.CodeError, err, "k8s.secret.delete_failed")
		return
	}
	common.Success(c, nil)
}
