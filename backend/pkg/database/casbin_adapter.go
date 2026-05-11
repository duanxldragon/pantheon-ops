package database

import (
	"fmt"

	"github.com/casbin/casbin/v2/model"
	"github.com/casbin/casbin/v2/persist"
	"gorm.io/gorm"
)

type CasbinRule struct {
	ID    uint64 `gorm:"primaryKey;autoIncrement"`
	PType string `gorm:"column:ptype;size:100;uniqueIndex:idx_casbin_rule"`
	V0    string `gorm:"size:100;uniqueIndex:idx_casbin_rule"`
	V1    string `gorm:"size:100;uniqueIndex:idx_casbin_rule"`
	V2    string `gorm:"size:100;uniqueIndex:idx_casbin_rule"`
	V3    string `gorm:"size:100;uniqueIndex:idx_casbin_rule"`
	V4    string `gorm:"size:100;uniqueIndex:idx_casbin_rule"`
	V5    string `gorm:"size:100;uniqueIndex:idx_casbin_rule"`
}

func (CasbinRule) TableName() string {
	return "casbin_rule"
}

type GormCasbinAdapter struct {
	db *gorm.DB
}

func NewGormCasbinAdapter(db *gorm.DB) (*GormCasbinAdapter, error) {
	if err := db.AutoMigrate(&CasbinRule{}); err != nil {
		return nil, err
	}
	return &GormCasbinAdapter{db: db}, nil
}

func (a *GormCasbinAdapter) LoadPolicy(m model.Model) error {
	var rules []CasbinRule
	if err := a.db.Order("id asc").Find(&rules).Error; err != nil {
		return err
	}

	for _, rule := range rules {
		if err := persist.LoadPolicyArray(rule.toPolicyArray(), m); err != nil {
			return err
		}
	}
	return nil
}

func (a *GormCasbinAdapter) SavePolicy(m model.Model) error {
	return a.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&CasbinRule{}).Error; err != nil {
			return err
		}

		for sec, ptypes := range m {
			if sec != "p" && sec != "g" {
				continue
			}
			for ptype, ast := range ptypes {
				for _, rule := range ast.Policy {
					if err := tx.Create(newCasbinRule(ptype, rule)).Error; err != nil {
						return err
					}
				}
			}
		}
		return nil
	})
}

func (a *GormCasbinAdapter) AddPolicy(_ string, ptype string, rule []string) error {
	return a.db.Create(newCasbinRule(ptype, rule)).Error
}

func (a *GormCasbinAdapter) RemovePolicy(_ string, ptype string, rule []string) error {
	return a.db.Where(ruleFilter(ptype, rule)).Delete(&CasbinRule{}).Error
}

func (a *GormCasbinAdapter) RemoveFilteredPolicy(_ string, ptype string, fieldIndex int, fieldValues ...string) error {
	query := a.db.Where("ptype = ?", ptype)
	for offset, value := range fieldValues {
		if value == "" {
			continue
		}
		query = query.Where(fmt.Sprintf("v%d = ?", fieldIndex+offset), value)
	}
	return query.Delete(&CasbinRule{}).Error
}

func newCasbinRule(ptype string, rule []string) *CasbinRule {
	entry := &CasbinRule{PType: ptype}
	fields := []*string{&entry.V0, &entry.V1, &entry.V2, &entry.V3, &entry.V4, &entry.V5}
	for index, value := range rule {
		if index >= len(fields) {
			break
		}
		*fields[index] = value
	}
	return entry
}

func (r CasbinRule) toPolicyArray() []string {
	values := []string{r.PType, r.V0, r.V1, r.V2, r.V3, r.V4, r.V5}
	last := len(values) - 1
	for last > 0 && values[last] == "" {
		last--
	}
	return values[:last+1]
}

func ruleFilter(ptype string, rule []string) map[string]interface{} {
	filter := map[string]interface{}{
		"ptype": ptype,
	}
	for index, value := range rule {
		if index > 5 {
			break
		}
		filter[fmt.Sprintf("v%d", index)] = value
	}
	return filter
}
