package auth

import (
	"errors"
	"sort"
	"strings"
	"time"

	user "pantheon-ops/backend/modules/system/iam/user"
	"pantheon-ops/backend/pkg/authsession"
	"pantheon-ops/backend/pkg/common"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type authSessionService struct {
	auth *AuthService
}

func newAuthSessionService(auth *AuthService) *authSessionService {
	return &authSessionService{auth: auth}
}

func (s *authSessionService) governInventory(now time.Time, policy authRuntimePolicy) error {
	return s.auth.governSessionInventory(now, policy)
}

func (s *authSessionService) countActiveSessions(userID uint64, now time.Time, policy authRuntimePolicy) (int64, error) {
	var activeSessionCount int64
	if err := authsession.ApplyActiveScope(s.auth.db.Model(&SystemUserSession{}), "", now, policy.SessionIdleMinutes).
		Where(userIDWhereClause, userID).
		Count(&activeSessionCount).Error; err != nil {
		return 0, err
	}
	return activeSessionCount, nil
}

func (s *authSessionService) CreateSession(currentUser *user.SystemUser, roles []string, ip, userAgent string) (*common.TokenPair, error) {
	if s.auth.db == nil {
		return nil, common.ErrDatabaseNotInitialized
	}
	policy := s.auth.getAuthRuntimePolicy()
	now := time.Now()
	if err := s.governInventory(now, policy); err != nil {
		return nil, err
	}
	if err := authsession.CleanupUserOverflowSessions(s.auth.db, currentUser.ID, now, policy.SessionIdleMinutes, maxInt(policy.MaxActiveSessions-1, 0)); err != nil {
		return nil, err
	}

	session := SystemUserSession{
		SessionID:        uuid.NewString(),
		UserID:           currentUser.ID,
		RefreshJTI:       uuid.NewString(),
		RefreshExpiresAt: now.Add(common.RefreshTokenTTL),
		LastActivityAt:   &now,
		LastIP:           ip,
		UserAgent:        truncateString(userAgent, 255),
	}

	if err := s.auth.db.Create(&session).Error; err != nil {
		return nil, err
	}
	return s.auth.issueTokenPair(currentUser, roles, &session)
}

func (s *authSessionService) RefreshSession(claims *common.CustomClaims, ip, userAgent string) (*common.TokenPair, error) {
	if s.auth.db == nil {
		return nil, common.ErrDatabaseNotInitialized
	}

	var session SystemUserSession
	err := s.auth.db.Where(sessionIDAndUserIDWhereClause, claims.SessionID, claims.UserID).First(&session).Error
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
	if err := s.auth.db.First(&currentUser, claims.UserID).Error; err != nil {
		return nil, err
	}
	roles, err := s.auth.GetUserRoles(currentUser.ID)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	session.RefreshJTI = uuid.NewString()
	session.RefreshExpiresAt = now.Add(common.RefreshTokenTTL)
	session.LastRefreshAt = &now
	session.LastActivityAt = &now
	session.LastIP = ip
	session.UserAgent = truncateString(userAgent, 255)
	if err := s.auth.db.Save(&session).Error; err != nil {
		return nil, err
	}

	return s.auth.issueTokenPair(&currentUser, roles, &session)
}

func (s *authSessionService) RevokeSession(sessionID string) error {
	if s.auth.db == nil || sessionID == "" {
		return nil
	}

	now := time.Now()
	return s.auth.db.Model(&SystemUserSession{}).
		Where("session_id = ? AND revoked_at IS NULL", sessionID).
		Updates(map[string]interface{}{"revoked_at": &now}).Error
}

func (s *authSessionService) TouchSessionActivity(sessionID string, userID uint64, ip, userAgent string) error {
	if s.auth.db == nil || strings.TrimSpace(sessionID) == "" || userID == 0 {
		return nil
	}

	now := time.Now()
	clientIP := normalizeSessionClientIP(ip)
	agent := normalizeSessionUserAgent(userAgent)

	return s.auth.db.Exec(
		touchSessionActivitySQL,
		now,
		clientIP,
		clientIP,
		agent,
		agent,
		sessionID,
		userID,
		now.Add(-1*time.Minute),
	).Error
}

func (s *authSessionService) ListSessions(userID uint64, currentSessionID string) ([]SessionResp, error) {
	if s.auth.db == nil {
		return nil, common.ErrDatabaseNotInitialized
	}

	now := time.Now()
	policy := s.auth.getAuthRuntimePolicy()
	if err := s.governInventory(now, policy); err != nil {
		return nil, err
	}

	var sessions []SystemUserSession
	if err := authsession.ApplyActiveScope(s.auth.db, "", now, policy.SessionIdleMinutes).
		Where(userIDWhereClause, userID).
		Order("created_at desc").
		Find(&sessions).Error; err != nil {
		return nil, err
	}

	result := make([]SessionResp, 0, len(sessions))
	for _, item := range sessions {
		result = append(result, buildSessionResp(item, currentSessionID))
	}
	sort.SliceStable(result, func(i, j int) bool {
		if result[i].IsCurrent != result[j].IsCurrent {
			return result[i].IsCurrent
		}
		return result[i].CreatedAt > result[j].CreatedAt
	})
	return result, nil
}

func (s *authSessionService) RevokeOwnedSession(userID uint64, currentSessionID, targetSessionID string) error {
	if s.auth.db == nil {
		return common.ErrDatabaseNotInitialized
	}
	if strings.TrimSpace(targetSessionID) == "" {
		return errors.New(errSessionInvalid)
	}
	if targetSessionID == currentSessionID {
		return errors.New(errCurrentSessionRevokeForbidden)
	}

	var session SystemUserSession
	if err := s.auth.db.Where(sessionIDAndUserIDWhereClause, targetSessionID, userID).First(&session).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New(errSessionInvalid)
		}
		return err
	}
	if session.RevokedAt != nil {
		return nil
	}

	now := time.Now()
	return s.auth.db.Model(&SystemUserSession{}).
		Where(sessionIDAndActiveUserIDWhereClause, targetSessionID, userID).
		Updates(map[string]interface{}{"revoked_at": &now}).Error
}

