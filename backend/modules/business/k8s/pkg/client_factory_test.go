package pkg

import (
	"context"
	"net/http"
	"strings"
	"testing"
)

const publicTestKubeconfig = `apiVersion: v1
kind: Config
clusters:
- name: test
  cluster:
    server: https://8.8.8.8:6443
contexts:
- name: test
  context:
    cluster: test
    user: test
current-context: test
users:
- name: test
  user:
    token: fixture
`

func TestKubeconfigUsesCheckedDial(t *testing.T) {
	config, err := RESTConfigFromKubeconfig(publicTestKubeconfig)
	if err != nil {
		t.Fatal(err)
	}
	if config.Dial == nil {
		t.Fatal("Kubernetes requests have no checked dial boundary")
	}
	for _, addr := range []string{"127.0.0.1:6443", "169.254.169.254:6443", "[::1]:6443"} {
		conn, err := config.Dial(context.Background(), "tcp", addr)
		if conn != nil {
			conn.Close()
		}
		if err == nil {
			t.Fatalf("checked dial accepted private target %s", addr)
		}
	}
}

type requestTrap struct{ called bool }

func (t *requestTrap) RoundTrip(*http.Request) (*http.Response, error) {
	t.called = true
	return &http.Response{StatusCode: 200, Body: http.NoBody}, nil
}

func TestKubeconfigRejectsProxyAndPrivateRedirect(t *testing.T) {
	proxyConfig := strings.Replace(publicTestKubeconfig, "server: https://8.8.8.8:6443", "server: https://8.8.8.8:6443\n    proxy-url: http://8.8.4.4:8080", 1)
	if _, err := RESTConfigFromKubeconfig(proxyConfig); err == nil {
		t.Fatal("proxy bypass accepted")
	}
	config, err := RESTConfigFromKubeconfig(publicTestKubeconfig)
	if err != nil {
		t.Fatal(err)
	}
	trap := &requestTrap{}
	guarded := config.WrapTransport(trap)
	request, err := http.NewRequest(http.MethodGet, "http://127.0.0.1/metadata", nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := guarded.RoundTrip(request); err == nil || trap.called {
		t.Fatal("private redirected request reached transport")
	}
	request, _ = http.NewRequest(http.MethodGet, "https://8.8.8.8:6443/api", nil)
	if _, err := guarded.RoundTrip(request); err != nil || !trap.called {
		t.Fatalf("public request rejected: %v", err)
	}
	if config.TLSClientConfig.Insecure {
		t.Fatal("TLS certificate verification disabled")
	}
}

func TestKubeconfigRejectsLocalCredentialSourcesAndInsecureTLS(t *testing.T) {
	for _, payload := range []string{
		strings.Replace(publicTestKubeconfig, "token: fixture", "tokenFile: /local/secret", 1),
		strings.Replace(publicTestKubeconfig, "token: fixture", "exec:\n      apiVersion: client.authentication.k8s.io/v1beta1\n      command: local-command", 1),
		strings.Replace(publicTestKubeconfig, "server: https://8.8.8.8:6443", "server: https://8.8.8.8:6443\n    insecure-skip-tls-verify: true", 1),
	} {
		if _, err := RESTConfigFromKubeconfig(payload); err == nil {
			t.Fatal("unsafe kubeconfig accepted")
		}
	}
}

func TestKubeconfigDoesNotForwardCredentialsToPublicRedirectOrigin(t *testing.T) {
	config, err := RESTConfigFromKubeconfig(publicTestKubeconfig)
	if err != nil {
		t.Fatal(err)
	}
	trap := &requestTrap{}
	client := &http.Client{Transport: config.WrapTransport(trap)}
	// Exercise the same per-request transport guard the redirect handler uses.
	request, _ := http.NewRequest(http.MethodGet, "https://8.8.4.4:6443/api", nil)
	request.Header.Set("Authorization", "Bearer fixture")
	if _, err := client.Do(request); err == nil || trap.called {
		t.Fatal("credentials reached a different public origin")
	}
}
