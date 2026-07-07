package mfa

import (
	"context"
	"errors"
	"strings"
	"time"

	"pantheon-ops/backend/pkg/authtoken"
	"pantheon-ops/backend/pkg/common"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PolicyProvider abstracts runtime auth policy.
type PolicyProvider interface {
	IsMFAEnabled() bool
}

// IdentityProvider abstracts user identity lookups needed after MFA verification.
type IdentityProvider interface {
	LoadUserByID(userID uint64) (*UserRecord, error)
	GetUserRoles(userID uint64) ([]string, error)
}

// SessionCreator abstracts session creation after MFA verification.
type SessionCreator interface {
	CreateSessionWithContext(ctx context.Context, userID uint64, roles []string, ip, userAgent string) (*authtoken.Pair, error)
}

// MFAVerifyResult is the result of a successful MFA challenge verification.
// It contains everything needed to build the HTTP response.
type MFAVerifyResult struct {
	UserID    uint64
	TokenPair *authtoken.Pair
	Username  string
	Nickname  string
	Avatar    string
	Email     string
	Phone     string
	Roles     []string
	Perms     []string
}

// MFAVerifyReq mirrors the auth-layer DTO so MFA service stays decoupled.
type MFAVerifyReq struct {
	ChallengeID string `json:"challengeId" binding:"required"`
	Code        string `json:"code" binding:"required"`
}

// MFAChallengeResp is the response when MFA is required before session creation.
type MFAChallengeResp struct {
	MFARequired      bool   `json:"mfaRequired"`
	ChallengeID      string `json:"challengeId"`
	SetupRequired    bool   `json:"setupRequired"`
	TOTPSecret       string `json:"totpSecret,omitempty"`
	TOTPProvisionURI string `json:"totpProvisionUri,omitempty"`
	ExpiresAt        string `json:"expiresAt"`
}

// Service handles TOTP MFA challenge lifecycle.
type Service struct {
	db       *gorm.DB
	policy   PolicyProvider
	creator  SessionCreator
	identity IdentityProvider
}

// NewService creates an MFA service.
func NewService(db *gorm.DB, policy PolicyProvider, creator SessionCreator, identity IdentityProvider) *Service {
	return &Service{db: db, policy: policy, creator: creator, identity: identity}
}

// CreateChallenge initiates an MFA challenge for a user.
func (s *Service) CreateChallenge(currentUser *UserRecord) (*MFAChallengeResp, error) {
	if s.db == nil {
		return nil, common.ErrDatabaseNotInitialized
	}
	if currentUser == nil || currentUser.ID == 0 {
		return nil, errors.New("auth.mfa.user_invalid")
	}
	if !s.policy.IsMFAEnabled() {
		return nil, errors.New("auth.mfa.disabled")
	}

	var factor SystemAuthFactor
	err := s.db.Where(userIDAndFactorTypeEnabledWhereClause, currentUser.ID, "totp", 1).First(&factor).Error
	setupRequired := errors.Is(err, gorm.ErrRecordNotFound)
	if err != nil && !setupRequired {
		return nil, err
	}

	secret := ""
	if setupRequired {
		var secretErr error
		secret, secretErr = GenerateTOTPSecret()
		if secretErr != nil {
			return nil, secretErr
		}
	}
	encryptedSecret, err := EncryptMFASecret(secret)
	if err != nil {
		return nil, err
	}

	expiresAt := time.Now().Add(5 * time.Minute)
	challenge := SystemAuthMFAChallenge{
		ChallengeID:     uuid.NewString(),
		UserID:          currentUser.ID,
		Purpose:         "login",
		SecretEncrypted: encryptedSecret,
		SetupRequired:   boolToInt(setupRequired),
		ExpiresAt:       expiresAt,
	}
	if err := s.db.Create(&challenge).Error; err != nil {
		return nil, err
	}

	resp := &MFAChallengeResp{
		MFARequired:   true,
		ChallengeID:   challenge.ChallengeID,
		SetupRequired: setupRequired,
		ExpiresAt:     expiresAt.Format(time.RFC3339),
	}
	if setupRequired {
		resp.TOTPSecret = secret
		resp.TOTPProvisionURI = BuildTOTPURL(currentUser.Username, secret)
	}
	return resp, nil
}

// VerifyChallenge verifies a TOTP code and creates a session on success.
func (s *Service) VerifyChallenge(req *MFAVerifyReq, ip, userAgent string) (*MFAVerifyResult, error) {
	return s.VerifyChallengeWithContext(context.Background(), req, ip, userAgent)
}

func (s *Service) VerifyChallengeWithContext(ctx context.Context, req *MFAVerifyReq, ip, userAgent string) (*MFAVerifyResult, error) {
	if s.db == nil {
		return nil, common.ErrDatabaseNotInitialized
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if req == nil || strings.TrimSpace(req.ChallengeID) == "" {
		return nil, errors.New("auth.mfa.challenge_required")
	}

	now := time.Now()
	challenge, err := s.loadActiveChallenge(strings.TrimSpace(req.ChallengeID), now)
	if err != nil {
		return nil, err
	}

	secret, err := s.loadChallengeSecret(*challenge)
	if err != nil {
		return nil, err
	}
	if !ValidateTOTPCode(secret, req.Code, now) {
		return nil, errors.New("auth.mfa.code_invalid")
	}

	currentUser, err := s.identity.LoadUserByID(challenge.UserID)
	if err != nil {
		return nil, err
	}
	if currentUser.Status == common.StatusDisabled {
		return nil, errors.New("user.login.error.disabled")
	}

	if err := s.finalizeChallenge(*challenge, secret, now); err != nil {
		return nil, err
	}

	roles, err := s.identity.GetUserRoles(currentUser.ID)
	if err != nil {
		return nil, err
	}
	tokenPair, err := s.creator.CreateSessionWithContext(ctx, currentUser.ID, roles, ip, userAgent)
	if err != nil {
		return nil, err
	}

	return &MFAVerifyResult{
		UserID:    currentUser.ID,
		TokenPair: tokenPair,
		Username:  currentUser.Username,
		Nickname:  currentUser.Nickname,
		Avatar:    currentUser.Avatar,
		Email:     currentUser.Email,
		Phone:     currentUser.Phone,
		Roles:     roles,
	}, nil
}

func (s *Service) loadActiveChallenge(challengeID string, now time.Time) (*SystemAuthMFAChallenge, error) {
	var challenge SystemAuthMFAChallenge
	if err := s.db.Where("challenge_id = ?", challengeID).First(&challenge).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("auth.mfa.challenge_invalid")
		}
		return nil, err
	}
	if challenge.ConsumedAt != nil || !challenge.ExpiresAt.After(now) {
		return nil, errors.New("auth.mfa.challenge_expired")
	}
	return &challenge, nil
}

