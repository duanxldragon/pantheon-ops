package session

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"pantheon-ops/backend/pkg/authsession"
	"pantheon-ops/backend/pkg/authtoken"
	"pantheon-ops/backend/pkg/common"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PolicyProvider abstracts runtime auth policy so SessionService stays decoupled from AuthService.
type PolicyProvider interface {
	GetSessionPolicy() AuthRuntimePolicy
}

// UserRef holds the minimal user identity needed by session operations.
type UserRef struct {
	ID       uint64
	Username string
}

// UserRoleLoader abstracts user+role lookups needed by RefreshSession.
type UserRoleLoader interface {
	GetUserByID(userID uint64) (*UserRef, error)
	GetUserRoles(userID uint64) ([]string, error)
}

// TokenIssuer abstracts token issuance so SessionService stays decoupled from AuthService.
type TokenIssuer interface {
	IssueTokenPairWithContext(ctx context.Context, userID uint64, username string, roles []string, sess *SystemUserSession) (*authtoken.Pair, error)
}

// Service owns session lifecycle and query operations for the auth domain.
type Service struct {
	db     *gorm.DB
	policy PolicyProvider
	loader UserRoleLoader
	issuer TokenIssuer
}

// NewService creates a SessionService.
func NewService(db *gorm.DB, policy PolicyProvider, loader UserRoleLoader, issuer TokenIssuer) *Service {
	return &Service{db: db, policy: policy, loader: loader, issuer: issuer}
}

func (s *Service) governSessionInventory(now time.Time, policy AuthRuntimePolicy) error {
	if err := authsession.CleanupInactiveSessions(s.db, now, policy.SessionIdleMinutes); err != nil {
		return err
	}
	return authsession.PurgeHistoricSessions(s.db, now, policy.SessionRetentionDays)
}

// RefreshSession refreshes an active session and issues a new token pair.
func (s *Service) RefreshSession(sessionID string, userID uint64, ip, userAgent string) (*authtoken.Pair, error) {
	return s.RefreshSessionWithContext(context.Background(), sessionID, userID, ip, userAgent)
}

func (s *Service) RefreshSessionWithContext(ctx context.Context, sessionID string, userID uint64, ip, userAgent string) (*authtoken.Pair, error) {
	if s.db == nil {
		return nil, common.ErrDatabaseNotInitialized
	}
	if ctx == nil {
		ctx = context.Background()
	}
	var sess SystemUserSession
	if err := s.db.Where(sessionIDAndUserIDWhereClause, sessionID, userID).First(&sess).Error; err != nil {
		return nil, err
	}
	if sess.RevokedAt != nil || sess.RefreshExpiresAt.Before(time.Now()) {
		return nil, common.ErrUnauthorized
	}

	u, err := s.loader.GetUserByID(userID)
	if err != nil {
		return nil, err
	}
	roles, err := s.loader.GetUserRoles(u.ID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	sess.RefreshJTI = uuid.NewString()
	sess.RefreshExpiresAt = now.Add(authtoken.RefreshTokenTTL)
	sess.LastRefreshAt = &now
	sess.LastActivityAt = &now
	sess.LastIP = NormalizeSessionClientIP(ip)
	sess.UserAgent = TruncateString(userAgent, 255)
	if err := s.db.Save(&sess).Error; err != nil {
		return nil, err
	}
	return s.issuer.IssueTokenPairWithContext(ctx, u.ID, u.Username, roles, &sess)
}

// RevokeSession marks a session as revoked.
func (s *Service) RevokeSession(sessionID string) error {
	if s.db == nil || sessionID == "" {
		return nil
	}
	now := time.Now()
	return s.db.Model(&SystemUserSession{}).
		Where("session_id = ? AND revoked_at IS NULL", sessionID).
		Updates(map[string]interface{}{"revoked_at": &now}).Error
}

// TouchSessionActivity updates last_activity_at for an active session.
func (s *Service) TouchSessionActivity(sessionID string, userID uint64, ip, userAgent string) error {
	if s.db == nil || strings.TrimSpace(sessionID) == "" || userID == 0 {
		return nil
	}
	now := time.Now()
	clientIP := NormalizeSessionClientIP(ip)
	agent := NormalizeSessionUserAgent(userAgent)
	return s.db.Exec(
		touchSessionActivitySQL,
		now, clientIP, clientIP, agent, agent,
		sessionID, userID,
		now.Add(-1*time.Minute),
	).Error
}

// ListSessions returns active sessions for a user.
func (s *Service) ListSessions(userID uint64, currentSessionID string) ([]SessionResp, error) {
	if s.db == nil {
		return nil, common.ErrDatabaseNotInitialized
	}
	now := time.Now()
	policy := s.policy.GetSessionPolicy()
	if err := s.governSessionInventory(now, policy); err != nil {
		return nil, err
	}

	var sessions []SystemUserSession
	if err := authsession.ApplyActiveScope(s.db, "", now, policy.SessionIdleMinutes).
		Where(userIDWhereClause, userID).
		Order("created_at desc").
		Find(&sessions).Error; err != nil {
		return nil, err
	}

	result := make([]SessionResp, 0, len(sessions))
	for _, item := range sessions {
		result = append(result, BuildSessionResp(item, currentSessionID))
	}
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].IsCurrent != result[j].IsCurrent {
			return result[i].IsCurrent
		}
		return result[i].CreatedAt > result[j].CreatedAt
	})
	return result, nil
}

