---
title: 调度中心数据库设计
doc_type: Design
layer: platform
status: Draft
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
  - docs/contracts/SYSTEM_CONFIG_CONTRACT.md
  - docs/contracts/TENANT_CONTRACT_V1.md
updated_at: 2026-11-04
---

# 调度中心数据库设计

> 未实现声明：本文为设计稿，仓库内尚无对应实现；在实现落地前状态保持 Draft。
> Not implemented: this is a design draft with no corresponding implementation in the repository; it stays Draft until implemented.

主设计：[SCHEDULER_CENTER_DESIGN.md](./SCHEDULER_CENTER_DESIGN.md)（英文 [SCHEDULER_CENTER_DESIGN.en.md](./SCHEDULER_CENTER_DESIGN.en.md)）
接口契约：SCHEDULER_CENTER_API_CONTRACT.md（planned/unavailable：`./SCHEDULER_CENTER_API_CONTRACT.md`；无当前交付承诺）
测试计划：SCHEDULER_CENTER_TEST_PLAN.md（planned/unavailable：`./SCHEDULER_CENTER_TEST_PLAN.md`；无当前交付承诺）

---

本文定义调度中心的表结构、索引、迁移编号、down 形状与保留/清理策略。
基础规范沿用 [DATABASE.md](./DATABASE.md)：InnoDB、`utf8mb4_general_ci`、
`bigint unsigned NOT NULL AUTO_INCREMENT` 主键、GORM 默认软删列 `deleted_at datetime(3) NULL`、
**严禁物理外键**（§4）、低基数状态列 + 索引（§4）。

## 1. 表清单与租户分类登记

| 表 | 用途 | 租户分类（矩阵 §1 四分类） | `tenant_id` 列 |
| :--- | :--- | :--- | :--- |
| `schedule_job` | 任务定义与调度状态 | `platform-global` | 无（见 §2.1，理由同 000030 的平台全局判断） |
| `schedule_lease` | 跨实例 single-flight 租约 | `platform-global` | 无 |
| `schedule_execution` | 一次逻辑执行（含最终结局） | `platform-global` | 无 |
| `schedule_execution_attempt` | 每次尝试（含重试与 skipped/overlap） | `platform-global` | 无 |

登记义务：以上分类必须在实现 PR 同步写入
[TENANT_RESOURCE_SCOPE_MATRIX.md](./TENANT_RESOURCE_SCOPE_MATRIX.md) §2 与 §3（唯一键登记）。
本设计包不修改该矩阵（主设计 §17 OD-S-5）。

为什么不放 `tenant_id`：任务清单是平台级运维资源，调度表达式与租约跨租户共享；
租户语义只存在于**处理器内部的取数**（主设计 §12.1 例外条款）。
若未来改为租户可自定义任务（OD-S-5 B 案），这四张表都要重建键，属于破坏性变更。

## 2. 建表形状（迁移正文的目标形态）

以下 DDL 是设计目标形状，实现时进入 `backend/pkg/database/migrations/`（编号见 §4）。
`schedule_job / schedule_lease` 带 `created_at / updated_at / deleted_at`；
两张执行记录表是**只追加的事实记录**，同样带 GORM 默认三列以保持全库一致形状
（`backend/pkg/database/migrations/000001_init_schema.up.sql:117`、`:120` 的 `system_user` 写法），
但业务上不执行软删，清理走 §6 的物理保留期。**严禁物理外键**（[DATABASE.md](./DATABASE.md) §4）。

### 2.1 `schedule_job`

