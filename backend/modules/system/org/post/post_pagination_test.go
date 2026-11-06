package post

import (
	"fmt"
	"slices"
	"testing"
)

func TestPostPaginationWithTiedSort(t *testing.T) {
	db := setupPostTestDB(t)
	const count = 205
	rows := make([]SystemPost, 0, count)
	for id := count; id > 0; id-- {
		rows = append(rows, SystemPost{ID: uint64(id), PostCode: fmt.Sprintf("tied_%03d", id), PostName: "Tied sort", Sort: 0, Status: 1})
	}
	if err := db.Create(&rows).Error; err != nil {
		t.Fatal(err)
	}
	service := NewPostService(db)
	for _, order := range []string{"asc", "desc"} {
		t.Run(order, func(t *testing.T) {
			ids := collectPostPageIDs(t, service, order)
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

func collectPostPageIDs(t *testing.T, service *PostService, order string) []uint64 {
	t.Helper()
	var ids []uint64
	for page := 1; page <= 3; page++ {
		result, err := service.ListPosts(&PostListQuery{Page: page, PageSize: 100, SortField: "sort", SortOrder: order})
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
