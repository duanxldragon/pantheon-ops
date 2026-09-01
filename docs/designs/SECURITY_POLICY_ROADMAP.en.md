---
title: Security Policy Deepening Roadmap
doc_type: Design
layer: system/auth
status: Active
linked_contracts:
  - docs/contracts/SYSTEM_AUTH_CONTRACT.md
updated_at: 2026-05-19
---

# Security Policy Deepening Roadmap

Chinese version: [SECURITY_POLICY_ROADMAP.md](./SECURITY_POLICY_ROADMAP.md)

This roadmap tracks the next security policy increments for `system/auth`.

MFA/TOTP is already part of the implemented baseline and should no longer be treated as a missing capability. The next priorities are login alerts, password policy hardening, and risk-control governance.

## Current Baseline

Implemented capabilities include:

- Local username/password sign-in.
- Access and refresh tokens.
- Session management.
- Security Center.
- Recent login history.
- Password change.
- MFA/TOTP.
- Step-up verification for sensitive actions.
- Basic lockout after repeated login failures.
- Password minimum length plus configurable digit and uppercase complexity rules.
- Source-level throttling and temporary lockout.
- Security event persistence and back-office querying for real authentication risks such as source lockouts and account lockouts.
- Password history reuse limits through `security.password_history_limit`.
- Password expiration reminders through `security.password_expire_days`, including visible expiration status in the Security Center.

## P1 Follow-up

### Login Alerts

Already implemented:

- Source lockouts and account lockouts are written to `system_auth_security_event`.
- The back-office security event page supports querying by account, event type, and risk level.
- No fake "email sent" or "SMS sent" state is shown before a real notification channel is connected.

Next enhancements:

- New-device login alerts.
- Unusual geo-location or IP login alerts.
- High-frequency login failure alerts.

Acceptance:

- Alert events are persisted in audit or security-event storage.
- The frontend Security Center can display recent risk reminders.
- No pseudo notification-delivery state is shown before the channel exists.

### Password Policy

Already implemented:

- Minimum length.
- Configurable digit and uppercase complexity switches.
- Password history reuse limits.
- Password expiration reminders.

Acceptance:

- Policy values come from `system/config`, while validation semantics stay in `system/auth`.
- The backend returns stable error keys.
- The frontend only renders policies that are actually enforced.

### Risk-Control Rules

Already implemented:

- Failure throttling and temporary lockout at the request-source dimension.
- Risk-control decisions are auditable as security events.

Next enhancements:

- Rate limiting by user, IP, and device fingerprint.
- Mandatory MFA for high-risk logins.
- Step-up verification for highly sensitive actions.

Acceptance:

- Risk-control decisions remain auditable.
- User-facing prompts do not leak excessive policy detail.
- Administrators can explain why a request was blocked.

## SSO Boundary

SSO, OAuth2, and OIDC are not part of the currently implemented capability set.

Until identity-source ownership, callback domains, and external identity binding policy are finalized:

- Do not show pseudo SSO buttons on the login page.
- Do not land half-wired provider tables in the backend without real usage.
- Complete design review first through [SSO_OIDC_DESIGN.md](./SSO_OIDC_DESIGN.md).
