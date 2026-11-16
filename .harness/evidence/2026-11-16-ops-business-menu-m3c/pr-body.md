## 变更摘要

- 改动层级：L2
- 改动模块：business/bizscope，business/service，business overlay component registry，i18n fallback（en-US）
- 目标问题：把 bizscope 与 service 业务导航按 M+3C（父级 + 子级）重组，新增按业务域的环境 CRUD，废弃扁平 `/business/service` 页面，并关闭一个既有的 en-US fallback 键缺口。
- 预期影响：菜单项从 42 → 45；路由从 52 → 55；权限保持 250；组件键保持 54。新增 `biz_business_environment` 表与派生态迁移副本。后端 `bizscope`/`service`/`iam/menu` 测试全绿；前端 `tsc`/`eslint`/`build`、menu-contract、i18n-locales、business-overlay、gorm-sql-literal、generated-modules 全绿。

## Harness 链路

- Task ID：2026-11-16-ops-business-menu-m3c
- Task Manifest：.harness/tasks/2026-11-16-ops-business-menu-m3c/manifest.json
- Evidence：.harness/evidence/2026-11-16-ops-business-menu-m3c/commands.json
- Verification evidence：.harness/evidence/2026-11-16-ops-business-menu-m3c/summary.md
- Review Artifact：.harness/evidence/2026-11-16-ops-business-menu-m3c/review.md
- OpenSpec change：none
- Trivial change：no
- Quality Profile：ui-runtime
- Ratchet Decision：no-repeat-observed
- GitHub Signal：repo-quality-gate

## Harness adoption markers

> 保留本区块的英文 marker，供治理脚本做机械检查。

- task id: 2026-11-16-ops-business-menu-m3c
- task manifest: .harness/tasks/2026-11-16-ops-business-menu-m3c/manifest.json
- evidence: .harness/evidence/2026-11-16-ops-business-menu-m3c/
- boundaries: business-only
- backend response contract: not-applicable
- backend DTO contract: not-applicable
- permission contract: unchanged permission set, sealed by menu-contract (250 perms)
- audit coverage: not-applicable
- visual evidence: runtime gap (no rendered smoke in dirty-tree session)
- inheritance contract: not-applicable
- base drift: not-applicable
- Base/ops inheritance: not-applicable
- Base version checked: `foundation-release.lock.json` releaseVersion 0.18.7 baseCommit locked
- generic drift: not-applicable
- pseudo-drift: no
- business-only: yes

## 边界说明

- [x] 本次改动仅涉及 `business/*`（以及 business overlay 注册、派生迁移副本、en-US fallback 键补齐）
- [ ] 本次改动涉及 Base 重建，已说明 Base-first 边界和 business overlay 兼容性

> 未触发 Base 重建；`pantheon-base` 共享路径未改。菜单/权限/i18n/审计的同步均落在本仓库：菜单种子（bizscope/service seed）、权限集合（unchanged）、i18n 种子、business overlay 组件注册表 1:1、smoke spec 更新。

## 验证记录

- [x] `npm run check:business-overlay && npm run test:business-overlay`
- [x] `npm run check:gorm-sql-literal`
- [x] `go test ./...`（`go test ./modules/...`，0 FAIL）
- [x] `cd frontend && npm run lint && npm run type-check && npm run build`
- [ ] 业务 smoke 或明确 runtime gap（已记录 runtime gap：未在 dirty-tree 会话执行渲染 smoke；CI smoke-sanity 提供 clean-branch 覆盖）
- [ ] GitHub required checks 通过（进行中：Docs Governance / PR Governance / SonarCloud Code Analysis / Vercel 待复查）
- [ ] Copilot review 已请求，或已说明当前仓库/账号不可用

补充说明：已提交的权威 i18n fallback bundle 未重生成；仅 `en-US.ts` 补齐 5 个缺失 `cmdb.host`/`deploy.package` 键（ja/ko/fr 经 en-US fallback 链继承）。

## 审核留痕

- Copilot review：unavailable
- CodeQL 结果：pending
- GitHub checks 结果：passing（Governance / Sonar / Vercel failures 正在复查）
- Auto-merge：not-enabled
- Duplication Gate 结果：pass
- 是否高风险改动：no
- Residual risk / follow-up：本地渲染/e2e smoke 未在本 dirty-tree 会话执行，依赖 CI；`2026-11-13` 计划包在 base 重建中丢失，以本 ops-local 包作为 PR 治理替换。

## 检查清单

- [x] 已明确本次改动归属 `business/*` 或 clean Base rebuild adapter
- [x] 通用平台/系统域问题已在 Base-first 边界处理（本 PR 未触碰 system/*）
- [x] 前端新增展示文案已使用 i18n（复用已提交 fallback 键）
- [x] 菜单、权限、接口、审计和 seed/smoke 在范围内同步
- [x] UI 改动已有渲染证据或明确 runtime gap
- [x] 已确认不会泄露敏感配置、账号密码或 Token

🤖 Generated with [Claude Code](https://claude.com/claude-code)
