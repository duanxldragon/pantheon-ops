---
title: Workflow context and verification improvements
doc_type: Remediation
layer: business
status: Archived
updated_at: 2026-11-24
linked_contracts:
  - docs/PROJECT_INHERITANCE.md
---

# Task Packet: workflow-context-improvement

## Goal

Make Pantheon development easier to resume and verify by aligning current entrypoints, reducing repeated context loading, and connecting method consumption and workflow checks.

## Primary Layer

business/*

## Workspace Context

- Target Repository: `pantheon-ops`
- Repository Role: `business-consumer`
- Upstream Dependencies: `none`
- Downstream Consumers: `pantheon-base, pantheon-ops`
- Sync Expectation: `deferred`
- Release Requirement: `method-release`

## Dependency Layers

- method
- inheritance-sync

## Harness Profile

- Template: custom
- Overlay: none
- Quality Profile: portable-method
- Portable Failure Class: instruction-gap
- Owner Layer: portable-method
- Coverage Dimensions:
  - maintainability
  - architecture-fitness
  - method-health

## Contract Anchors

- `AGENTS.md`
- `.github/workflows/quality.yml`
- `.harness/method-consumption.json`
- `scripts/harness/check-evidence.mjs`
- `scripts/harness/check-review.mjs`
- `docs/PROJECT_INHERITANCE.md`
- `business-overlay.json`

Canonical methodology was read from sibling Harness; the receipt pins its reviewed source.

## Scope

### In

- Current bilingual workspace/consumer entrypoints and task-specific reading routes.
- Compact current status, preserved original checkpoints, and a manifest-derived task index.
- Reproducible CI method-policy checkout, consumer receipt provenance and negative verification.
- Base inheritance-command validation and checker invocation classification; Ops-owned workflow wiring and rebuild preservation.
- Repository verification skills, execution environment guidance and small workflow-measurement records.
- Verification evidence and independent review of policy/CI changes.

### Out

- Product/backend/frontend behavior, credentials, production environments, foundation lock/version changes, full product acceptance, and unrelated in-progress work.
- Formal method/foundation release publication, consumer shared-source copying and gate exemptions. The user subsequently authorized scoped commits, branches, PRs, CI and merge for this task only.
- Claims of measured productivity gains or runtime/UI validation for this method-only change.

## Assumptions and Open Questions

- Confirmed Facts: user explicitly requested applying the review to methods and mechanisms; Harness 1.5.0 is published; consumer worktrees contain in-progress changes.
- Working Assumptions: preserve current risk tiers, required gates and maintainer authority; distinguish inherited manual utilities from configured CI gates.
- Open Questions: none blocking reversible local implementation; formal publication and product-source inheritance remain separate follow-through.

## Expected Files

### Create

- `scripts/harness/render-task-index.mjs` and behavioral tests.
- `.harness/evidence/2026-11-24-workflow-context-improvement/`
- Consumer `.harness/TASK_INDEX.md` and original status snapshots.
- Focused Base and Ops method/CI regression tests.

### Modify

- Workspace inheritance rules; active bilingual consumer entries, indexes and verification skills.
- Portable method consumption checker and its tests; script inventory and evaluation guidance.
- Base inheritance and checker-wiring sensors/tests and quality workflow.
- Ops-owned quality workflow, root script assembly and overlay declaration.

### Do Not Touch

- Existing product code, foundation lockfiles, product VERSION, original evidence and archived task decisions.
- Global settings, authentication and other tasks' process/runtime state.

## Implementation Notes

- Implementer: primary agent owns integration and reversible edits; unavailable CodeGraph tools use focused file retrieval.
- Reviewer: independent native code-reviewer checks ownership, risk gates, provenance, preserved history and behavior.
- Preserve the seven reviewed 1.5.0 policy bytes. New helper/checker revisions remain Unreleased; policy consumption pins reviewed merged Harness PR #13; formal release publication remains separate.

## Minimum Viable Approach

- Selected Rung: small local code
- Why This Is Enough: reuse manifests, receipts, CI jobs and standard-library helpers; no new dependencies or orchestration platform.
- Upgrade Trigger: observed drift or repeatable failures justify additional automation.

## Success Criteria

- Behaviour Outcome: new sessions can find current status and relevant contracts without broad mandatory loading; current commands are valid and workflow checks have explicit invocation roles.
- Verification Signal: targeted positive/negative tests, consumer preflights, docs and governance checks pass or retain attributable pre-existing gaps.
- Regression Watch: no original task is completed by summarization; unrelated dirty source files and original status history retain their bytes.
- Economics Watch: record actual elapsed checks, input sizes and unavailable metrics; do not infer a productivity speedup.

## Context Strategy

- Entry Sources: repository AGENTS.md, current task/index and scoped contracts.
- Retrieval Order: entry -> task/index -> summary/review -> raw evidence.
- Retrieval Helpers: CodeGraph unavailable; direct reads and literal rg.
- Promotion Target: repo entrypoints, task index and verification skills.
- Response Budget: terse
- Sensitive Context: no secrets, environment dumps or private transcripts.

## Structural Scope

- Affected Subgraph: canonical method -> active consumer entries/receipt -> CI checkout/check; manifests -> task index -> current status
- Boundary Crossings: method-source -> foundation-source/business-consumer; generic product sensors remain Base-first
- Risk Nodes: receipt writer, CI provenance, inherited checker invocation, status summarization
- Graph Focus: sensitive-input-flow

## Verification Plan

- Portable consumption/index behavioral tests and relevant strict Harness checks.
- Base command-drift/wiring tests and focused workflow regressions.
- Ops root-script assembly tests, current overlay contract, boundary/wiring checks and workflow regressions.
- Bilingual links/encoding/task metadata and skill validation.
- Original dirty-file and status-snapshot SHA256 preservation.
- Independent review with evidence; Harness PR #13 and merged main CI passed; consumer hosted checks remain pending, formal releases remain separate.
- Visual evidence: not applicable; no user-facing UI change.

## Linkage

- Task Manifest: .harness/tasks/2026-11-24-workflow-context-improvement/manifest.json
- Superpowers Plan: none
- Task ID: 2026-11-24-workflow-context-improvement
- OpenSpec Change: none
- Plan References: none
- Evidence Directory: .harness/evidence/2026-11-24-workflow-context-improvement/
- Review File: .harness/evidence/2026-11-24-workflow-context-improvement/review.md

## Evidence Required

- Commands with real exits, raw logs and hashes; behavioral negative cases.
- Independent review and preservation receipt.
- Current status size changes and explicit publication/inheritance/runtime gaps.

## Human Gates

- User has authorized local method/workflow improvements, including correction of inactive wiring; no existing required gate is waived or downgraded.
- Final functional/method acceptance and any later publication or foundation replacement remain separate decisions.

## Completion Checklist

- [x] Layer and boundary declared
- [x] Contract anchors read
- [x] Current entrypoints and state aligned
- [x] Verification run or exception recorded
- [x] Evidence saved or summarized
- [x] Review completed
- [x] Residual gaps explicit
