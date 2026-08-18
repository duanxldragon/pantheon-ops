# Execution Queue

This file is the cost-aware handoff queue for the unfinished Pantheon Ops
foundation hardening work. It is written for tools or models that have no memory
of the audit conversation.

## Current State

- Program status: `complete`.
- Implementation status: all eight child tasks are complete:
  migrations, capability boundaries, BizScope ownership, Deploy reliability,
  K8s Release reliability, ServiceInstance foundation, State Machines, and
  Quality Gates.
- Source audit: `docs/business-module-review-summary.md`.
- Program packet: `docs/harness/tasks/2026-08-17-ops-foundation-hardening.task.md`.
- Stateless entry: `.harness/tasks/2026-08-17-ops-foundation-hardening/HANDOFF.md`.
- Machine manifest: `.harness/tasks/2026-08-17-ops-foundation-hardening/manifest.json`.

Do not treat this queue as implementation approval. A child task may start only
when its dependency status is `complete` and its owner has recorded a fresh
dirty-worktree preflight in that child's `STATUS.md`.

There is no remaining dependency-ready child task in this program. Remaining
follow-ups are explicitly deferred production migration generation and
credentialed/runtime validation.

## Non-Negotiable Rules

1. A business module may access only tables it owns.
2. Cross-module reads use provider-owned typed Reader/Capability APIs.
3. Cross-module writes use provider-owned Command APIs.
4. Same-process calls use typed Go interfaces; HTTP/gRPC is reserved for a real
   deployment boundary.
5. Provider APIs must re-check authorization, data scope, state transitions, and
   audit metadata.
6. Cross-module DTOs must not expose GORM models or raw table schemas.
7. Do not copy `pantheon-base` platform/system behavior into `pantheon-ops`.
8. Preserve all existing dirty-tree changes unless the owner explicitly scopes
   them into the task.

## Batch Order

| Batch | Start condition | Tasks | Owner profile | Stop condition |
| --- | --- | --- | --- | --- |
| A1 | Now | `ops-business-versioned-migrations` | strong database/backend owner, single migration sequence owner | destructive migration, unclear MySQL online DDL impact, production outage decision |
| A2 | Now | `ops-cross-module-capability-boundaries` | strong architecture/backend owner, single contract owner | generic query API proposal, package cycle that leaks GORM models, base-owned contract |
| B1 | A2 complete | `ops-bizscope-datascope-ownership` | strong backend/auth owner; low-cost frontend lane only after API freeze | data-scope semantics not derivable from base contracts |
| B2 | A2 complete | `ops-deploy-immutability-idempotency` | strong backend/concurrency owner | incompatible Task/TaskHost/package snapshot contract |
| B3 | A2 complete | `ops-k8s-release-reliability` | strong K8s/backend owner; low-cost UI lane after DTO freeze | real external cluster mutation without maintainer gate |
| C | A1, A2, B1 complete | `ops-service-instance-foundation` | architect/strong single owner | model starts absorbing CMDB resource facts or K8s live object facts |
| D | B2 and C complete | `ops-resource-service-state-machines` | very strong single state-model owner | Host asset state and service runtime state cannot be cleanly separated |
| E | All implementation contracts frozen | `ops-architecture-quality-gates` | lower-cost script/test owners allowed; strong reviewer required | checker must allow production direct-table access to pass |

All batches above have reached their stop conditions and are recorded as
`complete` in the parent and child `STATUS.md`/`manifest.json` files.

`A1` and `A2` can run in parallel only if file ownership is recorded and there is
no migration/contract collision. Do not run multiple owners against the same
model, service, migration sequence, or public DTO.

## Child Task Cards

### ops-business-versioned-migrations

- Task packet: `docs/harness/tasks/2026-08-17-ops-business-versioned-migrations.task.md`.
- Handoff: `.harness/tasks/ops-business-versioned-migrations/HANDOFF.md`.
- Primary result: business schema is reproducible without production AutoMigrate.
- Expected ownership: migration files, business schema bootstrap, migration tests.
- Must not own: cross-module API design except approved schema fields.
- Minimum evidence: empty DB migration, upgrade DB migration, repeat run,
  duplicate-key behavior, failure-recovery note.

### ops-cross-module-capability-boundaries

- Task packet: `docs/harness/tasks/2026-08-17-ops-cross-module-capability-boundaries.task.md`.
- Handoff: `.harness/tasks/ops-cross-module-capability-boundaries/HANDOFF.md`.
- Primary result: all runtime cross-module access uses owner APIs.
- Expected ownership: provider Reader/Command DTOs, injection wiring, direct-access checker.
- Must not own: full BizScope authorization implementation, Deploy lease logic,
  ServiceInstance schema.
- Minimum evidence: fresh forbidden-access scan, provider contract tests,
  dependency diagram, denied-scope smoke.

