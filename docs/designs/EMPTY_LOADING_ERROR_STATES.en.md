---
title: Empty / Loading / Error State Rules
doc_type: Design
layer: platform / system/*
status: Active
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
updated_at: 2026-05-11
---

# Empty / Loading / Error State Rules

Chinese version: [EMPTY_LOADING_ERROR_STATES.md](./EMPTY_LOADING_ERROR_STATES.md)

This document defines the visual, copy, interaction, and ARIA behavior for non-happy-path page states in Pantheon Base.

Use it together with:

- `FRONTEND_UI_SPEC.md` state-design rules
- `FRONTEND_PAGE_TEMPLATES.md`
- `ACCESSIBILITY.md`

The UI spec explains the principles. This document gives page-type-specific variants.

## 1. State Categories

Core states:

- `loading`
- `empty-initial`
- `empty-filtered`
- `error-network`
- `error-server`
- `forbidden`
- `not-found`
- `submitting`

## 2. Shared Visual Spec

All states should use one shared `<PageState>` wrapper with:

- centered icon
- heading title
- body description
- primary and optional secondary CTA

Rules:

- minimum container height is roughly 60% of the visible parent height
- icons stay monochrome and tertiary in emphasis
- title and body follow shared type roles
- the whole state is centered
- CTA visibility still follows permission rules

## 3. List-Page Variants

Required variants:

- `loading`: table container uses `aria-busy="true"`
- `empty-initial`: guide the user to create the first resource
- `empty-filtered`: guide the user to clear or adjust filters
- `error-network`: retry-oriented network failure state
- `error-server`: retry plus escalation-oriented server failure state
- `forbidden`: access-denied state with safe exit

Copy rules:

- descriptions stay within two short sentences
- do not use vague text such as “unknown error” without guidance

## 4. Detail-Page Variants

Rules:

- loading should preserve layout through skeletons
- not-found uses a full-page state with return-to-list guidance
- server error uses full-page retry and return navigation
- forbidden uses a full-page denial state
- partial section failure should degrade at section level rather than failing the whole page

## 5. Form-Page Variants

Rules:

- edit-mode loading uses form skeletons
- submitting disables submit and keeps content visible, typically readonly
- validation errors move focus to the first invalid field
- server submission failures show a top-level alert while preserving user input
- forbidden must block the page before showing an empty form

Never clear the user’s filled data on submit failure.

## 6. Dashboard Variants

Dashboard widgets must fail independently:

- one widget loading must not block the whole dashboard
- one widget error must stay local
- one widget empty state should use a compact mini-state
- full-page forbidden applies only when the whole dashboard is inaccessible

## 7. States Inside Trees, Drawers, and Tabs

Rules:

- tree expansion loading uses local spinners, not full-screen masks
- tab switches use skeleton placeholders instead of flashing
- drawers and modals may show internal loading without blocking their open animation

## 8. Copy Library: Standard I18n Keys

All state copy must go through i18n keys, including:

- list empty
- filtered empty
- network error
- server error
- forbidden
- not found
- retry
- clear filters
- go back
- go to workbench
- contact admin

Modules may override these with specialized keys, but may not omit equivalent state coverage.

## 9. ARIA and Accessibility

State-specific ARIA rules:

- `loading`: `aria-busy="true"`
- `empty-*`: `role="status"`
- `error-*`: `role="alert"`
- `forbidden` and `not-found`: `role="alert"`
- `submitting`: disabled action plus progress announcement

## 10. Acceptance

Acceptance should verify:

- each new page implements the relevant state set for its page type
- state transitions do not cause flashing or layout collapse
- copy uses i18n keys
- error states provide concrete next steps
- local failures do not unnecessarily collapse whole pages

## 11. Related Documents

- `FRONTEND_UI_SPEC.md`
- `FRONTEND_PAGE_TEMPLATES.md`
- `FRONTEND_COMPONENT_PLAN.md`
- `ACCESSIBILITY.md`
- `THEME_TOKENS_REFERENCE.md`
