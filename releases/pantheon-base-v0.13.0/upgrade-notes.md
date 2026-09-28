# Upgrade Notes

1. Wait for the immutable `pantheon-base-v0.13.0` tag and GitHub Release to be published after required checks pass.
2. Update `pantheon-ops/foundation-release.lock.json` to the published manifest commit; do not consume `main` or this candidate commit directly.
3. Rebuild the consumer snapshot and business overlay in an isolated, clean Ops worktree.
4. Run inheritance, boundary, backend, frontend, and business smoke gates before merging the consumer upgrade.

No database migration or configuration rewrite is required by the governance changes in this candidate.
