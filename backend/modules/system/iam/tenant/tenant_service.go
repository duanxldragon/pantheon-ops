package tenant

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/database"
	"github.com/duanxldragon/pantheon-base/backend/pkg/logging"
	"github.com/duanxldragon/pantheon-base/backend/pkg/tenant"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// Shared predicates for the tenant table. They are declared once so the
// placeholder style stays identical across every query, and so a reader cannot
// mistake a literal for an interpolated value.
const (
	condIDEquals        = "id = ?"
	condIDNotEquals     = "id <> ?"
	condStatusEquals    = "status = ?"
	condStatusNotEquals = "status <> ?"
	condPlanEquals      = "plan = ?"
)

var (
	// ErrTenantNotFound is returned when no tenant matches the given id/code.
	ErrTenantNotFound = errors.New("tenant not found")
	// ErrTenantCodeExists is returned when creating a tenant with a code that a
	// live tenant already holds.
	ErrTenantCodeExists = errors.New("tenant code already exists")
	// ErrTenantCodeReserved is returned when creating a tenant with a code that
	// only a soft-deleted tenant holds. It is deliberately distinct from
	// ErrTenantCodeExists: the two have different remedies (rename the deleted
	// tenant vs. pick another code), and collapsing them is what left the
	// operator staring at a bare `tenant.conflict` with an invisible row in the
	// way. ReleaseDeletedTenantCodes clears this state at startup.
	ErrTenantCodeReserved = errors.New("tenant code reserved by a deleted tenant")
	// ErrTenantHasActiveUsers is returned when deleting a tenant that still has active members.
	ErrTenantHasActiveUsers = errors.New("tenant has active users, cannot delete")
	// ErrMembershipExists is returned when adding a user who is already a tenant member.
	ErrMembershipExists = errors.New("user is already a member of this tenant")
	// ErrMembershipNotFound is returned when removing a non-existent tenant membership.
	ErrMembershipNotFound = errors.New("tenant membership not found")
	// ErrLastOwner is returned when a removal would leave the tenant without an
	// active owner (contract §2.2: every non-archived tenant keeps at least one).
	ErrLastOwner = errors.New("tenant membership last owner")
)

const (
	tenantIDPredicate = "tenant_id = ?"
	// activeTenantMemberPredicate is table-qualified on purpose: the member
	// listing joins system_user, and since the tenant-isolation work gave
	// system_user its own tenant_id column, an unqualified `tenant_id` is
	// ambiguous (MySQL 1052) and fails the whole query.
	activeTenantMemberPredicate = "tenant_memberships.tenant_id = ? AND tenant_memberships.status = ?"
)

// tenantStatusDeleted is the soft-delete marker written by DeleteTenant. It is
// not a lifecycle state in contract §2.1, so it never appears in the default
// list and can only be selected through an explicit status filter. The token
// itself lives in pkg/tenant because the credential lookup also has to
// recognise it (see pkg/tenant.TenantStatusDeleted).
const tenantStatusDeleted = tenant.TenantStatusDeleted

// deletedTenantCodePrefix is the archived form a tenant's `code` takes when the
// tenant is deleted, so the code can be used again.
//
// `code` is UNIQUE (contract §2.1), and a soft-deleted row still occupies it at
// the storage layer. Leaving the original value in place would reserve the code
// forever behind a row the console hides by default — the operator sees an
// empty list, types the code, and gets a conflict they cannot explain.
//
// Archiving the identifier is the project-wide rule for soft-delete +
// unique-key entities (docs/designs/BACKEND.md), exactly as
// `system_user.username` (`__deleted_user_`) and `system_role.role_key`
// (`__deleted_role_`) already do. The leading underscore matters: the contract
// restricts codes to `^[a-z][a-z0-9-]{1,62}$`, so an archived value can never
// collide with a code an operator is allowed to type.
const deletedTenantCodePrefix = "__deleted_tenant_"

// platformGlobalTenantID is the reserved platform/global placeholder id
// (contract §2.3), re-exported from pkg/tenant so the service functions that
// already bind a local named `tenant` can still name it unambiguously.
const platformGlobalTenantID = tenant.PlatformGlobalTenantID

// likeEscape neutralizes the LIKE metacharacters a user can type into the
// console search box, so `%` and `_` match literally instead of widening the
// result set. It relies on the backslash being the default LIKE escape
// character, which holds for MySQL — the dialect both production and the test
// suite run on.
func likeEscape(value string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(value)
}

// Service handles tenant CRUD and lifecycle operations
type Service struct {
	db    *gorm.DB
	plans *PlanStore
}

// NewService creates a new tenant service instance
func NewService(db *gorm.DB) *Service {
	return &Service{db: db, plans: NewPlanStore(db)}
}

