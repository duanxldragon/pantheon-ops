# Business Scope Module Design

Chinese version: [BUSINESS_BIZSCOPE_MODULE_DESIGN.md](./BUSINESS_BIZSCOPE_MODULE_DESIGN.md)

Updated: 2026-06-18

Type: Design
Layer: business/bizscope
Status: Active

This document defines the `business/bizscope` module in `pantheon-ops`, including business boundary, data model, API, UI surface, and acceptance expectations. The module governs business scopes themselves and acts as the canonical trust source for CMDB host assignment and Deploy task scope selection.

## 1. Module Overview

`business/bizscope` is an operations-domain business module. It is a peer of `business/cmdb` and `business/deploy`, not a CMDB submodule.

It is responsible for:

- maintaining the business-scope registry with `code / name / owner / environment / status`
- providing the canonical host-assignment boundary for CMDB hosts
- providing the trusted `business_scope_id / business_scope_name` source used by Deploy
- showing scope details and host-count overview for ownership checks

It is not responsible for:

- host inventory, grouping, or label governance, which belong to `business/cmdb`
- software packages, templates, deployment orchestration, or execution, which belong to `business/deploy`
- users, roles, menus, organization, dictionaries, or shared audit infrastructure, which belong to `system/*`
- shared shell or UI rules, which belong to `pantheon-base`

## 2. Boundary and Dependencies

| Area | Allowed | Forbidden |
| :--- | :--- | :--- |
| platform | shared response helpers, module registration, menu and i18n seed contracts | redefining platform-shell rules locally |
| system/auth | login principal and JWT session context | direct auth-service dependency |
| system/iam | page permissions, button permissions, Casbin results | local permission-policy redefinition |
| system/org | data-scope context | direct org-repository dependency |
| business/cmdb | maintaining `business_scope_*` host snapshot fields | taking over CMDB inventory, labels, or groups |
| business/deploy | acting as the task-scope source | bypassing business-scope validation for task targets |

Boundary rules:

- `business/bizscope` is the single source of truth for business-scope definition.
- host bind/unbind operations may update `biz_cmdb_host.business_scope_id/code/name` and necessary status rollback only.
- Deploy must use business scope as its scope filter and audit context.

## 3. Data Model

Current table: `biz_business_scope`

| Field | Type | Notes |
| :--- | :--- | :--- |
| `id` | bigint | primary key |
| `code` | varchar(255) | unique scope code |
| `name` | varchar(255) | scope name |
| `owner` | varchar(255) | owner |
| `environment` | varchar(50) | `dev` / `test` / `prod` |
| `status` | varchar(50) | `active` / `inactive` |
| `remark` | text | remark |
| `created_at / updated_at / deleted_at` | datetime | audit fields |

Related host snapshot fields:

- `biz_cmdb_host.business_scope_id`
- `biz_cmdb_host.business_scope_code`
- `biz_cmdb_host.business_scope_name`

Rules:

- `code` must stay globally unique.
- a scope cannot be deleted while hosts are still bound to it.
- unbinding a host rolls `assigned` back to `pending`, but does not downgrade already-online hosts.

## 4. Business Flows

### 4.1 Scope Maintenance

```text
Create scope
  -> fill code / name / owner / environment / status / remark
  -> validate unique code
  -> persist into biz_business_scope
```

```text
Update scope
  -> modify governance or display fields
  -> keep code uniqueness if code changes
  -> do not retroactively rewrite old deploy snapshots
```

### 4.2 Host Binding

```text
Select scope
  -> query available hosts with empty business_scope_id
  -> bind hostIds in batch
  -> write back business_scope_id/code/name
  -> move host state from pending to assigned
```

### 4.3 Host Unbinding

```text
Select a bound host under a scope
  -> clear business_scope_id/code/name
  -> if host state is assigned, roll back to pending
  -> if host is already online, do not force a runtime downgrade here
```

## 5. API Design

Prefix: `/api/v1/business/bizscope`

