package login

import (
	"context"
	"strconv"
	"strings"

	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/contracts/businessauth"
	"github.com/duanxldragon/pantheon-base/backend/pkg/permissionroutes"
	"github.com/duanxldragon/pantheon-base/backend/pkg/tenant"
)

// AuthorizeJob is an auth-owned check for an already persisted, trusted job
// owner. It deliberately does not use a session, token or cached permission.
// Businesses must still revalidate their current source and targets before I/O.
func (r *businessAuthorizationReader) AuthorizeJob(ctx context.Context, input businessauth.JobInput) (businessauth.Authorization, error) {
	deny := businessauth.Authorization{}
	if r == nil || r.runtime == nil || r.runtime.db == nil || r.runtime.userRepo == nil || r.mode == nil ||
		ctx == nil || ctx.Err() != nil || input.ActorID == 0 || !registeredJobAction(input.Action) {
		return deny, businessauth.ErrDenied
	}
	mode, err := r.mode.ReadMode(ctx)
	tenantID := input.ResourceTenantID
	compatJob := mode == tenant.ModeCompat && tenantID == tenant.PlatformGlobalTenantID
	tenantJob := mode == tenant.ModeMulti && tenantID != tenant.PlatformGlobalTenantID
	if err != nil || (!compatJob && !tenantJob) {
		return deny, businessauth.ErrDenied
	}
	owner, err := r.runtime.userRepo.FindByID(ctx, input.ActorID)
	if err != nil || owner == nil || owner.ID != input.ActorID || owner.Status != common.StatusEnabled || owner.TenantID != tenantID {
		return deny, businessauth.ErrDenied
	}
	db := r.runtime.db.WithContext(ctx)
	if tenantJob && !validBusinessMembership(db, tenantID, input.ActorID) {
		return deny, businessauth.ErrDenied
	}
	bindings, err := r.readBindings(ctx, input.ActorID, tenantID)
	if err != nil {
		return deny, businessauth.ErrDenied
	}
	auth, err := r.grantBusinessAuthorization(ctx, db, businessauth.Input{
		Identity: businessauth.Identity{ActorID: input.ActorID}, ResourceTenantID: tenantID, Action: input.Action, DataScope: true,
	}, bindings)
	if err != nil || auth.DataScope == nil || ctx.Err() != nil {
		return deny, businessauth.ErrDenied
	}
	// The execution mode must still match the mode that supplied the grant.
	latestMode, err := r.mode.ReadMode(ctx)
	if err != nil || latestMode != mode || ctx.Err() != nil {
		return deny, businessauth.ErrDenied
	}
	return auth, nil
}

func registeredJobAction(action businessauth.Action) bool {
	if action.Permission == "" || action.Path == "" || action.RouteTemplate == "" || action.Method == "" {
		return false
	}
	for _, route := range permissionroutes.Lookup(action.Permission) {
		if route.Method == action.Method && route.Path == action.RouteTemplate && jobPathMatches(action.Path, route.Path) {
			return true
		}
	}
	return false
}

// The initial job contract supports positive numeric resource ID parameters.
// This ties the action
// to its provider template even when a broader Casbin rule also matches it.
func jobPathMatches(actual, template string) bool {
	actualParts, templateParts := strings.Split(actual, "/"), strings.Split(template, "/")
	if len(actualParts) != len(templateParts) {
		return false
	}
	for i, expected := range templateParts {
		if strings.HasPrefix(expected, ":") {
			id, err := strconv.ParseUint(actualParts[i], 10, 64)
			if err != nil || id == 0 || strconv.FormatUint(id, 10) != actualParts[i] {
				return false
			}
		} else if actualParts[i] != expected {
			return false
		}
	}
	return true
}
