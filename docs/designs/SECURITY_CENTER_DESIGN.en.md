---
title: Security Center Design
doc_type: Design
layer: system/auth
status: Active
linked_contracts:
  - docs/contracts/SYSTEM_AUTH_CONTRACT.md
updated_at: 2026-04-17
---

# Security Center Design

Chinese version: [SECURITY_CENTER_DESIGN.md](./SECURITY_CENTER_DESIGN.md)

This document defines the Pantheon Base security center as the main entry point for account security, session state, and login-risk visibility after auth-domain separation.

The security center is not just “a password form inside profile”. It is the entry point where users and administrators understand session health, login evidence, and security posture.

## 1. Design Goals

- separate authentication and security capability from `user/profile`
- let current users manage their own sessions and password
- let administrators inspect login evidence and security activity
- keep room for MFA, source throttling, password history, password expiry, SSO, and deeper risk controls
- preserve a restrained, clear, and credible enterprise-admin feel

## 2. Capability Boundary

The security center belongs to the `auth` domain.

It owns:

- current session
- online devices or sessions
- password changes
- login logs
- session revocation
- security hints
- future login-risk events

It does not own:

- user CRUD
- role authorization
- menu configuration
- business data permissions

## 3. Information Architecture

Suggested structure:

- security overview
- online sessions
- login logs
- password security
- future security policy area

## 4. Page Planning

### 4.1 Current-User Security Center

Suggested route:

- `/auth/security`

Uses profile-like and configuration-like page patterns.

Features:

- view account security status
- change password
- inspect active sessions
- kick other devices
- inspect recent login records

### 4.2 Online Sessions Page

Suggested route:

- `/auth/sessions`

Uses a list-page pattern.

Shows:

- current session marker
- login IP
- user agent
- last refresh time
- refresh-token expiry
- session creation time
- revoke state

### 4.3 Login Log Page

Suggested route:

- `/system/login-log`

Current users see their own logs. Administrators may inspect the global login-log view.

## 5. Frontend Page Structure

### 5.1 `SecurityCenter`

Recommended structure:

- `SecurityOverviewCard`
- tabbed panels
- `SecurityTips`

### 5.2 `SecurityOverviewCard`

Should show:

- current account
- recent login time
- current-session status
- future password-update time
- MFA-policy state

### 5.3 `SessionsPanel`

Shows:

- current device
- other devices
- last activity
- revoke action

The current device should not be directly deleted from here; it exits through logout.

### 5.4 `PasswordPanel`

Shows:

- current password
- new password
- confirm new password
- password-rule guidance

Rule:

- password change should revoke other sessions
- the current-session behavior must be explicit

Current recommendation:

- keep the current session valid
- revoke the others automatically

## 6. API Design

### 6.1 Current-User Security APIs

Suggested APIs:

- `GET /api/v1/auth/security`
- `GET /api/v1/auth/sessions`
- `DELETE /api/v1/auth/sessions/:id`
- `PUT /api/v1/auth/password`
- `GET /api/v1/auth/login-logs`
- `POST /api/v1/auth/mfa/verify`

### 6.2 Administrator Security-Audit APIs

Suggested APIs:

- login-log list
- login-log export
- retention-window cleanup
- batch delete with secondary confirmation
- global session list
- admin session kickout

Rules:

- full-table log clearing is no longer the default capability
- dangerous actions should converge on retention-window cleanup or selected-set deletion
- sensitive admin operations must use the shared secondary-confirmation path

## 7. Data Model

### 7.1 Current Existing Tables

Already available:

- `system_user_session`
- `system_log_login`
- `system_auth_factor`
- `system_auth_mfa_challenge`

### 7.2 `system_user_session`

Supports fields such as session ID, user ID, refresh JTI, refresh expiry, and refresh timestamps.

### 7.3 `system_log_login`

Carries login evidence for both self-service and administrator views.

## 8. Permission Design

### 8.1 Current-User Self-Service Permissions

Current-user security surfaces should be accessible as self-service auth capabilities rather than as role-gated admin-only pages.

### 8.2 Administrator Permissions

Administrative login-log and global-session inspection require dedicated management permissions.

## 9. Menu Design

Security-center entry points should exist in the user area and related self-service surfaces, while administrator audit pages continue through system governance navigation.

## 10. I18n Key Planning

Security-center labels, action text, errors, warnings, and status summaries must all remain key-driven and translatable.

## 11. UI Design Requirements

The security center should feel like a controlled, credible admin-security surface rather than a dense settings junk drawer.

## 12. State Design

The page must handle loading, empty, error, action-in-flight, and forbidden states explicitly.

## 13. Audit Rules

Security-sensitive actions must preserve operator, time, result, and summary evidence with the right distinction between self-service and administrator actions.

## 14. Phased Implementation

### 14.1 Phase 1

Reuse existing tables and establish the user-facing security center baseline.

### 14.2 Phase 2

Strengthen administrator audit views.

### 14.3 Phase 3

Deepen security policy exposure and related controls.

## 15. Relationship with Profile Center

The security center is separate from the general profile center. Identity and session security should not be collapsed back into one generic profile form.

## 16. Current Delivery Gap

The main remaining gap is continuing to evolve from the current closed loop into deeper auth-scale and risk-control capabilities without blurring the auth and IAM boundaries again.

## 17. Acceptance Checklist

Acceptance should verify:

- session visibility and revoke flows
- password change behavior
- login-log visibility by role
- MFA-related security status
- security-related audit traces

## 18. Recommended Next Document

Continue with the provider abstraction, security policy roadmap, and related auth-domain acceptance materials.
