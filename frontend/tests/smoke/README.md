# Smoke Test Layout

English version: [README.en.md](./README.en.md)

Pantheon smoke tests are grouped by ownership boundary.

## Directories

- `platform/`: application shell, visual contracts, navigation chrome, cross-domain full-page checks.
- `system/`: backend management platform domains such as auth, IAM, org, config, permissions, and governance.
- `system/api/`: system-domain API smoke tests that do not need a browser page.
- `system/governance/`: module governance, generator workbench, permission remediation, and policy workbench flows.
- `business/<module>/`: business-domain smoke tests. CMDB lives in `business/cmdb/`.
- `helpers/`: shared test helpers only. Do not put test cases here.

## Script Rules

- System/platform checks are run through `npm run test:smoke:system`.
- Business checks are run through `npm run test:smoke:business`.
- Add each new business module as `test:smoke:business:<module>` and include it in `test:smoke:business`.

## Coverage Matrix

| Area | Feature points | Test file |
| :--- | :--- | :--- |
| platform shell | Login submit, app shell, command lock, tabs, layout overflow, structural UI guards | `platform/platform-shell.spec.ts` |
| platform visual | Login visual, dashboard, responsive system pages, localized modal copy | `platform/backoffice-ui-visual.spec.ts` |
| platform visual contract | Breadcrumb, function bars, table rhythm, filter rhythm, dict tab rhythm | `platform/shell-visual-contract.spec.ts` |
| platform full pass | Login and platform/system pages across PC, tablet, phone viewports | `platform/full-system-pages.spec.ts` |
| system/auth | Login/logout, idle timeout, session lock, login log, session list permissions | `system/system-pages.spec.ts` |
| system/iam | User, role, menu, permission page reachability and list-only permission behavior | `system/system-pages.spec.ts`, `system/role-authorization.spec.ts` |
| system/org | Department and post page reachability, form state, import/export | `system/system-pages.spec.ts`, `system/api/system-import-export.spec.ts` |
| system/config | Dict, setting, i18n, dynamic module, generator page flows and permission boundaries | `system/system-pages.spec.ts`, `system/governance/*.spec.ts` |
| system/audit | Login log, session, operation log export/cleanup and list-only permissions | `system/system-pages.spec.ts`, `system/api/system-import-export.spec.ts` |
| system forms | Required state, format state, submitting state, server error recovery | `system/system-form-state-matrix.spec.ts` |
| system governance | Confirm copy, submitting state, server error recovery for governance actions | `system/governance/system-governance-action-matrix.spec.ts` |
| permission governance | Workbench drawer, recommended API remediation, secondary verify retry | `system/governance/governance-insight-drawer.spec.ts`, `system/governance/permission-workbench-remediation*.spec.ts` |
| module governance | Module registry, generator validation, generated module register/purge, DB-import generation | `system/governance/module-governance*.spec.ts` |
| system import/export | Templates, user, dept, post, permission, dict, role, login log, operation log | `system/api/system-import-export.spec.ts` |
| business/cmdb | Operations menu, host list/detail, group tree/empty state, host create/detail/filter/status/delete, group create/update/member calculation/delete, invalid condition rejection | `business/cmdb/cmdb-pages.spec.ts` |

## Current Gaps

- CMDB SSH collect is not yet covered by browser smoke because it requires a reachable SSH target and disposable credentials. Backend/service tests should cover parser and error handling; add an environment-gated smoke when a stable test host exists.
- Business-module permission negative cases are not yet covered in browser smoke: hidden menu, list-only buttons, and 403 API behavior for CMDB.
- Audit assertions for CMDB create/update/delete/collect are not yet automated in smoke.

## Cleanup Rules

Delete a smoke test only when its ownership is retired or another active smoke test covers the same user-facing risk with the same or better assertions. Prefer moving and renaming over deleting when the test is still valid but misplaced.
