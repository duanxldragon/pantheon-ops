package tenant

import (
	"fmt"
	"sync"
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/pkg/testmysql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestConcurrentProvisioningKeepsTenantResourcesIsolated turns the last open
// delivery residual into a checked claim. Two tenants created at the same time
// were previously described as "structurally safe but unverified": the unique
// key on `code` prevents duplicate tenants, and every provisioning write is
// keyed by tenant_id, but nothing exercised it.
//
// The invariant that matters is not that both creations succeed. Provisioning
// writes roles, role-menu grants, role-permission grants, a root department, an
// administrator, copied settings and copied dictionaries, all keyed by
// tenant_id. A path that forgot the tenant predicate would still create both
// tenants successfully; it would just give one of them the other's rows, or
// let one transaction overwrite the other's grants. So the assertions below are
// about ownership and disjointness, not about error codes.
//
// The admin username is deliberately identical in every tenant: `(tenant_id,
// username)` is the unique key, so a dropped tenant predicate collides instead
// of quietly succeeding.
func TestConcurrentProvisioningKeepsTenantResourcesIsolated(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)

	const (
		tenantCount   = 4
		adminUsername = "owner"
		adminPassword = "abc12345"
	)

	// One platform-level setting to copy. Without a source row the copy step is
	// a no-op and "0 settings" would be correct, so the assertion below would
	// prove nothing about whether the copy is tenant-scoped.
	require.NoError(t, db.Exec(
		`INSERT INTO system_setting (tenant_id, setting_key, setting_value, group_key, module, value_type)
		 VALUES (0, 'site.name', 'Pantheon', 'site', 'system', 'string')`,
	).Error)

	var (
		wg      sync.WaitGroup
		start   = make(chan struct{})
		created = make([]*Tenant, tenantCount)
		errs    = make([]error, tenantCount)
	)
	for i := 0; i < tenantCount; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start // release every goroutine as close together as possible
			created[i], errs[i] = service.CreateTenant(CreateTenantDTO{
				Code:          fmt.Sprintf("concurrent-%d", i),
				Name:          fmt.Sprintf("Concurrent %d", i),
				AdminUsername: adminUsername,
				AdminPassword: adminPassword,
			})
		}(i)
	}
	close(start)
	wg.Wait()

	for i, err := range errs {
		require.NoErrorf(t, err, "tenant %d must be created under concurrency", i)
	}

	roleOwner := make(map[uint64]uint64, tenantCount*len(tenantRoleDefinitions))
	for i, row := range created {
		require.NotNilf(t, row, "tenant %d is nil", i)

		var roles []struct {
			ID       uint64 `gorm:"column:id"`
			TenantID uint64 `gorm:"column:tenant_id"`
			RoleKey  string `gorm:"column:role_key"`
		}
		require.NoError(t, db.Table("system_role").
			Select("id, tenant_id, role_key").
			Where("tenant_id = ?", row.ID).
			Scan(&roles).Error)
		require.Len(t, roles, len(tenantRoleDefinitions),
			"tenant %d must own exactly its own role set, got %d", i, len(roles))
		for _, role := range roles {
			require.Equalf(t, row.ID, role.TenantID, "role %s leaked to another tenant", role.RoleKey)
			previous, taken := roleOwner[role.ID]
			require.Falsef(t, taken, "role id %d is shared by tenant %d and tenant %d", role.ID, previous, row.ID)
			roleOwner[role.ID] = row.ID
		}

		// Exactly one active membership: the administrator this creation made.
		var memberships []struct {
			UserID uint64 `gorm:"column:user_id"`
			Status string `gorm:"column:status"`
		}
		require.NoError(t, db.Model(&Membership{}).
			Select("user_id, status").
			Where("tenant_id = ?", row.ID).
			Scan(&memberships).Error)
		require.Len(t, memberships, 1, "tenant %d must have exactly one membership", i)
		assert.Equal(t, "active", memberships[0].Status)

		// The administrator belongs to this tenant, and the shared username did
		// not stop a second tenant from getting its own row.
		var admins []struct {
			ID       uint64 `gorm:"column:id"`
			TenantID uint64 `gorm:"column:tenant_id"`
		}
		require.NoError(t, db.Table("system_user").
			Select("id, tenant_id").
			Where("tenant_id = ? AND username = ?", row.ID, adminUsername).
			Scan(&admins).Error)
		require.Len(t, admins, 1, "tenant %d must own exactly one %q", i, adminUsername)
		assert.Equal(t, row.ID, admins[0].TenantID)
		assert.Equal(t, memberships[0].UserID, admins[0].ID, "the membership must point at this tenant's administrator")

		// One root department, not zero (provisioning skipped) and not two
		// (both transactions provisioned into one tenant).
		var rootDepts int64
		require.NoError(t, db.Table("system_dept").
			Where("tenant_id = ? AND is_root = ?", row.ID, 1).
			Count(&rootDepts).Error)
		assert.Equal(t, int64(1), rootDepts, "tenant %d must have exactly one root department", i)

		// The platform settings were copied into this tenant, with its own
		// tenant_id - not appended to the platform namespace and not written
		// into a sibling's.
		var copied []struct {
			TenantID uint64 `gorm:"column:tenant_id"`
			Value    string `gorm:"column:setting_value"`
		}
		require.NoError(t, db.Table("system_setting").
			Select("tenant_id, setting_value").
			Where("setting_key = ?", "site.name").
			Scan(&copied).Error)
		require.Len(t, copied, tenantCount+1,
			"expected the platform row plus one copy per tenant, got %d", len(copied))
		ownCopies := 0
		for _, setting := range copied {
			if setting.TenantID == 0 {
				assert.Equal(t, "Pantheon", setting.Value, "the platform row must keep its value")
				continue
			}
			if setting.TenantID == row.ID {
				ownCopies++
			}
		}
		assert.Equal(t, 1, ownCopies, "tenant %d must receive exactly one copy of the platform setting", i)
	}

	// No role row may exist outside the tenants this test created, and the
	// total must be exactly tenantCount full role sets - a tenant that
	// provisioned twice, or one that picked up a sibling's roles, changes this
	// number.
	var totalRoles int64
	require.NoError(t, db.Table("system_role").Count(&totalRoles).Error)
	assert.Equal(t, int64(tenantCount*len(tenantRoleDefinitions)), totalRoles)
}

