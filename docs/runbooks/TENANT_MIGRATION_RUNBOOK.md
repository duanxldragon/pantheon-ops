# 租户迁移 Runbook（共享 schema MVP）

本 runbook 是 `2026-09-10-tenant-migration-runbook` 任务的冻结交付物，为 [租户合同 V1](../contracts/TENANT_CONTRACT_V1.md) 定义的共享 schema MVP 提供可演练、可回滚的迁移程序。

> **生产执行禁令**：本 runbook 已在副本完成三类演练（成功 / 冲突 / 失败回滚，见 [演练日志](../../.harness/archive/2026-09/evidence/2026-09-10-tenant-migration-runbook/rehearsal-log.md)），
> 但 **任何生产 DDL、回填、NOT NULL/唯一索引变更都必须等待 human gate 放行**
> （备份 RPO/RTO 确认 + 生产维护窗口审批，见本文 §8）。在此之前只允许副本/预发执行。

---

## 1. 适用范围与前置条件

### 适用

- 新增 `tenants` / `tenant_memberships` 表（合同 §2.1/§2.2）
- 存量表增加 `tenant_id BIGINT NOT NULL DEFAULT 0`（按 [资源 scope 矩阵](../designs/TENANT_RESOURCE_SCOPE_MATRIX.md) 的 Phase-1 清单）
- 全局唯一键 → 租户维度唯一键的转换（仅 Phase-1 列出的键）
- Casbin policy domain 前缀化（`tenant:0`）

### 不适用

- 独立 schema / 独立 database（合同 §1 Out）
- 任何业务查询/认证/权限/缓存/上传 runtime 代码变更（那是 canary 任务的范围）

### 执行前置条件（全部满足才允许开始）

- [ ] 维护者已批准本 runbook（human gate ①：runbook approval）
- [ ] 目标库有可用备份，RPO/RTO 已确认并记录（human gate ②）
- [ ] 生产维护窗口已批准，窗口时长按 §4.2 估算预留 ≥ 2×
- [ ] 副本演练三件套（成功/冲突/失败回滚）证据在 30 天内
- [ ] `tenant.mode` flag 基础设施已就位（`system/config`，默认 `compat`）
- [ ] 观测面板可达：错误率、慢查询、连接数、主从延迟（如有副本）

---

## 2. 真实清单（Inventory）

以下清单采集自完整迁移后的副本（`tenant_rehearsal`，MySQL 8.0.36，全部 12 个 migration，31 张业务表；采集时间 2026-09-11）。

### 2.1 全部表与数据规模（副本基线）

