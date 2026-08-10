# Deploy Business Module Design

Chinese version: [BUSINESS_DEPLOY_MODULE_DESIGN.md](./BUSINESS_DEPLOY_MODULE_DESIGN.md)

Updated: 2026-07-17

Type: Design
Layer: `business/deploy`
Status: Active

This document defines the Deploy business module, a peer of `business/cmdb` and `business/bizscope`. It turns software, targets, execution state, and per-host results into traceable deployment tasks and preserves the boundary for later Agent execution.

---

## 1. Module Overview

Deploy owns software packages, reusable templates and steps, deployment tasks, per-host execution detail, uploaded source-package metadata, and execution traces. It supports manual, simulated, and real SSH execution and has validated a real-host loop covering onboarding, business-scope binding, uninstall, install/reinstall, and CMDB status/component write-back.

It does not own CMDB inventory, realtime connectivity monitoring, credential vaulting, the Agent process itself, a dedicated rollback entry point, or system IAM implementation.

## 2. Boundaries and Dependencies

| Area | Allowed | Forbidden |
| :--- | :--- | :--- |
| platform | shared responses, audit metadata, module registration | changing platform-shell business logic |
| system/auth | login principal context | importing auth services |
| system/iam | menu/permission/Casbin results | implementing roles inside Deploy |
| system/org | data-scope context | importing org repositories |
| business/cmdb | explicit read/query and write-back capabilities | direct `biz_cmdb_*` table or CMDB service/repository access |
| business/bizscope | business-scope trust boundary | deploying directly to unassigned hosts |

Current code reality: target resolution, host status write-back, and installed-component write-back are already routed through a CMDB-local capability. Deploy no longer directly reads or updates CMDB tables from its service layer.

### 2.1 CMDB Query Contract

| Capability | Purpose |
| :--- | :--- |
| `ListSelectableHosts` | scoped host options |
| `ListSelectableGroups` | scoped group options |
| `PreviewGroupMembers` | group member preview |
| `ResolveTaskTargets` | deduplicated host snapshots before execution |

Every result must obey `DataScopeReq`, and Deploy receives only task-required snapshots rather than CMDB-internal JSON structures.

## 3. Menus and Naming

| Menu key | Path | Component key | Page permission |
| :--- | :--- | :--- | :--- |
| `business.deploy.package` | `/operations/deploy/package` | `business/deploy/package/DeployPackageList` | `business:deploy:package:view` |
| `business.deploy.template` | `/operations/deploy/template` | `business/deploy/template/DeployTemplateList` | `business:deploy:template:list` |
| `business.deploy.task` | `/operations/deploy/task` | `business/deploy/task/DeployTaskList` | `business:deploy:task:view` |
| `business.deploy.task.detail` | `/operations/deploy/task/:id` | `business/deploy/task/DeployTaskDetail` | `business:deploy:task:detail` |

The module name is `business.deploy`. List and detail permissions are separate. Templates use a real header-plus-steps model; packages remain reusable execution assets.

## 4. Core Objects

| Object | Purpose | Key fields |
| :--- | :--- | :--- |
| `DeployPackage` | software component | name, version, execution mode, template code, source object, install/uninstall commands, status |
| `DeployTemplate` | reusable task template | name, version, category, default action, package, parameter schema, status |
| `DeployTemplateStep` | ordered template step | code, type, action, package, parameters, sort |
| `DeployTask` | deployment task | package/template snapshots, business scope, action, targets, executor, status, timing |
| `DeployTaskHost` | host execution detail | host snapshot, status, trace steps, stdout/stderr, error |

## 5. State Transitions

```text
draft -> pending -> running -> success
                         \-> failed
pending/running -> canceled
```

Host details move from `pending` to `running`, then `success` or `failed`; pending rows may become `skipped`.

Current executor modes are `manual`, `simulated`, and `ssh`; `agent` is reserved. Actions are `install`, `uninstall`, `upgrade`, and `reinstall`. SSH execution is real, and each host records connection, rendering, execution, service, and result-writeback steps.

Template steps execute in order and support `step_type=package` and `step_type=script`. Script steps use `step_config.script` with optional `precheckCommand` and `postcheckCommand`. Fixed-template `upgrade/reinstall` stops the target systemd service before replacing binaries.

Uninstall removes the matching installed-component snapshot. A host stays `online` while other components remain and returns to `assigned` only after the last component is removed.

## 6. Data Model

### 6.1 `biz_deploy_package`

