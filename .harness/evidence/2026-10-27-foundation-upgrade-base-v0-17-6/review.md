# Findings-first Review — Ops Consumption of pantheon-base-v0.17.6

## Findings

### P1 - The consumed release changes the ops schema baseline and that is not runtime-verified

`v0.17.6` moves the migration baseline to `000030` and seeds a new tenant-operations menu group with 5 permission codes. Every static gate is green, but startup plus `golang-migrate` on a real database was not executed in this round, so the schema consequence is asserted only from the release contents, not from a run.

Evidence: `.harness/evidence/2026-10-27-foundation-upgrade-base-v0-17-6/commands.json` (`runtime smoke` is recorded as `not-run`).

### P1 - Ops still has no consumer-side versioned migration channel

Four business modules rely on `AutoMigrate` instead of the numbered chain: `backend/modules/business/cicd/cicd.go:18`, `backend/modules/business/opsinit/opsinit.go:58` and `:70`, `backend/modules/business/k8s/backup/backup.go:21`, `backend/modules/business/k8s/rbac/rbac.go:86`. The swap inherits a Base schema that assumes versioned migrations, so ops-owned tables can drift from the chain without any gate noticing.

Evidence: `.harness/evidence/2026-10-27-foundation-upgrade-base-v0-17-6/summary.md` Known gaps.

### P2 - Five case-only renames needed an out-of-band removal step

Base renamed `AppModalActions.ts`, `RailSummary.ts`, `TableAction.ts`, `TableColumnPriority.ts` and `TableColumnWidth.ts` to lower camel case. On a case-insensitive filesystem the rebuild cannot express that as a rename; the old path had to be removed from the index and disk before staging, otherwise Git keeps both spellings and the `import path case guard` cannot tell the difference. The rebuild script itself is unchanged by this, so the next operator hits the same trap.

Evidence: `git status --porcelain` on the swapped tree reports 5 `R` entries; `check:import-path-case` passed for 348 source files.

### P2 - Historical evidence packages do not satisfy the current strict schema

Several `.harness/evidence/**/commands.json` packages use the older array root and fail `check-evidence --strict`. This predates the swap and was not rewritten here; it is governance debt, not a product defect, and it is not wired into ops CI (only `check:task-packet-template` runs in `quality.yml`).

Evidence: `.harness/evidence/2026-10-12-foundation-upgrade-base-v0-16-1/commands.json` fails with `Evidence root must be a JSON object.` while the package in this directory passes.

## Positive Signals

- Consumption went through the lock, not a local Base tree, so the release snapshot is the only source of truth; both asset digests were re-hashed locally and matched the published sidecars.
- Shared drift is zero-risk at the contract level: 805 shared files identical to Base, 10 allowlisted overlay hooks, 160 business-overlay files skipped, and that result holds both on the staged rebuild and on the applied working tree.
- The three consumer-side defects the swap exposed (31 broken `list-page.css` imports, 6 non-conforming `Input.Password` sites, missing `check:shared-drift` script) were fixed at the source, not exempted. The `check:shared-drift` entry was also added to `mergePackages` so a future rebuild cannot lose it again.
- The ops-only archived DDL is now an explicit `repositoryOverlayPaths` entry rather than something preserved by the accident that rebuild does not delete.
- Backend build/vet/gofmt/test, the CI rebuilt-tree test form, frontend type-check, build, lint and all 14 prebuild contract checks, plus structure/encoding/frontmatter/duplication governance checks, are green.

## Decision

Accept the baseline swap to `pantheon-base-v0.17.6` and keep this as the consumption round's verification record. Before shipping ops on this baseline, run the startup-plus-migrate acceptance against `000030` and open a bounded task for the consumer-side versioned migration channel. Neither blocks merging the consumption itself, because both are additive gaps the swap surfaced rather than regressions it introduced.

## Unknowns

- Whether the tenant-operations menu seed interacts with ops' own business menu seeds at runtime; static contract checks cannot answer this.
- Which owner decides the ops-side migration channel: a Base-provided consumer hook, or an ops-local runner that is allowlisted as overlay.

## Machine Readable

```json
{
  "taskId": "2026-10-27-foundation-upgrade-base-v0-17-6",
  "status": "reviewed",
  "verdict": "approved with documented P2 follow-up",
  "findings": [
    {"id": "C1", "severity": "P1", "title": "v0.17.6 schema baseline 000030 not runtime-verified", "state": "open-next-acceptance-round"},
    {"id": "C2", "severity": "P1", "title": "No consumer-side versioned migration channel; 4 AutoMigrate call sites", "state": "open-bounded-task"},
    {"id": "C3", "severity": "P2", "title": "Case-only renames need manual index removal on Windows", "state": "documented-workaround"},
    {"id": "C4", "severity": "P2", "title": "Historical commands.json packages fail the strict object schema", "state": "pre-existing-governance-debt"}
  ],
  "structuralReview": {
    "affectedSubgraph": [
      "backend/modules/business",
      "backend/modules/system/iam/menu",
      "frontend/src/modules/business",
      "frontend/src/components/patterns"
    ],
    "checks": ["cycle", "hub", "sensitive-flow"],
    "findings": [],
    "notes": "business module boundary check passed and shared drift reports only allowlisted overlay hooks; no new cross-layer edge was introduced by the swap."
  },
  "residualRisk": [
    "Runtime startup and golang-migrate to 000030 are unverified in this round.",
    "Ops-owned tables created through AutoMigrate can drift from the inherited versioned chain.",
    "Base main is ahead of the v0.17.6 tag, so against-main drift will differ from release-snapshot drift until the next release."
  ],
  "linkage": {
    "taskManifest": ".harness/tasks/2026-10-27-foundation-upgrade-base-v0-17-6/manifest.json",
    "taskPacket": "docs/harness/tasks/2026-10-27-foundation-upgrade-base-v0-17-6.task.md",
    "evidence": ".harness/evidence/2026-10-27-foundation-upgrade-base-v0-17-6/commands.json",
    "reviewFile": ".harness/evidence/2026-10-27-foundation-upgrade-base-v0-17-6/review.md",
    "changeRef": "none",
    "planRefs": [
      "docs/harness/tasks/2026-10-27-foundation-upgrade-base-v0-17-6.task.md",
      "docs/PROJECT_INHERITANCE.md"
    ]
  }
}
```
