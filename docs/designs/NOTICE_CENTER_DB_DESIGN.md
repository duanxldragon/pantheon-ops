---
title: 通知中心数据库设计
doc_type: Design
layer: platform
status: Draft
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
  - docs/contracts/TENANT_CONTRACT_V1.md
  - docs/designs/NOTICE_CENTER_API_CONTRACT.md
updated_at: 2026-11-04
---

# 通知中心数据库设计

English version: [NOTICE_CENTER_DB_DESIGN.en.md](./NOTICE_CENTER_DB_DESIGN.en.md)

> 未实现声明：本文为设计稿，仓库内尚无对应实现；在实现落地前状态保持 Draft。
> Not implemented: this is a design draft with no corresponding implementation in the repository; it stays Draft until implemented.

关联设计：
- `NOTICE_CENTER_DESIGN.md`
- `NOTICE_CENTER_API_CONTRACT.md`
- `NOTICE_CENTER_TEST_PLAN.md`

---

本文给出通知中心的表、列、索引、迁移与清理形状，使实现者不需要自行决定 DDL。所有列名/表名是**待实现契约**。

## 1. 前置与本仓库的建表规范

本文完全继承 `docs/designs/DATABASE.md`，逐条对应：

| 规范 | 出处 | 本文落法 |
|------|------|----------|
| InnoDB + `utf8mb4_general_ci` | `DATABASE.md` §1 | 三张表统一 |
| 主键 `bigint unsigned NOT NULL AUTO_INCREMENT` | `DATABASE.md` §1 | 用自增，不用雪花（`docs/contracts/TENANT_CONTRACT_V1.md` §2.1 的 `tenants.id` 是租户主数据的例外，不是通用规范） |
| 软删除 `deleted_at datetime(3) NULL`（GORM 默认） | `DATABASE.md` §1 | 只在本体表存在，见 §5 |
| 底座表 `system_` 前缀 | `DATABASE.md` §2 | `system_notice*`；**不**采用 `platform_notice` 前缀，因为 §2 把底座表前缀钉死为 `system_`，`platform` 是逻辑层不是表前缀 |
| 严禁物理外键 | `DATABASE.md` §4 | 表间关系仅由 Service 层维护 |
| `status` 单列低区分度不建独立索引 | `DATABASE.md` §4 | 只做复合索引的一部分 |
| 权威建库路径 = golang-migrate 编号迁移 + Go seed | `DATABASE.md` §5 | 见 §7 |

## 2. 表清单

| 表 | 角色 | 是否必须随 V1 落库 |
|----|------|--------------------|
| `system_notice` | 消息本体，一条消息一行 | 是 |
| `system_notice_target` | 收件人视图，收件人 × 消息一行 | 是 |
| `system_notice_delivery` | 站外渠道投递账本，收件人 × 消息 × 渠道一行 | 表建，但只有注册了非 `in_app` 渠道才写入 |
| `system_notice_channel_config` | 渠道参数与开关 | 否，随 §9 开放决策 D-1 一并裁定 |

## 3. 关系

```
system_notice (1) ──< system_notice_target (N) ──< system_notice_delivery (0..M, M=站外渠道数)
```

不使用外键（§1），因此三处关系由 Service 层保证，删除顺序见 §8.3。

## 4. 列级 DDL 意图

「意图」= 语义 + 类型 + 空值与默认值规则；实现时按 `docs/designs/DATABASE.md` 落成 SQL。

### 4.1 `system_notice`（消息本体）

