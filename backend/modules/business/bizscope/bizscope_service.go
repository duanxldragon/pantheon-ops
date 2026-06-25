package bizscope

import (
	"errors"
	"strings"
	"time"

	"pantheon-ops/backend/pkg/common"
	"pantheon-ops/backend/pkg/database"

	"gorm.io/gorm"
)

type bizScopeHostSnapshot struct {
	ID                uint64 `gorm:"column:id"`
	Hostname          string `gorm:"column:hostname"`
	IP                string `gorm:"column:ip"`
	OS                string `gorm:"column:os"`
	Status            string `gorm:"column:status"`
	BusinessScopeID   uint64 `gorm:"column:business_scope_id"`
	BusinessScopeName string `gorm:"column:business_scope_name"`
}

type Service struct {
	db *gorm.DB
}

func NewService(db *gorm.DB) *Service {
	return &Service{db: db}
}

func (s *Service) Migrate() error {
	if s.db == nil {
		return errors.New("database.not_initialized")
	}
	return s.db.AutoMigrate(&BizScope{})
}

func (s *Service) List(query *BizScopeListQuery, dataScope *common.DataScopeReq) (*BizScopeListPageResp, error) {
	if query == nil {
		query = &BizScopeListQuery{}
	}
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.PageSize <= 0 || query.PageSize > 100 {
		query.PageSize = 10
	}

	db := s.db.Model(&BizScope{})
	if query.Code != "" {
		db = db.Where("code LIKE ?", "%"+query.Code+"%")
	}
	if query.Name != "" {
		db = db.Where("name LIKE ?", "%"+query.Name+"%")
	}
	if query.Owner != "" {
		db = db.Where("owner LIKE ?", "%"+query.Owner+"%")
	}
	if query.Environment != "" {
		db = db.Where("environment = ?", query.Environment)
	}
	if query.Status != "" {
		db = db.Where("status = ?", query.Status)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}

	var rows []BizScope
	if err := db.Order("id desc").Offset((query.Page - 1) * query.PageSize).Limit(query.PageSize).Find(&rows).Error; err != nil {
		return nil, err
	}

	items := make([]BizScopeListResp, len(rows))
	for index, row := range rows {
		items[index] = toListResp(row)
	}
	return &BizScopeListPageResp{
		Items:    items,
		Total:    total,
		Page:     query.Page,
		PageSize: query.PageSize,
	}, nil
}

func (s *Service) ListOptions(dataScope *common.DataScopeReq) ([]BizScopeOptionItem, error) {
	var rows []BizScope
	query := s.db.Model(&BizScope{}).Where("status = ?", "active")
	if dataScope != nil && !dataScope.IsAdmin {
		query = query.Where("EXISTS (?)", s.scopedHostsQuery(dataScope).
			Select("1").
			Where("biz_cmdb_host.business_scope_id = biz_business_scope.id"))
	}
	if err := query.Order("id desc").Limit(100).Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]BizScopeOptionItem, len(rows))
	for index, row := range rows {
		items[index] = BizScopeOptionItem{
			Label: row.Name,
			Value: row.ID,
			ID:    row.ID,
			Name:  row.Name,
		}
	}
	return items, nil
}

func (s *Service) Get(id uint64, dataScope *common.DataScopeReq) (*BizScopeDetailResp, error) {
	var row BizScope
	if err := s.db.First(&row, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("bizscope.not_found")
		}
		return nil, err
	}
	var hostCount int64
	if err := s.scopedHostsQuery(dataScope).Where("biz_cmdb_host.business_scope_id = ?", id).Count(&hostCount).Error; err != nil {
		return nil, err
	}
	resp := toDetailRespWithHostCount(row, hostCount)
	return &resp, nil
}

func (s *Service) Create(req *CreateBizScopeRequest) (*BizScopeListResp, error) {
	if s.codeExists(req.Code, 0) {
		return nil, errors.New("bizscope.code_exists")
	}
	row := BizScope{
		Code:        strings.TrimSpace(req.Code),
		Name:        strings.TrimSpace(req.Name),
		Owner:       strings.TrimSpace(req.Owner),
		Environment: strings.TrimSpace(req.Environment),
		Status:      strings.TrimSpace(req.Status),
		Remark:      strings.TrimSpace(req.Remark),
	}
	if err := s.db.Create(&row).Error; err != nil {
		return nil, err
	}
	resp := toListResp(row)
	return &resp, nil
}

func (s *Service) Update(id uint64, req *UpdateBizScopeRequest) (*BizScopeListResp, error) {
	var row BizScope
	if err := s.db.First(&row, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("bizscope.not_found")
		}
		return nil, err
	}

	if req.Code != nil {
		code := strings.TrimSpace(*req.Code)
		if s.codeExists(code, id) {
			return nil, errors.New("bizscope.code_exists")
		}
		row.Code = code
	}
	if req.Name != nil {
		row.Name = strings.TrimSpace(*req.Name)
	}
	if req.Owner != nil {
		row.Owner = strings.TrimSpace(*req.Owner)
	}
	if req.Environment != nil {
		row.Environment = strings.TrimSpace(*req.Environment)
	}
	if req.Status != nil {
		row.Status = strings.TrimSpace(*req.Status)
	}
	if req.Remark != nil {
		row.Remark = strings.TrimSpace(*req.Remark)
	}
	if err := s.db.Save(&row).Error; err != nil {
		return nil, err
	}
	resp := toListResp(row)
	return &resp, nil
}