// Plans exposes the plan definition store so the console surfaces (overview,
// member list, plan editor) resolve limits through the same read-through cache
// the lifecycle validation uses, instead of a second copy of the table.
func (s *Service) Plans() *PlanStore { return s.plans }

// Migrate creates the plan table in AutoMigrate deployments, then runs the
// usual startup repairs. Versioned deployments already have the table from
// migration 000030, so Bootstrap alone is enough there.
func (s *Service) Migrate() error {
	if s.db == nil {
		return common.ErrDatabaseNotInitialized
	}
	if err := s.plans.Migrate(); err != nil {
		return err
	}
	return s.Bootstrap()
}

// CreateTenant creates a new tenant and provisions it.
//
// The tenant row alone is not a usable tenant: without roles, an administrator
// and a membership nobody can log in. ProvisionTenant supplies those, so a
// successful create returns a tenant that is immediately usable (contract §2.1
// lifecycle + §2.2 membership). Provisioning is idempotent, so if it fails the
// tenant row survives and re-running provisioning repairs it.
func (s *Service) CreateTenant(dto CreateTenantDTO) (*Tenant, error) {
	if err := s.validateAdminPassword(dto); err != nil {
		return nil, err
	}
	// The plan code must be a plan the platform actually defines (contract §2.4).
	// Validating here rather than in a binding tag is what lets the console add a
	// plan without a code release, and is what stops a typo from becoming a tenant
	// whose quota silently resolves to the free tier.
	if err := s.plans.ValidateAssignment(dto.Plan); err != nil {
		return nil, err
	}
	dto.Plan = normalizePlanCode(dto.Plan)

	// Reject a code that is already taken. The lookup deliberately spans every
	// status: `code` is a UNIQUE column (contract §2.1, "全局唯一"), so a
	// soft-deleted row still owns its code at the storage layer even though it
	// is invisible in the default console view. Reporting the two situations
	// with the same error is what made a hidden row look like a server fault.
	blocker, err := s.findTenantCodeBlocker(dto.Code)
	if err != nil {
		return nil, err
	}
	if blocker != nil {
		return nil, codeConflictError(blocker)
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
		// The pre-check above cannot be atomic: two callers carrying the same
		// code both pass it, and one of them loses at the UNIQUE index. Answer
		// the loser with the same error the pre-check would have produced,
		// instead of a 500 carrying the driver's message.
		if !database.IsDuplicateKey(err) {
			return nil, fmt.Errorf("create tenant: %w", err)
		}
		blocker, lookupErr := s.findTenantCodeBlocker(dto.Code)
		if lookupErr != nil {
			return nil, lookupErr
		}
		return nil, codeConflictError(blocker)
	}

	if _, err := s.ProvisionTenant(ProvisionInput{
		TenantID:      tenant.ID,
		AdminUsername: dto.AdminUsername,
		AdminPassword: dto.AdminPassword,
		AdminNickname: dto.AdminNickname,
	}); err != nil {
		// Compensate instead of leaving a stranded row. Provisioning runs in one
		// transaction, so a failure leaves no roles, policies or memberships
		// behind — but the tenant row would keep its `code` (a UNIQUE column) and
		// have no owner, which violates contract §2.2 and can only be repaired by
		// hand. Removing the row keeps the code reusable, so the operator's retry
		// is simply "create again".
		if delErr := s.db.Unscoped().Delete(&Tenant{}, tenant.ID).Error; delErr != nil {
			logging.Error("tenant provision failed and the tenant row could not be removed",
				zap.Uint64("tenantId", tenant.ID),
				zap.String("code", tenant.Code),
				zap.Error(err),
				zap.Error(delErr))
			return nil, fmt.Errorf("provision tenant %d (%s): %w (row rollback failed: %v)", tenant.ID, tenant.Code, err, delErr)
		}
		return nil, fmt.Errorf("provision tenant %d (%s): %w", tenant.ID, tenant.Code, err)
	}

	return tenant, nil
}

// findTenantCodeBlocker returns the row that holds `code`, or nil when the code
// is free.
//
// It spans every status on purpose: `code` is UNIQUE at the storage layer, so a
// soft-deleted row still owns its value even though the console hides it. It is
// also the single lookup behind both the pre-check and the post-insert race
// path, so the two cannot drift into reporting different things.
func (s *Service) findTenantCodeBlocker(code string) (*Tenant, error) {
	var blocker Tenant
	err := s.db.Model(&Tenant{}).
		Select("id", "code", "status").
		Where("code = ?", code).
		Limit(1).
		Take(&blocker).Error
	switch {
	case err == nil:
		return &blocker, nil
	case errors.Is(err, gorm.ErrRecordNotFound):
		return nil, nil
	default:
		return nil, fmt.Errorf("check tenant code uniqueness: %w", err)
	}
}

