# Ops Consumption of pantheon-base-v0.17.7

## What was consumed

- Locked release: `pantheon-base-v0.17.7`, release line `release/0.17`, base commit `2091a2d8409e6b4ed933b19c8ff8870f2d0eedbe` (squash merge of base PR #430), annotated tag object `6866fee007731280ee4914ffc2fc51769bcf3755`, published `2026-10-27T04:52:26Z`.
- Materialized under `.foundation/releases/pantheon-base-v0.17.7/` (tgz + `repo.tar` + extracted `repo/`).
- Both asset digests were re-hashed locally and match the lock, the published `.sha256` sidecars and the GitHub asset digests:
  - tgz `a101f5f6376e92bf6dc10bea4fff0954ab43202d123d6030abfc61a58de59585`
  - repo.tar `6b62cd453d949a85f3a70b8a82eb4a69f6470c2f404b53202a57664ea5dad98c`
- `manifest.json` read out of the tgz agrees with the lock on `releaseVersion`, `baseCommit` and all three `sharedPaths` groups.
- Release Gate run `40450323981` and Foundation Release Verify run `40452121919` are both success on the tagged commit; base itself reports 0 open code scanning alerts and a passed SonarCloud quality gate on that commit.
- The rebuild ran through the lock path (`node scripts/business-overlay/rebuild-from-base.mjs`, no `--base`), so the consumer only ever saw the published snapshot. Declared/generated files: 751.

## Switch shape

v0.17.7 is an incremental advance, not a baseline jump. Applying the rebuilt tree moved 6 tracked paths and added 1:

| Path | Why it moved |
| --- | --- |
| `VERSION` | `0.17.6` → `0.17.7` |
| `backend/pkg/logging/logger.go` | four `go/log-injection` false-positive annotations on the zap sinks |
| `backend/modules/system/config/setting/setting_service.go` | `//nolint:revive` on `SettingService`, which clears the hosted `Go Lint` revive finding |
| `frontend/scripts/tenant-matrix-fixture-setup.mjs` | seven `js/file-access-to-http` false-positive annotations |
| `RELEASE_NOTES_v0.17.7.md` | new base-owned release note |
| `foundation-release.lock.json`, `.business-overlay-report.json` | lock identity and rebuild report |

`CHANGELOG.md`, `README.md` and `README.en.md` did **not** move even though base updated them, because those are Ops overlay paths declared in `business-overlay.json`; base's v0.17.7 entries stay in the base history and Ops keeps its own changelog. `go.mod` / `go.sum` were rewritten by the rebuild's `go mod edit -require` pass and `go mod tidy` returned them byte-identical to the v0.17.6 round, which is the evidence that v0.17.7 moved no dependency.

Nothing in this release touches an API, a DTO, a migration or a permission, so the schema baseline stays `000030` and no menu/permission re-seed is implied.

## Consumer-side repair in this round

**`goconst` on the deploy package's nginx service name.** The hosted `Go Lint` job failed on two findings; one was base-owned and is fixed by the consumed `nolint`, the other is ops-owned: `backend/modules/business/deploy/deploy_rollback.go:81` used the literal `"nginx"`, which appears seven times in the package. Four of them are production code — the component-type map key at `deploy_helpers.go:186`, the systemd unit fallbacks at `deploy_rollback.go:81`, `deploy_rollback.go:126` and `deploy_service.go:3957`. The value is now named once as `defaultNginxServiceName` in `deploy_model.go` and all four sites reference it; only test fixtures still spell the literal, and `goconst` does not count those. `golangci-lint` with the CI scope (`--new-from-rev=edcca7eb`) reports 0 issues.

## Gate disposition: the code annotations were not enough

Base's v0.17.7 notes present the inline `// codeql <rule>: this is a false positive` comments as the suppression. That is only half true, and the consumer measured it: alerts 56-64 in this repository were created *after* the annotated commits were pushed and stayed open. CodeQL reads those comments into its own result metadata at analysis time on some deployments, but the branch/alert gate that `Security Gates` and the PR feedback gate look at is GitHub's alert state, and that only changes through the code scanning API. The nine alerts were therefore dismissed individually, each after re-reading the code it points at:

- **56** `go/log-injection`, `backend/pkg/logging/logger.go` — **false positive**. `SanitizeLogValue` maps CR, LF, U+2028 and U+2029 to a space and drops every other control character except TAB, and `sanitizeLogFields` applies it to the field key, string value and error text before the sink call, so no logged value can carry a line break.
- **57-63** `js/file-access-to-http`, `frontend/scripts/tenant-matrix-fixture-setup.mjs` — **used in tests**. The file named as the taint source is the manifest this same fixture writes into gitignored `.tmp`, and every value read back (tenant id, admin username, smoke password) is used to call the backend instance the fixture just provisioned. The script never ships to a browser.
- **64** `go/path-injection`, `backend/modules/business/filedist/storage_service.go:115` — **mitigated**, not dismissed as noise. `normalizeLocalStoragePath` trims, rejects an empty root and rejects any parent-directory segment before `filepath.Clean`, and object paths are built separately at `:254` through `filepath.Base(fileName)`, so the traversal the rule models cannot escape the root. What remains true is that an administrator may point a local storage location at any directory the process can write — that is the feature, reachable only through the permission-gated storage CRUD API, so it is recorded as an accepted mitigation rather than a defect.

This host's REST schema rejects `dismissal_comment`, so the reasons live here and in `review.md` rather than on the alert object.

## Verification summary

See `commands.json` for per-command readings. Everything executed passed: asset digests and manifest equivalence, overlay contract (53 declared paths), shared drift (805 identical / 10 allowlisted / 160 skipped against a Base checkout that is on the consumed commit), 13/13 business-overlay tests, `go mod tidy` with no residual diff, backend build/vet/gofmt/test (63 ok, RC 0), the deploy package tests after the constant repair, `golangci-lint` at 0 issues in the CI scope, and the governance set (encoding 2065 files, structure 2238 files, frontmatter 404 docs, duplication PASS). GitHub-side: open code scanning alerts went 9 → 0 on the PR ref, all nine review threads read `isResolved`, and `fetch-github-feedback --check` now exits 0 with no BLOCKED items.

## Known gaps

- No runtime acceptance run (backend startup + `golang-migrate` to `000030` + tenant-operations seeding) in this evidence set; no database credentials are recorded here. v0.17.7 does not move the schema, so this gap is inherited unchanged.
- The hosted `SonarCloud Scan` job fails with `GET https://api.sonarcloud.io/analysis/jres ... 403 Forbidden` while `SONAR_TOKEN` and `SONAR_HOST_URL` both exist as repository secrets, i.e. the credential itself is stale. It is not a required context; rotating it is a maintainer action.
- Ops has no consumer-side versioned migration channel. Production code in `backend/modules/business` contains 22 ungated `AutoMigrate` call sites across 22 files (for example `deploy/deploy_service.go:188`, `filedist/storage_service.go:46`, `k8s/backup/backup_service.go:47`), and none of them consults `database.ShouldAutoMigrate()`. The earlier round recorded "four modules"; that figure was carried forward without re-measuring and is corrected here.
- Frontend `tsc`/`eslint`/`vite build` were not re-run locally because the delta contains no `frontend/src` file; the hosted Frontend Contract and Smoke Sanity jobs cover them on the pushed commit.
