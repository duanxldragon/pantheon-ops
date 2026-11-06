# 000031 生产执行规程：DROP `permission_role_data_scope_policy`

English note: this is the **execution procedure** for migration
`000031_drop_permission_role_data_scope_policy` against a production database.
It exists because the up migration is a `DROP TABLE` and its down migration
rebuilds the **structure only**. The procedure below produces the evidence a
human needs to decide whether the rows are disposable; it does **not** decide
that for you, and it does **not** grant the human gate (see §3.5).

本规程回答四个问题，全部以**实测**回答：

1. 这个 DROP 会不会连同数据一起消失？—— 会，且 down 不恢复数据。
2. 部署时它到底在什么时候执行？—— 服务启动时，自动，且与流量切换是同一个动作。
3. 执行前必须看到什么？—— §4.1 预检六条读数 + §4.3 的表级导出还原证明。
4. 出问题怎么退？—— 结构能退，数据只能从 §4.3 的导出恢复。

---

## 1. 风险模型：为什么"不可逆"不足以描述它

### 1.1 迁移本身

| 方向 | 内容 | 数据后果 |
|------|------|----------|
| up | `DROP TABLE IF EXISTS permission_role_data_scope_policy` | 表与其全部行被删除 |
| down | `CREATE TABLE IF NOT EXISTS ...`（结构按 v30 重建） | **只恢复形状，行不回来**（见 down 文件头 `DATA IS NOT RESTORED`） |

因此 `migrate down` 之后再 `up` 一次，等于再 DROP 一次；down 不是"撤销"。

### 1.2 真正的风险点：DROP 与部署不可分离

`PANTHEON_AUTO_MIGRATE=false`（生产配置，见 `k8s/configmap.yaml`）时，
服务启动流程是：

```
cmd/server/main.go:101  runDatabaseMigrations(dsn)
  → database.ShouldAutoMigrate() == false
  → database.RunMigrations(dsn)          // 应用所有 pending 迁移，不只是 000031
      → bootstrapExistingCurrentSchema()  // 仅"看起来是当前 schema"时打点，见 §1.3
      → m.Up()
```

由此得到三条对执行方式有直接影响的结论：

1. **没有独立的迁移步骤可以插在流量切换之前。** 镜像只构建 `./cmd/server`
   （`Dockerfile:41`），部署动作本身就是迁移动作。
2. **滚动更新时，最先启动的 Pod 执行 DROP。** 三副本 Deployment 每个 Pod 启动都会调用
   `RunMigrations`；golang-migrate 的 MySQL driver 用 `SELECT GET_LOCK(?, 10)`
   （v4.17.1 `database/mysql/mysql.go:298`）串行化，先取到锁的那个执行，其余在锁上等待。
   等待超过 10s 会返回 `ErrLocked`，该 Pod 启动失败并被 k8s 重启——表现为"部署抖动"，
   不是数据损坏，但会让"哪个 Pod 执行的 DROP"变得不确定。
3. **一次部署会顺带应用所有 pending 迁移。** 若目标库当前版本远低于 31，同一次启动还会执行
   000025–000030。§4.1 第 1 条语句因此要先读版本，再看 §4.2 的版本差值。

### 1.3 一个容易误判的点：版本号可能"跳"到 24

`RunMigrations` 在 `m.Up()` 之前会调用 `bootstrapExistingCurrentSchema`：若库"看起来是当前
schema"，就把版本直接打成 `currentRuntimeSchemaVersion`（当前为 **24**，
`pkg/database/migrate.go:39`）。

- 24 < 31，所以 **000031 不会被这个打点跳过**，DROP 一定会执行。
- 但若看到 `schema_migrations.version = 24` 而库里表结构其实已是最新，这是打点行为，不是漏迁移。
- 反向保护存在：`bootstrapMigrationVersion` 在 `recorded version > targetVersion` 时直接返回
  （`migrate.go:212`），所以已到 31 的库不会被降回 24、不会重跑 DROP。

### 1.4 这张表到底还有没有人读

实测（`grep -rn "permission_role_data_scope_policy" backend/ --include=*.go`）：
除迁移文件与两个测试外，**没有任何运行时代码引用该表**。Go 侧唯一的模型
`PermissionRoleDataScopePolicy` 的 `TableName()` 返回 `system_role_data_scope`
（`modules/system/iam/permission/permission_data_scope_model.go:14`）。

