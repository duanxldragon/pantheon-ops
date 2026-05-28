package system

import (
	"strings"
	"testing"
)

func TestBuiltinLocaleResourcesIncludeCmdbGeneratedKeys(t *testing.T) {
	keys := []string{
		"business.cmdb.group.hero.eyebrow",
		"business.cmdb.host.hero.title",
		"business.cmdb.label.hero.title",
	}
	locales := []string{"zh-CN", "en-US", "ja-JP", "ko-KR", "fr-FR"}

	for _, locale := range locales {
		for _, key := range keys {
			value, ok := getBuiltinLocaleValue(locale, key)
			if !ok {
				t.Fatalf("expected builtin locale %s to include key %s", locale, key)
			}
			if strings.TrimSpace(value) == "" {
				t.Fatalf("expected builtin locale %s key %s to have a non-empty value", locale, key)
			}
		}
	}
}
