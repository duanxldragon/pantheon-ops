# Multi-Tenant Architecture Design for Pantheon Base

**Version**: 1.1  
**Date**: 2026-09-11  
**Status**: Superseded by contract V1 — see [TENANT_CONTRACT_V1](../contracts/TENANT_CONTRACT_V1.md) (frozen) and [TENANT_MIGRATION_RUNBOOK](../runbooks/TENANT_MIGRATION_RUNBOOK.md). This document remains as background material for Phase 2/3 deployment models; where it conflicts with the contract, the contract wins.  
**Target**: Enterprise Multi-Organization Support

---

## Executive Summary

This document defines the multi-tenant architecture for Pantheon Base, enabling a single deployment to serve multiple organizations (tenants) with complete data isolation, customization, and independent administration.

**Key Goals**:
- Complete data isolation between tenants
- Tenant-specific customization (branding, settings)
- Scalable tenant onboarding
- Flexible pricing/billing support
- Minimal performance overhead

**Deployment Models**:
- **Shared Database, Shared Schema** (with tenant_id column) - Phase 1
- **Shared Database, Separate Schemas** - Phase 2
- **Separate Databases** - Phase 3

---

## Architecture Overview

### Tenant Isolation Models

```
┌─────────────────────────────────────────────────────────┐
│              Model 1: Shared Schema (Phase 1)           │
│  ┌────────────────────────────────────────────────────┐ │
│  │         MySQL Database: pantheon                   │ │
│  │  ┌──────────────────────────────────────────────┐ │ │
│  │  │  Table: users                                 │ │ │
│  │  │  ┌────────┬──────────┬───────────┬─────────┐ │ │ │
│  │  │  │ id     │ username │ tenant_id │ email   │ │ │ │
│  │  │  ├────────┼──────────┼───────────┼─────────┤ │ │ │
│  │  │  │ 1      │ admin    │ tenant-a  │ a@...   │ │ │ │
│  │  │  │ 2      │ admin    │ tenant-b  │ b@...   │ │ │ │
│  │  │  └────────┴──────────┴───────────┴─────────┘ │ │ │
│  │  └──────────────────────────────────────────────┘ │ │
│  └────────────────────────────────────────────────────┘ │
│  • Data isolated by tenant_id column                    │
│  • Single connection pool                               │
│  • Row-level security                                   │
└─────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────┐
│         Model 2: Separate Schemas (Phase 2)             │
│  ┌────────────────────────────────────────────────────┐ │
│  │         MySQL Database: pantheon                   │ │
│  │  ┌──────────────┐      ┌──────────────┐          │ │
│  │  │ Schema:      │      │ Schema:      │          │ │
│  │  │ tenant_a     │      │ tenant_b     │          │ │
│  │  │  - users     │      │  - users     │          │ │
│  │  │  - roles     │      │  - roles     │          │ │
│  │  └──────────────┘      └──────────────┘          │ │
│  └────────────────────────────────────────────────────┘ │
│  • Schema-level isolation                               │
│  • Connection pool per tenant                           │
│  • Logical separation                                   │
└─────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────┐
│         Model 3: Separate Databases (Phase 3)           │
│  ┌──────────────┐      ┌──────────────┐               │
│  │  Database:   │      │  Database:   │               │
│  │  tenant_a    │      │  tenant_b    │               │
│  │   - users    │      │   - users    │               │
│  │   - roles    │      │   - roles    │               │
│  └──────────────┘      └──────────────┘               │
│  • Database-level isolation                             │
│  • Physical separation                                  │
│  • Maximum security                                     │
└─────────────────────────────────────────────────────────┘
```

---

## Phase 1: Shared Schema Design (MVP)

### Database Schema Changes

#### 1. Tenant Master Table

```sql
CREATE TABLE tenants (
    id VARCHAR(64) PRIMARY KEY COMMENT 'Tenant unique identifier (e.g., tenant-uuid)',
    name VARCHAR(255) NOT NULL COMMENT 'Tenant display name',
    slug VARCHAR(100) UNIQUE NOT NULL COMMENT 'URL-safe identifier',
    status VARCHAR(20) DEFAULT 'active' COMMENT 'active, suspended, deleted',
    
    -- Subscription
    plan_type VARCHAR(50) DEFAULT 'free' COMMENT 'free, basic, pro, enterprise',
    max_users INT DEFAULT 10 COMMENT 'Maximum user count',
    expires_at DATETIME NULL COMMENT 'Subscription expiry',
    
    -- Customization
    logo_url VARCHAR(500) NULL COMMENT 'Tenant logo',
    primary_color VARCHAR(20) NULL COMMENT 'Brand color',
    custom_domain VARCHAR(255) NULL COMMENT 'Custom domain (e.g., acme.pantheon.io)',
    
    -- Contact
    admin_email VARCHAR(255) NOT NULL COMMENT 'Tenant admin email',
    admin_name VARCHAR(100) NULL COMMENT 'Tenant admin name',
    
    -- Metadata
    settings JSON NULL COMMENT 'Tenant-specific settings',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    deleted_at DATETIME NULL COMMENT 'Soft delete',
    
    INDEX idx_tenants_slug (slug),
    INDEX idx_tenants_status (status),
    INDEX idx_tenants_custom_domain (custom_domain)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
```

