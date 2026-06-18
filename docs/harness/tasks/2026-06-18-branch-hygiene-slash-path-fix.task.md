# Task Packet: 2026-06-18-branch-hygiene-slash-path-fix

## Goal

Fix the hosted `branch-hygiene` cleanup flow so slash-separated branch names keep working against the real GitHub branch lookup and branch deletion REST endpoints.

## Primary Layer

platform/governance

## Dependency Layers

- none

## Harness Profile

- Template: pantheon-ops custom
- Overlay: none
- Quality Profile: ci-workflow
- Portable Failure Class: ci-signal-noise
- Owner Layer: repository-governance
- Coverage Dimensions:
  - behaviour
  - maintainability

## Contract Anchors

- `AGENTS.md`
- `docs/PROJECT_INHERITANCE.md`
- `../pantheon-base/AGENTS.md`
- `../pantheon-base/DESIGN.md`
- `../pantheon-base/docs/README.md`
- `docs/harness/tasks/2026-06-18-branch-hygiene-fallback.task.md`

## Scope

### In

- preserve raw slash-separated branch names for GitHub `/branches/{branch}` lookup paths
- preserve raw slash-separated branch names for GitHub `/git/refs/heads/{branch}` delete paths
- add regression coverage that locks this hosted GitHub API behavior for slash branch names

### Out

- changing the `pulls?head=` or `pulls?base=` query parameter encoding rules
- changing business-domain code or inheritance overlays
- changing merge strategy, PR governance, or local branch cleanup behavior

## Structural Scope

- Affected Subgraph: `push|schedule|workflow_dispatch -> branch-hygiene workflow -> cleanup script -> GitHub branch lookup/delete endpoints`
- Boundary Crossings: `none`
- Risk Nodes: `slash branch path handling`, `recreated branch safety`, `hosted verification fidelity`
- Graph Focus: `sensitive-input-flow`

## Expected Files

### Modify

- `scripts/cleanup-github-branches.mjs`
- `tests/scripts/cleanup-github-branches.test.mjs`

### Create

- `docs/harness/tasks/2026-06-18-branch-hygiene-slash-path-fix.task.md`
- `.harness/evidence/2026-06-18-branch-hygiene-slash-path-fix/commands.json`
- `.harness/evidence/2026-06-18-branch-hygiene-slash-path-fix/review.md`
- `.harness/evidence/2026-06-18-branch-hygiene-slash-path-fix/pr-body.md`

### Do Not Touch

- `.github/workflows/pr-automation.yml`
- `backend/modules/business/*`
- `frontend/src/modules/business/*`
- shared backend sync payloads from `pantheon-base`

## Implementation Notes

- Real GitHub hosted verification showed that these branch endpoints expect slash-separated branch names as path segments rather than `%2F`-encoded path atoms.
- Open-PR and base-branch query filters should keep their existing query encoding posture.
- The deletion guard must stay SHA-verified so a recreated branch with the same name is preserved.

## Method Readiness

- Consumer-Specific Controls: pantheon-ops repo-local governance
- Required Sensors: command
- Required Evidence: workflow test output | hosted residue verification note
- Ratchet Decision: sensor-added
- Deferred Code Issues: none

## Delivery Governance

- Design Gate: root-cause note captured
- Development Gate: regression coverage added before hosted rerun
- QA Acceptance Gate: local workflow and script verification
- GitHub Governance Gate: repo-quality-gate

## Execution Roles

- Implementer Posture: implementer
- Reviewer Posture: architecture | mechanical

## Stop Points

- stop before changing query parameter encoding semantics
- stop before widening branch deletion beyond the closed-PR residue contract

## Verification Plan

- `npm run test:branch-hygiene`
- hosted GitHub rerun against a real closed-PR residue branch after merge

## Linkage

- Task ID: `2026-06-18-branch-hygiene-slash-path-fix`
- OpenSpec Change: none
- Plan References: none
- Evidence Directory: `.harness/evidence/2026-06-18-branch-hygiene-slash-path-fix/`
- Review File: `.harness/evidence/2026-06-18-branch-hygiene-slash-path-fix/review.md`

## Evidence Required

- branch hygiene workflow test output
- hosted GitHub residue cleanup proof after merge

## Human Gates

- none

## Completion Checklist

- [ ] hosted slash-path root cause recorded
- [ ] regression coverage added
- [ ] local branch-hygiene tests rerun
- [ ] hosted residue cleanup rerun after merge
- [ ] residual risk updated
