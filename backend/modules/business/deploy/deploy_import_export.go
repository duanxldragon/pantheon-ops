package deploy

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/impexp"
	"gorm.io/gorm"
)

var deployPackageCSVHeaders = []string{"name", "version", "description", "installCommand", "uninstallCommand", "executionMode", "templateCode", "templateConfig", "sourceObjectKey", "sourceFileName", "sourceUrl", "status"}
var deployTemplateCSVHeaders = []string{"name", "version", "description", "category", "executionMode", "defaultAction", "packageId", "templateCode", "templateConfig", "parameterSchema", "status", "steps"}
var deployTaskCSVHeaders = []string{"id", "name", "templateName", "templateVersion", "packageName", "packageVersion", "businessScopeName", "serviceName", "serviceInstanceName", "action", "targetType", "executorType", "status", "startedAt", "finishedAt", "remark"}

// ExportPackages builds a CSV file for deploy packages.
func (s *DeployService) ExportPackages(query PackageQuery) (*impexp.CSVFile, error) {
	list, err := s.ListPackages(query)
	if err != nil {
		return nil, err
	}
	rows := make([][]string, 0, len(list.Items))
	for _, p := range list.Items {
		config, _ := json.Marshal(p.TemplateConfig)
		rows = append(rows, []string{p.Name, p.Version, p.Description, p.InstallCommand, p.UninstallCommand, p.ExecutionMode, p.TemplateCode, string(config), p.SourceObjectKey, p.SourceFileName, p.SourceURL, p.Status})
	}
	return &impexp.CSVFile{Filename: "deploy-packages.csv", Headers: deployPackageCSVHeaders, Rows: rows}, nil
}

// ExportTemplates builds a CSV file for deploy templates.
func (s *DeployService) ExportTemplates(query TemplateQuery) (*impexp.CSVFile, error) {
	list, err := s.ListTemplates(query)
	if err != nil {
		return nil, err
	}
	rows := make([][]string, 0, len(list.Items))
	for _, t := range list.Items {
		config, _ := json.Marshal(t.TemplateConfig)
		schema, _ := json.Marshal(t.ParameterSchema)
		steps, _ := json.Marshal(t.Steps)
		rows = append(rows, []string{t.Name, t.Version, t.Description, t.Category, t.ExecutionMode, t.DefaultAction, fmt.Sprint(t.PackageID), t.TemplateCode, string(config), string(schema), t.Status, string(steps)})
	}
	return &impexp.CSVFile{Filename: "deploy-templates.csv", Headers: deployTemplateCSVHeaders, Rows: rows}, nil
}

// ExportTasks builds a CSV file for deploy tasks.
func (s *DeployService) ExportTasks(query TaskQuery, scope *common.DataScopeReq) (*impexp.CSVFile, error) {
	list, err := s.ListTasks(query, scope)
	if err != nil {
		return nil, err
	}
	rows := make([][]string, 0, len(list.Items))
	for _, t := range list.Items {
		rows = append(rows, []string{fmt.Sprint(t.ID), t.Name, t.TemplateName, t.TemplateVersion, t.PackageName, t.PackageVersion, t.BusinessScopeName, t.ServiceName, t.ServiceInstanceName, t.Action, t.TargetType, t.ExecutorType, t.Status, formatDeployTime(t.StartedAt), formatDeployTime(t.FinishedAt), t.Remark})
	}
	return &impexp.CSVFile{Filename: "deploy-tasks.csv", Headers: deployTaskCSVHeaders, Rows: rows}, nil
}

func formatDeployTime(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.Format(time.RFC3339)
}

// ImportPackages validates and applies package CSV records transactionally.
func (s *DeployService) ImportPackages(records [][]string, actor string) (*impexp.ImportResult, error) {
	result := &impexp.ImportResult{Errors: []impexp.ImportError{}}
	if len(records) == 0 {
		impexp.AppendImportError(result, 0, "file", "import.file.empty")
		return result, nil
	}
	index := csvHeaderIndex(records[0])
	if !requireHeaders(result, index, deployPackageCSVHeaders[:2]) {
		return result, nil
	}
	return result, s.db.Transaction(func(tx *gorm.DB) error {
		return importPackageCSVRows(tx, records[1:], index, actor, result)
	})
}

// importPackageCSVRows applies the package data rows of an uploaded CSV. A
// failed row validation stops the remaining rows from being applied.
func importPackageCSVRows(tx *gorm.DB, records [][]string, index map[string]int, actor string, result *impexp.ImportResult) error {
	for rowNo, record := range records {
		if impexp.IsCSVRecordEmpty(record) {
			continue
		}
		row := parsePackageCSV(record, index, result, rowNo+2)
		if result.Failed > 0 {
			continue
		}
		if err := upsertPackageCSVRow(tx, row, actor, result); err != nil {
			return err
		}
	}
	result.Applied = result.Failed == 0
	return nil
}

