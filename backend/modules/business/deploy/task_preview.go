package deploy

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/duanxldragon/pantheon-base/backend/modules/business/cmdb"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"strings"
	"time"
)

// ErrTaskPreviewRequired indicates a task that is not in a reviewable state.
var ErrTaskPreviewRequired = errors.New("business.deploy.preview.required")

// ErrTaskPreviewConflict indicates the preview no longer matches the frozen task state.
var ErrTaskPreviewConflict = errors.New("business.deploy.preview.conflict")

// Derive a separate signing key from the existing deployment secret, shared by
// all replicas. Rotation invalidates outstanding reviews and requires refresh.
func taskPreviewSigningKey() ([]byte, error) {
	root, err := deployCredentialKey()
	if err != nil {
		return nil, err
	}
	mac := hmac.New(sha256.New, root)
	_, _ = mac.Write([]byte("pantheon-ops/deploy-preview/v1"))
	return mac.Sum(nil), nil
}

// TaskPreviewChange is a single diff between the current and intended task state.
type TaskPreviewChange struct {
	ID     string `json:"id"`
	Field  string `json:"field"`
	Kind   string `json:"kind"`
	Before string `json:"before,omitempty"`
	After  string `json:"after,omitempty"`
}

// TaskPreviewResponse describes the reviewed deployment intent.
type TaskPreviewResponse struct {
	IntentID          string              `json:"intentId"`
	IntentVersion     string              `json:"intentVersion"`
	CurrentVersion    string              `json:"currentVersion"`
	TargetVersion     string              `json:"targetVersion"`
	TargetCount       int                 `json:"targetCount"`
	BusinessScopeName string              `json:"businessScopeName"`
	Changes           []TaskPreviewChange `json:"changes"`
	Plan              *TaskPreviewPlan    `json:"plan"`
	RiskKeys          []string            `json:"riskKeys"`
	RollbackAvailable bool                `json:"rollbackAvailable"`
	ExpiresAt         time.Time           `json:"expiresAt"`
}
type taskPreviewTicket struct {
	TaskID   uint64 `json:"taskId"`
	Actor    string `json:"actor"`
	Revision string `json:"revision"`
	Expires  int64  `json:"expires"`
}

