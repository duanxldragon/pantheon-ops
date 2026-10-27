---
title: Module Contract Design
doc_type: Design
layer: platform
status: Active
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
  - docs/contracts/SYSTEM_IAM_CONTRACT.md
updated_at: 2026-04-17
---

# Module Contract Design

Chinese version: [MODULE_CONTRACT.md](./MODULE_CONTRACT.md)

This document defines the Pantheon Base module contract so that all future modules integrate through one consistent pattern instead of inventing their own.

It covers:

- how backend modules assemble
- how frontend modules register
- who owns menus, permissions, i18n, and seeds
- how business modules attach to the base without coupling to internal implementation details

## 1. Design Goals

- unify the integration path
- make responsibility boundaries explicit
- support both `system` and `business` modules
- support synchronized registration of menus, permissions, i18n, and seeds
- allow gradual evolution rather than freezing the design too early

## 2. Module Definition

A module in Pantheon Base is a functional unit that can register and evolve independently, and can own its own menus, permissions, i18n, and seed assets.

Two main categories:

- `system`: shared foundation modules such as `auth`, `user`, `role`, `menu`, `permission`, `dept`, and `post`
- `business`: business-domain modules such as `order`, `project`, and `ticket`

## 3. Core Principles of the Module Contract

### 3.1 One Module, One Boundary

Each module should center on one capability domain.

### 3.2 One Module, One Registration Set

Every module must explicitly declare:

- routes
- menus
- permissions
- i18n
- seeds

Do not scatter these across disconnected files and rely on memory.

### 3.3 Contract Before Implementation

Define what a module must expose before writing the module’s internal code.

### 3.4 Prefer Explicit Registration

At the current stage, explicit registration is preferred over auto-discovery because it is more readable and easier for AI and new engineers to follow.

## 4. Backend Module Contract

### 4.1 Backend Module Responsibilities

Each backend module owns:

- its routes
- its handlers and services
- its seed capability
- its menu and permission seeds
- its DTOs and data models

It should not:

- initialize the entire system directly
- rewrite another module’s internals
- bypass contracts and couple directly to another module’s service layer

### 4.2 Recommended Directory Structure

```text
backend/modules/{system|business}/{module}/
  module.go
  *_handler.go
  *_service.go
  *_dto.go
  *_model.go
```

### 4.3 Backend Module Assembly Interface

The recommended target contract is a single exported module definition object with at least:

```go
type BackendModule interface {
    Name() string
    RegisterRoutes(r *gin.RouterGroup)
    SeedMenus(db *gorm.DB) error
    SeedPerms(db *gorm.DB) error
    SeedI18n(db *gorm.DB) error
}
```

If needed later, extend with:

```go
Migrate(db *gorm.DB) error
```

#### 4.3.1 Minimum Requirement for the Current Stage

Even before the fully unified interface lands, modules should already be organized conceptually around:

- `RegisterRoutes`
- `SeedMenus`
- `SeedPerms`
- `SeedI18n`

### 4.4 Backend Assembly Rules

#### 4.4.1 Root Assembler Responsibilities

Root assemblers such as `main.go` or domain-level assemblers should only:

- initialize infrastructure
- assemble modules
- attach routes

They should not own:

- module business logic
- module-specific SQL
- module-specific validation rules

#### 4.4.2 Cross-Module Call Rules

Forbidden:

- direct deep imports from one module into another module’s handler, service, or repository internals
- `business/*` reading another `business/*` module’s tables or repositories directly

Allowed:

- identity lookup through `gin.Context`
- shared stable helpers from `pkg/common` and `pkg/contracts`
- explicitly defined capabilities, facades, or query contracts

If module A depends on module B, module B must expose a stable read capability with documented input, output, permission, and scope semantics.

### 4.5 Backend Seed Contract

Whenever a module owns platform metadata, it must ship synchronized seeds for:

- menus
- permissions
- i18n

#### 4.5.1 Menu Seed

Used for:

- sidebar navigation
- role-authorization trees
- breadcrumb baseline data

