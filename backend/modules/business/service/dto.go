package service

import "time"

type ApplicationQuery struct {
	Page     int    `form:"page"`
	PageSize int    `form:"pageSize"`
	Keyword  string `form:"keyword"`
	Status   string `form:"status"`
}

type ReconcileInstanceRequest struct {
	MaxAgeSeconds int `json:"maxAgeSeconds"`
}

type CreateApplicationRequest struct {
	Code            string `json:"code" binding:"required"`
	Name            string `json:"name" binding:"required"`
	BusinessScopeID uint64 `json:"businessScopeId" binding:"required"`
	Status          string `json:"status"`
	Owner           string `json:"owner"`
	Remark          string `json:"remark"`
}

type UpdateApplicationRequest struct {
	Name   *string `json:"name"`
	Status *string `json:"status"`
	Owner  *string `json:"owner"`
	Remark *string `json:"remark"`
}

type ApplicationResponse struct {
	ID                uint64    `json:"id"`
	Code              string    `json:"code"`
	Name              string    `json:"name"`
	BusinessScopeID   uint64    `json:"businessScopeId"`
	BusinessScopeName string    `json:"businessScopeName"`
	DeptID            uint64    `json:"deptId"`
	Status            string    `json:"status"`
	Owner             string    `json:"owner"`
	Remark            string    `json:"remark"`
	ServiceCount      int64     `json:"serviceCount"`
	CreatedAt         time.Time `json:"createdAt"`
	UpdatedAt         time.Time `json:"updatedAt"`
}

type ApplicationListResponse struct {
	Items    []ApplicationResponse `json:"items"`
	Total    int64                 `json:"total"`
	Page     int                   `json:"page"`
	PageSize int                   `json:"pageSize"`
}

type ServiceQuery struct {
	Page          int    `form:"page"`
	PageSize      int    `form:"pageSize"`
	Keyword       string `form:"keyword"`
	ApplicationID uint64 `form:"applicationId"`
	Status        string `form:"status"`
}

type CreateServiceRequest struct {
	ApplicationID uint64 `json:"applicationId" binding:"required"`
	Code          string `json:"code" binding:"required"`
	Name          string `json:"name" binding:"required"`
	RuntimeType   string `json:"runtimeType" binding:"required"`
	Description   string `json:"description"`
	Status        string `json:"status"`
}

type UpdateServiceRequest struct {
	Name        *string `json:"name"`
	RuntimeType *string `json:"runtimeType"`
	Description *string `json:"description"`
	Status      *string `json:"status"`
}

type ServiceResponse struct {
	ID              uint64    `json:"id"`
	ApplicationID   uint64    `json:"applicationId"`
	ApplicationCode string    `json:"applicationCode"`
	ApplicationName string    `json:"applicationName"`
	BusinessScopeID uint64    `json:"businessScopeId"`
	Code            string    `json:"code"`
	Name            string    `json:"name"`
	RuntimeType     string    `json:"runtimeType"`
	Description     string    `json:"description"`
	Status          string    `json:"status"`
	InstanceCount   int64     `json:"instanceCount"`
	CreatedAt       time.Time `json:"createdAt"`
	UpdatedAt       time.Time `json:"updatedAt"`
}

type ServiceListResponse struct {
	Items    []ServiceResponse `json:"items"`
	Total    int64             `json:"total"`
	Page     int               `json:"page"`
	PageSize int               `json:"pageSize"`
}

type InstanceQuery struct {
	Page       int    `form:"page"`
	PageSize   int    `form:"pageSize"`
	ServiceID  uint64 `form:"serviceId"`
	TargetType string `form:"targetType"`
	Status     string `form:"status"`
}

type CreateInstanceRequest struct {
	ServiceID      uint64 `json:"serviceId" binding:"required"`
	Environment    string `json:"environment" binding:"required"`
	TargetType     string `json:"targetType" binding:"required"`
	HostID         uint64 `json:"hostId"`
	K8sClusterID   uint64 `json:"k8sClusterId"`
	Namespace      string `json:"namespace"`
	WorkloadKind   string `json:"workloadKind"`
	WorkloadName   string `json:"workloadName"`
	DesiredVersion string `json:"desiredVersion"`
	Status         string `json:"status"`
}

type UpdateInstanceRequest struct {
	DesiredVersion *string `json:"desiredVersion"`
	CurrentVersion *string `json:"currentVersion"`
	Status         *string `json:"status"`
}

type InstanceStateTransitionRequest struct {
	InstanceID               uint64 `json:"instanceId"`
	Action                   string `json:"action" binding:"required"`
	ExpectedLifecycleVersion uint64 `json:"expectedLifecycleVersion"`
	DesiredState             string `json:"desiredState"`
	ObservedState            string `json:"observedState"`
	HealthState              string `json:"healthState"`
	DesiredVersion           string `json:"desiredVersion"`
	CurrentVersion           string `json:"currentVersion"`
	RollbackVersion          string `json:"rollbackVersion"`
	HealthMessage            string `json:"healthMessage"`
	HealthRevision           string `json:"healthRevision"`
	CorrelationID            string `json:"correlationId"`
}

type InstanceResponse struct {
	ID               uint64     `json:"id"`
	ServiceID        uint64     `json:"serviceId"`
	ServiceCode      string     `json:"serviceCode"`
	ServiceName      string     `json:"serviceName"`
	ApplicationID    uint64     `json:"applicationId"`
	ApplicationCode  string     `json:"applicationCode"`
	BusinessScopeID  uint64     `json:"businessScopeId"`
	DeptID           uint64     `json:"deptId"`
	Environment      string     `json:"environment"`
	TargetType       string     `json:"targetType"`
	HostID           uint64     `json:"hostId"`
	K8sClusterID     uint64     `json:"k8sClusterId"`
	Namespace        string     `json:"namespace"`
	WorkloadKind     string     `json:"workloadKind"`
	WorkloadName     string     `json:"workloadName"`
	DesiredVersion   string     `json:"desiredVersion"`
	CurrentVersion   string     `json:"currentVersion"`
	RollbackVersion  string     `json:"rollbackVersion"`
	DesiredState     string     `json:"desiredState"`
	ObservedState    string     `json:"observedState"`
	HealthState      string     `json:"healthState"`
	HealthObservedAt *time.Time `json:"healthObservedAt"`
	HealthMessage    string     `json:"healthMessage"`
	HealthRevision   string     `json:"healthRevision"`
	LastTransitionID string     `json:"lastTransitionId"`
	LastTransitionAt *time.Time `json:"lastTransitionAt"`
	LifecycleVersion uint64     `json:"lifecycleVersion"`
	Status           string     `json:"status"`
	CreatedAt        time.Time  `json:"createdAt"`
	UpdatedAt        time.Time  `json:"updatedAt"`
}

type InstanceListResponse struct {
	Items    []InstanceResponse `json:"items"`
	Total    int64              `json:"total"`
	Page     int                `json:"page"`
	PageSize int                `json:"pageSize"`
}

type OptionItem struct {
	ID    uint64 `json:"id"`
	Code  string `json:"code"`
	Name  string `json:"name"`
	Label string `json:"label"`
	Value uint64 `json:"value"`
}
