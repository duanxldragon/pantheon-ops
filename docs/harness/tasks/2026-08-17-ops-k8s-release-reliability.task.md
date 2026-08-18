# Task: Kubernetes Release Reliability

Task ID: `ops-k8s-release-reliability`  
Priority: P0  
Owner: `business/k8s`  
Depends on: capability boundaries; coordinate schema with migrations  
Coordinates with: BizScope ownership and ServiceInstance linkage  
Blocks: production K8s mutation acceptance and final quality gates

## Goal And Boundary

Keep Kubernetes API as the source of runtime object truth while making Pantheon
Release the durable change/audit record. Do not copy Namespace/Workload/Service
state into CMDB. CMDB may expose Cluster stable identity; K8s Manager owns live
resources, rollout observation, logs, scaling, restart, and release history.

## Required Invariants

1. A Release row with `pending`/`applying` intent, actor, target, image/config
   snapshot, scope, and correlation ID is committed before mutating Kubernetes.
2. Kubernetes apply success is not release success. Final success requires the
   matching workload generation/revision to reach rollout/availability conditions.
3. Apply, rollout, timeout, audit-write, and client errors each persist a terminal
   failure or a clearly recoverable transitional state.
4. Rollback creates a new immutable Release referencing the prior release; it
   never rewrites history.
5. Cluster deletion is rejected while Release/Service references exist, with a
   race-safe provider reference check.
6. Cluster and release scope validation uses `BizScopeReader.GetActive` and K8s
   provider authorization; no direct BizScope table/model query remains.
7. K8s DTOs expose metadata/status needed for UI, never kubeconfig/Secret payloads.

## Why This Is P0

The current flow changes the workload before reliably writing audit history,
ignores some failure-audit insert errors, marks success without waiting for rollout,
and lets Cluster deletion ignore Release references. This can leave an untracked
production mutation or falsely successful release.

## Evidence To Revalidate

- `backend/modules/business/k8s/release/release_service.go:42` (`Create`) and
  `:155` (`Rollback`).
- `backend/modules/business/k8s/cluster/cluster_service.go:192` (`Delete`) and
  `:332` (`getBusinessScope`).
- `backend/modules/business/k8s/release/release_model.go` and DTO/handler.
- `backend/modules/business/k8s/k8s_seed.go` for permissions/menu/audit coverage.
- Frontend baseline failures previously reported at `ClusterList.tsx:219`,
  `ReleaseList.tsx:15/186`, and `WorkloadList.tsx:14/231`; re-run type-check.

## In Scope

- Extend Release model/status with pending/applying/succeeded/failed/timed_out (or
  repository-equivalent canonical values), attempt, target generation/revision,
  correlation/idempotency ID, previous release ID, timestamps, error summary, and
  immutable request/target snapshot.
- Persist intent first. Then call Kubernetes. Update to applying and poll/watch
  Deployment/StatefulSet/DaemonSet conditions with explicit timeout/cancellation.
  Record observed generation, replicas/availability, and condition summary.
- On apply ambiguity (request timeout), reconcile the targeted generation before
  deciding retry/failure. Do not blindly issue a duplicate mutation.
- Make Create/Rollback idempotent and callback/reconciliation updates conditional.
  A retry with the same key returns the same Release.
- Make audit persistence failure visible. Since DB and Kubernetes are not one
  transaction, use durable intent plus reconciliation/repair; do not pretend a DB
  transaction can roll back the cluster mutation.
- Reject Cluster delete through Release/Service reference APIs. Coordinate a
  tombstone/lock or equivalent with create to close the delete/create race.
- Replace direct BizScope access with the capability task contract and reject
  inactive/unauthorized scope.
- Tighten action permissions so namespace/configmap/secret/workload/release
  mutations do not all reuse broad cluster update permission. Reuse existing IAM
  contracts and update seed, handlers, frontend buttons, and i18n together.
- Fix K8s TypeScript failures in files changed by this task. Render release,
  failure, rollout, empty/loading/error, and confirmation states on desktop/mobile.
- Update K8s design docs to match `client-go`, runtime ownership, and release states.

## Out Of Scope

- copying live Kubernetes objects into CMDB;
- merging K8s Release and VM Deploy execution code;
- storing kubeconfig/Secret plaintext in DTO/history;
- adding Namespace -> BizScope mapping unless shared-cluster ownership is an
  accepted current requirement. If needed, create a separate follow-up contract;
- a generic GitOps/Helm controller, multi-cluster scheduler, or service mesh;
- base IAM or secret-management redesign.

## Rollout Semantics

- Deployment: observed generation matches, progress not failed, desired replicas
  updated/available, and Available/Progressing conditions meet the accepted policy.
- StatefulSet: observed generation and updated/ready revisions/replicas meet policy.
- DaemonSet: observed generation and desired/updated/available counts meet policy.
- Timeout must preserve the last observed condition and permit reconciliation.
- Rollback success uses the same rollout rules as forward release.

Exact criteria and timeout values must be configuration/contract driven and tested
with fake clients/watches; do not equate one successful Update call with rollout.

## Expected Files

- `backend/modules/business/k8s/release/{release_model,release_dto,release_service,release_handler}.go`
  and tests;
- `backend/modules/business/k8s/cluster/cluster_service.go`, module wiring,
  `k8s_seed.go`, permission policy/overlay registry only where required;
- coordinated versioned migration SQL;
- `frontend/src/modules/business/k8s/api.ts`, Cluster/Release/Workload pages,
  permissions and i18n;
- K8s design docs and release/API/smoke/visual tests.

Do not touch `../pantheon-base/**`, CMDB Host tables, VM Deploy executor, or
unrelated system/platform UI.

## Verification Matrix

| Scenario                         | Required result                                              |
| -------------------------------- | ------------------------------------------------------------ |
| intent DB write fails            | no Kubernetes mutation                                       |
| Kubernetes apply fails           | Release terminal failed with cause                           |
| API timeout after possible apply | reconciliation determines observed outcome                   |
| rollout succeeds                 | Release succeeds only after matching generation is available |
| rollout stalls/fails/times out   | status and conditions persisted; retry/reconcile possible    |
| audit/result update fails        | detectable repair state; no false success                    |
| duplicate Create/Rollback        | one immutable Release per idempotency key                    |
| rollback                         | new Release references prior version and waits for rollout   |
| Cluster delete with reference    | rejected; delete/create race cannot orphan history           |
| inactive/unauthorized scope      | denied by provider before mutation                           |
| permissions                      | each mutation requires its server-side action permission     |
| frontend                         | type-check passes; rollout/error/confirmation states render  |

Commands:

```text
cd backend
go test -count=1 ./modules/business/k8s/...
go test -race ./modules/business/k8s/...
go vet ./...
cd ../frontend
npm run type-check
npm run build
```

Add focused K8s API/release smoke using fake or disposable cluster as documented.
Evidence root: `.harness/evidence/ops-k8s-release-reliability/`; rendered evidence
is mandatory for UI completion, otherwise record the runtime gap.

## Stop Conditions

Stop for base IAM/secret changes, production cluster credentials/operations,
unbounded rollout waits, a shared-cluster Namespace ownership requirement not yet
approved, or an API break. Never test mutations against an external cluster
without explicit authorization.
