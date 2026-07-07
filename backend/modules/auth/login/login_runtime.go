package login

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"pantheon-ops/backend/modules/auth/mfa"
	"pantheon-ops/backend/modules/auth/security"
	"pantheon-ops/backend/modules/auth/session"
	iamuser "pantheon-ops/backend/modules/system/iam/user"
	"pantheon-ops/backend/pkg/authsession"
	"pantheon-ops/backend/pkg/authtoken"
	"pantheon-ops/backend/pkg/common"
	"pantheon-ops/backend/pkg/database"
	"pantheon-ops/backend/pkg/impexp"
	"pantheon-ops/backend/pkg/platformprefs"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

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
	autoCleanupMinInterval         = 15 * time.Minute
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

	errSessionInvalid                = "session.invalid"
	errCurrentSessionRevokeForbidden = "auth.session.current_revoke_forbidden"

	userIDWhereClause     = "user_id = ?"
	settingKeyWhereClause = "setting_key = ?"

	sessionIDAndUserIDWhereClause            = "session_id = ? AND " + userIDWhereClause
	sessionIDAndActiveUserIDWhereClause      = sessionIDAndUserIDWhereClause + " AND revoked_at IS NULL"
	otherActiveSessionsByUserWhereClause     = userIDWhereClause + " AND session_id <> ? AND revoked_at IS NULL"
	systemUserRoleUserIDAndStatusWhereClause = "system_user_role.user_id = ? AND system_role.status = ?"
	systemUserRoleUserIDAndPermsWhereClause  = "system_user_role.user_id = ? AND system_role_permission.permission_key <> ''"
	systemUserUsernameLikeWhereClause        = "system_user.username LIKE ?"

	touchSessionActivitySQL = `UPDATE system_user_session
SET last_activity_at = ?,
    last_ip = CASE WHEN ? <> '' THEN ? ELSE last_ip END,
    user_agent = CASE WHEN ? <> '' THEN ? ELSE user_agent END
WHERE session_id = ? AND user_id = ? AND revoked_at IS NULL
  AND (last_activity_at IS NULL OR last_activity_at < ?)`
)

// Runtime is the root auth service that composes sub-domain services.
type Runtime struct {
	db *gorm.DB

	// Sub-services
	loginSvc    *LoginService
	mfaSvc      *mfa.Service
	securitySvc *security.Service
	sessionSvc  *session.Service

	// 账号安全策略缓存
	settingsMu                   sync.RWMutex
	settingsCache                map[string]int
	loginLogCleanupRetentionDays []int
	sessionCleanupRetentionDays  []int
	cleanupMu                    sync.Mutex
	lastCleanupAt                map[string]time.Time
}

