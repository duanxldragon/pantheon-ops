package system

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"pantheon-ops/backend/pkg/impexp"

	"gorm.io/gorm"
)

type I18nService struct {
	db *gorm.DB
	mu sync.RWMutex

	cache map[string]map[string]string
}

type i18nLocaleKey struct {
	Locale string
	Key    string
}

type i18nCanonicalEntry struct {
	Module string
	Group  string
	Value  string
}

var canonicalMenuLocaleEntries = map[i18nLocaleKey]i18nCanonicalEntry{
	{Locale: "zh-CN", Key: "system.menu.dashboard"}:     {Module: "platform", Group: "menu", Value: "工作台"},
	{Locale: "en-US", Key: "system.menu.dashboard"}:     {Module: "platform", Group: "menu", Value: "Workbench"},
	{Locale: "ja-JP", Key: "system.menu.dashboard"}:     {Module: "platform", Group: "menu", Value: "ワークベンチ"},
	{Locale: "ko-KR", Key: "system.menu.dashboard"}:     {Module: "platform", Group: "menu", Value: "워크벤치"},
	{Locale: "fr-FR", Key: "system.menu.dashboard"}:     {Module: "platform", Group: "menu", Value: "Espace de travail"},
	{Locale: "zh-CN", Key: "system.menu.access"}:        {Module: "system.iam", Group: "menu", Value: "访问控制"},
	{Locale: "en-US", Key: "system.menu.access"}:        {Module: "system.iam", Group: "menu", Value: "Access & Permissions"},
	{Locale: "ja-JP", Key: "system.menu.access"}:        {Module: "system.iam", Group: "menu", Value: "アクセスと権限"},
	{Locale: "ko-KR", Key: "system.menu.access"}:        {Module: "system.iam", Group: "menu", Value: "접근 및 권한"},
	{Locale: "fr-FR", Key: "system.menu.access"}:        {Module: "system.iam", Group: "menu", Value: "Accès et autorisations"},
	{Locale: "zh-CN", Key: "system.menu.org"}:           {Module: "system.org", Group: "menu", Value: "组织架构"},
	{Locale: "en-US", Key: "system.menu.org"}:           {Module: "system.org", Group: "menu", Value: "Organizations"},
	{Locale: "ja-JP", Key: "system.menu.org"}:           {Module: "system.org", Group: "menu", Value: "組織"},
	{Locale: "ko-KR", Key: "system.menu.org"}:           {Module: "system.org", Group: "menu", Value: "조직"},
	{Locale: "fr-FR", Key: "system.menu.org"}:           {Module: "system.org", Group: "menu", Value: "Organisation"},
	{Locale: "zh-CN", Key: "system.menu.config"}:        {Module: "system.config", Group: "menu", Value: "平台配置"},
	{Locale: "en-US", Key: "system.menu.config"}:        {Module: "system.config", Group: "menu", Value: "System Configuration"},
	{Locale: "ja-JP", Key: "system.menu.config"}:        {Module: "system.config", Group: "menu", Value: "システム設定"},
	{Locale: "ko-KR", Key: "system.menu.config"}:        {Module: "system.config", Group: "menu", Value: "시스템 설정"},
	{Locale: "fr-FR", Key: "system.menu.config"}:        {Module: "system.config", Group: "menu", Value: "Configuration système"},
	{Locale: "zh-CN", Key: "system.menu.lowcode"}:       {Module: "system.lowcode", Group: "menu", Value: "低代码平台"},
	{Locale: "en-US", Key: "system.menu.lowcode"}:       {Module: "system.lowcode", Group: "menu", Value: "Low-Code"},
	{Locale: "ja-JP", Key: "system.menu.lowcode"}:       {Module: "system.lowcode", Group: "menu", Value: "モジュール開発"},
	{Locale: "ko-KR", Key: "system.menu.lowcode"}:       {Module: "system.lowcode", Group: "menu", Value: "모듈 개발"},
	{Locale: "fr-FR", Key: "system.menu.lowcode"}:       {Module: "system.lowcode", Group: "menu", Value: "Développement de modules"},
	{Locale: "zh-CN", Key: "system.menu.security"}:      {Module: "system.auth", Group: "menu", Value: "安全审计"},
	{Locale: "en-US", Key: "system.menu.security"}:      {Module: "system.auth", Group: "menu", Value: "Security & Audit"},
	{Locale: "ja-JP", Key: "system.menu.security"}:      {Module: "system.auth", Group: "menu", Value: "セキュリティと監査"},
	{Locale: "ko-KR", Key: "system.menu.security"}:      {Module: "system.auth", Group: "menu", Value: "보안 및 감사"},
	{Locale: "fr-FR", Key: "system.menu.security"}:      {Module: "system.auth", Group: "menu", Value: "Sécurité et audit"},
	{Locale: "zh-CN", Key: "system.menu.user"}:          {Module: "system.iam", Group: "menu", Value: "用户管理"},
	{Locale: "en-US", Key: "system.menu.user"}:          {Module: "system.iam", Group: "menu", Value: "Users"},
	{Locale: "ja-JP", Key: "system.menu.user"}:          {Module: "system.iam", Group: "menu", Value: "ユーザー"},
	{Locale: "ko-KR", Key: "system.menu.user"}:          {Module: "system.iam", Group: "menu", Value: "사용자"},
	{Locale: "fr-FR", Key: "system.menu.user"}:          {Module: "system.iam", Group: "menu", Value: "Utilisateurs"},
	{Locale: "zh-CN", Key: "system.menu.role"}:          {Module: "system.iam", Group: "menu", Value: "角色管理"},
	{Locale: "en-US", Key: "system.menu.role"}:          {Module: "system.iam", Group: "menu", Value: "Roles"},
	{Locale: "ja-JP", Key: "system.menu.role"}:          {Module: "system.iam", Group: "menu", Value: "ロール"},
	{Locale: "ko-KR", Key: "system.menu.role"}:          {Module: "system.iam", Group: "menu", Value: "역할"},
	{Locale: "fr-FR", Key: "system.menu.role"}:          {Module: "system.iam", Group: "menu", Value: "Rôles"},
	{Locale: "zh-CN", Key: "system.menu.permission"}:    {Module: "system.iam", Group: "menu", Value: "权限管理"},
	{Locale: "en-US", Key: "system.menu.permission"}:    {Module: "system.iam", Group: "menu", Value: "Permissions"},
	{Locale: "ja-JP", Key: "system.menu.permission"}:    {Module: "system.iam", Group: "menu", Value: "権限ポリシー"},
	{Locale: "ko-KR", Key: "system.menu.permission"}:    {Module: "system.iam", Group: "menu", Value: "권한 정책"},
	{Locale: "fr-FR", Key: "system.menu.permission"}:    {Module: "system.iam", Group: "menu", Value: "Politiques d’autorisation"},
	{Locale: "zh-CN", Key: "system.menu.menu"}:          {Module: "system.iam", Group: "menu", Value: "菜单管理"},
	{Locale: "en-US", Key: "system.menu.menu"}:          {Module: "system.iam", Group: "menu", Value: "Menus"},
	{Locale: "ja-JP", Key: "system.menu.menu"}:          {Module: "system.iam", Group: "menu", Value: "ナビゲーションメニュー"},
	{Locale: "ko-KR", Key: "system.menu.menu"}:          {Module: "system.iam", Group: "menu", Value: "탐색 메뉴"},
	{Locale: "fr-FR", Key: "system.menu.menu"}:          {Module: "system.iam", Group: "menu", Value: "Gestion des menus"},
	{Locale: "zh-CN", Key: "system.menu.dept"}:          {Module: "system.org", Group: "menu", Value: "部门管理"},
	{Locale: "en-US", Key: "system.menu.dept"}:          {Module: "system.org", Group: "menu", Value: "Departments"},
	{Locale: "ja-JP", Key: "system.menu.dept"}:          {Module: "system.org", Group: "menu", Value: "部門"},
	{Locale: "ko-KR", Key: "system.menu.dept"}:          {Module: "system.org", Group: "menu", Value: "부서"},
	{Locale: "fr-FR", Key: "system.menu.dept"}:          {Module: "system.org", Group: "menu", Value: "Départements"},
	{Locale: "zh-CN", Key: "system.menu.post"}:          {Module: "system.org", Group: "menu", Value: "岗位管理"},
	{Locale: "en-US", Key: "system.menu.post"}:          {Module: "system.org", Group: "menu", Value: "Positions"},
	{Locale: "ja-JP", Key: "system.menu.post"}:          {Module: "system.org", Group: "menu", Value: "役職"},
	{Locale: "ko-KR", Key: "system.menu.post"}:          {Module: "system.org", Group: "menu", Value: "직책"},
	{Locale: "fr-FR", Key: "system.menu.post"}:          {Module: "system.org", Group: "menu", Value: "Postes"},
	{Locale: "zh-CN", Key: "system.menu.dict"}:          {Module: "system.config", Group: "menu", Value: "字典管理"},
	{Locale: "en-US", Key: "system.menu.dict"}:          {Module: "system.config", Group: "menu", Value: "Dictionaries"},
	{Locale: "ja-JP", Key: "system.menu.dict"}:          {Module: "system.config", Group: "menu", Value: "データ辞書"},
	{Locale: "ko-KR", Key: "system.menu.dict"}:          {Module: "system.config", Group: "menu", Value: "데이터 사전"},
	{Locale: "fr-FR", Key: "system.menu.dict"}:          {Module: "system.config", Group: "menu", Value: "Dictionnaires de données"},
	{Locale: "zh-CN", Key: "system.menu.setting"}:       {Module: "system.config", Group: "menu", Value: "系统设置"},
	{Locale: "en-US", Key: "system.menu.setting"}:       {Module: "system.config", Group: "menu", Value: "Settings"},
	{Locale: "ja-JP", Key: "system.menu.setting"}:       {Module: "system.config", Group: "menu", Value: "システム設定"},
	{Locale: "ko-KR", Key: "system.menu.setting"}:       {Module: "system.config", Group: "menu", Value: "시스템 설정"},
	{Locale: "fr-FR", Key: "system.menu.setting"}:       {Module: "system.config", Group: "menu", Value: "Paramètres système"},
	{Locale: "zh-CN", Key: "system.menu.i18n"}:          {Module: "system.config", Group: "menu", Value: "国际化管理"},
	{Locale: "en-US", Key: "system.menu.i18n"}:          {Module: "system.config", Group: "menu", Value: "Internationalization"},
	{Locale: "ja-JP", Key: "system.menu.i18n"}:          {Module: "system.config", Group: "menu", Value: "国際化管理"},
	{Locale: "ko-KR", Key: "system.menu.i18n"}:          {Module: "system.config", Group: "menu", Value: "국제화 관리"},
	{Locale: "fr-FR", Key: "system.menu.i18n"}:          {Module: "system.config", Group: "menu", Value: "Gestion de l'internationalisation"},
	{Locale: "zh-CN", Key: "system.menu.modules"}:       {Module: "system.lowcode", Group: "menu", Value: "模块管理"},
	{Locale: "en-US", Key: "system.menu.modules"}:       {Module: "system.lowcode", Group: "menu", Value: "Modules"},
	{Locale: "ja-JP", Key: "system.menu.modules"}:       {Module: "system.lowcode", Group: "menu", Value: "モジュール登録"},
	{Locale: "ko-KR", Key: "system.menu.modules"}:       {Module: "system.lowcode", Group: "menu", Value: "모듈 등록"},
	{Locale: "fr-FR", Key: "system.menu.modules"}:       {Module: "system.lowcode", Group: "menu", Value: "Registre des modules"},
	{Locale: "zh-CN", Key: "system.menu.generator"}:     {Module: "system.lowcode", Group: "menu", Value: "模块生成器"},
	{Locale: "en-US", Key: "system.menu.generator"}:     {Module: "system.lowcode", Group: "menu", Value: "Code Generator"},
	{Locale: "ja-JP", Key: "system.menu.generator"}:     {Module: "system.lowcode", Group: "menu", Value: "モジュール生成"},
	{Locale: "ko-KR", Key: "system.menu.generator"}:     {Module: "system.lowcode", Group: "menu", Value: "모듈 생성기"},
	{Locale: "fr-FR", Key: "system.menu.generator"}:     {Module: "system.lowcode", Group: "menu", Value: "Générateur de modules"},
	{Locale: "zh-CN", Key: "system.menu.loginLog"}:      {Module: "system.auth", Group: "menu", Value: "登录日志"},
	{Locale: "en-US", Key: "system.menu.loginLog"}:      {Module: "system.auth", Group: "menu", Value: "Login Logs"},
	{Locale: "ja-JP", Key: "system.menu.loginLog"}:      {Module: "system.auth", Group: "menu", Value: "ログインログ"},
	{Locale: "ko-KR", Key: "system.menu.loginLog"}:      {Module: "system.auth", Group: "menu", Value: "로그인 로그"},
	{Locale: "fr-FR", Key: "system.menu.loginLog"}:      {Module: "system.auth", Group: "menu", Value: "Journaux de connexion"},
	{Locale: "zh-CN", Key: "system.menu.session"}:       {Module: "system.auth", Group: "menu", Value: "会话管理"},
	{Locale: "en-US", Key: "system.menu.session"}:       {Module: "system.auth", Group: "menu", Value: "Sessions"},
	{Locale: "ja-JP", Key: "system.menu.session"}:       {Module: "system.auth", Group: "menu", Value: "セッション"},
	{Locale: "ko-KR", Key: "system.menu.session"}:       {Module: "system.auth", Group: "menu", Value: "세션"},
	{Locale: "fr-FR", Key: "system.menu.session"}:       {Module: "system.auth", Group: "menu", Value: "Sessions"},
	{Locale: "zh-CN", Key: "system.menu.securityEvent"}: {Module: "system.auth", Group: "menu", Value: "安全事件"},
	{Locale: "en-US", Key: "system.menu.securityEvent"}: {Module: "system.auth", Group: "menu", Value: "Security Events"},
	{Locale: "ja-JP", Key: "system.menu.securityEvent"}: {Module: "system.auth", Group: "menu", Value: "セキュリティイベント"},
	{Locale: "ko-KR", Key: "system.menu.securityEvent"}: {Module: "system.auth", Group: "menu", Value: "보안 이벤트"},
	{Locale: "fr-FR", Key: "system.menu.securityEvent"}: {Module: "system.auth", Group: "menu", Value: "Evenements de securite"},
	{Locale: "zh-CN", Key: "system.menu.operationLog"}:  {Module: "system.audit", Group: "menu", Value: "操作日志"},
	{Locale: "en-US", Key: "system.menu.operationLog"}:  {Module: "system.audit", Group: "menu", Value: "Operation Logs"},
	{Locale: "ja-JP", Key: "system.menu.operationLog"}:  {Module: "system.audit", Group: "menu", Value: "操作ログ"},
	{Locale: "ko-KR", Key: "system.menu.operationLog"}:  {Module: "system.audit", Group: "menu", Value: "작업 로그"},
	{Locale: "fr-FR", Key: "system.menu.operationLog"}:  {Module: "system.audit", Group: "menu", Value: "Journaux d'opération"},
}

