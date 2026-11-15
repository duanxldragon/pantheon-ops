package login

import (
	"context"
	"strconv"
	"strings"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/internal/middleware"
	"github.com/duanxldragon/pantheon-base/backend/pkg/authsession"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/contracts/businessauth"
	"github.com/duanxldragon/pantheon-base/backend/pkg/database"
	"github.com/duanxldragon/pantheon-base/backend/pkg/tenant"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type businessAuthorizationReader struct {
	runtime *Runtime
	rdb     *redis.Client
	mode    tenant.ModeReader
}

// NewBusinessAuthorizationReader builds the auth-owned business authorization port.
func NewBusinessAuthorizationReader(runtime *Runtime, rdb *redis.Client, mode tenant.ModeReader) businessauth.Reader {
	return &businessAuthorizationReader{runtime: runtime, rdb: rdb, mode: mode}
}

// ReadOwnerLifecycle reads only authoritative owner metadata. It deliberately
// avoids credentials and enabled/session/membership/role authorization gates:
// losing access does not mean an owner's private records may be destroyed.
func (r *businessAuthorizationReader) ReadOwnerLifecycle(ctx context.Context, tenantID, actorID uint64) (businessauth.OwnerLifecycle, error) {
	unknown := businessauth.OwnerLifecycle{}
	if r == nil || r.runtime == nil || r.runtime.db == nil || ctx == nil || ctx.Err() != nil || actorID == 0 {
		return unknown, businessauth.ErrOwnerLifecycleUnavailable
	}
	// Owner metadata and SQL failures stay private without changing auth logging.
	db := r.runtime.db.Session(&gorm.Session{Logger: logger.Discard}).WithContext(ctx)
	var actors int64
	if err := db.Table("system_user").Where("id = ? AND deleted_at IS NULL", actorID).Count(&actors).Error; err != nil {
		return unknown, businessauth.ErrOwnerLifecycleUnavailable
	}
	exists := businessauth.OwnerLifecycle{TenantExists: tenantID == 0, ActorExists: actors > 0}
	if tenantID > 0 {
		var tenants int64
		if err := db.Model(&tenant.Tenant{}).Where("id = ? AND status <> ?", tenantID, tenant.TenantStatusDeleted).Count(&tenants).Error; err != nil {
			return unknown, businessauth.ErrOwnerLifecycleUnavailable
		}
		exists.TenantExists = tenants > 0
	}
	if ctx.Err() != nil {
		return unknown, businessauth.ErrOwnerLifecycleUnavailable
	}
	return exists, nil
}

// RefreshPrincipal validates live session ownership and current enabled user,
// then reads actual global bindings. Token roles never participate in grants.
func (r *businessAuthorizationReader) RefreshPrincipal(ctx context.Context, identity businessauth.Identity) (businessauth.Principal, error) {
	if r == nil || r.runtime == nil || r.runtime.db == nil || r.runtime.userRepo == nil || ctx == nil || ctx.Err() != nil || identity.ActorID == 0 {
		return businessauth.Principal{}, businessauth.ErrDenied
	}
	data, err := businessauth.ReadSession(ctx, r.rdb, identity.AccessToken)
	if err != nil || businessauth.IdentityFromSession(data, identity.AccessToken) != identity {
		return businessauth.Principal{}, businessauth.ErrDenied
	}
	db := r.runtime.db.WithContext(ctx)
	// Read idle policy directly, without Runtime's settings cache or defaults on error.
	var policy struct{ SettingValue string }
	if err := db.Table("system_setting").Select("setting_value").Where("tenant_id = ? AND setting_key = ?", 0, settingSessionIdleMinutesKey).Take(&policy).Error; err != nil {
		return businessauth.Principal{}, businessauth.ErrDenied
	}
	idle, err := strconv.Atoi(tenant.NormalizeFlagValue(policy.SettingValue))
	if err != nil || idle < 0 || !businessauth.ValidActivity(data, time.Now(), idle) {
		return businessauth.Principal{}, businessauth.ErrDenied
	}
	var count int64
	active := authsession.ApplyActiveScope(db.Table("system_user_session"), "", time.Now(), idle)
	if err := active.Where("session_id = ? AND user_id = ? AND tenant_id = ?", identity.SessionID, identity.ActorID, identity.ClaimTenantID).Count(&count).Error; err != nil || count != 1 {
		return businessauth.Principal{}, businessauth.ErrDenied
	}
	user, err := r.runtime.userRepo.FindByID(ctx, identity.ActorID)
	if err != nil || user == nil || user.ID != identity.ActorID || user.Status != common.StatusEnabled {
		return businessauth.Principal{}, businessauth.ErrDenied
	}
	bindings, err := r.readBindings(ctx, identity.ActorID, 0)
	if err != nil || ctx.Err() != nil {
		return businessauth.Principal{}, businessauth.ErrDenied
	}
	return businessauth.Principal{Identity: identity, GlobalBindings: bindings}, nil
}

func (r *businessAuthorizationReader) readBindings(ctx context.Context, actor, tenantID uint64) ([]businessauth.RoleBinding, error) {
	var keys []string
	err := r.runtime.db.WithContext(ctx).Table("system_role r").Select("r.role_key").
		Joins("JOIN system_user_role ur ON ur.role_id = r.id AND ur.tenant_id = r.tenant_id").
		Where("ur.user_id = ? AND ur.tenant_id = ? AND r.tenant_id = ? AND r.status = ? AND r.deleted_at IS NULL", actor, tenantID, tenantID, common.StatusEnabled).
		Order("r.id asc").Pluck("r.role_key", &keys).Error
	if err != nil {
		return nil, businessauth.ErrDenied
	}
	result := make([]businessauth.RoleBinding, 0, len(keys))
	for _, key := range keys {
		if strings.TrimSpace(key) == "" {
			return nil, businessauth.ErrDenied
		}
		result = append(result, businessauth.RoleBinding{TenantID: tenantID, RoleKey: key})
	}
	return result, nil
}

func (r *businessAuthorizationReader) Authorize(ctx context.Context, input businessauth.Input) (businessauth.Authorization, error) {
	deny := businessauth.Authorization{}
	principal, err := r.RefreshPrincipal(ctx, input.Identity)
	if err != nil || r.mode == nil {
		return deny, businessauth.ErrDenied
	}
	mode, err := r.mode.ReadMode(ctx)
	if err != nil || mode != input.Resolution.Mode || input.ResourceTenantID != input.Resolution.TenantID {
		return deny, businessauth.ErrDenied
	}
	db := r.runtime.db.WithContext(ctx)
	bindings := append([]businessauth.RoleBinding(nil), principal.GlobalBindings...)
	switch mode {
	case tenant.ModeCompat:
		if input.Resolution.TenantID != 0 || input.Resolution.ResolvedBy != tenant.ResolvedByCompatFallback {
			return deny, businessauth.ErrDenied
		}
	case tenant.ModeMulti:
		bindings, err = r.authorizeMultiMode(ctx, db, input, bindings)
		if err != nil {
			return deny, err
		}
	default:
		return deny, businessauth.ErrDenied
	}
	return r.grantBusinessAuthorization(ctx, db, input, bindings)
}

func (r *businessAuthorizationReader) authorizeMultiMode(ctx context.Context, db *gorm.DB, input businessauth.Input, bindings []businessauth.RoleBinding) ([]businessauth.RoleBinding, error) {
	// Validate the originating claim even for a header override. An override
	// changes resource scope; it cannot revive a revoked claim/session.
	if input.Identity.ClaimTenantID > 0 {
		if err := r.validateBusinessClaimOrigin(ctx, db, input.Identity); err != nil {
			return bindings, err
		}
	}
	switch input.Resolution.ResolvedBy {
	case tenant.ResolvedBySubject:
		local, err := r.authorizeSubjectBindings(ctx, db, input)
		if err != nil {
			return bindings, err
		}
		return append(bindings, local...), nil
	case tenant.ResolvedByHeader:
		// Header scope keeps only real global bindings; target roles are not acquired.
		if err := authorizeHeaderResolution(db, input); err != nil {
			return bindings, err
		}
		return bindings, nil
	case tenant.ResolvedByPlatformFallback:
		if err := authorizePlatformFallbackResolution(db, input); err != nil {
			return bindings, err
		}
		return bindings, nil
	default:
		return bindings, businessauth.ErrDenied
	}
}

func (r *businessAuthorizationReader) validateBusinessClaimOrigin(ctx context.Context, db *gorm.DB, identity businessauth.Identity) error {
	if !validBusinessMembership(db, identity.ClaimTenantID, identity.ActorID) {
		return businessauth.ErrDenied
	}
	origin, err := r.readBindings(ctx, identity.ActorID, identity.ClaimTenantID)
	if err != nil || len(origin) == 0 {
		return businessauth.ErrDenied
	}
	return nil
}

func (r *businessAuthorizationReader) authorizeSubjectBindings(ctx context.Context, db *gorm.DB, input businessauth.Input) ([]businessauth.RoleBinding, error) {
	if input.Identity.ClaimTenantID == 0 || input.Resolution.TenantID != input.Identity.ClaimTenantID {
		return nil, businessauth.ErrDenied
	}
	local, err := r.readBindings(ctx, input.Identity.ActorID, input.Resolution.TenantID)
	if err != nil || len(local) == 0 {
		return nil, businessauth.ErrDenied
	}
	return local, nil
}

func authorizeHeaderResolution(db *gorm.DB, input businessauth.Input) error {
	if input.Resolution.TenantID > 0 && tenant.EnsureTenantLoginable(db, input.Resolution.TenantID) != nil {
		return businessauth.ErrDenied
	}
	return nil
}

func authorizePlatformFallbackResolution(db *gorm.DB, input businessauth.Input) error {
	population, failed := tenant.IsPlatformPopulationUser(db, input.Identity.ActorID)
	if input.Identity.ClaimTenantID != 0 || input.Resolution.TenantID != 0 || failed || !population {
		return businessauth.ErrDenied
	}
	return nil
}

func (r *businessAuthorizationReader) grantBusinessAuthorization(ctx context.Context, db *gorm.DB, input businessauth.Input, bindings []businessauth.RoleBinding) (businessauth.Authorization, error) {
	deny := businessauth.Authorization{}
	policy, err := database.ReadBusinessPolicy(ctx, db)
	if err != nil {
		return deny, businessauth.ErrDenied
	}
	granted, err := database.AuthorizeBusinessBindings(ctx, db, policy, bindings, input.Action)
	if err != nil {
		return deny, businessauth.ErrDenied
	}
	auth := businessauth.Authorization{Bindings: granted}
	if input.DataScope {
		auth.DataScope, err = middleware.ReadBusinessDataScope(ctx, db, input.Identity.ActorID, granted)
		if err != nil {
			return deny, businessauth.ErrDenied
		}
	}
	if ctx.Err() != nil {
		return deny, businessauth.ErrDenied
	}
	return auth, nil
}

func validBusinessMembership(db *gorm.DB, tenantID, actor uint64) bool {
	return tenant.EnsureTenantLoginable(db, tenantID) == nil && tenant.HasActiveMembership(db, tenantID, actor)
}
