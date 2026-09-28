package config

import "time"

// SystemSetting is the platform/global setting row (tenant_id = 0) or a
// tenant override row (tenant_id > 0). Uniqueness is (tenant_id, setting_key)
// per the tenant resource scope matrix ("tenant-overridable: 组合键候选"):
// the legacy global-unique index is replaced by the composite one in Migrate.
type SystemSetting struct {
	ID           uint64    `gorm:"primaryKey;autoIncrement" json:"id"`
	TenantID     uint64    `gorm:"not null;default:0;uniqueIndex:uk_system_setting_tenant_key,priority:1;index" json:"tenantId"`
	SettingKey   string    `gorm:"size:128;not null;uniqueIndex:uk_system_setting_tenant_key,priority:2" json:"settingKey"`
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