```sql
CREATE TABLE IF NOT EXISTS `schedule_job` (
  `id`                    BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `job_key`               VARCHAR(96)  NOT NULL,
  `name_key`              VARCHAR(128) NOT NULL DEFAULT '',
  `description_key`       VARCHAR(255) NOT NULL DEFAULT '',
  `job_type`              VARCHAR(32)  NOT NULL DEFAULT 'system',
  `source_domain`         VARCHAR(32)  NOT NULL DEFAULT '',
  `owner`                 VARCHAR(64)  NOT NULL DEFAULT '',
  `handler_key`           VARCHAR(96)  NOT NULL DEFAULT '',
  `schedule_type`         VARCHAR(16)  NOT NULL DEFAULT 'manual',
  `schedule_expr`         VARCHAR(128) NOT NULL DEFAULT '',
  `schedule_tz`           VARCHAR(64)  NOT NULL DEFAULT 'UTC',
  `enabled`               TINYINT(1)   NOT NULL DEFAULT 0,
  `paused`                TINYINT(1)   NOT NULL DEFAULT 0,
  `concurrency_policy`    VARCHAR(16)  NOT NULL DEFAULT 'forbid',
  `misfire_policy`        VARCHAR(16)  NOT NULL DEFAULT 'skip',
  `timeout_ms`            INT          NOT NULL DEFAULT 60000,
  `retry_max`             TINYINT      NOT NULL DEFAULT 0,
  `retry_backoff_seconds` INT          NOT NULL DEFAULT 60,
  `param_json`            TEXT         NULL,
  `next_fire_at`          DATETIME(3)  DEFAULT NULL,
  `prev_fire_at`          DATETIME(3)  DEFAULT NULL,
  `last_run_at`           DATETIME(3)  DEFAULT NULL,
  `last_run_status`       VARCHAR(16)  NOT NULL DEFAULT '',
  `last_run_summary`      VARCHAR(255) NOT NULL DEFAULT '',
  `last_duration_ms`      INT          NOT NULL DEFAULT 0,
  `source_route_key`      VARCHAR(96)  NOT NULL DEFAULT '',
  `created_by`            BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `updated_by`            BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `created_at`            DATETIME(3)  DEFAULT NULL,
  `updated_at`            DATETIME(3)  DEFAULT NULL,
  `deleted_at`            DATETIME(3)  DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE INDEX `uk_schedule_job_key` (`job_key`),
  INDEX `idx_schedule_job_due` (`enabled`, `paused`, `next_fire_at`),
  INDEX `idx_schedule_job_source_domain` (`source_domain`),
  INDEX `idx_schedule_job_schedule_type` (`schedule_type`),
  INDEX `idx_schedule_job_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

设计说明：

1. `uk_schedule_job_key` **不含 `deleted_at`**（§5）：任务删除后同名可复用，与既有做法一致
   （`000001_init_schema.up.sql:120` 的 `idx_system_user_username (username)` 同样不含 `deleted_at`）。
2. `idx_schedule_job_due` 直接服务 tick 查询（§3），顺序为 `enabled, paused, next_fire_at`：
   前两个是等值条件、最后一个是范围条件，范围列必须放最后才能吃到索引。
3. `next_fire_at` / `prev_fire_at` / `last_run_at` 全部是 **UTC 瞬时**（主设计 §6.2 约束 2），
   列名不写 `local`，避免误读。
4. `schedule_tz` 默认 `'UTC'` 是**列默认值**，不是运行默认；运行默认来自
   `schedule.default_timezone` 设置（主设计 §12.2、OD-S-6），落库时已解析为具体 IANA 名。
5. `name_key` / `description_key` 存 i18n key（[DESIGN.md](../../DESIGN.md) §6），
   `last_run_summary` 例外：它是处理器产出的**已归一化摘要**，按 §6 的约束只含错误键与计数，
   因此可以是自然语言片段——这条例外必须写进实现注释，避免被误读为"允许存文案"。
6. `param_json` 是任务级默认参数；执行时的实际参数快照写在
   `schedule_execution_attempt.param_digest`（§2.4），不回写本行。

### 2.2 `schedule_lease`

```sql
CREATE TABLE IF NOT EXISTS `schedule_lease` (
  `id`              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `job_id`          BIGINT UNSIGNED NOT NULL,
  `job_key`         VARCHAR(96)  NOT NULL DEFAULT '',
  `holder`          VARCHAR(64)  NOT NULL DEFAULT '',
  `lease_expire_at` DATETIME(3)  DEFAULT NULL,
  `lease_version`   BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `renewed_at`      DATETIME(3)  DEFAULT NULL,
  `created_at`      DATETIME(3)  DEFAULT NULL,
  `updated_at`      DATETIME(3)  DEFAULT NULL,
  `deleted_at`      DATETIME(3)  DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE INDEX `uk_schedule_lease_job` (`job_id`),
  INDEX `idx_schedule_lease_expire` (`lease_expire_at`),
  INDEX `idx_schedule_lease_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

设计说明：

