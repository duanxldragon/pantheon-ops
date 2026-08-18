package host

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	bizcap "pantheon-base/modules/business/capability"
	"pantheon-base/pkg/common"
	"pantheon-base/pkg/impexp"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// maxHostExportRows 限制单次主机资产导出的最大行数（var 便于测试降低阈值）。
var maxHostExportRows = 10000

var hostImportStatuses = map[string]struct{}{
	"pending":     {},
	"assigned":    {},
	"online":      {},
	"offline":     {},
	"maintenance": {},
}

type hostImportRow struct {
	Hostname          string
	IP                string
	SSHPort           int
	OS                string
	OSVersion         string
	CPUCores          int
	MemoryGB          float64
	DiskGB            float64
	Labels            []LabelEntry
	BusinessScopeID   uint64
	BusinessScopeCode string
	BusinessScopeName string
	DeptID            uint64
	Owner             string
	Status            string
	Remark            string
	Existing          *Host
}

func hostImportExportHeaders() []string {
	return []string{
		"hostname",
		"ip",
		"sshPort",
		"os",
		"osVersion",
		"cpuCores",
		"memoryGb",
		"diskGb",
		"labels",
		"businessScopeCode",
		"deptId",
		"owner",
		"status",
		"remark",
	}
}

// Export 导出当前数据范围与筛选条件内的主机资产。
func (s *HostService) Export(query HostListQuery, dataScope *common.DataScopeReq) (*impexp.CSVFile, error) {
	if s.db == nil {
		return nil, common.NewBadRequest("database.not_initialized")
	}

	db := s.hostQuery(dataScope)
	if keyword := strings.TrimSpace(query.Keyword); keyword != "" {
		like := "%" + common.EscapeLikePattern(keyword) + "%"
		db = db.Where("hostname LIKE ? OR ip LIKE ?", like, like)
	}
	if query.Status != "" {
		db = db.Where("status = ?", query.Status)
	}
	if query.OS != "" {
		db = db.Where("os = ?", query.OS)
	}
	if query.BusinessScopeID > 0 {
		db = db.Where("business_scope_id = ?", query.BusinessScopeID)
	}
	if query.DeptID > 0 {
		db = db.Where("dept_id = ?", query.DeptID)
	}

	var hosts []Host
	if err := db.Order("id DESC").Limit(maxHostExportRows).Find(&hosts).Error; err != nil {
		return nil, err
	}

	rows := make([][]string, 0, len(hosts))
	for i := range hosts {
		rows = append(rows, []string{
			hosts[i].Hostname,
			hosts[i].IP,
			strconv.FormatInt(int64(hosts[i].SSHPort), 10),
			hosts[i].OS,
			hosts[i].OSVersion,
			strconv.FormatInt(int64(hosts[i].CPUCores), 10),
			strconv.FormatFloat(hosts[i].MemoryGB, 'f', -1, 64),
			strconv.FormatFloat(hosts[i].DiskGB, 'f', -1, 64),
			formatHostExportLabels(hosts[i].LabelValues),
			hosts[i].BusinessScopeCode,
			strconv.FormatUint(hosts[i].DeptID, 10),
			hosts[i].Owner,
			hosts[i].Status,
			hosts[i].Remark,
		})
	}

	return &impexp.CSVFile{
		Filename: "cmdb-host-export.csv",
		Headers:  hostImportExportHeaders(),
		Rows:     rows,
	}, nil
}

func formatHostExportLabels(raw datatypes.JSON) string {
	if len(raw) == 0 {
		return ""
	}
	var labels []LabelEntry
	if err := json.Unmarshal(raw, &labels); err != nil {
		return ""
	}
	items := make([]string, 0, len(labels))
	for _, label := range labels {
		items = append(items, label.Key+"="+label.Val)
	}
	return strings.Join(items, "; ")
}

