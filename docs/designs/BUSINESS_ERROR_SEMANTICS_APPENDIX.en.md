# Ops Business Error Semantics Appendix

Chinese version: [BUSINESS_ERROR_SEMANTICS_APPENDIX.md](./BUSINESS_ERROR_SEMANTICS_APPENDIX.md)

Updated: 2026-07-18

Type: Design
Layer: `business/*`
Status: Active

This appendix is the repository-level canonical business error-key reference for `pantheon-ops`. Backend responses, frontend i18n, API integration, acceptance, and audit detail must use the same semantic vocabulary.

---

## 1. Rules

### 1.1 Canonical Naming

Business error keys use:

```text
business.<module>.<resource>.<reason>
```

`module` and `resource` identify the business domain; `reason` uses lowerCamelCase. Do not add legacy short namespaces such as `cmdbhost.*`, `cmdbgroup.*`, or `cmdblabel.*`.

### 1.2 Responsibility Split

- backend returns stable error keys
- frontend translates keys to localized text
- audit detail, import/export results, banners, and toasts resolve by key instead of hardcoded natural language

### 1.3 HTTP Guidance

| Scenario | Suggested HTTP | Example semantics |
| :--- | :--- | :--- |
| missing or malformed input | `400` | `*.nameRequired`, `*.invalidLabel` |
| forbidden | `403` | `permission.denied` or `*.forbidden` |
| missing resource | `404` | `*.notFound` |
| uniqueness or state conflict | `409` | `*.ipExists`, `*.invalidStartState` |
| dependency or execution failure | `422` or `500` | `*.sshAuthFailed`, `*.executionFailed` |

Generic Base keys such as `param.invalid`, `permission.denied`, and `request.failed` remain valid when no business-specific meaning is required.

---

## 2. CMDB Canonical List

Source: `BUSINESS_CMDB_MODULE_DESIGN.md`.

### 2.1 Host

| Key | Meaning |
| :--- | :--- |
| `business.cmdb.host.notFound` | host not found |
| `business.cmdb.host.ipExists` | IP already exists |
| `business.cmdb.host.invalidLabel` | invalid label format |
| `business.cmdb.host.unsupportedOs` | unsupported OS |

### 2.2 Collect

| Key | Meaning |
| :--- | :--- |
| `business.cmdb.collect.sshConnectFailed` | SSH connection failed |
| `business.cmdb.collect.sshAuthFailed` | SSH authentication failed |
| `business.cmdb.collect.executionFailed` | collection command failed |

### 2.3 Group

| Key | Meaning |
| :--- | :--- |
| `business.cmdb.group.notFound` | group not found |
| `business.cmdb.group.invalidConditions` | invalid filter expression |

### 2.4 Label

| Key | Meaning |
| :--- | :--- |
| `business.cmdb.label.keyExists` | label key already exists |
| `business.cmdb.label.invalid` | invalid label schema |
| `business.cmdb.label.inUse` | label is referenced by a host or group |
| `business.cmdb.label.notFound` | label schema not found |

---

## 3. Deploy Canonical List

Source: `BUSINESS_DEPLOY_MODULE_DESIGN.md`.

### 3.1 Package

| Key | Meaning |
| :--- | :--- |
| `business.deploy.package.nameRequired` | package name is required |
| `business.deploy.package.versionRequired` | package version is required |
| `business.deploy.package.nameVersionConflict` | package name/version conflicts |
| `business.deploy.package.commandTooLong` | command template is too long |

### 3.2 Task

| Key | Meaning |
| :--- | :--- |
| `business.deploy.task.notFound` | task not found |
| `business.deploy.task.forbidden` | current user cannot view or operate the task |
| `business.deploy.task.nameRequired` | task name is required |
| `business.deploy.task.packageRequired` | package is required |
| `business.deploy.task.packageDisabled` | package is disabled |
| `business.deploy.task.scopeRequired` | host deployment requires a business scope |
| `business.deploy.task.scopeInvalid` | business scope is invalid |
| `business.deploy.task.targetRequired` | target host or group is required |
| `business.deploy.task.invalidTargetType` | target type is invalid |
| `business.deploy.task.targetOutOfScope` | target is outside the current data scope |
| `business.deploy.task.invalidExecutorType` | executor type is invalid |
| `business.deploy.task.invalidAction` | task action is invalid |
| `business.deploy.task.targetStatusMismatch` | target status does not allow the action |
| `business.deploy.task.templateNotFound` | referenced template not found |
| `business.deploy.task.templateDisabled` | referenced template is disabled |
| `business.deploy.task.packageNotFound` | referenced package not found |
| `business.deploy.task.invalidStartState` | current state cannot start |
| `business.deploy.task.invalidUpdateState` | current state cannot be updated |
| `business.deploy.task.invalidDeleteState` | current state cannot be deleted |
| `business.deploy.task.emptyResolvedTargets` | target resolution returned no executable hosts |
| `business.deploy.task.invalidCancelState` | current state cannot be canceled |
| `business.deploy.task.templateParamsInvalid` | template parameters or variables are invalid |
| `business.deploy.task.templateInvalid` | fixed template definition is invalid |
| `business.deploy.task.installCommandRequired` | install action has no install command |
| `business.deploy.task.uninstallCommandRequired` | uninstall action has no uninstall command |
| `business.deploy.task.packageSourceMissing` | fixed-template package source is missing |
| `business.deploy.task.sshHostKeyRequired` | SSH host fingerprint is required |
| `business.deploy.task.sshHostKeyMismatch` | SSH host fingerprint mismatch |
| `business.deploy.task.sshUserRequired` | SSH user is required |
| `business.deploy.task.sshPasswordRequired` | password authentication requires a password |
| `business.deploy.task.sshPrivateKeyRequired` | private-key authentication requires a key |
| `business.deploy.task.sshAuthFailed` | SSH authentication failed |
| `business.deploy.task.sshConnectFailed` | SSH connection failed |

