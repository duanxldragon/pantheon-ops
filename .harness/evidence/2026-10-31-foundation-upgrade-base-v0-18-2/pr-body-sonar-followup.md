## 变更摘要

- 改动层级：new-code quality repair（承接任务包 `2026-10-31-foundation-upgrade-base-v0-18-2`）
- 改动模块：`backend/modules/business/k8s/{k8s.go,k8s_seed.go}`、
  `frontend/src/modules/business/k8s/index.ts`、对应 evidence 目录
- 目标问题：#166 的 SonarCloud Code Analysis 在新代码上报了 6 条 `go:S1192`（k8s_seed.go 的
  重复字面量）和 16 行重复（3.187% > 3% 门槛）。修复提交 `76cbc91` 推送时，#166 已在 01:59:39Z
  被自动合并（squash 为 `5676ffc`），修复因此留在已关闭的 head 上，没有进入 main。
- 预期影响：把同一份修复从合并后的 main 重放到独立分支并入主干；菜单集合、排序、237 条权限键、
  管理员绑定、路由表都不变，只是把 seed 里重复的字符串换成命名常量、把新增权限键按资源两两成行，
  让 Sonar 的新代码读数回到门槛内。

## Harness 链路

- Task ID：`2026-10-31-foundation-upgrade-base-v0-18-2`
- Task Manifest：`.harness/tasks/2026-10-31-foundation-upgrade-base-v0-18-2/manifest.json`
- Evidence：`.harness/evidence/2026-10-31-foundation-upgrade-base-v0-18-2/commands.json`
- Verification evidence：`.harness/evidence/2026-10-31-foundation-upgrade-base-v0-18-2/summary.md`
- Review Artifact：`.harness/evidence/2026-10-31-foundation-upgrade-base-v0-18-2/review.md`
- OpenSpec change：none
- Trivial change：no
- Quality Profile：i18n
- Ratchet Decision：no-repeat-observed
- GitHub Signal：repo-quality-gate

## Harness adoption markers

> 保留本区块的英文 marker，供治理脚本做机械检查。

- task id: 2026-10-31-foundation-upgrade-base-v0-18-2
- task manifest: .harness/tasks/2026-10-31-foundation-upgrade-base-v0-18-2/manifest.json
- evidence: .harness/evidence/2026-10-31-foundation-upgrade-base-v0-18-2/commands.json
- boundaries: 业务修复只落在 `backend/modules/business/k8s/**` 与 `frontend/src/modules/business/k8s/index.ts`；不触碰共享面与发行锁
- backend response contract: 未改动
- backend DTO contract: 未改动
- permission contract: 权限 key 集合与语义不变（仍为 237 条），仅把 titleKey 的重复字面量常量化
- audit coverage: 未改动；本轮不触碰审计链路
- visual evidence: 未提供（无 UI 渲染改动）
- inheritance contract: 未改动；本 PR 不移动 base 版本 marker
- base drift: 未改动共享文件，沿用任务包记录的 813 identical / 10 allowlisted / 161 skipped
- Base/ops inheritance: 仍为 `pantheon-base-v0.18.2`（`a44ae4e0`），本 PR 只补新代码质量修复
- Base version checked: `foundation-release.lock.json` releaseVersion / baseCommit = `pantheon-base-v0.18.2` / `a44ae4e012fff20430b1b7f91cb7a9296f3ea843`
- generic drift: 无
- pseudo-drift: 无
- business-only: yes

## 边界说明

- [x] 本次改动仅涉及 `business/*`

> 边界：两个后端文件与一个前端模块文件都是 `business/*` 归属；evidence 目录同步记录返工路径，
> 不修改共享面、发行锁或 `business-overlay.json`。

## 验证记录

- [x] `git cherry-pick 76cbc91` 到 `origin/main`（`5676ffc`）上 → clean
- [x] `cd backend && gofmt -l modules/business/k8s/ && go build ./... && go vet ./modules/business/k8s/` → 空 / 0 / 0
- [x] `node frontend/scripts/check-menu-contract.mjs` → `40 个菜单，50 条路由，237 个权限，52 个组件键`
- [x] `cd frontend && npm run lint && npm run type-check` → 0 / 0（在 `76cbc91` 的原始验证中记录，重放未改变文件内容）
- [x] 资源与继承门禁不适用：本 PR 不移动 lock、不重建 overlay
- [x] 业务 smoke 或明确 runtime gap：无运行时改动；k8s seed 只做字面量常量化，运行时生成的
      权限行内容与键名完全一致，runtime 缺口沿用任务包 `knownGaps`

## 审核留痕

- Copilot review：unavailable（本仓库/账号无法解析 `copilot` reviewer）
- CodeQL 结果：待 PR 检查回读
- GitHub checks 结果：待回读
- Auto-merge：enabled（沿用仓库既有 squash 策略）
- Duplication Gate 结果：PASS
- 是否高风险改动：no（无产品行为变更；权限键集合不变）
- Residual risk / follow-up：
  1. #166 的 SonarCloud Code Analysis 读数在历史上保持 red；新的读数只由本 PR 产生。
  2. 若新代码重复率仍高于 3%，继续在同一分支上把重复块拆开，不豁免、不移动基线。

## 检查清单

- [x] 已明确本次改动归属 `business/*`
- [x] 通用平台/系统域问题已在 Base-first 边界处理（本 PR 无新增平台问题）
- [x] 前端新增展示文案已使用 i18n（沿用任务包中已落地的 20 条权限标题）
- [x] 菜单、权限、接口、审计和 seed/smoke 在范围内同步（权限键集合与前端声明保持一致）
- [x] UI 改动已有渲染证据或明确 runtime gap（无 UI 渲染改动）
- [x] 已确认不会泄露敏感配置、账号密码或 Token
