---
title: Approval Workflow Skeleton Design
doc_type: Design
layer: platform / business/*
status: Draft
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
  - docs/contracts/SYSTEM_IAM_CONTRACT.md
  - docs/contracts/TENANT_CONTRACT_V1.md
updated_at: 2026-11-04
---

# Approval Workflow Skeleton Design

Chinese version: [APPROVAL_WORKFLOW_DESIGN.md](./APPROVAL_WORKFLOW_DESIGN.md)

> Not implemented: this is a design draft with no corresponding implementation in the repository; it stays Draft until implemented.

Related contracts:
- `PLATFORM_CONTRACT.md`
- `SYSTEM_IAM_CONTRACT.md`
- `TENANT_CONTRACT_V1.md`

Related roadmap:
- `P2_SCALE_ROADMAP.md`
- `PANTHEON_BASE_ARCHITECTURE_OVERVIEW.md`

Companion implementation docs (same design package; read together before implementing):

| Document | Responsibility |
| :--- | :--- |
| Approval workflow database design（planned/unavailable：`./APPROVAL_WORKFLOW_DB_DESIGN.en.md`；无当前交付承诺） (Chinese: APPROVAL_WORKFLOW_DB_DESIGN.md（planned/unavailable：`./APPROVAL_WORKFLOW_DB_DESIGN.md`；无当前交付承诺）) | tables, columns, indexes, tenant columns, migration numbering after 000031, down shape |
| Approval workflow API contract（planned/unavailable：`./APPROVAL_WORKFLOW_API_CONTRACT.en.md`；无当前交付承诺） (Chinese: APPROVAL_WORKFLOW_API_CONTRACT.md（planned/unavailable：`./APPROVAL_WORKFLOW_API_CONTRACT.md`；无当前交付承诺）) | REST surface, DTOs, error codes, pagination, permission keys per route, `SecureAction` scope, i18n namespace |
| Approval workflow test plan（planned/unavailable：`./APPROVAL_WORKFLOW_TEST_PLAN.en.md`；无当前交付承诺） (Chinese: APPROVAL_WORKFLOW_TEST_PLAN.md（planned/unavailable：`./APPROVAL_WORKFLOW_TEST_PLAN.md`；无当前交付承诺）) | acceptance criteria to cases (including the concurrent double-decision race), existing gates each case hangs on, explicit runtime gaps |
| [Workflow engine selection](./WORKFLOW_ENGINE_SELECTION.en.md) (Chinese: [WORKFLOW_ENGINE_SELECTION.md](./WORKFLOW_ENGINE_SELECTION.md)) | this design keeps its conclusion: phase one builds a self-hosted lightweight engine, no BPM/Temporal |

This document defines the "approval workflow skeleton" for Pantheon's standard backoffice product surface.

The current goal is not to build a full BPM platform. The goal is to converge on:

- one unified approval center entry
- one shared approval-item model
- one minimal action contract
- one consistent audit story

for views such as pending approvals, processed approvals, and my submissions.

## 1. Design Goals

The skeleton should ensure that:

1. the platform has one approval entry instead of each module inventing its own todo center
2. different business domains can plug into one unified approval view while preserving their own domain semantics
3. approval actions, state transitions, and audit traces are expressed consistently

## 2. Non-Goals

Not in scope now:

- visual workflow designers
- general-purpose rule orchestration engines
- arbitrary node-level script execution
- complex multi-party organizational approval choreography

## 3. Ownership Boundary

### 3.1 What `platform` Owns

- the approval center entry
- the unified approval-item list model
- shared status views
- the aggregate relationship with the task center

### 3.2 What `business/*` Owns

- business meaning of the approval item
- step-transition rules
- detail pages and handling actions
- domain-specific audit supplements

In one sentence:

`platform` owns the approval product surface, while business domains own the actual approval handling logic.

Two boundary constraints that come straight from existing code, not invented here:

- `modules/business/*` must not import `Service` / `Repository` of `modules/system/*` (`docs/DESIGN.md` §2.3). A business domain joins the approval center only through the narrow registration contract in section 10 plus its own `biz_*` tables; it must not reach into the platform approval service.
- Route-group middleware chains are contractual: business routes use `contracts.ProtectedGroup` / `contracts.DataScopedGroup` (`backend/pkg/contracts/route_groups.go`, chain `TokenAuth -> Casbin` (-> `DataScope`)); system and platform consoles use `contracts.SystemProtectedGroup` (`TokenAuth -> PlatformScopeTenantContext -> Casbin`, optional Capability / DataScope / RefreshSync). The approval center is a platform console surface, so it must use the `SystemProtectedGroup` shape, never a hand-rolled bare `r.Group(...)`.

## 4. Approval Item Model

Recommended minimum fields:

- `id`
- `workflowType`
- `sourceDomain`
- `bizType`
- `bizId`
- `title`
- `status`
- `currentStep`
- `applicantId`
- `currentApproverId`
- `submittedAt`
- `processedAt`
- `jumpTarget`

### 4.1 States

Minimum states:

- `draft`
- `submitted`
- `pending`
- `approved`
- `rejected`
- `cancelled`

These are the outward-facing values (dictionary, i18n, filters all use them). **Section 11 is the authoritative transition model**; no other document may restate its own transition graph.

### 4.2 Default Views

Minimum views:

- `pending-for-me`
- `processed-by-me`
- `submitted-by-me`

A view is a query predicate, not a state: `pending-for-me` = `status in (submitted, pending)` and the current candidate set resolves to me (including delegates from section 13); `processed-by-me` = I executed `approve/reject/transfer` on the item per `system_approval_action_log`; `submitted-by-me` = `applicant_id = me`.

### 4.3 Additional model fields required for implementation

The list in section 4 is the product view. Implementation also needs the columns below; the full definition lives in the database design document.

| Field | Purpose | Grounding |
| :--- | :--- | :--- |
| `version` | compare-and-set base for decide / transfer / withdraw (section 15) | new in this design; existing column-level precedent `backend/modules/system/refresh_sync.go:19` |
| `current_step_no` + `step_total` | list shows "step 2 of 3" without a join | queryable form of `currentStep` in section 4 |
| `due_at` | basis for the timeout and escalation scan (section 16) | new in this design |
| `escalated_count` | guard against escalation storms (section 16.3) | new in this design |
| `terminal_reason` | terminal attribution: `approved/rejected/cancelled/expired` | separated from `status` for reporting |
| `tenant_id` | tenant ownership (section 21) | `TENANT_CONTRACT_V1.md` §2.3 |
| `deleted_at` | soft delete (GORM default) | `docs/designs/DATABASE.md` §1 |

## 5. Action Contract

First-phase unified action semantics:

- `approve`
- `reject`
- `transfer`
- `cancel`

This design adds three more (sections 13, 15 and 17 all need them, without changing the four outward semantics above):

- `withdraw` (applicant withdraws, section 17)
- `escalate` (system action, never human, section 16)
- `delegate` (handled by a proxy; the persisted record is still `approve/reject` plus a delegation marker, section 13)

Rules:

- action availability is decided by the source domain
- the platform layer does not guarantee that every approval item supports every action
- the frontend must not show fake buttons without backend semantics

### 5.1 Action-guard matrix

Every action has four elements: who may trigger it, the required prior state, the guards, and the side effects. A missing element means the action is undefined and must not get a route.

| Action | Trigger | Prior state | Guards | Side effects |
| :--- | :--- | :--- | :--- | :--- |
| `submit` | applicant | `draft` | source-domain callback allows submission (section 10.3); `biz_id` exists and is not deleted | `status: draft -> pending`; writes `submitted_at`, `due_at`; creates step 1 |
| `approve` | current assignee or their delegate | `pending`, current step `running` | not the applicant (section 14); `version` matches; note required | logs the action; last step -> `approved`, otherwise advances one step; refreshes `current_approver_id` |
| `reject` | same as `approve` | same | same; note mandatory (no reasonless rejection) | logs the action; `status -> rejected`, item terminates |
| `transfer` | current assignee or their delegate | `pending` | target holds the step candidate role and is not the applicant; no self-transfer | reassigns `assignee_id`; logs `transferred_from/to`; `version + 1` |
| `cancel` | applicant | `draft`, or `pending` with no decision yet | none | `status -> cancelled`; terminal |
| `withdraw` | applicant | `pending`, current step `running` (nobody decided) | no prior decision record, or the type explicitly allows "withdraw after decision" | `status -> cancelled`, `terminal_reason = withdrawn`; re-submission allowed (section 17.3) |
| `escalate` | system maintenance task (section 16) | `pending` and `due_at` passed | `escalated_count < limit` | logs a system action; reassigns to the next candidate up the chain; `status` unchanged |

Mandatory notes for `approve` / `reject` match an existing convention in this repository: acknowledging a security event requires a note and returns `auth.security_event.acknowledge.note_required` otherwise (`backend/modules/auth/security/security_service.go:366-389`).

## 6. Relationship with the Task Center

Approval workflows and the task center should not become two parallel, unrelated models.

Recommended relationship:

- the approval item is the business object
- approval todos are one task type inside the task center
- the task center shows approval summaries
- the approval center provides approval-specific filters and handling entry points

Implementation rule (so there is never two sources of truth): the task center does not copy approval state. When [Task Center Design](./TASK_CENTER_DESIGN.en.md) (Chinese: [TASK_CENTER_DESIGN.md](./TASK_CENTER_DESIGN.md)) lands, approval task rows are **derived** from the approval tables (`derived` scope per `docs/designs/TENANT_RESOURCE_SCOPE_MATRIX.md` §1): a read-only projection plus a `jumpTarget` back to the approval center. The approval center is the only write path. Nobody may "click an approval away" inside the task center.

## 7. UI Skeleton

First-phase minimum UI:

- approval-center list page
- three tabs: pending for me, processed by me, submitted by me
- status filters
- row-level jump to source-domain detail or handling page

Not required yet:

- drag-and-drop flow diagrams
- node-level visual workflow composition
- approval analytics dashboards

Required page states (platform shell contract, `docs/designs/EMPTY_LOADING_ERROR_STATES.md`): loading / empty / error / forbidden / submitting. Cross-page selection follows `PLATFORM_CONTRACT.md` §7.3.1: paging keeps the selection set, changing the query context clears it.

Batch actions (batch approve / batch reject / batch transfer) are a **separate permission key + separate route + mandatory step-up verification**, and must return partial results (`successCount / failedCount / failures[]`). That is the established batch-governance shape in `SYSTEM_IAM_CONTRACT.md` §9.0 and `SYSTEM_CONFIG_CONTRACT.md` §8.0; the approval center must not invent its own. Batch decision semantics are in section 15.4.

## 8. Audit and Permissions

At minimum, the system needs:

- approval-center access permission
- approval-item view permission
- approval-action permissions
- audit records for approval actions

Audit must retain:

- applicant
- processor
- action
- time
- result
- summary note

The authoritative shapes are in section 19 (audit trail) and section 20 (permission keys, menu, i18n).

## 9. Minimum Pilot Requirements

The first phase must connect at least one real business scenario so the approval center does not remain a shell.

Recommended pilots:

- approval for enabling or disabling business resources
- high-sensitivity configuration-change approval
- asset-entry review

Pilot choice is constrained by the tenant contract (section 21.2): `modules/business/*` does not participate in tenant isolation in V1 (`TENANT_CONTRACT_V1.md` §2.6). Therefore a business pilot has no tenant semantics under `multi` mode. The phase-one reference pilot must be the `high-sensitivity configuration-change approval` (`system/config` domain); business pilots are only promised on compat single-tenant deployments.

## 10. Integration Contract (source-domain registration)

The approval center does not guess how a business domain wants to be reviewed. A source domain must explicitly register one approval type; that registration is the only contract surface between platform and business.

### 10.1 Registration elements

| Element | Meaning | Consequence if missing |
| :--- | :--- | :--- |
| `workflow_type` | stable code, `<domain>_<object>_<action>` style, e.g. `config_setting_change_approval` | cannot route to a handler |
| `source_domain` | one of `platform / system/* / business/*`, written directly to the audit search column `source_domain` (`docs/designs/DATABASE.md` §3.2) | audit cannot filter by source domain |
| `biz_type` + `biz_id` semantics | the domain's own object type and key; the platform never interprets them | jump and callback cannot locate the record |
| `step_definitions` | ordered step sequence (first step fixed), each step carrying one candidate set (section 12.1) | submission fails outright |
| `actions` | subset of the action vocabulary in section 5 supported by this type | frontend must hide the button |
| `jump_target` | detail route template, e.g. `/system/setting/approval/:id` or a business detail path | list row is not clickable |
| `permission_prefix` | action permission namespace for this type (section 20.1) | Casbin cannot converge |

### 10.2 Where registration lives

Phase one has no process designer, so registration is declared in code and seeded into the `system_approval_workflow_type` table (database design section 5). This mirrors how the repository already works: menu and permission points are declared as `menuSeed` structs in `backend/modules/system/seed.go:203-250`; the frontend declares routes, menus, permissions and i18n namespaces with `defineModule(...)` (`frontend/src/modules/system/audit/index.ts`) and is aggregated in `frontend/src/core/router/modules.ts`.

Forbidden: keeping "which types are connected" only in documents or verbal agreement. The `workflow_type` table is the source of truth, otherwise the permission-workbench gap detection in section 20.3 cannot work.

### 10.3 Callback boundary

The platform notifies the source domain on state transitions; the source domain performs its business side effect (updating business state, sending notifications). Three rules:

1. The callback is an **in-process narrow interface** (a Go handler registered per `workflow_type`), not an HTTP self-call. Business domains must not import a platform approval service and then re-wrap it.
2. Callback and transition run in the **same transaction**: a failing `beforeSubmit` check must keep the item out of `pending`. Cross-process asynchronous callbacks would create "approved but not effective" states, which this design explicitly rejects.
3. Callback failure means transition failure, returning a domain error key. Decide actions do not swallow this as a 500: see `approval.decision.side_effect_failed` in section 15.2, and the item stays `pending`.

## 11. State Machine

### 11.1 State list

| State | Meaning | Terminal | Counted as todo |
| :--- | :--- | :--- | :--- |
| `draft` | draft, editable by the applicant, no approver slot yet | no | no |
| `submitted` | submitted, resolving the first assignee (transient) | no | yes |
| `pending` | under review, a current assignee exists | no | yes |
| `approved` | approved, last step completed | yes | no |
| `rejected` | rejected, any rejection terminates the item | yes | no |
| `cancelled` | withdrawn or cancelled; `terminal_reason` distinguishes them | yes | no |

`submitted` is transient: on the happy path the same transaction moves it to `pending`. It exists only to give "assignee resolution failed after submit" an explainable landing point; section 16 converges it to `cancelled`, and it must not occupy the todo list indefinitely.

### 11.2 Transition table

| # | From | Action | To | Trigger | Guards (all must hold) |
| :--- | :--- | :--- | :--- | :--- | :--- |
| T1 | (none) | `create` | `draft` | applicant / source domain | `workflow_type` registered and `enabled`; `biz_id` exists |
| T2 | `draft` | `update` | `draft` | applicant | nobody decided yet; applicant only |
| T3 | `draft` | `submit` | `pending` (transiently via `submitted`) | applicant | section 5.1; source-domain `beforeSubmit` passes; step definitions non-empty |
| T4 | `pending` | `approve` (not last step) | `pending` | current assignee or delegate | not the applicant; `version` matches; current step `running`; note non-empty |
| T5 | `pending` | `approve` (last step) | `approved` | same as T4 | same as T4 plus current step is the last one |
| T6 | `pending` | `reject` | `rejected` | same as T4 | same as T4 plus note non-empty |
| T7 | `pending` | `transfer` | `pending` | current assignee or delegate | not the applicant; target valid and not the applicant; `version` matches |
| T8 | `pending` | `cancel` | `cancelled` | applicant | no `approve/reject` history |
| T9 | `pending` | `withdraw` | `cancelled` | applicant | same as T8, unless the type explicitly allows `withdraw_after_decision` |
| T10 | `pending` | `escalate` | `pending` | system maintenance task | `due_at` passed; `escalated_count < limit` |
| T11 | `submitted` | `escalate` (timeout) | `cancelled` | system maintenance task | stuck beyond threshold; `terminal_reason = submit_stuck` |
| T12 | `approved/rejected/cancelled` | (no outgoing edge) | - | - | terminal states are irreversible; correction goes through re-submission (section 17.4) |

Rules:

- This table is the **only** authority. Any combination not listed must be rejected by the service layer with `approval.transition_invalid` (error codes in section 20.2); "the frontend just hides it" is not acceptable.
- Terminal states never reverse. There is no `approved -> pending` reopen; reopening equals T12 plus a new item (section 17.4), keeping the old one as an audit fact.

## 12. Assignee Resolution and Multi-signer Semantics

### 12.1 Candidate sets

A step's candidate set comes from exactly one of three sources (listed in resolution priority); phase one supports nothing else:

| Source | Meaning | Existing dependency |
| :--- | :--- | :--- |
| `user` | one specific user | `system_user` |
| `role` | a role key; candidates are the enabled users holding it | `system_role` / `system_user_role` (`system/iam`) |
| `dept_leader_chain` | walk the department ancestor chain to its leaders, level by level | `system_dept.leader_user_id`, `ancestors` (`docs/designs/SYSTEM_ORG_DESIGN.md`; `system_dept` is `tenant-owned` per `docs/designs/TENANT_RESOURCE_SCOPE_MATRIX.md` §2) |

On resolution failure (empty candidate set): fall back per section 16 - next level up the leader chain, then the approval administrator role of the type. If that is also empty, T3 fails outright with `approval.approver_unresolved` and the item stays `draft`. Never create a `pending` item without an approver: that produces a todo nobody can ever clear.

### 12.2 Any-one approval (supported in phase one)

- A step with several candidates where any one decision completes the step is `any`.
- Deciding is a race: the first `approve/reject` wins, the others lose the `version` compare-and-set and receive `approval.decision_stale` (section 15.2).
- After a decision, other people's todos disappear through the view predicate (section 4.2); no compensating notification write is needed.

### 12.3 All-signer approval (out of phase one; semantics pinned now)

`all` (every candidate must decide) and quorum-based passing belong to the "complex multi-party choreography" excluded by section 2. To stop future implementations diverging, the rules are fixed here:

- Introducing it requires an `approval_step_vote` record with **one idempotency key per vote** (`step_id + voter_id` unique); re-voting uses revive-on-conflict UPSERT semantics, same shape as the membership revival in `TENANT_CONTRACT_V1.md` §2.2.
- Step completion must be decided by a single-row CAS on `version`, never by read-then-write, otherwise concurrent votes can both satisfy the threshold.

## 13. Delegation and Proxy

### 13.1 Model

Delegation means "hand the handling right of a class of todos to someone else inside a time window"; it does not change the item's `assignee`.

| Dimension | Rule |
| :--- | :--- |
| Scope | per `workflow_type`; "delegate everything forever" is not supported (unauditable) |
| Time window | `starts_at` / `ends_at` mandatory, `ends_at` after `starts_at`; window cap is a configuration value (default 90 days) |
| Depth | **no chained delegation**: while A->B is active, B may not delegate to C. Guard at write time: refuse when the delegatee is itself a delegatee, returning `approval.delegate_chain_not_allowed` |
| Effect | only affects todo visibility (section 4.2 `pending-for-me` includes valid delegations) and the actor check |
| Revocation | the delegator or a governance role may set `revoked_at` at any time; historical actions are never rolled back |

### 13.2 Identity and audit

- When a delegate runs `approve/reject`, `operator_id` is the delegate and `on_behalf_of_id` is the delegator; both go into `system_approval_action_log` (section 19.2). "Who clicked" and "whose authority it was" must be two columns.
- Authorization uses the **delegate's own** permission keys and Casbin result: delegation never widens permissions. If the delegate lacks `platform:approval:act` or has no policy for the route in their own roles, the answer is `403 permission.denied` (`backend/internal/middleware/casbin_middleware.go` evaluates `path + method` and does not know about delegation).
- The self-approval ban in section 14 applies through delegation as well: a delegate must not be the original applicant, and a delegator must not delegate to themselves.

### 13.3 Read rule for todo ownership

The `pending-for-me` query must be equivalent to:

```
current step candidates include me
  OR current step candidates include a delegator for whom I hold a valid, non-revoked delegation
```

Window validity is compared in the database (`starts_at <= NOW() AND ends_at > NOW() AND revoked_at IS NULL`), not with an application-side cached clock, so multi-instance drift cannot make todos flicker.

## 14. Self-Approval Prohibition

One hard rule: **an applicant can never be their own approver**, nor reach that result through any indirect path.

| Scenario | Verdict | Reason |
| :--- | :--- | :--- |
| `operator_id == applicant_id` | reject, `approval.self_approval_forbidden` | direct self-approval |
| delegate == original applicant | reject, same key | self-approval through delegation |
| delegator delegates an item they applied for | allow | a different person still decides; the todo source is recorded |
| the applicant happens to be inside the role candidate set | remove the applicant during candidate resolution; if the set becomes empty, fall back per section 12.1 | holding the role is not a licence to self-approve |
| the applicant is the delegatee's superior, forming `A -> B(applicant)` | refuse to create the delegation | indirect self-approval |

Implementation requirement: the guard lives in the **service layer, inside the write transaction**, not as a hidden frontend button. Its inputs are the triple `operator_id / applicant_id / on_behalf_of_id`; missing the `on_behalf_of` check means the rule is not implemented.

Relation to existing mechanisms: `SecureActionMiddleware` only proves that the session passed step-up recently, and it binds `claims.UserID` to the current session subject (`backend/internal/middleware/secure_action_middleware.go:37-49`). Step-up therefore complements, but cannot replace, the table above.

## 15. Concurrency and Races

### 15.1 Optimistic locking baseline

`system_approval_instance.version` (item level) and `system_approval_step.version` (step level) are the arbitration columns. Every transition is a conditional UPDATE plus a `RowsAffected` check:

```sql
UPDATE system_approval_instance
   SET status = 'approved', current_step_no = 3, version = version + 1, updated_at = NOW(3)
 WHERE id = ? AND tenant_id = ? AND status = 'pending' AND version = ?;
```

`RowsAffected == 0` means the item is missing, belongs to another tenant, or somebody already decided -> the service returns `common.NewConflict("approval.decision_stale")`. **The response shape is an envelope code, not HTTP 409**: `CodeForError` maps `ErrConflict` to `code=400` and the repository has no 409 channel at all (`backend/pkg/common/error.go:85-104`); the HTTP status stays 200 because business failures go through `FailWithError -> FailWithData` (`backend/pkg/common/http/response.go:47-59`, `:98-111`). This matches existing repository practice: acknowledging a security event already maps `RowsAffected == 0` to `gorm.ErrRecordNotFound` (`backend/modules/auth/security/security_service.go:374-389`), and the recorded governance conclusion is "the unique index is the final arbiter under concurrency; the pre-check stays as a fast path, the loser goes through `IsDuplicateKey`" (`docs/harness/tasks/2026-10-23-iam-residual-closeout.task.md:133`).

### 15.2 Two reviewers deciding at once (the core case)

| Time | Session A | Session B | Outcome |
| :--- | :--- | :--- | :--- |
| t0 | reads item, `version = 7` | reads item, `version = 7` | both see an actionable item |
| t1 | `approve`, CAS matches `version = 7`, writes, `version = 8` | - | A wins |
| t2 | - | `reject`, CAS on `version = 7` fails, `RowsAffected = 0` | B gets envelope `code=400` + `approval.decision_stale` and **writes no action record** |

Requirements:

- The loser must leave no `system_approval_action_log` row: the decision log and the state transition share one transaction, and a failed CAS rolls the whole thing back. A recorded-but-unapplied decision is a defect.
- The loser's client must refetch the item and show the terminal state; it must not silently replay as "then I reject instead".
- Step-level concurrency (any-one approval) needs both `system_approval_step.version` and the item `version` updated; missing either allows two candidates to each advance one step.

### 15.3 Other races

| Race | Arbitration |
| :--- | :--- |
| `transfer` vs `approve` | same `version` CAS; loser gets 409 |
| concurrent duplicate submit for one `biz_type + biz_id` | partial unique index `uk(tenant_id, workflow_type, biz_type, biz_id)` restricted to non-terminal states (database design section 6): the winner inserts, the loser maps `IsDuplicateKey` to `approval.duplicate_in_flight` |
| timeout scan vs human decision | the scan uses the same CAS statement; a stale `version` means skip this round, never overwrite a human decision |
| maintenance task running on several instances | reuse `backend/pkg/maintenance` overlap protection: a task never overlaps itself (`maintenance.Result{Outcome: OutcomeOverlap}`, `backend/pkg/maintenance/maintenance.go:155-190`) |

### 15.4 Batch decisions

Batch actions execute the same CAS **per item** in the service layer and return `successCount / failedCount / failures[]` (`failures[].reason` carries an error key). Batch must not be a single multi-row UPDATE: that cannot honour per-item `version` preconditions or per-item callbacks (section 10.3), and cannot produce a trustworthy partial failure. Batch input is capped at 500 ids, the same limit as security-event batch acknowledgement (`security_service.go:417-421`).

## 16. Timeout and Escalation

### 16.1 Definitions

- `due_at` = `submitted_at + per-type sla_hours`, or a "working hours" approximation configured per type (phase one implements natural hours only; working calendars are an open decision, section 23).
- Overdue = `status = 'pending' AND due_at < NOW()`.

### 16.2 Scan implementation

Phase one uses the platform's existing maintenance registry instead of waiting for the unimplemented scheduler center:

```go
reg.Register(maintenance.Task{
    Name:     ApprovalTimeoutTaskName,
    Interval: maintenance.DefaultInterval,
    Run:      func(context.Context) error { return s.RunApprovalTimeoutEscalation() },
})
```

That is the repository's current pattern (`backend/modules/system/audit/audit_service.go:317-323`, `backend/pkg/maintenance/maintenance.go`, registered at `backend/modules/system/system_modules.go:103`, driven from `backend/cmd/server/main.go:205-212`). The default cycle is 15 minutes (`backend/pkg/maintenance/defaults.go:10`), so the section 16.1 SLA promise is hourly at best - never claim minute precision. Once [Scheduler Center Design](./SCHEDULER_CENTER_DESIGN.en.md) (Chinese: [SCHEDULER_CENTER_DESIGN.md](./SCHEDULER_CENTER_DESIGN.md)) lands, this task may move under it; that move is an implementation change and must be written back here.

### 16.3 Escalation steps

On overdue, run in order and stop at the first success:

1. `escalate` to the next level up the department leader chain (section 12.1, third source).
2. No upper level - reassign to the fallback role configured for the type.
3. Fallback also empty - stop reassigning, write one system action record plus a notification fan-out (section 19.4), and stop incrementing `escalated_count`.

Guard: once `escalated_count >= approval.max_escalations` (default 2), escalation stops permanently so that a 15-minute cycle cannot flood todos and notifications.

### 16.4 Reminders and auto-decision

- Overdue never auto-approves and never auto-rejects. Auto-decision turns the SLA into "don't review and it passes". Changing this is a product decision - section 23.
- Pre-deadline reminders (for example at 25% remaining) only send notifications and change no state; fan-out goes through [Notice Center Design](./NOTICE_CENTER_DESIGN.en.md) (Chinese: [NOTICE_CENTER_DESIGN.md](./NOTICE_CENTER_DESIGN.md)), which this design does not duplicate.

## 17. Withdrawal and Re-submission

### 17.1 Withdrawal

- Only the applicant may initiate it (T9).
- Precondition: `status = 'pending'` and **no `approve/reject` action recorded yet**. An item somebody already decided cannot be withdrawn as if nothing happened - that would pit the decision log against the terminal state. A type may explicitly open this with `withdraw_after_decision`; default off.
- Semantics are `cancelled + terminal_reason = withdrawn`, not a physical delete and not a soft delete (`deleted_at`). Withdrawn items must stay visible under "my submissions".

### 17.2 Cancellation

`cancel` and `withdraw` reach the same state but mean different things: `cancel` is the generic "void the draft / stop the flow" action (including source-domain invalidation), `withdraw` is explicitly the applicant stepping back. Reporting and audit must tell them apart, hence `terminal_reason` is mandatory.

### 17.3 Re-submission

- After withdrawal or cancellation the applicant may re-submit **for the same `biz_type + biz_id`**: a new instance row (new `id`, `version = 0`, `source_instance_id` pointing at the previous one).
- Never revive a `cancelled` row: terminal states are irreversible (section 11.2), and revival destroys the audit trail.
- Duplicate in-flight items are blocked by the partial unique index in section 15.3: only one non-terminal instance may exist per bizId, otherwise `approval.duplicate_in_flight`.

### 17.4 Correcting an approved item

If an `approved` item turns out wrong: run the source domain's own "undo the applied change" flow and raise a new reverse approval (`workflow_type`) in the approval center. The platform does not gain a fake "reverse approval" state.

## 18. Idempotency of Decision Actions

The platform currently has **no** request-level idempotency infrastructure (repo-wide `idempot` hits are comments and migration semantics only; there is no `Idempotency-Key` middleware). Phase one therefore guarantees idempotency with three server-side invariants rather than a client convention:

1. **State guard**: only one effective decision per `(instance_id, step_no)`. Deciding again on a terminal item returns `approval.transition_invalid` and **adds no action record**.
2. **CAS**: mismatched `version` returns 409 `approval.decision_stale` (section 15).
3. **Unique key**: `(tenant_id, instance_id, step_no, operator_id, action)` arbitrates "the same person submitting the same decision twice". On `IsDuplicateKey` the service replays the **first** result (200 plus the original action id), not 409 - clicking "approve" twice as the same person is one business fact.

Replay data comes from the action record itself (`system_approval_action_log.id` plus `instance.version`); no extra table is introduced.

Whether to add a general `Idempotency-Key` header as a platform capability (it would affect write APIs of other domains) is recorded as open decision O3 in section 23 for the maintainer; this design does not pre-decide it.

Retry UX is a mitigations only: the decision button must show a `submitting` state and be disabled, but the three invariants above are the actual guarantee.

## 19. Audit Trail Shape

### 19.1 Two layers, no double bookkeeping

| Layer | Carrier | Written by | Purpose |
| :--- | :--- | :--- | :--- |
| unified operation log | `system_log_oper` | `middleware.OperationLogMiddleware` (`backend/internal/middleware/operation_log_middleware.go`), recording every non-GET request | governance and compliance search, export, retention cleanup |
| approval action trail | `system_approval_action_log` (new) | the approval service, in the same transaction as the transition | item timeline, accountability, deriving `processed-by-me` |

The two are not duplicates: the first says "an endpoint was called", the second says "what happened in the business sense". The timeline reads only the second; it never parses the payload of the first.

### 19.2 Action trail fields

| Field | Meaning |
| :--- | :--- |
| `instance_id` / `step_no` | location |
| `action` | `submit/approve/reject/transfer/cancel/withdraw/escalate` |
| `operator_id` / `operator_username` | actual actor (the delegate, in delegation cases) |
| `on_behalf_of_id` | delegator; 0 when not delegated |
| `result_status` | item state after this action |
| `from_assignee_id` / `to_assignee_id` | only for `transfer/escalate` |
| `note` | summary note; mandatory for decisions |
| `version_before` / `version_after` | concurrency arbitration evidence; essential for rebuilding the timeline |
| `request_id` | links `system_log_oper.request_id` and the `X-Request-ID` response header (`backend/pkg/common/http/request_context.go:10-13`) |
| `tenant_id` | tenant ownership (section 21) |
| `created_at` | action time |

`request_id` makes "who clicked approve in which request" traceable in both directions (`docs/designs/DATABASE.md` §3.2 already defines `request_id` as the field joining API responses, frontend errors and unified audit).

### 19.3 Writing audit metadata

Handlers must attach structured titles and business types to approval routes instead of inheriting the default path title:

```go
common.SetAuditMetadata(c, "audit.title.approval.decide", common.BusinessUpdate)
```

Source: `backend/pkg/common/audit.go:31-34` (`BusinessInsert/Update/Delete/Grant/Export/Import/Force/Clean`). `source_domain / source_page / failure_category` are derived search columns backfilled by startup migrations; runtime queries no longer trigger compensation backfill (`docs/designs/DATABASE.md` §3.2).

### 19.4 Relationship with the notice center

Approval events (new todo, about to expire, escalated, decided) fan out as messages with `category = task` to the notice center (`docs/designs/NOTICE_CENTER_DESIGN.md` §4 message model and §5 `jumpType/jumpTarget` contract). The approval center ships no messaging of its own and keeps no read state; jump targets must lead to the approval detail or the source-domain detail, and where `jumpType = none` no fake jump button may be rendered (notice center section 5.1).

## 20. Permission Keys, Menu and i18n Impact

### 20.1 Permission key inventory

Names must satisfy the governance validator grammar `(?:system|business|platform):[a-z0-9:-]+` (`scripts/harness/check-permission-contract.mjs:24`). The approval center is `platform`, so its keys are `platform:approval:*`; a business pilot's action keys stay in `business:<module>:*` and must not borrow platform keys.

| Permission key | Semantics | Bound routes (see API contract section 4) |
| :--- | :--- | :--- |
| `platform:approval:view` | approval-center page admission | `GET /api/v1/platform/approval/list` |
| `platform:approval:detail` | read one item and its timeline | `GET /api/v1/platform/approval/:id` |
| `platform:approval:submit` | create / submit | `POST /api/v1/platform/approval`, `POST /:id/submit` |
| `platform:approval:act` | execute `approve/reject` | `POST /:id/approve`, `POST /:id/reject` |
| `platform:approval:transfer` | transfer | `POST /:id/transfer` |
| `platform:approval:withdraw` | withdraw | `POST /:id/withdraw` |
| `platform:approval:batch-act` | batch decision (separate key, never reuses `act`) | `POST /api/v1/platform/approval/batch-decide` |
| `platform:approval:delegate-manage` | create / revoke own delegations | `POST/DELETE /api/v1/platform/approval/delegations...` |
| `platform:approval:type-manage` | maintain `workflow_type` definitions (sensitive; writes need step-up) | `PUT /api/v1/platform/approval/types/:code` |
| `platform:approval:export` | export (as `POST /export`) | `POST /api/v1/platform/approval/export` |
| `business:<module>:approval-act` | domain-owned action (registered per pilot) | business-domain routes |

Four hard rules:

1. **Page permission is not list permission**: `:view` guards the page, `:detail` / `:act` guard actions; one `:list` key must not cover create/approve/delete (`SYSTEM_IAM_CONTRACT.md` §7.2; `scripts/harness/check-permission-contract.mjs:122-145` reports `action menu Type F must not use a list permission`).
2. **Batch actions need a separate permission key + step-up + partial-success payload** (`SYSTEM_IAM_CONTRACT.md` §9.0).
3. Keys are declared through `page_perm` / `perms` in `menuSeed`, with action nodes (`Type: "F"`) carrying action keys (`backend/modules/system/seed.go:203-250`; `platform:dashboard:view` is the same shape for a page permission).
4. Menus carry navigation only; authorization lives in `system_role_permission` plus Casbin route policies - the two stay decoupled (`docs/designs/PERMISSION_MENU_DECOUPLING.md`, `backend/modules/system/iam/permission/`).

### 20.2 Error codes

The backend returns stable keys, never natural language (`docs/designs/ERROR_CODE_AND_I18N.md` §3.1). New `approval.*` domain codes (full table in the API contract section 5): `approval.not_found`, `approval.transition_invalid`, `approval.decision_stale`, `approval.self_approval_forbidden`, `approval.duplicate_in_flight`, `approval.approver_unresolved`, `approval.note_required`, `approval.delegate_chain_not_allowed`, `approval.no_permission_on_item`.

### 20.3 Permission workbench registration duty

Once the new keys and routes exist, they must also be added to `requiredAPIRoutesByPermission` (`backend/modules/system/iam/permission/permission_workbench.go:622-644`). That table has a drift guard: `TestRequiredAPIRoutesExistOnEngine` (`backend/modules/system/iam/permission/permission_workbench_routes_test.go`) builds the real route table and fails on stale entries. Skipping the registration means the workbench reports the approval center as an unknown permission.

### 20.4 Menu and routes

- Entry point: the approval center hangs on platform navigation (`/platform/approval`); `Component` uses the platform module path and `RouteName` must be globally unique (`check-menu-contract.mjs` reports duplicate `routeName`).
- The detail route `/platform/approval/:id` is a secondary route and must satisfy the corresponding smoke contract (e.g. `frontend/tests/smoke/system/system-routes.spec.ts`).
- Frontend registration follows the audit module exactly: `defineModule({ routes, menus, dashboardWidgets, permissions, i18nNamespaces })` (`frontend/src/modules/system/audit/index.ts`), appended to `systemModules` in `frontend/src/core/router/modules.ts`.
- Page admission: it must be decided whether the approval center is a **task-first page** or a **governance page** (`PLATFORM_CONTRACT.md` §7.4 allows governance drawers on governance and audit pages only). This design treats it as task-first: no governance drawer, no Hero card; the dashboard consumes a summary widget with `cleanupPolicy: 'hide_when_forbidden'`, exactly like the audit module.

### 20.5 i18n impact

| Namespace | Content | Location |
| :--- | :--- | :--- |
| `system.menu.approval*` | menu titles | `frontend/src/i18n/resources/zh-CN.ts` + `en-US.ts` (flat dotted keys, existing `system.menu.*` convention) |
| `platform.approval.*` | page copy, column headers, form hints, view tabs | frontend module declares `i18nNamespaces: ['platform.approval']` |
| `approval.status.*` / `approval.action.*` | state and action copy (one-to-one with section 11 states and section 5 actions) | same, backed by `system_dict_*` delivery |
| `approval.*` (error keys) | all codes in section 20.2 | backend built-in locale file `backend/modules/system/i18n/builtin_locale_resources.json` (where keys such as `auth.operation.verification_required` live, in all five locales zh-CN/en-US/ja-JP/ko-KR/fr-FR) |

Constraint: every displayed string goes through i18n; hardcoding is blocked by `npm run check:i18n-hardcode`, part of the `prebuild` chain in `frontend/package.json`. Adding a state or action means changing dictionary, backend keys and frontend copy in the same commit - otherwise it is not done (`SYSTEM_IAM_CONTRACT.md` §7.3: menu, route, permission point, i18n and seed are registered together).

## 21. Data and Tenant Boundary

### 21.1 Scope classification (registration duty)

Per the four-way classification in `docs/designs/TENANT_RESOURCE_SCOPE_MATRIX.md` §1:

| Table | Classification | Notes |
| :--- | :--- | :--- |
| `system_approval_workflow_type` | `platform-global` | type definitions are platform configuration with no `tenant_id`, same reasoning as `tenant_plans` ("platform-global table: no tenant_id column", matrix section 2) |
| `system_approval_instance` | `tenant-owned` | carries `tenant_id`; reads apply explicit `tenant.WithTenantScope` |
| `system_approval_step` | `derived` | follows the item through `instance_id`; does not arbitrate tenancy on its own |
| `system_approval_action_log` | `derived` | same; audit fact |
| `system_approval_delegation` | `tenant-owned` | delegation is "inside one tenant, this person hands todos to that person" and must be unique per tenant |

Unique-key registration (matrix section 3): the in-flight uniqueness key of `system_approval_instance`, `(tenant_id, workflow_type, biz_type, biz_id)` with the partial non-terminal condition (database design section 6), must be registered in matrix section 3 before implementation.

### 21.2 Consistency with the contracts

- Resolution priority, the meaning of `tenant_id = 0`, and "`multi` is a superset of `compat`" follow `TENANT_CONTRACT_V1.md` §3.1/§3.4/§6 exactly; this design adds no parallel rules. The approval center is a platform console route, so its chain must include `PlatformScopeTenantContextMiddleware` (`contracts.SystemProtectedGroup` / `SystemAuthGroup`, `backend/pkg/contracts/route_groups.go`).
- Cross-tenant operations reads (platform operator browsing all items under `multi`) are allowed only on an explicitly declared platform route with `X-Tenant-Id` plus platform capability and audit (contract §3.1 rule 1, §3.3). Phase one does **not** implement a cross-tenant aggregate view - open decision O4 in section 23.
- Business pilots: `modules/business/*` does not participate in tenant isolation in V1 (contract §2.6) and business tables carry no `tenant_id`. So "a business approval item of tenant A" still points at a global business object. This is a known semantic hole handled in sections 9 and 23-O1. Claiming that business approvals are tenant-isolated is forbidden (contract §2.6 "forbidden claims").
- Quota: approval-related numbers (for instance "max in-flight items per tenant") are **metrics and display only**; the backend must never reject a write because a plan quota is exceeded (contract §1 Out: plan quota is a measurement/display dimension). Turning that into a hard limit is a contract change.

### 21.3 Data-permission hooks

Whether a list read also applies the IAM row scope (`self / dept_and_children / all`) is an explicit option: `contracts.SystemProtectedGroup` decides it through `WithDataScope()` -> `middleware.DataScopeMiddleware` (`backend/pkg/contracts/route_groups.go:137-150`). This design fixes:

- `pending-for-me` / `processed-by-me` / `submitted-by-me` **do not** apply DataScope (the view predicate is already the ownership boundary).
- The "all approval items" governance view **must** apply DataScope, otherwise a role that should see one department sees every item on the platform. That view needs its own page permission key and answers `403 permission.denied` when it is missing.

## 22. Definition of Done

First phase is complete when Pantheon has:

- a unified approval-item model
- an approval-center entry
- a minimal action contract
- at least one real business pilot
- permission and audit traces

Plus the engineering surface the companion docs define, each individually checkable:

- all 12 transitions of section 11 have service-layer guards and error keys, so no unlisted combination is silently allowed
- section 14 self-approval is enforced in the service layer, including the indirect delegation path
- section 15 concurrency: two simultaneous decisions produce a 409 and the loser writes no action record; the test plan has the matching case
- section 16 timeout scanning is registered in `maintenance.Registry` with overlap protection and an escalation cap
- section 17 withdrawal creates a new instance instead of reviving a terminal one
- section 18 idempotency: the three invariants hold (a repeated identical action replays the first result and stores nothing extra)
- section 19 action trail and `system_log_oper` are joinable through `request_id`
- section 20 permission keys, menu, i18n and `requiredAPIRoutesByPermission` are all registered, with `Docs Governance` and the frontend `prebuild` contract chain green

It is not measured by whether a full BPM platform already exists.

## 23. Open decisions

The questions below are not decided by this design. Each carries options, impact and a recommendation. The decision owner is the **maintainer** (product/architecture ruling); implementers must not silently pick a default.

| ID | Topic | Options | Impact | Recommendation |
| :--- | :--- | :--- | :--- | :--- |
| O1 | Tenant semantics of a business-domain pilot under `multi` | A. phase one promises compat only; B. items carry `tenant_id` while the business detail jump addresses a global object (`tenant_id = 0`); C. wait until contract §2.6 V2 preconditions are met | sections 9, 21.2, acceptance scope | B - honest and non-blocking, with the limitation stated |
| O2 | SLA on a working calendar | A. natural hours; B. working days plus holiday calendar (new configuration and data source) | section 16.1, column semantics | A for phase one |
| O3 | Introduce a platform-wide `Idempotency-Key` middleware | A. keep only the three invariants of section 18; B. add platform capability (cross-domain shared contract change) | every write route, `route_groups.go` chains | A first; B needs its own design and contract review |
| O4 | Cross-tenant aggregate view for platform operators | A. not implemented; B. read-only aggregate per contract §3.1 rule 1 with `PlatformCapabilityMiddleware` | section 21.2, permission keys, audit | A for phase one |
| O5 | When all-signer / quorum approval enters | A. keep section 12.3 as frozen semantics, unimplemented; B. implement in phase one | sections 12, table count, test surface | A |
| O6 | Auto-decision on timeout | A. never; B. allow per-type "auto-approve on timeout" | section 16.4, security boundary | A; B equals an approval bypass surface |
| O7 | Page admission class of the approval center | A. task-first page (no governance drawer); B. governance page | section 20.4, `PLATFORM_CONTRACT.md` §7.4 whitelist | A; if an "all items" governance view is built, classify that view separately |
