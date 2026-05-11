package org

import (
	"time"

	"gorm.io/gorm"
)

type SystemDept struct {
	ID           uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	ParentID     uint64         `gorm:"default:0" json:"parentId"`
	Ancestors    string         `gorm:"size:255;default:''" json:"ancestors"`
	IsRoot       int            `gorm:"default:0" json:"isRoot"`
	DeptName     string         `gorm:"size:64;not null" json:"deptName"`
	Sort         int            `gorm:"default:0" json:"sort"`
	LeaderUserID uint64         `gorm:"default:0;index" json:"leaderUserId"`
	Leader       string         `gorm:"size:64" json:"leader"`
	Phone        string         `gorm:"size:20" json:"phone"`
	Email        string         `gorm:"size:128" json:"email"`
	Status       int            `gorm:"default:1" json:"status"`
	CreatedAt    time.Time      `json:"createdAt"`
	UpdatedAt    time.Time      `json:"updatedAt"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"-"`
}

func (SystemDept) TableName() string {
	return "system_dept"
}
