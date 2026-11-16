package observability

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"gorm.io/gorm"
)

//go:embed migrations/000037_o4_business_tenant_workspace.*.sql
var o4MigrationFiles embed.FS

// O4RecoveryOwnership records the scope of a migration ownership claim.
type O4RecoveryOwnership struct {
	Database                string `json:"database"`
	BeforeDatasourceSHA256  string `json:"beforeDatasourceSha256"`
	QuiescenceReceiptSHA256 string `json:"quiescenceReceiptSha256"`
	ExclusiveWritersStopped bool   `json:"exclusiveWritersStopped"`
}

// O4RecoveryReceipt describes the before/after state of a completed recovery.
type O4RecoveryReceipt struct {
	Database                string    `json:"database"`
	BeforeVersion           uint64    `json:"beforeVersion"`
	BeforeDirty             bool      `json:"beforeDirty"`
	AfterVersion            uint64    `json:"afterVersion"`
	AfterDirty              bool      `json:"afterDirty"`
	DatasourceSHA256        string    `json:"datasourceSha256"`
	MigrationSHA256         string    `json:"migrationSha256"`
	QuiescenceReceiptSHA256 string    `json:"quiescenceReceiptSha256"`
	RecoveredAt             time.Time `json:"recoveredAt"`
}

var disposableO4Database = regexp.MustCompile(`^ops_p0_acceptance_testo4_[a-z0-9_]+_[0-9]{19}_[0-9]{4}$`)
var o4DigestPattern = regexp.MustCompile(`^[a-f0-9]{64}$`)

func legacyDatasourceShape() tableShape {
	shape := queryWorkspaceShapes()[datasourceTableName]
	delete(shape.Columns, "tenant_id")
	delete(shape.Columns, "revision")
	delete(shape.Indexes, "idx_datasource_tenant_name")
	delete(shape.Indexes, "idx_datasource_tenant_id")
	delete(shape.Indexes, "idx_datasource_tenant_default")
	shape.Indexes["idx_obs_prometheus_datasource_name"] = expectedIndex{Columns: []string{"name"}, Unique: true}
	return shape
}

// Known partial37 states are whole DDL steps, never guessed column repairs.
func validateO4PartialSchema(db *gorm.DB) error {
	for table, shape := range queryWorkspaceShapes() {
		columns, indexes, err := inspectQueryTable(db, table)
		if err != nil {
			return err
		}
		if len(columns) == 0 {
			continue
		}
		if table != datasourceTableName {
			if err := validateQueryTable(db, table, shape, true); err != nil {
				return err
			}
			continue
		}
		hasTenant, hasRevision := false, false
		for _, column := range columns {
			hasTenant = hasTenant || column.Name == "tenant_id"
			hasRevision = hasRevision || column.Name == "revision"
		}
		if hasRevision && !hasTenant {
			return ErrQuerySchema
		}
		if !hasTenant {
			delete(shape.Columns, "tenant_id")
		}
		if !hasRevision {
			delete(shape.Columns, "revision")
		}
		for _, index := range indexes {
			if index.Name == "idx_obs_prometheus_datasource_name" {
				shape.Indexes[index.Name] = expectedIndex{Columns: []string{"name"}, Unique: true}
			}
		}
		if err := validateQueryTable(db, table, shape, true); err != nil {
			return err
		}
	}
	return nil
}

// DatasourceRetentionSHA256 hashes legacy data/ciphertext without exposing any
// field value. New ownership/revision columns are excluded from this invariant.
func DatasourceRetentionSHA256(db *gorm.DB) (string, error) {
	if db == nil {
		return "", ErrQuerySchema
	}
	if !db.Migrator().HasTable(datasourceTableName) {
		sum := sha256.Sum256(nil)
		return hex.EncodeToString(sum[:]), nil
	}
	var rows []map[string]interface{}
	if err := db.Table(datasourceTableName).Select("id,dept_id,name,type,url,access_mode,username,password,bearer_token,timeout,is_default,status,last_checked,created_at,updated_at,created_by,deleted_at").Order("id").Find(&rows).Error; err != nil {
		return "", ErrQuerySchema
	}
	hash := sha256.New()
	for _, row := range rows {
		bytes, err := json.Marshal(row)
		if err != nil {
			return "", ErrQuerySchema
		}
		_, _ = hash.Write(bytes)
		_, _ = hash.Write([]byte{10})
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}
func executeO4SQL(ctx context.Context, connection *sql.Conn, body []byte) error {
	// This repository-owned recipe has no routines or literals containing
	// semicolons, delimiter changes or external SQL input.
	var lines []string
	for _, line := range strings.Split(string(body), "\n") {
		if !strings.HasPrefix(strings.TrimSpace(line), "--") {
			lines = append(lines, line)
		}
	}
	for step, statement := range strings.Split(strings.Join(lines, "\n"), ";") {
		if strings.TrimSpace(statement) == "" {
			continue
		}
		if _, err := connection.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("%w:step_%d", ErrQuerySchema, step)
		}
	}
	return nil
}

