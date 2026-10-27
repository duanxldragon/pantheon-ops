---
title: Remediate all open SonarCloud code smells on main
doc_type: Remediation
layer: platform
status: Active
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
updated_at: 2026-08-05
---

# Task Packet: 2026-08-02-sonarcloud-open-issues

## Goal

Reduce the 77 unresolved SonarCloud code smells on main to zero without changing public behavior, then merge and restore main-only Git state.

## Primary Layer

platform

## Dependency Layers

- system/auth
- system/iam
- system/org
- system/config
- lowcode
- frontend

## Harness Profile

- Template: admin-platform
- Overlay: pantheon-base
- Quality Profile: ci-workflow, ui-runtime, permission-policy, generator
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
- `docs/README.md`

## Scope

### In

- 53 go:S3776 findings
- 18 typescript:S3776 findings
- 3 go:S107 findings
- 2 typescript:S6759 findings
- 1 typescript:S1135 finding
- Focused tests and evidence required by each refactor

### Out

- Public API or DTO behavior changes
- Database schema or seed changes
- Permission, menu, authentication, or audit contract changes
- New dependencies
- Pantheon Ops synchronization
- Visual redesign

## Structural Scope

- Affected Subgraph: none recorded
- Boundary Crossings: none recorded
- Risk Nodes: none recorded
- Graph Focus: none recorded

## Expected Files

### Create

- `.harness/evidence/2026-08-02-sonarcloud-open-issues/pr-body.md`
- `tests/scripts/create-pr.test.mjs`

### Modify

- `.agents/skills/repo-pr-gate/SKILL.md`
- `.github/workflows/quality.yml`
- `.harness/evidence/2026-08-02-sonarcloud-open-issues/review.md`
- `package.json`
- `scripts/create-pr.mjs`
- `tests/scripts/quality-workflow.test.mjs`

### Do Not Touch

- the Out scope remains authoritative

## Implementation Notes

- SonarCloud main quality gate remains OK
- SonarCloud unresolved issue count is zero
- All required GitHub checks pass
- No behavior, schema, permission, menu, or UI layout regression is introduced
- The remediation PR is merged and local/remote Git state returns to main-only

## Verification Plan

- strict Harness checks and the linked evidence record

## Linkage

- Task ID: `2026-08-02-sonarcloud-open-issues`
- Task Manifest: `.harness/tasks/2026-08-02-sonarcloud-open-issues/manifest.json`
- OpenSpec Change: `none`
- Superpowers Plan: `none`
- Plan References: `docs/harness/tasks/2026-08-02-sonarcloud-open-issues.task.md`
- Evidence Directory: `.harness/evidence/2026-08-02-sonarcloud-open-issues/`
- Review File: `.harness/evidence/2026-08-02-sonarcloud-open-issues/review.md`

## Evidence Required

- command result summary or explicit historical transcript gap
- runtime or visual evidence when applicable, otherwise an explicit gap
- linked findings-first review

## Human Gates

- Required GitHub checks and repository merge protection

## Sync Expectation

Pantheon Base only; no base-to-ops sync because behavior and contracts are unchanged.

## Completion Checklist

- [x] Layer and boundary declared
- [x] Quality profile or explicit none declared
- [x] Contract anchors read
- [x] Verification run or exception recorded
- [x] Evidence saved or summarized
- [x] Review completed
