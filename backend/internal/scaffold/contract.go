package scaffold

import (
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"regexp"
	"strings"
)

var moduleRelationFieldPattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*$`)

const msgInvalidDependency = "module.generate.invalid_dependency"

const (
	msgInvalidRelation        = "module.generate.invalid_relation"
	msgInvalidDataScope       = "module.generate.invalid_data_scope"
	msgInvalidTemplateVersion = "module.generate.invalid_template_version"
)

const scopeBusiness = "business"

func validateGovernanceContract(req *RegisterGeneratedModuleRequest) error {
	if err := validateTemplateAndDataScope(req.Schema); err != nil {
		return err
	}
	if err := validateRelationMetadata(req.Schema.Metadata); err != nil {
		return err
	}
	if err := validateDependencies(req.Schema.Name, req.Schema.Dependencies); err != nil {
		return err
	}
	for _, relation := range req.Schema.Relations {
		if !isValidRelationContract(req.Schema.Scope, relation) {
			return common.NewBadRequest(msgInvalidRelation)
		}
	}
	return nil
}

func validateTemplateAndDataScope(schema ModuleSchema) error {
	templateVersion := strings.TrimSpace(schema.TemplateVersion)
	if templateVersion != "" && templateVersion != "v1" && templateVersion != "v2" {
		return common.NewBadRequest(msgInvalidTemplateVersion)
	}
	dataScopeMode := strings.TrimSpace(schema.DataScopeMode)
	if dataScopeMode == "tenant" {
		return validateTenantDataScope(schema, templateVersion)
	}
	if dataScopeMode != "" && !isValidDataScopeMode(dataScopeMode) {
		return common.NewBadRequest(msgInvalidDataScope)
	}
	return nil
}

func validateTenantDataScope(schema ModuleSchema, templateVersion string) error {
	if templateVersion != "v2" || schema.Scope != scopeBusiness {
		return common.NewBadRequest(msgInvalidDataScope)
	}
	if len(schema.Relations) != 0 || schema.Metadata.TableRole == "detail" || schema.Metadata.TableRole == "relation" ||
		schema.Metadata.PrimaryTable != "" || schema.Metadata.RelationFromField != "" || schema.Metadata.RelationToField != "" {
		return common.NewBadRequest(msgInvalidRelation)
	}
	for _, field := range schema.Model.Fields {
		if strings.EqualFold(strings.ReplaceAll(field.Name, "_", ""), "tenantid") {
			return common.NewBadRequest(msgInvalidDataScope)
		}
		if field.Type == "relation" {
			return common.NewBadRequest(msgInvalidRelation)
		}
	}
	return nil
}

func validateRelationMetadata(metadata ModuleMetadata) error {
	for _, field := range []string{
		metadata.RelationFromField,
		metadata.RelationToField,
	} {
		if field != "" && (field != strings.TrimSpace(field) || !moduleRelationFieldPattern.MatchString(field)) {
			return common.NewBadRequest(msgInvalidRelation)
		}
	}
	return nil
}

func validateDependencies(name string, dependencies []ModuleDependency) error {
	moduleName := strings.TrimSpace(name)
	seenDependencies := make(map[string]struct{}, len(dependencies))
	for _, dependency := range dependencies {
		dependencyModule := strings.TrimSpace(dependency.Module)
		if !isValidModulePath(dependencyModule, true) || dependencyModule == moduleName {
			return common.NewBadRequest(msgInvalidDependency)
		}
		if _, ok := seenDependencies[dependencyModule]; ok {
			return common.NewBadRequest(msgInvalidDependency)
		}
		seenDependencies[dependencyModule] = struct{}{}
	}
	return nil
}

// isValidDataScopeMode validates the scaffold data-scope contract.
// The runtime truth source is backend/pkg/common/data_scope.go
// (all / self / dept / dept_and_children / custom mapped to generator modes);
// Tenant mode is separately validated as an opt-in v2 business contract.
func isValidDataScopeMode(mode string) bool {
	switch strings.TrimSpace(mode) {
	case "none", "owner", "dept", "custom":
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
