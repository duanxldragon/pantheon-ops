package config

import (
	"pantheon-ops/backend/pkg/common"
	// Required for the go:embed seed_data.yaml directive below.
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sort"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
	"gorm.io/gorm"
)

const settingModuleAuth = "system.auth"
const settingErrInvalidJSON = "setting.value.invalid_json"
const settingErrInvalidOption = "setting.value.invalid_option"
const settingValueRetentionDaysDefault = "[1,7,30]"

type defaultSettingSeed struct {
	SettingKey   string `yaml:"settingKey"`
	SettingValue string `yaml:"settingValue"`
	ValueType    string `yaml:"valueType"`
	GroupKey     string `yaml:"groupKey"`
	Module       string `yaml:"module"`
	IsPublic     int    `yaml:"isPublic"`
	IsEncrypted  int    `yaml:"isEncrypted"`
	Remark       string `yaml:"remark"`
}

var defaultSettingSeeds = []defaultSettingSeed{
	{SettingKey: "site.name", SettingValue: "Pantheon Base", ValueType: "string", GroupKey: "basic", Module: "system", IsPublic: 1, Remark: "system.setting.remark.site.name"},
	{SettingKey: "site.logo", SettingValue: "", ValueType: "string", GroupKey: "basic", Module: "system", IsPublic: 1, Remark: "system.setting.remark.site.logo"},
	{SettingKey: "platform.app_mode", SettingValue: "enterprise", ValueType: "string", GroupKey: "platform", Module: "platform", IsPublic: 1, Remark: "system.setting.remark.platform.app_mode"},
	{SettingKey: "org.enabled", SettingValue: "true", ValueType: "boolean", GroupKey: "platform", Module: "system.org", IsPublic: 1, Remark: "system.setting.remark.org.enabled"},
	{SettingKey: "org.required_for_user", SettingValue: "false", ValueType: "boolean", GroupKey: "platform", Module: "system.org", IsPublic: 1, Remark: "system.setting.remark.org.required_for_user"},
	{SettingKey: "security.password_min_length", SettingValue: "6", ValueType: "number", GroupKey: "security", Module: "system", IsPublic: 0, Remark: "system.setting.remark.security.password_min_length"},
	{SettingKey: "security.password_require_digit", SettingValue: "false", ValueType: "boolean", GroupKey: "security", Module: settingModuleAuth, IsPublic: 0, Remark: "system.setting.remark.security.password_require_digit"},
	{SettingKey: "security.password_require_uppercase", SettingValue: "false", ValueType: "boolean", GroupKey: "security", Module: settingModuleAuth, IsPublic: 0, Remark: "system.setting.remark.security.password_require_uppercase"},
	{SettingKey: "security.password_history_limit", SettingValue: "0", ValueType: "number", GroupKey: "security", Module: settingModuleAuth, IsPublic: 0, Remark: "system.setting.remark.security.password_history_limit"},
	{SettingKey: "security.password_expire_days", SettingValue: "0", ValueType: "number", GroupKey: "security", Module: settingModuleAuth, IsPublic: 0, Remark: "system.setting.remark.security.password_expire_days"},
	{SettingKey: "login.max_failed_attempts", SettingValue: "5", ValueType: "number", GroupKey: "login", Module: "system", IsPublic: 0, Remark: "system.setting.remark.login.max_failed_attempts"},
	{SettingKey: "login.lock_minutes", SettingValue: "15", ValueType: "number", GroupKey: "login", Module: "system", IsPublic: 0, Remark: "system.setting.remark.login.lock_minutes"},
	{SettingKey: "login.source_max_failed_attempts", SettingValue: "20", ValueType: "number", GroupKey: "login", Module: "system", IsPublic: 0, Remark: "system.setting.remark.login.source_max_failed_attempts"},
	{SettingKey: "login.source_window_minutes", SettingValue: "15", ValueType: "number", GroupKey: "login", Module: "system", IsPublic: 0, Remark: "system.setting.remark.login.source_window_minutes"},
	{SettingKey: "login.source_lock_minutes", SettingValue: "15", ValueType: "number", GroupKey: "login", Module: "system", IsPublic: 0, Remark: "system.setting.remark.login.source_lock_minutes"},
	{SettingKey: "login.security_event_enabled", SettingValue: "true", ValueType: "boolean", GroupKey: "login", Module: settingModuleAuth, IsPublic: 0, Remark: "system.setting.remark.login.security_event_enabled"},
	{SettingKey: "login.captcha_enabled", SettingValue: "false", ValueType: "boolean", GroupKey: "login", Module: "system", IsPublic: 0, Remark: "system.setting.remark.login.captcha_enabled"},
	{SettingKey: "login.mfa_enabled", SettingValue: "false", ValueType: "boolean", GroupKey: "login", Module: "system", IsPublic: 0, Remark: "system.setting.remark.login.mfa_enabled"},
	{SettingKey: "login.sso_enabled", SettingValue: "false", ValueType: "boolean", GroupKey: "login", Module: "system", IsPublic: 0, Remark: "system.setting.remark.login.sso_enabled"},
	{SettingKey: "login.session_idle_minutes", SettingValue: "30", ValueType: "number", GroupKey: "login", Module: "system", IsPublic: 1, Remark: "system.setting.remark.login.session_idle_minutes"},
	{SettingKey: "login.max_active_sessions_per_user", SettingValue: "1", ValueType: "number", GroupKey: "login", Module: "system", IsPublic: 0, Remark: "system.setting.remark.login.max_active_sessions_per_user"},
	{SettingKey: "audit.login_log_retention_options", SettingValue: settingValueRetentionDaysDefault, ValueType: "json", GroupKey: "audit", Module: "system", IsPublic: 0, Remark: "system.setting.remark.audit.login_log_retention_options"},
	{SettingKey: "audit.operation_log_retention_options", SettingValue: settingValueRetentionDaysDefault, ValueType: "json", GroupKey: "audit", Module: "system", IsPublic: 0, Remark: "system.setting.remark.audit.operation_log_retention_options"},
	{SettingKey: "audit.session_cleanup_retention_options", SettingValue: settingValueRetentionDaysDefault, ValueType: "json", GroupKey: "audit", Module: "system", IsPublic: 0, Remark: "system.setting.remark.audit.session_cleanup_retention_options"},
	{SettingKey: "audit.login_log_retention_days", SettingValue: "90", ValueType: "number", GroupKey: "audit", Module: "system", IsPublic: 0, Remark: "system.setting.remark.audit.login_log_retention_days"},
	{SettingKey: "audit.operation_log_retention_days", SettingValue: "180", ValueType: "number", GroupKey: "audit", Module: "system", IsPublic: 0, Remark: "system.setting.remark.audit.operation_log_retention_days"},
	{SettingKey: "audit.session_retention_days", SettingValue: "90", ValueType: "number", GroupKey: "audit", Module: "system", IsPublic: 0, Remark: "system.setting.remark.audit.session_retention_days"},
	{SettingKey: "i18n.default_language", SettingValue: "zh-CN", ValueType: "string", GroupKey: "i18n", Module: "system", IsPublic: 1, Remark: "system.setting.remark.i18n.default_language"},
	{SettingKey: "ui.default_theme", SettingValue: "indigo", ValueType: "string", GroupKey: "ui", Module: "system", IsPublic: 1, Remark: "system.setting.remark.ui.default_theme"},
	{SettingKey: "ui.enable_tab_bar", SettingValue: "true", ValueType: "boolean", GroupKey: "ui", Module: "system", IsPublic: 1, Remark: "system.setting.remark.ui.enable_tab_bar"},
	{SettingKey: "upload.storage_driver", SettingValue: "local", ValueType: "string", GroupKey: "upload", Module: "system", IsPublic: 0, Remark: "system.setting.remark.upload.storage_driver"},
	{SettingKey: "upload.max_file_size", SettingValue: "20", ValueType: "number", GroupKey: "upload", Module: "system", IsPublic: 0, Remark: "system.setting.remark.upload.max_file_size"},
	{SettingKey: "upload.allowed_types", SettingValue: "[\"jpg\",\"jpeg\",\"png\",\"pdf\",\"doc\",\"docx\",\"xls\",\"xlsx\",\"zip\",\"gz\",\"tgz\",\"tar\"]", ValueType: "json", GroupKey: "upload", Module: "system", IsPublic: 0, Remark: "system.setting.remark.upload.allowed_types"},
	{SettingKey: "upload.local_path", SettingValue: "./uploads", ValueType: "string", GroupKey: "upload", Module: "system", IsPublic: 0, Remark: "system.setting.remark.upload.local_path"},
	{SettingKey: "upload.public_base_url", SettingValue: "", ValueType: "string", GroupKey: "upload", Module: "system", IsPublic: 0, Remark: "system.setting.remark.upload.public_base_url"},
	{SettingKey: "upload.s3_endpoint", SettingValue: "", ValueType: "string", GroupKey: "upload", Module: "system", IsPublic: 0, Remark: "system.setting.remark.upload.s3_endpoint"},
	{SettingKey: "upload.s3_bucket", SettingValue: "", ValueType: "string", GroupKey: "upload", Module: "system", IsPublic: 0, Remark: "system.setting.remark.upload.s3_bucket"},
	{SettingKey: "upload.s3_region", SettingValue: "us-east-1", ValueType: "string", GroupKey: "upload", Module: "system", IsPublic: 0, Remark: "system.setting.remark.upload.s3_region"},
	{SettingKey: "upload.s3_access_key_id", SettingValue: "", ValueType: "string", GroupKey: "upload", Module: "system", IsPublic: 0, IsEncrypted: 1, Remark: "system.setting.remark.upload.s3_access_key_id"},
	{SettingKey: "upload.s3_secret_access_key", SettingValue: "", ValueType: "string", GroupKey: "upload", Module: "system", IsPublic: 0, IsEncrypted: 1, Remark: "system.setting.remark.upload.s3_secret_access_key"},
}

