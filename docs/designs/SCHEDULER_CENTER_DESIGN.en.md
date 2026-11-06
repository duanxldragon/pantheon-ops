---
title: Scheduler Center Design
doc_type: Design
layer: platform / system/config / business/*
status: Draft
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
  - docs/contracts/SYSTEM_CONFIG_CONTRACT.md
  - docs/contracts/TENANT_CONTRACT_V1.md
updated_at: 2026-11-04
---

# Scheduler Center Design

> 未实现声明：本文为设计稿，仓库内尚无对应实现；在实现落地前状态保持 Draft。
> Not implemented: this is a design draft with no corresponding implementation in the repository; it stays Draft until implemented.

Related contracts:
- [PLATFORM_CONTRACT.en.md](../contracts/PLATFORM_CONTRACT.md)
- [SYSTEM_CONFIG_CONTRACT.en.md](../contracts/SYSTEM_CONFIG_CONTRACT.md)
- [TENANT_CONTRACT_V1.en.md](../contracts/TENANT_CONTRACT_V1.md)

Related roadmaps:
- [P2_SCALE_ROADMAP.md](./P2_SCALE_ROADMAP.md)
- [PANTHEON_BASE_ARCHITECTURE_OVERVIEW.md](./PANTHEON_BASE_ARCHITECTURE_OVERVIEW.md)

This document defines the "scheduler center" product surface of the Pantheon standard admin backoffice.
The goal is not to grow into a heavyweight scheduling platform, but to establish one entry point for
scheduled work, one execution-record view and one governance boundary for integrations — and to close a
runtime fact that already exists in the repository: **today every periodic task runs independently in every
replica, with no cross-instance mutual exclusion, no execution history and no enable/pause control.**

---

## 1. Design Goals

The scheduler center exists to solve:

1. system-level and business-level periodic work has one registration and inventory entry
2. job definition, enable/pause state and last execution result are expressed consistently
3. high-sensitivity execution actions carry permission and audit trails
4. **a job executes once under multi-replica deployment** (this repository is deployed as multiple
   replicas, see §7)
5. execution outcomes are traceable (history, duration, error key, retry index) so the alert center
   and the dashboard can consume them

## 2. Non-Goals

Out of scope at this stage:

- a distributed job orchestration platform
- a visual DAG orchestrator
- a multi-cloud batch console
- an arbitrary remote script executor
- replacing all domain-local execution logic as a general execution engine
- inter-job dependency orchestration (A triggers B); the only cross-component coordination that already
  exists in the repository is the report center explicitly calling the scheduler contract (§14)
- sub-second scheduling and real-time stream triggering

## 3. Ownership Boundary

### 3.1 `platform` owns

- the scheduler center product surface
- aggregation of the registered job inventory
- the execution record overview
- the unified status summary
- the tick loop and lease acquisition (§6/§7)

### 3.2 `system/config` owns

- platform-level job configuration governance (default execution policy, retention, default timezone)
- high-sensitivity operation protection policy
- boundary discipline from
  [SYSTEM_CONFIG_CONTRACT.en.md](../contracts/SYSTEM_CONFIG_CONTRACT.md) §7.4: `setting` emits configuration
  values but does not take over consumption semantics — scheduling semantics are interpreted by the
  scheduler center

### 3.3 Source domains own

- the actual job logic (handler implementation)
- job parameter validation
- business meaning of the execution result, and their own domain-specific side effects
- **handler idempotency**: re-delivery of the same idempotency key must produce the same result (§9)

The layering direction is unchanged: `platform` never imports a concrete source-domain implementation;
handlers are injected by the composition root (same discipline as the `OrgGovernanceTaskLoader`
injection documented at `backend/modules/platform/routes.go:13-16`).

## 4. Existing Facts: What "Periodic Tasks" Mean in This Repository Today

This is the foundation of the design. Without it the scheduler center would become a second system
standing next to the mechanism that already exists.

| Fact | Source |
| :--- | :--- |
| Periodic task registry `maintenance.Registry`; a task is only `Name / Interval / Run` | `backend/pkg/maintenance/maintenance.go:31-35` |
| Default interval is 15 minutes; tasks registered without an interval get it | `backend/pkg/maintenance/defaults.go:8-10`, `maintenance.go:99-101` |
| Registration is idempotent and preserves existing throttle state | `maintenance.go:92-114` |
| Throttle plus overlap protection, and **overlap is evaluated before the throttle** | `maintenance.go:171-198` (`:181-187` overlap, `:188-191` skip) |
| Failures and panics are isolated: a panic becomes an error so one broken task cannot kill the loop | `maintenance.go:226-237` |
| Every run is instrumented (counter per task and outcome + duration histogram) | `maintenance.go:239-244`, `backend/pkg/metrics/prometheus.go:98-112` |
| Blocking runner: first pass runs immediately, then a ticker | `maintenance.go:246-267` |
| The runner starts as a goroutine **in every process** and can be switched off by env | `backend/cmd/server/main.go:204-213` (`PANTHEON_MAINTENANCE_ENABLED`, `PANTHEON_MAINTENANCE_INTERVAL_SECONDS`) |
| Registered real tasks: `auth.login_log_retention`, `auth.security_event_retention`, `auth.session_inventory`, `audit.operation_log_retention` | `backend/modules/auth/login/login_service.go:51`, `backend/modules/auth/security/security_service.go:120`, `backend/modules/auth/session/session_service.go:83`, `backend/modules/system/audit/audit_service.go:30` (each registered by `RegisterMaintenanceTasks` in the same file) |
| Retention days are read from `system_setting` with a code default fallback | `backend/modules/system/audit/audit_service.go:325-341` |

Conclusions (three facts that must be inherited as-is):

1. The registry's state is **process-local** (`sync.Mutex` + `map`, `maintenance.go:70-75`), so the overlap
   protection only holds inside one process. With three rolling replicas the same sweep runs three times.
   That is tolerable today because the tasks are idempotent, but it is the source of the "no cross-instance
   exclusion" gap and must not be described as a capability.
2. The existing model is **fixed-interval**: no cron, no timezone, no execution history, no enable/pause
   switch, no permission or audit around a manual trigger.
3. Deleting work has already been moved out of request paths and must not be moved back into handlers
   (the package comment at `maintenance.go:1-12` records the lesson: an ordinary read once paid for an
   unbounded purge).

## 5. Job Model

All draft fields (`id / jobType / sourceDomain / name / description / scheduleType / scheduleExpr /
enabled / lastRunAt / lastRunStatus / lastRunSummary / owner`) are kept and extended with the dimensions
implementation needs, including the next-run value and the jump target the English draft asked for.
Column-level definitions are in
SCHEDULER_CENTER_DB_DESIGN.en.md（planned/unavailable：`./SCHEDULER_CENTER_DB_DESIGN.en.md`；无当前交付承诺） §2.

| Group | Fields | Constraint source |
| :--- | :--- | :--- |
| Identity | `job_key` (globally unique, dotted domain prefix, e.g. `auth.login_log_retention`) | follows the existing task-name shape (table in §4) |
| Ownership | `source_domain`, `owner`, `job_type` | `system_log_oper.source_domain` already carries the same dimension (`backend/internal/middleware/operation_log_middleware.go`) |
| Schedule | `schedule_type` (`cron/manual/event`), `schedule_expr`, `schedule_tz` | draft §4.2 |
| State | `enabled`, `paused`, `next_fire_at`, `prev_fire_at`, `concurrency_policy` | §6, §8 |
| Execution policy | `timeout_ms`, `retry_max`, `retry_backoff_seconds`, `misfire_policy` | §8, §9 |
| Presentation | `name_key` (i18n key), `description_key`, `source_route_key` (jump target) | no natural language stored ([DESIGN.md](../../DESIGN.md) §6) |
| Observability | `last_run_at`, `last_run_status`, `last_run_summary`, `last_duration_ms` | draft §4 + §10 |

### 5.1 Registration shape: handlers in code, schedule state in the database

- **Handlers are registered in code**: a `handler_key` must map to a registered implementation, and an
  unknown key is rejected before enabling. Neither the report nor the scheduler surface offers
  "write a piece of logic into a database row" (consistent with §2 non-goals). The registration shape
  copies `maintenance.Registry.Register` (idempotent, empty name rejected, re-registration does not
  accumulate — `backend/pkg/maintenance/maintenance.go:92-114`).
- **Schedule state lives in the database**: cron expression, enable/pause, timeout and retry policy are
  adjustable at runtime, so they need a row plus a cache convergence strategy (§12).
- During migration a task may exist both in the maintenance registry and in the scheduler inventory, but
  the double-run window must be labelled explicitly and the implementation PR must state when the old
  registration is removed (OD-S-4).

### 5.2 Execution states

The draft state set is kept and aligned with the existing maintenance outcome vocabulary
(`maintenance.go:40-52`: `succeeded / failed / skipped / overlap / unknown`):

| `schedule_job` view state | Meaning |
| :--- | :--- |
| `idle` | never executed and not paused |
| `running` | holds the lease and is executing (visible across instances) |
| `success` | last run succeeded |
| `failed` | last run failed (including retries exhausted) |
| `paused` | paused by a human, excluded from ticking |

`schedule_execution.status` additionally has `pending / running / succeeded / failed / cancelled / skipped / overlap`,
where `skipped` (interval not elapsed) and `overlap` (previous run still in flight) **reuse the existing
maintenance outcome names verbatim** — no second vocabulary.

### 5.3 Schedule types

Phase one supports `cron / manual / event` (draft §4.2); `one-off` from the English draft is expressed as a
`manual` job plus a `run-once` call rather than a fourth type:

- `cron`: driven by the expression grammar in §6.
- `manual`: no automatic trigger, only the `run-once` action (§11).
- `event`: business code calls `Trigger(jobKey, idempotencyKey)` explicitly; phase one provides **no**
  generic event bus, event sources must be explicit call sites in code (OD-S-1).

## 6. Cron Parsing and Timezone

### 6.1 Committed grammar (phase one)

The repository currently has **no** cron dependency: the require list in `backend/go.mod` contains neither
`robfig/cron` nor any scheduling library (verified: `grep -n "cron" backend/go.mod` returns nothing).
Phase one therefore commits to a **five-field minimal grammar** instead of pulling in an unapproved
dependency:

| Field | Range | Supported syntax |
| :--- | :--- | :--- |
| minute | 0-59 | `*`, `*/n`, `a`, `a-b`, `a,b,c`, `a-b/n` |
| hour | 0-23 | same |
| day-of-month | 1-31 | same |
| month | 1-12 | same |
| day-of-week | 0-6 (0 = Sunday) | same |

Explicitly **not** supported: a seconds field, `@every`, macros such as `@daily`, `L`/`W`/`#`, a year
field, month name aliases. An illegal expression is rejected during validation with
`schedule.expr.invalid`; no "best-effort parsing".

