package release

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"pantheon-base/modules/business/k8s/cluster"
	"pantheon-base/pkg/common"
	"pantheon-base/pkg/database"
)

const (
	releaseStatusPending   = "pending"
	releaseStatusApplying  = "applying"
	releaseStatusSucceeded = "succeeded"
	releaseStatusFailed    = "failed"
	releaseStatusTimedOut  = "timed_out"

	defaultReleaseTimeout      = 30 * time.Second
	defaultRolloutPollInterval = 500 * time.Millisecond
	maxReleaseTimeout          = 10 * time.Minute
	maxRolloutPollInterval     = 30 * time.Second
)

var (
	errReleaseNotFound          = errors.New("k8s.release.not_found")
	errReleaseWorkloadNotFound  = errors.New("k8s.release.workload_not_found")
	errReleaseApplyFailed       = errors.New("k8s.release.apply_failed")
	errReleaseRollbackFailed    = errors.New("k8s.release.rollback_failed")
	errReleaseNoRollbackTarget  = errors.New("k8s.release.no_rollback_target")
	errReleaseIdempotency       = errors.New("k8s.release.idempotency_conflict")
	errReleaseInvalidWorkload   = errors.New("k8s.release.invalid_workload_type")
	errReleaseContainerNotFound = errors.New("k8s.release.container_not_found")
	errReleaseRolloutFailed     = errors.New("k8s.release.rollout_failed")
	errReleaseRolloutTimedOut   = errors.New("k8s.release.rollout_timed_out")
	errReleaseApplyAmbiguous    = errors.New("k8s.release.apply_ambiguous")
	errReleaseReconcilePending  = errors.New("k8s.release.reconciliation_pending")
)

type releaseClientProvider func(uint64, *common.DataScopeReq) (kubernetes.Interface, error)

type releaseRequestSnapshot struct {
	Operation           string `json:"operation"`
	Name                string `json:"name"`
	ClusterID           uint64 `json:"clusterId"`
	ServiceID           uint64 `json:"serviceId,omitempty"`
	ServiceInstanceID   uint64 `json:"serviceInstanceId,omitempty"`
	ServiceName         string `json:"serviceName,omitempty"`
	ServiceInstanceName string `json:"serviceInstanceName,omitempty"`
	Namespace           string `json:"namespace"`
	WorkloadType        string `json:"workloadType"`
	WorkloadName        string `json:"workloadName"`
	ContainerName       string `json:"containerName"`
	Image               string `json:"image"`
	Strategy            string `json:"strategy"`
	PreviousReleaseID   uint64 `json:"previousReleaseId,omitempty"`
}

type workloadObservation struct {
	Image             string
	Generation        int64
	Revision          string
	DesiredReplicas   int32
	UpdatedReplicas   int32
	AvailableReplicas int32
	ReadyReplicas     int32
	ConditionSummary  string
	RolloutSucceeded  bool
	RolloutFailed     bool
}

type ReleaseService struct {
	db               *gorm.DB
	clusterSvc       *cluster.ClusterService
	clientProvider   releaseClientProvider
	operationTimeout time.Duration
	pollInterval     time.Duration
}

func NewReleaseService(db *gorm.DB, clusterSvc *cluster.ClusterService) *ReleaseService {
	service := &ReleaseService{
		db:               db,
		clusterSvc:       clusterSvc,
		operationTimeout: releaseDurationFromEnv("PANTHEON_K8S_RELEASE_TIMEOUT", defaultReleaseTimeout, maxReleaseTimeout),
		pollInterval:     releaseDurationFromEnv("PANTHEON_K8S_RELEASE_POLL_INTERVAL", defaultRolloutPollInterval, maxRolloutPollInterval),
	}
	if clusterSvc != nil {
		service.clientProvider = func(clusterID uint64, dataScope *common.DataScopeReq) (kubernetes.Interface, error) {
			return clusterSvc.GetClientset(clusterID, dataScope)
		}
	}
	return service
}

// SetClientProvider is intentionally narrow: production wiring uses the
// Cluster-owned client factory, while fake/disposable clients can drive focused
// release verification without reaching an external cluster.
func (s *ReleaseService) SetClientProvider(provider func(uint64, *common.DataScopeReq) (kubernetes.Interface, error)) {
	s.clientProvider = provider
}

