---
title: 真实环境残余验收（多租户、受限角色、SA 轮换、Ingress、Velero、race Redis）
doc_type: Remediation
layer: business
status: Active
updated_at: 2026-11-25
linked_contracts:
  - docs/contracts/TENANT_CONTRACT_V1.md
  - docs/reviews/pantheon-ops真实环境验证清单-2026-11-20.md
---

# Task Packet: 2026-11-25-ops-residual-live-acceptance

真实环境残余验收（多租户、受限角色、SA 轮换、Ingress、Velero、race Redis）

## Goal

承接 T16 已完成同候选验证之后仍不可得的环境类项，在具备环境后逐项执行并留同候选证据，不以推断关闭。

## Primary Layer

business/*

## Dependency Layers

- business/*
- inheritance-sync

## Harness Profile

- Template: custom
- Overlay: pantheon-ops
- Quality Profile: ui-runtime
- Delivery Tier: L2
- Priority: P0
- Coverage Dimensions:
  - behaviour
  - runtime-quality

## Workspace Context

- Target Repository: pantheon-ops
- Repository Role: business-consumer
- Upstream Dependencies: pantheon-base
- Downstream Consumers: none
- Sync Expectation: deferred
- Release Requirement: none
- Downstream Sync Note: 只消费正式 foundation release 与既有业务实现；需要共享修复时回 Base 走正式 release。

## Contract Anchors

- `AGENTS.md`
- `docs/PROJECT_INHERITANCE.md`
- `docs/TASK_PACKET_OPS_TEMPLATE.md`
- `docs/contracts/TENANT_CONTRACT_V1.md`
- `docs/reviews/pantheon-ops真实环境验证清单-2026-11-20.md`
- `docs/designs/BUSINESS_QA_ACCEPTANCE_PLAN.md`
- `foundation-release.lock.json`

## Source Findings

- V4 双租户（`tenant_mode=multi` 未配置）
- V7 低权限 SSH restart/cancel 用户
- V8 SA 凭证撤销轮换
- K8s shelf：Ingress provider 变更/快照、WebSocket server 侧资源释放、Velero schedule/restore
- `GAP-ARGOCD-ENVKEY`（运行环境缺 `PANTHEON_ENCRYPTION_KEY`）
- 完整 Go race 所需 `127.0.0.1:26316` 测试 Redis

## Scope

### In

- V4 双租户端到端：租户 B 看不到 A 资源，猜 ID/修改/导出/回调无跨租户副作用
- V7 低权限 SSH restart/cancel 用户下的注入拒绝、指纹、租约跨重启与远程取消
- V8 SA 凭证持续请求下轮换、旧 client 不复活、撤销后合法请求正常
- Ingress provider 变更与快照、Velero schedule/restore、WebSocket server 侧资源释放观察
- 完整 `go test -race ./...`（含测试 Redis）与本轮候选 receipt
- `GAP-ARGOCD-ENVKEY` 用探针专用密钥补跑并记录

### Out

- 全新安装/升级/回滚与最终交付收口（T17）
- 业务实现改动；发现缺陷回到对应实现包
- 生产环境变更、凭据变更与门禁豁免

## Assumptions and Open Questions

- Confirmed Facts: T16 已关闭 deploy 7/7、K8s 17/17（k8s38）、CAP 31/32、V-probe 19/19、T03 角色矩阵 18/18；剩余项均为环境或依赖不可得。
- Working Assumptions: 环境由维护者提供；未提供时保持 gap，不做推断性关闭。
- Open Questions: 测试 Redis 26316 与多租户配置由谁提供、何时可用。
- Evidence State: 本包 planned，无执行证据。

## Structural Scope

- Affected Subgraph: 运行环境 -> 业务 API/队列 -> k8s/ArgoCD provider
- Boundary Crossings: business/* -> 外部 provider；环境依赖 -> 维护者配置
- Risk Nodes: V4/V7/V8
- Graph Focus: runtime-quality

## Expected Files

### Create

- `docs/harness/tasks/2026-11-25-ops-residual-live-acceptance.task.md`
- `.harness/tasks/2026-11-25-ops-residual-live-acceptance/manifest.json`
- `.harness/tasks/2026-11-25-ops-residual-live-acceptance/HANDOFF.md`
- `.harness/tasks/2026-11-25-ops-residual-live-acceptance/STATUS.md`
- `.harness/evidence/2026-11-25-ops-residual-live-acceptance/commands.json`
- `.harness/evidence/2026-11-25-ops-residual-live-acceptance/summary.md`
- `.harness/evidence/2026-11-25-ops-residual-live-acceptance/review.md`

### Modify

- `.harness/evidence/2026-11-25-ops-residual-live-acceptance/**`

### Do Not Touch

- `backend/modules/system/**`
- `backend/internal/**`
- `frontend/src/core/**`
- `foundation-release.lock.json`
- `既有归档原始证据与其他执行者未提交改动`

## Implementation Notes

- Minimal Complexity Rung: reuse
- 复用 T16 既有探针、lane 脚本与证据格式，不重写 harness。
- 每项记录真实环境身份（候选 commit/二进制哈希、DB 名、provider 版本）。
- 环境不可得时写 gap，不用推断替代运行。

## Acceptance Criteria

- [ ] V4/V7/V8 与 K8s shelf 项各有同候选真实运行证据或显式环境 gap
- [ ] 完整 race 结果按包分类，DATA RACE 为 0 或缺陷回对应实现包
- [ ] `GAP-ARGOCD-ENVKEY` 有补跑结果或明确环境策略结论

## Verification Plan

- `go test -race -count=1 ./...`，工作目录 `backend`（需测试 Redis 127.0.0.1:26316 与隔离 MySQL）
- `node scripts/check-runtime-evidence.mjs --root .`，工作目录 `.`
- 复用 T16 lane 脚本执行 V4/V7/V8 与 k8s shelf 项

- 真实双租户、受限 SSH 用户、SA 轮换与 Velero 备份恢复运行证
- 只在实际完成验收时关闭任务；缺环境保持 gap

## Execution Roles

- Implementer Posture: 真实验收负责人；复用既有探针，只追加证据，不改产品实现。
- Reviewer Posture: 对照 V1-V8 验收口径检查候选身份、环境身份与证据完整性。

## Stop Points

- 环境（多租户、受限用户、测试 Redis、Ingress/Velero）不可得时停止该项并记录 gap。
- 需要生产变更或凭据操作时交维护者。

## State Plan

- Checkpoint Expectation: 每项运行后写候选绑定回执；同一无变化尝试不重复执行。

## Linkage

- Task ID: 2026-11-25-ops-residual-live-acceptance
- Task Manifest: .harness/tasks/2026-11-25-ops-residual-live-acceptance/manifest.json
- Evidence Directory: .harness/evidence/2026-11-25-ops-residual-live-acceptance/
- Review File: none
- OpenSpec Change: none
- Change Ref: none
- Superpowers Plan: none
- Plan References: .harness/tasks/2026-11-20-ops-review-remediation/plan.md

## Evidence Required

- commands.json 含真实执行状态/cwd/环境与候选身份
- summary.md 记录各项结果、环境 gap 与来源判定
- review.md 为 findings-first 审查并含机器可读 linkage

## Human Gates

- 需求/范围偏好、门禁策略与豁免、最终功能验收按维护者三触点契约处理
- 环境提供、生产窗口与外部凭据操作需已有授权

## Rollback Plan

只产生证据与配置，无产品改动；环境配置按维护者策略恢复。

## Completion Checklist

- [x] Layer and boundary declared
- [ ] Contract anchors read
- [ ] Verification run or exception recorded
- [ ] Evidence saved or summarized
- [ ] Review completed
