package service

import (
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/pkg/testmysql"
)

func TestServiceMenusSeedThreePagesIdempotently(t *testing.T) {
	db := testmysql.Open(t)
	for _, sql := range []string{
		"CREATE TABLE system_role (id BIGINT PRIMARY KEY AUTO_INCREMENT,role_key VARCHAR(64))",
		"CREATE TABLE system_menu (id BIGINT PRIMARY KEY AUTO_INCREMENT,parent_id BIGINT,title_key VARCHAR(255),path VARCHAR(255),component VARCHAR(255),page_perm VARCHAR(255),perms VARCHAR(255),type VARCHAR(8),icon VARCHAR(64),route_name VARCHAR(128),module VARCHAR(64),sort INT,is_visible INT,is_cache INT,created_at DATETIME,updated_at DATETIME)",
		"CREATE TABLE system_role_menu (id BIGINT PRIMARY KEY AUTO_INCREMENT,role_id BIGINT,menu_id BIGINT)",
		"CREATE TABLE system_role_permission (id BIGINT PRIMARY KEY AUTO_INCREMENT,role_id BIGINT,permission_key VARCHAR(255))",
		"CREATE TABLE system_i18n (id BIGINT PRIMARY KEY AUTO_INCREMENT,module VARCHAR(64),locale VARCHAR(16),group_name VARCHAR(64),`key` VARCHAR(255),value TEXT,lifecycle_status VARCHAR(16),created_at DATETIME,updated_at DATETIME)",
		"INSERT INTO system_role (id, role_key) VALUES (1, 'admin')",
	} {
		if err := db.Exec(sql).Error; err != nil {
			t.Fatal(err)
		}
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
		table, where string
		want         int64
	}{
		{"system_menu", "type = 'M' AND path = '/business/service-catalog'", 1},
		{"system_menu", "type = 'C' AND page_perm = 'business:service:list'", 3},
		{"system_menu", "type = 'F'", 4},
		{"system_menu", "path = '/business/service'", 0},
		{"system_role_menu", "role_id = 1", 3},
		{"system_role_permission", "role_id = 1", 5},
	} {
		var count int64
		if err := db.Table(test.table).Where(test.where).Count(&count).Error; err != nil || count != test.want {
			t.Fatalf("%s %s: %d want %d (%v)", test.table, test.where, count, test.want, err)
		}
	}
	var orphanCount int64
	if err := db.Raw("SELECT COUNT(*) FROM system_menu child LEFT JOIN system_menu parent ON child.parent_id = parent.id WHERE child.parent_id > 0 AND parent.id IS NULL").Scan(&orphanCount).Error; err != nil || orphanCount != 0 {
		t.Fatalf("orphan menus: %d %v", orphanCount, err)
	}
}
