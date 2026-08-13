package system

import (
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"

	"pantheon-base/pkg/common"

	"gorm.io/gorm"
)

const (
	orderByModuleASC    = "module ASC"
	orderByGroupNameASC = "group_name ASC"
	orderByKeyASC       = "`key` ASC"
	orderByLocaleASC    = "locale ASC"
	condModuleEquals    = "module = ?"
	condModuleKeyEquals = "module = ? AND `key` = ?"
	condLocaleKeyEquals = "locale = ? AND `key` = ?"
)

func isI18nPlaceholderValue(value string) bool {
	trimmed := strings.TrimSpace(value)
	return strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]")
}

func hasStoredLocaleValue(value string) bool {
	trimmed := strings.TrimSpace(value)
	return trimmed != "" && !isI18nPlaceholderValue(trimmed)
}

func hasEffectiveLocaleValue(locale, key, value string) bool {
	if hasStoredLocaleValue(value) {
		return true
	}
	_, ok := getBuiltinLocaleValue(locale, key)
	return ok
}

func (s *I18nService) ScanErrorKeys() ([]string, error) {
	return scanI18nKeys(true)
}

func (s *I18nService) SyncMissingKeys() (*I18nSyncResp, error) {
	keys, err := s.ScanErrorKeys()
	if err != nil {
		return nil, err
	}
	resp := &I18nSyncResp{Keys: []string{}}
	supportedLocales, err := s.ListSupportedLocales()
	if err != nil {
		return nil, err
	}
	for _, k := range keys {
		createdForKey, err := s.syncMissingKeyForLocales(k, supportedLocales)
		if err != nil {
			return resp, err
		}
		if createdForKey {
			resp.Count++
			resp.Keys = append(resp.Keys, k)
		}
	}
	return resp, s.ReloadCache()
}

// syncMissingKeyForLocales 为单个 key 在所有受支持语言下补齐缺失的 i18n 记录：
// 仅当 (key, locale) 不存在时才写入（跳过已存在 key），写入值优先取内置语言值。
// 该逻辑为原 SyncMissingKeys 内层双重循环的等价提取，未改变写入语义与事务边界。
func (s *I18nService) syncMissingKeyForLocales(k string, supportedLocales []string) (bool, error) {
	created := false
	for _, locale := range supportedLocales {
		var exists int64
		if err := s.db.Model(&SystemI18n{}).Where("`key` = ? AND locale = ?", k, locale).Count(&exists).Error; err != nil {
			return false, err
		}
		if exists > 0 {
			continue
		}
		value := "[" + k + "]"
		if builtinValue, ok := getBuiltinLocaleValue(locale, k); ok {
			value = builtinValue
		}
		if err := s.db.Create(&SystemI18n{
			Module: "system.config",
			Group:  "messages",
			Key:    k,
			Locale: locale,
			Value:  value,
		}).Error; err != nil {
			return false, err
		}
		created = true
	}
	return created, nil
}

type i18nAuditRow struct {
	ID                uint64
	Module            string
	Group             string
	Key               string
	Locale            string
	Value             string
	LifecycleStatus   string
	LifecycleMarkedAt *time.Time
	UpdatedAt         time.Time
}

type i18nKeyAudit struct {
	modules map[string]struct{}
	groups  map[string]struct{}
	locales map[string]struct{}
	values  map[string]struct{}
	rows    int64
}

type i18nModuleAudit struct {
	entryCount         int64
	keys               map[string]struct{}
	unusedKeys         map[string]struct{}
	duplicateKeys      map[string]struct{}
	missingLocaleKeys  map[string]struct{}
	placeholderCount   int64
	stalePlaceholders  int64
	observingKeys      map[string]struct{}
	archivedKeys       map[string]struct{}
	deleteEligibleKeys map[string]struct{}
}

type i18nUnusedKeyAudit struct {
	module            string
	key               string
	groups            map[string]struct{}
	locales           map[string]struct{}
	values            map[string]struct{}
	lifecycleStatus   string
	lifecycleMarkedAt *time.Time
}

type i18nStalePlaceholderParams struct {
	moduleMeta *i18nModuleAudit
	resp       *I18nAuditResp
	item       i18nAuditRow
	module     string
	group      string
	key        string
	locale     string
	value      string
	now        time.Time
}

