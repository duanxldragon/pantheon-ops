package capability

import (
	"strings"

	"gorm.io/gorm"
)

// Supported platform application modes.
const (
	AppModeEnterprise = "enterprise"
	AppModeConsumer   = "consumer"
	AppModeHybrid     = "hybrid"
)

// PlatformCapabilities captures the current platform feature toggles.
type PlatformCapabilities struct {
	AppMode            string
	OrgEnabled         bool
	OrgRequiredForUser bool
}

type settingRow struct {
	SettingKey   string
	SettingValue string
}

// Defaults returns the platform capability defaults.
func Defaults() PlatformCapabilities {
	return PlatformCapabilities{
		AppMode:            AppModeEnterprise,
		OrgEnabled:         true,
		OrgRequiredForUser: false,
	}
}

// Load resolves platform capabilities from system settings.
func Load(db *gorm.DB) PlatformCapabilities {
	caps := Defaults()
	if db == nil {
		return caps
	}

	var rows []settingRow
	// Platform-scoped read: these flags describe the deployment, so only the
	// platform-global rows (tenant_id = 0) apply. Without the filter a tenant's
	// settings copy could win and make capabilities tenant-dependent.
	if err := db.Table("system_setting").
		Select("setting_key, setting_value").
		Where("setting_key IN ?", []string{"platform.app_mode", "org.enabled", "org.required_for_user"}).
		Where("tenant_id = ?", 0).
		Find(&rows).Error; err != nil {
		return caps
	}

	for _, row := range rows {
		switch strings.TrimSpace(row.SettingKey) {
		case "platform.app_mode":
			caps.AppMode = NormalizeAppMode(row.SettingValue)
		case "org.enabled":
			caps.OrgEnabled = normalizeBool(row.SettingValue, caps.OrgEnabled)
		case "org.required_for_user":
			caps.OrgRequiredForUser = normalizeBool(row.SettingValue, caps.OrgRequiredForUser)
		}
	}
	return caps
}

// NormalizeAppMode normalizes a raw app mode value.
func NormalizeAppMode(value string) string {
	switch strings.TrimSpace(value) {
	case AppModeConsumer:
		return AppModeConsumer
	case AppModeHybrid:
		return AppModeHybrid
	default:
		return AppModeEnterprise
	}
}

func normalizeBool(value string, fallback bool) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "true", "1", "yes", "on":
		return true
	case "false", "0", "no", "off":
		return false
	default:
		return fallback
	}
}
