---
title: system/config Contract
doc_type: Contract
layer: system/config
status: Active
updated_at: 2026-05-12
---

# system/config Contract

Chinese version: [SYSTEM_CONFIG_CONTRACT.md](./SYSTEM_CONFIG_CONTRACT.md)

This document defines the execution contract for Pantheon’s `system/config` capability domain.

It locks down the overall boundary of configuration-oriented shared capabilities so that `dict / setting / i18n / upload / dynamicmodule / generator` do not collapse back into one oversized “configuration misc room” with no risk grading and no responsibility separation.

## 1. Background

Pantheon’s current `system/config` has clearly outgrown “dictionary + settings page”.

It currently carries at least six capability groups:

1. `dict`
2. `setting`
3. `i18n`
4. `upload`
5. `dynamicmodule`
6. `generator`

Without a `system/config` contract, the likely regressions are:

- ordinary configuration capability and high-sensitivity governance capability being discussed under the same operating model
- `i18n / upload / dynamicmodule / generator` being mistaken for accessory features of the settings page
- design docs multiplying without a unified anchor
- `system/config` becoming the most obvious kitchen-sink system domain again

## 2. Ownership Layer

This contract belongs to `system/config`.

It covers:

- dictionary and runtime-option governance
- platform parameter and policy configuration
- translation assets and language governance
- unified upload entry and storage configuration
- dynamic-module governance
- module generators and assisted-development chains

It does not mean:

- `system/auth` login, sessions, password, or security-policy consumption semantics
- `system/iam` users, roles, menus, or permission governance
- `system/org` organization governance
- `platform` shell navigation and workbench aggregation

## 3. Goals

The `system/config` contract exists to lock down:

1. the rule that `config` is a shared configuration domain rather than a container for leftover work from other system domains
2. risk grading between ordinary configuration capabilities and high-sensitivity governance capabilities
3. the overall boundary across `dict / setting / i18n / upload / dynamicmodule / generator`
4. the rule that `config` defines configuration and governance without swallowing runtime responsibilities from other domains
5. the definition of done and acceptance semantics for `system/config`
6. stable extension points for future topic-specific subcontracts

## 4. Non-Goals

This contract does not cover:

- login, sessions, password, login-failure lockout, and other auth primary-path behavior
- role authorization, menu metadata semantics, or permission workbench
- departments, posts, and organization governance
- business-domain-specific configuration semantics and lifecycle
- treating `generator` or `dynamicmodule` as a mature runtime low-code platform

In other words:

- `system/config` may provide shared configuration and controlled assisted-development capabilities
- but it must not take over the core responsibilities of `auth / iam / org / platform`

## 5. Boundaries

### 5.1 Covered Surfaces

- `/system/dict`
- `/system/setting`
- `/system/i18n`
- `/system/modules`
- `/system/generator`
- unified upload APIs and upload configuration
- configuration-cache refresh
- configuration-change audit

### 5.2 Subdomain Boundaries

#### `dict`

- owns dictionary types, dictionary items, status, ordering, and option delivery

#### `setting`

- owns platform parameters, grouped configuration, public configuration, sensitive configuration, cache refresh, and audit
- the settings frontend must keep the “overview entry + per-group edit page” structure and must not degrade back into one oversized page carrying all config and audit concerns
- settings audit is a cross-capability concern between `setting` and `audit`; it may only converge into the audit group page and must not spread back into every settings group as a permanent block

#### `i18n`

- owns translation assets, language packs, import/export, missing-key detection, and key lifecycle governance

#### `upload`

- owns upload configuration, unified upload entry, storage-driver switching, and access-URL generation

#### `dynamicmodule`

- owns module-onboarding governance, registration status, and generated-registry alignment

#### `generator`

- owns business-module skeleton generation, schema validation, generation flow, and controlled registration entry

### 5.3 Excluded Surfaces

- `/login`
- `/auth/security`
- `/system/user`
- `/system/role`
- `/system/menu`
- `/system/permission`
- `/system/dept`
- `/system/post`
- platform shell navigation and workbench aggregation

