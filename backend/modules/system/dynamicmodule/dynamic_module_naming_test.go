package dynamicmodule

import "testing"

func TestNormalizeStaticModuleName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty string", "", ""},
		{"already normalized system", "system.auth", "system.auth"},
		{"business module", "business.order", "business.order"},
		{"platform lowcode mapped to system", "platform.lowcode", "system.lowcode"},
		{"whitespace trimmed", "  system.auth  ", "system.auth"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := normalizeStaticModuleName(tc.input); got != tc.expected {
				t.Errorf("normalizeStaticModuleName(%q) = %q, want %q", tc.input, got, tc.expected)
			}
		})
	}
}

func TestInferModuleScope(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"business prefix", "business.order", "business"},
		{"platform lowcode normalized to system", "platform.lowcode", "system"},
		{"platform prefix other", "platform.home", "platform"},
		{"system default", "system.auth", "system"},
		{"system default - no prefix", "auth", "system"},
		{"empty string defaults to system", "", "system"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := inferModuleScope(tc.input); got != tc.expected {
				t.Errorf("inferModuleScope(%q) = %q, want %q", tc.input, got, tc.expected)
			}
		})
	}
}

func TestSplitModuleKey(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantScope string
		wantName  string
		wantErr   bool
	}{
		{"valid system", "system.auth", "system", "auth", false},
		{"valid business", "business.order", "business", "order", false},
		{"nested only for business scope", "system.auth/role", "", "", true},
		{"empty string", "", "", "", true},
		{"no dot", "justname", "", "", true},
		{"too many dots", "a.b.c", "", "", true},
		{"invalid scope", "custom.auth", "", "", true},
		{"empty name after dot", "system.", "", "", true},
		{"double dots in name", "system.a..b", "", "", true},
		{"backslash in name", "system.a\\b", "", "", true},
		{"whitespace trimmed", "  system.auth  ", "system", "auth", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			scope, name, err := splitModuleKey(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Errorf("splitModuleKey(%q) expected error", tc.input)
				}
				return
			}
			if err != nil {
				t.Errorf("splitModuleKey(%q) unexpected error: %v", tc.input, err)
				return
			}
			if scope != tc.wantScope || name != tc.wantName {
				t.Errorf("splitModuleKey(%q) = (%q, %q), want (%q, %q)", tc.input, scope, name, tc.wantScope, tc.wantName)
			}
		})
	}
}

func TestBuildModuleKey(t *testing.T) {
	tests := []struct {
		name     string
		scope    string
		input    string
		expected string
	}{
		{"simple", "system", "auth", "system.auth"},
		{"nested path", "business", "order/manage", "business.order.manage"},
		{"trim leading slash", "system", "/auth/role", "system.auth.role"},
		{"trim trailing slash", "system", "auth/", "system.auth"},
		{"with whitespace", "  system  ", "  auth  ", "system.auth"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := buildModuleKey(tc.scope, tc.input); got != tc.expected {
				t.Errorf("buildModuleKey(%q, %q) = %q, want %q", tc.scope, tc.input, got, tc.expected)
			}
		})
	}
}

func TestResolveGeneratedParentMenu(t *testing.T) {
	tests := []struct {
		name           string
		scope          string
		moduleName     string
		explicitParent string
		wantPath       string
		wantMode       string
	}{
		{"explicit path", "system", "auth", "/admin/auth", "/admin/auth", "explicit"},
		{"explicit with backslash", "system", "auth", "\\admin\\auth", "/admin/auth", "explicit"},
		{"business inferred parent", "business", "order/manage", "", "/operations/order", "inferred"},
		{"business flat path", "business", "order", "", "", "top_level"},
		{"business flat path with slash", "business", "order/", "", "", "top_level"},
		{"system default top level", "system", "auth", "", "", "top_level"},
		{"empty explicit", "system", "auth", "", "", "top_level"},
		{"explicit already normalized", "system", "auth", "/admin", "/admin", "explicit"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotPath, gotMode := resolveGeneratedParentMenu(tc.scope, tc.moduleName, tc.explicitParent)
			if gotPath != tc.wantPath || gotMode != tc.wantMode {
				t.Errorf("resolveGeneratedParentMenu(%q, %q, %q) = (%q, %q), want (%q, %q)",
					tc.scope, tc.moduleName, tc.explicitParent, gotPath, gotMode, tc.wantPath, tc.wantMode)
			}
		})
	}
}

func TestInferGeneratedModelName(t *testing.T) {
	tests := []struct {
		name     string
		module   string
		explicit string
		expected string
	}{
		{"explicit takes priority", "order", "CustomModel", "CustomModel"},
		{"empty explicit from module name", "order_manage", "", "OrderManage"},
		{"empty explicit single word", "order", "", "Order"},
		{"empty explicit with dots", "order.manage", "", "OrderManage"},
		{"empty explicit with hyphens", "order-manage", "", "OrderManage"},
		{"empty explicit with slashes", "order/manage", "", "OrderManage"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := inferGeneratedModelName(tc.module, tc.explicit); got != tc.expected {
				t.Errorf("inferGeneratedModelName(%q, %q) = %q, want %q", tc.module, tc.explicit, got, tc.expected)
			}
		})
	}
}

