---
title: Clear the frontend follow-redirects high advisory so the dependency gate is green again
doc_type: Remediation
layer: inheritance-sync
status: Active
linked_contracts:
  - docs/PROJECT_INHERITANCE.md
updated_at: 2026-10-31
---

# Task Packet: 2026-10-31-frontend-follow-redirects-audit

Target repository: `pantheon-ops`
Layer: `inheritance-sync (frontend dependency lockfile owned by the consumer)`
Delivery tier: `L1`
Mode: unblock the hosted dependency vulnerability gate after a new upstream advisory

## Goal

`Dependency Vulnerabilities` on `main` fails again after time passed with no repository change:
`npm audit` for `frontend` now reports `follow-redirects` `1.16.1` under
GHSA-mrwp-f3q8-3g9p (high, "Cross-realm sensitiveHeaders arrays are discarded and custom
credentials can follow redirects"). The advisory range is `>=1.16.0 <=1.16.1`; `1.16.2` is published
and satisfies the `axios@1.20.0` constraint, so the fix is a lockfile refresh, not a dependency
upgrade or a source change.

## Primary Layer

inheritance-sync

## Workspace Context

- Target Repository: `pantheon-ops`
- Repository Role: `business-consumer`
- Upstream Dependencies: `pantheon-base` (base carries the same vulnerable pin; its lockfile is fixed
  in the same round so both consumers stay green)
- Downstream Consumers: `none`
- Sync Expectation: `not-required`
- Release Requirement: `none`

## Dependency Layers

- `frontend/package-lock.json` (`node_modules/follow-redirects`, resolved through `axios@1.20.0`)
- `.github/workflows/security.yml` `Dependency Vulnerabilities` job (`npm audit --audit-level=high`)
- `.github/workflows/security.yml` `Security Gates` aggregation job

## Harness Profile

- Template: api-service
- Overlay: pantheon-ops
- Quality Profile: ci-workflow
- Portable Failure Class: dependency-advisory-drift
- Owner Layer: inheritance-boundary
- Coverage Dimensions:
  - method-health

## Contract Anchors

- `frontend/package.json`
- `frontend/package-lock.json`
- `.github/workflows/security.yml`
- `docs/PROJECT_INHERITANCE.md`

## Scope

### In

- Refresh `node_modules/follow-redirects` in `frontend/package-lock.json` from `1.16.1` to
  `1.16.2` (the narrowest published version outside the advisory range), leaving every other lock
  entry untouched.
- Verify the installed tree resolves `axios@1.20.0 -> follow-redirects@1.16.2`, that the frontend
  audit reports zero high advisories, and that lint, type-check and build still pass.
- Record the hosted `Dependency Vulnerabilities` and `Security Gates` read-back as evidence.

### Out

- Upgrading, downgrading or re-resolving any other frontend dependency, including the 19 moderate
  advisories that `--audit-level=high` deliberately does not gate.
- Changing `.github/workflows/security.yml`, the gate thresholds or the advisory policy.
- Any change to `frontend/package.json` dependency ranges.
- Touching `backend/`, business source or i18n resources.

## Expected Files

### Create

- `docs/harness/tasks/2026-10-31-frontend-follow-redirects-audit.task.md`
- `.harness/tasks/2026-10-31-frontend-follow-redirects-audit/manifest.json`
- `.harness/evidence/2026-10-31-frontend-follow-redirects-audit/commands.json`
- `.harness/evidence/2026-10-31-frontend-follow-redirects-audit/summary.md`
- `.harness/evidence/2026-10-31-frontend-follow-redirects-audit/review.md`
- `.harness/evidence/2026-10-31-frontend-follow-redirects-audit/pr-body.md`

### Modify

- `frontend/package-lock.json`

### Do Not Touch

- `frontend/package.json`
- `.github/workflows/security.yml`
- `foundation-release.lock.json` and `.foundation/foundation-release.lock.json`
- `backend/modules/business/**`
- `frontend/src/modules/business/**`

## Implementation Notes

- `follow-redirects` is a transitive dependency of `axios@1.20.0`; `npm audit fix` would also
  re-resolve `@typescript-eslint/*`, `i18next` and `react-i18next`, which is a much wider change than
  the advisory requires. The `version` / `resolved` / `integrity` triple for the one package is
  edited directly and `npm ci` is run from the edited lock, so the result is exactly what the
  resolver would write for that package alone.
- `npm update` was rejected as the tool: with the local npm it also rewrites unrelated `libc` fields
  across optional platform packages, adding churn with no security value.
- No `overrides` entry is added: `1.16.2` already satisfies `axios`'s range, so a pin would only
  freeze a dependency the resolver can keep moving.

## Verification Plan

- `cd frontend && npm ci --ignore-scripts` from the edited lock, then `npm ls follow-redirects`.
- `cd frontend && npm audit --registry=https://registry.npmjs.org --json` with high/critical counting
  to zero.
- `cd frontend && npm run lint && npm run type-check && npm run build`.
- Hosted `Dependency Vulnerabilities` and `Security Gates` runs on the PR and on `main`.

## Linkage

- Task ID: `2026-10-31-frontend-follow-redirects-audit`
- Task Manifest: `.harness/tasks/2026-10-31-frontend-follow-redirects-audit/manifest.json`
- OpenSpec Change: `none`
- Superpowers Plan: `none`
- Plan References: `docs/harness/tasks/2026-10-31-frontend-follow-redirects-audit.task.md`
- Evidence Directory: `.harness/evidence/2026-10-31-frontend-follow-redirects-audit/`
- Review File: `.harness/evidence/2026-10-31-frontend-follow-redirects-audit/review.md`
- Related packet: `docs/harness/tasks/2026-10-31-foundation-upgrade-base-v0-18-2.task.md`

## Evidence Required

- Audit inventory before (1 high, `follow-redirects` 1.16.1) and after (0 high) the lock change.
- `npm ls follow-redirects` showing the resolved chain under `axios@1.20.0`.
- Frontend lint, type-check and build results from the edited lock.
- Hosted run identity and conclusion for `Dependency Vulnerabilities` and `Security Gates`.

## Human Gates

- None. The change is a lockfile refresh inside the consumer-owned frontend tree and does not alter
  product behaviour, gate policy or credentials.

## Completion Checklist

- [x] Layer and boundary declared
- [x] Contract anchors read
- [x] Verification run or exception recorded
- [x] Evidence saved or summarized
- [x] Review completed
