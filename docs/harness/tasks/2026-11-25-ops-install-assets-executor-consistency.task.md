---
title: 安装包、模板与执行器能力一致性
doc_type: Remediation
layer: business
status: Active
updated_at: 2026-11-25
linked_contracts:
  - docs/contracts/TENANT_CONTRACT_V1.md
  - docs/PROJECT_INHERITANCE.md
---

# Task Packet: 2026-11-25-ops-install-assets-executor-consistency

安装包、模板与执行器能力一致性

## Goal

消除“声明了但不可用”的安装能力：固定模板与 files/amd64 预置包建立机械一致性检查；Agent 执行器在无协议时前后端同时禁用；Helm 仅快照的口径在 UI 与文档中明示。

## Primary Layer

business/*

## Dependency Layers

- business/*
- inheritance-sync

## Harness Profile

- Template: custom
- Overlay: pantheon-ops
- Quality Profile: runtime-quality
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
- Sync Expectation: none
- Release Requirement: none
- Downstream Sync Note: 仅 business/*，无需 Base 同步。

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

- DEP-004
- NEW-03

## Scope

### In

- 新增检查：每个固定模板声明的安装包必须有预置包或显式标注“需自备包”，缺失则门禁失败或警示
- Agent 执行器：validExecutorType 与前端选项一并收敛，无调度分支时不可创建
- Helm 页面与文档明示“登记快照，不执行安装”
- jdk_systemd 已实机；nginx_systemd 等延后项在本包登记 fresh-install 前置而不重复验收

### Out

- 补齐 MySQL/MinIO/Harbor 安装包本身
- Agent 协议实现
- Helm 真实 Chart 安装
- 全新安装/升级/回滚收口（归 T17）

## Assumptions and Open Questions

- Confirmed Facts: 固定模板含 Nginx/MySQL/Redis/MinIO/Harbor/JDK/Prometheus；files/amd64 仅有 JDK、Nginx、Redis、Prometheus、node_exporter、VictoriaMetrics。deploy_service.go validExecutorType 接受 agent，但 dispatchStartedTask 对非 SSH 直接返回。k8s/helm/helm.go 包注释声明仅为数据库快照。
- Working Assumptions: 复用既有后端合同与测试，仅补缺口。
- Open Questions: 开工时核实当前候选与 T11 进度，避免重复实现。
- Evidence State: 本包 planned，尚无执行证据。

## Structural Scope

- Affected Subgraph: 请求路由匹配 -> 租户作用域仓储 -> 详情/导出响应
- Boundary Crossings: business/* -> pkg 公共路由契约；共享缺口 -> Base 正式 release -> Ops 重建
- Risk Nodes: ARC-001
- Graph Focus: sensitive-input-flow

## Expected Files

### Create

- `docs/harness/tasks/2026-11-25-ops-install-assets-executor-consistency.task.md`
- `.harness/tasks/2026-11-25-ops-install-assets-executor-consistency/manifest.json`
- `.harness/tasks/2026-11-25-ops-install-assets-executor-consistency/HANDOFF.md`
- `.harness/tasks/2026-11-25-ops-install-assets-executor-consistency/STATUS.md`

### Modify

- `backend/modules/business/deploy/**`
- `backend/modules/business/k8s/helm/**`
- `frontend/src/modules/business/deploy/**`
- `scripts/**`
- `docs/designs/BUSINESS_DEPLOY_MODULE_DESIGN.md`

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

### Work Breakdown

- S1 一致性检查脚本：读取 deploy_fixed_templates.go 的模板与包名约定，对照 files/amd64，缺包且无“需自备包”标记则失败；新增 npm 脚本并给缺包反例的单元测试
- S2 模板元数据：为 MySQL/MinIO/Harbor 等无预置包的模板增加“需自备包”显式标记，UI 选择包时提示，不让用户走到执行阶段才失败
- S3 Agent 执行器：validExecutorType 去掉 agent（或保留但创建期拒绝并返回明确错误键），前端 executorType 联合类型与选项同步收敛；补创建/更新/导入三条入口的回归测试，注意 deploy_service.go 的更新与导入路径也要覆盖
- S4 存量数据：已存在 executor_type=agent 的任务行不得被误改，只读展示并标注不可执行；补迁移或读取兼容测试
- S5 Helm 口径：k8s Helm 页面与 BUSINESS_DEPLOY/K8S 设计文档明示“登记快照，不执行安装”，与 helm.go 包注释一致
- S6 登记延后项：nginx_systemd 等需 fresh-install 环境的验证明确挂到 T17，不在本包重复
- S7 证据：go test deploy/helm；check-boundaries --strict；脚本反例输出

## Acceptance Criteria

- [ ] 一致性检查接入 npm 脚本或门禁，并有缺包反例测试
- [ ] agent 类型在 API 与 UI 均被拒绝或隐藏，含回归测试
- [ ] Helm 口径在 UI/文档可见
- [ ] 无新增对 base 或 system 域的改动

## Verification Plan

- `go test -count=1 ./modules/business/deploy/... ./modules/business/k8s/helm/...`，工作目录 `backend`
- `node scripts/harness/check-boundaries.mjs --strict`，工作目录 `.`
- 只在实际完成验收时关闭任务；缺环境保持 gap

## Execution Roles

- Implementer Posture: 业务租户路由负责人；只改声明的业务文件，消费正式 release，不手抄 Base。
- Reviewer Posture: 对照 ARC-001/V2、双租户反例与候选哈希审查，优先验证跨租户越权面。

## Stop Points

- 发现需改 Base/共享层时停止，登记上游任务。
- 需要发布动作或生产变更时交维护者。

## State Plan

- Checkpoint Expectation: 开工记录锁定快照 commit 与重建回执；每轮验证追加候选绑定证据。

## Linkage

- Task ID: 2026-11-25-ops-install-assets-executor-consistency
- Task Manifest: .harness/tasks/2026-11-25-ops-install-assets-executor-consistency/manifest.json
- Evidence Directory: .harness/evidence/2026-11-25-ops-install-assets-executor-consistency/
- Review File: none
- OpenSpec Change: none
- Change Ref: none
- Superpowers Plan: none
- Plan References: .harness/tasks/2026-11-20-ops-review-remediation/plan.md

## Evidence Required

- commands.json 含真实执行状态/cwd/源码及构建身份
- summary.md 记录验收结果与 gap
- review.md 为 findings-first 审查并含机器可读 linkage

## Human Gates

- 需求/范围偏好、门禁策略与豁免、最终功能验收按维护者三触点契约处理
- 正式发布与最终仓库基线替换需已有授权

## Rollback Plan

回退本包可逆代码。

## Completion Checklist

- [x] Layer and boundary declared
- [ ] Contract anchors read
- [ ] Verification run or exception recorded
- [ ] Evidence saved or summarized
- [ ] Review completed
