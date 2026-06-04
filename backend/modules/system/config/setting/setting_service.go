package config

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"pantheon-platform/backend/pkg/contracts"
	"pantheon-platform/backend/pkg/database"
	"pantheon-platform/backend/pkg/impexp"

	"gorm.io/gorm"
)

type SettingService struct {
	db          *gorm.DB
	cacheMu     sync.RWMutex
	listCache   map[string][]SettingResp
	groupCache  map[string]*SettingGroupResp
	publicCache *PublicSettingResp
}

type defaultSettingSeed struct {
	SettingKey   string
	SettingValue string
	ValueType    string
	GroupKey     string
	Module       string
	IsPublic     int
	IsEncrypted  int
	Remark       string
}

var defaultSettingSeeds = []defaultSettingSeed{
	{SettingKey: "site.name", SettingValue: "Pantheon Base", ValueType: "string", GroupKey: "basic", Module: "system", IsPublic: 1, Remark: "system.setting.remark.site.name"},
	{SettingKey: "site.logo", SettingValue: "", ValueType: "string", GroupKey: "basic", Module: "system", IsPublic: 1, Remark: "system.setting.remark.site.logo"},
	{SettingKey: "platform.app_mode", SettingValue: "enterprise", ValueType: "string", GroupKey: "platform", Module: "platform", IsPublic: 1, Remark: "system.setting.remark.platform.app_mode"},
	{SettingKey: "org.enabled", SettingValue: "true", ValueType: "boolean", GroupKey: "platform", Module: "system.org", IsPublic: 1, Remark: "system.setting.remark.org.enabled"},
	{SettingKey: "org.required_for_user", SettingValue: "false", ValueType: "boolean", GroupKey: "platform", Module: "system.org", IsPublic: 1, Remark: "system.setting.remark.org.required_for_user"},
	{SettingKey: "security.password_min_length", SettingValue: "6", ValueType: "number", GroupKey: "security", Module: "system", IsPublic: 0, Remark: "system.setting.remark.security.password_min_length"},
	{SettingKey: "security.password_require_digit", SettingValue: "false", ValueType: "boolean", GroupKey: "security", Module: "system.auth", IsPublic: 0, Remark: "system.setting.remark.security.password_require_digit"},
	{SettingKey: "security.password_require_uppercase", SettingValue: "false", ValueType: "boolean", GroupKey: "security", Module: "system.auth", IsPublic: 0, Remark: "system.setting.remark.security.password_require_uppercase"},
	{SettingKey: "security.password_history_limit", SettingValue: "0", ValueType: "number", GroupKey: "security", Module: "system.auth", IsPublic: 0, Remark: "system.setting.remark.security.password_history_limit"},
	{SettingKey: "security.password_expire_days", SettingValue: "0", ValueType: "number", GroupKey: "security", Module: "system.auth", IsPublic: 0, Remark: "system.setting.remark.security.password_expire_days"},
	{SettingKey: "login.max_failed_attempts", SettingValue: "5", ValueType: "number", GroupKey: "login", Module: "system", IsPublic: 0, Remark: "system.setting.remark.login.max_failed_attempts"},
	{SettingKey: "login.lock_minutes", SettingValue: "15", ValueType: "number", GroupKey: "login", Module: "system", IsPublic: 0, Remark: "system.setting.remark.login.lock_minutes"},
	{SettingKey: "login.source_max_failed_attempts", SettingValue: "20", ValueType: "number", GroupKey: "login", Module: "system", IsPublic: 0, Remark: "system.setting.remark.login.source_max_failed_attempts"},
	{SettingKey: "login.source_window_minutes", SettingValue: "15", ValueType: "number", GroupKey: "login", Module: "system", IsPublic: 0, Remark: "system.setting.remark.login.source_window_minutes"},
	{SettingKey: "login.source_lock_minutes", SettingValue: "15", ValueType: "number", GroupKey: "login", Module: "system", IsPublic: 0, Remark: "system.setting.remark.login.source_lock_minutes"},
	{SettingKey: "login.security_event_enabled", SettingValue: "true", ValueType: "boolean", GroupKey: "login", Module: "system.auth", IsPublic: 0, Remark: "system.setting.remark.login.security_event_enabled"},
	{SettingKey: "login.captcha_enabled", SettingValue: "false", ValueType: "boolean", GroupKey: "login", Module: "system", IsPublic: 0, Remark: "system.setting.remark.login.captcha_enabled"},
	{SettingKey: "login.mfa_enabled", SettingValue: "false", ValueType: "boolean", GroupKey: "login", Module: "system", IsPublic: 0, Remark: "system.setting.remark.login.mfa_enabled"},
	{SettingKey: "login.sso_enabled", SettingValue: "false", ValueType: "boolean", GroupKey: "login", Module: "system", IsPublic: 0, Remark: "system.setting.remark.login.sso_enabled"},
	{SettingKey: "login.session_idle_minutes", SettingValue: "30", ValueType: "number", GroupKey: "login", Module: "system", IsPublic: 1, Remark: "system.setting.remark.login.session_idle_minutes"},
	{SettingKey: "login.max_active_sessions_per_user", SettingValue: "1", ValueType: "number", GroupKey: "login", Module: "system", IsPublic: 0, Remark: "system.setting.remark.login.max_active_sessions_per_user"},
	{SettingKey: "audit.login_log_retention_options", SettingValue: "[1,7,30]", ValueType: "json", GroupKey: "audit", Module: "system", IsPublic: 0, Remark: "system.setting.remark.audit.login_log_retention_options"},
	{SettingKey: "audit.operation_log_retention_options", SettingValue: "[1,7,30]", ValueType: "json", GroupKey: "audit", Module: "system", IsPublic: 0, Remark: "system.setting.remark.audit.operation_log_retention_options"},
	{SettingKey: "audit.session_cleanup_retention_options", SettingValue: "[1,7,30]", ValueType: "json", GroupKey: "audit", Module: "system", IsPublic: 0, Remark: "system.setting.remark.audit.session_cleanup_retention_options"},
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

var defaultSettingSeedMap = buildDefaultSettingSeedMap(defaultSettingSeeds)

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

func NewSettingService(db *gorm.DB) *SettingService {
	return &SettingService{
		db:         db,
		listCache:  make(map[string][]SettingResp),
		groupCache: make(map[string]*SettingGroupResp),
	}
}

func (s *SettingService) Migrate() error {
	if s.db == nil {
		return errors.New("database.not_initialized")
	}
	if err := s.db.AutoMigrate(&SystemSetting{}); err != nil {
		return err
	}

	for _, item := range defaultSettingSeeds {
		var count int64
		if err := s.db.Model(&SystemSetting{}).Where("setting_key = ?", item.SettingKey).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			continue
		}

		normalizedValue, err := normalizeSettingValue(item.SettingKey, item.SettingValue)
		if err != nil {
			return err
		}
		storedValue, err := prepareSettingStoredValue(normalizedValue, item.IsEncrypted)
		if err != nil {
			return err
		}
		if err := s.db.Create(&SystemSetting{
			SettingKey:   item.SettingKey,
			SettingValue: storedValue,
			ValueType:    item.ValueType,
			GroupKey:     item.GroupKey,
			Module:       item.Module,
			IsPublic:     item.IsPublic,
			IsEncrypted:  item.IsEncrypted,
			Remark:       item.Remark,
		}).Error; err != nil {
			return err
		}
	}

	if err := s.normalizeLegacySettingValue("ui.default_theme"); err != nil {
		return err
	}
	if err := s.normalizeLegacySettingValue("upload.storage_driver"); err != nil {
		return err
	}
	if err := s.migrateLegacySettingValue("upload.allowed_types", "[\"jpg\",\"jpeg\",\"png\",\"pdf\",\"doc\",\"docx\",\"xls\",\"xlsx\"]", "[\"jpg\",\"jpeg\",\"png\",\"pdf\",\"doc\",\"docx\",\"xls\",\"xlsx\",\"zip\",\"gz\",\"tgz\",\"tar\"]"); err != nil {
		return err
	}
	if err := s.migrateLegacySettingValue("audit.session_cleanup_retention_options", "[7,30,90]", "[1,7,30]"); err != nil {
		return err
	}

	return nil
}

