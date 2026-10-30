## 变更摘要

- 改动层级：仅 `docs/harness/tasks/**` 与 `.harness/{evidence,tasks}/**`（任务包文档、summary、commands.json、sonar-triage.json、manifest、本 PR body）；**零代码改动**
- 改动模块：harness 治理记录（任务 2026-10-30-sonar-findings-remediation 的 round 8 验收留痕）
- 目标问题：记录 `main` 已达到 **Issues = 0 且 Security = 0** 的托管事实，并修正 round 7 的一条规则模型
- 预期影响：无运行时影响（零代码改动，不改任何二进制行为、接口、权限、菜单或 seed）；影响面只有治理记录本身——`main` 的验收读数、4 个 issue key 的流转留痕、以及「PR 侧流转会被分支分析继承」这条被读数修正的规则，供下一轮 Sonar 处置直接复用而不必重新试错
- 验收读数（`main` at `430a9f0`，PR #161 的 squash；分析 `2026-10-30T06:38:03+0000`，因自动化合并不产生 push 触发而由 `gh workflow run sonarcloud.yml --ref main` 显式派发）：
  - `/api/issues/search?branch=main&issueStatuses=OPEN,CONFIRMED` → total **0**，type facet `CODE_SMELL 0 / VULNERABILITY 0 / BUG 0`，rule facet 为空
  - `/api/hotspots/search?status=TO_REVIEW` → **0**
  - `/api/qualitygates/project_status` → **OK**（`new_reliability_rating`、`new_security_rating`、`new_maintainability_rating`、`new_duplicated_lines_density`、`new_security_hotspots_reviewed` 全部 OK；门禁定义、new-code 基线与阈值均未改动）
  - measures：`vulnerabilities 0`、`bugs 0`、`code_smells 0`、`security_hotspots 0`、`accepted_issues 0`、`false_positive_issues 36`、`duplicated_lines_density 2.0`、security/reliability/sqale rating 均为 A
  - 任务全程进度：242 open（62 漏洞 + 6 bug + 174 异味）→ 194 → 158 异味 + 1 漏洞 → 3 → **0**
- 本轮四处流转（全部 HTTP 200，逐 key 记录在 `sonar-triage.json` 的 `transitionBatches`，可用 `transition=to_reopen` 撤销）：
  - `argocd_service.go:409`（`branch=main at 7364a44`，key `AaGLf68W8dYv669deilk`）——PR #161 未触及该行，语句文本与 issue 身份跨合并保持不变，因此在分支侧流转
  - `argocd_service.go:274` / `argocd_service.go:409` / `backup_service.go:424`（`pullRequest=161`，keys `AaGLfV7TQ70MdggcuUIK` / `AaGLfV7TQ70MdggcuUIJ` / `AaGLfV_lQ70MdggcuUIL`）——PR 分析回读恰为预测的 3 条 `gosecurity:S3649` 且 **0 条异味**，直接证明「每个声明文件保留 1 处行内字面量」满足 `go:S1192`；流转后 PR 门禁在未重新扫描的情况下即回到 OK，GitHub 的 `SonarCloud Code Analysis` 检查项同时由 fail 转 pass
- 规则模型修正（round 7 的预测被读数推翻）：合并后 `main` 上 `argocd_service.go:274` 与 `backup_service.go:424` 仍以 round 4 的原 key（`AaGHE4OZVq8OAzu0RFHf`、`AaGHE4QxVq8OAzu0RFIK`）与 round 4 的行号呈现为 RESOLVED / FALSE-POSITIVE，尽管该行文本已变为 `Where(idWhereClause, id)` / `Where(whereByID, id)`。也就是说 PR 实例上的 FALSE_POSITIVE 流转会被分支分析继承；文本哈希只在没有在案流转时才会重开实例。round 7 预算的「合并后再流转 3 条」因此没有发生，代价只是合并前的 3 次流转
- 零达成的方式不含任何消音：未新增 `//nolint`/`NOSONAR`/`#nosec`/`eslint-disable`/`@ts-ignore`，未削弱任何测试断言，未改门禁定义/指标/new-code 基线；被改状态的只有上述 4 个 issue key
- 本 PR 自己的验证集合：文档治理面 `check:docs-frontmatter`（435 docs / 280 有 frontmatter 通过）、`check:task-packet-template`、`check:harness-adoption --strict`、`check:structure`、`check:encoding --strict`；JSON 记录文件以 `JSON.parse` 读回校验（`commands.json` 64 条、`sonar-triage.json` 2 个批次 18+4 条、manifest 20 条 implementationNotes）

## Harness 链路

