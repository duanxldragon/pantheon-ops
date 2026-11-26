# Plan: tenant runtime enforcement, module by module

## Why this is a plan and not one change

T04 and T14 phase 1 added `tenant_id` columns. They do not enforce anything: no business module uses `tenant.Scoper` or `TenantOwnerID`, and every module except observability mounts the legacy `DataScopedGroup`. In multi-tenant mode rows are written with the default 0 and read unfiltered. A column is not isolation.

Doing the extensions alone would leave multi-tenant mode half enforced, and doing everything at once is too large to review or roll back. So it is split by module, in dependency and risk order, each independently verifiable.

## The pattern (reference: observability, already tenant-aware)

1. **Route group**: switch to `TenantScopedBusinessGroup`. It is fail-closed, so every route needs a registered permission route (`permissionroutes.Register`), otherwise a non-admin is denied.
2. **Read scope**: the service holds a request-bound tenant context (`tenant.Scoper`, bound by a `WithTenantContext` clone) and applies `Scoped(db)` to every query.
3. **Write owner**: `TenantOwnerID()` from the context, never from the request body.
4. **Background work**: tenant identity is stored on the job row and passed explicitly to authorization and providers; queue and cache keys include the tenant.
5. **Compat mode**: the scope pins `tenant_id = 0` and owner 0, so behaviour is unchanged. Every module task must prove it.

## Order

| #   | Module                                 | Routes now | Permission routes registered | Why here                                                                                            |
| --- | -------------------------------------- | ---------- | ---------------------------- | --------------------------------------------------------------------------------------------------- |
| 0   | **CI MySQL service** (backend-tests)   | -          | -                            | Two-tenant tests are real-database tests; they skip in CI today. Prerequisite for everything below. |
| 1   | service + bizscope                     | 18 + 19    | service yes, bizscope no     | Small; bizscope is the parent every other module reads through `BizScopeReader`.                    |
| 2   | cmdb (host/group/label)                | 28         | yes                          | Medium; hosts feed deploy and k8s.                                                                  |
| 3   | deploy                                 | 64         | yes                          | Largest risk: async tasks, queue recovery, attempts. T04 already pins restart recovery.             |
| 4   | k8s                                    | 71         | yes (full)                   | Large but already has full permission-route coverage (#206).                                        |
| 5   | filedist                               | 24         | no                           | Async transfer engine; conditional on the capability being enabled (T10).                           |
| 6   | cicd/argocd                            | 21         | no                           | Reconciler is background work; conditional on ArgoCD (T12).                                         |
| 7   | observability alerts and notifications | 42         | yes                          | The evaluator skips datasources with `TenantID != 0` today, so alerting only works for compat data. |

Each module task: switch the group and register the missing permission routes; bind the scope and owner; carry tenant identity into its background entry points; write two-tenant real-database tests (tenant B cannot read or write tenant A, guessed ids, export and callbacks) plus a compat-mode test; rollback is reverting that module's PR since the schema is already in place and defaults to 0.

## Background entry points

| Entry point                            | Module            |
| -------------------------------------- | ----------------- |
| deploy worker and queue recovery       | deploy (3)        |
| k8s sync, backup reconciler, release   | k8s (4)           |
| filedist lifecycle and transfer engine | filedist (5)      |
| ArgoCD reconciler                      | cicd/argocd (6)   |
| alert evaluator                        | observability (7) |

## Corrections carried from T14 phase 1

- The T01 table list was wrong; the verified list is in migration `000043`.
- `biz_application` was missed by T04 as well; it is covered by `000043`.

## Open decisions for the maintainer

- Whether the shared two-tenant fixture should live in `modules/business/shared` (test-support package) or in Base.
- Whether `check:business-boundaries` becomes a CI check (gate policy).