#### 2. Add tenant_id to Existing Tables

```sql
-- Users table
ALTER TABLE users 
ADD COLUMN tenant_id VARCHAR(64) NOT NULL DEFAULT 'default' AFTER id,
ADD CONSTRAINT fk_users_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id),
ADD INDEX idx_users_tenant_id (tenant_id);

-- Unique constraint: username per tenant
ALTER TABLE users DROP INDEX idx_users_username;
ALTER TABLE users ADD UNIQUE INDEX idx_users_tenant_username (tenant_id, username);

-- Roles table
ALTER TABLE system_roles
ADD COLUMN tenant_id VARCHAR(64) NOT NULL DEFAULT 'default' AFTER id,
ADD CONSTRAINT fk_roles_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id),
ADD INDEX idx_roles_tenant_id (tenant_id);

-- Departments table
ALTER TABLE system_dept
ADD COLUMN tenant_id VARCHAR(64) NOT NULL DEFAULT 'default' AFTER id,
ADD CONSTRAINT fk_dept_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id),
ADD INDEX idx_dept_tenant_id (tenant_id);

-- Repeat for all domain tables:
-- - system_permissions
-- - system_menu
-- - system_dict_type
-- - system_dict_data
-- - system_config
-- - system_operation_log
-- - system_login_log
-- ... etc
```

#### 3. Tenant-Specific Settings Table

```sql
CREATE TABLE tenant_settings (
    id BIGINT UNSIGNED PRIMARY KEY AUTO_INCREMENT,
    tenant_id VARCHAR(64) NOT NULL,
    setting_key VARCHAR(100) NOT NULL COMMENT 'e.g., session_timeout, password_policy',
    setting_value TEXT NULL COMMENT 'JSON or plain text',
    data_type VARCHAR(20) DEFAULT 'string' COMMENT 'string, number, boolean, json',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    UNIQUE INDEX idx_tenant_settings_key (tenant_id, setting_key),
    CONSTRAINT fk_tenant_settings_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
```

---

## Tenant Identification Strategy

### Request Flow

```
Request → Middleware → Tenant Resolution → Context Injection → Handler
```

### Tenant Resolution Methods

**Priority Order**:
1. **Custom Domain** (highest priority)
2. **Subdomain**
3. **Header** (`X-Tenant-ID`)
4. **JWT Claim** (`tenant_id`)

#### Method 1: Subdomain-based

```
https://acme.pantheon.io/login    → tenant: acme
https://startup.pantheon.io/api   → tenant: startup
```

**Implementation**:
```go
func extractTenantFromSubdomain(host string) string {
    // Parse host: acme.pantheon.io → acme
    parts := strings.Split(host, ".")
    if len(parts) >= 3 {
        return parts[0]
    }
    return "default"
}
```

#### Method 2: Custom Domain

```
https://app.acme.com              → tenant: acme (via domain mapping)
```

**Lookup**:
```sql
SELECT tenant_id FROM tenants WHERE custom_domain = 'app.acme.com'
```

#### Method 3: Header-based (API clients)

```http
GET /api/v1/users
Host: api.pantheon.io
X-Tenant-ID: acme
Authorization: Bearer <token>
```

#### Method 4: JWT Claim

```json
{
  "user_id": 123,
  "tenant_id": "acme",
  "exp": 1234567890
}
```

---

## Middleware Implementation

### Tenant Context Middleware

**File**: `backend/internal/middleware/tenant_middleware.go`

