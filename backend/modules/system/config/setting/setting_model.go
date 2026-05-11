package config

import "time"

type SystemSetting struct {
	ID           uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	SettingKey   string    `gorm:"size:128;not null;uniqueIndex" json:"settingKey"`
	SettingValue string    `gorm:"type:text" json:"settingValue"`
	ValueType    string    `gorm:"size:16;not null;default:string" json:"valueType"`
	GroupKey     string    `gorm:"size:32;not null;index" json:"groupKey"`
	Module       string    `gorm:"size:64;not null;default:system" json:"module"`
	IsPublic     int       `gorm:"default:0" json:"isPublic"`
	IsEncrypted  int       `gorm:"default:0" json:"isEncrypted"`
	Remark       string    `gorm:"size:255" json:"remark"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

func (SystemSetting) TableName() string {
	return "system_setting"
}