func NewI18nService(db *gorm.DB) *I18nService {
	s := &I18nService{
		db:    db,
		cache: make(map[string]map[string]string),
	}
	if db != nil && db.Migrator().HasTable("system_i18n") {
		_ = s.ReloadCache()
	}
	return s
}

func (s *I18nService) Migrate() error {
	if err := s.db.AutoMigrate(&SystemI18n{}); err != nil {
		return err
	}
	if err := s.normalizeLocaleKeyDuplicates(); err != nil {
		return err
	}
	if err := s.ensureLocaleKeyUniqueIndex(); err != nil {
		return err
	}
	if err := s.ensureCanonicalMenuEntries(); err != nil {
		return err
	}
	return nil
}

func (s *I18nService) GetLangPack(locale string) (map[string]string, error) {
	normalized := strings.TrimSpace(locale)
	if normalized == "" {
		normalized = "zh-CN"
	}

	basePack, err := s.getRawLangPack("zh-CN")
	if err != nil {
		return nil, err
	}
	builtinBasePack := getBuiltinLocalePack("zh-CN")
	for key, value := range basePack {
		if !hasStoredLocaleValue(value) {
			continue
		}
		builtinBasePack[key] = value
	}
	if normalized == "zh-CN" {
		return builtinBasePack, nil
	}

	targetPack, err := s.getRawLangPack(normalized)
	if err != nil {
		return nil, err
	}
	builtinTargetPack := getBuiltinLocalePack(normalized)
	for key, value := range targetPack {
		if !hasStoredLocaleValue(value) {
			continue
		}
		builtinTargetPack[key] = value
	}

	merged := cloneLangPack(builtinBasePack)
	for key, value := range builtinTargetPack {
		if strings.TrimSpace(value) == "" {
			continue
		}
		merged[key] = value
	}
	return merged, nil
}

