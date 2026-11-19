package host

import (
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"
)

// stubConfigReader is a test config reader emulating a tenant-scoped setting
// view. A different instance (with different values) models a different tenant
// context; a nil map lookup models an unseeded key.
type stubConfigReader struct {
	values map[string]string
	err    error
	// missingKeys causes GetByKey to behave as if the key is absent (setting
	// service returns ErrRecordNotFound) no matter what values holds.
	missingKeys bool
	calls       []string
}

func (s *stubConfigReader) GetByKey(key string) (string, error) {
	s.calls = append(s.calls, key)
	if s.err != nil {
		return "", s.err
	}
	if s.missingKeys {
		return "", errors.New("record not found")
	}
	if v, ok := s.values[key]; ok {
		return v, nil
	}
	return "", errors.New("record not found")
}

func TestResolveCollectTimeoutDefaults(t *testing.T) {
	for name, reader := range map[string]ConfigReader{
		"nil reader":     nil,
		"missing key":    &stubConfigReader{values: map[string]string{}, missingKeys: true},
		"unrelated keys": &stubConfigReader{values: map[string]string{"some.other": "1"}},
		"read error":     &stubConfigReader{err: errors.New("boom")},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := resolveCollectTimeout(reader)
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if got != defaultSSHCollectTTL {
				t.Errorf("expected default %s, got %s", defaultSSHCollectTTL, got)
			}
		})
	}
}

func TestResolveCollectTimeoutFromConfig(t *testing.T) {
	r := &stubConfigReader{values: map[string]string{sshCollectTimeoutKey: "30"}}
	got, err := resolveCollectTimeout(r)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got != 30*time.Second {
		t.Errorf("expected 30s, got %s", got)
	}
	if len(r.calls) != 1 || r.calls[0] != sshCollectTimeoutKey {
		t.Errorf("expected single read of %s, got %v", sshCollectTimeoutKey, r.calls)
	}
}

func TestResolveCollectTimeoutBoundedFallback(t *testing.T) {
	oversized := fmt.Sprintf("%d", int(maxSSHCollectTTL/time.Second)+1)
	cases := map[string]string{
		"non-numeric":     "abc",
		"empty":           "",
		"negative":        "-5",
		"zero":            "0",
		"decimal":         "2.5",
		"oversized":       oversized,
		"max int seconds": fmt.Sprintf("%d", int(^uint(0)>>1)),
		"huge positive":   "9999999999999999999",
	}
	for name, val := range cases {
		t.Run(name, func(t *testing.T) {
			r := &stubConfigReader{values: map[string]string{sshCollectTimeoutKey: val}}
			got, err := resolveCollectTimeout(r)
			if err == nil {
				t.Fatalf("expected a bounded-fallback error for %q", val)
			}
			// Bounded: either the default or the ceiling, never the garbage value,
			// and never > the ceiling.
			if got <= 0 || got > maxSSHCollectTTL {
				t.Errorf("invalid value %s produced non-bounded %s", val, got)
			}
			// A dangerously large override must clamp to the ceiling, never wrap
			// negative (the pre-fix overflow) nor return an unbounded duration.
			if got != defaultSSHCollectTTL && got != maxSSHCollectTTL {
				t.Errorf("invalid value %s produced %s, want default or ceiling", val, got)
			}
		})
	}
}

// TestResolveCollectTimeoutOverflowClampsToCeiling pins the exact returned value
// for values whose Duration conversion would overflow (MaxInt64 seconds) — they
// must clamp to the 10-minute ceiling while still reporting the rejection.
func TestResolveCollectTimeoutOverflowClampsToCeiling(t *testing.T) {
	cases := map[string]string{
		"max int seconds": fmt.Sprintf("%d", int(^uint(0)>>1)),
		"huge positive":   "999999999999999", // parseable, far above the 600s bound
		"above max":       fmt.Sprintf("%d", maxSSHCollectSeconds+1),
	}
	for name, val := range cases {
		t.Run(name, func(t *testing.T) {
			r := &stubConfigReader{values: map[string]string{sshCollectTimeoutKey: val}}
			got, err := resolveCollectTimeout(r)
			if err == nil {
				t.Fatalf("expected a rejection error for %q", val)
			}
			if got != maxSSHCollectTTL {
				t.Errorf("expected ceiling %s, got %s", maxSSHCollectTTL, got)
			}
		})
	}
}

