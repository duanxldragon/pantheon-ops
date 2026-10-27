# D2: SSRF and Outbound Network Policy Analysis

**Task**: 2026-09-28-ops-delivery-readiness-gap-closure  
**Sub-task**: D2  
**Executed**: 2026-09-28  
**Status**: ⚠️ BLOCKED - Requires maintainer policy decision

## Executive Summary

**Finding**: pantheon-ops contains **5 distinct outbound network request surfaces** with **NO SSRF protection**:

1. **Observability Datasource Proxy** (Prometheus/Loki/Tempo queries)
2. **Notification Webhooks** (arbitrary HTTP/webhook URLs)
3. **ArgoCD API Client** (external ArgoCD servers)
4. **Kubernetes API** (external K8s clusters)
5. **Email SMTP** (external mail servers)

**Risk Level**: 🔴 **HIGH** - All accept user-controlled URLs with no validation for:
- Private IP addresses (127.0.0.1, 10.0.0.0/8, 192.168.0.0/16, 169.254.0.0/16)
- DNS rebinding attacks
- Cloud metadata endpoints (169.254.169.254)
- Redirect following to private networks
- Localhost/loopback addresses

**Production Impact**: These services MUST remain disabled or restricted to pre-approved endpoints until a SSRF policy is implemented.

---

## Network Request Inventory

### 1. Observability Datasource Proxy (CRITICAL)

**Location**: `backend/modules/business/observability/datasource_service.go`

**Risk**: User can create datasources with arbitrary URLs and proxy queries through the platform.

**Code Analysis**:
```go
// Line 42-89: Create datasource with user-provided URL
func (s *DatasourceService) Create(dto DatasourceDTO, ...) {
    record := &PrometheusDatasource{
        URL:        dto.URL,  // ⚠️ No validation
        AccessMode: dto.AccessMode,
        ...
    }
    // Test connection in background
    go s.testConnectionAsync(record.ID, scope)
}

// Line 339-379: Proxy queries to user-controlled URL
func (s *DatasourceService) proxyJSON(record *PrometheusDatasource, ...) {
    baseURL, err := url.Parse(record.URL)  // ⚠️ No SSRF check
    if err != nil {
        return nil, err
    }
    baseURL.Path = path.Join(baseURL.Path, requestPath)
    req, err := http.NewRequestWithContext(ctx, http.MethodGet, baseURL.String(), nil)
    ...
    resp, err := (&http.Client{Timeout: ...}).Do(req)  // ⚠️ Direct request
}

// Line 396-420: Test connection to user-controlled URL
func testDatasourceConnection(datasourceType, rawURL, ...) (bool, error) {
    baseURL, err := url.Parse(rawURL)  // ⚠️ No validation
    req, err := http.NewRequestWithContext(ctx, http.MethodGet, testURL, nil)
    client := &http.Client{Timeout: time.Duration(timeout) * time.Second}
    resp, err := client.Do(req)  // ⚠️ No redirect control
}
```

**Attack Vectors**:
1. **Private Network Scanning**: Create datasource with URL `http://192.168.1.1:9090` to scan internal network
2. **Cloud Metadata Access**: URL `http://169.254.169.254/latest/meta-data/` to steal cloud credentials
3. **Localhost Services**: URL `http://127.0.0.1:6379` to access Redis/databases
4. **DNS Rebinding**: URL resolves to public IP initially, then rebinds to private IP
5. **Redirect Chains**: Initial URL redirects to private endpoint

**Affected Endpoints**:
- `POST /api/v1/business/observability/datasources` (create)
- `PUT /api/v1/business/observability/datasources/:id` (update)
- `GET /api/v1/business/observability/datasources/:id/test` (test connection)
- `GET /api/v1/business/observability/datasources/:id/query` (proxy queries)
- `GET /api/v1/business/observability/datasources/:id/query-logs` (Loki proxy)
- `GET /api/v1/business/observability/datasources/:id/search-traces` (Tempo proxy)
- `GET /api/v1/business/observability/datasources/:id/traces/:traceId` (trace lookup)

