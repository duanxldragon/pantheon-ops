package dynamicmodule

import "testing"

func TestDynamicModuleEnabledDefaultsToTrueInDevelopment(t *testing.T) {
	t.Setenv("PANTHEON_ENV", "development")
	t.Setenv("PANTHEON_ENABLE_DYNAMIC_MODULES", "")

	if !dynamicModuleEnabled() {
		t.Fatal("expected dynamic modules to be enabled by default in development")
	}
}

func TestDynamicModuleEnabledDefaultsToFalseInProduction(t *testing.T) {
	t.Setenv("PANTHEON_ENV", "production")
	t.Setenv("PANTHEON_ENABLE_DYNAMIC_MODULES", "")

	if dynamicModuleEnabled() {
		t.Fatal("expected dynamic modules to be disabled by default in production")
	}
}

func TestDynamicModuleEnabledExplicitOverride(t *testing.T) {
	t.Setenv("PANTHEON_ENV", "production")
	t.Setenv("PANTHEON_ENABLE_DYNAMIC_MODULES", "true")

	if !dynamicModuleEnabled() {
		t.Fatal("expected explicit override to enable dynamic modules")
	}
}
