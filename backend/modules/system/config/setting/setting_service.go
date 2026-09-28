package config

import (
	"context"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/contracts"
	"github.com/duanxldragon/pantheon-base/backend/pkg/database"
	"github.com/duanxldragon/pantheon-base/backend/pkg/logging"
	"github.com/duanxldragon/pantheon-base/backend/pkg/tenant"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	settingKeyUITheme            = "ui.default_theme"
	settingKeyUploadDriver       = "upload.storage_driver"
	settingKeyI18nLanguage       = "i18n.default_language"
	settingKeyAppMode            = "platform.app_mode"
	settingKeyUploadAllowedTypes = "upload.allowed_types"
)

type settingCacheState struct {
	mu          sync.RWMutex
	listCache   map[string][]SettingResp
	groupCache  map[string]*SettingGroupResp
	publicCache map[string]*PublicSettingResp
	// watcherOnce guards the cross-instance invalidation goroutine so repeated
	// wiring (module re-register, tests) never spawns duplicate subscribers.
	watcherOnce sync.Once
}

// settingsRefreshChannel is the pubsub channel also published by
// notifyRuntimeSettingsChanged; keep both in sync.
const settingsRefreshChannel = "settings:refresh"

// SettingService serves platform/system settings with per-request tenant
// scoping.
type SettingService struct {
	db *gorm.DB
	// tenantCtx is the per-request tenant context (queue-5 settings slice).
	// Set from the Gin context by the handler via WithTenantContext; nil/compat
	// => platform-global rows only (legacy behavior preserved).
	tenantCtx *tenant.Context
	// cache is shared by pointer across all WithTenantContext-bound views so
	// the lock is never copied and invalidation reaches every view.
	cache *settingCacheState
}

// WithTenantContext returns a shallow view of the service bound to a request
// tenant context (same canary pattern as DictService). Under compat the shared
// service view is returned unchanged.
func (s *SettingService) WithTenantContext(ctx *tenant.Context) *SettingService {
	if ctx == nil || !ctx.IsMulti() {
		return s
	}
	return &SettingService{db: s.db, tenantCtx: ctx, cache: s.cache}
}

// tenantScope applies the settings read filter (contract §3.3 + scope matrix
// "tenant-overridable"): multi mode restricts to the request tenant's rows
// AND the platform-global population (tenant_id = 0) so overrides resolve on
// top of global defaults. Compat sees global rows only — unchanged.
func (s *SettingService) tenantScope() func(db *gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if s.tenantCtx == nil || !s.tenantCtx.IsMulti() {
			// Global rows only (tenant_id=0) is the compat population. Legacy
			// schemas always satisfied this implicitly; with override rows present
			// the filter must now be explicit.
			return db.Where("tenant_id = ?", tenant.PlatformGlobalTenantID)
		}
		return db.Where("tenant_id IN (?)", []uint64{tenant.PlatformGlobalTenantID, s.tenantCtx.TenantID})
	}
}

// tenantOwnerID is the tenant a new/updated row belongs to (write path: from
// context, never from the request body — contract §3.3). Compat => 0.
func (s *SettingService) tenantOwnerID() uint64 {
	if s.tenantCtx == nil || !s.tenantCtx.IsMulti() {
		return 0
	}
	return s.tenantCtx.TenantID
}

func NewSettingService(db *gorm.DB) *SettingService {
	return &SettingService{
		db: db,
		cache: &settingCacheState{
			listCache:   make(map[string][]SettingResp),
			groupCache:  make(map[string]*SettingGroupResp),
			publicCache: make(map[string]*PublicSettingResp),
		},
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

// WatchSettingsInvalidation subscribes to the cross-instance "settings:refresh"
// pubsub channel and clears this process's setting caches on every message, so
// an instance that did not serve the mutating request still drops its
// process-local cache (public/list/group). Idempotent; a no-op without Redis
// (single-instance or no-Redis deployments have nothing to sync).
func (s *SettingService) WatchSettingsInvalidation() {
	if database.RDB == nil || s.cache == nil {
		return
	}
	s.cache.watcherOnce.Do(func() {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					logging.Error("settings invalidation watcher panic", zap.Any("panic", r))
				}
			}()
			for {
				pubsub := database.RDB.Subscribe(context.TODO(), settingsRefreshChannel)
				for range pubsub.Channel() {
					s.invalidateSettingCache()
				}
				_ = pubsub.Close()
				logging.Warn("settings invalidation channel closed, reconnecting in 5s")
				time.Sleep(5 * time.Second)
			}
		}()
	})
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
	if err := s.migrateLegacySettingValue(settingKeyUploadAllowedTypes, settingValueUploadAllowedTypesLegacy, settingValueUploadAllowedTypesDefault); err != nil {
		return err
	}
	if err := s.migrateLegacySettingValue(settingKeyUploadAllowedTypes, settingValueUploadAllowedTypesArchives, settingValueUploadAllowedTypesDefault); err != nil {
		return err
	}
	if err := s.migrateLegacySettingValue("audit.session_cleanup_retention_options", "[7,30,90]", "[1,7,30]"); err != nil {
		return err
	}

	return nil
}

