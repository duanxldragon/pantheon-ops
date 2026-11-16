package observability

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/tenant"
	"github.com/duanxldragon/pantheon-base/backend/pkg/testmysql"
)

func TestDatasourceDashboardSummaryTenantScopeBoundAndSecrets(t *testing.T) {
	db := testmysql.Open(t)
	if err := db.AutoMigrate(&PrometheusDatasource{}); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 12; i++ {
		row := PrometheusDatasource{TenantID: uint64(i%2 + 11), DeptID: 71, Name: string(rune('a' + i)), Type: "prometheus", Status: "healthy", URL: "https://private-endpoint-marker", Password: "secret-marker"}
		if err := db.Create(&row).Error; err != nil {
			t.Fatal(err)
		}
	}
	svc, err := NewDatasourceService(db).WithTenantContext(&tenant.Context{TenantID: 11, Mode: tenant.ModeMulti})
	if err != nil {
		t.Fatal(err)
	}
	resp, err := svc.DashboardSummary(context.Background(), 3, &common.DataScopeReq{Mode: "dept", DeptID: 71})
	if err != nil || len(resp.Items) != 3 || !resp.Partial {
		t.Fatalf("tenant summary: %+v %v", resp, err)
	}
	for _, row := range resp.Items {
		if row.ID%2 != 1 {
			t.Fatalf("other tenant datasource leaked: %+v", row)
		}
	}
	bytes, _ := json.Marshal(resp)
	if strings.Contains(string(bytes), "secret-marker") || strings.Contains(string(bytes), "private-endpoint-marker") {
		t.Fatal("credential or endpoint entered summary")
	}
	if _, err := NewDatasourceService(db).DashboardSummary(context.Background(), 8, &common.DataScopeReq{Mode: "all"}); err == nil {
		t.Fatal("unbound tenant admitted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := svc.DashboardSummary(ctx, 8, &common.DataScopeReq{Mode: "all"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("pre-canceled summary: %v", err)
	}
}

func TestAlertRuleDashboardRecordedErrorsScopeAndProjection(t *testing.T) {
	db := testmysql.Open(t)
	if err := db.AutoMigrate(&PrometheusDatasource{}, &AlertRule{}); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 12; i++ {
		source := PrometheusDatasource{TenantID: uint64(i % 2), DeptID: 71, Name: fmt.Sprintf("source-%d", i), Type: "prometheus", Status: "healthy", URL: "secret-endpoint"}
		if err := db.Create(&source).Error; err != nil {
			t.Fatal(err)
		}
		rule := AlertRule{Name: fmt.Sprintf("rule-%d", i), DatasourceID: source.ID, DeptID: 71, Query: "secret-query", Status: "error", Severity: "warning"}
		if i == 0 {
			rule.Status = "active"
		}
		if i == 2 {
			rule.DeptID = 72
		}
		if err := db.Create(&rule).Error; err != nil {
			t.Fatal(err)
		}
	}
	svc := &AlertRuleService{db: db}
	resp, err := svc.DashboardAlertRules(context.Background(), 2, &common.DataScopeReq{Mode: "dept", DeptID: 71})
	if err != nil || len(resp.Items) != 2 || !resp.Partial {
		t.Fatalf("scoped errors: %+v %v", resp, err)
	}
	for _, row := range resp.Items {
		if row.ID%2 != 1 || row.ID <= 3 || row.Status != "error" {
			t.Fatalf("hidden/nonerror rule leaked: %+v", row)
		}
	}
	encoded, _ := json.Marshal(resp)
	if strings.Contains(string(encoded), "secret") || strings.Contains(string(encoded), "query") || strings.Contains(string(encoded), "datasourceId") {
		t.Fatalf("unsafe projection: %s", encoded)
	}
	if _, err := svc.DashboardAlertRules(context.Background(), 8, nil); err == nil {
		t.Fatal("nil scope admitted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := svc.DashboardAlertRules(ctx, 8, &common.DataScopeReq{Mode: "all"}); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation lost: %v", err)
	}
}
