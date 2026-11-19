package service

import (
	"context"
	"errors"

	biztx "github.com/duanxldragon/pantheon-base/backend/modules/business/shared/transaction"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"gorm.io/gorm"
)

// WithReaderTransaction binds owner reads to the caller's transaction. Current
// reads lock instance -> service -> application; deletion uses the same order.
func (m *Manager) WithReaderTransaction(tx *gorm.DB) Reader {
	return &transactionReader{owner: m, tx: tx}
}

type transactionReader struct {
	owner *Manager
	tx    *gorm.DB
}

func (m *Manager) transactionManager(ctx context.Context, tx *gorm.DB) (*Manager, error) {
	if m == nil || tx == nil || tx.Statement == nil {
		return nil, errors.New(errNotInitialized)
	}
	if _, ok := tx.Statement.ConnPool.(gorm.TxCommitter); !ok {
		return nil, errors.New("service.state.transactionRequired")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	owner := *m
	owner.db = tx.Session(&gorm.Session{SkipDefaultTransaction: true}).WithContext(biztx.WithReadCommittedContext(ctx, tx))
	owner.lockReads = true
	return &owner, nil
}

func (r *transactionReader) GetApplication(ctx context.Context, id uint64, scope *common.DataScopeReq) (ApplicationRef, error) {
	owner, err := r.owner.transactionManager(ctx, r.tx)
	if err != nil {
		return ApplicationRef{}, err
	}
	return owner.GetApplication(ctx, id, scope)
}

func (r *transactionReader) GetService(ctx context.Context, id uint64, scope *common.DataScopeReq) (ServiceRef, error) {
	owner, err := r.owner.transactionManager(ctx, r.tx)
	if err != nil {
		return ServiceRef{}, err
	}
	return owner.GetService(ctx, id, scope)
}

func (r *transactionReader) GetInstance(ctx context.Context, id uint64, scope *common.DataScopeReq) (InstanceRef, error) {
	owner, err := r.owner.transactionManager(ctx, r.tx)
	if err != nil {
		return InstanceRef{}, err
	}
	return owner.GetInstance(ctx, id, scope)
}
