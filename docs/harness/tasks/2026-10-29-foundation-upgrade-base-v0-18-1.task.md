---
title: Consume pantheon-base-v0.18.1 into pantheon-ops
doc_type: Remediation
layer: inheritance-sync
status: Active
linked_contracts:
  - docs/PROJECT_INHERITANCE.md
updated_at: 2026-10-29
---

# Task Packet: 2026-10-29-foundation-upgrade-base-v0-18-1

Target repository: `pantheon-ops`
Layer: `inheritance-sync (pantheon-base -> pantheon-ops)`
Delivery tier: `L2`
Mode: immutable release consumption, baseline jump 0.17.7 -> 0.18.1, consumer drift repair

## Goal

Advance the Ops consumer from `pantheon-base-v0.17.7` to the published `pantheon-base-v0.18.1`
(`93505225e36c8d3ba5aa9facc12975eb813167d3`) so the tree carries the 13-wave simplification
programme, and repair the three consumer-side drifts the jump exposes: the `TS2614` break that
v0.18.0 shipped to this repository's `DeployTaskDetail` page, the stale `scripts/task-manifest.mjs`
overlay that crashes `check:task-packet` and `check:harness-docs`, and the missing
`check:ui-quality-gate` step that base 0.18 now requires of the Docs Governance job.

## Primary Layer

inheritance-sync

## Workspace Context

- Target Repository: `pantheon-ops`
- Repository Role: `business-consumer`
- Upstream Dependencies: `pantheon-base`
- Downstream Consumers: `none`
- Sync Expectation: `completed`
- Release Requirement: `consumer-lock-update`

## Dependency Layers

- `pantheon-base` foundation release pipeline (tag, four assets, Release Gate, Foundation Release Verify)
- Ops business overlay assembly (`scripts/business-overlay/rebuild-from-base.mjs`)
- Ops shared-drift gate (`scripts/business-overlay/check-shared-drift.mjs`)
- Base-owned harness scripts that the consumer runs directly (`scripts/harness/*`)
- Ops Docs Governance job (`.github/workflows/quality.yml`)

## Harness Profile

- Template: api-service
- Overlay: pantheon-ops
- Quality Profile: foundation-release-consumer
- Portable Failure Class: static-sensor-gap
- Owner Layer: inheritance-boundary
- Coverage Dimensions:
  - behaviour
  - architecture-fitness
  - runtime-quality
  - method-health

## Contract Anchors

- `AGENTS.md`
- `docs/PROJECT_INHERITANCE.md`
- `foundation-release.lock.json`
- `.foundation/foundation-release.lock.json`
- `.foundation/allowed-drift.json`
- `business-overlay.json`
- `scripts/business-overlay/rebuild-from-base.mjs`
- `scripts/task-manifest.mjs`

## Scope

### In

- Update `foundation-release.lock.json` to `pantheon-base-v0.18.1`: release line, version, base
  commit, artifact name, materialization path, bundle checksum and `repo.tar` checksum.
- Rewrite `.foundation/foundation-release.lock.json` with the release identity and the gate facts
  read back from GitHub.
- Rebuild the Ops tree from the locked snapshot, validate it, and apply it to the live tree.
- Drop `scripts/task-manifest.mjs` and `scripts/cleanup-github-branches.mjs` from
  `repositoryOverlayPaths` and adopt base's copies, so the two harness scripts that import
  `resolveArchiveAwarePath` run again and branch deletion tolerates already-gone heads.
- Add the `check:ui-quality-gate` step to the Docs Governance job, exempt Dependabot from the PR
  body ceremony the way base does, and normalise the eleven ops manifests that still spell `status`
  as `complete`.
- Re-tidy the merged Go module graph and re-run every Docs Governance gate plus backend and
  frontend verification on the applied tree.

### Out

- Any change to the `pantheon-base` repository, to an existing tag, release or asset.
- Hand-copying base files into the Ops source tree instead of materializing the locked release.
- Rewriting `CHANGELOG.md` or `README*.md` from base; those are Ops overlay paths by contract.
- Any behaviour change in the business modules.
- Merging or closing the open Dependabot pull requests inside this packet (tracked separately).

## Expected Files

### Create

- `docs/harness/tasks/2026-10-29-foundation-upgrade-base-v0-18-1.task.md`
- `.harness/tasks/2026-10-29-foundation-upgrade-base-v0-18-1/manifest.json`
- `.harness/evidence/2026-10-29-foundation-upgrade-base-v0-18-1/commands.json`
- `.harness/evidence/2026-10-29-foundation-upgrade-base-v0-18-1/summary.md`
- `.harness/evidence/2026-10-29-foundation-upgrade-base-v0-18-1/review.md`

### Modify

- `foundation-release.lock.json`
- `.foundation/foundation-release.lock.json`
- `VERSION`
- `business-overlay.json`
- `scripts/task-manifest.mjs`
- `scripts/cleanup-github-branches.mjs`
- `.github/workflows/quality.yml`
- `.github/workflows/pr-automation.yml`
- `tests/scripts/workflow-baseline.test.mjs`
- `.business-overlay-report.json`
- `docs/PROJECT_INHERITANCE.md`
- 11 `.harness/tasks/*/manifest.json` status vocabulary

### Do Not Touch

- `.foundation/allowed-drift.json`
- `.foundation/releases/**` for any previously materialized release
- any existing tag, release or release asset
- `CHANGELOG.md`, `README.md`, `README.en.md` (Ops overlay copies)

## Implementation Notes

- `rebuild-from-base.mjs` resolves the base from the lock when `--base` is omitted, so this round
  consumed the published v0.18.1 archive and never the local Base working tree.
