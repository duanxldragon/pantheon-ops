package role

// SystemRolePermission grants one permission key to one role inside a tenant
// (contract §4). TenantID joins the natural key so a tenant-local role and a
// platform role can both hold the same permission key.
type SystemRolePermission struct {
	ID            uint64 `gorm:"primaryKey;autoIncrement" json:"id"`
	TenantID      uint64 `gorm:"not null;default:0;uniqueIndex:uk_role_permission_tenant,priority:1;index:idx_role_permission_tenant_id" json:"tenantId"`
	RoleID        uint64 `gorm:"not null;index;uniqueIndex:uk_role_permission_tenant,priority:2" json:"roleId"`
	PermissionKey string `gorm:"size:128;not null;uniqueIndex:uk_role_permission_tenant,priority:3" json:"permissionKey"`
}

func (SystemRolePermission) TableName() string {
	return "system_role_permission"
}