| 列 | 类型 | 空/默认 | 意图 |
|----|------|---------|------|
| `id` | `bigint unsigned` | PK AI | 主键 |
| `tenant_id` | `bigint` | `NOT NULL DEFAULT 0` | 租户归属；`0` = 平台级/全局资源（`TENANT_CONTRACT_V1` §2.3）；**禁止可空**（同节禁止条款） |
| `category` | `varchar(32)` | `NOT NULL` | `task/alert/audit/system/business` 枚举，见设计 §4.3 |
| `severity` | `varchar(16)` | `NOT NULL DEFAULT 'info'` | `info/warning/critical` |
| `status` | `varchar(16)` | `NOT NULL DEFAULT 'published'` | 本体状态 `draft/published/withdrawn`，与视图状态分列（设计 §5.3） |
| `source_domain` | `varchar(64)` | `NOT NULL` | 来源域标识，如 `platform`、`system.auth`、`business.*`；用于审计与跳转归因 |
| `source_type` | `varchar(64)` | `NOT NULL DEFAULT ''` | 源域内的事件/对象类型 |
| `source_id` | `varchar(64)` | `NOT NULL DEFAULT ''` | 源对象标识，字符串而非 bigint：源域主键形态不受平台约束 |
| `title` | `varchar(256)` | `NOT NULL` | 已渲染文本（设计 §4.2） |
| `summary` | `varchar(512)` | `NOT NULL DEFAULT ''` | 列表摘要；长文本不进列表字段 |
| `content` | `text` | `NULL` | 详情正文；V1 只允许纯文本/受控 Markdown（`content_type` 决定渲染） |
| `content_type` | `varchar(16)` | `NOT NULL DEFAULT 'text'` | `text/markdown`；富文本是设计 §2 非目标 |
| `payload` | `json` | `NULL` | 跳转参数与源域上下文快照；平台不解析，只透传给详情页 |
| `jump_type` | `varchar(16)` | `NOT NULL DEFAULT 'none'` | `route/external-url/none` |
| `jump_target` | `varchar(512)` | `NOT NULL DEFAULT ''` | 路由路径或 URL；落库前按设计 §6.2 校验 |
| `jump_perm` | `varchar(128)` | `NOT NULL DEFAULT ''` | 跳转目标页所需权限 key；空 = 无额外要求。仅为前端预隐藏，服务端仍独立鉴权 |
| `dedup_key` | `varchar(191)` | `NULL DEFAULT NULL` | 摄入幂等键（设计 §8.1）。**必须可空**：MySQL 唯一索引允许任意多行 `NULL`，而空字符串只允许一行，用 `''` 会让第二条无幂等键的消息直接撞唯一键 |
| `expires_at` | `datetime(3)` | `NULL` | 业务过期时间；到期后从列表隐藏，与 §8 的保留清理不同概念 |
| `created_by` | `bigint` | `NOT NULL DEFAULT 0` | 设计字段的 `actorId`；系统生成时为 `0` |
| `created_at` / `updated_at` | `datetime(3)` | `NULL` | GORM 默认 |
| `deleted_at` | `datetime(3)` | `NULL` | 本体软删，保留消息事实供审计回溯（设计 §10.3） |

### 4.2 `system_notice_target`（收件人视图）

| 列 | 类型 | 空/默认 | 意图 |
|----|------|---------|------|
| `id` | `bigint unsigned` | PK AI | 主键 |
| `tenant_id` | `bigint` | `NOT NULL DEFAULT 0` | 与本体同值，由写入路径从上下文赋值（`TENANT_CONTRACT_V1` §2.3.1 第 2 条） |
| `notice_id` | `bigint unsigned` | `NOT NULL` | → `system_notice.id`，无物理外键 |
| `recipient_id` | `bigint` | `NOT NULL` | 收件人 `system_user.id`；不做存在性外键校验，源数据有效性归源域（设计 §3.2） |
| `read_at` | `datetime(3)` | `NULL` | 非空即已读（设计 §9.1）；一旦写入不再清空 |
| `archived_at` | `datetime(3)` | `NULL` | 非空即归档 |
| `delivered_at` | `datetime(3)` | `NULL` | `in_app` 的送达时刻；站外渠道的送达看 `system_notice_delivery` |
| `created_at` / `updated_at` | `datetime(3)` | `NULL` | GORM 默认 |

