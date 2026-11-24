---
title: 能力文档、测试资产与治理对账
doc_type: Remediation
layer: business
status: Archived
updated_at: 2026-11-24
linked_contracts:
  - docs/designs/BUSINESS_FUNCTIONAL_DESIGN.md
  - docs/contracts/TENANT_CONTRACT_V1.md
---

# Task Packet: 2026-11-20-ops-review-docs-and-test-governance

能力文档、测试资产与治理对账

## Goal

对齐资产关系/Pod追溯、Helm登记、Agent、Nacos/RocketMQ现状及版本字段角色，修复当前真实失效链接

## Primary Layer

business/*

## Dependency Layers

- business/*
- inheritance-sync

## Dependencies

- blockedBy: 2026-11-20-ops-review-baseline-and-decisions, 2026-11-20-ops-business-permissions-and-task-visibility
- Conditional Dependencies: none
- Activation: core

## Harness Profile

- Template: custom
- Overlay: pantheon-ops
- Quality Profile: ui-runtime
- Delivery Tier: L2
- Priority: P2
- Coverage Dimensions:
  - behaviour
  - architecture-fitness
  - runtime-quality

## Workspace Context

- Target Repository: pantheon-ops
- Repository Role: business-consumer
- Upstream Dependencies: pantheon-base
- Downstream Consumers: none
- Sync Expectation: deferred
- Release Requirement: none
- Downstream Sync Note: 共享根因先回Base修复；需要时单列正式release与锁定重建消费任务。

## Contract Anchors

- `AGENTS.md`
- `docs/PROJECT_INHERITANCE.md`
- `docs/TASK_PACKET_OPS_TEMPLATE.md`
- `docs/reviews/pantheon-ops独立评审与外部报告核实-2026-11-20.md`
- `docs/reviews/pantheon-ops真实环境验证清单-2026-11-20.md`
- `docs/designs/BUSINESS_FUNCTIONAL_DESIGN.md`
- `docs/designs/BUSINESS_QA_ACCEPTANCE_PLAN.md`
- `business-overlay.json`
- `foundation-release.lock.json`

## Source Findings

- NEW-02
- NEW-03
- NEW-04
- 文档与版本
- 冗余与遗留代码

来源行号随报告当前字节绑定。更正顺序为文末逐条权限核对、基线更新、正文，开工以当前候选复核。详见父计划的发现映射。

## Scope

### In

- 对齐资产关系/Pod追溯、Helm登记、Agent、Nacos/RocketMQ现状及版本字段角色，修复当前真实失效链接
- 保留已接入 CI 的核心业务 smoke；核实业务单测与旧e2e资产是否被聚合调用，修正 stale API/未接入的实际缺口并补安全反例
- 小范围合并已确认重复 normalizeListLimit；deadcode有结果才移除候选，保留有意双份生成迁移；共享 CI/版本/钩子变更回 Base 再release消费

### Out

- platform/system/*、共享 UI/权限/租户/请求拦截器实现由 pantheon-base 拥有，按正式 foundation release 同步
- 未授权生产操作、发布、凭证变更及门禁豁免
- 资产关系/Pod→Node→VM、真实 Helm Chart、Nacos/RocketMQ、完整 Agent 协议作为独立后续范围

## Assumptions and Open Questions

- Confirmed Facts: 报告基线 444ce8e，当前规划基线 a2d69d4a759b5df2b26aaa9d26bdc5175babe9dc，旧十包已归档且保留未验收 residual。
- Working Assumptions: 按报告采用多租户目标；filedist/ArgoCD 按实际启用能力激活，模板体验增量排在核心P1之后。
- Open Questions: 开工核实父计划/T01已冻结合同，缺证据时保留gap并继续独立的安全可逆步骤。
- Evidence State: 本包planned，后续验证与review工件在实际执行时创建。

## Structural Scope

- Affected Subgraph: 业务能力/测试资产 -> 文档/CI调用 -> 门禁证据
- Boundary Crossings: business/* -> 公共契约；共享缺口 -> Base正式release -> Ops重建
- Risk Nodes: NEW-02, NEW-03, NEW-04, 文档与版本, 冗余与遗留代码
- Graph Focus: sensitive-input-flow, call-depth

## Expected Files

### Create

- `docs/harness/tasks/2026-11-20-ops-review-docs-and-test-governance.task.md`
- `.harness/tasks/2026-11-20-ops-review-docs-and-test-governance/manifest.json`
- `.harness/tasks/2026-11-20-ops-review-docs-and-test-governance/HANDOFF.md`
- `.harness/tasks/2026-11-20-ops-review-docs-and-test-governance/STATUS.md`
- `.harness/evidence/2026-11-20-ops-review-docs-and-test-governance/commands.json`
- `.harness/evidence/2026-11-20-ops-review-docs-and-test-governance/summary.md`
- `.harness/evidence/2026-11-20-ops-review-docs-and-test-governance/review.md`

### Modify

- `README.md`
- `docs/designs/BUSINESS_*`
- `frontend/tests/**`
- `e2e-tests/**`
- `business-overlay.json`

### Do Not Touch

- `backend/modules/system/**`
- `backend/internal/**`
- `frontend/src/core/**`
- `frontend/src/api/request.ts`
- `foundation-release.lock.json`
- `既有归档原始证据与其他执行者未提交改动`

## Implementation Notes

- Minimal Complexity Rung: reuse
- 审查主基线 444ce8e，当前规划基线 a2d69d4a759b5df2b26aaa9d26bdc5175babe9dc；每包开工核实当前候选，先复用 #195 实现与归档证据再补缺口
- 本包状态 planned；命令与运行场景是后续验证计划，执行者须记录真实 cwd/时间/exitCode/skip/源码哈希
- 接口/权限/菜单/i18n/schema/seed/fixture/业务overlay 同步关闭；业务 migration 源文件归业务模块，装配副本由重建器生成

- 核实当前候选和前置合同，逐条对照来源与旧验收，标记已经完成的实现
- 先复用现有合同/组件/测试，针对本包反例验证失败路径与状态副作用
- 在声明文件范围内完成最小修改，同步权限/i18n/迁移/overlay/fixture
- 运行本包验证，保存候选绑定 evidence，findings-first review 后更新状态

## Acceptance Criteria

- [ ] 现行业务文档与开放能力相符，Helm primary=registered/历史raw保留，版本含义清楚
- [ ] 五项已存在CI smoke保留，新授权/租户/恢复反例确实被运行；旧e2e路径与当前路由一致或退役有引用证据
- [ ] 本批文件链接/任务/evidence/review检查零新增错误，既有红灯有准确基线和owner，不新增全量英文翻译项目

## Verification Plan

- `npm run check:task-packet`，工作目录 `.`
- `node scripts/harness/check-doc-links.mjs --root .`，工作目录 `.`
- `node scripts/harness/check-doc-frontmatter.mjs --root .`，工作目录 `.`
- `npm run check:business-overlay`，工作目录 `.`

- E03 登记能力桌面/手机渲染，实际执行CI聚合与单测用例清单

- 在 backend/ 执行 Go 命令；race 使用 D:\msys64\mingw64\bin，CGO_ENABLED=1 和隔离 MySQL DSN；最终提交门禁执行 go test -race ./...
- 在 frontend/ 执行 lint/type-check/build/test:smoke:*；业务 overlay/重建/治理命令在仓库根执行
- UI 前后使用 impeccable 与 Base DESIGN.md，1440px/390px、明暗主题和相关 loading/empty/error/forbidden/submitting/键盘状态必须有 rendered evidence 或显式 gap
- 服务端二次确认、路由匹配器、tenant公共扩展和 shutdown 公共接口若缺失，登记 Base 上游任务与 release/消费依赖，不在 Ops override
- 只在实际完成验收时关闭任务；缺 provider/凭据/维护者终验保持 gap，安全可逆准备可继续

## Execution Roles

- Implementer Posture: 业务文档与验证负责人，只修改声明的业务文件，复用合同并保留已有实现和他人改动。
- Reviewer Posture: 对照本包来源、验收矩阵、实际副作用与候选哈希审查，优先验证授权/并发/旧数据边界。

## Linkage

- Task ID: 2026-11-20-ops-review-docs-and-test-governance
- Task Manifest: .harness/tasks/2026-11-20-ops-review-docs-and-test-governance/manifest.json
- Evidence Directory: .harness/evidence/2026-11-20-ops-review-docs-and-test-governance/
- Review File: .harness/evidence/2026-11-20-ops-review-docs-and-test-governance/review.md
- OpenSpec Change: none
- Change Ref: none
- Superpowers Plan: none
- Plan References: .harness/tasks/2026-11-20-ops-review-remediation/plan.md

## Evidence Required

- commands.json 含真实执行状态/cwd/源码及构建身份
- summary.md 记录验收结果、runtime/visual gap 和来源判定
- review.md 为真实 findings-first 审查并含机器可读 linkage；执行时更新 linkage.reviewFile
- E03 登记能力桌面/手机渲染，实际执行CI聚合与单测用例清单

## Human Gates

- 需求/范围偏好、门禁策略与豁免、最终视觉/功能验收按维护者三触点契约处理
- 生产窗口/外部凭据操作/正式发布/最终仓库基线替换需已有授权

## Rollback Plan

优先回退本包可逆代码；数据库按隔离演练过的数据回滚/备份方案处理，multi数据回滚遵循Base合同，生产动作另经授权

## Completion Checklist

- [x] Layer and boundary declared
- [x] Contract anchors read
- [x] Verification run or exception recorded
- [x] Evidence saved or summarized
- [x] Review completed
