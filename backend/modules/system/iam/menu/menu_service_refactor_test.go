package iam

import "testing"

// 这些测试仅覆盖重构中抽取出的纯函数/辅助逻辑，不需要数据库或 Redis，可在本地确定性运行。

func TestCollectMenusWithAncestorsIncludesAncestors(t *testing.T) {
	menus := []SystemMenu{
		{ID: 1, ParentID: 0, TitleKey: "root"},
		{ID: 2, ParentID: 1, TitleKey: "mid"},
		{ID: 3, ParentID: 2, TitleKey: "leaf"},
		{ID: 4, ParentID: 0, TitleKey: "other"},
	}
	got := collectMenusWithAncestors(menus, func(m SystemMenu) bool {
		return m.ID == 3
	})
	wantIDs := []uint64{1, 2, 3}
	if len(got) != len(wantIDs) {
		t.Fatalf("got %d menus, want %d", len(got), len(wantIDs))
	}
	for i, id := range wantIDs {
		if got[i].ID != id {
			t.Errorf("position %d: got id %d, want %d", i, got[i].ID, id)
		}
	}
}

func TestWalkAncestorsStopsAtExistingNode(t *testing.T) {
	menuMap := map[uint64]SystemMenu{
		1: {ID: 1, ParentID: 0},
		2: {ID: 2, ParentID: 1},
		3: {ID: 3, ParentID: 2},
	}
	selected := map[uint64]struct{}{2: {}}
	walkAncestors(selected, menuMap, 3)
	// 3 的祖先 2 已存在 -> 停止；1 不应被加入（与命中已选节点即停止的等价语义一致）
	if _, ok := selected[1]; ok {
		t.Error("node 1 should not be added when ancestor 2 already selected")
	}
	if _, ok := selected[3]; !ok {
		t.Error("node 3 should be added")
	}
}

func TestMenuMetaPredicates(t *testing.T) {
	if !isRouteNameRequired("C", "") {
		t.Error("C type with empty routeName should be required")
	}
	if isRouteNameRequired("C", "x") {
		t.Error("C type with routeName should not be required")
	}
	if isRouteNameRequired("M", "") {
		t.Error("M type should not require routeName")
	}

	if !isPagePermRequired("C", 0, "") {
		t.Error("C non-external without pagePerm required")
	}
	if isPagePermRequired("C", 1, "") {
		t.Error("C external without pagePerm should not be required")
	}
	if isPagePermRequired("M", 0, "") {
		t.Error("M type should not require pagePerm")
	}

	if !isPermsRequired("F", "") {
		t.Error("F without perms required")
	}
	if isPermsRequired("C", "") {
		t.Error("C type should not require perms via isPermsRequired")
	}

	if !isComponentRequired("C", "") {
		t.Error("C without component required")
	}
	if isComponentRequired("F", "") {
		t.Error("F should not require component")
	}

	// isComponentInvalid 依赖组件注册表（确定性静态数据）。
	if isComponentInvalid("C", "system.iam", "system/user/UserList") {
		t.Error("registered component key should be valid")
	}
	if !isComponentInvalid("C", "system.iam", "not/a/real/component") {
		t.Error("unregistered key under a module that requires registration should be invalid")
	}
	if isComponentInvalid("C", "custom.module", "whatever") {
		t.Error("module not requiring registration should not be marked invalid by key")
	}
}

func TestValidateExternalMenuPath(t *testing.T) {
	if validateExternalMenuPath("https://example.com/x") != nil {
		t.Error("valid https path should pass")
	}
	if validateExternalMenuPath("http://example.com") != nil {
		t.Error("valid http path should pass")
	}
	if validateExternalMenuPath("ftp://example.com") == nil {
		t.Error("non-http(s) scheme should fail")
	}
	if validateExternalMenuPath("not a url") == nil {
		t.Error("invalid url should fail")
	}
}
