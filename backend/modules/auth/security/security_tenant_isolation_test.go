package security

import (
	"github.com/duanxldragon/pantheon-base/backend/pkg/tenant"
	"testing"
)

func TestSecurityEventsAreTenantScoped(t *testing.T) {
	svc := newSecurityEventFixture(t)
	svc.db.Create(&SystemAuthSecurityEvent{TenantID: 101, Username: "alice", EventType: "x", Severity: "high", MessageKey: "x"})
	svc.db.Create(&SystemAuthSecurityEvent{TenantID: 202, Username: "bob", EventType: "x", Severity: "high", MessageKey: "x"})
	page, err := svc.WithTenantContext(&tenant.Context{TenantID: 101, Mode: tenant.ModeMulti}).ListSecurityEvents(&SecurityEventQuery{})
	if err != nil || page.Total != 1 || page.Items[0].Username != "alice" {
		t.Fatalf("tenant scope leaked: page=%+v err=%v", page, err)
	}
}
