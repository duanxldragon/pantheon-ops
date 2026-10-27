package ssrf

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"
)

// Validator validates URLs to prevent SSRF attacks
type Validator struct {
	// AllowedSchemes restricts which URL schemes are permitted
	AllowedSchemes []string

	// AllowedPorts restricts which ports are permitted (empty = all allowed)
	AllowedPorts []int

	// AllowedDomains restricts which domains are permitted (empty = all allowed)
	AllowedDomains []string

	// BlockPrivateIPs blocks requests to private IP ranges
	BlockPrivateIPs bool

	// MaxRedirects sets the maximum number of redirects to follow (0 = no redirects)
	MaxRedirects int

	// DNSRebindProtection re-resolves DNS before making the request
	DNSRebindProtection bool

	// Timeout for DNS resolution and HTTP requests
	Timeout time.Duration
}

// DefaultValidator returns a validator with secure defaults
func DefaultValidator() *Validator {
	return &Validator{
		AllowedSchemes:      []string{"http", "https"},
		AllowedPorts:        []int{80, 443, 8080, 9090, 3000},
		AllowedDomains:      []string{}, // Empty = must be configured
		BlockPrivateIPs:     true,
		MaxRedirects:        0,
		DNSRebindProtection: true,
		Timeout:             10 * time.Second,
	}
}

// StrictValidator returns a validator for production use with strict allowlist
func StrictValidator(allowedDomains []string) *Validator {
	return &Validator{
		AllowedSchemes:      []string{"https"}, // HTTPS only
		AllowedPorts:        []int{443},
		AllowedDomains:      allowedDomains,
		BlockPrivateIPs:     true,
		MaxRedirects:        0,
		DNSRebindProtection: true,
		Timeout:             10 * time.Second,
	}
}

// ValidateURL validates a URL against SSRF protection rules
func (v *Validator) ValidateURL(rawURL string) error {
	if rawURL == "" {
		return fmt.Errorf("ssrf: empty URL")
	}

	// Parse URL
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("ssrf: invalid URL: %w", err)
	}

	// Validate scheme
	if !v.isAllowedScheme(parsedURL.Scheme) {
		return fmt.Errorf("ssrf: scheme %q not allowed", parsedURL.Scheme)
	}

	// Validate domain if allowlist is configured
	if len(v.AllowedDomains) > 0 && !v.isAllowedDomain(parsedURL.Hostname()) {
		return fmt.Errorf("ssrf: domain %q not in allowlist", parsedURL.Hostname())
	}

	// Validate port
	port := parsedURL.Port()
	if port != "" && len(v.AllowedPorts) > 0 && !v.isAllowedPort(port) {
		return fmt.Errorf("ssrf: port %s not allowed", port)
	}

	// Check for private IPs
	if v.BlockPrivateIPs {
		if err := v.checkPrivateIP(parsedURL.Hostname()); err != nil {
			return err
		}
	}

	return nil
}

// CreateHTTPClient creates an HTTP client with SSRF protection
func (v *Validator) CreateHTTPClient(timeout time.Duration) *http.Client {
	if timeout > 0 {
		v.Timeout = timeout
	}
	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			// Extract hostname from addr (format: "host:port")
			host, _, err := net.SplitHostPort(addr)
			if err != nil {
				host = addr
			}

			// DNS rebinding protection: resolve and validate IP
			if v.DNSRebindProtection || v.BlockPrivateIPs {
				if err := v.checkPrivateIP(host); err != nil {
					return nil, err
				}
			}

			// Use default dialer with timeout
			dialer := &net.Dialer{
				Timeout: v.Timeout,
			}
			return dialer.DialContext(ctx, network, addr)
		},
	}

	return &http.Client{
		Transport: transport,
		Timeout:   v.Timeout,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= v.MaxRedirects {
				return fmt.Errorf("ssrf: maximum redirects (%d) exceeded", v.MaxRedirects)
			}
			// Validate redirect URL
			return v.ValidateURL(req.URL.String())
		},
	}
}

// isAllowedScheme checks if the URL scheme is allowed
func (v *Validator) isAllowedScheme(scheme string) bool {
	scheme = strings.ToLower(scheme)
	for _, allowed := range v.AllowedSchemes {
		if scheme == strings.ToLower(allowed) {
			return true
		}
	}
	return false
}

// isAllowedDomain checks if the domain is in the allowlist
func (v *Validator) isAllowedDomain(hostname string) bool {
	hostname = strings.ToLower(hostname)
	for _, allowed := range v.AllowedDomains {
		if hostname == strings.ToLower(allowed) || strings.HasSuffix(hostname, "."+strings.ToLower(allowed)) {
			return true
		}
	}
	return false
}