### 2. Notification Webhooks (HIGH)

**Location**: `backend/modules/business/observability/notification_service.go`

**Risk**: Alert rules can trigger webhooks to arbitrary URLs.

**Code Analysis**:
```go
// Line 217-238: Send webhook to user-provided URL
func (s *NotificationService) sendWebhook(config map[string]interface{}, ...) error {
    url, _ := config["url"].(string)  // ⚠️ No validation
    method, _ := config["method"].(string)
    
    if url == "" {
        return fmt.Errorf("missing webhook url")
    }
    
    req, err := http.NewRequest(method, url, bytes.NewBuffer(jsonData))
    client := &http.Client{Timeout: time.Duration(timeout) * time.Second}
    resp, err := client.Do(req)  // ⚠️ No SSRF check
}

// Line 240-268: Slack webhook (calls sendWebhook)
func (s *NotificationService) sendSlack(config, ...) error {
    webhookURL, _ := config["webhookUrl"].(string)  // ⚠️ No validation
    return s.sendWebhook(map[string]interface{}{"url": webhookURL, ...})
}

// Line 270-290: DingTalk webhook
func (s *NotificationService) sendDingTalk(config, ...) error {
    webhookURL, _ := config["webhookUrl"].(string)  // ⚠️ No validation
    // Adds signature but still no SSRF check
}

// Line 292-312: Feishu webhook
func (s *NotificationService) sendFeishu(config, ...) error {
    webhookURL, _ := config["webhookUrl"].(string)  // ⚠️ No validation
}
```

**Attack Vectors**:
1. Create notification channel with webhook URL pointing to internal service
2. Create alert rule that fires on controllable condition
3. Trigger alert to send webhook to internal endpoint
4. Extract response timing/errors to enumerate services

**Affected Endpoints**:
- `POST /api/v1/business/observability/channels` (create webhook channel)
- `PUT /api/v1/business/observability/channels/:id` (update webhook URL)
- Alert evaluation background job (automatic webhook triggers)

### 3. ArgoCD API Client (MEDIUM)

**Location**: `backend/modules/business/cicd/argocd/argocd_client.go`

**Risk**: ArgoCD integration points can be configured to arbitrary endpoints.

**Code Analysis**:
```go
// Line 22-36: Create ArgoCD client with user-provided endpoint
func NewArgoCDClient(endpoint, authToken string, insecureSkipVerify bool) *ArgoCDClient {
    transport := &http.Transport{
        TLSClientConfig: &tls.Config{
            InsecureSkipVerify: insecureSkipVerify,  // ⚠️ Can disable TLS
        },
    }
    return &ArgoCDClient{
        endpoint:   endpoint,  // ⚠️ No validation
        httpClient: &http.Client{Timeout: 30 * time.Second, Transport: transport},
    }
}

// Line 39-58: Test connection
func (c *ArgoCDClient) TestConnection(ctx context.Context) error {
    req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint+"/api/version", nil)
    resp, err := c.httpClient.Do(req)  // ⚠️ No SSRF check
}
```

**Attack Vectors**:
1. Configure ArgoCD endpoint to internal service
2. Use insecureSkipVerify to bypass TLS validation
3. Enumerate internal services via version endpoint

**Note**: ArgoCD clients are typically long-lived and require authentication tokens, reducing attack surface compared to datasources.

### 4. Kubernetes API (MEDIUM)

**Location**: `backend/modules/business/k8s/cluster/cluster_service.go`

**Risk**: K8s cluster configurations contain API server URLs and kubeconfig.

**Code Analysis**:
```go
// Line 180-200: GetClientset builds K8s client from cluster record
func (s *ClusterService) GetClientset(id uint64, dataScope *common.DataScopeReq) (*kubernetes.Clientset, error) {
    // Fetches cluster record with kubeconfig
    // kubeconfig contains API server URL
    // ⚠️ No validation of API server endpoint
}
```

