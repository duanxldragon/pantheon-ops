package iam

import (
	"testing"
)

// ---- normalizePolicyMethod ----

func TestNormalizePolicyMethod_UpperCasesKnownMethods(t *testing.T) {
	cases := map[string]string{
		"get":    "GET",
		" post ": "POST",
		"Put":    "PUT",
		" patch": "PATCH",
		"DELETE": "DELETE",
	}
	for input, want := range cases {
		if got := normalizePolicyMethod(input); got != want {
			t.Fatalf("expected %q to normalize to %q, got %q", input, want, got)
		}
	}
}

func TestNormalizePolicyMethod_RejectsUnknownMethods(t *testing.T) {
	for _, method := range []string{"", "  ", "TRACE", "OPTIONS", "get;drop", "CONNECT"} {
		if got := normalizePolicyMethod(method); got != "" {
			t.Fatalf("expected %q to normalize to empty, got %q", method, got)
		}
	}
}

// ---- normalizePermissionPageQuery ----

func TestNormalizePermissionPageQuery_Nil(t *testing.T) {
	page, pageSize := normalizePermissionPageQuery(nil)
	if page != 1 || pageSize != 10 {
		t.Fatalf("expected (1, 10), got (%d, %d)", page, pageSize)
	}
}

func TestNormalizePermissionPageQuery_ZeroValues(t *testing.T) {
	page, pageSize := normalizePermissionPageQuery(&PermissionPolicyQuery{})
	if page != 1 || pageSize != 10 {
		t.Fatalf("expected (1, 10) for zero values, got (%d, %d)", page, pageSize)
	}
}

func TestNormalizePermissionPageQuery_CustomValues(t *testing.T) {
	page, pageSize := normalizePermissionPageQuery(&PermissionPolicyQuery{Page: 5, PageSize: 40})
	if page != 5 || pageSize != 40 {
		t.Fatalf("expected (5, 40), got (%d, %d)", page, pageSize)
	}
}

func TestNormalizePermissionPageQuery_CapsPageSizeAt100(t *testing.T) {
	_, pageSize := normalizePermissionPageQuery(&PermissionPolicyQuery{Page: 1, PageSize: 999})
	if pageSize != 100 {
		t.Fatalf("expected pageSize capped at 100, got %d", pageSize)
	}
}

// ---- normalizePermissionPolicySort ----

func TestNormalizePermissionPolicySort_Nil(t *testing.T) {
	column, desc := normalizePermissionPolicySort(nil)
	if column != "id" || desc != true {
		t.Fatalf("expected (id, true), got (%s, %v)", column, desc)
	}
}

func TestNormalizePermissionPolicySort_DefaultSort(t *testing.T) {
	column, desc := normalizePermissionPolicySort(&PermissionPolicyQuery{})
	if column != "id" || desc != true {
		t.Fatalf("expected (id, true) for empty query, got (%s, %v)", column, desc)
	}
}

func TestNormalizePermissionPolicySort_WhitelistedFields(t *testing.T) {
	cases := []struct {
		sortField string
		want      string
	}{
		{sortField: "id", want: "id"},
		{sortField: "roleKey", want: "v0"},
		{sortField: "path", want: "v1"},
		{sortField: "method", want: "v2"},
	}
	for _, tc := range cases {
		column, _ := normalizePermissionPolicySort(&PermissionPolicyQuery{SortField: tc.sortField})
		if column != tc.want {
			t.Fatalf("expected %s to map to %s, got %s", tc.sortField, tc.want, column)
		}
	}
}

func TestNormalizePermissionPolicySort_InvalidFieldFallsBackToIDDesc(t *testing.T) {
	column, desc := normalizePermissionPolicySort(&PermissionPolicyQuery{SortField: "v3; DROP TABLE casbin_rule", SortOrder: "asc"})
	if column != "id" || desc != true {
		t.Fatalf("expected (id, true) fallback for unknown field, got (%s, %v)", column, desc)
	}
}

func TestNormalizePermissionPolicySort_AscAndDesc(t *testing.T) {
	column, desc := normalizePermissionPolicySort(&PermissionPolicyQuery{SortField: "path", SortOrder: "asc"})
	if column != "v1" || desc != false {
		t.Fatalf("expected (v1, false), got (%s, %v)", column, desc)
	}

	column, desc = normalizePermissionPolicySort(&PermissionPolicyQuery{SortField: "path", SortOrder: "DESC"})
	if column != "v1" || desc != true {
		t.Fatalf("expected case-insensitive desc, got (%s, %v)", column, desc)
	}
}

// ---- boolToCSVValue ----

func TestBoolToCSVValue(t *testing.T) {
	if got := boolToCSVValue(true); got != "true" {
		t.Fatalf("expected true, got %q", got)
	}
	if got := boolToCSVValue(false); got != "false" {
		t.Fatalf("expected false, got %q", got)
	}
}

// ---- joinWorkbenchPolicyKeys ----

func TestJoinWorkbenchPolicyKeys_JoinsMethodAndPathSorted(t *testing.T) {
	keys := joinWorkbenchPolicyKeys([]PermissionWorkbenchAPIPolicyResp{
		{Path: "/api/v1/system/user", Method: "post"},
		{Path: "/api/v1/system/user/list", Method: "GET"},
	})
	if keys != "GET /api/v1/system/user/list|POST /api/v1/system/user" {
		t.Fatalf("unexpected joined keys: %q", keys)
	}
}

func TestJoinWorkbenchPolicyKeys_SkipsEmptyMethodOrPath(t *testing.T) {
	keys := joinWorkbenchPolicyKeys([]PermissionWorkbenchAPIPolicyResp{
		{Path: "/api/v1/system/user/list", Method: ""},
		{Path: "", Method: "GET"},
		{Path: "/api/v1/system/user", Method: "POST"},
	})
	if keys != "POST /api/v1/system/user" {
		t.Fatalf("expected only complete policy key, got %q", keys)
	}
}

func TestJoinWorkbenchPolicyKeys_EmptyInput(t *testing.T) {
	if got := joinWorkbenchPolicyKeys(nil); got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}
