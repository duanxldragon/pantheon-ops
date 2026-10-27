# 生产备份恢复演练（G2 Gate）执行规程

English note: this procedure produces the **only** evidence that can flip gate G2
("备份 RPO/RTO 确认 + 恢复演练证据") of the tenant migration runbook §8. Until the
record below exists, G2 stays OPEN and production DDL remains prohibited
(runbook §8: G1–G4 未全绿前禁止生产 DDL/数据变更).

本规程回答三个问题，全部以**实测**回答，不接受纸面声明：

1. **RPO**（恢复点目标）：最近一次可用备份距故障时刻丢了多少数据？
2. **RTO**（恢复时间目标）：从宣布恢复到业务验证通过要多久？
3. **可恢复性**：生产备份真的能在预发环境还原成可用库吗？

---

## 1. 范围与角色

### 适用
- 租户迁移（`TENANT_MIGRATION_RUNBOOK.md`）执行前的 G2 gate 放行
- 任何以"生产备份可恢复"为前置条件的 schema 变更

### 不适用
- 副本/预发库的常规备份巡检（各环境自管）
- 对象存储（MinIO/S3）备份——本演练只覆盖 MySQL；对象存储快照恢复如需纳入，另立附录

### 角色
| 角色 | 职责 |
|------|------|
| DBA | 执行 §3 全部步骤，填写 §5 记录模板 |
| 维护者 | 提供预发环境与隔离网络；审核 §5 记录后签署 G2 |
| 应用负责人 | 提供 §3.6 验证所需的登录凭据与只读探针 |

**红线**：演练全程不接触生产库写路径；生产仅允许只读命令（§3.2）。

---

## 2. 前置条件（DBA 开工前逐项确认）

- [ ] 预发/隔离环境可用：一台能与生产备份存储连通、但与生产流量隔离的 MySQL 8.0.36 主机（版本必须与生产一致）
- [ ] 预发磁盘余量 ≥ 生产数据体积 × 2（还原副本 + binlog 重放工作区）
- [ ] 已知生产备份清单：全备周期、增量/binlog 保留策略、备份存储位置、加密方式与密钥可获取
- [ ] `mysql`/`mysqlbinlog`/`xtrabackup`（按实际备份工具）版本与生产备份兼容
- [ ] 维护者已获得演练时间窗（不需要生产停机窗口——这是预发演练）
- [ ] §5 记录模板已复制到 evidence 目录

---

## 3. 执行步骤

### 3.1 记录备份策略现状（只读）

在生产/备份管理机上采集，全部输出进 §5 记录：

```bash
# 备份任务清单与最近成功时间（按实际调度工具调整：crontab / 定时任务 / 备份控制台）
crontab -l | grep -i backup
# 备份文件清单（名称、体积、时间戳、校验和）
ls -lh <backup-dir> | tail -30
sha256sum <最新全备文件>
```

记录：全备频率 ____、binlog/增量频率 ____、保留天数 ____、最新全备时间 ____。

### 3.2 采集 RPO 事实（只读）

```sql
-- 生产库执行（只读）：
SHOW MASTER STATUS;          -- 当前 binlog 文件与位点
SELECT NOW(6) AS captured_at;
```

对照最新全备内记录的 `SHOW MASTER STATUS` 位点（xtrabackup: `xtrabackup_binlog_info`；mysqldump: dump 头部）：

```
RPO(最大窗口) = NOW(6) − (全备完成时刻 + 已重放 binlog 覆盖段)
```

同时确认：备份是否启用了 GTID；binlog 格式是否 `ROW`（非 ROW 格式必须如实记录为风险项）。

### 3.3 还原全备到预发（RTO 计时开始）

```bash
date +%s.%N > /tmp/g2-restore-start        # ⏱ RTO 计时起点：本条命令执行前一刻
# 按生产实际备份工具执行还原，示例（xtrabackup）：
# xtrabackup --decompress --target-dir=<prepared-dir>
# xtrabackup --prepare --target-dir=<prepared-dir>
# xtrabackup --copy-back --target-dir=<prepared-dir> && chown -R mysql:mysql /var/lib/mysql
# 示例（mysqldump）：
# mysql --init-command="SET SESSION sql_log_bin=0" < full_backup.sql
systemctl start mysql   # 或对应启动方式
date +%s.%N > /tmp/g2-restore-end
```

### 3.4 重放 binlog 至目标时刻

```bash
# 定位全备位点之后的 binlog 文件，重放到"最近一次全备后、演练开始前"的最后一个事务
mysqlbinlog --start-position=<全备位点> <binlog文件...> | mysql --init-command="SET SESSION sql_log_bin=0"
```

记录：重放的 binlog 文件数、总字节、重放耗时、遇到的告警。

### 3.5 一致性校验（还原库 vs 生产，只读对比）

对下列每张表执行双侧行数对比（生产侧只读）：

