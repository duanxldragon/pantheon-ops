---
title: Prepare the pantheon-base v0.10.0 release candidate
doc_type: Remediation
layer: platform
status: Archived
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
updated_at: 2026-08-05
---

# Task Packet: 2026-08-04-base-v0-10-0-release-candidate

## Goal

Close all identified Base-side release, packaging, gate, smoke, and evidence gaps and produce a candidate that is ready for the final maintainer publication gate.

## Primary Layer

platform

## Dependency Layers

- ci-workflow
- release-tooling
- ui-runtime

## Harness Profile

- Template: admin-platform
- Overlay: pantheon-base
- Quality Profile: ci-workflow, generator, ui-runtime
- Portable Failure Class: repo-quality-gate
- Owner Layer: consumer-repository
- Coverage Dimensions:
  - behaviour
  - maintainability
  - architecture-fitness
  - runtime-quality
  - method-health

## Contract Anchors

- `AGENTS.md`
- `DESIGN.md`
- `docs/designs/FOUNDATION_RELEASE_MODEL.md`
- `docs/designs/QUALITY_AND_SECURITY_STRATEGY.md`
- `docs/harness/VERIFICATION_EVIDENCE_SPEC.md`

## Scope

### In

- Enforce pantheon-base-vX.Y.Z and matching release/X.Y identities
- Record required candidate checks in release metadata
- Make foundation bundle creation deterministic on Windows
- Bind Release Gate and publish tooling to an immutable candidate commit
- Make security API failures block release
- Fix smoke cleanup false-green and clean-template drift
- Stabilize cleanup RangePicker smoke at month boundaries
- Remove the Dashboard nested-button runtime error
- Generate v0.10.0 metadata and bundle for the accepted candidate SHA

### Out

- Any pantheon-ops change
- New Base product features
- External tag creation or GitHub Release publication before maintainer approval

## Structural Scope

- Affected Subgraph: none recorded
- Boundary Crossings: none recorded
- Risk Nodes: none recorded
- Graph Focus: none recorded

## Expected Files

### Create

- `.harness/tasks/2026-08-04-base-v0-10-0-release-candidate/manifest.json`
- `.harness/evidence/2026-08-04-base-v0-10-0-release-candidate/commands.json`
- `.harness/evidence/2026-08-04-base-v0-10-0-release-candidate/review.md`
- `docs/harness/tasks/2026-08-04-base-v0-10-0-release-candidate.task.md`

### Modify

- none recorded in the historical manifest

### Do Not Touch

- the Out scope remains authoritative

## Implementation Notes

- Retrospective schema normalization only; implementation detail remains in the manifest, evidence, and review.

## Verification Plan

- `npm run test:foundation-release`
- `npm run test:release-gate-workflow`
- `go run github.com/rhysd/actionlint/cmd/actionlint@v1.7.10 -shellcheck=""`
- `go test -count=1 ./...`
- `go vet ./...`
- `frontend npm run lint`
- `frontend npm run test:unit`
- `frontend npm run build`
- `focused cleanup RangePicker Playwright smoke`
- `Dashboard visual smoke and screenshot`
- `GitHub required checks and Full Smoke on the final candidate`

## Linkage

- Task ID: `2026-08-04-base-v0-10-0-release-candidate`
- Task Manifest: `.harness/tasks/2026-08-04-base-v0-10-0-release-candidate/manifest.json`
- OpenSpec Change: `none`
- Superpowers Plan: `none`
- Plan References: `docs/harness/tasks/2026-08-04-base-v0-10-0-release-candidate.task.md`
- Evidence Directory: `.harness/evidence/2026-08-04-base-v0-10-0-release-candidate/`
- Review File: `.harness/evidence/2026-08-04-base-v0-10-0-release-candidate/review.md`

## Evidence Required

- command result summary or explicit historical transcript gap
- runtime or visual evidence when applicable, otherwise an explicit gap
- linked findings-first review

## Human Gates

- Stop before creating pantheon-base-v0.10.0 tag or publishing the GitHub Release.

## Sync Expectation

Base-only. Ops remains unchanged until a later explicit foundation upgrade task.

## Completion Checklist

- [x] Layer and boundary declared
- [x] Quality profile or explicit none declared
- [x] Contract anchors read
- [x] Verification run or exception recorded
- [x] Evidence saved or summarized
- [x] Review completed
