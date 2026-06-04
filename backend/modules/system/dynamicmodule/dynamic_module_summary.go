package dynamicmodule

import (
	"fmt"
	"pantheon-ops/backend/internal/scaffold"
	"path/filepath"
	"strings"
)

func (s *DynamicModuleService) buildGeneratedModuleSummary(req *scaffold.RegisterGeneratedModuleRequest, writtenFiles []string) *GeneratedModuleRegistrationSummary {
	scope := strings.TrimSpace(req.Schema.Scope)
	name := normalizeGeneratedModulePath(req.Schema.Name)
	moduleKey := buildModuleKey(scope, name)
	modelName := inferGeneratedModelName(name, req.Schema.Model.ModelName)
	routePath := "/" + scope + "/" + strings.ReplaceAll(name, "\\", "/")
	if scope == "business" {
		routePath = "/operations/" + strings.ReplaceAll(name, "\\", "/")
	}
	routeName := scope + "-" + strings.ReplaceAll(name, "/", "-")
	componentKey := scope + "/" + name + "/" + modelName + "List"
	permissionPrefix := scope + ":" + strings.ReplaceAll(name, "/", ":")
	parentMenuPath, parentMenuSource := resolveGeneratedParentMenu(scope, name, req.Schema.ParentMenu)
	parentMenuExists := s.generatedParentMenuExists(parentMenuPath)

	summary := &GeneratedModuleRegistrationSummary{
		ModuleKey:             moduleKey,
		RoutePath:             routePath,
		RouteName:             routeName,
		ComponentKey:          componentKey,
		PermissionPrefix:      permissionPrefix,
		Contract:              buildGeneratedModuleContractSummary(req.Schema),
		ParentMenuPath:        parentMenuPath,
		ParentMenuSource:      parentMenuSource,
		ParentMenuExists:      parentMenuExists,
		BackendModulePath:     filepath.ToSlash(filepath.Join("backend", "modules", scope, name)),
		FrontendModulePath:    filepath.ToSlash(filepath.Join("frontend", "src", "modules", scope, name)),
		SchemaPath:            filepath.ToSlash(filepath.Join("schema", "generated", scope, name+".json")),
		RequiresRestart:       true,
		RequiresFrontendBuild: true,
	}

	summary.Verifications = []GeneratedModuleVerification{
		s.verifyGeneratedFilesWritten(writtenFiles),
		s.verifyRegistryFile(
			"backend_registry",
			"pass",
			"module.generate.verify.backend_registry_updated",
			filepath.Join("backend", "modules", scope, "generated_registry.go"),
			fmt.Sprintf("backend/modules/%s/%s", scope, name),
			fmt.Sprintf("Init%sModule", toGeneratedPascal(name)),
		),
		s.verifyRegistryFile(
			"frontend_registry",
			"pass",
			"module.generate.verify.frontend_registry_updated",
			filepath.Join("frontend", "src", "modules", "generated", scope+".ts"),
			toGeneratedPascal(name)+"Module",
		),
		s.verifyRegistryFile(
			"frontend_component_registry",
			"pass",
			"module.generate.verify.component_registry_updated",
			filepath.Join("frontend", "src", "core", "router", "generatedComponentRegistry.ts"),
			componentKey,
		),
		s.verifyRegistryFile(
			"backend_component_registry",
			"pass",
			"module.generate.verify.backend_component_registry_updated",
			filepath.Join("backend", "modules", "system", "iam", "menu", "generated_component_registry.go"),
			componentKey,
		),
		s.verifyParentMenu(parentMenuPath, parentMenuSource, parentMenuExists),
		verifyGeneratedContract(req.Schema),
		{
			Code:       "pending_activation",
			Status:     "info",
			MessageKey: "module.generate.verify.pending_activation",
			Detail:     "status=pending_activation",
		},
		{
			Code:       "restart_required",
			Status:     "info",
			MessageKey: "module.generate.verify.restart_required",
			Detail:     "backend_restart_required=true",
		},
		{
			Code:       "frontend_build_required",
			Status:     "info",
			MessageKey: "module.generate.verify.frontend_build_required",
			Detail:     "frontend_build_required=true",
		},
	}

	return summary
}

