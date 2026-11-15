package observability

import (
	"database/sql"
	"errors"
	"sort"
	"strings"

	"github.com/duanxldragon/pantheon-base/backend/pkg/database"
	"gorm.io/gorm"
)

const QueryWorkspaceCapability = "observability-business-tenant-v2"

var ErrQuerySchema = errors.New("observability.schema_unavailable")

type schemaColumn struct {
	Name     string         `gorm:"column:column_name"`
	Type     string         `gorm:"column:column_type"`
	Nullable string         `gorm:"column:is_nullable"`
	Default  sql.NullString `gorm:"column:column_default"`
	Extra    string         `gorm:"column:extra"`
}
type schemaIndex struct {
	Name      string        `gorm:"column:index_name"`
	Column    string        `gorm:"column:column_name"`
	Sequence  int           `gorm:"column:seq_in_index"`
	NonUnique int           `gorm:"column:non_unique"`
	Prefix    sql.NullInt64 `gorm:"column:sub_part"`
}
type expectedColumn struct {
	Type     string
	Nullable bool
	Default  string
	Auto     bool
}
type expectedIndex struct {
	Columns []string
	Unique  bool
}
type tableShape struct {
	Columns map[string]expectedColumn
	Indexes map[string]expectedIndex
}

func queryWorkspaceShapes() map[string]tableShape {
	u := expectedColumn{Type: "bigint unsigned"}
	id := u
	id.Auto = true
	dt := expectedColumn{Type: "datetime(3)"}
	v := func(n string) expectedColumn { return expectedColumn{Type: "varchar(" + n + ")"} }
	nullable := func(c expectedColumn) expectedColumn { c.Nullable = true; return c }
	def := func(c expectedColumn, value string) expectedColumn { c.Default = value; return c }
	idx := func(unique bool, columns ...string) expectedIndex {
		return expectedIndex{Columns: columns, Unique: unique}
	}
	return map[string]tableShape{
		"obs_prometheus_datasource": {
			Columns: map[string]expectedColumn{
				"id": id, "tenant_id": def(u, "0"), "revision": def(u, "1"), "dept_id": nullable(u),
				"name": v("128"), "type": def(v("32"), "prometheus"), "url": v("512"), "access_mode": def(v("32"), "proxy"),
				"username": nullable(v("128")), "password": nullable(v("512")), "bearer_token": nullable(v("1024")),
				"timeout": def(nullable(expectedColumn{Type: "bigint"}), "30"), "is_default": def(nullable(expectedColumn{Type: "tinyint(1)"}), "0"),
				"status": def(v("32"), "unknown"), "last_checked": nullable(dt), "created_at": nullable(dt), "updated_at": nullable(dt), "created_by": nullable(v("64")), "deleted_at": nullable(dt),
			},
			Indexes: map[string]expectedIndex{"PRIMARY": idx(true, "id"), "idx_datasource_tenant_name": idx(true, "tenant_id", "name"), "idx_datasource_tenant_id": idx(false, "tenant_id", "id"), "idx_datasource_tenant_default": idx(false, "tenant_id", "is_default", "deleted_at"), "idx_obs_prometheus_datasource_dept_id": idx(false, "dept_id"), "idx_obs_prometheus_datasource_deleted_at": idx(false, "deleted_at")},
		},
		"obs_query_history": {
			Columns: map[string]expectedColumn{"id": id, "tenant_id": u, "actor_id": u, "datasource_id": u, "source_kind": v("32"), "time_intent": {Type: "json"}, "row_limit": {Type: "bigint"}, "result_mode": v("16"), "outcome": v("16"), "duration_ms": {Type: "int unsigned"}, "recipe": nullable(expectedColumn{Type: "json"}), "created_at": dt, "expires_at": dt},
			Indexes: map[string]expectedIndex{"PRIMARY": idx(true, "id"), "idx_history_owner": idx(false, "tenant_id", "actor_id", "created_at", "id"), "idx_history_source": idx(false, "tenant_id", "datasource_id"), "idx_history_expiry": idx(false, "expires_at", "id")},
		},
		"obs_saved_view": {
			Columns: map[string]expectedColumn{"id": id, "tenant_id": u, "actor_id": u, "datasource_id": u, "recipe": {Type: "json"}, "revision": def(u, "1"), "created_at": dt, "updated_at": dt},
			Indexes: map[string]expectedIndex{"PRIMARY": idx(true, "id"), "idx_view_owner": idx(false, "tenant_id", "actor_id", "updated_at", "id"), "idx_view_source": idx(false, "tenant_id", "datasource_id")},
		},
		"obs_query_owner_budget": {
			Columns: map[string]expectedColumn{"tenant_id": u, "actor_id": u, "created_at": dt, "updated_at": dt},
			Indexes: map[string]expectedIndex{"PRIMARY": idx(true, "tenant_id", "actor_id")},
		},
		"obs_schema_capability": {
			Columns: map[string]expectedColumn{"capability": v("64"), "version": {Type: "smallint unsigned"}, "created_at": dt},
			Indexes: map[string]expectedIndex{"PRIMARY": idx(true, "capability")},
		},
	}
}
func normalizedColumnType(value string) string {
	for _, width := range []string{"(20)", "(11)", "(10)", "(5)"} {
		if strings.HasPrefix(value, "bigint"+width) || strings.HasPrefix(value, "int"+width) || strings.HasPrefix(value, "smallint"+width) {
			return strings.Replace(value, width, "", 1)
		}
	}
	return value
}
func inspectQueryTable(db *gorm.DB, table string) ([]schemaColumn, []schemaIndex, error) {
	var columns []schemaColumn
	if err := db.Raw("SELECT COLUMN_NAME,COLUMN_TYPE,IS_NULLABLE,COLUMN_DEFAULT,EXTRA FROM information_schema.COLUMNS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME=? ORDER BY ORDINAL_POSITION", table).Scan(&columns).Error; err != nil {
		return nil, nil, ErrQuerySchema
	}
	var indexes []schemaIndex
	if err := db.Raw("SELECT INDEX_NAME,COLUMN_NAME,SEQ_IN_INDEX,NON_UNIQUE,SUB_PART FROM information_schema.STATISTICS WHERE TABLE_SCHEMA=DATABASE() AND TABLE_NAME=? ORDER BY INDEX_NAME,SEQ_IN_INDEX", table).Scan(&indexes).Error; err != nil {
		return nil, nil, ErrQuerySchema
	}
	return columns, indexes, nil
}
func validateQueryTable(db *gorm.DB, table string, shape tableShape, partial bool) error {
	columns, indexes, err := inspectQueryTable(db, table)
	if err != nil {
		return err
	}
	if partial && len(columns) == 0 {
		return nil
	}
	if len(columns) != len(shape.Columns) {
		return ErrQuerySchema
	}
	for _, column := range columns {
		expected, exists := shape.Columns[column.Name]
		extra := strings.TrimSpace(strings.ReplaceAll(column.Extra, "DEFAULT_GENERATED", ""))
		if !exists || normalizedColumnType(column.Type) != expected.Type || (column.Nullable == "YES") != expected.Nullable || column.Default.String != expected.Default ||
			(expected.Default == "" && column.Default.Valid) || (expected.Default != "" && !column.Default.Valid) || extra != map[bool]string{true: "auto_increment", false: ""}[expected.Auto] {
			return ErrQuerySchema
		}
	}
	actual := map[string]expectedIndex{}
	for _, index := range indexes {
		if index.Prefix.Valid {
			return ErrQuerySchema
		}
		entry := actual[index.Name]
		entry.Unique = index.NonUnique == 0
		if index.Sequence != len(entry.Columns)+1 {
			return ErrQuerySchema
		}
		entry.Columns = append(entry.Columns, index.Column)
		actual[index.Name] = entry
	}
	if !partial && len(actual) != len(shape.Indexes) {
		return ErrQuerySchema
	}
	for name, entry := range actual {
		expected, exists := shape.Indexes[name]
		if !exists || entry.Unique != expected.Unique || strings.Join(entry.Columns, ",") != strings.Join(expected.Columns, ",") {
			return ErrQuerySchema
		}
	}
	if !partial && len(actual) != len(shape.Indexes) {
		return ErrQuerySchema
	}
	return nil
}
func queryWorkspaceShapeReady(db *gorm.DB) error {
	if db == nil || db.Dialector.Name() != "mysql" {
		return ErrQuerySchema
	}
	shapes := queryWorkspaceShapes()
	var tables []string
	for table := range shapes {
		tables = append(tables, table)
	}
	sort.Strings(tables)
	for _, table := range tables {
		if err := validateQueryTable(db, table, shapes[table], false); err != nil {
			return err
		}
	}
	return nil
}