func (s *SettingService) List(query *SettingListQuery) ([]SettingResp, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}

	groupKey := ""
	module := ""
	if query != nil {
		groupKey = strings.TrimSpace(query.GroupKey)
		module = strings.TrimSpace(query.Module)
	}

	cacheKey := settingListCacheKey(groupKey, module)
	s.cacheMu.RLock()
	if cached, ok := s.listCache[cacheKey]; ok {
		s.cacheMu.RUnlock()
		return cloneSettingRespList(cached), nil
	}
	s.cacheMu.RUnlock()

	var rows []SystemSetting
	db := s.db.Model(&SystemSetting{})
	if groupKey != "" {
		db = db.Where("group_key = ?", groupKey)
	}
	if module != "" {
		db = db.Where("module = ?", module)
	}
	if err := db.Order("group_key asc, id asc").Find(&rows).Error; err != nil {
		return nil, err
	}

	result := make([]SettingResp, 0, len(rows))
	for _, row := range rows {
		result = append(result, toSettingResp(row))
	}

	s.cacheMu.Lock()
	s.listCache[cacheKey] = cloneSettingRespList(result)
	s.cacheMu.Unlock()
	return cloneSettingRespList(result), nil
}

func (s *SettingService) GetGroup(groupKey string) (*SettingGroupResp, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}

	groupKey = strings.TrimSpace(groupKey)
	if groupKey == "" {
		return nil, errors.New("setting.group.invalid")
	}

	s.cacheMu.RLock()
	if cached, ok := s.groupCache[groupKey]; ok {
		s.cacheMu.RUnlock()
		return cloneSettingGroupResp(cached), nil
	}
	s.cacheMu.RUnlock()

	items, err := s.List(&SettingListQuery{GroupKey: groupKey})
	if err != nil {
		return nil, err
	}

	group := &SettingGroupResp{GroupKey: groupKey, Items: items}
	s.cacheMu.Lock()
	s.groupCache[groupKey] = cloneSettingGroupResp(group)
	s.cacheMu.Unlock()
	return cloneSettingGroupResp(group), nil
}

