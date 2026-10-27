package permission

// PermissionRoleDataScopePolicy is a role's data-scope policy. TenantID scopes
// it (contract §2.3) and RoleKey is unique PER TENANT so a tenant-local role can
// carry its own policy without touching the platform's.
type PermissionRoleDataScopePolicy struct {
	ID       uint64 `gorm:"primaryKey;autoIncrement"`
	TenantID uint64 `gorm:"not null;default:0;uniqueIndex:uk_role_data_scope_tenant_role,priority:1"`
	RoleKey  string `gorm:"size:64;not null;uniqueIndex:uk_role_data_scope_tenant_role,priority:2"`
	Mode     string `gorm:"size:32;not null;default:'all'"`
	DeptIDs  string `gorm:"type:text"`
}

func (PermissionRoleDataScopePolicy) TableName() string {
	return "system_role_data_scope"
}
