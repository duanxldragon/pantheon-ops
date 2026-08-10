package system

import (
	// Blank import enables the //go:embed directives that load the builtin seed resources.
	_ "embed"
	"errors"
	"log"
	"strings"

	"gopkg.in/yaml.v3"
	"gorm.io/gorm"
)

const (
	moduleLowcode        = "system.lowcode"
	keyMenuLowcode       = "system.menu.lowcode"
	keyMenuModules       = "system.menu.modules"
	keyMenuGenerator     = "system.menu.generator"
	moduleIAM            = "system.iam"
	moduleOrg            = "system.org"
	moduleConfig         = "system.config"
	moduleAuth           = "system.auth"
	moduleAudit          = "system.audit"
	keyMenuDashboard     = "system.menu.dashboard"
	keyMenuAccess        = "system.menu.access"
	keyMenuOrg           = "system.menu.org"
	keyMenuConfig        = "system.menu.config"
	keyMenuSecurity      = "system.menu.security"
	keyMenuUser          = "system.menu.user"
	keyMenuRole          = "system.menu.role"
	keyMenuPermission    = "system.menu.permission"
	keyMenuMenu          = "system.menu.menu"
	keyMenuDept          = "system.menu.dept"
	keyMenuPost          = "system.menu.post"
	keyMenuDict          = "system.menu.dict"
	keyMenuSetting       = "system.menu.setting"
	keyMenuI18n          = "system.menu.i18n"
	keyMenuLoginLog      = "system.menu.loginLog"
	keyMenuSession       = "system.menu.session"
	keyMenuSecurityEvent = "system.menu.securityEvent"
	keyMenuOperationLog  = "system.menu.operationLog"
)

type i18nLocaleKey struct {
	Locale string
	Key    string
}

type i18nCanonicalEntry struct {
	Module string
	Group  string
	Value  string
}

//go:embed seed_locales.yaml
var localeSeedYAML []byte

type localeSeedYAMLItem struct {
	Locale string `yaml:"locale"`
	Key    string `yaml:"key"`
	Module string `yaml:"module"`
	Group  string `yaml:"group"`
	Value  string `yaml:"value"`
}

type localeSeedsYAML struct {
	Locales []localeSeedYAMLItem `yaml:"locales"`
}

// loadedMenuLocaleEntries holds the locale entries loaded from YAML (with Go fallback).
var loadedMenuLocaleEntries map[i18nLocaleKey]i18nCanonicalEntry

func init() {
	loadedMenuLocaleEntries = loadLocaleSeedsFromYAML()
}

func loadLocaleSeedsFromYAML() map[i18nLocaleKey]i18nCanonicalEntry {
	if len(localeSeedYAML) == 0 {
		log.Println("[i18n] WARNING: locale seed YAML is empty, falling back to hardcoded defaults")
		return canonicalMenuLocaleEntries
	}
	var data localeSeedsYAML
	if err := yaml.Unmarshal(localeSeedYAML, &data); err != nil {
		log.Printf("[i18n] WARNING: failed to parse locale seed YAML: %v, falling back to hardcoded defaults", err)
		return canonicalMenuLocaleEntries
	}
	if len(data.Locales) == 0 {
		log.Println("[i18n] WARNING: locale seed YAML has no entries, falling back to hardcoded defaults")
		return canonicalMenuLocaleEntries
	}
	result := make(map[i18nLocaleKey]i18nCanonicalEntry, len(data.Locales))
	for _, item := range data.Locales {
		result[i18nLocaleKey{Locale: item.Locale, Key: item.Key}] = i18nCanonicalEntry{
			Module: item.Module,
			Group:  item.Group,
			Value:  item.Value,
		}
	}
	return result
}

// menuLocaleEntries returns the active menu locale entries (YAML-loaded with Go fallback).
func menuLocaleEntries() map[i18nLocaleKey]i18nCanonicalEntry {
	return loadedMenuLocaleEntries
}

