# Ops Business Error Semantics Appendix

Chinese version: [BUSINESS_ERROR_SEMANTICS_APPENDIX.md](./BUSINESS_ERROR_SEMANTICS_APPENDIX.md)

Updated: 2026-05-19

Type: Design  
Layer: `business/*`  
Status: Active

This appendix is the canonical business error-key reference for `pantheon-ops`.

Its purpose is to keep backend error keys, frontend i18n, API integration, acceptance checks, and audit detail rendering on the same semantic vocabulary.

## Naming Rule

Business error keys must use:

```text
business.<module>.<resource>.<reason>
```

Examples:

- `business.cmdb.host.notFound`
- `business.cmdb.collect.sshAuthFailed`
- `business.deploy.task.invalidStartState`
- `business.deploy.taskHost.markFailed.reasonRequired`

Rules:

- `module` is the business module such as `cmdb` or `deploy`
- `resource` is the resource or interaction domain
- `reason` uses lowerCamelCase
- legacy short keys such as `cmdbhost.*` must not be extended further

## Responsibility Split

- backend returns stable error keys
- frontend translates error keys to localized text
- audit detail pages, import/export feedback, banners, and toasts should resolve by key instead of hardcoded natural language

## HTTP Guidance

- `400`: missing or malformed input
- `403`: forbidden access
- `404`: missing resource
- `409`: uniqueness or state conflict
- `422` or `500`: external dependency or execution failure

Platform-generic keys from `pantheon-base` still apply where no business-specific semantic is needed.

## CMDB Canonical Sets

Namespaces:

- `business.cmdb.host.*`
- `business.cmdb.collect.*`
- `business.cmdb.group.*`
- `business.cmdb.label.*`

Representative keys:

- `business.cmdb.host.ipExists`
- `business.cmdb.collect.executionFailed`
- `business.cmdb.group.invalidConditions`
- `business.cmdb.label.inUse`

## Deploy Canonical Sets

Namespaces:

- `business.deploy.package.*`
- `business.deploy.task.*`
- `business.deploy.taskHost.*`

Representative keys:

- `business.deploy.package.nameVersionConflict`
- `business.deploy.task.packageDisabled`
- `business.deploy.task.invalidCancelState`
- `business.deploy.taskHost.invalidResultState`

## Rule For New Business Modules

Any future `business/*` design must define, before implementation:

- its error-key namespace
- a resource-level canonical list
- validation errors
- state-transition errors
- permission or data-scope errors
- external dependency failures

If the appendix and a module-level design ever diverge, this appendix is the repo-level canonical reference and `pantheon-base` remains the higher-level rule source for generic error handling.
