---
title: Auth Module Split Design
doc_type: Design
layer: system/auth
status: Active
linked_contracts:
  - docs/contracts/SYSTEM_AUTH_CONTRACT.md
updated_at: 2026-04-29
---

# Auth Module Split Design

Chinese version: [AUTH_MODULE_DESIGN.md](./AUTH_MODULE_DESIGN.md)

This document defines how authentication and session concerns are split out from the mixed `system/user` implementation into a dedicated `system/auth` capability domain.

## 1. Design Goals

- make `auth` responsible only for identity verification and session validity
- keep `iam` responsible for users, roles, menus, and permissions
- preserve room for MFA, captcha, SSO, device governance, security policy, and future tenant login
- separate logic boundaries first and continue implementation cleanup afterward

## 2. Split Decision

The split is required.

But the current strategy is:

- split capability boundaries first
- split code layout accordingly
- do not split into microservices yet
- do not force physical table splitting in the short term

## 3. Current Problems

The mixed `system/user` area historically carried:

- login
- refresh rotation
- logout
- session management
- current-user profile
- password change
- login logs
- user CRUD
- role binding

That creates long-term security and maintainability problems, and it encourages AI or new engineers to keep putting auth logic back into the wrong place.

## 4. Target Module Boundaries

Recommended capability split:

- `auth`: authentication and session security
- `iam`: identity and authorization management
- `org`: organizational structure
- `i18n`: language assets
- `audit`: operation and login evidence
- `config`: dictionaries and settings

### 4.1 What `auth` Owns

`auth` owns:

- login
- refresh rotation
- logout
- current-session reads
- session revocation
- current-user password changes
- login logs
- security policy such as password rules, login failure limits, captcha, and MFA

The current runtime policy surface already includes:

- password minimum length and complexity
- password history and expiry
- account-level failed-login thresholds and lock duration
- source or IP-level failed-login thresholds, windows, and locks
- security-event recording switches
- idle session timeout
- maximum active sessions per user
- historical session retention
- real TOTP MFA enablement
- reserved switches for captcha and SSO

User creation and admin password reset in `system/iam` already consume shared password-policy settings to stay consistent.

### 4.2 What `auth` Does Not Own

`auth` does not own:

- user CRUD
- role CRUD
- menu CRUD
- permission CRUD
- department or post maintenance

Those belong to `iam` or `org`.

## 5. Recommended Directory Structure

### 5.1 Backend

Keep `backend/modules/auth/` as a thin module root. The root owns only `module.go`; concrete auth capabilities live in cohesive subdirectories:

- `login/`: login handlers, login runtime facade, login DTOs, login logs, and preference-facing auth endpoints
- `session/`: session lifecycle, refresh, revocation, admin session queries, and session DTO/model helpers
- `mfa/`: TOTP challenge lifecycle, factor persistence, MFA crypto, and MFA DTO/model helpers
- `security/`: password policy, operation-token issuance, security overview, password history, and security events

User-management logic remains in the `system/iam` area.

### 5.2 Frontend

Keep auth-specific screens under `frontend/src/modules/auth/` with the same capability split: `login/`, `session/`, `mfa/`, and `security/`. The auth module root keeps only module registration, shared auth styles, and exports.

## 6. API Boundary Recommendations

### 6.1 `auth` Domain APIs

Recommended `auth` routes include:

- `/api/v1/auth/login`
- `/api/v1/auth/refresh`
- `/api/v1/auth/logout`
- `/api/v1/auth/me`
- `/api/v1/auth/me/preferences`
- `/api/v1/auth/sessions`
- `/api/v1/auth/sessions/:id`
- `/api/v1/auth/password`
- `/api/v1/auth/security`

Boundary notes:

- the login page may expose language switching, but that is a platform session concern rather than a long-term auth preference write
- `PUT /api/v1/auth/me/preferences` remains the persistence point for shell-level user preferences, while the platform shell resolves them with the priority `login-session choice > user preference > system default`

### 6.2 `system/iam` Domain APIs

The following stay in `system/iam`:

- `/api/v1/system/user/*`
- `/api/v1/system/role/*`
- `/api/v1/system/menu/*`
- `/api/v1/system/permission/*`
- `/api/v1/system/dept/*`
- `/api/v1/system/post/*`

## 7. Data Model Recommendations

