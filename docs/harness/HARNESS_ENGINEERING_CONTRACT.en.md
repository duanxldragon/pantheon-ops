# Harness Engineering Contract

Chinese version: [HARNESS_ENGINEERING_CONTRACT.md](./HARNESS_ENGINEERING_CONTRACT.md)

Type: Contract
Layer: platform
Status: Active

This document defines the tool-agnostic Harness Engineering protocol for the Pantheon workspace. It constrains how all AI agents, automation tools, and human engineers receive tasks, read context, modify code, verify results, record evidence, and enter review.

## 1. Goal

Pantheon’s Harness Engineering goal is to make the repository itself the execution environment instead of locking process knowledge into a specific tool.

Codex, Claude Code, Cursor, GitHub Copilot, OpenHands, Aider, and human engineers may all participate in development, but they must follow the same repository protocol:

1. task boundaries are defined by contracts, design docs, and acceptance docs
2. tools are responsible for execution, verification, and evidence
3. critical decisions require a human gate or review gate
4. completion must be supported by tests, checks, browser evidence, and synchronized docs

## 2. Scope

This document applies to:

- development, review, fixes, and acceptance for `platform` and `system/*` in `pantheon-base`
- development, review, fixes, and acceptance for `business/*` in business repositories such as `pantheon-ops`
- drift governance, backporting, and inheritance upgrades between base and business repositories
- any task where an agent creates, modifies, or deletes code or docs
- any task that affects UI visuals, interaction, layout, responsive behavior, state rendering, or user experience

## 3. Non-Goals

This document does not define tool-specific shortcuts, private prompts, or plugin behavior. Tool-specific instructions belong in `.agents/adapters/` or in the corresponding tool configuration.

This document does not replace:

- `docs/WORKSPACE_INHERITANCE.md`
- `pantheon-base/AGENTS.md`
- `pantheon-base/docs/contracts/*`
- `pantheon-base/docs/acceptances/CODE_REVIEW_STANDARD.md`

## 4. Core Principles

### 4.1 Repository Protocol Comes First

Critical rules must live in the repository, with the following priority:

1. direct user instructions in the current session
2. repository entry files such as `AGENTS.md`, `CLAUDE.md`, `GEMINI.md`, `.cursor/rules`
3. tool-agnostic protocol under `docs/harness/*`
4. `docs/contracts/*`, `docs/designs/*`, `docs/acceptances/*`
5. tool adapters and private agent skills

If a tool adapter conflicts with a repository contract, the repository contract wins.

### 4.2 Tools Are Executors Only

Codex skills, Claude skills, Cursor rules, and Copilot instructions are adapters only. They may improve the workflow, but they must not become the only source of truth.

If a critical constraint exists only in a tool-specific configuration, it must be migrated or summarized into `docs/harness/*` or the relevant contract docs.

### 4.3 Humans Own Goals and Judgment

Humans are responsible for:

- defining goals and priorities
- deciding cross-layer boundaries and tradeoffs
- approving schema, permission, inheritance, deletion, and base-contract changes with high impact
- reviewing evidence and residual risk

Agents are responsible for:

- reading context through the task packet
- implementing the change
- running verification
- saving evidence
- producing review-ready notes

### 4.4 Evidence First

It is not enough to say “done”, “verified”, or “should be fine”. The executor must provide:

- commands that were run
- pass or fail results
- screenshots or smoke results for UI-relevant work
- known unverified items
- whether affected docs were synchronized

### 4.5 External Evaluator By Default

For `non-trivial` work, the harness should explicitly separate:

- implementer posture
- reviewer posture

The implementer may provide self-check notes, but “I checked it myself” is not the default completion decision.

The following work should default to an independent reviewer or explicit review gate:

- any task that crosses a Section 6 human gate
- security, permission, audit, schema, or inheritance-boundary work
- release, CI, secrets, deletion, or other high-impact operational changes

If only self-review is possible because of environment or task scale, the reason and residual risk must be stated explicitly.

### 4.6 Constraints Must Be Upgradeable And Retireable

Repeated failures should not only patch code. They should ratchet into guides, templates, sensors, or gates according to `FAILURE_RATCHET_POLICY.en.md`.

Likewise, old workarounds must not accumulate forever. After major model or tool upgrades, review which constraints can be downgraded, replaced, or removed according to `HARNESS_RETIREMENT_REVIEW.en.md`.

### 4.7 Default Execution Guardrails

Unless work is explicitly trivial, execution should apply `pantheon-harness/architecture/EXECUTION_GUARDRAILS.md` by default:

- think before coding: separate confirmed facts, working assumptions, and open questions
- simplicity first: record the smallest viable approach and its upgrade trigger in task packet `## Minimum Viable Approach`
- surgical changes: declare `Do Not Touch` and `Structural Scope` before editing files
- goal-driven verification: define `Success Criteria` and the verification signal before collecting evidence and review

## 5. Standard Workflow

All non-trivial tasks must follow this flow:

```text
Intake -> Context -> Plan -> Red -> Green -> Verify -> Evidence -> Review -> Handoff
```

### 5.1 Intake

Read the task packet or user request and declare:

- primary layer
- dependency layers
- touched contracts
- implementer posture
- reviewer posture
- expected verification
- human gates

### 5.2 Context

