---
title: 报表中心数据模型设计
doc_type: Design
layer: platform
status: Draft
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
  - docs/contracts/TENANT_CONTRACT_V1.md
updated_at: 2026-11-04
---

# 报表中心数据模型设计

> 未实现声明：本文为设计稿，仓库内尚无对应实现；在实现落地前状态保持 Draft。
> Not implemented: this is a design draft with no corresponding implementation in the repository; it stays Draft until implemented.

主设计：[REPORT_CENTER_DESIGN.md](./REPORT_CENTER_DESIGN.md)（英文 [REPORT_CENTER_DESIGN.en.md](./REPORT_CENTER_DESIGN.en.md)）
接口契约：[REPORT_CENTER_API_CONTRACT.md](./REPORT_CENTER_API_CONTRACT.md)
测试计划：[REPORT_CENTER_TEST_PLAN.md](./REPORT_CENTER_TEST_PLAN.md)

---

本文定义报表中心的表结构、索引、迁移编号、down 形状与保留/清理策略。
基础规范沿用 [DATABASE.md](./DATABASE.md)：InnoDB、`utf8mb4_general_ci`、
`bigint unsigned NOT NULL AUTO_INCREMENT` 主键、GORM 默认软删列 `deleted_at datetime(3) NULL`、**严禁物理外键**（§4）。

## 1. 表清单与租户分类登记

| 表 | 用途 | 租户分类（矩阵 §1 四分类） | `tenant_id` 列 |
| :--- | :--- | :--- | :--- |
| `report_dataset` | 取数口径注册 | `platform-global` | 无（见 §2.1） |
| `report_template` | 报表定义 | `tenant-owned` | 有 |
| `report_template_version` | 不可变版本快照 | `tenant-owned`（跟随模板） | 有 |
| `report_run` | 一次渲染执行记录 | `derived`（跟随来源） | 有 |
| `report_artifact` | 渲染产物元数据 | `derived`（跟随来源） | 有 |

登记义务：以上分类必须在实现 PR 同步写入
[TENANT_RESOURCE_SCOPE_MATRIX.md](./TENANT_RESOURCE_SCOPE_MATRIX.md) §2 与 §3（唯一键登记）。
本设计包不修改该矩阵（主设计 §16 OD-6）。

## 2. 建表形状（迁移正文的目标形态）

以下 DDL 是设计目标形状，实现时进入 `backend/pkg/database/migrations/`（编号见 §4）。
每一张表都带 `created_at / updated_at / deleted_at`（`DATETIME(3) DEFAULT NULL`）与
`idx_<table>_deleted_at` 索引，形状与 `backend/pkg/database/migrations/000001_init_schema.up.sql`
（如第 117/120 行的 `system_user`）一致。**严禁物理外键**（[DATABASE.md](./DATABASE.md) §4）。

### 2.1 `report_dataset`

```sql
CREATE TABLE IF NOT EXISTS `report_dataset` (
  `id`                BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `code`              VARCHAR(64)  NOT NULL,
  `name`              VARCHAR(128) NOT NULL DEFAULT '',
  `description`       VARCHAR(512) NOT NULL DEFAULT '',
  `source_domain`     VARCHAR(32)  NOT NULL DEFAULT '',
  `provider_key`      VARCHAR(64)  NOT NULL DEFAULT '',
  `param_schema_json` TEXT         NULL,
  `scope_binding`     VARCHAR(16)  NOT NULL DEFAULT 'none',
  `row_limit`         INT          NOT NULL DEFAULT 10000,
  `timeout_ms`        INT          NOT NULL DEFAULT 5000,
  `cacheable`         TINYINT(1)   NOT NULL DEFAULT 0,
  `freshness_seconds` INT          NOT NULL DEFAULT 0,
  `status`            VARCHAR(16)  NOT NULL DEFAULT 'active',
  `created_at`        DATETIME(3)  DEFAULT NULL,
  `updated_at`        DATETIME(3)  DEFAULT NULL,
  `deleted_at`        DATETIME(3)  DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE INDEX `uk_report_dataset_code` (`code`),
  INDEX `idx_report_dataset_source_domain` (`source_domain`),
  INDEX `idx_report_dataset_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

