---
title: system/config Extended Design
doc_type: Design
layer: system/config
status: Active
linked_contracts:
  - docs/contracts/SYSTEM_CONFIG_CONTRACT.md
updated_at: 2026-04-29
---

# system/config Extended Design

Chinese version: [SYSTEM_CONFIG_EXTENDED_DESIGN.md](./SYSTEM_CONFIG_EXTENDED_DESIGN.md)

This document recenters the already real extension surface of `system/config` into one higher-level design anchor.

It does not replace:

- `DICT_AND_SETTING_DESIGN.md`
- `ERROR_CODE_AND_I18N.md`
- `LOWCODE_GENERATOR_GUIDE.md`

It answers a higher-level question:

What subdomains now belong to `system/config`, what does each one own, how are they isolated, and which parts count as high-sensitivity governance capabilities?

## 1. Design Goals

`system/config` is no longer just dictionary and settings pages.

It should now be understood as a composite system domain covering six capability areas:

1. `dict`: runtime enums and option governance
2. `setting`: platform parameters and policy configuration
3. `i18n`: translation assets and language governance
4. `upload`: unified upload entry and storage configuration
5. `dynamicmodule`: dynamic module registration governance
6. `generator`: business-module scaffolding generator

Goals:

- group these six capabilities back under `system/config`
- clarify which are ordinary configuration capabilities and which are high-sensitivity governance capabilities
- prevent `system/config` from turning into an uncontrolled junk drawer
- provide a top-level anchor for acceptance checklists, permission matrices, and future specialized design documents

## 2. Overall Boundary

### 2.1 What `system/config` Owns

- platform-level configuration read, write, cache refresh, and audit
- unified governance of dictionaries and dropdown options
- translation assets, language packs, missing-locale repair, and lifecycle governance
- upload configuration and the unified upload entry
- module generation, module registration, and module-status governance

### 2.2 What `system/config` Does Not Own

- users, roles, menus, and authorization itself, which belong to `system/iam`
- login, sessions, passwords, and security events, which belong to `system/auth`
- organization structure and governance, which belong to `system/org`
- the audit platform itself, which belongs to `system/audit`
- runtime business workflows, which belong to `business/*`

### 2.3 Key Constraints

- `system/config` may host configuration-shaped shared capabilities, but may not absorb the responsibilities of other system domains
- `generator` and `dynamicmodule` live under `system/config` but are not ordinary settings-page functions
- high-sensitivity capabilities must stay clearly separated from public-readable configuration

## 3. Subdomain Split

### 3.1 `dict`

Owns:

- dictionary types
- dictionary items
- dictionary cache refresh
- frontend option delivery

Assessment:

- standard configuration subdomain
- low-risk governance capability focused on consistency and reuse

### 3.2 `setting`

Owns:

- grouped settings such as `basic`, `security`, `login`, `audit`, `upload`, `i18n`, and `ui`
- public platform configuration
- encrypted sensitive settings
- setting audit and cache refresh

Assessment:

- the central subdomain of `system/config`
- owns configuration values, but not every runtime meaning by itself

### 3.3 `i18n`

Owns:

- runtime language packs
- translation-record CRUD
- import, export, and templates
- missing-locale detection
- key-rename preview and migration
- builtin locale backfill
- lifecycle governance for unused keys

Assessment:

- already an independent subdomain rather than a subordinate settings-page feature
- combines content governance with runtime resource publishing

Boundary:

- `i18n` owns translation assets
- the frontend owns consumption and fallback
- business modules own namespace declaration, key addition, and acceptance coverage

### 3.4 `upload`

Owns:

- unified upload entry
- storage-driver switching
- size, type, and access-path limits
- local file access entry
- `s3-compatible` object-storage support

Assessment:

- shared foundation capability
- configuration belongs to `system/config`
- runtime file handling belongs to common platform capability, not to each business module

### 3.5 `dynamicmodule`

Owns:

- dynamic module registry queries
- module registration and uninstall
- post-generation module status
- generated registry updates and assembly alignment

Assessment:

- not an ordinary configuration feature
- a high-sensitivity platform-governance capability

Why:

- it affects workspace source and module assembly
- it changes the platform’s active module landscape
- mistakes can directly impact build, routing, and permission integration

Must keep:

- treat it primarily as internal development or governance capability
- protect write operations with stronger permissions and secondary confirmation
- document environment constraints, audit, rollback, and misuse protection explicitly

### 3.6 `generator`

Owns:

- business-module scaffolding from schema
- initial backend and frontend code, menus, permissions, and i18n skeletons
- output handoff into the dynamic-module governance registration chain

Assessment:

- a developer-acceleration subdomain under `system/config`
- not a runtime low-code orchestration platform

Must keep:

- key-first i18n
- generated content aligned with module contracts
- no bypass of dynamic-module governance or permission checks

## 4. Risk Classification

- `dict`: low
- `setting`: medium
- `i18n`: medium
- `upload`: medium-high
- `dynamicmodule`: high
- `generator`: medium-high

## 5. Frontend Page Ownership

Current page interpretation:

- `/system/dict` -> `dict`
- `/system/setting` -> `setting`
- `/system/i18n` -> `i18n`
- `/system/modules` -> `dynamicmodule`
- `/system/generator` -> `generator`

Constraint:

- these pages all belong to `system/config`
- but their permission strength and acceptance depth should not be treated as identical

## 6. Permission and Security Constraints

### 6.1 Ordinary Governance Capabilities

Examples include dictionary CRUD, standard setting reads and writes, translation-asset maintenance, and cache refresh flows with ordinary governance impact.

### 6.2 High-Sensitivity Governance Capabilities

Examples include dynamic module operations, generator-linked registration flows, sensitive configuration writes, and operations that can reshape the working module landscape.

These require stronger permission separation, clearer auditability, and tighter operation control.

## 7. Acceptance Requirements

Acceptance must verify:

- subdomain boundaries are clear
- public-readable settings are separated from sensitive capabilities
- shared runtime consumers still point to the correct owning subdomain
- high-sensitivity operations are treated differently from ordinary config CRUD
- specialized design documents still attach back to this higher-level anchor

## 8. Relationship with Other Documents

This document is the umbrella anchor. Specialized documents still define each subdomain in detail.

## 9. Current Conclusion

`system/config` is now a composite governance domain. It should keep absorbing configuration-shaped shared capabilities only when the boundary is explicit, the risk is classified, and the ownership does not override another domain’s responsibility.