func (s *SettingService) GetByKey(settingKey string) (string, error) {
	if s.db == nil {
		return "", errors.New("database.not_initialized")
	}

	var row SystemSetting
	if err := s.db.Where("setting_key = ?", strings.TrimSpace(settingKey)).First(&row).Error; err != nil {
		return "", err
	}
	if row.IsEncrypted == 1 {
		return decryptSettingValue(row.SettingValue)
	}
	return row.SettingValue, nil
}

func (s *SettingService) UpdateGroup(groupKey string, req *SettingGroupUpdateReq) (*SettingGroupResp, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}

	groupKey = strings.TrimSpace(groupKey)
	if groupKey == "" {
		return nil, errors.New("setting.group.invalid")
	}
	if req == nil || len(req.Items) == 0 {
		return nil, errors.New("param.invalid")
	}

	if err := s.db.Transaction(func(tx *gorm.DB) error {
		for _, item := range req.Items {
			settingKey := strings.TrimSpace(item.SettingKey)
			if settingKey == "" {
				return errors.New("setting.key.required")
			}

			var current SystemSetting
			if err := tx.Where("setting_key = ? AND group_key = ?", settingKey, groupKey).First(&current).Error; err != nil {
				return err
			}

			nextValue := strings.TrimSpace(item.SettingValue)
			if current.IsEncrypted == 1 && nextValue == "" {
				continue
			}

			normalizedValue, err := validateAndNormalizeSettingValue(current.SettingKey, current.ValueType, nextValue)
			if err != nil {
				return err
			}
			storedValue, err := prepareSettingStoredValue(normalizedValue, current.IsEncrypted)
			if err != nil {
				return err
			}
			if err := tx.Model(&current).Update("setting_value", storedValue).Error; err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}

	s.invalidateSettingCache()
	if err := s.notifyRuntimeSettingsChanged(); err != nil {
		return nil, err
	}
	return s.GetGroup(groupKey)
}

func (s *SettingService) BuildAuditPayload(groupKey string, req *SettingGroupUpdateReq, includeOld bool) (string, error) {
	if s.db == nil || req == nil || len(req.Items) == 0 {
		return "", nil
	}

	type auditPayload struct {
		GroupKey string                   `json:"groupKey"`
		Changes  []SettingAuditChangeResp `json:"changes"`
	}

	keys := make([]string, 0, len(req.Items))
	requestValueMap := make(map[string]string, len(req.Items))
	for _, item := range req.Items {
		settingKey := strings.TrimSpace(item.SettingKey)
		if settingKey == "" {
			continue
		}
		keys = append(keys, settingKey)
		requestValueMap[settingKey] = strings.TrimSpace(item.SettingValue)
	}
	if len(keys) == 0 {
		return "", nil
	}

	var rows []SystemSetting
	if err := s.db.Where("group_key = ? AND setting_key IN ?", strings.TrimSpace(groupKey), keys).Find(&rows).Error; err != nil {
		return "", err
	}

	payload := auditPayload{
		GroupKey: strings.TrimSpace(groupKey),
		Changes:  make([]SettingAuditChangeResp, 0, len(rows)),
	}

	for _, row := range rows {
		rawNewValue := requestValueMap[row.SettingKey]
		if row.IsEncrypted == 1 {
			if includeOld && strings.TrimSpace(rawNewValue) == "" {
				continue
			}
			change := SettingAuditChangeResp{
				SettingKey:  row.SettingKey,
				IsEncrypted: row.IsEncrypted,
			}
			if includeOld && strings.TrimSpace(row.SettingValue) != "" {
				change.OldValue = "***"
			}
			if strings.TrimSpace(rawNewValue) != "" {
				change.NewValue = "***"
			}
			payload.Changes = append(payload.Changes, change)
			continue
		}

		normalizedNewValue, err := normalizeSettingValue(row.SettingKey, rawNewValue)
		if err != nil {
			return "", err
		}
		if includeOld && row.SettingValue == normalizedNewValue {
			continue
		}

		change := SettingAuditChangeResp{
			SettingKey:  row.SettingKey,
			IsEncrypted: row.IsEncrypted,
			NewValue:    normalizedNewValue,
		}
		if includeOld {
			change.OldValue = row.SettingValue
		}
		payload.Changes = append(payload.Changes, change)
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (s *SettingService) GetPublicSettings() (*PublicSettingResp, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}

	s.cacheMu.RLock()
	if s.publicCache != nil {
		s.cacheMu.RUnlock()
		return clonePublicSettingResp(s.publicCache), nil
	}
	s.cacheMu.RUnlock()

	var rows []SystemSetting
	if err := s.db.Model(&SystemSetting{}).Where("is_public = ? AND is_encrypted = ?", 1, 0).Order("id asc").Find(&rows).Error; err != nil {
		return nil, err
	}

	settings := make(map[string]string, len(rows))
	for _, row := range rows {
		settings[row.SettingKey] = row.SettingValue
	}

	resp := &PublicSettingResp{Settings: settings}
	s.cacheMu.Lock()
	s.publicCache = clonePublicSettingResp(resp)
	s.cacheMu.Unlock()
	return clonePublicSettingResp(resp), nil
}

func (s *SettingService) GetOverview() (*SettingOverviewResp, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}

	var rows []SystemSetting
	if err := s.db.Order("group_key asc, id asc").Find(&rows).Error; err != nil {
		return nil, err
	}

	resp := &SettingOverviewResp{
		Issues: make([]SettingOverviewIssueResp, 0),
	}
	byKey := make(map[string]SystemSetting, len(rows))
	for _, row := range rows {
		resp.TotalSettingCount++
		if row.IsPublic == 1 {
			resp.PublicSettingCount++
		}
		if row.IsEncrypted == 1 {
			resp.EncryptedSettingCount++
		}
		byKey[row.SettingKey] = row
	}

	resp.StorageDriver = safeSettingOverviewValue(byKey["upload.storage_driver"], "local")
	resp.DefaultLanguage = safeSettingOverviewValue(byKey["i18n.default_language"], "zh-CN")
	resp.DefaultTheme = safeSettingOverviewValue(byKey["ui.default_theme"], "indigo")

	requiredKeys := []string{
		"site.name",
		"platform.app_mode",
		"org.enabled",
		"org.required_for_user",
		"security.password_min_length",
		"security.password_require_digit",
		"security.password_require_uppercase",
		"security.password_history_limit",
		"security.password_expire_days",
		"login.max_failed_attempts",
		"login.lock_minutes",
		"login.source_max_failed_attempts",
		"login.source_window_minutes",
		"login.source_lock_minutes",
		"login.security_event_enabled",
		"login.captcha_enabled",
		"login.mfa_enabled",
		"login.sso_enabled",
		"login.session_idle_minutes",
		"login.max_active_sessions_per_user",
		"audit.session_retention_days",
		"i18n.default_language",
		"ui.default_theme",
		"ui.enable_tab_bar",
		"upload.storage_driver",
		"upload.max_file_size",
		"upload.allowed_types",
	}
	if resp.StorageDriver == "s3" {
		requiredKeys = append(requiredKeys,
			"upload.s3_endpoint",
			"upload.s3_bucket",
			"upload.s3_access_key_id",
			"upload.s3_secret_access_key",
		)
	} else {
		requiredKeys = append(requiredKeys, "upload.local_path")
	}

	seenIssues := make(map[string]struct{})
	for _, settingKey := range requiredKeys {
		row, ok := byKey[settingKey]
		if !ok || !systemSettingHasValue(row) {
			resp.RequiredMissingCount++
			resp.Issues = appendSettingOverviewIssue(resp.Issues, seenIssues, SettingOverviewIssueResp{
				SettingKey: settingKey,
				GroupKey:   inferSettingGroupKey(settingKey),
				Severity:   "warning",
				ReasonKey:  "setting.overview.issue.required_missing",
			})
		}
	}

	for _, row := range rows {
		if row.IsPublic == 1 && row.IsEncrypted == 1 {
			resp.Issues = appendSettingOverviewIssue(resp.Issues, seenIssues, SettingOverviewIssueResp{
				SettingKey: row.SettingKey,
				GroupKey:   row.GroupKey,
				Severity:   "critical",
				ReasonKey:  "setting.overview.issue.public_encrypted_conflict",
			})
		}
	}

	if _, ok := allowedStorageDriverValues[resp.StorageDriver]; !ok {
		resp.Issues = appendSettingOverviewIssue(resp.Issues, seenIssues, SettingOverviewIssueResp{
			SettingKey: "upload.storage_driver",
			GroupKey:   "upload",
			Severity:   "critical",
			ReasonKey:  "setting.overview.issue.invalid_storage_driver",
		})
	}
	if _, ok := allowedLanguageValues[resp.DefaultLanguage]; !ok {
		resp.Issues = appendSettingOverviewIssue(resp.Issues, seenIssues, SettingOverviewIssueResp{
			SettingKey: "i18n.default_language",
			GroupKey:   "i18n",
			Severity:   "warning",
			ReasonKey:  "setting.overview.issue.invalid_default_language",
		})
	}
	if _, ok := allowedThemeValues[resp.DefaultTheme]; !ok {
		resp.Issues = appendSettingOverviewIssue(resp.Issues, seenIssues, SettingOverviewIssueResp{
			SettingKey: "ui.default_theme",
			GroupKey:   "ui",
			Severity:   "warning",
			ReasonKey:  "setting.overview.issue.invalid_default_theme",
		})
	}
	appMode := safeSettingOverviewValue(byKey["platform.app_mode"], "enterprise")
	if _, ok := allowedAppModeValues[appMode]; !ok {
		resp.Issues = appendSettingOverviewIssue(resp.Issues, seenIssues, SettingOverviewIssueResp{
			SettingKey: "platform.app_mode",
			GroupKey:   "platform",
			Severity:   "warning",
			ReasonKey:  "setting.overview.issue.invalid_app_mode",
		})
	}

	resp.RiskCount = len(resp.Issues)
	return resp, nil
}

func (s *SettingService) RefreshSettingCache(groupKeys []string) (*SettingCacheRefreshResp, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}

	normalizedGroups := normalizeSettingGroups(groupKeys)
	if len(normalizedGroups) == 0 {
		s.invalidateSettingCache()
		if err := s.notifyRuntimeSettingsChanged(); err != nil {
			return nil, err
		}
		return &SettingCacheRefreshResp{
			RefreshedGroups: []string{},
			ClearedAll:      1,
		}, nil
	}

	s.invalidateSettingCache()
	if err := s.notifyRuntimeSettingsChanged(); err != nil {
		return nil, err
	}
	for _, groupKey := range normalizedGroups {
		if _, err := s.GetGroup(groupKey); err != nil {
			return nil, err
		}
	}
	if _, err := s.GetPublicSettings(); err != nil {
		return nil, err
	}

	return &SettingCacheRefreshResp{
		RefreshedGroups: normalizedGroups,
		ClearedAll:      0,
	}, nil
}