本表**没有** `deleted_at`，理由见 §5.3。

### 4.3 `system_notice_delivery`（站外投递账本）

| 列 | 类型 | 空/默认 | 意图 |
|----|------|---------|------|
| `id` | `bigint unsigned` | PK AI | 主键 |
| `tenant_id` | `bigint` | `NOT NULL DEFAULT 0` | 同上 |
| `notice_id` | `bigint unsigned` | `NOT NULL` | 冗余，便于按消息聚合与清理 |
| `target_id` | `bigint unsigned` | `NOT NULL` | → `system_notice_target.id`，投递单元归属 |
| `recipient_id` | `bigint` | `NOT NULL` | 冗余，供跨租户运营读展示 |
| `channel` | `varchar(32)` | `NOT NULL` | `email/sms/webhook`；`in_app` 不落本表（设计 §7.3） |
| `status` | `varchar(16)` | `NOT NULL DEFAULT 'pending'` | `pending/success/failed/dead/skipped`（设计 §5.2） |
| `attempts` | `int` | `NOT NULL DEFAULT 0` | 含首次尝试，上限默认 5（设计 §8.2） |
| `next_attempt_at` | `datetime(3)` | `NULL` | 退避驱动列；worker 认领条件 |
| `last_attempt_at` | `datetime(3)` | `NULL` | 诊断用 |
| `external_id` | `varchar(191)` | `NOT NULL DEFAULT ''` | 渠道侧消息 id，排障与二次去重；**不**存凭证 |
| `last_error` | `varchar(512)` | `NULL` | 截断存储；禁止落 token/密码（`docs/designs/QUALITY_AND_SECURITY_STRATEGY.md`） |
| `sent_at` | `datetime(3)` | `NULL` | 送达确认时刻 |
| `created_at` / `updated_at` | `datetime(3)` | `NULL` | GORM 默认 |

同样没有 `deleted_at`：账本行是运维过程数据，到期硬删（设计 §10.3），保留 `dead` 行多一个周期。

## 5. 索引与唯一键

### 5.1 本仓库的硬规则：MySQL 唯一索引不含软删列

- `docs/designs/DATABASE.md` §3.3.2 记录用户名的做法：删除时先把唯一标识**归档改名**，再软删除，使键可被重新创建；§3.3.3 把同一策略推广到 `system_role.role_key`、`system_post.post_code`、`system_dict_type.dict_code`、`system_dict_item(dict_code, item_value)`。
- 代码侧可核对的实现：`backend/modules/system/iam/user/user_username.go:18`（`deletedUsernamePrefix = "__deleted_user_"`）、`:144`、`:164`（改名候选与冲突重试）。
- 结论：唯一索引里不塞 `deleted_at`「区分已删」的写法。释放唯一键只有两条出路：**改名**，或者该行没有保留价值时**直接物理删除**。

通知中心的选择：

| 表 | 出路 | 理由 |
|----|------|------|
| `system_notice` | 改名不适用 → 本体不建业务唯一键（除 `dedup_key`），软删可用 | 本体没有「重新创建同名」需求 |
| `system_notice_target` | 物理删除 | 视图行没有审计价值；改名（例如给 `recipient_id` 加后缀）会污染收件人标识 |
| `system_notice_delivery` | 物理删除 | 同上 |

### 5.2 键清单

