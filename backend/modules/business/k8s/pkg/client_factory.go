package pkg

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/duanxldragon/pantheon-base/backend/pkg/security/ssrf"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

var errKubeconfigInvalid = errors.New("k8s.kubeconfig_invalid")

// NewClientFromKubeconfig builds a kubernetes Clientset from a raw kubeconfig
// payload. The clientset is created lazily; no connection is established until
// the first request.
func NewClientFromKubeconfig(kubeconfig string) (*kubernetes.Clientset, error) {
	restConfig, err := RESTConfigFromKubeconfig(kubeconfig)
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, errKubeconfigInvalid
	}
	return clientset, nil
}

// NewClientFromEncrypted decrypts an encrypted kubeconfig and returns a
// Clientset for it. It combines the encryptor and client factory into the
// single call used by cluster services.
func NewClientFromEncrypted(encrypted string) (*kubernetes.Clientset, error) {
	kubeconfig, err := DecryptKubeconfig(encrypted)
	if err != nil {
		return nil, err
	}
	return NewClientFromKubeconfig(kubeconfig)
}

// ExtractAPIServer derives the API server URL from a kubeconfig. It returns an
// empty string when the config is malformed or the current context cannot be
// resolved.
func ExtractAPIServer(kubeconfig string) string {
	cfg, err := clientcmd.Load([]byte(kubeconfig))
	if err != nil {
		return ""
	}
	if cfg.CurrentContext == "" {
		return firstClusterServer(cfg)
	}
	ctx, ok := cfg.Contexts[cfg.CurrentContext]
	if !ok {
		return ""
	}
	cl, ok := cfg.Clusters[ctx.Cluster]
	if !ok {
		return ""
	}
	return cl.Server
}

func firstClusterServer(cfg *clientcmdapi.Config) string {
	for _, cl := range cfg.Clusters {
		return cl.Server
	}
	return ""
}

// RESTConfigFromKubeconfig creates a REST config from kubeconfig string.
// This is used for creating dynamic clients for CRD access (e.g., Velero).
func RESTConfigFromKubeconfig(kubeconfig string) (*rest.Config, error) {
	if kubeconfig == "" {
		return nil, errKubeconfigInvalid
	}
	restConfig, err := clientcmd.RESTConfigFromKubeConfig([]byte(kubeconfig))
	if err != nil {
		return nil, errKubeconfigInvalid
	}

	// Validate API server URL against SSRF attacks
	validator := ssrf.DefaultValidator()
	validator.AllowedPorts = []int{443, 6443, 8443, 16443}
	if err := validator.ValidateURL(restConfig.Host); err != nil {
		return nil, fmt.Errorf("k8s.apiserver_invalid_url: %w", err)
	}

	// A kubeconfig proxy can route through another network and bypass the checked
	// target dial. Uploaded kubeconfigs must also never invoke local credential
	// plugins, read local secret files or disable server certificate verification.
	// This consumer supports direct API connections with inline credentials only.
	if restConfig.Proxy != nil || restConfig.ExecProvider != nil || restConfig.AuthProvider != nil ||
		restConfig.BearerTokenFile != "" || restConfig.TLSClientConfig.CAFile != "" ||
		restConfig.TLSClientConfig.CertFile != "" || restConfig.TLSClientConfig.KeyFile != "" || restConfig.TLSClientConfig.Insecure {
		return nil, errKubeconfigInvalid
	}
	restConfig.Timeout = 30 * time.Second
	checkedClient := validator.CreateHTTPClient(restConfig.Timeout)
	restConfig.Dial = checkedClient.Transport.(*http.Transport).DialContext
	restConfig.Proxy = func(*http.Request) (*url.URL, error) { return nil, nil }
	restConfig.WrapTransport = func(transport http.RoundTripper) http.RoundTripper {
		return &checkedKubernetesTransport{validator: validator, transport: transport, origin: restConfig.Host}
	}

	return restConfig, nil
}

// Validate each request as well as each dial, including redirected requests.
type checkedKubernetesTransport struct {
	validator *ssrf.Validator
	transport http.RoundTripper
	origin    string
}

func (t *checkedKubernetesTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	origin, err := url.Parse(t.origin)
	if err != nil || request.URL.Scheme != origin.Scheme || request.URL.Host != origin.Host {
		return nil, errKubeconfigInvalid
	}
	if err := t.validator.ValidateURL(request.URL.String()); err != nil {
		return nil, err
	}
	return t.transport.RoundTrip(request)
}
