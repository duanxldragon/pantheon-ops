---
title: Ops official Base v0.18.7 consumption
doc_type: Remediation
layer: platform
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
status: Active
updated_at: 2026-11-13
---

# Task Packet: 2026-11-13-base-v0187-consumption

## Goal

Consume only official Base v0.18.7 assets through the existing assembler; validate the exact staged tree before replacement, PR and merge.

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
- Downstream Sync Note: official v0.18.7 only

## Contract Anchors

- `docs/PROJECT_INHERITANCE.md`
- `docs/contracts/PLATFORM_CONTRACT.md`
- `docs/designs/FOUNDATION_RELEASE_MODEL.md`

## Scope

### In

- Official four-asset acquisition of `pantheon-base-v0.18.7`, exact lock and standard clean rebuild
- Preservation of existing business assets and integration; unchanged assembler
- Readonly CGO full race, frontend, migration, overlay/drift and applicable smoke on the staged tree
- Independent CODE/ARCH review, required CI, merge and main-only consolidation

### Out

- Shared or business runtime implementation changes
- New dependencies, gate weakening, test overlays, post-assembly module mutation
- Production operations, S3 waiver, maintenance freeze signature

## Expected Files

### Create

- docs/harness/tasks/2026-11-13-base-v0187-consumption.task.md
- .harness/tasks/2026-11-13-base-v0187-consumption/manifest.json
- .harness/tasks/2026-11-13-base-v0187-consumption/plan.md
- .harness/evidence/2026-11-13-base-v0187-consumption/commands.json
- .harness/evidence/2026-11-13-base-v0187-consumption/summary.md
- .harness/evidence/2026-11-13-base-v0187-consumption/review.md

### Modify

- foundation-release.lock.json
- .business-overlay-report.json
- VERSION
- docs/PROJECT_INHERITANCE.md
- docs/PROJECT_INHERITANCE.en.md
- Officially assembled inherited release/version documentation
- Base-owned shared paths updated by the v0.18.6→v0.18.7 delta (backend middleware/permissionroutes/database/authtoken and shared frontend smoke helper)
- scripts/frontmatter-check.mjs aligned to the immutable v0.18.7 snapshot copy (required because the v0.18.7 prettier-formatted contract docs plus the already-synced checker test expect the blank-line-tolerant parser)
- frontend/src/modules/generated/system.ts restored to snapshot bytes (the materialization pass had dropped the generated DO-NOT-EDIT header; surfaced by round-1 CODE/ARCH review)

### Do Not Touch

- Business runtime source
- Assembler scripts
- Original historical task criteria and evidence
- GitHub quality/security policies

## Implementation Notes

- Reuse official GitHub acquisition and existing assembler; no new source code/dependency/helper.
- Landing side: Ops consumes; shared implementation remains in Base.
- Unlike the byte-identical v0.18.6 consumption, this delta carries real Base shared-path changes (PR #504 contracts, #505/#506/#507 remediations), so the staged tree must materialize into the consumer PR.
- Provider-owned permission routes must be registered before the startup seal; revocation is authoritative owning-tenant only.
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

- npm run fetch:base-release -- --release pantheon-base-v0.18.7
- npm run rebuild:from-base -- --target <isolated staging>
- npm run check:business-overlay
- npm run check:shared-drift
- npm run test:business-overlay
- go test -timeout=45m -buildvcs=false -mod=readonly -race -p 2 -count=1 ./...
- frontend lint/type-check/build/unit
- Fresh runtime migration plus standard generator smoke suites
- Required GitHub CI and exact source CODE/ARCH review

## Linkage

- Task ID: 2026-11-13-base-v0187-consumption
- OpenSpec Change: none
- Superpowers Plan: none
- Task Manifest: .harness/tasks/2026-11-13-base-v0187-consumption/manifest.json
- Evidence Directory: .harness/evidence/2026-11-13-base-v0187-consumption/
- Review File: .harness/evidence/2026-11-13-base-v0187-consumption/review.md
- Change Ref: none
- Plan References: .harness/tasks/2026-11-13-base-v0187-consumption/plan.md
- Predecessor Task: 2026-11-12-v0187-release-ops-consumption (Base-side publication)

## Evidence Required

实际执行命令、退出状态与日志、锁文件/资产校验、staged 树验证结果、消费 PR checks 与显式 gap。不伪造测试、独立 review、发布或消费通过。

## Human Gates

- Maintainer-authorized scope derives from the 2026-11-07 review closeout instruction (official release + Ops consumption); gate weakening, production operations and freeze signature remain outside scope.

## Completion Checklist

- [x] Layer and boundary declared
- [x] Contract anchors read
- [ ] Four official v0.18.7 assets verified against immutable release/source and consumer lock
- [ ] Verification run or exception recorded
- [ ] Evidence saved or summarized
- [ ] Review completed
- [ ] Exact standard assembled source passes readonly CGO full race, frontend, overlay/drift and applicable smoke before replacement/PR/merge
- [ ] Independent CODE/ARCH approval and required GitHub checks precede merge

## Status

in progress; official acquisition and staging gates passed, full race/frontend suites running; no consumption is claimed until the exact assembled source passes all sensors and hosted checks.
