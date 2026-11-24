---
name: repo-verify
description: Use when finishing a Pantheon Ops change or choosing the minimum local verification matrix for business-scope or base-sync work before commit or PR
---

# Repo Verify

In Pantheon Ops, the consumer tree is rebuilt from a locked Base snapshot. Do not verify business code while ignoring the business-overlay contract.

## Choose the Verification Phase

During iteration, run the affected package/page checks and regression counterexamples. At final delivery, apply the matrix below and repository-required gates. Reuse passing evidence only when it matches the same candidate and relevant configuration; rerun affected checks after edits. Narrow iteration results never replace required full checks.

Before runtime-sensitive verification, record candidate/binary identity, isolated DB/Redis fixtures, actual provider availability and resource readiness. A timeout is a failure/gap to diagnose, not a pass; change the hypothesis before repeating an unchanged run. Keep the environment boundary in the current task summary.

For method/entrypoint work run `node ../pantheon-harness/scripts/harness/check-method-consumption.mjs --root . --source ../pantheon-harness`. Update a pin only after review. For changed manifests explicitly regenerate the Unreleased task-index helper with `node ../pantheon-harness/scripts/harness/render-task-index.mjs --root . --write`.

## Use This Matrix

- Business overlay, manifest, or rebuild work:
  - `npm run check:business-overlay`
  - `npm run test:business-overlay`
  - `npm run rebuild:from-base -- --target .tmp/verify-overlay` when the manifest or business source changed; validate the staged tree before replacement
- Docs or root governance scripts:
  - `npm run check:docs-frontmatter`
  - `npm run check:task-packet-template`
  - `npm run check:generated-modules` when generated-module files changed
- Frontend code, menus, i18n, routes, shared business integration:
  - `cd frontend`
  - `npm run lint`
  - `npm run type-check`
  - `npm run build`
- Backend Go code (from `backend/`):
  - `go test -race ./...`
  - On Windows enable CGO and prepend the approved MinGW toolchain before the full race gate.
- System-domain inherited pages or authz behavior:
  - `cd frontend`
  - `npm run test:smoke:system`
- Platform shell or shared UI behavior:
  - `cd frontend`
  - `npm run test:smoke:platform`
- Ops business modules:
  - `cd frontend`
  - `npm run test:smoke:business:cmdb` for `business/cmdb`
  - `npm run test:smoke:business:deploy` for `business/deploy`
  - `npm run test:smoke:business` when multiple business modules or generated business-runtime flows changed
- Generator or module-governance behavior:
  - `cd frontend`
  - Select the current generator/dynamic-module scripts from `frontend/package.json`; record the actual commands and inherited contract.
- Security-sensitive dependency or secret work:
  - `go run golang.org/x/vuln/cmd/govulncheck@latest ./...`
  - `npm audit --registry=https://registry.npmjs.org --audit-level=moderate`
  - `cd frontend && npm audit --registry=https://registry.npmjs.org --audit-level=moderate`

## Hard Rules

- If the change may belong in `pantheon-base`, state that landing-side decision before claiming verification is sufficient.
- If UI behavior changed, attach rendered evidence or state the runtime gap explicitly.
- Record exact commands and outcomes.
