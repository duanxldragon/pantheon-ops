# Security Policy

Chinese version: [SECURITY.md](./SECURITY.md)

## Supported Scope

Current priority scope includes:

- `platform`
- `system/auth`
- `system/iam`
- `system/org`
- `system/config`

If a report concerns `business/*`, specify the concrete business module.

## Reporting a Vulnerability

Do not disclose the following in a public issue:

- admin credentials or verification codes
- access or refresh tokens
- database DSNs, Redis passwords, or third-party secrets
- sensitive payloads that make exploitation directly reproducible

Preferred reporting flow:

1. contact the maintainers through email or a private channel
2. state the impacted layer: `platform`, `system/*`, or `business/*`
3. provide minimal repro steps, impact scope, and supporting logs or screenshots
4. if the issue is an authorization bypass, specify whether it breaks navigation, page, action, or API authorization

## Response Expectations

- issue ownership and impact boundaries will be confirmed first
- high-risk auth, authorization, and sensitive configuration issues are prioritized
- related tests and docs should be updated after the fix to prevent regression
