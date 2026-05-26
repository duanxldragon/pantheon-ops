package host

type LabelEntry struct {
	Key string `json:"key"`
	Val string `json:"val"`
}

type ComponentEntry struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type HostListQuery struct {
	Page     int    `form:"page" json:"page"`
	PageSize int    `form:"pageSize" json:"pageSize"`
	Keyword  string `form:"keyword" json:"keyword"`
	Status   string `form:"status" json:"status"`
	OS       string `form:"os" json:"os"`
	DeptID   uint64 `form:"deptId" json:"deptId"`
}

type CreateHostRequest struct {
	Hostname  string       `json:"hostname" binding:"required"`
	IP        string       `json:"ip" binding:"required"`
	SSHPort   int          `json:"sshPort"`
	OS        string       `json:"os" binding:"required"`
	OSVersion string       `json:"osVersion"`
	CPUCores  int          `json:"cpuCores"`
	MemoryGB  float64      `json:"memoryGb"`
	DiskGB    float64      `json:"diskGb"`
	Labels    []LabelEntry `json:"labels"`
	DeptID    uint64       `json:"deptId"`
	Owner     string       `json:"owner"`
	Remark    string       `json:"remark"`
}

type UpdateHostRequest struct {
	Hostname  *string       `json:"hostname"`
	IP        *string       `json:"ip"`
	SSHPort   *int          `json:"sshPort"`
	OS        *string       `json:"os"`
	OSVersion *string       `json:"osVersion"`
	CPUCores  *int          `json:"cpuCores"`
	MemoryGB  *float64      `json:"memoryGb"`
	DiskGB    *float64      `json:"diskGb"`
	Labels    *[]LabelEntry `json:"labels"`
	DeptID    *uint64       `json:"deptId"`
	Owner     *string       `json:"owner"`
	Remark    *string       `json:"remark"`
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
	Status              string           `json:"status"`
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
