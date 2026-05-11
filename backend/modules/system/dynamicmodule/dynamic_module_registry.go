package dynamicmodule

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"

	"pantheon-ops/backend/internal/scaffold"
)

func (s *DynamicModuleService) RebuildGeneratedRegistries() error {
	if strings.TrimSpace(s.workspaceRoot) == "" {
		return errors.New("workspace.not_found")
	}
	if _, err := s.syncGeneratedModuleRegistrations(); err != nil {
		return err
	}
	refs, err := s.listGeneratedModuleRefs()
	if err != nil {
		return err
	}
	return scaffold.WriteGeneratedRegistries(s.workspaceRoot, refs)
}

func (s *DynamicModuleService) listGeneratedModuleRefs() ([]scaffold.GeneratedModuleRef, error) {
	var modules []ModuleRegistration
	if err := s.db.Where("table_name <> '' AND status <> ?", ModuleStatusUninstalled).Find(&modules).Error; err != nil {
		return nil, err
	}
	refs := make([]scaffold.GeneratedModuleRef, 0, len(modules))
	for _, module := range modules {
		scope, name, err := splitModuleKey(module.Name)
		if err != nil {
			continue
		}
		if !s.generatedModuleArtifactsExist(scope, name) {
			continue
		}
		refs = append(refs, scaffold.GeneratedModuleRef{Name: name, Scope: scope})
	}
	return refs, nil
}

func (s *DynamicModuleService) generatedModuleArtifactsExist(scope string, name string) bool {
	if strings.TrimSpace(s.workspaceRoot) == "" {
		return false
	}
	return generatedDirExists(filepath.Join(s.workspaceRoot, "backend", "modules", scope, name)) &&
		generatedDirExists(filepath.Join(s.workspaceRoot, "frontend", "src", "modules", scope, name)) &&
		generatedPathExists(filepath.Join(s.workspaceRoot, "schema", "generated", scope, name+".json"))
}

func (s *DynamicModuleService) loadGeneratedModuleSchema(scope string, name string) (*scaffold.ModuleSchema, error) {
	target := filepath.Join(s.workspaceRoot, "schema", "generated", scope, name+".json")
	content, err := os.ReadFile(target)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, errors.New("module.register.source_missing")
		}
		return nil, err
	}
	var schema scaffold.ModuleSchema
	if err := json.Unmarshal(content, &schema); err != nil {
		return nil, errors.New("module.register.schema_invalid")
	}
	if strings.TrimSpace(schema.Name) == "" || strings.TrimSpace(schema.Scope) == "" || strings.TrimSpace(schema.Model.TableName) == "" {
		return nil, errors.New("module.register.schema_invalid")
	}
	return &schema, nil
}

func generatedPathExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func generatedDirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func generatedFileContainsAll(path string, fragments ...string) bool {
	content, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	body := string(content)
	for _, fragment := range fragments {
		if strings.TrimSpace(fragment) == "" {
			continue
		}
		if !strings.Contains(body, fragment) {
			return false
		}
	}
	return true
}
