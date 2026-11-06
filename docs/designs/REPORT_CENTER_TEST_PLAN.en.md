---
title: Report Center Test Plan
doc_type: Design
layer: platform
status: Draft
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
  - docs/contracts/SYSTEM_IAM_CONTRACT.md
  - docs/contracts/TENANT_CONTRACT_V1.md
updated_at: 2026-11-04
---

# Report Center Test Plan

> 未实现声明：本文为设计稿，仓库内尚无对应实现；在实现落地前状态保持 Draft。
> Not implemented: this is a design draft with no corresponding implementation in the repository; it stays Draft until implemented.

Main design: [REPORT_CENTER_DESIGN.en.md](./REPORT_CENTER_DESIGN.en.md)
Data model: [REPORT_CENTER_DB_DESIGN.en.md](./REPORT_CENTER_DB_DESIGN.en.md)
API contract: [REPORT_CENTER_API_CONTRACT.en.md](./REPORT_CENTER_API_CONTRACT.en.md)

---

## 1. Scope and adjudication principles

1. This plan covers only the **phase-one commitments** (main design §15 definition of done).
   xlsx/pdf (OD-1), SSE progress (OD-API-2) and presigned URLs (OD-API-3) are out of scope —
   they never entered the commitment, and writing cases for them would turn an open decision
   into a requirement.
2. Every case must hang on **one existing gate** (a script in `package.json` /
   `frontend/package.json`, or a named job under `.github/workflows/`). A case that cannot be
   attached to a gate must be listed explicitly in §9 "Runtime gaps"; leaving it as
   "should be fine" inside the document is not allowed.
3. Priority definitions:
   - **P0**: proves a safety property (permission, tenant isolation, row caps, fail-closed, idempotency).
     Not done unless all P0 cases are green.
   - **P1**: proves contract shape (DTO, envelope, pagination, i18n, menu) and the main path works.
   - **P2**: experience and observability (state coverage, messaging, readable execution history).
4. Case IDs: `TC-R-<category>-<index>` where category is `N` (normal) / `B` (boundary) /
   `E` (error) / `CON` (concurrency) / `SEC` (permission and tenant) / `DB` (migration and retention) /
   `CT` (static contract gates) / `FE` (frontend and smoke).

## 2. Acceptance criteria → case groups (DoD mapping)

| Main design §15 criterion | Covering cases | Gate |
| :--- | :--- | :--- |
| Four business tables landed with migrations | TC-R-DB-1 ~ TC-R-DB-5 | `unit-tests` / `backend-tests` (CI jobs) |
| Report center entry (menu + routes + i18n + permission points) | TC-R-CT-1 ~ TC-R-CT-6, TC-R-FE-1 | `check:harness-permission-contract`, `check:menu-contract`, `check:i18n-hardcode` |
| View and export contract fully implemented and tested | TC-R-N-*, TC-R-E-* | `backend-tests` (`go test -race ./...`) |
| Every permission and audit constraint has a test or smoke | TC-R-SEC-1 ~ TC-R-SEC-9 | existing `internal/middleware` test suite family + `smoke-tenant` |
| Large-report boundaries provable by tests | TC-R-B-1 ~ TC-R-B-7 | `backend-tests` |
| At least one real report pilot attached | TC-R-N-5, TC-R-FE-3 | `pr-smoke-modules` / `test:smoke:system:governance` |
| All P0 green or explicitly waived | §13 waiver process | `quality-gates` |

## 3. Normal path cases

| ID | Level | Scenario | Expected | Gate |
| :--- | :--- | :--- | :--- | :--- |
| TC-R-N-1 | P1 | Published template + valid params → `POST /platform/report/run` | 200, `data.status = succeeded`, non-empty `columns/rows`, one `report_run` row written | `backend-tests` (handler integration) |
| TC-R-N-2 | P1 | Small report `POST /platform/report/export` (sync branch) | `Content-Disposition` attachment header, CSV carries BOM, cells sanitized against formula injection | reuses existing `impexp.WriteCSV` assertions (`backend/pkg/impexp/csv_test.go`) |
| TC-R-N-3 | P1 | Large report with `async = true` | returns `runId` immediately, `statusUrl` reachable, `artifactId` non-empty and downloadable once finished | `backend-tests` |
| TC-R-N-4 | P1 | Publish → rollback → export | export uses the rolled-back version; `current_version_id` advances/retreats correctly; published version rows never change (immutability) | `backend-tests` |
| TC-R-N-5 | P0 | Real pilot (user & permission governance report) end to end | dataset reads real tables, exported row count equals a direct SQL count, export leaves an audit trace | `test:smoke:system:governance` + `system_log_oper` assertions |
| TC-R-N-6 | P1 | History pagination and detail | `list` filtered by `tenant_id`, correct `total`, detail includes `scopeMode` and `paramDigest` | `backend-tests` |
| TC-R-N-7 | P2 | Detail preview with page sizes 10/20/50/100 | frontend SearchToolbar shape, no hardcoded copy | `check:search-toolbar-contract`, `check:i18n-hardcode` |