func (s *SettingService) ListAudit(query *SettingAuditQuery) (*SettingAuditPageResp, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}

	page := 1
	pageSize := 10
	if query != nil {
		if query.Page > 0 {
			page = query.Page
		}
		if query.PageSize > 0 && query.PageSize <= 100 {
			pageSize = query.PageSize
		}
	}

	db := s.db.Model(&systemSettingAuditLog{}).Where("title = ?", settingAuditTitle)
	if query != nil {
		if strings.TrimSpace(query.OperName) != "" {
			db = db.Where("oper_name LIKE ?", "%"+strings.TrimSpace(query.OperName)+"%")
		}
		if strings.TrimSpace(query.GroupKey) != "" {
			db = db.Where("oper_param LIKE ?", "%\"groupKey\":\""+strings.TrimSpace(query.GroupKey)+"\"%")
		}
		if strings.TrimSpace(query.SettingKey) != "" {
			db = db.Where("oper_param LIKE ?", "%\"settingKey\":\""+strings.TrimSpace(query.SettingKey)+"\"%")
		}
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}

	var rows []systemSettingAuditLog
	if err := db.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
		return nil, err
	}

	items := make([]SettingAuditResp, 0, len(rows))
	for _, row := range rows {
		groupKey, changes := parseSettingAuditPayload(row.OperParam)
		items = append(items, SettingAuditResp{
			ID:       row.ID,
			GroupKey: groupKey,
			OperName: row.OperName,
			OperIP:   row.OperIP,
			Status:   row.Status,
			ErrorMsg: row.ErrorMsg,
			OperTime: row.OperTime.Format(time.RFC3339),
			CostTime: row.CostTime,
			Changes:  changes,
		})
	}

	return &SettingAuditPageResp{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (s *SettingService) ExportAudit(query *SettingAuditQuery) (*impexp.CSVFile, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}

	db := s.db.Model(&systemSettingAuditLog{}).Where("title = ?", settingAuditTitle)
	if query != nil {
		if strings.TrimSpace(query.OperName) != "" {
			db = db.Where("oper_name LIKE ?", "%"+strings.TrimSpace(query.OperName)+"%")
		}
		if strings.TrimSpace(query.GroupKey) != "" {
			db = db.Where("oper_param LIKE ?", "%\"groupKey\":\""+strings.TrimSpace(query.GroupKey)+"\"%")
		}
		if strings.TrimSpace(query.SettingKey) != "" {
			db = db.Where("oper_param LIKE ?", "%\"settingKey\":\""+strings.TrimSpace(query.SettingKey)+"\"%")
		}
	}

	var rows []systemSettingAuditLog
	if err := db.Order("id desc").Find(&rows).Error; err != nil {
		return nil, err
	}

	result := make([][]string, 0, len(rows))
	for _, row := range rows {
		groupKey, changes := parseSettingAuditPayload(row.OperParam)
		result = append(result, []string{
			groupKey,
			row.OperName,
			row.OperIP,
			formatSettingAuditChanges(changes),
			strconv.Itoa(row.Status),
			row.ErrorMsg,
			row.OperTime.Format(time.RFC3339),
			strconv.FormatInt(row.CostTime, 10),
		})
	}

	return &impexp.CSVFile{
		Filename: "system-setting-audit-export.csv",
		Headers:  []string{"groupKey", "operName", "operIp", "changes", "status", "errorMsg", "operTime", "costTime"},
		Rows:     result,
	}, nil
}

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

