---
title: Close Dependabot PR governance workflow drift
doc_type: Remediation
layer: platform
status: Active
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
updated_at: 2026-08-05
---

# Task Packet: 2026-08-03-dependabot-pr-governance

## Goal

Make the PR automation governance prerequisite apply the same Dependabot policy as Docs Governance, then prove existing dependency PRs can pass without a maintainer bypass.

## Primary Layer

platform

## Dependency Layers

- none recorded in the historical manifest

## Harness Profile

- Template: admin-platform
- Overlay: pantheon-base
- Quality Profile: none
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

- Align pr-automation Dependabot handling with quality.yml
- Preserve the successful governance output required by auto-merge
- Add focused workflow regression coverage
- Re-run and merge the existing Dependabot PRs

### Out

- Product runtime behavior
- Weakening governance for human or agent-authored PRs
- Using the solo-override maintainer bypass

## Structural Scope

- Affected Subgraph: none recorded
- Boundary Crossings: none recorded
- Risk Nodes: none recorded
- Graph Focus: none recorded

## Expected Files

### Create

- `.harness/tasks/2026-08-03-dependabot-pr-governance/manifest.json`
- `.harness/evidence/2026-08-03-dependabot-pr-governance/commands.json`
- `.harness/evidence/2026-08-03-dependabot-pr-governance/review.md`
- `docs/harness/tasks/2026-08-03-dependabot-pr-governance.task.md`

### Modify

- none recorded in the historical manifest

### Do Not Touch

- the Out scope remains authoritative

## Implementation Notes

- Retrospective schema normalization only; implementation detail remains in the manifest, evidence, and review.

## Verification Plan

- `node --test tests/scripts/pr-automation-workflow.test.mjs`
- `node --test tests/scripts/quality-workflow.test.mjs`
- `npm run check:pr-governance`
- `npm run check:harness-encoding`
- `GitHub required checks on the governance fix PR`
- `PR Governance Prereq on Dependabot PRs`

## Linkage

- Task ID: `2026-08-03-dependabot-pr-governance`
- Task Manifest: `.harness/tasks/2026-08-03-dependabot-pr-governance/manifest.json`
- OpenSpec Change: `none`
- Superpowers Plan: `none`
- Plan References: `docs/harness/tasks/2026-08-03-dependabot-pr-governance.task.md`
- Evidence Directory: `.harness/evidence/2026-08-03-dependabot-pr-governance/`
- Review File: `.harness/evidence/2026-08-03-dependabot-pr-governance/review.md`

## Evidence Required

- command result summary or explicit historical transcript gap
- runtime or visual evidence when applicable, otherwise an explicit gap
- linked findings-first review

## Human Gates

- required GitHub checks and repository merge protection

## Sync Expectation

Pantheon Base only unless the manifest explicitly requires a later Ops foundation upgrade.

## Completion Checklist

- [x] Layer and boundary declared
- [x] Quality profile or explicit none declared
- [x] Contract anchors read
- [x] Verification run or exception recorded
- [x] Evidence saved or summarized
- [x] Review completed
