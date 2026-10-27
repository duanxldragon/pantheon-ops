package filedist

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/modules/business/secrets"
	"github.com/duanxldragon/pantheon-base/backend/pkg/testmysql"
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

func TestNormalizeLocalStoragePathRejectsParentTraversal(t *testing.T) {
	root := filepath.Join(string(filepath.Separator), "srv", "filedist")
	got, err := normalizeLocalStoragePath("  " + root + string(filepath.Separator) + " ")
	if err != nil {
		t.Fatalf("normalize %q: %v", root, err)
	}
	if got != root {
		t.Fatalf("expected the cleaned root %q, got %q", root, got)
	}

	traversal := strings.Join([]string{root, "..", "..", "etc"}, string(filepath.Separator))
	if _, err := normalizeLocalStoragePath(traversal); err == nil {
		t.Fatalf("expected %q to be rejected", traversal)
	}
	if _, err := normalizeLocalStoragePath("   "); err == nil {
		t.Fatal("expected an empty base path to be rejected")
	}
}

func TestCreateStorageRejectsTraversalBasePath(t *testing.T) {
	t.Setenv(secrets.BusinessSecretKeyEnv, strings.Repeat("00", 32))
	db := testmysql.Open(t)
	service := NewStorageServiceWithBasePath(db, t.TempDir())
	if err := service.Migrate(); err != nil {
		t.Fatalf("migrate storage: %v", err)
	}

	if _, err := service.CreateStorage(StorageDTO{
		Name: "escaped", Type: "local", BasePath: "../outside",
	}, "tester"); err == nil {
		t.Fatal("expected a traversal base path to be rejected")
	}
}