Whether to adopt a mature cron library instead is open decision OD-S-2 (a new dependency needs approval
under the AGENTS rule "do not introduce dependencies that were not requested").

### 6.2 Timezone

Timezone must be modelled explicitly because the repository's current state is **not uniform**:

| Fact | Source |
| :--- | :--- |
| No `time.LoadLocation` call exists anywhere in the backend Go code (i.e. IANA zone resolution does not exist today) | whole-tree check: `grep -rn "LoadLocation" backend/ --include=*.go` returns nothing |
| DSN normalization sets the driver's interpretation zone to `time.Local` (the process environment zone) when `loc` is absent | `backend/pkg/database/gorm.go:36-41` |
| CI DSNs pin `loc=UTC` while the MySQL service containers run `TZ=Asia/Shanghai` | `.github/workflows/quality.yml:411`, `:393` |
| The frontend renders timestamps in the browser's local zone only, cannot render an arbitrary named zone, and parses offset-less strings as local time | `frontend/src/core/format/dateTime.ts:52-70`, `:83-99` |

Four hard constraints follow:

1. **Every job carries an explicit `schedule_tz`** (IANA name such as `Asia/Shanghai`), defaulted from the
   platform setting `schedule.default_timezone`; implicit "whatever the process zone is" semantics are not
   allowed.
