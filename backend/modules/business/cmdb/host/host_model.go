package host

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Host struct {
	ID                  uint64         `gorm:"primaryKey;autoIncrement" json:"id"`
	Hostname            string         `gorm:"size:128;not null" json:"hostname"`
	IP                  string         `gorm:"size:45;not null;index:uk_ip_deleted,unique" json:"ip"`
	SSHPort             int            `gorm:"default:22" json:"sshPort"`
	OS                  string         `gorm:"size:32;not null;index" json:"os"`
	OSVersion           string         `gorm:"size:128" json:"osVersion"`
	CPUCores            int            `json:"cpuCores"`
	MemoryGB            float64        `json:"memoryGb"`
	DiskGB              float64        `json:"diskGb"`
	LabelValues         datatypes.JSON `gorm:"type:json" json:"labelValues"`
	InstalledComponents datatypes.JSON `gorm:"type:json" json:"installedComponents"`
	Status              string         `gorm:"size:32;default:pending;index" json:"status"`
	BusinessScopeID     uint64         `gorm:"column:business_scope_id;index" json:"businessScopeId"`
	BusinessScopeCode   string         `gorm:"column:business_scope_code;size:64" json:"businessScopeCode"`
	BusinessScopeName   string         `gorm:"column:business_scope_name;size:128" json:"businessScopeName"`
	DeptID              uint64         `gorm:"column:dept_id;index" json:"deptId"`
	Owner               string         `gorm:"size:64" json:"owner"`
	Remark              string         `gorm:"type:text" json:"remark"`
	CreatedAt           time.Time      `json:"createdAt"`
	UpdatedAt           time.Time      `json:"updatedAt"`
	CreatedBy           string         `gorm:"size:64" json:"createdBy"`
	UpdatedBy           string         `gorm:"size:64" json:"updatedBy"`
	DeletedAt           gorm.DeletedAt `gorm:"index:uk_ip_deleted,unique" json:"-"`
}

func (Host) TableName() string {
	return "biz_cmdb_host"
}
