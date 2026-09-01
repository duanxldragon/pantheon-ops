---
title: Error Code and I18n Design
doc_type: Design
layer: platform
status: Active
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
  - docs/contracts/SYSTEM_AUTH_CONTRACT.md
updated_at: 2026-04-28
---

# Error Code and I18n Design

Chinese version: [ERROR_CODE_AND_I18N.md](./ERROR_CODE_AND_I18N.md)

This document defines the responsibility boundary between backend error semantics and frontend language rendering in Pantheon Base.

It answers:

- whether the backend returns natural language or stable keys
- whether frontend toast messages should be translated first
- who owns success and failure copy
- what fallback behavior is required
- who must add i18n keys when a new module is introduced

## 1. Design Goals

- the backend owns stable error semantics
- the frontend owns final display language
- error keys and i18n keys should converge
- success and failure feedback should follow one strategy
- fallback behavior must prevent UI breakage when translations are missing

## 2. Unified Response Principle

The current response shape is:

```go
type Response struct {
    Code    int
    Data    interface{}
    Message string
}
```

Interpretation:

- `Code`: business status code
- `Data`: business payload
- `Message`: must be treated primarily as an i18n key

## 3. Backend Return Rules

### 3.1 The Backend Does Not Return Natural Language by Default

The backend should not directly return:

- Chinese user-facing sentences
- English user-facing sentences
- final terminal-display copy

It should return stable keys such as:

- `param.invalid`
- `permission.denied`
- `user.login.error.not_found`
- `user.role.required`
- `refresh_token.invalid`

### 3.2 Backend Errors Use Three Semantic Layers

- `platform`: infrastructure-level failures
- `domain`: business-rule failures
- `security`: authentication and authorization failures

### 3.3 Backend Business Code Rules

Current short-term codes remain:

- `200`: success
- `400`: invalid parameters
- `401`: unauthenticated or authentication expired
- `403`: forbidden
- `500`: generic failure

Even if finer code systems appear later, the rule that `Message` is a key must not be broken.

## 4. Frontend Display Rules

### 4.1 The Frontend Must Translate `message`

Given:

```json
{
  "code": 403,
  "message": "permission.denied"
}
```

The frontend must display:

```ts
t("permission.denied");
```

### 4.2 Current Runtime Behavior

The request layer already treats `message` as a key first.

Current rules:

- if the backend `message` resolves to a known i18n key, translate it directly
- if the backend returns a non-key transport string, fall back to a frontend base key rather than exposing raw English fallback copy
- network, timeout, and browser-level errors map to stable frontend keys
- development mode may append raw transport text for debugging, but production must not expose it directly

### 4.3 Correct Behavior

The request layer should:

1. translate `message` when it is a key
2. use a fallback if the key is missing
3. use default frontend fallback keys for network or non-standard errors

Backend handlers should also avoid passing framework or database error strings directly through to the client.

## 5. Success Message Rules

### 5.1 Success Feedback Is Primarily Controlled by the Frontend

The frontend understands page context best, so it should usually own success toasts such as:

- `common.createSuccess`
- `common.updateSuccess`
- `common.deleteSuccess`

### 5.2 Backend Success `message`

The backend may still return `message: "success"`, but the frontend should not rely on that value as the main toast source.

## 6. I18n Key Naming Rules

### 6.1 Common Keys

Examples:

- `common.*`
- `auth.*`
- `permission.*`

### 6.2 Module-Level Keys

Examples:

- `system.user.*`
- `system.role.*`
- `system.menu.*`
- `system.permission.*`
- `auth.session.*`
- `biz.order.*`

Constraint:

- runtime translation uniqueness in `system_i18n` should be enforced by `locale + key`
- `module` is for ownership, filtering, and export boundaries, not for runtime uniqueness

### 6.3 Error Keys

Recommended pattern:

```text
{module}.{action}.error.{reason}
```

### 6.3.1 Common Error Keys

Shared keys include:

- `success`
- `param.invalid`
- `permission.denied`
- `permission.escalation.forbidden`
- `permission.engine.not_initialized`
- `database.not_initialized`
- `network.error`
- `request.failed`

## 7. Fallback Rules

### 7.1 Frontend Translation Fallback

Missing keys must degrade to stable fallback behavior rather than raw broken UI output.

### 7.2 Network Error Fallback

Network and timeout failures should resolve to stable frontend keys, not to transport-level raw text.

### 7.3 Unknown Error Fallback

Unknown failures should converge to a controlled fallback key rather than leaking internal stack semantics.

## 8. Backoffice I18n Governance Capabilities

### 8.0 New Locale Admission and Expansion Flow

New locales should only be added for real market, delivery, or compliance reasons, and must pass the existing import/export, missing-key, and fallback validation chain.

### 8.1 Duplicate Key Conflicts

Key conflicts are governance failures and must be normalized to canonical records.

### 8.2 Unused Keys

Unused translations should be detectable and governable rather than silently accumulating forever.

### 8.3 Module-Based Export

Assets should still be exportable by module ownership even though runtime uniqueness is not module-scoped.

### 8.4 Conflict-Repair Assistance

The governance layer should help operators find and repair conflicts safely.

### 8.5 Long-Lived Placeholder Warnings

Long-term placeholder translations should be detectable as quality debt.

### 8.6 Key-Rename Workflow

Key renames should be explicit, previewable, and traceable.

### 8.7 Generator and Frontend Source Anti-Regression

Generated code and frontend source should stay key-first and be checked against hardcoded-copy regressions.

## 9. Frontend and Backend Responsibility Boundary

### 9.1 Backend Owns

- stable business and security semantics
- stable keys
- consistent response structure

### 9.2 Frontend Owns

- translation
- fallback rendering
- toast or page-level presentation

### 9.3 Modules Own

- module-specific keys
- error coverage
- translation completeness for new behaviors

## 10. Error Presentation Rules

### 10.1 Toast

Use toast for lightweight action feedback, but keep messages translated and stable.

### 10.2 Form-Field Errors

Validation errors that belong to fields should stay near their fields.

### 10.3 Page-Level Errors

Page-wide failure states should use explicit page-level components rather than generic toast spam.

### 10.4 Empty-State Hints

Empty-state copy must also follow the i18n chain.

## 11. Request-Layer Rules

The request layer is the convergence point for key translation, fallback logic, network error normalization, and authentication-expiry handling.

## 12. Login and Authentication Error Rules

Auth-related failures such as user-not-found, password mismatch, MFA challenge expiry, and session invalidation must resolve through stable keys and consistent frontend rendering.

## 13. Permission Error Rules

Permission denials should be normalized through consistent forbidden keys and page-level or action-level UI treatment.

The current permission key set should also include `permission.escalation.forbidden` for guarded management-policy writes.

## 14. What Every New Module Must Add

Every new module must provide:

- its i18n namespace
- its user-visible keys
- its error keys
- its success-feedback strategy

## 15. Current Delivery Gaps

The main remaining gap is not the response structure itself, but full enforcement and cleanup across all modules so that no natural-language or mixed-style leakage remains.

## 16. Acceptance Checklist

Acceptance should verify:

- backend returns stable keys
- frontend translates keys before display
- fallback rules work
- success and failure messages stay consistent
- new modules include their translation and error coverage

## 17. Recommended Next Document

Continue with module contracts and related frontend i18n implementation rules.
