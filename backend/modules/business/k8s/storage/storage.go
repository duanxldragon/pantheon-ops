// Package storage provides DB-snapshot management of Kubernetes PersistentVolume,
// PersistentVolumeClaim and StorageClass resources. Rows mirror cluster state
// at the last explicit sync.
package storage

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"pantheon-base/modules/business/k8s/cluster"
	"pantheon-base/modules/business/k8s/namespace"
	"pantheon-base/modules/business/k8s/rbac"
	"pantheon-base/pkg/common"
	"pantheon-base/pkg/database"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	msgParamInvalid = "common.param_invalid"
	opTimeout       = 30 * time.Second
	defaultClassKey = "storageclass.kubernetes.io/is-default-class"
)

// PV is the DB snapshot of a Kubernetes PersistentVolume.
type PV struct {
	ID              uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	Name            string         `gorm:"size:255;not null;index" json:"name"`
	ClusterID       uint64         `gorm:"column:cluster_id;index" json:"clusterId"`
	BusinessScopeID uint64         `gorm:"column:business_scope_id;index" json:"-"`
	DeptID          uint64         `gorm:"column:dept_id;index" json:"-"`
	Capacity        string         `gorm:"size:64" json:"capacity"`
	StorageClass    string         `gorm:"column:storage_class;size:128" json:"storageClass"`
	AccessModes     datatypes.JSON `gorm:"column:access_modes;type:json" json:"accessModes"`
	ReclaimPolicy   string         `gorm:"column:reclaim_policy;size:64" json:"reclaimPolicy"`
	Status          string         `gorm:"size:64;index" json:"status"`
	ClaimRef        string         `gorm:"column:claim_ref;size:255" json:"claimRef"`
	VolumeMode      string         `gorm:"column:volume_mode;size:64" json:"volumeMode"`
	CreatedAt       time.Time      `json:"createdAt"`
	UpdatedAt       time.Time      `json:"updatedAt"`
}

// TableName returns the PV snapshot table name.
func (PV) TableName() string { return "biz_k8s_pv" }

// PVC is the DB snapshot of a Kubernetes PersistentVolumeClaim.
type PVC struct {
	ID              uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	Name            string         `gorm:"size:255;not null;index" json:"name"`
	Namespace       string         `gorm:"size:255;not null;index" json:"namespace"`
	ClusterID       uint64         `gorm:"column:cluster_id;index" json:"clusterId"`
	BusinessScopeID uint64         `gorm:"column:business_scope_id;index" json:"-"`
	DeptID          uint64         `gorm:"column:dept_id;index" json:"-"`
	PVName          string         `gorm:"column:pv_name;size:255" json:"pvName"`
	StorageClass    string         `gorm:"column:storage_class;size:128" json:"storageClass"`
	RequestStorage  string         `gorm:"column:request_storage;size:64" json:"requestStorage"`
	AccessModes     datatypes.JSON `gorm:"column:access_modes;type:json" json:"accessModes"`
	VolumeMode      string         `gorm:"column:volume_mode;size:64" json:"volumeMode"`
	Status          string         `gorm:"size:64;index" json:"status"`
	CreatedAt       time.Time      `json:"createdAt"`
	UpdatedAt       time.Time      `json:"updatedAt"`
}

// TableName returns the PVC snapshot table name.
func (PVC) TableName() string { return "biz_k8s_pvc" }

// StorageClass is the DB snapshot of a Kubernetes StorageClass.
type StorageClass struct {
	ID                   uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	Name                 string    `gorm:"size:255;not null;index" json:"name"`
	ClusterID            uint64    `gorm:"column:cluster_id;index" json:"clusterId"`
	BusinessScopeID      uint64    `gorm:"column:business_scope_id;index" json:"-"`
	DeptID               uint64    `gorm:"column:dept_id;index" json:"-"`
	Provisioner          string    `gorm:"size:255" json:"provisioner"`
	ReclaimPolicy        string    `gorm:"column:reclaim_policy;size:64" json:"reclaimPolicy"`
	VolumeBindingMode    string    `gorm:"column:volume_binding_mode;size:64" json:"volumeBindingMode"`
	AllowVolumeExpansion bool      `gorm:"column:allow_volume_expansion;default:false" json:"allowVolumeExpansion"`
	IsDefault            bool      `gorm:"column:is_default;default:false" json:"isDefault"`
	CreatedAt            time.Time `json:"createdAt"`
	UpdatedAt            time.Time `json:"updatedAt"`
}