// upsertPackageCSVRow updates the package identified by name and version, or
// creates it when it does not exist yet.
func upsertPackageCSVRow(tx *gorm.DB, row packageCSVRow, actor string, result *impexp.ImportResult) error {
	var item DeployPackage
	err := tx.Where("name = ? AND version = ?", row.Name, row.Version).First(&item).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return err
	}
	if err == nil {
		if err := tx.Model(&item).Updates(row.updates(actor)).Error; err != nil {
			return err
		}
		result.Updated++
		return nil
	}
	if err := tx.Create(&DeployPackage{Name: row.Name, Version: row.Version, Description: row.Description, InstallCommand: row.InstallCommand, UninstallCommand: row.UninstallCommand, ExecutionMode: row.ExecutionMode, TemplateCode: row.TemplateCode, TemplateConfig: row.TemplateConfig, SourceObjectKey: row.SourceObjectKey, SourceFileName: row.SourceFileName, SourceURL: row.SourceURL, Status: row.Status, CreatedBy: actor, UpdatedBy: actor}).Error; err != nil {
		return err
	}
	result.Created++
	return nil
}

type packageCSVRow struct {
	Name, Version, Description, InstallCommand, UninstallCommand, ExecutionMode, TemplateCode, SourceObjectKey, SourceFileName, SourceURL, Status string
	TemplateConfig                                                                                                                                []byte
}

func (r packageCSVRow) updates(actor string) map[string]any {
	return map[string]any{"description": r.Description, "install_command": r.InstallCommand, "uninstall_command": r.UninstallCommand, "execution_mode": r.ExecutionMode, "template_code": r.TemplateCode, "template_config": r.TemplateConfig, "source_object_key": r.SourceObjectKey, "source_file_name": r.SourceFileName, "source_url": r.SourceURL, "status": r.Status, "updated_by": actor}
}
func parsePackageCSV(record []string, index map[string]int, result *impexp.ImportResult, row int) packageCSVRow {
	r := packageCSVRow{Name: strings.TrimSpace(impexp.ReadCSVField(record, index, "name")), Version: strings.TrimSpace(impexp.ReadCSVField(record, index, "version")), Description: impexp.ReadCSVField(record, index, "description"), InstallCommand: impexp.ReadCSVField(record, index, "installCommand"), UninstallCommand: impexp.ReadCSVField(record, index, "uninstallCommand"), ExecutionMode: strings.TrimSpace(impexp.ReadCSVField(record, index, "executionMode")), TemplateCode: strings.TrimSpace(impexp.ReadCSVField(record, index, "templateCode")), SourceObjectKey: impexp.ReadCSVField(record, index, "sourceObjectKey"), SourceFileName: impexp.ReadCSVField(record, index, "sourceFileName"), SourceURL: impexp.ReadCSVField(record, index, "sourceUrl"), Status: strings.TrimSpace(impexp.ReadCSVField(record, index, "status"))}
	if r.Name == "" {
		impexp.AppendImportError(result, row, "name", "deploypackage.name_required")
	}
	if r.Version == "" {
		impexp.AppendImportError(result, row, "version", "deploypackage.version_required")
	}
	if r.ExecutionMode == "" {
		r.ExecutionMode = ExecutionModeFixed
	}
	if r.Status == "" {
		r.Status = PackageStatusEnabled
	}
	raw := strings.TrimSpace(impexp.ReadCSVField(record, index, "templateConfig"))
	if raw != "" && !json.Valid([]byte(raw)) {
		impexp.AppendImportError(result, row, "templateConfig", "deploypackage.template_config_invalid")
	} else if raw != "" {
		r.TemplateConfig = []byte(raw)
	}
	return r
}

func csvHeaderIndex(headers []string) map[string]int {
	index := make(map[string]int, len(headers))
	for i, h := range headers {
		index[strings.TrimSpace(h)] = i
	}
	return index
}
func requireHeaders(result *impexp.ImportResult, index map[string]int, headers []string) bool {
	ok := true
	for _, h := range headers {
		if _, exists := index[h]; !exists {
			impexp.AppendImportError(result, 0, h, "import.header.missing")
			ok = false
		}
	}
	return ok
}

// ImportTemplates validates and applies template CSV records transactionally.
func (s *DeployService) ImportTemplates(records [][]string, actor string) (*impexp.ImportResult, error) {
	result := &impexp.ImportResult{Errors: []impexp.ImportError{}}
	if len(records) == 0 {
		impexp.AppendImportError(result, 0, "file", "import.file.empty")
		return result, nil
	}
	index := csvHeaderIndex(records[0])
	if !requireHeaders(result, index, deployTemplateCSVHeaders[:2]) {
		return result, nil
	}
	return result, s.db.Transaction(func(tx *gorm.DB) error {
		return importTemplateCSVRows(tx, records[1:], index, actor, result)
	})
}

