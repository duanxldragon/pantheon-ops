package backup

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/duanxldragon/pantheon-base/backend/modules/business/k8s/cluster"
	"github.com/duanxldragon/pantheon-base/backend/modules/business/k8s/namespace"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

const (
	StatusUnknown            = "Unknown"
	phasePendingSubmission   = "PendingSubmission"
	phaseSubmissionUnknown   = "SubmissionUnknown"
	phaseSubmitted           = "Submitted"
	recordIdentityAnnotation = "pantheon.io/velero-record-id"
)

func copyBackupScope(scope *common.DataScopeReq) *common.DataScopeReq {
	if scope == nil {
		return nil
	}
	copy := *scope
	copy.DeptIDs = append([]uint64(nil), scope.DeptIDs...)
	copy.RoleKeys = append([]string(nil), scope.RoleKeys...)
	return &copy
}

// Only errors after CREATE was invoked can represent a lost response.
type submissionError struct{ error }

func (e submissionError) Unwrap() error { return e.error }

func submissionStatus(err error) string {
	if err == nil {
		return StatusInProgress
	}
	var submitted submissionError
	if !errors.As(err, &submitted) {
		return StatusFailed
	}
	// These API responses prove refusal. Timeouts, transport errors, conflicts
	// with an existing stable name, and 5xx responses require observation only.
	if apierrors.IsForbidden(err) || apierrors.IsUnauthorized(err) || apierrors.IsInvalid(err) || apierrors.IsBadRequest(err) || apierrors.IsMethodNotSupported(err) || apierrors.IsNotFound(err) {
		return StatusFailed
	}
	return StatusUnknown
}

func (s *BackupService) finishSubmission(model any, id uint64, err error) {
	status := submissionStatus(err)
	phase := phaseSubmitted
	if status == StatusUnknown {
		phase = phaseSubmissionUnknown
	}
	if status == StatusFailed {
		phase = StatusFailed
	}
	// Never overwrite a provider terminal state observed concurrently.
	_ = s.db.Model(model).Where("id = ? AND status IN ?", id, []string{StatusNew, StatusUnknown, StatusInProgress}).Updates(map[string]any{"status": status, "phase": phase}).Error
}

// Lock order is cluster -> reference -> source -> attempt -> sorted bindings.
// The caller holds the existing cluster lock through exactly one bounded CREATE.
func (s *BackupService) lockedVeleroClient(ctx context.Context, tx *gorm.DB, target *cluster.Cluster, scope *common.DataScopeReq) (*VeleroClient, error) {
	if target.KubeconfigCredentialRefID > 0 {
		var ref cluster.ClusterCredentialRef
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&ref, target.KubeconfigCredentialRefID).Error; err != nil {
			return nil, fmt.Errorf("k8s.cluster.credential_not_found: %w", err)
		}
		if ref.Status != "active" || ref.ClusterID != target.ID || strings.TrimSpace(ref.Encrypted) == "" {
			return nil, errors.New("k8s.cluster.credential_not_found")
		}
	}
	// Reads use the fresh authoritative helper, with reference writes serialized
	// by the same transaction. Rebuild on rotation instead of using the pre-GET client.
	return s.veleroClient(ctx, target.ID, scope)
}

func (s *BackupService) requireRestoreTargets(ctx context.Context, tx *gorm.DB, target *cluster.Cluster, req CreateRestoreRequest, lock bool) error {
	// There is no established cluster-admin action contract in this service.
	// Restrict the restore to finite explicit namespaces and no PV reconstruction.
	if req.RestorePVs {
		return errors.New("k8s.backup.restore_create_failed")
	}
	if len(req.IncludedNamespaces) == 0 {
		return errors.New("k8s.namespace.binding_required")
	}
	included := map[string]bool{}
	excluded := map[string]bool{}
	for _, name := range req.ExcludedNamespaces {
		excluded[name] = true
	}
	targets := map[string]bool{}
	for _, name := range req.IncludedNamespaces {
		if strings.TrimSpace(name) == "" || strings.Contains(name, "*") {
			return errors.New("k8s.namespace.binding_required")
		}
		included[name] = true
		if excluded[name] {
			continue
		}
		dest := name
		if mapped, ok := req.NamespaceMapping[name]; ok {
			dest = mapped
		}
		if strings.TrimSpace(dest) == "" || strings.Contains(dest, "*") {
			return errors.New("k8s.namespace.binding_required")
		}
		targets[dest] = true
	}
	for name := range req.NamespaceMapping {
		if !included[name] {
			return errors.New("k8s.namespace.binding_required")
		}
	}
	if len(targets) == 0 {
		return errors.New("k8s.namespace.binding_required")
	}
	names := make([]string, 0, len(targets))
	for name := range targets {
		names = append(names, name)
	}
	sort.Strings(names)
	ns := namespace.NewNamespaceService(nil, tx)
	for _, name := range names {
		var binding namespace.NamespaceBinding
		q := tx.WithContext(ctx).Where("cluster_id = ? AND namespace = ?", target.ID, name)
		if lock {
			q = q.Clauses(clause.Locking{Strength: "UPDATE"})
		}
		if err := q.First(&binding).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("k8s.namespace.binding_required")
			}
			return err
		}
		if binding.BusinessScopeID == 0 || binding.BusinessScopeID != target.BusinessScopeID || binding.Environment == "" || binding.Environment != target.Environment {
			return errors.New("k8s.namespace.binding_required")
		}
		if err := ns.RequireWriteWithContext(ctx, target.ID, name, "backup:restore"); err != nil {
			return err
		}
	}
	return nil
}
