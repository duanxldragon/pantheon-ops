# Review Artifact

## Scope

- Layer: `ci-workflow`
- Runtime-sensitive: yes, because release installation and consumption behavior changed.
- UI impact: none; visual evidence is not applicable.

## Review Focus

- Routine sync cannot read a release without a valid marker matching the lock checksum.
- Manifest or release-tree mutation after verification is rejected before synchronization.
- Explicit upgrades bind the marker to the target archive instead of the previous lock checksum.
- Windows/MSYS archive extraction uses relative working paths.
- Workflow runtimes and action pins stay aligned with the certified Base baseline.

## Residual Gates

- Hosted GitHub checks must pass on the Ops pull request.
- Ops must consume and lock the next certified Base release after publication.