var canonicalMenuLocaleEntries = map[i18nLocaleKey]i18nCanonicalEntry{
	{Locale: "zh-CN", Key: keyMenuDashboard}:     {Module: "platform", Group: "menu", Value: "工作台"},
	{Locale: "en-US", Key: keyMenuDashboard}:     {Module: "platform", Group: "menu", Value: "Workbench"},
	{Locale: "ja-JP", Key: keyMenuDashboard}:     {Module: "platform", Group: "menu", Value: "ワークベンチ"},
	{Locale: "ko-KR", Key: keyMenuDashboard}:     {Module: "platform", Group: "menu", Value: "워크벤치"},
	{Locale: "fr-FR", Key: keyMenuDashboard}:     {Module: "platform", Group: "menu", Value: "Espace de travail"},
	{Locale: "zh-CN", Key: keyMenuAccess}:        {Module: moduleIAM, Group: "menu", Value: "访问控制"},
	{Locale: "en-US", Key: keyMenuAccess}:        {Module: moduleIAM, Group: "menu", Value: "Access & Permissions"},
	{Locale: "ja-JP", Key: keyMenuAccess}:        {Module: moduleIAM, Group: "menu", Value: "アクセスと権限"},
	{Locale: "ko-KR", Key: keyMenuAccess}:        {Module: moduleIAM, Group: "menu", Value: "접근 및 권한"},
	{Locale: "fr-FR", Key: keyMenuAccess}:        {Module: moduleIAM, Group: "menu", Value: "Accès et autorisations"},
	{Locale: "zh-CN", Key: keyMenuOrg}:           {Module: moduleOrg, Group: "menu", Value: "组织架构"},
	{Locale: "en-US", Key: keyMenuOrg}:           {Module: moduleOrg, Group: "menu", Value: "Organizations"},
	{Locale: "ja-JP", Key: keyMenuOrg}:           {Module: moduleOrg, Group: "menu", Value: "組織"},
	{Locale: "ko-KR", Key: keyMenuOrg}:           {Module: moduleOrg, Group: "menu", Value: "조직"},
	{Locale: "fr-FR", Key: keyMenuOrg}:           {Module: moduleOrg, Group: "menu", Value: "Organisation"},
	{Locale: "zh-CN", Key: keyMenuConfig}:        {Module: moduleConfig, Group: "menu", Value: "平台配置"},
	{Locale: "en-US", Key: keyMenuConfig}:        {Module: moduleConfig, Group: "menu", Value: "System Configuration"},
	{Locale: "ja-JP", Key: keyMenuConfig}:        {Module: moduleConfig, Group: "menu", Value: "システム設定"},
	{Locale: "ko-KR", Key: keyMenuConfig}:        {Module: moduleConfig, Group: "menu", Value: "시스템 설정"},
	{Locale: "fr-FR", Key: keyMenuConfig}:        {Module: moduleConfig, Group: "menu", Value: "Configuration système"},
	{Locale: "zh-CN", Key: keyMenuLowcode}:       {Module: moduleLowcode, Group: "menu", Value: "低代码平台"},
	{Locale: "en-US", Key: keyMenuLowcode}:       {Module: moduleLowcode, Group: "menu", Value: "Low-Code"},
	{Locale: "ja-JP", Key: keyMenuLowcode}:       {Module: moduleLowcode, Group: "menu", Value: "モジュール開発"},
	{Locale: "ko-KR", Key: keyMenuLowcode}:       {Module: moduleLowcode, Group: "menu", Value: "모듈 개발"},
	{Locale: "fr-FR", Key: keyMenuLowcode}:       {Module: moduleLowcode, Group: "menu", Value: "Développement de modules"},
	{Locale: "zh-CN", Key: keyMenuSecurity}:      {Module: moduleAuth, Group: "menu", Value: "安全审计"},
	{Locale: "en-US", Key: keyMenuSecurity}:      {Module: moduleAuth, Group: "menu", Value: "Security & Audit"},
	{Locale: "ja-JP", Key: keyMenuSecurity}:      {Module: moduleAuth, Group: "menu", Value: "セキュリティと監査"},
	{Locale: "ko-KR", Key: keyMenuSecurity}:      {Module: moduleAuth, Group: "menu", Value: "보안 및 감사"},
	{Locale: "fr-FR", Key: keyMenuSecurity}:      {Module: moduleAuth, Group: "menu", Value: "Sécurité et audit"},
	{Locale: "zh-CN", Key: keyMenuUser}:          {Module: moduleIAM, Group: "menu", Value: "用户管理"},
	{Locale: "en-US", Key: keyMenuUser}:          {Module: moduleIAM, Group: "menu", Value: "Users"},
	{Locale: "ja-JP", Key: keyMenuUser}:          {Module: moduleIAM, Group: "menu", Value: "ユーザー"},
	{Locale: "ko-KR", Key: keyMenuUser}:          {Module: moduleIAM, Group: "menu", Value: "사용자"},
	{Locale: "fr-FR", Key: keyMenuUser}:          {Module: moduleIAM, Group: "menu", Value: "Utilisateurs"},
	{Locale: "zh-CN", Key: keyMenuRole}:          {Module: moduleIAM, Group: "menu", Value: "角色管理"},
	{Locale: "en-US", Key: keyMenuRole}:          {Module: moduleIAM, Group: "menu", Value: "Roles"},
	{Locale: "ja-JP", Key: keyMenuRole}:          {Module: moduleIAM, Group: "menu", Value: "ロール"},
	{Locale: "ko-KR", Key: keyMenuRole}:          {Module: moduleIAM, Group: "menu", Value: "역할"},
	{Locale: "fr-FR", Key: keyMenuRole}:          {Module: moduleIAM, Group: "menu", Value: "Rôles"},
	{Locale: "zh-CN", Key: keyMenuPermission}:    {Module: moduleIAM, Group: "menu", Value: "权限管理"},
	{Locale: "en-US", Key: keyMenuPermission}:    {Module: moduleIAM, Group: "menu", Value: "Permissions"},
	{Locale: "ja-JP", Key: keyMenuPermission}:    {Module: moduleIAM, Group: "menu", Value: "権限ポリシー"},
	{Locale: "ko-KR", Key: keyMenuPermission}:    {Module: moduleIAM, Group: "menu", Value: "권한 정책"},
	{Locale: "fr-FR", Key: keyMenuPermission}:    {Module: moduleIAM, Group: "menu", Value: "Politiques d'autorisation"},
	{Locale: "zh-CN", Key: keyMenuMenu}:          {Module: moduleIAM, Group: "menu", Value: "菜单管理"},
	{Locale: "en-US", Key: keyMenuMenu}:          {Module: moduleIAM, Group: "menu", Value: "Menus"},
	{Locale: "ja-JP", Key: keyMenuMenu}:          {Module: moduleIAM, Group: "menu", Value: "ナビゲーションメニュー"},
	{Locale: "ko-KR", Key: keyMenuMenu}:          {Module: moduleIAM, Group: "menu", Value: "탐색 메뉴"},
	{Locale: "fr-FR", Key: keyMenuMenu}:          {Module: moduleIAM, Group: "menu", Value: "Gestion des menus"},
	{Locale: "zh-CN", Key: keyMenuDept}:          {Module: moduleOrg, Group: "menu", Value: "部门管理"},
	{Locale: "en-US", Key: keyMenuDept}:          {Module: moduleOrg, Group: "menu", Value: "Departments"},
	{Locale: "ja-JP", Key: keyMenuDept}:          {Module: moduleOrg, Group: "menu", Value: "部門"},
	{Locale: "ko-KR", Key: keyMenuDept}:          {Module: moduleOrg, Group: "menu", Value: "부서"},
	{Locale: "fr-FR", Key: keyMenuDept}:          {Module: moduleOrg, Group: "menu", Value: "Départements"},
	{Locale: "zh-CN", Key: keyMenuPost}:          {Module: moduleOrg, Group: "menu", Value: "岗位管理"},
	{Locale: "en-US", Key: keyMenuPost}:          {Module: moduleOrg, Group: "menu", Value: "Positions"},
	{Locale: "ja-JP", Key: keyMenuPost}:          {Module: moduleOrg, Group: "menu", Value: "役職"},
	{Locale: "ko-KR", Key: keyMenuPost}:          {Module: moduleOrg, Group: "menu", Value: "직책"},
	{Locale: "fr-FR", Key: keyMenuPost}:          {Module: moduleOrg, Group: "menu", Value: "Postes"},
	{Locale: "zh-CN", Key: keyMenuDict}:          {Module: moduleConfig, Group: "menu", Value: "字典管理"},
	{Locale: "en-US", Key: keyMenuDict}:          {Module: moduleConfig, Group: "menu", Value: "Dictionaries"},
	{Locale: "ja-JP", Key: keyMenuDict}:          {Module: moduleConfig, Group: "menu", Value: "データ辞書"},
	{Locale: "ko-KR", Key: keyMenuDict}:          {Module: moduleConfig, Group: "menu", Value: "데이터 사전"},
	{Locale: "fr-FR", Key: keyMenuDict}:          {Module: moduleConfig, Group: "menu", Value: "Dictionnaires de données"},
	{Locale: "zh-CN", Key: keyMenuSetting}:       {Module: moduleConfig, Group: "menu", Value: "系统设置"},
	{Locale: "en-US", Key: keyMenuSetting}:       {Module: moduleConfig, Group: "menu", Value: "Settings"},
	{Locale: "ja-JP", Key: keyMenuSetting}:       {Module: moduleConfig, Group: "menu", Value: "システム設定"},
	{Locale: "ko-KR", Key: keyMenuSetting}:       {Module: moduleConfig, Group: "menu", Value: "시스템 설정"},
	{Locale: "fr-FR", Key: keyMenuSetting}:       {Module: moduleConfig, Group: "menu", Value: "Paramètres système"},
	{Locale: "zh-CN", Key: keyMenuI18n}:          {Module: moduleConfig, Group: "menu", Value: "国际化管理"},
	{Locale: "en-US", Key: keyMenuI18n}:          {Module: moduleConfig, Group: "menu", Value: "Internationalization"},
	{Locale: "ja-JP", Key: keyMenuI18n}:          {Module: moduleConfig, Group: "menu", Value: "国際化管理"},
	{Locale: "ko-KR", Key: keyMenuI18n}:          {Module: moduleConfig, Group: "menu", Value: "국제화 관리"},
	{Locale: "fr-FR", Key: keyMenuI18n}:          {Module: moduleConfig, Group: "menu", Value: "Gestion de l'internationalisation"},
	{Locale: "zh-CN", Key: keyMenuModules}:       {Module: moduleLowcode, Group: "menu", Value: "模块管理"},
	{Locale: "en-US", Key: keyMenuModules}:       {Module: moduleLowcode, Group: "menu", Value: "Modules"},
	{Locale: "ja-JP", Key: keyMenuModules}:       {Module: moduleLowcode, Group: "menu", Value: "モジュール登録"},
	{Locale: "ko-KR", Key: keyMenuModules}:       {Module: moduleLowcode, Group: "menu", Value: "모듈 등록"},
	{Locale: "fr-FR", Key: keyMenuModules}:       {Module: moduleLowcode, Group: "menu", Value: "Registre des modules"},
	{Locale: "zh-CN", Key: keyMenuGenerator}:     {Module: moduleLowcode, Group: "menu", Value: "模块生成器"},
	{Locale: "en-US", Key: keyMenuGenerator}:     {Module: moduleLowcode, Group: "menu", Value: "Code Generator"},
	{Locale: "ja-JP", Key: keyMenuGenerator}:     {Module: moduleLowcode, Group: "menu", Value: "モジュール生成"},
	{Locale: "ko-KR", Key: keyMenuGenerator}:     {Module: moduleLowcode, Group: "menu", Value: "모듈 생성기"},
	{Locale: "fr-FR", Key: keyMenuGenerator}:     {Module: moduleLowcode, Group: "menu", Value: "Générateur de modules"},
	{Locale: "zh-CN", Key: keyMenuLoginLog}:      {Module: moduleAuth, Group: "menu", Value: "登录日志"},
	{Locale: "en-US", Key: keyMenuLoginLog}:      {Module: moduleAuth, Group: "menu", Value: "Login Logs"},
	{Locale: "ja-JP", Key: keyMenuLoginLog}:      {Module: moduleAuth, Group: "menu", Value: "ログインログ"},
	{Locale: "ko-KR", Key: keyMenuLoginLog}:      {Module: moduleAuth, Group: "menu", Value: "로그인 로그"},
	{Locale: "fr-FR", Key: keyMenuLoginLog}:      {Module: moduleAuth, Group: "menu", Value: "Journaux de connexion"},
	{Locale: "zh-CN", Key: keyMenuSession}:       {Module: moduleAuth, Group: "menu", Value: "会话管理"},
	{Locale: "en-US", Key: keyMenuSession}:       {Module: moduleAuth, Group: "menu", Value: "Sessions"},
	{Locale: "ja-JP", Key: keyMenuSession}:       {Module: moduleAuth, Group: "menu", Value: "セッション"},
	{Locale: "ko-KR", Key: keyMenuSession}:       {Module: moduleAuth, Group: "menu", Value: "세션"},
	{Locale: "fr-FR", Key: keyMenuSession}:       {Module: moduleAuth, Group: "menu", Value: "Sessions"},
	{Locale: "zh-CN", Key: keyMenuSecurityEvent}: {Module: moduleAuth, Group: "menu", Value: "安全事件"},
	{Locale: "en-US", Key: keyMenuSecurityEvent}: {Module: moduleAuth, Group: "menu", Value: "Security Events"},
	{Locale: "ja-JP", Key: keyMenuSecurityEvent}: {Module: moduleAuth, Group: "menu", Value: "セキュリティイベント"},
	{Locale: "ko-KR", Key: keyMenuSecurityEvent}: {Module: moduleAuth, Group: "menu", Value: "보안 이벤트"},
	{Locale: "fr-FR", Key: keyMenuSecurityEvent}: {Module: moduleAuth, Group: "menu", Value: "Evenements de securite"},
	{Locale: "zh-CN", Key: keyMenuOperationLog}:  {Module: moduleAudit, Group: "menu", Value: "操作日志"},
	{Locale: "en-US", Key: keyMenuOperationLog}:  {Module: moduleAudit, Group: "menu", Value: "Operation Logs"},
	{Locale: "ja-JP", Key: keyMenuOperationLog}:  {Module: moduleAudit, Group: "menu", Value: "操作ログ"},
	{Locale: "ko-KR", Key: keyMenuOperationLog}:  {Module: moduleAudit, Group: "menu", Value: "작업 로그"},
	{Locale: "fr-FR", Key: keyMenuOperationLog}:  {Module: moduleAudit, Group: "menu", Value: "Journaux d'opération"},
}

