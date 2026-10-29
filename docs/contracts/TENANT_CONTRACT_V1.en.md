---
title: Tenant Contract V1 (Shared-Schema MVP)
doc_type: Contract
layer: platform / system/auth / system/iam / system/org / system/config
status: Approved
updated_at: 2026-10-26
---

# Tenant Contract V1 (Shared-Schema MVP)

Chinese version: [TENANT_CONTRACT_V1.md](./TENANT_CONTRACT_V1.md)

This contract is the frozen deliverable of the `2026-09-10-tenant-contract-design` task and provides the execution contract for the shared-schema MVP phase of the [P2 scale roadmap](../designs/P2_SCALE_ROADMAP.md). Design rationale: [tenant-ready single-tenant design](../designs/TENANT_READY_SINGLE_TENANT_DESIGN.md).

“Frozen” = the V1 **model, ownership boundaries, and invariants** are locked; concrete table-schema evolution is governed by the runbook (`2026-09-10-tenant-migration-runbook`).

---

## 1. Scope (In / Out)

### In

- Tenant master-data model and lifecycle (active / suspended / archived)
- TenantMembership: many-to-many user-tenant affiliation with in-tenant roles
- Tenant context: request-scoped tenant resolution, injection, propagation, validation
- Resource tenancy: business/system resources owned via `tenant_id`, constrained by the resource-scope matrix
- Casbin domain-based permission isolation (policies carry a tenant domain)
- Compat mode: a single-tenant compatibility switch so existing deployments keep running with zero migration
- Plan definitions and quota-threshold **storage plus read-side display** (new in V1.4, §2.4), and the platform-side tenant-operations **cross-tenant aggregate read paths** (§2.5)

### Out

- Schema-per-tenant / database-per-tenant isolation (V1 is shared schema + `tenant_id` only)
- Commercial tenant billing and plan sales (ordering, upgrades/downgrades, billing periods — separate
  commercial workstream). Boundary note: the plan **definitions** and quota **thresholds** are DB-backed
  and console-editable since V1.4 (§2.4), but a quota is a measurement and a display dimension only —
  **the backend never rejects a write because a quota was exceeded**.
- Cross-tenant data sharing / federated queries
- Tenant custom domains, per-tenant deployment shapes

---

## 2. Core Model (frozen)

### 2.1 Tenant master data

```
tenants
  id            bigint PK          -- snowflake ID from the platform ID generator
  code          varchar(64)  UK    -- globally unique, immutable after creation
  name          varchar(128)
  status        varchar(16)        -- active | suspended | archived | deleted (tombstone, see invariants)
  plan          varchar(32)        -- plan marker: resolves to the §2.4 quota thresholds (measurement/display only)
  created_at / updated_at
  deleted_at                       -- reserved column, not written by the current implementation
```

Invariants:
- `code` is immutable after creation; matches `^[a-z][a-z0-9-]{1,62}$`
- Status transitions: `active ↔ suspended`, and `active|suspended → archived`; `archived` is terminal
- `archived` tenants must not log in, must not receive new tokens, must not produce new business writes
- Deletion is allowed only for `archived` tenants and is an ops action, never exposed on business APIs
- `deleted` is a **tombstone**, not a lifecycle state (V1.3 clarification): it is written only by the
  delete action, only after `archived`, in the same transaction that releases `code`. It has no
  outgoing transition, **must not be accepted as create or update input**, and never appears in the
  default list (only under an explicit status filter). Addressing a deleted tenant by id or by `code`
  behaves as "not found" (consistent with §5, where missing and foreign are indistinguishable), so a
  deleted tenant cannot be reactivated, cannot gain members and cannot be re-provisioned.
- Deleting an `archived` tenant disables its remaining memberships in the same transaction: §2.2's
  owner requirement constrains **non-archived** tenants only, so after archiving the memberships no
  longer block deletion (V1.3 clarification).
- The delete marker is `status = 'deleted'`, **not** `deleted_at`: that column exists in the schema but
  the current implementation neither writes nor filters on it. Two markers would let "is this deleted?"
  have two answers that can disagree.

### 2.2 TenantMembership

```
tenant_memberships
  id            bigint PK
  tenant_id     bigint       -- FK → tenants.id, part of the composite unique key
  user_id       bigint       -- FK → iam users.id, part of the composite unique key
  role          varchar(32)  -- owner | admin | member (in-tenant role, not a global role)
  status        varchar(16)  -- active | disabled
  created_at / updated_at
  UK(tenant_id, user_id)
```

