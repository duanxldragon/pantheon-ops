# Review

## Findings

- `business/deploy` used `len(base) + len(override)` as an allocation capacity. Removing the preallocation eliminates the CodeQL integer-overflow path while preserving parameter precedence, covered by a focused unit test.
- Dependabot proposed TypeScript 7 while the locked `typescript-eslint` peer range is `<6.1`, causing deterministic `npm ci` failures. The incompatible TypeScript major update is ignored; no peer-dependency override weakens CI.
- Base-derived quality controls now cancel stale runs, validate new Go code with golangci-lint on PRs and merge groups, and execute quality gates after main and release pushes. Full-repository lint remains report-only on push until inherited historical lint debt is retired.
- Zizmor's cache-poisoning check required the new Go lint job to disable `setup-go` dependency caching; the workflow now uses the same no-cache posture as the existing Ops jobs.

## Residual Risk

Hosted CodeQL must recalculate the historical shared-path and cookie alerts against the current Base foundation. Hosted smoke remains the runtime evidence gate.
