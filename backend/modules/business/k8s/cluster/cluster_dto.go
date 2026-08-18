package cluster

type ClusterListQuery struct {
	Page            int    `form:"page" json:"page"`
	PageSize        int    `form:"pageSize" json:"pageSize"`
	Keyword         string `form:"keyword" json:"keyword"`
	Environment     string `form:"environment" json:"environment"`
	Status          string `form:"status" json:"status"`
	BusinessScopeID uint64 `form:"businessScopeId" json:"businessScopeId"`
}

type CreateClusterRequest struct {
	Code            string `json:"code" binding:"required"`
	Name            string `json:"name" binding:"required"`
	Environment     string `json:"environment" binding:"required"`
	BusinessScopeID uint64 `json:"businessScopeId"`
	Kubeconfig      string `json:"kubeconfig" binding:"required"`
	Remark          string `json:"remark"`
}

type UpdateClusterRequest struct {
	Name            *string `json:"name"`
	Environment     *string `json:"environment"`
	BusinessScopeID *uint64 `json:"businessScopeId"`
	Kubeconfig      *string `json:"kubeconfig"`
	Remark          *string `json:"remark"`
}

type ClusterResponse struct {
	ID                uint64  `json:"id"`
	Code              string  `json:"code"`
	Name              string  `json:"name"`
	Environment       string  `json:"environment"`
	BusinessScopeID   uint64  `json:"businessScopeId"`
	BusinessScopeName string  `json:"businessScopeName"`
	DeptID            uint64  `json:"deptId"`
	APIServer         string  `json:"apiServer"`
	Version           string  `json:"version"`
	Status            string  `json:"status"`
	TotalNodes        int     `json:"totalNodes"`
	ReadyNodes        int     `json:"readyNodes"`
	TotalPods         int     `json:"totalPods"`
	RunningPods       int     `json:"runningPods"`
	CPUCapacity       float64 `json:"cpuCapacity"`
	CPUAllocatable    float64 `json:"cpuAllocatable"`
	MemoryCapacity    float64 `json:"memoryCapacity"`
	MemoryAllocatable float64 `json:"memoryAllocatable"`
	LastSyncedAt      string  `json:"lastSyncedAt"`
	Remark            string  `json:"remark"`
	CreatedAt         string  `json:"createdAt"`
	UpdatedAt         string  `json:"updatedAt"`
	CreatedBy         string  `json:"createdBy"`
	UpdatedBy         string  `json:"updatedBy"`
}

type ClusterListResponse struct {
	Items    []ClusterResponse `json:"items"`
	Total    int64             `json:"total"`
	Page     int               `json:"page"`
	PageSize int               `json:"pageSize"`
}

type NodeListResponse struct {
	Items []NodeSnapshot `json:"items"`
	Total int            `json:"total"`
}
