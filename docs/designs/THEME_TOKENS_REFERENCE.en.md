---
title: Theme Tokens Reference
doc_type: Design
layer: platform
status: Active
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
updated_at: 2026-07-09
---

# Theme Tokens Reference

Chinese version: [THEME_TOKENS_REFERENCE.md](./THEME_TOKENS_REFERENCE.md)

This document records the real implementation in `frontend/src/index.css`. It is the single light-mode source of truth. It no longer promises unshipped `--pantheon-*`, dark-mode, or generic spacing/type/motion/z-index systems. When styling changes, align with this file and `frontend/src/index.css` first.

---

## 1. Naming Rules

The current implementation uses these token groups:

- `--brand-*`: brand colors and transparency derivatives
- `--panel-*`: surfaces, borders, shadows
- `--text-*`: text colors
- `--radius-*`: radii
- `--shell-*`: shell spacing and layout rhythm
- `--color-*`: semantic status colors
- `--primary-1..10`: per-theme RGB ramps for `color-mix()` / `rgba()` helpers

Compatibility aliases remain in place for migration only:

- `--danger-soft`
- `--danger-border`
- `--status-success-soft`
- `--status-warning-soft`

---

## 2. Neutral Tokens

| Token                   | Value                                                                 | Purpose                 |
| ----------------------- | --------------------------------------------------------------------- | ----------------------- |
| `--app-bg`              | `#f7f8fa`                                                             | App background          |
| `--app-grid-line`       | `rgba(29, 33, 41, 0.035)`                                             | Background grid         |
| `--app-shell-overlay`   | `#ffffff`                                                             | Shell overlay base      |
| `--panel-bg`            | `#ffffff`                                                             | Default surface         |
| `--panel-bg-solid`      | `#ffffff`                                                             | Solid card/control base |
| `--panel-muted`         | `#f7f8fa`                                                             | Muted surface           |
| `--panel-border`        | `rgba(229, 230, 235, 0.82)`                                           | Default border          |
| `--panel-border-strong` | `rgba(209, 213, 219, 0.78)`                                           | Strong border           |
| `--panel-shadow`        | `0 12px 28px rgba(15, 23, 42, 0.055)`                                 | Main shadow             |
| `--panel-shadow-soft`   | `0 1px 2px rgba(15, 23, 42, 0.04), 0 8px 18px rgba(15, 23, 42, 0.03)` | Soft elevation          |
| `--panel-shadow-strong` | `0 18px 42px rgba(15, 23, 42, 0.14)`                                  | Strong elevation        |
| `--text-primary`        | `#1f2329`                                                             | Primary text            |
| `--text-secondary`      | `#4e5969`                                                             | Secondary text          |
| `--text-tertiary`       | `#86909c`                                                             | Tertiary text           |

---

## 3. Brand Themes

The four built-in themes are mounted through `:root[data-pantheon-theme=...]`.

### 3.1 indigo

- `--brand-primary`: `#165dff`
- `--brand-primary-soft`: `rgba(22, 93, 255, 0.12)`
- `--brand-primary-muted`: `rgba(22, 93, 255, 0.08)`
- `--brand-gradient`: `#165dff`
- `--shell-sider-bg`: `#162033`
- `--shell-sider-elevated`: `#1d2940`
- `--shell-brand-shadow`: `0 12px 28px rgba(22, 93, 255, 0.24)`
- `--login-glow`: `rgba(22, 93, 255, 0.12)`

### 3.2 emerald

- `--brand-primary`: `#00a870`
- `--brand-primary-soft`: `rgba(0, 168, 112, 0.13)`
- `--brand-primary-muted`: `rgba(0, 168, 112, 0.08)`
- `--brand-gradient`: `#00a870`
- `--shell-sider-bg`: `#143126`
- `--shell-sider-elevated`: `#1a3d32`
- `--shell-brand-shadow`: `0 12px 28px rgba(0, 168, 112, 0.26)`
- `--login-glow`: `rgba(0, 168, 112, 0.14)`

### 3.3 violet

- `--brand-primary`: `#722ed1`
- `--brand-primary-soft`: `rgba(114, 46, 209, 0.13)`
- `--brand-primary-muted`: `rgba(114, 46, 209, 0.08)`
- `--brand-gradient`: `#722ed1`
- `--shell-sider-bg`: `#22193b`
- `--shell-sider-elevated`: `#2c2350`
- `--shell-brand-shadow`: `0 12px 28px rgba(114, 46, 209, 0.26)`
- `--login-glow`: `rgba(114, 46, 209, 0.14)`

### 3.4 slate

- `--brand-primary`: `#334155`
- `--brand-primary-soft`: `rgba(51, 65, 85, 0.12)`
- `--brand-primary-muted`: `rgba(51, 65, 85, 0.08)`
- `--brand-gradient`: `#334155`
- `--shell-sider-bg`: `#182235`
- `--shell-sider-elevated`: `#243148`
- `--shell-brand-shadow`: `0 12px 28px rgba(51, 65, 85, 0.24)`
- `--login-glow`: `rgba(51, 65, 85, 0.12)`

> Each theme also defines a `--primary-1..10` RGB ramp in code. Those exact ramp numbers live in `frontend/src/index.css` and are intentionally not duplicated here.

---

## 4. Semantic Status Tokens

| Token                | Value                      | Purpose            |
| -------------------- | -------------------------- | ------------------ |
| `--color-success`    | `#00b42a`                  | Success            |
| `--color-success-bg` | `rgba(0, 180, 42, 0.12)`   | Success background |
| `--color-warning`    | `#ff7d00`                  | Warning            |
| `--color-warning-bg` | `rgba(255, 125, 0, 0.12)`  | Warning background |
| `--color-error`      | `#f53f3f`                  | Error              |
| `--color-error-bg`   | `rgba(245, 63, 63, 0.12)`  | Error background   |
| `--color-info`       | `#3491fa`                  | Info               |
| `--color-info-bg`    | `rgba(52, 145, 250, 0.12)` | Info background    |