## 4. Boundary cases

All row caps come from existing constants; tests must not introduce their own magic numbers.

| ID | Level | Boundary | Expected | Basis |
| :--- | :--- | :--- | :--- | :--- |
| TC-R-B-1 | P0 | List request with `pageSize = 500` | server clamps to 100, no error and no amplification | `common.NormalizePagination` (`backend/pkg/common/handler.go:39`); existing assertions in `backend/pkg/common/pagination_test.go` |
| TC-R-B-2 | P0 | Dataset hits 10000 rows, synchronous export | must switch to async or return `status = truncated` with an explicit warning; **silent truncation forbidden** | `impexp.MaxExportRows()` (`backend/pkg/impexp/export_limits.go:9`) |
| TC-R-B-3 | P0 | Dataset registered with `row_limit` > 10000 | rejected at parameter validation (400), never persisted | main design §8.4 |
| TC-R-B-4 | P1 | Render duration approaching `WriteTimeout` | must take the async branch; the sync branch must not hang the connection | server timeout configuration in `backend/cmd/server/main.go` |
| TC-R-B-5 | P1 | Both `versionId = 0` and `versionId = <published version>` | identical results (follow current); a `versionId` pointing at a draft is rejected | API contract §4.1 |
| TC-R-B-6 | P1 | Number/time column formatting (year boundary, timezone, zero values) | exported values equal preview values; list times are timezone-carrying RFC3339 | `check:system-datetime-presentation` (`frontend/package.json`) |
| TC-R-B-7 | P2 | Params contain a non-whitelisted key | `report.param.invalid`, the illegal key/value is not echoed | API contract §2 |

## 5. Error path cases

| ID | Level | Fault injection | Expected | Basis |
| :--- | :--- | :--- | :--- | :--- |
| TC-R-E-1 | P0 | Data provider returns an error | `report_run.status = failed` plus an error key; must **not** return an empty set pretending "no data" | main design §7 constraints |
| TC-R-E-2 | P0 | Data scope policy missing / table absent / illegal mode | render judged `failed` (or a 500 envelope), never rendered as an empty report; consistent with middleware fail-closed | `backend/internal/middleware/data_scope_middleware.go:117-131`, `:210-213`, `backend/pkg/database/scope.go:66-68`; existing suite `backend/internal/middleware/data_scope_failclosed_test.go` |
| TC-R-E-3 | P0 | Caller lacks `platform:report:run` | `permission.denied` with business code 403, written as a real status via `FailWithCode` | `backend/internal/middleware/casbin_middleware.go`, existing `casbin_middleware_test.go` |
| TC-R-E-4 | P0 | High-sensitivity action without `X-Operation-Token` | `auth.operation.verification_required`; distinct keys for expired / mismatch | `backend/internal/middleware/secure_action_middleware_test.go` |
| TC-R-E-5 | P1 | `PUT` on a published version / two consecutive `publish` calls | second call returns `report.version.immutable`, no second version row | API contract §9.4 |
| TC-R-E-6 | P1 | Download an expired artifact | `report.artifact.expired`, no new row written | DB design §6 |
| TC-R-E-7 | P1 | Request an uncommitted format (`format = xlsx`) | explicit rejection (not in `supported_formats`), **not** a silent fallback to CSV | main design §8.2 |
| TC-R-E-8 | P2 | Error key shape regression | every `report.*` key matches `^[a-z0-9_]+(?:\.[a-z0-9_]+)+$`, no hyphens | `backend/pkg/common/http/response.go:28`, see `backend/pkg/common/error_code_test.go` |

## 6. Concurrency and idempotency cases

Concurrency assertions rely on existing fixtures rather than hand-rolled waiting logic:
`Open/OpenTB` from `backend/pkg/testmysql`, `AssertRowLocked`
(`backend/pkg/testmysql/locking.go:18`, probes with `SELECT ... FOR UPDATE NOWAIT`) and
`AssertSerialisedWriters` (`:39`, proves two writers of the same parent row cannot interleave).

