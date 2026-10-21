# Upgrade Notes

1. Pin `pantheon-ops/foundation-release.lock.json` to the published `pantheon-base-v0.13.0` tag; do not consume `main` directly.
3. Rebuild the consumer snapshot and business overlay in an isolated, clean Ops worktree.
4. Run inheritance, boundary, backend, frontend, and business smoke gates before merging the consumer upgrade.

No database migration or configuration rewrite is required by the governance changes in this release.
