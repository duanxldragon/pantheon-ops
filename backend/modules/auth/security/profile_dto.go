package security

import (
	"pantheon-ops/backend/modules/auth/session"
	"pantheon-ops/backend/pkg/platformprefs"
)

// UserInfoResp 当前登录主体信息 DTO
type UserInfoResp struct {
	ID          uint64                            `json:"id"`
	Username    string                            `json:"username"`
	Nickname    string                            `json:"nickname"`
	Avatar      string                            `json:"avatar"`
	Email       string                            `json:"email"`
	Phone       string                            `json:"phone"`
	Roles       []string                          `json:"roles"`
	Perms       []string                          `json:"perms"`
	Preferences *platformprefs.PlatformPreference `json:"preferences,omitempty"`
}

type UserPlatformPreferenceUpdateReq struct {
	Theme       string `json:"theme"`
	Language    string `json:"language"`
	LayoutMode  string `json:"layoutMode"`
	DensityMode string `json:"densityMode"`
}

type UserPreferenceUpdateResult struct {
	User      *UserInfoResp
	Previous  *platformprefs.PlatformPreference
	Current   *platformprefs.PlatformPreference
	Persisted string
}

type SecurityOverviewResp struct {
	User                 *UserInfoResp        `json:"user"`
	CurrentSession       *session.SessionResp `json:"currentSession"`
	ActiveSessionCount   int64                `json:"activeSessionCount"`
	LastLoginAt          *string              `json:"lastLoginAt"`
	PasswordExpired      bool                 `json:"passwordExpired"`
	PasswordExpiresAt    *string              `json:"passwordExpiresAt"`
	RecentSecurityEvents []SecurityEventResp  `json:"recentSecurityEvents"`
	Policy               SecurityPolicyResp   `json:"policy"`
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

type SecurityEventAcknowledgeReq struct {
	AcknowledgementNote string `json:"acknowledgementNote"`
}
