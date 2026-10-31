---
title: Consume pantheon-base-v0.18.2 into pantheon-ops
doc_type: Remediation
layer: inheritance-sync
status: Active
linked_contracts:
  - docs/PROJECT_INHERITANCE.md
updated_at: 2026-10-31
---

# Task Packet: 2026-10-31-foundation-upgrade-base-v0-18-2

Target repository: `pantheon-ops`
Layer: `inheritance-sync (pantheon-base -> pantheon-ops)`
Delivery tier: `L2`
Mode: immutable release consumption, patch jump 0.18.1 -> 0.18.2, sensor-exposed business repair

## Goal

Advance the Ops consumer from `pantheon-base-v0.18.1` to the published `pantheon-base-v0.18.2`
(`a44ae4e012fff20430b1b7f91cb7a9296f3ea843`) so the tree carries the new shared
`frontend/scripts/check-menu-contract.mjs` sensor, and repair what that sensor exposes on its first
consumer run: 20 k8s permission menu seeds whose `titleKey` was computed at runtime inside `init()`
and therefore unresolvable and untranslated. Two pre-existing rebuild defects are repaired in the
same pass so the next deterministic rebuild is lossless: the undeclared `check-gorm-sql-literal`
scripts and two i18n keys that existed only in generated files.

## Primary Layer

inheritance-sync

## Workspace Context

- Target Repository: `pantheon-ops`
- Repository Role: `business-consumer`
- Upstream Dependencies: `pantheon-base`
- Downstream Consumers: `none`
- Sync Expectation: `completed`
- Release Requirement: `consumer-lock-update`

## Dependency Layers

- `pantheon-base` foundation release pipeline (tag, four assets, Release Gate, Foundation Release Verify)
- Ops business overlay assembly (`scripts/business-overlay/rebuild-from-base.mjs`)
- Ops shared-drift gate (`scripts/business-overlay/check-shared-drift.mjs`)
- Shared menu-contract sensor (`frontend/scripts/check-menu-contract.mjs`, new in 0.18.2)
- Ops k8s business module seeds (`backend/modules/business/k8s/k8s_seed.go`)

## Harness Profile

- Template: api-service
- Overlay: pantheon-ops
- Quality Profile: foundation-release-consumer
- Portable Failure Class: static-sensor-gap
- Owner Layer: inheritance-boundary
- Coverage Dimensions:
  - behaviour
  - architecture-fitness
  - runtime-quality
  - method-health

## Contract Anchors

- `AGENTS.md`
- `docs/PROJECT_INHERITANCE.md`
- `foundation-release.lock.json`
- `.foundation/foundation-release.lock.json`
- `.foundation/allowed-drift.json`
- `business-overlay.json`
- `scripts/business-overlay/rebuild-from-base.mjs`

## Scope

### In

- Update `foundation-release.lock.json` to `pantheon-base-v0.18.2`: release line, version, base
  commit, artifact name, materialization path, bundle checksum and `repo.tar` checksum.
- Rewrite `.foundation/foundation-release.lock.json` with the release identity, the gate facts read
  back from GitHub, the v0.18.2 `pendingWork` and `notes`.
- Apply the rebuilt assembly to the live tree with an enumerated file copy, preserving the
  Dependabot bumps on `frontend/package.json` / `frontend/package-lock.json`.
- Replace the `init()`-derived k8s permission menu seeds with static canonical-constant seeds, add
  the four missing `k8sServicePerm*` constants, the 16 missing frontend permission declarations and
  the 20 permission title translations the new sensor requires.
- Declare `scripts/check-gorm-sql-literal.mjs` and its test as `repositoryOverlayPaths`, add the two
  script names to `opsOnlyScripts`, and restore the two observability placeholder i18n keys into the
  `v12locales` sources.
- Re-run every Docs Governance, harness and business gate on the applied tree.

### Out

- Any change to the `pantheon-base` repository, an existing tag, release or asset.
- Hand-copying base files into the Ops source tree instead of materializing the locked release.
- Any behaviour change in the k8s business module: menu membership, sort order, admin permission
  binding and route table are unchanged.
