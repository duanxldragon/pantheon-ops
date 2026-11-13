package businessauth

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/pkg/tenant"
)

func TestEphemeralTypesDoNotSerialize(t *testing.T) {
	for _, value := range []any{Identity{ActorID: 7, AccessToken: "synthetic-secret"}, Principal{Identity: Identity{AccessToken: "synthetic-secret"}}, Input{Identity: Identity{ActorID: 7}, Resolution: tenant.Context{TenantID: 9}}, RoleBinding{TenantID: 9, RoleKey: "admin"}, Action{Path: "secret"}, Authorization{Bindings: []RoleBinding{{RoleKey: "admin"}}}} {
		raw, err := json.Marshal(value)
		if err != nil || string(raw) != "{}" {
			t.Fatal("ephemeral authority leaked into JSON")
		}
	}
	if _, err := ReadSession(context.Background(), nil, "synthetic-token"); err == nil {
		t.Fatal("nil store accepted")
	}
	if _, err := FromGin(nil); err == nil {
		t.Fatal("nil checkpoint accepted")
	}
	if _, err := Reauthorize(nil, nil); err == nil {
		t.Fatal("nil recheck accepted")
	}
}
