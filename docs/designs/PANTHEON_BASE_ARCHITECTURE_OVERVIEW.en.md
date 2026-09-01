---
title: Pantheon Base Architecture Overview
doc_type: Design
layer: platform / system/auth / system/iam / system/org / system/config / business/*
status: Active
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
  - docs/contracts/SYSTEM_AUTH_CONTRACT.md
  - docs/contracts/SYSTEM_IAM_CONTRACT.md
  - docs/contracts/SYSTEM_ORG_CONTRACT.md
  - docs/contracts/SYSTEM_CONFIG_CONTRACT.md
updated_at: 2026-05-12
---

# Pantheon Base Architecture Overview

Chinese version: [PANTHEON_BASE_ARCHITECTURE_OVERVIEW.md](./PANTHEON_BASE_ARCHITECTURE_OVERVIEW.md)

This document is the orientation map for engineers who are new to Pantheon Base. It does not replace the detailed design documents. It explains what the system is, how the layers fit together, how runtime assembly works, what is already stable, and what still needs to mature.

## 1. One-Sentence Summary

Pantheon Base is not just a login page, menu system, and CRUD starter. It is a modular-monolith foundation for enterprise backoffice products.

It keeps shared capabilities in `platform + system/*`, then connects `business/*` through explicit module contracts so that:

1. shared platform capabilities can evolve steadily
2. business modules do not pollute the foundation
3. menus, permissions, i18n, audit, and configuration can be governed consistently
4. AI tools and new engineers can understand and extend the project quickly

## 2. Build the Right Mental Model First

### 2.1 It Is a Workspace, Not Just a Repo

`pantheon-platform` is the workspace. `pantheon-base` is the authoritative source for the base architecture and contracts. Derived repos such as `pantheon-ops` should carry their own `business/*` logic.

The intended model is:

- `pantheon-base` defines the foundation
- derived repos inherit the foundation
- business-specific capabilities stay in business repos whenever possible

### 2.2 It Is a Modular Monolith, Not Microservices

The backend starts from one server entry and wires together:

- Gin
- middleware
- MySQL / Redis / Casbin
- `platform`
- `auth`
- `system`
- `business`

The current strategy is to clean up module boundaries first and reduce coupling through contracts and governance, rather than splitting prematurely into microservices.

## 3. How to Read the Layers

### 3.1 `platform`: Shell and Aggregation

`platform` owns the parts that make the backoffice feel like one coherent system:

- application shell
- routing assembly
- page skeletons
- dashboard and workbench
- platform-level state feedback
- cross-domain aggregate views

It may aggregate data across domains, but it must not take over the internal responsibilities of any one domain.

### 3.2 `system/*`: Shared Foundation Capabilities

Pantheon explicitly separates the system layer instead of keeping one large `system` junk drawer.

#### `system/auth`

Owns identity verification, login, session validity, MFA, login logs, and the security center.

#### `system/iam`

Owns users, roles, menus, page permissions, action permissions, API permissions, and the permission workbench.

#### `system/org`

Owns organization structure: departments, posts, organization tree governance, and user-organization structural dependencies.

#### `system/config`

Owns shared configurable capabilities such as dictionaries, system settings, i18n governance assets, upload and storage configuration, dynamic module governance, generators, and selected high-sensitivity actions.

### 3.3 `business/*`: Business-Domain Capabilities

`business/*` is the extension area for real product domains. Business modules should not directly couple to internal base implementation details. They should connect through:

- backend `BackendModule`
- frontend `ModuleConfig`
- menu seeds
- permission seeds
- i18n seeds
- page and component registration

## 4. How Runtime Assembly Works

### 4.1 Backend Main Path

The backend entry:

1. initializes shared runtime concerns such as security config and time zone
2. initializes MySQL, Redis, and Casbin
3. mounts middleware such as CORS, request context, operation logging, and CSRF
4. registers `platform`, `dashboard`, `system`, `auth`, and `business`

The system-domain assembler collects migrations, menu seeds, permission seeds, i18n seeds, and route registration into one controlled path.

Optional multi-instance deployment switches are available without changing the single-instance default behavior:

- When `PANTHEON_CASBIN_WATCHER=true` and Redis is available, policy writes fan out through a Redis watcher and trigger `LoadPolicy()` on peer instances.
- When `PANTHEON_TOKEN_CACHE_TTL_SECONDS=0`, the in-process token cache is disabled and session / permission checks go directly to Redis.
- If both switches stay off, the default single-instance behavior remains unchanged.

### 4.2 Frontend Main Path

The frontend runtime is organized around:

1. `App.tsx`
2. `frontend/src/core/*`
3. `frontend/src/modules/*`

The shell, router, permissions, i18n, menus, and modules are assembled through explicit contracts rather than page-local hacks.

### 4.3 Module Integration Path

A real module enters the system by declaring itself clearly on both sides:

- backend route and seed registration
- frontend route and menu registration
- aligned permission and i18n resources

## 5. What Is Already in Place

### 5.1 Stable Base Skeleton

Pantheon Base already has a stable modular-monolith baseline, shell structure, system-domain separation, and shared contracts for modules and documents.

### 5.2 Closed Core System Flows

The main auth, IAM, organization, configuration, dashboard, settings, dictionary, and audit paths already have meaningful end-to-end closure.

### 5.3 Areas That Have Entered Governance Mode

Some parts are no longer just feature pages. They are governance capabilities, such as:

- permission workbench
- settings audit and cache refresh
- organization governance summaries
- dynamic module governance

### 5.4 Early Business Integration Validation

The base has already begun validating how business modules should enter the system through contract-based integration rather than direct shell mutation.

## 6. Biggest Current Risks

### 6.1 `system/config` Can Still Keep Expanding

It is the easiest place for unrelated governance and platform capabilities to pile up if boundaries are not enforced.

### 6.2 `system/iam` Is Not Fully at Remediation Closure Yet

Permission governance has advanced, but the full cleanup and explanation loop still needs strengthening.

### 6.3 `business/*` Still Needs More Real Samples

The architecture is ready for business modules, but more high-quality real examples are still needed to keep future work from drifting back into shell-coupled shortcuts.

## 7. What to Strengthen Next

### 7.1 Near-Term Priorities

First stabilize Harness Engineering closure and then continue filling standard backoffice capabilities.

#### Phase 1: Harness Engineering Closure

Keep the documentation, governance, verification, and inheritance story consistent and enforceable.

#### Phase 2: Standard Backoffice Completion

Continue expanding the stable enterprise-admin baseline without dissolving domain boundaries.

### 7.2 P2 Evolution Direction

P2 should deepen platform governance, stronger auth scale-up patterns, and more mature business integration.

### 7.3 Four Second-Phase Tracks

#### `auth-scale`

Push stronger authentication, session, MFA, and future SSO or risk-control evolution.

#### `platform-ops`

Strengthen operational governance, shell consistency, and platform evidence workflows.

#### `governance-core`

Strengthen contract-driven governance, permission repair loops, configuration safety, and auditability.

#### `enterprise-backoffice`

Continue hardening the shared admin UX, page templates, permission surfaces, and reusable patterns.

### 7.4 Areas Not Yet Truly Implemented

Some topics are still future-facing and should not be treated as already productized just because their architecture has been outlined.

## 8. Recommended Reading Order

Read this overview first, then move into:

1. the relevant contract for your target layer
2. the detailed design document for that domain
3. the linked remediation, assessment, and acceptance materials when they exist