000025 已把 `tenant_id = 0` 的策略按角色归属回填到 `system_role_data_scope`
（`INSERT IGNORE`，幂等）。

**残留风险恰好在这里**：000025 只搬"当时存在"的行。任何绕过代码写入旧表的数据
（手工 SQL、历史脚本、旧版本未下线实例）不会被 000025 覆盖，只存在于旧表里。
§4.1 第 3/5 条语句就是用来发现这种情形的——**不要用"代码不读它"替代这两条读数**。

---

## 2. 适用范围

### 适用
- 任何即将部署到包含 000031 的版本、且 `schema_migrations.version < 31` 的生产库

### 不适用
- 已经 `version = 31` 的库（DROP 已完成，本规程只剩 §4.6 的事后核对）
- 一次性本地/测试库（`testmysql` 每次开新库，往返读数不代表生产）

---

## 3. 前置条件（全部满足才允许进入 §4）

- [ ] 已确认目标库 `version < 31` 且 `dirty = 0`（§4.2）
- [ ] §4.1 预检已执行，输出已归档到变更记录
- [ ] `rows_to_be_dropped = 0`，**或**已按 §4.3 导出并完成还原计数比对
- [ ] 已确认本次部署会应用的迁移清单（§4.2），非仅 000031
- [ ] §3.5 的放行决定已由维护者/DB 负责人做出并记录

### 3.5 放行口径（必须显式处理，不得默认）

[租户迁移 runbook §8](TENANT_MIGRATION_RUNBOOK.md) 的常设禁令是"G1–G4 未全绿前，禁止对任何
生产库执行本文任何 DDL/数据变更"，当前状态为 **G2 ✗ / G3 ✗**（`§8.1`）。

000031 是一次**破坏性 DDL**，与 G2（"生产备份可恢复"）直接相关。本文**不**代替 G2 放行，
也不主张 G2 已绿。可选的放行路径有两条，由维护者选择并记录：

| 路径 | 内容 | 代价 |
|------|------|------|
| A | 先完成 [G2 恢复演练规程](PRODUCTION_BACKUP_RESTORE_DRILL_G2.md)，再执行本规程 | 重，但一次性解决所有后续生产 DDL |
| B | 接受 §4.3 的**表级**导出+还原证明作为本次变更的等效控制 | 轻，但只覆盖这一张表；不能复用于其它 DDL |

路径 B 的合理性边界：000031 的影响面被限定在**一张已知死表**，行数由预检直接读出，
导出体积可忽略，还原验证成本很低。若预检显示行数异常大或存在非预期行，
路径 B 的前提（影响面有界）不成立，应退回路径 A。

> 代理不得代签本条。历史上有过代签并被否定的先例，见 `TENANT_MIGRATION_RUNBOOK.md` §8.1 审计注记。

---

## 4. 执行步骤

### 4.1 预检（只读，部署前）

脚本：`backend/pkg/database/migrations/preflight/000031_preflight.sql`
（6 条 SELECT，无写入、无锁表；由 `migrate_preflight_test.go` 锁定其读数语义）

```bash
mysql -h <host> -u <user> -p --table <db> \
  < backend/pkg/database/migrations/preflight/000031_preflight.sql
```

逐条读法：

| # | 输出列 | 期望（部署前） | 不符合时的含义 |
|---|--------|----------------|----------------|
| 1 | `version, dirty` | `30`（或 < 31 的任意值）/ `false` | `31` = 已执行过，跳到 §4.6；`dirty=true` = 上次迁移中断，**停止** |
| 2 | `dead_table_exists` | `1` | `0` = 表不存在，第 3/6 条会报错，属预期 |
| 3 | `rows_to_be_dropped` | `0` 最好 | `> 0` = 有行会被销毁，必须走 §4.3 |
| 4 | `replacement_table_exists` | `1` | `0` = 替换表缺失，**停止**，schema 与预期不符 |
| 5 | `replacement_rows` | `> 0`（若第 3 条 > 0） | 第 3 条 `> 0` 而本条 `0` = 旧表可能是唯一副本，**停止** |
| 6 | 最多 20 行明细 | 空，或与 §4.3 导出一致 | 用来肉眼确认"要消失的是什么" |

