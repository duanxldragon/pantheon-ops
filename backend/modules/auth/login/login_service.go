package login

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"pantheon-ops/backend/modules/auth/security"
	user "pantheon-ops/backend/modules/system/iam/user"
	"pantheon-ops/backend/pkg/common"
	"pantheon-ops/backend/pkg/impexp"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// PolicyProvider abstracts runtime auth policy so LoginService stays decoupled from AuthService.
type PolicyProvider interface {
	GetRuntimePolicy() RuntimePolicy
}

// SecurityEventRecorder allows LoginService to emit security events without importing auth.
type SecurityEventRecorder interface {
	RecordSecurityEvent(event security.SystemAuthSecurityEvent)
}

// LoginService handles credential authentication and login throttling.
type LoginService struct {
	db       *gorm.DB
	policy   PolicyProvider
	recorder SecurityEventRecorder
}

// NewLoginService creates a LoginService with the given DB and policy provider.
func NewLoginService(db *gorm.DB, policy PolicyProvider, recorder SecurityEventRecorder) *LoginService {
	return &LoginService{db: db, policy: policy, recorder: recorder}
}

// Authenticate verifies username/password and returns the user if valid.
func (s *LoginService) Authenticate(req *LoginReq) (*user.SystemUser, error) {
	return s.AuthenticateWithSource(req, "")
}

// AuthenticateWithSource verifies credentials with source/IP throttling.
func (s *LoginService) AuthenticateWithSource(req *LoginReq, sourceKey string) (*user.SystemUser, error) {
	if s.db == nil {
		return nil, common.ErrDatabaseNotInitialized
	}
	policy := s.policy.GetRuntimePolicy()
	now := time.Now()

	if err := s.ensureSourceThrottleAllowed(sourceKey, policy, now); err != nil {
		return nil, err
	}

	currentUser, err := s.loadLoginUser(strings.TrimSpace(req.Username), sourceKey, policy, now)
	if err != nil {
		return nil, err
	}
	if err := s.ensureUserAvailable(currentUser, sourceKey, policy, now); err != nil {
		return nil, err
	}

	if err := bcrypt.CompareHashAndPassword([]byte(currentUser.Password), []byte(req.Password)); err != nil {
		return nil, s.handlePasswordMismatch(currentUser, sourceKey, policy, now)
	}
	if err := s.clearFailedLoginState(currentUser.ID); err != nil {
		return nil, err
	}
	return currentUser, nil
}

func (s *LoginService) ensureSourceThrottleAllowed(sourceKey string, policy RuntimePolicy, now time.Time) error {
	blocked, err := s.checkSourceThrottle(sourceKey, policy, now)
	if err != nil {
		return err
	}
	if blocked {
		return errors.New(errLoginSourceBlocked)
	}
	return nil
}

func (s *LoginService) loadLoginUser(username, sourceKey string, policy RuntimePolicy, now time.Time) (*user.SystemUser, error) {
	if username == "" {
		_ = s.failLoginSourceBlocked(nil, sourceKey, policy, now)
		return nil, errors.New(errUserNotFound)
	}
	var currentUser user.SystemUser
	result := s.db.Where("username = ?", username).First(&currentUser)
	if result.Error == nil {
		return &currentUser, nil
	}
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		if err := s.failLoginSourceBlocked(nil, sourceKey, policy, now); err != nil {
			return nil, err
		}
		return nil, errors.New(errUserNotFound)
	}
	return nil, result.Error
}

func (s *LoginService) ensureUserAvailable(currentUser *user.SystemUser, sourceKey string, policy RuntimePolicy, now time.Time) error {
	if currentUser.Status == common.StatusDisabled {
		if err := s.failLoginSourceBlocked(currentUser, sourceKey, policy, now); err != nil {
			return err
		}
		return errors.New(errUserDisabled)
	}
	if currentUser.LoginLockedUntil != nil && currentUser.LoginLockedUntil.After(now) {
		if err := s.failLoginSourceBlocked(currentUser, sourceKey, policy, now); err != nil {
			return err
		}
		return errors.New(errUserLocked)
	}
	return nil
}

func (s *LoginService) handlePasswordMismatch(currentUser *user.SystemUser, sourceKey string, policy RuntimePolicy, now time.Time) error {
	locked, err := s.recordFailedLoginAttempt(currentUser, policy)
	if err != nil {
		return err
	}
	blocked, err := s.recordSourceFailure(sourceKey, policy, now)
	if err != nil {
		return err
	}
	if blocked {
		s.emitSecurityEvent(currentUser, "source_blocked", "high", sourceKey, msgSecurityEventSourceBlocked, "")
		return errors.New(errLoginSourceBlocked)
	}
	if locked {
		s.emitSecurityEvent(currentUser, "account_locked", "high", sourceKey, msgSecurityEventAccountLocked, "")
		return errors.New(errUserLocked)
	}
	s.emitSecurityEvent(currentUser, "password_wrong", "medium", sourceKey, msgSecurityEventPasswordWrong, loginSourceIP(sourceKey))
	return errors.New(errPasswordWrong)
}