1. **一任务一行**，由 `uk_schedule_lease_job` 保证；行由任务创建时同事务建出，
   缺失时领取协议必须先 `INSERT ... ON DUPLICATE KEY UPDATE` 补行再锁。
2. 该表是**唯一的跨实例互斥来源**（主设计 §7.2）。`holder` 是 `pid-unixnano` 形态的实例标识，
   与既有实现一致（`backend/pkg/database/casbin_watcher.go:83`）。
3. `lease_version` 是 fencing token，写回执行结果时必须比较（主设计 §7.2 步骤 3）。
4. `job_key` 是冗余可读列，便于运维直接看租约表定位任务；权威关联仍是 `job_id`。
5. `idx_schedule_lease_expire` 服务孤儿对账（查 `lease_expire_at < now()`），不是热路径索引。
6. 本表**不参与软删语义**：租约是状态机不是业务对象；任务删除时随任务一并物理清理（§6.3）。

### 2.3 `schedule_execution`

```sql
CREATE TABLE IF NOT EXISTS `schedule_execution` (
  `id`             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `job_id`         BIGINT UNSIGNED NOT NULL,
  `job_key`        VARCHAR(96)  NOT NULL DEFAULT '',
  `fire_instance`  VARCHAR(128) NOT NULL,
  `trigger_source` VARCHAR(16)  NOT NULL DEFAULT 'cron',
  `schedule_at`    DATETIME(3)  DEFAULT NULL,
  `status`         VARCHAR(16)  NOT NULL DEFAULT 'pending',
  `final_status`   VARCHAR(16)  NOT NULL DEFAULT '',
  `attempt_count`  INT          NOT NULL DEFAULT 0,
  `started_at`     DATETIME(3)  DEFAULT NULL,
  `finished_at`    DATETIME(3)  DEFAULT NULL,
  `duration_ms`    INT          NOT NULL DEFAULT 0,
  `error_key`      VARCHAR(64)  NOT NULL DEFAULT '',
  `result_summary` VARCHAR(255) NOT NULL DEFAULT '',
  `operator_id`    BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `holder`         VARCHAR(64)  NOT NULL DEFAULT '',
  `lease_version`  BIGINT UNSIGNED NOT NULL DEFAULT 0,
  `created_at`     DATETIME(3)  DEFAULT NULL,
  `updated_at`     DATETIME(3)  DEFAULT NULL,
  `deleted_at`     DATETIME(3)  DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE INDEX `uk_schedule_execution_fire` (`job_id`, `fire_instance`),
  INDEX `idx_schedule_execution_job_time` (`job_id`, `started_at`),
  INDEX `idx_schedule_execution_status_time` (`status`, `created_at`),
  INDEX `idx_schedule_execution_created_at` (`created_at`),
  INDEX `idx_schedule_execution_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

设计说明：

1. `uk_schedule_execution_fire (job_id, fire_instance)` 是主设计 §9.3 的幂等落点：
   重复投递命中唯一键即返回既有行。`fire_instance` 对 cron 任务取计划触发时间的 UTC 串，
   对手动任务取 `manual:<idempotencyKey>`，长度上限 128 由 `job_key` 96 + 前缀 20 + 余量决定。
2. 唯一键**不含 `deleted_at`**（§5、主设计 §9.3）。
3. 重试不新建执行行（否则幂等键失去意义），而是本行 `attempt_count + 1` + 追加 attempt 行（§2.4）。
   `status` 是进行时状态，`final_status` 只允许被写一次（fencing 校验后），便于 UI 区分"在跑"与"已定论"。
4. `operator_id` 只在 `trigger_source = manual` 时有值，用于回答"谁手动跑了它"（主设计 §11 约束）。
5. `holder` + `lease_version` 冗余记录"这次由哪个实例、第几代租约执行"，是孤儿对账的判据
   （主设计 §7.3），不要用"所有 `running` 都视为孤儿"这种粗暴规则。

### 2.4 `schedule_execution_attempt`

