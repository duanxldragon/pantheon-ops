package observability

import (
	"encoding/json"
	"strings"
	"testing"

	"pantheon-base/modules/business/secrets"
	"pantheon-base/pkg/testmysql"
)

func TestNotificationSecretsAreEncryptedAndRedacted(t *testing.T) {
	t.Setenv(secrets.BusinessSecretKeyEnv, strings.Repeat("11", 32))
	db := testmysql.Open(t)
	service := NewNotificationService(db)
	if err := service.Migrate(); err != nil {
		t.Fatalf("migrate notifications: %v", err)
	}

	response, err := service.CreateChannel(NotificationChannelDTO{
		Name: "secure-mail", Type: "email", Enabled: true,
		Config: map[string]interface{}{"smtpHost": "smtp.example", "password": "top-secret"},
	}, "tester")
	if err != nil {
		t.Fatalf("create channel: %v", err)
	}
	if _, exposed := response.Config["password"]; exposed || response.Config["hasSecret"] != true {
		t.Fatalf("notification response did not redact secret: %+v", response.Config)
	}

	var record NotificationChannel
	if err := db.First(&record, response.ID).Error; err != nil {
		t.Fatalf("load channel: %v", err)
	}
	var stored map[string]interface{}
	if err := json.Unmarshal(record.Config, &stored); err != nil {
		t.Fatalf("decode stored config: %v", err)
	}
	secretValue, _ := stored["password"].(string)
	if secretValue == "top-secret" || !strings.HasPrefix(secretValue, "v1:") {
		t.Fatalf("notification secret was not encrypted: %q", secretValue)
	}
}
