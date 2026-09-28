---
title: Reduce Pantheon Base SonarCloud duplication below three percent
doc_type: Remediation
layer: system
status: Active
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
  - docs/contracts/DOCUMENT_GOVERNANCE_CONTRACT.md
updated_at: 2026-09-01
---

# Task Packet: 2026-09-01-sonar-duplication-reduction

## Goal

Reduce the `pantheon-base` SonarCloud overall duplication from the observed `4.10%` to below `3.00%` through behavior-preserving reuse of repeated production and test patterns.

## Primary Layer

platform

## Workspace Context

- Target Repository: `pantheon-base`
- Repository Role: `foundation-source`
- Upstream Dependencies: `pantheon-harness`
- Downstream Consumers: `pantheon-ops`
- Sync Expectation: `not-required`
- Release Requirement: `none`

## Dependency Layers

- shared frontend platform components
- backend service and test helpers

## Harness Profile

- Template: admin-platform
- Overlay: pantheon-base
- Quality Profile: ci-workflow
- Portable Failure Class: method-health-gap
- Owner Layer: consumer-repository
- Coverage Dimensions:
  - behaviour
  - maintainability
  - architecture-fitness
  - method-health

## Contract Anchors

- `AGENTS.md`
- `DESIGN.md`
- `docs/designs/QUALITY_AND_SECURITY_STRATEGY.md`
- `scripts/check-duplication.mjs`

## Scope

### In

- Top Sonar duplication hotspots in frontend production modules and backend service/test packages.
- Focused type-check, lint, build, Go tests, local duplication gate, and diff validation.

### Out

- SonarCloud project settings or exclusions.
- API, schema, seed semantics, permissions, menus, Ops repository, and visual redesign.

## Expected Files

### Create

- `.harness/tasks/2026-09-01-sonar-duplication-reduction/manifest.json`
- `.harness/evidence/2026-09-01-sonar-duplication-reduction/commands.json`
- `.harness/evidence/2026-09-01-sonar-duplication-reduction/summary.md`
- `.harness/evidence/2026-09-01-sonar-duplication-reduction/review.md`

### Modify

- only files confirmed by the Sonar hotspot ranking and assigned implementation scope

### Do Not Touch

- `.sonarcloud.properties`
- `pantheon-ops/**`
- `database/**`, schema contracts, permissions, menus, i18n resource policy, and release configuration

## Implementation Notes

- Reuse existing local helpers/components and extract only repeated blocks shown by the Sonar file ranking.
- Do not add dependencies, change exclusions, or rewrite generated/fixture content to game the metric.
- Preserve i18n, authorization, audit, menu, and runtime behavior.

## Verification Plan

- Backend: focused `go test` for changed packages, then `go test ./...` if feasible.
- Frontend: `npm run type-check`, `npm run lint`, `npm run build`.
- Duplication: `npm run check:duplication` and public SonarCloud measures after CI.
- Runtime Evidence: no API/server runtime path changed; record explicit gap.

## Linkage

- Task ID: `2026-09-01-sonar-duplication-reduction`
- Task Manifest: `.harness/tasks/2026-09-01-sonar-duplication-reduction/manifest.json`
- OpenSpec Change: `none`
- Superpowers Plan: `none`
- Plan References: `docs/harness/tasks/2026-09-01-sonar-duplication-reduction.task.md`
- Evidence Directory: `.harness/evidence/2026-09-01-sonar-duplication-reduction/`
- Review File: `.harness/evidence/2026-09-01-sonar-duplication-reduction/review.md`

## Evidence Required

- local duplication gate result
- focused backend and frontend verification results
- hosted SonarCloud duplication measure after CI
- explicit runtime gap note
- review summary

## Human Gates

- Hosted SonarCloud analysis and final acceptance of any residual density above the target.

## Completion Checklist

- [x] Layer and boundary declared
- [x] Contract anchors read
- [x] Verification run or exception recorded
- [x] Evidence saved or summarized
- [x] Review completed
