---
title: Notice Center Design
doc_type: Design
layer: platform
status: Draft
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
  - docs/contracts/TENANT_CONTRACT_V1.md
  - docs/contracts/SYSTEM_IAM_CONTRACT.md
  - docs/designs/NOTICE_CENTER_API_CONTRACT.md
updated_at: 2026-11-04
---

# Notice Center Design

Chinese version: [NOTICE_CENTER_DESIGN.md](./NOTICE_CENTER_DESIGN.md)

> Not implemented: this is a design draft with no corresponding implementation in the repository; it stays Draft until implemented.

Linked contracts:
- `PLATFORM_CONTRACT.md`
- `TENANT_CONTRACT_V1.md`
- `SYSTEM_IAM_CONTRACT.md`

Linked designs:
- `NOTICE_CENTER_DB_DESIGN.md`
- `NOTICE_CENTER_API_CONTRACT.md`
- `NOTICE_CENTER_TEST_PLAN.md`

Linked roadmaps:
- `P2_SCALE_ROADMAP.md`
- `PANTHEON_BASE_ARCHITECTURE_OVERVIEW.md`

---

This document defines the product and technical skeleton of the Pantheon platform notice center and is the entry point of the "Notice Center document package". It is a design draft, not an implementation report: every table name, route and permission key below is a contract to be implemented, and must not be claimed as available before implementation lands.

The package always has four documents with fixed responsibilities:

| Document | Owns |
|----------|------|
| `NOTICE_CENTER_DESIGN.md` (this file) | goals, boundaries, model, state machines, semantics, open decisions |
| `NOTICE_CENTER_DB_DESIGN.md` | tables / columns / indexes / migrations / purge |
| `NOTICE_CENTER_API_CONTRACT.md` | REST surface, DTOs, error codes, permissions, audit headers, i18n namespace |
| `NOTICE_CENTER_TEST_PLAN.md` | acceptance criteria → cases → existing gate per case → runtime gaps |

The goal is not to build an IM system, but to make the platform-level message surface governable, navigable and auditable through one entry point.

## 1. Design Goals

The notice center must solve:

1. platform-level messages stop living in the corners of individual system pages
2. cross-domain messages share one entry point, one read-state model and one jump model
3. the dashboard, the shell header and later task/alert summaries can share one message semantics
4. there is an **idempotent, retryable, traceable** delivery boundary between producers (source domains) and consumers (users), instead of every module writing into the table itself

## 2. Non-Goals

Out of scope at this stage:

- instant chat
- group chat / direct messages
- rich-text message composer
- multi-vendor push gateway (vendor push, mini-program subscribe messages, app badges)
- a full message template engine
- "who has seen this" collaborative read views (receipts stay at recipient granularity)

## 3. Ownership Boundary

### 3.1 What `platform` Owns

- the notice center entry and aggregate view
- the message model, recipient views and read/unread state
- the jump contract
- execution and ledger of channel delivery (including retries)
- the carrier of retention and purge

### 3.2 What System / Business Domains Own

- the business semantics of a message
- validity of the source data
- the detail page that a jump target lands on

In one sentence:

> `platform` closes the message entry point; it does not replace source-domain business.

Concrete boundary: source domains do **not** write the notice tables; they call the `platform` intake interface (in-process service call or the internal HTTP surface declared in §13.2). Justification is a citable clause: `docs/contracts/PLATFORM_CONTRACT.md` §7.1 states the aggregate layer may read several sub-domains but must not intrude into their responsibilities — the mirror direction applies: sub-domains must not hold the platform message tables either.

### 3.3 Division Against Adjacent Design Drafts

Four adjacent unimplemented drafts exist in this repository; the notice center does not absorb their concepts:

