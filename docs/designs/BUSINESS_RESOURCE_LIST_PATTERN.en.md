---
title: Business Resource List Pattern
doc_type: Design
layer: business/* (abstracted from CMDB + Deploy patterns in pantheon-ops)
status: Active
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
updated_at: 2026-05-11
---

# Business Resource List Pattern

Chinese version: [BUSINESS_RESOURCE_LIST_PATTERN.md](./BUSINESS_RESOURCE_LIST_PATTERN.md)

This document defines the standard implementation pattern for business resource list pages. All `business/<module>/<resource>` lists, such as CMDB hosts, deploy jobs, deploy packages, CMDB groups, CMDB tag standards, and future CRM or WMS resources, should follow this pattern.

It does not replace the generic `ListPage` skeleton in `FRONTEND_PAGE_TEMPLATES.md`. That document defines the shell. This document defines the business-domain specialization.

## 1. When to Use This Pattern

Use this pattern when the business resource has one or more of the following:

- list, create, update, and delete APIs
- filtering and pagination
- a status field such as enabled/disabled, online/offline, or pending/running/success
- permission-gated action buttons

Do not use it for:

- tree resources
- dashboard or board-style pages
- extremely simple config-style resources

## 2. File Structure

Recommended structure:

```text
frontend/src/modules/business/<module>/<resource>/
├── api.ts
├── <Resource>List.tsx
├── <Resource>Form.tsx
├── <Resource>Detail.tsx
└── locales/
    ├── zh-CN.json
    └── en-US.json
```

## 3. Required Platform Components

The list page should build on shared platform infrastructure such as:

- `PageContainer`
- `PageHeader`
- `FilterPanel`
- `AppTable`
- `ListHeaderActions`
- `AppModal`
- `PageLoading`
- `PageEmpty`
- `PageError`
- `usePermission`

Forbidden:

- custom replacement table components
- custom modal abstractions that bypass shared `AppModal`
- custom status-color logic outside shared theme semantics

## 4. Standard State Shape

The page should keep explicit state for:

- data rows
- total count
- loading
- error
- query object with page and page size
- filter fields such as keyword and status
- modal visibility
- current editing item
- submit-in-flight state

## 5. Permission Flags

Permission names must follow:

- `business:<module>:<resource>:<action>`

Typical flags:

- `create`
- `update`
- `delete`
- business-specific actions such as `collect`

Do not introduce `biz:*`.

## 6. Status-Color Mapping

Business status tags must use the base semantic status palette from the theme-token reference rather than arbitrary custom hex colors.

Typical mapping examples:

- pending -> neutral
- running -> info
- success -> success
- failed -> error
- maintenance -> warning
- enabled -> success
- disabled -> neutral

Status labels must still flow through i18n keys.

## 7. `loadData` Closure

The list-page data loader should:

- set `loading=true` before requests
- clear previous error state
- fetch list data
- update `items` and `total`
- preserve failures for page-level error rendering
- set `loading=false` in `finally`

Do not swallow errors. Let `PageError` render.

## 8. Rendering Skeleton

Typical composition:

- `PageContainer`
- `PageHeader`
- `FilterPanel`
- `ListHeaderActions`
- one of `PageLoading`, `PageError`, `PageEmpty`, or `AppTable`
- `AppModal` for create/edit flow

Rules:

- filtered empty state and initial empty state must be distinguished
- pagination updates must stay aligned with query state
- modal title and behavior must distinguish create and edit clearly

## 9. Table-Column Definition Constraints

Table columns should:

- use translated titles
- keep action columns fixed to the right
- preserve status rendering through semantic tags
- keep business-specific columns explicit rather than inferred from ad hoc row rendering

## 10. Create, Edit, and Delete Handlers

Handlers should:

- respect permission flags
- open shared modal flows for create and edit
- use confirmation for destructive actions
- refresh the list after successful mutation
- keep feedback routed through translated frontend messages

## 11. Page-State Variants

Every business list page must explicitly support:

- loading
- error
- initial empty
- filtered empty
- normal data state
- submitting state for modal actions

## 12. Accessibility

The pattern must follow the accessibility rules for:

- keyboard navigation
- visible focus
- table semantics
- modal focus entry and return
- button labels and icon accessibility

## 13. Reference Examples

Use CMDB and Deploy business pages as pattern references when available, but do not copy page-local drift that violates shared platform rules.

## 14. Acceptance

A business resource list page is only complete when it has:

- shared page skeleton usage
- shared table and modal usage
- permission-aware actions
- state completeness
- i18n coverage
- semantic status rendering
- acceptance against shared business and frontend design rules

## 15. Related Documents

- `FRONTEND_PAGE_TEMPLATES.md`
- `BUSINESS_LIFECYCLE_DETAIL_PATTERN.md`
- `PERMISSION_MODEL.md`
- `ACCESSIBILITY.md`
