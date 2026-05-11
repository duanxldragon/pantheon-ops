package iam

type SystemRoleMenu struct {
	RoleID uint64 `gorm:"primaryKey;autoIncrement:false" json:"roleId"`
	MenuID uint64 `gorm:"primaryKey;autoIncrement:false" json:"menuId"`
}

func (SystemRoleMenu) TableName() string {
	return "system_role_menu"
}
