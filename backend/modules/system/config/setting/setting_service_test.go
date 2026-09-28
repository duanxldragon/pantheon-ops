package config

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/testmysql"
	"gorm.io/gorm"
)

func setupSettingTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	return testmysql.Open(t)
}

func TestSettingService_UpdateGroupInvalidatesPublicCache(t *testing.T) {
	db := setupSettingTestDB(t)
	service := NewSettingService(db)
	if err := service.Migrate(); err != nil {
		t.Fatalf("migrate setting: %v", err)
	}

	publicSettings, err := service.GetPublicSettings()
	if err != nil {
		t.Fatalf("load public settings: %v", err)
	}
	if publicSettings.Settings["site.name"] != "Pantheon Base" {
		t.Fatalf("expected default site name, got %s", publicSettings.Settings["site.name"])
	}

	updated, err := service.UpdateGroup("basic", &SettingGroupUpdateReq{Items: []SettingUpdateItemReq{
		{SettingKey: "site.name", SettingValue: "Pantheon QA"},
		{SettingKey: "site.logo", SettingValue: "https://example.com/logo.png"},
	}})
	if err != nil {
		t.Fatalf("update basic settings: %v", err)
	}
	if updated.Items[0].SettingValue != "Pantheon QA" {
		t.Fatalf("expected updated group value, got %+v", updated.Items)
	}

	publicSettings, err = service.GetPublicSettings()
	if err != nil {
		t.Fatalf("reload public settings: %v", err)
	}
	if publicSettings.Settings["site.name"] != "Pantheon QA" {
		t.Fatalf("expected cache invalidated site name, got %s", publicSettings.Settings["site.name"])
	}
	if publicSettings.Settings["site.logo"] != "https://example.com/logo.png" {
		t.Fatalf("expected updated site logo, got %s", publicSettings.Settings["site.logo"])
	}

	if _, err := service.UpdateGroup("ui", &SettingGroupUpdateReq{Items: []SettingUpdateItemReq{
		{SettingKey: "ui.default_theme", SettingValue: "emerald"},
	}}); err != nil {
		t.Fatalf("update ui theme: %v", err)
	}

	publicSettings, err = service.GetPublicSettings()
	if err != nil {
		t.Fatalf("reload public settings for theme: %v", err)
	}
	if publicSettings.Settings["ui.default_theme"] != "emerald" {
		t.Fatalf("expected updated public theme emerald, got %s", publicSettings.Settings["ui.default_theme"])
	}
}

func TestSettingService_EncryptedEmptyUpdateKeepsCurrentValue(t *testing.T) {
	db := setupSettingTestDB(t)
	service := NewSettingService(db)
	if err := service.Migrate(); err != nil {
		t.Fatalf("migrate setting: %v", err)
	}

	if _, err := service.UpdateGroup("upload", &SettingGroupUpdateReq{Items: []SettingUpdateItemReq{
		{SettingKey: "upload.s3_access_key_id", SettingValue: "secret-access-key"},
	}}); err != nil {
		t.Fatalf("set encrypted setting: %v", err)
	}

	if _, err := service.UpdateGroup("upload", &SettingGroupUpdateReq{Items: []SettingUpdateItemReq{
		{SettingKey: "upload.s3_access_key_id", SettingValue: ""},
	}}); err != nil {
		t.Fatalf("keep encrypted setting with empty value: %v", err)
	}

	value, err := service.GetByKey("upload.s3_access_key_id")
	if err != nil {
		t.Fatalf("get encrypted setting: %v", err)
	}
	if value != "secret-access-key" {
		t.Fatalf("expected encrypted value to be kept, got %s", value)
	}
}