#### 4.5.2 Permission Seed

Used for:

- button permissions
- API permission naming
- initial role authorization

#### 4.5.3 I18n Seed

Used for:

- menu titles
- page titles
- button copy
- error-key translations

## 5. Frontend Module Contract

### 5.1 Frontend Module Responsibilities

Each frontend module owns:

- page components
- API wrappers
- route registration
- menu metadata declaration
- permission declarations
- i18n namespace declarations

### 5.2 Recommended Directory Structure

```text
frontend/src/modules/{system|business}/{module}/
  index.ts
  api.ts
  pages/
  components/
  locales/
```

Temporary transition structures are allowed, but this is the target direction.

### 5.3 Frontend Module Manifest

The long-term `ModuleConfig` should grow beyond only `name` and `routes`.

#### 5.3.1 Minimum Constraint for the Current Stage

Even now, the frontend module should at least be organized as if it owns:

- route set
- menu metadata
- permission metadata
- i18n namespace ownership

### 5.4 Frontend Route Registration Rules

Routes should register through explicit manifests and centralized assembly rather than direct layout hardcoding.

### 5.5 Page and Menu Relationships

Pages and menus are related, but not identical. A page may be reachable without being a primary left-nav entry, and menu presence must not replace explicit permission handling.

## 6. Menu Contract

### 6.1 Menu Definition

Menus define navigation structure and routing-related metadata.

### 6.2 Menu Ownership

Each module should own the menus that represent its functional boundary.

### 6.3 Menu Source Rules

Menu sources should be explicit and synchronized with module registration. Do not maintain separate hidden menu truths.

### 6.4 Component Registry Discovery Convention

The menu-contract consistency check (`frontend/scripts/check-menu-contract.mjs`) discovers component registry files by name convention instead of a hardcoded list:

- frontend: `frontend/src/core/router/*Registry.ts`
- backend: `backend/modules/system/iam/menu/*registry.go`

Downstream consumers (e.g. `pantheon-ops`) that inject overlay registries (`businessOverlayComponentRegistry.ts` / `business_overlay_component_registry.go`) are therefore picked up by the gate instead of being silently dropped. New registry files following this naming convention are discovered without editing the check script.

## 7. Permission Contract

### 7.1 Permission Sources

Permissions are sourced from module-owned metadata plus backend policy registration.

### 7.2 Permission Registration Requirements

Permission points must be registered with stable naming and kept aligned with menus, routes, and backend API policy expectations.

## 8. I18n Contract

### 8.1 I18n Namespaces

Each module should keep a stable namespace boundary for user-facing copy.

### 8.2 I18n Division of Responsibility

- modules own their domain text
- the platform owns shared shells and fallback behavior
- runtime governance owns import, export, and lifecycle management

### 8.3 I18n Requirements for New Modules

New modules must add:

- locale assets
- key naming that matches their namespace
- coverage for menus, titles, buttons, and errors

## 9. Seed and Documentation Sync Contract

Seeds and documentation must stay synchronized. If a module adds menus, permissions, or user-facing states, its design and contract materials must reflect that.

## 10. Business Module Integration Contract

Business modules must attach through the same contract discipline as system modules, not through direct shell patching or cross-module implementation shortcuts.

## 11. Current-Stage Implementation Requirements

### Backend

Backend modules should already be organized around explicit registration and seed responsibilities, even where some assembly is still manual.

### Frontend

Frontend modules should already move toward manifest-driven route, menu, permission, and i18n registration.

## 12. Gap Between This Document and Current Code

The main gap is not the direction, but the degree of enforcement. Some areas still rely on partial manual assembly or lighter manifests than the contract intends.

## 13. Acceptance Questions

Acceptance should verify:

- does the module have one clear boundary
- are route, menu, permission, and i18n definitions explicit
- are seeds synchronized
- is cross-module coupling kept behind stable contracts

## 14. Recommended Next Document

Continue with page-template, permission, and i18n companion documents to make the contract executable in day-to-day implementation.
