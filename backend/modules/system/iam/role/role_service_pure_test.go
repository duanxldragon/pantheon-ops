package iam

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
)

const roleSortFieldTestName = "role_name"

// ---- normalizeRoleStatus ----

func TestNormalizeRoleStatus_EnabledAndDisabledPassThrough(t *testing.T) {
	if got := normalizeRoleStatus(common.StatusEnabled); got != common.StatusEnabled {
		t.Fatalf("expected %d, got %d", common.StatusEnabled, got)
	}
	if got := normalizeRoleStatus(common.StatusDisabled); got != common.StatusDisabled {
		t.Fatalf("expected %d, got %d", common.StatusDisabled, got)
	}
}

func TestNormalizeRoleStatus_UnknownDefaultsToEnabled(t *testing.T) {
	if got := normalizeRoleStatus(0); got != common.StatusEnabled {
		t.Fatalf("expected enabled default, got %d", got)
	}
	if got := normalizeRoleStatus(99); got != common.StatusEnabled {
		t.Fatalf("expected enabled default for unknown status, got %d", got)
	}
}

// ---- normalizeRolePageQuery ----

func TestNormalizeRolePageQuery_Nil(t *testing.T) {
	page, pageSize := normalizeRolePageQuery(nil)
	if page != 1 || pageSize != 10 {
		t.Fatalf("expected (1, 10), got (%d, %d)", page, pageSize)
	}
}

func TestNormalizeRolePageQuery_ZeroValues(t *testing.T) {
	page, pageSize := normalizeRolePageQuery(&RoleListQuery{})
	if page != 1 || pageSize != 10 {
		t.Fatalf("expected (1, 10) for zero values, got (%d, %d)", page, pageSize)
	}
}

func TestNormalizeRolePageQuery_CustomValues(t *testing.T) {
	page, pageSize := normalizeRolePageQuery(&RoleListQuery{Page: 4, PageSize: 25})
	if page != 4 || pageSize != 25 {
		t.Fatalf("expected (4, 25), got (%d, %d)", page, pageSize)
	}
}

func TestNormalizeRolePageQuery_CapsPageSizeAt100(t *testing.T) {
	_, pageSize := normalizeRolePageQuery(&RoleListQuery{Page: 1, PageSize: 1000})
	if pageSize != 100 {
		t.Fatalf("expected pageSize capped at 100, got %d", pageSize)
	}
}

// ---- normalizeRoleMemberPageQuery ----

func TestNormalizeRoleMemberPageQuery_Nil(t *testing.T) {
	page, pageSize := normalizeRoleMemberPageQuery(nil)
	if page != 1 || pageSize != 10 {
		t.Fatalf("expected (1, 10), got (%d, %d)", page, pageSize)
	}
}

func TestNormalizeRoleMemberPageQuery_ZeroValues(t *testing.T) {
	page, pageSize := normalizeRoleMemberPageQuery(&RoleMemberQuery{})
	if page != 1 || pageSize != 10 {
		t.Fatalf("expected (1, 10) for zero values, got (%d, %d)", page, pageSize)
	}
}

func TestNormalizeRoleMemberPageQuery_CapsPageSizeAt100(t *testing.T) {
	_, pageSize := normalizeRoleMemberPageQuery(&RoleMemberQuery{Page: 2, PageSize: 200})
	if pageSize != 100 {
		t.Fatalf("expected pageSize capped at 100, got %d", pageSize)
	}
}

// ---- normalizeRoleSort ----

func TestNormalizeRoleSort_Nil(t *testing.T) {
	column, desc := normalizeRoleSort(nil)
	if column != "id" || desc != true {
		t.Fatalf("expected (id, true), got (%s, %v)", column, desc)
	}
}

func TestNormalizeRoleSort_DefaultSort(t *testing.T) {
	// Empty non-nil query: desc only when SortOrder is explicitly "desc".
	column, desc := normalizeRoleSort(&RoleListQuery{})
	if column != "id" || desc != false {
		t.Fatalf("expected (id, false) for empty query, got (%s, %v)", column, desc)
	}
}

func TestNormalizeRoleSort_WhitelistedFields(t *testing.T) {
	cases := []struct {
		sortField string
		want      string
	}{
		{sortField: "id", want: "id"},
		{sortField: "roleName", want: roleSortFieldTestName},
		{sortField: roleSortFieldTestName, want: roleSortFieldTestName},
		{sortField: "roleKey", want: "role_key"},
		{sortField: "role_key", want: "role_key"},
		{sortField: "sort", want: "sort"},
		{sortField: "status", want: "status"},
		{sortField: "createdAt", want: "created_at"},
		{sortField: "created_at", want: "created_at"},
	}
	for _, tc := range cases {
		column, _ := normalizeRoleSort(&RoleListQuery{SortField: tc.sortField})
		if column != tc.want {
			t.Fatalf("expected %s to map to %s, got %s", tc.sortField, tc.want, column)
		}
	}
}

func TestNormalizeRoleSort_InvalidFieldFallsBackToID(t *testing.T) {
	column, _ := normalizeRoleSort(&RoleListQuery{SortField: "role_key; DROP TABLE system_role"})
	if column != "id" {
		t.Fatalf("expected id fallback for injection-like field, got %s", column)
	}
}

