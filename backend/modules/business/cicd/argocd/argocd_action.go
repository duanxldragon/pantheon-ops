package argocd

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/logging"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// SyncApplication keeps the legacy signature for a failclosed compatibility
// boundary. It cannot execute: the old webhook has no verified actor/review.
// Automation must migrate to a reviewed machine contract or native Argo policy.
func (s *ArgoCDService) SyncApplication(id uint64, revision string) (*SyncHistoryResponse, error) {
	return s.SyncApplicationContext(context.Background(), id, revision)
}

// SyncApplicationContext records provider acceptance separately from completion.
func (s *ArgoCDService) SyncApplicationContext(ctx context.Context, id uint64, revision string, actors ...ActionActor) (*SyncHistoryResponse, error) {
	return s.submitApplicationAction(ctx, id, revision, false, actors...)
}

// RollbackApplication restores a deployed provider history entry.
func (s *ArgoCDService) RollbackApplication(id uint64, revision string, actors ...ActionActor) (*SyncHistoryResponse, error) {
	return s.RollbackApplicationContext(context.Background(), id, revision, actors...)
}

// RollbackApplicationContext keeps historical source/configuration owned by Argo CD.
func (s *ArgoCDService) RollbackApplicationContext(ctx context.Context, id uint64, revision string, actors ...ActionActor) (*SyncHistoryResponse, error) {
	return s.submitApplicationAction(ctx, id, revision, true, actors...)
}

