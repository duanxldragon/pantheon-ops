# T04 核心业务资源租户归属与兼容迁移 — 实施摘要

- Task: `2026-11-20-ops-core-business-tenant-isolation`
- 日期: 2026-11-24
- 状态: **schema 迁移已完成并验证；运行时租户隔离被 Base 上游缺口阻塞**
- 交付层级: L2（自审见 `review.md`）

## 已交付

| 验收项 | 状态 | 证据 |
|---|---|---|
| compat 既有数据可用（tenant_id=0） | schema 就绪 | 迁移对既有表幂等加列；未做 live 数据核对（gap） |
| 旧库升级：加 tenant_id、唯一键重建为 tenant-first | 已完成 | `000041_core_business_tenant_ownership.{up,down}.sql`，48 条 `SET @sql := IF(...); PREPARE; EXECUTE; DEALLOCATE` 幂等 DDL |
| 装配副本一致性 | 已完成 | 业务模块源与 `backend/pkg/database/migrations/` 副本 `cmp` 字节一致（up/down） |
| 双租户同名资源共存 + 单租户内重名拒绝 | 已完成（schema 级） | `TestTenantOwnershipMigrationUpgradesLegacyNameKeys` PASS（真实 MySQL testmysql） |
| 猜 ID/导出/回调/跨模块引用不能跨租户读写 | **阻塞** | Base `keyMatch2` 使 `GET /packages/:id` 同时匹配 `/packages/export`，详情路由无法安全绑定 → 保留未绑定（anti-widening），运行时隔离需 Base 正式 release |
| 异步任务重启后仍使用原租户 | 未验证 | gap |

## 验证

- 迁移测试 PASS 0.635s；up/down 镜像 `cmp` 一致；48 条幂等 DDL。

## 显式 gap（上游依赖）

1. Base `keyMatch2` 静态路由守卫缺失，阻止核心业务模块转为租户作用域路由。
2. compat/multi 真实库升级、回滚行数与归属核对未在本候选执行。
3. 异步任务原租户保持未端到端验证。