func (s *Service) Delete(id uint64) error {
	var hostCount int64
	if err := s.db.Table("biz_cmdb_host").Where("business_scope_id = ? AND deleted_at IS NULL", id).Count(&hostCount).Error; err != nil {
		return err
	}
	if hostCount > 0 {
		return errors.New("bizscope.in_use")
	}
	result := s.db.Delete(&BizScope{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("bizscope.not_found")
	}
	return nil
}

func (s *Service) codeExists(code string, excludeID uint64) bool {
	var count int64
	query := s.db.Model(&BizScope{}).Where("code = ?", strings.TrimSpace(code))
	if excludeID > 0 {
		query = query.Where("id <> ?", excludeID)
	}
	_ = query.Count(&count).Error
	return count > 0
}

func toListResp(row BizScope) BizScopeListResp {
	return BizScopeListResp{
		ID:          row.ID,
		Code:        row.Code,
		Name:        row.Name,
		Owner:       row.Owner,
		Environment: row.Environment,
		Status:      row.Status,
		Remark:      row.Remark,
		CreatedAt:   row.CreatedAt.Format(time.RFC3339),
	}
}

func toDetailRespWithHostCount(row BizScope, hostCount int64) BizScopeDetailResp {
	return BizScopeDetailResp{
		ID:          row.ID,
		Code:        row.Code,
		Name:        row.Name,
		Owner:       row.Owner,
		Environment: row.Environment,
		Status:      row.Status,
		Remark:      row.Remark,
		HostCount:   hostCount,
		CreatedAt:   row.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   row.UpdatedAt.Format(time.RFC3339),
	}
}

func (s *Service) ListBoundHosts(scopeID uint64, dataScope *common.DataScopeReq) (*BizScopeHostListResp, error) {
	var scope BizScope
	if err := s.db.First(&scope, scopeID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("bizscope.not_found")
		}
		return nil, err
	}
	var rows []bizScopeHostSnapshot
	if err := s.scopedHostsQuery(dataScope).
		Where("biz_cmdb_host.business_scope_id = ?", scopeID).
		Order("id DESC").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]BizScopeHostItem, len(rows))
	for index, row := range rows {
		items[index] = BizScopeHostItem{
			ID:                row.ID,
			Hostname:          row.Hostname,
			IP:                row.IP,
			OS:                row.OS,
			Status:            row.Status,
			BusinessScopeID:   row.BusinessScopeID,
			BusinessScopeName: row.BusinessScopeName,
		}
	}
	return &BizScopeHostListResp{Items: items, Total: int64(len(items))}, nil
}

func (s *Service) ListAvailableHosts(scopeID uint64, dataScope *common.DataScopeReq) (*BizScopeHostListResp, error) {
	var scope BizScope
	if err := s.db.First(&scope, scopeID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("bizscope.not_found")
		}
		return nil, err
	}
	var rows []bizScopeHostSnapshot
	if err := s.scopedHostsQuery(dataScope).
		Where("(biz_cmdb_host.business_scope_id = 0 OR biz_cmdb_host.business_scope_id IS NULL)").
		Order("id DESC").
		Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]BizScopeHostItem, len(rows))
	for index, row := range rows {
		items[index] = BizScopeHostItem{
			ID:                row.ID,
			Hostname:          row.Hostname,
			IP:                row.IP,
			OS:                row.OS,
			Status:            row.Status,
			BusinessScopeID:   row.BusinessScopeID,
			BusinessScopeName: row.BusinessScopeName,
		}
	}
	return &BizScopeHostListResp{Items: items, Total: int64(len(items))}, nil
}

func (s *Service) BindHosts(scopeID uint64, hostIDs []uint64, dataScope *common.DataScopeReq) error {
	var scope BizScope
	if err := s.db.First(&scope, scopeID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("bizscope.not_found")
		}
		return err
	}
	if len(hostIDs) == 0 {
		return errors.New("param.invalid")
	}
	normalizedHostIDs := common.NormalizeUint64IDs(hostIDs)
	if len(normalizedHostIDs) == 0 {
		return errors.New("param.invalid")
	}
	result := s.scopedHostsQuery(dataScope).
		Where("biz_cmdb_host.id IN ?", normalizedHostIDs).
		Updates(map[string]any{
			"business_scope_id":   scope.ID,
			"business_scope_code": scope.Code,
			"business_scope_name": scope.Name,
			"status":              "assigned",
			"updated_at":          time.Now(),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != int64(len(normalizedHostIDs)) {
		return errors.New("bizscope.not_found")
	}
	return nil
}

func (s *Service) UnbindHost(scopeID uint64, hostID uint64, dataScope *common.DataScopeReq) error {
	var scope BizScope
	if err := s.db.First(&scope, scopeID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("bizscope.not_found")
		}
		return err
	}
	updates := map[string]any{
		"business_scope_id":   uint64(0),
		"business_scope_code": "",
		"business_scope_name": "",
		"updated_at":          time.Now(),
	}
	var row bizScopeHostSnapshot
	if err := s.scopedHostsQuery(dataScope).Where("biz_cmdb_host.id = ?", hostID).Take(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("bizscope.not_found")
		}
		return err
	}
	if row.Status == "assigned" {
		updates["status"] = "pending"
	}
	result := s.scopedHostsQuery(dataScope).
		Where("biz_cmdb_host.id = ? AND biz_cmdb_host.business_scope_id = ?", hostID, scopeID).
		Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("bizscope.not_found")
	}
	return nil
}

func (s *Service) scopedHostsQuery(dataScope *common.DataScopeReq) *gorm.DB {
	return s.db.Table("biz_cmdb_host").Scopes(database.WithDataScope(dataScope)).Where("biz_cmdb_host.deleted_at IS NULL")
}