// SetRolloutPolicy configures bounded observation. A non-positive value keeps
// the production default, so callers cannot accidentally create an unbounded
// release wait.
func (s *ReleaseService) SetRolloutPolicy(timeout, pollInterval time.Duration) {
	if timeout > 0 && timeout <= maxReleaseTimeout {
		s.operationTimeout = timeout
	}
	if pollInterval > 0 && pollInterval <= maxRolloutPollInterval {
		s.pollInterval = pollInterval
	}
}

func releaseDurationFromEnv(name string, fallback, maximum time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	parsed, err := time.ParseDuration(value)
	if err != nil || parsed <= 0 || parsed > maximum {
		return fallback
	}
	return parsed
}

func (s *ReleaseService) Migrate() error {
	if s.db == nil {
		return errors.New("database.not_initialized")
	}
	return s.db.AutoMigrate(&Release{})
}

// Create first persists immutable release intent under the target Cluster row
// lock. Kubernetes mutation cannot start when the intent write fails.
func (s *ReleaseService) Create(req CreateReleaseRequest, createdBy string, dataScope *common.DataScopeReq) (*ReleaseResponse, error) {
	intent, err := normalizeCreateRequest(req)
	if err != nil {
		return nil, err
	}
	record, replay, err := s.persistIntentWithKey(intent, req.IdempotencyKey, createdBy, dataScope)
	if err != nil {
		return nil, err
	}
	if replay {
		resp := toResponse(record)
		return &resp, nil
	}

	if err := s.execute(record.ID, dataScope); err != nil {
		return nil, err
	}
	return s.responseByID(record.ID, dataScope)
}

func (s *ReleaseService) List(query ReleaseListQuery, dataScope *common.DataScopeReq) (*ReleaseListResponse, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}
	if query.Page <= 0 {
		query.Page = 1
	}
	if query.PageSize <= 0 || query.PageSize > 100 {
		query.PageSize = 10
	}

	db := s.releaseQuery(dataScope)
	if query.ClusterID > 0 {
		db = db.Where("cluster_id = ?", query.ClusterID)
	}
	if query.Namespace != "" {
		db = db.Where("namespace = ?", query.Namespace)
	}
	if query.WorkloadType != "" {
		db = db.Where("workload_type = ?", normalizeWorkloadType(query.WorkloadType))
	}
	if query.WorkloadName != "" {
		db = db.Where("workload_name = ?", query.WorkloadName)
	}
	if query.Status != "" {
		db = db.Where("status = ?", query.Status)
	}

	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}

	var releases []Release
	offset := (query.Page - 1) * query.PageSize
	if err := db.Order("id DESC").Offset(offset).Limit(query.PageSize).Find(&releases).Error; err != nil {
		return nil, err
	}

	items := make([]ReleaseResponse, len(releases))
	for i := range releases {
		items[i] = toResponse(&releases[i])
	}
	return &ReleaseListResponse{Items: items, Total: total, Page: query.Page, PageSize: query.PageSize}, nil
}

// Rollback creates another immutable release intent. It never changes the
// target release row and observes rollout using exactly the same policy as a
// forward image update.
func (s *ReleaseService) Rollback(id uint64, req RollbackReleaseRequest, createdBy string, dataScope *common.DataScopeReq) (*ReleaseResponse, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}
	var target Release
	if err := s.releaseQuery(dataScope).First(&target, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errReleaseNotFound
		}
		return nil, err
	}
	if target.Status != releaseStatusSucceeded || target.ImageBefore == "" {
		return nil, errReleaseNoRollbackTarget
	}

	intent := releaseRequestSnapshot{
		Operation:           "rollback",
		Name:                target.Name + "-rollback",
		ClusterID:           target.ClusterID,
		Namespace:           target.Namespace,
		WorkloadType:        target.WorkloadType,
		WorkloadName:        target.WorkloadName,
		ContainerName:       target.ContainerName,
		Image:               target.ImageBefore,
		Strategy:            target.Strategy,
		ServiceID:           target.ServiceID,
		ServiceInstanceID:   target.ServiceInstanceID,
		ServiceName:         target.ServiceName,
		ServiceInstanceName: target.ServiceInstanceName,
		PreviousReleaseID:   target.ID,
	}
	record, replay, err := s.persistIntentWithKey(intent, req.IdempotencyKey, createdBy, dataScope)
	if err != nil {
		return nil, err
	}
	if replay {
		resp := toResponse(record)
		return &resp, nil
	}

	if err := s.execute(record.ID, dataScope); err != nil {
		if errors.Is(err, errReleaseApplyFailed) {
			return nil, errReleaseRollbackFailed
		}
		return nil, err
	}
	return s.responseByID(record.ID, dataScope)
}

