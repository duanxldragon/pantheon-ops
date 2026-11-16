package bizscope

type EnvironmentListResp struct {
	ID              uint64 `json:"id"`
	BusinessScopeID uint64 `json:"businessScopeId"`
	Code            string `json:"code"`
	Name            string `json:"name"`
	EnvType         string `json:"envType"`
	Status          string `json:"status"`
	Remark          string `json:"remark"`
	CreatedAt       string `json:"createdAt"`
}

type EnvironmentListPageResp struct {
	Items    []EnvironmentListResp `json:"items"`
	Total    int64                 `json:"total"`
	Page     int                   `json:"page"`
	PageSize int                   `json:"pageSize"`
}

type EnvironmentListQuery struct {
	Code     string `form:"code" json:"code"`
	Name     string `form:"name" json:"name"`
	Status   string `form:"status" json:"status"`
	Page     int    `form:"page" json:"page"`
	PageSize int    `form:"pageSize" json:"pageSize"`
}

type CreateEnvironmentRequest struct {
	Code    string `json:"code" binding:"required"`
	Name    string `json:"name" binding:"required"`
	EnvType string `json:"envType" binding:"required"`
	Status  string `json:"status" binding:"required"`
	Remark  string `json:"remark"`
}

type UpdateEnvironmentRequest struct {
	Code    *string `json:"code"`
	Name    *string `json:"name"`
	EnvType *string `json:"envType"`
	Status  *string `json:"status"`
	Remark  *string `json:"remark"`
}

type EnvironmentOptionItem struct {
	Label string `json:"label"`
	Value string `json:"value"`
	Code  string `json:"code"`
	Name  string `json:"name"`
}