func (s *I18nService) loadI18nAuditRows() ([]i18nAuditRow, error) {
	var rows []i18nAuditRow
	if err := s.db.Model(&SystemI18n{}).
		Select("id, module, group_name as `group`, `key`, locale, value, lifecycle_status, lifecycle_marked_at, updated_at").
		Order(orderByModuleASC).
		Order(orderByGroupNameASC).
		Order(orderByKeyASC).
		Order(orderByLocaleASC).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (s *I18nService) collectI18nUsedKeys() (map[string]struct{}, error) {
	usedKeys, err := scanI18nKeys(true)
	if err != nil {
		return nil, err
	}
	usedSet := make(map[string]struct{}, len(usedKeys))
	for _, key := range usedKeys {
		usedSet[key] = struct{}{}
	}
	return usedSet, nil
}

func (s *I18nService) ensureI18nModuleAudit(moduleAudits map[string]*i18nModuleAudit, module string) *i18nModuleAudit {
	meta, ok := moduleAudits[module]
	if !ok {
		meta = &i18nModuleAudit{
			keys:               make(map[string]struct{}),
			unusedKeys:         make(map[string]struct{}),
			duplicateKeys:      make(map[string]struct{}),
			missingLocaleKeys:  make(map[string]struct{}),
			observingKeys:      make(map[string]struct{}),
			archivedKeys:       make(map[string]struct{}),
			deleteEligibleKeys: make(map[string]struct{}),
		}
		moduleAudits[module] = meta
	}
	return meta
}

func (s *I18nService) trackI18nUnusedKey(unusedKeyAudits map[string]*i18nUnusedKeyAudit, module, key, group, locale, value string, item i18nAuditRow) {
	unusedCompositeKey := module + "|" + key
	unusedMeta, exists := unusedKeyAudits[unusedCompositeKey]
	if !exists {
		unusedMeta = &i18nUnusedKeyAudit{
			module:            module,
			key:               key,
			groups:            make(map[string]struct{}),
			locales:           make(map[string]struct{}),
			values:            make(map[string]struct{}),
			lifecycleStatus:   normalizeI18nLifecycleStatus(item.LifecycleStatus),
			lifecycleMarkedAt: item.LifecycleMarkedAt,
		}
		unusedKeyAudits[unusedCompositeKey] = unusedMeta
	}
	if group != "" {
		unusedMeta.groups[group] = struct{}{}
	}
	if locale != "" {
		unusedMeta.locales[locale] = struct{}{}
	}
	if value != "" {
		unusedMeta.values[value] = struct{}{}
	}
}

func (s *I18nService) processI18nAuditRows(rows []i18nAuditRow, now time.Time, resp *I18nAuditResp) (map[string]*i18nKeyAudit, map[string]*i18nModuleAudit, map[string]*i18nUnusedKeyAudit) {
	keyAudits := make(map[string]*i18nKeyAudit)
	moduleAudits := make(map[string]*i18nModuleAudit)
	unusedKeyAudits := make(map[string]*i18nUnusedKeyAudit)
	for _, item := range rows {
		key := strings.TrimSpace(item.Key)
		if key == "" {
			continue
		}
		module := strings.TrimSpace(item.Module)
		group := strings.TrimSpace(item.Group)
		locale := strings.TrimSpace(item.Locale)
		value := strings.TrimSpace(item.Value)

		keyMeta, ok := keyAudits[key]
		if !ok {
			keyMeta = &i18nKeyAudit{
				modules: make(map[string]struct{}),
				groups:  make(map[string]struct{}),
				locales: make(map[string]struct{}),
				values:  make(map[string]struct{}),
			}
			keyAudits[key] = keyMeta
		}
		keyMeta.rows++
		if module != "" {
			keyMeta.modules[module] = struct{}{}
		}
		if group != "" {
			keyMeta.groups[group] = struct{}{}
		}
		if locale != "" {
			keyMeta.locales[locale] = struct{}{}
		}
		if value != "" {
			keyMeta.values[value] = struct{}{}
		}

		moduleMeta := s.ensureI18nModuleAudit(moduleAudits, module)
		moduleMeta.entryCount++
		moduleMeta.keys[key] = struct{}{}
		if !hasEffectiveLocaleValue(locale, key, value) {
			moduleMeta.placeholderCount++
			s.trackI18nStalePlaceholder(i18nStalePlaceholderParams{
				moduleMeta: moduleMeta,
				resp:       resp,
				item:       item,
				module:     module,
				group:      group,
				key:        key,
				locale:     locale,
				value:      value,
				now:        now,
			})
		}

		s.trackI18nUnusedKey(unusedKeyAudits, module, key, group, locale, value, item)
	}
	return keyAudits, moduleAudits, unusedKeyAudits
}

func (s *I18nService) trackI18nStalePlaceholder(params i18nStalePlaceholderParams) {
	staleDays := int64(params.now.Sub(params.item.UpdatedAt).Hours() / 24)
	if staleDays >= I18nStalePlaceholderThresholdDays {
		params.moduleMeta.stalePlaceholders++
		params.resp.StalePlaceholders = append(params.resp.StalePlaceholders, I18nStalePlaceholderItem{
			ID:        params.item.ID,
			Module:    params.module,
			Group:     params.group,
			Key:       params.key,
			Locale:    params.locale,
			Value:     params.value,
			UpdatedAt: params.item.UpdatedAt.Format(time.RFC3339),
			StaleDays: staleDays,
		})
	}
}

func (s *I18nService) appendI18nDuplicateConflict(key string, meta *i18nKeyAudit, moduleAudits map[string]*i18nModuleAudit, resp *I18nAuditResp) {
	modules := sortedSetKeys(meta.modules)
	for _, module := range modules {
		moduleAudits[module].duplicateKeys[key] = struct{}{}
	}
	suggestions := make([]I18nRenameSuggestion, 0, len(modules))
	for _, module := range modules {
		suggestions = append(suggestions, I18nRenameSuggestion{
			Module:       module,
			SuggestedKey: suggestScopedI18nKey(module, key),
		})
	}
	resp.DuplicateKeys = append(resp.DuplicateKeys, I18nDuplicateKeyConflict{
		Key:         key,
		Modules:     modules,
		Groups:      sortedSetKeys(meta.groups),
		Locales:     sortedSetKeys(meta.locales),
		Values:      sortedSetKeys(meta.values),
		RowCount:    meta.rows,
		Suggestions: suggestions,
	})
}

func (s *I18nService) resolveI18nKeyConflicts(keyAudits map[string]*i18nKeyAudit, moduleAudits map[string]*i18nModuleAudit, locales []string, resp *I18nAuditResp) {
	for key, meta := range keyAudits {
		s.fillI18nBuiltinLocales(meta, locales, key)
		if len(meta.modules) > 1 || len(meta.groups) > 1 {
			s.appendI18nDuplicateConflict(key, meta, moduleAudits, resp)
		}
		if int64(len(meta.locales)) < int64(len(locales)) {
			for _, module := range sortedSetKeys(meta.modules) {
				moduleAudits[module].missingLocaleKeys[key] = struct{}{}
			}
		}
	}
}

func (s *I18nService) fillI18nBuiltinLocales(meta *i18nKeyAudit, locales []string, key string) {
	for _, locale := range locales {
		if _, ok := meta.locales[locale]; ok {
			continue
		}
		if _, builtinOk := getBuiltinLocaleValue(locale, key); builtinOk {
			meta.locales[locale] = struct{}{}
		}
	}
}

func (s *I18nService) appendI18nUnusedKeyItem(meta *i18nUnusedKeyAudit, moduleMeta *i18nModuleAudit, observingDays int64, markedAt string, resp *I18nAuditResp) {
	if meta.lifecycleStatus == I18nLifecycleStatusObserving {
		moduleMeta.observingKeys[meta.key] = struct{}{}
	}
	if meta.lifecycleStatus == I18nLifecycleStatusArchived {
		moduleMeta.archivedKeys[meta.key] = struct{}{}
	}
	eligibleForDelete := meta.lifecycleStatus == I18nLifecycleStatusArchived && observingDays >= I18nArchivedRetentionThresholdDays
	if eligibleForDelete {
		moduleMeta.deleteEligibleKeys[meta.key] = struct{}{}
	}
	resp.UnusedKeys = append(resp.UnusedKeys, I18nUnusedKeyItem{
		Key:                meta.key,
		Module:             meta.module,
		Modules:            []string{meta.module},
		Groups:             sortedSetKeys(meta.groups),
		Locales:            sortedSetKeys(meta.locales),
		Placeholder:        allValuesMissing(meta.values),
		LifecycleStatus:    meta.lifecycleStatus,
		LifecycleMarkedAt:  markedAt,
		ObservingDays:      observingDays,
		EligibleForArchive: meta.lifecycleStatus == I18nLifecycleStatusObserving && observingDays >= I18nUnusedObservationThresholdDays,
		EligibleForDelete:  eligibleForDelete,
	})
}

func (s *I18nService) resolveI18nUnusedKeys(unusedKeyAudits map[string]*i18nUnusedKeyAudit, moduleAudits map[string]*i18nModuleAudit, usedSet map[string]struct{}, now time.Time, resp *I18nAuditResp) {
	for compositeKey, meta := range unusedKeyAudits {
		if _, ok := usedSet[meta.key]; ok {
			s.maybeResetI18nLifecycle(compositeKey, meta)
			continue
		}
		moduleMeta := moduleAudits[meta.module]
		moduleMeta.unusedKeys[meta.key] = struct{}{}
		observingDays, markedAt := s.resolveI18nUnusedObservationDays(meta, now)
		s.appendI18nUnusedKeyItem(meta, moduleMeta, observingDays, markedAt, resp)
	}
}

func (s *I18nService) maybeResetI18nLifecycle(compositeKey string, meta *i18nUnusedKeyAudit) {
	if meta.lifecycleStatus != I18nLifecycleStatusActive {
		if err := s.resetI18nLifecycle(compositeKey, meta.module, meta.key); err == nil {
			meta.lifecycleStatus = I18nLifecycleStatusActive
			meta.lifecycleMarkedAt = nil
		}
	}
}

func (s *I18nService) resolveI18nUnusedObservationDays(meta *i18nUnusedKeyAudit, now time.Time) (int64, string) {
	observingDays := int64(0)
	markedAt := ""
	if meta.lifecycleMarkedAt != nil {
		markedAt = meta.lifecycleMarkedAt.Format(time.RFC3339)
		observingDays = int64(now.Sub(*meta.lifecycleMarkedAt).Hours() / 24)
	}
	return observingDays, markedAt
}

func (s *I18nService) buildI18nModuleAuditSummary(moduleAudits map[string]*i18nModuleAudit, resp *I18nAuditResp) {
	moduleNames := make([]string, 0, len(moduleAudits))
	for module := range moduleAudits {
		moduleNames = append(moduleNames, module)
	}
	sort.Strings(moduleNames)
	for _, module := range moduleNames {
		item := moduleAudits[module]
		resp.Modules = append(resp.Modules, I18nModuleAuditItem{
			Module:                 module,
			EntryCount:             item.entryCount,
			KeyCount:               int64(len(item.keys)),
			UnusedKeyCount:         int64(len(item.unusedKeys)),
			DuplicateKeyCount:      int64(len(item.duplicateKeys)),
			MissingLocaleCount:     int64(len(item.missingLocaleKeys)),
			PlaceholderCount:       item.placeholderCount,
			StalePlaceholderCount:  item.stalePlaceholders,
			ObservingKeyCount:      int64(len(item.observingKeys)),
			ArchivedKeyCount:       int64(len(item.archivedKeys)),
			DeleteEligibleKeyCount: int64(len(item.deleteEligibleKeys)),
		})
	}
}

func (s *I18nService) sortI18nAuditResults(resp *I18nAuditResp) {
	sort.Slice(resp.DuplicateKeys, func(i, j int) bool { return resp.DuplicateKeys[i].Key < resp.DuplicateKeys[j].Key })
	sort.Slice(resp.UnusedKeys, func(i, j int) bool { return resp.UnusedKeys[i].Key < resp.UnusedKeys[j].Key })
	sort.Slice(resp.StalePlaceholders, func(i, j int) bool {
		if resp.StalePlaceholders[i].StaleDays == resp.StalePlaceholders[j].StaleDays {
			if resp.StalePlaceholders[i].Key == resp.StalePlaceholders[j].Key {
				return resp.StalePlaceholders[i].Locale < resp.StalePlaceholders[j].Locale
			}
			return resp.StalePlaceholders[i].Key < resp.StalePlaceholders[j].Key
		}
		return resp.StalePlaceholders[i].StaleDays > resp.StalePlaceholders[j].StaleDays
	})
}

// GetAudit 汇总 i18n 词条治理审计结果：重复键、未使用键、过期占位符与模块统计。
func (s *I18nService) GetAudit() (*I18nAuditResp, error) {
	resp := &I18nAuditResp{
		DuplicateKeys:                  make([]I18nDuplicateKeyConflict, 0),
		UnusedKeys:                     make([]I18nUnusedKeyItem, 0),
		StalePlaceholders:              make([]I18nStalePlaceholderItem, 0),
		Modules:                        make([]I18nModuleAuditItem, 0),
		StalePlaceholderThresholdDays:  I18nStalePlaceholderThresholdDays,
		UnusedObservationThresholdDays: I18nUnusedObservationThresholdDays,
		ArchivedRetentionThresholdDays: I18nArchivedRetentionThresholdDays,
	}
	if s.db == nil {
		return resp, nil
	}

	rows, err := s.loadI18nAuditRows()
	if err != nil {
		return nil, err
	}
	usedSet, err := s.collectI18nUsedKeys()
	if err != nil {
		return nil, err
	}
	locales, err := s.ListSupportedLocales()
	if err != nil {
		return nil, err
	}

	now := time.Now()
	keyAudits, moduleAudits, unusedKeyAudits := s.processI18nAuditRows(rows, now, resp)
	s.resolveI18nKeyConflicts(keyAudits, moduleAudits, locales, resp)
	s.resolveI18nUnusedKeys(unusedKeyAudits, moduleAudits, usedSet, now, resp)
	s.buildI18nModuleAuditSummary(moduleAudits, resp)
	s.sortI18nAuditResults(resp)
	return resp, nil
}

func (s *I18nService) CleanupUnusedKeys(module string) (*I18nCleanupUnusedResp, error) {
	audit, err := s.GetAudit()
	if err != nil {
		return nil, err
	}
	resp := &I18nCleanupUnusedResp{
		Keys:   make([]string, 0),
		Module: strings.TrimSpace(module),
	}
	if s.db == nil {
		return resp, nil
	}

	keys := make([]string, 0, len(audit.UnusedKeys))
	for _, item := range audit.UnusedKeys {
		if resp.Module != "" && !containsString(item.Modules, resp.Module) {
			continue
		}
		keys = append(keys, item.Key)
	}
	if len(keys) == 0 {
		return resp, nil
	}
	sort.Strings(keys)
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		query := tx.Where("`key` IN ?", keys)
		if resp.Module != "" {
			query = query.Where(condModuleEquals, resp.Module)
		}
		deleteResult := query.Delete(&SystemI18n{})
		if deleteResult.Error != nil {
			return deleteResult.Error
		}
		resp.Deleted = deleteResult.RowsAffected
		return nil
	}); err != nil {
		return nil, err
	}
	resp.Keys = keys
	return resp, s.ReloadCache()
}

