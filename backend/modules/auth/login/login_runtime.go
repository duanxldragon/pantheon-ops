//nolint:revive // Runtime exposes a broad auth facade and interface surface.
package login

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/modules/auth/mfa"
	"github.com/duanxldragon/pantheon-base/backend/modules/auth/security"
	"github.com/duanxldragon/pantheon-base/backend/modules/auth/session"
	"github.com/duanxldragon/pantheon-base/backend/pkg/authsession"
	"github.com/duanxldragon/pantheon-base/backend/pkg/authtoken"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/contracts/authuser"
	"github.com/duanxldragon/pantheon-base/backend/pkg/database"
	"github.com/duanxldragon/pantheon-base/backend/pkg/impexp"
	"github.com/duanxldragon/pantheon-base/backend/pkg/logging"
	"github.com/duanxldragon/pantheon-base/backend/pkg/maintenance"
	"github.com/duanxldragon/pantheon-base/backend/pkg/platformprefs"
	"github.com/duanxldragon/pantheon-base/backend/pkg/tenant"

	"github.com/google/uuid"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const logMsgAuthSettingsReloadFallback = "auth settings reload: read setting failed, using fallback"

const (
	defaultPasswordMinLength       = 6
	defaultMaxFailedAttempts       = 5
	defaultLockMinutes             = 15
	defaultSourceMaxFailedAttempts = 20
	defaultSourceWindowMinutes     = 15
	defaultSourceLockMinutes       = 15
	defaultSessionIdleMinutes      = 30
	defaultMaxActiveSessions       = authsession.DefaultMaxActiveSessionsPerUser
	defaultLoginLogRetentionDays   = 90
	defaultSessionRetentionDays    = authsession.DefaultSessionRetentionDays
)

const (
	settingPasswordMinLengthKey        = "security.password_min_length"
	settingPasswordRequireDigitKey     = "security.password_require_digit"
	settingPasswordRequireUpperKey     = "security.password_require_uppercase"
	settingPasswordHistoryLimitKey     = "security.password_history_limit"
	settingPasswordExpireDaysKey       = "security.password_expire_days"
	settingMaxFailedAttemptsKey        = "login.max_failed_attempts"
	settingLockMinutesKey              = "login.lock_minutes"
	settingSourceMaxFailedAttemptsKey  = "login.source_max_failed_attempts"
	settingSourceWindowMinutesKey      = "login.source_window_minutes"
	settingSourceLockMinutesKey        = "login.source_lock_minutes"
	settingSessionIdleMinutesKey       = "login.session_idle_minutes"
	settingMaxActiveSessionsKey        = "login.max_active_sessions_per_user"
	settingLoginLogRetentionDaysKey    = "audit.login_log_retention_days"
	settingSessionRetentionDaysKey     = "audit.session_retention_days"
	settingLoginLogRetentionOptionsKey = "audit.login_log_retention_options"
	settingSessionCleanupOptionsKey    = "audit.session_cleanup_retention_options"
	settingSecurityEventEnabledKey     = "login.security_event_enabled"
	settingCaptchaEnabledKey           = "login.captcha_enabled"
	settingMFAEnabledKey               = "login.mfa_enabled"
	settingSSOEnabledKey               = "login.sso_enabled"

	settingKeyWhereClause = "setting_key = ?"
)

// Runtime is the root auth service that composes sub-domain services.
type Runtime struct {
	db *gorm.DB
	// userRepo is the credential port over system user state. It is injected by
	// the composition root (backend/cmd/server/main.go) so auth never imports
	// modules/system/iam/user (docs/designs/REPOSITORY_LAYOUT.md §8.2).
	userRepo  authuser.Repository
	tenantCtx *tenant.Context

	// Sub-services
	loginSvc    *LoginService
	mfaSvc      *mfa.Service
	securitySvc *security.Service
	sessionSvc  *session.Service

	// 账号安全策略缓存. Shared by pointer so request-scoped tenant facades
	// never copy the embedded lock (vet lock-copy rule).
	settings *runtimeSettingsState
}

// runtimeSettingsState holds the mutex-guarded account-security policy cache.
type runtimeSettingsState struct {
	mu                           sync.RWMutex
	cache                        map[string]int
	loginLogCleanupRetentionDays []int
	sessionCleanupRetentionDays  []int
}

// WithTenantContext returns a request-scoped facade for tenant-bound reads and writes.
func (s *Runtime) WithTenantContext(ctx *tenant.Context) *Runtime {
	clone := *s
	clone.tenantCtx = ctx
	clone.loginSvc = s.loginSvc.WithTenantContext(ctx)
	clone.securitySvc = s.securitySvc.WithTenantContext(ctx)
	clone.loginSvc.recorder = &clone
	return &clone
}

// NewRuntime constructs the root auth service and its sub-services.
func NewRuntime(db *gorm.DB, userRepo authuser.Repository) *Runtime {
	s := &Runtime{
		db:       db,
		userRepo: userRepo,
		settings: &runtimeSettingsState{
			cache: make(map[string]int),
		},
	}

	// Build sub-services, wiring them back through interfaces on Runtime.
	s.loginSvc = NewLoginService(db, userRepo, s, s)
	s.mfaSvc = mfa.NewService(db, s, s, s)
	s.securitySvc = security.NewService(db, userRepo, s)

	// SessionService needs Runtime to implement PolicyProvider, UserRoleLoader, TokenIssuer.
	s.sessionSvc = session.NewService(db, s, s, s)

	// 启动时同步加载一次核心设置
	_ = s.ReloadSettings()
	return s
}

// Migrate 初始化认证域表结构。
func (s *Runtime) Migrate() error {
	if s.db == nil {
		return common.ErrDatabaseNotInitialized
	}

	if err := s.db.AutoMigrate(
		&session.SystemUserSession{},
		&SystemLogLogin{},
		&SystemLoginThrottle{},
		&mfa.SystemAuthFactor{},
		&mfa.SystemAuthMFAChallenge{},
		&security.SystemAuthSecurityEvent{},
		&security.SystemUserPasswordHistory{},
	); err != nil {
		return err
	}
	return nil
}