func (s *I18nService) List(query *I18nQuery) (*I18nPageResp, error) {
	query = normalizeI18nQuery(query)

	db := s.db.Model(&SystemI18n{})
	if query.Module != "" {
		db = db.Where("module = ?", query.Module)
	}
	if query.Group != "" {
		db = db.Where("group_name = ?", query.Group)
	}
	if query.Locale != "" {
		db = db.Where("locale = ?", query.Locale)
	}
	if query.Key != "" {
		db = db.Where("`key` LIKE ?", "%"+query.Key+"%")
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}

	var items []SystemI18n
	orderField := map[string]string{
		"createdAt": "created_at",
		"updatedAt": "updated_at",
		"key":       "`key`",
		"module":    "module",
		"locale":    "locale",
	}[query.SortBy]
	if orderField == "" {
		orderField = "locale"
	}
	orderDirection := "ASC"
	if strings.EqualFold(query.SortOrder, "desc") {
		orderDirection = "DESC"
	}
	err := db.Order(fmt.Sprintf("%s %s", orderField, orderDirection)).Order("module ASC").Order("`key` ASC").
		Offset((query.Page - 1) * query.PageSize).
		Limit(query.PageSize).
		Find(&items).Error
	respItems := make([]I18nResp, 0, len(items))
	for _, item := range items {
		respItems = append(respItems, toI18nResp(item))
	}
	return &I18nPageResp{Items: respItems, Total: total, Page: query.Page, PageSize: query.PageSize}, err
}

