package system

type OperationLogResp struct {
	TenantID        uint64 `json:"tenantId"`
	ID              uint64 `json:"id"`
	RequestID       string `json:"requestId"`
	Title           string `json:"title"`
	BusinessType    int    `json:"businessType"`
	Method          string `json:"method"`
	OperName        string `json:"operName"`
	OperURL         string `json:"operUrl"`
	OperIP          string `json:"operIp"`
	SourceDomain    string `json:"sourceDomain"`
	SourcePage      string `json:"sourcePage"`
	OperParam       string `json:"operParam"`
	JsonResult      string `json:"jsonResult"`
	Status          int    `json:"status"`
	FailureCategory string `json:"failureCategory"`
	ErrorMsg        string `json:"errorMsg"`
	OperTime        string `json:"operTime"`
	CostTime        int64  `json:"costTime"`
}

type OperationLogPageResp struct {
	Items []OperationLogResp `json:"items"`
	Total int64              `json:"total"`
	// SuccessCount/FailedCount aggregate the whole filtered set (all pages),
	// so the governance bar shows global numbers instead of page-local ones.
	SuccessCount int64 `json:"successCount"`
	FailedCount  int64 `json:"failedCount"`
	Page         int   `json:"page"`
	PageSize     int   `json:"pageSize"`
}

type OperationLogQuery struct {
	// TenantIDFilter narrows results to one tenant (platform-ops cross-tenant
	// query, contract §7). Never bound from a tenant-subject request — the
	// handler enforces the authorization rule below.
	TenantIDFilter  uint64 `json:"tenantIdFilter,omitempty"`
	Keyword         string `form:"keyword" json:"keyword"`
	Title           string `form:"title" json:"title"`
	RequestID       string `form:"requestId" json:"requestId"`
	OperName        string `form:"operName" json:"operName"`
	Status          *int   `form:"status" json:"status"`
	BusinessType    *int   `form:"businessType" json:"businessType"`
	SourceDomain    string `form:"sourceDomain" json:"sourceDomain"`
	SourcePage      string `form:"sourcePage" json:"sourcePage"`
	FailureCategory string `form:"failureCategory" json:"failureCategory"`
	StartedAt       string `form:"startedAt" json:"startedAt"`
	EndedAt         string `form:"endedAt" json:"endedAt"`
	Page            int    `form:"page" json:"page"`
	PageSize        int    `form:"pageSize" json:"pageSize"`
	SortField       string `form:"sortField" json:"sortField"`
	SortOrder       string `form:"sortOrder" json:"sortOrder"`
}

type OperationLogCleanupReq struct {
	RetentionDays int    `json:"retentionDays"`
	StartedAt     string `json:"startedAt"`
	EndedAt       string `json:"endedAt"`
}

type OperationLogBatchDeleteReq struct {
	IDs []uint64 `json:"ids"`
}
