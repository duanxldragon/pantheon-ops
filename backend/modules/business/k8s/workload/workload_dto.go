package workload

type WorkloadListQuery struct {
	ClusterID uint64 `form:"clusterId" json:"clusterId" binding:"required"`
	Namespace string `form:"namespace" json:"namespace"`
	Kind      string `form:"kind" json:"kind"`
}

type WorkloadItem struct {
	Kind          string   `json:"kind"`
	Name          string   `json:"name"`
	Namespace     string   `json:"namespace"`
	Replicas      int32    `json:"replicas"`
	ReadyReplicas int32    `json:"readyReplicas"`
	Images        []string `json:"images"`
	Status        string   `json:"status"`
	Age           string   `json:"age"`
}

type WorkloadListResponse struct {
	Items []WorkloadItem `json:"items"`
	Total int            `json:"total"`
}

type PodItem struct {
	Name      string `json:"name"`
	Status    string `json:"status"`
	NodeName  string `json:"nodeName"`
	Restarts  int32  `json:"restarts"`
	CreatedAt string `json:"createdAt"`
}

type PodListResponse struct {
	Items []PodItem `json:"items"`
	Total int       `json:"total"`
}

type ScaleRequest struct {
	Replicas int32 `json:"replicas" binding:"required,min=0"`
}
