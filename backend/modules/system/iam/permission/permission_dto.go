package iam

type PermissionPolicyResp struct {
	ID      uint64 `json:"id"`
	PType   string `json:"ptype"`
	RoleKey string `json:"roleKey"`
	Path    string `json:"path"`
	Method  string `json:"method"`
}

type PermissionPolicyPageResp struct {
	Items    []PermissionPolicyResp `json:"items"`
	Total    int64                  `json:"total"`
	Page     int                    `json:"page"`
	PageSize int                    `json:"pageSize"`
}

type PermissionPolicyQuery struct {
	RoleKey  string `form:"roleKey" json:"roleKey"`
	Path     string `form:"path" json:"path"`
	Method   string `form:"method" json:"method"`
	Page     int    `form:"page" json:"page"`
	PageSize int    `form:"pageSize" json:"pageSize"`
}

type PermissionPolicyCreateReq struct {
	RoleKey string `json:"roleKey" binding:"required"`
	Path    string `json:"path" binding:"required"`
	Method  string `json:"method" binding:"required"`
}

type PermissionPolicyUpdateReq struct {
	RoleKey string `json:"roleKey" binding:"required"`
	Path    string `json:"path" binding:"required"`
	Method  string `json:"method" binding:"required"`
}

type PermissionWorkbenchQuery struct {
	RoleKey   string `form:"roleKey"`
	Status    *int   `form:"status"`
	Integrity string `form:"integrity"`
	Coverage  string `form:"coverage"`
}

type PermissionDataScopeQuery struct {
	RoleKey string `form:"roleKey" json:"roleKey"`
	Status  *int   `form:"status" json:"status"`
}

type PermissionDataScopePolicyResp struct {
	ID           uint64   `json:"id"`
	RoleName     string   `json:"roleName"`
	RoleKey      string   `json:"roleKey"`
	Status       int      `json:"status"`
	Mode         string   `json:"mode"`
	DeptIDs      []uint64 `json:"deptIds"`
	PolicyExists bool     `json:"policyExists"`
}

type PermissionDataScopePolicyListResp struct {
	Items []PermissionDataScopePolicyResp `json:"items"`
	Total int                             `json:"total"`
}

type PermissionDataScopePolicyUpdateReq struct {
	Mode    string   `json:"mode" binding:"required"`
	DeptIDs []uint64 `json:"deptIds"`
}

type PermissionWorkbenchOverviewResp struct {
	RoleCount                        int `json:"roleCount"`
	EnabledRoleCount                 int `json:"enabledRoleCount"`
	NavigationAssignmentCount        int `json:"navigationAssignmentCount"`
	PagePermissionAssignmentCount    int `json:"pagePermissionAssignmentCount"`
	ActionPermissionAssignmentCount  int `json:"actionPermissionAssignmentCount"`
	APIActionCount                   int `json:"apiActionCount"`
	UnknownPermissionAssignmentCount int `json:"unknownPermissionAssignmentCount"`
	PageGapRoleCount                 int `json:"pageGapRoleCount"`
	APIGapRoleCount                  int `json:"apiGapRoleCount"`
}

type PermissionWorkbenchMenuResp struct {
	ID       uint64 `json:"id"`
	TitleKey string `json:"titleKey"`
	Path     string `json:"path"`
	Module   string `json:"module"`
}

type PermissionWorkbenchPermissionResp struct {
	Key      string `json:"key"`
	TitleKey string `json:"titleKey"`
	Path     string `json:"path"`
	Module   string `json:"module"`
	Kind     string `json:"kind"`
}

type PermissionWorkbenchAPIPolicyResp struct {
	ID     uint64 `json:"id"`
	Path   string `json:"path"`
	Method string `json:"method"`
}

type PermissionWorkbenchRemediateReq struct {
	RoleKey string `json:"roleKey" binding:"required"`
}

type PermissionWorkbenchRemediateResp struct {
	RoleKey         string                             `json:"roleKey"`
	CreatedCount    int                                `json:"createdCount"`
	SkippedCount    int                                `json:"skippedCount"`
	CreatedPolicies []PermissionWorkbenchAPIPolicyResp `json:"createdPolicies"`
}

type PermissionWorkbenchRemediationQuery struct {
	RoleKey string `form:"roleKey" json:"roleKey"`
	Limit   int    `form:"limit" json:"limit"`
}

type PermissionWorkbenchRemediationResp struct {
	ID           uint64 `json:"id"`
	RoleKey      string `json:"roleKey"`
	IssueType    string `json:"issueType"`
	IssueKey     string `json:"issueKey"`
	BeforeState  string `json:"beforeState"`
	AfterState   string `json:"afterState"`
	Action       string `json:"action"`
	CreatedCount int    `json:"createdCount"`
	SkippedCount int    `json:"skippedCount"`
	CreatedAt    string `json:"createdAt"`
}

type PermissionWorkbenchRoleResp struct {
	ID                     uint64                              `json:"id"`
	RoleName               string                              `json:"roleName"`
	RoleKey                string                              `json:"roleKey"`
	Status                 int                                 `json:"status"`
	MenuCount              int                                 `json:"menuCount"`
	PagePermissionCount    int                                 `json:"pagePermissionCount"`
	ActionPermissionCount  int                                 `json:"actionPermissionCount"`
	APIPolicyCount         int                                 `json:"apiPolicyCount"`
	RequiredAPIPolicyCount int                                 `json:"requiredApiPolicyCount"`
	MissingAPIPolicyCount  int                                 `json:"missingApiPolicyCount"`
	UnknownPermissionCount int                                 `json:"unknownPermissionCount"`
	HasPageGap             bool                                `json:"hasPageGap"`
	HasAPIGap              bool                                `json:"hasApiGap"`
	Menus                  []PermissionWorkbenchMenuResp       `json:"menus"`
	PagePermissions        []PermissionWorkbenchPermissionResp `json:"pagePermissions"`
	ActionPermissions      []PermissionWorkbenchPermissionResp `json:"actionPermissions"`
	UnknownPermissions     []PermissionWorkbenchPermissionResp `json:"unknownPermissions"`
	APIPolicies            []PermissionWorkbenchAPIPolicyResp  `json:"apiPolicies"`
	MissingAPIPolicies     []PermissionWorkbenchAPIPolicyResp  `json:"missingApiPolicies"`
}

type PermissionWorkbenchResp struct {
	Overview PermissionWorkbenchOverviewResp `json:"overview"`
	Roles    []PermissionWorkbenchRoleResp   `json:"roles"`
}