| ID | Level | Scenario | Expected | Gate |
| :--- | :--- | :--- | :--- | :--- |
| TC-R-CON-1 | P0 | Two concurrent exports with the same `idempotencyKey` | exactly one `report_artifact` row; the second request reuses it | `backend-tests` (`go test -race`) |
| TC-R-CON-2 | P0 | Provider call counting under the same key | **two concurrent requests produce exactly one data fetch** (provider call count = 1); the explicit proof point of API contract §9.2 | `backend-tests` |
| TC-R-CON-3 | P0 | Concurrent template creation with the same code in one tenant | the loser gets 409 semantics, no duplicate row; after deletion the same code is reusable again (rename release) | DB design §5; fixture `AssertSerialisedWriters` |
| TC-R-CON-4 | P1 | Concurrent `publish` of the same draft | only one succeeds, the rest return `report.version.immutable` | `backend-tests` |
| TC-R-CON-5 | P1 | Cancel while rendering (`status = cancelled`) | no artifact produced after cancellation; an already-produced artifact is marked cleanup-eligible | `backend-tests` |
| TC-R-CON-6 | P1 | Retention cleanup running while a render is in flight | the maintenance task does not re-enter (overlap protection) and does not delete a row being written | `backend/pkg/maintenance/maintenance.go`, existing `maintenance_test.go` |

## 7. Permission and tenant security cases (the P0 core)

| ID | Scenario | Expected | Basis |
| :--- | :--- | :--- | :--- |
| TC-R-SEC-1 | `scope_binding = dept` for a `dept_and_children` user | result set contains only that department subtree; `scopeMode` echo matches | `backend/pkg/database/scope.go:38-57`, `data_scope_middleware.go:324` (the `system_dept.ancestors` expansion stays in the middleware; the report side does not copy it) |
| TC-R-SEC-2 | Same dataset requested by two users with different scopes | the second user must **not** hit the first user's cached result | main design §10 constraint 1 (`scopeFingerprint` is mandatory) |
| TC-R-SEC-3 | Multi-role user (one narrow, one wide) | the widest mode wins | `data_scope_middleware.go:368` `resolveDataScopeMode` |
| TC-R-SEC-4 | `admin` / `tenant_admin` request | skips the dept policy but stays narrowed inside its own tenant | `data_scope_middleware.go:455`, `:478` |
| TC-R-SEC-5 | Cross-tenant read of a template or artifact | admitted only by `tenant_id` + `visibility`; another tenant's private template returns not-found semantics rather than "exists but denied" | [TENANT_CONTRACT_V1.en.md](../contracts/TENANT_CONTRACT_V1.md) §6 |
| TC-R-SEC-6 | Run the whole main path once in `compat` and once in `multi` | identical behaviour; a missing tenant context fails closed with `TENANT_CONTEXT_MISSING`, never silently widened | contract §3.2/§5; existing `backend/internal/middleware/tenant_platform_scope_test.go` |
| TC-R-SEC-7 | Tenant quota already exceeded | the report side only displays it, it does not block writes (display-phase semantics) | the "quota display phase" clause of the contract |
| TC-R-SEC-8 | Whether a list permission falls back to write permission | a user holding only `list` sees the inventory but cannot run or export | [DESIGN.md](../../DESIGN.md) §4.4 |
| TC-R-SEC-9 | Hostile browser calls the report URL directly with forged parameters | rejected in the same shape as the existing hostile suite | `frontend` script `test:smoke:tenant` (`tests/smoke-core/tenant-hostile-browser-matrix*.spec.ts`), CI job `smoke-tenant` (`.github/workflows/smoke-core.yml:138`) |

## 8. Migration and retention cases

| ID | Scenario | Expected | Gate |
| :--- | :--- | :--- | :--- |
| TC-R-DB-1 | Fresh database migrated up to the report migration numbers | tables/indexes/charset match DB design §2 (`ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`) | `go test ./pkg/database/` (the package containing `backend/pkg/database/migrate_test.go`) |
| TC-R-DB-2 | Down shape | down reverts only this package's new objects and touches no other domain; table order is the reverse of up | existing assertion style of `backend/pkg/database/migrate_down_shape_test.go`, `migrate_down_test.go` |
| TC-R-DB-3 | Unique keys exclude `deleted_at` | after a soft delete the same code is reusable; index definition matches the convention at `backend/pkg/database/migrations/000001_init_schema.up.sql:117-120` | `migrate_preflight_test.go` (`backend/pkg/database/migrations/preflight/`) |
| TC-R-DB-4 | Retention cleanup | batched `LIMIT` deletes, no full-table lock; `report.artifact_retention_days` / `report.run_retention_days` honoured; object storage cleaned too | reuse `RegisterMaintenanceTasks` (precedent `backend/modules/system/audit/audit_service.go:317`) |
| TC-R-DB-5 | Cleanup task while maintenance is disabled | no illusion of execution; both docs and tests state the operator action when `PANTHEON_MAINTENANCE_ENABLED=false` | runner env gates in `backend/cmd/server/main.go` |