| 表 | 键 | 列 | 意图 |
|----|----|----|------|
| `system_notice` | `uk_notice_tenant_dedup` | `(tenant_id, dedup_key)` | 摄入幂等（设计 §8.1）；`dedup_key` 为 `NULL` 时不参与约束 |
| `system_notice` | `idx_notice_tenant_created` | `(tenant_id, created_at)` | 平台面按租户倒序列表 |
| `system_notice` | `idx_notice_source` | `(source_domain, source_type, source_id)` | 源域反查「这条业务单据发过哪些消息」 |
| `system_notice` | `idx_notice_status_expires` | `(status, expires_at)` | 撤回/过期扫描；`status` 单列不建索引的规则在此以复合形式满足 |
| `system_notice_target` | `uk_notice_target_notice_recipient` | `(notice_id, recipient_id)` | fan-out 不重复；不含软删列（§5.1） |
| `system_notice_target` | `idx_notice_target_recipient_read` | `(tenant_id, recipient_id, read_at)` | 未读计数覆盖索引（设计 §9.2） |
| `system_notice_target` | `idx_notice_target_recipient_created` | `(recipient_id, id)` | 收件人列表分页游标（§6.2） |
| `system_notice_target` | `idx_notice_target_archived` | `(recipient_id, archived_at)` | 归档视图筛选 |
| `system_notice_delivery` | `uk_notice_delivery_target_channel` | `(target_id, channel)` | 一个投递单元一行，重试原地更新（§5.4） |
| `system_notice_delivery` | `idx_notice_delivery_claim` | `(status, next_attempt_at)` | worker 认领扫描 |
| `system_notice_delivery` | `idx_notice_delivery_notice` | `(notice_id)` | 按消息清理/聚合 |

`external_id` 不建唯一键：不同渠道命名空间不互通，跨渠道共键会造成假冲突。

### 5.3 为什么视图表没有软删列

如果 `system_notice_target` 带 `deleted_at`，那么「用户删除这条消息」之后再收到同一 `dedup_key` 的消息时，`uk(notice_id, recipient_id)` 或幂等链路会被一条不可见的软删行占住——这正是 `DATABASE.md` §3.3.2 描述、并用改名绕开的老问题；通知视图既没有改名空间（§5.1），也没有审计留存价值（消息事实保存在本体表与操作日志里），所以删除语义定为物理删除。设计文档 §5.1 状态机的「删除」行与 §10.3 与此对齐。

### 5.4 为什么不建尝试历史表

`attempts + last_error + last_attempt_at` 足以回答「为什么没送到」与「还要不要再试」，逐次尝试历史属可观测性增强。V1 不建第四张表（最小复杂度阶梯），改为把失败分布交给指标（设计 §14 的 `pantheon_notice_delivery_total{outcome}`）。是否需要逐次历史登记为 §9 D-3。

## 6. 数据形状与读路径约束

### 6.1 分页有界

- 列表分页统一走 `common.NormalizePagination(page, pageSize, maxPageSize)`（`backend/pkg/common/handler.go:39-50`：`page<=0 → 1`、`pageSize<=0 → 10`、超过模块上限则钳到上限，并派生 offset）。
- 通知模块上限取 **100**，与既有模块一致：`backend/modules/system/audit/audit_service.go:47`（`maxOperationLogPageSize = 100`）、`backend/modules/system/config/dict/dict_service.go:24`、`permission_service.go:937`、`role_helper.go:534`、`post_service.go:663` 等传参均为 100。合同层面只承诺「有界」，不承诺超限输入被静默改小（对齐 `TENANT_CONTRACT_V1` §2.5 规则 4 的表述方式）。
- 导出（若开放）另设行数上限，参照 `maxOperationLogExportRows` 的既有做法（常量 `audit_service.go:48`，用于 `audit_service.go:372` 的 `Limit()`）。

### 6.2 深翻页

收件人列表以 `(recipient_id, id)` 做游标式排序，不承诺任意 `page` 的常数时间；平台管理面若需要深翻页，走时间窗过滤而不是 offset 大跳。

### 6.3 体量假设

- 一条平台级消息 = 1 本体行 + N 收件人行（N = 收件人数）。广播型消息（全员）会产生与用户数同量级的行，因此 §8.2 的 fan-out 必须分批。
- V1 不承诺「全员广播」性能：建议 fan-out 批次上限 500 行/批，与 `audit_service.go:357` 的批量入参上限同源直觉。