// BuildImportTemplate 构建主机导入 CSV 模板。
func (s *HostService) BuildImportTemplate() *impexp.CSVFile {
	return &impexp.CSVFile{
		Filename: "cmdb-host-import-template.csv",
		Headers:  hostImportExportHeaders(),
		Rows: [][]string{
			{"#说明：保留第一行表头；hostname/ip/os 必填；labels 多个用 \"; \" 分隔（key=val）；businessScopeCode 对应业务域编码；status 取值 pending/assigned/online/offline/maintenance，留空默认 pending。", "", "", "", "", "", "", "", "", "", "", "", "", ""},
			{"web-01", "10.0.0.1", "22", "linux", "Ubuntu 22.04", "4", "8", "100", "env=prod; region=cn-east-1", "volcano-prod", "0", "ops", "pending", ""},
		},
	}
}

// Import 解析并校验主机 CSV，在全部记录合法后通过单一事务创建或更新主机。
func (s *HostService) Import(records [][]string, dataScope *common.DataScopeReq, createdBy string) (*impexp.ImportResult, error) {
	result := &impexp.ImportResult{
		Applied: false,
		Errors:  []impexp.ImportError{},
	}
	if s.db == nil {
		return nil, common.ErrDatabaseNotInitialized
	}
	if len(records) == 0 {
		impexp.AppendImportError(result, 0, "file", "import.file.empty")
		return result, nil
	}

	headerIndex := make(map[string]int, len(records[0]))
	for index, header := range records[0] {
		headerIndex[strings.TrimSpace(header)] = index
	}
	for _, header := range []string{"hostname", "ip", "os"} {
		if _, ok := headerIndex[header]; !ok {
			impexp.AppendImportError(result, 0, header, "import.header.missing")
		}
	}
	if result.Failed > 0 {
		return result, nil
	}

	existingByIP, err := s.loadHostsForImport(dataScope)
	if err != nil {
		return nil, err
	}
	scopeCodes := extractHostImportScopeCodes(records, headerIndex)
	scopesByCode, err := s.loadHostImportScopes(scopeCodes, dataScope)
	if err != nil {
		return nil, err
	}

	rows := make([]hostImportRow, 0, len(records)-1)
	seenIP := make(map[string]int, len(records)-1)
	for rowIndex := 1; rowIndex < len(records); rowIndex++ {
		record := records[rowIndex]
		if impexp.IsCSVRecordEmpty(record) {
			continue
		}
		rowNumber := rowIndex + 1
		row := parseHostImportRow(record, headerIndex, rowNumber, result, scopesByCode)
		if firstRow, ok := seenIP[row.IP]; ok && row.IP != "" {
			impexp.AppendImportError(result, rowNumber, "ip", fmt.Sprintf("import.duplicate.row.%d", firstRow))
		} else if row.IP != "" {
			seenIP[row.IP] = rowNumber
		}
		row.Existing = existingByIP[row.IP]
		rows = append(rows, row)
	}

	if result.Failed > 0 {
		return result, nil
	}
	if err := s.applyHostImportRows(result, rows, createdBy); err != nil {
		return nil, err
	}

	result.Applied = true
	return result, nil
}

func (s *HostService) loadHostsForImport(dataScope *common.DataScopeReq) (map[string]*Host, error) {
	var hosts []Host
	if err := s.hostQuery(dataScope).Find(&hosts).Error; err != nil {
		return nil, err
	}
	existingByIP := make(map[string]*Host, len(hosts))
	for i := range hosts {
		existingByIP[hosts[i].IP] = &hosts[i]
	}
	return existingByIP, nil
}

func extractHostImportScopeCodes(records [][]string, headerIndex map[string]int) []string {
	codes := make([]string, 0)
	seen := make(map[string]struct{})
	for rowIndex := 1; rowIndex < len(records); rowIndex++ {
		code := strings.TrimSpace(impexp.ReadCSVField(records[rowIndex], headerIndex, "businessScopeCode"))
		if code == "" {
			continue
		}
		if _, ok := seen[code]; ok {
			continue
		}
		seen[code] = struct{}{}
		codes = append(codes, code)
	}
	return codes
}

