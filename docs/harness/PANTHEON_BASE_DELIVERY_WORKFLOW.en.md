---
title: Pantheon Base Multi-Agent Delivery Workflow
doc_type: Acceptance
layer: platform
status: Active
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
  - docs/contracts/DOCUMENT_GOVERNANCE_CONTRACT.md
updated_at: 2026-06-23
---

# Pantheon Base Multi-Agent Delivery Workflow

Chinese version: [PANTHEON_BASE_DELIVERY_WORKFLOW.md](./PANTHEON_BASE_DELIVERY_WORKFLOW.md)

This document defines how `pantheon-base` uses Claude, Codex, `acpx`, and `omc` to run the Harness Engineering workflow.

The goal is that the human states goals and key decisions, while the active dispatcher routes tools and records repository artifacts.

## Roles

| Role | Default Tool | Responsibility | Must Not Do |
|---|---|---|---|
| Human Owner | Human | goals, priority, risk acceptance, human gates | manually shuttle Claude/Codex context |
| Dispatcher | active coordinating agent | call planner, executor, reviewer; preserve task/evidence/review linkage | treat chat as the source of truth |
| Planner | Claude | scope, task packet, task manifest linkage, acceptance, stop points | edit business code |
| Explorer | Codex | code structure, impact, CodeGraph, tests and docs | expand scope |
| Executor | Codex | implementation, tests, evidence, finding fixes | skip task packets or evidence |
| Reviewer | Claude | findings-first review | fix code directly |
| Mechanical Gates | GitHub Actions / local scripts | repeatable validation | replace high-risk human decisions |

## Default Flow

```text
Human Goal
  -> Dispatcher intake
  -> Claude planner
  -> Dispatcher scope check
  -> Codex explorer
  -> Task packet and task manifest finalized
  -> Codex executor
  -> Local sensors and evidence
  -> Claude reviewer
  -> Codex fixer when needed
  -> GitHub / CI governance
  -> PR merge and branch cleanup
  -> Ratchet or closeout
```

## Tool Policy

Use `acpx` as the deterministic dispatch surface for named agents, sessions, and permission posture.

Use `omc` for discovery, capability routing, and future orchestration. Do not rely on `omc -q` for non-trivial or high-risk tasks unless its routing is explicitly configured to keep Claude as planner/reviewer and Codex as executor.

GitHub Actions are mechanical gates. `Quality Gates` and `Security Gates` are PR-required signals. CodeQL is the primary security signal in `Security Gates`, while Copilot review status plus PR residual-risk notes cover architecture, intent, and maintainability gaps that scanners do not prove. `Full Smoke Suite` remains advisory, scheduled, or manual deep signal.

Worktrees are for local implementation and validation, not as long-lived delivery artifacts. The fuller worktree -> push -> PR -> merge -> cleanup loop lives in `docs/designs/WORKFLOW.md`.

## Human Gates

The dispatcher must stop for human approval before schema, permissions, auth, audit, CI, secrets, destructive, release, or external-service changes; when planner and explorer disagree on scope; or when review findings require expanding the original task.

## Minimum Closeout

Every non-trivial task needs:

- task packet or parent task packet linkage
- task manifest path
- evidence path or command summary
- reviewer role and review result
- GitHub signal classification
- merged PR URL and merge commit
- branch cleanup status
- known gaps
- ratchet decision

If any item is missing, the task is only partially closed.

## Planning Expectations

Planner output should include at least:

- In / Out / Do Not Touch
- Assumptions and Open Questions
- Minimum Viable Approach
- Success Criteria
- the minimum verification set
- task manifest / evidence / review linkage

Those artifacts should land in `docs/harness/tasks/*.task.md` and `.harness/tasks/<task-id>/manifest.json`, not remain only in chat.

## Execution Expectations

Executor work must keep task packet, task manifest, evidence, and review linkage aligned while implementation, verification, and closeout proceed.
