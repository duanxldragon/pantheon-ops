---
title: system/auth Contract
doc_type: Contract
layer: system/auth
status: Active
updated_at: 2026-11-04
---

# system/auth Contract

Chinese version: [SYSTEM_AUTH_CONTRACT.md](./SYSTEM_AUTH_CONTRACT.md)

This document defines the execution contract for the Pantheon `system/auth` capability domain.

It locks down the responsibility boundaries for authentication, sessions, security center, login logs, and authentication strategy so that `auth`, `iam`, `org`, and `config` do not collapse back into one oversized “system security misc room”.

## 1. Background

One of Pantheon’s earliest structural problems was that authentication and backoffice user management were written together.

Without a `system/auth` contract, the likely regressions are:

- login, refresh, logout, sessions, passwords, and login logs getting pushed back into the user module
- security center and user-management pages collapsing into one capability domain
- reserved capabilities such as `MFA / CAPTCHA / SSO` having no stable landing place
- the authentication primary path being dragged down by backoffice CRUD concerns

This contract makes the ownership explicit:

> `auth` is responsible for “who you are, whether you can log in, whether your session is still valid, and how security policy applies,” not for system-management CRUD.

## 2. Ownership Layer

This contract belongs to `system/auth`.

It covers:

- login
- refresh-token rotation
- logout
- current principal identity
- current-account session management
- login logs
- security center
- authentication-domain security policy

It does not mean:

- `system/iam` user, role, menu, and permission management
- `system/org` organization ownership governance
- `platform` application shell and dashboard aggregation

## 3. Goals

The `system/auth` contract exists to lock down:

1. the boundary between `auth` and `iam`
2. the rule that the auth primary path revolves only around identity verification and session validity
3. the rule that security center, session management, and login logs belong to `auth`
4. the distinction between reserved future security capabilities and real protocol implementation
5. the definition of done and acceptance semantics for the auth domain

## 4. Non-Goals

This contract does not cover:

- user-list CRUD
- role CRUD
- menu and permission-policy CRUD
- department and post governance
- business-domain account models
- real `CAPTCHA / SSO` protocol flows before the identity source is defined

In other words:

- `auth` may reserve capability switches such as `CAPTCHA / SSO`
- but it must not damage the current local username-password primary path in the name of future flexibility

## 5. Boundaries

### 5.1 Covered Surfaces

- `POST /api/v1/auth/login`
- `POST /api/v1/auth/mfa/verify`
- `POST /api/v1/auth/refresh`
- `POST /api/v1/auth/logout`
- `GET /api/v1/auth/me`
- `GET /api/v1/auth/sessions`
- `DELETE /api/v1/auth/sessions/:id`
- `PUT /api/v1/auth/password`
- `GET /api/v1/auth/security`
- `/login`
- `/auth/security`
- `/system/session`
- `/system/login-log`

### 5.2 Excluded Surfaces

- `/system/user`
- `/system/role`
- `/system/menu`
- `/system/permission`
- `/system/dept`
- `/system/post`
- platform-shell navigation and dashboard

## 6. Dependencies

This contract depends on:

- [DESIGN.md](../../DESIGN.md)
- [AGENTS.md](../../AGENTS.md)
- [BACKEND.md](../designs/BACKEND.md)
- [FRONTEND.md](../designs/FRONTEND.md)
- [AUTH_MODULE_DESIGN.md](../designs/AUTH_MODULE_DESIGN.md)
- [SECURITY_CENTER_DESIGN.md](../designs/SECURITY_CENTER_DESIGN.md)
- [ERROR_CODE_AND_I18N.md](../designs/ERROR_CODE_AND_I18N.md)
- [ACCEPTANCE_CHECKLIST.md](../acceptances/ACCEPTANCE_CHECKLIST.md)

## 7. Hard Constraints

### 7.1 Domain-Boundary Constraints

- authentication, sessions, tokens, and security policy belong to `system/auth`
- user profiles, role grants, menu management, and permission-point governance belong to `system/iam`
- `auth` must not take over backoffice user-management CRUD

### 7.2 Primary-Path Constraints