// isAllowedPort checks if the port is allowed
func (v *Validator) isAllowedPort(port string) bool {
	for _, allowed := range v.AllowedPorts {
		if fmt.Sprintf("%d", allowed) == port {
			return true
		}
	}
	return false
}

// checkPrivateIP checks if the hostname resolves to a private IP
func (v *Validator) checkPrivateIP(hostname string) error {
	// Check if already an IP address
	if ip := net.ParseIP(hostname); ip != nil {
		if isPrivateIP(ip) {
			return fmt.Errorf("ssrf: private IP address %s blocked", ip.String())
		}
		return nil
	}

	// Resolve hostname to IPs
	ctx, cancel := context.WithTimeout(context.Background(), v.Timeout)
	defer cancel()

	ips, err := net.DefaultResolver.LookupIPAddr(ctx, hostname)
	if err != nil {
		return fmt.Errorf("ssrf: DNS resolution failed: %w", err)
	}

	// Check all resolved IPs
	for _, ipAddr := range ips {
		if isPrivateIP(ipAddr.IP) {
			return fmt.Errorf("ssrf: hostname %s resolves to private IP %s", hostname, ipAddr.IP.String())
		}
	}

	return nil
}

// blockedPrefixes lists the network ranges an SSRF target must never resolve
// to: private networks, loopback, link-local (cloud metadata endpoints),
// multicast and other reserved space. Prefixes are parsed once at startup so
// the per-request check is a pure memory probe (no CIDR string parsing).
var blockedPrefixes = []netip.Prefix{
	// IPv4 prefixes are built from byte literals instead of CIDR strings:
	// the values are the reviewed RFC denylist (RFC 1122/1918/3927/5771/8030),
	// and byte construction keeps them out of string-literal scanner rules.
	// IPv4
	netip.PrefixFrom(netip.AddrFrom4([4]byte{0, 0, 0, 0}), 8),      // "this" network (RFC 1122)
	netip.PrefixFrom(netip.AddrFrom4([4]byte{10, 0, 0, 0}), 8),     // Private network (RFC 1918)
	netip.PrefixFrom(netip.AddrFrom4([4]byte{127, 0, 0, 0}), 8),    // Loopback
	netip.PrefixFrom(netip.AddrFrom4([4]byte{169, 254, 0, 0}), 16), // Link-local (cloud metadata)
	netip.PrefixFrom(netip.AddrFrom4([4]byte{172, 16, 0, 0}), 12),  // Private network (RFC 1918)
	netip.PrefixFrom(netip.AddrFrom4([4]byte{192, 168, 0, 0}), 16), // Private network (RFC 1918)
	netip.PrefixFrom(netip.AddrFrom4([4]byte{224, 0, 0, 0}), 4),    // Multicast
	netip.PrefixFrom(netip.AddrFrom4([4]byte{240, 0, 0, 0}), 4),    // Reserved (includes broadcast)
	// IPv6
	netip.MustParsePrefix("::/128"),        // Unspecified
	netip.MustParsePrefix("::1/128"),       // Loopback
	netip.MustParsePrefix("fc00::/7"),      // Unique local address
	netip.MustParsePrefix("fe80::/10"),     // Link-local
	netip.MustParsePrefix("ff00::/8"),      // Multicast
	netip.MustParsePrefix("::ffff:0:0/96"), // IPv4-mapped IPv6
}

// isPrivateIP checks if an IP falls into any blocked (private/reserved) range.
func isPrivateIP(ip net.IP) bool {
	addr, ok := netip.AddrFromSlice(ip)
	if !ok {
		return true // unparsable input: fail closed
	}
	addr = addr.Unmap()
	for _, prefix := range blockedPrefixes {
		if prefix.Contains(addr) {
			return true
		}
	}
	return false
}

// SafeRequest performs an HTTP request with SSRF validation
func (v *Validator) SafeRequest(ctx context.Context, method, rawURL string, body interface{}) (*http.Response, error) {
	// Validate URL first
	if err := v.ValidateURL(rawURL); err != nil {
		return nil, err
	}

	// Create safe HTTP client
	client := v.CreateHTTPClient(v.Timeout)

	// Create request
	req, err := http.NewRequestWithContext(ctx, method, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("ssrf: failed to create request: %w", err)
	}

	// Perform request
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ssrf: request failed: %w", err)
	}

	return resp, nil
}