| Adjacent document | It owns | Notice center owns |
|-------------------|---------|--------------------|
| `NOTIFICATION_SERVICE_DESIGN.md` | channel capabilities (email/sms/websocket), sender-side API, user preferences | message body, recipient views, jumps, platform entry; channel execution calls into it |
| `ALERT_MONITORING_CENTER_DESIGN.md` | alert rules, event aggregation and grading | presenting already-graded alerts as `alert` messages to recipients |
| `TASK_CENTER_DESIGN.md` | actionable work items and their states | "assigned / due / result" messages about tasks |
| `PLATFORM_DASHBOARD_DESIGN.md` | homepage summary cards | the data source and jump semantics of those summaries |

`NOTIFICATION_SERVICE_DESIGN.md` already sketches a `system_notification` table (§2.1 of that document, not implemented). It and the three tables of this design **cannot both land**; the convergence path is registered as open decision OD-1 in §17.

### 3.4 Current Baseline (Falsifiable)

- The shell already has a notice-center placeholder entry: `frontend/src/core/layout/AppLayout.tsx` renders `app.notice.*` copy (`frontend/src/i18n/resources/zh-CN.ts`, starting at `app.notice.title`); its empty state literally says "no live notice source is wired yet", and the data shown comes from login/session/security events, not from a message model.
- There is no backend notice module: `backend/modules/platform/` currently contains only `dashboard_*` and `health_*`, and `backend/cmd/server/main.go:169` `RegisterPlatformRoutes` registers only `/dashboard/summary` plus health.
- There is no WebSocket infrastructure in the backend (no hits for `websocket`/`Upgrader` in shipping code), so "real-time push" is treated as not landed throughout this document.

## 4. Message Model

### 4.1 Body and Recipient View Are Separate

One message = **one body row**; every recipient = **one view row**. The reason is §9: read state must be stored per "recipient × message". Putting `read_at` on the body row means the second recipient can never read it.

- `system_notice`: message body (title/summary/category/severity/source domain/jump/payload/tenant)
- `system_notice_target`: recipient view (read/archived/delivered state)
- `system_notice_delivery`: out-of-app delivery ledger (written only when email/sms/webhook is enabled)

Column-level intent lives in `NOTICE_CENTER_DB_DESIGN.md` §4.

### 4.2 Fields

Minimum fields (merged external shape of body + view):

- `id`
- `category`
- `sourceDomain`
- `title`
- `summary`
- `severity`
- `status`
- `jumpType`
- `jumpTarget`
- `createdAt`
- `readAt`
- `actorId`

Additional constraint: `title` / `summary` store **rendered text**, not template keys; localization is decided by the source domain at intake time per recipient language (see open decision OD-2 in §17).

### 4.3 Category

Minimum categories:

- `task`
- `alert`
- `audit`
- `system`
- `business`

Category is **product semantics**, not a free-form source string. Adding a category means extending this enumeration first, then `system_dict` and i18n; otherwise the list filter degenerates into free input (see `docs/contracts/SYSTEM_CONFIG_CONTRACT.md`).

### 4.4 Severity

- `info`
- `warning`
- `critical`

`critical` may be highlighted in the list and gets its own unread counter bucket (§9.2), but it does **not** trigger an out-of-app channel by default: external delivery is decided by channel configuration, never by severity.

### 4.5 State

- `unread`
- `read`
- `archived`

State is derived from `system_notice_target`, not stored on the body: `archived` wins over `read`, `read` wins over `unread` (precedence in §5.1).

## 5. State Machines

This repository's `docs/designs/` does not use diagram syntax (zero `mermaid` hits in existing documents), so state machines are expressed as transition tables to avoid introducing a rendering dependency.

### 5.1 Recipient View State Machine

| Current | Event | Target | Precondition | Side effect |
|---------|-------|--------|--------------|-------------|
| `unread` | open detail | `read` | recipient is the current subject | write `read_at`; unread count decreases |
| `unread` | mark read | `read` | same | write `read_at` |
| `read` | archive | `archived` | same | write `archived_at` |
| `unread` | archive | `archived` | same | write `read_at` (if empty) and `archived_at` |
| `archived` | unarchive | `read` | same | clear `archived_at`; `read_at` is not rolled back |
| any | delete | view row removed (physical delete) | same | delete the `system_notice_target` row; the body and the audit facts survive (keeping a soft-deleted row would pin the unique key — `NOTICE_CENTER_DB_DESIGN.md` §5.3) |

