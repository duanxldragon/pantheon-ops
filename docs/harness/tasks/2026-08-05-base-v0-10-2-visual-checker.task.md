---
title: Release shared visual checker fix in pantheon-base v0.10.2
doc_type: Remediation
layer: platform
status: Active
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
updated_at: 2026-08-05
---

# Task Packet: 2026-08-05-base-v0-10-2-visual-checker

## Goal

Fix exact CSS declaration matching in the Base-owned visual checker, distribute the helper through a patch foundation release, and let Pantheon Ops consume it without an Ops-only generic rule fork.

## Primary Layer

platform

## Dependency Layers

- shared UI governance
- foundation release tooling
- consumer inheritance

## Harness Profile

- Template: admin-platform
- Overlay: pantheon-base
- Quality Profile: shared-ui, foundation-release
- Portable Failure Class: inherited-checker-drift
- Owner Layer: foundation-repository
- Coverage Dimensions:
  - behaviour
  - maintainability
  - architecture-fitness
  - method-health

## Contract Anchors

- `AGENTS.md`
- `DESIGN.md`
- `docs/designs/BACKOFFICE_STYLE_CONSTRAINTS.md`
- `docs/designs/FOUNDATION_RELEASE_MODEL.md`

## Scope

### In

- Match CSS declaration property names exactly so `min-height` cannot satisfy `height`
- Add a focused regression test
- Include the shared matcher in foundation frontend assets
- Publish `pantheon-base-v0.10.2` after required checks pass

### Out

- Runtime UI styling or product behavior changes
- Database, API, permission, menu, or i18n changes
- Broad visual checker refactoring

## Structural Scope

- Affected Subgraph: visual checker -> CSS matcher -> foundation manifest -> release bundle -> Ops inheritance check
- Boundary Crossings: Base build tooling -> foundation artifact -> Ops tooling adapter
- Risk Nodes: false-positive visual gates | shared tooling drift | immutable release checksum
- Graph Focus: call-depth | sensitive-flow

## Implementation Notes

- Reuse Node standard library and the existing release bundle format.
- Add no dependency and do not change `frontend/src` runtime code.

## Expected Files

### Create

- `frontend/scripts/lib/css-declarations.mjs`
- `frontend/scripts/lib/css-declarations.test.mjs`
- task and evidence artifacts for this task ID

### Modify

- `frontend/scripts/check-shell-visual-contract.mjs`
- `scripts/foundation-release/build-release-manifest.mjs`
- focused foundation release tests

### Do Not Touch

- `frontend/src/**`
- backend, database, permission, menu, API, and i18n runtime contracts

## Verification Plan

- `node --test frontend/scripts/lib/css-declarations.test.mjs`
- `npm run test:foundation-release`
- `npm --prefix frontend run check:shell-visual-contract`
- `npm --prefix frontend run lint`
- `npm --prefix frontend run type-check`
- `git diff --check`
- hosted required checks and release checksum verification

## Linkage

- Task ID: `2026-08-05-base-v0-10-2-visual-checker`
- Task Manifest: `.harness/tasks/2026-08-05-base-v0-10-2-visual-checker/manifest.json`
- OpenSpec Change: `none`
- Superpowers Plan: `none`
- Plan References: `docs/harness/tasks/2026-08-05-base-v0-10-2-visual-checker.task.md`
- Evidence Directory: `.harness/evidence/2026-08-05-base-v0-10-2-visual-checker/`
- Review File: `.harness/evidence/2026-08-05-base-v0-10-2-visual-checker/review.md`

## Evidence Required

- exact local command results
- independent findings-first review
- hosted required checks
- immutable v0.10.2 tag, archive, and checksum identity
- downstream Ops consumption evidence

## Human Gates

- The maintainer already authorized publishing the next Base version and downstream Ops consumption.

## Completion Checklist

- [x] Layer and boundary declared
- [x] Quality profile declared
- [x] Contract anchors read
- [x] Verification run or exception recorded
- [x] Evidence saved or summarized
- [x] Review completed
- [ ] Hosted checks, release, and Ops handoff completed
