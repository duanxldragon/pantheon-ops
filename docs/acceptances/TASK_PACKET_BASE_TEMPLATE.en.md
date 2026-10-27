---
title: Pantheon Base Task Packet Template
doc_type: Acceptance
layer: platform
status: Active
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
updated_at: 2026-06-08
---

# Pantheon Base Task Packet Template

Chinese version: [TASK_PACKET_BASE_TEMPLATE.md](./TASK_PACKET_BASE_TEMPLATE.md)

This is the structured task-packet example for `pantheon-base`. It consumes the portable `pantheon-harness` method and only adds constraints from the current admin-platform architecture.

Use it for:

- `platform`
- `system/auth`
- `system/iam`
- `system/org`
- `system/config`
- `system/lowcode`

Copy and fill:

Target repo: pantheon-base. Cross-repository tasks must fill `## Workspace Context`. Sync expectation: use `Sync Expectation` and `## Sync expectation`.

```md
# Task Packet: <task-name>

## Goal

<one sentence>

## Primary Layer

platform | system/auth | system/iam | system/org | system/config | business/* | inheritance-sync

## Workspace Context (required for inheritance-sync)

- Target Repository: `pantheon-base`
- Repository Role: `foundation-source`
- Upstream Dependencies: `pantheon-harness`
- Downstream Consumers: `pantheon-ops`
- Sync Expectation: `none | not-required | deferred | required | completed`
- Release Requirement: `none | method-release | foundation-release | consumer-lock-update`

`Sync Expectation` and `Release Requirement` describe the actual handoff boundary; they do not claim that a release or downstream sync has already happened.

## Dependency Layers

- none

## Harness Profile

- Template: admin-platform
- Overlay: pantheon-base
- Quality Profile: auth-security | permission-policy | i18n | ui-runtime | generator | ci-workflow | none
- Portable Failure Class: instruction-gap | task-boundary-gap | architecture-drift | test-gap | static-sensor-gap | runtime-evidence-gap | security-boundary-gap | ci-signal-noise | method-health-gap | none
- Owner Layer: portable-method | consumer-template | consumer-repository | agent-adapter | no-action
- Coverage Dimensions:
  - behaviour
  - maintainability
  - architecture-fitness
  - runtime-quality
  - method-health

## Contract Anchors

- `pantheon-base/AGENTS.md`
- `pantheon-base/DESIGN.md`
- `pantheon-base/docs/README.md`
- `<matching contract / design / acceptance>`

## Scope

### In

- <explicit closure for this turn>

### Out

- <explicit non-goals for this turn>

## Structural Scope

- Affected Subgraph: `<entry -> core path -> exit/side effect>` | `none`
- Boundary Crossings: `none | platform -> system/auth | system/* -> pkg/* | base -> ops`
- Risk Nodes: `none | auth handler | permission service | menu registry | generator orchestrator`
- Graph Focus: `none | cycle-check | hub-check | call-depth | sensitive-input-flow`

## Expected Files

### Create

- `path` | none

### Modify

- `path` | none

### Do Not Touch

- `path` | none

## Implementation Notes

- If the work touches shared pagination, upload, tables, i18n, or the shared admin shell, treat it as base-owned by default.
- If the turn changes downstream inheritance behavior, explicitly state whether a `base -> ops` sync is required.
- Apply the minimal complexity ladder first: skip | reuse | standard library | native platform | installed dependency | one local expression | minimum new code. Do not simplify away authorization, audit, i18n, accessibility, or runtime evidence.

## Method Readiness

- Consumer-Specific Controls: `pantheon-base` contract | checker | smoke path | none
- Required Sensors: command | review | runtime evidence | none
- Required Evidence: command summary | screenshot | smoke result | runtime gap | review summary
- Minimal Complexity Rung: skip | reuse | stdlib | native-platform | installed-dependency | one-local-expression | minimum-new-code
- Ratchet Decision: no-repeat-observed | guide-updated | sensor-added | gate-updated | template-updated | adapter-updated | registry-only
- Deferred Code Issues: none | symptom plus recommended follow-up task

## Delivery Governance

- Design Gate: spec reference | short boundary note | none
- Development Gate: expected files declared | do-not-touch declared | none
- QA Acceptance Gate: command | browser | runtime | human review | none
- GitHub Governance Gate: method-gate | repo-quality-gate | runtime-evidence-gate | external-flaky | not-applicable

## Execution Roles

- Implementer Posture: implementer | reviewer-assisted | docs-only | none
- Reviewer Posture: architecture | security | UX-QA | mechanical | none

## Stop Points

- `none`
- or pause before schema / contract / delete / release gate

## State Plan

- Checkpoint Expectation: none | path | artifact name
- Resume Artifacts: none | path

## Verification Plan

### Backend

- `go test ...` | `go test ./...` | none

### Frontend

- `npm run build` | none

### Browser / Smoke

- `node scripts/run-smoke-suite.mjs ...` | none

### Runtime Evidence

- focused logs | smoke path | metrics sample | trace | explicit runtime gap | none

## Linkage

- Task ID: `<YYYY-MM-DD-task-name>`
- OpenSpec Change: `openspec/changes/<name>/` | none
- Superpowers Plan: `docs/superpowers/plans/<file>.md` | none
- Evidence Directory: `.harness/evidence/<task-id>/`
- Review File: `.harness/evidence/<task-id>/review.md` | none

## Evidence Required

- command result summary
- screenshots if UI changed
- smoke JSON if browser flow changed
- runtime logs / metrics / traces / performance signal, or explicit runtime gap if runtime-sensitive
- review summary

## Human Gates

- none
- schema / contract / deletion / high-risk permission / release gate

## Sync expectation

- base-only
- if shared behavior will affect pantheon-ops, record whether sync is deferred or included this turn
- if `base -> ops` sync is required, state the trigger, sync scope, and verification command

## Foundation Release Handoff

- Shared change owner: `<pantheon-base | pantheon-harness | none>`
- Foundation release required: `yes | no | deferred`
- Consumer sync status: `not-started | planned | in-progress | completed | blocked`
- Downstream validation command: `<exact command or none>`
- Release/lock stop point: `<immutable release, consumer lock update, migration, human gate, or none>`

## Completion Checklist

- [ ] Layer and boundary declared
- [ ] Quality profile or explicit `none` declared
- [ ] Ratchet decision declared for repeated failures
- [ ] Delivery governance gates declared
- [ ] Contract anchors read
- [ ] Verification run or exception recorded
- [ ] Evidence saved or summarized
- [ ] Review completed
```

Additional rules:

- Login, permissions, menu routing, import/export, generator, dynamic modules, async chains, or external integrations need a runtime-sensitive evidence plan by default.
- If the same failure pattern has already appeared before, Ratchet Decision must not be `no-repeat-observed`.
- If this turn only patches code without promoting the issue into a rule, checker, or smoke path, record why in Deferred Code Issues or the failure registry.
