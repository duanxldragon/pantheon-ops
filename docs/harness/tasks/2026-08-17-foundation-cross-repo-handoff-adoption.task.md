---
title: Adopt cross-repository handoff evidence in the foundation source
doc_type: Remediation
layer: inheritance-sync
status: Active
updated_at: 2026-08-17
linked_contracts:
  - docs/acceptances/TASK_PACKET_BASE_TEMPLATE.md
  - docs/designs/FOUNDATION_RELEASE_MODEL.md
---

# Task Packet: 2026-08-17-foundation-cross-repo-handoff-adoption

## Goal

Adopt the Harness workspace-context and stateless handoff contract in Pantheon Base without publishing a release or modifying downstream consumers.

## Priority

`high`

## Estimated Complexity

`moderate`

## Primary Layer

inheritance-sync

## Workspace Context

- Target Repository: pantheon-base
- Repository Role: foundation-source
- Upstream Dependencies: pantheon-harness
- Downstream Consumers: pantheon-ops
- Sync Expectation: required
- Release Requirement: foundation-release

## Dependency Layers

- method-source: pantheon-harness
- business-consumer: pantheon-ops

## Dependencies

- blockedBy: 2026-08-17-cross-repo-handoff-evidence-strength
- blocks: pantheon-ops handoff-plan adoption and consumer lock update

## Harness Profile

- Template: admin-platform
- Overlay: pantheon-base
- Quality Profile: ci-workflow
- Portable Failure Class: task-boundary-gap
- Owner Layer: consumer-template
- Coverage Dimensions:
  - maintainability
  - architecture-fitness
  - method-health

## Contract Anchors

- `AGENTS.md`
- `docs/acceptances/TASK_PACKET_BASE_TEMPLATE.md`
- `docs/acceptances/TASK_PACKET_BASE_TEMPLATE.en.md`
- `docs/designs/FOUNDATION_RELEASE_MODEL.md`
- `scripts/harness/check-task-packet.mjs`

## Scope

### In

- Add conditional Workspace Context validation for `inheritance-sync` task packets.
- Add Base-specific foundation release and consumer sync handoff fields to the bilingual task template.
- Add template marker regression coverage.
- Create a task manifest, status, evidence, review, and handoff package that a stateless tool can resume.

### Out

- No backend, frontend, database, permission, menu, i18n, or runtime behavior changes.
- No Harness version publication.
- No Base foundation release publication.
- No Pantheon Ops file or consumer lock changes.

## Assumptions and Open Questions

- Confirmed Facts: Pantheon Base is the foundation source; Pantheon Ops is a downstream business consumer; release/lock updates are explicit gates.
- Working Assumptions: the Harness contract will be released before downstream consumers are required to enforce it.
- Open Questions: none

## Structural Scope

- Affected Subgraph: `Harness method contract -> Base task template/checker -> foundation handoff -> Ops adoption`
- Boundary Crossings: `method-source -> foundation-source -> business-consumer`
- Risk Nodes: `historical task compatibility, task manifest linkage, release state claims`
- Graph Focus: `none`

## Expected Files

### Create

- `docs/harness/tasks/2026-08-17-foundation-cross-repo-handoff-adoption.task.md`
- `.harness/tasks/2026-08-17-foundation-cross-repo-handoff-adoption/manifest.json`
- `.harness/state/2026-08-17-foundation-cross-repo-handoff-adoption/status.md`
- `.harness/evidence/2026-08-17-foundation-cross-repo-handoff-adoption/summary.md`
- `.harness/evidence/2026-08-17-foundation-cross-repo-handoff-adoption/commands.json`
- `.harness/evidence/2026-08-17-foundation-cross-repo-handoff-adoption/review.md`
- `.harness/evidence/2026-08-17-foundation-cross-repo-handoff-adoption/handoff.md`

### Modify

- `docs/acceptances/TASK_PACKET_BASE_TEMPLATE.md`
- `docs/acceptances/TASK_PACKET_BASE_TEMPLATE.en.md`
- `scripts/check-task-packet-template.mjs`
- `scripts/harness/check-task-packet.mjs`
- `tests/scripts/check-task-packet-template.test.mjs`

### Do Not Touch

- `backend/`
- `frontend/`
- `database/`
- `pantheon-ops/`
- release tags, release assets, and consumer locks

## Implementation Notes

- Preserve Base's Task Manifest linkage and existing checker behavior.
- Require Workspace Context only for `inheritance-sync`; do not invalidate historical single-repository packets.
- Record the actual release/sync state. A local source change is not a published foundation release.
- Cross-module or cross-repository runtime data access remains API/contract based; this governance task does not add table access or product APIs.

