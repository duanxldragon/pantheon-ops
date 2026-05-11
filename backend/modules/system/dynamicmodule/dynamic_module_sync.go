package dynamicmodule

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gorm.io/gorm"

	"pantheon-ops/backend/internal/scaffold"
)

func (s *DynamicModuleService) SyncBuiltInModules() error {
	return s.syncModuleRegistrationRecords(true)
}

func (s *DynamicModuleService) syncModuleRegistrationRecords(rewriteGeneratedRegistries bool) error {
	if s.db == nil {
		return nil
	}
	if err := s.db.AutoMigrate(&ModuleRegistration{}); err != nil {
		return err
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

	type menuModuleRow struct {
		Module string
	}

	var rows []menuModuleRow
	if err := s.db.Table("system_menu").
		Select("DISTINCT module AS module").
		Where("module <> '' AND type IN ?", []string{"M", "C"}).
		Order("module ASC").
		Scan(&rows).Error; err != nil {
		return err
	}

	now := time.Now().Format(time.RFC3339)
	for _, row := range rows {
		moduleName := strings.TrimSpace(row.Module)
		if moduleName == "" {
			continue
		}

		registration := ModuleRegistration{
			Name:           moduleName,
			DisplayName:    moduleName,
			Scope:          inferModuleScope(moduleName),
			Source:         inferStaticModuleSource(moduleName),
			ModelTableName: "",
			Status:         ModuleStatusActive,
			InstalledAt:    now,
		}

		var existing ModuleRegistration
		err := s.db.Where("name = ?", moduleName).First(&existing).Error
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			if err := s.db.Create(&registration).Error; err != nil {
				return err
			}
		case err != nil:
			return err
		default:
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
				if err := s.db.Model(&existing).Updates(updates).Error; err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func (s *DynamicModuleService) AuditAndRepairGeneratedRegistries() (*RegistryRepairSummary, error) {
	if s.db == nil {
		return nil, nil
	}
	if err := s.db.AutoMigrate(&ModuleRegistration{}); err != nil {
		return nil, err
	}
	markedUninstalled, err := s.syncGeneratedModuleRegistrations()
	if err != nil {
		return nil, err
	}
	refs, err := s.listGeneratedModuleRefs()
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(s.workspaceRoot) == "" {
		return nil, errors.New("workspace.not_found")
	}
	if err := scaffold.WriteGeneratedRegistries(s.workspaceRoot, refs); err != nil {
		return nil, err
	}

	var modules []ModuleRegistration
	if err := s.db.Where("table_name <> ''").Find(&modules).Error; err != nil {
		return nil, err
	}

	summary := &RegistryRepairSummary{
		CheckedModules:           len(modules),
		GeneratedRegistryRefs:    len(refs),
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

func (s *DynamicModuleService) syncGeneratedModuleRegistrations() (int, error) {
	if strings.TrimSpace(s.workspaceRoot) == "" {
		return 0, nil
	}
	generatedSchemaRoot := filepath.Join(s.workspaceRoot, "schema", "generated")
	info, err := os.Stat(generatedSchemaRoot)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return 0, nil
		}
		return 0, err
	}
	if !info.IsDir() {
		return 0, nil
	}

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
		} `json:"metadata"`
		Model struct {
			TableName string `json:"tableName"`
		} `json:"model"`
	}

	now := time.Now().Format(time.RFC3339)
	discovered := make(map[string]struct{})
	markedUninstalled := 0

	walkErr := filepath.WalkDir(generatedSchemaRoot, func(path string, d os.DirEntry, walkErr error) error {
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

		var schema generatedSchema
		if err := json.Unmarshal(content, &schema); err != nil {
			return err
		}
		scope := strings.TrimSpace(schema.Scope)
		name := strings.TrimSpace(schema.Name)
		tableName := strings.TrimSpace(schema.Model.TableName)
		if (scope != "system" && scope != "business") || name == "" || tableName == "" {
			return nil
		}

		moduleKey := buildModuleKey(scope, name)
		if !s.generatedModuleArtifactsExist(scope, name) {
			return nil
		}
		discovered[moduleKey] = struct{}{}
		displayName := strings.TrimSpace(schema.DisplayName)
		if displayName == "" {
			displayName = moduleKey
		}

		registration := ModuleRegistration{
			Name:           moduleKey,
			DisplayName:    displayName,
			Scope:          scope,
			Source:         inferRegistrationSource(scope, schema.Metadata.SourceMode, schema.Name, true),
			Owner:          strings.TrimSpace(schema.Metadata.Owner),
			BoundedContext: strings.TrimSpace(schema.Metadata.BoundedContext),
			Summary:        strings.TrimSpace(schema.Metadata.Summary),
			SourceTable:    strings.TrimSpace(schema.Metadata.SourceTable),
			ModelTableName: tableName,
			Status:         ModuleStatusActive,
			InstalledAt:    now,
		}

		var existing ModuleRegistration
		err = s.db.Where("name = ?", moduleKey).First(&existing).Error
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			return s.db.Create(&registration).Error
		case err != nil:
			return err
		default:
			updates := map[string]interface{}{
				"display_name":    displayName,
				"scope":           scope,
				"source":          registration.Source,
				"owner":           registration.Owner,
				"bounded_context": registration.BoundedContext,
				"summary":         registration.Summary,
				"source_table":    registration.SourceTable,
				"table_name":      tableName,
			}
			if existing.Status != ModuleStatusUninstalled {
				updates["status"] = ModuleStatusActive
			}
			if strings.TrimSpace(existing.InstalledAt) == "" {
				updates["installed_at"] = registration.InstalledAt
			}
			return s.db.Model(&existing).Updates(updates).Error
		}
	})
	if walkErr != nil {
		return 0, walkErr
	}

	var managedModules []ModuleRegistration
	if err := s.db.Where("table_name <> ''").Find(&managedModules).Error; err != nil {
		return 0, err
	}
	for _, module := range managedModules {
		scope, name, err := splitModuleKey(module.Name)
		if err != nil {
			continue
		}
		_, found := discovered[module.Name]
		if found || s.generatedModuleArtifactsExist(scope, name) {
			continue
		}
		if module.Status == ModuleStatusUninstalled {
			continue
		}
		verifications := []GeneratedModuleVerification{buildArtifactMissingVerification(scope, name)}
		encoded, err := encodeModuleVerifications(verifications)
		if err != nil {
			return 0, err
		}
		updates := map[string]interface{}{
			"status":                   ModuleStatusUninstalled,
			"uninstalled_at":           now,
			"last_verified_at":         now,
			"last_error":               "module.artifacts_missing",
			"last_verification_result": encoded,
		}
		if err := s.db.Model(&module).Updates(updates).Error; err != nil {
			return 0, err
		}
		markedUninstalled++
	}
	return markedUninstalled, nil
}
