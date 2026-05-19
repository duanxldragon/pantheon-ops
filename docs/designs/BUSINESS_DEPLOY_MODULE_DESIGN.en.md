# Deploy Business Module Design

Chinese version: [BUSINESS_DEPLOY_MODULE_DESIGN.md](./BUSINESS_DEPLOY_MODULE_DESIGN.md)

Updated: 2026-05-19

Type: Design  
Layer: `business/deploy`  
Status: Active

This document defines the Deploy module for `pantheon-ops`. It is the second business module after CMDB and establishes task-based software deployment over CMDB hosts and groups.

## Scope

The module owns:

- deployable software packages
- deployment tasks
- per-host execution detail records

The module does not own:

- CMDB inventory maintenance
- credential vaulting
- realtime monitoring
- agent runtime implementation in v1
- system-domain IAM implementation

## Boundary Model

Deploy is not allowed to read `biz_cmdb_*` tables directly.

It must consume CMDB through explicit read-only capabilities such as:

- `ListSelectableHosts`
- `ListSelectableGroups`
- `PreviewGroupMembers`
- `ResolveTaskTargets`

All cross-module reads must still obey `DataScopeReq`.

## Core Objects

- `DeployPackage`
- `DeployTask`
- `DeployTaskHost`

Primary tables:

- `biz_deploy_package`
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

Version-one execution modes:

- `manual`
- `simulated`

Later reserved modes:

- `agent`
- `ssh`

## APIs

Prefix: `/api/v1/business/deploy`

Main resources:

- packages: list, create, update, delete
- tasks: list, create, detail, update, start, cancel
- task hosts: mark result, reserved agent report endpoint

Permissions follow:

- `business:deploy:package:*`
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
- `DeployPackageForm`
- `DeployTaskList`
- `DeployTaskForm`
- `DeployTaskDetail`

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
- permission separation across navigation, page, detail, and action scopes
- no direct table-level coupling to CMDB internals
- i18n-complete user feedback and error handling

Use the Chinese source document for the full field-level schema, request payloads, UI flow details, and roadmap assumptions.