**Attack Vectors**:
1. Create cluster record with kubeconfig pointing to internal service
2. Use K8s API calls to probe internal endpoints
3. Exploit K8s admission webhooks pointing to internal services

**Mitigation**: K8s API requires valid kubeconfig with certificates, significantly raising attack bar.

### 5. Email SMTP (LOW)

**Location**: `backend/modules/business/observability/notification_service.go`

**Risk**: SMTP configuration can point to arbitrary mail servers.

**Code Analysis**:
```go
// Line 184-215: Send email via SMTP
func (s *NotificationService) sendEmail(config, ...) error {
    smtpHost, _ := config["smtpHost"].(string)  // ⚠️ No validation
    smtpPort, _ := config["smtpPort"].(float64)
    addr := fmt.Sprintf("%s:%d", smtpHost, int(smtpPort))
    return smtp.SendMail(addr, auth, fromAddr, to, msg)  // ⚠️ Direct connection
}
```

**Attack Vectors**:
1. Configure SMTP to internal mail relay
2. Probe internal services on port 25/587/465
3. Use email as exfiltration channel

**Mitigation**: SMTP requires authentication and is typically used for outbound notifications only.

---

## Current State: No SSRF Protection

**Verification Commands**:
```bash
# Search for SSRF protection code
$ grep -r "privateIPCheck\|allowPrivate\|isPrivateIP\|SSRF" backend/modules/business --include="*.go"
(no results)

# Search for URL validation
$ grep -r "URL.*valid\|parseURL\|sanitize.*url" backend/modules/business --include="*.go" -i
(no results)
```

**Findings**:
- ✅ No existing SSRF protection middleware
- ✅ No IP address validation before HTTP requests
- ✅ No DNS rebinding protection
- ✅ No redirect following limits
- ✅ No private network range checks
- ✅ No cloud metadata endpoint blocks

---

## Required SSRF Policy Components

### 1. URL Validation Layer (Base-first)

**Requirement**: Shared middleware in `pantheon-base/backend/pkg/security/ssrf/`

**Must implement**:
```go
type SSRFValidator struct {
    AllowedSchemes   []string          // ["http", "https"]
    AllowedPorts     []int             // [80, 443, 9090, 3000]
    BlockedIPRanges  []*net.IPNet      // Private IPs, loopback, link-local
    AllowedDomains   []string          // Explicit whitelist
    MaxRedirects     int               // 0 = no redirects
    DNSRebindTimeout time.Duration     // Re-resolve before request
}

func (v *SSRFValidator) ValidateURL(rawURL string) error
func (v *SSRFValidator) CreateHTTPClient() *http.Client
```

**Blocked IP Ranges**:
- `127.0.0.0/8` (loopback)
- `10.0.0.0/8` (private)
- `172.16.0.0/12` (private)
- `192.168.0.0/16` (private)
- `169.254.0.0/16` (link-local, AWS metadata)
- `fe80::/10` (IPv6 link-local)
- `::1/128` (IPv6 loopback)
- `fc00::/7` (IPv6 unique local)

**DNS Rebinding Protection**:
1. Resolve URL to IP before request
2. Validate IP against blocked ranges
3. Re-resolve immediately before HTTP request
4. Compare resolved IPs
5. Fail if IP changed or new IP is blocked

### 2. Per-Service Configuration

**Datasource Proxy Policy**:
```yaml
# Ops config or environment
OBSERVABILITY_DATASOURCE_ALLOWED_DOMAINS: "prometheus.example.com,loki.example.com"
OBSERVABILITY_DATASOURCE_ALLOW_PRIVATE: "false"
OBSERVABILITY_DATASOURCE_MAX_REDIRECTS: "0"
```

**Webhook Policy**:
```yaml
NOTIFICATION_WEBHOOK_ALLOWED_DOMAINS: "hooks.slack.com,oapi.dingtalk.com"
NOTIFICATION_WEBHOOK_ALLOW_PRIVATE: "false"
```

