package generator

import "testing"

func TestSuggestModuleNameMatchesScopeConventions(t *testing.T) {
	tests := []struct {
		tableName string
		want      string
	}{
		{tableName: "biz_cmdb_host", want: "cmdb/host"},
		{tableName: "biz_vendor", want: "vendor"},
		{tableName: "system_login_log", want: "login_log"},
	}

	for _, tt := range tests {
		t.Run(tt.tableName, func(t *testing.T) {
			if got := suggestModuleName(tt.tableName); got != tt.want {
				t.Fatalf("suggestModuleName(%q) = %q, want %q", tt.tableName, got, tt.want)
			}
		})
	}
}

func TestMapColumnToFieldInfersEnvironmentEnumFromVarchar(t *testing.T) {
	field := mapColumnToField(columnRow{
		ColumnName: "environment",
		DataType:   "varchar",
		ColumnType: "varchar(50)",
		IsNullable: "NO",
	})

	if field.Type != "enum" {
		t.Fatalf("expected environment field type enum, got %q", field.Type)
	}
	if field.DictCode != "environment" {
		t.Fatalf("expected dictCode environment, got %q", field.DictCode)
	}
	if len(field.EnumOptions) != 4 {
		t.Fatalf("expected 4 enum options, got %d", len(field.EnumOptions))
	}
	if field.Validation == nil || len(field.Validation.Enum) != 4 {
		t.Fatalf("expected validation enum to be populated")
	}
	if field.Label != "环境" || field.LabelEn != "Environment" {
		t.Fatalf("unexpected environment labels: %q / %q", field.Label, field.LabelEn)
	}
	if field.Placeholder != "请选择环境" || field.PlaceholderEn != "Select environment" {
		t.Fatalf("unexpected environment placeholders: %q / %q", field.Placeholder, field.PlaceholderEn)
	}
	if field.EnumOptions[0].Label != "开发" || field.EnumOptions[0].LabelEn != "Development" {
		t.Fatalf("unexpected environment option labels: %#v", field.EnumOptions[0])
	}
}

func TestMapColumnToFieldInfersStatusEnumFromVarchar(t *testing.T) {
	field := mapColumnToField(columnRow{
		ColumnName: "status",
		DataType:   "varchar",
		ColumnType: "varchar(50)",
		IsNullable: "YES",
	})

	if field.Type != "enum" {
		t.Fatalf("expected status field type enum, got %q", field.Type)
	}
	if field.DictCode != "status" {
		t.Fatalf("expected dictCode status, got %q", field.DictCode)
	}
	if len(field.EnumOptions) != 2 {
		t.Fatalf("expected 2 enum options, got %d", len(field.EnumOptions))
	}
	if field.EnumOptions[0].Label != "启用" || field.EnumOptions[0].LabelEn != "Active" {
		t.Fatalf("unexpected status option labels: %#v", field.EnumOptions[0])
	}
}

func TestMapColumnToFieldSplitsChineseAndEnglishFallbacks(t *testing.T) {
	field := mapColumnToField(columnRow{
		ColumnName: "ip_address",
		DataType:   "varchar",
		ColumnType: "varchar(64)",
		IsNullable: "NO",
	})

	if field.Name != "ipAddress" {
		t.Fatalf("unexpected field name: %s", field.Name)
	}
	if field.Label != "IP 地址" || field.LabelEn != "IP address" {
		t.Fatalf("unexpected labels: %q / %q", field.Label, field.LabelEn)
	}
	if field.Placeholder != "请输入IP 地址" || field.PlaceholderEn != "Enter ip address" {
		t.Fatalf("unexpected placeholders: %q / %q", field.Placeholder, field.PlaceholderEn)
	}
}

func TestNormalizeDatasourceReqRejectsUnsafeHosts(t *testing.T) {
	tests := []struct {
		name      string
		host      string
		envValue  string
		wantError string
	}{
		{name: "reject localhost", host: "localhost", wantError: "generator.datasource.host_invalid"},
		{name: "reject loopback ip", host: "127.0.0.1", wantError: "generator.datasource.host_invalid"},
		{name: "reject private ip by default", host: "10.10.10.10", wantError: "generator.datasource.host_private_disabled"},
		{name: "reject invalid host chars", host: "db/internal", wantError: "generator.datasource.host_invalid"},
		{name: "allow private ip with env", host: "10.10.10.10", envValue: "true"},
		{name: "allow public hostname", host: "db.example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("PANTHEON_ALLOW_PRIVATE_GENERATOR_DATASOURCE", tt.envValue)
			_, err := normalizeDatasourceReq(&UpsertGeneratorDatasourceReq{
				Name:         "demo",
				Driver:       "mysql",
				Host:         tt.host,
				Port:         3306,
				DatabaseName: "demo",
				Username:     "root",
				Password:     "secret",
				Status:       generatorDatasourceEnabled,
			}, true)
			if tt.wantError == "" && err != nil {
				t.Fatalf("expected success, got %v", err)
			}
			if tt.wantError != "" {
				if err == nil || err.Error() != tt.wantError {
					t.Fatalf("expected %s, got %v", tt.wantError, err)
				}
			}
		})
	}
}