Stores component identity, description, install/uninstall commands, `fixed/orchestrated` mode, template code/config, uploaded source metadata, status, audit actors, timestamps, and soft delete. Name/version is unique within the current single-tenant scope.

### 6.2 `biz_deploy_task`

Stores template/package/business-scope snapshots, action, target type/IDs, executor type, execution mode, template parameters, lifecycle state, external task ID, timing, actors, and soft delete.

### 6.3 `biz_deploy_task_host`

Stores task/host IDs, host snapshots, status, JSON trace steps, stdout/stderr, error message, executor ID, report time, and execution timestamps.

### 6.4 `biz_deploy_template`

Stores template identity, category, execution mode, default action/package, template code/config, parameter schema, status, actors, timestamps, and soft delete.

### 6.5 `biz_deploy_template_step`

Stores ordered package or script steps, action, package snapshot, template code/params, extensible `step_config`, and timestamps.

### 6.6 Tenant Readiness

The current runtime is single-tenant and does not add `tenant_id`. Package uniqueness, task lists/exports, and audit queries must retain a future scope-injection point.

## 7. API Design

Prefix: `/api/v1/business/deploy`

### 7.1 Common Contract

Responses use shared success/failure envelopes; lists return `items/total/page/pageSize`; writes return identity, state, and update time. Errors use `business.deploy.package.*`, `business.deploy.task.*`, and `business.deploy.taskHost.*`. Every API requires an explicit permission, and all writes are audited.

### 7.2 Endpoint List

| Method | Path | Purpose | Permission |
| :--- | :--- | :--- | :--- |
| `GET` | `/packages` | package list | `business:deploy:package:list` |
| `GET` | `/packages/:id` | package detail | **Permission mapping gap:** the route exists, but `business:deploy:package:list` currently maps only `GET /packages`, not this route. Track in `2026-07-17-business-permission-gaps`. |
| `POST` | `/packages` | create package | `business:deploy:package:create` |
| `PUT` | `/packages/:id` | update package | `business:deploy:package:update` |
| `DELETE` | `/packages/:id` | delete package | `business:deploy:package:delete` |
| `GET` | `/templates` | template list | `business:deploy:template:list` |
| `POST` | `/templates` | create template | `business:deploy:template:create` |
| `GET` | `/templates/:id` | template detail | `business:deploy:template:list` |
| `PUT` | `/templates/:id` | update template | `business:deploy:template:update` |
| `DELETE` | `/templates/:id` | delete template | `business:deploy:template:delete` |
| `GET` | `/tasks` | task list | `business:deploy:task:list` |
| `POST` | `/tasks` | create task | `business:deploy:task:create` |
| `GET` | `/tasks/:id` | task detail | `business:deploy:task:detail` |
| `PUT` | `/tasks/:id` | update an unstarted task | `business:deploy:task:update` |
| `DELETE` | `/tasks/:id` | delete a `draft/pending` task and its task-host rows | `business:deploy:task:delete` |
| `POST` | `/tasks/:id/start` | start task | `business:deploy:task:start` |
| `POST` | `/tasks/:id/cancel` | cancel task | `business:deploy:task:cancel` |
| `POST` | `/task-hosts/:id/result` | manually mark host result | `business:deploy:task:mark-result` |
| `POST` | `/task-hosts/:id/report` | reserved Agent report; currently bound to the same handler as `/result`, so manual marking and executor reporting are not yet distinguished | Agent-stage follow-up |

The `GET /packages/:id` detail route is implemented but remains outside the current `business:deploy:package:list` Casbin mapping; it is a documented permission gap, not an authorized-route claim.

### 7.3 Request and Response Notes

#### 7.3.1 `GET /packages`

Supports keyword/status pagination and distinguishes initial empty from filtered empty results.

#### 7.3.2 `POST /packages` / `PUT /packages/:id`

Accept package identity, commands, execution mode, template/source metadata, and status. Key errors include `nameRequired`, `versionRequired`, `nameVersionConflict`, and `commandTooLong` under `business.deploy.package.*`.

#### 7.3.3 `GET /tasks`

Supports keyword, status, package, target, executor, date range, and pagination filters and returns task identity, snapshots, counts, status, and timing.

#### 7.3.4 `POST /tasks`

Accepts task name, template or package, business scope, target type/IDs, executor type, parameters, and remark. The server resolves scoped target snapshots before persistence. Host status must match the action: install/reinstall allow `assigned/online`; uninstall/upgrade require `online`.

#### 7.3.5 `GET /tasks/:id`

Returns task header, per-host detail, and aggregates. Missing and forbidden cases use `business.deploy.task.notFound` and `business.deploy.task.forbidden`.