// codeConflictError is the single mapping from "some row holds this code" to
// the error the caller sees. A nil blocker means the row disappeared between
// the failed insert and the read - a concurrent delete released the code - and
// the conflict is still the honest answer, because this caller's insert did
// fail and retrying will succeed.
func codeConflictError(blocker *Tenant) error {
	if blocker != nil && blocker.Status == tenantStatusDeleted {
		return ErrTenantCodeReserved
	}
	return ErrTenantCodeExists
}

// tenantAdminPasswordMinLengthDefault mirrors the seeded default of
// `security.password_min_length` (setting_seed.go) and the platform default
// already used by user creation and password change.
const tenantAdminPasswordMinLengthDefault = 6

// validateAdminPassword enforces the deployment's password policy for the
// tenant administrator.
//
// The minimum comes from `security.password_min_length` rather than a hardcoded
// number, because that single setting governs every other password entry point
// (user creation, password change). Hardcoding it here is what let the console
// form accept a 6-character password that the request binding then rejected
// with a bare `param.invalid` — no field, no reason, no way for the operator to
// know the form and the server disagreed.
//
// The password is only meaningful when an administrator is being provisioned,
// so a tenant created without one is not constrained.
func (s *Service) validateAdminPassword(dto CreateTenantDTO) error {
	if strings.TrimSpace(dto.AdminUsername) == "" {
		return nil
	}
	if len(dto.AdminPassword) < s.configuredPasswordMinLength() {
		// Reuses the key already translated by every console for the same
		// condition, so the operator sees a real sentence instead of a token.
		return common.NewBadRequest("user.update.error.password_too_short")
	}
	return nil
}

// configuredPasswordMinLength reads the platform-wide password policy.
//
// The row is pinned to the platform namespace (tenant_id = 0). The setting is
// tenant-overridable and therefore copied into every tenant, so an unfiltered
// read would resolve to whichever copy the storage engine returned first —
// the same non-determinism that made `platform.tenant_mode` unreliable.
func (s *Service) configuredPasswordMinLength() int {
	if s.db == nil {
		return tenantAdminPasswordMinLengthDefault
	}
	var raw string
	if err := s.db.Table("system_setting").
		Select("setting_value").
		Where("setting_key = ? AND tenant_id = ?", "security.password_min_length", platformGlobalTenantID).
		Limit(1).
		Pluck("setting_value", &raw).Error; err != nil {
		return tenantAdminPasswordMinLengthDefault
	}
	value, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || value <= 0 {
		return tenantAdminPasswordMinLengthDefault
	}
	return value
}

// GetTenantByID retrieves a tenant by id. The reserved platform/global
// placeholder (id 0) is not a tenant (contract §2.3) and reports as missing.
//
// A deleted tenant answers ErrTenantNotFound. `deleted` is a tombstone rather
// than a lifecycle state (contract §2.1, V1.3): the row keeps a synthetic code
// (`__deleted_tenant_<id>`) that no operator can type, so a caller that could
// still address it would be able to revive a tenant nobody can look up by code,
// or add members to it. Deleted rows stay reachable through the explicit
// status-filtered list, which is the operator's audit view.
func (s *Service) GetTenantByID(id uint64) (*Tenant, error) {
	if id == platformGlobalTenantID {
		return nil, ErrTenantNotFound
	}
	var tenant Tenant
	// Use Where clause with placeholder to satisfy SonarCloud taint analysis
	if err := s.db.Where(condIDEquals, id).First(&tenant).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrTenantNotFound
		}
		return nil, fmt.Errorf("get tenant by id: %w", err)
	}
	if tenant.Status == tenantStatusDeleted {
		return nil, ErrTenantNotFound
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

// ListTenants returns a paginated list of tenants.
//
// The reserved platform/global placeholder is never listed (contract §2.3):
// it is not a manageable tenant, and surfacing it is what made the console
// render a bare `__global__` row with zero-value timestamps.
func (s *Service) ListTenants(filter Filter) (*ListResponse, error) {
	// Set defaults
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Size < 1 {
		filter.Size = 20
	}

	base := s.db.Model(&Tenant{}).Where(condIDNotEquals, platformGlobalTenantID)

	if keyword := strings.TrimSpace(filter.Keyword); keyword != "" {
		pattern := "%" + likeEscape(keyword) + "%"
		base = base.Where("code LIKE ? OR name LIKE ?", pattern, pattern)
	}
	if filter.Plan != "" {
		base = base.Where("plan = ?", filter.Plan)
	}

	// Soft-deleted tenants stay out of the default view; an explicit
	// status=deleted filter is the only way to reach them.
	scoped := base.Session(&gorm.Session{})
	if filter.Status != "" {
		scoped = scoped.Where(condStatusEquals, filter.Status)
	} else {
		scoped = scoped.Where(condStatusNotEquals, tenantStatusDeleted)
	}

	// Count total
	var total int64
	if err := scoped.Count(&total).Error; err != nil {
		return nil, fmt.Errorf("count tenants: %w", err)
	}

	summary, err := summarizeTenants(base)
	if err != nil {
		return nil, err
	}

	// Fetch page
	var tenants []Tenant
	offset := (filter.Page - 1) * filter.Size
	if err := scoped.Order("id DESC").Limit(filter.Size).Offset(offset).Find(&tenants).Error; err != nil {
		return nil, fmt.Errorf("list tenants: %w", err)
	}

	return &ListResponse{
		Items:   tenants,
		Total:   total,
		Page:    filter.Page,
		Size:    filter.Size,
		Summary: summary,
	}, nil
}

// summarizeTenants counts tenants per lifecycle status under the given filters
// (which must exclude `status`). Serving the counts from the database is what
// makes the console summary cards consistent with the total instead of being
// derived from whatever rows happen to be on the current page.
func summarizeTenants(base *gorm.DB) (*Summary, error) {
	var rows []struct {
		Status string
		Total  int64
	}
	if err := base.Session(&gorm.Session{}).
		Select("status, COUNT(*) AS total").
		Group("status").
		Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("summarize tenants: %w", err)
	}

	summary := &Summary{}
	for _, row := range rows {
		switch row.Status {
		case tenant.TenantStatusActive:
			summary.Active = row.Total
		case tenant.TenantStatusSuspended:
			summary.Suspended = row.Total
		case tenant.TenantStatusArchived:
			summary.Archived = row.Total
		case tenantStatusDeleted:
			summary.Deleted = row.Total
		}
	}
	summary.Total = summary.Active + summary.Suspended + summary.Archived
	return summary, nil
}

