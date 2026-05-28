# Deploy Business Module Design

Chinese version: [BUSINESS_DEPLOY_MODULE_DESIGN.md](./BUSINESS_DEPLOY_MODULE_DESIGN.md)

Updated: 2026-05-28

Type: Design  
Layer: `business/deploy`  
Status: Active

This document defines the Deploy module for `pantheon-ops`. It is the second business module after CMDB and establishes task-based software deployment over CMDB hosts, groups, and business scopes under the Workspace / Operations menu.

## Scope

The module owns:

- deployable software packages
- reusable deploy templates and template steps
- deployment tasks
- per-host execution detail records
- uploaded source-package metadata for offline deployment
- task execution traces for operator observability
- a validated real-host lifecycle loop for host onboarding, business-scope binding, SSH uninstall, SSH install or reinstall, and CMDB status write-back

The module does not own:

- CMDB inventory maintenance
- credential vaulting
- realtime monitoring
- agent runtime implementation in v1
- dedicated rollback execution entry points
- system-domain IAM implementation

## Boundary Model

Deploy is not allowed to read `biz_cmdb_*` tables directly as the target architecture.

Current code reality: deploy target resolution and CMDB host write-backs are now routed through a CMDB-local capability, so deploy no longer reads or updates `biz_cmdb_*` tables directly from its own service layer. A later cleanup can still standardize how such cross-module capabilities are registered and shared.

It must consume CMDB through explicit read-only capabilities such as:

- `ListSelectableHosts`
- `ListSelectableGroups`
- `PreviewGroupMembers`
- `ResolveTaskTargets`

All cross-module reads must still obey `DataScopeReq`.

## Core Objects

- `DeployPackage`
- `DeployTemplate`
- `DeployTemplateStep`
- `DeployTask`
- `DeployTaskHost`

Primary tables:

- `biz_deploy_package`
- `biz_deploy_template`
- `biz_deploy_template_step`
- `biz_deploy_task`
- `biz_deploy_task_host`

The repository currently remains single-tenant, but task and audit structures must keep a future scope-injection point.

## State Model

Task states:

- `draft`
- `pending`
- `running`
- `success`
- `failed`
- `canceled`

Host execution states:

- `pending`
- `running`
- `success`
- `failed`
- `skipped`

Current execution modes:

- `manual`
- `simulated`
- `ssh`

Per-host trace steps must capture connect, render, install or uninstall, service, and result-writeback phases for executed tasks.

Task actions in the current implementation:

- `install`
- `uninstall`
- `upgrade`
- `reinstall`

`upgrade` and `reinstall` currently reuse the install-oriented fixed-template flow.

`uninstall` removes the matching installed-component entry from CMDB. If other installed components still remain on the host after removal, the host stays `online`; it returns to `assigned` only when the last installed component is removed.

Template execution now runs step-by-step in order. The current implementation supports both `step_type=package` and `step_type=script`.

For `script` steps, `step_config.script` stores the main script body and optional `precheckCommand` / `postcheckCommand` hooks can be used for guard checks around the main action.

Each step writes per-host trace events and aggregates stdout and stderr by `precheck / script / postcheck` phase.

For fixed templates, the `reinstall / upgrade` path now stops the target systemd service and drains the old process before copying binaries, avoiding `Text file busy` failures during in-place replacement.

Later reserved modes:

- `agent`

## APIs

Prefix: `/api/v1/business/deploy`

Main resources:

- packages: list, create, update, delete
- templates: list, create, detail, update, delete
- tasks: list, create, detail, update, start, cancel
- task hosts: mark result, reserved agent report endpoint

Permissions follow:

- `business:deploy:package:*`
- `business:deploy:template:*`
- `business:deploy:task:*`

## Error-Key Namespace

Canonical backend keys stay inside:

- `business.deploy.package.*`
- `business.deploy.task.*`
- `business.deploy.taskHost.*`

Examples include `business.deploy.task.notFound`, `business.deploy.task.invalidStartState`, and `business.deploy.taskHost.markFailed.reasonRequired`.

## Frontend Shape

Main pages and forms:

- `DeployPackageList`
- `DeployTemplateList`
- `DeployPackageForm`
- `DeployTaskList`
- `DeployTaskForm`
- `DeployTaskDetail`

Current UI behavior also includes:

- fixed-template vs orchestrated package definitions
- built-in fixed template codes for `nginx_systemd`, `mysql_systemd`, `redis_systemd`, `minio_systemd`, and `harbor_offline`
- uploaded source archives for offline package delivery
- business-scope-first host selection
- per-host trace-step rendering in task detail
- the task-template page now uses a real template model with dedicated template and step tables
- fixed-template parameter panels now render dynamically from the selected template code instead of hard-coding the nginx case

UI rules:

- reuse Pantheon Base page containers, tables, filters, modal forms, and state containers
- keep visual tokens aligned with inherited base docs
- localize all user-facing copy through `business.deploy.*`
- separate list, detail, and action permissions in both UI and API behavior

## Audit and Governance

The following actions must always be audited:

- package create, update, delete
- task create, update
- task start, cancel
- host-level result marking

High-sensitivity actions must record operator identity, task identity, target counts, and status transitions.

## Acceptance Focus

Minimum acceptance includes:

- package CRUD
- task creation from host or group targets
- host execution detail generation on task start
- task result aggregation from host rows
- source-package based offline deployment support
- visible per-host execution traces in task detail
- permission separation across navigation, page, detail, and action scopes
- no direct table-level coupling to CMDB internals
- i18n-complete user feedback and error handling
- a real-host closed loop has been validated for `uninstall -> reinstall`, including CMDB installed-component write-back and status transitions that now respect multi-component hosts (`online -> assigned` only when the last component is removed)

The live closed-loop verification script now adapts to current host state: it runs `uninstall -> reinstall` only when the target component is already installed, and skips directly to `install` for newly onboarded hosts.

Use the Chinese source document for the full field-level schema, request payloads, UI flow details, and roadmap assumptions.
