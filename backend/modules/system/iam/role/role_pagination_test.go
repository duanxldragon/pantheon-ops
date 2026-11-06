package role

import (
	"fmt"
	"slices"
	"testing"
)

func TestRolePaginationWithTiedSort(t *testing.T) {
	db := setupRoleTestDB(t)
	const count = 205
	rows := make([]SystemRole, 0, count)
	for id := count; id > 0; id-- {
		rows = append(rows, SystemRole{ID: uint64(id), RoleKey: fmt.Sprintf("tied_%03d", id), RoleName: "Tied sort", Sort: 0, Status: 1})
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatal(err)
	}
	service := NewRoleService(db)
	for _, order := range []string{"asc", "desc"} {
		t.Run(order, func(t *testing.T) {
			ids := collectRolePageIDs(t, service, order)
			want := make([]uint64, count)
			for i := range want {
				want[i] = uint64(i + 1)
			}
			if order == "desc" {
				slices.Reverse(want)
			}
			if !slices.Equal(ids, want) {
				t.Fatalf("tied-sort pages must return every ID once in stable order: got %v, want %v", ids, want)
			}
		})
	}
}

func collectRolePageIDs(t *testing.T, service *RoleService, order string) []uint64 {
	t.Helper()
	var ids []uint64
	for page := 1; page <= 3; page++ {
		result, err := service.ListRoles(&RoleListQuery{Page: page, PageSize: 100, SortField: "sort", SortOrder: order})
		if err != nil {
			t.Fatal(err)
		}
		if result.Total != 205 {
			t.Fatalf("unexpected total: %d", result.Total)
		}
		for _, item := range result.Items {
			ids = append(ids, item.ID)
		}
	}
	return ids
}
