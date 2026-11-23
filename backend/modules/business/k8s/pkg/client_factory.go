package pkg

import (
	"errors"
	"fmt"

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
	if kubeconfig == "" {
		return nil, errKubeconfigInvalid
	}
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

// validateEmbeddedCredentials rejects kubeconfig credential sources that run
// commands or read local files on the server. client-go would otherwise
// execute an exec/auth-provider plugin or read a TokenFile/certificate path
// supplied by the uploader when the first request is made, so the upload
// boundary only accepts fully embedded material.
func validateEmbeddedCredentials(cfg *clientcmdapi.Config) error {
	for name, auth := range cfg.AuthInfos {
		if auth == nil {
			continue
		}
		if auth.Exec != nil || auth.AuthProvider != nil || auth.TokenFile != "" || auth.ClientCertificate != "" || auth.ClientKey != "" {
			return fmt.Errorf("%w: user %q requires exec/auth-provider or local credential files", errKubeconfigInvalid, name)
		}
	}
	for name, cl := range cfg.Clusters {
		if cl == nil {
			continue
		}
		if cl.CertificateAuthority != "" {
			return fmt.Errorf("%w: cluster %q requires a local CA file", errKubeconfigInvalid, name)
		}
	}
	return nil
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
	cfg, err := clientcmd.Load([]byte(kubeconfig))
	if err != nil {
		return nil, errKubeconfigInvalid
	}
	if err := validateEmbeddedCredentials(cfg); err != nil {
		return nil, err
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

	// Client-side throttle: without it a burst of console requests saturates
	// one cluster's API server; the values are conservative console defaults.
	restConfig.QPS = 20
	restConfig.Burst = 40

	return restConfig, nil
}