// session.PolicyProvider — provides session policy subset.
func (s *Runtime) GetSessionPolicy() session.AuthRuntimePolicy {
	policy := s.getAuthRuntimePolicy()
	return session.AuthRuntimePolicy{
		SessionIdleMinutes:   policy.SessionIdleMinutes,
		SessionRetentionDays: policy.SessionRetentionDays,
		MaxActiveSessions:    policy.MaxActiveSessions,
		CleanupRetentionDays: cloneIntSlice(policy.SessionCleanupRetentionDays),
	}
}

// session.UserRoleLoader — provides user+role lookups.
func (s *Runtime) GetUserByID(userID uint64) (*session.UserRef, error) {
	if s.db == nil {
		return nil, common.ErrDatabaseNotInitialized
	}
	u, err := s.userRepo.FindByID(context.Background(), userID)
	if err != nil {
		return nil, err
	}
	return &session.UserRef{ID: u.ID, Username: u.Username}, nil
}

// session.TokenIssuer — issues token pairs for session operations.
func (s *Runtime) IssueTokenPair(userID uint64, username string, roles []string, sess *session.SystemUserSession) (*authtoken.Pair, error) {
	return s.IssueTokenPairWithContext(context.Background(), userID, username, roles, sess)
}

func (s *Runtime) IssueTokenPairWithContext(ctx context.Context, userID uint64, username string, roles []string, sess *session.SystemUserSession) (*authtoken.Pair, error) {
	return s.issueTokenPairForSession(ctx, userID, username, roles, sess)
}

// ─────────────────────────────────────────────────────────────
// login.PolicyProvider implementation
// ─────────────────────────────────────────────────────────────
func (s *Runtime) GetRuntimePolicy() RuntimePolicy {
	s.settings.mu.RLock()
	defer s.settings.mu.RUnlock()
	return RuntimePolicy{
		MaxFailedAttempts:            s.settings.cache[settingMaxFailedAttemptsKey],
		LockMinutes:                  s.settings.cache[settingLockMinutesKey],
		SourceMaxFailedAttempts:      s.settings.cache[settingSourceMaxFailedAttemptsKey],
		SourceWindowMinutes:          s.settings.cache[settingSourceWindowMinutesKey],
		SourceLockMinutes:            s.settings.cache[settingSourceLockMinutesKey],
		SecurityEventEnabled:         s.settings.cache[settingSecurityEventEnabledKey] == 1,
		LoginLogRetentionDays:        s.settings.cache[settingLoginLogRetentionDaysKey],
		LoginLogCleanupRetentionDays: cloneIntSlice(s.settings.loginLogCleanupRetentionDays),
	}
}

// ─────────────────────────────────────────────────────────────
// login.SecurityEventRecorder implementation
// ─────────────────────────────────────────────────────────────
func (s *Runtime) RecordSecurityEvent(event security.SystemAuthSecurityEvent) {
	if s.db == nil || !s.getSecurityEventEnabled() {
		return
	}
	event.SourceKey = strings.TrimSpace(event.SourceKey)
	event.Username = strings.TrimSpace(event.Username)
	if event.Severity == "" {
		event.Severity = "medium"
	}
	if s.tenantCtx != nil && s.tenantCtx.IsMulti() {
		event.TenantID = s.tenantCtx.TenantID
	}
	if err := s.db.Create(&event).Error; err != nil {
		logging.Warn("record security event failed",
			zap.String("event_type", event.EventType), zap.Error(err))
	}
}

// ─────────────────────────────────────────────────────────────
// security.PolicyProvider implementation
// ─────────────────────────────────────────────────────────────
func (s *Runtime) GetAuthRuntimePolicy() security.AuthRuntimePolicy {
	internal := s.getAuthRuntimePolicy()
	return security.AuthRuntimePolicy{
		PasswordMinLength:    internal.PasswordMinLength,
		PasswordRequireDigit: internal.PasswordRequireDigit,
		PasswordRequireUpper: internal.PasswordRequireUpper,
		PasswordHistoryLimit: internal.PasswordHistoryLimit,
		PasswordExpireDays:   internal.PasswordExpireDays,
		MaxFailedAttempts:    internal.MaxFailedAttempts,
		LockMinutes:          internal.LockMinutes,

		SourceMaxFailedAttempts: internal.SourceMaxFailedAttempts,
		SourceWindowMinutes:     internal.SourceWindowMinutes,
		SourceLockMinutes:       internal.SourceLockMinutes,
		SessionIdleMinutes:      internal.SessionIdleMinutes,
		MaxActiveSessions:       internal.MaxActiveSessions,
		SessionRetentionDays:    internal.SessionRetentionDays,

		SecurityEventEnabled: internal.SecurityEventEnabled,
		CaptchaEnabled:       internal.CaptchaEnabled,
		MFAEnabled:           internal.MFAEnabled,
		SSOEnabled:           internal.SSOEnabled,
	}
}

// ─────────────────────────────────────────────────────────────
// mfa.PolicyProvider implementation
// ─────────────────────────────────────────────────────────────
func (s *Runtime) IsMFAEnabled() bool {
	s.settings.mu.RLock()
	defer s.settings.mu.RUnlock()
	return s.settings.cache[settingMFAEnabledKey] == 1
}

// ─────────────────────────────────────────────────────────────
// mfa.IdentityProvider implementation
// ─────────────────────────────────────────────────────────────
func (s *Runtime) LoadUserByID(userID uint64) (*mfa.UserRecord, error) {
	u, err := s.userRepo.FindByID(context.Background(), userID)
	if err != nil {
		return nil, err
	}
	return &mfa.UserRecord{
		ID:       u.ID,
		Username: u.Username,
		Nickname: u.Nickname,
		Avatar:   u.Avatar,
		Email:    u.Email,
		Phone:    u.Phone,
		Status:   u.Status,
	}, nil
}

