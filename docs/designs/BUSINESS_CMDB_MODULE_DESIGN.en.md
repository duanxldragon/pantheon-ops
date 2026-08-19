# CMDB Business Module Design

Chinese version: [BUSINESS_CMDB_MODULE_DESIGN.md](./BUSINESS_CMDB_MODULE_DESIGN.md)

Updated: 2026-07-17

Type: Design
Layer: `business/cmdb`
Status: Active

This document defines the lightweight CMDB module for `pantheon-ops`. It replaces the retired low-code validation sample and is the canonical business design for host inventory, label governance, and reusable host groups.

The module covers `business/cmdb/host`, `business/cmdb/group`, and `business/cmdb/label`. `business/bizscope` is a separate peer module and the trust source for host assignment and downstream deployment.

---

## 1. Module Overview

CMDB owns the base resource ledger used by deployment and later operations capabilities. It maintains physical hosts, VMs, and Kubernetes nodes; orthogonal labels such as `env` and `biz`; reusable label-based groups; business-scope assignment snapshots; and manual or one-shot SSH collection.

It does not own authentication, IAM, organization, system configuration, realtime liveness monitoring, full ITIL dependency topology, or persistent SSH credentials.

## 2. Boundaries and Dependencies

| Area | Allowed | Forbidden |
| :--- | :--- | :--- |
| platform | shared responses, audit metadata, data-scope context | changing platform-shell or dashboard logic |
| system/auth | principal and session context | importing auth services |
| system/iam | page/action permissions and Casbin results | implementing role policy inside CMDB |
| system/org | department ownership and data scope | importing org repositories |
| system/config | dictionary, setting, i18n, and encrypted-setting contracts | duplicating system configuration protocols |

Downstream modules consume CMDB through APIs or explicit capabilities, never direct table access. `assigned` means bound to a business scope and ready for deployment; `online` means deployment completed and the host is available for normal operations.

## 3. Core Business Objects

| Object | Purpose | Key attributes |
| :--- | :--- | :--- |
| `Host` | physical or virtual host inventory | hostname, IP, OS, SSH port, capacity, labels, business scope, installed components, status |
| `Group` | dynamically calculated host view | name, parent, AND/OR condition expression, description |
| `Label` | key-value data stored on a host | key and value |
| `LabelSchema` | governance for label keys and values | key, name, value mode, dictionary code, options, required flag, status |

Host labels and group conditions are JSON-backed; groups do not persist a physical member list. `LabelSchema` supports `free`, `enum`, and reserved `dict` value modes. Recommended keys include `env`, `biz`, `cluster`, `region`, `db_type`, and `os`, while custom keys remain allowed.

## 4. Business Flows

### 4.1 Host Entry and Configuration Collection

| Mode | Current status | Security boundary |
| :--- | :--- | :--- |
| manual entry | implemented | no credential exposure |
| SSH collection | implemented | credentials exist only for the request and are never persisted |
| Agent report | reserved | future token-authenticated, read-only reporting |

SSH collection calls `POST /api/v1/business/cmdb/hosts/:id/collect`, connects with `ssh.Dial`, runs inspection commands, updates capacity and OS fields, and releases credentials when the request ends. Windows hosts are manual-only for now; WinRM remains future work.

### 4.2 Host Group Calculation

Groups store conditions, not members. Conditions support `AND`/`OR` with `eq`, `neq`, `in`, and `notIn`. Member lists and counts must use the current `DataScopeReq`. Child groups inherit the full parent condition chain with a cross-level `AND`.

Statistics distinguish `memberCount`, deduplicated `aggregateMemberCount`, direct `childCount`, and recursive `descendantGroupCount`.

### 4.3 Host Status Transitions

```text
pending -> assigned -> online
                      \-> offline
                      \-> maintenance
```

| Status | Display meaning | Operations meaning |
| :--- | :--- | :--- |
| `pending` | pending onboarding | inventoried but not yet a formal operations target |
| `assigned` | assigned | bound to a business scope and waiting for deployment |
| `online` | operations-ready | deployment completed and eligible for normal operations tasks |
| `offline` | offline | excluded from normal operations targets |
| `maintenance` | under maintenance | excluded from automated batches unless explicitly selected |

This is an operations lifecycle state, not realtime connectivity. Future monitoring or Agent capabilities should maintain separate connectivity fields.

## 5. Data Model Design

### 5.1 Host Table

Table: `biz_cmdb_host`

| Field | Type | Notes |
| :--- | :--- | :--- |
| `id` | bigint | primary key |
| `hostname` | varchar(128) | required |
| `ip` | varchar(45) | required, indexed |
| `ssh_port` | int | default `22` |
| `os` / `os_version` | varchar | OS family and version |
| `cpu_cores` | int | collected or manual |
| `memory_gb` / `disk_gb` | decimal | collected or manual |
| `label_values` | json | host label instances |
| `installed_components` | json | downstream component snapshots |
| `status` | varchar(32) | default `pending` |
| `business_scope_id` | bigint | business-scope binding snapshot, indexed |
| `business_scope_code` | varchar(64) | business-scope code snapshot |
| `business_scope_name` | varchar(128) | business-scope name snapshot |
| `dept_id` | bigint | data-scope ownership, indexed |
| `owner` / `remark` | varchar/text | optional metadata |
| audit and soft-delete fields | datetime/varchar | timestamps and actors |

