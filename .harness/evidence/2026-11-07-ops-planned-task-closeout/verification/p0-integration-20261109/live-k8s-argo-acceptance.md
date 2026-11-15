# Bounded live K8s / Argo acceptance — 2026-11-09

Parent task: `2026-11-07-ops-planned-task-closeout`; tier L2; Generator runtime evidence only. O2/O3/O4/D1/D2/D3 and whole P0 remain open. No independent final product approval is supplied.

## Verified outcome

| Scope                                                            | Exact command                                                                                                                                      | Result                                                                                            |
| ---------------------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------------- |
| Product public ServiceService + isolated MySQL + real public K3s | `python .local/ops-acceptance/live-k8s-argo-service.py`                                                                                            | exit 0; 22 checks; race-instrumented driver; cleanup exit 0                                       |
| Real Argo v3.2.0 provider only                                   | `python .local/ops-acceptance/live-k8s-argo-provider-v7.py`                                                                                        | exit 0; 28 checks; each owned-resource/server cleanup exit 0                                      |
| Final provider health / owned resource absence                   | `python .local/ops-acceptance/live-k8s-argo-postflight.py`                                                                                         | exit 0; four Ready core Pods; 960Mi total limits; zero owned namespaces, applications or projects |
| Driver diagnostics                                               | Python compile, `go vet -mod=readonly <ignored driver>`, `go build -mod=readonly -race <ignored driver>` with Windows MinGW PATH and CGO_ENABLED=1 | all exit 0                                                                                        |

Service coverage: create/read/update/delete public methods; live field preservation; stale update/delete; exactly one concurrent CAS winner; cached-client department/read/write, namespace action/binding and credential-reference revocations; pre-cancel; same-name ABA; provider RBAC denied update/delete. Actual UID/version/selector state was checked through the live provider and independently through kubectl. Neither HTTP authorization nor product audit was exercised.

Argo provider coverage: verified certificate and authenticated API; unauthenticated and invalid credential 401; finite 900-second project token with exact application get/sync permissions; two pinned revision syncs to terminal success; provider history IDs 0/1; history-ID rollback restored actual Service selector from v2 to v1; token revocation returned 401 and preserved completed operation; unowned destination returned 400 and was not persisted; zero workload Pods created. This is **provider-only**, not an Ops ArgoCDService/reconciler or HTTP acceptance claim.

## Evidence and recovery findings

- [Product Service result](live-k8s-argo-service-result.json), [Argo provider v7 result](live-k8s-argo-provider-v7-result.json), [final health](live-k8s-argo-postflight-result.json), [aggregate exact scope](live-k8s-argo-acceptance-result.json), [driver diagnostics](live-k8s-argo-driver-diagnostics.json).
- Failed Service and Argo attempts retain their original exit 1 evidence. The first Service create failed; certificate inspection showed no public IP SAN. The successful driver uses the supported `tls-server-name: kubernetes` with trusted CA verification and the same public origin. Checked dial and private network policy were not weakened.
- Argo repo-server's public GitHub and public-host fixture probes timed out (124 at 15s). An isolated in-cluster source was sufficient for provider acceptance. External Git egress remains unverified. The initial dumb HTTP fixture produced a real Argo `unexpected EOF` rejection; the final standard smart HTTP fixture resolved that driver problem. An intermediate fixed-port allocation failed; final fixture uses an ephemeral port.
- Owned resources were checked for exact UID and both owner/driver labels before deletion, then checked absent. Fixture server PID/command identity was checked before termination. Six temporary Git directories were removed only after resolved path, commit hash, owner labels and server absence checks; both credential files are confirmed absent ([cleanup exit 0](live-k8s-argo-file-cleanup-result.json)). Installed providers were retained. Only the two isolated local fixture DBs are retained, explicitly listed in the aggregate result; remote in-memory/0600 credential configuration was consumed and removed. Evidence contains no passwords, tokens or private keys.

## Exact remaining gaps

- Ops ArgoCDService action persistence/reconciler, Ops HTTP routes and actor/DataScope authorization, product audit, verified public Argo origin/production CA integration.
- Service HTTP and audit, BizScope/CMDB key-only grant/revoke, in-flight revocation, delayed-provider timeout, restart/recovery and complex deployment cancellation/retry.
- O3 broader manifests/config/resources and complex/old-version deployment restoration; current Argo source changes only an isolated Service selector.
- Velero and signed backup/restore/RPO/RTO/capacity acceptance.
- Independent final CODE/ARCH and visual/functional acceptance. No UI changes were made, so rendered UI evidence was not produced.

No new product defect was reproduced within the tested scope. Generator assertions and diagnostics establish these bounded results only; they do not supply independent review or approve whole P0. Backend/SQL/mod/sum, foundation lock, consumer baseline, overlays, shared closeout files and other agents' edits were untouched by this adapter.

Gate Outcomes: real Service CAS/RBAC/revocation checks passed | real Argo authentication, terminal/history rollback and refusal checks passed | fixture protocol/port diagnostics caught harness failures; no gate waived.
