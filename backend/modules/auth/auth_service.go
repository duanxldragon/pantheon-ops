package auth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	user "pantheon-ops/backend/modules/system/iam/user"
	"pantheon-ops/backend/pkg/authsession"
	"pantheon-ops/backend/pkg/common"
	"pantheon-ops/backend/pkg/database"
	"pantheon-ops/backend/pkg/impexp"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthService struct {
	db *gorm.DB

	// 核心安全策略缓存
	settingsMu    sync.RWMutex
	settingsCache map[string]int
	cleanupMu     sync.Mutex
	lastCleanupAt map[string]time.Time
}

type UserPreferenceUpdateResult struct {
	User      *UserInfoResp
	Previous  *user.UserPlatformPreferenceResp
	Current   *user.UserPlatformPreferenceResp
	Persisted string
}

type authRuntimePolicy struct {
	PasswordMinLength       int
	PasswordRequireDigit    bool
	PasswordRequireUpper    bool
	PasswordHistoryLimit    int
	PasswordExpireDays      int
	MaxFailedAttempts       int
	LockMinutes             int
	SourceMaxFailedAttempts int
	SourceWindowMinutes     int
	SourceLockMinutes       int
	SessionIdleMinutes      int
	MaxActiveSessions       int
	LoginLogRetentionDays   int
	SessionRetentionDays    int
	SecurityEventEnabled    bool
	CaptchaEnabled          bool
	MFAEnabled              bool
	SSOEnabled              bool
}

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

// NewAuthService 构造函数
func NewAuthService(db *gorm.DB) *AuthService {
	s := &AuthService{
		db:            db,
		settingsCache: make(map[string]int),
		lastCleanupAt: make(map[string]time.Time),
	}
	// 启动时同步加载一次核心设置
	_ = s.ReloadSettings()
	return s
}

// ReloadSettings 重新加载核心安全策略缓存
func (s *AuthService) ReloadSettings() error {
	policy := authRuntimePolicy{
		PasswordMinLength:       s.fetchSettingIntFromDB("security.password_min_length", defaultPasswordMinLength),
		PasswordRequireDigit:    s.fetchSettingBoolFromDB("security.password_require_digit", false),
		PasswordRequireUpper:    s.fetchSettingBoolFromDB("security.password_require_uppercase", false),
		PasswordHistoryLimit:    s.fetchSettingIntFromDB("security.password_history_limit", 0),
		PasswordExpireDays:      s.fetchSettingIntFromDB("security.password_expire_days", 0),
		MaxFailedAttempts:       s.fetchSettingIntFromDB("login.max_failed_attempts", defaultMaxFailedAttempts),
		LockMinutes:             s.fetchSettingIntFromDB("login.lock_minutes", defaultLockMinutes),
		SourceMaxFailedAttempts: s.fetchSettingIntFromDB("login.source_max_failed_attempts", defaultSourceMaxFailedAttempts),
		SourceWindowMinutes:     s.fetchSettingIntFromDB("login.source_window_minutes", defaultSourceWindowMinutes),
		SourceLockMinutes:       s.fetchSettingIntFromDB("login.source_lock_minutes", defaultSourceLockMinutes),
		SessionIdleMinutes:      s.fetchSettingIntFromDB("login.session_idle_minutes", defaultSessionIdleMinutes),
		MaxActiveSessions:       s.fetchSettingIntFromDB("login.max_active_sessions_per_user", defaultMaxActiveSessions),
		LoginLogRetentionDays:   s.fetchSettingIntFromDB("audit.login_log_retention_days", defaultLoginLogRetentionDays),
		SessionRetentionDays:    s.fetchSettingIntFromDB("audit.session_retention_days", defaultSessionRetentionDays),
		SecurityEventEnabled:    s.fetchSettingBoolFromDB("login.security_event_enabled", true),
		CaptchaEnabled:          s.fetchSettingBoolFromDB("login.captcha_enabled", false),
		MFAEnabled:              s.fetchSettingBoolFromDB("login.mfa_enabled", false),
		SSOEnabled:              s.fetchSettingBoolFromDB("login.sso_enabled", false),
	}

	s.settingsMu.Lock()
	s.settingsCache["security.password_min_length"] = policy.PasswordMinLength
	s.settingsCache["security.password_require_digit"] = boolToInt(policy.PasswordRequireDigit)
	s.settingsCache["security.password_require_uppercase"] = boolToInt(policy.PasswordRequireUpper)
	s.settingsCache["security.password_history_limit"] = policy.PasswordHistoryLimit
	s.settingsCache["security.password_expire_days"] = policy.PasswordExpireDays
	s.settingsCache["login.max_failed_attempts"] = policy.MaxFailedAttempts
	s.settingsCache["login.lock_minutes"] = policy.LockMinutes
	s.settingsCache["login.source_max_failed_attempts"] = policy.SourceMaxFailedAttempts
	s.settingsCache["login.source_window_minutes"] = policy.SourceWindowMinutes
	s.settingsCache["login.source_lock_minutes"] = policy.SourceLockMinutes
	s.settingsCache["login.session_idle_minutes"] = policy.SessionIdleMinutes
	s.settingsCache["login.max_active_sessions_per_user"] = policy.MaxActiveSessions
	s.settingsCache["audit.login_log_retention_days"] = policy.LoginLogRetentionDays
	s.settingsCache["audit.session_retention_days"] = policy.SessionRetentionDays
	s.settingsCache["login.security_event_enabled"] = boolToInt(policy.SecurityEventEnabled)
	s.settingsCache["login.captcha_enabled"] = boolToInt(policy.CaptchaEnabled)
	s.settingsCache["login.mfa_enabled"] = boolToInt(policy.MFAEnabled)
	s.settingsCache["login.sso_enabled"] = boolToInt(policy.SSOEnabled)
	s.settingsMu.Unlock()

	return nil
}

// WatchSettings 监听配置刷新信号 (跨模块/实例同步)
func (s *AuthService) WatchSettings() {
	if database.RDB == nil {
		return
	}
	pubsub := database.RDB.Subscribe(context.Background(), "settings:refresh")
	go func() {
		defer pubsub.Close()
		ch := pubsub.Channel()
		for range ch {
			_ = s.ReloadSettings()
		}
	}()
}

// Migrate 初始化认证域表结构。
func (s *AuthService) Migrate() error {
	if s.db == nil {
		return errors.New("database.not_initialized")
	}
	if err := s.db.AutoMigrate(&SystemUserSession{}, &SystemLogLogin{}, &SystemLoginThrottle{}, &SystemAuthFactor{}, &SystemAuthMFAChallenge{}, &SystemAuthSecurityEvent{}, &SystemUserPasswordHistory{}); err != nil {
		return err
	}
	return nil
}

// VerifyPasswordForOperation 敏感操作前的密码二次验证
func (s *AuthService) VerifyPasswordForOperation(userID uint64, sessionID string, password string) (string, error) {
	if s.db == nil {
		return "", errors.New("database.not_initialized")
	}
	if strings.TrimSpace(sessionID) == "" {
		return "", errors.New("auth.operation.verification_mismatch")
	}

	var currentUser user.SystemUser
	if err := s.db.First(&currentUser, userID).Error; err != nil {
		return "", err
	}

	// 校验密码
	if err := bcrypt.CompareHashAndPassword([]byte(currentUser.Password), []byte(password)); err != nil {
		return "", errors.New("auth.password.verify_failed")
	}

	// 生成操作令牌 (Operation Token)，有效期 5 分钟
	// 这里复用 JWT 逻辑，但添加一个特定的 Claim
	token, err := common.GenerateOperationToken(userID, sessionID, "secure_action", 5*time.Minute)
	if err != nil {
		return "", err
	}

	return token, nil
}

// Login 用户登录
func (s *AuthService) Login(req *LoginReq) (*user.SystemUser, error) {
	return s.LoginWithSource(req, "")
}

