package security

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode"

	"pantheon-ops/backend/modules/auth/session"
	user "pantheon-ops/backend/modules/system/iam/user"
	"pantheon-ops/backend/pkg/authsession"
	"pantheon-ops/backend/pkg/authtoken"
	"pantheon-ops/backend/pkg/common"
	"pantheon-ops/backend/pkg/database"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// PolicyProvider abstracts runtime auth policy.
type PolicyProvider interface {
	GetAuthRuntimePolicy() AuthRuntimePolicy
}

// SessionRevoker allows password update to revoke other sessions.
type SessionRevoker interface {
	RevokeOtherSessions(userID uint64, currentSessionID string) error
}

// SessionCounter counts active sessions for a user.
type SessionCounter interface {
	CountActiveSessions(userID uint64, now time.Time) (int64, error)
}

// UserLoader loads a user by ID.
type UserLoader interface {
	LoadUserByID(userID uint64) (*user.SystemUser, error)
}

// SecurityEventRecorder persists security events.
type SecurityEventRecorder struct {
	db *gorm.DB
}

func NewSecurityEventRecorder(db *gorm.DB) *SecurityEventRecorder {
	return &SecurityEventRecorder{db: db}
}

func (r *SecurityEventRecorder) Record(event SystemAuthSecurityEvent) {
	if r.db == nil {
		return
	}
	if strings.TrimSpace(event.EventType) == "" || strings.TrimSpace(event.MessageKey) == "" {
		return
	}
	if strings.TrimSpace(event.Severity) == "" {
		event.Severity = "medium"
	}
	event.SourceKey = strings.TrimSpace(event.SourceKey)
	event.Username = strings.TrimSpace(event.Username)
	_ = r.db.Create(&event).Error
}

// AuthRuntimePolicy is the runtime security policy snapshot.
type AuthRuntimePolicy struct {
	PasswordMinLength       int
	PasswordRequireDigit    bool
	PasswordRequireUpper    bool
	PasswordHistoryLimit    int
	PasswordExpireDays      int
	MaxFailedAttempts       int
	LockMinutes             int
	SourceMaxFailedAttempts int
	SourceWindowMinutes     int
	SourceLockMinutes       int
	SessionIdleMinutes      int
	MaxActiveSessions       int
	SessionRetentionDays    int
	SecurityEventEnabled    bool
	CaptchaEnabled          bool
	MFAEnabled              bool
	SSOEnabled              bool
}

// Service handles password management and security event tracking.
type Service struct {
	db     *gorm.DB
	policy PolicyProvider
}

// NewService creates a SecurityService.
func NewService(db *gorm.DB, policy PolicyProvider) *Service {
	return &Service{db: db, policy: policy}
}

// VerifyPasswordForOperation checks the password and issues a short-lived operation token.
func (s *Service) VerifyPasswordForOperation(userID uint64, sessionID, password string) (string, error) {
	return s.VerifyPasswordForOperationWithContext(context.Background(), userID, sessionID, password)
}

