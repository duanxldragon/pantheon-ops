# Smoke Test Layout

Pantheon smoke tests are grouped by ownership boundary.

## Directories

- `platform/`: application shell, visual contracts, navigation chrome, cross-domain full-page checks.
- `system/`: backend management platform domains such as auth, IAM, org, config, permissions, and governance.
- `system/api/`: system-domain API smoke tests that do not need a browser page.
- `system/governance/`: system-domain governance workbench, permission remediation, and module governance console flows.
- `business/generated/`: generated business-module smoke tests, including create/register/purge, database-import, master-detail, many-to-many, and auto-recycle flows.
- `business/<module>/`: business-domain smoke tests for a concrete built-in module when platform main carries that module source.
- `helpers/`: shared test helpers only. Do not put test cases here.

## Script Rules

- Platform shell and cross-domain UI checks are run through `npm run test:smoke:platform`.
- System-domain checks are run through `npm run test:smoke:system`.
- Business checks are run through `npm run test:smoke:business`.
- `npm run test:smoke:all` is the top-level smoke command and must cover login, system domains, and business-domain flows together.
- `npm run check:smoke-coverage-contract` is the executable drift gate for smoke entrypoints, spec files, and this coverage matrix; if routes, page ownership, or smoke files change, update them in the same patch.
- Browser-based smoke commands must run through `scripts/run-smoke-suite.mjs` so the started Vite server and Playwright `baseURL` stay on the same `PANTHEON_WEB_BASE_URL`.
- `scripts/run-smoke-suite.mjs` must only continue after receiving the ready signal from the server process it started; if the target port is already occupied, the smoke run must fail fast instead of silently reusing an unrelated server.
- Do not hard-code `http://127.0.0.1:5173` or `:5174` inside smoke tests, Playwright configs, or QA helper scripts. Read `PANTHEON_WEB_BASE_URL` instead.
- Do not add compatibility aliases such as `test:smoke:role-auth` or `test:smoke:impexp`; keep names aligned to ownership and actual surface.
- Add each new business module or business-runtime variant as `test:smoke:business:<module>` and include it in `test:smoke:business`.

## Test Tiers

Smoke runs in tiers so a change only pays for the depth it can actually affect. The tier is chosen
from the **changed paths**, not from a manual label.

| Tier | Command | Scope | Gate |
| :--- | :--- | :--- | :--- |
| L0 static | `npm run test:smoke:l0` | Smoke wiring contracts only (`check:smoke-coverage-contract`, `check:smoke-web-base`, `check:system-page-admission`). No browser, no server. | part of `npm run prebuild` |
| L1 critical path | `npm run test:smoke:l1` | `platform:contracts` + `system:routes` — shell/pagination contract plus every system route reachable. | **blocking on every PR** |
| L2 module | `npm run test:smoke:module:<module>` | One business module suite. `<module>` is `iam`, `config`, `i18n`, `audit`, `org` or `platform`. | **blocking, but only for touched modules** |
| L2 all | `npm run test:smoke:l2` | Every module suite plus platform surfaces and business flows. | manual / pre-release |
| L3 full | `npm run test:smoke:all` | Everything, including the PC/tablet/phone viewport matrix (`platform:full`). | report-only / pre-release |

Path-driven module selection lives in `scripts/resolve-smoke-modules.mjs`. The Quality Gates
workflow maps a PR's changed files to module suites and runs only those. **Any change that is not
unambiguously inside one module resolves to every module** — the default is deliberately the safe
one, so a shared-core edit can never silently skip a module's suite.

The per-module spec files were split out of the former `system/system-pages.spec.ts` monolith.
Shared scaffolding for them lives in `system/system-pages-helpers.ts`; that is a non-spec module, so
it is not a smoke entrypoint and is exempt from the coverage contract.

## Fixture Control

- Default behavior stays unchanged: browser smoke cleans generated modules before and after the run, and opted-in fixture scopes are cleaned automatically.
- Simple operator rule:
  - If the operator says `烟测保留数据`, run a preserve entry for the supported suite and keep the seeded records.
  - If the operator says `清理烟测数据`, run the cleanup entry directly.
- To preserve supported smoke fixtures after a run, set `PANTHEON_SMOKE_PRESERVE_FIXTURES=1` for that command.
- Convenience entries:
  - `npm run smoke:preserve`
  - `npm run smoke:preserve:platform-shell`
  - `npm run smoke:preserve:system`
  - `npm run test:smoke:platform:shell:preserve`
  - `npm run test:smoke:system:preserve`
- Manual cleanup entries:
  - `npm run smoke:cleanup`
  - `npm run smoke:cleanup:iam`
  - `npm run smoke:cleanup:org`
  - `npm run smoke:cleanup:config`
