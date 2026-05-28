package bizscope

import (
	"time"

	"gorm.io/gorm"
)

type BizScope struct {
	ID          uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	Code        string         `gorm:"uniqueIndex;not null;size:255" json:"code"`
	Name        string         `gorm:"not null;size:255" json:"name"`
	Owner       string         `gorm:"size:255" json:"owner"`
	Environment string         `gorm:"not null;size:50" json:"environment"`
	Status      string         `gorm:"not null;size:50" json:"status"`
	Remark      string         `gorm:"type:text" json:"remark"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

func (BizScope) TableName() string {
	return "biz_business_scope"
}
