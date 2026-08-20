package deploy

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	PackageStatusEnabled      = "enabled"
	PackageStatusDisabled     = "disabled"
	ExecutionModeFixed        = "fixed"
	ExecutionModeOrchestrated = "orchestrated"
	TemplateCodeNginxSystemd  = "nginx_systemd"
	TemplateCodeMySQLSystemd  = "mysql_systemd"
	TemplateCodeRedisSystemd  = "redis_systemd"
	TemplateCodeMinIOSystemd  = "minio_systemd"
	TemplateCodeHarborOffline = "harbor_offline"
	TemplateStepTypePackage   = "package"
	TemplateStepTypeScript    = "script"

	TemplateStatusEnabled  = "enabled"
	TemplateStatusDisabled = "disabled"

	TaskActionInstall   = "install"
	TaskActionUninstall = "uninstall"
	TaskActionUpgrade   = "upgrade"
	TaskActionReinstall = "reinstall"
	TaskActionStart     = "start"
	TaskActionHealth    = "health"
	TaskActionStop      = "stop"
	TaskActionRollback  = "rollback"
	TaskActionRetire    = "retire"

	TaskStatusDraft    = "draft"
	TaskStatusPending  = "pending"
	TaskStatusRunning  = "running"
	TaskStatusSuccess  = "success"
	TaskStatusFailed   = "failed"
	TaskStatusCanceled = "canceled"

	TaskHostStatusPending = "pending"
	TaskHostStatusRunning = "running"
	TaskHostStatusSuccess = "success"
	TaskHostStatusFailed  = "failed"
	TaskHostStatusSkipped = "skipped"

	TargetTypeHost  = "host"
	TargetTypeGroup = "group"

	ExecutorTypeManual    = "manual"
	ExecutorTypeSimulated = "simulated"
	ExecutorTypeAgent     = "agent"
	ExecutorTypeSSH       = "ssh"
)

type DeployPackage struct {
	ID               uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	Name             string         `gorm:"size:128;not null;index:uk_deploy_package_name_version_deleted,unique" json:"name"`
	Version          string         `gorm:"size:64;not null;index:uk_deploy_package_name_version_deleted,unique" json:"version"`
	Description      string         `gorm:"size:512" json:"description"`
	InstallCommand   string         `gorm:"type:text" json:"installCommand"`
	UninstallCommand string         `gorm:"type:text" json:"uninstallCommand"`
	ExecutionMode    string         `gorm:"size:32;default:fixed;index" json:"executionMode"`
	TemplateCode     string         `gorm:"size:64;index" json:"templateCode"`
	TemplateConfig   datatypes.JSON `gorm:"type:json" json:"templateConfig"`
	SourceObjectKey  string         `gorm:"size:255" json:"sourceObjectKey"`
	SourceFileName   string         `gorm:"size:255" json:"sourceFileName"`
	SourceURL        string         `gorm:"size:512" json:"sourceUrl"`
	Status           string         `gorm:"size:32;default:enabled;index" json:"status"`
	CreatedAt        time.Time      `json:"createdAt"`
	UpdatedAt        time.Time      `json:"updatedAt"`
	CreatedBy        string         `gorm:"size:64" json:"createdBy"`
	UpdatedBy        string         `gorm:"size:64" json:"updatedBy"`
	DeletedAt        gorm.DeletedAt `gorm:"index:uk_deploy_package_name_version_deleted,unique" json:"-"`
}

func (DeployPackage) TableName() string { return "biz_deploy_package" }