// Reconcile observes an existing intent without issuing another Kubernetes
// mutation. It is the repair path for an apply/result-write ambiguity.
func (s *ReleaseService) Reconcile(id uint64, dataScope *common.DataScopeReq) (*ReleaseResponse, error) {
	if s.db == nil {
		return nil, errors.New("database.not_initialized")
	}
	var record Release
	if err := s.releaseQuery(dataScope).First(&record, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errReleaseNotFound
		}
		return nil, err
	}
	if isTerminalReleaseStatus(record.Status) {
		resp := toResponse(&record)
		return &resp, nil
	}
	if record.Status == releaseStatusPending {
		now := time.Now()
		if err := s.db.Model(&Release{}).Where("id = ? AND status = ?", record.ID, releaseStatusPending).
			Updates(map[string]interface{}{"last_reconciled_at": &now, "updated_at": now}).Error; err != nil {
			return nil, err
		}
		return s.responseByID(record.ID, dataScope)
	}

	clientset, err := s.clientFor(record.ClusterID, dataScope)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), s.operationTimeout)
	defer cancel()

	observed, err := observeWorkload(ctx, clientset, record.WorkloadType, record.Namespace, record.WorkloadName, record.ContainerName)
	if err != nil {
		if persistErr := s.persistObservation(record.ID, observed, true, err.Error()); persistErr != nil {
			return nil, persistErr
		}
		return nil, err
	}
	if err := s.persistObservation(record.ID, observed, true, ""); err != nil {
		return nil, err
	}
	// A reconciler must not close an applying record before the executor has
	// durably recorded the generation accepted by Kubernetes. Otherwise a
	// stale failure condition from the preceding rollout could win the race
	// before the image update is issued.
	if record.TargetGeneration <= 0 {
		resp, responseErr := s.responseByID(record.ID, dataScope)
		if responseErr != nil {
			return nil, responseErr
		}
		return resp, errReleaseReconcilePending
	}

	switch {
	case rolloutSucceeded(record, observed):
		if err := s.finish(record.ID, releaseStatusSucceeded, observed, ""); err != nil {
			return nil, err
		}
	case observed.RolloutFailed:
		if err := s.finish(record.ID, releaseStatusFailed, observed, observed.ConditionSummary); err != nil {
			return nil, err
		}
	}

	resp, err := s.responseByID(record.ID, dataScope)
	if err != nil {
		return nil, err
	}
	if resp.Status == releaseStatusApplying || resp.Status == releaseStatusTimedOut {
		return resp, errReleaseReconcilePending
	}
	return resp, nil
}

// HasReferences is the K8s-owned ReferenceChecker capability consumed by the
// Cluster service during deletion. History remains tied to the Cluster identity.
func (s *ReleaseService) HasReferences(_ context.Context, tx *gorm.DB, clusterID uint64) (bool, error) {
	if tx == nil {
		return false, errors.New("database.not_initialized")
	}
	var count int64
	if err := tx.Model(&Release{}).Where("cluster_id = ?", clusterID).Count(&count).Error; err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *ReleaseService) execute(id uint64, dataScope *common.DataScopeReq) error {
	var record Release
	if err := s.releaseQuery(dataScope).First(&record, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errReleaseNotFound
		}
		return err
	}
	if record.Status != releaseStatusPending {
		return nil
	}

	now := time.Now()
	result := s.db.Model(&Release{}).Where("id = ? AND status = ?", id, releaseStatusPending).
		Updates(map[string]interface{}{"status": releaseStatusApplying, "started_at": &now, "updated_at": now})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return nil
	}
	record.Status = releaseStatusApplying
	record.StartedAt = &now

	clientset, err := s.clientFor(record.ClusterID, dataScope)
	if err != nil {
		return s.fail(record.ID, workloadObservation{}, err.Error(), err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), s.operationTimeout)
	defer cancel()

	before, err := observeWorkload(ctx, clientset, record.WorkloadType, record.Namespace, record.WorkloadName, record.ContainerName)
	if err != nil {
		return s.fail(record.ID, before, err.Error(), errReleaseWorkloadNotFound)
	}
	if err := s.persistBeforeApply(record.ID, before); err != nil {
		return err
	}

	target, err := applyImage(ctx, clientset, record.WorkloadType, record.Namespace, record.WorkloadName, record.ContainerName, record.ImageAfter)
	if err != nil {
		if isAmbiguousApplyError(err) {
			if persistErr := s.persistObservation(record.ID, before, false, err.Error()); persistErr != nil {
				return persistErr
			}
			_, reconcileErr := s.Reconcile(record.ID, dataScope)
			if reconcileErr == nil {
				return nil
			}
			if errors.Is(reconcileErr, errReleaseReconcilePending) {
				return errReleaseApplyAmbiguous
			}
			return reconcileErr
		}
		return s.fail(record.ID, before, err.Error(), errReleaseApplyFailed)
	}
	if err := s.persistTarget(record.ID, target); err != nil {
		// Kubernetes has already accepted the change. Leave the intent applying
		// so Reconcile can repair the durable observation later.
		return err
	}

	observed, rolloutErr := s.waitForRollout(ctx, clientset, record, target)
	switch {
	case rolloutErr == nil:
		return s.finish(record.ID, releaseStatusSucceeded, observed, "")
	case errors.Is(rolloutErr, errReleaseRolloutTimedOut):
		if err := s.finish(record.ID, releaseStatusTimedOut, observed, observed.ConditionSummary); err != nil {
			return err
		}
		return errReleaseRolloutTimedOut
	default:
		if err := s.finish(record.ID, releaseStatusFailed, observed, errorMessage(rolloutErr, observed.ConditionSummary)); err != nil {
			return err
		}
		return rolloutErr
	}
}

