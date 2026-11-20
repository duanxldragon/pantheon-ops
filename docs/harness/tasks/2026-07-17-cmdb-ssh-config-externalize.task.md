---
title: CMDB SSH 采集配置外部化
doc_type: Remediation
layer: business
status: Archived
linked_contracts:
  - docs/PROJECT_INHERITANCE.md
  - docs/designs/BUSINESS_CMDB_MODULE_DESIGN.md
updated_at: 2026-11-20
---

# Task Packet: 2026-07-17-cmdb-ssh-config-externalize

> 历史卡片原文已字节保全于
> `.harness/evidence/2026-07-17-cmdb-ssh-config-externalize/original-task-card.md`。
> 本页由 2026-11-16 历史收尾批迁移至当前 Harness 模板，taskId 不变；原文事实不伪造。

## 目标

把 CMDB SSH 采集的 `cmdb.ssh.collect_timeout`（SSH 连接超时，秒，默认 `10`）与
`cmdb.ssh.default_port`（默认 SSH 端口，默认 `22`）从硬编码改为配置化：

- 业务模块注册配置 seed（归属 `business.cmdb`），默认 10 / 22。
- host collect 链路读取配置值，缺失时回落现有 10s / 22；主机有显式端口时优先主机值，默认端口仅用于无显式端口主机的拨号地址。
- 处理非法值（非数字 / 负数 / 超大 / 无效端口）并有界回落；不允许无界采集或修改全局共享默认。
- 配置租户/上下文与 Host 所属范围一致；配置读取失败不泄露凭证。
- 复用已暴露的 ConfigReader 能力；不直接 import `system/config` 内部 Service。

## 边界

- In：`backend/modules/business/cmdb/**`（含本卡片引用的 host 配置读取逻辑与回归测试）、
  `docs/designs/BUSINESS_CMDB_MODULE_DESIGN.md` §11.2 真实现状、
  本任务 manifest/plan 与同 taskId evidence 目录。
- Out：`system/config` 模块改动、其它业务模块、frontend、共享 backend/pkg/system/platform、
  正式 Base 锁版本、其它任务包的人条目。

## Why

- CMDB 设计 §11.2 原声称通过 `system/config` 管理，但 `host_service.go` 硬编码 `10s`、
  端口直接回落 `22`，无配置 seed。
- 配置外部化必须在 Base 暴露的配置读取能力（`GetByKey` 形状的 ConfigReader）之上完成，
  不得在 Ops 本地绕过缺失的共享层合同。

## 验证

- `go test ./backend/modules/business/cmdb/host`（含新增配置解析回归：配置生效、默认回落、
  显式 Host 端口优先、非法值、不同租户上下文；以及本批新增的超大溢出回落到上限、
  配置变更随注入 reader 下次 collect 生效、`Create` 持久化配置 default_port）。
- 本批曾因共享 Go 构建缓存 `D:\cache\go` 对依赖缓存条目 Access denied 阻断构建/测试，
  未将 not-run 写为 passed；Codex 已复核纯配置回归 PASS。
- 后续会话因权限无批准面无法执行 go 命令，本批新增测试再次交由 Codex 在可编译环境
  （真实 MySQL 隔离测试）统一复核；如实记 not-run，不伪造 hosted/liveSSH 证据。

## 关联

- [[2026-07-17-cmdb-ssh-config-externalize]]
- [[ops-business-menu-ia-refactor-2026-11-13]]

## Primary Layer

