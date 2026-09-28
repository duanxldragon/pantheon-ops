package config

import (
	"errors"
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/tenant"
	"github.com/duanxldragon/pantheon-base/backend/pkg/testmysql"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// tenantFixture is the two-tenant canary fixture (task packet: "两个租户 fixture").
// Tenant 0 rows are platform/global (compat population). Tenants 101 and 202
// own same-code dicts to prove tenant-local uniqueness and isolation.
type tenantFixture struct {
	db       *gorm.DB
	service  *DictService
	ctxA     *tenant.Context // tenant 101
	ctxB     *tenant.Context // tenant 202
	ctxWorld *tenant.Context // compat/global
}

func newTenantFixture(t *testing.T) *tenantFixture {
	t.Helper()
	db := testmysql.Open(t)
	if err := db.AutoMigrate(&SystemDictType{}, &SystemDictItem{}, &tenant.Membership{}); err != nil {
		t.Fatalf("migrate dict: %v", err)
	}

	f := &tenantFixture{
		db:       db,
		service:  NewDictService(db),
		ctxA:     &tenant.Context{TenantID: 101, Mode: tenant.ModeMulti, ResolvedBy: "subject"},
		ctxB:     &tenant.Context{TenantID: 202, Mode: tenant.ModeMulti, ResolvedBy: "subject"},
		ctxWorld: &tenant.Context{TenantID: tenant.PlatformGlobalTenantID, Mode: tenant.ModeCompat, ResolvedBy: "compat-fallback"},
	}

	// Global seed dict (compat population, tenant 0).
	if _, err := f.service.CreateDictType(&DictTypeCreateReq{DictCode: "shared_code", DictName: "global-shared", Status: 1}); err != nil {
		t.Fatalf("seed global dict: %v", err)
	}
	if _, err := f.service.CreateDictItem(&DictItemCreateReq{DictCode: "shared_code", ItemLabelKey: "label.global.1", ItemValue: "global-1", Status: 1}); err != nil {
		t.Fatalf("seed global item: %v", err)
	}
	return f
}

func (f *tenantFixture) svcA() *DictService { return f.service.WithTenantContext(f.ctxA) }
func (f *tenantFixture) svcB() *DictService { return f.service.WithTenantContext(f.ctxB) }

// isNotFound asserts unreachability of the row. The service layer returns
// either the common.ErrNotFound classification or gorm's raw record-not-found
// depending on the path; both mean "not visible to the caller" (contract §5:
// no existence leak, no new error code).
func isNotFound(err error) bool {
	return errors.Is(err, common.ErrNotFound) || errors.Is(err, gorm.ErrRecordNotFound)
}

func mustCreateType(t *testing.T, s *DictService, code, name string) *DictTypeResp {
	t.Helper()
	row, err := s.CreateDictType(&DictTypeCreateReq{DictCode: code, DictName: name, Status: 1})
	if err != nil {
		t.Fatalf("create dict type %s: %v", code, err)
	}
	return row
}

func mustCreateItem(t *testing.T, s *DictService, code, value string) *DictItemResp {
	t.Helper()
	row, err := s.CreateDictItem(&DictItemCreateReq{DictCode: code, ItemLabelKey: "label." + value, ItemValue: value, Status: 1})
	if err != nil {
		t.Fatalf("create dict item %s/%s: %v", code, value, err)
	}
	return row
}

// 1. Two tenants may own the same dict_code (tenant-local uniqueness).
func TestTenantCanary_SameCodeCoexistence(t *testing.T) {
	f := newTenantFixture(t)

	a := mustCreateType(t, f.svcA(), "shared_code", "tenant-a-shared")
	b := mustCreateType(t, f.svcB(), "shared_code", "tenant-b-shared")
	if a.ID == 0 || b.ID == 0 || a.ID == b.ID {
		t.Fatalf("expected distinct rows per tenant, got %+v / %+v", a, b)
	}

	// Same item value in both tenants: allowed.
	mustCreateItem(t, f.svcA(), "shared_code", "dup-value")
	mustCreateItem(t, f.svcB(), "shared_code", "dup-value")

	// But uniqueness is enforced within a tenant.
	if _, err := f.svcA().CreateDictItem(&DictItemCreateReq{DictCode: "shared_code", ItemLabelKey: "x", ItemValue: "dup-value", Status: 1}); !errors.Is(err, common.ErrConflict) {
		t.Fatalf("expected tenant-local conflict, got %v", err)
	}
	if _, err := f.svcB().CreateDictItem(&DictItemCreateReq{DictCode: "shared_code", ItemLabelKey: "x", ItemValue: "dup-value", Status: 1}); !errors.Is(err, common.ErrConflict) {
		t.Fatalf("expected tenant-local conflict in tenant B, got %v", err)
	}
}

// 2. Cross-tenant reads return empty/404 (no existence leak).
func TestTenantCanary_CrossTenantReadDenied(t *testing.T) {
	f := newTenantFixture(t)
	created := mustCreateType(t, f.svcA(), "secret_code", "tenant-a-secret")
	mustCreateItem(t, f.svcA(), "secret_code", "secret-1")

	// Tenant B list must not contain tenant A's dict.
	rowsB, err := f.svcB().ListDictTypes(&DictTypeListQuery{})
	if err != nil {
		t.Fatalf("list as tenant B: %v", err)
	}
	for _, row := range rowsB {
		if row.DictCode == "secret_code" {
			t.Fatal("tenant B must not see tenant A dict type")
		}
	}

	// Tenant B detail read by ID => not found.
	if _, err := f.svcB().UpdateDictType(created.ID, &DictTypeUpdateReq{DictCode: "secret_code", DictName: "hijack", Status: 1}); !isNotFound(err) {
		t.Fatalf("expected not-found cross-tenant update, got %v", err)
	}

	// Items of the same code in tenant B: only B's own rows. Tenant B owns its
	// own "shared_code" dict type (same code as tenant A/global — allowed).
	mustCreateType(t, f.svcB(), "shared_code", "tenant-b-shared")
	mustCreateItem(t, f.svcB(), "shared_code", "b-only")
	pageB, err := f.svcB().ListDictItems(&DictItemListQuery{DictCode: "shared_code"})
	if err != nil {
		t.Fatalf("list items as tenant B: %v", err)
	}
	for _, item := range pageB.Items {
		if item.ItemValue == "global-1" || item.ItemValue == "dup-value" && item.ID == 0 {
			t.Fatal("tenant B must not observe other tenants' items")
		}
	}
	if len(pageB.Items) != 1 || pageB.Items[0].ItemValue != "b-only" {
		t.Fatalf("tenant B expected only its own item, got %+v", pageB.Items)
	}
}

// 3. ID tampering on update/delete/batch is contained within the tenant.
func TestTenantCanary_IDTamperingContained(t *testing.T) {
	f := newTenantFixture(t)
	mustCreateType(t, f.svcA(), "shared_code", "tenant-a-shared")
	aItem := mustCreateItem(t, f.svcA(), "shared_code", "a-treasure")
	aType := mustCreateType(t, f.svcA(), "a_type", "tenant-a-type")
	mustCreateType(t, f.svcB(), "b_type", "tenant-b-type")
	ownItem := mustCreateItem(t, f.svcB(), "b_type", "b-own")

	// Tenant B tries to delete A's item by ID.
	if err := f.svcB().DeleteDictItem(aItem.ID); !isNotFound(err) {
		t.Fatalf("expected not-found for cross-tenant item delete, got %v", err)
	}
	// Tenant B tries to batch-status A's item among its own IDs.
	count, err := f.svcB().BatchUpdateDictItemStatus([]uint64{ownItem.ID, aItem.ID}, 2)
	if err == nil {
		t.Fatalf("expected batch containing foreign ID to fail, got count=%d", count)
	}
	// Tenant B tries to delete A's type.
	if err := f.svcB().DeleteDictType(aType.ID); !isNotFound(err) {
		t.Fatalf("expected not-found for cross-tenant type delete, got %v", err)
	}
}

// 4. Compat/global rows stay invisible to multi-mode tenants (and vice versa).
func TestTenantCanary_GlobalRowsNotLeaking(t *testing.T) {
	f := newTenantFixture(t)

	rowsA, err := f.svcA().ListDictTypes(&DictTypeListQuery{})
	if err != nil {
		t.Fatalf("list as tenant A: %v", err)
	}
	for _, row := range rowsA {
		if row.DictCode == "shared_code" && row.DictName == "global-shared" {
			t.Fatal("multi-mode tenant must not read platform-global rows through the scoped service")
		}
	}

	// Global service (compat) sees only tenant-0 rows.
	rowsWorld, err := f.service.ListDictTypes(&DictTypeListQuery{})
	if err != nil {
		t.Fatalf("list as global: %v", err)
	}
	for _, row := range rowsWorld {
		if row.DictName == "tenant-a-shared" {
			t.Fatal("global view must not contain tenant rows")
		}
	}
}

// 5. Export/export paths are tenant-filtered.
func TestTenantCanary_ExportFiltered(t *testing.T) {
	f := newTenantFixture(t)
	mustCreateType(t, f.svcA(), "export_a", "tenant-a-export-type")
	mustCreateType(t, f.svcB(), "export_b", "tenant-b-export-type")
	mustCreateItem(t, f.svcA(), "export_a", "a-export")
	mustCreateItem(t, f.svcB(), "export_b", "b-export")

	fileA, err := f.svcA().ExportDictItems(&DictItemListQuery{DictCode: "export_a"})
	if err != nil {
		t.Fatalf("export as tenant A: %v", err)
	}
	for _, row := range fileA.Rows {
		if row[2] == "b-export" {
			t.Fatal("tenant A export must not contain tenant B items")
		}
	}
	if len(fileA.Rows) != 1 || fileA.Rows[0][2] != "a-export" {
		t.Fatalf("tenant A export expected exactly its own item, got %+v", fileA.Rows)
	}
}

// 6. Option cache is tenant-namespaced: no cross-tenant cache hits.
func TestTenantCanary_OptionCacheIsolated(t *testing.T) {
	f := newTenantFixture(t)
	mustCreateType(t, f.svcA(), "cache_a", "tenant-a-cache-type")
	mustCreateType(t, f.svcB(), "cache_b", "tenant-b-cache-type")
	mustCreateItem(t, f.svcA(), "cache_a", "a-opt")
	mustCreateItem(t, f.svcB(), "cache_b", "b-opt")

	// A loads options first (warm cache for A).
	optsA, err := f.svcA().GetDictOptions([]string{"cache_a"})
	if err != nil {
		t.Fatalf("options as tenant A: %v", err)
	}
	if len(optsA["cache_a"]) != 1 || optsA["cache_a"][0].Value != "a-opt" {
		t.Fatalf("tenant A expected its own option, got %+v", optsA["cache_a"])
	}

	// B requests the same code: must hit DB, not A's cache rows.
	optsB, err := f.svcB().GetDictOptions([]string{"cache_b"})
	if err != nil {
		t.Fatalf("options as tenant B: %v", err)
	}
	if len(optsB["cache_b"]) != 1 || optsB["cache_b"][0].Value != "b-opt" {
		t.Fatalf("tenant B expected its own option, got %+v", optsB["cache_b"])
	}

	// Mutating B's dict must not disturb A's cached view.
	bType := mustCreateType(t, f.svcB(), "cache_b2", "tenant-b-cache2")
	if err := f.svcB().DeleteDictType(bType.ID); err != nil {
		t.Fatalf("cleanup tenant B type: %v", err)
	}
}

// 7. Flag-off regression: nil/compat context reproduces legacy behavior.
func TestTenantCanary_FlagOffRegression(t *testing.T) {
	f := newTenantFixture(t)

	// Shared base service (no tenant context) = legacy behavior.
	rows, err := f.service.ListDictTypes(&DictTypeListQuery{})
	if err != nil {
		t.Fatalf("flag-off list: %v", err)
	}
	found := false
	for _, row := range rows {
		if row.DictCode == "shared_code" && row.DictName == "global-shared" {
			found = true
		}
	}
	if !found {
		t.Fatal("flag-off must still see global dicts")
	}

	// Create through the base service stamps tenant 0.
	row := mustCreateType(t, f.service, "legacy_code", "legacy")
	if row.ID == 0 {
		t.Fatal("legacy create failed")
	}
	var stored SystemDictType
	if err := f.db.First(&stored, row.ID).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if stored.TenantID != 0 {
		t.Fatalf("flag-off rows must be tenant 0, got %d", stored.TenantID)
	}

	// Multi-tenant service must not see the legacy row.
	if _, err := f.svcA().UpdateDictType(row.ID, &DictTypeUpdateReq{DictCode: "legacy_code", DictName: "x", Status: 1}); !isNotFound(err) {
		t.Fatalf("expected not-found for tenant reading tenant-0 row, got %v", err)
	}
}

// 8. Context forgery: untrusted context construction is denied by resolution.
func TestTenantCanary_ContextForgeryDenied(t *testing.T) {
	// Header without platform permission.
	if _, err := tenant.ResolveForCanary("multi", "42", "999", false); !errors.Is(err, tenant.ErrTenantForbidden) {
		t.Fatalf("forged header must be denied, got %v", err)
	}
	// Missing context in multi mode.
	if _, err := tenant.ResolveForCanary("multi", "", "", false); !errors.Is(err, tenant.ErrTenantContextMissing) {
		t.Fatalf("missing context must be denied, got %v", err)
	}
	// Tenant 0 can never be selected explicitly (would widen to global).
	if _, err := tenant.ResolveForCanary("multi", "0", "", false); !errors.Is(err, tenant.ErrTenantForbidden) {
		t.Fatalf("tenant 0 claim must be denied, got %v", err)
	}
}

// 9. Membership check denies non-members (contract §5: TENANT_FORBIDDEN path).
func TestTenantCanary_MembershipDenyByDefault(t *testing.T) {
	f := newTenantFixture(t)
	if tenant.HasActiveMembership(f.db, 101, 7) {
		t.Fatal("empty membership table must deny")
	}
	if tenant.HasActiveMembership(f.db, 0, 7) {
		t.Fatal("tenant 0 membership must deny (0 is not a tenant)")
	}
	if err := f.db.Create(&tenant.Membership{TenantID: 101, UserID: 7, Role: "member", Status: tenant.MembershipDisabled}).Error; err != nil {
		t.Fatalf("seed disabled membership: %v", err)
	}
	if tenant.HasActiveMembership(f.db, 101, 7) {
		t.Fatal("disabled membership must deny")
	}
	if err := f.db.Model(&tenant.Membership{}).Where("tenant_id = ? AND user_id = ?", 101, 7).Update("status", tenant.MembershipActive).Error; err != nil {
		t.Fatalf("activate membership: %v", err)
	}
	if !tenant.HasActiveMembership(f.db, 101, 7) {
		t.Fatal("active membership must allow")
	}
}

// 10. Handler binding: compat request reuses shared service; multi gets a view.
func TestTenantCanary_HandlerBinding(t *testing.T) {
	gin.SetMode(gin.TestMode)
	f := newTenantFixture(t)

	c, _ := gin.CreateTestContext(nil)
	bound := f.service.WithTenantContext(tenant.FromGin(c))
	if bound != f.service {
		t.Fatal("missing context must return the shared (compat) service")
	}

	c2, _ := gin.CreateTestContext(nil)
	tenant.SetGin(c2, f.ctxA)
	if f.service.WithTenantContext(tenant.FromGin(c2)) == f.service {
		t.Fatal("multi-mode context must produce a tenant-bound view")
	}
}
