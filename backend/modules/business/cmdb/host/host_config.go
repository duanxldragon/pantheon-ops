package host

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// SSH collection configuration externalized through the platform config
// capability (business/cmdb design §11.2). Keys are owned by business.cmdb and
// are read through a config reader injected into HostService — never by
// importing the system/config Service directly.
const (
	sshCollectTimeoutKey = "cmdb.ssh.collect_timeout"
	sshDefaultPortKey    = "cmdb.ssh.default_port"

	// defaultSSHCollectTTL is the documented default collect connection timeout
	// (10s), matching the pre-configuration hard-coded behavior.
	defaultSSHCollectTTL = 10 * time.Second
	// maxSSHCollectTTL bounds a mis-set override so a bad config can never yield
	// an unbounded (or runaway) collection wait. Values above the bound fall
	// back to the bound.
	maxSSHCollectTTL = 10 * time.Minute
	// maxSSHCollectSeconds is maxSSHCollectTTL in integer seconds (600); used to
	// bound a numeric override before Duration conversion so oversized values
	// clamp cleanly instead of overflowing.
	maxSSHCollectSeconds = int(maxSSHCollectTTL / time.Second)

	defaultSSHDefaultPort = 22
	sshPortMin            = 1
	sshPortMax            = 65535
)

// ConfigReader is the minimal config-read seam CMDB's SSH collection depends on.
// Its shape is identical to the public tenant.SettingFlagReader (GetByKey), so a
// *setting.SettingService view bound to a request tenant context, or any test
// stub, satisfies it. CMDB never reaches for the system/config Service directly;
// the tenant scoping of the injected view is what keeps a read consistent with
// the host's owning tenant.
type ConfigReader interface {
	GetByKey(key string) (string, error)
}

// resolveCollectTimeout returns the effective SSH collect connection timeout.
// A nil reader or an absent/missing key falls back to the documented 10s
// default; a non-numeric, non-positive, or oversized override is rejected with
// an error and a bounded fallback is returned so collection never waits on a
// garbage configuration.
func resolveCollectTimeout(reader ConfigReader) (time.Duration, error) {
	if reader == nil {
		return defaultSSHCollectTTL, nil
	}
	raw, err := reader.GetByKey(sshCollectTimeoutKey)
	if err != nil {
		// Key not seeded yet: the documented default is the intended behavior.
		return defaultSSHCollectTTL, nil
	}
	secs, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || secs <= 0 {
		return defaultSSHCollectTTL, fmt.Errorf("%s: invalid %q, falling back to default %ds", sshCollectTimeoutKey, raw, int(defaultSSHCollectTTL/time.Second))
	}
	// Bound in integer seconds BEFORE converting to a Duration: a numeric value
	// near MaxInt64 would otherwise make time.Duration(secs)*time.Second overflow
	// into a negative duration and slip past any post-conversion bound check.
	if secs > maxSSHCollectSeconds {
		return maxSSHCollectTTL, fmt.Errorf("%s: %ds exceeds bound %ds", sshCollectTimeoutKey, secs, maxSSHCollectSeconds)
	}
	return time.Duration(secs) * time.Second, nil
}

// resolveDefaultPort returns the effective default SSH port used only when a
// host carries no explicit port. A nil reader or missing key falls back to 22;
// an out-of-range or non-numeric override is rejected with an error and falls
// back to 22 (a bounded fallback — never an unbounded or invalid dial target).
func resolveDefaultPort(reader ConfigReader) (int, error) {
	if reader == nil {
		return defaultSSHDefaultPort, nil
	}
	raw, err := reader.GetByKey(sshDefaultPortKey)
	if err != nil {
		return defaultSSHDefaultPort, nil
	}
	p, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || p < sshPortMin || p > sshPortMax {
		return defaultSSHDefaultPort, fmt.Errorf("%s: invalid %q, falling back to default %d", sshDefaultPortKey, raw, defaultSSHDefaultPort)
	}
	return p, nil
}

// collectTimeout returns the service's effective SSH collect timeout (bounded
// fallback on invalid config).
func (s *HostService) collectTimeout() time.Duration {
	d, _ := resolveCollectTimeout(s.configReader)
	return d
}

// defaultSSHPort returns the service's effective default SSH port (bounded
// fallback on invalid config). It only applies when a host has no explicit port.
func (s *HostService) defaultSSHPort() int {
	p, _ := resolveDefaultPort(s.configReader)
	return p
}

// effectiveSSHPort resolves the port used for a collect dial: an explicit host
// port always wins; otherwise the platform-configured default (fallback 22) is
// used for this stage (the address this host will be dialed on).
func (s *HostService) effectiveSSHPort(hostPort int) int {
	if hostPort > 0 {
		return hostPort
	}
	return s.defaultSSHPort()
}
