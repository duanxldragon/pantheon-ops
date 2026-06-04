package auth

import user "pantheon-ops/backend/modules/system/iam/user"

// LoginReq 登录请求 DTO
type LoginReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type MFAVerifyReq struct {
	ChallengeID string `json:"challengeId" binding:"required"`
	Code        string `json:"code" binding:"required"`
}

// RefreshTokenReq 刷新令牌请求 DTO
type RefreshTokenReq struct {
	RefreshToken string `json:"refreshToken" binding:"required"`
}

// PasswordUpdateReq 当前登录用户修改密码请求 DTO
type PasswordUpdateReq struct {
	OldPassword string `json:"oldPassword" binding:"required"`
	NewPassword string `json:"newPassword" binding:"required"`
}

// UserInfoResp 当前登录主体信息 DTO
type UserInfoResp struct {
	ID          uint64                           `json:"id"`
	Username    string                           `json:"username"`
	Nickname    string                           `json:"nickname"`
	Avatar      string                           `json:"avatar"`
	Email       string                           `json:"email"`
	Phone       string                           `json:"phone"`
	Roles       []string                         `json:"roles"`
	Perms       []string                         `json:"perms"`
	Preferences *user.UserPlatformPreferenceResp `json:"preferences,omitempty"`
}

type UserPlatformPreferenceUpdateReq struct {
	Theme       string `json:"theme"`
	Language    string `json:"language"`
	LayoutMode  string `json:"layoutMode"`
	DensityMode string `json:"densityMode"`
}

type ClientInfoResp struct {
	Browser   string `json:"browser"`
	OS        string `json:"os"`
	Device    string `json:"device"`
	UserAgent string `json:"userAgent"`
}

// AuthTokenResp 认证返回 DTO
type AuthTokenResp struct {
	MFARequired      bool          `json:"mfaRequired,omitempty"`
	ChallengeID      string        `json:"challengeId,omitempty"`
	SetupRequired    bool          `json:"setupRequired,omitempty"`
	TOTPSecret       string        `json:"totpSecret,omitempty"`
	TOTPProvisionURI string        `json:"totpProvisionUri,omitempty"`
	Token            string        `json:"token"`
	AccessToken      string        `json:"accessToken"`
	RefreshToken     string        `json:"refreshToken"`
	TokenType        string        `json:"tokenType"`
	AccessExpiresAt  string        `json:"accessExpiresAt"`
	RefreshExpiresAt string        `json:"refreshExpiresAt"`
	SessionID        string        `json:"sessionId"`
	User             *UserInfoResp `json:"user"`
}

type MFAChallengeResp struct {
	MFARequired      bool   `json:"mfaRequired"`
	ChallengeID      string `json:"challengeId"`
	SetupRequired    bool   `json:"setupRequired"`
	TOTPSecret       string `json:"totpSecret,omitempty"`
	TOTPProvisionURI string `json:"totpProvisionUri,omitempty"`
	ExpiresAt        string `json:"expiresAt"`
}

// SessionResp 当前用户会话信息 DTO
type SessionResp struct {
	SessionID        string  `json:"sessionId"`
	IsCurrent        bool    `json:"isCurrent"`
	LastIP           string  `json:"lastIp"`
	Browser          string  `json:"browser"`
	OS               string  `json:"os"`
	Device           string  `json:"device"`
	UserAgent        string  `json:"userAgent"`
	RefreshExpiresAt string  `json:"refreshExpiresAt"`
	LastRefreshAt    *string `json:"lastRefreshAt"`
	LastActivityAt   *string `json:"lastActivityAt"`
	RevokedAt        *string `json:"revokedAt"`
	CreatedAt        string  `json:"createdAt"`
}

type SecurityOverviewResp struct {
	User                 *UserInfoResp       `json:"user"`
	CurrentSession       *SessionResp        `json:"currentSession"`
	ActiveSessionCount   int64               `json:"activeSessionCount"`
	LastLoginAt          *string             `json:"lastLoginAt"`
	PasswordExpired      bool                `json:"passwordExpired"`
	PasswordExpiresAt    *string             `json:"passwordExpiresAt"`
	RecentSecurityEvents []SecurityEventResp `json:"recentSecurityEvents"`
	Policy               SecurityPolicyResp  `json:"policy"`
}

type SecurityPolicyResp struct {
	PasswordMinLength       int  `json:"passwordMinLength"`
	PasswordRequireDigit    bool `json:"passwordRequireDigit"`
	PasswordRequireUpper    bool `json:"passwordRequireUpper"`
	PasswordHistoryLimit    int  `json:"passwordHistoryLimit"`
	PasswordExpireDays      int  `json:"passwordExpireDays"`
	MaxFailedAttempts       int  `json:"maxFailedAttempts"`
	LockMinutes             int  `json:"lockMinutes"`
	SourceMaxFailedAttempts int  `json:"sourceMaxFailedAttempts"`
	SourceWindowMinutes     int  `json:"sourceWindowMinutes"`
	SourceLockMinutes       int  `json:"sourceLockMinutes"`
	SessionIdleMinutes      int  `json:"sessionIdleMinutes"`
	MaxActiveSessions       int  `json:"maxActiveSessions"`
	SessionRetentionDays    int  `json:"sessionRetentionDays"`
	CaptchaEnabled          bool `json:"captchaEnabled"`
	MFAEnabled              bool `json:"mfaEnabled"`
	SSOEnabled              bool `json:"ssoEnabled"`
}