2. `next_fire_at` must be computed inside that named zone (DST and leap-second boundaries in §8.3), but
   **persisted as a UTC `DATETIME(3)` instant**; the API returns RFC3339 with an offset.
3. `time/tzdata` is part of the **Go standard library** (`import _ "time/tzdata"`), so importing it keeps
   named-zone resolution working on slim images without system tzdata and is not a new external
   dependency. Adding that import is an implementation task, listed for the implementation PR rather than
   promised by this design.
4. "Next 5 fire times" previews must be **computed server-side** and only rendered by the frontend;
   recomputing cron in the browser with `Intl`/local getters is forbidden (see the frontend fact above).

### 6.3 Validation and preview endpoint

`POST /platform/schedule/job/validate` parses without writing back and returns the normalized expression
plus the `nextFireAt` list. The value stored in `schedule_expr` must pass through the same validation code —
parsing and validation share one function so "the preview accepts it but storing it fails" cannot happen.

## 7. Triggering, Dispatch and Multi-Replica Single-Flight

### 7.1 Tick loop

Every instance runs one tick loop (shape copied from `maintenance.Run`: first pass immediately, then a
ticker — `backend/pkg/maintenance/maintenance.go:246-267`) at a fixed 60s period
(`schedule.tick_interval_seconds`). **No leader election**: all instances tick and the lease in §7.2
selects the single executor per job. Rationale: leader election adds a new lifecycle and new failure modes,
while the capability the repository already has (MySQL row locking) is sufficient at job granularity;
job granularity also means one crashed instance only affects the leases it holds.