## 7. 迁移编号与形状

### 7.1 编号

当前最新编号迁移是 `backend/pkg/database/migrations/000031_drop_permission_role_data_scope_policy`，目录内 000001–000031 连续。因此：

| 迁移 | 内容 |
|------|------|
| `000032_notice_center.up.sql` | 建 `system_notice`、`system_notice_target`、`system_notice_delivery`（三表一版，一个功能包一个编号） |
| `000032_notice_center.down.sql` | 逆序 `DROP TABLE IF EXISTS` 三表 |
| `000033_task_center.up.sql` | 任务中心表（见 `TASK_CENTER_DB_DESIGN.md`） |

规则依据：迁移由 golang-migrate 以 embed FS 执行（`backend/pkg/database/migrate.go:14-22`），文件名必须 `<version>_<name>.up.sql` 成对存在；`000030` 与 `000031` 是成对样例。

### 7.2 up 形状要求

参照 `backend/pkg/database/migrations/000030_tenant_plan_definitions.up.sql` 的写法：

1. 文件顶部写 **WHY THIS TABLE EXISTS / SEMANTICS** 注释块，说明为什么是这个形状（000030 用注释把「平台全局表不加 `tenant_id`」「-1 表示无上限」「种子只插不改」的理由钉在 DDL 旁边）。
2. `CREATE TABLE IF NOT EXISTS`，`ENGINE=InnoDB DEFAULT CHARSET=utf8mb4`。
3. 需要 seed 时写成 `INSERT ... SELECT ... WHERE NOT EXISTS (...)` 的逐键幂等形态（000030 第 23-28 行的注释解释了原因：迁移重放是真实状态，重放时改写既有值会静默撤销运营者；同规则见 `TENANT_CONTRACT_V1` §2.4 不变量 6）。
4. 通知中心 V1 **不在迁移里 seed 消息数据**；设置项与菜单/权限走 Go seed（§7.4）。

### 7.3 down 形状要求

- 本迁移的 down 是「表在版本 31 时不存在」，因此 down 必须只 DROP 本次新建的三张表，不得顺手 DROP 相邻表。
- 钉住方式沿用仓库既有测试形态：`backend/pkg/database/migrate_down_shape_test.go`（`TestDropPolicyTableDownReconstructsVersion30Shape` 用 `SHOW CREATE TABLE` 字节比较，验证「直接 up 到 N」与「up 到 N+1 再 down 回 N」同形）、`backend/pkg/database/migrate_down_test.go`。通知中心应新增同构测试：up 到 32 → down 到 31 → 断言三表不存在且其余 schema 与 31 基线一致。
- 顺序性：000031 已经是一个「删除表」的迁移，本包不重新引入它删掉的表。

### 7.4 与 AutoMigrate 双路径

`docs/contracts/TENANT_CONTRACT_V1.md` §2.4 记录了本仓库的双路径事实：编号 DDL 是权威，同时 AutoMigrate 部署走模块自己的 `MigrateFunc`（`PlanStore.Migrate()` 例子）。通知中心同样两处都要有：

- 编号迁移：`000032_notice_center`
- 模块 `MigrateFunc`：`contracts.FuncModule{ModuleName: "notice", MigrateFunc: ...}`（形态见 `backend/modules/system/system_modules.go:332-355`），保证未跑迁移的开发库也能起。

seed 侧：菜单 `SeedMenusFunc`（`backend/modules/system/seed.go:88-103` 的 `ensureMenuSeeds` 幂等）、i18n canonical 条目、`system_setting` 的保留期 key（`backend/modules/system/config/setting/setting_seed.go:65-68` 是同一形态）。

### 7.5 租户 scope 登记义务

三张表都带 `tenant_id`，因此必须在 `docs/designs/TENANT_RESOURCE_SCOPE_MATRIX.md` §2/§3 登记后才能合入（`TENANT_CONTRACT_V1` §2.3：「新增表必须在登记表注册后才允许合入」）。登记项：

