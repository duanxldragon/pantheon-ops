# Ops Consumption of pantheon-base-v0.18.2

## What was consumed

- Locked release: `pantheon-base-v0.18.2`, release line `release/0.18`, base commit
  `a44ae4e012fff20430b1b7f91cb7a9296f3ea843` (squash merge of base PR #479), annotated tag object
  `2e32bd465678e7259604ac2a16f4c8cb0e53308e`, published `2026-10-31T00:48:38Z`.
- Materialized under `.foundation/releases/pantheon-base-v0.18.2/` (tgz + `repo.tar` + extracted
  `repo/`). Both digests were re-hashed locally and match the lock and the GitHub asset digests:
  - tgz `c01ebbc1c515ae09ba097b421071800cc1e19fec516fea9b147b35b0b2fc356a` (30,051,238 B)
  - repo.tar `c1360fb147610ff8629cc952d50491b993b04c07993d41f6c29eec37ecfc5d51` (47,923,200 B)
- `manifest.json` inside the tgz agrees with the lock on `releaseVersion`, `baseCommit` and all
  three `sharedPaths` groups; the groups grew from 52 to 53 entries because
  `frontend/scripts/check-menu-contract.mjs` is now shared (backend 4 / frontend 47 / docs 2).
- Release Gate run `41146688290` is success on the tagged commit. Foundation Release Verify run
  `41153048213` is success after a manual `workflow_dispatch`: the release was published by
  `github-actions[bot]` with `GITHUB_TOKEN`, and GitHub suppresses `release: published` events for
  those, so the tag-triggered fan-out never fired by itself.
- The rebuild ran through the lock path (`node scripts/business-overlay/rebuild-from-base.mjs`, no
  `--base`) and re-derived 789 declared/generated files.

## What the consumer actually changed

The release delta is one shared sensor. Everything else that moved in ops is either generated base
content or the repair that sensor forced:

| Path group | Move |
| --- | --- |
| `VERSION` | `0.18.1` → `0.18.2` |
| `docs/release-notes/` | index gained v0.18.2; `RELEASE_NOTES_v0.18.2.md` added |
| `frontend/scripts/check-menu-contract.mjs` | new shared sensor (116-line delta vs the 0.18.1 copy) |
| `tests/scripts/check-menu-contract.test.mjs` | new, from base |
| `frontend/src/i18n/resources/business/*.ts` | regenerated from the business i18n sources |
| `backend/modules/business/k8s/k8s.go` | four `k8sServicePerm*` constants |
| `backend/modules/business/k8s/k8s_seed.go` | 20 static menu seeds + 40 permission i18n rows replace an `init()` derivation |
| `frontend/src/modules/business/k8s/index.ts` | 16 missing permission declarations |
| `frontend/src/modules/business/v12locales/*.json` | 20 permission titles + 2 restored placeholders |
| `business-overlay.json`, `rebuild-from-base.mjs` | gorm gate declared as overlay, script names kept in the merge |
| `package.json` | base script set, ops-only scripts preserved |
| `tests/scripts/{check-pr-governance,quality-workflow,release-gate-workflow}.test.mjs` | base test revisions |
| `docs/reviews/CODE_SIMPLIFICATION_PLAN.md` | base doc revision |
| `.business-overlay-report.json` | base commit `a44ae4e0` |

`frontend/package.json` and `frontend/package-lock.json` were deliberately **not** overwritten: the
live tree carries the merged Dependabot bumps (i18next 26.5.11, react-i18next 17.1.9) while base
still pins 26.5.6 / 17.1.3.

## The sensor's first consumer finding

`frontend/scripts/check-menu-contract.mjs` parses Go seed struct literals and requires every bare
identifier to resolve to a constant in the same package. The k8s module built its permission menu
seeds at runtime:

```go
titleKey := strings.ReplaceAll(item.perm, ":", ".") + ".permission"
```

That is not a resolvable constant, and the resulting key spelling
(`business.k8s.cluster.sync.permission`) exists in no i18n catalogue. The frontend build failed with
`后端 seed 引用了无法解析的常量: Key: titleKey` twice. The repair removes the `init()` loop and
unrolls 20 static menu seeds plus 40 zh-CN/en-US permission i18n rows on canonical constants. The
seeded set is unchanged in size: the sensor still counts 40 menus / 50 routes / 237 permissions /
52 component keys.

## Rebuild-completeness repairs

1. **`check-gorm-sql-literal` was undeclared.** `scripts/check-gorm-sql-literal.mjs` and
   `tests/scripts/check-gorm-sql-literal.test.mjs` predate the overlay declaration, so a clean
   rebuild would have deleted both files and dropped `check:gorm-sql-literal` /
   `test:gorm-sql-literal` from `package.json`. Both paths are now `repositoryOverlayPaths` and both
   script names are in `opsOnlyScripts`.
2. **Two i18n keys existed only in generated files.**
   `business.observability.monitoring.field.endpointPlaceholder` and `...urlPlaceholder` were present
   only in `frontend/src/i18n/resources/business/*.ts`, which the rebuild regenerates from the
   business JSON sources; they would have silently disappeared. Both are now in
   `frontend/src/modules/business/v12locales/{zh-CN,en-US}.json`.

## Gates on the applied tree

| Gate | Result |
| --- | --- |
| `check:shared-drift --base ../pantheon-base/.tmp/rel-0182` | OK, 813 identical / 10 allowlisted / 161 skipped |
| `check:business-overlay` | OK, 53 declared paths |
| `test:business-overlay` | 13/13 pass |
| backend `go build` / `go vet` / `gofmt -l` | 0 / 0 / clean |
| backend `go test -count=1 ./...` | 64 packages ok, no FAIL |
| frontend `lint` / `type-check` / `build` | 0 / 0 / `✓ built in 804ms`, sensor 40 menus / 50 routes / 237 permissions / 52 component keys |
| Docs Governance command set | all exit 0 |
| `check:task-packet`, `check:harness-adoption`, `check:failure-registry`, `check:business-*`, `check:duplication`, `check:harness-docs`, `check:gorm-sql-literal` | all exit 0 |

## Known gaps

- **Runtime acceptance** was not re-run this round. v0.18.2 moves no migration baseline and the k8s
  change is seed spelling only (same 237 permission keys), but the startup/seeding path itself has
  no fresh reading in this evidence set; the v0.18.1 isolated-schema run remains the last one.
- **`check:harness-docs`** still reports dangling links inherited from base documents that point at
  base-only task packets and the base evidence archive. The check is report-only and the repair
  belongs to base.
- **`go test -race`** cannot run on this Windows host (ThreadSanitizer shadow-memory allocation fails
  with error code 87); the plain test run passes and CI runs the race variant on Linux.
- **SonarCloud Scan / Vercel** carry forward unchanged from the v0.18.1 round: the `SONAR_TOKEN`
  secret is rejected with HTTP 403 on hosted runs, and the Vercel project configuration lives outside
  the repository. Neither is a required context.
