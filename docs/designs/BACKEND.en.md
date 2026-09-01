---
title: Backend Architecture Design and Development Rules
doc_type: Design
layer: platform
status: Active
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
  - docs/contracts/SYSTEM_ORG_CONTRACT.md
updated_at: 2026-04-17
---

# Backend Architecture Design and Development Rules

Chinese version: [BACKEND.md](./BACKEND.md)

This companion provides the English reading surface for the main backend design document.

## 1. Core Architecture: Modular Monolith

Pantheon Base uses a modular monolith. Modules are physically separated, but they communicate through explicit contracts exposed by shared platform packages such as `pkg/common`.

## 2. Development Pattern: Vertical Slices

Each feature package must own its full stack. Do not split a domain into horizontal folders that mix unrelated features.

### 2.1 Expected Slice Shape

- `*_model.go`: physical database mapping
- `*_dto.go`: request and response contracts
- `*_repository.go`: persistence access
- `*_service.go`: business flow, permission rules, transactions
- `*_handler.go`: HTTP routing, binding, response assembly

## 3. Layer Responsibilities

- `Model`: database alignment only, no business logic
- `DTO`: transport contract, must mask sensitive fields
- `Service`: the single business orchestration layer
- `Handler`: HTTP boundary, returns standard envelopes through shared helpers

## 4. Key Platform Capabilities

- Authentication uses opaque Redis Token with access and refresh tokens.
- Sessions persist refresh-token JTI in `system_user_session`, rotate on refresh, and revoke on logout.
- Access tokens carry `userId`, `username`, `roleKeys`, and `sessionId`.
- Casbin enforces RESTful route policies, with `admin` as the default full-access role.
- Casbin persistence is bootstrapped by SQL initialization plus runtime migration through the GORM adapter.
- Non-admin users cannot create, update, or delete Casbin policies whose target path matches `/api/v1/system/permission*` or `/api/v1/system/role*`; the service returns `permission.escalation.forbidden`.
- When `PANTHEON_CASBIN_WATCHER=true` and Redis is available, policy writes fan out through a Redis watcher and trigger `LoadPolicy()` on peer instances. The default remains single-instance behavior.
- `PANTHEON_TOKEN_CACHE_TTL_SECONDS` controls the in-process token cache TTL. The default is `60` seconds; setting it to `0` disables the process-local cache so every check goes through Redis.
- Operation audit logs are written asynchronously to `system_log_oper`, with recursive masking for sensitive keys.
- Preference updates from `PUT /api/v1/auth/me/preferences` are audited without leaking secrets.
- Request tracing is normalized through `X-Request-ID` and `X-Trace-ID`, and persisted into audit records.
- Runtime i18n packs are served from `/api/v1/system/i18n/pack`, with Redis preferred as cache.
- `GET /api/v1/health` returns process, database, and Redis status through the unified response envelope.
- Cross-domain aggregate APIs such as dashboard and workbench belong to the `platform` layer, not to a single `system/*` subdomain.
- Runtime database support is now MySQL-only; `PANTHEON_DSN` must be a valid MySQL DSN.

## 5. Implemented System APIs

Authentication is in a transition period where both new and legacy paths exist:

- Primary auth namespace: `/api/v1/auth/*`
- Legacy-compatible namespace: `/api/v1/system/login`, `/api/v1/system/refresh`, `/api/v1/system/logout`, `/api/v1/system/user/info`, `/api/v1/system/profile/password`

The current API surface covers:

- platform health and dashboard summary
- auth login, MFA verify, refresh, logout, activity heartbeat, profile, password, sessions, login logs, and security overview
- `system/iam` user, role, permission, and menu management
- `system/org` department and post governance
- `system/config` dictionaries, settings, uploads, and dynamic modules
- `system/auth` and `system/audit` administrative logs

Keep the Chinese primary document as the authoritative endpoint inventory.

### 5.1 MFA During Login

- If `login.mfa_enabled=false`, `/api/v1/auth/login` keeps the standard username-password flow.
- If `login.mfa_enabled=true`, successful password validation returns a challenge instead of tokens.
- Bound users continue with `POST /api/v1/auth/mfa/verify`.
- Unbound users receive `totpSecret` and `totpProvisionUri` for enrollment.
- A challenge is valid for 5 minutes and can only be consumed once.

## 6. Decoupling Rules

- New business packages must live under `modules/business/`.
- Modules may not import each other's services directly.
- Identity and role context must come from `gin.Context`.
- Backend modules must register through the shared `pkg/contracts.BackendModule` contract and declare `Name`, `Migrate`, `RegisterRoutes`, `SeedMenus`, `SeedPerms`, and `SeedI18n`.

## 7. Input Validation Additions

The backend must enforce:

- unique usernames, role keys, post codes, and dictionary identifiers
- protection for built-in admin identities and root organization nodes
- safe organization relationships for users, departments, and posts
- encrypted handling of sensitive settings
- tenant-ready thinking for new business tables and uniqueness rules
- unified upload constraints from `system/config`
- governance restrictions for audit-log retention and destructive cleanup actions

### 7.1 Import and Export Conventions

- CSV is the current standard format.
- Export endpoints use `POST /export` so they enter unified audit logging.
- Template downloads use `GET /import-template`.
- Import APIs return structured summaries even when some rows fail validation.

## 8. User Pagination Rules

- `page` defaults to `1`
- `pageSize` defaults to `10` and caps at `100`
- supported sort fields are whitelisted
- responses always return `items`, `total`, `page`, and `pageSize`

## 9. Menu Tree Sorting Rules

- default order is `sort asc, id asc`
- only whitelisted sort fields are allowed
- responses remain tree-shaped through `children`

## 10. Role Pagination Rules

- `page` defaults to `1`
- `pageSize` defaults to `10` and caps at `100`
- sorting is field-whitelisted
- each role item also carries `menuIds` for the authorization form

## 11. Permissions and Menu Visibility

- Casbin checks all `roleKeys`; any matching role grants access.
- Self-service endpoints for logged-in users must not depend on menu authorization.
- `scope=nav` only returns visible, authorized navigation nodes and auto-completes ancestors.
- `scope=manage` returns the full tree for menu management and role authorization.
- New menus are auto-bound to `admin`.
- Policy mutations trigger `Casbin LoadPolicy()` immediately.
- High-sensitivity pages such as system settings require both frontend permissions and backend route policies.
