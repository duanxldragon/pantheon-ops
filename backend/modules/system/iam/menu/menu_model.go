package iam

import (
	"time"
)

// SystemMenu 系统菜单模型
type SystemMenu struct {
	ID         uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	ParentID   uint64    `gorm:"default:0" json:"parentId"`
	TitleKey   string    `gorm:"size:64;not null" json:"titleKey"` // i18n key
	Path       string    `gorm:"size:255;default:''" json:"path"`
	Component  string    `gorm:"size:255;default:''" json:"component"`
	PagePerm   string    `gorm:"size:128;default:''" json:"pagePerm"`
	Perms      string    `gorm:"size:128;default:''" json:"perms"`
	Type       string    `gorm:"size:1;default:'M'" json:"type"` // M目录 C菜单 F按钮
	Icon       string    `gorm:"size:128;default:''" json:"icon"`
	RouteName  string    `gorm:"size:128;default:''" json:"routeName"`
	Module     string    `gorm:"size:64;default:'system'" json:"module"`
	Sort       int       `gorm:"default:0" json:"sort"`
	IsVisible  int       `gorm:"default:1" json:"isVisible"` // 1:是, 0:否
	IsCache    int       `gorm:"default:0" json:"isCache"`
	IsExternal int       `gorm:"default:0" json:"isExternal"`
	ActiveMenu string    `gorm:"size:255;default:''" json:"activeMenu"`
	HideInNav  int       `gorm:"default:0" json:"hideInNav"` // 1:隐藏导航, 0:正常显示
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

func (SystemMenu) TableName() string {
	return "system_menu"
}
