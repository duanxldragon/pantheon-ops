package system

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"gorm.io/gorm"
)

type I18nService struct {
	db *gorm.DB
	mu sync.RWMutex

	cache map[string]map[string]string
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
	return s.Bootstrap()
}

func (s *I18nService) Bootstrap() error {
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
