## 变更摘要

- 改动层级：business/*；平台层与系统域来自正式 Base release
- 改动模块：foundation overlay、CMDB host state version、bizscope/deploy/observability 回归
- 目标问题：消费 Base v0.18.3，保留业务修复并消除业务迁移与 Base 000031 的编号冲突
- 预期影响：旧 host 零版本回填后执行 CAS；业务迁移来源、生成目标和漂移校验一致

## Harness 链路

- Task ID：2026-11-05-base-v0183-consumption
- Task Manifest：.harness/tasks/2026-11-05-base-v0183-consumption/manifest.json
- Evidence：.harness/evidence/2026-11-05-base-v0183-consumption/commands.json
- Verification evidence：.harness/evidence/2026-11-05-base-v0183-consumption/summary.md
- Review Artifact：.harness/evidence/2026-11-05-base-v0183-consumption/review.md
- OpenSpec change：none
- Trivial change：no
- Quality Profile：auth-security
- Ratchet Decision：gate-updated
- GitHub Signal：repo-quality-gate

## Harness adoption markers

> 保留本区块的英文 marker，供治理脚本做机械检查。

- task id: 2026-11-05-base-v0183-consumption
- task manifest: .harness/tasks/2026-11-05-base-v0183-consumption/manifest.json
- evidence: .harness/evidence/2026-11-05-base-v0183-consumption/commands.json
- boundaries: Base owns shared platform/system; Ops owns business code, migration sources and consumer assembly
- backend response contract: Inherited Base PLATFORM_CONTRACT; business errors retain existing semantics
- backend DTO contract: Inherited Base contracts; no consumer override
- permission contract: Business capability and scope fixtures verified under existing contracts
- audit coverage: Versioned migration and state CAS regressions; official inherited security fixes
- visual evidence: Existing Base rendered evidence; final Ops business smoke pending
- inheritance contract: docs/PROJECT_INHERITANCE.md
- base drift: Official artifact hashes and final rebuild pending
- Base/ops inheritance: Consume official release through rebuild-from-base only
- Base version checked: `foundation-release.lock.json` releaseVersion / baseCommit =
- generic drift: handled in `pantheon-base` first (yes / no / not-applicable)
- pseudo-drift: excluded as line-count or formatting noise without contract break (yes / no)
- business-only:

## 边界说明

- [ ] 本次改动仅涉及 `business/*`
- [ ] 本次改动涉及 Base 重建，已说明 Base-first 边界和 business overlay 兼容性

> 如果跨层，请说明 Base 与 Ops 各自职责、Base commit、overlay 变化，以及是否影响菜单/权限/i18n/审计。

## 验证记录

- [ ] `npm run check:business-overlay && npm run test:business-overlay`
- [ ] `npm run check:gorm-sql-literal`
- [ ] `go test ./...`
- [ ] `cd frontend && npm run lint && npm run type-check && npm run build`
- [ ] 业务 smoke 或明确 runtime gap
- [ ] GitHub required checks 通过
- [ ] Copilot review 已请求，或已说明当前仓库/账号不可用

补充说明：

业务 SQL 的 canonical 来源属于 modules/business，生成目标由装配器固定推导。缺失源文件、篡改目标、未声明路径及 Base 编号冲突均失败。37项 overlay 回归通过；隔离MySQL验证缺表/缺列/Down后Up/CAS。正式重建后的全量验证结果写入同一任务证据。


## 审核留痕

- Copilot review：unavailable; independent gpt-6-astra code/architecture reviews
- CodeQL 结果：Current consumer CI pending
- GitHub checks 结果：Current consumer CI pending
- Auto-merge：not-enabled
- Duplication Gate 结果：Current consumer CI pending
- 是否高风险改动：yes; official foundation sync, migration and ownership gate
- Residual risk / follow-up：Formal release/rebuild/full verification pending; production DDL not executed; real external providers require their deployment environment

## 检查清单

- [ ] 已明确本次改动归属 `business/*` 或 clean Base rebuild adapter
- [ ] 通用平台/系统域问题已在 Base-first 边界处理
- [ ] 前端新增展示文案已使用 i18n
- [ ] 菜单、权限、接口、审计和 seed/smoke 在范围内同步
- [ ] UI 改动已有渲染证据或明确 runtime gap
- [ ] 已确认不会泄露敏感配置、账号密码或 Token
