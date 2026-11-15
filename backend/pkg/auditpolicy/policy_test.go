package auditpolicy

import (
	"github.com/gin-gonic/gin"
	"net/http/httptest"
	"testing"
)

func TestExactSealedRegistration(t *testing.T) {
	registry.Lock()
	registry.routes = nil
	registry.sealed = false
	registry.Unlock()
	t.Cleanup(func() { registry.Lock(); registry.routes = nil; registry.sealed = false; registry.Unlock() })
	route := Route{Method: "POST", Template: "/audit/items/:id"}
	for _, bad := range []Route{{Method: "post", Template: route.Template}, {Method: "POST", Template: "/audit/*all"}, {Method: "POST", Template: "/audit/items?q=secret"}} {
		if RegisterMetadataOnly([]Route{bad}) == nil {
			t.Fatal("invalid registration accepted")
		}
	}
	if err := RegisterMetadataOnly([]Route{route}); err != nil {
		t.Fatal(err)
	}
	if RegisterMetadataOnly([]Route{route}) == nil {
		t.Fatal("conflicting registration accepted")
	}
	if ValidateRoutes(nil) == nil {
		t.Fatal("missing route accepted")
	}
	engine := gin.New()
	matched := false
	engine.Use(func(c *gin.Context) { _, matched = Match(c); c.AbortWithStatus(403) })
	engine.POST(route.Template, func(c *gin.Context) { t.Error("denied handler ran") })
	if err := ValidateRoutes(engine.Routes()); err != nil {
		t.Fatal(err)
	}
	engine.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("POST", "/audit/items/secret?raw=secret", nil))
	if !matched {
		t.Fatal("exact FullPath unavailable before global denial")
	}
	matched = false
	engine.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("GET", "/audit/items/secret", nil))
	if matched {
		t.Fatal("wrong method matched")
	}
	if err := Seal(); err != nil {
		t.Fatal(err)
	}
	if RegisterMetadataOnly([]Route{{Method: "POST", Template: "/late"}}) == nil {
		t.Fatal("registration after seal accepted")
	}
}

func TestSafeOutcomePrecedence(t *testing.T) {
	route := Route{Method: "POST", Template: "/outcome/items/:id"}
	if err := RegisterMetadataOnly([]Route{route}); err != nil {
		t.Fatal(err)
	}
	engine := gin.New()
	engine.POST(route.Template, func(c *gin.Context) {
		SetMetadata(c, SafeMetadata{Outcome: OutcomeSuccess})
		SetOutcome(c, OutcomeDenied)
		if Metadata(c).Outcome == OutcomeSuccess {
			t.Fatal("business HTTP200 failure audited success")
		}
		SetMetadata(c, SafeMetadata{ActorID: 7, Outcome: OutcomeStorage})
		if Metadata(c).Outcome != OutcomeStorage {
			t.Fatal("safe category overwritten")
		}
		SetMetadata(c, SafeMetadata{ActorID: 7, Count: 10001})
		if Metadata(c).ActorID != 0 {
			t.Fatal("invalid metadata not reset")
		}
	})
	engine.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest("POST", "/outcome/items/1", nil))
}