func (s *AuthService) LoginWithSource(req *LoginReq, sourceKey string) (*user.SystemUser, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}
	policy := s.getAuthRuntimePolicy()
	if blocked, err := s.checkSourceThrottle(sourceKey, policy, time.Now()); err != nil {
		return nil, err
	} else if blocked {
		return nil, errors.New("auth.login.error.source_blocked")
	}
	username := strings.TrimSpace(req.Username)
	if username == "" {
		_, _ = s.recordSourceFailure(sourceKey, policy, time.Now())
		return nil, errors.New("user.login.error.not_found")
	}

	var currentUser user.SystemUser
	result := s.db.Where("username = ?", username).First(&currentUser)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			if blocked, markErr := s.recordSourceFailure(sourceKey, policy, time.Now()); markErr != nil {
				return nil, markErr
			} else if blocked {
				return nil, errors.New("auth.login.error.source_blocked")
			}
			return nil, errors.New("user.login.error.not_found")
		}
		return nil, result.Error
	}

	if currentUser.Status == 2 {
		if blocked, markErr := s.recordSourceFailure(sourceKey, policy, time.Now()); markErr != nil {
			return nil, markErr
		} else if blocked {
			return nil, errors.New("auth.login.error.source_blocked")
		}
		return nil, errors.New("user.login.error.disabled")
	}
	if currentUser.LoginLockedUntil != nil && currentUser.LoginLockedUntil.After(time.Now()) {
		if blocked, markErr := s.recordSourceFailure(sourceKey, policy, time.Now()); markErr != nil {
			return nil, markErr
		} else if blocked {
			return nil, errors.New("auth.login.error.source_blocked")
		}
		return nil, errors.New("user.login.error.locked")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(currentUser.Password), []byte(req.Password)); err != nil {
		locked, markErr := s.recordFailedLoginAttempt(&currentUser, policy)
		if markErr != nil {
			return nil, markErr
		}
		if blocked, sourceErr := s.recordSourceFailure(sourceKey, policy, time.Now()); sourceErr != nil {
			return nil, sourceErr
		} else if blocked {
			s.recordSecurityEvent(SystemAuthSecurityEvent{
				UserID:     currentUser.ID,
				Username:   currentUser.Username,
				EventType:  "source_blocked",
				Severity:   "high",
				SourceKey:  sourceKey,
				MessageKey: "auth.security.event.source_blocked",
			})
			return nil, errors.New("auth.login.error.source_blocked")
		}
		if locked {
			s.recordSecurityEvent(SystemAuthSecurityEvent{
				UserID:     currentUser.ID,
				Username:   currentUser.Username,
				EventType:  "account_locked",
				Severity:   "high",
				SourceKey:  sourceKey,
				MessageKey: "auth.security.event.account_locked",
			})
			return nil, errors.New("user.login.error.locked")
		}
		s.recordSecurityEvent(SystemAuthSecurityEvent{
			UserID:     currentUser.ID,
			Username:   currentUser.Username,
			EventType:  "password_wrong",
			Severity:   "medium",
			SourceKey:  sourceKey,
			IP:         loginSourceIP(sourceKey),
			MessageKey: "auth.security.event.password_wrong",
		})
		return nil, errors.New("user.login.error.password_wrong")
	}
	if err := s.clearFailedLoginState(currentUser.ID); err != nil {
		return nil, err
	}

	return &currentUser, nil
}

func (s *AuthService) Authenticate(req *LoginReq) (*user.SystemUser, error) {
	return s.Login(req)
}

func (s *AuthService) CreateMFAChallenge(currentUser *user.SystemUser) (*MFAChallengeResp, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}
	if currentUser == nil || currentUser.ID == 0 {
		return nil, errors.New("auth.mfa.user_invalid")
	}
	if !s.getAuthRuntimePolicy().MFAEnabled {
		return nil, errors.New("auth.mfa.disabled")
	}

	var factor SystemAuthFactor
	err := s.db.Where("user_id = ? AND factor_type = ? AND enabled = ?", currentUser.ID, "totp", 1).First(&factor).Error
	setupRequired := errors.Is(err, gorm.ErrRecordNotFound)
	if err != nil && !setupRequired {
		return nil, err
	}

	secret := ""
	if setupRequired {
		var secretErr error
		secret, secretErr = generateTOTPSecret()
		if secretErr != nil {
			return nil, secretErr
		}
	}
	encryptedSecret, err := encryptMFASecret(secret)
	if err != nil {
		return nil, err
	}

	expiresAt := time.Now().Add(5 * time.Minute)
	challenge := SystemAuthMFAChallenge{
		ChallengeID:     uuid.NewString(),
		UserID:          currentUser.ID,
		Purpose:         "login",
		SecretEncrypted: encryptedSecret,
		SetupRequired:   boolToInt(setupRequired),
		ExpiresAt:       expiresAt,
	}
	if err := s.db.Create(&challenge).Error; err != nil {
		return nil, err
	}

	resp := &MFAChallengeResp{
		MFARequired:   true,
		ChallengeID:   challenge.ChallengeID,
		SetupRequired: setupRequired,
		ExpiresAt:     expiresAt.Format(time.RFC3339),
	}
	if setupRequired {
		resp.TOTPSecret = secret
		resp.TOTPProvisionURI = buildTOTPURL(currentUser.Username, secret)
	}
	return resp, nil
}

func (s *AuthService) VerifyMFAChallenge(req *MFAVerifyReq, ip string, userAgent string) (*AuthTokenResp, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}
	if req == nil || strings.TrimSpace(req.ChallengeID) == "" {
		return nil, errors.New("auth.mfa.challenge_required")
	}

	var challenge SystemAuthMFAChallenge
	if err := s.db.Where("challenge_id = ?", strings.TrimSpace(req.ChallengeID)).First(&challenge).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("auth.mfa.challenge_invalid")
		}
		return nil, err
	}
	if challenge.ConsumedAt != nil || !challenge.ExpiresAt.After(time.Now()) {
		return nil, errors.New("auth.mfa.challenge_expired")
	}

	secret := ""
	if challenge.SetupRequired == 1 {
		decrypted, err := decryptMFASecret(challenge.SecretEncrypted)
		if err != nil {
			return nil, err
		}
		secret = decrypted
	} else {
		var factor SystemAuthFactor
		if err := s.db.Where("user_id = ? AND factor_type = ? AND enabled = ?", challenge.UserID, "totp", 1).First(&factor).Error; err != nil {
			return nil, errors.New("auth.mfa.factor_missing")
		}
		decrypted, err := decryptMFASecret(factor.SecretEncrypted)
		if err != nil {
			return nil, err
		}
		secret = decrypted
	}
	if !validateTOTPCode(secret, req.Code, time.Now()) {
		return nil, errors.New("auth.mfa.code_invalid")
	}

	var currentUser user.SystemUser
	if err := s.db.First(&currentUser, challenge.UserID).Error; err != nil {
		return nil, err
	}
	if currentUser.Status == 2 {
		return nil, errors.New("user.login.error.disabled")
	}

	now := time.Now()
	if err := s.db.Transaction(func(tx *gorm.DB) error {
		if challenge.SetupRequired == 1 {
			encryptedSecret, err := encryptMFASecret(secret)
			if err != nil {
				return err
			}
			var factor SystemAuthFactor
			err = tx.Where("user_id = ? AND factor_type = ?", challenge.UserID, "totp").First(&factor).Error
			if errors.Is(err, gorm.ErrRecordNotFound) {
				factor = SystemAuthFactor{
					UserID:          challenge.UserID,
					FactorType:      "totp",
					SecretEncrypted: encryptedSecret,
					Enabled:         1,
					ConfirmedAt:     &now,
				}
				if err := tx.Create(&factor).Error; err != nil {
					return err
				}
			} else if err != nil {
				return err
			} else if err := tx.Model(&factor).Updates(map[string]any{
				"secret_encrypted": encryptedSecret,
				"enabled":          1,
				"confirmed_at":     &now,
			}).Error; err != nil {
				return err
			}
		}
		result := tx.Model(&SystemAuthMFAChallenge{}).
			Where("id = ? AND consumed_at IS NULL", challenge.ID).
			Update("consumed_at", &now)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return errors.New("auth.mfa.challenge_expired")
		}
		return nil
	}); err != nil {
		return nil, err
	}

	roles, err := s.GetUserRoles(currentUser.ID)
	if err != nil {
		return nil, err
	}
	tokenPair, err := s.CreateSession(&currentUser, roles, ip, userAgent)
	if err != nil {
		return nil, err
	}
	userInfo, err := s.GetCurrentUserInfo(currentUser.ID)
	if err != nil {
		return nil, err
	}

	return &AuthTokenResp{
		Token:            tokenPair.AccessToken,
		AccessToken:      tokenPair.AccessToken,
		RefreshToken:     tokenPair.RefreshToken,
		TokenType:        tokenPair.TokenType,
		AccessExpiresAt:  tokenPair.AccessExpiresAt.Format("2006-01-02 15:04:05"),
		RefreshExpiresAt: tokenPair.RefreshExpiresAt.Format("2006-01-02 15:04:05"),
		SessionID:        tokenPair.SessionID,
		User:             userInfo,
	}, nil
}

