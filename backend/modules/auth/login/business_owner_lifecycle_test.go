package login

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/pkg/contracts/businessauth"
	"github.com/duanxldragon/pantheon-base/backend/pkg/tenant"
	"github.com/duanxldragon/pantheon-base/backend/pkg/testmysql"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func newOwnerLifecycleFixture(t *testing.T) (*gorm.DB, businessauth.OwnerLifecycleReader) {
	t.Helper()
	db := testmysql.Open(t)
	// Only authoritative lifecycle columns exist. No credential, role, session
	// or membership schema is available to accidentally authorize through.
	require.NoError(t, db.Exec("CREATE TABLE system_user (id BIGINT UNSIGNED PRIMARY KEY, tenant_id BIGINT UNSIGNED NOT NULL, status INT NOT NULL, deleted_at DATETIME(3) NULL)").Error)
	require.NoError(t, db.Exec("CREATE TABLE tenants (id BIGINT UNSIGNED PRIMARY KEY, status VARCHAR(16) NOT NULL)").Error)
	require.NoError(t, db.Exec("INSERT INTO system_user (id, tenant_id, status) VALUES (7, 0, 1), (8, 0, 2), (9, 101, 1)").Error)
	require.NoError(t, db.Exec("INSERT INTO tenants (id, status) VALUES (101, 'active'), (102, 'suspended'), (103, 'archived'), (104, 'deleted')").Error)
	reader, ok := NewBusinessAuthorizationReader(&Runtime{db: db}, nil, nil).(businessauth.OwnerLifecycleReader)
	require.True(t, ok)
	return db, reader
}

func TestBusinessOwnerLifecycleExistenceIsIndependentOfAuthorization(t *testing.T) {
	db, reader := newOwnerLifecycleFixture(t)
	var queries []string
	require.NoError(t, db.Callback().Query().After("gorm:query").Register("test:lifecycle-projections", func(tx *gorm.DB) {
		queries = append(queries, strings.ToLower(tx.Statement.SQL.String()))
	}))
	for _, tc := range []struct {
		tenant, actor             uint64
		tenantExists, actorExists bool
	}{
		{101, 7, true, true}, // Platform actor without membership in target tenant.
		{101, 8, true, true}, // Disabled actor still owns its private metadata.
		{102, 9, true, true}, {103, 9, true, true},
		{0, 7, true, true}, {0, 8, true, true},
		{104, 7, false, true}, {999, 7, false, true},
		{101, 999, true, false}, {0, 999, true, false},
	} {
		got, err := reader.ReadOwnerLifecycle(context.Background(), tc.tenant, tc.actor)
		require.NoError(t, err)
		require.Equal(t, businessauth.OwnerLifecycle{TenantExists: tc.tenantExists, ActorExists: tc.actorExists}, got)
		raw, err := json.Marshal(got)
		require.NoError(t, err)
		require.Equal(t, "{}", string(raw), "lifecycle projection must remain private")
	}
	require.NoError(t, db.Exec("UPDATE system_user SET deleted_at = ? WHERE id = ?", time.Now(), 7).Error)
	got, err := reader.ReadOwnerLifecycle(context.Background(), 101, 7)
	require.NoError(t, err)
	require.Equal(t, businessauth.OwnerLifecycle{TenantExists: true, ActorExists: false}, got)
	for _, query := range queries {
		require.Contains(t, query, "count(*)")
		for _, secret := range []string{"password", "username", "session", "role", "membership", "select *"} {
			require.NotContains(t, query, secret)
		}
	}
}

