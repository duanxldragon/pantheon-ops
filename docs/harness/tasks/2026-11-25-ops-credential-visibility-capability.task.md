---
title: Credential visibility through the BizScopeReader capability
doc_type: Remediation
layer: platform
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
status: Active
updated_at: 2026-11-25
---

# Task Packet: 2026-11-25-ops-credential-visibility-capability

## Goal

Remove the direct `bizscope.BizScope` access from deploy credential visibility, which the manual-only business boundary check flagged after T05, without changing which credentials a scoped role can see.

## Priority

P1；L1 轻量闭环。

## Primary Layer

business/*

## Dependency Layers

- business/*

## Harness Profile

- Template: custom
- Overlay: pantheon-ops
- Quality Profile: auth-security
- Delivery Tier: L1
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
- Downstream Sync Note: 仅 Ops 业务模块内部契约；不涉及 Base。

## Contract Anchors

- `AGENTS.md`
- `docs/PROJECT_INHERITANCE.md`
- `docs/designs/BUSINESS_DEPLOY_MODULE_DESIGN.md`

## Scope

### In

- Add `BizScopeReader.VisibleIDs` and implement it in bizscope
- Use it from deploy `credentialVisibleQuery`, failing closed on error or a missing reader
- Update the four test fakes

### Out

- Any change to which credentials are visible
- Phase 2 tenant enforcement
- CI workflow changes

## Expected Files

### Create

- `docs/harness/tasks/2026-11-25-ops-credential-visibility-capability.task.md`
- `.harness/tasks/2026-11-25-ops-credential-visibility-capability/manifest.json`
- `.harness/evidence/2026-11-25-ops-credential-visibility-capability/commands.json`
- `.harness/evidence/2026-11-25-ops-credential-visibility-capability/summary.md`
- `.harness/evidence/2026-11-25-ops-credential-visibility-capability/review.md`

### Modify

- `backend/modules/business/capability/capability.go`
- `backend/modules/business/bizscope/bizscope_service.go`
- `backend/modules/business/deploy/deploy_credential.go`
- four test fakes implementing BizScopeReader

### Do Not Touch

- `backend/modules/system/**`
- GitHub quality/security policies

## Implementation Notes

- Minimal Complexity Rung: reuse（沿用既有 BizScopeReader 契约，只加一个方法）
- `check:business-boundaries` 在 CI 中是手动项，因此 T05 合并时必需检查为绿但边界已被违反；是否接入 CI 属门禁策略，留给维护者。

## Verification Plan

- npm run check:business-boundaries
- go test -run TestDeployCredential ./modules/business/deploy/ （真实 MySQL，PANTHEON_TEST_DSN）
- 负例对照：VisibleIDs 返回空时列表测试应失败

## Linkage

- Task ID: 2026-11-25-ops-credential-visibility-capability
- OpenSpec Change: none
- Superpowers Plan: none
- Task Manifest: .harness/tasks/2026-11-25-ops-credential-visibility-capability/manifest.json
- Evidence Directory: .harness/evidence/2026-11-25-ops-credential-visibility-capability/
- Review File: .harness/evidence/2026-11-25-ops-credential-visibility-capability/review.md
- Change Ref: none
- Plan References: none

## Evidence Required

实际执行命令、退出状态、真实库测试结果与负例对照；不伪造独立 review。

## Human Gates

- 无新增；最终验收仍归维护者。

## Completion Checklist

- [x] Layer and boundary declared
- [x] Contract anchors read
- [x] Verification run or exception recorded
- [x] Evidence saved or summarized
- [ ] Review completed (independent review not performed; recorded as a gap)

## Status

边界违规已消除，凭据可见性行为经真实 MySQL 测试确认不变。见 `.harness/evidence/2026-11-25-ops-credential-visibility-capability/summary.md`。
