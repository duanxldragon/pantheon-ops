package dynamicmodule

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"pantheon-ops/backend/internal/scaffold"
	"pantheon-ops/backend/pkg/testmysql"

	"gorm.io/gorm"
)

func openDynamicModuleTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db := testmysql.Open(t)
	if err := db.AutoMigrate(&ModuleRegistration{}); err != nil {
		t.Fatalf("migrate module registration: %v", err)
	}
	return db
}

func prepareDynamicModuleWorkspace(t *testing.T) string {
	t.Helper()

	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, "go.mod"), "module pantheon-ops\n\ngo 1.25.4\n")
	mustMkdirAll(t, filepath.Join(root, "backend", "modules", "business"))
	mustMkdirAll(t, filepath.Join(root, "backend", "modules", "system", "iam", "menu"))
	mustMkdirAll(t, filepath.Join(root, "frontend", "scripts"))
	mustMkdirAll(t, filepath.Join(root, "frontend", "src", "modules", "generated"))
	mustMkdirAll(t, filepath.Join(root, "frontend", "src", "core", "router"))
	mustMkdirAll(t, filepath.Join(root, "schema", "generated", "business"))
	mustWriteFile(t, filepath.Join(root, "frontend", "scripts", "export-generated-module.mjs"), `import { readFileSync } from 'node:fs';

function pascalCase(value) {
  return String(value || '')
    .split(/[_\-/\s]+/)
    .filter(Boolean)
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1).toLowerCase())
    .join('');
}

const schema = JSON.parse(readFileSync(process.argv[2], 'utf8'));
const scope = schema.scope;
const name = schema.name;
const leafName = name.split('/').filter(Boolean).at(-1) || 'module';
const modelName = pascalCase(name);
const files = [
  {
    path: 'backend/modules/' + scope + '/' + name + '/module.go',
    content: 'package ' + leafName + '\n',
    language: 'go',
  },
  {
    path: 'frontend/src/modules/' + scope + '/' + name + '/index.ts',
    content: 'export const ' + modelName + 'Module = {}\n',
    language: 'typescript',
  },
  {
    path: 'frontend/src/modules/' + scope + '/' + name + '/' + modelName + 'List.tsx',
    content: 'export default function ' + modelName + 'List() { return null; }\n',
    language: 'tsx',
  },
  {
    path: 'frontend/src/modules/' + scope + '/' + name + '/' + modelName + 'Detail.tsx',
    content: 'export default function ' + modelName + 'Detail() { return null; }\n',
    language: 'tsx',
  },
];
process.stdout.write(JSON.stringify(files));
`)
	return root
}

func mustWriteGeneratedRegistryStubs(t *testing.T, root string) {
	t.Helper()
	mustWriteFile(t, filepath.Join(root, "backend", "modules", "business", "generated_registry.go"), "package business\n")
	mustWriteFile(t, filepath.Join(root, "backend", "modules", "system", "generated_registry.go"), "package system\n")
	mustWriteFile(t, filepath.Join(root, "backend", "modules", "system", "iam", "menu", "generated_component_registry.go"), "package iam\n")
	mustWriteFile(t, filepath.Join(root, "frontend", "src", "modules", "generated", "business.ts"), "export const generatedBusinessModules = []\n")
	mustWriteFile(t, filepath.Join(root, "frontend", "src", "modules", "generated", "system.ts"), "export const generatedSystemModules = []\n")
	mustWriteFile(t, filepath.Join(root, "frontend", "src", "core", "router", "generatedComponentRegistry.ts"), "export const generatedComponentRegistry = {}\n")
}

func newGeneratedModuleRequest(scope string, name string, displayName string, tableName string) *scaffold.RegisterGeneratedModuleRequest {
	req := &scaffold.RegisterGeneratedModuleRequest{}
	req.Schema.Name = name
	req.Schema.Scope = scope
	req.Schema.DisplayName = displayName
	req.Schema.Model.TableName = tableName
	return req
}

func mustMkdirAll(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", dir, err)
	}
}

func mustWriteFile(t *testing.T, target string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatalf("mkdir for %s: %v", target, err)
	}
	if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", target, err)
	}
}

func assertFileContains(t *testing.T, target string, fragment string) {
	t.Helper()
	content, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read %s: %v", target, err)
	}
	if !strings.Contains(string(content), fragment) {
		t.Fatalf("expected %s to contain %q, got:\n%s", target, fragment, string(content))
	}
}

func assertFileNotContains(t *testing.T, target string, fragment string) {
	t.Helper()
	content, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read %s: %v", target, err)
	}
	if strings.Contains(string(content), fragment) {
		t.Fatalf("expected %s not to contain %q, got:\n%s", target, fragment, string(content))
	}
}

func assertHasVerification(t *testing.T, items []GeneratedModuleVerification, code string, status string) {
	t.Helper()
	for _, item := range items {
		if item.Code == code && item.Status == status {
			return
		}
	}
	t.Fatalf("expected verification %s with status %s, got %#v", code, status, items)
}
