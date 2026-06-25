package scaffold

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"text/template"
	"unicode"
)

var managedTableNamePattern = regexp.MustCompile(`^[a-z0-9_]+$`)

const workspaceRootEnvKey = "PANTHEON_WORKSPACE_ROOT"
const generatedModuleExporterScript = "frontend/scripts/export-generated-module.mjs"
const GeneratedFeatureLedgerRelativePath = "schema/generated/feature-ledger.json"

func isWorkspaceRoot(candidate string) bool {
	return fileExists(filepath.Join(candidate, "go.mod")) &&
		dirExists(filepath.Join(candidate, "backend")) &&
		dirExists(filepath.Join(candidate, "frontend"))
}

func ResolveWorkspaceRoot(start string) (string, error) {
	current := strings.TrimSpace(start)
	if current == "" {
		if configuredRoot := strings.TrimSpace(os.Getenv(workspaceRootEnvKey)); configuredRoot != "" {
			resolved, err := filepath.Abs(configuredRoot)
			if err != nil {
				return "", err
			}
			if !isWorkspaceRoot(resolved) {
				return "", errors.New("workspace.not_found")
			}
			return resolved, nil
		}

		var err error
		current, err = os.Getwd()
		if err != nil {
			return "", err
		}
	}
	current, _ = filepath.Abs(current)

	for {
		if isWorkspaceRoot(current) {
			return current, nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}

	return "", errors.New("workspace.not_found")
}

func ValidateRegisterRequest(req *RegisterGeneratedModuleRequest) error {
	if req == nil {
		return errors.New("module.generate.invalid_payload")
	}
	name := strings.TrimSpace(req.Schema.Name)
	scope := strings.TrimSpace(req.Schema.Scope)
	displayName := strings.TrimSpace(req.Schema.DisplayName)
	tableName := strings.TrimSpace(req.Schema.Model.TableName)
	if !isValidModulePath(name, scope == "business") {
		return errors.New("module.generate.invalid_name")
	}
	if scope != "system" && scope != "business" {
		return errors.New("module.generate.invalid_scope")
	}
	if displayName == "" {
		return errors.New("module.generate.display_name_required")
	}
	if tableName == "" {
		return errors.New("module.generate.table_name_required")
	}
	if err := ValidateManagedTableName(scope, tableName); err != nil {
		return errors.New("module.generate.invalid_table_name")
	}
	if err := validateGovernanceContract(req); err != nil {
		return err
	}
	return nil
}

func ValidateManagedTableName(scope string, tableName string) error {
	normalizedScope := strings.TrimSpace(scope)
	normalizedTableName := strings.TrimSpace(tableName)
	if normalizedTableName == "" {
		return errors.New("module.generate.invalid_table_name")
	}
	if !managedTableNamePattern.MatchString(normalizedTableName) {
		return errors.New("module.generate.invalid_table_name")
	}
	if strings.Contains(normalizedTableName, "__") {
		return errors.New("module.generate.invalid_table_name")
	}
	switch normalizedScope {
	case "system":
		if !strings.HasPrefix(normalizedTableName, "system_") {
			return errors.New("module.generate.invalid_table_name")
		}
	case "business":
		if !strings.HasPrefix(normalizedTableName, "biz_") {
			return errors.New("module.generate.invalid_table_name")
		}
	default:
		return errors.New("module.generate.invalid_table_name")
	}
	return nil
}

func WriteGeneratedModuleSource(workspaceRoot string, req *RegisterGeneratedModuleRequest) ([]string, error) {
	if err := ValidateRegisterRequest(req); err != nil {
		return nil, err
	}

	name := strings.TrimSpace(req.Schema.Name)
	scope := strings.TrimSpace(req.Schema.Scope)
	backendPrefix := filepath.ToSlash(filepath.Join("backend", "modules", scope, name)) + "/"
	frontendPrefix := filepath.ToSlash(filepath.Join("frontend", "src", "modules", scope, name)) + "/"
	files := req.Files
	if len(files) == 0 {
		generatedFiles, err := GenerateModuleFilesFromSchema(workspaceRoot, req.Schema)
		if err != nil {
			return nil, err
		}
		files = generatedFiles
		req.Files = generatedFiles
	}

	written := make([]string, 0, len(files)+1)
	seen := make(map[string]struct{}, len(files))
	workspaceAbs, err := filepath.Abs(workspaceRoot)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		relativePath := normalizeRelativePath(file.Path)
		if relativePath == "" || relativePath == "." || isRelativeEscapePath(relativePath) {
			return nil, errors.New("module.generate.invalid_path")
		}
		if !strings.HasPrefix(relativePath, backendPrefix) && !strings.HasPrefix(relativePath, frontendPrefix) {
			return nil, errors.New("module.generate.invalid_path")
		}
		if _, ok := seen[relativePath]; ok {
			return nil, errors.New("module.generate.duplicate_file")
		}
		seen[relativePath] = struct{}{}

		absolutePath, err := resolveWorkspacePath(workspaceAbs, relativePath)
		if err != nil {
			return nil, errors.New("module.generate.invalid_path")
		}
		if err := os.MkdirAll(filepath.Dir(absolutePath), 0o755); err != nil {
			return nil, err
		}
		if !req.Overwrite && fileExists(absolutePath) {
			return nil, errors.New("module.generate.file_exists")
		}
		if err := os.WriteFile(absolutePath, []byte(file.Content), 0o644); err != nil {
			return nil, err
		}
		written = append(written, relativePath)
	}

	schemaPath, err := resolveWorkspacePath(workspaceAbs, filepath.ToSlash(filepath.Join("schema", "generated", scope, name+".json")))
	if err != nil {
		return nil, errors.New("module.generate.invalid_path")
	}
	if err := os.MkdirAll(filepath.Dir(schemaPath), 0o755); err != nil {
		return nil, err
	}
	schemaJSON, err := json.MarshalIndent(req.Schema, "", "  ")
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(schemaPath, schemaJSON, 0o644); err != nil {
		return nil, err
	}
	written = append(written, filepath.ToSlash(filepath.Join("schema", "generated", scope, name+".json")))
	if err := WriteGeneratedFallbackResources(workspaceRoot); err != nil {
		return nil, err
	}

	return written, nil
}

