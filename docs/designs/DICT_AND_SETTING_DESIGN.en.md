---
title: Dictionary and System Setting Design
doc_type: Design
layer: system/config
status: Active
linked_contracts:
  - docs/contracts/SYSTEM_CONFIG_CONTRACT.md
updated_at: 2026-04-28
---

# Dictionary and System Setting Design

Chinese version: [DICT_AND_SETTING_DESIGN.md](./DICT_AND_SETTING_DESIGN.md)

This document defines the Pantheon Base design for dictionary management and system settings. Both belong to `system/config` and form part of the shared platform foundation.

Current baseline closure already covers:

- `system/setting`: model, migration, defaults, public reads, grouped admin reads and writes, setting page, cache refresh, runtime health summary
- `system/dict`: model, migration, seed data, type and item CRUD, public options API, frontend workbench, server-side item pagination, aggregate counts, cache refresh
- upload configuration groups, encrypted sensitive settings, change-audit details, and cache strategy; the shared upload allowlist now includes `webp` and `gif` to stay aligned with shared frontend upload consumers
- first-batch runtime wiring for site identity, password rules, login lock policy, session idle timeout, audit retention, default language, default theme, tab-bar visibility, and upload drivers

## 1. Design Goals

- centralize business enums, status values, and dropdown options
- centralize platform, security, upload, login, i18n, and UI configuration
- support cache and refresh
- support modular registration
- support reuse by future business modules

## 2. Capability Boundaries

### 2.1 Dictionary Management Owns

- dictionary types
- dictionary items
- ordering
- item status
- cache refresh
- frontend option delivery

### 2.2 System Setting Owns

- platform basics
- security policy
- upload configuration
- login policy
- i18n defaults
- UI defaults
- allowed default-language values

`generator` also belongs to `system/config`, but manages controlled engineering metadata such as managed data sources rather than runtime platform parameters.

### 2.3 Out of Scope

Dictionary and setting modules do not own:

- user authorization logic
- business workflow state machines
- large business datasets
- plaintext secret storage

## 3. Module Ownership

Recommended locations:

- `backend/modules/system/dict/`
- `backend/modules/system/setting/`
- `frontend/src/modules/system/dict/`
- `frontend/src/modules/system/setting/`

Domain ownership stays in `system/config`.

## 4. Dictionary Management Design

### 4.1 Dictionary Model

Use two tables:

- `system_dict_type`
- `system_dict_item`

### 4.2 `system_dict_type`

The type table owns unique `dict_code`, human-readable `dict_name`, module ownership, status, remarks, and timestamps.

### 4.3 `system_dict_item`

Each item stores `dict_code`, `item_label_key`, `item_value`, optional color, sorting, status, remarks, and timestamps.

### 4.4 Dictionary I18n Rules

Do not store natural-language labels as the primary source. Store `item_label_key` and translate at runtime.

### 4.5 Dictionary Usage Rules

Frontend:

- fetch by `dict_code`
- render through `item_label_key`
- submit `item_value`
- optionally use `item_color` for status tags

Backend:

- validate values against dictionary definitions
- never depend on display text

### 4.6 Current Dictionary Workbench Constraints

- the frontend uses a type-and-item workbench rather than a hardcoded master-detail dependency
- type lists return aggregate counts and last-update hints
- item lists use server-side pagination with `dictCode`, `status`, `keyword`, `page`, and `pageSize`
- exports follow the current filter context, not the current page slice

## 5. System Setting Design

### 5.0 Managed Generator Data Sources

The current recommended table is `system_generator_datasource`, with encrypted passwords, readonly metadata-only scope, connection-check audit fields, and first-phase MySQL-only support.

### 5.1 Setting Model

Use `system_setting` with fields such as:

- `setting_key`
- `setting_value`
- `value_type`
- `group_key`
- `module`
- `is_public`
- `is_encrypted`

### 5.2 `value_type`

Support explicit value typing rather than treating all settings as opaque strings.

### 5.3 `group_key`

Group settings by operational domain such as `basic`, `security`, `login`, `audit`, `upload`, `i18n`, and `ui`.

