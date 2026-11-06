# Compat to Multi-Tenant Upgrade Guide

Updated: 2026-11-05. Authority: [Tenant Contract V1](../contracts/TENANT_CONTRACT_V1.md), [resource scope matrix](../designs/TENANT_RESOURCE_SCOPE_MATRIX.md), [initialization guide](../TENANT_INITIALIZATION_GUIDE.md).

The default is `compat`. `multi` adds tenant scope while retaining platform/global resources. `tenant_id = 0` is a platform/global scope, not a row that must be moved into a default tenant. Existing platform operators may remain tenant-less. Membership alone grants no role or permission.

## Before the change

1. Record the exact release, clean migration version and schema backup. From `backend`, with `PANTHEON_DSN` configured, run `go run ./cmd/tenantmigration status`. Apply the reviewed release migration chain separately; foundation migration `000031` has its own [production preflight](../runbooks/MIGRATION_000031_PRODUCTION_DROP.md).
2. Read `GET /api/v1/system/setting/public`; the envelope's `data.settings["platform.tenant_mode"]` contains the plain string `compat` or `multi`. Configuration is stored in `system_setting`, not controlled by editing `config/application.yml`.
3. Classify the existing records using the resource scope matrix. Keep platform users, global menus/permissions and platform configuration in their global scope. Do not bulk update every `tenant_id = 0` row or assign every platform user a membership.
4. Provision target tenants through `POST /api/v1/system/tenant`, including their initial administrator. After creating the tenant row, provisioning creates the owning user, tenant roles/bindings, owner membership and policy/default resources in its own transaction. Provisioning failure triggers compensating tenant-row deletion; compensation failure is logged. See the initialization guide for fields and authentication.
5. If existing business data or users must move, prepare a separate reviewed data migration that preserves ownership, relationships and authorization. The V1 business-domain contract is single-tenant; changing a setting does not isolate those tables.
6. Rehearse the change and backup restore in an isolated copy. Acceptance includes tenant login and system-domain CRUD, negative cross-tenant requests, export/upload namespaces, refresh/revocation, tenant lifecycle and platform management.

## Enable multi through the supported setting API

Use an authenticated platform operator and the existing secondary operation verification. The setting group route preserves validation, cache synchronization and audit.

```http
PUT /api/v1/system/setting/group/platform
Content-Type: application/json
Cookie: <authenticated session cookies>
X-CSRF-Token: <session csrf token>
X-Operation-Token: <token from POST /api/v1/auth/operation-verify>

{"items":[{"settingKey":"platform.tenant_mode","settingValue":"multi"}]}
```

Check the common envelope's `code = 200`, then re-read `/api/v1/system/setting/public` and verify the plain value `multi`. The value must not contain extra JSON quote characters such as `"\"multi\""`.

## Verify the actual behavior

- Sign in through `POST /api/v1/auth/login` using the tenant administrator's credentials. Current UI signs in directly and does not expose the historical pre-login tenant picker. Session credentials are Redis opaque tokens carried in cookies, not JWTs returned in a `token` body.
- Read `GET /api/v1/auth/tenants/current` and `GET /api/v1/auth/tenants/switchable`; role-bearing, active membership is required for an authorized switch through `POST /api/v1/auth/tenants/switch`.
- Use the platform operator for `/api/v1/system/tenant/list` and tenant operations. Use tenant-bound sessions for their system resources. Test both positive operations and foreign IDs/headers/export/upload rejection.
- On a dedicated test instance in multi mode, run `npm run test:smoke:tenant` from `frontend`. Run the full backend race with real isolated MySQL/Redis and required fixture flags. A default compat instance does not satisfy this suite's multi-mode prerequisite.

## Recovery

Once multi-mode writes exist, changing only the flag back to compat is not a valid rollback. The settings service refuses downgrade while live tenants exist; bypassing it with SQL can expose incorrectly scoped data.

If rollback is required, stop all writers and restore a verified pre-upgrade database snapshot together with the matching application/configuration state; handle Redis sessions and caches through the release recovery procedure before admitting users. Validate scope and authentication after restoration. Recovery time depends on data volume and the rehearsed restore, so this guide makes no one-minute rollback promise.

No production conversion or restore is performed by this document or by the isolated smoke commands.
