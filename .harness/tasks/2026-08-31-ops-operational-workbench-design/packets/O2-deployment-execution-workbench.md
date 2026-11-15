# O2 Deployment Execution Workbench

- Priority: `P0`
- Layer: `business/deploy`
- Status: `in-progress; scoped implementation verified, complete acceptance open`
- Base Dependency: B1 sticky actions + B3 `ExecutionStepRail` and `TaskLogViewer`
- Business Dependency: immutable task/host/attempt/step/event contracts

## Outcome

把部署任务详情升级为可快速定位目标、步骤、日志和恢复动作的执行工作台。

## In Scope

- 任务 hero、状态/新鲜度、目标列表、步骤轨、日志主区和审计/参数抽屉。
- host/attempt/step/stream selection 与 URL/deep-link 状态。
- SSE last-sequence reconnect、stale/polling fallback 和 partial failure。
- permission/state-aware pause/resume/cancel/retry/rollback entry points。

## Out Of Scope

- 不在前端推导状态机或伪造运行进度。
- 不改变 Deploy 引擎本身的幂等、租约、回写和恢复模型。
- 不显示明文 credential、秘密参数或未脱敏命令。

## Expected Files

- `frontend/src/modules/business/deploy/task/DeployTaskDetail.tsx` 及拆分的业务 composition。
- Deploy API/event adapters、i18n、smoke/Playwright。
- 后端只补现有 task event/log DTO 的必要字段与测试。

## Acceptance

1. 用户能从失败汇总在两次交互内定位 host + step + first error log。
2. SSE 断开后保留内容并从 last sequence 恢复；fallback 显式 stale。
3. 动作严格匹配后端状态、权限、DataScope 和 state version；冲突要求刷新。
4. mobile 首屏优先显示状态和失败目标，目标/步骤/日志切换无内容遮挡。
5. 多主机、长日志和实时更新不引起布局跳动或无限 DOM 增长。

## Evidence

- backend：event authorization、sequence reconnect、redaction、state transition and idempotency tests。
- frontend：target/step/log selection、action matrix、stale/conflict/partial failure tests。
- browser：happy path、one-host failure、multi-host partial、disconnect/reconnect、forbidden。
- runtime：脱敏 SSE/log 片段；无真实 VM/credential 时记录明确 gap。
- screenshots：desktop/mobile x running/failed/stale/forbidden x light/dark。

## Gates

- Base lock、Deploy public event DTO、状态机/权限变化、真实主机动作和最终视觉验收。

## Reconciliation 2026-11-07

The required Base primitive is present in official v0.18.6. This resolves the prior component-release dependency only; all implementation and acceptance criteria above remain open. See the parent EXECUTION_QUEUE.md and current execution evidence.

## Scoped implementation 2026-11-07

Scoped business implementation: target/step deep links, safe persisted step DTOs, stable SSE instance and bounded cursor logs. Backend MySQL tests and desktop/mobile light/dark browser states pass. Task command version binding is now implemented and verified; full SSH/attempt/action acceptance and official Base step-rail/style-status consumption remain open. Current evidence: `.harness/evidence/2026-11-07-ops-planned-task-closeout/summary.md`.

Manual cancellation uses the existing CAS terminal path without a worker, including legacy cancel_requested rows. Backend regression verifies one terminal event, finishedAt, successful-host preservation and duplicate-call rejection; owned runtime fixture cleanup passes. This repair does not close the complete command/version and action matrix.

## Version-bound control verification

Public pause/resume/retry/cancel require the viewed opaque stateVersion. Visibility precedes validation; request-context transaction locking commits task/host/event atomically, with worker effects after commit on the long-lived service. Task-level event sequence prevents ABA; logs and host/step events do not spuriously invalidate controls. Manual pause/resume/retry reuse manual execution rather than starting SSH. Frontend409 reports a persistent warning and refreshes without automatic replay; batch cancellation removes successful choices and retains failures.

Real MySQL concurrency/state/ABA/scope/cancellation/failure-rollback tests, complete hash-stable CGO race492.88s, frontend19/19 and one complete local browser/API24/24 pass. Latest screenshots cover desktop1440/mobile390, light/dark conflict and partial batch failure. Independent frontend review approves its supplied initial source; later UI deltas and backend reviews remain open after bounded timeouts. Full original packet acceptance remains open.

## Immutable log context continuation — 2026-11-08