func buildGeneratedModuleContractSummary(schema scaffold.ModuleSchema) GeneratedModuleContractSummary {
	templateVersion := strings.TrimSpace(schema.TemplateVersion)
	if templateVersion == "" {
		templateVersion = "v1"
	}
	dataScopeMode := strings.TrimSpace(schema.DataScopeMode)
	if schema.EnableDataScope && dataScopeMode == "" {
		dataScopeMode = "dept"
	}
	if !schema.EnableDataScope && dataScopeMode == "" {
		dataScopeMode = "none"
	}
	return GeneratedModuleContractSummary{
		TemplateVersion:  templateVersion,
		DataScopeEnabled: schema.EnableDataScope,
		DataScopeMode:    dataScopeMode,
		DependencyCount:  len(schema.Dependencies),
		RelationCount:    len(schema.Relations),
		Dependencies:     schema.Dependencies,
		Relations:        schema.Relations,
	}
}

func verifyGeneratedContract(schema scaffold.ModuleSchema) GeneratedModuleVerification {
	contract := buildGeneratedModuleContractSummary(schema)
	return GeneratedModuleVerification{
		Code:       "contract_governance",
		Status:     "pass",
		MessageKey: "module.generate.verify.contract_governance",
		Detail: fmt.Sprintf(
			"template=%s,data_scope=%s,dependencies=%d,relations=%d",
			contract.TemplateVersion,
			contract.DataScopeMode,
			contract.DependencyCount,
			contract.RelationCount,
		),
	}
}

func (s *DynamicModuleService) verifyGeneratedFilesWritten(writtenFiles []string) GeneratedModuleVerification {
	missing := make([]string, 0)
	for _, relativePath := range writtenFiles {
		if !generatedPathExists(s.workspaceRoot, relativePath) {
			missing = append(missing, relativePath)
		}
	}
	if len(missing) == 0 {
		return GeneratedModuleVerification{
			Code:       "source_written",
			Status:     "pass",
			MessageKey: "module.generate.verify.source_written",
			Detail:     fmt.Sprintf("%d files written", len(writtenFiles)),
		}
	}
	return GeneratedModuleVerification{
		Code:       "source_written",
		Status:     "warn",
		MessageKey: "module.generate.verify.source_write_incomplete",
		Detail:     strings.Join(missing, ", "),
	}
}

func (s *DynamicModuleService) verifyRegistryFile(code string, passStatus string, passKey string, relativePath string, fragments ...string) GeneratedModuleVerification {
	if generatedFileContainsAll(s.workspaceRoot, relativePath, fragments...) {
		return GeneratedModuleVerification{
			Code:       code,
			Status:     passStatus,
			MessageKey: passKey,
			Detail:     filepath.ToSlash(relativePath),
		}
	}
	return GeneratedModuleVerification{
		Code:       code,
		Status:     "warn",
		MessageKey: "module.generate.verify.registry_check_failed",
		Detail:     filepath.ToSlash(relativePath),
	}
}

func (s *DynamicModuleService) verifyParentMenu(parentMenuPath string, parentMenuSource string, parentMenuExists bool) GeneratedModuleVerification {
	if parentMenuSource == "top_level" {
		return GeneratedModuleVerification{
			Code:       "parent_menu",
			Status:     "info",
			MessageKey: "module.generate.verify.parent_menu_top_level",
			Detail:     "top_level",
		}
	}
	if parentMenuExists {
		return GeneratedModuleVerification{
			Code:       "parent_menu",
			Status:     "pass",
			MessageKey: "module.generate.verify.parent_menu_found",
			Detail:     parentMenuPath,
		}
	}
	return GeneratedModuleVerification{
		Code:       "parent_menu",
		Status:     "warn",
		MessageKey: "module.generate.verify.parent_menu_missing",
		Detail:     parentMenuPath,
	}
}

func (s *DynamicModuleService) generatedParentMenuExists(parentMenuPath string) bool {
	if strings.TrimSpace(parentMenuPath) == "" {
		return false
	}
	if s.db == nil || !s.db.Migrator().HasTable("system_menu") {
		return false
	}
	var count int64
	_ = s.db.Table("system_menu").Where("path = ?", parentMenuPath).Count(&count).Error
	return count > 0
}