- The consumer had a concrete reason to jump: v0.18.0's simplification wave S1 removed
  `isForbiddenRequestError` from `frontend/src/api/request.ts` after a repo-local zero-caller
  search, while `frontend/src/modules/business/deploy/task/DeployTaskDetail.tsx` still imports it.
  The ops frontend build failed with `TS2614` against v0.18.0; v0.18.1 restores the export.
- The staging tree was validated before it touched the working copy: drift, overlay contract,
  13/13 overlay tests, backend build/vet/test and frontend lint/type-check/build all ran inside
  `.tmp/business-overlay-rebuild`. The live-tree replacement then applied the same bytes and the
  same gate set re-ran on the applied tree.
- The replacement is the documented L2 human gate; it ran with the maintainer's explicit
  instruction to consume the latest base release, and the pre-swap working tree was snapshotted to
  `.tmp/backup-pre-0181/` (`tracked.patch` plus `untracked.tar.gz`) before any file moved.
- `scripts/task-manifest.mjs` was pinned as an Ops overlay path from the pre-archive era. Base 0.18
  turned it archive-aware and `scripts/harness/check-task-packet.mjs` began importing
  `resolveArchiveAwarePath` from it, so the pinned copy made both that checker and
  `check-doc-links.mjs` die with a module SyntaxError. The file carried no Ops-specific
  customisation, so the pin was removed rather than patched.
- `scripts/cleanup-github-branches.mjs` had the same defect in a quieter form: the pinned copy threw
  on `404` and `403` where base's exported `deleteBranch` returns `already-gone` or
  `protected-or-rejected`. Adopting base's copy also makes the branch cleanup that follows this
  merge tolerate heads that are already deleted.
- `pr-automation.yml` ran the PR body ceremony for every PR while `check-pr-governance.mjs` has no
  Dependabot branch of its own, so every Dependabot PR failed `PR Governance Prereq` and could
  never reach auto-merge. The workflow now mirrors base's exemption and output expression; ops has
  no `RELEASE_GATE_TOKEN` secret, so the merge step keeps `github.token` and one inherited test
  assertion stays red.
- Ops Docs Governance never ran `check:task-packet`, which is why the crashed checker stayed
  invisible to CI. The checker is now fixed and green for current-format packets, but re-enabling it
  as a CI gate needs a 60-entry allowlist for pre-template Ops packets, so that wiring is a separate
  task rather than a side effect of this consumption.
- `check:ui-quality-gate` reports one finding when the workflow never runs it, which is a workflow
  contract rather than a task finding; wiring the step is the intended repair.
- The deleted set is base content retired by the simplification waves, not Ops business code: the
  logging middleware, four tenant iam files, `pkg/common/token.go`, `TenantSwitcher.tsx` and 13
  frontend capture scripts.

## Verification Plan

- `sha256sum` both materialized assets against the lock and the published `.sha256` sidecars
- `node scripts/business-overlay/rebuild-from-base.mjs` (lock-resolved)
- `node scripts/business-overlay/check-business-overlay.mjs`
- `node scripts/business-overlay/check-shared-drift.mjs --base ../pantheon-base`
- `node --test tests/scripts/business-overlay/*.test.mjs`
- `go mod tidy && go build ./... && go vet ./... && gofmt -l .` and `go test -count=1 ./...` in `backend`
- `npm ci --ignore-scripts` at the root and in `frontend`, then `lint`, `type-check`, `build`
- Docs Governance set: `check:docs-frontmatter`, `check:encoding`, `check:structure`,
  `check:task-packet-template`, `check:pr-governance`, `check:business-overlay`,
  `test:business-overlay`, `check:generated-modules`, `check:ui-quality-gate`
- `check:task-packet`, `check:harness-adoption`, `check:failure-registry`,
  `check:business-boundaries`, `check:business-foundation`, `check:duplication`
- hosted `Quality Gates`, `Security Gates`, `SonarCloud Scan`, `Inheritance Drift Detection`,
  `Branch Hygiene` and `PR Automation` on the consumption PR

## Linkage

- Task ID: `2026-10-29-foundation-upgrade-base-v0-18-1`
- Task Manifest: `.harness/tasks/2026-10-29-foundation-upgrade-base-v0-18-1/manifest.json`
- OpenSpec Change: `none`
- Superpowers Plan: `none`
- Plan References: `docs/harness/tasks/2026-10-29-foundation-upgrade-base-v0-18-1.task.md`
- Evidence Directory: `.harness/evidence/2026-10-29-foundation-upgrade-base-v0-18-1/`
- Review File: `.harness/evidence/2026-10-29-foundation-upgrade-base-v0-18-1/review.md`

## Evidence Required

- the published release identity: tag object, target commit, publishedAt, asset count, and the two
  digests re-hashed locally after materialization
- `sharedPaths` equivalence between the Ops lock and the v0.18.1 release manifest read out of the tgz
- overlay contract, shared-drift and business-overlay test results on both the staging and the
  applied tree
- backend build/vet/gofmt/test and frontend lint/type-check/build results on the applied tree
- the three consumer-drift repairs with the failing command before and the passing command after
- review summary

## Human Gates

- Stop if either SHA-256 digest does not match the lock, or if `repoSnapshot.baseCommit` differs from
  `lock.baseCommit`.
- Stop if a business overlay anchor stops resolving exactly once; that is a base assembly contract
  change and must be fixed in base first.
- The live-tree replacement is the L2 gate. It ran only because the maintainer explicitly asked for
  the consumer to move to the latest base release; the pre-swap diff was saved first.
- Deciding the fate of the open Dependabot pull requests and non-main branches stays with the
  maintainer.

## Completion Checklist

- [x] Layer and boundary declared
- [x] Contract anchors read
- [x] Verification run or exception recorded
- [x] Evidence saved or summarized
- [x] Review completed
