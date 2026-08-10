package common

import "testing"

func TestGetLocationByIPLocalhost(t *testing.T) {
	loc := GetLocationByIP("127.0.0.1")
	if loc != "location.intranet" {
		t.Fatalf("expected location.intranet for 127.0.0.1, got %s", loc)
	}
}

func TestGetLocationByIPIPv6Localhost(t *testing.T) {
	loc := GetLocationByIP("::1")
	if loc != "location.intranet" {
		t.Fatalf("expected location.intranet for ::1, got %s", loc)
	}
}

func TestGetLocationByIPPrivateRange(t *testing.T) {
	loc := GetLocationByIP("192.168.1.100")
	if loc != "location.intranet" {
		t.Fatalf("expected location.intranet for 192.168.x.x, got %s", loc)
	}
}

func TestGetLocationByIPPublicIP(t *testing.T) {
	loc := GetLocationByIP("8.8.8.8")
	if loc != "location.unknown" {
		t.Fatalf("expected location.unknown for public IP, got %s", loc)
	}
}

func TestGetLocationByIPEmptyString(t *testing.T) {
	loc := GetLocationByIP("")
	if loc != "location.unknown" {
		t.Fatalf("expected location.unknown for empty IP, got %s", loc)
	}
}

func TestLocationDisplayText(t *testing.T) {
	if got := LocationDisplayText("location.intranet"); got != "内网 IP" {
		t.Fatalf("expected 内网 IP, got %s", got)
	}
	if got := LocationDisplayText("location.unknown"); got != "未知位置" {
		t.Fatalf("expected 未知位置, got %s", got)
	}
	// legacy / unknown values pass through unchanged
	if got := LocationDisplayText("北京市"); got != "北京市" {
		t.Fatalf("expected passthrough 北京市, got %s", got)
	}
}

func TestInitLocationServiceDoesNotPanic(t *testing.T) {
	InitLocationService()
}
