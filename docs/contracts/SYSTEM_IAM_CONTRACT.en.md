---
title: system/iam Contract
doc_type: Contract
layer: system/iam
status: Active
updated_at: 2026-05-12
---

# system/iam Contract

Chinese version: [SYSTEM_IAM_CONTRACT.md](./SYSTEM_IAM_CONTRACT.md)

This document defines the execution contract for Pantheon’s `system/iam` capability domain.

It locks down the governance boundaries for users, roles, menus, and permissions so that navigation, pages, buttons, and API permissions do not collapse back into the old “one list permission controls everything” model.

## 1. Background

Pantheon’s `iam` is not just a collection of user-management pages. It is one of the easiest layers in the foundation to let responsibilities blur:

- user management gets mixed with authentication
- menus get mistaken for the permission model itself
- page permissions, button permissions, and API permissions get stuffed into one field
- role grants, menu trees, page guards, and Casbin policy get out of sync

Without a `system/iam` contract, the likely regressions are:

- blurred boundaries between `auth` and `iam`
- navigation, page, button, and API permissions mixed again
- new-module onboarding letting menus, permissions, routes, i18n, and seeds evolve independently
- `list` permissions turning back into the fallback for create/edit/delete actions

## 2. Ownership Layer

This contract belongs to `system/iam`.

It covers:

- user management
- role management
- menu management
- permission strategy and permission workbench
- four-layer governance across navigation, page, button, and API permissions
- onboarding constraints for menus, permissions, routes, i18n, and seeds in module registration

It does not mean:

- `system/auth` login, refresh, sessions, security center
- `system/org` departments, posts, organization governance
- `platform` shell navigation style and aggregate workbench

## 3. Goals

The `system/iam` contract exists to lock down:

1. the boundary between `iam` and `auth`
2. the rule that menus are not permissions and page permissions are not list permissions
3. the rule that users, roles, menus, and permission workbench belong to one governance domain
4. the rule that menus, permissions, routes, i18n, and seeds must be registered together during module onboarding
5. the rule that the permission workbench exists to discover, explain, export, and safely remediate governance problems
6. the definition of done and acceptance semantics for `iam`

## 4. Non-Goals

This contract does not cover:

- login, refresh, logout, sessions, or password policy
- organization rules such as departments and posts
- settings, dictionaries, i18n runtime assets, or upload configuration
- business-domain data-permission details
- business-domain menu-structure design

It also does not mistake “menu configuration” for “platform shell visual strategy”.

Navigation style belongs to `platform`; navigation metadata and authorization semantics belong to `system/iam`.

## 5. Boundaries

### 5.1 Covered Surfaces

- `/system/user`
- `/system/role`
- `/system/menu`
- `/system/permission`
- backend APIs for users, roles, menus, and permissions
- dynamic menu metadata
- semantics of page and button permissions
- API permission and Casbin policy mapping
- menu/permission/i18n/seed contracts for module onboarding

### 5.2 Excluded Surfaces

- `/login`
- `/auth/security`
- `/system/session`
- `/system/login-log`
- `/system/dept`
- `/system/post`
- `/system/setting`
- platform shell navigation style and dual-mode shell acceptance

## 6. Dependencies

This contract depends on:

- [DESIGN.md](../../DESIGN.md)
- [AGENTS.md](../../AGENTS.md)
- [BACKEND.md](../designs/BACKEND.md)
- [FRONTEND.md](../designs/FRONTEND.md)
- [PERMISSION_MODEL.md](../designs/PERMISSION_MODEL.md)
- [MODULE_CONTRACT.md](../designs/MODULE_CONTRACT.md)
- [FRONTEND_PAGE_TEMPLATES.md](../designs/FRONTEND_PAGE_TEMPLATES.md)
- [ERROR_CODE_AND_I18N.md](../designs/ERROR_CODE_AND_I18N.md)
- [ACCEPTANCE_CHECKLIST.md](../acceptances/ACCEPTANCE_CHECKLIST.md)

## 7. Hard Constraints

### 7.1 Domain-Boundary Constraints

- `auth` answers “who you are and whether you may log in”
- `iam` answers “what you can see, enter, act on, and call”
- user CRUD must not take back ownership of the auth primary path
- `system_user.preference_json.language` is a long-term user preference, not the forced runtime language for every session

### 7.1.1 User Preference vs Session Override

- user language preference participates in runtime resolution only when the current session has no explicit language override
- login-page language choice and in-system temporary language switching are `platform` session semantics, not a `system/iam` forced persistence rule

### 7.2 Four-Layer Permission Constraints

- navigation permissions, page permissions, button permissions, and API permissions must remain distinct layers
- menus are not the permission model itself
- page permission is not list-query permission
- the old pattern where one `list` permission controls create/edit/delete must not return

### 7.3 Module-Onboarding Constraints

- new modules must register menus, routes, permission points, i18n, and seeds together
- scattered manual registration by memory is not allowed
- onboarding contract takes priority over individual implementation convenience

