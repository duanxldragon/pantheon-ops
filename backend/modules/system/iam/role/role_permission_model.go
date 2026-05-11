package iam

type SystemRolePermission struct {
	ID            uint64 `gorm:"primaryKey;autoIncrement" json:"id"`
	RoleID        uint64 `gorm:"not null;uniqueIndex:idx_role_permission_unique,priority:1;index" json:"roleId"`
	PermissionKey string `gorm:"size:128;not null;uniqueIndex:idx_role_permission_unique,priority:2" json:"permissionKey"`
}

func (SystemRolePermission) TableName() string {
	return "system_role_permission"
}
