---
title: Frontend Component Plan
doc_type: Design
layer: platform
status: Active
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
updated_at: 2026-04-17
---

# Frontend Component Plan

Chinese version: [FRONTEND_COMPONENT_PLAN.md](./FRONTEND_COMPONENT_PLAN.md)

This document continues the visual and interaction rules from `FRONTEND_UI_SPEC.md` and the structural rules from `FRONTEND_PAGE_TEMPLATES.md`. It answers the practical question: which shared frontend components should Pantheon Base establish, and how should they be layered so that pages stop rebuilding the same shells repeatedly.

## 1. Design Goals

- unify enterprise-admin page skeletons
- reduce repeated CRUD-page work
- preserve a restrained and coherent backoffice feel
- avoid a component system that drifts even when pages technically function
- let both AI and human engineers build on the same component mental model

## 2. Component Layers

Pantheon Base should think in four layers.

### 2.1 Design Token Layer

Owns:

- color
- spacing
- radius
- shadow
- type scale
- line height
- z-index
- multi-theme token mapping such as `indigo`, `emerald`, `violet`, and `slate`

This layer should not carry business logic or page-specific coupling. It should land primarily through Arco theme configuration and CSS variables. Login, layout, modal, drawer, message, and notification must all share the same token set.

### 2.2 Primitive Layer

Wrap lightweight reusable UI atoms such as:

- `AppIcon`
- `AppLink`
- `AppTag`
- `AppStatusBadge`
- `AppEmptyIllustration`

These are broadly reusable and should not bind to specific business modules.

### 2.3 Pattern Layer

This is the most important backoffice layer. It should encapsulate recurring interaction shapes such as:

- page header
- filter area
- table toolbar
- list-state region
- form area
- detail area
- permission states

Business pages should rely on this layer as their primary infrastructure.

### 2.4 Module Layer

This is where module-specific reusable components live, such as an order timeline, SLA card, or project member matrix. These should not pollute the global shared layer.

## 3. Suggested Directory Structure

Recommended evolution:

```text
frontend/src/components/
  primitives/
  patterns/
  feedback/
  data-display/
  data-entry/
  navigation/
  auth/
  permission/
```

Module-private components stay under:

```text
frontend/src/modules/{system|business}/{module}/components/
```

Rules:

- cross-module shared components go to `src/components`
- module-local reusable components stay inside the module
- do not dump business-specific parts into the global library

## 4. First Batch of Required Shared Components

## 4.1 Page Skeleton Components

### `PageContainer`

Owns page container width, padding, and vertical rhythm.

### `PageHeader`

Owns title, subtitle, breadcrumb handoff, right-side primary actions, and alignment rhythm with shell-level entries such as theme or language controls.

### `ThemeSwitcher`

Owns platform-theme switching and theme-preference persistence. It belongs to the platform core rather than to any one system subdomain.

### `PageSection`

Owns titled content grouping and spacing rhythm between sections.

### `PageActions`

Owns create, export, batch, and more-actions patterns while keeping action priority and dangerous-action placement consistent.

## 4.2 List-Page Components

### `FilterPanel`

Owns query form layout, collapse behavior, reset and search actions, and advanced filters.

### `DataToolbar`

Owns batch actions, column visibility, refresh, export, and right-side utility actions.

### `AppTable`

Wraps Arco Table with consistent handling for:

- loading
- empty states
- row keys
- permission-aware columns
- text overflow
- fixed action columns
- density and scrolling strategy

### `TableSelectionBar`

Owns selected-count display and batch operations such as export, delete, or status updates.

## 4.3 Form Components

### `FormPage`

Owns page-level form layout, primary actions, cancel or back behavior, and leave-confirmation flow.

### `FormSection`

Owns grouped form areas with stable internal rhythm.

### `SubmitBar`

Owns stable submit-area layout and primary-versus-secondary action order.

## 4.4 Detail Components

### `DetailPage`

Owns the high-level layout for detail-oriented screens.

### `DetailSection`

Owns grouped detail blocks and scan-friendly structure.

### `TimelineCard`

Provides a consistent pattern for chronology-based detail context where it is actually needed.

## 4.5 Feedback and State Components

### `PageLoading`
### `PageEmpty`
### `PageError`
### `PageForbidden`
### `InlineError`
### `ConfirmDangerAction`

These components unify page-level and inline feedback, forbidden states, and dangerous action confirmation behavior.

## 4.6 Permission and Routing Components

### `PermissionGuard`
### `ActionGuard`
### `RouteMetaTitle`

These components keep page entry, action gating, and route-title behavior consistent with shared permission and i18n rules.

## 4.7 Navigation Components

### `AppSider`
### `AppTopbar`
### `BreadcrumbBar`

These belong to the shell and must preserve one coherent navigation language across layouts.

## 4.8 Rail-Pattern Components

### `PageSplitLayout`
### `GovernanceRailPanel`
### `GovernanceRailSummary`
### `StandardRailSummary`
### `StandardRailNotePanel`

These components should be the only sanctioned way to build side-rail layouts. They must stay aligned with the side-rail rules in the UI spec and must not regress into ad hoc right-column inventions.

## 5. Components That Should Not Be Abstracted Too Early

Do not rush to extract global components for behaviors that are still unstable, domain-specific, or only appear once. The goal is to extract repeatable patterns, not to build an abstract library for its own sake.

## 6. Component Design Constraints

### 6.1 Naming Rules

Names should reflect stable UI or interaction roles, not temporary page wording.

### 6.2 Styling Rules

Shared components must consume tokens and shared shells instead of embedding page-local style systems.

### 6.3 Permission Rules

Permission-sensitive components should consume the shared permission model rather than introducing custom gating logic in each page.

### 6.4 I18n Rules

All user-visible text must remain key-driven and translatable.

### 6.5 Prefer Composition Over Inheritance

Compose smaller stable pieces into richer page patterns. Avoid inheritance-heavy abstractions.

## 7. Mapping Templates to Components

Page templates should map directly to shared components. Templates define the structural contracts; components make those contracts executable.

## 8. Recommended Rollout Order

### P0

Land the skeleton, list, form, state, and navigation foundations first.

### P1

Expand into richer detail, governance, and shell-variant patterns.

### P2

Extract deeper module-facing abstractions only after the shared patterns are proven.

## 9. Component Definition of Done

A shared component is done when it:

- solves a recurring pattern rather than a one-off page
- uses platform tokens
- supports permission and i18n requirements where relevant
- keeps enterprise-admin rhythm intact
- reduces divergence instead of creating another local style island

## 10. Boundary with Other Documents

- `FRONTEND_UI_SPEC` defines the visual and interaction rules
- `FRONTEND_PAGE_TEMPLATES` defines the page skeleton contracts
- this document defines the shared component system that realizes those rules

## 11. Current-Phase Conclusion

Before large-scale page implementation continues, the component layers and reuse strategy need to stay explicit. Otherwise the codebase will accumulate working pages without a stable frontend system.
