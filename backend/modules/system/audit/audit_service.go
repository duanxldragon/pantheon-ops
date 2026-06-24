package system

import (
	"encoding/json"
	"errors"
	"fmt"
	"pantheon-ops/backend/pkg/common"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"pantheon-ops/backend/internal/middleware"
	"pantheon-ops/backend/pkg/impexp"

	"gorm.io/gorm"
)

type AuditService struct {
	db              *gorm.DB
	lastCleanupAtMu sync.Mutex
	lastCleanupAt   map[string]time.Time
}

func NewAuditService(db *gorm.DB) *AuditService {
	return &AuditService{
		db:            db,
		lastCleanupAt: make(map[string]time.Time),
	}
}

const (
	defaultOperationLogRetentionDays = 180
	auditAutoCleanupMinInterval      = 15 * time.Minute
)

func (s *AuditService) Migrate() error {
	if s.db == nil {
		return common.ErrDatabaseNotInitialized
	}
	if err := s.db.AutoMigrate(&middleware.SystemLogOper{}); err != nil {
		return err
	}
	return s.Bootstrap()
}

func (s *AuditService) Bootstrap() error {
	if s.db == nil {
		return common.ErrDatabaseNotInitialized
	}
	return s.backfillOperationLogDerivedFields()
}

func (s *AuditService) ListOperationLogs(query *OperationLogQuery) (*OperationLogPageResp, error) {
	if s.db == nil {
		return nil, common.ErrDatabaseNotInitialized
	}
	s.ensureAutomaticOperationLogRetention()

	page := 1
	pageSize := 10
	if query != nil {
		if query.Page > 0 {
			page = query.Page
		}
		if query.PageSize > 0 {
			pageSize = query.PageSize
		}
	}

	db := s.applyOperationLogBaseQuery(s.db.Model(&middleware.SystemLogOper{}), query)
	var total int64
	var rows []middleware.SystemLogOper
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}
	if err := db.Order("id desc").Offset((page - 1) * pageSize).Limit(pageSize).Find(&rows).Error; err != nil {
		return nil, err
	}

	items := make([]OperationLogResp, 0, len(rows))
	for _, row := range rows {
		items = append(items, operationLogToResp(row))
	}

	return &OperationLogPageResp{
		Items:    items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}, nil
}

func (s *AuditService) GetOperationLog(logID uint64) (*OperationLogResp, error) {
	if s.db == nil {
		return nil, common.ErrDatabaseNotInitialized
	}
	s.ensureAutomaticOperationLogRetention()

	var row middleware.SystemLogOper
	if err := s.db.First(&row, logID).Error; err != nil {
		return nil, err
	}
	resp := operationLogToResp(row)
	return &resp, nil
}

func (s *AuditService) ExportOperationLogs(query *OperationLogQuery) (*impexp.CSVFile, error) {
	if s.db == nil {
		return nil, common.ErrDatabaseNotInitialized
	}
	s.ensureAutomaticOperationLogRetention()

	rows, err := s.listOperationLogsForExport(query)
	if err != nil {
		return nil, err
	}
	result := make([][]string, 0, len(rows))
	for _, row := range rows {
		sourceDomain := strings.TrimSpace(row.SourceDomain)
		if sourceDomain == "" {
			sourceDomain = detectOperationLogSourceDomain(row.OperURL)
		}
		sourcePage := strings.TrimSpace(row.SourcePage)
		if sourcePage == "" {
			sourcePage = detectOperationLogSourcePage(row.OperURL)
		}
		failureCategory := strings.TrimSpace(row.FailureCategory)
		if failureCategory == "" {
			failureCategory = detectOperationLogFailureCategory(row.Status, row.ErrorMsg, row.JsonResult)
		}
		result = append(result, []string{
			row.RequestID,
			row.Title,
			fmt.Sprintf("%d", row.BusinessType),
			sourceDomain,
			sourcePage,
			row.Method,
			row.OperName,
			row.OperURL,
			row.OperIP,
			fmt.Sprintf("%d", row.Status),
			failureCategory,
			row.ErrorMsg,
			row.OperTime.Format(time.RFC3339),
			fmt.Sprintf("%d", row.CostTime),
		})
	}

	return &impexp.CSVFile{
		Filename: "system-operation-log-export.csv",
		Headers:  []string{"requestId", "title", "businessType", "sourceDomain", "sourcePage", "method", "operName", "operUrl", "operIp", "status", "failureCategory", "errorMsg", "operTime", "costTime"},
		Rows:     result,
	}, nil
}

func (s *AuditService) DeleteOperationLog(logID uint64) error {
	if s.db == nil {
		return common.ErrDatabaseNotInitialized
	}
	return s.db.Delete(&middleware.SystemLogOper{}, logID).Error
}

