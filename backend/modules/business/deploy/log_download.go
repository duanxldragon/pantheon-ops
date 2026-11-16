package deploy

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/pkg/auditpolicy"
	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/contracts/businessauth"
	"github.com/duanxldragon/pantheon-base/backend/pkg/permissionroutes"
	"github.com/gin-gonic/gin"
)

const (
	deployLogDownloadRoute    = "/tasks/:id/log-download"
	deployLogDownloadTemplate = "/api/v1/business/deploy" + deployLogDownloadRoute
	deployLogDownloadRows     = 10000
	deployLogDownloadBytes    = 2 * 1024 * 1024
	deployLogChunkBytes       = 16384
	// Reserve the response envelope and fixed snapshot metadata, including the
	// longer false truncation value. Items are counted by their encoded size.
	deployLogEnvelopeBytes = 1024
)

func registerDeployLogPolicies() error {
	if err := auditpolicy.RegisterMetadataOnly([]auditpolicy.Route{{Method: "GET", Template: deployLogDownloadTemplate}}); err != nil {
		return err
	}
	return permissionroutes.Register("business:deploy", []permissionroutes.Binding{{Permission: "business:deploy:task:export", Routes: []permissionroutes.Route{
		{Method: "GET", Path: "/api/v1/business/deploy/tasks/export"},
		{Method: "GET", Path: deployLogDownloadTemplate},
	}}})
}

// RegisterLogDownloadRoutes wires the log download route group.
func (h *DeployHandler) RegisterLogDownloadRoutes(r gin.IRoutes) {
	r.GET(deployLogDownloadRoute, h.DownloadTaskLogs)
}

// Every output page is bounded and uses the same task/host/step projection as
// the viewer. Unknown secrets cannot be inferred from arbitrary application output.
func (s *DeployService) safeTaskLogItems(taskID uint64, chunks []DeployTaskLogChunk) ([]TaskLogItem, bool, error) {
	secrets, complete, err := s.loadDeployTaskSecretMaterial(taskID)
	if err != nil || !complete {
		return nil, false, errors.New("deploytask.logs_failed")
	}
	return redactTaskLogItems(chunks, secrets)
}

func redactTaskLogItems(chunks []DeployTaskLogChunk, secrets map[string]string) ([]TaskLogItem, bool, error) {
	keys := make([]string, 0, len(secrets))
	for key, value := range secrets {
		if value != "" {
			keys = append(keys, key)
		}
	}
	sort.Slice(keys, func(i, j int) bool {
		if len(secrets[keys[i]]) != len(secrets[keys[j]]) {
			return len(secrets[keys[i]]) > len(secrets[keys[j]])
		}
		return keys[i] < keys[j]
	})
	pairs := make([]string, 0, len(keys)*2)
	for _, key := range keys {
		pairs = append(pairs, secrets[key], secretSentinelPrefix+key+"}}")
	}
	// A single replacement pass cannot re-redact its own generated markers.
	replacer := strings.NewReplacer(pairs...)
	items := make([]TaskLogItem, 0, len(chunks))
	truncated := false
	for _, chunk := range chunks {
		content := strings.ToValidUTF8(replacer.Replace(chunk.Content), "")
		if len(content) > deployLogChunkBytes {
			content = strings.ToValidUTF8(content[:deployLogChunkBytes], "")
			truncated = true
		}
		items = append(items, TaskLogItem{ID: chunk.ID, Sequence: chunk.Sequence, Stream: chunk.Stream,
			TaskHostID: chunk.TaskHostID, TaskStepID: chunk.TaskStepID, AttemptNo: chunk.AttemptNo,
			Content: content, CreatedAt: chunk.CreatedAt.Format(time.RFC3339)})
	}
	return items, truncated, nil
}

func reauthorizeLogDownload(c *gin.Context) (*common.DataScopeReq, error) {
	input, err := businessauth.FromGin(c)
	if err != nil {
		return nil, businessauth.ErrDenied
	}
	for _, key := range []string{"business:deploy:task:export", "business:deploy:task:detail"} {
		action := input.Action
		action.Permission = key
		if _, err := businessauth.Reauthorize(c, &action); err != nil {
			return nil, businessauth.ErrDenied
		}
	}
	return common.GetDataScope(c), nil
}

// DownloadTaskLogs returns a safe downloadable snapshot, not an unbounded log
// archive. The response declares every retention/size limit and never audits output.
func (h *DeployHandler) DownloadTaskLogs(c *gin.Context) {
	common.SetAuditMetadata(c, "business.deploy.logs.download", common.BusinessExport)
	common.SetAuditParam(c, "")
	common.SetAuditResult(c, "")
	id, ok := parseIDParam(c)
	if !ok {
		return
	}
	after, filter, err := parseTaskLogQuery(c)
	if err != nil {
		common.Fail(c, common.CodeParamInvalid, msgParamInvalid)
		return
	}
	ctx, cancel := context.WithTimeout(c.Request.Context(), 10*time.Second)
	defer cancel()
	c.Request = c.Request.WithContext(ctx)
	reads := h.svc.targetRequestService(ctx)
	items := make([]TaskLogItem, 0)
	bytesUsed, limited := deployLogEnvelopeBytes, false
	for {
		scope, authErr := reauthorizeLogDownload(c)
		if authErr != nil {
			common.FailWithCode(c, 403, "permission.denied")
			return
		}
		chunks, loadErr := reads.loadScopedLogChunks(id, after, filter, scope)
		if loadErr != nil {
			failDeployTaskError(c, loadErr, "deploytask.logs_failed")
			return
		}
		page, clipped, safeErr := reads.safeTaskLogItems(id, chunks)
		if safeErr != nil {
			failDeployTaskError(c, safeErr, "deploytask.logs_failed")
			return
		}
		limited = limited || clipped
		for _, item := range page {
			encoded, encodeErr := json.Marshal(item)
			if encodeErr != nil {
				common.Fail(c, common.CodeError, "deploytask.logs_failed")
				return
			}
			if len(items) == deployLogDownloadRows || bytesUsed+len(encoded)+1 > deployLogDownloadBytes {
				limited = true
				break
			}
			items = append(items, item)
			bytesUsed += len(encoded) + 1
			after = item.Sequence
		}
		if len(page) == 0 || len(page) < 500 || filter.firstError || len(items) == deployLogDownloadRows || bytesUsed >= deployLogDownloadBytes || (len(page) > 0 && after != page[len(page)-1].Sequence) {
			if len(items) == deployLogDownloadRows {
				limited = true
			}
			break
		}
	}
	// A scope change during paging must also remove any earlier collected hosts.
	scope, err := reauthorizeLogDownload(c)
	if err != nil {
		common.FailWithCode(c, 403, "permission.denied")
		return
	}
	visible, _, err := reads.taskReadHostIDs(id, scope)
	if err != nil {
		failDeployTaskError(c, err, "deploytask.logs_failed")
		return
	}
	allowed := make(map[uint64]bool, len(visible))
	for _, hostID := range visible {
		allowed[hostID] = true
	}
	retained := items[:0]
	for _, item := range items {
		if allowed[item.TaskHostID] {
			retained = append(retained, item)
		}
	}
	common.SetAuditResult(c, "{\"taskId\":"+strconv.FormatUint(id, 10)+",\"rows\":"+strconv.Itoa(len(retained))+",\"truncated\":"+strconv.FormatBool(limited)+"}")
	c.Header("Cache-Control", "no-store")
	common.Success(c, map[string]any{"items": retained, "truncated": limited, "redaction": "known-task-secrets", "rowLimit": deployLogDownloadRows, "byteLimit": deployLogDownloadBytes})
}
