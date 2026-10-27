package role

import (
	"strings"

	"github.com/casbin/casbin/v2/util"
	"github.com/duanxldragon/pantheon-base/backend/modules/system/iam/permission"
	"github.com/duanxldragon/pantheon-base/backend/pkg/database"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type roleAPIRoute struct {
	path   string
	method string
}

// syncRoleAPIPolicies keeps the Casbin policies of one role in step with the
// permission keys the role currently holds. Only routes declared by the shared
// permission→route binding table are managed here: policies outside that table
// (wildcards written by tenant provisioning, manual grants from the permission
// page) are never touched.
// syncRoleAPIPolicies reconciles one role's Casbin rows with the routes its
// permission keys declare. The three steps are separate functions because each
// one is a different question - what the binding table manages, which rows are
// stale, which grants are missing - and the policy set is only correct when all
// three read the same route maps.
func syncRoleAPIPolicies(tx *gorm.DB, subject string, permissionKeys []string) error {
	subject = strings.TrimSpace(subject)
	if subject == "" {
		return nil
	}

	managed, desired := rolePolicyRouteSets(permissionKeys)
	if len(managed) == 0 {
		return nil
	}

	var existing []database.CasbinRule
	if err := tx.Where("ptype = ? AND v0 = ?", "p", subject).Find(&existing).Error; err != nil {
		return err
	}

	if err := deleteRetiredRolePolicies(tx, existing, managed, desired); err != nil {
		return err
	}

	created := missingRolePolicies(subject, desired, existing)
	if len(created) == 0 {
		return nil
	}
	// Remediation and role saves can race on the same natural key; the unique
	// index keeps one row and the loser simply observes it as already present.
	return tx.Clauses(clause.OnConflict{DoNothing: true}).CreateInBatches(created, 100).Error
}

// rolePolicyRouteSets returns every route the shared binding table declares and
// the subset of those routes the given permission keys grant.
func rolePolicyRouteSets(permissionKeys []string) (managed, desired map[roleAPIRoute]struct{}) {
	heldKeys := make(map[string]struct{}, len(permissionKeys))
	for _, key := range normalizePermissionKeys(permissionKeys) {
		heldKeys[key] = struct{}{}
	}

	managed = make(map[roleAPIRoute]struct{})
	desired = make(map[roleAPIRoute]struct{})
	for permissionKey, entries := range permission.RequiredAPIRoutesByPermission() {
		for _, entry := range entries {
			route := roleAPIRoute{
				path:   strings.TrimSpace(entry.Path),
				method: normalizeRolePolicyMethod(entry.Method),
			}
			if route.path == "" || route.method == "" {
				continue
			}
			managed[route] = struct{}{}
			if _, ok := heldKeys[permissionKey]; ok {
				desired[route] = struct{}{}
			}
		}
	}
	return managed, desired
}

// deleteRetiredRolePolicies drops the stored rows that the binding table
// manages but the role no longer grants. Rows outside `managed` are never
// touched, so wildcard and manual grants survive a role save.
func deleteRetiredRolePolicies(
	tx *gorm.DB,
	existing []database.CasbinRule,
	managed, desired map[roleAPIRoute]struct{},
) error {
	removedIDs := make([]uint64, 0)
	for _, policy := range existing {
		route := roleAPIRoute{
			path:   strings.TrimSpace(policy.V1),
			method: normalizeRolePolicyMethod(policy.V2),
		}
		if _, ok := managed[route]; !ok {
			continue
		}
		if _, ok := desired[route]; !ok {
			removedIDs = append(removedIDs, policy.ID)
		}
	}
	if len(removedIDs) == 0 {
		return nil
	}
	return tx.Where("id IN ?", removedIDs).Delete(&database.CasbinRule{}).Error
}

// missingRolePolicies builds the rows to insert for granted routes the role has
// no policy for yet.
func missingRolePolicies(
	subject string,
	desired map[roleAPIRoute]struct{},
	existing []database.CasbinRule,
) []database.CasbinRule {
	present := make(map[roleAPIRoute]struct{}, len(existing))
	for _, policy := range existing {
		route := roleAPIRoute{
			path:   strings.TrimSpace(policy.V1),
			method: normalizeRolePolicyMethod(policy.V2),
		}
		present[route] = struct{}{}
	}

	created := make([]database.CasbinRule, 0, len(desired))
	for route := range desired {
		if _, ok := present[route]; ok {
			continue
		}
		// A wildcard grant (`/api/v1/*`) already enforces this route; adding an
		// exact duplicate would only bloat the policy table.
		if rolePolicyCoversRoute(existing, route) {
			continue
		}
		created = append(created, database.CasbinRule{
			PType: "p",
			V0:    subject,
			V1:    route.path,
			V2:    route.method,
		})
	}
	return created
}

func normalizeRolePolicyMethod(method string) string {
	return strings.ToUpper(strings.TrimSpace(method))
}

// rolePolicyCoversRoute reports whether an existing policy already authorises
// the route under the Casbin keyMatch2 matcher used by the enforcement model.
func rolePolicyCoversRoute(policies []database.CasbinRule, route roleAPIRoute) bool {
	for _, policy := range policies {
		if normalizeRolePolicyMethod(policy.V2) != route.method {
			continue
		}
		if util.KeyMatch2(route.path, strings.TrimSpace(policy.V1)) {
			return true
		}
	}
	return false
}
