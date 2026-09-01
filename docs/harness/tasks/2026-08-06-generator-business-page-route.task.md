---
title: Restore generated business page routes and release tooling
doc_type: Remediation
layer: platform
status: Active
linked_contracts:
  - docs/designs/LOWCODE_GENERATOR_GUIDE.md
  - docs/designs/GENERATOR_MODULE_DESIGN.md
  - docs/designs/FOUNDATION_RELEASE_MODEL.md
updated_at: 2026-08-06
---

# Task Packet: 2026-08-06-generator-business-page-route

## Goal

Restore generated business pages and inferred menus to `/business/*`, publish the generator exporter tooling required by consumers, and deliver the fix through the next foundation patch release.

## Primary Layer

platform

## Dependency Layers

- shared lowcode generator
- generated `business/*` contracts
- dynamic module governance
- foundation release producer and consumer

## Harness Profile

- Template: admin-platform
- Overlay: pantheon-base
- Quality Profile: generator, ui-runtime, foundation-release
- Portable Failure Class: generated-business-route-drift
- Owner Layer: foundation-repository
- Coverage Dimensions:
  - behaviour
  - maintainability
  - architecture-fitness
  - runtime-quality
  - method-health

## Contract Anchors

- `AGENTS.md`
- `DESIGN.md`
- `docs/designs/LOWCODE_GENERATOR_GUIDE.md`
- `docs/designs/GENERATOR_MODULE_DESIGN.md`
- `docs/designs/FOUNDATION_RELEASE_MODEL.md`

## Scope

### In

- Generated business page routes, menu seeds, activation summaries, and inferred parents use `/business/*`.
- Generated source remains under `backend/modules/business/*` and `frontend/src/modules/business/*`.
- Business APIs remain under `/api/v1/business/*`.
- Foundation releases include `export-generated-module.mjs` and `transpile-typescript-files.mjs`.
- Foundation releases include the shared system/shell smoke contracts, runner tests, and their exact runtime helpers so consumer tests cannot drift behind shared UI changes.
- Reachable high-severity dependency debt surfaced by the release gate is remediated; upstream-only findings are reachability-assessed.
- Base and Ops runtime smoke, release, PR, CI, and branch closeout.

### Out

- Migration of manually designed business pages that intentionally use `/operations/*`.
- Generator UI redesign, new dependencies, database changes, or API prefix changes.

## Structural Scope

- Affected Subgraph: module schema -> page route helper -> frontend/backend generators -> dynamic module summary -> generated runtime smoke | foundation manifest -> release bundle -> Ops consumer tooling allowlist -> server-side exporter
- Boundary Crossings: platform lowcode -> generated business module | Base release -> Ops consumer
- Risk Nodes: route/menu mismatch | activation diagnostics | missing server exporter tooling
- Graph Focus: call-depth | sensitive-flow

## Expected Files

### Create

- `.harness/tasks/2026-08-06-generator-business-page-route/manifest.json`
- `.harness/evidence/2026-08-06-generator-business-page-route/*`
- `docs/harness/tasks/2026-08-06-generator-business-page-route.task.md`

### Modify

- lowcode generator route helpers and contract tests
- dynamic module summary and parent-menu inference tests
- generated business runtime smoke expectations
- foundation release manifest and tests
- shared smoke contract distribution and consumer sync coverage
- frontend dependency lock only when mandatory security gates find reachable debt

### Do Not Touch

- manually designed `/operations/*` pages
- business API prefixes
- database schema, permissions, audit contracts, or i18n product copy

## Implementation Notes

- Reuse `buildRoutePath` and `buildPageRoutePath` as the single page-route source.
- Extend the existing exact release tooling allowlist; add no packaging abstraction.
- Preserve API routes and manually designed Ops navigation.
- Do not downgrade dependencies when the downgrade reintroduces older reachable advisories.

## Verification Plan

- `go test -race ./...` from `backend/`
- `npm run test:generator:smoke` from `frontend/`
- `npm run test:smoke:business` from `frontend/`
- `npm run test:foundation-release`
- hosted Ops `Smoke Sanity` against the released shared smoke contracts
- frontend lint, type-check, build, dependency, docs, Harness, and security gates
- Ops foundation consumer, sync, inheritance, generator, and business smoke gates

## Linkage

- Task ID: `2026-08-06-generator-business-page-route`
- Task Manifest: `.harness/tasks/2026-08-06-generator-business-page-route/manifest.json`
- OpenSpec Change: `none`
- Superpowers Plan: `none`
- Plan References: `docs/harness/tasks/2026-08-06-generator-business-page-route.task.md`
- Evidence Directory: `.harness/evidence/2026-08-06-generator-business-page-route/`
- Review File: `.harness/evidence/2026-08-06-generator-business-page-route/review.md`

## Evidence Required

- exact local command results and security reachability assessment
- two-lane independent review
- rendered real generated business-page smoke under `/business/*`
- hosted required checks
- immutable release tag, archive, and checksum identity
- downstream Ops consumption and server-side exporter evidence

## Human Gates

- Satisfied by the maintainer decision that generated business pages belong under `/business/*` and the authorization to release Base and complete Ops consumption.

## Completion Checklist

- [x] Layer and boundary declared
- [x] Quality profile declared
- [x] Contract anchors read
- [x] Verification run or exception recorded
- [x] Evidence saved or summarized
- [x] Review completed
- [ ] Hosted checks, release, and Ops handoff completed
