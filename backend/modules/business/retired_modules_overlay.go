package business

// 本文件由 pantheon-ops 维护，承载业务模块（cmdb 等）的退役条目。
// base 的 retired_modules.go 只保留通用 hook（retiredBusinessModules 初始为空 + 清理逻辑），
// 具体退役清单属于 ops 业务语义，通过 init() 追加到 base 的 retiredBusinessModules。
// 不要覆盖 base 的 retired_modules.go —— 重建时 base 的 hook 会保留，这里只注入数据。
func init() {
	retiredBusinessModules = append(retiredBusinessModules, retiredModuleSpec{
		ModuleNames: []string{
			"business.cmdb",
			"business.cmdb.host",
			"business.cmdbhostqa",
			"business.cmdb.group",
			"business.cmdb.label",
			"cmdb",
			"cmdb.host",
			"cmdbhostqa",
			"cmdb.group",
		},
		PermissionPrefixes: []string{
			"business:cmdb:",
			"business:cmdbhostqa:",
			"cmdb:",
		},
		MenuPaths: []string{
			"/business/cmdb",
			"/business/cmdb/host",
			"/business/cmdb/host/:id",
			"/business/cmdbhostqa",
			"/business/cmdbhostqa/:id",
			"/business/cmdb/group",
			"/business/cmdb/label",
			"/operations/cmdb",
			"/operations/cmdb/host",
			"/operations/cmdb/host/:id",
			"/operations/cmdb/group",
			"/operations/cmdb/label",
		},
		ComponentKeys: []string{
			"business/cmdb/host/CmdbHostList",
			"business/cmdb/host/CmdbHostDetail",
			"business/cmdbhostqa/CmdbhostqaList",
			"business/cmdbhostqa/CmdbhostqaDetail",
			"business/cmdb/group/CmdbGroupList",
			"business/cmdb/label/CmdbLabelSchemaList",
		},
	})
}
