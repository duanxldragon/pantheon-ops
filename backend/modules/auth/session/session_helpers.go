package session

import (
	"errors"
	"sort"
	"strings"
	"time"

	"pantheon-ops/backend/pkg/authsession"
	"pantheon-ops/backend/pkg/common"

	"gorm.io/gorm"
)

// AuthRuntimePolicy mirrors the subset of auth policy needed by SessionService.
type AuthRuntimePolicy struct {
	SessionIdleMinutes   int
	SessionRetentionDays int
	MaxActiveSessions    int
	CleanupRetentionDays []int
}

type cleanupWindow struct {
	StartedAt time.Time
	EndedAt   time.Time
}

func parseCleanupWindow(startedAt, endedAt, invalidErr string) (*cleanupWindow, error) {
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

func isAllowedSessionCleanupRetentionDays(retentionDays int, allowedDays []int) bool {
	if len(allowedDays) == 0 {
		allowedDays = []int{1, 7, 30}
	}
	for _, allowed := range allowedDays {
		if allowed == retentionDays {
			return true
		}
	}
	return false
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

func normalizePageQuery(page, pageSize int) (int, int) {
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

type adminSessionRow struct {
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

func applyAdminSessionFilters(db *gorm.DB, query *AdminSessionQuery, now time.Time, policy AuthRuntimePolicy) *gorm.DB {
	if query == nil {
		return db
	}
	if strings.TrimSpace(query.Username) != "" {
		db = db.Where("system_user.username LIKE ?", "%"+common.EscapeLikePattern(strings.TrimSpace(query.Username))+"%")
	}
	if strings.TrimSpace(query.LastIP) != "" {
		db = db.Where("system_user_session.last_ip LIKE ?", "%"+common.EscapeLikePattern(strings.TrimSpace(query.LastIP))+"%")
	}
	if query.Status == nil {
		return db
	}
	if *query.Status == common.SessionStatusActive {
		return authsession.ApplyActiveScope(db, "system_user_session", now, policy.SessionIdleMinutes)
	}
	if *query.Status == common.SessionStatusRevoked {
		return db.Where("system_user_session.revoked_at IS NOT NULL")
	}
	return db
}

func matchesAdminSessionFilters(query *AdminSessionQuery, clientInfo ClientInfoResp) bool {
	if query == nil {
		return true
	}
	if strings.TrimSpace(query.Browser) != "" && !strings.EqualFold(strings.TrimSpace(query.Browser), clientInfo.Browser) {
		return false
	}
	if strings.TrimSpace(query.OS) != "" && !strings.EqualFold(strings.TrimSpace(query.OS), clientInfo.OS) {
		return false
	}
	if strings.TrimSpace(query.Device) != "" && !strings.EqualFold(strings.TrimSpace(query.Device), clientInfo.Device) {
		return false
	}
	return true
}

func buildAdminSessionResp(row adminSessionRow, clientInfo ClientInfoResp) AdminSessionResp {
	return AdminSessionResp{
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
		LastRefreshAt:    FormatNullableTime(row.LastRefreshAt),
		LastActivityAt:   FormatNullableTime(row.LastActivityAt),
		RevokedAt:        FormatNullableTime(row.RevokedAt),
		CreatedAt:        row.CreatedAt.Format(time.RFC3339),
	}
}

// SortSessions sorts a slice of SessionResp by (current first, then by created_at desc).
func SortSessions(items []SessionResp, currentSessionID string) {
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].IsCurrent != items[j].IsCurrent {
			return items[i].IsCurrent
		}
		return items[i].CreatedAt > items[j].CreatedAt
	})
}