Alias mapping:

- `--danger-soft` -> `--color-error-bg`
- `--danger-border` -> `color-mix(in srgb, var(--color-error) 32%, transparent)`
- `--status-success-soft` -> `--color-success-bg`
- `--status-warning-soft` -> `--color-warning-bg`

---

## 5. Spacing Scale

There is no shared `--space-*` scale yet. The shell uses `--shell-*` variables for layout rhythm:

| Token                                    | Value           | Purpose                       |
| ---------------------------------------- | --------------- | ----------------------------- |
| `--shell-page-gap`                       | `16px`          | Page spacing                  |
| `--shell-page-header-gap`                | `12px`          | Page header spacing           |
| `--shell-panel-body-padding`             | `18px`          | Panel padding                 |
| `--shell-panel-head-min-height`          | `56px`          | Panel header min height       |
| `--shell-page-split-gap`                 | `16px`          | Split layout gap              |
| `--shell-page-main-gap`                  | `16px`          | Main column gap               |
| `--shell-page-side-gap`                  | `12px`          | Side column gap               |
| `--shell-table-cell-padding-y`           | `10px`          | Table cell vertical padding   |
| `--shell-table-cell-padding-x`           | `14px`          | Table cell horizontal padding |
| `--shell-table-card-padding`             | `12px 14px 6px` | Table card body padding       |
| `--shell-table-pagination-padding`       | `12px 14px 2px` | Pagination padding            |
| `--shell-table-action-min-height`        | `32px`          | Table action button height    |
| `--shell-control-min-height`             | `32px`          | Generic control height        |
| `--shell-filter-body-padding`            | `14px 16px 6px` | Filter body padding           |
| `--shell-filter-control-min-height`      | `34px`          | Filter control height         |
| `--shell-filter-form-item-margin-bottom` | `12px`          | Filter form item spacing      |
| `--shell-filter-label-padding-bottom`    | `4px`           | Filter label spacing          |
| `--shell-list-actions-gap`               | `8px 12px`      | Row action spacing            |
| `--shell-action-bar-gap`                 | `8px 12px`      | Batch action spacing          |
| `--shell-action-bar-min-height`          | `32px`          | Batch action height           |
| `--shell-table-head-gap`                 | `8px 12px`      | Table header spacing          |
| `--shell-governance-select-width`        | `200px`         | Governance selector width     |

---

## 6. Typography

There is no shared type scale token yet. `body` still uses a system font stack, and component CSS writes font sizes directly.

The truth is:

- there is no `--font-size-*` / `--line-height-*`
- there is no global `Source Sans 3` import
- code/data font special cases exist only in local component CSS

If typography unification becomes a goal later, the implementation needs to come first.

---

## 7. Radius

| Token              | Value              | Purpose                    |
| ------------------ | ------------------ | -------------------------- |
| `--radius-xs`      | `4px`              | Smallest radius            |
| `--radius-sm`      | `4px`              | Small buttons / tags       |
| `--radius-md`      | `6px`              | Inputs / standard controls |
| `--radius-lg`      | `8px`              | Cards / overlays           |
| `--radius-overlay` | `8px`              | Overlay / modal baseline   |
| `--radius-control` | `var(--radius-md)` | Control alias              |
| `--radius-action`  | `var(--radius-sm)` | Action button alias        |
| `--radius-pill`    | `999px`            | Pills / badges             |

There is no `--radius-xl`, and the current implementation does not encourage free-form radius values beyond this scale.

---

## 8. Shadows

| Token                   | Value                                                                 | Purpose          |
| ----------------------- | --------------------------------------------------------------------- | ---------------- |
| `--panel-shadow`        | `0 12px 28px rgba(15, 23, 42, 0.055)`                                 | Main card shadow |
| `--panel-shadow-soft`   | `0 1px 2px rgba(15, 23, 42, 0.04), 0 8px 18px rgba(15, 23, 42, 0.03)` | Soft elevation   |
| `--panel-shadow-strong` | `0 18px 42px rgba(15, 23, 42, 0.14)`                                  | Strong overlay   |

There are no dark-mode shadow tokens because the runtime does not ship dark mode.

---

## 9. Motion

There is no shared `--motion-*` token yet. Components use raw `0.18s` / `0.2s` transition values and `prefers-reduced-motion` for fallback.

That means:

- motion is an implementation detail, not a published token contract
- if motion tokenization is ever added, code must change first and docs second

---

## 10. Z-Index

There is no shared `--z-*` scale yet. Layering mainly relies on Arco defaults and local stacking contexts.

That means:

- dropdowns, tooltips, modals, and drawers still resolve layering through component behavior and local CSS
- if a shared z-index scale becomes necessary, it should be implemented before it is documented

---

## 11. Theme Switching Rules

- themes mount through `:root[data-pantheon-theme="indigo|emerald|violet|slate"]`
- switching a theme only changes CSS variables; it does not rebuild the DOM
- the runtime currently supports light mode only
- there is no `data-color-mode`
- `DARK_MODE_DESIGN.md` is Deferred, not shipped

---

## 12. Acceptance

- all 4 themes render correctly under the same shell contract
- Dashboard, list pages, row actions, and status hints consume semantic tokens
- no new runtime references to `--pantheon-*`
- `dashboard.css` is covered by the shell visual contract checker
- dark mode is not described as an already-delivered feature
