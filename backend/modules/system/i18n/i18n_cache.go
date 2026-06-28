package system

import (
	"strings"
)

const (
	i18nSortModuleASC = "module ASC"
	i18nSortKeyASC    = "`key` ASC"
)

func (s *I18nService) ReloadCache() error {
	var items []SystemI18n
	if err := s.db.Order("locale ASC").Order(i18nSortModuleASC).Order(i18nSortKeyASC).Find(&items).Error; err != nil {
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
		if err := s.db.Where("locale = ?", normalized).Order(i18nSortModuleASC).Order(i18nSortKeyASC).Find(&items).Error; err != nil {
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
	if err := s.db.Where("locale = ?", locale).Order(i18nSortModuleASC).Order(i18nSortKeyASC).Find(&items).Error; err != nil {
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