// RecoverO4Migration37 is never called by ordinary startup. It accepts only an
// explicitly owned disposable fixture, holds cooperative exclusivity, preserves
// dirty37 on every failure, and marks clean only after full shape/data checks.
func RecoverO4Migration37(ctx context.Context, db *gorm.DB, ownership O4RecoveryOwnership) (O4RecoveryReceipt, error) {
	var receipt O4RecoveryReceipt
	if db == nil || !disposableO4Database.MatchString(ownership.Database) || !ownership.ExclusiveWritersStopped ||
		!o4DigestPattern.MatchString(ownership.BeforeDatasourceSHA256) || !o4DigestPattern.MatchString(ownership.QuiescenceReceiptSHA256) {
		return receipt, ErrQuerySchema
	}
	var actual string
	if err := db.Raw("SELECT DATABASE()").Scan(&actual).Error; err != nil || actual != ownership.Database {
		return receipt, ErrQuerySchema
	}
	sqlDB, err := db.DB()
	if err != nil {
		return receipt, ErrQuerySchema
	}
	conn, err := sqlDB.Conn(ctx)
	if err != nil {
		return receipt, ErrQuerySchema
	}
	defer func() { _ = conn.Close() }()
	lockHash := sha256.Sum256([]byte(actual))
	lockName := "o4-recovery:" + hex.EncodeToString(lockHash[:20])
	var acquired int
	if err := conn.QueryRowContext(ctx, "SELECT GET_LOCK(?,0)", lockName).Scan(&acquired); err != nil || acquired != 1 {
		return receipt, ErrQuerySchema
	}
	defer func() {
		cleanup, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_, _ = conn.ExecContext(cleanup, "SELECT RELEASE_LOCK(?)", lockName)
	}()
	var state []struct {
		Version uint64
		Dirty   bool
	}
	if err := db.Raw("SELECT version,dirty FROM schema_migrations").Scan(&state).Error; err != nil || len(state) != 1 || state[0].Version != 37 || !state[0].Dirty {
		return receipt, ErrQuerySchema
	}
	receipt = O4RecoveryReceipt{Database: actual, BeforeVersion: 37, BeforeDirty: true, QuiescenceReceiptSHA256: ownership.QuiescenceReceiptSHA256}
	before, err := DatasourceRetentionSHA256(db)
	if err != nil || before != ownership.BeforeDatasourceSHA256 {
		return receipt, ErrQuerySchema
	}
	if err := validateO4PartialSchema(db); err != nil {
		return receipt, err
	}
	body, err := o4MigrationFiles.ReadFile("migrations/000037_o4_business_tenant_workspace.up.sql")
	if err != nil {
		return receipt, ErrQuerySchema
	}
	sum := sha256.Sum256(body)
	receipt.MigrationSHA256 = hex.EncodeToString(sum[:])
	if err := executeO4SQL(ctx, conn, body); err != nil {
		return receipt, err
	}
	if err := queryWorkspaceShapeReady(db); err != nil {
		return receipt, err
	}
	var capability SchemaCapability
	if err := db.Where("capability = ? AND version=2", QueryWorkspaceCapability).First(&capability).Error; err != nil {
		return receipt, ErrQuerySchema
	}
	after, err := DatasourceRetentionSHA256(db)
	if err != nil || after != before {
		return receipt, ErrQuerySchema
	}
	// Narrow equivalent of Force37, guarded by the exact original dirty state.
	result := db.WithContext(ctx).Exec("UPDATE schema_migrations SET dirty=false WHERE version=37 AND dirty=true")
	if result.Error != nil || result.RowsAffected != 1 {
		return receipt, ErrQuerySchema
	}
	if err := QueryWorkspaceReady(db); err != nil {
		return receipt, err
	}
	receipt.AfterVersion = 37
	receipt.AfterDirty = false
	receipt.DatasourceSHA256 = after
	receipt.RecoveredAt = time.Now().UTC()
	return receipt, nil
}
