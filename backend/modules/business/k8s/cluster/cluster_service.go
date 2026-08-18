package cluster

import (
	"context"
	"errors"
	"fmt"
	"time"

	bizcap "pantheon-base/modules/business/capability"
	k8spkg "pantheon-base/modules/business/k8s/pkg"
	"pantheon-base/pkg/common"
	"pantheon-base/pkg/database"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	idWhereClause = "id = ?"
	syncTimeout   = 30 * time.Second
)

// ClusterService manages registered Kubernetes clusters and their clients.
//
//nolint:revive // retained as the public service name for this package.
type ClusterService struct {
	db               *gorm.DB
	referenceChecker ReferenceChecker
	bizScopeReader   bizcap.BizScopeReader
}

// ReferenceChecker checks whether a cluster is referenced by another business resource.
type ReferenceChecker interface {
	HasReferences(ctx context.Context, tx *gorm.DB, clusterID uint64) (bool, error)
}

// NewClusterService creates the Kubernetes cluster service.
func NewClusterService(db *gorm.DB, readers ...bizcap.BizScopeReader) *ClusterService {
	service := &ClusterService{db: db}
	if len(readers) > 0 {
		service.bizScopeReader = readers[0]
	}
	return service
}

// SetReferenceChecker configures the owner-module reference checker used before deletion.
func (s *ClusterService) SetReferenceChecker(checker ReferenceChecker) {
	s.referenceChecker = checker
}

// SetBizScopeReader configures the business-scope reader used during cluster updates.
func (s *ClusterService) SetBizScopeReader(reader bizcap.BizScopeReader) {
	s.bizScopeReader = reader
}

// Migrate creates or upgrades the Kubernetes cluster schema.
func (s *ClusterService) Migrate() error {
	if s.db == nil {
		return errors.New("database.not_initialized")
	}
	return s.db.AutoMigrate(&Cluster{})
}

func (s *ClusterService) clusterQuery(dataScope *common.DataScopeReq) *gorm.DB {
	return s.db.Model(&Cluster{}).Scopes(database.WithDataScope(dataScope))
}

// List returns Kubernetes clusters visible in the supplied data scope.
func (s *ClusterService) List(query ClusterListQuery, dataScope *common.DataScopeReq) (*ClusterListResponse, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.PageSize <= 0 || query.PageSize > 100 {
		query.PageSize = 10
	}

	db := s.clusterQuery(dataScope)
	if query.Keyword != "" {
		like := "%" + query.Keyword + "%"
		db = db.Where("code LIKE ? OR name LIKE ?", like, like)
	}
	if query.Environment != "" {
		db = db.Where("environment = ?", query.Environment)
	}
	if query.Status != "" {
		db = db.Where("status = ?", query.Status)
	}
	if query.BusinessScopeID > 0 {
		db = db.Where("business_scope_id = ?", query.BusinessScopeID)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}

	var clusters []Cluster
	offset := (query.Page - 1) * query.PageSize
	if err := db.Order("id DESC").Offset(offset).Limit(query.PageSize).Find(&clusters).Error; err != nil {
		return nil, err
	}

	items := make([]ClusterResponse, len(clusters))
	for i, c := range clusters {
		items[i] = toResponse(&c)
	}
	return &ClusterListResponse{Items: items, Total: total, Page: query.Page, PageSize: query.PageSize}, nil
}

// GetByID returns one Kubernetes cluster visible in the supplied data scope.
func (s *ClusterService) GetByID(id uint64, dataScope *common.DataScopeReq) (*ClusterResponse, error) {
	cluster, err := s.findCluster(id, dataScope)
	if err != nil {
		return nil, err
	}
	resp := toResponse(cluster)
	return &resp, nil
}