func (s *Runtime) GetUserRoles(userID uint64) ([]string, error) {
	var roles []string
	err := s.db.Table("system_role").
		Select("system_role.role_key").
		Joins("JOIN system_user_role ON system_user_role.role_id = system_role.id").
		Where("system_user_role.user_id = ? AND system_role.status = ?", userID, common.StatusEnabled).
		Pluck("system_role.role_key", &roles).Error
	return roles, err
}

// ─────────────────────────────────────────────────────────────
// mfa.SessionCreator implementation
// ─────────────────────────────────────────────────────────────
func (s *Runtime) CreateSession(userID uint64, roles []string, ip, userAgent string) (*authtoken.Pair, error) {
	return s.CreateSessionWithContext(context.Background(), userID, roles, ip, userAgent)
}

func (s *Runtime) CreateSessionWithContext(ctx context.Context, userID uint64, roles []string, ip, userAgent string) (*authtoken.Pair, error) {
	return s.CreateSessionForTenantWithContext(ctx, userID, roles, ip, userAgent, 0)
}

// CreateSessionForTenantWithContext creates a session stamped with the given
// explicit tenant choice (0 = auto-discovery). The choice is still validated
// by the issuance gate — callers cannot stamp a claim they lack membership for.
func (s *Runtime) CreateSessionForTenantWithContext(ctx context.Context, userID uint64, roles []string, ip, userAgent string, tenantChoice uint64) (*authtoken.Pair, error) {
	if s.db == nil {
		return nil, common.ErrDatabaseNotInitialized
	}
	if ctx == nil {
		ctx = context.Background()
	}
	policy := s.getAuthRuntimePolicy()
	now := time.Now()

	// Tenant discovery (contract §3.1 source 2): multi mode resolves the
	// login tenant from the explicit choice or active membership;
	// ambiguous/erroneous membership state denies login. Compat mode returns
	// no claim (zero behavior change).
	tenantClaim, err := s.resolveLoginTenantClaim(userID, tenantChoice)
	if err != nil {
		return nil, err
	}

	// Per-user overflow pruning stays inline: it is bounded by
	// MaxActiveSessions and is part of this user's own login correctness, not a
	// table-wide sweep. The global inventory sweep is a maintenance task.
	if err := authsession.CleanupUserOverflowSessions(s.db, userID, now, policy.SessionIdleMinutes, maxInt(policy.MaxActiveSessions-1, 0)); err != nil {
		return nil, err
	}

	u, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	sess := session.SystemUserSession{
		SessionID:        uuid.NewString(),
		UserID:           u.ID,
		RefreshJTI:       uuid.NewString(),
		RefreshExpiresAt: now.Add(authtoken.RefreshTokenTTL),
		LastActivityAt:   &now,
		LastIP:           ip,
		UserAgent:        session.TruncateString(userAgent, 255),
		TenantID:         tenantClaim,
	}
	if err := s.db.Create(&sess).Error; err != nil {
		return nil, err
	}
	pair, err := s.issueTenantTokenPair(ctx, u.ID, u.Username, roles, &sess, tenantClaim)
	if err != nil {
		// Token issuance failed after the session row was persisted; remove it so
		// failed logins don't accumulate orphan sessions in the governance views.
		if delErr := s.db.Delete(&sess).Error; delErr != nil {
			logging.Warn("cleanup orphan session after token issuance failure failed",
				zap.String("sessionId", sess.SessionID), zap.Error(delErr))
		}
		return nil, err
	}
	return pair, nil
}

// resolveLoginTenantClaim discovers the tenant claim to stamp into the new
// session (contract §3.1). Compat mode: always 0. Multi mode:
//   - explicit tenantChoice > 0 wins and is validated through the issuance
//     gate (active membership + active tenant master) — this is the picker
//     path for multi-membership users (slice 2);
//   - without a choice, single-membership subjects resolve deterministically;
//   - zero memberships => 0 (the subject logs into the platform population);
//   - ambiguous memberships without a choice deny (never silently narrowed).
func (s *Runtime) resolveLoginTenantClaim(userID, tenantChoice uint64) (uint64, error) {
	mode := tenant.NormalizeMode(tenant.FeatureFlagSettingKeyReader(s.db))
	if mode != tenant.ModeMulti {
		// Compat ignores any requested tenant: no claim is ever stamped.
		return 0, nil
	}
	if tenantChoice != 0 {
		return tenant.GateSessionIssuance(s.db, tenant.IssuanceCheckInput{
			Mode:     mode,
			UserID:   userID,
			TenantID: tenantChoice,
		})
	}
	discovery, err := tenant.DiscoverDefaultTenant(s.db, userID)
	if err != nil {
		return 0, err
	}
	if !discovery.Resolved {
		return 0, nil
	}
	// Gate: membership + tenant master status must both hold (contract §5).
	return tenant.GateSessionIssuance(s.db, tenant.IssuanceCheckInput{
		Mode:     mode,
		UserID:   userID,
		TenantID: discovery.TenantID,
	})
}

// ─────────────────────────────────────────────────────────────
// security.PolicyProvider implementation
// ─────────────────────────────────────────────────────────────
func (s *Runtime) GetSecurityRuntimePolicy() security.AuthRuntimePolicy {
	return s.GetAuthRuntimePolicy()
}

// ─────────────────────────────────────────────────────────────
// Runtime facade methods used by HTTP handlers and module callers.
// ─────────────────────────────────────────────────────────────

func (s *Runtime) Login(req *LoginReq) (*authuser.User, error) {
	return s.loginSvc.Authenticate(req)
}

func (s *Runtime) LoginWithSource(req *LoginReq, sourceKey string) (*authuser.User, error) {
	return s.loginSvc.AuthenticateWithSource(req, sourceKey)
}

