---
title: Permission Model Design
doc_type: Design
layer: system/iam
status: Active
linked_contracts:
  - docs/contracts/SYSTEM_IAM_CONTRACT.md
updated_at: 2026-04-29
---

# Permission Model Design

Chinese version: [PERMISSION_MODEL.md](./PERMISSION_MODEL.md)

This document defines the Pantheon Base permission model so that menu visibility, page entry, button actions, and backend API access are treated as separate concerns.

## 1. Design Goals

- make the four permission layers explicit
- keep frontend and backend responsibilities aligned
- decouple menu, page, action, and API permissions
- stay compatible with the current Casbin-based implementation
- reserve room for future data permission and multi-tenant expansion

## 2. Four-Layer Permission Model

Pantheon Base splits authorization into:

- `L1` navigation permission: whether a menu is visible
- `L2` page permission: whether a route can be entered
- `L3` action permission: whether a button or action is available
- `L4` API permission: whether a backend endpoint may be called

These layers are related, but must not be collapsed into one.

## 3. Core Principles

### 3.1 Menus Are Not Permissions

Menus define navigation information architecture. Permissions define operational authority. A menu may carry permission metadata, but the menu itself is not the permission model.

### 3.2 Page Permission Is Not List Permission

Page permission governs route entry. List permission is only one operation inside the page.

### 3.3 Button Permissions Must Be Fine-Grained

Do not let one broad key such as `system:user:list` control create, edit, delete, reset, and batch actions at the same time. Split them into explicit action keys.

### 3.4 Frontend Hiding Is Not Security

Button hiding is a user-experience rule. Real security depends on backend API permission checks.

## 4. Current Implementation State

The current platform already has:

- `system_menu`
- `system_role_menu`
- button permission metadata in `system_menu.perms`
- API policy rules in `casbin_rule`
- frontend `usePermission` handling for fine-grained visibility

The platform is effectively running a dual-track model:

- menu and button authorization for navigation and UI visibility
- Casbin API policies for backend enforcement

### 4.1 Current Governance Progress

The `system/iam` permission workbench now supports:

- integrity checks for unknown permission assignments
- coverage checks for missing page permissions or missing API strategies
- exportable governance reports
- controlled remediation for recommended single-role Casbin repair flows

In practice, it can now surface three break types:

- unknown permission drift
- page-access gaps
- API-policy gaps

## 5. Recommended Target Model

### 5.1 Permission Entities

Long-term concepts:

- `Menu`
- `PagePermission`
- `ActionPermission`
- `ApiPermission`
- `DataPermission`

### 5.2 Data Relationships

Model every role as owning separate sets of menus, page permissions, action permissions, and API permissions.

## 6. Permission Naming Rules

### 6.1 Naming Format

Canonical patterns:

- `system/auth`: `{scope}:{resource}:{action}`
- `business`: `business:{module}:{resource}:{action}`

### 6.2 Scope Rules

Use stable scope roots such as `system`, `auth`, or `business:{module}`. Do not invent ad hoc mixed prefixes.

### 6.3 Resource Rules

Resource names should map to durable domain entities or bounded pages, not to unstable UI wording.

### 6.4 Action Rules

Prefer explicit actions such as `view`, `list`, `create`, `update`, `delete`, `reset`, and `batch-update`.

#### 6.4.1 `view` vs `list`

- `view`: page entry or detail read
- `list`: list-query capability inside the page

#### 6.4.2 High-Sensitivity Governance Actions

For `system/config` and similar high-risk areas, dangerous actions must be split into dedicated keys rather than hidden under a generic `update`.

## 7. Navigation Permission Design

### 7.1 Purpose

Control whether a user can discover a page through shell navigation.

### 7.2 Rules

- navigation follows role-authorized menus
- ancestor nodes may be auto-completed to preserve tree integrity
- navigation permission does not replace route or API authorization

### 7.3 Forbidden Behaviors

