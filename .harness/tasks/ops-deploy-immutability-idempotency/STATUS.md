# Status: Deploy Reliability

- Status: `complete`
- Updated: `2026-08-18`
- Executor: Codex `/root` (Buffy takeover)
- Baseline commit: `ee213508a32ebdf27685128e0c719e9036272de5`
- Dirty-file collisions: `backend/modules/business/deploy/deploy.go`,
  `deploy_seed.go`, and `deploy_service.go` are existing dirty changes from the
  capability-boundary child. Preserved and extended in place; no reset/restore.

## Completed

- Frozen execution snapshot (`biz_deploy_task.execution_snapshot`) built at
  creation and on every draft/pending update; Start and SSH execution consume it
  only, never live package/template rows.
- Frozen target snapshot (`biz_deploy_task.target_snapshot`) plus per-host
  connection facts (`ssh_port`, `business_scope_id`, `resolved_at`) frozen in the
  Start transaction.
- Package/template immutability on task reference: execution-defining edits
  return `business.deploy.package.immutable` / `business.deploy.template.immutable`;
  status/disable edits stay allowed; new versions remain new rows.
- Idempotent Start: persisted `start_request_key`, conditional
  `WHERE status IN (draft,pending)` claim, same-key replay
  (`startRequestReused=true`), different-key-while-running conflict
  (`business.deploy.task.alreadyRunning`).
- Callback CAS: persisted `report_key`, conditional
  `WHERE status IN (pending,running)` transition, duplicate report no-op,
  conflicting terminal report `business.deploy.taskHost.staleReport`.
- Host lease (`biz_deploy_host_lease`, unique per host): Start acquires, terminal
  host result releases, expired lease takeover; `business.deploy.task.leaseConflict`.
- Migration `000013_deploy_snapshot_idempotency_lease` (up + down).
- Deploy test suite unmasked against local MySQL 8.0.36 and made green (fixed
  four stale assertions and four `dept_id` fixture gaps). Focused tests added in
  `deploy_reliability_test.go`.

## Decisions

- Freeze point: execution intent is frozen at task creation and rebuilt on
  draft/pending update; the target snapshot is frozen in the Start transaction
  (group targets resolve only at Start). Start never reads live package/template
  or CMDB rows to reconstruct commands.
- Lease is database-backed and scoped to the host resource; owner-only
  release and expiry takeover; no distributed lock service.
- Idempotency identity is per-task (`start_request_key`); empty key still gets the
  single-winner guarantee but cannot replay.
- Callback identity is `report_key`; empty key falls back to matching the existing
  terminal status for the harmless-duplicate case.

## Evidence

- `.harness/evidence/ops-deploy-immutability-idempotency/commands.json`
- `.harness/evidence/ops-deploy-immutability-idempotency/summary.md`
- `.harness/evidence/ops-deploy-immutability-idempotency/review.md`

## Next Atomic Action

Carry the frozen Deploy contract into `ops-resource-service-state-machines`;
production model-to-migration upgrade scripting remains deferred.

## Blockers

- None for this child task. The prior Cygwin/MinGW race gap is closed by the
  MinGW evidence recorded above.