func (s *Runtime) Authenticate(req *LoginReq) (*authuser.User, error) {
	return s.Login(req)
}

func (s *Runtime) ListOwnLoginLogs(username string, query *LoginLogQuery) (*LoginLogPageResp, error) {
	return s.loginSvc.ListOwnLoginLogs(username, query)
}

func (s *Runtime) ListLoginLogs(query *LoginLogQuery) (*LoginLogPageResp, error) {
	_ = s.ReloadSettings()
	return s.loginSvc.ListLoginLogs(query)
}

func (s *Runtime) ExportLoginLogs(query *LoginLogQuery) (*impexp.CSVFile, error) {
	_ = s.ReloadSettings()
	return s.loginSvc.ExportLoginLogs(query)
}

func (s *Runtime) CleanupLoginLogs(retentionDays int, startedAt, endedAt string) (int64, error) {
	_ = s.ReloadSettings()
	return s.loginSvc.CleanupLoginLogs(retentionDays, startedAt, endedAt)
}

func (s *Runtime) BatchDeleteLoginLogs(ids []uint64) (int64, error) {
	return s.loginSvc.BatchDeleteLoginLogs(ids)
}

func (s *Runtime) RecordLoginLog(requestID, username, ip, browser, os string, status int, msg string) {
	_ = s.ReloadSettings()
	s.loginSvc.RecordLoginLog(requestID, username, ip, browser, os, status, msg)
}

func (s *Runtime) CreateMFAChallenge(currentUser *authuser.User) (*mfa.MFAChallengeResp, error) {
	return s.CreateMFAChallengeForTenant(currentUser, 0)
}

func (s *Runtime) CreateMFAChallengeForTenant(currentUser *authuser.User, tenantID uint64) (*mfa.MFAChallengeResp, error) {
	mfaUser := &mfa.UserRecord{
		ID:       currentUser.ID,
		Username: currentUser.Username,
		Nickname: currentUser.Nickname,
		Avatar:   currentUser.Avatar,
		Email:    currentUser.Email,
		Phone:    currentUser.Phone,
		Status:   currentUser.Status,
	}
	return s.mfaSvc.CreateChallengeForTenant(mfaUser, tenantID)
}

func (s *Runtime) VerifyMFAChallenge(req *MFAVerifyReq, ip, userAgent string) (*AuthTokenResp, error) {
	return s.VerifyMFAChallengeWithContext(context.Background(), req, ip, userAgent)
}

func (s *Runtime) VerifyMFAChallengeWithContext(ctx context.Context, req *MFAVerifyReq, ip, userAgent string) (*AuthTokenResp, error) {
	result, err := s.mfaSvc.VerifyChallengeWithContext(ctx, req, ip, userAgent)
	if err != nil {
		return nil, err
	}
	return &AuthTokenResp{
		Token:            result.TokenPair.AccessToken,
		AccessToken:      result.TokenPair.AccessToken,
		RefreshToken:     result.TokenPair.RefreshToken,
		TokenType:        result.TokenPair.TokenType,
		AccessExpiresAt:  result.TokenPair.AccessExpiresAt.Format("2006-01-02 15:04:05"),
		RefreshExpiresAt: result.TokenPair.RefreshExpiresAt.Format("2006-01-02 15:04:05"),
		SessionID:        result.TokenPair.SessionID,
		User: &security.UserInfoResp{
			ID:       result.UserID,
			Username: result.Username,
			Nickname: result.Nickname,
			Avatar:   result.Avatar,
			Email:    result.Email,
			Phone:    result.Phone,
			Roles:    result.Roles,
			Perms:    result.Perms,
		},
	}, nil
}

func (s *Runtime) RefreshSession(sessionID string, userID uint64, ip, userAgent string) (*authtoken.Pair, error) {
	return s.RefreshSessionWithContext(context.Background(), sessionID, userID, ip, userAgent)
}

func (s *Runtime) RefreshSessionWithContext(ctx context.Context, sessionID string, userID uint64, ip, userAgent string) (*authtoken.Pair, error) {
	return s.sessionSvc.RefreshSessionWithContext(ctx, sessionID, userID, ip, userAgent)
}

func (s *Runtime) RevokeSession(sessionID string) error {
	return s.sessionSvc.RevokeSession(sessionID)
}

func (s *Runtime) TouchSessionActivity(sessionID string, userID uint64, ip, userAgent string) error {
	return s.sessionSvc.TouchSessionActivity(sessionID, userID, ip, userAgent)
}

func (s *Runtime) ListSessions(userID uint64, currentSessionID string) ([]SessionResp, error) {
	return s.sessionSvc.ListSessions(userID, currentSessionID)
}

func (s *Runtime) RevokeOwnedSession(userID uint64, currentSessionID, targetSessionID string) error {
	return s.sessionSvc.RevokeOwnedSession(userID, currentSessionID, targetSessionID)
}

func (s *Runtime) CleanupHistoricSessions(retentionDays int, startedAt, endedAt string) (int64, error) {
	_ = s.ReloadSettings()
	return s.sessionSvc.CleanupHistoricSessions(retentionDays, startedAt, endedAt)
}

func (s *Runtime) BatchRevokeSessions(currentSessionID string, sessionIDs []string) (int64, error) {
	return s.sessionSvc.BatchRevokeSessions(currentSessionID, sessionIDs)
}

func (s *Runtime) ListAllSessions(query *AdminSessionQuery) (*AdminSessionPageResp, error) {
	return s.sessionSvc.ListAllSessions(query)
}

func (s *Runtime) RevokeAnySession(currentSessionID, targetSessionID string) error {
	return s.sessionSvc.RevokeAnySession(currentSessionID, targetSessionID)
}

