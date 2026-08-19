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

	"gorm.io/gorm"
)

const deployCredentialKeyEnv = "PANTHEON_DEPLOY_CREDENTIAL_KEY"

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
	return DeployCredentialResponse{ID: item.ID, Name: item.Name, Username: item.Username, AuthMode: item.AuthMode, Version: item.Version, Status: item.Status, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}
}

func (s *DeployService) ListCredentials() ([]DeployCredentialResponse, error) {
	var items []DeployCredentialRef
	if err := s.db.Order("id DESC").Find(&items).Error; err != nil {
		return nil, err
	}
	result := make([]DeployCredentialResponse, 0, len(items))
	for i := range items {
		result = append(result, credentialResponse(&items[i]))
	}
	return result, nil
}

func (s *DeployService) CreateCredential(req CreateDeployCredentialRequest, actor string) (*DeployCredentialResponse, error) {
	if req.AuthMode != "password" && req.AuthMode != "private_key" {
		return nil, errors.New("business.deploy.credential.auth_mode_invalid")
	}
	secret, err := encryptDeployCredential(req.Secret)
	if err != nil {
		return nil, err
	}
	item := DeployCredentialRef{Name: strings.TrimSpace(req.Name), Username: strings.TrimSpace(req.Username), AuthMode: req.AuthMode, SecretEncrypted: secret, Version: 1, Status: "active", CreatedAt: time.Now(), UpdatedAt: time.Now()}
	if err := s.db.Create(&item).Error; err != nil {
		return nil, err
	}
	resp := credentialResponse(&item)
	return &resp, nil
}

func (s *DeployService) UpdateCredential(id uint64, req UpdateDeployCredentialRequest, actor string) (*DeployCredentialResponse, error) {
	var item DeployCredentialRef
	if err := s.db.First(&item, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("business.deploy.credential.not_found")
		}
		return nil, err
	}
	updates := map[string]any{"updated_at": time.Now()}
	if req.Name != nil {
		updates["name"] = strings.TrimSpace(*req.Name)
	}
	if req.Username != nil {
		updates["username"] = strings.TrimSpace(*req.Username)
	}
	if req.AuthMode != nil {
		if *req.AuthMode != "password" && *req.AuthMode != "private_key" {
			return nil, errors.New("business.deploy.credential.auth_mode_invalid")
		}
		updates["auth_mode"] = *req.AuthMode
	}
	if req.Status != nil {
		if *req.Status != "active" && *req.Status != "disabled" {
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
	if err := s.db.First(&item, id).Error; err != nil {
		return nil, err
	}
	resp := credentialResponse(&item)
	return &resp, nil
}

func (s *DeployService) DeleteCredential(id uint64) error {
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

func (s *DeployService) resolveStartCredential(req StartTaskRequest) (StartTaskRequest, uint64, error) {
	if req.CredentialRefID == 0 {
		return req, 0, errors.New("business.deploy.credential.required")
	}
	var ref DeployCredentialRef
	if err := s.db.First(&ref, req.CredentialRefID).Error; err != nil {
		return req, 0, errors.New("business.deploy.credential.not_found")
	}
	if ref.Status != "active" {
		return req, 0, errors.New("business.deploy.credential.inactive")
	}
	secret, err := decryptDeployCredential(ref.SecretEncrypted)
	if err != nil {
		return req, 0, err
	}
	req.SSHUser, req.AuthMode = ref.Username, ref.AuthMode
	if ref.AuthMode == "private_key" {
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
