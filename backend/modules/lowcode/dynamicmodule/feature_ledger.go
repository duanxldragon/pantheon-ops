package dynamicmodule

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"pantheon-ops/backend/internal/scaffold"
	"pantheon-ops/backend/pkg/common"
)

const featureLedgerVersion = 1

type FeatureLedgerSnapshot struct {
	Version       int                  `json:"version"`
	SourceOfTruth FeatureLedgerSources `json:"sourceOfTruth"`
	Entries       []FeatureLedgerEntry `json:"entries"`
	Issues        []FeatureLedgerIssue `json:"issues,omitempty"`
}

type FeatureLedgerSources struct {
	Registrations string `json:"registrations"`
	Schemas       string `json:"schemas"`
	Snapshot      string `json:"snapshot"`
}

type FeatureLedgerEntry struct {
	ModuleKey      string `json:"moduleKey"`
	Name           string `json:"name"`
	Scope          string `json:"scope"`
	DisplayName    string `json:"displayName"`
	Owner          string `json:"owner"`
	BoundedContext string `json:"boundedContext"`
	SourceMode     string `json:"sourceMode"`
	Source         string `json:"source"`
	Maturity       string `json:"maturity"`
	Status         string `json:"status"`
	TableName      string `json:"tableName"`
	SchemaPath     string `json:"schemaPath"`
	BuiltIn        bool   `json:"builtIn"`
	AutoRecycle    bool   `json:"autoRecycle"`
}

type FeatureLedgerIssue struct {
	ModuleKey string `json:"moduleKey"`
	Severity  string `json:"severity"`
	Code      string `json:"code"`
	Field     string `json:"field,omitempty"`
	Detail    string `json:"detail"`
}

type featureLedgerSchemaFile struct {
	Scope        string
	Name         string
	RelativePath string
	AbsolutePath string
}

type featureLedgerEvaluationRule struct {
	File            featureLedgerSchemaFile
	SchemaPresent   bool
	Registration    ModuleRegistration
	HasRegistration bool
}

type featureLedgerEvaluationResult struct {
	Entry  FeatureLedgerEntry
	Issues []FeatureLedgerIssue
}

type featureLedgerEntryContext struct {
	File            featureLedgerSchemaFile
	ModuleKey       string
	Schema          *scaffold.ModuleSchema
	SchemaPresent   bool
	Registration    ModuleRegistration
	HasRegistration bool
}

type featureLedgerRegistrationField struct {
	Target *string
	Value  string
	Code   string
	Field  string
}

type featureLedgerRequiredField struct {
	Value  string
	Code   string
	Field  string
	Detail string
}

func (s *DynamicModuleService) refreshGeneratedWorkspaceArtifacts() (*FeatureLedgerSnapshot, int, error) {
	if strings.TrimSpace(s.workspaceRoot) == "" {
		return nil, 0, errors.New("workspace.not_found")
	}

	refs, err := s.listGeneratedModuleRefs()
	if err != nil {
		return nil, 0, err
	}
	if err := scaffold.WriteGeneratedRegistries(s.workspaceRoot, refs); err != nil {
		return nil, 0, err
	}

	snapshot, err := s.buildFeatureLedgerSnapshot()
	if err != nil {
		return nil, 0, err
	}
	if err := scaffold.WriteGeneratedFeatureLedgerSnapshot(s.workspaceRoot, snapshot); err != nil {
		return nil, 0, err
	}
	return snapshot, len(refs), nil
}

func (s *DynamicModuleService) refreshGeneratedWorkspaceArtifactsIfAvailable() (*FeatureLedgerSnapshot, error) {
	if strings.TrimSpace(s.workspaceRoot) == "" {
		return nil, nil
	}
	snapshot, _, err := s.refreshGeneratedWorkspaceArtifacts()
	return snapshot, err
}