- 全表清单行数（`information_schema.tables` 汇总）
- 抽样 5 张核心业务表 checksum（`CHECKSUM TABLE` 或按主键分段聚合）

```sql
-- 预发还原库：
SELECT table_name, table_rows FROM information_schema.tables WHERE table_schema='<restored>';
-- 生产（只读）：
SELECT table_name, table_rows FROM information_schema.tables WHERE table_schema='<prod>';
```

行数不一致处，须能被 3.4 重放截止时刻之后的正常业务写入解释，否则记为失败项。

### 3.6 应用级验证（RTO 计时结束点 = 本步通过）

还原库上启动后端（指向还原库 + 隔离 Redis），依次通过：

- [ ] `/api/v1/health` 返回 `database: ok`
- [ ] admin 登录成功（凭据由应用负责人提供，用后作废）
- [ ] `GET /system/menu/tree`、`GET /system/dict/type/list` 返回 200
- [ ] 迁移相关表存在性探针：`tenants`、`tenant_memberships`（若备份晚于队列表迁移则应存在；记录实际观察值）

```bash
date +%s.%N > /tmp/g2-app-verified       # ⏱ RTO 计时终点
```

```
RTO = g2-app-verified − g2-restore-start
    = 还原耗时 + binlog 重放耗时 + 启动耗时 + 应用验证耗时
```

### 3.7 结论判定

| 指标 | 判定 | 记录 |
|------|------|------|
| 还原成功率 | 全备还原 + binlog 重放无错误 → PASS | |
| RPO 实测 | 必须记录绝对值；租户迁移要求的上限见 §4 | |
| RTO 实测 | 同上 | |
| 一致性 | 行数/checksum 可解释 → PASS | |

任何一步失败：**如实记录失败现象与根因，G2 保持 OPEN**，修复备份/恢复管线后重跑本规程。

---

## 4. G2 放行标准（维护者评审用）

| 项 | 标准 |
|----|------|
| RPO | 实测值 ≤ 迁移窗口内可接受的数据丢失上限，且 ≤ 生产备份策略承诺值。租户迁移场景建议：RPO ≤ 15 分钟（binlog 连续可重放） |
| RTO | 实测值 × 2 ≤ 生产维护窗口（runbook §4.2 预估 ×2 的预留仍独立生效，两者取大） |
| 可恢复性 | §3.5 一致性校验全部可解释；§3.6 应用验证全过 |
| 证据 | §5 记录完整，含命令输出原文与时间戳；由 DBA 署名 |

满足以上四项 → 维护者在 runbook §8.1 表中将 G2 改为 GRANTED 并引用本演练记录；
任一项不满足 → G2 保持 OPEN，差距写入记录。

---

## 5. 演练记录模板（复制到 `.harness/evidence/2026-09-10-tenant-verification-and-gray/artifacts/`）

```markdown
# G2 生产备份恢复演练记录

- Run ID: g2-restore-<yyyymmdd_hhmmss>
- Date: ____
- 执行人（DBA）: ____    复核（维护者）: ____
- 生产 MySQL 版本: ____    预发还原环境: ____

## 备份策略现状
- 全备: 频率 ____，最新成功 ____，体积 ____，sha256 ____
- 增量/binlog: 频率 ____，保留 ____ 天，格式（ROW/_STMT）____，GTID: ____
- 备份存储位置与加密: ____

## RPO 实测
- 全备完成时刻: ____    全备内 binlog 位点: ____
- 演练采集时刻: ____    生产当前位点: ____
- **RPO = ____ 分钟**（计算过程）

## RTO 实测
- 还原开始: ____    还原完成: ____（耗时 ____）
- binlog 重放: 文件数 ____，字节 ____，耗时 ____
- 应用验证通过: ____（耗时 ____）
- **RTO = ____ 分钟**

## 一致性校验
- 全表行数差异清单（含解释）: ____
- 抽样 checksum 差异（含解释）: ____

## 应用级验证
- health: ____    登录: ____    menu/tree: ____    dict list: ____    表探针: ____

## 结论
- 还原: PASS/FAIL    RPO 达标: 是/否    RTO 达标: 是/否
- 未解决问题/风险: ____
- G2 建议: 放行 / 继续OPEN（原因）
```

---

## 6. 与租户迁移 runbook 的衔接

- 本规程通过后：更新 `TENANT_MIGRATION_RUNBOOK.md` §8.1 中 G2 行（引用本演练 Run ID），
  并勾选 §1 前置条件第 2 项。
- G3 仍需独立的生产表行数清单与 §4.2 窗口计算；本规程的还原库恰好可用于首次
  生产量级行数盘点（`information_schema.tables` 输出即为 G3 输入）。
- 演练中若发现备份策略缺陷（如 binlog 非 ROW、备份过期、密钥不可用），
  先修备份管线再重跑；不得以"迁移窗口临近"为由降低 §4 标准。