// GetUserRoles 获取用户角色标识
func (s *AuthService) GetUserRoles(userID uint64) ([]string, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}

	var roles []string
	err := s.db.Table("system_role").
		Select("system_role.role_key").
		Joins("JOIN system_user_role ON system_user_role.role_id = system_role.id").
		Where("system_user_role.user_id = ? AND system_role.status = ?", userID, 1).
		Pluck("system_role.role_key", &roles).Error
	if err != nil {
		return nil, err
	}
	return roles, nil
}

// GetUserPerms 获取用户按钮/接口权限标识
func (s *AuthService) GetUserPerms(userID uint64) ([]string, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}

	var permissionKeys []string
	err := s.db.Table("system_role_permission").
		Select("DISTINCT system_role_permission.permission_key").
		Joins("JOIN system_user_role ON system_user_role.role_id = system_role_permission.role_id").
		Where("system_user_role.user_id = ? AND system_role_permission.permission_key <> ''", userID).
		Pluck("system_role_permission.permission_key", &permissionKeys).Error
	if err != nil {
		return nil, err
	}
	return mergePermissionKeys(permissionKeys), nil
}

// GetCurrentUserInfo 获取当前登录主体信息
func (s *AuthService) GetCurrentUserInfo(userID uint64) (*UserInfoResp, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}

	var currentUser user.SystemUser
	if err := s.db.First(&currentUser, userID).Error; err != nil {
		return nil, err
	}

	roles, err := s.GetUserRoles(currentUser.ID)
	if err != nil {
		return nil, err
	}
	perms, err := s.GetUserPerms(currentUser.ID)
	if err != nil {
		return nil, err
	}

	return &UserInfoResp{
		ID:          currentUser.ID,
		Username:    currentUser.Username,
		Nickname:    currentUser.Nickname,
		Avatar:      currentUser.Avatar,
		Email:       currentUser.Email,
		Phone:       currentUser.Phone,
		Roles:       roles,
		Perms:       perms,
		Preferences: user.ParseUserPlatformPreferences(currentUser.PreferenceJSON),
	}, nil
}

func (s *AuthService) UpdateCurrentUserPreferences(userID uint64, req *UserPlatformPreferenceUpdateReq) (*UserPreferenceUpdateResult, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}

	var currentUser user.SystemUser
	if err := s.db.First(&currentUser, userID).Error; err != nil {
		return nil, err
	}

	previousPreferences := user.ParseUserPlatformPreferences(currentUser.PreferenceJSON)
	nextPreferences := user.NormalizeUserPlatformPreferences(&user.UserPlatformPreferenceResp{
		Theme:       req.Theme,
		Language:    req.Language,
		LayoutMode:  req.LayoutMode,
		DensityMode: req.DensityMode,
	})
	preferenceJSON, err := user.MarshalUserPlatformPreferences(nextPreferences)
	if err != nil {
		return nil, err
	}

	if preferenceJSON != currentUser.PreferenceJSON {
		if err := s.db.Model(&user.SystemUser{}).
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

// UpdatePassword 修改当前登录用户密码
func (s *AuthService) UpdatePassword(userID uint64, currentSessionID string, req *PasswordUpdateReq) error {
	if s.db == nil {
		return errors.New("database.not_initialized")
	}

	oldPassword := strings.TrimSpace(req.OldPassword)
	newPassword := strings.TrimSpace(req.NewPassword)
	policy := s.getAuthRuntimePolicy()
	if len(newPassword) < policy.PasswordMinLength {
		return errors.New("user.update.error.password_too_short")
	}
	if !passwordMatchesComplexity(newPassword, policy) {
		return errors.New("user.update.error.password_weak")
	}

	var currentUser user.SystemUser
	if err := s.db.First(&currentUser, userID).Error; err != nil {
		return err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(currentUser.Password), []byte(oldPassword)); err != nil {
		return errors.New("user.password.error.old_password_invalid")
	}
	if oldPassword == newPassword {
		return errors.New("user.password.error.same_as_old")
	}
	if err := s.ensurePasswordNotRecentlyUsed(currentUser.ID, newPassword, currentUser.Password, policy); err != nil {
		return err
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	return s.db.Transaction(func(tx *gorm.DB) error {
		if policy.PasswordHistoryLimit > 0 {
			if err := tx.Create(&SystemUserPasswordHistory{
				UserID:       currentUser.ID,
				PasswordHash: currentUser.Password,
				ChangedAt:    time.Now(),
			}).Error; err != nil {
				return err
			}
		}
		if err := tx.Model(&currentUser).Update("password", string(passwordHash)).Error; err != nil {
			return err
		}
		if strings.TrimSpace(currentSessionID) == "" {
			return nil
		}
		now := time.Now()
		return tx.Model(&SystemUserSession{}).
			Where("user_id = ? AND session_id <> ? AND revoked_at IS NULL", userID, currentSessionID).
			Updates(map[string]interface{}{
				"revoked_at": &now,
			}).Error
	})
}

// CreateSession 创建登录会话并签发 token pair
func (s *AuthService) CreateSession(currentUser *user.SystemUser, roles []string, ip string, userAgent string) (*common.TokenPair, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}
	policy := s.getAuthRuntimePolicy()
	now := time.Now()
	if err := s.governSessionInventory(now, policy); err != nil {
		return nil, err
	}
	if err := authsession.CleanupUserOverflowSessions(s.db, currentUser.ID, now, policy.SessionIdleMinutes, maxInt(policy.MaxActiveSessions-1, 0)); err != nil {
		return nil, err
	}

	session := SystemUserSession{
		SessionID:        uuid.NewString(),
		UserID:           currentUser.ID,
		RefreshJTI:       uuid.NewString(),
		RefreshExpiresAt: now.Add(7 * 24 * time.Hour),
		LastActivityAt:   &now,
		LastIP:           ip,
		UserAgent:        truncateString(userAgent, 255),
	}

	if err := s.db.Create(&session).Error; err != nil {
		return nil, err
	}
	return s.issueTokenPair(currentUser, roles, &session)
}

// RefreshSession 轮换 refresh token 并返回新的 token pair
func (s *AuthService) RefreshSession(claims *common.CustomClaims, ip string, userAgent string) (*common.TokenPair, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}

	var session SystemUserSession
	err := s.db.Where("session_id = ? AND user_id = ?", claims.SessionID, claims.UserID).First(&session).Error
	if err != nil {
		return nil, err
	}
	if session.RevokedAt != nil || session.RefreshExpiresAt.Before(time.Now()) {
		return nil, errors.New("refresh_token.invalid")
	}
	if session.RefreshJTI != claims.ID {
		return nil, errors.New("refresh_token.rotated")
	}

	var currentUser user.SystemUser
	if err := s.db.First(&currentUser, claims.UserID).Error; err != nil {
		return nil, err
	}
	roles, err := s.GetUserRoles(currentUser.ID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	session.RefreshJTI = uuid.NewString()
	session.RefreshExpiresAt = now.Add(7 * 24 * time.Hour)
	session.LastRefreshAt = &now
	session.LastActivityAt = &now
	session.LastIP = ip
	session.UserAgent = truncateString(userAgent, 255)
	if err := s.db.Save(&session).Error; err != nil {
		return nil, err
	}

	return s.issueTokenPair(&currentUser, roles, &session)
}

// RevokeSession 吊销会话
func (s *AuthService) RevokeSession(sessionID string) error {
	if s.db == nil || sessionID == "" {
		return nil
	}

	now := time.Now()
	return s.db.Model(&SystemUserSession{}).
		Where("session_id = ? AND revoked_at IS NULL", sessionID).
		Updates(map[string]interface{}{"revoked_at": &now}).Error
}

func (s *AuthService) TouchSessionActivity(sessionID string, userID uint64, ip string, userAgent string) error {
	if s.db == nil || strings.TrimSpace(sessionID) == "" || userID == 0 {
		return nil
	}

	now := time.Now()
	updates := map[string]interface{}{
		"last_activity_at": &now,
	}
	if strings.TrimSpace(ip) != "" {
		updates["last_ip"] = ip
	}
	if strings.TrimSpace(userAgent) != "" {
		updates["user_agent"] = truncateString(userAgent, 255)
	}

	return s.db.Model(&SystemUserSession{}).
		Where("session_id = ?", sessionID).
		Where("user_id = ?", userID).
		Where("revoked_at IS NULL").
		Where("(last_activity_at IS NULL OR last_activity_at < ?)", now.Add(-1*time.Minute)).
		Updates(updates).Error
}

// ListSessions 获取当前用户会话列表
func (s *AuthService) ListSessions(userID uint64, currentSessionID string) ([]SessionResp, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}

	now := time.Now()
	policy := s.getAuthRuntimePolicy()
	if err := s.governSessionInventory(now, policy); err != nil {
		return nil, err
	}

	var sessions []SystemUserSession
	if err := authsession.ApplyActiveScope(s.db, "", now, policy.SessionIdleMinutes).
		Where("user_id = ?", userID).
		Order("created_at desc").
		Find(&sessions).Error; err != nil {
		return nil, err
	}

	result := make([]SessionResp, 0, len(sessions))
	for _, item := range sessions {
		result = append(result, buildSessionResp(item, currentSessionID))
	}
	sort.SliceStable(result, func(i, j int) bool {
		return result[i].IsCurrent && !result[j].IsCurrent
	})
	return result, nil
}

