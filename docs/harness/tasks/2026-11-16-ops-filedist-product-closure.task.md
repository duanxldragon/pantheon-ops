---
title: 文件分发创建入口与可靠控制
doc_type: Remediation
layer: business
status: Archived
linked_contracts:
  - docs/PROJECT_INHERITANCE.md
updated_at: 2026-11-20
---

# Task Packet: 2026-11-16-ops-filedist-product-closure

## Goal

文件分发创建入口与可靠控制

## Priority

P1；L2；扩展条件队列，不计首期核心退出。

## Primary Layer

business/*

## Dependency Layers

- business/*
- inheritance-sync

## Harness Profile

- Template: admin-platform
- Overlay: pantheon-ops
- Coverage Dimensions:
  - behaviour
  - architecture-fitness
  - runtime-quality

## Workspace Context

- Target Repository: pantheon-ops
- Repository Role: business-consumer
- Upstream Dependencies: pantheon-base
- Downstream Consumers: none
- Sync Expectation: not-required
- Release Requirement: none
- Downstream Sync Note: 仅业务消费者；共享变更回 Base 正式 release 后再消费

## Contract Anchors

- `AGENTS.md`
- `docs/PROJECT_INHERITANCE.md`
- `docs/designs/BUSINESS_FUNCTIONAL_DESIGN.md`
- `docs/designs/BUSINESS_QA_ACCEPTANCE_PLAN.md`

## Scope

### In

- 增加浏览器创建任务、目标选择、分页和进度刷新，复用后端 CreateTask。
- 按已声明 scope 与 authorizer 能力接入，保留 nil fail-closed；all/admin 限制不能被 UI 偷绕开。
- 建立在途传输可取消、未来任务触发及重启收敛；出队取消保护已存在，不重复实现。

### Out

- platform/system/* 通用实现与规则副本；共享问题 Base-first
- 未授权生产操作、规则豁免与风险接受
- 自动 exporter/file_sd/商业 License 等后续产品扩张

## Expected Files

### Create

- docs/harness/tasks/2026-11-16-ops-filedist-product-closure.task.md
- .harness/tasks/2026-11-16-ops-filedist-product-closure/manifest.json
- .harness/evidence/2026-11-16-ops-filedist-product-closure/summary.md
- .harness/evidence/2026-11-16-ops-filedist-product-closure/review.md
- .harness/evidence/2026-11-16-ops-filedist-product-closure/commands.json

### Modify

- frontend/src/modules/business/filedist
- backend/modules/business/filedist
- frontend/tests/smoke/business/observability-filedist-pages.spec.ts

### Do Not Touch

- ../pantheon-base/**
- foundation-release.lock.json
- backend/modules/system/**
- frontend/src/modules/system/**

## Implementation Notes

- Owner: business/filedist
- Minimum Complexity Rung: reuse
- 开始实现前重新读取当前候选与结构；本审查行号绑定旧基线，不能覆盖并发升级。
- Windows race 必须 PATH 前置 D:\msys64\mingw64\bin，CGO_ENABLED=1；普通 Go tests 不替代。
- 业务 mutation 的 API/权限/菜单/i18n/迁移/fixture/smoke 同步关闭；复用共享契约，禁止本地 override。
- 默认 planned，不开放本扩展时不阻塞首期；选择开放时相关 P0 成为必过。

## Structural Scope

- Affected Subgraph: business/filedist
- Boundary Crossings: 业务 owner capability / HTTP UI / 外部 provider；治理任务只关联任务与证据
- Risk Nodes: 目标身份、ownership、状态回写、引用、发布与证据真实性
- Graph Focus: frontend/src/modules/business/filedist/FiledistList.tsx:99; backend/modules/business/filedist/distribution_service.go:253

## Execution Roles

- Implementer Posture: executor/Generator 在明确 owner 内实现；当前计划会话只改治理资料
- Reviewer Posture: 独立 code-reviewer/verifier 对照 In/Out 和运行证据；静态推断不得写成已复现

## Verification Plan

- go test -race -count=1 ./modules/business/filedist/...
- npm run test:smoke:business:observability

## Acceptance Criteria

- 用户从上传/选文件到选目标、创建、传输校验、进度与结果回读全程走通 UI。
- 大文件开始后取消能按合同停止在途写入；未来 scheduledAt 到期执行，重启后的任务进入确定状态。
- 已取消队列项不再执行；受限角色行为与后端实际开放范围一致。

## Dependencies

- none

## Linkage

- Task ID: 2026-11-16-ops-filedist-product-closure
- OpenSpec Change: none
- Superpowers Plan: none
- Task Manifest: .harness/tasks/2026-11-16-ops-filedist-product-closure/manifest.json
- Evidence Directory: .harness/evidence/2026-11-16-ops-filedist-product-closure/
- Review File: .harness/evidence/2026-11-16-ops-filedist-product-closure/review.md
- Change Ref: none
- Plan References: .harness/tasks/2026-11-16-ops-pilot-delivery/plan.md

## Evidence Required

- commands.json 的真实命令/cwd/退出状态；summary.md 区分 passed/failed/not-run/skip；review.md 独立结论。
- 业务及外部验收还需候选 revision、foundation、迁移、fixture/hash、正反向结果、截图或明确 gap。
- planned 任务不预写通过证据，不以父计划文档验证冒充子任务完成。

## Human Gates

- 范围已由维护者确认：按业务设计评估试点；安全可逆步骤自主执行。
- 生产操作、门禁豁免、风险接受及最终视觉/功能验收由维护者决定；不得以等候时间作为批准。

## Stop Points

- 生产/凭据授权缺失、破坏性数据操作、门禁豁免、关键合同冲突时停止依赖动作；继续独立可逆工作。

## Completion Checklist

- [x] Layer and boundary declared
- [x] Contract anchors read
- [ ] Verification run or exception recorded
- [ ] Evidence saved or summarized
- [ ] Review completed

## Status

in-progress；首期默认裁剪，启用时转为必验。

## 2026-11-18T07:31:44.466Z Continuation checkpoint

Status: in-progress. claim SQL失败已与正常竞争区分，保留 active item在远端IO前结算；单次DB写5s、等待1s可取消，总结算时长无固定上限。调度发现失败/恢复有脱敏日志。中央真实MySQL CGO/race：17顶层PASS/4.634s。Claude UI resume exit1/max_turns，写入2个UI文件和1个计划，由 native续收口。

Original acceptance and dependencies remain unchanged. Whole-package acceptance is incomplete; see [summary](../../../.harness/evidence/2026-11-16-ops-filedist-product-closure/summary.md) and [progress receipt](../../../.harness/evidence/2026-11-17-ops-history-audit/continuation-ten-packages-progress-20261118.json).

## 2026-11-18T07:58:38.569Z Integration continuation

Status: in-progress. startup/Stop、6个E01 canonical key和44个UI key已装配并提供17文件冻结handoff；中央filedist5.021s/opsinit1.115s窄MySQL CGO/race通过，748源运行内稳定。UI historical-final02为14PASS/1.74s及app type/lint通过，但首轮独立REQUEST CHANGES发现两MEDIUM，正在修detail sibling overlap/list旧closure及404/forbidden fixture、first100 picker；新UI尚未冻结。

Backend repaired scope保留COMMENT；UI第一轮REQUEST CHANGES：E01-UI-R1/E01-UI-R2两MEDIUM，原final02哈希和结果保留，不以14PASS覆盖审查。 Startup/Helm独立窄CLEAR、整包WATCH，17＋4绑定hash核验；Base/public authority/runtime边界保留。 Original acceptance, dependencies and human gates remain unmet. See [summary](../../../.harness/evidence/2026-11-16-ops-filedist-product-closure/summary.md) and [latest progress](../../../.harness/evidence/2026-11-17-ops-history-audit/continuation-ten-packages-progress-integration-20261118.json).

## 2026-11-19T00:11:15.272Z Verified continuation checkpoint

Status: in-progress. startup/Stop及6canonical＋44UI keys/5locale装配已通过中央窄race；此前两项MEDIUM修后获9文件UI round2独立COMMENT，该评审仅覆盖其绑定候选。此前独立canonical28PASS/4.21s和同28重复fixture不累计56；当前集中canonical为35PASS/8.68s。两项真实UI修复获独立incremental COMMENT/0issues；新e01-final stage/1242源、frontend35PASS/build/runtime通过。actual04 exit0/sourceStable，21图及桌面/手机footer入屏证据由上一轮协调者核验。本轮只复核保存证据与当前源码哈希，未新增浏览器执行或视觉审查。

旧round1 REQUEST CHANGES、round2 COMMENT及新两文件incremental COMMENT/0issues均保留各自绑定范围。actual04及上一轮21图查看关闭本地footer证据缺口；真实provider/受限角色和整包验收仍开放。 Original acceptance and human gates remain unmet. See [summary](../../../.harness/evidence/2026-11-16-ops-filedist-product-closure/summary.md) and [structured progress](../../../.harness/evidence/2026-11-17-ops-history-audit/continuation-ten-packages-progress-final-20261118.json).

## Result — History Closeout 2026-11-20

Manifest disposition: completed

FileDist 产品化收口 + 71 证据文件随 #195 落地；真实 SSH/结构和外部 provider 验收保留为 residual。

Residual finding/owner/successor/proof: .harness/evidence/2026-11-20-ops-board-complete/dispositions.json. Canonical references remain archive-aware. Historical body/evidence retained. Candidate/production/maintainer gates remain separate.

## Result — History Closeout 2026-11-20

Manifest disposition: completed

FileDist 产品化收口 + 71 证据文件随 #195 落地；真实 SSH/结构和外部 provider 验收保留为 residual。

Residual finding/owner/successor/proof: .harness/evidence/2026-11-20-ops-board-complete/dispositions.json. Canonical references remain archive-aware. Historical body/evidence retained. Candidate/production/maintainer gates remain separate.