Irreversibility: once `read_at` is written it is never cleared (§9.5). Unarchiving must not return the row to `unread`, otherwise "read" becomes a falsifiable field.

### 5.2 Delivery State Machine

`system_notice_target.delivered_at` and `system_notice_delivery.status`:

| State | Meaning | Outgoing |
|-------|---------|----------|
| `pending` | registered, waiting for a worker to claim it | `success` / `failed` / `skipped` |
| `success` | channel acknowledged delivery | terminal |
| `failed` | one attempt failed | `pending` (after backoff) / `dead` |
| `dead` | attempts exhausted | terminal, counted in the dead-letter metric |
| `skipped` | recipient preference or unconfigured channel, never attempted | terminal |

### 5.3 Illegal Transitions

Every write must validate "current state ∈ allowed source states" as a single conditional `UPDATE` (`WHERE status = ?`), not read-then-write. Basis:

- `unread → archived` is legal; `read → unread` is illegal and always returns `notice.state.invalid` from the API contract.
- `dead → pending` only happens through an explicit replay action (§8.3), never automatically.
- Body state (`draft` / `published` / `withdrawn`) and view state stay in separate columns; withdrawing does not delete recipient views, it filters and marks them in the list, preserving the traceability required by §12.

## 6. Jump Contract

Every message must declare:

- source domain
- jump target
- whether a permission is required

Recommended jump types:

- `route`
- `external-url`
- `none`

### 6.1 Jump Constraints

- without a stable target, `jumpType = none` is allowed
- faking a jump button is not allowed
- the jump is still subject to page permissions

### 6.2 Target Validation

- `route` must be a path known to the frontend route table: `frontend/src/core/router/componentRegistry.ts` maps the menu `Component` string to a lazy import, so jump validation is conditioned on the menu/route being registered — the notice side must not create a second path registry.
- `external-url` allows https only and only whitelisted hosts (configuration carrier registered in OD-3 of §17); validated before persisting and re-validated before rendering.
- when the source record is gone, the message is not reclaimed: the jump lands on the target page and that page returns its own empty state. The notice side performs no cross-domain "does the target exist" query — that would be the reverse dependency §3.3 forbids.

## 7. Channel and Transport Model

### 7.1 Channel Matrix

| Channel | Defined here | Repository status | Conclusion |
|---------|--------------|-------------------|------------|
| `in_app` | yes | not implemented, but only needs `system_notice_target`, no external dependency | the only channel V1 must deliver |
| `email` | yes (interface) | no SMTP dependency in the backend | contract and configuration slot only, no V1 promise |
| `sms` | yes (interface) | no cloud vendor dependency | same |
| `webhook` | yes (interface) | none | outbound HTTP client, requires SSRF constraints |
| `websocket` | yes (interface) | no WS infrastructure in the backend | registered as a runtime gap, see `NOTICE_CENTER_TEST_PLAN.md` §5 |

### 7.2 Channel Abstraction

```go
// backend/modules/platform/notice/channel/channel.go (to be implemented)
type Channel interface {
    Name() string
    Send(ctx context.Context, msg *Message) error
}
```

The signature stays identical to `NOTIFICATION_SERVICE_DESIGN.md` §3.2 so one capability does not grow two interfaces. Constraints:

- `Send` must be idempotent (§8.2): the input carries `deliveryID`, which the channel uses for external deduplication.
- `Send` must not block a request path: all out-of-app delivery runs in the worker (§8.2).
- `Send` must return an error on failure rather than swallowing it: same shape as the `Task.Run` contract in `backend/pkg/maintenance/maintenance.go`, whose comment states a task "should return an error instead of logging-and-continuing so failures stay measurable".

