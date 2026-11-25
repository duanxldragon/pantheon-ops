---
title: Pantheon Base Multi-Agent Delivery Workflow
doc_type: Acceptance
layer: platform
status: Active
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
  - docs/contracts/DOCUMENT_GOVERNANCE_CONTRACT.md
updated_at: 2026-11-23
---

# Pantheon Base Multi-Agent Delivery Workflow

Chinese version: [PANTHEON_BASE_DELIVERY_WORKFLOW.md](./PANTHEON_BASE_DELIVERY_WORKFLOW.md)

This document consumes [Capability-Adaptive Delivery](../../../pantheon-harness/patterns/capability-adaptive-delivery.md). The primary agent plans, explores, implements, verifies, and fixes in one session; current runtime tools, authorization, and task risk determine delegation.

The goal is that the human states goals and key decisions, while the active dispatcher routes tools and records repository artifacts.

## Roles

The primary agent performs Planner/Explorer/Executor responsibilities sequentially within the authorized scope. Planning-only and review-only assignments stay read-only. Ordinary L1 may use findings-first self-review; L2 retains review artifacts and high-risk work uses an independent reviewer or an explicit review gap and acceptance gate. Dispatcher responsibilities apply only when delegating, with ownership, budget, evidence, stop condition, and integration owner. Mechanical gates do not replace authorization or risk acceptance.

## Default Flow

```text
Goal -> tier/capability/scope/acceptance -> focused exploration/plan
     -> implement/verify/fix in the primary context
     -> tier-appropriate review and evidence -> final acceptance
     -> PR/release and branch closure only when requested -> ratchet/closeout
```

## Tool Policy

Preserve current model/effort/provider/settings. Record required actual tools, sandbox, delegation and recovery capabilities, with unknowns stated. Native subagents, acpx, omc, and CI matrix jobs are optional adapters for self-contained branches with a concrete benefit. They are not mandatory prerequisites or vendor-fixed role routing. Background helpers follow workspace Windows hidden-launch rules.

GitHub Actions are mechanical gates. Quality Gates and Security Gates remain PR-required. CodeQL is the primary security signal; review and residual-risk notes cover intent/architecture gaps. Full Smoke Suite remains a deep advisory/scheduled/manual signal and does not replace focused evidence. Worktree/PR rules remain in docs/designs/WORKFLOW.md.

## Human Gates

Authorized, reversible changes to schema descriptions, permissions, authentication, audit and CI proceed autonomously within the agreed task scope. The maintainer intervenes at three touchpoints: initial requirements, quality-gate policy decisions, and final visual/functional acceptance. Stop only for unapproved destructive/production or credentialed external actions, actual publication, material scope expansion, or a change to required/advisory gate policy. Do not request the same authorization again merely because a sensitive domain is touched.

## Minimum Closeout

### Preserve original archive bytes

Historical task/evidence files and preservation snapshots under `.harness/archive/`
retain their original bytes and recorded hashes. The root `.prettierignore` protects
the entire directory from default Prettier and lint-staged writes; active documents
remain formatted. Do not replace or disable the ignore file to bulk-format archives.
Record new facts in current evidence or an added closeout note while retaining the
original files and hashes.

`node --test tests/scripts/harness-archive-formatting.test.mjs` demonstrates the
unprotected defect and checks worktree/index bytes through normal lint-staged.
Ops receives this shared configuration through a formal Base release and consumer
verification; copying the local Base file is not an inheritance upgrade.

L2 local work needs its task packet/existing manifest, evidence, review result, gaps, and ratchet decision. The following publication fields apply when PR/release closeout is requested; ordinary L1 may use a short plan and verification summary:

- task packet or parent task packet linkage
- task manifest path
- evidence path or command summary
- reviewer role and review result
- GitHub signal classification
- merged PR URL and merge commit
- branch cleanup status
- known gaps
- ratchet decision

Missing required evidence or review stays an explicit gap. Unrequested PR/publication is not a local implementation stop condition.

## Planning Expectations

Planner output should include at least:

- In / Out / Do Not Touch
- Assumptions and Open Questions
- Minimum Viable Approach
- Success Criteria
- the minimum verification set
- task manifest / evidence / review linkage

L2 artifacts land in the task packet and existing task manifest; ordinary L1 does not require a new full manifest.

## Execution Expectations

The primary agent or an exclusively owned execution branch keeps scope and evidence aligned. Check `.harness/method-consumption.json` before work and at closeout with `node ../pantheon-harness/scripts/harness/check-method-consumption.mjs --root . --source ../pantheon-harness`. Review intentional changes before adding --update. Method consumption is independent of product VERSION; shared Base product/config/checker changes still reach Ops through a formal foundation release.