// TableName returns the StorageClass snapshot table name.
func (StorageClass) TableName() string { return "biz_k8s_storage_class" }

// ListQuery carries shared storage list filters.
type ListQuery struct {
	ClusterID    uint64 `form:"clusterId"`
	StorageClass string `form:"storageClass"`
	Status       string `form:"status"`
	Name         string `form:"name"`
	Namespace    string `form:"namespace"`
}

// SyncResponse reports how many snapshot rows were written.
type SyncResponse struct {
	Synced int `json:"synced"`
}

// SyncRequest carries the target cluster id.
type SyncRequest struct {
	ClusterID uint64 `json:"clusterId" binding:"required"`
}

// ListResponse contains snapshot rows.
type ListResponse struct {
	Items interface{} `json:"items"`
	Total int64       `json:"total"`
}

// StorageService manages Kubernetes storage snapshots.
//
//nolint:revive // retained as the public service name for this package.
type StorageService struct {
	db           *gorm.DB
	clusterSvc   *cluster.ClusterService
	namespaceSvc *namespace.NamespaceService
}

// NewStorageService creates a storage snapshot service.
func NewStorageService(db *gorm.DB, clusterSvc *cluster.ClusterService, namespaceSvc *namespace.NamespaceService) *StorageService {
	return &StorageService{db: db, clusterSvc: clusterSvc, namespaceSvc: namespaceSvc}
}

// Migrate creates the storage snapshot tables.
func (s *StorageService) Migrate() error {
	if s.db == nil {
		return errors.New("database.not_initialized")
	}
	return s.db.AutoMigrate(&PV{}, &PVC{}, &StorageClass{})
}

// ListPVs returns PV snapshots.
func (s *StorageService) ListPVs(query ListQuery, dataScope *common.DataScopeReq) (*ListResponse, error) {
	db := s.snapshotQuery(&PV{}, query.ClusterID, dataScope)
	if query.StorageClass != "" {
		db = db.Where("storage_class = ?", query.StorageClass)
	}
	if query.Status != "" {
		db = db.Where("status = ?", query.Status)
	}
	if query.Name != "" {
		db = db.Where("name = ?", query.Name)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}
	var items []PV
	if err := db.Order("id DESC").Find(&items).Error; err != nil {
		return nil, err
	}
	return &ListResponse{Items: items, Total: total}, nil
}

// ListPVCs returns PVC snapshots.
func (s *StorageService) ListPVCs(query ListQuery, dataScope *common.DataScopeReq) (*ListResponse, error) {
	db := s.snapshotQuery(&PVC{}, query.ClusterID, dataScope)
	if query.StorageClass != "" {
		db = db.Where("storage_class = ?", query.StorageClass)
	}
	if query.Status != "" {
		db = db.Where("status = ?", query.Status)
	}
	if query.Name != "" {
		db = db.Where("name = ?", query.Name)
	}
	if query.Namespace != "" {
		db = db.Where("namespace = ?", query.Namespace)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}
	var items []PVC
	if err := db.Order("id DESC").Find(&items).Error; err != nil {
		return nil, err
	}
	return &ListResponse{Items: items, Total: total}, nil
}

// ListStorageClasses returns StorageClass snapshots.
func (s *StorageService) ListStorageClasses(query ListQuery, dataScope *common.DataScopeReq) (*ListResponse, error) {
	db := s.snapshotQuery(&StorageClass{}, query.ClusterID, dataScope)
	if query.Name != "" {
		db = db.Where("name = ?", query.Name)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}
	var items []StorageClass
	if err := db.Order("id DESC").Find(&items).Error; err != nil {
		return nil, err
	}
	return &ListResponse{Items: items, Total: total}, nil
}