Key indexes include unique `uk_ip_deleted`, `idx_status`, and `idx_os`.

### 5.2 Group Table

`biz_cmdb_group` stores `parent_id`, `name`, JSON `conditions`, description, timestamps, and soft-delete state.

### 5.3 LabelSchema Table

`biz_cmdb_label_schema` stores unique `key`, display `name`, `value_mode`, optional `dict_code`, JSON `options`, `required`, `status`, description, timestamps, and soft-delete state.

### 5.4 Design Constraints

- tables use the `biz_` prefix
- queries use GORM for MySQL/PostgreSQL/SQLite portability
- host and group reads and writes obey `DataScopeReq`
- new hosts default `dept_id` from the current principal
- label-schema deletion checks Host labels and Group conditions

### 5.5 Tenant Readiness

The current runtime is single-tenant and does not add `tenant_id`. Future multitenancy must make host IP uniqueness tenant-scoped and inject a tenant filter into list queries.

## 6. API Design

Prefix: `/api/v1/business/cmdb`

### 6.1 Host API

| Method | Path | Purpose | Permission |
| :--- | :--- | :--- | :--- |
| `GET` | `/hosts` | host list | `business:cmdb:host:list` |
| `GET` | `/hosts/:id` | host detail | `business:cmdb:host:detail` |
| `POST` | `/hosts` | create host | `business:cmdb:host:create` |
| `PUT` | `/hosts/:id` | update host | `business:cmdb:host:update` |
| `DELETE` | `/hosts/:id` | soft-delete host | `business:cmdb:host:delete` |
| `POST` | `/hosts/:id/collect` | one-shot SSH collection | `business:cmdb:host:collect` |
| `PATCH` | `/hosts/:id/status` | downstream status update | `business:cmdb:host:status` |

SSH collection accepts password or private-key authentication. Credentials must not be persisted, logged, cached, or returned.

### 6.2 Group API

| Method | Path | Purpose | Permission |
| :--- | :--- | :--- | :--- |
| `GET` | `/groups` | group list | `business:cmdb:group:list` |
| `GET` | `/groups/:id` | group detail with members | `business:cmdb:group:detail` |
| `GET` | `/groups/:id/members` | calculated member list | `business:cmdb:group:detail` |
| `POST` | `/groups` | create group | `business:cmdb:group:create` |
| `PUT` | `/groups/:id` | update conditions | `business:cmdb:group:update` |
| `DELETE` | `/groups/:id` | delete group | `business:cmdb:group:delete` |

### 6.3 Label API

| Method | Path | Purpose | Permission |
| :--- | :--- | :--- | :--- |
| `GET` | `/labels` | label-schema list | `business:cmdb:label:list` |
| `GET` | `/labels/options` | label-schema select options | **Permission mapping gap:** the route exists, but `business:cmdb:label:list` currently maps only `GET /labels`, not this route. Track in `2026-07-17-business-permission-gaps`. |
| `POST` | `/labels` | create schema | `business:cmdb:label:create` |
| `PUT` | `/labels/:id` | update schema | `business:cmdb:label:update` |
| `DELETE` | `/labels/:id` | delete schema | `business:cmdb:label:delete` |

### 6.4 Error Keys

The repository-level canonical list is `BUSINESS_ERROR_SEMANTICS_APPENDIX.md`. CMDB uses `business.cmdb.host.*`, `business.cmdb.collect.*`, `business.cmdb.group.*`, and `business.cmdb.label.*`.

## 7. Permission Model

Page permissions are `business:cmdb:host:view`, `business:cmdb:group:view`, and `business:cmdb:label:view`.

### 7.1 Host Permission Points

Host permissions cover `view`, `list`, `detail`, `create`, `update`, `delete`, `collect`, and `status`.

### 7.2 Group Permission Points

Group permissions cover `view`, `list`, `detail`, `create`, `update`, and `delete`.

### 7.3 Label Permission Points

Label permissions cover `view`, `list`, `create`, `update`, and `delete`.

`view`, `list`, `detail`, and write actions remain independent. A list permission must never backstop create, update, or delete.

## 8. Menu and Route Design

| Menu key | Path | Component key | Page permission |
| :--- | :--- | :--- | :--- |
| `operations.cmdb.host` | `/business/cmdb/host` | `business/cmdb/host/CmdbHostList` | `business:cmdb:host:view` |
| `operations.cmdb.group` | `/business/cmdb/group` | `business/cmdb/group/CmdbGroupList` | `business:cmdb:group:view` |
| `operations.cmdb.label` | `/business/cmdb/label` | `business/cmdb/label/CmdbLabelSchemaList` | `business:cmdb:label:view` |

