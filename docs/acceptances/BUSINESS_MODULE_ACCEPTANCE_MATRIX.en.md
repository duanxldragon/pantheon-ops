---
title: business/* Acceptance Matrix
doc_type: Acceptance
layer: business/*
status: Active
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
updated_at: 2026-05-19
---

# business/* Acceptance Matrix

Chinese version: [BUSINESS_MODULE_ACCEPTANCE_MATRIX.md](./BUSINESS_MODULE_ACCEPTANCE_MATRIX.md)

This matrix turns business-module acceptance into a fixed governance surface for all `business/*` modules in downstream repositories.

## Nine Required Dimensions

Every business module should answer with evidence across:

- boundary
- data model
- APIs
- pages and page states
- menus and component keys
- permissions
- i18n
- audit
- regression coverage

## Implementation Entry Gate

Before implementation starts, a business module should already have:

- a business design document
- data model and tenant-readiness judgment
- API and permission list
- menu and component-key list
- i18n namespace and key inventory
- dictionary and config dependencies
- audit-point list

## Release Gate

Before a business module is marked complete, it should pass:

- backend module tests
- frontend build
- menu-contract check
- i18n hardcode check
- at least one mainline smoke path
- both authorized and unauthorized permission scenarios

Use the Chinese source document for the full matrix and fixed command set.