**第 3 与第 5 条必须一起读**：单看第 3 条为 0 只说明旧表空，单看第 5 条为 0 只说明新表空。
只有"3 = 0"或"3 > 0 且 5 > 0 且导出已验证"两种组合可以继续。

### 4.2 版本与 dirty 状态（只读）

`cmd/tenantmigration` 提供只读的 `status` 子命令，**不执行迁移**：

```bash
cd backend
PANTHEON_DSN='<dsn>' go run ./cmd/tenantmigration status
# 输出：Connected; migration version=30 dirty=false
```

注意两点：

- `status` 分支只 `RunMigrations` 当参数是 `up`；`status` 不会写库（`cmd/tenantmigration/main.go:23`）。
- 该二进制**不在 Docker 镜像内**（`Dockerfile` 只构建 `./cmd/server`），所以只能在仓库内
  `go run`，或直接用 §4.1 第 1 条语句读版本。见 §7 的登记。

同时确认本次部署会应用的迁移清单：

```bash
ls backend/pkg/database/migrations/*.up.sql | sed 's#.*/##' | sort | tail -8
```

当前最高版本为 31。若目标库版本是 N，本次启动会执行 N+1 … 31 的全部 up 迁移，
**不只是 000031**——这一点必须写进变更记录，否则变更记录会低估改动范围。

### 4.3 表级导出与还原证明（仅当 `rows_to_be_dropped > 0`）

```bash
# 导出（只导数据，结构由 down 迁移负责，不需要）
mysqldump -h <host> -u <user> -p --single-transaction --no-create-info \
  --skip-add-locks --skip-disable-keys \
  <db> permission_role_data_scope_policy \
  > 000031_permission_role_data_scope_policy_$(date +%Y%m%d_%H%M%S).sql

# 还原到一次性库并计数（这是"可还原"的证明，不是"已备份"的声明）
mysql -h <host> -u <user> -p -e 'CREATE DATABASE pf_restore_check'
mysql -h <host> -u <user> -p pf_restore_check < 000031_permission_role_data_scope_policy_*.sql
mysql -h <host> -u <user> -p pf_restore_check \
  -e 'SELECT COUNT(*) FROM permission_role_data_scope_policy'
```

**还原计数必须等于 §4.1 第 3 条读数**，否则导出不完整，不得继续。验证后删除一次性库。

导出文件与两条计数读数一起归档到变更记录——它们才是路径 B 放行的证据。

### 4.4 变更记录签署

在变更记录中写入以下内容（缺一项即视为未完成）：

- 目标库标识、执行时间窗
- §4.1 六条读数原样粘贴
- §4.2 版本号与本次将应用的迁移区间（N+1 … 31）
- §4.3 导出文件路径 + 还原计数比对结果（若适用）
- §3.5 选择的路径（A / B）与决定人

### 4.5 部署（DROP 在此发生）

按常规部署流程发布包含 000031 的版本即可，无需额外迁移命令。
执行者需要知道：**镜像启动的那一刻 DROP 就发生了**，没有回退窗口。

多副本滚动更新时，观察启动日志确认迁移由谁执行：

```
database migrations: all migrations applied successfully
```

### 4.6 部署后验证

```bash
mysql -h <host> -u <user> -p --table <db> -e "
SELECT version, dirty FROM schema_migrations;
SELECT COUNT(*) AS dead_table_exists FROM information_schema.tables
 WHERE table_schema = DATABASE() AND table_name = 'permission_role_data_scope_policy';
SELECT COUNT(*) AS replacement_rows FROM system_role_data_scope;"
```

期望：`version=31`、`dirty=false`、`dead_table_exists=0`，且 `replacement_rows`
与部署前 §4.1 第 5 条读数**一致**（DROP 不应影响替换表）。

随后做一次功能性验证：以一个配置了数据权限的角色登录，确认列表按预期范围过滤
（读取路径 `loadRoleDataScopePolicies` 走的是 `system_role_data_scope`）。

---

## 5. 回滚

### 5.1 触发条件
- 部署后出现数据权限相关 500（`fail-closed` 查找未命中）
- 发现 `replacement_rows` 与部署前不一致

### 5.2 回滚步骤

**顺序不可颠倒**：DROP 之后表已不存在，必须先建回结构，才能导入数据。

