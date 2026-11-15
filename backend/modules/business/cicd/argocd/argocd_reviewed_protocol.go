package argocd

import (
	"bytes"
	"context"
	"encoding/json"
	"net/url"
	"regexp"

	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
)

const reviewedActionsProtocol = "pantheon-reviewed-actions/v1"

type reviewedActionsCapabilities struct {
	Protocol           string `json:"protocol"`
	RevisionScope      string `json:"revisionScope"`
	MarkerName         string `json:"markerName"`
	ConditionalUpdate  bool   `json:"conditionalUpdate"`
	NativePolicy       bool   `json:"nativePolicy"`
	CorrelatedRollback bool   `json:"correlatedRollback"`
}

var requiredReviewedCapabilities = reviewedActionsCapabilities{reviewedActionsProtocol, "single-git-sha1", actionInfoName, true, true, true}
var reviewedRevision = regexp.MustCompile(`^[a-f0-9]{40}$`)
var reviewedMarker = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,128}$`)

type reviewedActionFields struct {
	ExpectedUID             string `json:"expectedUID"`
	ExpectedResourceVersion string `json:"expectedResourceVersion"`
	ResolvedRevision        string `json:"resolvedRevision"`
	ActionMarker            string `json:"actionMarker"`
}

func (c *ArgoCDClient) requireReviewedCapabilities(ctx context.Context, name string, app *argoApplication) error {
	var spec struct {
		Project string `json:"project"`
	}
	if app == nil || json.Unmarshal(app.Spec, &spec) != nil {
		return common.NewConflict(msgParamInvalid)
	}
	query := url.Values{"appNamespace": {app.Metadata.Namespace}, "project": {spec.Project}}
	var raw json.RawMessage
	if err := c.readProvider(ctx, "/api/v1/applications/"+url.PathEscape(name)+"/reviewed-actions/capabilities?"+query.Encode(), &raw); err != nil {
		return err
	}
	if !validReviewedCapabilities(raw) {
		return common.NewConflict(msgParamInvalid)
	}
	return nil
}

func validReviewedCapabilities(raw json.RawMessage) bool {
	// Ambiguous duplicate fields and unknown protocol extensions are not a
	// negotiation result. readProvider already rejects trailing JSON content.
	decoder := json.NewDecoder(bytes.NewReader(raw))
	first, err := decoder.Token()
	if err != nil || first != json.Delim('{') {
		return false
	}
	fields := map[string]json.RawMessage{}
	for decoder.More() {
		key, err := decoder.Token()
		if err != nil {
			return false
		}
		name, ok := key.(string)
		if !ok {
			return false
		}
		if _, duplicate := fields[name]; duplicate {
			return false
		}
		var value json.RawMessage
		if decoder.Decode(&value) != nil {
			return false
		}
		fields[name] = value
	}
	if _, err := decoder.Token(); err != nil {
		return false
	}
	data, err := json.Marshal(fields)
	if err != nil {
		return false
	}
	var capabilities reviewedActionsCapabilities
	strict := json.NewDecoder(bytes.NewReader(data))
	strict.DisallowUnknownFields()
	return len(fields) == 6 && strict.Decode(&capabilities) == nil && capabilities == requiredReviewedCapabilities
}

func boundedProviderIdentity(value string) bool {
	if len(value) == 0 || len(value) > 128 {
		return false
	}
	for _, b := range []byte(value) {
		if b < 32 || b > 126 {
			return false
		}
	}
	return true
}

func reviewedFields(app *argoApplication, operation *argoOperation) (reviewedActionFields, error) {
	if app == nil || operation == nil || !boundedProviderIdentity(app.Metadata.UID) || !boundedProviderIdentity(app.Metadata.ResourceVersion) || !reviewedRevision.MatchString(operation.Sync.Revision) || len(operation.Info) != 1 || operation.Info[0].Name != actionInfoName || !reviewedMarker.MatchString(operation.Info[0].Value) {
		return reviewedActionFields{}, common.NewConflict(msgParamInvalid)
	}
	return reviewedActionFields{app.Metadata.UID, app.Metadata.ResourceVersion, operation.Sync.Revision, operation.Info[0].Value}, nil
}
