# Task Packet: 2026-09-01-release-docs-reconciliation

## Goal

Align public Base documentation and release metadata references with the published `pantheon-base-v0.10.25` release.

## Primary Layer

platform

## Dependency Layers

- public documentation
- released foundation artifact identity

## Harness Profile

- Template: api-service
- Overlay: pantheon-base
- Quality Profile: ci-workflow
- Portable Failure Class: method-health-gap
- Owner Layer: consumer-repository
- Coverage Dimensions:
  - behaviour
  - maintainability
  - method-health

## Contract Anchors

- `AGENTS.md`
- `DESIGN.md`
- `README.md`
- `docs/designs/FOUNDATION_RELEASE_MODEL.md`

## Scope

### In

- README and changelog release references
- Foundation release model documentation
- Release identity and consumer-impact statements

### Out

- Product code
- GitHub release assets
- `pantheon-ops` changes
- Release publication or tag mutation

## Expected Files

### Create

- `.harness/tasks/2026-09-01-release-docs-reconciliation/manifest.json`
- `.harness/evidence/2026-09-01-release-docs-reconciliation/`

### Modify

- `README.md` release and branch-policy references
- `CHANGELOG.md` release entries
- the bilingual foundation release model documentation

### Do Not Touch

- product code and runtime behavior
- published GitHub release assets and tags
- `pantheon-ops` sources

## Implementation Notes

- Reconcile the wording against the actual published release identity; do not restate a version the repository does not carry.
- Keep the bilingual documents consistent when either side is edited.

## Verification Plan

- `npm run check:docs-frontmatter`
- `npm run check:harness-docs`
- `git diff --check`

## Linkage

- Task ID: `2026-09-01-release-docs-reconciliation`
- Task Manifest: `.harness/tasks/2026-09-01-release-docs-reconciliation/manifest.json`
- OpenSpec Change: `none`
- Superpowers Plan: `none`
- Plan References: `docs/harness/tasks/2026-09-01-release-docs-reconciliation.task.md`
- Evidence Directory: `.harness/evidence/2026-09-01-release-docs-reconciliation/`
- Review File: `.harness/evidence/2026-09-01-release-docs-reconciliation/review.md`

## Evidence Required

- documentation gate results
- diff of the reconciled release references
- review summary

## Human Gates

- GitHub required checks and PR merge.

## Completion Checklist

- [x] Layer and boundary declared
- [x] Contract anchors read
- [x] Verification run or exception recorded
- [x] Evidence saved or summarized
- [x] Review completed
