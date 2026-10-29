---
title: Restore the Ops SonarCloud CI analysis configuration
doc_type: Remediation
layer: inheritance-sync
status: Active
linked_contracts:
  - docs/PROJECT_INHERITANCE.md
updated_at: 2026-10-29
---

# Task Packet: 2026-10-29-sonar-ci-analysis-restore

Target repository: `pantheon-ops`
Layer: `inheritance-sync (pantheon-base -> pantheon-ops consumer configuration)`
Delivery tier: `L2`
Mode: repair an Ops-owned root configuration file that a foundation consumption silently dropped

## Goal

Make the hosted `SonarCloud Scan` workflow analyse this repository again. With the rotated
`SONAR_TOKEN` installed the scan authenticates but aborts with `You must define the following
mandatory properties: sonar.projectKey, sonar.organization`, because `sonar-project.properties` was
deleted when the consumer replaced the live tree with the `pantheon-base-v0.17.7` snapshot: the file
is Ops-owned (base ships only `.sonarcloud.properties`, which the scanner CLI never reads) and was
not declared in `business-overlay.json`, so the rebuild had no instruction to keep it.

## Primary Layer

inheritance-sync

## Workspace Context

- Target Repository: `pantheon-ops`
- Repository Role: `business-consumer`
- Upstream Dependencies: `pantheon-base`
- Downstream Consumers: `none`
- Sync Expectation: `not-required`
- Release Requirement: `none`

## Dependency Layers

- Ops repository overlay declaration (`business-overlay.json` `repositoryOverlayPaths`)
- Ops overlay assembly (`scripts/business-overlay/rebuild-from-base.mjs`)
- Ops overlay contract checker and tests (`scripts/business-overlay/check-business-overlay.mjs`)
- Ops CI workflow (`.github/workflows/sonarcloud.yml`) and the sonar-scanner CLI it invokes
- SonarCloud organization `duanxldragon` and project `duanxldragon_pantheon-ops`

## Harness Profile

- Template: api-service
- Overlay: pantheon-ops
- Quality Profile: ci-workflow
- Portable Failure Class: static-sensor-gap
- Owner Layer: inheritance-boundary
- Coverage Dimensions:
  - architecture-fitness
  - method-health

## Contract Anchors

- `AGENTS.md`
- `docs/PROJECT_INHERITANCE.md`
- `business-overlay.json`
- `scripts/business-overlay/rebuild-from-base.mjs`
- `.github/workflows/sonarcloud.yml`
- `.sonarcloud.properties`
- `sonar-project.properties`

## Scope

### In

- Restore `sonar-project.properties` with the content it held at `5c12964f^`, including
  `sonar.projectKey`, `sonar.organization`, the source exclusions, `sonar.coverage.exclusions` and
  `sonar.cpd.exclusions`.
- Declare `sonar-project.properties` in `business-overlay.json` `repositoryOverlayPaths` so a future
  foundation consumption cannot drop it again.
- Record the overlay contract, rebuild reproducibility and hosted SonarCloud Scan results as
  evidence.

### Out

- Any change to `.github/workflows/sonarcloud.yml` semantics, job conditions or gate policy.
- Any change to SonarCloud project settings, quality gates or organization membership.
- Committing, re-asserting or rotating secret material inside the repository.
- Remediation of the code smells and vulnerabilities the restored scan now reports.
- Making Dependabot runs able to scan, which is a gate-policy decision for the maintainer.

## Expected Files

### Create

- `sonar-project.properties`
- `docs/harness/tasks/2026-10-29-sonar-ci-analysis-restore.task.md`
- `.harness/tasks/2026-10-29-sonar-ci-analysis-restore/manifest.json`
- `.harness/evidence/2026-10-29-sonar-ci-analysis-restore/commands.json`
- `.harness/evidence/2026-10-29-sonar-ci-analysis-restore/summary.md`
- `.harness/evidence/2026-10-29-sonar-ci-analysis-restore/review.md`
- `.harness/evidence/2026-10-29-sonar-ci-analysis-restore/pr-body.md`

### Modify

- `business-overlay.json`

### Do Not Touch

- `.github/workflows/sonarcloud.yml`
- `.sonarcloud.properties` (scopes SonarCloud automatic analysis and is already declared)
- `foundation-release.lock.json` and `.foundation/foundation-release.lock.json` (no lock advance)
- any repository secret value inside the working tree

## Implementation Notes

- `copyDeclaredPath` copies every declared `repositoryOverlayPaths` entry from the Ops tree
  regardless of whether base ships that path, and the product-root guard only rejects `backend`,
  `frontend/src`, `frontend/tests/smoke`, `database` and `schema`, so a root configuration file is a
  legal repository overlay.
- The file disappeared at `5c12964f` (consume `pantheon-base-v0.17.7`) because it was never
  declared; the assembly had no instruction to keep an Ops-only root file. This is the mirror image
  of the pinned `scripts/task-manifest.mjs` overlay found in the 0.18.1 round: overlay declarations,
  not file contents, decide what survives a consumption.
- `.sonarcloud.properties` and `sonar-project.properties` are both required and are not duplicates:
  the scanner CLI that the workflow runs only reads `sonar-project.properties`, while
  `.sonarcloud.properties` scopes SonarCloud's automatic analysis.
- The restored file also carries `sonar.coverage.exclusions=**/*` and the test-file
  `sonar.cpd.exclusions`. The organization is on the SonarCloud Free plan, which cannot attach a
  custom quality gate, so the built-in `new_coverage` condition can only be neutralised by making
  coverage not computed.
- `SONAR_TOKEN` was rotated by the maintainer for this repository; with it installed the scan
  authenticates and now fails only on the missing mandatory properties, which is what this packet
  closes.

## Verification Plan

- `npm run check:business-overlay` and `npm run test:business-overlay` after the declaration.
- `npm run rebuild:from-base` and confirm the rebuilt tree still contains
  `sonar-project.properties`.
- `node scripts/harness/check-task-packet.mjs` on this packet and `npm run check:harness-adoption`.
- Hosted `SonarCloud Scan` run reaching an analysis instead of the missing-property error, with the
  rotated `SONAR_TOKEN`.

## Linkage

- Task ID: `2026-10-29-sonar-ci-analysis-restore`
- Task Manifest: `.harness/tasks/2026-10-29-sonar-ci-analysis-restore/manifest.json`
- OpenSpec Change: `none`
- Superpowers Plan: `none`
- Plan References: `docs/harness/tasks/2026-10-29-sonar-ci-analysis-restore.task.md`
- Evidence Directory: `.harness/evidence/2026-10-29-sonar-ci-analysis-restore/`
- Review File: `.harness/evidence/2026-10-29-sonar-ci-analysis-restore/review.md`
- Related packet: `docs/harness/tasks/2026-10-29-foundation-upgrade-base-v0-18-1.task.md`

## Evidence Required

- Scanner error text before the change (`mandatory properties` missing) and the analysis result
  after it.
- Overlay contract, overlay test and rebuild reproducibility after the declaration.
- SonarCloud API read-back for token validity, organization and project identity.
- Hosted `SonarCloud Scan` run identity and conclusion.

## Human Gates

- Credential rotation: `SONAR_TOKEN` for this repository was replaced with the maintainer-supplied
  token; the previous value returned HTTP 403 from `api.sonarcloud.io`.
- Gate policy: whether Dependabot runs should scan (Dependabot secrets) or be skipped is left to the
  maintainer and is not decided in this packet.

## Completion Checklist

- [x] Layer and boundary declared
- [x] Contract anchors read
- [x] Verification run or exception recorded
- [x] Evidence saved or summarized
- [x] Review completed
