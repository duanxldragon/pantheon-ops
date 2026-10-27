# Login Risk Control Design

**Version**: 1.0  
**Date**: 2026-09-08  
**Status**: Design Phase  
**Target**: Enhanced Authentication Security

---

## Executive Summary

This document defines a comprehensive login risk control system for Pantheon Base to detect and prevent unauthorized access through anomaly detection, device fingerprinting, and adaptive authentication.

**Key Goals**:
- Detect suspicious login attempts
- Implement device fingerprinting
- Adaptive MFA based on risk level
- Geographic anomaly detection
- Automated threat response

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                    Login Flow with Risk Control              │
└─────────────────────────────────────────────────────────────┘

User Login Attempt
     │
     ▼
┌────────────────────────┐
│  Device Fingerprinting │ ──> Store/Compare device info
└──────────┬─────────────┘
           │
           ▼
┌────────────────────────┐
│   Risk Assessment      │
│  ┌──────────────────┐ │
│  │ Location Check   │ │ ──> Compare to historical locations
│  │ Device Check     │ │ ──> Known device?
│  │ Time Pattern     │ │ ──> Unusual time?
│  │ Velocity Check   │ │ ──> Too many attempts?
│  │ IP Reputation    │ │ ──> Known bad IP?
│  └──────────────────┘ │
└──────────┬─────────────┘
           │
           ▼
┌────────────────────────┐
│   Risk Score (0-100)   │
└──────────┬─────────────┘
           │
           ├─> Score 0-30: Low Risk ──> Allow Login
           │
           ├─> Score 31-60: Medium Risk ──> Require MFA
           │
           ├─> Score 61-80: High Risk ──> MFA + Email Verification
           │
           └─> Score 81-100: Critical Risk ──> Block + Admin Alert