func canonicalEntryFor(locale, key string) (i18nCanonicalEntry, bool) {
	entry, ok := menuLocaleEntries()[i18nLocaleKey{
		Locale: strings.TrimSpace(locale),
		Key:    strings.TrimSpace(key),
	}]
	return entry, ok
}

func (s *I18nService) BatchInsert(items []SystemI18n) error {
	for _, item := range items {
		if strings.TrimSpace(item.Group) == "" {
			item.Group = "messages"
		}

		var existing SystemI18n
		err := s.db.Where(condLocaleKeyEquals, item.Locale, item.Key).First(&existing).Error
		switch {
		case err == nil:
			updates := s.buildBatchInsertUpdates(existing, item)
			if err := s.applyLocaleUpdates(&existing, updates); err != nil {
				return err
			}
		case errors.Is(err, gorm.ErrRecordNotFound):
			if err := s.db.Create(&item).Error; err != nil {
				return err
			}
		default:
			return err
		}
	}
	return s.ReloadCache()
}

// buildBatchInsertUpdates composes the field updates for an existing i18n row
// during a batch insert, based on the incoming item and the canonical entry.
func (s *I18nService) buildBatchInsertUpdates(existing, item SystemI18n) map[string]interface{} {
	updates := map[string]interface{}{}
	s.fillBatchInsertFromItem(updates, existing, item)
	s.fillBatchInsertFromCanonical(updates, existing)
	return updates
}

