---
title: Ops official Base v0.19.0 consumption
doc_type: Remediation
layer: platform
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
status: Active
updated_at: 2026-11-25
---

# Task Packet: 2026-11-25-base-v0190-consumption

## Goal

Consume only the official Base v0.19.0 assets through the existing assembler; validate the exact staged tree before the consumer PR.

## Priority

P1；L2 发行与继承协调。

## Primary Layer

business/*

## Dependency Layers

- inheritance-sync

## Harness Profile

- Template: admin-platform
- Overlay: pantheon-ops
- Coverage Dimensions:
  - behaviour
  - maintainability

## Workspace Context

- Target Repository: pantheon-ops
- Repository Role: business-consumer
- Upstream Dependencies: pantheon-base
- Downstream Consumers: none
- Sync Expectation: required
- Release Requirement: foundation-release
- Downstream Sync Note: official v0.19.0 only

## Contract Anchors

- `docs/PROJECT_INHERITANCE.md`
- `docs/contracts/PLATFORM_CONTRACT.md`
- `docs/designs/FOUNDATION_RELEASE_MODEL.md`

## Scope

### In

- Official four-asset verification of `pantheon-base-v0.19.0`, exact lock and standard clean rebuild
- Keep the ops-owned inputs the rebuild would otherwise lose (SecretTextArea location, k8s namespace locale source, overlay manifest declarations, ops-owned test fixture)
- Backend build/vet/test, frontend lint/type-check/build/unit, overlay/drift and governance checks on the staged tree

### Out

- Business runtime behaviour changes
- New dependencies, gate weakening, assembler changes
- Production operations, S3 waiver, maintenance freeze signature

## Expected Files

### Create

- docs/harness/tasks/2026-11-25-base-v0190-consumption.task.md
- .harness/tasks/2026-11-25-base-v0190-consumption/manifest.json
- .harness/evidence/2026-11-25-base-v0190-consumption/commands.json
- .harness/evidence/2026-11-25-base-v0190-consumption/summary.md
- .harness/evidence/2026-11-25-base-v0190-consumption/review.md
- frontend/src/modules/business/k8s/locales/en-US.json
- frontend/src/modules/business/k8s/locales/zh-CN.json

### Modify

- foundation-release.lock.json
- .business-overlay-report.json
- business-overlay.json
- VERSION
- SHELL_VERSION.json
- docs/PROJECT_INHERITANCE.md
- docs/PROJECT_INHERITANCE.en.md
- tests/scripts/check-inheritance-contract.test.mjs
- Base-owned shared paths updated by the v0.18.9 to v0.19.0 delta

### Do Not Touch

- Assembler scripts
- Business runtime source beyond the SecretTextArea import path
- GitHub quality/security policies

## Implementation Notes

- Reuse official GitHub acquisition and the existing assembler; no new dependency or helper.
- The v0.18.9 consumption was lock-only, so this is the first full materialization since; it surfaced ops inputs outside declared overlay paths.
- Quality Profile: ci-workflow
- Portable Failure Class: release-consumer-handoff
- Owner Layer: business/* / foundation-consumer
- Required Evidence: lock/asset digests, staged-tree results, explicit gaps
- Ratchet Decision: registry-only
- Delivery Governance: L2
- GitHub Signal: repo-quality-gate
- Minimum Complexity Rung: reuse

## Verification Plan

- npm run fetch:base-release -- --release pantheon-base-v0.19.0
- npm run rebuild:from-base -- --target .tmp/clean-base-overlay
- npm run check:business-overlay
- npm run check:shared-drift
- go build / go vet / go test on the staged backend
- frontend lint/type-check/build/unit on the staged tree
- Required GitHub CI (Linux race, smoke)

## Linkage

- Task ID: 2026-11-25-base-v0190-consumption
- OpenSpec Change: none
- Superpowers Plan: none
- Task Manifest: .harness/tasks/2026-11-25-base-v0190-consumption/manifest.json
- Evidence Directory: .harness/evidence/2026-11-25-base-v0190-consumption/
- Review File: .harness/evidence/2026-11-25-base-v0190-consumption/review.md
- Change Ref: none
- Plan References: none

## Evidence Required

实际执行命令、退出状态、锁文件与资产校验、staged 树验证结果、PR checks 与显式 gap。不伪造测试、独立 review 或消费通过。

## Human Gates

- 最终功能验收与合并由维护者决定；不放宽门禁，不涉及生产操作。

## Completion Checklist

- [x] Layer and boundary declared
- [x] Contract anchors read
- [x] Four official v0.19.0 assets verified against the published digest and consumer lock
- [x] Verification run or exception recorded
- [x] Evidence saved or summarized
- [ ] Independent CODE/ARCH review (not performed; recorded as a gap)
- [ ] Hosted required checks pass before merge

## Status

assembled and locally verified; consumer PR and hosted required checks pending. No consumption is claimed until the PR merges.
