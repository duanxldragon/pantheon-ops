// Package businessauth defines ephemeral read ports for opted-in business routes.
package businessauth

import (
	"context"
	"errors"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/pkg/authtoken"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/tenant"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

// Identity is the session-backed subject a business route was admitted with.
type Identity struct {
	ActorID       uint64 `json:"-"`
	SessionID     string `json:"-"`
	AccessToken   string `json:"-"`
	ClaimTenantID uint64 `json:"-"`
}

// RoleBinding is one role key scoped to a tenant; tenant 0 means global.
type RoleBinding struct {
	TenantID uint64 `json:"-"`
	RoleKey  string `json:"-"`
}

// Principal is the refreshed subject plus its live global bindings.
type Principal struct {
	Identity       Identity      `json:"-"`
	GlobalBindings []RoleBinding `json:"-"`
}

// Action is the request action being authorized.
type Action struct {
	Permission    string `json:"-"`
	Method        string `json:"-"`
	Path          string `json:"-"`
	RouteTemplate string `json:"-"`
}

// Input carries the frozen authorization input reused for reauthorization.
type Input struct {
	Identity         Identity       `json:"-"`
	Resolution       tenant.Context `json:"-"`
	ResourceTenantID uint64         `json:"-"`
	Action           Action         `json:"-"`
	DataScope        bool           `json:"-"`
}

// Authorization is the granted bindings and optional data scope.
type Authorization struct {
	DataScope *common.DataScopeReq `json:"-"`
	Bindings  []RoleBinding        `json:"-"`
}

// Reader is the business authorization port implemented by the auth module.
type Reader interface {
	RefreshPrincipal(context.Context, Identity) (Principal, error)
	Authorize(context.Context, Input) (Authorization, error)
}

// OwnerLifecycle is a metadata-only existence projection, never an access grant.
// False is authoritative only when ReadOwnerLifecycle returns a nil error.
type OwnerLifecycle struct {
	TenantExists bool `json:"-"`
	ActorExists  bool `json:"-"`
}

// OwnerLifecycleReader is optional so existing authorization Readers remain
// compatible. Maintenance requires this capability before reclaiming orphans.
// Disabled actors and suspended/archived tenants still exist; tenant 0 is the
// implicit platform owner. Actor 0 and uncertain storage state return an error.
type OwnerLifecycleReader interface {
	ReadOwnerLifecycle(ctx context.Context, tenantID, actorID uint64) (OwnerLifecycle, error)
}

// ErrOwnerLifecycleUnavailable reports that owner existence could not be proven.
var ErrOwnerLifecycleUnavailable = errors.New("business.owner.lifecycle_unavailable")

// ErrDenied is the single opaque business denial.
var ErrDenied = errors.New("permission.denied")

// IdentityKey stores the admitted business identity on the Gin context.
const IdentityKey = "pantheon_business_identity"
const checkpointKey = "pantheon_business_checkpoint"

// ReadSession never mutates a token, blacklist, cache, or activity timestamp.
func ReadSession(ctx context.Context, rdb *redis.Client, token string) (*authtoken.SessionData, error) {
	if ctx == nil || ctx.Err() != nil || rdb == nil || token == "" {
		return nil, ErrDenied
	}
	data, err := authtoken.ValidateSession(ctx, rdb, token)
	if err != nil || data == nil || data.UserID == 0 || data.SessionID == "" {
		return nil, ErrDenied
	}
	ttl, err := rdb.PTTL(ctx, authtoken.SessionKey(token)).Result()
	if err != nil || ttl <= 0 {
		return nil, ErrDenied
	}
	for _, key := range []string{authtoken.BlacklistUserKey(data.UserID), authtoken.BlacklistSessionKey(data.SessionID)} {
		// Only an absent key is acceptable. Empty and persistent markers still revoke.
		if _, err := rdb.Get(ctx, key).Result(); err != redis.Nil {
			return nil, ErrDenied
		}
	}
	if ctx.Err() != nil {
		return nil, ErrDenied
	}
	return data, nil
}

// IdentityFromSession projects a validated session into the route identity.
func IdentityFromSession(data *authtoken.SessionData, token string) Identity {
	return Identity{ActorID: data.UserID, SessionID: data.SessionID, AccessToken: token, ClaimTenantID: data.TenantID}
}

type checkpoint struct {
	reader Reader
	mode   tenant.ModeReader
	input  Input
}

// BindGin is middleware-only; callers receive copies of the frozen input.
func BindGin(c *gin.Context, reader Reader, mode tenant.ModeReader, input Input) {
	c.Set(checkpointKey, checkpoint{reader: reader, mode: mode, input: input})
}

// FromGin returns a copy of the input frozen by the authorization middleware.
func FromGin(c *gin.Context) (Input, error) {
	if c == nil {
		return Input{}, ErrDenied
	}
	value, ok := c.Get(checkpointKey)
	bound, valid := value.(checkpoint)
	if !ok || !valid {
		return Input{}, ErrDenied
	}
	return bound.input, nil
}

// Reauthorize refreshes the frozen subject, provenance, action and DataScope.
// A secondary action may check a source-kind permission; it cannot change tenancy.
func Reauthorize(c *gin.Context, action *Action) (Authorization, error) {
	if c == nil || c.Request == nil {
		return Authorization{}, ErrDenied
	}
	value, _ := c.Get(checkpointKey)
	bound, ok := value.(checkpoint)
	if !ok || bound.reader == nil || bound.mode == nil {
		return Authorization{}, ErrDenied
	}
	mode, err := bound.mode.ReadMode(c.Request.Context())
	if err != nil || mode != bound.input.Resolution.Mode {
		return Authorization{}, ErrDenied
	}
	input := bound.input
	if action != nil {
		input.Action = *action
	}
	auth, err := bound.reader.Authorize(c.Request.Context(), input)
	if err != nil || (input.DataScope && auth.DataScope == nil) {
		return Authorization{}, ErrDenied
	}
	if auth.DataScope != nil {
		c.Set(common.DataScopeContextKey, auth.DataScope)
	}
	keys := make([]string, 0, len(auth.Bindings))
	for _, binding := range auth.Bindings {
		key := tenant.RoleSubject(binding.RoleKey, binding.TenantID)
		if binding.TenantID == 0 {
			key = tenant.GlobalRoleSubject(binding.RoleKey)
		}
		keys = append(keys, key)
	}
	c.Set("roleKeys", keys)
	c.Set("roleKey", "")
	if len(keys) > 0 {
		c.Set("roleKey", keys[0])
	}
	return auth, nil
}

// ValidActivity rejects an expired Redis activity snapshot without refreshing it.
func ValidActivity(data *authtoken.SessionData, now time.Time, idleMinutes int) bool {
	return idleMinutes <= 0 || (data.LastActivityAt > 0 && !time.Unix(data.LastActivityAt, 0).Add(time.Duration(idleMinutes)*time.Minute).Before(now))
}
