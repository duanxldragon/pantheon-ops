---
title: Frontend UI Detailed Specification
doc_type: Design
layer: platform
status: Active
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
updated_at: 2026-05-12
---

# Frontend UI Detailed Specification

Chinese version: [FRONTEND_UI_SPEC.md](./FRONTEND_UI_SPEC.md)

This English companion captures the governing UI rules from the Chinese primary document. It is intended to support international collaborators without changing the Chinese-first authoring model.

## 1. Design Goals

- standard enterprise backoffice tone: restrained, stable, clear, credible
- high reuse across CRUD, detail, and configuration pages
- safe for multilingual text expansion
- explicit permission states including forbidden, partial access, and read-only
- desktop first, with controlled downgrade paths for smaller screens

## 2. Design DNA

### 2.1 Keywords

- Indigo
- Enterprise
- Clean
- Structured
- Calm
- Figma-inspired
- Low-AI-smell

### 2.1.1 Multi-Theme Strategy

Theme capability belongs to the `platform` layer. Login, dashboard, system pages, dialogs, drawers, messages, and notifications share one token system.

The first recommended themes are:

- `indigo`: default enterprise backoffice
- `emerald`: collaboration and operations tone
- `violet`: platform and intelligence tone
- `slate`: low-saturation office tone

User shell preferences currently converge on `theme`, `layoutMode`, and `densityMode`.

### 2.2 Visual Principles

- less decoration, more structure
- weaker separators, stronger rhythm
- scannability over showiness
- operational confidence over visual impact
- restrained interface chrome, expressive business content
- hierarchy should come from type, weight, density, and spacing, not large gradients

### 2.3 References and Tradeoffs

Pantheon Base borrows from AI-readable design systems and Figma-like precision, but does not copy Figma marketing pages. Color is reserved for state, emphasis, charts, and selective accents. All rules must map back to Arco Design, tokens, and shared components.

## 3. Design Tokens

This document defines semantic usage. Final numeric values live in `THEME_TOKENS_REFERENCE.md`.

### 3.1 Color

Core semantic roles:

- accent for brand and primary CTA
- app background
- default surface
- default border
- primary, secondary, and tertiary foreground text
- success, warning, and error status colors

### 3.1.1 Figma-Inspired Color Strategy

Keep the base UI mostly neutral. Let color belong to content and critical feedback. Avoid gradient-heavy hero treatments, rainbow icon cards, and glassmorphism as defaults.

### 3.2 Corner Radius

Use semantic radius groups:

- control radius for inputs and selectors
- action radius for buttons
- panel radius for cards and table containers
- overlay radius for modal and drawer shells

### 3.3 Shadow

Use small shadows for static cards, medium shadows for dropdowns, and strong shadows for modal-level overlays.

### 3.4 Spacing

Use platform spacing scales only. Avoid page-local freehand spacing numbers.

### 3.5 Type Scale

Use semantic type roles such as caption, body-sm, body, heading, and display. Keep page titles and numeric emphasis at `600`. Avoid pseudo-premium custom font weights.

### 3.5.1 Typography Strategy

Use `Source Sans 3` as the preferred backoffice body font, with system fallbacks. Do not switch the admin UI to `Inter` by default. Use monospace only for tags, permission codes, and route paths.

## 4. Overall Layout Rules

### 4.1 Application Shell

The shell must support both:

- left sidebar navigation by default
- top horizontal navigation as a user preference

Shell preferences are platform-level concerns and persist through `PUT /api/v1/auth/me/preferences`. Runtime language priority is fixed as:

`login-session choice > user preference > system default`

Density switching may change rhythm for headers, cards, tables, pagination, and action bars, but not page responsibilities.

### 4.2 Sidebar

- expanded width: roughly `220-240px`
- collapsed width: roughly `56-64px`
- white background with a subtle right divider

The sidebar only handles wayfinding. It must not become a summary panel, help panel, or statistics container.

### 4.2.1 Sidebar Visual Details

- menu item height around `40px`
- modest radii, not exaggerated pills
- selected state uses light indigo fill plus indigo text
- linear icons around `16px`
- stable indentation for nested items

### 4.2.2 Sidebar Information Architecture

- reflect `platform`, `system/auth`, `system/iam`, `system/org`, `system/config`, and `business/*` boundaries
- avoid collapsing all system pages into one flat mega-menu
- keep only one explicit expanded level

### 4.3 Top Bar

Must include:

- breadcrumb
- theme switch
- language switch
- user info
- profile entry
- logout

### 4.3.1 Top Bar Visual Details

- height around `56px`
- subtle breadcrumb hierarchy
- icon-first action buttons with tooltips
- minimal user-area chrome

## 5. Navigation and Information Architecture

### 5.1 Recommended First-Level Navigation

