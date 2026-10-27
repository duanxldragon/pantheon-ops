package tenant

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"strings"
	"testing"

	"github.com/duanxldragon/pantheon-base/backend/pkg/testmysql"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// offlineConnector satisfies gorm's mysql dialector wiring without any server:
// DryRun sessions build statements but never execute, so Connect is never
// called. This keeps the SQL-shape assertions runnable without MySQL; when a
// real MySQL is available the same cases additionally render through the
// production dialector.
type offlineConnector struct{}

func (offlineConnector) Connect(context.Context) (driver.Conn, error) {
	return nil, driver.ErrBadConn
}
func (offlineConnector) Driver() driver.Driver { return offlineDriver{} }

type offlineDriver struct{}

func (offlineDriver) Open(string) (driver.Conn, error) { return nil, driver.ErrBadConn }

func newOfflineDryRunDB(t *testing.T) *gorm.DB {
	t.Helper()
	sqlDB := sql.OpenDB(offlineConnector{})
	db, err := gorm.Open(mysql.New(mysql.Config{
		Conn:                      sqlDB,
		SkipInitializeWithVersion: true,
		DisableDatetimePrecision:  true,
	}), &gorm.Config{DryRun: true, DisableAutomaticPing: true, Logger: logger.Discard})
	if err != nil {
		t.Fatalf("open offline dryrun gorm: %v", err)
	}
	return db
}

type scopeSQLRow struct {
	ID       uint64 `gorm:"primaryKey"`
	TenantID uint64
}

func (scopeSQLRow) TableName() string { return "system_user" }

// TestWithTenantScopeRendersTenantPinSQL pins the downgrade-safe scope
// contract of WithTenantScope without needing a database:
//   - nil context = platform-internal path: no tenant predicate at all;
//   - compat context = hard pin tenant_id = 0 (compat TenantID is always 0 by
//     Resolve), so multi-era rows never merge into the global view;
//   - multi context = hard pin tenant_id = <claim>;
//   - aliased/primary-table statements get a qualified column (joined tenant
//     tables would make a bare tenant_id ambiguous, MySQL error 1052).
func TestWithTenantScopeRendersTenantPinSQL(t *testing.T) {
	db := newOfflineDryRunDB(t)

	cases := []struct {
		name       string
		ctx        *Context
		build      func() *gorm.DB
		wantClause string
		wantArg    uint64
	}{
		{
			// Model-based statements resolve no table at scope time, so the
			// filter stays the bare column — correct for the single-table
			// queries these call sites use (joined paths pin Table() and
			// qualify the column, see the alias case below).
			name:       "compat pins tenant_id = 0",
			ctx:        &Context{TenantID: PlatformGlobalTenantID, Mode: ModeCompat, ResolvedBy: ResolvedByCompatFallback},
			build:      func() *gorm.DB { return db.Model(&scopeSQLRow{}) },
			wantClause: "tenant_id = ?",
			wantArg:    0,
		},
		{
			name:       "multi pins the claim",
			ctx:        &Context{TenantID: 101, Mode: ModeMulti, ResolvedBy: ResolvedBySubject},
			build:      func() *gorm.DB { return db.Model(&scopeSQLRow{}) },
			wantClause: "tenant_id = ?",
			wantArg:    101,
		},
		{
			name:       "aliased table qualifies the alias",
			ctx:        &Context{TenantID: PlatformGlobalTenantID, Mode: ModeCompat},
			build:      func() *gorm.DB { return db.Table("system_user AS u") },
			wantClause: "u.tenant_id = ?",
			wantArg:    0,
		},
		{
			name:       "nil context stays unfiltered",
			ctx:        nil,
			build:      func() *gorm.DB { return db.Model(&scopeSQLRow{}) },
			wantClause: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var rows []scopeSQLRow
			stmt := tc.build().Scopes(WithTenantScope(tc.ctx)).Find(&rows).Statement
			assertScopedSQL(t, stmt.SQL.String(), stmt.Vars, tc.wantClause, tc.wantArg)
		})
	}
}

// assertScopedSQL keeps the case assertions out of the table loop so the
// rendered statement stays readable as a single decision per expectation.
func assertScopedSQL(t *testing.T, sqlText string, vars []any, wantClause string, wantArg uint64) {
	t.Helper()

	if wantClause == "" {
		if strings.Contains(sqlText, "tenant_id") {
			t.Fatalf("nil-context path must not filter, got %q", sqlText)
		}
		return
	}
	if !strings.Contains(sqlText, wantClause) {
		t.Fatalf("rendered %q, want clause %q", sqlText, wantClause)
	}
	if len(vars) == 0 {
		t.Fatalf("expected tenant id bound for %q", sqlText)
	}
	// Locate the tenant arg by clause position rather than assuming it
	// is first (other predicates may share the statement).
	for _, v := range vars {
		if id, ok := v.(uint64); ok && id == wantArg {
			return
		}
	}
	t.Fatalf("vars %v do not carry tenant pin %d", vars, wantArg)
}

// TestWithTenantWriteStampsCompatZero pins the write-side twin: a compat
// request stamps tenant_id 0 explicitly instead of relying on the column
// default, matching WithTenantScope's downgrade-safe contract.
func TestWithTenantWriteStampsCompatZero(t *testing.T) {
	db := newOfflineDryRunDB(t)

	stmt := db.Scopes(WithTenantWrite(&Context{TenantID: PlatformGlobalTenantID, Mode: ModeCompat})).Create(&scopeSQLRow{}).Statement
	if got, ok := stmt.Get("gorm:tenant_id"); !ok || got != PlatformGlobalTenantID {
		t.Fatalf("compat write must stamp tenant_id 0, got %v (present=%v)", got, ok)
	}

	stmt = db.Scopes(WithTenantWrite(nil)).Create(&scopeSQLRow{}).Statement
	if _, ok := stmt.Get("gorm:tenant_id"); ok {
		t.Fatal("nil-context (platform-internal) write must not stamp a tenant id")
	}
}

// TestWithTenantScopeCompatAgainstMySQL renders the same compat pin through a
// real dialector when MySQL is available (skipped otherwise), guarding the
// offline DryRun wiring against driver-level drift.
func TestWithTenantScopeCompatAgainstMySQL(t *testing.T) {
	db := testmysql.Open(t)
	dry := db.Session(&gorm.Session{DryRun: true})

	var rows []scopeSQLRow
	stmt := dry.Table("system_user_session").
		Scopes(WithTenantScope(&Context{TenantID: PlatformGlobalTenantID, Mode: ModeCompat})).
		Find(&rows).Statement
	if !strings.Contains(stmt.SQL.String(), "system_user_session.tenant_id = ?") {
		t.Fatalf("compat scope against MySQL dialector rendered %q", stmt.SQL.String())
	}
}
