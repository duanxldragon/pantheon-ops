package tenant

import (
	"errors"
	"fmt"

	"gorm.io/gorm"
)

var (
	// ErrTenantNotFound is returned when no tenant matches the given id/code.
	ErrTenantNotFound = errors.New("tenant not found")
	// ErrTenantCodeExists is returned when creating a tenant with an already-taken code.
	ErrTenantCodeExists = errors.New("tenant code already exists")
	// ErrTenantHasActiveUsers is returned when deleting a tenant that still has active members.
	ErrTenantHasActiveUsers = errors.New("tenant has active users, cannot delete")
	// ErrMembershipExists is returned when adding a user who is already a tenant member.
	ErrMembershipExists = errors.New("user is already a member of this tenant")
	// ErrMembershipNotFound is returned when removing a non-existent tenant membership.
	ErrMembershipNotFound = errors.New("tenant membership not found")
)

const (
	tenantIDPredicate           = "tenant_id = ?"
	activeTenantMemberPredicate = "tenant_id = ? AND status = ?"
)

// Service handles tenant CRUD and lifecycle operations
type Service struct {
	db *gorm.DB
}

// NewService creates a new tenant service instance
func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

// CreateTenant creates a new tenant with default settings
func (s *Service) CreateTenant(dto CreateTenantDTO) (*Tenant, error) {
	// Check if code already exists
	var exists int64
	if err := s.db.Model(&Tenant{}).Where("code = ?", dto.Code).Count(&exists).Error; err != nil {
		return nil, fmt.Errorf("check tenant code uniqueness: %w", err)
	}
	if exists > 0 {
		return nil, ErrTenantCodeExists
	}

	tenant := &Tenant{
		Code:     dto.Code,
		Name:     dto.Name,
		Status:   "active",
		Plan:     dto.Plan,
		Metadata: dto.Metadata,
	}

	if tenant.Plan == "" {
		tenant.Plan = "free"
	}

	if err := s.db.Create(tenant).Error; err != nil {
		return nil, fmt.Errorf("create tenant: %w", err)
	}

	return tenant, nil
}

// GetTenantByID retrieves a tenant by its ID
func (s *Service) GetTenantByID(id uint64) (*Tenant, error) {
	var tenant Tenant
	// Use Where clause with placeholder to satisfy SonarCloud taint analysis
	if err := s.db.Where("id = ?", id).First(&tenant).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTenantNotFound
		}
		return nil, fmt.Errorf("get tenant by id: %w", err)
	}
	return &tenant, nil
}

// GetTenantByCode retrieves a tenant by its code
func (s *Service) GetTenantByCode(code string) (*Tenant, error) {
	var tenant Tenant
	if err := s.db.Where("code = ?", code).First(&tenant).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTenantNotFound
		}
		return nil, fmt.Errorf("get tenant by code: %w", err)
	}
	return &tenant, nil
}

// ListTenants returns a paginated list of tenants
func (s *Service) ListTenants(filter Filter) (*ListResponse, error) {
	// Set defaults
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Size < 1 {
		filter.Size = 20
	}

	query := s.db.Model(&Tenant{})

	// Apply filters
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.Plan != "" {
		query = query.Where("plan = ?", filter.Plan)
	}

	// Count total
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("count tenants: %w", err)
	}

	// Fetch page
	var tenants []Tenant
	offset := (filter.Page - 1) * filter.Size
	if err := query.Order("id DESC").Limit(filter.Size).Offset(offset).Find(&tenants).Error; err != nil {
		return nil, fmt.Errorf("list tenants: %w", err)
	}

	return &ListResponse{
		Items: tenants,
		Total: total,
		Page:  filter.Page,
		Size:  filter.Size,
	}, nil
}

// UpdateTenant updates tenant fields
func (s *Service) UpdateTenant(id uint64, dto UpdateTenantDTO) error {
	tenant, err := s.GetTenantByID(id)
	if err != nil {
		return err
	}

	// Update fields individually to satisfy SonarCloud taint analysis
	// This avoids map/struct updates that trigger false positive SQL injection warnings
	hasUpdates := false

	if dto.Name != nil && *dto.Name != "" {
		if err := s.db.Model(tenant).Update("name", *dto.Name).Error; err != nil {
			return fmt.Errorf("update tenant name: %w", err)
		}
		hasUpdates = true
	}
	if dto.Status != nil && *dto.Status != "" {
		if err := s.db.Model(tenant).Update("status", *dto.Status).Error; err != nil {
			return fmt.Errorf("update tenant status: %w", err)
		}
		hasUpdates = true
	}
	if dto.Plan != nil && *dto.Plan != "" {
		if err := s.db.Model(tenant).Update("plan", *dto.Plan).Error; err != nil {
			return fmt.Errorf("update tenant plan: %w", err)
		}
		hasUpdates = true
	}
	if dto.Metadata != nil {
		if err := s.db.Model(tenant).Update("metadata", *dto.Metadata).Error; err != nil {
			return fmt.Errorf("update tenant metadata: %w", err)
		}
		hasUpdates = true
	}

	if !hasUpdates {
		return nil // No updates
	}

	return nil
}

