package system

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestSystemModulesDeclareGroupsThroughContracts is the ARCH-01 ratchet.
//
// Every authenticated /system group used to spell out its middleware chain at
// the registration site (15 hand-written chains, three variations). Hand-written
// chains are where a missing DataScope or a missing capability gate hides: the
// route still compiles, still answers 200, and silently has different
// permissions.
//
// This test reads the source instead of the router because building the real
// dependency graph needs MySQL and Redis. It fails if a chain is written by hand
// again, which forces the new group through contracts.System*Group and its
// pinned chain test (pkg/contracts/route_groups_system_test.go).
func TestSystemModulesDeclareGroupsThroughContracts(t *testing.T) {
	text := readSystemModulesSource(t)

	if count := strings.Count(text, ".Group(routeGroupSystem)."); count != 0 {
		t.Fatalf("system_modules.go chains .Group(routeGroupSystem) %d time(s); route groups must be declared through contracts.System*Group", count)
	}
	if count := strings.Count(text, ".Use(middleware."); count != 0 {
		t.Fatalf("system_modules.go installs middleware.Middleware by hand %d time(s); declare the group through contracts.System*Group instead", count)
	}

	// Positive half: the four helpers the modules are supposed to use must
	// actually be in use, so deleting a chain cannot satisfy the test above by
	// deleting the route.
	for _, helper := range []string{
		"contracts.SystemTokenGroup(",
		"contracts.SystemAuthGroup(",
		"contracts.SystemProtectedGroup(",
	} {
		if !strings.Contains(text, helper) {
			t.Fatalf("system_modules.go never calls %s", helper)
		}
	}
}

// TestSystemModulesPinGroupOptions is the per-group half of the ARCH-01 ratchet.
//
// TestSystemModulesDeclareGroupsThroughContracts only checks that a chain is
// declared through contracts.System*Group. Deleting one WithCapability() or
// WithDataScope() still passes that test, and the route still compiles.
// Capability and DataScope are permission decisions, so each registration is
// pinned here. RefreshSync is the bool passed to systemGroupOptions, not a
// separate option type.
//
// The expected strings are the registration lines as written, so a boolean
// change fails this test. Route-level RefreshSyncMiddleware (dict, setting,
// i18n writes) is not a group option and is not pinned here.
func TestSystemModulesPinGroupOptions(t *testing.T) {
	text := readSystemModulesSource(t)

	want := []string{
		// refresh-sync: token only, no tenant context, no Casbin.
		`contracts.SystemTokenGroup(r, routeGroupSystem, database.RDB)`,
		// user console, plus the list/export group that adds DataScope.
		`systemProtected := contracts.SystemProtectedGroup(r, routeGroupSystem, systemGroupOptions(deps, true))
				systemDataScoped := contracts.SystemProtectedGroup(r, routeGroupSystem, systemGroupOptions(deps, true).WithDataScope())`,
		// menu, role, permission, dept, post: refresh on, no capability, no data scope.
		`systemProtected := contracts.SystemProtectedGroup(r, routeGroupSystem, systemGroupOptions(deps, true))`,
		// tenant console and i18n console: capability gate, refresh differs.
		`systemGroupOptions(deps, true).WithCapability()`,
		`systemGroupOptions(deps, false).WithCapability()`,
		// dict, setting console, audit: no group-level refresh.
		`systemProtected := contracts.SystemProtectedGroup(r, routeGroupSystem, systemGroupOptions(deps, false))`,
		// upload serve and upload write: token + tenant context, no Casbin.
		`uploadProtected := contracts.SystemAuthGroup(r, routeGroupSystem, systemGroupOptions(deps, false))`,
		`systemAuth := contracts.SystemAuthGroup(r, routeGroupSystem, systemGroupOptions(deps, false))`,
	}
	for _, snippet := range want {
		if !strings.Contains(text, snippet) {
			t.Fatalf("system_modules.go lost a pinned group option:\n%s", snippet)
		}
	}

	if got, want := strings.Count(text, ".WithDataScope()"), 1; got != want {
		t.Fatalf("WithDataScope() count = %d, want %d (user list/export only)", got, want)
	}
	if got, want := strings.Count(text, ".WithCapability()"), 2; got != want {
		t.Fatalf("WithCapability() count = %d, want %d (tenant console and i18n console)", got, want)
	}
	if got, want := strings.Count(text, "systemGroupOptions(deps, true)"), 8; got != want {
		t.Fatalf("group-level refresh count = %d, want %d (user x2, menu, role, permission, tenant, dept, post)", got, want)
	}
	if got, want := strings.Count(text, "systemGroupOptions(deps, false)"), 6; got != want {
		t.Fatalf("group-level no-refresh count = %d, want %d (dict, upload x2, setting, i18n, audit)", got, want)
	}
}

func readSystemModulesSource(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot resolve the path of system_modules_route_groups_test.go")
	}
	source, err := os.ReadFile(filepath.Join(filepath.Dir(thisFile), "system_modules.go"))
	if err != nil {
		t.Fatalf("read system_modules.go: %v", err)
	}
	return string(source)
}
