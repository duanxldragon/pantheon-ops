package bizscope

type BizScopeListResp struct {
	ID          uint64 `json:"id"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	Owner       string `json:"owner"`
	Environment string `json:"environment"`
	Status      string `json:"status"`
	Remark      string `json:"remark"`
	CreatedAt   string `json:"createdAt"`
}

type BizScopeListPageResp struct {
	Items    []BizScopeListResp `json:"items"`
	Total    int64              `json:"total"`
	Page     int                `json:"page"`
	PageSize int                `json:"pageSize"`
}

type BizScopeDetailResp struct {
	ID          uint64 `json:"id"`
	Code        string `json:"code"`
	Name        string `json:"name"`
	Owner       string `json:"owner"`
	Environment string `json:"environment"`
	Status      string `json:"status"`
	Remark      string `json:"remark"`
	HostCount    int64  `json:"hostCount"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
}

type BizScopeListQuery struct {
	Code        string `form:"code" json:"code"`
	Name        string `form:"name" json:"name"`
	Owner       string `form:"owner" json:"owner"`
	Environment string `form:"environment" json:"environment"`
	Status      string `form:"status" json:"status"`
	Page        int    `form:"page" json:"page"`
	PageSize    int    `form:"pageSize" json:"pageSize"`
	SortField   string `form:"sortField" json:"sortField"`
	SortOrder   string `form:"sortOrder" json:"sortOrder"`
}

type CreateBizScopeRequest struct {
	Code        string `json:"code" binding:"required"`
	Name        string `json:"name" binding:"required"`
	Owner       string `json:"owner"`
	Environment string `json:"environment" binding:"required"`
	Status      string `json:"status" binding:"required"`
	Remark      string `json:"remark"`
}

type UpdateBizScopeRequest struct {
	Code        *string `json:"code"`
	Name        *string `json:"name"`
	Owner       *string `json:"owner"`
	Environment *string `json:"environment"`
	Status      *string `json:"status"`
	Remark      *string `json:"remark"`
}

type BizScopeOptionItem struct {
	Label string `json:"label"`
	Value uint64 `json:"value"`
	ID    uint64 `json:"id"`
	Name  string `json:"name"`
}

type BizScopeHostItem struct {
	ID                uint64 `json:"id"`
	Hostname          string `json:"hostname"`
	IP                string `json:"ip"`
	OS                string `json:"os"`
	Status            string `json:"status"`
	BusinessScopeID   uint64 `json:"businessScopeId"`
	BusinessScopeName string `json:"businessScopeName"`
}

type BizScopeHostListResp struct {
	Items []BizScopeHostItem `json:"items"`
	Total int64              `json:"total"`
}

type BindBizScopeHostsRequest struct {
	HostIDs []uint64 `json:"hostIds" binding:"required"`
}
