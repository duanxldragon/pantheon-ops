package dynamicmodule

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gorm.io/gorm"

	"pantheon-base/internal/scaffold"
	"pantheon-base/pkg/database"
)

func (s *DynamicModuleService) SyncBuiltInModules() error {
	return s.syncModuleRegistrationRecords(true)
}

func (s *DynamicModuleService) syncModuleRegistrationRecords(rewriteGeneratedRegistries bool) error {
	if s.db == nil {
		return nil
	}
	if database.ShouldAutoMigrate() {
		if err := s.db.AutoMigrate(&ModuleRegistration{}); err != nil {
			return err
		}
	}
	if _, err := s.syncGeneratedModuleRegistrations(); err != nil {
		return err
	}
	if rewriteGeneratedRegistries {
		if err := s.RebuildGeneratedRegistries(); err != nil {
			return err
		}
	}
	if !s.db.Migrator().HasTable("system_menu") {
		return nil
	}

	return s.syncMenuBackedModuleRegistrations()
}

func (s *DynamicModuleService) syncMenuBackedModuleRegistrations() error {
	if s.db == nil {
		return nil
	}
	if !s.db.Migrator().HasTable("system_menu") {
		return nil
	}
	if err := s.normalizeStaticModuleAliases(); err != nil {
		return err
	}

	moduleNames, err := s.queryMenuBackedModuleNames()
	if err != nil {
		return err
	}

	now := time.Now().Format(time.RFC3339)
	for _, rawModule := range moduleNames {
		if err := s.upsertMenuBackedModule(rawModule, now); err != nil {
			return err
		}
	}
	return nil
}

// queryMenuBackedModuleNames returns the distinct non-empty module identifiers
// referenced by menu rows of type M (menu) or C (catalog).
func (s *DynamicModuleService) queryMenuBackedModuleNames() ([]string, error) {
	type menuModuleRow struct {
		Module string
	}

	var rows []menuModuleRow
	if err := s.db.Table("system_menu").
		Select("DISTINCT module AS module").
		Where("module <> '' AND type IN ?", []string{"M", "C"}).
		Order("module ASC").
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	names := make([]string, 0, len(rows))
	for _, row := range rows {
		names = append(names, row.Module)
	}
	return names, nil
}

// upsertMenuBackedModule ensures a registration exists for the given raw module
// identifier, creating or patching it as needed.
func (s *DynamicModuleService) upsertMenuBackedModule(rawModule, now string) error {
	moduleName := normalizeStaticModuleName(rawModule)
	if moduleName == "" {
		return nil
	}

	registration := s.buildMenuBackedRegistration(moduleName, now)

	var existing ModuleRegistration
	err := s.db.Where(condNameEquals, moduleName).First(&existing).Error
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		return s.db.Create(&registration).Error
	case err != nil:
		return err
	default:
		return s.applyMenuBackedRegistrationUpdate(&existing, &registration, now)
	}
}

// buildMenuBackedRegistration constructs the desired registration state for a
// menu-backed module.
func (s *DynamicModuleService) buildMenuBackedRegistration(moduleName, now string) ModuleRegistration {
	return ModuleRegistration{
		Name:           moduleName,
		DisplayName:    moduleName,
		Scope:          inferModuleScope(moduleName),
		Source:         inferStaticModuleSource(moduleName),
		ModelTableName: "",
		Status:         ModuleStatusActive,
		InstalledAt:    now,
	}
}

// applyMenuBackedRegistrationUpdate patches mutable fields on an existing
// registration, preserving any values already populated.
func (s *DynamicModuleService) applyMenuBackedRegistrationUpdate(existing *ModuleRegistration, registration *ModuleRegistration, now string) error {
	updates := map[string]interface{}{
		"status": ModuleStatusActive,
	}
	if strings.TrimSpace(existing.DisplayName) == "" {
		updates["display_name"] = registration.DisplayName
	}
	if strings.TrimSpace(existing.Scope) == "" {
		updates["scope"] = registration.Scope
	}
	if strings.TrimSpace(existing.Source) == "" {
		updates["source"] = registration.Source
	}
	if strings.TrimSpace(existing.InstalledAt) == "" {
		updates["installed_at"] = registration.InstalledAt
	}
	if len(updates) > 0 {
		return s.db.Model(existing).Updates(updates).Error
	}
	return nil
}

