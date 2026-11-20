# O4 Observability Query Workbench

- Priority: `P0`
- Layer: `business/observability`
- Status: `blocked`
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
