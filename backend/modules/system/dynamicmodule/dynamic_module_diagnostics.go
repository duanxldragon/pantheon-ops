package dynamicmodule

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"time"
)

func (s *DynamicModuleService) persistModuleDiagnostics(module *ModuleRegistration, status int, lastError string, verifications []GeneratedModuleVerification) error {
	if s.db == nil || module == nil {
		return nil
	}
	now := time.Now().Format(time.RFC3339)
	encoded, err := encodeModuleVerifications(verifications)
	if err != nil {
		return err
	}
	updates := map[string]interface{}{
		"status":                   status,
		"last_verified_at":         now,
		"last_error":               trimDiagnosticError(lastError),
		"last_verification_result": encoded,
	}
	if err := s.db.Model(module).Updates(updates).Error; err != nil {
		return err
	}
	module.Status = status
	module.LastVerifiedAt = now
	module.LastError = trimDiagnosticError(lastError)
	module.LastVerificationResult = encoded
	return nil
}

func encodeModuleVerifications(verifications []GeneratedModuleVerification) (string, error) {
	if len(verifications) == 0 {
		return "[]", nil
	}
	raw, err := json.Marshal(verifications)
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

func trimDiagnosticError(value string) string {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) <= 512 {
		return trimmed
	}
	return trimmed[:512]
}

func buildArtifactMissingVerification(scope string, name string) GeneratedModuleVerification {
	return GeneratedModuleVerification{
		Code:       "artifact_check",
		Status:     "warn",
		MessageKey: "module.generate.verify.artifacts_missing",
		Detail:     filepath.ToSlash(filepath.Join("modules", scope, name)),
	}
}