// SyncPVs refreshes PV snapshots for one cluster.
func (s *StorageService) SyncPVs(clusterID uint64, dataScope *common.DataScopeReq) (int, error) {
	clientset, scope, err := s.clientScope(clusterID, dataScope)
	if err != nil {
		return 0, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), opTimeout)
	defer cancel()

	list, err := clientset.CoreV1().PersistentVolumes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return 0, errors.New("k8s.storage.pv_sync_failed")
	}
	rows := make([]PV, 0, len(list.Items))
	for i := range list.Items {
		pv := &list.Items[i]
		rows = append(rows, PV{
			Name:            pv.Name,
			ClusterID:       clusterID,
			BusinessScopeID: scope.businessScopeID,
			DeptID:          scope.deptID,
			Capacity:        quantity(pv.Spec.Capacity, "storage"),
			StorageClass:    pv.Spec.StorageClassName,
			AccessModes:     jsonList(accessModes(pv.Spec.AccessModes)),
			ReclaimPolicy:   string(pv.Spec.PersistentVolumeReclaimPolicy),
			Status:          string(pv.Status.Phase),
			ClaimRef:        claimRefName(pv.Spec.ClaimRef),
			VolumeMode:      volumeMode(pv.Spec.VolumeMode),
		})
	}
	return len(rows), s.replaceRows(&PV{}, clusterID, rows)
}

// SyncPVCs refreshes PVC snapshots for one cluster.
func (s *StorageService) SyncPVCs(clusterID uint64, dataScope *common.DataScopeReq) (int, error) {
	clientset, scope, err := s.clientScope(clusterID, dataScope)
	if err != nil {
		return 0, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), opTimeout)
	defer cancel()

	list, err := clientset.CoreV1().PersistentVolumeClaims("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return 0, errors.New("k8s.storage.pvc_sync_failed")
	}
	rows := make([]PVC, 0, len(list.Items))
	for i := range list.Items {
		pvc := &list.Items[i]
		rows = append(rows, PVC{
			Name:            pvc.Name,
			Namespace:       pvc.Namespace,
			ClusterID:       clusterID,
			BusinessScopeID: scope.businessScopeID,
			DeptID:          scope.deptID,
			PVName:          pvc.Spec.VolumeName,
			StorageClass:    storageClassName(pvc.Spec.StorageClassName),
			RequestStorage:  quantity(pvc.Spec.Resources.Requests, "storage"),
			AccessModes:     jsonList(accessModes(pvc.Spec.AccessModes)),
			VolumeMode:      volumeMode(pvc.Spec.VolumeMode),
			Status:          string(pvc.Status.Phase),
		})
	}
	return len(rows), s.replaceRows(&PVC{}, clusterID, rows)
}

// SyncStorageClasses refreshes StorageClass snapshots for one cluster.
func (s *StorageService) SyncStorageClasses(clusterID uint64, dataScope *common.DataScopeReq) (int, error) {
	clientset, scope, err := s.clientScope(clusterID, dataScope)
	if err != nil {
		return 0, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), opTimeout)
	defer cancel()

	list, err := clientset.StorageV1().StorageClasses().List(ctx, metav1.ListOptions{})
	if err != nil {
		return 0, errors.New("k8s.storage.storage_class_sync_failed")
	}
	rows := make([]StorageClass, 0, len(list.Items))
	for i := range list.Items {
		sc := &list.Items[i]
		reclaimPolicy := corev1.PersistentVolumeReclaimDelete
		if sc.ReclaimPolicy != nil {
			reclaimPolicy = *sc.ReclaimPolicy
		}
		bindingMode := ""
		if sc.VolumeBindingMode != nil {
			bindingMode = string(*sc.VolumeBindingMode)
		}
		rows = append(rows, StorageClass{
			Name:                 sc.Name,
			ClusterID:            clusterID,
			BusinessScopeID:      scope.businessScopeID,
			DeptID:               scope.deptID,
			Provisioner:          sc.Provisioner,
			ReclaimPolicy:        string(reclaimPolicy),
			VolumeBindingMode:    bindingMode,
			AllowVolumeExpansion: sc.AllowVolumeExpansion != nil && *sc.AllowVolumeExpansion,
			IsDefault:            sc.Annotations[defaultClassKey] == "true",
		})
	}
	return len(rows), s.replaceRows(&StorageClass{}, clusterID, rows)
}

// DeletePV removes one PV snapshot.
func (s *StorageService) DeletePV(id uint64, dataScope *common.DataScopeReq) error {
	return s.deleteSnapshot(&PV{}, id, dataScope, true)
}

// DeletePVC removes one PVC snapshot after verifying the namespace binding.
func (s *StorageService) DeletePVC(id uint64, dataScope *common.DataScopeReq) error {
	return s.deleteSnapshot(&PVC{}, id, dataScope, true)
}