// Create registers a Kubernetes cluster after validating and encrypting its kubeconfig.
func (s *ClusterService) Create(req CreateClusterRequest, createdBy string, deptID uint64) (*ClusterResponse, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}
	if s.codeExists(req.Code, 0) {
		return nil, errors.New("k8s.cluster.code_exists")
	}

	// Validate the kubeconfig before persisting so registration fails fast.
	if _, err := k8spkg.NewClientFromKubeconfig(req.Kubeconfig); err != nil {
		return nil, err
	}
	encrypted, err := k8spkg.EncryptKubeconfig(req.Kubeconfig)
	if err != nil {
		return nil, err
	}

	businessScopeName := ""
	if req.BusinessScopeID > 0 {
		scope, err := s.getBusinessScope(req.BusinessScopeID)
		if err != nil {
			return nil, err
		}
		businessScopeName = scope.Name
	}

	cluster := Cluster{
		Code:                req.Code,
		Name:                req.Name,
		Environment:         req.Environment,
		BusinessScopeID:     req.BusinessScopeID,
		BusinessScopeName:   businessScopeName,
		DeptID:              deptID,
		APIServer:           k8spkg.ExtractAPIServer(req.Kubeconfig),
		KubeconfigEncrypted: encrypted,
		Status:              "unknown",
		Remark:              req.Remark,
		CreatedBy:           createdBy,
		UpdatedBy:           createdBy,
	}

	if err := s.db.Create(&cluster).Error; err != nil {
		return nil, err
	}
	resp := toResponse(&cluster)
	return &resp, nil
}

// Update changes a Kubernetes cluster registration.
func (s *ClusterService) Update(id uint64, req UpdateClusterRequest, updatedBy string, dataScope *common.DataScopeReq) (*ClusterResponse, error) {
	cluster, err := s.findCluster(id, dataScope)
	if err != nil {
		return nil, err
	}

	updates := map[string]interface{}{"updated_by": updatedBy, "updated_at": time.Now()}
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.Environment != nil {
		updates["environment"] = *req.Environment
	}
	if req.Remark != nil {
		updates["remark"] = *req.Remark
	}
	if req.BusinessScopeID != nil {
		if *req.BusinessScopeID == 0 {
			updates["business_scope_id"] = uint64(0)
			updates["business_scope_name"] = ""
		} else {
			scope, err := s.getBusinessScope(*req.BusinessScopeID)
			if err != nil {
				return nil, err
			}
			updates["business_scope_id"] = scope.ID
			updates["business_scope_name"] = scope.Name
		}
	}
	if req.Kubeconfig != nil && *req.Kubeconfig != "" {
		if _, err := k8spkg.NewClientFromKubeconfig(*req.Kubeconfig); err != nil {
			return nil, err
		}
		encrypted, err := k8spkg.EncryptKubeconfig(*req.Kubeconfig)
		if err != nil {
			return nil, err
		}
		updates["kubeconfig_encrypted"] = encrypted
		updates["api_server"] = k8spkg.ExtractAPIServer(*req.Kubeconfig)
	}

	if err := s.db.Model(cluster).Updates(updates).Error; err != nil {
		return nil, err
	}
	return s.GetByID(id, dataScope)
}

// Delete removes a Kubernetes cluster when no owner-module references remain.
func (s *ClusterService) Delete(id uint64, dataScope *common.DataScopeReq) error {
	if s.db == nil {
		return errors.New("database.not_initialized")
	}
	return s.WithClusterLock(id, dataScope, func(tx *gorm.DB, _ *Cluster) error {
		if s.referenceChecker != nil {
			hasReferences, err := s.referenceChecker.HasReferences(context.Background(), tx, id)
			if err != nil {
				return err
			}
			if hasReferences {
				return errors.New("k8s.cluster.has_references")
			}
		}
		result := tx.Where(idWhereClause, id).Delete(&Cluster{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return errors.New("k8s.cluster.not_found")
		}
		return nil
	})
}

// WithClusterLock serializes operations that mutate a cluster or create
// cluster-owned history. MySQL uses an advisory lock; all dialects also take
// a row lock inside the transaction.
func (s *ClusterService) WithClusterLock(id uint64, dataScope *common.DataScopeReq, action func(tx *gorm.DB, cluster *Cluster) error) error {
	if s.db == nil {
		return errors.New("database.not_initialized")
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		var item Cluster
		query := tx.Model(&Cluster{}).Scopes(database.WithDataScope(dataScope)).
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Where(idWhereClause)
		if err := query.First(&item, id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("k8s.cluster.not_found")
			}
			return err
		}

		lockName := fmt.Sprintf("pantheon:k8s:cluster:%d", id)
		locked := false
		if tx.Name() == "mysql" {
			var acquired int
			if err := tx.Raw("SELECT GET_LOCK(?, 10)", lockName).Scan(&acquired).Error; err != nil {
				return err
			}
			if acquired != 1 {
				return errors.New("k8s.cluster.lock_timeout")
			}
			locked = true
			defer func() {
				_ = tx.Exec("SELECT RELEASE_LOCK(?)", lockName).Error
			}()
		}
		_ = locked
		return action(tx, &item)
	})
}