func (s *ReleaseService) waitForRollout(ctx context.Context, clientset kubernetes.Interface, record Release, target workloadObservation) (workloadObservation, error) {
	ticker := time.NewTicker(s.pollInterval)
	defer ticker.Stop()

	var last workloadObservation
	for {
		observed, err := observeWorkload(ctx, clientset, record.WorkloadType, record.Namespace, record.WorkloadName, record.ContainerName)
		if err != nil {
			return last, err
		}
		last = observed
		if rolloutSucceeded(recordWithTarget(record, target), observed) {
			return observed, nil
		}
		if observed.RolloutFailed {
			return observed, errReleaseRolloutFailed
		}

		select {
		case <-ctx.Done():
			return last, errReleaseRolloutTimedOut
		case <-ticker.C:
		}
	}
}

func (s *ReleaseService) persistIntentWithKey(snapshot releaseRequestSnapshot, providedKey, createdBy string, dataScope *common.DataScopeReq) (*Release, bool, error) {
	if s.db == nil || s.clusterSvc == nil {
		return nil, false, errors.New("database.not_initialized")
	}
	key := strings.TrimSpace(providedKey)
	if key == "" {
		key = uuid.NewString()
	}
	fingerprint, encodedSnapshot, err := fingerprintSnapshot(snapshot)
	if err != nil {
		return nil, false, err
	}

	var record Release
	var replay bool
	err = s.clusterSvc.WithClusterLock(snapshot.ClusterID, dataScope, func(tx *gorm.DB, item *cluster.Cluster) error {
		var existing Release
		err := tx.Where("idempotency_key = ?", key).First(&existing).Error
		switch {
		case err == nil:
			if existing.RequestFingerprint != fingerprint {
				return errReleaseIdempotency
			}
			record = existing
			replay = true
			return nil
		case !errors.Is(err, gorm.ErrRecordNotFound):
			return err
		}

		record = Release{
			Name:                snapshot.Name,
			ClusterID:           snapshot.ClusterID,
			Namespace:           snapshot.Namespace,
			BusinessScopeID:     item.BusinessScopeID,
			DeptID:              item.DeptID,
			ServiceID:           snapshot.ServiceID,
			ServiceInstanceID:   snapshot.ServiceInstanceID,
			ServiceName:         snapshot.ServiceName,
			ServiceInstanceName: snapshot.ServiceInstanceName,
			WorkloadType:        snapshot.WorkloadType,
			WorkloadName:        snapshot.WorkloadName,
			ContainerName:       snapshot.ContainerName,
			ImageAfter:          snapshot.Image,
			Strategy:            snapshot.Strategy,
			Status:              releaseStatusPending,
			IdempotencyKey:      key,
			RequestFingerprint:  fingerprint,
			RequestSnapshot:     datatypes.JSON(encodedSnapshot),
			PreviousReleaseID:   snapshot.PreviousReleaseID,
			Attempt:             1,
			CreatedBy:           createdBy,
		}
		return tx.Create(&record).Error
	})
	if err != nil {
		return nil, false, err
	}
	return &record, replay, nil
}

func (s *ReleaseService) releaseQuery(dataScope *common.DataScopeReq) *gorm.DB {
	return s.db.Model(&Release{}).Scopes(database.WithDataScope(dataScope))
}

