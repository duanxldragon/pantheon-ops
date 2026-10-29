package common

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNormalizeSortWhitelistAndFallback(t *testing.T) {
	allowed := map[string]string{
		"id":         "id",
		"createdAt":  "created_at",
		"created_at": "created_at",
	}
	for _, tc := range []struct {
		name     string
		input    string
		expected string
	}{
		{"exact alias", "createdAt", "created_at"},
		{"snake alias", "created_at", "created_at"},
		{"padded alias", "  created_at\t", "created_at"},
		{"unknown symbol", "password_hash", "id"},
		{"blank", "", "id"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expected, NormalizeSort(tc.input, allowed, "id"))
		})
	}

	t.Run("nil map falls back", func(t *testing.T) {
		require.Equal(t, "sort", NormalizeSort("id", nil, "sort"))
	})
}