// TestResolveCollectTimeoutAtCeiling verifies the valid boundary (600s = the
// ceiling itself is accepted without error) and one second under it.
func TestResolveCollectTimeoutAtCeiling(t *testing.T) {
	for _, val := range []string{
		fmt.Sprintf("%d", maxSSHCollectSeconds-1),
		fmt.Sprintf("%d", maxSSHCollectSeconds),
	} {
		r := &stubConfigReader{values: map[string]string{sshCollectTimeoutKey: val}}
		got, err := resolveCollectTimeout(r)
		if err != nil {
			t.Fatalf("value %s: unexpected error %v", val, err)
		}
		if got <= 0 || got > maxSSHCollectTTL {
			t.Errorf("value %s produced non-bounded %s", val, got)
		}
	}
}

// TestConfigChangeAffectsCollectReapplied proves that the two values Collect
// actually consumes (collectTimeout() for the dial Timeout and
// effectiveSSHPort(host.SSHPort) for the dial port) are re-resolved from the
// injected reader on every call, so a runtime config change takes effect
// without a restart. It exercises the SetConfigReader injection seam cmdb owns;
// it does not claim production assembly (the recursive Base/opsinit gap this
// task tracks).
func TestConfigChangeAffectsCollectReapplied(t *testing.T) {
	r := &stubConfigReader{values: map[string]string{
		sshCollectTimeoutKey: "30",
		sshDefaultPortKey:    "2222",
	}}
	svc := NewHostService(nil)
	svc.SetConfigReader(r)

	if got := svc.collectTimeout(); got != 30*time.Second {
		t.Fatalf("begin: collect timeout %s, want 30s", got)
	}
	if got := svc.effectiveSSHPort(0); got != 2222 {
		t.Fatalf("begin: dial port %d, want config default 2222", got)
	}

	// Change the config value the reader serves; the next collect resolution
	// must observe it (no stale cache, no restart).
	r.values[sshCollectTimeoutKey] = "5"
	r.values[sshDefaultPortKey] = "8022"
	if got := svc.collectTimeout(); got != 5*time.Second {
		t.Errorf("collect timeout after change %s, want 5s", got)
	}
	if got := svc.effectiveSSHPort(0); got != 8022 {
		t.Errorf("dial port after change %d, want config default 8022", got)
	}
}

// TestCreatePersistsConfiguredDefaultPort proves the externalized
// cmdb.ssh.default_port is runtime-effective, not just an effectiveSSHPort(0)
// branch that never triggers: a host created with no explicit SSHPort is stored
// with the configured default (fallback 22), so the value reaches the persisted
// dial address. An explicit host port still wins.
func TestCreatePersistsConfiguredDefaultPort(t *testing.T) {
	db := setupTestDB(t)
	svc := NewHostService(db)
	svc.SetConfigReader(&stubConfigReader{values: map[string]string{sshDefaultPortKey: "2222"}})

	withDefault, err := svc.Create(CreateHostRequest{Hostname: "cfg-default", IP: "10.50.0.1", OS: "linux"}, "1")
	if err != nil {
		t.Fatalf("create host without explicit port: %v", err)
	}
	if withDefault.SSHPort != 2222 {
		t.Errorf("host without explicit port stored %d, want configured default 2222", withDefault.SSHPort)
	}

	explicit, err := svc.Create(CreateHostRequest{Hostname: "cfg-explicit", IP: "10.50.0.2", OS: "linux", SSHPort: 9999}, "1")
	if err != nil {
		t.Fatalf("create host with explicit port: %v", err)
	}
	if explicit.SSHPort != 9999 {
		t.Errorf("explicit host port stored %d, want 9999", explicit.SSHPort)
	}
}

// TestCreatePersistsFallbackDefaultPort mirrors TestCreatePersistsConfiguredDefaultPort
// with no reader injected, pinning the legacy fallback: omitted port still stores 22.
func TestCreatePersistsFallbackDefaultPort(t *testing.T) {
	db := setupTestDB(t)
	svc := NewHostService(db) // no reader -> defaultSSHPort() == 22

	created, err := svc.Create(CreateHostRequest{Hostname: "no-cfg", IP: "10.50.0.3", OS: "linux"}, "1")
	if err != nil {
		t.Fatalf("create host: %v", err)
	}
	if created.SSHPort != 22 {
		t.Errorf("host without reader stored %d, want fallback 22", created.SSHPort)
	}
}

func TestResolveDefaultPortDefaults(t *testing.T) {
	for name, reader := range map[string]ConfigReader{
		"nil reader":  nil,
		"missing key": &stubConfigReader{values: map[string]string{}, missingKeys: true},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := resolveDefaultPort(reader)
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if got != defaultSSHDefaultPort {
				t.Errorf("expected default %d, got %d", defaultSSHDefaultPort, got)
			}
		})
	}
}