```

---

## Risk Factors & Scoring

### Risk Factor Categories

#### 1. Device Trust (0-25 points)

**New Device** (+25): Never seen before  
**Untrusted Device** (+15): Seen but not trusted  
**Trusted Device** (0): Marked as trusted by user  
**Fingerprint Mismatch** (+20): Device changed characteristics

#### 2. Geographic Anomaly (0-25 points)

**New Country** (+25): First login from this country  
**New City** (+15): First login from this city  
**Impossible Travel** (+25): Too fast between locations  
**High-Risk Country** (+10): Known for attacks  
**Normal Location** (0): Usual login location

#### 3. Behavioral Anomaly (0-20 points)

**Unusual Time** (+10): Login at abnormal hour  
**Unusual Frequency** (+10): Too many attempts  
**Password Reset Recent** (+5): Just reset password  
**Account Inactive Long** (+10): Haven't logged in for 90+ days

#### 4. IP Reputation (0-20 points)

**Known VPN/Proxy** (+10)  
**Tor Exit Node** (+20)  
**Datacenter IP** (+15)  
**Blacklisted IP** (+20)  
**Residential IP** (0)

#### 5. Failed Attempts (0-10 points)

**1-2 failures** (+2)  
**3-5 failures** (+5)  
**5+ failures** (+10)

---

## Risk Score Thresholds

```
┌─────────────────────────────────────────────────────────┐
│  Risk Level │ Score Range │ Action                      │
├─────────────┼─────────────┼─────────────────────────────┤
│  Low        │    0-30     │ Allow                       │
│  Medium     │   31-60     │ Require MFA                 │
│  High       │   61-80     │ MFA + Email Verification    │
│  Critical   │   81-100    │ Block + Admin Alert         │
└─────────────────────────────────────────────────────────┘
```

### Response Actions

**Low Risk (0-30)**:
- Allow login
- Log event
- Update device trust score

**Medium Risk (31-60)**:
- Require MFA (TOTP, SMS, Email)
- Log event with risk details
- Prompt user to trust device

**High Risk (61-80)**:
- Require MFA
- Send verification email
- Temporary account lock (5 min)
- Admin notification

**Critical Risk (81-100)**:
- Block login attempt
- Lock account (admin unlock required)
- Email user about suspicious activity
- Admin alert with details
- Log to security incident system

---

## Device Fingerprinting

### Client-Side Fingerprint Collection

**JavaScript Library**: FingerprintJS or custom implementation

**Collected Attributes**:
```javascript
{
  "userAgent": "Mozilla/5.0...",
  "screen": {
    "width": 1920,
    "height": 1080,
    "colorDepth": 24,
    "pixelRatio": 1
  },
  "timezone": "America/Los_Angeles",
  "language": "en-US",
  "platform": "MacIntel",
  "plugins": ["Plugin1", "Plugin2"],
  "canvas": "canvas_hash_value",
  "webgl": "webgl_renderer_info",
  "fonts": ["Arial", "Times New Roman"],
  "audio": "audio_context_hash"
}
```

**Fingerprint Hash**:
```javascript
// Generate stable hash from collected attributes
const fingerprintHash = sha256(JSON.stringify(sortedAttributes));
```

### Server-Side Storage

**Table**: `login_devices`

```sql
CREATE TABLE login_devices (
    id BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
    user_id BIGINT UNSIGNED NOT NULL,
    device_fingerprint VARCHAR(64) NOT NULL COMMENT 'SHA256 hash',
    device_name VARCHAR(255) NULL COMMENT 'User-friendly name',
    
    -- Device Info
    user_agent TEXT,
    screen_resolution VARCHAR(50),
    timezone VARCHAR(100),
    platform VARCHAR(100),
    
    -- Trust
    trust_status VARCHAR(20) DEFAULT 'new' COMMENT 'new, trusted, untrusted',
    trusted_at DATETIME NULL,
    
    -- Usage
    first_seen_at DATETIME NOT NULL,
    last_seen_at DATETIME NOT NULL,
    login_count INT DEFAULT 1,
    
    -- Location (last known)
    last_ip VARCHAR(45),
    last_country VARCHAR(100),
    last_city VARCHAR(255),
    
    -- Metadata
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    INDEX idx_login_devices_user_id (user_id),
    UNIQUE INDEX idx_login_devices_user_fingerprint (user_id, device_fingerprint),
    CONSTRAINT fk_login_devices_user FOREIGN KEY (user_id) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

---

## Geographic Tracking

### IP Geolocation

**Service**: MaxMind GeoLite2 or ipapi.co

**Collected Data**:
- Country
- City
- Latitude/Longitude
- ISP
- Connection type (datacenter, residential)

**Table**: `login_locations`

```sql
CREATE TABLE login_locations (
    id BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
    user_id BIGINT UNSIGNED NOT NULL,
    ip_address VARCHAR(45) NOT NULL,
    
    -- Geolocation
    country VARCHAR(100),
    city VARCHAR(255),
    latitude DECIMAL(10, 8),
    longitude DECIMAL(11, 8),
    
    -- Network
    isp VARCHAR(255),
    connection_type VARCHAR(50) COMMENT 'residential, datacenter, mobile',
    is_vpn BOOLEAN DEFAULT FALSE,
    is_proxy BOOLEAN DEFAULT FALSE,
    is_tor BOOLEAN DEFAULT FALSE,
    
    -- Timing
    logged_at DATETIME NOT NULL,
    
    INDEX idx_login_locations_user_id (user_id),
    INDEX idx_login_locations_logged_at (logged_at),
    CONSTRAINT fk_login_locations_user FOREIGN KEY (user_id) REFERENCES users(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

### Impossible Travel Detection

**Logic**:
```go
func detectImpossibleTravel(lastLogin, currentLogin *LoginLocation) bool {
    timeDiff := currentLogin.LoggedAt.Sub(lastLogin.LoggedAt).Hours()
    distance := haversineDistance(
        lastLogin.Latitude, lastLogin.Longitude,
        currentLogin.Latitude, currentLogin.Longitude,
    )
    
    // Average commercial flight speed: 800 km/h
    // Add buffer for ground transportation
    maxPossibleDistance := timeDiff * 1000 // km/h with buffer
    
    return distance > maxPossibleDistance
}

// Haversine formula for distance between two points on Earth
func haversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
    // Returns distance in kilometers
    // ... implementation
}
```

---

## Risk Assessment Engine

### Implementation

**File**: `backend/modules/auth/security/risk_engine.go`

```go
package security

type RiskEngine struct {
    db           *gorm.DB
    geolocator   GeoLocator
    ipChecker    IPReputationChecker
}

