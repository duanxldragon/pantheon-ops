package common

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizePaginationDefaultsAndCap(t *testing.T) {
	for _, tc := range []struct {
		name         string
		page         int
		pageSize     int
		maxPageSize  int
		wantPage     int
		wantPageSize int
		wantOffset   int
	}{
		{"zero page and size", 0, 0, 100, 1, 10, 0},
		{"negative page and size", -5, -1, 100, 1, 10, 0},
		{"valid middle", 3, 20, 100, 3, 20, 40},
		{"oversized pageSize capped", 7, 5000, 100, 7, 100, 600},
		{"cap below default applies", 2, 0, 5, 2, 5, 5},
	} {
		t.Run(tc.name, func(t *testing.T) {
			page, pageSize, offset := NormalizePagination(tc.page, tc.pageSize, tc.maxPageSize)
			require.Equal(t, tc.wantPage, page)
			require.Equal(t, tc.wantPageSize, pageSize)
			require.Equal(t, tc.wantOffset, offset)
		})
	}
}
