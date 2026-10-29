# Tenant Management API Documentation

Version: 1.2 (2026-10-25: adds the platform tenant-ops console endpoints §10–§14)  
Base URL: `/api/v1`

## Overview

The Tenant Management API provides endpoints for managing multi-tenant resources in Pantheon Base.

**Mounted surface.** The console endpoints live under `/api/v1/system/tenant*` and are mounted by
`backend/modules/system/system_modules.go` on a group that carries
`TokenAuthMiddleware` + `PlatformScopeTenantContextMiddleware` + `CasbinMiddleware` +
`PlatformCapabilityMiddleware`. They are platform-only: a tenant administrator (or any subject
without a platform-level grant) is rejected with `403 tenant.platformOnly` even when a request
reaches the handler, and Casbin still requires `system:tenant:*` on top.

The earlier `/api/v1/tenants*` group in `tenant_routes.go` is **not mounted**; do not integrate
against it.

**Envelope.** Every response uses the platform envelope `{ "code": 200, "data": ..., "message": "success" }`.
The payloads below describe `data`; errors carry a non-200 `code` and an i18n `message` key.

## Authentication

All endpoints require authentication via Bearer token:

```
Authorization: Bearer <your-jwt-token>
```

## Endpoints

### 1. Create Tenant

**POST** `/system/tenant`

Creates a new tenant.

**Authorization**: Platform capability required (`PlatformCapabilityMiddleware` + `system:tenant:*`)

**Request Body**:
```json
{
  "code": "acme-corp",
  "name": "ACME Corporation",
  "plan": "enterprise",
  "metadata": "{\"industry\":\"technology\"}"
}
```

**Response** (200 OK):
```json
{
  "code": 200,
  "data": {
    "id": 1,
    "code": "acme-corp",
    "name": "ACME Corporation",
    "status": "active",
    "plan": "enterprise",
    "metadata": "{\"industry\":\"technology\"}",
    "created_at": "2026-09-29T10:00:00Z",
    "updated_at": "2026-09-29T10:00:00Z"
  },
  "message": "success"
}
```

**Errors**:
- `400 Bad Request`: Invalid input
- Envelope `code` `400` + `tenant.code_taken`: the tenant code is already held by a live tenant
- Envelope `code` `400` + `tenant.code_reserved`: the code is held by a soft-deleted tenant

---

### 2. List Tenants

**GET** `/system/tenant/list`

Lists all tenants with pagination and filtering.

**Authorization**: Platform capability required (`PlatformCapabilityMiddleware` + `system:tenant:*`)

**Query Parameters**:
- `status` (optional): Filter by status (`active`, `suspended`, `deleted`)
- `plan` (optional): Filter by plan (`free`, `basic`, `professional`, `enterprise`)
- `page` (optional): Page number (default: 1)
- `size` (optional): Page size (default: 20, max: 100)

**Response** (200 OK):
```json
{
  "items": [
    {
      "id": 1,
      "code": "acme-corp",
      "name": "ACME Corporation",
      "status": "active",
      "plan": "enterprise",
      "created_at": "2026-09-29T10:00:00Z",
      "updated_at": "2026-09-29T10:00:00Z"
    }
  ],
  "total": 50,
  "page": 1,
  "size": 20
}
```

---

### 3. Get Tenant

**GET** `/system/tenant/:id`

Retrieves a tenant by ID.

**Authorization**: Platform capability required (`PlatformCapabilityMiddleware` + `system:tenant:*`)

**Response** (200 OK):
```json
{
  "id": 1,
  "code": "acme-corp",
  "name": "ACME Corporation",
  "status": "active",
  "plan": "enterprise",
  "created_at": "2026-09-29T10:00:00Z",
  "updated_at": "2026-09-29T10:00:00Z"
}
```

**Errors**:
- `404 Not Found`: Tenant not found

---

### 4. Update Tenant

**PUT** `/system/tenant/:id`

Updates tenant properties.

**Authorization**: Platform capability required (`PlatformCapabilityMiddleware` + `system:tenant:*`)

**Request Body**:
```json
{
  "name": "ACME Corporation Ltd",
  "status": "suspended",
  "plan": "professional"
}
```

All fields are optional. Only provided fields will be updated.

**Response** (200 OK):
```json
{
  "code": 200,
  "data": {},
  "message": "success"
}
```