**ArgoCD Policy**:
```yaml
ARGOCD_ALLOWED_ENDPOINTS: "argocd.example.com:443"
ARGOCD_ALLOW_INSECURE_TLS: "false"
```

### 3. Audit and Monitoring

**Required Audit Fields**:
- `outbound_request_url` (sanitized, no query params with secrets)
- `outbound_request_ip` (resolved IP)
- `outbound_request_duration_ms`
- `outbound_request_status`
- `ssrf_validation_result` ("allowed" / "blocked_private_ip" / "blocked_scheme" / etc)

**Metrics**:
- `pantheon_outbound_requests_total{service, outcome}`
- `pantheon_ssrf_blocks_total{service, reason}`

### 4. Error Semantics

**New Error Keys**:
- `observability.datasource_url_invalid` - Malformed URL
- `observability.datasource_url_private_ip` - Private IP blocked
- `observability.datasource_url_blocked_scheme` - Non-HTTP/HTTPS
- `observability.datasource_url_blocked_port` - Non-allowed port
- `observability.datasource_url_not_in_allowlist` - Domain not approved
- `observability.datasource_dns_rebind_detected` - DNS rebinding attempt

---

## Recommended Policy Decision

### Option A: Strict Allowlist (RECOMMENDED for production)

**Policy**:
- Datasources: ONLY pre-approved domains in config
- Webhooks: ONLY pre-approved webhook providers (Slack, DingTalk, Feishu official domains)
- ArgoCD: ONLY pre-approved ArgoCD instances
- K8s: Keep current behavior (kubeconfig validation is sufficient)
- Email: ONLY pre-approved SMTP relays

**Pros**:
- ✅ Maximum security
- ✅ Explicit approval for each external service
- ✅ Easy to audit and monitor

**Cons**:
- ⚠️ Requires maintainer to pre-approve every datasource
- ⚠️ Less flexible for dynamic environments

**Implementation Effort**: 2-3 days (1 day Base middleware + 1 day Ops integration + 1 day testing)

### Option B: Block Private + DNS Rebind Protection

**Policy**:
- Block all private IP ranges
- Block cloud metadata endpoints
- DNS rebinding protection
- Allow any public IP with validation
- Max 0 redirects

**Pros**:
- ✅ Good security baseline
- ✅ More flexible than allowlist
- ✅ Covers most SSRF attacks

**Cons**:
- ⚠️ Cannot access internal Prometheus/ArgoCD (requires VPN or proxy)
- ⚠️ May need exceptions for hybrid cloud

**Implementation Effort**: 3-4 days (2 days Base middleware + 1 day Ops integration + 1 day testing)

### Option C: Configurable Per-Environment

**Policy**:
- Dev/staging: Block private IPs but allow with explicit flag
- Production: Strict allowlist only
- Audit all requests regardless of environment

**Pros**:
- ✅ Flexibility for development
- ✅ Security for production
- ✅ Clear environment boundaries

**Cons**:
- ⚠️ More complex configuration
- ⚠️ Risk of misconfiguration

**Implementation Effort**: 4-5 days (2 days Base middleware + 2 days Ops multi-env config + 1 day testing)

---

## Base-First Implementation Plan

### Phase 1: Base Middleware (pantheon-base)

**Location**: `backend/pkg/security/ssrf/`

**Deliverables**:
1. `validator.go` - Core SSRF validation logic
2. `client.go` - HTTP client wrapper with validation
3. `middleware.go` - Gin middleware for request validation (if needed)
4. `validator_test.go` - Unit tests for all attack vectors

**Tests Must Cover**:
- ✅ Private IP blocking (127.0.0.1, 10.x, 192.168.x, 169.254.x)
- ✅ DNS rebinding detection
- ✅ Redirect following limits
- ✅ Scheme validation (http/https only)
- ✅ Port validation
- ✅ Domain allowlist enforcement
- ✅ IPv6 private ranges
- ✅ Cloud metadata endpoints (169.254.169.254, metadata.google.internal)

**Estimated**: 2 engineer-days

### Phase 2: Ops Consumer Integration

