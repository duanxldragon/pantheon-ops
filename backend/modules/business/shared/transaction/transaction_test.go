package transaction

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/pkg/testmysql"
	"gorm.io/gorm"
)

type typedNilBeginPool struct{ gorm.ConnPool }

func (typedNilBeginPool) BeginTx(context.Context, *sql.TxOptions) (gorm.ConnPool, error) {
	var tx *sql.Tx
	return tx, nil
}

func TestReadCommittedRejectsTypedNilBegin(t *testing.T) {
	db := readCommittedFixture(t)
	faulted := db.WithContext(context.Background())
	faulted.Statement.ConnPool = typedNilBeginPool{ConnPool: db.Statement.ConnPool}
	called := false
	var err error
	var panicValue any
	func() {
		defer func() { panicValue = recover() }()
		err = ReadCommittedTransaction(context.Background(), faulted, func(*gorm.DB) error { called = true; return nil }, nil)
	}()
	if panicValue != nil {
		t.Fatalf("typed-nil BEGIN rejection panicked during rollback: %v", panicValue)
	}
	if !errors.Is(err, ErrTransactionIsolationUnknown) || called {
		t.Fatalf("typed-nil BEGIN was not rejected before callback: err=%v called=%v", err, called)
	}
}

func readCommittedFixture(t *testing.T) *gorm.DB {
	t.Helper()
	db := testmysql.Open(t)
	if err := db.Exec("CREATE TABLE rc_proof_probe (id INT PRIMARY KEY)").Error; err != nil {
		t.Fatal(err)
	}
	return db
}

func TestReadCommittedTransactionFreshReads(t *testing.T) {
	db := readCommittedFixture(t)
	if err := ReadCommittedTransaction(context.Background(), db, func(tx *gorm.DB) error {
		var count int64
		if err := tx.Table("rc_proof_probe").Count(&count).Error; err != nil {
			return err
		}
		if count != 0 {
			t.Fatal("initial helper read unexpectedly contained a committed reference")
		}
		if err := db.Exec("INSERT INTO rc_proof_probe VALUES (1)").Error; err != nil {
			return err
		}
		if err := tx.Table("rc_proof_probe").Count(&count).Error; err != nil {
			return err
		}
		if count != 1 {
			t.Fatal("same helper transaction retained a stale snapshot after separate commit")
		}
		return nil
	}, nil); err != nil {
		t.Fatal(err)
	}
}

func TestReadCommittedProofContextAndPoolIdentity(t *testing.T) {
	db := readCommittedFixture(t)
	type requestKey struct{}
	ctx, cancel := context.WithTimeout(context.WithValue(context.Background(), requestKey{}, "request"), time.Minute)
	defer cancel()
	if err := ReadCommittedTransaction(ctx, db, func(tx *gorm.DB) error {
		if !IsReadCommittedTransaction(tx) || tx.Statement.Context.Value(requestKey{}) != "request" {
			t.Fatal("fresh transaction lost isolation proof or request context")
		}
		if deadline, ok := tx.Statement.Context.Deadline(); !ok || time.Until(deadline) > time.Minute {
			t.Fatal("fresh transaction lost request deadline")
		}
		clone := tx.Session(&gorm.Session{SkipDefaultTransaction: true})
		clone = clone.WithContext(WithReadCommittedContext(context.Background(), tx))
		if !IsReadCommittedTransaction(clone) {
			t.Fatal("same-pool clone lost valid proof")
		}
		// A structurally identical wrapper is still a different connection identity.
		poolCopy := *tx.Statement.ConnPool.(*readCommittedPool)
		other := tx.Session(&gorm.Session{NewDB: true}).WithContext(tx.Statement.Context)
		other.Statement.ConnPool = &poolCopy
		if IsReadCommittedTransaction(other) {
			t.Fatal("distinct same-structure wrapper borrowed original proof")
		}
		for _, isolation := range []sql.IsolationLevel{sql.LevelDefault, sql.LevelRepeatableRead, sql.LevelReadCommitted} {
			raw := db.Begin(&sql.TxOptions{Isolation: isolation})
			if raw.Error != nil {
				return raw.Error
			}
			copied := raw.WithContext(tx.Statement.Context)
			if !errors.Is(RequireReadCommittedTransaction(copied), ErrTransactionIsolationUnknown) {
				t.Fatal("different SQL transaction borrowed proof context")
			}
			called := false
			err := ReadCommittedTransaction(ctx, copied, func(*gorm.DB) error { called = true; return nil }, nil)
			if err := raw.Rollback().Error; err != nil {
				return err
			}
			if !errors.Is(err, ErrTransactionIsolationUnknown) || called {
				t.Fatal("unknown transaction was accepted or executed callback")
			}
		}
		lost := tx.WithContext(context.Background())
		if IsReadCommittedTransaction(lost) {
			t.Fatal("transaction without proof was trusted")
		}
		resurrected := lost.WithContext(WithReadCommittedContext(tx.Statement.Context, lost))
		if IsReadCommittedTransaction(resurrected) {
			t.Fatal("supplied request context resurrected lost proof")
		}
		var nilTx *sql.Tx
		invalid := tx.Session(&gorm.Session{NewDB: true}).WithContext(tx.Statement.Context)
		invalid.Statement.ConnPool = nilTx
		if IsReadCommittedTransaction(invalid) {
			t.Fatal("typed-nil transaction was trusted")
		}
		if IsReadCommittedTransaction(nil) || IsReadCommittedTransaction(&gorm.DB{}) {
			t.Fatal("uninitialized transaction was trusted")
		}
		return nil
	}, nil); err != nil {
		t.Fatal(err)
	}
}

