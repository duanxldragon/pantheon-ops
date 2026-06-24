package system

import "testing"

func TestIsLikelyI18nKey(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"system.menu.security", true},
		{"common.username", true},
		{"singleword", false},
		{"", false},
		{"123.abc", false},
		{"abc.123", true},
		{"abc.def.ghi", true},
		{"abc.def.go", false},
		{"abc.def.tsx", false},
		{"abc.def.json", false},
		{"abc.123.go", false},
		{"a.b", true},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			if got := isLikelyI18nKey(tc.input); got != tc.want {
				t.Errorf("isLikelyI18nKey(%q) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

func TestIgnoredI18nUsageFile(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"/frontend/src/i18n/index.ts", true},
		{"/frontend/src/i18n/resources/zh-CN.ts", true},
		{"/backend/modules/system/i18n/seed_data.go", true},
		{"/frontend/src/pages/Login.vue", false},
		{"/backend/pkg/common/response.go", false},
		{"", false},
	}
	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			if got := isIgnoredI18nUsageFile(tc.path); got != tc.want {
				t.Errorf("isIgnoredI18nUsageFile(%q) = %v, want %v", tc.path, got, tc.want)
			}
		})
	}
}

func TestSortedSetKeys(t *testing.T) {
	tests := []struct {
		name  string
		input map[string]struct{}
		want  []string
	}{
		{"empty map", map[string]struct{}{}, []string{}},
		{"single entry", map[string]struct{}{"a": {}}, []string{"a"}},
		{"multiple entries", map[string]struct{}{"b": {}, "a": {}, "c": {}}, []string{"a", "b", "c"}},
		{"skips empty", map[string]struct{}{"a": {}, "": {}, "b": {}}, []string{"a", "b"}},
		{"skips whitespace", map[string]struct{}{"a": {}, "  ": {}}, []string{"a"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := sortedSetKeys(tc.input)
			if len(got) != len(tc.want) {
				t.Fatalf("sortedSetKeys() = %v (len=%d), want %v (len=%d)", got, len(got), tc.want, len(tc.want))
			}
			for i := range got {
				if got[i] != tc.want[i] {
					t.Errorf("sortedSetKeys()[%d] = %q, want %q", i, got[i], tc.want[i])
				}
			}
		})
	}
}

func TestContainsString(t *testing.T) {
	tests := []struct {
		name   string
		items  []string
		target string
		want   bool
	}{
		{"found", []string{"a", "b", "c"}, "b", true},
		{"not found", []string{"a", "b", "c"}, "d", false},
		{"empty list", []string{}, "a", false},
		{"nil list", nil, "a", false},
		{"with empty string", []string{"", "b"}, "", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := containsString(tc.items, tc.target); got != tc.want {
				t.Errorf("containsString(%v, %q) = %v, want %v", tc.items, tc.target, got, tc.want)
			}
		})
	}
}

func TestAllValuesMissing(t *testing.T) {
	tests := []struct {
		name  string
		input map[string]struct{}
		want  bool
	}{
		{"empty map", map[string]struct{}{}, true},
		{"nil map", nil, true},
		{"all empty strings", map[string]struct{}{"": {}, " ": {}}, true},
		{"all placeholders", map[string]struct{}{"[key1]": {}, "[key2]": {}}, true},
		{"has real value", map[string]struct{}{"[key]": {}, "real": {}}, false},
		{"single real value", map[string]struct{}{"real": {}}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := allValuesMissing(tc.input); got != tc.want {
				t.Errorf("allValuesMissing(%v) = %v, want %v", tc.input, got, tc.want)
			}
		})
	}
}

func TestSuggestScopedI18nKey(t *testing.T) {
	tests := []struct {
		module string
		key    string
		want   string
	}{
		{"system.auth", "menu.security", "system.auth.menu.security"},
		{"system.auth", "system.auth.menu.security", "system.auth.menu.security"},
		{"", "menu.security", "menu.security"},
		{"system.auth", "", ""},
		{"", "", ""},
		{"system.auth", " auth.login ", "system.auth.auth.login"},
	}
	for _, tc := range tests {
		t.Run(tc.module+"/"+tc.key, func(t *testing.T) {
			if got := suggestScopedI18nKey(tc.module, tc.key); got != tc.want {
				t.Errorf("suggestScopedI18nKey(%q, %q) = %q, want %q", tc.module, tc.key, got, tc.want)
			}
		})
	}
}

func TestNormalizeI18nLifecycleStatus(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{I18nLifecycleStatusActive, I18nLifecycleStatusActive},
		{I18nLifecycleStatusObserving, I18nLifecycleStatusObserving},
		{I18nLifecycleStatusArchived, I18nLifecycleStatusArchived},
		{" active ", I18nLifecycleStatusActive},
		{" unknown ", I18nLifecycleStatusActive},
		{"", I18nLifecycleStatusActive},
	}
	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			if got := normalizeI18nLifecycleStatus(tc.input); got != tc.want {
				t.Errorf("normalizeI18nLifecycleStatus(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}
