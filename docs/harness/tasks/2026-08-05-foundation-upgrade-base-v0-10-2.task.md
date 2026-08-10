Target repository: pantheon-ops
Layer: inheritance-sync (pantheon-base -> pantheon-ops)
Delivery tier: L2
Mode: formal release consumption, verification, and repository closeout

Source release:
- releaseVersion: pantheon-base-v0.10.2
- releaseLine: release/0.10
- baseCommit: e38ce7395c46cecd71f644952e95268404c8a10c
- GitHub release archive SHA-256: b300c17f7e605e113ef4d2ceb8de1ac59bc29a2fad1cbcc4d9f1a6fdf04af09c

In scope:
- Install and verify the formal Base release archive from GitHub.
- Run the release consumer dry-run and rollback-protected apply.
- Update the foundation lock and inheritance documents from the verified manifest.
- Prove the Base-owned CSS matcher is inherited through the exact tooling allowlist.
- Preserve every business overlay and validate inherited backend/frontend contracts.
- Record local, hosted, runtime, and visual evidence before merging the Ops PR.

Out of scope:
- Editing or retagging the immutable Base release.
- Manual copies from the Base worktree.
- New platform or system behavior owned by Ops.
- New runtime UI design or styling changes.

Landing-side decision:
- Base owns the shared checker and immutable release artifact.
- Ops owns the lock, verified installer, consumer adapter, inheritance evidence, and business overlays.

Acceptance criteria:
- The lock, installed verification marker, manifest, tag, and GitHub asset all identify the exact v0.10.2 release.
- Dry-run succeeds before any shared-path write.
- Apply uses rollback protection and preserves business overlays.
- Foundation, inheritance, base-sync, Go race, frontend, and business smoke gates pass.
- The final Ops PR head and final main workflows pass before all non-main branches are removed.

Execution guardrails:
- Install only after committing the lock and returning to a clean worktree.
- Apply only from a clean worktree.
- Use the formal GitHub release asset, not the sibling Base worktree.
- Stop on any release identity, checksum, rollback, overlay, or inheritance failure.

Evidence:
- Task manifest: `.harness/tasks/2026-08-05-foundation-upgrade-base-v0-10-2/manifest.json`
- Evidence directory: `.harness/evidence/2026-08-05-foundation-upgrade-base-v0-10-2/`
- Review: `.harness/evidence/2026-08-05-foundation-upgrade-base-v0-10-2/review.md`
