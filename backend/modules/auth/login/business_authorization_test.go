package login

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/internal/middleware"
	"github.com/duanxldragon/pantheon-base/backend/modules/auth/session"
	"github.com/duanxldragon/pantheon-base/backend/modules/system/config/setting"
	"github.com/duanxldragon/pantheon-base/backend/modules/system/iam/role"
	"github.com/duanxldragon/pantheon-base/backend/modules/system/iam/user"
	"github.com/duanxldragon/pantheon-base/backend/pkg/authtoken"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/contracts"
	"github.com/duanxldragon/pantheon-base/backend/pkg/contracts/businessauth"
	"github.com/duanxldragon/pantheon-base/backend/pkg/database"
	"github.com/duanxldragon/pantheon-base/backend/pkg/tenant"
	"github.com/duanxldragon/pantheon-base/backend/pkg/testredis"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type businessFixture struct {
	db     *gorm.DB
	rdb    *redis.Client
	reader businessauth.Reader
	mode   tenant.ModeReader
	input  businessauth.Input
	local  role.SystemRole
	actor  user.SystemUser
}

func newBusinessFixture(t *testing.T) businessFixture {
	t.Helper()
	db := setupTestDB(t)
	if err := db.AutoMigrate(&tenant.Membership{}, &database.CasbinRule{}, &middleware.SystemRoleDataScope{}); err != nil {
		t.Fatal(err)
	}
	seedSettings(t, db, map[string]string{tenant.FeatureFlagSettingKey: "multi", settingSessionIdleMinutesKey: "30"})
	for _, id := range []uint64{101, 202} {
		if err := db.Create(&tenant.Tenant{ID: id, Code: strconvID(id), Name: "synthetic", Status: tenant.TenantStatusActive}).Error; err != nil {
			t.Fatal(err)
		}
	}
	actor := user.SystemUser{Username: "synthetic", Password: "unused", Status: 1, DeptID: 8, TenantID: 101}
	if err := db.Create(&actor).Error; err != nil {
		t.Fatal(err)
	}
	membership := tenant.Membership{TenantID: 101, UserID: actor.ID, Status: tenant.MembershipActive}
	if err := db.Create(&membership).Error; err != nil {
		t.Fatal(err)
	}
	local := role.SystemRole{RoleKey: "admin", RoleName: "tenant-local same-name", TenantID: 101, Status: 1}
	if err := db.Create(&local).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&user.SystemUserRole{UserID: actor.ID, RoleID: local.ID, TenantID: 101}).Error; err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&middleware.SystemRoleDataScope{TenantID: 101, RoleKey: "admin", Mode: common.DataScopeModeSelf}).Error; err != nil {
		t.Fatal(err)
	}
	action := businessauth.Action{Method: "POST", Path: "/api/v1/business/fresh/items/1", RouteTemplate: "/api/v1/business/fresh/items/:id"}
	if err := db.Create(&database.CasbinRule{PType: "p", V0: tenant.RoleSubject(local.RoleKey, 101), V1: action.RouteTemplate, V2: "POST"}).Error; err != nil {
		t.Fatal(err)
	}
	rdb := testredis.OpenDB(t, testredis.DBLogin)
	token := authtoken.NewAccessToken()
	now := time.Now()
	sess := session.SystemUserSession{SessionID: authtoken.NewAccessToken(), UserID: actor.ID, TenantID: 101, RefreshExpiresAt: now.Add(time.Hour), LastActivityAt: &now}
	if err := db.Create(&sess).Error; err != nil {
		t.Fatal(err)
	}
	data := &authtoken.SessionData{UserID: actor.ID, SessionID: sess.SessionID, TenantID: 101, LastActivityAt: now.Unix(), RoleKeys: []string{"admin", "platform_ops"}}
	if err := authtoken.StoreSession(context.Background(), rdb, token, data, time.Minute); err != nil {
		t.Fatal("seed opaque session")
	}
	mode := tenant.NewStrictSettingModeReader(setting.NewSettingService(db))
	runtime := &Runtime{db: db, userRepo: testCredentialRepo(db)}
	reader := NewBusinessAuthorizationReader(runtime, rdb, mode)
	input := businessauth.Input{Identity: businessauth.IdentityFromSession(data, token), Resolution: tenant.Context{Mode: "multi", TenantID: 101, ResolvedBy: tenant.ResolvedBySubject}, ResourceTenantID: 101, Action: action, DataScope: true}
	return businessFixture{db: db, rdb: rdb, reader: reader, mode: mode, input: input, local: local, actor: actor}
}
func strconvID(id uint64) string {
	if id == 101 {
		return "a"
	}
	return "b"
}
func (f businessFixture) globalGrant(t *testing.T) {
	t.Helper()
	global := role.SystemRole{RoleKey: "platform_ops", RoleName: "actual global", Status: 1}
	if err := f.db.Create(&global).Error; err != nil {
		t.Fatal(err)
	}
	if err := f.db.Create(&user.SystemUserRole{UserID: f.actor.ID, RoleID: global.ID, TenantID: 0}).Error; err != nil {
		t.Fatal(err)
	}
	if err := f.db.Create(&database.CasbinRule{PType: "p", V0: "platform_ops", V1: f.input.Action.RouteTemplate, V2: "POST"}).Error; err != nil {
		t.Fatal(err)
	}
	if err := f.db.Create(&middleware.SystemRoleDataScope{TenantID: 0, RoleKey: "platform_ops", Mode: common.DataScopeModeDept}).Error; err != nil {
		t.Fatal(err)
	}
}