func (s *SettingService) migrateLegacySettingValue(settingKey string, legacyValue string, nextValue string) error {
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

func settingListCacheKey(groupKey string, module string) string {
	return strings.TrimSpace(groupKey) + "|" + strings.TrimSpace(module)
}

func cloneSettingRespList(items []SettingResp) []SettingResp {
	if len(items) == 0 {
		return []SettingResp{}
	}
	result := make([]SettingResp, len(items))
	copy(result, items)
	return result
}

func cloneSettingGroupResp(resp *SettingGroupResp) *SettingGroupResp {
	if resp == nil {
		return nil
	}
	return &SettingGroupResp{
		GroupKey: resp.GroupKey,
		Items:    cloneSettingRespList(resp.Items),
	}
}

func clonePublicSettingResp(resp *PublicSettingResp) *PublicSettingResp {
	if resp == nil {
		return nil
	}
	settings := make(map[string]string, len(resp.Settings))
	for key, value := range resp.Settings {
		settings[key] = value
	}
	return &PublicSettingResp{Settings: settings}
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

func (s *SettingService) notifyRuntimeSettingsChanged() error {
	if err := contracts.NotifyRuntimeSettingsChanged(); err != nil {
		return err
	}
	if database.RDB != nil {
		_ = database.RDB.Publish(context.Background(), "settings:refresh", "updated").Err()
	}
	return nil
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

func validateAndNormalizeSettingValue(settingKey string, valueType string, value string) (string, error) {
	normalizedValue, err := normalizeSettingValue(settingKey, value)
	if err != nil {
		return "", err
	}

	switch strings.TrimSpace(valueType) {
	case "string":
		return normalizedValue, nil
	case "number":
		if _, err := strconv.ParseFloat(strings.TrimSpace(normalizedValue), 64); err != nil {
			return "", errors.New("setting.value.invalid_number")
		}
		return normalizedValue, nil
	case "boolean":
		if _, err := strconv.ParseBool(strings.TrimSpace(normalizedValue)); err != nil {
			return "", errors.New("setting.value.invalid_boolean")
		}
		return normalizedValue, nil
	case "json":
		trimmed := strings.TrimSpace(normalizedValue)
		if trimmed == "" {
			return normalizedValue, nil
		}
		var target interface{}
		if err := json.Unmarshal([]byte(trimmed), &target); err != nil {
			return "", errors.New("setting.value.invalid_json")
		}
		if settingKey == "upload.allowed_types" {
			if _, ok := target.([]interface{}); !ok {
				return "", errors.New("setting.value.invalid_json")
			}
		}
		if settingKey == "audit.login_log_retention_options" || settingKey == "audit.operation_log_retention_options" || settingKey == "audit.session_cleanup_retention_options" {
			normalizedJSON, err := normalizeAuditRetentionOptions(trimmed)
			if err != nil {
				return "", err
			}
			return normalizedJSON, nil
		}
		return trimmed, nil
	default:
		return "", errors.New("setting.value_type.invalid")
	}
}

func normalizeSettingValue(settingKey string, value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	switch settingKey {
	case "platform.app_mode":
		if trimmed == "" {
			trimmed = "enterprise"
		}
		if _, ok := allowedAppModeValues[trimmed]; !ok {
			return "", errors.New("setting.value.invalid_option")
		}
	case "upload.storage_driver":
		switch trimmed {
		case "s3-compatible":
			trimmed = "s3"
		case "":
			trimmed = "local"
		}
		if _, ok := allowedStorageDriverValues[trimmed]; !ok {
			return "", errors.New("setting.value.invalid_option")
		}
	case "ui.default_theme":
		switch trimmed {
		case "light":
			trimmed = "indigo"
		case "":
			trimmed = "indigo"
		}
		if _, ok := allowedThemeValues[trimmed]; !ok {
			return "", errors.New("setting.value.invalid_option")
		}
	case "i18n.default_language":
		if trimmed == "" {
			trimmed = "zh-CN"
		}
		if _, ok := allowedLanguageValues[trimmed]; !ok {
			return "", errors.New("setting.value.invalid_option")
		}
	}
	return trimmed, nil
}

func normalizeAuditRetentionOptions(raw string) (string, error) {
	var values []int
	if err := json.Unmarshal([]byte(strings.TrimSpace(raw)), &values); err != nil {
		return "", errors.New("setting.value.invalid_json")
	}
	if len(values) == 0 {
		return "", errors.New("setting.value.invalid_option")
	}

	seen := make(map[int]struct{}, len(values))
	normalized := make([]int, 0, len(values))
	for _, value := range values {
		if value <= 0 || value > 365 {
			return "", errors.New("setting.value.invalid_option")
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		normalized = append(normalized, value)
	}
	if len(normalized) == 0 {
		return "", errors.New("setting.value.invalid_option")
	}

	sort.Ints(normalized)
	normalizedJSON, err := json.Marshal(normalized)
	if err != nil {
		return "", errors.New("setting.value.invalid_json")
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

func defaultSettingValue(settingKey string) string {
	seed, ok := defaultSettingSeedMap[settingKey]
	if !ok {
		return ""
	}
	return seed.SettingValue
}

func parseSettingAuditPayload(raw string) (string, []SettingAuditChangeResp) {
	var payload struct {
		GroupKey string                   `json:"groupKey"`
		Changes  []SettingAuditChangeResp `json:"changes"`
	}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return "", []SettingAuditChangeResp{}
	}
	return payload.GroupKey, payload.Changes
}

func formatSettingAuditChanges(changes []SettingAuditChangeResp) string {
	if len(changes) == 0 {
		return ""
	}
	parts := make([]string, 0, len(changes))
	for _, change := range changes {
		if change.IsEncrypted == 1 {
			parts = append(parts, change.SettingKey+":***->***")
			continue
		}
		parts = append(parts, change.SettingKey+":"+change.OldValue+"->"+change.NewValue)
	}
	return strings.Join(parts, " | ")
}

func toSettingResp(item SystemSetting) SettingResp {
	hasValue := 0
	if strings.TrimSpace(item.SettingValue) != "" {
		hasValue = 1
	}

	displayValue := item.SettingValue
	if item.IsEncrypted == 1 {
		displayValue = ""
	}

	return SettingResp{
		ID:           item.ID,
		SettingKey:   item.SettingKey,
		SettingValue: displayValue,
		DefaultValue: defaultSettingValue(item.SettingKey),
		ValueType:    item.ValueType,
		GroupKey:     item.GroupKey,
		Module:       item.Module,
		IsPublic:     item.IsPublic,
		IsEncrypted:  item.IsEncrypted,
		HasValue:     hasValue,
		Remark:       item.Remark,
		CreatedAt:    item.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    item.UpdatedAt.Format(time.RFC3339),
	}
}