func (s *Service) loadChallengeSecret(challenge SystemAuthMFAChallenge) (string, error) {
	if challenge.SetupRequired == 1 {
		return DecryptMFASecret(challenge.SecretEncrypted)
	}
	var factor SystemAuthFactor
	if err := s.db.Where(userIDAndFactorTypeEnabledWhereClause, challenge.UserID, "totp", 1).First(&factor).Error; err != nil {
		return "", errors.New("auth.mfa.factor_missing")
	}
	return DecryptMFASecret(factor.SecretEncrypted)
}

func (s *Service) finalizeChallenge(challenge SystemAuthMFAChallenge, secret string, now time.Time) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		if challenge.SetupRequired == 1 {
			encryptedSecret, err := EncryptMFASecret(secret)
			if err != nil {
				return err
			}
			if err := upsertMFAFactor(tx, challenge.UserID, encryptedSecret, now); err != nil {
				return err
			}
		}
		result := tx.Model(&SystemAuthMFAChallenge{}).
			Where("id = ? AND consumed_at IS NULL", challenge.ID).
			Update("consumed_at", &now)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return errors.New("auth.mfa.challenge_expired")
		}
		return nil
	})
}

func upsertMFAFactor(tx *gorm.DB, userID uint64, encryptedSecret string, now time.Time) error {
	var factor SystemAuthFactor
	err := tx.Where(userIDAndFactorTypeWhereClause, userID, "totp").First(&factor).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		factor = SystemAuthFactor{
			UserID:          userID,
			FactorType:      "totp",
			SecretEncrypted: encryptedSecret,
			Enabled:         1,
			ConfirmedAt:     &now,
		}
		return tx.Create(&factor).Error
	}
	if err != nil {
		return err
	}
	return tx.Model(&factor).Updates(map[string]any{
		"secret_encrypted": encryptedSecret,
		"enabled":          common.StatusEnabled,
		"confirmed_at":     &now,
	}).Error
}

// UserRecord is the minimal user data needed by MFA service.
type UserRecord struct {
	ID       uint64
	Username string
	Nickname string
	Avatar   string
	Email    string
	Phone    string
	Status   int
}

const (
	userIDAndFactorTypeEnabledWhereClause = "user_id = ? AND factor_type = ? AND enabled = ?"
	userIDAndFactorTypeWhereClause        = "user_id = ? AND factor_type = ?"
)

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
