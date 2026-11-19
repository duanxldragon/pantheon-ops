package transaction

import (
	"context"
	"database/sql"
	"errors"
	"reflect"

	"gorm.io/gorm"
)

// ErrTransactionIsolationUnknown reports that a *gorm.DB lacks the exact
// private READ COMMITTED wrapper created by ReadCommittedTransaction, so the
// caller must not assume READ COMMITTED isolation semantics.
var ErrTransactionIsolationUnknown = errors.New("service.state.transactionIsolationUnknown")

type readCommittedProofKey struct{}

// ReadCommittedTransactionHooks observes the helper's own transaction. Hooks
// cannot replace its connection or certify an arbitrary caller transaction.
type ReadCommittedTransactionHooks struct {
	BeforeCommit func() error
	OnError      func(error)
}

type readCommittedPool struct {
	gorm.ConnPool
	commit gorm.TxCommitter
	ctx    context.Context
	hooks  ReadCommittedTransactionHooks
}

func (p *readCommittedPool) failed(err error) {
	if err != nil && p.hooks.OnError != nil {
		p.hooks.OnError(err)
	}
}

func (p *readCommittedPool) Commit() error {
	if p.hooks.BeforeCommit != nil {
		if err := p.hooks.BeforeCommit(); err != nil {
			p.failed(err)
			return err
		}
	}
	err := p.commit.Commit()
	if err == nil {
		err = p.ctx.Err()
	}
	p.failed(err)
	return err
}

func (p *readCommittedPool) Rollback() error { return p.commit.Rollback() }

func (p *readCommittedPool) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	result, err := p.ConnPool.ExecContext(ctx, query, args...)
	p.failed(err)
	return result, err
}

func (p *readCommittedPool) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	rows, err := p.ConnPool.QueryContext(ctx, query, args...)
	p.failed(err)
	return rows, err
}

func (p *readCommittedPool) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	row := p.ConnPool.QueryRowContext(ctx, query, args...)
	p.failed(row.Err())
	return row
}

// IsReadCommittedTransaction verifies the exact private wrapper created after
// this helper's outermost READ COMMITTED begin. Copied context values on another
// pool, lost proof, arbitrary caller transactions and canceled scopes fail closed.
func IsReadCommittedTransaction(tx *gorm.DB) bool {
	if tx == nil || tx.Statement == nil || tx.Statement.Context == nil || tx.Statement.Context.Err() != nil {
		return false
	}
	pool, ok := tx.Statement.ConnPool.(*readCommittedPool)
	if !ok || pool == nil || pool.ctx == nil || pool.ctx.Err() != nil {
		return false
	}
	proof, ok := tx.Statement.Context.Value(readCommittedProofKey{}).(*readCommittedPool)
	return ok && proof == pool
}

// RequireReadCommittedTransaction returns either nil when tx carries the
// verified READ COMMITTED proof or ErrTransactionIsolationUnknown otherwise.
func RequireReadCommittedTransaction(tx *gorm.DB) error {
	if !IsReadCommittedTransaction(tx) {
		return ErrTransactionIsolationUnknown
	}
	return nil
}

// WithReadCommittedContext carries verified proof only from the source pool.
// Invalid sources shadow any proof already present in the supplied context.
func WithReadCommittedContext(ctx context.Context, tx *gorm.DB) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	var proof *readCommittedPool
	if IsReadCommittedTransaction(tx) {
		proof, _ = tx.Statement.ConnPool.(*readCommittedPool)
	}
	return context.WithValue(ctx, readCommittedProofKey{}, proof)
}

// ReadCommittedTransaction begins an actual READ COMMITTED transaction. Nested
// verified calls use GORM savepoints and cannot independently commit or add hooks.
func ReadCommittedTransaction(ctx context.Context, db *gorm.DB, write func(*gorm.DB) error, hooks *ReadCommittedTransactionHooks) error {
	if ctx == nil {
		return ErrTransactionIsolationUnknown
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if db == nil || db.Statement == nil {
		return errors.New("database.not_initialized")
	}
	if _, active := db.Statement.ConnPool.(gorm.TxCommitter); active {
		if err := RequireReadCommittedTransaction(db); err != nil {
			return err
		}
		if hooks != nil {
			return ErrTransactionIsolationUnknown
		}
		return db.WithContext(WithReadCommittedContext(ctx, db)).Transaction(func(tx *gorm.DB) error {
			if err := ctx.Err(); err != nil {
				return err
			}
			if err := write(tx); err != nil {
				return err
			}
			return ctx.Err()
		})
	}
	return db.WithContext(WithReadCommittedContext(ctx, db)).Transaction(func(tx *gorm.DB) error {
		committer, ok := tx.Statement.ConnPool.(gorm.TxCommitter)
		if !ok || nilConnPool(tx.Statement.ConnPool) {
			return ErrTransactionIsolationUnknown
		}
		pool := &readCommittedPool{ConnPool: tx.Statement.ConnPool, commit: committer, ctx: ctx}
		if hooks != nil {
			pool.hooks = *hooks
		}
		// Mutate the original callback transaction so GORM's outer Commit uses
		// the same wrapper whose exact identity is recorded by the proof.
		tx.Statement.ConnPool = pool
		tx.Statement.Context = context.WithValue(tx.Statement.Context, readCommittedProofKey{}, pool)
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := write(tx); err != nil {
			return err
		}
		return ctx.Err()
	}, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
}

func nilConnPool(pool gorm.ConnPool) bool {
	if pool == nil {
		return true
	}
	value := reflect.ValueOf(pool)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}
