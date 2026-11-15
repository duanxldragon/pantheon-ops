package cluster

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/testmysql"
)

func TestClusterDashboardSummaryNoCredentialLoadsAndScopedBound(t *testing.T) {
	db := testmysql.Open(t)
	// Deliberately create no credential table: summaries must use only registration metadata.
	if err := db.AutoMigrate(&Cluster{}); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 12; i++ {
		row := Cluster{Code: string(rune('a' + i)), Name: "cluster", Status: "healthy", Environment: "test", DeptID: uint64(i%2 + 71), KubeconfigCredentialRefID: 99, KubeconfigEncrypted: "secret-marker", APIServer: "https://private-endpoint-marker"}
		if err := db.Create(&row).Error; err != nil {
			t.Fatal(err)
		}
	}
	svc := NewClusterService(db)
	resp, err := svc.DashboardSummary(context.Background(), 3, &common.DataScopeReq{Mode: "dept", DeptID: 71})
	if err != nil || len(resp.Items) != 3 || !resp.Partial {
		t.Fatalf("scoped metadata query: %+v %v", resp, err)
	}
	for _, row := range resp.Items {
		if row.ID%2 != 1 {
			t.Fatalf("other department cluster leaked: %+v", row)
		}
	}
	bytes, _ := json.Marshal(resp)
	if strings.Contains(string(bytes), "secret-marker") || strings.Contains(string(bytes), "private-endpoint-marker") {
		t.Fatal("credential or endpoint entered summary")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := svc.DashboardSummary(ctx, 8, &common.DataScopeReq{Mode: "all"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled summary: %v", err)
	}
	if _, err := svc.DashboardSummary(context.Background(), 8, nil); err == nil {
		t.Fatal("missing scope admitted")
	}
}