设计说明：

1. **没有 `tenant_id`**：与 `tenant_plans` 同理——本模块的租户收窄由各 service 显式 `WithTenantScope`
   施加而非全局插件，给平台注册表加 `tenant_id` 会让平台控制台的读变成空集
   （理由原文见 `backend/pkg/database/migrations/000030_tenant_plan_definitions.up.sql:11-17`）。
2. `scope_binding` 只允许 `none` / `dept` / `self`，取值与
   `backend/pkg/database/scope.go` 的行级谓词一一对应（主设计 §9.2）。
3. `row_limit` 默认 10000 = `impexp.MaxExportRows()`（`backend/pkg/impexp/export_limits.go:9`）；
   服务层写入时拒绝大于该值（主设计 §8.4）。
4. `param_schema_json` 是白名单，不是可执行片段；服务层解析失败即拒绝注册。

### 2.2 `report_template`

```sql
CREATE TABLE IF NOT EXISTS `report_template` (
  `id`                 BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `tenant_id`          BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `code`               VARCHAR(64)  NOT NULL,
  `report_type`        VARCHAR(24)  NOT NULL DEFAULT 'list',
  `name`               VARCHAR(128) NOT NULL DEFAULT '',
  `description`        VARCHAR(512) NOT NULL DEFAULT '',
  `category`           VARCHAR(32)  NOT NULL DEFAULT '',
  `dataset_code`       VARCHAR(64)  NOT NULL DEFAULT '',
  `visibility`         VARCHAR(16)  NOT NULL DEFAULT 'platform',
  `owner_user_id`      BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `supported_formats`  VARCHAR(64)  NOT NULL DEFAULT 'csv',
  `default_param_json` TEXT         NULL,
  `permission_key`     VARCHAR(64)  NOT NULL DEFAULT '',
  `sensitive`          TINYINT(1)   NOT NULL DEFAULT 0,
  `status`             VARCHAR(16)  NOT NULL DEFAULT 'draft',
  `current_version_id` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `created_by`         BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `updated_by`         BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `created_at`         DATETIME(3)  DEFAULT NULL,
  `updated_at`         DATETIME(3)  DEFAULT NULL,
  `deleted_at`         DATETIME(3)  DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE INDEX `uk_report_template_tenant_code` (`tenant_id`, `code`),
  INDEX `idx_report_template_tenant_visibility_status` (`tenant_id`, `visibility`, `status`),
  INDEX `idx_report_template_dataset_code` (`dataset_code`),
  INDEX `idx_report_template_owner_user` (`owner_user_id`),
  INDEX `idx_report_template_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

设计说明：

1. 唯一键是 `(tenant_id, code)`，与 `system_role_data_scope`（`uk_role_data_scope_tenant_role`，
   `backend/internal/middleware/data_scope_middleware.go:47-48`）、`system_user`
   （`uk_system_user_tenant_username`，迁移 000024，登记见
   [TENANT_RESOURCE_SCOPE_MATRIX.md](./TENANT_RESOURCE_SCOPE_MATRIX.md) §3）同一形状。
2. `visibility` 与 `permission_key` 是两个正交维度：前者控制清单可见，后者控制可执行动作
   （主设计 §6.1）。
3. `created_by` 存在使 `scope_binding = self` 的报表能直接复用
   `backend/pkg/database/scope.go:70-96` 的 `created_by / create_by` 识别逻辑。
4. `sensitive = 1` 的模板其导出动作走 `SecureActionMiddleware`（主设计 §9.4）。

### 2.3 `report_template_version`