// GetSecurityOverview 获取当前账号安全概览
func (s *AuthService) GetSecurityOverview(userID uint64, username string, currentSessionID string) (*SecurityOverviewResp, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
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
			session := sessions[i]
			currentSession = &session
			break
		}
	}

	var activeSessionCount int64
	if err := authsession.ApplyActiveScope(s.db.Model(&SystemUserSession{}), "", now, policy.SessionIdleMinutes).
		Where("user_id = ?", userID).
		Count(&activeSessionCount).Error; err != nil {
		return nil, err
	}

	var lastLoginAt *string
	var lastLogin SystemLogLogin
	err = s.db.Where("username = ? AND status = ?", username, 1).
		Order("login_time desc, id desc").
		First(&lastLogin).Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	if err == nil {
		formatted := lastLogin.LoginTime.Format(time.RFC3339)
		lastLoginAt = &formatted
	}

	return &SecurityOverviewResp{
		User:                 info,
		CurrentSession:       currentSession,
		ActiveSessionCount:   activeSessionCount,
		LastLoginAt:          lastLoginAt,
		PasswordExpired:      s.isPasswordExpired(userID, policy, now),
		PasswordExpiresAt:    s.passwordExpiresAt(userID, policy),
		RecentSecurityEvents: s.listRecentSecurityEvents(userID, 5),
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

// RevokeOwnedSession 吊销当前用户的指定会话
func (s *AuthService) RevokeOwnedSession(userID uint64, currentSessionID string, targetSessionID string) error {
	if s.db == nil {
		return errors.New("database.not_initialized")
	}
	if strings.TrimSpace(targetSessionID) == "" {
		return errors.New("session.invalid")
	}
	if targetSessionID == currentSessionID {
		return errors.New("auth.session.current_revoke_forbidden")
	}

	var session SystemUserSession
	if err := s.db.Where("session_id = ? AND user_id = ?", targetSessionID, userID).First(&session).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("session.invalid")
		}
		return err
	}
	if session.RevokedAt != nil {
		return nil
	}

	now := time.Now()
	return s.db.Model(&SystemUserSession{}).
		Where("session_id = ? AND user_id = ? AND revoked_at IS NULL", targetSessionID, userID).
		Updates(map[string]interface{}{"revoked_at": &now}).Error
}

// ListOwnLoginLogs 获取当前用户登录日志
func (s *AuthService) ListOwnLoginLogs(username string, query *LoginLogQuery) (*LoginLogPageResp, error) {
	if strings.TrimSpace(username) == "" {
		return nil, errors.New("token.invalid")
	}
	return s.listLoginLogs(&LoginLogQuery{
		Username: username,
		Status:   queryStatus(query),
		Page:     queryPage(query),
		PageSize: queryPageSize(query),
	})
}

// ListLoginLogs 获取管理员登录日志
func (s *AuthService) ListLoginLogs(query *LoginLogQuery) (*LoginLogPageResp, error) {
	s.ensureAutomaticLoginLogRetention()
	return s.listLoginLogs(query)
}

func (s *AuthService) ListSecurityEvents(query *SecurityEventQuery) (*SecurityEventPageResp, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}

	page, pageSize := normalizeSecurityEventPageQuery(query)
	db := s.db.Model(&SystemAuthSecurityEvent{})
	if query != nil {
		if strings.TrimSpace(query.Username) != "" {
			db = db.Where("username LIKE ?", "%"+strings.TrimSpace(query.Username)+"%")
		}
		if strings.TrimSpace(query.EventType) != "" {
			db = db.Where("event_type = ?", strings.TrimSpace(query.EventType))
		}
		if strings.TrimSpace(query.Severity) != "" {
			db = db.Where("severity = ?", strings.TrimSpace(query.Severity))
		}
		if query.Acknowledged != nil {
			if *query.Acknowledged {
				db = db.Where("acknowledged_at IS NOT NULL")
			} else {
				db = db.Where("acknowledged_at IS NULL")
			}
		}
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}
	var events []SystemAuthSecurityEvent
	if err := db.Order("created_at desc, id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&events).Error; err != nil {
		return nil, err
	}
	return &SecurityEventPageResp{
		Items:    toSecurityEventRespList(events),
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (s *AuthService) ExportLoginLogs(query *LoginLogQuery) (*impexp.CSVFile, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}
	s.ensureAutomaticLoginLogRetention()

	logs, err := s.listLoginLogsForExport(query)
	if err != nil {
		return nil, err
	}

	rows := make([][]string, 0, len(logs))
	for _, item := range logs {
		rows = append(rows, []string{
			item.Username,
			item.Ipaddr,
			item.LoginLocation,
			item.Browser,
			item.Os,
			fmt.Sprintf("%d", item.Status),
			item.Msg,
			item.LoginTime.Format(time.RFC3339),
		})
	}

	return &impexp.CSVFile{
		Filename: "system-login-log-export.csv",
		Headers:  []string{"username", "ipaddr", "loginLocation", "browser", "os", "status", "msg", "loginTime"},
		Rows:     rows,
	}, nil
}