Deployment selection now binds task/host/recorded attempt/step/stream to the URL and resets the provider cursor/window on source changes. The provider filters authorized reads before its 500-row cap and returns a bounded earliest stderr independently of the retained tail. Visibility precedes query validation, duplicate/invalid filters are rejected, and reads use the existing request-context/capability adapter. Forbidden responses clear both cursor and paused UI snapshots.

Checkpoint attempt numbers are mutable, so migration000034 adds immutable attempt metadata to newly appended log chunks. Existing output remains attempt0 and is available in explicit all-attempt mode, without reconstructed history. Recorded log attempts remain selectable after a checkpoint is retried; historical step state is explicitly unavailable. Migration is additive, replay-safe, guarded for system-only databases and retains column/data on down. No execution state/lease/idempotency policy changed.

Verification at this checkpoint: real MySQL paging/ownership/first-error/cancellation plus retry attribution and migration/replay/retained down pass; frontend22/22, full lint/prebuild/type-check/build pass; locked official rebuild and shared drift pass. Local API/browser2/2 with1202 selected-attempt chunks passed at1440 desktop /390 mobile, light/dark; eight screenshots and derived contact/crop sheets were inspected. Transport log/step fixtures are explicit; real SSH is not asserted. Full vet found a copied synchronization map; the fix reuses targetRequestService and passes vet. New request-adapter runtime/stage and final hash-stable CGO race are pending. The earlier complete race584.58s predates immutable metadata and must not be treated as its final verification.

Independent supplied-source Claude review timed out120.065s, providing no approval. Complete O2 remains open for final source review, historical step-state snapshots, real SSH/action acceptance, formal Base dark/status step-rail consumption and final human acceptance. Other formal packets remain open under their existing criteria.

## Final local priority checkpoint 2026-11-08

Current scoped source passes complete hash-stable CGO race, backend vet/build, operational54/54 units, frontend lint/type-check/build and final combined browser6/6 with28 inspected desktop/mobile light/dark screenshots. The final browser waits for mobile shell transitions. O4 now labels completed result source/range/limit, stale inputs and safe warning counts; upstream errors/result-type mismatch cannot become false empty/success and cross-stream limits keep newer rows. Official locked assembly/contract/drift and62 overlay tests pass. This supersedes prior pending local verification notes, while independent approval and every omitted formal criterion remain open. See current closeout summary.

## Prior SSH and label-budget checkpoint 2026-11-08

O2 now cancels actual SSH TCP/handshake/session/run work with its host context and joins output copying before reading buffers. Pre-canceled work opens no session. Connection-stage pause/cancel reuses the existing command handling instead of persisting a host failure; paused requests acknowledge immediately. No public API, schema, credential storage, engine, lease or idempotency model changed. O4 caps each row at six label previews, eighty characters plus an ellipsis, with explicit omission/shortening notices and retained raw detail. All formal O/D implementation packets remain open.

- Actual ephemeral loopback SSH protocol plus isolated real MySQL: thirteen leaf cases pass under CGO race, including password/private-key auth, pinned host-key rejection, stdin/exit status, live output cancellation, withheld session/handshake, encrypted credential/frozen script, step/log attempt attribution, writeback and pause/cancel interruption. No fixture runs an OS command or validates a real VM.
- Final complete Windows CGO race: exit0, 686.688s, 695 Go/SQL/mod/sum source hashes unchanged before/after. Required MySQL33316/Redis26316 and migration DSN configured; valid unchanged shared-package cache permitted. The earlier 879.922s Go-exit0 run changed source during its run and is explicitly invalidated, with its original log/hash receipts retained.
- Final backend vet/build and locked official v0.18.6 stage rebuild/contract/shared drift pass; current business overlay contract and62 overlay tests pass. Business source/document ownership is refreshed only after hash verification; no shared source copying or lock replacement.
- Frontend operational56/56, full lint/type-check/build pass. Two failed-before label-budget tests observed500 tags and11004 text characters; both pass after repair. The seven query authority cases remain green.
- Current Loki browser2/2 passes at1440x900 and390x844 in light/dark: success/stale/warning/cancel/denial/error/empty/recovery, field/time/raw modes and500-label details. Eighteen screenshots were produced; the four new wide-label desktop/mobile renders were inspected in the contact sheet. Mobile uses the existing intentional horizontal table scrolling; the page has no horizontal overflow. Full raw detail contains omitted label499. The first browser run failed from504 Outdated Optimize Dep; a verified owned Vite restart with--force recovered it. No gate or source workaround.
- Browser transport remains a Loki fixture with real local authentication; it is not a real datasource. Its runtime backend predates the tiny final connection-command branch; that final SSH path is verified directly by required MySQL/protocol tests and the current build, without asserting a newer true VM/API integration.
- Independent supplied-source CLI timed out120.062s and supplied no approval. The later connection-command helper and added cases are also unreviewed. Real VM/cluster/datasource, safe history/saved views, repaired formal Base release/consumption, independent complete-source and final visual/function/production signatures remain open. Test-owned services were stopped after identity verification; data/logs are retained. No upload, production operation, commit/push/publication or gate waiver.

