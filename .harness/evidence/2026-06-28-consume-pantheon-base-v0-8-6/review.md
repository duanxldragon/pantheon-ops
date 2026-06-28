# Review: 2026-06-28-consume-pantheon-base-v0-8-6

## Verdict

findings addressed

## Findings

1. The `base-v0.8.6` sync initially left older `pantheon-ops` shared directories in place alongside the new base layout. That created duplicate ownership for `common`, `auth`, and `lowcode` paths and caused compilation and validation conflicts.
2. The current auth flow now uses secure cookies as the primary token transport. Smoke helpers still assumed raw tokens must be present in login JSON, which broke `Smoke Sanity` with `auth.login.response_invalid` even though backend startup and login execution were otherwise healthy.
3. The code sync itself was locally green, but the PR could not merge because the branch lacked task/evidence artifacts and a PR body that matched the repository governance template.

## Residual Risk

- This upgrade touches a large inheritance surface, so future `base -> ops` syncs should continue to prefer convergence over dual-layout compatibility.
- Full browser smoke still depends on CI service wiring; the local regression work here fixes the known auth contract mismatch but does not replace hosted end-to-end confirmation.
- Frontend lint will keep depending on the repository-local ignore boundary for temporary `.tmp` test output produced by harness scripts.

## Verification Checked

- `go test ./...`
- `npm --prefix frontend run lint`
- `npm --prefix frontend run build`
- `npm --prefix frontend run test:smoke:scripts`
- `npm run check:docs-frontmatter`
- `npm run check:task-packet-template`
- `npm run check:pr-governance`
- `npm run check:generated-modules`
- `gh pr checks 66 --repo duanxldragon/pantheon-ops`