func (s *AuthService) CleanupLoginLogs(retentionDays int, startedAt string, endedAt string) (int64, error) {
	if s.db == nil {
		return 0, errors.New("database.not_initialized")
	}
	window, err := parseCleanupWindow(startedAt, endedAt, "auth.login_log.cleanup.range_invalid")
	if err != nil {
		return 0, err
	}
	db := s.db.Model(&SystemLogLogin{})
	if window != nil {
		db = db.Where("login_time >= ? AND login_time <= ?", window.StartedAt, window.EndedAt)
	} else {
		if !s.isAllowedLoginLogRetentionDays(retentionDays) {
			return 0, errors.New("auth.login_log.cleanup.days_invalid")
		}
		cutoff := time.Now().AddDate(0, 0, -retentionDays)
		db = db.Where("login_time < ?", cutoff)
	}
	result := db.Delete(&SystemLogLogin{})
	if result.Error != nil {
		return 0, result.Error
	}
	return result.RowsAffected, nil
}

func (s *AuthService) CleanupHistoricSessions(retentionDays int, startedAt string, endedAt string) (int64, error) {
	if s.db == nil {
		return 0, errors.New("database.not_initialized")
	}
	window, err := parseCleanupWindow(startedAt, endedAt, "auth.session.cleanup.range_invalid")
	if err != nil {
		return 0, err
	}

	now := time.Now()
	policy := s.getAuthRuntimePolicy()
	if err := s.governSessionInventory(now, policy); err != nil {
		return 0, err
	}

	db := s.db.Table("system_user_session").Where("revoked_at IS NOT NULL")
	if window != nil {
		db = db.Where("revoked_at >= ? AND revoked_at <= ?", window.StartedAt, window.EndedAt)
	} else {
		if !s.isAllowedSessionCleanupRetentionDays(retentionDays) {
			return 0, errors.New("auth.session.cleanup.days_invalid")
		}
		cutoff := now.AddDate(0, 0, -retentionDays)
		db = db.Where("revoked_at < ?", cutoff)
	}
	result := db.Delete(nil)
	if result.Error != nil {
		return 0, result.Error
	}
	return result.RowsAffected, nil
}

func (s *AuthService) BatchRevokeSessions(currentSessionID string, sessionIDs []string) (int64, error) {
	if s.db == nil {
		return 0, errors.New("database.not_initialized")
	}

	normalized := normalizeSessionIDs(sessionIDs)
	if len(normalized) == 0 {
		return 0, errors.New("session.invalid")
	}
	for _, sessionID := range normalized {
		if sessionID == currentSessionID {
			return 0, errors.New("auth.session.current_revoke_forbidden")
		}
	}

	now := time.Now()
	result := s.db.Model(&SystemUserSession{}).
		Where("session_id IN ? AND revoked_at IS NULL", normalized).
		Updates(map[string]interface{}{"revoked_at": &now})
	if result.Error != nil {
		return 0, result.Error
	}
	return result.RowsAffected, nil
}