func (s *ReleaseService) clientFor(clusterID uint64, dataScope *common.DataScopeReq) (kubernetes.Interface, error) {
	if s.clientProvider == nil {
		return nil, errors.New("k8s.release.client_not_configured")
	}
	return s.clientProvider(clusterID, dataScope)
}

func (s *ReleaseService) persistBeforeApply(id uint64, observed workloadObservation) error {
	now := time.Now()
	result := s.db.Model(&Release{}).Where("id = ? AND status = ?", id, releaseStatusApplying).Updates(map[string]interface{}{
		"image_before":        observed.Image,
		"observed_generation": observed.Generation,
		"observed_revision":   observed.Revision,
		"desired_replicas":    observed.DesiredReplicas,
		"updated_replicas":    observed.UpdatedReplicas,
		"available_replicas":  observed.AvailableReplicas,
		"ready_replicas":      observed.ReadyReplicas,
		"condition_summary":   observed.ConditionSummary,
		"updated_at":          now,
	})
	return requireReleaseTransition(result)
}

func (s *ReleaseService) persistTarget(id uint64, target workloadObservation) error {
	now := time.Now()
	result := s.db.Model(&Release{}).Where("id = ? AND status = ?", id, releaseStatusApplying).Updates(map[string]interface{}{
		"target_generation": target.Generation,
		"target_revision":   target.Revision,
		"updated_at":        now,
	})
	return requireReleaseTransition(result)
}

func (s *ReleaseService) persistObservation(id uint64, observed workloadObservation, reconciled bool, message string) error {
	now := time.Now()
	updates := observationUpdates(observed, now)
	if reconciled {
		updates["last_reconciled_at"] = &now
	}
	if message != "" {
		updates["error_message"] = message
	}
	result := s.db.Model(&Release{}).
		Where("id = ? AND status IN ?", id, []string{releaseStatusApplying, releaseStatusTimedOut}).
		Updates(updates)
	return requireReleaseTransition(result)
}

func (s *ReleaseService) finish(id uint64, status string, observed workloadObservation, message string) error {
	now := time.Now()
	updates := observationUpdates(observed, now)
	updates["status"] = status
	updates["finished_at"] = &now
	updates["error_message"] = message
	result := s.db.Model(&Release{}).
		Where("id = ? AND status IN ?", id, []string{releaseStatusApplying, releaseStatusTimedOut}).
		Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 1 {
		return nil
	}
	var current Release
	if err := s.db.Select("status").First(&current, id).Error; err != nil {
		return err
	}
	if current.Status != status {
		return errors.New("k8s.release.transition_conflict")
	}
	return nil
}

func requireReleaseTransition(result *gorm.DB) error {
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return errors.New("k8s.release.transition_conflict")
	}
	return nil
}

func (s *ReleaseService) fail(id uint64, observed workloadObservation, message string, apiErr error) error {
	if err := s.finish(id, releaseStatusFailed, observed, message); err != nil {
		return err
	}
	return apiErr
}

func (s *ReleaseService) responseByID(id uint64, dataScope *common.DataScopeReq) (*ReleaseResponse, error) {
	var record Release
	if err := s.releaseQuery(dataScope).First(&record, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errReleaseNotFound
		}
		return nil, err
	}
	resp := toResponse(&record)
	return &resp, nil
}

func toResponse(record *Release) ReleaseResponse {
	startedAt := ""
	if record.StartedAt != nil {
		startedAt = record.StartedAt.Format(time.RFC3339)
	}
	finishedAt := ""
	if record.FinishedAt != nil {
		finishedAt = record.FinishedAt.Format(time.RFC3339)
	}
	lastReconciledAt := ""
	if record.LastReconciledAt != nil {
		lastReconciledAt = record.LastReconciledAt.Format(time.RFC3339)
	}
	return ReleaseResponse{
		ID:                  record.ID,
		Name:                record.Name,
		ClusterID:           record.ClusterID,
		ServiceID:           record.ServiceID,
		ServiceInstanceID:   record.ServiceInstanceID,
		ServiceName:         record.ServiceName,
		ServiceInstanceName: record.ServiceInstanceName,
		Namespace:           record.Namespace,
		WorkloadType:        record.WorkloadType,
		WorkloadName:        record.WorkloadName,
		ContainerName:       record.ContainerName,
		ImageBefore:         record.ImageBefore,
		ImageAfter:          record.ImageAfter,
		Strategy:            record.Strategy,
		Status:              record.Status,
		IdempotencyKey:      record.IdempotencyKey,
		PreviousReleaseID:   record.PreviousReleaseID,
		Attempt:             record.Attempt,
		TargetGeneration:    record.TargetGeneration,
		TargetRevision:      record.TargetRevision,
		ObservedGeneration:  record.ObservedGeneration,
		ObservedRevision:    record.ObservedRevision,
		DesiredReplicas:     record.DesiredReplicas,
		UpdatedReplicas:     record.UpdatedReplicas,
		AvailableReplicas:   record.AvailableReplicas,
		ReadyReplicas:       record.ReadyReplicas,
		ConditionSummary:    record.ConditionSummary,
		ErrorMessage:        record.ErrorMessage,
		StartedAt:           startedAt,
		FinishedAt:          finishedAt,
		LastReconciledAt:    lastReconciledAt,
		CreatedBy:           record.CreatedBy,
		CreatedAt:           record.CreatedAt.Format(time.RFC3339),
	}
}