### 7.3 Delivery Unit

One delivery unit = `(notice_id, recipient_id, channel)`. `in_app` writes no ledger row and only updates `delivered_at` on the view row; every other channel writes `system_notice_delivery`. Reason: for `in_app` delivery and view creation are the same event, and a ledger row would only add a second source of truth.

### 7.4 User Preferences

Recipients may disable non-`critical` messages per channel. Whether preferences live in `system/config` or in a dedicated table is registered as OD-4 in §17; this document only locks "preferences affect channel selection, never message existence" — a suppressed message still gets a recipient view row, and its external delivery is recorded as `skipped`.

## 8. Idempotency and Retry

### 8.1 Intake Idempotency

Source domains must provide a `dedupKey` (a stable concatenation of `(sourceDomain, sourceType, sourceId, eventType, recipientId)` is recommended). The idempotency key lives on the **body table** as unique key `(tenant_id, dedup_key)`; fan-out uniqueness is guaranteed by the recipient table's `(notice_id, recipient_id)` (column types and the "no soft-delete column inside a MySQL unique key" trade-off are in `NOTICE_CENTER_DB_DESIGN.md` §5).

- unique-key hit → no new row, return the existing id with `deduplicated: true`; this is an **idempotent 2xx**, not a 409.
- empty `dedupKey` → allowed, but duplicate protection is lost; intake counts it in `notice_intake_missing_dedup_total` (§14) so "forgot to pass it" becomes a visible metric.

### 8.2 Delivery Retry and Backoff

| Parameter | Default | Semantics |
|-----------|---------|-----------|
| `maxAttempts` | 5 | including the first attempt |
| `backoffBase` | 30s | after failure n wait `base * 2^(n-1)` |
| `backoffCap` | 30min | backoff ceiling |
| `attemptTimeout` | 10s | per-`Send` context timeout |

Retries are driven by the `next_attempt_at` column, not by in-memory timers. The execution carrier reuses the existing background loop: `backend/pkg/maintenance/maintenance.go` (`Registry` with interval throttle, overlap guard, panic isolation and outcome metrics) plus the `maintenance.Run` switch wired at `backend/cmd/server/main.go:205-212`. Reason: that registry is already the mechanism for "batch scanning/purging outside request paths"; a second background loop would mean a second scheduling semantics to govern.

### 8.3 Dead Letters and Replay

Exhausted attempts → `dead`, keeping only readable diagnostics (`last_error` truncated, channel credentials must never be stored). Replay is an explicit administrative action requiring step-up validation (§13.2 `X-Operation-Token`); whether `attempts` resets to zero or keeps accumulating after replay is registered as OD-5 in §17.

### 8.4 At-Least-Once and Who Deduplicates

Delivery semantics are **at-least-once**: a crash between "sent" and "state persisted" produces a duplicate. The repository has no distributed lock primitive today (`grep SetNX backend/` returns nothing), so this document does not promise exactly-once. Deduplication is a two-layer responsibility: the channel side uses `deliveryID`, the recipient side uses the §8.1 `dedupKey`.

## 9. Read / Unread Semantics

### 9.1 Definition

- Read = the recipient view row has a non-null `read_at`. Trigger: a successful detail read.
- Archived is not the opposite of read; archived rows do not count as unread.
- "Opened" is not "finished reading": this design adds no reading-time or scroll-percentage signal, so UX guesses never enter the contract.

### 9.2 Unread Count

- The badge returns `(total, byCategory)`, with `critical` in its own bucket.
- Counting uses `COUNT(*)` on a covering index `(tenant_id, recipient_id, read_at)` and is not cached in-process. Reason: only reading the same rows across instances can stay consistent; the in-process cache pattern in `backend/modules/system/config/setting/setting_service.go:110` exists for rarely-changing configuration, not for per-request state.
- Cap display (`N+` beyond a threshold) is a frontend decision and stays out of the backend contract.

### 9.3 Batch Mark-Read

