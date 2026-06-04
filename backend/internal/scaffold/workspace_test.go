package scaffold

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateRegisterRequestHonorsScopeSpecificModuleNameRules(t *testing.T) {
	tests := []struct {
		name      string
		req       *RegisterGeneratedModuleRequest
		wantError string
	}{
		{
			name: "business scope allows nested path",
			req: &RegisterGeneratedModuleRequest{
				Schema: ModuleSchema{
					Name:        "cmdb/host",
					Scope:       "business",
					DisplayName: "主机管理",
					Model: struct {
						TableName string        `json:"tableName"`
						ModelName string        `json:"modelName"`
						Fields    []ModuleField `json:"fields"`
					}{
						TableName: "biz_cmdb_host",
					},
				},
				Files: []GeneratedFile{{Path: "backend/modules/business/cmdb/host/module.go", Content: "package host"}},
			},
		},
		{
			name: "system scope rejects nested path",
			req: &RegisterGeneratedModuleRequest{
				Schema: ModuleSchema{
					Name:        "config/audit",
					Scope:       "system",
					DisplayName: "审计配置",
					Model: struct {
						TableName string        `json:"tableName"`
						ModelName string        `json:"modelName"`
						Fields    []ModuleField `json:"fields"`
					}{
						TableName: "system_config_audit",
					},
				},
				Files: []GeneratedFile{{Path: "backend/modules/system/config/audit/module.go", Content: "package system"}},
			},
			wantError: "module.generate.invalid_name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateRegisterRequest(tt.req)
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

func TestValidateRegisterRequestRejectsUnsafeManagedTableName(t *testing.T) {
	req := &RegisterGeneratedModuleRequest{
		Schema: ModuleSchema{
			Name:        "asset",
			Scope:       "business",
			DisplayName: "资产管理",
			Model: struct {
				TableName string        `json:"tableName"`
				ModelName string        `json:"modelName"`
				Fields    []ModuleField `json:"fields"`
			}{
				TableName: "biz_asset;drop_table",
			},
		},
		Files: []GeneratedFile{{Path: "backend/modules/business/asset/module.go", Content: "package asset"}},
	}

	err := ValidateRegisterRequest(req)
	if err == nil || err.Error() != "module.generate.invalid_table_name" {
		t.Fatalf("expected invalid table name error, got %v", err)
	}
}

func TestValidateRegisterRequestRejectsInvalidGovernanceContract(t *testing.T) {
	tests := []struct {
		name      string
		mutate    func(*RegisterGeneratedModuleRequest)
		wantError string
	}{
		{
			name: "unsupported template version",
			mutate: func(req *RegisterGeneratedModuleRequest) {
				req.Schema.TemplateVersion = "v2"
			},
			wantError: "module.generate.invalid_template_version",
		},
		{
			name: "invalid data scope mode",
			mutate: func(req *RegisterGeneratedModuleRequest) {
				req.Schema.EnableDataScope = true
				req.Schema.DataScopeMode = "project"
			},
			wantError: "module.generate.invalid_data_scope",
		},
		{
			name: "module cannot depend on itself",
			mutate: func(req *RegisterGeneratedModuleRequest) {
				req.Schema.Dependencies = []ModuleDependency{{Module: "asset", Required: true}}
			},
			wantError: "module.generate.invalid_dependency",
		},
		{
			name: "duplicate dependency",
			mutate: func(req *RegisterGeneratedModuleRequest) {
				req.Schema.Dependencies = []ModuleDependency{
					{Module: "cmdb/vendor", Required: true},
					{Module: "cmdb/vendor", Required: true},
				}
			},
			wantError: "module.generate.invalid_dependency",
		},
		{
			name: "relation target module must be valid",
			mutate: func(req *RegisterGeneratedModuleRequest) {
				req.Schema.Relations = []ModuleRelation{{
					Name:         "assetOwner",
					Type:         "lookup",
					TargetModule: "CMDB/vendor",
					LocalField:   "vendorId",
					TargetField:  "id",
				}}
			},
			wantError: "module.generate.invalid_relation",
		},
		{
			name: "many to many relation requires junction table",
			mutate: func(req *RegisterGeneratedModuleRequest) {
				req.Schema.Relations = []ModuleRelation{{
					Name:         "assetGroups",
					Type:         "manyToMany",
					TargetModule: "cmdb/group",
					LocalField:   "id",
					TargetField:  "id",
				}}
			},
			wantError: "module.generate.invalid_relation",
		},
		{
			name: "lookup api must start with slash",
			mutate: func(req *RegisterGeneratedModuleRequest) {
				req.Schema.Relations = []ModuleRelation{{
					Name:             "assetOwner",
					Type:             "lookup",
					TargetModule:     "cmdb/vendor",
					LocalField:       "vendorId",
					TargetField:      "id",
					TargetLabelField: "name",
					LookupAPI:        "system/lookup/vendors",
					LookupValueField: "id",
				}}
			},
			wantError: "module.generate.invalid_relation",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := newScaffoldTestRequest()
			tt.mutate(req)

			err := ValidateRegisterRequest(req)
			if err == nil || err.Error() != tt.wantError {
				t.Fatalf("expected %s, got %v", tt.wantError, err)
			}
		})
	}
}

func TestValidateRegisterRequestAcceptsP2GovernanceContract(t *testing.T) {
	req := newScaffoldTestRequest()
	req.Schema.TemplateVersion = "v1"
	req.Schema.EnableDataScope = true
	req.Schema.DataScopeMode = "dept"
	req.Schema.Dependencies = []ModuleDependency{{Module: "cmdb/vendor", Required: true, Reason: "asset needs vendor"}}
	req.Schema.Relations = []ModuleRelation{
		{
			Name:             "assetVendor",
			Type:             "lookup",
			TargetModule:     "cmdb/vendor",
			LocalField:       "vendorId",
			TargetField:      "id",
			TargetLabelField: "name",
			LookupAPI:        "/business/cmdb/vendor/options",
			LookupValueField: "id",
		},
		{
			Name:          "assetGroups",
			Type:          "manyToMany",
			TargetModule:  "cmdb/group",
			LocalField:    "id",
			TargetField:   "id",
			JunctionTable: "biz_cmdb_asset_group",
		},
	}

	if err := ValidateRegisterRequest(req); err != nil {
		t.Fatalf("expected valid P2 governance contract, got %v", err)
	}
}

func TestWriteGeneratedFallbackResourcesBuildsGeneratedLocaleFiles(t *testing.T) {
	root := t.TempDir()
	schemaDir := filepath.Join(root, "schema", "generated", "business", "cmdb")
	if err := os.MkdirAll(schemaDir, 0o755); err != nil {
		t.Fatalf("mkdir schema dir: %v", err)
	}
	schemaContent := `{
  "name": "cmdb/host",
  "displayName": "主机管理",
  "scope": "business",
  "model": {
    "tableName": "biz_cmdb_host",
    "modelName": "CmdbHost",
    "fields": []
  },
  "i18n": {
    "namespace": "business.cmdb.host",
    "translations": {
      "zh": {
        "business.cmdb.host.title": "主机管理",
        "business.cmdb.host.permission.export": "导出主机管理"
      },
      "en": {
        "business.cmdb.host.title": "Host Management",
        "business.cmdb.host.permission.export": "Export Host Management"
      }
    }
  }
}`
	if err := os.WriteFile(filepath.Join(schemaDir, "host.json"), []byte(schemaContent), 0o644); err != nil {
		t.Fatalf("write schema: %v", err)
	}

	if err := WriteGeneratedFallbackResources(root); err != nil {
		t.Fatalf("write generated fallback resources: %v", err)
	}

	zhContent, err := os.ReadFile(filepath.Join(root, "frontend", "src", "i18n", "resources", "generated", "zh-CN.ts"))
	if err != nil {
		t.Fatalf("read zh generated file: %v", err)
	}
	if !strings.Contains(string(zhContent), `"business.cmdb.host.permission.export": "导出主机管理"`) {
		t.Fatalf("expected zh generated fallback to include host export permission, got %s", string(zhContent))
	}

	enContent, err := os.ReadFile(filepath.Join(root, "frontend", "src", "i18n", "resources", "generated", "en-US.ts"))
	if err != nil {
		t.Fatalf("read en generated file: %v", err)
	}
	if !strings.Contains(string(enContent), `"business.cmdb.host.title": "Host Management"`) {
		t.Fatalf("expected en generated fallback to include host title, got %s", string(enContent))
	}

	jaContent, err := os.ReadFile(filepath.Join(root, "frontend", "src", "i18n", "resources", "generated", "ja-JP.ts"))
	if err != nil {
		t.Fatalf("read ja generated file: %v", err)
	}
	if !strings.Contains(string(jaContent), `"business.cmdb.host.title": "Host Management"`) {
		t.Fatalf("expected ja generated fallback to include English host title, got %s", string(jaContent))
	}
}

func TestWriteGeneratedModuleSourceBuildsFilesServerSideWhenFilesOmitted(t *testing.T) {
	root := prepareScaffoldWorkspaceRoot(t)
	scriptDir := filepath.Join(root, "frontend", "scripts")
	if err := os.MkdirAll(scriptDir, 0o755); err != nil {
		t.Fatalf("mkdir script dir: %v", err)
	}
	script := `import { readFileSync } from 'node:fs';
const schema = JSON.parse(readFileSync(process.argv[2], 'utf8'));
const files = [
  {
    path: 'backend/modules/business/asset/module.go',
    content: 'package asset\n',
    language: 'go',
  },
  {
    path: 'frontend/src/modules/business/asset/index.ts',
    content: 'export const AssetModule = {}\n',
    language: 'typescript',
  },
];
process.stdout.write(JSON.stringify(files));
`
	if err := os.WriteFile(filepath.Join(scriptDir, "export-generated-module.mjs"), []byte(script), 0o644); err != nil {
		t.Fatalf("write script: %v", err)
	}

	req := newScaffoldTestRequest()
	req.Files = nil

	written, err := WriteGeneratedModuleSource(root, req)
	if err != nil {
		t.Fatalf("write generated source with server-side generation: %v", err)
	}

	if len(written) != 3 {
		t.Fatalf("expected 3 written artifacts, got %d (%v)", len(written), written)
	}
	if !fileExists(filepath.Join(root, "backend", "modules", "business", "asset", "module.go")) {
		t.Fatal("expected backend module file to be generated")
	}
	if !fileExists(filepath.Join(root, "frontend", "src", "modules", "business", "asset", "index.ts")) {
		t.Fatal("expected frontend module file to be generated")
	}
	if !fileExists(filepath.Join(root, "schema", "generated", "business", "asset.json")) {
		t.Fatal("expected schema file to be written")
	}
}

func TestWriteGeneratedModuleSourceRejectsEscapingGeneratedFilePath(t *testing.T) {
	root := prepareScaffoldWorkspaceRoot(t)
	req := newScaffoldTestRequest()
	req.Files = []GeneratedFile{{
		Path:    "backend/modules/business/asset/../../outside.go",
		Content: "package outside\n",
	}}

	if _, err := WriteGeneratedModuleSource(root, req); err == nil || err.Error() != "module.generate.invalid_path" {
		t.Fatalf("expected invalid path error, got %v", err)
	}
}

func newScaffoldTestRequest() *RegisterGeneratedModuleRequest {
	req := &RegisterGeneratedModuleRequest{
		Schema: ModuleSchema{
			Name:        "asset",
			Scope:       "business",
			DisplayName: "资产管理",
			Model: struct {
				TableName string        `json:"tableName"`
				ModelName string        `json:"modelName"`
				Fields    []ModuleField `json:"fields"`
			}{
				TableName: "biz_asset",
			},
		},
		Files: []GeneratedFile{{Path: "backend/modules/business/asset/module.go", Content: "package asset"}},
	}
	return req
}

func TestResolveWorkspaceRootHonorsConfiguredEnvWhenStartIsEmpty(t *testing.T) {
	root := prepareScaffoldWorkspaceRoot(t)
	t.Setenv(workspaceRootEnvKey, root)

	resolved, err := ResolveWorkspaceRoot("")
	if err != nil {
		t.Fatalf("resolve workspace root from env: %v", err)
	}
	if resolved != root {
		t.Fatalf("expected workspace root %s, got %s", root, resolved)
	}
}

func TestResolveWorkspaceRootIgnoresEnvWhenExplicitStartProvided(t *testing.T) {
	envRoot := prepareScaffoldWorkspaceRoot(t)
	startRoot := prepareScaffoldWorkspaceRoot(t)
	nestedStart := filepath.Join(startRoot, "backend", "modules")
	if err := os.MkdirAll(nestedStart, 0o755); err != nil {
		t.Fatalf("mkdir nested start: %v", err)
	}
	t.Setenv(workspaceRootEnvKey, envRoot)

	resolved, err := ResolveWorkspaceRoot(nestedStart)
	if err != nil {
		t.Fatalf("resolve workspace root from explicit start: %v", err)
	}
	if resolved != startRoot {
		t.Fatalf("expected explicit start root %s, got %s", startRoot, resolved)
	}
}

func TestResolveWorkspaceRootFallsBackToSourceTreeWhenCwdIsNotWorkspace(t *testing.T) {
	originalWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	tempWd := t.TempDir()
	if err := os.Chdir(tempWd); err != nil {
		t.Fatalf("chdir temp: %v", err)
	}
	defer func() {
		_ = os.Chdir(originalWd)
	}()

	resolved, err := ResolveWorkspaceRoot("")
	if err != nil {
		t.Fatalf("resolve workspace root from source fallback: %v", err)
	}
	resolvedBase := filepath.Base(filepath.Clean(resolved))
	if !strings.Contains(resolvedBase, "pantheon-base") {
		t.Fatalf("expected source-tree fallback under pantheon-base workspace, got %s", resolved)
	}
}

func prepareScaffoldWorkspaceRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module pantheon-platform\n\ngo 1.25.4\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "backend"), 0o755); err != nil {
		t.Fatalf("mkdir backend: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "frontend"), 0o755); err != nil {
		t.Fatalf("mkdir frontend: %v", err)
	}
	return root
}
