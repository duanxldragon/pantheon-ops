# Project Inheritance

Chinese version: [PROJECT_INHERITANCE.md](./PROJECT_INHERITANCE.md)

## 1. Inheritance Source

- Base repository: `../pantheon-base`
- Base release line: `release/0.8`
- Base version: `pantheon-base-v0.8.5` (`185e31f030108ad0e85cf8a6c87912222656ba3a`)
- Inheritance mode: `foundation-release-consumer`

This repository no longer treats `base/main` as the default consumer surface. `main` may continue to absorb optimization and governance work, while ops upgrades only to explicit foundation releases or tags by default.
`foundation-release.lock.json` is the machine-readable inheritance anchor. Default base-sync checks validate against that lock instead of directly against the current `pantheon-base` worktree.

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

Repository-local workflow skills live under `.agents/skills/` and cover:

- `repo-verify`
- `repo-pr-gate`
- `gh-address-comments`
- `repo-ci-triage`
- `gh-fix-ci`

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

In this document, "sync" should now be read as:

- choose a newer `pantheon-base` foundation release
- install that release artifact from GitHub Releases or a local archive
- upgrade the release consumed by ops
- repair only the real overlay breakpoints against that release

It should not mean continuously tracking `main`.

The recommended path is now release artifact installation plus the release consumer instead of manual tree copies:

- `npm run foundation:install`: download and install the release artifact pinned by `foundation-release.lock.json`
- `npm run foundation:install -- --archive <foundation-release-version>.tgz`: install the release artifact from a local archive
- `npm run upgrade:foundation:apply -- --manifest <bundle-root>\manifest.json --bundle <bundle-root>`
- if `pantheon-base/releases/<version>/manifest.json` already exists locally, ops can build the bundle and consume it directly via `npm run upgrade:foundation:local-plan -- --release-version <version>` or `npm run upgrade:foundation:local-apply -- --release-version <version>`
- the command syncs shared backend/frontend files, preserves ops-local overlays such as menu registries and generator workspace files, rewrites shared backend imports to the `pantheon-ops` module path, and re-runs frontend `base-sync` plus `menu-contract`
- during normal feature work, `npm run check:base-sync` validates only against the release artifact pinned in `foundation-release.lock.json`; use `npm run check:base-sync:workspace` explicitly when you want to compare ops against the current `pantheon-base` worktree and decide whether to start a new upgrade pass

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

The following paths are local extension points and are not expected to stay byte-identical with `pantheon-base`, but their intent should stay explicit:

- `backend/modules/business/generated_registry.go`: local ops `business/*` module registration is allowed
- `backend/modules/system/iam/menu/generated_component_registry.go`: local ops business page component keys are allowed
- `business.*` entries inside `backend/modules/system/i18n/builtin_locale_resources.json`: local ops business wording is allowed

## 6.3 Executable Sync Command List

Recommended order for one `base -> ops` sync pass:

1. finish the shared foundation change in `pantheon-base` and record the base commit

```powershell
git -C D:\workspace\go\pantheon-platform\pantheon-base rev-parse --short HEAD
```

2. install the currently pinned foundation release artifact in `pantheon-ops`

```powershell
Set-Location D:\workspace\go\pantheon-platform\pantheon-ops
npm run foundation:install
```

3. run the one-shot inheritance check in `pantheon-ops` so template linkage, inheritance markers, and shared backend/frontend alignment are validated together first

```powershell
npm run check:inheritance
```

4. during normal business development, validate only whether ops still matches the currently pinned release artifact

```powershell
npm run check:base-sync:backend
npm run check:base-sync:frontend
```

5. when you need to know whether recent base evolution now requires a sync pass, run the explicit workspace comparison

```powershell
npm run check:base-sync:workspace
```

6. once the workspace comparison says an upgrade is warranted, cut a new `pantheon-base` release first and then consume that release instead of hand-copying files

```powershell
npm run upgrade:foundation:local-plan -- --release-version base-v0.8.3
npm run upgrade:foundation:local-apply -- --release-version base-v0.8.3
```

7. if shared backend files must be synced, sync them file-by-file and do not overwrite `business/*`

```powershell
git diff --name-only -- D:\workspace\go\pantheon-platform\pantheon-base\backend
```

8. run the minimum validation in both repositories

```powershell
Set-Location D:\workspace\go\pantheon-platform\pantheon-base
go test ./...

Set-Location D:\workspace\go\pantheon-platform\pantheon-ops
go test ./...
npm run check:base-sync:backend
```

9. if the turn also touched shared frontend shell behavior, pagination, shared tables, or shared i18n, add minimum frontend validation or smoke

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

Common local command:

- `npm run foundation:install`: install the release artifact pinned by `foundation-release.lock.json`
- `npm run check:inheritance`: one-shot check for task-packet template linkage, inheritance markers, foundation lock, and shared backend/frontend alignment
- `npm run check:base-sync`: shared backend and frontend alignment check against the release artifact pinned by `foundation-release.lock.json`
- `npm run check:base-sync:workspace`: explicit check against the current `pantheon-base` worktree to decide whether a new upgrade pass is needed

## 7. Runtime Isolation

- Runtime database is isolated from `pantheon-base`.
- Recommended default DSN target for this repository is `pantheon_ops`.
- Sharing a MySQL instance is allowed; sharing the same database schema with `pantheon-base` is not allowed.
