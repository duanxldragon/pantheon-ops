# Pantheon Base v0.17.0 — Release Notes

**Release date**: 2026-10-21
**Tag**: `pantheon-base-v0.17.0`
**Previous release**: `pantheon-base-v0.16.1`

## Summary

v0.17.0 is the first minor release after the artifact channel was repaired in v0.16.1. It carries
the tenant isolation remediation that was deliberately held back from the v0.16.1 cut, the
follow-up permission and seeding fixes, the dependency bumps, and the release-pipeline repairs.

Three P0/P1 security defects are closed: a tenant could write a **global** permission policy, a
tenant could create a role key that shadows the platform namespace (`admin`, `platform_ops`) and
inherit platform grants at login, and `platform.tenant_mode` was never made public so the tenant
controls never rendered and `Tenant Smoke` was red on every `main` commit. Each is fixed at the
namespace layer, not by hiding the surface. A subsequent full-repository acceptance review
(2026-10-21) closed one further tenant-isolation P0 and a set of P1/P2 defects before this release
was cut; they are listed under **Acceptance remediation** below.

No consumer-facing interface is removed. The only data change is additive: migrations `000025`
through `000028` backfill tenant-scoped rows and add `tenant_id` to the remediation event table.

## Security

### The permission console is bound to the request tenant (#393, #386)

Every policy endpoint now resolves the tenant from the request context. A tenant-scoped write always
stores `role:<key>@tenant:<id>`, a body `tenantId` naming another tenant is rejected with
`TENANT_FORBIDDEN`, and `tenantId=0` no longer produces a global policy. List, export, update and
delete only touch the caller's domain (contract §2.3.1). Policy import and workbench remediation
resolve the stored subject from the context instead of writing the bare role key.

### Tenant role keys cannot shadow the platform namespace (#393)

Role create/rename/import reject keys already owned by tenant 0 and the reserved `platform_ops`
key. CSV import stamps the caller's tenant; it used to write `tenant_id = 0`, which is a platform
role. Import role-key validation and the bootstrap admin binding are scoped to the target namespace.

### Role lifecycle understands the tenant-domain subject form (#393)

Delete removes exactly the role's own subject, rename moves the policies in the same transaction,
and the bootstrap orphan cleanup recognises `role:<key>@tenant:<id>` instead of deleting every
tenant policy on each boot.

### The remediation history is per tenant (#393)

`permission_workbench_remediation_event` gains `tenant_id` (migration `000028`), so remediation
history is no longer shared between tenants that own the same role key.

## Fixed

### `platform.tenant_mode` is published on upgraded deployments (#393)