| 表 | scope | 唯一键目标形态 |
|----|-------|----------------|
| `system_notice` | `tenant-replicated`（平台级消息用 `tenant_id = 0`） | `uk(tenant_id, dedup_key)`，租户内唯一 |
| `system_notice_target` | 同上 | `uk(notice_id, recipient_id)` |
| `system_notice_delivery` | 同上 | `uk(target_id, channel)` |

读写义务直接引用 `TENANT_CONTRACT_V1` §2.3.1 三条（读带租户谓词、写从上下文取租户、按裸键做存在性校验同样算读），实现评审可直接以条款号驳回。

## 8. 生命周期与清理

### 8.1 摄入 fan-out

1. 校验 `dedup_key`；命中 `uk_notice_tenant_dedup` → 返回既有 `notice_id`，不新建（设计 §8.1）。
2. 插入本体行（`tenant_id` 来自上下文，不来自请求体）。
3. 分批插入 target 行（每批 500），冲突忽略（`INSERT IGNORE` 语义或等价的 `(notice_id, recipient_id)` 去重）。
4. 站外渠道按 target × channel 生成 delivery 行。

### 8.2 fan-out 必须可重入

步骤 3/4 中断后重跑不得产生重复行——由 §5.2 的两个唯一键保证，而不是靠「先查后插」。

### 8.3 删除顺序

撤回（本体 `status = withdrawn`）不删任何行；清理按「先 delivery → 再 target → 最后本体软删/硬删」的顺序执行，避免出现有视图无本体的孤儿视图（无外键约束时只能靠顺序保证）。

### 8.4 清理任务

`maintenance.Task{Name: "notice.retention", Interval: maintenance.DefaultInterval, Run: ...}`，注册形态与 `backend/modules/system/audit/audit_service.go:317-323` 一致；保留期从 `system_setting` 读，缺省回落（同文件 `:325` 的 `getRetentionDaysFromSetting` 模式）。

## 9. 开放决策（数据层）

| 编号 | 议题 | 选项 | 建议 | 决策人 |
|------|------|------|------|--------|
| D-1 | 渠道配置载体 | A `system_notice_channel_config` 表；B `system_setting` 键值；C 配置文件/环境变量 | V1 无站外渠道 → 建议先 B（零新表），有渠道时再评估 A | 维护者 |
| D-2 | 本体是否需要独立「广播」表 | A fan-out 到 target；B 本体 + 惰性视图（首次访问才建行） | 建议 A：惰性视图会让未读计数变成不可计算的聚合，代价远大于 fan-out 行数 | 维护者 |
| D-3 | 是否需要逐次尝试历史 | A 单行原地重试（本文）；B 第四张 `system_notice_delivery_attempt` | 建议 A，等出现「必须复盘第 3 次为什么失败」的真实需求再补 | 维护者 |
| D-4 | `payload` 列类型 | A `json`；B `text` 存 JSON 串 | 建议 A：仓库运行时标准库是 MySQL（`DATABASE.md` §3.1 迁移约束），且平台不解析该列 | 维护者 |
| D-5 | 视图表行保留期与「已删消息」查询 | A 物理删（本文）；B 本体软删 + 视图物理删 | 建议 B 的现状写法即本文 §5.3，若要求「用户删过的消息可复原」则整包升为软删 + 改名策略 | 维护者 |

## 10. 完成定义

本表设计算「可实现」当且仅当：

- 三张表的列/键与 §4/§5 一致，且有编号迁移 + `MigrateFunc` 双路径
- up/down 成对且 down 形状有测试钉住（§7.3）
- 租户列与唯一键在 `TENANT_RESOURCE_SCOPE_MATRIX` 登记（§7.5）
- 分页与批量上限按 §6.1 落地
- §9 的 D-1/D-2 在实现前有裁定