```sql
CREATE TABLE IF NOT EXISTS `report_template_version` (
  `id`               BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `tenant_id`        BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `template_id`      BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `version_no`       INT          NOT NULL DEFAULT 1,
  `spec_json`        TEXT         NULL,
  `dataset_snapshot` TEXT         NULL,
  `status`           VARCHAR(16)  NOT NULL DEFAULT 'draft',
  `published_by`     BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `published_at`     DATETIME(3)  DEFAULT NULL,
  `created_at`       DATETIME(3)  DEFAULT NULL,
  `updated_at`       DATETIME(3)  DEFAULT NULL,
  `deleted_at`       DATETIME(3)  DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE INDEX `uk_report_template_version` (`template_id`, `version_no`),
  INDEX `idx_report_version_tenant_status` (`tenant_id`, `status`),
  INDEX `idx_report_version_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

设计说明：

1. 唯一键**不含 tenant_id**：`template_id` 已经租户唯一，再加一列只会给跨租户写入留一个不一致窗口。
2. `published` 行由服务层拒绝 UPDATE（只允许 INSERT 新版本、指针前移、按 §5 软删）。
3. 版本行数按模板受控增长，第一阶段不做版本保留期清理；模板软删时同事务软删其版本行。

### 2.4 `report_run`

```sql
CREATE TABLE IF NOT EXISTS `report_run` (
  `id`                    BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `tenant_id`             BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `template_id`           BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `template_version_id`   BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `dataset_code`          VARCHAR(64)  NOT NULL DEFAULT '',
  `trigger_source`        VARCHAR(16)  NOT NULL DEFAULT 'manual',
  `schedule_execution_id` BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `requester_user_id`     BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `format`                VARCHAR(16)  NOT NULL DEFAULT 'csv',
  `param_digest`          VARCHAR(64)  NOT NULL DEFAULT '',
  `scope_fingerprint`     VARCHAR(64)  NOT NULL DEFAULT '',
  `status`                VARCHAR(16)  NOT NULL DEFAULT 'pending',
  `row_count`             INT          NOT NULL DEFAULT 0,
  `truncated`             TINYINT(1)   NOT NULL DEFAULT 0,
  `duration_ms`           INT          NOT NULL DEFAULT 0,
  `error_key`             VARCHAR(64)  NOT NULL DEFAULT '',
  `started_at`            DATETIME(3)  DEFAULT NULL,
  `finished_at`           DATETIME(3)  DEFAULT NULL,
  `created_at`            DATETIME(3)  DEFAULT NULL,
  `updated_at`            DATETIME(3)  DEFAULT NULL,
  `deleted_at`            DATETIME(3)  DEFAULT NULL,
  PRIMARY KEY (`id`),
  INDEX `idx_report_run_tenant_time` (`tenant_id`, `created_at`),
  INDEX `idx_report_run_template_time` (`template_id`, `created_at`),
  INDEX `idx_report_run_requester_time` (`requester_user_id`, `created_at`),
  INDEX `idx_report_run_status` (`status`),
  INDEX `idx_report_run_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

设计说明：

1. 无唯一键：run 是事件记录，重复渲染必须留下两条记录（审计要求），因此只有查询索引。
2. `param_digest` / `scope_fingerprint` 存哈希（主设计 §13），既能让审计回答"谁用什么条件跑了什么"，
   又不把可能含敏感值的大参数原文塞进行里（与 `system_log_oper.oper_param` 的脱敏纪律同源，
   [DATABASE.md](./DATABASE.md) §3.2）。
3. `scope_fingerprint` 的组成必须是 `mode` + 归一化 `deptIDs` + `userID`（主设计 §10 约束 1）。
4. `schedule_execution_id` 指向 `schedule_execution.id`（调度侧表），逻辑关联、无物理外键。

### 2.5 `report_artifact`

