---
title: Close Ops delivery readiness gaps
doc_type: Task
layer: business/*
status: Planned
updated_at: 2026-09-28
linked_contracts:
  - docs/PROJECT_INHERITANCE.md
  - docs/runbooks/TENANT_MIGRATION_RUNBOOK.md
  - docs/runbooks/PRODUCTION_BACKUP_RESTORE_DRILL_G2.md
---

# Task Packet: 2026-09-28-ops-delivery-readiness-gap-closure

## Goal

把当前仓库“本地 RC-ready”与“可以直接生产交付”之间的差距拆成可续跑的证据任务，并保留 Base-first、外部系统、SSRF 和租户迁移的真实门禁。

## Priority

`P0`

## Estimated Complexity

`epic`

## Primary Layer

business/*

## Workspace Context

- Target Repository: `pantheon-ops`
- Repository Role: `business-consumer`
- Upstream Dependencies: `pantheon-harness`, `pantheon-base`
- Downstream Consumers: `none`
- Foundation Lock: `pantheon-base-v0.13.0` / `c254e4c8c63f4409d49752eedb097291627d0b82`
- Sync Expectation: `required`
- Release Requirement: `none`

## Dependency Layers

- foundation-source: `pantheon-base`
- business domains: `business/*`
- runtime dependencies: Prometheus/VictoriaMetrics, Loki, Tempo, Kubernetes, ArgoCD, Velero
- migration dependencies: DBA backup access and approved maintenance window

## Dependencies

- blockedBy: production-like endpoints/credentials, SSRF policy decision, and tenant G2/G3 evidence when multi-tenant migration is in scope
- blocks: direct production approval and any claim of multi-tenant production readiness

## Harness Profile

- Template: custom
- Overlay: business-overlay
- Quality Profile: release-sensitive
- Portable Failure Class: runtime-evidence-gate
- Owner Layer: consumer-repository
- Coverage Dimensions:
  - behaviour
  - maintainability
  - architecture-fitness
  - runtime-quality
  - method-health

## Contract Anchors

- `AGENTS.md`
- `docs/PROJECT_INHERITANCE.md`
- `foundation-release.lock.json`
- `business-overlay.json`
- `.harness/evidence/2026-08-31-ops-v1-2-production-readiness-round2/summary.md`
- `docs/runbooks/TENANT_MIGRATION_RUNBOOK.md`
- `docs/runbooks/PRODUCTION_BACKUP_RESTORE_DRILL_G2.md`
- `docs/acceptances/BUSINESS_MODULE_ACCEPTANCE_MATRIX.md`

## Scope

### In

- production-like external acceptance with bounded, reversible test resources;
- explicit SSRF and outbound-network policy handoff to Base where generic;
- current foundation lock and overlay ownership re-review;
- tenant G2/G3 evidence as a separately gated migration track;
- active Harness, generated-artifact, docs-link, and UI-quality gate triage;
- evidence, estimates, and maintainer stop points.

### Out

- unapproved production mutation or DDL;
- new business feature implementation;
- Ops-local copies or overrides of Base-owned platform/system code;
- declaring the old v1.2 completion reports to be production evidence;
- implementing full multi-tenant migration in this planning packet.

## Assumptions and Open Questions

- Confirmed Facts: repository-local backend/frontend/overlay checks are green; the latest production review still records external, SSRF, and Base-first gates; tenant G2/G3 are open.
- Working Assumption: required endpoints and credentials will be supplied as a scoped test environment, not embedded in the repository.
- Open Questions: whether the intended release is single-tenant RC only or also includes tenant migration; which owner approves the SSRF policy; when DBA and external-system windows are available.

## Structural Scope

- Affected Subgraph: `foundation lock -> business overlay -> runtime adapters -> external systems -> evidence/release gate`
- Boundary Crossings: `pantheon-base -> pantheon-ops`, `business API -> external integration`, `migration runbook -> DBA/runtime evidence`
- Risk Nodes: `outbound URL proxy`, `credential handling`, `tenant scope`, `cleanup/idempotency`, `generated artifact ownership`
- Graph Focus: `runtime-evidence-gate`

## Expected Files

### Create

- `.harness/tasks/2026-09-28-ops-delivery-readiness-gap-closure/**`
- `.harness/evidence/2026-09-28-ops-delivery-readiness-gap-closure/**`
- `docs/harness/tasks/2026-09-28-ops-delivery-readiness-gap-closure.task.md`

### Modify

- only the owned task/evidence files after the gate owner accepts each child packet;
- Base-owned policy or shared fixes must land in `pantheon-base` and be consumed through a new foundation release.

### Do Not Touch

- `../pantheon-base/**` from this repository;
- production databases, clusters, observability endpoints, ArgoCD, or Velero;
- the existing user dirty-tree change until D5 records its ownership decision;
- unrelated historical task packets.

## Implementation Notes

- Treat old “100% complete” v1.2 summaries as historical context, not current release evidence.
- Keep direct single-tenant delivery and multi-tenant migration as separate decisions.
- Every external action needs scoped credentials, bounded timeout, cleanup, redacted evidence, and reviewer sign-off.
- Generic platform problems return to Base; business-specific adapters remain in Ops.

## Minimum Viable Approach

- Selected Rung: reuse
- Why This Is Enough: all required local checks and existing runtime evidence formats already exist; the remaining work is closure and external proof.
- Upgrade Trigger: add implementation work only when a gate reproduces a concrete product defect rather than an evidence or ownership gap.

## Success Criteria

- Behaviour Outcome: release status distinguishes local RC readiness, direct production approval, and multi-tenant migration readiness.
- Verification Signal: child packets have fresh commands, evidence, review, and explicit unknowns.
- Regression Watch: overlay ownership, permission/DataScope/audit boundaries, and current green build/test checks remain intact.
- Economics Watch: no time is spent repairing historical Harness debt inside an external acceptance window.

## Context Strategy

- Entry Sources: `AGENTS.md`, `docs/PROJECT_INHERITANCE.md`, this task packet, child queue, latest production review, tenant runbook.
- Retrieval Order: `entry -> current checks -> child packet -> runtime evidence -> review`.
- Sensitive Context: credentials, tokens, kubeconfig, URLs with secrets, and raw production data never enter `.harness`.

## Method Readiness

- Required Sensors: command, review, runtime evidence, external-gap record.
- Required Evidence: backend/frontend/business gates, external integration proof, SSRF policy, Base-first comparison, tenant G2/G3 evidence where applicable.
- Minimal Complexity Rung: reuse.
- Ratchet Decision: guide-updated.

## Delivery Governance

- Design Gate: parent packet and child queue accepted.
- Development Gate: Base dependency and overlay checks remain green.
- QA Acceptance Gate: runtime, browser, permission, cleanup, and reviewer evidence.
- GitHub Governance Gate: repo quality and PR governance checks after implementation.

## Execution Roles

- Implementer Posture: reviewer-assisted.
- Reviewer Posture: findings-first release reviewer.

## Stop Points

- production mutation, credentials, new schema/API/permission contract, Base lock update, or final visual/functional acceptance;
- unresolved cross-tenant access, SSRF bypass, cleanup failure, or inherited-source drift;
- any request to use screenshots or mocks as a substitute for real external evidence.

## State Plan

- Checkpoint Expectation: `.harness/tasks/2026-09-28-ops-delivery-readiness-gap-closure/STATUS.md`.
- Resume Artifacts: manifest, queue, child packets, commands, summary, and review.

## Verification Plan

### Backend

- `go test -count=1 ./...`
- `go vet ./...`
- Windows CGO `go test -race ./...` with `D:\msys64\mingw64\bin`.

### Frontend

- `npm run type-check`
- `npm run lint`
- `npm run build`

### Browser / Smoke

- Reuse the existing round-2 desktop/mobile screenshots as baseline only.
- New external or tenant claims require runtime evidence, not screenshot-only evidence.

### Runtime Evidence

- D1 external integration acceptance;
- D2 SSRF policy tests;
- D3 clean foundation consumer and overlay rebuild;
- D4 G2/G3 migration evidence when multi-tenant migration is requested.

## Linkage

- Task ID: `2026-09-28-ops-delivery-readiness-gap-closure`
- Task Manifest: `.harness/tasks/2026-09-28-ops-delivery-readiness-gap-closure/manifest.json`
- OpenSpec Change: `none`
- Superpowers Plan: `none`
- Plan References: `.harness/tasks/2026-09-28-ops-delivery-readiness-gap-closure/EXECUTION_QUEUE.md`
- Evidence Directory: `.harness/evidence/2026-09-28-ops-delivery-readiness-gap-closure/`
- Review File: `.harness/evidence/2026-09-28-ops-delivery-readiness-gap-closure/review.md`

## Evidence Required

- `commands.json`, `summary.md`, `review.md`;
- fresh local quality-gate outputs;
- external endpoint/credential gap or real acceptance evidence;
- SSRF policy and Base-first ownership decision;
- tenant G2/G3 records if migration is in scope;
- no secrets or raw production data.

## Human Gates

- external endpoints, credentials, and production-like test window;
- SSRF outbound policy;
- Base foundation release or upstream change;
- DBA/maintenance-window approval for tenant G2/G3;
- final visual and functional acceptance.

## Foundation Consumption

- Shared change owner: `pantheon-base`.
- Current consumer lock: `pantheon-base-v0.13.0`.
- Consumer validation: `npm run check:business-foundation`.
- Stop point: immutable Base release -> Ops lock update -> overlay/build/smoke -> acceptance.

## Completion Checklist

- [x] Layer and boundary declared
- [x] Quality profile declared
- [x] Contract anchors read
- [x] Verification run or exception recorded
- [x] Evidence saved or summarized
- [x] Human gates declared
- [x] Review completed
- [ ] Direct production approval granted
- [ ] Multi-tenant G2/G3 granted (only if that release track is selected)