func TestSettingService_UpdateGroupValidatesValueType(t *testing.T) {
	db := setupSettingTestDB(t)
	service := NewSettingService(db)
	if err := service.Migrate(); err != nil {
		t.Fatalf("migrate setting: %v", err)
	}

	cases := []settingUpdateValidationCase{
		{name: "invalid json", group: "upload", key: "upload.allowed_types", value: "[invalid json]", wantError: "setting.value.invalid_json"},
		{name: "invalid number", group: "upload", key: "upload.max_file_size", value: "not-a-number", wantError: "setting.value.invalid_number"},
		{name: "invalid storage driver", group: "upload", key: "upload.storage_driver", value: "ftp", wantError: "setting.value.invalid_option"},
		{name: "legacy storage driver alias", group: "upload", key: "upload.storage_driver", value: "s3-compatible", wantValue: "s3"},
		{name: "supported default language", group: "i18n", key: "i18n.default_language", value: "fr-FR"},
		{name: "legacy theme alias", group: "ui", key: "ui.default_theme", value: "light"},
		{name: "json object instead of array", group: "upload", key: "upload.allowed_types", value: `{"jpg":true}`, wantError: "setting.value.invalid_json"},
		{name: "normalize login retention options", group: "audit", key: "audit.login_log_retention_options", value: "[30,7,30,1]", wantValue: "[1,7,30]"},
		{name: "normalize session retention options", group: "audit", key: "audit.session_cleanup_retention_options", value: "[30,7,30,1]", wantValue: "[1,7,30]"},
		{name: "empty retention options", group: "audit", key: "audit.operation_log_retention_options", value: "[]", wantError: "setting.value.invalid_option"},
		{name: "non-positive retention option", group: "audit", key: "audit.operation_log_retention_options", value: "[0,7]", wantError: "setting.value.invalid_option"},
		{name: "invalid app mode", group: "platform", key: "platform.app_mode", value: "invalid", wantError: "setting.value.invalid_option"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assertSettingUpdateValidation(t, service, tc)
		})
	}
}

type settingUpdateValidationCase struct {
	name      string
	group     string
	key       string
	value     string
	wantError string
	wantValue string
}

func assertSettingUpdateValidation(t *testing.T, service *SettingService, tc settingUpdateValidationCase) {
	t.Helper()
	updated, err := service.UpdateGroup(tc.group, &SettingGroupUpdateReq{Items: []SettingUpdateItemReq{
		{SettingKey: tc.key, SettingValue: tc.value},
	}})
	if tc.wantError != "" {
		if err == nil || common.ErrMessage(err) != tc.wantError {
			t.Fatalf("expected %s, got %v", tc.wantError, err)
		}
		return
	}
	if err != nil {
		t.Fatalf("expected update to succeed, got %v", err)
	}
	if tc.wantValue != "" {
		assertUpdatedSettingValue(t, updated.Items, tc.key, tc.wantValue)
	}
}

func assertUpdatedSettingValue(t *testing.T, items []SettingResp, key, want string) {
	t.Helper()
	for _, item := range items {
		if item.SettingKey == key {
			if item.SettingValue != want {
				t.Fatalf("expected %s to normalize to %s, got %s", key, want, item.SettingValue)
			}
			return
		}
	}
	t.Fatalf("expected updated settings to contain %s", key)
}

func TestSettingService_MigrateSeedsPlatformCapabilities(t *testing.T) {
	db := setupSettingTestDB(t)
	service := NewSettingService(db)
	if err := service.Migrate(); err != nil {
		t.Fatalf("migrate setting: %v", err)
	}

	publicSettings, err := service.GetPublicSettings()
	if err != nil {
		t.Fatalf("load public settings: %v", err)
	}
	if publicSettings.Settings["platform.app_mode"] != "enterprise" {
		t.Fatalf("expected enterprise app mode, got %s", publicSettings.Settings["platform.app_mode"])
	}
	if publicSettings.Settings["org.enabled"] != "true" {
		t.Fatalf("expected org enabled, got %s", publicSettings.Settings["org.enabled"])
	}
	if publicSettings.Settings["org.required_for_user"] != "false" {
		t.Fatalf("expected org not required for users, got %s", publicSettings.Settings["org.required_for_user"])
	}
}

func TestSettingService_MigrateSeedsS3Region(t *testing.T) {
	db := setupSettingTestDB(t)
	service := NewSettingService(db)
	if err := service.Migrate(); err != nil {
		t.Fatalf("migrate setting: %v", err)
	}

	value, err := service.GetByKey("upload.s3_region")
	if err != nil {
		t.Fatalf("get upload.s3_region: %v", err)
	}
	if value != "us-east-1" {
		t.Fatalf("expected default s3 region, got %s", value)
	}
}

func TestSettingService_MigrateSeedsUploadAllowedTypesForDeployArchives(t *testing.T) {
	db := setupSettingTestDB(t)
	service := NewSettingService(db)
	if err := service.Migrate(); err != nil {
		t.Fatalf("migrate setting: %v", err)
	}

	value, err := service.GetByKey("upload.allowed_types")
	if err != nil {
		t.Fatalf("get upload.allowed_types: %v", err)
	}
	if value != settingValueUploadAllowedTypesDefault {
		t.Fatalf("unexpected upload.allowed_types default: %s", value)
	}
}

