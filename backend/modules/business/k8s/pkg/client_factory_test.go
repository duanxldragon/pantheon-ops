package pkg

import (
	"errors"
	"strings"
	"testing"
)

func factoryKubeconfig(userBlock string) string {
	return `apiVersion: v1
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
` + userBlock
}

// TestKubeconfigRejectsLocalExecutionAndFileReads proves the upload boundary
// only accepts embedded credential material: client-go would execute an exec
// plugin or read a local TokenFile/certificate path on the server when the
// first request is made.
func TestKubeconfigRejectsLocalExecutionAndFileReads(t *testing.T) {
	cases := map[string]string{
		"exec plugin":        "    exec:\n      command: /tmp/evil.sh\n      apiVersion: client.authentication.k8s.io/v1\n      interactiveMode: Never\n",
		"auth provider":      "    auth-provider:\n      name: oidc\n",
		"token file":         "    tokenFile: /etc/passwd\n",
		"client certificate": "    client-certificate: /etc/ssl/client.crt\n",
		"client key":         "    client-key: /etc/ssl/client.key\n",
	}
	for name, block := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := NewClientFromKubeconfig(factoryKubeconfig(block))
			if !errors.Is(err, errKubeconfigInvalid) {
				t.Fatalf("err = %v, want errKubeconfigInvalid", err)
			}
			_, err = RESTConfigFromKubeconfig(factoryKubeconfig(block))
			if !errors.Is(err, errKubeconfigInvalid) {
				t.Fatalf("REST err = %v, want errKubeconfigInvalid", err)
			}
		})
	}
}

func TestKubeconfigRejectsLocalCAFile(t *testing.T) {
	kubeconfig := strings.Replace(factoryKubeconfig("    token: embedded-token\n"),
		"    server: https://8.8.8.8:6443",
		"    server: https://8.8.8.8:6443\n    certificate-authority: /etc/ssl/ca.crt", 1)
	if _, err := NewClientFromKubeconfig(kubeconfig); !errors.Is(err, errKubeconfigInvalid) {
		t.Fatalf("err = %v, want errKubeconfigInvalid", err)
	}
}

// TestRESTConfigPinsClientThrottle keeps a console burst from saturating one
// cluster's API server: the factory must always set QPS/Burst.
func TestRESTConfigPinsClientThrottle(t *testing.T) {
	restConfig, err := RESTConfigFromKubeconfig(factoryKubeconfig("    token: embedded-token\n"))
	if err != nil {
		t.Fatalf("embedded token rejected: %v", err)
	}
	if restConfig.QPS != 20 || restConfig.Burst != 40 {
		t.Fatalf("QPS/Burst = %v/%d, want 20/40", restConfig.QPS, restConfig.Burst)
	}
}

func TestKubeconfigAcceptsEmbeddedCredentials(t *testing.T) {
	if _, err := NewClientFromKubeconfig(factoryKubeconfig("    token: embedded-token\n")); err != nil {
		t.Fatalf("embedded token rejected: %v", err)
	}
	if _, err := NewClientFromKubeconfig(factoryKubeconfig("    username: admin\n    password: embedded\n")); err != nil {
		t.Fatalf("embedded basic auth rejected: %v", err)
	}
}