func (s *DynamicModuleService) normalizeStaticModuleAliases() error {
	if s.db == nil {
		return nil
	}

	aliases := map[string]string{
		"platform.lowcode": "system.lowcode",
	}
	for legacyName, canonicalName := range aliases {
		if err := s.migrateStaticModuleAlias(legacyName, canonicalName); err != nil {
			return err
		}
	}
	return nil
}

// migrateStaticModuleAlias rewrites a legacy static module name to its canonical
// form across the menu, i18n and registration tables.
func (s *DynamicModuleService) migrateStaticModuleAlias(legacyName, canonicalName string) error {
	if err := s.db.Table("system_menu").Where(condModuleEquals, legacyName).Update("module", canonicalName).Error; err != nil {
		return err
	}
	if s.db.Migrator().HasTable("system_i18n") {
		if err := s.db.Table("system_i18n").Where(condModuleEquals, legacyName).Update("module", canonicalName).Error; err != nil {
			return err
		}
	}

	var canonicalCount int64
	if err := s.db.Model(&ModuleRegistration{}).Where(condNameEquals, canonicalName).Count(&canonicalCount).Error; err != nil {
		return err
	}
	if canonicalCount > 0 {
		return s.db.Where("name = ? AND table_name = ''", legacyName).Delete(&ModuleRegistration{}).Error
	}

	return s.db.Model(&ModuleRegistration{}).
		Where("name = ? AND table_name = ''", legacyName).
		Updates(map[string]any{
			"name":   canonicalName,
			"scope":  inferModuleScope(canonicalName),
			"source": inferStaticModuleSource(canonicalName),
		}).Error
}

func (s *DynamicModuleService) AuditAndRepairGeneratedRegistries() (*RegistryRepairSummary, error) {
	if s.db == nil {
		return nil, nil
	}
	if database.ShouldAutoMigrate() {
		if err := s.db.AutoMigrate(&ModuleRegistration{}); err != nil {
			return nil, err
		}
	}
	markedUninstalled, err := s.syncGeneratedModuleRegistrations()
	if err != nil {
		return nil, err
	}
	_, generatedRegistryRefs, err := s.refreshGeneratedWorkspaceArtifacts()
	if err != nil {
		return nil, err
	}

	var modules []ModuleRegistration
	if err := s.db.Where("table_name <> ''").Find(&modules).Error; err != nil {
		return nil, err
	}

	summary := &RegistryRepairSummary{
		CheckedModules:           len(modules),
		GeneratedRegistryRefs:    generatedRegistryRefs,
		MarkedUninstalledModules: markedUninstalled,
	}
	for _, module := range modules {
		scope, name, err := splitModuleKey(module.Name)
		if err != nil {
			continue
		}
		if s.generatedModuleArtifactsExist(scope, name) {
			summary.ArtifactReadyModules++
		}
		if module.Status == ModuleStatusUninstalled {
			summary.PreservedUninstalledCount++
		}
	}
	return summary, nil
}

// generatedSchema mirrors the on-disk JSON shape of a generated module schema.
type generatedSchema struct {
	Name        string `json:"name"`
	DisplayName string `json:"displayName"`
	Scope       string `json:"scope"`
	Metadata    struct {
		BoundedContext string `json:"boundedContext"`
		Owner          string `json:"owner"`
		Summary        string `json:"summary"`
		SourceMode     string `json:"sourceMode"`
		SourceTable    string `json:"sourceTable"`
		AutoRecycle    bool   `json:"autoRecycle"`
	} `json:"metadata"`
	Model struct {
		TableName string `json:"tableName"`
	} `json:"model"`
}

func (s *DynamicModuleService) syncGeneratedModuleRegistrations() (int, error) {
	schemaRoot, ok, err := s.resolveGeneratedSchemaRoot()
	if err != nil {
		return 0, err
	}
	if !ok {
		return 0, nil
	}

	now := time.Now().Format(time.RFC3339)
	discovered, err := s.discoverGeneratedModuleRegistrations(schemaRoot, now)
	if err != nil {
		return 0, err
	}

	return s.reconcileMissingGeneratedModules(discovered, now)
}

