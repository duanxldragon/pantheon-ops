package dashboard

import (
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
)

func TestSummaryLimitsAndFailClosedScope(t *testing.T) {
	for _, raw := range []string{"", "bad", "-1", "0", "99999999999999999", "9"} {
		if Limit(raw) != 8 {
			t.Fatalf("unsafe limit accepted: %s", raw)
		}
	}
	if Limit("2") != 2 {
		t.Fatal("smaller widget budget not honored")
	}
	for _, scope := range []*common.DataScopeReq{nil, {}, {Mode: "invalid"}} {
		if ScopeAllowed(scope) {
			t.Fatal("unknown or missing scope admitted")
		}
	}
	for _, mode := range []string{"all", "self", "dept", "dept_and_children", "custom"} {
		if !ScopeAllowed(&common.DataScopeReq{Mode: mode}) {
			t.Fatalf("known scope %s rejected", mode)
		}
	}
	snapshot := NewSnapshot(make([]Row, 9), 8, false)
	if len(snapshot.Items) != 8 || !snapshot.Partial || snapshot.SampledAt.IsZero() {
		t.Fatalf("sample does not expose its bound and freshness: %+v", snapshot)
	}
}