- Task ID：2026-10-30-sonar-findings-remediation
- Task Manifest：.harness/tasks/2026-10-30-sonar-findings-remediation/manifest.json
- Evidence：.harness/evidence/2026-10-30-sonar-findings-remediation/commands.json
- Verification evidence：.harness/evidence/2026-10-30-sonar-findings-remediation/summary.md
- Hosted read-back：.harness/evidence/2026-10-30-sonar-findings-remediation/sonar-triage.json（`postRound8ReadBack`）
- Review Artifact：.harness/evidence/2026-10-30-sonar-findings-remediation/review.md
- OpenSpec change：none
- Trivial change：no
- Quality Profile：ci-workflow
- Ratchet Decision：no-repeat-observed
- GitHub Signal：repo-quality-gate

## Harness adoption markers

> 保留本区块的英文 marker，供治理脚本做机械检查。
>
> 注：本文件是 round 8 的 PR body 留痕，提交在 PR #162 **之前**，因此下面几处「pending / not-applicable」在合并后已翻绿——
> `GitHub checks`（PR #162 的 `Quality Gates`、`Security Gates`、`SonarCloud Code Analysis` 全部 pass，合并后 main 的四个
> push 工作流亦全部 success）、`Duplication Gate`（PR #162 实际运行并 pass）。当前权威读数见 `commands.json` 最后三条记录
> 与 `sonar-triage.json` 的 `postRound8ReadBack.finalMergeEvidence`。

- task id: 2026-10-30-sonar-findings-remediation
- task manifest: `.harness/tasks/2026-10-30-sonar-findings-remediation/manifest.json`
- evidence: `.harness/evidence/2026-10-30-sonar-findings-remediation/commands.json`
- boundaries: 仅治理文档与 evidence JSON；不改 `backend/**`、`frontend/**`、`.github/workflows/**`、门禁定义、Sonar 指标或任何测试断言
- backend response contract: not-applicable（无代码改动）
- backend DTO contract: not-applicable（无代码改动）
- permission contract: not-applicable（未触及路由/权限键/菜单/seed）
- audit coverage: not-applicable（无审计链变更）
- visual evidence: not-applicable（无 UI 面变更，也无渲染差异需要证据）
- inheritance contract: `docs/PROJECT_INHERITANCE.md` 未变
- base drift: `node --test tests/scripts/*.test.mjs` 仍为 177 pass / 14 fail，与 `a5b5910` 干净 worktree 基线同一集合，本 PR 不修
- Base/ops inheritance: 无新增传感器、无规则变更；`check-menu-contract.mjs` 对内联 `const` 的解析缺陷仍按 Base-first 回 base（任务 #38）
- Base version checked: `foundation-release.lock.json` releaseVersion / baseCommit = 未变化（本 PR 不推进 lock）
- generic drift: not-applicable
- pseudo-drift: not-applicable
- business-only: not-applicable（纯治理文档，作用域为 `docs/harness` 与 `.harness`）

## 边界说明

- [x] 本次改动仅涉及 `business/*`（更严格地说：不涉及任何代码，只涉及该 business 任务的治理记录）
- [ ] 本次改动涉及 Base 重建，已说明 Base-first 边界和 business overlay 兼容性

> 第二个框不勾选是准确的：本 PR 不改 base、不改 overlay 声明与 lock，代码零改动。

## 验证记录

- [x] 托管 `main` 分析回读：0 open issues（type facet 三类皆 0）、0 hotspots TO_REVIEW、quality gate OK、`false_positive_issues 36`、`duplicated_lines_density 2.0`
- [x] 托管 PR #161 分析回读：3 条 `gosecurity:S3649` + 0 条异味 → 逐 key 流转后 PR 门禁 OK，`SonarCloud Code Analysis` 检查项 pass
- [x] FP 存活核对：`/api/issues/search?branch=main&issueStatuses=FALSE_POSITIVE&rules=gosecurity:S3649` 返回 33 条，与工作树 `argocd_service.go:274/409`、`backup_service.go:424` 的常量形式逐行对齐（证明流转随 PR 实例继承到分支）
- [x] `npm run check:docs-frontmatter`、`npm run check:task-packet-template`（本 PR 前已跑通，round 7 同集合）
- [x] `node scripts/check-gorm-sql-literal.mjs` exit 0（4 条 waiver 可见）——本 PR 不改代码，传感器结论与 round 7 一致
- [x] evidence JSON 语法校验：`commands.json`（64 records）、`sonar-triage.json`（`transitionBatches` 18+4、新增 `postRound8ReadBack`）、manifest（20 notes）均可 `JSON.parse`
- [x] `main` 流水线：自动合并不产生 push 运行，因此以 `workflow_dispatch` 显式派发并读回——`SonarCloud Scan`（run 41015187554，success）、`Branch Hygiene`（41015609933，success）、`Security Gates`（41015618942，success）、`Code Quality Gates`（41015614741，运行中，本 PR 合并后新提交会再跑一轮）
- [ ] GitHub required checks 通过（合并前要求 `Quality Gates` / `Security Gates` 全绿，`main` 的 `strict=false`）
- [x] Copilot review 已请求（由 PR Automation 自动请求）

