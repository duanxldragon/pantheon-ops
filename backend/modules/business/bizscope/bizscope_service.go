package bizscope

import (
	"context"
	"errors"
	"strings"
	"time"

	bizcap "pantheon-base/modules/business/capability"
	"pantheon-base/pkg/common"
	"pantheon-base/pkg/database"

	"gorm.io/gorm"
)

const (
	bizScopeCodeExistsKey = "business.bizscope.codeExists"
	bizScopeInUseKey      = "business.bizscope.inUse"
	bizScopeNotFoundKey   = "business.bizscope.notFound"
	cmdbCapabilityMissing = "business.cmdb.capability.notConfigured"
)

type Service struct {
	db               *gorm.DB
	hostReader       bizcap.CMDBHostReader
	ownershipCommand bizcap.CMDBOwnershipCommand
}

type ServiceDependencies struct {
	HostReader       bizcap.CMDBHostReader
	OwnershipCommand bizcap.CMDBOwnershipCommand
}

func NewService(db *gorm.DB, dependencies ...ServiceDependencies) *Service {
	service := &Service{db: db}
	if len(dependencies) > 0 {
		service.hostReader = dependencies[0].HostReader
		service.ownershipCommand = dependencies[0].OwnershipCommand
	}
	return service
}

func (s *Service) Migrate() error {
	if s.db == nil {
		return errors.New("database.not_initialized")
	}
	return s.db.AutoMigrate(&BizScope{})
}

func (s *Service) GetActive(ctx context.Context, id uint64, dataScope *common.DataScopeReq) (bizcap.BizScopeRef, error) {
	if s.db == nil {
		return bizcap.BizScopeRef{}, errors.New("database.not_initialized")
	}
	if id == 0 {
		return bizcap.BizScopeRef{}, errors.New(bizScopeNotFoundKey)
	}
	var row BizScope
	if err := s.db.WithContext(ctx).Model(&BizScope{}).Scopes(database.WithDataScope(dataScope)).Where("id = ? AND status = ?", id, "active").First(&row).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return bizcap.BizScopeRef{}, errors.New(bizScopeNotFoundKey)
		}
		return bizcap.BizScopeRef{}, err
	}
	return toBizScopeRef(row), nil
}

func (s *Service) ResolveActiveByCodes(ctx context.Context, codes []string, dataScope *common.DataScopeReq) (map[string]bizcap.BizScopeRef, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}
	normalized := normalizeBizScopeCodes(codes)
	if len(normalized) == 0 {
		return map[string]bizcap.BizScopeRef{}, nil
	}
	var rows []BizScope
	if err := s.db.WithContext(ctx).Model(&BizScope{}).Scopes(database.WithDataScope(dataScope)).Where("code IN ? AND status = ?", normalized, "active").Find(&rows).Error; err != nil {
		return nil, err
	}
	result := make(map[string]bizcap.BizScopeRef, len(rows))
	for _, row := range rows {
		result[row.Code] = toBizScopeRef(row)
	}
	return result, nil
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
	if query.DeptID > 0 {
		db = db.Where("dept_id = ?", query.DeptID)
	}
	db = db.Scopes(database.WithDataScope(dataScope))

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
	query := s.db.Model(&BizScope{}).Scopes(database.WithDataScope(dataScope)).Where("status = ?", "active")
	if err := query.Order("id desc").Limit(100).Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]BizScopeOptionItem, 0, len(rows))
	for _, row := range rows {
		if requiresScopedHostVisibility(dataScope) {
			page, err := s.cmdbHosts().ListByBusinessScope(context.Background(), bizcap.HostScopeQuery{
				BusinessScopeID: row.ID,
				DataScope:       dataScope,
			})
			if err != nil {
				return nil, err
			}
			if page.Total == 0 {
				continue
			}
		}
		items = append(items, BizScopeOptionItem{
			Label: row.Name,
			Value: row.ID,
			ID:    row.ID,
			Name:  row.Name,
		})
	}
	return items, nil
}

func (s *Service) Get(id uint64, dataScope *common.DataScopeReq) (*BizScopeDetailResp, error) {
	var row BizScope
	if err := s.db.Model(&BizScope{}).Scopes(database.WithDataScope(dataScope)).First(&row, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New(bizScopeNotFoundKey)
		}
		return nil, err
	}
	page, err := s.cmdbHosts().ListByBusinessScope(context.Background(), bizcap.HostScopeQuery{
		BusinessScopeID: id,
		DataScope:       dataScope,
	})
	if err != nil {
		return nil, err
	}
	resp := toDetailRespWithHostCount(row, page.Total)
	return &resp, nil
}

