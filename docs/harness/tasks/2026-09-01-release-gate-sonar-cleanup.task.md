# Task Packet: 2026-09-01-release-gate-sonar-cleanup

## Goal

Resolve the final unresolved SonarCloud issue on `main` so the Base Release Gate can pass and a new foundation release can be published for Pantheon Ops.

## Primary Layer

platform

## Quality Profile

- `ui-runtime`
- `ci-workflow`

## Scope

### In

- Preserve the shared `AppTable` behavior while reducing the reported `typescript:S3776` cognitive complexity.
- Verify frontend static checks and the shared pagination browser path.
- Use hosted SonarCloud analysis and the Release Gate as final external evidence.

### Out

- Changes to the public `AppTable` API, menus, permissions, i18n resources, audit behavior, APIs, schemas, or Pantheon Ops sources.
- Visual redesign or dependency updates.

## Assumptions And Open Questions

- SonarCloud reported exactly one unresolved issue at `frontend/src/components/data-display/AppTable.tsx:494` on `main`.
- The refactor preserves rendered controls and table states; hosted analysis is required to confirm the S3776 issue is closed.

## Minimum Viable Approach

Reuse the existing table component contract and extract its presentation branch into a private component. No new dependency, public abstraction, or behavioral branch is introduced.

## Structural Scope

- Affected subgraph: `AppTable` presentation composition and its existing table/pagination dependencies.
- Boundary crossings: none; this stays in the `platform` frontend component layer.
- Risk nodes: responsive hint, empty state, row selection, pagination renderer.
- Graph focus: `AppTable` consumers retain the same exported component and props.

## Verification Plan

- `cd frontend && npm run type-check`
- `cd frontend && npm run lint`
- `cd frontend && npm run build`
- `cd frontend && node scripts/run-smoke-suite.mjs --host 127.0.0.1 --port 5173 --config playwright.config.ts -- tests/smoke/platform/pagination-contract.spec.ts --workers=1`
- Hosted SonarCloud analysis and `Release Gate Summary` on the merged commit.

## Linkage

- Task ID: `2026-09-01-release-gate-sonar-cleanup`
- Task Manifest: `.harness/tasks/2026-09-01-release-gate-sonar-cleanup/manifest.json`
- Evidence: `.harness/evidence/2026-09-01-release-gate-sonar-cleanup/commands.json`
- Review Artifact: `.harness/evidence/2026-09-01-release-gate-sonar-cleanup/review.md`

## Delivery Governance

- Portable Failure Class: `repo-quality-gate`
- Owner Layer: `consumer-repository`
- Ratchet Decision: `no-repeat-observed`
- GitHub Signal: `repo-quality-gate`
- Human Gate: GitHub required checks, SonarCloud analysis, and Release Gate Summary.
