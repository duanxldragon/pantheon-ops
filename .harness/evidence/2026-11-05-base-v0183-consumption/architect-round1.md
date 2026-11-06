**ARCH lane：BLOCK。** 业务修复和受限迁移装配方向可接受，但最新实现仍有一处必须补齐的继承门禁问题；与 v0.18.3 尚未发布无关。

审查覆盖原始 **11 个修复资产 + manifest**，并补审期间落地的 **3 个装配文件、2 份继承说明，共 17 项**；迁移搬迁按同一资产计。

1. **阻塞：生成迁移未纳入共享漂移检查。**  
   重建器已将业务 SQL 装配到固定目标并记入报告（[rebuild-from-base.mjs:269](/D:/workspace/go/pantheon-platform/audit-artifacts/ops-test-fixes-2026-11-05/scripts/business-overlay/rebuild-from-base.mjs:269)），但漂移检查仅识别 `businessPaths`（[check-shared-drift.mjs:81](/D:/workspace/go/pantheon-platform/audit-artifacts/ops-test-fixes-2026-11-05/scripts/business-overlay/check-shared-drift.mjs:81)）。两个 `000032` 目标均不属于该集合，也没有漂移例外，因此会进入 `DRIFT_MISSING_IN_BASE` 并失败（[同文件:140](/D:/workspace/go/pantheon-platform/audit-artifacts/ops-test-fixes-2026-11-05/scripts/business-overlay/check-shared-drift.mjs:140)）。只读诊断已确认这两个分类结果。

   **根因：装配器扩展了业务资产类型，所有权门禁没有同步。**  
   建议小范围复用 `planBusinessMigrations`，识别其派生的精确目标，并校验目标与 canonical SQL 一致；保留 Base 冲突拒绝和通用目录禁令，避免直接豁免全部 `generatedFiles`。

2. **业务修复未发现新增架构阻塞。**  
   创建、导入均显式初始化版本 1，匹配现有正版本 CAS 条件（[host_service.go:180](/D:/workspace/go/pantheon-platform/audit-artifacts/ops-test-fixes-2026-11-05/backend/modules/business/cmdb/host/host_service.go:180)、[host_import_export.go:472](/D:/workspace/go/pantheon-platform/audit-artifacts/ops-test-fixes-2026-11-05/backend/modules/business/cmdb/host/host_import_export.go:472)、[host_state.go:52](/D:/workspace/go/pantheon-platform/audit-artifacts/ops-test-fixes-2026-11-05/backend/modules/business/cmdb/host/host_state.go:52)）。部署修复仍断言错误持久化、活动执行保持 running 和租约保留，没有以删除关键断言换取通过（[deploy_service_test.go:1074](/D:/workspace/go/pantheon-platform/audit-artifacts/ops-test-fixes-2026-11-05/backend/modules/business/deploy/deploy_service_test.go:1074)、[deploy_reliability_test.go:92](/D:/workspace/go/pantheon-platform/audit-artifacts/ops-test-fixes-2026-11-05/backend/modules/business/deploy/deploy_reliability_test.go:92)）。SSRF 放宽仅注入测试实例，生产构造仍使用原策略（[datasource_service_test.go:16](/D:/workspace/go/pantheon-platform/audit-artifacts/ops-test-fixes-2026-11-05/backend/modules/business/observability/datasource_service_test.go:16)、[datasource_service.go:34](/D:/workspace/go/pantheon-platform/audit-artifacts/ops-test-fixes-2026-11-05/backend/modules/business/observability/datasource_service.go:34)）。

3. **迁移设计可保留，但需明确其边界。**  
   SQL 守卫表/列存在性，仅回填零版本；它没有修改数据库列默认值的 DDL（[000032.up.sql:37](/D:/workspace/go/pantheon-platform/audit-artifacts/ops-test-fixes-2026-11-05/backend/modules/business/cmdb/host/migrations/000032_backfill_cmdb_host_state_version.up.sql:37)）。模块内保存 canonical 文件、装配到既有版本链，符合业务域所有权；代价是继续共享迁移编号空间，未来碰撞必须停止升级，不能重编号已发布迁移（[PROJECT_INHERITANCE.md:87](/D:/workspace/go/pantheon-platform/audit-artifacts/ops-test-fixes-2026-11-05/docs/PROJECT_INHERITANCE.md:87)）。

解除本轮阻塞所需验证：

- 重建后执行漂移检查：Base 文件保持受保护，两个派生 SQL 被正确识别；篡改、未声明目标仍失败。
- 补齐缺失实体文件、缺表/缺列、down 后重新应用的回归；现有迁移测试第二次调用仅验证“无待执行迁移”，不足以独立证明 SQL 重放行为（[迁移测试:58](/D:/workspace/go/pantheon-platform/audit-artifacts/ops-test-fixes-2026-11-05/backend/modules/business/cmdb/host/host_state_version_migration_test.go:58)）。
- 刷新源码 manifest，使其覆盖搬迁和装配扩展。

本轮亲自完成只读 manifest 正反断言、overlay 检查、脚本语法及差异检查；读取了最新 **14 项 overlay 测试通过及 MySQL 定向 race 通过**的[执行记录](/D:/workspace/go/pantheon-platform/audit-artifacts/base-freeze-2026-11-05/opsAssembly-restricted-migration-results.json:3)，未重跑长套件。正式制品哈希、最终重建、全量 race、前端与业务 smoke 仍按[消费计划](/D:/workspace/go/pantheon-platform/audit-artifacts/ops-test-fixes-2026-11-05/.harness/tasks/2026-11-05-base-v0183-consumption/plan.md:7)后续执行，不属于本次发布完成声明。