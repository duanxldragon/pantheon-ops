---
title: Mobile Responsive Breakpoints
doc_type: Design
layer: platform
status: Active
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
updated_at: 2026-05-11
---

# Mobile Responsive Breakpoints

Chinese version: [MOBILE_RESPONSIVE_BREAKPOINTS.md](./MOBILE_RESPONSIVE_BREAKPOINTS.md)

This document defines Pantheon Base breakpoints, container-width strategy, and component downgrade behavior across breakpoints.

Use it together with `FRONTEND_UI_SPEC.md`. The UI spec defines the responsive principles. This document defines the numeric breakpoints and concrete component behavior.

When breakpoint values, list-page downgrade behavior, or mobile-component fallback rules conflict with other docs, this document is the source of truth.

## 1. Priority and Boundary

- Pantheon is desktop-first
- mobile is a degraded but usable experience, not a mobile-first redesign
- main target screens are desktop and laptop widths, with occasional tablet and limited phone use
- do not optimize for game-like touch interaction or mobile-native card-wall patterns

## 2. Breakpoint Definitions

Breakpoints:

- `xs`: `< 480px`
- `sm`: `480 - 767px`
- `md`: `768 - 1023px`
- `lg`: `1024 - 1279px`
- `xl`: `1280 - 1599px`
- `2xl`: `>= 1600px`

The Chinese primary document remains the authoritative numeric table.

## 3. Shell Behavior Across Breakpoints

Behavior summary:

- `xl` and `2xl`: full sidebar, full top bar, tab bar visible
- `lg`: sidebar rail mode
- `md`: drawer-style sidebar, compact shell
- `sm` and `xs`: drawer sidebar, simplified top bar, tab bar hidden

Rules:

- rail mode starts at `lg`
- sidebar becomes drawer-only at `md`
- extra vertical space is preserved on the smallest screens

## 4. List-Page Behavior Across Breakpoints

Behavior summary:

- `xl` and `2xl`: full filter row and full table
- `lg`: fewer visible filters and lower-priority columns hidden
- `md`: filter drawer plus reduced table columns
- `sm` and `xs`: card rendering instead of classic tables

Column visibility should be driven by declared priority rather than ad hoc hiding.

## 5. Detail-Page Behavior Across Breakpoints

Behavior summary:

- large screens may use outline, main content, and side rail
- medium screens collapse into simpler main-column flows
- small screens become one-column stacks

## 6. Form-Page Behavior Across Breakpoints

Behavior summary:

- large screens may use 2-3 columns with side labels
- medium screens move labels above fields and reduce columns
- small screens become single-column with full-width bottom actions

## 7. Dashboard Behavior Across Breakpoints

Behavior summary:

- wider screens use 3-4 columns
- medium screens use 2 columns
- small screens use 1 column
- mobile chart cards should simplify rather than just shrink

## 8. Component-Level Downgrade Rules

### 8.1 Table

- hide lower-priority columns below `md`
- fully switch to card rendering below `sm`
- keep overflow actions available

### 8.2 Modal and Drawer

- centered modal and right drawer above `md`
- full-screen modal and bottom-sheet drawer below `md`

### 8.3 Dropdown and Select

- normal dropdown on desktop
- action-sheet-style fallback on the smallest screens

### 8.4 Datepicker

- richer calendar on desktop
- simpler calendar or native date input on small screens

### 8.5 Tooltip

- no hover-only tooltip dependence on touch devices
- critical tooltip meaning must have visible fallback

### 8.6 Form Field

- desktop controls may stay smaller
- mobile controls should respect touch-target accessibility sizes

## 9. Container Width Strategy

Different container types should cap at different widths:

- business-page main areas
- forms
- article or detail content
- dashboards with uncapped card grids where appropriate

## 10. Type Behavior Across Breakpoints

Type tokens do not change by breakpoint. Density changes through `densityMode` instead.

Recommended:

- mobile defaults toward more comfortable density
- desktop defaults to standard density

## 11. Screen Orientation

- no forced portrait-only mode
- horizontal phone or tablet layout still follows the same breakpoint logic
- third-party iframe orientation remains the third party’s responsibility

## 12. Acceptance

Every new page should be checked at least at:

- `xl`
- `lg`
- `md`
- `sm`

Minimum bar:

- all interactions remain accessible
- no horizontal scrolling
- no overlap or truncation
- touch-target sizes remain compliant

## 13. Related Documents

- `FRONTEND_UI_SPEC.md`
- `FRONTEND_PAGE_TEMPLATES.md`
- `BACKOFFICE_STYLE_CONSTRAINTS.md`
- `ACCESSIBILITY.md`
