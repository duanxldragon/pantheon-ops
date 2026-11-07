---
title: Ops official Base v0.18.6 consumption
doc_type: Remediation
layer: platform
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
status: Active
updated_at: 2026-11-07
---

# Task Packet: 2026-11-07-base-v0186-consumption

## Goal

Consume only official Base v0.18.6 assets through the existing assembler; validate exact staged tree before replacement, PR and merge.

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
- Downstream Sync Note: official186 only

## Contract Anchors

- `../pantheon-base/docs/designs/FOUNDATION_RELEASE_MODEL.md`
- `docs/PROJECT_INHERITANCE.md`

## Scope

### In

- Official four-asset acquisition, exact lock and standard clean rebuild
- Preserve existing business assets and integration, unchanged assembler
- Readonly CGO full race, frontend, migration, overlay/drift and applicable smoke
- Independent CODE/ARCH review, required CI, merge and main-only consolidation

### Out

- Shared or business runtime implementation changes
- New dependencies, gate weakening, test overlays, post-assembly module mutation
- Production operations, S3 waiver, maintenance freeze signature

## Expected Files

### Create

- docs/harness/tasks/2026-11-07-base-v0186-consumption.task.md
- .harness/tasks/2026-11-07-base-v0186-consumption/manifest.json
- .harness/tasks/2026-11-07-base-v0186-consumption/plan.md
- .harness/evidence/2026-11-07-base-v0186-consumption/commands.json
- .harness/evidence/2026-11-07-base-v0186-consumption/summary.md
- .harness/evidence/2026-11-07-base-v0186-consumption/review.md

### Modify

- foundation-release.lock.json
- .business-overlay-report.json
- VERSION
- CHANGELOG.md
- docs/PROJECT_INHERITANCE.md
- docs/PROJECT_INHERITANCE.en.md
- Officially assembled inherited release/version documentation

### Do Not Touch

- Business runtime source
- Assembler scripts
- Original historical task criteria and evidence
- GitHub quality/security policies

## Implementation Notes

- Reuse official GitHub acquisition and existing assembler; no new source code/dependency/helper.
- Landing side: Ops consumes; shared implementation remains in Base.
- Quality Profile: ci-workflow
- Portable Failure Class: release-consumer-handoff
- Owner Layer: business/* / foundation-consumer
- Consumer-Specific Controls: exact lock and official verified snapshot
- Required Sensors: readonly CGO full race, frontend, runtime, overlay/drift and hosted CI
- Required Evidence: exact source/provenance/hash/results and two independent reviews
- Ratchet Decision: registry-only
- Delivery Governance: L2
- GitHub Signal: repo-quality-gate
- Deferred Code Issues: QA002/extra host-import and prior explicit residuals remain
- Implementer: Dispatcher operates standard adapters
- Reviewer: independent CODE and ARCH
- Minimum Complexity Rung: reuse
- Minimum Runtime Evidence: fresh isolated official assembled runtime and standard suites
- Stop points: no production, gate weakening or source override

## Verification Plan

- npm run fetch:base-release -- --release pantheon-base-v0.18.6
- npm run rebuild:from-base -- --target <isolated staging>
- npm run check:business-overlay
- npm run check:shared-drift
- npm run test:business-overlay
- go test -timeout=30m -buildvcs=false -mod=readonly -race -json -p 2 -count=1 ./...
- frontend lint/type-check/build/unit
- Fresh runtime migration plus standard generator smoke suites
- Required GitHub CI and exact source CODE/ARCH review

## Linkage

- Task ID: 2026-11-07-base-v0186-consumption
- OpenSpec Change: none
- Superpowers Plan: none
- Task Manifest: .harness/tasks/2026-11-07-base-v0186-consumption/manifest.json
- Evidence Directory: .harness/evidence/2026-11-07-base-v0186-consumption/
- Review File: .harness/evidence/2026-11-07-base-v0186-consumption/review.md
- Change Ref: none
- Plan References: .harness/tasks/2026-11-07-base-v0186-consumption/plan.md

## Evidence Required

实际执行命令、退出状态、隔离复现/失败反证、下游影响和显式gap。不伪造测试/独立review通过。

## Human Gates

- User authorized push, merge, safe archived branch cleanup, release and Ops consumption; gate weakening/production and freeze signature remain outside scope.

## Completion Checklist

- [x] Four official v0.18.6 assets verified against immutable release/source and consumer lock
- [x] Exact standard assembled source passes readonly CGO full race, frontend, overlay/drift and applicable smoke before replacement/PR/merge
- [x] Independent CODE/ARCH approval and required GitHub checks precede merge
- [x] Only main remains locally/remotely, previous work and rollback artifacts preserved
- [x] Layer and boundary declared
- [x] Contract anchors read
- [x] Verification run or exception recorded
- [x] Evidence saved or summarized
- [x] Review completed