// importTemplateCSVRows applies the template data rows of an uploaded CSV. A
// failed row validation stops the remaining rows from being applied.
func importTemplateCSVRows(tx *gorm.DB, records [][]string, index map[string]int, actor string, result *impexp.ImportResult) error {
	for rowNo, record := range records {
		if impexp.IsCSVRecordEmpty(record) {
			continue
		}
		row := parseTemplateCSV(record, index, result, rowNo+2)
		if result.Failed > 0 {
			continue
		}
		steps, ok := readTemplateCSVSteps(record, index, result, rowNo+2)
		if !ok {
			continue
		}
		row.Steps = steps
		if err := upsertTemplateCSVRow(tx, row, actor, result); err != nil {
			return err
		}
	}
	result.Applied = result.Failed == 0
	return nil
}

type templateCSVRow struct {
	Name, Version, Description, Category, ExecutionMode, DefaultAction, TemplateCode, Status string
	Steps                                                                                    []TemplateStepPayload
}

func (r templateCSVRow) updates(actor string) map[string]any {
	return map[string]any{"description": r.Description, "category": r.Category, "execution_mode": r.ExecutionMode, "default_action": r.DefaultAction, "template_code": r.TemplateCode, "status": r.Status, "updated_by": actor}
}

// parseTemplateCSV reads the template columns of one CSV record and reports the
// required-key errors.
func parseTemplateCSV(record []string, index map[string]int, result *impexp.ImportResult, row int) templateCSVRow {
	r := templateCSVRow{
		Name:          strings.TrimSpace(impexp.ReadCSVField(record, index, "name")),
		Version:       strings.TrimSpace(impexp.ReadCSVField(record, index, "version")),
		Description:   impexp.ReadCSVField(record, index, "description"),
		Category:      impexp.ReadCSVField(record, index, "category"),
		ExecutionMode: impexp.ReadCSVField(record, index, "executionMode"),
		DefaultAction: impexp.ReadCSVField(record, index, "defaultAction"),
		TemplateCode:  impexp.ReadCSVField(record, index, "templateCode"),
		Status:        impexp.ReadCSVField(record, index, "status"),
	}
	if r.Name == "" {
		impexp.AppendImportError(result, row, "name", "deploytemplate.name_required")
	}
	if r.Version == "" {
		impexp.AppendImportError(result, row, "version", "deploytemplate.version_required")
	}
	return r
}

// readTemplateCSVSteps decodes the JSON step payload of one CSV record. It
// returns false when the payload is invalid and the row must be skipped.
func readTemplateCSVSteps(record []string, index map[string]int, result *impexp.ImportResult, row int) ([]TemplateStepPayload, bool) {
	rawSteps := impexp.ReadCSVField(record, index, "steps")
	if rawSteps == "" {
		return nil, true
	}
	var steps []TemplateStepPayload
	if json.Unmarshal([]byte(rawSteps), &steps) != nil {
		impexp.AppendImportError(result, row, "steps", "deploytemplate.steps_invalid")
		return nil, false
	}
	return steps, true
}

// upsertTemplateCSVRow updates the template identified by name and version, or
// creates it when it does not exist yet, and rewrites its steps.
func upsertTemplateCSVRow(tx *gorm.DB, row templateCSVRow, actor string, result *impexp.ImportResult) error {
	var item DeployTemplate
	err := tx.Where("name = ? AND version = ?", row.Name, row.Version).First(&item).Error
	switch {
	case err == nil:
		if err := tx.Model(&item).Updates(row.updates(actor)).Error; err != nil {
			return err
		}
		result.Updated++
	case errors.Is(err, gorm.ErrRecordNotFound):
		item = DeployTemplate{Name: row.Name, Version: row.Version, Description: row.Description, Category: row.Category, ExecutionMode: row.ExecutionMode, DefaultAction: row.DefaultAction, TemplateCode: row.TemplateCode, Status: row.Status, CreatedBy: actor, UpdatedBy: actor}
		if item.ExecutionMode == "" {
			item.ExecutionMode = ExecutionModeFixed
		}
		if item.Status == "" {
			item.Status = TemplateStatusEnabled
		}
		if err := tx.Create(&item).Error; err != nil {
			return err
		}
		result.Created++
	default:
		return err
	}
	return replaceTemplateCSVSteps(tx, item.ID, row.Steps)
}

// replaceTemplateCSVSteps rewrites the persisted step rows of a template; an
// empty payload keeps the existing steps untouched.
func replaceTemplateCSVSteps(tx *gorm.DB, templateID uint64, steps []TemplateStepPayload) error {
	if len(steps) == 0 {
		return nil
	}
	if err := tx.Where("template_id = ?", templateID).Delete(&DeployTemplateStep{}).Error; err != nil {
		return err
	}
	for i := range steps {
		if err := tx.Create(&DeployTemplateStep{TemplateID: templateID, StepCode: steps[i].StepCode, StepName: steps[i].StepName, StepType: steps[i].StepType, Action: steps[i].Action, PackageID: steps[i].PackageID, PackageName: steps[i].PackageName, PackageVersion: steps[i].PackageVersion, TemplateCode: steps[i].TemplateCode, Sort: steps[i].Sort}).Error; err != nil {
			return err
		}
	}
	return nil
}
