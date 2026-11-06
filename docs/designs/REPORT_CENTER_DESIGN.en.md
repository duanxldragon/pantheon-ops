---
title: Report Center Design
doc_type: Design
layer: platform / business/*
status: Draft
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
  - docs/contracts/SYSTEM_IAM_CONTRACT.md
  - docs/contracts/TENANT_CONTRACT_V1.md
updated_at: 2026-11-04
---

# Report Center Design

> Not implemented: this is a design draft with no corresponding implementation in the repository; it stays Draft until implemented.
> 未实现声明：本文为设计稿，仓库内尚无对应实现；在实现落地前状态保持 Draft。

Chinese version: [REPORT_CENTER_DESIGN.md](./REPORT_CENTER_DESIGN.md)

Related contracts:
- `PLATFORM_CONTRACT.md`
- `SYSTEM_IAM_CONTRACT.md`
- `TENANT_CONTRACT_V1.md`

Related roadmap:
- `P2_SCALE_ROADMAP.md` (§8.3 Scheduler / Report / Alert centers)
- `PANTHEON_BASE_ARCHITECTURE_OVERVIEW.md`

The other three documents of this package (data model, API contract, test plan) are indexed in §17.

This document defines the report center as a unified platform surface for report discovery, rendering,
export and governance while leaving report semantics with source domains. It is not a BI platform:
the first phase unifies the report catalogue, the entry point, the export boundary and the
permission/audit semantics.

## 1. Design Goals

1. One platform-level entry for reports across domains instead of reports scattered in business pages
2. One governance model for report definition, applicable permissions and export actions
3. Dashboards, the scheduler center and report archives can reuse the same metadata
4. Large reports have an explicit row / timeout / async boundary; unbounded queries are not allowed

## 2. Non-Goals

Out of scope now:

- self-service drag-and-drop analytics
- multi-dimensional OLAP engine
- visual data-modelling workbench
- massive interactive chart designer
- free-form SQL authoring for end users (§5.1; the generator's external datasources are still limited to
  `information_schema` structure probing — see [DATABASE.md](./DATABASE.md) §3.3.4)
- cross-tenant federated queries ([TENANT_CONTRACT_V1.en.md](../contracts/TENANT_CONTRACT_V1.en.md) §1 Out)

## 3. Ownership Boundary

### 3.1 What `platform` Owns

- the report-center entry and aggregated catalogue
- report metadata views (dataset / template / version / run / artifact)
- orchestration and state machine of the render pipeline
- governance of the export entry (permission keys, audit, row and timeout caps)

### 3.2 What Source Domains Own

- report data generation logic (the fetch provider)
- data semantics and column meaning
- report detail implementation
- correctness of exported content

Boundary basis: [PLATFORM_CONTRACT.en.md](../contracts/PLATFORM_CONTRACT.en.md) §7.1 ("the aggregation layer may
read multiple sub-domains but must not intrude into their responsibilities") and
[DESIGN.md](../../DESIGN.md) §2.3. Symmetrically, the report center never imports a
`modules/system/*` or `modules/business/*` service; it calls providers registered by the source domain
(composition-root injection pattern as in `backend/modules/platform/routes.go`, boundary rules in
[REPOSITORY_LAYOUT.md](./REPOSITORY_LAYOUT.md)).

## 4. End-to-End Chain

```text
dataset (register semantics) -> template (presentation & params) -> version (publish) ->
run (render: authorize -> fetch -> format -> artifact) -> export (download) / schedule
```

| Stage | Owner | Table | Key constraint |
| :--- | :--- | :--- | :--- |
| dataset | source-domain developer | `report_dataset` | only registered semantics can be referenced |
| template | report author | `report_template` | visibility and permission key declared at definition time |
| version | report author | `report_template_version` | immutable after publish; referenced by run/schedule |
| run | report center | `report_run` | must record scope fingerprint and parameter digest |
| artifact | report center | `report_artifact` | object key must carry the tenant segment |
| schedule | scheduler center | `schedule_job` | see §11; this center does not implement cron |

## 5. Data-Source Model (dataset)

### 5.1 Semantics Are Registered Capability, Not a Query Entry

`report_dataset` is the smallest unit of a reusable fetch semantics. Phase one supports exactly one source kind:

| Field | Meaning |
| :--- | :--- |
| `code` | semantics identifier, unique platform-wide |
| `name` / `description` | display and notes (display text via i18n keys, §13) |
| `source_domain` | owning domain, e.g. `system.iam` / `system.audit` / `business.*` |
| `provider_key` | fetch provider id, registered by the source domain, called by key |
| `param_schema_json` | parameter whitelist: name, type, required, max value / max collection length |
| `scope_binding` | row-scope binding: `none` / `dept` / `self` (§9) |
| `row_limit` | per-fetch row cap, must be <= the global synchronous cap |
| `timeout_ms` | fetch timeout for this semantics |
| `cacheable` | whether result caching is allowed (default false) |
| `freshness_seconds` | tolerated staleness, used for the "as of" display |

Constraints:

1. The report center accepts no SQL text, table name or column expression; a dataset only declares
   "which semantics + which parameters".
2. Dataset definitions are `platform-global` resources, aligned with the 2026-10-15 ruling for
   `system_menu` / `system_permission` in
   [TENANT_RESOURCE_SCOPE_MATRIX.en.md](./TENANT_RESOURCE_SCOPE_MATRIX.en.md) §2: semantics are release
   artifacts, not tenant-maintained; tenant differences come from Casbin and row scope.
3. Datasets are registered from code (module seed / registry) and read-only at runtime; console CRUD for
   datasets is not in phase one.

### 5.2 Relationship with the Existing Import/Export Surface

`backend/pkg/impexp` already carries CSV import/export for the system domains. The report center is its
**consumer** and does not add a second export implementation:

| Existing capability | File | How the report center uses it |
| :--- | :--- | :--- |
| CSV writer (BOM, formula-injection neutralising, RFC 5987 filename) | [../../backend/pkg/impexp/csv.go](../../backend/pkg/impexp/csv.go) (`WriteCSV` line 56, `sanitizeCSVCell` line 33) | every CSV artifact must go through it, cells via `sanitizeCSVRow` |
| synchronous export hard cap = 10000 rows | [../../backend/pkg/impexp/export_limits.go](../../backend/pkg/impexp/export_limits.go) (`MaxExportRows()` line 9) | ceiling of synchronous rendering; SQL must push `Limit()`, as `backend/modules/system/org/post/post_service.go:537` does |
| five-step import receive pipeline | `backend/pkg/impexp/pipeline.go` (`ReadImportCSV`, `ImportFileField = "file"`) | the report center does not import; a "bulk parameter import" must reuse this receive surface |
| import template CSV assembly | `backend/pkg/impexp/template.go` (`BuildImportTemplate`) | parameter template / example-row download reuses it |
| governance export headers and labels | `backend/pkg/impexp/governance_template.go` | governance reports (§14) reuse these label maps instead of duplicating them |

## 6. Template Definition and Versioning

### 6.1 Template

`report_template` describes "one report as seen by a user": the bound dataset, visibility, supported
export formats, permission key and default parameters.

Minimal fields (the draft's metadata model, turned into executable constraints):

| Field | Meaning |
| :--- | :--- |
| `id` / `tenant_id` | primary key / tenant dimension (§9.3) |
| `code` | unique inside the tenant (released by renaming on delete, see DB design §5) |
| `report_type` | classification: `list` / `summary` / `governance` / `audit` |
| `name` / `description` | display name (i18n key) and notes |
| `category` | filter category for the list page |
| `dataset_code` | bound semantics |
| `visibility` | `private` / `team` / `platform` (draft decision retained) |
| `owner_user_id` | owner for `private` visibility |
| `supported_formats` | comma separated, only formats actually implemented (phase one: `csv`, §8) |
| `default_param_json` | default parameters |
| `permission_key` | permission key required to view the report (§9.1) |
| `status` | `draft` / `published` / `disabled` |
| `current_version` | pointer to the published version |

`visibility` only answers "who can see this report in the list"; "which rows they see" is decided by
§9. The two must not be conflated (same discipline as
[SYSTEM_IAM_CONTRACT.en.md](../contracts/SYSTEM_IAM_CONTRACT.en.md): a menu is not the permission model).

### 6.2 Versions

1. A version holds the presentation spec (columns, order, formatting, default parameters) plus a snapshot
   of the bound dataset.
2. Published versions are **immutable**: editing a published template writes a new draft version row and
   publishing moves the `current_version` pointer.
3. Every `report_run` records `template_version_id`, so historical exports stay reproducible.
4. Scheduled reports (§11) reference `template_id + version`: follow `current_version` by default, pinning
   allowed. If a pinned version is removed, the schedule job must go to `disabled` — no dangling reference.
5. Rollback = point `current_version` back to a previously published version; history rows are rewritten
   never.

## 7. Render Pipeline

State machine of a single render (`report_run.status`):

```text
pending -> running -> succeeded
                   -> failed
                   -> cancelled
running  -> truncated (row cap hit and a file was still produced)
```

Execution steps; the order is a security property, not a style choice:

1. **Route authorization**: Casbin (the `TokenAuth -> PlatformScopeTenantContext -> Casbin` chain of
   `backend/pkg/contracts/route_groups.go:137`).
2. **Template admission**: `status = published` + `visibility` + `permission_key`.
3. **Row-scope resolution**: read `common.GetDataScope(c)` injected by
   `backend/internal/middleware/data_scope_middleware.go:102` and derive the scope fingerprint.
4. **Parameter validation** against the `param_schema_json` whitelist; out-of-range input fails as `param.invalid`.
5. **Fetch**: call the provider by `provider_key`, enforce `LIMIT row_limit` and `timeout_ms`; when
   `scope_binding != none` push the filter through `db.Scopes(database.WithDataScope(...))`
   (`backend/pkg/database/scope.go:12`) — application-level filtering of a full table is forbidden.
6. **Format**: column mapping plus number/time formatting (time presentation is gated by
   `check:system-datetime-presentation`, see `frontend/package.json`).
7. **Produce**:
   - synchronous: stream back through `impexp.WriteCSV`;
   - asynchronous: write an artifact (local or S3 driver via `backend/pkg/upload`, §8.3) and record its id.
8. **Audit**: persist `report_run`, and let the operation-log middleware
   (`backend/internal/middleware/operation_log_middleware.go`) record the request-level trace.

Constraints:

- a failure in step 5 must not degrade into an empty result pretending "no data": the run becomes `failed`
  with an error key.
- rendering writes no business tables; `report_run` and `report_artifact` are the only write targets.

## 8. Export Formats

### 8.1 Phase One Commits to CSV Only

`csv` is the only encoded export path implemented in this repository (§5.2). The allowed value set of
`supported_formats` in phase one is `{csv}`.

### 8.2 xlsx / pdf Are Not Promised

The draft proposed `csv / xlsx / pdf` for phase one. Verified reality: the require list of
`backend/go.mod` contains no xlsx (excelize or similar) and no pdf (maroto/gotenberg or similar)
dependency, and `backend/pkg/impexp` implements CSV only. xlsx / pdf are therefore explicitly demoted to an
**open decision** (§16 OD-1). Until a dependency is approved:

- the API exposes no xlsx/pdf option;
- the frontend shows no such button ([DESIGN.md](../../DESIGN.md) §7.5: controls without real behaviour
  must not be presented as clickable capability).

### 8.3 Artifact Carrier

Asynchronous artifacts reuse the driver abstraction of `backend/pkg/upload/service.go`
(`upload.storage_driver` = `local` | `s3`; object key = `scope + date + uuid`). The report scope segment
must carry the tenant dimension — same rule as [TENANT_RESOURCE_SCOPE_MATRIX.en.md](./TENANT_RESOURCE_SCOPE_MATRIX.en.md)
§2 ("export files / upload objects: `derived`, the key must be able to carry the tenant so that lookups
cannot cross tenants").

### 8.4 Large-Report Limits (No Unbounded Queries)

| Limit | Source | Behaviour on violation |
| :--- | :--- | :--- |
| synchronous rows | `impexp.MaxExportRows()` = 10000 (`backend/pkg/impexp/export_limits.go:9`) | switch to the async job, or `truncated` with an explicit notice; silent truncation forbidden |
| dataset rows | `report_dataset.row_limit`, and `<= 10000` | rejected by parameter validation (400) |
| list pagination | `common.NormalizePagination(page, pageSize, 100)` (`backend/pkg/common/handler.go:39`; every system domain passes 100) | pageSize clamped to 100 |
| HTTP write timeout | `WriteTimeout` in `backend/cmd/server/main.go` (synchronous downloads must fit inside) | beyond the threshold the run must be async |
| per-request rate | `middleware.RateLimiter` in `registerAPIRoutes` (`backend/cmd/server/main.go`) | 429 |

Any implementation shape that scans a whole table and filters afterwards is forbidden by this design.

## 9. Permission Model: Who May Read Which Rows

Three orthogonal layers, each narrowing the previous one; no layer may substitute for another.

### 9.1 Route Layer (Casbin Permission Keys)

- Key naming follows the existing shape `platform:<resource>:<action>`
  (`PERMISSION_KEY_PATTERN` at `scripts/harness/check-permission-contract.mjs:30`). The checker's blocking
  finding is not "the action word must be whitelisted" but "an action menu node (`Type: \"F\"`) bound to a
  `...:list` permission" (`:125-136`); `ACTION_TOKENS` (`:11-28`) only feeds its heuristic warnings.
- The key set used by this package is registered in
  [REPORT_CENTER_API_CONTRACT.en.md](./REPORT_CENTER_API_CONTRACT.en.md) §6.
- Every action needs its own key; `...:list` must not double as the write guard
  ([DESIGN.md](../../DESIGN.md) §4.4).

### 9.2 Row Layer (Reuse the Data-Scope Middleware Semantics, Do Not Reimplement)

The report center **directly reuses** `backend/internal/middleware/data_scope_middleware.go`:

| dataset `scope_binding` | Reused mechanism | Semantics source |
| :--- | :--- | :--- |
| `none` | no row predicate | the semantics has no row ownership (e.g. global summaries) |
| `dept` | `database.WithDataScope` -> `dept_id = ?` / `dept_id IN ?` | `backend/pkg/database/scope.go:38-57` |
| `self` | `database.WithDataScope` -> `created_by` / `create_by` / `id` | `backend/pkg/database/scope.go:70-96` |

Rulings that must be inherited verbatim (each from code actually read):

1. **Fixed mode set**: `all` / `self` / `dept` / `dept_and_children` / `custom`
   (`backend/pkg/common/data_scope.go:11-15`); multiple roles resolve to the widest
   (`data_scope_middleware.go:368` `resolveDataScopeMode`).
2. **Fail-closed**: when the policy read fails, the table is missing, or the mode is invalid, non-admin
   requests are rejected (500 or the JSON error envelope) and the scope becomes `custom` with an empty dept
   set, which makes `WithDataScope` emit `1 = 0`
   (`data_scope_middleware.go:117-131`, `:210-213`, `backend/pkg/database/scope.go:66-68`). Rendering must
   treat this as `failed`, never as an "empty report".
3. **Admin relaxation applies only to the caller's own request**: `admin` and `tenant_admin` skip the dept
   policy as full administrators of their own scope (`data_scope_middleware.go:455`).
4. **Tenant-local roles are exempt from the dept policy**: `tenant_admin` / `tenant_member` are confined by
   the tenant scope, not the platform dept policy (`data_scope_middleware.go:478`).
5. **Policies are read per tenant**: the unique key of `system_role_data_scope` is `(tenant_id, role_key)`,
   so policy reads must carry the tenant predicate (`data_scope_middleware.go:45-51`, `:250`).
6. **`dept_and_children` expands children through `system_dept.ancestors`**
   (`data_scope_middleware.go:324`); the report side does not copy that expansion.

The report center's own tables do **not** use the dept row model (report definitions are not
department-owned data); row scope applies only to business rows produced by a dataset.

### 9.3 Tenant Layer

- Routes must use the chain shaped like `contracts.SystemProtectedGroup(...)`:
  `TokenAuth -> PlatformScopeTenantContext -> Casbin`
  (`backend/pkg/contracts/route_groups.go:137`; dual-mode semantics in
  [TENANT_CONTRACT_V1.en.md](../contracts/TENANT_CONTRACT_V1.en.md) §6).
- If the entry ends up under the `platform` module rather than the `/system` prefix, the platform scope
  must still be declared explicitly; otherwise a missing context fails closed as `TENANT_CONTEXT_MISSING`
  (contract §3.2/§5) instead of silently widening.
- Resource classification (registered in the DB design): dataset `platform-global`;
  template/version `tenant-owned`; run/artifact `derived` (follows the source).
- `business/*` routes do not participate in tenant isolation in V1 (contract §2.6,
  `backend/pkg/contracts/route_groups.go:21-26`): a business report's tenant dimension can only come from
  the business rows themselves; attaching a tenant middleware to business routes creates no isolation.

### 9.4 Sensitive Exports

The following actions require the second-factor `X-Operation-Token`
(`backend/internal/middleware/secure_action_middleware.go`; precedents at
`backend/modules/auth/module.go:97-108` and `backend/modules/system/system_modules.go:188-191`):

- deleting a template
- publishing a version
- triggering an export of a report flagged `sensitive = true`
- bulk cleanup of runs / artifacts

## 10. Caching

| Layer | Content | Key | Invalidation |
| :--- | :--- | :--- | :--- |
| result cache | dataset fetch results (only when `cacheable = true`) | connection namespace + `tenantID` + `datasetCode` + `paramDigest` + `scopeFingerprint` + `freshness` | TTL (`freshness_seconds`) |
| artifact cache | artifact files | object key (includes the tenant segment) | retention (DB design §6) |
| metadata cache | template/version definitions | `tenantID` + `templateID` + `current_version` | explicit broadcast on publish |

Constraints:

1. **`scopeFingerprint` is mandatory**, derived from `mode` + normalized `deptIDs` + `userID`. Dropping it
   lets a `dept`-scoped user hit an `all`-scoped result — that is a row-permission leak, not a
   performance issue.
2. Cache namespaces must be built from the connection identity, not a pool pointer
   (`backend/internal/middleware/data_scope_middleware.go:315-322` and `database.ConnectionCacheNamespace`).
3. Metadata invalidation follows the existing cross-instance direction: `PANTHEON_CASBIN_WATCHER` + Redis
   ([BACKEND.md](./BACKEND.md), "Casbin multi-instance broadcast"). Phase one may ship without a
   cross-instance broadcast and rely on TTL, but that must be stated as an explicit gap, not claimed as
   real-time sync.
4. Caching lives in process memory or Redis; no new storage component.

## 11. Interplay with the Scheduler Center

Division of labour (details in [SCHEDULER_CENTER_DESIGN.en.md](./SCHEDULER_CENTER_DESIGN.en.md), not duplicated here):

- the scheduler center owns cron parsing, triggering, multi-replica single-flight, retry and history;
- the report center contributes exactly two things:
  1. the handler `report.generate` (`handler_key`) taking `templateId`, `versionPolicy`, `paramJson`,
     `receiverScope`;
  2. the write-back: one scheduled execution creates a `report_run` (`trigger_source = schedule`) and its
     `report_artifact`;
- the report side must **not** implement its own cron, timer or lock; registration and enable/disable go
  through the scheduler contract (SCHEDULER_CENTER_API_CONTRACT.en.md（planned/unavailable：`./SCHEDULER_CENTER_API_CONTRACT.en.md`；无当前交付承诺） §5);
- failure semantics: a render failure returns a classified error key, the scheduler applies its retry
  policy, and after `retry_max` the scheduler emits a failure summary consumed by the alert center
  ([ALERT_MONITORING_CENTER_DESIGN.en.md](./ALERT_MONITORING_CENTER_DESIGN.en.md));
- idempotency: duplicate deliveries of one scheduled fire are suppressed by the scheduler's idempotency
  key; the report-side dedup key is `(template_id, schedule_fire_key)` and reuses the existing artifact
  instead of producing a second file.

## 12. UI Skeleton

Phase-one minimum (list pages follow [FRONTEND_PAGE_TEMPLATES.en.md](./FRONTEND_PAGE_TEMPLATES.en.md) and
the `SearchToolbar` contract; states follow [EMPTY_LOADING_ERROR_STATES.en.md](./EMPTY_LOADING_ERROR_STATES.en.md)):

- report list page: category filter, source-domain filter, visibility indicator, updated time, notes
- report detail page: parameter area + paged preview (10/20/50/100) + export entry
- template management page: draft editing, version list, publish/rollback
- my exports / history page: `report_run` list + artifact download + status (including `truncated`)
- forbidden / empty / loading / submitting states are mandatory

Not required yet: big-screen designer, chart layout editor, subscriptions.

## 13. Audit and Traces

Page action -> permission key -> audit record must all be present:

| Must be recorded | Carrier |
| :--- | :--- |
| who viewed a report | `report_run` (`trigger_source = manual`) + `system_log_oper` |
| who exported it and in which format | `report_run.format` + `report_artifact` |
| filter-condition summary | `report_run.param_digest` + `scope_fingerprint` (digests, not large raw parameters) |
| who edited/published a template | `system_log_oper` + the `report_template_version` row itself |
| second factor for sensitive actions | `SecureActionMiddleware` 403 branch already has error keys; successful actions rely on the operation log |

- `system_log_oper` already provides `source_domain / source_page / failure_category / cost_time` with
  indexes; the report side only sets the title key and source domain instead of adding audit columns
  ([DATABASE.md](./DATABASE.md) §3.2 and `backend/internal/middleware/operation_log_middleware.go`).
- i18n: menu titles use `title_key`, permission points use the permission group, errors return keys rather
  than natural language ([ERROR_CODE_AND_I18N.md](./ERROR_CODE_AND_I18N.md) §3.1; seed shape at
  `backend/modules/system/i18n/seed_data.go:79`, menu/permission registration at
  `backend/modules/auth/login/login_menu.go:81-89`).
- Retention and purge go through the `maintenance` registry, never as full-table DELETEs inside a request
  (package comment of `backend/pkg/maintenance/maintenance.go`; precedent
  `backend/modules/system/audit/audit_service.go:317`).

## 14. Minimum Pilot Requirement

At least one real report must be wired in, preferring sources that already have an export implementation:

1. audit event summary report (`system.audit`, `scope_binding = none`)
2. user and permission governance report (`system.iam`, may reuse the label sets in
   `backend/pkg/impexp/governance_template.go`)
3. business resource inventory report (`business/*`, respecting the §9.3 business-domain boundary)

The pilot requires real semantics + real permissions + real export traces, not demo data.

## 15. Definition of Done

Phase one is done when:

- the unified report metadata model exists (the DB design's tables shipped with migrations)
- the report-center entry exists (menu + route + i18n + permission points registered together)
- the view/export action contract is implemented per API contract §3, with tests
- permission and audit constraints from §9/§13 each have a test or smoke case
- large-report limits are provable by tests (§8.4)
- at least one real report pilot is connected
- P0 cases of [REPORT_CENTER_TEST_PLAN.en.md](./REPORT_CENTER_TEST_PLAN.en.md) are green or explicitly waived

"Being a complete BI platform" is not the completion criterion.

## 16. Open decisions

| ID | Decision | Options | Impact | Owner |
| :--- | :--- | :--- | :--- | :--- |
| OD-1 | whether xlsx / pdf enter phase one | A: CSV only; B: add a pure-Go xlsx dependency; C: pdf via an external rendering service | B/C add dependencies (AGENTS: no unrequested dependencies); C also adds an external service boundary | maintainer |
| OD-2 | mount the entry under `platform` or `system` | A: `/api/v1/platform/report/*`; B: `/api/v1/system/report/*` | changes Casbin policy shape, reuse of `SystemProtectedGroup`, menu placement | maintainer |
| OD-3 | how datasets are registered | A: code registry (module seed, like menus/permissions); B: console-authorable (needs SQL whitelist governance) | B reintroduces the free-query entry risk; this design prefers A | maintainer |
| OD-4 | meaning of `visibility = team` | A: `system_dept` subtree; B: by role; C: no `team` in phase one | the repository has no "team" entity, only dept and role; C is the minimum viable | maintainer |
| OD-5 | whether artifact download re-checks row scope | A: re-validate the requester's scope on download; B: once generated, visible inside the tenant | A is safer but requires the artifact to record the scope fingerprint and be re-filterable | maintainer |
| OD-6 | whether the tenant classification must be written back to the scope matrix | A: the implementation PR updates `TENANT_RESOURCE_SCOPE_MATRIX.md`; B: a separate documentation task | this package only designs; the matrix must be synced at implementation time | maintainer |

## 17. Related Documents

| Document | Chinese | English |
| :--- | :--- | :--- |
| Main design (this file) | [REPORT_CENTER_DESIGN.md](./REPORT_CENTER_DESIGN.md) | `REPORT_CENTER_DESIGN.en.md` |
| Data model and migrations | [REPORT_CENTER_DB_DESIGN.md](./REPORT_CENTER_DB_DESIGN.md) | [REPORT_CENTER_DB_DESIGN.en.md](./REPORT_CENTER_DB_DESIGN.en.md) |
| API contract | [REPORT_CENTER_API_CONTRACT.md](./REPORT_CENTER_API_CONTRACT.md) | [REPORT_CENTER_API_CONTRACT.en.md](./REPORT_CENTER_API_CONTRACT.en.md) |
| Test plan | [REPORT_CENTER_TEST_PLAN.md](./REPORT_CENTER_TEST_PLAN.md) | [REPORT_CENTER_TEST_PLAN.en.md](./REPORT_CENTER_TEST_PLAN.en.md) |

Scheduler counterpart: [SCHEDULER_CENTER_DESIGN.md](./SCHEDULER_CENTER_DESIGN.md).
Directory index: [README.en.md](./README.en.md).

## Change log

| Date | Change |
| :--- | :--- |
| 2026-11-04 | Added data-source model, template versioning, render pipeline, export boundary, three-layer permission reuse, caching and large-report limits, scheduler interplay; added open decisions and the package index |
| 2026-10-31 | Initial draft |
