---
title: Kubernetes Business Module Design
doc_type: Design
layer: business/k8s
status: Active
linked_contracts:
  - docs/PROJECT_INHERITANCE.md
updated_at: 2026-08-24
---

# Kubernetes Business Module Design

The [global functional design (Chinese canonical)](./BUSINESS_FUNCTIONAL_DESIGN.md) and [V1 checklist](../V1_RELEASE_CHECKLIST.md) govern implementation and release status.

> Status note: Active; Service CRUD and the external runtime matrix are V1 freeze targets

This document defines the `business/k8s` ownership boundary and delivery
contract. Authentication, Casbin, DataScope, audit, menus, i18n, and secret
protection remain Base system capabilities and are not duplicated here.

## Ownership And Source Of Truth

- `Cluster` is the stable registered identity of a Kubernetes cluster.
- The Kubernetes API is the runtime source of truth for Namespace, Node,
  Workload, Service, ConfigMap, and Secret objects. The module returns live DTOs and
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

`NamespaceBinding` explicitly binds cluster/namespace to BizScope, environment, and allowed actions. Every namespace mutation validates Token, Casbin, DataScope, Cluster ownership, and this binding before calling Kubernetes.

## V1 Freeze Additions

- ClientPool never writes under `RLock`, keeps slow parsing/network work outside the global write lock, and coordinates replace/rotation/cleanup consistently.
- Kubernetes Service supports structured list/get/create/update/delete with type, selectors, and ports.
- Service update/delete requires `resourceVersion`; stale writes return HTTP 409 with a stable error key.
- Secret reads expose keys and metadata only, never values.
- The isolated Nginx/Redis external matrix records authorization failures, conflicts, timeouts, and cleanup.

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