func (s *ArgoCDService) submitApplicationAction(ctx context.Context, id uint64, revision string, rollback bool, actors ...ActionActor) (*SyncHistoryResponse, error) {
	actor := actionActor(actors)
	plain, err := DecryptSecret(actor.IntentVersion)
	var review applicationReview
	if err != nil || plain == "" || json.Unmarshal([]byte(plain), &review) != nil || review.ApplicationID != id || review.Rollback != rollback || !gitRevision.MatchString(review.Revision) || review.UID == "" {
		return nil, common.NewConflict(msgParamInvalid)
	}
	if actor.ID == 0 || review.Actor != actor.ID {
		return nil, common.NewForbidden("common.forbidden")
	}
	if revision != review.RequestedRevision && revision != review.Revision {
		return nil, common.NewConflict(msgParamInvalid)
	}
	opCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	// Exact ticket correlation is private and stable. Recovery precedes expiry
	// and provider/config drift checks and can only return a persisted result.
	ticketHash := sha256.Sum256([]byte(actor.IntentVersion))
	correlation := hex.EncodeToString(ticketHash[:])
	if result, found, err := s.recoverReviewedAction(opCtx, s.db, id, actor, review, correlation); found || err != nil {
		return result, err
	}
	if review.Expires < time.Now().Unix() {
		return nil, common.NewConflict(msgParamInvalid)
	}
	// Old reserved actions recover above. Obsolete undispatched reviews must
	// obtain a new capability-bound ticket; zero is a valid native history ID.
	if review.Protocol != reviewedActionsProtocol || !reviewedRevision.MatchString(review.Revision) || (rollback && (review.HistoryID == nil || *review.HistoryID < 0)) || (!rollback && review.HistoryID != nil) {
		return nil, common.NewConflict(msgParamInvalid)
	}
	plan, err := s.prepareApplicationAction(opCtx, id, ApplicationPreviewRequest{Revision: revision, Rollback: rollback}, actor)
	if err != nil {
		return nil, err
	}
	if review.Revision != plan.operation.Sync.Revision || review.Fingerprint != plan.fingerprint || review.UID != plan.provider.Metadata.UID || review.Version != plan.provider.Metadata.ResourceVersion || review.Diff != plan.diff || !sameHistoryID(review.HistoryID, plan.historyID) {
		return nil, common.NewConflict(msgParamInvalid)
	}
	nonce, err := actionNonce()
	if err != nil {
		return nil, safeApplicationActionError(opCtx, rollback)
	}
	plan.operation.Info = []argoOperationInfo{{Name: actionInfoName, Value: nonce}}
	metadata := actionMetadata{Nonce: nonce, UID: review.UID, Revision: review.Revision, Rollback: rollback, Protocol: review.Protocol, HistoryID: review.HistoryID, Actor: actor.ID, Review: correlation,
		ServerID: plan.server.ID, ClusterID: plan.server.ClusterID, SourceURL: plan.app.SourceRepoURL, DestinationServer: plan.app.DestinationServer, Namespace: plan.app.DestinationNS}
	history := ArgoCDSyncHistory{ApplicationID: id, Revision: review.Revision, Status: AppStatusUnknown, StartedAt: time.Now(), Message: encodeActionMetadata(metadata)}
	var recovered *SyncHistoryResponse
	// Serialize only the existing application/history reservation. Unknown records
	// block another submission across restarts, even if their metadata is legacy.
	err = s.db.WithContext(opCtx).Transaction(func(tx *gorm.DB) error {
		var app ArgoCDApplication
		if e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&app, id).Error; e != nil {
			return e
		}
		// A concurrent request with this ticket may have committed while remote
		// reads were in flight. Recheck under the reservation serialization lock.
		if result, found, e := s.recoverReviewedAction(opCtx, tx, id, actor, review, correlation); found || e != nil {
			recovered = result
			return e
		}
		var count int64
		if e := tx.Model(&ArgoCDSyncHistory{}).Where("application_id = ? AND status IN ? AND finished_at IS NULL", id, []string{AppStatusUnknown, "Running"}).Count(&count).Error; e != nil {
			return e
		}
		if count != 0 {
			return common.NewConflict(msgParamInvalid)
		}
		// Re-read current configuration/credential references after remote reads.
		var server ArgoCDServer
		if e := tx.First(&server, app.ServerID).Error; e != nil {
			return e
		}
		var repositories []ArgoCDRepository
		if e := tx.Where("server_id = ? AND repo_url = ?", server.ID, app.SourceRepoURL).Order("id ASC").Find(&repositories).Error; e != nil {
			return e
		}
		if !actionConfigurationAvailable(&server, repositories) {
			return common.NewForbidden("common.forbidden")
		}
		if actionFingerprint(&app, &server, repositories) != review.Fingerprint {
			return common.NewConflict(msgParamInvalid)
		}
		return tx.Create(&history).Error
	})
	if err != nil {
		if errors.Is(err, common.ErrConflict) || errors.Is(err, common.ErrForbidden) {
			return nil, err
		}
		return nil, safeApplicationActionError(opCtx, rollback)
	}
	if recovered != nil {
		return recovered, nil
	}
	action := "sync"
	if rollback {
		action = "rollback"
	}
	dispatchStarted := false
	// Keep only the existing configuration rows locked through the bounded
	// provider call. A credential/config deletion or rotation that committed
	// before this read cannot be authorized by the earlier request snapshot.
	err = s.db.WithContext(opCtx).Transaction(func(tx *gorm.DB) error {
		var app ArgoCDApplication
		if e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&app, id).Error; e != nil {
			return e
		}
		var server ArgoCDServer
		if e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&server, app.ServerID).Error; e != nil {
			return e
		}
		var repositories []ArgoCDRepository
		if e := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("server_id = ? AND repo_url = ?", server.ID, app.SourceRepoURL).Order("id ASC").Find(&repositories).Error; e != nil {
			return e
		}
		if !actionConfigurationAvailable(&server, repositories) {
			return common.NewForbidden("common.forbidden")
		}
		if actionFingerprint(&app, &server, repositories) != review.Fingerprint {
			return common.NewConflict(msgParamInvalid)
		}
		if e := plan.client.requireReviewedCapabilities(opCtx, app.Name, plan.provider); e != nil {
			return e
		}
		if e := s.authorizeAction(opCtx, actor, action, &app, &server, []string{app.SourceRepoURL}); e != nil {
			return e
		}
		dispatchStarted = true
		return plan.client.submitReviewedOperation(opCtx, app.Name, plan.provider, plan.operation, plan.historyID)
	})
	appUpdates := map[string]interface{}{"status": "Running", "last_error": ""}
	if err == nil {
		history.Status = "Running"
	}
	if err != nil {
		var rejected *providerStatusError
		definitiveRejection := !dispatchStarted || (errors.As(err, &rejected) && definitiveApplicationRejection(rejected.status))
		if !errors.Is(err, common.ErrForbidden) && !errors.Is(err, common.ErrConflict) {
			err = safeApplicationActionError(opCtx, rollback)
		}
		history.Status = AppStatusUnknown
		metadata.Error = publicActionMessage(err.Error())
		appUpdates["status"] = AppStatusUnknown
		if definitiveRejection {
			now := time.Now()
			history.Status = "Failed"
			history.FinishedAt = &now
			appUpdates["status"] = AppStatusError
		}
		appUpdates["last_error"] = metadata.Error
	}
	history.Message = encodeActionMetadata(metadata)
	recordCtx, recordCancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer recordCancel()
	if recordErr := s.db.WithContext(recordCtx).Transaction(func(tx *gorm.DB) error {
		// A fast controller/reconciler may already have resolved the exact action.
		if e := tx.Model(&history).Where("finished_at IS NULL").Updates(map[string]interface{}{"status": history.Status, "message": history.Message, "finished_at": history.FinishedAt}).Error; e != nil {
			return e
		}
		var current ArgoCDSyncHistory
		if e := tx.First(&current, history.ID).Error; e != nil {
			return e
		}
		if current.FinishedAt != nil && !definitiveApplicationRejectionForHistory(history) {
			history = current
			return nil
		}
		history = current
		return tx.Model(&plan.app).Updates(appUpdates).Error
	}); recordErr != nil {
		logging.Error("Argo CD action outcome persistence failed", zap.String("action", action), zap.Uint64("application_id", id), zap.Uint64("history_id", history.ID), zap.String("error_type", fmt.Sprintf("%T", recordErr)))
		if err != nil {
			return nil, err
		}
		return nil, safeApplicationActionError(opCtx, rollback)
	}
	if err != nil {
		return nil, err
	}
	return actionHistoryResponse(history), nil
}

