## 变更摘要

- 改动层级：inheritance-sync（`pantheon-ops` consumer 侧 evidence 记录）
- 改动模块：`.harness/evidence/2026-10-29-sonar-ci-analysis-restore/{commands.json,summary.md,pr-body.md}`
- 目标问题：`#153` 恢复了 `sonar-project.properties` 并把它声明为 repository overlay，扫描在 PR head 上已经 `ANALYSIS SUCCESSFUL`，但合并后的托管结果（PR check rollup 与 `main` 分支基线分析）还停留在 evidence 里的 `pending` 记录上，验收条件没有落成事实。
- 预期影响：把两条托管证据写入同一任务包的 evidence——PR run `40851210564` 通过并把结果发布到 `pullRequest=153`；合并后 `gh workflow run sonarcloud.yml --ref main`（run `40852770827`，提交 `8799b2e9`）通过，`/api/project_analyses/search?branch=main` 出现 `2026-10-29T09:50:45+0000 @ 8799b2e9`，即自 2026-08-22 之后第一次 `main` 基线分析。同时补齐两条已知缺口：Dependabot 触发运行读不到仓库 secret、`sonar.sources=.` 会把已提交的 PNG 当文本读取并产生编码告警。零产品代码变更。

## Harness 链路

- Task ID：2026-10-29-sonar-ci-analysis-restore
- Task Manifest：.harness/tasks/2026-10-29-sonar-ci-analysis-restore/manifest.json
- Evidence：.harness/evidence/2026-10-29-sonar-ci-analysis-restore/commands.json
- Verification evidence：.harness/evidence/2026-10-29-sonar-ci-analysis-restore/summary.md
- Review Artifact：.harness/evidence/2026-10-29-sonar-ci-analysis-restore/review.md
- OpenSpec change：none
- Trivial change：yes
- Quality Profile：ci-workflow
- Ratchet Decision：no-repeat-observed
- GitHub Signal：repo-quality-gate

## Harness adoption markers

> 保留本区块的英文 marker，供治理脚本做机械检查。

- task id: 2026-10-29-sonar-ci-analysis-restore
- task manifest: `.harness/tasks/2026-10-29-sonar-ci-analysis-restore/manifest.json`
- evidence: `.harness/evidence/2026-10-29-sonar-ci-analysis-restore/commands.json`
- boundaries: base 拥有共享面；本 PR 只写 ops 自己的 evidence 目录，不改 base、不改工作流、不改 overlay 声明
- backend response contract: not-applicable
- backend DTO contract: not-applicable
- permission contract: not-applicable
- audit coverage: not-applicable
- visual evidence: not-applicable
- inheritance contract: `docs/PROJECT_INHERITANCE.md` 未变，仍为 `pantheon-base-v0.18.1`
- base drift: none
- Base/ops inheritance: not-applicable（无 base 侧问题，根因已在 `#153` 于 consumer 侧闭合）
- Base version checked: `foundation-release.lock.json` releaseVersion / baseCommit = pantheon-base-v0.18.1 / 93505225e36c8d3ba5aa9facc12975eb813167d3
- generic drift: not-applicable
- pseudo-drift: not-applicable
- business-only: no（evidence-only 的继承同步记录）

## 边界说明

- [ ] 本次改动仅涉及 `business/*`
- [ ] 本次改动涉及 Base 重建，已说明 Base-first 边界和 business overlay 兼容性

> 本次不涉及任何代码层：只把已经发生的托管结果写入既有任务包的 evidence 目录，`.github/workflows/*`、`business-overlay.json`、lock 与 `.foundation/**` 均未改动。

## 验证记录

- [ ] `npm run check:business-overlay && npm run test:business-overlay`
- [ ] `go test ./...`
- [ ] `cd frontend && npm run lint && npm run type-check && npm run build`
- [x] 业务 smoke 或明确 runtime gap
- [x] GitHub required checks 通过
- [x] Copilot review 已请求，或已说明当前仓库/账号不可用

补充说明：改动仅落在 `.harness/evidence/**`，因此本地执行与改动范围匹配的门禁为 `check:docs-frontmatter`（434 docs 通过）、`check:encoding`（2093 文件 0 finding）、`check:task-packet-template`（OK）、`check:harness-adoption`（0 finding）；`commands.json` 通过 `JSON.parse` 校验。后端与前端门禁在 `#153` 的同一棵树上已全绿，本 PR 未触碰任何 Go/TS 文件，不适用。Runtime gap：无新增运行时；运行态证据即托管侧两条结果——PR run `40851210564` 与 main dispatch `40852770827`，以及 `main` 基线分析记录。

## 审核留痕

- Copilot review：requested（由 PR Automation 自动请求）
- CodeQL 结果：pending-hosted-run
- GitHub checks 结果：pending-hosted-run
- Auto-merge：enabled
- Duplication Gate 结果：not-applicable（无源码改动，Markdown/JSON 记录）
- 是否高风险改动：no（仅 evidence 记录，不含配置或凭证变更；token 值不入库）
- Residual risk / follow-up：① Dependabot 触发的运行读不到仓库 secret，扫描在该类 PR 上仍为红，是否配置 Dependabot secret 或对 `dependabot[bot]` 跳过 job 属维护者门禁策略决策；② 项目质量门禁 `new_security_rating` 为 ERROR（4 vulnerabilities / 245 code smells），通道恢复后这些真实存量重新可见，工作流不因门禁 ERROR 变红；③ Vercel 仍为红，属外部集成；④ `sonar.sources=.` 会扫描已提交 PNG 产生编码告警，收窄源码集是独立决策。

## 检查清单

- [ ] 已明确本次改动归属 `business/*` 或 clean Base rebuild adapter
- [ ] 通用平台/系统域问题已在 Base-first 边界处理
- [ ] 前端新增展示文案已使用 i18n
- [ ] 菜单、权限、接口、审计和 seed/smoke 在范围内同步
- [x] UI 改动已有渲染证据或明确 runtime gap
- [x] 已确认不会泄露敏感配置、账号密码或 Token