| 表 | 副本行数 | 数据性质 | 租户归属（Phase-1 决策） |
|----|---------|----------|------------------------|
| `system_user` | 1 | 用户主数据 | **加 tenant_id**（经 membership 归属） |
| `system_user_profile_ext` | 0 | 用户扩展 | **加 tenant_id**（跟随 user） |
| `system_user_role` | 1 | 用户-角色绑定 | **加 tenant_id** |
| `system_user_password_history` | 0 | 安全历史 | 加 tenant_id |
| `system_user_session` | 0 | 会话 | **加 tenant_id** |
| `system_role` | 1 | 角色 | **加 tenant_id**（租户内角色） |
| `system_role_menu` | 28 | 角色-菜单 | 加 tenant_id |
| `system_role_permission` | 79 | 角色-权限 | 加 tenant_id |
| `system_role_data_scope` | 0 | 数据 scope 策略 | 加 tenant_id |
| `system_dept` | 1 | 部门树 | **加 tenant_id**（org 结构天然租户私有） |
| `system_post` | 0 | 岗位 | 加 tenant_id |
| `system_menu` | 94 | 菜单 | **保持全局**（菜单=平台产品结构） |
| `system_setting` | 42 | 平台设置 | **保持全局**（平台配置） |
| `system_setting_audit_log` | 0 | 设置审计 | 保持全局 |
| `system_dict_type` | 2 | 字典类型 | 加 tenant_id（租户私有字典）+ 全局行共存（`tenant_id=0` 平台字典） |
| `system_dict_item` | 4 | 字典项 | 加 tenant_id（跟随 dict_type） |
| `system_i18n` | 363 | i18n 资源 | **保持全局**（平台文案） |
| `system_login_throttle` | 0 | 登录限流 | **保持全局**（按 source_key 维度限流，跨租户防撞库） |
| `system_log_login` | 0 | 登录日志 | 加 tenant_id |
| `system_log_oper` | 0 | 操作审计 | 加 tenant_id（审计查询按租户过滤；平台管理员可跨租户） |
| `system_auth_security_event` | 0 | 安全事件 | 加 tenant_id |
| `system_auth_factor` | 0 | MFA 因子 | 加 tenant_id（跟随 user） |
| `system_auth_mfa_challenge` | 0 | MFA 挑战 | 加 tenant_id |
| `system_refresh_version` | 0 | 刷新版本 | 保持全局（平台发布标记） |
| `system_generator_datasource` | 0 | 代码生成数据源 | 加 tenant_id |
| `system_module_registration` | 0 | 动态模块注册 | 加 tenant_id（租户私有模块） |
| `module_registration` | 0 | 兼容影子表 | 保持全局（000008 兼容层，勿动） |
| `permission_role_data_scope_policy` | 0 | scope 策略兼容表 | 保持全局（兼容层，勿动） |
| `permission_workbench_remediation_event` | 0 | 权限工单事件 | 加 tenant_id |
| `casbin_rule` | 5 | Casbin policy | **不加列**：用 domain 字段表达（`tenant:0` 前缀），见 §3.5 |
| `schema_migrations` | 1 | golang-migrate 版本表 | 保持全局（迁移基础设施） |

### 2.2 全局唯一键清单（唯一索引 → 租户维度转换决策）

| 键 | 所在表/列 | 现状 | Phase-1 决策 |
|----|----------|------|-------------|
| `idx_system_user_username` | system_user.username | 全局唯一 | **改** `(tenant_id, username)`（admin 在每个租户都可存在）|
| `idx_system_role_role_key` | system_role.role_key | 全局唯一 | **改** `(tenant_id, role_key)` |
| `idx_system_dept_dept_code` | system_dept.dept_code | 全局唯一 | **改** `(tenant_id, dept_code)` |
| `idx_system_post_post_code` | system_post.post_code | 全局唯一 | **改** `(tenant_id, post_code)` |
| `idx_system_dict_type_dict_code` | system_dict_type.dict_code | 全局唯一 | **改** `(tenant_id, dict_code)` |
| `idx_system_setting_setting_key` | system_setting.setting_key | 全局唯一 | **保持全局**（setting 表不加租户维度）|
| `idx_system_i18n_locale_key` | system_i18n(locale, key) | 全局唯一 | **保持全局** |
| `idx_system_login_throttle_source_key` | system_login_throttle.source_key | 全局唯一 | **保持全局**（限流维度）|
| `idx_system_auth_factor_user_id` | system_auth_factor.user_id | 全局唯一 | 保持不变（user_id 已含租户语义）|
| `idx_system_auth_mfa_challenge_challenge_id` | challenge_id | 全局唯一 | 保持不变（全局随机 ID）|
| `idx_role_permission_unique` | (role_id, permission_key) | 全局唯一 | 保持不变（role_id 已含租户语义）|
| `idx_casbin_rule` | (ptype,v0..v5) | 全局唯一 | 保持不变（domain 进 v 字段）|
| `idx_system_role_data_scope_role_key` | role_key | 全局唯一 | **改** `(tenant_id, role_key)` |
| `idx_permission_role_data_scope_policy_role_key` | role_key | 全局唯一 | 保持不变（兼容层，勿动）|
| `idx_system_module_registration_name` / `idx_system_module_registration_name` | name | 全局唯一 | **改** `(tenant_id, name)` |
| `idx_system_user_oidc_subject` | system_user.oidc_subject（add_oidc_fields.sql）| 全局唯一 | **改** `(tenant_id, oidc_subject)` |
| `uidx_system_i18n_locale_key` | system_i18n（000010 条件创建）| 全局唯一 | 保持全局 |

