package setting

import (
	"context"
	"errors"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/pkg/contracts"
	"github.com/duanxldragon/pantheon-base/backend/pkg/tenant"
	"github.com/duanxldragon/pantheon-base/backend/pkg/testmysql"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func businessSettingsDefinitions() []contracts.BusinessSettingDefinition {
	return []contracts.BusinessSettingDefinition{
		{BusinessSettingSeed: contracts.BusinessSettingSeed{Key: "cmdb.ssh.default_port", Value: "22", ValueType: "number", GroupKey: "cmdb", Remark: "SSH port"}, Module: "business.cmdb"},
		{BusinessSettingSeed: contracts.BusinessSettingSeed{Key: "cmdb.ssh.collect_timeout", Value: "10", ValueType: "number", GroupKey: "cmdb", Remark: "SSH timeout"}, Module: "business.cmdb"},
	}
}
func businessSettingFixture(t *testing.T, mode string) *SettingService {
	t.Helper()
	db := testmysql.Open(t)
	if err := db.AutoMigrate(&SystemSetting{}); err != nil {
		t.Fatal(err)
	}
	if err := db.Create(&SystemSetting{SettingKey: tenant.FeatureFlagSettingKey, SettingValue: mode, ValueType: "string", GroupKey: "platform", Module: "platform"}).Error; err != nil {
		t.Fatal(err)
	}
	return NewSettingService(db)
}
func businessSettingsGin(t *testing.T, mode, subject string) *gin.Context {
	t.Helper()
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	c.Request = httptest.NewRequest("GET", "/business", nil)
	if mode != "" {
		resolved, err := tenant.Resolve(tenant.ResolveInput{Mode: mode, SubjectTenantID: subject})
		if err != nil {
			t.Fatal(err)
		}
		c.Set(tenant.ContextKey, resolved)
	}
	return c
}
func seedBusinessOverride(t *testing.T, s *SettingService, id uint64, value string) {
	t.Helper()
	definition := businessSettingsDefinitions()[1]
	if err := s.db.Create(&SystemSetting{TenantID: id, SettingKey: definition.Key, SettingValue: value, ValueType: definition.ValueType, GroupKey: definition.GroupKey, Module: definition.Module}).Error; err != nil {
		t.Fatal(err)
	}
}

func TestBusinessRuntimeSettingsDefaultsPreserveAndConcurrent(t *testing.T) {
	s := businessSettingFixture(t, tenant.ModeCompat)
	definitions := businessSettingsDefinitions()
	results := make(chan error, 6)
	var wg sync.WaitGroup
	for i := 0; i < 6; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); results <- s.BootstrapBusinessSettings(definitions) }()
	}
	wg.Wait()
	close(results)
	for err := range results {
		if err != nil {
			t.Fatal(err)
		}
	}
	var count int64
	if err := s.db.Model(&SystemSetting{}).Where("module = ?", "business.cmdb").Count(&count).Error; err != nil || count != 2 {
		t.Fatalf("duplicate defaults: count=%d err=%v", count, err)
	}
	if err := s.db.Model(&SystemSetting{}).Where("setting_key = ?", definitions[0].Key).Update("setting_value", "2222").Error; err != nil {
		t.Fatal(err)
	}
	if err := s.BootstrapBusinessSettings(definitions); err != nil {
		t.Fatal(err)
	}
	reader, err := s.BusinessReaderFor(businessSettingsGin(t, "", ""), definitions)
	if err != nil {
		t.Fatal(err)
	}
	value, err := reader.GetByKey(definitions[0].Key)
	if err != nil || value != "2222" {
		t.Fatalf("operator value overwritten: %q %v", value, err)
	}
}

func TestBusinessRuntimeSettingsBootstrapRollback(t *testing.T) {
	for _, field := range []string{"module", "public", "encrypted", "type", "group", "value"} {
		t.Run(field, func(t *testing.T) {
			s := businessSettingFixture(t, tenant.ModeCompat)
			definition := businessSettingsDefinitions()[0]
			row := SystemSetting{SettingKey: definition.Key, SettingValue: "2222", ValueType: definition.ValueType, GroupKey: definition.GroupKey, Module: definition.Module}
			switch field {
			case "module":
				row.Module = "system.auth"
			case "public":
				row.IsPublic = 1
			case "encrypted":
				row.IsEncrypted = 1
			case "type":
				row.ValueType = "string"
			case "group":
				row.GroupKey = "other"
			case "value":
				row.SettingValue = "invalid-number"
			}
			if err := s.db.Create(&row).Error; err != nil {
				t.Fatal(err)
			}
			if err := s.BootstrapBusinessSettings(businessSettingsDefinitions()); err == nil {
				t.Fatal("incompatible existing metadata accepted")
			}
			var count int64
			if err := s.db.Model(&SystemSetting{}).Where("setting_key = ?", businessSettingsDefinitions()[1].Key).Count(&count).Error; err != nil || count != 0 {
				t.Fatalf("partial defaults survived rollback: %d %v", count, err)
			}
			var current SystemSetting
			if err := s.db.First(&current, row.ID).Error; err != nil {
				t.Fatal(err)
			}
			if current.SettingValue != row.SettingValue || current.Module != row.Module || current.IsPublic != row.IsPublic || current.IsEncrypted != row.IsEncrypted {
				t.Fatal("conflict row modified")
			}
		})
	}
}

