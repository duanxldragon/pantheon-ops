# Task Packet: 2026-07-30-local-consolidation

## Goal

Consolidate all validated local Pantheon Base work into `main`, repair the
remaining PR #220 quality failures, and remove superseded branches and
worktrees after GitHub verification.

## Primary Layer

platform

## Dependency Layers

- system/auth
- system/iam
- system/org
- system/i18n

## Harness Profile

- Template: admin-platform
- Overlay: pantheon-base
- Quality Profile: auth-security
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
- `docs/harness/PANTHEON_BASE_DELIVERY_WORKFLOW.md`
- `docs/harness/AI_QUALITY_GOVERNANCE.md`
- `docs/acceptances/AGENT_EXECUTION_CHECKLIST.md`

## Scope

### In

- Preserve the newer secure scaffold implementation while resolving recovery-worktree conflicts.
- Repair the identified PR #220 backend test, import-validation, menu-seed, lint, and token-key failures.
- Create the governance linkage for this consolidation and update the PR body.
- Merge the green PR, then remove redundant worktrees, local/remote branches, stashes, and temporary diagnostics.

### Out

- Public API, database schema, permission-policy, menu-contract, and UI behavior changes.
- Foundation release or Pantheon Ops synchronization.
- Replacing repository history or deleting unverified recovery work.

## Structural Scope

- Affected Subgraph: `token middleware -> blacklist key`; `seed cleanup -> GORM Pluck`; `post import -> i18n error key`; `user profile test -> query failure propagation`.
- Boundary Crossings: `platform/internal -> system/auth`, `system/iam`, `system/org`, and `system/i18n`.
- Risk Nodes: token blacklist invalidation, seed deletion selection, import error localization, test database callback lifecycle.
- Graph Focus: sensitive-input-flow and test/runtime failure propagation.

## Expected Files

### Create

- `.harness/tasks/2026-07-30-local-consolidation/manifest.json`
- `.harness/evidence/2026-07-30-local-consolidation/commands.json`
- `.harness/evidence/2026-07-30-local-consolidation/summary.md`
- `.harness/evidence/2026-07-30-local-consolidation/review.md`
- `docs/harness/tasks/2026-07-30-local-consolidation.task.md`

### Modify

- `backend/internal/middleware/token_middleware.go`
- `backend/modules/system/i18n/i18n_export.go`
- `backend/modules/system/i18n/i18n_helpers_test.go`
- `backend/modules/system/iam/user/user_service_test.go`
- `backend/modules/system/org/post/post_import_regression_test.go`
- `backend/modules/system/org/post/post_service.go`
- `backend/modules/system/seed.go`

### Do Not Touch

- Public HTTP/API DTO contracts
- Database migrations and permission policy
- Pantheon Ops source or foundation-release artifacts

## Implementation Notes

- Minimal Complexity Rung: reuse existing helpers and tests; make only local correctness fixes.
- Resolve recovery conflicts in favor of the current `main` implementation because it enforces restrictive generated-file permissions and guarded reads.
- Reuse `authtoken.BlacklistUserKey` and `common.ErrMessage`; add no dependencies or abstractions.

## Method Readiness

- Consumer-Specific Controls: PR governance, Go race suite, golangci-lint new-code gate, frontend contract, and GitHub smoke sanity.
- Required Sensors: Go tests, formatting/diff check, repository governance checks, hosted race/smoke gates, independent review.
- Required Evidence: command summary, runtime gap, review summary, hosted check result.
- Ratchet Decision: sensor-added
- Deferred Code Issues: Windows local environment has no supported MinGW C compiler, so `go test -race ./...` is verified in GitHub Actions.

## Delivery Governance

- Design Gate: short boundary note; behavior-preserving CI closure only.
- Development Gate: expected-files and do-not-touch boundaries declared.
- QA Acceptance Gate: full non-race Go test locally; Linux race, lint, frontend contract, and smoke in GitHub Actions.
- GitHub Governance Gate: repo-quality-gate

## Execution Roles

- Implementer Posture: reviewer-assisted
- Reviewer Posture: architecture, security, mechanical

## Stop Points

- Stop before deleting a branch, worktree, or stash until its commits and worktree status have been verified.
- Stop if GitHub reports a remaining required-check, review, or merge-protection failure after the current commit.

## State Plan

- Checkpoint Expectation: PR #220 head update and GitHub run URL.
- Resume Artifacts: this task packet, linked evidence, PR #220, and `git worktree list`.

## Verification Plan

### Backend

- `go test ./...`
- `go test -race ./...` in GitHub Actions

### Frontend

- `npm run lint`
- `npm run build`
- `npm run test:unit`

### Browser / Smoke

- GitHub `Smoke Sanity` on the updated PR head.

### Runtime Evidence

- GitHub Linux race suite and MySQL/Redis-backed smoke run; local Windows race is blocked by the unavailable native cgo compiler.

## Linkage

- Task ID: `2026-07-30-local-consolidation`
- Task Manifest: `.harness/tasks/2026-07-30-local-consolidation/manifest.json`
- OpenSpec Change: none
- Superpowers Plan: none
- Plan References: `docs/harness/tasks/2026-07-30-local-consolidation.task.md`
- Evidence Directory: `.harness/evidence/2026-07-30-local-consolidation/`
- Review File: `.harness/evidence/2026-07-30-local-consolidation/review.md`

## Evidence Required

- command result summary
- explicit local race-test gap
- GitHub check result after the PR head update
- independent review summary

## Human Gates

- Required GitHub checks and repository merge protection.

## Sync expectation

- Only `pantheon-base` is modified.
- No `base -> ops` sync is required because no shared external contract or behavior changes.

## Completion Checklist

- [x] Layer and boundary declared
- [x] Quality profile declared
- [x] Ratchet decision declared for the CI failure class
- [x] Delivery governance gates declared
- [x] Contract anchors read
- [x] Verification run or exception recorded
- [x] Evidence saved or summarized
- [x] Review completed
