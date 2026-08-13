package scaffold

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"pantheon-base/pkg/common"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"text/template"
	"unicode"
)

var managedTableNamePattern = regexp.MustCompile(`^[a-z0-9_]+$`)

const workspaceRootEnvKey = "PANTHEON_WORKSPACE_ROOT"
const nodeBinaryEnvKey = "PANTHEON_NODE_BIN"
const generatedModuleExporterScript = "frontend/scripts/export-generated-module.mjs"
const GeneratedFeatureLedgerRelativePath = "schema/generated/feature-ledger.json"

const msgInvalidTableName = "module.generate.invalid_table_name"
const msgInvalidPath = "module.generate.invalid_path"
const msgServerExportFailed = "module.generate.server_export_failed"

func isWorkspaceRoot(candidate string) bool {
	return fileExists(filepath.Join(candidate, "backend", "go.mod")) &&
		dirExists(filepath.Join(candidate, "backend")) &&
		dirExists(filepath.Join(candidate, "frontend"))
}

func ResolveWorkspaceRoot(start string) (string, error) {
	current := strings.TrimSpace(start)
	if current == "" {
		if resolved, err, handled := tryResolveWorkspaceRootFromEnv(); handled {
			return resolved, err
		}
		cwd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		current = cwd
	}
	current, _ = filepath.Abs(current)

	if found, ok := findWorkspaceRootUpward(current); ok {
		return found, nil
	}
	if found, ok := findWorkspaceRootFromCaller(); ok {
		return found, nil
	}
	return "", common.NewNotFound("workspace.not_found")
}

func tryResolveWorkspaceRootFromEnv() (string, error, bool) {
	configuredRoot := strings.TrimSpace(os.Getenv(workspaceRootEnvKey))
	if configuredRoot == "" {
		return "", nil, false
	}
	resolved, err := filepath.Abs(configuredRoot)
	if err != nil {
		return "", err, true
	}
	if !isWorkspaceRoot(resolved) {
		return "", common.NewNotFound("workspace.not_found"), true
	}
	return resolved, nil, true
}

func findWorkspaceRootUpward(start string) (string, bool) {
	current := start
	for {
		if isWorkspaceRoot(current) {
			return current, true
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", false
		}
		current = parent
	}
}

func findWorkspaceRootFromCaller() (string, bool) {
	_, sourceFile, _, ok := runtime.Caller(0)
	if !ok {
		return "", false
	}
	return findWorkspaceRootUpward(filepath.Dir(sourceFile))
}

func ValidateRegisterRequest(req *RegisterGeneratedModuleRequest) error {
	if req == nil {
		return common.NewBadRequest("module.generate.invalid_payload")
	}
	name := strings.TrimSpace(req.Schema.Name)
	scope := strings.TrimSpace(req.Schema.Scope)
	displayName := strings.TrimSpace(req.Schema.DisplayName)
	tableName := strings.TrimSpace(req.Schema.Model.TableName)
	if !isValidModulePath(name, scope == "business") {
		return common.NewBadRequest("module.generate.invalid_name")
	}
	if scope != "system" && scope != "business" {
		return common.NewBadRequest("module.generate.invalid_scope")
	}
	if displayName == "" {
		return common.NewBadRequest("module.generate.display_name_required")
	}
	if tableName == "" {
		return common.NewBadRequest("module.generate.table_name_required")
	}
	if err := ValidateManagedTableName(scope, tableName); err != nil {
		return common.NewBadRequest(msgInvalidTableName)
	}
	if err := validateGovernanceContract(req); err != nil {
		return err
	}
	return nil
}

