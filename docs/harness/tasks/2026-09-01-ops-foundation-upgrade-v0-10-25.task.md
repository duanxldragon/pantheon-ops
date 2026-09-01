---
title: Consume foundation release pantheon-base-v0.10.25
doc_type: Remediation
layer: inheritance-sync
status: Active
updated_at: 2026-09-01
linked_contracts:
  - docs/PROJECT_INHERITANCE.md
  - foundation-release.lock.json
  - business-overlay.json
---

# Task Packet: 2026-09-01-ops-foundation-upgrade-v0-10-25

## Goal

Consume the immutable `pantheon-base-v0.10.25` foundation release (operational
workbench contracts B1-B5) and unblock the Ops operational workbench program
(O1-O6). Repair real overlay breakpoints so a deterministic clean rebuild from
the release snapshot reproduces the full V1.2 product tree.

## Scope

### In

- Lock upgrade from v0.10.24 (commit 28fa6ac0) to v0.10.25 (commit
  e7bd9646, per the published release asset manifest).
- Migrate base-owned drift back into legal overlay surfaces:
  - `backend/pkg/metrics/business_metrics.go` + metric vars ->
    `backend/modules/business/metrics` (new overlay package).
  - `database.WithCustomTimeout` ->
    `backend/modules/business/shared.WithCustomTimeout`.
  - global idempotency middleware + `r.Static("/uploads")` from main.go ->
    `backend/modules/business/opsinit` composition root (business-group
    scoped registration).
  - `common.ResponseSuccess/ResponseError/GetUsernameFromContext` shim ->
    canonical `common.Success/FailWithCode` + `c.GetString("username")` in
    filedist/observability handlers.
  - `check-menu-contract.mjs` observability v2 exemption -> dropped (base
    HEAD converged).
- Rebuild `business-overlay.json` from the V1.2 reality: 8 frontend modules,
  29 components, opsinit backend composition, full businessPaths/
  repositoryOverlayPaths/goDependencies/businessSmokeScripts inventory
  (incl. CHANGELOG, e2e-tests, audits, V1_RELEASE_CHECKLIST, sftp/k8s/websocket
  deps, v12locales i18n extraction).
- Rebuilder fixes: preserve ops-only governance scripts in mergePackages;
  decode git core.quotepath octal escapes for non-ASCII overlay paths.
- Staged-tree gates (see Verification) and deterministic rebuild proof.
- Explicit gap recorded for `PANTHEON_MAX_BODY_BYTES` (stays a Base-first
  change; the 10 MB engine-level body limit persists until base ships it).

### Out Of Scope

- Implementing O1-O7 workbench packets (separate follow-up packets).
- Copying files from base/main (only the published release artifact is used).
- Any new business behavior, schema, or permission changes.

## Stop Conditions

- Final baseline replacement is an L2 human gate: the staged tree replaces
  the ops working tree only after maintainer confirmation.
- Stop and record gap if the release asset manifest disagrees with the lock,
  or staged gates fail.

## Verification

Staged tree (`.tmp/business-overlay-rebuild`, rebuilt from the locked
v0.10.25 snapshot):

- `node scripts/business-overlay/check-business-overlay.mjs --root .tmp/business-overlay-rebuild` -> OK (53 declared paths)
- backend: `go mod tidy && go build ./...` -> pass; `go test -count=1 ./modules/business/...` -> 18 ok, 0 fail
- frontend: `npm run type-check` -> pass; `npm run build` (incl. menu-contract, i18n, UI contract prebuild gates: 37 menus / 47 routes / 208 permissions / 49 component keys) -> pass
- determinism: two consecutive rebuilds produce byte-identical `.business-overlay-report.json`
- `npm run check:docs-frontmatter` -> pass (305 docs; 1 base doc defect fixed base-first)
- `npm run check:business-boundaries` / `check:business-foundation` / `check:encoding` / `check:structure` -> pass
- Windows CGO race gate: run after baseline replacement (requires full tree)

Evidence: `.harness/evidence/2026-09-01-ops-foundation-upgrade-v0-10-25/`

## Linkage

- Task ID: `2026-09-01-ops-foundation-upgrade-v0-10-25`
- Release: `pantheon-base-v0.10.25` (asset checksum
  `c8b8a3df8701795c1f1b4d7f97a13175182b1c6ed924515e615fb972ff3c50df`,
  repo.tar sha256
  `db4173819341e0b4c32c44a3345b5119b744da38ed18e227f1afc4a15b55a0b5`)
- Unblocks: `2026-08-31-ops-operational-workbench-design` O1-O6