func (s *DynamicModuleService) buildFeatureLedgerSnapshot() (*FeatureLedgerSnapshot, error) {
	if s.db == nil {
		return nil, common.ErrDatabaseNotInitialized
	}

	schemaFiles, err := s.collectFeatureLedgerSchemaFiles()
	if err != nil {
		return nil, err
	}
	registrations, err := s.collectFeatureLedgerRegistrations()
	if err != nil {
		return nil, err
	}

	rules, issues := collectFeatureLedgerEvaluationRules(schemaFiles, registrations)
	results := evaluateFeatureLedgerRules(rules)

	return formatFeatureLedgerSnapshot(results, issues), nil
}

func collectFeatureLedgerEvaluationRules(schemaFiles []featureLedgerSchemaFile, registrations map[string]ModuleRegistration) ([]featureLedgerEvaluationRule, []FeatureLedgerIssue) {
	rules := make([]featureLedgerEvaluationRule, 0, len(schemaFiles)+len(registrations))
	issues := make([]FeatureLedgerIssue, 0)
	seen := make(map[string]struct{}, len(schemaFiles))

	for _, file := range schemaFiles {
		moduleKey := buildModuleKey(file.Scope, file.Name)
		registration, hasRegistration := registrations[moduleKey]
		rules = append(rules, featureLedgerEvaluationRule{
			File:            file,
			SchemaPresent:   true,
			Registration:    registration,
			HasRegistration: hasRegistration,
		})
		seen[moduleKey] = struct{}{}
	}

	for moduleKey, registration := range registrations {
		if _, ok := seen[moduleKey]; ok {
			continue
		}
		rule, err := buildRegistrationOnlyFeatureLedgerRule(moduleKey, registration)
		if err != nil {
			issues = append(issues, invalidFeatureLedgerRegistrationIssue(moduleKey, err))
			continue
		}
		rules = append(rules, rule)
	}

	return rules, issues
}

func buildRegistrationOnlyFeatureLedgerRule(moduleKey string, registration ModuleRegistration) (featureLedgerEvaluationRule, error) {
	scope, name, err := splitModuleKey(moduleKey)
	if err != nil {
		return featureLedgerEvaluationRule{}, err
	}
	return featureLedgerEvaluationRule{
		File: featureLedgerSchemaFile{
			Scope:        scope,
			Name:         name,
			RelativePath: filepath.ToSlash(filepath.Join("schema", "generated", scope, name+".json")),
		},
		SchemaPresent:   false,
		Registration:    registration,
		HasRegistration: true,
	}, nil
}

func invalidFeatureLedgerRegistrationIssue(moduleKey string, err error) FeatureLedgerIssue {
	return FeatureLedgerIssue{
		ModuleKey: moduleKey,
		Severity:  "error",
		Code:      "registration_key_invalid",
		Detail:    err.Error(),
	}
}

func evaluateFeatureLedgerRules(rules []featureLedgerEvaluationRule) []featureLedgerEvaluationResult {
	results := make([]featureLedgerEvaluationResult, 0, len(rules))
	for _, rule := range rules {
		results = append(results, evaluateFeatureLedgerRule(rule))
	}
	return results
}

func evaluateFeatureLedgerRule(rule featureLedgerEvaluationRule) featureLedgerEvaluationResult {
	var schema *scaffold.ModuleSchema
	var issues []FeatureLedgerIssue
	moduleKey := buildModuleKey(rule.File.Scope, rule.File.Name)

	if rule.SchemaPresent {
		schema, issues = loadFeatureLedgerSchema(rule.File.AbsolutePath)
		applyFeatureLedgerIssueModuleKey(issues, moduleKey)
	}

	entry, entryIssues := buildFeatureLedgerEntry(rule.File, schema, rule.SchemaPresent, rule.Registration, rule.HasRegistration)
	issues = append(issues, entryIssues...)
	return featureLedgerEvaluationResult{
		Entry:  entry,
		Issues: issues,
	}
}

func applyFeatureLedgerIssueModuleKey(issues []FeatureLedgerIssue, moduleKey string) {
	for index := range issues {
		if issues[index].ModuleKey == "" {
			issues[index].ModuleKey = moduleKey
		}
	}
}