//go:embed seed_data.yaml
var settingSeedYAML []byte

// settingSeedsYAML is the YAML-compatible struct for parsing seed_data.yaml.
type settingSeedsYAML struct {
	Settings []defaultSettingSeed `yaml:"settings"`
}

// loadedSettingSeeds holds the seeds loaded from YAML (with Go fallback).
var loadedSettingSeeds []defaultSettingSeed
var defaultSettingSeedMap map[string]defaultSettingSeed

func init() {
	loadedSettingSeeds = loadSettingSeedsFromYAML()
	defaultSettingSeedMap = buildDefaultSettingSeedMap(loadedSettingSeeds)
}

func loadSettingSeedsFromYAML() []defaultSettingSeed {
	if len(settingSeedYAML) == 0 {
		log.Println("[config] WARNING: setting seed YAML is empty, falling back to hardcoded defaults")
		return defaultSettingSeeds
	}
	var data settingSeedsYAML
	if err := yaml.Unmarshal(settingSeedYAML, &data); err != nil {
		log.Printf("[config] WARNING: failed to parse setting seed YAML: %v, falling back to hardcoded defaults", err)
		return defaultSettingSeeds
	}
	if len(data.Settings) == 0 {
		log.Println("[config] WARNING: setting seed YAML has no entries, falling back to hardcoded defaults")
		return defaultSettingSeeds
	}
	if err := validateSettingSeeds(data.Settings); err != nil {
		log.Printf("[config] WARNING: setting seed YAML has invalid entries: %v, falling back to hardcoded defaults", err)
		return defaultSettingSeeds
	}
	return data.Settings
}

