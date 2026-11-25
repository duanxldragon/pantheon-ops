---
title: Base 发版后恢复业务详情路由租户绑定
doc_type: Remediation
layer: business
status: Active
updated_at: 2026-11-25
linked_contracts:
  - docs/contracts/TENANT_CONTRACT_V1.md
  - docs/PROJECT_INHERITANCE.md
---

# Task Packet: 2026-11-25-ops-tenant-detail-route-binding

Base 发版后恢复业务详情路由租户绑定

## Goal

消费携带静态/参数路由隔离修复的正式 foundation release 后，恢复业务详情路由（`GET /packages/:id` 等）的租户绑定，并用双租户反例证明猜 ID 无法跨租户读取。

## Primary Layer

business/*

## Dependency Layers

- business/*
- inheritance-sync

## Harness Profile

- Template: custom
- Overlay: pantheon-ops
- Quality Profile: auth-security
- Delivery Tier: L2
- Priority: P1
- Coverage Dimensions:
  - behaviour
  - architecture-fitness

## Workspace Context

- Target Repository: pantheon-ops
- Repository Role: business-consumer
- Upstream Dependencies: pantheon-base
- Downstream Consumers: none
- Sync Expectation: deferred
- Release Requirement: foundation-release
- Downstream Sync Note: 前置是 Base 正式发布携带 `63940437`（#525 静态/参数路由隔离）的 foundation release，并按锁定快照消费重建；不在 Ops 手抄 Base 文件或 override 路由匹配器。

## Contract Anchors

- `AGENTS.md`
- `docs/PROJECT_INHERITANCE.md`
- `docs/TASK_PACKET_OPS_TEMPLATE.md`
- `docs/contracts/TENANT_CONTRACT_V1.md`
- `docs/reviews/pantheon-ops独立评审与外部报告核实-2026-11-20.md`
- `docs/reviews/pantheon-ops真实环境验证清单-2026-11-20.md`
- `business-overlay.json`
- `foundation-release.lock.json`

## Source Findings

- ARC-001
- V2（真实环境验证清单：列表 200 / export 403 / 详情与 keyMatch2 缺口）

## Scope

### In

- 等 Base 正式 release 携带静态路由守卫后，按锁定快照消费并重建 Ops
- 恢复核心业务详情路由的租户作用域绑定，保留导出等静态路由的独立权限
- 新增双租户反例：猜 ID 详情/导出/回调跨租户被拒，正控同租户详情可用

### Out

- Base 内部路由匹配器实现与 Base 发布动作
- `backend/modules/system/**`、共享权限/租户拦截器
- 未授权生产操作、凭证变更与门禁豁免

## Assumptions and Open Questions

- Confirmed Facts: Base `origin/main` 的 `63940437` 已合并但 `VERSION` 仍为 `0.18.9`、无新 tag，锁定快照 `6316f9b7` 不含该修复。
- Working Assumptions: 该修复随下一次 foundation release 发布，Ops 只做消费与业务侧绑定。
- Open Questions: 新 release 的 tag/commit 与 overlay 兼容性在开工时核实。
- Evidence State: 本包 planned，尚无执行证据。

## Structural Scope

- Affected Subgraph: 请求路由匹配 -> 租户作用域仓储 -> 详情/导出响应
- Boundary Crossings: business/* -> pkg 公共路由契约；共享缺口 -> Base 正式 release -> Ops 重建
- Risk Nodes: ARC-001
- Graph Focus: sensitive-input-flow

## Expected Files

### Create

- `docs/harness/tasks/2026-11-25-ops-tenant-detail-route-binding.task.md`
- `.harness/tasks/2026-11-25-ops-tenant-detail-route-binding/manifest.json`
- `.harness/tasks/2026-11-25-ops-tenant-detail-route-binding/HANDOFF.md`
- `.harness/tasks/2026-11-25-ops-tenant-detail-route-binding/STATUS.md`

### Modify

- `backend/modules/business/deploy/**`
- `backend/modules/business/cmdb/**`
- `backend/modules/business/service/**`
- `business-overlay.json`
- `foundation-release.lock.json`

### Do Not Touch

- `backend/modules/system/**`
- `backend/internal/**`
- `frontend/src/core/**`
- `frontend/src/api/request.ts`
- `既有归档原始证据与其他执行者未提交改动`

## Implementation Notes

- Minimal Complexity Rung: reuse
- 先消费正式 release，再绑定业务详情路由；不得用 URL 改写或反向 override 绕过匹配器。
- 迁移与详情绑定按 T04 已完成的 schema 合同复用，不重做迁移。
- 接口/权限/i18n/overlay 同步关闭；未达验收保持 gap。

## Acceptance Criteria

- [ ] 锁定快照已升级到携带静态路由守卫的正式 release，且 `foundation-release.lock.json` 与重建产物一致
- [ ] 双租户下猜 ID 详情/导出/回调被拒，正控同租户详情 200
- [ ] 路由矩阵与既有权限绑定测试全绿，无新增 403 回归

## Verification Plan

- `go test -count=1 ./modules/business/deploy/...`，工作目录 `backend`
- `go test -count=1 ./modules/business/cmdb/... ./modules/business/service/...`，工作目录 `backend`
- `node scripts/harness/check-boundaries.mjs --strict`，工作目录 `.`

- 路由矩阵：同租户详情 200、跨租户详情/导出 403/404
- 在 backend/ 执行 Go 命令；race 使用 D:\msys64\mingw64\bin、CGO_ENABLED=1 与隔离 MySQL DSN
- 只在实际完成验收时关闭任务；缺 release/环境保持 gap

## Execution Roles

- Implementer Posture: 业务租户路由负责人；只改声明的业务文件，消费正式 release，不手抄 Base。
- Reviewer Posture: 对照 ARC-001/V2、双租户反例与候选哈希审查，优先验证跨租户越权面。

## Stop Points

- Base 正式 release 未发布或锁定快照不可重建时停止，不进入手改路由实现。
- 需要发布动作或生产变更时交维护者。

## State Plan

- Checkpoint Expectation: 开工记录锁定快照 commit 与重建回执；每轮验证追加候选绑定证据。

## Linkage

- Task ID: 2026-11-25-ops-tenant-detail-route-binding
- Task Manifest: .harness/tasks/2026-11-25-ops-tenant-detail-route-binding/manifest.json
- Evidence Directory: .harness/evidence/2026-11-25-ops-tenant-detail-route-binding/
- Review File: none
- OpenSpec Change: none
- Change Ref: none
- Superpowers Plan: none
- Plan References: .harness/tasks/2026-11-20-ops-review-remediation/plan.md

## Evidence Required

- commands.json 含真实执行状态/cwd/源码及构建身份
- summary.md 记录双租户反例结果与 release 身份
- review.md 为 findings-first 审查并含机器可读 linkage

## Human Gates

- 需求/范围偏好、门禁策略与豁免、最终功能验收按维护者三触点契约处理
- 正式发布与最终仓库基线替换需已有授权

## Rollback Plan

回退本包路由绑定改动；release 消费按锁定快照回退到上一版并重建。

## Completion Checklist

- [x] Layer and boundary declared
- [x] Contract anchors read
- [x] Verification run or exception recorded
- [x] Evidence saved or summarized
- [ ] Review completed (independent review not performed; recorded as a gap)

## Status

Authorization binding done under Base v0.19.0 and covered by route/policy contract tests. The HTTP-level dual-tenant counterexample and row-level tenant isolation are not proven here and move to T19. See `.harness/evidence/2026-11-25-ops-tenant-detail-route-binding/summary.md`.