func actionHistoryResponse(history ArgoCDSyncHistory) *SyncHistoryResponse {
	return &SyncHistoryResponse{ID: history.ID, Revision: history.Revision, Status: history.Status, Message: publicActionMessage(history.Message), StartedAt: history.StartedAt, FinishedAt: history.FinishedAt}
}

// recoverReviewedAction has no provider/client path and never creates a record.
// It checks both current authorization and the exact originally reviewed source
// and target. Revocation still denies access; expired execution permission and
// provider drift cannot make an already dispatched result disappear.
func (s *ArgoCDService) recoverReviewedAction(ctx context.Context, db *gorm.DB, id uint64, actor ActionActor, review applicationReview, correlation string) (*SyncHistoryResponse, bool, error) {
	var records []ArgoCDSyncHistory
	// The bounded candidate query uses only our hex ticket hash; metadata is
	// decoded and checked exactly below. No latest-50 public-history limit.
	if err := db.WithContext(ctx).Where("application_id = ? AND message LIKE ?", id, "%\"Review\":\""+correlation+"\"%").Order("id ASC").Limit(2).Find(&records).Error; err != nil {
		return nil, false, resourceReadError(ctx, err, "cicd.argocd.history_get_failed")
	}
	if len(records) == 0 {
		return nil, false, nil
	}
	metadata, ok := decodeActionMetadata(records[0].Message)
	if len(records) != 1 || !ok || metadata.Review != correlation || metadata.Actor != actor.ID || metadata.UID != review.UID || metadata.Revision != review.Revision || metadata.Rollback != review.Rollback || records[0].Revision != review.Revision || metadata.ServerID == 0 || metadata.SourceURL == "" {
		return nil, true, common.NewConflict(msgParamInvalid)
	}
	if metadata.Protocol != review.Protocol || !sameHistoryID(metadata.HistoryID, review.HistoryID) {
		return nil, true, common.NewConflict(msgParamInvalid)
	}
	var current ArgoCDApplication
	if err := db.WithContext(ctx).First(&current, id).Error; err != nil {
		return nil, true, resourceReadError(ctx, err, "cicd.argocd.app_get_failed")
	}
	var server ArgoCDServer
	if err := db.WithContext(ctx).First(&server, current.ServerID).Error; err != nil {
		return nil, true, resourceReadError(ctx, err, "cicd.argocd.server_get_failed")
	}
	action := "sync"
	if review.Rollback {
		action = "rollback"
	}
	if err := s.authorizeAction(ctx, actor, action, &current, &server, []string{current.SourceRepoURL}); err != nil {
		return nil, true, err
	}
	// Current and original source registrations must still be accessible. A
	// credential rotation is compatible with result reads; deletion/revocation
	// is not. Recovery does not decrypt credentials or test provider conditions.
	for _, source := range []struct {
		serverID uint64
		url      string
	}{{current.ServerID, current.SourceRepoURL}, {metadata.ServerID, metadata.SourceURL}} {
		var repositories []ArgoCDRepository
		if err := db.WithContext(ctx).Where("server_id = ? AND repo_url = ?", source.serverID, source.url).Find(&repositories).Error; err != nil {
			return nil, true, resourceReadError(ctx, err, "cicd.argocd.repo_get_failed")
		}
		if len(repositories) != 1 || (repositories[0].Status != "connected" && repositories[0].Status != "unknown") {
			return nil, true, common.NewForbidden("common.forbidden")
		}
	}
	var originalServer ArgoCDServer
	if err := db.WithContext(ctx).First(&originalServer, metadata.ServerID).Error; err != nil {
		return nil, true, resourceReadError(ctx, err, "cicd.argocd.server_get_failed")
	}
	original := ArgoCDApplication{ID: id, ServerID: metadata.ServerID, SourceRepoURL: metadata.SourceURL, DestinationServer: metadata.DestinationServer, DestinationNS: metadata.Namespace}
	if err := s.authorizeAction(ctx, actor, action, &original, &originalServer, []string{metadata.SourceURL}); err != nil {
		return nil, true, err
	}
	if originalServer.ClusterID != metadata.ClusterID {
		// Also authorize the original cluster identity if registration changed.
		originalServer.ClusterID = metadata.ClusterID
		if err := s.authorizeAction(ctx, actor, action, &original, &originalServer, []string{metadata.SourceURL}); err != nil {
			return nil, true, err
		}
	}
	return actionHistoryResponse(records[0]), true, nil
}

