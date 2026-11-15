package bizscope

type EnvironmentListQuery struct {
	BusinessScopeID uint64 `form:"businessScopeId" json:"businessScopeId"`
	EnvType         string `form:"envType" json:"envType"`
	Status          string `form:"status" json:"status"`
	Keyword         string `form:"keyword" json:"keyword"`
	Page            int    `form:"page" json:"page"`
	PageSize        int    `form:"pageSize" json:"pageSize"`
}

type EnvironmentResp struct {
	ID                uint64 `json:"id"`
	BusinessScopeID   uint64 `json:"businessScopeId"`
	BusinessScopeName string `json:"businessScopeName"`
	Code              string `json:"code"`
	Name              string `json:"name"`
	EnvType           string `json:"envType"`
	Status            string `json:"status"`
	Remark            string `json:"remark"`
	CreatedAt         string `json:"createdAt"`
	UpdatedAt         string `json:"updatedAt"`
}

type EnvironmentListPageResp struct {
	Items    []EnvironmentResp `json:"items"`
	Total    int64             `json:"total"`
	Page     int               `json:"page"`
	PageSize int               `json:"pageSize"`
}

type CreateEnvironmentRequest struct {
	BusinessScopeID uint64 `json:"businessScopeId"`
	Code            string `json:"code"`
	Name            string `json:"name"`
	EnvType         string `json:"envType"`
	Status          string `json:"status"`
	Remark          string `json:"remark"`
}

type UpdateEnvironmentRequest struct {
	BusinessScopeID *uint64 `json:"businessScopeId"`
	Code            *string `json:"code"`
	Name            *string `json:"name"`
	EnvType         *string `json:"envType"`
	Status          *string `json:"status"`
	Remark          *string `json:"remark"`
}