// resolveGeneratedSchemaRoot returns the path to the generated schema directory.
// The boolean result is false when synchronization should be skipped.
func (s *DynamicModuleService) resolveGeneratedSchemaRoot() (string, bool, error) {
	if strings.TrimSpace(s.workspaceRoot) == "" {
		return "", false, nil
	}
	schemaRoot := filepath.Join(s.workspaceRoot, "schema", "generated")
	info, err := os.Stat(schemaRoot)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", false, nil
		}
		return "", false, err
	}
	if !info.IsDir() {
		return "", false, nil
	}
	return schemaRoot, true, nil
}

// discoverGeneratedModuleRegistrations walks the generated schema directory and
// returns the set of module keys that currently have backing artifacts.
func (s *DynamicModuleService) discoverGeneratedModuleRegistrations(schemaRoot, now string) (map[string]struct{}, error) {
	discovered := make(map[string]struct{})
	walkErr := filepath.WalkDir(schemaRoot, func(path string, d os.DirEntry, err error) error {
		return s.processGeneratedSchemaFile(path, d, err, now, discovered)
	})
	if walkErr != nil {
		return nil, walkErr
	}
	return discovered, nil
}

// processGeneratedSchemaFile validates and upserts a single generated schema
// file, recording its module key in the discovered set on success.
func (s *DynamicModuleService) processGeneratedSchemaFile(path string, d os.DirEntry, walkErr error, now string, discovered map[string]struct{}) error {
	if s.shouldSkipGeneratedSchemaFile(path, d, walkErr) {
		return nil
	}

	schema, ok, err := s.loadGeneratedSchema(path)
	if !ok || err != nil {
		return err
	}

	scope, name, tableName, valid := s.parseGeneratedSchemaIdentity(&schema)
	if !valid {
		return nil
	}
	if !s.generatedModuleArtifactsExist(scope, name) {
		return nil
	}

	moduleKey := buildModuleKey(scope, name)
	discovered[moduleKey] = struct{}{}
	return s.upsertGeneratedModuleRegistration(&schema, scope, name, tableName, moduleKey, now)
}

// shouldSkipGeneratedSchemaFile reports whether the walker entry is not a
// generated schema file we should process.
func (s *DynamicModuleService) shouldSkipGeneratedSchemaFile(path string, d os.DirEntry, walkErr error) bool {
	if walkErr != nil {
		return true
	}
	if d.IsDir() || !strings.EqualFold(filepath.Ext(path), ".json") {
		return true
	}
	if strings.EqualFold(filepath.Base(path), filepath.Base(scaffold.GeneratedFeatureLedgerRelativePath)) {
		return true
	}
	return false
}

// loadGeneratedSchema reads and unmarshals a generated schema JSON file. The
// boolean result is false when reading or parsing fails.
func (s *DynamicModuleService) loadGeneratedSchema(path string) (generatedSchema, bool, error) {
	content, err := os.ReadFile(path) // #nosec G304 -- path comes from filepath.WalkDir under the controlled generated-schema root.
	if err != nil {
		return generatedSchema{}, false, err
	}
	var schema generatedSchema
	if err := json.Unmarshal(content, &schema); err != nil {
		return generatedSchema{}, false, err
	}
	return schema, true, nil
}

// parseGeneratedSchemaIdentity extracts the validated scope/name/tableName from
// a schema. The boolean result is false when the schema is not eligible.
func (s *DynamicModuleService) parseGeneratedSchemaIdentity(schema *generatedSchema) (string, string, string, bool) {
	scope := strings.TrimSpace(schema.Scope)
	name := strings.TrimSpace(schema.Name)
	tableName := strings.TrimSpace(schema.Model.TableName)
	if (scope != "system" && scope != "business") || name == "" || tableName == "" {
		return "", "", "", false
	}
	return scope, name, tableName, true
}

// upsertGeneratedModuleRegistration creates or patches the registration for a
// generated module based on the current schema.
func (s *DynamicModuleService) upsertGeneratedModuleRegistration(schema *generatedSchema, scope, name, tableName, moduleKey, now string) error {
	registration := s.buildGeneratedRegistration(schema, scope, name, tableName, moduleKey, now)

	var existing ModuleRegistration
	err := s.db.Where(condNameEquals, moduleKey).First(&existing).Error
	switch {
	case errors.Is(err, gorm.ErrRecordNotFound):
		return s.db.Create(&registration).Error
	case err != nil:
		return err
	default:
		return s.applyGeneratedRegistrationUpdate(existing, &registration, now)
	}
}

