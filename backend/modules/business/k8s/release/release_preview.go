package release

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/modules/business/k8s/cluster"
	k8spkg "github.com/duanxldragon/pantheon-base/backend/modules/business/k8s/pkg"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/database"
	"github.com/google/uuid"
	"gorm.io/gorm"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation"
	"k8s.io/client-go/kubernetes"
)

const msgPreviewConflict = "k8s.release.preview_conflict"

// Guard is private persisted metadata, never accepted from the HTTP request.
type releaseReviewGuard struct {
	UID             string `json:"uid"`
	ResourceVersion string `json:"resourceVersion"`
	ClusterVersion  string `json:"clusterVersion"`
}

type releaseReviewTicket struct {
	Audience    string             `json:"audience"`
	Actor       string             `json:"actor"`
	IntentID    string             `json:"intentId"`
	Fingerprint string             `json:"fingerprint"`
	Guard       releaseReviewGuard `json:"guard"`
	Expires     int64              `json:"expires"`
}

// PreviewChange contains the image difference and safe unchanged workload context.
type PreviewChange struct {
	ID     string `json:"id"`
	Field  string `json:"field"`
	Kind   string `json:"kind"`
	Before string `json:"before"`
	After  string `json:"after"`
}

// ReleasePreviewResponse intentionally excludes manifests/configuration and credentials.
type ReleasePreviewResponse struct {
	IntentID          string          `json:"intentId"`
	IntentVersion     string          `json:"intentVersion"`
	CurrentVersion    string          `json:"currentVersion"`
	TargetVersion     string          `json:"targetVersion"`
	TargetCount       int             `json:"targetCount"`
	Environment       string          `json:"environment"`
	Namespace         string          `json:"namespace"`
	WorkloadType      string          `json:"workloadType"`
	WorkloadName      string          `json:"workloadName"`
	ContainerName     string          `json:"containerName"`
	Changes           []PreviewChange `json:"changes"`
	RiskKeys          []string        `json:"riskKeys"`
	RollbackAvailable bool            `json:"rollbackAvailable"`
	ExpiresAt         time.Time       `json:"expiresAt"`
}

func reviewConflict() error { return common.NewConflict(msgPreviewConflict) }

