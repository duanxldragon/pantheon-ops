---
title: Backoffice UI Style Constraints
doc_type: Design
layer: platform
status: Active
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
updated_at: 2026-05-01
---

# Backoffice UI Style Constraints

Chinese version: [BACKOFFICE_STYLE_CONSTRAINTS.md](./BACKOFFICE_STYLE_CONSTRAINTS.md)

This document constrains the shared backoffice UI style implementation in Pantheon Base and is meant to stop recurring style drift in side rails, overlays, cards, tables, and filter areas.

It is an implementation-level companion to `FRONTEND_UI_SPEC.md`, not a replacement for it.

## 1. Ownership and Boundary

The shared style language belongs to the `platform` layer.

- `platform` owns shared tokens, shell layout, overlay systems, side-rail systems, and global feedback visuals
- system domains and business modules consume those rules rather than inventing their own border, radius, or overlay language

## 2. Audit Conclusions

The repository currently shows dual-track style drift in:

- right-side rails
- overlays
- cards and panels

The problem is not one bad page. It is style fragmentation across multiple shared surfaces.

## 3. Mandatory Implementation Constraints

### 3.1 Right Side Rail

Must use shared side-rail layout and components.

Forbidden:

- page-local `Card`-based long-term side-rail patterns
- multiple title-pattern styles inside one rail
- using the side rail as a second main content column

### 3.2 Overlay System

Must unify on:

- `AppModal`
- `AppDrawer`
- shared confirm/success/error modal helpers

Do not mix raw modal styles or page-local radius and border overrides.

### 3.3 Cards and Panels

Shared surfaces must follow one radius, border, shadow, and spacing language.

### 3.4 System Table Visual Contract

System-domain list pages must share:

- one table-card shell
- one header-background language
- one radius contract
- one fixed-column visual treatment

### 3.5 System Filter and Action Visual Contract

System-domain pages must share:

- `FilterPanel`
- shared filter spacing tokens
- shared list-header action bars
- shared batch-action and governance-action bars

## 4. Shared Token Constraints

### 4.1 Required Shared Tokens

Use the platform panel, shadow, and radius tokens rather than page-local values.

### 4.2 Legacy Patterns That Must Not Spread

Do not keep expanding:

- old border-token usage as a primary panel language
- ad hoc 14px or 18px corner radii
- page-local modal and drawer shell rewrites

## 5. Component Usage Rules

### 5.1 Page Layout

Use the sanctioned split-layout, single-column, and side-rail component patterns.

### 5.2 Overlays

Use shared overlay components by intent: edit, detail, danger confirm.

### 5.3 Direct-Writing Patterns That Are Forbidden

Do not recreate side-rail structures or overlay shells directly in page CSS as a long-term pattern.

## 6. Acceptance Checklist

Any shared-backoffice visual change should verify:

- side rails still use shared components
- panel borders stay unified
- modal and drawer overlay radii stay unified
- summary, governance, and side-rail cards still share the same visual language
- no new page-level hardcoded border, radius, or shadow values appear

## 7. Recommended Governance Order

When visual drift is found:

1. check whether shared components were bypassed
2. check whether local tokens or freehand style values were added
3. check for dual-track global versus local overrides
4. only then do page-level fine tuning

## 8. Relationship with Existing Documents

- `FRONTEND_UI_SPEC.md`
- `BACKOFFICE_UI_REMEDIATION_PLAN_20260423.md`
