package session

import (
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/pkg/authsession"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"

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
	if strings.TrimSpace(query.Keyword) != "" {
		keyword := "%" + common.EscapeLikePattern(strings.TrimSpace(query.Keyword)) + "%"
		db = db.Where("system_user.username LIKE ? OR system_user_session.last_ip LIKE ?", keyword, keyword)
	}
	if strings.TrimSpace(query.Username) != "" {
		db = db.Where("system_user.username LIKE ?", "%"+common.EscapeLikePattern(strings.TrimSpace(query.Username))+"%")
	}
	if strings.TrimSpace(query.LastIP) != "" {
		db = db.Where("system_user_session.last_ip LIKE ?", "%"+common.EscapeLikePattern(strings.TrimSpace(query.LastIP))+"%")
	}
	if start, ok := parseSessionFilterTime(query.StartedAt); ok {
		db = db.Where("system_user_session.created_at >= ?", start)
	}
	if end, ok := parseSessionFilterTime(query.EndedAt); ok {
		db = db.Where("system_user_session.created_at <= ?", end)
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

// parseSessionFilterTime accepts the same formats the login-log list filter
// does, so all audit toolbars share one frontend time-range component.
func parseSessionFilterTime(value string) (time.Time, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, false
	}
	for _, layout := range []string{time.RFC3339, "2006-01-02 15:04:05", "2006-01-02 15:04"} {
		if parsed, err := time.ParseInLocation(layout, value, time.Local); err == nil {
			return parsed, true
		}
	}
	return time.Time{}, false
}

const (
	// sessionUserAgentLowerExpr is the lower-cased user_agent expression every
	// client-side filter matches against.
	sessionUserAgentLowerExpr     = "LOWER(system_user_session.user_agent)"
	sessionUserAgentLikeClause    = sessionUserAgentLowerExpr + " LIKE ?"
	sessionUserAgentNotLikeClause = sessionUserAgentLowerExpr + " NOT LIKE ?"
	mobileTokenPattern            = "%mobile%"
	androidTokenPattern           = "%android%"
)

// applyAdminSessionClientFilters pushes the browser/OS/device filters into
// SQL as user_agent LIKE conditions. The detection tokens mirror
// DetectBrowser/DetectOS/DetectDevice in session_user_agent.go (lowercase
// substring probes), so SQL-side matching stays consistent with the client
// info rendered on each row and counts cannot drift from pages.
func applyAdminSessionClientFilters(db *gorm.DB, query *AdminSessionQuery) *gorm.DB {
	if query == nil {
		return db
	}
	if browser := strings.ToLower(strings.TrimSpace(query.Browser)); browser != "" && browser != "unknown" {
		if token, ok := browserDetectionTokens[browser]; ok {
			db = db.Where(sessionUserAgentLikeClause, "%"+common.EscapeLikePattern(token)+"%")
		} else {
			db = db.Where(sessionUserAgentLikeClause, "%"+common.EscapeLikePattern(browser)+"%")
		}
	}
	if os := strings.ToLower(strings.TrimSpace(query.OS)); os != "" && os != "unknown" {
		if token, ok := osDetectionTokens[os]; ok {
			db = db.Where(sessionUserAgentLikeClause, "%"+common.EscapeLikePattern(token)+"%")
		} else {
			db = db.Where(sessionUserAgentLikeClause, "%"+common.EscapeLikePattern(os)+"%")
		}
	}
	if device := strings.ToLower(strings.TrimSpace(query.Device)); device != "" {
		db = applyAdminSessionDeviceFilter(db, device)
	}
	return db
}

// applyAdminSessionDeviceFilter maps the device filter to the same logic as
// DetectDevice: Android Phone requires android+mobile, Android Tablet
// requires android without mobile, Desktop requires no mobile token.
func applyAdminSessionDeviceFilter(db *gorm.DB, device string) *gorm.DB {
	switch device {
	case "android phone":
		return db.Where(sessionUserAgentLikeClause+" AND "+sessionUserAgentLikeClause, androidTokenPattern, mobileTokenPattern)
	case "android tablet":
		return db.Where(sessionUserAgentLikeClause+" AND "+sessionUserAgentNotLikeClause, androidTokenPattern, mobileTokenPattern)
	case "mobile":
		return db.Where(sessionUserAgentLikeClause, mobileTokenPattern)
	case "desktop":
		return db.Where(sessionUserAgentNotLikeClause, mobileTokenPattern)
	case "ipad", "iphone":
		return db.Where(sessionUserAgentLikeClause, "%"+common.EscapeLikePattern(device)+"%")
	default:
		return db.Where(sessionUserAgentLikeClause, "%"+common.EscapeLikePattern(device)+"%")
	}
}

// browserDetectionTokens maps the public browser filter values to the first
// detection token DetectBrowser would match (keep in sync with
// session_user_agent.go).
var browserDetectionTokens = map[string]string{
	"chrome":  "chrome/",
	"edge":    "edg/",
	"opera":   "opr/",
	"firefox": "firefox/",
	"safari":  "version/",
	"wechat":  "micromessenger",
}

// osDetectionTokens maps the public OS filter values to detection tokens
// (keep in sync with DetectOS).
var osDetectionTokens = map[string]string{
	"windows": "windows",
	"macos":   "mac os x",
	"ios":     "iphone",
	"android": "android",
	"linux":   "linux",
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