func (s *I18nService) Get(id uint64) (*I18nResp, error) {
	var item SystemI18n
	if err := s.db.First(&item, id).Error; err != nil {
		return nil, err
	}
	resp := toI18nResp(item)
	return &resp, nil
}

func (s *I18nService) Create(req *I18nCreateReq) (*I18nResp, error) {
	row := SystemI18n{
		Module: strings.TrimSpace(req.Module),
		Group:  strings.TrimSpace(req.Group),
		Key:    strings.TrimSpace(req.Key),
		Locale: strings.TrimSpace(req.Locale),
		Value:  strings.TrimSpace(req.Value),
		Remark: strings.TrimSpace(req.Remark),
	}
	if row.Group == "" {
		row.Group = "messages"
	}
	if row.Module == "" || row.Key == "" || row.Locale == "" || row.Value == "" {
		return nil, errors.New("i18n.create.invalid")
	}

	var count int64
	if err := s.db.Model(&SystemI18n{}).
		Where("locale = ? AND `key` = ?", row.Locale, row.Key).
		Count(&count).Error; err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, errors.New("i18n.key.duplicate")
	}

	if err := s.db.Create(&row).Error; err != nil {
		return nil, err
	}
	var created SystemI18n
	if err := s.db.Where("locale = ? AND `key` = ?", row.Locale, row.Key).First(&created).Error; err != nil {
		return nil, err
	}
	resp := toI18nResp(created)
	return &resp, s.ReloadCache()
}