type RiskAssessment struct {
    Score       int
    Level       string // low, medium, high, critical
    Factors     []RiskFactor
    Action      string // allow, mfa, block
    Explanation string
}

type RiskFactor struct {
    Category string
    Points   int
    Reason   string
}

func (e *RiskEngine) AssessLoginRisk(ctx context.Context, req *LoginRequest) (*RiskAssessment, error) {
    score := 0
    factors := []RiskFactor{}
    
    // 1. Device Trust Check
    deviceScore, deviceFactors := e.assessDeviceTrust(ctx, req)
    score += deviceScore
    factors = append(factors, deviceFactors...)
    
    // 2. Geographic Anomaly
    geoScore, geoFactors := e.assessGeographicAnomaly(ctx, req)
    score += geoScore
    factors = append(factors, geoFactors...)
    
    // 3. Behavioral Anomaly
    behaviorScore, behaviorFactors := e.assessBehavioralAnomaly(ctx, req)
    score += behaviorScore
    factors = append(factors, behaviorFactors...)
    
    // 4. IP Reputation
    ipScore, ipFactors := e.assessIPReputation(ctx, req)
    score += ipScore
    factors = append(factors, ipFactors...)
    
    // 5. Failed Attempts
    failScore, failFactors := e.assessFailedAttempts(ctx, req)
    score += failScore
    factors = append(factors, failFactors...)
    
    // Determine level and action
    level, action := e.determineRiskLevel(score)
    
    return &RiskAssessment{
        Score:       score,
        Level:       level,
        Factors:     factors,
        Action:      action,
        Explanation: e.buildExplanation(factors),
    }, nil
}

func (e *RiskEngine) assessDeviceTrust(ctx context.Context, req *LoginRequest) (int, []RiskFactor) {
    var device LoginDevice
    err := e.db.Where("user_id = ? AND device_fingerprint = ?", 
        req.UserID, req.DeviceFingerprint).First(&device).Error
    
    if err == gorm.ErrRecordNotFound {
        // New device
        return 25, []RiskFactor{{
            Category: "device",
            Points:   25,
            Reason:   "New device never seen before",
        }}
    }
    
    if device.TrustStatus == "trusted" {
        return 0, []RiskFactor{}
    }
    
    if device.TrustStatus == "untrusted" {
        return 15, []RiskFactor{{
            Category: "device",
            Points:   15,
            Reason:   "Device marked as untrusted",
        }}
    }
    
    // Device seen but not explicitly trusted
    return 10, []RiskFactor{{
        Category: "device",
        Points:   10,
        Reason:   "Known but not trusted device",
    }}
}

func (e *RiskEngine) assessGeographicAnomaly(ctx context.Context, req *LoginRequest) (int, []RiskFactor) {
    // Get current location from IP
    currentLoc := e.geolocator.Lookup(req.IPAddress)
    
    // Get last login location
    var lastLoc LoginLocation
    err := e.db.Where("user_id = ?", req.UserID).
        Order("logged_at DESC").First(&lastLoc).Error
    
    if err == gorm.ErrRecordNotFound {
        // First login
        return 5, []RiskFactor{{
            Category: "location",
            Points:   5,
            Reason:   "First login from this account",
        }}
    }
    
    factors := []RiskFactor{}
    score := 0
    
    // Check for new country
    if currentLoc.Country != lastLoc.Country {
        score += 20
        factors = append(factors, RiskFactor{
            Category: "location",
            Points:   20,
            Reason:   fmt.Sprintf("New country: %s (was: %s)", currentLoc.Country, lastLoc.Country),
        })
    }
    
    // Check for impossible travel
    if e.detectImpossibleTravel(&lastLoc, currentLoc) {
        score += 25
        factors = append(factors, RiskFactor{
            Category: "location",
            Points:   25,
            Reason:   "Impossible travel detected",
        })
    }
    
    return score, factors
}