// buildGeneratedRegistration constructs the desired registration state for a
// generated module.
func (s *DynamicModuleService) buildGeneratedRegistration(schema *generatedSchema, scope, name, tableName, moduleKey, now string) ModuleRegistration {
	displayName := strings.TrimSpace(schema.DisplayName)
	if displayName == "" {
		displayName = moduleKey
	}
	return ModuleRegistration{
		Name:           moduleKey,
		DisplayName:    displayName,
		Scope:          scope,
		Source:         inferRegistrationSource(scope, schema.Metadata.SourceMode, schema.Name, true),
		Owner:          strings.TrimSpace(schema.Metadata.Owner),
		BoundedContext: strings.TrimSpace(schema.Metadata.BoundedContext),
		Summary:        strings.TrimSpace(schema.Metadata.Summary),
		SourceTable:    strings.TrimSpace(schema.Metadata.SourceTable),
		AutoRecycle:    schema.Metadata.AutoRecycle,
		ModelTableName: tableName,
		Status:         ModuleStatusPendingActivation,
		InstalledAt:    now,
	}
}

// applyGeneratedRegistrationUpdate patches mutable fields on an existing
// generated registration, preserving an explicit uninstall or pending state.
func (s *DynamicModuleService) applyGeneratedRegistrationUpdate(existing ModuleRegistration, registration *ModuleRegistration, now string) error {
	updates := map[string]interface{}{
		"display_name":    registration.DisplayName,
		"scope":           registration.Scope,
		"source":          registration.Source,
		"owner":           registration.Owner,
		"bounded_context": registration.BoundedContext,
		"summary":         registration.Summary,
		"source_table":    registration.SourceTable,
		"auto_recycle":    registration.AutoRecycle,
		"table_name":      registration.ModelTableName,
	}
	if existing.Status != ModuleStatusUninstalled && existing.Status != ModuleStatusPendingActivation {
		updates["status"] = ModuleStatusActive
	}
	if strings.TrimSpace(existing.InstalledAt) == "" {
		updates["installed_at"] = registration.InstalledAt
	}
	return s.db.Model(&existing).Updates(updates).Error
}

// reconcileMissingGeneratedModules marks managed generated modules whose backing
// artifacts have disappeared as uninstalled.
func (s *DynamicModuleService) reconcileMissingGeneratedModules(discovered map[string]struct{}, now string) (int, error) {
	var managedModules []ModuleRegistration
	if err := s.db.Where("table_name <> ''").Find(&managedModules).Error; err != nil {
		return 0, err
	}

	markedUninstalled := 0
	for _, module := range managedModules {
		if s.isGeneratedModuleStillPresent(&module, discovered) {
			continue
		}
		if module.Status == ModuleStatusUninstalled {
			continue
		}
		if err := s.markGeneratedModuleMissing(&module, now); err != nil {
			return 0, err
		}
		markedUninstalled++
	}
	return markedUninstalled, nil
}

// isGeneratedModuleStillPresent reports whether the module still has a discovered
// schema or live backing artifacts.
func (s *DynamicModuleService) isGeneratedModuleStillPresent(module *ModuleRegistration, discovered map[string]struct{}) bool {
	if _, found := discovered[module.Name]; found {
		return true
	}
	scope, name, err := splitModuleKey(module.Name)
	if err != nil {
		return false
	}
	return s.generatedModuleArtifactsExist(scope, name)
}

// markGeneratedModuleMissing records that a managed generated module's artifacts
// are missing by transitioning it to the uninstalled state.
func (s *DynamicModuleService) markGeneratedModuleMissing(module *ModuleRegistration, now string) error {
	scope, name, err := splitModuleKey(module.Name)
	if err != nil {
		return err
	}
	verifications := []GeneratedModuleVerification{buildArtifactMissingVerification(scope, name)}
	encoded, err := encodeModuleVerifications(verifications)
	if err != nil {
		return err
	}
	updates := map[string]interface{}{
		"status":                   ModuleStatusUninstalled,
		"uninstalled_at":           now,
		"last_verified_at":         now,
		"last_error":               "module.artifacts_missing",
		"last_verification_result": encoded,
	}
	return s.db.Model(module).Updates(updates).Error
}