// fillBatchInsertFromItem copies non-empty incoming fields into the update map
// only when the existing row leaves those fields empty.
func (s *I18nService) fillBatchInsertFromItem(updates map[string]interface{}, existing, item SystemI18n) {
	if strings.TrimSpace(existing.Module) == "" && strings.TrimSpace(item.Module) != "" {
		updates["module"] = item.Module
	}
	if strings.TrimSpace(existing.Group) == "" && item.Group != "" {
		updates["group_name"] = item.Group
	}
	if strings.TrimSpace(existing.Remark) == "" && strings.TrimSpace(item.Remark) != "" {
		updates["remark"] = item.Remark
	}
	if strings.TrimSpace(existing.Value) == "" && strings.TrimSpace(item.Value) != "" {
		updates["value"] = item.Value
	}
}

// fillBatchInsertFromCanonical overrides fields of the existing row with the
// canonical menu entry values when they differ (or the value is empty).
func (s *I18nService) fillBatchInsertFromCanonical(updates map[string]interface{}, existing SystemI18n) {
	canonical, ok := canonicalEntryFor(existing.Locale, existing.Key)
	if !ok {
		return
	}
	if strings.TrimSpace(existing.Module) != canonical.Module {
		updates["module"] = canonical.Module
	}
	if strings.TrimSpace(existing.Group) != canonical.Group {
		updates["group_name"] = canonical.Group
	}
	if strings.TrimSpace(existing.Value) == "" || strings.TrimSpace(existing.Value) != canonical.Value {
		updates["value"] = canonical.Value
	}
}

