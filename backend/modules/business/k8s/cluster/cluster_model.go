package cluster

import (
	"time"

	"gorm.io/gorm"
)

// Cluster is a registered Kubernetes cluster managed by the k8s module. The
// kubeconfig is stored encrypted; it is only decrypted in memory when a
// clientset is built for an operation.
type Cluster struct {
	ID                        uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	Code                      string         `gorm:"size:128;not null;index:uk_k8s_cluster_code,unique" json:"code"`
	Name                      string         `gorm:"size:255;not null" json:"name"`
	Environment               string         `gorm:"size:50;not null;index" json:"environment"`
	BusinessScopeID           uint64         `gorm:"column:business_scope_id;index" json:"businessScopeId"`
	BusinessScopeName         string         `gorm:"column:business_scope_name;size:255" json:"businessScopeName"`
	DeptID                    uint64         `gorm:"column:dept_id;index" json:"deptId"`
	APIServer                 string         `gorm:"column:api_server;size:512" json:"apiServer"`
	KubeconfigEncrypted       string         `gorm:"column:kubeconfig_encrypted;type:text" json:"-"`
	KubeconfigCredentialRefID uint64         `gorm:"column:kubeconfig_credential_ref_id;index" json:"-"`
	Version                   string         `gorm:"size:64" json:"version"`
	Status                    string         `gorm:"size:50;not null;default:unknown;index" json:"status"`
	TotalNodes                int            `gorm:"column:total_nodes;default:0" json:"totalNodes"`
	ReadyNodes                int            `gorm:"column:ready_nodes;default:0" json:"readyNodes"`
	TotalPods                 int            `gorm:"column:total_pods;default:0" json:"totalPods"`
	RunningPods               int            `gorm:"column:running_pods;default:0" json:"runningPods"`
	CPUCapacity               float64        `gorm:"column:cpu_capacity" json:"cpuCapacity"`
	CPUAllocatable            float64        `gorm:"column:cpu_allocatable" json:"cpuAllocatable"`
	MemoryCapacity            float64        `gorm:"column:memory_capacity" json:"memoryCapacity"`
	MemoryAllocatable         float64        `gorm:"column:memory_allocatable" json:"memoryAllocatable"`
	LastSyncedAt              *time.Time     `gorm:"column:last_synced_at" json:"lastSyncedAt"`
	Remark                    string         `gorm:"type:text" json:"remark"`
	CreatedAt                 time.Time      `json:"createdAt"`
	UpdatedAt                 time.Time      `json:"updatedAt"`
	CreatedBy                 string         `gorm:"size:64" json:"createdBy"`
	UpdatedBy                 string         `gorm:"size:64" json:"updatedBy"`
	DeletedAt                 gorm.DeletedAt `gorm:"index:uk_k8s_cluster_code,unique" json:"-"`
}

// ClusterCredentialRef stores encrypted kubeconfig material separately from
// cluster metadata. Secret material never crosses the API DTO boundary.
type ClusterCredentialRef struct {
	ID        uint64 `gorm:"primaryKey;autoIncrement"`
	ClusterID uint64 `gorm:"not null;index"`
	Encrypted string `gorm:"column:encrypted;type:text;not null" json:"-"`
	Version   uint64 `gorm:"not null;default:1"`
	Status    string `gorm:"size:32;not null;default:active"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (ClusterCredentialRef) TableName() string { return "biz_k8s_cluster_credential_ref" }

// TableName returns the Kubernetes cluster table name.
func (Cluster) TableName() string {
	return "biz_k8s_cluster"
}

// NodeSnapshot is a transient view of a cluster node returned by the API. It
// is not persisted; node details are fetched live from the cluster.
type NodeSnapshot struct {
	Name            string `json:"name"`
	Status          string `json:"status"`
	InternalIP      string `json:"internalIp"`
	OS              string `json:"os"`
	KubeletVersion  string `json:"kubeletVersion"`
	CPUCapacity     string `json:"cpuCapacity"`
	MemoryCapacity  string `json:"memoryCapacity"`
	PodCapacity     int64  `json:"podCapacity"`
	AllocatablePods int64  `json:"allocatablePods"`
}
