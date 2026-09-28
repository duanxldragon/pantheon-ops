package login

import (
	"github.com/duanxldragon/pantheon-base/backend/pkg/tenant"
	"testing"
)

func TestLoginLogsAreTenantScoped(t *testing.T) {
	db := setupTestDB(t)
	r := NewRuntime(db, testCredentialRepo(db))
	r.WithTenantContext(&tenant.Context{TenantID: 101, Mode: tenant.ModeMulti}).RecordLoginLog("a", "alice", "1.1.1.1", "", "", 1, "ok")
	r.WithTenantContext(&tenant.Context{TenantID: 202, Mode: tenant.ModeMulti}).RecordLoginLog("b", "bob", "2.2.2.2", "", "", 1, "ok")
	page, err := r.WithTenantContext(&tenant.Context{TenantID: 101, Mode: tenant.ModeMulti}).ListLoginLogs(&LoginLogQuery{})
	if err != nil || page.Total != 1 || page.Items[0].Username != "alice" {
		t.Fatalf("tenant scope leaked: page=%+v err=%v", page, err)
	}
}
