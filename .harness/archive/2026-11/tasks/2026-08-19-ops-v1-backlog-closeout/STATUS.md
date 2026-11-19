# Status: Pantheon Ops V1 Backlog Closeout

- Status: `complete-with-explicit-gates`
- Updated: `2026-08-20`
- Executor: Codex `/root`
- Baseline: `a20dc7915bac4c49452982a92c39c7fa2806d6ec`
- Worktree preflight: clean, `main...origin/main`

## Current Batch

`v1-high-risk-gates` (code and isolated external gates complete; Base release gate remains external)

## Completed

- Read workspace, Ops, Base, inheritance, design, delivery-tier, and verification contracts.
- Confirmed the business overlay and cross-module boundary gates pass on the baseline.
- Revalidated the outstanding work against current task artifacts and implementation.
- Applied the Impeccable pre-implementation gate: operational data-table UI,
  dense and calm, existing Arco/shared action controls, no new visual system.
- Added repeatable legacy generated-key repair migration `000015`.
- Made `000013` and `000014` additive columns repeatable on MySQL 8.
- Added a MySQL-backed legacy business-schema upgrade and repeat-run test.
- Added durable Deploy task-attempt records and changed SSH execution to return
  from Start before same-process worker execution begins.
- Added Package/Template CSV import-export and Task CSV export APIs plus typed
  frontend client helpers.
- Added an Ops-owned NamespaceBinding model and migration; namespace creation
  now requires an explicit business scope and environment, and delete requires
  a persisted binding.
- Extended NamespaceBinding action enforcement to Workload scale/restart,
  ConfigMap create/delete, Secret create/delete, and Release create. These
  paths now reject before a Kubernetes client call when unbound or unauthorized.
- Added encrypted Deploy CredentialRef CRUD with secret-free responses,
  version rotation, and optional `credentialRefId` task start resolution.
- Added bounded SSH worker retries and stale-attempt reconciliation.
- Enforced Deploy SSH CredentialRef-only starts; persisted credential id/version,
  host fingerprint and bounded timeout on the task, with restart reconciliation.
- Added replaceable DeployExecutor/provider boundary, context-aware SSH execution,
  cancellation propagation, durable queue recovery and task terminal convergence.
- Added K8s ClusterCredentialRef storage, preferred client resolution, and
  resourceVersion preconditions for namespace, workload, ConfigMap and Secret
  mutations.
- Added credential-ref backfill migration, metadata-only credential lookup and
  rotation API; K8s list endpoints now support bounded Limit/Continue paging and
  expose resourceVersion metadata without secret values.
- Expanded K8s action permission seeds for cluster sync/nodes, workload operations,
  namespace, ConfigMap and Secret actions; conflict errors map to common conflict
  semantics.
- Wired Deploy import/export API helpers into existing Arco list controls and
  added their dedicated permissions.
- Closed P0-AUTH-03 with a MySQL concurrency regression covering ownership move,
  cluster row locking, atomic scope/department update, and stale scoped writes.
- Extended production-style snapshot rehearsal to assert zero active-key
  duplicates, generated-key expressions, unique index names/columns, and
  rollback/reapply row-count preservation.
- Completed reversible isolated Kubernetes ConfigMap/Secret writes with stale
  `resourceVersion` conflict verification and namespace cleanup.
- Completed reversible isolated SSH marker-file write, hash verification, and
  cleanup using a verified ED25519 host key.
- Rebuilt a clean Base overlay and recorded strict Harness method/adoption
  results; consumer-root strict checks remain blocked until Base publishes a
  release whose Release Gate is green.

## Decisions

- Keep every implementation Ops-owned under `business/*` and business migrations.
- Use existing dependencies and same-process database-backed workers.
- Do not mutate production databases or live clusters.

## Evidence

Migration, targeted test, frontend, MySQL snapshot, K8s/SSH mutation, and
Harness inheritance evidence is recorded in
`.harness/evidence/2026-08-19-ops-v1-backlog-closeout/`.

## Next Atomic Action

Maintainer review of the evidence and explicit approval of any production
snapshot/live mutation rehearsal. No live write operation is required for the
local F-stage closeout.

## Blockers

- Read-only K8s smoke passed with `--insecure-skip-tls-verify=true`. In
  addition, the separately provided isolated Kylin host completed K3s install,
  system-component, Deployment/Service/Job, DNS, PVC, metrics, restart, and
  external-kubeconfig acceptance. Production/live-cluster mutation was not
  performed.
- F-01 QA matrix, F-02 test-plan evidence, and F-03 clean-build/release rehearsal
  are complete with explicit external human gates below.

## F-stage Outcome

- F-01: all locally executable P0 checks pass; P1 checks have code/test and
  isolated runtime evidence, with direct production mutation separately gated.
- F-02: backend `go test ./...`, Windows CGO `go test -race ./...`, `go vet ./...`,
  frontend build, overlay/boundary tests, isolated MySQL migration tests, and
  authenticated Deploy desktop/mobile smoke pass.
- F-03: clean-style backend/frontend artifacts were rebuilt in an isolated
  rehearsal directory; backend SHA-256 is recorded in the evidence summary.
  Migration rollback SQL was reviewed and the latest up path was rehearsed on an
  isolated temporary MySQL schema, which was removed afterward.

## Explicit Gates

- Direct production database mutation remains operator-gated; the supplied
  `pantheon_ops` schema was only dumped/read and the rehearsal schema was
  deleted after verification.
- The provided isolated K3s host and isolated SSH host were authorized for
  reversible writes; no production/live service was mutated.
- Clean overlay Harness method/adoption checks pass. Consumer-root strict checks
  remain blocked by the locked Base release age and Base's red Release Gate
  (SonarCloud reports 156 pre-existing unresolved issues); Ops did not copy
  generic Harness files into a local fork.
