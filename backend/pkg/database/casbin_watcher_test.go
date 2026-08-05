package database

import "testing"

func TestInitCasbinWatcherDisabledDoesNotPanic(t *testing.T) {
	t.Setenv("PANTHEON_CASBIN_WATCHER", "false")
	if watcher := initCasbinWatcher(nil); watcher != nil {
		t.Fatalf("expected no watcher when disabled, got %#v", watcher)
	}
}

func TestInitCasbinWatcherEnabledWithoutRedisDoesNotPanic(t *testing.T) {
	t.Setenv("PANTHEON_CASBIN_WATCHER", "true")
	if watcher := initCasbinWatcher(nil); watcher != nil {
		t.Fatalf("expected no watcher when redis client is nil, got %#v", watcher)
	}
}
