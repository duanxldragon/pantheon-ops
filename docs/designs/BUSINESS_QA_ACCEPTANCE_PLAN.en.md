---
title: pantheon-ops QA Acceptance Plan
doc_type: Acceptance
layer: business
status: Draft
linked_contracts:
  - docs/PROJECT_INHERITANCE.md
updated_at: 2026-08-18
---

# pantheon-ops QA Acceptance Plan

Normative acceptance questions: [BUSINESS_QA_ACCEPTANCE_PLAN.md](./BUSINESS_QA_ACCEPTANCE_PLAN.md).

Release is blocked by any P0 failure. The acceptance matrix covers:

- Empty/upgrade/repeat/failure-recovery MySQL migrations.
- Cluster and Namespace DataScope/ownership enforcement.
- Cross-module boundary checks and capability DTOs.
- Idempotent create/start/callback, lease takeover, cancellation, and worker recovery.
- ServiceInstance target consistency and immutable deployment snapshots.
- Kubernetes resource-version conflicts, rollout reconciliation, secret redaction, and credential rotation.
- Pagination, permissions, audit, frontend states, and rollback evidence.

P1 exceptions require an explicit owner, deadline, risk, and compensating control.