func TestResolveDefaultPortFromConfig(t *testing.T) {
	r := &stubConfigReader{values: map[string]string{sshDefaultPortKey: "2222"}}
	got, err := resolveDefaultPort(r)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if got != 2222 {
		t.Errorf("expected 2222, got %d", got)
	}
	if len(r.calls) != 1 || r.calls[0] != sshDefaultPortKey {
		t.Errorf("expected single read of %s, got %v", sshDefaultPortKey, r.calls)
	}
}

func TestResolveDefaultPortBoundedFallback(t *testing.T) {
	cases := map[string]string{
		"too low":     "0",
		"negative":    "-1",
		"too high":    "70000",
		"non-numeric": "ssh",
		"empty":       "",
	}
	for name, val := range cases {
		t.Run(name, func(t *testing.T) {
			r := &stubConfigReader{values: map[string]string{sshDefaultPortKey: val}}
			got, err := resolveDefaultPort(r)
			if err == nil {
				t.Fatalf("expected a bounded-fallback error for %q", val)
			}
			if got != defaultSSHDefaultPort {
				t.Errorf("expected bounded fallback %d, got %d", defaultSSHDefaultPort, got)
			}
		})
	}
}

// TestEffectiveSSHPortHostPriority verifies an explicit host port always wins
// over the platform-configured default, and the default only applies when the
// host carries no port — the stage the default is meant for (the dial address).
func TestEffectiveSSHPortHostPriority(t *testing.T) {
	cases := []struct {
		name     string
		hostPort int
		reader   ConfigReader
		want     int
	}{
		{
			name:     "explicit host port wins over config default",
			hostPort: 2222,
			reader:   &stubConfigReader{values: map[string]string{sshDefaultPortKey: "2222"}},
			want:     2222,
		},
		{
			name:     "explicit host port wins over fallback default",
			hostPort: 2200,
			reader:   nil,
			want:     2200,
		},
		{
			name:     "no host port uses config default",
			hostPort: 0,
			reader:   &stubConfigReader{values: map[string]string{sshDefaultPortKey: "2222"}},
			want:     2222,
		},
		{
			name:     "no host port, no config falls back to 22",
			hostPort: 0,
			reader:   nil,
			want:     22,
		},
		{
			name:     "no host port, invalid config falls back to 22",
			hostPort: 0,
			reader:   &stubConfigReader{values: map[string]string{sshDefaultPortKey: "70000"}},
			want:     22,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := NewHostService(nil)
			svc.SetConfigReader(tc.reader)
			if got := svc.effectiveSSHPort(tc.hostPort); got != tc.want {
				t.Errorf("expected port %d, got %d", tc.want, got)
			}
		})
	}
}

// TestDifferentTenantContexts models two tenants whose scoped setting views
// resolve different config values; the injected, tenant-scoped reader must drive
// different results, proving config reads follow the host's tenant context
// rather than a shared global value.
func TestDifferentTenantContexts(t *testing.T) {
	tenantA := &stubConfigReader{values: map[string]string{
		sshCollectTimeoutKey: "10",
		sshDefaultPortKey:    "22",
	}}
	tenantB := &stubConfigReader{values: map[string]string{
		sshCollectTimeoutKey: "45",
		sshDefaultPortKey:    "8022",
	}}

	svcA := NewHostService(nil)
	svcA.SetConfigReader(tenantA)
	svcB := NewHostService(nil)
	svcB.SetConfigReader(tenantB)

	if got := svcA.collectTimeout(); got != 10*time.Second {
		t.Errorf("tenant A expected 10s, got %s", got)
	}
	if got := svcB.collectTimeout(); got != 45*time.Second {
		t.Errorf("tenant B expected 45s, got %s", got)
	}
	if a, b := svcA.defaultSSHPort(), svcB.defaultSSHPort(); a != 22 || b != 8022 {
		t.Errorf("tenant A %d / tenant B %d: tenant-scoped config must not cross", a, b)
	}
}

func TestCollectDialAddressPortSource(t *testing.T) {
	// The dial address for a host with no explicit port is built from the config
	// default (or 22); an explicit host port is used verbatim.
	svc := NewHostService(nil)
	svc.SetConfigReader(&stubConfigReader{values: map[string]string{sshDefaultPortKey: "2222"}})

	addr := fmt.Sprintf("%s:%d", "127.0.0.1", svc.effectiveSSHPort(0))
	if !strings.HasSuffix(addr, ":2222") {
		t.Errorf("expected default-port dial address, got %s", addr)
	}
	addr = fmt.Sprintf("%s:%d", "127.0.0.1", svc.effectiveSSHPort(2200))
	if !strings.HasSuffix(addr, ":2200") {
		t.Errorf("expected explicit-host-port dial address, got %s", addr)
	}
}