func (e *RiskEngine) determineRiskLevel(score int) (string, string) {
    switch {
    case score <= 30:
        return "low", "allow"
    case score <= 60:
        return "medium", "mfa"
    case score <= 80:
        return "high", "mfa_email"
    default:
        return "critical", "block"
    }
}
```

---

## Adaptive Authentication

### MFA Triggers

**Always Require MFA**:
- Admin roles
- Sensitive operations (password change, delete account)
- First login after password reset

**Risk-Based MFA**:
- Medium risk score (31-60)
- High risk score (61-80)
- New device
- New location

**No MFA Required**:
- Low risk score (0-30)
- Trusted device
- Normal location
- Normal time pattern

### MFA Methods

**Supported**:
1. **TOTP** (Time-based OTP) - Google Authenticator, Authy
2. **SMS** - Text message code
3. **Email** - Email verification code
4. **Backup Codes** - One-time recovery codes

**Priority Order**:
1. TOTP (if enabled)
2. SMS (if phone number on file)
3. Email (always available)

---

## Security Event Logging

### Table: `security_events`

```sql
CREATE TABLE security_events (
    id BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
    user_id BIGINT UNSIGNED NULL,
    event_type VARCHAR(50) NOT NULL,
    event_category VARCHAR(50) NOT NULL,
    
    -- Risk Assessment
    risk_score INT NULL,
    risk_level VARCHAR(20) NULL,
    risk_factors JSON NULL,
    
    -- Request Info
    ip_address VARCHAR(45),
    user_agent TEXT,
    device_fingerprint VARCHAR(64),
    
    -- Location
    country VARCHAR(100),
    city VARCHAR(255),
    
    -- Outcome
    action_taken VARCHAR(50) NOT NULL,
    success BOOLEAN,
    
    -- Metadata
    details JSON NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    
    INDEX idx_security_events_user_id (user_id),
    INDEX idx_security_events_type (event_type),
    INDEX idx_security_events_risk (risk_level),
    INDEX idx_security_events_created (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

**Event Types**:
- `login_attempt`
- `login_success`
- `login_failed`
- `login_blocked`
- `mfa_required`
- `mfa_success`
- `mfa_failed`
- `device_trusted`
- `password_reset`
- `account_locked`

---

## User Experience

### Device Trust Prompt

After successful login from new device:

```
✓ Login Successful

This is a new device. Do you want to trust this device?
Trusted devices won't require additional verification.

[ ] Trust this device

[Yes, Trust] [No, Don't Trust]
```

### Risk Alert Email

When high/critical risk detected:

```
Subject: Unusual login attempt detected

Hello [User],

We detected a login attempt to your account from an unusual location:

Location: Tokyo, Japan
IP: 123.45.67.89
Time: 2026-09-08 14:23 UTC
Device: Chrome on Windows

If this was you:
- No action needed
- Consider trusting this device

If this wasn't you:
- Change your password immediately
- Review account activity

[View Login History] [Change Password]
```

### Admin Alert Dashboard

Critical risk events shown in admin dashboard:

```
🚨 High-Risk Login Attempts (Last 24 Hours)

┌──────────┬───────────┬─────────┬──────────┬────────┐
│ User     │ Location  │ Risk    │ Action   │ Time   │
├──────────┼───────────┼─────────┼──────────┼────────┤
│ alice    │ Beijing   │ 85      │ Blocked  │ 2h ago │
│ bob      │ Moscow    │ 72      │ MFA Req  │ 5h ago │
│ charlie  │ Mumbai    │ 68      │ MFA Req  │ 8h ago │
└──────────┴───────────┴─────────┴──────────┴────────┘

[View Details] [Export Report]
```

---

## API Endpoints

```
POST   /api/v1/auth/login                    # Enhanced with risk assessment
POST   /api/v1/auth/mfa/verify               # MFA verification
POST   /api/v1/auth/devices                  # List user devices
PUT    /api/v1/auth/devices/:id/trust       # Trust a device
DELETE /api/v1/auth/devices/:id              # Remove device
GET    /api/v1/auth/security-events          # User security events
GET    /api/v1/admin/security-dashboard      # Admin security dashboard
GET    /api/v1/admin/high-risk-events        # High-risk events
```

---

## Configuration

### Environment Variables

```bash
# Risk Control
PANTHEON_RISK_CONTROL_ENABLED=true
PANTHEON_RISK_LOW_THRESHOLD=30
PANTHEON_RISK_MEDIUM_THRESHOLD=60
PANTHEON_RISK_HIGH_THRESHOLD=80

# Device Fingerprinting
PANTHEON_DEVICE_FINGERPRINT_ENABLED=true

# Geolocation
PANTHEON_GEOLOCATION_PROVIDER=maxmind
PANTHEON_GEOLOCATION_API_KEY=your-api-key

# IP Reputation
PANTHEON_IP_REPUTATION_ENABLED=true
PANTHEON_IP_REPUTATION_PROVIDER=ipqualityscore

# MFA
PANTHEON_MFA_RISK_BASED=true
PANTHEON_MFA_REQUIRED_FOR_ADMINS=true

# Alerts
PANTHEON_SECURITY_ALERT_EMAIL=security@example.com
PANTHEON_HIGH_RISK_ALERT_ENABLED=true
```

---

## Performance Considerations

### Caching

**Device Cache**:
- Cache known devices per user
- TTL: 1 hour
- Invalidate on trust status change

**IP Reputation Cache**:
- Cache IP reputation lookups
- TTL: 24 hours
- Shared across all users

**Geolocation Cache**:
- Cache IP geolocation
- TTL: 24 hours
- Shared across all users

### Database Optimization

**Indexes**:
```sql
-- Critical for performance
CREATE INDEX idx_login_devices_user_fingerprint ON login_devices(user_id, device_fingerprint);
CREATE INDEX idx_login_locations_user_recent ON login_locations(user_id, logged_at DESC);
CREATE INDEX idx_security_events_user_type ON security_events(user_id, event_type);
```

### Async Processing

**Background Jobs**:
- IP reputation check (async after login)
- Security event aggregation
- Risk score recalculation for historical data

---

## Testing Strategy

### Unit Tests

```go
func TestRiskEngine_AssessDeviceTrust_NewDevice(t *testing.T) {
    engine := setupTestRiskEngine(t)
    
    req := &LoginRequest{
        UserID:            1,
        DeviceFingerprint: "new-fingerprint-123",
    }
    
    score, factors := engine.assessDeviceTrust(context.Background(), req)
    
    assert.Equal(t, 25, score)
    assert.Len(t, factors, 1)
    assert.Equal(t, "New device never seen before", factors[0].Reason)
}

func TestRiskEngine_DetectImpossibleTravel(t *testing.T) {
    lastLogin := &LoginLocation{
        Latitude:  37.7749,  // San Francisco
        Longitude: -122.4194,
        LoggedAt:  time.Now().Add(-2 * time.Hour),
    }
    
    currentLogin := &LoginLocation{
        Latitude:  51.5074,  // London
        Longitude: -0.1278,
        LoggedAt:  time.Now(),
    }
    
    // SF to London in 2 hours is impossible
    assert.True(t, detectImpossibleTravel(lastLogin, currentLogin))
}
```

### Integration Tests

**Scenarios**:
1. Normal login from trusted device → Low risk
2. Login from new country → Medium risk, MFA required
3. Login from Tor → High risk, blocked
4. Multiple failed attempts → Account locked

---

## Rollout Plan

### Phase 1: Foundation (Current Design)
- Device fingerprinting
- Basic risk scoring
- Security event logging

**Timeline**: 12 hours implementation

### Phase 2: Enhanced Detection
- IP reputation integration
- Geographic anomaly detection
- Adaptive MFA

**Timeline**: 8 hours

### Phase 3: Advanced Features
- Machine learning risk model
- Behavioral biometrics
- Real-time threat intelligence

**Timeline**: 20 hours

---

## Success Criteria

- [ ] Device fingerprinting implemented
- [ ] Risk assessment engine working
- [ ] Security events logged
- [ ] Adaptive MFA functional
- [ ] Admin dashboard showing alerts
- [ ] User can trust devices
- [ ] Email alerts for high risk
- [ ] Performance acceptable (< 100ms overhead)

---

## Estimated Effort

- **Design**: 3 hours (✅ completed)
- **Implementation**: 12 hours (pending)
  - Device fingerprinting: 3h
  - Risk engine: 4h
  - Security events: 2h
  - API endpoints: 2h
  - Testing: 1h
- **Total**: 15 hours

---

**Document Status**: ✅ Design Complete  
**Next Phase**: Implementation (12 hours)  
**Dependencies**: P1-2 SSO/OIDC (for integrated risk assessment)
