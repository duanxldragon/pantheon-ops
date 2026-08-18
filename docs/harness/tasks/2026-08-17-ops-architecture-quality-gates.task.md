# Task: Architecture And Quality Gates

Task ID: `ops-architecture-quality-gates`  
Priority: P0 closeout / P1 hardening  
Owner: repository quality/harness  
Depends on: all implementation contracts; checker bootstrap may start after capability contract  
Blocks: program completion

## Goal

Convert the audit's architecture and lifecycle invariants into deterministic
checks so that future tools cannot silently reintroduce direct-table access,
unversioned schema, unauthorized targets, duplicate execution, or frontend compile
failures.

This task does not repair product behavior owned by earlier tasks. It adds and
wires gates, contract/smoke coverage, evidence rules, and truthful documentation.

## Required Gates

1. **Business boundary checker**: production runtime code may reference only its
   own tables/Models. Cross-module usage must be an allowlisted typed capability.
   Migration, seed, backup/restore, and isolated fixture paths have narrow,
   documented allowlists and may not be imported by request paths.
2. **Migration gate**: empty database, current upgrade database, repeat run, and
   failure recovery are executed against MySQL 8; production code does not depend
   on untracked AutoMigrate.
3. **Capability contract tests**: stable DTOs, error keys, active/visible scope,
   DataScope, soft delete, batch behavior, and Command conflict semantics.
4. **Business invariants**: concurrent host bind, delete/bind race, immutable
   snapshot, duplicate Start/callback, lease takeover, invalid/stale state change,
   and K8s rollout failure/reconciliation.
5. **Cross-module smoke**: BizScope -> CMDB -> Service -> Deploy and BizScope ->
   K8s Release, including unauthorized/inactive scope and reference deletion.
6. **Frontend/overlay gates**: type-check, build, permission/menu/component/i18n
   registration, loading/error/empty/confirm states, and business overlay contract.
7. **Visual evidence**: user-facing changes require desktop/mobile rendered proof
   or an explicit reviewed runtime gap. Source inspection cannot claim polish.
8. **Docs truth**: current design, API, state, migration, and task evidence links
   match implementation. Suggestions remain labeled as suggestions.

## Why This Is P0/P1

The audit baseline had passing business Go tests but failed frontend type-check
and no architecture/concurrency/cross-module invariant gate. Without automation,
low-cost or stateless tools can produce locally compiling changes that violate
the core ownership rule or deployment safety.

## In Scope

- Add a repository-local checker such as
  `scripts/check-business-module-boundaries.mjs` with unit fixtures under
  `tests/scripts/`. Prefer AST/Go tooling where practical; a text scan must be
  conservative, tested, and allowlist-driven rather than fragile substring magic.
- Add package scripts for business foundation checks and deterministic command
  aggregation. Keep commands runnable locally without CI-only secrets.
- Add provider contract and cross-module integration tests in the owning Go
  packages or a business integration test package. Do not duplicate production
  repositories solely for tests.
- Add/extend API and Playwright business smoke for Deploy, CMDB, BizScope, Service,
  and K8s. Use fake/disposable external dependencies by default.
- Make `npm run type-check` green, including the baseline K8s errors. Product-page
  fixes remain owned/reviewed by the K8s task if they overlap.
- Wire business-specific checks through existing Ops scripts. If a generic shared
  `.github/workflows/quality.yml` change is required, stop and create a base-first
  task; do not fork the inherited workflow locally.
- Add a concise quality-gate matrix to design/task docs and evidence summaries.

## Out Of Scope

- changing lifecycle/product semantics to make a gate pass;
- replacing base CI/harness/visual infrastructure;
- calling production Kubernetes, SSH, or secrets in CI;
- requiring `go test -race ./...` if an unrelated inherited package is known to be
  unsupported without first scoping and recording the gap;
- broad refactors, formatting churn, or new test frameworks;
- treating test quantity as acceptance without invariant coverage.

## Checker Contract

The architecture checker must fail on at least these fixtures:

- Deploy importing BizScope/CMDB GORM models;
- K8s/CMDB containing `Table("biz_business_scope")`;
- BizScope updating `biz_cmdb_host`;
- raw cross-owner table name in a request service;
- consumer contract DTO containing a provider GORM Model or `*gorm.DB`.

It must pass CMDB internal host/group access and explicit migration/seed/test
fixtures. Every allowlist entry names owner, reason, expiry/review condition, and
test proving it is not reachable from production request wiring.

## Expected Files

- create `scripts/check-business-module-boundaries.mjs` (or Go equivalent),
  `tests/scripts/check-business-module-boundaries.test.mjs`, integration tests,
  and business smoke specs;
- modify root `package.json`, business test packages, frontend smoke config/scripts,
  task/design docs, and Ops-specific quality documentation;
- modify `.github/workflows/quality.yml` only through a separate base-first change
  if the shared workflow owns that contract.

Do not touch product services/models except to coordinate with their owning task,
`../pantheon-base/**`, inherited system/platform UI, or external environments.

## Verification Commands

From `pantheon-ops`:

```text
node --test tests/scripts/check-business-module-boundaries.test.mjs
node scripts/check-business-module-boundaries.mjs
npm run check:business-overlay
npm run test:business-overlay
```

From `pantheon-ops/backend`:

```text
go test -count=1 ./modules/business/...
go test -race ./modules/business/...
go test -count=1 ./pkg/database/...
go vet ./...
```

From `pantheon-ops/frontend`:

```text
npm run type-check
npm run build
npm run test:smoke:business:cmdb
npm run test:smoke:business:deploy:api
npm run test:smoke:business:deploy
```

Add new Service/K8s smoke command names to `package.json` and this packet once
implemented; do not leave placeholder commands in a passing aggregate.

## Acceptance Matrix

- Checker has positive/negative fixtures and no unreviewed broad exclusions.
- All child-task invariant tests are linked and runnable by a clean executor.
- The prior five K8s TypeScript errors are gone or superseded by current evidence.
- Cross-module denied-scope cases fail before mutation.
- Migration and race tests use MySQL semantics where required.
- Business overlay, permissions, menu, component, i18n, and audit coverage pass.
- UI changed by the program has rendered desktop/mobile evidence or explicit gaps.
- Each child evidence directory has `commands.json`, `summary.md`, and `review.md`.
- Parent status/evidence reports remaining risks; no checkbox-only completion.

## Cost-Aware Assignment

- Lower-cost models: checker fixtures, manifest/link validation, isolated frontend
  type fixes, deterministic API tests, docs/evidence formatting.
- Strong model/reviewer: allowlist design, race/authorization test adequacy,
  false-positive/false-negative review, and final program acceptance.
- A lower-cost model may not change product state predicates, migration semantics,
  authorization policy, or public capability DTOs while fixing a gate.

## Evidence

Write command metadata, checker fixture output, child invariant results, frontend
and overlay output, visual evidence inventory, findings-first review, and residual
risk to `.harness/evidence/ops-architecture-quality-gates/`.

## Stop Conditions

Stop for a generic shared CI change, an external dependency requiring credentials,
a gate that can pass only by broad exclusion, or a product behavior mismatch. Send
the issue back to the owning child task instead of weakening the gate.