func (s *HostService) loadHostImportScopes(codes []string, dataScope *common.DataScopeReq) (map[string]bizcap.BizScopeRef, error) {
	if len(codes) == 0 {
		return map[string]bizcap.BizScopeRef{}, nil
	}
	if s.bizScopeReader == nil {
		return nil, fmt.Errorf("business.bizscope.readerNotConfigured")
	}
	return s.bizScopeReader.ResolveActiveByCodes(context.Background(), codes, dataScope)
}

func parseHostImportRow(
	record []string,
	headerIndex map[string]int,
	rowNumber int,
	result *impexp.ImportResult,
	scopesByCode map[string]bizcap.BizScopeRef,
) hostImportRow {
	row := hostImportRow{
		Hostname:  strings.TrimSpace(impexp.ReadCSVField(record, headerIndex, "hostname")),
		IP:        strings.TrimSpace(impexp.ReadCSVField(record, headerIndex, "ip")),
		OS:        strings.TrimSpace(impexp.ReadCSVField(record, headerIndex, "os")),
		OSVersion: strings.TrimSpace(impexp.ReadCSVField(record, headerIndex, "osVersion")),
		Owner:     strings.TrimSpace(impexp.ReadCSVField(record, headerIndex, "owner")),
		Status:    strings.TrimSpace(impexp.ReadCSVField(record, headerIndex, "status")),
		Remark:    strings.TrimSpace(impexp.ReadCSVField(record, headerIndex, "remark")),
		Labels:    []LabelEntry{},
	}

	if row.Hostname == "" {
		impexp.AppendImportError(result, rowNumber, "hostname", "cmdbhost.hostname_required")
	}
	if row.IP == "" {
		impexp.AppendImportError(result, rowNumber, "ip", "cmdbhost.ip_required")
	}
	if row.OS == "" {
		impexp.AppendImportError(result, rowNumber, "os", "cmdbhost.os_required")
	}

	sshPortRaw := strings.TrimSpace(impexp.ReadCSVField(record, headerIndex, "sshPort"))
	row.SSHPort = parseHostImportInt(sshPortRaw, rowNumber, "sshPort", result)
	if sshPortRaw == "" {
		row.SSHPort = 22
	}
	row.CPUCores = parseHostImportInt(
		impexp.ReadCSVField(record, headerIndex, "cpuCores"),
		rowNumber,
		"cpuCores",
		result,
	)
	row.MemoryGB = parseHostImportFloat(
		impexp.ReadCSVField(record, headerIndex, "memoryGb"),
		rowNumber,
		"memoryGb",
		result,
	)
	row.DiskGB = parseHostImportFloat(
		impexp.ReadCSVField(record, headerIndex, "diskGb"),
		rowNumber,
		"diskGb",
		result,
	)
	row.Labels = parseHostImportLabels(
		impexp.ReadCSVField(record, headerIndex, "labels"),
		rowNumber,
		result,
	)

	scopeCode := strings.TrimSpace(impexp.ReadCSVField(record, headerIndex, "businessScopeCode"))
	if scopeCode != "" {
		scope, ok := scopesByCode[scopeCode]
		if !ok {
			impexp.AppendImportError(result, rowNumber, "businessScopeCode", "business.bizscope.notFound")
		} else {
			row.BusinessScopeID = scope.ID
			row.BusinessScopeCode = scope.Code
			row.BusinessScopeName = scope.Name
		}
	}

	deptID := parseHostImportInt(
		impexp.ReadCSVField(record, headerIndex, "deptId"),
		rowNumber,
		"deptId",
		result,
	)
	if deptID < 0 {
		impexp.AppendImportError(result, rowNumber, "deptId", "param.invalid")
	} else {
		row.DeptID = uint64(deptID)
	}

	if row.Status != "" {
		if _, ok := hostImportStatuses[row.Status]; !ok {
			impexp.AppendImportError(result, rowNumber, "status", "cmdbhost.status.invalid")
		}
	}
	return row
}

