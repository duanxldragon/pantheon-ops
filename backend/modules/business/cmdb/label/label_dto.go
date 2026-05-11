package label

type CreateLabelSchemaRequest struct {
	Key         string   `json:"key" binding:"required"`
	Name        string   `json:"name" binding:"required"`
	ValueMode   string   `json:"valueMode"`
	DictCode    string   `json:"dictCode"`
	Options     []string `json:"options"`
	Required    bool     `json:"required"`
	Status      string   `json:"status"`
	Description string   `json:"description"`
}

type UpdateLabelSchemaRequest struct {
	Name        *string   `json:"name"`
	ValueMode   *string   `json:"valueMode"`
	DictCode    *string   `json:"dictCode"`
	Options     *[]string `json:"options"`
	Required    *bool     `json:"required"`
	Status      *string   `json:"status"`
	Description *string   `json:"description"`
}

type LabelSchemaQuery struct {
	Keyword string `form:"keyword" json:"keyword"`
	Status  string `form:"status" json:"status"`
}

type LabelSchemaResponse struct {
	ID          uint64   `json:"id"`
	Key         string   `json:"key"`
	Name        string   `json:"name"`
	ValueMode   string   `json:"valueMode"`
	DictCode    string   `json:"dictCode"`
	Options     []string `json:"options"`
	Required    bool     `json:"required"`
	Status      string   `json:"status"`
	Description string   `json:"description"`
	CreatedAt   string   `json:"createdAt"`
	UpdatedAt   string   `json:"updatedAt"`
}
