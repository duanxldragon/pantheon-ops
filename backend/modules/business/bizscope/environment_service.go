package bizscope

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/database"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	environmentNotFoundKey      = "business.bizscope.environment.notFound"
	environmentCodeExistsKey    = "business.bizscope.environment.codeExists"
	environmentCodeRequiredKey  = "business.bizscope.environment.codeRequired"
	environmentNameRequiredKey  = "business.bizscope.environment.nameRequired"
	environmentScopeRequiredKey = "business.bizscope.environment.scopeRequired"
	environmentTypeInvalidKey   = "business.bizscope.environment.envTypeInvalid"
	environmentStatusInvalidKey = "business.bizscope.environment.statusInvalid"
)

// environmentQuery applies data scope to the owning scope, rather than the child's ID.
func environmentQuery(db *gorm.DB, scope *common.DataScopeReq) *gorm.DB {
	parents := db.Model(&BizScope{}).Scopes(database.WithDataScope(scope)).Select("id")
	return db.Model(&BusinessEnvironment{}).Where("business_scope_id IN (?)", parents)
}

func (s *Service) ListEnvironments(ctx context.Context, query EnvironmentListQuery, scope *common.DataScopeReq) (*EnvironmentListPageResp, error) {
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.PageSize <= 0 || query.PageSize > 100 {
		query.PageSize = 10
	}
	db := environmentQuery(s.db.WithContext(ctx), scope)
	if query.BusinessScopeID > 0 {
		db = db.Where("business_scope_id = ?", query.BusinessScopeID)
	}
	if query.EnvType != "" {
		db = db.Where("env_type = ?", query.EnvType)
	}
	if query.Status != "" {
		db = db.Where("status = ?", query.Status)
	}
	if keyword := strings.TrimSpace(query.Keyword); keyword != "" {
		db = db.Where("(code LIKE ? OR name LIKE ?)", "%"+keyword+"%", "%"+keyword+"%")
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}
	var rows []BusinessEnvironment
	if err := db.Order("id DESC").Offset((query.Page - 1) * query.PageSize).Limit(query.PageSize).Find(&rows).Error; err != nil {
		return nil, err
	}
	items := make([]EnvironmentResp, 0, len(rows))
	ids := make([]uint64, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.BusinessScopeID)
	}
	var parents []BizScope
	if len(ids) > 0 {
		if err := s.db.WithContext(ctx).Model(&BizScope{}).Scopes(database.WithDataScope(scope)).Where("id IN ?", ids).Find(&parents).Error; err != nil {
			return nil, err
		}
	}
	names := make(map[uint64]string, len(parents))
	for _, parent := range parents {
		names[parent.ID] = parent.Name
	}
	for _, row := range rows {
		items = append(items, toEnvironmentResp(row, names[row.BusinessScopeID]))
	}
	return &EnvironmentListPageResp{Items: items, Total: total, Page: query.Page, PageSize: query.PageSize}, nil
}

func (s *Service) GetEnvironment(ctx context.Context, id uint64, scope *common.DataScopeReq) (*EnvironmentResp, error) {
	var row BusinessEnvironment
	if err := environmentQuery(s.db.WithContext(ctx), scope).First(&row, id).Error; err != nil {
		return nil, environmentReadError(err)
	}
	var parent BizScope
	if err := s.db.WithContext(ctx).Model(&BizScope{}).Scopes(database.WithDataScope(scope)).First(&parent, row.BusinessScopeID).Error; err != nil {
		return nil, environmentReadError(err)
	}
	resp := toEnvironmentResp(row, parent.Name)
	return &resp, nil
}

func validateEnvironment(row *BusinessEnvironment) error {
	row.Code = strings.TrimSpace(row.Code)
	row.Name = strings.TrimSpace(row.Name)
	row.EnvType = strings.TrimSpace(row.EnvType)
	row.Status = strings.TrimSpace(row.Status)
	row.Remark = strings.TrimSpace(row.Remark)
	switch {
	case row.BusinessScopeID == 0:
		return errors.New(environmentScopeRequiredKey)
	case row.Code == "":
		return errors.New(environmentCodeRequiredKey)
	case row.Name == "":
		return errors.New(environmentNameRequiredKey)
	case row.EnvType != "dev" && row.EnvType != "test" && row.EnvType != "prod":
		return errors.New(environmentTypeInvalidKey)
	case row.Status != "active" && row.Status != "inactive":
		return errors.New(environmentStatusInvalidKey)
	case utf8.RuneCountInString(row.Code) > 128 || utf8.RuneCountInString(row.Name) > 255:
		return errors.New(msgParamInvalid)
	default:
		return nil
	}
}

