# Task: Deploy Immutability, Snapshots, Idempotency, And Lease

Task ID: `ops-deploy-immutability-idempotency`  
Priority: P0  
Owner: `business/deploy`  
Depends on: `ops-cross-module-capability-boundaries`; coordinate schema with migrations and final authorization smoke with `ops-bizscope-datascope-ownership`  
Blocks: reliable VM lifecycle actions and ServiceInstance state transitions

## Goal And Invariants

Make a Deploy Task a durable change record whose execution intent cannot change
under it and whose start/retry/callback behavior is safe under HTTP retries and
concurrent operators.

Required invariants:

1. A package/template execution definition used by a task is frozen in an explicit
   snapshot. Historical execution never reads mutable current package/template
   rows to reconstruct commands.
2. Published execution fields cannot be edited in place. A new version creates a
   new package/template; disable/retire is separate from mutation.
3. `POST /tasks/:id/start` is idempotent with an `Idempotency-Key` or equivalent
   persisted request identity. Concurrent starts produce one execution intent.
4. A task host appears at most once per task and host. Duplicate callbacks are
   harmless; conflicting terminal callbacks are rejected.
5. The same Host/ServiceInstance cannot run two conflicting executions at once.
   A lease has owner, expiry, heartbeat/release semantics, and recovery after
   timeout.
6. Scope and target authorization are rechecked by BizScope/CMDB provider APIs at
   task creation and start; frontend filtering is not security.
7. Task status transitions are conditional and auditable; a request timeout does
   not leave an unowned, unrecoverable `running` execution.

## Why This Is P0

The audit found mutable packages, incomplete task snapshots, non-atomic Start,
missing TaskHost uniqueness, no lease, and synchronous SSH execution in the HTTP
request. These can deploy the wrong version, duplicate installation, or leave a
host in an unknown state.

## Evidence To Revalidate

- `backend/modules/business/deploy/deploy_model.go:53` for current package,
  template, task, and task-host fields.
- `deploy/deploy_service.go:208` for package mutation;
  `:573` for draft/pending update rules;
  `:807` for Start ordering;
  `:954` for CMDB writeback;
  `:1131` for task visibility direct join.
- `backend/modules/business/deploy/deploy_handler.go` Start and report routes.
- Existing `DeployCMDBCapability` contract and current task API/frontend behavior.

## In Scope

- Add a version/publish/freeze policy without removing existing package/template
  CRUD routes. At minimum, execution-defining fields become immutable once a
  package/template is referenced by a task; edits return a stable conflict. New
  versions are created as new rows.
- Add a typed execution snapshot containing package/template IDs, names/versions,
  commands/steps, template parameters, source identity, and checksum/digest when
  available. Snapshot must not contain secrets.
- Freeze complete target snapshots at the contract-defined point (recommended
  task creation/start transaction) including host ID, hostname, IP, SSH port, OS,
  ownership ID, and the target resolution timestamp. Do not use current CMDB rows
  to rewrite history.
- Add persisted Start idempotency identity and conditional state update. Define
  behavior for same key/different payload and different key while already running.
- Add active uniqueness for `(task_id, host_id)` and a lease/lock record or fields
  with owner, expiry, heartbeat, and takeover rules. Prefer existing DB patterns;
  do not add a distributed lock service.
- Make report/callback handling idempotent with a report/event identity or strict
  terminal-state CAS. Record executor ID, correlation ID, timestamps, and result.
- Keep execution out of the request lifecycle as a follow-up boundary if a full
  worker is too large for this task; at minimum define a recoverable execution
  record and explicit timeout/retry semantics. Do not claim worker support unless
  a restart/reconciliation test passes.
- Update Deploy API DTOs and frontend to display frozen version/target and clear
  conflict/running states. Preserve existing routes where possible.

## Out Of Scope

- start/stop/health/rollback service actions (task 6);
- the ServiceInstance table itself (task 5);
- K8s Release implementation;
- a new artifact registry, Docker/Helm executor, or event-sourcing system;
- direct CMDB/BizScope table access;
- persisting credentials or private keys.

## Contract Details

Suggested additions, adapted to local DTO conventions:

- `StartTaskRequest` accepts request idempotency through the existing header
  handling or a clearly documented field; response includes task/execution ID and
  whether it was reused.
- `DeployTask` stores immutable `ExecutionSnapshot`, `TargetSnapshot`,
  `start_request_key`, `execution_revision`, and correlation/actor metadata.
- `DeployTaskHost` stores connection facts and report identity captured at freeze.
- Lease operations are owner-only (`acquire`, `renew`, `release`, `takeoverAfter`
  expiry) and never exposed as arbitrary CRUD.
- Error semantics distinguish immutable conflict, duplicate/idempotent replay,
  lease conflict, stale callback, invalid state, target authorization failure,
  and executor timeout.

## Expected Files

- `backend/modules/business/deploy/deploy_model.go`, `deploy_dto.go`,
  `deploy_service.go`, `deploy_handler.go`, `deploy.go`, and focused tests;
- coordinated migration SQL for snapshots, idempotency, uniqueness, and lease;
- request provider DTO expansion from the capability/CMDB owner; the Deploy task
  must not edit CMDB capability implementation in parallel;
- `frontend/src/modules/business/deploy/package/*`, `template/*`, `task/*` and
  i18n/permission files only where the API contract requires it;
- Deploy API/smoke tests and evidence.

Do not touch `../pantheon-base/**`, BizScope internals except injected API calls,
ServiceInstance files owned by task 5, or unrelated frontend modules.

## Verification Matrix

| Scenario                                         | Required result                                              |
| ------------------------------------------------ | ------------------------------------------------------------ |
| edit referenced package/template execution field | stable immutable conflict                                    |
| create task                                      | complete package/template/target snapshot is stored          |
| current CMDB changes after create/start          | historical task remains unchanged                            |
| same Start request twice                         | same execution result, no duplicate TaskHost                 |
| concurrent Start requests                        | one state transition, one lease owner                        |
| different idempotency key while running          | deterministic conflict                                       |
| duplicate callback                               | no duplicate side effect; original terminal result retained  |
| conflicting callback                             | stale/conflict error and audit event                         |
| lease expiry                                     | one controlled takeover/recovery, no double execution        |
| unauthorized/inactive target                     | provider rejects at backend                                  |
| process/request timeout                          | task remains recoverable and reconciliation state is visible |

Commands from `pantheon-ops/backend`:

```text
go test -count=1 ./modules/business/deploy/...
go test -race ./modules/business/deploy/...
go vet ./...
```

From `pantheon-ops/frontend` as applicable:

```text
npm run type-check
npm run test:smoke:business:deploy:api
npm run test:smoke:business:deploy
```

Write `.harness/evidence/ops-deploy-immutability-idempotency/` with request
replay/concurrency output, schema evidence, API/UI evidence, and findings-first
review. Runtime/visual gaps must be explicit.

## Stop Conditions

Stop for a requirement to change the base executor framework, distributed lock,
destructive package history rewrite, secret persistence, or an incompatible
external API. Record the smallest compatible contract and deferment.
