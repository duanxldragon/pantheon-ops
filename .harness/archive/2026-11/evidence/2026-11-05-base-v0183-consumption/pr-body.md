## 变更摘要

- 改动层级：Base 共享快照消费（backend/frontend/scripts/docs 共享基线）+ `business/*` 适配（000032 迁移重编号）+ 消费工具链（GitHub-only fetch）
- 改动模块：共享基线全量、`business/cmdb/host` 迁移、`scripts/business-overlay/*`、`.github/workflows/inheritance-drift-detection.yml`
- 目标问题：将 ops 基线从 v0.18.2 升级到官方 GitHub foundation release `pantheon-base-v0.18.4`（含 minimatch GHSA-g3hv-75wg-m249 修复），并把消费路径收敛为“只从官方 release 资产获取”，同时修复 host-state backfill 迁移编号冲突（000031 → 000032 under module）
- 预期影响：共享代码与 base v0.18.4 字节一致；overlay 契约、迁移链（000031 base + 000032 ops）、CI 工作流均按 GitHub-only 消费模式刷新

## Harness 链路

- Task ID：2026-11-05-base-v0183-consumption
- Task Manifest：.harness/tasks/2026-11-05-base-v0183-consumption/manifest.json
- Evidence：.harness/evidence/2026-11-05-base-v0183-consumption/commands.json
- Verification evidence：.harness/evidence/2026-11-05-base-v0183-consumption/summary.md
- Review Artifact：.harness/evidence/2026-11-05-base-v0183-consumption/review.md
- OpenSpec change：不适用
- Trivial change：no
- Quality Profile：ci-workflow
- Ratchet Decision：gate-updated
- GitHub Signal：repo-quality-gate

## Harness adoption markers

> 保留本区块的英文 marker，供治理脚本做机械检查。

- task id: 2026-11-05-base-v0183-consumption
- task manifest: .harness/tasks/2026-11-05-base-v0183-consumption/manifest.json
- evidence: .harness/evidence/2026-11-05-base-v0183-consumption/summary.md
- boundaries: business/* 适配 + base shared snapshot 经 rebuild-from-base 组合，未手抄 base main 文件
- backend response contract: 未改动（随 base v0.18.4 快照更新）
- backend DTO contract: 未改动（随 base v0.18.4 快照更新）
- permission contract: 未改动（随 base v0.18.4 快照更新）
- audit coverage: 未改动（随 base v0.18.4 快照更新）
- visual evidence: 无 UI 净改动；host-real 页面差异已在 staging 对照基线复验为既有问题
- inheritance contract: `npm run check:business-overlay` 通过（53 契约路径）
- base drift: shared-drift 826 文件一致 / 10 allowlisted / 2 派生迁移显式说明
- Base/ops inheritance: rebuild-from-base 官方 release 路线，scaffold 827 文件
- Base version checked: `foundation-release.lock.json` releaseVersion / baseCommit = pantheon-base-v0.18.4 / 0a8229c67ab482c2e163622fc4e5b87301c53399
- generic drift: handled in `pantheon-base` first (yes)
- pseudo-drift: excluded as line-count or formatting noise without contract break (yes)
- business-only: no（跨层消费，business 侧仅 000032 迁移重编号与 overlay 适配）

## 边界说明

- [ ] 本次改动仅涉及 `business/*`
- [x] 本次改动涉及 Base 重建，已说明 Base-first 边界和 business overlay 兼容性

Base 职责：v0.18.4 快照（含 minimatch override、OIDC/scaffold 移除、000031 数据域迁移）。Ops 职责：`business/*` 域与 overlay 注册表组合；`business/cmdb/host` 的 state-version backfill 迁移重编号为 000032 避免与 base 000031 冲突。消费工具链改为 GitHub-only：`npm run fetch:base-release` 校验 4 项 release 资产（SHA256 + GitHub digest + git-archive commit + manifest 契约）后原子写入 lock。菜单/权限/i18n/审计无净变化，均来自 base 快照本身。

## 验证记录

- [x] `npm run check:business-overlay && npm run test:business-overlay`（57/57 通过）
- [x] `npm run check:gorm-sql-literal`
- [x] `go test ./...`（staging 全量 CGO race，exit 0；Windows 需 `CGO_ENABLED=1` + mingw PATH）
- [x] `cd frontend && npm run lint && npm run type-check && npm run build`（218 单测通过）
- [x] 业务 smoke 或明确 runtime gap（generated-module 4/5；host-real 与 tenant fresh-DB 失败已在 v0.18.2 基线复现，属既有 fixture 状态依赖，交由托管 CI 复验）
- [ ] GitHub required checks 通过
- [x] Copilot review 已请求，或已说明当前仓库/账号不可用（copilot-swe-agent 非本仓库 collaborator，API 请求 422，无法指派）

补充说明：fresh DB 迁移链在 staging 实测回放至 version=32、dirty=0（`PANTHEON_AUTO_MIGRATE=false` golang-migrate 模式，DB `pantheon_ops_v0184_staging`）。基线替换后工作区与 staging 树 byte-identical（backend/frontend/src/scripts/tests 共享范围为 0 差异）。

## 审核留痕

- Copilot review：unavailable（copilot-swe-agent 非仓库 collaborator，无法通过 API 指派评审）
- CodeQL 结果：待 CI
- GitHub checks 结果：待 CI
- Auto-merge：not-enabled
- Duplication Gate 结果：本地 PASS（structure 0 findings）
- 是否高风险改动：是（基线替换 + workflow 变更，属继承边界与 CI 面）
- Residual risk / follow-up：15 个 `test:scripts` 失败已在 HEAD 复现证明为既有问题；2 个 base 继承 checker（check-boundaries/check-coverage）未接线、`check:harness-docs` 3 处断链均为 base 侧既有状态，随下轮 base-first 处理

## 检查清单

- [x] 已明确本次改动归属 `business/*` 或 clean Base rebuild adapter
- [x] 通用平台/系统域问题已在 Base-first 边界处理
- [x] 前端新增展示文案已使用 i18n
- [x] 菜单、权限、接口、审计和 seed/smoke 在范围内同步
- [x] UI 改动已有渲染证据或明确 runtime gap
- [x] 已确认不会泄露敏感配置、账号密码或 Token
