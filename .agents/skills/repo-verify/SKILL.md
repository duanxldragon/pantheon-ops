---
name: repo-verify
description: Use when finishing a Pantheon Ops change or choosing the minimum local verification matrix for business-scope or base-sync work before commit or PR
---

# Repo Verify

In Pantheon Ops, inheritance checks are part of verification. Do not verify business code while ignoring base-sync drift.

## Use This Matrix

- Inheritance or shared-path sync work:
  - `npm run check:inheritance`
  - `npm run check:base-sync` when frontend or shared business mounts changed
- Docs or root governance scripts:
  - `npm run check:docs-frontmatter`
  - `npm run check:task-packet-template`
  - `npm run check:generated-modules` when generated-module files changed
- Frontend code, menus, i18n, routes, shared business integration:
  - `cd frontend`
  - `npm run check:base-sync`
  - `npm run lint`
  - `npm run type-check`
  - `npm run build`
- Backend Go code:
  - `go test -race ./...`
- System-domain inherited pages or authz behavior:
  - `cd frontend`
  - `npm run test:smoke:system`
  - use `npm run test:smoke:role-auth` or `npm run test:smoke:impexp` when the scope is narrower
- Platform shell or shared UI behavior:
  - `cd frontend`
  - `npm run test:smoke:backoffice-ui`
- Ops business modules:
  - `cd frontend`
  - `npm run test:smoke:business:cmdb` for `business/cmdb`
  - `npm run test:smoke:business:deploy` for `business/deploy`
  - `npm run test:smoke:business` when multiple business modules or generated business-runtime flows changed
- Generator or module-governance behavior:
  - `cd frontend`
  - `npm run test:generator:smoke`
  - add `npm run test:smoke:generator-workbench` or `npm run test:smoke:module-governance-host` when that runtime changed
- Security-sensitive dependency or secret work:
  - `go run golang.org/x/vuln/cmd/govulncheck@latest ./...`
  - `npm audit --registry=https://registry.npmjs.org --audit-level=moderate`
  - `cd frontend && npm audit --registry=https://registry.npmjs.org --audit-level=moderate`

## Hard Rules

- If the change may belong in `pantheon-base`, state that landing-side decision before claiming verification is sufficient.
- If UI behavior changed, attach rendered evidence or state the runtime gap explicitly.
- Record exact commands and outcomes.