## 6. Dependencies

This contract depends on:

- [DESIGN.md](../../DESIGN.md)
- [AGENTS.md](../../AGENTS.md)
- [BACKEND.md](../designs/BACKEND.md)
- [FRONTEND.md](../designs/FRONTEND.md)
- [DICT_AND_SETTING_DESIGN.md](../designs/DICT_AND_SETTING_DESIGN.md)
- [SYSTEM_CONFIG_EXTENDED_DESIGN.md](../designs/SYSTEM_CONFIG_EXTENDED_DESIGN.md)
- [I18N_MODULE_DESIGN.md](../designs/I18N_MODULE_DESIGN.md)
- [UPLOAD_AND_STORAGE_DESIGN.md](../designs/UPLOAD_AND_STORAGE_DESIGN.md)
- [DYNAMIC_MODULE_GOVERNANCE_DESIGN.md](../designs/DYNAMIC_MODULE_GOVERNANCE_DESIGN.md)
- [GENERATOR_MODULE_DESIGN.md](../designs/GENERATOR_MODULE_DESIGN.md)
- [ACCEPTANCE_CHECKLIST.md](../acceptances/ACCEPTANCE_CHECKLIST.md)

## 7. Hard Constraints

### 7.1 Domain-Boundary Constraints

- `system/config` owns only shared configuration capabilities and controlled assisted-development capabilities
- it must not redefine the core responsibilities of `auth / iam / org / platform`
- subdomains may share configuration sources, but must not swallow each other’s semantics

### 7.2 Risk-Grading Constraints

- `dict / setting / i18n` are ordinary configuration or content-governance capabilities
- `upload` is shared infrastructure capability: configured in `config`, consumed at runtime by shared platform packages
- `dynamicmodule / generator` are high-sensitivity governance capabilities and must not be treated like ordinary settings pages

### 7.3 High-Sensitivity Capability Constraints

- `dynamicmodule` should default to internal development/governance capability semantics
- write operations must be guarded by stronger permissions, environment gates, and second confirmation
- the high-sensitivity action in `generator` is “generate and register”, not “view the page”
- real write paths must not bypass backend Casbin, environment guards, or second confirmation
- `/system/modules`, `/system/generator`, `/system/i18n`, `/system/setting`, and `/system/setting/:groupKey` must follow the unified page-admission rules
- page headers should express only the primary page task and must not duplicate right-side positioning menus
- the first screen may contain only one governance summary container and must not stack “explanation card walls”
- high-sensitivity actions must be concentrated in header action area, table-header action area, or governance drawer, not scattered across multiple first-screen zones
- one page must not try to be overview center, deep editor, and long-form explainer all at once

### 7.4 Runtime Consumption Constraints

- `setting` may output configuration values, but must not take over every consumption semantic
- when `i18n.default_language`, `login.*`, `audit.*`, and similar values are consumed by other system domains or the platform, boundaries must remain explicit
- `i18n.default_language` means only the platform default language; it applies at runtime only when there is no explicit session override and no user language preference
- `system/config` provides the default-value source, but does not own the final priority rule across login-page language choice, user preference, and current-session language
- the unified upload entry must be reused by business modules; bespoke upload protocols must not spread

### 7.5 Documentation Constraints

- all `system/config` design, assessment, remediation, and acceptance docs must link back to this contract
- future topic-specific subcontracts may be split under this contract, but must not bypass it

## 8. Definition of Done

`system/config` should count as currently complete only when:

### 8.0 Batch-Delete Capability Constraints

- dictionary types and dictionary items support controlled batch delete under `system/config/dict`
- batch delete must use dedicated permission point `system:dict:batch-delete`; batch enable/disable uses `system:dict:batch-update`, and the two must not be conflated
- batch-delete endpoints must reuse single-delete validations, preserving protections such as type-in-use checks, unique-key soft-delete release, and dictionary-cache invalidation
- batch delete is a high-risk write operation and must require second confirmation plus partial-success results: `deletedCount`, `failedCount`, `failures[]`

### 8.1 Responsibility Completion

