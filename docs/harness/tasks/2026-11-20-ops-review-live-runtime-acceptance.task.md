---
title: 同候选真实环境与受限角色验收
doc_type: Remediation
layer: business
status: Active
updated_at: 2026-11-20
---

# Task Packet: 2026-11-20-ops-review-live-runtime-acceptance

同候选真实环境与受限角色验收

## Goal

本地启动Pantheon Ops，从CMDB开始验证全部在册业务功能；115.190.251.143仅作SSH部署provider，承接V1-V8和C01-C06/E01-E04原未验收标准。

## Primary Layer

business/*

## Dependency Layers

- business/*
- inheritance-sync

## Dependencies

- blockedBy: none
- Conditional Dependencies: 2026-11-20-ops-filedist-durable-lifecycle (filedist-enabled), 2026-11-20-ops-argocd-ownership-and-webhook (argocd-enabled), 2026-11-20-ops-deploy-template-and-task-experience (after-core-p1)
- Activation: core

## Harness Profile

- Template: custom
- Overlay: pantheon-ops
- Quality Profile: ui-runtime
- Delivery Tier: L2
- Priority: P0
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

- V1
- V2
- V3
- V4
- V5
- V6
- V7
- V8

来源行号随报告当前字节绑定。更正顺序为文末逐条权限核对、基线更新、正文，开工以当前候选复核。详见父计划的发现映射。

## Scope

### In

- 本地启动Pantheon Ops，从CMDB开始验证全部在册业务功能；115.190.251.143仅作SSH部署provider，承接V1-V8和C01-C06/E01-E04原未验收标准
- 冻结源码/lock/构建哈希/数据库版本/provider/角色，隔离环境运行授权、双租户、SSH/K8s、事务、故障、重启、取消与UI矩阵
- 扩展按 T01 启用清单加入 filedist/Argo/Helm，产出日志/响应/数据库副作用/桌面和窄屏截图/清理回执

### Out

- platform/system/*、共享 UI/权限/租户/请求拦截器实现由 pantheon-base 拥有，按正式 foundation release 同步
- 未授权生产操作、发布、凭证变更及门禁豁免
- 资产关系/Pod→Node→VM、真实 Helm Chart、Nacos/RocketMQ、完整 Agent 协议作为独立后续范围

## Assumptions and Open Questions

- Confirmed Facts: 报告基线 444ce8e，当前规划基线 a2d69d4a759b5df2b26aaa9d26bdc5175babe9dc，旧十包已归档且保留未验收 residual。
- Working Assumptions: 按报告采用多租户目标；filedist/ArgoCD 按实际启用能力激活，模板体验增量排在核心P1之后。
- Open Questions: 开工核实父计划/T01已冻结合同，缺证据时保留gap并继续独立的安全可逆步骤。
- Evidence State: 本包in-progress；当前候选后端/前端构建通过，真实密码SSH连接和主机指纹通过，业务运行与视觉验收尚未执行，脚本审查发现问题交Claude修复。

## Structural Scope

- Affected Subgraph: 同候选产物 -> 隔离DB/SSH/K8s/角色 -> 验收证据
- Boundary Crossings: business/* -> 公共契约；共享缺口 -> Base正式release -> Ops重建
- Risk Nodes: V1, V2, V3, V4, V5, V6, V7, V8
- Graph Focus: sensitive-input-flow, call-depth

## Expected Files

### Create

- `docs/harness/tasks/2026-11-20-ops-review-live-runtime-acceptance.task.md`
- `.harness/tasks/2026-11-20-ops-review-live-runtime-acceptance/manifest.json`
- `.harness/tasks/2026-11-20-ops-review-live-runtime-acceptance/HANDOFF.md`
- `.harness/tasks/2026-11-20-ops-review-live-runtime-acceptance/STATUS.md`
- `.harness/evidence/2026-11-20-ops-review-live-runtime-acceptance/commands.json`
- `.harness/evidence/2026-11-20-ops-review-live-runtime-acceptance/summary.md`
- `.harness/evidence/2026-11-20-ops-review-live-runtime-acceptance/review.md`

### Modify

- `docs/designs/BUSINESS_QA_ACCEPTANCE_PLAN.md`

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

- [ ] V1-V8 每项有实际结果、证据及对结论的影响；缺凭据/provider记录 NOT_RUN 和owner，验收保持开放
- [ ] 同候选完整 CGO race、frontend lint/type/build、现有业务smoke、正式重建与drift全部通过
- [ ] 角色与双租户拒绝路径有真实返回和无副作用证据，UI覆盖 loading/empty/error/forbidden/submitting 与暗色/键盘，维护者原范围终验记录

## Verification Plan

- `go test -race ./...`，工作目录 `backend`
- `npm run lint`，工作目录 `frontend`
- `npm run type-check`，工作目录 `frontend`
- `npm run build`，工作目录 `frontend`
- `npm run test:smoke:business:cmdb`，工作目录 `frontend`
- `npm run test:smoke:business:deploy:api`，工作目录 `frontend`
- `npm run test:smoke:business:deploy`，工作目录 `frontend`
- `npm run test:smoke:business:service`，工作目录 `frontend`
- `npm run test:smoke:business:k8s`，工作目录 `frontend`
- `npm run test:smoke:business:observability`，工作目录 `frontend`
- `npm run test:business-overlay`，工作目录 `.`
- `npm run check:business-overlay`，工作目录 `.`
- `npm run rebuild:from-base -- --target .tmp/review-acceptance-candidate`，工作目录 `.`
- `npm run check:shared-drift -- --root .tmp/review-acceptance-candidate`，工作目录 `.`

- V1-V8+C05+已启用E04 原矩阵，保留失败和cleanup原始输出

- 在 backend/ 执行 Go 命令；race 使用 D:\msys64\mingw64\bin，CGO_ENABLED=1 和隔离 MySQL DSN；最终提交门禁执行 go test -race ./...
- 在 frontend/ 执行 lint/type-check/build/test:smoke:*；业务 overlay/重建/治理命令在仓库根执行
- UI 前后使用 impeccable 与 Base DESIGN.md，1440px/390px、明暗主题和相关 loading/empty/error/forbidden/submitting/键盘状态必须有 rendered evidence 或显式 gap
- 服务端二次确认、路由匹配器、tenant公共扩展和 shutdown 公共接口若缺失，登记 Base 上游任务与 release/消费依赖，不在 Ops override
- 只在实际完成验收时关闭任务；缺 provider/凭据/维护者终验保持 gap，安全可逆准备可继续

## Execution Roles

- Implementer Posture: 业务运行验收负责人，只修改声明的业务文件，复用合同并保留已有实现和他人改动。
- Reviewer Posture: 对照本包来源、验收矩阵、实际副作用与候选哈希审查，优先验证授权/并发/旧数据边界。

## Linkage

- Task ID: 2026-11-20-ops-review-live-runtime-acceptance
- Task Manifest: .harness/tasks/2026-11-20-ops-review-live-runtime-acceptance/manifest.json
- Evidence Directory: .harness/evidence/2026-11-20-ops-review-live-runtime-acceptance/
- Review File: none
- OpenSpec Change: none
- Change Ref: none
- Superpowers Plan: none
- Plan References: .harness/tasks/2026-11-20-ops-review-remediation/plan.md

## Evidence Required

- commands.json 含真实执行状态/cwd/源码及构建身份
- summary.md 记录验收结果、runtime/visual gap 和来源判定
- review.md 为真实 findings-first 审查并含机器可读 linkage；执行时更新 linkage.reviewFile
- V1-V8+C05+已启用E04 原矩阵，保留失败和cleanup原始输出

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

## Current Acceptance Stage

维护者最新指令先验V1-V8与旧十包当前候选原范围，原T01-T15依赖作为实际失败触发的followupRemediationTasks保留。Claude执行准备、脚本和修复，Codex验收。