// settingSeeds returns the active setting seeds (YAML-loaded with Go fallback).
func settingSeeds() []defaultSettingSeed {
	return loadedSettingSeeds
}

var (
	allowedLanguageValues = map[string]struct{}{
		"zh-CN": {},
		"en-US": {},
		"ja-JP": {},
		"ko-KR": {},
		"fr-FR": {},
	}
	allowedThemeValues = map[string]struct{}{
		"indigo":  {},
		"emerald": {},
		"violet":  {},
		"slate":   {},
	}
	allowedStorageDriverValues = map[string]struct{}{
		"local": {},
		"s3":    {},
	}
	allowedAppModeValues = map[string]struct{}{
		"enterprise": {},
		"consumer":   {},
		"hybrid":     {},
	}
)

func (s *SettingService) normalizeLegacySettingValue(settingKey string) error {
	var row SystemSetting
	if err := s.db.Where("setting_key = ?", settingKey).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	if row.IsEncrypted == 1 {
		return nil
	}
	normalizedValue, err := normalizeSettingValue(row.SettingKey, row.SettingValue)
	if err != nil {
		return err
	}
	if normalizedValue == row.SettingValue {
		return nil
	}
	return s.db.Model(&row).Update("setting_value", normalizedValue).Error
}

func (s *SettingService) migrateLegacySettingValue(settingKey, legacyValue, nextValue string) error {
	var row SystemSetting
	if err := s.db.Where("setting_key = ?", settingKey).First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return err
	}
	if row.IsEncrypted == 1 {
		return nil
	}
	if strings.TrimSpace(row.SettingValue) != strings.TrimSpace(legacyValue) {
		return nil
	}
	normalizedValue, err := validateAndNormalizeSettingValue(row.SettingKey, row.ValueType, nextValue)
	if err != nil {
		return err
	}
	return s.db.Model(&row).Update("setting_value", normalizedValue).Error
}

