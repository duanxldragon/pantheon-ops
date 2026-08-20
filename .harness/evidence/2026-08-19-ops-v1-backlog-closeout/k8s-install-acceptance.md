# Isolated K3s Install and Deployment Acceptance

Date: 2026-08-19
Host: provided isolated Kylin Linux V10 server (`vbox`)

## Install

- K3s `v1.34.5+k3s1` installed as an enabled systemd service.
- Node `vbox` is `Ready`, role `control-plane`, runtime
  `containerd://2.1.5-k3s1`.
- K3s binary SHA-256:
  `efaa84416cf59f36f7c1b45bd12988dcf0112288f588a9fd5c0fbca6d309e9d9`.
- The matching amd64 air-gap image archive was verified with SHA-256
  `c4b6795a54bb193ea4b156c76a742dd4f93e03bbd03b739d8356d7298aa8a9be` and
  imported locally because external registry blob access was unreliable.
- firewalld persistently allows only the Kubernetes API port `6443/tcp` for
  external kubeconfig access. Verification used
  `--insecure-skip-tls-verify=true` as requested.

## Acceptance Results

| Check | Result |
|---|---|
| `acceptance-web` Deployment Ready | PASS, `1/1` |
| ClusterIP Service | PASS, DNS name resolved to `10.43.247.117` and returned the app response |
| ConfigMap and Secret injection | PASS, v1 and v2 responses reported `secret_present=true` |
| Deployment update | PASS, response changed from `release=v1` to `release=v2` |
| Job execution | PASS, `Complete 1/1`, expected completion log |
| CoreDNS | PASS, `1/1 Running` |
| local-path-provisioner | PASS, `1/1 Running`; 16Mi PVC Bound and read/write passed |
| metrics-server | PASS, `kubectl top node` returned CPU/memory |
| K3s restart recovery | PASS, service active, node Ready, workloads recovered |
| External kubeconfig | PASS with `--insecure-skip-tls-verify=true` |

The `pantheon-ops-test` namespace is retained for inspection. Temporary test
scripts, archives, and client kubeconfig copies were removed. This is an
isolated single-node acceptance environment, not a production HA topology.