```sql
CREATE TABLE IF NOT EXISTS `report_artifact` (
  `id`               BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `tenant_id`        BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `run_id`           BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `storage_driver`   VARCHAR(16)  NOT NULL DEFAULT 'local',
  `object_key`       VARCHAR(255) NOT NULL DEFAULT '',
  `filename`         VARCHAR(128) NOT NULL DEFAULT '',
  `content_type`     VARCHAR(64)  NOT NULL DEFAULT '',
  `size_bytes`       BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `row_count`        INT          NOT NULL DEFAULT 0,
  `scope_fingerprint` VARCHAR(64) NOT NULL DEFAULT '',
  `status`           VARCHAR(16)  NOT NULL DEFAULT 'ready',
  `expires_at`       DATETIME(3)  DEFAULT NULL,
  `created_at`       DATETIME(3)  DEFAULT NULL,
  `updated_at`       DATETIME(3)  DEFAULT NULL,
  `deleted_at`       DATETIME(3)  DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE INDEX `uk_report_artifact_object_key` (`object_key`),
  INDEX `idx_report_artifact_tenant_expires` (`tenant_id`, `expires_at`),
  INDEX `idx_report_artifact_run` (`run_id`),
  INDEX `idx_report_artifact_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

设计说明：

1. `storage_driver` / `object_key` 语义直接对齐 `backend/pkg/upload/service.go`
   （`upload.storage_driver` ∈ `local` | `s3`；key = `scope/日期/uuid.ext`）。
2. `object_key` 的第一段（scope）必须含租户维度，避免串租户命中
   （[TENANT_RESOURCE_SCOPE_MATRIX.md](./TENANT_RESOURCE_SCOPE_MATRIX.md) §2 导出文件条目）。
3. `scope_fingerprint` 让"下载时重新校验请求者行级范围"（主设计 §16 OD-5 的 A 案）有落点。
4. `expires_at` 在写入时按 §6 的保留期计算，清理任务只看这一列，避免每次扫全表。

## 3. 索引与查询匹配

| 查询（API 契约 §4） | 命中索引 |
| :--- | :--- |
| 列表页按租户 + 可见性 + 状态筛模板 | `idx_report_template_tenant_visibility_status` |
| 按 code 取模板 | `uk_report_template_tenant_code` |
| 模板版本列表 | `uk_report_template_version`（前缀 template_id） |
| 运行历史分页 | `idx_report_run_tenant_time` / `idx_report_run_template_time` |
| "我的导出" | `idx_report_run_requester_time` |
| artifact 过期扫描 | `idx_report_artifact_tenant_expires` |

不做索引的情形：`status` 单独列不加索引（[DATABASE.md](./DATABASE.md) §4「区分度低」），
只在复合索引里出现。

## 4. 迁移编号与文件

当前 `backend/pkg/database/migrations/` 最新编号是 **000031**
（`000031_drop_permission_role_data_scope_policy.up.sql` / `.down.sql`）。
报表中心占用：

| 编号 | 文件 | 内容 |
| :--- | :--- | :--- |
| 000032 | `000032_report_center_core_tables.up.sql` | 建 §2 的 5 张表（全部 `CREATE TABLE IF NOT EXISTS`） |
| 000032 | `000032_report_center_core_tables.down.sql` | 见 §7 |
| 000033 | `000033_report_center_setting_seed.up.sql` | 幂等 seed §6 的 `system_setting` 键 |
| 000033 | `000033_report_center_setting_seed.down.sql` | 删除这些键行 |

编号登记规则：**同一 PR 内不得复用编号**；若与其他并行设计包冲突，实现时以先到者为准并重编号，
冲突结果写进 PR 说明（本包与调度中心包按 000032/000033、000034/000035 顺序分配）。

seed 写法必须沿用 000030 的守卫式插入（不做 UPDATE 覆盖），原文理由见
`backend/pkg/database/migrations/000030_tenant_plan_definitions.up.sql:22-28`：

```sql
INSERT INTO `system_setting` (`setting_key`, `setting_value`, ...)
SELECT 'report.artifact_retention_days', '7', ...
WHERE NOT EXISTS (SELECT 1 FROM `system_setting` WHERE `setting_key` = 'report.artifact_retention_days');
```

其他必须遵守的迁移事实：