func (s *Runtime) GetUserPerms(userID uint64) ([]string, error) {
	if s.db == nil {
		return nil, common.ErrDatabaseNotInitialized
	}

	// Try cache first (5-minute TTL)
	cacheKey := fmt.Sprintf("user_perms:%d", userID)
	if database.RDB != nil {
		cached, err := database.RDB.Get(context.Background(), cacheKey).Result()
		if err == nil && cached != "" {
			var perms []string
			if json.Unmarshal([]byte(cached), &perms) == nil {
				return perms, nil
			}
		}
	}

	// Cache miss or Redis unavailable: query DB
	var permissionKeys []string
	err := s.db.Table("system_role_permission").
		Select("DISTINCT system_role_permission.permission_key").
		Joins("JOIN system_user_role ON system_user_role.role_id = system_role_permission.role_id").
		Where("system_user_role.user_id = ? AND system_role_permission.permission_key <> ''", userID).
		Pluck("system_role_permission.permission_key", &permissionKeys).Error

	perms := mergePermissionKeys(permissionKeys)

	// Cache for 5 minutes
	if database.RDB != nil && len(perms) > 0 && err == nil {
		if data, marshalErr := json.Marshal(perms); marshalErr == nil {
			database.RDB.Set(context.Background(), cacheKey, data, 5*time.Minute)
		}
	}

	return perms, err
}

func (s *Runtime) GetCurrentUserInfo(userID uint64) (*UserInfoResp, error) {
	if s.db == nil {
		return nil, common.ErrDatabaseNotInitialized
	}
	u, err := s.userRepo.FindByID(context.Background(), userID)
	if err != nil {
		return nil, err
	}
	roles, err := s.GetUserRoles(u.ID)
	if err != nil {
		return nil, err
	}
	perms, err := s.GetUserPerms(u.ID)
	if err != nil {
		return nil, err
	}
	return &UserInfoResp{
		ID:          u.ID,
		Username:    u.Username,
		Nickname:    u.Nickname,
		Avatar:      u.Avatar,
		Email:       u.Email,
		Phone:       u.Phone,
		Roles:       roles,
		Perms:       perms,
		Preferences: platformprefs.Parse(u.PreferenceJSON),
	}, nil
}

func (s *Runtime) UpdateCurrentUserPreferences(userID uint64, req *UserPlatformPreferenceUpdateReq) (*UserPreferenceUpdateResult, error) {
	if s.db == nil {
		return nil, common.ErrDatabaseNotInitialized
	}
	u, err := s.userRepo.FindByID(context.Background(), userID)
	if err != nil {
		return nil, err
	}
	previousPreferences := platformprefs.Parse(u.PreferenceJSON)
	nextPreferences := platformprefs.Normalize(&platformprefs.PlatformPreference{
		Theme:       req.Theme,
		Language:    req.Language,
		LayoutMode:  req.LayoutMode,
		DensityMode: req.DensityMode,
	})
	preferenceJSON, err := platformprefs.Marshal(nextPreferences)
	if err != nil {
		return nil, err
	}
	if preferenceJSON != u.PreferenceJSON {
		if err := s.userRepo.UpdatePreferenceJSON(context.Background(), userID, preferenceJSON); err != nil {
			return nil, err
		}
	}
	userInfo, err := s.GetCurrentUserInfo(userID)
	if err != nil {
		return nil, err
	}
	return &UserPreferenceUpdateResult{
		User:      userInfo,
		Previous:  previousPreferences,
		Current:   nextPreferences,
		Persisted: preferenceJSON,
	}, nil
}

func (s *Runtime) VerifyPasswordForOperation(userID uint64, sessionID, password string) (string, error) {
	return s.securitySvc.VerifyPasswordForOperation(userID, sessionID, password)
}

func (s *Runtime) VerifyPasswordForOperationWithContext(ctx context.Context, userID uint64, sessionID, password string) (string, error) {
	return s.securitySvc.VerifyPasswordForOperationWithContext(ctx, userID, sessionID, password)
}

func (s *Runtime) UpdatePassword(userID uint64, currentSessionID string, req *PasswordUpdateReq) error {
	secReq := &security.PasswordChangeReq{
		OldPassword: req.OldPassword,
		NewPassword: req.NewPassword,
	}
	return s.securitySvc.UpdatePassword(userID, currentSessionID, secReq)
}

func (s *Runtime) GetSecurityOverview(userID uint64, username, currentSessionID string) (*SecurityOverviewResp, error) {
	if s.db == nil {
		return nil, common.ErrDatabaseNotInitialized
	}
	// Read path: session inventory governance is a registered maintenance
	// task, never an inline sweep while rendering this overview.
	policy := s.getAuthRuntimePolicy()
	now := time.Now()

	info, err := s.GetCurrentUserInfo(userID)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(username) == "" {
		username = info.Username
	}

	sessions, err := s.ListSessions(userID, currentSessionID)
	if err != nil {
		return nil, err
	}
	var currentSession *SessionResp
	for i := range sessions {
		if sessions[i].IsCurrent {
			s := sessions[i]
			currentSession = &s
			break
		}
	}

	activeSessionCount, err := s.securitySvc.CountActiveSessions(userID, now)
	if err != nil {
		return nil, err
	}

	var lastLoginAt *string
	var lastLogin SystemLogLogin
	err = s.db.Where("username = ? AND status = ?", username, common.LoginStatusSuccess).
		Order(loginTimeDescOrderClause).
		First(&lastLogin).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if err == nil {
		formatted := lastLogin.LoginTime.Format(time.RFC3339)
		lastLoginAt = &formatted
	}

	secEvents := s.securitySvc.ListRecentSecurityEvents(userID, 5)

	return &SecurityOverviewResp{
		User:                 info,
		CurrentSession:       currentSession,
		ActiveSessionCount:   activeSessionCount,
		LastLoginAt:          lastLoginAt,
		PasswordExpired:      s.securitySvc.IsPasswordExpired(userID),
		PasswordExpiresAt:    s.securitySvc.GetPasswordExpiresAt(userID),
		RecentSecurityEvents: toSecurityEventRespList(secEvents),
		Policy: SecurityPolicyResp{
			PasswordMinLength:       policy.PasswordMinLength,
			PasswordRequireDigit:    policy.PasswordRequireDigit,
			PasswordRequireUpper:    policy.PasswordRequireUpper,
			PasswordHistoryLimit:    policy.PasswordHistoryLimit,
			PasswordExpireDays:      policy.PasswordExpireDays,
			MaxFailedAttempts:       policy.MaxFailedAttempts,
			LockMinutes:             policy.LockMinutes,
			SourceMaxFailedAttempts: policy.SourceMaxFailedAttempts,
			SourceWindowMinutes:     policy.SourceWindowMinutes,
			SourceLockMinutes:       policy.SourceLockMinutes,
			SessionIdleMinutes:      policy.SessionIdleMinutes,
			MaxActiveSessions:       policy.MaxActiveSessions,
			SessionRetentionDays:    policy.SessionRetentionDays,
			CaptchaEnabled:          policy.CaptchaEnabled,
			MFAEnabled:              policy.MFAEnabled,
			SSOEnabled:              policy.SSOEnabled,
		},
	}, nil
}

