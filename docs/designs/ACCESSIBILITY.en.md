---
title: Accessibility Design Rules
doc_type: Design
layer: platform / system/*
status: Active
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
updated_at: 2026-05-11
---

# Accessibility Design Rules

Chinese version: [ACCESSIBILITY.md](./ACCESSIBILITY.md)

This document is Pantheon Base’s accessibility design and acceptance specification. It complements `FRONTEND_UI_SPEC.md` and focuses on interaction accessibility rather than repeating visual-token rules.

The goal is that keyboard users, screen-reader users, and low-vision users can complete core administrative tasks.

## 1. Scope and Minimum Standard

Minimum baseline:

- WCAG 2.1 AA
- full keyboard completion of interactive flows
- validation with NVDA + Chrome on Windows and VoiceOver + Safari on macOS
- text contrast and control-boundary contrast at AA levels
- touch targets at least `44x44` CSS px on mobile

AAA is not required, but critical flows such as login, form submission, and batch actions must meet AA.

## 2. Keyboard Navigation

### 2.1 Focus Order

- tab order must follow visual reading order
- decorative elements should be skipped
- tables and dense list internals should use arrow-key models where appropriate rather than forcing tab-through-every-cell behavior
- when modal or drawer opens, focus must enter it automatically and return to the trigger when it closes

### 2.2 Visible Focus

- every focusable element needs a clearly visible focus ring
- focus indication must not rely on color alone
- focus contrast must stay at or above `3:1` against adjacent backgrounds
- `outline: none` is forbidden unless a valid replacement is provided

### 2.3 Skip Link

- every protected page after login must provide a “skip to main content” link
- the target should be the main content `<main>` region or a stable `id="page-content"` anchor

### 2.4 Shortcuts

Suggested global shortcuts:

- `/` for search
- `Esc` for closing the topmost overlay
- `?` for the shortcut panel

Local shortcuts may exist, but they must not conflict with global shortcuts and must have visible help.

## 3. ARIA and Semantics

### 3.1 Required ARIA Rules

Examples:

- icon-only buttons must have `aria-label` or equivalent naming
- decorative icons must be `aria-hidden="true"`
- dialogs and drawers require `role="dialog"` plus `aria-labelledby`
- toasts and notifications should use `role="status"` or `role="alert"` depending on severity
- tables need semantic headers and selection state exposure
- form errors need `aria-describedby` and `aria-invalid`
- loading containers should use `aria-busy="true"`
- route changes should announce the new page title once the transition completes

### 3.2 Things Not to Do

- do not replace semantic `<button>` or `<a>` with generic `<div role="button">` when a native element fits
- do not hide focusable elements under `aria-hidden="true"`
- do not misuse `tabindex` in ways that break child accessibility

## 4. Form Accessibility

### 4.1 Required

- every control needs a visible `<label>`
- placeholder must not replace the label
- required fields need both visual marking and `aria-required`
- error text must be linked through `aria-describedby`
- submit-time validation should move focus to the first invalid field and announce the error

### 4.2 Composite Controls

- radio and checkbox groups should use `<fieldset>` and `<legend>`
- custom select controls should implement proper ARIA combobox or listbox patterns
- date pickers must support direct keyboard entry, not only visual clicking

## 5. Dynamic Content and Async States

Examples of required announcement strategy:

- announce result counts after list filtering
- announce success after form submission
- announce failure with `role="alert"` when submission fails
- announce route-title changes after navigation
- toggle `aria-busy` during async loads
- avoid constant unsolicited announcements for auto-refreshing dashboards

## 6. Color and Contrast

- normal body text contrast must be at least `4.5:1`
- large text may use the `3:1` threshold
- placeholder text must still meet accessibility contrast
- error, warning, and success states must not rely on color alone

Detailed color tokens still belong to the theme-token and dark-mode design documents.

## 7. Mobile Touch Accessibility

- touch targets at least `44x44` CSS px
- spacing between targets at least `8px`
- long-press, double-tap, or swipe behaviors must have keyboard or explicit fallback alternatives
- drawers and modals should support swipe-down dismissal on mobile but must also expose a clear close button

## 8. Acceptance Checklist Before Merge

Each new page or component should verify:

- keyboard-only completion of the main flow
- understandable structure even with CSS disabled
- readable name, role, and state in screen readers
- screen-reader announcement of form errors
- visible focus rings with sufficient contrast
- ARIA coverage for loading, empty, error, and forbidden states
- contrast compliance in both light and dark themes
- correct tab order
- focus entry and return for modals
- compliant mobile touch-target sizing

## 9. Automated Checks

- pre-build accessibility checks, such as an `axe-core` rule set
- end-to-end accessibility assertions in browser-based QA, including route-change announcements, focus return, and error-focus behavior

## 10. Related Documents

- `FRONTEND_UI_SPEC.md`
- `THEME_TOKENS_REFERENCE.md`
- `DARK_MODE_DESIGN.md`
- `EMPTY_LOADING_ERROR_STATES.md`
