package argocd

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/pkg/common"
	"github.com/duanxldragon/pantheon-base/backend/pkg/security/ssrf"
)

// ErrInvalidCACert rejects a stored CA bundle that contains no parseable certificate, so a
// mistyped bundle fails at save time instead of turning every request into a TLS error.
var ErrInvalidCACert = errors.New("ca_cert_contains_no_valid_certificate")

// authorizationBearerPrefix is the RFC 6750 scheme Argo CD expects in front of the API token.
const authorizationBearerPrefix = "Bearer "

// Provider response bodies can contain credentials or rendered manifests.
// Keep only the HTTP status so callers can distinguish rejection from an
// uncertain transport/server outcome without persisting private payloads.
type providerStatusError struct {
	action string
	status int
}

func (e *providerStatusError) Error() string {
	return fmt.Sprintf("%s failed with status %d", e.action, e.status)
}

func (e *providerStatusError) Unwrap() error {
	if e.action == "submit operation" {
		switch e.status {
		case http.StatusBadRequest, http.StatusConflict, http.StatusPreconditionFailed, http.StatusUnprocessableEntity:
			return common.ErrConflict
		}
	}
	return nil
}

// ArgoCDClient wraps HTTP client for ArgoCD API communication.
//
//nolint:revive // the exported client name keeps the Argo CD domain prefix for API clarity.
type ArgoCDClient struct {
	endpoint      string
	authToken     string
	httpClient    *http.Client
	ssrfValidator *ssrf.Validator
}

// NewArgoCDClient creates a new ArgoCD API client.
//
// Server certificates are always verified: against the system trust store, or against the
// server record's own CA bundle when caCertPEM is set. Self-hosted Argo CD endpoints that ship
// a private certificate authority are supported by storing that CA on the server record; there
// is no mode that disables verification, because every caller passed one unconditionally before
// this and the endpoint then accepted any certificate.
func NewArgoCDClient(endpoint, authToken, caCertPEM string) (*ArgoCDClient, error) {
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, errors.New("cicd.argocd_invalid_endpoint")
	}
	endpoint = strings.TrimRight(endpoint, "/")
	validator := ssrf.DefaultValidator()
	validator.AllowedPorts = []int{80, 443, 8080}

	if err := validator.ValidateURL(endpoint); err != nil {
		return nil, errors.New("cicd.argocd_invalid_endpoint")
	}

	httpClient := validator.CreateHTTPClient(30 * time.Second)
	tlsConfig, err := serverTLSConfig(caCertPEM)
	if err != nil {
		return nil, err
	}
	if tlsConfig != nil {
		if t, ok := httpClient.Transport.(*http.Transport); ok {
			t.TLSClientConfig = tlsConfig
		}
	}

	return &ArgoCDClient{
		endpoint:      endpoint,
		authToken:     authToken,
		httpClient:    httpClient,
		ssrfValidator: validator,
	}, nil
}

// serverTLSConfig returns the transport TLS settings for a server record. An empty CA bundle
// means "use the system trust store", expressed as a nil config so the standard verification
// path stays intact; a bundle produces a pool of system roots plus that CA.
func serverTLSConfig(caCertPEM string) (*tls.Config, error) {
	bundle := strings.TrimSpace(caCertPEM)
	if bundle == "" {
		return nil, nil
	}
	pool, err := x509.SystemCertPool()
	if err != nil || pool == nil {
		pool = x509.NewCertPool()
	}
	if !pool.AppendCertsFromPEM([]byte(bundle)) {
		return nil, fmt.Errorf("cicd.argocd_invalid_ca_cert: %w", ErrInvalidCACert)
	}
	return &tls.Config{RootCAs: pool, MinVersion: tls.VersionTLS12}, nil
}

// TestConnection tests connectivity to ArgoCD server.
func (c *ArgoCDClient) TestConnection(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint+"/api/version", nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", authorizationBearerPrefix+c.authToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return &providerStatusError{action: "connection", status: resp.StatusCode}
	}

	return nil
}

