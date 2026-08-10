package common

// 内置超级管理员保护常量。
// 系统初始化时创建的内置管理员用户与管理员角色不允许被禁用、降权或删除，
// 各模块的保护判断统一引用这里的常量，禁止散布魔法值。
const (
	// BuiltinAdminUserID 内置超级管理员用户 ID（system_init.sql 种子数据）。
	BuiltinAdminUserID uint64 = 1
	// BuiltinAdminRoleID 内置管理员角色 ID（system_init.sql 种子数据）。
	BuiltinAdminRoleID uint64 = 1
	// AdminRoleKey 内置管理员角色 Key，拥有全部权限。
	AdminRoleKey = "admin"
)