- Current preserve coverage is intentionally limited to page-review fixtures such as user, role, dept, post, readonly viewer accounts, dict/i18n/permission UI seeds, and related API smoke records.
- Generated business modules, codegen/runtime registry mutations, schema artifacts, and generated i18n resources are still treated as ephemeral test artifacts and should not be preserved by default.

## Script Entry Map

- `test:smoke:platform:contracts` -> `platform/shell-visual-contract.spec.ts`, `platform/pagination-contract.spec.ts`
- `test:smoke:platform:surfaces` -> `platform/platform-shell.spec.ts`, `platform/backoffice-ui-visual.spec.ts`, `platform/shell-top-panels.spec.ts`, `platform/system-layout-contract.spec.ts`, `platform/system-secondary-route-visual.spec.ts`, `platform/system-secondary-route-interaction.spec.ts`, `platform/system-secondary-route-failure.spec.ts`, `platform/system-secondary-route-permission.spec.ts`
- `test:smoke:platform:shell:preserve` -> `platform/platform-shell.spec.ts`
- `test:smoke:platform:full` -> `platform/full-system-pages.spec.ts`, `platform/full-page-audit.spec.ts`
- `test:smoke:system:routes` -> `system/system-routes.spec.ts`
- `test:smoke:system:iam` -> `system/system-iam.spec.ts`
- `test:smoke:system:config` -> `system/system-config.spec.ts`
- `test:smoke:system:i18n` -> `system/system-i18n.spec.ts`
- `test:smoke:system:audit` -> `system/system-audit.spec.ts`
- `test:smoke:system:org` -> `system/system-org.spec.ts`
- `test:smoke:system:platform` -> `system/system-platform.spec.ts`
- `test:smoke:system:forms` -> `system/system-form-state-matrix.spec.ts`
- `test:smoke:system:iam-authz` -> `system/role-authorization.spec.ts`
- `test:smoke:system:governance` -> `system/governance/system-governance-action-matrix.spec.ts`, `system/governance/governance-insight-drawer.spec.ts`, `system/governance/permission-workbench-remediation.spec.ts`, `system/governance/permission-workbench-remediation-real.spec.ts`, `system/governance/module-governance.spec.ts`, `system/governance/cleanup-range-ui.spec.ts`
- `test:smoke:system:api` -> `system/api/system-import-export.spec.ts`, `system/api/system-batch-delete.spec.ts`
- `test:smoke:business:generated` -> `business/generated/module-governance-real.spec.ts`
- `test:smoke:business:database-import` -> `business/generated/module-governance-host-real.spec.ts`
- `test:smoke:business:master-detail` -> `business/generated/module-master-detail-real.spec.ts`
- `test:smoke:business:many-to-many` -> `business/generated/module-many-to-many-real.spec.ts`
- `test:smoke:business:auto-recycle` -> `business/generated/module-auto-recycle-real.spec.ts`

## Coverage Matrix