// applyLocaleUpdates applies the composed update map to the given existing row.
func (s *I18nService) applyLocaleUpdates(existing *SystemI18n, updates map[string]interface{}) error {
	if len(updates) == 0 {
		return nil
	}
	return s.db.Model(existing).Updates(updates).Error
}

func (s *I18nService) normalizeLocaleKeyDuplicates() error {
	if s.db == nil {
		return nil
	}

	type duplicateGroup struct {
		Locale string
		Key    string
		Count  int64
	}
	var groups []duplicateGroup
	if err := s.db.Model(&SystemI18n{}).
		Select("locale, `key`, COUNT(*) AS count").
		Group("locale, `key`").
		Having("COUNT(*) > 1").
		Find(&groups).Error; err != nil {
		return err
	}

	for _, group := range groups {
		if err := s.resolveDuplicateLocaleKey(group.Locale, group.Key); err != nil {
			return err
		}
	}

	return nil
}

// resolveDuplicateLocaleKey loads all rows for a duplicated (locale, key) pair,
// promotes a winner and removes the losing rows.
func (s *I18nService) resolveDuplicateLocaleKey(locale, key string) error {
	var rows []SystemI18n
	if err := s.db.Where(condLocaleKeyEquals, locale, key).
		Order("updated_at DESC").
		Order("id DESC").
		Find(&rows).Error; err != nil {
		return err
	}
	if len(rows) <= 1 {
		return nil
	}

	winner := pickLocaleKeyWinner(rows)
	if err := s.applyWinnerUpdates(winner, rows); err != nil {
		return err
	}
	return s.deleteLoserRows(winner, rows)
}

