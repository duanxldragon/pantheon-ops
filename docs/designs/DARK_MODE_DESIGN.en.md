---
title: Dark Mode Design
doc_type: Design
layer: platform
status: Draft
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
updated_at: 2026-07-09
---

# Dark Mode Design

Chinese version: [DARK_MODE_DESIGN.md](./DARK_MODE_DESIGN.md)

This document is kept as a future design note. The current runtime only ships light mode with four theme keys. There is no `data-color-mode`, no dark token set, and no dark screenshot baseline. The real token source of truth is `THEME_TOKENS_REFERENCE.md`.

---

## 1. Design Goals

These are future requirements for a dark-mode rollout:

- do not rely on naive inversion
- keep each brand theme recognizable in dark mode
- avoid pure black backgrounds
- avoid pure white text
- keep focus rings readable and accessible

---

## 2. Token Inversion Strategy

Not implemented today. If this document is revived, semantic roles should be redefined instead of mechanically inverted.

Guiding direction:

| Semantic role            | Future dark direction                     |
| ------------------------ | ----------------------------------------- |
| App background           | deep blue-gray, not pure black            |
| Default surface          | one step brighter than the app background |
| Elevated surface         | one more step brighter                    |
| Primary text             | near-white, not pure white                |
| Brand accent             | brighter and slightly less saturated      |
| Subtle accent background | deep brand-tinted panels                  |
| Shadow                   | darker and more opaque                    |

---

## 3. Brand-Color Adjustment Rules

If dark mode is added later, accent colors should:

1. get brighter
2. lose a little saturation
3. keep the same hue
4. flip subtle accent backgrounds into dark brand panels

---

## 4. Elements That Cannot Be Naively Inverted

These elements will still need explicit treatment:

- screenshots
- logos
- business charts
- code blocks
- zebra tables
- toast / notification borders

---

## 5. Shadow Strategy

Future dark mode should use deeper, higher-opacity shadows, sometimes with a subtle highlight edge so overlays remain visually separated.

---

## 6. Focus Rings

If dark mode is ever implemented, focus rings should use the brightened accent color and remain clearly visible.

---

## 7. Switching Behavior

The current runtime does **not** provide a switch. If this is restored later:

- switching should only change root attributes
- CSS variables should recalculate
- the page should not reload
- the component tree should not rebuild

---

## 8. Dark-Mode-Specific Notes

Future work will need to account for:

- transparent PNG edges
- iframe mode synchronization
- print output forced back to light
- light / dark screenshot evidence

---

## 9. Implementation Checklist

Everything below is currently unshipped:

- `<html data-color-mode="light|dark">`
- early boot-time color-mode hydration
- dark token variables
- contrast checks across 4 themes × 2 modes
- no-flash switching
- clear overlay layering
- dark variants for charts and code blocks

---

## 10. Acceptance

If dark mode is reintroduced later, acceptance should include:

- screenshots for the main pages across 4 themes × 2 modes
- user preference preserved after refresh
- system preference honored when no explicit choice exists
- CSS fallback working without JavaScript

---

## 11. Related Documents

- `THEME_TOKENS_REFERENCE.md`
- `ACCESSIBILITY.md`
- `FRONTEND_UI_SPEC.md`
- `FRONTEND.md`