// GetVersion retrieves ArgoCD server version.
func (c *ArgoCDClient) GetVersion(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint+"/api/version", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", authorizationBearerPrefix+c.authToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	var result struct {
		Version string `json:"Version"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	return result.Version, nil
}

// ArgoCDApplicationStatus represents the status of an ArgoCD application.
//
//nolint:revive // the exported DTO name keeps the Argo CD domain prefix for API clarity.
type ArgoCDApplicationStatus struct {
	ApplicationUID string `json:"-"`
	History        []struct {
		ID       int64             `json:"id"`
		Revision string            `json:"revision"`
		Source   json.RawMessage   `json:"source"`
		Sources  []json.RawMessage `json:"sources"`
	} `json:"history"`
	Health struct {
		Status string `json:"status"`
	} `json:"health"`
	Sync struct {
		Status   string `json:"status"`
		Revision string `json:"revision"`
	} `json:"sync"`
	OperationState *struct {
		Phase      string         `json:"phase"`
		Message    string         `json:"message"`
		StartedAt  *time.Time     `json:"startedAt"`
		FinishedAt *time.Time     `json:"finishedAt"`
		Operation  *argoOperation `json:"operation"`
		SyncResult struct {
			Revision string `json:"revision"`
		} `json:"syncResult"`
	} `json:"operationState,omitempty"`
}

// GetApplicationStatus retrieves the current status of an application.
func (c *ArgoCDClient) GetApplicationStatus(ctx context.Context, appName string) (*ArgoCDApplicationStatus, error) {
	var result argoApplication
	if err := c.readProvider(ctx, "/api/v1/applications/"+url.PathEscape(appName), &result); err != nil {
		return nil, err
	}
	result.Status.ApplicationUID = result.Metadata.UID
	return &result.Status, nil
}

type argoOperation struct {
	Info []argoOperationInfo `json:"info"`
	Sync struct {
		Revision string          `json:"revision"`
		Prune    bool            `json:"prune"`
		Source   json.RawMessage `json:"source,omitempty"`
	} `json:"sync"`
}
type argoOperationInfo struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}
type argoApplication struct {
	Metadata struct {
		UID             string `json:"uid"`
		ResourceVersion string `json:"resourceVersion"`
		Namespace       string `json:"namespace"`
	} `json:"metadata"`
	Spec      json.RawMessage         `json:"spec"`
	Operation *argoOperation          `json:"operation"`
	Status    ArgoCDApplicationStatus `json:"status"`
}

const providerReadLimit = 4 << 20

func (c *ArgoCDClient) readProvider(ctx context.Context, path string, result interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint+path, nil)
	if err != nil {
		return errors.New("cicd.argocd.app_sync_failed")
	}
	req.Header.Set("Authorization", authorizationBearerPrefix+c.authToken)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		return errors.New("cicd.argocd.app_sync_failed")
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return &providerStatusError{action: "read application", status: resp.StatusCode}
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, providerReadLimit+1))
	if err != nil || len(data) > providerReadLimit || json.Unmarshal(data, result) != nil {
		return errors.New("cicd.argocd.app_sync_failed")
	}
	return nil
}

func (c *ArgoCDClient) readApplication(ctx context.Context, name string) (*argoApplication, error) {
	var app argoApplication
	if err := c.readProvider(ctx, "/api/v1/applications/"+url.PathEscape(name), &app); err != nil {
		return nil, err
	}
	if app.Metadata.UID == "" || app.Metadata.ResourceVersion == "" || len(app.Spec) == 0 {
		return nil, errors.New("cicd.argocd.app_sync_failed")
	}
	app.Status.ApplicationUID = app.Metadata.UID
	return &app, nil
}

// submitReviewedOperation uses only the independently reviewed native route.
// The caller negotiates capability before reservation and again before dispatch.
// An unconfirmed reply stays Unknown; there is no PATCH/native/credential fallback.
func (c *ArgoCDClient) submitReviewedOperation(ctx context.Context, name string, app *argoApplication, operation *argoOperation, historyID *int64) error {
	fields, err := reviewedFields(app, operation)
	if err != nil || (historyID != nil && *historyID < 0) {
		return common.NewConflict(msgParamInvalid)
	}
	var spec struct {
		Project string `json:"project"`
	}
	if json.Unmarshal(app.Spec, &spec) != nil {
		return common.NewConflict(msgParamInvalid)
	}
	payload := map[string]interface{}{"name": name, "appNamespace": app.Metadata.Namespace, "project": spec.Project, "prune": operation.Sync.Prune, "reviewed": fields}
	action := "sync"
	if historyID == nil {
		payload["revision"] = fields.ResolvedRevision
	} else {
		action = "rollback"
		payload["id"] = *historyID
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return common.NewConflict(msgParamInvalid)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint+"/api/v1/applications/"+url.PathEscape(name)+"/reviewed-actions/"+action, bytes.NewReader(body))
	if err != nil {
		return errors.New("cicd.argocd.app_sync_failed")
	}
	req.Header.Set("Authorization", authorizationBearerPrefix+c.authToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return &providerStatusError{action: "submit operation", status: resp.StatusCode}
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, providerReadLimit+1))
	var accepted argoApplication
	if err != nil || len(data) > providerReadLimit || json.Unmarshal(data, &accepted) != nil ||
		accepted.Metadata.UID != app.Metadata.UID || accepted.Metadata.ResourceVersion == "" ||
		accepted.Metadata.ResourceVersion == app.Metadata.ResourceVersion || !sameProviderJSON(accepted.Spec, app.Spec) {
		return errors.New("cicd.argocd.app_sync_failed")
	}
	if !sameProviderOperation(accepted.Operation, operation) {
		state := accepted.Status.OperationState
		if state == nil || !sameProviderOperation(state.Operation, operation) {
			return errors.New("cicd.argocd.app_sync_failed")
		}
		switch state.Phase {
		case "Running", "Succeeded", "Failed", "Error", "Terminating":
		default:
			return errors.New("cicd.argocd.app_sync_failed")
		}
	}
	return nil
}

func sameProviderJSON(a, b json.RawMessage) bool {
	if len(a) == 0 || len(b) == 0 {
		return len(a) == 0 && len(b) == 0
	}
	var x, y interface{}
	dx, dy := json.NewDecoder(bytes.NewReader(a)), json.NewDecoder(bytes.NewReader(b))
	dx.UseNumber()
	dy.UseNumber()
	return dx.Decode(&x) == nil && dy.Decode(&y) == nil && reflect.DeepEqual(x, y)
}

func sameProviderOperation(a, b *argoOperation) bool {
	return a != nil && b != nil && a.Sync.Revision == b.Sync.Revision && a.Sync.Prune == b.Sync.Prune &&
		reflect.DeepEqual(a.Info, b.Info) && sameProviderJSON(a.Sync.Source, b.Sync.Source)
}

// SyncApplication triggers a sync operation for an application.
func (c *ArgoCDClient) SyncApplication(ctx context.Context, appName, revision string) error {
	url := fmt.Sprintf("%s/api/v1/applications/%s/sync", c.endpoint, url.PathEscape(appName))

	body := map[string]interface{}{
		"revision": revision,
		"prune":    false,
		"dryRun":   false,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", authorizationBearerPrefix+c.authToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return &providerStatusError{action: "sync", status: resp.StatusCode}
	}

	return nil
}

// RefreshApplication triggers a refresh of application manifests.
func (c *ArgoCDClient) RefreshApplication(ctx context.Context, appName string) error {
	url := fmt.Sprintf("%s/api/v1/applications/%s", c.endpoint, url.PathEscape(appName))

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url+"?refresh=normal", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", authorizationBearerPrefix+c.authToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("refresh failed with status %d", resp.StatusCode)
	}

	return nil
}

// CreateApplication creates a new ArgoCD application.
func (c *ArgoCDClient) CreateApplication(ctx context.Context, req *CreateApplicationPayload) error {
	url := c.endpoint + "/api/v1/applications"

	jsonBody, err := json.Marshal(req)
	if err != nil {
		return err
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(jsonBody))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Authorization", authorizationBearerPrefix+c.authToken)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		return &providerStatusError{action: "create application", status: resp.StatusCode}
	}

	return nil
}

// CreateApplicationPayload represents the payload for creating an ArgoCD application.
type CreateApplicationPayload struct {
	Metadata applicationObjectMeta `json:"metadata"`
	Spec     applicationSpec       `json:"spec"`
}

// applicationObjectMeta is the Kubernetes-style object metadata Argo CD reads on create.
type applicationObjectMeta struct {
	Name string `json:"name"`
}

// applicationSpec is the Argo CD application spec: which project owns the app and where its
// manifests come from and land.
type applicationSpec struct {
	Project     string                 `json:"project"`
	Source      applicationSource      `json:"source"`
	Destination applicationDestination `json:"destination"`
}

// applicationSource points the application at a path inside a Git repository.
type applicationSource struct {
	RepoURL        string `json:"repoURL"`
	Path           string `json:"path"`
	TargetRevision string `json:"targetRevision"`
}

// applicationDestination points the application at a cluster namespace.
type applicationDestination struct {
	Server    string `json:"server"`
	Namespace string `json:"namespace"`
}

// DeleteApplication deletes an ArgoCD application.
func (c *ArgoCDClient) DeleteApplication(ctx context.Context, appName string, cascade bool) error {
	url := fmt.Sprintf("%s/api/v1/applications/%s?cascade=%t", c.endpoint, url.PathEscape(appName), cascade)

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", authorizationBearerPrefix+c.authToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		return &providerStatusError{action: "delete application", status: resp.StatusCode}
	}

	return nil
}

// RollbackApplication restores a provider-owned deployment history ID, including
// its historical source configuration rather than re-syncing current settings.
func (c *ArgoCDClient) RollbackApplication(ctx context.Context, appName string, historyID int64) error {
	if historyID < 0 {
		return errors.New("cicd.argocd.app_rollback_failed")
	}
	body, err := json.Marshal(map[string]interface{}{"id": historyID, "prune": false})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint+"/api/v1/applications/"+url.PathEscape(appName)+"/rollback", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", authorizationBearerPrefix+c.authToken)
	req.Header.Set("Content-Type", "application/json")
	response, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode != http.StatusOK {
		return &providerStatusError{action: "rollback", status: response.StatusCode}
	}
	return nil
}
