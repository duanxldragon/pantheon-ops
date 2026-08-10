package system

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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

// scanRootCollector 是 resolveI18nScanRoots 中 appendRoot 闭包行为的等价实现，
// 用于在测试中收集去重后的扫描根路径。使用指针方法避免切片头按值返回的捕获陷阱。
type scanRootCollector struct {
	roots []string
	seen  map[string]struct{}
}

func newScanRootCollector() *scanRootCollector {
	return &scanRootCollector{roots: make([]string, 0), seen: map[string]struct{}{}}
}

func (c *scanRootCollector) appendRoot(root string) {
	normalized := strings.TrimSpace(filepath.Clean(root))
	if normalized == "" {
		return
	}
	if _, ok := c.seen[normalized]; ok {
		return
	}
	c.seen[normalized] = struct{}{}
	c.roots = append(c.roots, normalized)
}

func mkScanRootBase(t *testing.T, withBackend, withFrontend bool) string {
	t.Helper()
	base := t.TempDir()
	if withBackend {
		if err := os.MkdirAll(filepath.Join(base, "backend"), 0o750); err != nil {
			t.Fatalf("mkdir backend: %v", err)
		}
	}
	if withFrontend {
		if err := os.MkdirAll(filepath.Join(base, "frontend"), 0o750); err != nil {
			t.Fatalf("mkdir frontend: %v", err)
		}
	}
	return base
}

func TestAttemptScanRootPair(t *testing.T) {
	t.Run("both present", func(t *testing.T) {
		base := mkScanRootBase(t, true, true)
		c := newScanRootCollector()
		if !attemptScanRootPair(c.appendRoot, base) {
			t.Fatalf("expected attemptScanRootPair to return true")
		}
		if len(c.roots) != 2 {
			t.Fatalf("expected 2 roots, got %d: %v", len(c.roots), c.roots)
		}
		if c.roots[0] != filepath.Join(base, "backend") || c.roots[1] != filepath.Join(base, "frontend") {
			t.Fatalf("unexpected roots order/content: %v", c.roots)
		}
	})

	t.Run("missing backend", func(t *testing.T) {
		base := mkScanRootBase(t, false, true)
		c := newScanRootCollector()
		if attemptScanRootPair(c.appendRoot, base) {
			t.Fatalf("expected attemptScanRootPair to return false")
		}
		if len(c.roots) != 0 {
			t.Fatalf("expected 0 roots, got %d: %v", len(c.roots), c.roots)
		}
	})

	t.Run("empty dir", func(t *testing.T) {
		c := newScanRootCollector()
		if attemptScanRootPair(c.appendRoot, "") {
			t.Fatalf("expected attemptScanRootPair to return false for empty dir")
		}
		if len(c.roots) != 0 {
			t.Fatalf("expected 0 roots, got %d: %v", len(c.roots), c.roots)
		}
	})
}