func releaseClusterVersion(db *gorm.DB, item *cluster.Cluster) (string, error) {
	var credential cluster.ClusterCredentialRef
	if item.KubeconfigCredentialRefID > 0 {
		if err := db.First(&credential, item.KubeconfigCredentialRefID).Error; err != nil {
			return "", err
		}
		if credential.ClusterID != item.ID || credential.Status != "active" {
			return "", reviewConflict()
		}
	}
	// Hash secrets privately. Neither the raw material nor its individual hashes are projected.
	data, err := json.Marshal(struct {
		ID, Scope, Dept, Ref, Version          uint64
		Environment, Server, Encrypted, Status string
	}{
		item.ID, item.BusinessScopeID, item.DeptID, item.KubeconfigCredentialRefID, credential.Version, item.Environment, item.APIServer, item.KubeconfigEncrypted + credential.Encrypted, credential.Status})
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func (s *ReleaseService) visibleReviewCluster(ctx context.Context, id uint64, scope *common.DataScopeReq) (*cluster.Cluster, error) {
	if s.db == nil {
		return nil, common.ErrDatabaseNotInitialized
	}
	var item cluster.Cluster
	err := s.db.WithContext(ctx).Scopes(database.WithDataScope(scope)).First(&item, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, common.NewNotFound("k8s.cluster.not_found")
	}
	return &item, err
}

// Public image fields reject URL/userinfo and control characters rather than exposing secrets.
func safeReviewImage(value string) bool {
	if value == "" || len(value) > 512 || strings.Contains(value, "://") || strings.Contains(value, "//") {
		return false
	}
	for _, r := range value {
		if !(r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r >= '0' && r <= '9' || strings.ContainsRune("._/-:@", r)) {
			return false
		}
	}
	if index := strings.IndexByte(value, '@'); index >= 0 {
		digest := value[index+1:]
		if !strings.HasPrefix(digest, "sha256:") || len(digest) != 71 {
			return false
		}
		if _, err := hex.DecodeString(digest[7:]); err != nil {
			return false
		}
	}
	return true
}

func reviewWorkload(ctx context.Context, client kubernetes.Interface, intent releaseRequestSnapshot) (metav1.Object, string, error) {
	switch intent.WorkloadType {
	case releaseWorkloadDeployment:
		item, err := client.AppsV1().Deployments(intent.Namespace).Get(ctx, intent.WorkloadName, metav1.GetOptions{})
		if err != nil {
			return nil, "", err
		}
		image, err := containerImage(item.Spec.Template.Spec.Containers, intent.ContainerName)
		return item, image, err
	case releaseWorkloadStatefulSet:
		item, err := client.AppsV1().StatefulSets(intent.Namespace).Get(ctx, intent.WorkloadName, metav1.GetOptions{})
		if err != nil {
			return nil, "", err
		}
		image, err := containerImage(item.Spec.Template.Spec.Containers, intent.ContainerName)
		return item, image, err
	case releaseWorkloadDaemonSet:
		item, err := client.AppsV1().DaemonSets(intent.Namespace).Get(ctx, intent.WorkloadName, metav1.GetOptions{})
		if err != nil {
			return nil, "", err
		}
		image, err := containerImage(item.Spec.Template.Spec.Containers, intent.ContainerName)
		return item, image, err
	default:
		return nil, "", errReleaseInvalidWorkload
	}
}

func (s *ReleaseService) previewIntent(ctx context.Context, intent releaseRequestSnapshot, key, actor string, scope *common.DataScopeReq) (*ReleasePreviewResponse, error) {
	item, err := s.visibleReviewCluster(ctx, intent.ClusterID, scope)
	if err != nil {
		return nil, err
	}
	if actor == "" || actor == "0" {
		return nil, common.NewUnauthorized("permission.denied")
	}
	s = s.scopedReviewService(ctx)
	if len(intent.Name) > 255 || len(validation.IsDNS1123Label(intent.Namespace)) > 0 || len(validation.IsDNS1123Subdomain(intent.WorkloadName)) > 0 || (intent.ContainerName != "" && len(validation.IsDNS1123Label(intent.ContainerName)) > 0) {
		return nil, common.NewBadRequest("common.param_invalid")
	}
	if !safeReviewImage(intent.Image) {
		return nil, common.NewBadRequest("common.param_invalid")
	}
	if s.namespaceSvc != nil {
		if err := s.namespaceSvc.RequireWrite(intent.ClusterID, intent.Namespace, "release:"+intent.Operation); err != nil {
			return nil, err
		}
	}
	version, err := releaseClusterVersion(s.db.WithContext(ctx), item)
	if err != nil {
		return nil, err
	}
	client, err := s.clientFor(intent.ClusterID, scope)
	if err != nil {
		return nil, err
	}
	bounded, cancel := context.WithTimeout(ctx, s.operationTimeout)
	defer cancel()
	target, image, err := reviewWorkload(bounded, client, intent)
	if err != nil {
		return nil, err
	}
	if target.GetUID() == "" || target.GetResourceVersion() == "" {
		return nil, reviewConflict()
	}
	latest, err := s.visibleReviewCluster(ctx, intent.ClusterID, scope)
	if err != nil {
		return nil, err
	}
	current, err := releaseClusterVersion(s.db.WithContext(ctx), latest)
	if err != nil {
		return nil, err
	}
	if current != version {
		return nil, reviewConflict()
	}
	fingerprint, _, err := fingerprintSnapshot(intent)
	if err != nil {
		return nil, err
	}
	key = strings.TrimSpace(key)
	if key == "" {
		key = uuid.NewString()
	}
	if len(key) > 128 {
		return nil, common.NewBadRequest("common.param_invalid")
	}
	guard := releaseReviewGuard{string(target.GetUID()), target.GetResourceVersion(), version}
	expires := time.Now().UTC().Add(10 * time.Minute)
	ticket := releaseReviewTicket{"k8s-image-release/v1", actor, key, fingerprint, guard, expires.Unix()}
	encoded, err := json.Marshal(ticket)
	if err != nil {
		return nil, err
	}
	token, err := k8spkg.SignReleaseReview(encoded)
	if err != nil {
		return nil, err
	}
	kind := "modified"
	if image == intent.Image {
		kind = "unchanged"
	}
	rollback := safeReviewImage(image)
	if !rollback {
		image = "[hidden]"
	}
	changes := append([]PreviewChange{{"image", "image", kind, image, intent.Image}}, previewWorkloadContext(target, intent.ContainerName)...)
	return &ReleasePreviewResponse{IntentID: key, IntentVersion: token, CurrentVersion: guard.ResourceVersion, TargetVersion: fingerprint, TargetCount: 1, Environment: item.Environment, Namespace: intent.Namespace, WorkloadType: intent.WorkloadType, WorkloadName: intent.WorkloadName, ContainerName: intent.ContainerName, Changes: changes, RiskKeys: []string{"k8s.release.preview_image_only", "business.k8s.release.preview.contextOnly", "k8s.release.preview_rollout_required", "k8s.release.preview_rollback_limits"}, RollbackAvailable: rollback, ExpiresAt: expires}, nil
}

// PreviewCreate reads the authoritative workload without mutation or persistence.
func (s *ReleaseService) PreviewCreate(ctx context.Context, req CreateReleaseRequest, actor string, scope *common.DataScopeReq) (*ReleasePreviewResponse, error) {
	intent, err := normalizeCreateRequest(req)
	if err != nil {
		return nil, err
	}
	return s.previewIntent(ctx, intent, req.IdempotencyKey, actor, scope)
}

func (s *ReleaseService) rollbackIntent(ctx context.Context, id uint64, scope *common.DataScopeReq) (releaseRequestSnapshot, error) {
	var target Release
	if s.db == nil {
		return releaseRequestSnapshot{}, common.ErrDatabaseNotInitialized
	}
	if err := s.releaseQuery(scope).WithContext(ctx).First(&target, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return releaseRequestSnapshot{}, common.NewNotFound("k8s.release.not_found")
		}
		return releaseRequestSnapshot{}, err
	}
	if target.Status != releaseStatusSucceeded || !safeReviewImage(target.ImageBefore) {
		return releaseRequestSnapshot{}, errReleaseNoRollbackTarget
	}
	return releaseRequestSnapshot{Operation: "rollback", Name: target.Name + "-rollback", ClusterID: target.ClusterID, Namespace: target.Namespace, WorkloadType: target.WorkloadType, WorkloadName: target.WorkloadName, ContainerName: target.ContainerName, Image: target.ImageBefore, Strategy: target.Strategy, ServiceID: target.ServiceID, ServiceInstanceID: target.ServiceInstanceID, ServiceName: target.ServiceName, ServiceInstanceName: target.ServiceInstanceName, PreviousReleaseID: target.ID}, nil
}