func (s *I18nService) Update(id uint64, req *I18nUpdateReq) error {
	req.Value = strings.TrimSpace(req.Value)
	req.Remark = strings.TrimSpace(req.Remark)
	if req.Value == "" {
		return errors.New("i18n.value.required")
	}

	var t SystemI18n
	if err := s.db.First(&t, id).Error; err != nil {
		return err
	}
	if err := s.db.Model(&t).Updates(map[string]interface{}{
		"value":  req.Value,
		"remark": req.Remark,
	}).Error; err != nil {
		return err
	}
	return s.ReloadCache()
}

func (s *I18nService) Delete(id uint64) error {
	var item SystemI18n
	if err := s.db.First(&item, id).Error; err != nil {
		return err
	}
	if err := s.db.Delete(&item).Error; err != nil {
		return err
	}
	return s.ReloadCache()
}

func (s *I18nService) DeleteBatch(ids []uint64) error {
	if len(ids) == 0 {
		return nil
	}
	if err := s.db.Where("id IN ?", ids).Delete(&SystemI18n{}).Error; err != nil {
		return err
	}
	return s.ReloadCache()
}

func (s *I18nService) Export(query *I18nQuery) (*impexp.CSVFile, error) {
	query = normalizeI18nQuery(query)

	db := s.db.Model(&SystemI18n{})
	if query.Module != "" {
		db = db.Where("module = ?", query.Module)
	}
	if query.Group != "" {
		db = db.Where("group_name = ?", query.Group)
	}
	if query.Locale != "" {
		db = db.Where("locale = ?", query.Locale)
	}
	if query.Key != "" {
		db = db.Where("`key` LIKE ?", "%"+query.Key+"%")
	}

	var rows []SystemI18n
	if err := db.Order("locale ASC").Order("module ASC").Order("`key` ASC").Find(&rows).Error; err != nil {
		return nil, err
	}

	result := make([][]string, 0, len(rows))
	for _, row := range rows {
		result = append(result, []string{
			row.Module,
			row.Group,
			row.Key,
			row.Locale,
			row.Value,
			row.Remark,
			row.CreatedAt.Format("2006-01-02 15:04:05"),
			row.UpdatedAt.Format("2006-01-02 15:04:05"),
		})
	}

	return &impexp.CSVFile{
		Filename: "system-i18n-export.csv",
		Headers:  []string{"module", "group", "key", "locale", "value", "remark", "createdAt", "updatedAt"},
		Rows:     result,
	}, nil
}