func (s *Runtime) ListSecurityEvents(query *SecurityEventQuery) (*SecurityEventPageResp, error) {
	return s.securitySvc.ListSecurityEvents(query)
}

func (s *Runtime) AcknowledgeSecurityEvent(eventID, actorID uint64, actorUsername, note string) error {
	return s.securitySvc.AcknowledgeSecurityEvent(eventID, actorID, actorUsername, note)
}

func (s *Runtime) BatchAcknowledgeSecurityEvents(eventIDs []uint64, actorID uint64, actorUsername, note string) (int64, error) {
	return s.securitySvc.BatchAcknowledgeSecurityEvents(eventIDs, actorID, actorUsername, note)
}

func (s *Runtime) CleanupSecurityEvents(retentionDays int, startedAt, endedAt string) (int64, error) {
	return s.securitySvc.CleanupSecurityEvents(retentionDays, startedAt, endedAt)
}

// ─────────────────────────────────────────────────────────────
// Settings management
// ─────────────────────────────────────────────────────────────

func (s *Runtime) ReloadSettings() error {
	policy := authRuntimePolicy{
		PasswordMinLength:            s.fetchSettingIntFromDB(settingPasswordMinLengthKey, defaultPasswordMinLength),
		PasswordRequireDigit:         s.fetchSettingBoolFromDB(settingPasswordRequireDigitKey, false),
		PasswordRequireUpper:         s.fetchSettingBoolFromDB(settingPasswordRequireUpperKey, false),
		PasswordHistoryLimit:         s.fetchSettingIntFromDB(settingPasswordHistoryLimitKey, 0),
		PasswordExpireDays:           s.fetchSettingIntFromDB(settingPasswordExpireDaysKey, 0),
		MaxFailedAttempts:            s.fetchSettingIntFromDB(settingMaxFailedAttemptsKey, defaultMaxFailedAttempts),
		LockMinutes:                  s.fetchSettingIntFromDB(settingLockMinutesKey, defaultLockMinutes),
		SourceMaxFailedAttempts:      s.fetchSettingIntFromDB(settingSourceMaxFailedAttemptsKey, defaultSourceMaxFailedAttempts),
		SourceWindowMinutes:          s.fetchSettingIntFromDB(settingSourceWindowMinutesKey, defaultSourceWindowMinutes),
		SourceLockMinutes:            s.fetchSettingIntFromDB(settingSourceLockMinutesKey, defaultSourceLockMinutes),
		SessionIdleMinutes:           s.fetchSettingIntFromDB(settingSessionIdleMinutesKey, defaultSessionIdleMinutes),
		MaxActiveSessions:            s.fetchSettingIntFromDB(settingMaxActiveSessionsKey, defaultMaxActiveSessions),
		LoginLogRetentionDays:        s.fetchSettingIntFromDB(settingLoginLogRetentionDaysKey, defaultLoginLogRetentionDays),
		SessionRetentionDays:         s.fetchSettingIntFromDB(settingSessionRetentionDaysKey, defaultSessionRetentionDays),
		LoginLogCleanupRetentionDays: s.fetchSettingIntSliceFromDB(settingLoginLogRetentionOptionsKey, defaultCleanupRetentionDays()),
		SessionCleanupRetentionDays:  s.fetchSettingIntSliceFromDB(settingSessionCleanupOptionsKey, defaultCleanupRetentionDays()),
		SecurityEventEnabled:         s.fetchSettingBoolFromDB(settingSecurityEventEnabledKey, true),
		CaptchaEnabled:               s.fetchSettingBoolFromDB(settingCaptchaEnabledKey, false),
		MFAEnabled:                   s.fetchSettingBoolFromDB(settingMFAEnabledKey, false),
		SSOEnabled:                   s.fetchSettingBoolFromDB(settingSSOEnabledKey, false),
	}

	s.settings.mu.Lock()
	s.settings.cache[settingPasswordMinLengthKey] = policy.PasswordMinLength
	s.settings.cache[settingPasswordRequireDigitKey] = boolToInt(policy.PasswordRequireDigit)
	s.settings.cache[settingPasswordRequireUpperKey] = boolToInt(policy.PasswordRequireUpper)
	s.settings.cache[settingPasswordHistoryLimitKey] = policy.PasswordHistoryLimit
	s.settings.cache[settingPasswordExpireDaysKey] = policy.PasswordExpireDays
	s.settings.cache[settingMaxFailedAttemptsKey] = policy.MaxFailedAttempts
	s.settings.cache[settingLockMinutesKey] = policy.LockMinutes
	s.settings.cache[settingSourceMaxFailedAttemptsKey] = policy.SourceMaxFailedAttempts
	s.settings.cache[settingSourceWindowMinutesKey] = policy.SourceWindowMinutes
	s.settings.cache[settingSourceLockMinutesKey] = policy.SourceLockMinutes
	s.settings.cache[settingSessionIdleMinutesKey] = policy.SessionIdleMinutes
	s.settings.cache[settingMaxActiveSessionsKey] = policy.MaxActiveSessions
	s.settings.cache[settingLoginLogRetentionDaysKey] = policy.LoginLogRetentionDays
	s.settings.cache[settingSessionRetentionDaysKey] = policy.SessionRetentionDays
	s.settings.loginLogCleanupRetentionDays = cloneIntSlice(policy.LoginLogCleanupRetentionDays)
	s.settings.sessionCleanupRetentionDays = cloneIntSlice(policy.SessionCleanupRetentionDays)
	s.settings.cache[settingSecurityEventEnabledKey] = boolToInt(policy.SecurityEventEnabled)
	s.settings.cache[settingCaptchaEnabledKey] = boolToInt(policy.CaptchaEnabled)
	s.settings.cache[settingMFAEnabledKey] = boolToInt(policy.MFAEnabled)
	s.settings.cache[settingSSOEnabledKey] = boolToInt(policy.SSOEnabled)
	s.settings.mu.Unlock()

	return nil
}