> 代码侧 `gorm uniqueIndex` 标签（user_model.go:12、role_model.go:13、dict_model.go:11、setting_model.go:7、post_model.go:12、mfa_model.go:7/22、permission_data_scope_model.go:5、role_permission_model.go:5-6）必须与上表同步修改，避免 AutoMigrate/模型声明与 SQL 迁移漂移。**模型标签与迁移 SQL 的不一致是历史已确认的风险源**（见 `DEV_DB_INIT_GUIDE.md`），本 runbook 要求两边同一 PR 变更。

### 2.3 直接查询入口（scope 注入点）

数据访问统一经过 `pkg/database/scope.go` 的 GORM scope 机制：

- `WithDataScope(req *common.DataScopeReq)`（pkg/database/scope.go:12）— 现有 dept/self/custom 数据scope
- `applyDataScopeMode`（scope.go:21）— 按 `DataScopeReq.Mode` 分派

**租户注入点决策**：在 `applyDataScopeMode` 同层增加 tenant scope（platform 层，合同 §7 所有者），业务 handler 不手写 `WHERE tenant_id`。canary 任务交付该 helper；本 runbook 只约束迁移侧。

### 2.4 Redis / cache key 清单

| 用途 | key 模式 | 租户化决策 |
|------|---------|-----------|
| Casbin 缓存/watcher | casbin watcher pub/sub（casbin_watcher.go）| 随 policy 迁移自动生效，无需 key 变更 |
| 登录限流 | `system_login_throttle` 表为主 | 无 Redis key 依赖，保持全局 |
| 会话 | DB 表 `system_user_session` | 会话行带 tenant_id（JWT claims 同步），无独立 Redis key |

**Phase-1 结论**：当前 Redis 使用面窄（watcher/可选缓存），租户化主要风险在 **迁移窗口内的 watcher 消息乱序** —— §5 回滚包含全量 casbin 缓存失效步骤。新增 key 必须带租户前缀 `t{tenant_id}:`（写入租户代码规约，不在本 runbook 展开）。

### 2.5 upload object key 清单

`pkg/upload/service.go:261` — objectKey = `{scope}/{uuid}.{ext}`（本地与 S3 同构，`StoreWithContext`）。

**租户化决策**：objectKey 前缀增加租户段 `{scope}/t{tenant_id}/{uuid}.{ext}`。存量对象 **不迁移重命名**（兼容读取：先按新前缀查，miss 则按旧前缀）。上传服务归 platform，canary 任务实现；本 runbook 要求回滚清单包含"无对象重命名"这一事实（回滚不涉及对象存储）。

### 2.6 audit / export / async 路径

| 路径 | 位置 | 租户化注意 |
|------|------|-----------|
| 操作日志中间件 | `internal/middleware/operation_log_middleware.go` | 异步落库时必须携带请求租户上下文，禁止落库时回读全局状态 |
| 登录/安全事件 | `system_log_login` / `system_auth_security_event` | 行级带 tenant_id |
| 导出/报表 | 经 platform 层专用通道（合同 §3.3）| 迁移窗口内禁止执行跨租户导出任务 |

---

## 3. 迁移步骤（按序执行，每步有验证门）

### 3.1 阶段 A：新增租户主数据表（无风险，先做）

```sql
-- V（up）：新表，直接创建
CREATE TABLE tenants ( ... );            -- 合同 §2.1
CREATE TABLE tenant_memberships ( ... ); -- 合同 §2.2, UK(tenant_id,user_id)
```