func (s *SettingService) bootstrapSettingSeed(item defaultSettingSeed) error {
	var count int64
	// Seeds are platform-global rows (tenant_id = 0) — never per-tenant copies.
	if err := s.db.Model(&SystemSetting{}).Where("setting_key = ? AND tenant_id = 0", item.SettingKey).Count(&count).Error; err != nil {
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

	cacheKey := s.settingListCacheKeyTenant(groupKey, module)
	s.cache.mu.RLock()
	if cached, ok := s.cache.listCache[cacheKey]; ok {
		s.cache.mu.RUnlock()
		return cloneSettingRespList(cached), nil
	}
	s.cache.mu.RUnlock()

	var rows []SystemSetting
	db := s.tenantScope()(s.db.Model(&SystemSetting{}))
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

	s.cache.mu.Lock()
	s.cache.listCache[cacheKey] = cloneSettingRespList(result)
	s.cache.mu.Unlock()
	return cloneSettingRespList(result), nil
}

func (s *SettingService) GetGroup(groupKey string) (*SettingGroupResp, error) {
	if s.db == nil {
		return nil, common.ErrDatabaseNotInitialized
	}

	groupKey = strings.TrimSpace(groupKey)
	if groupKey == "" {
		return nil, common.NewBadRequest("setting.group.invalid")
	}

	s.cache.mu.RLock()
	if cached, ok := s.cache.groupCache[s.settingGroupCacheKey(groupKey)]; ok {
		s.cache.mu.RUnlock()
		return cloneSettingGroupResp(cached), nil
	}
	s.cache.mu.RUnlock()

	items, err := s.List(&SettingListQuery{GroupKey: groupKey})
	if err != nil {
		return nil, err
	}

	group := &SettingGroupResp{GroupKey: groupKey, Items: items}
	s.cache.mu.Lock()
	s.cache.groupCache[s.settingGroupCacheKey(groupKey)] = cloneSettingGroupResp(group)
	s.cache.mu.Unlock()
	return cloneSettingGroupResp(group), nil
}

// GetByKey resolves a setting with tenant override semantics (queue-5
// settings slice, scope matrix "tenant-overridable"): in multi mode a tenant
// override row (tenant_id = ctx) wins over the platform-global default
// (tenant_id = 0). Compat resolves global rows only — unchanged.
func (s *SettingService) GetByKey(settingKey string) (string, error) {
	if s.db == nil {
		return "", common.ErrDatabaseNotInitialized
	}

	var rows []SystemSetting
	if err := s.tenantScope()(s.db).
		Where("setting_key = ?", strings.TrimSpace(settingKey)).
		Order("tenant_id asc").
		Find(&rows).Error; err != nil {
		return "", err
	}
	if len(rows) == 0 {
		return "", gorm.ErrRecordNotFound
	}
	// Ascending tenant_id puts the override (>0) after the global row (0);
	// the last match is therefore the tenant-specific value when present.
	row := rows[len(rows)-1]
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
		return nil, common.NewBadRequest("setting.group.invalid")
	}
	if req == nil || len(req.Items) == 0 {
		return nil, common.NewBadRequest("param.invalid")
	}

	if err := s.db.Transaction(func(tx *gorm.DB) error {
		for _, item := range req.Items {
			if err := updateSettingGroupItem(tx, groupKey, item, s.tenantOwnerID()); err != nil {
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

// updateSettingGroupItem applies a single setting update within an existing transaction.
// It returns nil (skipping the item) when an encrypted setting is cleared with an empty
// value, which is byte-for-byte equivalent to the original `continue` semantics.
// ownerTenantID is the tenant the write belongs to (0 = platform/global; contract
// §3.3 write ownership from context, never the request body).
func updateSettingGroupItem(tx *gorm.DB, groupKey string, item SettingUpdateItemReq, ownerTenantID uint64) error {
	settingKey := strings.TrimSpace(item.SettingKey)
	if settingKey == "" {
		return common.NewBadRequest("setting.key.required")
	}

	// The write targets the request tenant's effective row: the tenant override
	// row when one exists, otherwise the global row — which the tenant then
	// overrides by writing a tenant-owned copy (never mutating the global row).
	var rows []SystemSetting
	db := tx.Where("setting_key = ? AND group_key = ?", settingKey, groupKey)
	if ownerTenantID == 0 {
		db = db.Where("tenant_id = ?", 0)
	} else {
		db = db.Where("tenant_id IN (?)", []uint64{0, ownerTenantID})
	}
	if err := db.Order("tenant_id asc").Find(&rows).Error; err != nil {
		return err
	}
	if len(rows) == 0 {
		return gorm.ErrRecordNotFound
	}
	current := rows[len(rows)-1]

	nextValue := strings.TrimSpace(item.SettingValue)
	if current.IsEncrypted == 1 && nextValue == "" {
		return nil
	}

	normalizedValue, err := validateAndNormalizeSettingValue(current.SettingKey, current.ValueType, nextValue)
	if err != nil {
		return err
	}
	storedValue, err := prepareSettingStoredValue(normalizedValue, current.IsEncrypted)
	if err != nil {
		return err
	}
	if ownerTenantID == 0 || current.TenantID == ownerTenantID {
		// Platform write, or the tenant override row already exists: update in place.
		if err := tx.Model(&current).Update("setting_value", storedValue).Error; err != nil {
			return err
		}
		return nil
	}
	// First tenant override of a global row: create a tenant-owned copy. The
	// composite unique key (tenant_id, setting_key) makes duplicates impossible.
	return tx.Create(&SystemSetting{
		TenantID:     ownerTenantID,
		SettingKey:   current.SettingKey,
		SettingValue: storedValue,
		ValueType:    current.ValueType,
		GroupKey:     current.GroupKey,
		Module:       current.Module,
		IsPublic:     current.IsPublic,
		IsEncrypted:  current.IsEncrypted,
		Remark:       current.Remark,
	}).Error
}

func (s *SettingService) GetPublicSettings() (*PublicSettingResp, error) {
	if s.db == nil {
		return nil, common.ErrDatabaseNotInitialized
	}

	// Cache key is tenant-namespaced in multi mode (same canary pattern as
	// list/group caches) so tenant A/B never read each other's public rows
	// through the process-local cache.
	cacheKey := s.publicSettingsCacheKey()

	s.cache.mu.RLock()
	if cached, ok := s.cache.publicCache[cacheKey]; ok && cached != nil {
		s.cache.mu.RUnlock()
		return clonePublicSettingResp(cached), nil
	}
	s.cache.mu.RUnlock()

	// Tenant-scoped read (contract §3.3): multi mode resolves global defaults
	// plus the request tenant's overrides; compat sees global rows only.
	// Ordering by tenant_id ascending makes the last row per key the tenant
	// override (same resolution rule as GetByKey).
	var rows []SystemSetting
	if err := s.tenantScope()(s.db.Model(&SystemSetting{})).
		Where("is_public = ? AND is_encrypted = ?", 1, 0).
		Order("tenant_id asc, id asc").
		Find(&rows).Error; err != nil {
		return nil, err
	}

	settings := make(map[string]string, len(rows))
	for _, row := range rows {
		settings[row.SettingKey] = row.SettingValue
	}

	resp := &PublicSettingResp{Settings: settings}
	s.cache.mu.Lock()
	s.cache.publicCache[cacheKey] = clonePublicSettingResp(resp)
	s.cache.mu.Unlock()
	return clonePublicSettingResp(resp), nil
}

// publicSettingsCacheKey namespaces the public settings cache per tenant
// (multi mode). Compat and platform-global share the legacy base key.
func (s *SettingService) publicSettingsCacheKey() string {
	if s.tenantCtx == nil || !s.tenantCtx.IsMulti() {
		return "public"
	}
	return "t" + strconv.FormatUint(s.tenantCtx.TenantID, 10) + ":public"
}

func (s *SettingService) GetOverview() (*SettingOverviewResp, error) {
	if s.db == nil {
		return nil, common.ErrDatabaseNotInitialized
	}

	var rows []SystemSetting
	if err := s.tenantScope()(s.db).Order("group_key asc, id asc").Find(&rows).Error; err != nil {
		return nil, err
	}

	resp := &SettingOverviewResp{
		Issues: make([]SettingOverviewIssueResp, 0),
	}
	byKey := make(map[string]SystemSetting, len(rows))
	aggregateSettingOverviewCounts(rows, resp, byKey)

	resp.StorageDriver = safeSettingOverviewValue(byKey[settingKeyUploadDriver], "local")
	resp.DefaultLanguage = safeSettingOverviewValue(byKey[settingKeyI18nLanguage], "zh-CN")
	resp.DefaultTheme = safeSettingOverviewValue(byKey[settingKeyUITheme], "indigo")

	requiredKeys := buildRequiredSettingKeys(resp.StorageDriver)
	seenIssues := make(map[string]struct{})
	checkRequiredSettingKeys(requiredKeys, byKey, resp, seenIssues)
	checkPublicEncryptedConflicts(rows, resp, seenIssues)
	appendAllowedValueIssues(resp, seenIssues, byKey)

	resp.RiskCount = len(resp.Issues)
	return resp, nil
}

// aggregateSettingOverviewCounts tallies total/public/encrypted counts and indexes rows by key.
// It preserves the exact behavior of the original inline aggregation loop.
func aggregateSettingOverviewCounts(rows []SystemSetting, resp *SettingOverviewResp, byKey map[string]SystemSetting) {
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
}

// buildRequiredSettingKeys returns the ordered list of required setting keys for the
// current storage driver. S3 requires extra S3-specific keys; otherwise a local path key is required.
func buildRequiredSettingKeys(storageDriver string) []string {
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
		settingKeyUploadAllowedTypes,
	}
	if storageDriver == "s3" {
		requiredKeys = append(requiredKeys,
			"upload.s3_endpoint",
			"upload.s3_bucket",
			"upload.s3_access_key_id",
			"upload.s3_secret_access_key",
		)
	} else {
		requiredKeys = append(requiredKeys, "upload.local_path")
	}
	return requiredKeys
}

// checkRequiredSettingKeys flags any required key that is missing or has no value.
func checkRequiredSettingKeys(requiredKeys []string, byKey map[string]SystemSetting, resp *SettingOverviewResp, seenIssues map[string]struct{}) {
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
}

// checkPublicEncryptedConflicts flags settings that are simultaneously public and encrypted.
func checkPublicEncryptedConflicts(rows []SystemSetting, resp *SettingOverviewResp, seenIssues map[string]struct{}) {
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
}

// overviewAllowedValueCheck describes a single allowed-value validation for the overview.
type overviewAllowedValueCheck struct {
	Value      string
	Allowed    map[string]struct{}
	SettingKey string
	GroupKey   string
	Severity   string
	ReasonKey  string
}

// appendAllowedValueIssues flags overview issues for invalid storage driver, language, theme, and app mode.
func appendAllowedValueIssues(resp *SettingOverviewResp, seenIssues map[string]struct{}, byKey map[string]SystemSetting) {
	appMode := safeSettingOverviewValue(byKey[settingKeyAppMode], "enterprise")
	checks := []overviewAllowedValueCheck{
		{
			Value:      resp.StorageDriver,
			Allowed:    allowedStorageDriverValues,
			SettingKey: settingKeyUploadDriver,
			GroupKey:   "upload",
			Severity:   "critical",
			ReasonKey:  "setting.overview.issue.invalid_storage_driver",
		},
		{
			Value:      resp.DefaultLanguage,
			Allowed:    allowedLanguageValues,
			SettingKey: settingKeyI18nLanguage,
			GroupKey:   "i18n",
			Severity:   "warning",
			ReasonKey:  "setting.overview.issue.invalid_default_language",
		},
		{
			Value:      resp.DefaultTheme,
			Allowed:    allowedThemeValues,
			SettingKey: settingKeyUITheme,
			GroupKey:   "ui",
			Severity:   "warning",
			ReasonKey:  "setting.overview.issue.invalid_default_theme",
		},
		{
			Value:      appMode,
			Allowed:    allowedAppModeValues,
			SettingKey: settingKeyAppMode,
			GroupKey:   "platform",
			Severity:   "warning",
			ReasonKey:  "setting.overview.issue.invalid_app_mode",
		},
	}
	for _, check := range checks {
		if _, ok := check.Allowed[check.Value]; !ok {
			resp.Issues = appendSettingOverviewIssue(resp.Issues, seenIssues, SettingOverviewIssueResp{
				SettingKey: check.SettingKey,
				GroupKey:   check.GroupKey,
				Severity:   check.Severity,
				ReasonKey:  check.ReasonKey,
			})
		}
	}
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

// settingListCacheKeyTenant namespaces the list cache per tenant (multi mode)
// so two tenants never collide through the process cache (canary pattern).
func (s *SettingService) settingListCacheKeyTenant(groupKey, module string) string {
	base := settingListCacheKey(groupKey, module)
	if s.tenantCtx == nil || !s.tenantCtx.IsMulti() {
		return base
	}
	return "t" + strconv.FormatUint(s.tenantCtx.TenantID, 10) + ":" + base
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
		_ = database.RDB.Publish(context.TODO(), settingsRefreshChannel, "updated").Err()
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