```sql
CREATE TABLE IF NOT EXISTS `schedule_execution_attempt` (
  `id`            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT,
  `execution_id`  BIGINT UNSIGNED NOT NULL,
  `job_id`        BIGINT UNSIGNED NOT NULL,
  `attempt_no`    INT          NOT NULL DEFAULT 1,
  `outcome`       VARCHAR(16)  NOT NULL DEFAULT '',
  `started_at`    DATETIME(3)  DEFAULT NULL,
  `finished_at`   DATETIME(3)  DEFAULT NULL,
  `duration_ms`   INT          NOT NULL DEFAULT 0,
  `error_key`     VARCHAR(64)  NOT NULL DEFAULT '',
  `param_digest`  VARCHAR(64)  NOT NULL DEFAULT '',
  `holder`        VARCHAR(64)  NOT NULL DEFAULT '',
  `created_at`    DATETIME(3)  DEFAULT NULL,
  `updated_at`    DATETIME(3)  DEFAULT NULL,
  `deleted_at`    DATETIME(3)  DEFAULT NULL,
  PRIMARY KEY (`id`),
  UNIQUE INDEX `uk_schedule_attempt_no` (`execution_id`, `attempt_no`),
  INDEX `idx_schedule_attempt_job_time` (`job_id`, `created_at`),
  INDEX `idx_schedule_attempt_created_at` (`created_at`),
  INDEX `idx_schedule_attempt_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

设计说明：

1. `outcome` 取值直接复用既有 maintenance 结局词表：`succeeded / failed / skipped / overlap`
   （`backend/pkg/maintenance/maintenance.go:40-52`），加 `cancelled`（主设计 §9.2）。
   **skipped 与 overlap 也必须落行**——主设计 §9.2 规定"任何结局都不允许静默"，
   否则历史视图会出现无法解释的空洞。
2. `uk_schedule_attempt_no` 保证同一执行的重试序号唯一，重复写入是 bug 而不是数据。
3. `param_digest` 存摘要不存原文，与 `system_log_oper.oper_param` 的脱敏纪律同源
   （[DATABASE.md](./DATABASE.md) §3.2）。
4. 本表只追加、不更新；参数快照一旦写下不可变，因此"任务后来被编辑"不会改写历史
   （主设计 §11 约束 2）。

## 3. 索引与查询匹配

| 热查询 | 走的索引 | 备注 |
| :--- | :--- | :--- |
| tick 取到期任务 | `idx_schedule_job_due` | `enabled = 1 AND paused = 0 AND next_fire_at <= now()` + `ORDER BY next_fire_at` + 限量 50（主设计 §7.1，禁止无界扫描） |
| 租约领取 | `uk_schedule_lease_job` + 主键行锁 | `SELECT ... FOR UPDATE NOWAIT`（主设计 §7.2 步骤 1） |
| 幂等命中 | `uk_schedule_execution_fire` | 插入冲突即读取既有行 |
| 列表页（按来源域/状态筛选） | `idx_schedule_job_source_domain` / `idx_schedule_job_schedule_type` | 低基数列单独建索引只为覆盖筛选组合；组合过多会反噬写入，见 §9 DB-OD-2 |
| 历史列表 | `idx_schedule_execution_job_time` | 分页走 `common.NormalizePagination(page, pageSize, 100)`（`backend/pkg/common/handler.go:39`） |
| 健康摘要（dashboard） | `idx_schedule_execution_status_time` | 聚合最近窗口，不做全表 `COUNT(*)` |
| 孤儿对账 | `idx_schedule_lease_expire` | 只在启动与 tick 内跑 |

状态列使用低基数 `VARCHAR(16)` 且允许为空串默认值，符合 [DATABASE.md](./DATABASE.md) §4 的
"状态列 + 索引，禁止物理外键"约束；表间关联全部是逻辑外键（`job_id` / `execution_id`）。

## 4. 迁移编号与文件

当前 `backend/pkg/database/migrations/` 最新编号是 **000031**
（`000031_drop_permission_role_data_scope_policy.up.sql` / `.down.sql`）。
报表中心包占用 000032/000033（见 [REPORT_CENTER_DB_DESIGN.md](./REPORT_CENTER_DB_DESIGN.md) §4），
本包按预约顺序占用：

| 编号 | 文件 | 内容 |
| :--- | :--- | :--- |
| 000034 | `000034_scheduler_center_core_tables.up.sql` | 建 §2 的 4 张表（全部 `CREATE TABLE IF NOT EXISTS`） |
| 000034 | `000034_scheduler_center_core_tables.down.sql` | 见 §7 |
| 000035 | `000035_scheduler_center_setting_seed.up.sql` | 幂等 seed §6 的 `system_setting` 键 |
| 000035 | `000035_scheduler_center_setting_seed.down.sql` | 删除这些键行 |