func normalizeCreateRequest(req CreateReleaseRequest) (releaseRequestSnapshot, error) {
	workloadType := normalizeWorkloadType(req.WorkloadType)
	if workloadType == "" {
		return releaseRequestSnapshot{}, errReleaseInvalidWorkload
	}
	snapshot := releaseRequestSnapshot{
		Operation:           "create",
		Name:                strings.TrimSpace(req.Name),
		ClusterID:           req.ClusterID,
		ServiceID:           req.ServiceID,
		ServiceInstanceID:   req.ServiceInstanceID,
		ServiceName:         strings.TrimSpace(req.ServiceName),
		ServiceInstanceName: strings.TrimSpace(req.ServiceInstanceName),
		Namespace:           strings.TrimSpace(req.Namespace),
		WorkloadType:        workloadType,
		WorkloadName:        strings.TrimSpace(req.WorkloadName),
		ContainerName:       strings.TrimSpace(req.ContainerName),
		Image:               strings.TrimSpace(req.Image),
		Strategy:            strings.TrimSpace(req.Strategy),
	}
	if snapshot.Name == "" || snapshot.ClusterID == 0 || snapshot.Namespace == "" || snapshot.WorkloadName == "" || snapshot.Image == "" {
		return releaseRequestSnapshot{}, errors.New("common.param_invalid")
	}
	if snapshot.Strategy == "" {
		snapshot.Strategy = "RollingUpdate"
	}
	return snapshot, nil
}

func normalizeWorkloadType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "deployment":
		return "deployment"
	case "statefulset":
		return "statefulset"
	case "daemonset":
		return "daemonset"
	default:
		return ""
	}
}

func fingerprintSnapshot(snapshot releaseRequestSnapshot) (string, []byte, error) {
	encoded, err := json.Marshal(snapshot)
	if err != nil {
		return "", nil, err
	}
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:]), encoded, nil
}

func recordWithTarget(record Release, target workloadObservation) Release {
	record.TargetGeneration = target.Generation
	record.TargetRevision = target.Revision
	return record
}

func observationUpdates(observed workloadObservation, now time.Time) map[string]interface{} {
	return map[string]interface{}{
		"observed_generation": observed.Generation,
		"observed_revision":   observed.Revision,
		"desired_replicas":    observed.DesiredReplicas,
		"updated_replicas":    observed.UpdatedReplicas,
		"available_replicas":  observed.AvailableReplicas,
		"ready_replicas":      observed.ReadyReplicas,
		"condition_summary":   observed.ConditionSummary,
		"updated_at":          now,
	}
}

func rolloutSucceeded(record Release, observed workloadObservation) bool {
	if observed.Image != record.ImageAfter {
		return false
	}
	if record.TargetGeneration > 0 && observed.Generation < record.TargetGeneration {
		return false
	}
	if record.TargetRevision != "" && observed.Revision != "" && observed.Revision != record.TargetRevision {
		return false
	}
	return observed.RolloutSucceeded
}

func isTerminalReleaseStatus(status string) bool {
	return status == releaseStatusSucceeded || status == releaseStatusFailed
}

func isAmbiguousApplyError(err error) bool {
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return true
	}
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}

func errorMessage(err error, fallback string) string {
	if fallback != "" {
		return fallback
	}
	if err == nil {
		return ""
	}
	return err.Error()
}

