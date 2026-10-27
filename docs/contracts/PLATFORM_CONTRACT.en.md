---
title: Platform Contract
doc_type: Contract
layer: platform
status: Active
updated_at: 2026-04-30
---

# Platform Contract

Chinese version: [PLATFORM_CONTRACT.md](./PLATFORM_CONTRACT.md)

This document defines the execution contract for the Pantheon `platform` layer.

It is not a page-level mockup or a one-off remediation note. It is the upper-boundary contract that all later design, implementation, assessment, remediation, and acceptance work for the platform layer must attach back to.

## 1. Background

Pantheon is no longer just “login + menus + CRUD shell”.

But the platform layer still faces two recurring risks:

- cross-domain capabilities such as `dashboard`, navigation shell, overlays, and global state pages get pushed back into a `system/*` subdomain
- platform-layer design, assessment, remediation, and acceptance work has grown, but there has been no single upper contract defining what belongs to platform, what counts as done, and what should stop spreading

Without this contract, the most likely failures are not missing features, but:

- aggregate-layer boundary drift
- shell-style divergence
- new pages copying old skeletons again
- historical remediation docs turning back into the de facto source of truth

## 2. Ownership Layer

This contract belongs to the `platform` layer.

Here, `platform` is a logical layer. It does not require a physical directory literally named `platform`.

It covers:

- login-page language selection and current-session language resolution
- post-login application shell
- platform workbench / dashboard
- global navigation modes
- platform-level page skeleton and overlay baselines
- cross-domain aggregation entry points
- platform-level state, feedback, and acceptance discipline

It does not mean:

- `system/auth`
- `system/iam`
- `system/org`
- `system/config`

These system domains may be aggregated by the platform layer, but they must not replace platform responsibilities in reverse.

## 3. Goals

The platform contract exists to lock down five things:

1. clarify the responsibility boundary between `platform` and each system domain
2. clarify that shell, navigation, dashboard, and overlays are governed uniformly at the platform layer
3. clarify that the platform layer is responsible for aggregation, navigation, and shared skeletons, not single-domain business logic
4. clarify that future UI convergence must center on a unified skeleton instead of copying historical templates again
5. clarify definition of done and acceptance discipline for platform changes

## 4. Non-Goals

This contract does not:

- design a specific `business/*` page
- carry detailed business logic for `system/auth`, `system/iam`, `system/org`, or `system/config`
- define every field and interaction of every system page
- replace detailed implementation design docs

This contract also does not define “low-code platform” as Pantheon’s current primary goal.

Low-code capabilities may exist inside platform governance, but must not rewrite the core positioning of the platform layer.

## 5. Boundaries

### 5.1 Covered Surfaces

- `dashboard` / workbench / platform home
- application shell: navigation, top bar, tabs, brand area, layout switching
- platform-level shared skeletons: `PageContainer`, `PageHeader`, `FilterPanel`, `PageSplitLayout`, `SideRail`, `AppModal`, `AppDrawer`
- platform-level feedback states: loading / empty / error / forbidden / submitting
- platform-level navigation icon semantics
- platform-level acceptance templates, migration matrices, and PR discipline

### 5.2 Excluded Surfaces

- internal auth/session models in `system/auth`
- internal authorization data structures in `system/iam`
- internal organization governance in `system/org`
- internal storage and runtime asset details in `system/config`
- business interactions inside specific business-module pages

## 6. Dependencies

This contract depends on:

- [DESIGN.md](../../DESIGN.md)
- [AGENTS.md](../../AGENTS.md)
- [BACKEND.md](../designs/BACKEND.md)
- [FRONTEND.md](../designs/FRONTEND.md)
- [FRONTEND_UI_SPEC.md](../designs/FRONTEND_UI_SPEC.md)
- [FRONTEND_PAGE_TEMPLATES.md](../designs/FRONTEND_PAGE_TEMPLATES.md)
- [PLATFORM_DASHBOARD_DESIGN.md](../designs/PLATFORM_DASHBOARD_DESIGN.md)
- [ACCEPTANCE_CHECKLIST.md](../acceptances/ACCEPTANCE_CHECKLIST.md)

## 7. Hard Constraints

### 7.1 Aggregate-Layer Constraints

- dashboard, workbench, home overview, and cross-domain statistic cards belong to `platform`
- the aggregate layer may read multiple subdomains, but must not invade their internal responsibilities
- every new platform card must declare its source domain, destination, and permission boundary

### 7.2 Shell Constraints

- navigation, top bar, tabs, brand area, and layout switching belong to the platform shell
- left navigation is for navigation only; it must not carry help cards, stats cards, or explanation cards
- vertical sidebar and horizontal top-nav modes must share the same state language
- runtime language resolution priority is fixed: current login choice > user preference > system default
- login-page language choice is a platform-scoped session override, not an automatic write-back of long-term user preference
- current-session language override must be cleared on logout and must not leak to the next login principal

