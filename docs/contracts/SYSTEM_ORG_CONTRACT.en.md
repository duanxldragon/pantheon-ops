---
title: system/org Contract
doc_type: Contract
layer: system/org
status: Active
updated_at: 2026-04-30
---

# system/org Contract

Chinese version: [SYSTEM_ORG_CONTRACT.md](./SYSTEM_ORG_CONTRACT.md)

This document defines the execution contract for Pantheon’s `system/org` capability domain.

It locks down the ownership boundaries of departments, posts, organization trees, and organization-governance context so that organization-structure governance does not drift back into `iam` user authorization or `config` page logic.

## 1. Background

In many backoffice systems, `system/org` gets reduced to “two CRUD pages: departments and posts”.

In Pantheon, its more accurate responsibility is:

> organization structure and organization ownership governance, not identity authentication, role authorization, or configuration-asset management.

Without a `system/org` contract, the likely regressions are:

- department and post logic pushed back into the user-management page
- governance summaries for organization and authorization stacked redundantly
- blurred ownership and accountability across user affiliation, department trees, and post data
- `org` pages reusing inappropriate `iam` governance semantics just for visual alignment

## 2. Ownership Layer

This contract belongs to `system/org`.

It covers:

- department management
- post management
- organization tree
- structural governance that user organization membership depends on
- organization-governance context and issue-location entry points

It does not mean:

- `system/auth` login, sessions, security center
- `system/iam` roles, menus, permission strategy
- `system/config` dictionaries, settings, i18n, upload, dynamic modules, generators
- `platform` shell navigation and workbench aggregation

## 3. Goals

The `system/org` contract exists to lock down:

1. the boundary between organization-structure governance and identity/authorization governance
2. the rule that departments, posts, and organization trees belong to `org`, not as side fields under `iam`
3. the rule that right-side governance information on `org` pages must support organizational diagnosis only and must not duplicate `iam` summaries
4. the definition of done and acceptance semantics for organization governance
5. the rule that future organization-governance extensions must not invade `auth / iam / config`

## 4. Non-Goals

This contract does not cover:

- login, refresh, sessions, security policy
- role authorization, menu configuration, permission workbench
- dictionaries, settings, i18n, upload, and other configuration-asset governance
- business-domain-specific organization semantics

It also must not treat “which department/post a user belongs to” as an `auth` or `iam` sub-responsibility.

## 5. Boundaries

### 5.1 Covered Surfaces

- `/system/dept`
- `/system/post`
- department tree, post lists, organization-affiliation summary
- organization issue counts and governance entry points

### 5.2 Excluded Surfaces

- `/system/user`
- `/system/role`
- `/system/menu`
- `/system/permission`
- `/login`
- `/auth/security`
- `/system/setting`
- platform shell navigation and aggregate workbench

## 6. Dependencies

This contract depends on:

- [DESIGN.md](../../DESIGN.md)
- [AGENTS.md](../../AGENTS.md)
- [BACKEND.md](../designs/BACKEND.md)
- [FRONTEND.md](../designs/FRONTEND.md)
- [FRONTEND_UI_SPEC.md](../designs/FRONTEND_UI_SPEC.md)
- [FRONTEND_PAGE_TEMPLATES.md](../designs/FRONTEND_PAGE_TEMPLATES.md)
- [ACCEPTANCE_CHECKLIST.md](../acceptances/ACCEPTANCE_CHECKLIST.md)

## 7. Hard Constraints

### 7.1 Domain-Boundary Constraints

- departments, posts, and organization trees belong to `system/org`
- user affiliation may be consumed by `iam`, but organization-structure definition must not be pushed back into `iam`
- `org` must not take ownership of authentication primary paths or configuration-asset governance

### 7.2 Page and Governance Constraints

- the main column of `org` pages carries real organization-task flow
- the right-side assist rail may contain only organization positioning, governance issue counts, and next-step actions
- existing `iam` authorization summaries must not be duplicated

### 7.3 Data and Semantic Constraints

- organization tree, posts, and user affiliation must all be explained through organization-governance semantics
- posts and departments must not be treated like temporary dictionary items
- organization-governance expansion must preserve structural clarity and must not invade `dict` or `setting`

### 7.4 Documentation Constraints

- all `system/org` design, assessment, remediation, and acceptance docs must link back to this contract
- if future topic docs are added, they must declare whether they still belong to `org` or have become cross-layer topics

## 8. Definition of Done

`system/org` should count as currently complete only when:

### 8.1 Responsibility Completion

- departments, posts, and organization governance have clear boundaries
- boundaries with `iam / auth / config` are clear

### 8.2 Page Completion

- department and post primary paths are stable
- organization-governance information is no longer mixed with authorization-governance information
- right-side assist rails stay within lightweight organization-governance semantics

### 8.3 Data Completion

- organization trees and post structures have stable governance semantics
- user affiliation depends on organization structure without polluting organization-governance ownership

### 8.4 Documentation and Acceptance Completion

- main `org` design, remediation, and acceptance docs all link back to this contract
- acceptance clearly distinguishes organization governance from authorization governance

## 9. Acceptance Standards

Changes in `system/org` must at least pass:

### 9.0 Batch-Delete Capability Constraints

- departments and posts support controlled batch delete under `system/org` and must not be pushed into `iam` or `config`
- batch delete must use dedicated permission points: `system:dept:batch-delete`, `system:post:batch-delete`
- batch-delete endpoints must reuse single-delete validations, preserving protections for root department, child departments, post occupancy, and user occupancy
- batch delete is a high-risk write operation and must require second confirmation plus partial-success results: `deletedCount`, `failedCount`, `failures[]`

### 9.1 Documentation Acceptance

- [ACCEPTANCE_CHECKLIST.md](../acceptances/ACCEPTANCE_CHECKLIST.md)
- [DOCUMENT_GOVERNANCE_CONTRACT.md](../contracts/DOCUMENT_GOVERNANCE_CONTRACT.md)
- [DOCUMENT_METADATA_AND_STATUS.md](../contracts/DOCUMENT_METADATA_AND_STATUS.md)

### 9.2 Frontend and Build Acceptance

- `cd frontend && npm run build`
- if the organization-governance primary path changes, add page-level smoke or acceptance evidence

### 9.3 Page and Primary-Path Acceptance

- `/system/dept`
- `/system/post`

### 9.4 Organization-Governance Acceptance

- right-side organization info must not degrade into a second main content column
- organization-governance information must help locate and process issues, not become redundant summary stacking

## 10. Related Documents

### 10.1 Design

- [BACKEND.md](../designs/BACKEND.md)
- [FRONTEND_PAGE_TEMPLATES.md](../designs/FRONTEND_PAGE_TEMPLATES.md)
- [FRONTEND_UI_SPEC.md](../designs/FRONTEND_UI_SPEC.md)

### 10.2 Acceptance

- [ACCEPTANCE_CHECKLIST.md](../acceptances/ACCEPTANCE_CHECKLIST.md)
- [PLATFORM_ACCEPTANCE_MATRIX_20260430_UI_MIGRATION.md](../acceptances/PLATFORM_ACCEPTANCE_MATRIX_20260430_UI_MIGRATION.md)
