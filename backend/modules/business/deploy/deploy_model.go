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
	ID                uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	Name              string         `gorm:"size:128;not null" json:"name"`
	TemplateID        uint64         `gorm:"index" json:"templateId"`
	TemplateName      string         `gorm:"size:128" json:"templateName"`
	TemplateVersion   string         `gorm:"size:64" json:"templateVersion"`
	PackageID         uint64         `gorm:"not null;index" json:"packageId"`
	PackageName       string         `gorm:"size:128" json:"packageName"`
	PackageVersion    string         `gorm:"size:64" json:"packageVersion"`
	BusinessScopeID   uint64         `gorm:"column:business_scope_id;index" json:"businessScopeId"`
	BusinessScopeName string         `gorm:"column:business_scope_name;size:128" json:"businessScopeName"`
	Action            string         `gorm:"size:32;default:install;index" json:"action"`
	TargetType        string         `gorm:"size:32;not null;index" json:"targetType"`
	TargetIDs         datatypes.JSON `gorm:"type:json" json:"targetIds"`
	ExecutorType      string         `gorm:"size:32;default:manual;index" json:"executorType"`
	ExecutionMode     string         `gorm:"size:32;default:fixed;index" json:"executionMode"`
	TemplateParams    datatypes.JSON `gorm:"type:json" json:"templateParams"`
	Status            string         `gorm:"size:32;default:draft;index" json:"status"`
	Remark            string         `gorm:"size:512" json:"remark"`
	ExternalTaskID    string         `gorm:"size:128" json:"externalTaskId"`
	StartedAt         *time.Time     `json:"startedAt"`
	FinishedAt        *time.Time     `json:"finishedAt"`
	CreatedAt         time.Time      `json:"createdAt"`
	UpdatedAt         time.Time      `json:"updatedAt"`
	CreatedBy         string         `gorm:"size:64" json:"createdBy"`
	UpdatedBy         string         `gorm:"size:64" json:"updatedBy"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`
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
	ID           uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	TaskID       uint64         `gorm:"not null;index" json:"taskId"`
	HostID       uint64         `gorm:"not null;index" json:"hostId"`
	Hostname     string         `gorm:"size:128" json:"hostname"`
	HostIP       string         `gorm:"size:45" json:"hostIp"`
	OS           string         `gorm:"size:32" json:"os"`
	Status       string         `gorm:"size:32;default:pending;index" json:"status"`
	Stdout       string         `gorm:"type:text" json:"stdout"`
	Stderr       string         `gorm:"type:text" json:"stderr"`
	ErrorMessage string         `gorm:"size:512" json:"errorMessage"`
	ExecutorID   string         `gorm:"size:128" json:"executorId"`
	TraceSteps   datatypes.JSON `gorm:"type:json" json:"traceSteps"`
	StartedAt    *time.Time     `json:"startedAt"`
	FinishedAt   *time.Time     `json:"finishedAt"`
	ReportedAt   *time.Time     `json:"reportedAt"`
	CreatedAt    time.Time      `json:"createdAt"`
	UpdatedAt    time.Time      `json:"updatedAt"`
	UpdatedBy    string         `gorm:"size:64" json:"updatedBy"`
}

func (DeployTaskHost) TableName() string { return "biz_deploy_task_host" }

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
