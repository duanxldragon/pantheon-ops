---
title: Consume pantheon-base-v0.17.7 into pantheon-ops
doc_type: Remediation
layer: inheritance-sync
status: Active
linked_contracts:
  - docs/PROJECT_INHERITANCE.md
updated_at: 2026-10-27
---

# Task Packet: 2026-10-27-foundation-upgrade-base-v0-17-7

Target repository: `pantheon-ops`
Layer: `inheritance-sync (pantheon-base -> pantheon-ops)`
Delivery tier: `L2`
Mode: immutable release consumption, incremental baseline advance, consumer gate repair

## Goal

Advance the Ops consumer from `pantheon-base-v0.17.6` to the published `pantheon-base-v0.17.7`
(`2091a2d8409e6b4ed933b19c8ff8870f2d0eedbe`) so the tree carries the base-side lint and CodeQL
dispositions instead of re-suppressing them locally, and close the two PR #149 gate reds that the
consumption itself is responsible for: the `revive` stutter on `setting.SettingService` and the nine
open CodeQL alerts. One remaining ops-owned lint finding (`goconst` on the deploy package's nginx
service name) is repaired at its source in the same round.

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

- `pantheon-base` foundation release pipeline (producer side: tag, assets, Release Gate, Foundation Release Verify)
- Ops business overlay assembly (`scripts/business-overlay/rebuild-from-base.mjs`)
- Ops shared-drift gate (`scripts/business-overlay/check-shared-drift.mjs`)
- Ops Go lint baseline (`golangci-lint --new-from-rev=edcca7eb` in the hosted `Go Lint` job)
- GitHub code scanning state and the PR review feedback gate

## Harness Profile

- Template: api-service
- Overlay: pantheon-ops
- Quality Profile: foundation-release-consumer
- Portable Failure Class: consumer-lock-drift
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
- `scripts/fetch-github-feedback.mjs`

## Scope

### In

- Update `foundation-release.lock.json` to `pantheon-base-v0.17.7`: version, base commit, artifact
  name, materialization path, bundle checksum and `repo.tar` checksum.
- Rewrite `.foundation/foundation-release.lock.json` with the release identity and the gate facts
  read back from GitHub (tag object, publishedAt, four assets, Release Gate run, Foundation Release
  Verify run, sharedPaths equivalence against the release manifest).
- Rebuild the Ops tree from the locked snapshot, apply it, and re-tidy the merged Go module graph.
- Repair the ops-owned `goconst` finding in `backend/modules/business/deploy` by naming the repeated
  nginx service string once.
- Record the GitHub-side disposition for the nine open CodeQL alerts on the PR ref and confirm the
  PR feedback gate no longer reports blocked items.

### Out

- Any change to the `pantheon-base` repository, to an existing tag, release or asset.
- Hand-copying base files into the Ops source tree; the consumer materializes from the release only.
- Any behaviour change in the business modules beyond replacing a repeated string literal with a
  named constant.
- Rewriting `CHANGELOG.md` or `README*.md` from base; those are Ops overlay paths by contract.
- Merging or closing the open Dependabot pull requests, and deleting remote branches (maintainer).

## Expected Files

### Create

- `docs/harness/tasks/2026-10-27-foundation-upgrade-base-v0-17-7.task.md`
- `.harness/tasks/2026-10-27-foundation-upgrade-base-v0-17-7/manifest.json`
- `.harness/evidence/2026-10-27-foundation-upgrade-base-v0-17-7/commands.json`
- `.harness/evidence/2026-10-27-foundation-upgrade-base-v0-17-7/summary.md`
- `.harness/evidence/2026-10-27-foundation-upgrade-base-v0-17-7/review.md`
- `RELEASE_NOTES_v0.17.7.md` (base-owned, arrives through the rebuild)

### Modify

- `foundation-release.lock.json`
- `.foundation/foundation-release.lock.json`
- `VERSION`
- `backend/pkg/logging/logger.go`
- `backend/modules/system/config/setting/setting_service.go`
- `frontend/scripts/tenant-matrix-fixture-setup.mjs`
- `.business-overlay-report.json`
- `backend/modules/business/deploy/deploy_model.go`, `deploy_helpers.go`, `deploy_rollback.go`,
  `deploy_service.go`

### Do Not Touch

- `.foundation/allowed-drift.json`
- `.foundation/releases/**` for any previously materialized release
- any existing tag, release or release asset
- `CHANGELOG.md`, `README.md`, `README.en.md` (Ops overlay copies)

## Implementation Notes

- `rebuild-from-base.mjs` resolves the base from the lock when `--base` is omitted, so this round
  consumed the published v0.17.7 archive and never the local Base working tree; the local checkout
  happens to sit on the same commit (`2091a2d`), which is what made the against-main drift read
  meaningful rather than coincidental.