func TestNormalizeGeneratedModulePath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty", "", ""},
		{"already clean", "auth", "auth"},
		{"leading slash", "/auth/role", "auth/role"},
		{"trailing slash", "auth/", "auth"},
		{"backslash converted", "auth\\role", "auth/role"},
		{"mixed slashes", "auth\\role/test", "auth/role/test"},
		{"whitespace trimmed", "  auth  ", "auth"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := normalizeGeneratedModulePath(tc.input); got != tc.expected {
				t.Errorf("normalizeGeneratedModulePath(%q) = %q, want %q", tc.input, got, tc.expected)
			}
		})
	}
}

func TestNormalizeGeneratedMenuPath(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty", "", ""},
		{"already normalized", "/admin/auth", "/admin/auth"},
		{"missing leading slash", "admin/auth", "/admin/auth"},
		{"trailing slash trimmed", "/admin/auth/", "/admin/auth"},
		{"backslash", "\\admin\\auth", "/admin/auth"},
		{"whitespace trimmed", "  /admin  ", "/admin"},
		{"just backslashes", "\\", "/"},
		{"just whitespace", "  ", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := normalizeGeneratedMenuPath(tc.input); got != tc.expected {
				t.Errorf("normalizeGeneratedMenuPath(%q) = %q, want %q", tc.input, got, tc.expected)
			}
		})
	}
}

func TestToGeneratedPascal(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"snake case", "order_manage", "OrderManage"},
		{"kebab case", "order-manage", "OrderManage"},
		{"dot case", "order.manage", "OrderManage"},
		{"path case", "order/manage", "OrderManage"},
		{"single word", "order", "Order"},
		{"already Pascal", "OrderManage", "OrderManage"},
		{"mixed separators", "order_manage-v2", "OrderManageV2"},
		{"empty string", "", ""},
		{"underscore only", "_", ""},
		{"starts with separator", "/order", "Order"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := toGeneratedPascal(tc.input); got != tc.expected {
				t.Errorf("toGeneratedPascal(%q) = %q, want %q", tc.input, got, tc.expected)
			}
		})
	}
}

func TestInferStaticModuleSource(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"platform prefix", "platform.home", "core"},
		{"system prefix", "system.auth", "core"},
		{"bare system", "system", "core"},
		{"business module", "business.order", "static"},
		{"unknown module", "auth", "static"},
		{"empty module", "", "static"},
		{"platform lowcode (mapped to system)", "platform.lowcode", "core"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := inferStaticModuleSource(tc.input); got != tc.expected {
				t.Errorf("inferStaticModuleSource(%q) = %q, want %q", tc.input, got, tc.expected)
			}
		})
	}
}

func TestInferRegistrationSource(t *testing.T) {
	tests := []struct {
		name       string
		scope      string
		sourceMode string
		moduleName string
		managed    bool
		expected   string
	}{
		{"database source mode", "system", "database", "auth", false, "database"},
		{"manual source mode", "system", "manual", "auth", false, "manual"},
		{"managed generated", "system", "", "auth", true, "generated"},
		{"core module", "system", "", "auth", false, "core"},
		{"static business", "business", "", "order", false, "static"},
		{"whitespace trimmed", "system", "  database  ", "auth", false, "database"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := inferRegistrationSource(tc.scope, tc.sourceMode, tc.moduleName, tc.managed); got != tc.expected {
				t.Errorf("inferRegistrationSource(%q, %q, %q, %v) = %q, want %q",
					tc.scope, tc.sourceMode, tc.moduleName, tc.managed, got, tc.expected)
			}
		})
	}
}

func TestIsValidDynamicModulePath(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		allowNested bool
		expected    bool
	}{
		{"valid single segment", "auth", false, true},
		{"single uppercase invalid", "Auth", false, false},
		{"single digit ok", "auth2", false, true},
		{"single underscore ok", "auth_role", false, true},
		{"empty", "", false, false},
		{"nested not allowed", "auth/role", false, false},
		{"nested allowed", "auth/role", true, true},
		{"nested with uppercase", "Auth/Role", true, false},
		{"double slash", "auth//role", true, false},
		{"leading slash", "/auth", false, false},
		{"trailing slash", "auth/", false, false},
		{"special chars", "auth@role", false, false},
		{"spaces", "auth role", false, false},
		{"deeply nested", "a/b/c", true, true},
		{"business nested with uppercase", "OrderMan/Detail", true, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := isValidDynamicModulePath(tc.input, tc.allowNested); got != tc.expected {
				t.Errorf("isValidDynamicModulePath(%q, %v) = %v, want %v", tc.input, tc.allowNested, got, tc.expected)
			}
		})
	}
}
