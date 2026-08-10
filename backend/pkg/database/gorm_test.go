package database

import (
	"testing"

	"gorm.io/gorm/logger"
)

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

func TestGormLogLevelUsesWarnInProduction(t *testing.T) {
	t.Setenv("PANTHEON_ENV", "production")
	if got := gormLogLevel(); got != logger.Warn {
		t.Fatalf("expected warn log level in production, got %v", got)
	}
}

func TestGormLogLevelUsesInfoOutsideProduction(t *testing.T) {
	t.Setenv("PANTHEON_ENV", "development")
	if got := gormLogLevel(); got != logger.Info {
		t.Fatalf("expected info log level outside production, got %v", got)
	}
}

func TestEnvInt(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  int
	}{
		{name: "empty uses default", value: "", want: 7},
		{name: "invalid uses default", value: "abc", want: 7},
		{name: "zero uses default", value: "0", want: 7},
		{name: "negative uses default", value: "-1", want: 7},
		{name: "positive value", value: "12", want: 12},
		{name: "trimmed positive value", value: " 15 ", want: 15},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("PANTHEON_TEST_ENV_INT", tt.value)
			if got := envInt("PANTHEON_TEST_ENV_INT", 7); got != tt.want {
				t.Fatalf("envInt() = %d, want %d", got, tt.want)
			}
		})
	}
}
