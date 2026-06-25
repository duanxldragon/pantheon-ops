package system

import (
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"

	"pantheon-ops/backend/pkg/common"

	"gorm.io/gorm"
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
		createdForKey := false
		for _, locale := range supportedLocales {
			var exists int64
			if err := s.db.Model(&SystemI18n{}).Where("`key` = ? AND locale = ?", k, locale).Count(&exists).Error; err != nil {
				return resp, err
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
				return resp, err
			}
			createdForKey = true
		}
		if createdForKey {
			resp.Count++
			resp.Keys = append(resp.Keys, k)
		}
	}
	return resp, s.ReloadCache()
}

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

	type row struct {
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
	var rows []row
	if err := s.db.Model(&SystemI18n{}).
		Select("id, module, group_name as `group`, `key`, locale, value, lifecycle_status, lifecycle_marked_at, updated_at").
		Order("module ASC").
		Order("group_name ASC").
		Order("`key` ASC").
		Order("locale ASC").
		Find(&rows).Error; err != nil {
		return nil, err
	}

	usedKeys, err := scanI18nKeys(true)
	if err != nil {
		return nil, err
	}
	usedSet := make(map[string]struct{}, len(usedKeys))
	for _, key := range usedKeys {
		usedSet[key] = struct{}{}
	}

	locales, err := s.ListSupportedLocales()
	if err != nil {
		return nil, err
	}

	type keyAudit struct {
		modules map[string]struct{}
		groups  map[string]struct{}
		locales map[string]struct{}
		values  map[string]struct{}
		rows    int64
	}
	type moduleAudit struct {
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

	keyAudits := make(map[string]*keyAudit)
	type unusedKeyAudit struct {
		module            string
		key               string
		groups            map[string]struct{}
		locales           map[string]struct{}
		values            map[string]struct{}
		lifecycleStatus   string
		lifecycleMarkedAt *time.Time
	}
	unusedKeyAudits := make(map[string]*unusedKeyAudit)
	moduleAudits := make(map[string]*moduleAudit)
	now := time.Now()
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
			keyMeta = &keyAudit{
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

		moduleMeta, ok := moduleAudits[module]
		if !ok {
			moduleMeta = &moduleAudit{
				keys:               make(map[string]struct{}),
				unusedKeys:         make(map[string]struct{}),
				duplicateKeys:      make(map[string]struct{}),
				missingLocaleKeys:  make(map[string]struct{}),
				observingKeys:      make(map[string]struct{}),
				archivedKeys:       make(map[string]struct{}),
				deleteEligibleKeys: make(map[string]struct{}),
			}
			moduleAudits[module] = moduleMeta
		}
		moduleMeta.entryCount++
		moduleMeta.keys[key] = struct{}{}
		if !hasEffectiveLocaleValue(locale, key, value) {
			moduleMeta.placeholderCount++
			staleDays := int64(now.Sub(item.UpdatedAt).Hours() / 24)
			if staleDays >= I18nStalePlaceholderThresholdDays {
				moduleMeta.stalePlaceholders++
				resp.StalePlaceholders = append(resp.StalePlaceholders, I18nStalePlaceholderItem{
					ID:        item.ID,
					Module:    module,
					Group:     group,
					Key:       key,
					Locale:    locale,
					Value:     value,
					UpdatedAt: item.UpdatedAt.Format(time.RFC3339),
					StaleDays: staleDays,
				})
			}
		}

		unusedCompositeKey := module + "|" + key
		unusedMeta, exists := unusedKeyAudits[unusedCompositeKey]
		if !exists {
			unusedMeta = &unusedKeyAudit{
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

	for key, meta := range keyAudits {
		for _, locale := range locales {
			if _, ok := meta.locales[locale]; ok {
				continue
			}
			if _, builtinOk := getBuiltinLocaleValue(locale, key); builtinOk {
				meta.locales[locale] = struct{}{}
			}
		}
		if len(meta.modules) > 1 || len(meta.groups) > 1 {
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
		if int64(len(meta.locales)) < int64(len(locales)) {
			for _, module := range sortedSetKeys(meta.modules) {
				moduleAudits[module].missingLocaleKeys[key] = struct{}{}
			}
		}
	}

	for compositeKey, meta := range unusedKeyAudits {
		if _, ok := usedSet[meta.key]; ok {
			if meta.lifecycleStatus != I18nLifecycleStatusActive {
				if err := s.resetI18nLifecycle(compositeKey, meta.module, meta.key); err == nil {
					meta.lifecycleStatus = I18nLifecycleStatusActive
					meta.lifecycleMarkedAt = nil
				}
			}
			continue
		}
		moduleMeta := moduleAudits[meta.module]
		moduleMeta.unusedKeys[meta.key] = struct{}{}
		observingDays := int64(0)
		markedAt := ""
		if meta.lifecycleMarkedAt != nil {
			markedAt = meta.lifecycleMarkedAt.Format(time.RFC3339)
			observingDays = int64(now.Sub(*meta.lifecycleMarkedAt).Hours() / 24)
		}
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
			query = query.Where("module = ?", resp.Module)
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
	type target struct {
		module string
		key    string
	}
	targets := make([]target, 0)
	for _, item := range audit.UnusedKeys {
		if resp.Module != "" && item.Module != resp.Module {
			continue
		}
		if item.EligibleForArchive {
			targets = append(targets, target{module: item.Module, key: item.Key})
			resp.AffectedKeys = append(resp.AffectedKeys, item.Key)
		}
	}
	if len(targets) == 0 {
		return resp, nil
	}
	now := time.Now()
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		for _, item := range targets {
			updateResult := tx.Model(&SystemI18n{}).
				Where("module = ? AND `key` = ?", item.module, item.key).
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
	}); err != nil {
		return nil, err
	}
	sort.Strings(resp.AffectedKeys)
	return resp, s.ReloadCache()
}

func (s *I18nService) DeleteArchivedUnusedKeys(module string, confirmArchived bool) (*I18nUnusedLifecycleResp, error) {
	if !confirmArchived {
		return nil, errors.New("i18n.lifecycle.delete.confirm_required")
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
	type target struct {
		module string
		key    string
	}
	targets := make([]target, 0)
	for _, item := range audit.UnusedKeys {
		if resp.Module != "" && item.Module != resp.Module {
			continue
		}
		if requireEligible && !item.EligibleForDelete {
			continue
		}
		if item.LifecycleStatus == I18nLifecycleStatusArchived {
			targets = append(targets, target{module: item.Module, key: item.Key})
			resp.AffectedKeys = append(resp.AffectedKeys, item.Key)
		}
	}
	if len(targets) == 0 {
		return resp, nil
	}
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		for _, item := range targets {
			deleteResult := tx.Where("module = ? AND `key` = ?", item.module, item.key).Delete(&SystemI18n{})
			if deleteResult.Error != nil {
				return deleteResult.Error
			}
			resp.AffectedRows += deleteResult.RowsAffected
		}
		return nil
	}); err != nil {
		return nil, err
	}
	sort.Strings(resp.AffectedKeys)
	return resp, s.ReloadCache()
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
		return nil, errors.New("i18n.rename.invalid")
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
	if err := s.db.Where("module = ? AND `key` = ?", module, oldKey).Order("locale ASC").Find(&sourceRows).Error; err != nil {
		return nil, err
	}
	resp.AffectedRows = int64(len(sourceRows))
	if resp.AffectedRows == 0 {
		return nil, errors.New("i18n.rename.source_not_found")
	}
	for _, row := range sourceRows {
		resp.AffectedLocales = append(resp.AffectedLocales, row.Locale)
	}

	var targetRows []SystemI18n
	if err := s.db.Where("module = ? AND `key` = ?", module, newKey).Order("locale ASC").Find(&targetRows).Error; err != nil {
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
		return nil, errors.New("i18n.rename.target_exists")
	}
	if preview.RequiresCodeMigration && !req.ConfirmSourceUpdated {
		return nil, errors.New("i18n.rename.source_not_confirmed")
	}

	resp := &I18nRenameExecuteResp{
		Module:         preview.Module,
		OldKey:         preview.OldKey,
		NewKey:         preview.NewKey,
		RenamedLocales: append([]string(nil), preview.AffectedLocales...),
	}
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		updateResult := tx.Model(&SystemI18n{}).
			Where("module = ? AND `key` = ?", preview.Module, preview.OldKey).
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
	if err := s.db.Model(&SystemI18n{}).Distinct("locale").Order("locale ASC").Pluck("locale", &rows).Error; err != nil {
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

	type overviewRow struct {
		Module string
		Group  string
		Key    string
		Locale string
		Value  string
	}
	var rows []overviewRow
	if err := s.db.Model(&SystemI18n{}).
		Select("module, group_name as `group`, `key`, locale, value").
		Find(&rows).Error; err != nil {
		return nil, err
	}

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

	entryCountByLocale := make(map[string]int64, len(locales))
	missingByLocale := make(map[string]int64, len(locales))
	for key, localeSet := range keyLocaleSet {
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

	for _, locale := range locales {
		resp.Coverage = append(resp.Coverage, I18nLocaleCoverage{
			Locale:       locale,
			EntryCount:   entryCountByLocale[locale],
			MissingCount: missingByLocale[locale],
		})
	}

	return resp, nil
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
		query = query.Where("module = ?", module)
	}
	if err := query.
		Select("module, group_name as `group`, `key`, locale").
		Order("module ASC").
		Order("group_name ASC").
		Order("`key` ASC").
		Find(&rows).Error; err != nil {
		return nil, err
	}

	type keyMeta struct {
		module  string
		group   string
		locales map[string]struct{}
	}
	keyMap := make(map[string]*keyMeta, len(rows))
	for _, item := range rows {
		key := strings.TrimSpace(item.Key)
		if key == "" {
			continue
		}
		meta, ok := keyMap[key]
		if !ok {
			meta = &keyMeta{
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

	resp.Total = int64(len(resp.Items))
	return resp, nil
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

func (s *I18nService) HydrateBuiltinLocales(module string) (*I18nHydrateBuiltinResp, error) {
	module = strings.TrimSpace(module)
	resp := &I18nHydrateBuiltinResp{
		Locales: make([]string, 0),
		Keys:    make([]string, 0),
	}
	if s.db == nil {
		return resp, nil
	}

	type row struct {
		ID     uint64
		Module string
		Group  string
		Key    string
		Locale string
		Value  string
	}
	var rows []row
	query := s.db.Model(&SystemI18n{}).Select("id, module, group_name as `group`, `key`, locale, value")
	if module != "" {
		query = query.Where("module = ?", module)
	}
	if err := query.Order("module ASC").Order("group_name ASC").Order("`key` ASC").Order("locale ASC").Find(&rows).Error; err != nil {
		return nil, err
	}

	missing, err := s.ListMissingLocales(module)
	if err != nil {
		return nil, err
	}

	localeSet := make(map[string]struct{})
	keySet := make(map[string]struct{})
	if err := s.db.Transaction(func(tx *gorm.DB) error {
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
		backendRoot := filepath.Join(configuredRoot, "backend")
		frontendRoot := filepath.Join(configuredRoot, "frontend")
		if dirExists(backendRoot) && dirExists(frontendRoot) {
			appendRoot(backendRoot)
			appendRoot(frontendRoot)
			return roots
		}
	}

	base := ""
	if cwd, err := os.Getwd(); err == nil {
		base = cwd
	}
	if base == "" {
		_, currentFile, _, ok := runtime.Caller(0)
		if ok {
			base = currentFile
		}
	}
	if base == "" {
		appendRoot("backend")
		appendRoot("frontend")
		return roots
	}

	current := base
	if info, err := os.Stat(current); err == nil && !info.IsDir() {
		current = filepath.Dir(current)
	}
	for {
		backendRoot := filepath.Join(current, "backend")
		frontendRoot := filepath.Join(current, "frontend")
		if dirExists(backendRoot) && dirExists(frontendRoot) {
			appendRoot(backendRoot)
			appendRoot(frontendRoot)
			return roots
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}

	appendRoot(filepath.Join(base, "backend"))
	appendRoot(filepath.Join(base, "frontend"))
	return roots
}

func scanI18nKeys(excludeCatalog bool) ([]string, error) {
	re := regexp.MustCompile("[\"'`]([A-Za-z0-9_]+\\.[A-Za-z0-9_\\.]+)[\"'`]")
	keyMap := make(map[string]struct{})
	for _, root := range resolveI18nScanRoots() {
		if err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil || info == nil || info.IsDir() {
				return nil
			}
			ext := strings.ToLower(filepath.Ext(path))
			if ext != ".go" && ext != ".ts" && ext != ".tsx" {
				return nil
			}
			if excludeCatalog && isIgnoredI18nUsageFile(path) {
				return nil
			}
			content, readErr := os.ReadFile(path)
			if readErr != nil {
				return nil
			}
			for _, m := range re.FindAllStringSubmatch(string(content), -1) {
				key := strings.TrimSpace(m[1])
				if !isLikelyI18nKey(key) {
					continue
				}
				keyMap[key] = struct{}{}
			}
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

var (
	i18nKeySegmentPattern = regexp.MustCompile("^[A-Za-z0-9_]+$")
	i18nKeyLetterPattern  = regexp.MustCompile("[A-Za-z]")
)

func isLikelyI18nKey(key string) bool {
	normalized := strings.TrimSpace(key)
	if normalized == "" || !i18nKeyLetterPattern.MatchString(normalized) {
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
	if !((firstRune >= 'a' && firstRune <= 'z') || (firstRune >= 'A' && firstRune <= 'Z')) {
		return false
	}

	for _, segment := range segments {
		trimmed := strings.TrimSpace(segment)
		if trimmed == "" || !i18nKeySegmentPattern.MatchString(trimmed) {
			return false
		}
	}

	last := strings.ToLower(segments[len(segments)-1])
	switch last {
	case "go", "ts", "tsx", "js", "jsx", "json", "csv", "txt", "png", "gif", "jpg", "jpeg", "svg", "ico", "css", "scss", "less", "html", "map", "md", "yml", "yaml":
		return false
	}

	if len(segments) <= 3 {
		switch first {
		case "db", "api", "www", "mail", "smtp", "cdn", "img", "static", "files", "localhost":
			switch last {
			case "com", "net", "org", "io", "cn", "dev", "app", "local", "internal", "lan":
				return false
			}
		}
	}

	return true
}

func scanI18nKeyReferenceFiles(targetKey string, newKey string, excludeCatalog bool) ([]I18nKeyReferenceFile, error) {
	normalizedTarget := strings.TrimSpace(targetKey)
	if normalizedTarget == "" {
		return []I18nKeyReferenceFile{}, nil
	}
	results := make([]I18nKeyReferenceFile, 0)
	for _, root := range resolveI18nScanRoots() {
		if err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil || info == nil || info.IsDir() {
				return nil
			}
			ext := strings.ToLower(filepath.Ext(path))
			if ext != ".go" && ext != ".ts" && ext != ".tsx" {
				return nil
			}
			if excludeCatalog && isIgnoredI18nUsageFile(path) {
				return nil
			}
			content, readErr := os.ReadFile(path)
			if readErr != nil {
				return nil
			}
			text := string(content)
			if !strings.Contains(text, normalizedTarget) {
				return nil
			}
			relativePath := path
			if cwd, cwdErr := os.Getwd(); cwdErr == nil {
				if rel, relErr := filepath.Rel(cwd, path); relErr == nil {
					relativePath = filepath.ToSlash(rel)
				}
			}
			matches := buildI18nKeyReferenceMatches(text, normalizedTarget, strings.TrimSpace(newKey))
			results = append(results, I18nKeyReferenceFile{
				Path:                 relativePath,
				MatchCount:           len(matches),
				SuggestedReplacement: strings.TrimSpace(newKey),
				Matches:              matches,
			})
			return nil
		}); err != nil {
			return nil, err
		}
	}
	sort.Slice(results, func(i, j int) bool { return results[i].Path < results[j].Path })
	return results, nil
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
		Where("module = ? AND `key` = ?", module, key).
		Updates(map[string]interface{}{
			"lifecycle_status":    I18nLifecycleStatusActive,
			"lifecycle_marked_at": nil,
		}).Error
}

func (s *I18nService) transitionUnusedLifecycle(module string, fromStatus string, toStatus string, requireConfirm bool) (*I18nUnusedLifecycleResp, error) {
	if requireConfirm {
		return nil, errors.New("i18n.lifecycle.transition.invalid")
	}
	return s.transitionUnusedLifecycleWithFilter(module, fromStatus, toStatus, nil)
}

func (s *I18nService) transitionUnusedLifecycleWithFilter(module string, fromStatus string, toStatus string, filter func(I18nUnusedKeyItem) bool) (*I18nUnusedLifecycleResp, error) {
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
	type target struct {
		module string
		key    string
	}
	targets := make([]target, 0)
	for _, item := range audit.UnusedKeys {
		if resp.Module != "" && item.Module != resp.Module {
			continue
		}
		if filter != nil && !filter(item) {
			continue
		}
		if normalizeI18nLifecycleStatus(item.LifecycleStatus) == fromStatus {
			targets = append(targets, target{module: item.Module, key: item.Key})
			resp.AffectedKeys = append(resp.AffectedKeys, item.Key)
		}
	}
	if len(targets) == 0 {
		return resp, nil
	}
	now := time.Now()
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		for _, item := range targets {
			updateResult := tx.Model(&SystemI18n{}).
				Where("module = ? AND `key` = ?", item.module, item.key).
				Updates(map[string]interface{}{
					"lifecycle_status":    toStatus,
					"lifecycle_marked_at": now,
				})
			if updateResult.Error != nil {
				return updateResult.Error
			}
			resp.AffectedRows += updateResult.RowsAffected
		}
		return nil
	}); err != nil {
		return nil, err
	}
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