func TestBusinessRuntimeSettingsTenantIsolationAndSnapshot(t *testing.T) {
	s := businessSettingFixture(t, tenant.ModeMulti)
	definitions := businessSettingsDefinitions()
	if err := s.BootstrapBusinessSettings(definitions); err != nil {
		t.Fatal(err)
	}
	seedBusinessOverride(t, s, 101, "15")
	seedBusinessOverride(t, s, 202, "20")
	request101 := businessSettingsGin(t, tenant.ModeMulti, "101")
	reader101, err := s.BusinessReaderFor(request101, definitions)
	if err != nil {
		t.Fatal(err)
	}
	tenant.FromGin(request101).TenantID = 202
	definitions[1].Module = "tampered"
	for _, check := range []struct {
		reader contracts.BusinessRuntimeSettingsReader
		want   string
	}{{reader101, "15"}} {
		got, err := check.reader.GetByKey("cmdb.ssh.collect_timeout")
		if err != nil || got != check.want {
			t.Fatalf("snapshot/tenant drift: %q %v", got, err)
		}
	}
	for _, check := range []struct{ subject, want string }{{"202", "20"}, {"303", "10"}} {
		reader, err := s.BusinessReaderFor(businessSettingsGin(t, tenant.ModeMulti, check.subject), businessSettingsDefinitions())
		if err != nil {
			t.Fatal(err)
		}
		got, err := reader.GetByKey("cmdb.ssh.collect_timeout")
		if err != nil || got != check.want {
			t.Fatalf("tenant %s: %q %v", check.subject, got, err)
		}
	}
	if got, err := reader101.GetByKey("login.password"); err == nil || got != "" {
		t.Fatal("foreign unregistered key exposed")
	}
	if _, err := s.BusinessReaderFor(businessSettingsGin(t, "", ""), businessSettingsDefinitions()); err == nil {
		t.Fatal("missing multi context fell back global")
	}
}

func TestBusinessRuntimeSettingsModeDriftAndCancellation(t *testing.T) {
	for _, mode := range []string{tenant.ModeCompat, tenant.ModeMulti} {
		t.Run(mode, func(t *testing.T) {
			s := businessSettingFixture(t, mode)
			if err := s.BootstrapBusinessSettings(businessSettingsDefinitions()); err != nil {
				t.Fatal(err)
			}
			subject := ""
			if mode == tenant.ModeMulti {
				subject = "101"
			}
			c := businessSettingsGin(t, mode, subject)
			ctx, cancel := context.WithCancel(c.Request.Context())
			defer cancel()
			c.Request = c.Request.WithContext(ctx)
			reader, err := s.BusinessReaderFor(c, businessSettingsDefinitions())
			if err != nil {
				t.Fatal(err)
			}
			next := tenant.ModeMulti
			if mode == tenant.ModeMulti {
				next = tenant.ModeCompat
			}
			if err := s.db.Model(&SystemSetting{}).Where("setting_key = ?", tenant.FeatureFlagSettingKey).Update("setting_value", next).Error; err != nil {
				t.Fatal(err)
			}
			if got, err := reader.GetByKey("cmdb.ssh.collect_timeout"); err == nil || got != "" {
				t.Fatal("mode drift reinterpreted request")
			}
			cancel()
			if _, err := reader.GetByKey("cmdb.ssh.collect_timeout"); !errors.Is(err, context.Canceled) {
				t.Fatalf("cancellation ignored: %v", err)
			}
		})
	}
}

