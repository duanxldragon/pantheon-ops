# Business Service Module Design

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

## Compatibility

- Host `installed_components` is legacy inspection data only.
- Service foundation does not dual-write host JSON.
- Any later migration must be deterministic and explicit.

## API Surface

- `/business/service/applications`
- `/business/service/services`
- `/business/service/instances`

The module exposes typed Reader/Command interfaces in-process.

## Verification

- `go test -count=1 ./modules/business/service/...`
- `go test -race ./modules/business/service/...`
- `go vet ./...`
- `npm run type-check`
- `npm run build`
- `npm run check:menu-contract`
- `node scripts/check-business-module-boundaries.mjs`

