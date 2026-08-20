# Pantheon Ops V1 Backlog Closeout

Task ID: `2026-08-19-ops-v1-backlog-closeout`

## Classification

- Repository: `pantheon-ops`
- Tier: `L2`
- Layer: `business/deploy`, `business/k8s`, business migrations, business UI
- Mode: implementation, runtime verification, visual verification, documentation closeout

## Goal

Close the remaining V1 work that is still observable in the current repository:
legacy business-schema upgrades, asynchronous Deploy execution, Kubernetes
ownership and credential boundaries, Deploy import/export, and release evidence.

## In Scope

1. Idempotent MySQL upgrade migrations for legacy AutoMigrate business tables.
2. Durable Deploy attempts with worker claims, timeout, retry, cancellation, and reconciliation.
3. Namespace-to-BizScope ownership, credential references, Kubernetes write-conflict guards, and action permissions.
4. Deploy Package/Template CSV import-export and Deploy Task CSV export.
5. Backend, frontend, overlay, smoke, runtime, visual, evidence, review, and stale task-state closeout.

## Out Of Scope

- Changes to `pantheon-base` platform or `system/*` behavior.
- Direct mutation of a production database or a live Kubernetes cluster.
- A new external queue, vault, KMS, worker service, or dependency.
- V2 observability, SLO, incident, on-call, GitOps, or multi-tenant work.
- Replacing the existing Arco-based business UI or shared import/export controls.

## Assumptions And Open Questions

- MySQL 8 is the production database contract.
- The first worker implementation is database-backed and same-process; the durable
  attempt and lease contracts allow a later standalone worker without changing HTTP DTOs.
- Credential references use an Ops-owned encrypted credential table and never return plaintext.
- Namespace ownership is stored in the Ops database while Kubernetes remains runtime truth.
- External production credentials and live-cluster authority are unavailable; those
  actions remain human-gated runtime acceptance steps.

## Minimum Viable Approach

- Reuse the existing versioned migration runner, GORM transaction patterns,
  `pkg/impexp`, `ImportCsvButton`, export helper, K8s client factory, and business capabilities.
- Use database row claims and leases instead of adding Redis or a queue dependency.
- Extend existing Deploy and K8s modules vertically; do not create a generic platform task center.

## Structural Scope

- Affected subgraph: migration runner -> business schema; Deploy handler -> service ->
  attempt/worker -> SSH executor -> CMDB/Service capabilities; K8s handler -> owner
  services -> client-go; business list UI -> existing import/export APIs.
- Boundary crossings: Deploy to CMDB/Service typed capabilities; K8s to BizScope typed
  capability; browser to business APIs; migration runner to Ops-owned tables.
- Risk nodes: migration idempotency, task claim/cancel races, credential plaintext,
  namespace cross-scope mutation, permission seed drift, CSV transactional import.
- Graph focus: no cross-owner table access and no new dependency cycle.

## Execution Batches

1. Legacy migration and schema tests.
2. Deploy attempts and worker lifecycle.
3. K8s ownership, credential, conflict, and permission contracts.
4. Deploy import/export backend and UI.
5. Runtime/visual verification, release evidence, review, and task metadata cleanup.

## Success Criteria

- Legacy schema upgrades reach the current models on MySQL 8, are repeatable, and
  have explicit rollback limits.
- Deploy Start returns after durable enqueue; worker attempts are leased, retryable,
  timeout-bounded, cancellable, and reconciled after stale claims.
- K8s writes require a visible Namespace binding; stored credentials are referenced,
  encrypted, versioned, rotatable, and never returned; stale writes are rejected.
- Deploy import/export endpoints, permissions, i18n, UI, and tests are complete.
- Targeted and aggregate Go race, vet, frontend lint/type-check/build, overlay,
  boundary, smoke, runtime, and visual gates either pass or have a precise external gap.

## Human Gates And Stop Conditions

- Stop before destructive/ambiguous production data conversion.
- Stop before using production database, SSH, browser, or Kubernetes credentials.
- Stop if a required generic IAM, audit, migration-runner, or shared UI change belongs in `pantheon-base`.

## Linkage

- Task packet: `docs/harness/tasks/2026-08-19-ops-v1-backlog-closeout.task.md`
- Manifest: `.harness/tasks/2026-08-19-ops-v1-backlog-closeout/manifest.json`
- Status: `.harness/tasks/2026-08-19-ops-v1-backlog-closeout/STATUS.md`
- Evidence: `.harness/evidence/2026-08-19-ops-v1-backlog-closeout/`
- Review: `.harness/evidence/2026-08-19-ops-v1-backlog-closeout/review.md`
- Plan anchors: `docs/designs/BUSINESS_DEVELOPMENT_PLAN.md`,
  `docs/designs/BUSINESS_QA_ACCEPTANCE_PLAN.md`, and
  `docs/designs/BUSINESS_TEST_PLAN.md`