// DeleteStorageClass removes one StorageClass snapshot.
func (s *StorageService) DeleteStorageClass(id uint64, dataScope *common.DataScopeReq) error {
	return s.deleteSnapshot(&StorageClass{}, id, dataScope, false)
}

type clusterScope struct {
	businessScopeID uint64
	deptID          uint64
}

func (s *StorageService) clientScope(clusterID uint64, dataScope *common.DataScopeReq) (*kubernetes.Clientset, clusterScope, error) {
	if s.clusterSvc == nil {
		return nil, clusterScope{}, errors.New("k8s.storage.cluster_required")
	}
	clientset, err := s.clusterSvc.GetClientset(clusterID, dataScope)
	if err != nil {
		return nil, clusterScope{}, err
	}
	scope := clusterScope{}
	if s.db != nil {
		var row struct {
			BusinessScopeID uint64
			DeptID          uint64
		}
		if s.db.Table("biz_k8s_cluster").Select("business_scope_id, dept_id").Where("id = ?", clusterID).Scan(&row).Error == nil {
			scope.businessScopeID = row.BusinessScopeID
			scope.deptID = row.DeptID
		}
	}
	return clientset, scope, nil
}

func (s *StorageService) snapshotQuery(model interface{}, clusterID uint64, dataScope *common.DataScopeReq) *gorm.DB {
	db := s.db.Model(model).Scopes(database.WithDataScope(dataScope))
	if clusterID > 0 {
		db = db.Where("cluster_id = ?", clusterID)
	}
	return db
}

func (s *StorageService) replaceRows(model interface{}, clusterID uint64, rows interface{}) error {
	if s.db == nil {
		return errors.New("database.not_initialized")
	}
	// Replace the cluster snapshot: existing rows are stale after refresh.
	if err := s.db.Where("cluster_id = ?", clusterID).Delete(model).Error; err != nil {
		return err
	}
	return s.db.Create(rows).Error
}

func (s *StorageService) deleteSnapshot(model interface{}, id uint64, dataScope *common.DataScopeReq, requireBinding bool) error {
	if s.db == nil {
		return errors.New("database.not_initialized")
	}
	if requireBinding && s.namespaceSvc != nil {
		var row struct {
			ClusterID uint64
			Namespace string
		}
		if err := s.db.Table(s.tableName(model)).Where("id = ?", id).Scan(&row).Error; err != nil {
			return err
		}
		if err := s.namespaceSvc.RequireWrite(row.ClusterID, row.Namespace, "delete"); err != nil {
			return err
		}
	}
	return s.db.Scopes(database.WithDataScope(dataScope)).Delete(model, id).Error
}

func (s *StorageService) tableName(model interface{}) string {
	if t, ok := model.(interface{ TableName() string }); ok {
		return t.TableName()
	}
	return ""
}

// StorageHandler exposes storage snapshot endpoints.
//
//nolint:revive // retained as the public handler name for this package.
type StorageHandler struct {
	svc     *StorageService
	rbacSvc *rbac.K8sRBACService
}

// NewStorageHandler creates the storage HTTP handler.
func NewStorageHandler(svc *StorageService, rbacSvc *rbac.K8sRBACService) *StorageHandler {
	return &StorageHandler{svc: svc, rbacSvc: rbacSvc}
}

// RegisterRoutes registers storage snapshot endpoints.
func (h *StorageHandler) RegisterRoutes(r gin.IRoutes) {
	r.GET("/storage/pvs", h.ListPVs)
	r.POST("/storage/pvs/sync", h.SyncPVs)
	r.DELETE("/storage/pvs/:id", h.DeletePV)
	r.GET("/storage/pvcs", h.ListPVCs)
	r.POST("/storage/pvcs/sync", h.SyncPVCs)
	r.DELETE("/storage/pvcs/:id", h.DeletePVC)
	r.GET("/storage/storage-classes", h.ListStorageClasses)
	r.POST("/storage/storage-classes/sync", h.SyncStorageClasses)
}

// ListPVs handles PV snapshot listing.
func (h *StorageHandler) ListPVs(c *gin.Context) {
	h.list(c, h.svc.ListPVs)
}

// ListPVCs handles PVC snapshot listing.
func (h *StorageHandler) ListPVCs(c *gin.Context) {
	h.list(c, h.svc.ListPVCs)
}