func (s *Service) VerifyPasswordForOperationWithContext(ctx context.Context, userID uint64, sessionID, password string) (string, error) {
	if s.db == nil {
		return "", common.ErrDatabaseNotInitialized
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if strings.TrimSpace(sessionID) == "" {
		return "", errors.New("auth.operation.verification_mismatch")
	}
	var currentUser user.SystemUser
	if err := s.db.First(&currentUser, userID).Error; err != nil {
		return "", err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(currentUser.Password), []byte(password)); err != nil {
		return "", errors.New("auth.password.verify_failed")
	}
	token, err := authtoken.GenerateOperationTokenWithContext(ctx, userID, sessionID, authtoken.ScopeSecureAction, 5*time.Minute, database.RDB)
	if err != nil {
		return "", err
	}
	return token, nil
}

// UpdatePassword changes the user's password with policy enforcement.
func (s *Service) UpdatePassword(userID uint64, currentSessionID string, req *PasswordChangeReq) error {
	if s.db == nil {
		return common.ErrDatabaseNotInitialized
	}
	oldPassword := strings.TrimSpace(req.OldPassword)
	newPassword := strings.TrimSpace(req.NewPassword)
	policy := s.policy.GetAuthRuntimePolicy()

	if len(newPassword) < policy.PasswordMinLength {
		return errors.New("user.update.error.password_too_short")
	}
	if !passwordMatchesComplexity(newPassword, policy) {
		return errors.New("user.update.error.password_weak")
	}

	var currentUser user.SystemUser
	if err := s.db.First(&currentUser, userID).Error; err != nil {
		return err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(currentUser.Password), []byte(oldPassword)); err != nil {
		return errors.New("user.password.error.old_password_invalid")
	}
	if oldPassword == newPassword {
		return errors.New("user.password.error.same_as_old")
	}
	if err := s.ensurePasswordNotRecentlyUsed(userID, newPassword, currentUser.Password, policy); err != nil {
		return err
	}

	passwordHash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return s.persistPasswordUpdate(currentUser, userID, currentSessionID, string(passwordHash), policy.PasswordHistoryLimit > 0)
}

// ListSecurityEvents returns paginated security events.
func (s *Service) ListSecurityEvents(query *SecurityEventQuery) (*SecurityEventPageResp, error) {
	if s.db == nil {
		return nil, common.ErrDatabaseNotInitialized
	}
	page, pageSize := normalizeSecurityEventPageQuery(query)
	db := applySecurityEventFilters(s.db.Model(&SystemAuthSecurityEvent{}), query)

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}
	var events []SystemAuthSecurityEvent
	if err := db.Order("created_at desc, id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&events).Error; err != nil {
		return nil, err
	}
	return &SecurityEventPageResp{
		Items:    toSecurityEventRespList(events),
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

// AcknowledgeSecurityEvent marks a security event as acknowledged.
func (s *Service) AcknowledgeSecurityEvent(eventID, actorID uint64, actorUsername, note string) error {
	if s.db == nil {
		return common.ErrDatabaseNotInitialized
	}
	note = strings.TrimSpace(note)
	if note == "" {
		return errors.New("auth.security_event.acknowledge.note_required")
	}
	result := s.db.Model(&SystemAuthSecurityEvent{}).
		Where("id = ?", eventID).
		Updates(map[string]interface{}{
			"acknowledged_at":      time.Now(),
			"acknowledged_by":      actorID,
			"acknowledged_by_user": strings.TrimSpace(actorUsername),
			"acknowledgement_note": note,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// CountActiveSessions returns how many active sessions a user has.
func (s *Service) CountActiveSessions(userID uint64, now time.Time) (int64, error) {
	if s.db == nil {
		return 0, common.ErrDatabaseNotInitialized
	}
	policy := s.policy.GetAuthRuntimePolicy()
	var count int64
	err := authsession.ApplyActiveScope(s.db.Model(&session.SystemUserSession{}), "", now, policy.SessionIdleMinutes).
		Where("user_id = ?", userID).
		Count(&count).Error
	return count, err
}

// GetPasswordExpiresAt returns when the user's password expires (nil if never).
func (s *Service) GetPasswordExpiresAt(userID uint64) *string {
	policy := s.policy.GetAuthRuntimePolicy()
	if policy.PasswordExpireDays <= 0 {
		return nil
	}
	changedAt := s.passwordLastChangedAt(userID)
	if changedAt.IsZero() {
		return nil
	}
	expiresAt := changedAt.AddDate(0, 0, policy.PasswordExpireDays).Format(time.RFC3339)
	return &expiresAt
}

// IsPasswordExpired checks if the user's password has expired.
func (s *Service) IsPasswordExpired(userID uint64) bool {
	expiresAt := s.GetPasswordExpiresAt(userID)
	if expiresAt == nil {
		return false
	}
	parsed, err := time.Parse(time.RFC3339, *expiresAt)
	if err != nil {
		return false
	}
	return !parsed.After(time.Now())
}

// ListRecentSecurityEvents returns recent security events for a user.
func (s *Service) ListRecentSecurityEvents(userID uint64, limit int) []SecurityEventResp {
	if s.db == nil || userID == 0 || limit <= 0 {
		return []SecurityEventResp{}
	}
	var events []SystemAuthSecurityEvent
	if err := s.db.Where("user_id = ?", userID).Order("created_at desc, id desc").Limit(limit).Find(&events).Error; err != nil {
		return []SecurityEventResp{}
	}
	return toSecurityEventRespList(events)
}

// RevokeOtherSessionsForUser revokes all sessions for a user except the current one.
// This is called after a password change.
func (s *Service) RevokeOtherSessionsForUser(tx *gorm.DB, userID uint64, currentSessionID string) error {
	if strings.TrimSpace(currentSessionID) == "" {
		return nil
	}
	now := time.Now()
	return tx.Model(&session.SystemUserSession{}).
		Where("user_id = ? AND session_id <> ? AND revoked_at IS NULL", userID, currentSessionID).
		Updates(map[string]interface{}{"revoked_at": &now}).Error
}

func (s *Service) ensurePasswordNotRecentlyUsed(userID uint64, newPassword, currentPasswordHash string, policy AuthRuntimePolicy) error {
	if policy.PasswordHistoryLimit <= 0 {
		return nil
	}
	if bcrypt.CompareHashAndPassword([]byte(currentPasswordHash), []byte(newPassword)) == nil {
		return errors.New("user.password.error.reused")
	}
	var rows []SystemUserPasswordHistory
	if err := s.db.Where("user_id = ?", userID).
		Order("changed_at desc, id desc").
		Limit(policy.PasswordHistoryLimit).
		Find(&rows).Error; err != nil {
		return err
	}
	for _, row := range rows {
		if bcrypt.CompareHashAndPassword([]byte(row.PasswordHash), []byte(newPassword)) == nil {
			return errors.New("user.password.error.reused")
		}
	}
	return nil
}

func (s *Service) passwordLastChangedAt(userID uint64) time.Time {
	var row SystemUserPasswordHistory
	if err := s.db.Where("user_id = ?", userID).Order("changed_at desc, id desc").First(&row).Error; err == nil {
		return row.ChangedAt
	}
	var currentUser user.SystemUser
	if err := s.db.First(&currentUser, userID).Error; err == nil {
		if !currentUser.UpdatedAt.IsZero() {
			return currentUser.UpdatedAt
		}
		return currentUser.CreatedAt
	}
	return time.Time{}
}

func (s *Service) persistPasswordUpdate(currentUser user.SystemUser, userID uint64, currentSessionID, passwordHash string, keepHistory bool) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if keepHistory {
			if err := tx.Create(&SystemUserPasswordHistory{
				UserID:       currentUser.ID,
				PasswordHash: currentUser.Password,
				ChangedAt:    time.Now(),
			}).Error; err != nil {
				return err
			}
		}
		if err := tx.Model(&currentUser).Update("password", passwordHash).Error; err != nil {
			return err
		}
		return s.RevokeOtherSessionsForUser(tx, userID, currentSessionID)
	})
}

func applySecurityEventFilters(db *gorm.DB, query *SecurityEventQuery) *gorm.DB {
	if query == nil {
		return db
	}
	if strings.TrimSpace(query.Username) != "" {
		db = db.Where("username LIKE ?", "%"+common.EscapeLikePattern(strings.TrimSpace(query.Username))+"%")
	}
	if strings.TrimSpace(query.EventType) != "" {
		db = db.Where("event_type = ?", strings.TrimSpace(query.EventType))
	}
	if strings.TrimSpace(query.Severity) != "" {
		db = db.Where("severity = ?", strings.TrimSpace(query.Severity))
	}
	if query.Acknowledged == nil {
		return db
	}
	if *query.Acknowledged {
		return db.Where("acknowledged_at IS NOT NULL")
	}
	return db.Where("acknowledged_at IS NULL")
}

func passwordMatchesComplexity(password string, policy AuthRuntimePolicy) bool {
	if !policy.PasswordRequireDigit && !policy.PasswordRequireUpper {
		return true
	}
	hasDigit := false
	hasUpper := false
	for _, r := range password {
		if unicode.IsDigit(r) {
			hasDigit = true
		}
		if unicode.IsUpper(r) {
			hasUpper = true
		}
	}
	if policy.PasswordRequireDigit && !hasDigit {
		return false
	}
	if policy.PasswordRequireUpper && !hasUpper {
		return false
	}
	return true
}

func toSecurityEventRespList(events []SystemAuthSecurityEvent) []SecurityEventResp {
	result := make([]SecurityEventResp, 0, len(events))
	for _, item := range events {
		result = append(result, SecurityEventResp{
			ID:                  item.ID,
			UserID:              item.UserID,
			Username:            item.Username,
			EventType:           item.EventType,
			Severity:            item.Severity,
			SourceKey:           item.SourceKey,
			IP:                  item.IP,
			UserAgent:           item.UserAgent,
			MessageKey:          item.MessageKey,
			Metadata:            item.Metadata,
			AcknowledgedAt:      formatNullableTime(item.AcknowledgedAt),
			AcknowledgedBy:      item.AcknowledgedBy,
			AcknowledgedByUser:  item.AcknowledgedByUser,
			AcknowledgementNote: item.AcknowledgementNote,
			CreatedAt:           item.CreatedAt.Format(time.RFC3339),
		})
	}
	return result
}

func formatNullableTime(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.Format(time.RFC3339)
	return &formatted
}

func normalizeSecurityEventPageQuery(query *SecurityEventQuery) (int, int) {
	if query == nil {
		return 1, 10
	}
	page := query.Page
	pageSize := query.PageSize
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

// PasswordChangeReq is the password update request DTO.
type PasswordChangeReq struct {
	OldPassword string `json:"oldPassword" binding:"required"`
	NewPassword string `json:"newPassword" binding:"required"`
}

// SecurityEventQuery mirrors the auth-layer DTO.
type SecurityEventQuery struct {
	Username     string `form:"username" json:"username"`
	EventType    string `form:"eventType" json:"eventType"`
	Severity     string `form:"severity" json:"severity"`
	Acknowledged *bool  `form:"acknowledged" json:"acknowledged"`
	Page         int    `form:"page" json:"page"`
	PageSize     int    `form:"pageSize" json:"pageSize"`
}

// SecurityEventResp mirrors the auth-layer DTO.
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

// SecurityEventPageResp mirrors the auth-layer DTO.
type SecurityEventPageResp struct {
	Items    []SecurityEventResp `json:"items"`
	Total    int64               `json:"total"`
	Page     int                 `json:"page"`
	PageSize int                 `json:"pageSize"`
}
