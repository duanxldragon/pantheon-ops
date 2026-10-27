package tenant

import (
	"fmt"
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/pkg/tenant"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// seedReservedPlaceholder inserts the reserved platform/global row the way the
// schema does: id 0, status archived (contract TENANT_CONTRACT_V1 §2.3).
//
// It is a two-step idiom, not a plain insert. MySQL reads an inserted 0 in an
// AUTO_INCREMENT column as "assign the next counter value", so
// `INSERT ... VALUES (0, ...)` would land the row at id 1 and make every
// exclusion assertion below vacuous. The row is therefore inserted normally and
// then adopted onto id 0 — exactly what migration 000013 does when it repairs a
// placeholder that auto-increment displaced.
func seedReservedPlaceholder(t *testing.T, db *gorm.DB) {
	t.Helper()
	err := db.Exec(
		"INSERT INTO tenants (code, name, status, plan) VALUES (?, ?, ?, ?)",
		"__global__", "Platform Global", "archived", "",
	).Error
	require.NoError(t, err)
	err = db.Exec(
		"UPDATE tenants SET id = ? WHERE code = ?",
		tenant.PlatformGlobalTenantID, "__global__",
	).Error
	require.NoError(t, err)
}

func seedTenants(t *testing.T, service *Service, specs []struct{ Code, Name, Plan string }) {
	t.Helper()
	for _, spec := range specs {
		_, err := service.CreateTenant(CreateTenantDTO{Code: spec.Code, Name: spec.Name, Plan: spec.Plan})
		require.NoError(t, err)
	}
}

func codesOf(items []Tenant) []string {
	codes := make([]string, 0, len(items))
	for _, item := range items {
		codes = append(codes, item.Code)
	}
	return codes
}

func TestListTenants_ExcludesReservedPlaceholder(t *testing.T) {
	db := setupTestDB(t)
	seedReservedPlaceholder(t, db)
	service := NewService(db)

	_, err := service.CreateTenant(CreateTenantDTO{Code: "acme", Name: "Acme", Plan: "free"})
	require.NoError(t, err)

	response, err := service.ListTenants(Filter{Page: 1, Size: 20})
	require.NoError(t, err)
	assert.Equal(t, int64(1), response.Total)
	assert.Equal(t, []string{"acme"}, codesOf(response.Items))

	// The reserved row is not addressable either: it is not a tenant.
	_, err = service.GetTenantByID(tenant.PlatformGlobalTenantID)
	assert.Equal(t, ErrTenantNotFound, err)

	// ... and it cannot be soft-deleted, which would orphan every tenant_id = 0 row.
	assert.Equal(t, ErrTenantNotFound, service.DeleteTenant(tenant.PlatformGlobalTenantID))
}

func TestListTenants_KeywordMatchesCodeAndName(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)
	seedTenants(t, service, []struct{ Code, Name, Plan string }{
		{Code: "acme-corp", Name: "Acme Corporation", Plan: "free"},
		{Code: "globex", Name: "Globex Industries", Plan: "free"},
		{Code: "initech", Name: "Initech", Plan: "free"},
	})

	byCode, err := service.ListTenants(Filter{Keyword: "globex"})
	require.NoError(t, err)
	assert.Equal(t, []string{"globex"}, codesOf(byCode.Items))

	// Case-insensitive substring over the name as well.
	byName, err := service.ListTenants(Filter{Keyword: "corporation"})
	require.NoError(t, err)
	assert.Equal(t, []string{"acme-corp"}, codesOf(byName.Items))

	noMatch, err := service.ListTenants(Filter{Keyword: "no-such-tenant"})
	require.NoError(t, err)
	assert.Empty(t, noMatch.Items)
	assert.Equal(t, int64(0), noMatch.Total)
}

func TestListTenants_ArchivedIsFilterable(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)
	created, err := service.CreateTenant(CreateTenantDTO{Code: "legacy", Name: "Legacy", Plan: "free"})
	require.NoError(t, err)

	archived := tenant.TenantStatusArchived
	require.NoError(t, service.UpdateTenant(created.ID, UpdateTenantDTO{Status: &archived}))

	archivedList, err := service.ListTenants(Filter{Status: tenant.TenantStatusArchived})
	require.NoError(t, err)
	assert.Equal(t, []string{"legacy"}, codesOf(archivedList.Items))

	activeList, err := service.ListTenants(Filter{Status: tenant.TenantStatusActive})
	require.NoError(t, err)
	assert.Empty(t, activeList.Items)
}

