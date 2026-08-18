package service

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	StatusActive   = "active"
	StatusInactive = "inactive"

	TargetTypeVM  = "vm"
	TargetTypeK8s = "k8s"

	InstanceStatusActive   = "active"
	InstanceStatusDisabled = "disabled"

	DesiredStateStopped = "stopped"
	DesiredStateRunning = "running"
	DesiredStateRetired = "retired"

	ObservedStateUnknown    = "unknown"
	ObservedStateInstalling = "installing"
	ObservedStateStopped    = "stopped"
	ObservedStateStarting   = "starting"
	ObservedStateRunning    = "running"
	ObservedStateStopping   = "stopping"
	ObservedStateUpgrading  = "upgrading"
	ObservedStateFailed     = "failed"
	ObservedStateRetired    = "retired"

	HealthStateUnknown   = "unknown"
	HealthStateHealthy   = "healthy"
	HealthStateDegraded  = "degraded"
	HealthStateUnhealthy = "unhealthy"
)

type Application struct {
	ID                uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	Code              string         `gorm:"size:128;not null" json:"code"`
	Name              string         `gorm:"size:255;not null" json:"name"`
	BusinessScopeID   uint64         `gorm:"column:business_scope_id;not null;index" json:"businessScopeId"`
	BusinessScopeName string         `gorm:"column:business_scope_name;size:255" json:"businessScopeName"`
	DeptID            uint64         `gorm:"column:dept_id;not null;index" json:"deptId"`
	Status            string         `gorm:"size:32;not null;default:active;index" json:"status"`
	Owner             string         `gorm:"size:128" json:"owner"`
	Remark            string         `gorm:"type:text" json:"remark"`
	CreatedAt         time.Time      `json:"createdAt"`
	UpdatedAt         time.Time      `json:"updatedAt"`
	CreatedBy         string         `gorm:"size:64" json:"createdBy"`
	UpdatedBy         string         `gorm:"size:64" json:"updatedBy"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Application) TableName() string { return "biz_application" }

type Service struct {
	ID            uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	ApplicationID uint64         `gorm:"column:application_id;not null;index" json:"applicationId"`
	Code          string         `gorm:"size:128;not null" json:"code"`
	Name          string         `gorm:"size:255;not null" json:"name"`
	RuntimeType   string         `gorm:"column:runtime_type;size:64;not null" json:"runtimeType"`
	Description   string         `gorm:"type:text" json:"description"`
	Status        string         `gorm:"size:32;not null;default:active;index" json:"status"`
	CreatedAt     time.Time      `json:"createdAt"`
	UpdatedAt     time.Time      `json:"updatedAt"`
	CreatedBy     string         `gorm:"size:64" json:"createdBy"`
	UpdatedBy     string         `gorm:"size:64" json:"updatedBy"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Service) TableName() string { return "biz_service" }

type ServiceInstance struct {
	ID               uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	ServiceID        uint64         `gorm:"column:service_id;not null;index" json:"serviceId"`
	Environment      string         `gorm:"size:32;not null" json:"environment"`
	TargetType       string         `gorm:"column:target_type;size:32;not null" json:"targetType"`
	HostID           uint64         `gorm:"column:host_id;index" json:"hostId"`
	K8sClusterID     uint64         `gorm:"column:k8s_cluster_id;index" json:"k8sClusterId"`
	Namespace        string         `gorm:"size:255" json:"namespace"`
	WorkloadKind     string         `gorm:"column:workload_kind;size:64" json:"workloadKind"`
	WorkloadName     string         `gorm:"column:workload_name;size:255" json:"workloadName"`
	DesiredVersion   string         `gorm:"column:desired_version;size:128" json:"desiredVersion"`
	CurrentVersion   string         `gorm:"column:current_version;size:128" json:"currentVersion"`
	RollbackVersion  string         `gorm:"column:rollback_version;size:128" json:"rollbackVersion"`
	DesiredState     string         `gorm:"column:desired_state;size:32;not null;default:stopped;index" json:"desiredState"`
	ObservedState    string         `gorm:"column:observed_state;size:32;not null;default:unknown;index" json:"observedState"`
	HealthState      string         `gorm:"column:health_state;size:32;not null;default:unknown;index" json:"healthState"`
	HealthDefinition datatypes.JSON `gorm:"column:health_definition;type:json" json:"healthDefinition"`
	HealthObservedAt *time.Time     `gorm:"column:health_observed_at" json:"healthObservedAt"`
	HealthMessage    string         `gorm:"column:health_message;size:512" json:"healthMessage"`
	HealthRevision   string         `gorm:"column:health_revision;size:128" json:"healthRevision"`
	LastTransitionID string         `gorm:"column:last_transition_id;size:128" json:"lastTransitionId"`
	LastTransitionAt *time.Time     `gorm:"column:last_transition_at" json:"lastTransitionAt"`
	LifecycleVersion uint64         `gorm:"column:lifecycle_version;not null;default:0" json:"lifecycleVersion"`
	Status           string         `gorm:"size:32;not null;default:active;index" json:"status"`
	CreatedAt        time.Time      `json:"createdAt"`
	UpdatedAt        time.Time      `json:"updatedAt"`
	CreatedBy        string         `gorm:"size:64" json:"createdBy"`
	UpdatedBy        string         `gorm:"size:64" json:"updatedBy"`
	DeletedAt        gorm.DeletedAt `gorm:"index" json:"-"`
}

func (ServiceInstance) TableName() string { return "biz_service_instance" }