func (s *I18nService) StartUnusedObservation(module string) (*I18nUnusedLifecycleResp, error) {
	return s.transitionUnusedLifecycle(module, I18nLifecycleStatusActive, I18nLifecycleStatusObserving, false)
}

func (s *I18nService) StartUnusedObservationByKeyPrefixes(module string, prefixes []string) (*I18nUnusedLifecycleResp, error) {
	normalizedPrefixes := make([]string, 0, len(prefixes))
	for _, prefix := range prefixes {
		trimmed := strings.TrimSpace(prefix)
		if trimmed == "" {
			continue
		}
		normalizedPrefixes = append(normalizedPrefixes, trimmed)
	}
	if len(normalizedPrefixes) == 0 {
		return &I18nUnusedLifecycleResp{
			Module:       strings.TrimSpace(module),
			AffectedKeys: make([]string, 0),
		}, nil
	}
	return s.transitionUnusedLifecycleWithFilter(module, I18nLifecycleStatusActive, I18nLifecycleStatusObserving, func(item I18nUnusedKeyItem) bool {
		for _, prefix := range normalizedPrefixes {
			if item.Key == prefix || strings.HasPrefix(item.Key, prefix+".") {
				return true
			}
		}
		return false
	})
}

func (s *I18nService) ArchiveObservedUnusedKeys(module string) (*I18nUnusedLifecycleResp, error) {
	audit, err := s.GetAudit()
	if err != nil {
		return nil, err
	}
	resp := &I18nUnusedLifecycleResp{
		Module:       strings.TrimSpace(module),
		AffectedKeys: make([]string, 0),
	}
	if s.db == nil {
		return resp, nil
	}
	targets := collectI18nArchiveTargets(audit.UnusedKeys, resp)
	if len(targets) == 0 {
		return resp, nil
	}
	now := time.Now()
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		return archiveI18nTargetsInTransaction(tx, targets, now, resp)
	}); err != nil {
		return nil, err
	}
	sort.Strings(resp.AffectedKeys)
	return resp, s.ReloadCache()
}

func collectI18nArchiveTargets(items []I18nUnusedKeyItem, resp *I18nUnusedLifecycleResp) []i18nLifecycleTarget {
	targets := make([]i18nLifecycleTarget, 0)
	for _, item := range items {
		if resp.Module != "" && item.Module != resp.Module {
			continue
		}
		if !item.EligibleForArchive {
			continue
		}
		targets = append(targets, i18nLifecycleTarget{module: item.Module, key: item.Key})
		resp.AffectedKeys = append(resp.AffectedKeys, item.Key)
	}
	return targets
}

func archiveI18nTargetsInTransaction(tx *gorm.DB, targets []i18nLifecycleTarget, now time.Time, resp *I18nUnusedLifecycleResp) error {
	for _, item := range targets {
		updateResult := tx.Model(&SystemI18n{}).
			Where(condModuleKeyEquals, item.module, item.key).
			Updates(map[string]interface{}{
				"lifecycle_status":    I18nLifecycleStatusArchived,
				"lifecycle_marked_at": now,
			})
		if updateResult.Error != nil {
			return updateResult.Error
		}
		resp.AffectedRows += updateResult.RowsAffected
	}
	return nil
}

func (s *I18nService) DeleteArchivedUnusedKeys(module string, confirmArchived bool) (*I18nUnusedLifecycleResp, error) {
	if !confirmArchived {
		return nil, common.NewBadRequest("i18n.lifecycle.delete.confirm_required")
	}
	return s.deleteArchivedUnusedKeys(module, false)
}

func (s *I18nService) DeleteExpiredArchivedUnusedKeys(module string) (*I18nUnusedLifecycleResp, error) {
	return s.deleteArchivedUnusedKeys(module, true)
}