- `currentRuntimeSchemaVersion` 与兼容 marker 机制会把"看起来已是当前 schema"的库直接基线化
  （`backend/pkg/database/migrate.go:39`、`:157-202`）。新增表若属于运行期必需，**不需要**新增 marker；
  但绝不能依赖 `PANTHEON_AUTO_MIGRATE` 的 GORM AutoMigrate 作为第二份 schema 权威
  （教训原文见 `backend/internal/middleware/data_scope_middleware.go:73-85`）。
- 迁移由 golang-migrate 的 MySQL driver 执行，该 driver 用 `SELECT GET_LOCK(?, 10)` 做跨 Pod 串行化，
  多副本滚动更新时只有先拿到锁的实例执行迁移（事实与处置见
  [MIGRATION_000031_PRODUCTION_DROP.md](../runbooks/MIGRATION_000031_PRODUCTION_DROP.md) §1.2、
  `docs/reviews/PANTHEON_BASE_GLOBAL_AUDIT_2026-10-31.md:1070`）。
- 生产环境的破坏性变更必须走 `preflight` 脚本先行的流程，先例是
  `backend/pkg/database/migrations/preflight/000031_preflight.sql`。

## 5. 软删除与唯一键释放（仓库既有规则）

MySQL 的唯一索引**不包含** `deleted_at`：软删的行仍占用唯一键，所以删除时必须先把标识改名释放。
仓库已有四处同一实现，报表中心沿用同一模式，不新造机制：

| 既有实现 | 位置 | 前缀常量 |
| :--- | :--- | :--- |
| 岗位 `post_code` | `backend/modules/system/org/post/post_service.go:40`、`:806` | `__deleted_post_` |
| 字典类型/项 | `backend/modules/system/config/dict/dict_service.go:1483`、`:1501` | `__deleted_dict_type_` 等 |
| 租户 code | `backend/modules/system/iam/tenant/tenant_service.go:847` | `__deleted_tenant_` |
| 用户名 | [DATABASE.md](./DATABASE.md) §3.3.2 | 归档为内部保留值 |

报表侧规则：

1. 删除 `report_template` 时，同事务把 `code` 改为 `__deleted_report_<id>`（冲突时追加 `_<unixnano>`，
   重试上限 5 次，最终失败返回冲突错误键——完全沿用
   `allocateDeletedPostCode` 的循环与错误语义）。
2. `report_dataset.code` 同理（若允许删除注册项）。
3. `report_artifact.object_key` 是系统生成、不复用，因此不需要改名；删除即软删 + 后台清理对象。
4. `report_run` 不做改名（无唯一键）。
5. 唯一索引形状统一：`uk_*` 只覆盖业务标识列（+ `tenant_id`），**永不**把 `deleted_at` 纳入索引列。

## 6. 保留期与清理（retention / purge）

| 配置键（`system_setting`） | 默认 | 作用 |
| :--- | :--- | :--- |
| `report.artifact_retention_days` | 7 | `report_artifact.expires_at = created_at + N` |
| `report.run_retention_days` | 90 | `report_run` 行的保留期 |
| `report.sync_row_cap` | 10000 | 只用于展示，实际天花板仍以 `impexp.MaxExportRows()` 为准（不新增第二权威） |
| `report.default_timeout_ms` | 5000 | dataset 未声明 timeout 时的回退 |

清理执行方式（必须照既有形状，不在请求路径里做全表 DELETE）：

1. 服务实现 `RegisterMaintenanceTasks(reg *maintenance.Registry)`，注册两个任务：
   `report.artifact_retention` 与 `report.run_retention`，`Interval = maintenance.DefaultInterval`
   （15 分钟，`backend/pkg/maintenance/defaults.go`）。
   先例：`backend/modules/system/audit/audit_service.go:317`。
2. 注册表自带节流与**重叠保护**（`OutcomeOverlap`），因此任务实现只需描述"怎么清"，不描述"多久跑一次"。
3. 后台 runner 由 `PANTHEON_MAINTENANCE_ENABLED` / `PANTHEON_MAINTENANCE_INTERVAL_SECONDS` 控制
   （`backend/cmd/server/main.go` 的 `runServer`）。
   **运维事实必须写进验收**：该开关关闭时产物不会自动过期，只会在下载时被判定过期。