func observeWorkload(ctx context.Context, clientset kubernetes.Interface, workloadType, namespace, name, containerName string) (workloadObservation, error) {
	switch normalizeWorkloadType(workloadType) {
	case "deployment":
		item, err := clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return workloadObservation{}, err
		}
		return observeDeployment(item, containerName)
	case "statefulset":
		item, err := clientset.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return workloadObservation{}, err
		}
		return observeStatefulSet(item, containerName)
	case "daemonset":
		item, err := clientset.AppsV1().DaemonSets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return workloadObservation{}, err
		}
		return observeDaemonSet(item, containerName)
	default:
		return workloadObservation{}, errReleaseInvalidWorkload
	}
}

func applyImage(ctx context.Context, clientset kubernetes.Interface, workloadType, namespace, name, containerName, image string) (workloadObservation, error) {
	switch normalizeWorkloadType(workloadType) {
	case "deployment":
		item, err := clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return workloadObservation{}, err
		}
		if err := setContainerImageStrict(item.Spec.Template.Spec.Containers, containerName, image); err != nil {
			return workloadObservation{}, err
		}
		updated, err := clientset.AppsV1().Deployments(namespace).Update(ctx, item, metav1.UpdateOptions{})
		if err != nil {
			return workloadObservation{}, err
		}
		target, err := observeDeployment(updated, containerName)
		target.Generation = updated.Generation
		target.Revision = ""
		return target, err
	case "statefulset":
		item, err := clientset.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return workloadObservation{}, err
		}
		if err := setContainerImageStrict(item.Spec.Template.Spec.Containers, containerName, image); err != nil {
			return workloadObservation{}, err
		}
		updated, err := clientset.AppsV1().StatefulSets(namespace).Update(ctx, item, metav1.UpdateOptions{})
		if err != nil {
			return workloadObservation{}, err
		}
		target, err := observeStatefulSet(updated, containerName)
		target.Generation = updated.Generation
		target.Revision = ""
		return target, err
	case "daemonset":
		item, err := clientset.AppsV1().DaemonSets(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return workloadObservation{}, err
		}
		if err := setContainerImageStrict(item.Spec.Template.Spec.Containers, containerName, image); err != nil {
			return workloadObservation{}, err
		}
		updated, err := clientset.AppsV1().DaemonSets(namespace).Update(ctx, item, metav1.UpdateOptions{})
		if err != nil {
			return workloadObservation{}, err
		}
		target, err := observeDaemonSet(updated, containerName)
		target.Generation = updated.Generation
		target.Revision = ""
		return target, err
	default:
		return workloadObservation{}, errReleaseInvalidWorkload
	}
}

func observeDeployment(item *appsv1.Deployment, containerName string) (workloadObservation, error) {
	image, err := containerImage(item.Spec.Template.Spec.Containers, containerName)
	if err != nil {
		return workloadObservation{}, err
	}
	desired := replicas(item.Spec.Replicas)
	available := deploymentCondition(item.Status.Conditions, appsv1.DeploymentAvailable, corev1.ConditionTrue)
	failed := deploymentCondition(item.Status.Conditions, appsv1.DeploymentProgressing, corev1.ConditionFalse) ||
		deploymentCondition(item.Status.Conditions, appsv1.DeploymentReplicaFailure, corev1.ConditionTrue)
	return workloadObservation{
		Image:             image,
		Generation:        item.Status.ObservedGeneration,
		Revision:          item.Annotations["deployment.kubernetes.io/revision"],
		DesiredReplicas:   desired,
		UpdatedReplicas:   item.Status.UpdatedReplicas,
		AvailableReplicas: item.Status.AvailableReplicas,
		ReadyReplicas:     item.Status.ReadyReplicas,
		ConditionSummary:  summarizeDeploymentConditions(item.Status.Conditions),
		RolloutSucceeded:  item.Status.ObservedGeneration >= item.Generation && item.Status.UpdatedReplicas >= desired && item.Status.AvailableReplicas >= desired && (desired == 0 || available),
		RolloutFailed:     failed,
	}, nil
}

