---
title: 服务实例到部署的前端入口与完整分页
doc_type: Remediation
layer: business
status: Active
updated_at: 2026-11-25
linked_contracts:
  - docs/contracts/TENANT_CONTRACT_V1.md
  - docs/PROJECT_INHERITANCE.md
---

# Task Packet: 2026-11-25-ops-service-instance-deploy-entry

服务实例到部署的前端入口与完整分页

## Goal

补齐用户主路径“服务实例 → 发起部署 → 任务终态/日志回看”的前端入口：实例列表完整分页、实例上直接发起部署并预填服务/实例/主机/环境、任务详情与实例详情互相回链。承接 T11 的前端部分，后端准入与回写复用既有实现。

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
- `../pantheon-base/docs/designs/BUSINESS_RESOURCE_LIST_PATTERN.md`
- `../pantheon-base/docs/designs/BUSINESS_LIFECYCLE_DETAIL_PATTERN.md`

## Source Findings

- DEP-002

## Scope

### In

- ServiceInstanceList 去掉 pageSize=100 且关闭分页的硬上限，改服务端分页、筛选，101+ 实例可达
- 实例行增加“部署”操作，预填 serviceId/serviceInstanceId/目标主机/环境，调用既有预览与启动接口
- 任务详情回显所属服务与实例，实例详情回显最近任务与终态，二者可互相跳转
- 按 impeccable 与 DESIGN.md 补桌面/390px、loading/empty/error/forbidden/submitting 渲染证据与 Playwright smoke

### Out

- 后端准入、回写、引用保护的重写（已在 deploy/service 包内，只补缺口）
- 凭据管理页（归 T05）
- Agent 执行器与 Helm 真实安装
- Base 共享 UI/请求拦截器实现

## Assumptions and Open Questions

- Confirmed Facts: 评审时 frontend/src/modules/business/deploy 下 serviceId/serviceInstanceId 零引用；ServiceInstanceList.tsx 以 getServiceInstanceList({page:1,pageSize:100}) 加载且 pagination={false}，仅有“同步”操作。后端 service_link.go、ApplyServiceInstanceState 已就绪并有测试。
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

- `docs/harness/tasks/2026-11-25-ops-service-instance-deploy-entry.task.md`
- `.harness/tasks/2026-11-25-ops-service-instance-deploy-entry/manifest.json`
- `.harness/tasks/2026-11-25-ops-service-instance-deploy-entry/HANDOFF.md`
- `.harness/tasks/2026-11-25-ops-service-instance-deploy-entry/STATUS.md`

### Modify

- `frontend/src/modules/business/{serviceinstance,service,deploy}/**`
- `frontend/tests/smoke/business/{service,deploy}/**`
- `frontend/src/modules/business/**/locales/**`

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

- S1 后端：TaskQuery 增加 serviceId/serviceInstanceId 过滤并下推到 SQL（deploy_dto.go、deploy_service.go ListTasks，导出复用同一过滤），service 包 InstanceQuery 增加 environment 过滤；补带数据范围的反例测试（越范围调用方得到空结果，不泄露存在性）。状态：已在真实 MySQL 上验证新增测试与邻近回归通过；deploy 整包超时缺口见 STATUS
- S2 前端 API：service/api.ts getServiceInstanceList 的参数类型补 environment/targetType/status；deploy/api.ts 的 getDeployTaskList 参数本就是宽松 Record，传 serviceInstanceId 无需改动，只在调用处封装带类型的辅助函数
- S3 实例列表：按 Base BUSINESS_RESOURCE_LIST_PATTERN 重构 ServiceInstanceList（PageHeader/FilterPanel/ListHeaderActions、服务端分页），去掉 pageSize=100 与 pagination={false}；分页上限决策（须在开工时二选一并记录）：(a) 前端 pageSize 永不超过 100，保持 service 包 normalizePage 现状（size>100 静默回落 20）；(b) service 包改用 Base 共享 common.NormalizePagination（超限夹到上限，deploy 包同理会改行为），属行为变更须补测试。加环境/目标类型/状态筛选，保留 reconcile
- S4 部署入口：实例行增加“部署”操作（权限 business:deploy:task:create 才显示），复用 DeployTaskList 的创建弹窗并预填 serviceId/serviceInstanceId/environment/目标主机；先把弹窗装配从 1556 行的 DeployTaskList 抽成可复用组件再接入，禁止复制粘贴整段表单
- S5 回链：DeployTaskDetail 展示所属服务/实例并跳转实例；实例详情展示最近任务与终态（依赖 S1）
- S6 异常路径：stale 目标、实例被并发删除/下线、无权限直调接口，页面给出可恢复提示；补 i18n（zh-CN/en-US 必填，其余语言按仓库门禁）
- S7 测试与证据：tests/smoke/business/service 增加 101+ 实例分页、筛选、受限角色；tests/smoke/business/deploy 增加“实例→预览→启动→终态/日志”主流程；1440px/390px 与 loading/empty/error/forbidden/submitting 渲染证据落 .harness/evidence；跑 test:smoke:business:service/deploy 与 go test deploy/service

## Acceptance Criteria

- [ ] 实例列表服务端分页，造 101+ 实例后第 101 条可见、可筛选
- [ ] 从实例发起部署：预览→启动→终态/日志，回显实例当前版本与状态
- [ ] 无部署权限角色不显示部署入口，直调接口被拒；stale 目标恢复提示正确
- [ ] 桌面与 390px 渲染证据齐备，smoke 覆盖主路径
- [ ] 实例列表与部署入口符合 Base BUSINESS_RESOURCE_LIST_PATTERN：PageHeader/FilterPanel/ListHeaderActions、AppTable 服务端分页、状态色用 token、权限点四段式；偏离处在 review 中逐条说明
- [ ] 任务与实例详情回链符合 Base BUSINESS_LIFECYCLE_DETAIL_PATTERN（hero stats、子资源表、动作 in-flight 状态）；渲染证据覆盖该模板 §11 页面状态变体清单

## Verification Plan

- `npm run test:smoke:business:service`，工作目录 `frontend`
- `npm run test:smoke:business:deploy`，工作目录 `frontend`
- `go test -count=1 ./modules/business/service/... ./modules/business/deploy/...`，工作目录 `backend`
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

- Task ID: 2026-11-25-ops-service-instance-deploy-entry
- Task Manifest: .harness/tasks/2026-11-25-ops-service-instance-deploy-entry/manifest.json
- Evidence Directory: .harness/evidence/2026-11-25-ops-service-instance-deploy-entry/
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