Each pass works on `enabled = 1 AND paused = 0 AND next_fire_at <= now()` ordered by `next_fire_at ASC`
with a bounded batch (default 50, following the same "clamp server-side" idea as
`common.NormalizePagination`); unbounded scans are forbidden.

### 7.2 Where the lock comes from: a MySQL lease table — not GET_LOCK, not Redis

The "where does the lock come from" question must be answered with verifiable facts:

| Candidate | Repository facts | Conclusion |
| :--- | :--- | :--- |
| `GET_LOCK()` | **No** application code in this repository calls `GET_LOCK`. Its only user is the golang-migrate MySQL driver, which serializes migrations across pods (`docs/runbooks/MIGRATION_000031_PRODUCTION_DROP.md` §1.2, line 48; `docs/reviews/PANTHEON_BASE_GLOBAL_AUDIT_2026-10-31.md:1070` records "the pod that wins `GET_LOCK` (`mysql.go:298`, 10s wait) runs them") | not used as a runtime job lock: its holder is the migration path, its semantics are connection-scoped and released with the connection, and long jobs holding a wait-based lock couple scheduling to migration behaviour |
| In-process `sync.Mutex` | exists and only holds inside one process (`maintenance.go:70-75`, `:171-198`) | kept as the second-layer in-instance overlap guard; cannot serve as a cross-instance guarantee |
| Redis | `database.RDB` may be `nil`: outside production without `PANTHEON_REDIS_REQUIRED=true`, a failed connection degrades to nil (`backend/pkg/database/redis.go:27-54`) | not the lease store (scheduling would silently stop in dev/test); usable for cross-instance broadcast (§12.2) |
| **MySQL row lock + lease row** | `SELECT ... FOR UPDATE` is already used in production code paths here (`backend/modules/system/iam/role/role_service.go:560`, `backend/modules/system/org/dept/dept_service.go:170`), and ready-made fixtures prove the lock semantics (`backend/pkg/testmysql/locking.go:18-32` probes with `FOR UPDATE NOWAIT`; `:39` onwards `AssertSerialisedWriters` proves two writers of one parent row cannot interleave) | **adopted**: `schedule_lease` has one row per job; acquire/renew/release are conditional writes on that row |

Acquisition protocol (`schedule_lease`, columns in DB design §2.3):

1. `BEGIN` → `SELECT ... FOR UPDATE NOWAIT` locks the lease row for the job; if another instance holds it,
   give up this pass immediately (the same trade-off is already documented in the fixture:
   `backend/pkg/testmysql/locking.go:13-17` explains `FOR UPDATE NOWAIT` is used so a missing lock fails
   fast instead of waiting for a lock timeout).
2. Check `lease_expire_at < now() OR holder = ''`; if it fails, record `skipped` — **not** `failed`.
3. Otherwise write `holder = <instanceID>`, `lease_expire_at = now() + lease_seconds`,
   `lease_version = lease_version + 1` and `COMMIT`. **`lease_version` is the fencing token**: the final
   write-back only takes effect with the same version, so a stale holder that lost the lease cannot
   overwrite the winner's result (§7.3).
4. Renew every `lease_seconds/3` while running; a failed renewal means lost ownership and cancels the job
   context immediately.
5. Release on completion (`holder = ''`) and write the `schedule_execution` row.

`instanceID` reuses the existing instance identity shape `pid-unixnano`
(`backend/pkg/database/casbin_watcher.go:83`).

### 7.3 Crashes and lease expiry

- After a crash the lease expires at `lease_expire_at` and another instance can acquire it on the next
  pass; the orphaned `running` execution row is flipped to `failed` by startup reconciliation with the
  summary `schedule.execution.reconciled_orphan`. Reconciliation must decide by `lease_version` and must
  not blanket-clear `running`.
- Graceful shutdown has only a 15-second default budget (`PANTHEON_SHUTDOWN_TIMEOUT_SECONDS`,
  `backend/cmd/server/main.go:215-226`). Therefore handlers **must** consume the passed `context.Context`
  and be cancellable; the loop's context comes from `signal.NotifyContext` (same file `:192-195`).
  A job still running at shutdown is recorded per §9.2 (`cancelled`/`failed`), and the next instance decides
  whether to catch up according to `misfire_policy`.