func normalizeSettingGroups(groupKeys []string) []string {
	result := make([]string, 0, len(groupKeys))
	seen := make(map[string]struct{}, len(groupKeys))
	for _, groupKey := range groupKeys {
		trimmed := strings.TrimSpace(groupKey)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}

func (s *SettingService) invalidateSettingCache() {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	s.listCache = make(map[string][]SettingResp)
	s.groupCache = make(map[string]*SettingGroupResp)
	s.publicCache = nil
}

// invalidateSettingCacheForGroup invalidates only the cache entries
// related to a specific group, preserving the rest.
func (s *SettingService) invalidateSettingCacheForGroup(groupKey string) {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	// Remove the specific group cache
	delete(s.groupCache, groupKey)
	// Remove list cache entries that may contain this group's settings.
	// Since listCache may be keyed by various criteria, safest to clear it all
	// (listCache is typically small and rebuilt quickly on demand).
	s.listCache = make(map[string][]SettingResp)
	// If the group contains public settings, publicCache must also be invalidated
	s.publicCache = nil
}

func appendSettingOverviewIssue(issues []SettingOverviewIssueResp, seen map[string]struct{}, issue SettingOverviewIssueResp) []SettingOverviewIssueResp {
	key := issue.SettingKey + "|" + issue.ReasonKey
	if _, ok := seen[key]; ok {
		return issues
	}
	seen[key] = struct{}{}
	return append(issues, issue)
}

func inferSettingGroupKey(settingKey string) string {
	switch {
	case strings.HasPrefix(settingKey, "site."):
		return "basic"
	case strings.HasPrefix(settingKey, "platform."), strings.HasPrefix(settingKey, "org."):
		return "platform"
	case strings.HasPrefix(settingKey, "security."):
		return "security"
	case strings.HasPrefix(settingKey, "login."):
		return "login"
	case strings.HasPrefix(settingKey, "audit."):
		return "audit"
	case strings.HasPrefix(settingKey, "upload."):
		return "upload"
	case strings.HasPrefix(settingKey, "i18n."):
		return "i18n"
	case strings.HasPrefix(settingKey, "ui."):
		return "ui"
	default:
		return "system"
	}
}

func systemSettingHasValue(item SystemSetting) bool {
	return strings.TrimSpace(item.SettingValue) != ""
}

func safeSettingOverviewValue(item SystemSetting, fallback string) string {
	if strings.TrimSpace(item.SettingValue) == "" {
		return fallback
	}
	if item.IsEncrypted == 1 {
		return fallback
	}
	return item.SettingValue
}

func validateAndNormalizeSettingValue(settingKey, valueType, value string) (string, error) {
	normalizedValue, err := normalizeSettingValue(settingKey, value)
	if err != nil {
		return "", err
	}

	switch strings.TrimSpace(valueType) {
	case "string":
		return normalizedValue, nil
	case "number":
		return validateNumberSettingValue(normalizedValue)
	case "boolean":
		return validateBooleanSettingValue(normalizedValue)
	case "json":
		return validateJSONSettingValue(settingKey, normalizedValue)
	default:
		return "", common.NewBadRequest("setting.value_type.invalid")
	}
}

func validateNumberSettingValue(normalizedValue string) (string, error) {
	if _, err := strconv.ParseFloat(strings.TrimSpace(normalizedValue), 64); err != nil {
		return "", common.NewBadRequest("setting.value.invalid_number")
	}
	return normalizedValue, nil
}

func validateBooleanSettingValue(normalizedValue string) (string, error) {
	if _, err := strconv.ParseBool(strings.TrimSpace(normalizedValue)); err != nil {
		return "", common.NewBadRequest("setting.value.invalid_boolean")
	}
	return normalizedValue, nil
}

func validateJSONSettingValue(settingKey, normalizedValue string) (string, error) {
	trimmed := strings.TrimSpace(normalizedValue)
	if trimmed == "" {
		return normalizedValue, nil
	}
	target, err := parseSettingJSON(trimmed)
	if err != nil {
		return "", err
	}
	if err := validateSettingJSONShape(settingKey, target); err != nil {
		return "", err
	}
	if isAuditRetentionOptionsSetting(settingKey) {
		return normalizeAuditRetentionOptions(trimmed)
	}
	return trimmed, nil
}

func parseSettingJSON(raw string) (interface{}, error) {
	var target interface{}
	if err := json.Unmarshal([]byte(raw), &target); err != nil {
		return nil, errors.New(settingErrInvalidJSON)
	}
	return target, nil
}

func validateSettingJSONShape(settingKey string, target interface{}) error {
	if settingKey != "upload.allowed_types" {
		return nil
	}
	if _, ok := target.([]interface{}); !ok {
		return errors.New(settingErrInvalidJSON)
	}
	return nil
}

func isAuditRetentionOptionsSetting(settingKey string) bool {
	return settingKey == "audit.login_log_retention_options" ||
		settingKey == "audit.operation_log_retention_options" ||
		settingKey == "audit.session_cleanup_retention_options"
}

// SettingNormalizer normalizes a setting value for a specific key.
// Returns the normalized value and an error if the value is invalid.
type SettingNormalizer func(value string) (string, error)

// settingNormalizers is a registry of key-specific normalizers.
// New setting keys can be registered via RegisterSettingNormalizer without modifying normalizeSettingValue.
var settingNormalizers = map[string]SettingNormalizer{
	"platform.app_mode": func(value string) (string, error) {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			trimmed = "enterprise"
		}
		if _, ok := allowedAppModeValues[trimmed]; !ok {
			return "", errors.New(settingErrInvalidOption)
		}
		return trimmed, nil
	},
	"upload.storage_driver": func(value string) (string, error) {
		trimmed := strings.TrimSpace(value)
		switch trimmed {
		case "s3-compatible":
			trimmed = "s3"
		case "":
			trimmed = "local"
		}
		if _, ok := allowedStorageDriverValues[trimmed]; !ok {
			return "", errors.New(settingErrInvalidOption)
		}
		return trimmed, nil
	},
	"ui.default_theme": func(value string) (string, error) {
		trimmed := strings.TrimSpace(value)
		switch trimmed {
		case "light":
			trimmed = "indigo"
		case "":
			trimmed = "indigo"
		}
		if _, ok := allowedThemeValues[trimmed]; !ok {
			return "", errors.New(settingErrInvalidOption)
		}
		return trimmed, nil
	},
	"i18n.default_language": func(value string) (string, error) {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			trimmed = "zh-CN"
		}
		if _, ok := allowedLanguageValues[trimmed]; !ok {
			return "", errors.New(settingErrInvalidOption)
		}
		return trimmed, nil
	},
}

