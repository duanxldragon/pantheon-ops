package release

type CreateReleaseRequest struct {
	Name                string `json:"name" binding:"required"`
	ClusterID           uint64 `json:"clusterId" binding:"required"`
	ServiceID           uint64 `json:"serviceId"`
	ServiceInstanceID   uint64 `json:"serviceInstanceId"`
	ServiceName         string `json:"serviceName"`
	ServiceInstanceName string `json:"serviceInstanceName"`
	Namespace           string `json:"namespace" binding:"required"`
	WorkloadType        string `json:"workloadType" binding:"required"`
	WorkloadName        string `json:"workloadName" binding:"required"`
	ContainerName       string `json:"containerName"`
	Image               string `json:"image" binding:"required"`
	Strategy            string `json:"strategy"`
	IdempotencyKey      string `json:"idempotencyKey"`
}

type RollbackReleaseRequest struct {
	IdempotencyKey string `json:"idempotencyKey"`
}

type ReleaseListQuery struct {
	Page         int    `form:"page" json:"page"`
	PageSize     int    `form:"pageSize" json:"pageSize"`
	ClusterID    uint64 `form:"clusterId" json:"clusterId"`
	Namespace    string `form:"namespace" json:"namespace"`
	WorkloadType string `form:"workloadType" json:"workloadType"`
	WorkloadName string `form:"workloadName" json:"workloadName"`
	Status       string `form:"status" json:"status"`
}

type ReleaseResponse struct {
	ID                  uint64 `json:"id"`
	Name                string `json:"name"`
	ClusterID           uint64 `json:"clusterId"`
	ServiceID           uint64 `json:"serviceId"`
	ServiceInstanceID   uint64 `json:"serviceInstanceId"`
	ServiceName         string `json:"serviceName"`
	ServiceInstanceName string `json:"serviceInstanceName"`
	Namespace           string `json:"namespace"`
	WorkloadType        string `json:"workloadType"`
	WorkloadName        string `json:"workloadName"`
	ContainerName       string `json:"containerName"`
	ImageBefore         string `json:"imageBefore"`
	ImageAfter          string `json:"imageAfter"`
	Strategy            string `json:"strategy"`
	Status              string `json:"status"`
	IdempotencyKey      string `json:"idempotencyKey"`
	PreviousReleaseID   uint64 `json:"previousReleaseId"`
	Attempt             int    `json:"attempt"`
	TargetGeneration    int64  `json:"targetGeneration"`
	TargetRevision      string `json:"targetRevision"`
	ObservedGeneration  int64  `json:"observedGeneration"`
	ObservedRevision    string `json:"observedRevision"`
	DesiredReplicas     int32  `json:"desiredReplicas"`
	UpdatedReplicas     int32  `json:"updatedReplicas"`
	AvailableReplicas   int32  `json:"availableReplicas"`
	ReadyReplicas       int32  `json:"readyReplicas"`
	ConditionSummary    string `json:"conditionSummary"`
	ErrorMessage        string `json:"errorMessage"`
	StartedAt           string `json:"startedAt"`
	FinishedAt          string `json:"finishedAt"`
	LastReconciledAt    string `json:"lastReconciledAt"`
	CreatedBy           string `json:"createdBy"`
	CreatedAt           string `json:"createdAt"`
}

type ReleaseListResponse struct {
	Items    []ReleaseResponse `json:"items"`
	Total    int64             `json:"total"`
	Page     int               `json:"page"`
	PageSize int               `json:"pageSize"`
}