```go
package middleware

import (
    "context"
    "github.com/gin-gonic/gin"
    "gorm.io/gorm"
)

type TenantMiddleware struct {
    db *gorm.DB
    tenantCache map[string]*Tenant // In-memory cache
}

func NewTenantMiddleware(db *gorm.DB) *TenantMiddleware {
    return &TenantMiddleware{
        db: db,
        tenantCache: make(map[string]*Tenant),
    }
}

func (m *TenantMiddleware) Handle() gin.HandlerFunc {
    return func(c *gin.Context) {
        // 1. Resolve tenant
        tenantID := m.resolveTenant(c)
        
        // 2. Load tenant info
        tenant, err := m.loadTenant(tenantID)
        if err != nil {
            c.JSON(404, gin.H{"error": "Tenant not found"})
            c.Abort()
            return
        }
        
        // 3. Check tenant status
        if tenant.Status != "active" {
            c.JSON(403, gin.H{"error": "Tenant suspended"})
            c.Abort()
            return
        }
        
        // 4. Inject tenant into context
        c.Set("tenant_id", tenant.ID)
        c.Set("tenant", tenant)
        
        c.Next()
    }
}

func (m *TenantMiddleware) resolveTenant(c *gin.Context) string {
    // Priority 1: Custom domain
    host := c.Request.Host
    if tenant := m.lookupByDomain(host); tenant != "" {
        return tenant
    }
    
    // Priority 2: Subdomain
    if tenant := extractTenantFromSubdomain(host); tenant != "default" {
        return tenant
    }
    
    // Priority 3: Header
    if tenantID := c.GetHeader("X-Tenant-ID"); tenantID != "" {
        return tenantID
    }
    
    // Priority 4: JWT claim
    if tenantID, exists := c.Get("tenant_id"); exists {
        return tenantID.(string)
    }
    
    // Default tenant
    return "default"
}

func (m *TenantMiddleware) loadTenant(tenantID string) (*Tenant, error) {
    // Check cache
    if tenant, ok := m.tenantCache[tenantID]; ok {
        return tenant, nil
    }
    
    // Load from DB
    var tenant Tenant
    if err := m.db.Where("id = ? AND deleted_at IS NULL", tenantID).First(&tenant).Error; err != nil {
        return nil, err
    }
    
    // Cache for 5 minutes
    m.tenantCache[tenantID] = &tenant
    
    return &tenant, nil
}
```

### GORM Scoped Queries

**Automatic tenant filtering**:

```go
// Set up tenant scope on DB session
func WithTenant(db *gorm.DB, tenantID string) *gorm.DB {
    return db.Where("tenant_id = ?", tenantID)
}

// Usage in repository
type UserRepository struct {
    db *gorm.DB
}

func (r *UserRepository) List(ctx context.Context) ([]*User, error) {
    tenantID := GetTenantID(ctx)
    
    var users []*User
    err := WithTenant(r.db, tenantID).Find(&users).Error
    return users, err
}

func (r *UserRepository) Create(ctx context.Context, user *User) error {
    tenantID := GetTenantID(ctx)
    user.TenantID = tenantID
    return r.db.Create(user).Error
}
```

---

## Tenant Onboarding Flow

### Registration Flow

```
1. User visits /register/tenant
2. Fill form: Company Name, Admin Email, Plan
3. Submit → Backend validates
4. Create tenant record
5. Create default admin user
6. Initialize tenant data (roles, permissions, menus)
7. Send welcome email
8. Redirect to tenant subdomain
```

### Tenant Initialization

```go
func (s *TenantService) CreateTenant(req *CreateTenantRequest) (*Tenant, error) {
    // 1. Generate tenant ID
    tenantID := "tenant-" + uuid.New().String()
    
    // 2. Create tenant record
    tenant := &Tenant{
        ID:         tenantID,
        Name:       req.Name,
        Slug:       generateSlug(req.Name),
        PlanType:   req.Plan,
        AdminEmail: req.AdminEmail,
        Status:     "active",
    }
    
    if err := s.db.Create(tenant).Error; err != nil {
        return nil, err
    }
    
    // 3. Create default admin user
    admin := &User{
        TenantID: tenantID,
        Username: "admin",
        Email:    req.AdminEmail,
        Password: hashPassword(req.Password),
        Status:   "active",
    }
    
    if err := s.db.Create(admin).Error; err != nil {
        return nil, err
    }
    
    // 4. Initialize tenant data
    if err := s.initializeTenantData(tenantID); err != nil {
        return nil, err
    }
    
    return tenant, nil
}

func (s *TenantService) initializeTenantData(tenantID string) error {
    // Create default roles
    roles := []Role{
        {TenantID: tenantID, Key: "admin", Name: "Administrator"},
        {TenantID: tenantID, Key: "user", Name: "Regular User"},
    }
    
    for _, role := range roles {
        s.db.Create(&role)
    }
    
    // Create default permissions
    // Create default menus
    // ... etc
    
    return nil
}
```

---

## Data Isolation Strategies

### 1. Query-Level Isolation (Code)