func (s *authSessionService) CleanupHistoricSessions(retentionDays int, startedAt, endedAt string) (int64, error) {
	if s.auth.db == nil {
		return 0, common.ErrDatabaseNotInitialized
	}
	window, err := parseCleanupWindow(startedAt, endedAt, "auth.session.cleanup.range_invalid")
	if err != nil {
		return 0, err
	}

	now := time.Now()
	policy := s.auth.getAuthRuntimePolicy()
	if err := s.governInventory(now, policy); err != nil {
		return 0, err
	}

	db := s.auth.db.Table("system_user_session").Where("revoked_at IS NOT NULL")
	if window != nil {
		db = db.Where("revoked_at >= ? AND revoked_at <= ?", window.StartedAt, window.EndedAt)
	} else {
		if !s.auth.isAllowedSessionCleanupRetentionDays(retentionDays) {
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

func (s *authSessionService) BatchRevokeSessions(currentSessionID string, sessionIDs []string) (int64, error) {
	if s.auth.db == nil {
		return 0, common.ErrDatabaseNotInitialized
	}

	normalized := normalizeSessionIDs(sessionIDs)
	if len(normalized) == 0 {
		return 0, errors.New(errSessionInvalid)
	}
	for _, sessionID := range normalized {
		if sessionID == currentSessionID {
			return 0, errors.New(errCurrentSessionRevokeForbidden)
		}
	}

	now := time.Now()
	result := s.auth.db.Model(&SystemUserSession{}).
		Where("session_id IN ? AND revoked_at IS NULL", normalized).
		Updates(map[string]interface{}{"revoked_at": &now})
	if result.Error != nil {
		return 0, result.Error
	}
	return result.RowsAffected, nil
}

func (s *authSessionService) ListAllSessions(query *AdminSessionQuery) (*AdminSessionPageResp, error) {
	if s.auth.db == nil {
		return nil, common.ErrDatabaseNotInitialized
	}

	now := time.Now()
	policy := s.auth.getAuthRuntimePolicy()
	if err := s.governInventory(now, policy); err != nil {
		return nil, err
	}

	page, pageSize := normalizePageQuery(queryPageFromAdminSession(query), queryPageSizeFromAdminSession(query))
	db := s.auth.db.Table("system_user_session").
		Select("system_user_session.session_id, system_user_session.user_id, system_user.username, system_user.nickname, system_user_session.last_ip, system_user_session.user_agent, system_user_session.refresh_expires_at, system_user_session.last_refresh_at, system_user_session.last_activity_at, system_user_session.revoked_at, system_user_session.created_at").
		Joins("LEFT JOIN system_user ON system_user.id = system_user_session.user_id")
	db = applyAdminSessionFilters(db, query, now, policy)

	var rows []adminSessionRow
	if err := db.Order("system_user_session.created_at desc").Scan(&rows).Error; err != nil {
		return nil, err
	}

	items := make([]AdminSessionResp, 0, len(rows))
	var activeCount int64
	var revokedCount int64
	for _, row := range rows {
		clientInfo := parseClientInfo(row.UserAgent)
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

func (s *authSessionService) RevokeAnySession(currentSessionID, targetSessionID string) error {
	if s.auth.db == nil {
		return common.ErrDatabaseNotInitialized
	}
	if strings.TrimSpace(targetSessionID) == "" {
		return errors.New(errSessionInvalid)
	}
	if targetSessionID == currentSessionID {
		return errors.New(errCurrentSessionRevokeForbidden)
	}

	now := time.Now()
	return s.auth.db.Model(&SystemUserSession{}).
		Where("session_id = ? AND revoked_at IS NULL", targetSessionID).
		Updates(map[string]interface{}{"revoked_at": &now}).Error
}
