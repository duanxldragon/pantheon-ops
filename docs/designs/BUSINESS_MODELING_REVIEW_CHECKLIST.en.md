---
title: Business Module Modeling Review Checklist
doc_type: Design
layer: business/*
status: Active
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
updated_at: 2026-05-04
---

# Business Module Modeling Review Checklist

Chinese version: [BUSINESS_MODELING_REVIEW_CHECKLIST.md](./BUSINESS_MODELING_REVIEW_CHECKLIST.md)

This document is the unified review checklist to use before a business module enters DDL, API, and page implementation.

Its goal is straightforward:

Keep future `business/*` modules moving in a single-tenant runtime today without blocking future tenant-readiness, data isolation, audit governance, or platform integration paths.

This is not an acceptance checklist. It is a pre-modeling review checklist.

## 1. Scope

Use this checklist whenever Pantheon:

- adds `backend/modules/business/*`
- adds `frontend/src/modules/business/*`
- adds or changes `biz_*` tables
- adds module menus, permissions, dictionaries, configuration, import, or export flows
- adds business list, detail, statistic, or state-transition APIs

## 2. Module Boundary Review

Before anything else, answer:

1. which business domain owns the requirement
2. why it belongs to `business/*` instead of a system domain
3. which base capabilities it depends on
4. whether it is accidentally reintroducing base responsibilities inside the business module

Must confirm:

- business modules do not directly import base internal services or repositories
- business modules integrate through `gin.Context`, `pkg/common`, `pkg/contracts`, shared hooks, and explicit contracts
- auth, user, role, org, and config models are not reimplemented inside the module
- `business/*` modules do not directly read each other’s tables or service internals
- cross-business consumption must go through an owning module’s explicit query contract, facade, or capability

If the boundary is unclear, the module should not enter schema design yet.

## 3. Business Object Review

Clarify:

- primary entity
- detail entity
- extension entity
- aggregate root
- stable core fields
- presentation-only or volatile fields

Avoid overloading the primary table with unstable future fields or using different names for the same concept across layers.

## 4. Table-Structure Review

For every `biz_*` table, check:

- correct prefix
- primary-key strategy
- whether soft delete is needed
- whether audit fields are needed
- whether detail, extension, snapshot, history, or log tables are needed
- whether indexes map to real query paths

Also judge:

- which fields are real search keys
- which fields will later affect filtering, sorting, or export
- which fields belong in extension JSON rather than fixed columns

## 5. Tenant-Readiness Review

This is the most important section.

Every module must answer:

1. is future tenant isolation likely
2. if yes, should version one already include `tenant_id`
3. if not, why not
4. is uniqueness global now or likely tenant-local later
5. will list, export, and statistic interfaces later need unified tenant filtering
6. will audit later need tenant-based retrieval

Suggested judgments:

- clearly platform-global objects may omit `tenant_id`, but the reason must be documented
- likely tenantized business objects should usually add `tenant_id` early
- uncertain cases should be marked tenant-ready pending and must not hardcode global uniqueness and query assumptions casually

## 6. Uniqueness Review

For each unique constraint, decide whether it is:

- platform-global unique
- tenant-local unique
- parent-scope unique
- state-scope unique

High-risk examples:

- order number
- customer code
- project code
- contract number
- ticket number

Review must state:

- why uniqueness is needed
- in what scope it is unique
- whether future scope changes can migrate smoothly

## 7. Query and Statistics Review

All multi-row read interfaces must check:

- whether list queries can reuse unified query DTO patterns
- whether export aligns with list filters
- whether statistics cross future business-space boundaries
- whether unified scope or context injection is needed

Rules:

- do not scatter filtering logic everywhere
- do not abandon unified injection points just because today there is only one space
- prefer existing `DataScopeReq + WithDataScope` extension paths

## 8. Permission and Menu Review

At minimum, separate:

- navigation permission
- page access permission
- action permission
- API permission

Must verify:

- menus only express navigation
- `titleKey` coverage is complete
- page and button permissions are separated
- the module is not lazily collapsing everything into one `list` permission

## 9. Dictionary and Configuration Review

Judge:

- whether an enum belongs to a business dictionary or a shared system dictionary
- whether a value set should be modeled as a dictionary rather than hardcoded
- whether configuration is platform-global or business-local
- whether future tenant overrides are plausible

Do not bury business states in scattered if-else blocks and constants.

## 10. Audit Review

For each of these actions, decide whether audit is required:

- create
- update
- delete
- state transition
- import
- export
- batch operation
- rollback, revoke, or void

Must clarify:

- which actions are high sensitivity
- which need second confirmation
- which need before-after snapshots
- whether future tenant dimensions are relevant

## 11. Import and Export Review

If the module may need batch data handling, decide early:

- whether import is needed
- whether export is needed
- whether template fields match the real model
- whether import errors need structured return payloads
- whether export is constrained by permissions and future data-isolation scope

## 12. Frontend Page Review

At minimum, identify whether the module has:

- list pages
- detail pages
- create or edit pages
- action dialogs for state changes
- configuration pages
- approval pages

For each page, clarify:

- target user
- main operation frequency
- page states
- consistency with backend model
- whether the UI should leave room for future tenant, org, or data-scope entry points

## 13. Required Conclusions

This review is only useful if it ends in explicit conclusions rather than vague “looks okay” approval.

The reviewer should be able to say clearly:

- whether the boundary is acceptable
- whether schema and uniqueness are acceptable
- whether tenant-readiness is acceptable
- whether permission, menu, audit, and UI planning are acceptable
- what must be fixed before implementation

## 14. Recommended Usage

Use this checklist together with:

- the business-module design template
- tenant-ready design rules
- permission model
- page-template rules
- module contracts

It should be part of the standard review gate before new business modules move from idea to schema and code.