func TestBusinessAuthorizationFreshNamespacesAndProvenance(t *testing.T) {
	f := newBusinessFixture(t)
	ctx := context.Background()
	p, err := f.reader.RefreshPrincipal(ctx, f.input.Identity)
	if err != nil || len(p.GlobalBindings) != 0 {
		t.Fatal("token role promoted to global binding")
	}
	auth, err := f.reader.Authorize(ctx, f.input)
	if err != nil || auth.DataScope == nil || auth.DataScope.IsAdmin || auth.DataScope.Mode != common.DataScopeModeSelf {
		t.Fatal("local admin promoted or policy not applied")
	}
	if err := f.db.Model(&user.SystemUser{}).Where("id = ?", f.actor.ID).Update("dept_id", 99).Error; err != nil {
		t.Fatal(err)
	}
	auth, err = f.reader.Authorize(ctx, f.input)
	if err != nil || auth.DataScope.DeptID != 99 {
		t.Fatal("cached department authorized")
	}
	foreign := f.input
	foreign.Resolution.TenantID = 202
	foreign.ResourceTenantID = 202
	if _, err := f.reader.Authorize(ctx, foreign); err == nil {
		t.Fatal("foreign member resolution accepted")
	}
	foreign.Resolution.ResolvedBy = tenant.ResolvedByHeader
	if _, err := f.reader.Authorize(ctx, foreign); err == nil {
		t.Fatal("token platform_ops accepted override")
	}
	f.globalGrant(t)
	auth, err = f.reader.Authorize(ctx, foreign)
	if err != nil || len(auth.Bindings) != 1 || auth.Bindings[0].TenantID != 0 || auth.DataScope.Mode != common.DataScopeModeDept {
		t.Fatal("real platform override incorrectly acquired target roles or required membership")
	}
	if err := f.db.Model(&tenant.Membership{}).Where("user_id = ?", f.actor.ID).Update("status", tenant.MembershipDisabled).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := f.reader.Authorize(ctx, foreign); err == nil {
		t.Fatal("header resurrected revoked originating claim")
	}
}

