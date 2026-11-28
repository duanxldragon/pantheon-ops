---
title: CMDB 主机与服务实例、业务域关联视图
doc_type: Remediation
layer: business
status: Active
updated_at: 2026-11-25
linked_contracts:
  - docs/contracts/TENANT_CONTRACT_V1.md
  - docs/PROJECT_INHERITANCE.md
---

# Task Packet: 2026-11-25-ops-cmdb-asset-relation-views

CMDB 主机与服务实例、业务域关联视图

## Goal

让 CMDB 主机、业务域、服务实例三者在界面上互相可查：主机详情展示承载的服务实例与最近部署，业务域详情展示其下应用/服务/实例统计；补 bizscope 的跨租户、跨业务域与并发删除测试。通用资产模型与 Pod→Node→VM 追溯不在本包。

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
- `../pantheon-base/docs/designs/BUSINESS_RESOURCE_LIST_PATTERN.md`
- `../pantheon-base/docs/designs/BUSINESS_LIFECYCLE_DETAIL_PATTERN.md`

## Source Findings

- NEW-02

## Scope

### In

- 主机详情增加“关联服务实例/最近部署任务”只读视图，复用 HasActiveHostReferences 与现有读取接口
- 业务域详情增加应用/服务/实例计数与跳转，环境字符串与实例 environment 的一致性校验复用 service 侧准入
- bizscope 补跨租户、跨业务域引用、并发删除/改名、环境绑定回归测试
- 受限角色与数据范围下关联视图不泄露不可见对象

### Out

- 通用资产类型、自定义属性、变更历史
- Pod→Node→VM 追溯（新立项）
- K8s 模块改动
- Base 共享组件实现

## Assumptions and Open Questions

- Confirmed Facts: cmdb 仅有 host/group/label 三类模型；CmdbHostDetail.tsx 与 BizScopeDetail.tsx 中无 service/instance 引用；bizscope 仅 2 个测试文件；ServiceInstance.HostID 为裸外键。
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

- `docs/harness/tasks/2026-11-25-ops-cmdb-asset-relation-views.task.md`
- `.harness/tasks/2026-11-25-ops-cmdb-asset-relation-views/manifest.json`
- `.harness/tasks/2026-11-25-ops-cmdb-asset-relation-views/HANDOFF.md`
- `.harness/tasks/2026-11-25-ops-cmdb-asset-relation-views/STATUS.md`

### Modify

- `backend/modules/business/{bizscope,cmdb,service}/**`
- `frontend/src/modules/business/{cmdb,bizscope}/**`
- `frontend/tests/smoke/business/cmdb/**`

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

- S1 后端读取接口：按 hostId 列出承载的服务实例（带数据范围，复用 service 包 instanceQuery 与 HasActiveHostReferences），按 businessScopeId 汇总应用/服务/实例计数；新路由登记权限路由矩阵与 seed
- S2 一致性：实例的 environment 与所属业务域环境的对应关系，复用 service 侧准入，不在 cmdb 再写一份规则；不一致时返回明确错误键
- S3 bizscope 测试：跨租户、跨业务域引用、环境绑定改名/删除时的引用保护、并发删除；当前只有 2 个测试文件，先补这些反例再动功能
- S4 主机详情：CmdbHostDetail 增加“关联服务实例/最近部署”只读区块，无权限时整块隐藏而不是空白
- S5 业务域详情：BizScopeDetail 增加应用/服务/实例统计与跳转
- S6 受限角色：数据范围之外的实例/主机不出现在关联视图，也不出现在计数里；补接口级与 smoke 级用例
- S7 证据：go test bizscope/cmdb/service；test:smoke:business:cmdb；1440px/390px 渲染证据

## Acceptance Criteria

- [ ] 主机详情可见其承载实例，且仅含数据范围内对象
- [ ] 业务域详情统计与服务目录一致
- [ ] bizscope 新增反例测试覆盖跨租户/跨业务域/并发删除并全绿
- [ ] 桌面与 390px 渲染证据齐备
- [ ] 主机与业务域详情新增区块符合 Base BUSINESS_LIFECYCLE_DETAIL_PATTERN（元信息块、子资源表、无权限整块隐藏）；渲染证据覆盖模板的页面状态变体与可达性要求

## Verification Plan

- `go test -count=1 ./modules/business/bizscope/... ./modules/business/cmdb/... ./modules/business/service/...`，工作目录 `backend`
- `npm run test:smoke:business:cmdb`，工作目录 `frontend`
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

- Task ID: 2026-11-25-ops-cmdb-asset-relation-views
- Task Manifest: .harness/tasks/2026-11-25-ops-cmdb-asset-relation-views/manifest.json
- Evidence Directory: .harness/evidence/2026-11-25-ops-cmdb-asset-relation-views/
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