Gate Outcomes: actual SSH race/cancellation regressions and connection-command state regressions fail before/pass after |500-label DOM/text regressions fail before/pass after | source hashes invalidate the earlier whole-suite run | browser cache failure is diagnosed and repaired without gate change | independent final review remains open.

## O2 checkpoint-history checkpoint — 2026-11-08 (current)

The business deployment workbench now separates recorded step attempts from mutable current checkpoints. Each existing step status update commits an append-only safe observation in the same transaction; history returns the last recorded version per step/attempt, at most500 entries with explicit truncation. Old checkpoints/attempt0 are not reconstructed. No engine retry/lease/recovery policy changes or commands/config/credentials/error text in history. GetTask host, current-step, log-attempt and history collections follow CMDB host visibility; full task metadata and other endpoints are not certified by this change.

- Seven required real MySQL history tests pass under CGO race/count=1: lost prior-attempt failure, atomic insert-failure rollback, twelve concurrent version/attempt writes, pause/legacy/retry, visible-host/foreign-step/request cancellation, projection cap, and fresh/replay/retained-down migration. Initial lost-history and mixed-host leakage failures are retained. Additive business migration35 is formally assembled; normal AutoMigrate=false server bootstrap reports schema35/dirty0. Production/schema-compatibility/retention/restore acceptance stays open.
- Final full backend Windows CGO race exits0 after1507.812s, all701 Go/SQL/mod/sum hashes stable before/after. Required isolated MySQL33316/Redis26316 and migration DSN configured. Valid unchanged-package cache permitted. The owned disposable MySQL began with durable1/1, used the previously recorded2/0 test I/O profile during slow seed initialization, and is restored to1/1 before shutdown; this is not performance or production durability evidence. Latest backend vet/build pass; the actual runtime uses this build. Earlier695-file SSH race is historical and does not validate the new writer.
- Current frontend61/61 operational units, full lint/type-check/build pass. Explicit valid missing attempts stay selected and show unavailable state rather than current results. Historical attempts remain selectable without logs; current checkpoint wins over an older same-attempt snapshot, and all-attempt mode labels its current-rail scope.
- Deploy module smoke18/18 passes with actual local authentication/business API fixtures, plus the focused history browser2/2. Steps/logs transport in the workbench browser remains explicit fixtures; actual history persistence is proved separately by MySQL. No real VM claim. Initial suite-port contention and missing local proxy parameter failures are retained; invocation environment/ownership is corrected without changing shared scripts or gates.
- Rendered history desktop1440x900/mobile390x844, light/dark was inspected. The initial screenshots exposed clipping hidden by the app shell; new child-boundary assertions fail on473.828px overflow in both themes. Business-only minmax grid/shrinkable control items/wrapped history caption repair it; both cases pass, with12 final screenshots. Existing Base dark step-rail white background/raw status text and full200% zoom/reduced-motion/multi-host matrix remain open; no UI polish approval is asserted.
- Locked official Basev0.18.6 final reconstruction, contract/shared drift, current overlay contract and62 overlay tests pass. Generated SQL/locale assets and business/document metadata are refreshed only after membership/SHA256 verification. No shared source or lock is copied/replaced.
- Source self-review is verification/o2-step-history-source-self-review.md. Prior installed adapters503/CLI timeout/unauthenticated Gemini supply no independent approval; no same-adapter retry or full-source approval is inferred. Real VM/complete action/restart/event/source-revocation matrix, formal repaired Base consumption, independent reviewer and final functional/visual/production signatures remain open. All O/D parent packets stay open; O7 stays deferred.

Gate Outcomes: lost-history/mixed-host/missing-attempt regressions fail before/pass after | rendered clipping motivates a stricter content boundary check and business layout repair | complete final backend hash-stable CGO race passes | runtime/proxy invocation failures are retained | independent/production/formal parent acceptance remains open.