### 5.4 `setting_key` Examples

Representative runtime keys include:

- site identity
- password minimum length
- login failure and lock parameters
- session idle timeout
- active-session caps
- audit retention windows
- default language and theme
- upload driver and size limits

#### 5.4.1 Runtime Semantics for Upload Settings

Upload settings already drive the unified upload service, URL generation, and local versus `s3-compatible` runtime behavior.

#### 5.4.2 Runtime Semantics for Log Governance

Audit retention settings already control login-log, operation-log, and session-retention governance behavior.

### 5.5 Public Settings

Public settings may be read by the frontend shell, but encrypted sensitive settings must never leak through the public surface.

## 6. API Design

### 6.1 Dictionary APIs

Support type CRUD, item CRUD, options retrieval, template download, import, export, and cache refresh.

### 6.2 Setting APIs

Support grouped reads and writes, public reads, audit reads and exports, cache refresh, and health-summary retrieval.

## 7. Frontend Page Design

### 7.1 Dictionary Page

Use a workbench that supports type filtering, item ordering, status maintenance, import, export, and cache refresh.

### 7.2 System Setting Page

Use grouped configuration editing, runtime health summary, sensitive-field masking, audit details, and cache refresh.

### 7.3 Dictionary Cache Refresh

Provide both automatic invalidation after writes and a manual refresh entry for debugging and operations.

## 8. Permission Design

Use explicit `system/config` permissions and keep high-risk operations split cleanly.

### 8.1 Dual-Track Permission Constraints for System Setting

The settings page requires both frontend page permission and backend API policy coverage. One without the other creates a broken experience.

### 8.2 Audit-Block Degradation Rules for Lower-Privilege Users

Audit subareas may degrade or hide based on permissions, but the main settings experience must stay understandable and safe.

## 9. Menu Design

Configuration pages should live under `system/config` navigation and preserve clear separation from `system/iam`, `system/org`, and `system/auth`.

## 10. I18n Key Planning

Dictionary labels, setting group titles, field labels, health summaries, import/export feedback, and runtime prompts must all use structured i18n keys.

## 11. Cache Design

### 11.1 Dictionary Cache

Cache dictionary options by `dict_code`, invalidate on type or item changes, and allow manual refresh.

### 11.2 Setting Cache

Cache grouped and public settings with controlled invalidation after updates.

### 11.3 Refresh Strategy

Prefer automatic invalidation after writes, with manual refresh and prewarm entries for operations workflows.

## 12. Relationship with I18n

I18n is adjacent but distinct:

- dictionaries provide structured option metadata
- settings provide runtime default language and i18n governance
- translation assets and language packs still follow the dedicated i18n workflow

## 13. Relationship with Business Modules

Business modules should consume shared dictionaries and public settings through stable contracts instead of hardcoding their own parallel enum and config systems.

## 14. Security Rules

- sensitive settings must be encrypted at rest
- public APIs must never leak encrypted secrets
- generator data sources are read-only and metadata-only
- runtime defaults and user preferences must stay clearly separated

## 15. Phased Implementation

### 15.1 Phase 1: Dictionary Management

Complete the dictionary CRUD, options delivery, cache, and frontend workbench baseline.

### 15.2 Phase 2: System Settings

This baseline is already closed for grouped settings, public reads, audits, runtime wiring, and cache behavior.

### 15.3 Phase 3: Module Seeds

Continue pushing stable seed registration so modules can contribute dictionary and setting defaults in a controlled way.

## 16. Current Delivery Gaps

The remaining gaps are mostly around deeper runtime coverage, stronger governance tooling, and more complete reuse by future business modules.

## 17. Acceptance Checklist

Acceptance should verify:

- type and item CRUD
- options delivery and translation behavior
- grouped settings reads and writes
- encryption and public-read boundaries
- cache invalidation and manual refresh
- runtime consumption of key platform settings

## 18. Recommended Next Document

Continue with the related `system/config` design, remediation, and acceptance documents referenced by the contract set.