func GenerateModuleFilesFromSchema(workspaceRoot string, schema ModuleSchema) ([]GeneratedFile, error) {
	scriptPath := filepath.Join(workspaceRoot, filepath.FromSlash(generatedModuleExporterScript))
	if !fileExists(scriptPath) {
		return nil, errors.New("module.generate.server_export_failed")
	}

	schemaFile, err := os.CreateTemp("", "pantheon-module-schema-*.json")
	if err != nil {
		return nil, err
	}
	schemaPath := schemaFile.Name()
	defer os.Remove(schemaPath)

	schemaJSON, err := json.Marshal(schema)
	if err != nil {
		_ = schemaFile.Close()
		return nil, err
	}
	if _, err := schemaFile.Write(schemaJSON); err != nil {
		_ = schemaFile.Close()
		return nil, err
	}
	if err := schemaFile.Close(); err != nil {
		return nil, err
	}

	cmd := exec.Command("node", scriptPath, schemaPath)
	cmd.Dir = workspaceRoot
	output, err := cmd.Output()
	if err != nil {
		return nil, errors.New("module.generate.server_export_failed")
	}

	var files []GeneratedFile
	if err := json.Unmarshal(output, &files); err != nil {
		return nil, errors.New("module.generate.server_export_failed")
	}
	if len(files) == 0 {
		return nil, errors.New("module.generate.server_export_failed")
	}
	return files, nil
}

func RemoveGeneratedModuleSource(workspaceRoot string, scope string, name string) error {
	if !isValidModulePath(name, scope == "business") {
		return errors.New("module.generate.invalid_name")
	}
	if scope != "system" && scope != "business" {
		return errors.New("module.generate.invalid_scope")
	}

	targets := []string{
		filepath.Join(workspaceRoot, "backend", "modules", scope, name),
		filepath.Join(workspaceRoot, "frontend", "src", "modules", scope, name),
		filepath.Join(workspaceRoot, "schema", "generated", scope, name+".json"),
	}
	workspaceAbs, err := filepath.Abs(workspaceRoot)
	if err != nil {
		return err
	}

	for _, target := range targets {
		if !pathWithinRoot(workspaceAbs, target) {
			return errors.New("module.generate.invalid_path")
		}
		if err := os.RemoveAll(target); err != nil {
			return err
		}
	}
	return WriteGeneratedFallbackResources(workspaceRoot)
}