func (s *AuthService) AcknowledgeSecurityEvent(eventID uint64, actorID uint64, actorUsername string, note string) error {
	if s.db == nil {
		return errors.New("database.not_initialized")
	}
	note = strings.TrimSpace(note)
	if note == "" {
		return errors.New("auth.security_event.acknowledge.note_required")
	}

	result := s.db.Model(&SystemAuthSecurityEvent{}).
		Where("id = ?", eventID).
		Updates(map[string]interface{}{
			"acknowledged_at":      time.Now(),
			"acknowledged_by":      actorID,
			"acknowledged_by_user": strings.TrimSpace(actorUsername),
			"acknowledgement_note": note,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (s *AuthService) isAllowedLoginLogRetentionDays(retentionDays int) bool {
	for _, allowed := range s.getLoginLogRetentionOptions() {
		if allowed == retentionDays {
			return true
		}
	}
	return false
}

func (s *AuthService) getLoginLogRetentionOptions() []int {
	return s.getRetentionOptionsFromSetting("audit.login_log_retention_options", []int{1, 7, 30})
}

func (s *AuthService) isAllowedSessionCleanupRetentionDays(retentionDays int) bool {
	for _, allowed := range s.getSessionCleanupRetentionOptions() {
		if allowed == retentionDays {
			return true
		}
	}
	return false
}

func (s *AuthService) getSessionCleanupRetentionOptions() []int {
	return s.getRetentionOptionsFromSetting("audit.session_cleanup_retention_options", []int{1, 7, 30})
}

func (s *AuthService) getRetentionOptionsFromSetting(settingKey string, fallback []int) []int {
	if s.db == nil {
		return fallback
	}

	var row struct {
		SettingValue string `gorm:"column:setting_value"`
	}
	if err := s.db.Table("system_setting").Select("setting_value").Where("setting_key = ?", settingKey).Take(&row).Error; err != nil {
		return fallback
	}

	var values []int
	if err := json.Unmarshal([]byte(strings.TrimSpace(row.SettingValue)), &values); err != nil {
		return fallback
	}

	normalized := make([]int, 0, len(values))
	seen := make(map[int]struct{}, len(values))
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
		return fallback
	}

	sort.Ints(normalized)
	return normalized
}

func (s *AuthService) BatchDeleteLoginLogs(ids []uint64) (int64, error) {
	if s.db == nil {
		return 0, errors.New("database.not_initialized")
	}

	normalized := normalizeUint64IDs(ids)
	if len(normalized) == 0 {
		return 0, errors.New("auth.login_log.delete.ids_required")
	}

	result := s.db.Where("id IN ?", normalized).Delete(&SystemLogLogin{})
	if result.Error != nil {
		return 0, result.Error
	}
	return result.RowsAffected, nil
}

func (s *AuthService) listLoginLogs(query *LoginLogQuery) (*LoginLogPageResp, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}
	s.ensureAutomaticLoginLogRetention()

	var logs []SystemLogLogin
	db := s.db.Model(&SystemLogLogin{})
	page, pageSize := normalizePageQuery(queryPage(query), queryPageSize(query))
	if query != nil {
		if strings.TrimSpace(query.Username) != "" {
			db = db.Where("username LIKE ?", fmt.Sprintf("%%%s%%", strings.TrimSpace(query.Username)))
		}
		if query.Status != nil && (*query.Status == 0 || *query.Status == 1) {
			db = db.Where("status = ?", *query.Status)
		}
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}
	if err := db.Order("login_time desc, id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&logs).Error; err != nil {
		return nil, err
	}

	items := make([]LoginLogResp, 0, len(logs))
	for _, item := range logs {
		items = append(items, LoginLogResp{
			ID:            item.ID,
			Username:      item.Username,
			Ipaddr:        item.Ipaddr,
			LoginLocation: item.LoginLocation,
			Browser:       item.Browser,
			Os:            item.Os,
			Status:        item.Status,
			Msg:           item.Msg,
			LoginTime:     item.LoginTime.Format(time.RFC3339),
		})
	}
	return &LoginLogPageResp{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (s *AuthService) listLoginLogsForExport(query *LoginLogQuery) ([]SystemLogLogin, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}
	s.ensureAutomaticLoginLogRetention()

	var logs []SystemLogLogin
	db := s.db.Model(&SystemLogLogin{})
	if query != nil {
		if strings.TrimSpace(query.Username) != "" {
			db = db.Where("username LIKE ?", fmt.Sprintf("%%%s%%", strings.TrimSpace(query.Username)))
		}
		if query.Status != nil && (*query.Status == 0 || *query.Status == 1) {
			db = db.Where("status = ?", *query.Status)
		}
	}
	if err := db.Order("login_time desc, id desc").Find(&logs).Error; err != nil {
		return nil, err
	}
	return logs, nil
}

// ListAllSessions 获取管理员会话列表
func (s *AuthService) ListAllSessions(query *AdminSessionQuery) (*AdminSessionPageResp, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}

	now := time.Now()
	policy := s.getAuthRuntimePolicy()
	if err := s.governSessionInventory(now, policy); err != nil {
		return nil, err
	}

	type sessionRow struct {
		SessionID        string     `gorm:"column:session_id"`
		UserID           uint64     `gorm:"column:user_id"`
		Username         string     `gorm:"column:username"`
		Nickname         string     `gorm:"column:nickname"`
		LastIP           string     `gorm:"column:last_ip"`
		UserAgent        string     `gorm:"column:user_agent"`
		RefreshExpiresAt time.Time  `gorm:"column:refresh_expires_at"`
		LastRefreshAt    *time.Time `gorm:"column:last_refresh_at"`
		LastActivityAt   *time.Time `gorm:"column:last_activity_at"`
		RevokedAt        *time.Time `gorm:"column:revoked_at"`
		CreatedAt        time.Time  `gorm:"column:created_at"`
	}

	page, pageSize := normalizePageQuery(queryPageFromAdminSession(query), queryPageSizeFromAdminSession(query))
	db := s.db.Table("system_user_session").
		Select("system_user_session.session_id, system_user_session.user_id, system_user.username, system_user.nickname, system_user_session.last_ip, system_user_session.user_agent, system_user_session.refresh_expires_at, system_user_session.last_refresh_at, system_user_session.last_activity_at, system_user_session.revoked_at, system_user_session.created_at").
		Joins("LEFT JOIN system_user ON system_user.id = system_user_session.user_id")
	if query != nil {
		if strings.TrimSpace(query.Username) != "" {
			db = db.Where("system_user.username LIKE ?", fmt.Sprintf("%%%s%%", strings.TrimSpace(query.Username)))
		}
		if strings.TrimSpace(query.LastIP) != "" {
			db = db.Where("system_user_session.last_ip LIKE ?", fmt.Sprintf("%%%s%%", strings.TrimSpace(query.LastIP)))
		}
		if query.Status != nil {
			if *query.Status == 1 {
				db = authsession.ApplyActiveScope(db, "system_user_session", now, policy.SessionIdleMinutes)
			} else if *query.Status == 2 {
				db = db.Where("system_user_session.revoked_at IS NOT NULL")
			}
		}
	}

	var rows []sessionRow
	if err := db.Order("system_user_session.created_at desc").Scan(&rows).Error; err != nil {
		return nil, err
	}

	items := make([]AdminSessionResp, 0, len(rows))
	var activeCount int64
	var revokedCount int64
	for _, row := range rows {
		clientInfo := parseClientInfo(row.UserAgent)
		if query != nil {
			if strings.TrimSpace(query.Browser) != "" && !strings.EqualFold(strings.TrimSpace(query.Browser), clientInfo.Browser) {
				continue
			}
			if strings.TrimSpace(query.OS) != "" && !strings.EqualFold(strings.TrimSpace(query.OS), clientInfo.OS) {
				continue
			}
			if strings.TrimSpace(query.Device) != "" && !strings.EqualFold(strings.TrimSpace(query.Device), clientInfo.Device) {
				continue
			}
		}
		if row.RevokedAt == nil {
			activeCount++
		} else {
			revokedCount++
		}
		items = append(items, AdminSessionResp{
			SessionID:        row.SessionID,
			UserID:           row.UserID,
			Username:         row.Username,
			Nickname:         row.Nickname,
			LastIP:           row.LastIP,
			Browser:          clientInfo.Browser,
			OS:               clientInfo.OS,
			Device:           clientInfo.Device,
			UserAgent:        clientInfo.UserAgent,
			RefreshExpiresAt: row.RefreshExpiresAt.Format(time.RFC3339),
			LastRefreshAt:    formatNullableTime(row.LastRefreshAt),
			LastActivityAt:   formatNullableTime(row.LastActivityAt),
			RevokedAt:        formatNullableTime(row.RevokedAt),
			CreatedAt:        row.CreatedAt.Format(time.RFC3339),
		})
	}

	total := int64(len(items))
	start := (page - 1) * pageSize
	if start > len(items) {
		start = len(items)
	}
	end := start + pageSize
	if end > len(items) {
		end = len(items)
	}

	return &AdminSessionPageResp{
		Items:        items[start:end],
		Total:        total,
		ActiveCount:  activeCount,
		RevokedCount: revokedCount,
		Page:         page,
		PageSize:     pageSize,
	}, nil
}

// RevokeAnySession 管理员吊销任意会话
func (s *AuthService) RevokeAnySession(currentSessionID string, targetSessionID string) error {
	if s.db == nil {
		return errors.New("database.not_initialized")
	}
	if strings.TrimSpace(targetSessionID) == "" {
		return errors.New("session.invalid")
	}
	if targetSessionID == currentSessionID {
		return errors.New("auth.session.current_revoke_forbidden")
	}

	now := time.Now()
	return s.db.Model(&SystemUserSession{}).
		Where("session_id = ? AND revoked_at IS NULL", targetSessionID).
		Updates(map[string]interface{}{"revoked_at": &now}).Error
}

// RecordLoginLog 记录登录日志
func (s *AuthService) RecordLoginLog(requestID, username, ip, browser, os string, status int, msg string) {
	if s.db == nil {
		return
	}
	s.ensureAutomaticLoginLogRetention()

	loginLog := SystemLogLogin{
		RequestID:     strings.TrimSpace(requestID),
		Username:      username,
		Ipaddr:        ip,
		Browser:       browser,
		Os:            os,
		Status:        status,
		Msg:           msg,
		LoginTime:     time.Now(),
		LoginLocation: common.GetLocationByIP(ip),
	}
	_ = s.db.Create(&loginLog).Error
}

func (s *AuthService) getAuthRuntimePolicy() authRuntimePolicy {
	s.settingsMu.RLock()
	defer s.settingsMu.RUnlock()

	return authRuntimePolicy{
		PasswordMinLength:       s.settingsCache["security.password_min_length"],
		PasswordRequireDigit:    s.settingsCache["security.password_require_digit"] == 1,
		PasswordRequireUpper:    s.settingsCache["security.password_require_uppercase"] == 1,
		PasswordHistoryLimit:    s.settingsCache["security.password_history_limit"],
		PasswordExpireDays:      s.settingsCache["security.password_expire_days"],
		MaxFailedAttempts:       s.settingsCache["login.max_failed_attempts"],
		LockMinutes:             s.settingsCache["login.lock_minutes"],
		SourceMaxFailedAttempts: maxInt(s.settingsCache["login.source_max_failed_attempts"], defaultSourceMaxFailedAttempts),
		SourceWindowMinutes:     maxInt(s.settingsCache["login.source_window_minutes"], defaultSourceWindowMinutes),
		SourceLockMinutes:       maxInt(s.settingsCache["login.source_lock_minutes"], defaultSourceLockMinutes),
		SessionIdleMinutes:      s.settingsCache["login.session_idle_minutes"],
		MaxActiveSessions:       maxInt(s.settingsCache["login.max_active_sessions_per_user"], defaultMaxActiveSessions),
		LoginLogRetentionDays:   maxInt(s.settingsCache["audit.login_log_retention_days"], defaultLoginLogRetentionDays),
		SessionRetentionDays:    maxInt(s.settingsCache["audit.session_retention_days"], defaultSessionRetentionDays),
		SecurityEventEnabled:    s.settingsCache["login.security_event_enabled"] == 1,
		CaptchaEnabled:          s.settingsCache["login.captcha_enabled"] == 1,
		MFAEnabled:              s.settingsCache["login.mfa_enabled"] == 1,
		SSOEnabled:              s.settingsCache["login.sso_enabled"] == 1,
	}
}

func passwordMatchesComplexity(password string, policy authRuntimePolicy) bool {
	if !policy.PasswordRequireDigit && !policy.PasswordRequireUpper {
		return true
	}
	hasDigit := false
	hasUpper := false
	for _, r := range password {
		if unicode.IsDigit(r) {
			hasDigit = true
		}
		if unicode.IsUpper(r) {
			hasUpper = true
		}
	}
	if policy.PasswordRequireDigit && !hasDigit {
		return false
	}
	if policy.PasswordRequireUpper && !hasUpper {
		return false
	}
	return true
}

func (s *AuthService) governSessionInventory(now time.Time, policy authRuntimePolicy) error {
	if err := authsession.CleanupInactiveSessions(s.db, now, policy.SessionIdleMinutes); err != nil {
		return err
	}
	return authsession.PurgeHistoricSessions(s.db, now, policy.SessionRetentionDays)
}

func (s *AuthService) ensureAutomaticLoginLogRetention() {
	if s.db == nil {
		return
	}

	now := time.Now()
	s.cleanupMu.Lock()
	lastRun := s.lastCleanupAt["login_log_retention"]
	if !lastRun.IsZero() && now.Sub(lastRun) < autoCleanupMinInterval {
		s.cleanupMu.Unlock()
		return
	}
	s.lastCleanupAt["login_log_retention"] = now
	s.cleanupMu.Unlock()

	retentionDays := s.getSettingInt("audit.login_log_retention_days", defaultLoginLogRetentionDays)
	if retentionDays <= 0 {
		retentionDays = defaultLoginLogRetentionDays
	}
	cutoff := now.AddDate(0, 0, -retentionDays)
	_ = s.db.Where("login_time < ?", cutoff).Delete(&SystemLogLogin{}).Error
}

func (s *AuthService) getSettingInt(settingKey string, fallback int) int {
	s.settingsMu.RLock()
	if val, ok := s.settingsCache[settingKey]; ok {
		s.settingsMu.RUnlock()
		return val
	}
	s.settingsMu.RUnlock()

	return s.fetchSettingIntFromDB(settingKey, fallback)
}

func (s *AuthService) fetchSettingIntFromDB(settingKey string, fallback int) int {
	if s.db == nil {
		return fallback
	}

	var rawValue string
	err := s.db.Table("system_setting").
		Select("setting_value").
		Where("setting_key = ?", settingKey).
		Limit(1).
		Pluck("setting_value", &rawValue).Error
	if err != nil {
		return fallback
	}

	value, err := strconv.Atoi(strings.TrimSpace(rawValue))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func (s *AuthService) fetchSettingBoolFromDB(settingKey string, fallback bool) bool {
	if s.db == nil {
		return fallback
	}

	var rawValue string
	err := s.db.Table("system_setting").
		Select("setting_value").
		Where("setting_key = ?", settingKey).
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

func (s *AuthService) ensurePasswordNotRecentlyUsed(userID uint64, newPassword string, currentPasswordHash string, policy authRuntimePolicy) error {
	if policy.PasswordHistoryLimit <= 0 {
		return nil
	}
	if bcrypt.CompareHashAndPassword([]byte(currentPasswordHash), []byte(newPassword)) == nil {
		return errors.New("user.password.error.reused")
	}

	var rows []SystemUserPasswordHistory
	if err := s.db.Where("user_id = ?", userID).
		Order("changed_at desc, id desc").
		Limit(policy.PasswordHistoryLimit).
		Find(&rows).Error; err != nil {
		return err
	}
	for _, row := range rows {
		if bcrypt.CompareHashAndPassword([]byte(row.PasswordHash), []byte(newPassword)) == nil {
			return errors.New("user.password.error.reused")
		}
	}
	return nil
}

func (s *AuthService) recordSecurityEvent(event SystemAuthSecurityEvent) {
	if s.db == nil || !s.getAuthRuntimePolicy().SecurityEventEnabled {
		return
	}
	if strings.TrimSpace(event.EventType) == "" || strings.TrimSpace(event.MessageKey) == "" {
		return
	}
	if strings.TrimSpace(event.Severity) == "" {
		event.Severity = "medium"
	}
	event.SourceKey = strings.TrimSpace(event.SourceKey)
	event.Username = strings.TrimSpace(event.Username)
	_ = s.db.Create(&event).Error
}

func loginSourceIP(sourceKey string) string {
	trimmed := strings.TrimSpace(sourceKey)
	if strings.HasPrefix(trimmed, "ip:") {
		return strings.TrimSpace(strings.TrimPrefix(trimmed, "ip:"))
	}
	return ""
}

type cleanupWindow struct {
	StartedAt time.Time
	EndedAt   time.Time
}

func parseCleanupWindow(startedAt string, endedAt string, invalidErr string) (*cleanupWindow, error) {
	startedAt = strings.TrimSpace(startedAt)
	endedAt = strings.TrimSpace(endedAt)
	if startedAt == "" && endedAt == "" {
		return nil, nil
	}
	if startedAt == "" || endedAt == "" {
		return nil, errors.New(invalidErr)
	}
	start, err := time.Parse(time.RFC3339, startedAt)
	if err != nil {
		return nil, errors.New(invalidErr)
	}
	end, err := time.Parse(time.RFC3339, endedAt)
	if err != nil {
		return nil, errors.New(invalidErr)
	}
	if end.Before(start) {
		return nil, errors.New(invalidErr)
	}
	return &cleanupWindow{StartedAt: start, EndedAt: end}, nil
}

func (s *AuthService) listRecentSecurityEvents(userID uint64, limit int) []SecurityEventResp {
	if s.db == nil || userID == 0 || limit <= 0 {
		return []SecurityEventResp{}
	}
	var events []SystemAuthSecurityEvent
	if err := s.db.Where("user_id = ?", userID).Order("created_at desc, id desc").Limit(limit).Find(&events).Error; err != nil {
		return []SecurityEventResp{}
	}
	return toSecurityEventRespList(events)
}

func toSecurityEventRespList(events []SystemAuthSecurityEvent) []SecurityEventResp {
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
			AcknowledgedAt:      formatOptionalTime(item.AcknowledgedAt),
			AcknowledgedBy:      item.AcknowledgedBy,
			AcknowledgedByUser:  item.AcknowledgedByUser,
			AcknowledgementNote: item.AcknowledgementNote,
			CreatedAt:           item.CreatedAt.Format(time.RFC3339),
		})
	}
	return result
}