func (s *I18nService) BuildImportTemplate() *impexp.CSVFile {
	return &impexp.CSVFile{
		Filename: "system-i18n-import-template.csv",
		Headers:  []string{"module", "group", "key", "locale", "value", "remark"},
		Rows: [][]string{
			{"#说明：保留第一行表头；group 为空时默认 messages；module/key/locale/value 必填；已存在记录按 locale + key 更新 value/remark/group；若 module 与现有记录归属不一致，该行会被阻断。", "", "", "", "", ""},
			{"system.config", "messages", "i18n.sample.key", "zh-CN", "示例文案", "sample"},
			{"system.config", "messages", "i18n.sample.key", "en-US", "Sample Text", "sample"},
		},
	}
}

func (s *I18nService) Import(records [][]string) (*impexp.ImportResult, error) {
	result := &impexp.ImportResult{
		Applied: false,
		Errors:  []impexp.ImportError{},
	}
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}
	if len(records) == 0 {
		impexp.AppendImportError(result, 0, "file", "import.file.empty")
		return result, nil
	}

	headerIndex := make(map[string]int, len(records[0]))
	for index, header := range records[0] {
		headerIndex[strings.TrimSpace(header)] = index
	}
	requiredHeaders := []string{"module", "group", "key", "locale", "value", "remark"}
	for _, header := range requiredHeaders {
		if _, ok := headerIndex[header]; !ok {
			impexp.AppendImportError(result, 0, header, "import.header.missing")
		}
	}
	if result.Failed > 0 {
		return result, nil
	}

	type importRow struct {
		module string
		group  string
		key    string
		locale string
		value  string
		remark string
	}

	type validatedImportRow struct {
		importRow
		rowNumber int
	}

	rows := make([]validatedImportRow, 0, len(records)-1)
	seen := make(map[string]int, len(records)-1)
	for rowIndex := 1; rowIndex < len(records); rowIndex++ {
		record := records[rowIndex]
		if impexp.IsCSVRecordEmpty(record) || impexp.IsCSVRecordBlank(record) {
			continue
		}
		rowNumber := rowIndex + 1
		module := strings.TrimSpace(impexp.ReadCSVField(record, headerIndex, "module"))
		group := strings.TrimSpace(impexp.ReadCSVField(record, headerIndex, "group"))
		key := strings.TrimSpace(impexp.ReadCSVField(record, headerIndex, "key"))
		locale := strings.TrimSpace(impexp.ReadCSVField(record, headerIndex, "locale"))
		value := strings.TrimSpace(impexp.ReadCSVField(record, headerIndex, "value"))
		remark := strings.TrimSpace(impexp.ReadCSVField(record, headerIndex, "remark"))
		if group == "" {
			group = "messages"
		}

		if module == "" {
			impexp.AppendImportError(result, rowNumber, "module", "i18n.module.required")
		}
		if key == "" {
			impexp.AppendImportError(result, rowNumber, "key", "i18n.key.required")
		}
		if locale == "" {
			impexp.AppendImportError(result, rowNumber, "locale", "i18n.locale.required")
		}
		if value == "" {
			impexp.AppendImportError(result, rowNumber, "value", "i18n.value.required")
		}

		duplicateKey := fmt.Sprintf("%s|%s|%s", module, key, locale)
		if firstRow, ok := seen[duplicateKey]; ok {
			impexp.AppendImportError(result, rowNumber, "key", fmt.Sprintf("import.duplicate.row.%d", firstRow))
		} else {
			seen[duplicateKey] = rowNumber
		}

		rows = append(rows, validatedImportRow{
			importRow: importRow{
				module: module,
				group:  group,
				key:    key,
				locale: locale,
				value:  value,
				remark: remark,
			},
			rowNumber: rowNumber,
		})
	}
	if result.Failed > 0 {
		return result, nil
	}

	if err := s.db.Transaction(func(tx *gorm.DB) error {
		for _, row := range rows {
			var existing SystemI18n
			err := tx.Where("locale = ? AND `key` = ?", row.locale, row.key).First(&existing).Error
			switch {
			case err == nil:
				if strings.TrimSpace(existing.Module) != "" && strings.TrimSpace(existing.Module) != row.module {
					impexp.AppendImportError(result, row.rowNumber, "module", fmt.Sprintf("import.conflict.owner.%s", existing.Module))
					continue
				}
				if err := tx.Model(&existing).Updates(map[string]interface{}{
					"module":     row.module,
					"group_name": row.group,
					"value":      row.value,
					"remark":     row.remark,
				}).Error; err != nil {
					return err
				}
				result.Updated++
			case errors.Is(err, gorm.ErrRecordNotFound):
				if err := tx.Create(&SystemI18n{
					Module: row.module,
					Group:  row.group,
					Key:    row.key,
					Locale: row.locale,
					Value:  row.value,
					Remark: row.remark,
				}).Error; err != nil {
					return err
				}
				result.Created++
			default:
				return err
			}
		}
		return nil
	}); err != nil {
		return nil, err
	}

	result.Applied = result.Created > 0 || result.Updated > 0
	return result, s.ReloadCache()
}