// NewRuntime constructs the root auth service and its sub-services.
func NewRuntime(db *gorm.DB) *Runtime {
	s := &Runtime{
		db:            db,
		settingsCache: make(map[string]int),
		lastCleanupAt: make(map[string]time.Time),
	}

	// Build sub-services, wiring them back through interfaces on Runtime.
	s.loginSvc = NewLoginService(db, s, s)
	s.mfaSvc = mfa.NewService(db, s, s, s)
	s.securitySvc = security.NewService(db, s)

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
	var u iamuser.SystemUser
	if err := s.db.First(&u, userID).Error; err != nil {
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
	s.settingsMu.RLock()
	defer s.settingsMu.RUnlock()
	return RuntimePolicy{
		MaxFailedAttempts:            s.settingsCache[settingMaxFailedAttemptsKey],
		LockMinutes:                  s.settingsCache[settingLockMinutesKey],
		SourceMaxFailedAttempts:      s.settingsCache[settingSourceMaxFailedAttemptsKey],
		SourceWindowMinutes:          s.settingsCache[settingSourceWindowMinutesKey],
		SourceLockMinutes:            s.settingsCache[settingSourceLockMinutesKey],
		SecurityEventEnabled:         s.settingsCache[settingSecurityEventEnabledKey] == 1,
		LoginLogRetentionDays:        s.settingsCache[settingLoginLogRetentionDaysKey],
		LoginLogCleanupRetentionDays: cloneIntSlice(s.loginLogCleanupRetentionDays),
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
	_ = s.db.Create(&event).Error
}

// ─────────────────────────────────────────────────────────────
// security.PolicyProvider implementation
// ─────────────────────────────────────────────────────────────
func (s *Runtime) GetAuthRuntimePolicy() security.AuthRuntimePolicy {
	s.settingsMu.RLock()
	defer s.settingsMu.RUnlock()
	return security.AuthRuntimePolicy{
		PasswordMinLength:       s.settingsCache[settingPasswordMinLengthKey],
		PasswordRequireDigit:    s.settingsCache[settingPasswordRequireDigitKey] == 1,
		PasswordRequireUpper:    s.settingsCache[settingPasswordRequireUpperKey] == 1,
		PasswordHistoryLimit:    s.settingsCache[settingPasswordHistoryLimitKey],
		PasswordExpireDays:      s.settingsCache[settingPasswordExpireDaysKey],
		MaxFailedAttempts:       s.settingsCache[settingMaxFailedAttemptsKey],
		LockMinutes:             s.settingsCache[settingLockMinutesKey],
		SourceMaxFailedAttempts: s.settingsCache[settingSourceMaxFailedAttemptsKey],
		SourceWindowMinutes:     s.settingsCache[settingSourceWindowMinutesKey],
		SourceLockMinutes:       s.settingsCache[settingSourceLockMinutesKey],
		SessionIdleMinutes:      s.settingsCache[settingSessionIdleMinutesKey],
		MaxActiveSessions:       s.settingsCache[settingMaxActiveSessionsKey],
		SessionRetentionDays:    s.settingsCache[settingSessionRetentionDaysKey],
		SecurityEventEnabled:    s.settingsCache[settingSecurityEventEnabledKey] == 1,
		CaptchaEnabled:          s.settingsCache[settingCaptchaEnabledKey] == 1,
		MFAEnabled:              s.settingsCache[settingMFAEnabledKey] == 1,
		SSOEnabled:              s.settingsCache[settingSSOEnabledKey] == 1,
	}
}

// ─────────────────────────────────────────────────────────────
// mfa.PolicyProvider implementation
// ─────────────────────────────────────────────────────────────
func (s *Runtime) IsMFAEnabled() bool {
	s.settingsMu.RLock()
	defer s.settingsMu.RUnlock()
	return s.settingsCache[settingMFAEnabledKey] == 1
}

// ─────────────────────────────────────────────────────────────
// mfa.IdentityProvider implementation
// ─────────────────────────────────────────────────────────────
func (s *Runtime) LoadUserByID(userID uint64) (*mfa.UserRecord, error) {
	var u iamuser.SystemUser
	if err := s.db.First(&u, userID).Error; err != nil {
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
	if s.db == nil {
		return nil, common.ErrDatabaseNotInitialized
	}
	if ctx == nil {
		ctx = context.Background()
	}
	policy := s.getAuthRuntimePolicy()
	now := time.Now()

	if err := s.governSessionInventory(now, policy); err != nil {
		return nil, err
	}
	if err := authsession.CleanupUserOverflowSessions(s.db, userID, now, policy.SessionIdleMinutes, maxInt(policy.MaxActiveSessions-1, 0)); err != nil {
		return nil, err
	}

	var u iamuser.SystemUser
	if err := s.db.First(&u, userID).Error; err != nil {
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
	}
	if err := s.db.Create(&sess).Error; err != nil {
		return nil, err
	}
	return s.issueTokenPair(ctx, &u, roles, &sess)
}

// ─────────────────────────────────────────────────────────────
// security.PolicyProvider implementation
// ─────────────────────────────────────────────────────────────
func (s *Runtime) GetSecurityRuntimePolicy() security.AuthRuntimePolicy {
	s.settingsMu.RLock()
	defer s.settingsMu.RUnlock()
	return security.AuthRuntimePolicy{
		PasswordMinLength:       s.settingsCache[settingPasswordMinLengthKey],
		PasswordRequireDigit:    s.settingsCache[settingPasswordRequireDigitKey] == 1,
		PasswordRequireUpper:    s.settingsCache[settingPasswordRequireUpperKey] == 1,
		PasswordHistoryLimit:    s.settingsCache[settingPasswordHistoryLimitKey],
		PasswordExpireDays:      s.settingsCache[settingPasswordExpireDaysKey],
		MaxFailedAttempts:       s.settingsCache[settingMaxFailedAttemptsKey],
		LockMinutes:             s.settingsCache[settingLockMinutesKey],
		SourceMaxFailedAttempts: s.settingsCache[settingSourceMaxFailedAttemptsKey],
		SourceWindowMinutes:     s.settingsCache[settingSourceWindowMinutesKey],
		SourceLockMinutes:       s.settingsCache[settingSourceLockMinutesKey],
		SessionIdleMinutes:      s.settingsCache[settingSessionIdleMinutesKey],
		MaxActiveSessions:       s.settingsCache[settingMaxActiveSessionsKey],
		SessionRetentionDays:    s.settingsCache[settingSessionRetentionDaysKey],
		SecurityEventEnabled:    s.settingsCache[settingSecurityEventEnabledKey] == 1,
		CaptchaEnabled:          s.settingsCache[settingCaptchaEnabledKey] == 1,
		MFAEnabled:              s.settingsCache[settingMFAEnabledKey] == 1,
		SSOEnabled:              s.settingsCache[settingSSOEnabledKey] == 1,
	}
}

// ─────────────────────────────────────────────────────────────
// Runtime facade methods used by HTTP handlers and module callers.
// ─────────────────────────────────────────────────────────────

func (s *Runtime) Login(req *LoginReq) (*iamuser.SystemUser, error) {
	return s.loginSvc.Authenticate(req)
}

func (s *Runtime) LoginWithSource(req *LoginReq, sourceKey string) (*iamuser.SystemUser, error) {
	return s.loginSvc.AuthenticateWithSource(req, sourceKey)
}

func (s *Runtime) Authenticate(req *LoginReq) (*iamuser.SystemUser, error) {
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

func (s *Runtime) CreateMFAChallenge(currentUser *iamuser.SystemUser) (*mfa.MFAChallengeResp, error) {
	mfaUser := &mfa.UserRecord{
		ID:       currentUser.ID,
		Username: currentUser.Username,
		Nickname: currentUser.Nickname,
		Avatar:   currentUser.Avatar,
		Email:    currentUser.Email,
		Phone:    currentUser.Phone,
		Status:   currentUser.Status,
	}
	return s.mfaSvc.CreateChallenge(mfaUser)
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
	var permissionKeys []string
	err := s.db.Table("system_role_permission").
		Select("DISTINCT system_role_permission.permission_key").
		Joins("JOIN system_user_role ON system_user_role.role_id = system_role_permission.role_id").
		Where("system_user_role.user_id = ? AND system_role_permission.permission_key <> ''", userID).
		Pluck("system_role_permission.permission_key", &permissionKeys).Error
	return mergePermissionKeys(permissionKeys), err
}

func (s *Runtime) GetCurrentUserInfo(userID uint64) (*UserInfoResp, error) {
	if s.db == nil {
		return nil, common.ErrDatabaseNotInitialized
	}
	var u iamuser.SystemUser
	if err := s.db.First(&u, userID).Error; err != nil {
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
	var u iamuser.SystemUser
	if err := s.db.First(&u, userID).Error; err != nil {
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
		if err := s.db.Model(&iamuser.SystemUser{}).
			Where("id = ?", userID).
			Update("preference_json", preferenceJSON).Error; err != nil {
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
	policy := s.getAuthRuntimePolicy()
	now := time.Now()
	if err := s.governSessionInventory(now, policy); err != nil {
		return nil, err
	}

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

	s.settingsMu.Lock()
	s.settingsCache[settingPasswordMinLengthKey] = policy.PasswordMinLength
	s.settingsCache[settingPasswordRequireDigitKey] = boolToInt(policy.PasswordRequireDigit)
	s.settingsCache[settingPasswordRequireUpperKey] = boolToInt(policy.PasswordRequireUpper)
	s.settingsCache[settingPasswordHistoryLimitKey] = policy.PasswordHistoryLimit
	s.settingsCache[settingPasswordExpireDaysKey] = policy.PasswordExpireDays
	s.settingsCache[settingMaxFailedAttemptsKey] = policy.MaxFailedAttempts
	s.settingsCache[settingLockMinutesKey] = policy.LockMinutes
	s.settingsCache[settingSourceMaxFailedAttemptsKey] = policy.SourceMaxFailedAttempts
	s.settingsCache[settingSourceWindowMinutesKey] = policy.SourceWindowMinutes
	s.settingsCache[settingSourceLockMinutesKey] = policy.SourceLockMinutes
	s.settingsCache[settingSessionIdleMinutesKey] = policy.SessionIdleMinutes
	s.settingsCache[settingMaxActiveSessionsKey] = policy.MaxActiveSessions
	s.settingsCache[settingLoginLogRetentionDaysKey] = policy.LoginLogRetentionDays
	s.settingsCache[settingSessionRetentionDaysKey] = policy.SessionRetentionDays
	s.loginLogCleanupRetentionDays = cloneIntSlice(policy.LoginLogCleanupRetentionDays)
	s.sessionCleanupRetentionDays = cloneIntSlice(policy.SessionCleanupRetentionDays)
	s.settingsCache[settingSecurityEventEnabledKey] = boolToInt(policy.SecurityEventEnabled)
	s.settingsCache[settingCaptchaEnabledKey] = boolToInt(policy.CaptchaEnabled)
	s.settingsCache[settingMFAEnabledKey] = boolToInt(policy.MFAEnabled)
	s.settingsCache[settingSSOEnabledKey] = boolToInt(policy.SSOEnabled)
	s.settingsMu.Unlock()

	return nil
}

func (s *Runtime) WatchSettings() {
	if database.RDB == nil {
		return
	}
	pubsub := database.RDB.Subscribe(context.TODO(), "settings:refresh")
	go func() {
		defer pubsub.Close()
		for range pubsub.Channel() {
			_ = s.ReloadSettings()
		}
	}()
}

func (s *Runtime) getAuthRuntimePolicy() authRuntimePolicy {
	s.settingsMu.RLock()
	defer s.settingsMu.RUnlock()

	return authRuntimePolicy{
		PasswordMinLength:            s.settingsCache[settingPasswordMinLengthKey],
		PasswordRequireDigit:         s.settingsCache[settingPasswordRequireDigitKey] == 1,
		PasswordRequireUpper:         s.settingsCache[settingPasswordRequireUpperKey] == 1,
		PasswordHistoryLimit:         s.settingsCache[settingPasswordHistoryLimitKey],
		PasswordExpireDays:           s.settingsCache[settingPasswordExpireDaysKey],
		MaxFailedAttempts:            s.settingsCache[settingMaxFailedAttemptsKey],
		LockMinutes:                  s.settingsCache[settingLockMinutesKey],
		SourceMaxFailedAttempts:      s.settingsCache[settingSourceMaxFailedAttemptsKey],
		SourceWindowMinutes:          s.settingsCache[settingSourceWindowMinutesKey],
		SourceLockMinutes:            s.settingsCache[settingSourceLockMinutesKey],
		SessionIdleMinutes:           s.settingsCache[settingSessionIdleMinutesKey],
		MaxActiveSessions:            s.settingsCache[settingMaxActiveSessionsKey],
		LoginLogRetentionDays:        s.settingsCache[settingLoginLogRetentionDaysKey],
		SessionRetentionDays:         s.settingsCache[settingSessionRetentionDaysKey],
		LoginLogCleanupRetentionDays: cloneIntSlice(s.loginLogCleanupRetentionDays),
		SessionCleanupRetentionDays:  cloneIntSlice(s.sessionCleanupRetentionDays),
		SecurityEventEnabled:         s.settingsCache[settingSecurityEventEnabledKey] == 1,
		CaptchaEnabled:               s.settingsCache[settingCaptchaEnabledKey] == 1,
		MFAEnabled:                   s.settingsCache[settingMFAEnabledKey] == 1,
		SSOEnabled:                   s.settingsCache[settingSSOEnabledKey] == 1,
	}
}

func (s *Runtime) getSecurityEventEnabled() bool {
	s.settingsMu.RLock()
	defer s.settingsMu.RUnlock()
	return s.settingsCache[settingSecurityEventEnabledKey] == 1
}

func (s *Runtime) getSettingInt(settingKey string, fallback int) int {
	s.settingsMu.RLock()
	if val, ok := s.settingsCache[settingKey]; ok {
		s.settingsMu.RUnlock()
		return val
	}
	s.settingsMu.RUnlock()
	return s.fetchSettingIntFromDB(settingKey, fallback)
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
		return cloneIntSlice(fallback)
	}
	return normalizeRetentionDays(rawValue, fallback)
}

func (s *Runtime) governSessionInventory(now time.Time, policy authRuntimePolicy) error {
	if err := authsession.CleanupInactiveSessions(s.db, now, policy.SessionIdleMinutes); err != nil {
		return err
	}
	return authsession.PurgeHistoricSessions(s.db, now, policy.SessionRetentionDays)
}

func (s *Runtime) issueTokenPair(ctx context.Context, u *iamuser.SystemUser, roles []string, sess *session.SystemUserSession) (*authtoken.Pair, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	accessToken := authtoken.NewAccessToken()
	refreshToken := authtoken.NewRefreshToken()
	now := time.Now()
	accessTTL := authtoken.AccessTokenTTL
	refreshTTL := authtoken.RefreshTokenTTL

	accessData := &authtoken.SessionData{
		UserID:         u.ID,
		Username:       u.Username,
		RoleKeys:       roles,
		SessionID:      sess.SessionID,
		LastActivityAt: now.Unix(),
	}
	if err := authtoken.StoreSession(ctx, database.RDB, accessToken, accessData, accessTTL); err != nil {
		return nil, err
	}
	if err := authtoken.StoreRefresh(ctx, database.RDB, refreshToken, u.ID, sess.SessionID, refreshTTL); err != nil {
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

func (s *Runtime) issueTokenPairForSession(ctx context.Context, userID uint64, username string, roles []string, sess *session.SystemUserSession) (*authtoken.Pair, error) {
	if ctx == nil {
		ctx = context.Background()
	}
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
	for _, item := range events {
		result = append(result, SecurityEventResp{
			ID:                  item.ID,
			UserID:              item.UserID,
			Username:            item.Username,
			EventType:           item.EventType,
			Severity:            item.Severity,
			SourceKey:           item.SourceKey,
			IP:                  item.IP,
			UserAgent:           item.UserAgent,
			MessageKey:          item.MessageKey,
			Metadata:            item.Metadata,
			AcknowledgedAt:      item.AcknowledgedAt,
			AcknowledgedBy:      item.AcknowledgedBy,
			AcknowledgedByUser:  item.AcknowledgedByUser,
			AcknowledgementNote: item.AcknowledgementNote,
			CreatedAt:           item.CreatedAt,
		})
	}
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
