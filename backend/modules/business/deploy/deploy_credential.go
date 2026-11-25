package deploy

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"strings"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/modules/business/bizscope"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/database"

	"gorm.io/gorm"
)

const deployCredentialKeyEnv = "PANTHEON_DEPLOY_CREDENTIAL_KEY"
const deployCredentialAuthModePassword = "password"
const deployCredentialAuthModePrivateKey = "private_key"

func encryptDeployCredential(value string) (string, error) {
	key, err := deployCredentialKey()
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	return "v1:" + base64.StdEncoding.EncodeToString(gcm.Seal(nonce, nonce, []byte(value), nil)), nil
}

func credentialResponse(item *DeployCredentialRef) DeployCredentialResponse {
	return DeployCredentialResponse{ID: item.ID, BusinessScopeID: item.BusinessScopeID, Name: item.Name, Username: item.Username, AuthMode: item.AuthMode, Version: item.Version, Status: item.Status, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}
}

// credentialVisibleQuery limits credential rows to business scopes the actor may
// read. SEC-004: a scoped actor never sees another department/business scope's
// credential, and legacy rows without an owner (business_scope_id = 0) stay
// admin-only instead of silently becoming a global secret pool.
func (s *DeployService) credentialVisibleQuery(dataScope *common.DataScopeReq) *gorm.DB {
	db := s.db.Model(&DeployCredentialRef{})
	if dataScope == nil || dataScope.IsAdmin {
		return db
	}
	visible := s.db.Model(&bizscope.BizScope{}).Select("id").Scopes(database.WithDataScope(dataScope))
	return db.Where("business_scope_id IN (?)", visible)
}

func (s *DeployService) loadVisibleCredential(id uint64, dataScope *common.DataScopeReq) (*DeployCredentialRef, error) {
	var item DeployCredentialRef
	if err := s.credentialVisibleQuery(dataScope).First(&item, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("business.deploy.credential.not_found")
		}
		return nil, err
	}
	return &item, nil
}

// validateCredentialBusinessScope requires an owner scope the actor can operate.
// scope 0 is the platform/admin-owned credential (legacy rows included); only an
// administrator may create or move a credential into that global bucket, so a
// restricted role can never turn a secret into a shared one.
func (s *DeployService) validateCredentialBusinessScope(id uint64, dataScope *common.DataScopeReq) error {
	if id == 0 {
		if dataScope == nil || dataScope.IsAdmin {
			return nil
		}
		return errors.New("business.deploy.credential.business_scope_required")
	}
	if s.bizScopeReader == nil {
		return errors.New("business.deploy.credential.business_scope_required")
	}
	if _, err := s.bizScopeReader.GetActive(s.requestContext(), id, dataScope); err != nil {
		return errors.New("business.deploy.credential.business_scope_forbidden")
	}
	return nil
}

// ListCredentials returns redacted credential metadata without secret material.
func (s *DeployService) ListCredentials(dataScope *common.DataScopeReq) ([]DeployCredentialResponse, error) {
	var items []DeployCredentialRef
	if err := s.credentialVisibleQuery(dataScope).Order("id DESC").Find(&items).Error; err != nil {
		return nil, err
	}
	result := make([]DeployCredentialResponse, 0, len(items))
	for i := range items {
		result = append(result, credentialResponse(&items[i]))
	}
	return result, nil
}

