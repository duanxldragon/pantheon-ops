package label

type CreateLabelSchemaRequest struct {
	Key         string   `json:"key" binding:"required"`
	Name        string   `json:"name" binding:"required"`
	Category    string   `json:"category"`
	ValueMode   string   `json:"valueMode"`
	DictCode    string   `json:"dictCode"`
	Options     []string `json:"options"`
	Required    bool     `json:"required"`
	Status      string   `json:"status"`
	Description string   `json:"description"`
}

type UpdateLabelSchemaRequest struct {
	Name        *string   `json:"name"`
	Category    *string   `json:"category"`
	ValueMode   *string   `json:"valueMode"`
	DictCode    *string   `json:"dictCode"`
	Options     *[]string `json:"options"`
	Required    *bool     `json:"required"`
	Status      *string   `json:"status"`
	Description *string   `json:"description"`
}

type LabelSchemaQuery struct {
	Keyword  string `form:"keyword" json:"keyword"`
	Status   string `form:"status" json:"status"`
	Category string `form:"category" json:"category"`
	Page     int    `form:"page" json:"page"`
	PageSize int    `form:"pageSize" json:"pageSize"`
}

type LabelSchemaResponse struct {
	ID          uint64   `json:"id"`
	Key         string   `json:"key"`
	Name        string   `json:"name"`
	Category    string   `json:"category"`
	ValueMode   string   `json:"valueMode"`
	DictCode    string   `json:"dictCode"`
	Options     []string `json:"options"`
	Required    bool     `json:"required"`
	Status      string   `json:"status"`
	Description string   `json:"description"`
	CreatedAt   string   `json:"createdAt"`
	UpdatedAt   string   `json:"updatedAt"`
}

type LabelSchemaListResponse struct {
	Items    []LabelSchemaResponse `json:"items"`
	Total    int64                 `json:"total"`
	Page     int                   `json:"page"`
	PageSize int                   `json:"pageSize"`
}