## 9. Static contract and documentation gates

| ID | Gate | Command | Assertion |
| :--- | :--- | :--- | :--- |
| TC-R-CT-1 | Permission key shape | `npm run check:harness-permission-contract` (`scripts/harness/check-permission-contract.mjs`) | every report `Type: "F"` action node binds an action key (`view/run/export/create/update/delete/clear`) and none binds `platform:report:list` — that exact case is the checker's only blocking finding (`:125-136`) |
| TC-R-CT-2 | DTO shape | `npm run check:backend-dto-contract` | handlers return `*Resp`-style types, no bare `map[string]any` |
| TC-R-CT-3 | Response envelope | `npm run check:backend-response-contract` | unified `Success/Fail/FailWithCode`, no bare `c.JSON` |
| TC-R-CT-4 | Docs frontmatter | `npm run check:docs-frontmatter` (`scripts/frontmatter-check.mjs`) | all eight documents in this package have complete fields and `linked_contracts` pointing at existing files |
| TC-R-CT-5 | Doc links | `npm run check:harness-docs` | no dead links; new docs registered in the index |
| TC-R-CT-6 | Menu contract | `npm run check:menu-contract` (`frontend/scripts/check-menu-contract.mjs`) | report menu node key / permission point / title key complete |
| TC-R-CT-7 | i18n | `npm run check:i18n-hardcode`, `check:i18n-generated-scope` | no hardcoded page copy; zh-CN and en-US registered together |
| TC-R-CT-8 | Encoding and structure | `npm run check:harness-encoding`, `check:structure` (CI job `docs-governance`, `.github/workflows/quality.yml:100`) | UTF-8 without mojibake; documents live under `docs/designs/` |
| TC-R-CT-9 | Backend lint | CI job `go-lint` (`.github/workflows/ci.yml:249`) | new packages introduce no lint regression |

## 10. Frontend and smoke cases

| ID | Scenario | Gate |
| :--- | :--- | :--- |
| TC-R-FE-1 | Report center menu reachable; page admission denies users without permission | `check:system-page-admission` (`frontend/package.json`) |
| TC-R-FE-2 | Four states present: no permission / empty / loading / submitting | discipline of [EMPTY_LOADING_ERROR_STATES.en.md](./EMPTY_LOADING_ERROR_STATES.md) + a new spec |
| TC-R-FE-3 | List and export interaction of the real pilot report | new `frontend/tests/smoke/system/report-center.spec.ts` plus a registered `test:smoke:system:report` script |
| TC-R-FE-4 | The truncation warning (`truncated`) is visible | same new spec |
| TC-R-FE-5 | Frontend unit tests (pure presentation functions: column mapping, parameter digest) | `npm run test:unit` (vitest, `frontend/package.json`) |

**Mandatory registration when adding a spec**: `frontend/scripts/check-smoke-coverage-contract.mjs`
requires every `tests/smoke/**/*.spec.ts` file to be referenced by some `test:smoke:*` leaf script and
mapped in `frontend/tests/smoke/README.md` (findings raised at `:85`, `:91`, `:96` of that script).
So TC-R-FE-3 must land as "spec + package.json script + smoke README row" in the same PR, otherwise
`check:smoke-coverage-contract` (part of the `prebuild` chain) fails outright.

## 11. Runtime gaps (what this plan does not prove)

Stated explicitly so the document cannot be read as covering what it does not:

| Gap | Current state | Required action | Decision owner |
| :--- | :--- | :--- | :--- |
| Large-report performance | the repository only has an **advisory** bench job (`bench-perf-smoke`, `.github/workflows/ci.yml:310`; single iteration, and it runs only the audit-side `BenchmarkAuditServiceListOperationLogs`); there is no report capacity baseline | a manual load run at realistic volume (>= 100k rows) with recorded evidence; this plan does not claim the target is met | Maintainer |
| Measured headroom between sync export and `WriteTimeout` | only the configuration fact exists, no measurement | measure during the load run and calibrate the switch-to-async threshold | Maintainer |
| Cross-instance cache invalidation | phase one may converge via TTL (main design §10 constraint 3) | real-time invalidation needs the existing watcher direction plus a cross-instance test | Maintainer |
| Behaviour when Redis is unavailable | fail-fast in production or when `PANTHEON_REDIS_REQUIRED=true` (`backend/pkg/database/redis.go`, test `redis_required_test.go`) | the report side adds no bespoke fallback; the implementation only needs to confirm this precondition | — |
| Coverage | the CI threshold comes from `vars.COVERAGE_THRESHOLD` (default 50, `coverage-gate`, `.github/workflows/ci.yml:396`) | new packages must prove they do not lower the gate; no numeric promise in this document | — |