`mark-all-read` must carry a filter (`category` optional) and only ever acts on **the current subject's own rows**. Batch input limits follow existing practice: `backend/modules/system/audit/audit_service.go:357` rejects an id list `>500` with `param.invalid`; the notice side rejects at the binding layer rather than silently truncating.

### 9.4 Concurrency

Concurrent "mark read" versus "archive" is resolved by conditional update: `UPDATE ... SET read_at = ? WHERE id = ? AND read_at IS NULL`. `RowsAffected == 0` is not an error; it means "already updated concurrently", the response stays 2xx and the frontend re-fetches the counter. This is deliberately the opposite reading of the `RowsAffected == 0` ambiguity recorded in `docs/contracts/TENANT_CONTRACT_V1.md` §2.2: here `RowsAffected` is not used to distinguish "missing" from "already updated", because addressing is already pinned by primary key plus recipient predicate.

### 9.5 What Is Not Done

- read state is never revoked.
- no derived metrics such as "unread timeline".
- platform administrators cannot read other recipients' read state; cross-recipient reads only exist on the declared platform surface of §13 and are aggregate-only.

## 10. Retention and Purge

### 10.1 Retention Parameters Live in `system_setting`

Aligned with existing retention practice: `backend/modules/system/audit/audit_service.go:302-313` (`RunOperationLogRetention` reads `audit.operation_log_retention_days`, default 180) and `backend/modules/system/config/setting/setting_seed.go:65-68` (four `audit.*_retention_days` seeds plus `system.setting.remark.*` keys).

Suggested notice keys (to be implemented; names contain no hyphen so they match the i18n key regex in `backend/pkg/common/http/response.go:29`):

| setting key | suggested default | scope |
|-------------|-------------------|-------|
| `notice.read_target_retention_days` | 180 | retention of read/archived view rows |
| `notice.unread_retention_days` | 365 | retention of unread rows (never silently delete unread) |
| `notice.delivery_ledger_retention_days` | 90 | retention of the delivery ledger |

### 10.2 Purge Runs in the Maintenance Loop, Never in a Request

The purge is a `maintenance.Task` named `notice.retention`. Request paths never run a full-table delete — that is the positive requirement coming out of the historical defect recorded in the header comment of `backend/pkg/maintenance/maintenance.go` ("list/export handlers used to run full-table DELETE sweeps inline").

### 10.3 Hard Delete versus Soft Delete

- `system_notice_target`: hard delete at expiry (recipient views carry no audit value, and soft-deleting them would pin the §8.1 unique key forever).
- `system_notice`: soft delete (`deleted_at`), keeping the message fact for §12 traceability; withdrawal and deletion stay separate concepts.
- `system_notice_delivery`: hard delete at expiry, with `dead` rows surviving one extra retention period for troubleshooting.

### 10.4 Archive and Compliance

Messages that need audit-grade retention (`category = audit`) either opt out of §10.1 purging or are governed by a separate longer retention key; that ruling is OD-6 in §17.

## 11. UI Skeleton

First-phase minimum UI:

- header notice entry
- unread count badge
- notice center list page or drawer
- basic filters: all / unread / category

States that must be covered (`docs/contracts/PLATFORM_CONTRACT.md` §7.3 and the AGENTS.md rule "pages cover loading / empty / error / forbidden / submitting"):

| State | Concrete notice-center shape |
|-------|------------------------------|
| loading | list skeleton; the badge number must not flicker to 0 |
| empty | distinguish "no messages" from "no result after filtering" with different copy |
| error | keep the last data and offer retry; never clear the list |
| forbidden | if the entry is visible but the page answers 403, hide the badge instead of showing 0 |
| submitting | optimistic mark-read/archive, roll back the single row on failure |

Not required in phase one:

- complex grouping
- advanced search
- multi-column detail layout

All copy goes through i18n: the existing `app.notice.*` shell placeholder keys converge into the `notice.*` namespace at implementation time; the migration and key inventory are in `NOTICE_CENTER_API_CONTRACT.md` §8.

