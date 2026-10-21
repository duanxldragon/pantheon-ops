package tenant

import (
	"errors"
	"fmt"

	"gorm.io/gorm"
)

// ErrQuotaExceeded is returned when a tenant operation exceeds its plan quota.
var (
	ErrQuotaExceeded = errors.New("tenant quota exceeded")
)

// Quota represents quota limits for a tenant
type Quota struct {
	MaxUsers     int `json:"max_users"`
	MaxRoles     int `json:"max_roles"`
	MaxDepts     int `json:"max_depts"`
	MaxAPIKeys   int `json:"max_api_keys"`
	MaxStorageMB int `json:"max_storage_mb"`
}

// GetDefaultQuota returns default quota based on plan
func GetDefaultQuota(plan string) Quota {
	switch plan {
	case "free":
		return Quota{
			MaxUsers:     5,
			MaxRoles:     3,
			MaxDepts:     5,
			MaxAPIKeys:   2,
			MaxStorageMB: 100,
		}
	case "basic":
		return Quota{
			MaxUsers:     20,
			MaxRoles:     10,
			MaxDepts:     20,
			MaxAPIKeys:   10,
			MaxStorageMB: 1000,
		}
	case "professional":
		return Quota{
			MaxUsers:     100,
			MaxRoles:     50,
			MaxDepts:     100,
			MaxAPIKeys:   50,
			MaxStorageMB: 10000,
		}
	case "enterprise":
		return Quota{
			MaxUsers:     -1, // Unlimited
			MaxRoles:     -1,
			MaxDepts:     -1,
			MaxAPIKeys:   -1,
			MaxStorageMB: -1,
		}
	default:
		return GetDefaultQuota("free")
	}
}

// QuotaEnforcer handles tenant quota enforcement
type QuotaEnforcer struct {
	db *gorm.DB
}

// NewQuotaEnforcer creates a new quota enforcer instance
func NewQuotaEnforcer(db *gorm.DB) *QuotaEnforcer {
	return &QuotaEnforcer{db: db}
}

// CheckUserQuota verifies if tenant can add more users
func (e *QuotaEnforcer) CheckUserQuota(tenantID uint64) error {
	var tenant Tenant
	if err := e.db.First(&tenant, tenantID).Error; err != nil {
		return fmt.Errorf("get tenant: %w", err)
	}

	quota := GetDefaultQuota(tenant.Plan)
	if quota.MaxUsers < 0 {
		return nil // Unlimited
	}

	var count int64
	if err := e.db.Model(&Membership{}).
		Where(activeTenantMemberPredicate, tenantID, "active").
		Count(&count).Error; err != nil {
		return fmt.Errorf("count users: %w", err)
	}

	if int(count) >= quota.MaxUsers {
		return fmt.Errorf("%w: max users (%d) reached", ErrQuotaExceeded, quota.MaxUsers)
	}

	return nil
}

// CheckRoleQuota verifies if tenant can create more roles
func (e *QuotaEnforcer) CheckRoleQuota(tenantID uint64) error {
	var tenant Tenant
	if err := e.db.First(&tenant, tenantID).Error; err != nil {
		return fmt.Errorf("get tenant: %w", err)
	}

	quota := GetDefaultQuota(tenant.Plan)
	if quota.MaxRoles < 0 {
		return nil // Unlimited
	}

	var count int64
	if err := e.db.Table("system_role").
		Where(tenantIDPredicate, tenantID).
		Count(&count).Error; err != nil {
		return fmt.Errorf("count roles: %w", err)
	}

	if int(count) >= quota.MaxRoles {
		return fmt.Errorf("%w: max roles (%d) reached", ErrQuotaExceeded, quota.MaxRoles)
	}

	return nil
}

// GetTenantUsage returns current resource usage for a tenant
func (e *QuotaEnforcer) GetTenantUsage(tenantID uint64) (map[string]int64, error) {
	usage := make(map[string]int64)

	// Count users
	var userCount int64
	e.db.Model(&Membership{}).
		Where(activeTenantMemberPredicate, tenantID, "active").
		Count(&userCount)
	usage["users"] = userCount

	// Count roles
	var roleCount int64
	e.db.Table("system_role").
		Where(tenantIDPredicate, tenantID).
		Count(&roleCount)
	usage["roles"] = roleCount

	// Count depts
	var deptCount int64
	e.db.Table("system_dept").
		Where(tenantIDPredicate, tenantID).
		Count(&deptCount)
	usage["depts"] = deptCount

	// Count API keys (if table exists)
	var apiKeyCount int64
	e.db.Table("api_keys").
		Where(tenantIDPredicate, tenantID).
		Count(&apiKeyCount)
	usage["api_keys"] = apiKeyCount

	return usage, nil
}

