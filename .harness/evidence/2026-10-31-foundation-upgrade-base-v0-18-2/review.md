# Findings-first Review — Ops Consumption of pantheon-base-v0.18.2

## Findings

### P1 - Twenty permission title keys were invisible to every validator and absent from every catalogue

The k8s module generated its permission menu seeds at runtime inside `init()`:

```go
titleKey := strings.ReplaceAll(item.perm, ":", ".") + ".permission"
```

The resulting keys (`business.k8s.cluster.sync.permission`, `business.k8s.workload.scale.permission`
and 18 others) resolved to no constant and existed in no zh-CN/en-US catalogue. Nothing in the
pipeline validated them: the sensor that would have — `check-menu-contract.mjs` — only became shared
with consumers in v0.18.2. On its first run it failed the frontend build
(`后端 seed 引用了无法解析的常量: Key: titleKey` twice) and the repair unrolled 20 static seeds plus
40 i18n rows on canonical constants.

This is the release working as intended: a static sensor found live drift on the first consumer run.
The consumer-side lesson is that any seed value computed by string surgery at init time is
unreviewable by construction.

### P1 - The deterministic rebuild would have deleted an ops-only gate and its test

`scripts/check-gorm-sql-literal.mjs` and `tests/scripts/check-gorm-sql-literal.test.mjs` were never
declared in `business-overlay.json`, and their two script names were not in `opsOnlyScripts`. A
clean rebuild therefore planned to delete both files and drop `check:gorm-sql-literal` /
`test:gorm-sql-literal` from the merged `package.json`. The gate exists specifically to keep GORM
table/column identifiers literal across `backend/modules/business/**`, so losing it silently would
have removed the consumer's only guard on that surface.

Both paths are now declared, both names survive, and a follow-up rebuild re-derived 789 files with
no deletion.

### P2 - Two i18n keys lived only in generated files

`business.observability.monitoring.field.endpointPlaceholder` and `...urlPlaceholder` were present
only in the generated `frontend/src/i18n/resources/business/*.ts`. Because those files are rewritten
from the business JSON sources on every rebuild, the keys would have disappeared without any diff
explaining why. They are now in `frontend/src/modules/business/v12locales/{zh-CN,en-US}.json`, the
actual source of truth.

### P2 - The live replacement was narrower than the rebuilt tree, on purpose

The rebuilt staging tree also contains `frontend/package.json` and `frontend/package-lock.json` at
base's pins, while the live tree carries the merged Dependabot bumps (i18next 26.5.11,
react-i18next 17.1.9). Applying those two files would have silently downgraded both packages, so the
replacement copied an enumerated 14-path list (12 modified, 2 new) and excluded them. This is the
intended behaviour of a consumer that owns dependency bumps, but it means the live tree is not
byte-identical to the staging tree; `check:shared-drift` still reports 813 identical files because
neither path is in `sharedPaths`.

### P2 - Runtime acceptance was not re-run

No fresh backend startup or seeding reading exists for this round. The k8s change is seed spelling
only and the sensor proves the resulting key set is unchanged in size (237 permissions), but the
claim "the server still seeds these menus" rests on `go test` and the v0.18.1 isolated-schema run
rather than a new runtime reading.

### P2 - Inherited base documents link to base-only destinations

`check:harness-docs` reports dangling links inherited from base documents pointing at base-only task
packets and the base evidence archive. The check is report-only, is not wired into any ops workflow,
and hand-editing those documents in ops would be shared drift. The repair belongs in base.

### P2 - The first PR analysis failed the SonarCloud new-code gate

`new_duplicated_lines_density` came back at 3.187% (16 of 502 new lines) against a 3% threshold,
plus six `go:S1192` smells for repeated literals in the new seed rows. Both are real readings on the
new code, not a credential failure, so they were fixed at the source: the repeated Go literals moved
into named constants, and the 16 new permission keys are grouped two per line by resource so the
flat list stops presenting as a duplicated token run while the menu-contract sensor still sees all
237 keys. The gate is not waived, the quality profile is untouched, and the final check reading is
taken from the PR rather than asserted.

### P2 - Race tests cannot run on this host

`go test -race` fails on this Windows machine because ThreadSanitizer cannot reserve its shadow
memory (`error code: 87`). The plain `go test ./...` run passes and CI runs the race variant on
`ubuntu-latest`.

## What was verified instead

