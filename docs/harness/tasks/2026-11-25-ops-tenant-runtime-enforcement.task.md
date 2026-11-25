---
title: Tenant runtime enforcement for business modules
doc_type: Remediation
layer: platform
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
status: Active
updated_at: 2026-11-25
---

# Task Packet: 2026-11-25-ops-tenant-runtime-enforcement

## Goal

Make `tenant_id` effective at runtime, one module at a time: read scoping, write ownership, tenant-scoped route groups with registered permission routes, and tenant identity carried into background work, with compat-mode behaviour unchanged.

## Priority

P1；L2，拆为按模块的后续任务。

## Primary Layer

business/*

## Dependency Layers

- business/*

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
- Release Requirement: none
- Downstream Sync Note: 共享缺陷先回 Base 修复。

## Contract Anchors

- `AGENTS.md`
- `docs/PROJECT_INHERITANCE.md`
- `docs/contracts/TENANT_CONTRACT_V1.md`

## Scope

### In

- Per-module enforcement in the order of `.harness/tasks/2026-11-25-ops-tenant-runtime-enforcement/PLAN.md`
- A shared two-tenant test fixture
- Rollout that keeps compat mode unchanged

### Out

- Base changes (shared defects go Base-first)
- Production operations
- Gate waivers

## Expected Files

### Create

- `docs/harness/tasks/2026-11-25-ops-tenant-runtime-enforcement.task.md`
- `.harness/tasks/2026-11-25-ops-tenant-runtime-enforcement/manifest.json`
- `.harness/tasks/2026-11-25-ops-tenant-runtime-enforcement/PLAN.md`

### Modify

- `backend/modules/business/**` (per module, in later tasks)

### Do Not Touch

- `backend/modules/system/**`
- `foundation-release.lock.json`

## Implementation Notes

- 本包是规划父包，status planned；不含代码。模块任务在各自 PR 中创建 manifest 与证据。
- 前置：CI 的 backend-tests 需要 MySQL 服务，否则双租户真实库测试在 CI 中全部 skip。
- Delivery Governance: L2

## Verification Plan

- 每个模块任务：真实 MySQL 双租户测试 + compat 模式测试 + `check:business-boundaries` + `check:business-overlay`

## Linkage

- Task ID: 2026-11-25-ops-tenant-runtime-enforcement
- OpenSpec Change: none
- Superpowers Plan: none
- Task Manifest: .harness/tasks/2026-11-25-ops-tenant-runtime-enforcement/manifest.json
- Evidence Directory: .harness/evidence/2026-11-25-ops-tenant-runtime-enforcement/
- Review File: none
- Change Ref: none
- Plan References: .harness/tasks/2026-11-25-ops-tenant-runtime-enforcement/PLAN.md

## Evidence Required

规划包无运行证据；各模块任务按其验证计划保存。

## Human Gates

- 维护者已于 2026-11-25 接受按模块拆分；每个模块 PR 仍需验收。

## Completion Checklist

- [x] Layer and boundary declared
- [x] Contract anchors read
- [ ] Verification run or exception recorded
- [ ] Evidence saved or summarized
- [ ] Review completed

## Status

planned。依赖 T14 第一阶段（PR #225）与 CI MySQL 服务。
