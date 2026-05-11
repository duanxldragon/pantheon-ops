package config

type SettingResp struct {
	ID           uint64 `json:"id"`
	SettingKey   string `json:"settingKey"`
	SettingValue string `json:"settingValue"`
	DefaultValue string `json:"defaultValue"`
	ValueType    string `json:"valueType"`
	GroupKey     string `json:"groupKey"`
	Module       string `json:"module"`
	IsPublic     int    `json:"isPublic"`
	IsEncrypted  int    `json:"isEncrypted"`
	HasValue     int    `json:"hasValue"`
	Remark       string `json:"remark"`
	CreatedAt    string `json:"createdAt"`
	UpdatedAt    string `json:"updatedAt"`
}

type SettingListQuery struct {
	GroupKey string `form:"groupKey"`
	Module   string `form:"module"`
}

type SettingGroupResp struct {
	GroupKey string        `json:"groupKey"`
	Items    []SettingResp `json:"items"`
}

type SettingUpdateItemReq struct {
	SettingKey   string `json:"settingKey" binding:"required"`
	SettingValue string `json:"settingValue"`
}

type SettingGroupUpdateReq struct {
	Items []SettingUpdateItemReq `json:"items" binding:"required,min=1,dive"`
}

type PublicSettingResp struct {
	Settings map[string]string `json:"settings"`
}

type SettingCacheRefreshReq struct {
	GroupKeys []string `json:"groupKeys"`
}

type SettingCacheRefreshResp struct {
	RefreshedGroups []string `json:"refreshedGroups"`
	ClearedAll      int      `json:"clearedAll"`
}

type SettingAuditQuery struct {
	GroupKey   string `form:"groupKey"`
	SettingKey string `form:"settingKey"`
	OperName   string `form:"operName"`
	Page       int    `form:"page"`
	PageSize   int    `form:"pageSize"`
}

type SettingAuditChangeResp struct {
	SettingKey  string `json:"settingKey"`
	OldValue    string `json:"oldValue"`
	NewValue    string `json:"newValue"`
	IsEncrypted int    `json:"isEncrypted"`
}

type SettingAuditResp struct {
	ID       uint64                   `json:"id"`
	GroupKey string                   `json:"groupKey"`
	OperName string                   `json:"operName"`
	OperIP   string                   `json:"operIp"`
	Status   int                      `json:"status"`
	ErrorMsg string                   `json:"errorMsg"`
	OperTime string                   `json:"operTime"`
	CostTime int64                    `json:"costTime"`
	Changes  []SettingAuditChangeResp `json:"changes"`
}

type SettingAuditPageResp struct {
	Items    []SettingAuditResp `json:"items"`
	Total    int64              `json:"total"`
	Page     int                `json:"page"`
	PageSize int                `json:"pageSize"`
}

type SettingOverviewIssueResp struct {
	SettingKey string `json:"settingKey"`
	GroupKey   string `json:"groupKey"`
	Severity   string `json:"severity"`
	ReasonKey  string `json:"reasonKey"`
}

type SettingOverviewResp struct {
	TotalSettingCount     int                        `json:"totalSettingCount"`
	PublicSettingCount    int                        `json:"publicSettingCount"`
	EncryptedSettingCount int                        `json:"encryptedSettingCount"`
	RequiredMissingCount  int                        `json:"requiredMissingCount"`
	RiskCount             int                        `json:"riskCount"`
	StorageDriver         string                     `json:"storageDriver"`
	DefaultLanguage       string                     `json:"defaultLanguage"`
	DefaultTheme          string                     `json:"defaultTheme"`
	Issues                []SettingOverviewIssueResp `json:"issues"`
}