- Downgrading the Dependabot dependency bumps (i18next 26.5.11 / react-i18next 17.1.9).
- Re-enabling `check:task-packet` in CI; the legacy allowlist remains a separate task.

## Expected Files

### Create

- `docs/harness/tasks/2026-10-31-foundation-upgrade-base-v0-18-2.task.md`
- `.harness/tasks/2026-10-31-foundation-upgrade-base-v0-18-2/manifest.json`
- `.harness/evidence/2026-10-31-foundation-upgrade-base-v0-18-2/commands.json`
- `.harness/evidence/2026-10-31-foundation-upgrade-base-v0-18-2/summary.md`
- `.harness/evidence/2026-10-31-foundation-upgrade-base-v0-18-2/review.md`
- `.harness/evidence/2026-10-31-foundation-upgrade-base-v0-18-2/pr-body.md`
- `docs/release-notes/RELEASE_NOTES_v0.18.2.md`
- `tests/scripts/check-menu-contract.test.mjs`

### Modify

- `foundation-release.lock.json`
- `.foundation/foundation-release.lock.json`
- `VERSION`
- `business-overlay.json`
- `.business-overlay-report.json`
- `scripts/business-overlay/rebuild-from-base.mjs`
- `backend/modules/business/k8s/k8s.go`
- `backend/modules/business/k8s/k8s_seed.go`
- `frontend/scripts/check-menu-contract.mjs`
- `frontend/src/modules/business/k8s/index.ts`
- `frontend/src/modules/business/v12locales/zh-CN.json`
- `frontend/src/modules/business/v12locales/en-US.json`
- `frontend/src/i18n/resources/business/zh-CN.ts`
- `frontend/src/i18n/resources/business/en-US.ts`
- `package.json`
- `docs/PROJECT_INHERITANCE.md`
- `docs/release-notes/README.md`
- `docs/release-notes/README.en.md`
- `docs/reviews/CODE_SIMPLIFICATION_PLAN.md`
- `tests/scripts/check-pr-governance.test.mjs`
- `tests/scripts/quality-workflow.test.mjs`
- `tests/scripts/release-gate-workflow.test.mjs`

### Do Not Touch

- `.foundation/allowed-drift.json`
- `.foundation/releases/**` for any previously materialized release
- any existing tag, release or release asset
- `frontend/package.json`, `frontend/package-lock.json` (Dependabot-owned dependency bumps)
- `CHANGELOG.md`, `README.md`, `README.en.md` (Ops overlay copies)

## Implementation Notes

- The v0.18.2 delta is small: `sharedPaths` grows by one entry
  (`frontend/scripts/check-menu-contract.mjs`), so the manifest is backend 4 / frontend 47 / docs 2 =
  53, and the lock's `sharedPaths` lists were verified byte-equal against the manifest inside the tgz.
- The live replacement used an enumerated 14-path copy rather than a bulk archive overlay, because a
  wide-area overwrite was refused by the execution guard and because two paths must not be
  overwritten: the live `frontend/package.json` and `frontend/package-lock.json` already carry the
  Dependabot bumps (i18next 26.5.11, react-i18next 17.1.9) while base still pins 26.5.6 / 17.1.3.
  The pre-swap tree was snapshotted to `.tmp/backup-pre-0182/` first (the untracked set was empty).
- The new menu-contract sensor parses Go seed struct literals. The k8s module built its permission
  menu seeds inside `init()` with
  `titleKey := strings.ReplaceAll(item.perm, ":", ".") + ".permission"`, which is neither a
  resolvable constant nor a key that exists in any catalogue (runtime spelling:
  `business.k8s.cluster.sync.permission`). It failed the build with
  `后端 seed 引用了无法解析的常量: Key: titleKey` twice. The fix removes the derivation and unrolls
  20 static `F` menu seeds plus 40 zh-CN/en-US permission i18n rows on canonical constants.
- Adding canonical permission keys surfaced 16 permissions the frontend module never declared
  (cluster sync/nodes; workload scale/restart/logs; namespace view/create/delete; configmap
  view/create/update/delete; secret view/create/update/delete), so `frontend/src/modules/business/k8s/index.ts`
  gained those strings and the `v12locales` JSON sources gained their titles.
