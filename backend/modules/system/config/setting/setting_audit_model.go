package config

import "time"

type systemSettingAuditLog struct {
	ID           uint64 `gorm:"primaryKey;autoIncrement"`
	Title        string `gorm:"size:64"`
	BusinessType int    `gorm:"default:0"`
	Method       string `gorm:"size:128"`
	OperName     string `gorm:"size:64"`
	OperURL      string `gorm:"size:255"`
	OperIP       string `gorm:"size:128"`
	OperParam    string `gorm:"type:text"`
	JsonResult   string `gorm:"type:text"`
	Status       int    `gorm:"default:1"`
	ErrorMsg     string `gorm:"type:text"`
	OperTime     time.Time
	CostTime     int64
}

func (systemSettingAuditLog) TableName() string {
	return "system_log_oper"
}