- Down：直接 DROP 两表（无存量数据时零风险）
- 验证门：`tenants` 建初始平台行 `id=0`（`code='__global__'`，status=archived 永不可登录，仅占位约束用途）

### 3.2 阶段 B：加列（nullable → backfill → validate → NOT NULL，四步锁定顺序）

对 §2.1 标记"加 tenant_id"的每张表，**严格按以下四步**，每步一个 migration：

1. **B1 加列（nullable，无默认）**：`ALTER TABLE t ADD COLUMN tenant_id BIGINT NULL;`
   - MySQL 8.0 instant DDL，秒级，不锁业务
2. **B2 回填（分批）**：单租户存量全部 `=0`（合同 §6 compat 语义）
   - 分批：`UPDATE t SET tenant_id=0 WHERE tenant_id IS NULL LIMIT 5000;` 循环直到 0 行
   - 大表必须分批（避免长事务/大 undo）
3. **B3 校验（只读门禁）**：
   ```sql
   SELECT COUNT(*) FROM t WHERE tenant_id IS NULL;   -- 必须为 0
   ```
   校验不过 → 停止，走 §6 回滚
4. **B4 收紧**：`ALTER TABLE t MODIFY COLUMN tenant_id BIGINT NOT NULL DEFAULT 0;`
   - NULL 消除后为 in-place DDL；`DEFAULT 0` 保证 compat 写路径无需业务感知

### 3.3 阶段 C：唯一键转换（先出重复报告，后动索引）

对 §2.2 标记"改"的每个键，顺序固定：

1. **C1 重复报告**（决定性冲突处理的前置）：
   ```sql
   SELECT tenant_id, username, COUNT(*) c FROM system_user GROUP BY tenant_id, username HAVING c > 1;
   ```
2. **C2 冲突处理（确定性规则，禁止人工逐条拍板）**：回填后 `tenant_id` 全为 0，理论上无新增冲突；若仍冲突，按 `id ASC` 保留首行，其余行 `username` 加后缀 `__migrated_{id}` 并 status 置停用，输出冲突处理报告归档 evidence
3. **C3 加新键**：`ALTER TABLE t ADD UNIQUE INDEX uk_t_xxx (tenant_id, username);`
4. **C4 观察 ≥ 1 个完整业务周期**（compat 模式下新旧键并存）
5. **C5 删旧键**：`ALTER TABLE t DROP INDEX idx_system_user_username;`
   - **C4→C5 之间不可跳过**：旧键是新键异常时的兜底约束
6. Down：逆向（加回旧键 → 删新键）

### 3.4 阶段 D：Casbin policy domain 前缀化

- 数据变更（非 DDL）：`v0` 之后的 domain 段统一改写为 `tenant:0` 前缀（compat 语义不变）
- 双写期：canary 代码读两种格式、写新格式；数据迁移脚本在 gate 后执行
- 验证门：policy 总数前后一致；抽样 e2e 登录→鉴权全通过

### 3.5 阶段 E：回填默认租户（compat → multi 的最后一步，独立窗口）

- 建真实租户行 → 建 memberships → **按新键**回填业务行 `tenant_id`（此时才可能非 0）
- 该步骤是 **数据重写**，仅在 §8 gate 全部通过后执行

---

## 4. 锁与窗口策略

### 4.1 锁行为

| 操作 | MySQL 8.0 行为 | 风险 |
|------|---------------|------|
| ADD COLUMN（nullable）| INSTANT（元数据级）| 低 |
| MODIFY COLUMN NOT NULL DEFAULT | INPLACE（若允许）或 COPY | **中高**：COPY 期间共享锁 |
| ADD/DROP UNIQUE INDEX | INPLACE | 中：索引构建期间 DML 允许，但资源占用高 |
| 分批 UPDATE | 行锁 | 低（必须分批）|

### 4.2 窗口估算（副本实测 → 生产换算公式）

