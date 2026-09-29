package ssrf

import (
	"net"
	"testing"
	"time"
)

func TestDefaultValidator(t *testing.T) {
	v := DefaultValidator()

	if len(v.AllowedSchemes) != 2 {
		t.Errorf("Expected 2 allowed schemes, got %d", len(v.AllowedSchemes))
	}

	if !v.BlockPrivateIPs {
		t.Error("Expected BlockPrivateIPs to be true")
	}

	if !v.DNSRebindProtection {
		t.Error("Expected DNSRebindProtection to be true")
	}
}

func TestValidateURL_Scheme(t *testing.T) {
	v := DefaultValidator()

	tests := []struct {
		url     string
		wantErr bool
	}{
		{"http://example.com", false},
		{"https://example.com", false},
		{"ftp://example.com", true},
		{"file:///etc/passwd", true},
		{"javascript:alert(1)", true},
	}

	for _, tt := range tests {
		err := v.ValidateURL(tt.url)
		if (err != nil) != tt.wantErr {
			t.Errorf("ValidateURL(%q) error = %v, wantErr %v", tt.url, err, tt.wantErr)
		}
	}
}

func TestValidateURL_PrivateIP(t *testing.T) {
	v := DefaultValidator()
	v.AllowedDomains = []string{} // Allow all domains for IP testing

	tests := []struct {
		url     string
		wantErr bool
	}{
		{"http://127.0.0.1", true},       // Loopback
		{"http://localhost", true},       // Localhost
		{"http://10.0.0.1", true},        // Private
		{"http://172.16.0.1", true},      // Private
		{"http://192.168.1.1", true},     // Private
		{"http://169.254.169.254", true}, // AWS metadata
		{"http://8.8.8.8", false},        // Public (Google DNS)
		{"http://1.1.1.1", false},        // Public (Cloudflare DNS)
		{"http://example.com", false},    // Public domain
	}

	for _, tt := range tests {
		err := v.ValidateURL(tt.url)
		if (err != nil) != tt.wantErr {
			t.Errorf("ValidateURL(%q) error = %v, wantErr %v", tt.url, err, tt.wantErr)
		}
	}
}

func TestValidateURL_DomainAllowlist(t *testing.T) {
	v := DefaultValidator()
	v.AllowedDomains = []string{"example.com", "api.github.com"}
	v.BlockPrivateIPs = false // Disable for domain testing

	tests := []struct {
		url     string
		wantErr bool
	}{
		{"http://example.com", false},
		{"http://subdomain.example.com", false},
		{"http://api.github.com", false},
		{"http://evil.com", true},
		{"http://exampleXcom", true},
	}

	for _, tt := range tests {
		err := v.ValidateURL(tt.url)
		if (err != nil) != tt.wantErr {
			t.Errorf("ValidateURL(%q) error = %v, wantErr %v", tt.url, err, tt.wantErr)
		}
	}
}

func TestValidateURL_Port(t *testing.T) {
	v := DefaultValidator()
	v.AllowedPorts = []int{80, 443}
	v.BlockPrivateIPs = false
	v.AllowedDomains = []string{} // Allow all domains

	tests := []struct {
		url     string
		wantErr bool
	}{
		{"http://example.com:80", false},
		{"https://example.com:443", false},
		{"http://example.com:8080", true},
		{"http://example.com:22", true},
	}

	for _, tt := range tests {
		err := v.ValidateURL(tt.url)
		if (err != nil) != tt.wantErr {
			t.Errorf("ValidateURL(%q) error = %v, wantErr %v", tt.url, err, tt.wantErr)
		}
	}
}

func TestIsPrivateIP(t *testing.T) {
	tests := []struct {
		ip   string
		want bool
	}{
		{"127.0.0.1", true},
		{"10.0.0.1", true},
		{"172.16.0.1", true},
		{"192.168.1.1", true},
		{"169.254.169.254", true},
		{"8.8.8.8", false},
		{"1.1.1.1", false},
		{"::1", true},
		{"fe80::1", true},
		{"fc00::1", true},
		{"2001:4860:4860::8888", false},
	}

	for _, tt := range tests {
		ip := net.ParseIP(tt.ip)
		if ip == nil {
			t.Errorf("Failed to parse IP: %s", tt.ip)
			continue
		}
		got := isPrivateIP(ip)
		if got != tt.want {
			t.Errorf("isPrivateIP(%s) = %v, want %v", tt.ip, got, tt.want)
		}
	}
}

func TestStrictValidator(t *testing.T) {
	v := StrictValidator([]string{"github.com"})

	// Should only allow HTTPS
	if err := v.ValidateURL("http://github.com"); err == nil {
		t.Error("Expected HTTP to be blocked in strict mode")
	}

	// Should allow HTTPS to allowed domain
	if err := v.ValidateURL("https://github.com"); err != nil {
		t.Errorf("Expected HTTPS to allowed domain to pass: %v", err)
	}

	// Should block non-allowed domain (using private IP to avoid DNS lookup)
	if err := v.ValidateURL("https://192.168.1.1"); err == nil {
		t.Error("Expected non-allowed domain to be blocked")
	}
}

func TestCreateHTTPClient(t *testing.T) {
	v := DefaultValidator()
	v.MaxRedirects = 0

	client := v.CreateHTTPClient(v.Timeout)

	if client == nil {
		t.Fatal("Expected HTTP client to be created")
	}

	if client.Timeout != v.Timeout {
		t.Errorf("Expected timeout %v, got %v", v.Timeout, client.Timeout)
	}
}

func TestCreateHTTPClientTimeoutOverride(t *testing.T) {
	v := DefaultValidator()

	client := v.CreateHTTPClient(30 * time.Second)
	if client.Timeout != 30*time.Second {
		t.Errorf("Expected timeout 30s, got %v", client.Timeout)
	}
	if v.Timeout != 30*time.Second {
		t.Errorf("Expected validator timeout to be updated to 30s, got %v", v.Timeout)
	}

	v2 := DefaultValidator()
	original := v2.Timeout
	client2 := v2.CreateHTTPClient(0)
	if client2.Timeout != original {
		t.Errorf("Expected zero timeout to keep default %v, got %v", original, client2.Timeout)
	}
}

func BenchmarkValidateURL(b *testing.B) {
	v := DefaultValidator()
	v.AllowedDomains = []string{"example.com"}

	for i := 0; i < b.N; i++ {
		_ = v.ValidateURL("https://example.com/api/data")
	}
}

func BenchmarkIsPrivateIP(b *testing.B) {
	ip := net.ParseIP("192.168.1.1")

	for i := 0; i < b.N; i++ {
		_ = isPrivateIP(ip)
	}
}
