package config

import (
	"encoding/json"
	"strconv"
	"strings"
	"time"

	"pantheon-ops/backend/pkg/common"
	"pantheon-ops/backend/pkg/impexp"

	"gorm.io/gorm"
)

type settingAuditPayload struct {
	GroupKey string                   `json:"groupKey"`
	Changes  []SettingAuditChangeResp `json:"changes"`
}

func (s *SettingService) BuildAuditPayload(groupKey string, req *SettingGroupUpdateReq, includeOld bool) (string, error) {
	if s.db == nil || req == nil || len(req.Items) == 0 {
		return "", nil
	}

	keys, requestValueMap := collectAuditRequestValues(req.Items)
	if len(keys) == 0 {
		return "", nil
	}

	rows, err := s.findAuditRows(groupKey, keys)
	if err != nil {
		return "", err
	}
	changes, err := buildAuditChanges(rows, requestValueMap, includeOld)
	if err != nil {
		return "", err
	}

	payload := settingAuditPayload{
		GroupKey: strings.TrimSpace(groupKey),
		Changes:  changes,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func collectAuditRequestValues(items []SettingUpdateItemReq) ([]string, map[string]string) {
	keys := make([]string, 0, len(items))
	requestValueMap := make(map[string]string, len(items))
	for _, item := range items {
		settingKey := strings.TrimSpace(item.SettingKey)
		if settingKey == "" {
			continue
		}
		keys = append(keys, settingKey)
		requestValueMap[settingKey] = strings.TrimSpace(item.SettingValue)
	}
	return keys, requestValueMap
}

func (s *SettingService) findAuditRows(groupKey string, keys []string) ([]SystemSetting, error) {
	var rows []SystemSetting
	err := s.db.Where("group_key = ? AND setting_key IN ?", strings.TrimSpace(groupKey), keys).Find(&rows).Error
	return rows, err
}

func buildAuditChanges(rows []SystemSetting, requestValueMap map[string]string, includeOld bool) ([]SettingAuditChangeResp, error) {
	changes := make([]SettingAuditChangeResp, 0, len(rows))
	for _, row := range rows {
		change, ok, err := buildAuditChange(row, requestValueMap[row.SettingKey], includeOld)
		if err != nil {
			return nil, err
		}
		if ok {
			changes = append(changes, change)
		}
	}
	return changes, nil
}

func buildAuditChange(row SystemSetting, rawNewValue string, includeOld bool) (SettingAuditChangeResp, bool, error) {
	if row.IsEncrypted == 1 {
		return buildEncryptedAuditChange(row, rawNewValue, includeOld), shouldIncludeEncryptedAuditChange(rawNewValue, includeOld), nil
	}
	return buildPlainAuditChange(row, rawNewValue, includeOld)
}

func shouldIncludeEncryptedAuditChange(rawNewValue string, includeOld bool) bool {
	return !includeOld || strings.TrimSpace(rawNewValue) != ""
}

func buildEncryptedAuditChange(row SystemSetting, rawNewValue string, includeOld bool) SettingAuditChangeResp {
	change := SettingAuditChangeResp{
		SettingKey:  row.SettingKey,
		IsEncrypted: row.IsEncrypted,
	}
	if includeOld && strings.TrimSpace(row.SettingValue) != "" {
		change.OldValue = "***"
	}
	if strings.TrimSpace(rawNewValue) != "" {
		change.NewValue = "***"
	}
	return change
}

func buildPlainAuditChange(row SystemSetting, rawNewValue string, includeOld bool) (SettingAuditChangeResp, bool, error) {
	normalizedNewValue, err := normalizeSettingValue(row.SettingKey, rawNewValue)
	if err != nil {
		return SettingAuditChangeResp{}, false, err
	}
	if includeOld && row.SettingValue == normalizedNewValue {
		return SettingAuditChangeResp{}, false, nil
	}
	change := SettingAuditChangeResp{
		SettingKey:  row.SettingKey,
		IsEncrypted: row.IsEncrypted,
		NewValue:    normalizedNewValue,
	}
	if includeOld {
		change.OldValue = row.SettingValue
	}
	return change, true, nil
}

// applyAuditFilters applies common filter conditions for audit queries.
// Uses JSON_EXTRACT for oper_param JSON field queries instead of LIKE,
// which enables MySQL to use generated column indexes and avoids full table scans.
func applyAuditFilters(db *gorm.DB, query *SettingAuditQuery) *gorm.DB {
	if query == nil {
		return db
	}
	if strings.TrimSpace(query.OperName) != "" {
		db = db.Where("oper_name LIKE ?", "%"+common.EscapeLikePattern(strings.TrimSpace(query.OperName))+"%")
	}
	if strings.TrimSpace(query.GroupKey) != "" {
		// JSON_EXTRACT with unquote for exact match on JSON field — index-friendly
		db = db.Where("JSON_UNQUOTE(JSON_EXTRACT(oper_param, '$.groupKey')) = ?", strings.TrimSpace(query.GroupKey))
	}
	if strings.TrimSpace(query.SettingKey) != "" {
		db = db.Where("JSON_UNQUOTE(JSON_EXTRACT(oper_param, '$.settingKey')) = ?", strings.TrimSpace(query.SettingKey))
	}
	return db
}

func (s *SettingService) ListAudit(query *SettingAuditQuery) (*SettingAuditPageResp, error) {
	if s.db == nil {
		return nil, common.ErrDatabaseNotInitialized
	}

	page := 1
	pageSize := 10
	if query != nil {
		if query.Page > 0 {
			page = query.Page
		}
		if query.PageSize > 0 && query.PageSize <= 100 {
			pageSize = query.PageSize
		}
	}

	db := applyAuditFilters(s.db.Model(&systemSettingAuditLog{}).Where("title = ?", settingAuditTitle), query)

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}

	var rows []systemSettingAuditLog
	if err := db.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
		return nil, err
	}

	items := make([]SettingAuditResp, 0, len(rows))
	for _, row := range rows {
		groupKey, changes := parseSettingAuditPayload(row.OperParam)
		items = append(items, SettingAuditResp{
			ID:       row.ID,
			GroupKey: groupKey,
			OperName: row.OperName,
			OperIP:   row.OperIP,
			Status:   row.Status,
			ErrorMsg: row.ErrorMsg,
			OperTime: row.OperTime.Format(time.RFC3339),
			CostTime: row.CostTime,
			Changes:  changes,
		})
	}

	return &SettingAuditPageResp{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (s *SettingService) ExportAudit(query *SettingAuditQuery) (*impexp.CSVFile, error) {
	if s.db == nil {
		return nil, common.ErrDatabaseNotInitialized
	}

	db := applyAuditFilters(s.db.Model(&systemSettingAuditLog{}).Where("title = ?", settingAuditTitle), query)

	var rows []systemSettingAuditLog
	if err := db.Order("id desc").Find(&rows).Error; err != nil {
		return nil, err
	}

	result := make([][]string, 0, len(rows))
	for _, row := range rows {
		groupKey, changes := parseSettingAuditPayload(row.OperParam)
		result = append(result, []string{
			groupKey,
			row.OperName,
			row.OperIP,
			formatSettingAuditChanges(changes),
			strconv.Itoa(row.Status),
			row.ErrorMsg,
			row.OperTime.Format(time.RFC3339),
			strconv.FormatInt(row.CostTime, 10),
		})
	}

	return &impexp.CSVFile{
		Filename: "system-setting-audit-export.csv",
		Headers:  []string{"groupKey", "operName", "operIp", "changes", "status", "errorMsg", "operTime", "costTime"},
		Rows:     result,
	}, nil
}

func parseSettingAuditPayload(raw string) (string, []SettingAuditChangeResp) {
	var payload struct {
		GroupKey string                   `json:"groupKey"`
		Changes  []SettingAuditChangeResp `json:"changes"`
	}
	if err := json.Unmarshal([]byte(raw), &payload); err != nil {
		return "", []SettingAuditChangeResp{}
	}
	return payload.GroupKey, payload.Changes
}

func formatSettingAuditChanges(changes []SettingAuditChangeResp) string {
	if len(changes) == 0 {
		return ""
	}
	parts := make([]string, 0, len(changes))
	for _, change := range changes {
		if change.IsEncrypted == 1 {
			parts = append(parts, change.SettingKey+":***->***")
			continue
		}
		parts = append(parts, change.SettingKey+":"+change.OldValue+"->"+change.NewValue)
	}
	return strings.Join(parts, " | ")
}
