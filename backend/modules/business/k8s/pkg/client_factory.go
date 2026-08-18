package pkg

import (
	"errors"

	"k8s.io/client-go/kubernetes"
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
	restConfig, err := clientcmd.RESTConfigFromKubeConfig([]byte(kubeconfig))
	if err != nil {
		return nil, errKubeconfigInvalid
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
