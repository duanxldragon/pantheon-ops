package tenant

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/database"

	"gorm.io/gorm"
)

// planCacheTTL bounds how long a process serves plan rows it did not write.
//
// The console reads quotas to render usage, so a threshold an operator changes
// must reach every instance without a restart; the same 5-second read-through
// window pkg/tenant.ModeLoader uses for tenant_mode is the project's answer for
// platform config that is read on every request and written rarely.
const planCacheTTL = 5 * time.Second

var (
	// ErrPlanNotFound is returned when no plan definition matches a code.
	ErrPlanNotFound = errors.New("tenant plan not found")
	// ErrPlanCodeExists is returned when creating a plan whose code is taken.
	ErrPlanCodeExists = errors.New("tenant plan code already exists")
	// ErrPlanInUse is returned when retiring a plan tenants still sit on.
	ErrPlanInUse = errors.New("tenant plan is assigned to tenants")
)

// PlanStore owns the plan definition table: the read-through cache every quota
// consumer uses, the validation that decides which code a tenant may carry, and
// the console's create/update path.
type PlanStore struct {
	db *gorm.DB

	mu       sync.RWMutex
	codes    map[string]Plan
	loadedAt time.Time
	loaded   bool
	now      func() time.Time
}

// NewPlanStore builds the store over the tenant master database.
func NewPlanStore(db *gorm.DB) *PlanStore {
	return &PlanStore{db: db, now: time.Now}
}

// Migrate creates the table in AutoMigrate deployments and guarantees the
// built-in plans exist afterwards. Versioned deployments get the DDL from
// migration 000030; the seed is idempotent either way.
func (s *PlanStore) Migrate() error {
	if s.db == nil {
		return common.ErrDatabaseNotInitialized
	}
	if err := s.db.AutoMigrate(&Plan{}); err != nil {
		return err
	}
	return s.SeedDefaults()
}

// SeedDefaults inserts the four plans the previous hardcoded table described.
//
// It never updates an existing row: once the console can edit a threshold, a
// seed that rewrote values on every boot would silently undo the operator. A
// plan the operator deleted outright is therefore also not resurrected — but
// the four built-in codes stay assignable regardless, because `tenants.plan`
// rows and the legacy fallback still speak them.
func (s *PlanStore) SeedDefaults() error {
	if s.db == nil {
		return common.ErrDatabaseNotInitialized
	}
	for index, code := range builtInPlanCodes {
		quota := GetDefaultQuota(code)
		var count int64
		if err := s.db.Model(&Plan{}).Where("code = ?", code).Count(&count).Error; err != nil {
			return fmt.Errorf("count tenant plans: %w", err)
		}
		if count > 0 {
			continue
		}
		row := Plan{
			Code:         code,
			Name:         defaultPlanName(code),
			MaxUsers:     quota.MaxUsers,
			MaxRoles:     quota.MaxRoles,
			MaxDepts:     quota.MaxDepts,
			MaxAPIKeys:   quota.MaxAPIKeys,
			MaxStorageMB: quota.MaxStorageMB,
			Status:       PlanStatusActive,
			Sort:         (index + 1) * 10,
		}
		if err := s.db.Create(&row).Error; err != nil {
			// Provisioning and a concurrent boot can race on the natural key;
			// the loser observes the row as already present.
			if database.IsDuplicateKey(err) {
				continue
			}
			return fmt.Errorf("seed tenant plan %s: %w", code, err)
		}
	}
	s.Invalidate()
	return nil
}

// Invalidate drops the cached snapshot (after any write, and for tests).
func (s *PlanStore) Invalidate() {
	s.mu.Lock()
	s.loaded = false
	s.codes = nil
	s.mu.Unlock()
}

// snapshot returns the cached plan rows by code, reloading when the TTL or an
// invalidation says so.
func (s *PlanStore) snapshot() (map[string]Plan, error) {
	s.mu.RLock()
	fresh := s.loaded && time.Since(s.loadedAt) < planCacheTTL
	cached := s.codes
	s.mu.RUnlock()
	if fresh {
		return cached, nil
	}

	rows, err := s.load()
	if err != nil {
		return nil, err
	}

	s.mu.Lock()
	s.codes = rows
	s.loaded = true
	s.loadedAt = s.now()
	s.mu.Unlock()
	return rows, nil
}

func (s *PlanStore) load() (map[string]Plan, error) {
	if s.db == nil {
		return nil, common.ErrDatabaseNotInitialized
	}
	var plans []Plan
	if err := s.db.Order("sort ASC, id ASC").Find(&plans).Error; err != nil {
		return nil, fmt.Errorf("load tenant plans: %w", err)
	}
	rows := make(map[string]Plan, len(plans))
	for _, plan := range plans {
		rows[normalizePlanCode(plan.Code)] = plan
	}
	return rows, nil
}

