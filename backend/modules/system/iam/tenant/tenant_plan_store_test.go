package tenant

import (
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/pkg/common"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// These tests pin the plan definition table (contract §2.4): the thresholds the
// tenant-ops console reports usage against are operator data now, not the
// compiled-in table. The compiled-in table still matters, because it is what an
// unknown or pre-plan code resolves to, so both halves are asserted here.

func ptrInt(value int) *int {
	return &value
}

func ptrString(value string) *string {
	return &value
}

func planCeilings(maxUsers, maxRoles, maxDepts, maxAPIKeys, maxStorageMB int) PlanQuotaReq {
	return PlanQuotaReq{
		MaxUsers:     ptrInt(maxUsers),
		MaxRoles:     ptrInt(maxRoles),
		MaxDepts:     ptrInt(maxDepts),
		MaxAPIKeys:   ptrInt(maxAPIKeys),
		MaxStorageMB: ptrInt(maxStorageMB),
	}
}

// seedPlanStore builds a store over a fixture database holding exactly the four
// built-in plans: the state every installation reaches on its first boot.
func seedPlanStore(t *testing.T) (*gorm.DB, *PlanStore) {
	t.Helper()
	db := setupTestDB(t)
	store := NewPlanStore(db)
	require.NoError(t, store.SeedDefaults())
	return db, store
}

func mustInsertPlanTenant(t *testing.T, db *gorm.DB, id uint64, code, plan, status string) {
	t.Helper()
	require.NoError(t, db.Create(&Tenant{ID: id, Code: code, Name: code, Plan: plan, Status: status}).Error)
}

// TestPlanStore_SeedDefaultsIsIdempotentAndNeverUndoesTheOperator is what makes
// the seed safe to boot against: a second seed adds nothing, and a threshold the
// operator changed survives it. A seed that rewrote rows would silently undo
// the console on every restart.
func TestPlanStore_SeedDefaultsIsIdempotentAndNeverUndoesTheOperator(t *testing.T) {
	db, store := seedPlanStore(t)

	require.NoError(t, store.SeedDefaults())
	require.NoError(t, store.SeedDefaults())

	rows, err := store.List()
	require.NoError(t, err)
	require.Len(t, rows, 4)
	for _, row := range rows {
		assert.Equal(t, GetDefaultQuota(row.Code), row.AsQuota(), "seed must match the compiled-in table")
	}

	require.NoError(t, db.Model(&Plan{}).Where("code = ?", "free").Update("max_users", 8).Error)
	require.NoError(t, store.SeedDefaults())
	store.Invalidate()

	assert.Equal(t, 8, store.QuotaFor("free").MaxUsers)
}

// TestPlanStore_QuotaForResolvesThroughTheTable pins both halves of resolution:
// a stored row wins, and a code the table does not describe falls back to the
// compiled-in table rather than failing or borrowing another plan's row.
func TestPlanStore_QuotaForResolvesThroughTheTable(t *testing.T) {
	db, store := seedPlanStore(t)

	_, err := store.Create(PlanCreateReq{Code: "startup", Name: "Startup", PlanQuotaReq: planCeilings(3, 2, 1, 1, 512)})
	require.NoError(t, err)
	require.NoError(t, db.Model(&Plan{}).Where("code = ?", "free").Update("max_users", 9).Error)
	store.Invalidate()

	assert.Equal(t, 3, store.QuotaFor("startup").MaxUsers)
	assert.Equal(t, 9, store.QuotaFor("free").MaxUsers)

	// Unknown and unchosen codes are the state every tenant provisioned before
	// this feature is in, so they must not fail. They resolve through the
	// compiled table (free's 5 users), which is also why the edited row's 9 does
	// not leak into them.
	assert.Equal(t, GetDefaultQuota("free").MaxUsers, store.QuotaFor("trial").MaxUsers)
	assert.Equal(t, GetDefaultQuota("free"), store.QuotaFor(""))
}

// TestPlanStore_CacheIsBoundedAndDroppedByWrites is why a threshold change
// reaches every instance without a restart: a write through the store drops the
// snapshot immediately, and a write made outside it is picked up when the
// read-through window is invalidated.
func TestPlanStore_CacheIsBoundedAndDroppedByWrites(t *testing.T) {
	db, store := seedPlanStore(t)

	require.Equal(t, GetDefaultQuota("basic").MaxUsers, store.QuotaFor("basic").MaxUsers)

	require.NoError(t, db.Model(&Plan{}).Where("code = ?", "basic").Update("max_users", 42).Error)
	assert.Equal(t, GetDefaultQuota("basic").MaxUsers, store.QuotaFor("basic").MaxUsers, "the open window still serves the snapshot")

	store.Invalidate()
	assert.Equal(t, 42, store.QuotaFor("basic").MaxUsers)

	_, err := store.Update("basic", PlanUpdateReq{PlanQuotaReq: planCeilings(43, 10, 20, 10, 1000)})
	require.NoError(t, err)
	assert.Equal(t, 43, store.QuotaFor("basic").MaxUsers)
}

// TestPlanStore_CreateRequiresEveryCeiling: a plan with an undefined ceiling
// cannot be reported against, so create rejects a partial block instead of
// storing an unreportable row. 0 is rejected too - the sentinel for "no ceiling"
// is -1, and 0 would make unlimited and exhausted indistinguishable.
func TestPlanStore_CreateRequiresEveryCeiling(t *testing.T) {
	_, store := seedPlanStore(t)

	_, err := store.Create(PlanCreateReq{Code: "half", Name: "Half", PlanQuotaReq: PlanQuotaReq{MaxUsers: ptrInt(5)}})
	assert.ErrorIs(t, err, common.ErrBadRequest)
	assert.ErrorContains(t, err, "tenant.plan.quota_required")

	_, err = store.Create(PlanCreateReq{Code: "zero", Name: "Zero", PlanQuotaReq: planCeilings(0, 1, 1, 1, 1)})
	assert.ErrorIs(t, err, common.ErrBadRequest)
	assert.ErrorContains(t, err, "tenant.plan.quota_invalid")

	created, err := store.Create(PlanCreateReq{
		Code:         "unlimited",
		PlanQuotaReq: planCeilings(UnlimitedQuota, UnlimitedQuota, UnlimitedQuota, UnlimitedQuota, UnlimitedQuota),
	})
	require.NoError(t, err)
	assert.Equal(t, UnlimitedQuota, created.MaxUsers)
	assert.Equal(t, PlanStatusActive, created.Status, "an omitted status is an enabled plan")
	// The console reads a name; a plan created without one is still speakable.
	assert.Equal(t, "unlimited", created.Name)
}

// TestPlanStore_CreateRejectsIllegalCodes pins the code grammar at the write
// boundary: the same value travels inside `tenants.plan` and returns as a query
// parameter, so it is constrained where it is authored.
func TestPlanStore_CreateRejectsIllegalCodes(t *testing.T) {
	_, store := seedPlanStore(t)

	for _, code := range []string{"", "a", "1starter", "start er", "über"} {
		_, err := store.Create(PlanCreateReq{Code: code, PlanQuotaReq: planCeilings(1, 1, 1, 1, 1)})
		assert.ErrorIs(t, err, common.ErrBadRequest, code)
		assert.ErrorContains(t, err, "tenant.plan.code_invalid", code)
	}

	_, err := store.Create(PlanCreateReq{Code: "basic", Name: "Duplicate", PlanQuotaReq: planCeilings(1, 1, 1, 1, 1)})
	assert.ErrorIs(t, err, ErrPlanCodeExists)

	// Mixed case is the same code, not a new one: normalization happens before
	// the uniqueness check.
	_, err = store.Create(PlanCreateReq{Code: "BASIC", Name: "Caps", PlanQuotaReq: planCeilings(1, 1, 1, 1, 1)})
	assert.ErrorIs(t, err, ErrPlanCodeExists)
}

// TestPlanStore_UpdateKeepsOmittedFields: nil means "leave the stored value",
// but a request that touches one ceiling must state all five - the same rule
// the console's editor applies.
func TestPlanStore_UpdateKeepsOmittedFields(t *testing.T) {
	_, store := seedPlanStore(t)
	before := store.QuotaFor("professional")

	updated, err := store.Update("professional", PlanUpdateReq{Sort: ptrInt(99)})
	require.NoError(t, err)
	assert.Equal(t, 99, updated.Sort)
	assert.Equal(t, before, updated.AsQuota())

	_, err = store.Update("professional", PlanUpdateReq{Name: ptrString("   ")})
	assert.ErrorIs(t, err, common.ErrBadRequest)
	assert.ErrorContains(t, err, "tenant.plan.name_required")

	_, err = store.Update("professional", PlanUpdateReq{PlanQuotaReq: PlanQuotaReq{MaxUsers: ptrInt(200)}})
	assert.ErrorIs(t, err, common.ErrBadRequest)
	assert.ErrorContains(t, err, "tenant.plan.quota_required")

	_, err = store.Update("ghost", PlanUpdateReq{Sort: ptrInt(1)})
	assert.ErrorIs(t, err, ErrPlanNotFound)
}

// TestPlanStore_RetiringAnAssignedPlanIsRefused is the guard that keeps the
// usage page honest: a tenant stranded on a retired plan reads as a limit
// nobody chose. A deleted tenant no longer occupies anything, so retirement
// becomes possible once the last one is gone.
func TestPlanStore_RetiringAnAssignedPlanIsRefused(t *testing.T) {
	db, store := seedPlanStore(t)

	_, err := store.Create(PlanCreateReq{Code: "pilot", Name: "Pilot", PlanQuotaReq: planCeilings(2, 2, 2, 2, 2)})
	require.NoError(t, err)
	mustInsertPlanTenant(t, db, 401, "pilot-user", "pilot", "active")

	_, err = store.Update("pilot", PlanUpdateReq{Status: ptrString(PlanStatusDisabled)})
	assert.ErrorIs(t, err, ErrPlanInUse)

	// Refusing to retire does not half-apply the edit.
	current, err := store.Get("pilot")
	require.NoError(t, err)
	assert.Equal(t, PlanStatusActive, current.Status)

	require.NoError(t, db.Model(&Tenant{}).Where(condIDEquals, 401).Update("status", tenantStatusDeleted).Error)
	retired, err := store.Update("pilot", PlanUpdateReq{Status: ptrString(PlanStatusDisabled)})
	require.NoError(t, err)
	assert.Equal(t, PlanStatusDisabled, retired.Status)

	// Retiring a plan changes what may be assigned, not what is reported: the
	// code still resolves while a historical row carries it.
	assert.Equal(t, 2, store.QuotaFor("pilot").MaxUsers)
}

// TestPlanStore_ValidateAssignmentDefersToBuiltInsWhileUnseeded: a database the
// seed has not reached yet is not the operator's mistake, and the four codes the
// previous release accepted must stay assignable or provisioning breaks.
func TestPlanStore_ValidateAssignmentDefersToBuiltInsWhileUnseeded(t *testing.T) {
	db := setupTestDB(t)
	store := NewPlanStore(db)

	require.NoError(t, store.ValidateAssignment(""))
	require.NoError(t, store.ValidateAssignment("free"))
	require.NoError(t, store.ValidateAssignment("  ENTERPRISE "))

	err := store.ValidateAssignment("pilot")
	assert.ErrorIs(t, err, common.ErrBadRequest)
	assert.ErrorContains(t, err, "tenant.plan.unknown")

	require.NoError(t, store.SeedDefaults())
	err = store.ValidateAssignment("pilot")
	assert.ErrorContains(t, err, "tenant.plan.unknown")

	created, err := store.Create(PlanCreateReq{Code: "pilot", PlanQuotaReq: planCeilings(1, 1, 1, 1, 1)})
	require.NoError(t, err)
	require.NoError(t, store.ValidateAssignment(created.Code))

	mustInsertPlanTenant(t, db, 403, "pilot-live", "pilot", "active")
	require.NoError(t, db.Model(&Tenant{}).Where(condIDEquals, 403).Update("status", tenantStatusDeleted).Error)
	_, err = store.Update("pilot", PlanUpdateReq{Status: ptrString(PlanStatusDisabled)})
	require.NoError(t, err)

	// A retired definition stays readable but can no longer be assigned.
	assert.Equal(t, 1, store.QuotaFor("pilot").MaxUsers)
	err = store.ValidateAssignment("pilot")
	assert.ErrorIs(t, err, common.ErrBadRequest)
	assert.ErrorContains(t, err, "tenant.plan.disabled")
}
