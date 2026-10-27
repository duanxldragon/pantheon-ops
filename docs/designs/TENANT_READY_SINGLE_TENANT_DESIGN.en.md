---
title: Single-Tenant First, Tenant-Ready Design
doc_type: Design
layer: platform / system/auth / system/config / business/*
status: Active
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
  - docs/contracts/SYSTEM_AUTH_CONTRACT.md
  - docs/contracts/SYSTEM_CONFIG_CONTRACT.md
updated_at: 2026-05-04
---

# Single-Tenant First, Tenant-Ready Design

Chinese version: [TENANT_READY_SINGLE_TENANT_DESIGN.md](./TENANT_READY_SINGLE_TENANT_DESIGN.md)

This document answers a practical question: Pantheon currently serves internal and backoffice scenarios and does not implement real multi-tenancy yet, but the base must stop making future tenant evolution unnecessarily expensive.

The goal is not to implement multi-tenancy now. The goal is to define the minimum rules for being tenant-ready while continuing to run as a single-tenant system.

## 1. Conclusion

Current strategy:

- runtime mode stays single-tenant
- no real tenant registration, billing, isolation middleware, or tenant-management console is introduced now
- all new platform, system, and business design work must explicitly judge whether the capability may later need tenant semantics
- new modules do not automatically require `tenant_id`, but they must explicitly decide whether tenant fields, tenant-scoped uniqueness, or tenant-context injection will be needed later

In one sentence:

Do not build multi-tenant features now, but stop writing base and business code that clearly blocks future multi-tenant migration.

## 2. Why Real Multi-Tenancy Is Not Implemented Now

Three main reasons:

1. the real tenant model, tenant boundary, pricing mode, and tenant-admin responsibilities are still unclear
2. multi-tenancy is a cross-cutting system constraint touching `auth`, `iam`, `org`, `config`, `audit`, and `business/*`
3. if the product never truly needs multi-tenancy, premature tenant layers become maintenance cost

The current correct objective is to control future migration cost, not to ship a half-built tenant system.

## 3. Why It Still Cannot Be Ignored

Tenant readiness affects:

- table structure
- uniqueness rules
- query constraints
- permission scope
- configuration scope
- audit semantics
- business-module scaffolding

Ignoring it now raises future migration cost sharply, especially once many `biz_*` tables and global-uniqueness assumptions already exist.

## 4. Boundaries to Lock in the Current Phase

### 4.1 Platform Layer

The platform shell stays single-tenant and should not add tenant switchers or tenant workbenches now.

But it should preserve future extension points for:

- request-context tenant injection
- separation between platform defaults and possible future tenant overrides
- tenant-filterable aggregate views

### 4.2 System Domains

#### `system/auth`

- keep local `system_user` login
- do not implement tenant login discovery or tenant-domain routing yet
- future external-identity models may reserve `tenant_code`-style extension points without activating them now

#### `system/iam`

- users, roles, permissions, and menus remain platform-scope for now
- new models and constraints must still ask whether they may later become tenant-scoped
- do not blindly hardcode every uniqueness rule as platform-global

#### `system/org`

- organization remains an organization model
- departments and posts must not be treated as a future tenant model

#### `system/config`

- configuration remains primarily platform-global
- if tenant-level overrides are needed later, evolve toward `platform default + tenant override`
- do not create `tenant_setting` now, but do judge the future scope of new settings

### 4.3 Business Domains

This is the most important area.

Every new business module must answer:

1. is this data likely to need tenant isolation later
2. if uniqueness is required, is it global or tenant-local
3. will list, export, and statistic APIs need unified tenant filtering later
4. should audit records later preserve tenant dimensions

If the likely answer is yes, the design must explicitly reserve room instead of hardcoding global assumptions.

## 5. Minimum Tenant-Ready Rules

### 5.1 Table-Structure Rules

For new `biz_*` tables:

- if the capability clearly will not be tenantized, document why `tenant_id` is unnecessary
- if it is likely to become tenantized, consider adding `tenant_id` from version one even if runtime remains single-tenant
- if it is uncertain, at least mark the design as tenant-ready pending and review uniqueness and query patterns carefully

### 5.2 Uniqueness Rules

Every new unique constraint must first answer:

- is this globally unique across the platform
- or only unique within a tenant

Many business identifiers such as project codes or order numbers are more likely to be tenant-local than platform-global.

### 5.3 Query Rules

Multi-row business interfaces should continue to use unified query-injection points rather than scattering custom filtering logic everywhere.

Current extension points such as `DataScopeReq` and `WithDataScope` should remain the standard hook for future tenant filtering.

### 5.4 Configuration Rules

New settings must be classified as:

- platform-global settings
- settings that may later support tenant overrides
- user-personal preferences

Do not mix these into one undefined category.

### 5.5 Audit Rules

Audit does not need a real `tenant_id` immediately, but designs must retain the assumption that future audit retrieval may need tenant slicing.

## 6. Actions Recommended Right Now

Do only low-cost readiness work:

1. document the single-tenant-first, tenant-ready principle
2. add tenant-readiness checks to business-module templates
3. keep using `DataScopeReq + WithDataScope` as the future tenant-filter extension point
4. require DDL reviews to answer whether `tenant_id` is needed
5. require uniqueness reviews to answer global versus tenant-local scope

## 7. Things Explicitly Not Done Now

Not in scope now:

- `system_tenant`, `tenant_user`, `tenant_setting`, or full tenant-domain models
- tenant registration, provisioning, billing
- tenant-admin consoles
- tenant-specific menu, role, or permission cloning systems
- tenant detection by subdomain or dedicated domain
- forced tenant discovery during login

If real multi-tenancy starts later, it should get its own dedicated design documents rather than being patched into the current base piecemeal.

## 8. Advice for the Current Project

Recommended direction:

- `system/auth`: keep local login plus MFA; do not implement SSO yet
- `system/config`: keep platform-level configuration; do not model tenant configuration yet
- `business/*`: start reviewing schemas and uniqueness from a tenant-ready perspective now
- `platform`: preserve future context-extension points but do not add tenant-shell features

This keeps the base stable now without closing off future evolution.

## 9. Current Readiness Review Notes

As of this review:

- `DataScopeReq + WithDataScope` already exist as future tenant-filter extension points
- `system/iam` already provides a role data-scope strategy page
- `business/cmdb/host` already consumes data-scope context and can act as a sample
- `BUSINESS_MODULE_ACCEPTANCE_MATRIX.md` already includes tenant-readiness checks

Still to be judged case by case for future business work:

- whether new `biz_*` tables need `tenant_id` immediately
- whether business identifiers should be globally unique or tenant-local
- whether export, statistics, and audit flows should extend unified scope handling with tenant filters
- whether real multi-tenancy later will require `system_tenant`, tenant identification, user-tenant relationships, and config-scope design
