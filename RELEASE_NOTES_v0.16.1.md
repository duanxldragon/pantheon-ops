# Pantheon Base v0.16.1 — Release Notes

**Release date**: 2026-10-12
**Tag**: `pantheon-base-v0.16.1`
**Previous release**: `pantheon-base-v0.16.0`

## Summary

v0.16.1 is a patch release on top of v0.16.0. It carries no breaking changes, adds no system
module, and changes no consumer-facing interface: no API route, permission code, menu seed, i18n
key or shared-contract shape moves.

It does two things. First, it root-causes the two recurring `main`-branch flakes that had been
dismissed as noise, fixing each at the race rather than by loosening an assertion. Second, and more
consequentially for the consumer, it publishes the **complete four-asset foundation bundle** —
something v0.14.0, v0.15.0 and v0.16.0 all failed to do.

## Fixed

### Data-scope caches were keyed by the connection pool's address (#377)

`buildDatabaseCacheNamespace` used `fmt.Sprintf("%p", sqlDB)`, so the process-local
"does `system_role_data_scope` exist" and user-dept caches were scoped to a memory address rather
than to a database. `testmysql.Open` drops each per-test database in `t.Cleanup`, which makes the
pool unreachable and lets Go recycle the address; a fresh test database then inherited the previous
one's cached verdict, `MigrateDataScopePolicy` skipped its `CREATE TABLE`, and the next `INSERT`
failed with `Error 1146 (42S02)`. The failure presented as a real backend regression and forced
main-branch reruns.

The cache key now derives from the DSN's `host:port/dbname`, with a pool-pointer fallback for
non-MySQL dialectors. Production always passes the process-wide singleton pool, so no
consumer-visible behaviour changes — this is hardening against a defect that was reachable only
through test fixtures.

### A smoke spec sampled a CSS transition mid-flight (#377)

`platform-shell-critical.spec.ts` read the sidebar's `boundingBox()` immediately after clicking the
collapse toggle, while Arco Sider's 200 ms width transition was still running (248 vs 248 on one
attempt, 215.5625 vs 108.265625 on the retry). The `toBeVisible` wait that followed the click
returned immediately, because the sidebar is always visible.

Browser layout measurements now go through a shared `settleElementLayout` helper that waits for the
box signature to stabilise and for `getAnimations()` to finish. The pre-existing bounds stay in
force.

## Added

### A gate that asserts a published release carries its artifacts (#380)

`scripts/foundation-release/check-release-assets.mjs` verifies the four-asset consumable set and
validates each local archive against its checksum sidecar. It reuses the publish path's own
`validateReleaseAssetChecksum`, so verification semantics cannot drift from publication. It carries
21 hermetic unit tests (no network, no subprocess).

- `Foundation Release Verify` runs it on `release: published`, so a release published without
  assets fails at the moment it is created instead of being discovered by the next consumer.
- `Foundation Release Backfill` repairs a release that was already published without assets.
- `Foundation Release Cut` runs the canonical `release:foundation:cut` and
  `release:foundation:publish` scripts on a runner where `git`, `tar` and `gh` all work.

Both new workflows are `workflow_dispatch`-only and default to `dry_run: true`.

## Why the artifacts matter

`pantheon-ops` does not consume Base from a sibling working tree. It materializes a release from
`repo.tar` plus the bundle archive and verifies both by SHA-256, using the checksums recorded in its
lock. A release without assets is therefore not merely incomplete — it is **unconsumable**, and the
lock cannot advance to it.

Because v0.14.0, v0.15.0 and v0.16.0 were each published with zero assets, the ops lock has been
stuck at `pantheon-base-v0.13.0` and the baseline silently fell three minor versions behind.
v0.16.1 restores the channel:

| Asset | Consumed as |
| --- | --- |
| `foundation-release-pantheon-base-v0.16.1.tgz` | The bundle; verified against `releaseArtifact.checksum` |
| `foundation-release-pantheon-base-v0.16.1.tgz.sha256` | Its checksum sidecar |
| `repo.tar` | Extracted to `repo/` and used as the consumer's `baseRoot` |
| `repo.tar.sha256` | Verified against `repoSnapshot.sha256` |

## Quality

- All required checks green on the release commit, plus the Release Gate.
- `sharedPaths` is unchanged from v0.13.0 onward: the same 52 entries (backend 4, frontend 46,
  docs 2), so a consumer lock needs only its version, commit and checksums advanced.
- The six business-overlay assembly anchors were re-verified against this base tree and each still
  resolves exactly once.

## Verification

- The two flake fixes are covered by deterministic tests rather than timing tolerances:
  `connection_cache_namespace_test.go`, `TestDataScopeCacheKeysAreScopedPerConnection` and
  `TestMigrateDataScopePolicyIgnoresForeignCachedVerdict`.
- The asset gate carries 21 hermetic unit tests.
- Hosted: required checks and Release Gate green on the release commit; `Foundation Release Verify`
  green on the published release.

## Consumer impact (`pantheon-ops`)

No base-side interface changed. What arrives inside the declared shared paths is one new internal
file (`backend/pkg/database/connection_cache_namespace.go`), a behaviour-preserving change in
`backend/internal/middleware/data_scope_middleware.go`, and a new test helper
(`frontend/tests/smoke/helpers/layout-settle.ts`). None alters an exported symbol the business
overlay depends on, and no overlay change is required.

Advance the lock to `pantheon-base-v0.16.1` / `release/0.16` and run `npm run rebuild:from-base`.
