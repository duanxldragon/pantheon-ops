---
title: Consume pantheon-base-v0.16.1 into pantheon-ops
doc_type: Remediation
layer: inheritance-sync
status: Active
linked_contracts:
  - docs/PROJECT_INHERITANCE.md
updated_at: 2026-10-12
---

# Task Packet: 2026-10-12-foundation-upgrade-base-v0-16-1

Target repository: `pantheon-ops`
Layer: `inheritance-sync (pantheon-base -> pantheon-ops)`
Delivery tier: `L2`
Mode: immutable release consumption and consumer lock update

## Goal

Advance the Ops foundation consumer lock from `pantheon-base-v0.13.0` to the newly published
`pantheon-base-v0.16.1`, materialize that release through the locked-artifact path, and prove the
business overlay still assembles on the new base. `v0.14.0`, `v0.15.0` and `v0.16.0` were each
published with zero assets, so the lock could not advance for three releases; `v0.16.1` is the first
release since `v0.13.0` that carries the four consumable assets the consumer requires.

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
- `.foundation/allowed-drift.json`
- `scripts/business-overlay/rebuild-from-base.mjs`
- `scripts/business-overlay/check-shared-drift.mjs`

## Scope

### In

- Update `foundation-release.lock.json` to `pantheon-base-v0.16.1`: release line, version, base
  commit, artifact name, local materialization path, bundle checksum and `repo.tar` checksum.
- Materialize the release under `.foundation/releases/pantheon-base-v0.16.1/` and verify both
  SHA-256 digests against the lock.
- Re-verify that `sharedPaths` is byte-identical between the lock, the new manifest and the
  previously locked `v0.13.0` manifest, so no path declaration changes with the version.
- Re-verify that every business overlay assembly anchor still resolves exactly once against the new
  base tree.
- Record harness evidence for the lock update and the consumer pre-flight.

### Out

- Any change to the `pantheon-base` repository or to an existing immutable release.
- Hand-copying base files into the Ops source tree; the consumer materializes from the release only.
- Replacing, deleting or rewriting `business/*` overlay files.
- Any change to backend or frontend runtime behaviour in Ops.

## Expected Files

### Create

- `docs/harness/tasks/2026-10-12-foundation-upgrade-base-v0-16-1.task.md`
- `.harness/tasks/2026-10-12-foundation-upgrade-base-v0-16-1/manifest.json`
- `.harness/evidence/2026-10-12-foundation-upgrade-base-v0-16-1/commands.json`
- `.harness/evidence/2026-10-12-foundation-upgrade-base-v0-16-1/summary.md`
- `.harness/evidence/2026-10-12-foundation-upgrade-base-v0-16-1/review.md`

### Modify

- `foundation-release.lock.json`

### Do Not Touch

- `business/**`
- `backend/modules/business/**`
- `frontend/src/modules/business/**`
- `.foundation/allowed-drift.json`
- any existing tag, release, or previously materialized `.foundation/releases/*` entry

## Implementation Notes

- The lock is the only tracked consumer surface. `.foundation/releases/**` is gitignored, so
  materialized artifacts never enter the repository; only the digests in the lock do.
- `rebuild-from-base.mjs` resolves the base from the lock when `--base` is not passed: it downloads
  the bundle and `repo.tar` from the locked GitHub release when they are not already cached, verifies
  `releaseArtifact.checksum` and `repoSnapshot.sha256`, extracts `repo.tar` into
  `<localPath>/repo/`, and asserts `repoSnapshot.baseCommit === lock.baseCommit`.
- Do not hand-edit `sharedPaths`. It is produced by the base manifest builder; a genuine path change
  would be a contract change and belongs in base first.

## Verification Plan

- `node scripts/business-overlay/check-shared-drift.mjs --base <materialized base root>`
- `node scripts/business-overlay/check-business-overlay.mjs --root .tmp/business-overlay-rebuild`
- `npm run check:business-overlay`
- `npm run test:business-overlay`
- `npm run check:docs-frontmatter`
- hosted `Base Compatibility Rebuild` (`.github/workflows/inheritance-drift-detection.yml`), which
  rebuilds Ops against the base default branch and type-checks the rebuilt frontend

## Linkage

- Task ID: `2026-10-12-foundation-upgrade-base-v0-16-1`
- Task Manifest: `.harness/tasks/2026-10-12-foundation-upgrade-base-v0-16-1/manifest.json`
- OpenSpec Change: `none`
- Superpowers Plan: `none`
- Plan References: `docs/harness/tasks/2026-10-12-foundation-upgrade-base-v0-16-1.task.md`
- Evidence Directory: `.harness/evidence/2026-10-12-foundation-upgrade-base-v0-16-1/`
- Review File: `.harness/evidence/2026-10-12-foundation-upgrade-base-v0-16-1/review.md`

## Evidence Required

- the published release identity: tag, base commit and the four asset names
- SHA-256 of `foundation-release-pantheon-base-v0.16.1.tgz` and of `repo.tar`, and the lock diff
- `sharedPaths` equivalence result across the lock, the new manifest and the `v0.13.0` manifest
- business overlay anchor resolution result against the new base tree
- hosted `Base Compatibility Rebuild` conclusion
- review summary

## Human Gates

- Stop if either SHA-256 digest does not match the lock, or if `repoSnapshot.baseCommit` differs
  from `lock.baseCommit`.
- Stop if any business overlay anchor stops resolving exactly once; that is a base assembly contract
  change and must be fixed in base first.
- Stop if the consumer would need to replace or delete a `business/*` overlay file.

## Completion Checklist

- [x] Layer and boundary declared
- [x] Contract anchors read
- [x] Verification run or exception recorded
- [x] Evidence saved or summarized
- [x] Review completed
