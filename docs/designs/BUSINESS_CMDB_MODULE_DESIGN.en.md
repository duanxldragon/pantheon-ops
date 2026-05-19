# CMDB Business Module Design

Chinese version: [BUSINESS_CMDB_MODULE_DESIGN.md](./BUSINESS_CMDB_MODULE_DESIGN.md)

Updated: 2026-05-19

Type: Design  
Layer: `business/cmdb`  
Status: Active

This document defines the CMDB module for `pantheon-ops`. It replaces the earlier low-code validation sample and serves as the canonical business design for host inventory, label governance, and group-based target selection.

## Scope

The module covers:

- `business/cmdb/host`: host inventory
- `business/cmdb/group`: host groups as persistent label-filter views
- `business/cmdb/label`: label schema governance

The module does not own authentication, IAM, org, system config, realtime monitoring, or credential vaulting.

## Core Responsibilities

- Maintain a unified host inventory across physical hosts, VMs, and Kubernetes nodes.
- Support orthogonal label dimensions such as `env` and `biz`.
- Provide reusable group views for downstream deployment and operations workflows.
- Support manual entry and one-shot SSH collection.

## Boundary Rules

- Downstream modules must consume CMDB through APIs, not direct database access.
- Host visibility and group member calculation must obey `DataScopeReq`.
- `business/cmdb` may reference system dictionaries through declared contracts, but must not import `system/config` internals.

## Data Model

Primary tables:

- `biz_cmdb_host`
- `biz_cmdb_group`
- `biz_cmdb_label_schema`

Important constraints:

- host and group records are soft-deleted
- labels and group conditions are JSON-backed
- host ownership uses `dept_id` for data-scope filtering
- current mode is single-tenant, but tenant-ready expansion is reserved

## APIs

Prefix: `/api/v1/business/cmdb`

Main resources:

- hosts: list, detail, create, update, delete, collect, status update
- groups: list, detail, member query, create, update, delete
- labels: list, create, update, delete

Permission naming follows:

- `business:cmdb:host:*`
- `business:cmdb:group:*`
- `business:cmdb:label:*`

## Error-Key Namespace

Canonical backend keys stay inside:

- `business.cmdb.host.*`
- `business.cmdb.group.*`
- `business.cmdb.label.*`
- `business.cmdb.collect.*`

Examples include `business.cmdb.host.notFound`, `business.cmdb.group.invalidConditions`, and `business.cmdb.collect.sshAuthFailed`.

## Frontend Shape

Main pages:

- `CmdbHostList`
- `CmdbHostDetail`
- `CmdbHostForm`
- `CmdbGroupList`
- `CmdbGroupForm`
- `CmdbLabelSchemaList`

UI rules:

- reuse Pantheon Base page templates and state patterns
- keep all text under `business.cmdb.*` i18n namespaces
- honor accessibility, theme token, dark-mode, and responsive guidance inherited from `pantheon-base`

## Security and Audit

- SSH credentials are request-scoped only and must never be persisted.
- Host, group, and collection actions must emit audit records.
- Destructive actions use soft delete and confirmation flows.

## Acceptance Focus

Minimum acceptance includes:

- host CRUD
- group CRUD and member resolution
- SSH collection success and failure paths
- permission separation across page, list, detail, and action scopes
- bilingual UI copy and canonical error-key handling
- data-scope enforcement for both host and group flows

For full implementation details, field definitions, state flows, and test expectations, use the Chinese source document as the authoritative design surface.