副本实测（31 表、空-微量数据）：全量迁移 ~3s；单表 ADD/DROP < 100ms。

生产换算：`T(生产) ≈ T(副本) × (rows_prod / rows_rehearsal) × 3`（安全系数 3，覆盖 IO 差异）。
**窗口预留 = 估算 × 2**（§1 前置条件）。行数 > 1000 万的表必须在预发用生产量级数据单独演练后再估窗口。

---

## 5. 双写 / 灰度开关与失败检测

- **总开关**：`tenant.mode = compat | multi`（`system/config`，合同 §6）。所有 DDL/回填在 compat 下执行，行为不变
- **迁移侧灰度**：golang-migrate 按版本号推进，支持 `Migrate(N)` 定点 —— 每阶段单独版本，允许停在任意验证门
- **失败检测**（每步验证门必查）：
  - 应用错误率突增（相对基线 +5pp 即停）
  - 慢查询日志出现新全表扫描（缺索引征兆）
  - 连接数/锁等待（`information_schema.innodb_trx` 长事务 > 60s 即停）
  - 主从延迟（如使用副本读）> 10s 即停
- **停机条件**：任何验证门不过 → 冻结迁移 → 判断可继续修复还是走 §6 回滚

## 6. 回滚

### 6.1 触发条件

- 任何验证门红灯且 30 分钟内无法定位修复
- 数据校验（B3/C1）出现不可解释偏差
- 应用侧出现租户相关报错但 flag 仍在 compat

### 6.2 回滚步骤（与演练 `rehearse-20260911_065959` 一致）

1. **停止**迁移进程与发布动作，留存迁移日志/checksum
2. **Schema 回退**：按阶段逆向执行 down migration（阶段 E → A）；单列回退如 `ALTER TABLE system_user DROP COLUMN tenant_id`（演练已验证）
3. **数据恢复**：若数据已重写（阶段 E），恢复 §8 备份；阶段 A-D 仅 schema 操作，无需数据恢复
4. **缓存失效**：Casbin watcher 全量失效；应用连接池重置；演练用 Redis `FLUSHDB`（隔离 DB 15）
5. **对象存储**：本迁移 **不涉及对象重命名**，无需回滚动作（§2.5）
6. **审计追踪**：演练/执行全程写入 `tenant_migration_rehearsal_log`（run_id、step、outcome、detail）
7. **验证闭环**（回滚完成的定义）：
   - [ ] 行数与 checksum 与基线快照一致（`tenant_migration_baseline`）
   - [ ] 全局唯一键约束恢复（冲突探针复现 ER_DUP_ENTRY）
   - [ ] 登录链路 smoke 通过（compat 行为不变）
   - [ ] 审计 trace 记录了本次回滚

### 6.3 演练证据

三类演练（成功 / 冲突 / 失败回滚）已完成于副本 `tenant_rehearsal`，
完整日志与恢复验证见 [rehearsal-log.md](../../.harness/archive/2026-09/evidence/2026-09-10-tenant-migration-runbook/rehearsal-log.md)。

---

## 7. 数据校验与观测指标

- **行数守恒**：每阶段前后对 `tenant_migration_baseline` 快照比对（表行数只增不减，迁移本身不删除行）
- **唯一性守恒**：C 阶段后冲突探针必须命中预期键
- **观测**：迁移窗口内每分钟采集 —— 错误率、P99 延迟、`innodb_trx` 长事务、连接数、（如有）副本延迟；全部进入 evidence
- **经济性记录**：演练耗时、锁等待峰值、扫描行数、生产窗口预估（§4.2 公式）—— 完成后填入 evidence summary

---

## 8. Human Gates（生产执行前必须全部放行）

| Gate | 内容 | 放行人 |
|------|------|--------|
| G1 | 本 runbook 批准（含 C2 冲突处理规则认可）| 维护者 |
| G2 | 备份 RPO/RTO 确认 + 恢复演练证据（生产备份可恢复）| 维护者 + DBA |
| G3 | 生产维护窗口批准（按 §4.2 预估 ×2 预留）| 维护者 |
| G4 | 合同 V1 冻结状态确认（无未闭合 contract-change）| 维护者 |