func TestListTenants_SoftDeletedHiddenUnlessRequested(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)
	created, err := service.CreateTenant(CreateTenantDTO{Code: "doomed", Name: "Doomed", Plan: "free"})
	require.NoError(t, err)
	require.NoError(t, service.DeleteTenant(created.ID))

	defaultView, err := service.ListTenants(Filter{Page: 1, Size: 20})
	require.NoError(t, err)
	assert.Empty(t, defaultView.Items)
	assert.Equal(t, int64(0), defaultView.Total)

	deletedView, err := service.ListTenants(Filter{Status: tenantStatusDeleted})
	require.NoError(t, err)
	// The row is still listed under the deleted filter, but its `code` has been
	// archived so the original value is free again — see
	// deletedTenantCodePrefix. The id is what keeps the row traceable.
	assert.Equal(t, []string{fmt.Sprintf("%s%d", deletedTenantCodePrefix, created.ID)}, codesOf(deletedView.Items))
}

func TestListTenants_SummaryIsPagedIndependently(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)
	seedTenants(t, service, []struct{ Code, Name, Plan string }{
		{Code: "alpha", Name: "Alpha", Plan: "free"},
		{Code: "bravo", Name: "Bravo", Plan: "free"},
		{Code: "charlie", Name: "Charlie", Plan: "free"},
	})

	// Park one tenant in each of the two non-active states.
	var all []Tenant
	require.NoError(t, db.Order("id asc").Find(&all).Error)
	suspended := tenant.TenantStatusSuspended
	archived := tenant.TenantStatusArchived
	require.NoError(t, service.UpdateTenant(all[1].ID, UpdateTenantDTO{Status: &suspended}))
	require.NoError(t, service.UpdateTenant(all[2].ID, UpdateTenantDTO{Status: &archived}))

	// One row per page: the summary must still describe the whole set.
	page, err := service.ListTenants(Filter{Page: 1, Size: 1})
	require.NoError(t, err)
	assert.Len(t, page.Items, 1)
	assert.Equal(t, int64(3), page.Total)
	require.NotNil(t, page.Summary)
	assert.Equal(t, int64(3), page.Summary.Total)
	assert.Equal(t, int64(1), page.Summary.Active)
	assert.Equal(t, int64(1), page.Summary.Suspended)
	assert.Equal(t, int64(1), page.Summary.Archived)
	assert.Equal(t, int64(0), page.Summary.Deleted)

	// Facets ignore the status filter, so they stay meaningful while one status
	// is selected.
	filtered, err := service.ListTenants(Filter{Status: tenant.TenantStatusActive})
	require.NoError(t, err)
	require.NotNil(t, filtered.Summary)
	assert.Equal(t, int64(1), filtered.Total)
	assert.Equal(t, int64(3), filtered.Summary.Total)

	// ... and they follow the keyword filter.
	keyworded, err := service.ListTenants(Filter{Keyword: "charlie"})
	require.NoError(t, err)
	require.NotNil(t, keyworded.Summary)
	assert.Equal(t, int64(1), keyworded.Summary.Total)
	assert.Equal(t, int64(0), keyworded.Summary.Active)
	assert.Equal(t, int64(1), keyworded.Summary.Archived)
}

func TestListTenants_SummaryCountsDeletedSeparately(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)
	created, err := service.CreateTenant(CreateTenantDTO{Code: "gone", Name: "Gone", Plan: "free"})
	require.NoError(t, err)
	require.NoError(t, service.DeleteTenant(created.ID))

	response, err := service.ListTenants(Filter{Page: 1, Size: 20})
	require.NoError(t, err)
	require.NotNil(t, response.Summary)
	// `total` counts what the default view shows; deleted rows are reported
	// separately instead of inflating it.
	assert.Equal(t, int64(0), response.Summary.Total)
	assert.Equal(t, int64(1), response.Summary.Deleted)
}

func TestDeleteTenant_UnknownIDReportsNotFound(t *testing.T) {
	db := setupTestDB(t)
	service := NewService(db)

	assert.Equal(t, ErrTenantNotFound, service.DeleteTenant(4242))
}