func (s *I18nService) deleteArchivedUnusedKeys(module string, requireEligible bool) (*I18nUnusedLifecycleResp, error) {
	audit, err := s.GetAudit()
	if err != nil {
		return nil, err
	}
	resp := &I18nUnusedLifecycleResp{
		Module:       strings.TrimSpace(module),
		AffectedKeys: make([]string, 0),
	}
	if s.db == nil {
		return resp, nil
	}
	targets := make([]i18nLifecycleTarget, 0)
	for _, item := range audit.UnusedKeys {
		if resp.Module != "" && item.Module != resp.Module {
			continue
		}
		if requireEligible && !item.EligibleForDelete {
			continue
		}
		if item.LifecycleStatus == I18nLifecycleStatusArchived {
			targets = append(targets, i18nLifecycleTarget{module: item.Module, key: item.Key})
			resp.AffectedKeys = append(resp.AffectedKeys, item.Key)
		}
	}
	if len(targets) == 0 {
		return resp, nil
	}
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		return s.deleteI18nTargetsInTransaction(tx, targets, resp)
	}); err != nil {
		return nil, err
	}
	sort.Strings(resp.AffectedKeys)
	return resp, s.ReloadCache()
}

// deleteI18nTargetsInTransaction 在事务内按 module+key 逐个删除 i18n 记录并累计影响行数。
// 该逻辑为原 deleteArchivedUnusedKeys 事务闭包的等价提取，未改变删除事务边界与判定条件。
func (s *I18nService) deleteI18nTargetsInTransaction(tx *gorm.DB, targets []i18nLifecycleTarget, resp *I18nUnusedLifecycleResp) error {
	for _, item := range targets {
		deleteResult := tx.Where(condModuleKeyEquals, item.module, item.key).Delete(&SystemI18n{})
		if deleteResult.Error != nil {
			return deleteResult.Error
		}
		resp.AffectedRows += deleteResult.RowsAffected
	}
	return nil
}

func (s *I18nService) AdvanceUnusedLifecycle(module string) (*I18nUnusedLifecycleAdvanceResp, error) {
	resp := &I18nUnusedLifecycleAdvanceResp{
		Module:                         strings.TrimSpace(module),
		ObservedKeys:                   make([]string, 0),
		ArchivedKeys:                   make([]string, 0),
		DeletedKeys:                    make([]string, 0),
		ArchivedRetentionThresholdDays: I18nArchivedRetentionThresholdDays,
	}
	if s.db == nil {
		return resp, nil
	}

	observeResp, err := s.StartUnusedObservation(resp.Module)
	if err != nil {
		return nil, err
	}
	if observeResp != nil {
		resp.ObservedKeys = append(resp.ObservedKeys, observeResp.AffectedKeys...)
		resp.ObservedRows = observeResp.AffectedRows
	}

	archiveResp, err := s.ArchiveObservedUnusedKeys(resp.Module)
	if err != nil {
		return nil, err
	}
	if archiveResp != nil {
		resp.ArchivedKeys = append(resp.ArchivedKeys, archiveResp.AffectedKeys...)
		resp.ArchivedRows = archiveResp.AffectedRows
	}

	deleteResp, err := s.DeleteExpiredArchivedUnusedKeys(resp.Module)
	if err != nil {
		return nil, err
	}
	if deleteResp != nil {
		resp.DeletedKeys = append(resp.DeletedKeys, deleteResp.AffectedKeys...)
		resp.DeletedRows = deleteResp.AffectedRows
	}

	resp.ObservationOnly = resp.ObservedRows > 0 && resp.ArchivedRows == 0 && resp.DeletedRows == 0
	sort.Strings(resp.ObservedKeys)
	sort.Strings(resp.ArchivedKeys)
	sort.Strings(resp.DeletedKeys)
	return resp, nil
}

func (s *I18nService) PreviewRenameKey(req *I18nRenamePreviewReq) (*I18nRenamePreviewResp, error) {
	module := strings.TrimSpace(req.Module)
	oldKey := strings.TrimSpace(req.OldKey)
	newKey := strings.TrimSpace(req.NewKey)
	if module == "" || oldKey == "" || newKey == "" || oldKey == newKey {
		return nil, common.NewBadRequest("i18n.rename.invalid")
	}

	resp := &I18nRenamePreviewResp{
		Module:                module,
		OldKey:                oldKey,
		NewKey:                newKey,
		AffectedLocales:       make([]string, 0),
		ExistingTargetLocales: make([]string, 0),
		ReferenceFiles:        make([]I18nKeyReferenceFile, 0),
	}
	if s.db == nil {
		return resp, common.ErrDatabaseNotInitialized
	}

	var sourceRows []SystemI18n
	if err := s.db.Where(condModuleKeyEquals, module, oldKey).Order(orderByLocaleASC).Find(&sourceRows).Error; err != nil {
		return nil, err
	}
	resp.AffectedRows = int64(len(sourceRows))
	if resp.AffectedRows == 0 {
		return nil, common.NewNotFound("i18n.rename.source_not_found")
	}
	for _, row := range sourceRows {
		resp.AffectedLocales = append(resp.AffectedLocales, row.Locale)
	}

	var targetRows []SystemI18n
	if err := s.db.Where(condModuleKeyEquals, module, newKey).Order(orderByLocaleASC).Find(&targetRows).Error; err != nil {
		return nil, err
	}
	resp.ExistingTargetRows = int64(len(targetRows))
	for _, row := range targetRows {
		resp.ExistingTargetLocales = append(resp.ExistingTargetLocales, row.Locale)
	}

	referenceFiles, err := scanI18nKeyReferenceFiles(oldKey, newKey, true)
	if err != nil {
		return nil, err
	}
	resp.ReferenceFiles = referenceFiles
	resp.RequiresCodeMigration = len(referenceFiles) > 0
	resp.CanExecute = resp.ExistingTargetRows == 0
	return resp, nil
}

func (s *I18nService) RenameKey(req *I18nRenameExecuteReq) (*I18nRenameExecuteResp, error) {
	preview, err := s.PreviewRenameKey(&I18nRenamePreviewReq{
		Module: req.Module,
		OldKey: req.OldKey,
		NewKey: req.NewKey,
	})
	if err != nil {
		return nil, err
	}
	if preview.ExistingTargetRows > 0 {
		return nil, common.NewConflict("i18n.rename.target_exists")
	}
	if preview.RequiresCodeMigration && !req.ConfirmSourceUpdated {
		return nil, common.NewBadRequest("i18n.rename.source_not_confirmed")
	}

	resp := &I18nRenameExecuteResp{
		Module:         preview.Module,
		OldKey:         preview.OldKey,
		NewKey:         preview.NewKey,
		RenamedLocales: append([]string(nil), preview.AffectedLocales...),
	}
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		updateResult := tx.Model(&SystemI18n{}).
			Where(condModuleKeyEquals, preview.Module, preview.OldKey).
			Updates(map[string]interface{}{
				"key": preview.NewKey,
			})
		if updateResult.Error != nil {
			return updateResult.Error
		}
		resp.RenamedRows = updateResult.RowsAffected
		return nil
	}); err != nil {
		return nil, err
	}
	return resp, s.ReloadCache()
}

func (s *I18nService) ListSupportedLocales() ([]string, error) {
	locales := []string{"zh-CN", "en-US", "ja-JP", "ko-KR", "fr-FR"}
	if s.db == nil {
		return locales, nil
	}

	var rows []string
	if err := s.db.Model(&SystemI18n{}).Distinct("locale").Order(orderByLocaleASC).Pluck("locale", &rows).Error; err != nil {
		return nil, err
	}

	seen := make(map[string]struct{}, len(locales)+len(rows))
	normalized := make([]string, 0, len(locales)+len(rows))
	for _, locale := range append(locales, rows...) {
		value := strings.TrimSpace(locale)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		normalized = append(normalized, value)
	}
	sort.Strings(normalized)
	return normalized, nil
}

type i18nOverviewRow struct {
	Module string
	Group  string
	Key    string
	Locale string
	Value  string
}