// Sync reconnects to the cluster, refreshes its version, node and pod counts,
// and aggregate resource capacity. On any failure the cluster status is marked
// unreachable but the cluster record itself is preserved.
func (s *ClusterService) Sync(id uint64, dataScope *common.DataScopeReq) (*ClusterResponse, error) {
	cluster, err := s.findCluster(id, dataScope)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), syncTimeout)
	defer cancel()

	clientset, err := k8spkg.NewClientFromEncrypted(cluster.KubeconfigEncrypted)
	if err != nil {
		s.markUnreachable(cluster)
		return nil, err
	}

	info, err := collectClusterInfo(ctx, clientset)
	if err != nil {
		s.markUnreachable(cluster)
		return nil, errors.New("k8s.cluster.sync_failed")
	}

	now := time.Now()
	updates := map[string]interface{}{
		"status":             "healthy",
		"version":            info.Version,
		"total_nodes":        info.TotalNodes,
		"ready_nodes":        info.ReadyNodes,
		"total_pods":         info.TotalPods,
		"running_pods":       info.RunningPods,
		"cpu_capacity":       info.CPUCapacity,
		"cpu_allocatable":    info.CPUAllocatable,
		"memory_capacity":    info.MemoryCapacity,
		"memory_allocatable": info.MemoryAllocatable,
		"last_synced_at":     &now,
		"updated_at":         now,
	}
	if err := s.db.Model(cluster).Updates(updates).Error; err != nil {
		return nil, err
	}
	return s.GetByID(id, dataScope)
}

// GetNodes returns a live view of the cluster's nodes. It is not persisted.
func (s *ClusterService) GetNodes(id uint64, dataScope *common.DataScopeReq) (*NodeListResponse, error) {
	cluster, err := s.findCluster(id, dataScope)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), syncTimeout)
	defer cancel()

	clientset, err := k8spkg.NewClientFromEncrypted(cluster.KubeconfigEncrypted)
	if err != nil {
		return nil, err
	}

	nodeList, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, errors.New("k8s.cluster.sync_failed")
	}

	items := make([]NodeSnapshot, 0, len(nodeList.Items))
	for _, node := range nodeList.Items {
		items = append(items, toNodeSnapshot(&node))
	}
	return &NodeListResponse{Items: items, Total: len(items)}, nil
}

// GetClientset returns a live kubernetes clientset for a cluster after
// validating data scope access. Other k8s submodules (namespace, workload,
// release) reuse this to operate on the cluster without re-implementing
// kubeconfig decryption and access control.
func (s *ClusterService) GetClientset(id uint64, dataScope *common.DataScopeReq) (*kubernetes.Clientset, error) {
	cluster, err := s.findCluster(id, dataScope)
	if err != nil {
		return nil, err
	}
	return k8spkg.NewClientFromEncrypted(cluster.KubeconfigEncrypted)
}

// GetMeta returns the business scope and department ids for a cluster after
// validating data scope access. Release records inherit these ids so release
// history is filtered by the same boundaries as the cluster.
func (s *ClusterService) GetMeta(id uint64, dataScope *common.DataScopeReq) (businessScopeID, deptID uint64, err error) {
	cluster, err := s.findCluster(id, dataScope)
	if err != nil {
		return 0, 0, err
	}
	return cluster.BusinessScopeID, cluster.DeptID, nil
}

func (s *ClusterService) markUnreachable(cluster *Cluster) {
	_ = s.db.Model(cluster).Updates(map[string]interface{}{
		"status":     "unreachable",
		"updated_at": time.Now(),
	})
}

func (s *ClusterService) findCluster(id uint64, dataScope *common.DataScopeReq) (*Cluster, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}
	var cluster Cluster
	if err := s.clusterQuery(dataScope).First(&cluster, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("k8s.cluster.not_found")
		}
		return nil, err
	}
	return &cluster, nil
}

func (s *ClusterService) codeExists(code string, excludeID uint64) bool {
	var count int64
	q := s.db.Model(&Cluster{}).Where("code = ?", code)
	if excludeID > 0 {
		q = q.Where("id != ?", excludeID)
	}
	q.Count(&count)
	return count > 0
}

func (s *ClusterService) getBusinessScope(id uint64) (bizcap.BizScopeRef, error) {
	if id == 0 {
		return bizcap.BizScopeRef{}, errors.New("business.bizscope.notFound")
	}
	if s.bizScopeReader == nil {
		return bizcap.BizScopeRef{}, errors.New("business.bizscope.readerNotConfigured")
	}
	return s.bizScopeReader.GetActive(context.Background(), id, nil)
}