func TestWalkUpScanRoots(t *testing.T) {
	base := t.TempDir()
	nested := filepath.Join(base, "a", "b", "c")
	if err := os.MkdirAll(nested, 0o750); err != nil {
		t.Fatalf("mkdir nested: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(base, "backend"), 0o750); err != nil {
		t.Fatalf("mkdir backend: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(base, "frontend"), 0o750); err != nil {
		t.Fatalf("mkdir frontend: %v", err)
	}

	c := newScanRootCollector()
	if !walkUpScanRoots(c.appendRoot, nested) {
		t.Fatalf("expected walkUpScanRoots to find pair starting from %q", nested)
	}
	if len(c.roots) != 2 {
		t.Fatalf("expected 2 roots, got %d: %v", len(c.roots), c.roots)
	}
	if c.roots[0] != filepath.Join(base, "backend") || c.roots[1] != filepath.Join(base, "frontend") {
		t.Fatalf("unexpected roots: %v", c.roots)
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

func TestMatchesLifecycleTransitionTarget(t *testing.T) {
	tests := []struct {
		name       string
		item       I18nUnusedKeyItem
		module     string
		fromStatus string
		filter     func(I18nUnusedKeyItem) bool
		want       bool
	}{
		{
			name:       "active matches active, no module scope",
			item:       I18nUnusedKeyItem{Module: "system", Key: "a", LifecycleStatus: I18nLifecycleStatusActive},
			module:     "",
			fromStatus: I18nLifecycleStatusActive,
			want:       true,
		},
		{
			name:       "module scope match",
			item:       I18nUnusedKeyItem{Module: "system", Key: "a", LifecycleStatus: I18nLifecycleStatusActive},
			module:     "system",
			fromStatus: I18nLifecycleStatusActive,
			want:       true,
		},
		{
			name:       "module scope mismatch",
			item:       I18nUnusedKeyItem{Module: "system", Key: "a", LifecycleStatus: I18nLifecycleStatusActive},
			module:     "other",
			fromStatus: I18nLifecycleStatusActive,
			want:       false,
		},
		{
			name:       "lifecycle status mismatch",
			item:       I18nUnusedKeyItem{Module: "system", Key: "a", LifecycleStatus: I18nLifecycleStatusObserving},
			module:     "",
			fromStatus: I18nLifecycleStatusActive,
			want:       false,
		},
		{
			name:       "status normalized before compare",
			item:       I18nUnusedKeyItem{Module: "system", Key: "a", LifecycleStatus: " active "},
			module:     "",
			fromStatus: I18nLifecycleStatusActive,
			want:       true,
		},
		{
			name:       "filter rejects",
			item:       I18nUnusedKeyItem{Module: "system", Key: "a", LifecycleStatus: I18nLifecycleStatusActive},
			module:     "",
			fromStatus: I18nLifecycleStatusActive,
			filter:     func(I18nUnusedKeyItem) bool { return false },
			want:       false,
		},
		{
			name:       "filter accepts",
			item:       I18nUnusedKeyItem{Module: "system", Key: "a", LifecycleStatus: I18nLifecycleStatusActive},
			module:     "",
			fromStatus: I18nLifecycleStatusActive,
			filter:     func(I18nUnusedKeyItem) bool { return true },
			want:       true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := matchesLifecycleTransitionTarget(tc.item, tc.module, tc.fromStatus, tc.filter); got != tc.want {
				t.Errorf("matchesLifecycleTransitionTarget() = %v, want %v", got, tc.want)
			}
		})
	}
}

const testKeepActiveSecondKey = "keep-active-2"

func TestCollectUnusedLifecycleTargets(t *testing.T) {
	audit := &I18nAuditResp{
		UnusedKeys: []I18nUnusedKeyItem{
			{Module: "system", Key: "keep-active", LifecycleStatus: I18nLifecycleStatusActive},
			{Module: "system", Key: "drop-observing", LifecycleStatus: I18nLifecycleStatusObserving},
			{Module: "other", Key: "other-active", LifecycleStatus: I18nLifecycleStatusActive},
			{Module: "system", Key: testKeepActiveSecondKey, LifecycleStatus: I18nLifecycleStatusActive},
		},
	}

	cases := []struct {
		name        string
		module      string
		status      string
		filter      func(I18nUnusedKeyItem) bool
		wantKeys    []string
		wantModules []string
	}{
		{
			name:        "active scoped to system",
			module:      "system",
			status:      I18nLifecycleStatusActive,
			wantKeys:    []string{"keep-active", testKeepActiveSecondKey},
			wantModules: []string{"system", "system"},
		},
		{
			name:     "observing no module scope",
			status:   I18nLifecycleStatusObserving,
			wantKeys: []string{"drop-observing"},
		},
		{
			name:        "with filter",
			module:      "system",
			status:      I18nLifecycleStatusActive,
			filter:      func(item I18nUnusedKeyItem) bool { return item.Key == testKeepActiveSecondKey },
			wantKeys:    []string{testKeepActiveSecondKey},
			wantModules: []string{"system"},
		},
		{name: "no matches returns empty", module: "missing", status: I18nLifecycleStatusActive},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assertUnusedLifecycleTargets(t, audit, tc.module, tc.status, tc.filter, tc.wantKeys, tc.wantModules)
		})
	}
}

func assertUnusedLifecycleTargets(
	t *testing.T,
	audit *I18nAuditResp,
	module string,
	status string,
	filter func(I18nUnusedKeyItem) bool,
	wantKeys []string,
	wantModules []string,
) {
	t.Helper()
	targets, keys := collectUnusedLifecycleTargets(audit, module, status, filter)
	if strings.Join(keys, "|") != strings.Join(wantKeys, "|") {
		t.Fatalf("unexpected affected keys: got=%v want=%v", keys, wantKeys)
	}
	if len(targets) != len(wantKeys) {
		t.Fatalf("unexpected targets: got=%v want count=%d", targets, len(wantKeys))
	}
	for i, wantModule := range wantModules {
		if targets[i].module != wantModule {
			t.Fatalf("unexpected target modules: %v", targets)
		}
	}
}