func lockEnvironmentParent(tx *gorm.DB, id uint64, scope *common.DataScopeReq) (*BizScope, error) {
	var parent BizScope
	if err := tx.Model(&BizScope{}).Scopes(database.WithDataScope(scope)).Clauses(clause.Locking{Strength: "UPDATE"}).First(&parent, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New(bizScopeNotFoundKey)
		}
		return nil, err
	}
	return &parent, nil
}

func saveEnvironment(tx *gorm.DB, row *BusinessEnvironment) error {
	// Include deleted rows: the chosen unique index keeps retired codes reserved.
	var count int64
	if err := tx.Unscoped().Model(&BusinessEnvironment{}).Where("business_scope_id = ? AND code = ? AND id <> ?", row.BusinessScopeID, row.Code, row.ID).Count(&count).Error; err != nil {
		return err
	}
	if count > 0 {
		return errors.New(environmentCodeExistsKey)
	}
	if err := tx.Save(row).Error; err != nil {
		if database.IsDuplicateKey(err) {
			return errors.New(environmentCodeExistsKey)
		}
		return err
	}
	return nil
}

func (s *Service) CreateEnvironment(ctx context.Context, req CreateEnvironmentRequest, scope *common.DataScopeReq) (*EnvironmentResp, error) {
	row := BusinessEnvironment{BusinessScopeID: req.BusinessScopeID, Code: req.Code, Name: req.Name, EnvType: req.EnvType, Status: req.Status, Remark: req.Remark}
	if err := validateEnvironment(&row); err != nil {
		return nil, err
	}
	var resp EnvironmentResp
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		parent, err := lockEnvironmentParent(tx, row.BusinessScopeID, scope)
		if err != nil {
			return err
		}
		if err := saveEnvironment(tx, &row); err != nil {
			return err
		}
		resp = toEnvironmentResp(row, parent.Name)
		return nil
	})
	return &resp, err
}

func applyEnvironmentUpdate(row *BusinessEnvironment, req UpdateEnvironmentRequest) {
	if req.BusinessScopeID != nil {
		row.BusinessScopeID = *req.BusinessScopeID
	}
	if req.Code != nil {
		row.Code = *req.Code
	}
	if req.Name != nil {
		row.Name = *req.Name
	}
	if req.EnvType != nil {
		row.EnvType = *req.EnvType
	}
	if req.Status != nil {
		row.Status = *req.Status
	}
	if req.Remark != nil {
		row.Remark = *req.Remark
	}
}

func (s *Service) UpdateEnvironment(ctx context.Context, id uint64, req UpdateEnvironmentRequest, scope *common.DataScopeReq) (*EnvironmentResp, error) {
	var resp EnvironmentResp
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row BusinessEnvironment
		if err := environmentQuery(tx, scope).Clauses(clause.Locking{Strength: "UPDATE"}).First(&row, id).Error; err != nil {
			return environmentReadError(err)
		}
		applyEnvironmentUpdate(&row, req)
		if err := validateEnvironment(&row); err != nil {
			return err
		}
		parent, err := lockEnvironmentParent(tx, row.BusinessScopeID, scope)
		if err != nil {
			return err
		}
		if err := saveEnvironment(tx, &row); err != nil {
			return err
		}
		resp = toEnvironmentResp(row, parent.Name)
		return nil
	})
	return &resp, err
}

func (s *Service) DeleteEnvironment(ctx context.Context, id uint64, scope *common.DataScopeReq) error {
	result := environmentQuery(s.db.WithContext(ctx), scope).Where("id = ?", id).Delete(&BusinessEnvironment{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New(environmentNotFoundKey)
	}
	return nil
}

func environmentReadError(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return errors.New(environmentNotFoundKey)
	}
	return err
}

func toEnvironmentResp(row BusinessEnvironment, scopeName string) EnvironmentResp {
	return EnvironmentResp{ID: row.ID, BusinessScopeID: row.BusinessScopeID, BusinessScopeName: scopeName, Code: row.Code, Name: row.Name, EnvType: row.EnvType, Status: row.Status, Remark: row.Remark, CreatedAt: row.CreatedAt.Format(time.RFC3339), UpdatedAt: row.UpdatedAt.Format(time.RFC3339)}
}
