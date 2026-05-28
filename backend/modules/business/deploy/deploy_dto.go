package deploy

import "time"

type PackageQuery struct {
	Page          int    `form:"page"`
	PageSize      int    `form:"pageSize"`
	Keyword       string `form:"keyword"`
	Status        string `form:"status"`
	ExecutionMode string `form:"executionMode"`
	TemplateCode  string `form:"templateCode"`
}

type CreatePackageRequest struct {
	Name             string `json:"name" binding:"required"`
	Version          string `json:"version" binding:"required"`
	Description      string `json:"description"`
	InstallCommand   string `json:"installCommand"`
	UninstallCommand string `json:"uninstallCommand"`
	ExecutionMode    string `json:"executionMode"`
	TemplateCode     string `json:"templateCode"`
	TemplateConfig   map[string]any `json:"templateConfig"`
	SourceObjectKey  string `json:"sourceObjectKey"`
	SourceFileName   string `json:"sourceFileName"`
	SourceURL        string `json:"sourceUrl"`
	Status           string `json:"status"`
}

type UpdatePackageRequest struct {
	Name             *string `json:"name"`
	Version          *string `json:"version"`
	Description      *string `json:"description"`
	InstallCommand   *string `json:"installCommand"`
	UninstallCommand *string `json:"uninstallCommand"`
	ExecutionMode    *string `json:"executionMode"`
	TemplateCode     *string `json:"templateCode"`
	TemplateConfig   *map[string]any `json:"templateConfig"`
	SourceObjectKey  *string `json:"sourceObjectKey"`
	SourceFileName   *string `json:"sourceFileName"`
	SourceURL        *string `json:"sourceUrl"`
	Status           *string `json:"status"`
}

type PackageResponse struct {
	ID               uint64    `json:"id"`
	Name             string    `json:"name"`
	Version          string    `json:"version"`
	Description      string    `json:"description"`
	InstallCommand   string    `json:"installCommand"`
	UninstallCommand string    `json:"uninstallCommand"`
	ExecutionMode    string    `json:"executionMode"`
	TemplateCode     string    `json:"templateCode"`
	TemplateConfig   map[string]any `json:"templateConfig"`
	SourceObjectKey  string    `json:"sourceObjectKey"`
	SourceFileName   string    `json:"sourceFileName"`
	SourceURL        string    `json:"sourceUrl"`
	Status           string    `json:"status"`
	LatestDeployedAt *time.Time `json:"latestDeployedAt,omitempty"`
	LatestTaskID     uint64    `json:"latestTaskId"`
	LatestTaskName   string    `json:"latestTaskName"`
	LatestTaskStatus string    `json:"latestTaskStatus"`
	LatestHostCount  int       `json:"latestHostCount"`
	LatestSuccessCount int     `json:"latestSuccessCount"`
	CreatedAt        time.Time `json:"createdAt"`
	UpdatedAt        time.Time `json:"updatedAt"`
	CreatedBy        string    `json:"createdBy"`
	UpdatedBy        string    `json:"updatedBy"`
}

type PackageListResponse struct {
	Items    []PackageResponse `json:"items"`
	Total    int64             `json:"total"`
	Page     int               `json:"page"`
	PageSize int               `json:"pageSize"`
}

type TemplateQuery struct {
	Page          int    `form:"page"`
	PageSize      int    `form:"pageSize"`
	Keyword       string `form:"keyword"`
	Status        string `form:"status"`
	ExecutionMode string `form:"executionMode"`
	Category      string `form:"category"`
}

type TemplateStepPayload struct {
	StepCode       string         `json:"stepCode"`
	StepName       string         `json:"stepName"`
	StepType       string         `json:"stepType"`
	Action         string         `json:"action"`
	PackageID      uint64         `json:"packageId"`
	PackageName    string         `json:"packageName"`
	PackageVersion string         `json:"packageVersion"`
	TemplateCode   string         `json:"templateCode"`
	TemplateParams map[string]any `json:"templateParams"`
	StepConfig     map[string]any `json:"stepConfig"`
	Sort           int            `json:"sort"`
}

