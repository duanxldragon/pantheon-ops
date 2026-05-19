# Contributing

Chinese version: [CONTRIBUTING.md](./CONTRIBUTING.md)

This document defines the baseline collaboration, commit, and verification rules for `pantheon-ops`.

## Layer Ownership

Every change should first identify its layer:

- `platform`
- `system/auth`
- `system/iam`
- `system/org`
- `system/config`
- `business/*`

Cross-layer changes should explain boundaries, dependencies, and verification in the PR description.

## Commit Format

Use Conventional Commits:

```text
type(scope): subject
```

Suggested scopes include:

- `platform`
- `system-auth`
- `system-iam`
- `system-org`
- `system-config`
- `system-audit`
- `business-cmdb`
- `frontend`
- `backend`
- `docs`
- `ci`
- `tests`

## Local Setup

```bash
git config commit.template .gitmessage
git config core.hooksPath .githooks
```

This enables the shared commit template and commit-message validation hook.

## Verification

Run the minimum set that matches your change area, including backend tests, frontend build, and relevant smoke suites.

Representative commands:

```bash
go test ./backend/modules/auth ./backend/modules/system/...
cd frontend
npm run build
npm run test:smoke:system
npm run test:smoke:role-auth
npm run test:smoke:impexp
npm run test:smoke:backoffice-ui
```

## PR Expectations

- explain the ownership layer and any cross-domain dependency
- keep response shape and permission boundaries intact
- localize new user-facing text
- update docs when required
- run tests that match the change scope

The Chinese source document remains the authoritative detailed checklist.