func WriteGeneratedRegistries(workspaceRoot string, refs []GeneratedModuleRef) error {
	normalized := normalizeGeneratedModuleRefs(refs)
	if err := writeGeneratedBackendRegistry(workspaceRoot, "business", filterGeneratedModuleRefs(normalized, "business")); err != nil {
		return err
	}
	if err := writeGeneratedBackendRegistry(workspaceRoot, "system", filterGeneratedModuleRefs(normalized, "system")); err != nil {
		return err
	}
	if err := writeGeneratedFrontendModuleRegistry(workspaceRoot, "business", filterGeneratedModuleRefs(normalized, "business")); err != nil {
		return err
	}
	if err := writeGeneratedFrontendModuleRegistry(workspaceRoot, "system", filterGeneratedModuleRefs(normalized, "system")); err != nil {
		return err
	}
	if err := writeGeneratedFrontendComponentRegistry(workspaceRoot, normalized); err != nil {
		return err
	}
	return writeGeneratedBackendComponentRegistry(workspaceRoot, normalized)
}

func normalizeGeneratedModuleRefs(refs []GeneratedModuleRef) []GeneratedModuleRef {
	seen := make(map[string]struct{}, len(refs))
	normalized := make([]GeneratedModuleRef, 0, len(refs))
	for _, ref := range refs {
		name := strings.TrimSpace(ref.Name)
		scope := strings.TrimSpace(ref.Scope)
		if !isValidModulePath(name, scope == "business") || (scope != "system" && scope != "business") {
			continue
		}
		key := scope + ":" + name
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		normalized = append(normalized, GeneratedModuleRef{Name: name, Scope: scope})
	}
	sort.Slice(normalized, func(i, j int) bool {
		if normalized[i].Scope == normalized[j].Scope {
			return normalized[i].Name < normalized[j].Name
		}
		return normalized[i].Scope < normalized[j].Scope
	})
	return normalized
}

func filterGeneratedModuleRefs(refs []GeneratedModuleRef, scope string) []GeneratedModuleRef {
	filtered := make([]GeneratedModuleRef, 0, len(refs))
	for _, ref := range refs {
		if ref.Scope == scope {
			filtered = append(filtered, ref)
		}
	}
	return filtered
}

func writeGeneratedBackendRegistry(workspaceRoot string, scope string, refs []GeneratedModuleRef) error {
	target := filepath.Join(workspaceRoot, "backend", "modules", scope, "generated_registry.go")

	type entry struct {
		Alias      string
		ImportPath string
		InitFunc   string
	}
	entries := make([]entry, 0, len(refs))
	for _, ref := range refs {
		name := ref.Name
		entries = append(entries, entry{
			Alias:      safeIdentifier(name),
			ImportPath: fmt.Sprintf("pantheon-ops/backend/modules/%s/%s", scope, name),
			InitFunc:   "Init" + toPascal(name) + "Module",
		})
	}

	data := struct {
		Package    string
		ScopeTitle string
		Entries    []entry
	}{
		Package:    scope,
		ScopeTitle: toPascal(scope),
		Entries:    entries,
	}

	return writeTemplateFile(target, generatedBackendRegistryTemplate, data)
}

func writeGeneratedFrontendModuleRegistry(workspaceRoot string, scope string, refs []GeneratedModuleRef) error {
	target := filepath.Join(workspaceRoot, "frontend", "src", "modules", "lowcode", "generated", scope+".ts")

	type entry struct {
		ImportPath string
		ExportName string
	}
	entries := make([]entry, 0, len(refs))
	for _, ref := range refs {
		entries = append(entries, entry{
			ImportPath: fmt.Sprintf("../%s/%s", scope, ref.Name),
			ExportName: toPascal(ref.Name) + "Module",
		})
	}

	data := struct {
		Entries    []entry
		ScopeTitle string
	}{
		Entries:    entries,
		ScopeTitle: toPascal(scope),
	}

	return writeTemplateFile(target, generatedFrontendModuleRegistryTemplate, data)
}

func writeGeneratedFrontendComponentRegistry(workspaceRoot string, refs []GeneratedModuleRef) error {
	target := filepath.Join(workspaceRoot, "frontend", "src", "core", "router", "generatedComponentRegistry.ts")

	type entry struct {
		Key        string
		ImportPath string
	}
	entries := make([]entry, 0, len(refs))
	for _, ref := range refs {
		listComponentName := toPascal(ref.Name) + "List"
		detailComponentName := toPascal(ref.Name) + "Detail"
		entries = append(entries,
			entry{
				Key:        fmt.Sprintf("%s/%s/%s", ref.Scope, ref.Name, listComponentName),
				ImportPath: fmt.Sprintf("../../modules/%s/%s/%s", ref.Scope, ref.Name, listComponentName),
			},
			entry{
				Key:        fmt.Sprintf("%s/%s/%s", ref.Scope, ref.Name, detailComponentName),
				ImportPath: fmt.Sprintf("../../modules/%s/%s/%s", ref.Scope, ref.Name, detailComponentName),
			},
		)
	}

	data := struct {
		Entries []entry
	}{
		Entries: entries,
	}

	return writeTemplateFile(target, generatedFrontendComponentRegistryTemplate, data)
}

