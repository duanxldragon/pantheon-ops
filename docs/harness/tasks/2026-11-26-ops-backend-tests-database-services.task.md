---
title: Backend Tests gate with MySQL and Redis
doc_type: Remediation
layer: platform
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
status: Archived
updated_at: 2026-11-26
---

# Task Packet: 2026-11-26-ops-backend-tests-database-services

## Goal

Make the required Backend Tests job run the testmysql and testredis tests instead of silently skipping them, and fail if the DSN or Redis address goes missing.

## Priority

P1；L2（门禁策略变更）。

## Primary Layer

platform

## Dependency Layers

- business/*

## Harness Profile

- Template: custom
- Overlay: pantheon-ops
- Quality Profile: ci-workflow
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
- Downstream Sync Note: 镜像 Base quality.yml 的 backend-tests 写法；仅改 Ops 工作流。

## Contract Anchors

- `AGENTS.md`
- `docs/PROJECT_INHERITANCE.md`

## Scope

### In

- Add MySQL (UTC) and a generic Redis service to backend-tests
- Set PANTHEON_TEST_DSN, PANTHEON_TEST_DSN_REQUIRED, PANTHEON_TEST_REDIS_ADDR, PANTHEON_TEST_REDIS_REQUIRED
- A workflow test that pins this so it cannot silently regress

### Out

- Fixing product tests
- Making the job slower than needed
- Changing which jobs are required

## Expected Files

### Create

- `docs/harness/tasks/2026-11-26-ops-backend-tests-database-services.task.md`
- `.harness/tasks/2026-11-26-ops-backend-tests-database-services/manifest.json`
- `.harness/evidence/2026-11-26-ops-backend-tests-database-services/commands.json`
- `.harness/evidence/2026-11-26-ops-backend-tests-database-services/summary.md`
- `.harness/evidence/2026-11-26-ops-backend-tests-database-services/review.md`

### Modify

- `.github/workflows/quality.yml`
- `tests/scripts/quality-workflow.test.mjs`

### Do Not Touch

- `backend/modules/system/**`
- The required-check list

## Implementation Notes

- Minimal Complexity Rung: reuse（照抄 Base quality.yml 的 backend-tests）
- 服务器时区必须是 UTC：DSN 写了 `loc=UTC`，而代码在 SQL 里用 `CURRENT_TIMESTAMP(6)` 比较 Go 写入的租约时间；时区不一致会让租约被判过期。
- Quality Profile: ci-workflow
- Portable Failure Class: silent-test-skip
- Ratchet Decision: sensor-added
- Delivery Governance: L2

## Verification Plan

- node --test tests/scripts/quality-workflow.test.mjs
- Hosted Backend Tests on this PR

## Linkage

- Task ID: 2026-11-26-ops-backend-tests-database-services
- OpenSpec Change: none
- Superpowers Plan: none
- Task Manifest: .harness/tasks/2026-11-26-ops-backend-tests-database-services/manifest.json
- Evidence Directory: .harness/evidence/2026-11-26-ops-backend-tests-database-services/
- Review File: .harness/evidence/2026-11-26-ops-backend-tests-database-services/review.md
- Change Ref: none
- Plan References: none

## Evidence Required

实际命令与退出状态、本机预测结果、托管 CI 结果；不伪造独立 review。

## Human Gates

- 门禁策略变更：维护者于 2026-11-26 授权给 CI 补 MySQL（回复写作"需要不mysql"，按"补"理解）。

## Completion Checklist

- [x] Layer and boundary declared
- [x] Contract anchors read
- [x] Verification run or exception recorded
- [x] Evidence saved or summarized
- [x] Review completed

## Status

completed：PR #227 的托管 Backend Tests 与合并后 main 验证均通过。Redis/UTC 回归断言已在独立 worktree 补强，14 项工作流测试和独立复审均通过。补强与收口记录待合入，Windows 完整 CGO race 通过（77 个测试包）。