Detail/edit routes are reached from list pages. Menu titles use i18n keys, and component keys must be registered in frontend and backend allowlists.

## 9. Frontend Page Design

### 9.1 `CmdbHostList`

Hero, filters, table, pagination, capacity/label/status columns, and detail/edit/delete/collect actions.

### 9.2 `CmdbHostDetail`

Shows identity, capacity, labels, installed components, status, and request-scoped SSH collection.

### 9.3 `CmdbHostForm`

Supports identity, IP/port, OS, capacity, labels, owner, and remark with validation.

### 9.4 `CmdbGroupList`

Uses a real group tree on the left and the selected group/member view on the right. Parents with children cannot be deleted.

### 9.5 `CmdbGroupForm`

Maintains parent, description, and key-operator-value condition rows with AND/OR selection.

### 9.6 `CmdbLabelSchemaList`

Maintains key, name, value mode, dictionary code, options, required flag, status, and description. Referenced schemas return `business.cmdb.label.inUse` on delete.

### 9.7 Page States

All pages cover loading, initial empty, filtered empty, forbidden, server error, submitting, and delete confirmation.

### 9.8 Theme and Accessibility

Use inherited Base tokens, dark-mode behavior, keyboard navigation, ARIA, and focus rules. CMDB must not introduce a local visual system.

### 9.9 Responsive Behavior

Host tables collapse by column priority and become cards at narrow widths. The group tree/table layout becomes a single column below `md`.

## 10. Internationalization

Use `operations.cmdb.*`, `business.cmdb.host.*`, `business.cmdb.group.*`, `business.cmdb.label.*`, and `business.cmdb.collect.*`. Menus, breadcrumbs, filters, columns, fields, validation, actions, states, errors, empty states, and confirmations must be bilingual.

## 11. Dictionary and Configuration Dependencies

### 11.1 Dictionaries

Expected dictionaries include `cmdb_host_status`, `cmdb_os_type`, `cmdb_label_key`, and `cmdb_env`. CMDB still owns label business rules when a schema references a system dictionary code.

### 11.2 Configuration

| Key | Target state | Current truth |
| :--- | :--- | :--- |
| `cmdb.ssh.collect_timeout` | read from `system/config`, default `10` seconds | **not externalized:** `host_service.go` hard-codes `10s` |
| `cmdb.ssh.default_port` | read from `system/config`, default `22` | **not externalized:** the model/service falls back directly to `22` |

Externalization remains a target, not an implemented fact. Track it in `2026-07-17-cmdb-ssh-config-externalize`.

## 12. Audit and Security Requirements

### 12.1 Audit Points

Audit host create/update/delete/collect and group create/update/delete. Future import, export, and batch operations require dedicated actions.

### 12.2 Security Requirements

SSH credentials are request-scoped and never persisted or logged; private keys are preferred; collection records operator, time, target, and result; forms prevent duplicate submission; hosts and groups use soft delete.

## 13. Seed and Initialization

Seeds cover `operations`, `operations.cmdb`, `operations.cmdb.host`, `operations.cmdb.group`, `operations.cmdb.label`; Host/Group/Label permissions; CMDB i18n; dictionaries; and the target configuration keys.

Registered component keys include:

- `business/cmdb/host/CmdbHostList`
- `business/cmdb/host/CmdbHostDetail`
- `business/cmdb/group/CmdbGroupList`
- `business/cmdb/label/CmdbLabelSchemaList`

## 14. Risks and Out-of-Scope Items

### 14.1 Explicitly Out of Scope

Full ITIL topology, automatic discovery, liveness monitoring, credential-vault integration, and first-version CSV/Excel import/export are excluded.

### 14.2 Risks

SSH failures fall back to manual entry; label drift is mitigated by schema governance; expressions stay limited to AND/OR plus four operators; GORM protects database portability.

## 15. Tests and Acceptance

### 15.1 API Tests

Cover Host CRUD, Group CRUD/member calculation, SSH success/failure, status transitions, and duplicate-IP rejection.

### 15.2 Permission Tests

Cover independent `view`, `list`, `detail`, write, and collection permissions with API-side 403 behavior.

### 15.3 UI State Tests

Cover empty states, validation, confirmation, submit deduplication, and loading skeletons.

### 15.4 i18n Tests

Verify complete Chinese/English menus, pages, forms, states, and errors.

### 15.5 Browser Smoke

Run `cd frontend && npm run test:smoke:business:cmdb` for host list/detail and group routes.

### 15.6 Audit Tests

Verify Host/Group writes and SSH collection audit records.

### 15.7 Host Data Scope and Permission Closure

Verify `DataScopeReq + WithDataScope`, `dept_id` filtering, business menu placement, page-template consistency, audit actions, independent button permissions, and canonical Host/Collect keys.

### 15.8 Group Data Scope and Permission Closure

Verify visible-host-only member calculations, condition validation, tree behavior, audit actions, independent permissions, and canonical Group/Label keys.