func (s *AuditService) CleanupOperationLogs(retentionDays int, startedAt string, endedAt string) (int64, error) {
	if s.db == nil {
		return 0, common.ErrDatabaseNotInitialized
	}
	window, err := parseOperationCleanupWindow(startedAt, endedAt)
	if err != nil {
		return 0, err
	}

	db := s.db.Model(&middleware.SystemLogOper{})
	if window != nil {
		db = db.Where("oper_time >= ? AND oper_time <= ?", window.StartedAt, window.EndedAt)
	} else {
		if !s.isAllowedOperationLogRetentionDays(retentionDays) {
			return 0, errors.New("audit.operation_log.cleanup.days_invalid")
		}
		cutoff := time.Now().AddDate(0, 0, -retentionDays)
		db = db.Where("oper_time < ?", cutoff)
	}
	result := db.Delete(&middleware.SystemLogOper{})
	if result.Error != nil {
		return 0, result.Error
	}
	return result.RowsAffected, nil
}

type operationCleanupWindow struct {
	StartedAt time.Time
	EndedAt   time.Time
}

func parseOperationCleanupWindow(startedAt, endedAt string) (*operationCleanupWindow, error) {
	startedAt = strings.TrimSpace(startedAt)
	endedAt = strings.TrimSpace(endedAt)
	if startedAt == "" && endedAt == "" {
		return nil, nil
	}
	if startedAt == "" || endedAt == "" {
		return nil, errors.New("audit.operation_log.cleanup.range_invalid")
	}
	start, err := time.Parse(time.RFC3339, startedAt)
	if err != nil {
		return nil, errors.New("audit.operation_log.cleanup.range_invalid")
	}
	end, err := time.Parse(time.RFC3339, endedAt)
	if err != nil {
		return nil, errors.New("audit.operation_log.cleanup.range_invalid")
	}
	if end.Before(start) {
		return nil, errors.New("audit.operation_log.cleanup.range_invalid")
	}
	return &operationCleanupWindow{StartedAt: start, EndedAt: end}, nil
}

func (s *AuditService) isAllowedOperationLogRetentionDays(retentionDays int) bool {
	for _, allowed := range s.getOperationLogRetentionOptions() {
		if allowed == retentionDays {
			return true
		}
	}
	return false
}

func (s *AuditService) getOperationLogRetentionOptions() []int {
	return s.getRetentionOptionsFromSetting("audit.operation_log_retention_options", []int{1, 7, 30})
}

