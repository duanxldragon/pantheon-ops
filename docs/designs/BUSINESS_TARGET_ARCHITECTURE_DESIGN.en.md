# pantheon-ops Target Business Architecture

Updated: 2026-08-18  
Type: Design  
Status: Proposed

Normative design: [BUSINESS_TARGET_ARCHITECTURE_DESIGN.md](./BUSINESS_TARGET_ARCHITECTURE_DESIGN.md).

The target architecture keeps a logical modular monolith first and postpones physical service decomposition. The business domains are:

- CMDB Resource: identity, lifecycle, ownership, relationships, and discovery summaries.
- Business Catalog: `BizScope -> Application -> Service -> ServiceInstance -> TargetRef`.
- Change/Execution: `ChangeIntent -> ExecutionPlan -> Queue -> Worker -> Attempt -> Reconcile`.
- Kubernetes provider: clusters, credential references, namespace bindings, object references, and release revisions.
- Observability references: links, labels, ownership, and summaries; raw metrics/logs/traces stay in their native systems.

Cross-module request paths use explicit capabilities and stable DTOs. Every mutation requires DataScope, ownership, permission, resource-version handling, and audit evidence. V1 does not split every module into microservices or copy time-series data into CMDB.