func (s *I18nService) GetOverview() (*I18nOverviewResp, error) {
	locales, err := s.ListSupportedLocales()
	if err != nil {
		return nil, err
	}

	resp := &I18nOverviewResp{
		Locales:  locales,
		Coverage: make([]I18nLocaleCoverage, 0, len(locales)),
	}
	if s.db == nil {
		return resp, nil
	}

	var rows []i18nOverviewRow
	if err := s.db.Model(&SystemI18n{}).
		Select("module, group_name as `group`, `key`, locale, value").
		Find(&rows).Error; err != nil {
		return nil, err
	}

	keyLocaleSet := s.aggregateOverviewRows(rows, locales, resp)

	entryCountByLocale := make(map[string]int64, len(locales))
	missingByLocale := make(map[string]int64, len(locales))
	for key, localeSet := range keyLocaleSet {
		s.classifyOverviewCoverage(key, localeSet, locales, resp, entryCountByLocale, missingByLocale)
	}

	for _, locale := range locales {
		resp.Coverage = append(resp.Coverage, I18nLocaleCoverage{
			Locale:       locale,
			EntryCount:   entryCountByLocale[locale],
			MissingCount: missingByLocale[locale],
		})
	}

	return resp, nil
}

// aggregateOverviewRows 汇总各 i18n 行的基础计数（模块/分组/键集合、缺失值数、总条目数），
// 并返回 key→locale 的有效集合。逻辑与原 GetOverview 首轮遍历逐行一致。
func (s *I18nService) aggregateOverviewRows(rows []i18nOverviewRow, locales []string, resp *I18nOverviewResp) map[string]map[string]struct{} {
	moduleSet := make(map[string]struct{})
	groupSet := make(map[string]struct{})
	keyLocaleSet := make(map[string]map[string]struct{}, len(rows))
	for _, row := range rows {
		module := strings.TrimSpace(row.Module)
		group := strings.TrimSpace(row.Group)
		key := strings.TrimSpace(row.Key)
		locale := strings.TrimSpace(row.Locale)
		value := strings.TrimSpace(row.Value)

		if module != "" {
			moduleSet[module] = struct{}{}
		}
		if group != "" {
			groupSet[group] = struct{}{}
		}
		if !hasEffectiveLocaleValue(locale, key, value) {
			resp.MissingValueCount++
		}
		resp.TotalEntries++

		if key == "" || locale == "" {
			continue
		}
		if _, ok := keyLocaleSet[key]; !ok {
			keyLocaleSet[key] = make(map[string]struct{}, len(locales))
		}
		if hasEffectiveLocaleValue(locale, key, value) {
			keyLocaleSet[key][locale] = struct{}{}
		}
	}
	resp.ModuleCount = int64(len(moduleSet))
	resp.GroupCount = int64(len(groupSet))
	resp.UniqueKeyCount = int64(len(keyLocaleSet))
	return keyLocaleSet
}

// classifyOverviewCoverage 对单个 key 在各 locale 下的覆盖情况分类：存在内置值则补足，
// 仍缺失则计入 MissingLocaleCount，否则计入 entryCountByLocale。逻辑与原 GetOverview 次轮遍历一致。
func (s *I18nService) classifyOverviewCoverage(key string, localeSet map[string]struct{}, locales []string, resp *I18nOverviewResp, entryCountByLocale, missingByLocale map[string]int64) {
	for _, locale := range locales {
		if _, ok := localeSet[locale]; !ok {
			if _, builtinOk := getBuiltinLocaleValue(locale, key); builtinOk {
				localeSet[locale] = struct{}{}
			}
		}
		if _, ok := localeSet[locale]; !ok {
			resp.MissingLocaleCount++
			missingByLocale[locale]++
			continue
		}
		entryCountByLocale[locale]++
	}
}

type i18nKeyMeta struct {
	module  string
	group   string
	locales map[string]struct{}
}

func (s *I18nService) ListMissingLocales(module string) (*I18nMissingLocaleResp, error) {
	locales, err := s.ListSupportedLocales()
	if err != nil {
		return nil, err
	}
	resp := &I18nMissingLocaleResp{
		Items: make([]I18nMissingLocaleItem, 0),
	}
	if s.db == nil {
		return resp, nil
	}

	type row struct {
		Module string
		Group  string
		Key    string
		Locale string
	}
	var rows []row
	query := s.db.Model(&SystemI18n{})
	module = strings.TrimSpace(module)
	if module != "" {
		query = query.Where(condModuleEquals, module)
	}
	if err := query.
		Select("module, group_name as `group`, `key`, locale").
		Order(orderByModuleASC).
		Order(orderByGroupNameASC).
		Order(orderByKeyASC).
		Find(&rows).Error; err != nil {
		return nil, err
	}

	keyMap := make(map[string]*i18nKeyMeta, len(rows))
	for _, item := range rows {
		key := strings.TrimSpace(item.Key)
		if key == "" {
			continue
		}
		meta, ok := keyMap[key]
		if !ok {
			meta = &i18nKeyMeta{
				module:  strings.TrimSpace(item.Module),
				group:   strings.TrimSpace(item.Group),
				locales: make(map[string]struct{}, len(locales)),
			}
			keyMap[key] = meta
		}
		locale := strings.TrimSpace(item.Locale)
		if locale != "" {
			meta.locales[locale] = struct{}{}
		}
	}

	s.appendMissingLocaleItems(keyMap, locales, resp)

	resp.Total = int64(len(resp.Items))
	return resp, nil
}

// appendMissingLocaleItems 对每个 key 计算缺失的 locale（无有效值且无内置值），
// 并将缺失项追加到 resp.Items。逻辑与原 ListMissingLocales 末轮遍历逐行一致。
func (s *I18nService) appendMissingLocaleItems(keyMap map[string]*i18nKeyMeta, locales []string, resp *I18nMissingLocaleResp) {
	keys := make([]string, 0, len(keyMap))
	for key := range keyMap {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range keys {
		meta := keyMap[key]
		missing := make([]string, 0, len(locales))
		for _, locale := range locales {
			if _, ok := meta.locales[locale]; !ok {
				if _, builtinOk := getBuiltinLocaleValue(locale, key); builtinOk {
					continue
				}
				missing = append(missing, locale)
			}
		}
		if len(missing) == 0 {
			continue
		}
		resp.Items = append(resp.Items, I18nMissingLocaleItem{
			Module:         meta.module,
			Group:          meta.group,
			Key:            key,
			MissingLocales: missing,
		})
	}
}

func (s *I18nService) FillMissingLocales(module string) (*I18nFillMissingLocaleResp, error) {
	missing, err := s.ListMissingLocales(module)
	if err != nil {
		return nil, err
	}

	resp := &I18nFillMissingLocaleResp{
		Locales: make([]string, 0),
		Keys:    make([]string, 0),
	}
	if s.db == nil || missing.Total == 0 {
		return resp, nil
	}

	localeSet := make(map[string]struct{})
	keySet := make(map[string]struct{})
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		return s.fillMissingLocaleEntries(tx, missing, resp, localeSet, keySet)
	}); err != nil {
		return nil, err
	}

	for locale := range localeSet {
		resp.Locales = append(resp.Locales, locale)
	}
	for key := range keySet {
		resp.Keys = append(resp.Keys, key)
	}
	sort.Strings(resp.Locales)
	sort.Strings(resp.Keys)
	return resp, s.ReloadCache()
}

// fillMissingLocaleEntries 在事务内为每个缺失 (key, locale) 补齐记录（内置值优先），
// 累计 resp.Created 与已创建的 locale/key 集合。逻辑与原 FillMissingLocales 事务闭包逐行一致。
func (s *I18nService) fillMissingLocaleEntries(tx *gorm.DB, missing *I18nMissingLocaleResp, resp *I18nFillMissingLocaleResp, localeSet, keySet map[string]struct{}) error {
	for _, item := range missing.Items {
		for _, locale := range item.MissingLocales {
			value := "[" + item.Key + "]"
			if builtinValue, ok := getBuiltinLocaleValue(locale, item.Key); ok {
				value = builtinValue
			}
			if err := tx.Create(&SystemI18n{
				Module: item.Module,
				Group:  item.Group,
				Key:    item.Key,
				Locale: locale,
				Value:  value,
			}).Error; err != nil {
				return err
			}
			resp.Created++
			localeSet[locale] = struct{}{}
			keySet[item.Key] = struct{}{}
		}
	}
	return nil
}