func (s *AuditService) getRetentionOptionsFromSetting(settingKey string, fallback []int) []int {
	if s.db == nil {
		return fallback
	}

	var row struct {
		SettingValue string `gorm:"column:setting_value"`
	}
	if err := s.db.Table("system_setting").Select("setting_value").Where("setting_key = ?", settingKey).Take(&row).Error; err != nil {
		return fallback
	}

	var values []int
	if err := json.Unmarshal([]byte(strings.TrimSpace(row.SettingValue)), &values); err != nil {
		return fallback
	}

	normalized := make([]int, 0, len(values))
	seen := make(map[int]struct{}, len(values))
	for _, value := range values {
		if value <= 0 {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		normalized = append(normalized, value)
	}
	if len(normalized) == 0 {
		return fallback
	}

	sort.Ints(normalized)
	return normalized
}

func (s *AuditService) ensureAutomaticOperationLogRetention() {
	if s.db == nil {
		return
	}

	now := time.Now()
	s.lastCleanupAtMu.Lock()
	lastRun := s.lastCleanupAt["operation_log_retention"]
	if !lastRun.IsZero() && now.Sub(lastRun) < auditAutoCleanupMinInterval {
		s.lastCleanupAtMu.Unlock()
		return
	}
	s.lastCleanupAt["operation_log_retention"] = now
	s.lastCleanupAtMu.Unlock()

	retentionDays := s.getRetentionDaysFromSetting("audit.operation_log_retention_days", defaultOperationLogRetentionDays)
	if retentionDays <= 0 {
		retentionDays = defaultOperationLogRetentionDays
	}
	cutoff := now.AddDate(0, 0, -retentionDays)
	_ = s.db.Where("oper_time < ?", cutoff).Delete(&middleware.SystemLogOper{}).Error
}

func (s *AuditService) getRetentionDaysFromSetting(settingKey string, fallback int) int {
	if s.db == nil {
		return fallback
	}

	var row struct {
		SettingValue string `gorm:"column:setting_value"`
	}
	if err := s.db.Table("system_setting").Select("setting_value").Where("setting_key = ?", settingKey).Take(&row).Error; err != nil {
		return fallback
	}

	value, err := strconv.Atoi(strings.TrimSpace(row.SettingValue))
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func (s *AuditService) BatchDeleteOperationLogs(ids []uint64) (int64, error) {
	if s.db == nil {
		return 0, common.ErrDatabaseNotInitialized
	}

	normalized := normalizeAuditLogIDs(ids)
	if len(normalized) == 0 {
		return 0, errors.New("audit.operation_log.delete.ids_required")
	}

	result := s.db.Where("id IN ?", normalized).Delete(&middleware.SystemLogOper{})
	if result.Error != nil {
		return 0, result.Error
	}
	return result.RowsAffected, nil
}

func (s *AuditService) listOperationLogsForExport(query *OperationLogQuery) ([]middleware.SystemLogOper, error) {
	var rows []middleware.SystemLogOper
	db := s.applyOperationLogBaseQuery(s.db.Model(&middleware.SystemLogOper{}), query)
	if err := db.Order("id desc").Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func (s *AuditService) applyOperationLogBaseQuery(db *gorm.DB, query *OperationLogQuery) *gorm.DB {
	if query == nil {
		return db
	}
	if strings.TrimSpace(query.Title) != "" {
		db = db.Where("title LIKE ?", "%"+strings.TrimSpace(query.Title)+"%")
	}
	if strings.TrimSpace(query.RequestID) != "" {
		db = db.Where("request_id = ?", strings.TrimSpace(query.RequestID))
	}
	if strings.TrimSpace(query.OperName) != "" {
		db = db.Where("oper_name LIKE ?", "%"+strings.TrimSpace(query.OperName)+"%")
	}
	if query.Status != nil {
		db = db.Where("status = ?", *query.Status)
	}
	if query.BusinessType != nil {
		db = db.Where("business_type = ?", *query.BusinessType)
	}
	if strings.TrimSpace(query.SourceDomain) != "" {
		db = db.Where("source_domain = ?", strings.TrimSpace(query.SourceDomain))
	}
	if strings.TrimSpace(query.SourcePage) != "" {
		db = db.Where("source_page = ?", strings.TrimSpace(query.SourcePage))
	}
	if strings.TrimSpace(query.FailureCategory) != "" {
		db = db.Where("failure_category = ?", strings.TrimSpace(query.FailureCategory))
	}
	return db
}

func operationLogToResp(row middleware.SystemLogOper) OperationLogResp {
	return OperationLogResp{
		ID:              row.ID,
		RequestID:       strings.TrimSpace(row.RequestID),
		Title:           row.Title,
		BusinessType:    row.BusinessType,
		Method:          row.Method,
		OperName:        row.OperName,
		OperURL:         row.OperURL,
		OperIP:          row.OperIP,
		SourceDomain:    strings.TrimSpace(row.SourceDomain),
		SourcePage:      strings.TrimSpace(row.SourcePage),
		OperParam:       row.OperParam,
		JsonResult:      row.JsonResult,
		Status:          row.Status,
		FailureCategory: strings.TrimSpace(row.FailureCategory),
		ErrorMsg:        row.ErrorMsg,
		OperTime:        row.OperTime.Format(time.RFC3339),
		CostTime:        row.CostTime,
	}
}

func (s *AuditService) backfillOperationLogDerivedFields() error {
	if s.db == nil {
		return common.ErrDatabaseNotInitialized
	}

	var rows []middleware.SystemLogOper
	if err := s.db.
		Where("COALESCE(source_domain, '') = '' OR COALESCE(source_page, '') = '' OR (status = ? AND COALESCE(failure_category, '') = '')", 2).
		Find(&rows).Error; err != nil {
		return err
	}

	for _, row := range rows {
		sourceDomain := strings.TrimSpace(row.SourceDomain)
		if sourceDomain == "" {
			sourceDomain = detectOperationLogSourceDomain(row.OperURL)
		}
		sourcePage := strings.TrimSpace(row.SourcePage)
		if sourcePage == "" {
			sourcePage = detectOperationLogSourcePage(row.OperURL)
		}
		failureCategory := strings.TrimSpace(row.FailureCategory)
		if failureCategory == "" {
			failureCategory = detectOperationLogFailureCategory(row.Status, row.ErrorMsg, row.JsonResult)
		}
		if err := s.db.Model(&middleware.SystemLogOper{}).
			Where("id = ?", row.ID).
			Updates(map[string]any{
				"source_domain":    sourceDomain,
				"source_page":      sourcePage,
				"failure_category": failureCategory,
			}).Error; err != nil {
			return err
		}
	}
	return nil
}

func normalizeAuditLogIDs(ids []uint64) []uint64 {
	if len(ids) == 0 {
		return nil
	}

	seen := make(map[uint64]struct{}, len(ids))
	result := make([]uint64, 0, len(ids))
	for _, id := range ids {
		if id == 0 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	return result
}