type SecurityEventQuery struct {
	Username     string `form:"username" json:"username"`
	EventType    string `form:"eventType" json:"eventType"`
	Severity     string `form:"severity" json:"severity"`
	Acknowledged *bool  `form:"acknowledged" json:"acknowledged"`
	Page         int    `form:"page" json:"page"`
	PageSize     int    `form:"pageSize" json:"pageSize"`
}

type SecurityEventResp struct {
	ID                  uint64  `json:"id"`
	UserID              uint64  `json:"userId"`
	Username            string  `json:"username"`
	EventType           string  `json:"eventType"`
	Severity            string  `json:"severity"`
	SourceKey           string  `json:"sourceKey"`
	IP                  string  `json:"ip"`
	UserAgent           string  `json:"userAgent"`
	MessageKey          string  `json:"messageKey"`
	Metadata            string  `json:"metadata"`
	AcknowledgedAt      *string `json:"acknowledgedAt"`
	AcknowledgedBy      uint64  `json:"acknowledgedBy"`
	AcknowledgedByUser  string  `json:"acknowledgedByUser"`
	AcknowledgementNote string  `json:"acknowledgementNote"`
	CreatedAt           string  `json:"createdAt"`
}

type SecurityEventPageResp struct {
	Items    []SecurityEventResp `json:"items"`
	Total    int64               `json:"total"`
	Page     int                 `json:"page"`
	PageSize int                 `json:"pageSize"`
}

// LoginLogQuery 登录日志查询
type LoginLogQuery struct {
	Username string `form:"username" json:"username"`
	Status   *int   `form:"status" json:"status"`
	Page     int    `form:"page" json:"page"`
	PageSize int    `form:"pageSize" json:"pageSize"`
}

type LoginLogCleanupReq struct {
	RetentionDays int    `json:"retentionDays"`
	StartedAt     string `json:"startedAt"`
	EndedAt       string `json:"endedAt"`
}

type LoginLogBatchDeleteReq struct {
	IDs []uint64 `json:"ids"`
}

// LoginLogResp 登录日志 DTO
type LoginLogResp struct {
	ID            uint64 `json:"id"`
	Username      string `json:"username"`
	Ipaddr        string `json:"ipaddr"`
	LoginLocation string `json:"loginLocation"`
	Browser       string `json:"browser"`
	Os            string `json:"os"`
	Status        int    `json:"status"`
	Msg           string `json:"msg"`
	LoginTime     string `json:"loginTime"`
}

type LoginLogPageResp struct {
	Items    []LoginLogResp `json:"items"`
	Total    int64          `json:"total"`
	Page     int            `json:"page"`
	PageSize int            `json:"pageSize"`
}

type LoginLogCleanupResp struct {
	ClearedCount int64 `json:"clearedCount"`
}

type SessionCleanupReq struct {
	RetentionDays int    `json:"retentionDays"`
	StartedAt     string `json:"startedAt"`
	EndedAt       string `json:"endedAt"`
}

type SessionBatchRevokeReq struct {
	SessionIDs []string `json:"sessionIds"`
}

type SessionCleanupResp struct {
	ClearedCount int64 `json:"clearedCount"`
}

type SecurityEventAcknowledgeReq struct {
	AcknowledgementNote string `json:"acknowledgementNote"`
}

// AdminSessionQuery 管理员会话查询
type AdminSessionQuery struct {
	Username string `form:"username" json:"username"`
	LastIP   string `form:"lastIp" json:"lastIp"`
	Browser  string `form:"browser" json:"browser"`
	OS       string `form:"os" json:"os"`
	Device   string `form:"device" json:"device"`
	Status   *int   `form:"status" json:"status"`
	Page     int    `form:"page" json:"page"`
	PageSize int    `form:"pageSize" json:"pageSize"`
}

// AdminSessionResp 管理员会话 DTO
type AdminSessionResp struct {
	SessionID        string  `json:"sessionId"`
	UserID           uint64  `json:"userId"`
	Username         string  `json:"username"`
	Nickname         string  `json:"nickname"`
	LastIP           string  `json:"lastIp"`
	Browser          string  `json:"browser"`
	OS               string  `json:"os"`
	Device           string  `json:"device"`
	UserAgent        string  `json:"userAgent"`
	RefreshExpiresAt string  `json:"refreshExpiresAt"`
	LastRefreshAt    *string `json:"lastRefreshAt"`
	LastActivityAt   *string `json:"lastActivityAt"`
	RevokedAt        *string `json:"revokedAt"`
	CreatedAt        string  `json:"createdAt"`
}

type AdminSessionPageResp struct {
	Items        []AdminSessionResp `json:"items"`
	Total        int64              `json:"total"`
	ActiveCount  int64              `json:"activeCount"`
	RevokedCount int64              `json:"revokedCount"`
	Page         int                `json:"page"`
	PageSize     int                `json:"pageSize"`
}