### 7.4 Permission-Workbench Constraints

- the permission workbench is for governance, not arbitrary strategy editing
- unknown permissions, page gaps, and API gaps must be identified and explained
- controlled remediation is allowed, but only through recommendation mapping and boundary controls
- the workbench first screen must serve remediation rather than acting as a generic full-body health table
- the first screen must support browsing complete authorization results by role; it must not hide clean roles by default filtering
- role governance status is fixed to `pending | remediated | clean`
- the overview must provide at least:
  - `pendingRemediationRoleCount`
  - `remediatedRoleCount`
  - `unknownPermissionAssignmentCount`
  - `recentRemediationCount`
- role results must provide at least:
  - `governanceStatus`
  - `lastRemediationAt`
  - `lastRemediationAction`

### 7.5 Documentation Constraints

- all `system/iam` design, assessment, remediation, and acceptance docs must link back to this contract
- no new permission-model change may bypass [PERMISSION_MODEL.md](../designs/PERMISSION_MODEL.md)

### 7.6 Batch-Governance Interaction Constraints

- selection sets for user/role/permission governance lists are bound to query context rather than current page
- pagination must not clear selection; search, filter, sort, and reset actions that change query context must clear selection
- batch governance actions must execute on the full selection set; current-page selection state only mirrors visible items

## 8. Definition of Done

`system/iam` should count as currently complete only when:

### 8.1 Responsibility Completion

- ownership of users, roles, menus, and permissions is clear
- boundaries with `auth / org / config` are clear

### 8.2 Permission-Model Completion

- navigation/page/button/API permission model is stable
- page guards, button visibility, and backend authorization have consistent semantics
- coarse `list` permissions no longer act as the fallback for all actions

### 8.3 Onboarding-Contract Completion

- menus, permissions, routes, i18n, and seeds have a coordinated registration mechanism
- new-module onboarding no longer fragments into multiple disconnected sources

### 8.4 Governance-Loop Completion

- the permission workbench can discover unknown permissions, page gaps, and API gaps
- it supports export and controlled remediation
- post-remediation state distinguishes pending, remediated, and clean
- recent remediation actions can be reviewed through remediation events

### 8.5 Documentation and Acceptance Completion

- `iam` design, remediation, and acceptance docs all link back to this contract
- linkage between dynamic menus, permission model, and UI templates is stable

## 9. Acceptance Standards

Changes in `system/iam` must at least pass:

### 9.0 Batch-Delete Capability Constraints

- users, roles, and permission strategies support controlled batch delete within `system/iam`
- batch delete must use dedicated permission points: `system:user:batch-delete`, `system:role:batch-delete`, `system:permission:batch-delete`
- batch-delete endpoints must reuse single-delete service validation, preserving protections for built-in admin, built-in roles, role occupancy, and Casbin policy refresh
- batch delete is a high-risk write operation and must require second confirmation plus partial-success results: `deletedCount`, `failedCount`, `failures[]`

### 9.1 Documentation Acceptance

- [ACCEPTANCE_CHECKLIST.md](../acceptances/ACCEPTANCE_CHECKLIST.md)
- [DOCUMENT_GOVERNANCE_CONTRACT.md](../contracts/DOCUMENT_GOVERNANCE_CONTRACT.md)
- [DOCUMENT_METADATA_AND_STATUS.md](../contracts/DOCUMENT_METADATA_AND_STATUS.md)

### 9.2 Backend and Permission Acceptance

- `go test ./backend/modules/system/permission`
- user/role/menu module changes should add corresponding tests

### 9.3 Frontend and Build Acceptance

- `cd frontend && npm run build`
- `cd frontend && npm run check:menu-contract`
- if page-permission or role-authorization primary paths are touched, add smoke or targeted acceptance evidence

### 9.4 Page and Primary-Path Acceptance

- `/system/user`
- `/system/role`
- `/system/menu`
- `/system/permission`

### 9.5 Permission-Consistency Acceptance

- page guards and backend Casbin policy must not drift semantically
- menu visibility and page accessibility must not be conflated
- workbench-discovered gaps must have explicit explanations, not silent failures
- remediation summaries, role statuses, and remediation-event timelines must stay aligned

## 10. Related Documents

### 10.1 Design

- [PERMISSION_MODEL.md](../designs/PERMISSION_MODEL.md)
- [MODULE_CONTRACT.md](../designs/MODULE_CONTRACT.md)
- [FRONTEND_PAGE_TEMPLATES.md](../designs/FRONTEND_PAGE_TEMPLATES.md)

### 10.2 Acceptance

- [ACCEPTANCE_CHECKLIST.md](../acceptances/ACCEPTANCE_CHECKLIST.md)
- [PLATFORM_ACCEPTANCE_MATRIX_20260430_UI_MIGRATION.md](../acceptances/PLATFORM_ACCEPTANCE_MATRIX_20260430_UI_MIGRATION.md)
