# Kubernetes Business Module Design

This document defines the `business/k8s` ownership boundary and delivery
contract. Authentication, Casbin, DataScope, audit, menus, i18n, and secret
protection remain Base system capabilities and are not duplicated here.

## Ownership And Source Of Truth

- `Cluster` is the stable registered identity of a Kubernetes cluster.
- The Kubernetes API is the runtime source of truth for Namespace, Node,
  Workload, ConfigMap, and Secret objects. The module returns live DTOs and
  does not mirror those objects into CMDB.
- `Release` is an immutable image-change intent and rollout observation record,
  not a Workload copy.
- VM Deploy and K8s Release share future Service identity and audit semantics,
  never an executor.

## Boundaries

K8s validates business-scope ownership only through the provider-owned
`business/capability.BizScopeReader` typed interface. Cluster deletion asks the
K8s-owned Release reference capability while holding the same Cluster row lock
used by Release intent creation, closing the create/delete race.

## Release Lifecycle

```text
pending -> applying -> succeeded
                    -> failed
                    -> timed_out -> succeeded | failed
```

Intent is committed before any Kubernetes mutation. Success requires the
requested image, accepted target generation, and valid rollout conditions for
Deployment, StatefulSet, or DaemonSet. `timed_out` is recoverable only through
observation; reconciliation never blindly reapplies a change.

The same idempotency key plus the same immutable snapshot returns the existing
Release. A different snapshot conflicts. A missing key is generated once for
legacy single submissions and is not retry deduplication.

## Execution, Security, And Limits

All API routes remain behind Base Token, Casbin, and DataScope middleware.
Release create, rollback, and reconcile have independent business permission
and audit keys. UI permission checks are convenience only; server-side Casbin
remains the enforcement point.

Kubeconfig and Secret values never appear in Release DTOs, audit snapshots, or
the UI. The module does not implement Namespace ownership for shared clusters,
GitOps, Helm, a universal CMDB object mirror, or a merged VM/Kubernetes
executor.

The legacy production AutoMigrate-to-versioned-table conversion is explicitly
deferred to the maintainer's later model-generated migration work. Existing
production databases need that follow-up migration before newly added Release
fields are available.
