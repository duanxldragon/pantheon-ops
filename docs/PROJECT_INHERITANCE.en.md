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

## 6.1 Landing-Side Decision Rules

When a problem appears, decide the landing side before editing:

- fix it in `pantheon-base` first if it belongs to `platform`, any `system/*` domain, shared admin-shell behavior, shared pagination, shared tables, shared upload, shared i18n, or shared smoke helpers
- keep it local to `pantheon-ops` if it belongs to `business/cmdb`, `business/deploy`, or `business/bizscope`
- if the visible symptom is in ops but the root cause is a shared shell or shared component, route the fix back to base
- if unsure, read the base contracts and this inheritance file first instead of guessing from file paths alone

## 6.2 Sync Closure Checklist

A `base -> ops` sync should at least make these answers explicit:

- which base commit introduced the shared change
- which shared paths were synced and which were intentionally left out
- whether local `business/*` paths remained untouched
- whether menus, permissions, i18n, tests, smoke, and docs were aligned with the shared change
- whether base and ops each received their minimum validation pass
- whether remaining drift was recorded explicitly instead of left implicit

## 6.3 Executable Sync Command List

Recommended order for one `base -> ops` sync pass:

1. finish the shared foundation change in `pantheon-base` and record the base commit

```powershell
git -C D:\workspace\go\pantheon-platform\pantheon-base rev-parse --short HEAD
```

2. run the local inheritance guard in `pantheon-ops`

```powershell
Set-Location D:\workspace\go\pantheon-platform\pantheon-ops
npm run check:inheritance-contract
```

3. verify shared backend alignment against base

```powershell
npm run check:base-sync:backend
```

4. if shared backend files must be synced, sync them file-by-file and do not overwrite `business/*`

```powershell
git diff --name-only -- D:\workspace\go\pantheon-platform\pantheon-base\backend
```

5. run the minimum validation in both repositories

```powershell
Set-Location D:\workspace\go\pantheon-platform\pantheon-base
go test ./...

Set-Location D:\workspace\go\pantheon-platform\pantheon-ops
go test ./...
npm run check:base-sync:backend
```

6. if the turn also touched shared frontend shell behavior, pagination, shared tables, or shared i18n, add minimum frontend validation or smoke

```powershell
Set-Location D:\workspace\go\pantheon-platform\pantheon-ops\frontend
npm run build
```

At minimum, record:

- the base commit
- which shared paths were synced
- which paths were intentionally left out
- whether local `business/*` paths stayed intact
- the minimum validation result for both base and ops

## 7. Runtime Isolation

- Runtime database is isolated from `pantheon-base`.
- Recommended default DSN target for this repository is `pantheon_ops`.
- Sharing a MySQL instance is allowed; sharing the same database schema with `pantheon-base` is not allowed.
