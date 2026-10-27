package user

// SystemUserRole binds a user to a role within one tenant (contract §2.3/§4).
//
// TenantID is part of the primary key so the same (user, role) pair can exist in
// different tenants without collapsing into one row. Role keys are namespaced by
// tenant as well, so a tenant's `tenant_admin` never resolves to the platform's.
type SystemUserRole struct {
	TenantID uint64 `gorm:"column:tenant_id;not null;default:0;primaryKey;autoIncrement:false;index:idx_system_user_role_tenant_user,priority:1" json:"tenantId"`
	UserID   uint64 `gorm:"column:user_id;not null;primaryKey;autoIncrement:false;index:idx_system_user_role_user;index:idx_system_user_role_tenant_user,priority:2" json:"userId"`
	RoleID   uint64 `gorm:"column:role_id;not null;primaryKey;autoIncrement:false;index:idx_system_user_role_role" json:"roleId"`
}

func (SystemUserRole) TableName() string {
	return "system_user_role"
}