### 7.4 Concurrency policy

`concurrency_policy` is fixed to `forbid` in phase one (globally one executor per job). `replace` and
`allow` are not offered — they need cancellation policy and side-effect governance, which are §2
non-goals. `run-once` and a cron trigger share the same lease, so a manual trigger colliding with an
automatic one results in `overlap` (reusing the outcome definition at `maintenance.go:47-49`), not queuing.

## 8. Missed-Run (Misfire) Policy

Clock facts first: the tick period is 60s, so any "miss" is at most `tick + acquisition latency`; real
long misses come from a full stack outage or an unavailable database.

| `misfire_policy` | Semantics | Phase one |
| :--- | :--- | :--- |
| `skip` | advance to the latest planned point only; all missed points are dropped and one `skipped` row records how many were missed | **default** |
| `coalesce` | many missed points produce exactly one catch-up run, then advance to the next future point | allowed |
| `fire_all` | run every missed point | **forbidden** (catching up cleanup/export jobs amplifies load and adds an unrequested capability) |

Advance rule: `next_fire_at` is the only cursor. The previous **actual** start time must not be the basis,
otherwise one long run shifts all later planned points — a common implementation bug in cron semantics,
recorded here to prevent it.

Timezone boundaries (§8.3 of the cron grammar above): a spring-forward time that does not exist is skipped
to the next valid point; a repeated fall-back local time fires once (because `next_fire_at` is stored as a
UTC instant per §6.2, the duplicate local time maps to two different UTC instants and must be
de-duplicated while advancing).

Clock skew or rollback beyond one tick: no automatic catch-up, record `skipped` and surface "planned point
missed" in the UI.

## 9. Retry, Backoff and Idempotency

### 9.1 Retry

- `retry_max` defaults to 0 (no retry), capped at 5; `retry_backoff_seconds` defaults to 60 and doubles
  exponentially up to `schedule.retry_backoff_max_seconds` (default 1800).
- Backoff is implemented by updating `next_fire_at` (or the `fire_at` of one standalone execution), never by
  `time.Sleep` occupying a goroutine — consistent with the 15-second shutdown budget (§7.3).
- Non-retryable error classes (record `failed` immediately, do not consume the retry budget):
  `schedule.expr.invalid`, `schedule.handler_not_found`, `schedule.param.invalid`,
  `schedule.tenant_context_missing`. Reason: these are configuration errors; retrying only hides them.
- Retryable: any other handler error, context deadline, and deadlock-class database errors.
- After retries are exhausted a failure summary is produced for the alert center (§14); the summary carries
  only `error_key` plus counts, never a stack trace or raw parameters.

### 9.2 Cancellation and outcomes

`cancelled` is produced only by an explicit cancel action (requires `platform:schedule:cancel`, see API
contract §5) or by shutdown-induced context cancellation; every other failure is `failed`.
**No outcome may be silent**: every acquisition attempt writes a `schedule_execution` row, including
`skipped` and `overlap`, otherwise the history view and the "last run summary" in §10 develop
unexplained holes.

### 9.3 Idempotency keys

- Automatic trigger: `fire_instance = job_key + ':' + UTC(schedule_fire_at)` — the planned point, not the
  actual start time.
- Manual trigger: `fire_instance = job_key + ':manual:' + <client idempotencyKey>`; when absent the server
  generates `sha1(job_key|operatorID|unixMilli)` and echoes it back.
- The unique key `(job_key, fire_instance)` on `schedule_execution` is the final dedup. A duplicate delivery
  that hits it returns the existing execution row instead of re-running the handler.
- That unique key **excludes `deleted_at`**, following the repository's unique-index convention (soft delete
  releases the key by renaming — see SCHEDULER_CENTER_DB_DESIGN.en.md（planned/unavailable：`./SCHEDULER_CENTER_DB_DESIGN.en.md`；无当前交付承诺） §4
  and the existing precedent `backend/pkg/database/migrations/000001_init_schema.up.sql:117-120`).
- Handler obligation: calling with the same `fire_instance` must not perform a second business write.
  Cleanup-style tasks are inherently idempotent under repetition
  (`maintenance.go:27-30` already requires `Run` to be safely callable repeatedly).

## 10. Execution History and Retention