func TestBusinessAuthorizationLiveRecheckMatrix(t *testing.T) {
	cases := []struct {
		name   string
		revoke func(*testing.T, businessFixture)
	}{
		{"session", func(t *testing.T, f businessFixture) {
			mustBusinessSQL(t, f.db.Model(&session.SystemUserSession{}).Where("session_id = ?", f.input.Identity.SessionID).Update("revoked_at", time.Now()))
		}},
		{"expiry", func(t *testing.T, f businessFixture) {
			mustBusinessSQL(t, f.db.Model(&session.SystemUserSession{}).Where("session_id = ?", f.input.Identity.SessionID).Update("refresh_expires_at", time.Now().Add(-time.Minute)))
		}},
		{"idle", func(t *testing.T, f businessFixture) {
			mustBusinessSQL(t, f.db.Model(&session.SystemUserSession{}).Where("session_id = ?", f.input.Identity.SessionID).Update("last_activity_at", time.Now().Add(-time.Hour)))
		}},
		{"user", func(t *testing.T, f businessFixture) {
			mustBusinessSQL(t, f.db.Model(&user.SystemUser{}).Where("id = ?", f.actor.ID).Update("status", 2))
		}},
		{"deleted_user", func(t *testing.T, f businessFixture) { mustBusinessSQL(t, f.db.Delete(&f.actor)) }},
		{"role", func(t *testing.T, f businessFixture) {
			mustBusinessSQL(t, f.db.Model(&role.SystemRole{}).Where("id = ?", f.local.ID).Update("status", 2))
		}},
		{"deleted_role", func(t *testing.T, f businessFixture) { mustBusinessSQL(t, f.db.Delete(&f.local)) }},
		{"binding", func(t *testing.T, f businessFixture) {
			mustBusinessSQL(t, f.db.Where("user_id = ?", f.actor.ID).Delete(&user.SystemUserRole{}))
		}},
		{"membership", func(t *testing.T, f businessFixture) {
			mustBusinessSQL(t, f.db.Model(&tenant.Membership{}).Where("user_id = ?", f.actor.ID).Update("status", "disabled"))
		}},
		{"lifecycle", func(t *testing.T, f businessFixture) {
			mustBusinessSQL(t, f.db.Model(&tenant.Tenant{}).Where("id = ?", 101).Update("status", "suspended"))
		}},
		{"policy", func(t *testing.T, f businessFixture) {
			mustBusinessSQL(t, f.db.Where("ptype = ?", "p").Delete(&database.CasbinRule{}))
		}},
		{"datascope", func(t *testing.T, f businessFixture) {
			mustBusinessSQL(t, f.db.Where("tenant_id = ?", 101).Delete(&middleware.SystemRoleDataScope{}))
		}},
		{"mode", func(t *testing.T, f businessFixture) {
			mustBusinessSQL(t, f.db.Model(&setting.SystemSetting{}).Where("setting_key = ?", tenant.FeatureFlagSettingKey).Update("setting_value", "compat"))
		}},
		{"redis", func(t *testing.T, f businessFixture) {
			if err := f.rdb.Del(context.Background(), authtoken.SessionKey(f.input.Identity.AccessToken)).Err(); err != nil {
				t.Fatal(err)
			}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newBusinessFixture(t)
			engine := gin.New()
			routes, err := contracts.TenantScopedBusinessGroup(engine.Group("/api/v1"), "/business/fresh", contracts.BusinessTenantGroupOptions{DB: f.db, RDB: f.rdb, ModeReader: f.mode, Authorization: f.reader, DataScope: true})
			if err != nil {
				t.Fatal(err)
			}
			started := make(chan struct{})
			release := make(chan struct{})
			done := make(chan struct{})
			routes.POST("/items/:id", func(c *gin.Context) {
				close(started)
				<-release
				if _, err := businessauth.Reauthorize(c, nil); err != nil {
					common.Fail(c, 403, "permission.denied")
					return
				}
				common.Success(c, "provider-secret-sentinel")
			})
			response := httptest.NewRecorder()
			req := httptest.NewRequest("POST", f.input.Action.Path, strings.NewReader(`{"actorId":999,"tenantId":202}`))
			req.Header.Set("Authorization", "Bearer "+f.input.Identity.AccessToken)
			go func() { defer close(done); engine.ServeHTTP(response, req) }()
			select {
			case <-started:
			case <-time.After(5 * time.Second):
				t.Fatal("entry not authorized")
			}
			tc.revoke(t, f)
			close(release)
			select {
			case <-done:
			case <-time.After(5 * time.Second):
				t.Fatal("recheck did not finish")
			}
			if strings.Contains(response.Body.String(), "provider-secret-sentinel") || !strings.Contains(response.Body.String(), "403") {
				t.Fatal("revocation leaked late provider result")
			}
		})
	}
}
func mustBusinessSQL(t *testing.T, result *gorm.DB) {
	t.Helper()
	if result.Error != nil {
		t.Fatal(result.Error)
	}
}