### 3.3 TaskHost

| Key | Meaning |
| :--- | :--- |
| `business.deploy.taskHost.notFound` | host execution detail not found |
| `business.deploy.taskHost.invalidResultState` | current state cannot be marked |
| `business.deploy.taskHost.markFailed.reasonRequired` | failure marking requires a reason |

### 3.4 Template

Deploy Template CRUD is implemented, but a canonical error-key list for the Template resource has not yet been defined. This is an explicit placeholder, not permission to invent keys in this task. Define keys here first when template-specific business errors are introduced, then synchronize the module design and i18n resources.

---

## 4. BizScope Migrated State

Source: `BUSINESS_BIZSCOPE_MODULE_DESIGN.md`.

### 4.1 Canonical List

| Key | Meaning | Source |
| :--- | :--- | :--- |
| `business.bizscope.codeExists` | duplicate business-scope code | Create / Update uniqueness validation |
| `business.bizscope.inUse` | scope still has bound hosts | Delete reference protection |
| `business.bizscope.notFound` | scope, binding target, or in-scope host not found | Detail / Hosts / Bind / Unbind / Update / Delete |
| `param.invalid` | invalid request | generic Base parameter key |
| `request.failed` | query, database, or framework failure without business meaning | generic Base request-failure key |

`backend/modules/business/bizscope` no longer returns `bizscope.code_exists`, `bizscope.in_use`, or `bizscope.not_found`.

### 4.2 Handler Pipeline-Key Migration

| Retired handler key | Migrated handling |
| :--- | :--- |
| `bizscope.list.error` | no distinct business meaning; falls back to `request.failed` |
| `bizscope.options.error` | no distinct business meaning; falls back to `request.failed` |
| `bizscope.detail.error` | returns `business.bizscope.notFound` for a missing resource; otherwise falls back to `request.failed` |
| `bizscope.hosts.error` | returns `business.bizscope.notFound` for a missing scope; otherwise falls back to `request.failed` |
| `bizscope.availableHosts.error` | returns `business.bizscope.notFound` for a missing scope; otherwise falls back to `request.failed` |
| `bizscope.bindHosts.error` | returns `business.bizscope.notFound` for a missing scope or in-scope host and `param.invalid` for empty hostIds; otherwise falls back to `request.failed` |
| `bizscope.unbindHost.error` | returns `business.bizscope.notFound` for a missing scope, host, or binding; otherwise falls back to `request.failed` |
| `bizscope.create.error` | returns `business.bizscope.codeExists` for a code conflict; otherwise falls back to `request.failed` |
| `bizscope.update.error` | returns `business.bizscope.notFound` for a missing resource and `business.bizscope.codeExists` for a code conflict; otherwise falls back to `request.failed` |
| `bizscope.delete.error` | returns `business.bizscope.inUse` while hosts are bound and `business.bizscope.notFound` for a missing resource; otherwise falls back to `request.failed` |

### 4.3 Legacy-Key Fallback Decision

Decision: **remove the legacy keys directly; do not retain a one-version fallback mapping**.

The frontend request layer only validates that backend `message` looks like an i18n key and then performs an exact translation lookup. It has no business branch for the old BizScope short keys. A missing translation displays `request.failed` plus the raw key in development and only `request.failed` in production. Retaining aliases would therefore extend a dual-key lifecycle without required compatibility. The zh-CN / en-US module locales and backend seed now use canonical keys, and the seed removes the three legacy runtime rows owned by `business.bizscope`.

---

## 5. Requirements for New Modules

Every new `business/*` design must define its namespace, resource-level canonical list, validation errors, state-transition errors, permission/data-scope errors, and external-dependency failures before implementation.

---

## 6. Change Rules

- update this appendix first when adding, renaming, or retiring a business error key
- module error sections are local summaries; this appendix is the ops repository canonical list
- `pantheon-base/docs/designs/ERROR_CODE_AND_I18N.md` remains the higher-level rule source for generic error handling
