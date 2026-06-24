package iam

import "strings"

var staticRegisteredMenuComponentKeys = map[string]struct{}{
	"dashboard":                                   {},
	"auth/SecurityCenter":                         {},
	"auth/LoginLogList":                           {},
	"auth/SecurityEventList":                      {},
	"auth/SessionList":                            {},
	"system/profile/ProfileCenter":                {},
	"system/config/dict/DictPage":                 {},
	"system/i18n/I18nList":                        {},
	"system/org/dept/DeptList":                    {},
	"system/iam/menu/MenuList":                    {},
	"system/iam/permission/PermissionList":        {},
	"system/org/post/PostList":                    {},
	"system/iam/role/RoleList":                    {},
	"system/config/setting/SettingOverviewPage":   {},
	"system/config/setting/SettingGroupPage":      {},
	"system/iam/user/UserList":                    {},
	"system/iam/user/UserDetail":                  {},
	"system/audit/OperationLogList":               {},
	"system/dynamicmodule/ModuleManager":          {},
	"system/generator/ModuleWizard":               {},
	"business/cmdb/host/CmdbHostList":             {},
	"business/cmdb/host/CmdbHostDetail":           {},
	"business/cmdb/group/CmdbGroupList":           {},
	"business/cmdb/label/CmdbLabelSchemaList":     {},
	"business/deploy/package/DeployPackageList":   {},
	"business/deploy/template/DeployTemplateList": {},
	"business/deploy/task/DeployTaskList":         {},
	"business/deploy/task/DeployTaskDetail":       {},
}

var registeredMenuComponentKeys = mergeMenuComponentKeys(staticRegisteredMenuComponentKeys, generatedMenuComponentKeys)

func isRegisteredMenuComponentKey(value string) bool {
	_, ok := registeredMenuComponentKeys[strings.TrimSpace(value)]
	return ok
}

func requiresRegisteredMenuComponent(module string) bool {
	normalized := strings.TrimSpace(module)
	return normalized == "platform" ||
		strings.HasPrefix(normalized, "system.") ||
		strings.HasPrefix(normalized, "business.")
}

func mergeMenuComponentKeys(groups ...map[string]struct{}) map[string]struct{} {
	merged := make(map[string]struct{})
	for _, group := range groups {
		for key := range group {
			merged[key] = struct{}{}
		}
	}
	return merged
}