func definitiveApplicationRejectionForHistory(history ArgoCDSyncHistory) bool {
	return history.Status == "Failed" && history.FinishedAt != nil
}

const actionInfoName = "pantheon-action-id"
const actionMetadataPrefix = "argo-action:v1:"

type actionMetadata struct {
	Nonce, UID, Revision                    string
	Rollback                                bool
	Protocol                                string `json:",omitempty"`
	HistoryID                               *int64 `json:",omitempty"`
	Actor                                   uint64
	Review                                  string `json:",omitempty"`
	ServerID, ClusterID                     uint64 `json:",omitempty"`
	SourceURL, DestinationServer, Namespace string `json:",omitempty"`
	Error                                   string
}

func sameHistoryID(a, b *int64) bool {
	return (a == nil && b == nil) || (a != nil && b != nil && *a == *b)
}

func encodeActionMetadata(metadata actionMetadata) string {
	data, _ := json.Marshal(metadata)
	return actionMetadataPrefix + string(data)
}
func decodeActionMetadata(value string) (actionMetadata, bool) {
	var metadata actionMetadata
	if !strings.HasPrefix(value, actionMetadataPrefix) {
		return metadata, false
	}
	err := json.Unmarshal([]byte(strings.TrimPrefix(value, actionMetadataPrefix)), &metadata)
	return metadata, err == nil && metadata.Nonce != "" && metadata.UID != "" && gitRevision.MatchString(metadata.Revision)
}
func publicActionMessage(value string) string {
	if metadata, ok := decodeActionMetadata(value); ok {
		value = metadata.Error
	}
	switch value {
	case "cicd.argocd.app_sync_failed", "cicd.argocd.app_rollback_failed", "context canceled", "context deadline exceeded":
		return value
	}
	return ""
}

func definitiveApplicationRejection(status int) bool {
	// Standard client rejections are terminal, including representation and
	// precondition failures. HTTP408 and nonstandard cancellation/proxy codes
	// do not prove that the provider did not perform the mutation.
	return status >= http.StatusBadRequest && status < http.StatusInternalServerError &&
		status != http.StatusRequestTimeout && http.StatusText(status) != ""
}

func safeApplicationActionError(ctx context.Context, rollback bool) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if rollback {
		return errors.New("cicd.argocd.app_rollback_failed")
	}
	return errors.New("cicd.argocd.app_sync_failed")
}

// Provider operation messages may include repository credentials or rendered
// configuration. Persist only the existing localized business failure key.
func applicationOperationMessage(status *ArgoCDApplicationStatus) string {
	if status.OperationState != nil && (status.OperationState.Phase == "Failed" || status.OperationState.Phase == "Error") {
		return "cicd.argocd.app_sync_failed"
	}
	return ""
}