func (s *I18nService) ReloadCache() error {
	var items []SystemI18n
	if err := s.db.Order("locale ASC").Order("module ASC").Order("`key` ASC").Find(&items).Error; err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cache = make(map[string]map[string]string)
	for _, item := range items {
		if _, ok := s.cache[item.Locale]; !ok {
			s.cache[item.Locale] = make(map[string]string)
		}
		s.cache[item.Locale][item.Key] = item.Value
	}
	return nil
}

func (s *I18nService) ReloadLocales(locales []string) error {
	if len(locales) == 0 {
		return s.ReloadCache()
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, locale := range locales {
		normalized := strings.TrimSpace(locale)
		if normalized == "" {
			continue
		}
		var items []SystemI18n
		if err := s.db.Where("locale = ?", normalized).Order("module ASC").Order("`key` ASC").Find(&items).Error; err != nil {
			return err
		}
		pack := make(map[string]string, len(items))
		for _, item := range items {
			pack[item.Key] = item.Value
		}
		s.cache[normalized] = pack
	}
	return nil
}

func (s *I18nService) LoadAndCache(locale string) (map[string]string, error) {
	var items []SystemI18n
	if err := s.db.Where("locale = ?", locale).Order("module ASC").Order("`key` ASC").Find(&items).Error; err != nil {
		return nil, err
	}
	pack := make(map[string]string)
	for _, item := range items {
		pack[item.Key] = item.Value
	}
	s.mu.Lock()
	s.cache[locale] = cloneLangPack(pack)
	s.mu.Unlock()
	return cloneLangPack(pack), nil
}

func (s *I18nService) getRawLangPack(locale string) (map[string]string, error) {
	s.mu.RLock()
	if pack, ok := s.cache[locale]; ok {
		s.mu.RUnlock()
		return cloneLangPack(pack), nil
	}
	s.mu.RUnlock()
	return s.LoadAndCache(locale)
}

func (s *I18nService) BatchInsert(items []SystemI18n) error {
	for _, item := range items {
		if strings.TrimSpace(item.Group) == "" {
			item.Group = "messages"
		}

		var existing SystemI18n
		err := s.db.Where("locale = ? AND `key` = ?", item.Locale, item.Key).First(&existing).Error
		switch {
		case err == nil:
			updates := map[string]interface{}{}
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
			if canonical, ok := canonicalEntryFor(existing.Locale, existing.Key); ok {
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
			if len(updates) > 0 {
				if err := s.db.Model(&existing).Updates(updates).Error; err != nil {
					return err
				}
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

func canonicalEntryFor(locale string, key string) (i18nCanonicalEntry, bool) {
	entry, ok := canonicalMenuLocaleEntries[i18nLocaleKey{
		Locale: strings.TrimSpace(locale),
		Key:    strings.TrimSpace(key),
	}]
	return entry, ok
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
		var rows []SystemI18n
		if err := s.db.Where("locale = ? AND `key` = ?", group.Locale, group.Key).
			Order("updated_at DESC").
			Order("id DESC").
			Find(&rows).Error; err != nil {
			return err
		}
		if len(rows) <= 1 {
			continue
		}

		winner := pickLocaleKeyWinner(rows)
		updates := buildWinnerUpdates(winner, rows)
		if len(updates) > 0 {
			if err := s.db.Model(&SystemI18n{}).Where("id = ?", winner.ID).Updates(updates).Error; err != nil {
				return err
			}
		}

		deleteIDs := make([]uint64, 0, len(rows)-1)
		for _, row := range rows {
			if row.ID != winner.ID {
				deleteIDs = append(deleteIDs, row.ID)
			}
		}
		if len(deleteIDs) > 0 {
			if err := s.db.Where("id IN ?", deleteIDs).Delete(&SystemI18n{}).Error; err != nil {
				return err
			}
		}
	}

	return nil
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

	if canonical, ok := canonicalEntryFor(winner.Locale, winner.Key); ok {
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

	if strings.TrimSpace(winner.Remark) == "" {
		for _, row := range rows {
			if strings.TrimSpace(row.Remark) != "" {
				updates["remark"] = strings.TrimSpace(row.Remark)
				break
			}
		}
	}

	if strings.TrimSpace(winner.Group) == "" {
		for _, row := range rows {
			if strings.TrimSpace(row.Group) != "" {
				updates["group_name"] = strings.TrimSpace(row.Group)
				break
			}
		}
	}

	return updates
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
	for localeKey, canonical := range canonicalMenuLocaleEntries {
		var existing SystemI18n
		err := s.db.Where("locale = ? AND `key` = ?", localeKey.Locale, localeKey.Key).First(&existing).Error
		switch {
		case err == nil:
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
			if len(updates) > 0 {
				if err := s.db.Model(&existing).Updates(updates).Error; err != nil {
					return err
				}
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
	}
	return nil
}

func (s *I18nService) ScanErrorKeys() ([]string, error) {
	return scanI18nKeys(false)
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
		entryCount        int64
		keys              map[string]struct{}
		unusedKeys        map[string]struct{}
		duplicateKeys     map[string]struct{}
		missingLocaleKeys map[string]struct{}
		placeholderCount  int64
		stalePlaceholders int64
		observingKeys     map[string]struct{}
		archivedKeys      map[string]struct{}
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
				keys:              make(map[string]struct{}),
				unusedKeys:        make(map[string]struct{}),
				duplicateKeys:     make(map[string]struct{}),
				missingLocaleKeys: make(map[string]struct{}),
				observingKeys:     make(map[string]struct{}),
				archivedKeys:      make(map[string]struct{}),
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
			EligibleForDelete:  meta.lifecycleStatus == I18nLifecycleStatusArchived,
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
			Module:                module,
			EntryCount:            item.entryCount,
			KeyCount:              int64(len(item.keys)),
			UnusedKeyCount:        int64(len(item.unusedKeys)),
			DuplicateKeyCount:     int64(len(item.duplicateKeys)),
			MissingLocaleCount:    int64(len(item.missingLocaleKeys)),
			PlaceholderCount:      item.placeholderCount,
			StalePlaceholderCount: item.stalePlaceholders,
			ObservingKeyCount:     int64(len(item.observingKeys)),
			ArchivedKeyCount:      int64(len(item.archivedKeys)),
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
		if item.EligibleForDelete {
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
		return resp, errors.New("database.not_initialized")
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

func isI18nPlaceholderValue(value string) bool {
	trimmed := strings.TrimSpace(value)
	return strings.HasPrefix(trimmed, "[") && strings.HasSuffix(trimmed, "]")
}

func hasStoredLocaleValue(value string) bool {
	trimmed := strings.TrimSpace(value)
	return trimmed != "" && !isI18nPlaceholderValue(trimmed)
}

func hasEffectiveLocaleValue(locale string, key string, value string) bool {
	if hasStoredLocaleValue(value) {
		return true
	}
	_, ok := getBuiltinLocaleValue(locale, key)
	return ok
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
	re := regexp.MustCompile(`"([a-z0-9_]+\.[a-z0-9_\.]+)"`)
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
				keyMap[m[1]] = struct{}{}
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

func buildI18nKeyReferenceMatches(content string, oldKey string, newKey string) []I18nKeyReferenceMatch {
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
	if strings.HasSuffix(normalized, "_test.go") ||
		strings.HasSuffix(normalized, ".spec.ts") ||
		strings.HasSuffix(normalized, ".spec.tsx") ||
		strings.Contains(normalized, "/frontend/tests/") {
		return true
	}
	return strings.HasSuffix(normalized, "/frontend/src/i18n/index.ts") ||
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

func (s *I18nService) resetI18nLifecycle(_ string, module string, key string) error {
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
