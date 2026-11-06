---
title: Report Center API Contract
doc_type: Design
layer: platform
status: Draft
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
  - docs/contracts/SYSTEM_IAM_CONTRACT.md
  - docs/contracts/TENANT_CONTRACT_V1.md
updated_at: 2026-11-04
---

# Report Center API Contract

> 未实现声明：本文为设计稿，仓库内尚无对应实现；在实现落地前状态保持 Draft。
> Not implemented: this is a design draft with no corresponding implementation in the repository; it stays Draft until implemented.

Main design: [REPORT_CENTER_DESIGN.en.md](./REPORT_CENTER_DESIGN.en.md)
Data model: [REPORT_CENTER_DB_DESIGN.en.md](./REPORT_CENTER_DB_DESIGN.en.md)
Test plan: [REPORT_CENTER_TEST_PLAN.en.md](./REPORT_CENTER_TEST_PLAN.en.md)

---

## 1. Mount point and route chain

- Prefix: `/api/v1` (the `registerAPIRoutes` group at `backend/cmd/server/main.go:159`).
- This contract is written using **option A** of main design §16 OD-2: `/api/v1/platform/report/*`.
  If the ruling switches to option B (`/system/report/*`), only the prefix and the menu ownership change;
  DTOs, error keys and permission keys stay identical.
- The route chain must mirror the existing system-domain console chain
  (`backend/pkg/contracts/route_groups.go:137`):
  `TokenAuth -> PlatformScopeTenantContext -> Casbin`, optionally extended with
  `PlatformCapability` / `SecureAction`. The dataset/template **list, detail and management** group does
  **not** install `DataScopeMiddleware` (a report definition is not department-owned data, see the end of
  main design §9.2); the **render and export** handlers must consume `common.GetDataScope(c)`, so the only
  groups that need the middleware are `run` / `export` / `artifact download`
  (`route_groups.go:70-84` states explicitly that "omitting an option is itself a permission decision").

## 2. Unified envelope and error semantics

| Fact | Source |
| :--- | :--- |
| Envelope `{code, data, message}`; on success `message = "success"` | `backend/pkg/common/http/response.go:15-38` |
| `Fail/FailWithData` write **HTTP 200**, business code lives in `code` | same file `:47-60` |
| `FailWithCode` writes the **real HTTP status** (use it for the 403 branch) | same file `:113-118` |
| `message` must be an i18n key shaped `^[a-z0-9_]+(?:\.[a-z0-9_]+)+$` | same file `:28` |

Two implementation constraints follow:

1. No error key in this contract may contain a hyphen (`report.code.conflict` is legal;
   `report-code-conflict` would be treated as natural language and passed through to the frontend
   verbatim), consistent with [ERROR_CODE_AND_I18N.en.md](./ERROR_CODE_AND_I18N.md) §3.1:
   the backend never returns natural language.
2. Permission denial reuses the existing Casbin middleware key `permission.denied`
   (`failPermissionCheck` in `backend/internal/middleware/casbin_middleware.go`, business code 403);
   a missing second factor reuses
   `auth.operation.verification_required` / `auth.operation.verification_expired`
   (`backend/internal/middleware/secure_action_middleware.go`). The report side adds **no** synonym keys.

The set of business codes stays as defined in `backend/pkg/common/http/response.go:21-26`:
`200 / 400 / 401 / 403 / 404 / 500`.

## 3. Route inventory