func writeGeneratedBackendComponentRegistry(workspaceRoot string, refs []GeneratedModuleRef) error {
	target := filepath.Join(workspaceRoot, "backend", "modules", "system", "iam", "menu", "generated_component_registry.go")

	type entry struct {
		Key string
	}
	entries := make([]entry, 0, len(refs))
	for _, ref := range refs {
		entries = append(entries,
			entry{Key: fmt.Sprintf("%s/%s/%sList", ref.Scope, ref.Name, toPascal(ref.Name))},
			entry{Key: fmt.Sprintf("%s/%s/%sDetail", ref.Scope, ref.Name, toPascal(ref.Name))},
		)
	}

	data := struct {
		Entries []entry
	}{
		Entries: entries,
	}

	return writeTemplateFile(target, generatedBackendComponentRegistryTemplate, data)
}

func writeTemplateFile(target string, templateSource string, data any) error {
	tpl, err := template.New(filepath.Base(target)).Parse(templateSource)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	file, err := os.Create(target)
	if err != nil {
		return err
	}
	defer file.Close()
	return tpl.Execute(file, data)
}

func isValidModulePath(name string, allowNested bool) bool {
	normalized := strings.TrimSpace(name)
	if normalized == "" {
		return false
	}
	segments := strings.Split(normalized, "/")
	if !allowNested && len(segments) != 1 {
		return false
	}
	for _, segment := range segments {
		if segment == "" {
			return false
		}
		for index, char := range segment {
			if index == 0 {
				if !unicode.IsLower(char) {
					return false
				}
				continue
			}
			if !(unicode.IsLower(char) || unicode.IsDigit(char) || char == '_') {
				return false
			}
		}
	}
	return true
}

func toPascal(value string) string {
	if value == "" {
		return ""
	}
	parts := strings.FieldsFunc(value, func(char rune) bool {
		return char == '_' || char == '-' || char == ' ' || char == '/'
	})
	builder := strings.Builder{}
	for _, part := range parts {
		runes := []rune(part)
		if len(runes) == 0 {
			continue
		}
		runes[0] = unicode.ToUpper(runes[0])
		for index := 1; index < len(runes); index++ {
			runes[index] = unicode.ToLower(runes[index])
		}
		builder.WriteString(string(runes))
	}
	return builder.String()
}

func safeIdentifier(value string) string {
	identifier := strings.ReplaceAll(strings.TrimSpace(value), "-", "_")
	identifier = strings.ReplaceAll(identifier, " ", "_")
	identifier = strings.ReplaceAll(identifier, "/", "_")
	if identifier == "" {
		return "generatedmodule"
	}
	if unicode.IsDigit([]rune(identifier)[0]) {
		return "m_" + identifier
	}
	return identifier
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func writeJSONFile(target string, payload any) error {
	serialized, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	return os.WriteFile(target, serialized, 0o644)
}

func WriteGeneratedFeatureLedgerSnapshot(workspaceRoot string, snapshot any) error {
	target := filepath.Join(workspaceRoot, filepath.FromSlash(GeneratedFeatureLedgerRelativePath))
	return writeJSONFile(target, snapshot)
}

func normalizeRelativePath(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return ""
	}
	return filepath.ToSlash(filepath.Clean(filepath.FromSlash(trimmed)))
}

func isRelativeEscapePath(value string) bool {
	return value == ".." || strings.HasPrefix(value, "../")
}

func pathWithinRoot(root string, target string) bool {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return false
	}
	targetAbs, err := filepath.Abs(target)
	if err != nil {
		return false
	}
	relative, err := filepath.Rel(rootAbs, targetAbs)
	if err != nil {
		return false
	}
	return relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func resolveWorkspacePath(workspaceRoot string, relativePath string) (string, error) {
	absolutePath := filepath.Join(workspaceRoot, filepath.FromSlash(relativePath))
	if !pathWithinRoot(workspaceRoot, absolutePath) {
		return "", errors.New("module.generate.invalid_path")
	}
	return absolutePath, nil
}

