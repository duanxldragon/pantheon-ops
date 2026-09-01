---
title: Frontend Architecture and UI Rules
doc_type: Design
layer: platform
status: Active
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
updated_at: 2026-04-17
---

# Frontend Architecture and UI Rules

Chinese version: [FRONTEND.md](./FRONTEND.md)

This document is the architecture-level companion for Pantheon Base frontend design. For the detailed page skeletons, navigation, state, form, table, responsive, and permission rules, read [FRONTEND_UI_SPEC.en.md](./FRONTEND_UI_SPEC.en.md).

Numeric source-of-truth boundaries:

- visual token values are defined by `docs/designs/THEME_TOKENS_REFERENCE.md`
- responsive breakpoints and downgrade behavior are defined by `docs/designs/MOBILE_RESPONSIVE_BREAKPOINTS.md`
- this document and `FRONTEND_UI_SPEC` define architecture, semantics, and rules, not the final number tables

## 1. Architecture Goals: Modular, Declarative, Decoupled

The frontend base is a shell. Business modules register themselves through explicit configuration rather than patching the shell ad hoc.

### 1.1 Current-Phase Principles

- finish design before implementation
- lock boundaries before writing pages
- settle shared skeletons before building business modules
- let the platform shell own left navigation, right side rail, and overlay containers

Authentication, security, permission, and configuration pages must define page type, interaction states, and module boundaries in documentation before coding starts.

## 2. Visual Direction: The Indigo Identity

- Core tone: Indigo blue with restrained neutral gray support.
- Visual reference: AI-readable Markdown design systems plus Figma-like precision, light geometry, and controlled color accents.
- Anti-patterns: default purple-blue gradients, generic three-column feature cards, icon piles, and content-free card walls.
- Enterprise baseline: login, shell, dashboard, and system pages must feel calm, credible, and tool-like.
- Layering:
  - base background
  - white surface panels with subtle borders
  - white overlays with diffused shadow
- Typography:
  - strict 1.25 scale rhythm
  - title `600`, emphasized body `500`, regular body `400`
- Depth:
  - subtle shadows for static cards
  - stronger shadows for overlays and active components

Additional shell constraints:

- the left sidebar only handles navigation
- the right side rail only carries secondary context and risk prompts
- `Modal` and `Drawer` belong to one shared overlay system

## 3. Core Interaction Details

- Prefer lineless layouts and use spacing rhythm instead of heavy separators.
- Status colors must stay semantic and restrained.
- Buttons darken slightly on hover and compress slightly on active.
- Inputs use indigo focus plus a controlled outer glow.
- Motion stays short and `ease-out`, with no entertainment-style bounce transitions.

## 4. Module Registration

- business modules live in `src/modules/business/`
- system modules live in `src/modules/system/`
- each page module exports a manifest from `index.ts`
- `src/core/router/modules.ts` registers modules explicitly

### 4.1 Example Registration

```typescript
export const OrderModule = {
  name: 'order',
  routes: [
    { path: 'order/list', titleKey: 'biz.order.menu.list', component: React.lazy(() => import('./pages/list')) }
  ]
};
```

In the live codebase, `ModuleConfig` has already evolved into a richer manifest with `scope`, `menus`, `permissions`, `i18nNamespaces`, and `pagePermission`.

## 5. Dynamic I18n

- `i18next` is the core engine.
- The app loads runtime translations from `/api/v1/system/i18n/pack`.
- Built-in fallback resources exist for `zh-CN`, `en-US`, `ja-JP`, `ko-KR`, and `fr-FR`.
- Modules keep their own `locales/{locale}.json` files.
- `npm run i18n:generate-module` scans module locales and generates runtime resources.
- UI text must use `t()` or `<Trans />`.
- Runtime translation updates should be refreshable without full page reload.
- Request fallback messages must resolve through i18n keys, not hardcoded English.
- Import/export result summaries and conflict prompts must also follow the translation chain.
- Team-internal Chinese modeling vocabulary is allowed; the hard requirement applies to user-facing runtime text.
- Generators must be key-first and may not embed natural-language labels directly into source code.

## 6. Component Development Standard

- Prefer Arco Design layout and composition instead of large amounts of custom CSS.
- Standardize form validation and business-error display.
- Standardize request handling for token injection, `401/403`, and request tracing.
- Before build, run:
  - `npm run i18n:generate-module`
  - `npm run check:i18n-missing-keys`
  - `npm run check:i18n-hardcode`
- After locale expansion or large translation changes, also run `npm run audit:i18n-locales`.

## 7. Current Feature Closure

The current frontend already covers:

- login and MFA flows under `src/modules/auth`
- shell-level idle timer, activity heartbeat, and lock-screen flow
- security center, session management, and login log pages
- request auto-refresh behavior in `src/api/request.ts`
- route-level page permission guards
- user, role, menu, department, post, permission, profile, setting, dictionary, and audit pages
- dashboard data wired to live platform summaries
- platform preference persistence through `GET/PUT /api/v1/auth/me/preferences`
- unified page skeleton components under `src/components/`
- differentiated `network / timeout / server / business` error handling

## 8. Interaction Additions

- system lists support coordinated filter, pagination, sorting, and batch actions
- button visibility follows fine-grained permission keys through `usePermission`
- selection sets for batch actions are query-context based, not page-local
- role authorization separates navigation, page, and action permissions
- menu metadata supports `pagePerm`, `perms`, `routeName`, `module`, `isCache`, `isExternal`, and `activeMenu`
- sensitive settings use encrypted-state interaction instead of showing plaintext
- import/export flows use structured result dialogs instead of fragile toast-only feedback
- shell, menus, headers, and import/export feedback all participate in language switching verification

## 9. Gaps and Required Companion Documents

`FRONTEND.md` only defines direction and current capability coverage. Large-scale page delivery must also follow:

- `docs/designs/FRONTEND_UI_SPEC.md`
- `docs/designs/AUTH_MODULE_DESIGN.md`
- `DESIGN.md`

The detailed UI spec fixes three shell containers as platform-level rules:

- left navigation for wayfinding and domain boundaries only
- right side rail for context and risk only
- `Modal` and `Drawer` for short tasks or continuous editing, not compressed pages
