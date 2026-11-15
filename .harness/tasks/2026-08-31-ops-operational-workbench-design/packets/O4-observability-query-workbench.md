# O4 Observability Query Workbench

- Priority: `P0`
- Layer: `business/observability`
- Status: `in-progress; scoped implementation verified, complete acceptance open`
- Base Dependency: B2 saved views + B3 `ConditionBuilder`
- Business Dependency: scoped Prometheus/Loki/Tempo query APIs

## Outcome

把 `LogExplorer` 等查询入口升级为带时间、历史、趋势、结构化/原始结果、字段统计和资源预算的查询工作台。

## In Scope

- sticky query bar：datasource、time range、query/conditions、run/stop、refresh。
- query history 与 saved view，保存脱敏表达式和视图元数据。
- result tabs：trend、logs、field stats、raw，按数据源能力显示。
- timezone、absolute range、limit/sample、partial shard failure 和 live tail mode。

## Out Of Scope

- 不自研 Prometheus/Loki/Tempo 或存储查询结果。
- 不让客户端 AST 直接拼 SQL/PromQL/LogQL。
- 不给所有数据源伪造相同能力。

## Expected Files

- `frontend/src/modules/business/observability/LogExplorer.tsx` 及 query workspace compositions。
- observability API/DTO/compiler whitelist、history metadata、tests 和 i18n。
- existing datasource permission/DataScope/audit contracts。

## Acceptance

1. 时间范围明确 timezone 和绝对起止，refresh/live tail 不混淆。
2. stop 能取消后端查询；timeout/limit/cost budget 有明确反馈。
3. partial result、no data、query error、datasource down、forbidden、stale 各自可辨。
4. history/saved view 不保存秘密或未脱敏响应，并按用户/租户隔离。
5. desktop 与 mobile 均能运行、停止、查看结果和恢复错误；长查询不溢出。

## Evidence

- backend：query whitelist/compiler、DataScope、timeout/cancel/limit、audit、history redaction。
- frontend：time range、history、tabs、structured/raw、field include/exclude、state matrix。
- browser：query success/empty/error/partial/cancel/live tail，desktop/mobile。
- runtime：真实 Prometheus/Loki/Tempo endpoint；缺任一 endpoint/credential 分别记录 gap。
- screenshots：desktop/mobile x result/empty/error/partial x light/dark。

## Gates

- Base lock、新 query language/public API/schema、敏感历史持久化和生产数据源操作。

## Reconciliation 2026-11-07

The required Base primitive is present in official v0.18.6. This resolves the prior component-release dependency only; all implementation and acceptance criteria above remain open. See the parent EXECUTION_QUEUE.md and current execution evidence.

## Scoped implementation 2026-11-07

Scoped Loki implementation: explicit UTC ranges, run/stop, response/row budgets, request-context cancellation, denial/source isolation and recovery. Provider tests use real local HTTP/MySQL; browser transport fixtures verify UI states. Editable Base ConditionBuilder has no onChange and its add-rule control is disabled. History/redaction, multi-datasource modes, partial/live tail and real endpoints remain open. Current evidence: `.harness/evidence/2026-11-07-ops-planned-task-closeout/summary.md`.

## Bounded returned-row modes 2026-11-08

Loki now exposes logs, returned-row time buckets, field counts/distinct counts and bounded raw JSON. Statistics never claim total query matches. Field width/count and256KiB UTF-8 output budgets are explicit; invalid timestamps and omitted rows are reported, raw content remains valid JSON and supports keyboard scrolling.26 scoped units, full lint/build and4 combined light/dark desktop/mobile browser cases passed;18 screenshots were inspected. Query expressions/results are not persisted. History/saved-view isolation/redaction, official editable ConditionBuilder consumption, partial/live tail, other data sources and actual endpoints remain open. Current evidence is the closeout summary.

## Final local priority checkpoint 2026-11-08

