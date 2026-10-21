package tenant

import (
	"fmt"

	"gorm.io/gorm"
)

// BootstrapService handles tenant-related bootstrap operations
type BootstrapService struct {
	db            *gorm.DB
	tenantService *Service
}

// NewBootstrapService creates a new bootstrap service
func NewBootstrapService(db *gorm.DB, tenantService *Service) *BootstrapService {
	return &BootstrapService{
		db:            db,
		tenantService: tenantService,
	}
}

// EnsureDefaultTenant creates the default tenant if it doesn't exist
// This is called during system bootstrap to ensure compat → multi migration path
func (s *BootstrapService) EnsureDefaultTenant() (*Tenant, error) {
	// Check if default tenant already exists
	tenant, err := s.tenantService.GetTenantByCode("default")
	if err == nil {
		return tenant, nil // Already exists
	}

	if err != ErrTenantNotFound {
		return nil, fmt.Errorf("check default tenant existence: %w", err)
	}

	// Create default tenant
	tenant, err = s.tenantService.CreateTenant(CreateTenantDTO{
		Code:     "default",
		Name:     "默认租户",
		Plan:     "enterprise",
		Metadata: `{"bootstrap":true,"description":"Auto-created default tenant for compat mode migration"}`,
	})
	if err != nil {
		return nil, fmt.Errorf("create default tenant: %w", err)
	}

	return tenant, nil
}

// AssignAdminToDefaultTenant assigns the admin user to the default tenant
// This is called during bootstrap after admin user creation
func (s *BootstrapService) AssignAdminToDefaultTenant(adminUserID uint64) error {
	// Get or create default tenant
	tenant, err := s.EnsureDefaultTenant()
	if err != nil {
		return err
	}

	// Check if admin is already a member (bounded single-membership lookup)
	_, isMember, err := s.tenantService.ActiveMembershipRole(tenant.ID, adminUserID)
	if err != nil {
		return fmt.Errorf("check admin membership: %w", err)
	}
	if isMember {
		return nil // Already a member
	}

	// Add admin as owner of default tenant
	if err := s.tenantService.AddMember(tenant.ID, adminUserID, "owner"); err != nil {
		return fmt.Errorf("add admin to default tenant: %w", err)
	}

	return nil
}

// MigrateExistingUsersToDefaultTenant assigns all existing users to the default tenant
// This is useful for compat → multi mode migration
func (s *BootstrapService) MigrateExistingUsersToDefaultTenant() (int, error) {
	// Get or create default tenant
	tenant, err := s.EnsureDefaultTenant()
	if err != nil {
		return 0, err
	}

	// Find all users not in any tenant
	var userIDs []uint64
	if err := s.db.Raw(`
		SELECT u.id
		FROM system_user u
		LEFT JOIN tenant_memberships tm ON tm.user_id = u.id AND tm.status = 'active'
		WHERE tm.id IS NULL
	`).Scan(&userIDs).Error; err != nil {
		return 0, fmt.Errorf("find unmapped users: %w", err)
	}

	// Assign each user to default tenant
	migrated := 0
	for _, userID := range userIDs {
		if err := s.tenantService.AddMember(tenant.ID, userID, "member"); err != nil {
			// Log error but continue
			fmt.Printf("Warning: failed to migrate user %d to default tenant: %v\n", userID, err)
			continue
		}
		migrated++
	}

	return migrated, nil
}

// InitializeTenantInfrastructure creates necessary tenant infrastructure
// (called once during first-time system setup)
func (s *BootstrapService) InitializeTenantInfrastructure() error {
	// Ensure default tenant exists
	_, err := s.EnsureDefaultTenant()
	if err != nil {
		return fmt.Errorf("ensure default tenant: %w", err)
	}

	// Set platform.tenant_mode to compat if not set
	var settingExists int64
	if err := s.db.Raw(`
		SELECT COUNT(*)
		FROM system_setting
		WHERE setting_key = 'platform.tenant_mode'
	`).Scan(&settingExists).Error; err != nil {
		return fmt.Errorf("check tenant_mode setting: %w", err)
	}

	if settingExists == 0 {
		// Create compat mode setting
		if err := s.db.Exec(`
			INSERT INTO system_setting (tenant_id, setting_key, setting_value, created_at, updated_at)
			VALUES (0, 'platform.tenant_mode', '"compat"', NOW(), NOW())
		`).Error; err != nil {
			return fmt.Errorf("create tenant_mode setting: %w", err)
		}
	}

	return nil
}