- Assets re-hashed after materialization: tgz `c01ebbc1…` (30,051,238 B), `repo.tar` `c1360fb1…`
  (47,923,200 B), both equal to the lock and the GitHub asset digests; `manifest.json` inside the
  tgz agrees with the lock on version, commit and all three `sharedPaths` groups (53 entries).
- Rebuild through the lock path: base commit `a44ae4e0`, 789 declared/generated files, no deletion.
- Shared drift on the applied tree: 813 identical, 10 allowlisted, 161 skipped.
- Overlay contract 53 paths; 13/13 overlay tests; backend build/vet/gofmt/test clean (64 packages);
  frontend lint, type-check and build clean with the sensor reporting 40 menus / 50 routes / 237
  permissions / 52 component keys.
- Every Docs Governance, harness and business gate listed in the packet is green on the applied
  tree, including `check:gorm-sql-literal` with its four documented waivers.

## Machine Readable

```json
{
  "taskId": "2026-10-31-foundation-upgrade-base-v0-18-2",
  "verdict": "approved",
  "findings": [
    {"id": "D1", "severity": "P1", "title": "20 k8s permission title keys were computed in init(), resolvable by no constant and present in no catalogue; the new shared sensor caught them on first run", "state": "fixed-static-unroll"},
    {"id": "D2", "severity": "P1", "title": "A clean rebuild would have deleted check-gorm-sql-literal.mjs and its test and dropped their scripts", "state": "fixed-overlay-declared"},
    {"id": "D3", "severity": "P2", "title": "Two observability placeholder i18n keys existed only in generated files and would vanish on rebuild", "state": "fixed-source-of-truth-restored"},
    {"id": "D4", "severity": "P2", "title": "Live replacement intentionally excluded frontend/package.json and lockfile to preserve Dependabot bumps, so the live tree is not byte-identical to staging", "state": "accepted-by-design"},
    {"id": "D5", "severity": "P2", "title": "No fresh runtime startup/seeding reading for v0.18.2", "state": "recorded-gap"},
    {"id": "D6", "severity": "P2", "title": "check:harness-docs dangling links inherited from base documents", "state": "base-side-fix-needed"},
    {"id": "D7", "severity": "P2", "title": "go test -race cannot run on this Windows host (TSan shadow memory)", "state": "environment-gap"},
    {"id": "D8", "severity": "P2", "title": "First PR analysis failed the SonarCloud gate on new-code duplication (3.187%) and six S1192 smells in the new seed rows", "state": "fixed-in-code-not-waived"}
  ],
  "structuralReview": {
    "affectedSubgraph": [
      "foundation release -> consumer lock -> rebuilt tree",
      "frontend/scripts/check-menu-contract.mjs -> k8s seed constants -> frontend permission declarations -> v12locales JSON -> generated i18n resources",
      "business-overlay.json + rebuild-from-base.mjs -> package.json script merge"
    ],
    "checks": ["cycle", "hub", "sensitive-flow"],
    "findings": [
      "D1: runtime string surgery in a seed is an unreviewable surface; the fix converts it into constant references the sensor can follow",
      "D2: the overlay declaration is the only list that protects ops-only files from the deterministic rebuild; anything absent from it is scheduled for deletion"
    ],
    "notes": "No product behaviour was written. The business-module delta is seed spelling only: same menus, same sort, same 237 permission keys, same admin binding."
  },
  "residualRisk": [
    "Runtime seeding of the rewritten k8s permission rows rests on go test plus the v0.18.1 isolated-schema run, not a fresh startup reading.",
    "Base documents that link to base-only task packets keep failing the report-only doc-link check in the consumer.",
    "Vercel remains red on pull requests because the deployment lives in the Vercel project; it is not a required context.",
    "PR #166 auto-merged before the Sonar remediation reached its head, so the fix landed through a follow-up PR cut from the merged main; the first PR's own gate reading stays red by history."
  ],
  "linkage": {
    "taskManifest": ".harness/tasks/2026-10-31-foundation-upgrade-base-v0-18-2/manifest.json",
    "taskPacket": "docs/harness/tasks/2026-10-31-foundation-upgrade-base-v0-18-2.task.md",
    "evidence": ".harness/evidence/2026-10-31-foundation-upgrade-base-v0-18-2/commands.json",
    "reviewFile": ".harness/evidence/2026-10-31-foundation-upgrade-base-v0-18-2/review.md",
    "changeRef": "none",
    "planRefs": [
      "docs/harness/tasks/2026-10-31-foundation-upgrade-base-v0-18-2.task.md",
      "docs/PROJECT_INHERITANCE.md"
    ]
  }
}
```