| # | Method | Path | Permission key | Extra chain items | Notes |
| :--- | :--- | :--- | :--- | :--- | :--- |
| 1 | GET | `/platform/report/dataset/list` | `platform:report:list` | — | Read-only registry of available datasets (OD-3 option A: no write API) |
| 2 | GET | `/platform/report/template/list` | `platform:report:list` | — | Paginated; filtered by `visibility`/`category`/`sourceDomain` |
| 3 | GET | `/platform/report/template/:id` | `platform:report:view` | — | Includes the presentation spec of the current published version |
| 4 | POST | `/platform/report/template` | `platform:report:create` | — | Creates template plus its first draft version |
| 5 | PUT | `/platform/report/template/:id` | `platform:report:update` | — | Edits the draft only, never a published version |
| 6 | DELETE | `/platform/report/template/:id` | `platform:report:delete` | **SecureAction** | Soft delete + code rename release (DB design §5) |
| 7 | GET | `/platform/report/template/:id/version/list` | `platform:report:view` | — | Version history (paginated) |
| 8 | POST | `/platform/report/template/:id/publish` | `platform:report:update` | **SecureAction** | Publishes the draft as an immutable version, advances `current_version_id` |
| 9 | POST | `/platform/report/template/:id/rollback` | `platform:report:update` | **SecureAction** | Points the pointer back to a previously published version |
| 10 | POST | `/platform/report/run` | `platform:report:run` | DataScope | Synchronous render for preview; row caps in §5 |
| 11 | POST | `/platform/report/export` | `platform:report:export` | DataScope (plus SecureAction when `sensitive`) | Export: small reports return CSV directly, large reports return a `runId` for the async artifact |
| 12 | GET | `/platform/report/run/list` | `platform:report:view` | — | History (paginated) |
| 13 | GET | `/platform/report/run/:id` | `platform:report:view` | — | Single execution detail |
| 14 | GET | `/platform/report/artifact/:id/download` | `platform:report:export` | DataScope | Artifact download; expired returns `report.artifact.expired` |
| 15 | POST | `/platform/report/run/cleanup` | `platform:report:clear` | **SecureAction** | Manual retention cleanup (same implementation as the maintenance task) |

Rationale for the SecureAction selection: same shape as existing high-sensitivity precedents
(`backend/modules/auth/module.go:97-108` for cleanup/batch operations,
`backend/modules/system/system_modules.go:188-191` for batch delete and password reset).

## 4. DTO contract

Naming follows `DTO_LIKE_SUFFIXES` in `scripts/harness/check-backend-dto-contract.mjs`
(`Resp` / `Result` / `Summary` / `Option` and friends); handlers must not return bare `map[string]any`.

### 4.1 Request

```jsonc
// POST /platform/report/run  and  POST /platform/report/export
{
  "templateId": 1024,
  "versionId": 0,             // 0 = follow current_version
  "format": "csv",            // only values present in report_template.supported_formats
  "params": {                 // constrained by the report_dataset.param_schema_json whitelist
    "dateFrom": "2026-10-01",
    "dateTo": "2026-10-31",
    "deptIds": [3, 7]
  },
  "idempotencyKey": "",       // optional; if empty the server derives it per §9
  "async": false              // true returns runId immediately without row data
}
```

### 4.2 Response

```jsonc
// GET /platform/report/template/list -> data
{
  "list": [
    {
      "id": 1024,
      "tenantId": 0,
      "code": "iam-user-governance",
      "name": "system.menu.report.iam_user_governance",
      "reportType": "governance",
      "category": "governance",
      "sourceDomain": "system.iam",
      "datasetCode": "iam.user.governance",
      "visibility": "platform",
      "supportedFormats": ["csv"],
      "sensitive": false,
      "status": "published",
      "currentVersionId": 2048,
      "updatedAt": "2026-11-04T10:00:00+08:00"
    }
  ],
  "total": 1,
  "page": 1,
  "pageSize": 20
}

// POST /platform/report/run -> data (ReportRunResultResp)
{
  "runId": 4096,
  "status": "succeeded",
  "rowCount": 137,
  "truncated": false,
  "durationMs": 412,
  "columns": [
    { "key": "username", "titleKey": "system.column.username", "type": "string" }
  ],
  "rows": [ { "username": "alice" } ],
  "asOfAt": "2026-11-04T10:05:00+08:00",
  "scopeMode": "dept_and_children",
  "artifactId": 0
}

// POST /platform/report/export -> data (ReportExportResp)
{
  "mode": "sync",             // for "sync" the response body is already the CSV stream; shown for the async branch
  "runId": 4097,
  "artifactId": 0,
  "statusUrl": "/api/v1/platform/report/run/4097"
}
```

Field constraints:

| Field | Constraint | Basis |
| :--- | :--- | :--- |
| `rows` | preview rows <= `min(dataset.row_limit, 10000)` | `backend/pkg/impexp/export_limits.go:9` |
| `scopeMode` | read-only echo; values come from `backend/pkg/common/data_scope.go:11-15` | tells the user "which scope this run was computed under" |
| `truncated` | must match `status = truncated`; the frontend has to surface it explicitly | main design §8.4 |
| `name` | stores an i18n key, never natural language | [DESIGN.md](../../DESIGN.md) §6 |
| `asOfAt` | derived from `freshness_seconds`; on a cache hit it is still the real data-as-of time | main design §10 |

## 5. Pagination, sorting and caps

| Item | Rule | Basis |
| :--- | :--- | :--- |
| Pagination params | `page` (default 1), `pageSize` (default 10, **clamped server-side to 100**) | `common.NormalizePagination` (`backend/pkg/common/handler.go:39`); system-domain handlers pass 100 consistently |
| Sorting params | `sort` + `order`, field must pass a whitelist | `common.NormalizeSort` (`backend/pkg/common/handler.go:57`) |
| Rendered rows | synchronous cap 10000; beyond that must go async | `impexp.MaxExportRows()` |
| Time fields | list responses return timezone-carrying RFC3339; the frontend renders in the user timezone | `check:system-datetime-presentation` (`frontend/package.json`) |

## 6. Permission key register

| Key | Semantics | Frontend button permission | Menu attachment point |
| :--- | :--- | :--- | :--- |
| `platform:report:list` | access to the report center and its inventory | no (page admission) | Report Center menu |
| `platform:report:view` | view a single report and its history | yes | Report Center / View |
| `platform:report:run` | execute a render (preview) | yes | Report Center / Run |
| `platform:report:export` | export and download artifacts | yes | Report Center / Export |
| `platform:report:create` | create a template | yes | Template management / Create |
| `platform:report:update` | edit and publish | yes | Template management / Edit, Publish |
| `platform:report:delete` | delete a template | yes | Template management / Delete |
| `platform:report:clear` | clean up runs / artifacts | yes | Template management / Cleanup |

Existing keys of the same shape (copy the shape, do not invent vocabulary): `view` as an action word is
already used at `backend/modules/system/seed.go:210` (`PagePerm: "platform:dashboard:view"`), and a cleanup
action bound to a `Type: "F"` node uses `clear` — `backend/modules/system/seed.go:548`
(`system:operation-log:clear`).

Sync obligations required by the contracts (an implementation without them counts as unfinished):

1. Every key needs a Casbin `p` policy and a permission-point registration. The naming shape follows the
   existing scan convention `platform:<resource>:<action>` (`PERMISSION_KEY_PATTERN` at
   `scripts/harness/check-permission-contract.mjs:30`). The checker's **real constraint is not an action-word
   whitelist**: `ACTION_TOKENS` (`:11-28`) only feeds its heuristic context scan; the blocking finding it
   raises is "an action menu node (`Type: \"F\"`) bound to a `...:list` permission" (`:125-136`), plus a
   non-blocking warning when `:list` appears next to action wording (Go `:138-165`, TS `:170-208`).
   All eight keys in this table are therefore implementable, but the implementation **must** keep
   read/navigation gating on `:list` and bind every action node to an action key — never fall back to
   `list` for an action ([DESIGN.md](../../DESIGN.md) §4.4).
2. Permission points must be registered both as menu nodes (`Type = "F"`) and in the i18n permission group;
   follow the shape of `backend/modules/auth/login/login_menu.go:81-89`
   (`Key / ParentKey / TitleKey / Perms / Type / Sort`).
3. A list permission must never fall back to a write permission ([DESIGN.md](../../DESIGN.md) §4.4).

## 7. i18n namespace

| Group | Prefix | Example keys |
| :--- | :--- | :--- |
| menu | `platform.menu.report_*` | `platform.menu.report_center` |
| permission | `platform.permission.report_*` | `platform.permission.report.export` |
| messages (errors) | `report.*` | `report.render.row_limit_exceeded` |
| page copy | `platform.report.*` | `platform.report.truncated.tip` |