#### 7.3.6 `POST /tasks/:id/start`

Only `draft/pending` tasks may start. SSH start requires request-time credentials and host fingerprint validation. Template parameters, commands, variables, and source package must validate before state moves to `running`.

#### 7.3.7 `POST /tasks/:id/cancel`

Only `pending/running` tasks may be canceled; otherwise return `business.deploy.task.invalidCancelState`.

#### 7.3.8 `DELETE /tasks/:id`

Only `draft/pending` tasks may be deleted. Deletion also removes related `biz_deploy_task_host` rows. Invalid state returns `business.deploy.task.invalidDeleteState`.

#### 7.3.9 `POST /task-hosts/:id/result`

Accepts `success/failed/skipped`, stdout/stderr, and an error message. Failed results require a reason and use canonical `business.deploy.taskHost.*` keys.

## 8. Frontend and UI

Frontend path: `frontend/src/modules/business/deploy`.

### 8.1 Shared Components

Reuse Base page containers, headers, filters, tables, list actions, form sections, submit bars, modals, and state containers. Do not add a UI library.

### 8.2 Visual and State Rules

Use inherited theme/status tokens, responsive rules, and loading/empty/error/forbidden states. Destructive actions require confirmation.

### 8.3 Notifications

Success uses localized messages; failures resolve backend error keys. Start, cancel, and result marking require confirmation. V1 uses polling/manual refresh rather than WebSocket notifications.

### 8.4 `DeployPackageList`

Provides keyword/status filters, package/version/mode/status columns, create/edit/delete/enable actions, and complete empty/loading/error/forbidden states.

### 8.5 `DeployPackageForm`

Supports fixed and orchestrated packages, uploaded offline archives, install/uninstall commands, and built-in templates `nginx_systemd`, `mysql_systemd`, `redis_systemd`, `minio_systemd`, and `harbor_offline`.

### 8.6 `DeployTaskList`

Provides task/status filters, target and business-scope columns, state-aware detail/start/cancel/edit/delete actions, and five-second polling while pending/running tasks exist.

### 8.7 `DeployTaskForm`

Selects template/package, business scope, targets, executor, action, parameters, and remark. Host selection is business-scope-first, action/status compatibility is enforced, and fixed templates default to SSH.

### 8.8 `DeployTaskDetail`

Shows task summary, state path, per-host status distribution, trace steps, raw stdout/stderr, state-aware operations, and optional audit timeline.

### 8.9 Result-Marking Interaction

Success needs no input; failure requires `business.deploy.taskHost.markFailed.reasonRequired`. Updating a host row may immediately re-aggregate task status.

### 8.10 i18n Prefixes

Use `business.deploy.package.*`, `business.deploy.task.*`, `business.deploy.taskHost.*`, and `business.deploy.state.*` for all user-facing copy.

### 8.11 Responsive Behavior

Tables follow Base column priorities. Below `md`, task-detail sections stack and host rows become cards.

### 8.12 Base References

UI behavior follows Base frontend, accessibility, theme, dark-mode, empty/loading/error, and responsive design documents.

## 9. Permissions and Audit

Page/navigation permissions are `business:deploy:package:view`, `business:deploy:task:view`, and `business:deploy:task:detail`.

Action/API permissions include package `view/list/create/update/delete`, template `list/create/update/delete`, and task `view/list/detail/create/update/delete/start/cancel/mark-result`.

Template permission truth:

- all four `business:deploy:template:list/create/update/delete` API route mappings exist in `permission_policies.go`
- `deploy_seed.go` currently seeds only the template list C-menu with `PagePerm=business:deploy:template:list`
- create/update/delete F-button seeds are missing, so role authorization for those buttons must not be documented as complete; track in `2026-07-17-business-permission-gaps`

Audit package create/update/delete; task create/update/delete; task start/cancel; and host result marking. High-sensitivity events record operator, task ID, target count, and state transition. Future Agent reports must distinguish executor reporting from manual marking.

## 10. Acceptance Criteria

- package CRUD and template-backed task creation work
- tasks can resolve host or group targets through explicit CMDB capabilities
- start creates per-host detail and SSH can execute and write back results
- `DELETE /tasks/:id` enforces state and cleans task-host rows
- result marking aggregates task state
- menu, permission, i18n, build, and responsive checks pass
- Deploy does not directly access `biz_cmdb_*` tables or `modules/system/*` internals
- page/list/detail/action permissions remain separate
- all errors use i18n keys without hardcoded English fallbacks
