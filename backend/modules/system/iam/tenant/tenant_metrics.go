package tenant

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"gorm.io/gorm"
)

var (
	// TenantCount tracks the number of active tenants
	TenantCount = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "pantheon_tenant_count",
		Help: "Total number of active tenants in the system",
	})

	// TenantUserCount tracks users per tenant
	TenantUserCount = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pantheon_tenant_user_count",
		Help: "Number of active users per tenant",
	}, []string{"tenant_id", "tenant_code"})

	// TenantAPIRequests tracks API request count per tenant
	TenantAPIRequests = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "pantheon_tenant_api_requests_total",
		Help: "Total number of API requests per tenant",
	}, []string{"tenant_id", "tenant_code", "method", "path"})

	// TenantQuotaUsage tracks resource usage against quotas
	TenantQuotaUsage = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pantheon_tenant_quota_usage_percent",
		Help: "Tenant resource usage as percentage of quota",
	}, []string{"tenant_id", "tenant_code", "resource"})

	// TenantOperations tracks tenant lifecycle operations
	TenantOperations = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "pantheon_tenant_operations_total",
		Help: "Total number of tenant operations",
	}, []string{"operation", "status"})

	// TenantDataIsolationViolations tracks isolation integrity issues
	TenantDataIsolationViolations = promauto.NewCounter(prometheus.CounterOpts{
		Name: "pantheon_tenant_isolation_violations_total",
		Help: "Total number of tenant data isolation violations detected",
	})

	// TenantSwitchOperations tracks tenant context switches
	TenantSwitchOperations = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "pantheon_tenant_switch_operations_total",
		Help: "Total number of tenant switch operations",
	}, []string{"from_tenant", "to_tenant", "status"})
)

// MetricsCollector periodically collects and updates tenant metrics
type MetricsCollector struct {
	db            *gorm.DB
	tenantService *Service
	quotaEnforcer *QuotaEnforcer
	healthChecker *HealthChecker
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(
	db *gorm.DB,
	tenantService *Service,
	quotaEnforcer *QuotaEnforcer,
	healthChecker *HealthChecker,
) *MetricsCollector {
	return &MetricsCollector{
		db:            db,
		tenantService: tenantService,
		quotaEnforcer: quotaEnforcer,
		healthChecker: healthChecker,
	}
}

// CollectMetrics collects all tenant-related metrics
func (c *MetricsCollector) CollectMetrics() error {
	// Update tenant count
	var activeTenantCount int64
	if err := c.db.Model(&Tenant{}).Where("status = ?", "active").Count(&activeTenantCount).Error; err != nil {
		return err
	}
	TenantCount.Set(float64(activeTenantCount))

	// Get all active tenants
	var tenants []Tenant
	if err := c.db.Where("status = ?", "active").Find(&tenants).Error; err != nil {
		return err
	}

	// Update per-tenant metrics
	for _, tenant := range tenants {
		// User count
		var userCount int64
		c.db.Model(&Membership{}).
			Where("tenant_id = ? AND status = ?", tenant.ID, "active").
			Count(&userCount)
		TenantUserCount.WithLabelValues(
			string(rune(tenant.ID)),
			tenant.Code,
		).Set(float64(userCount))

		// Quota usage
		usage, err := c.quotaEnforcer.GetTenantUsage(tenant.ID)
		if err == nil {
			quota := GetDefaultQuota(tenant.Plan)

			// User quota
			if quota.MaxUsers > 0 {
				userUsagePercent := float64(usage["users"]) / float64(quota.MaxUsers) * 100
				TenantQuotaUsage.WithLabelValues(
					string(rune(tenant.ID)),
					tenant.Code,
					"users",
				).Set(userUsagePercent)
			}

			// Role quota
			if quota.MaxRoles > 0 {
				roleUsagePercent := float64(usage["roles"]) / float64(quota.MaxRoles) * 100
				TenantQuotaUsage.WithLabelValues(
					string(rune(tenant.ID)),
					tenant.Code,
					"roles",
				).Set(roleUsagePercent)
			}
		}

		// Check isolation violations
		health, err := c.healthChecker.CheckTenantHealth(tenant.ID)
		if err == nil {
			if violations, ok := health["isolation_violations"].(int64); ok && violations > 0 {
				TenantDataIsolationViolations.Add(float64(violations))
			}
		}
	}

	return nil
}

// RecordTenantOperation records a tenant lifecycle operation
func RecordTenantOperation(operation string, success bool) {
	status := "success"
	if !success {
		status = "failure"
	}
	TenantOperations.WithLabelValues(operation, status).Inc()
}

// RecordTenantSwitch records a tenant context switch
func RecordTenantSwitch(fromTenantID, toTenantID uint64, success bool) {
	status := "success"
	if !success {
		status = "failure"
	}
	TenantSwitchOperations.WithLabelValues(
		string(rune(fromTenantID)),
		string(rune(toTenantID)),
		status,
	).Inc()
}

// RecordTenantAPIRequest records an API request for a tenant
func RecordTenantAPIRequest(tenantID uint64, tenantCode, method, path string) {
	TenantAPIRequests.WithLabelValues(
		string(rune(tenantID)),
		tenantCode,
		method,
		path,
	).Inc()
}
