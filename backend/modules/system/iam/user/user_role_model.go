package iam

type SystemUserRole struct {
	UserID uint64 `gorm:"column:user_id;not null;index:idx_system_user_role_user"`
	RoleID uint64 `gorm:"column:role_id;not null;index:idx_system_user_role_role"`
}

func (SystemUserRole) TableName() string {
	return "system_user_role"
}