## Minimum Viable Approach

- Selected Rung: small local code
- Why This Is Enough: existing Markdown templates and checker logic can represent and validate the boundary without new dependencies.
- Upgrade Trigger: none

## Success Criteria

- Behaviour Outcome: Base inheritance-sync packets fail without complete workspace context, while existing ordinary packets remain valid.
- Verification Signal: task-template test, targeted Base task/evidence/review checks, and strict Harness adoption/sync gates pass.
- Regression Watch: existing Task Manifest semantics and foundation release tooling remain unchanged.
- Economics Watch: handoff artifacts contain enough context for a lower-cost stateless tool to resume without chat history.

## Context Strategy

- Entry Sources: `AGENTS.md`, this task packet, Base task template, Harness handoff artifact
- Retrieval Order: `entry -> status/handoff -> task packet -> evidence -> raw checker source`
- Retrieval Helpers: none
- Promotion Target: Base task template and handoff artifact
- Response Budget: terse
- Sensitive Context: none

## Method Readiness

- Consumer-Specific Controls: `pantheon-base` task manifest, checker, and foundation release model
- Required Sensors: command | review
- Required Evidence: command summary | review summary | release/sync gap
- Minimal Complexity Rung: minimum-new-code
- Ratchet Decision: gate-updated
- Deferred Code Issues: Harness method release, Base foundation release, and Ops consumer lock update remain separate explicit gates

## Delivery Governance

- Design Gate: foundation ownership and release boundary documented
- Development Gate: governance files only; product directories excluded
- QA Acceptance Gate: checker tests and strict repository gates
- GitHub Governance Gate: repo-quality-gate

## Execution Roles

- Implementer Posture: docs-only
- Reviewer Posture: architecture

## Stop Points

- Stop before publishing a Harness method release or Base foundation release.
- Stop before editing Pantheon Ops or updating a consumer lock.

## State Plan

- Checkpoint Expectation: `.harness/state/2026-08-17-foundation-cross-repo-handoff-adoption/status.md`
- Resume Artifacts: task packet, manifest, status, evidence summary, review, and handoff

## Verification Plan

- `npm run check:task-packet-template`
- `node --test tests/scripts/check-task-packet-template.test.mjs tests/scripts/harness-check-task-packet-context.test.mjs`
- `node scripts/harness/check-task-packet.mjs --root . docs/harness/tasks/2026-08-17-foundation-cross-repo-handoff-adoption.task.md`
- `node scripts/harness/check-evidence.mjs --root . --strict .harness/evidence/2026-08-17-foundation-cross-repo-handoff-adoption/commands.json`
- `node scripts/harness/check-review.mjs --root . --strict .harness/evidence/2026-08-17-foundation-cross-repo-handoff-adoption/review.md`
- `npm run check:harness-method`
- `npm run check:harness-adoption`
- `npm run check:harness-template`
- `npm run check:harness-docs`
- `npm run check:harness-sync`
- `npm run check:harness-encoding`
- `git diff --check`

## Linkage

- Task ID: `2026-08-17-foundation-cross-repo-handoff-adoption`
- Task Manifest: `.harness/tasks/2026-08-17-foundation-cross-repo-handoff-adoption/manifest.json`
- OpenSpec Change: `none`
- Superpowers Plan: `none`
- Plan References: `none`
- Evidence Directory: `.harness/evidence/2026-08-17-foundation-cross-repo-handoff-adoption/`
- Review File: `.harness/evidence/2026-08-17-foundation-cross-repo-handoff-adoption/review.md`

## Evidence Required

- targeted checker test output
- strict Harness gate output
- review summary
- explicit method/foundation release and consumer lock gaps

## Human Gates

- Harness method release publication
- Base foundation release publication
- Pantheon Ops consumer lock update

## Foundation Release Handoff

- Shared change owner: `pantheon-base`
- Foundation release required: `deferred`
- Consumer sync status: `not-started`
- Downstream validation command: `npm run check:harness-adoption`
- Release/lock stop point: immutable Harness release -> Base foundation release -> Ops lock update

## Completion Checklist

- [x] Layer and boundary declared
- [x] Quality profile or explicit `none` declared
- [x] Ratchet decision declared for repeated failures
- [x] Delivery governance gates declared
- [x] Contract anchors read
- [x] Tests or checks updated
- [x] Verification run or exception recorded
- [x] Evidence saved or summarized
- [x] Docs updated if contracts changed
- [x] Review completed
