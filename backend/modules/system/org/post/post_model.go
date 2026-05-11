package org

import (
	"time"

	"gorm.io/gorm"
)

type SystemPost struct {
	ID        uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	DeptID    uint64         `gorm:"default:0;index" json:"deptId"`
	PostCode  string         `gorm:"size:64;not null;uniqueIndex" json:"postCode"`
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