| Area                      | Feature points                                                                                                                                                                      | Test file                                                                                                                                                                                                        |
| :------------------------ | :---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | :--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| platform shell            | Login submit, app shell, command lock, tabs, layout overflow, structural UI guards                                                                                                  | `platform/platform-shell.spec.ts`                                                                                                                                                                                |
| platform visual           | Login visual, dashboard, responsive system pages, localized modal copy                                                                                                              | `platform/backoffice-ui-visual.spec.ts`                                                                                                                                                                          |
| platform visual contract  | Breadcrumb, function bars, table rhythm, filter rhythm, dict tab rhythm                                                                                                             | `platform/shell-visual-contract.spec.ts`                                                                                                                                                                         |
| platform overlays         | Notice center, platform preferences, profile narrow-screen containment                                                                                                              | `platform/shell-top-panels.spec.ts`                                                                                                                                                                              |
| platform pagination       | Shared pager contract, alignment, boundary navigation, mobile wrapping                                                                                                              | `platform/pagination-contract.spec.ts`                                                                                                                                                                           |
| platform layout contract  | Dense system page spacing, card rhythm, table container stability                                                                                                                   | `platform/system-layout-contract.spec.ts`                                                                                                                                                                        |
| platform secondary routes | Secondary route visuals, interaction states, failure states, permission fallback                                                                                                    | `platform/system-secondary-route-visual.spec.ts`, `platform/system-secondary-route-interaction.spec.ts`, `platform/system-secondary-route-failure.spec.ts`, `platform/system-secondary-route-permission.spec.ts` |
| platform full pass        | Login and platform/system pages across PC, tablet, phone viewports                                                                                                                  | `platform/full-system-pages.spec.ts`                                                                                                                                                                             |
| platform full-page audit  | Per-page screenshots, console errors, broken-state, overflow, create-dialog checks, and the sidebar-beside-content layout regression guard                                          | `platform/full-page-audit.spec.ts`                                                                                                                                                                               |
| system routes             | Every system and workspace route reachable, page title, no page error, console-clean bootstrap                                                                                      | `system/system-routes.spec.ts`                                                                                                                                                                                   |
| system/auth               | Login/logout, idle timeout, session lock, login log, session list permissions                                                                                                       | `system/system-iam.spec.ts`, `system/system-audit.spec.ts`, `system/system-platform.spec.ts`                                                                                                                     |
| system/iam                | User, role, menu, permission page reachability and list-only permission behavior                                                                                                    | `system/system-iam.spec.ts`, `system/role-authorization.spec.ts`                                                                                                                                                 |
| system/org                | Department and post page reachability, form state, import/export                                                                                                                    | `system/system-org.spec.ts`, `system/api/system-import-export.spec.ts`                                                                                                                                           |
| system/config             | Dict, setting, UI preference, dynamic module, generator page flows and permission boundaries                                                                                        | `system/system-config.spec.ts`, `system/governance/*.spec.ts`                                                                                                                                                    |
| system/i18n               | i18n workspace, detail/edit/create/delete dialogs, CSV import, permission boundary, refresh sync                                                                                    | `system/system-i18n.spec.ts`                                                                                                                                                                                     |
| system/audit              | Login log, session, operation log export/cleanup and list-only permissions                                                                                                          | `system/system-audit.spec.ts`, `system/governance/cleanup-range-ui.spec.ts`, `system/api/system-import-export.spec.ts`                                                                                           |
| system/platform           | Lock screen, module registry lifecycle, cross-cutting page admission and ownership labels                                                                                           | `system/system-platform.spec.ts`                                                                                                                                                                                 |
| system forms              | Required state, format state, submitting state, server error recovery                                                                                                               | `system/system-form-state-matrix.spec.ts`                                                                                                                                                                        |
| system governance         | Confirm copy, submitting state, server error recovery for governance actions                                                                                                        | `system/governance/system-governance-action-matrix.spec.ts`                                                                                                                                                      |
| permission governance     | Workbench drawer, recommended API remediation, secondary verify retry                                                                                                               | `system/governance/governance-insight-drawer.spec.ts`, `system/governance/permission-workbench-remediation.spec.ts`, `system/governance/permission-workbench-remediation-real.spec.ts`                           |
| module governance         | Module registry, generator validation, overwrite confirmation, datasource management, and workbench gating                                                                          | `system/governance/module-governance.spec.ts`                                                                                                                                                                    |
| system cleanup governance | Cleanup range submission, inline action bar layout, selected-row CSV export                                                                                                         | `system/governance/cleanup-range-ui.spec.ts`                                                                                                                                                                     |
| system batch delete       | Batch delete remove/report behavior across user, role, dept, post, dict, permission                                                                                                 | `system/api/system-batch-delete.spec.ts`                                                                                                                                                                         |
| business generated module | `test:smoke:business:generated`, login, generate/register temporary business module, module registry cleanup                                                                        | `business/generated/module-governance-real.spec.ts`                                                                                                                                                              |
| business database-import  | `test:smoke:business:database-import`, bootstrap `biz_cmdb_host`, import source table, register generated business module, open generated page, purge without dropping source table | `business/generated/module-governance-host-real.spec.ts`                                                                                                                                                         |
| business master-detail    | `test:smoke:business:master-detail`, login, generated business detail page child-row create/edit                                                                                    | `business/generated/module-master-detail-real.spec.ts`                                                                                                                                                           |
| business many-to-many     | `test:smoke:business:many-to-many`, login, generated business detail page relation bind/unbind                                                                                      | `business/generated/module-many-to-many-real.spec.ts`                                                                                                                                                            |
| business auto-recycle     | `test:smoke:business:auto-recycle`, login, module purge with managed business table auto-recycle                                                                                    | `business/generated/module-auto-recycle-real.spec.ts`                                                                                                                                                            |
| system import/export      | Templates, user, dept, post, permission, dict, role, login log, operation log                                                                                                       | `system/api/system-import-export.spec.ts`                                                                                                                                                                        |

## Current Gaps

- Platform main currently covers generated business-module runtime smoke.
- Concrete built-in business modules should still keep their own smoke in the owning business repository or in `business/<module>/` when the source is hosted here.

## Cleanup Rules

Delete a smoke test only when its ownership is retired or another active smoke test covers the same user-facing risk with the same or better assertions. Prefer moving and renaming over deleting when the test is still valid but misplaced.

## Ops Business Overlay

- `test:smoke:business:cmdb` -> `business/cmdb/cmdb-pages.spec.ts`
- `test:smoke:business:deploy` -> `business/deploy/deploy-pages.spec.ts`
- `test:smoke:business:deploy:api` -> `business/deploy/deploy-api.spec.ts`
- `test:smoke:business:observability` -> `business/observability-filedist-pages.spec.ts`, `business/observability-menu.spec.ts`
- `test:smoke:business:service` -> `business/service/service-pages.spec.ts`
- `test:smoke:business:k8s` -> `business/k8s/k8s-pages.spec.ts`
