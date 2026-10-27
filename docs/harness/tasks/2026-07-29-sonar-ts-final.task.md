# Task Packet: 2026-07-29-sonar-ts-final

## Goal

Close the final TypeScript Sonar remediation batch without changing platform or system UI behavior.

## Primary Layer

platform

## Dependency Layers

- system/auth
- system/iam
- system/org
- system/config

## Harness Profile

- Template: admin-platform
- Overlay: pantheon-base
- Quality Profile: ui-runtime
- Portable Failure Class: repo-quality-gate
- Owner Layer: consumer-repository
- Coverage Dimensions:
  - behaviour
  - maintainability
  - runtime-quality

## Contract Anchors

- `AGENTS.md`
- `DESIGN.md`
- `docs/designs/FRONTEND_UI_SPEC.md`
- `docs/acceptances/AGENT_EXECUTION_CHECKLIST.md`

## Scope

### In

- Complete the final nine TypeScript Sonar findings already remediated in the recovery branch.
- Make cross-page operation-log row selection type-safe and validate IDs before batch deletion.
- Apply Prettier only to the eight files in the frozen finding batch.

### Out

- UI layout, route, permission, i18n, API, database, and smoke-fixture changes.
- Go or CSS Sonar remediation, release tagging, publication, PR creation, and merge.

## Structural Scope

- Affected Subgraph: `OperationLogList -> crossPageSelection -> batchDeleteOperationLogs`
- Boundary Crossings: `platform -> system/audit`
- Risk Nodes: `cross-page row keys; destructive batch delete`
- Graph Focus: `sensitive-input-flow`

## Expected Files

### Create

- `.harness/tasks/2026-07-29-sonar-ts-final/manifest.json`
- `.harness/evidence/2026-07-29-sonar-ts-final/commands.json`
- `.harness/evidence/2026-07-29-sonar-ts-final/summary.md`
- `.harness/evidence/2026-07-29-sonar-ts-final/review.md`
- `docs/harness/tasks/2026-07-29-sonar-ts-final.task.md`

### Modify

- `frontend/src/core/layout/index.tsx`
- `frontend/src/modules/auth/login/components/Login.tsx`
- `frontend/src/modules/auth/security/components/SecurityCenter.tsx`
- `frontend/src/modules/platform/Dashboard.tsx`
- `frontend/src/modules/system/audit/OperationLogList.tsx`
- `frontend/src/modules/system/dept/DeptList.tsx`
- `frontend/src/modules/system/profile/ProfileCenter.tsx`
- `frontend/src/modules/system/user/UserFormModal.tsx`

### Do Not Touch

- `pantheon-base` main worktree, `.wt-sonar-ts`, `pantheon-ops`, backend source, release workflows

## Implementation Notes

- Minimal Complexity Rung: reuse existing `CrossPageRowKey`, `mergeCrossPageSelection`, and `batchDeleteOperationLogs`; add only local numeric validation.
- The audit table remains a dense Arco operational surface; no rendered layout or interaction contract is intentionally changed.

## Verification Plan

- `frontend: npm run lint`
- `frontend: npm run type-check`
- `frontend: npm run test:unit`
- `frontend: npm run build`
- `frontend: npx prettier --check <eight frozen files>`
- `git diff --check`
- Harness task, evidence, review, runtime, and visual-evidence checkers

## Linkage

- Task ID: `2026-07-29-sonar-ts-final`
- Task Manifest: `.harness/tasks/2026-07-29-sonar-ts-final/manifest.json`
- OpenSpec Change: `none`
- Superpowers Plan: `none`
- Plan References: `none`
- Evidence Directory: `.harness/evidence/2026-07-29-sonar-ts-final/`
- Review File: `.harness/evidence/2026-07-29-sonar-ts-final/review.md`

## Evidence Required

- command result summary
- explicit visual/runtime gap when no shared local application services are running
- independent-review request and author self-review

## Human Gates

- independent review before PR merge
- release gate remains owned by the coordinator

## Completion Checklist

- [x] Layer and boundary declared
- [x] Contract anchors read
- [x] Verification run or exception recorded
- [x] Evidence saved or summarized
- [ ] Review completed
