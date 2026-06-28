package scaffold

import (
	"pantheon-ops/backend/pkg/common"
	"regexp"
	"strings"
)

var moduleRelationFieldPattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*$`)

func validateGovernanceContract(req *RegisterGeneratedModuleRequest) error {
	templateVersion := strings.TrimSpace(req.Schema.TemplateVersion)
	if templateVersion != "" && templateVersion != "v1" {
		return common.NewBadRequest("module.generate.invalid_template_version")
	}

	dataScopeMode := strings.TrimSpace(req.Schema.DataScopeMode)
	if dataScopeMode != "" && !isValidDataScopeMode(dataScopeMode) {
		return common.NewBadRequest("module.generate.invalid_data_scope")
	}

	moduleName := strings.TrimSpace(req.Schema.Name)
	seenDependencies := make(map[string]struct{}, len(req.Schema.Dependencies))
	for _, dependency := range req.Schema.Dependencies {
		dependencyModule := strings.TrimSpace(dependency.Module)
		if !isValidModulePath(dependencyModule, true) {
			return common.NewBadRequest("module.generate.invalid_dependency")
		}
		if dependencyModule == moduleName {
			return common.NewBadRequest("module.generate.invalid_dependency")
		}
		if _, ok := seenDependencies[dependencyModule]; ok {
			return common.NewBadRequest("module.generate.invalid_dependency")
		}
		seenDependencies[dependencyModule] = struct{}{}
	}

	for _, relation := range req.Schema.Relations {
		if !isValidRelationContract(req.Schema.Scope, relation) {
			return common.NewBadRequest("module.generate.invalid_relation")
		}
	}
	return nil
}

func isValidDataScopeMode(mode string) bool {
	switch strings.TrimSpace(mode) {
	case "none", "owner", "dept", "tenant", "custom":
		return true
	default:
		return false
	}
}

func isValidRelationContract(scope string, relation ModuleRelation) bool {
	if strings.TrimSpace(relation.Name) == "" ||
		!isValidModulePath(strings.TrimSpace(relation.TargetModule), true) ||
		!moduleRelationFieldPattern.MatchString(strings.TrimSpace(relation.LocalField)) ||
		!moduleRelationFieldPattern.MatchString(strings.TrimSpace(relation.TargetField)) {
		return false
	}
	if labelField := strings.TrimSpace(relation.TargetLabelField); labelField != "" && !moduleRelationFieldPattern.MatchString(labelField) {
		return false
	}
	if valueField := strings.TrimSpace(relation.LookupValueField); valueField != "" && !moduleRelationFieldPattern.MatchString(valueField) {
		return false
	}
	if lookupAPI := strings.TrimSpace(relation.LookupAPI); lookupAPI != "" && !strings.HasPrefix(lookupAPI, "/") {
		return false
	}

	switch strings.TrimSpace(relation.Type) {
	case "oneToMany", "lookup":
		return true
	case "manyToMany":
		return ValidateManagedTableName(scope, relation.JunctionTable) == nil
	default:
		return false
	}
}