// applyWinnerUpdates writes the composed winner updates onto the winning row.
func (s *I18nService) applyWinnerUpdates(winner SystemI18n, rows []SystemI18n) error {
	updates := buildWinnerUpdates(winner, rows)
	if len(updates) == 0 {
		return nil
	}
	return s.db.Model(&SystemI18n{}).Where("id = ?", winner.ID).Updates(updates).Error
}

// deleteLoserRows removes every duplicated row except the chosen winner.
func (s *I18nService) deleteLoserRows(winner SystemI18n, rows []SystemI18n) error {
	deleteIDs := make([]uint64, 0, len(rows)-1)
	for _, row := range rows {
		if row.ID != winner.ID {
			deleteIDs = append(deleteIDs, row.ID)
		}
	}
	if len(deleteIDs) == 0 {
		return nil
	}
	return s.db.Where("id IN ?", deleteIDs).Delete(&SystemI18n{}).Error
}

func pickLocaleKeyWinner(rows []SystemI18n) SystemI18n {
	if len(rows) == 0 {
		return SystemI18n{}
	}
	if canonical, ok := canonicalEntryFor(rows[0].Locale, rows[0].Key); ok {
		for _, row := range rows {
			if strings.TrimSpace(row.Module) == canonical.Module {
				return row
			}
		}
	}
	for _, row := range rows {
		if strings.TrimSpace(row.Module) != "system" {
			return row
		}
	}
	return rows[0]
}

func buildWinnerUpdates(winner SystemI18n, rows []SystemI18n) map[string]interface{} {
	updates := map[string]interface{}{}
	addWinnerCanonicalUpdates(updates, winner)
	addWinnerRemarkUpdate(updates, winner, rows)
	addWinnerGroupUpdate(updates, winner, rows)
	return updates
}