func parseHostImportInt(raw string, rowNumber int, field string, result *impexp.ImportResult) int {
	value, err := impexp.ParseCSVInt(raw)
	if err != nil {
		impexp.AppendImportError(result, rowNumber, field, "param.invalid")
		return 0
	}
	return value
}

func parseHostImportFloat(raw string, rowNumber int, field string, result *impexp.ImportResult) float64 {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return 0
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		impexp.AppendImportError(result, rowNumber, field, "param.invalid")
		return 0
	}
	return value
}

func parseHostImportLabels(raw string, rowNumber int, result *impexp.ImportResult) []LabelEntry {
	labels := make([]LabelEntry, 0)
	for _, item := range strings.FieldsFunc(raw, func(r rune) bool {
		return r == ';' || r == ','
	}) {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		key, value, ok := strings.Cut(item, "=")
		if !ok {
			impexp.AppendImportError(result, rowNumber, "labels", "param.invalid")
			continue
		}
		labels = append(labels, LabelEntry{
			Key: strings.TrimSpace(key),
			Val: strings.TrimSpace(value),
		})
	}
	return labels
}

func (s *HostService) applyHostImportRows(result *impexp.ImportResult, rows []hostImportRow, createdBy string) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		for i := range rows {
			labelsJSON, err := json.Marshal(rows[i].Labels)
			if err != nil {
				return err
			}
			if rows[i].Existing != nil {
				updates := map[string]interface{}{
					"hostname":            rows[i].Hostname,
					"os":                  rows[i].OS,
					"os_version":          rows[i].OSVersion,
					"ssh_port":            rows[i].SSHPort,
					"cpu_cores":           rows[i].CPUCores,
					"memory_gb":           rows[i].MemoryGB,
					"disk_gb":             rows[i].DiskGB,
					"label_values":        datatypes.JSON(labelsJSON),
					"business_scope_id":   rows[i].BusinessScopeID,
					"business_scope_code": rows[i].BusinessScopeCode,
					"business_scope_name": rows[i].BusinessScopeName,
					"dept_id":             rows[i].DeptID,
					"owner":               rows[i].Owner,
					"remark":              rows[i].Remark,
					"updated_by":          createdBy,
					"updated_at":          time.Now(),
				}
				if rows[i].Status != "" {
					updates["status"] = rows[i].Status
				}
				if err := tx.Model(rows[i].Existing).Updates(updates).Error; err != nil {
					return err
				}
				result.Updated++
				continue
			}

			status := rows[i].Status
			if status == "" {
				status = "pending"
				if rows[i].BusinessScopeID > 0 {
					status = "assigned"
				}
			}
			host := Host{
				Hostname:          rows[i].Hostname,
				IP:                rows[i].IP,
				SSHPort:           rows[i].SSHPort,
				OS:                rows[i].OS,
				OSVersion:         rows[i].OSVersion,
				CPUCores:          rows[i].CPUCores,
				MemoryGB:          rows[i].MemoryGB,
				DiskGB:            rows[i].DiskGB,
				LabelValues:       datatypes.JSON(labelsJSON),
				BusinessScopeID:   rows[i].BusinessScopeID,
				BusinessScopeCode: rows[i].BusinessScopeCode,
				BusinessScopeName: rows[i].BusinessScopeName,
				DeptID:            rows[i].DeptID,
				Owner:             rows[i].Owner,
				Status:            status,
				Remark:            rows[i].Remark,
				CreatedBy:         createdBy,
				UpdatedBy:         createdBy,
			}
			if host.SSHPort == 0 {
				host.SSHPort = 22
			}
			if err := tx.Create(&host).Error; err != nil {
				return err
			}
			result.Created++
		}
		return nil
	})
}
