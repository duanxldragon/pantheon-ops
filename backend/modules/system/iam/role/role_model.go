package role

import (
	"time"

	"gorm.io/gorm"
)

// SystemRole 系统角色模型
//
// TenantID is the owning tenant: 0 is a platform/global role, >0 a tenant-local
// one. RoleKey is unique PER TENANT, so each tenant can define its own
// `tenant_admin` without colliding with the platform's roles (contract §4/§7).
type SystemRole struct {
	ID        uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	TenantID  uint64         `gorm:"not null;default:0;index:idx_system_role_tenant_id;uniqueIndex:uk_system_role_tenant_key,priority:1" json:"tenantId"`
	RoleName  string         `gorm:"size:64;not null" json:"roleName"`
	RoleKey   string         `gorm:"size:64;not null;uniqueIndex:uk_system_role_tenant_key,priority:2" json:"roleKey"`
	Sort      int            `gorm:"default:0" json:"sort"`
	Status    int            `gorm:"default:1" json:"status"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (SystemRole) TableName() string {
	return "system_role"
}
