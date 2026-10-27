package role

// SystemRoleMenu grants a role one menu within the role's tenant (contract §4).
//
// TenantID joins the primary key so a tenant-local role and a platform role can
// both reference the same menu row without conflicting.
type SystemRoleMenu struct {
	TenantID uint64 `gorm:"column:tenant_id;not null;default:0;primaryKey;autoIncrement:false" json:"tenantId"`
	RoleID   uint64 `gorm:"primaryKey;autoIncrement:false" json:"roleId"`
	MenuID   uint64 `gorm:"primaryKey;autoIncrement:false;index:idx_system_role_menu_menu" json:"menuId"`
}

func (SystemRoleMenu) TableName() string {
	return "system_role_menu"
}
