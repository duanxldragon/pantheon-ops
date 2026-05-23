# Smoke Test Layout

English version: [README.en.md](./README.en.md)

Pantheon smoke tests are grouped by ownership boundary.

## Directories

- `platform/`: application shell, visual contracts, navigation chrome, and cross-domain full-page checks.
- `auth/`: login and preference behavior that still belongs to the shared platform shell.
- `system/`: backend management domains such as IAM, org, config, and governance.
- `system/api/`: system-domain API smoke tests that do not need a browser page.
- `system/governance/`: generator, module registry, cleanup governance, and permission remediation flows.
- `business/cmdb/`: CMDB business smoke owned by ops.
- `business/deploy/`: deploy business smoke owned by ops.
- `helpers/`: shared helpers only. Do not put test cases here.

## Script Rules

- Platform shell and cross-domain UI checks are run through `npm run test:smoke:platform`.
- System-domain checks are run through `npm run test:smoke:system`.
- Business checks are run through `npm run test:smoke:business`.
- `npm run test:smoke:all` is the top-level smoke command and must cover login, system domains, and business-domain flows together.
- `npm run check:smoke-coverage-contract` is the drift gate for smoke entrypoints, spec files, and this coverage matrix.
- Browser-based smoke commands must run through `scripts/run-smoke-suite.mjs` so the started Vite server and Playwright `baseURL` stay on the same `PANTHEON_WEB_BASE_URL`.
- Do not hard-code `http://127.0.0.1:5173` or `:5174` inside browser smoke tests, Playwright configs, or QA helper scripts. Read `PANTHEON_WEB_BASE_URL` instead.
- Do not hard-code `--proxy-target http://127.0.0.1:8080` in smoke npm scripts. Set `PANTHEON_API_PROXY_TARGET` in the environment so suites follow the active backend instance.
- When routes, smoke specs, or test scripts change, update the matching smoke command and this README in the same patch.

## Fixture Control

- Default behavior stays unchanged: smoke fixtures are cleaned automatically.
- Simple operator rule:
  - If the operator says `烟测保留数据`, run a preserve entry for the supported suite and keep the seeded records.
  - If the operator says `清理烟测数据`, run the cleanup entry directly.
- Preserve entries:
  - `npm run smoke:preserve`
  - `npm run smoke:preserve:platform-shell`
  - `npm run smoke:preserve:system-pages`
  - `npm run test:smoke:platform:shell:preserve`
  - `npm run test:smoke:system:pages:preserve`
- Cleanup entries:
  - `npm run smoke:cleanup`
  - `npm run smoke:cleanup:iam`
  - `npm run smoke:cleanup:org`
  - `npm run smoke:cleanup:config`

## Script Entry Map

- `test:smoke:platform:contracts` -> `platform/shell-visual-contract.spec.ts`, `auth/login-language-preference.spec.ts`
- `test:smoke:platform:surfaces` -> `platform/platform-shell.spec.ts`, `platform/backoffice-ui-visual.spec.ts`, `platform/system-secondary-route-visual.spec.ts`, `system/system-form-state-matrix.spec.ts`
- `test:smoke:platform:shell:preserve` -> `platform/platform-shell.spec.ts`
- `test:smoke:platform:full` -> `platform/full-system-pages.spec.ts`
- `test:smoke:system:pages` -> `system/system-pages.spec.ts`
- `test:smoke:system:pages:preserve` -> `system/system-pages.spec.ts`
- `test:smoke:system:iam-authz` -> `system/role-authorization.spec.ts`
- `test:smoke:system:governance` -> `system/governance/system-governance-action-matrix.spec.ts`, `system/governance/governance-insight-drawer.spec.ts`, `system/governance/permission-workbench-remediation.spec.ts`, `system/governance/permission-workbench-remediation-real.spec.ts`, `system/governance/module-governance.spec.ts`, `system/governance/module-governance-real.spec.ts`, `system/governance/module-governance-host-real.spec.ts`, `system/governance/cleanup-range-ui.spec.ts`
- `test:smoke:system:generator-workbench` -> `system/governance/module-governance.spec.ts`
- `test:smoke:system:module-governance-host` -> `system/governance/module-governance-host-real.spec.ts`
- `test:smoke:system:api` -> `system/api/system-import-export.spec.ts`, `system/api/system-batch-delete.spec.ts`
- `test:smoke:business:cmdb` -> `business/cmdb/cmdb-pages.spec.ts`
- `test:smoke:business:deploy` -> `business/deploy/deploy-pages.spec.ts`
- `test:smoke:business:master-detail` -> `system/governance/module-master-detail-real.spec.ts`
- `test:smoke:business:many-to-many` -> `system/governance/module-many-to-many-real.spec.ts`
- `test:smoke:business:auto-recycle` -> `system/governance/module-auto-recycle-real.spec.ts`