// QueryWorkspaceReady inspects actual shape and dirty state before seed, opted
// routes or evaluator startup. A model or capability marker alone proves nothing.
func QueryWorkspaceReady(db *gorm.DB) error {
	if err := queryWorkspaceShapeReady(db); err != nil {
		return err
	}
	if err := queryMigrationStateReady(db); err != nil {
		return err
	}
	var capabilities []SchemaCapability
	if err := db.Where("capability = ?", QueryWorkspaceCapability).Find(&capabilities).Error; err != nil || len(capabilities) != 1 || capabilities[0].Version != 2 {
		return ErrQuerySchema
	}
	return nil
}
func BootstrapQueryWorkspaceDevelopment(db *gorm.DB) error {
	if db == nil || !database.ShouldAutoMigrate() {
		return ErrQuerySchema
	}
	if err := queryMigrationStateReady(db); err != nil {
		return err
	}
	// Never repair unknown existing shapes with an unconstrained AutoMigrate.
	if err := validateO4PartialSchema(db); err != nil {
		return err
	}
	// Apply the versioned guarded recipe for known legacy indexes/columns; GORM
	// alone cannot prove removal of the historical global name reservation.
	if state, err := QueryWorkspaceSchemaState(db); err == nil && (state == "legacy" || state == "empty") {
		sqlDB, err := db.DB()
		if err != nil {
			return ErrQuerySchema
		}
		conn, err := sqlDB.Conn(db.Statement.Context)
		if err != nil {
			return ErrQuerySchema
		}
		defer conn.Close()
		body, err := o4MigrationFiles.ReadFile("migrations/000037_o4_business_tenant_workspace.up.sql")
		if err != nil {
			return ErrQuerySchema
		}
		if err := executeO4SQL(db.Statement.Context, conn, body); err != nil {
			return err
		}
	}
	if err := db.AutoMigrate(&PrometheusDatasource{}, &QueryHistory{}, &SavedView{}, &QueryOwnerBudget{}, &SchemaCapability{}); err != nil {
		return ErrQuerySchema
	}
	if err := queryWorkspaceShapeReady(db); err != nil {
		return err
	}
	if err := db.Where("capability = ?", QueryWorkspaceCapability).FirstOrCreate(&SchemaCapability{Capability: QueryWorkspaceCapability, Version: 2}).Error; err != nil {
		return ErrQuerySchema
	}
	return QueryWorkspaceReady(db)
}
func queryMigrationStateReady(db *gorm.DB) error {
	if !db.Migrator().HasTable("schema_migrations") {
		return nil
	}
	var rows []struct {
		Version uint64
		Dirty   bool
	}
	if err := db.Raw("SELECT version,dirty FROM schema_migrations").Scan(&rows).Error; err != nil || len(rows) != 1 || rows[0].Dirty {
		return ErrQuerySchema
	}
	return nil
}

// QueryWorkspaceSchemaState is the supported-start shape classifier. Retained
// O4 columns/tables never become legacy just because a marker/version changes.
func QueryWorkspaceSchemaState(db *gorm.DB) (string, error) {
	if db == nil || db.Dialector.Name() != "mysql" {
		return "", ErrQuerySchema
	}
	if err := queryMigrationStateReady(db); err != nil {
		return "", err
	}
	if err := QueryWorkspaceReady(db); err == nil {
		return "o4", nil
	}
	for _, table := range []string{"obs_query_history", "obs_saved_view", "obs_query_owner_budget", "obs_schema_capability"} {
		if db.Migrator().HasTable(table) {
			return "", ErrQuerySchema
		}
	}
	if !db.Migrator().HasTable("obs_prometheus_datasource") {
		return "empty", nil
	}
	if err := validateQueryTable(db, "obs_prometheus_datasource", legacyDatasourceShape(), false); err != nil {
		return "", err
	}
	return "legacy", nil
}
