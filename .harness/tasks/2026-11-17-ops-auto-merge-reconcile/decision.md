# Decision — Auto-merge token reconciliation (2026-11-17)

Parent: `2026-11-16-ops-historical-closeout` / G1-auto-merge-reconcile.

## Contradiction

`.github/workflows/pr-automation.yml` "Enable squash auto-merge" used
`GH_TOKEN: ${{ github.token }}` (deterministic, live), but two historical records
disagreed:

- `2026-08-12-pr-auto-merge-push-trigger`: prefer `secrets.RELEASE_GATE_TOKEN`, fail
  closed, no `github.token` fallback; add a manual exact-commit CI trigger.
- `2026-08-12-pr-auto-merge-token`: authenticate the step with `github.token`.

## Chosen approach: A (push-trigger contract wins)

`github.token` (default `GITHUB_TOKEN`) intentionally suppresses push-triggered
workflows that share the repository, so an auto-merge authorized by it does not fire
the post-merge push signal that foundation publication needs. The release-gate token
(`secrets.RELEASE_GATE_TOKEN`) authorizes the merge in a way that DOES fire the
post-merge push workflow on the merged SHA.

1. Auto-merge step uses `GH_TOKEN: ${{ secrets.RELEASE_GATE_TOKEN }}`.
2. Fails closed when the token is absent
   (`if [ -z "${GH_TOKEN:-}" ]; then echo ...exit 1; fi`) — never falls back to
   `github.token`.
3. Failure propagates (`set -euo pipefail` + `exit 1`), satisfying the token record's
   fail-on-enable requirement.
4. **No `.github/workflows/ci.yml` is created.** Code Quality Gates
   (`quality.yml`) already provides the exact-commit entrypoints: `workflow_dispatch:`
   for manual exact-commit recovery, and `push:` on `main`/`release/**` that re-runs on
   the pushed merge SHA after auto-merge. Reusing it avoids a duplicate/empty CI
   workflow invented only to satisfy an old test's hard-coded filename.
5. `2026-08-12-pr-auto-merge-token` is **superseded** by this reconcile; its task doc
   status → `Superseded`, `superseded_by` → this package. The push-trigger doc
   `2026-08-12-pr-auto-merge-push-trigger` remains the in-force contract.

## Scope boundary

- CI/repo-config domain only — no `pantheon-base` change (Base-first respected).
- Did not create/read/modify any Secret, did not run `gh merge`, publish, or release.
- Preserved `governance` / `feedback` / `draft` / same-repo / author-association gates
  and failure propagation in `pr-automation.yml` unchanged.
- Did not touch `dependabot-pr-governance`, `quality.yml` (G2-owned), or other tasks.
