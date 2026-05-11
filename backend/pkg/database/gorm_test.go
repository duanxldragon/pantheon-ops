package database

import "testing"

func TestNormalizeMySQLDSN(t *testing.T) {
	result, err := normalizeMySQLDSN("root:pass@tcp(127.0.0.1:3306)/pantheon")
	if err != nil {
		t.Fatalf("expected mysql dsn accepted, got %v", err)
	}
	if result == "" {
		t.Fatalf("expected normalized mysql dsn")
	}
}

func TestNormalizeMySQLDSNRejectsSQLiteStyleDSN(t *testing.T) {
	if _, err := normalizeMySQLDSN("test.db"); err == nil {
		t.Fatalf("expected sqlite-like dsn rejected")
	}
	if _, err := normalizeMySQLDSN(":memory:"); err == nil {
		t.Fatalf("expected sqlite memory dsn rejected")
	}
}