func (s *Service) Create(req *CreateBizScopeRequest, dataScopes ...*common.DataScopeReq) (*BizScopeListResp, error) {
	if s.codeExists(req.Code, 0) {
		return nil, errors.New(bizScopeCodeExistsKey)
	}
	deptID, err := validateBizScopeDept(req.DeptID, firstDataScope(dataScopes))
	if err != nil {
		return nil, err
	}
	row := BizScope{
		Code:        strings.TrimSpace(req.Code),
		Name:        strings.TrimSpace(req.Name),
		Owner:       strings.TrimSpace(req.Owner),
		Environment: strings.TrimSpace(req.Environment),
		Status:      strings.TrimSpace(req.Status),
		DeptID:      deptID,
		Remark:      strings.TrimSpace(req.Remark),
	}
	if err := s.db.Create(&row).Error; err != nil {
		return nil, err
	}
	resp := toListResp(row)
	return &resp, nil
}

func (s *Service) Update(id uint64, req *UpdateBizScopeRequest, dataScopes ...*common.DataScopeReq) (*BizScopeListResp, error) {
	var row BizScope
	var dataScope *common.DataScopeReq
	if len(dataScopes) > 0 {
		dataScope = dataScopes[0]
	}
	if err := s.db.Model(&BizScope{}).Scopes(database.WithDataScope(dataScope)).First(&row, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New(bizScopeNotFoundKey)
		}
		return nil, err
	}

	if req.Code != nil {
		code := strings.TrimSpace(*req.Code)
		if s.codeExists(code, id) {
			return nil, errors.New(bizScopeCodeExistsKey)
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
	if req.DeptID != nil {
		deptID, err := validateBizScopeDept(*req.DeptID, dataScope)
		if err != nil {
			return nil, err
		}
		row.DeptID = deptID
	}
	if err := s.db.Save(&row).Error; err != nil {
		return nil, err
	}
	resp := toListResp(row)
	return &resp, nil
}

func firstDataScope(dataScopes []*common.DataScopeReq) *common.DataScopeReq {
	if len(dataScopes) == 0 {
		return nil
	}
	return dataScopes[0]
}

func validateBizScopeDept(requested uint64, dataScope *common.DataScopeReq) (uint64, error) {
	if dataScope == nil || dataScope.IsAdmin || dataScope.Mode == "" || dataScope.Mode == common.DataScopeModeAll {
		return requested, nil
	}
	allowed := dataScope.DeptIDs
	if len(allowed) == 0 && dataScope.DeptID > 0 {
		allowed = []uint64{dataScope.DeptID}
	}
	if len(allowed) == 0 {
		return 0, errors.New("permission.denied")
	}
	if requested == 0 {
		if dataScope.Mode == common.DataScopeModeDept {
			return dataScope.DeptID, nil
		}
		return 0, errors.New("permission.denied")
	}
	for _, deptID := range allowed {
		if requested == deptID {
			return requested, nil
		}
	}
	return 0, errors.New("permission.denied")
}

func (s *Service) Delete(id uint64, dataScopes ...*common.DataScopeReq) error {
	var dataScope *common.DataScopeReq
	if len(dataScopes) > 0 {
		dataScope = dataScopes[0]
	}
	return s.cmdbOwnership().WithBusinessScopeOwnershipLock(context.Background(), id, func() error {
		var scope BizScope
		if err := s.db.Model(&BizScope{}).Scopes(database.WithDataScope(dataScope)).First(&scope, id).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New(bizScopeNotFoundKey)
			}
			return err
		}
		hasReferences, err := s.cmdbHosts().HasBusinessScopeReferences(context.Background(), id)
		if err != nil {
			return err
		}
		if hasReferences {
			return errors.New(bizScopeInUseKey)
		}
		result := s.db.Delete(&BizScope{}, id)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return errors.New(bizScopeNotFoundKey)
		}
		return nil
	})
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
		DeptID:      row.DeptID,
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
		DeptID:      row.DeptID,
		HostCount:   hostCount,
		CreatedAt:   row.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   row.UpdatedAt.Format(time.RFC3339),
	}
}

