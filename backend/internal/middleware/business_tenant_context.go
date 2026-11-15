package middleware

import (
	"context"
	"strconv"
	"strings"

	"github.com/duanxldragon/pantheon-base/backend/pkg/contracts/businessauth"
	"github.com/duanxldragon/pantheon-base/backend/pkg/database"
	"github.com/duanxldragon/pantheon-base/backend/pkg/tenant"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const permissionDeniedCode = "permission.denied"

// BusinessTenantContextMiddleware establishes current principal before any
// header capability check and freezes provenance from tenant.Resolve itself.
func BusinessTenantContextMiddleware(db *gorm.DB, mode tenant.ModeReader, reader businessauth.Reader, platformScope bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		resolved, err := resolveBusinessTenantContext(c, db, mode, reader, platformScope)
		if err != nil {
			failPermissionCheck(c, permissionDeniedCode)
			return
		}
		tenant.SetGin(c, resolved)
		c.Next()
	}
}

func resolveBusinessTenantContext(c *gin.Context, db *gorm.DB, mode tenant.ModeReader, reader businessauth.Reader, platformScope bool) (*tenant.Context, error) {
	value, _ := c.Get(businessauth.IdentityKey)
	identity, ok := value.(businessauth.Identity)
	if !ok || db == nil || reader == nil || mode == nil {
		return nil, businessauth.ErrDenied
	}
	ctx := c.Request.Context()
	principal, err := reader.RefreshPrincipal(ctx, identity)
	if err != nil || principal.Identity != identity {
		return nil, businessauth.ErrDenied
	}
	currentMode, err := mode.ReadMode(ctx)
	if err != nil || !businessTenantExecutableMode(currentMode) {
		return nil, businessauth.ErrDenied
	}
	header := strings.TrimSpace(c.GetHeader(tenant.HeaderTenantID))
	if currentMode == tenant.ModeMulti && identity.ClaimTenantID > 0 && !businessClaimAuthorized(ctx, db, identity) {
		return nil, businessauth.ErrDenied
	}
	capable, err := businessHeaderCapability(ctx, db, c, currentMode, platformScope, principal, header)
	if err != nil {
		return nil, err
	}
	claim := ""
	if identity.ClaimTenantID > 0 {
		claim = strconv.FormatUint(identity.ClaimTenantID, 10)
	}
	population, failed := businessPlatformPopulationStatus(ctx, db, currentMode, platformScope, identity, header)
	// Compat needs no membership lookup. Multi fallback must prove no memberships.
	resolved, err := tenant.Resolve(tenant.ResolveInput{Mode: currentMode, SubjectTenantID: claim, HeaderTenantID: header, HeaderAllowed: capable, PlatformScoped: platformScope, PlatformCapable: capable && !failed && population})
	if err != nil {
		return nil, err
	}
	return resolved, nil
}

func businessTenantExecutableMode(currentMode string) bool {
	return currentMode == tenant.ModeCompat || currentMode == tenant.ModeMulti
}

func businessClaimAuthorized(ctx context.Context, db *gorm.DB, identity businessauth.Identity) bool {
	currentDB := db.WithContext(ctx)
	if tenant.EnsureTenantLoginable(currentDB, identity.ClaimTenantID) != nil || !tenant.HasActiveMembership(currentDB, identity.ClaimTenantID, identity.ActorID) {
		return false
	}
	var count int64
	err := currentDB.Table("system_user_role ur").Joins("JOIN system_role r ON r.id = ur.role_id AND r.tenant_id = ur.tenant_id").
		Where("ur.user_id = ? AND ur.tenant_id = ? AND r.status = ? AND r.deleted_at IS NULL", identity.ActorID, identity.ClaimTenantID, 1).Count(&count).Error
	return err == nil && count > 0
}

func businessHeaderCapability(ctx context.Context, db *gorm.DB, c *gin.Context, currentMode string, platformScope bool, principal businessauth.Principal, header string) (bool, error) {
	if currentMode != tenant.ModeMulti {
		return false, nil
	}
	if header == "" && (!platformScope || principal.Identity.ClaimTenantID != 0) {
		return false, nil
	}
	for _, binding := range principal.GlobalBindings {
		if binding.TenantID != 0 {
			return false, businessauth.ErrDenied
		}
	}
	policy, err := database.ReadBusinessPolicy(ctx, db)
	if err != nil {
		return false, businessauth.ErrDenied
	}
	_, err = database.AuthorizeBusinessBindings(ctx, db, policy, principal.GlobalBindings, businessauth.Action{Method: c.Request.Method, Path: c.Request.URL.Path, RouteTemplate: c.FullPath()})
	return err == nil, nil
}

func businessPlatformPopulationStatus(ctx context.Context, db *gorm.DB, currentMode string, platformScope bool, identity businessauth.Identity, header string) (population, failed bool) {
	if currentMode != tenant.ModeMulti || !platformScope || identity.ClaimTenantID != 0 || header != "" {
		return false, false
	}
	return tenant.IsPlatformPopulationUser(db.WithContext(ctx), identity.ActorID)
}
