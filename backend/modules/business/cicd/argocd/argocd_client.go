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
	"strings"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/pkg/security/ssrf"
)

// ErrInvalidCACert rejects a stored CA bundle that contains no parseable certificate, so a
// mistyped bundle fails at save time instead of turning every request into a TLS error.
var ErrInvalidCACert = errors.New("ca_cert_contains_no_valid_certificate")

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
	validator := ssrf.DefaultValidator()
	validator.AllowedPorts = []int{80, 443, 8080}

	if err := validator.ValidateURL(endpoint); err != nil {
		return nil, fmt.Errorf("cicd.argocd_invalid_endpoint: %w", err)
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
	req.Header.Set("Authorization", "Bearer "+c.authToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// GetVersion retrieves ArgoCD server version.
func (c *ArgoCDClient) GetVersion(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.endpoint+"/api/version", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.authToken)

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
	Health struct {
		Status string `json:"status"`
	} `json:"health"`
	Sync struct {
		Status   string `json:"status"`
		Revision string `json:"revision"`
	} `json:"sync"`
	OperationState *struct {
		Phase     string     `json:"phase"`
		Message   string     `json:"message"`
		StartedAt *time.Time `json:"startedAt"`
	} `json:"operationState,omitempty"`
}

// GetApplicationStatus retrieves the current status of an application.
func (c *ArgoCDClient) GetApplicationStatus(ctx context.Context, appName string) (*ArgoCDApplicationStatus, error) {
	url := fmt.Sprintf("%s/api/v1/applications/%s", c.endpoint, appName)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.authToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	var result struct {
		Status ArgoCDApplicationStatus `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result.Status, nil
}

// SyncApplication triggers a sync operation for an application.
func (c *ArgoCDClient) SyncApplication(ctx context.Context, appName, revision string) error {
	url := fmt.Sprintf("%s/api/v1/applications/%s/sync", c.endpoint, appName)

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
	req.Header.Set("Authorization", "Bearer "+c.authToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("sync failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// RefreshApplication triggers a refresh of application manifests.
func (c *ArgoCDClient) RefreshApplication(ctx context.Context, appName string) error {
	url := fmt.Sprintf("%s/api/v1/applications/%s", c.endpoint, appName)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url+"?refresh=normal", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.authToken)

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
	httpReq.Header.Set("Authorization", "Bearer "+c.authToken)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("create application failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// CreateApplicationPayload represents the payload for creating an ArgoCD application.
type CreateApplicationPayload struct {
	Metadata struct {
		Name string `json:"name"`
	} `json:"metadata"`
	Spec struct {
		Project string `json:"project"`
		Source  struct {
			RepoURL        string `json:"repoURL"`
			Path           string `json:"path"`
			TargetRevision string `json:"targetRevision"`
		} `json:"source"`
		Destination struct {
			Server    string `json:"server"`
			Namespace string `json:"namespace"`
		} `json:"destination"`
	} `json:"spec"`
}

// DeleteApplication deletes an ArgoCD application.
func (c *ArgoCDClient) DeleteApplication(ctx context.Context, appName string, cascade bool) error {
	url := fmt.Sprintf("%s/api/v1/applications/%s?cascade=%t", c.endpoint, appName, cascade)

	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+c.authToken)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete application failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
