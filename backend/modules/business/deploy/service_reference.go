package deploy

import (
	"context"
	"errors"

	bizcap "github.com/duanxldragon/pantheon-base/backend/modules/business/capability"
	biztx "github.com/duanxldragon/pantheon-base/backend/modules/business/shared/transaction"
	"gorm.io/gorm"
)

// WithDeploymentReferenceTransaction binds reference reads to the Service
// owner transaction. These reads never lock Deploy task rows; verified READ
// COMMITTED preserves current references and the caller's own terminal writes.
func (s *DeployService) WithDeploymentReferenceTransaction(tx *gorm.DB) bizcap.DeploymentReferenceReader {
	return &deploymentReferences{db: tx, transactionRequired: true}
}

// HasExecutingServiceReferences reports whether any executing deploy task still
// references the given service, optionally narrowed to a single instance.
func (s *DeployService) HasExecutingServiceReferences(ctx context.Context, serviceID, instanceID uint64) (bool, error) {
	return (&deploymentReferences{db: s.db}).HasExecutingServiceReferences(ctx, serviceID, instanceID)
}

type deploymentReferences struct {
	db                  *gorm.DB
	transactionRequired bool
}

func (r *deploymentReferences) HasExecutingServiceReferences(ctx context.Context, serviceID, instanceID uint64) (bool, error) {
	if r.db == nil || r.db.Statement == nil {
		return false, errors.New("database.not_initialized")
	}
	if r.transactionRequired {
		if _, ok := r.db.Statement.ConnPool.(gorm.TxCommitter); !ok {
			return false, errors.New("service.state.transactionRequired")
		}
		if err := biztx.RequireReadCommittedTransaction(r.db); err != nil {
			return false, err
		}
	}
	query := r.db.WithContext(ctx).Model(&DeployTask{}).
		Where("status IN ?", []string{TaskStatusRunning, TaskStatusPauseRequested, TaskStatusPaused, TaskStatusCancelRequested})
	if instanceID != 0 {
		query = query.Where("service_instance_id = ?", instanceID)
	} else {
		query = query.Where("service_id = ?", serviceID)
	}
	var count int64
	err := query.Count(&count).Error
	return count != 0, err
}