type CreateTemplateRequest struct {
	Name            string                `json:"name" binding:"required"`
	Version         string                `json:"version" binding:"required"`
	Description     string                `json:"description"`
	Category        string                `json:"category"`
	ExecutionMode   string                `json:"executionMode"`
	DefaultAction   string                `json:"defaultAction"`
	PackageID       uint64                `json:"packageId"`
	TemplateCode    string                `json:"templateCode"`
	TemplateConfig  map[string]any        `json:"templateConfig"`
	ParameterSchema map[string]any        `json:"parameterSchema"`
	Status          string                `json:"status"`
	Steps           []TemplateStepPayload `json:"steps"`
}

type UpdateTemplateRequest struct {
	Name            *string                `json:"name"`
	Version         *string                `json:"version"`
	Description     *string                `json:"description"`
	Category        *string                `json:"category"`
	ExecutionMode   *string                `json:"executionMode"`
	DefaultAction   *string                `json:"defaultAction"`
	PackageID       *uint64                `json:"packageId"`
	TemplateCode    *string                `json:"templateCode"`
	TemplateConfig  *map[string]any        `json:"templateConfig"`
	ParameterSchema *map[string]any        `json:"parameterSchema"`
	Status          *string                `json:"status"`
	Steps           *[]TemplateStepPayload `json:"steps"`
}

type TemplateStepResponse struct {
	ID             uint64         `json:"id"`
	TemplateID     uint64         `json:"templateId"`
	StepCode       string         `json:"stepCode"`
	StepName       string         `json:"stepName"`
	StepType       string         `json:"stepType"`
	Action         string         `json:"action"`
	PackageID      uint64         `json:"packageId"`
	PackageName    string         `json:"packageName"`
	PackageVersion string         `json:"packageVersion"`
	TemplateCode   string         `json:"templateCode"`
	TemplateParams map[string]any `json:"templateParams"`
	StepConfig     map[string]any `json:"stepConfig"`
	Sort           int            `json:"sort"`
	CreatedAt      time.Time      `json:"createdAt"`
	UpdatedAt      time.Time      `json:"updatedAt"`
}

type TemplateResponse struct {
	ID              uint64                 `json:"id"`
	Name            string                 `json:"name"`
	Version         string                 `json:"version"`
	Description     string                 `json:"description"`
	Category        string                 `json:"category"`
	ExecutionMode   string                 `json:"executionMode"`
	DefaultAction   string                 `json:"defaultAction"`
	PackageID       uint64                 `json:"packageId"`
	PackageName     string                 `json:"packageName"`
	PackageVersion  string                 `json:"packageVersion"`
	TemplateCode    string                 `json:"templateCode"`
	TemplateConfig  map[string]any         `json:"templateConfig"`
	ParameterSchema map[string]any         `json:"parameterSchema"`
	Status          string                 `json:"status"`
	StepCount       int                    `json:"stepCount"`
	Steps           []TemplateStepResponse `json:"steps"`
	CreatedAt       time.Time              `json:"createdAt"`
	UpdatedAt       time.Time              `json:"updatedAt"`
	CreatedBy       string                 `json:"createdBy"`
	UpdatedBy       string                 `json:"updatedBy"`
}

type TemplateListResponse struct {
	Items    []TemplateResponse `json:"items"`
	Total    int64              `json:"total"`
	Page     int                `json:"page"`
	PageSize int                `json:"pageSize"`
}

type TaskQuery struct {
	Page         int    `form:"page"`
	PageSize     int    `form:"pageSize"`
	Keyword      string `form:"keyword"`
	Status       string `form:"status"`
	ExecutorType string `form:"executorType"`
}

type CreateTaskRequest struct {
	Name            string   `json:"name" binding:"required"`
	TemplateID      uint64   `json:"templateId"`
	PackageID       uint64   `json:"packageId"`
	BusinessScopeID uint64   `json:"businessScopeId"`
	Action          string   `json:"action"`
	TargetType      string   `json:"targetType" binding:"required"`
	TargetIDs       []uint64 `json:"targetIds" binding:"required"`
	ExecutorType    string   `json:"executorType"`
	TemplateParams  map[string]any `json:"templateParams"`
	Remark          string   `json:"remark"`
}

