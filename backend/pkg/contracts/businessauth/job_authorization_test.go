package businessauth

import (
	"encoding/json"
	"testing"
)

func TestJobAuthorizationInputsDoNotSerialize(t *testing.T) {
	raw, err := json.Marshal(JobInput{ActorID: 7, ResourceTenantID: 101, Action: Action{Permission: "private", Path: "private"}})
	if err != nil || string(raw) != "{}" {
		t.Fatalf("job authorization input leaked: %s, %v", raw, err)
	}
}