// UpdateTenant updates tenant fields
func (s *Service) UpdateTenant(id uint64, dto UpdateTenantDTO) error {
	row, err := s.GetTenantByID(id)
	if err != nil {
		return err
	}
	previousStatus := row.Status

	// Contract §2.1 declares archived terminal: an archived tenant cannot come
	// back. Validating before the first write keeps a rejected transition from
	// leaving the name/plan half-updated.
	if dto.Status != nil && !tenantStatusTransitionAllowed(previousStatus, *dto.Status) {
		return common.NewConflict("tenant.status.terminal")
	}
	if dto.Plan != nil && *dto.Plan != "" {
		if err := s.plans.ValidateAssignment(*dto.Plan); err != nil {
			return err
		}
	}

	// Update fields individually to satisfy SonarCloud taint analysis
	// This avoids map/struct updates that trigger false positive SQL injection warnings
	hasUpdates := false

	if dto.Name != nil && *dto.Name != "" {
		if err := s.db.Model(row).Update("name", *dto.Name).Error; err != nil {
			return fmt.Errorf("update tenant name: %w", err)
		}
		hasUpdates = true
	}
	if dto.Status != nil && *dto.Status != "" {
		if err := s.db.Model(row).Update("status", *dto.Status).Error; err != nil {
			return fmt.Errorf("update tenant status: %w", err)
		}
		hasUpdates = true
	}
	if dto.Plan != nil && *dto.Plan != "" {
		if err := s.db.Model(row).Update("plan", normalizePlanCode(*dto.Plan)).Error; err != nil {
			return fmt.Errorf("update tenant plan: %w", err)
		}
		hasUpdates = true
	}
	if dto.Metadata != nil {
		if err := s.db.Model(row).Update("metadata", *dto.Metadata).Error; err != nil {
			return fmt.Errorf("update tenant metadata: %w", err)
		}
		hasUpdates = true
	}

	if !hasUpdates {
		return nil // No updates
	}

	// A tenant leaving `active` must not leave its members' sessions running:
	// suspension is only real once the live tokens die. Revocation failures
	// are logged, never rolled back — the status change is already committed
	// and GateSessionRefresh still denies the next refresh as backstop
	// (pkg/tenant authz.go: "Membership changes invalidate affected sessions").
	if dto.Status != nil && statusChangeRequiresSessionRevocation(previousStatus, *dto.Status) {
		s.revokeTenantMemberSessions(id)
	}

	return nil
}

// statusChangeRequiresSessionRevocation is the pure decision half of the
// suspension wiring: only a transition INTO a non-active lifecycle state
// (suspended / archived / deleted marker) kills sessions; reactivation must
// not revoke anything, and a no-op status write is not a transition.
func statusChangeRequiresSessionRevocation(previous, next string) bool {
	next = strings.TrimSpace(next)
	if next == "" || next == previous {
		return false
	}
	return next != tenant.TenantStatusActive
}