// TestConcurrentProvisioningOfTheSameTenantIsConvergent covers the residual
// that same-tenant provisioning was still read-then-insert: roles were looked up
// and inserted without an arbiter, so concurrent callers collided on
// uk_system_role_tenant_key (and, one step later, on the root department, which
// has no unique key at all). The post-commit copies are the other half: they
// decide what to insert from a tenant-side existence check, `system_setting`
// and `system_dict_type` would turn a lost race into a duplicate-key 500, and
// `system_dict_item` has no unique key at all, so a lost race silently doubles
// every item.
//
// The scenario is a repair entry point running twice at once — an operator
// re-running provisioning while a startup repair does the same. Every caller
// must succeed and converge on one role set, one administrator, one binding,
// one membership, one root department and one copy of each platform default.
func TestConcurrentProvisioningOfTheSameTenantIsConvergent(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)

	tenantRow, err := service.CreateTenant(CreateTenantDTO{Code: "same-tenant", Name: "Same Tenant"})
	require.NoError(t, err)

	// Platform defaults the copy step reads. Without source rows every copy is
	// a no-op and contention on the read-then-insert copy would be invisible;
	// the volume keeps the window wide enough for the contenders to overlap.
	const seedRows = 20
	settings := make([]map[string]any, 0, seedRows)
	for i := 0; i < seedRows; i++ {
		settings = append(settings, map[string]any{
			"tenant_id":     0,
			"setting_key":   fmt.Sprintf("seed.setting.%02d", i),
			"setting_value": "platform",
			"group_key":     "seed",
			"module":        "system",
			"value_type":    "string",
		})
	}
	require.NoError(t, db.Table("system_setting").Create(&settings).Error)

	require.NoError(t, db.Exec(
		`INSERT INTO system_dict_type (tenant_id, dict_code, dict_name, module, status)
		 VALUES (0, 'seed_dict', 'Seed', 'system', 1)`).Error)
	items := make([]map[string]any, 0, seedRows)
	for i := 0; i < seedRows; i++ {
		items = append(items, map[string]any{
			"tenant_id":      0,
			"dict_code":      "seed_dict",
			"item_label_key": fmt.Sprintf("dict.seed_dict.%02d", i),
			"item_value":     fmt.Sprintf("%02d", i),
			"sort":           i,
			"status":         1,
		})
	}
	require.NoError(t, db.Table("system_dict_item").Create(&items).Error)

	const (
		contenders    = 4
		adminUsername = "owner"
		adminPassword = "abc12345"
	)

	var (
		wg      sync.WaitGroup
		start   = make(chan struct{})
		results = make([]*ProvisionResult, contenders)
		errs    = make([]error, contenders)
	)
	for i := 0; i < contenders; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			results[i], errs[i] = service.ProvisionTenant(ProvisionInput{
				TenantID:      tenantRow.ID,
				AdminUsername: adminUsername,
				AdminPassword: adminPassword,
			})
		}(i)
	}
	close(start)
	wg.Wait()

	for i, err := range errs {
		require.NoErrorf(t, err, "contender %d must converge instead of colliding", i)
	}
	for i := 1; i < contenders; i++ {
		assert.Equalf(t, results[0].RoleIDs, results[i].RoleIDs,
			"contender %d resolved a different role set", i)
	}

	var roles []struct {
		ID      uint64 `gorm:"column:id"`
		RoleKey string `gorm:"column:role_key"`
	}
	require.NoError(t, db.Table("system_role").
		Select("id, role_key").
		Where("tenant_id = ?", tenantRow.ID).
		Scan(&roles).Error)
	require.Len(t, roles, len(tenantRoleDefinitions), "the tenant must own exactly one role per definition")
	roleIDs := make(map[string]uint64, len(roles))
	for _, role := range roles {
		roleIDs[role.RoleKey] = role.ID
	}
	for _, def := range tenantRoleDefinitions {
		assert.NotZerof(t, roleIDs[def.Key], "role %s is missing", def.Key)
	}

	var admins []struct {
		ID uint64 `gorm:"column:id"`
	}
	require.NoError(t, db.Table("system_user").
		Select("id").
		Where("tenant_id = ? AND username = ?", tenantRow.ID, adminUsername).
		Scan(&admins).Error)
	require.Len(t, admins, 1, "the administrator must exist exactly once")

	var bindings int64
	require.NoError(t, db.Table("system_user_role").
		Where("tenant_id = ? AND user_id = ?", tenantRow.ID, admins[0].ID).
		Count(&bindings).Error)
	assert.Equal(t, int64(1), bindings, "the administrator must hold exactly one role binding")

	var memberships int64
	require.NoError(t, db.Table("tenant_memberships").
		Where("tenant_id = ? AND user_id = ?", tenantRow.ID, admins[0].ID).
		Count(&memberships).Error)
	assert.Equal(t, int64(1), memberships, "the owner membership must exist exactly once")

	var rootDepts int64
	require.NoError(t, db.Table("system_dept").
		Where("tenant_id = ? AND is_root = ?", tenantRow.ID, 1).
		Count(&rootDepts).Error)
	assert.Equal(t, int64(1), rootDepts, "the tenant must own exactly one root department")

	var settingCopies, dictTypeCopies, dictItemCopies int64
	require.NoError(t, db.Table("system_setting").
		Where("tenant_id = ? AND setting_key LIKE ?", tenantRow.ID, "seed.setting.%").
		Count(&settingCopies).Error)
	assert.Equal(t, int64(seedRows), settingCopies, "the platform settings must be copied exactly once")
	require.NoError(t, db.Table("system_dict_type").
		Where("tenant_id = ? AND dict_code = ?", tenantRow.ID, "seed_dict").
		Count(&dictTypeCopies).Error)
	assert.Equal(t, int64(1), dictTypeCopies, "the platform dict type must be copied exactly once")
	require.NoError(t, db.Table("system_dict_item").
		Where("tenant_id = ? AND dict_code = ?", tenantRow.ID, "seed_dict").
		Count(&dictItemCopies).Error)
	assert.Equal(t, int64(seedRows), dictItemCopies,
		"dict items have no unique key, so only the serialisation keeps the copy single")

	// Exactly one contender may have done the copying; the others must observe
	// its committed rows and copy nothing.
	var copiedSettings, copiedTypes, copiedItems int64
	for i, result := range results {
		require.NotNilf(t, result, "contender %d returned no result", i)
		copiedSettings += result.SettingsCopied
		copiedTypes += result.DictTypesCopied
		copiedItems += result.DictItemsCopied
	}
	assert.Equal(t, int64(seedRows), copiedSettings, "the settings copy must run once across all contenders")
	assert.Equal(t, int64(1), copiedTypes, "the dict type copy must run once across all contenders")
	assert.Equal(t, int64(seedRows), copiedItems, "the dict item copy must run once across all contenders")
}