Current scoped source passes complete hash-stable CGO race, backend vet/build, operational54/54 units, frontend lint/type-check/build and final combined browser6/6 with28 inspected desktop/mobile light/dark screenshots. The final browser waits for mobile shell transitions. O4 now labels completed result source/range/limit, stale inputs and safe warning counts; upstream errors/result-type mismatch cannot become false empty/success and cross-stream limits keep newer rows. Official locked assembly/contract/drift and62 overlay tests pass. This supersedes prior pending local verification notes, while independent approval and every omitted formal criterion remain open. See current closeout summary.

## SSH and label-budget checkpoint 2026-11-08 (current)

O2 now cancels actual SSH TCP/handshake/session/run work with its host context and joins output copying before reading buffers. Pre-canceled work opens no session. Connection-stage pause/cancel reuses the existing command handling instead of persisting a host failure; paused requests acknowledge immediately. No public API, schema, credential storage, engine, lease or idempotency model changed. O4 caps each row at six label previews, eighty characters plus an ellipsis, with explicit omission/shortening notices and retained raw detail. All formal O/D implementation packets remain open.

- Actual ephemeral loopback SSH protocol plus isolated real MySQL: thirteen leaf cases pass under CGO race, including password/private-key auth, pinned host-key rejection, stdin/exit status, live output cancellation, withheld session/handshake, encrypted credential/frozen script, step/log attempt attribution, writeback and pause/cancel interruption. No fixture runs an OS command or validates a real VM.
- Final complete Windows CGO race: exit0, 686.688s, 695 Go/SQL/mod/sum source hashes unchanged before/after. Required MySQL33316/Redis26316 and migration DSN configured; valid unchanged shared-package cache permitted. The earlier 879.922s Go-exit0 run changed source during its run and is explicitly invalidated, with its original log/hash receipts retained.
- Final backend vet/build and locked official v0.18.6 stage rebuild/contract/shared drift pass; current business overlay contract and62 overlay tests pass. Business source/document ownership is refreshed only after hash verification; no shared source copying or lock replacement.
- Frontend operational56/56, full lint/type-check/build pass. Two failed-before label-budget tests observed500 tags and11004 text characters; both pass after repair. The seven query authority cases remain green.
- Current Loki browser2/2 passes at1440x900 and390x844 in light/dark: success/stale/warning/cancel/denial/error/empty/recovery, field/time/raw modes and500-label details. Eighteen screenshots were produced; the four new wide-label desktop/mobile renders were inspected in the contact sheet. Mobile uses the existing intentional horizontal table scrolling; the page has no horizontal overflow. Full raw detail contains omitted label499. The first browser run failed from504 Outdated Optimize Dep; a verified owned Vite restart with--force recovered it. No gate or source workaround.
- Browser transport remains a Loki fixture with real local authentication; it is not a real datasource. Its runtime backend predates the tiny final connection-command branch; that final SSH path is verified directly by required MySQL/protocol tests and the current build, without asserting a newer true VM/API integration.
- Independent supplied-source CLI timed out120.062s and supplied no approval. The later connection-command helper and added cases are also unreviewed. Real VM/cluster/datasource, safe history/saved views, repaired formal Base release/consumption, independent complete-source and final visual/function/production signatures remain open. Test-owned services were stopped after identity verification; data/logs are retained. No upload, production operation, commit/push/publication or gate waiver.

Gate Outcomes: actual SSH race/cancellation regressions and connection-command state regressions fail before/pass after |500-label DOM/text regressions fail before/pass after | source hashes invalidate the earlier whole-suite run | browser cache failure is diagnosed and repaired without gate change | independent final review remains open.

## Explicit Loki follow candidate — 2026-11-09

Bounded Loki follow now uses completion-serialized five-second rolling fifteen-minute UTC queries with no response/expression persistence. Stop, edit, source change, error/denial and unmount are verified by17 scoped cases;86 operational units and current complete frontend lint/build plus official locked assembly/contract/drift pass. Rolling snapshots keep provenance and may omit logs. The existing remote browser matrix is extended but has not executed under current SSH restrictions. This is local candidate evidence, not full O4 acceptance: history/saved-view user/tenant isolation, released editable conditions, datasource-specific modes, full partial/tail and remote functional/visual/provider matrices remain open.