- dashboard
- access control
- organization
- platform configuration
- security audit

### 5.2 Breadcrumb Rules

- home is always the first item
- generate the full parent chain from the menu tree
- profile and security center may register their own breadcrumb metadata

### 5.3 Tab Strategy

Distinguish breadcrumbs from open tabs. Tabs are for recent page switching and should support close-current, close-others, close-right, pinning, drag sorting, and title resolution from route or menu `titleKey`.

## 6. Page Skeleton Rules

### 6.1 Standard List Page

Required order:

1. page header
2. action bar
3. filter area
4. table area
5. pagination

Use a side rail only when there is real contextual or risk information.

### 6.1.1 List-Page Visual Details

Keep the title and primary actions aligned. Use white surfaces for filters and tables. Only one visual center is allowed on the screen.

### 6.2 Standard Detail Page

Use a title area, basic information card, sectioned detail blocks, and a bottom action bar.

### 6.3 Standard Configuration Page

Use left-side category navigation, a right-side configuration panel, and a sticky save bar.

### 6.3.1 High-Sensitivity Configuration Admission Rules

Pages such as `/system/modules`, `/system/generator`, `/system/i18n`, and `/system/setting` must avoid summary-card walls, concept-heavy intros, and mixed responsibilities.

### 6.4 Standard Dashboard

Dashboard pages should include:

- summary metrics
- quick entry points
- pending actions
- recent activity
- alert summaries

### 6.5 Figma-Inspired Component Shapes

Allowed references:

- pill controls for filters and segmented states
- circular icon-only actions
- consistent focus outlines
- light color accents in dashboard graphics

Avoid huge gradient heroes, oversized marketing headlines, and unreadably thin admin typography.

### 6.6 Right Side Rail Rules

The right side rail is a secondary context container owned by the platform shell.

#### 6.6.1 Single Responsibility

It may only carry:

- current context
- real risk and next-step prompts

#### 6.6.2 Explicitly Forbidden

- duplicated KPI cards
- a second dashboard
- long handbook-style explanations
- arbitrary miscellaneous information
- visual treatment that outshines the main table

#### 6.6.3 Allowed Templates

- `SummaryRail`
- `RiskRail`
- `PolicyRail`

#### 6.6.4 Layout Constraints

- width around `280-320px`, or `360px` only on very wide screens
- no more than three stacked cards by default
- card descriptions should stay within `2-4` lines

#### 6.6.5 Page Mapping

- `system/iam`: light summary context only
- `system/org`: issue counts and positioning entry points only
- `system/config`: emphasize runtime risk and cache state
- `system/audit` and `system/auth`: concise, actionable risk context only
- `business/*`: disable by default unless real workflow context exists

#### 6.6.6 Implementation Boundary

The shell owns the side-rail width, spacing rhythm, and token usage. Domain pages may fill content, but may not invent new side-rail layouts.

#### 6.6.7 Migration Rules

Legacy classes such as `system-page-side`, `system-page-summary-card`, and `system-page-note` are migration leftovers and must not become reference patterns for new pages.

## 7. Form Rules

### 7.1 Grid

Use a 24-column grid on desktop and preserve responsive props across `xs`, `sm`, `md`, and `lg`.

### 7.2 Labels and Help Copy

Keep labels left-aligned and concise. Put help text below the field. Do not overload placeholders with rules.

### 7.3 Validation

Separate:

- required-field errors
- format errors
- business-rule errors
- submit failures

All error copy must go through i18n keys.

### 7.4 Dangerous Fields

Passwords, disabling states, deletes, permission changes, and system-level settings require stronger warning semantics and secondary confirmation.

## 8. Table Rules

### 8.0 Semantic Column Width Rules

Column width belongs to the shared platform table contract. Prefer semantic aliases from `frontend/src/components/patterns/TableColumnWidth.ts` over raw numbers.

### 8.1 Column Types

Common types:

- text
- status
- time
- tags
- related entity
- actions

### 8.2 Action Column

Default order:

- view
- edit
- delete

Fold extra actions into a `More` menu after three visible actions.

### 8.3 Empty States

Distinguish:

- first-use empty state
- no-result empty state
- forbidden empty state
- load-failed state

### 8.4 Batch Actions

Batch interaction must bind selection to the full query context rather than only to the current page.

## 9. State Design

Every page must define:

- `loading`
- `success`
- `empty`
- `error`
- `forbidden`
- `submitting`

### 9.1 `loading`

Use skeletons or scoped loading states. Do not flash the whole page.

### 9.2 `empty`

Separate first-use emptiness from no-result emptiness.

### 9.3 `error`

Cover network, server, timeout, and generic data-load failures.

### 9.4 `forbidden`

Provide one unified 403 experience for both page-level and action-level denial.

### 9.5 Focus and Accessibility