func TestReadCommittedNestedSavepointAndOuterRollback(t *testing.T) {
	db := readCommittedFixture(t)
	nestedFailure, outerFailure := errors.New("rollback nested"), errors.New("rollback outer")
	err := ReadCommittedTransaction(context.Background(), db, func(tx *gorm.DB) error {
		if err := tx.Exec("INSERT INTO rc_proof_probe VALUES (1)").Error; err != nil {
			return err
		}
		err := ReadCommittedTransaction(context.Background(), tx, func(nested *gorm.DB) error {
			if nested.Statement.ConnPool != tx.Statement.ConnPool || !IsReadCommittedTransaction(nested) {
				t.Fatal("nested helper changed pool or lost proof")
			}
			if err := nested.Exec("INSERT INTO rc_proof_probe VALUES (2)").Error; err != nil {
				return err
			}
			return nestedFailure
		}, nil)
		if !errors.Is(err, nestedFailure) {
			t.Fatalf("nested rollback lost error: %v", err)
		}
		var count int64
		if err := tx.Table("rc_proof_probe").Count(&count).Error; err != nil {
			return err
		}
		if count != 1 {
			t.Fatal("savepoint rollback did not preserve only outer write")
		}
		called := false
		err = ReadCommittedTransaction(context.Background(), tx, func(*gorm.DB) error { called = true; return nil }, &ReadCommittedTransactionHooks{})
		if !errors.Is(err, ErrTransactionIsolationUnknown) || called {
			t.Fatal("nested helper installed new hooks")
		}
		canceled, cancel := context.WithCancel(context.Background())
		cancel()
		err = ReadCommittedTransaction(canceled, tx, func(*gorm.DB) error { called = true; return nil }, nil)
		if !errors.Is(err, context.Canceled) || called {
			t.Fatal("canceled nested helper executed callback")
		}
		return outerFailure
	}, nil)
	if !errors.Is(err, outerFailure) {
		t.Fatalf("outer rollback lost error: %v", err)
	}
	var count int64
	if err := db.Table("rc_proof_probe").Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatal("helper independently committed outer writes")
	}
}

func TestReadCommittedCancellationRollsBack(t *testing.T) {
	db := readCommittedFixture(t)
	ctx, cancel := context.WithCancelCause(context.Background())
	cause := errors.New("caller cancellation cause")
	err := ReadCommittedTransaction(ctx, db, func(tx *gorm.DB) error {
		if err := tx.Exec("INSERT INTO rc_proof_probe VALUES (1)").Error; err != nil {
			return err
		}
		cancel(cause)
		if !errors.Is(context.Cause(tx.Statement.Context), cause) {
			t.Fatal("bound transaction lost caller cancellation cause")
		}
		return nil
	}, nil)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("helper did not return cancellation: %v", err)
	}
	called := false
	if err := ReadCommittedTransaction(ctx, db, func(*gorm.DB) error { called = true; return nil }, nil); !errors.Is(err, context.Canceled) || called {
		t.Fatal("canceled fresh helper executed callback")
	}
	var count int64
	if err := db.Table("rc_proof_probe").Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatal("canceled helper committed writes")
	}
}

func TestReadCommittedHooksFenceActualOuterCommit(t *testing.T) {
	db := readCommittedFixture(t)
	blocked := errors.New("commit fence refused")
	commits, failures := 0, 0
	hooks := &ReadCommittedTransactionHooks{
		BeforeCommit: func() error { commits++; return blocked },
		OnError: func(err error) {
			if !errors.Is(err, blocked) {
				t.Fatalf("lost commit fence error: %v", err)
			}
			failures++
		},
	}
	err := ReadCommittedTransaction(context.Background(), db, func(tx *gorm.DB) error {
		return tx.Exec("INSERT INTO rc_proof_probe VALUES (1)").Error
	}, hooks)
	if !errors.Is(err, blocked) || commits != 1 || failures != 1 {
		t.Fatalf("outer commit bypassed hooks: err=%v commits=%d failures=%d", err, commits, failures)
	}
	var count int64
	if err := db.Table("rc_proof_probe").Count(&count).Error; err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatal("failed commit fence left committed writes")
	}
}

func TestReadCommittedHooksObserveSQLErrors(t *testing.T) {
	for _, entry := range []string{"exec", "query", "row"} {
		t.Run(entry, func(t *testing.T) {
			db := readCommittedFixture(t)
			var observed error
			err := ReadCommittedTransaction(context.Background(), db, func(tx *gorm.DB) error {
				switch entry {
				case "exec":
					return tx.Exec("INSERT INTO absent_rc_probe VALUES (1)").Error
				case "query":
					rows, err := tx.Statement.ConnPool.QueryContext(tx.Statement.Context, "SELECT * FROM absent_rc_probe")
					if rows != nil {
						rows.Close()
					}
					return err
				default:
					return tx.Statement.ConnPool.QueryRowContext(tx.Statement.Context, "SELECT * FROM absent_rc_probe").Err()
				}
			}, &ReadCommittedTransactionHooks{OnError: func(err error) { observed = err }})
			if err == nil || observed != err {
				t.Fatalf("SQL error did not reach hook before rollback: err=%v observed=%v", err, observed)
			}
		})
	}
}
