---
title: Remediate SonarCloud source encoding warning
doc_type: Remediation
layer: platform
status: Active
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
updated_at: 2026-08-05
---

# Task Packet: 2026-08-03-sonarcloud-encoding-warning

## Goal

Remove confirmed replacement-character corruption and prevent recurrence, then verify the warning disappears from the merged main analysis.

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

- Restore two corrupted Harness documents
- Detect U+FFFD in the strict encoding gate
- Add focused regression coverage
- Verify hosted SonarCloud after merge

### Out

- Product runtime behavior
- SonarCloud quality profile changes
- Analysis exclusions

## Structural Scope

- Affected Subgraph: none recorded
- Boundary Crossings: none recorded
- Risk Nodes: none recorded
- Graph Focus: none recorded

## Expected Files

### Create

- `.harness/tasks/2026-08-03-sonarcloud-encoding-warning/manifest.json`
- `.harness/evidence/2026-08-03-sonarcloud-encoding-warning/commands.json`
- `.harness/evidence/2026-08-03-sonarcloud-encoding-warning/review.md`
- `docs/harness/tasks/2026-08-03-sonarcloud-encoding-warning.task.md`

### Modify

- none recorded in the historical manifest

### Do Not Touch

- the Out scope remains authoritative

## Implementation Notes

- Retrospective schema normalization only; implementation detail remains in the manifest, evidence, and review.

## Verification Plan

- `npm run check:harness-encoding`
- `node --test tests/scripts/harness-check-encoding.test.mjs`
- `npm run check:docs-frontmatter`
- `npm run check:harness-docs`
- `GitHub required checks`
- `SonarCloud main analysis`

## Linkage

- Task ID: `2026-08-03-sonarcloud-encoding-warning`
- Task Manifest: `.harness/tasks/2026-08-03-sonarcloud-encoding-warning/manifest.json`
- OpenSpec Change: `none`
- Superpowers Plan: `none`
- Plan References: `docs/harness/tasks/2026-08-03-sonarcloud-encoding-warning.task.md`
- Evidence Directory: `.harness/evidence/2026-08-03-sonarcloud-encoding-warning/`
- Review File: `.harness/evidence/2026-08-03-sonarcloud-encoding-warning/review.md`

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