func TestBusinessAuthorizationIdentityAndDependencyFailures(t *testing.T) {
	f := newBusinessFixture(t)
	for _, identity := range []businessauth.Identity{{}, {ActorID: f.actor.ID, SessionID: "forged", AccessToken: f.input.Identity.AccessToken, ClaimTenantID: 101}, {ActorID: f.actor.ID + 1, SessionID: f.input.Identity.SessionID, AccessToken: f.input.Identity.AccessToken, ClaimTenantID: 101}, {ActorID: f.actor.ID, SessionID: f.input.Identity.SessionID, AccessToken: f.input.Identity.AccessToken, ClaimTenantID: 202}} {
		if _, err := f.reader.RefreshPrincipal(context.Background(), identity); err == nil {
			t.Fatal("identity mismatch accepted")
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := f.reader.Authorize(ctx, f.input); err == nil {
		t.Fatal("cancelled checkpoint accepted")
	}
	for _, reader := range []businessauth.Reader{NewBusinessAuthorizationReader(nil, nil, nil), NewBusinessAuthorizationReader(&Runtime{}, f.rdb, f.mode), NewBusinessAuthorizationReader(&Runtime{db: f.db, userRepo: testCredentialRepo(f.db)}, nil, f.mode)} {
		if _, err := reader.Authorize(context.Background(), f.input); err == nil {
			t.Fatal("nil dependency accepted")
		}
	}
	for _, field := range []string{"mode", "resource", "provenance"} {
		input := f.input
		switch field {
		case "mode":
			input.Resolution.Mode = "unknown"
		case "resource":
			input.ResourceTenantID = 202
		case "provenance":
			input.Resolution.ResolvedBy = "body"
		}
		if _, err := f.reader.Authorize(context.Background(), input); err == nil {
			t.Fatal("forged provenance accepted")
		}
	}
}

func TestBusinessAuthorizationCompatFallbackAndCurrentPermission(t *testing.T) {
	f := newBusinessFixture(t)
	f.globalGrant(t)
	ctx := context.Background()
	// A key checkpoint needs both current key and current actual action policy.
	input := f.input
	input.Action.Permission = "business:fresh:item:view"
	if _, err := f.reader.Authorize(ctx, input); err == nil {
		t.Fatal("route policy substituted for required permission")
	}
	var global role.SystemRole
	if err := f.db.Where("tenant_id = ? AND role_key = ?", 0, "platform_ops").Take(&global).Error; err != nil {
		t.Fatal(err)
	}
	mustBusinessSQL(t, f.db.Create(&role.SystemRolePermission{RoleID: global.ID, TenantID: 0, PermissionKey: input.Action.Permission}))
	if _, err := f.reader.Authorize(ctx, input); err != nil {
		t.Fatal("current key and route not granted")
	}
	mustBusinessSQL(t, f.db.Where("role_id = ?", global.ID).Delete(&role.SystemRolePermission{}))
	if _, err := f.reader.Authorize(ctx, input); err == nil {
		t.Fatal("revoked key stayed authorized")
	}
	// Valid compat uses global0 and does not consult membership.
	mustBusinessSQL(t, f.db.Model(&setting.SystemSetting{}).Where("setting_key = ?", tenant.FeatureFlagSettingKey).Update("setting_value", "compat"))
	mustBusinessSQL(t, f.db.Model(&tenant.Membership{}).Where("user_id = ?", f.actor.ID).Update("status", "disabled"))
	input = f.input
	input.Resolution = tenant.Context{Mode: "compat", TenantID: 0, ResolvedBy: tenant.ResolvedByCompatFallback}
	input.ResourceTenantID = 0
	if auth, err := f.reader.Authorize(ctx, input); err != nil || auth.Bindings[0].TenantID != 0 {
		t.Fatal("explicit compat didn't retain global-only semantics")
	}
	// A claim-less multi session can use declared platform fallback only if it
	// has no memberships and a real current global action grant.
	mustBusinessSQL(t, f.db.Model(&setting.SystemSetting{}).Where("setting_key = ?", tenant.FeatureFlagSettingKey).Update("setting_value", "multi"))
	data, err := authtoken.ValidateSession(ctx, f.rdb, f.input.Identity.AccessToken)
	if err != nil {
		t.Fatal("read synthetic session")
	}
	data.TenantID = 0
	if err := authtoken.StoreSession(ctx, f.rdb, f.input.Identity.AccessToken, data, time.Minute); err != nil {
		t.Fatal("restamp synthetic session")
	}
	mustBusinessSQL(t, f.db.Model(&session.SystemUserSession{}).Where("session_id = ?", data.SessionID).Update("tenant_id", 0))
	input.Identity = businessauth.IdentityFromSession(data, f.input.Identity.AccessToken)
	input.Resolution = tenant.Context{Mode: "multi", TenantID: 0, ResolvedBy: tenant.ResolvedByPlatformFallback}
	if _, err := f.reader.Authorize(ctx, input); err != nil {
		t.Fatal("valid platform fallback refused")
	}
	mustBusinessSQL(t, f.db.Model(&tenant.Membership{}).Where("user_id = ?", f.actor.ID).Update("status", "active"))
	if _, err := f.reader.Authorize(ctx, input); err == nil {
		t.Fatal("fallback shadowed active membership")
	}
}

func TestBusinessRouteHeaderAndBodyCannotAcquireIdentity(t *testing.T) {
	f := newBusinessFixture(t)
	engine := gin.New()
	routes, err := contracts.TenantScopedBusinessGroup(engine.Group("/api/v1"), "/business/fresh", contracts.BusinessTenantGroupOptions{DB: f.db, RDB: f.rdb, ModeReader: f.mode, Authorization: f.reader, DataScope: true, PlatformScope: true})
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	routes.POST("/items/:id", func(c *gin.Context) {
		calls++
		assertBusinessRouteAdoptedIdentity(t, f, c)
		common.Success(c, nil)
	})
	request := func(header string) {
		req := httptest.NewRequest("POST", f.input.Action.Path, strings.NewReader(`{"actorId":999,"tenantId":202,"resolvedBy":"header"}`))
		req.Header.Set("Authorization", "Bearer "+f.input.Identity.AccessToken)
		req.Header.Set(tenant.HeaderTenantID, header)
		engine.ServeHTTP(httptest.NewRecorder(), req)
	}
	request("")
	if calls != 1 {
		t.Fatal("valid claim refused")
	}
	assertFreshAuthorizationKeepsSQLActivity(t, f, request)
	calls-- // Keep the header refusal assertions relative to their original baseline.
	assertBusinessHeaderOverridesRefused(t, request, &calls)
	assertBusinessGlobalOverrideLifecycle(t, f, request, &calls)
}

func assertBusinessRouteAdoptedIdentity(t *testing.T, f businessFixture, c *gin.Context) {
	t.Helper()
	for _, key := range common.GetRoleKeys(c) {
		if key == "admin" || (key == "platform_ops" && tenant.FromGin(c).ResolvedBy != tenant.ResolvedByHeader) {
			t.Error("token/local role key promoted in request context")
		}
	}
	input, err := businessauth.FromGin(c)
	if err != nil || input.Identity.ActorID != f.actor.ID {
		t.Error("body replaced identity")
	}
	if input.Resolution.TenantID != 101 && input.Resolution.ResolvedBy != tenant.ResolvedByHeader {
		t.Error("body replaced tenant")
	}
}

func assertFreshAuthorizationKeepsSQLActivity(t *testing.T, f businessFixture, request func(header string)) {
	t.Helper()
	var live session.SystemUserSession
	if err := f.db.Where("session_id = ?", f.input.Identity.SessionID).Take(&live).Error; err != nil {
		t.Fatal(err)
	}
	if live.LastActivityAt == nil || time.Since(*live.LastActivityAt) > 30*time.Second {
		t.Fatal("unexpected session activity")
	}
	before := *live.LastActivityAt
	request("")
	if err := f.db.Where("session_id = ?", f.input.Identity.SessionID).Take(&live).Error; err != nil {
		t.Fatal(err)
	}
	if live.LastActivityAt == nil || !live.LastActivityAt.Equal(before) {
		t.Fatal("fresh authorization wrote SQL activity")
	}
}

func assertBusinessHeaderOverridesRefused(t *testing.T, request func(header string), calls *int) {
	t.Helper()
	for _, header := range []string{"202", "-1", "not-a-number", "18446744073709551616"} {
		request(header)
		if *calls != 1 {
			t.Fatal("untrusted override acquired scope")
		}
	}
}

func assertBusinessGlobalOverrideLifecycle(t *testing.T, f businessFixture, request func(header string), calls *int) {
	t.Helper()
	f.globalGrant(t)
	request("202")
	if *calls != 2 {
		t.Fatal("real global override required target membership")
	}
	mustBusinessSQL(t, f.db.Model(&tenant.Tenant{}).Where("id = ?", 202).Update("status", "suspended"))
	request("202")
	if *calls != 2 {
		t.Fatal("suspended override reached handler")
	}
	mustBusinessSQL(t, f.db.Model(&tenant.Membership{}).Where("user_id = ?", f.actor.ID).Update("status", "disabled"))
	request("0")
	if *calls != 2 {
		t.Fatal("header resurrected revoked claim")
	}
}
