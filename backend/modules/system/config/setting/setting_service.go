package config

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"pantheon-ops/backend/pkg/common"
	"pantheon-ops/backend/pkg/contracts"
	"pantheon-ops/backend/pkg/database"

	"gorm.io/gorm"
)

const (
	settingKeyUITheme      = "ui.default_theme"
	settingKeyUploadDriver = "upload.storage_driver"
	settingKeyI18nLanguage = "i18n.default_language"
	settingKeyAppMode      = "platform.app_mode"
)

type SettingService struct {
	db          *gorm.DB
	cacheMu     sync.RWMutex
	listCache   map[string][]SettingResp
	groupCache  map[string]*SettingGroupResp
	publicCache *PublicSettingResp
}

func NewSettingService(db *gorm.DB) *SettingService {
	return &SettingService{
		db:         db,
		listCache:  make(map[string][]SettingResp),
		groupCache: make(map[string]*SettingGroupResp),
	}
}

func (s *SettingService) Migrate() error {
	if s.db == nil {
		return common.ErrDatabaseNotInitialized
	}
	if err := s.db.AutoMigrate(&SystemSetting{}); err != nil {
		return err
	}
	return s.Bootstrap()
}

func (s *SettingService) Bootstrap() error {
	if s.db == nil {
		return common.ErrDatabaseNotInitialized
	}
	for _, item := range settingSeeds() {
		if err := s.bootstrapSettingSeed(item); err != nil {
			return err
		}
	}

	if err := s.normalizeLegacySettingValue(settingKeyUITheme); err != nil {
		return err
	}
	if err := s.normalizeLegacySettingValue(settingKeyUploadDriver); err != nil {
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

func (s *SettingService) bootstrapSettingSeed(item defaultSettingSeed) error {
	var count int64
	if err := s.db.Model(&SystemSetting{}).Where("setting_key = ?", item.SettingKey).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	normalizedValue, err := normalizeSettingValue(item.SettingKey, item.SettingValue)
	if err != nil {
		return err
	}
	storedValue, err := prepareSettingStoredValue(normalizedValue, item.IsEncrypted)
	if err != nil {
		return err
	}
	return s.db.Create(&SystemSetting{
		SettingKey:   item.SettingKey,
		SettingValue: storedValue,
		ValueType:    item.ValueType,
		GroupKey:     item.GroupKey,
		Module:       item.Module,
		IsPublic:     item.IsPublic,
		IsEncrypted:  item.IsEncrypted,
		Remark:       item.Remark,
	}).Error
}

func (s *SettingService) List(query *SettingListQuery) ([]SettingResp, error) {
	if s.db == nil {
		return nil, common.ErrDatabaseNotInitialized
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
		return nil, common.ErrDatabaseNotInitialized
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
		return "", common.ErrDatabaseNotInitialized
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
		return nil, common.ErrDatabaseNotInitialized
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

	s.invalidateSettingCacheForGroup(groupKey)
	if err := s.notifyRuntimeSettingsChanged(); err != nil {
		return nil, err
	}
	return s.GetGroup(groupKey)
}

func (s *SettingService) GetPublicSettings() (*PublicSettingResp, error) {
	if s.db == nil {
		return nil, common.ErrDatabaseNotInitialized
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
		return nil, common.ErrDatabaseNotInitialized
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

	resp.StorageDriver = safeSettingOverviewValue(byKey[settingKeyUploadDriver], "local")
	resp.DefaultLanguage = safeSettingOverviewValue(byKey[settingKeyI18nLanguage], "zh-CN")
	resp.DefaultTheme = safeSettingOverviewValue(byKey[settingKeyUITheme], "indigo")

	requiredKeys := []string{
		"site.name",
		settingKeyAppMode,
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
		settingKeyI18nLanguage,
		settingKeyUITheme,
		"ui.enable_tab_bar",
		settingKeyUploadDriver,
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
			SettingKey: settingKeyUploadDriver,
			GroupKey:   "upload",
			Severity:   "critical",
			ReasonKey:  "setting.overview.issue.invalid_storage_driver",
		})
	}
	if _, ok := allowedLanguageValues[resp.DefaultLanguage]; !ok {
		resp.Issues = appendSettingOverviewIssue(resp.Issues, seenIssues, SettingOverviewIssueResp{
			SettingKey: settingKeyI18nLanguage,
			GroupKey:   "i18n",
			Severity:   "warning",
			ReasonKey:  "setting.overview.issue.invalid_default_language",
		})
	}
	if _, ok := allowedThemeValues[resp.DefaultTheme]; !ok {
		resp.Issues = appendSettingOverviewIssue(resp.Issues, seenIssues, SettingOverviewIssueResp{
			SettingKey: settingKeyUITheme,
			GroupKey:   "ui",
			Severity:   "warning",
			ReasonKey:  "setting.overview.issue.invalid_default_theme",
		})
	}
	appMode := safeSettingOverviewValue(byKey[settingKeyAppMode], "enterprise")
	if _, ok := allowedAppModeValues[appMode]; !ok {
		resp.Issues = appendSettingOverviewIssue(resp.Issues, seenIssues, SettingOverviewIssueResp{
			SettingKey: settingKeyAppMode,
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
		return nil, common.ErrDatabaseNotInitialized
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

	for _, gk := range normalizedGroups {
		s.invalidateSettingCacheForGroup(gk)
	}
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

func settingListCacheKey(groupKey, module string) string {
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

func (s *SettingService) notifyRuntimeSettingsChanged() error {
	if err := contracts.NotifyRuntimeSettingsChanged(); err != nil {
		return err
	}
	if database.RDB != nil {
		_ = database.RDB.Publish(context.TODO(), "settings:refresh", "updated").Err()
	}
	return nil
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
