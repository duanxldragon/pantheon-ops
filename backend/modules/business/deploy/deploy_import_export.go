package deploy

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
	"pantheon-base/pkg/common"
	"pantheon-base/pkg/impexp"
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
		for rowNo, record := range records[1:] {
			if impexp.IsCSVRecordEmpty(record) {
				continue
			}
			row := parsePackageCSV(record, index, result, rowNo+2)
			if result.Failed > 0 {
				continue
			}
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
			} else {
				if err := tx.Create(&DeployPackage{Name: row.Name, Version: row.Version, Description: row.Description, InstallCommand: row.InstallCommand, UninstallCommand: row.UninstallCommand, ExecutionMode: row.ExecutionMode, TemplateCode: row.TemplateCode, TemplateConfig: row.TemplateConfig, SourceObjectKey: row.SourceObjectKey, SourceFileName: row.SourceFileName, SourceURL: row.SourceURL, Status: row.Status, CreatedBy: actor, UpdatedBy: actor}).Error; err != nil {
					return err
				}
				result.Created++
			}
		}
		result.Applied = result.Failed == 0
		return nil
	})
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
		for rowNo, record := range records[1:] {
			if impexp.IsCSVRecordEmpty(record) {
				continue
			}
			name := strings.TrimSpace(impexp.ReadCSVField(record, index, "name"))
			version := strings.TrimSpace(impexp.ReadCSVField(record, index, "version"))
			if name == "" {
				impexp.AppendImportError(result, rowNo+2, "name", "deploytemplate.name_required")
			}
			if version == "" {
				impexp.AppendImportError(result, rowNo+2, "version", "deploytemplate.version_required")
			}
			if result.Failed > 0 {
				continue
			}
			rawSteps := impexp.ReadCSVField(record, index, "steps")
			var steps []TemplateStepPayload
			if rawSteps != "" && json.Unmarshal([]byte(rawSteps), &steps) != nil {
				impexp.AppendImportError(result, rowNo+2, "steps", "deploytemplate.steps_invalid")
				continue
			}
			var item DeployTemplate
			err := tx.Where("name = ? AND version = ?", name, version).First(&item).Error
			values := map[string]any{"description": impexp.ReadCSVField(record, index, "description"), "category": impexp.ReadCSVField(record, index, "category"), "execution_mode": impexp.ReadCSVField(record, index, "executionMode"), "default_action": impexp.ReadCSVField(record, index, "defaultAction"), "template_code": impexp.ReadCSVField(record, index, "templateCode"), "status": impexp.ReadCSVField(record, index, "status"), "updated_by": actor}
			if err == nil {
				if err := tx.Model(&item).Updates(values).Error; err != nil {
					return err
				}
				result.Updated++
			} else if errors.Is(err, gorm.ErrRecordNotFound) {
				item = DeployTemplate{Name: name, Version: version, Description: values["description"].(string), Category: values["category"].(string), ExecutionMode: values["execution_mode"].(string), DefaultAction: values["default_action"].(string), TemplateCode: values["template_code"].(string), Status: values["status"].(string), CreatedBy: actor, UpdatedBy: actor}
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
			} else {
				return err
			}
			if len(steps) > 0 {
				if err := tx.Where("template_id = ?", item.ID).Delete(&DeployTemplateStep{}).Error; err != nil {
					return err
				}
				for i := range steps {
					if err := tx.Create(&DeployTemplateStep{TemplateID: item.ID, StepCode: steps[i].StepCode, StepName: steps[i].StepName, StepType: steps[i].StepType, Action: steps[i].Action, PackageID: steps[i].PackageID, PackageName: steps[i].PackageName, PackageVersion: steps[i].PackageVersion, TemplateCode: steps[i].TemplateCode, Sort: steps[i].Sort}).Error; err != nil {
						return err
					}
				}
			}
		}
		result.Applied = result.Failed == 0
		return nil
	})
}
