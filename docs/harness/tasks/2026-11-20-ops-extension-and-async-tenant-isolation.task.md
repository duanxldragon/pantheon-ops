---
title: 扩展业务与异步链路租户隔离
doc_type: Remediation
layer: business
status: Active
updated_at: 2026-11-20
linked_contracts:
  - docs/designs/BUSINESS_FUNCTIONAL_DESIGN.md
  - docs/contracts/TENANT_CONTRACT_V1.md
---

# Task Packet: 2026-11-20-ops-extension-and-async-tenant-isolation

扩展业务与异步链路租户隔离

## Goal

按 T01 清单完成 T04 之外 tenant-owned/derived/overridable 表、唯一键、路由与关联隔离，核实已有 observability 局部租户实现

## Primary Layer

business/*

## Dependency Layers

- business/*
- inheritance-sync

## Dependencies

- blockedBy: 2026-11-20-ops-business-permissions-and-task-visibility, 2026-11-20-ops-core-business-tenant-isolation
- Conditional Dependencies: none
- Activation: core

## Harness Profile

- Template: custom
- Overlay: pantheon-ops
- Quality Profile: ui-runtime
- Delivery Tier: L2
- Priority: P1
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

- ARC-001

来源行号随报告当前字节绑定。更正顺序为文末逐条权限核对、基线更新、正文，开工以当前候选复核。详见父计划的发现映射。

## Scope

### In

- 按 T01 清单完成 T04 之外 tenant-owned/derived/overridable 表、唯一键、路由与关联隔离，核实已有 observability 局部租户实现
- 后台同步/告警评估/通知/文件分发任务存储租户身份并显式传给授权与provider，缓存和队列身份不跨租户复用
- 按启用能力验证公开callback、系统启动任务和 legacy数据授权；业务迁移编号与 T04 协调，共享扩展接口缺口 Base-first

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

- Affected Subgraph: 持久化tenant -> 同步/告警/文件job -> callback/通知/provider
- Boundary Crossings: business/* -> 公共契约；共享缺口 -> Base正式release -> Ops重建
- Risk Nodes: ARC-001
- Graph Focus: sensitive-input-flow, call-depth

## Expected Files

### Create

- `docs/harness/tasks/2026-11-20-ops-extension-and-async-tenant-isolation.task.md`
- `.harness/tasks/2026-11-20-ops-extension-and-async-tenant-isolation/manifest.json`
- `.harness/tasks/2026-11-20-ops-extension-and-async-tenant-isolation/HANDOFF.md`
- `.harness/tasks/2026-11-20-ops-extension-and-async-tenant-isolation/STATUS.md`
- `.harness/evidence/2026-11-20-ops-extension-and-async-tenant-isolation/commands.json`
- `.harness/evidence/2026-11-20-ops-extension-and-async-tenant-isolation/summary.md`
- `.harness/evidence/2026-11-20-ops-extension-and-async-tenant-isolation/review.md`

### Modify

- `backend/modules/business/{cicd,filedist,observability,secrets,opsinit}/**`
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

- [ ] T01 全部业务表逐表已处理或有合同支持的分类依据，现有无 tenant_id 模型不会隐式成为共享资源
- [ ] 双租户查询/写入/回调/同步/告警/文件job各有反例，重启与异步执行身份保持正确
- [ ] compat 历史数据、已有 observability tenant数据均可升级且无重复/丢失；未启用扩展的数据归属也已定义

## Verification Plan

- `go test -count=1 ./modules/business/...`，工作目录 `backend`
- `npm run test:business-overlay`，工作目录 `.`
- `npm run check:business-overlay`，工作目录 `.`

- V4 扩展数据/异步隔离，两域legacy与当前对象并存

- 在 backend/ 执行 Go 命令；race 使用 D:\msys64\mingw64\bin，CGO_ENABLED=1 和隔离 MySQL DSN；最终提交门禁执行 go test -race ./...
- 在 frontend/ 执行 lint/type-check/build/test:smoke:*；业务 overlay/重建/治理命令在仓库根执行
- UI 前后使用 impeccable 与 Base DESIGN.md，1440px/390px、明暗主题和相关 loading/empty/error/forbidden/submitting/键盘状态必须有 rendered evidence 或显式 gap
- 服务端二次确认、路由匹配器、tenant公共扩展和 shutdown 公共接口若缺失，登记 Base 上游任务与 release/消费依赖，不在 Ops override
- 只在实际完成验收时关闭任务；缺 provider/凭据/维护者终验保持 gap，安全可逆准备可继续

## Execution Roles

- Implementer Posture: 扩展业务租户迁移负责人，只修改声明的业务文件，复用合同并保留已有实现和他人改动。
- Reviewer Posture: 对照本包来源、验收矩阵、实际副作用与候选哈希审查，优先验证授权/并发/旧数据边界。

## Linkage

- Task ID: 2026-11-20-ops-extension-and-async-tenant-isolation
- Task Manifest: .harness/tasks/2026-11-20-ops-extension-and-async-tenant-isolation/manifest.json
- Evidence Directory: .harness/evidence/2026-11-20-ops-extension-and-async-tenant-isolation/
- Review File: none
- OpenSpec Change: none
- Change Ref: none
- Superpowers Plan: none
- Plan References: .harness/tasks/2026-11-20-ops-review-remediation/plan.md

## Evidence Required

- commands.json 含真实执行状态/cwd/源码及构建身份
- summary.md 记录验收结果、runtime/visual gap 和来源判定
- review.md 为真实 findings-first 审查并含机器可读 linkage；执行时更新 linkage.reviewFile
- V4 扩展数据/异步隔离，两域legacy与当前对象并存

## Human Gates

- 需求/范围偏好、门禁策略与豁免、最终视觉/功能验收按维护者三触点契约处理
- 生产窗口/外部凭据操作/正式发布/最终仓库基线替换需已有授权

## Rollback Plan

优先回退本包可逆代码；数据库按隔离演练过的数据回滚/备份方案处理，multi数据回滚遵循Base合同，生产动作另经授权

## Completion Checklist

- [x] Layer and boundary declared
- [ ] Contract anchors read
- [ ] Verification run or exception recorded
- [ ] Evidence saved or summarized
- [ ] Review completed
