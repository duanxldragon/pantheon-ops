# pantheon-base v0.13.0

**Status**: Published on 2026-09-27 as immutable tag `pantheon-base-v0.13.0`.

## Highlights

- Archived 116 completed Harness tasks with their evidence and normalized the active-task inventory.
- Added durable governance entry points: `.harness/STATUS.md`, `.harness/ARCHIVE.md`, and the v0.13.0 preparation record.
- Closed the September naming, layer-boundary, generated-artifact, session-revocation, tenant-settings, upload-authorization, pagination, maintenance, and production-Redis remediation rounds.
- Preserved the foundation-release consumer contract and recorded the remaining consumer follow-up without treating it as a Base release blocker.

## Compatibility

No intentional API or database breaking changes are included. Consumers should use the published immutable tag and must not lock to a mutable branch.

## Verification boundary

The repository-local audit passed backend tests/vet, frontend type-check/lint/build, npm audit, and frontend contract checks. Hosted CI, security, smoke, SonarCloud, Release Gate, final release assets, and GitHub Release publication all passed. The `pantheon-ops` lock update remains a consumer follow-up.