func observeStatefulSet(item *appsv1.StatefulSet, containerName string) (workloadObservation, error) {
	image, err := containerImage(item.Spec.Template.Spec.Containers, containerName)
	if err != nil {
		return workloadObservation{}, err
	}
	desired := replicas(item.Spec.Replicas)
	failed := statefulSetFailed(item.Status.Conditions)
	return workloadObservation{
		Image:             image,
		Generation:        item.Status.ObservedGeneration,
		Revision:          item.Status.UpdateRevision,
		DesiredReplicas:   desired,
		UpdatedReplicas:   item.Status.UpdatedReplicas,
		AvailableReplicas: item.Status.AvailableReplicas,
		ReadyReplicas:     item.Status.ReadyReplicas,
		ConditionSummary:  summarizeStatefulSetConditions(item.Status.Conditions),
		RolloutSucceeded:  item.Status.ObservedGeneration >= item.Generation && item.Status.UpdatedReplicas >= desired && item.Status.ReadyReplicas >= desired && (item.Status.UpdateRevision == "" || item.Status.CurrentRevision == item.Status.UpdateRevision || item.Status.UpdatedReplicas >= desired),
		RolloutFailed:     failed,
	}, nil
}

func observeDaemonSet(item *appsv1.DaemonSet, containerName string) (workloadObservation, error) {
	image, err := containerImage(item.Spec.Template.Spec.Containers, containerName)
	if err != nil {
		return workloadObservation{}, err
	}
	desired := item.Status.DesiredNumberScheduled
	return workloadObservation{
		Image:             image,
		Generation:        item.Status.ObservedGeneration,
		Revision:          item.Annotations["daemonset.kubernetes.io/revision"],
		DesiredReplicas:   desired,
		UpdatedReplicas:   item.Status.UpdatedNumberScheduled,
		AvailableReplicas: item.Status.NumberAvailable,
		ReadyReplicas:     item.Status.NumberReady,
		ConditionSummary:  summarizeDaemonSetConditions(item.Status.Conditions),
		RolloutSucceeded:  item.Status.ObservedGeneration >= item.Generation && item.Status.UpdatedNumberScheduled >= desired && item.Status.NumberAvailable >= desired && item.Status.NumberReady >= desired,
		RolloutFailed:     daemonSetFailed(item.Status.Conditions),
	}, nil
}

func containerImage(containers []corev1.Container, containerName string) (string, error) {
	if len(containers) == 0 {
		return "", errReleaseContainerNotFound
	}
	if containerName == "" {
		return containers[0].Image, nil
	}
	for _, container := range containers {
		if container.Name == containerName {
			return container.Image, nil
		}
	}
	return "", errReleaseContainerNotFound
}

func setContainerImageStrict(containers []corev1.Container, containerName, image string) error {
	if len(containers) == 0 {
		return errReleaseContainerNotFound
	}
	if containerName == "" {
		containers[0].Image = image
		return nil
	}
	for i := range containers {
		if containers[i].Name == containerName {
			containers[i].Image = image
			return nil
		}
	}
	return errReleaseContainerNotFound
}

func replicas(value *int32) int32 {
	if value == nil {
		return 0
	}
	return *value
}

func deploymentCondition(conditions []appsv1.DeploymentCondition, conditionType appsv1.DeploymentConditionType, status corev1.ConditionStatus) bool {
	for _, condition := range conditions {
		if condition.Type == conditionType && condition.Status == status {
			return true
		}
	}
	return false
}

func summarizeDeploymentConditions(conditions []appsv1.DeploymentCondition) string {
	parts := make([]string, 0, len(conditions))
	for _, condition := range conditions {
		parts = append(parts, fmt.Sprintf("%s=%s:%s", condition.Type, condition.Status, condition.Reason))
	}
	return strings.Join(parts, "; ")
}

func summarizeStatefulSetConditions(conditions []appsv1.StatefulSetCondition) string {
	parts := make([]string, 0, len(conditions))
	for _, condition := range conditions {
		parts = append(parts, fmt.Sprintf("%s=%s:%s", condition.Type, condition.Status, condition.Reason))
	}
	return strings.Join(parts, "; ")
}

func summarizeDaemonSetConditions(conditions []appsv1.DaemonSetCondition) string {
	parts := make([]string, 0, len(conditions))
	for _, condition := range conditions {
		parts = append(parts, fmt.Sprintf("%s=%s:%s", condition.Type, condition.Status, condition.Reason))
	}
	return strings.Join(parts, "; ")
}

func statefulSetFailed(conditions []appsv1.StatefulSetCondition) bool {
	for _, condition := range conditions {
		if condition.Status == corev1.ConditionTrue && strings.Contains(strings.ToLower(condition.Reason), "fail") {
			return true
		}
	}
	return false
}

func daemonSetFailed(conditions []appsv1.DaemonSetCondition) bool {
	for _, condition := range conditions {
		if condition.Status == corev1.ConditionTrue && strings.Contains(strings.ToLower(condition.Reason), "fail") {
			return true
		}
	}
	return false
}