编号登记规则：**同一 PR 内不得复用编号**；若与其他并行设计包冲突，实现时以先到者为准并重编号，
冲突结果写进 PR 说明。

seed 写法必须沿用 000030 的守卫式插入（不做 UPDATE 覆盖），原文理由见
`backend/pkg/database/migrations/000030_tenant_plan_definitions.up.sql:22-28`：

```sql
INSERT INTO `system_setting` (`setting_key`, `setting_value`, ...)
SELECT 'schedule.execution_retention_days', '30', ...
WHERE NOT EXISTS (SELECT 1 FROM `system_setting` WHERE `setting_key` = 'schedule.execution_retention_days');
```

其他必须遵守的迁移事实：

- `currentRuntimeSchemaVersion` 与兼容 marker 机制会把"看起来已是当前 schema"的库直接基线化
  （`backend/pkg/database/migrate.go:39`、`:157-202`）。新增表属于运行期必需，**不需要**新增 marker；
  但绝不能依赖 `PANTHEON_AUTO_MIGRATE` 的 GORM AutoMigrate 作为第二份 schema 权威
  （教训原文见 `backend/internal/middleware/data_scope_middleware.go:73-85`）。
- 迁移由 golang-migrate 的 MySQL driver 执行，该 driver 用 `SELECT GET_LOCK(?, 10)` 做跨 Pod 串行化，
  多副本滚动更新时只有先拿到锁的实例执行迁移（事实与处置见
  [MIGRATION_000031_PRODUCTION_DROP.md](../runbooks/MIGRATION_000031_PRODUCTION_DROP.md) §1.2、
  `docs/reviews/PANTHEON_BASE_GLOBAL_AUDIT_2026-10-31.md:1070`）。
  **注意**：这把锁只属于迁移路径，调度中心运行期不使用 `GET_LOCK`（主设计 §7.2）。
- 000034 是**纯新增表**，不含破坏性 `DROP`，因此不触发 preflight 流程；
  若实现阶段改成回收既有表，必须先补 `preflight` 脚本，先例是
  `backend/pkg/database/migrations/preflight/000031_preflight.sql`。

## 5. 软删除与唯一键释放（仓库既有规则）

规则：MySQL 唯一索引**不得包含 `deleted_at`**（先例 `000001_init_schema.up.sql:120`，
约定原文见 [DATABASE.md](./DATABASE.md) §1、§3.3）。因此"删除后同名可复用"必须靠**改名释放**实现，
本仓库已有四处同形实现可直接照抄：

| 先例 | 位置 | 形状 |
| :--- | :--- | :--- |
| 岗位 code 归档 | `backend/modules/system/org/post/post_service.go:40`、`:789-806` | `__deleted_post_<id>`，最多重试 5 次，用 `unixnano` 造唯一尾巴，冲突返回 `common.NewConflict` |
| 字典键归档 | `backend/modules/system/config/dict/dict_service.go:1483`、`:1501` | 同上 |
| 租户码归档 | `backend/modules/system/iam/tenant/tenant_service.go:847` | 同上 |
| 报表（本包同族设计） | [REPORT_CENTER_DB_DESIGN.md](./REPORT_CENTER_DB_DESIGN.md) §5 | `__deleted_report_<id>` |

本包规则：

1. 删除 `schedule_job` 时把 `job_key` 改写为 `__deleted_schedule_<id>` 形式再软删，
   冲突时按既有先例重试 5 次并返回既有冲突错误键，**不得**返回自然语言。
2. 删除任务必须同事务删除其 `schedule_lease` 行（租约不是历史，不需要保留）。
3. `schedule_execution` / `schedule_execution_attempt` 不参与改名释放：它们没有业务唯一键
   （执行行的唯一键是幂等键，幂等键本身已含 `job_key` 的历史值，复用名不会撞车——
   这是 §9 DB-OD-1 里需要复核的点）。
4. 禁止用"把 `deleted_at` 加进唯一索引"来绕过改名，那会让同一次删除后仍无法重建同名任务。

## 6. 保留期与清理（retention / purge）

### 6.1 设置键

