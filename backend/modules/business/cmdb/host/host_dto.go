package host

type LabelEntry struct {
	Key string `json:"key"`
	Val string `json:"val"`
}

type ComponentEntry struct {
	Name         string `json:"name"`
	Version      string `json:"version"`
	DeployedAt   string `json:"deployedAt,omitempty"`
	DeployTaskID uint64 `json:"deployTaskId,omitempty"`
	DeployTaskName string `json:"deployTaskName,omitempty"`
	ExecutorType string `json:"executorType,omitempty"`
}

type MatchedGroupEntry struct {
	ID       uint64 `json:"id"`
	ParentID uint64 `json:"parentId"`
	Name     string `json:"name"`
	FullPath string `json:"fullPath"`
}

type HostListQuery struct {
	Page            int    `form:"page" json:"page"`
	PageSize        int    `form:"pageSize" json:"pageSize"`
	Keyword         string `form:"keyword" json:"keyword"`
	Status          string `form:"status" json:"status"`
	OS              string `form:"os" json:"os"`
	BusinessScopeID uint64 `form:"businessScopeId" json:"businessScopeId"`
	DeptID          uint64 `form:"deptId" json:"deptId"`
}

type CreateHostRequest struct {
	Hostname        string       `json:"hostname" binding:"required"`
	IP              string       `json:"ip" binding:"required"`
	SSHPort         int          `json:"sshPort"`
	OS              string       `json:"os" binding:"required"`
	OSVersion       string       `json:"osVersion"`
	CPUCores        int          `json:"cpuCores"`
	MemoryGB        float64      `json:"memoryGb"`
	DiskGB          float64      `json:"diskGb"`
	Labels          []LabelEntry `json:"labels"`
	BusinessScopeID uint64       `json:"businessScopeId"`
	DeptID          uint64       `json:"deptId"`
	Owner           string       `json:"owner"`
	Remark          string       `json:"remark"`
}

type UpdateHostRequest struct {
	Hostname        *string       `json:"hostname"`
	IP              *string       `json:"ip"`
	SSHPort         *int          `json:"sshPort"`
	OS              *string       `json:"os"`
	OSVersion       *string       `json:"osVersion"`
	CPUCores        *int          `json:"cpuCores"`
	MemoryGB        *float64      `json:"memoryGb"`
	DiskGB          *float64      `json:"diskGb"`
	Labels          *[]LabelEntry `json:"labels"`
	BusinessScopeID *uint64       `json:"businessScopeId"`
	DeptID          *uint64       `json:"deptId"`
	Owner           *string       `json:"owner"`
	Remark          *string       `json:"remark"`
}

type CollectRequest struct {
	SSHUser         string `json:"sshUser" binding:"required"`
	SSHPassword     string `json:"sshPassword"`
	SSHPrivateKey   string `json:"sshPrivateKey"`
	HostFingerprint string `json:"hostFingerprint" binding:"required"`
	AuthMode        string `json:"authMode" binding:"required"`
}

type UpdateStatusRequest struct {
	Status string `json:"status" binding:"required"`
}

type HostResponse struct {
	ID                  uint64           `json:"id"`
	Hostname            string           `json:"hostname"`
	IP                  string           `json:"ip"`
	SSHPort             int              `json:"sshPort"`
	OS                  string           `json:"os"`
	OSVersion           string           `json:"osVersion"`
	CPUCores            int              `json:"cpuCores"`
	MemoryGB            float64          `json:"memoryGb"`
	DiskGB              float64          `json:"diskGb"`
	LabelValues         []LabelEntry     `json:"labelValues"`
	InstalledComponents []ComponentEntry `json:"installedComponents"`
	MatchedGroups       []MatchedGroupEntry `json:"matchedGroups"`
	MatchedGroupCount   int              `json:"matchedGroupCount"`
	Status              string           `json:"status"`
	BusinessScopeID     uint64           `json:"businessScopeId"`
	BusinessScopeCode   string           `json:"businessScopeCode"`
	BusinessScopeName   string           `json:"businessScopeName"`
	DeptID              uint64           `json:"deptId"`
	Owner               string           `json:"owner"`
	Remark              string           `json:"remark"`
	CreatedAt           string           `json:"createdAt"`
	UpdatedAt           string           `json:"updatedAt"`
	CreatedBy           string           `json:"createdBy"`
	UpdatedBy           string           `json:"updatedBy"`
}

type HostListResponse struct {
	Items    []HostResponse `json:"items"`
	Total    int64          `json:"total"`
	Page     int            `json:"page"`
	PageSize int            `json:"pageSize"`
}