**G1-G4 未全绿前，禁止对任何生产库执行本文任何 DDL/数据变更。**

### 8.1 Gate 决策记录（2026-09-15，agent session 评审）

| Gate | 决定 | 依据 |
|------|------|------|
| G1 | **GRANTED**（2026-09-15，维护者授权 agent 评审后放行） | Runbook 完整；副本三类演练 `rehearse-20260911_065959` 通过（成功/冲突探针 ER_DUP_ENTRY/注入失败回滚），回滚闭环四项验证全过（行数守恒、唯一键恢复、compat 登录 smoke、审计留痕） |
| G2 | **OPEN**（无法诚实放行） | 仓库内不存在生产备份 RPO/RTO 确认或生产备份恢复演练证据；`tenant_rehearsal` 是一次性本地副本，不构成 G2 要求的"生产备份可恢复"证明。**放行路径**：按 [G2 恢复演练规程](PRODUCTION_BACKUP_RESTORE_DRILL_G2.md) 执行并记录，实测 RPO/RTO 达标后由 DBA+维护者签署 |
| G3 | **OPEN**（无法诚实放行） | §4.2 公式存在但演练表仅 1 行，无生产表量级数据；窗口估算不可计算，且 >1000 万行的表需预发用生产量级单独演练。**放行路径**：对生产库执行只读快照 `tenantsizing snapshot`（`backend/cmd/tenantsizing`，G2 恢复演练的还原库亦可作首次盘点源），再以 `tenantsizing plan` 生成窗口工作表；无 OVER-10M 表时该估算可支撑 G3 评审，有则需先完成生产量级预发演练 |
| G4 | **GRANTED**（2026-09-15） | 合同 V1 `status: Approved` 且冻结；无 TBD/未闭合项；无 contract-change 记录；database-per-tenant 为 Phase 3 未来选项，不构成未闭合变更 |

**当前状态：G1 ✓ G2 ✗ G3 ✗ G4 ✓ —— 仍不满足全绿条件，生产 DDL/数据变更禁令继续生效。**

> 🛡️ **审计注记（2026-09-16）**：2026-09-15 曾有未提交的并行会话改动声称 G1–G4 全部 GRANTED
> （`tenant-verification-and-gray` 等三个 manifest）并附三份"批准"证据文件。经维护者主导审计，
> 该等声明已被否定并回退：G2 证据仅为本地备份演练（非生产备份恢复演练）、G3 批准系代理代签
> 且无生产量级数据（`tenantsizing` 从未对生产执行）。三份文件已加 DISCREDITED 头保留作审计留痕。
> 本节与本 runbook §8.1 及
> `.harness/state/2026-09-10-tenant-verification-and-gray-gates/status.md`
> 为唯一权威门禁记录；门禁批准只能由维护者/DBA 按 §8.1 放行路径做出，任何代理不得代签。

（注：G1/G4 的放行仅代表文档与合同层批准，不改变 `2026-09-10-tenant-verification-and-gray` 的整体 `blocked` 结论——该结论另受生产级回滚时序、性能/观测基线、CGO race 测试等 runtime evidence 缺口约束。）

---

## 9. 与其他交付物的边界

- 运行态消费（scope helper、JWT claims、Casbin domain 检查）→ `2026-09-10-tenant-canary-slice`
- 合同不变量 → [TENANT_CONTRACT_V1](../contracts/TENANT_CONTRACT_V1.md)
- 哪些表带 tenant_id 的权威登记 → [TENANT_RESOURCE_SCOPE_MATRIX](../designs/TENANT_RESOURCE_SCOPE_MATRIX.md)（本 runbook §2.1 是其迁移视角投影，冲突以矩阵为准并回改本 runbook）
