# Base 发版后恢复业务详情路由租户绑定

- Status: planned
- Updated: 2026-11-25
- Baseline: 444ce8e（报告）/ a2d69d4a（规划）；锁定 foundation pantheon-base-v0.18.9（6316f9b7）
- Owner: 业务租户路由负责人
- Priority: P1
- Activation: core
- Dependencies: 无；前置为 Base 正式 foundation release 携带 #525（`63940437` 静态/参数路由隔离）

## 来源

由 T04 `2026-11-20-ops-core-business-tenant-isolation` 关闭时拆出的残余：Base `keyMatch2` 让 `GET /packages/:id` 同时匹配 `/packages/export`，详情路由因此保持未绑定。Base 侧已合并修复但 `VERSION` 仍为 `0.18.9`、无新 tag，Ops 锁定快照不含该修复。

## Next Atomic Action

确认 Base 是否已发布携带该修复的正式 release；若已发布，按锁定快照消费并重建 Ops，先在隔离库补双租户详情/导出反例，再恢复详情路由绑定。

## Evidence

尚未执行。开工前先核实锁定快照 commit 与重建回执。

## Known Gaps

- 详情路由绑定在正式 release 发布前不可做。
- 写入路径（claim 时落 `tenant_id`）与 multi 端到端归 T14 `2026-11-20-ops-extension-and-async-tenant-isolation` 与 `2026-11-25-ops-residual-live-acceptance`。
