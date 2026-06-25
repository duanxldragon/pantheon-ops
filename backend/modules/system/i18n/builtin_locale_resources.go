package system

import (
	_ "embed"
	"encoding/json"
	"strings"
	"sync"
)

//go:embed builtin_locale_resources.json
var builtinLocaleResourcesJSON []byte

var (
	builtinLocaleResourcesOnce sync.Once
	builtinLocaleResources     map[string]map[string]string
)

func loadBuiltinLocaleResources() map[string]map[string]string {
	builtinLocaleResourcesOnce.Do(func() {
		builtinLocaleResources = make(map[string]map[string]string)
		if len(builtinLocaleResourcesJSON) == 0 {
			return
		}
		_ = json.Unmarshal(builtinLocaleResourcesJSON, &builtinLocaleResources)
	})
	return builtinLocaleResources
}

func getBuiltinLocalePack(locale string) map[string]string {
	pack, ok := loadBuiltinLocaleResources()[strings.TrimSpace(locale)]
	if !ok {
		return map[string]string{}
	}
	return cloneLangPack(pack)
}

func getBuiltinLocaleValue(locale, key string) (string, bool) {
	pack := loadBuiltinLocaleResources()[strings.TrimSpace(locale)]
	if len(pack) == 0 {
		return "", false
	}
	value, ok := pack[strings.TrimSpace(key)]
	if !ok {
		return "", false
	}
	trimmed := strings.TrimSpace(value)
	if trimmed == "" || isI18nPlaceholderValue(trimmed) {
		return "", false
	}
	return value, true
}
