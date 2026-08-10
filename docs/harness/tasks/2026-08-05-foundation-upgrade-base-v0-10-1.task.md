Target repository: pantheon-ops
Layer: inheritance-sync (pantheon-base -> pantheon-ops)
Delivery tier: L2
Mode: implementation, release consumer, verification

Read first:
- pantheon-ops/AGENTS.md
- pantheon-ops/docs/PROJECT_INHERITANCE.md
- pantheon-base/DESIGN.md
- pantheon-base/AGENTS.md
- pantheon-base/docs/README.md

Source release:
- releaseVersion: pantheon-base-v0.10.1
- releaseLine: release/0.10
- baseCommit: 929fcf5fdb0d8f663d7b66b68d6668e193daf8b9
- GitHub release archive SHA-256: 2123073d7c01d44c48537d1a0f1fb8655d095ca00ac3205586a436c33f62487e

In scope:
- Install and verify the formal Base release archive, then use its manifest and bundle as the only shared-path source.
- Strengthen the Ops release consumer to require rollback protection, enforce release compatibility, and preserve all backend/frontend business overlays.
- Update the foundation lock, inheritance documents, and local verified release artifact through the consumer.
- Verify inherited backend/frontend alignment and the smallest relevant backend, frontend, menu, i18n, and smoke gates.
- Relocate the obsolete shared PageHeader into the business extension area without changing its markup, classes, or detail-page behavior.

Out of scope:
- Editing or copying Pantheon Base platform/system source into Ops.
- Changing business/cmdb, business/deploy, or business/bizscope behavior except to preserve it during the shared-path update.
- Retagging or changing the immutable pantheon-base-v0.10.1 release.

Landing-side decision:
- Base owns the foundation source, release artifact, platform, and system domains. Those were released before this task.
- Ops owns the consumer, release lock, inheritance documentation, and business overlays. The consumer safeguards belong in Ops because they govern installation of an immutable Base artifact into this repository.

Acceptance criteria:
- foundation-release.lock.json points exactly to pantheon-base-v0.10.1 / release/0.10 / 929fcf5fdb0d8f663d7b66b68d6668e193daf8b9.
- The installed archive digest agrees with the GitHub Release asset digest.
- Consumer tests prove no business/** path is overwritten, writes require --rollback-on-error, incompatible transitions fail by default, and rollback restores changed files.
- check:inheritance and locked-release base-sync checks pass after apply.
- Backend and frontend validation results, hosted checks, and any explicit runtime gap are recorded in evidence.

Execution guardrails:
- Start the apply only from a clean Ops worktree.
- Run the manifest dry-run before apply.
- Apply with --apply-shared-backend --apply-shared-frontend --update-inheritance-docs --rollback-on-error --check.
- The release starts at release/0.8 while the v0.10.1 manifest requires release/0.10. A forward line jump must therefore be explicit with --allow-release-line-jump and recorded in evidence.
- Stop if the formal release identity, business-overlay preservation, or mandatory inherited checks do not pass.