// PreviewRollback reviews restoring the recorded image, not reversing config or data.
func (s *ReleaseService) PreviewRollback(ctx context.Context, id uint64, req RollbackReleaseRequest, actor string, scope *common.DataScopeReq) (*ReleasePreviewResponse, error) {
	intent, err := s.rollbackIntent(ctx, id, scope)
	if err != nil {
		return nil, err
	}
	return s.previewIntent(ctx, intent, req.IdempotencyKey, actor, scope)
}

func verifyReleaseReview(intent releaseRequestSnapshot, key, actor, token string, allowExpired bool) (*releaseReviewGuard, error) {
	if token == "" {
		return nil, reviewConflict()
	}
	payload, err := k8spkg.OpenReleaseReview(token)
	if err != nil {
		return nil, reviewConflict()
	}
	var ticket releaseReviewTicket
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&ticket); err != nil {
		return nil, reviewConflict()
	}
	fingerprint, _, err := fingerprintSnapshot(intent)
	if err != nil {
		return nil, err
	}
	if ticket.Audience != "k8s-image-release/v1" || ticket.Actor != actor || actor == "" || actor == "0" || ticket.IntentID != strings.TrimSpace(key) || ticket.IntentID == "" || ticket.Fingerprint != fingerprint || (!allowExpired && ticket.Expires <= time.Now().Unix()) || ticket.Guard.UID == "" || ticket.Guard.ResourceVersion == "" || ticket.Guard.ClusterVersion == "" {
		return nil, reviewConflict()
	}
	return &ticket.Guard, nil
}