func formatOptionalTime(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.Format(time.RFC3339)
	return &formatted
}

func (s *AuthService) passwordExpiresAt(userID uint64, policy authRuntimePolicy) *string {
	if policy.PasswordExpireDays <= 0 {
		return nil
	}
	changedAt := s.passwordLastChangedAt(userID)
	if changedAt.IsZero() {
		return nil
	}
	expiresAt := changedAt.AddDate(0, 0, policy.PasswordExpireDays).Format(time.RFC3339)
	return &expiresAt
}

func (s *AuthService) isPasswordExpired(userID uint64, policy authRuntimePolicy, now time.Time) bool {
	expiresAt := s.passwordExpiresAt(userID, policy)
	if expiresAt == nil {
		return false
	}
	parsed, err := time.Parse(time.RFC3339, *expiresAt)
	if err != nil {
		return false
	}
	return !parsed.After(now)
}

func (s *AuthService) passwordLastChangedAt(userID uint64) time.Time {
	var row SystemUserPasswordHistory
	if err := s.db.Where("user_id = ?", userID).Order("changed_at desc, id desc").First(&row).Error; err == nil {
		return row.ChangedAt
	}
	var currentUser user.SystemUser
	if err := s.db.First(&currentUser, userID).Error; err == nil {
		if !currentUser.UpdatedAt.IsZero() {
			return currentUser.UpdatedAt
		}
		return currentUser.CreatedAt
	}
	return time.Time{}
}

