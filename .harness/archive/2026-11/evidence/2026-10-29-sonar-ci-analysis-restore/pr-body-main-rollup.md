## 变更摘要

- 改动层级：inheritance-sync（`pantheon-ops` consumer 侧 evidence 与 manifest 记录）
- 改动模块：`.harness/evidence/2026-10-29-sonar-ci-analysis-restore/{commands.json,summary.md}`、`.harness/tasks/2026-10-29-sonar-ci-analysis-restore/manifest.json`
- 目标问题：`#153`/`#154` 之后 main 上仍留有一个红色上下文，且 evidence 里读到的质量数字是恢复前的旧基线。需要把「GitHub Actions 全绿」与「SonarCloud 质量门禁因真实存量而红」这两件事分别落成事实，并替换掉过期数字。
- 预期影响：evidence 记录 main 的最终 rollup——Code Quality Gates / Security Gates / SonarCloud Scan / Branch Hygiene / Base Compatibility Rebuild 在 `d79dc024` 全部 success（dispatch 与 push 双路径），并说明 push run `40854031854` 的红灯是与同组并发 dispatch 互斥取消造成、已用 `gh run rerun --failed` 复跑为 success；同时把质量门禁读数更正为新基线上的真实值：`new_security_rating` = E、`new_reliability_rating` = C，未关闭问题 62 个漏洞 / 6 个 bug / 174 个 code smell。零产品代码变更。

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
- boundaries: base 拥有共享面；本 PR 只写 ops 自己的 evidence 与 task manifest，不改 base、不改工作流、不改 overlay 声明、不改任何质量门禁配置
- backend response contract: not-applicable
- backend DTO contract: not-applicable
- permission contract: not-applicable
- audit coverage: not-applicable
- visual evidence: not-applicable
- inheritance contract: `docs/PROJECT_INHERITANCE.md` 未变，仍为 `pantheon-base-v0.18.1`
- base drift: none
- Base/ops inheritance: not-applicable（扫描通道根因已在 `#153` 于 consumer 侧闭合）
- Base version checked: `foundation-release.lock.json` releaseVersion / baseCommit = pantheon-base-v0.18.1 / 93505225e36c8d3ba5aa9facc12975eb813167d3
- generic drift: not-applicable
- pseudo-drift: not-applicable
- business-only: no（evidence-only 的继承同步记录）

## 边界说明

- [ ] 本次改动仅涉及 `business/*`
- [ ] 本次改动涉及 Base 重建，已说明 Base-first 边界和 business overlay 兼容性

> 本次不涉及任何代码层：只把已经发生的托管结果与质量门禁读数写进既有任务包，`.github/workflows/*`、`business-overlay.json`、`sonar-project.properties`、lock 与 `.foundation/**` 均未改动。

## 验证记录

- [ ] `npm run check:business-overlay && npm run test:business-overlay`
- [ ] `go test ./...`
- [ ] `cd frontend && npm run lint && npm run type-check && npm run build`
- [x] 业务 smoke 或明确 runtime gap
- [x] GitHub required checks 通过
- [x] Copilot review 已请求，或已说明当前仓库/账号不可用

补充说明：改动仅落在 `.harness/**`，因此本地执行与范围匹配的门禁是 `check:docs-frontmatter`、`check:encoding`、`check:harness-adoption` 与 `check-pr-governance.mjs --event`，`commands.json` 另做 `JSON.parse` 校验；后端/前端门禁在本 PR 未触碰的文件上已于同一棵树全绿。Runtime gap：无新增运行时；运行态证据就是托管侧读数——五个 workflow 的 main run、`/api/project_analyses/search?branch=main` 的基线历史、`/api/qualitygates/project_status` 与 `/api/issues/search` 的门禁与问题计数。

## 审核留痕

- Copilot review：requested（由 PR Automation 自动请求）
- CodeQL 结果：pass（main head `d79dc024` 上 CodeQL Security Scan 与 CodeQL Alert Gate 均 success）
- GitHub checks 结果：main head 上全部 GitHub Actions check 为绿，唯一红灯是 SonarCloud 自有的 `SonarCloud Code Analysis` 质量门禁上下文（run 123382238408）
- Auto-merge：enabled
- Duplication Gate 结果：not-applicable（无源码改动，Markdown/JSON 记录）
- 是否高风险改动：no（仅记录，不含配置或凭证变更；token 值不入库）
- Residual risk / follow-up：① 新基线上的 `new_security_rating` = E / `new_reliability_rating` = C 是真实存量，主要落在 `backend/modules/business/*` 的 GORM 动态条件污点、`filedist` 路径穿越、`cicd/argocd` 关闭证书与主机名校验、e2e 脚本的 npm/npx 生命周期脚本、`security.yml` 的 pip `--only-binary`、以及四个业务列表页的浮动 Promise；这些属于业务层产品修复，需要独立任务包，且部分共享面问题要按 Base-first 回 base。② 未采用任何降噪手段：不把问题标为 false positive、不移动 new code 基线、不换更松的门禁。③ Dependabot 触发运行读不到仓库 secret，扫描在该类 PR 上仍为红，策略由维护者决定。④ Vercel 仍为红，属外部集成且本机 CLI 未登录。

## 检查清单

- [ ] 已明确本次改动归属 `business/*` 或 clean Base rebuild adapter
- [ ] 通用平台/系统域问题已在 Base-first 边界处理
- [ ] 前端新增展示文案已使用 i18n
- [ ] 菜单、权限、接口、审计和 seed/smoke 在范围内同步
- [x] UI 改动已有渲染证据或明确 runtime gap
- [x] 已确认不会泄露敏感配置、账号密码或 Token
