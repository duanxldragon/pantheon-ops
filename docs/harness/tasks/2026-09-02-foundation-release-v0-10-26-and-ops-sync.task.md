# Task Packet: 2026-09-02-foundation-release-v0-10-26-and-ops-sync

## Goal

Merge the remaining Pantheon Base PRs and branches into `main`, publish `pantheon-base-v0.10.26`, and update Pantheon Ops through the foundation-release consumer pipeline.

## Primary Layer

inheritance-sync

## Workspace Context

- Target Repository: `pantheon-base`
- Repository Role: `foundation-source`
- Upstream Dependencies: `pantheon-harness`
- Downstream Consumers: `pantheon-ops`
- Sync Expectation: `required`
- Release Requirement: `foundation-release`

## Dependency Layers

- repository branch and PR closure
- foundation release tooling
- Pantheon Ops consumer pipeline

## Harness Profile

- Template: api-service
- Overlay: pantheon-base
- Quality Profile: ci-workflow
- Portable Failure Class: method-health-gap
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
- `docs/contracts/PLATFORM_CONTRACT.md`

## Scope

### In

- Close actionable feedback on PR `#279`, merge it, and delete its remote source branch.
- Merge the existing `chore/sonar-duplication-reduction` branch after independent review and required checks.
- Correct `CLAUDE.md` so it defers execution-role boundaries to `AGENTS.md`.
- Remove unreferenced one-off backend import-rewrite scripts and recognize the existing frontend library export entrypoint in the repository-layout contract.
- Keep only `main` locally and on GitHub after verified merges.
- Cut immutable `pantheon-base-v0.10.26` from the checked `main` commit and publish its required assets.
- Rebuild and validate a clean Pantheon Ops consumer worktree from the release, then update its foundation lock through the existing pipeline.

### Out

- Changes to Base API, schema, permissions, menus, i18n semantics, or business behavior.
- Hand-copying Base files into Pantheon Ops.
- Overwriting the dirty primary `pantheon-ops` worktree or its unpushed commits.
- Mutating any existing tag or GitHub Release.

## Expected Files

### Create

- `docs/harness/tasks/2026-09-02-foundation-release-v0-10-26-and-ops-sync.task.md`
- `.harness/tasks/2026-09-02-foundation-release-v0-10-26-and-ops-sync/manifest.json`
- `.harness/evidence/2026-09-02-foundation-release-v0-10-26-and-ops-sync/`

### Modify

- `CLAUDE.md`
- `docs/designs/REPOSITORY_LAYOUT.md`
- `docs/designs/REPOSITORY_LAYOUT.en.md`
- `scripts/harness/check-structure-contract.mjs`

### Do Not Touch

- runtime backend modules, packages, and contracts; deletion is limited to the unreferenced one-off import-rewrite scripts at `backend/` root.
- `frontend/src/`
- Primary `pantheon-ops` worktree

## Implementation Notes

- Deliver the fix by merging into `main` and publishing an immutable release; never copy Base files into Ops by hand.
- Use an isolated consumer worktree for the Ops rebuild so the dirty primary worktree stays untouched.

## Verification Plan

- `npm run check:docs-frontmatter`
- `npm run check:task-packet-template`
- GitHub required checks and `Release Gate Summary` on the final Base commit
- `npm run release:foundation:cut -- --release-version pantheon-base-v0.10.26 --release-line release/0.10 --base-commit <final-main-sha>`
- `npm run upgrade:foundation:local-plan -- --release-version pantheon-base-v0.10.26`
- Pantheon Ops overlay, Go race, frontend, and business-smoke gates in an isolated worktree

## Linkage

- Task ID: `2026-09-02-foundation-release-v0-10-26-and-ops-sync`
- Task Manifest: `.harness/tasks/2026-09-02-foundation-release-v0-10-26-and-ops-sync/manifest.json`
- OpenSpec Change: `none`
- Superpowers Plan: `none`
- Plan References: `docs/harness/tasks/2026-09-02-foundation-release-v0-10-26-and-ops-sync.task.md`
- Evidence Directory: `.harness/evidence/2026-09-02-foundation-release-v0-10-26-and-ops-sync/`
- Review File: `.harness/evidence/2026-09-02-foundation-release-v0-10-26-and-ops-sync/review.md`

## Evidence Required

- PR closure and branch-deletion record
- exact-commit Release Gate and required check results
- published immutable release with verified assets
- Ops consumer rebuild and lock update result
- review summary

## Human Gates

- Stop before an immutable tag/release if the matching `Release Gate Summary` is absent or failed.
- Stop before modifying the dirty primary Ops worktree; use an isolated consumer worktree instead.

## Completion Checklist

- [x] Layer and boundary declared
- [x] Contract anchors read
- [x] Verification run or exception recorded
- [x] Evidence saved or summarized
- [x] Review completed