Read the necessary docs according to the repository reading order. Do not do unbounded full-repository reading.

For `pantheon-base`, the default reading order is:

1. `DESIGN.md`
2. `AGENTS.md`
3. `docs/README.md`
4. matching `docs/contracts/*`
5. matching `docs/designs/*`
6. `docs/acceptances/*`

For `pantheon-ops`, the default reading order is:

1. `../docs/WORKSPACE_INHERITANCE.md`
2. `AGENTS.md`
3. `docs/PROJECT_INHERITANCE.md`
4. `../pantheon-base/AGENTS.md`
5. matching base contracts and local business design docs

### 5.3 Plan

Non-trivial tasks must have a minimum plan. Cross-layer tasks, new features, refactors, drift governance, and sensitive configuration changes must use a task packet.

When the repository enables manifest-first automation, the task packet must also be paired with `.harness/tasks/<task-id>/manifest.json` so evidence and review checks can share the same machine-readable linkage.

Classification between `trivial` and `non-trivial` follows `TRIVIALITY_CLASSIFICATION_POLICY.en.md`.

### 5.4 Red

When behavior can be pinned down with a test or check, write the failing test or failing check first.

Exceptions must be explained:

- documentation-only cleanup
- exploratory audit work
- external dependencies that cannot be run in the current environment

### 5.5 Green

Implementation must stay within the task packet scope. Do not opportunistically refactor unrelated code or “fix” issues that have not been attributed.

### 5.6 Verify

Choose the smallest verification set that matches the change surface:

- backend: `go test` on relevant packages, and `go test ./...` when needed
- frontend: `npm run type-check`, `npm run lint`, `npm run build`
- contracts: menu, i18n, page access, permission, schema checks
- UI: smoke, screenshots, console errors, browser paths
- visual quality: run `impeccable` or an equivalent visual quality gate according to `VISUAL_QUALITY_PROTOCOL.md`

### 5.7 Evidence

Save or summarize verification results according to `VERIFICATION_EVIDENCE_SPEC.md`.

### 5.8 Review

Follow `REVIEW_LOOP_SPEC.md` and `CODE_REVIEW_STANDARD.md` for findings-first review.

Except for trivial work or clearly documented low-risk exceptions, review should exist as a post-implementation external evaluation stage rather than as a synonym for author self-check.

### 5.9 Handoff

The handoff note must include:

- changed files
- commands run
- pass/fail result
- evidence path
- review mode
- known gaps
- required human decisions

### 5.10 Default Adoption For Ordinary Work

Unless a task is explicitly trivial under Section 8, any code, contract, design, acceptance, UI, route, permission, i18n, schema, or workflow change must ship with:

- a task packet or an explicit link to an approved parent task packet
- verification evidence in the repository-defined structure
- a review mode, or a documented reason why only self-review was used
- known gaps recorded instead of omitted

This rule applies equally to:

- ordinary feature delivery
- bugfixes
- refactors that change behavior or contracts
- UI polish work
- cross-layer remediation work

## 6. Human Gates

The following operations require explicit human confirmation first, or must enter a PR review gate explicitly:

- deleting large numbers of files or directories
- changing database schema, migrations, or seed semantics
- changing permission, menu, audit, or i18n key rules
- changing `pantheon-base` contracts or system-domain boundaries
- overriding base platform or system-domain implementation inside a business repository
- introducing a new runtime dependency, external service, or security boundary
- changing CI gates, release flow, or secret-handling behavior

When a task crosses a human gate, it should also default to an independent reviewer posture instead of implementer self-check only.

## 7. Definition of Done

A task may be marked complete only when all of the following are true:

1. ownership layer and cross-layer boundaries have been declared
2. the corresponding contracts, design docs, and acceptance docs have been read
3. implementer posture and reviewer posture have been declared, or the reason for self-review-only mode has been documented
4. the task packet or minimum plan has been executed
5. verification commands matching the change surface have been run
6. verification evidence has been saved or summarized
7. if contracts, interfaces, menus, permissions, i18n, database, or acceptance semantics were touched, docs have been synchronized
8. if UI was touched, the visual quality gate has been run and screenshots, browser evidence, or a recorded reason for not running have been saved
9. unverified items and residual risk have been listed
10. the review gate has no unresolved P0/P1 findings
11. if document governance is enabled in the repository, the frontmatter / README / contract linkage checks defined by `DOCUMENT_FRONTMATTER_SPEC.md` must also pass

## 8. Trivial Task Exceptions

The following tasks may skip task-packet creation, but must still obey the relevant contracts:

- single-word typo fixes
- small clarifications to comments or README
- read-only queries, log inspection, or status reporting
- formatting-only changes that do not affect behavior and are covered by an existing formatter command

Even exception tasks may not bypass human gates for sensitive operations.

If trivial classification is disputed, use `TRIVIALITY_CLASSIFICATION_POLICY.en.md` and fall back to `non-trivial` by default.

## 9. Migration Path

Harness maturity progresses in three stages:

1. protocol layer: establish `docs/harness/*` and `.agents/*` so the workflow is tool-agnostic
2. evidence layer: establish task packet, evidence schema, and evidence directory conventions
3. gate layer: push architecture boundaries, permission, i18n, audit, drift, and document frontmatter governance down into scripts and CI