## 12. Audit and Trace

Record at least:

- where a notice was generated from
- a user viewing it
- a user marking it read
- a user archiving it

The notice center is not an audit system, but it must be able to answer "where did this message come from and who touched it".

Mechanism constraints (not wishes):

- the global operation log persists **every non-GET** request: `backend/internal/middleware/operation_log_middleware.go:224-231` (a `MethodGet` request short-circuits without logging). Every write endpoint of the API contract therefore already has an audit row without extra wiring.
- "view" is a GET and leaves no automatic trace. This design does **not** write an audit row per view (it would flood the audit surface); instead `system_notice_target.read_at` is kept as the minimal fact field. Stronger view auditing belongs to OD-7 in §17.
- administrative writes (withdraw, replay, purge) must set an explicit audit title and business type via `common.SetAuditMetadata(c, titleKey, businessType)` (`backend/pkg/common/audit.go:31`), using `BusinessDelete = 3` / `BusinessGrant = 4` (same file, lines 20-23), the same shape as `backend/modules/system/system_modules.go:535` (operation-log cleanup).

## 13. Permission, Menu and i18n Impact

A new platform page touches permissions, routes, menus and i18n at once; all four must land in the same change (the AGENTS.md "close the sync" rule).

### 13.1 Permission Keys

| key | meaning |
|-----|---------|
| `platform:notice:view` | page admission plus reading one's own list |
| `platform:notice:update` | mark read / archive / delete one's own view rows |
| `platform:notice:publish` | source domain or administrator publishing a message |
| `platform:notice:manage` | withdraw, replay, purge (destructive, step-up required) |

Naming follows `platform:<resource>:<action>`, the same prefix and level as the existing `platform:dashboard:view` (`backend/modules/system/seed.go:210`); the key regex in `scripts/harness/check-permission-contract.mjs` only recognizes the `system|business|platform` three-part prefix, so stepping outside it removes gate-level attribution.

### 13.2 Route Group and Middleware

- The user surface and the platform surface both mount under a `notice` group below `/api/v1`, matching the dashboard precedent at `backend/modules/platform/routes.go:34-37`: `TokenAuth → PlatformScopeTenantContext → Casbin`.
- A system-console form using `contracts.SystemProtectedGroup` inherits the fixed chain `TokenAuth → PlatformScopeTenantContext → Casbin → (optional Capability/DataScope/RefreshSync)`; the order is a security property, not a style choice (`backend/pkg/contracts/route_groups.go:137-150`).
- If a business domain ingests through `contracts.ProtectedGroup`, it must know that chain installs **no** tenant middleware (lines 13-26 of the same file plus `docs/contracts/TENANT_CONTRACT_V1.md` §2.6 record that this is a contract requirement, not an oversight), so tenant attribution on the intake side must come from explicit input plus server-side resolution, never from an assumed context value.
- Destructive actions (withdraw / purge / replay) add `middleware.SecureActionMiddleware()`: it validates `X-Operation-Token` and answers 403 with `auth.operation.verification_required` / `verification_expired` / `verification_mismatch` for missing / expired / mismatched tokens, also checking token user and session identity (`backend/internal/middleware/secure_action_middleware.go:16-53`).

### 13.3 Menu Seed

Menu rows land through `menuSeed` (`backend/modules/system/seed.go:33-50`, fields `Key/ParentKey/TitleKey/Path/Component/PagePerm/Perms/Type/Icon/RouteName/Module/Sort/HideInNav`); `ensureMenuSeeds` handles idempotency, de-duplication and obsolete-menu cleanup (same file, lines 88-103). The frontend `Component` string must resolve in `frontend/src/core/router/componentRegistry.ts`, otherwise the menu is visible but the route unreachable.

### 13.4 i18n