func formatFeatureLedgerSnapshot(results []featureLedgerEvaluationResult, issues []FeatureLedgerIssue) *FeatureLedgerSnapshot {
	entries := make([]FeatureLedgerEntry, 0, len(results))
	for _, result := range results {
		entries = append(entries, result.Entry)
		issues = append(issues, result.Issues...)
	}

	sortFeatureLedgerEntries(entries)
	sortFeatureLedgerIssues(issues)

	return &FeatureLedgerSnapshot{
		Version: featureLedgerVersion,
		SourceOfTruth: FeatureLedgerSources{
			Registrations: "system_module_registration",
			Schemas:       "schema/generated",
			Snapshot:      scaffold.GeneratedFeatureLedgerRelativePath,
		},
		Entries: entries,
		Issues:  issues,
	}
}

func (s *DynamicModuleService) collectFeatureLedgerSchemaFiles() ([]featureLedgerSchemaFile, error) {
	schemaRoot := filepath.Join(s.workspaceRoot, "schema", "generated")
	info, err := os.Stat(schemaRoot)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []featureLedgerSchemaFile{}, nil
		}
		return nil, err
	}
	if !info.IsDir() {
		return nil, errors.New("workspace.schema_root_invalid")
	}

	files := make([]featureLedgerSchemaFile, 0)
	walkErr := filepath.WalkDir(schemaRoot, func(path string, d os.DirEntry, walkErr error) error {
		if hasFeatureLedgerWalkError(walkErr) {
			return walkErr
		}
		if !isFeatureLedgerSchemaCandidate(path, d) {
			return nil
		}

		relativePath, err := filepath.Rel(schemaRoot, path)
		if err != nil {
			return err
		}
		scope, name, ok := splitFeatureLedgerSchemaRelativePath(relativePath)
		if !ok {
			return nil
		}
		files = append(files, featureLedgerSchemaFile{
			Scope:        scope,
			Name:         name,
			RelativePath: filepath.ToSlash(relativePath),
			AbsolutePath: path,
		})
		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}

	sortFeatureLedgerSchemaFiles(files)

	return files, nil
}

func (s *DynamicModuleService) collectFeatureLedgerRegistrations() (map[string]ModuleRegistration, error) {
	var modules []ModuleRegistration
	if err := s.db.Where("table_name <> '' AND status <> ?", ModuleStatusUninstalled).Order("scope ASC").Order("name ASC").Find(&modules).Error; err != nil {
		return nil, err
	}

	registrations := make(map[string]ModuleRegistration, len(modules))
	for _, module := range modules {
		registrations[module.Name] = module
	}
	return registrations, nil
}

func loadFeatureLedgerSchema(path string) (*scaffold.ModuleSchema, []FeatureLedgerIssue) {
	issues := make([]FeatureLedgerIssue, 0, 1)
	content, err := os.ReadFile(path)
	if err != nil {
		issues = append(issues, FeatureLedgerIssue{
			Severity: "error",
			Code:     "schema_unreadable",
			Detail:   err.Error(),
		})
		return nil, issues
	}

	var schema scaffold.ModuleSchema
	if err := json.Unmarshal(content, &schema); err != nil {
		issues = append(issues, FeatureLedgerIssue{
			Severity: "error",
			Code:     "schema_invalid",
			Detail:   err.Error(),
		})
		return nil, issues
	}
	return &schema, issues
}

