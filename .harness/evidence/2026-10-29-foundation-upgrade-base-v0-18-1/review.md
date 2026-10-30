# Findings-first Review — Ops Consumption of pantheon-base-v0.18.1

## Findings

### P1 - Base shipped a consumer-breaking export removal; the consumer sensor that would have caught it never ran

v0.18.0's simplification wave S1 removed `isForbiddenRequestError` from
`frontend/src/api/request.ts` because a search inside `pantheon-base` found no callers. The
`pantheon-ops` deploy detail page imports it (`frontend/src/modules/business/deploy/task/DeployTaskDetail.tsx:29`)
and branches on it, so the consumer frontend build died with `TS2614` on v0.18.0 and the whole
release was unusable for this repository. v0.18.1 restores the export, which is why this round
consumes a patch instead of the minor.

The base side already carries the lesson: `FR-021` in the failure registry plus
`AI_QUALITY_GOVERNANCE.md` section 5.4 require a workspace-wide consumer reference search before
deleting a shared export, and require the consumer build to pass before a foundation release counts
as verified. This review records the consumer half so the two sides stay linked: the release that
"passed" did not compile its only consumer.

### P1 - Two Ops overlay pins had turned base-owned checkers into crashers

`scripts/task-manifest.mjs` and `scripts/cleanup-github-branches.mjs` were pinned as Ops overlay
paths from the pre-archive era, and neither carried Ops-specific customisation. Base 0.18 made the
task-manifest module archive-aware and both `scripts/harness/check-task-packet.mjs` and
`check-doc-links.mjs` now import `resolveArchiveAwarePath` from it, so both checkers died with a
module SyntaxError instead of reporting anything. The same pattern hid base's null-safe branch
deletion: base's `deleteBranch` exports, tolerates `404 already-gone` and reports
`protected-or-rejected`, while the pinned copy threw on both.

Both pins were removed and base's copies adopted. A pinned overlay that only repeats an older copy
of a base file is not ownership, it is silent drift, and it fails in exactly the direction that
disables the check.

### P1 - The consumer pipeline never ran the checker that broke

`check:task-packet` is a blocking step in base's Docs Governance job and is absent from the Ops
job, so the crash above was invisible to CI: the pipeline stayed green while the checker could not
run at all.

The checker itself is fixed and verified locally: the new packet passes, and the two v0.17 packets
that were the only current-format docs pass with it. Wiring it into CI is deliberately deferred:
60 of the 66 Ops task docs predate the section template, base's legacy allowlist is empty, and
re-enabling the gate would require a 60-entry allowlist that then has to shrink every time an Ops
packet is archived. That is a task of its own, not a side effect of a release consumption.

### P1 - Dependabot PRs could never satisfy the governance prerequisite

`.github/workflows/pr-automation.yml` ran `check-pr-governance.mjs` on every PR with no Dependabot
exemption, while `scripts/check-pr-governance.mjs` has no Dependabot branch of its own. Every
Dependabot PR therefore failed `PR Governance Prereq` on a body it rewrites on each push, and the
auto-merge job could never be reached. Base solved this in its own workflow by exempting
`dependabot[bot]` from the body ceremony while keeping the structural checks in `quality.yml`; the
consumer now mirrors that. This is a precondition for closing the open Dependabot PRs at all.

One inherited assertion still cannot pass: base's `pr-automation-workflow.test.mjs` expects the
merge step to use `secrets.RELEASE_GATE_TOKEN`, and this repository has no such secret (only
`SONAR_TOKEN` and `SONAR_HOST_URL`). Creating a repository secret is a maintainer action, so the
workflow keeps `github.token` and the assertion stays red.

### P2 - Inherited runtime acceptance gap

No backend startup, `golang-migrate` run or seeding was executed. v0.18.x adds a tenant-console
smoke spec and tenant platform handler tests but moves no migration baseline, so the gap is
inherited unchanged from the v0.17.7 round rather than widened. It is still the one unverified
claim in this consumption.

### P2 - Inherited base documents link to base-only destinations

