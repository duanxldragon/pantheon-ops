---
title: Tenant Resource Scope Matrix Template and Global Unique-Key Registry
doc_type: Design
layer: platform
status: Active
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
  - docs/contracts/SYSTEM_CONFIG_CONTRACT.md
updated_at: 2026-09-10
---

# Tenant Resource Scope Matrix Template and Global Unique-Key Registry

Chinese version: [TENANT_RESOURCE_SCOPE_MATRIX.md](./TENANT_RESOURCE_SCOPE_MATRIX.md)

This is a deliverable of task 0 (`2026-09-10-tenant-ready-guardrails`), implementing
section 5 of the tenant-ready single-tenant design and feeding the tenant contract
design task.

> The runtime stays single-tenant. This is a governance template, not a multi-tenant
> implementation commitment; the generator's `tenant` data-scope option has been retired.

## 1. Four Resource Scope Classes

Every resource (table, config, cache, file, async job) must be classified as one of:

| Class | Definition | Tenant filter | Tenant-local unique | Examples |
| :--- | :--- | :--- | :--- | :--- |
| `platform-global` | Platform-wide, never split per tenant | never | n/a | platform menus, platform dicts, global settings, platform admins |
| `tenant-owned` | Rows belong to exactly one tenant | required | default | business documents, tenant config overrides |
| `tenant-overridable` | Platform default, tenant may override | conditional | composite key | tenant-level setting overrides |
| `derived` | Aggregates/derivatives inherit source scope | follows source | n/a | aggregates, export files, report caches |

Rules:

1. The class must be stated in the design document; "defaults to global" is not allowed.
2. Query/export/aggregate paths of `tenant-owned` / `tenant-overridable` resources must allow layering a tenant filter.
3. Reclassification is a shared-contract change and requires re-review.

## 2. Initial Inventory

| Resource | Current class | Notes |
| :--- | :--- | :--- |
| `system_user` | pending contract freeze | membership cardinality undecided |
| `system_role` / `system_menu` / `system_permission` | pending contract freeze | tenant-local replication undecided |
| `system_dept` / `system_post` | `platform-global` (leaning) | org is org, tenant is tenant |
| system settings | `tenant-overridable` candidate | canary preferred domain (`system/config`) |
| dictionaries | `tenant-overridable` candidate | platform default + tenant override |
| audit logs | `derived` (follows write source) | must keep a splittable dimension |
| export/upload objects | `derived` (follows source) | key must allow a tenant segment |
| cache keys | `derived` (follows source) | key design must allow a tenant segment |
| async jobs | `derived` (follows target) | context propagation frozen by contract |

## 3. Global Unique-Key Registry

| Table | Unique key | Current scope | Target scope | Conflict policy |
| :--- | :--- | :--- | :--- | :--- |
| `system_user` | username | platform-global | pending freeze | — |
| `system_role` | role_key | platform-global | pending freeze | — |
| `system_setting` | key (or key+scope composite) | platform-global | `tenant-overridable`: composite candidate | freeze at contract |
| business `biz_*` tables | register per DDL review | per review | per review | per review |

Rules:

1. Every new DDL review must answer "platform-global or tenant-local unique" (see the four mandatory questions in the acceptance matrix).
2. The contract-design task takes over this registry; until then this table is the single entry point.
3. Keys marked "pending freeze" must not be converted to composite keys before the contract freezes.

## 4. Tenant Identification (all pending decision)

Header, subdomain, custom domain and JWT claim are decision inputs only — none may be
implemented before the contract freezes. Missing, conflicting or untrusted tenant
context is deny-by-default for `tenant-owned` / `tenant-overridable` resources.

## 5. Ownership

| Item | Owner task |
| :--- | :--- |
| Final resource classification freeze | `2026-09-10-tenant-contract-design` |
| Unique-key conflict policy and migration order | `2026-09-10-tenant-migration-runbook` |
| First canary resource selection | `2026-09-10-tenant-canary-slice` (approved by contract reviewer) |
