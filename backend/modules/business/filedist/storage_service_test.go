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

func TestResolveWithinRootKeepsPathsInsideStorage(t *testing.T) {
	root := t.TempDir()
	inside, err := resolveWithinRoot(root, filepath.Join("2026/10/29", "1730000000-report.txt"))
	if err != nil {
		t.Fatalf("expected an in-root path to resolve: %v", err)
	}
	if filepath.Dir(inside) != filepath.Join(root, "2026", "10", "29") {
		t.Fatalf("resolved path escaped the intended directory: %s", inside)
	}

	// A leading separator is joined as a relative segment, so it stays inside the root; only
	// segments that actually walk out of the root are refused.
	for _, escaped := range []string{"../outside.txt", "2026/../../outside.txt", "../../root.txt"} {
		if got, err := resolveWithinRoot(root, escaped); err == nil {
			t.Fatalf("expected %q to be refused, resolved to %s", escaped, got)
		}
	}
}

func TestSanitizeUploadFileNameReducesNameToOneElement(t *testing.T) {
	for name, want := range map[string]string{
		"report.txt":              "report.txt",
		"../../etc/passwd":        "passwd",
		`C:\Users\ops\report.txt`: "report.txt",
		"nested/dir/name.log":     "name.log",
	} {
		got, err := sanitizeUploadFileName(name)
		if err != nil {
			t.Fatalf("sanitize %q: %v", name, err)
		}
		if got != want {
			t.Fatalf("sanitize %q = %q, want %q", name, got, want)
		}
	}

	for _, bad := range []string{"", "   ", "..", "."} {
		if _, err := sanitizeUploadFileName(bad); err == nil {
			t.Fatalf("expected %q to be rejected", bad)
		}
	}
}