// addWinnerCanonicalUpdates overrides the winner's fields with the canonical
// menu entry values when they differ.
func addWinnerCanonicalUpdates(updates map[string]interface{}, winner SystemI18n) {
	canonical, ok := canonicalEntryFor(winner.Locale, winner.Key)
	if !ok {
		return
	}
	if strings.TrimSpace(winner.Module) != canonical.Module {
		updates["module"] = canonical.Module
	}
	if strings.TrimSpace(winner.Group) != canonical.Group {
		updates["group_name"] = canonical.Group
	}
	if strings.TrimSpace(winner.Value) != canonical.Value {
		updates["value"] = canonical.Value
	}
}

// addWinnerRemarkUpdate fills the remark from the first row that has one.
func addWinnerRemarkUpdate(updates map[string]interface{}, winner SystemI18n, rows []SystemI18n) {
	if strings.TrimSpace(winner.Remark) != "" {
		return
	}
	for _, row := range rows {
		if strings.TrimSpace(row.Remark) != "" {
			updates["remark"] = strings.TrimSpace(row.Remark)
			return
		}
	}
}

// addWinnerGroupUpdate fills the group_name from the first row that has one.
func addWinnerGroupUpdate(updates map[string]interface{}, winner SystemI18n, rows []SystemI18n) {
	if strings.TrimSpace(winner.Group) != "" {
		return
	}
	for _, row := range rows {
		if strings.TrimSpace(row.Group) != "" {
			updates["group_name"] = strings.TrimSpace(row.Group)
			return
		}
	}
}

func (s *I18nService) ensureLocaleKeyUniqueIndex() error {
	if s.db == nil {
		return nil
	}
	if s.db.Migrator().HasIndex(&SystemI18n{}, "uidx_system_i18n_locale_key") {
		return nil
	}
	return s.db.Exec("CREATE UNIQUE INDEX uidx_system_i18n_locale_key ON system_i18n (locale, `key`)").Error
}

func (s *I18nService) ensureCanonicalMenuEntries() error {
	if s.db == nil {
		return nil
	}
	for localeKey, canonical := range menuLocaleEntries() {
		if err := s.ensureCanonicalMenuEntry(localeKey, canonical); err != nil {
			return err
		}
	}
	return nil
}

// ensureCanonicalMenuEntry reconciles a single canonical menu entry against the
// database: updating the existing row or creating it when missing.
func (s *I18nService) ensureCanonicalMenuEntry(localeKey i18nLocaleKey, canonical i18nCanonicalEntry) error {
	var existing SystemI18n
	err := s.db.Where(condLocaleKeyEquals, localeKey.Locale, localeKey.Key).First(&existing).Error
	switch {
	case err == nil:
		if err := s.applyCanonicalUpdates(&existing, canonical); err != nil {
			return err
		}
	case errors.Is(err, gorm.ErrRecordNotFound):
		if err := s.db.Create(&SystemI18n{
			Module: canonical.Module,
			Group:  canonical.Group,
			Key:    localeKey.Key,
			Locale: localeKey.Locale,
			Value:  canonical.Value,
		}).Error; err != nil {
			return err
		}
	default:
		return err
	}
	return nil
}

// applyCanonicalUpdates writes the canonical field overrides onto the given row.
func (s *I18nService) applyCanonicalUpdates(existing *SystemI18n, canonical i18nCanonicalEntry) error {
	updates := s.buildCanonicalUpdates(*existing, canonical)
	if len(updates) == 0 {
		return nil
	}
	return s.db.Model(existing).Updates(updates).Error
}

// buildCanonicalUpdates composes the field overrides for a canonical menu entry.
func (s *I18nService) buildCanonicalUpdates(existing SystemI18n, canonical i18nCanonicalEntry) map[string]interface{} {
	updates := map[string]interface{}{}
	if strings.TrimSpace(existing.Module) != canonical.Module {
		updates["module"] = canonical.Module
	}
	if strings.TrimSpace(existing.Group) != canonical.Group {
		updates["group_name"] = canonical.Group
	}
	if strings.TrimSpace(existing.Value) != canonical.Value {
		updates["value"] = canonical.Value
	}
	return updates
}
