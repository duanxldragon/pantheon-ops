package argocd

import (
	"context"
	"crypto/tls"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/pkg/security/ssrf"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestArgoCDClient creates a client with relaxed SSRF validation for testing. Server
// certificate verification is never disabled: pass the test server's own CA PEM as caCertPEM for
// TLS servers, or an empty bundle to exercise the system trust store path.
func newTestArgoCDClient(endpoint, authToken, caCertPEM string) (*ArgoCDClient, error) {
	validator := ssrf.DefaultValidator()
	validator.AllowedPorts = []int{}      // Allow all ports for testing
	validator.BlockPrivateIPs = false     // Allow localhost for testing
	validator.DNSRebindProtection = false // Disable DNS rebind check for testing

	if err := validator.ValidateURL(endpoint); err != nil {
		return nil, err
	}

	httpClient := validator.CreateHTTPClient(30 * time.Second)
	tlsConfig, err := serverTLSConfig(caCertPEM)
	if err != nil {
		return nil, err
	}
	if tlsConfig != nil {
		if transport, ok := httpClient.Transport.(*http.Transport); ok {
			transport.TLSClientConfig = tlsConfig
		}
	}

	return &ArgoCDClient{
		endpoint:      endpoint,
		authToken:     authToken,
		httpClient:    httpClient,
		ssrfValidator: validator,
	}, nil
}

// caPEMFrom returns the test server's certificate in PEM form so it can be stored as a server
// record CA bundle.
func caPEMFrom(t *testing.T, server *httptest.Server) string {
	t.Helper()
	return string(pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: server.Certificate().Raw,
	}))
}

func TestArgoCDClient_TestConnection(t *testing.T) {
	tests := []struct {
		name       string
		statusCode int
		wantErr    bool
	}{
		{
			name:       "successful connection",
			statusCode: http.StatusOK,
			wantErr:    false,
		},
		{
			name:       "unauthorized",
			statusCode: http.StatusUnauthorized,
			wantErr:    true,
		},
		{
			name:       "server error",
			statusCode: http.StatusInternalServerError,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, "/api/version", r.URL.Path)
				assert.Contains(t, r.Header.Get("Authorization"), "Bearer ")
				w.WriteHeader(tt.statusCode)
				if tt.statusCode == http.StatusOK {
					w.Write([]byte(`{"Version":"v2.8.0"}`))
				}
			}))
			defer server.Close()

			client, err := newTestArgoCDClient(server.URL, "test-token", "")
			require.NoError(t, err)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			err = client.TestConnection(ctx)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestArgoCDClient_GetVersion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/version", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"Version":"v2.8.0+abc123"}`))
	}))
	defer server.Close()

	client, err := newTestArgoCDClient(server.URL, "test-token", "")
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	version, err := client.GetVersion(ctx)
	require.NoError(t, err)
	assert.Equal(t, "v2.8.0+abc123", version)
}

func TestArgoCDClient_GetApplicationStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/applications/test-app", r.URL.Path)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"status": {
				"health": {"status": "Healthy"},
				"sync": {"status": "Synced", "revision": "abc123"}
			}
		}`))
	}))
	defer server.Close()

	client, err := newTestArgoCDClient(server.URL, "test-token", "")
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	status, err := client.GetApplicationStatus(ctx, "test-app")
	require.NoError(t, err)
	assert.Equal(t, "Healthy", status.Health.Status)
	assert.Equal(t, "Synced", status.Sync.Status)
	assert.Equal(t, "abc123", status.Sync.Revision)
}

func TestArgoCDClient_SyncApplication(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/applications/test-app/sync", r.URL.Path)
		assert.Equal(t, http.MethodPost, r.Method)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := newTestArgoCDClient(server.URL, "test-token", "")
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = client.SyncApplication(ctx, "test-app", "main")
	assert.NoError(t, err)
}

func TestArgoCDClient_RefreshApplication(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/applications/test-app", r.URL.Path)
		assert.Equal(t, "normal", r.URL.Query().Get("refresh"))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := newTestArgoCDClient(server.URL, "test-token", "")
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = client.RefreshApplication(ctx, "test-app")
	assert.NoError(t, err)
}

func TestArgoCDClient_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second) // Simulate slow server
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := newTestArgoCDClient(server.URL, "test-token", "")
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err = client.TestConnection(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context deadline exceeded")
}

func TestServerTLSConfigKeepsSystemTrustWhenCAIsEmpty(t *testing.T) {
	for _, bundle := range []string{"", "  \n "} {
		config, err := serverTLSConfig(bundle)
		require.NoError(t, err)
		assert.Nil(t, config, "an empty CA bundle must leave the standard verification path untouched")
	}
}

func TestServerTLSConfigAddsStoredCAAndKeepsVerificationOn(t *testing.T) {
	server := httptest.NewTLSServer(http.NotFoundHandler())
	defer server.Close()

	config, err := serverTLSConfig(caPEMFrom(t, server))
	require.NoError(t, err)
	require.NotNil(t, config)
	assert.False(t, config.InsecureSkipVerify)
	assert.NotNil(t, config.RootCAs)
	assert.Equal(t, uint16(tls.VersionTLS12), config.MinVersion)
}

func TestServerTLSConfigRejectsBundleWithoutCertificate(t *testing.T) {
	_, err := serverTLSConfig("-----BEGIN CERTIFICATE-----\nnot-base64\n-----END CERTIFICATE-----")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidCACert)
}

func TestArgoCDClient_VerifiesServerCertificate(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"Version":"v2.8.0"}`))
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	untrusted, err := newTestArgoCDClient(server.URL, "test-token", "")
	require.NoError(t, err)
	err = untrusted.TestConnection(ctx)
	require.Error(t, err, "a certificate the platform does not trust must be refused")
	assert.Contains(t, err.Error(), "certificate")

	trusted, err := newTestArgoCDClient(server.URL, "test-token", caPEMFrom(t, server))
	require.NoError(t, err)
	assert.NoError(t, trusted.TestConnection(ctx))
}