### ops-bizscope-datascope-ownership

- Task packet: `docs/harness/tasks/2026-08-17-ops-bizscope-datascope-ownership.task.md`.
- Handoff: `.harness/tasks/ops-bizscope-datascope-ownership/HANDOFF.md`.
- Primary result: BizScope becomes a server-side data boundary and host ownership
  is race-safe.
- Expected ownership: BizScope data-scope enforcement, CMDB ownership command,
  bind/unbind/delete reference checks, visible host selection.
- Must not own: generic IAM/DataScope framework changes in base.
- Minimum evidence: unauthorized scope denial, concurrent bind collision,
  delete-with-reference denial, rename/snapshot behavior.

### ops-deploy-immutability-idempotency

- Task packet: `docs/harness/tasks/2026-08-17-ops-deploy-immutability-idempotency.task.md`.
- Handoff: `.harness/tasks/ops-deploy-immutability-idempotency/HANDOFF.md`.
- Primary result: a deploy task is a frozen, idempotent, recoverable change
  record.
- Expected ownership: package/template/task snapshot model, TaskHost uniqueness,
  Start idempotency, execution lease, duplicate callback handling.
- Must not own: ServiceInstance full lifecycle, K8s release executor.
- Minimum evidence: duplicate Start, duplicate callback, stale lease recovery,
  immutable snapshot, denied unowned host target.

### ops-k8s-release-reliability

- Task packet: `docs/harness/tasks/2026-08-17-ops-k8s-release-reliability.task.md`.
- Handoff: `.harness/tasks/ops-k8s-release-reliability/HANDOFF.md`.
- Primary result: K8s release records exist before mutation and reflect rollout
  outcome.
- Expected ownership: release pending/apply/rollout/fail sequence, cluster
  reference protection, BizScope API usage, K8s type-check fixes in this lane if
  DTOs are stable.
- Must not own: copying every Kubernetes object into CMDB.
- Minimum evidence: pending audit before apply, rollout success, rollout timeout,
  apply failure, cluster delete blocked by releases.

### ops-service-instance-foundation

- Task packet: `docs/harness/tasks/2026-08-17-ops-service-instance-foundation.task.md`.
- Handoff: `.harness/tasks/ops-service-instance-foundation/HANDOFF.md`.
- Primary result: Application/Service/ServiceInstance provides a common service
  identity for VM and K8s without turning CMDB into runtime state storage.
- Expected ownership: minimal service bounded context, schema/API/DTO,
  VM/K8s binding refs, Deploy reference contract.
- Must not own: tenant/project hierarchy, universal asset table, K8s object cache.
- Minimum evidence: BizScope to Service to VM instance path, K8s binding path,
  delete-reference denial, Deploy task requires service identity.

### ops-resource-service-state-machines

- Task packet: `docs/harness/tasks/2026-08-17-ops-resource-service-state-machines.task.md`.
- Handoff: `.harness/tasks/ops-resource-service-state-machines/HANDOFF.md`.
- Primary result: Host asset lifecycle and ServiceInstance desired/observed/health
  state have explicit legal transitions.
- Expected ownership: state transition services, illegal transition tests, Deploy
  callback ordering, observed-state reconciliation hooks.
- Must not own: package snapshot design or service schema creation.
- Minimum evidence: illegal transition denial, out-of-order callback behavior,
  failed health check, rollback transition, Host status unaffected by service stop.

### ops-architecture-quality-gates

- Task packet: `docs/harness/tasks/2026-08-17-ops-architecture-quality-gates.task.md`.
- Handoff: `.harness/tasks/ops-architecture-quality-gates/HANDOFF.md`.
- Primary result: architecture, migration, contract, backend, frontend, and smoke
  checks fail future regressions.
- Expected ownership: deterministic scripts/tests/docs gates and final aggregate
  evidence.
- Must not own: silently weakening checks to pass the dirty baseline.
- Minimum evidence: direct-access checker, migration smoke, contract tests,
  `go test`, race, vet, frontend type-check, business overlay check.

## Cost-Aware Assignment Guidance

- Use strong models for schema, authorization, state machine, concurrency, and
  K8s mutation semantics.
- Use lower-cost models only for isolated tests, docs, deterministic scripts, and
  frontend type fixes after DTOs and contracts are frozen.
- Every lower-cost lane must produce evidence, but a strong reviewer must approve
  the owning task before it moves to `complete`.
- Do not split a task across tools when they would edit the same files or public
  contract at the same time.

## Required Handoff Update

Before a tool stops, it must update the selected child `STATUS.md` with:

1. current status;
2. owner/tool identity;
3. baseline commit and dirty-file collisions;
4. completed atomic changes;
5. decisions made;
6. evidence files created;
7. next atomic action;
8. blockers or residual risk.

No later tool should need chat history to resume.
