# Ops Consumption of pantheon-base-v0.18.1

## What was consumed

- Locked release: `pantheon-base-v0.18.1`, release line `release/0.18`, base commit `93505225e36c8d3ba5aa9facc12975eb813167d3` (squash merge of base PR #469), annotated tag object `b63974357334aa6765e6a07bee9f12841dcb784c`, published `2026-10-29T04:19:38Z`.
- Materialized under `.foundation/releases/pantheon-base-v0.18.1/` (tgz + `repo.tar` + extracted `repo/`).
- Both asset digests were re-hashed locally and match the lock and the GitHub asset digests:
  - tgz `2fe49d70f8ac628351fc51076026cecb9c35cfa76180f7cdf3fdb5092ac77f03` (29,888,926 B)
  - repo.tar `37ae7a0873f9e49a076f21fdaa53651cbce1b72160d98e04a4a04cfc132303d9` (47,411,200 B)
- `manifest.json` read out of the tgz agrees with the lock on `releaseVersion`, `baseCommit` and all three `sharedPaths` groups.
- All eight base workflows on the tagged commit are success, including Release Gate run `40814653285` and Foundation Release Verify run `40817118092`; base reports 0 open code scanning alerts.
- The rebuild ran through the lock path (`node scripts/business-overlay/rebuild-from-base.mjs`, no `--base`), so the consumer only ever saw the published snapshot. Declared/generated files: 756.

## Why the consumer had to move

v0.18.0 failed the ops frontend build. The simplification wave S1 (#439, `b0a1952a`) removed the
`isForbiddenRequestError` export from `frontend/src/api/request.ts` because a search inside
`pantheon-base` found no callers, while `frontend/src/modules/business/deploy/task/DeployTaskDetail.tsx:29`
imports it and branches on it at `:182`. The build died with `TS2614`. v0.18.1 restores the export;
that is the entire reason this jump is to a patch rather than to v0.18.0.

## Switch shape

| Path group | Move |
| --- | --- |
| 7 base source files | deleted: logging middleware, four `system/iam/tenant` files, `pkg/common/token.go`, `TenantSwitcher.tsx` |
| 13 `frontend/scripts/capture-*` and helpers | deleted |
| 14 backend test/pkg files, `tenant-console.spec.ts`, harness doc-link test | added |
| 18 release notes under `docs/release-notes/` | added; the 13 root `RELEASE_NOTES_v0.13.1..v0.17.7` files moved there |
| 9 review docs under `docs/reviews/` | added |
| `VERSION` | `0.17.7` → `0.18.1` |
| `backend/go.mod`/`go.sum` | go directive `1.26.9` → `1.26.10`, go-oidc `3.9.0` → `3.21.0`, go-jose v3 → v4 |
| `Dockerfile` | builder image `golang:1.26.5-alpine` → `golang:1.26.10-alpine` |
| `frontend/package.json` | `test:smoke:tenant` gains `tenant-console.spec.ts` |

Final staged shape: 130 modified, 30 added, 20 deleted, 13 renamed.

## Consumer drift repaired in the same round

1. **`scripts/task-manifest.mjs` overlay pin.** The file was pinned as an ops overlay path from the
   pre-archive era and carried no ops-specific customisation. Base 0.18 made it archive-aware and
   `scripts/harness/check-task-packet.mjs` plus `check-doc-links.mjs` now import
   `resolveArchiveAwarePath` from it, so both checkers died with a module SyntaxError. The pin was
   removed and base's copy adopted.
2. **`check:ui-quality-gate` workflow step.** Base 0.18's gate requires the Docs Governance job to
   run it; the ops workflow did not, so the gate reported `ui_gate_workflow_step_missing`.
3. **Manifest status vocabulary.** Base's canonical values are `planned / in-progress / completed /
   abandoned`; eleven ops manifests still said `complete`. Two were caught by `check:task-packet`,
   all eleven were normalised.
4. **`scripts/cleanup-github-branches.mjs` overlay pin.** The pinned copy threw on `404` and `403`
   where base's exported `deleteBranch` returns `already-gone` / `protected-or-rejected`. Base's copy
   is adopted and the pin removed; the inherited branch-hygiene test passes 4/4.
5. **Dependabot could never pass the governance prerequisite.** `pr-automation.yml` ran the PR body
   ceremony unconditionally while `check-pr-governance.mjs` has no Dependabot branch, so every
   Dependabot PR was red and auto-merge unreachable. The workflow now mirrors base's exemption.
6. **Docs Governance never ran `check:task-packet`.** That is why the crashed checker above stayed
   invisible. The checker is fixed and green for current-format packets, but re-enabling it in CI
   needs a 60-entry allowlist for pre-template Ops packets and is deferred to its own task. The
   stale `setup-node` pin in the ops-owned `workflow-baseline.test.mjs` was updated to the pin
   every workflow already uses.

## Gates on the applied tree

| Gate | Result |
| --- | --- |
| `check:shared-drift --base ../pantheon-base` | OK, 812 identical / 10 allowlisted / 160 skipped |
| `check:business-overlay` | OK, 53 declared paths |
| `test:business-overlay` | 13/13 pass |
| backend `go build` / `go vet` / `gofmt -l` | 0 / 0 / clean |
| backend `go test ./...` | RC 0 |
| frontend `lint` / `type-check` / `build` | 0 / 0 / `✓ built in 886ms` |
| Docs Governance command set | all exit 0 |
| `check:task-packet`, `check:harness-adoption`, `check:failure-registry`, `check:business-*`, `check:duplication` | all exit 0 |

The staging tree was gated with the same set before the live replacement, including the frontend
build that proves the `TS2614` break is gone (`✓ built in 725ms`).

## Hosted verification after the merge

- PR #151 squash-merged to `main` as `fc6bc850` at 2026-10-29T07:44:33Z; PR #150 (i18next 26.5.11 /
  react-i18next 17.1.9, the only Dependabot PR still valid after the 0.18.1 dependency refresh)
  squash-merged as `165014c8` at 2026-10-29T07:59:50Z.
- Six Dependabot PRs were closed as obsolete because `main` already carries newer versions from the
  0.18.1 tree: #129 js-yaml, #130 vitest, #131 baseline-browser-mapping, #135/#136 otlptrace 1.45.0
  and #137 root npm group. After the merges `git ls-remote --heads origin` returns only `main`, and
  no PR is open.
- Push pipelines on `165014c8`: Code Quality Gates `40839019495` success, Security Gates
  `40839019357` success, Branch Hygiene `40839019345` success, SonarCloud Scan `40839019352`
  failure (HTTP 403 from api.sonarcloud.io: the `SONAR_TOKEN` secret is rejected).
- Dispatched on the same commit to prove the gates against the final tree: Code Quality Gates
  `40840649887` success, Security Gates `40840655454` success (frontend `npm audit` 0
  vulnerabilities, OSV call analysis 3 advisories all `called:false`), Branch Hygiene
  `40840665983` success, Base Compatibility Rebuild `40840676477` success.
- Branch protection requires only `Quality Gates` and `Security Gates`; both are green on every main
  push and on the merge PRs.

## Known gaps

- **Runtime acceptance** (backend startup + `golang-migrate` + seeding) still has no consumer
  evidence run; v0.18.x moves no migration baseline, so the gap is inherited unchanged.
- **`check:harness-docs`** reports four dangling links, all inherited from base documents pointing
  at base-only task packets and the base evidence archive. The check is report-only, is not wired
  into any ops workflow, and its repair belongs to base.
- **`go test -race`** cannot run on this Windows host (ThreadSanitizer shadow-memory allocation
  fails with error code 87); the plain test run passes and CI runs the race variant on Linux.
- **SonarCloud Scan** is the only failing GitHub Actions workflow. Two distinct causes: the
  `SONAR_TOKEN` secret is rejected with HTTP 403 on push and normal-PR runs, and Dependabot PR runs
  have repository secrets withheld by GitHub, leaving `SONAR_HOST_URL` empty ("URI with undefined
  scheme"). Rotating the token, and either adding Dependabot secrets or skipping the job for
  Dependabot, are maintainer actions; the check is not a required context.
- **Vercel** is an external integration, not a GitHub Actions workflow: production deployment
  `7407012001` (`dpl_oh9Pv9fibTSxf72d7QWfrtuW5Kuq`) for `165014c8` failed, as did every preview
  build before it. The repository has never carried a `vercel.json`, so the configuration lives in
  the Vercel project and needs dashboard access to diagnose.