func TestSettingService_MigrateUpgradesLegacyUploadAllowedTypes(t *testing.T) {
	cases := []struct {
		name        string
		legacyValue string
	}{
		{
			name:        "legacy pre-archive seed",
			legacyValue: settingValueUploadAllowedTypesLegacy,
		},
		{
			name:        "legacy archive seed",
			legacyValue: settingValueUploadAllowedTypesArchives,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assertLegacyUploadAllowedTypesMigration(t, tc.legacyValue)
		})
	}
}

func assertLegacyUploadAllowedTypesMigration(t *testing.T, legacyValue string) {
	t.Helper()
	db := setupSettingTestDB(t)
	service := NewSettingService(db)
	if err := db.AutoMigrate(&SystemSetting{}); err != nil {
		t.Fatalf("create setting table: %v", err)
	}
	if err := db.Create(&SystemSetting{
		SettingKey:   "upload.allowed_types",
		SettingValue: legacyValue,
		ValueType:    "json",
		GroupKey:     "upload",
		Module:       "system",
		Remark:       "legacy upload types",
	}).Error; err != nil {
		t.Fatalf("seed legacy upload.allowed_types: %v", err)
	}
	if err := service.Migrate(); err != nil {
		t.Fatalf("migrate setting: %v", err)
	}
	value, err := service.GetByKey("upload.allowed_types")
	if err != nil {
		t.Fatalf("get upload.allowed_types: %v", err)
	}
	if value != settingValueUploadAllowedTypesDefault {
		t.Fatalf("unexpected migrated upload.allowed_types: %s", value)
	}
}

func TestSettingService_ListIncludesDefaultValueMetadata(t *testing.T) {
	db := setupSettingTestDB(t)
	service := NewSettingService(db)
	if err := service.Migrate(); err != nil {
		t.Fatalf("migrate setting: %v", err)
	}

	items, err := service.List(&SettingListQuery{GroupKey: "audit"})
	if err != nil {
		t.Fatalf("list audit settings: %v", err)
	}
	if len(items) == 0 {
		t.Fatalf("expected audit settings")
	}

	defaults := map[string]string{}
	for _, item := range items {
		defaults[item.SettingKey] = item.DefaultValue
	}
	if defaults["audit.login_log_retention_options"] != "[1,7,30]" {
		t.Fatalf("expected login log retention default metadata, got %s", defaults["audit.login_log_retention_options"])
	}
	if defaults["audit.operation_log_retention_options"] != "[1,7,30]" {
		t.Fatalf("expected operation log retention default metadata, got %s", defaults["audit.operation_log_retention_options"])
	}
	if defaults["audit.session_cleanup_retention_options"] != "[1,7,30]" {
		t.Fatalf("expected session cleanup retention default metadata, got %s", defaults["audit.session_cleanup_retention_options"])
	}
	if defaults["audit.login_log_retention_days"] != "90" {
		t.Fatalf("expected login log retention days default metadata, got %s", defaults["audit.login_log_retention_days"])
	}
	if defaults["audit.operation_log_retention_days"] != "180" {
		t.Fatalf("expected operation log retention days default metadata, got %s", defaults["audit.operation_log_retention_days"])
	}
	if defaults["audit.session_retention_days"] != "90" {
		t.Fatalf("expected session retention days default metadata, got %s", defaults["audit.session_retention_days"])
	}
}