// RegisterSettingNormalizer adds a custom normalizer for a setting key.
// This allows external modules to register their own normalizers without modifying this file.
func RegisterSettingNormalizer(key string, normalizer SettingNormalizer) {
	settingNormalizers[key] = normalizer
}

func normalizeSettingValue(settingKey, value string) (string, error) {
	if normalizer, ok := settingNormalizers[settingKey]; ok {
		return normalizer(value)
	}
	return strings.TrimSpace(value), nil
}

func normalizeAuditRetentionOptions(raw string) (string, error) {
	var values []int
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &values); err != nil {
		return "", errors.New(settingErrInvalidJSON)
	}
	if len(values) == 0 {
		return "", errors.New(settingErrInvalidOption)
	}

	seen := make(map[int]struct{}, len(values))
	normalized := make([]int, 0, len(values))
	for _, value := range values {
		if value <= 0 || value > 365 {
			return "", errors.New(settingErrInvalidOption)
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		normalized = append(normalized, value)
	}
	if len(normalized) == 0 {
		return "", errors.New(settingErrInvalidOption)
	}

	sort.Ints(normalized)
	normalizedJSON, err := json.Marshal(normalized)
	if err != nil {
		return "", errors.New(settingErrInvalidJSON)
	}
	return string(normalizedJSON), nil
}

