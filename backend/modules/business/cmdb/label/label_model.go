package label

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type LabelSchema struct {
	ID          uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	Key         string         `gorm:"size:64;not null;uniqueIndex:uk_cmdb_label_schema_key" json:"key"`
	Name        string         `gorm:"size:128;not null" json:"name"`
	ValueMode   string         `gorm:"size:16;not null;default:free" json:"valueMode"`
	DictCode    string         `gorm:"size:64" json:"dictCode"`
	Options     datatypes.JSON `gorm:"type:json" json:"options"`
	Required    bool           `gorm:"default:false" json:"required"`
	Status      string         `gorm:"size:16;not null;default:enabled;index" json:"status"`
	Description string         `gorm:"size:512" json:"description"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

func (LabelSchema) TableName() string {
	return "biz_cmdb_label_schema"
}
