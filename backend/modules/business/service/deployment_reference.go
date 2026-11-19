package service

import (
	"context"
	"errors"
	"reflect"

	bizcap "github.com/duanxldragon/pantheon-base/backend/modules/business/capability"
	biztx "github.com/duanxldragon/pantheon-base/backend/modules/business/shared/transaction"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"gorm.io/gorm"
)

// SetDeploymentReferenceReader connects Deploy before its recovery workers
// start. Missing transaction-capable owners reject deletion and retirement.
func (m *Manager) SetDeploymentReferenceReader(reader bizcap.DeploymentReferenceReader) {
	if nilDeploymentReader(reader) {
		reader = nil
	}
	m.deploymentReader = reader
	m.deploymentReaderTransaction = nil
	if owner, ok := reader.(interface {
		WithDeploymentReferenceTransaction(*gorm.DB) bizcap.DeploymentReferenceReader
	}); ok {
		m.deploymentReaderTransaction = owner.WithDeploymentReferenceTransaction
	}
}

func (m *Manager) checkDeploymentReferences(ctx context.Context, serviceID, instanceID uint64, inUse string) error {
	if m.deploymentReader == nil {
		return errors.New("business.service.deployment_reader_not_configured")
	}
	if m.deploymentReaderTransaction == nil {
		return errors.New("business.service.deployment_reader_not_configured")
	}
	reader := m.deploymentReaderTransaction(m.db)
	if nilDeploymentReader(reader) {
		return errors.New("business.service.deployment_reader_not_configured")
	}
	active, err := reader.HasExecutingServiceReferences(ctx, serviceID, instanceID)
	if err != nil {
		return err
	}
	if active {
		return errors.New(inUse)
	}
	return nil
}

func nilDeploymentReader(reader bizcap.DeploymentReferenceReader) bool {
	if reader == nil {
		return true
	}
	value := reflect.ValueOf(reader)
	switch value.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return value.IsNil()
	default:
		return false
	}
}

func (m *Manager) deleteServiceWithReferences(id uint64, scope *common.DataScopeReq) error {
	if m.db == nil {
		return errors.New(errNotInitialized)
	}
	return biztx.ReadCommittedTransaction(m.db.Statement.Context, m.db, func(tx *gorm.DB) error {
		owner, err := m.transactionManager(tx.Statement.Context, tx)
		if err != nil {
			return err
		}
		row, err := owner.findService(id, scope)
		if err != nil {
			return err
		}
		var count int64
		if err := owner.db.Model(&ServiceInstance{}).Where(whereServiceIDEq, row.ID).Count(&count).Error; err != nil {
			return err
		}
		if count != 0 {
			return errors.New(errServiceInUse)
		}
		if err := owner.checkDeploymentReferences(tx.Statement.Context, row.ID, 0, errServiceInUse); err != nil {
			return err
		}
		return owner.db.Delete(row).Error
	}, nil)
}

func (m *Manager) deleteInstanceWithReferences(id uint64, scope *common.DataScopeReq) error {
	if m.db == nil {
		return errors.New(errNotInitialized)
	}
	return biztx.ReadCommittedTransaction(m.db.Statement.Context, m.db, func(tx *gorm.DB) error {
		owner, err := m.transactionManager(tx.Statement.Context, tx)
		if err != nil {
			return err
		}
		row, err := owner.findInstance(id, scope)
		if err != nil {
			return err
		}
		if err := owner.checkDeploymentReferences(tx.Statement.Context, row.ServiceID, row.ID, "business.service.instance_in_use"); err != nil {
			return err
		}
		return owner.db.Delete(row).Error
	}, nil)
}
