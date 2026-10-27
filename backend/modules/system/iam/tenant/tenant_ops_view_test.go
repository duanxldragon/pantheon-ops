package tenant

import (
	"fmt"
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/pkg/tenant"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// These tests pin the tenant-ops read model (contract §2.5). The numbers are the
// product: an operator judges "is this installation healthy" from them, so the
// predicates that decide what counts - the reserved platform placeholder, the
// soft-delete marker, which membership counts as occupancy - are asserted here
// rather than trusted.

func mustInsertMembership(t *testing.T, db *gorm.DB, tenantID, userID uint64, role, status string) {
	t.Helper()
	require.NoError(t, db.Table("tenant_memberships").Create(map[string]any{
		"tenant_id": tenantID,
		"user_id":   userID,
		"role":      role,
		"status":    status,
	}).Error)
}

func mustInsertUser(t *testing.T, db *gorm.DB, id uint64, username, nickname string) {
	t.Helper()
	require.NoError(t, db.Table("system_user").Create(map[string]any{
		"id":       id,
		"username": username,
		"nickname": nickname,
		"password": "not-a-real-hash",
	}).Error)
}

func mustInsertRole(t *testing.T, db *gorm.DB, tenantID uint64, key, name string) {
	t.Helper()
	require.NoError(t, db.Table("system_role").Create(map[string]any{
		"tenant_id": tenantID,
		"role_key":  key,
		"role_name": name,
	}).Error)
}

func mustInsertDept(t *testing.T, db *gorm.DB, tenantID uint64, name string) {
	t.Helper()
	require.NoError(t, db.Table("system_dept").Create(map[string]any{
		"tenant_id": tenantID,
		"dept_name": name,
	}).Error)
}

// seedOpsFixture builds the smallest installation that reaches every branch of
// the read model: tenants on three different plan states (a defined tier, a
// second tier, a code no definition describes), a deleted tenant, a membership
// that no longer counts as occupancy, and the platform's own rows at tenant_id 0
// that must never be reported as some tenant's usage.
func seedOpsFixture(t *testing.T) (*Service, *gorm.DB) {
	t.Helper()
	db := setupTestDB(t)
	store := NewPlanStore(db)
	require.NoError(t, store.SeedDefaults())

	mustInsertPlanTenant(t, db, 101, "acme", "free", "active")
	mustInsertPlanTenant(t, db, 102, "globex", "basic", "suspended")
	mustInsertPlanTenant(t, db, 103, "gone", "free", tenantStatusDeleted)
	mustInsertPlanTenant(t, db, 104, "wildcard", "trial", "active")

	mustInsertUser(t, db, 201, "ada", "Ada")
	mustInsertUser(t, db, 202, "bob", "Bob")
	mustInsertUser(t, db, 203, "cy", "Cy")

	mustInsertMembership(t, db, 101, 201, "owner", tenant.MembershipActive)
	mustInsertMembership(t, db, 101, 202, "member", tenant.MembershipActive)
	mustInsertMembership(t, db, 101, 203, "member", tenant.MembershipDisabled)
	mustInsertMembership(t, db, 102, 201, "admin", tenant.MembershipActive)
	mustInsertMembership(t, db, platformGlobalTenantID, 202, "owner", tenant.MembershipActive)

	mustInsertRole(t, db, 101, "ops", "Ops")
	mustInsertRole(t, db, 101, "auditor", "Auditors")
	mustInsertRole(t, db, platformGlobalTenantID, "platform", "Platform")
	mustInsertDept(t, db, 102, "HQ")
	mustInsertDept(t, db, platformGlobalTenantID, "Platform HQ")

	return NewService(db), db
}

// TestOverview_ScopesToRealTenants is the platform placeholder rule (contract
// §2.3) plus the soft-delete rule: tenant 0 is not a tenant, and a deleted one is
// only reachable through an explicit status filter.
func TestOverview_ScopesToRealTenants(t *testing.T) {
	service, _ := seedOpsFixture(t)

	resp, err := service.Overview(Filter{Page: 1, Size: 20})
	require.NoError(t, err)

	assert.Equal(t, int64(3), resp.Total)
	assert.Equal(t, int64(2), resp.Summary.Active)
	assert.Equal(t, int64(1), resp.Summary.Suspended)
	assert.Equal(t, int64(1), resp.Summary.Deleted)
	assert.Equal(t, int64(3), resp.Summary.Total, "the facets must add up to the list total")

	ids := make([]uint64, 0, len(resp.Items))
	for _, item := range resp.Items {
		ids = append(ids, item.TenantID)
	}
	assert.Equal(t, []uint64{101, 102, 104}, ids)

	deleted, err := service.Overview(Filter{Status: tenantStatusDeleted, Page: 1, Size: 20})
	require.NoError(t, err)
	require.Len(t, deleted.Items, 1)
	assert.Equal(t, uint64(103), deleted.Items[0].TenantID)
}

// TestOverview_ReportsOccupancyAgainstThePlanCeiling is the point of the page:
// usage and limit on one row. A disabled membership is not occupancy, the
// platform's own roles and departments are not any tenant's usage, and the
// ceiling is the stored one so an operator edit shows up without a restart.
func TestOverview_ReportsOccupancyAgainstThePlanCeiling(t *testing.T) {
	service, db := seedOpsFixture(t)

	resp, err := service.Overview(Filter{Page: 1, Size: 20})
	require.NoError(t, err)
	require.Len(t, resp.Items, 3)

	acme := resp.Items[0]
	assert.Equal(t, "acme", acme.Code)
	assert.Equal(t, int64(2), acme.Members)
	assert.Equal(t, int64(2), acme.Roles)
	assert.Equal(t, int64(0), acme.Depts)
	assert.Equal(t, GetDefaultQuota("free").MaxUsers, acme.MaxUsers)
	assert.Equal(t, GetDefaultQuota("free").MaxRoles, acme.MaxRoles)

	globex := resp.Items[1]
	assert.Equal(t, int64(1), globex.Members)
	assert.Equal(t, int64(0), globex.Roles)
	assert.Equal(t, int64(1), globex.Depts)
	assert.Equal(t, GetDefaultQuota("basic").MaxUsers, globex.MaxUsers)

	assert.Equal(t, int64(3), resp.Totals.Members)
	assert.Equal(t, int64(2), resp.Totals.Roles)
	assert.Equal(t, int64(1), resp.Totals.Depts)

	require.NoError(t, db.Model(&Plan{}).Where("code = ?", "free").Update("max_users", 7).Error)
	service.Plans().Invalidate()
	after, err := service.Overview(Filter{Page: 1, Size: 20})
	require.NoError(t, err)
	assert.Equal(t, 7, after.Items[0].MaxUsers)
}

// TestOverview_UnknownPlanIsReportedAsUnmanaged: a code with no definition is not
// the free tier. Filing it there would hide data the operator has to see, so the
// row keeps its own code, carries no definition name, and says `known: false`.
func TestOverview_UnknownPlanIsReportedAsUnmanaged(t *testing.T) {
	service, _ := seedOpsFixture(t)

	resp, err := service.Overview(Filter{Page: 1, Size: 20})
	require.NoError(t, err)

	var wildcard *OverviewTenantRow
	for index := range resp.Items {
		if resp.Items[index].Code == "wildcard" {
			wildcard = &resp.Items[index]
		}
	}
	require.NotNil(t, wildcard)
	assert.Equal(t, "trial", wildcard.Plan)
	assert.Empty(t, wildcard.PlanName)

	var trial *OverviewPlanRow
	for index := range resp.Plans {
		if resp.Plans[index].Plan == "trial" {
			trial = &resp.Plans[index]
		}
	}
	require.NotNil(t, trial)
	assert.False(t, trial.Known)
	assert.Equal(t, int64(1), trial.TenantCount)
	assert.Equal(t, GetDefaultQuota("trial"), trial.Quota, "resolution falls back to the compiled table")

	assert.Equal(t, int64(1), planCount(resp.Plans, "free"), "the deleted tenant is outside the distribution")
}

// planCount reads one tier out of the distribution; -1 means the tier is absent.

func planCount(rows []OverviewPlanRow, code string) int64 {
	for _, row := range rows {
		if row.Plan == code {
			return row.TenantCount
		}
	}
	return -1
}

// TestOverview_Filters: keyword narrows the table and the facets together, plan
// narrows to one tier, and LIKE metacharacters typed into the search box match
// literally instead of widening the result set.
func TestOverview_Filters(t *testing.T) {
	service, _ := seedOpsFixture(t)

	// "a" is in two of the four codes, so it must not read as a wildcard.
	byLetter, err := service.Overview(Filter{Keyword: "a", Page: 1, Size: 20})
	require.NoError(t, err)
	assert.Equal(t, int64(2), byLetter.Total)

	for _, keyword := range []string{"acme", "globex", "wildcard"} {
		resp, err := service.Overview(Filter{Keyword: keyword, Page: 1, Size: 20})
		require.NoError(t, err)
		assert.Equal(t, int64(1), resp.Total, keyword)
		assert.Equal(t, int64(1), resp.Summary.Total, keyword)
	}

	escaped, err := service.Overview(Filter{Keyword: "%", Page: 1, Size: 20})
	require.NoError(t, err)
	assert.Equal(t, int64(0), escaped.Total)

	// The facets apply the plan filter but not the status one, so the deleted
	// tenant shows up as the deleted facet rather than in the table.
	free, err := service.Overview(Filter{Plan: "free", Page: 1, Size: 20})
	require.NoError(t, err)
	assert.Equal(t, int64(1), free.Total)
	assert.Equal(t, int64(1), free.Summary.Total)
	assert.Equal(t, int64(1), free.Summary.Deleted)

	basic, err := service.Overview(Filter{Plan: "basic", Page: 1, Size: 20})
	require.NoError(t, err)
	require.Len(t, basic.Items, 1)
	assert.Equal(t, "globex", basic.Items[0].Code)
}

// TestMemberListing_SpansTenantsWithTheirOwnCeilings is the drill-down's promise:
// every active membership of every real tenant, each row carrying its tenant's
// plan, ceiling and usage. The platform's own membership and a deleted tenant's
// members are not part of it.
func TestMemberListing_SpansTenantsWithTheirOwnCeilings(t *testing.T) {
	service, _ := seedOpsFixture(t)

	resp, err := service.ListMembersAcrossPages(MemberRowFilter{Page: 1, PageSize: 20})
	require.NoError(t, err)
	require.Equal(t, int64(3), resp.Total)

	for _, row := range resp.Items {
		assert.NotEqual(t, platformGlobalTenantID, row.TenantID)
		assert.NotEmpty(t, row.Username)
		assert.NotEmpty(t, row.JoinedAt)
		switch row.TenantID {
		case 101:
			assert.Equal(t, "acme", row.TenantCode)
			assert.Equal(t, "free", row.Plan)
			assert.Equal(t, GetDefaultQuota("free").MaxUsers, row.MaxUsers)
			assert.Equal(t, int64(2), row.TenantUsage)
		case 102:
			assert.Equal(t, "globex", row.TenantCode)
			assert.Equal(t, GetDefaultQuota("basic").MaxUsers, row.MaxUsers)
			assert.Equal(t, int64(1), row.TenantUsage)
		default:
			t.Fatalf("unexpected tenant in the member listing: %d", row.TenantID)
		}
	}

	oneTenant, err := service.ListMembersAcrossPages(MemberRowFilter{TenantID: 101, Page: 1, PageSize: 20})
	require.NoError(t, err)
	require.Equal(t, int64(2), oneTenant.Total)
	for _, row := range oneTenant.Items {
		assert.Equal(t, uint64(101), row.TenantID)
	}

	owners, err := service.ListMembersAcrossPages(MemberRowFilter{Role: "owner", Page: 1, PageSize: 20})
	require.NoError(t, err)
	require.Equal(t, int64(1), owners.Total)
	assert.Equal(t, uint64(101), owners.Items[0].TenantID)

	disabled, err := service.ListMembersAcrossPages(
		MemberRowFilter{Status: tenant.MembershipDisabled, Page: 1, PageSize: 20})
	require.NoError(t, err)
	require.Equal(t, int64(1), disabled.Total)
	assert.Equal(t, uint64(203), disabled.Items[0].UserID)

	// One account, two tenants: the row key the console needs is the pair, not the
	// user id.
	byUser, err := service.ListMembersAcrossPages(MemberRowFilter{Keyword: "ada", Page: 1, PageSize: 20})
	require.NoError(t, err)
	assert.Equal(t, int64(2), byUser.Total)

	byTenant, err := service.ListMembersAcrossPages(MemberRowFilter{Keyword: "globex", Page: 1, PageSize: 20})
	require.NoError(t, err)
	assert.Equal(t, int64(1), byTenant.Total)
}

// TestMemberListing_PaginationIsStableAndBounded: the page-two link an operator
// pastes has to show rows the page-one link did not, and a hostile page size is
// clamped rather than trusted.
func TestMemberListing_PaginationIsStableAndBounded(t *testing.T) {
	service, db := seedOpsFixture(t)
	for index, id := range []uint64{301, 302, 303, 304} {
		name := fmt.Sprintf("member%d", index)
		mustInsertUser(t, db, id, name, name)
		mustInsertMembership(t, db, 101, id, "member", tenant.MembershipActive)
	}

	first, err := service.ListMembersAcrossPages(MemberRowFilter{Page: 1, PageSize: 2})
	require.NoError(t, err)
	require.Len(t, first.Items, 2)
	assert.Equal(t, int64(7), first.Total)

	second, err := service.ListMembersAcrossPages(MemberRowFilter{Page: 2, PageSize: 2})
	require.NoError(t, err)
	require.Len(t, second.Items, 2)
	assert.NotEqual(t, first.Items[0].UserID, second.Items[0].UserID)

	oversized, err := service.ListMembersAcrossPages(MemberRowFilter{Page: 1, PageSize: 5000})
	require.NoError(t, err)
	assert.Equal(t, maxTenantMemberPageSize, oversized.PageSize)
}

// TestListPlans_CountsTenantsPerPlan is the number behind the retire action: only
// a plan with no live tenant on it can be retired, and a code nobody defines is
// not a plan row at all.
func TestListPlans_CountsTenantsPerPlan(t *testing.T) {
	service, db := seedOpsFixture(t)

	counts := planCounts(t, service)
	assert.Equal(t, int64(1), counts["free"], "the deleted tenant does not occupy its plan")
	assert.Equal(t, int64(1), counts["basic"])
	assert.Equal(t, int64(0), counts["professional"])
	assert.Equal(t, int64(0), counts["enterprise"])

	require.NoError(t, db.Exec("UPDATE tenants SET plan = 'unheardof' WHERE id = 102").Error)

	after := planCounts(t, service)
	assert.Equal(t, int64(0), after["basic"], "the tenant left the plan, so its count dropped")
	assert.NotContains(t, after, "unheardof", "an undefined code is a distribution row, not a plan")
}

func planCounts(t *testing.T, service *Service) map[string]int64 {
	t.Helper()
	list, err := service.ListPlans()
	require.NoError(t, err)
	counts := make(map[string]int64, len(list))
	for _, row := range list {
		counts[row.Code] = row.TenantCount
	}
	return counts
}

// TestOverview_PageSizeIsClamped: the rollup counts the whole installation, so an
// unbounded page size would turn one request into a table scan.
func TestOverview_PageSizeIsClamped(t *testing.T) {
	service, db := seedOpsFixture(t)
	for index := 0; index < 3; index++ {
		mustInsertPlanTenant(t, db, uint64(500+index), fmt.Sprintf("bulk%d", index), "free", "active")
	}

	resp, err := service.Overview(Filter{Page: 1, Size: 5000})
	require.NoError(t, err)
	assert.Equal(t, maxTenantOverviewSize, resp.Size)
	assert.Len(t, resp.Items, 6, "the clamp limits the page, and every tenant fits")
	assert.Equal(t, int64(6), resp.Total, "the total is the installation, not the page")

	bare, err := service.Overview(Filter{})
	require.NoError(t, err)
	assert.Equal(t, defaultTenantOverviewSize, bare.Size)
	assert.Equal(t, 1, bare.Page)
}