| Method | Path | Purpose | Permission |
| :--- | :--- | :--- | :--- |
| `GET` | `/list` | paged scope list | `business:bizscope:list` |
| `GET` | `/options` | scope select options | `business:bizscope:list` |
| `GET` | `/:id` | scope detail | `business:bizscope:view` |
| `GET` | `/:id/hosts` | bound host list | `business:bizscope:view` |
| `GET` | `/:id/available-hosts` | bindable host list | `business:bizscope:view` |
| `POST` | `/:id/hosts/bind` | bind hosts | `business:bizscope:update` |
| `DELETE` | `/:id/hosts/:hostId` | unbind one host | `business:bizscope:update` |
| `POST` | `` | create scope | `business:bizscope:create` |
| `PUT` | `/:id` | update scope | `business:bizscope:update` |
| `DELETE` | `/:id` | delete scope | `business:bizscope:delete` |

Local error summary:

| Key | Meaning |
| :--- | :--- |
| `bizscope.code_exists` | duplicate scope code |
| `bizscope.in_use` | scope still has bound hosts |
| `bizscope.not_found` | scope not found |
| `param.invalid` | invalid request payload |

Repository-wide canonical error semantics still live in `BUSINESS_ERROR_SEMANTICS_APPENDIX.md`.

## 6. Menus, Routes, and Permissions

Current menu and route anchors:

| Item | Value |
| :--- | :--- |
| menu title key | `operations.bizscope.menu` |
| list route | `/operations/business-scope` |
| detail route | `/operations/business-scope/:id` |
| list component key | `business/bizscope/BizScopeList` |
| detail component key | `business/bizscope/BizScopeDetail` |
| module name | `business.bizscope` |

Permission keys:

- `business:bizscope:list`
- `business:bizscope:view`
- `business:bizscope:create`
- `business:bizscope:update`
- `business:bizscope:delete`

Rules:

- list entry and detail entry stay separated: the current list route uses `list`, while detail uses `view`
- host bind/unbind operations stay under `update`
- delete must not imply read or update permissions

## 7. Frontend Surface

Current frontend module path: `frontend/src/modules/business/bizscope`

### 7.1 `BizScopeList`

- hero + filters + table + pagination
- hero metrics: total scopes, active scopes, production scopes
- filters: `code`, `name`, `owner`, `environment`, `status`
- columns: code, name, owner, environment, status, remark, actions
- actions: detail, edit, delete, plus batch delete when allowed

### 7.2 `BizScopeForm`

- used for create and update
- fields: code, name, owner, environment, status, remark
- environment values: `dev / test / prod`
- status values: `active / inactive`

### 7.3 `BizScopeDetail`

- shows code, name, owner, environment, status, remark
- the detail API already returns `hostCount`; if the detail page needs expansion, reuse this response instead of introducing a parallel summary API

### 7.4 UI Rules

- keep using base system-page patterns, state pages, theme tokens, and responsive rules
- do not invent local visual rules in ops
- when touching UI here, still follow `pantheon-base` design references first

## 8. i18n and Audit

Namespace: `business.bizscope`

Important existing keys include:

- `operations.bizscope.menu`
- `operations.bizscope.detail`
- `business.bizscope.hero.*`
- `business.bizscope.field.*`
- `business.bizscope.environment.*`
- `business.bizscope.status.*`
- `business.bizscope.permission.*`
- `business.bizscope.audit.*`

Audit actions:

- `business.bizscope.audit.create`
- `business.bizscope.audit.update`
- `business.bizscope.audit.delete`

Notes:

- host bind/unbind currently reuse `business.bizscope.audit.update`
- if binding and field-editing need distinct audit meaning later, add explicit audit keys and acceptance instead of keeping the action overloaded

## 9. Collaboration with CMDB and Deploy

- a CMDB host without a business scope is considered unassigned inventory
- Deploy task creation must use a valid business scope as scope filter and audit context
- `business/bizscope` is the boundary anchor between `business/cmdb` and `business/deploy`; neither side should redefine business scope locally

## 10. Acceptance and Cleanup Expectations

Minimum acceptance:

- list, detail, create, update, and delete all work
- duplicate code returns a clear error
- scopes with bound hosts cannot be deleted
- host bind/unbind correctly maintain `biz_cmdb_host.business_scope_*`
- menu, permission, i18n, and page entry stay aligned

Documentation governance:

- `docs/README.md` and the repo README must list `business/bizscope` as a formal business-module entry
- if this module expands later, update this design doc first instead of leaving BizScope behavior described only indirectly inside CMDB or Deploy docs
