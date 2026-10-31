# Pantheon Base v0.18.2 — Release Notes

Patch release on the `release/0.18` line. Supersedes `pantheon-base-v0.18.1` as tag
`pantheon-base-v0.18.2`.

v0.18.1 shipped the platform as it stood on 2026-10-29. Since then the menu-contract sensor and the
release gate were repaired, and the retired-module menu filter was rewritten around whole-clause
constants. Two of those changes live in the shared surface. This patch publishes them; nothing on the
platform plane changes behaviour.

## Summary

- **No schema change.** The migration chain still ends at `000030_tenant_plan_definitions`
  (schema 30).
- **No route, permission code, menu key or i18n key change.**
- `sharedPaths` grows from 52 to **53** entries: `frontend/scripts/check-menu-contract.mjs` is now
  declared shared, because a consumer runs it in its own `prebuild` and therefore needs the base
  version of it.
- Two consumer-visible files change: `frontend/scripts/check-menu-contract.mjs` and
  `backend/modules/business/retired_modules.go`.

## What changed

### Fixed — the menu-contract sensor

`frontend/scripts/check-menu-contract.mjs` compared backend menu seeds against the frontend module
descriptors. Two defects made it under-report:

1. **A bare identifier was matched by prefix.** The Go-constant resolver turned any
   `Key: <word>` into a constant lookup, so `item.Key`, `strings.ToLower(x)` and `menuKey(i)` were
   read as if they named package constants. Fields that had no constant table entry silently became
   `''`.
2. **An unresolved reference collapsed to `''`.** The caller then compared `''` against the frontend
   value, which produced a bogus mismatch — or, when the emptied value matched an emptied frontend
   field, no finding at all.

The sensor now resolves `const ( name = "value" )` blocks and single-line `const` declarations, only
treats a bare identifier as a reference (no `.`, `(`, `[` following it), and reports every unresolved
bare reference in a block it keeps as a finding that fails the check. The resolution helpers are
exported and covered by `tests/scripts/check-menu-contract.test.mjs` (`npm run test:menu-contract`).

### Fixed — the sensor reaches the consumer

`frontend/scripts/check-menu-contract.mjs` was absent from the release manifest's `sharedPaths`, even
though `pantheon-ops` carries a byte-identical copy and runs it from its frontend `prebuild`. A file
that a consumer executes must be published, or the next repair stops at the base repository. The file
joins `sharedPaths.frontend` and `requiredFrontendEntries` in
`scripts/foundation-release/frontend-ownership.mjs`, so a future removal from the list fails the
manifest build instead of passing silently.

### Changed — the retired-module menu filter

`backend/modules/business/retired_modules.go` built its filters as `column + " IN ?"`. The columns were
call-site literals, so the statement text was assembled from a variable. The three clauses are now
package-level constants (`menuModuleInClause`, `menuPathInClause`, `menuComponentInClause`) annotated
for `scripts/check-gorm-sql-literal.mjs`, and `applyMenuFieldFilter` takes the whole clause. The
generated SQL is the same and the values are still bound through `?`; what changes is that the clause
is a literal a reader and a sensor can both see. This is the base-first half of the shared-drift
convergence that `pantheon-ops` completed on its side.

### Governance — release gate and PR governance

- Six `githubactions` findings across `ci.yml`, `quality.yml`, `release-gate.yml` and `security.yml`
  were cleared in #473 (`c1f0b1dd`).
- The release gate in #478 (`beace427`) treats a transiently failed check-list API read as "no verdict
  yet" and keeps polling inside its existing budget, instead of failing the job on the read; the
  terminal timeout still fails closed.
- #474 (`05ee0ec1`) archived six completed task packets as renames and taught `check-pr-governance`
  the documented monthly-archive fallback, so a completed packet can be archived without breaking the
  PR that references it.

## Upgrade notes

- There is no migration and no configuration change. Move the lock to `pantheon-base-v0.18.2`
  (`release/0.18`), rebuild the tree and re-run the builds.
- `sharedPaths` gained one entry. A consumer whose rebuild script or lock validator pins the count must
  accept 53; `rebuild-from-base.mjs` and `check-shared-drift.mjs` both read the list from the lock and
  need no change.
- Expect `check:shared-drift` to report the two files above as modified until the rebuild writes the
  release versions over the live tree.

## Consumer impact

- `pantheon-ops` consumes both files. Its `frontend/scripts/check-menu-contract.mjs` copy is identical
  to base v0.18.1 and is replaced by the release; its `backend/modules/business/retired_modules.go`
  copy already carries the whole-clause constants from its own convergence work, so the rebuild
  produces an identical file and the drift reading returns to 0 modified.
- Acceptance reading: `npm run rebuild:from-base`, `npm run check:business-overlay`,
  `npm run check:shared-drift` (0 modified), backend `go build ./...` and frontend `npm run build` in
  the rebuilt tree.

## Verification

- `node frontend/scripts/check-menu-contract.mjs` → `✅ manifest / menu seed 一致性检查通过`.
- `npm run test:menu-contract` → parser and resolver tests green.
- `node --test tests/scripts/foundation-release/` → manifest builder, bundle builder and secret checker
  green with the 53-entry list.
- Release assets and `Foundation Release Verify` follow the same four-asset contract as v0.18.1.
