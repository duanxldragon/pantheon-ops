---
title: Business Module Design Template
doc_type: Design
layer: platform
status: Active
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
updated_at: 2026-04-17
---

# Business Module Design Template

Chinese version: [BUSINESS_MODULE_TEMPLATE.md](./BUSINESS_MODULE_TEMPLATE.md)

This document is not the implementation note for one specific business module. It is the standard template to use whenever Pantheon adds a new `business/*` module.

It answers:

- how a business-module design document should be written
- how business modules integrate into the platform without damaging the base boundary
- what minimum design work is required before a business module is considered shippable, maintainable, and extensible

Without this template, modules quickly diverge in design depth, permission modeling, menus, i18n, and integration assumptions.

## 1. Scope

This template applies to:

- `backend/modules/business/*`
- `frontend/src/modules/business/*`
- business-module DDL in `database/*`
- menus, permissions, dictionaries, configuration, and audit design related to business modules

Related review checklist:

- `BUSINESS_MODELING_REVIEW_CHECKLIST.md`

## 2. Standard Design-Document Sections

Each business-module design should contain at least the following.

### 2.1 Module Overview

Must state:

- Chinese and English module name
- owning business domain
- module goal
- what problem the module solves
- what it explicitly does not own

### 2.2 Boundary and Dependencies

Must confirm:

- the module belongs to `business/*`, not `system/*`
- which platform capabilities it depends on
- which system-domain capabilities it depends on
- which direct implementation couplings are forbidden

Recommended dependency matrix:

- platform-level allowed dependencies
- system-domain allowed dependencies
- business-domain explicit capabilities only

### 2.3 Business Objects and Vocabulary

List core business objects and stable terms:

- primary entities
- child entities
- aggregate roots
- state-machine nodes
- numbering rules

Avoid multiple names for the same concept across frontend, backend, and documentation.

### 2.4 Business Flow

At minimum cover:

- happy-path flow
- exception flow
- state transitions
- approval, confirmation, cancellation, or revoke actions

The document must clearly express who initiates, who approves, who may revoke, and which states are editable versus read-only.

### 2.5 Data Model Design

Must describe:

- primary tables
- detail tables
- extension tables
- relationships
- unique constraints
- indexes
- soft-delete, audit, and optional tenant fields

It must also include an explicit tenant-readiness judgment:

- is the module clearly single-space or likely to become tenant-scoped later
- if tenantization is likely, should version one already carry `tenant_id`
- is uniqueness platform-global or likely tenant-local later
- will list, export, and statistic interfaces need unified tenant filtering later

### 2.6 API Design

Must list:

- module API prefix
- key endpoints
- request parameters
- response shape
- error keys
- permission points

Also state:

- which endpoints are list-oriented
- which are state-transition actions
- which must enter audit
- which require idempotency
- which external module capability or facade is consumed, if any

### 2.7 Permission Model

Business modules may not collapse everything into one `list` permission.

At minimum distinguish:

- navigation permission
- page access permission
- action permission
- API permission

Prefer canonical names of the form:

- `business:{module}:{resource}:{action}`

Do not introduce `biz:*`.

### 2.8 Menu and Navigation Design

Must describe:

- top-level navigation ownership
- menu hierarchy
- route path
- `titleKey`
- `routeName`
- `module`
- component key
- page entry points
- whether detail pages appear in navigation

Menus express navigation only. They must not carry business logic.

### 2.9 Frontend Page Design

Must state which page types exist:

- list page
- detail page
- create or edit page
- approval page
- configuration page
- relation dialog or drawer

For each page, specify:

- page goal
- skeleton type
- filter fields
- table columns
- action buttons
- form fields
- required states

Also link back to the shared UI and page-template rules.

### 2.10 Multilingual Design

Each module must declare its i18n namespace and key prefix strategy.

### 2.11 Dictionary and Configuration Dependencies

State which enums belong to shared dictionaries, which settings are reused, and whether any future tenant override scope may be relevant.

### 2.12 Audit and Security Requirements

State:

- which actions enter audit
- which are high-sensitivity
- which need second confirmation
- whether snapshot-before and snapshot-after evidence is required

### 2.13 Seed and Initialization Requirements

State which menus, permissions, i18n assets, dictionaries, or sample records must be seeded.

### 2.14 Test and Acceptance

State the required interface, permission, UI, audit, import-export, and domain-flow verification needed before acceptance.

## 3. Standard Section Template

The template should be reusable as the scaffold for a concrete module design document, from module overview through risk, out-of-scope items, and acceptance.

## 4. Backend Delivery Template

Backend work should follow the vertical-slice module pattern and shared backend module-contract expectations.

## 5. Frontend Delivery Template

Frontend work should follow module manifests, page templates, shared components, permission rules, and i18n-first text handling.

## 6. DDL Template Requirements

- table names must use `biz_`
- field naming should be explicit
- enum origin must be documented
- history, log, or snapshot tables must be justified when needed

## 7. Module Integration Checklist

Before integration, confirm routes, menus, permissions, i18n, seeds, DDL, and acceptance expectations are all explicit.

## 8. Boundary with Other Documents

This template is the mother pattern for business-module design documents. It works together with the business-model review checklist, tenant-ready rules, permission model, page-template rules, and module contracts.

## 9. Definition of Done

A business module is not design-complete until its domain boundary, data model, APIs, permissions, menus, pages, i18n, audit expectations, seeds, and acceptance path are all explicit enough to implement without guesswork.