## 审核留痕

- Copilot review：requested（由 PR Automation 自动请求）
- CodeQL 结果：not-applicable（本 PR 无代码改动，不新增 sink）
- GitHub checks 结果：pending（`Quality Gates` / `Security Gates` 为 `main` 的 required contexts）
- Duplication Gate 结果：not-applicable for this diff（本 PR 只改 markdown 与 evidence JSON，不新增任何源码行，因此 `node scripts/check-duplication.mjs` 的 base 侧复制粘贴面为空；作为回归位仍跑过 round 7 的 `check:duplication` 全绿集合，且 `main` 的 `duplicated_lines_density` 读数为 2.0）
- Auto-merge：本 PR 创建后需在 required contexts 变绿前用 GraphQL `disablePullRequestAutoMerge` 关闭，否则 `app/github-actions` 会在变绿瞬间落地——PR #160 与 #161 两次都是这样合并的，显式 `gh pr merge --squash --delete-branch` 因「head branch is not up to date with the base branch」被拒
- 分支收口：`fix/sonar-duplication-literals` 与 `refactor/sonar-code-smell-backlog` 在删除前已核验对 `main` **无代码 diff**（`git diff --stat origin/main origin/<b> -- backend frontend scripts .github` 空输出），远端与本地均已删除；`docs/sonar-triage-record`、`fix/sonar-security-findings`、`test/filedist-containment-on-main` 三条早已 squash 合并的本地分支一并清理，当前本地与远端都只剩 `main`
- 是否高风险改动：no（纯治理文档与 evidence 记录；风险点仅在「验收结论是否属实」，因此每条结论都附托管读数与可复查 URL/命令）
- Ratchet 说明：不加新传感器、不改门禁规则。两条被读数改写的认识以任务包 + evidence 固化——① `go:S1192` 把常量声明行计作一次出现（round 7 记录，round 8 的 PR 分析以 0 异味反证修复形状正确）；② FALSE_POSITIVE 流转会随 PR 实例被分支分析继承，因此「合并前在 PR 视图流转」优于「合并后在分支补流转」，round 7 的成本估算据此修正
- Residual risk / follow-up：① `main` 的 push 触发流水线依赖合并方式：由 `app/github-actions` 执行的合并不产生 push 运行，故验收必须显式 `gh workflow run sonarcloud.yml --ref main`；若要该依赖消失，需改 `pr-automation.yml` 或由其承接合并后派发（本轮未改，属工作流面变更需单独评审）。② 首轮派发的扫描曾因 SonarCloud 侧 `POST /analysis/analyses` 返回 HTTP 502 而失败，重跑即通过——说明「分析缺失」与「门禁红灯」必须用日志区分，已记入 `commands.json`。③ 36 条 FALSE_POSITIVE 的成立完全依赖 `scripts/check-gorm-sql-literal.mjs` 在 `docs-governance` 作业中持续阻断动态 SQL；撤销该传感器会同时撤掉这层依据。④ 本地 MySQL 的 11 条 Go 测试失败仍是环境相关既有项，托管 `backend-tests`（全新 `mysql:8.0`）为权威；`tests/scripts/*.test.mjs` 的 14 条失败与 `check:harness-generated` 红灯属 base 漂移既有项。⑤ `biz_argocd_application` 与服务实例无关联列，reconciler 无法推进服务实例状态机，已写入 manifest 的 `deferredCodeIssues`。⑥ `check-menu-contract.mjs` 对内联 `const` 静默解析为空串，按 Base-first 回 base 修（任务 #38）。⑦ Dependabot 运行读不到 `SONAR_TOKEN`、`sonar.sources=.` 会把 PNG evidence 当源码读、无覆盖率上传（`new_coverage` 未被真正评估）、ArgoCD `caCert` 仅 API 可达、Vercel 检查项按维护者指示移出范围且非 required context——均为先前轮次已记录的既有缺口

## 检查清单

- [x] 已明确本次改动归属 `business/*` 或 clean Base rebuild adapter（本 PR 仅治理记录，代码零改动）
- [x] 通用平台/系统域问题已在 Base-first 边界处理（未新增平台/系统域能力副本，未手抄 base 文件；`check-menu-contract` 缺陷回 base）
- [x] 前端新增展示文案已使用 i18n（无新增展示文案）
- [x] 菜单、权限、接口、审计和 seed/smoke 在范围内同步（范围内无这些对象）
- [x] UI 改动已有渲染证据或明确 runtime gap（无 UI 改动）
- [x] 已确认不会泄露敏感配置、账号密码或 Token（SONAR_TOKEN 仅以 shell 变量使用，未写入仓库；diff 为 markdown 与 JSON，不含任何 token、DSN 或凭据字面量；evidence 中的 URL 与 issue key 均为可撤销的公开治理标识）
