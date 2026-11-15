---
title: Consume latest official Base and execute remaining Ops plans
doc_type: Remediation
layer: business
status: Active
updated_at: 2026-11-14
linked_contracts:
  - docs/PROJECT_INHERITANCE.md
---

# Task Packet: 2026-11-14-ops-latest-foundation-and-plans

## Goal

Consume GitHub's latest stable immutable foundation release and finish executable Ops planning work with real verification.

## Primary Layer

business/*

## Workspace Context

- Target Repository: `pantheon-ops`
- Repository Role: `business-consumer`
- Upstream Dependencies: `pantheon-base`, `pantheon-harness`
- Downstream Consumers: `none`
- Sync Expectation: `required`
- Release Requirement: `consumer-lock-update`

## Dependency Layers

Released platform, authentication, IAM and business extension contracts.

## Harness Profile

- Template: custom
- Overlay: pantheon-ops
- Quality Profile: runtime-quality
- Portable Failure Class: consumer-release-compatibility
- Owner Layer: consumer-repository
- Coverage Dimensions:
  - behaviour
  - architecture-fitness
  - runtime-quality

## Scope

### In

- Formal release acquisition/rebuild, business menu/environment, bounded recorded-state operational widgets, existing implementation verification and current status reconciliation.
- Current authorized continuation: finish active O1–O6 and D1–D3/D5 implementation and executable acceptance, then reassemble the declared business overlay and rerun source-bound checks. See `.harness/tasks/2026-11-14-ops-latest-foundation-and-plans/current-execution-plan.md` for ownership, order and unchanged production/human gates.

### Out

- Production operations, source-directory Git comparisons, shared overrides, historical fixture replay, gate waivers and explicitly deferred O7.

## Expected Files

### Create

Business environment sources/migration000038, three replacement service pages, dashboard provider helpers and tests, task/evidence/review files.

### Modify

Business module declarations/seeds, role bindings, business-overlay.json, release lock, business smoke and inheritance/status documentation. Shared files are replaced through the standard official rebuild only.

### Do Not Touch

Original databases/preview services, production, credentials, previous failed acceptance fixtures and unapproved shared source overrides.

## Implementation Notes

User authorized planned execution on2026-11-14. The maintainer-approved scope shapes in the2026-11-13 packet apply; current versioned migration startup replaces its obsolete AutoMigrate assumption. Use independent review and actual source-bound validation.

## Contract Anchors

- `AGENTS.md`
- `docs/PROJECT_INHERITANCE.md`
- `.harness/tasks/2026-11-13-ops-business-menu-ia-refactor/plan.md`
- `.harness/tasks/2026-11-07-ops-planned-task-closeout/remaining-priorities.md`

## Minimum Viable Approach

Verify official assets, rebuild complete Base plus declared business overlay, preserve existing dirty inputs, use released APIs/components and validate staging before reversible working-tree replacement.

## Verification Plan

- `npm run test:business-overlay` and `npm run check:business-overlay`.
- `npm run rebuild:from-base` and `npm run check:shared-drift` against the immutable formal release candidate; cached replays explicitly record no fresh GitHub lookup.
- `CGO_ENABLED=1 go test -race -p=1 -buildvcs=false -mod=readonly -json ./...` with isolated MySQL/Redis and MSYS GCC.
- Frontend `npm run lint`, `npm run type-check`, `npm run build` and `npm run test:unit -- --run --maxWorkers=2`.
- Actual isolated business smoke and desktop1440/mobile390 screenshots; native runner and full visual/state gaps remain explicit.
- Scoped task/evidence/review/runtime/visual checks and current whole-repository documentation/review scan.

## Success Criteria

Passing verification proves the stated delivery scope; unexecuted external scenarios and unsigned human/production gates remain explicit.

## Linkage

- Task ID: 2026-11-14-ops-latest-foundation-and-plans
- Task Manifest: .harness/tasks/2026-11-14-ops-latest-foundation-and-plans/manifest.json
- Superpowers Plan: none
- Plan References: .harness/tasks/2026-11-13-ops-business-menu-ia-refactor/plan.md
- Evidence Directory: .harness/evidence/2026-11-14-ops-latest-foundation-and-plans/
- Review File: .harness/evidence/2026-11-14-ops-latest-foundation-and-plans/review.md
- Change Reference: none
- OpenSpec Change: none

## Evidence Required

Exact verification commands/outcomes, stable-source hashes, official identity/checksums, native race results, frontend results, runtime/seed and rendered evidence with explicit gaps, independent review.

## Human Gates

Final visual/functional acceptance remains the maintainer's responsibility. Production policy/signature and a new upstream publication are not inferred from this task request.

## Completion Checklist

- [x] Layer and boundary declared
- [x] Contract anchors read
- [x] Verification run or exception recorded
- [x] Evidence saved or summarized
- [x] Review completed
- [x] Official release fetched and checked.
- [x] Declared overlay rebuilt and checked.
- [x] Stable-source complete backend/frontend checks recorded; final full race uses63 cached/7 fresh packages and skips6 external tests.
- [x] Fresh local runtime and scoped rendered evidence recorded; full visual matrix remains open.
- [x] Independent final review recorded and linked; verdict remains blocked on adoption and complete acceptance.
- [x] Parent tasks and unsigned acceptance gaps explicitly reconciled.
- [x] Active v0.18.8 installation completed through official release assets; 165 writes, zero deletes, verified backup and unchanged Git index.
- [ ] Final visual/functional human acceptance and complete parent criteria met.

## Current Execution

Official v0.18.8 is now installed in the active tree at `34428814a9efc997fd93d4434406a53df6460323`. The r7 fresh-fetch/standard-assembler candidate passed contract, authoritative shared drift and source-identity checks; a newly reviewed 165-write plan preserved original dirty business inputs and all rollback evidence. Active overlay, standalone shared drift, Go readonly dependency/test loading, frontend lint/type-check/build and 414 units passed. Prior full CGO race and adapted browser evidence are reused only by complete relevant source hash identity; this is not a fresh active race or native smoke claim. Parent planning, provider and final visual/functional acceptance remain open. See verification/r7-active-installation.json.

### Earlier r5 checkpoint

The official v0.18.8 candidate is `.tmp/ops-v0188-final-r5`. Installation was denied by Windows directory permissions and fully rolled back; the active lock/report remain v0.18.6. Candidate product source is frozen; later root governance documentation updates are separate and require a newly reviewed preimage plan before any future installation. No shared-source override, permission change, historical fixture replay, commit, push or publication occurred.
