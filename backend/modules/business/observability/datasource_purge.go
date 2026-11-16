package observability

import (
	"errors"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/database"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"sort"
)

func datasourceOwners(tx *gorm.DB, tenantID, id uint64, locking bool) ([]QueryOwner, error) {
	owners := []QueryOwner{{TenantID: tenantID, ActorID: 0}}
	for _, model := range []any{&QueryHistory{}, &SavedView{}} {
		var found []QueryOwner
		q := tx.Model(model).Where("tenant_id = ? AND datasource_id = ?", tenantID, id).Select("tenant_id,actor_id")
		if locking {
			q = q.Clauses(clause.Locking{Strength: "UPDATE"})
		}
		if err := q.Find(&found).Error; err != nil {
			return nil, err
		}
		owners = append(owners, found...)
	}
	sort.Slice(owners, func(i, j int) bool { return owners[i].ActorID < owners[j].ActorID })
	unique := owners[:0]
	for _, owner := range owners {
		if len(unique) == 0 || unique[len(unique)-1] != owner {
			unique = append(unique, owner)
		}
	}
	return unique, nil
}
func sameOwners(a, b []QueryOwner) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func (s *DatasourceService) purgeDatasource(id uint64, scope *common.DataScopeReq) error {
	for attempt := 0; attempt < 3; attempt++ {
		owners, err := datasourceOwners(s.db, s.tenantID, id, false)
		if err != nil {
			return err
		}
		err = s.db.Transaction(func(tx *gorm.DB) error {
			if err := lockQueryOwners(tx, owners); err != nil {
				return err
			}
			var source PrometheusDatasource
			if err := tx.Where("tenant_id = ?", s.tenantID).Scopes(database.WithDataScope(scope)).Clauses(clause.Locking{Strength: "UPDATE"}).First(&source, id).Error; err != nil {
				return ErrWorkspaceDenied
			}
			current, err := datasourceOwners(tx, s.tenantID, id, true)
			if err != nil {
				return err
			}
			if !sameOwners(owners, current) {
				return ErrWorkspaceConflict
			}
			var count int64
			if err := tx.Model(&AlertRule{}).Where("datasource_id = ?", id).Count(&count).Error; err != nil {
				return err
			}
			if count > 0 {
				return errors.New("observability.datasource_in_use")
			}
			for _, model := range []any{&QueryHistory{}, &SavedView{}} {
				if err := tx.Unscoped().Where("tenant_id = ? AND datasource_id = ?", s.tenantID, id).Delete(model).Error; err != nil {
					return err
				}
			}
			return tx.Delete(&source).Error
		})
		if !errors.Is(err, ErrWorkspaceConflict) {
			return err
		}
	}
	return ErrWorkspaceConflict
}
