# pantheon-ops Business Development Plan

Updated: 2026-08-18  
Type: Plan  
Status: Proposed

Normative plan: [BUSINESS_DEVELOPMENT_PLAN.md](./BUSINESS_DEVELOPMENT_PLAN.md).

The eight-week V1 plan is ordered as follows:

1. Release baseline and versioned schema.
2. DataScope, ownership, and capability contracts.
3. Immutable target snapshots and ServiceInstance validation.
4. Queue/Worker/Lease/Attempt/Reconcile execution center.
5. Kubernetes Namespace ownership, credential references, and fine-grained permissions.
6. Cross-module QA, migration rehearsal, smoke tests, and clean release.
7. V2 observability preparation only after V1 gates pass.

P0 work blocks feature expansion. Every task must include migration, permissions, audit, tests, smoke, evidence, and rollback impact. Platform or system-domain defects must be sent to `pantheon-base` rather than copied into ops.