// i18nHydrateRow 是 HydrateBuiltinLocales 读取既有记录时使用的投影结构，
// 与数据库行一一对应，仅保留后续内置值覆盖判断所需的字段。
type i18nHydrateRow struct {
	ID     uint64
	Module string
	Group  string
	Key    string
	Locale string
	Value  string
}

// HydrateBuiltinLocales 将内置语言值补齐到数据库：对已有但因占位符/空值缺失有效值的记录用
// 内置值覆盖（Update），对 ListMissingLocales 返回的真正缺失 (key, locale) 用内置值新建记录（Create）。
// 覆盖优先级与顺序与原实现保持一致：先更新既有行，再补齐缺失行，最后汇总受影响的 locale/key 集合。
func (s *I18nService) HydrateBuiltinLocales(module string) (*I18nHydrateBuiltinResp, error) {
	module = strings.TrimSpace(module)
	resp := &I18nHydrateBuiltinResp{
		Locales: make([]string, 0),
		Keys:    make([]string, 0),
	}
	if s.db == nil {
		return resp, nil
	}

	rows, err := s.loadHydrateBuiltinRows(module)
	if err != nil {
		return nil, err
	}
	missing, err := s.ListMissingLocales(module)
	if err != nil {
		return nil, err
	}

	localeSet := make(map[string]struct{})
	keySet := make(map[string]struct{})
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if err := s.hydrateBuiltinExistingRows(tx, rows, resp, localeSet, keySet); err != nil {
			return err
		}
		return s.hydrateBuiltinMissingRows(tx, missing, resp, localeSet, keySet)
	}); err != nil {
		return nil, err
	}

	s.collectHydrateBuiltinSets(resp, localeSet, keySet)
	return resp, s.ReloadCache()
}

// loadHydrateBuiltinRows 读取待内置值覆盖的既有 i18n 记录（按模块可选过滤，多列排序稳定）。
func (s *I18nService) loadHydrateBuiltinRows(module string) ([]i18nHydrateRow, error) {
	var rows []i18nHydrateRow
	query := s.db.Model(&SystemI18n{}).Select("id, module, group_name as `group`, `key`, locale, value")
	if module != "" {
		query = query.Where(condModuleEquals, module)
	}
	if err := query.Order(orderByModuleASC).Order(orderByGroupNameASC).Order(orderByKeyASC).Order(orderByLocaleASC).Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

// hydrateBuiltinExistingRows 遍历既有记录：仅当该记录没有有效存储值且存在对应内置值时，
// 用内置值覆盖其 value（Update），并累计受影响 locale/key。逻辑与原事务首轮遍历逐行一致。
func (s *I18nService) hydrateBuiltinExistingRows(tx *gorm.DB, rows []i18nHydrateRow, resp *I18nHydrateBuiltinResp, localeSet, keySet map[string]struct{}) error {
	for _, item := range rows {
		if hasStoredLocaleValue(item.Value) {
			continue
		}
		builtinValue, ok := getBuiltinLocaleValue(item.Locale, item.Key)
		if !ok {
			continue
		}
		if err := tx.Model(&SystemI18n{}).Where("id = ?", item.ID).Update("value", builtinValue).Error; err != nil {
			return err
		}
		resp.Updated++
		localeSet[item.Locale] = struct{}{}
		keySet[item.Key] = struct{}{}
	}
	return nil
}

// hydrateBuiltinMissingRows 遍历缺失 (key, locale)：仅当存在对应内置值时用内置值新建记录（Create），
// 并累计受影响 locale/key。逻辑与原事务次轮遍历逐行一致，未改变新建记录的字段与判定。
func (s *I18nService) hydrateBuiltinMissingRows(tx *gorm.DB, missing *I18nMissingLocaleResp, resp *I18nHydrateBuiltinResp, localeSet, keySet map[string]struct{}) error {
	for _, item := range missing.Items {
		for _, locale := range item.MissingLocales {
			builtinValue, ok := getBuiltinLocaleValue(locale, item.Key)
			if !ok {
				continue
			}
			if err := tx.Create(&SystemI18n{
				Module: item.Module,
				Group:  item.Group,
				Key:    item.Key,
				Locale: locale,
				Value:  builtinValue,
			}).Error; err != nil {
				return err
			}
			resp.Created++
			localeSet[locale] = struct{}{}
			keySet[item.Key] = struct{}{}
		}
	}
	return nil
}

// collectHydrateBuiltinSets 将去重后的 locale/key 集合汇总进响应并按字典序排序，
// 与原实现末尾聚合循环等价。
func (s *I18nService) collectHydrateBuiltinSets(resp *I18nHydrateBuiltinResp, localeSet, keySet map[string]struct{}) {
	for locale := range localeSet {
		resp.Locales = append(resp.Locales, locale)
	}
	for key := range keySet {
		resp.Keys = append(resp.Keys, key)
	}
	sort.Strings(resp.Locales)
	sort.Strings(resp.Keys)
}

// resolveI18nScanRoots 解析 i18n 扫描根路径，优先级顺序与原实现一致：
//  1. 若 PANTHEON_WORKSPACE_ROOT 指向含 backend+frontend 的目录，则取之；
//  2. 否则以 os.Getwd()（回退 runtime.Caller 文件）为基准向上回溯，寻找含 backend+frontend 的祖先目录；
//  3. 若基准为空（无法获取工作目录/调用文件），则回退为相对 "backend"/"frontend"；
//  4. 若回溯到底仍未命中，则以基准目录拼接 backend/frontend 作为兜底。
//
// 路径解析使用 filepath.Join/Clean，去重通过 seen 集合完成；向上回溯在遇到「父目录等于当前目录」时
// 终止，因此不会越界也不会死循环。该实现仅将原函数逻辑等价拆解为阶段 helper，未改变任何解析结果。
func resolveI18nScanRoots() []string {
	seen := map[string]struct{}{}
	roots := make([]string, 0, 2)
	appendRoot := func(root string) {
		normalized := strings.TrimSpace(filepath.Clean(root))
		if normalized == "" {
			return
		}
		if _, ok := seen[normalized]; ok {
			return
		}
		seen[normalized] = struct{}{}
		roots = append(roots, normalized)
	}

	if configuredRoot := strings.TrimSpace(os.Getenv("PANTHEON_WORKSPACE_ROOT")); configuredRoot != "" {
		if attemptScanRootPair(appendRoot, configuredRoot) {
			return roots
		}
	}

	base := resolveScanRootBase()
	if base == "" {
		appendRoot("backend")
		appendRoot("frontend")
		return roots
	}

	if walkUpScanRoots(appendRoot, base) {
		return roots
	}

	appendRoot(filepath.Join(base, "backend"))
	appendRoot(filepath.Join(base, "frontend"))
	return roots
}

// attemptScanRootPair 若 dir 下同时包含 backend 与 frontend 子目录，则将两个绝对路径追加进 roots
// 并返回 true，否则返回 false。路径拼接与原实现一致使用 filepath.Join。
func attemptScanRootPair(appendRoot func(string), dir string) bool {
	if dir == "" {
		return false
	}
	backendRoot := filepath.Join(dir, "backend")
	frontendRoot := filepath.Join(dir, "frontend")
	if dirExists(backendRoot) && dirExists(frontendRoot) {
		appendRoot(backendRoot)
		appendRoot(frontendRoot)
		return true
	}
	return false
}

// resolveScanRootBase 解析扫描基准目录：优先取当前工作目录，回退为运行时调用源文件所在路径，
// 两者皆不可得时返回空字符串。与原实现读取顺序一致。
func resolveScanRootBase() string {
	if cwd, err := os.Getwd(); err == nil {
		return cwd
	}
	_, currentFile, _, ok := runtime.Caller(0)
	if ok {
		return currentFile
	}
	return ""
}

// walkUpScanRoots 自 base（若为文件则取其所在目录）开始逐级向上回溯，寻找首个同时包含 backend 与
// frontend 的祖先目录；命中即追加并返回 true，到文件系统的根（父目录等于当前目录）仍未命中则返回 false。
// 与原实现回溯循环等价：单次步进，父目录等于当前目录时终止，避免越界与死循环。
func walkUpScanRoots(appendRoot func(string), base string) bool {
	current := base
	if info, err := os.Stat(current); err == nil && !info.IsDir() {
		current = filepath.Dir(current)
	}
	for {
		if attemptScanRootPair(appendRoot, current) {
			return true
		}
		parent := filepath.Dir(current)
		if parent == current {
			return false
		}
		current = parent
	}
}

// scanI18nKeys walks every i18n scan root and collects all likely i18n keys
// referenced in source files. It is behaviourally identical to the previous
// implementation; the per-file matching logic now lives in scanI18nKeysInFile so
// the control flow here stays flat.
func scanI18nKeys(excludeCatalog bool) ([]string, error) {
	re := regexp.MustCompile("[\"'`]([A-Za-z0-9_]+\\.[A-Za-z0-9_\\.]+)[\"'`]")
	keyMap := make(map[string]struct{})
	for _, root := range resolveI18nScanRoots() {
		if err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil || info == nil || info.IsDir() {
				return nil
			}
			scanI18nKeysInFile(path, excludeCatalog, re, keyMap)
			return nil
		}); err != nil {
			return nil, err
		}
	}
	keys := make([]string, 0, len(keyMap))
	for key := range keyMap {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys, nil
}