func ValidateManagedTableName(scope, tableName string) error {
	normalizedScope := strings.TrimSpace(scope)
	normalizedTableName := strings.TrimSpace(tableName)
	if normalizedTableName == "" {
		return common.NewBadRequest(msgInvalidTableName)
	}
	if !managedTableNamePattern.MatchString(normalizedTableName) {
		return common.NewBadRequest(msgInvalidTableName)
	}
	if strings.Contains(normalizedTableName, "__") {
		return common.NewBadRequest(msgInvalidTableName)
	}
	switch normalizedScope {
	case "system":
		if !strings.HasPrefix(normalizedTableName, "system_") {
			return common.NewBadRequest(msgInvalidTableName)
		}
	case "business":
		if !strings.HasPrefix(normalizedTableName, "biz_") {
			return common.NewBadRequest(msgInvalidTableName)
		}
	default:
		return common.NewBadRequest(msgInvalidTableName)
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

	for _, file := range files {
		relativePath, err := writeSingleGeneratedModuleFile(workspaceRoot, backendPrefix, frontendPrefix, file, req.Overwrite, seen)
		if err != nil {
			return nil, err
		}
		written = append(written, relativePath)
	}

	schemaRelativePath, err := writeGeneratedModuleSchemaFile(workspaceRoot, scope, name, req.Schema)
	if err != nil {
		return nil, err
	}
	written = append(written, schemaRelativePath)

	if err := WriteGeneratedFallbackResources(workspaceRoot); err != nil {
		return nil, err
	}
	return written, nil
}

func writeSingleGeneratedModuleFile(workspaceRoot, backendPrefix, frontendPrefix string, file GeneratedFile, overwrite bool, seen map[string]struct{}) (string, error) {
	relativePath := filepath.ToSlash(strings.TrimSpace(file.Path))
	if relativePath == "" || strings.Contains(relativePath, "..") || !filepath.IsLocal(relativePath) {
		return "", common.NewBadRequest(msgInvalidPath)
	}
	if !strings.HasPrefix(relativePath, backendPrefix) && !strings.HasPrefix(relativePath, frontendPrefix) {
		return "", common.NewBadRequest(msgInvalidPath)
	}
	if _, ok := seen[relativePath]; ok {
		return "", common.NewConflict("module.generate.duplicate_file")
	}
	seen[relativePath] = struct{}{}

	absolutePath := filepath.Join(workspaceRoot, filepath.FromSlash(relativePath))
	if err := os.MkdirAll(filepath.Dir(absolutePath), 0o750); err != nil {
		return "", err
	}
	if !overwrite && fileExists(absolutePath) {
		return "", common.NewConflict("module.generate.file_exists")
	}
	if err := os.WriteFile(absolutePath, []byte(file.Content), 0o600); err != nil {
		return "", err
	}
	return relativePath, nil
}

func writeGeneratedModuleSchemaFile(workspaceRoot, scope, name string, schema ModuleSchema) (string, error) {
	schemaRelativePath := filepath.ToSlash(filepath.Join("schema", "generated", scope, name+".json"))
	if !filepath.IsLocal(schemaRelativePath) {
		return "", common.NewBadRequest(msgInvalidPath)
	}
	schemaPath := filepath.Join(workspaceRoot, filepath.FromSlash(schemaRelativePath))
	if err := os.MkdirAll(filepath.Dir(schemaPath), 0o755); err != nil {
		return "", err
	}
	schemaJSON, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(schemaPath, schemaJSON, 0o644); err != nil {
		return "", err
	}
	return schemaRelativePath, nil
}

func GenerateModuleFilesFromSchema(workspaceRoot string, schema ModuleSchema) ([]GeneratedFile, error) {
	scriptPath := filepath.Join(workspaceRoot, filepath.FromSlash(generatedModuleExporterScript))
	if !fileExists(scriptPath) {
		return nil, common.NewInternal(msgServerExportFailed)
	}
	nodeBinary, err := resolveNodeBinary()
	if err != nil {
		return nil, common.NewInternal(msgServerExportFailed)
	}

	schemaFile, err := os.CreateTemp("", "pantheon-module-schema-*.json")
	if err != nil {
		return nil, err
	}
	schemaPath := schemaFile.Name()
	defer func() { _ = os.Remove(schemaPath) }()

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

	cmd := exec.Command(nodeBinary, scriptPath, schemaPath)
	cmd.Dir = workspaceRoot
	output, err := cmd.Output()
	if err != nil {
		return nil, common.NewInternal(msgServerExportFailed)
	}

	var files []GeneratedFile
	if err := json.Unmarshal(output, &files); err != nil {
		return nil, common.NewInternal(msgServerExportFailed)
	}
	if len(files) == 0 {
		return nil, common.NewInternal(msgServerExportFailed)
	}
	return files, nil
}

func resolveNodeBinary() (string, error) {
	if configured := strings.TrimSpace(os.Getenv(nodeBinaryEnvKey)); configured != "" {
		if !filepath.IsAbs(configured) {
			return "", common.NewBadRequest("module.generate.invalid_node_binary")
		}
		return configured, nil
	}
	return exec.LookPath("node")
}

func RemoveGeneratedModuleSource(workspaceRoot, scope, name string) error {
	if !isValidModulePath(name, scope == "business") {
		return common.NewBadRequest("module.generate.invalid_name")
	}
	if scope != "system" && scope != "business" {
		return common.NewBadRequest("module.generate.invalid_scope")
	}

	targets := []string{
		filepath.ToSlash(filepath.Join("backend", "modules", scope, name)),
		filepath.ToSlash(filepath.Join("frontend", "src", "modules", scope, name)),
		filepath.ToSlash(filepath.Join("schema", "generated", scope, name+".json")),
	}

	for _, relativeTarget := range targets {
		if !filepath.IsLocal(relativeTarget) {
			return common.NewBadRequest(msgInvalidPath)
		}
		target := filepath.Join(workspaceRoot, filepath.FromSlash(relativeTarget))
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

func WriteGeneratedFeatureLedgerSnapshot(workspaceRoot string, snapshot any) error {
	target := filepath.Join(workspaceRoot, filepath.FromSlash(GeneratedFeatureLedgerRelativePath))
	return writeJSONFile(target, snapshot)
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
			ImportPath: fmt.Sprintf("pantheon-base/modules/%s/%s", scope, name),
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
	target := filepath.Join(workspaceRoot, "frontend", "src", "modules", "generated", scope+".ts")

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
	defer func() { _ = file.Close() }()
	return tpl.Execute(file, data)
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
		if !isValidModulePathSegment(segment) {
			return false
		}
	}
	return true
}

func isValidModulePathSegment(segment string) bool {
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
		if !unicode.IsLower(char) && !unicode.IsDigit(char) && char != '_' {
			return false
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

func WriteGeneratedFallbackResources(workspaceRoot string) error {
	localePayload := newEmptyLocalePayload()
	schemaRoot := filepath.Join(workspaceRoot, "schema", "generated")
	if dirExists(schemaRoot) {
		if err := collectGeneratedModuleLocales(schemaRoot, localePayload); err != nil {
			return err
		}
	}
	mergeFallbackLocales(localePayload)

	resourceDir := filepath.Join(workspaceRoot, "frontend", "src", "i18n", "resources", "generated")
	if err := os.MkdirAll(resourceDir, 0o750); err != nil {
		return err
	}
	return writeLocaleResourceFiles(resourceDir, localePayload)
}

func newEmptyLocalePayload() map[string]map[string]string {
	return map[string]map[string]string{
		"zh-CN": {},
		"en-US": {},
		"ja-JP": {},
		"ko-KR": {},
		"fr-FR": {},
	}
}

func collectGeneratedModuleLocales(schemaRoot string, localePayload map[string]map[string]string) error {
	return filepath.WalkDir(schemaRoot, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if shouldSkipFallbackSchemaFile(path, d) {
			return nil
		}
		content, err := os.ReadFile(path) // #nosec G304 -- path comes from filepath.WalkDir under the controlled schema root.
		if err != nil {
			return err
		}
		var schema ModuleSchema
		if err := json.Unmarshal(content, &schema); err != nil {
			return nil
		}
		copySchemaTranslations(localePayload, "zh-CN", schema.I18n.Translations.Zh)
		copySchemaTranslations(localePayload, "en-US", schema.I18n.Translations.En)
		return nil
	})
}

func shouldSkipFallbackSchemaFile(path string, d os.DirEntry) bool {
	if d.IsDir() {
		return true
	}
	if !strings.EqualFold(filepath.Ext(path), ".json") {
		return true
	}
	if strings.EqualFold(filepath.Base(path), filepath.Base(GeneratedFeatureLedgerRelativePath)) {
		return true
	}
	return false
}

func copySchemaTranslations(localePayload map[string]map[string]string, locale string, translations map[string]string) {
	for key, value := range translations {
		if strings.TrimSpace(key) == "" || strings.TrimSpace(value) == "" {
			continue
		}
		localePayload[locale][key] = value
	}
}

func mergeFallbackLocales(localePayload map[string]map[string]string) {
	for _, locale := range []string{"ja-JP", "ko-KR", "fr-FR"} {
		for key, value := range localePayload["en-US"] {
			if strings.TrimSpace(localePayload[locale][key]) == "" {
				localePayload[locale][key] = value
			}
		}
	}
}

func writeLocaleResourceFiles(resourceDir string, localePayload map[string]map[string]string) error {
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
