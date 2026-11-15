package permissionroutes

import (
	"errors"
	"reflect"
	"sync"
	"testing"
)

func TestRegistryPreservesExistingResourcePermissionSpelling(t *testing.T) {
	for _, test := range []struct {
		permission string
		route      Route
	}{
		{"business:observability:metric_source:view", Route{Method: "POST", Path: "/api/v1/business/observability/datasources/:id/query"}},
		{"business:observability:query-history:view", Route{Method: "GET", Path: "/api/v1/business/observability/query-history"}},
		{"business:observability:saved-view:view", Route{Method: "GET", Path: "/api/v1/business/observability/saved-views"}},
	} {
		t.Run(test.permission, func(t *testing.T) {
			binding := Binding{Permission: test.permission, Routes: []Route{test.route}}
			for i := 0; i < 2; i++ {
				if err := Register("business:observability", []Binding{binding}); err != nil {
					t.Fatalf("existing permission spelling rejected: %v", err)
				}
			}
			want := []Route{test.route}
			if !reflect.DeepEqual(Lookup(test.permission), want) || !reflect.DeepEqual(Snapshot()[test.permission], want) {
				t.Fatal("registered permission did not retain its exact key and route")
			}
			binding.Routes[0].Path = "/changed-input"
			copied := Lookup(test.permission)
			copied[0].Path = "/changed-lookup"
			if !reflect.DeepEqual(Lookup(test.permission), want) {
				t.Fatal("input or lookup mutation changed registered binding")
			}
		})
	}
	if len(Lookup("business:observability:metric-source:view")) != 0 {
		t.Fatal("registration invented a normalized permission grant")
	}
}

func TestRegistryLegacyPermissionSpellingKeepsBoundaries(t *testing.T) {
	const module = "business:registrycompatguard"
	const permission = module + ":metric_source:view"
	const path = "/api/v1/business/registrycompatguard/datasources/:id/query"
	for _, test := range []struct {
		name       string
		module     string
		permission string
		route      Route
	}{
		{"underscore-module", "business:registry_compatguard", "business:registry_compatguard:metric_source:view", Route{Method: "POST", Path: "/api/v1/business/registry_compatguard/datasources/:id/query"}},
		{"foreign-permission", module, "business:other:metric_source:view", Route{Method: "POST", Path: path}},
		{"system-permission", module, "system:metric_source:view", Route{Method: "POST", Path: path}},
		{"leading-underscore", module, module + ":_metric_source:view", Route{Method: "POST", Path: path}},
		{"permission-wildcard", module, module + ":metric_source:*", Route{Method: "POST", Path: path}},
		{"method-wildcard", module, permission, Route{Method: "*", Path: path}},
		{"foreign-path", module, permission, Route{Method: "POST", Path: "/api/v1/business/other/datasources/:id/query"}},
		{"path-traversal", module, permission, Route{Method: "POST", Path: "/api/v1/business/registrycompatguard/../other/query"}},
		{"path-wildcard", module, permission, Route{Method: "POST", Path: "/api/v1/business/registrycompatguard/*"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			before := Snapshot()
			valid := Binding{Permission: module + ":pending", Routes: []Route{{Method: "GET", Path: "/api/v1/business/registrycompatguard/pending"}}}
			invalid := Binding{Permission: test.permission, Routes: []Route{test.route}}
			if err := Register(test.module, []Binding{valid, invalid}); !errors.Is(err, ErrInvalid) {
				t.Fatalf("invalid binding accepted: %v", err)
			}
			if !reflect.DeepEqual(Snapshot(), before) {
				t.Fatal("invalid batch changed registry")
			}
		})
	}
}