// DeleteTenant soft-deletes a tenant (marks as deleted status)
func (s *Service) DeleteTenant(id uint64) error {
	// Check if tenant has active members
	var memberCount int64
	if err := s.db.Model(&Membership{}).
		Where(activeTenantMemberPredicate, id, "active").
		Count(&memberCount).Error; err != nil {
		return fmt.Errorf("check tenant members: %w", err)
	}

	if memberCount > 0 {
		return ErrTenantHasActiveUsers
	}

	// Mark tenant as deleted
	if err := s.db.Model(&Tenant{}).Where("id = ?", id).Update("status", "deleted").Error; err != nil {
		return fmt.Errorf("delete tenant: %w", err)
	}

	return nil
}

// AddMember adds a user to a tenant with specified role
func (s *Service) AddMember(tenantID, userID uint64, role string) error {
	// Verify tenant exists
	if _, err := s.GetTenantByID(tenantID); err != nil {
		return err
	}

	// Check if membership already exists
	var exists int64
	if err := s.db.Model(&Membership{}).
		Where("tenant_id = ? AND user_id = ?", tenantID, userID).
		Count(&exists).Error; err != nil {
		return fmt.Errorf("check membership existence: %w", err)
	}
	if exists > 0 {
		return ErrMembershipExists
	}

	membership := &Membership{
		TenantID: tenantID,
		UserID:   userID,
		Role:     role,
		Status:   "active",
	}

	if err := s.db.Create(membership).Error; err != nil {
		return fmt.Errorf("create membership: %w", err)
	}

	return nil
}

// RemoveMember removes a user from a tenant
func (s *Service) RemoveMember(tenantID, userID uint64) error {
	result := s.db.Where("tenant_id = ? AND user_id = ?", tenantID, userID).
		Delete(&Membership{})

	if result.Error != nil {
		return fmt.Errorf("remove membership: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrMembershipNotFound
	}

	return nil
}

const (
	defaultTenantMemberPageSize = 20
	maxTenantMemberPageSize     = 100
)

// ListAllTenantMembers preserves the legacy unpaginated member listing.
func (s *Service) ListAllTenantMembers(tenantID uint64) ([]Membership, error) {
	memberships := make([]Membership, 0)
	if err := s.db.Where(activeTenantMemberPredicate, tenantID, "active").
		Order("created_at ASC, id ASC").
		Find(&memberships).Error; err != nil {
		return nil, fmt.Errorf("list tenant members: %w", err)
	}
	return memberships, nil
}

// ListTenantMembers returns one bounded page of active memberships and its total.
func (s *Service) ListTenantMembers(tenantID uint64, page, pageSize int) ([]Membership, int64, error) {
	page, pageSize = normalizeTenantMemberPagination(page, pageSize)

	scope := s.db.Model(&Membership{}).
		Where(activeTenantMemberPredicate, tenantID, "active")

	var total int64
	if err := scope.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count tenant members: %w", err)
	}

	memberships := make([]Membership, 0)
	if err := scope.
		Order("created_at ASC, id ASC").
		Limit(pageSize).
		Offset((page - 1) * pageSize).
		Find(&memberships).Error; err != nil {
		return nil, 0, fmt.Errorf("list tenant members: %w", err)
	}
	return memberships, total, nil
}

func normalizeTenantMemberPagination(page, pageSize int) (int, int) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = defaultTenantMemberPageSize
	}
	if pageSize > maxTenantMemberPageSize {
		pageSize = maxTenantMemberPageSize
	}
	if maxPage := int(^uint(0)>>1) / pageSize; page > maxPage {
		page = maxPage
	}
	return page, pageSize
}

// ActiveMembershipRole returns the role of a user's active membership in one
// tenant without loading the full member list.
func (s *Service) ActiveMembershipRole(tenantID, userID uint64) (string, bool, error) {
	var membership Membership
	err := s.db.
		Where("tenant_id = ? AND user_id = ? AND status = ?", tenantID, userID, "active").
		First(&membership).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("check tenant membership: %w", err)
	}
	return membership.Role, true, nil
}

// ActiveMembershipRolesByTenant resolves the user's active role for each given
// tenant with a single bounded query instead of one full member list per
// tenant (which made tenant switch listings O(tenants × members)).
func (s *Service) ActiveMembershipRolesByTenant(userID uint64, tenantIDs []uint64) (map[uint64]string, error) {
	roles := make(map[uint64]string, len(tenantIDs))
	if len(tenantIDs) == 0 {
		return roles, nil
	}
	var rows []Membership
	if err := s.db.
		Where("user_id = ? AND tenant_id IN ? AND status = ?", userID, tenantIDs, "active").
		Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("list user membership roles: %w", err)
	}
	for _, row := range rows {
		roles[row.TenantID] = row.Role
	}
	return roles, nil
}

// GetUserTenants returns all tenants a user belongs to
func (s *Service) GetUserTenants(userID uint64) ([]Tenant, error) {
	var tenants []Tenant
	if err := s.db.Table("tenants").
		Joins("INNER JOIN tenant_memberships ON tenant_memberships.tenant_id = tenants.id").
		Where("tenant_memberships.user_id = ? AND tenant_memberships.status = ?", userID, "active").
		Where("tenants.status = ?", "active").
		Order("tenants.id ASC").
		Find(&tenants).Error; err != nil {
		return nil, fmt.Errorf("get user tenants: %w", err)
	}
	return tenants, nil
}

// InitializeTenantDefaults creates default resources for a new tenant
// (roles, menus, settings, etc.)
func (s *Service) InitializeTenantDefaults(tenantID uint64) error {
	// Phase 2.2 default-resource creation (roles, menus, settings, permissions)
	// has not shipped: this entry point is reserved for the per-tenant seeding
	// variant and currently has no callers, so it stays a documented no-op.
	return nil
}
