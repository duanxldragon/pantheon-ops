package bizscope

import (
	"time"

	"gorm.io/gorm"
)

// BusinessEnvironment belongs to a business scope; authorization follows its parent.
type BusinessEnvironment struct {
	ID              uint64 `gorm:"primaryKey;autoIncrement" json:"id"`
	BusinessScopeID uint64 `gorm:"not null;index;uniqueIndex:idx_scope_env_code" json:"businessScopeId"`
	// A deleted row retains its code, matching the explicitly chosen environment contract.
	Code      string         `gorm:"not null;size:128;uniqueIndex:idx_scope_env_code" json:"code"`
	Name      string         `gorm:"not null;size:255" json:"name"`
	EnvType   string         `gorm:"not null;size:16" json:"envType"`
	Status    string         `gorm:"not null;size:16" json:"status"`
	Remark    string         `gorm:"type:text" json:"remark"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (BusinessEnvironment) TableName() string { return "biz_business_environment" }
