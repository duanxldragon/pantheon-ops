package session

import (
	"context"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/pkg/authsession"
	"github.com/duanxldragon/pantheon-base/backend/pkg/authtoken"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/database"
	"github.com/duanxldragon/pantheon-base/backend/pkg/logging"
	"github.com/duanxldragon/pantheon-base/backend/pkg/maintenance"
	"github.com/duanxldragon/pantheon-base/backend/pkg/tenant"

	"github.com/google/uuid"
	"go.uber.org/zap"
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

// SessionInventoryGovernanceTaskName is the maintenance registry key for the
// periodic session inventory sweep (idle-session revocation + retention
// purge). It is the only automatic caller of this work.
const SessionInventoryGovernanceTaskName = "auth.session_inventory"

// RunSessionInventoryGovernance revokes idle sessions and purges sessions past
// the configured retention window.
//
// It is a maintenance entry point: callers are the background maintenance
// runner and explicitly-triggered maintenance, never list/read handlers
// (task 2026-09-22-request-path-maintenance). Failures are returned so the
// runner can log and count them instead of them hiding inside a read request.
func (s *Service) RunSessionInventoryGovernance() error {
	if s.db == nil {
		return common.ErrDatabaseNotInitialized
	}
	now := time.Now()
	policy := s.policy.GetSessionPolicy()
	if err := authsession.CleanupInactiveSessions(s.db, now, policy.SessionIdleMinutes); err != nil {
		return err
	}
	return authsession.PurgeHistoricSessions(s.db, now, policy.SessionRetentionDays)
}

// RegisterMaintenanceTasks registers the session inventory sweep with the
// background maintenance registry.
func (s *Service) RegisterMaintenanceTasks(reg *maintenance.Registry) {
	reg.Register(maintenance.Task{
		Name:     SessionInventoryGovernanceTaskName,
		Interval: maintenance.DefaultInterval,
		Run: func(context.Context) error {
			return s.RunSessionInventoryGovernance()
		},
	})
}

// RefreshSession refreshes an active session and issues a new token pair.
func (s *Service) RefreshSession(sessionID string, userID uint64, ip, userAgent string) (*authtoken.Pair, error) {
	return s.RefreshSessionWithContext(context.Background(), sessionID, userID, ip, userAgent)
}

// RefreshSessionWithContext refreshes an active session and issues a new token
// pair, propagating ctx to the underlying database and token operations.
func (s *Service) RefreshSessionWithContext(ctx context.Context, sessionID string, userID uint64, ip, userAgent string) (*authtoken.Pair, error) {
	if s.db == nil {
		return nil, common.ErrDatabaseNotInitialized
	}
	if ctx == nil {
		ctx = context.Background()
	}
	db := s.db.WithContext(ctx)
	var sess SystemUserSession
	if err := db.Where(sessionIDAndUserIDWhereClause, sessionID, userID).First(&sess).Error; err != nil {
		return nil, err
	}
	if sess.RevokedAt != nil || sess.RefreshExpiresAt.Before(time.Now()) {
		return nil, common.ErrUnauthorized
	}

	// Tenant refresh gate (contract §2.2/§5, task packet risk node "stale
	// membership"): a session must not outlive its membership. Compat mode
	// and claim-less sessions pass unchanged (flag-off regression guarantee).
	if err := tenant.GateSessionRefresh(db, tenant.RefreshCheckInput{
		Mode:     tenant.NormalizeMode(tenant.FeatureFlagSettingKeyReader(db)),
		UserID:   sess.UserID,
		TenantID: sess.TenantID,
	}); err != nil {
		return nil, err
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
	if err := db.Save(&sess).Error; err != nil {
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
	if err := s.db.Model(&SystemUserSession{}).
		Where("session_id = ? AND revoked_at IS NULL", sessionID).
		Updates(map[string]interface{}{"revoked_at": &now}).Error; err != nil {
		return err
	}
	return RevokeSessionArtifacts(sessionID)
}

// RevokeSessionArtifacts 使会话在 Redis 侧的三条失效路径同时生效：access
// token 黑名单（每请求校验，立即阻断存量 access token）、refresh token 级联
// 删除、以及 middleware 本地会话缓存无法感知的失效窗口。DB revoked_at 已由
// 调用方写入；这里任一写失败都必须返回错误——access 路径没有 DB 兜底，
// 静默失败会形成“已撤销仍可用”的安全假象。
func RevokeSessionArtifacts(sessionID string) error {
	if err := authtoken.BlacklistSession(context.Background(), database.RDB, sessionID); err != nil {
		logging.Warn("blacklist session access token failed",
			zap.String("session_id", sessionID), zap.Error(err))
		return err
	}
	if err := authtoken.RevokeSessionRefresh(context.Background(), database.RDB, sessionID); err != nil {
		logging.Warn("cascade revoke session refresh token failed",
			zap.String("session_id", sessionID), zap.Error(err))
		return err
	}
	return nil
}

// CascadeRevokeSessionRefresh 级联删除会话绑定的 refresh token（Redis）。
// 失败仅记日志不回滚：DB 侧 revoked_at 已生效，refresh 路径仍会被 session 状态校验拦截。
// 仅用于已删除会话行等无法逐会话返回错误的批量场景；显式撤销路径请使用
// RevokeSessionArtifacts 以获得 fail-fast 语义。
func CascadeRevokeSessionRefresh(sessionIDs ...string) {
	for _, sid := range sessionIDs {
		if strings.TrimSpace(sid) == "" {
			continue
		}
		if err := authtoken.RevokeSessionRefresh(context.Background(), database.RDB, sid); err != nil {
			logging.Warn("cascade revoke session refresh token failed",
				zap.String("session_id", sid), zap.Error(err))
		}
	}
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
	// Read path: no inventory purge here. Idle/expired sessions are swept by
	// the registered maintenance task, not while serving this list.
	now := time.Now()
	policy := s.policy.GetSessionPolicy()

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
	if err := s.db.Model(&SystemUserSession{}).
		Where(sessionIDAndActiveUserIDWhereClause, targetSessionID, userID).
		Updates(map[string]interface{}{"revoked_at": &now}).Error; err != nil {
		return err
	}
	return RevokeSessionArtifacts(targetSessionID)
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
	// Cap batch input so a single request cannot smuggle an arbitrarily
	// large IN clause past BodySizeLimit.
	if len(normalized) > 500 {
		return 0, errors.New("param.invalid")
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
	if result.Error != nil {
		return result.RowsAffected, result.Error
	}
	for _, sid := range normalized {
		if err := RevokeSessionArtifacts(sid); err != nil {
			return result.RowsAffected, err
		}
	}
	return result.RowsAffected, nil
}

// ListAllSessions returns paginated session records for admin use.
// Filters (including browser/OS/device, derived from user_agent) are pushed
// down to SQL; COUNT and LIMIT/OFFSET run in the database — no full-scan
// into memory pagination (task 2026-09-22-export-and-session-pagination).
func (s *Service) ListAllSessions(query *AdminSessionQuery) (*AdminSessionPageResp, error) {
	if s.db == nil {
		return nil, common.ErrDatabaseNotInitialized
	}
	// Read path: the session inventory sweep is a registered maintenance task
	// (RunSessionInventoryGovernance), deliberately not part of this query.
	now := time.Now()
	policy := s.policy.GetSessionPolicy()

	page, pageSize := normalizePageQuery(queryPageFromAdminSession(query), queryPageSizeFromAdminSession(query))

	base := func() *gorm.DB {
		db := s.db.Table("system_user_session").
			Joins("LEFT JOIN system_user ON system_user.id = system_user_session.user_id")
		return applyAdminSessionFilters(db, query, now, policy)
	}
	// Browser/OS/device filters become user_agent LIKE conditions so the
	// whole filtered set lives in SQL and counts match pages exactly.
	db := applyAdminSessionClientFilters(base(), query)

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}

	// Whole-filtered-set aggregates so active/revoked counts match the
	// paginated result exactly (both derive from the same SQL filter).
	var activeCount, revokedCount int64
	counts := struct {
		ActiveCount  *int64
		RevokedCount *int64
	}{}
	if err := base().Session(&gorm.Session{}).
		Select("SUM(CASE WHEN system_user_session.revoked_at IS NULL THEN 1 ELSE 0 END) AS active_count, " +
			"SUM(CASE WHEN system_user_session.revoked_at IS NOT NULL THEN 1 ELSE 0 END) AS revoked_count").
		Scan(&counts).Error; err != nil {
		return nil, err
	}
	if counts.ActiveCount != nil {
		activeCount = *counts.ActiveCount
	}
	if counts.RevokedCount != nil {
		revokedCount = *counts.RevokedCount
	}

	var rows []adminSessionRow
	if err := base().Session(&gorm.Session{}).
		Select("system_user_session.session_id, system_user_session.user_id, system_user.username, system_user.nickname, system_user_session.last_ip, system_user_session.user_agent, system_user_session.refresh_expires_at, system_user_session.last_refresh_at, system_user_session.last_activity_at, system_user_session.revoked_at, system_user_session.created_at").
		Order("system_user_session.created_at desc").
		Offset((page - 1) * pageSize).Limit(pageSize).
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	items := make([]AdminSessionResp, 0, len(rows))
	for _, row := range rows {
		items = append(items, buildAdminSessionResp(row, ParseClientInfo(row.UserAgent)))
	}

	return &AdminSessionPageResp{
		Items:        items,
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
	if err := s.db.Model(&SystemUserSession{}).
		Where("session_id = ? AND revoked_at IS NULL", targetSessionID).
		Updates(map[string]interface{}{"revoked_at": &now}).Error; err != nil {
		return err
	}
	// 与 RevokeSession/RevokeOwnedSession 保持同一失效语义：DB revoked_at、
	// refresh token 删除、access token 黑名单三路同时生效，管理员撤销后
	// 存量 access token 不允许继续使用到 TTL 结束。
	return RevokeSessionArtifacts(targetSessionID)
}
