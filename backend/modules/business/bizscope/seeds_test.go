package bizscope

import (
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/pkg/testmysql"
	"gorm.io/gorm"
)

func TestBizScopeMenuSeedsCreateDirectoryThreePagesAndReparentActions(t *testing.T) {
	db := testmysql.Open(t)
	createBizScopeSeedTables(t, db)
	if err := db.Exec("INSERT INTO system_role (id, role_key) VALUES (1, 'admin')").Error; err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 2; i++ {
		if err := seedMenus(db); err != nil {
			t.Fatal(err)
		}
		if err := seedI18n(db); err != nil {
			t.Fatal(err)
		}
	}
	for _, test := range []struct {
		where string
		args  []any
		want  int64
	}{
		{"type = ?", []any{"M"}, 1}, {"type = ?", []any{"C"}, 3}, {"type = ?", []any{"F"}, 12},
		{"path = ? AND type = ? AND component = ?", []any{"/business/bizscope", "M", ""}, 1},
		{"path = ? AND page_perm = ?", []any{"/business/bizscope/scopes", "business:bizscope:list"}, 1},
		{"path = ? AND page_perm = ?", []any{"/business/bizscope/host-bindings", "business:bizscope:host:view"}, 1},
		{"path = ? AND page_perm = ?", []any{"/business/bizscope/environments", "business:bizscope:environment:list"}, 1},
	} {
		var count int64
		if err := db.Table("system_menu").Where(test.where, test.args...).Count(&count).Error; err != nil || count != test.want {
			t.Fatalf("%s: %d want %d (%v)", test.where, count, test.want, err)
		}
	}
	var bindingCount int64
	if err := db.Table("system_role_menu").Where("role_id = ?", 1).Count(&bindingCount).Error; err != nil || bindingCount != 3 {
		t.Fatalf("admin pages: %d %v", bindingCount, err)
	}
	var permissions int64
	if err := db.Table("system_role_permission").Where("role_id = ?", 1).Count(&permissions).Error; err != nil || permissions != 13 {
		t.Fatalf("admin permissions: %d %v", permissions, err)
	}
	var orphanCount int64
	if err := db.Raw("SELECT COUNT(*) FROM system_menu child LEFT JOIN system_menu parent ON child.parent_id = parent.id WHERE child.parent_id > 0 AND parent.id IS NULL").Scan(&orphanCount).Error; err != nil || orphanCount != 0 {
		t.Fatalf("orphan menus: %d %v", orphanCount, err)
	}
}

func createBizScopeSeedTables(t *testing.T, db *gorm.DB) {
	t.Helper()
	for _, sql := range []string{
		"CREATE TABLE system_role (id BIGINT PRIMARY KEY AUTO_INCREMENT, role_key VARCHAR(64))",
		"CREATE TABLE system_menu (id BIGINT PRIMARY KEY AUTO_INCREMENT,parent_id BIGINT,title_key VARCHAR(255),path VARCHAR(255),component VARCHAR(255),page_perm VARCHAR(255),perms VARCHAR(255),type VARCHAR(8),icon VARCHAR(64),route_name VARCHAR(128),module VARCHAR(64),sort INT,is_visible INT,is_cache INT,created_at DATETIME,updated_at DATETIME)",
		"CREATE TABLE system_role_menu (id BIGINT PRIMARY KEY AUTO_INCREMENT,role_id BIGINT,menu_id BIGINT)",
		"CREATE TABLE system_role_permission (id BIGINT PRIMARY KEY AUTO_INCREMENT,role_id BIGINT,permission_key VARCHAR(255))",
		"CREATE TABLE system_i18n (id BIGINT PRIMARY KEY AUTO_INCREMENT,module VARCHAR(64),locale VARCHAR(16),group_name VARCHAR(64),`key` VARCHAR(255),value TEXT,lifecycle_status VARCHAR(16),created_at DATETIME,updated_at DATETIME)",
	} {
		if err := db.Exec(sql).Error; err != nil {
			t.Fatal(err)
		}
	}
}
