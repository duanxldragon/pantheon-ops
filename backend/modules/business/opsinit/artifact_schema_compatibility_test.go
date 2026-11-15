package opsinit

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/modules/business/observability"
	"github.com/duanxldragon/pantheon-base/backend/pkg/testmysql"
)

func o4TestArtifact(t *testing.T, capable bool) O4ArtifactPins {
	t.Helper()
	root := t.TempDir()
	write := func(name string, body []byte) (string, string) {
		path := filepath.Join(root, name)
		if err := os.WriteFile(path, body, 0600); err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(body)
		return path, hex.EncodeToString(sum[:])
	}
	descriptorPath, descriptorSHA := write("descriptor.json", []byte(`{"kind":"pantheon-base-candidate","candidateIdentity":"test-owned-candidate"}`))
	name := "fixture-old.exe"
	caps := []string{}
	if capable {
		name = "fixture-new.exe"
		caps = []string{observability.QueryWorkspaceCapability}
	}
	binaryPath, binarySHA := write(name, []byte("test-owned-executable-fixture-"+name))
	receipt := O4BuildReceipt{Kind: "pantheon-o4-build-v1", DescriptorSHA256: descriptorSHA, CandidateIdentity: "test-owned-candidate", ExecutableSHA256: binarySHA, Capabilities: caps, ManagedProcessNames: []string{"fixture-new.exe", "fixture-old.exe"}, BuiltAt: time.Now().UTC()}
	bytes, _ := json.Marshal(receipt)
	receiptPath, receiptSHA := write("build-receipt.json", bytes)
	return O4ArtifactPins{DescriptorPath: descriptorPath, DescriptorSHA256: descriptorSHA, BuildReceiptPath: receiptPath, BuildReceiptSHA256: receiptSHA, ExecutablePath: binaryPath, ExecutableSHA256: binarySHA}
}
func TestO4SupportedStartArtifactAndRetainedSchemaMatrix(t *testing.T) {
	db := testmysql.Open(t)
	ctx := context.Background()
	old, newer := o4TestArtifact(t, false), o4TestArtifact(t, true)
	empty := func(context.Context) ([]O4Process, error) { return nil, nil }
	if _, err := CheckO4SupportedStart(ctx, db, old, empty); err != nil {
		t.Fatal("known old empty startup refused")
	}
	if _, err := CheckO4SupportedStart(ctx, db, newer, empty); err != nil {
		t.Fatal("capable clean transition refused")
	}
	t.Setenv("PANTHEON_AUTO_MIGRATE", "true")
	if err := observability.BootstrapQueryWorkspaceDevelopment(db); err != nil {
		t.Fatal(err)
	}
	if _, err := CheckO4SupportedStart(ctx, db, old, empty); err == nil {
		t.Fatal("old artifact accepted retained O4")
	}
	if _, err := CheckO4SupportedStart(ctx, db, newer, empty); err != nil {
		t.Fatal(err)
	}
	oldProcess := func(context.Context) ([]O4Process, error) {
		return []O4Process{{PID: 42, Name: "fixture-old.exe", ExecutablePath: old.ExecutablePath}}, nil
	}
	if _, err := CheckO4SupportedStart(ctx, db, newer, oldProcess); err == nil {
		t.Fatal("old/new managed writers coexisted")
	}
	if err := db.Exec("CREATE TABLE schema_migrations (version BIGINT NOT NULL PRIMARY KEY,dirty BOOLEAN NOT NULL)").Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("INSERT INTO schema_migrations VALUES (36,false)").Error; err != nil {
		t.Fatal(err)
	}
	if _, err := CheckO4SupportedStart(ctx, db, old, empty); err == nil {
		t.Fatal("retained down36/AutoMigrate enabled old artifact")
	}
	if err := db.Exec("UPDATE schema_migrations SET dirty=true").Error; err != nil {
		t.Fatal(err)
	}
	if _, err := CheckO4SupportedStart(ctx, db, newer, empty); err == nil {
		t.Fatal("dirty ordinary startup accepted")
	}
	if err := db.Exec("UPDATE schema_migrations SET dirty=false").Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Exec("DELETE FROM obs_schema_capability").Error; err != nil {
		t.Fatal(err)
	}
	for _, artifact := range []O4ArtifactPins{old, newer} {
		if _, err := CheckO4SupportedStart(ctx, db, artifact, empty); err == nil {
			t.Fatal("missing marker bypassed actual retained shape")
		}
	}
}
func TestO4SupportedStartBindsExecutableBytes(t *testing.T) {
	db := testmysql.Open(t)
	artifact := o4TestArtifact(t, true)
	if err := os.WriteFile(artifact.ExecutablePath, []byte("unknown-different-binary"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := CheckO4SupportedStart(context.Background(), db, artifact, func(context.Context) ([]O4Process, error) { return nil, nil }); err == nil {
		t.Fatal("source receipt substituted for executable identity")
	}
}