## Coverage Matrix

| Area | Feature points | Test file |
| :--- | :--- | :--- |
| platform shell | Login submit, app shell, command lock, tabs, layout overflow, structural UI guards | `platform/platform-shell.spec.ts` |
| platform visual | Login visual, dashboard, responsive system pages, localized modal copy | `platform/backoffice-ui-visual.spec.ts` |
| platform secondary routes | Profile, user detail, and setting group routes keep stable shell, identity, and viewport evidence | `platform/system-secondary-route-visual.spec.ts` |
| platform visual contract | Breadcrumb, function bars, table rhythm, filter rhythm, dict tab rhythm | `platform/shell-visual-contract.spec.ts` |
| platform full pass | Login and platform/system pages across PC, tablet, phone viewports | `platform/full-system-pages.spec.ts` |
| login preference | Explicit login-page language overrides saved preference after sign-in | `auth/login-language-preference.spec.ts` |
| system reachability | Auth, IAM, org, config, generator, and audit page reachability | `system/system-pages.spec.ts` |
| system IAM authz | User, role, menu, permission page list-only permission behavior | `system/role-authorization.spec.ts` |
| system forms | Required state, format state, submitting state, server error recovery | `system/system-form-state-matrix.spec.ts` |
| system governance actions | Confirm copy, submitting state, server error recovery for governance actions | `system/governance/system-governance-action-matrix.spec.ts` |
| permission governance | Workbench drawer, recommended remediation, secondary verify retry | `system/governance/governance-insight-drawer.spec.ts`, `system/governance/permission-workbench-remediation.spec.ts`, `system/governance/permission-workbench-remediation-real.spec.ts` |
| module governance | Module registry, generator validation, register/purge, and generator workbench entry | `system/governance/module-governance.spec.ts`, `system/governance/module-governance-real.spec.ts` |
| module database import | DB import source table registration and generated module open/purge flow | `system/governance/module-governance-host-real.spec.ts` |
| cleanup governance | Cleanup range submission, inline action bar layout, selected-row CSV export | `system/governance/cleanup-range-ui.spec.ts` |
| system batch delete | Batch delete remove/report behavior across user, role, dept, post, dict, permission | `system/api/system-batch-delete.spec.ts` |
| system import/export | Templates, user, dept, post, permission, dict, role, login log, operation log | `system/api/system-import-export.spec.ts` |
| business/cmdb | Host list/detail, group tree and empty state, create/filter/status/delete flows | `business/cmdb/cmdb-pages.spec.ts` |
| business/deploy | Deploy package/task pages and related user-facing route flows | `business/deploy/deploy-pages.spec.ts` |
| generated master-detail | Generated business detail page child-row create/edit interaction | `system/governance/module-master-detail-real.spec.ts` |
| generated many-to-many | Generated business detail page relation bind/unbind interaction | `system/governance/module-many-to-many-real.spec.ts` |
| generated auto-recycle | Temporary generated business table auto-recycle on purge | `system/governance/module-auto-recycle-real.spec.ts` |

## Current Gaps

- CMDB SSH collect still needs an environment-gated smoke because it requires a reachable SSH target and disposable credentials.
- Deploy business negative-permission paths are not yet covered in browser smoke.
- Generated business runtime smoke still lives under `system/governance/` paths; ownership is expressed through script entrypoints until the directory layout is fully normalized.

## Cleanup Rules

Delete a smoke test only when its ownership is retired or another active smoke test covers the same user-facing risk with the same or better assertions. Prefer moving and renaming over deleting when the test is still valid but misplaced.
