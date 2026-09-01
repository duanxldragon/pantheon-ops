---
title: Navigation Information Architecture Deep-Dive
doc_type: Design
layer: platform / system/iam
status: Active
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
  - docs/contracts/SYSTEM_IAM_CONTRACT.md
updated_at: 2026-05-11
---

# Navigation Information Architecture Deep-Dive

Chinese version: [NAVIGATION_IA_STRATEGY.md](./NAVIGATION_IA_STRATEGY.md)

This document defines the deeper rules for menu IA, external links, iframe pages, tab caching, breadcrumbs, and active-menu behavior.

Use it together with:

- `FRONTEND_UI_SPEC.md`
- `PERMISSION_MODEL.md`
- `MOBILE_RESPONSIVE_BREAKPOINTS.md`

## 1. Boundary

- navigation visuals, shell modes, tabs, and breadcrumbs belong to `platform`
- menu metadata, page permissions, button permissions, and component keys belong to `system/iam`
- business modules declare their own entry points, but do not control shell behavior directly
- top-level navigation should prefer business-domain entry points before platform workbench and system governance entries

## 2. Menu Types

Core types:

- `M`: directory
- `C`: page
- `F`: button or action permission node
- external link
- iframe page

Rules:

- directory nodes do not carry page business logic
- page nodes must carry path, route name, title key, component key, and page permission
- action nodes do not appear in sidebar navigation
- external links must declare whether they open in a new window
- iframe entries must pass whitelist and safety rules

## 3. Highlighting and Breadcrumbs

- detail, create, and edit pages usually do not appear as sidebar entries
- such pages must use `activeMenu` to point back to their parent list page
- breadcrumbs should come from route metadata and the menu tree, not hardcoded natural-language text

## 4. Tabs and Caching

If tab caching is enabled:

- list pages may cache filters and pagination
- form pages should not keep in-flight submit state by default
- high-sensitivity pages must not cache sensitive inputs
- iframe pages require their own cache and destroy strategy review

## 5. External-Link and Iframe Security

- external links must declare `isExternal`
- iframes must declare whitelist domains, sandbox policy, and permission boundary
- dynamic menus must not let administrators bypass iframe policy by writing arbitrary URLs

## 6. Menu State Machine

Each menu item may be:

- hidden
- visible
- hover
- active
- loading

Rules:

- hidden items do not render and do not reserve space
- active items must expose page-current semantics
- loading states use skeleton behavior
- “visible but disabled” is forbidden for no-permission nodes; hide them instead

## 7. Detailed Tab-Caching Algorithm

Each tab should keep lightweight metadata such as:

- route name
- title key
- active menu id
- page type
- cache policy
- scroll position
- optional filters and pagination state

### 7.1 Cache Policy

Suggested cache policies:

- `memo`
- `fresh`
- `sensitive`

Recommended defaults:

- list and dashboard pages -> `memo`
- detail and most form pages -> `fresh`
- security and secret-bearing pages -> `sensitive`

### 7.2 Tab Count Limit

- default limit around 12
- user-adjustable range
- recently closed tabs may be reopened for a short grace period

### 7.3 Force Refresh

Right-click refresh should force one `fresh` cycle without changing the stored cache policy.

Parameterized detail route changes should be treated as new instances under the detail-page `fresh` rule.

## 8. Breadcrumb Generation Algorithm

Breadcrumbs must not be handwritten by pages.

The unified breadcrumb component should derive them from:

- matched route chain
- route metadata title keys
- `activeMenu` fallback where needed

### 8.1 Three Node Types

- directory node
- list-page node
- current-page node

### 8.2 Dynamic Parameters

Only the last breadcrumb node may be dynamic, such as a resource name.

### 8.3 Multi-Level Folding

When the chain grows too deep, middle nodes may collapse into an expandable ellipsis.

## 9. Iframe Whitelist and Safety Constraints

### 9.1 Whitelist Format

Whitelist entries should carry:

- exact domain
- sandbox policy
- whether `postMessage` is allowed
- description

### 9.2 Validation When Creating Menus

When an iframe menu is saved:

1. parse the URL protocol and domain
2. require `https`
3. require exact whitelist match rather than broad wildcard trust
4. inherit sandbox policy from the whitelist record

### 9.3 Runtime Enforcement

Runtime iframe rendering should force:

- sandbox attributes
- strict referrer policy
- safe lazy loading
- no override from dynamic menu payload

### 9.4 `postMessage` Protocol

If postMessage is allowed, parent windows must still validate requested navigation targets before honoring them.

## 10. External-Link UX

- single-click opens a new window safely
- browser-default middle-click and modifier-click behavior should remain
- visual external-link indicators should be additive rather than replacing translated titles

External links do not participate in tabs, breadcrumbs, or cache policy.

## 11. Global Search Navigation

Suggested behavior:

- `/` or `Cmd+K` opens a command panel
- search source is current-user-accessible menus plus recent paths
- fuzzy match titles, aliases, and path keywords
- navigation result still respects tab cache policy

Search results must never reveal hidden no-permission menu items.

## 12. Navigation Differences by Tenant or Role

- role or tenant differences should usually change node visibility, not the overall conceptual tree shape
- empty directories caused by hidden children should also be hidden
- tenant switches should trigger a fresh menu-metadata fetch

## 13. Route Restoration

After refresh, the shell should restore:

- current route and params
- open-tab metadata
- sidebar collapse state
- active theme

It does not need to restore all internal page component state by default.

## 14. Navigation Under Mobile Breakpoints

Mobile navigation behavior must follow the responsive breakpoint rules:

- drawer-based shell navigation
- simplified top bar
- different tab behavior as needed

## 15. Instrumentation and Observability

Navigation should eventually be observable enough to answer:

- what users try to navigate to
- which nodes are most frequently opened
- whether activeMenu or breadcrumb recovery fails
- whether tab restore and cache behavior are working as intended

## 16. Acceptance

Acceptance should verify:

- active highlighting and breadcrumb behavior
- tab cache policy correctness
- iframe and external-link safety enforcement
- hidden-node behavior for permission constraints
- route restore and responsive navigation behavior

## 17. Related Documents

- `FRONTEND_UI_SPEC.md`
- `PERMISSION_MODEL.md`
- `MOBILE_RESPONSIVE_BREAKPOINTS.md`
- `ACCESSIBILITY.md`
