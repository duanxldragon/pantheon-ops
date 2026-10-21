# Foundation Upgrade Evidence — pantheon-base-v0.16.1

Task: `2026-10-12-foundation-upgrade-base-v0-16-1`.

## Outcome

The ops consumer moved from the `pantheon-base-v0.13.0` baseline (`c254e4c8`) to
`pantheon-base-v0.16.1` (`c83f8e4c`).

- `foundation-release.lock.json` now pins `release/0.16`, tag `pantheon-base-v0.16.1`, base commit
  `c83f8e4c865b19ecd5351ecefac4106ec416e0bc`, bundle checksum `f22305e1…` and `repo.tar`
  checksum `daf8c890…`.
- `.foundation/foundation-release.lock.json` no longer claims the never-published `v0.13.1`; it
  records the same release, the verified digests and the materialization path.
- `.foundation` is now a declared `repositoryOverlayPaths` entry, so the drift allowlist and the
  release ledger survive a baseline swap instead of disappearing from the rebuilt tree.
- `docs/harness/tasks` is now pruned from the base tree before the ops overlay is applied. The
  directory is ops-owned, but base's own release task packets used to survive the copy and made
  the task-packet gate fail on `pantheon-ops`.
- The ops tree was replaced by the verified rebuild of that release, preserving every declared
  business overlay asset.

## Verification

| Check | Result |
| --- | --- |
| bundle + `repo.tar` SHA-256 vs published sidecars / GitHub digests | match |
| `repo.tar` tree vs tag `c83f8e4c` by file hash | 2250/2252 identical; the 2 differences are `.bat`/`.ps1` files with `eol=crlf` in `.gitattributes` |
| `sharedPaths` lock vs v0.13.0 vs v0.16.1 | identical, 52 entries |
| rebuild from the locked release | OK, 719 declared/generated files |
| `check-business-overlay` (staged and swapped tree) | OK, 53 declared paths |
| `check-shared-drift` vs the v0.16.1 snapshot | 704 shared files identical, 10 allowlisted overlay hooks, 0 unexpected |
| `test:business-overlay` | 13/13 pass |
| `go build ./...` / `go vet` on business + `system/iam/menu` | clean |
| `go test ./modules/business/... ./modules/system/iam/menu/...` | all pass |
| frontend `npm ci` + `type-check` | clean |
| `check:business-boundaries` | pass |
| `check:docs-frontmatter` / `check:encoding` / `check:structure` | clean |
| `check-task-packet` | 59 failing legacy packets, identical count on the pre-swap commit — pre-existing debt, the upgraded packet passes |

## Repairs made because the rebuild exposed them

- `rebuild-from-base.mjs` derives the overlay menu registry package from the copied base tree.
  Base v0.16.0 renamed that directory to `package menu`; the hardcoded `package iam` had made
  every rebuild fail since 2026-09-14.
- The hosted `Base Compatibility Rebuild` now runs `go mod tidy` before compiling. The overlay
  merges Base's requirement graph with only the business modules' direct dependencies, so the
  assembled tree is not buildable until the indirect requirements are recomputed.

## Not done / limits

- `check-shared-drift --base ../pantheon-base` still reports drift against base **main**: the
  release tag is 13 commits behind main (`#385`, `#391`–`#394`). Those commits reach ops through
  the next published release; consuming base main would break the immutable-release contract.
- The published `Base Compatibility Rebuild` workflow result is pending a push; this upgrade is
  committed on the local dev line only.
- `go test ./...` (and the sqlite-backed `system/iam/tenant` package) cannot run on this machine:
  the local cgo toolchain fails even for a trivial cgo program, independently of this change.
- The v0.16.1 `verification-summary.json` lists the six required checks without per-check results,
  so no SonarCloud issue count is asserted in the ledger.