// CreateCredential encrypts and stores a deploy credential, returning metadata only.
func (s *DeployService) CreateCredential(req CreateDeployCredentialRequest, actor string, dataScope *common.DataScopeReq) (*DeployCredentialResponse, error) {
	if req.AuthMode != deployCredentialAuthModePassword && req.AuthMode != deployCredentialAuthModePrivateKey {
		return nil, errors.New("business.deploy.credential.auth_mode_invalid")
	}
	if err := s.validateCredentialBusinessScope(req.BusinessScopeID, dataScope); err != nil {
		return nil, err
	}
	secret, err := encryptDeployCredential(req.Secret)
	if err != nil {
		return nil, err
	}
	item := DeployCredentialRef{BusinessScopeID: req.BusinessScopeID, Name: strings.TrimSpace(req.Name), Username: strings.TrimSpace(req.Username), AuthMode: req.AuthMode, SecretEncrypted: secret, Version: 1, Status: bizScopeStatusActive, CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := s.db.Create(&item).Error; err != nil {
		return nil, err
	}
	resp := credentialResponse(&item)
	return &resp, nil
}

// UpdateCredential changes deploy credential metadata or rotates its encrypted secret.
func (s *DeployService) UpdateCredential(id uint64, req UpdateDeployCredentialRequest, actor string, dataScope *common.DataScopeReq) (*DeployCredentialResponse, error) {
	item, err := s.loadVisibleCredential(id, dataScope)
	if err != nil {
		return nil, err
	}
	if req.BusinessScopeID != nil && *req.BusinessScopeID != item.BusinessScopeID {
		if err := s.validateCredentialBusinessScope(*req.BusinessScopeID, dataScope); err != nil {
			return nil, err
		}
	}
	updates := map[string]any{"updated_at": time.Now()}
	if req.BusinessScopeID != nil {
		updates["business_scope_id"] = *req.BusinessScopeID
	}
	if req.Name != nil {
		updates["name"] = strings.TrimSpace(*req.Name)
	}
	if req.Username != nil {
		updates["username"] = strings.TrimSpace(*req.Username)
	}
	if req.AuthMode != nil {
		if *req.AuthMode != deployCredentialAuthModePassword && *req.AuthMode != deployCredentialAuthModePrivateKey {
			return nil, errors.New("business.deploy.credential.auth_mode_invalid")
		}
		updates["auth_mode"] = *req.AuthMode
	}
	if req.Status != nil {
		if *req.Status != bizScopeStatusActive && *req.Status != "disabled" {
			return nil, errors.New("business.deploy.credential.status_invalid")
		}
		updates["status"] = *req.Status
	}
	if req.Secret != nil && strings.TrimSpace(*req.Secret) != "" {
		encrypted, err := encryptDeployCredential(*req.Secret)
		if err != nil {
			return nil, err
		}
		updates["secret_encrypted"] = encrypted
		updates["version"] = item.Version + 1
	}
	if err := s.db.Model(&item).Updates(updates).Error; err != nil {
		return nil, err
	}
	if err := s.db.First(item, id).Error; err != nil {
		return nil, err
	}
	resp := credentialResponse(item)
	return &resp, nil
}

// DeleteCredential removes a deploy credential reference by ID. A credential
// still referenced by any task is refused: tasks keep the immutable id/version
// that produced their execution snapshot, including after they finish.
func (s *DeployService) DeleteCredential(id uint64, dataScope *common.DataScopeReq) error {
	if _, err := s.loadVisibleCredential(id, dataScope); err != nil {
		return err
	}
	var used int64
	if err := s.db.Model(&DeployTask{}).Where("credential_ref_id = ?", id).Count(&used).Error; err != nil {
		return err
	}
	if used > 0 {
		return errors.New("business.deploy.credential.in_use")
	}
	result := s.db.Delete(&DeployCredentialRef{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("business.deploy.credential.not_found")
	}
	return nil
}

func decryptDeployCredential(value string) (string, error) {
	version, encoded, ok := strings.Cut(value, ":")
	if !ok || version != "v1" {
		return "", errors.New("business.deploy.credential.malformed")
	}
	key, err := deployCredentialKey()
	if err != nil {
		return "", err
	}
	raw, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", errors.New("business.deploy.credential.malformed")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(raw) < gcm.NonceSize() {
		return "", errors.New("business.deploy.credential.malformed")
	}
	plain, err := gcm.Open(nil, raw[:gcm.NonceSize()], raw[gcm.NonceSize():], nil)
	if err != nil {
		return "", errors.New("business.deploy.credential.tampered")
	}
	return string(plain), nil
}

func deployCredentialKey() ([]byte, error) {
	raw := strings.TrimSpace(os.Getenv(deployCredentialKeyEnv))
	if raw == "" {
		return nil, errors.New("business.deploy.credential.key_missing")
	}
	key, err := hex.DecodeString(raw)
	if err != nil || len(key) != 32 {
		return nil, errors.New("business.deploy.credential.key_invalid")
	}
	return key, nil
}

// resolveStartCredential resolves an SSH credential for a task that is already
// authorized. The credential must belong to the same business scope as the
// task, so an actor cannot start a task with another scope's secret by ID. A
// global (scope 0) credential is admin-owned and only a system/admin caller may
// use it.
func (s *DeployService) resolveStartCredential(req StartTaskRequest, task DeployTask, allowGlobal bool) (StartTaskRequest, uint64, error) {
	if req.CredentialRefID == 0 {
		return req, 0, errors.New("business.deploy.credential.required")
	}
	var ref DeployCredentialRef
	if err := s.db.First(&ref, req.CredentialRefID).Error; err != nil {
		return req, 0, errors.New("business.deploy.credential.not_found")
	}
	if ref.BusinessScopeID != task.BusinessScopeID && (ref.BusinessScopeID != 0 || !allowGlobal) {
		return req, 0, errors.New("business.deploy.credential.out_of_scope")
	}
	if ref.Status != bizScopeStatusActive {
		return req, 0, errors.New("business.deploy.credential.inactive")
	}
	secret, err := decryptDeployCredential(ref.SecretEncrypted)
	if err != nil {
		return req, 0, err
	}
	req.SSHUser, req.AuthMode = ref.Username, ref.AuthMode
	if ref.AuthMode == deployCredentialAuthModePrivateKey {
		req.SSHPrivateKey = secret
	} else {
		req.SSHPassword = secret
	}
	return req, ref.Version, nil
}

func validateCredentialReferenceRequest(req StartTaskRequest) error {
	if req.CredentialRefID == 0 {
		return errors.New("business.deploy.credential.required")
	}
	if strings.TrimSpace(req.SSHUser) != "" || strings.TrimSpace(req.SSHPassword) != "" || strings.TrimSpace(req.SSHPrivateKey) != "" || strings.TrimSpace(req.AuthMode) != "" {
		return errors.New("business.deploy.credential.inline_forbidden")
	}
	return nil
}
