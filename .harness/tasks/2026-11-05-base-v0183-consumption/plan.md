# Latest official GitHub Base consumption (target v0.18.4)

Task ID retains the original v0183 wording for continuity. Latest user steering requires v0.18.4 security publication before final Ops consumption.

1. Preserve dbce4ae business fixes and Base000031/Ops000032 migration ownership.
2. Complete Base minimatch52 patch, new exact-candidate checks/review and official v0.18.4 release.
3. Run `npm run fetch:base-release`: GitHub latest stable metadata, four uploaded assets, SHA256, manifest/tag/git-archive provenance, then atomic lock update.
4. Run locked rebuild and drift validation; local Base CLI sources and enclosing Git HEAD are forbidden.
5. Validate complete staged CGO race, frontend, business smoke and hosted gates; independently review exact source before commit/push/merge.
6. Apply verified staging only after validation; close parent and child evidence with actual immutable release/PR/check references.

Current local consumer and migration tests:54 pass/0 fail/0 skip. Network EACCES blocks dependency installation, GitHub acquisition and independent review. Do not consume the earlier v0.18.3 as final, update the live lock prematurely, or treat historical review fingerprints as current approval.

## Historical preparation

# Official Base v0.18.3 consumption

Parent: pantheon-base/docs/harness/tasks/2026-11-05-base-freeze-release.task.md.

1. Preserve dbce4ae business fixes and declare new business-only migration/test assets.
2. Renumber the unpublished Ops backfill to 000032; Base owns 000031.
3. Wait for the approved official v0.18.3 release; verify asset hashes and update lock.
4. Rebuild a staging tree using the official overlay entrypoint, validate race/frontend/business smoke and source ownership.
5. Apply the verified tree, independently review, commit/push/CI/merge.

No production migration is executed. Approval of release consumption is already recorded by the parent task. Completion requires actual official release and consumer verification.

## Business migration assembly

Canonical host backfill SQL now lives in backend/modules/business/cmdb/host/migrations, and its regression lives in the host package. businessMigrations lists the paired numbered SQL files; the Ops rebuild script derives fixed backend/pkg/database/migrations destinations and adds them to the generated report. Generic product paths remain forbidden in businessPaths/repositoryOverlayPaths. Missing pairs, duplicate versions/directions, unsafe sources and foundation version collisions block reconstruction.

The initial declaration under backend/pkg/database failed the real rebuild preflight despite the previous shallow check passing. That red evidence is preserved at workspace audit-artifacts/base-freeze-2026-11-05/ops-overlay-invalid-business-path.log. The current checker reuses manifest validation. Fixture reconstruction verifies Base000031 unchanged, exact copies of Ops000032, deterministic output and collision rejection. Current 14 overlay tests and isolated MySQL backfill race pass; the real official-v0183 rebuild is still pending publication.

## Shared drift repair

Architecture round1 found derived migration targets absent from drift ownership validation. The checker now derives exact migration destinations from the same planner and validates regular files, canonical bytes and Base version collisions before recognizing ownership. New tests first failed with DRIFT_MISSING_IN_BASE, then all 37 passed. Real MySQL race covers missing table/column and genuine down/reapply with a newly inserted zero-version row plus CAS rejection. Generic paths and arbitrary generatedFiles remain protected. Source hashes are refreshed; independent dual-lane re-review remains pending.
