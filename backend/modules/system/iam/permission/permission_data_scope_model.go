package iam

type PermissionRoleDataScopePolicy struct {
	ID      uint64 `gorm:"primaryKey;autoIncrement"`
	RoleKey string `gorm:"size:64;not null;uniqueIndex"`
	Mode    string `gorm:"size:32;not null;default:'all'"`
	DeptIDs string `gorm:"type:text"`
}

func (PermissionRoleDataScopePolicy) TableName() string {
	return "system_role_data_scope"
}