func toResponse(c *Cluster) ClusterResponse {
	lastSynced := ""
	if c.LastSyncedAt != nil {
		lastSynced = c.LastSyncedAt.Format(time.RFC3339)
	}
	return ClusterResponse{
		ID:                c.ID,
		Code:              c.Code,
		Name:              c.Name,
		Environment:       c.Environment,
		BusinessScopeID:   c.BusinessScopeID,
		BusinessScopeName: c.BusinessScopeName,
		DeptID:            c.DeptID,
		APIServer:         c.APIServer,
		Version:           c.Version,
		Status:            c.Status,
		TotalNodes:        c.TotalNodes,
		ReadyNodes:        c.ReadyNodes,
		TotalPods:         c.TotalPods,
		RunningPods:       c.RunningPods,
		CPUCapacity:       c.CPUCapacity,
		CPUAllocatable:    c.CPUAllocatable,
		MemoryCapacity:    c.MemoryCapacity,
		MemoryAllocatable: c.MemoryAllocatable,
		LastSyncedAt:      lastSynced,
		Remark:            c.Remark,
		CreatedAt:         c.CreatedAt.Format(time.RFC3339),
		UpdatedAt:         c.UpdatedAt.Format(time.RFC3339),
		CreatedBy:         c.CreatedBy,
		UpdatedBy:         c.UpdatedBy,
	}
}

type clusterInfo struct {
	Version           string
	TotalNodes        int
	ReadyNodes        int
	TotalPods         int
	RunningPods       int
	CPUCapacity       float64
	CPUAllocatable    float64
	MemoryCapacity    float64
	MemoryAllocatable float64
}

func collectClusterInfo(ctx context.Context, clientset kubernetes.Interface) (*clusterInfo, error) {
	versionInfo, err := clientset.Discovery().ServerVersion()
	if err != nil {
		return nil, err
	}
	nodes, err := clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	pods, err := clientset.CoreV1().Pods("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	info := &clusterInfo{
		Version:    versionInfo.GitVersion,
		TotalNodes: len(nodes.Items),
		TotalPods:  len(pods.Items),
	}
	for _, node := range nodes.Items {
		if isNodeReady(&node) {
			info.ReadyNodes++
		}
		info.CPUCapacity += quantityToCores(node.Status.Capacity.Cpu())
		info.CPUAllocatable += quantityToCores(node.Status.Allocatable.Cpu())
		info.MemoryCapacity += quantityToGB(node.Status.Capacity.Memory())
		info.MemoryAllocatable += quantityToGB(node.Status.Allocatable.Memory())
	}
	for _, pod := range pods.Items {
		if pod.Status.Phase == corev1.PodRunning {
			info.RunningPods++
		}
	}
	return info, nil
}

func isNodeReady(node *corev1.Node) bool {
	for _, cond := range node.Status.Conditions {
		if cond.Type == corev1.NodeReady {
			return cond.Status == corev1.ConditionTrue
		}
	}
	return false
}

func quantityToCores(q *resource.Quantity) float64 {
	if q == nil {
		return 0
	}
	return float64(q.MilliValue()) / 1000.0
}

func quantityToGB(q *resource.Quantity) float64 {
	if q == nil {
		return 0
	}
	return float64(q.Value()) / (1024 * 1024 * 1024)
}

func toNodeSnapshot(node *corev1.Node) NodeSnapshot {
	ip := ""
	os := node.Status.NodeInfo.OSImage
	for _, addr := range node.Status.Addresses {
		if addr.Type == corev1.NodeInternalIP {
			ip = addr.Address
			break
		}
	}
	status := "unknown"
	for _, cond := range node.Status.Conditions {
		if cond.Type == corev1.NodeReady {
			if cond.Status == corev1.ConditionTrue {
				status = "ready"
			} else {
				status = "not_ready"
			}
			break
		}
	}
	return NodeSnapshot{
		Name:            node.Name,
		Status:          status,
		InternalIP:      ip,
		OS:              os,
		KubeletVersion:  node.Status.NodeInfo.KubeletVersion,
		CPUCapacity:     node.Status.Capacity.Cpu().String(),
		MemoryCapacity:  node.Status.Capacity.Memory().String(),
		PodCapacity:     node.Status.Capacity.Pods().Value(),
		AllocatablePods: node.Status.Allocatable.Pods().Value(),
	}
}
