# Task Packet: 2026-06-28-consume-pantheon-base-v0-8-6

## Goal

Consume `pantheon-base` `base-v0.8.6` into `pantheon-ops`, remove legacy conflicting structures left from older layouts, and restore a green backend/frontend validation path for ongoing `ops` development.

## Primary Layer

inheritance-sync/platform

## Dependency Layers

- backend/shared
- frontend/shared
- repository-governance

## Harness Profile

- Template: pantheon-ops custom
- Overlay: none
- Quality Profile: inheritance-sync
- Portable Failure Class: repo-quality-gate
- Owner Layer: inheritance-sync
- Coverage Dimensions:
  - maintainability
  - compatibility
  - release-safety

## Contract Anchors

- `AGENTS.md`
- `docs/PROJECT_INHERITANCE.md`
- `../pantheon-base/AGENTS.md`
- `../pantheon-base/README.md`

## Scope

### In

- consume `pantheon-base` `base-v0.8.6` across backend shared modules, frontend shared contracts, and release consumption scripts
- remove legacy duplicate `common`, `auth`, and `system/lowcode` structures that conflict with the new base layout
- align scaffold workspace helpers with the current generated feature-ledger contract
- restore backend tests, frontend build, and smoke-helper compatibility required by this base sync
- add governance evidence and PR body linkage required to merge this upgrade branch

### Out

- new business features in `pantheon-ops`
- follow-up cleanup of unrelated historical GitHub workflow runs
- further redesign of smoke coverage beyond compatibility fixes needed for `base-v0.8.6`

## Expected Files

### Create

- `docs/harness/tasks/2026-06-28-consume-pantheon-base-v0-8-6.task.md`
- `.harness/tasks/2026-06-28-consume-pantheon-base-v0-8-6/manifest.json`
- `.harness/evidence/2026-06-28-consume-pantheon-base-v0-8-6/commands.json`
- `.harness/evidence/2026-06-28-consume-pantheon-base-v0-8-6/review.md`
- `.harness/evidence/2026-06-28-consume-pantheon-base-v0-8-6/pr-body.md`

### Modify

- `backend/**`
- `frontend/**`
- `scripts/**`
- `docs/PROJECT_INHERITANCE.md`
- `README.md`

### Do Not Touch

- unrelated historical branches or workflow-run retention settings
- `../pantheon-base/**`

## Implementation Notes

- The backend sync is intentionally converged to the current base layout instead of keeping dual old/new directory structures alive.
- Auth/session responses now rely on secure cookies for tokens, so smoke helpers must not assume login responses expose raw tokens in JSON.
- This task includes governance artifact backfill because the code sync was already complete but merge was blocked by PR governance prerequisites.

## Method Readiness

- Consumer-Specific Controls:
  - pantheon-ops inheritance sync
- Required Sensors:
  - command
- Required Evidence:
  - commands.json
  - review.md
- Ratchet Decision: base-updated
- Deferred Code Issues:
  - broader smoke-suite expansion remains future work

## Delivery Governance

- Design Gate:
  - inheritance boundary confirmation
- Development Gate:
  - shared structure convergence
  - governance artifact linkage
- QA Acceptance Gate:
  - backend tests
  - frontend build
  - smoke-helper regression checks
- GitHub Governance Gate:
  - repo-quality-gate

## Structural Scope

- Affected Subgraph:
  - `backend/shared auth/common/lowcode -> frontend smoke helpers -> release sync scripts`
- Boundary Crossings:
  - `pantheon-base -> pantheon-ops`
- Risk Nodes:
  - `login session contract`
  - `generated feature-ledger snapshot`
  - `legacy shared-path duplicates`
- Graph Focus:
  - `inheritance sync compatibility`

## Verification Plan

- `go test ./...`
- `npm --prefix frontend run build`
- `npm --prefix frontend run test:smoke:scripts`
- `npm run check:docs-frontmatter`
- `npm run check:task-packet-template`
- `npm run check:pr-governance`

## Linkage

- Task ID: 2026-06-28-consume-pantheon-base-v0-8-6
- Task Manifest: `.harness/tasks/2026-06-28-consume-pantheon-base-v0-8-6/manifest.json`
- OpenSpec Change: none
- Plan References: none
- Evidence Directory: `.harness/evidence/2026-06-28-consume-pantheon-base-v0-8-6/`
- Review File: `.harness/evidence/2026-06-28-consume-pantheon-base-v0-8-6/review.md`

## Evidence Required

- backend test result summary
- frontend build result summary
- smoke helper compatibility check
- PR governance body linkage

## Human Gates

- base version

## Completion Checklist

- [ ] base version and landing side declared
- [ ] legacy conflicting structures removed
- [ ] verification run or exception recorded
- [ ] governance artifacts saved
- [ ] PR body updated to real artifact paths