// AuditLogger logs tenant-related audit events
type AuditLogger struct {
	db *gorm.DB
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger(db *gorm.DB) *AuditLogger {
	return &AuditLogger{db: db}
}

// LogTenantCreated logs tenant creation event
func (l *AuditLogger) LogTenantCreated(tenantID, operatorID uint64, snapshot string) error {
	return l.logEvent(tenantID, operatorID, "tenant.created", snapshot)
}

// LogTenantUpdated logs tenant update event
func (l *AuditLogger) LogTenantUpdated(tenantID, operatorID uint64, snapshot string) error {
	return l.logEvent(tenantID, operatorID, "tenant.updated", snapshot)
}

// LogTenantDeleted logs tenant deletion event
func (l *AuditLogger) LogTenantDeleted(tenantID, operatorID uint64, snapshot string) error {
	return l.logEvent(tenantID, operatorID, "tenant.deleted", snapshot)
}

// LogMemberAdded logs member addition event
func (l *AuditLogger) LogMemberAdded(tenantID, operatorID, memberUserID uint64, role string) error {
	snapshot := fmt.Sprintf(`{"user_id":%d,"role":"%s"}`, memberUserID, role)
	return l.logEvent(tenantID, operatorID, "tenant.member.added", snapshot)
}

// LogMemberRemoved logs member removal event
func (l *AuditLogger) LogMemberRemoved(tenantID, operatorID, memberUserID uint64) error {
	snapshot := fmt.Sprintf(`{"user_id":%d}`, memberUserID)
	return l.logEvent(tenantID, operatorID, "tenant.member.removed", snapshot)
}

func (l *AuditLogger) logEvent(tenantID, operatorID uint64, eventType, detail string) error {
	// Log to operation_logs table (which already has tenant_id support)
	return l.db.Exec(`
		INSERT INTO operation_logs (tenant_id, user_id, module, operation, detail, created_at)
		VALUES (?, ?, 'tenant', ?, ?, NOW())
	`, tenantID, operatorID, eventType, detail).Error
}

// HealthChecker provides health check utilities
type HealthChecker struct {
	db *gorm.DB
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(db *gorm.DB) *HealthChecker {
	return &HealthChecker{db: db}
}

// CheckTenantHealth performs comprehensive health check on a tenant
func (c *HealthChecker) CheckTenantHealth(tenantID uint64) (map[string]interface{}, error) {
	result := make(map[string]interface{})

	// Check tenant exists
	var tenant Tenant
	if err := c.db.First(&tenant, tenantID).Error; err != nil {
		return nil, fmt.Errorf("tenant not found: %w", err)
	}
	result["tenant_status"] = tenant.Status

	// Check member count
	var memberCount int64
	c.db.Model(&Membership{}).
		Where(activeTenantMemberPredicate, tenantID, "active").
		Count(&memberCount)
	result["member_count"] = memberCount

	// Check for orphaned data
	var orphanedUsers int64
	c.db.Raw(`
		SELECT COUNT(*)
		FROM system_user
		WHERE tenant_id = ?
		AND id NOT IN (SELECT user_id FROM tenant_memberships WHERE tenant_id = ? AND status = 'active')
	`, tenantID, tenantID).Scan(&orphanedUsers)
	result["orphaned_users"] = orphanedUsers

	// Check isolation integrity
	var isolationViolations int64
	c.db.Raw(`
		SELECT COUNT(*)
		FROM system_role r
		LEFT JOIN system_user_role ur ON ur.role_id = r.id
		WHERE r.tenant_id = ? AND ur.tenant_id != r.tenant_id
	`, tenantID).Scan(&isolationViolations)
	result["isolation_violations"] = isolationViolations

	result["healthy"] = orphanedUsers == 0 && isolationViolations == 0

	return result, nil
}
