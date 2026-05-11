package group

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Group struct {
	ID          uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	ParentID    uint64         `gorm:"index;default:0" json:"parentId"`
	Name        string         `gorm:"size:128;not null" json:"name"`
	Conditions  datatypes.JSON `gorm:"type:json;not null" json:"conditions"`
	Description string         `gorm:"size:512" json:"description"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}

func (Group) TableName() string {
	return "biz_cmdb_group"
}

// Local host model for condition filtering in the group package.
// Avoids circular import with the host package.
type Host struct {
	ID          uint64         `gorm:"primaryKey"`
	Hostname    string         `gorm:"column:hostname"`
	IP          string         `gorm:"column:ip"`
	LabelValues datatypes.JSON `gorm:"column:label_values"`
	Status      string         `gorm:"column:status"`
	DeptID      uint64         `gorm:"column:dept_id"`
	OS          string         `gorm:"column:os"`
	OSVersion   string         `gorm:"column:os_version"`
	CPUCores    int            `gorm:"column:cpu_cores"`
	MemoryGB    float64        `gorm:"column:memory_gb"`
	DiskGB      float64        `gorm:"column:disk_gb"`
	DeletedAt   gorm.DeletedAt
}

func (Host) TableName() string { return "biz_cmdb_host" }