### 7.1 Existing Tables That Can Stay

The current phase may keep:

- `system_user`
- `system_user_session`
- `system_log_login`

But logical ownership changes:

- `system_user` belongs to `iam`
- `system_user_session` belongs to `auth`
- `system_log_login` belongs to `audit/auth`

### 7.2 Future-Ready Tables

Recommended future entities include:

- `system_auth_policy`
- `system_auth_factor`
- `system_auth_mfa_challenge`
- `system_user_device`
- `system_login_risk_event`
- `system_user_external_identity`
- `system_auth_provider`

### 7.2.1 Current TOTP MFA Implementation

`login.mfa_enabled` is already a real runtime feature.

Current behavior:

- when disabled, `/api/v1/auth/login` issues tokens directly after username-password verification
- when enabled, successful password verification returns an MFA challenge instead of tokens
- users with a bound factor continue with `POST /api/v1/auth/mfa/verify`
- users without a bound factor receive a TOTP secret and `otpauth://` URI for first-time enrollment
- MFA secrets are encrypted with an AES-GCM key derived from `PANTHEON_MFA_SECRET`

#### 7.2.1.1 Main Login Sequence

The live sequence is:

1. the user opens `/login`
2. submits username and password
3. the frontend calls `POST /api/v1/auth/login`
4. the backend either issues tokens directly or returns an MFA challenge
5. the frontend completes verification if MFA is required
6. the backend issues the session only after successful verification

#### 7.2.1.2 TOTP Parameters and Constraints

- challenges are short-lived
- verification uses a six-digit code
- the backend must not fake second-factor behavior by echoing a current code

#### 7.2.1.3 End-User Usage

Users bind an authenticator app, then submit the live TOTP code during MFA verification.

#### 7.2.1.4 Frontend Interaction Constraints

The UI must distinguish:

- password failure
- MFA required
- challenge expiry
- code failure
- first-time factor binding

#### 7.2.1.5 Operations and Security Notes

Production must explicitly configure the MFA secret. The flow is designed to avoid locking out administrators simply because MFA was enabled before factors were bound.

### 7.3 Reservation for SSO and External Identity

The split keeps space for future identity-provider integrations and external account mapping.

### 7.4 Current Code State and Future Entry Points

The base already has a physical `auth` module and a real MFA main path. Future work should keep extending this boundary rather than sliding concerns back into user management.

### 7.5 Suggested Future SSO Flow

SSO should later plug into the `auth` domain rather than into `iam` CRUD flows.

### 7.6 First-Bind and Account-Mapping Strategy

External identity binding and account mapping should remain explicit auth-domain concerns, with auditable rules and safe first-login behavior.

### 7.7 Frontend Design Constraints

Auth screens should keep a focused, security-console UX rather than turning into marketing pages or generic profile pages.

### 7.8 Audit, Logout, and Risk-Control Constraints

Session revocation, login logging, and security-event recording must stay explicit and auditable.

### 7.9 Conclusion

The architectural direction is stable: `auth` is now a real capability boundary, not just a naming suggestion.

## 8. Frontend Page Planning

### 8.1 Auth Entry Pages

Login and MFA verification belong to the dedicated auth experience.

### 8.2 Security Center

The security center is the primary self-service security surface for the current account.

### 8.3 Account Profile

Profile remains a user-facing self-service page, but identity verification and session security should still point back to `auth`.

## 9. Phased Refactor Plan

### Phase 1: Lock the Boundary in Documentation

Define the domain split first.

### Phase 2: Backend Logic Split

Continue relocating auth logic to the dedicated module boundary.

### Phase 3: Frontend Module Split

Keep auth screens and APIs under the auth module.

### Phase 4: API Closure

Converge old and new paths toward the proper auth domain.

## 9.1 Current Implementation State

The physical auth module and TOTP MFA chain already exist. The next work is about deepening the boundary rather than proving the split from scratch.

## 10. Design Decisions

### Must Keep

- auth and IAM responsibilities separate
- MFA as a real runtime flow
- shell preference semantics coordinated with platform, not mixed into auth
- future SSO and risk-control expansion points inside `auth`

### Not Doing Yet

- microservice split
- forced table split for all legacy storage
- fake placeholder capabilities for SSO or captcha

## 11. Related Documents

Continue with the linked system-auth contract and related frontend, security, and acceptance materials.