## 12. Execution checklist (self-check in the implementation PR)

```bash
# backend (equivalent to the CI job backend-tests; needs MySQL 8 + Redis and the test env vars)
cd backend && go test -race ./...
cd backend && go test ./pkg/database/ ./pkg/impexp/ ./pkg/maintenance/ ./internal/middleware/

# static contract gates
npm run check:docs-frontmatter
npm run check:harness-docs
npm run check:harness-permission-contract
npm run check:backend-dto-contract
npm run check:backend-response-contract

# frontend
cd frontend && npm run test:unit
cd frontend && npm run check:menu-contract && npm run check:i18n-hardcode \
  && npm run check:system-datetime-presentation && npm run check:smoke-coverage-contract

# smoke (needs local services; scripts come from frontend/package.json)
cd frontend && npm run test:smoke:core
cd frontend && npm run test:smoke:tenant
```

The MySQL/Redis fixtures read `backend/pkg/testmysql` / `backend/pkg/testredis`; in CI the related
variables are `PANTHEON_TEST_DSN` / `PANTHEON_TEST_DSN_REQUIRED` / `PANTHEON_TEST_REDIS_ADDR` /
`PANTHEON_TEST_REDIS_REQUIRED` (see the `backend-tests` job at `.github/workflows/quality.yml:379`),
and a missing DSN is designed to **fail that gate rather than skip silently**.

## 13. Waiver process

A waived P0 case must state, in the implementation PR: the waived case ID, the reason, the compensating
evidence (a manual run record or runbook link) and the re-review date. Missing any of these means the
case is not waived. Waiver records live under `.harness/evidence/` (precedent referenced by the
`coverage-gate` threshold comment in `.github/workflows/ci.yml`, which cites
`.harness/evidence/2026-09-10-tenant-ready-guardrails/`).

## 14. Open decisions

| ID | Decision | Options | Decision owner |
| :--- | :--- | :--- | :--- |
| OD-TEST-1 | Whether report smoke joins the core suite (`test:smoke:core` / CI `pr-smoke-core`) | A: keep it out of core, standalone `test:smoke:system:report`; B: add it to core | Maintainer (leans A; the core suite currently pins exactly 9 specs, see `frontend/package.json`) |
| OD-TEST-2 | Form of the large-report performance evidence | A: a one-off manual record under `.harness/evidence/`; B: a new advisory bench job | Maintainer |
| OD-TEST-3 | Whether concurrency proofs must use the real MySQL fixture | A: use `testmysql` (closer to real lock semantics); B: use an in-memory stand-in (faster but weaker) | Maintainer (leans A; P0 cases do not accept B) |

## 15. Related documents

| Document | Chinese | English |
| :--- | :--- | :--- |
| Main design | [REPORT_CENTER_DESIGN.md](./REPORT_CENTER_DESIGN.md) | [REPORT_CENTER_DESIGN.en.md](./REPORT_CENTER_DESIGN.en.md) |
| Data model and migrations | [REPORT_CENTER_DB_DESIGN.md](./REPORT_CENTER_DB_DESIGN.md) | [REPORT_CENTER_DB_DESIGN.en.md](./REPORT_CENTER_DB_DESIGN.en.md) |
| API contract | [REPORT_CENTER_API_CONTRACT.md](./REPORT_CENTER_API_CONTRACT.md) | [REPORT_CENTER_API_CONTRACT.en.md](./REPORT_CENTER_API_CONTRACT.en.md) |
| Test plan (this document) | `REPORT_CENTER_TEST_PLAN.md` | [REPORT_CENTER_TEST_PLAN.en.md](./REPORT_CENTER_TEST_PLAN.en.md) |

Scheduler companion: SCHEDULER_CENTER_TEST_PLAN.en.md（planned/unavailable：`./SCHEDULER_CENTER_TEST_PLAN.md`；无当前交付承诺）.
Database conventions: [DATABASE.en.md](./DATABASE.md). Index: [README.en.md](./README.en.md).

## Change log

| Date | Change |
| :--- | :--- |
| 2026-11-04 | Built the acceptance-criteria-to-case mapping, the four case classes, permission and tenant security cases, migration and retention cases, the static gate checklist, runtime gaps and the waiver process |
