---
title: Code Quality and Security Governance Strategy
doc_type: Design
layer: platform
status: Active
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
  - docs/contracts/DOCUMENT_GOVERNANCE_CONTRACT.md
updated_at: 2026-06-08
---

# Code Quality and Security Governance Strategy

Chinese version: [QUALITY_AND_SECURITY_STRATEGY.md](./QUALITY_AND_SECURITY_STRATEGY.md)

This document defines how Pantheon uses code-quality and security tools so GitHub Actions, CodeQL, and GitHub-native AI review each keep a clear role without competing for merge authority.

## Principles

Pantheon uses one primary merge-gate system and one primary code-security signal.

- GitHub Actions plus Branch Protection is the merge authority.
- CodeQL is the primary code-security signal.
- Copilot review can add architectural, intent, and maintainability signals, but it does not replace required checks.
- Codacy is no longer part of the merge gate.
- Block only security, correctness, shared-foundation stability, and regression risks. Low-risk style or maintainability suggestions should go to backlog.

Tool precedence is fixed:

```text
Correctness: GitHub Actions tests, builds, and contract checks
Security reachability: CodeQL plus secret, dependency, and workflow posture scans
Architecture and maintainability judgment: Copilot review plus PR residual-risk notes
External comparison: none
```

## Repository Tiers

### pantheon-base

`pantheon-base` is the foundation for future business repositories and follows the stricter tier.

Merge blockers:

- GitHub required checks are failing.
- CodeQL reports a reachable high-severity security issue.
- High-risk areas change without explanation: `system/auth`, `system/iam`, `system/config`, permissions, audit, shared `pkg/*`, generator, CI, or credential handling.
Targets:

- New Code coverage at or above `80%` by default, with PR-level exceptions only.
- High-risk changes must document residual risk, rollback steps, and added verification instead of depending on extra human approvals.

Frontend duplication is the current main hotspot. Fix runtime pages, shared components, layouts, forms, tables, and generator templates first. Clean smoke, fixture, and example duplication when it pollutes runtime quality signals or repeated security triage. Vendored method-shell scripts such as `scripts/harness/` are excluded from product duplication and governed in `pantheon-harness`. Do not create premature abstractions for isolated business leaf logic.

### pantheon-ops

`pantheon-ops` is a business repository. It inherits base security and architecture boundaries without duplicating every base governance document.

Rules:

- Generic backoffice foundation problems flow back to `pantheon-base`.
- Business modules evolve inside `business/*` and must not intrude into system domains.
- Auth, IAM, config, permission, audit, CI, and credential changes follow the base high-risk standard.

Targets:

- Core business-path changes need tests, smoke evidence, or acceptance evidence.
- Low-risk duplication inside business leaf modules can be tracked as backlog, but must not spread into shared packages, platform shell, or system domains.

## Tool Responsibilities

GitHub Actions is the primary gate for backend tests, frontend lint/build, contract checks, documentation checks, menu checks, i18n checks, and generated-module checks. Branch Protection should require only GitHub-native checks. Do not add Codacy or other external scanner checks to required checks.

CodeQL is the primary code-security signal. In `pantheon-base`, CodeQL findings block by default unless they are documented false positives. In `pantheon-ops`, business-domain findings can be risk-tiered, but high-risk foundation areas still follow the base standard.

A successful CodeQL analysis job does not prove there are zero open alerts. `Security Gates` must record an open-alert report. Existing high/critical baseline alerts are report-only for PR and merge queue, then enforced on protected-branch push, scheduled security review, or manual security review.

Dependency, secret, and workflow-posture scans complement CodeQL. New dependencies, dependency upgrades, workflow changes, and credential-handling changes must review these reports. Real secret leaks, reachable high-severity dependency issues, and dangerous workflow permissions must be fixed immediately.

Copilot review is the preferred GitHub-native supplemental signal for what scanners do not prove: architecture boundaries, business intent, test adequacy, and exception handling. Copilot comments do not count as approvals and must not replace required checks.

## Exceptions

Exceptions are allowed only with lightweight traceability:

- record the reason in the PR
- state whether `pantheon-base` foundation stability is affected
- record follow-up testing, cleanup, or refactoring
- require residual-risk and rollback notes for high-risk exceptions

No exception is allowed for confirmed reachable high-severity vulnerabilities, unreviewed Security Hotspots, real secret leaks, or changes that break authentication, authorization, audit, or secure configuration boundaries.

## PR Evidence

Each PR should record GitHub required-check status, CodeQL status or link, Copilot review status, high-risk scope, residual risk, and rollback notes when required.