type DeployTask struct {
	ID                      uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	Name                    string         `gorm:"size:128;not null" json:"name"`
	TemplateID              uint64         `gorm:"index" json:"templateId"`
	TemplateName            string         `gorm:"size:128" json:"templateName"`
	TemplateVersion         string         `gorm:"size:64" json:"templateVersion"`
	PackageID               uint64         `gorm:"not null;index" json:"packageId"`
	PackageName             string         `gorm:"size:128" json:"packageName"`
	PackageVersion          string         `gorm:"size:64" json:"packageVersion"`
	BusinessScopeID         uint64         `gorm:"column:business_scope_id;index" json:"businessScopeId"`
	BusinessScopeName       string         `gorm:"column:business_scope_name;size:128" json:"businessScopeName"`
	ServiceID               uint64         `gorm:"column:service_id;index" json:"serviceId"`
	ServiceInstanceID       uint64         `gorm:"column:service_instance_id;index" json:"serviceInstanceId"`
	ServiceName             string         `gorm:"column:service_name;size:255" json:"serviceName"`
	ServiceInstanceName     string         `gorm:"column:service_instance_name;size:255" json:"serviceInstanceName"`
	Action                  string         `gorm:"size:32;default:install;index" json:"action"`
	TargetType              string         `gorm:"size:32;not null;index" json:"targetType"`
	TargetIDs               datatypes.JSON `gorm:"type:json" json:"targetIds"`
	ExecutorType            string         `gorm:"size:32;default:manual;index" json:"executorType"`
	ExecutionMode           string         `gorm:"size:32;default:fixed;index" json:"executionMode"`
	TemplateParams          datatypes.JSON `gorm:"type:json" json:"templateParams"`
	ExecutionSnapshot       datatypes.JSON `gorm:"type:json" json:"executionSnapshot"`
	TargetSnapshot          datatypes.JSON `gorm:"type:json" json:"targetSnapshot"`
	StartRequestKey         string         `gorm:"size:128" json:"startRequestKey"`
	CredentialRefID         uint64         `gorm:"column:credential_ref_id;index" json:"credentialRefId"`
	CredentialRefVersion    uint64         `gorm:"column:credential_ref_version" json:"credentialRefVersion"`
	SSHHostFingerprint      string         `gorm:"column:ssh_host_fingerprint;size:255" json:"-"`
	ExecutionTimeoutSeconds int            `gorm:"column:execution_timeout_seconds;default:1800" json:"executionTimeoutSeconds"`
	Status                  string         `gorm:"size:32;default:draft;index" json:"status"`
	Remark                  string         `gorm:"size:512" json:"remark"`
	ExternalTaskID          string         `gorm:"size:128" json:"externalTaskId"`
	StartedAt               *time.Time     `json:"startedAt"`
	FinishedAt              *time.Time     `json:"finishedAt"`
	CreatedAt               time.Time      `json:"createdAt"`
	UpdatedAt               time.Time      `json:"updatedAt"`
	CreatedBy               string         `gorm:"size:64" json:"createdBy"`
	UpdatedBy               string         `gorm:"size:64" json:"updatedBy"`
	DeletedAt               gorm.DeletedAt `gorm:"index" json:"-"`
}

func (DeployTask) TableName() string { return "biz_deploy_task" }

type DeployTemplate struct {
	ID              uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	Name            string         `gorm:"size:128;not null;index:uk_deploy_template_name_version_deleted,unique" json:"name"`
	Version         string         `gorm:"size:64;not null;index:uk_deploy_template_name_version_deleted,unique" json:"version"`
	Description     string         `gorm:"size:512" json:"description"`
	Category        string         `gorm:"size:64;index" json:"category"`
	ExecutionMode   string         `gorm:"size:32;default:fixed;index" json:"executionMode"`
	DefaultAction   string         `gorm:"size:32;default:install;index" json:"defaultAction"`
	PackageID       uint64         `gorm:"index" json:"packageId"`
	PackageName     string         `gorm:"size:128" json:"packageName"`
	PackageVersion  string         `gorm:"size:64" json:"packageVersion"`
	TemplateCode    string         `gorm:"size:64;index" json:"templateCode"`
	TemplateConfig  datatypes.JSON `gorm:"type:json" json:"templateConfig"`
	ParameterSchema datatypes.JSON `gorm:"type:json" json:"parameterSchema"`
	Status          string         `gorm:"size:32;default:enabled;index" json:"status"`
	CreatedAt       time.Time      `json:"createdAt"`
	UpdatedAt       time.Time      `json:"updatedAt"`
	CreatedBy       string         `gorm:"size:64" json:"createdBy"`
	UpdatedBy       string         `gorm:"size:64" json:"updatedBy"`
	DeletedAt       gorm.DeletedAt `gorm:"index:uk_deploy_template_name_version_deleted,unique" json:"-"`
}

func (DeployTemplate) TableName() string { return "biz_deploy_template" }