- login, refresh, logout, sessions, and password form one authentication primary path
- security policy may only affect the auth primary path and must not leak without boundaries into other system-domain pages
- current-account sessions and administrator-visible login logs should both be understood from auth-domain semantics, not pushed into `iam`
- `/login` may expose language selection, but the runtime priority of that choice is interpreted by the `platform` shell
- `system/auth` must not automatically persist login-page language choice as long-term user preference
- `preferences.language` in `auth/me` or login responses is only the default preference source, not the final interpreter of current-session language

### 7.3 MFA and Reserved-Capability Constraints

- `login.mfa_enabled` now participates in real TOTP second-factor verification; disabling it must not break the original username-password path
- real MFA currently means TOTP verification after local username-password login, not external-identity SSO
- when MFA is enabled for the first time, users without a bound factor must be allowed to bind through QR/manual-secret flow instead of being locked out
- the login page must expose at least two of the following: QR code, manual secret, `otpauth://` URI, with QR preferred and manual secret as fallback
- the backend must never echo the current OTP code itself; it may return only the secret/URI required for a local authenticator app to generate the six-digit code
- `login.captcha_enabled` and `login.sso_enabled` may remain as configuration switches
- until a real protocol exists, fake CAPTCHA or SSO entry points must not be displayed
- until the identity source is defined, real `SSO / OIDC / OAuth2 / WeCom` integration must not be implemented early
- 2026-10-31 disposition (AUD-005, option (a) step 1): `backend/modules/auth/oidc/` (4 files / 587 lines) was unreachable implementation written before either design or contract approval — zero package-external imports, zero route registrations, no registered environment variables, no tests, in direct conflict with this clause; it has been deleted. Any future re-implementation must take `docs/designs/SSO_OIDC_DESIGN.md` (still Draft) as design input and go through approval again
- Kept: migration `000029_oidc_fields` and the four nullable `system_user` columns (`auth_type` / `oidc_*`) are not dropped — published migrations must not be rewritten, the columns have no consumers and zero carrying cost, and the schema stays reusable once the identity source is decided
- Deferred (registered here, not a verbal promise): delete the code first, observe one release, then land a column-drop migration separately under the contract-change discipline of §8

### 7.4 Security Constraints

- auth failure throttling, source/IP limits, session idle timeout, and maximum active sessions must be centrally owned by `auth`
- login logs, request identity, and session revocation chains must remain traceable
- auth-domain errors should prefer stable keys rather than natural-language strings

### 7.5 Documentation Constraints

- all `system/auth` design, assessment, remediation, and acceptance docs must link back to this contract
- if an “auth-related” change actually modifies user-management behavior in `iam`, the boundary must be explained first

### 7.6 Legacy Dual Entry (`/system/*` Auth Endpoints): Deprecation and Sunset Constraints

`/system/login`, `/system/refresh`, `/system/logout` and `/system/user/info` are legacy dual entry points sharing handlers with `/auth/*` (AUD-014, 2026-10-31 decision: keep and mark deprecated instead of immediate removal; both base and ops frontends call only `/auth/*`).

| Endpoint | Legacy path | Current path | Disposition |
| --- | --- | --- | --- |
| login | `POST /api/v1/system/login` | `POST /api/v1/auth/login` | kept until 2026-12-31 with deprecation headers |
| refresh | `POST /api/v1/system/refresh` | `POST /api/v1/auth/refresh` | kept until 2026-12-31 with deprecation headers |
| logout | `POST /api/v1/system/logout` | `POST /api/v1/auth/logout` | kept until 2026-12-31 with deprecation headers |
| current user info | `GET /api/v1/system/user/info` | `GET /api/v1/auth/me` | kept until 2026-12-31 with deprecation headers |

- Responses matching the four paths must carry `Deprecation: true` and `Sunset: Sat, 31 Dec 2026 23:59:59 GMT` (`middleware.LegacyDeprecationMiddleware` matching `legacySelfServicePaths`; see `backend/internal/middleware/csrf_middleware.go` and the two mounts in `backend/modules/auth/module.go`)
- The sunset date is registered as **2026-12-31**. Removing the endpoints changes an external promise: the maintainer must first confirm that no external integrator still calls the four paths
- Teardown must remove the two routes in the `sys` public group (which then becomes empty and is deleted), the `/logout` and `/user/info` lines under `systemProtected`, both `LegacyDeprecationMiddleware` mounts, and the exact `/api/v1/system/login` and `/api/v1/system/refresh` CSRF exemptions in `csrf_middleware.go` — no exemption without a matching route may remain. Teardown plan: `docs/runbooks/DECOMMISSION_LEGACY_AUTH_ROUTES_2026-12-31.md`

