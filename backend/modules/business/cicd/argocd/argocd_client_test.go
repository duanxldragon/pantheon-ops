package argocd

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

			client := NewArgoCDClient(server.URL, "test-token", true)
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			err := client.TestConnection(ctx)
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

	client := NewArgoCDClient(server.URL, "test-token", true)
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

	client := NewArgoCDClient(server.URL, "test-token", true)
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

	client := NewArgoCDClient(server.URL, "test-token", true)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := client.SyncApplication(ctx, "test-app", "main")
	assert.NoError(t, err)
}

func TestArgoCDClient_RefreshApplication(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/applications/test-app", r.URL.Path)
		assert.Equal(t, "normal", r.URL.Query().Get("refresh"))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewArgoCDClient(server.URL, "test-token", true)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := client.RefreshApplication(ctx, "test-app")
	assert.NoError(t, err)
}

func TestArgoCDClient_Timeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second) // Simulate slow server
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewArgoCDClient(server.URL, "test-token", true)
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := client.TestConnection(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context deadline exceeded")
}