// RevokeOwnedSession revokes a specific session owned by the user (not current session).
func (s *Service) RevokeOwnedSession(userID uint64, currentSessionID, targetSessionID string) error {
	if s.db == nil {
		return common.ErrDatabaseNotInitialized
	}
	if strings.TrimSpace(targetSessionID) == "" {
		return common.ErrUnauthorized
	}
	if targetSessionID == currentSessionID {
		return common.ErrUnauthorized
	}
	var sess SystemUserSession
	if err := s.db.Where(sessionIDAndUserIDWhereClause, targetSessionID, userID).First(&sess).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return common.ErrUnauthorized
		}
		return err
	}
	if sess.RevokedAt != nil {
		return nil
	}
	now := time.Now()
	return s.db.Model(&SystemUserSession{}).
		Where(sessionIDAndActiveUserIDWhereClause, targetSessionID, userID).
		Updates(map[string]interface{}{"revoked_at": &now}).Error
}

// CleanupHistoricSessions removes expired session records.
func (s *Service) CleanupHistoricSessions(retentionDays int, startedAt, endedAt string) (int64, error) {
	if s.db == nil {
		return 0, common.ErrDatabaseNotInitialized
	}
	window, err := parseCleanupWindow(startedAt, endedAt, "auth.session.cleanup.range_invalid")
	if err != nil {
		return 0, err
	}
	now := time.Now()
	policy := s.policy.GetSessionPolicy()
	if err := s.governSessionInventory(now, policy); err != nil {
		return 0, err
	}
	db := s.db.Table("system_user_session").Where("revoked_at IS NOT NULL")
	if window != nil {
		db = db.Where("revoked_at >= ? AND revoked_at <= ?", window.StartedAt, window.EndedAt)
	} else {
		if !isAllowedSessionCleanupRetentionDays(retentionDays, policy.CleanupRetentionDays) {
			return 0, errors.New("auth.session.cleanup.days_invalid")
		}
		cutoff := now.AddDate(0, 0, -retentionDays)
		db = db.Where("revoked_at < ?", cutoff)
	}
	result := db.Delete(nil)
	return result.RowsAffected, result.Error
}

// BatchRevokeSessions revokes multiple sessions in one operation.
func (s *Service) BatchRevokeSessions(currentSessionID string, sessionIDs []string) (int64, error) {
	if s.db == nil {
		return 0, common.ErrDatabaseNotInitialized
	}
	normalized := normalizeSessionIDs(sessionIDs)
	if len(normalized) == 0 {
		return 0, common.ErrUnauthorized
	}
	for _, sid := range normalized {
		if sid == currentSessionID {
			return 0, errors.New("auth.session.current_revoke_forbidden")
		}
	}
	now := time.Now()
	result := s.db.Model(&SystemUserSession{}).
		Where("session_id IN ? AND revoked_at IS NULL", normalized).
		Updates(map[string]interface{}{"revoked_at": &now})
	return result.RowsAffected, result.Error
}

// ListAllSessions returns paginated session records for admin use.
func (s *Service) ListAllSessions(query *AdminSessionQuery) (*AdminSessionPageResp, error) {
	if s.db == nil {
		return nil, common.ErrDatabaseNotInitialized
	}
	now := time.Now()
	policy := s.policy.GetSessionPolicy()
	if err := s.governSessionInventory(now, policy); err != nil {
		return nil, err
	}

	page, pageSize := normalizePageQuery(queryPageFromAdminSession(query), queryPageSizeFromAdminSession(query))
	db := s.db.Table("system_user_session").
		Select("system_user_session.session_id, system_user_session.user_id, system_user.username, system_user.nickname, system_user_session.last_ip, system_user_session.user_agent, system_user_session.refresh_expires_at, system_user_session.last_refresh_at, system_user_session.last_activity_at, system_user_session.revoked_at, system_user_session.created_at").
		Joins("LEFT JOIN system_user ON system_user.id = system_user_session.user_id")
	db = applyAdminSessionFilters(db, query, now, policy)

	var rows []adminSessionRow
	if err := db.Order("system_user_session.created_at desc").Scan(&rows).Error; err != nil {
		return nil, err
	}

	items := make([]AdminSessionResp, 0, len(rows))
	var activeCount, revokedCount int64
	for _, row := range rows {
		clientInfo := ParseClientInfo(row.UserAgent)
		if !matchesAdminSessionFilters(query, clientInfo) {
			continue
		}
		if row.RevokedAt == nil {
			activeCount++
		} else {
			revokedCount++
		}
		items = append(items, buildAdminSessionResp(row, clientInfo))
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

// RevokeAnySession allows an admin to revoke any session.
func (s *Service) RevokeAnySession(currentSessionID, targetSessionID string) error {
	if s.db == nil {
		return common.ErrDatabaseNotInitialized
	}
	if strings.TrimSpace(targetSessionID) == "" {
		return common.ErrUnauthorized
	}
	if targetSessionID == currentSessionID {
		return common.ErrUnauthorized
	}
	now := time.Now()
	return s.db.Model(&SystemUserSession{}).
		Where("session_id = ? AND revoked_at IS NULL", targetSessionID).
		Updates(map[string]interface{}{"revoked_at": &now}).Error
}
