package generator

type GeneratorDatasourceResp struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	Driver          string `json:"driver"`
	Host            string `json:"host,omitempty"`
	Port            int    `json:"port,omitempty"`
	DatabaseName    string `json:"databaseName"`
	Username        string `json:"username,omitempty"`
	Status          int    `json:"status"`
	Remark          string `json:"remark,omitempty"`
	ReadonlyScope   string `json:"readonlyScope,omitempty"`
	LastCheckedAt   string `json:"lastCheckedAt,omitempty"`
	LastCheckStatus string `json:"lastCheckStatus,omitempty"`
	LastCheckError  string `json:"lastCheckError,omitempty"`
	IsCurrent       bool   `json:"isCurrent"`
}

type UpsertGeneratorDatasourceReq struct {
	Name         string `json:"name"`
	Driver       string `json:"driver"`
	Host         string `json:"host"`
	Port         int    `json:"port"`
	DatabaseName string `json:"databaseName"`
	Username     string `json:"username"`
	Password     string `json:"password"`
	Status       int    `json:"status"`
	Remark       string `json:"remark"`
}

type TableOptionResp struct {
	TableName string `json:"tableName"`
	Comment   string `json:"comment"`
	Engine    string `json:"engine"`
	Rows      int64  `json:"rows"`
}

type EnumOptionResp struct {
	Value   string `json:"value"`
	Label   string `json:"label"`
	LabelEn string `json:"labelEn,omitempty"`
}

type FieldValidationResp struct {
	Required bool     `json:"required,omitempty"`
	Unique   bool     `json:"unique,omitempty"`
	Enum     []string `json:"enum,omitempty"`
}

type ModuleFieldResp struct {
	Name          string               `json:"name"`
	Type          string               `json:"type"`
	Label         string               `json:"label"`
	LabelEn       string               `json:"labelEn,omitempty"`
	Required      bool                 `json:"required"`
	Searchable    bool                 `json:"searchable"`
	Sortable      bool                 `json:"sortable"`
	VisibleInList bool                 `json:"visibleInList"`
	VisibleInForm bool                 `json:"visibleInForm"`
	Placeholder   string               `json:"placeholder,omitempty"`
	PlaceholderEn string               `json:"placeholderEn,omitempty"`
	HelpText      string               `json:"helpText,omitempty"`
	HelpTextEn    string               `json:"helpTextEn,omitempty"`
	DictCode      string               `json:"dictCode,omitempty"`
	EnumOptions   []EnumOptionResp     `json:"enumOptions,omitempty"`
	Validation    *FieldValidationResp `json:"validation,omitempty"`
}

type TableSchemaPreviewResp struct {
	TableName      string            `json:"tableName"`
	TableComment   string            `json:"tableComment"`
	SuggestedName  string            `json:"suggestedName"`
	SuggestedScope string            `json:"suggestedScope"`
	SuggestedTitle string            `json:"suggestedTitle"`
	Fields         []ModuleFieldResp `json:"fields"`
}