| 键 | 默认 | 消费方 | 说明 |
| :--- | :--- | :--- | :--- |
| `schedule.execution_retention_days` | 30 | 清理任务 | 执行与尝试行的保留天数；读不到用代码默认值，写法沿用 `audit.*` 先例（`backend/modules/system/audit/audit_service.go:325-341`） |
| `schedule.tick_interval_seconds` | 60 | tick 循环 | 主设计 §7.1 |
| `schedule.lease_seconds` | 120 | 租约 | 必须 > 最长任务超时，否则租约会被续租失败误判为丢失 |
| `schedule.retry_backoff_max_seconds` | 1800 | 重试 | 主设计 §9.1 封顶 |
| `schedule.default_timezone` | 待定（OD-S-6） | cron 计算 | 主设计 §6.2 |
| `schedule.sync_row_cap` | 50 | tick 批大小 | 与"禁止无界扫描"同族的上限护栏 |

这些键属于平台级执行策略，归 `system/config` 治理但由调度中心解释消费语义
（[SYSTEM_CONFIG_CONTRACT.md](../contracts/SYSTEM_CONFIG_CONTRACT.md) §7.4）。

### 6.2 清理实现

- 以任务形式注册，注册通道形状照抄既有 `RegisterMaintenanceTasks`
  （`backend/modules/system/audit/audit_service.go:317`；registry 语义见
  `backend/pkg/maintenance/maintenance.go:92-114`），重叠保护由 registry 提供
  （`maintenance.go:181-187`），**清理逻辑自己不再实现锁**。
- 删除必须分批：`DELETE ... WHERE created_at < ? ORDER BY id LIMIT 500` 循环，直到影响行数为 0，
  避免一次大事务锁表并把 binlog 撑爆；这是既有清理任务的同一形状
  （清理类任务的"可重复调用"要求见 `maintenance.go:27-30`）。
- 先删 `schedule_execution_attempt`，再删 `schedule_execution`（无物理外键，顺序错了会留下孤儿行；
  孤儿行必须可对账，实现时要有一条"孤儿 attempt 计数"的检查）。
- **清理不得挂在请求路径上**：列表/详情 handler 只读，理由与既有教训一致
  （`maintenance.go:1-12` 包注释：一次普通读取曾为无界清理买单）。
- 运维事实：`PANTHEON_MAINTENANCE_ENABLED=false` 时后台 runner 不启动
  （`backend/cmd/server/main.go:204-213`），执行历史会无限增长。文档、UI 与告警都必须显式承认这一点，
  不得声称"保留期一定会生效"。

### 6.3 租约与状态收敛

- 租约行的生命周期跟随任务，不参与保留期清理（§5 规则 2）。
- `schedule_execution.status = running` 且租约已过期的行由 §7 对账逻辑收敛为 `failed`，
  不靠清理任务处理。

## 7. down 形状

`000034_scheduler_center_core_tables.down.sql` 的目标形状：

```sql
-- DATA PROMISE: 本 down 会删除调度中心的执行历史，数据不可恢复。
DROP TABLE IF EXISTS `schedule_execution_attempt`;
DROP TABLE IF EXISTS `schedule_execution`;
DROP TABLE IF EXISTS `schedule_lease`;
DROP TABLE IF EXISTS `schedule_job`;
```

约束与理由：

1. 顺序必须是 §2 的**逆序**（与既有 down 文件同一纪律；形状本身有测试守护，见 §8）。
2. 本仓库对 down 的既有态度是"形状优先 + 数据承诺写在文件头"，
   `000031_drop_permission_role_data_scope_policy.down.sql` 的原文措辞是直接可抄的模板：
   `DATA PROMISE: this up migration destroys the table and any rows in it` / `DATA IS NOT RESTORED`。
   调度侧同理：down 不回滚执行历史，任何"执行历史可以还原"的暗示都不允许出现在文档里。
3. down **不得**触碰报表中心表、`system_setting` 非本包键、或任何既有表。
4. `000035` 的 down 只删本包 seed 的 6 个键行（按 `setting_key` 精确匹配），不得用前缀模糊删除——
   前缀 `schedule.` 未来可能被其他域复用。

## 8. 落库前的门禁清单（实现 PR 必须过）