- boundaries across the six subdomains are clear
- risk grading between ordinary configuration capability and high-sensitivity governance capability is clear

### 8.2 Runtime Completion

- primary paths for dictionary, settings, i18n, upload, dynamic module, and generator are stable
- configuration values and runtime consumption semantics are no longer muddled

### 8.3 Risk-Control Completion

- high-sensitivity actions in `dynamicmodule / generator` have controlled boundaries
- upload has unified configuration and a unified entry path
- `i18n` is treated as an independent subdomain instead of a settings-page accessory
- high-sensitivity pages have clear first-screen responsibilities: `modules` governs onboarded modules, `generator` handles new-module onboarding, `i18n` governs runtime language assets, and `setting` preserves overview/per-group split

### 8.4 Documentation and Acceptance Completion

- the main design, remediation, and acceptance docs for `config` all link back to this contract
- further topic subcontracts may continue to split under this contract

## 9. Acceptance Standards

Changes in `system/config` must at least pass:

### 9.1 Documentation Acceptance

- [ACCEPTANCE_CHECKLIST.md](../acceptances/ACCEPTANCE_CHECKLIST.md)
- [DOCUMENT_GOVERNANCE_CONTRACT.md](../contracts/DOCUMENT_GOVERNANCE_CONTRACT.md)
- [DOCUMENT_METADATA_AND_STATUS.md](../contracts/DOCUMENT_METADATA_AND_STATUS.md)

### 9.2 Backend and Configuration Acceptance

- add matching `go test` coverage for affected modules
- if upload, i18n, dynamicmodule, or generator is touched, verify the corresponding path explicitly

### 9.3 Frontend and Build Acceptance

- `cd frontend && npm run build`
- `cd frontend && npm run check:i18n-hardcode`
- if primary paths for configuration or high-sensitivity governance pages change, add page-level smoke or acceptance evidence

### 9.4 Page and Primary-Path Acceptance

- `/system/dict`
- `/system/setting`
- `/system/i18n`
- `/system/modules`
- `/system/generator`

### 9.5 High-Sensitivity Governance Acceptance

- `dynamicmodule` and `generator` must not be accepted under ordinary settings-page semantics
- permission, environment limits, second confirmation, and audit chains must be checked explicitly
- page admission must also be checked explicitly: no redundant right-side positioning menu, no hero wall, one governance summary container only

## 10. Related Documents

### 10.1 Design

- [DICT_AND_SETTING_DESIGN.md](../designs/DICT_AND_SETTING_DESIGN.md)
- [SYSTEM_CONFIG_EXTENDED_DESIGN.md](../designs/SYSTEM_CONFIG_EXTENDED_DESIGN.md)
- [I18N_MODULE_DESIGN.md](../designs/I18N_MODULE_DESIGN.md)
- [UPLOAD_AND_STORAGE_DESIGN.md](../designs/UPLOAD_AND_STORAGE_DESIGN.md)
- [DYNAMIC_MODULE_GOVERNANCE_DESIGN.md](../designs/DYNAMIC_MODULE_GOVERNANCE_DESIGN.md)
- [GENERATOR_MODULE_DESIGN.md](../designs/GENERATOR_MODULE_DESIGN.md)

### 10.2 Acceptance

- [ACCEPTANCE_CHECKLIST.md](../acceptances/ACCEPTANCE_CHECKLIST.md)
- [PLATFORM_ACCEPTANCE_MATRIX_20260430_UI_MIGRATION.md](../acceptances/PLATFORM_ACCEPTANCE_MATRIX_20260430_UI_MIGRATION.md)

## 11. Reserved Topic Subcontracts

This contract is the top-level contract for `system/config`.

If the domain is deepened further, topic-level subcontracts may be added beneath it, for example:

- `system/config -> i18n contract`
- `system/config -> upload contract`
- `system/config -> dynamicmodule contract`
- `system/config -> generator contract`

Rules:

- subcontracts must not rewrite the boundary of this top contract
- subcontracts only add more detailed definition-of-done and acceptance rules
- until a topic subcontract exists, this top contract remains the governing source