func TestSettingService_MigrateSeedsAuthSecurityPolicySettings(t *testing.T) {
	db := setupSettingTestDB(t)
	service := NewSettingService(db)
	if err := service.Migrate(); err != nil {
		t.Fatalf("migrate setting: %v", err)
	}

	items, err := service.List(&SettingListQuery{GroupKey: "security", Module: "system.auth"})
	if err != nil {
		t.Fatalf("list auth security settings: %v", err)
	}
	defaults := map[string]string{}
	for _, item := range items {
		defaults[item.SettingKey] = item.DefaultValue
	}
	if defaults["security.password_history_limit"] != "0" {
		t.Fatalf("expected password history limit default metadata, got %s", defaults["security.password_history_limit"])
	}
	if defaults["security.password_expire_days"] != "0" {
		t.Fatalf("expected password expire days default metadata, got %s", defaults["security.password_expire_days"])
	}

	loginItems, err := service.List(&SettingListQuery{GroupKey: "login", Module: "system.auth"})
	if err != nil {
		t.Fatalf("list auth login settings: %v", err)
	}
	loginDefaults := map[string]string{}
	for _, item := range loginItems {
		loginDefaults[item.SettingKey] = item.DefaultValue
	}
	if loginDefaults["login.security_event_enabled"] != "true" {
		t.Fatalf("expected security event enabled default metadata, got %s", loginDefaults["login.security_event_enabled"])
	}
}

func TestSettingService_MigrateUpgradesLegacySessionCleanupRetentionDefault(t *testing.T) {
	db := setupSettingTestDB(t)
	service := NewSettingService(db)
	if err := db.AutoMigrate(&SystemSetting{}); err != nil {
		t.Fatalf("create setting table: %v", err)
	}

	if err := db.Create(&SystemSetting{
		SettingKey:   "audit.session_cleanup_retention_options",
		SettingValue: "[7,30,90]",
		ValueType:    "json",
		GroupKey:     "audit",
		Module:       "system",
		Remark:       "legacy",
	}).Error; err != nil {
		t.Fatalf("seed legacy setting: %v", err)
	}

	if err := service.Migrate(); err != nil {
		t.Fatalf("migrate setting: %v", err)
	}

	value, err := service.GetByKey("audit.session_cleanup_retention_options")
	if err != nil {
		t.Fatalf("get migrated setting: %v", err)
	}
	if value != "[1,7,30]" {
		t.Fatalf("expected upgraded session cleanup options, got %s", value)
	}
}

func TestSettingService_BuildAuditPayloadNormalizesSemanticValues(t *testing.T) {
	db := setupSettingTestDB(t)
	service := NewSettingService(db)
	if err := service.Migrate(); err != nil {
		t.Fatalf("migrate setting: %v", err)
	}

	payload, err := service.BuildAuditPayload("upload", &SettingGroupUpdateReq{Items: []SettingUpdateItemReq{
		{SettingKey: "upload.storage_driver", SettingValue: "s3-compatible"},
	}}, true)
	if err != nil {
		t.Fatalf("build upload audit payload: %v", err)
	}

	var uploadAudit struct {
		GroupKey string                   `json:"groupKey"`
		Changes  []SettingAuditChangeResp `json:"changes"`
	}
	if err := json.Unmarshal([]byte(payload), &uploadAudit); err != nil {
		t.Fatalf("unmarshal upload audit payload: %v", err)
	}
	if uploadAudit.GroupKey != "upload" {
		t.Fatalf("expected upload group, got %s", uploadAudit.GroupKey)
	}
	if len(uploadAudit.Changes) != 1 {
		t.Fatalf("expected 1 upload change, got %d", len(uploadAudit.Changes))
	}
	if uploadAudit.Changes[0].NewValue != "s3" {
		t.Fatalf("expected normalized storage driver in audit, got %s", uploadAudit.Changes[0].NewValue)
	}
	if uploadAudit.Changes[0].OldValue != "local" {
		t.Fatalf("expected old storage driver local, got %s", uploadAudit.Changes[0].OldValue)
	}

	payload, err = service.BuildAuditPayload("ui", &SettingGroupUpdateReq{Items: []SettingUpdateItemReq{
		{SettingKey: "ui.default_theme", SettingValue: "light"},
	}}, true)
	if err != nil {
		t.Fatalf("build ui audit payload: %v", err)
	}

	var uiAudit struct {
		GroupKey string                   `json:"groupKey"`
		Changes  []SettingAuditChangeResp `json:"changes"`
	}
	if err := json.Unmarshal([]byte(payload), &uiAudit); err != nil {
		t.Fatalf("unmarshal ui audit payload: %v", err)
	}
	if len(uiAudit.Changes) != 0 {
		t.Fatalf("expected normalized alias with no actual ui change, got %+v", uiAudit.Changes)
	}
}

