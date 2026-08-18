# Task: BizScope Data Scope And Ownership

Task ID: `ops-bizscope-datascope-ownership`  
Priority: P0  
Owner: BizScope + CMDB ownership coordinator  
Depends on: `ops-cross-module-capability-boundaries`; migration changes coordinate with `ops-business-versioned-migrations`  
Blocks: ServiceInstance and end-to-end Deploy authorization

## Goal And Invariants

Make BizScope a real responsibility and authorization boundary while keeping
CMDB as the sole owner of Host facts and ownership writes.

Required invariants:

1. A non-admin request sees and mutates only BizScopes and resources allowed by
   the existing base DataScope contract.
2. A Host can have at most one active BizScope. Binding an already-owned Host is
   a conflict, never an overwrite.
3. Only the CMDB ownership Command changes Host `business_scope_id`; ordinary
   Host update, import, or Deploy code cannot change ownership.
4. Binding/unbinding is transactional, conditional, actor-attributed, and checks
   `RowsAffected`.
5. Deleting a BizScope is rejected while any provider reports an active reference;
   the check cannot race with a new bind.
6. `business_scope_id` is authoritative. `code`/`name` on Host/Cluster/Task are
   display snapshots only and are never used as identity or authorization.

## Why This Is P0

The current service ignores the BizScope list data scope, allows last-writer-wins
host binding, permits Host Update to bypass ownership rules, and checks deletion
references only through a direct CMDB query. A user with broad Deploy permission
could otherwise operate across responsibility domains.

## Preflight Evidence

- `backend/modules/business/bizscope/bizscope_service.go:45` (`List` scope path),
  `:191` (delete reference check), `:307` (batch bind), and `:376` (`scopedHostsQuery`).
- `backend/modules/business/cmdb/host/host_service.go:180` and `:389` for update
  and status paths; confirm current ownership field handling.
- `backend/modules/business/cmdb/host/host_handler.go` and import path for all
  write entry points.
- Base `common.DataScopeReq`, `database.WithDataScope`, and existing system data
  scope tests. Do not invent a new IAM model before checking base.

## In Scope

- Add the smallest ownership attribute needed for BizScope filtering. Recommended
  baseline: `dept_id` on `biz_business_scope`, aligned with Host `dept_id` and base
  DataScope. If current base contracts require a different owner mapping, record
  the exact substitution before editing; do not add tenant/member tables here.
- Apply data scope consistently to BizScope list/options/get/export and all
  reference/ownership commands. Admin/all scope remains compatible with base.
- Implement provider APIs from the capability task: active/visible scope lookup,
  batch code resolution, CMDB available/bound listing, reference check, and
  ownership bind/unbind commands.
- Use a transaction with a lock or conditional update:
  `WHERE id IN (...) AND deleted_at IS NULL AND business_scope_id = 0` for bind,
  and expected-owner/version conditions for unbind. A partial batch must fail
  atomically unless the public contract explicitly returns per-item results.
- Prevent new binds while scope deletion is being finalized. Use the smallest
  same-database coordination primitive already supported (transaction lock,
  tombstone/status transition, or provider coordinator); document the race proof.
- Reject Host ordinary Update requests that contain ownership changes; expose a
  dedicated command path used by BizScope orchestration.
- Return canonical conflict/not-found/inactive/forbidden errors and audit actor,
  scope, host IDs, action, and result using base audit contracts.
- Keep current UI routes; filter scope and host selectors server-side and surface
  conflict/forbidden errors. Add confirmation for unbind/delete if current page
  lacks it.

## Out Of Scope

- tenant/project/application hierarchy;
- a general membership/ACL subsystem in `system/iam`;
- direct cross-module joins, shared repositories, or cross-module FKs as the
  primary authorization mechanism;
- ServiceInstance lifecycle actions;
- changing historical task/release snapshots after a scope rename;
- redesigning base DataScope.

## API/DTO Contract

The provider must distinguish:

- `GetActive` vs `GetVisible`: inactive scopes are not valid deployment targets;
- `Bind`/`Unbind` commands vs readers;
- conflict (`host.alreadyAssigned`), forbidden (`scope.forbidden`), inactive,
  not-found, and stale-owner/version errors.

Request DTOs carry actor/data scope, expected owner/version, and an idempotency or
correlation value. Responses expose IDs and canonical scope refs, not GORM models.
External handlers remain thin and keep project pagination/error response style.

## Expected Files

- `backend/modules/business/bizscope/bizscope_model.go`, `bizscope_dto.go`,
  `bizscope_service.go`, `bizscope_handler.go`, `module.go`;
- `backend/modules/business/cmdb/host/host_model.go`, `host_service.go`,
  `host_handler.go`, `host_import_export.go`, `cmdb/capability.go`;
- `backend/modules/business/k8s/cluster/cluster_service.go` only for reference
  API integration, not K8s object redesign;
- versioned migration files coordinated with task 1;
- focused backend tests, API contract tests, and CMDB/BizScope frontend selectors.

Do not touch `../pantheon-base/**`, system IAM tables, Deploy execution internals,
or unrelated K8s pages.

## Verification Matrix

| Scenario                               | Required result                                                                 |
| -------------------------------------- | ------------------------------------------------------------------------------- |
| non-admin list/get/export              | only permitted scope rows and host counts                                       |
| inactive scope as Deploy target        | rejected by provider API                                                        |
| bind pending unowned Host              | succeeds once and records ownership                                             |
| two concurrent binds                   | one succeeds; one deterministic conflict; no overwrite                          |
| bind already-owned Host                | rejected and original owner unchanged                                           |
| ordinary Host Update with scope field  | rejected or scope field ignored with explicit error; no ownership change        |
| unbind wrong owner/version             | stale conflict; no mutation                                                     |
| delete with Host/K8s/Service reference | rejected through provider reference APIs                                        |
| delete racing with bind                | no new active bind after deletion commit                                        |
| rename/code change                     | ID remains identity; projections are documented as snapshot/current API display |

Commands:

```text
go test -count=1 ./modules/business/bizscope/...
go test -count=1 ./modules/business/cmdb/...
go test -race ./modules/business/bizscope/... ./modules/business/cmdb/...
go vet ./...
```

Add an integration/concurrency test using MySQL where SQLite semantics would be
misleading. Write evidence to `.harness/evidence/ops-bizscope-datascope-ownership/`.

## Stop Conditions

Stop if implementing scope authorization requires a new system IAM contract, if
delete/bind race proof needs distributed transactions, or if existing data cannot
be mapped from current snapshots to an authoritative ID without a migration
decision. Record the smallest safe alternative and affected child tasks.