- `scripts/check-gorm-sql-literal.mjs` and its test predate the overlay declaration and were not in
  `opsOnlyScripts`; a clean rebuild would delete both files and drop `check:gorm-sql-literal` /
  `test:gorm-sql-literal` from `package.json`. Both paths are now declared and both script names
  survive the merge, so a follow-up rebuild re-derived 789 files with no deletion.
- `business.observability.monitoring.field.endpointPlaceholder` and `...urlPlaceholder` were present
  only in the generated `frontend/src/i18n/resources/business/*.ts`, so any rebuild would silently
  drop them; both are now in the `v12locales` sources that generate those files.

## Verification Plan

- `sha256sum` both materialized assets against the lock and the published `.sha256` sidecars
- `gh release view pantheon-base-v0.18.2` plus `git ls-remote` tag-object/peeled resolution
- `node scripts/business-overlay/rebuild-from-base.mjs` (lock-resolved)
- `node scripts/business-overlay/check-business-overlay.mjs`
- `node scripts/business-overlay/check-shared-drift.mjs --base ../pantheon-base/.tmp/rel-0182`
- `node --test tests/scripts/business-overlay/*.test.mjs`
- `go build ./... && go vet ./... && gofmt -l .` and `go test -count=1 ./...` in `backend`
- `npm run lint && npm run type-check && npm run build` in `frontend` (prebuild runs the sensor)
- Docs Governance set: `check:docs-frontmatter`, `check:encoding`, `check:structure`,
  `check:task-packet-template`, `check:pr-governance`, `check:generated-modules`,
  `check:ui-quality-gate`
- `check:task-packet`, `check:harness-adoption`, `check:failure-registry`,
  `check:business-boundaries`, `test:business-boundaries`, `check:business-foundation`,
  `check:duplication`, `check:harness-docs`, `check:gorm-sql-literal`
- hosted `Quality Gates`, `Security Gates`, `Inheritance Drift Detection` and `Branch Hygiene` on the
  consumption PR

## Linkage

- Task ID: `2026-10-31-foundation-upgrade-base-v0-18-2`
- Task Manifest: `.harness/tasks/2026-10-31-foundation-upgrade-base-v0-18-2/manifest.json`
- OpenSpec Change: `none`
- Superpowers Plan: `none`
- Plan References: `docs/harness/tasks/2026-10-31-foundation-upgrade-base-v0-18-2.task.md`
- Evidence Directory: `.harness/evidence/2026-10-31-foundation-upgrade-base-v0-18-2/`
- Review File: `.harness/evidence/2026-10-31-foundation-upgrade-base-v0-18-2/review.md`

## Evidence Required

- the published release identity: tag object, target commit, publishedAt, asset count, and the two
  digests re-hashed locally after materialization
- `sharedPaths` equivalence between the Ops lock and the v0.18.2 release manifest read out of the tgz
- overlay contract, shared-drift and business-overlay test results on the applied tree
- backend build/vet/gofmt/test and frontend lint/type-check/build results, including the sensor
  payload
- the sensor exposure and the k8s repair with the failing and the passing sensor readings
- the two rebuild-completeness repairs
- review summary

## Human Gates

- Stop if either SHA-256 digest does not match the lock, or if `repoSnapshot.baseCommit` differs from
  `lock.baseCommit`.
- Stop if a business overlay anchor stops resolving exactly once; that is a base assembly contract
  change and must be fixed in base first.
- The live-tree replacement is the L2 gate. It ran only because the maintainer explicitly asked for
  the consumer to move to the latest base release; the pre-swap diff was saved first.
- Any behaviour change in the k8s module beyond static seed spelling is out of scope; the seeded
  permission set must stay exactly the 237 keys the sensor counts.

## Completion Checklist

- [x] Layer and boundary declared
- [x] Contract anchors read
- [x] Verification run or exception recorded
- [x] Evidence saved or summarized
- [x] Review completed
