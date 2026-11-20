---
title: Ops 历史遗留清查与按时间收尾
doc_type: Remediation
layer: business
status: Archived
linked_contracts:
  - docs/PROJECT_INHERITANCE.md
updated_at: 2026-11-20
---

# Task Packet: 2026-11-17-ops-history-audit

## Goal

全面清查Ops历史遗留并按时间完成有效工作；已完成或失效任务凭真实证据收尾，不重复旧版本实现，不把生产/外部/维护者验收改为完成。

## Primary Layer

business/*

## Dependency Layers

- inheritance-sync
- system/config 的公共扩展点由Base拥有，先在Base修再正式消费。

## Harness Profile

- Template: custom
- Overlay: pantheon-ops
- Coverage Dimensions:
  - behaviour
  - architecture-fitness
  - runtime-quality
  - method-health

## Workspace Context

- Target Repository: pantheon-ops
- Repository Role: business-consumer
- Upstream Dependencies: pantheon-base
- Downstream Consumers: none
- Sync Expectation: deferred
- Release Requirement: none
- Downstream Sync Note: 共享修复回Base；正式release和当前消费锁按独立任务验收，不手抄共享文件。

## Contract Anchors

- `AGENTS.md`
- `docs/PROJECT_INHERITANCE.md`
- `docs/TASK_PACKET_OPS_TEMPLATE.md`
- `business-overlay.json`
- `foundation-release.lock.json`

## Scope

### In

- 84条旧记录按原范围对账；证据充分的完成/替代任务归档并保全SHA。
- 历史SSH、权限、工作台与自动合并的有效缺口；当前治理失败收敛。
- C01-C06/E01-E04按依赖实现与同候选验证，不以旧实现替代新验收。
- 状态、索引、overlay和证据链同步；最终清理和双角色独立审查。

### Out

- 未授权生产操作、凭证轮换、代维护者/DBA签署。
- 手抄平台/系统域实现，变更正式foundation lock，门禁豁免、发布或合并。
- 对缺证据、缺外部环境、未完成实现作完成声明。

## Expected Files

### Create

- `.harness/tasks/2026-11-17-ops-history-audit/manifest.json`
- `.harness/evidence/2026-11-17-ops-history-audit/commands.json`
- `.harness/evidence/2026-11-17-ops-history-audit/summary.md`
- `.harness/evidence/2026-11-17-ops-history-audit/review.md`

### Modify

- `.harness/STATUS.md`
- `.harness/ARCHIVE.md`
- `.harness/tasks/2026-11-16-ops-pilot-delivery/OPEN_TASKS.md`
- `business-overlay.json`
- 子包明确拥有的Ops业务与治理文件。

### Do Not Touch

- `foundation-release.lock.json`
- 已发布Base资产、历史summary/review/commands原始字节。
- 其他执行者未提交改动不得回滚。

## Implementation Notes

当前维护者授权Codex处理高风险，Claude可处理低风险；覆盖旧计划的“Codex只派发”说明。保持小步、范围清楚、证据可复跑。历史归档18组有独立批准；中断恢复按journal原始哈希继续，后台进程用隐藏窗口。

## Verification Plan

- 真实归档哈希与同ID引用、三处故障注入恢复、任务卡状态与链接。
- task/evidence/review/frontmatter/template/overlay/boundary门禁；原红灯保留，不能通过删除有效测试隐藏失败。
- backend完整Windows CGO race（D:\msys64\mingw64\bin），要求真实隔离DSN；frontend lint/type-check/build及适用业务烟测、桌面/窄屏渲染。
- 当前源码重建和最终候选运行；目标、fixture、迁移、包SHA和日志必须对应同一候选。

## Linkage

- Task ID: 2026-11-17-ops-history-audit
- Task Manifest: .harness/tasks/2026-11-17-ops-history-audit/manifest.json
- Evidence Directory: .harness/evidence/2026-11-17-ops-history-audit/
- Review File: .harness/evidence/2026-11-17-ops-history-audit/review.md
- OpenSpec Change: none
- Change Ref: none
- Plan References: .harness/tasks/2026-11-17-ops-history-audit/brief.md
- OMX Plan: .omx/ultragoal/goals.json
- Superpowers Plan: none

## Evidence Required

- dispositions.json及逐项finding/owner/successor/proof。
- archive preflight/journal/result与原始文件哈希、独立角色审查。
- commands.json真实cwd/status和knownGaps；运行与渲染只以当前证据认定。

## Human Gates

需求范围已确认。生产窗口、凭证、门禁策略、最终维护者功能/视觉验收保持开放；本地安全可逆工作自主执行。

## Completion Checklist

- [x] Layer and boundary declared
- [x] Contract anchors read
- [ ] Verification run or exception recorded
- [x] Evidence saved or summarized
- [ ] Review completed
