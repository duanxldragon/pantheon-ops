# Deploy Immutability / Idempotency Evidence

Task: `ops-deploy-immutability-idempotency` (P0, L2). Owner: `business/deploy`.

## Outcome

A Deploy task is now a durable, immutable change record:

- **Execution snapshot** (`biz_deploy_task.execution_snapshot`) freezes the
  resolved package/template definition (commands, template steps, step
  parameters, source identity) at task creation and on every draft/pending
  update. `Start` and SSH execution reconstruct commands **only** from the
  snapshot; they no longer read live package/template rows.
- **Target snapshot** (`biz_deploy_task.target_snapshot`) plus per-host
  connection facts (`biz_deploy_task_host.ssh_port`, `business_scope_id`,
  `resolved_at`) are frozen inside the Start transaction.
- **Package/template immutability** — once a package or template is referenced
  by a task, execution-defining edits return a stable conflict
  (`business.deploy.package.immutable` / `business.deploy.template.immutable`);
  status/disable edits remain allowed and new versions remain new rows.
- **Idempotent Start** — `StartTaskRequest.idempotencyKey` is persisted on the
  task; the running transition is a conditional `WHERE status IN (draft,pending)`
  update, so concurrent Starts produce one execution. Replay with the same key
  returns the current task with `startRequestReused=true`; a different key while
  running returns `business.deploy.task.alreadyRunning`.
- **Callback CAS** — `MarkHostResultRequest.reportKey` is persisted on the task
  host; the terminal transition is a conditional `WHERE status IN
  (pending,running)` update. A duplicate terminal report is a no-op, a
  conflicting terminal report returns `business.deploy.taskHost.staleReport`.
- **Host lease** — `biz_deploy_host_lease` (unique per host) guards concurrent
  execution per host with owner/expiry; Start acquires, a terminal host result
  releases, and an expired lease is taken over on the next Start.
  `business.deploy.task.leaseConflict` is returned when a host is already leased.

## Changed contracts / schema

- Migration `000013_deploy_snapshot_idempotency_lease` (up + down), verified
  against MySQL 8.0.36 for empty/upgrade/rollback.
- Models: `DeployTask` (+execution_snapshot, target_snapshot, start_request_key),
  `DeployTaskHost` (+ssh_port, business_scope_id, report_key, resolved_at, and a
  `(task_id,host_id)` unique index aligned with the migration), new
  `DeployHostLease`.
- DTOs: `StartTaskRequest.idempotencyKey`, `MarkHostResultRequest.reportKey`,
  `TaskResponse.startRequestReused`, expanded `TaskHostResponse`.
- Service: snapshot build/consume, immutable package/template updates,
  idempotent Start, callback CAS, and lease acquire/release/takeover.

## Validation

- `go test -count=1 ./modules/business/deploy/...` with local MySQL 8.0.36: **pass**
  (the previously skipped MySQL suite now runs; see residual-risk note below).
- `go build ./...` and `go vet ./...`: **pass**.
- `node scripts/check-business-module-boundaries.mjs`: **pass**.
- `git diff --check`: **pass**.
- Migration smoke: `000012` + `000013` up creates the columns/lease table; down
  drops them.

## Residual risk

- `go test -race` now builds and runs with the native MinGW CGO toolchain at
  `D:\msys64\mingw64\bin` (`PATH=.../mingw64/bin:$PATH CC=gcc CGO_ENABLED=1`):
  `go test -race -count=1 ./modules/business/deploy/...` passes, so the
  lease/callback-CAS/idempotent-start concurrency is now also race-detector
  proven, not just conditional-update proven.
- `go test -count=1 ./...` now passes across the backend except the pre-existing
  `TestPurgeModuleAllowsBusinessStaticModuleWithoutTable` failure in
  `modules/lowcode/dynamicmodule`, which reproduces identically at the clean base
  commit `ee21350` (verified via a throwaway worktree) and is tracked as an
  explicit out-of-scope gap. The former `bizscope`/`cmdb` failures are resolved
  by the `ops-bizscope-datascope-ownership` child's verification pass.
- The deploy test suite had several latent failures that were previously hidden
  because the MySQL suite skipped without a DSN. Four were stale assertions
  (decorated stdout, variable-based tarball/service name, trace ordering) and
  four were `dept_id` fixture gaps; all are fixed in this task's test file so the
  module is green against real MySQL.
- Frontend `npm run type-check` was already red at the K8s pages per the audit and
  is not part of this task; the Deploy DTO changes are additive (new optional
  fields) and do not change existing frontend contracts.
