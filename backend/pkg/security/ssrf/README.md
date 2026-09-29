# SSRF Protection Package

Package `ssrf` provides Server-Side Request Forgery (SSRF) protection for outbound HTTP requests.

## Features

- ✅ URL scheme validation (HTTP/HTTPS only)
- ✅ Private IP address blocking (RFC 1918, loopback, link-local)
- ✅ Domain allowlist enforcement
- ✅ Port restriction
- ✅ DNS rebinding protection
- ✅ Redirect following control
- ✅ IPv4 and IPv6 support
- ✅ Cloud metadata endpoint blocking (169.254.169.254)

## Usage

### Basic Usage

```go
import "github.com/duanxldragon/pantheon-base/backend/pkg/security/ssrf"

// Create validator with secure defaults
validator := ssrf.DefaultValidator()

// Validate URL before making request
if err := validator.ValidateURL("https://example.com/api"); err != nil {
    return err
}

// Create safe HTTP client
client := validator.CreateHTTPClient(validator.Timeout)
resp, err := client.Get("https://example.com/api")
```

### Strict Mode (Production)

```go
// Only allow HTTPS to specific domains
validator := ssrf.StrictValidator([]string{
    "api.example.com",
    "prometheus.internal.company.com",
})

if err := validator.ValidateURL(userProvidedURL); err != nil {
    return fmt.Errorf("URL not allowed: %w", err)
}
```

### Custom Configuration

```go
validator := &ssrf.Validator{
    AllowedSchemes:      []string{"http", "https"},
    AllowedPorts:        []int{80, 443, 9090},
    AllowedDomains:      []string{"api.example.com"},
    BlockPrivateIPs:     true,
    MaxRedirects:        0,
    DNSRebindProtection: true,
    Timeout:             10 * time.Second,
}

// Validate and make safe request
ctx := context.Background()
resp, err := validator.SafeRequest(ctx, "GET", userURL, nil)
```

## Blocked Targets

The validator blocks requests to:

### IPv4 Private Ranges
- `10.0.0.0/8` - Private network
- `172.16.0.0/12` - Private network
- `192.168.0.0/16` - Private network
- `127.0.0.0/8` - Loopback
- `169.254.0.0/16` - Link-local (AWS metadata service)
- `0.0.0.0/8` - Current network
- `224.0.0.0/4` - Multicast
- `240.0.0.0/4` - Reserved
- `255.255.255.255/32` - Broadcast

### IPv6 Private Ranges
- `::1/128` - Loopback
- `fe80::/10` - Link-local
- `fc00::/7` - Unique local address
- `ff00::/8` - Multicast
- `::/128` - Unspecified
- `::ffff:0:0/96` - IPv4-mapped IPv6

## Attack Vectors Prevented

1. **Private Network Scanning**: Blocks access to internal IP ranges
2. **Cloud Metadata Access**: Prevents access to 169.254.169.254 (AWS, GCP, Azure)
3. **Localhost Exploitation**: Blocks 127.0.0.1 and localhost
4. **DNS Rebinding**: Re-resolves DNS before request and validates IPs
5. **Redirect Chains**: Limits redirects and validates each hop
6. **Port Scanning**: Restricts to allowed ports only

## Integration Examples

### Observability Datasource Proxy

```go
func (s *DatasourceService) proxyQuery(datasourceURL, query string) (interface{}, error) {
    validator := ssrf.StrictValidator(s.getAllowedDatasourceDomains())
    
    if err := validator.ValidateURL(datasourceURL); err != nil {
        return nil, fmt.Errorf("datasource URL blocked: %w", err)
    }
    
    client := validator.CreateHTTPClient(validator.Timeout)
    // Make request...
}
```

### Notification Webhooks

```go
func (s *NotificationService) sendWebhook(webhookURL string, payload interface{}) error {
    validator := ssrf.StrictValidator(s.getAllowedWebhookDomains())
    
    if err := validator.ValidateURL(webhookURL); err != nil {
        return fmt.Errorf("webhook URL blocked: %w", err)
    }
    
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()
    
    resp, err := validator.SafeRequest(ctx, "POST", webhookURL, payload)
    // Handle response...
}
```

## Configuration

Set allowed domains via environment variables or config:

```yaml
# config.yaml
security:
  ssrf:
    datasources:
      allowed_domains:
        - prometheus.example.com
        - loki.example.com
        - tempo.example.com
    webhooks:
      allowed_domains:
        - hooks.slack.com
        - oapi.dingtalk.com
        - open.feishu.cn
```

## Testing

Run tests:
```bash
go test ./pkg/security/ssrf/...
go test -race ./pkg/security/ssrf/...
go test -bench=. ./pkg/security/ssrf/...
```

## Performance

- URL validation: ~5-10μs per URL
- Private IP check: ~1-2μs per IP
- DNS resolution: ~1-5ms (cached by system resolver)

## Security Notes

- Always use `StrictValidator` in production with explicit allowlists
- Review and update allowed domains regularly
- Monitor blocked requests for potential attacks
- Log all SSRF validation failures with request context

## Related

- [OWASP SSRF Prevention Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Server_Side_Request_Forgery_Prevention_Cheat_Sheet.html)
- [CWE-918: Server-Side Request Forgery](https://cwe.mitre.org/data/definitions/918.html)
