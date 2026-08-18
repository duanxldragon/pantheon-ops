# Task: Cross-Module Capability Boundaries

Task ID: `ops-cross-module-capability-boundaries`  
Priority: P0  
Owner: business module integration owner  
Depends on: none for contract design; migration task for schema-affecting changes  
Blocks: BizScope ownership, Deploy reliability, ServiceInstance, K8s reliability

## Goal And Invariant

Enforce the rule requested by the maintainer:

```text
一个业务模块不能查询另一个业务模块的表；跨模块读取必须通过统一的
provider-owned typed API，跨模块写入必须通过 provider-owned Command API。
```

In the current monolith, “API” means an injected Go interface with stable DTOs;
it does not mean a loopback HTTP call. REST/gRPC is reserved for a future process
boundary. Provider APIs must reapply authorization, status, soft-delete, and
audit rules.

## Why This Is P0

The audit found bidirectional CMDB/BizScope schema coupling and Deploy/K8s paths
that bypass the existing `DeployCMDBCapability`. This is a security and data
ownership defect, not a cosmetic refactor. A new feature must not add another
direct-table path while old ones are being removed.

## Evidence To Revalidate

- CMDB -> BizScope: `backend/modules/business/cmdb/host/host_service.go:734`
  and `host/host_import_export.go:233`.
- BizScope -> CMDB: `backend/modules/business/bizscope/bizscope_service.go`,
  especially `ListOptions`, `Get`, `Delete`, `ListBoundHosts`,
  `ListAvailableHosts`, `BindHosts`, `UnbindHost`, and `scopedHostsQuery`.
- Deploy -> BizScope: `deploy/deploy_service.go:424`, `resolveDeployScopeName`.
- Deploy -> CMDB: `deploy/deploy_service.go:1131`, `taskHasVisibleTaskHost`,
  and `:1870`, `upsertHostInstalledComponent`.
- Existing direction to preserve: `cmdb/capability.go:55`,
  `DeployCMDBCapability`.
- K8s -> BizScope: `k8s/cluster/cluster_service.go:332`,
  `ClusterService.getBusinessScope`.

Re-run searches against current code because the worktree is dirty.

## In Scope

1. Define stable, narrow DTOs and interfaces for:

   - `BizScopeReader`: visible/active lookup by ID and batch active lookup by code;
   - `CMDBHostReader`: deploy target resolution, scoped/available host listing,
     ownership/reference checks, and stable host snapshots;
   - `CMDBOwnershipCommand`: bind/unbind ownership with CAS and actor metadata;
   - optional `K8sClusterReferenceReader` for deletion/reference protection.

2. Resolve package-cycle risk deliberately. Prefer provider-owned contracts or a
   small business contract package that contains DTOs only. Never solve a cycle by
   importing a provider's GORM model into the consumer.
3. Inject implementations at `business/*/module.go`, `business/business.go`,
   `cmdb/cmdb.go`, `deploy/deploy.go`, and `k8s/k8s.go` wiring points.
4. Replace all runtime direct cross-module Model/table/Join access in the evidence
   list. Keep CMDB internal group/host queries because they share the CMDB owner.
5. Add architecture/contract tests and a static allowlist checker for runtime
   packages. Migration, seed, and isolated fixture paths may use tables only when
   explicitly allowlisted and never called by request handlers.
6. Preserve current external REST routes and response shapes unless a contract
   change is necessary; document any changed error key or DTO field.

## Out Of Scope

- forcing HTTP between same-process modules;
- changing `pantheon-base` or system/platform contracts;
- implementing full BizScope data-scope rules, Deploy leases, or ServiceInstance;
- exposing GORM models, `*gorm.DB`, table names, raw SQL, or JSON internals in a
  cross-module contract;
- a universal “resource query” API or a shared database repository.

## Contract Requirements

Reader DTOs must contain stable identifiers and display snapshots only; they must
not leak credentials or mutable provider models. Every method accepts actor/data
scope context where the provider must enforce visibility.

Suggested shape (adapt names to existing conventions without widening scope):

```go
type BizScopeRef struct { ID uint64; Code, Name, Environment, Status string }
type BizScopeReader interface {
    GetActive(ctx context.Context, id uint64, scope *common.DataScopeReq) (BizScopeRef, error)
    ResolveActiveByCodes(ctx context.Context, codes []string, scope *common.DataScopeReq) (map[string]BizScopeRef, error)
}

type CMDBHostReader interface {
    ResolveDeployTargets(ctx context.Context, req ResolveTargetRequest) ([]HostTarget, error)
    ListByBusinessScope(ctx context.Context, req HostScopeQuery) (HostPage, error)
    ListAvailable(ctx context.Context, req AvailableHostQuery) (HostPage, error)
    HasBusinessScopeReferences(ctx context.Context, id uint64) (bool, error)
}

type CMDBOwnershipCommand interface {
    Bind(ctx context.Context, req BindOwnershipRequest) error
    Unbind(ctx context.Context, req UnbindOwnershipRequest) error
}
```

Each command specifies actor, data scope, expected current owner/version, and
idempotency or conflict behavior. Error keys must be stable and provider-owned.
Batch methods are required where a page or import would otherwise create N+1
calls.

## Expected Files

Likely create/modify after confirming current architecture:

- `backend/modules/business/cmdb/capability.go` or a DTO-only contract package;
- `backend/modules/business/bizscope/bizscope_service.go`, `module.go`;
- `backend/modules/business/cmdb/host/host_service.go`, `host_import_export.go`;
- `backend/modules/business/deploy/deploy.go`, `deploy_service.go`;
- `backend/modules/business/k8s/cluster/cluster_service.go`, `k8s.go`;
- `backend/modules/business/business.go` and module wiring;
- focused contract tests plus an architecture checker under `scripts/`.

Do not touch `../pantheon-base/**`, system/platform modules, unrelated frontend,
or child-task-specific state/migration implementations.

## Verification Matrix

- Static scan proves forbidden imports, table names, `Table(...)`, and cross-owner
  joins are absent from production request paths, with migration/seed/fixture
  allowlist exceptions tested.
- Provider contract tests cover active/inactive/not-found, data-scope denied,
  soft-deleted, conflict, and stable error-key behavior.
- Caller unit tests use fakes, not a second direct database query.
- Cross-module smoke covers BizScope -> CMDB bind/list -> Deploy target resolve;
  inactive or unauthorized scope is denied server-side.
- `go test -count=1 ./modules/business/...`, `go test -race ./modules/business/...`,
  and `go vet ./...` pass from `pantheon-ops/backend`.

## Evidence And Stop Conditions

Write `.harness/evidence/ops-cross-module-capability-boundaries/` with scan output,
contract tests, dependency diagram, review, and residual risks. Stop if an API
would require a base-owned authorization contract, if a package cycle cannot be
resolved without a shared DTO package, or if a caller needs an unbounded query
that would turn the provider into a generic database proxy.