## 8. Definition of Done

`system/auth` should count as currently complete only when:

### 8.1 Responsibility Completion

- `auth` is modeled independently from `user`
- login, sessions, security center, and login logs have clear ownership

### 8.2 Primary-Path Completion

- login, refresh, logout, session listing, and session revocation are stable
- current-user password update is stable
- login logs and security center are stable

### 8.3 Policy Completion

- minimum password length, password complexity, login-failure lockout, source/IP throttling, idle timeout, and max-session policy have a unified policy entry
- reserved capabilities are explicitly marked as either “reserved only” or “real implementation”

### 8.4 Documentation and Acceptance Completion

- auth-related design, acceptance, and remediation docs all link back to this contract
- acceptance can clearly distinguish between “design reserved” and “real protocol implementation”

## 9. Acceptance Standards

Changes in `system/auth` must at least pass:

### 9.1 Documentation Acceptance

- [ACCEPTANCE_CHECKLIST.md](../acceptances/ACCEPTANCE_CHECKLIST.md)
- [DOCUMENT_GOVERNANCE_CONTRACT.md](../contracts/DOCUMENT_GOVERNANCE_CONTRACT.md)
- [DOCUMENT_METADATA_AND_STATUS.md](../contracts/DOCUMENT_METADATA_AND_STATUS.md)

### 9.2 Backend and API Acceptance

- `go test ./backend/modules/auth`
- if tracking chains or middleware are touched, add `go test ./backend/internal/middleware`

### 9.3 Frontend and Build Acceptance

- `cd frontend && npm run build`
- if the auth primary path or security-page primary path changes, add smoke or acceptance evidence

### 9.4 Page and Primary-Path Acceptance

- `/login`
- `/auth/security`
- `/system/session`
- `/system/login-log`

Additional checks:

- when `login.mfa_enabled=false`, username-password login must remain usable
- when `login.mfa_enabled=true` and the user has no bound factor, `/login` must enter the binding flow instead of failing directly
- when the username or password is wrong, the login page must show the translated auth-domain error key, not a vague generic failure
- when second-factor code is wrong, challenge expires, or source/IP is limited, the login page must prioritize translated backend error keys

### 9.5 Reserved-Capability Acceptance

- if docs claim `MFA / CAPTCHA / SSO` are reserved only, they must explicitly say they are not real protocol implementations
- if the implementation becomes real, it must have a dedicated design and acceptance chain instead of reusing the “reserved” framing

## 10. Related Documents

### 10.1 Design

- [AUTH_MODULE_DESIGN.md](../designs/AUTH_MODULE_DESIGN.md)
- [SECURITY_CENTER_DESIGN.md](../designs/SECURITY_CENTER_DESIGN.md)
- [ERROR_CODE_AND_I18N.md](../designs/ERROR_CODE_AND_I18N.md)

### 10.2 Acceptance

- [ACCEPTANCE_CHECKLIST.md](../acceptances/ACCEPTANCE_CHECKLIST.md)

## Refresh rotation: atomic single use（2026-11-04）

The HTTP refresh handler atomically consumes the old Redis mapping with `authtoken.ConsumeRefresh` (GETDEL, Redis 7 deployment baseline) before changing the DB session or issuing a pair. Across instances, at most one request using the same old token may proceed. A replay or consume/storage error never returns a successful pair. `ValidateRefresh` remains read-only for other consumers; exported existing signatures and tenant V1 scope are unchanged.

Once consumed, the old token is not restored on a later database/issuance failure or an uncertain response. The client may need to log in again. Concurrent tabs/retries can receive an unauthorized response; this is the security tradeoff of single use. The reverse index is replaced by normal issuance or cleared during revocation; consumption must not erase a newer index. Revocation, password changes, tenant membership gates and cookie response policy remain required regressions.

Refresh and tenant switching update the session row only while it is unrevoked and unexpired and the observed refresh JTI/tenant remain unchanged. A stale read cannot overwrite a completed revocation, rotation, or tenant switch; conflicts fail as unauthorized. DB and Redis operations remain separate; a later failure does not restore the old refresh token.
