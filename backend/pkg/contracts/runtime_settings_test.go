package contracts

import (
	"errors"
	"reflect"
	"testing"
)

func TestRuntimeSettingReloadersNotifyInNameOrder(t *testing.T) {
	calls := make([]string, 0)
	unregisterB := RegisterRuntimeSettingReloader("b", func() error {
		calls = append(calls, "b")
		return nil
	})
	defer unregisterB()
	unregisterA := RegisterRuntimeSettingReloader("a", func() error {
		calls = append(calls, "a")
		return nil
	})
	defer unregisterA()

	if err := NotifyRuntimeSettingsChanged(); err != nil {
		t.Fatalf("notify runtime settings: %v", err)
	}
	if !reflect.DeepEqual(calls, []string{"a", "b"}) {
		t.Fatalf("expected deterministic reload order, got %v", calls)
	}
}

func TestRuntimeSettingReloadersReturnsReloadErrors(t *testing.T) {
	unregister := RegisterRuntimeSettingReloader("broken", func() error {
		return errors.New("boom")
	})
	defer unregister()

	err := NotifyRuntimeSettingsChanged()
	if err == nil || err.Error() != "runtime settings reload failed: broken: boom" {
		t.Fatalf("expected aggregated reload error, got %v", err)
	}
}