func (s *Service) ListBoundHosts(scopeID uint64, dataScope *common.DataScopeReq) (*BizScopeHostListResp, error) {
	if _, err := s.GetActive(context.Background(), scopeID, dataScope); err != nil {
		return nil, err
	}
	page, err := s.cmdbHosts().ListByBusinessScope(context.Background(), bizcap.HostScopeQuery{
		BusinessScopeID: scopeID,
		DataScope:       dataScope,
	})
	if err != nil {
		return nil, err
	}
	items := make([]BizScopeHostItem, len(page.Items))
	for index, row := range page.Items {
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
	return &BizScopeHostListResp{Items: items, Total: page.Total}, nil
}

func (s *Service) ListAvailableHosts(scopeID uint64, dataScope *common.DataScopeReq) (*BizScopeHostListResp, error) {
	if _, err := s.GetActive(context.Background(), scopeID, dataScope); err != nil {
		return nil, err
	}
	page, err := s.cmdbHosts().ListAvailable(context.Background(), bizcap.AvailableHostQuery{DataScope: dataScope})
	if err != nil {
		return nil, err
	}
	items := make([]BizScopeHostItem, len(page.Items))
	for index, row := range page.Items {
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
	return &BizScopeHostListResp{Items: items, Total: page.Total}, nil
}

func (s *Service) BindHosts(scopeID uint64, hostIDs []uint64, dataScope *common.DataScopeReq) error {
	scope, err := s.GetActive(context.Background(), scopeID, dataScope)
	if err != nil {
		return err
	}
	if len(hostIDs) == 0 {
		return errors.New("param.invalid")
	}
	normalizedHostIDs := common.NormalizeUint64IDs(hostIDs)
	if len(normalizedHostIDs) == 0 {
		return errors.New("param.invalid")
	}
	return s.cmdbOwnership().Bind(context.Background(), bizcap.BindOwnershipRequest{
		BusinessScopeID:   scope.ID,
		BusinessScopeCode: scope.Code,
		BusinessScopeName: scope.Name,
		HostIDs:           normalizedHostIDs,
		DataScope:         dataScope,
	})
}

func (s *Service) UnbindHost(scopeID uint64, hostID uint64, dataScope *common.DataScopeReq) error {
	if _, err := s.GetActive(context.Background(), scopeID, dataScope); err != nil {
		return err
	}
	return s.cmdbOwnership().Unbind(context.Background(), bizcap.UnbindOwnershipRequest{
		BusinessScopeID: scopeID,
		HostID:          hostID,
		DataScope:       dataScope,
	})
}

func (s *Service) cmdbHosts() bizcap.CMDBHostReader {
	if s.hostReader != nil {
		return s.hostReader
	}
	return missingCMDBCapability{}
}

func (s *Service) cmdbOwnership() bizcap.CMDBOwnershipCommand {
	if s.ownershipCommand != nil {
		return s.ownershipCommand
	}
	return missingCMDBCapability{}
}

type missingCMDBCapability struct{}

func (missingCMDBCapability) GetByIDs(context.Context, bizcap.HostIDsQuery) (bizcap.HostPage, error) {
	return bizcap.HostPage{}, errors.New(cmdbCapabilityMissing)
}

func (missingCMDBCapability) ListByBusinessScope(context.Context, bizcap.HostScopeQuery) (bizcap.HostPage, error) {
	return bizcap.HostPage{}, errors.New(cmdbCapabilityMissing)
}

func (missingCMDBCapability) ListAvailable(context.Context, bizcap.AvailableHostQuery) (bizcap.HostPage, error) {
	return bizcap.HostPage{}, errors.New(cmdbCapabilityMissing)
}

func (missingCMDBCapability) HasBusinessScopeReferences(context.Context, uint64) (bool, error) {
	return false, errors.New(cmdbCapabilityMissing)
}

func (missingCMDBCapability) Bind(context.Context, bizcap.BindOwnershipRequest) error {
	return errors.New(cmdbCapabilityMissing)
}

func (missingCMDBCapability) Unbind(context.Context, bizcap.UnbindOwnershipRequest) error {
	return errors.New(cmdbCapabilityMissing)
}

func (missingCMDBCapability) WithBusinessScopeOwnershipLock(context.Context, uint64, func() error) error {
	return errors.New(cmdbCapabilityMissing)
}

func toBizScopeRef(row BizScope) bizcap.BizScopeRef {
	return bizcap.BizScopeRef{
		ID:          row.ID,
		Code:        row.Code,
		Name:        row.Name,
		Environment: row.Environment,
		Status:      row.Status,
		DeptID:      row.DeptID,
	}
}

func normalizeBizScopeCodes(codes []string) []string {
	result := make([]string, 0, len(codes))
	seen := make(map[string]struct{}, len(codes))
	for _, code := range codes {
		code = strings.TrimSpace(code)
		if code == "" {
			continue
		}
		if _, ok := seen[code]; ok {
			continue
		}
		seen[code] = struct{}{}
		result = append(result, code)
	}
	return result
}

func requiresScopedHostVisibility(dataScope *common.DataScopeReq) bool {
	if dataScope == nil || dataScope.IsAdmin {
		return false
	}
	mode := strings.TrimSpace(dataScope.Mode)
	return mode != "" && mode != common.DataScopeModeAll
}
