package middleware

import (
	"context"
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/pkg/contracts/businessauth"
)

func TestBusinessDataScopeNilFailsClosed(t *testing.T) {
	if _, err := ReadBusinessDataScope(context.Background(), nil, 7, []businessauth.RoleBinding{{RoleKey: "admin"}}); err == nil {
		t.Fatal("nil DB accepted")
	}
}