1. **先把应用回退到旧版本并停止启动新版本。** 若只回滚数据库而新版本仍在运行，
   下一次 Pod 启动会再次执行 000031，把刚导入的数据再删一次。数据库回滚与应用回滚是同一个动作。
2. 建回结构（等价于 down 迁移）：
   ```bash
   mysql -h <host> -u <user> -p <db> \
     < backend/pkg/database/migrations/000031_drop_permission_role_data_scope_policy.down.sql
   ```
3. 导入 §4.3 的导出：
   ```bash
   mysql -h <host> -u <user> -p <db> < 000031_permission_role_data_scope_policy_*.sql
   ```
4. 把版本标记退回 30（golang-migrate 的 down 也会写这个状态；此处为手工等价）：
   ```sql
   UPDATE schema_migrations SET version = 30, dirty = 0;
   ```
5. 复核：`SELECT COUNT(*) FROM permission_role_data_scope_policy` 必须等于导出前 §4.1 第 3 条读数。

> 第 2–4 步绕过了 golang-migrate 的 `GET_LOCK`。执行前必须确认没有任何 Pod 正在启动
> （`kubectl get pods` 无 `ContainerCreating`/`CrashLoopBackOff`），否则会与自动迁移竞争。

### 5.3 回滚不能做什么
- 没有导出就没有数据回滚。down 迁移明确声明不恢复数据。
- 回滚后若再次部署同一版本，会再执行一次 DROP；重跑前必须重新执行 §4.1 预检。

---

## 6. 失败模式速查

| 症状 | 判定 | 处置 |
|------|------|------|
| Pod 启动失败，日志含 `try lock failed` / `ErrLocked` | 另一副本正在迁移，锁等待超 10s | 等待其完成；k8s 会自动重启该 Pod，通常自愈。不要手工解锁 |
| 启动失败，`schema_migrations` 中 `dirty=true` | 迁移中途失败 | **停止**，按 golang-migrate 惯例先修 schema 再 `force` 版本；不要盲目重跑 |
| 预检第 4 条 `replacement_table_exists=0` | 库结构与预期不符 | 停止，核对版本；可能库尚未到 24 |
| 预检第 3 条 `> 0` 且第 5 条 `= 0` | 旧表可能是唯一副本 | 停止，导出并人工核对行内容后再决策 |
| 部署后数据权限 500 | 策略未落到 `system_role_data_scope` | 按 §5 回滚；这是 000025 未覆盖行的真实后果 |

---

## 7. 本规程发现的部署侧缺口（已登记，未修复）

写这份规程时发现两处文档与制品不一致，**不影响本规程的可执行性**（本文已改用等价命令），
但会影响任何"按文档手工迁移"的操作者：

1. `k8s/README.md:52` 写"`PANTHEON_AUTO_MIGRATE`: Set to `false` (run migrations manually)"，
   但镜像只构建 `./cmd/server`（`Dockerfile:41`），**镜像内没有任何可手工执行的迁移入口**；
   而 `false` 恰恰意味着迁移在服务启动时自动执行。"manual" 在当前制品下不可实现。
2. `cmd/tenantmigration` 存在且 `status` 是只读的、`up` 可独立执行迁移，但它不在镜像内，
   只能从仓库 `go run`。名字叫 tenantmigration，实际执行的是全量平台迁移——名字与职责不符。

这两条已作为残留登记在 `.harness`，不在本规程内修改。

---

## 8. 记录模板

```
迁移：000031_drop_permission_role_data_scope_policy
目标库：________   当前版本：____   dirty：____
本次将应用迁移区间：____ … 31

预检（粘贴 6 条读数）：
  1 version/dirty        = ____ / ____
  2 dead_table_exists    = ____
  3 rows_to_be_dropped   = ____
  4 replacement_exists   = ____
  5 replacement_rows     = ____
  6 sample rows          = ____ 行

导出（若第 3 条 > 0）：
  文件：____________________________________
  还原计数：____（须等于第 3 条）

放行路径：A / B（删除不适用项）
  路径 B 前提确认：影响面有界，行数 ____，非预期行：有 / 无
决定人：________   时间：________

部署后验证：
  version=____ dirty=____ dead_table_exists=____ replacement_rows=____
  功能验证（数据权限列表过滤）：通过 / 未通过
执行人：________   时间：________
```