- treating menu visibility as sufficient proof of access
- mixing menu grouping concerns with operation authorization

## 8. Page Permission Design

### 8.1 Purpose

Control whether the user may enter a route.

### 8.2 Recommended Goal

Every meaningful page, especially detail and governance pages, should have an explicit page permission rather than relying only on menu presence.

### 8.3 Forbidden Experience

Users must not see a menu, click it, and only then discover a late 403 caused by a missing page permission.

## 9. Action Permission Design

### 9.1 Purpose

Control button visibility and user-triggered operations.

### 9.2 Frontend Rules

- use `usePermission`
- hide by default
- optionally degrade to disabled plus tooltip for special cases

### 9.3 Backend Rules

Frontend action permission never replaces backend authorization. Sensitive actions still require API policy checks and, where needed, secondary confirmation.

### 9.4 High-Sensitivity `system/config` Capability Splits

Dynamic modules, generators, settings writes, cache refreshes, and similar governance operations should use dedicated permission keys and not share one flat admin toggle.

## 10. API Permission Design

### 10.1 Current Implementation

Backend route authorization is enforced through Casbin rules.

### 10.1.1 Current Workbench Governance View

The workbench can now detect missing API coverage relative to known page and action permissions and can guide controlled repair.

### 10.2 Target Rules

- every protected route must map to a clear role policy
- policy generation should be predictable and auditable
- recommended mappings should be machine-checkable

### 10.3 Self-Service Whitelist

Logged-in users must retain access to self-service APIs such as logout, profile, security center, current sessions, and password change without depending on menu authorization.

## 11. Role Authorization Design

### 11.1 Current Problems

- mixed concepts on one authorization surface
- unclear naming
- hard-to-explain breaks between UI and backend behavior

### 11.2 Target Experience

Role editing should clearly separate:

- navigation authorization
- page authorization
- action authorization
- API strategy coverage

### 11.3 Short-Term Strategy

Keep the current storage model if needed, but present and reason about it through the four-layer model.

## 12. Admin Role Rules

`admin` remains the default full-access role, but the system should still keep its permissions explicit enough to support governance, export, inspection, and future tightening.

## 13. Permissions and Menu Node Types

Menu node types and permission semantics must stay aligned, but page and action permissions must not collapse back into tree-node shape alone.

## 14. Permission Seed Rules

Permission seeds must be deterministic, domain-aware, and versionable. New modules should register menus, page permissions, action permissions, and API policies together.

## 15. Frontend Permission Consumption Rules

- use one shared permission hook
- keep route guards and button visibility on the same source of truth
- make 403 states explicit
- avoid hardcoded permission checks scattered through pages

## 16. Backend Permission Consumption Rules

- enforce APIs through Casbin
- keep route registration and permission seed logic connected
- avoid silent privilege inheritance

## 17. Data Permission Reservation

The minimum data-permission baseline is already in place, including middleware injection, role data-scope policy, GORM Scope, and business samples. Coverage is still expanding.

Future evolution should continue to leave room for:

- self-owned data
- department-owned data
- department plus descendants
- designated departments
- custom scope
- tenant dimension

## 18. Acceptance Checklist

Acceptance must verify:

- menu visibility
- page entry
- action visibility and availability
- backend API enforcement
- consistency between role authorization screens and live behavior

### 18.1 Extra Acceptance for `system/config -> setting`

Verify the dual-track requirement: frontend page access and backend route policies must both be complete.

### 18.2 Extra Acceptance for `system/config -> dynamicmodule / generator`

Verify dedicated high-sensitivity permissions, controlled remediation, and the absence of overly broad shared admin toggles.

## 19. Current Delivery Gaps

The remaining gap is mostly governance clarity: naming consistency, workbench guidance, synchronized frontend-backend understanding, and broader data-scope coverage across more modules and tenant-aware scenarios.

## 20. Recommended Next Document

Continue with the related detailed design and remediation documents referenced by the `system/iam` contract.