// tenantStatusTransitionAllowed enforces contract §2.1:
// active <-> suspended, active|suspended -> archived, and archived is terminal.
// An empty next value is a no-op write, which every state allows; an unknown
// previous value (a legacy row) is left to the enum validation in the DTO.
func tenantStatusTransitionAllowed(previous, next string) bool {
	next = strings.TrimSpace(next)
	previous = strings.TrimSpace(previous)
	if next == "" || next == previous {
		return true
	}
	switch previous {
	case tenant.TenantStatusActive:
		return next == tenant.TenantStatusSuspended || next == tenant.TenantStatusArchived
	case tenant.TenantStatusSuspended:
		return next == tenant.TenantStatusActive || next == tenant.TenantStatusArchived
	case tenant.TenantStatusArchived:
		return false
	case tenantStatusDeleted:
		// A tombstone has no outgoing transition. Without this case the
		// `default` branch below would answer true for it, which is how a
		// deleted tenant could be brought back by a status update.
		return false
	default:
		return true
	}
}

// revokeTenantMemberSessions force-expires the live sessions of every active
// member through the production primitive tenant.RevokeUserSessionsInTenant.
// Best-effort by design: a redis/database hiccup is logged with both ids so
// it stays traceable, and the login/refresh gates keep the tenant locked even
// if the kill did not land.
func (s *Service) revokeTenantMemberSessions(tenantID uint64) {
	var userIDs []uint64
	if err := s.db.Model(&Membership{}).
		Where(activeTenantMemberPredicate, tenantID, "active").
		Pluck("user_id", &userIDs).Error; err != nil {
		logging.Warn("tenant session revocation: member lookup failed",
			zap.Uint64("tenantId", tenantID), zap.Error(err))
		return
	}
	ctx := context.Background()
	for _, userID := range userIDs {
		if _, err := tenant.RevokeUserSessionsInTenant(ctx, database.RDB, s.db, userID); err != nil {
			logging.Warn("tenant session revocation failed",
				zap.Uint64("tenantId", tenantID), zap.Uint64("userId", userID), zap.Error(err))
		}
	}
}

// DeleteTenant soft-deletes a tenant and releases its code.
//
// The reserved platform/global placeholder is not deletable (contract §2.3);
// it owns the compat-mode population, so soft-deleting it would orphan every
// tenant_id = 0 row.
//
// Releasing the code is part of the operation, not a side effect: see
// deletedTenantCodePrefix for why a hidden row must not keep its code.
//
// An archived tenant is deletable even while it still has active members.
// Contract §2.2 only requires an owner on every *non-archived* tenant, and
// §2.1 makes archived terminal, so the remaining memberships are disabled in
// the same transaction instead of blocking the delete. Without this the two
// rules deadlock: archiving is the documented way to retire a tenant, yet
// `RemoveMember` refuses to drop the last owner, so a provisioned tenant could
// never be deleted at all.
func (s *Service) DeleteTenant(id uint64) error {
	if id == platformGlobalTenantID {
		return ErrTenantNotFound
	}

	var row Tenant
	if err := s.db.Select("id", "code", "status").Where(condIDEquals, id).Take(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrTenantNotFound
		}
		return fmt.Errorf("load tenant for delete: %w", err)
	}

	// Check if tenant has active members
	var memberCount int64
	if err := s.db.Model(&Membership{}).
		Where(activeTenantMemberPredicate, id, "active").
		Count(&memberCount).Error; err != nil {
		return fmt.Errorf("check tenant members: %w", err)
	}

	if memberCount > 0 && row.Status != tenant.TenantStatusArchived {
		return ErrTenantHasActiveUsers
	}

	// Mark tenant as deleted, releasing its code first. Every write shares one
	// transaction: a code archived without the status change (or the reverse),
	// or memberships disabled for a tenant that stayed live, would leave the
	// table in a state neither the console nor the repair routine expects.
	return s.db.Transaction(func(tx *gorm.DB) error {
		if memberCount > 0 {
			if err := tx.Model(&Membership{}).
				Where("tenant_id = ? AND status = ?", row.ID, tenant.MembershipActive).
				Update("status", tenant.MembershipDisabled).Error; err != nil {
				return fmt.Errorf("disable tenant memberships: %w", err)
			}
		}

		archivedCode, err := s.allocateDeletedTenantCode(tx, row.ID)
		if err != nil {
			return err
		}
		if err := tx.Model(&Tenant{}).Where(condIDEquals, row.ID).
			Update("code", archivedCode).Error; err != nil {
			return fmt.Errorf("archive tenant code: %w", err)
		}
		if err := tx.Model(&Tenant{}).Where(condIDEquals, row.ID).
			Update("status", tenantStatusDeleted).Error; err != nil {
			return fmt.Errorf("delete tenant: %w", err)
		}
		return nil
	})
}