// ListStorageClasses handles StorageClass snapshot listing.
func (h *StorageHandler) ListStorageClasses(c *gin.Context) {
	h.list(c, h.svc.ListStorageClasses)
}

func (h *StorageHandler) list(c *gin.Context, fn func(ListQuery, *common.DataScopeReq) (*ListResponse, error)) {
	var query ListQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	resp, err := fn(query, common.GetDataScope(c))
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "k8s.storage.list_failed")
		return
	}
	common.Success(c, resp)
}

// SyncPVs handles PV snapshot refresh.
func (h *StorageHandler) SyncPVs(c *gin.Context) {
	h.sync(c, "k8s.storage.audit.sync_pv", h.svc.SyncPVs)
}

// SyncPVCs handles PVC snapshot refresh.
func (h *StorageHandler) SyncPVCs(c *gin.Context) {
	h.sync(c, "k8s.storage.audit.sync_pvc", h.svc.SyncPVCs)
}

// SyncStorageClasses handles StorageClass snapshot refresh.
func (h *StorageHandler) SyncStorageClasses(c *gin.Context) {
	h.sync(c, "k8s.storage.audit.sync_storage_class", h.svc.SyncStorageClasses)
}

func (h *StorageHandler) sync(c *gin.Context, auditTitle string, fn func(uint64, *common.DataScopeReq) (int, error)) {
	common.SetAuditMetadata(c, auditTitle, common.BusinessUpdate)

	var req SyncRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	if h.rbacSvc != nil && !h.rbacSvc.CheckHandlerPermission(c, req.ClusterID, "", rbac.ResourceTypeStorage, rbac.ActionUpdate) {
		common.Fail(c, common.CodeForbidden, "k8s.rbac.permission_denied")
		return
	}
	rows, err := fn(req.ClusterID, common.GetDataScope(c))
	if err != nil {
		common.FailWithError(c, common.CodeError, err, "k8s.storage.sync_failed")
		return
	}
	common.Success(c, SyncResponse{Synced: rows})
}

// DeletePV handles PV snapshot deletion.
func (h *StorageHandler) DeletePV(c *gin.Context) {
	h.delete(c, "k8s.storage.audit.delete_pv", h.svc.DeletePV)
}

// DeletePVC handles PVC snapshot deletion.
func (h *StorageHandler) DeletePVC(c *gin.Context) {
	h.delete(c, "k8s.storage.audit.delete_pvc", h.svc.DeletePVC)
}

// DeleteStorageClass handles StorageClass snapshot deletion.
func (h *StorageHandler) DeleteStorageClass(c *gin.Context) {
	h.delete(c, "k8s.storage.audit.delete_storage_class", h.svc.DeleteStorageClass)
}

func (h *StorageHandler) delete(c *gin.Context, auditTitle string, fn func(uint64, *common.DataScopeReq) error) {
	common.SetAuditMetadata(c, auditTitle, common.BusinessDelete)

	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	if h.rbacSvc != nil && !h.rbacSvc.CheckHandlerPermission(c, 0, "", rbac.ResourceTypeStorage, rbac.ActionDelete) {
		common.Fail(c, common.CodeForbidden, "k8s.rbac.permission_denied")
		return
	}
	if err := fn(id, common.GetDataScope(c)); err != nil {
		common.FailWithError(c, common.CodeError, err, "k8s.storage.delete_failed")
		return
	}
	common.Success(c, nil)
}

func accessModes(modes []corev1.PersistentVolumeAccessMode) []string {
	out := make([]string, 0, len(modes))
	for _, m := range modes {
		out = append(out, string(m))
	}
	return out
}

func volumeMode(mode *corev1.PersistentVolumeMode) string {
	if mode == nil {
		return ""
	}
	return string(*mode)
}

func claimRefName(ref *corev1.ObjectReference) string {
	if ref == nil {
		return ""
	}
	return ref.Namespace + "/" + ref.Name
}

func quantity(res corev1.ResourceList, key corev1.ResourceName) string {
	if res == nil {
		return ""
	}
	q, ok := res[key]
	if !ok {
		return ""
	}
	return q.String()
}

func storageClassName(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func jsonList(v []string) datatypes.JSON {
	if v == nil {
		v = []string{}
	}
	b, _ := json.Marshal(v)
	return datatypes.JSON(b)
}