`check:harness-docs` reports four dangling links, all in base-owned documents:
`docs/reviews/TENANT_MULTI_MEMBERSHIP_DECISION_2026-10-27.md` and
`docs/reviews/TENANT_PLAN_LADDER_DECISION_2026-10-28.md` point at
`docs/harness/tasks/2026-10-27-tenant-completeness-remediation.task.md`, a path the consumer
deliberately excludes, and `docs/runbooks/TENANT_MIGRATION_RUNBOOK.md` points at evidence inside
base's archive. Hand-editing them in Ops would be shared drift, so the repair belongs in base. The
check is report-only and not wired into any Ops workflow.

### P2 - Race tests cannot run on this host

`go test -race` fails on this Windows machine because ThreadSanitizer cannot reserve its shadow
memory (`failed to allocate ... error code: 87`). Every package aborts in the runtime, not in the
product code, and the plain `go test ./...` run passes. CI runs the race variant on
`ubuntu-latest`, so this is recorded as an environment gap rather than skipped silently.

### P2 - Inherited base workflow tests assert a `.github` the consumer replaces

`release-gate-workflow.test.mjs` expects `.github/workflows/release-gate.yml`,
`quality-workflow.test.mjs` expects base's docs-only/governance-only scope classification and
report-only go-lint enforcement, and one `pr-automation-workflow.test.mjs` assertion expects
`RELEASE_GATE_TOKEN`. The Ops repository owns its `.github` by contract (`replaceRoots`), has no
release-gate workflow and no such secret, so these assertions cannot pass without either copying
base's release pipeline into a consumer or fabricating a credential. They are not part of any
workflow. Two related failures were real and are fixed: the stale setup-node pin in
`workflow-baseline.test.mjs` (now `949feb24`) and the missing `deleteBranch` export in the pinned
branch-cleanup script (now base's copy, 4/4 tests pass).

## What was verified instead

- Assets re-hashed after materialization: tgz `2fe49d70…`, `repo.tar` `37ae7a08…`, both equal to
  the lock and the GitHub asset digests; `manifest.json` inside the tgz agrees with the lock on
  version, commit and all three `sharedPaths` groups.
- Rebuild through the lock path twice, before and after the overlay-config repair, both landing on
  base commit `93505225`; second run reports 758 declared/generated files.
- Shared drift on the applied tree: 812 identical, 10 allowlisted, 160 skipped.
- Overlay contract 53 paths; 13/13 overlay tests; backend build/vet/gofmt/test clean; frontend lint,
  type-check and build clean after `npm ci` at the root and in `frontend`.
- Every Docs Governance command the workflow runs is green on the applied tree; `check:task-packet`
  is green for current-format packets and red only on the 60 pre-template docs described above.
- Hosted verification on the merged main `165014c8`: push runs for Code Quality Gates
  (`40839019495`), Security Gates (`40839019357`) and Branch Hygiene (`40839019345`) all succeeded,
  and a `workflow_dispatch` of the same commit proved Code Quality Gates (`40840649887`), Security
  Gates (`40840655454`), Branch Hygiene (`40840665983`) and Base Compatibility Rebuild
  (`40840676477`). Only SonarCloud Scan (`40839019352`) failed, on the credential described in D9.
  The earlier squash merge `fc6bc850` produced no push runs, so the dispatch was necessary.
- All seven Dependabot PRs were resolved: #150 merged, #129/#130/#131/#135/#136/#137 closed as
  obsolete because main already carries newer versions. Only `main` remains on the remote; no PR is
  open.

## Runtime acceptance review (2026-10-31)

- 边界复核：本轮新增的是 evidence 记录与 lock 文件 `pendingWork` 状态，业务源码零改动
  （`git status` 干净，一次性驱动文件已删除）。因此不存在新的消费者契约面。
- 判定：`runtime-acceptance` 的验收口径由「声称能跑」提升到「在隔离 schema 上真跑通」，
  并且覆盖了 base v0.18.x 最敏感的继承面——租户核心表 DDL（migration 20/24 修复线）与
  租户控制台 HTTP 契约。写路径还顺带验证了 secure-action 守卫与软删除的 code 释放约定。
- 保留意见：UI 冒烟（tenant-console.spec.ts + matrix fixture）未跑，不能据此声称浏览器面已验收；
  该项在 knownGaps 里显式留着，而不是被本轮的 API 读数掩盖。
- 会话卫生：一次 shell `source .env` 失败回显了本地开发库连接串（未入仓、未入提交）。已改为
  进程内解析，不再经 shell 展开；建议轮换该本地口令。

## Machine Readable

```json
{
  "taskId": "2026-10-29-foundation-upgrade-base-v0-18-1",
  "verdict": "approved",
  "findings": [
    {"id": "D1", "severity": "P1", "title": "v0.18.0 removed a shared export with a repo-local caller search; consumer build broke, v0.18.1 restores it", "state": "closed-by-v0181"},
    {"id": "D2", "severity": "P1", "title": "Two Ops overlay pins served stale copies of base scripts and turned check-task-packet/check-doc-links into crashes", "state": "fixed-unpinned"},
    {"id": "D3", "severity": "P1", "title": "Ops Docs Governance never ran check:task-packet, so the broken checker stayed invisible", "state": "checker-fixed-ci-wiring-deferred"},
    {"id": "D4", "severity": "P1", "title": "Dependabot PRs could never satisfy PR Governance Prereq; auto-merge unreachable", "state": "fixed-exemption-added"},
    {"id": "D5", "severity": "P2", "title": "No runtime acceptance (startup + golang-migrate) in this round", "state": "closed-2026-10-31"},
    {"id": "D6", "severity": "P2", "title": "check:harness-docs has four dangling links inherited from base documents", "state": "base-side-fix-needed"},
    {"id": "D7", "severity": "P2", "title": "go test -race cannot run on this Windows host (TSan shadow memory)", "state": "environment-gap"},
    {"id": "D8", "severity": "P2", "title": "Inherited base workflow tests assert base's .github shape and RELEASE_GATE_TOKEN", "state": "explicitly-accepted-red"},
    {"id": "D9", "severity": "P2", "title": "SonarCloud Scan is the only failing workflow: SONAR_TOKEN rejected with HTTP 403, and Dependabot PR runs have repository secrets withheld", "state": "maintainer-credential-action"},
    {"id": "D10", "severity": "P2", "title": "Vercel production and preview deployments fail from an external project configuration the repository cannot see", "state": "external-integration-gap"}
  ],
  "structuralReview": {
    "affectedSubgraph": [
      "foundation release -> consumer lock -> rebuilt tree",
      "scripts/task-manifest.mjs -> scripts/harness/check-task-packet.mjs, check-doc-links.mjs",
      "scripts/cleanup-github-branches.mjs -> branch-hygiene workflow",
      ".github/workflows/quality.yml -> Docs Governance",
      ".github/workflows/pr-automation.yml -> auto-merge"
    ],
    "checks": ["cycle", "hub", "sensitive-flow"],
    "findings": [
      "D2: an overlay pin that duplicates a base file is a drift channel; both pinned copies silently froze base behaviour at the consumer",
      "D4: the auto-merge chain had a structurally unsatisfiable prerequisite for the bot account that owns seven open PRs"
    ],
    "notes": "No product code was written. The consumed delta deletes base content the simplification waves retired and adds base tests; the Ops-owned edits are two overlay unpins, one workflow step, one workflow condition and one stale test pin."
  },
  "residualRisk": [
    "Runtime startup and golang-migrate remain unverified in this round.",
    "Base documents that link to base-only task packets and archive evidence will keep failing the consumer doc-link report until base changes them.",
    "Inherited base workflow tests stay red locally for release-gate, quality-workflow and the RELEASE_GATE_TOKEN assertion; none of them is executed by a workflow.",
    "SonarCloud Scan stays red until a maintainer rotates SONAR_TOKEN and handles Dependabot secret access; Vercel stays red until its project configuration is fixed in the Vercel dashboard.",
    "A workflow-triggered squash merge can land without push pipelines running (observed on fc6bc850), so the security/quality gates on the final tree were proven with workflow_dispatch rather than by a push event."
  ],
  "linkage": {
    "taskManifest": ".harness/tasks/2026-10-29-foundation-upgrade-base-v0-18-1/manifest.json",
    "taskPacket": "docs/harness/tasks/2026-10-29-foundation-upgrade-base-v0-18-1.task.md",
    "evidence": ".harness/evidence/2026-10-29-foundation-upgrade-base-v0-18-1/commands.json",
    "reviewFile": ".harness/evidence/2026-10-29-foundation-upgrade-base-v0-18-1/review.md",
    "changeRef": "none",
    "planRefs": [
      "docs/harness/tasks/2026-10-29-foundation-upgrade-base-v0-18-1.task.md",
      "docs/PROJECT_INHERITANCE.md"
    ]
  }
}
```