The seed marks it public and `Bootstrap` raises `is_public` on an existing row, never lowers it.
Upgraded deployments now render the tenant picker, switcher and viewpoint switcher again — this was
one cause of the red `Tenant Smoke` job on every `main` commit. A second, independent trigger (the
server-side auto-join colliding with the console's re-add call, returning `tenant.member_exists`)
was root-caused from the CI trace artifacts and removed by the idempotent membership write described
below. The job is not claimed fixed: the same commit both failed (`119005706470`, the membership
assertion in `tenant-viewpoint-switcher.spec.ts`) and passed (`119067519288`, 10 passed), so the
remaining red is flakiness whose full cause set is still open.

### The auth module seeds its own permission keys (#392)

Seeding covered the auth module's menus but not its permission keys, so a fresh tenant had a login
surface whose actions could not be authorised without a global grant.

### Tenant menu seeds and SonarCloud findings (#391)

Tenant menu seeds were completed and every open SonarCloud finding on `main` was closed, including
the security-surface documentation the quality gate requires.

## Acceptance remediation (2026-10-21 review)

Full evidence, severity classification and per-issue call chains live in
`docs/reviews/PANTHEON_BASE_ISSUES_AND_REMEDIATION_2026-10-21.md`. Summary of what changed:

- **FIX-1 (P0, tenant isolation):** menu write routes (`POST/PUT/DELETE /menu`) and the i18n system
  group writes were reachable by any tenant-scoped grant. `tenant_admin` holds `/api/v1/*` in its
  own domain, so it could rewrite the global menu tree — and menu save cascades a
  `system_role_permission` cleanup that crosses tenants. Both surfaces now carry
  `PlatformCapabilityMiddleware`; `/menu/tree` and `/i18n/pack` stay tenant-readable.
- **FIX-2 (P1, compat hardening):** `WithTenantScope`/`WithTenantWrite` no longer skip filtering in
  compat mode; a nil-safe context always pins `tenant_id`, so a mis-flipped `tenant_mode` cannot
  silently expose tenant rows. A downgrade guard rejects `multi → compat` while non-zero tenants
  exist.
- **FIX-3 (P1, lifecycle):** suspending a tenant or removing a member now revokes the affected
  users' sessions in that tenant instead of leaving live tokens until TTL.
- **FIX-4 (P2, provisioning):** tenant provisioning copies platform dictionary types/items
  (idempotent INSERT…SELECT), so a new tenant starts with usable dictionaries.
- **FIX-5 (P2):** the role-key namespace mutex is now bidirectional — platform-side create/rename
  can no longer take a key owned by a tenant role.
- **FIX-6 (P3):** production builds run Gin in release mode.
- **FIX-7 (P2):** the legacy tenant handler's raw `{"error": …}` responses were unified to the
  standard envelope without leaking internal error strings.
- **FIX-8 (P1, schema):** OIDC columns lived in an untracked stray SQL file; folded into guarded
  migration `000029`, with the global `oidc_subject` unique key replaced by
  `(tenant_id, oidc_provider, oidc_subject)` and all queries tenant-scoped.
- **FIX-9 (P1, frontend):** route-warm data is invalidated at logout/login boundaries, so a second
  account in the same tab can no longer see the first account's pre-fetched pages.
- **FIX-10 (P2, frontend):** the second-factor operation token is cleared on every session teardown.
- **FIX-11 (P2, membership):** `AddMember` is idempotent — re-adding an existing member updates or
  reactivates the row instead of failing, which was the second root cause of the red
  `Tenant Smoke` job and a false "join failed" toast for every tenant-viewpoint user create.
  Owner rows are protected from demotion through this path.

## Changed

- Data-scope and role-menu migrations now backfill tenant-scoped rows before the permission console
  is exposed (`000025`–`000027`).
- `backend` package naming was aligned with the v0.16.0 correspondence model, so
  `backend/modules/system/iam/menu` is `package menu`.

## Dependencies

- 18 frontend npm packages, 2 root dev dependencies, 4 GitHub Actions (still SHA-pinned).

## Release tooling

- The cut workflow validates inputs after the checkout, builds the bundle from the requested
  `base_commit`, creates the tag through the API and verifies it with `git ls-remote`.
- The v0.16.1 closeout record was repaired to match what was actually published.

## Verification

- Required checks green on the release commit: `CI Summary`, `Quality Gates`, `Security Gates`,
  `Actionlint`, `Full Smoke`, `SonarCloud Code Analysis`, and `Release Gate Summary`.
- Backend `go test ./...` against real MySQL in CI; the new tenant cases are two-tenant and
  role-lifecycle tests, not mocks.
- Frontend `tsc -b` clean.
- The four consumable assets are published with the release and verified by
  `Foundation Release Verify` on `release: published`.

## Consumer impact (`pantheon-ops`)

Low. `sharedPaths` is unchanged at 52 entries (backend 4, frontend 46, docs 2), so the consumer lock
moves only version, commit and checksums. The migrations are additive and the business overlay's six
assembly anchors are unaffected. See `upgrade-notes.md` and `consumer-impact.md` in the release
evidence directory.

## Known limits

- Tenant role keys that already shadow the platform namespace are **not** renamed automatically.
  Operators must run the documented cleanup before upgrading an existing deployment.
- The browser-level `Tenant Smoke` assertion is confirmed by CI, not by a local run; the underlying
  seeding defect is covered by unit tests.
- The 2026-10-21 acceptance remediation (FIX-1…FIX-11) is verified by local unit tests and the full
  local gate; its browser-level confirmation lands only when the remediation reaches CI, and the
  `smoke-tenant` job remains advisory (`continue-on-error: true`).
- "Full local gate" needed a qualifier: `pkg/testmysql.Open` and `pkg/testredis.Open` both report a
  pass-shaped result when their store is missing, so a MySQL-only run silently skipped every
  Redis-backed test. One such test (`TestSessionAdminOps_CompatPinsToPlatformTenant`) was therefore
  locally green and red in CI; the assertion itself was wrong and is corrected in the follow-up
  commit. With both stores configured the backend suite is 46 packages ok, exit 0.
- Release-readiness, measured on `main` at `c17ef999` (2026-10-21): #396/#397 landed on `release/v0.17.0`, #395 squash-merged that branch into
  `main`, and #398 carried the Sonar follow-up. All functional gates are now green on `main` — `CI` (which runs `go test -race`),
  `Backend Tests`, `Full Smoke Suite`, `Core Smoke Tests`, `Security Gates`, `CodeQL`, `Lint Workflows`. **`Release Gate` is still red**,
  for one reason: the branch-level Sonar analysis puts `Security Rating on New Code` at E (threshold A) because `gosecurity:S3649`
  reports two remaining BLOCKERs in `Service.AddMember()` (the `(tenant_id, user_id)` read at `:711` and the update at `:726`), which are
  bound-parameter false positives. Two semantics-preserving restructurings took that rule from 3 blockers to 2; the remaining edge is the
  tenant/user lookup itself, so it cannot be removed without suppressing the scanner or changing what the endpoint does.
  The maintainer then chose the suppression path (2026-10-21): both `AddMember` sinks carry a trailing `// NOSONAR gosecurity:S3649`, while the
  Sonar rule configuration, the Quality Gate thresholds and the `Release Gate` definition stay untouched. Verified on the hosted service: PR #402
  reported 0 new-code issues, and `main` at `17af0d85` reports open issues = 0, Quality Gate OK on all five conditions, open Dependabot alerts = 0
  and `Release Gate` success. The `pantheon-base-v0.17.0` tag is the remaining maintainer action; nothing is red.