**Errors**:
- `404 Not Found`: Tenant not found

---

### 5. Delete Tenant

**DELETE** `/system/tenant/:id`

Marks a tenant as deleted (soft delete).

**Authorization**: Platform capability required (`PlatformCapabilityMiddleware` + `system:tenant:*`)

**Response** (200 OK):
```json
{
  "code": 200,
  "data": {},
  "message": "success"
}
```

**Errors**:
- `400 Bad Request` + `tenant.not_archived`: the tenant is not archived — deletion is reserved
  for archived tenants (contract §2.1)
- `404 Not Found`: Tenant not found

**Note**: Deletion is the final step of the documented retirement path, not a
member-management action. The guard is the tenant's status, not its member count: an active
tenant is refused even when it has no active members (a state reachable by deleting every
user, which disables their memberships). Because the last owner can never be removed (§2.2),
set the tenant to `archived` first and then delete it. Deleting an archived tenant disables
its remaining memberships in the same transaction.

---

### 6. Add Tenant Member

**POST** `/system/tenant/:id/members`

Adds a user to a tenant.

**Authorization**: Platform capability required (`PlatformCapabilityMiddleware` + `system:tenant:*`)

**Request Body**:
```json
{
  "user_id": 100,
  "role": "admin"
}
```

**Roles**:
- `owner`: Full control over tenant
- `admin`: Administrative access
- `member`: Standard member access

**Response** (200 OK):
```json
{
  "code": 200,
  "data": {},
  "message": "success"
}
```

**Errors**:
- Envelope `code` `404` + `tenant.not_found`: the tenant id is unknown or the tenant is a deleted
  tombstone. This guard runs first, so a tombstone answers with the tenant error even when
  `user_id` is also unknown.
- Envelope `code` `404` + `user.not_found`: `user_id` does not name a `system_user` row that still
  exists (`deleted_at IS NULL`). `tenant_memberships` has no foreign key to `system_user`, so without
  this guard a typo in the console created a membership row naming nobody — and a row written as
  `owner` for a nonexistent user satisfied the last-owner invariant for a user who can never sign in.
  A disabled user is still addable.
- Envelope `code` `400` + `tenant.member.user_not_in_tenant`: the user is owned by another tenant
  (`system_user.tenant_id` is not the target tenant). Role bindings are stamped from the owning
  tenant, so a cross-tenant membership would grant login eligibility with zero roles — an unusable
  context (tenant contract §2.2, V1.7). This closes the defect where such a row was accepted and the
  login picker then offered a tenant that rendered an empty console.
- Envelope `code` `400` + `tenant.member_exists`: the user is already an `owner` and a non-owner role
  was submitted (re-adding is idempotent for non-owner rows)

---

### 7. List Tenant Members

**GET** `/system/tenant/:id/members`

Lists one bounded page of active tenant members, ordered by creation time and ID. The response envelope is stable for all consumers. This is the only mounted member-list route: no `/members/page` variant exists (the legacy route group that carried it is not registered).

**Authorization**: Platform capability required (`PlatformCapabilityMiddleware` + `system:tenant:*`)

**Query parameters**: `page` (default 1, minimum 1), `pageSize` (default 20, maximum 100). Invalid or non-positive values use the defaults; larger page sizes are capped at 100.

**Response** (200 OK):
```json
{
  "items": [
    {
      "id": 1,
      "tenant_id": 1,
      "user_id": 100,
      "role": "owner",
      "status": "active",
      "created_at": "2026-09-29T10:00:00Z",
      "updated_at": "2026-09-29T10:00:00Z"
    }
  ],
  "total": 1,
  "page": 1,
  "pageSize": 20
}
```

---

### 8. Remove Tenant Member

**DELETE** `/system/tenant/:id/members/:userId`

Removes a user from a tenant.

**Authorization**: Platform capability required (`PlatformCapabilityMiddleware` + `system:tenant:*`)

**Response** (200 OK):
```json
{
  "message": "member removed successfully"
}
```

**Errors**:
- `404 Not Found`: Membership not found

---

### 9. Get My Tenants

**GET** `/auth/tenants/switchable`

Returns the tenants the current authenticated user may switch into: active memberships in active
tenants **where the user holds at least one role binding** (tenant contract §2.2, V1.7 — a
membership without roles is not a usable context). This is the mounted equivalent of the legacy
`GET /tenants/my`; the legacy group is not registered.

