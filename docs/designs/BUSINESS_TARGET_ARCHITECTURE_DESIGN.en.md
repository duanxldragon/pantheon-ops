---
title: pantheon-ops Target Business Architecture
doc_type: Design
layer: business
status: Active
linked_contracts:
  - docs/PROJECT_INHERITANCE.md
updated_at: 2026-08-24
---

# pantheon-ops Target Business Architecture

Normative design: [BUSINESS_TARGET_ARCHITECTURE_DESIGN.md](./BUSINESS_TARGET_ARCHITECTURE_DESIGN.md).

> Status note: Active target; V1/P1/P2 boundaries follow the freeze checklist and global functional design

The current V1 release is blocked. Generic Resource models, Observability, Agent, GitOps, and broad provider expansion are post-V1 targets and must not be presented as implemented V1 behavior.

The target architecture keeps a logical modular monolith first and postpones physical service decomposition. The business domains are:

- CMDB Resource: identity, lifecycle, ownership, relationships, and discovery summaries.
- Business Catalog: `BizScope -> Application -> Service -> ServiceInstance -> TargetRef`.
- Change/Execution: `ChangeIntent -> ExecutionPlan -> Queue -> Worker -> Attempt -> Reconcile`.
- Kubernetes provider: clusters, credential references, namespace bindings, object references, and release revisions.
- Observability references: links, labels, ownership, and summaries; raw metrics/logs/traces stay in their native systems.

Cross-module request paths use explicit capabilities and stable DTOs. Every mutation requires DataScope, ownership, permission, resource-version handling, and audit evidence. V1 does not split every module into microservices or copy time-series data into CMDB.