func (s *Runtime) WatchSettings() {
	if database.RDB == nil {
		return
	}
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("settings watcher goroutine panic", "panic", r)
			}
		}()
		s.watchSettingsLoop()
	}()
}

func (s *Runtime) watchSettingsLoop() {
	for {
		pubsub := database.RDB.Subscribe(context.TODO(), "settings:refresh")
		for msg := range pubsub.Channel() {
			_ = msg // consume message
			_ = s.ReloadSettings()
		}
		// Channel closed (Redis disconnect or timeout), reconnect after delay
		_ = pubsub.Close()
		slog.Warn("settings pubsub channel closed, reconnecting in 5s")
		time.Sleep(5 * time.Second)
	}
}

func (s *Runtime) getAuthRuntimePolicy() authRuntimePolicy {
	s.settings.mu.RLock()
	defer s.settings.mu.RUnlock()

	return authRuntimePolicy{
		PasswordMinLength:            s.settings.cache[settingPasswordMinLengthKey],
		PasswordRequireDigit:         s.settings.cache[settingPasswordRequireDigitKey] == 1,
		PasswordRequireUpper:         s.settings.cache[settingPasswordRequireUpperKey] == 1,
		PasswordHistoryLimit:         s.settings.cache[settingPasswordHistoryLimitKey],
		PasswordExpireDays:           s.settings.cache[settingPasswordExpireDaysKey],
		MaxFailedAttempts:            s.settings.cache[settingMaxFailedAttemptsKey],
		LockMinutes:                  s.settings.cache[settingLockMinutesKey],
		SourceMaxFailedAttempts:      s.settings.cache[settingSourceMaxFailedAttemptsKey],
		SourceWindowMinutes:          s.settings.cache[settingSourceWindowMinutesKey],
		SourceLockMinutes:            s.settings.cache[settingSourceLockMinutesKey],
		SessionIdleMinutes:           s.settings.cache[settingSessionIdleMinutesKey],
		MaxActiveSessions:            s.settings.cache[settingMaxActiveSessionsKey],
		LoginLogRetentionDays:        s.settings.cache[settingLoginLogRetentionDaysKey],
		SessionRetentionDays:         s.settings.cache[settingSessionRetentionDaysKey],
		LoginLogCleanupRetentionDays: cloneIntSlice(s.settings.loginLogCleanupRetentionDays),
		SessionCleanupRetentionDays:  cloneIntSlice(s.settings.sessionCleanupRetentionDays),
		SecurityEventEnabled:         s.settings.cache[settingSecurityEventEnabledKey] == 1,
		CaptchaEnabled:               s.settings.cache[settingCaptchaEnabledKey] == 1,
		MFAEnabled:                   s.settings.cache[settingMFAEnabledKey] == 1,
		SSOEnabled:                   s.settings.cache[settingSSOEnabledKey] == 1,
	}
}

func (s *Runtime) getSecurityEventEnabled() bool {
	s.settings.mu.RLock()
	defer s.settings.mu.RUnlock()
	return s.settings.cache[settingSecurityEventEnabledKey] == 1
}

func (s *Runtime) fetchSettingIntFromDB(settingKey string, fallback int) int {
	if s.db == nil {
		return fallback
	}
	var rawValue string
	err := s.db.Table("system_setting").
		Select("setting_value").
		Where(settingKeyWhereClause, settingKey).
		Limit(1).
		Pluck("setting_value", &rawValue).Error
	if err != nil {
		logging.Warn(logMsgAuthSettingsReloadFallback,
			zap.String("setting_key", settingKey), zap.Error(err))
		return fallback
	}
	val, err := strconv.Atoi(strings.TrimSpace(rawValue))
	if err != nil || val <= 0 {
		return fallback
	}
	return val
}

func (s *Runtime) fetchSettingBoolFromDB(settingKey string, fallback bool) bool {
	if s.db == nil {
		return fallback
	}
	var rawValue string
	err := s.db.Table("system_setting").
		Select("setting_value").
		Where(settingKeyWhereClause, settingKey).
		Limit(1).
		Pluck("setting_value", &rawValue).Error
	if err != nil {
		logging.Warn(logMsgAuthSettingsReloadFallback,
			zap.String("setting_key", settingKey), zap.Error(err))
		return fallback
	}
	parsed, err := strconv.ParseBool(strings.TrimSpace(rawValue))
	if err != nil {
		return fallback
	}
	return parsed
}

func (s *Runtime) fetchSettingIntSliceFromDB(settingKey string, fallback []int) []int {
	if s.db == nil {
		return cloneIntSlice(fallback)
	}
	var rawValue string
	err := s.db.Table("system_setting").
		Select("setting_value").
		Where(settingKeyWhereClause, settingKey).
		Limit(1).
		Pluck("setting_value", &rawValue).Error
	if err != nil {
		logging.Warn(logMsgAuthSettingsReloadFallback,
			zap.String("setting_key", settingKey), zap.Error(err))
		return cloneIntSlice(fallback)
	}
	return normalizeRetentionDays(rawValue, fallback)
}