4. 保留期天数读取 `system_setting`，读不到时回退默认值的实现方式与
   `backend/modules/system/audit/audit_service.go:325-341`（`getRetentionDaysFromSetting`）一致。
5. 清理顺序：先删对象存储（尽力而为，失败只告警不阻塞），再软删 `report_artifact`，最后按时间删 `report_run`。
6. 删除 `report_run` 用带 `LIMIT` 的分批循环（每批建议 1000 行），避免长事务锁表；
   这是本设计新增的约束，理由是 run 表是唯一会高速膨胀的报表表。

## 7. down 形状

| 迁移 | down 行为 | 数据承诺 |
| :--- | :--- | :--- |
| 000032 | `DROP TABLE IF EXISTS` 五张表 | **销毁数据**：回滚只回结构，run/artifact 记录与对象不可恢复 |
| 000033 | `DELETE FROM system_setting WHERE setting_key IN (4 个键)` | 只回收 seed 行；若运维已改过值，回滚会丢掉改动，必须在 runbook 提示 |

down 的两条硬要求：

1. down 必须重建"该迁移之前那一版真实存在的形状"，不得多加索引或列。
   该纪律的反例与断言方式见 `backend/pkg/database/migrate_down_shape_test.go`
   （用 `SHOW CREATE TABLE` 逐字节比对 "直接 up 到 N" 与 "up 到 N+1 再 down 到 N"）。
   报表侧对应测试：`000032/000033` up→down→up 后形状一致，且 down 不残留索引。
2. down 文件头部必须写清 DATA PROMISE（是否毁数据），沿用
   `backend/pkg/database/migrations/000031_drop_permission_role_data_scope_policy.up.sql` / `.down.sql`
   的写法（up 声明 `DATA PROMISE: this up migration destroys the table and any rows in it`，
   down 声明只重建 shape、`DATA IS NOT RESTORED`）。

## 8. 落库前的门禁清单（实现 PR 必须过）

| 检查 | 命令 |
| :--- | :--- |
| 迁移 up/down 形状 | `cd backend && go test ./pkg/database/`（既有夹具 `backend/pkg/testmysql`；形状断言的写法参照 `backend/pkg/database/migrate_down_shape_test.go`、`migrate_down_test.go`、`migrate_preflight_test.go`） |
| 迁移 preflight（若含破坏性变更） | 参照 `backend/pkg/database/migrations/preflight/000031_preflight.sql` 新增对应文件 |
| 后端单测 | `cd backend && go test ./...`（CI job `unit-tests` / `backend-tests`） |
| Go lint | `golangci-lint`（CI job `go-lint`） |
| 权限键形状 | `npm run check:harness-permission-contract` |
| DTO / 响应信封 | `npm run check:backend-dto-contract`、`npm run check:backend-response-contract` |
| 文档 | `npm run check:docs-frontmatter`、`npm run check:harness-docs` |

## 9. 开放决策 / Open decisions

| 编号 | 决策 | 选项 | 决策人 |
| :--- | :--- | :--- | :--- |
| DB-OD-1 | 新表是否显式写 `COLLATE=utf8mb4_general_ci`（[DATABASE.md](./DATABASE.md) §1 要求）还是沿用近期迁移的 `DEFAULT CHARSET=utf8mb4`（000030 实际写法） | A：按 DATABASE.md 显式写 COLLATE；B：与近期迁移保持一致 | 维护者 |
| DB-OD-2 | `report_run` 是否按月分区 | A：不分区，靠保留期清理；B：`PARTITION BY RANGE (created_at)` | 维护者（取决于真实写入量） |
| DB-OD-3 | artifact 是否需要保留原始 `param_json` 以便复现 | A：只留 digest（当前设计）；B：额外留一列脱敏后的原文 | 维护者 |
| DB-OD-4 | 版本表是否需要"版本保留期"清理 | A：不清理（当前设计）；B：保留最近 N 版 | 维护者 |
