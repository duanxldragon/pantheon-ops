---
title: Frontend Page Template Specification
doc_type: Design
layer: platform
status: Active
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
  - docs/contracts/SYSTEM_IAM_CONTRACT.md
  - docs/contracts/SYSTEM_ORG_CONTRACT.md
updated_at: 2026-04-17
---

# Frontend Page Template Specification

Chinese version: [FRONTEND_PAGE_TEMPLATES.md](./FRONTEND_PAGE_TEMPLATES.md)

This document defines the standard page templates for Pantheon Base frontend delivery. Its purpose is to stop every page from inventing its own layout, stop AI from generating a different style every time, and keep enterprise-admin flows on one stable information architecture.

Use this together with:

- `docs/designs/FRONTEND_UI_SPEC.md`
- `docs/designs/MODULE_CONTRACT.md`
- `docs/designs/PERMISSION_MODEL.md`
- `docs/designs/ERROR_CODE_AND_I18N.md`

## 1. Page Type Overview

Pantheon Base recognizes these standard page families:

- `ListPage`
- `TreePage`
- `DetailPage`
- `FormPage`
- `ConfigPage`
- `DashboardPage`
- `AuthPage`
- `ProfilePage`
- `StatePage`

## 2. Common Page Structure

Most pages should follow:

```text
Page
  ├── PageHeader
  ├── PageToolbar / ActionBar
  ├── PageContent
  └── PageState
```

### 2.1 `PageHeader`

Contains:

- title
- optional description
- breadcrumb handoff area when needed
- primary actions on the right

Rules:

- titles use i18n keys
- keep primary actions to two or fewer
- if the shell already provides breadcrumbs, do not duplicate them in the content area
- if a summary or hero card follows, it must not repeat the page title

### 2.2 `PageContent`

The body uses surface panels, stable spacing rhythm, and should avoid arbitrary large inline styling.

### 2.3 `PageState`

Every page must account for:

- loading
- empty
- error
- forbidden
- submitting

## 3. `ListPage` Template

Typical uses:

- users
- roles
- posts
- permission policies
- dictionary items

### 3.1 Standard Structure

```text
ListPage
  ├── PageHeader
  ├── FilterPanel
  ├── TableCard
  │   ├── Table
  │   └── Pagination
  └── CreateEditModal / Drawer
```

### 3.2 `FilterPanel`

Rules:

- place above the table
- keep simple filters within one row where possible
- support collapse after roughly four fields
- align action buttons on the right in the order `Reset`, then `Search`
- filters must coordinate with pagination and sorting
- use the shared `FilterPanel` component instead of page-local `Card + Form` rewrites
- rely on platform spacing and density tokens rather than page-local overrides

### 3.3 `TableCard`

Rules:

- use table-level loading
- separate first-empty from no-result empty states
- fix the action column to the right
- default action order: view, edit, delete, more
- keep table container padding, radii, header background, and pagination alignment on platform tokens
- avoid brand-tinted or gradient table headers

### 3.3.1 List Action Bar

Use shared components for:

- header utility actions such as import, export, and refresh
- primary header actions such as create or generate
- batch actions after row selection
- governance cleanup bars for retention-driven log cleanup

Pages must not locally override spacing, control height, or alignment patterns for these shared bars.

### 3.4 Modal and Drawer

- use `Modal` for simple create or edit flows
- use `Drawer` for more complex forms
- use `Popconfirm` or confirmation modal for dangerous actions

### 3.5 Permission Rules

Typical mapping:

- page entry requires `view`
- list query requires `list`
- create button requires `create`
- edit button requires `update`
- delete button requires `delete`

### 3.6 Definition of Done

A finished list page includes:

- filtering
- table
- pagination
- loading
- empty state
- error state
- button permissions
- i18n
- feedback for create, edit, and delete flows

## 4. `TreePage` Template

Typical uses:

- departments
- menus
- organization trees

### 4.1 Standard Structure

```text
TreePage
  ├── PageHeader
  ├── FilterPanel
  ├── TreeTableCard
  └── CreateEditDrawer
```

### 4.2 Tree-Table Rules

- stable same-level sorting
- clear parent-child hierarchy
- fixed action column
- no delete for nodes that still have children
- support expand and collapse

### 4.3 Special Rules for Menu Trees

Menu management must display:

- menu type
- route path
- component path
- permission markers
- visibility state
- sort order

## 5. `DetailPage` Template

### 5.1 Standard Structure

Use:

- page header
- summary area
- section cards
- bottom or local actions

### 5.2 `SummaryCard`

The summary card gives key overview context only. It should not turn into a duplicate dashboard or a long handbook.

### 5.3 `SectionCard`

Use section cards to break detailed information into stable, scan-friendly groups.

### 5.4 Permission Rules

Detail pages should guard page entry separately from edit or destructive actions.

## 6. `FormPage` Template

### 6.1 Standard Structure

Use a page header, structured form sections, and a stable submit area.

### 6.2 Form Rules

- keep labels, help copy, and validation states consistent
- dangerous fields require stronger feedback
- do not turn placeholders into rule manuals

## 7. `ConfigPage` Template

### 7.1 Standard Structure

Use left-side grouping or navigation with a right-side config panel and a stable save bar.

### 7.2 Configuration Categories

Keep categories explicit and operational rather than mixing all settings into one long page.

### 7.3 Save Rules

Configuration saves must show clear success, validation, encryption, and audit behavior.

## 8. `DashboardPage` Template

### 8.1 Standard Structure

Include:

- summary metrics
- quick actions
- attention items
- recent activity
- risk or alert summaries

### 8.2 Design Rules

- do not build a card wall
- keep first-screen metrics limited
- favor actionable content over decorative overview

### 8.3 Recommended Metrics

Metrics should reflect platform or domain actionability, not arbitrary decorative counts.

## 9. `AuthPage` Template

### 9.1 Standard Structure

Keep auth pages focused on authentication tasks and explicit security feedback.

### 9.2 Design Rules

The auth surface is a security console, not a marketing page. No fake controls, no oversized hero theatrics.

## 10. `ProfilePage` Template

### 10.1 Standard Structure

Keep a stable self-service layout for current-user information and personal settings.

### 10.2 Section Suggestions

Typical sections include identity basics, contact information, security entry points, and preference management.

## 11. `StatePage` Template

### 11.1 Standard Structure

Use one unified experience for global exception pages.

### 11.2 `403`

Must clearly explain access denial without pretending the page succeeded.

### 11.3 `404`

Must clearly signal missing routes and offer a safe recovery path.

### 11.4 `500 / network error`

Must distinguish infrastructure or network failure from business validation problems.

## 12. Template Selection Rules

Choose the page template by job-to-be-done first. Do not start from whatever component a previous page happened to use.

## 13. Rules for AI-Generated Pages

AI-generated pages must:

- identify the correct page family first
- reuse the shared skeleton
- keep text on i18n
- keep permissions explicit
- keep states explicit
- avoid inventing a fresh visual system on each screen

## 14. Current Codebase Refactor Guidance

Use these templates as the migration target for existing pages that still diverge in layout, filter rhythm, table shell, or state handling.

## 15. Definition of Done

A page is not done when it merely renders. It is done when its template, permissions, state handling, i18n, and interaction rhythm are aligned with the shared platform contract.

## 16. Recommended Next Document

Continue with the shared component plan and the detailed UI specification.