// RegisterMaintenanceTasks registers the auth-domain periodic housekeeping
// (session inventory governance, login-log retention, security-event
// retention) with the background maintenance registry. Registration is
// idempotent, so repeated module wiring cannot duplicate sweeps.
func (s *Runtime) RegisterMaintenanceTasks(reg *maintenance.Registry) {
	s.sessionSvc.RegisterMaintenanceTasks(reg)
	s.loginSvc.RegisterMaintenanceTasks(reg)
	s.securitySvc.RegisterMaintenanceTasks(reg)
}

// issueTenantTokenPair stores the session/refresh pair with the resolved
// tenant claim. claim=0 keeps the legacy payload shape (compat behavior).
func (s *Runtime) issueTenantTokenPair(ctx context.Context, userID uint64, username string, roles []string, sess *session.SystemUserSession, tenantClaim uint64) (*authtoken.Pair, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	return storeSessionTokenPair(ctx, userID, username, roles, sess, tenantClaim)
}

func (s *Runtime) issueTokenPairForSession(ctx context.Context, userID uint64, username string, roles []string, sess *session.SystemUserSession) (*authtoken.Pair, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	// Refresh carries the session's original claim (no re-discovery): the
	// membership re-check for the claim happened in RefreshSessionWithContext
	// before reaching here; rotating to a different tenant claim silently
	// would break session↔tenant auditability.
	return storeSessionTokenPair(ctx, userID, username, roles, sess, tenant.SessionTenantClaim(sess.TenantID))
}

// storeSessionTokenPair generates and stores the access/refresh pair for a
// session with the given tenant claim, rotating the session's refresh expiry.
func storeSessionTokenPair(ctx context.Context, userID uint64, username string, roles []string, sess *session.SystemUserSession, tenantClaim uint64) (*authtoken.Pair, error) {
	accessToken := authtoken.NewAccessToken()
	refreshToken := authtoken.NewRefreshToken()
	now := time.Now()
	accessTTL := authtoken.AccessTokenTTL
	refreshTTL := authtoken.RefreshTokenTTL

	accessData := &authtoken.SessionData{
		UserID:         userID,
		Username:       username,
		RoleKeys:       roles,
		SessionID:      sess.SessionID,
		LastActivityAt: now.Unix(),
		TenantID:       tenantClaim,
	}
	if err := authtoken.StoreSession(ctx, database.RDB, accessToken, accessData, accessTTL); err != nil {
		return nil, err
	}
	if err := authtoken.StoreRefresh(ctx, database.RDB, refreshToken, userID, sess.SessionID, refreshTTL); err != nil {
		return nil, err
	}
	sess.RefreshExpiresAt = now.Add(refreshTTL)
	return &authtoken.Pair{
		AccessToken:      accessToken,
		RefreshToken:     refreshToken,
		TokenType:        authtoken.TypeAccess,
		AccessExpiresAt:  now.Add(accessTTL),
		RefreshExpiresAt: now.Add(refreshTTL),
		SessionID:        sess.SessionID,
	}, nil
}

// ─────────────────────────────────────────────────────────────
// Internal helpers
// ─────────────────────────────────────────────────────────────

type authRuntimePolicy struct {
	PasswordMinLength            int
	PasswordRequireDigit         bool
	PasswordRequireUpper         bool
	PasswordHistoryLimit         int
	PasswordExpireDays           int
	MaxFailedAttempts            int
	LockMinutes                  int
	SourceMaxFailedAttempts      int
	SourceWindowMinutes          int
	SourceLockMinutes            int
	SessionIdleMinutes           int
	MaxActiveSessions            int
	LoginLogRetentionDays        int
	SessionRetentionDays         int
	LoginLogCleanupRetentionDays []int
	SessionCleanupRetentionDays  []int
	SecurityEventEnabled         bool
	CaptchaEnabled               bool
	MFAEnabled                   bool
	SSOEnabled                   bool
}

func toSecurityEventRespList(events []security.SecurityEventResp) []SecurityEventResp {
	result := make([]SecurityEventResp, 0, len(events))
	result = append(result, events...)
	return result
}

func mergePermissionKeys(groups ...[]string) []string {
	result := make([]string, 0)
	seen := make(map[string]struct{})
	for _, group := range groups {
		for _, item := range group {
			key := strings.TrimSpace(item)
			if key == "" {
				continue
			}
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			result = append(result, key)
		}
	}
	return result
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func defaultCleanupRetentionDays() []int {
	return []int{1, 7, 30}
}

func cloneIntSlice(values []int) []int {
	if len(values) == 0 {
		return nil
	}
	cloned := make([]int, len(values))
	copy(cloned, values)
	return cloned
}

func normalizeRetentionDays(rawValue string, fallback []int) []int {
	var values []int
	if err := json.Unmarshal([]byte(strings.TrimSpace(rawValue)), &values); err != nil {
		return cloneIntSlice(fallback)
	}
	seen := make(map[int]struct{}, len(values))
	normalized := make([]int, 0, len(values))
	for _, value := range values {
		if value <= 0 {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		normalized = append(normalized, value)
	}
	if len(normalized) == 0 {
		return cloneIntSlice(fallback)
	}
	sort.Ints(normalized)
	return normalized
}

// InvalidateUserPermCache invalidates the cached permissions for a user.
// This should be called after role assignments, permission changes, or role updates.
func (s *Runtime) InvalidateUserPermCache(userID uint64) error {
	if database.RDB == nil {
		return nil // No cache to invalidate
	}
	cacheKey := fmt.Sprintf("user_perms:%d", userID)
	return database.RDB.Del(context.Background(), cacheKey).Err()
}