// TestCopyPlatformDefaultsRunOnTheLockedTenantRow pins the half of the copy
// serialisation that timing cannot prove: the copies must execute on the
// caller's transaction, which is the one holding the tenant row lock. If they
// fell back to the pool they would read and insert outside the lock, and the
// dict items — which have no unique key — would be duplicated by a concurrent
// caller even though the settings and dict type rows would merely conflict.
//
// The proof is two-sided: while the copy transaction is open another connection
// cannot take the tenant row (NOWAIT, MySQL 3572), and rolling the transaction
// back leaves no copied rows, so the writes really went through tx.
func TestCopyPlatformDefaultsRunOnTheLockedTenantRow(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)

	tenantRow, err := service.CreateTenant(CreateTenantDTO{Code: "locked-copy", Name: "Locked Copy"})
	require.NoError(t, err)

	require.NoError(t, db.Exec(
		`INSERT INTO system_setting (tenant_id, setting_key, setting_value, group_key, module, value_type)
		 VALUES (0, 'site.name', 'Pantheon', 'site', 'system', 'string')`).Error)
	require.NoError(t, db.Exec(
		`INSERT INTO system_dict_type (tenant_id, dict_code, dict_name, module, status)
		 VALUES (0, 'seed_dict', 'Seed', 'system', 1)`).Error)
	require.NoError(t, db.Exec(
		`INSERT INTO system_dict_item (tenant_id, dict_code, item_label_key, item_value, sort, status)
		 VALUES (0, 'seed_dict', 'dict.seed_dict.yes', '1', 1, 1)`).Error)

	tx := db.Begin()
	require.NoError(t, tx.Error)
	t.Cleanup(func() { _ = tx.Rollback() })

	require.NoError(t, lockTenantRow(tx, tenantRow.ID))
	settings, err := service.copyPlatformSettingsToTenant(tx, tenantRow.ID)
	require.NoError(t, err)
	require.Equal(t, int64(1), settings)
	dictTypes, dictItems, err := service.copyPlatformDictsToTenant(tx, tenantRow.ID)
	require.NoError(t, err)
	require.Equal(t, int64(1), dictTypes)
	require.Equal(t, int64(1), dictItems)

	testmysql.AssertRowLocked(t, db, "tenants", tenantRow.ID)

	require.NoError(t, tx.Rollback().Error)

	var copiedSettingsRows, copiedDictRows int64
	require.NoError(t, db.Table("system_setting").
		Where("tenant_id = ?", tenantRow.ID).
		Count(&copiedSettingsRows).Error)
	assert.Zero(t, copiedSettingsRows, "the settings copy must write through tx, not the pool")
	require.NoError(t, db.Table("system_dict_item").
		Where("tenant_id = ?", tenantRow.ID).
		Count(&copiedDictRows).Error)
	assert.Zero(t, copiedDictRows, "the dict copy must write through tx, not the pool")
}