// QuotaFor resolves the limits a plan code carries.
//
// Resolution never fails: an unknown, empty or retired code falls back to the
// built-in table, which is what a tenant whose plan predates this feature
// already got. Returning an error here would turn a reporting page into a
// failure path for data the operator cannot have broken.
func (s *PlanStore) QuotaFor(code string) Quota {
	rows, err := s.snapshot()
	if err != nil {
		return GetDefaultQuota(code)
	}
	plan, ok := rows[normalizePlanCode(code)]
	if !ok {
		return GetDefaultQuota(code)
	}
	return plan.AsQuota()
}

// ValidateAssignment rejects a plan code a tenant may not carry.
//
// Empty is accepted (the column default and every tenant provisioned before
// this feature). When the table is unreadable or has no rows the check defers to
// the legacy built-in set, because a schema the seed has not reached yet is not
// the operator's mistake — rejecting the four codes the previous release
// accepted would break tenant provisioning on an un-seeded database.
func (s *PlanStore) ValidateAssignment(code string) error {
	code = normalizePlanCode(code)
	if code == "" {
		return nil
	}
	if !planCodeAllowed(code) {
		return common.NewBadRequest("tenant.plan.code_invalid")
	}
	rows, err := s.snapshot()
	if err != nil || len(rows) == 0 {
		if isKnownBuiltInPlan(code) {
			return nil
		}
		return common.NewBadRequest("tenant.plan.unknown")
	}
	plan, ok := rows[code]
	if !ok {
		return common.NewBadRequest("tenant.plan.unknown")
	}
	if !plan.IsEnabled() {
		return common.NewBadRequest("tenant.plan.disabled")
	}
	return nil
}

// isKnownBuiltInPlan reports whether the code exists in the compiled-in table,
// including the default branch that maps anything unknown onto free.
func isKnownBuiltInPlan(code string) bool {
	normalized := normalizePlanCode(code)
	for _, builtIn := range builtInPlanCodes {
		if builtIn == normalized {
			return true
		}
	}
	return false
}

// Get returns one plan definition by code.
func (s *PlanStore) Get(code string) (Plan, error) {
	rows, err := s.load()
	if err != nil {
		return Plan{}, err
	}
	plan, ok := rows[normalizePlanCode(code)]
	if !ok {
		return Plan{}, ErrPlanNotFound
	}
	return plan, nil
}

// List returns every definition, disabled ones included, in console order.
func (s *PlanStore) List() ([]Plan, error) {
	if s.db == nil {
		return nil, common.ErrDatabaseNotInitialized
	}
	var plans []Plan
	if err := s.db.Order("sort ASC, id ASC").Find(&plans).Error; err != nil {
		return nil, fmt.Errorf("list tenant plans: %w", err)
	}
	return plans, nil
}

// Create stores a new plan definition.
func (s *PlanStore) Create(req PlanCreateReq) (*Plan, error) {
	if s.db == nil {
		return nil, common.ErrDatabaseNotInitialized
	}
	code := normalizePlanCode(req.Code)
	// `planCodeAllowed` also accepts empty (that is the "no plan chosen" value a
	// tenant row carries), but a definition without a code cannot be assigned or
	// reported against, so the write path requires the pattern.
	if code == "" || !planCodePattern.MatchString(code) {
		return nil, common.NewBadRequest("tenant.plan.code_invalid")
	}
	quota, err := req.resolve(true)
	if err != nil {
		return nil, err
	}
	row := Plan{
		Code:         code,
		Name:         strings.TrimSpace(req.Name),
		MaxUsers:     quota.MaxUsers,
		MaxRoles:     quota.MaxRoles,
		MaxDepts:     quota.MaxDepts,
		MaxAPIKeys:   quota.MaxAPIKeys,
		MaxStorageMB: quota.MaxStorageMB,
		Status:       normalizePlanStatus(req.Status),
		Sort:         req.Sort,
	}
	if row.Name == "" {
		row.Name = code
	}
	if err := s.db.Create(&row).Error; err != nil {
		if database.IsDuplicateKey(err) {
			return nil, ErrPlanCodeExists
		}
		return nil, fmt.Errorf("create tenant plan: %w", err)
	}
	s.Invalidate()
	return &row, nil
}

