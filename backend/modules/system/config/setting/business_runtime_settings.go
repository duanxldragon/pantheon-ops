package setting

import (
	"net/http"
	"sort"
	"strings"

	"github.com/duanxldragon/pantheon-base/backend/pkg/contracts"
	"github.com/duanxldragon/pantheon-base/backend/pkg/tenant"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// BootstrapBusinessSettings prepares only declared business defaults. It is a
// separate required stage, because the legacy system runner logs seed errors.
func (s *SettingService) BootstrapBusinessSettings(definitions []contracts.BusinessSettingDefinition) error {
	if s == nil || s.db == nil {
		return contracts.ErrBusinessSettings
	}
	definitions = append([]contracts.BusinessSettingDefinition(nil), definitions...)
	sort.Slice(definitions, func(i, j int) bool { return definitions[i].Key < definitions[j].Key })
	for _, definition := range definitions {
		root := strings.SplitN(definition.Key, ".", 2)[0]
		for _, core := range settingSeeds() {
			if strings.SplitN(core.SettingKey, ".", 2)[0] == root {
				return contracts.ErrBusinessSettings
			}
		}
	}
	return s.db.Transaction(func(tx *gorm.DB) error {
		for _, definition := range definitions {
			value, err := validateAndNormalizeSettingValue(definition.Key, definition.ValueType, definition.Value)
			if err != nil {
				return contracts.ErrBusinessSettings
			}
			row := SystemSetting{SettingKey: definition.Key, SettingValue: value, ValueType: definition.ValueType, GroupKey: definition.GroupKey, Module: definition.Module, Remark: definition.Remark}
			if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "tenant_id"}, {Name: "setting_key"}}, DoNothing: true}).Create(&row).Error; err != nil {
				return err
			}
			var stored SystemSetting
			if err := tx.Where("tenant_id = 0 AND setting_key = ?", definition.Key).First(&stored).Error; err != nil {
				return err
			}
			if !validBusinessSettingRow(stored, definition) {
				return contracts.ErrBusinessSettings
			}
		}
		return nil
	})
}

// MigrateBusinessSettings is used only in the opt-in auto-migrate startup mode.
func (s *SettingService) MigrateBusinessSettings(definitions []contracts.BusinessSettingDefinition) error {
	if s == nil || s.db == nil {
		return contracts.ErrBusinessSettings
	}
	if err := s.db.AutoMigrate(&SystemSetting{}); err != nil {
		return err
	}
	return s.BootstrapBusinessSettings(definitions)
}

func validBusinessSettingRow(row SystemSetting, definition contracts.BusinessSettingDefinition) bool {
	if row.SettingKey != definition.Key || row.Module != definition.Module || row.ValueType != definition.ValueType || row.GroupKey != definition.GroupKey || row.IsPublic != 0 || row.IsEncrypted != 0 {
		return false
	}
	_, err := validateAndNormalizeSettingValue(definition.Key, definition.ValueType, row.SettingValue)
	return err == nil
}

type businessRuntimeSettingReader struct {
	db           *gorm.DB
	req          *http.Request
	snapshot     *tenant.Context
	expectedMode string
	definitions  map[string]contracts.BusinessSettingDefinition
}

// BusinessReaderFor snapshots middleware tenancy. It never uses request body or
// raw headers to invent a tenant; compiled callers must use the trusted chain.
func (s *SettingService) BusinessReaderFor(c *gin.Context, definitions []contracts.BusinessSettingDefinition) (contracts.BusinessRuntimeSettingsReader, error) {
	if s == nil || s.db == nil || c == nil || c.Request == nil || len(definitions) == 0 {
		return nil, contracts.ErrBusinessSettings
	}
	reader := &businessRuntimeSettingReader{db: s.db, req: c.Request, expectedMode: tenant.ModeCompat, definitions: make(map[string]contracts.BusinessSettingDefinition, len(definitions))}
	if bound := tenant.FromGin(c); bound != nil {
		snapshot := *bound
		if !validBusinessSettingContext(snapshot) {
			return nil, contracts.ErrBusinessSettings
		}
		reader.snapshot = &snapshot
		reader.expectedMode = snapshot.Mode
	}
	for _, definition := range definitions {
		reader.definitions[definition.Key] = definition
	}
	if err := reader.checkMode(); err != nil {
		return nil, err
	}
	return reader, nil
}

func validBusinessSettingContext(ctx tenant.Context) bool {
	switch ctx.Mode {
	case tenant.ModeCompat:
		return ctx.TenantID == 0 && ctx.ResolvedBy == tenant.ResolvedByCompatFallback
	case tenant.ModeMulti:
		switch ctx.ResolvedBy {
		case tenant.ResolvedByHeader:
			return true
		case tenant.ResolvedBySubject:
			return ctx.TenantID > 0
		case tenant.ResolvedByPlatformFallback:
			return ctx.TenantID == 0
		}
	}
	return false
}

func (r *businessRuntimeSettingReader) checkMode() error {
	if err := r.req.Context().Err(); err != nil {
		return err
	}
	modeReader := tenant.NewStrictSettingModeReader(NewSettingService(r.db.WithContext(r.req.Context())))
	current, err := modeReader.ReadMode(r.req.Context())
	if err != nil || current != r.expectedMode {
		return contracts.ErrBusinessSettings
	}
	if current == tenant.ModeMulti && r.snapshot == nil {
		return contracts.ErrBusinessSettings
	}
	return nil
}

func (r *businessRuntimeSettingReader) GetByKey(key string) (string, error) {
	definition, ok := r.definitions[key]
	if !ok {
		return "", contracts.ErrBusinessSettings
	}
	if err := r.checkMode(); err != nil {
		return "", err
	}
	view := &SettingService{db: r.db.WithContext(r.req.Context()), Scoper: tenant.Scoper{TenantCtx: r.snapshot}}
	var rows []SystemSetting
	if err := view.tenantScope()(view.db).Where("setting_key = ?", key).Order("tenant_id asc").Find(&rows).Error; err != nil {
		return "", contracts.ErrBusinessSettings
	}
	if len(rows) == 0 {
		return "", contracts.ErrBusinessSettings
	}
	for _, row := range rows {
		if !validBusinessSettingRow(row, definition) {
			return "", contracts.ErrBusinessSettings
		}
	}
	if err := r.checkMode(); err != nil {
		return "", err
	}
	return rows[len(rows)-1].SettingValue, nil
}