func prepareSettingStoredValue(value string, isEncrypted int) (string, error) {
	trimmed := strings.TrimSpace(value)
	if isEncrypted != 1 {
		return trimmed, nil
	}
	return encryptSettingValue(trimmed)
}

func buildDefaultSettingSeedMap(seeds []defaultSettingSeed) map[string]defaultSettingSeed {
	result := make(map[string]defaultSettingSeed, len(seeds))
	for _, seed := range seeds {
		result[seed.SettingKey] = seed
	}
	return result
}

func validateSettingSeeds(seeds []defaultSettingSeed) error {
	for index, seed := range seeds {
		switch {
		case strings.TrimSpace(seed.SettingKey) == "":
			return fmt.Errorf("seed[%d] missing settingKey", index)
		case strings.TrimSpace(seed.ValueType) == "":
			return fmt.Errorf("seed[%d] missing valueType for %s", index, seed.SettingKey)
		case strings.TrimSpace(seed.GroupKey) == "":
			return fmt.Errorf("seed[%d] missing groupKey for %s", index, seed.SettingKey)
		case strings.TrimSpace(seed.Module) == "":
			return fmt.Errorf("seed[%d] missing module for %s", index, seed.SettingKey)
		case strings.TrimSpace(seed.Remark) == "":
			return fmt.Errorf("seed[%d] missing remark for %s", index, seed.SettingKey)
		}
	}
	return nil
}

func defaultSettingValue(settingKey string) string {
	seed, ok := defaultSettingSeedMap[settingKey]
	if !ok {
		return ""
	}
	return seed.SettingValue
}
