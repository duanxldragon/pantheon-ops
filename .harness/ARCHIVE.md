# Pantheon Ops 任务归档索引

更新：2026-11-16。完成按原任务范围判断，不代表最新产品验收；原 evidence 字节保全，canonical 地址保持 archive-aware 可解析。

当前计划：.harness/tasks/2026-11-16-ops-pilot-delivery/plan.md；归档决策/文件 SHA-256 清单：.harness/evidence/2026-11-16-ops-readiness-plan-closeout/archive-inventory.json。

| Task                                       | 原范围                         | Manifest                                                                                     | Evidence                                                                                    | 后续                      |
| ------------------------------------------ | ------------------------------ | -------------------------------------------------------------------------------------------- | ------------------------------------------------------------------------------------------- | ------------------------- |
| 2026-08-17-ops-foundation-hardening        | 原范围已完成                   | [manifest](./archive/2026-08/tasks/2026-08-17-ops-foundation-hardening/manifest.json)        | [summary](./archive/2026-08/evidence/2026-08-17-ops-foundation-hardening/summary.md)        | 新业务 gap 由首期计划跟踪 |
| ops-business-versioned-migrations          | 原范围已完成                   | [manifest](./archive/2026-08/tasks/ops-business-versioned-migrations/manifest.json)          | [summary](./archive/2026-08/evidence/ops-business-versioned-migrations/summary.md)          | 新业务 gap 由首期计划跟踪 |
| ops-cross-module-capability-boundaries     | 原范围已完成                   | [manifest](./archive/2026-08/tasks/ops-cross-module-capability-boundaries/manifest.json)     | [summary](./archive/2026-08/evidence/ops-cross-module-capability-boundaries/summary.md)     | 新业务 gap 由首期计划跟踪 |
| ops-bizscope-datascope-ownership           | 原范围已完成                   | [manifest](./archive/2026-08/tasks/ops-bizscope-datascope-ownership/manifest.json)           | [summary](./archive/2026-08/evidence/ops-bizscope-datascope-ownership/summary.md)           | 新业务 gap 由首期计划跟踪 |
| ops-deploy-immutability-idempotency        | 原范围已完成                   | [manifest](./archive/2026-08/tasks/ops-deploy-immutability-idempotency/manifest.json)        | [summary](./archive/2026-08/evidence/ops-deploy-immutability-idempotency/summary.md)        | 新业务 gap 由首期计划跟踪 |
| ops-service-instance-foundation            | 原范围已完成                   | [manifest](./archive/2026-08/tasks/ops-service-instance-foundation/manifest.json)            | [summary](./archive/2026-08/evidence/ops-service-instance-foundation/summary.md)            | 新业务 gap 由首期计划跟踪 |
| ops-resource-service-state-machines        | 原范围已完成                   | [manifest](./archive/2026-08/tasks/ops-resource-service-state-machines/manifest.json)        | [summary](./archive/2026-08/evidence/ops-resource-service-state-machines/summary.md)        | 新业务 gap 由首期计划跟踪 |
| ops-k8s-release-reliability                | 原范围已完成                   | [manifest](./archive/2026-08/tasks/ops-k8s-release-reliability/manifest.json)                | [summary](./archive/2026-08/evidence/ops-k8s-release-reliability/summary.md)                | 新业务 gap 由首期计划跟踪 |
| ops-architecture-quality-gates             | 原范围已完成                   | [manifest](./archive/2026-08/tasks/ops-architecture-quality-gates/manifest.json)             | [summary](./archive/2026-08/evidence/ops-architecture-quality-gates/summary.md)             | 新业务 gap 由首期计划跟踪 |
| 2026-10-31-frontend-follow-redirects-audit | 原范围已完成                   | [manifest](./archive/2026-10/tasks/2026-10-31-frontend-follow-redirects-audit/manifest.json) | [summary](./archive/2026-10/evidence/2026-10-31-frontend-follow-redirects-audit/summary.md) | 新业务 gap 由首期计划跟踪 |
| 2026-11-07-base-v0186-consumption          | 原范围已完成                   | [manifest](./archive/2026-11/tasks/2026-11-07-base-v0186-consumption/manifest.json)          | [summary](./archive/2026-11/evidence/2026-11-07-base-v0186-consumption/summary.md)          | 新业务 gap 由首期计划跟踪 |
| 2026-11-13-base-v0187-consumption          | 原范围完成；PR183 合并事实补记 | [manifest](./archive/2026-11/tasks/2026-11-13-base-v0187-consumption/manifest.json)          | [summary](./archive/2026-11/evidence/2026-11-13-base-v0187-consumption/summary.md)          | 新业务 gap 由首期计划跟踪 |

未归档：新菜单验收、综合交付/生产/运行环境缺口、缺摘要日志目录与未核实关闭的旧任务。见当前 STATUS 与 archive-inventory retained；不按 readout 标题或旧 completed 字段批量抹除未完成事实。

归档兼容说明：八月 8 个子包的旧文档文件名含日期，但机器 taskId 不含日期；文档地址保留为 Superseded 历史入口，机器任务/evidence 按真实 taskId 归档。原始 evidence 不复制、不改写。PR183 的本轮 GitHub 复查因认证/限流失败，合并事实由上一轮 hosted 观察及本轮本地 commit ancestry 支撑；不冒充新的 GitHub raw 证据。
