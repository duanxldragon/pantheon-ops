package middleware

import "testing"

func TestIsSelfServiceRouteBySignature_MenuTreeScopeBoundary(t *testing.T) {
	if !isSelfServiceRouteBySignature("/api/v1/system/menu/tree", "GET", "nav") {
		t.Fatalf("expected nav menu tree to be self-service")
	}
	if !isSelfServiceRouteBySignature("/api/v1/system/menu/tree", "GET", "") {
		t.Fatalf("expected default menu tree scope to be self-service")
	}
	if isSelfServiceRouteBySignature("/api/v1/system/menu/tree", "GET", "manage") {
		t.Fatalf("expected manage menu tree to remain protected")
	}
}