**Changes Required**:
1. `observability/datasource_service.go`:
   - Wrap `proxyJSON()` with SSRF validator
   - Wrap `testDatasourceConnection()` with validator
   - Add config for allowed domains

2. `observability/notification_service.go`:
   - Wrap `sendWebhook()` with validator
   - Add config for allowed webhook domains

3. `cicd/argocd/argocd_client.go`:
   - Validate endpoint in `NewArgoCDClient()`
   - Add config for allowed ArgoCD endpoints

4. Configuration:
   - Add environment variables or config file for allowlists
   - Document policy in `DESIGN.md`

**Estimated**: 1-2 engineer-days

### Phase 3: Testing and Evidence

**Test Coverage**:
1. Unit tests for validator (in Base)
2. Integration tests for each service:
   - Datasource creation with private IP (should fail)
   - Webhook with localhost (should fail)
   - ArgoCD with 169.254.169.254 (should fail)
   - Valid public endpoint (should succeed)

3. Manual verification:
   - Deploy to test environment
   - Attempt SSRF attacks
   - Verify audit logs

**Evidence**:
- Unit test results
- Integration test results
- Attack attempt logs showing blocks
- Successful legitimate request logs

**Estimated**: 1 engineer-day

---

## Blocking Conditions for D2 Completion

🔴 **BLOCKED** - Cannot complete D2 without:

1. **Maintainer Policy Decision** (choose Option A, B, or C)
2. **Approved Datasource Domains** (if Option A chosen)
3. **Approved Webhook Domains** (if Option A chosen)
4. **Base Task Assignment** (who will implement SSRF middleware in pantheon-base)
5. **Target Base Release** (which base version will include SSRF protection)

---

## Interim Risk Mitigation

**Until SSRF policy is implemented**:

### 1. Disable Datasource Proxy (CRITICAL)
```go
// In observability/datasource_service.go
func (s *DatasourceService) Create(...) (*DatasourceResponse, error) {
    return nil, errors.New("observability.datasource_proxy_disabled_pending_security_review")
}
```

### 2. Restrict Notification Channels to Admins Only
- Move notification channel CRUD to platform admin role
- Document: "Webhook notifications require admin approval"

### 3. Document SSRF Risk in SECURITY.md
```markdown
## Known Limitations

**Outbound Network Requests**: The following features accept user-provided URLs 
without SSRF protection and should only be used with pre-approved, trusted endpoints:
- Observability datasource proxy
- Notification webhooks
- ArgoCD integration

Do not enable these features in production without implementing SSRF protection 
per the Base roadmap.
```

### 4. Add Monitoring
- Alert on datasource creation events
- Alert on webhook channel creation
- Manual review queue for all outbound endpoint configurations

---

## Next Actions

1. **Maintainer**: Choose SSRF policy (Option A/B/C)
2. **Maintainer**: Assign Base SSRF middleware task
3. **Base Team**: Implement and test SSRF validator
4. **Base Team**: Release foundation with SSRF protection
5. **Ops**: Consume new Base release
6. **Ops**: Integrate SSRF validator into observability/cicd modules
7. **Ops**: Run integration tests and document policy
8. **QA**: Verify SSRF attacks are blocked
9. **D2**: Close with evidence of successful blocking

---

## Conclusion

**D2 Status**: ⚠️ **BLOCKED ON MAINTAINER DECISION**

pantheon-ops has 5 outbound network request surfaces with no SSRF protection. This is a **production blocker** for datasource proxy and webhook features.

**Recommended Path Forward**:
1. Implement Option A (strict allowlist) in pantheon-base
2. Disable datasource proxy until SSRF protection is deployed
3. Restrict webhook channels to admin-only
4. Document risk and mitigation plan
5. Resume D2 after Base SSRF middleware is available

**Estimated Time to Complete**: 3-5 engineer-days after policy decision, spread across Base implementation (2-3 days) and Ops integration (1-2 days).

**Cannot proceed to production without SSRF policy implementation.**
