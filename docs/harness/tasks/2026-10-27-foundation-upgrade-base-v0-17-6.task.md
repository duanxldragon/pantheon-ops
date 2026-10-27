---
title: Consume pantheon-base-v0.17.6 into pantheon-ops
doc_type: Remediation
layer: inheritance-sync
status: Active
linked_contracts:
  - docs/PROJECT_INHERITANCE.md
updated_at: 2026-10-27
---

# Task Packet: 2026-10-27-foundation-upgrade-base-v0-17-6

Target repository: `pantheon-ops`
Layer: `inheritance-sync (pantheon-base -> pantheon-ops)`
Delivery tier: `L2`
Mode: immutable release consumption, baseline swap and consumer gate repair

## Goal

Move the Ops consumer from `pantheon-base-v0.16.1` to the published `pantheon-base-v0.17.6`
(`c9bc6c0f5d351016afb4fffdedf12e72cc73e61d`), swap the Ops baseline onto the rebuilt assembly, and
close the defects that the swap exposes on the consumer side. This is the first release on the
`release/0.17` line that Ops consumes, so it carries the whole `v0.17.0..v0.17.6` range into Ops,
including the schema-visible tenant-plan migration (`000030`) and the tenant-operations surface.

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

- `pantheon-base` foundation release pipeline (release producer, out of scope here)
- Ops business overlay assembly (`scripts/business-overlay/rebuild-from-base.mjs`)
- Ops shared-drift gate (`scripts/business-overlay/check-shared-drift.mjs`)
- Ops frontend contract checks (menu, password toggle, import-path case)
- Ops harness evidence and lock records

## Harness Profile

- Template: api-service
- Overlay: pantheon-ops
- Quality Profile: foundation-release-consumer
- Portable Failure Class: consumer-lock-drift
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
- `scripts/business-overlay/check-shared-drift.mjs`

## Scope

### In

- Update `foundation-release.lock.json` to `pantheon-base-v0.17.6`: release line, version, base
  commit, artifact name, materialization path, bundle checksum and `repo.tar` checksum.
- Rewrite `.foundation/foundation-release.lock.json` (the consumer ledger) with the same identity
  and the release-gate facts actually read back from GitHub.
- Rebuild the whole Ops tree from the locked release snapshot and verify the staged tree before the
  swap (overlay contract, shared drift against base, backend build/test, frontend type-check).
- Swap the Ops baseline onto the verified staged tree: 141 new base files, 5 case-only renames,
  2 dead base-path files removed, 225 shared files refreshed.
- Declare `docs/migrations/legacy-ops-business-v010x` as an Ops repository overlay so the archived
  pre-overlay business DDL survives a future rebuild.
- Repair the consumer defects the swap surfaces: 31 stale `list-page.css` specifiers, 6 password
  fields that must use the shared `PasswordInput`, and the `check:shared-drift` script that
  `Base Compatibility Rebuild` calls but the merged root `package.json` never exposed.

### Out

- Any change to the `pantheon-base` repository, to an existing tag, release or asset.
- Hand-copying base files into the Ops source tree; the consumer materializes from the release only.
- Any change to business module behaviour, routes, permissions, menus or migrations beyond the
  mechanical repairs above.
- Turning the four advisory Ops harness checkers into CI gates (maintainer decision, see findings).

## Expected Files

### Create

- `docs/harness/tasks/2026-10-27-foundation-upgrade-base-v0-17-6.task.md`
- `.harness/tasks/2026-10-27-foundation-upgrade-base-v0-17-6/manifest.json`
- `.harness/evidence/2026-10-27-foundation-upgrade-base-v0-17-6/commands.json`
- `.harness/evidence/2026-10-27-foundation-upgrade-base-v0-17-6/summary.md`
- `.harness/evidence/2026-10-27-foundation-upgrade-base-v0-17-6/review.md`

### Modify

- `foundation-release.lock.json`
- `.foundation/foundation-release.lock.json`
- `business-overlay.json`
- `package.json`
- `scripts/business-overlay/rebuild-from-base.mjs`
- the swapped base-owned tree, the 31 css import sites and the 4 password-field files

### Do Not Touch

- `.foundation/allowed-drift.json`
- `.foundation/releases/**` for any previously materialized release
- any existing tag, release or asset
- `docs/migrations/legacy-ops-business-v010x/**` contents (archive only)