Invariants:
- A user may belong to multiple tenants; `(tenant_id, user_id)` is unique
- Every non-archived tenant has at least one `owner`; owner transfer is an explicit API action
- Membership removal is **split by path** (V1.5 clarification, verified line by line against the
  implementation; the previous wording "always soft-disable, never a physical delete, to keep the audit
  chain intact" did not match the code):

  | Path | Actual behaviour | Code |
  |---|---|---|
  | Tenant deletion | Soft-disable: inside one transaction the tenant's remaining `active` memberships become `status = disabled`, committed together with `status = 'deleted'` and the released `code` (§2.1) | `tenant_service.go:608-615` |
  | User deletion | Soft-disable: `status = disabled`; a tenant-scoped caller only touches its own rows, a platform-scoped caller clears the user's memberships everywhere (§2.3.1 rule 2) | `user/user_membership.go:78-88`, regression `user_membership_test.go:152-200` |
  | Explicit member removal | **Physical delete** (the only exception today): `ensureNotLastOwner` first, then delete by `(tenant_id, user_id)` regardless of `status` (the model has no `deleted_at` column, `tenant_model.go:25-33`); `RowsAffected == 0` → `ErrMembershipNotFound`; the member's sessions are then revoked best effort | `tenant_service.go:914-940` |
  | Re-join / re-created user | **Revives the old row**: an existing `(tenant_id, user_id)` row is updated to the requested role + `active` instead of inserting a second row | `tenant_service.go:869-888`, `user_membership.go:45-51` |

  Soft-disable is the default direction; the physical delete is the exception that still needs
  converging. The reason for keeping rows is stated in the next bullet.
- Why the rows are kept at all: a membership is the record that "this person once belonged to this
  tenant", and the login gate already treats anything but `active` as a denial
  (`pkg/tenant.GateSessionIssuance`), so disabling is sufficient to cut access. For the physical-delete
  path the audit fact is carried by the **global operation log** instead of by the row:
  `operation_log_middleware.go:224-276` records every non-GET request, and the platform-side add/remove
  handlers additionally set `SetAuditMetadata(..., BusinessGrant/BusinessDelete)`
  (`tenant_platform_handler.go:201-239`). The tenant-side
  `DELETE /tenants/:id/members/:user_id` handler sets no title and relies on the middleware default — an
  improvement worth making, not a contract requirement.
- **Turning member removal into a soft-disable as well is a contract change, not a clarification**, and
  was not done this round (write-side behaviour untouched). Prerequisites registered: ① define the error
  semantics of a repeated removal — the current code distinguishes "no such row" through
  `RowsAffected == 0`, and an `UPDATE ... status = disabled` also returns `RowsAffected = 0` in MySQL for
  an unchanged row, so that signal can no longer be reused for existence; ② the revival path must cover
  every membership insert site (`AddMember`, user create/import on the `user` side, tenant
  provisioning) or they will collide with `UK(tenant_id, user_id)`; ③ the member list, the cross-tenant
  member view (§2.5) and the "at least one owner" count must state explicitly whether `disabled` rows
  are included.
- In-tenant roles (owner/admin/member) and global IAM roles (`iam roles`) are **namespace-separated** with no implicit mapping

### 2.3 Resource tenancy (`tenant_id` propagation)

- Tables that carry tenancy get `tenant_id bigint NOT NULL DEFAULT 0`:
  - `tenant_id = 0` is reserved for **platform-level / global resources** (existing data under compat mode)
  - `tenant_id > 0` marks tenant-private resources
- Which tables carry `tenant_id` and which stay global is registered in the [tenant resource-scope matrix](../designs/TENANT_RESOURCE_SCOPE_MATRIX.md); new tables must register before merge (acceptance matrix T4)
- **Forbidden**: nullable `tenant_id` (ambiguous NULL semantics); foreign keys on tenant code strings; business code querying `WHERE tenant_code = ?` directly

#### 2.3.1 Read and write duties (V1.2 addition, 2026-10-19)

`tenant_id` is part of a row's **identity**, not a decorative column. Every read and write against a table
that carries `tenant_id` must satisfy all three rules below; each one is independently citable by a
reviewer when rejecting a change.

1. **Reads must carry the tenant predicate.** A query must append `tenant_id = ctx.tenant_id` (or have it
   injected through the §3.3 scope helper). Business keys that are "unique within a tenant"
   (`role_key`, `username`, `dept_code`, `post_code`, …) are constrained as `(tenant_id, key)` in the
   database, **not** globally unique. So `WHERE role_key = ?` is already a cross-tenant read — even with
   `LIMIT 1`, and even when the result is only used to echo one field back.
2. **Writes must take the tenant from the context.** `tenant_id` may only come from the request context
   (§3.2), never from the request body, and an upsert's conflict key / `ON CONFLICT` must include
   `tenant_id`. Using the bare key as the conflict key **rewrites a tenant-private row into a
   `tenant_id = 0` platform row**, and platform rows apply to every tenant — the most insidious kind of
   amplification, because the writer only ever sees "saved successfully".
3. **Existence and ownership checks on a bare key count as reads too.** "Does this `role_key` exist?",
   "Are these `dept_id`s mine?" must carry the tenant predicate. Otherwise one tenant can attach a policy
   to another tenant's resource by guessing its key, or write another tenant's master-data id into its
   own configuration (`dept_id`, `post_id` come from a global sequence, so the same number denotes
   different objects in different tenants).

**The only two permitted scopes** (matching the two cross-tenant paths of §3.3):

| Scope | May read/write | Does thereby gain |
|---|---|---|
| Platform scope (`ctx.tenant_id = 0`, requires the §3.4 declaration + the platform capability) | the global rows with `tenant_id = 0` | no power to operate on **tenant-private rows** by bare key |
| Tenant scope (`ctx.tenant_id > 0`) | its own share | no rows of any other tenant, including "same-named" ones |

When the context is missing and the mode is not compat → reject per §3.2; the fallback must **never** be
"read everything". When a policy or configuration read fails, the default is fail-closed, not a retreat to
the most permissive tier.

> Citable clauses: to reject a `WHERE role_key = ?`, cite "§2.3.1 rule 1"; to reject an
> `ON CONFLICT(role_key)`, cite "§2.3.1 rule 2"; to reject a `Count()` ownership check without the tenant,
> cite "§2.3.1 rule 3". Regression tests for this clause:
> `backend/internal/middleware/data_scope_tenant_test.go`,
> `backend/modules/system/iam/permission/permission_data_scope_tenant_test.go`.

---

### 2.4 Plan definitions and quota semantics (new in V1.4, 2026-10-25)

```
tenant_plans
  id              bigint unsigned PK AUTO_INCREMENT
  code            varchar(32)  UK  -- plan code, ^[a-z][a-z0-9_-]{1,31}$, compared after lowercasing
  name            varchar(64)      -- display name, defaults to the code
  max_users       int              -- member ceiling
  max_roles       int              -- role ceiling
  max_depts       int              -- department ceiling
  max_api_keys    int              -- API key ceiling
  max_storage_mb  int              -- storage ceiling (MB)
  status          varchar(16)      -- active | disabled
  sort            int              -- console ordering
  created_at / updated_at
  UK(code)
```

DDL and seed: numbered migration `000030_tenant_plan_definitions`; AutoMigrate deployments go through
`PlanStore.Migrate()` (called by `Service.Migrate()`, which is the system module's `MigrateFunc`).

Invariants:

1. **A platform-global table with no `tenant_id` column** — same shape as `tenants` itself. Tenant
   confinement in this module is applied by explicit `WithTenantScope` calls in the services that need
   it, not by a global GORM plugin, so adding `tenant_id` here would filter the platform console's own
   reads into nothingness (§2.3 constrains tables that *carry* the column; it does not require every
   table to carry it).
2. **`-1` means unlimited (`tenant.UnlimitedQuota`); `0` is not a legal value** and the write path
   rejects it. Every consumer distinguishes "no ceiling" from "ceiling reached", and allowing `0` would
   let a stored plan collapse the two. Validation happens on the write side (`PlanQuotaReq.resolve`),
   not on the read side.
3. **An empty code means different things on the two sides**: on the definition side
   (`tenant_plans.code`) empty is illegal, because such a row can be neither assigned nor reported
   against; on the tenant side (`tenants.plan`) empty is legal — it is the column default meaning "no
   plan chosen", and what every tenant row predating this feature stores.
4. **Resolution never fails; unknown values fall back to the compiled-in table.** `PlanStore.QuotaFor`
   maps empty, unknown and retired codes onto `GetDefaultQuota`, whose default branch files anything
   unknown under free. A reporting page must not become a failure path for data the operator cannot
   have broken. `GetDefaultQuota` is therefore not dead legacy: it seeds the table and is the only
   resolution source while the table does not yet exist (first AutoMigrate boot, hand-made unit-test schemas).
5. **Retirement is `status = disabled`; row deletion is not modelled.** Disabling is refused while any
   non-deleted tenant still carries the code (`ErrPlanInUse`): a row vanishing underneath a tenant would
   leave that tenant's quota undefined by "a limit nobody chose".
6. **The seed is idempotent and insert-only** (`INSERT … WHERE NOT EXISTS` / per-code count then skip).
   Once the console can edit a threshold, a seed that rewrote values on replay would silently undo the
   operator — and replay is a real state in this repository (see the baseline note in `000024`).
7. **The read-through cache is bounded**: a 5-second TTL (`planCacheTTL`), dropped immediately by any
   write via `Invalidate()`. A changed threshold must reach every instance without a restart, and this
   data is read per request and written rarely — the same policy `pkg/tenant.ModeLoader` uses for
   `tenant_mode`.
8. **The server decides which code may be assigned to a tenant** (added by V1.5, 2026-10-26).
   `PlanStore.ValidateAssignment` is called on the tenant create and update paths, and its order is its
   meaning: an empty value is legal ("no plan selected", see invariant 3) → an ill-formed value is
   `tenant.plan.code_invalid` → when the table is unreadable or empty only the compiled-in four tiers are
   accepted (a seed that has not reached the database is not the operator's fault, and provisioning must
   not stall because of it; any other code is `tenant.plan.unknown`) → a defined but disabled code is
   `tenant.plan.disabled`. "Disabled" follows `tenant.Plan.IsEnabled`, which **fails open**: any status
   other than `disabled` counts as enabled. The console's assignment surface mirrors this one rule rather
   than inventing a second policy.

   The tenant editor therefore takes its choices from `GET /system/tenant-ops/plan/list` and keeps the
   server's `sort ASC, id ASC` ordering (that is what presents the built-in ladder in order — the frontend
   does not re-sort it). The write path lists only assignable codes, but **the code the tenant currently
   carries stays selectable even if it is disabled or no longer defined**, because a form that drops it
   silently rewrites that tenant's plan on save. The read-side filter lists every defined code, disabled
   included, since tenants still carrying one have to be filterable and reported as-is (§2.5).

#### 2.4.1 Quota semantics: a measurement, not a gate (the boundary clause of this section)

- **The backend rejects no write on any path because a plan quota was exceeded.** A quota exists so the
  overview and member pages can report "usage vs limit" honestly; over-limit is highlighted in the UI,
  not blocked. This is the explicit design decision for this round (real statistics and display only),
  and commercial plan selling stays in §1 Out.
- Citable clause: any change that adds `usage >= limit → reject` to a write path is a **contract change**
  and must go through §8 (version bump + change record + runbook/matrix sync); it cannot merge as
  "while I was in there I added a check". The same applies to any "temporary guard" that turns an
  overview threshold into a write-side enforcement.
- **`QuotaEnforcer` (`tenant_production.go`) and `MetricsCollector` (`tenant_metrics.go`) are implemented
  but not wired into the shipped request path**: no production code constructs them. This is recorded so
  that "enforcement looks available" is not misread as "enforcement exists". They are kept because their
  usage counts share one source with the console and they read the same definitions through
  `quotaFor → PlanStore.QuotaFor`, so **if enforcement is ever switched on it cannot disagree with the
  limits the console already shows**. Do not wire them before the clause above has been followed.

### 2.5 Platform-side tenant-operations read paths (new in V1.4, 2026-10-25)

| Route | Permission key | Nature |
|---|---|---|
| `GET /api/v1/system/tenant-ops/overview` | `system:tenant:overview` | cross-tenant aggregate read |
| `GET /api/v1/system/tenant-ops/member` | `system:tenant:member-list` | cross-tenant member read |
| `GET /api/v1/system/tenant-ops/plan/list` | `system:tenant:plan:list` | platform-global read |
| `POST /api/v1/system/tenant-ops/plan` | `system:tenant:plan:create` | platform-global write, audited |
| `PUT /api/v1/system/tenant-ops/plan/:code` | `system:tenant:plan:update` | platform-global write, audited |

Rules (each independently citable by a reviewer):

1. **Every route is an explicitly declared platform-scoped route** in the sense of §3.3 / §3.4: it lives
   inside the protected `system/*` group and is guarded by the tenant console's
   `PlatformCapabilityMiddleware`. A cross-tenant read is allowed in **this shape only**; no
   tenant-facing API may reuse `Service.Overview` / `Service.ListMembersAcrossPages`, which deliberately
   ignore the requesting tenant. The paths form their own group rather than `/system/tenant/overview`
   for a routing-shape reason, not an aesthetic one: `GET /system/tenant/:id` already owns that group's
   wildcard child, so a static sibling is a decision waiting to panic on a Gin upgrade.
2. **Permission keys stay under the `system:tenant:` prefix** so
   `pkg/tenant.IsPlatformOnlyPermissionKey` classifies them without a second list. The backend
   (`pkg/tenant/platform_scope.go`) and the frontend (`core/layout/useLayoutMenu.tsx`) platform-only
   checks must keep applying to that same prefix; a new key outside it must update both sides in the
   same patch.
3. **The numbers must agree with the tenant management page.** Every read shares `opsTenantScope`
   (excludes `id = platformGlobalTenantID`; excludes `status = deleted` unless an explicit status filter
   asks for it, per the §2.1 tombstone rule). Occupancy reuses the module's shared constants
   (`activeTenantMemberPredicate`, `tenantStatusDeleted`, `platformGlobalTenantID`): members count only
   `status = 'active'`, roles and departments respect soft delete, and platform rows (`tenant_id = 0`)
   count toward no tenant's occupancy. **Forbidden**: inventing a second status vocabulary in the read
   model — membership status is only `active | disabled` (§2.2).
4. **Paging is bounded**: overview and member pages default to 20 and cap at 100. The cap has two
   layers with different semantics: request binding (`Filter.Size` / `MemberRowFilter.PageSize`,
   `max=100`) **rejects** oversized input — verified live, `?size=5000` returns `param.invalid` — while
   the service layer clamps again (`maxTenantOverviewSize`, `maxTenantMemberPageSize`) to protect
   internal callers that bypass binding. The contract promises only "bounded", not "silently reduced".
   Reason: every request aggregates the whole installation, so an unbounded size turns one console call
   into a full table scan.
5. **An unknown plan is displayed as itself**: with `OverviewPlanRow.Known = false` and an empty
   `PlanName`, the page shows the raw value and must not silently file it under free — "nobody manages
   this plan" and "this is the free plan" are two different operational facts.
6. **The metric set is frozen to the core group**: tenants, members, roles, departments. Last login,
   session counts and security events are **out** of this contract's overview (explicit Out), otherwise
   the overview slowly grows into a second dashboard.
7. **Permission↔route bindings are gate-protected**: each route is registered in
   `requiredAPIRoutesByPermission` (`permission_workbench.go`) and verified by
   `TestRequiredAPIRoutesExistOnEngine` against the registration mirror in
   `buildWorkbenchRouteProbeEngine()`. A new route or a change of route shape must update the probe in
   the same patch, or the gate fails — the only mechanical defence against "binding table drifts from
   actual registration".

---

## 3. Tenant Context (frozen)

### 3.1 Resolution priority (high → low)

1. Explicit header `X-Tenant-Id` (platform ops / admin scenarios; requires platform-level permission + audit)
2. Authenticated subject (`tenant_id` claim in the JWT, issued at login from the user's default tenant or the requested tenant)
3. Platform-scope fallback: the request landed on a route **explicitly declared as a platform admin API**, the subject holds a platform-level grant, and the subject belongs to no tenant → `tenant_id = 0` (multi mode, see §3.4)
4. Compat-mode fallback: `tenant_id = 0`

> **V1.1 clarification (2026-10-13)**: V1 listed only items 1/2/4 and confined the fallback to compat mode, so under multi mode a "non-tenant (platform) subject" was classified as §3.2's "missing context" (500 `TENANT_CONTEXT_MISSING`) and the platform console was entirely unusable. The new item 3 lands §3.3's already-acknowledged "platform admin API (explicit, platform-permission-gated, audited)" in the resolution layer: **`multi` is a superset of `compat`**, and `tenant_id = 0` (platform-level / global, §2.3/§4) is a first-class scope in both modes. Items 1/2 keep their priority; the compat fallback keeps its meaning (only its number moved to 4).

### 3.2 Propagation rules

- Tenant context is resolved **after the auth middleware, before business handlers**, and is read-only afterwards
- Business code **reads** the context only; rewriting the current request's tenant context inside a handler is forbidden
- Background/async jobs must carry tenant context explicitly (worker messages carry `tenant_id`); process-level globals are forbidden
- Missing context outside compat mode → reject (500 class = configuration error; 401/403 = authn/authz classes, see §5). **Scope**: route groups that have not declared the platform scope; a route declared as a platform admin API resolves per §3.1 item 3 (see §3.4)

### 3.4 Platform-scope declaration (V1.1 clarification)

§3.3 requires the platform admin API to be "explicitly declared". The declaration carrier is the **route group**, and it is the enablement condition for §3.1 item 3:

- An **undeclared** route group keeps V1 semantics: under multi mode a claim-less subject is rejected (deny-by-default) and gets no platform-scope fallback.
- A **declared** route group gains one extra fallback, gated on two conditions that must both hold:
  1. The subject holds a **platform-level grant** — a global role (e.g. `platform_ops`) or a global (domain-less) Casbin policy allowing this route. An in-tenant role (`role:<key>@tenant:<id>`) is not a platform grant (§4: the two namespaces are never merged or implicitly escalated).
  2. The subject **belongs to no tenant** — no active membership. That subject is exactly the §3.1 item-2 "zero memberships → no claim issued" platform population, so both sides must agree.
- The fallback therefore never overrides items 1/2: a subject with a claim resolves via item 2, and a subject holding a membership reaches the platform scope only through the explicit header of item 1 (which requires the platform grant).
- The fallback decides the **context** only; it does not relax §3.3's read invariants: a request resolving to `tenant_id = 0` may only read and write platform/global rows.
- Reference implementation: `internal/middleware.PlatformScopeTenantContextMiddleware` (declaring side) and `internal/middleware.TenantContextMiddleware` (strict semantics for the undeclared side); resolution logic lives in `pkg/tenant.Resolve`.

### 3.3 Data-access invariants (most critical)

- Every query against a `tenant_id`-bearing table **must** include `tenant_id = ctx.tenant_id` (or get it injected via a unified scope helper / GORM scope)
- Hand-written raw queries bypassing the scope helper are forbidden in business code; cross-tenant reads are allowed only via:
  1. Platform admin APIs (explicit, platform-permission-gated, audited)
  2. Statistics/reporting aggregation (a platform-layer read-only channel)
- Write paths must inject `tenant_id` via the unified helper; taking `tenant_id` from the request body to decide ownership is forbidden (sole exception: platform admin APIs specifying it explicitly)

---

## 4. Casbin Domain Dimension (frozen)

- The Casbin policy domain field is `tenant:<tenant_id>`; global (compat/platform-level) uses `tenant:0`
- Role namespace: in-tenant roles map to `role:<name>@tenant:<tenant_id>`; global roles carry no domain
- Check order: global roles/permissions first (platform capabilities), then in-domain roles (tenant capabilities); the two are never merged or implicitly escalated
- Legacy policy migration: under compat mode everything is `tenant:0`; migration is executed by the runbook, never implicitly

---

## 5. Error Codes (frozen)

| Scenario | HTTP | code |
|----------|------|------|
| Header names a tenant but subject lacks platform permission | 403 | `TENANT_FORBIDDEN` |
| Subject has no membership, or membership disabled | 403 | `TENANT_FORBIDDEN` |
| Login or write against a suspended / archived tenant | 403 | `TENANT_SUSPENDED` / `TENANT_ARCHIVED` |
| Missing context (non-compat) | 500 | `TENANT_CONTEXT_MISSING` (configuration error, alert) |
| Resource not in current tenant (query returns empty; no existence leak) | 200/404 | no new code: empty set or 404; "not found" vs "other tenant" is not distinguished |
| Deleting a tenant that is not archived (member count does not matter, V1.6) | 400 | `tenant.not_archived` |

---

## 6. Compat Mode (frozen)

- Switch: a feature flag in `system/config` (`tenant.mode = compat | multi`, default `compat`); readable at runtime, changes follow the config change process
- `compat` mode promises:
  - Existing single-tenant deployments keep running with **zero migration**: `tenant_id = 0` semantics; resolution lands directly on §3.1 item 3
  - No membership checks; all Casbin domains are `tenant:0`
  - Existing tests and API behavior unchanged — this is the regression baseline during canary (`2026-09-10-tenant-canary-slice`)
- `multi` mode: fully enables §3/§4/§5
- **`multi` is a superset of `compat`** (V1.1 clarification): under multi, §3.1 item 4 (the compat fallback) no longer applies, but the platform-level/global scope (`tenant_id = 0`) remains a first-class scope (§3.1 item 3). Non-tenant (platform) subjects and tenant subjects therefore **coexist in one deployment**; `platform.tenant_mode` expresses "is the tenant dimension enabled", not "an all-or-nothing world switch"
- Mode switching is one-way (per deployment): compat → multi; rollback means data rollback per the runbook — running multi and flipping back to compat to continue writes is forbidden

---

## 7. Ownership Boundaries (frozen)

| Concern | Owning domain | Forbidden |
|---------|---------------|-----------|
| tenants CRUD, lifecycle | `system/org` (a tenant is an org entity) | auth/iam/business writing the tenants table directly |
| Membership management | `system/org` provides capability; `system/iam` consumes it (at login) | iam maintaining the memberships table |
| Login-time tenant resolution & claims issuance | `system/auth` | auth querying the tenants table directly (go through org's read interface) |
| Casbin domain rules | `system/iam` | auth or business modules defining their own domain rules |
| Feature flag `tenant.mode` | `system/config` | modules inventing their own switches |
| tenant_id scope helper | `platform` | domains copy-pasting private implementations |
| Business resource tenancy | `business/*` (registered in the matrix) | business defining its own isolation mechanism |

---

## 8. Acceptance Anchors

- The [tenant-readiness checks](../acceptances/BUSINESS_MODULE_ACCEPTANCE_MATRIX.md) (matrix T1–T4) apply to every change touching tenancy semantics
- The canary slice (task 3) is the first runtime validation of this contract's behavioral promises; its isolation tests are the contract's acceptance tests
- Contract changes: after V1 freeze, modifications follow the contract-change process (bump version + change log + sync runbook/matrix)

---

## 9. Change Log

| Version | Date | Change | Nature |
|---------|------|--------|--------|
| V1.1 | 2026-10-13 | §3.1 gains a new item 3 "platform-scope fallback" (the former item 3 becomes item 4 with unchanged meaning); new §3.4 "platform-scope declaration"; §3.2 notes the scope of the reject rule; §6 notes "multi is a superset of compat" | Clarification |
| V1.2 | 2026-10-19 | New §2.3.1 "read and write duties": reads must carry the tenant predicate, writes must take the tenant from the context (conflict keys must include it), bare-key existence/ownership checks count as reads; the only two permitted scopes; fail-closed default | Clarification |
| V1.3 | 2026-10-23 | §2.1 adds the fourth value: `deleted` is a tombstone rather than a lifecycle state, not a valid input, addressing it by id or code is "not found"; the delete marker is `status`, not the reserved `deleted_at`; deleting an archived tenant disables its memberships in the same transaction (§2.2's owner requirement constrains non-archived tenants only) | Clarification |
| V1.4 | 2026-10-25 | New §2.4 "Plan definitions and quota semantics" (`tenant_plans`, platform-global, `-1` unlimited / `0` illegal, unknown codes fall back to the compiled table, retirement instead of deletion, idempotent seed, 5-second read-through cache) and §2.4.1 "a quota is a measurement, not a gate" (no write path rejects on quota; `QuotaEnforcer`/`MetricsCollector` are unwired); new §2.5 "Platform-side tenant-operations read paths" (the five `/system/tenant-ops/*` routes: declared platform scope, `system:tenant:` prefix, one shared predicate set, bounded paging, unknown plans shown as-is, frozen core metric set, permission↔route probe gate); §1 In/Out restates the plan/quota ownership boundary; §2.1's `plan` column comment changes from "not consumed in V1" to "resolves to §2.4 thresholds" | Extension |
| V1.5 | 2026-10-26 | §2.2's "membership removal = `status=disabled`, never a physical delete" is restated as a **four-row per-path table** (tenant deletion / user deletion = soft-disable, explicit member removal = physical delete, re-join = row revival), and "audit chain" is re-attributed from row retention to the global operation log; §2.4 gains invariant 8, moving the **plan assignment surface** that V1.4 registered as a residual gap into the In set and wiring it in the frontend (backend `AssignableCodes` deleted); this English file is brought up to date with §2.3.1 (V1.2) and its change log. Write-side enforcement behaviour unchanged; "make member removal a soft-disable too" is registered as a future contract-change candidate with its prerequisites | Clarification (§2.2) + Extension (§2.4 invariant 8) |
| V1.6 | 2026-10-27 | §5 registers the delete-guard error code `tenant.not_archived` (400). The implementation refused deletion on "member count > 0 and not archived", which contradicts §2.1's "deletion only for `archived` tenants": deleting every user drove the active member count to zero and let a live tenant be tombstoned without archiving. The guard is now the §2.1 rule verbatim (the status, not the member count); both the platform and the legacy handler register the code, and the console disables the delete action on non-archived rows; §2.1 and §2.2 wording unchanged | Clarification + Extension (§5 error code) |

**Trigger and scope of V1.5**

- Trigger: V1.4's change log closed with an explicitly registered doc↔code drift in §2.2 ("this round
  does not adjudicate it"). This round adjudicates it.
- Decision: **the contract follows the implementation**, the write side is not changed. Three of the four
  paths already soft-disable and are pinned by tests (`DeleteTenant`, `deactivateTenantMemberships` +
  `user_membership_test.go:152-200`); only `RemoveMember` deletes physically, and converting it would also
  change the error semantics of a repeated removal (it relies on `RowsAffected == 0`) and the membership
  counting basis — a write-side change of the §2.4.1 kind that needs its own round.
- Nature: **two parts, a Clarification and an Extension**. §2.2 is the Clarification at the level of clause
  semantics — its "at least one owner" and "namespace separation" invariants are unchanged, only "how a
  membership is retired" moved from one sentence to a per-path table. §2.4 gains invariant 8, which is an
  **Extension**: V1.4 registered the assignment surface as a residual gap (Out) and this round moves it In
  as a normative clause. §2.1, §2.3, §2.3.1, §2.5, §3-§7 are semantically untouched, and write-side
  enforcement still sits behind §2.4.1's contract-change gate.
- **Provenance and cost (stated plainly, not glossed over)**: "soft-disable, no physical delete" was not
  loose wording — it was a *Frozen* decision of the 2026-09-10 contract freeze, justified as "audit chain
  integrity (audit reviewer requirement)"
  (`.harness/archive/2026-09/evidence/2026-09-10-tenant-contract-design/summary.md:24`). Rewriting the
  contract to ratify the implementation therefore **acknowledges that a frozen requirement is unimplemented
  on one path**; that fact does not disappear because the document now matches the code. Two things are kept
  on record: ① §2.2 states "soft-disable is the default direction; the physical delete is the exception that
  still needs converging"; ② the convergence candidate and its prerequisites are registered (last bullet of
  §2.2). Whether to instead change the implementation is the maintainer's call at the acceptance touchpoint —
  this contract does not declare the frozen decision void.
- Implementation and regression coverage for the assignment surface:
  `frontend/src/modules/system/iam/tenant/planOptions.ts` (the derivation rules that mirror §2.4
  invariant 8), `TenantList.tsx` (the editor form and the filter now consume it) and
  `getPlanList({ skipErrorMessage })` in `api.ts` (the list is advisory for the form — if it cannot be
  read the editor falls back to the built-in tiers and must not raise a user-facing error), plus
  `system.tenant.plan.outsideLadder` in the five locale resources.
  `frontend/tests/unit/modules/system/iam/tenant/planOptions.test.ts` covers five cases: fallback on an
  unreadable/empty table, server ordering plus an operator-defined code, a disabled code hidden from the
  write path but kept in the read filter, the carried-over current code (no duplicate, built-ins unaffected),
  and a built-in code arriving from the table still labelled as built-in. The backend's `AssignableCodes`
  was deleted — see the closure note on the gap below.
- Also in this round (no contract semantics): the English file gains the §2.3.1 translation that V1.2
  never received, and this change log gains the V1.2/V1.3/V1.5 rows. The translation note below is
  corrected accordingly — it was itself stale, since V1.3's tombstone content had already been translated
  into §2.1.
- Registered then adjudicated by evidence ("premise does not hold, no change made"): an earlier bullet
  claimed the tenant-side `DELETE /tenants/:id/members/:user_id` handler sets no audit title and therefore
  diverges from the platform side's `SetAuditMetadata(..., BusinessDelete)`. That route belongs to the
  legacy `Handler` (`tenant_handler.go:244-263`), whose only mounting entry point, `RegisterRoutes`
  (`tenant_routes.go:9-28`), has no caller in shipped code — outside the package definition, the only calls
  to `RegisterRoutes` and `NewHandler(` are in the same-package `tenant_handler_test.go` (six of them,
  unqualified), and `system_modules.go:57` declares the field type as `*tenantiam.PlatformHandler` itself.
  Every shipped tenant write route is mounted on `PlatformHandler` by `system_modules.go:290-306` and sets
  its title explicitly (`tenant_platform_handler.go:131/148/168/203/223`, `tenant_ops_handler.go:70/87`).
  `docs/api/TENANT_API.md:17` already records that group as "not mounted", and the 2026-10-22 delivery
  audit measured `/api/v1/tenants` returning 404 in a running build. So no runtime asymmetry exists, and
  adding a title to a handler with no routes would be dead code.

> Translation note: the gap recorded here before — "this English file lags the Chinese source for V1.2
> (§2.3.1) and V1.3 (the `deleted` tombstone)" — was itself partly stale: V1.3's tombstone content had
> already been translated into §2.1. As of 2026-10-26 §2.3.1 is translated too, and the section structure
> of the two files matches 1:1 (§1-§9, §2.1-§2.5, §2.3.1, §2.4.1 — verified by heading comparison this
> round). What that does **not** claim: a clause-by-clause re-audit of §3-§7 prose parity was not part of
> this round. The Chinese file [TENANT_CONTRACT_V1.md](./TENANT_CONTRACT_V1.md) stays authoritative: where
> the two disagree, the Chinese text governs and this file has the defect.

**Residual gaps of V1.4 (explicit)**

- Plan *selling* (ordering, upgrades/downgrades, billing periods) stays Out per §1. `max_api_keys` and
  `max_storage_mb` are defined and persisted but not reported by the overview, because §2.5 rule 6's
  core metric set does not include them.
- **Assignment surface is not wired.** `PlanStore.AssignableCodes` (`tenant_plan_store.go:175`) exists
  and is unit-tested, but no shipped code path calls it: the tenant editor still lists the four literal
  built-in codes (`frontend/src/modules/system/iam/tenant/TenantList.tsx:696-705`). A newly defined
  custom code can therefore be stored and is displayed honestly on the read side (`Known = false`), yet
  cannot be assigned to a tenant from the console. Same reasoning as §2.4.1 — this round kept the read
  side and the editable thresholds and left the write side alone; the assignment surface is also coupled
  to the frontend's own plan ladder (`core/entitlements/plans.ts`, four ordered tiers, an unknown code
  fails closed to `free`) and needs its own round.
  **→ Wired by V1.5**: `frontend/src/modules/system/iam/tenant/planOptions.ts` derives the choices from
  `/system/tenant-ops/plan/list` (rules mirroring §2.4 invariant 8), `TenantList.tsx` uses them for both
  the editor and the filter, and the server's `ValidateAssignment` still has the last word on save.
  `AssignableCodes` itself is deleted: it sorted alphabetically (wrong for a ladder) and never had a
  shipped caller, so leaving it in place only invites a second policy.
  **Still open**: the frontend capability ladder. `core/entitlements/plans.ts` remains four hardcoded
  tiers with an unknown code failing closed to `free`, so an operator-defined plan's console **feature
  entries** render at the free tier while its quota ceilings still apply. The form states this in a
  warning line instead of hiding it; making it real per-plan menu/entitlement trimming is the commercial
  semantics that stays in §1 Out and needs its own round.

**Trigger and scope of this clarification**

- Trigger: task `2026-10-13-tenant-console-remediation`. Under multi mode, `dashboard` / `system/dict/*` / `system/setting/*` / `system/operation-log/*` / `auth/operation-verify` all returned `500 tenant.context.missing`, `X-Tenant-Id` was unusable for any role other than `platform_ops` (403), and the platform console was unusable as a whole. Because the settings page failed the same way, the UI could not switch back to compat either (§6 also forbids switching back).
- Gap located: §3.3 requires the platform admin API to be "explicitly declared", but §3.1 never said how a declared route resolves its context. The implementation granted the fallback to compat only, so a platform-level subject under multi fell into §3.2's "missing context".
- Scope of the clarification: only §3 resolution semantics and §6 mode semantics are completed. **§2 (model), §4 (Casbin domain), §5 (error codes) and §7 (ownership boundaries) are unchanged**; §3.1's priority order and the item 1/2 conditions are unchanged.
- Implementation: `pkg/tenant.Resolve` (the `platform-fallback` resolution source), `internal/middleware.PlatformScopeTenantContextMiddleware`, `pkg/tenant.IsPlatformPopulationUser`; the route-group declaration list is in the task packet `2026-10-13-tenant-console-remediation`.

> Governance note: this record means runbook `TENANT_MIGRATION_RUNBOOK.md` §8 gate G4 ("no open contract-change") no longer holds automatically. G4 is the maintainer's call; this clarification does not declare G4 granted. The production DDL/data-change ban is unaffected by this document.
