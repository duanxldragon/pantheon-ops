package filedist

import (
	"encoding/json"
	"strings"
	"testing"

	"pantheon-base/modules/business/secrets"
	"pantheon-base/pkg/testmysql"
)

func TestStorageSecretsAreEncryptedAndRedacted(t *testing.T) {
	t.Setenv(secrets.BusinessSecretKeyEnv, strings.Repeat("00", 32))
	db := testmysql.Open(t)
	service := NewStorageService(db)
	if err := service.Migrate(); err != nil {
		t.Fatalf("migrate storage: %v", err)
	}

	response, err := service.CreateStorage(StorageDTO{
		Name: "secure-s3", Type: "s3", BasePath: "bucket/prefix",
		Config: map[string]interface{}{"accessKeyId": "key-id", "secretAccessKey": "top-secret"},
	}, "tester")
	if err != nil {
		t.Fatalf("create storage: %v", err)
	}
	if _, exposed := response.Config["secretAccessKey"]; exposed || response.Config["hasSecret"] != true {
		t.Fatalf("storage response did not redact secret: %+v", response.Config)
	}

	var record FileStorage
	if err := db.First(&record, response.ID).Error; err != nil {
		t.Fatalf("load storage: %v", err)
	}
	var stored map[string]interface{}
	if err := json.Unmarshal(record.Config, &stored); err != nil {
		t.Fatalf("decode stored config: %v", err)
	}
	secretValue, _ := stored["secretAccessKey"].(string)
	if secretValue == "top-secret" || !strings.HasPrefix(secretValue, "v1:") {
		t.Fatalf("storage secret was not encrypted: %q", secretValue)
	}
}