type DeployTemplateStep struct {
	ID             uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	TemplateID     uint64         `gorm:"not null;index" json:"templateId"`
	StepCode       string         `gorm:"size:64;not null" json:"stepCode"`
	StepName       string         `gorm:"size:128;not null" json:"stepName"`
	StepType       string         `gorm:"size:32;not null;index" json:"stepType"`
	Action         string         `gorm:"size:32;default:install;index" json:"action"`
	PackageID      uint64         `gorm:"index" json:"packageId"`
	PackageName    string         `gorm:"size:128" json:"packageName"`
	PackageVersion string         `gorm:"size:64" json:"packageVersion"`
	TemplateCode   string         `gorm:"size:64;index" json:"templateCode"`
	TemplateParams datatypes.JSON `gorm:"type:json" json:"templateParams"`
	StepConfig     datatypes.JSON `gorm:"type:json" json:"stepConfig"`
	Sort           int            `gorm:"default:1;index" json:"sort"`
	CreatedAt      time.Time      `json:"createdAt"`
	UpdatedAt      time.Time      `json:"updatedAt"`
}

func (DeployTemplateStep) TableName() string { return "biz_deploy_template_step" }

type DeployTaskHost struct {
	ID              uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	TaskID          uint64         `gorm:"not null;index;uniqueIndex:uk_deploy_task_host" json:"taskId"`
	HostID          uint64         `gorm:"not null;index;uniqueIndex:uk_deploy_task_host" json:"hostId"`
	Hostname        string         `gorm:"size:128" json:"hostname"`
	HostIP          string         `gorm:"size:45" json:"hostIp"`
	SSHPort         int            `gorm:"default:22" json:"sshPort"`
	OS              string         `gorm:"size:32" json:"os"`
	BusinessScopeID uint64         `gorm:"column:business_scope_id;index" json:"businessScopeId"`
	Status          string         `gorm:"size:32;default:pending;index" json:"status"`
	Stdout          string         `gorm:"type:text" json:"stdout"`
	Stderr          string         `gorm:"type:text" json:"stderr"`
	ErrorMessage    string         `gorm:"size:512" json:"errorMessage"`
	ExecutorID      string         `gorm:"size:128" json:"executorId"`
	ReportKey       string         `gorm:"size:128" json:"reportKey"`
	TraceSteps      datatypes.JSON `gorm:"type:json" json:"traceSteps"`
	StartedAt       *time.Time     `json:"startedAt"`
	FinishedAt      *time.Time     `json:"finishedAt"`
	ReportedAt      *time.Time     `json:"reportedAt"`
	ResolvedAt      *time.Time     `json:"resolvedAt"`
	CreatedAt       time.Time      `json:"createdAt"`
	UpdatedAt       time.Time      `json:"updatedAt"`
	UpdatedBy       string         `gorm:"size:64" json:"updatedBy"`
}

func (DeployTaskHost) TableName() string { return "biz_deploy_task_host" }

// DeployHostLease records the active execution lease for a deployment host.
//
//nolint:revive // Public model names retain the deploy domain prefix for API clarity.
type DeployHostLease struct {
	ID        uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	HostID    uint64    `gorm:"not null;uniqueIndex:uk_deploy_host_lease_host" json:"hostId"`
	TaskID    uint64    `gorm:"not null;index" json:"taskId"`
	Owner     string    `gorm:"size:128;not null" json:"owner"`
	ExpiresAt time.Time `gorm:"index" json:"expiresAt"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// TableName returns the deployment host lease table name.
func (DeployHostLease) TableName() string { return "biz_deploy_host_lease" }

// DeployTaskAttempt records one durable worker execution claim per host.
//
//nolint:revive // Public model name retains the deploy domain prefix for compatibility.
type DeployTaskAttempt struct {
	ID             uint64     `gorm:"primaryKey;autoIncrement" json:"id"`
	TaskID         uint64     `gorm:"not null;index" json:"taskId"`
	TaskHostID     uint64     `gorm:"not null;index" json:"taskHostId"`
	AttemptNo      int        `gorm:"not null" json:"attemptNo"`
	Status         string     `gorm:"size:32;not null;index" json:"status"`
	WorkerID       string     `gorm:"size:128" json:"workerId"`
	LeaseExpiresAt *time.Time `gorm:"index" json:"leaseExpiresAt"`
	StartedAt      *time.Time `json:"startedAt"`
	FinishedAt     *time.Time `json:"finishedAt"`
	ErrorMessage   string     `gorm:"size:512" json:"errorMessage"`
	CreatedAt      time.Time  `json:"createdAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
}

// TableName returns the deployment task attempt table name.
func (DeployTaskAttempt) TableName() string { return "biz_deploy_task_attempt" }