func signPreviewTicket(ticket taskPreviewTicket) (string, error) {
	payload, err := json.Marshal(ticket)
	if err != nil {
		return "", err
	}
	key, err := taskPreviewSigningKey()
	if err != nil {
		return "", err
	}
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write(payload)
	return base64.RawURLEncoding.EncodeToString(payload) + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil)), nil
}
func parsePreviewTicket(token string) (taskPreviewTicket, error) {
	var ticket taskPreviewTicket
	if len(token) > 1024 {
		return ticket, ErrTaskPreviewConflict
	}
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return ticket, ErrTaskPreviewConflict
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return ticket, ErrTaskPreviewConflict
	}
	signature, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return ticket, ErrTaskPreviewConflict
	}
	key, err := taskPreviewSigningKey()
	if err != nil {
		return ticket, err
	}
	mac := hmac.New(sha256.New, key)
	_, _ = mac.Write(payload)
	if !hmac.Equal(signature, mac.Sum(nil)) {
		return ticket, ErrTaskPreviewConflict
	}
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&ticket); err != nil {
		return ticket, ErrTaskPreviewConflict
	}
	if ticket.Expires <= time.Now().Unix() {
		return ticket, ErrTaskPreviewConflict
	}
	return ticket, nil
}
func taskPreviewRevision(db *gorm.DB, task *DeployTask, hosts []cmdbHostSnapshot, req StartTaskRequest, credentialVersion uint64, lock bool) (string, error) {
	var secrets []struct {
		SecretKey       string
		SecretEncrypted string
	}
	query := db.Model(&DeployTaskSecret{}).Where(taskIDWhereClause, task.ID).Order("secret_key ASC")
	if lock {
		query = query.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	if err := query.Find(&secrets).Error; err != nil {
		return "", err
	}
	request := struct {
		CredentialRefID uint64
		TimeoutSeconds  int
		HostFingerprint string
	}{req.CredentialRefID, int(normalizeDeployExecutionTimeout(req.TimeoutSeconds).Seconds()), strings.TrimSpace(req.HostFingerprint)}
	material := struct {
		Task              *DeployTask
		Hosts             []cmdbHostSnapshot
		Secrets           any
		Request           any
		CredentialVersion uint64
	}{task, hosts, secrets, request, credentialVersion}
	payload, err := json.Marshal(material)
	if err != nil {
		return "", err
	}
	hash := sha256.Sum256(payload)
	return hex.EncodeToString(hash[:]), nil
}

// PreviewTask computes a deployment preview for the caller session.
func (s *DeployService) PreviewTask(id uint64, req StartTaskRequest, actor string, scope *common.DataScopeReq) (*TaskPreviewResponse, error) {
	return s.PreviewTaskContext(context.Background(), id, req, actor, scope)
}

// PreviewTaskContext computes a deployment preview with an explicit context.
func (s *DeployService) PreviewTaskContext(ctx context.Context, id uint64, req StartTaskRequest, actor string, scope *common.DataScopeReq) (*TaskPreviewResponse, error) {
	ctx, cancel := context.WithTimeout(ctx, s.ownershipTiming().operation)
	defer cancel()
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return s.targetDiscoveryService(ctx).previewTask(id, req, actor, scope)
}
func (s *DeployService) previewTask(id uint64, req StartTaskRequest, actor string, scope *common.DataScopeReq) (*TaskPreviewResponse, error) {
	task, err := s.loadVisibleTask(id, scope)
	if err != nil {
		return nil, err
	}
	if task.Status != TaskStatusDraft && task.Status != TaskStatusPending {
		return nil, ErrTaskPreviewConflict
	}
	if err := validateRetireTaskIntent(*task); err != nil {
		return nil, err
	}
	credentialVersion := uint64(0)
	if task.ExecutorType == ExecutorTypeSSH {
		resolved, version, err := s.prepareStartCredential(req, *task, scope == nil || scope.IsAdmin)
		if err != nil {
			return nil, err
		}
		req = resolved
		credentialVersion = version
	}
	hosts, err := s.prepareStartTargets(task, req, scope)
	if err != nil {
		return nil, err
	}
	for _, host := range hosts {
		if !hostStatusAllowedForAction(host.Status, task.Action) {
			return nil, ErrTaskPreviewConflict
		}
	}
	if err := s.validateSelectionForPreview(s.requestContext(), task, scope, false); err != nil {
		return nil, err
	}
	revision, err := taskPreviewRevision(s.db, task, hosts, req, credentialVersion, false)
	if err != nil {
		return nil, err
	}
	expires := time.Now().UTC().Add(10 * time.Minute)
	token, err := signPreviewTicket(taskPreviewTicket{TaskID: id, Actor: actor, Revision: revision, Expires: expires.Unix()})
	if err != nil {
		return nil, err
	}
	plan, err := s.taskPreviewPlan(task)
	if err != nil {
		return nil, err
	}
	targetHash := sha256.Sum256(task.ExecutionSnapshot)
	result := &TaskPreviewResponse{IntentID: fmt.Sprintf("deploy:%d", id), IntentVersion: token, CurrentVersion: revision, TargetVersion: hex.EncodeToString(targetHash[:]), TargetCount: len(hosts), BusinessScopeName: task.BusinessScopeName, Changes: []TaskPreviewChange{}, RiskKeys: []string{"business.deploy.preview.healthUnknown", "business.deploy.preview.rollbackManual"}, RollbackAvailable: false, ExpiresAt: expires}
	result.Plan = plan
	for _, host := range hosts {
		current := ""
		known := true
		var installed []struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		}
		if len(host.InstalledComponents) > 0 && json.Unmarshal(host.InstalledComponents, &installed) != nil {
			known = false
		}
		for _, component := range installed {
			if component.Name == task.PackageName {
				current = component.Version
			}
		}
		kind := "modified"
		target := task.PackageVersion
		if !known {
			kind = "conflict"
		} else if current == "" {
			kind = "added"
		} else if current == target {
			kind = "unchanged"
		}
		switch task.Action {
		case TaskActionUninstall:
			kind = "removed"
			target = ""
		case TaskActionRetire:
			if known {
				kind = "unchanged"
			}
			target = current
		}
		result.Changes = append(result.Changes, TaskPreviewChange{ID: fmt.Sprintf("host:%d", host.ID), Field: host.Hostname + " / " + task.PackageName, Kind: kind, Before: current, After: target})
	}
	if len(hosts) > 100 {
		result.RiskKeys = append(result.RiskKeys, "business.deploy.preview.largeScope")
	}
	if task.Action == TaskActionUninstall || task.Action == TaskActionRetire || task.Action == TaskActionStop {
		result.RiskKeys = append(result.RiskKeys, "business.deploy.preview.disruptive")
	}
	return result, nil
}
func (s *DeployService) validateSelectionForPreview(ctx context.Context, task *DeployTask, scope *common.DataScopeReq, lock bool) error {
	var snapshot deployTargetSnapshot
	if len(task.TargetSnapshot) == 0 {
		return nil
	}
	if err := json.Unmarshal(task.TargetSnapshot, &snapshot); err != nil {
		return ErrTaskPreviewConflict
	}
	if snapshot.Selection == nil {
		return nil
	}
	provider, ok := s.cmdbCapability.(cmdb.TargetSelectionCapability)
	if !ok {
		return cmdb.ErrTargetSelectionInvalid
	}
	resolved, err := provider.ResolveTargetSelection(ctx, *snapshot.Selection, scope, lock)
	if err != nil {
		return err
	}
	if resolved.ScopeVersion != snapshot.ScopeVersion || resolved.UnavailableCount > 0 || resolved.IneligibleCount > 0 {
		return cmdb.ErrTargetSelectionConflict
	}
	return nil
}
func (c *startTaskClaim) validatePreview(tx *gorm.DB) error {
	var selection deployTargetSnapshot
	if len(c.task.TargetSnapshot) > 0 && json.Unmarshal(c.task.TargetSnapshot, &selection) != nil {
		return ErrTaskPreviewConflict
	}
	reviewed := c.req.RequirePreview || c.req.IntentVersion != "" || selection.Selection != nil
	var ticket taskPreviewTicket
	if reviewed {
		if c.req.IntentVersion == "" {
			return ErrTaskPreviewRequired
		}
		var err error
		ticket, err = parsePreviewTicket(c.req.IntentVersion)
		if err != nil {
			return err
		}
		if ticket.TaskID != c.task.ID || ticket.Actor != c.actor {
			return ErrTaskPreviewConflict
		}
	}
	provider := c.service.cmdbCapability
	owner, ok := provider.(cmdb.TargetSelectionCapability)
	if !ok {
		return cmdb.ErrTargetSelectionInvalid
	}
	bound, valid := owner.WithTargetTransaction(tx).(cmdb.DeployCMDBCapability)
	if !valid {
		return cmdb.ErrTargetSelectionInvalid
	}
	provider = bound
	inner := c.service.targetTransactionService(tx, provider)
	if err := inner.validateSelectionForPreview(tx.Statement.Context, c.task, c.dataScope, true); err != nil {
		return err
	}
	hosts, err := inner.prepareStartTargets(c.task, c.req, c.dataScope)
	if err != nil {
		return err
	}
	// Final locked discovery cannot silently grow/shrink the admitted footprint.
	candidate := make(map[uint64]bool, len(c.hosts))
	for _, h := range c.hosts {
		candidate[h.ID] = true
	}
	if len(hosts) != len(candidate) {
		return ErrTaskPreviewConflict
	}
	for _, h := range hosts {
		if !candidate[h.ID] {
			return ErrTaskPreviewConflict
		}
	}
	if c.task.ExecutorType == ExecutorTypeSSH {
		var credential DeployCredentialRef
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&credential, c.req.CredentialRefID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrTaskPreviewConflict
			}
			return err
		}
		if credential.Version != c.credentialVersion ||
			credential.Status != bizScopeStatusActive ||
			!credentialMatchesTaskScope(credential, *c.task, c.dataScope == nil || c.dataScope.IsAdmin) {
			return ErrTaskPreviewConflict
		}
	}
	if reviewed {
		revision, err := taskPreviewRevision(tx, c.task, hosts, c.req, c.credentialVersion, true)
		if err != nil {
			return err
		}
		if ticket.Revision != revision {
			return ErrTaskPreviewConflict
		}
	}
	c.hosts = hosts
	return nil
}