func (s *ReleaseService) executeReviewed(ctx context.Context, intent releaseRequestSnapshot, key, actor, token string, scope *common.DataScopeReq) (*ReleaseResponse, error) {
	if _, err := s.visibleReviewCluster(ctx, intent.ClusterID, scope); err != nil {
		return nil, err
	}
	// An authenticated expired ticket can only retrieve its matching persisted result.
	// This resolves an ambiguous transport failure without issuing another mutation.
	guard, err := verifyReleaseReview(intent, key, actor, token, true)
	if err != nil {
		return nil, err
	}
	intent.Review = guard
	var existing Release
	replayErr := s.releaseQuery(scope).WithContext(ctx).Where("idempotency_key = ? AND created_by = ?", strings.TrimSpace(key), actor).First(&existing).Error
	if replayErr == nil {
		fingerprint, _, e := fingerprintSnapshot(intent)
		if e != nil {
			return nil, e
		}
		if fingerprint != existing.RequestFingerprint {
			return nil, reviewConflict()
		}
		response := toResponse(&existing)
		return &response, nil
	}
	if !errors.Is(replayErr, gorm.ErrRecordNotFound) {
		return nil, replayErr
	}
	intent.Review = nil
	guard, err = verifyReleaseReview(intent, key, actor, token, false)
	if err != nil {
		return nil, err
	}
	intent.Review = guard
	if s.namespaceSvc != nil {
		if err := s.namespaceSvc.RequireWrite(intent.ClusterID, intent.Namespace, "release:"+intent.Operation); err != nil {
			return nil, err
		}
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if s.db == nil {
		return nil, common.ErrDatabaseNotInitialized
	}
	s = s.scopedReviewService(ctx)
	record, replay, err := s.persistIntentWithKey(intent, key, actor, scope)
	if err != nil {
		return nil, err
	}
	if replay {
		resp := toResponse(record)
		return &resp, nil
	}
	if err := s.execute(record.ID, scope); err != nil {
		return nil, err
	}
	return s.responseByID(record.ID, scope)
}

// CreateReviewed is the public image mutation boundary.
func (s *ReleaseService) CreateReviewed(ctx context.Context, req CreateReleaseRequest, actor string, scope *common.DataScopeReq) (*ReleaseResponse, error) {
	intent, err := normalizeCreateRequest(req)
	if err != nil {
		return nil, err
	}
	return s.executeReviewed(ctx, intent, req.IdempotencyKey, actor, req.IntentVersion, scope)
}

// RollbackReviewed is the public recorded-image restoration boundary.
func (s *ReleaseService) RollbackReviewed(ctx context.Context, id uint64, req RollbackReleaseRequest, actor string, scope *common.DataScopeReq) (*ReleaseResponse, error) {
	intent, err := s.rollbackIntent(ctx, id, scope)
	if err != nil {
		return nil, err
	}
	return s.executeReviewed(ctx, intent, req.IdempotencyKey, actor, req.IntentVersion, scope)
}

func requireReviewTarget(item metav1.Object, guard *releaseReviewGuard) error {
	if guard != nil && (string(item.GetUID()) != guard.UID || item.GetResourceVersion() != guard.ResourceVersion) {
		return reviewConflict()
	}
	return nil
}

func (s *ReleaseService) operationContext() context.Context {
	if s.requestCtx != nil {
		return s.requestCtx
	}
	return context.Background()
}
func (s *ReleaseService) scopedReviewService(ctx context.Context) *ReleaseService {
	return &ReleaseService{db: s.db.WithContext(ctx), clusterSvc: s.clusterSvc, namespaceSvc: s.namespaceSvc, clientProvider: s.clientProvider, operationTimeout: s.operationTimeout, pollInterval: s.pollInterval, requestCtx: ctx}
}