| Item | Rule | Basis |
| :--- | :--- | :--- |
| One row per execution | `schedule_execution` (start/end/duration/outcome/error key/retry index/holding instance/`fire_instance`) | §7, §9 |
| Denormalized summary on the job | `last_run_at / last_run_status / last_run_summary / last_duration_ms` | draft §4 |
| Retention | setting key `schedule.execution_retention_days` (default 30), batched `LIMIT` deletes by day | precedent: `backend/modules/system/audit/audit_service.go:317-341` (retention days from a setting, purge as a registered task) |
| Purge implementation | registered as a scheduler-center job, while **keeping** the maintenance registry channel open (OD-S-4) | `backend/pkg/maintenance/maintenance.go:92-114` |
| Purge must stay out of request paths | list/detail handlers are read-only | lesson recorded in the package comment `maintenance.go:1-12` |
| When the purge switch is off | with `PANTHEON_MAINTENANCE_ENABLED=false` nothing purges; both the docs and the UI must state the operator action | `backend/cmd/server/main.go:204-213` |
| Pagination | history list uses `common.NormalizePagination(page, pageSize, 100)`; `pageSize` clamped server-side to 100 | `backend/pkg/common/handler.go:39` |

## 11. Enable, Disable, Pause and Manual Run

The draft action set (`enable / disable / run-once / view-history`, plus `pause / resume / cancel / jump to
source context` from the English draft) is kept with tightened semantics:

| Action | Semantics | Relation to state |
| :--- | :--- | :--- |
| `enable` | resume automatic triggering | `enabled = 1`; if `paused = 1` the job still does not fire and needs an explicit resume |
| `disable` | stop automatic triggering | `enabled = 0`; **does not cancel a run in flight**, only stops subsequent acquisition |
| `pause` | silence while keeping the job visible with history | `paused = 1`; `run-once` remains available (operators need it) |
| `resume` | clear the pause | `paused = 0` |
| `run-once` | enqueue one immediate execution (draft's "run now") | high-sensitivity: separate permission key + second factor + audit (draft §5 constraint) |
| `view-history` | read execution history | read-only |
| `view-source` | jump to the source context | read-only navigation; the target route authorizes itself, the scheduler center never grants on its behalf |
| `cancel` | cancel an in-flight execution | separate permission; cancels via context, outcome `cancelled` (§9.2) |
| `update-schedule` | change cron/timezone/policy | must recompute `next_fire_at`; the old cursor may not be kept |

Constraints:

1. `disable` and `pause` are two different facts and are not merged into one boolean: the first says "this
   job is not part of this environment's plan", the second says "temporarily silenced". The draft lists both
   `enabled` and `paused` in §4.1; this design keeps that distinction.
2. `run-once` must state explicitly whether the current parameter snapshot is reused; parameters are frozen
   into the execution row at trigger time so later edits never rewrite that record.
3. State changes (enable/disable/pause/resume) and `run-once` are writes and must pass Casbin; the two
   high-sensitivity ones (`run-once`, `cancel`) also pass `SecureActionMiddleware`
   (`backend/internal/middleware/secure_action_middleware.go`; precedents
   `backend/modules/auth/module.go:96-108` and `backend/modules/system/system_modules.go:188-191`).

## 12. Permissions, Tenancy and Caching

### 12.1 Three permission layers

- **Route layer**: `TokenAuth -> PlatformScopeTenantContext -> Casbin`, identical in shape to the platform
  dashboard (`backend/modules/platform/routes.go:34-37`; contract requirement
  [PLATFORM_CONTRACT.en.md](../contracts/PLATFORM_CONTRACT.md) §3.3).
- **Action layer**: `platform:schedule:{list,view,update,run,cancel,clear}` (full register in API contract §5).
  Read/navigation gating uses `:list` and no action menu node may bind `:list` — that case is the only
  blocking finding of `scripts/harness/check-permission-contract.mjs:125-136`.
- **Job visibility layer**: in phase one the job inventory is a **platform-level operational resource** and is
  not tenant row-isolated: `schedule_job` carries no `tenant_id`, following the existing platform-global
  table precedent (rationale documented at
  `backend/pkg/database/migrations/000030_tenant_plan_definitions.up.sql:11-17`).
  The scope-matrix registration obligation is in
  [TENANT_RESOURCE_SCOPE_MATRIX.md](./TENANT_RESOURCE_SCOPE_MATRIX.md) §1/§3 (OD-S-5).
- **Exception**: data reads inside a handler must still carry their own domain's tenant semantics. The
  scheduler center neither performs row-level data scope nor replaces it
  (`business/*` does not participate in tenant isolation in V1 —
  `backend/pkg/contracts/route_groups.go:21-26`).

### 12.2 Configuration and cache convergence

- Platform setting keys (stored in `system_setting`, consumed the same way as the existing `audit.*`
  precedent): `schedule.default_timezone`, `schedule.tick_interval_seconds`, `schedule.lease_seconds`,
  `schedule.execution_retention_days`, `schedule.retry_backoff_max_seconds`, `schedule.sync_row_cap`.
- Convergence of job definitions across instances:
  - prefer reusing the existing Redis pub/sub broadcast shape (`backend/pkg/database/casbin_watcher.go:17-33`,
    `:138-140`, env gate `PANTHEON_CASBIN_WATCHER`) with a sibling channel for "definition changed" signals;
  - the broadcast is **best effort**: when the env gate is off or Redis is missing, convergence must fall back
    to "re-read due jobs every tick". Therefore the broadcast must never be documented as a consistency
    guarantee (same discipline as the report center's caching section).

## 13. UI Skeleton

Phase-one minimal UI (draft §6 kept and completed):

- Scheduler center list: `SearchToolbar` contract ([DESIGN.md](../../DESIGN.md) §7.2), enabled-state filter,
  execution-state filter, schedule-type filter, source-domain filter, row-level actions, last execution
  result, next fire time (server-computed)
- Job detail: definition snapshot, execution policy, last N executions, lease holder (operational visibility
  only; no internal addresses)
- History page/drawer: status, duration, error key, retry index, `fire_instance`
- Action area: enable/disable/pause/resume/run-once/cancel; the two high-sensitivity ones require the second
  factor per §11
- Jump-to-source-context entry (read-only navigation; the target page authorizes itself)
- Expression validation/preview: enter cron + timezone → show the next 5 fire times (returned by the server)
- All four states present: no permission / empty / loading / submitting
  ([EMPTY_LOADING_ERROR_STATES.md](./EMPTY_LOADING_ERROR_STATES.md))

Still not required (draft §6): dependency topology graphs, live log streams, complex batch orchestration.
One more explicit non-goal: **no polling of execution log content on the list page** — phase one stores
summaries and error keys only, never stdout.

## 14. Relationship with the Report Center, Alert Center and Dashboard

- The report center is a **consumer** of the scheduler center: it registers the `report.generate` handler and
  performs registration/enable/disable only through the scheduler contract, without building its own cron,
  timers or locks ([REPORT_CENTER_DESIGN.en.md](./REPORT_CENTER_DESIGN.en.md) §11).
- The scheduler center governs scheduled execution, the alert center governs operational signals. It is
  **not** an alerting system; it emits failure summaries for the alert center to consume
  ([ALERT_MONITORING_CENTER_DESIGN.md](./ALERT_MONITORING_CENTER_DESIGN.md)) and the two centers stay
  conceptually separate.
- The dashboard consumes scheduling health summaries (success rate, consecutive failures, longest run) as a
  read-only aggregate, reusing the `platform:dashboard:view` grant shape
  (`backend/modules/system/seed.go:210`).
- Metrics reuse the existing shape: counter by task and outcome plus duration histogram
  (`backend/pkg/metrics/prometheus.go:98-112`); new scheduler metrics must not change the existing label
  cardinality conventions.

## 15. Minimum Pilot Requirements

The three draft pilot categories are kept, each with an integration point that already exists today so the
pilot cannot degenerate into a demo. At least one real schedulable source domain must integrate before the
center is treated as complete:

1. **Audit / login / security-event retention jobs**: migrate the four existing maintenance tasks (table in §4)
   into the scheduler center to gain cross-instance exclusion, history and enable/pause. This is the most
   convincing pilot because it directly removes the gap stated in §4 conclusion 1.
2. **i18n asset sync / dynamic module sync**: onboard as `manual` + `event` jobs to exercise the non-cron path.
   Note that `dynamicmodule / generator` is an independent high-sensitivity capability, not a `system/config`
   sub-domain ([SYSTEM_CONFIG_CONTRACT.en.md](../contracts/SYSTEM_CONFIG_CONTRACT.md) §7.2), so its trigger
   action must keep the original permission and environment guards.
3. **Report generation**: `report.generate` (§14), exercising "scheduler triggers + another domain writes back
   the result".

A pilot must be a real job with real permissions and real traces, not sample data.

## 16. Definition of Done

Phase one is done when:

- there is a unified job model and registration shape (§5), with four tables landed and migrated
  (SCHEDULER_CENTER_DB_DESIGN.en.md（planned/unavailable：`./SCHEDULER_CENTER_DB_DESIGN.en.md`；无当前交付承诺）)
- the cron minimal subset, named-zone computation and server-side preview exist (§6) with tests
- multi-replica single-flight has a **provably implemented** mechanism (§7.2 protocol plus a test using the
  `testmysql` lock fixtures), not "it looks like it uses a lock"
- misfire policy, retry/backoff and idempotency keys exist (§8/§9) and every attempt is recorded (§9.2)
- there is an execution history view with retention purging (§10)
- enable/pause/manual-run actions exist with their permissions and audit (§11)
- there is a scheduler center entry (menu + routes + i18n + permission points registered together)
- at least one real job pilot is onboarded (§15 item 1)
- all P0 cases in SCHEDULER_CENTER_TEST_PLAN.en.md（planned/unavailable：`./SCHEDULER_CENTER_TEST_PLAN.md`；无当前交付承诺） are green or explicitly waived

"Enterprise-grade job orchestration capability" is not the completion bar.

## 17. Open decisions

| ID | Decision | Options | Impact | Decision owner |
| :--- | :--- | :--- | :--- | :--- |
| OD-S-1 | Shape of `event` job trigger sources | A: explicit `Trigger()` calls in code only; B: introduce a generic event/message bus | B turns the scheduler center into a message middleware; this design leans A | Maintainer |
| OD-S-2 | cron implementation | A: the five-field minimal subset of §6.1; B: adopt `robfig/cron` or similar; C: fixed intervals only | B needs approval under "do not introduce dependencies that were not requested"; C would make `schedule_type = cron` an empty promise | Maintainer |
| OD-S-3 | Lease store | A: MySQL lease table (this design); B: Redis `SET NX` + TTL; C: `GET_LOCK` | B silently stops scheduling where `RDB = nil` outside production (`backend/pkg/database/redis.go:27-54`); C competes with the migration path over the same mechanism | Maintainer |
| OD-S-4 | Whether the four existing maintenance tasks are retired | A: migrate and delete the old registrations; B: run both for one window, then delete; C: long-term coexistence | decides whether §15 item 1 can remove the §4 conclusion 1 gap immediately; B is preferred (leaves a rollback path) | Maintainer |
| OD-S-5 | Whether jobs need a tenant dimension | A: platform-level resource, no `tenant_id` (this design); B: optional `tenant_id` allowing tenant-defined jobs | B brings back the "arbitrary remote script execution" risk and conflicts with §2; it would also require writing back to the scope matrix | Maintainer |
| OD-S-6 | Default timezone value | A: `schedule.default_timezone = Asia/Shanghai`; B: force `UTC` | A matches operator intuition but depends on the tzdata precondition in §6.2; B is safest but makes "next fire" less readable for users | Maintainer |

## 18. Related documents

| Document | Chinese | English |
| :--- | :--- | :--- |
| Main design (this document) | [SCHEDULER_CENTER_DESIGN.md](./SCHEDULER_CENTER_DESIGN.md) | `SCHEDULER_CENTER_DESIGN.en.md` |
| Data model and migrations | [SCHEDULER_CENTER_DB_DESIGN.md](./SCHEDULER_CENTER_DB_DESIGN.md) | SCHEDULER_CENTER_DB_DESIGN.en.md（planned/unavailable：`./SCHEDULER_CENTER_DB_DESIGN.en.md`；无当前交付承诺） |
| API contract | SCHEDULER_CENTER_API_CONTRACT.md（planned/unavailable：`./SCHEDULER_CENTER_API_CONTRACT.md`；无当前交付承诺） | SCHEDULER_CENTER_API_CONTRACT.en.md（planned/unavailable：`./SCHEDULER_CENTER_API_CONTRACT.en.md`；无当前交付承诺） |
| Test plan | SCHEDULER_CENTER_TEST_PLAN.md（planned/unavailable：`./SCHEDULER_CENTER_TEST_PLAN.md`；无当前交付承诺） | SCHEDULER_CENTER_TEST_PLAN.en.md（planned/unavailable：`./SCHEDULER_CENTER_TEST_PLAN.en.md`；无当前交付承诺） |

Report companion: [REPORT_CENTER_DESIGN.en.md](./REPORT_CENTER_DESIGN.en.md).
Index: [README.en.md](./README.en.md).

## Change log

| Date | Change |
| :--- | :--- |
| 2026-11-04 | Added the audit of existing periodic-task facts, the cron subset and timezone hard constraints, multi-replica single-flight (MySQL lease plus fencing token), misfire policy, retry/backoff and idempotency keys, history retention, enable/pause/manual-run semantics, permission and cache convergence, open decisions and the package index |
| 2026-05-18 | Earlier English overview draft |
| 2026-10-31 | Initial draft |