func (s *LoginService) clearFailedLoginState(userID uint64) error {
	if s.db == nil {
		return common.ErrDatabaseNotInitialized
	}
	return s.db.Model(&user.SystemUser{}).
		Where("id = ? AND (failed_login_attempts <> 0 OR login_locked_until IS NOT NULL)", userID).
		Updates(map[string]any{
			"failed_login_attempts": 0,
			"login_locked_until":    nil,
		}).Error
}

// ListOwnLoginLogs returns login logs for a specific username.
func (s *LoginService) ListOwnLoginLogs(username string, query *LoginLogQuery) (*LoginLogPageResp, error) {
	if strings.TrimSpace(username) == "" {
		return nil, errors.New(errTokenInvalid)
	}
	return s.listLoginLogs(query, username)
}

// ListLoginLogs returns all login logs.
func (s *LoginService) ListLoginLogs(query *LoginLogQuery) (*LoginLogPageResp, error) {
	s.ensureAutomaticLoginLogRetention()
	return s.listLoginLogs(query, "")
}

func (s *LoginService) listLoginLogs(query *LoginLogQuery, filterUsername string) (*LoginLogPageResp, error) {
	if s.db == nil {
		return nil, common.ErrDatabaseNotInitialized
	}

	var logs []SystemLogLogin
	db := s.db.Model(&SystemLogLogin{})
	page, pageSize := normalizePageQuery(queryPage(query), queryPageSize(query))

	if filterUsername != "" {
		db = db.Where(usernameLikeWhereClause, "%"+filterUsername+"%")
	} else if query != nil && strings.TrimSpace(query.Username) != "" {
		db = db.Where(usernameLikeWhereClause, "%"+strings.TrimSpace(query.Username)+"%")
	}
	if query != nil && query.Status != nil && common.IsLoginStatus(*query.Status) {
		db = db.Where("status = ?", *query.Status)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}
	if err := db.Order(loginTimeDescOrderClause).Offset((page - 1) * pageSize).Limit(pageSize).Find(&logs).Error; err != nil {
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
	return &LoginLogPageResp{Items: items, Total: total, Page: page, PageSize: pageSize}, nil
}

// ExportLoginLogs exports login logs as CSV.
func (s *LoginService) ExportLoginLogs(query *LoginLogQuery) (*impexp.CSVFile, error) {
	if s.db == nil {
		return nil, common.ErrDatabaseNotInitialized
	}
	s.ensureAutomaticLoginLogRetention()

	logs, err := s.listLoginLogsForExport(query)
	if err != nil {
		return nil, err
	}

	rows := make([][]string, 0, len(logs))
	for _, item := range logs {
		rows = append(rows, []string{
			item.Username, item.Ipaddr, item.LoginLocation, item.Browser, item.Os,
			fmt.Sprintf("%d", item.Status), item.Msg, item.LoginTime.Format(time.RFC3339),
		})
	}
	return &impexp.CSVFile{
		Filename: "system-login-log-export.csv",
		Headers:  []string{"username", "ipaddr", "loginLocation", "browser", "os", "status", "msg", "loginTime"},
		Rows:     rows,
	}, nil
}

// CleanupLoginLogs removes old login log entries.
func (s *LoginService) CleanupLoginLogs(retentionDays int, startedAt, endedAt string) (int64, error) {
	if s.db == nil {
		return 0, common.ErrDatabaseNotInitialized
	}
	window, err := parseCleanupWindow(startedAt, endedAt, errLoginLogCleanupRangeInvalid)
	if err != nil {
		return 0, err
	}
	db := s.db.Model(&SystemLogLogin{})
	if window != nil {
		db = db.Where("login_time >= ? AND login_time <= ?", window.StartedAt, window.EndedAt)
	} else {
		if !s.isAllowedLoginLogRetentionDays(retentionDays) {
			return 0, errors.New(errLoginLogCleanupDaysInvalid)
		}
		cutoff := time.Now().AddDate(0, 0, -retentionDays)
		db = db.Where("login_time < ?", cutoff)
	}
	result := db.Delete(&SystemLogLogin{})
	return result.RowsAffected, result.Error
}

// BatchDeleteLoginLogs deletes login logs by IDs.
func (s *LoginService) BatchDeleteLoginLogs(ids []uint64) (int64, error) {
	if s.db == nil {
		return 0, common.ErrDatabaseNotInitialized
	}
	normalized := normalizeUint64IDs(ids)
	if len(normalized) == 0 {
		return 0, errors.New(errLoginLogDeleteIdsRequired)
	}
	result := s.db.Where("id IN ?", normalized).Delete(&SystemLogLogin{})
	return result.RowsAffected, result.Error
}

func (s *LoginService) listLoginLogsForExport(query *LoginLogQuery) ([]SystemLogLogin, error) {
	if s.db == nil {
		return nil, common.ErrDatabaseNotInitialized
	}
	s.ensureAutomaticLoginLogRetention()

	var logs []SystemLogLogin
	db := s.db.Model(&SystemLogLogin{})
	if query != nil && strings.TrimSpace(query.Username) != "" {
		db = db.Where(usernameLikeWhereClause, "%"+strings.TrimSpace(query.Username)+"%")
	}
	if query != nil && query.Status != nil && common.IsLoginStatus(*query.Status) {
		db = db.Where("status = ?", *query.Status)
	}
	return logs, db.Order(loginTimeDescOrderClause).Limit(maxLoginLogExportRows).Find(&logs).Error
}

// RecordLoginLog writes a login log entry.
func (s *LoginService) RecordLoginLog(requestID, username, ip, browser, os string, status int, msg string) {
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

func (s *LoginService) ensureAutomaticLoginLogRetention() {
	if s.db == nil {
		return
	}
	now := time.Now()
	policy := s.policy.GetRuntimePolicy()
	cutoff := now.AddDate(0, 0, -maxInt(policy.LoginLogRetentionDays, 1))
	_ = s.db.Where("login_time < ?", cutoff).Delete(&SystemLogLogin{}).Error
}

func (s *LoginService) recordFailedLoginAttempt(currentUser *user.SystemUser, policy RuntimePolicy) (bool, error) {
	if s.db == nil || currentUser == nil {
		return false, common.ErrDatabaseNotInitialized
	}
	nextAttempts := currentUser.FailedLoginAttempts + 1
	updates := map[string]any{"failed_login_attempts": nextAttempts}
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

func (s *LoginService) checkSourceThrottle(sourceKey string, policy RuntimePolicy, now time.Time) (bool, error) {
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
		_ = s.db.Model(&throttle).Updates(map[string]any{
			"failure_count":     0,
			"window_started_at": nil,
			"blocked_until":     nil,
		}).Error
	}
	return false, nil
}

func (s *LoginService) failLoginSourceBlocked(currentUser *user.SystemUser, sourceKey string, policy RuntimePolicy, now time.Time) error {
	blocked, err := s.recordSourceFailure(sourceKey, policy, now)
	if err != nil {
		return err
	}
	if blocked {
		s.emitSecurityEvent(currentUser, "source_blocked", "high", sourceKey, msgSecurityEventSourceBlocked, "")
		return errors.New(errLoginSourceBlocked)
	}
	return nil
}

func (s *LoginService) recordSourceFailure(sourceKey string, policy RuntimePolicy, now time.Time) (bool, error) {
	normalizedKey := strings.TrimSpace(sourceKey)
	if normalizedKey == "" || policy.SourceMaxFailedAttempts <= 0 {
		return false, nil
	}
	var throttle SystemLoginThrottle
	err := s.db.Where("source_key = ?", normalizedKey).First(&throttle).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return s.createSourceThrottleFailure(normalizedKey, policy, now)
	}
	if err != nil {
		return false, err
	}
	if sourceThrottleBlocked(throttle.BlockedUntil, now) {
		return true, nil
	}
	return s.updateSourceThrottleFailure(&throttle, policy, now)
}

func (s *LoginService) createSourceThrottleFailure(sourceKey string, policy RuntimePolicy, now time.Time) (bool, error) {
	throttle := SystemLoginThrottle{
		SourceKey:       sourceKey,
		FailureCount:    1,
		WindowStartedAt: &now,
		LastAttemptAt:   &now,
		BlockedUntil:    sourceThrottleBlockedUntil(policy, now, policy.SourceMaxFailedAttempts <= 1),
	}
	if err := s.db.Create(&throttle).Error; err != nil {
		return false, err
	}
	return sourceThrottleBlocked(throttle.BlockedUntil, now), nil
}

func (s *LoginService) updateSourceThrottleFailure(throttle *SystemLoginThrottle, policy RuntimePolicy, now time.Time) (bool, error) {
	windowStartedAt := throttle.WindowStartedAt
	if s.isSourceThrottleWindowExpired(windowStartedAt, policy, now) || windowStartedAt == nil {
		windowStartedAt = &now
		throttle.FailureCount, throttle.BlockedUntil = 0, nil
	}
	throttle.FailureCount++
	throttle.WindowStartedAt = windowStartedAt
	throttle.LastAttemptAt = &now
	if throttle.FailureCount >= policy.SourceMaxFailedAttempts {
		throttle.BlockedUntil = sourceThrottleBlockedUntil(policy, now, true)
	}
	_ = s.db.Model(&throttle).Updates(map[string]any{
		"failure_count":     throttle.FailureCount,
		"window_started_at": throttle.WindowStartedAt,
		"last_attempt_at":   throttle.LastAttemptAt,
		"blocked_until":     throttle.BlockedUntil,
	}).Error
	return sourceThrottleBlocked(throttle.BlockedUntil, now), nil
}

func (s *LoginService) isSourceThrottleWindowExpired(windowStartedAt *time.Time, policy RuntimePolicy, now time.Time) bool {
	if windowStartedAt == nil {
		return true
	}
	windowMinutes := maxInt(policy.SourceWindowMinutes, 1)
	return windowStartedAt.Add(time.Duration(windowMinutes) * time.Minute).Before(now)
}

func (s *LoginService) emitSecurityEvent(currentUser *user.SystemUser, eventType, severity, sourceKey, messageKey, ip string) {
	if s.recorder == nil || !s.policy.GetRuntimePolicy().SecurityEventEnabled {
		return
	}
	username := ""
	userID := uint64(0)
	if currentUser != nil {
		userID = currentUser.ID
		username = currentUser.Username
	}
	s.recorder.RecordSecurityEvent(security.SystemAuthSecurityEvent{
		UserID:     userID,
		Username:   username,
		EventType:  eventType,
		Severity:   severity,
		SourceKey:  sourceKey,
		IP:         ip,
		MessageKey: messageKey,
	})
}

func (s *LoginService) isAllowedLoginLogRetentionDays(retentionDays int) bool {
	for _, allowed := range s.policy.GetRuntimePolicy().LoginLogCleanupRetentionDays {
		if allowed == retentionDays {
			return true
		}
	}
	return false
}

// RuntimePolicy mirrors the subset of auth policy needed by LoginService.
type RuntimePolicy struct {
	MaxFailedAttempts            int
	LockMinutes                  int
	SourceMaxFailedAttempts      int
	SourceWindowMinutes          int
	SourceLockMinutes            int
	SecurityEventEnabled         bool
	LoginLogRetentionDays        int
	LoginLogCleanupRetentionDays []int
}

const (
	errLoginSourceBlocked          = "auth.login.error.source_blocked"
	errUserNotFound                = "user.login.error.not_found"
	errUserDisabled                = "user.login.error.disabled"
	errUserLocked                  = "user.login.error.locked"
	errPasswordWrong               = "user.login.error.password_wrong"
	errTokenInvalid                = "token.invalid"
	errLoginLogCleanupRangeInvalid = "auth.login_log.cleanup.range_invalid"
	errLoginLogCleanupDaysInvalid  = "auth.login_log.cleanup.days_invalid"
	errLoginLogDeleteIdsRequired   = "auth.login_log.delete.ids_required"
	msgSecurityEventPasswordWrong  = "auth.security.event.password_wrong"
	msgSecurityEventSourceBlocked  = "auth.security.event.source_blocked"
	msgSecurityEventAccountLocked  = "auth.security.event.account_locked"

	usernameLikeWhereClause  = "username LIKE ?"
	loginTimeDescOrderClause = "login_time desc, id desc"
	maxLoginLogExportRows    = 10000
)

func loginSourceIP(sourceKey string) string {
	trimmed := strings.TrimSpace(sourceKey)
	if strings.HasPrefix(trimmed, "ip:") {
		return strings.TrimSpace(strings.TrimPrefix(trimmed, "ip:"))
	}
	return ""
}

func sourceThrottleBlocked(blockedUntil *time.Time, now time.Time) bool {
	return blockedUntil != nil && blockedUntil.After(now)
}

func sourceThrottleBlockedUntil(policy RuntimePolicy, now time.Time, shouldBlock bool) *time.Time {
	if !shouldBlock {
		return nil
	}
	blockedUntil := now.Add(time.Duration(maxInt(policy.SourceLockMinutes, 1)) * time.Minute)
	return &blockedUntil
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

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