func (s *AuthService) recordFailedLoginAttempt(currentUser *user.SystemUser, policy authRuntimePolicy) (bool, error) {
	if s.db == nil || currentUser == nil {
		return false, errors.New("database.not_initialized")
	}

	nextAttempts := currentUser.FailedLoginAttempts + 1
	updates := map[string]any{
		"failed_login_attempts": nextAttempts,
	}
	if currentUser.LoginLockedUntil != nil && currentUser.LoginLockedUntil.Before(time.Now()) {
		updates["login_locked_until"] = nil
		currentUser.LoginLockedUntil = nil
	}

	if policy.MaxFailedAttempts > 0 && nextAttempts >= policy.MaxFailedAttempts {
		lockUntil := time.Now().Add(time.Duration(maxInt(policy.LockMinutes, 1)) * time.Minute)
		updates["failed_login_attempts"] = 0
		updates["login_locked_until"] = &lockUntil
		currentUser.FailedLoginAttempts = 0
		currentUser.LoginLockedUntil = &lockUntil
		if err := s.db.Model(currentUser).Updates(updates).Error; err != nil {
			return false, err
		}
		return true, nil
	}

	currentUser.FailedLoginAttempts = nextAttempts
	if err := s.db.Model(currentUser).Updates(updates).Error; err != nil {
		return false, err
	}
	return false, nil
}

func (s *AuthService) checkSourceThrottle(sourceKey string, policy authRuntimePolicy, now time.Time) (bool, error) {
	normalizedKey := strings.TrimSpace(sourceKey)
	if normalizedKey == "" || policy.SourceMaxFailedAttempts <= 0 {
		return false, nil
	}

	var throttle SystemLoginThrottle
	if err := s.db.Where("source_key = ?", normalizedKey).First(&throttle).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return false, nil
		}
		return false, err
	}

	if throttle.BlockedUntil != nil && throttle.BlockedUntil.After(now) {
		return true, nil
	}
	if s.isSourceThrottleWindowExpired(throttle.WindowStartedAt, policy, now) || (throttle.BlockedUntil != nil && !throttle.BlockedUntil.After(now)) {
		updates := map[string]any{
			"failure_count":     0,
			"window_started_at": nil,
			"blocked_until":     nil,
		}
		if err := s.db.Model(&throttle).Updates(updates).Error; err != nil {
			return false, err
		}
	}
	return false, nil
}

func (s *AuthService) recordSourceFailure(sourceKey string, policy authRuntimePolicy, now time.Time) (bool, error) {
	normalizedKey := strings.TrimSpace(sourceKey)
	if normalizedKey == "" || policy.SourceMaxFailedAttempts <= 0 {
		return false, nil
	}

	var throttle SystemLoginThrottle
	err := s.db.Where("source_key = ?", normalizedKey).First(&throttle).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		throttle = SystemLoginThrottle{
			SourceKey:       normalizedKey,
			FailureCount:    1,
			WindowStartedAt: &now,
			LastAttemptAt:   &now,
		}
		if policy.SourceMaxFailedAttempts <= 1 {
			blockedUntil := now.Add(time.Duration(maxInt(policy.SourceLockMinutes, 1)) * time.Minute)
			throttle.BlockedUntil = &blockedUntil
		}
		if err := s.db.Create(&throttle).Error; err != nil {
			return false, err
		}
		return throttle.BlockedUntil != nil && throttle.BlockedUntil.After(now), nil
	}
	if err != nil {
		return false, err
	}

	if throttle.BlockedUntil != nil && throttle.BlockedUntil.After(now) {
		return true, nil
	}

	windowStartedAt := throttle.WindowStartedAt
	if s.isSourceThrottleWindowExpired(windowStartedAt, policy, now) || windowStartedAt == nil {
		windowStartedAt = &now
		throttle.FailureCount = 0
		throttle.BlockedUntil = nil
	}

	throttle.FailureCount++
	throttle.WindowStartedAt = windowStartedAt
	throttle.LastAttemptAt = &now

	if throttle.FailureCount >= policy.SourceMaxFailedAttempts {
		blockedUntil := now.Add(time.Duration(maxInt(policy.SourceLockMinutes, 1)) * time.Minute)
		throttle.BlockedUntil = &blockedUntil
	}

	if err := s.db.Model(&throttle).Updates(map[string]any{
		"failure_count":     throttle.FailureCount,
		"window_started_at": throttle.WindowStartedAt,
		"last_attempt_at":   throttle.LastAttemptAt,
		"blocked_until":     throttle.BlockedUntil,
	}).Error; err != nil {
		return false, err
	}
	return throttle.BlockedUntil != nil && throttle.BlockedUntil.After(now), nil
}

func (s *AuthService) isSourceThrottleWindowExpired(windowStartedAt *time.Time, policy authRuntimePolicy, now time.Time) bool {
	if windowStartedAt == nil {
		return true
	}
	windowMinutes := maxInt(policy.SourceWindowMinutes, 1)
	return windowStartedAt.Add(time.Duration(windowMinutes) * time.Minute).Before(now)
}

func (s *AuthService) clearFailedLoginState(userID uint64) error {
	if s.db == nil {
		return errors.New("database.not_initialized")
	}
	return s.db.Model(&user.SystemUser{}).
		Where("id = ? AND (failed_login_attempts <> 0 OR login_locked_until IS NOT NULL)", userID).
		Updates(map[string]any{
			"failed_login_attempts": 0,
			"login_locked_until":    nil,
		}).Error
}

func maxInt(a int, b int) int {
	if a > b {
		return a
	}
	return b
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func (s *AuthService) issueTokenPair(currentUser *user.SystemUser, roles []string, session *SystemUserSession) (*common.TokenPair, error) {
	accessJTI := uuid.NewString()
	pair, err := common.GenerateTokenPair(currentUser.ID, currentUser.Username, roles, session.SessionID, accessJTI, session.RefreshJTI)
	if err != nil {
		return nil, err
	}
	session.RefreshExpiresAt = pair.RefreshExpiresAt
	return pair, nil
}

func truncateString(value string, length int) string {
	if len(value) <= length {
		return value
	}
	return value[:length]
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

func buildSessionResp(item SystemUserSession, currentSessionID string) SessionResp {
	clientInfo := parseClientInfo(item.UserAgent)

	return SessionResp{
		SessionID:        item.SessionID,
		IsCurrent:        item.SessionID == currentSessionID,
		LastIP:           item.LastIP,
		Browser:          clientInfo.Browser,
		OS:               clientInfo.OS,
		Device:           clientInfo.Device,
		UserAgent:        clientInfo.UserAgent,
		RefreshExpiresAt: item.RefreshExpiresAt.Format(time.RFC3339),
		LastRefreshAt:    formatNullableTime(item.LastRefreshAt),
		LastActivityAt:   formatNullableTime(item.LastActivityAt),
		RevokedAt:        formatNullableTime(item.RevokedAt),
		CreatedAt:        item.CreatedAt.Format(time.RFC3339),
	}
}

func formatNullableTime(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.Format(time.RFC3339)
	return &formatted
}

func normalizePageQuery(page int, pageSize int) (int, int) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}
	return page, pageSize
}

func normalizeSecurityEventPageQuery(query *SecurityEventQuery) (int, int) {
	if query == nil {
		return 1, 10
	}
	return normalizePageQuery(query.Page, query.PageSize)
}

func normalizeUint64IDs(ids []uint64) []uint64 {
	if len(ids) == 0 {
		return nil
	}

	seen := make(map[uint64]struct{}, len(ids))
	result := make([]uint64, 0, len(ids))
	for _, id := range ids {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	return result
}

func normalizeSessionIDs(ids []string) []string {
	if len(ids) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(ids))
	result := make([]string, 0, len(ids))
	for _, id := range ids {
		normalized := strings.TrimSpace(id)
		if normalized == "" {
			continue
		}
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		result = append(result, normalized)
	}
	return result
}

func queryPage(query *LoginLogQuery) int {
	if query == nil {
		return 1
	}
	return query.Page
}

func queryPageSize(query *LoginLogQuery) int {
	if query == nil {
		return 10
	}
	return query.PageSize
}

func queryStatus(query *LoginLogQuery) *int {
	if query == nil {
		return nil
	}
	return query.Status
}

func queryPageFromAdminSession(query *AdminSessionQuery) int {
	if query == nil {
		return 1
	}
	return query.Page
}

func queryPageSizeFromAdminSession(query *AdminSessionQuery) int {
	if query == nil {
		return 10
	}
	return query.PageSize
}
