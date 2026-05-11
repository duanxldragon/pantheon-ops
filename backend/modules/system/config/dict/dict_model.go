package config

import (
	"time"

	"gorm.io/gorm"
)

type SystemDictType struct {
	ID        uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	DictCode  string         `gorm:"size:64;not null;uniqueIndex" json:"dictCode"`
	DictName  string         `gorm:"size:64;not null" json:"dictName"`
	Module    string         `gorm:"size:64;not null;default:system" json:"module"`
	Status    int            `gorm:"default:1" json:"status"`
	Remark    string         `gorm:"size:255" json:"remark"`
	CreatedAt time.Time      `json:"createdAt"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (SystemDictType) TableName() string {
	return "system_dict_type"
}

type SystemDictItem struct {
	ID           uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	DictCode     string         `gorm:"size:64;not null;index:idx_dict_item_code_sort,priority:1" json:"dictCode"`
	ItemLabelKey string         `gorm:"size:128;not null" json:"itemLabelKey"`
	ItemValue    string         `gorm:"size:64;not null;index:idx_dict_item_code_value,priority:2" json:"itemValue"`
	ItemColor    string         `gorm:"size:32" json:"itemColor"`
	Sort         int            `gorm:"default:0;index:idx_dict_item_code_sort,priority:2" json:"sort"`
	Status       int            `gorm:"default:1" json:"status"`
	Remark       string         `gorm:"size:255" json:"remark"`
	CreatedAt    time.Time      `json:"createdAt"`
	UpdatedAt    time.Time      `json:"updatedAt"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

func (SystemDictItem) TableName() string {
	return "system_dict_item"
}
