package post

import (
	"time"

	"gorm.io/gorm"
)

// SystemPost is a post/job title. TenantID is the owning tenant (contract §2.3)
// and PostCode is unique PER TENANT rather than globally.
type SystemPost struct {
	ID        uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	TenantID  uint64         `gorm:"not null;default:0;index:idx_system_post_tenant_id;uniqueIndex:uk_system_post_tenant_code,priority:1" json:"tenantId"`
	DeptID    uint64         `gorm:"default:0;index" json:"deptId"`
	PostCode  string         `gorm:"size:64;not null;uniqueIndex:uk_system_post_tenant_code,priority:2" json:"postCode"`
	PostName  string         `gorm:"size:64;not null" json:"postName"`
	Sort      int            `gorm:"default:0" json:"sort"`
	Status    int            `gorm:"default:1" json:"status"`
	Remark    string         `gorm:"size:255" json:"remark"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (SystemPost) TableName() string {
	return "system_post"
}