// Update applies a partial edit to one plan definition.
//
// Retiring a plan (status=disabled) is refused while tenants still sit on it:
// the assignment is what the usage page reports against, and a tenant quietly
// stranded on a retired plan reads as a limit nobody chose.
func (s *PlanStore) Update(code string, req PlanUpdateReq) (*Plan, error) {
	if s.db == nil {
		return nil, common.ErrDatabaseNotInitialized
	}
	code = normalizePlanCode(code)
	current, err := s.Get(code)
	if err != nil {
		return nil, err
	}

	quota, err := req.resolve(false)
	if err != nil {
		return nil, err
	}
	next := current
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return nil, common.NewBadRequest("tenant.plan.name_required")
		}
		next.Name = name
	}
	if req.Sort != nil {
		next.Sort = *req.Sort
	}
	if quota != nil {
		next.MaxUsers = quota.MaxUsers
		next.MaxRoles = quota.MaxRoles
		next.MaxDepts = quota.MaxDepts
		next.MaxAPIKeys = quota.MaxAPIKeys
		next.MaxStorageMB = quota.MaxStorageMB
	}
	if req.Status != nil {
		status := normalizePlanStatus(*req.Status)
		if status == PlanStatusDisabled && current.IsEnabled() {
			inUse, err := s.planInUse(code)
			if err != nil {
				return nil, err
			}
			if inUse {
				return nil, ErrPlanInUse
			}
		}
		next.Status = status
	}

	// One line per statement: gosecurity:S3649 reports at the start of the call
	// chain, and a NOSONAR only suppresses the line it sits on.
	updates := map[string]any{
		"name":           next.Name,
		"max_users":      next.MaxUsers,
		"max_roles":      next.MaxRoles,
		"max_depts":      next.MaxDepts,
		"max_api_keys":   next.MaxAPIKeys,
		"max_storage_mb": next.MaxStorageMB,
		"status":         next.Status,
		"sort":           next.Sort,
	}
	if err := s.db.Model(&Plan{}).Where(condIDEquals, current.ID).Updates(updates).Error; err != nil { // NOSONAR gosecurity:S3649 - the map keys are literals and every value is a bound parameter
		return nil, fmt.Errorf("update tenant plan: %w", err)
	}
	s.Invalidate()
	return &next, nil
}

// planInUse reports whether any non-deleted tenant carries the code.
func (s *PlanStore) planInUse(code string) (bool, error) {
	var count int64
	if err := s.db.Model(&Tenant{}).
		Where(condPlanEquals, code).
		Where(condStatusNotEquals, tenantStatusDeleted).
		Where(condIDNotEquals, platformGlobalTenantID).
		Count(&count).Error; err != nil {
		return false, fmt.Errorf("count tenants on plan: %w", err)
	}
	return count > 0, nil
}

// resolve reads the request's quota block. requireAll is true on create, where
// a missing ceiling would leave the row unreportable; on update a nil field
// means "keep the stored value" and returns a nil quota.
func (r PlanQuotaReq) resolve(requireAll bool) (*Quota, error) {
	if !requireAll && r.MaxUsers == nil && r.MaxRoles == nil && r.MaxDepts == nil &&
		r.MaxAPIKeys == nil && r.MaxStorageMB == nil {
		return nil, nil
	}
	pick := func(value *int) (int, error) {
		if value == nil {
			return 0, common.NewBadRequest("tenant.plan.quota_required")
		}
		if *value != UnlimitedQuota && *value < 1 {
			return 0, common.NewBadRequest("tenant.plan.quota_invalid")
		}
		return *value, nil
	}
	maxUsers, err := pick(r.MaxUsers)
	if err != nil {
		return nil, err
	}
	maxRoles, err := pick(r.MaxRoles)
	if err != nil {
		return nil, err
	}
	maxDepts, err := pick(r.MaxDepts)
	if err != nil {
		return nil, err
	}
	maxAPIKeys, err := pick(r.MaxAPIKeys)
	if err != nil {
		return nil, err
	}
	maxStorage, err := pick(r.MaxStorageMB)
	if err != nil {
		return nil, err
	}
	return &Quota{
		MaxUsers:     maxUsers,
		MaxRoles:     maxRoles,
		MaxDepts:     maxDepts,
		MaxAPIKeys:   maxAPIKeys,
		MaxStorageMB: maxStorage,
	}, nil
}

func defaultPlanName(code string) string {
	switch normalizePlanCode(code) {
	case planCodeFree:
		return "Free"
	case planCodeBasic:
		return "Basic"
	case planCodeProfessional:
		return "Professional"
	case planCodeEnterprise:
		return "Enterprise"
	default:
		return code
	}
}
