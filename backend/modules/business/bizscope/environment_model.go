package bizscope

import (
	"time"

	"gorm.io/gorm"
)

// BusinessEnvironment is a per-scope environment definition promoted in the
// bizscope M+3C. Each entry belongs to one business scope (business_scope_id)
// and carries an environment type (dev/test/prod) plus an operational code.
// It is the only net-new CRUD entity in the bizscope M+3C.
type BusinessEnvironment struct {
	ID              uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	TenantID        uint64         `gorm:"column:tenant_id;not null;default:0" json:"-"`
	BusinessScopeID uint64         `gorm:"not null;index" json:"businessScopeId"`
	Code            string         `gorm:"not null;size:50" json:"code"`
	Name            string         `gorm:"not null;size:255" json:"name"`
	EnvType         string         `gorm:"not null;size:50" json:"envType"`
	Status          string         `gorm:"not null;size:50" json:"status"`
	Remark          string         `gorm:"type:text" json:"remark"`
	CreatedAt       time.Time      `json:"createdAt"`
	UpdatedAt       time.Time      `json:"updatedAt"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
}

// TableName returns the environment storage table.
func (BusinessEnvironment) TableName() string {
	return "biz_business_environment"
}
