package session

import "time"

// SystemUserSession 登录会话模型
type SystemUserSession struct {
	SessionID        string     `gorm:"primaryKey;size:64" json:"sessionId"`
	UserID           uint64     `gorm:"index;not null" json:"userId"`
	RefreshJTI       string     `gorm:"size:64;not null" json:"refreshJti"`
	RefreshExpiresAt time.Time  `json:"refreshExpiresAt"`
	LastRefreshAt    *time.Time `json:"lastRefreshAt"`
	LastActivityAt   *time.Time `json:"lastActivityAt"`
	LastIP           string     `gorm:"size:128" json:"lastIp"`
	UserAgent        string     `gorm:"size:255" json:"userAgent"`
	RevokedAt        *time.Time `json:"revokedAt"`
	CreatedAt        time.Time  `json:"createdAt"`
	UpdatedAt        time.Time  `json:"updatedAt"`
}

func (SystemUserSession) TableName() string {
	return "system_user_session"
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

type ClientInfoResp struct {
	Browser   string `json:"browser"`
	OS        string `json:"os"`
	Device    string `json:"device"`
	UserAgent string `json:"userAgent"`
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

// ─────────────────────────────────────────────────────────────
// SQL constants used by session service
// ─────────────────────────────────────────────────────────────

const (
	sessionIDAndUserIDWhereClause       = "session_id = ? AND user_id = ?"
	sessionIDAndActiveUserIDWhereClause = sessionIDAndUserIDWhereClause + " AND revoked_at IS NULL"
	userIDWhereClause                   = "user_id = ?"

	touchSessionActivitySQL = `UPDATE system_user_session
SET last_activity_at = ?,
    last_ip = CASE WHEN ? <> '' THEN ? ELSE last_ip END,
    user_agent = CASE WHEN ? <> '' THEN ? ELSE user_agent END
WHERE session_id = ? AND user_id = ? AND revoked_at IS NULL
  AND (last_activity_at IS NULL OR last_activity_at < ?)`
)

// AdminRuntimePolicy is the subset of auth policy needed by session queries.
type AdminRuntimePolicy struct {
	SessionIdleMinutes   int
	SessionRetentionDays int
}