// scanI18nKeysInFile inspects a single source file for i18n key string literals
// and records every key that isLikelyI18nKey accepts. Files that are not source
// files, ignored catalogs, or unreadable are skipped. This is the exact logic
// that previously ran inline inside the filepath.Walk callback of scanI18nKeys.
func scanI18nKeysInFile(path string, excludeCatalog bool, re *regexp.Regexp, keyMap map[string]struct{}) {
	ext := strings.ToLower(filepath.Ext(path))
	if ext != ".go" && ext != ".ts" && ext != ".tsx" {
		return
	}
	if excludeCatalog && isIgnoredI18nUsageFile(path) {
		return
	}
	content, readErr := os.ReadFile(path) // #nosec G304 -- path comes from filepath.Walk under the repo source roots being scanned.
	if readErr != nil {
		return
	}
	for _, m := range re.FindAllStringSubmatch(string(content), -1) {
		key := strings.TrimSpace(m[1])
		if !isLikelyI18nKey(key) {
			continue
		}
		keyMap[key] = struct{}{}
	}
}

var (
	i18nKeySegmentPattern = regexp.MustCompile("^[A-Za-z0-9_]+$")
	i18nKeyLetterPattern  = regexp.MustCompile("[A-Za-z]")
)

// isLikelyI18nKey reports whether key looks like a dotted i18n message key
// rather than a file path, URL, or domain. The original monolithic checks are
// delegated to small predicates so the control flow stays flat while remaining
// byte-for-byte equivalent.
func isLikelyI18nKey(key string) bool {
	normalized := strings.TrimSpace(key)
	if !i18nKeyPreliminaryValid(normalized) {
		return false
	}

	segments := strings.Split(normalized, ".")
	if len(segments) < 2 {
		return false
	}
	first := strings.TrimSpace(segments[0])
	if first == "" {
		return false
	}
	firstRune := rune(first[0])
	if !isAlphaRune(firstRune) {
		return false
	}

	if !i18nKeySegmentsAllValid(segments) {
		return false
	}

	last := strings.ToLower(segments[len(segments)-1])
	if i18nKeyLooksLikeFileExtension(last) {
		return false
	}

	if i18nKeyLooksLikeDomain(first, last, len(segments)) {
		return false
	}

	return true
}

// i18nKeyPreliminaryValid mirrors the original leading guard: an empty key or a
// key containing no letter cannot be an i18n key.
func i18nKeyPreliminaryValid(normalized string) bool {
	if normalized == "" {
		return false
	}
	return i18nKeyLetterPattern.MatchString(normalized)
}

// isAlphaRune reports whether r is an ASCII letter, matching the original
// first-segment leading rune check.
func isAlphaRune(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')
}

// i18nKeySegmentsAllValid validates every dotted segment of a candidate key the
// same way the original per-segment loop did.
func i18nKeySegmentsAllValid(segments []string) bool {
	for _, segment := range segments {
		trimmed := strings.TrimSpace(segment)
		if trimmed == "" || !i18nKeySegmentPattern.MatchString(trimmed) {
			return false
		}
	}
	return true
}

// i18nKeyLooksLikeFileExtension reports whether a trailing segment is a common
// file extension, matching the original switch that rejected such keys.
func i18nKeyLooksLikeFileExtension(last string) bool {
	switch last {
	case "go", "ts", "tsx", "js", "jsx", "json", "csv", "txt", "png", "gif", "jpg", "jpeg", "svg", "ico", "css", "scss", "less", "html", "map", "md", "yml", "yaml":
		return true
	}
	return false
}

// i18nKeyLooksLikeDomain reports whether a short key resembles a domain/host
// name (e.g. "api.example.com"), matching the original nested switch block.
func i18nKeyLooksLikeDomain(first, last string, segmentCount int) bool {
	if segmentCount <= 3 {
		switch first {
		case "db", "api", "www", "mail", "smtp", "cdn", "img", "static", "files", "localhost":
			switch last {
			case "com", "net", "org", "io", "cn", "dev", "app", "local", "internal", "lan":
				return true
			}
		}
	}
	return false
}

// scanI18nKeyReferenceFiles finds every source file that references targetKey
// and reports the matched lines. The previous inlined walk callback now lives in
// scanI18nKeyReferenceInFile so this function stays flat.
func scanI18nKeyReferenceFiles(targetKey string, newKey string, excludeCatalog bool) ([]I18nKeyReferenceFile, error) {
	normalizedTarget := strings.TrimSpace(targetKey)
	if normalizedTarget == "" {
		return []I18nKeyReferenceFile{}, nil
	}
	trimNewKey := strings.TrimSpace(newKey)
	results := make([]I18nKeyReferenceFile, 0)
	for _, root := range resolveI18nScanRoots() {
		if err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil || info == nil || info.IsDir() {
				return nil
			}
			scanI18nKeyReferenceInFile(path, normalizedTarget, trimNewKey, excludeCatalog, &results)
			return nil
		}); err != nil {
			return nil, err
		}
	}
	sort.Slice(results, func(i, j int) bool { return results[i].Path < results[j].Path })
	return results, nil
}

// scanI18nKeyReferenceInFile inspects a single source file for references to
// normalizedTarget and appends a result entry when found. Skipped files and
// unreadable files behave exactly as in the original implementation.
func scanI18nKeyReferenceInFile(path, normalizedTarget, trimNewKey string, excludeCatalog bool, results *[]I18nKeyReferenceFile) {
	ext := strings.ToLower(filepath.Ext(path))
	if ext != ".go" && ext != ".ts" && ext != ".tsx" {
		return
	}
	if excludeCatalog && isIgnoredI18nUsageFile(path) {
		return
	}
	content, readErr := os.ReadFile(path) // #nosec G304 -- path comes from filepath.Walk under the repo source roots being scanned.
	if readErr != nil {
		return
	}
	text := string(content)
	if !strings.Contains(text, normalizedTarget) {
		return
	}
	relativePath := relativizePath(path)
	matches := buildI18nKeyReferenceMatches(text, normalizedTarget, trimNewKey)
	*results = append(*results, I18nKeyReferenceFile{
		Path:                 relativePath,
		MatchCount:           len(matches),
		SuggestedReplacement: trimNewKey,
		Matches:              matches,
	})
}

// relativizePath returns path relative to the current working directory using
// forward slashes, falling back to the absolute path when relativization fails.
// Equivalent to the inline computation that previously lived in
// scanI18nKeyReferenceFiles.
func relativizePath(path string) string {
	if cwd, cwdErr := os.Getwd(); cwdErr == nil {
		if rel, relErr := filepath.Rel(cwd, path); relErr == nil {
			return filepath.ToSlash(rel)
		}
	}
	return path
}

