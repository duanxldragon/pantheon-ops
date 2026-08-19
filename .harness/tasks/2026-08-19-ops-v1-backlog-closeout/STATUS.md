# Status: Pantheon Ops V1 Backlog Closeout

- Status: `complete-with-explicit-gates`
- Updated: `2026-08-19`
- Executor: Codex `/root`
- Baseline: `759d0a3313043dd02defdff008fea95d3e43105a`
- Worktree preflight: clean, `main...origin/main`

## Current Batch

`deploy-k8s-import-export` (code complete; external runtime gate recorded)

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

## Decisions

- Keep every implementation Ops-owned under `business/*` and business migrations.
- Use existing dependencies and same-process database-backed workers.
- Do not mutate production databases or live clusters.

## Evidence

Migration, targeted test, frontend, and read-only K8s evidence is recorded in
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

- F-01: all locally executable P0 checks pass; P1 checks have code/test evidence
  or are explicitly deferred where they require live SSH/K8s mutation.
- F-02: backend `go test ./...`, Windows CGO `go test -race ./...`, `go vet ./...`,
  frontend build, overlay/boundary tests, isolated MySQL migration tests, and
  authenticated Deploy desktop/mobile smoke pass.
- F-03: clean-style backend/frontend artifacts were rebuilt in an isolated
  rehearsal directory; backend SHA-256 is recorded in the evidence summary.
  Migration rollback SQL was reviewed and the latest up path was rehearsed on an
  isolated temporary MySQL schema, which was removed afterward.

## Explicit Gates

- Production database duplicate/index conversion rehearsal requires maintainer
  approval and a production snapshot.
- Production/live Kubernetes and SSH mutation scenarios remain gated. The
  isolated acceptance host was explicitly authorized and is documented in
  `k8s-install-acceptance.md`.
- Harness strict method/adoption checks remain blocked by missing shared Base
  landing/adaptor files; this is a Base-first inheritance gap, not copied into Ops.
