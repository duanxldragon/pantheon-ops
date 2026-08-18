package release

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// Release is the durable intent and observed outcome for one Kubernetes
// workload image change. Kubernetes remains the runtime source of truth; this
// record captures the immutable requested target and its observed rollout.
type Release struct {
	ID                  uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	Name                string         `gorm:"size:255;not null" json:"name"`
	ClusterID           uint64         `gorm:"column:cluster_id;index" json:"clusterId"`
	Namespace           string         `gorm:"size:255;not null" json:"namespace"`
	BusinessScopeID     uint64         `gorm:"column:business_scope_id;index" json:"businessScopeId"`
	DeptID              uint64         `gorm:"column:dept_id;index" json:"deptId"`
	ServiceID           uint64         `gorm:"column:service_id;index" json:"serviceId"`
	ServiceInstanceID   uint64         `gorm:"column:service_instance_id;index" json:"serviceInstanceId"`
	ServiceName         string         `gorm:"column:service_name;size:255" json:"serviceName"`
	ServiceInstanceName string         `gorm:"column:service_instance_name;size:255" json:"serviceInstanceName"`
	WorkloadType        string         `gorm:"column:workload_type;size:50;not null" json:"workloadType"`
	WorkloadName        string         `gorm:"column:workload_name;size:255;not null" json:"workloadName"`
	ContainerName       string         `gorm:"column:container_name;size:255" json:"containerName"`
	ImageBefore         string         `gorm:"column:image_before;size:512" json:"imageBefore"`
	ImageAfter          string         `gorm:"column:image_after;size:512;not null" json:"imageAfter"`
	Strategy            string         `gorm:"size:50;not null;default:RollingUpdate" json:"strategy"`
	Status              string         `gorm:"size:50;not null;default:pending;index" json:"status"`
	IdempotencyKey      string         `gorm:"column:idempotency_key;size:128;not null;uniqueIndex:uk_k8s_release_idempotency" json:"idempotencyKey"`
	RequestFingerprint  string         `gorm:"column:request_fingerprint;size:64;not null" json:"-"`
	RequestSnapshot     datatypes.JSON `gorm:"column:request_snapshot;type:json" json:"-"`
	PreviousReleaseID   uint64         `gorm:"column:previous_release_id;index" json:"previousReleaseId"`
	Attempt             int            `gorm:"not null;default:1" json:"attempt"`
	TargetGeneration    int64          `gorm:"column:target_generation" json:"targetGeneration"`
	TargetRevision      string         `gorm:"column:target_revision;size:255" json:"targetRevision"`
	ObservedGeneration  int64          `gorm:"column:observed_generation" json:"observedGeneration"`
	ObservedRevision    string         `gorm:"column:observed_revision;size:255" json:"observedRevision"`
	DesiredReplicas     int32          `gorm:"column:desired_replicas" json:"desiredReplicas"`
	UpdatedReplicas     int32          `gorm:"column:updated_replicas" json:"updatedReplicas"`
	AvailableReplicas   int32          `gorm:"column:available_replicas" json:"availableReplicas"`
	ReadyReplicas       int32          `gorm:"column:ready_replicas" json:"readyReplicas"`
	ConditionSummary    string         `gorm:"column:condition_summary;type:text" json:"conditionSummary"`
	ErrorMessage        string         `gorm:"column:error_message;type:text" json:"errorMessage"`
	StartedAt           *time.Time     `gorm:"column:started_at" json:"startedAt"`
	FinishedAt          *time.Time     `gorm:"column:finished_at" json:"finishedAt"`
	LastReconciledAt    *time.Time     `gorm:"column:last_reconciled_at" json:"lastReconciledAt"`
	CreatedBy           string         `gorm:"size:64" json:"createdBy"`
	CreatedAt           time.Time      `json:"createdAt"`
	UpdatedAt           time.Time      `json:"updatedAt"`
	DeletedAt           gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName returns the Kubernetes release table name.
func (Release) TableName() string {
	return "biz_k8s_release"
}
