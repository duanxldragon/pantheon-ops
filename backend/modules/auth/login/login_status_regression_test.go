package login

import (
	"testing"

	"pantheon-base/pkg/common"
)

// Regression: SystemLogLogin.Status used to carry `gorm:"default:1"`, which made
// GORM replace the zero value (LoginStatusFailure = 0) with 1 on insert, so every
// failed login was persisted as a success.
func TestRecordLoginLogPersistsFailureStatus(t *testing.T) {
	db := setupTestDB(t)
	s := NewRuntime(db)

	s.RecordLoginLog("req-fail", "admin", "127.0.0.1", "curl", "Linux", common.LoginStatusFailure, "token.store.not_initialized")
	s.RecordLoginLog("req-ok", "admin", "127.0.0.1", "Chrome", "Windows", common.LoginStatusSuccess, "auth.loginSuccess")

	var rows []SystemLogLogin
	if err := db.Order("id").Find(&rows).Error; err != nil {
		t.Fatalf("load login logs: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("expected 2 login logs, got %d", len(rows))
	}
	if rows[0].Status != common.LoginStatusFailure {
		t.Fatalf("failed login persisted with status %d, want %d (failure)", rows[0].Status, common.LoginStatusFailure)
	}
	if rows[1].Status != common.LoginStatusSuccess {
		t.Fatalf("successful login persisted with status %d, want %d (success)", rows[1].Status, common.LoginStatusSuccess)
	}
}