**Pros**:
- Explicit tenant filtering in queries
- Full control over data access
- Easy to implement

**Cons**:
- Risk of forgetting tenant filter
- Requires discipline

**Implementation**:
```go
// Always include tenant_id in WHERE clause
db.Where("tenant_id = ? AND status = ?", tenantID, "active").Find(&users)
```

### 2. Middleware-Level Isolation

**Pros**:
- Automatic injection
- Centralized logic

**Cons**:
- Global state considerations
- Complexity in nested calls

**Implementation**:
```go
// Middleware injects tenant_id into context
// Repositories extract from context
```

### 3. Database-Level Isolation (Views)

**Pros**:
- Enforced at DB level
- No code changes needed

**Cons**:
- Performance overhead
- Less flexible

**Implementation**:
```sql
CREATE VIEW tenant_users AS
SELECT * FROM users WHERE tenant_id = CURRENT_TENANT_ID();
```

### 4. Row-Level Security (RLS)

**Pros**:
- Ultimate safety
- Cannot bypass

**Cons**:
- Database-specific (PostgreSQL has native RLS, MySQL needs workarounds)
- Complexity

---

## Tenant Customization

### Settings System

**Tenant-specific overrides**:

```go
type TenantSettings struct {
    SessionTimeout    int    `json:"session_timeout"`
    PasswordPolicy    string `json:"password_policy"`
    AllowedDomains    []string `json:"allowed_domains"`
    MaxUploadSize     int    `json:"max_upload_size"`
    EnableMFA         bool   `json:"enable_mfa"`
}

func (s *TenantService) GetSettings(tenantID string) (*TenantSettings, error) {
    // Load from tenant_settings table
    var settings TenantSettings
    
    // Get all settings for tenant
    rows, err := s.db.Table("tenant_settings").
        Where("tenant_id = ?", tenantID).
        Select("setting_key, setting_value").
        Rows()
    
    // Parse into struct
    // Apply defaults for missing values
    
    return &settings, nil
}
```

### Branding Customization

**File**: `backend/modules/tenant/branding.go`

```go
type TenantBranding struct {
    LogoURL      string `json:"logo_url"`
    PrimaryColor string `json:"primary_color"`
    SecondaryColor string `json:"secondary_color"`
    FaviconURL   string `json:"favicon_url"`
    CompanyName  string `json:"company_name"`
}

func (s *TenantService) GetBranding(tenantID string) (*TenantBranding, error) {
    var tenant Tenant
    if err := s.db.Select("logo_url, primary_color, name").
        Where("id = ?", tenantID).
        First(&tenant).Error; err != nil {
        return nil, err
    }
    
    return &TenantBranding{
        LogoURL:      tenant.LogoURL,
        PrimaryColor: tenant.PrimaryColor,
        CompanyName:  tenant.Name,
    }, nil
}
```

---

## Performance Considerations

### Caching Strategy

**Tenant Info Cache**:
- Cache tenant metadata (name, status, settings)
- TTL: 5 minutes
- Invalidate on tenant update

**Connection Pooling**:
- Single connection pool for all tenants (Phase 1)
- Per-tenant pools only if needed (Phase 2+)

### Query Optimization

**Indexes**:
```sql
-- Critical for tenant queries
CREATE INDEX idx_users_tenant_id ON users(tenant_id);
CREATE INDEX idx_roles_tenant_id ON system_roles(tenant_id);
CREATE INDEX idx_dept_tenant_id ON system_dept(tenant_id);

-- Composite indexes for common queries
CREATE INDEX idx_users_tenant_status ON users(tenant_id, status);
CREATE INDEX idx_users_tenant_username ON users(tenant_id, username);
```

**Query Patterns**:
```sql
-- Always use tenant_id as first condition
SELECT * FROM users WHERE tenant_id = 'acme' AND status = 'active';

-- Avoid cross-tenant queries
-- Bad: SELECT * FROM users WHERE status = 'active';
-- Good: SELECT * FROM users WHERE tenant_id = 'acme' AND status = 'active';
```

---

## Security Considerations

### Tenant Isolation Validation

**Automated Tests**:
```go
func TestTenantIsolation_UserListDoesNotLeakCrossTenant(t *testing.T) {
    // Create two tenants
    tenant1 := createTestTenant("tenant1")
    tenant2 := createTestTenant("tenant2")
    
    // Create users in each tenant
    user1 := createTestUser(tenant1.ID, "user1")
    user2 := createTestUser(tenant2.ID, "user2")
    
    // Query as tenant1
    ctx := contextWithTenant(tenant1.ID)
    users, _ := userRepo.List(ctx)
    
    // Assert: Only tenant1's users returned
    assert.Len(t, users, 1)
    assert.Equal(t, user1.ID, users[0].ID)
}
```