func buildFeatureLedgerEntry(file featureLedgerSchemaFile, schema *scaffold.ModuleSchema, schemaPresent bool, registration ModuleRegistration, hasRegistration bool) (FeatureLedgerEntry, []FeatureLedgerIssue) {
	ctx := featureLedgerEntryContext{
		File:            file,
		ModuleKey:       buildModuleKey(file.Scope, file.Name),
		Schema:          schema,
		SchemaPresent:   schemaPresent,
		Registration:    registration,
		HasRegistration: hasRegistration,
	}
	entry := newFeatureLedgerEntry(ctx)
	issues := make([]FeatureLedgerIssue, 0, 8)

	issues = append(issues, applyFeatureLedgerSchema(&entry, ctx)...)
	issues = append(issues, applyFeatureLedgerRegistration(&entry, ctx)...)
	applyFeatureLedgerDefaults(&entry, ctx)
	issues = append(issues, validateFeatureLedgerCompleteness(entry, ctx.ModuleKey)...)

	return entry, dedupeFeatureLedgerIssues(issues)
}

func newFeatureLedgerEntry(ctx featureLedgerEntryContext) FeatureLedgerEntry {
	entry := FeatureLedgerEntry{
		ModuleKey:  ctx.ModuleKey,
		Name:       ctx.File.Name,
		Scope:      ctx.File.Scope,
		SchemaPath: ctx.File.RelativePath,
	}
	return entry
}

func applyFeatureLedgerSchema(entry *FeatureLedgerEntry, ctx featureLedgerEntryContext) []FeatureLedgerIssue {
	if ctx.Schema == nil {
		if ctx.SchemaPresent {
			return nil
		}
		return []FeatureLedgerIssue{{
			ModuleKey: ctx.ModuleKey,
			Severity:  "error",
			Code:      "schema_missing",
			Detail:    ctx.File.RelativePath,
		}}
	}

	entry.DisplayName = strings.TrimSpace(ctx.Schema.DisplayName)
	entry.Owner = strings.TrimSpace(ctx.Schema.Metadata.Owner)
	entry.BoundedContext = strings.TrimSpace(ctx.Schema.Metadata.BoundedContext)
	entry.SourceMode = strings.TrimSpace(ctx.Schema.Metadata.SourceMode)
	entry.TableName = strings.TrimSpace(ctx.Schema.Model.TableName)
	entry.AutoRecycle = ctx.Schema.Metadata.AutoRecycle
	return validateFeatureLedgerSchemaIdentity(ctx)
}

func validateFeatureLedgerSchemaIdentity(ctx featureLedgerEntryContext) []FeatureLedgerIssue {
	issues := make([]FeatureLedgerIssue, 0, 2)
	if hasFeatureLedgerSchemaNameMismatch(ctx.Schema, ctx.File) {
		issues = append(issues, featureLedgerMismatchIssue(ctx.ModuleKey, "schema_name_mismatch", "name", ctx.Schema.Name, ctx.File.Name))
	}
	if hasFeatureLedgerSchemaScopeMismatch(ctx.Schema, ctx.File) {
		issues = append(issues, featureLedgerMismatchIssue(ctx.ModuleKey, "schema_scope_mismatch", "scope", ctx.Schema.Scope, ctx.File.Scope))
	}
	return issues
}

func hasFeatureLedgerSchemaNameMismatch(schema *scaffold.ModuleSchema, file featureLedgerSchemaFile) bool {
	return strings.TrimSpace(schema.Name) != "" && strings.TrimSpace(schema.Name) != file.Name
}

func hasFeatureLedgerSchemaScopeMismatch(schema *scaffold.ModuleSchema, file featureLedgerSchemaFile) bool {
	return strings.TrimSpace(schema.Scope) != "" && strings.TrimSpace(schema.Scope) != file.Scope
}

func applyFeatureLedgerRegistration(entry *FeatureLedgerEntry, ctx featureLedgerEntryContext) []FeatureLedgerIssue {
	if !ctx.HasRegistration {
		return applyMissingFeatureLedgerRegistration(entry, ctx)
	}

	entry.Source = strings.TrimSpace(ctx.Registration.Source)
	entry.Status = featureLedgerStatusLabel(ctx.Registration.Status)
	entry.BuiltIn = ctx.Registration.ModelTableName == ""

	issues := applyFeatureLedgerRegistrationFields(entry, ctx)
	issues = append(issues, applyFeatureLedgerRegistrationAutoRecycle(entry, ctx)...)
	issues = append(issues, applyFeatureLedgerRegistrationSource(entry, ctx)...)
	return issues
}

