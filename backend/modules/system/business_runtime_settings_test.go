package system

import (
	"github.com/duanxldragon/pantheon-base/backend/modules/system/config/setting"
	"github.com/duanxldragon/pantheon-base/backend/pkg/contracts"
	"github.com/duanxldragon/pantheon-base/backend/pkg/testmysql"
	"github.com/gin-gonic/gin"
	"testing"
)

func TestBusinessRuntimeSettingsRequiredStartup(t *testing.T) {
	for _, auto := range []string{"false", "true"} {
		t.Run(auto, func(t *testing.T) {
			t.Setenv("PANTHEON_AUTO_MIGRATE", auto)
			db := testmysql.Open(t)
			if err := db.AutoMigrate(&setting.SystemSetting{}); err != nil {
				t.Fatal(err)
			}
			definitions := []contracts.BusinessSettingDefinition{{BusinessSettingSeed: contracts.BusinessSettingSeed{Key: "cmdb.ssh.port", Value: "22", ValueType: "number", GroupKey: "cmdb", Remark: "port"}, Module: "business.cmdb"}}
			if err := db.Create(&setting.SystemSetting{SettingKey: definitions[0].Key, SettingValue: "2222", ValueType: "number", GroupKey: "cmdb", Module: "foreign"}).Error; err != nil {
				t.Fatal(err)
			}
			r := gin.New()
			if err := prepareBusinessRuntimeSettings(r.Group("/"), db, setting.NewSettingService(db), definitions); err == nil {
				t.Fatal("required startup ignored configuration conflict")
			}
			if len(r.Routes()) != 0 {
				t.Fatal("routes exposed on configuration failure")
			}
		})
	}
}
func TestBusinessRuntimeSettingsStartupSchemaModes(t *testing.T) {
	for _, auto := range []string{"false", "true"} {
		t.Run(auto, func(t *testing.T) {
			t.Setenv("PANTHEON_AUTO_MIGRATE", auto)
			db := testmysql.Open(t)
			definitions := []contracts.BusinessSettingDefinition{{BusinessSettingSeed: contracts.BusinessSettingSeed{Key: "cmdb.ssh.port", Value: "22", ValueType: "number", GroupKey: "cmdb", Remark: "port"}, Module: "business.cmdb"}}
			err := prepareBusinessRuntimeSettings(gin.New().Group("/"), db, setting.NewSettingService(db), definitions)
			if auto == "false" {
				if err == nil || db.Migrator().HasTable(&setting.SystemSetting{}) {
					t.Fatal("versioned mode silently created missing schema")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			var row setting.SystemSetting
			if err := db.Where("setting_key = ?", definitions[0].Key).First(&row).Error; err != nil || row.SettingValue != "22" {
				t.Fatalf("auto mode did not bootstrap: %v", err)
			}
		})
	}
}
