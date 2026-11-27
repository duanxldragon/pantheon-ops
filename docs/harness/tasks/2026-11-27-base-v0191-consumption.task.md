---
title: Consume official Pantheon Base v0.19.1 in Ops
doc_type: Remediation
layer: platform
linked_contracts:
  - docs/contracts/SYSTEM_AUTH_CONTRACT.md
status: Archived
updated_at: 2026-11-27
---

# Task Packet: 2026-11-27-base-v0191-consumption

## Goal

Consume the official Pantheon Base v0.19.1 Foundation release in Ops. Verify
the assembled candidate and the version locked on Ops main.

## Priority

P1, L2 inheritance sync.

## Primary Layer

business/*

## Dependency Layers

`system/auth`, inherited from Base through the release.

## Harness Profile

- Template: admin-platform
- Overlay: pantheon-ops
- Coverage Dimensions:
  - behaviour
  - architecture-fitness

## Workspace Context

- Target Repository: pantheon-ops
- Repository Role: business-consumer
- Upstream Dependencies: pantheon-base
- Downstream Consumers: none
- Sync Expectation: required
- Release Requirement: foundation-release

## Contract Anchors

- `docs/PROJECT_INHERITANCE.md`
- `docs/contracts/SYSTEM_AUTH_CONTRACT.md`
- `docs/designs/FOUNDATION_RELEASE_MODEL.md`

## Scope

### In

- Verify the four official assets and the immutable tag for v0.19.1.
- Rebuild and verify the consumer, then merge the version and shared authorization update.

### Out

- Ops business implementation, provider acceptance and deployment.
- Changes to workflows, assembler behavior, schema and permission catalogs.

## Expected Files

### Create

- This task packet, manifest and evidence.

### Modify

- The Foundation lock, product VERSION, assembly report and English and Chinese inheritance guides.
- Shared authorization implementation, tests and contract from the verified assembly.

### Do Not Touch

- Ops business sources, migrations and overlay manifest.
- Existing development data and working trees.

## Implementation Notes

- Minimal Complexity Rung: reuse the official release fetcher and overlay assembler.
- Owner Layer: Foundation consumer. Base owns the shared authorization implementation.
- Quality Profile: auth-security.
- Required Evidence: release hashes, assembly comparison, local verification and hosted checks.
- Ratchet Decision: no-repeat-observed.
- GitHub Signal: repo-quality-gate.

## Verification Plan

- npm run fetch:base-release -- --release pantheon-base-v0.19.1
- npm run rebuild:from-base -- --target ../.local/ops-foundation-0191/candidate-rebuild
- npm run check:business-overlay
- npm run check:shared-drift
- npm run test:business-overlay
- go test -race ./... with Windows CGO, isolated local MySQL and task-owned Redis
- npm run lint, npm run type-check, npm run build and npm run test:unit in frontend
- GitHub required checks, Smoke Sanity and post-merge lock and report verification

## Linkage

- Task ID: 2026-11-27-base-v0191-consumption
- Task Manifest: .harness/tasks/2026-11-27-base-v0191-consumption/manifest.json
- Evidence Directory: .harness/evidence/2026-11-27-base-v0191-consumption/
- Review File: .harness/evidence/2026-11-27-base-v0191-consumption/review.md
- Change Ref: none
- OpenSpec Change: none
- Superpowers Plan: none
- Plan References: none

## Evidence Required

See the [verification summary](../../../.harness/evidence/2026-11-27-base-v0191-consumption/summary.md).

## Human Gates

- Maintainer authorized the official Base release and Ops consumption.
- Independent reviewers assess the staged update before the consumption PR is merged.

## Completion Checklist

- [x] Layer and boundary declared
- [x] Contract anchors read
- [x] Verification run or exception recorded
- [x] Evidence saved or summarized
- [x] Review completed