func TestSettingService_GetOverview(t *testing.T) {
	db := setupSettingTestDB(t)
	service := NewSettingService(db)
	if err := service.Migrate(); err != nil {
		t.Fatalf("migrate setting: %v", err)
	}

	if _, err := service.UpdateGroup("upload", &SettingGroupUpdateReq{Items: []SettingUpdateItemReq{
		{SettingKey: "upload.storage_driver", SettingValue: "s3"},
		{SettingKey: "upload.local_path", SettingValue: ""},
		{SettingKey: "upload.s3_endpoint", SettingValue: ""},
		{SettingKey: "upload.s3_bucket", SettingValue: ""},
		{SettingKey: "upload.s3_access_key_id", SettingValue: ""},
		{SettingKey: "upload.s3_secret_access_key", SettingValue: ""},
	}}); err != nil {
		t.Fatalf("prepare upload settings: %v", err)
	}

	overview, err := service.GetOverview()
	if err != nil {
		t.Fatalf("get overview: %v", err)
	}
	if overview.TotalSettingCount == 0 || overview.PublicSettingCount == 0 || overview.EncryptedSettingCount == 0 {
		t.Fatalf("expected non-zero basic overview counts, got %+v", overview)
	}
	if overview.StorageDriver != "s3" {
		t.Fatalf("expected storage driver s3, got %+v", overview)
	}
	if overview.RequiredMissingCount != 4 {
		t.Fatalf("expected 4 required missing settings for s3 mode, got %+v", overview)
	}
	if overview.RiskCount != 4 {
		t.Fatalf("expected 4 overview risks, got %+v", overview)
	}

	issueKeys := make(map[string]string, len(overview.Issues))
	for _, issue := range overview.Issues {
		issueKeys[issue.SettingKey] = issue.ReasonKey
	}
	expectedKeys := []string{
		"upload.s3_endpoint",
		"upload.s3_bucket",
		"upload.s3_access_key_id",
		"upload.s3_secret_access_key",
	}
	for _, key := range expectedKeys {
		if issueKeys[key] != "setting.overview.issue.required_missing" {
			t.Fatalf("expected missing issue for %s, got %+v", key, overview.Issues)
		}
	}
}

func TestSettingService_ExportAudit(t *testing.T) {
	db := setupSettingTestDB(t)
	service := NewSettingService(db)
	if err := service.Migrate(); err != nil {
		t.Fatalf("migrate setting: %v", err)
	}
	if err := db.AutoMigrate(&systemSettingAuditLog{}); err != nil {
		t.Fatalf("migrate setting audit log: %v", err)
	}

	payload, err := json.Marshal(map[string]any{
		"groupKey": "upload",
		"changes": []map[string]any{
			{
				"settingKey":  "upload.storage_driver",
				"oldValue":    "local",
				"newValue":    "s3",
				"isEncrypted": 0,
			},
			{
				"settingKey":  "upload.s3_secret_access_key",
				"oldValue":    "***",
				"newValue":    "***",
				"isEncrypted": 1,
			},
		},
	})
	if err != nil {
		t.Fatalf("marshal audit payload: %v", err)
	}
	if err := db.Create(&systemSettingAuditLog{
		Title:     settingAuditTitle,
		OperName:  "admin",
		OperIP:    "127.0.0.1",
		OperParam: string(payload),
		Status:    1,
		OperTime:  time.Date(2026, 4, 28, 10, 0, 0, 0, time.UTC),
		CostTime:  18,
	}).Error; err != nil {
		t.Fatalf("seed setting audit log: %v", err)
	}

	file, err := service.ExportAudit(&SettingAuditQuery{GroupKey: "upload"})
	if err != nil {
		t.Fatalf("export setting audit: %v", err)
	}
	if file.Filename != "system-setting-audit-export.csv" {
		t.Fatalf("unexpected filename: %s", file.Filename)
	}
	if len(file.Headers) != 8 || file.Headers[0] != "groupKey" || file.Headers[3] != "changes" {
		t.Fatalf("unexpected headers: %+v", file.Headers)
	}
	if len(file.Rows) != 1 {
		t.Fatalf("expected one audit export row, got %+v", file.Rows)
	}
	if file.Rows[0][0] != "upload" || file.Rows[0][1] != "admin" {
		t.Fatalf("unexpected audit export identity row: %+v", file.Rows[0])
	}
	if file.Rows[0][3] != "upload.storage_driver:local->s3 | upload.s3_secret_access_key:***->***" {
		t.Fatalf("unexpected changes export column: %+v", file.Rows[0])
	}
}