## Implementation Notes

- `rebuild-from-base.mjs` resolves the base from the lock when `--base` is omitted: it uses the
  materialized `.foundation/releases/pantheon-base-v0.17.6/` archive, verifies `releaseArtifact
  .checksum` and `repoSnapshot.sha256`, extracts `repo.tar`, and asserts
  `repoSnapshot.baseCommit === lock.baseCommit`. Nothing in this round reads `../pantheon-base`
  as a base source; only the drift gate compares against it.
- The v0.17.6 materialization keeps the bundle inside the tgz, so it has no pre-extracted `bundle/`
  or top-level `manifest.json`. The ledger records that difference instead of claiming the v0.16.1
  layout matched.
- Base renamed five pattern modules to lower camel case (`AppModalActions.ts` → `appModalActions.ts`
  and friends). A Windows checkout cannot express the rename by copy alone, so the old-case entries
  were removed from the index and disk before the staged tree was applied.
- `backend/pkg/database/migrate.go` is a base-owned shared file; its `currentRuntimeSchemaVersion`
  moved from 20 to 24 with the swap, together with migrations up to `000030`. No Ops-side pin needed
  updating.
- The merged root `package.json` only carries an explicit list of Ops scripts. `check:shared-drift`
  was missing from it, so the scheduled `Base Compatibility Rebuild` failed at its first drift step
  for every run since the workflow was added; the script name is now injected by the merge.

## Verification Plan

- `node scripts/business-overlay/rebuild-from-base.mjs --target .tmp/overlay-v0176`
- `node scripts/business-overlay/check-business-overlay.mjs --root .tmp/overlay-v0176`
- `node scripts/business-overlay/check-shared-drift.mjs --root .tmp/overlay-v0176 --base ../pantheon-base`
- `npm run check:shared-drift -- --base ../pantheon-base` (worktree form, after the swap)
- `npm run test:business-overlay`
- `go mod tidy && go build ./... && go vet ./... && gofmt -l .`
- `go test ./...` in `backend`, plus the hosted job's tree form
  `go test ./modules/business/... ./modules/system/iam/menu/...`
- `npm run type-check`, `npm run lint`, `npm run build` and all 14 frontend contract checks
- `npm run check:docs-frontmatter`, `npm run check:encoding`, `npm run check:structure`,
  `npm run check:business-overlay`, `npm run check:generated-modules`, `npm run check:duplication`
- hosted `Base Compatibility Rebuild`, `Code Quality Gates` and `Security Gates` on the PR

## Linkage

- Task ID: `2026-10-27-foundation-upgrade-base-v0-17-6`
- Task Manifest: `.harness/tasks/2026-10-27-foundation-upgrade-base-v0-17-6/manifest.json`
- OpenSpec Change: `none`
- Superpowers Plan: `none`
- Plan References: `docs/harness/tasks/2026-10-27-foundation-upgrade-base-v0-17-6.task.md`
- Evidence Directory: `.harness/evidence/2026-10-27-foundation-upgrade-base-v0-17-6/`
- Review File: `.harness/evidence/2026-10-27-foundation-upgrade-base-v0-17-6/review.md`

## Evidence Required

- the published release identity: tag object, target commit and the two asset digests re-hashed
  locally after materialization
- `sharedPaths` equivalence between the Ops lock and the v0.17.6 release manifest
- overlay contract and shared-drift results on both the staged tree and the swapped worktree
- backend build/vet/test and frontend type-check/lint/build/contract results
- the named consumer repairs, each with its before/after gate result
- hosted `Base Compatibility Rebuild` conclusion for the `check:shared-drift` fix

## Human Gates

- Stop if either SHA-256 digest does not match the lock, or if `repoSnapshot.baseCommit` differs
  from `lock.baseCommit`.
- Stop if any business overlay anchor stops resolving exactly once; that is a base assembly contract
  change and must be fixed in base first.
- Stop before deleting any Ops-tracked file outside the base-owned surface; the two removals here
  are base-path files that base no longer ships and that nothing references.
- Deleting or closing remote branches and open Dependabot pull requests stays with the maintainer.

## Completion Checklist

- [x] Layer and boundary declared
- [x] Contract anchors read
- [x] Verification run or exception recorded
- [x] Evidence saved or summarized
- [x] Review completed
