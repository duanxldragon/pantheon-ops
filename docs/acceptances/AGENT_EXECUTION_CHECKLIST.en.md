---
title: Pantheon Base Agent Execution Checklist
doc_type: Acceptance
layer: platform
status: Active
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
  - docs/contracts/SYSTEM_AUTH_CONTRACT.md
  - docs/contracts/SYSTEM_IAM_CONTRACT.md
  - docs/contracts/SYSTEM_ORG_CONTRACT.md
  - docs/contracts/SYSTEM_CONFIG_CONTRACT.md
updated_at: 2026-06-10
---

# Pantheon Base Agent Execution Checklist

Chinese version: [AGENT_EXECUTION_CHECKLIST.md](./AGENT_EXECUTION_CHECKLIST.md)

This checklist turns a few high-value Harness Engineering practices into daily defaults for `pantheon-base`.

Use it together with:

- [../../AGENTS.md](../../AGENTS.md)
- [TASK_PACKET_BASE_TEMPLATE.md](./TASK_PACKET_BASE_TEMPLATE.md)
- [ACCEPTANCE_CHECKLIST.md](./ACCEPTANCE_CHECKLIST.md)
- [CODE_REVIEW_STANDARD.md](./CODE_REVIEW_STANDARD.md)

## Default Decision Order

Before editing:

1. classify the ownership layer: `platform`, `system/*`, or `business/*`
2. decide whether the issue is base-owned or business-owned
3. classify the task profile: `UI`, `contract`, `schema`, `runtime-sensitive`, `inheritance-sync`, `generator`
4. name the implementer posture and at least one reviewer posture
5. decide the minimum evidence set for this turn

## Base-Owned By Default

Treat the following as `pantheon-base` work unless proven otherwise:

- platform shell, navigation, dashboard, shared page scaffolding
- shared `auth / iam / org / config` behavior
- generic permissions, menus, i18n, audit, pagination, upload, tables
- lowcode generation, dynamic-module registration, shared smoke helpers
- reusable UX or state-quality fixes

## Constraint-As-Enhancement

Default execution rules:

- apply the minimal complexity ladder first: skip, reuse, standard library, native platform, installed dependency, one local expression, then minimum new code
- one task closes one bounded slice
- no opportunistic unrelated refactors
- split cross-domain work at explicit gates
- use CodeGraph first for structure, `rg` for literal strings and logs
- record `In / Out / Do Not Touch` instead of relying on chat memory
- do not use simplification to weaken authorization, audit, i18n, accessibility, runtime evidence, or explicit user requirements

## External Evaluator Default

Every non-trivial task should name at least one reviewer posture:

- architecture
- security
- UX / QA
- mechanical gate

High-risk work should not self-approve. Review output should stay findings-first and point at the same task packet and evidence.

## Repeat-Failure Ratchet

If the same failure pattern repeats:

1. first time: record it in closeout, review, or evidence
2. second time: promote it into `AGENTS.md`, a checklist, a template, or a contract entry
3. third time or cross-repo recurrence: promote it into a checker, smoke case, fixture, or failure-registry entry

The rule is not fully absorbed until a deterministic sensor can catch it.

## Runtime-Sensitive Evidence

Treat these areas as runtime-sensitive by default:

- login, session, token, authorization flows
- menu assembly, route guard, permission gating
- import/export, upload, batch actions, workflow paths
- `generator`, `dynamicmodule`, activation or registration flows
- async jobs, external integrations, retries, idempotency, concurrency
- performance-sensitive changes

In addition to tests, provide one of:

- focused logs
- a smoke or browser path
- metrics or latency samples
- a trace or request-chain explanation
- an explicit runtime gap with reason and risk

## UI And State Evidence

For UI-affecting work, check:

- `loading`
- `empty`
- `error`
- `forbidden`
- `submitting`

Also record:

- final URL or route
- console-error result
- screenshot or explicit visual-gap record
- desktop/mobile scope when layout changes

## Minimum Closeout

Every handoff should state:

- what changed
- what did not change
- commands run
- evidence location
- remaining gaps or risks
- whether `base -> ops` sync is required
- merged PR URL and merge commit
- branch cleanup status: whether the remote head branch was deleted and the local worktree was cleaned up