### Penetration Testing Checklist

- [ ] Cross-tenant data access attempts (tamper tenant_id in JWT)
- [ ] SQL injection with tenant_id bypass attempts
- [ ] Session hijacking across tenants
- [ ] API endpoint tenant validation
- [ ] File upload/download tenant isolation

---

## Migration Strategy

### Existing Single-Tenant → Multi-Tenant

**Steps**:
1. **Add tenant master table**
2. **Create default tenant**: `tenant-default`
3. **Add tenant_id columns** to all tables (default: `tenant-default`)
4. **Migrate data**: All existing data → `tenant-default`
5. **Update application code**: Add tenant middleware
6. **Test thoroughly**: Ensure backward compatibility
7. **Deploy**: Feature flag for multi-tenant mode

**SQL Migration**:
```sql
-- Step 1: Create tenants table
CREATE TABLE tenants (...);

-- Step 2: Insert default tenant
INSERT INTO tenants (id, name, slug, status) VALUES ('tenant-default', 'Default Tenant', 'default', 'active');

-- Step 3: Add tenant_id to users
ALTER TABLE users ADD COLUMN tenant_id VARCHAR(64) NOT NULL DEFAULT 'tenant-default';

-- Step 4: Update foreign key
ALTER TABLE users ADD CONSTRAINT fk_users_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id);

-- Step 5: Add index
CREATE INDEX idx_users_tenant_id ON users(tenant_id);

-- Repeat for all tables
```

---

## API Changes

### Tenant Management Endpoints

```
POST   /api/v1/tenants                  # Create tenant (super admin only)
GET    /api/v1/tenants                  # List all tenants (super admin)
GET    /api/v1/tenants/:id              # Get tenant details
PUT    /api/v1/tenants/:id              # Update tenant
DELETE /api/v1/tenants/:id              # Delete tenant (soft)

GET    /api/v1/tenants/current          # Get current tenant info
PUT    /api/v1/tenants/current/settings # Update tenant settings
PUT    /api/v1/tenants/current/branding # Update branding
```

### Tenant Registration (Public)

```
POST   /api/v1/register/tenant          # Register new tenant
GET    /api/v1/register/tenant/check-slug  # Check slug availability
```

---

## Frontend Integration

### Tenant Context Hook

```typescript
// hooks/useTenant.ts
export const useTenant = () => {
  const [tenant, setTenant] = useState<Tenant | null>(null);

  useEffect(() => {
    loadTenant();
  }, []);

  const loadTenant = async () => {
    const data = await api.get('/api/v1/tenants/current');
    setTenant(data);
  };

  return { tenant, reloadTenant: loadTenant };
};
```

### Dynamic Branding

```typescript
// App.tsx
const { tenant } = useTenant();

useEffect(() => {
  if (tenant?.branding) {
    document.documentElement.style.setProperty(
      '--primary-color',
      tenant.branding.primaryColor
    );
    
    if (tenant.branding.logoURL) {
      setLogoURL(tenant.branding.logoURL);
    }
  }
}, [tenant]);
```

---

## Rollout Plan

### Phase 1: MVP (Current Design)
- Shared database, shared schema
- Subdomain-based tenant identification
- Automatic tenant filtering in queries
- Basic tenant settings

### Phase 2: Enhanced (Future)
- Custom domain support
- Per-tenant connection pools
- Advanced branding customization
- Tenant analytics dashboard

### Phase 3: Enterprise (Future)
- Separate schemas per tenant
- Separate databases (optional)
- Tenant data export/import
- Compliance (GDPR, SOC2)

---

## Success Criteria

- [ ] Tenant table created
- [ ] All domain tables have tenant_id
- [ ] Tenant middleware implemented
- [ ] Automatic query filtering working
- [ ] Tenant isolation tests passing
- [ ] Onboarding flow functional
- [ ] Settings system working
- [ ] Branding customization working
- [ ] API endpoints implemented
- [ ] Frontend tenant context working

---

## Estimated Effort

- Design: 4 hours (✅ completed)
- Database migration: 3 hours
- Middleware implementation: 3 hours
- Repository updates: 4 hours
- API endpoints: 2 hours
- Frontend integration: 2 hours
- Testing: 2 hours
- **Total: 20 hours** (design + implementation)

---

**Document Status**: ✅ Design Complete  
**Next Phase**: Implementation (16 hours)  
**Owner**: Pantheon Base Backend Team