// Bootstrap runs the tenant table's startup repairs, mirroring
// RoleService.Bootstrap and UserService.Bootstrap: heal the soft-delete
// identifier leftovers before anything else reads the table.
func (s *Service) Bootstrap() error {
	if s.db == nil {
		return common.ErrDatabaseNotInitialized
	}
	// Plan definitions come first: every quota the console reports, and the
	// assignment check on create/update, resolves against this table.
	if err := s.plans.SeedDefaults(); err != nil {
		return err
	}
	if err := s.ReleaseDeletedTenantCodes(); err != nil {
		return err
	}
	if err := s.EnsureTenantRolePolicies(); err != nil {
		return err
	}
	if err := s.EnsureTenantRootDepts(); err != nil {
		return err
	}
	return s.RepairTenantPlatformGrants()
}

// EnsureTenantRootDepts guarantees every active tenant owns a root department,
// so its console can create departments — and therefore posts — at all.
//
// This is the repair half of provisionTenantRootDept, and it exists for the same
// reason EnsureTenantRolePolicies does: a tenant provisioned before root-dept
// provisioning existed (or one whose root was removed out of band) otherwise
// stays permanently unable to create a department. DeptService.CreateDept
// rejects `parent_id = 0` by design ("departments must be placed under the
// organisation root or an existing department"), so with no root there is no
// legal parent and the create form answers `dept.parent.required` forever —
// nothing in the UI can self-heal that, and the post module depends on a
// non-root department existing.
//
// Scope is "active tenants only", matching EnsureTenantRolePolicies: a suspended
// or archived tenant is left untouched, and nothing here ever REMOVES a
// department. Additive and idempotent.
func (s *Service) EnsureTenantRootDepts() error {
	if s.db == nil {
		return common.ErrDatabaseNotInitialized
	}

	var tenantIDs []uint64
	if err := s.db.Model(&Tenant{}).
		Where(condStatusEquals, tenant.TenantStatusActive).
		Where(condIDNotEquals, platformGlobalTenantID).
		Order("id asc").
		Pluck("id", &tenantIDs).Error; err != nil {
		return fmt.Errorf("list active tenants for root dept repair: %w", err)
	}
	if len(tenantIDs) == 0 {
		return nil
	}

	created := 0
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		for _, tenantID := range tenantIDs {
			var existing struct{ ID uint64 }
			if err := tx.Table("system_dept").
				Select("id").
				Where("tenant_id = ? AND is_root = ?", tenantID, common.StatusFlagYes).
				Limit(1).
				Scan(&existing).Error; err != nil {
				return fmt.Errorf("look up root dept for tenant %d: %w", tenantID, err)
			}
			if existing.ID != 0 {
				continue
			}
			if _, err := provisionTenantRootDept(tx, tenantID); err != nil {
				return err
			}
			created++
		}
		return nil
	}); err != nil {
		return err
	}

	if created > 0 {
		logging.Info("tenant root dept repair created missing roots",
			zap.Int("tenants", len(tenantIDs)),
			zap.Int("rootsCreated", created))
	}
	return nil
}

// EnsureTenantRolePolicies re-grants the Casbin domain policies that every
// active tenant's roles must hold (contract §4).
//
// ProvisionTenant writes them, but nothing re-derives them. `casbin_rule` has
// no foreign key to `tenants`, and the sanctioned ops path for hard-deleting a
// tenant purges `role:*@tenant:<id>` rows by hand
// (docs/runbooks/TENANT_TROUBLESHOOTING.md §2b) — so one over-broad purge, or a
// tenant provisioned before this step existed, leaves a tenant whose subjects
// can authenticate but hold no permissions. The console then signs in and every
// page answers `403 permission.denied`, which is indistinguishable from a
// misconfigured role and cannot be self-diagnosed from the UI.
//
// Scope is deliberately "active tenants only". Suspension is the supported way
// to lock a tenant out (the login gate refuses a suspended tenant), so this
// repair must not fight it: a suspended or archived tenant keeps whatever it
// has, and nothing here ever REMOVES a policy. Re-granting is additive and
// idempotent (AddPolicy is a no-op for a rule that already exists), so it
// restores exactly what tenantRoleDefinitions defines and nothing more.
//
// Like ReleaseDeletedTenantCodes this is the repair half of the
// "启动迁移也会自动修复历史遗留数据" rule, and it is a no-op when Casbin has not
// been initialised (unit-test fixtures, or a process that never reached
// InitCasbin).
func (s *Service) EnsureTenantRolePolicies() error {
	if s.db == nil {
		return common.ErrDatabaseNotInitialized
	}

	var tenantIDs []uint64
	if err := s.db.Model(&Tenant{}).
		Where(condStatusEquals, tenant.TenantStatusActive).
		Where(condIDNotEquals, platformGlobalTenantID).
		Order("id asc").
		Pluck("id", &tenantIDs).Error; err != nil {
		return fmt.Errorf("list active tenants for policy repair: %w", err)
	}
	if len(tenantIDs) == 0 {
		return nil
	}

	restored := 0
	for _, tenantID := range tenantIDs {
		restored += grantTenantRolePolicies(s.db, tenantID)
	}
	if restored > 0 {
		logging.Info("tenant policy repair restored missing Casbin domain policies",
			zap.Int("tenants", len(tenantIDs)),
			zap.Int("policiesRestored", restored))
	}
	return nil
}