// TestConcurrentProvisioningOfTheSameCodeCreatesOneTenant is the other half:
// serialisation is not required for correctness, because `code` is UNIQUE. One
// caller wins and the rest must receive a conflict rather than a second tenant
// or a 500.
func TestConcurrentProvisioningOfTheSameCodeCreatesOneTenant(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)

	const contenders = 4
	var (
		wg      sync.WaitGroup
		start   = make(chan struct{})
		errs    = make([]error, contenders)
		created = make([]*Tenant, contenders)
	)
	for i := 0; i < contenders; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			created[i], errs[i] = service.CreateTenant(CreateTenantDTO{Code: "race-co", Name: "Race Co"})
		}(i)
	}
	close(start)
	wg.Wait()

	succeeded := 0
	for i, err := range errs {
		if err == nil {
			succeeded++
			require.NotNil(t, created[i])
			continue
		}
		// Either the pre-check saw the winner, or the UNIQUE index rejected the
		// insert. Both are contract-shaped conflicts; a 500 or a raw driver
		// error would not be.
		assert.ErrorIsf(t, err, ErrTenantCodeExists, "contender %d must fail as a code conflict, got %v", i, err)
	}
	assert.Equal(t, 1, succeeded, "exactly one contender may create the tenant")

	var rows int64
	require.NoError(t, db.Model(&Tenant{}).Where("code = ?", "race-co").Count(&rows).Error)
	assert.Equal(t, int64(1), rows, "the unique index must leave exactly one tenant row")
}