func applyFeatureLedgerRegistrationFields(entry *FeatureLedgerEntry, ctx featureLedgerEntryContext) []FeatureLedgerIssue {
	fields := []featureLedgerRegistrationField{
		{Target: &entry.DisplayName, Value: ctx.Registration.DisplayName, Code: "display_name_mismatch", Field: "displayName"},
		{Target: &entry.Owner, Value: ctx.Registration.Owner, Code: "owner_mismatch", Field: "owner"},
		{Target: &entry.BoundedContext, Value: ctx.Registration.BoundedContext, Code: "bounded_context_mismatch", Field: "boundedContext"},
		{Target: &entry.TableName, Value: ctx.Registration.ModelTableName, Code: "table_name_mismatch", Field: "tableName"},
	}

	issues := make([]FeatureLedgerIssue, 0, len(fields))
	for _, field := range fields {
		issues = append(issues, applyFeatureLedgerRegistrationField(ctx.ModuleKey, field)...)
	}
	return issues
}

func applyFeatureLedgerRegistrationField(moduleKey string, field featureLedgerRegistrationField) []FeatureLedgerIssue {
	value := strings.TrimSpace(field.Value)
	if value == "" {
		return nil
	}
	if *field.Target == "" {
		*field.Target = value
		return nil
	}
	if value == *field.Target {
		return nil
	}
	return []FeatureLedgerIssue{featureLedgerMismatchIssue(moduleKey, field.Code, field.Field, field.Value, *field.Target)}
}

func applyFeatureLedgerRegistrationAutoRecycle(entry *FeatureLedgerEntry, ctx featureLedgerEntryContext) []FeatureLedgerIssue {
	if ctx.Schema == nil {
		entry.AutoRecycle = ctx.Registration.AutoRecycle
		return nil
	}
	if !hasFeatureLedgerAutoRecycleMismatch(entry, ctx.Registration) {
		return nil
	}
	return []FeatureLedgerIssue{featureLedgerMismatchIssue(ctx.ModuleKey, "auto_recycle_mismatch", "autoRecycle", fmt.Sprintf("%t", ctx.Registration.AutoRecycle), fmt.Sprintf("%t", entry.AutoRecycle))}
}

func hasFeatureLedgerAutoRecycleMismatch(entry *FeatureLedgerEntry, registration ModuleRegistration) bool {
	return registration.AutoRecycle != entry.AutoRecycle
}

func applyFeatureLedgerRegistrationSource(entry *FeatureLedgerEntry, ctx featureLedgerEntryContext) []FeatureLedgerIssue {
	if ctx.Schema == nil {
		return nil
	}

	derivedSource := inferFeatureLedgerSource(*entry, ctx.File)
	entry.Source = strings.TrimSpace(ctx.Registration.Source)
	entry.Maturity = featureLedgerMaturity(ctx.Registration)
	if !hasFeatureLedgerSourceMismatch(ctx.Registration, derivedSource) {
		return nil
	}
	return []FeatureLedgerIssue{featureLedgerMismatchIssue(ctx.ModuleKey, "source_mismatch", "source", ctx.Registration.Source, derivedSource)}
}

func hasFeatureLedgerSourceMismatch(registration ModuleRegistration, derivedSource string) bool {
	return strings.TrimSpace(registration.Source) != "" && strings.TrimSpace(registration.Source) != derivedSource
}

func applyMissingFeatureLedgerRegistration(entry *FeatureLedgerEntry, ctx featureLedgerEntryContext) []FeatureLedgerIssue {
	if entry.Source == "" {
		entry.Source = inferFeatureLedgerSource(*entry, ctx.File)
	}
	entry.Maturity = "draft"
	entry.Status = "missing_registration"
	return []FeatureLedgerIssue{{
		ModuleKey: ctx.ModuleKey,
		Severity:  "error",
		Code:      "registration_missing",
		Detail:    ctx.File.RelativePath,
	}}
}

