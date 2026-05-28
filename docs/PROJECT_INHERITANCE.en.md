# Project Inheritance

Chinese version: [PROJECT_INHERITANCE.md](./PROJECT_INHERITANCE.md)

## 1. Inheritance Source

- Base repository: `../pantheon-base`
- Base branch: `main`
- Base version: `0b06ee4` (`0b06ee40ae2a281bf2a0004343368599a326bc67`)
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
- `business/bizscope`

## 5. Local Additions Only

- `docs/designs/BUSINESS_CMDB_MODULE_DESIGN.md`
- `docs/designs/BUSINESS_DEPLOY_MODULE_DESIGN.md`

Derived repository changes may add more `business/*` documents over time, but should not duplicate base platform or system-domain contracts.

## 6. Override Policy

- Allowed: business-domain additions, business acceptance notes, and repository-local execution details
- Not allowed: redefining base contracts, base layer ownership, key-first i18n rules, menu/permission split, or shared UI hard constraints
- If a foundation rule must change, update `pantheon-base` first and then upgrade `pantheon-ops`

Shared foundation changes such as upload behavior, pagination behavior, shared admin-shell layout, and common table patterns must follow a base-first flow:

1. update `pantheon-base`
2. validate in `pantheon-base`
3. sync shared paths into `pantheon-ops`
4. re-validate local business modules in `pantheon-ops`

## 7. Runtime Isolation

- Runtime database is isolated from `pantheon-base`.
- Recommended default DSN target for this repository is `pantheon_ops`.
- Sharing a MySQL instance is allowed; sharing the same database schema with `pantheon-base` is not allowed.