**Compatibility surface (V1.7)**: the login tenant picker and the shell TenantSwitcher control were
removed because a subject can hold role bindings in at most one tenant (its owning one), so this
list is filtered down to at most one item and the controls could never render. The endpoint stays
for API compatibility: a client that still calls it gets the filtered list, not an error. The same
predicate filters `GET /auth/login-tenants` (the login candidate list).

**Authorization**: Requires authentication (any authenticated user)

**Response** (200 OK):
```json
[
  {
    "id": 1,
    "code": "acme-corp",
    "name": "ACME Corporation",
    "status": "active",
    "plan": "enterprise",
    "created_at": "2026-09-29T10:00:00Z",
    "updated_at": "2026-09-29T10:00:00Z"
  }
]
```

---

## Tenant-ops console (platform-global rollups)

The pages behind **租户运营** read from a sibling group, `/system/tenant-ops/*`, on purpose:
`GET /system/tenant/:id` already owns that group's wildcard child, so a static
`/system/tenant/overview` would be a routing-shape risk. Contract: `TENANT_CONTRACT_V1` §2.5.

These endpoints span tenants by design, so they sit behind the same
`PlatformCapabilityMiddleware` group and are platform-only. Quotas reported here are a **measurement,
not a write gate** (§2.4.1): exceeding a plan limit never causes a write to be rejected.

### 10. Tenant Overview

**GET** `/system/tenant-ops/overview`

**Authorization**: Platform capability (`system:tenant:overview`)

**Query**: `keyword`, `status` (`active|suspended|archived|deleted`), `plan`, `page`, `size`
(default 20; the 100 ceiling is enforced by request binding, so a larger value returns
`param.invalid` rather than a truncated page).

**Response** (200 OK):
```json
{
  "summary": { "total": 3, "active": 2, "suspended": 1, "archived": 0, "deleted": 1 },
  "totals": { "members": 3, "roles": 2, "depts": 1 },
  "plans": [
    { "plan": "free", "name": "Free", "known": true, "tenantCount": 1,
      "quota": { "max_users": 5, "max_roles": 3, "max_depts": 5, "max_api_keys": 2, "max_storage_mb": 100 } }
  ],
  "items": [
    { "tenantId": 101, "code": "acme", "name": "ACME", "status": "active", "plan": "free",
      "planName": "Free", "members": 2, "roles": 2, "depts": 0,
      "maxUsers": 5, "maxRoles": 3, "maxDepts": 5,
      "createdAt": "2026-10-25T09:00:00Z", "updatedAt": "2026-10-25T09:00:00Z" }
  ],
  "total": 3,
  "page": 1,
  "size": 20
}
```

Notes:
- `summary` is a facet set: it applies every filter except `status`, and `total` there is
  `active + suspended + archived` (the `deleted` tombstone is reported separately, per §2.1).
- A `-1` in any `max_*` means **no ceiling**; it is not a very large number and must not be rendered as
  a ratio.
- An unknown or empty `plan` reports `known: false` with an empty `planName` and the fallback quota. It
  is shown as itself — never silently relabelled `free`.
- Occupancy counts active memberships only, respects soft delete for roles and departments, and excludes
  platform rows (`tenant_id = 0`).

### 11. Members Across Tenants

**GET** `/system/tenant-ops/member`

**Authorization**: Platform capability (`system:tenant:member-list`)

**Query**: `tenantId`, `status` (`active|disabled`), `role` (`owner|admin|member`), `keyword`, `page`,
`pageSize` (default 20, max 100). Default status filter is `active`.

**Response** (200 OK):
```json
{
  "items": [
    { "tenantId": 101, "tenantCode": "acme", "tenantName": "ACME", "tenantStatus": "active",
      "plan": "free", "planName": "Free", "maxUsers": 5, "tenantUsage": 2,
      "userId": 201, "username": "ada", "nickname": "Ada", "memberRole": "owner",
      "memberStatus": "active", "joinedAt": "2026-10-25T09:00:00Z" }
  ],
  "total": 3,
  "page": 1,
  "pageSize": 20
}
```

`tenantUsage` is the whole tenant's active-member count, so a row shows its own person and the ceiling
they share.

### 12. List Plans

