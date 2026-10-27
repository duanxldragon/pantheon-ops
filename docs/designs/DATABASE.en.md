---
title: Database Design Rules and Detailed Notes
doc_type: Design
layer: platform
status: Active
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
updated_at: 2026-04-17
---

# Database Design Rules and Detailed Notes

Chinese version: [DATABASE.md](./DATABASE.md)

This document defines the shared database rules for Pantheon Base and highlights several core tables that carry platform-level semantics.

## 1. Base Rules

- engine: `InnoDB`
- charset and collation: `utf8mb4_general_ci`
- primary key: `bigint unsigned NOT NULL AUTO_INCREMENT`
- soft delete: use GORM `deleted_at` with `datetime(3)` and `NULL` default

## 2. Naming Contract

- base tables use `system_` prefixes
- business tables use `biz_` prefixes
- mapping tables use `_rel_` or `_mapping`

## 3. Core Tables

### 3.1 Runtime I18n Storage: `system_i18n`

- belongs to `system/config`
- stores shared runtime translation assets for platform and business domains
- key fields include `module`, `group_name`, `key`, and `locale`
- runtime uniqueness and governance considerations must follow the i18n design rules
- MySQL is now the only runtime database target

### 3.2 Audit Logs

Operation log: `system_log_oper`

- stores `request_id` for cross-linking frontend failures, API responses, and audit traces
- stores masked JSON request bodies in `oper_param`
- stores latency in `cost_time`
- stores derived filtering fields such as `source_domain`, `source_page`, and `failure_category`
- uses dedicated indexes for request tracing and governance filtering

Login log: `system_log_login`

- must record `request_id`, `ipaddr`, `browser`, and `login_location`

### 3.3 User Session: `system_user_session`

- `session_id` is the session primary identifier and is written into access and refresh token context
- `refresh_jti` rotates on refresh
- `revoked_at` is written during logout or forced kickout
- `refresh_expires_at` limits the maximum session lifetime

### 3.3.1 Login-Source Throttle: `system_login_throttle`

- belongs to `system/auth`
- records source or IP throttling state in sliding windows
- key fields include `source_key`, `failure_count`, `window_started_at`, and `blocked_until`
- uses a unique index on `source_key` plus an auxiliary index on `blocked_until`

### 3.3.2 User Soft Delete and Unique-Key Reuse

- `system_user.username` remains physically unique
- before soft delete, services archive the old username into an internal reserved value so the original username can be recreated later

### 3.3.3 Other Soft-Delete Unique Keys in System Domains

The same archive-before-soft-delete strategy also applies to:

- `system_role.role_key`
- `system_post.post_code`
- `system_dict_type.dict_code`
- `system_dict_item(dict_code, item_value)`

### 3.3.4 Generator Managed Data Source: `system_generator_datasource`

- belongs to `system/config -> generator`
- stores readonly external-database connection metadata for schema inspection
- currently only supports `mysql`
- stores encrypted passwords
- restricts scope to metadata inspection only
- must not execute arbitrary SQL or write to external databases

### 3.3.5 Platform Shell Preferences: `system_user.preference_json`

- physically stored on `system_user`, but semantically belongs to platform shell preferences rather than to `system/iam` profile fields
- currently only supports `theme`, `language`, `layoutMode`, and `densityMode`
- migration rewrites legacy aliases into the current normalized field set
- runtime resolution order is current-session override, then user preference, then system default

### 3.4 Casbin Policy Table: `casbin_rule`

- bootstrapped by `database/system_init.sql`
- synchronized again by runtime migration through the GORM adapter
- stores persistent `p` and `g` policy rules
- default admin policies ensure `/api/v1/*` coverage for standard HTTP verbs

### 3.5 Organization Tables: `system_dept`, `system_post`

- `system_dept` stores trees through `parent_id + ancestors`
- root organization is identified by `is_root=1`
- `system_post` links to departments logically through `dept_id` without physical foreign keys
- `system_user.dept_id` and `system_user.post_id` participate in user-management organization semantics

## 4. Index and Performance Rules

- low-cardinality status fields usually should not have standalone indexes unless combined in real compound-query patterns
- physical foreign keys are prohibited to keep future modular split flexibility
- data integrity should be maintained in service-layer logic instead

## 5. Initial Initialization Path

- see `database/system_init.sql`
- the SQL script no longer writes the default admin account directly
- the first admin user is created by backend migration logic
- in non-production, missing `PANTHEON_INITIAL_ADMIN_PASSWORD` falls back to local development defaults
- in production, `PANTHEON_INITIAL_ADMIN_PASSWORD` must be explicitly set and must be at least 12 characters
- runtime `PANTHEON_DSN` must be a MySQL DSN, and tests also standardize on MySQL fixtures
