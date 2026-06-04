package generator

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"pantheon-ops/backend/internal/scaffold"
)

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
		wantError string
	}{
		{name: "reject localhost", host: "localhost", wantError: "generator.datasource.host_invalid"},
		{name: "reject loopback ip", host: "127.0.0.1", wantError: "generator.datasource.host_invalid"},
		{name: "allow private ip", host: "10.10.10.10"},
		{name: "allow internal hostname", host: "db.internal"},
		{name: "reject invalid host chars", host: "db/internal", wantError: "generator.datasource.host_invalid"},
		{name: "allow public hostname", host: "db.example.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

func TestBuildGeneratedModuleArchiveUsesServerSideFiles(t *testing.T) {
	root := t.TempDir()
	for _, dir := range []string{"backend", "frontend"} {
		if err := os.MkdirAll(filepath.Join(root, dir), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", dir, err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module pantheon-ops\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	scriptDir := filepath.Join(root, "frontend", "scripts")
	if err := os.MkdirAll(scriptDir, 0o755); err != nil {
		t.Fatalf("mkdir script dir: %v", err)
	}
	script := `const files = [
  {
    path: 'backend/modules/business/cmdb/host/module.go',
    content: 'package host\n',
    language: 'go',
  },
  {
    path: 'frontend/src/modules/business/cmdb/host/index.ts',
    content: 'export const CmdbHostModule = {}\n',
    language: 'typescript',
  },
];
process.stdout.write(JSON.stringify(files));
`
	if err := os.WriteFile(filepath.Join(scriptDir, "export-generated-module.mjs"), []byte(script), 0o644); err != nil {
		t.Fatalf("write exporter script: %v", err)
	}

	service := &GeneratorService{workspaceRoot: root}
	archive, filename, err := service.BuildGeneratedModuleArchive(&scaffold.ModuleSchema{
		Name:        "cmdb/host",
		Scope:       "business",
		DisplayName: "主机管理",
		Model: struct {
			TableName string                 `json:"tableName"`
			ModelName string                 `json:"modelName"`
			Fields    []scaffold.ModuleField `json:"fields"`
		}{
			TableName: "biz_cmdb_host",
		},
	})
	if err != nil {
		t.Fatalf("build generated archive: %v", err)
	}
	if filename != "cmdb-host-module.zip" {
		t.Fatalf("unexpected filename: %s", filename)
	}

	reader, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		t.Fatalf("open generated archive: %v", err)
	}
	if len(reader.File) != 2 {
		t.Fatalf("expected 2 files in archive, got %d", len(reader.File))
	}
	paths := map[string]struct{}{}
	for _, file := range reader.File {
		paths[file.Name] = struct{}{}
	}
	for _, path := range []string{
		"backend/modules/business/cmdb/host/module.go",
		"frontend/src/modules/business/cmdb/host/index.ts",
	} {
		if _, ok := paths[path]; !ok {
			t.Fatalf("expected archive to contain %s, got %#v", path, paths)
		}
	}
}
