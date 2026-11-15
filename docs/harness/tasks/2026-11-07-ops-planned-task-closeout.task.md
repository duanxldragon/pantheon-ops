---
title: Execute open Ops planning tasks
doc_type: Remediation
layer: business
status: Active
updated_at: 2026-11-07
linked_contracts:
  - docs/PROJECT_INHERITANCE.md
---

# Task Packet: 2026-11-07-ops-planned-task-closeout

## Goal

Execute locally available work in the formal Ops queue against official Base v0.18.6, fix remaining business outbound connection gaps, and keep unavailable release dependencies and production acceptance explicit.

## Primary Layer

business/*

## Workspace Context

- Target Repository: `pantheon-ops`
- Repository Role: `business-consumer`
- Upstream Dependencies: `pantheon-base`, `pantheon-harness`
- Downstream Consumers: `none`
- Sync Expectation: `not-required`
- Release Requirement: `none`

## Dependency Layers

- business/observability, business/k8s, business/cmdb
- inherited Base security/ssrf and operational component contracts

## Harness Profile

- Template: custom
- Overlay: pantheon-ops
- Quality Profile: runtime-quality
- Portable Failure Class: repo-quality-gate
- Owner Layer: consumer-repository
- Coverage Dimensions:
  - behaviour
  - architecture-fitness
  - runtime-quality
  - method-health

## Contract Anchors

- `AGENTS.md`
- `docs/PROJECT_INHERITANCE.md`
- `docs/designs/BUSINESS_OPERATIONAL_WORKBENCH_DESIGN.md`
- `docs/designs/BUSINESS_CMDB_MODULE_DESIGN.md`
- `docs/designs/BUSINESS_ERROR_SEMANTICS_APPENDIX.md`

## Scope

### In

- Add the consumer-owned 000033 missing core table baseline through businessMigrations; test fresh/replay/existing-data preservation in isolated MySQL.

- Implement scoped O2/O5 target/step log compositions and O4 UTC/stop query controls, reusing released Base APIs/components.
- Reconcile O1-O7 and D1-D5 against current released contracts and evidence.
- Reuse the released checked dial in Kubernetes and SMTP business clients.
- Meaningful outbound boundary regression tests; current build/race and overlay/rebuild checks.
- D5 active task lifecycle and linkage repairs without changing gates.
- Isolated local CMDB smoke and host import/export acceptance when available.

### Out

- Base/platform/system source overrides and new dependencies.
- Production mutations, tenant conversion, external credentials, release/publish.
- Gate exemptions, policy waivers, rewriting unrelated inherited history.

## Expected Files

### Create

- `backend/modules/business/opsinit/migrations/000033_core_business_schema.up.sql`
- `backend/modules/business/opsinit/migrations/000033_core_business_schema.down.sql`

- `backend/modules/business/k8s/pkg/client_factory_test.go`
- `backend/modules/business/observability/notification_smtp.go`
- `backend/modules/business/observability/notification_network_test.go`
- `docs/harness/tasks/2026-11-07-ops-planned-task-closeout.task.md`
- `.harness/tasks/2026-11-07-ops-planned-task-closeout/manifest.json`
- `.harness/evidence/2026-11-07-ops-planned-task-closeout/commands.json`
- `.harness/evidence/2026-11-07-ops-planned-task-closeout/summary.md`
- `.harness/evidence/2026-11-07-ops-planned-task-closeout/review.md`

- `frontend/tests/smoke/business/deploy/deploy-workbench.spec.ts`
- `frontend/tests/smoke/business/observability-workbench.spec.ts`
- `frontend/src/modules/business/observability/log-explorer.css`

### Modify

- `business-overlay.json`

- `backend/modules/business/k8s/pkg/client_factory.go`
- `backend/modules/business/observability/notification_service.go`
- original Ops planning task status/queue, business design and linked evidence

- `backend/modules/business/cmdb/host/host_state_version_migration_test.go`
- `backend/modules/business/observability/datasource_service.go`
- `backend/modules/business/observability/datasource_service_test.go`
- `backend/modules/business/observability/observability_handler.go`
- `backend/modules/business/observability/seed.go`
- `backend/modules/business/observability/seed_test.go`
- `frontend/src/modules/business/observability/LogExplorer.tsx`
- `frontend/src/modules/business/observability/api.ts`
- `frontend/src/modules/business/v12locales/zh-CN.json`
- `frontend/src/modules/business/v12locales/en-US.json`
- `frontend/tests/smoke/business/deploy/deploy-pages.spec.ts`
- `frontend/src/i18n/resources/business/zh-CN.ts`
- `frontend/src/i18n/resources/business/en-US.ts`

### Do Not Touch

- shared source under `backend/pkg/**` (business-derived assembler outputs are allowed), `backend/modules/system/**`, `frontend/src/core/**`
- inherited shared source, shared checker configuration and allowlists
- external production systems

## Implementation Notes

- Clone the existing validator for SMTP so protocol settings do not mutate webhook settings.
- Bound SMTP connection and protocol I/O, preserve TLS certificate validation and valid configured ports.
- Kubernetes validates each dial and request; configured proxies are rejected because they bypass the direct checked target contract.
- The design parent is complete as a design artifact; its implementation children are separate and cannot inherit that completion.

## Assumptions and Open Questions

- Confirmed Facts: official Base lock is v0.18.6; the historical AST index is stale for some business files.
- Working Assumptions: existing direct public-endpoint policy stays in force; no new private-network exception is approved.
- Open Questions: none for local work; external endpoint/policy and tenant production evidence remain unresolved gates.

## Minimum Viable Approach

- Selected Rung: reuse
- Why This Is Enough: use released Base checked HTTP transport dial with standard SMTP/Kubernetes client interfaces.
- Upgrade Trigger: private-network/proxy support requires explicit upstream policy and a separate reviewed release.

## Success Criteria

- Private and metadata targets cannot reach SMTP or Kubernetes side effects.
- Current local checks and isolated runtime results are recorded with exact logs.
- Every formal child has factual status; no unsupported production or UI completion claim.

## Structural Scope

- Affected Subgraph: `business client -> Base ssrf validator -> network dial -> external endpoint`
- Boundary Crossings: business configuration to external network; Base contract consumed by Ops
- Risk Nodes: Kubernetes dial/redirect/proxy, SMTP dial/TLS/auth, host CSV import
- Graph Focus: sensitive-input-flow

## Context Strategy

- Retrieval Order: formal parent -> current lock -> local AST index -> identified current files -> verification
- Retrieval Helpers: CodeGraph SQLite index read-only; exposed MCP/CLI unavailable
- Promotion Target: original task status and this evidence summary
- Response Budget: terse
- Sensitive Context: no credentials, production data or unredacted log payloads in evidence

## Delivery Governance

- Design Gate: original O/D packets and immutable Base contract
- Development Gate: scoped red/green tests and business ownership
- QA Acceptance Gate: full CGO race, business rebuild and isolated smoke
- GitHub Governance Gate: not requested; local required checks recorded

## Execution Roles

- Implementer Posture: self-execution after installed executor returned repeated service 503
- Reviewer Posture: findings-first self-review; independent reviewer unavailable

## Verification Plan

### Backend

- Windows `D:\msys64\mingw64\bin`, `CGO_ENABLED=1`; `go test -mod=readonly -race ./...`
- `go vet ./...`
- `go test -race ./modules/business/k8s/pkg ./modules/business/observability`

### Frontend

- `npm run lint`, `npm run type-check`, `npm run build`
- business overlay contract/tests and clean rebuild

### Browser / Smoke

- isolated CMDB/CSV smoke; serial Deploy and Loki workbench browser states with real login/business fixtures and explicit transport fixtures; desktop/mobile and light/dark screenshots

### Runtime Evidence

- direct network boundary rejection tests and isolated MySQL/Redis/backend runtime
- external Kubernetes, SMTP and observability positive acceptance remains a gap without endpoints

## Linkage

- Task ID: `2026-11-07-ops-planned-task-closeout`
- Task Manifest: `.harness/tasks/2026-11-07-ops-planned-task-closeout/manifest.json`
- OpenSpec Change: `none`
- Superpowers Plan: `none`
- Plan References: `.harness/tasks/2026-11-07-ops-planned-task-closeout/plan.md`
- Evidence Directory: `.harness/evidence/2026-11-07-ops-planned-task-closeout/`
- Review File: `.harness/evidence/2026-11-07-ops-planned-task-closeout/review.md`

## Evidence Required

- commands, red/green boundary tests, current frontend/backend and rebuild results
- original task dispositions and precise inherited/global gate failures
- findings-first review, unavailable independent-review and external acceptance gaps

## Human Gates

- production endpoints/credentials, outbound/private-network exception, tenant G2/G3
- Base release required to complete missing shared workbench functionality
- final actual visual/functional acceptance and any production release

## Completion Checklist

- [x] Layer and boundary declared
- [x] Contract anchors read
- [ ] Verification run or exception recorded
- [ ] Evidence saved or summarized
- [ ] Review completed

## Current Execution Status

In-progress. Business network and versioned core-schema repairs are verified locally; O1/O3 implementation and the remaining O2/O4/O5 acceptance, production external/network/tenant gates and Base-first permission/document/Dashboard requirements remain open. Generated migration destinations are materialized from the checked formal business assembler only. Independent installed role agents were unavailable; review is a self-review.

## Scoped Workbench Evidence

O2/O5 reuse released step/log primitives, bounded provider logs, stable SSE lifecycle and source isolation. O4 propagates browser stop through request context to the upstream Loki query, with explicit UTC bounds and response budget. Local provider tests and browser transport fixtures are distinct from external acceptance. The shared Base step rail renders a white button surface in dark mode and emits raw status text; this remains a Base-first visual/i18n gap. No overall workbench acceptance or independent review is asserted.

## SSH Transport and Label Budget Continuation

Business-only O2 SSH connection/session cancellation and buffer-join repairs are verified with the actual local SSH protocol and isolated MySQL business execution. O4 row-label previews are bounded with localized truncation notices and retained raw detail; source/runtime/rendered evidence is linked from the current summary. Private factory context propagation does not alter public contracts, schema, engine or credential storage. Independent final review remains pending after a bounded supplied-source CLI timeout. Real VM/datasource/cluster, repaired Base release/consumption and final signed acceptance remain open.

O2 checkpoint-history candidate: business migration35 and append-only safe observations commit atomically with existing checkpoint updates, without new routes or engine policy changes. Existing Task DTO exposes at most500 latest step/attempt observations with truncation; CMDB host visibility filters detail hosts/current steps/log-attempt/history metadata. Historical views state recorded capture-time provenance and preserve explicit missing attempts. Legacy state is not reconstructed. Scope, failed-before regressions and pending final validation are recorded under the same task/evidence; independent/production/formal acceptance stays open.

## 2026-11-12 Progress

CMDB key-only acceptance R3 executed publicly and **failed** at the revocation stage: 69 requests, 57 passing assertions, failure `exact current CMDB permission keys without wildcard` when the original session's `/auth/me` did not match the expected empty CMDB key set after role35 was publicly emptied and read back. Product-defect-versus-oracle is unresolved and the mutation is not replayed; R3 roles35-38/users26-29, R2 roles31-34/users22-25 and the host1 fixture are retained.

O2 shared-permission product repair design-r3 has independent CODE design APPROVE and ARCH design APPROVE/CLEAR (design only). The isolated Ops draft covers 16 business permission keys and 27 routes with init-first registration and six static GET guards; preimage RED27 -> draft GREEN0, with full native compile/tests waiting on a new normal Base candidate. O4 complete-positive runner r1 was rejected by independent CODE for three real runner defects (activation-only claim, unbounded build/run logs, lost logs on timeout/start failure) and is not runnable; independent r2 is in progress with failed-before witnesses retained. D2 public adapter closeout is active with strict TS and 27 pure oracles passing and no HTTP/SQL/SSH executed. Velero stays deferred by maintainer order.

All six P0 parents O2/O3/O4/D1/D2/D3 remain open; no official lock change, formal release consumption, independent final approval or visual/functional sign-off is claimed.
