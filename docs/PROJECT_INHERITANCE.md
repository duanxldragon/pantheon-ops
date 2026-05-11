# Project Inheritance

## 1. Inheritance Source

- Base repository: `../pantheon-base`
- Base branch: `main`
- Base version: `d119872` (`d1198723d85c1cfc7e71a4144518560e81afdb06`)
- Inheritance mode: `foundation-only`

## 2. Inherited Base Rules

This repository inherits from `pantheon-base`:

- layer model: `platform / system/auth / system/iam / system/org / system/config / business/*`
- contract-first document flow
- shared backend, frontend, permission, i18n, audit, and acceptance rules
- shared shell and system-domain UI constraints

## 3. Required Base Reading Order

Before editing this repository, read:

1. `../../docs/WORKSPACE_INHERITANCE.md`
2. `../pantheon-base/DESIGN.md`
3. `../pantheon-base/AGENTS.md`
4. `../pantheon-base/docs/README.md`
5. matching base contracts, designs, and acceptance docs

## 4. Local Business Scope

- `business/cmdb`
- `business/deploy`

## 5. Local Additions Only

- `docs/designs/BUSINESS_CMDB_MODULE_DESIGN.md`
- `docs/designs/BUSINESS_DEPLOY_MODULE_DESIGN.md`

Derived repository changes may add more `business/*` documents over time, but should not duplicate base platform or system-domain contracts.

## 6. Override Policy

- Allowed: business-domain additions, business acceptance notes, and repository-local execution details
- Not allowed: redefining base contracts, base layer ownership, key-first i18n rules, menu/permission split, or shared UI hard constraints
- If a foundation rule must change, update `pantheon-base` first and then upgrade `pantheon-ops`
