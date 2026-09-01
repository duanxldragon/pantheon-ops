---
title: Authentication Provider Abstraction Design
doc_type: Design
layer: system/auth
status: Active
linked_contracts:
  - docs/contracts/SYSTEM_AUTH_CONTRACT.md
  - docs/contracts/SYSTEM_IAM_CONTRACT.md
updated_at: 2026-05-18
---

# Authentication Provider Abstraction Design

Chinese version: [AUTH_PROVIDER_ABSTRACTION.md](./AUTH_PROVIDER_ABSTRACTION.md)

This document defines the provider-abstraction boundary that `system/auth` must establish before Pantheon Base implements real SSO, OIDC, LDAP, or enterprise identity-provider integrations.

The goal is not to support a live external provider immediately. The goal is to make Pantheon Base:

- `sso-ready`
- `captcha-ready`
- `risk-ready`

without exposing fake capabilities before a provider is actually integrated.

## 1. Design Goals

This design solves four things:

1. separate local login from external-identity login clearly
2. reserve one shared provider abstraction instead of building custom branching logic per provider
3. define external-identity binding, audit semantics, and fallback strategy
4. prevent misleading controls in frontend or backend before a provider is real

## 2. Non-Goals

Not in scope now:

- real OIDC, OAuth2, LDAP, WeCom, DingTalk, or other provider integration
- federated logout
- automatic master-data synchronization
- complex identity-merge strategies
- removal of local-login fallback

## 3. Abstraction Boundary

### 3.1 What `system/auth` Owns

- login-entry orchestration
- local-login fallback
- provider status exposure
- external-identity binding reads
- login audit and risk-event traces

### 3.2 What `system/auth` Does Not Own

- the organization model of an external identity source
- `system/org` governance
- rewrites of `system/iam` user, role, or permission semantics
- fake SSO flows before a provider is actually connected

## 4. Provider Abstraction Model

### 4.1 Provider Types

Recommended minimum enum:

- `local`
- `oidc`
- `oauth2`
- `ldap`
- `saml`
- `custom`

The only truly available type right now is:

- `local`

All other types may exist only as readiness metadata until actual integration exists.

### 4.2 Provider State

Recommended minimum states:

- `disabled`
- `configured`
- `ready`
- `error`

Meaning:

- `disabled`: not enabled or not configured
- `configured`: configuration exists, but runtime connectivity or login entry is not ready
- `ready`: real login is available
- `error`: configuration or runtime status is broken

A provider must not be marked `ready` without real provider code behind it.

## 5. External Identity Binding Model

Recommended unified binding fields:

- `provider_type`
- `provider_key`
- `external_subject`
- `external_account`
- `local_user_id`
- `binding_status`
- `last_login_at`
- `last_sync_at`
- `created_at`
- `updated_at`

### 5.1 Binding Semantics

- one local user may bind to multiple external identities
- one external identity inside the same provider may bind to only one local user
- binding is an authentication mapping, not the source of authorization

### 5.2 Permission Semantics

- roles, permissions, menus, and data scope still come from local `system/iam`
- external identity may tell Pantheon who the subject is, but it must not rewrite Pantheon’s internal authorization model directly

## 6. Login Orchestration

### 6.1 Current Default Path

The current formal login path must remain:

- local account + local password

### 6.2 Future External Login Path

When a provider becomes real, the suggested flow is:

1. user chooses the provider
2. redirect to the external identity source
3. callback returns to `system/auth`
4. resolve binding by `external_subject`
5. issue a Pantheon session for the mapped local user
6. write audit and risk-event traces

### 6.3 Local Fallback

Whatever future provider is integrated, Pantheon must retain:

- local admin fallback login
- a local recovery entry when the provider fails

Do not convert login into “external only” without a controllable local rescue path.

## 7. Configuration Model

Security-center or settings surfaces must expose only real status, not fake features.

Allowed configuration exposure:

- provider type
- whether configuration exists
- whether the login entry may be shown
- current status
- error summary
- whether local fallback is enabled

Not allowed before integration:

- clickable SSO buttons with no real flow
- CAPTCHA switches that save but have no backend meaning

## 8. Audit and Risk Events

Provider-related actions should record at least:

- provider type
- provider key
- summarized external-subject identity
- local user ID
- login result
- source IP, UA, and timestamp

Suggested minimum risk-event types:

- `provider-login-success`
- `provider-login-failed`
- `provider-binding-missing`
- `provider-callback-invalid`
- `provider-disabled-attempt`

## 9. Frontend Presentation Constraints

Before a provider is real:

- the login page must not show a fake primary SSO entry
- the security center may only show readiness information and explanatory text
- any future capability indicator must be explicitly labeled as disabled, reserved, or not integrated

## 10. Definition of Done

The first phase of `auth-scale` is complete when:

- there is a provider-abstraction document
- a binding model is defined
- local-login fallback semantics are explicit
- audit and risk-event reservations exist
- neither frontend nor backend exposes fake provider capability

## 11. Preconditions Before Real Implementation

Before implementing a real provider, Pantheon must first settle:

- target identity-source type
- callback-domain strategy
- account auto-create versus manual binding strategy
- admin fallback strategy
- logout and session-invalidity semantics
- failure-switch and rollback strategy