func TestBusinessOwnerLifecycleUnknownIsAlwaysAnError(t *testing.T) {
	db, reader := newOwnerLifecycleFixture(t)
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	for _, call := range []func() (businessauth.OwnerLifecycle, error){
		// SA1012: the nil-context guard is exactly the behaviour under test.
		func() (businessauth.OwnerLifecycle, error) { return reader.ReadOwnerLifecycle(nil, 101, 7) }, //nolint:staticcheck
		func() (businessauth.OwnerLifecycle, error) { return reader.ReadOwnerLifecycle(cancelled, 101, 7) },
		func() (businessauth.OwnerLifecycle, error) {
			return reader.ReadOwnerLifecycle(context.Background(), 101, 0)
		},
		func() (businessauth.OwnerLifecycle, error) {
			return (*businessAuthorizationReader)(nil).ReadOwnerLifecycle(context.Background(), 101, 7)
		},
		func() (businessauth.OwnerLifecycle, error) {
			return (&businessAuthorizationReader{}).ReadOwnerLifecycle(context.Background(), 101, 7)
		},
		func() (businessauth.OwnerLifecycle, error) {
			return (&businessAuthorizationReader{runtime: &Runtime{}}).ReadOwnerLifecycle(context.Background(), 101, 7)
		},
	} {
		got, err := call()
		require.ErrorIs(t, err, businessauth.ErrOwnerLifecycleUnavailable)
		require.Equal(t, businessauth.OwnerLifecycle{}, got)
	}
	// Failure after one successful query must not turn a partial snapshot into
	// proof that either owner was deleted, or expose the SQL/provider error.
	injected := errors.New("synthetic private schema failure")
	require.NoError(t, db.Callback().Query().Before("gorm:query").Register("test:lifecycle-query-failure", func(tx *gorm.DB) {
		if tx.Statement.Table == (tenant.Tenant{}).TableName() {
			tx.AddError(injected)
		}
	}))
	got, err := reader.ReadOwnerLifecycle(context.Background(), 101, 7)
	require.ErrorIs(t, err, businessauth.ErrOwnerLifecycleUnavailable)
	require.Equal(t, businessauth.OwnerLifecycle{}, got)
	require.NotContains(t, err.Error(), injected.Error())
	require.NoError(t, db.Callback().Query().Remove("test:lifecycle-query-failure"))
	// A cancellation arriving after SQL returned still leaves the result unknown.
	duringRead, cancelRead := context.WithCancel(context.Background())
	defer cancelRead()
	require.NoError(t, db.Callback().Query().After("gorm:query").Register("test:lifecycle-cancel", func(tx *gorm.DB) {
		if tx.Statement.Context == duringRead {
			cancelRead()
		}
	}))
	got, err = reader.ReadOwnerLifecycle(duringRead, 0, 7)
	require.ErrorIs(t, err, businessauth.ErrOwnerLifecycleUnavailable)
	require.Equal(t, businessauth.OwnerLifecycle{}, got)
	require.NoError(t, db.Callback().Query().Remove("test:lifecycle-cancel"))
	broken := db.Session(&gorm.Session{})
	broken.AddError(injected)
	got, err = (&businessAuthorizationReader{runtime: &Runtime{db: broken}}).ReadOwnerLifecycle(context.Background(), 0, 7)
	require.ErrorIs(t, err, businessauth.ErrOwnerLifecycleUnavailable)
	require.Equal(t, businessauth.OwnerLifecycle{}, got)
	require.NoError(t, db.Exec("DROP TABLE tenants").Error)
	got, err = reader.ReadOwnerLifecycle(context.Background(), 101, 7)
	require.ErrorIs(t, err, businessauth.ErrOwnerLifecycleUnavailable)
	require.Equal(t, businessauth.OwnerLifecycle{}, got)
	require.NoError(t, db.Exec("ALTER TABLE system_user DROP COLUMN deleted_at").Error)
	got, err = reader.ReadOwnerLifecycle(context.Background(), 0, 7)
	require.ErrorIs(t, err, businessauth.ErrOwnerLifecycleUnavailable)
	require.Equal(t, businessauth.OwnerLifecycle{}, got)
	// The implicit platform owner must still check the actor's real schema.
	require.NoError(t, db.Exec("DROP TABLE system_user").Error)
	_, err = reader.ReadOwnerLifecycle(context.Background(), 0, 7)
	require.ErrorIs(t, err, businessauth.ErrOwnerLifecycleUnavailable)
}
