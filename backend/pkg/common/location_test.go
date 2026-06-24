package common

import "testing"

func TestGetLocationByIPLocalhost(t *testing.T) {
	loc := GetLocationByIP("127.0.0.1")
	if loc != "内网IP" {
		t.Fatalf("expected 内网IP for 127.0.0.1, got %s", loc)
	}
}

func TestGetLocationByIPIPv6Localhost(t *testing.T) {
	loc := GetLocationByIP("::1")
	if loc != "内网IP" {
		t.Fatalf("expected 内网IP for ::1, got %s", loc)
	}
}

func TestGetLocationByIPPrivateRange(t *testing.T) {
	loc := GetLocationByIP("192.168.1.100")
	if loc != "内网IP" {
		t.Fatalf("expected 内网IP for 192.168.x.x, got %s", loc)
	}
}

func TestGetLocationByIPPublicIP(t *testing.T) {
	loc := GetLocationByIP("8.8.8.8")
	if loc != "未知位置" {
		t.Fatalf("expected 未知位置 for public IP, got %s", loc)
	}
}

func TestGetLocationByIPEmptyString(t *testing.T) {
	loc := GetLocationByIP("")
	if loc != "未知位置" {
		t.Fatalf("expected 未知位置 for empty IP, got %s", loc)
	}
}

func TestInitLocationServiceDoesNotPanic(t *testing.T) {
	InitLocationService()
}