func TestBusinessRuntimeSettingsModeChangesDuringRead(t *testing.T) {
	for _, mode := range []string{tenant.ModeCompat, tenant.ModeMulti} {
		t.Run(mode, func(t *testing.T) {
			s := businessSettingFixture(t, mode)
			definitions := businessSettingsDefinitions()
			if err := s.BootstrapBusinessSettings(definitions); err != nil {
				t.Fatal(err)
			}
			subject, next := "", tenant.ModeMulti
			if mode == tenant.ModeMulti {
				subject, next = "101", tenant.ModeCompat
				seedBusinessOverride(t, s, 101, "15")
			}
			reader, err := s.BusinessReaderFor(businessSettingsGin(t, mode, subject), definitions)
			if err != nil {
				t.Fatal(err)
			}
			flipped, transitionErr := flipModeDuringQuery(t, s, definitions[1].Key, next)
			got, err := reader.GetByKey(definitions[1].Key)
			if !*flipped {
				t.Fatal("mode transition was not exercised during the settings query")
			}
			if *transitionErr != nil {
				t.Fatalf("mode transition failed: %v", *transitionErr)
			}
			if !errors.Is(err, contracts.ErrBusinessSettings) || got != "" {
				t.Fatalf("mode changed during query: returned %q with error %v", got, err)
			}
		})
	}
}

// flipModeDuringQuery flips the tenant mode once, inside the first settings
// query for key, so a read can observe a mode change mid-flight.
func flipModeDuringQuery(t *testing.T, s *SettingService, key, next string) (*bool, *error) {
	t.Helper()
	flipped := new(bool)
	transitionErr := new(error)
	const callback = "test:business-settings-mode-drift"
	if err := s.db.Callback().Query().After("gorm:query").Register(callback, func(tx *gorm.DB) {
		rows, ok := tx.Statement.Dest.(*[]SystemSetting)
		if !ok || len(*rows) == 0 || (*rows)[0].SettingKey != key || *flipped {
			return
		}
		*flipped = true
		*transitionErr = s.db.Model(&SystemSetting{}).Where("setting_key = ?", tenant.FeatureFlagSettingKey).Update("setting_value", next).Error
		if *transitionErr != nil {
			tx.AddError(*transitionErr)
		}
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.db.Callback().Query().Remove(callback) })
	return flipped, transitionErr
}

func TestBusinessRuntimeSettingsRejectsMetadataTamper(t *testing.T) {
	for _, field := range []string{"module", "public", "encrypted", "type", "group", "value", "global"} {
		t.Run(field, func(t *testing.T) {
			s := businessSettingFixture(t, tenant.ModeMulti)
			definitions := businessSettingsDefinitions()
			if err := s.BootstrapBusinessSettings(definitions); err != nil {
				t.Fatal(err)
			}
			seedBusinessOverride(t, s, 101, "15")
			reader, err := s.BusinessReaderFor(businessSettingsGin(t, tenant.ModeMulti, "101"), definitions)
			if err != nil {
				t.Fatal(err)
			}
			query := s.db.Model(&SystemSetting{}).Where("setting_key = ? AND tenant_id = ?", "cmdb.ssh.collect_timeout", 101)
			changes := map[string]interface{}{}
			switch field {
			case "module":
				changes["module"] = "business.deploy"
			case "public":
				changes["is_public"] = 1
			case "encrypted":
				changes["is_encrypted"] = 1
			case "type":
				changes["value_type"] = "string"
			case "group":
				changes["group_key"] = "foreign"
			case "value":
				changes["setting_value"] = "wrong"
			case "global":
				query = s.db.Model(&SystemSetting{}).Where("setting_key = ? AND tenant_id = 0", "cmdb.ssh.collect_timeout")
				changes["is_public"] = 1
			}
			if err := query.Updates(changes).Error; err != nil {
				t.Fatal(err)
			}
			if got, err := reader.GetByKey("cmdb.ssh.collect_timeout"); err == nil || got != "" {
				t.Fatal("incompatible/secret value returned")
			}
		})
	}
}

func TestBusinessRuntimeSettingsInvalidContextAndMode(t *testing.T) {
	s := businessSettingFixture(t, tenant.ModeMulti)
	for _, ctx := range []tenant.Context{{Mode: tenant.ModeMulti, TenantID: 0, ResolvedBy: tenant.ResolvedBySubject}, {Mode: tenant.ModeMulti, TenantID: 101, ResolvedBy: tenant.ResolvedByPlatformFallback}, {Mode: tenant.ModeCompat, TenantID: 101, ResolvedBy: tenant.ResolvedByCompatFallback}, {Mode: tenant.ModeMulti, TenantID: 101, ResolvedBy: "request-body"}} {
		c := businessSettingsGin(t, "", "")
		c.Set(tenant.ContextKey, &ctx)
		if _, err := s.BusinessReaderFor(c, businessSettingsDefinitions()); err == nil {
			t.Fatal("invalid provenance accepted")
		}
	}
	if err := s.db.Model(&SystemSetting{}).Where("setting_key = ?", tenant.FeatureFlagSettingKey).Update("setting_value", "unknown").Error; err != nil {
		t.Fatal(err)
	}
	if _, err := s.BusinessReaderFor(businessSettingsGin(t, tenant.ModeMulti, "101"), businessSettingsDefinitions()); err == nil {
		t.Fatal("unknown mode accepted")
	}
}