func buildI18nKeyReferenceMatches(content, oldKey, newKey string) []I18nKeyReferenceMatch {
	lines := strings.Split(content, "\n")
	matches := make([]I18nKeyReferenceMatch, 0)
	for index, line := range lines {
		searchStart := 0
		for {
			offset := strings.Index(line[searchStart:], oldKey)
			if offset < 0 {
				break
			}
			column := searchStart + offset + 1
			snippet := strings.TrimSpace(line)
			replacementHint := snippet
			if newKey != "" {
				replacementHint = strings.ReplaceAll(snippet, oldKey, newKey)
			}
			matches = append(matches, I18nKeyReferenceMatch{
				Line:            index + 1,
				Column:          column,
				Snippet:         snippet,
				ReplacementHint: replacementHint,
			})
			searchStart += offset + len(oldKey)
		}
	}
	return matches
}

func isIgnoredI18nUsageFile(path string) bool {
	normalized := filepath.ToSlash(strings.TrimSpace(path))
	if strings.Contains(normalized, "/frontend/node_modules/") ||
		strings.Contains(normalized, "/frontend/dist/") ||
		strings.Contains(normalized, "/frontend/test-results/") ||
		strings.Contains(normalized, "/frontend/playwright-report/") ||
		strings.Contains(normalized, "/frontend/artifacts/") {
		return true
	}
	if strings.HasSuffix(normalized, "_test.go") ||
		strings.HasSuffix(normalized, ".spec.ts") ||
		strings.HasSuffix(normalized, ".spec.tsx") ||
		strings.Contains(normalized, "/frontend/tests/") {
		return true
	}
	return strings.HasSuffix(normalized, "/frontend/src/i18n/index.ts") ||
		strings.Contains(normalized, "/frontend/src/i18n/resources/") ||
		strings.HasSuffix(normalized, "/backend/modules/system/i18n/seed_data.go")
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func sortedSetKeys(values map[string]struct{}) []string {
	items := make([]string, 0, len(values))
	for value := range values {
		if strings.TrimSpace(value) == "" {
			continue
		}
		items = append(items, value)
	}
	sort.Strings(items)
	return items
}

func containsString(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}

func allValuesMissing(values map[string]struct{}) bool {
	if len(values) == 0 {
		return true
	}
	for value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" && !strings.HasPrefix(trimmed, "[") {
			return false
		}
	}
	return true
}

func suggestScopedI18nKey(module, key string) string {
	normalizedModule := strings.TrimSpace(module)
	normalizedKey := strings.TrimSpace(key)
	if normalizedModule == "" || normalizedKey == "" {
		return normalizedKey
	}
	prefix := normalizedModule + "."
	if strings.HasPrefix(normalizedKey, prefix) {
		return normalizedKey
	}
	return prefix + normalizedKey
}

func normalizeI18nLifecycleStatus(status string) string {
	switch strings.TrimSpace(status) {
	case I18nLifecycleStatusObserving:
		return I18nLifecycleStatusObserving
	case I18nLifecycleStatusArchived:
		return I18nLifecycleStatusArchived
	default:
		return I18nLifecycleStatusActive
	}
}

func (s *I18nService) resetI18nLifecycle(_ string, module, key string) error {
	return s.db.Model(&SystemI18n{}).
		Where(condModuleKeyEquals, module, key).
		Updates(map[string]interface{}{
			"lifecycle_status":    I18nLifecycleStatusActive,
			"lifecycle_marked_at": nil,
		}).Error
}

func (s *I18nService) transitionUnusedLifecycle(module string, fromStatus string, toStatus string, requireConfirm bool) (*I18nUnusedLifecycleResp, error) {
	if requireConfirm {
		return nil, common.NewBadRequest("i18n.lifecycle.transition.invalid")
	}
	return s.transitionUnusedLifecycleWithFilter(module, fromStatus, toStatus, nil)
}

// i18nLifecycleTarget identifies a single unused i18n key selected for transition.
type i18nLifecycleTarget struct {
	module string
	key    string
}

// matchesLifecycleTransitionTarget reports whether an unused key qualifies for a
// lifecycle transition from fromStatus, honoring the optional caller filter and module scope.
func matchesLifecycleTransitionTarget(item I18nUnusedKeyItem, module string, fromStatus string, filter func(I18nUnusedKeyItem) bool) bool {
	if module != "" && item.Module != module {
		return false
	}
	if filter != nil && !filter(item) {
		return false
	}
	return normalizeI18nLifecycleStatus(item.LifecycleStatus) == fromStatus
}

// collectUnusedLifecycleTargets walks the audit's unused keys and returns the
// matching transition targets plus the corresponding affected key names.
func collectUnusedLifecycleTargets(audit *I18nAuditResp, module string, fromStatus string, filter func(I18nUnusedKeyItem) bool) ([]i18nLifecycleTarget, []string) {
	targets := make([]i18nLifecycleTarget, 0)
	affectedKeys := make([]string, 0)
	for _, item := range audit.UnusedKeys {
		if !matchesLifecycleTransitionTarget(item, module, fromStatus, filter) {
			continue
		}
		targets = append(targets, i18nLifecycleTarget{module: item.Module, key: item.Key})
		affectedKeys = append(affectedKeys, item.Key)
	}
	return targets, affectedKeys
}

// applyUnusedLifecycleTransition persists the lifecycle transition for all targets
// inside a single DB transaction and returns the total number of affected rows.
func (s *I18nService) applyUnusedLifecycleTransition(targets []i18nLifecycleTarget, toStatus string, now time.Time) (int64, error) {
	var affected int64
	err := s.db.Transaction(func(tx *gorm.DB) error {
		for _, item := range targets {
			result := tx.Model(&SystemI18n{}).
				Where(condModuleKeyEquals, item.module, item.key).
				Updates(map[string]interface{}{
					"lifecycle_status":    toStatus,
					"lifecycle_marked_at": now,
				})
			if result.Error != nil {
				return result.Error
			}
			affected += result.RowsAffected
		}
		return nil
	})
	return affected, err
}

func (s *I18nService) transitionUnusedLifecycleWithFilter(module string, fromStatus string, toStatus string, filter func(I18nUnusedKeyItem) bool) (*I18nUnusedLifecycleResp, error) {
	audit, err := s.GetAudit()
	if err != nil {
		return nil, err
	}
	module = strings.TrimSpace(module)
	resp := &I18nUnusedLifecycleResp{
		Module:       module,
		AffectedKeys: make([]string, 0),
	}
	if s.db == nil {
		return resp, nil
	}
	targets, affectedKeys := collectUnusedLifecycleTargets(audit, module, fromStatus, filter)
	resp.AffectedKeys = affectedKeys
	if len(targets) == 0 {
		return resp, nil
	}
	now := time.Now()
	affected, err := s.applyUnusedLifecycleTransition(targets, toStatus, now)
	if err != nil {
		return nil, err
	}
	resp.AffectedRows += affected
	sort.Strings(resp.AffectedKeys)
	return resp, s.ReloadCache()
}

func normalizeI18nQuery(query *I18nQuery) *I18nQuery {
	if query == nil {
		query = &I18nQuery{}
	}
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.PageSize <= 0 {
		query.PageSize = 20
	}
	if query.PageSize > 200 {
		query.PageSize = 200
	}
	query.Module = strings.TrimSpace(query.Module)
	query.Group = strings.TrimSpace(query.Group)
	query.Locale = strings.TrimSpace(query.Locale)
	query.Key = strings.TrimSpace(query.Key)
	query.SortBy = strings.TrimSpace(query.SortBy)
	query.SortOrder = strings.TrimSpace(query.SortOrder)
	return query
}

func cloneLangPack(pack map[string]string) map[string]string {
	cloned := make(map[string]string, len(pack))
	for key, value := range pack {
		cloned[key] = value
	}
	return cloned
}