Canonical i18n rows land in `system_i18n` keyed by `(locale, key)`, and every key must ship all five locales (`canonicalMenuLocaleEntries` starting at `backend/modules/system/i18n/i18n_seed.go:108`; the locale set is `zh-CN/en-US/ja-JP/ko-KR/fr-FR` per `backend/modules/system/i18n/i18n_helper.go:815`). Menu titles, permission descriptions, error keys and page copy all need five languages; a missing locale shows up as "key resolves but the UI falls back to the default language", which is a measurable defect.

### 13.5 Gate Sync Checklist

| Change | Must sync |
|--------|-----------|
| new route | `requiredAPIRoutesByPermission` at `backend/modules/system/iam/permission/permission_workbench.go:622` plus the engine probe in `permission_workbench_routes_test.go` (lines 602-620 state that drift is caught there) |
| new menu/permission | `menuSeed` list + five-locale i18n + `npm run test:menu-contract` |
| new table | `docs/designs/TENANT_RESOURCE_SCOPE_MATRIX.md` §3 unique-key registration (`docs/contracts/TENANT_CONTRACT_V1.md` §2.3 requires registering before adding a table) |
| new document | `docs/designs/README.md` / `README.en.md` index + `npm run check:harness-docs -- --strict` + `npm run check:docs-frontmatter` |

## 14. Metrics and Observability

Metrics are exposed on the existing Prometheus surface: `backend/pkg/metrics/prometheus.go` defines `pantheon_http_requests_total` / `pantheon_http_request_duration_seconds` (HTTP instrumentation in `backend/internal/middleware/prometheus_middleware.go`), and `/metrics` is closed by default, requiring `PANTHEON_METRICS_BEARER_TOKEN` or `PANTHEON_METRICS_PUBLIC=true` (`backend/cmd/server/main.go:150-156`). Background tasks already have one reusable dimension: `metrics.MaintenanceRunsTotal{name, outcome}` via `recordOutcome` in `backend/pkg/maintenance/maintenance.go`.

Suggested notice metrics (to be implemented, keeping the `pantheon_` prefix):

| Metric | Type | Labels | Question it answers |
|--------|------|--------|---------------------|
| `pantheon_notice_intake_total` | counter | `source_domain, category, deduplicated` | who sends messages and how often they duplicate |
| `pantheon_notice_intake_missing_dedup_total` | counter | `source_domain` | which source domains omit the idempotency key |
| `pantheon_notice_delivery_total` | counter | `channel, outcome` | per-channel delivered / failed / skipped |
| `pantheon_notice_delivery_duration_seconds` | histogram | `channel` | channel latency distribution |
| `pantheon_notice_delivery_dead_total` | counter | `channel` | dead-letter accumulation |
| `pantheon_notice_pending_age_seconds` | gauge | `channel` | age of the oldest undelivered message (the only signal that reveals a stuck worker) |
| `pantheon_notice_retention_deleted_rows` | counter | `table` | whether the purge actually runs |

## 15. Relationship with Dashboard and Task Center

The dashboard must not keep its own copy of the message model.

Correct relationship:

- the dashboard only consumes notice center summaries
- the notice center is the platform-level message entry
- business domains expose summaries to the platform through the unified message model

Landing constraint: the dashboard already aggregates governance to-dos (`OrgGovernanceTaskCount` / `OrgGovernanceTasks` in `backend/modules/platform/dashboard_dto.go:23-24`). That is a work-item summary, not a message summary; at implementation time the two must not be merged into one dataset, otherwise the dashboard becomes a second message store — exactly the aggregate-layer overreach `PLATFORM_CONTRACT.md` §7.1 forbids.

## 16. Landing Constraints (Existing Mechanisms)

Every constraint in this document maps to a mechanism that already exists, so implementers do not have to invent one:

| Clause here | Existing mechanism | Source (file actually read) |
|-------------|--------------------|-----------------------------|
| §8.2 background carrier | maintenance registry (throttle / overlap / panic isolation / metrics) | `backend/pkg/maintenance/maintenance.go` |
| §10 purge off the request path | defect recorded in that registry's header comment | same file, lines 4-9 |
| §10.1 configurable retention | `system_setting` plus default fallback | `backend/modules/system/audit/audit_service.go:302-313, 317-323, 325` |
| §12 automatic audit for non-GET | operation log middleware skips GET | `backend/internal/middleware/operation_log_middleware.go:224-231` |
| §13.2 middleware chain | `SystemProtectedGroup`, order is a security property | `backend/pkg/contracts/route_groups.go:137-150` |
| §13.2 step-up | `X-Operation-Token` validation | `backend/internal/middleware/secure_action_middleware.go:16-53` |
| §13.5 permission↔route drift | workbench binding table plus engine probe test | `backend/modules/system/iam/permission/permission_workbench.go:602-652` |
| §9.3 batch cap | existing `>500 → param.invalid` guard | `backend/modules/system/audit/audit_service.go:357` |
| §14 metric exposure | Prometheus registry and endpoint switch | `backend/pkg/metrics/prometheus.go`, `backend/cmd/server/main.go:150-156` |
| §13.4 five-locale i18n | canonical locale entries plus supported locale set | `backend/modules/system/i18n/i18n_seed.go:108+`, `i18n_helper.go:815` |

## 17. Open Decisions

The following are product or architecture choices this document must not settle silently. Each lists options and a recommendation; the maintainer decides at the quality-gate / acceptance touchpoint:

| ID | Topic | Options | Recommendation and reason | Decider |
|----|-------|---------|---------------------------|---------|
| OD-1 | how to converge with the `system_notification` table of `NOTIFICATION_SERVICE_DESIGN.md` | A notice center owns all three tables and the draft table is retired; B notice center keeps body+view and channel tables belong to the notification service | A: both are unimplemented, and landing both creates two recipient states; the channel abstraction of §7.2 stays shared, so nothing in the other document's semantics is lost | maintainer |
| OD-2 | who renders localized content | A source domain renders per recipient language at intake; B store template key plus variables and render platform-side | A for V1: a platform template engine is a non-goal in §2; B needs a template governance contract first | maintainer |
| OD-3 | carrier of the `external-url` host whitelist | A `system_setting`; B dedicated channel config table; C environment variable | A: same surface as the §10.1 retention settings, avoids a new configuration store | maintainer |
| OD-4 | ownership of user notification preferences | A `system_user.preference_json`; B user-scoped `system/config` settings; C dedicated table | before choosing C, first decide whether V1 needs preferences at all: §7.4 only promises "message existence is unaffected", so this can wait until an out-of-app channel exists | maintainer |
| OD-5 | does `attempts` reset after dead-letter replay | A reset (new lifecycle); B keep accumulating (visible total cost) | A plus a separate `total_attempts` observation column: closer to the meaning of an operational action | maintainer |
| OD-6 | is `category = audit` exempt from purge | A exempt; B its own longer retention key; C purged like everything else | B: exemption lets unbounded "unread audit messages" accumulate | maintainer |
| OD-7 | does every view need auditing | A store `read_at` only; B write an audit row per GET | A: §12 explains that B floods the audit surface | maintainer |
| OD-8 | intake as in-process service or internal HTTP | A in-process (composition root injects the port); B declared internal HTTP endpoint | A: cross-module calls already use `pkg/contracts` ports (see the authuser port comment at `backend/cmd/server/main.go:172-176`); a new HTTP surface means a second auth and idempotency regime | maintainer |

## 18. Definition of Done

Phase one is done when:

- a unified message model exists
- read/unread semantics exist
- the platform entry exists
- the minimal jump contract exists
- basic audit tracing exists
- the four synchronizations of §13 (permission / menu / i18n / document) and the metrics of §14 exist
- at least one real source domain is connected through the §8.1 idempotent intake

"Feature-parity with WeCom" is not the completion criterion.

Verification is governed by `NOTICE_CENTER_TEST_PLAN.md`; all four documents of this package stay `Draft` until that plan lands.
