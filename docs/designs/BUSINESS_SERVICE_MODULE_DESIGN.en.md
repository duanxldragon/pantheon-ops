# Business Service Module Design

- Updated: 2026-08-24
- Status: Active V1 business identity layer

The Chinese document is normative. See the [global functional design](./BUSINESS_FUNCTIONAL_DESIGN.md) and [V1 checklist](../V1_RELEASE_CHECKLIST.md) for cross-module and release status.

## Goal

Provide the minimum business-owned identity layer:

`BizScope -> Application -> Service -> ServiceInstance`

This module is not CMDB and does not store runtime object copies.

## Ownership

- `Application` owns business scope and department context.
- `Service` belongs to one `Application`.
- `ServiceInstance` belongs to one `Service` and carries one runtime target.

## Target Rules

- VM instance target: exactly one `host_id`.
- K8s instance target: exactly one `(cluster_id, namespace, workload_kind, workload_name)`.
- Mixed or missing target shapes are rejected.
- Target resolution goes through owner capability APIs only.
- Target ownership and DataScope must match the parent Application/BizScope.

## State Model

ServiceInstance keeps desired, observed, and health state independently. Every transition uses `lifecycle_version`/CAS and a transition identifier. Deploy results update observed/health through `ServiceInstanceStateCommand`; failures cannot leave a false running projection. Reconcile observes owner facts and never creates a new user intent.

## Compatibility

- Host `installed_components` is legacy inspection data only.
- Service foundation does not dual-write host JSON.
- Any later migration must be deterministic and explicit.

## API Surface

- `/business/service/applications`
- `/business/service/services`
- `/business/service/instances`
- `/business/service/instances/:id/state`
- `/business/service/instances/:id/reconcile`

The module exposes typed Reader/Command interfaces in-process.

## Verification

- `go test -count=1 ./modules/business/service/...`
- `go test -race ./modules/business/service/...`
- `go vet ./...`
- `npm run type-check`
- `npm run build`
- `npm run check:menu-contract`
- `node scripts/check-business-module-boundaries.mjs`
