# Findings-first Review — Ops Consumption of pantheon-base-v0.17.7

## Findings

### P1 - The claim that inline `// codeql` annotations suppress alerts is only half true, and the consumer paid for the difference

Base presents the v0.17.7 `// codeql <rule>: this is a false positive` comments in `backend/pkg/logging/logger.go` and `frontend/scripts/tenant-matrix-fixture-setup.mjs` as the CodeQL fix. On this repository they are not. Alerts 56-64 were created *after* the annotated commits landed and stayed open until each one was dismissed through `PATCH /repos/{owner}/{repo}/code-scanning/alerts/{number}`. The gate that `Security Gates` and the PR feedback prereq read is GitHub's alert state, and that state has no code-side input. An annotation-only release therefore leaves the consumer red and teaches the next operator to expect a suppression that does not happen.

This is a method defect owned by Base, so the repair is base-first: the release procedure must either publish the alert numbers it expects to be closed and dismiss them through the API as part of the release, or stop describing annotations as a suppression. Nothing in this round changes base code, because the consumed code is correct — only the claim about it was.

Evidence: `gh api 'repos/duanxldragon/pantheon-ops/code-scanning/alerts?state=open&ref=refs/pull/149/merge'` returned 9 alerts (56-64) before the API dismissals and 0 after; the unfiltered repo listing showed 0 throughout, which is why the alerts were initially invisible. The dismissals are what auto-resolved all nine PR review threads and made `fetch-github-feedback.mjs --check --repo duanxldragon/pantheon-ops --pr 149` exit 0.

### P1 - Inherited runtime acceptance gap

No backend startup, `golang-migrate` run or tenant-operations seeding was executed in this round, so nothing here proves the tree boots. v0.17.7 moves no migration, no seed and no permission, so the gap is inherited from the v0.17.6 round unchanged rather than widened — but it is still the one unverified claim in this consumption, and it is the gate that must close before ops ships on this baseline. No database credentials were recorded anywhere in this evidence set.

Evidence: `.harness/evidence/2026-10-27-foundation-upgrade-base-v0-17-7/commands.json` records the runtime smoke as `not-run`; schema baseline remains `000030` in `backend/pkg/database/migrations`.

### P1 - Still no consumer-side versioned migration channel, and the previously recorded scope was wrong

The last review round carried forward "four `AutoMigrate` call sites" without re-measuring. Re-measured against this tree, production code in `backend/modules/business` has 22 ungated `AutoMigrate` calls across 22 files — for example `deploy/deploy_service.go:188`, `filedist/storage_service.go:46`, `k8s/backup/backup_service.go:47` — and none of them consults `database.ShouldAutoMigrate()`, unlike the system-domain paths that do. Ops-owned tables are therefore created and evolved outside the inherited numbered chain, with no gate that can notice drift. The correction of the count is itself the signal: this gap has no measurement loop, which is exactly why it needs a bounded task rather than another mention.

Evidence: `grep -rn "AutoMigrate(" backend/modules/business --include=*.go | grep -v _test.go | wc -l` → 22; `grep -rn "ShouldAutoMigrate" backend/modules/business --include=*.go | grep -v _test.go` → empty.

### P2 - Hosted `SonarCloud Scan` fails on a stale credential, not on code

The job dies with `GET https://api.sonarcloud.io/analysis/jres ... 403 Forbidden` while both `SONAR_TOKEN` and `SONAR_HOST_URL` exist as repository secrets, so the secret is present and no longer valid. It is not a required context and no gate outcome here depends on it, but it does mean the "SonarCloud Open issues 0" claim cannot be re-established for this commit from CI. Rotating the token is a maintainer action; nothing code-side was changed to route around it.

Evidence: `gh pr view 149 --json statusCheckRollup` reports `SonarCloud Scan` = FAILURE while `Security Gates` and `CodeQL Alert Gate` are SUCCESS.

### P2 - One ops-owned lint defect reached CI instead of the local gate

`Go Lint` was red on two findings: one base-owned revive stutter, cleared by the consumed `//nolint:revive`, and one ops-owned `goconst` on the literal `"nginx"` in the deploy package. The ops half was fixed at the source — `defaultNginxServiceName` declared once in `deploy_model.go` and referenced from `deploy_helpers.go:186`, `deploy_rollback.go:81`, `deploy_rollback.go:126` and `deploy_service.go:3957`; only test fixtures still spell the literal, which `goconst` does not count. Worth naming plainly: the value was reachable by running the CI lint scope locally before pushing, and that is the ratchet here — `golangci-lint` with `--new-from-rev` against the merge base belongs in the pre-push set for backend rounds, not in the fix-after-CI loop.

Evidence: local `golangci-lint run --new-from-rev=edcca7eb... ./...` in `backend` → `0 issues.`

## Positive Signals