- The consumer delta is three base-owned files plus `VERSION` and the release notes. `go mod tidy`
  reproduced `go.mod` and `go.sum` byte-identical to the v0.17.6 round, which is the proof that
  v0.17.7 moved no dependency; without the tidy the rebuild's `go mod edit -require` output reverts
  to Base's shorter indirect block and the tree does not build.
- Base's v0.17.7 release notes state that the four `go/log-injection` and seven
  `js/file-access-to-http` sites now carry inline `// codeql ... false positive` annotations. Those
  annotations document the conclusion for a human reader; they do not change GitHub's alert state.
  Alerts 56-64 stayed open after the annotated commits landed and had to be dismissed through the
  code scanning API, exactly as alerts 52 and 53 were in August.
- The `// codeql` comment lines shift line numbers, so a dismissal recorded in GitHub must be keyed
  to the alert rather than to the line; the dismissals here were applied to alert numbers, and the
  re-scan on the new commit reuses the same `primaryLocationLineHash`, so they do not reopen.
- `goconst` counted seven occurrences of `"nginx"`; four are production code (`deploy_helpers.go:186`
  as a component-type key, `deploy_rollback.go:81` and `:126` and `deploy_service.go:3957` as the
  systemd unit fallback). Naming it `defaultNginxServiceName` removes the literal from production
  code entirely; the remaining occurrences are test fixtures, which `goconst` does not flag.

## Verification Plan

- `sha256sum` both materialized assets against the lock and the published `.sha256` sidecars
- `node scripts/business-overlay/rebuild-from-base.mjs` (lock-resolved)
- `node scripts/business-overlay/check-business-overlay.mjs`
- `node scripts/business-overlay/check-shared-drift.mjs --base ../pantheon-base`
- `node --test tests/scripts/business-overlay/*.test.mjs`
- `go mod tidy && go build ./... && go vet ./... && gofmt -l .` and `go test -count=1 ./...` in `backend`
- `golangci-lint run --path-mode=abs --max-issues-per-linter 0 --max-same-issues 0 --new-from-rev=edcca7eb7bcbfd264e9c437e73ea44a7adfcd454 ./...`
- `node scripts/check-encoding.mjs --strict`, `node scripts/harness/check-structure-contract.mjs --root . --strict`,
  `node scripts/frontmatter-check.mjs`, `node scripts/check-duplication.mjs`
- `node scripts/fetch-github-feedback.mjs --check --repo duanxldragon/pantheon-ops --pr 149`
- code scanning alert counts before and after the dismissal
- hosted `Quality Gates`, `Security Gates`, `Go Lint`, `CodeQL` and `GitHub Feedback Prereq` on the PR

## Linkage

- Task ID: `2026-10-27-foundation-upgrade-base-v0-17-7`
- Task Manifest: `.harness/tasks/2026-10-27-foundation-upgrade-base-v0-17-7/manifest.json`
- OpenSpec Change: `none`
- Superpowers Plan: `none`
- Plan References: `docs/harness/tasks/2026-10-27-foundation-upgrade-base-v0-17-7.task.md`
- Evidence Directory: `.harness/evidence/2026-10-27-foundation-upgrade-base-v0-17-7/`
- Review File: `.harness/evidence/2026-10-27-foundation-upgrade-base-v0-17-7/review.md`

## Evidence Required

- the published release identity: tag object, target commit, publishedAt, asset count, and the two
  digests re-hashed locally after materialization
- `sharedPaths` equivalence between the Ops lock and the v0.17.7 release manifest read out of the tgz
- overlay contract, shared-drift and business-overlay test results on the applied tree
- backend build/vet/gofmt/test and the CI-scoped `golangci-lint` result, before and after the
  `goconst` repair
- the alert numbers dismissed, their reasons, and the open-alert count before and after
- the `GitHub Feedback Prereq` result re-run locally against PR #149

## Human Gates

- Stop if either SHA-256 digest does not match the lock, or if `repoSnapshot.baseCommit` differs from
  `lock.baseCommit`.
- Stop if a business overlay anchor stops resolving exactly once; that is a base assembly contract
  change and must be fixed in base first.
- Dismissing a code scanning alert is a gate disposition. Each of the nine here was checked against
  the code before dismissal and the reason is recorded in `review.md`; a future alert on the same
  rule must be re-checked, not bulk-dismissed.
- Rotating the SonarCloud token and deciding the fate of the open Dependabot pull requests and
  non-main branches stays with the maintainer.

## Completion Checklist

- [x] Layer and boundary declared
- [x] Contract anchors read
- [x] Verification run or exception recorded
- [x] Evidence saved or summarized
- [x] Review completed
