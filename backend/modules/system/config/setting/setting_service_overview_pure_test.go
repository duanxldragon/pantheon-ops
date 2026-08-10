package config

import (
	"testing"
)

// These tests exercise the extracted, pure overview/aggregation helpers without any
// external service (no MySQL DSN, no Redis). They lock the statistics aggregation 口径
// (required-key list, counts, issue detection) so the equivalent refactor cannot
// silently change behavior. Integration tests for UpdateGroup/GetOverview remain in
// setting_service_test.go and are DSN-gated (skip locally).

func containsString(s []string, target string) bool {
	for _, v := range s {
		if v == target {
			return true
		}
	}
	return false
}

func TestBuildRequiredSettingKeys(t *testing.T) {
	base := buildRequiredSettingKeys("local")
	for _, k := range []string{
		"site.name",
		settingKeyAppMode,
		"org.enabled",
		settingKeyI18nLanguage,
		settingKeyUITheme,
		settingKeyUploadDriver,
		settingKeyUploadAllowedTypes,
		"upload.local_path",
	} {
		if !containsString(base, k) {
			t.Fatalf("local profile missing required key %q; got %v", k, base)
		}
	}
	if containsString(base, "upload.s3_endpoint") {
		t.Fatalf("local profile must not include s3 keys; got %v", base)
	}

	s3 := buildRequiredSettingKeys("s3")
	for _, k := range []string{
		"upload.s3_endpoint",
		"upload.s3_bucket",
		"upload.s3_access_key_id",
		"upload.s3_secret_access_key",
	} {
		if !containsString(s3, k) {
			t.Fatalf("s3 profile missing required key %q; got %v", k, s3)
		}
	}
	if containsString(s3, "upload.local_path") {
		t.Fatalf("s3 profile must not include local_path; got %v", s3)
	}
}

func TestAggregateSettingOverviewCounts(t *testing.T) {
	rows := []SystemSetting{
		{SettingKey: "a", IsPublic: 1, IsEncrypted: 0},
		{SettingKey: "b", IsPublic: 0, IsEncrypted: 1},
		{SettingKey: "c", IsPublic: 1, IsEncrypted: 1},
	}
	resp := &SettingOverviewResp{Issues: make([]SettingOverviewIssueResp, 0)}
	byKey := make(map[string]SystemSetting)
	aggregateSettingOverviewCounts(rows, resp, byKey)

	if resp.TotalSettingCount != 3 {
		t.Fatalf("total = %d, want 3", resp.TotalSettingCount)
	}
	if resp.PublicSettingCount != 2 {
		t.Fatalf("public = %d, want 2", resp.PublicSettingCount)
	}
	if resp.EncryptedSettingCount != 2 {
		t.Fatalf("encrypted = %d, want 2", resp.EncryptedSettingCount)
	}
	if len(byKey) != 3 {
		t.Fatalf("byKey len = %d, want 3", len(byKey))
	}
}

func TestCheckRequiredSettingKeys(t *testing.T) {
	byKey := map[string]SystemSetting{
		"site.name":         {SettingKey: "site.name", SettingValue: "Pantheon"},
		"login.mfa_enabled": {SettingKey: "login.mfa_enabled", SettingValue: ""},
	}
	resp := &SettingOverviewResp{Issues: make([]SettingOverviewIssueResp, 0)}
	seen := make(map[string]struct{})
	checkRequiredSettingKeys(buildRequiredSettingKeys("local"), byKey, resp, seen)

	if resp.RequiredMissingCount == 0 {
		t.Fatalf("expected at least one missing required key, got 0")
	}
}

func TestCheckPublicEncryptedConflicts(t *testing.T) {
	rows := []SystemSetting{
		{SettingKey: "pub", GroupKey: "g", IsPublic: 1, IsEncrypted: 1},
		{SettingKey: "ok", GroupKey: "g", IsPublic: 1, IsEncrypted: 0},
	}
	resp := &SettingOverviewResp{Issues: make([]SettingOverviewIssueResp, 0)}
	seen := make(map[string]struct{})
	checkPublicEncryptedConflicts(rows, resp, seen)

	if len(resp.Issues) != 1 || resp.Issues[0].SettingKey != "pub" {
		t.Fatalf("expected single public-encrypted conflict for pub, got %+v", resp.Issues)
	}
	if resp.Issues[0].Severity != "critical" {
		t.Fatalf("expected critical severity, got %s", resp.Issues[0].Severity)
	}
}

func TestAppendAllowedValueIssues(t *testing.T) {
	resp := &SettingOverviewResp{
		Issues:          make([]SettingOverviewIssueResp, 0),
		StorageDriver:   "bogus-driver",
		DefaultLanguage: "xx-XX",
		DefaultTheme:    "tangerine",
	}
	byKey := map[string]SystemSetting{
		settingKeyAppMode: {SettingKey: settingKeyAppMode, SettingValue: "nonsense"},
	}
	seen := make(map[string]struct{})
	appendAllowedValueIssues(resp, seen, byKey)

	if len(resp.Issues) != 4 {
		t.Fatalf("expected 4 issues for invalid values, got %d: %+v", len(resp.Issues), resp.Issues)
	}

	respValid := &SettingOverviewResp{
		Issues:          make([]SettingOverviewIssueResp, 0),
		StorageDriver:   "local",
		DefaultLanguage: "zh-CN",
		DefaultTheme:    "indigo",
	}
	byKeyValid := map[string]SystemSetting{
		settingKeyAppMode: {SettingKey: settingKeyAppMode, SettingValue: "enterprise"},
	}
	seenValid := make(map[string]struct{})
	appendAllowedValueIssues(respValid, seenValid, byKeyValid)

	if len(respValid.Issues) != 0 {
		t.Fatalf("expected 0 issues for valid values, got %d: %+v", len(respValid.Issues), respValid.Issues)
	}
}