- Consumption went through the published lock and assets, never a local Base tree and never `--base`; both digests re-hashed locally match the lock, the `.sha256` sidecars and the GitHub asset digests, and the `manifest.json` inside the tgz agrees with the lock on `releaseVersion`, `baseCommit` and all three `sharedPaths` groups.
- The delta is exactly the lint/annotation surface: 6 tracked paths moved and 1 was added, `go mod tidy` returned `go.mod`/`go.sum` byte-identical to the v0.17.6 round, and nothing touched an API, DTO, migration or permission. That is the proof the contract works — an incremental release produces an incremental, explainable switch.
- Overlay ownership did real work: base's v0.17.7 updates to `CHANGELOG.md`, `README.md` and `README.en.md` deliberately did not land, because those are declared Ops overlay paths in `business-overlay.json`.
- Shared drift stays at contract level: 805 shared files identical to a Base checkout that is on the consumed commit, 10 allowlisted overlay hooks, 160 business-overlay files skipped, 53 declared paths valid, 13/13 business-overlay tests green, 751 declared/generated files rebuilt.
- Every dismissal in this round was decided by re-reading the code the alert points at, not by bulk action — 56 on a sanitizer that maps CR/LF/U+2028/U+2029 to a space and drops other control characters except TAB, 57-63 on a gitignored fixture that never ships to a browser, 64 accepted as a mitigation because `normalizeLocalStoragePath` rejects parent-directory segments before `filepath.Clean` and object paths are rebuilt through `filepath.Base`.

## Decision

Accept the baseline advance to `pantheon-base-v0.17.7` and merge the consumption once the required `Quality Gates` and `Security Gates` contexts are green on the pushed commit. The P1s do not block this merge: the CodeQL and lint items are closed by this round, the runtime gap and the migration channel are pre-existing additive gaps that need their own bounded rounds, and the credential item is outside agent authority. They do block shipping ops on this baseline.

## Unknowns

- Whether GitHub's CodeQL analysis ever consumed those inline annotations at all on this host, or whether the annotations only affect the SARIF result metadata; the distinction matters for what base should change and cannot be settled from the alert listing alone.
- Which owner takes the ops-side migration channel: a Base-provided consumer hook, or an ops-local runner allowlisted as overlay.
- Whether the stale `SONAR_TOKEN` is repo-scoped or organization-scoped, which decides whether one rotation fixes every consumer repository.

## Machine Readable

```json
{
  "taskId": "2026-10-27-foundation-upgrade-base-v0-17-7",
  "status": "reviewed",
  "verdict": "approved with documented P2 follow-up",
  "findings": [
    {"id": "D1", "severity": "P1", "title": "Inline codeql annotations do not change GitHub alert state; base-first release-procedure fix", "state": "closed-for-this-round-open-on-base"},
    {"id": "D2", "severity": "P1", "title": "No runtime acceptance (startup + golang-migrate 000030) inherited from v0.17.6", "state": "open-next-acceptance-round"},
    {"id": "D3", "severity": "P1", "title": "No consumer-side versioned migration channel; 22 ungated AutoMigrate call sites in business", "state": "open-bounded-task"},
    {"id": "D4", "severity": "P2", "title": "Hosted SonarCloud Scan 403 on stale SONAR_TOKEN repository secret", "state": "maintainer-action"},
    {"id": "D5", "severity": "P2", "title": "Ops-owned goconst defect reached CI; add CI-scoped golangci-lint to pre-push set", "state": "fixed-at-source-ratchet-noted"}
  ],
  "structuralReview": {
    "affectedSubgraph": [
      "backend/pkg/logging",
      "backend/modules/business/deploy",
      "backend/modules/business/filedist",
      "backend/modules/system/config/setting",
      "frontend/scripts"
    ],
    "checks": ["cycle", "hub", "sensitive-flow"],
    "findings": [
      "D3: business modules create their schema through AutoMigrate outside the inherited versioned chain",
      "D1: alert-state dismissal is a release-procedure step, not a code step, so the security-sensitive logging path has no in-repo gate"
    ],
    "notes": "No new cross-layer edge was introduced: the consumed delta is comment-level in system and logging code, and the ops repair is a package-internal constant extraction in business/deploy. Boundary and structure checks pass on the applied tree."
  },
  "residualRisk": [
    "Runtime startup and golang-migrate to 000030 remain unverified in this round.",
    "Ops-owned tables evolved through AutoMigrate can drift from the inherited numbered chain with no gate noticing.",
    "SonarCloud quality-gate status for this commit cannot be asserted from CI while the token returns 403.",
    "Base main is ahead of the v0.17.7 tag, so against-main drift will differ from release-snapshot drift until the next release."
  ],
  "linkage": {
    "taskManifest": ".harness/tasks/2026-10-27-foundation-upgrade-base-v0-17-7/manifest.json",
    "taskPacket": "docs/harness/tasks/2026-10-27-foundation-upgrade-base-v0-17-7.task.md",
    "evidence": ".harness/evidence/2026-10-27-foundation-upgrade-base-v0-17-7/commands.json",
    "reviewFile": ".harness/evidence/2026-10-27-foundation-upgrade-base-v0-17-7/review.md",
    "changeRef": "none",
    "planRefs": [
      "docs/harness/tasks/2026-10-27-foundation-upgrade-base-v0-17-7.task.md",
      "docs/PROJECT_INHERITANCE.md"
    ]
  }
}
```