// DeployCredentialRef holds an encrypted SSH secret. The plaintext is never
// serialized and a task stores only this immutable reference id/version.
//
//nolint:revive // Public model name retains the deploy domain prefix for compatibility.
type DeployCredentialRef struct {
	ID              uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	Name            string         `gorm:"size:128;not null;uniqueIndex:uk_deploy_credential_name_deleted" json:"name"`
	Username        string         `gorm:"size:128;not null" json:"username"`
	AuthMode        string         `gorm:"size:32;not null" json:"authMode"`
	SecretEncrypted string         `gorm:"type:text;not null" json:"-"`
	Version         uint64         `gorm:"not null;default:1" json:"version"`
	Status          string         `gorm:"size:32;not null;default:active" json:"status"`
	CreatedAt       time.Time      `json:"createdAt"`
	UpdatedAt       time.Time      `json:"updatedAt"`
	DeletedAt       gorm.DeletedAt `gorm:"index:uk_deploy_credential_name_deleted" json:"-"`
}

// TableName returns the deployment credential reference table name.
func (DeployCredentialRef) TableName() string { return "biz_deploy_credential_ref" }

type cmdbHostSnapshot struct {
	ID                uint64         `gorm:"column:id"`
	Hostname          string         `gorm:"column:hostname"`
	IP                string         `gorm:"column:ip"`
	SSHPort           int            `gorm:"column:ssh_port"`
	OS                string         `gorm:"column:os"`
	Status            string         `gorm:"column:status"`
	BusinessScopeID   uint64         `gorm:"column:business_scope_id"`
	BusinessScopeName string         `gorm:"column:business_scope_name"`
	LabelValues       datatypes.JSON `gorm:"column:label_values"`
	DeptID            uint64         `gorm:"column:dept_id"`
}

type cmdbGroupSnapshot struct {
	ID         uint64         `gorm:"column:id"`
	ParentID   uint64         `gorm:"column:parent_id"`
	Name       string         `gorm:"column:name"`
	Conditions datatypes.JSON `gorm:"column:conditions"`
}

// deployPackageSnapshot is the immutable execution-definition view of a package
// captured at task creation. It never reads the live package row at run time.
type deployPackageSnapshot struct {
	ID               uint64         `json:"id"`
	Name             string         `json:"name"`
	Version          string         `json:"version"`
	InstallCommand   string         `json:"installCommand"`
	UninstallCommand string         `json:"uninstallCommand"`
	ExecutionMode    string         `json:"executionMode"`
	TemplateCode     string         `json:"templateCode"`
	TemplateConfig   map[string]any `json:"templateConfig"`
	SourceObjectKey  string         `json:"sourceObjectKey"`
	SourceFileName   string         `json:"sourceFileName"`
	SourceURL        string         `json:"sourceUrl"`
}

// deployStepSnapshot is a frozen template step with its fully resolved package.
type deployStepSnapshot struct {
	StepCode       string                `json:"stepCode"`
	StepName       string                `json:"stepName"`
	StepType       string                `json:"stepType"`
	Action         string                `json:"action"`
	Package        deployPackageSnapshot `json:"package"`
	TemplateParams map[string]any        `json:"templateParams"`
	StepConfig     map[string]any        `json:"stepConfig"`
}

// deployExecutionSnapshot is the immutable execution intent persisted on a task.
type deployExecutionSnapshot struct {
	TemplateID      uint64               `json:"templateId"`
	TemplateName    string               `json:"templateName"`
	TemplateVersion string               `json:"templateVersion"`
	Steps           []deployStepSnapshot `json:"steps"`
}

// deployTargetHostSnapshot is a frozen host connection fact captured at Start.
type deployTargetHostSnapshot struct {
	HostID            uint64 `json:"hostId"`
	Hostname          string `json:"hostname"`
	IP                string `json:"ip"`
	SSHPort           int    `json:"sshPort"`
	OS                string `json:"os"`
	BusinessScopeID   uint64 `json:"businessScopeId"`
	BusinessScopeName string `json:"businessScopeName"`
	DeptID            uint64 `json:"deptId"`
	Status            string `json:"status"`
}

// deployTargetSnapshot is the immutable target intent frozen at Start.
type deployTargetSnapshot struct {
	TargetType string                     `json:"targetType"`
	ResolvedAt string                     `json:"resolvedAt"`
	Hosts      []deployTargetHostSnapshot `json:"hosts"`
}
