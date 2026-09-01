---
title: I18n Module Design
doc_type: Design
layer: system/config
status: Active
linked_contracts:
  - docs/contracts/SYSTEM_CONFIG_CONTRACT.md
updated_at: 2026-04-29
---

# I18n Module Design

Chinese version: [I18N_MODULE_DESIGN.md](./I18N_MODULE_DESIGN.md)

This document defines the independent boundary of `system/config -> i18n`.

It focuses on four questions:

- what layer `i18n` belongs to in Pantheon
- who owns runtime translation assets, frontend fallback behavior, and backend message keys
- why `i18n` is more than a CRUD page for translations
- what acceptance should verify when adding locales, renaming keys, or cleaning unused keys

## 1. Module Positioning

`i18n` belongs to the `system/config` domain.

Its job is not just to “show Chinese or English on pages”. It exists to:

- maintain runtime translation assets
- provide stable language packs for frontend rendering and import-export flows
- govern key lifecycle
- provide namespace and translation convergence points for new modules

It is not:

- a casual copy editor for page text
- an optional side page that modules may bypass
- a frontend-only tool

## 2. Boundary

### 2.1 What `i18n` Owns

- language-pack reads
- translation-item CRUD
- import, export, and template download
- missing-locale detection
- missing-key sync
- builtin-locale backfill
- key-rename preview and execution
- observation, archive, and deletion for unused keys

### 2.2 What `i18n` Does Not Own

- backend business-code design
- final page layout and presentation
- menu, permission, or module-boundary ownership
- business-domain meaning itself

### 2.3 Collaboration Boundary

- the backend returns stable `message` keys
- the frontend handles `t(message)` and fallback
- the `i18n` subdomain makes keys and translation assets governable, searchable, and migratable

## 3. Runtime Model

### 3.1 Asset Sources

Runtime translation priority:

1. remote translation assets in `system_i18n`
2. frontend local `fallbackResources`
3. common fallback keys
4. only then the raw unresolved key

### 3.2 Current Locale Baseline

Built-in fallback locales:

- `zh-CN`
- `en-US`
- `ja-JP`
- `ko-KR`
- `fr-FR`

Rules:

- do not pre-expand locales for imagined future markets
- new locales must be driven by real delivery need

### 3.3 Uniqueness Constraints

Runtime asset uniqueness must be enforced by:

- `locale + key`

not by:

- `module + locale + key`

`module` is for governance ownership and filtering, not runtime lookup uniqueness.

## 4. Pages and APIs

### 4.1 Frontend Page

Main page:

- `/system/i18n`

Current page permission:

- `system:i18n:list`

Current action permissions:

- `system:i18n:create`
- `system:i18n:update`
- `system:i18n:delete`
- `system:i18n:export`
- `system:i18n:import`
- `system:i18n:refresh`

### 4.2 Backend APIs

Public runtime read:

- `GET /api/v1/system/i18n/pack`

Governance APIs include:

- list, read, create, update, delete
- batch delete
- import, export, cache refresh
- key sync
- overview and audit
- missing-locale detection
- rename preview and execution
- builtin-locale hydration
- unused-key lifecycle operations

## 5. Lifecycle Governance

The hard part of `i18n` is not adding one translation row. The hard part is preventing the asset set from becoming ungovernable.

Minimum lifecycle:

1. detect missing keys
2. fill missing coverage
3. observe unused keys
4. archive or delete safely

### 5.1 Missing-Coverage Governance

Includes:

- missing-key sync
- missing-locale detection
- placeholder fill

Goal:

- stop modules from shipping with blank locale coverage

### 5.2 Rename Governance

Includes:

- rename preview
- rename execution
- migration reports

Goal:

- make key refactors explicit rather than manual luck-based edits

### 5.3 Unused-Key Governance

Includes:

- observe
- archive
- delete

Goal:

- create a safe buffer before destructive cleanup

## 6. Admission Rules for New Locales

A new locale must have:

- a real market, customer, or delivery need
- clear translation ownership
- a real long-term maintainer

At minimum, onboarding a locale requires:

1. local fallback resources
2. runtime asset import into `system_i18n`
3. `check:i18n-hardcode`
4. `audit:i18n-locales`
5. `npm run build`
6. regression checks across pages and import-export flows

## 7. Relationship with Error-Code Design

`i18n` cannot be separated from error-semantic design.

The coordination model is:

- backend owns stable `message` keys
- frontend owns translation and display
- `i18n` owns governable translation assets for those keys

Therefore:

- the backend should not return natural-language final copy
- the frontend should not expose raw keys
- the i18n admin surface should not redefine business rules

## 8. Risks

### 8.1 Common Risks

- same meaning under different keys
- duplicate runtime keys across modules
- long-lived placeholder values
- key renames that drift from source code and assets
- locale expansion without real operational ownership

### 8.2 Quality Gates

Use missing-key audits, hardcode checks, locale audits, build verification, and page-level regression checks as the minimum quality line.

## 9. Acceptance Requirements

Acceptance should verify:

- runtime pack reads work
- frontend fallback still works
- key uniqueness is governed correctly
- missing locales and unused keys are governable
- rename workflows are explicit and auditable
- new locales follow the admission and validation path

## 10. Current Conclusion

`i18n` is now a real `system/config` subdomain with runtime publication and lifecycle governance responsibilities. It should no longer be treated as a simple translation CRUD add-on.