type UpdateTaskRequest struct {
	Name            *string  `json:"name"`
	TemplateID      *uint64  `json:"templateId"`
	PackageID       *uint64  `json:"packageId"`
	BusinessScopeID *uint64  `json:"businessScopeId"`
	Action          *string  `json:"action"`
	TargetType      *string  `json:"targetType"`
	TargetIDs       []uint64 `json:"targetIds"`
	ExecutorType    *string  `json:"executorType"`
	TemplateParams  *map[string]any `json:"templateParams"`
	Remark          *string  `json:"remark"`
}

type StartTaskRequest struct {
	SSHUser         string `json:"sshUser"`
	SSHPassword     string `json:"sshPassword"`
	SSHPrivateKey   string `json:"sshPrivateKey"`
	HostFingerprint string `json:"hostFingerprint"`
	AuthMode        string `json:"authMode"`
}

type MarkHostResultRequest struct {
	Status       string `json:"status" binding:"required"`
	Stdout       string `json:"stdout"`
	Stderr       string `json:"stderr"`
	ErrorMessage string `json:"errorMessage"`
	ExecutorID   string `json:"executorId"`
}

type TaskResponse struct {
	ID                uint64             `json:"id"`
	Name              string             `json:"name"`
	TemplateID        uint64             `json:"templateId"`
	TemplateName      string             `json:"templateName"`
	TemplateVersion   string             `json:"templateVersion"`
	PackageID         uint64             `json:"packageId"`
	PackageName       string             `json:"packageName"`
	PackageVersion    string             `json:"packageVersion"`
	BusinessScopeID   uint64             `json:"businessScopeId"`
	BusinessScopeName string             `json:"businessScopeName"`
	Action            string             `json:"action"`
	TargetType        string             `json:"targetType"`
	TargetIDs         []uint64           `json:"targetIds"`
	ExecutorType      string             `json:"executorType"`
	ExecutionMode     string             `json:"executionMode"`
	TemplateParams    map[string]any     `json:"templateParams"`
	Status            string             `json:"status"`
	Remark            string             `json:"remark"`
	ExternalTaskID    string             `json:"externalTaskId"`
	StartedAt         *time.Time         `json:"startedAt"`
	FinishedAt        *time.Time         `json:"finishedAt"`
	HostCount         int                `json:"hostCount"`
	SuccessCount      int                `json:"successCount"`
	FailedCount       int                `json:"failedCount"`
	RunningCount      int                `json:"runningCount"`
	SkippedCount      int                `json:"skippedCount"`
	DurationSeconds   int64              `json:"durationSeconds"`
	CreatedAt         time.Time          `json:"createdAt"`
	UpdatedAt         time.Time          `json:"updatedAt"`
	CreatedBy         string             `json:"createdBy"`
	UpdatedBy         string             `json:"updatedBy"`
	Hosts             []TaskHostResponse `json:"hosts"`
}

type TaskHostResponse struct {
	ID           uint64     `json:"id"`
	TaskID       uint64     `json:"taskId"`
	HostID       uint64     `json:"hostId"`
	Hostname     string     `json:"hostname"`
	HostIP       string     `json:"hostIp"`
	OS           string     `json:"os"`
	Status       string     `json:"status"`
	Stdout       string     `json:"stdout"`
	Stderr       string     `json:"stderr"`
	ErrorMessage string     `json:"errorMessage"`
	ExecutorID   string     `json:"executorId"`
	TraceSteps   []map[string]any `json:"traceSteps"`
	StartedAt    *time.Time `json:"startedAt"`
	FinishedAt   *time.Time `json:"finishedAt"`
	ReportedAt   *time.Time `json:"reportedAt"`
	DurationSeconds int64   `json:"durationSeconds"`
	UpdatedAt    time.Time  `json:"updatedAt"`
	UpdatedBy    string     `json:"updatedBy"`
}

type TaskListResponse struct {
	Items    []TaskResponse `json:"items"`
	Total    int64          `json:"total"`
	Page     int            `json:"page"`
	PageSize int            `json:"pageSize"`
}
