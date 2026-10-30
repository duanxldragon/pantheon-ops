## 变更摘要

- 改动层级：inheritance-sync
- 改动模块：`.harness/evidence/2026-10-29-foundation-upgrade-base-v0-18-1/{commands.json,summary.md}`
- 目标问题：#164 把 v0.18.1 台账的 `runtime-acceptance` 转为 `done`，但它由
  `app/github-actions` 的 auto-merge 落地，而本仓已两次测得「自动化 squash 不产生 main 的
  push 工作流」。没有 main 侧读数的收口，等于台账声称完成而验收链在合并后断掉。
- 预期影响：纯证据追加。补上 `main = 4e69224f` 的四个工作流读数，并第三次复现并记录该合并机制；
  不改业务源码、不改锁文件、不改门禁配置。

## Harness 链路

- Task ID：`2026-10-29-foundation-upgrade-base-v0-18-1`
- Task Manifest：`.harness/tasks/2026-10-29-foundation-upgrade-base-v0-18-1/manifest.json`
- Evidence：`.harness/evidence/2026-10-29-foundation-upgrade-base-v0-18-1/commands.json`
- Verification evidence：`.harness/evidence/2026-10-29-foundation-upgrade-base-v0-18-1/summary.md`
- Review Artifact：`.harness/evidence/2026-10-29-foundation-upgrade-base-v0-18-1/review.md`
- OpenSpec change：none
- Trivial change：yes
- Quality Profile：ci-workflow
- Ratchet Decision：no-repeat-observed
- GitHub Signal：runtime-evidence-gate

## Harness adoption markers

> 保留本区块的英文 marker，供治理脚本做机械检查。

- task id: 2026-10-29-foundation-upgrade-base-v0-18-1
- task manifest: .harness/tasks/2026-10-29-foundation-upgrade-base-v0-18-1/manifest.json
- evidence: .harness/evidence/2026-10-29-foundation-upgrade-base-v0-18-1/commands.json
- boundaries: 仅证据文件追加；无源码、无 lock 状态变化（四项 pendingWork 已在 #164 全部 done）
- backend response contract: 不涉及
- backend DTO contract: 不涉及
- permission contract: 不涉及
- audit coverage: 不涉及
- visual evidence: 不涉及
- inheritance contract: 不涉及（releaseVersion / baseCommit 未变，`docs/PROJECT_INHERITANCE.md` marker 不动）
- base drift: 不涉及
- Base/ops inheritance: 无需发布；本轮吸收的是本仓 main 的运行时读数
- Base version checked: `foundation-release.lock.json` releaseVersion / baseCommit = `pantheon-base-v0.18.1` / `93505225`
- generic drift: handled in `pantheon-base` first (not-applicable)
- pseudo-drift: excluded as line-count or formatting noise without contract break (no)
- business-only: no（本轮只动继承台账的证据面）

## 边界说明

- [ ] 本次改动仅涉及 `business/*`
- [x] 本次改动涉及 Base 重建，已说明 Base-first 边界和 business overlay 兼容性

> 边界：本轮把「main 侧读数缺失」这一已知机制问题记录为待决，而不是顺手改合并流程——
> 把 auto-merge 关掉换成显式合并属于流程变更，归维护者裁决。

## 验证记录

- [x] `npm run check:business-overlay`（exit 0）；`test:business-overlay` 不涉及（无 overlay 变化）
- [x] `npm run check:gorm-sql-literal`（不涉及：无 Go 源码改动）
- [x] `go test ./...`（不涉及：无 Go 源码改动；#164 轮已跑租户与平台处理器测试）
- [x] `cd frontend && npm run lint && npm run type-check && npm run build`（不涉及：无前端改动）
- [x] 业务 smoke 或明确 runtime gap：#164 已给出隔离 schema 的运行时读数；本轮补 main 侧——
      PR #164 读数 18 pass / 3 skipping / 1 fail（Vercel 外部集成，非必需上下文），
      `main = 4e69224f`，四个 main 工作流（Code Quality Gates / Security Gates /
      SonarCloud Scan / Branch Hygiene）手工派发后全部 completed success
- [x] 机制复现记录：`api .../actions/runs?head_sha=4e69224f` → total_count 0、
      `commits/4e69224f/check-runs` 为空，与 `7364a44`、`430a9f0` 同形；显式合并的
      `38a07c0`、`b85e1a3` 则四个 push 工作流全跑。第三次观测，仍按既有处置手工派发。
- [x] 治理矩阵：`check:docs-frontmatter` / `check:harness-inventory` / `check:structure` /
      `check:encoding` / `check:harness-encoding` / `check:pr-governance` /
      `check:failure-registry` → exit 0；`check-evidence --strict` 与
      `check-review --strict` 对本包 PASS
- [x] GitHub required checks 通过（必需上下文仅 Quality Gates 与 Security Gates）
- [ ] Copilot review 已请求，或已说明当前仓库/账号不可用（见审核留痕）

补充说明：本 PR 不含任何抑制手段，未使用 Sonar 凭据，凭据仅在本机进程内解析且未写入仓库。

## 审核留痕

- Copilot review：unavailable（本仓库/账号无法解析 `copilot` reviewer）
- CodeQL 结果：本 PR 仅 `.json`/`.md` 证据追加，无新增抽取面；回读后确认
- GitHub checks 结果：待回读
- Auto-merge：enabled（沿用仓库 squash 策略；本 PR 若由自动化落地，main 读数同样需手工派发补齐）
- Duplication Gate 结果：PASS（`npm run check:duplication` exit 0）
- 是否高风险改动：no（证据追加）
- Residual risk / follow-up：
  1. 「自动化 squash 不触发 main push run」第三次复现。可选处置：收口改用显式
     `gh pr merge --squash`，或给 main 增加定时 `workflow_dispatch` 复核。属流程变更，待维护者定。
  2. `check:harness-docs` 与 `check:task-packet` 在本仓仍红（继承自 base 文档的悬空链接与 68 个历史包
     的词表问题）。实测在干净 `main` 上同样失败，非本轮引入；D3/D6 已登记。
  3. `check:release-assets` / `check:shared-drift` 在本仓 MODULE_NOT_FOUND（package.json 引用了
     未随 v0.18.1 落地的脚本），同样是干净 main 上的既有缺口。
  4. 浏览器侧 `tenant-console.spec.ts` 仍未跑，UI 面继续显式留空。

## 检查清单

- [x] 已明确本次改动归属 `business/*` 或 clean Base rebuild adapter（本次为 inheritance-sync 证据收口）
- [x] 通用平台/系统域问题已在 Base-first 边界处理（无新增平台问题；继承缺口显式指向 base）
- [x] 前端新增展示文案已使用 i18n（不涉及）
- [x] 菜单、权限、接口、审计和 seed/smoke 在范围内同步（不涉及源码改动）
- [x] UI 改动已有渲染证据或明确 runtime gap（无 UI 改动；gap 显式保留）
- [x] 已确认不会泄露敏感配置、账号密码或 Token