**GET** `/system/tenant-ops/plan/list`

**Authorization**: Platform capability (`system:tenant:plan:list`)

**Response** (200 OK):
```json
{
  "items": [
    { "id": 1, "code": "free", "name": "Free", "maxUsers": 5, "maxRoles": 3, "maxDepts": 5,
      "maxApiKeys": 2, "maxStorageMb": 100, "status": "active", "sort": 10,
      "createdAt": "2026-10-25T09:00:00Z", "updatedAt": "2026-10-25T09:00:00Z", "tenantCount": 1 }
  ]
}
```

Disabled plans are included; `tenantCount` is the number of non-deleted tenants carrying the code, which
is exactly the number that decides whether the plan can be retired.

### 13. Create Plan

**POST** `/system/tenant-ops/plan`

**Authorization**: Platform capability (`system:tenant:plan:create`). Audited (`plan_create`).

**Request Body** — all five ceilings are required on create:
```json
{
  "code": "starter",
  "name": "Starter",
  "status": "active",
  "sort": 15,
  "maxUsers": 10,
  "maxRoles": 5,
  "maxDepts": 10,
  "maxApiKeys": 5,
  "maxStorageMb": 500
}
```

**Errors**: `tenant.plan.code_invalid` (empty or outside `^[a-z][a-z0-9_-]{1,31}$`),
`tenant.plan.quota_required` (a ceiling is missing), `tenant.plan.quota_invalid` (`0` or any value below
`-1`), `tenant.plan.code_taken` (the natural key is used). All report as caller errors, not server faults.

`-1` is accepted and means unlimited. `0` is deliberately rejected: consumers must tell "no ceiling"
apart from "ceiling reached".

### 14. Update Plan

**PUT** `/system/tenant-ops/plan/:code`

**Authorization**: Platform capability (`system:tenant:plan:update`). Audited (`plan_update`).

Partial update: an omitted field keeps its stored value, but if any ceiling is supplied all five must be
(they are one editing unit — a half-set of limits is not reportable).

**Errors**: `tenant.plan.name_required`, `tenant.plan.quota_required`, `tenant.plan.quota_invalid`,
`tenant.plan.not_found` (unknown code), and `tenant.plan.in_use` — setting `status = "disabled"` is
refused while non-deleted tenants still carry the code.

Plan deletion is not modelled — retirement is `status = "disabled"`.

---

## Usage Examples

### Create a new tenant

```bash
curl -X POST http://localhost:8080/api/v1/system/tenant \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "code": "acme-corp",
    "name": "ACME Corporation",
    "plan": "enterprise"
  }'
```

### Add admin user to tenant

```bash
curl -X POST http://localhost:8080/api/v1/system/tenant/1/members \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "user_id": 100,
    "role": "owner"
  }'
```

### List my tenants

```bash
curl -X GET http://localhost:8080/api/v1/auth/tenants/switchable \
  -H "Authorization: Bearer <token>"
```

---

## Error Responses

Errors use the same envelope, with a non-200 `code` and an i18n `message` key:

```json
{
  "code": 403,
  "data": null,
  "message": "tenant.platformOnly"
}
```

Codes seen on this surface:

| HTTP | `code` | `message` | When |
| --- | --- | --- | --- |
| 200 | 200 | `success` | OK |
| 200 | 400 | `param.invalid` | Binding/validation failure (also used for state conflicts; the envelope has no 409) |
| 200 | 400 | `tenant.member.last_owner` | Removing the last active owner (contract §2.2) |
| 200 | 400 | `tenant.member.user_not_in_tenant` | Add-member submitted a user owned by another tenant (contract §2.2 V1.7) |
| 200 | 400 | `tenant.status.terminal` | Trying to move an archived tenant (contract §2.1) |
| 200 | 403 | `tenant.platformOnly` | Non-platform subject on the tenant console |
| 200 | 403 | `tenant.forbidden` | Foreign tenant id in the body / untrusted `X-Tenant-Id` |
| 200 | 404 | `tenant.not_found` | Unknown tenant or membership |
| 200 | 404 | `user.not_found` | Add-member submitted an id that names no existing user |

> The transport status stays `200` for business failures; always branch on the envelope `code`,
> not on the HTTP status. This is the platform-wide convention, not tenant-specific.
- `500 Internal Server Error`: Server error
