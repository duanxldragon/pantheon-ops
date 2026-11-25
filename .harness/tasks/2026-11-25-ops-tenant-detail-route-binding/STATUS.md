# Base 发版后恢复业务详情路由租户绑定

- Status: completed
- Updated: 2026-11-25
- Baseline: 444ce8e（报告）/ a2d69d4a（规划）；锁定 foundation pantheon-base-v0.19.0（cc2fe737）
- Owner: 业务租户路由负责人
- Priority: P1
- Activation: core
- Dependencies: 无；前置为 Base 正式 foundation release 携带 #525（`63940437` 静态/参数路由隔离）

## 来源

由 T04 `2026-11-20-ops-core-business-tenant-isolation` 关闭时拆出的残余：Base `keyMatch2` 让 `GET /packages/:id` 同时匹配 `/packages/export`，详情路由因此保持未绑定。Base 侧已合并修复但 `VERSION` 仍为 `0.18.9`、无新 tag，Ops 锁定快照不含该修复。

## Result 2026-11-25

Base v0.19.0（含 #525）已由 Ops PR #223 消费。详情类 GET 路由已按路由模板授权绑定：cmdb `host:detail`、`group:detail`；deploy package/template 走 view/list 键，task 详情族走 `task:detail`。路由/策略契约测试证明详情不会放行 export，反之亦然。证据见 `../../evidence/2026-11-25-ops-tenant-detail-route-binding/`。

## Known Gaps

- HTTP 层双租户反例（猜 ID 详情/导出/回调跨租户被拒、同租户详情 200）未在本地运行，需完整认证栈与真实多租户库，转 T19。
- `WithDataScope` 本身不加 `tenant_id` 过滤；详情读取的行级租户隔离在本包内未独立证明，本包证明的是授权绑定。
- 写入路径（claim 时落 `tenant_id`）与 multi 端到端归 T14 与 T19。
