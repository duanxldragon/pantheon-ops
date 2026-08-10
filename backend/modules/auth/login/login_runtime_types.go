package login

import (
	"pantheon-ops/backend/modules/auth/mfa"
	"pantheon-ops/backend/modules/auth/security"
	"pantheon-ops/backend/modules/auth/session"
)

type UserInfoResp = security.UserInfoResp

type AuthTokenResp struct {
	Token            string                 `json:"-"`
	AccessToken      string                 `json:"-"`
	RefreshToken     string                 `json:"-"`
	TokenType        string                 `json:"tokenType"`
	AccessExpiresAt  string                 `json:"accessExpiresAt"`
	RefreshExpiresAt string                 `json:"refreshExpiresAt"`
	SessionID        string                 `json:"sessionId"`
	User             *security.UserInfoResp `json:"user,omitempty"`
}

type UserPlatformPreferenceUpdateReq = security.UserPlatformPreferenceUpdateReq
type UserPreferenceUpdateResult = security.UserPreferenceUpdateResult
type SecurityOverviewResp = security.SecurityOverviewResp
type SecurityPolicyResp = security.SecurityPolicyResp
type SecurityEventQuery = security.SecurityEventQuery
type SecurityEventResp = security.SecurityEventResp
type SecurityEventPageResp = security.SecurityEventPageResp
type SecurityEventAcknowledgeReq = security.SecurityEventAcknowledgeReq
type PasswordUpdateReq = security.PasswordChangeReq

type SessionResp = session.SessionResp
type SessionCleanupReq = session.SessionCleanupReq
type SessionBatchRevokeReq = session.SessionBatchRevokeReq
type SessionCleanupResp = session.SessionCleanupResp
type AdminSessionQuery = session.AdminSessionQuery
type AdminSessionResp = session.AdminSessionResp
type AdminSessionPageResp = session.AdminSessionPageResp

type MFAVerifyReq = mfa.MFAVerifyReq
type MFAChallengeResp = mfa.MFAChallengeResp

// TokenRefreshClaims is used for refresh token parsing in auth handler.
type TokenRefreshClaims struct {
	UserID    uint64
	SessionID string
	ID        string
}

type SystemAuthFactor = mfa.SystemAuthFactor
type SystemAuthMFAChallenge = mfa.SystemAuthMFAChallenge
type SystemUserSession = session.SystemUserSession
type SystemAuthSecurityEvent = security.SystemAuthSecurityEvent
type SystemUserPasswordHistory = security.SystemUserPasswordHistory
