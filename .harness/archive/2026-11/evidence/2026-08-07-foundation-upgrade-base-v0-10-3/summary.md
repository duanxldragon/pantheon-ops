# Foundation Upgrade Evidence: pantheon-base v0.10.3

Date: 2026-08-07
Repository: pantheon-ops

## Release identity

- Release: `pantheon-base-v0.10.3`
- Release line: `release/0.10`
- Base commit: `c8c5832d85f33f2f245526f4d0eab9df93225880`
- Archive SHA-256: `c0d3981c315cd323234e1272cc2cd473cbf114c213938abb4b4ff268a94e1239`
- `foundation:install`: passed
- Apply dry-run: `No changes needed`

## Implementation result

- Generated backend/frontend sources and inferred menus use canonical `business/*` and `/business/*`; handwritten CMDB, Deploy, and BizScope pages retain their intended `/operations/*` routes.
- BizScope backend, frontend, component, locale, and menu registrations are restored. Cleanup preserves the six handwritten backend/frontend business module directories.
- Generated registries moved from `modules/lowcode/generated` to `modules/generated`; an empty component registry no longer emits unused TypeScript helpers.
- Stale generator policy/smoke paths now use `/lowcode/generator/*`, with a permission-policy regression test.
- Ops backports Base commit `dec10461e` operation-token caching because the v0.10.3 manifest excludes `frontend/src/api`; opaque Redis tokens use stored metadata and a five-minute fallback TTL.
- Dynamic purge rebuilds registries before deleting source, preventing Vite/Go file watchers from observing dangling imports, then rebuilds the feature ledger from the final workspace.
- Dynamic i18n regeneration loads foundation and static business locales, uses canonical TypeScript formatting, and preserves deterministic Node-compatible key order.
- Database-import QA no longer mutates obsolete CMDB `host_code` data.

## Verification

Passed locally:

- `npm run check:inheritance`: Base v0.10.3 backend/frontend alignment, 180 foundation locale entries, 2895 generated keys, and 6 overlay consumer packages.
- `npm run test:foundation-release`: 71/71.
- Frontend `lint`, `type-check`, production `build`, generator smoke 5/5, menu contract (23 menus, 30 routes, 131 permissions, 32 component keys), and static i18n contract (3031 fallback keys).
- `npm run test:smoke:business`: 26/26 (CMDB 9, Deploy API 4, Deploy UI 10, master-detail 1, many-to-many 1, auto-recycle 1).
- Isolated module-governance-host 1/1 and dynamic i18n generation/purge check passed.
- With `PATH=D:\\msys64\\mingw64\\bin` and `CGO_ENABLED=1`, `go test -race ./...` passed for the full repository.
- Post-smoke cleanup retained handwritten `bizscope`, `cmdb`, and `deploy` backend/frontend directories and left no dynamic QA source directories.

## Repository debt snapshot

- Open PRs: 9, comprising 8 Dependabot PRs and feature/CI PR `#75`; all are `BLOCKED` by governance/quality checks.
- Remote branches: 11, comprising `main`, 8 Dependabot branches, and 2 feature branches. Ten branches are not merged into `origin/main`.
- PR `#75` has 28 remote commits over main. The current local branch is another 21 commits ahead of its remote head, so local HEAD is 49 commits ahead of `origin/main`.
- `origin/feat/lowcode-cmdb-retired-receive` is one commit ahead of main, has no open PR, and requires a human decision on whether it has been subsumed.
- Hosted checks remain external: the local v0.10.3 work must be pushed before GitHub can validate it. No remote merge, closure, deletion, or push was performed.

Local v0.10.3 implementation work is complete. Remaining work is remote integration and debt cleanup, estimated in `review.md`.
