package tenant

import (
	"context"
	"errors"
	"testing"
)

type strictFlagStub struct {
	value  string
	err    error
	calls  int
	cancel context.CancelFunc
}

func (s *strictFlagStub) GetByKey(key string) (string, error) {
	s.calls++
	if key != FeatureFlagSettingKey {
		panic("wrong flag")
	}
	if s.cancel != nil {
		s.cancel()
	}
	return s.value, s.err
}
func TestStrictModeReader(t *testing.T) {
	for _, value := range []string{"compat", "multi", `"compat"`, `"multi"`, "", "garbage", "COMPAT", `"compat`} {
		t.Run(value, func(t *testing.T) {
			assertStrictModeReaderValue(t, value)
		})
	}
	var nilStub *strictFlagStub
	for _, reader := range []ModeReader{NewStrictSettingModeReader(nil), NewStrictSettingModeReader(nilStub), NewStrictSettingModeReader(&strictFlagStub{err: errors.New("store failed")})} {
		if _, err := reader.ReadMode(context.Background()); err == nil {
			t.Fatal("unavailable reader authorized compat")
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	stub := &strictFlagStub{value: "compat"}
	if _, err := NewStrictSettingModeReader(stub).ReadMode(ctx); err == nil || stub.calls != 0 {
		t.Fatal("cancelled context read store")
	}
	ctx, cancel = context.WithCancel(context.Background())
	if _, err := NewStrictSettingModeReader(&strictFlagStub{value: "compat", cancel: cancel}).ReadMode(ctx); err == nil {
		t.Fatal("cancellation after read accepted")
	}
}

func assertStrictModeReaderValue(t *testing.T, value string) {
	t.Helper()
	stub := &strictFlagStub{value: value}
	reader := NewStrictSettingModeReader(stub)
	mode, err := reader.ReadMode(context.Background())
	valid := value == "compat" || value == "multi" || value == `"compat"` || value == `"multi"`
	if (err == nil) != valid {
		t.Fatal("strict mode accepted an unknown/missing value")
	}
	if valid && mode != NormalizeFlagValue(value) {
		t.Fatal("wrong mode")
	}
	stub.value = "multi"
	if got, err := reader.ReadMode(context.Background()); got != "multi" || err != nil || stub.calls != 2 {
		t.Fatal("mode reader cached authority")
	}
}
