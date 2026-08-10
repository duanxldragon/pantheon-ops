package iam

import "testing"

func TestFilterMenusWithAncestorsKeepsAncestorChain(t *testing.T) {
	menus := []SystemMenu{
		{ID: 1, ParentID: 0, TitleKey: "menu.system", Path: "/system"},
		{ID: 2, ParentID: 1, TitleKey: "menu.system.user", Path: "/system/user"},
		{ID: 3, ParentID: 1, TitleKey: "menu.system.role", Path: "/system/role"},
		{ID: 4, ParentID: 0, TitleKey: "menu.platform", Path: "/dashboard"},
	}

	cases := []struct {
		name    string
		query   *MenuListQuery
		wantIDs []uint64
	}{
		{
			name:    "keyword matches child keeps parent chain",
			query:   &MenuListQuery{Keyword: "user"},
			wantIDs: []uint64{1, 2},
		},
		{
			name:    "keyword matches parent keeps only parent",
			query:   &MenuListQuery{Keyword: "dashboard"},
			wantIDs: []uint64{4},
		},
		{
			name:    "keyword matches path across levels",
			query:   &MenuListQuery{Keyword: "/system"},
			wantIDs: []uint64{1, 2, 3},
		},
		{
			name:    "titleKey filter keeps ancestor chain",
			query:   &MenuListQuery{TitleKey: "role"},
			wantIDs: []uint64{1, 3},
		},
		{
			name:    "path filter keeps ancestor chain",
			query:   &MenuListQuery{Path: "/system/user"},
			wantIDs: []uint64{1, 2},
		},
		{
			name:    "combined filters intersect",
			query:   &MenuListQuery{Keyword: "system", TitleKey: "user"},
			wantIDs: []uint64{1, 2},
		},
		{
			name:    "no match returns empty",
			query:   &MenuListQuery{Keyword: "missing"},
			wantIDs: []uint64{},
		},
		{
			name:    "case insensitive match",
			query:   &MenuListQuery{Keyword: "USER"},
			wantIDs: []uint64{1, 2},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := filterMenusWithAncestors(menus, tc.query)
			if len(got) != len(tc.wantIDs) {
				t.Fatalf("expected %d menus, got %d", len(tc.wantIDs), len(got))
			}
			for index, wantID := range tc.wantIDs {
				if got[index].ID != wantID {
					t.Fatalf("position %d: expected id %d, got %d", index, wantID, got[index].ID)
				}
			}
		})
	}
}

func TestFilterMenusWithAncestorsPreservesInputOrder(t *testing.T) {
	// sort 序：platform(排前) -> system -> children
	menus := []SystemMenu{
		{ID: 4, ParentID: 0, TitleKey: "menu.platform", Path: "/dashboard"},
		{ID: 1, ParentID: 0, TitleKey: "menu.system", Path: "/system"},
		{ID: 2, ParentID: 1, TitleKey: "menu.system.user", Path: "/system/user"},
	}
	got := filterMenusWithAncestors(menus, &MenuListQuery{Keyword: "menu."})
	if len(got) != 3 || got[0].ID != 4 || got[1].ID != 1 || got[2].ID != 2 {
		t.Fatalf("expected input order preserved [4 1 2], got %+v", got)
	}
}