func applyFeatureLedgerDefaults(entry *FeatureLedgerEntry, ctx featureLedgerEntryContext) {
	if shouldInferFeatureLedgerMaturity(*entry, ctx) {
		entry.Maturity = featureLedgerMaturity(ctx.Registration)
	}
	if isFeatureLedgerDisplayNameMissing(*entry) {
		entry.DisplayName = ctx.ModuleKey
	}
	if isFeatureLedgerSourceMissing(*entry) {
		entry.Source = inferFeatureLedgerSource(*entry, ctx.File)
	}
	if isFeatureLedgerStatusMissing(*entry) && ctx.HasRegistration {
		entry.Status = featureLedgerStatusLabel(ctx.Registration.Status)
	}
	if isFeatureLedgerMaturityMissing(*entry) {
		entry.Maturity = "draft"
	}
}

func shouldInferFeatureLedgerMaturity(entry FeatureLedgerEntry, ctx featureLedgerEntryContext) bool {
	return ctx.HasRegistration && entry.Maturity == ""
}

func isFeatureLedgerDisplayNameMissing(entry FeatureLedgerEntry) bool {
	return strings.TrimSpace(entry.DisplayName) == ""
}

func isFeatureLedgerSourceMissing(entry FeatureLedgerEntry) bool {
	return strings.TrimSpace(entry.Source) == ""
}

func isFeatureLedgerStatusMissing(entry FeatureLedgerEntry) bool {
	return strings.TrimSpace(entry.Status) == ""
}

func isFeatureLedgerMaturityMissing(entry FeatureLedgerEntry) bool {
	return entry.Maturity == ""
}

func inferFeatureLedgerSource(entry FeatureLedgerEntry, file featureLedgerSchemaFile) string {
	return inferRegistrationSource(file.Scope, entry.SourceMode, file.Name, true)
}

func validateFeatureLedgerCompleteness(entry FeatureLedgerEntry, moduleKey string) []FeatureLedgerIssue {
	requiredFields := []featureLedgerRequiredField{
		{
			Value:  entry.Owner,
			Code:   "owner_missing",
			Field:  "owner",
			Detail: "owner is required for feature ledger completeness",
		},
		{
			Value:  entry.BoundedContext,
			Code:   "bounded_context_missing",
			Field:  "boundedContext",
			Detail: "bounded context is required for feature ledger completeness",
		},
		{
			Value:  entry.SourceMode,
			Code:   "source_mode_missing",
			Field:  "sourceMode",
			Detail: "source mode is required for feature ledger completeness",
		},
		{
			Value:  entry.TableName,
			Code:   "table_name_missing",
			Field:  "tableName",
			Detail: "table name is required for feature ledger completeness",
		},
	}

	issues := make([]FeatureLedgerIssue, 0, len(requiredFields))
	for _, field := range requiredFields {
		if strings.TrimSpace(field.Value) == "" {
			issues = append(issues, missingFeatureLedgerFieldIssue(moduleKey, field))
		}
	}
	return issues
}

func missingFeatureLedgerFieldIssue(moduleKey string, field featureLedgerRequiredField) FeatureLedgerIssue {
	return FeatureLedgerIssue{
		ModuleKey: moduleKey,
		Severity:  "warn",
		Code:      field.Code,
		Field:     field.Field,
		Detail:    field.Detail,
	}
}

func featureLedgerMaturity(registration ModuleRegistration) string {
	if registration.ModelTableName == "" {
		return "core"
	}
	switch registration.Status {
	case ModuleStatusActive:
		return "stable"
	case ModuleStatusPendingActivation:
		return "experimental"
	case ModuleStatusFailed:
		return "draft"
	case ModuleStatusUninstalled:
		return "draft"
	default:
		return "draft"
	}
}