func TestRegistryOwnsImmutableAtomicBindings(t *testing.T) {
	binding := Binding{Permission: "business:registrytest:read", Routes: []Route{{Method: "GET", Path: "/api/v1/business/registrytest/items/:id"}}}
	if err := Register("business:registrytest", []Binding{binding}); err != nil {
		t.Fatal(err)
	}
	binding.Routes[0].Path = "/changed"
	snapshot := Snapshot()
	if snapshot["business:registrytest:read"][0].Path != "/api/v1/business/registrytest/items/:id" {
		t.Fatal("input mutation leaked")
	}
	snapshot["business:registrytest:read"][0].Path = "/changed"
	if Snapshot()["business:registrytest:read"][0].Path == "/changed" {
		t.Fatal("snapshot mutation leaked")
	}
	if err := Register("business:registrytest", []Binding{{Permission: "business:registrytest:new", Routes: []Route{{Method: "GET", Path: "/api/v1/business/registrytest/new"}}}, {Permission: "business:registrytest:read", Routes: []Route{{Method: "POST", Path: "/api/v1/business/registrytest/items/:id"}}}}); !errors.Is(err, ErrConflict) {
		t.Fatalf("conflict accepted: %v", err)
	}
	if _, ok := Snapshot()["business:registrytest:new"]; ok {
		t.Fatal("conflicting batch partially committed")
	}
}
func TestRegistryRejectsPrivilegeAndWildcardEscapes(t *testing.T) {
	for _, binding := range []Binding{
		{Permission: "system:user:create", Routes: []Route{{Method: "POST", Path: "/api/v1/system/user"}}},
		{Permission: "business:registrybad:read", Routes: []Route{{Method: "*", Path: "/api/v1/business/registrybad/items"}}},
		{Permission: "business:registrybad:read", Routes: []Route{{Method: "GET", Path: "/api/v1/business/other/items"}}},
		{Permission: "business:registrybad:read", Routes: []Route{{Method: "GET", Path: "/api/v1/business/registrybad/*"}}},
	} {
		if err := Register("business:registrybad", []Binding{binding}); !errors.Is(err, ErrInvalid) {
			t.Fatalf("bad binding: %v", err)
		}
	}
}
func TestRegistryConcurrentCopies(t *testing.T) {
	var group sync.WaitGroup
	binding := Binding{Permission: "business:registryrace:read", Routes: []Route{{Method: "GET", Path: "/api/v1/business/registryrace/items"}}}
	for i := 0; i < 20; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			if err := Register("business:registryrace", []Binding{binding}); err != nil {
				t.Error(err)
			}
			_ = Snapshot()
		}()
	}
	group.Wait()
}

func TestRegistrySealFreezesRegistrationButKeepsReads(t *testing.T) {
	const module = "business:registryseal"
	const permission = module + ":read"
	const path = "/api/v1/business/registryseal/items"
	want := []Route{{Method: "GET", Path: path}}
	binding := Binding{Permission: permission, Routes: []Route{{Method: "GET", Path: path}}}

	// The registry is process-scoped; this test owns the seal flag and its own keys.
	registry.Lock()
	previousSealed := registry.sealed
	registry.sealed = false
	registry.Unlock()
	t.Cleanup(func() {
		registry.Lock()
		defer registry.Unlock()
		registry.sealed = previousSealed
		delete(registry.bindings, permission)
	})

	if err := Register(module, []Binding{binding}); err != nil {
		t.Fatalf("registration before seal rejected: %v", err)
	}
	if err := Seal(); err != nil {
		t.Fatalf("seal failed: %v", err)
	}
	if err := Seal(); err != nil {
		t.Fatalf("repeat seal failed: %v", err)
	}

	// Fail closed for every mutation path, including an identical replay.
	before := Snapshot()
	for _, test := range []struct {
		name     string
		module   string
		bindings []Binding
	}{
		{"identical-replay", module, []Binding{binding}},
		{"new-key", module, []Binding{{Permission: module + ":write", Routes: []Route{{Method: "POST", Path: "/api/v1/business/registryseal/items"}}}}},
		{"new-module", "business:registrysealafter", []Binding{{Permission: "business:registrysealafter:read", Routes: []Route{{Method: "GET", Path: "/api/v1/business/registrysealafter/items"}}}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := Register(test.module, test.bindings); !errors.Is(err, ErrSealed) {
				t.Fatalf("registration after seal accepted: %v", err)
			}
			if !reflect.DeepEqual(Snapshot(), before) {
				t.Fatal("rejected registration after seal changed registry")
			}
		})
	}

	// Read paths keep working and keep returning deep copies.
	if !reflect.DeepEqual(Lookup(permission), want) {
		t.Fatal("sealed lookup lost the binding")
	}
	snapshot := Snapshot()
	if !reflect.DeepEqual(snapshot[permission], want) {
		t.Fatal("sealed snapshot lost the binding")
	}
	snapshot[permission][0].Path = "/changed"
	if !reflect.DeepEqual(Lookup(permission), want) {
		t.Fatal("sealed snapshot mutation leaked")
	}
}

func TestRegistryRejectsMalformedRouterParameters(t *testing.T) {
	for _, suffix := range []string{"/:", "/:id:other", "/items:name", "/:1id", "/items/", "//items", "/../system/user", "/%2fuser", "/items?all=1"} {
		err := Register("business:registrysegments", []Binding{{Permission: "business:registrysegments:read", Routes: []Route{{Method: "GET", Path: "/api/v1/business/registrysegments" + suffix}}}})
		if !errors.Is(err, ErrInvalid) {
			t.Fatalf("invalid router path accepted %q: %v", suffix, err)
		}
	}
}