// ReleaseDeletedTenantCodes archives the `code` of tenants that were soft
// deleted before the identifier-archiving rule reached this table, so those
// codes become usable again.
//
// This is the repair half of docs/designs/BACKEND.md's rule ("启动迁移也会自动
// 修复历史遗留数据"). Without it, any tenant deleted before the fix keeps its
// code reserved forever — which is exactly the trap that made `test-tenant`
// uncreatable while the console reported zero tenants.
//
// The predicate is `status = 'deleted'`, not `deleted_at IS NOT NULL`: this
// table's soft delete is status-based (see tenantStatusDeleted), and the
// `deleted_at` column is unused by the tenant lifecycle.
func (s *Service) ReleaseDeletedTenantCodes() error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		var rows []Tenant
		if err := tx.Unscoped().
			Select("id", "code").
			Where("status = ? AND id <> ?", tenantStatusDeleted, platformGlobalTenantID).
			Find(&rows).Error; err != nil {
			return err
		}

		for _, row := range rows {
			if strings.HasPrefix(row.Code, deletedTenantCodePrefix) {
				continue
			}
			archivedCode, err := s.allocateDeletedTenantCode(tx, row.ID)
			if err != nil {
				return err
			}
			if err := tx.Model(&Tenant{}).Where(condIDEquals, row.ID).
				Update("code", archivedCode).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// allocateDeletedTenantCode builds the unique placeholder that replaces a
// released code. The id keeps it stable and traceable back to the row; the
// nanosecond suffix is a fallback for the pathological case where that exact
// name is already taken.
func (s *Service) allocateDeletedTenantCode(tx *gorm.DB, tenantID uint64) (string, error) {
	for attempt := 0; attempt < 5; attempt++ {
		candidate := fmt.Sprintf("%s%d", deletedTenantCodePrefix, tenantID)
		if attempt > 0 {
			candidate = fmt.Sprintf("%s%d_%d", deletedTenantCodePrefix, tenantID, time.Now().UnixNano())
		}

		var count int64
		if err := tx.Unscoped().Model(&Tenant{}).
			Where("code = ? AND id <> ?", candidate, tenantID).
			Count(&count).Error; err != nil {
			return "", err
		}
		if count == 0 {
			return candidate, nil
		}
	}
	return "", common.NewConflict("tenant.delete.error.archive_code_conflict")
}

// membershipScope returns GORM's bound-parameter condition for one
// (tenant, user) membership row. Both keys are always supplied, so unlike a
// struct condition it cannot silently drop a zero tenant id, and the generated
// SQL keeps the same placeholders as the raw fragment it replaces.
func membershipScope(tenantID, userID uint64) map[string]any {
	return map[string]any{"tenant_id": tenantID, "user_id": userID}
}

// AddMember adds a user to a tenant with specified role.
//
// Re-adding is idempotent for non-owner rows: user create inside a tenant
// viewpoint already inserts a `member` membership server-side
// (user.ensureTenantMembership — the login gate needs it), so the console's
// "join this tenant" submit posts the same pair again right after. Rejecting
// that with member_exists made every in-viewpoint user create report a
// membership failure for a membership that exists. An existing row therefore
// updates its role and re-activates instead. `owner` rows stay protected:
// re-adding owner is a no-op, adding a non-owner role over owner is rejected
// so this endpoint can never demote the tenant owner.
func (s *Service) AddMember(tenantID, userID uint64, role string) error {
	// Verify tenant exists
	if _, err := s.GetTenantByID(tenantID); err != nil {
		return err
	}

	var existing Membership
	// The two NOSONAR marks below suppress gosecurity:S3649 only. tenantID and userID arrive as
	// parsed uint64 (path id via strconv.ParseUint, body via the DTO's required binding) and reach
	// GORM exclusively as bound `?` arguments from membershipScope; nothing is concatenated into SQL
	// text. The analyzer flags any builder call fed from the request regardless of binding, so the
	// rule cannot be satisfied from this code path - suppression is done here on the maintainer's
	// instruction instead of changing the Sonar rule configuration or the Quality Gate.
	memberRow := s.db.Model(&Membership{}).Where(membershipScope(tenantID, userID)) // NOSONAR gosecurity:S3649 - bound placeholders only, no dynamic SQL text
	err := memberRow.Limit(1).First(&existing).Error
	if err == nil {
		if existing.Role == "owner" {
			if role == "owner" {
				return nil
			}
			return ErrMembershipExists
		}
		// Update the row that was just loaded, by its primary key, and name the
		// columns explicitly: nothing request-supplied reaches a column or
		// condition position, and the target row is the one whose role was
		// already checked above. The NOSONAR mark suppresses the same
		// gosecurity:S3649 false positive described above - `role` is a validated
		// enum value bound as a parameter, never SQL text.
		membershipUpdate := s.db.Model(&existing).Select("role", "status").Updates(Membership{Role: role, Status: "active"}) // NOSONAR gosecurity:S3649 - bound parameters only, see the note above
		if err := membershipUpdate.Error; err != nil {
			return fmt.Errorf("update membership: %w", err)
		}
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("check membership existence: %w", err)
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

// RemoveMember removes a user from a tenant.
//
// Contract §2.2 requires every non-archived tenant to keep at least one owner.
// The add path already refuses to demote an owner, so removal is the only way
// to empty the role; a tenant with no owner can no longer be administered from
// its own console and can only be repaired by re-provisioning.
func (s *Service) RemoveMember(tenantID, userID uint64) error {
	if err := s.ensureNotLastOwner(tenantID, userID); err != nil {
		return err
	}

	result := s.db.Where(membershipScope(tenantID, userID)).
		Delete(&Membership{})

	if result.Error != nil {
		return fmt.Errorf("remove membership: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return ErrMembershipNotFound
	}

	// Wiring contract from pkg/tenant authz.go ("Membership changes invalidate
	// affected sessions"): dropping the membership must kill the member's live
	// sessions immediately, not only at the next refresh. Best-effort with a
	// logged failure; GateSessionRefresh remains the backstop.
	if _, err := tenant.RevokeUserSessionsInTenant(context.Background(), database.RDB, s.db, userID); err != nil {
		logging.Warn("tenant member removal session revocation failed",
			zap.Uint64("tenantId", tenantID), zap.Uint64("userId", userID), zap.Error(err))
	}

	return nil
}

// ensureNotLastOwner refuses a removal that would leave the tenant without an
// active owner. Removing a non-owner, or one of several owners, is unaffected;
// a missing row falls through to the delete path so it still answers
// ErrMembershipNotFound once.
func (s *Service) ensureNotLastOwner(tenantID, userID uint64) error {
	var member Membership
	if err := s.db.Model(&Membership{}).
		Where(membershipScope(tenantID, userID)).
		Limit(1).First(&member).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil
		}
		return fmt.Errorf("check membership before removal: %w", err)
	}
	if member.Role != tenant.MembershipRoleOwner || member.Status != tenant.MembershipActive {
		return nil
	}
	var owners int64
	if err := s.db.Model(&Membership{}).
		Where("tenant_id = ? AND role = ? AND status = ?", tenantID, tenant.MembershipRoleOwner, tenant.MembershipActive).
		Count(&owners).Error; err != nil {
		return fmt.Errorf("count tenant owners: %w", err)
	}
	if owners <= 1 {
		return ErrLastOwner
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

// MemberResp enriches a membership row with the member's identity so
// the management console can render names without a second lookup.
type MemberResp struct {
	Membership
	Username string `json:"username"`
	Nickname string `json:"nickname"`
}

// ListTenantMembersWithUser returns one bounded page of active memberships
// joined with system_user identity columns, plus the total member count.
func (s *Service) ListTenantMembersWithUser(tenantID uint64, page, pageSize int) ([]MemberResp, int64, error) {
	page, pageSize = normalizeTenantMemberPagination(page, pageSize)

	scope := s.db.Model(&Membership{}).
		Where(activeTenantMemberPredicate, tenantID, "active")

	var total int64
	if err := scope.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("count tenant members: %w", err)
	}

	items := make([]MemberResp, 0)
	if err := scope.
		Select("tenant_memberships.*, system_user.username AS username, system_user.nickname AS nickname").
		Joins("LEFT JOIN system_user ON system_user.id = tenant_memberships.user_id").
		Order("tenant_memberships.created_at ASC, tenant_memberships.id ASC").
		Limit(pageSize).
		Offset((page - 1) * pageSize).
		Scan(&items).Error; err != nil {
		return nil, 0, fmt.Errorf("list tenant members: %w", err)
	}
	return items, total, nil
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

// InitializeTenantDefaults provisions the resources a tenant needs to be
// usable: its role set, its Casbin policies and its settings copy.
//
// It used to be a documented no-op ("Phase 2.2 default-resource creation has
// not shipped"), which is why a created tenant had no roles and no
// administrator. It now delegates to ProvisionTenant, which is idempotent and
// therefore doubles as a repair entry point for tenants created before
// provisioning existed.
func (s *Service) InitializeTenantDefaults(tenantID uint64) error {
	_, err := s.ProvisionTenant(ProvisionInput{TenantID: tenantID})
	return err
}