| 检查 | 命令 |
| :--- | :--- |
| 迁移 up/down 形状 | `cd backend && go test ./pkg/database/`（既有夹具 `backend/pkg/testmysql`；形状断言的写法参照 `backend/pkg/database/migrate_down_shape_test.go`、`migrate_down_test.go`、`migrate_preflight_test.go`） |
| 唯一键不含 `deleted_at` | 同上包内新增断言；参照 `backend/pkg/database/migrations/000001_init_schema.up.sql:117-120` 的形状 |
| 租约协议可证明 | `cd backend && go test ./pkg/schedule/ -race`，用 `backend/pkg/testmysql/locking.go:18`（`AssertRowLocked`）与 `:39`（`AssertSerialisedWriters`） |
| 后端单测 | `cd backend && go test ./...`（CI job `unit-tests` / `backend-tests`） |
| Go lint | `golangci-lint`（CI job `go-lint`） |
| 权限键形状 | `npm run check:harness-permission-contract` |
| DTO / 响应信封 | `npm run check:backend-dto-contract`、`npm run check:backend-response-contract` |
| 文档 | `npm run check:docs-frontmatter`、`npm run check:harness-docs` |

## 9. 开放决策 / Open decisions

| 编号 | 决策 | 选项 | 影响 | 决策人 |
| :--- | :--- | :--- | :--- | :--- |
| DB-OD-1 | 幂等键是否随任务改名一起失效 | A：执行行只认 `(job_id, fire_instance)`（当前设计，`job_id` 稳定）；B：把 `job_key` 纳入唯一键 | B 会让"删除后重建同名任务"产生跨代历史混淆；当前选 A，实现时需复核 `fire_instance` 的生成不含已改名的 `job_key` | 维护者 |
| DB-OD-2 | 列表筛选需要哪些组合索引 | A：只保留单列索引（当前）；B：为 `(source_domain, schedule_type)` 等组合建索引 | B 提升筛选但增加写放大；第一阶段列表基数低（数十行），A 足够 | 维护者 |
| DB-OD-3 | 执行历史是否需要保留 stdout | A：只存摘要与错误键（当前）；B：新增 `schedule_execution_log` 表 | B 与主设计 §13"不轮询日志内容"的非目标冲突，且引入新的保留期对象 | 维护者 |
| DB-OD-4 | 租约表与任务表是否合并为一行内的几列 | A：独立 `schedule_lease` 表（当前）；B：在 `schedule_job` 上加 `holder/lease_*` 列 | B 少一次连接开销，但租约行锁会把任务定义的读也一起阻塞（同表同行锁）；第一阶段选 A | 维护者 |

## 10. 关联文档

| 文档 | 中文 | 英文 |
| :--- | :--- | :--- |
| 主设计 | [SCHEDULER_CENTER_DESIGN.md](./SCHEDULER_CENTER_DESIGN.md) | [SCHEDULER_CENTER_DESIGN.en.md](./SCHEDULER_CENTER_DESIGN.en.md) |
| 数据模型与迁移（本文） | `SCHEDULER_CENTER_DB_DESIGN.md` | SCHEDULER_CENTER_DB_DESIGN.en.md（planned/unavailable：`./SCHEDULER_CENTER_DB_DESIGN.en.md`；无当前交付承诺） |
| 接口契约 | SCHEDULER_CENTER_API_CONTRACT.md（planned/unavailable：`./SCHEDULER_CENTER_API_CONTRACT.md`；无当前交付承诺） | SCHEDULER_CENTER_API_CONTRACT.en.md（planned/unavailable：`./SCHEDULER_CENTER_API_CONTRACT.en.md`；无当前交付承诺） |
| 测试计划 | SCHEDULER_CENTER_TEST_PLAN.md（planned/unavailable：`./SCHEDULER_CENTER_TEST_PLAN.md`；无当前交付承诺） | SCHEDULER_CENTER_TEST_PLAN.en.md（planned/unavailable：`./SCHEDULER_CENTER_TEST_PLAN.en.md`；无当前交付承诺） |

数据库基础规范：[DATABASE.md](./DATABASE.md)。
租户资源分类矩阵：[TENANT_RESOURCE_SCOPE_MATRIX.md](./TENANT_RESOURCE_SCOPE_MATRIX.md)。
目录索引：[README.md](./README.md)。

## 变更日志

| 日期 | 变更 |
| :--- | :--- |
| 2026-11-04 | 定义 4 张表与索引、迁移编号 000034/000035、唯一键与改名释放、保留期与分批清理、down 形状、落库门禁清单与开放决策 |