Registration format: the `{Module, Locale, Group, Key, Value}` tuples in
`backend/modules/system/i18n/seed_data.go` (example at `:79`); both the zh-CN and en-US copies must ship
together (paired locales appear throughout that file). No hardcoded copy in the frontend —
`check:i18n-hardcode` blocks it (the `prebuild` chain in `frontend/package.json`).

## 8. Audit conventions

| Action | Recording mechanism |
| :--- | :--- |
| All non-GET actions | operation log middleware writes `system_log_oper` automatically (`backend/internal/middleware/operation_log_middleware.go`) |
| Export / download | handler explicitly sets the title key and source domain; `oper_param` stores a parameter digest, not the raw payload ([DATABASE.en.md](./DATABASE.md) §3.2 sanitization discipline) |
| Render execution | the `report_run` row itself (including `scope_fingerprint`, `param_digest`) |
| High-sensitivity actions | both the SecureAction 403 and the success leave traces: the 403 via the middleware error key, the success via the operation log |

## 9. Idempotency and concurrency

1. **Export idempotency**: when `idempotencyKey` is absent the server derives
   `sha1(tenantId|templateId|versionId|format|paramDigest|scopeFingerprint)`. A hit on a non-expired
   artifact is reused without re-rendering (main design §11).
2. **Concurrency**: concurrent requests sharing one `idempotencyKey` allow only a single real render;
   the others wait and reuse the result. The implementation must prove "two concurrent requests produce
   exactly one data fetch" (test plan TC-R-CON-2).
3. **No side effects**: apart from writing `report_run` / `report_artifact`, `run` / `export` mutate no
   business table.
4. Publishing twice in a row (same draft, two consecutive publishes) must return `report.version.immutable`
   on the second call and must not create a second version row.

## 10. Compatibility and versioning strategy

- Phase one introduces no URL version segment (stays on `/api/v1`); breaking contract changes follow
  [PLATFORM_CONTRACT.en.md](../contracts/PLATFORM_CONTRACT.md) §7.5 and the documentation governance
  contract process.
- Adding a value to `supportedFormats` is a **compatible extension**, but only after the dependency
  preconditions of main design §8.2 are met.
- Fields may be added, never redefined; if `scopeMode` / `asOfAt` are missing, the frontend must degrade to
  "unknown scope" and refuse to render.

## 11. Open decisions

| ID | Decision | Options | Decision owner |
| :--- | :--- | :--- | :--- |
| OD-API-1 | Whether `run` (rendering) needs a permission key separate from `view` (reading) | A: separate key (current contract — rendering is a cost action and should be grantable apart from read-only viewing); B: merge into `platform:report:view` and rely on the row cap plus rate limiting | Maintainer |
| OD-API-2 | Progress query shape for large async reports | A: poll `GET /run/:id`; B: add an SSE event stream | Maintainer (phase one leans to A) |
| OD-API-3 | Whether artifact download uses short-lived signed URLs | A: always through the in-app download endpoint; B: presigned URLs when the S3 driver is active | Maintainer (B needs tenant re-validation, see main design OD-5) |
| OD-API-4 | Whether `run` and `export` should merge into one endpoint | A: keep them separate (current contract); B: merge into `POST /platform/report/render` distinguished by `mode` | Maintainer |

## 12. Related documents

- Main design: [REPORT_CENTER_DESIGN.md](./REPORT_CENTER_DESIGN.md) / [REPORT_CENTER_DESIGN.en.md](./REPORT_CENTER_DESIGN.en.md)
- Data model: [REPORT_CENTER_DB_DESIGN.md](./REPORT_CENTER_DB_DESIGN.md) / [REPORT_CENTER_DB_DESIGN.en.md](./REPORT_CENTER_DB_DESIGN.en.md)
- Test plan: [REPORT_CENTER_TEST_PLAN.md](./REPORT_CENTER_TEST_PLAN.md) / [REPORT_CENTER_TEST_PLAN.en.md](./REPORT_CENTER_TEST_PLAN.en.md)
- Scheduler center API: SCHEDULER_CENTER_API_CONTRACT.en.md（planned/unavailable：`./SCHEDULER_CENTER_API_CONTRACT.md`；无当前交付承诺）
- Index: [README.en.md](./README.en.md)
