package iam

import (
	"time"

	"gorm.io/gorm"
)

// SystemUser 系统用户模型
type SystemUser struct {
	ID                  uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	Username            string         `gorm:"size:64;not null;uniqueIndex" json:"username"`
	Password            string         `gorm:"size:255;not null" json:"-"` // 不参与 JSON 序列化
	Nickname            string         `gorm:"size:64" json:"nickname"`
	Avatar              string         `gorm:"size:255" json:"avatar"`
	Email               string         `gorm:"size:128" json:"email"`
	Phone               string         `gorm:"size:20" json:"phone"`
	PreferenceJSON      string         `gorm:"type:text" json:"-"`
	DeptID              uint64         `gorm:"default:0" json:"deptId"`
	PostID              uint64         `gorm:"default:0" json:"postId"`
	Status              int            `gorm:"default:1" json:"status"` // 1:正常, 2:禁用
	FailedLoginAttempts int            `gorm:"default:0" json:"-"`
	LoginLockedUntil    *time.Time     `json:"-"`
	CreatedAt           time.Time      `json:"createdAt"`
	UpdatedAt           time.Time      `json:"updatedAt"`
	DeletedAt           gorm.DeletedAt `gorm:"index" json:"-"`
}

type SystemUserProfileExt struct {
	UserID      uint64 `gorm:"primaryKey" json:"userId"`
	ProfileJSON string `gorm:"type:text" json:"-"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (SystemUserProfileExt) TableName() string {
	return "system_user_profile_ext"
}

// TableName 指定表名
func (SystemUser) TableName() string {
	return "system_user"
}