func featureLedgerStatusLabel(status int) string {
	switch status {
	case ModuleStatusActive:
		return "active"
	case ModuleStatusUninstalled:
		return "uninstalled"
	case ModuleStatusPendingActivation:
		return "pending_activation"
	case ModuleStatusFailed:
		return "failed"
	default:
		return "unknown"
	}
}

func featureLedgerMismatchIssue(moduleKey, code, field, actual, expected string) FeatureLedgerIssue {
	return FeatureLedgerIssue{
		ModuleKey: moduleKey,
		Severity:  "warn",
		Code:      code,
		Field:     field,
		Detail:    fmt.Sprintf("actual=%q expected=%q", strings.TrimSpace(actual), strings.TrimSpace(expected)),
	}
}

func dedupeFeatureLedgerIssues(issues []FeatureLedgerIssue) []FeatureLedgerIssue {
	if len(issues) <= 1 {
		return issues
	}
	seen := make(map[string]struct{}, len(issues))
	filtered := make([]FeatureLedgerIssue, 0, len(issues))
	for _, issue := range issues {
		key := issue.ModuleKey + "|" + issue.Code + "|" + issue.Field + "|" + issue.Detail + "|" + issue.Severity
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		filtered = append(filtered, issue)
	}
	return filtered
}

func sortFeatureLedgerEntries(entries []FeatureLedgerEntry) {
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Scope == entries[j].Scope {
			return entries[i].Name < entries[j].Name
		}
		return entries[i].Scope < entries[j].Scope
	})
}

func sortFeatureLedgerIssues(issues []FeatureLedgerIssue) {
	sort.SliceStable(issues, func(i, j int) bool {
		return lessFeatureLedgerIssue(issues[i], issues[j])
	})
}

func lessFeatureLedgerIssue(left, right FeatureLedgerIssue) bool {
	if left.ModuleKey != right.ModuleKey {
		return left.ModuleKey < right.ModuleKey
	}
	if left.Code != right.Code {
		return left.Code < right.Code
	}
	if left.Field != right.Field {
		return left.Field < right.Field
	}
	if left.Severity != right.Severity {
		return left.Severity < right.Severity
	}
	return left.Detail < right.Detail
}

func sortFeatureLedgerSchemaFiles(files []featureLedgerSchemaFile) {
	sort.Slice(files, func(i, j int) bool {
		if files[i].Scope == files[j].Scope {
			return files[i].Name < files[j].Name
		}
		return files[i].Scope < files[j].Scope
	})
}

func hasFeatureLedgerWalkError(walkErr error) bool {
	return walkErr != nil
}

func isFeatureLedgerSchemaCandidate(path string, d os.DirEntry) bool {
	return !d.IsDir() && isFeatureLedgerJSONFile(path) && !isFeatureLedgerSnapshotFile(path)
}

func isFeatureLedgerJSONFile(path string) bool {
	return strings.EqualFold(filepath.Ext(path), ".json")
}

func isFeatureLedgerSnapshotFile(path string) bool {
	return strings.EqualFold(filepath.Base(path), filepath.Base(scaffold.GeneratedFeatureLedgerRelativePath))
}

func splitFeatureLedgerSchemaRelativePath(relativePath string) (string, string, bool) {
	normalized := filepath.ToSlash(strings.TrimSpace(relativePath))
	if isInvalidFeatureLedgerSchemaPath(normalized) {
		return "", "", false
	}
	parts := strings.Split(normalized, "/")
	if len(parts) < 2 {
		return "", "", false
	}
	scope := strings.TrimSpace(parts[0])
	if !isFeatureLedgerSchemaScope(scope) {
		return "", "", false
	}
	name := strings.TrimSuffix(strings.Join(parts[1:], "/"), ".json")
	name = strings.TrimSpace(name)
	if name == "" {
		return "", "", false
	}
	return scope, name, true
}

func isInvalidFeatureLedgerSchemaPath(normalized string) bool {
	return normalized == "" || isFeatureLedgerSnapshotFile(normalized)
}

func isFeatureLedgerSchemaScope(scope string) bool {
	return scope == "system" || scope == "business"
}
