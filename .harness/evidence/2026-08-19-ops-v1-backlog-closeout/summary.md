# V1 Backlog Closeout Evidence

## Current Outcome

Legacy migration compatibility, Deploy credentials/retry handling, CSV UI APIs,
NamespaceBinding, and K8s concurrency controls are implemented and targeted
verification passes. K8s read-only runtime acceptance also passes with the
provided kubeconfig when TLS verification is explicitly skipped.

F-01, F-02 and F-03 are complete for all locally executable gates; external
mutation and production snapshot gates remain explicit human acceptance steps.

## F-stage Acceptance Matrix

| Task | Result | Evidence |
|---|---|---|
| F-01 QA acceptance issue set | Pass with explicit gates | Backend/race/vet, boundary, migration, Deploy smoke and K8s read-only results |
| F-02 test plan and business smoke | Pass | Gate A/B/C commands, isolated MySQL rehearsal, authenticated desktop/mobile screenshots |
| F-03 clean release and rollback rehearsal | Pass with explicit gates | Backend/frontend artifacts, checksums, isolated migration rehearsal, rollback SQL review |

## Verification Results

- `go test ./... -count=1`, `go vet ./...`, and Windows CGO `go test -race ./...` passed.
- Frontend `npm run build` passed all contract checks and Vite production build.
- Overlay rebuild tests and boundary regression tests passed.
- MySQL 8 migration tests passed against isolated temporary schemas; a full
  `pantheon_ops` dump was restored into `pantheon_ops_f_final_rehearsal`, the
  schema reached version 21/clean, row counts and active-key unique indexes
  matched the source, and the rehearsal schema was dropped afterward.
- Authenticated Deploy package/template/task desktop smoke passed and produced
  three screenshots. Mobile 390x844 package rendering passed with no horizontal
  overflow; screenshots are in `screenshots/`.
- Backend artifact: 78,667,776 bytes, SHA-256
  `C5B5AE18C8ED454D8DC28619A96415A8CBAD7AD9A18FDB635BEC3EAB5BB84B3F`.
- Frontend `dist` size: 3,608,659 bytes; `index.html` SHA-256
  `503B686309DFE708E36B70F0BC407DE306D7AF8EE1D2AA6D7270F3CAD6CDEECA`.

## Changes

- `000013` and `000014` use repeatable MySQL 8 column additions.
- `000015_business_legacy_upgrade` repairs generated active-key columns on
  business tables created by the legacy AutoMigrate path.
- A regression test seeds a legacy `biz_business_scope`, runs all migrations,
  verifies generated `active_code`, and runs migrations a second time.
- `DeployTaskAttempt` stores per-host worker execution history; SSH Start
  returns after enqueueing same-process worker work rather than waiting for SSH.
- Deploy Package/Template exports and imports plus Task export use the shared
  CSV helper and typed frontend client methods.
- `NamespaceBinding` persists cluster/namespace/scope/environment ownership;
  namespace deletes reject missing bindings and forward resourceVersion as a
  Kubernetes delete precondition.
- Workload, ConfigMap, Secret, and Release mutation paths enforce the
  NamespaceBinding action allowlist before contacting the Kubernetes API.

## Residual Risk

- Deploy CredentialRef CRUD now encrypts secret material, returns only
  redacted metadata, and increments its version on rotation.
- Deploy attempts now reconcile expired leases to `retryable` and use a bounded
  retry budget; ClusterCredentialRef is preferred by K8s client construction.
- SSH starts require CredentialRef-only input, persist credential id/version and
  timeout metadata, reconstruct credentials after worker restart, and propagate
  context cancellation through the provider boundary.
- Cluster credential backfill/rotation, bounded Kubernetes Limit/Continue paging,
  resourceVersion response metadata and conflict classification are implemented.
- K8s permission seeds now enumerate cluster, workload, namespace, ConfigMap and
  Secret actions instead of relying on a single coarse update permission.
- The package/template import and export controls plus task export control are
  wired into the existing Arco operational list surfaces and permission model.
- `go test ./pkg/database -run TestRunMigrations -count=1` passed against
  MySQL 8.0.36 using isolated temporary databases derived from the supplied
  DSN; the user `pantheon_ops` schema was not changed.
- Read-only K8s smoke with `--insecure-skip-tls-verify=true` passed: API server
  v1.34.5+k3s1, six namespaces listed, and node `k3s` reported Ready.
- No live K8s or SSH mutation was attempted.
- Authenticated frontend/backend runtime was started locally for visual smoke;
  desktop and mobile evidence is captured under `screenshots/`.

- Historical duplicate data and legacy index-name conversion still require a
  production snapshot rehearsal and maintainer gate.
- Clean foundation-release overlay rebuild now preserves Base `docs/harness`
  method files while overlaying only Ops-owned task/evidence paths; strict
  method health reports 0 findings and 0 warnings (method kit/repo shell 1.4.0).
- SSH read-only acceptance passed with the supplied credentials (`uid=0`,
  Linux host, UTC date); no remote mutation was attempted.
- A full production-style snapshot copy completed `21 -> 22 -> 16 -> 22` with
  the real migration engine. Exact core business row counts were unchanged,
  all Ops objects were restored, and the isolated snapshot database was
  deleted. The source database was never migrated down.
- Direct live-database rollback and live Kubernetes mutation remain
  operator-gated production actions.

## Isolated K8s Deployment Acceptance (2026-08-19)

- Installed K3s `v1.34.5+k3s1` on the provided Kylin Linux V10 single-node host
  (`vbox`, control-plane, Ready). The pinned binary SHA-256 is
  `efaa84416cf59f36f7c1b45bd12988dcf0112288f588a9fd5c0fbca6d309e9d9`.
- Imported the fixed-version K3s amd64 air-gap image bundle after verifying
  SHA-256 `c4b6795a54bb193ea4b156c76a742dd4f93e03bbd03b739d8356d7298aa8a9be`.
  This removed the external registry dependency for CoreDNS,
  local-path-provisioner, and metrics-server.
- Acceptance Deployment reached `1/1 Ready`; its ClusterIP returned
  `pantheon-k3s-ok release=v1 environment=test secret_present=true` and then
  `release=v2 environment=test secret_present=true` after a ConfigMap update
  and rollout restart. The Job completed with
  `pantheon k3s acceptance job complete`.
- Cluster DNS resolved `acceptance-web.pantheon-ops-test.svc.cluster.local`;
  a `local-path` 16Mi PVC reached `Bound` and a BusyBox pod wrote/read the
  mounted volume. Metrics API returned node CPU and memory usage.
- K3s restart recovery passed: the service remained active, the node returned
  Ready, all three system deployments were Running, and the acceptance web
  Deployment remained Ready.
- External API access was verified with the provided kubeconfig and the
  requested `--insecure-skip-tls-verify=true` flag. firewalld was changed only
  to persistently allow `6443/tcp`; no other inbound service port was opened.
- The acceptance namespace and Job are intentionally retained for inspection;
  temporary scripts, image archives, and client kubeconfig copies were removed
  from the server/workspace. The host is a constrained single-node profile
  (1 vCPU, 973MiB RAM, 5.5GiB free disk), so this is an isolated deployment
  acceptance environment rather than a production HA topology.
