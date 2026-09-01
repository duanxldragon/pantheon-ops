---
title: Business Lifecycle Detail Pattern
doc_type: Design
layer: business/* (abstracted from CMDB + Deploy patterns in pantheon-ops)
status: Active
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
updated_at: 2026-05-11
---

# Business Lifecycle Detail Pattern

Chinese version: [BUSINESS_LIFECYCLE_DETAIL_PATTERN.md](./BUSINESS_LIFECYCLE_DETAIL_PATTERN.md)

This document defines the standard implementation pattern for business resource detail pages, especially resources with real lifecycle states such as deploy jobs or CMDB hosts.

Use it together with `BUSINESS_RESOURCE_LIST_PATTERN.md`: that document covers list pages, while this one covers detail pages with lifecycle behavior.

It does not replace the generic detail-page shell in `FRONTEND_PAGE_TEMPLATES.md`. It adds business-specific patterns such as hero stats, lifecycle actions, and child-resource tables.

## 1. When to Use This Pattern

Use this pattern when the resource detail page has one or more of the following:

- a status field with multiple values
- executable state-transition actions
- related child-resource lists
- metadata blocks such as owner, created time, or related entities

Do not use it for:

- minimal resources that fit inside list-page modal editing
- pure read-only descriptions-only pages
- highly specialized tree or graph visualizations

## 2. File Structure

Recommended structure:

```text
frontend/src/modules/business/<module>/<resource>/
├── api.ts
├── <Resource>List.tsx
├── <Resource>Form.tsx
├── <Resource>Detail.tsx
└── locales/
```

## 3. Required Platform Components

The detail page should rely on shared platform components such as:

- `PageContainer`
- `PageHeader`
- `PageLoading`
- `PageError`
- `PageEmpty`
- `AppModal`
- `FormSection`
- `SubmitBar`
- `AppTable` for child-resource tables
- `usePermission`

## 4. Standard State Shape

The detail page should keep explicit state for:

- current resource
- loading
- error
- action-in-flight status
- child action or modal state

## 5. `loadData`

The detail loader should:

- fetch by route `id`
- set loading and clear previous error state
- preserve failures for `PageError`
- use `finally` to close the loading phase

## 6. Rendering Skeleton: Five-Layer Structure

Recommended composition:

1. `PageHeader` with back navigation and breadcrumbs
2. hero stats card
3. summary metadata block
4. lifecycle visualization area when the resource really has a lifecycle
5. child-resource table or related-entity section where applicable

Rules:

- loading, error, and not-found states must be explicit
- child actions should stay inside shared modal or action patterns

## 7. Hero Stats Design

Hero stats should:

- summarize only the most important indicators
- stay within about six items
- use semantic tones from the base theme
- keep status values key-driven and translated

Do not turn the hero area into a dashboard wall.

## 8. State-Transition Actions

Action rendering must satisfy both:

- permission checks
- current lifecycle-state checks

Rules:

- dangerous actions must use confirmation
- in-flight actions should show loading state
- concurrent action spam should be prevented
- successful actions must refetch the detail resource

## 9. Child-Resource Tables and Child Actions

When a resource owns child rows, use shared table patterns and keep child actions explicit and auditable.

Child-resource behavior must not become a second unrelated page embedded without structure.

## 10. Metadata Blocks with `Descriptions`

Use metadata sections for stable summary facts such as:

- created time
- owner
- related entity references
- identifiers

Do not overload these sections with long operational help text.

## 11. Mobile Breakpoint Behavior

On smaller screens:

- keep summary and action areas readable
- collapse complexity safely
- preserve lifecycle and destructive-action clarity

Do not assume the desktop layout can simply shrink unchanged.

## 12. Page-State Variants

Every lifecycle detail page must explicitly support:

- loading
- error
- not found
- normal detail state
- action in flight
- child action modal state where relevant

## 13. Reference Examples

Deploy and CMDB lifecycle pages are the reference abstraction sources for this pattern, but future modules should follow the pattern rather than cloning any one page mechanically.

## 14. Acceptance

A lifecycle detail page is only complete when it has:

- explicit lifecycle-state rendering
- permission-aware action rendering
- shared page skeleton usage
- translated hero and metadata labels
- child-resource handling where needed
- state completeness and refetch behavior after transitions

## 15. Related Documents

- `BUSINESS_RESOURCE_LIST_PATTERN.md`
- `FRONTEND_PAGE_TEMPLATES.md`
- `PERMISSION_MODEL.md`
- `ACCESSIBILITY.md`
