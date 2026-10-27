---
title: business/* Acceptance Matrix
doc_type: Acceptance
layer: business/*
status: Active
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
updated_at: 2026-09-10
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

### Tenant-Readiness Four Mandatory Questions (added 2026-09-10, paired with DDL review)

Per the tenant-ready single-tenant design and the [tenant resource scope matrix](../designs/TENANT_RESOURCE_SCOPE_MATRIX.md),
every new business module / DDL review must explicitly answer four questions before passing,
and record the conclusion in the design document:

1. **tenant field** — could this table become tenant-scoped? If likely, does the first version include `tenant_id` (or record an explicit "will not be tenant-scoped" rationale)?
2. **tenant-local uniqueness** — is each unique key platform-global or tenant-local? Business document numbers default to tenant-local evaluation.
3. **query/export/aggregate filtering** — do list, detail, batch, export and aggregate endpoints all go through the unified scope injection point (`DataScopeReq + WithDataScope`) so tenant filtering can be layered later?
4. **audit dimension** — will audit records need tenant-based retrieval? Do business write audits keep a splittable dimension?

> The runtime stays single-tenant; the generator's `tenant` data-scope option has been retired
> (runtime truth source: `backend/pkg/common/data_scope.go`). These four questions are review
> requirements, not implementation requirements.

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