business/*

## Dependency Layers

- business/*
- foundation-config-capability

## Harness Profile

- Template: admin-platform
- Overlay: pantheon-ops
- Coverage Dimensions:
  - behaviour
  - architecture-fitness
  - runtime-quality
- Quality Profile: none

## Contract Anchors

- AGENTS.md
- docs/PROJECT_INHERITANCE.md
- ../pantheon-base/DESIGN.md
- docs/designs/BUSINESS_CMDB_MODULE_DESIGN.md

## Scope

### In

- CMDB host 配置解析逻辑（新 `host_config.go`）与 `Collect` 接线。
- 配置解析/边界回归测试（`host_config_test.go`）。
- §11.2 真实现状更新；任务卡迁移（原文已保全）。

### Out

- 生产接线（`opsinit` 装配点注入 tenant 作用域 ConfigReader）与业务配置 seed 注册属共享层
  /装配层 gap，需 Base/装配层最小修复后由 Codex 验收关闭；不在 Ops 绕过。

## Expected Files

### Create

- .harness/tasks/2026-07-17-cmdb-ssh-config-externalize/manifest.json
- .harness/evidence/2026-07-17-cmdb-ssh-config-externalize/summary.md
- .harness/evidence/2026-07-17-cmdb-ssh-config-externalize/commands.json
- .harness/evidence/2026-07-17-cmdb-ssh-config-externalize/original-task-card.md
- backend/modules/business/cmdb/host/host_config.go
- backend/modules/business/cmdb/host/host_config_test.go

### Modify

- backend/modules/business/cmdb/host/host_service.go
- docs/designs/BUSINESS_CMDB_MODULE_DESIGN.md
- docs/harness/tasks/2026-07-17-cmdb-ssh-config-externalize.task.md

### Do Not Touch

- 正式 foundation-release.lock.json
- 其它业务模块、frontend、共享 backend/pkg/system/platform

## Implementation Notes

Minimal Complexity Rung: reuse。复用已暴露的 `GetByKey` 形状 ConfigReader（与公开的
`tenant.SettingFlagReader` 一致），在 cmdb/host 内实现有界解析与回落；生产注入与 seed 注册
为独立共享层 gap，不本地绕过。

## Verification Plan

- 纯函数配置回归（无 DB、无网络）：配置生效 / 缺失回落 / 显式端口优先 / 非法值有界回落 /
  租户作用域隔离。
- 本会话因共享缓存 Access denied 无法执行构建/测试，命令与原始报错已记录为 not-run/blocked，
  交由 Codex 在可编译环境统一复核。

## Linkage

- Task ID: 2026-07-17-cmdb-ssh-config-externalize
- Task Manifest: .harness/tasks/2026-07-17-cmdb-ssh-config-externalize/manifest.json
- Evidence Directory: .harness/evidence/2026-07-17-cmdb-ssh-config-externalize/
- Review File: .harness/evidence/2026-07-17-cmdb-ssh-config-externalize/review.md
- OpenSpec Change: none
- Plan References: .harness/tasks/2026-11-16-ops-pilot-delivery/plan.md
- Superpowers Plan: none

## Evidence Required

- 新增配置解析回归源码；命令/结果经 commands.json 如实记录（含被阻断项）。
- 生产注入与 seed 注册 gap 作为待验收项，不宣称完整完成。

## Human Gates

任务保持 in-progress / awaiting-review；构建/测试复核与最终验收由独立 Codex 判断，
本执行者不自批准。

## Completion Checklist

- [x] Layer and boundary declared
- [x] Contract anchors read
- [x] Verification run or exception recorded（not-run/blocked 已在 evidence 如实记录并于本批补充说明）
- [x] Evidence saved or summarized
- [ ] Review completed（awaiting-review，Codex 验收关闭，本执行者不自批准）

## Result — History Closeout 2026-11-20

Manifest disposition: completed

CMDB host SSH 采集配置（collect_timeout/default_port）已外部化到 base ConfigReader，实施随 PR #195 落地 main；host_config.go 与测试在仓内，evidence summary+commands 已保全。原范围完成，无人工门禁。

Residual finding/owner/successor/proof: .harness/evidence/2026-11-20-ops-board-complete/dispositions.json. Canonical references remain archive-aware. Historical body/evidence retained. Candidate/production/maintainer gates remain separate.

## Result — History Closeout 2026-11-20

Manifest disposition: completed

CMDB host SSH 采集配置（collect_timeout/default_port）已外部化到 base ConfigReader，实施随 PR #195 落地 main；host_config.go 与测试在仓内，evidence summary+commands 已保全。原范围完成，无人工门禁。

Residual finding/owner/successor/proof: .harness/evidence/2026-11-20-ops-board-complete/dispositions.json. Canonical references remain archive-aware. Historical body/evidence retained. Candidate/production/maintainer gates remain separate.
