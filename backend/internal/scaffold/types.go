package scaffold

type GeneratedFile struct {
	Path     string `json:"path"`
	Content  string `json:"content"`
	Language string `json:"language"`
}

type ModuleField struct {
	Name          string           `json:"name"`
	Type          string           `json:"type"`
	Label         string           `json:"label"`
	LabelEn       string           `json:"labelEn,omitempty"`
	Required      bool             `json:"required"`
	Searchable    bool             `json:"searchable"`
	Sortable      bool             `json:"sortable"`
	VisibleInList bool             `json:"visibleInList"`
	VisibleInForm bool             `json:"visibleInForm"`
	DefaultValue  string           `json:"defaultValue,omitempty"`
	Validation    *FieldValidation `json:"validation,omitempty"`
	Placeholder   string           `json:"placeholder,omitempty"`
	PlaceholderEn string           `json:"placeholderEn,omitempty"`
	HelpText      string           `json:"helpText,omitempty"`
	HelpTextEn    string           `json:"helpTextEn,omitempty"`
	TemplateKey   string           `json:"templateKey,omitempty"`
	DictCode      string           `json:"dictCode,omitempty"`
	EnumOptions   []EnumOption     `json:"enumOptions,omitempty"`
}

type EnumOption struct {
	Value   string `json:"value"`
	Label   string `json:"label"`
	LabelEn string `json:"labelEn,omitempty"`
	Color   string `json:"color,omitempty"`
}

type FieldValidation struct {
	Required  bool     `json:"required,omitempty"`
	MinLength int      `json:"minLength,omitempty"`
	MaxLength int      `json:"maxLength,omitempty"`
	Min       float64  `json:"min,omitempty"`
	Max       float64  `json:"max,omitempty"`
	Pattern   string   `json:"pattern,omitempty"`
	Enum      []string `json:"enum,omitempty"`
	Unique    bool     `json:"unique,omitempty"`
}

type MenuSeedConfig struct {
	Key            string `json:"key"`
	ParentKey      string `json:"parentKey,omitempty"`
	TitleKey       string `json:"titleKey"`
	Path           string `json:"path,omitempty"`
	Component      string `json:"component,omitempty"`
	PagePermission string `json:"pagePermission,omitempty"`
	Perms          string `json:"perms,omitempty"`
	Type           string `json:"type"`
	Icon           string `json:"icon,omitempty"`
	RouteName      string `json:"routeName,omitempty"`
	Module         string `json:"module,omitempty"`
	Sort           int    `json:"sort,omitempty"`
	IsCache        bool   `json:"isCache,omitempty"`
	IsExternal     bool   `json:"isExternal,omitempty"`
	ActiveMenu     string `json:"activeMenu,omitempty"`
}

type PermissionConfig struct {
	Key    string `json:"key"`
	Name   string `json:"name"`
	Type   string `json:"type"`
	Module string `json:"module"`
}

type ModuleDependency struct {
	Module   string `json:"module"`
	Required bool   `json:"required,omitempty"`
	Reason   string `json:"reason,omitempty"`
}

type ModuleRelation struct {
	Name          string `json:"name"`
	Type          string `json:"type"`
	TargetModule  string `json:"targetModule"`
	LocalField    string `json:"localField"`
	TargetField   string `json:"targetField"`
	JunctionTable string `json:"junctionTable,omitempty"`
}

type ModuleSchema struct {
	Name               string             `json:"name"`
	TemplateVersion    string             `json:"templateVersion,omitempty"`
	DisplayName        string             `json:"displayName"`
	DisplayNameEn      string             `json:"displayNameEn,omitempty"`
	Description        string             `json:"description"`
	Scope              string             `json:"scope"`
	ParentMenu         string             `json:"parentMenu"`
	TemplateLevel      string             `json:"templateLevel"`
	PageActionTemplate string             `json:"pageActionTemplate,omitempty"`
	PageActions        []string           `json:"pageActions,omitempty"`
	Dependencies       []ModuleDependency `json:"dependencies,omitempty"`
	Relations          []ModuleRelation   `json:"relations,omitempty"`
	DataScopeMode      string             `json:"dataScopeMode,omitempty"`
	Metadata           struct {
		BusinessContext        string `json:"businessContext"`
		BusinessContextTitle   string `json:"businessContextTitle"`
		BusinessContextTitleEn string `json:"businessContextTitleEn"`
		TableRole              string `json:"tableRole"`
		PrimaryTable           string `json:"primaryTable"`
		RelationFromField      string `json:"relationFromField"`
		RelationToField        string `json:"relationToField"`
		BoundedContext         string `json:"boundedContext"`
		Owner                  string `json:"owner"`
		Summary                string `json:"summary"`
		SourceMode             string `json:"sourceMode"`
		SourceDatasourceID     string `json:"sourceDatasourceId"`
		SourceDatasourceName   string `json:"sourceDatasourceName"`
		SourceTable            string `json:"sourceTable"`
	} `json:"metadata"`
	Model struct {
		TableName string        `json:"tableName"`
		ModelName string        `json:"modelName"`
		Fields    []ModuleField `json:"fields"`
	} `json:"model"`
	Menus       []MenuSeedConfig   `json:"menus,omitempty"`
	Permissions []PermissionConfig `json:"permissions,omitempty"`
	I18n        struct {
		Namespace    string `json:"namespace"`
		Translations struct {
			Zh map[string]string `json:"zh"`
			En map[string]string `json:"en"`
		} `json:"translations"`
	} `json:"i18n"`
	EnableExport           bool `json:"enableExport,omitempty"`
	EnableImport           bool `json:"enableImport,omitempty"`
	EnableAudit            bool `json:"enableAudit,omitempty"`
	EnableDataScope        bool `json:"enableDataScope,omitempty"`
	IncludeDashboardWidget bool `json:"includeDashboardWidget,omitempty"`
}

type RegisterGeneratedModuleRequest struct {
	Schema    ModuleSchema    `json:"schema"`
	Files     []GeneratedFile `json:"files"`
	Overwrite bool            `json:"overwrite"`
}

type GeneratedModuleRef struct {
	Name  string
	Scope string
}
