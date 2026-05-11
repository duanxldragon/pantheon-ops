package deploy

import "time"

type PackageQuery struct {
	Page     int    `form:"page"`
	PageSize int    `form:"pageSize"`
	Keyword  string `form:"keyword"`
	Status   string `form:"status"`
}

type CreatePackageRequest struct {
	Name             string `json:"name" binding:"required"`
	Version          string `json:"version" binding:"required"`
	Description      string `json:"description"`
	InstallCommand   string `json:"installCommand"`
	UninstallCommand string `json:"uninstallCommand"`
	Status           string `json:"status"`
}

type UpdatePackageRequest struct {
	Name             *string `json:"name"`
	Version          *string `json:"version"`
	Description      *string `json:"description"`
	InstallCommand   *string `json:"installCommand"`
	UninstallCommand *string `json:"uninstallCommand"`
	Status           *string `json:"status"`
}

type PackageResponse struct {
	ID               uint64    `json:"id"`
	Name             string    `json:"name"`
	Version          string    `json:"version"`
	Description      string    `json:"description"`
	InstallCommand   string    `json:"installCommand"`
	UninstallCommand string    `json:"uninstallCommand"`
	Status           string    `json:"status"`
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

type TaskQuery struct {
	Page         int    `form:"page"`
	PageSize     int    `form:"pageSize"`
	Keyword      string `form:"keyword"`
	Status       string `form:"status"`
	ExecutorType string `form:"executorType"`
}

type CreateTaskRequest struct {
	Name         string   `json:"name" binding:"required"`
	PackageID    uint64   `json:"packageId" binding:"required"`
	TargetType   string   `json:"targetType" binding:"required"`
	TargetIDs    []uint64 `json:"targetIds" binding:"required"`
	ExecutorType string   `json:"executorType"`
	Remark       string   `json:"remark"`
}

type UpdateTaskRequest struct {
	Name         *string  `json:"name"`
	PackageID    *uint64  `json:"packageId"`
	TargetType   *string  `json:"targetType"`
	TargetIDs    []uint64 `json:"targetIds"`
	ExecutorType *string  `json:"executorType"`
	Remark       *string  `json:"remark"`
}

type MarkHostResultRequest struct {
	Status       string `json:"status" binding:"required"`
	Stdout       string `json:"stdout"`
	Stderr       string `json:"stderr"`
	ErrorMessage string `json:"errorMessage"`
	ExecutorID   string `json:"executorId"`
}

type TaskResponse struct {
	ID             uint64             `json:"id"`
	Name           string             `json:"name"`
	PackageID      uint64             `json:"packageId"`
	PackageName    string             `json:"packageName"`
	PackageVersion string             `json:"packageVersion"`
	TargetType     string             `json:"targetType"`
	TargetIDs      []uint64           `json:"targetIds"`
	ExecutorType   string             `json:"executorType"`
	Status         string             `json:"status"`
	Remark         string             `json:"remark"`
	ExternalTaskID string             `json:"externalTaskId"`
	StartedAt      *time.Time         `json:"startedAt"`
	FinishedAt     *time.Time         `json:"finishedAt"`
	CreatedAt      time.Time          `json:"createdAt"`
	UpdatedAt      time.Time          `json:"updatedAt"`
	CreatedBy      string             `json:"createdBy"`
	UpdatedBy      string             `json:"updatedBy"`
	Hosts          []TaskHostResponse `json:"hosts"`
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
	StartedAt    *time.Time `json:"startedAt"`
	FinishedAt   *time.Time `json:"finishedAt"`
	ReportedAt   *time.Time `json:"reportedAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
	UpdatedBy    string     `json:"updatedBy"`
}

type TaskListResponse struct {
	Items    []TaskResponse `json:"items"`
	Total    int64          `json:"total"`
	Page     int            `json:"page"`
	PageSize int            `json:"pageSize"`
}