Every interactive element must have a clear focus state. Do not rely on color alone. Table action buttons need readable labels or tooltips.

## 10. Permission Interaction Rules

### 10.1 Three Permission Layers

- navigation permission
- page-entry permission
- action permission

### 10.2 Page Forbidden

Prefer pre-entry guard checks or a unified 403 page, not a late API failure that leaves the page half-loaded.

### 10.3 Button Forbidden

Hide by default. In selected educational or high-risk cases, disabled plus tooltip is acceptable.

## 11. Multilingual Design Rules

### 11.1 Copy Rules

All user-visible copy must go through i18n, including titles, column labels, buttons, prompts, modal copy, errors, and empty states.

### 11.2 Key Naming

Prefer domain-based namespaces such as:

- `common.*`
- `auth.*`
- `system.user.*`
- `system.role.*`
- `system.permission.*`
- `biz.order.*`

### 11.3 Long-Text Compatibility

Layouts must tolerate English, German, and other longer languages without breaking buttons, filters, or page headers.

## 12. Responsive Rules

### 12.1 Breakpoints

Numerical breakpoints and downgrade rules live in `MOBILE_RESPONSIVE_BREAKPOINTS.md`.

### 12.2 Mid and Small Screen Strategy

- collapsible filters
- single-column forms
- card-view or drawer-filter fallback when needed
- drawer-style navigation

### 12.3 Current-Phase Strategy

Desktop first is acceptable, but implementation may not assume desktop-only usage forever.

## 13. Motion Rules

- light fade for page transitions
- unified easing and duration for drawers and modals
- subtle hover feedback
- danger actions emphasize clarity, not entertainment

## 14. Global Experience Consistency

This section continues the backoffice remediation baseline.

### 14.1 Modal and Drawer

- use shared radii, borders, panel backgrounds, and shadows
- `Modal` is for short focused tasks
- `Drawer` is for longer detail or edit flows that must preserve context
- do not compress full pages into overlays
- keep footer order consistent: secondary on the left, primary on the right

### 14.1.1 Overlay Migration Constraints

New `platform`, `auth`, and `system/*` pages should not directly invent raw native `Modal` usage for business overlays. Existing raw overlays are historical debt, not new examples.

### 14.2 Message and Notification

Use restrained semantic colors, subtle shadows, and i18n-based copy.

### 14.3 Login and App-Shell Consistency

Login must share tokens with the app shell. It is an authentication console, not a marketing landing page. Do not show fake clickable capabilities.

### 14.4 Application Shell Consistency

Use neutral surfaces, stable table spacing, right-aligned filter buttons, fixed right-side action columns, and a default `middle` table density.

### 14.5 Platform Dashboard Consistency

Dashboard belongs to the platform aggregate layer. Keep the first screen to a small set of primary metrics and prioritize actionable content over decorative summaries.

## 15. Recommended Page Inventory

### 15.1 P0

- permission workbench
- security center
- online sessions
- login logs
- dictionary management
- system settings

### 15.2 P1

- user detail page
- menu icon picker
- live dashboard workbench
- 403, 500, and network exception pages

## 16. Anti-AI-Slop Rules

### 16.1 Forbidden Patterns

- default purple-blue gradients
- generic three-column feature cards with colorful icons
- identical card radius, shadow, and spacing everywhere
- centered everything
- oversized hero headlines in admin pages
- gradient primary buttons
- content-free card stacks
- marketing-page styling for table pages
- fake or unimplemented controls

### 16.2 Required Enterprise Backoffice Qualities

- solve the task first
- stable alignment for columns, filters, and actions
- explicit empty, error, and forbidden states
- sparse and intentional icon use
- restrained semantic color
- short, clear motion
- every page should look like part of one system

### 16.3 Correct Figma Borrowing

Borrow tool-like precision, content-first composition, geometric clarity, and clean focus behavior. Do not turn the admin UI into a brand-marketing site.

## 17. AI Code Generation Constraints

Generated frontend code must:

- identify the page type first
- apply the corresponding skeleton directly
- route all copy through `t()`
- handle all required states explicitly
- avoid nonstandard inline styles
- keep permissions, routes, menus, and i18n keys in sync

Also forbidden:

- generating the legacy right-rail structures again
- copying a current system page as the universal template
- inventing raw `Modal` or `Drawer` patterns inside `platform`, `auth`, or `system/*`
- placing helper cards, risk cards, or summary cards inside the left sidebar

## 18. References

- `awesome-design-md`
- `getdesign.md` Figma-inspired design reference
- Google Stitch `DESIGN.md` concept references

## 19. Related Documents

- `docs/designs/FRONTEND.md`
- `docs/designs/AUTH_MODULE_DESIGN.md`
- `DESIGN.md`
- `AGENTS.md`
