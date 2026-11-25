package observability

import (
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/modules/business/shared/tenantschematest"
)

func TestTenantOwnership(t *testing.T) {
	tenantschematest.RequireTenantOwned(t, &MonitoringTarget{}, &AlertRule{}, &AlertEvent{}, &NotificationChannel{}, &NotificationLog{})
	tenantschematest.RequireTenantFirstUnique(t, &AlertRule{}, "uk_obs_alert_rule_name", "tenant_id", "name")
	tenantschematest.RequireTenantFirstUnique(t, &NotificationChannel{}, "uk_obs_notification_channel_name", "tenant_id", "name")
}
