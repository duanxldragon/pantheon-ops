# pantheon-base v0.13.0 release candidate

**Status**: Candidate metadata prepared locally; tag and GitHub Release remain pending required checks and maintainer publication.

## Highlights

- Archived 116 completed Harness tasks with their evidence and normalized the active-task inventory.
- Added durable governance entry points: `.harness/STATUS.md`, `.harness/ARCHIVE.md`, and the v0.13.0 preparation record.
- Closed the September naming, layer-boundary, generated-artifact, session-revocation, tenant-settings, upload-authorization, pagination, maintenance, and production-Redis remediation rounds.
- Preserved the foundation-release consumer contract and recorded the remaining runtime evidence gaps without treating them as release evidence.

## Compatibility

No intentional API or database breaking changes are included in this candidate. Consumer upgrades must use the published immutable tag after the release gate passes; consumers must not lock to this candidate commit before publication.

## Verification boundary

The repository-local audit passed backend tests/vet, frontend type-check/lint/build, npm audit, and frontend contract checks. Hosted required checks, final release assets, and the `pantheon-ops` lock update are release-gate work and are not claimed by this candidate metadata.