### 7.3 Skeleton Constraints

- the platform layer owns page skeletons, right-side assist rails, and overlay baselines
- business pages must not reintroduce old right-side templates or bypass platform overlay wrappers
- the platform layer may let system-domain pages fill content, but system domains may not invent new shell modes

### 7.3.1 Table Interaction Constraints

- cross-page selection is a `platform` interaction contract, not a current-page-only behavior
- changing page number or page size must not clear the selection set
- search terms, filters, sort fields, sort direction, and reset actions change query context and therefore must clear the selection set
- batch actions must execute against the full selection set, not just visible rows on the current page
- current-page checkbox state should be derived from full selection set intersected with current-page visible data

### 7.4 Page Admission Constraints

- task-first pages should not let governance drawers or governance hero blocks take over the main narrative, for example `/system/user`
- configuration pages may show overview cards and risk notes, but must not reintroduce right-side governance menus that duplicate tabs or breadcrumbs, for example `/system/setting`
- only governance-heavy and audit-heavy pages may use governance drawers, such as roles, permissions, organization, or logs
- page admission rules must be maintained as a shared checklist and enforced by automation and smoke whitelists, not by per-page memory

### 7.5 Documentation Constraints

- all platform-layer design, assessment, remediation, and acceptance docs must link back to this contract
- dated platform-layer evaluation drafts should be deleted or archived once replaced by newer matrices or norms

## 8. Definition of Done

The platform layer should count as currently complete only when:

### 8.1 Responsibility Completion

- `dashboard` is stably owned by `platform`
- shell, navigation, overlays, and platform-level states have clear ownership

### 8.2 Visual and Interaction Completion

- left navigation and top-nav modes share the same menu-state language
- right-side assist rails have moved from historical templates to platform skeleton semantics
- business-layer spread of raw `Modal.*` and raw `<Drawer>` has stopped

### 8.3 Documentation and Process Completion

- the platform layer has a stable migration matrix
- the platform layer has a fixed dual-mode acceptance template
- shell-related submissions must include acceptance records and scan summaries

### 8.4 Regression-Control Completion

- no new legacy shell class names are introduced
- no new business-layer overlay entry points bypass platform wrappers
- platform indexes and primary docs are no longer drowned out by temporary evaluation drafts
- page admission rules are enforced by automation and smoke whitelists, not by human memory

## 9. Acceptance Standards

Platform-related changes must at least pass:

### 9.1 Documentation Acceptance

- [ACCEPTANCE_CHECKLIST.md](../acceptances/ACCEPTANCE_CHECKLIST.md)
- [DOCUMENT_GOVERNANCE_CONTRACT.md](../contracts/DOCUMENT_GOVERNANCE_CONTRACT.md)
- [DOCUMENT_METADATA_AND_STATUS.md](../contracts/DOCUMENT_METADATA_AND_STATUS.md)

### 9.2 UI Acceptance

- [PLATFORM_ACCEPTANCE_MATRIX_20260430_UI_MIGRATION.md](../acceptances/PLATFORM_ACCEPTANCE_MATRIX_20260430_UI_MIGRATION.md)
- [PLATFORM_SHELL_DUAL_MODE_ACCEPTANCE_TEMPLATE.md](../acceptances/PLATFORM_SHELL_DUAL_MODE_ACCEPTANCE_TEMPLATE.md)

### 9.3 Fixed Scans

- `rg "system-page-side|system-page-summary-card|system-page-note|system-page-main-grid|system-page-main" frontend/src`
- `rg "Modal\\.confirm|Modal\\.(success|error|info|warning)" frontend/src`
- `rg "<Modal|<Drawer" frontend/src/modules frontend/src/components`

### 9.4 Build and Regression

- `cd frontend && npm run build`
- if shell interaction or a system-page primary path changed, add page-level smoke or an acceptance record

## 10. Related Documents

### 10.1 Design

- [PLATFORM_DASHBOARD_DESIGN.md](../designs/PLATFORM_DASHBOARD_DESIGN.md)
- [FRONTEND_UI_SPEC.md](../designs/FRONTEND_UI_SPEC.md)
- [FRONTEND_PAGE_TEMPLATES.md](../designs/FRONTEND_PAGE_TEMPLATES.md)

### 10.2 Acceptance

- [ACCEPTANCE_CHECKLIST.md](../acceptances/ACCEPTANCE_CHECKLIST.md)
- [PLATFORM_ACCEPTANCE_MATRIX_20260430_UI_MIGRATION.md](../acceptances/PLATFORM_ACCEPTANCE_MATRIX_20260430_UI_MIGRATION.md)
- [PLATFORM_SHELL_DUAL_MODE_ACCEPTANCE_TEMPLATE.md](../acceptances/PLATFORM_SHELL_DUAL_MODE_ACCEPTANCE_TEMPLATE.md)