func TestNormalizeRoleSort_OrderOnlyDescWhenExplicit(t *testing.T) {
	column, desc := normalizeRoleSort(&RoleListQuery{SortField: roleSortFieldTestName, SortOrder: "asc"})
	if column != roleSortFieldTestName || desc != false {
		t.Fatalf("expected (role_name, false), got (%s, %v)", column, desc)
	}

	column, desc = normalizeRoleSort(&RoleListQuery{SortField: roleSortFieldTestName, SortOrder: "DESC"})
	if column != roleSortFieldTestName || desc != true {
		t.Fatalf("expected case-insensitive desc, got (%s, %v)", column, desc)
	}

	column, desc = normalizeRoleSort(&RoleListQuery{SortField: roleSortFieldTestName, SortOrder: "invalid"})
	if column != roleSortFieldTestName || desc != false {
		t.Fatalf("expected non-desc order to be asc, got (%s, %v)", column, desc)
	}
}

// ---- normalizeUint64IDs (role package variant) ----

func TestNormalizeRoleUint64IDs_EmptyInputReturnsEmptySlice(t *testing.T) {
	result := normalizeUint64IDs(nil)
	if result == nil || len(result) != 0 {
		t.Fatalf("expected empty non-nil slice, got %v", result)
	}
}

func TestNormalizeRoleUint64IDs_RemovesZerosAndDuplicates(t *testing.T) {
	result := normalizeUint64IDs([]uint64{5, 0, 5, 7})
	if len(result) != 2 || result[0] != 5 || result[1] != 7 {
		t.Fatalf("expected [5 7], got %v", result)
	}
}

// ---- normalizePermissionKeys ----

func TestNormalizePermissionKeys_TrimsAndDeduplicates(t *testing.T) {
	result := normalizePermissionKeys([]string{" sys:user:list ", "sys:user:create", "sys:user:list", ""})
	if len(result) != 2 || result[0] != "sys:user:list" || result[1] != "sys:user:create" {
		t.Fatalf("expected trimmed deduped keys, got %v", result)
	}
}

func TestNormalizePermissionKeys_EmptyInput(t *testing.T) {
	result := normalizePermissionKeys(nil)
	if result == nil || len(result) != 0 {
		t.Fatalf("expected empty non-nil slice, got %v", result)
	}
}

// ---- normalizeRoleDataScope / isValidRoleDataScopeMode ----

func TestNormalizeRoleDataScope_KnownModesPassThrough(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{input: " self ", want: common.DataScopeModeSelf},
		{input: "DEPT", want: common.DataScopeModeDept},
		{input: "Dept_And_Children", want: common.DataScopeModeDeptAndChildren},
		{input: "\tCustom\n", want: common.DataScopeModeCustom},
		{input: common.DataScopeModeAll, want: common.DataScopeModeAll},
	}
	for _, tc := range cases {
		if got := normalizeRoleDataScope(tc.input); got != tc.want {
			t.Fatalf("expected %q to normalize to %q, got %q", tc.input, tc.want, got)
		}
	}
}

func TestIsValidRoleDataScopeMode(t *testing.T) {
	for _, mode := range []string{
		common.DataScopeModeAll,
		common.DataScopeModeSelf,
		common.DataScopeModeDept,
		common.DataScopeModeDeptAndChildren,
		common.DataScopeModeCustom,
	} {
		if !isValidRoleDataScopeMode(mode) {
			t.Fatalf("expected %q to be valid", mode)
		}
	}
	for _, mode := range []string{"", "unknown", "ALL ", "self;drop"} {
		if isValidRoleDataScopeMode(mode) {
			t.Fatalf("expected %q to be invalid", mode)
		}
	}
}

// ---- buildRoleListItems ----

// 回归: 无菜单/权限关联的角色在列表 DTO 中必须是空切片而非 null,
// 否则前端 RoleRow.menuIds.map(String) 会在编辑角色时直接抛错。
func TestBuildRoleListItems_NormalizesNilAssociations(t *testing.T) {
	roles := []SystemRole{{ID: 7, RoleName: "lonely", RoleKey: "lonely"}}
	associations := roleListAssociations{
		menus:       map[uint64][]uint64{},
		permissions: map[uint64][]string{},
		dataScopes:  map[uint64]string{7: common.DataScopeModeAll},
	}

	items := buildRoleListItems(roles, associations)
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].MenuIDs == nil {
		t.Fatal("expected MenuIDs to be normalized to empty slice, got nil")
	}
	if items[0].PermissionKeys == nil {
		t.Fatal("expected PermissionKeys to be normalized to empty slice, got nil")
	}

	// JSON 序列化结果必须是 [] 而非 null
	encoded, err := json.Marshal(items[0])
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	if !strings.Contains(string(encoded), `"menuIds":[]`) {
		t.Fatalf("expected serialized menuIds to be [], got: %s", encoded)
	}
	if !strings.Contains(string(encoded), `"permissionKeys":[]`) {
		t.Fatalf("expected serialized permissionKeys to be [], got: %s", encoded)
	}
}
