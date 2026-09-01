# Task Packet: 2026-08-21-sonarcloud-remediation

## Goal

Resolve all unresolved SonarCloud issues on main to unblock the Release Gate and enable `pantheon-ops` to consume a green foundation release.

## Primary Layer

platform

## Dependency Layers

- system/auth
- system/iam
- system/org
- system/config
- lowcode
- frontend

## Harness Profile

- Template: admin-platform
- Overlay: pantheon-base
- Quality Profile: ci-workflow, ui-runtime, permission-policy, generator
- Portable Failure Class: repo-quality-gate
- Owner Layer: consumer-repository
- Coverage Dimensions:
  - behaviour
  - maintainability
  - architecture-fitness
  - runtime-quality
  - method-health

## Contract Anchors

- `AGENTS.md`
- `DESIGN.md`
- `docs/README.md`
- `.github/workflows/release-gate.yml`

## Scope

### In

- All unresolved SonarCloud BUG, VULNERABILITY, and CODE_SMELL issues
- Focus on high-impact rules: S3776 (complexity), S107 (parameters), S1192 (duplicated literals)
- Verify quality gate passes after remediation

### Out

- Public API or DTO behavior changes
- Database schema or seed changes
- Permission, menu, authentication, or audit contract changes
- New dependencies
- Pantheon Ops synchronization
- Visual redesign

## Implementation Notes

1. First, query SonarCloud API to get current issue list
2. Categorize issues by rule and file
3. Prioritize by severity (BUG > VULNERABILITY > CODE_SMELL)
4. Apply behavior-preserving refactors only
5. Run local verification before PR
6. Ensure Release Gate passes after merge

## Verification Plan

- `cd backend && go test -count=1 ./... && go vet ./...`
- `cd frontend && npm run lint && npm run type-check && npm run test:unit`
- `cd frontend && npm run build`
- SonarCloud quality gate OK
- Zero unresolved SonarCloud issues

## Expected Files

### Create

- `.harness/evidence/2026-08-21-sonarcloud-remediation/summary.md`
- `.harness/evidence/2026-08-21-sonarcloud-remediation/review.md`
- `.harness/evidence/2026-08-21-sonarcloud-remediation/commands.json`

### Modify

- Backend Go files (complexity reduction, parameter consolidation)
- Frontend TypeScript files (complexity reduction, readonly fixes)
- `docs/harness/tasks/2026-08-21-sonarcloud-remediation.task.md`

### Do Not Touch

- Public API contracts
- Database migrations
- Permission/audit/i18n contracts

## Success Criteria

- [ ] SonarCloud quality gate: OK
- [ ] Unresolved SonarCloud issues: 0
- [ ] All required GitHub checks pass
- [ ] Release Gate passes
- [ ] `pantheon-ops` can consume the new foundation release

## Linkage

- Task ID: `2026-08-21-sonarcloud-remediation`
- Evidence Directory: `.harness/evidence/2026-08-21-sonarcloud-remediation/`
- Plan References: `.github/workflows/release-gate.yml`

## Human Gates

- Required GitHub checks and repository merge protection
- Release Gate approval for new foundation release

## Completion Checklist

- [ ] Layer and boundary declared
- [ ] Quality profile declared
- [ ] Contract anchors read
- [ ] Verification run or exception recorded
- [ ] Evidence saved or summarized
- [ ] Review completed
