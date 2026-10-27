package permission

import (
	"errors"
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/database"
	tenantctx "github.com/duanxldragon/pantheon-base/backend/pkg/tenant"

	"github.com/stretchr/testify/require"
)

// The workbench judges stored policies with Casbin's keyMatch2 matcher. The
// platform administrator and every tenant role are granted `/api/v1/*`, so an
// exact string comparison reported "missing all API policies" for roles that
// actually hold every route.
func TestDiffMissingAPIPolicies_WildcardCountsAsCoverage(t *testing.T) {
	required := []permissionRequiredAPIPolicy{
		{Path: "/api/v1/system/user/list", Method: "GET"},
		{Path: "/api/v1/system/user", Method: "POST"},
		{Path: "/api/v1/system/user/export", Method: "POST"},
	}
	actual := []PermissionWorkbenchAPIPolicyResp{
		{Path: "/api/v1/*", Method: "GET"},
		{Path: "/api/v1/*", Method: "POST"},
	}

	require.Empty(t, diffMissingAPIPolicies(required, actual))
}

// The HTTP method is part of the Casbin rule, not the path pattern: a GET-only
// wildcard must still report the POST routes it does not authorise.
func TestDiffMissingAPIPolicies_GetWildcardDoesNotCoverPost(t *testing.T) {
	required := []permissionRequiredAPIPolicy{
		{Path: "/api/v1/system/user/list", Method: "GET"},
		{Path: "/api/v1/system/user", Method: "POST"},
	}
	actual := []PermissionWorkbenchAPIPolicyResp{{Path: "/api/v1/*", Method: "GET"}}

	missing := diffMissingAPIPolicies(required, actual)

	require.Equal(t, []PermissionWorkbenchAPIPolicyResp{
		{Path: "/api/v1/system/user", Method: "POST"},
	}, missing)
}

// Each scope governs the roles it owns. The platform board used to include
// every tenant's roles, but role_key is only unique per tenant — every tenant
// owns a `tenant_admin` — so the rows collided in roleKey-keyed maps, policies
// resolved by the bare key (always zero for a domain subject), and remediation
// would have written a dead global policy.
func TestFetchWorkbenchRoles_ScopesToTheCallerTenant(t *testing.T) {
	db := setupPermissionTestDB(t)
	require.NoError(t, db.Create(&[]permissionTestRole{
		{ID: 1, TenantID: 0, RoleName: "platform admin", RoleKey: "admin", Status: 1},
		{ID: 2, TenantID: 101, RoleName: "tenant admin", RoleKey: tenantctx.TenantRoleAdminKey, Status: 1},
		{ID: 3, TenantID: 202, RoleName: "editor", RoleKey: "editor", Status: 1},
	}).Error)

	platformRoles, err := NewPermissionService(db).fetchWorkbenchRoles(nil)
	require.NoError(t, err)
	require.Len(t, platformRoles, 1)
	require.Equal(t, "admin", platformRoles[0].RoleKey)

	tenantRoles, err := NewPermissionService(db).
		WithTenantContext(&tenantctx.Context{TenantID: 101, Mode: tenantctx.ModeMulti}).
		fetchWorkbenchRoles(nil)
	require.NoError(t, err)
	require.Len(t, tenantRoles, 1)
	require.Equal(t, tenantctx.TenantRoleAdminKey, tenantRoles[0].RoleKey)
}

// A tenant role key is only unique inside its tenant. The platform console no
// longer lists tenant roles, and the remediation API must not accept one
// either: it has no tenant target, so it would write a dead global subject for
// `tenant_admin` instead of the tenant namespace.
func TestRemediateWorkbenchPolicies_PlatformScopeCannotTargetTenantRole(t *testing.T) {
	db := setupPermissionTestDB(t)
	require.NoError(t, db.Create(&permissionTestRole{
		ID: 2, TenantID: 101, RoleName: "tenant admin", RoleKey: tenantctx.TenantRoleAdminKey, Status: 1,
	}).Error)

	_, err := NewPermissionService(db).RemediateWorkbenchPolicies(&PermissionWorkbenchRemediateReq{
		RoleKey: tenantctx.TenantRoleAdminKey,
	})
	require.Error(t, err)
	require.True(t, errors.Is(err, common.ErrBadRequest), "expected a bad request, got %v", err)

	var globalPolicies int64
	require.NoError(t, db.Model(&database.CasbinRule{}).
		Where("v0 = ?", tenantctx.TenantRoleAdminKey).
		Count(&globalPolicies).Error)
	require.Zero(t, globalPolicies, "remediation must not write a global subject for a tenant role")
}