func WriteGeneratedFallbackResources(workspaceRoot string) error {
	schemaRoot := filepath.Join(workspaceRoot, "schema", "generated")
	localePayload := map[string]map[string]string{
		"zh-CN": {},
		"en-US": {},
		"ja-JP": {},
		"ko-KR": {},
		"fr-FR": {},
	}

	if dirExists(schemaRoot) {
		walkErr := filepath.WalkDir(schemaRoot, func(path string, d os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if d.IsDir() || !strings.EqualFold(filepath.Ext(path), ".json") {
				return nil
			}

			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}

			var schema ModuleSchema
			if err := json.Unmarshal(content, &schema); err != nil {
				return nil
			}
			for key, value := range schema.I18n.Translations.Zh {
				if strings.TrimSpace(key) == "" || strings.TrimSpace(value) == "" {
					continue
				}
				localePayload["zh-CN"][key] = value
			}
			for key, value := range schema.I18n.Translations.En {
				if strings.TrimSpace(key) == "" || strings.TrimSpace(value) == "" {
					continue
				}
				localePayload["en-US"][key] = value
			}
			return nil
		})
		if walkErr != nil {
			return walkErr
		}
	}
	for _, locale := range []string{"ja-JP", "ko-KR", "fr-FR"} {
		for key, value := range localePayload["en-US"] {
			if strings.TrimSpace(localePayload[locale][key]) == "" {
				localePayload[locale][key] = value
			}
		}
	}

	resourceDir := filepath.Join(workspaceRoot, "frontend", "src", "i18n", "resources", "generated")
	if err := os.MkdirAll(resourceDir, 0o755); err != nil {
		return err
	}
	for locale, payload := range localePayload {
		serialized, err := json.MarshalIndent(payload, "", "  ")
		if err != nil {
			return err
		}
		identifier := generatedLocaleIdentifier(locale)
		content := fmt.Sprintf("const %s = %s;\n\nexport default %s;\n", identifier, string(serialized), identifier)
		target := filepath.Join(resourceDir, locale+".ts")
		if err := os.WriteFile(target, []byte(content), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func generatedLocaleIdentifier(locale string) string {
	cleaned := strings.NewReplacer("-", "", "_", "", ".", "").Replace(locale)
	if cleaned == "" {
		cleaned = "Locale"
	}
	return "generated" + cleaned + "Fallback"
}

const generatedBackendRegistryTemplate = `package {{.Package}}

import (
{{- if .Entries }}
{{- range .Entries }}
	{{ .Alias }} "{{ .ImportPath }}"
{{- end }}
{{- end }}
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func InitGenerated{{ .ScopeTitle }}Modules(r *gin.RouterGroup, db *gorm.DB) {
{{- if .Entries }}
{{- range .Entries }}
	{{ .Alias }}.{{ .InitFunc }}(r, db)
{{- end }}
{{- else }}
	_ = r
	_ = db
{{- end }}
}
`

const generatedFrontendModuleRegistryTemplate = `import type { ModuleConfig } from '../../core/router/types';
{{- if .Entries }}
{{- range .Entries }}
import { {{ .ExportName }} } from '{{ .ImportPath }}';
{{- end }}
{{ end }}

export const generated{{ .ScopeTitle }}Modules: ModuleConfig[] = [
{{- range .Entries }}
  {{ .ExportName }},
{{- end }}
];
`

const generatedFrontendComponentRegistryTemplate = `{{- if .Entries }}import { lazy, type LazyExoticComponent, type ComponentType } from 'react';

type ComponentLoader = () => Promise<{ default: ComponentType }>;

interface RegistryEntry {
	component: LazyExoticComponent<ComponentType>;
	preload: ComponentLoader;
}

function defineRegistryEntry(loader: ComponentLoader): RegistryEntry {
	return {
		component: lazy(loader),
		preload: loader,
	};
}

export const generatedComponentRegistry = {
{{- range .Entries }}
  '{{ .Key }}': defineRegistryEntry(() => import('{{ .ImportPath }}')),
{{- end }}
} satisfies Record<string, RegistryEntry>;
{{- else }}export const generatedComponentRegistry = {};
{{- end }}
`

const generatedBackendComponentRegistryTemplate = `package iam

var generatedMenuComponentKeys = map[string]struct{}{
{{- range .Entries }}
	"{{ .Key }}": {},
{{- end }}
}
`
