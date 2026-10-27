# Pantheon Base 租户迁移执行指南

生成时间：2026-09-22  
目标：执行租户数据库迁移（000014/000015/000016）

## 📋 迁移概述

本次迁移将为以下表添加 `tenant_id` 列和相关索引：

| 迁移编号 | 目标表 | 变更内容 |
|---------|--------|---------|
| 000014 | `system_setting` | 添加 `tenant_id` 列 + 组合唯一键 `(tenant_id, setting_key)` |
| 000015 | `system_log_login`, `system_auth_security_event` | 添加 `tenant_id` 列 + 索引 |
| 000016 | `system_auth_mfa_challenge` | 添加 `tenant_id` 列 + 索引 |

**特性**：
- ✅ 所有迁移都是 **additive**（只添加，不删除）
- ✅ 使用 `DEFAULT 0` 确保现有数据不受影响
- ✅ 使用 `information_schema` 守护，幂等安全
- ✅ 在 `compat` 模式下不改变应用行为

## 🔍 预检查清单

在执行迁移前，请确认：

- [ ] 数据库可访问且有写权限
- [ ] 当前数据库备份可用（建议但非必需，G2 已豁免）
- [ ] 应用可以短暂停止服务（或保持运行，迁移是在线安全的）
- [ ] 已阅读并理解回滚流程

## 📝 执行步骤

### 步骤 1：检查当前迁移状态

```bash
cd D:\workspace\go\pantheon-platform\pantheon-base\backend
go run ./cmd/server migrate status
```

**预期输出**：显示当前已应用的迁移列表，000014/000015/000016 应该是 pending 状态。

### 步骤 2：（可选）创建数据库快照

如果您想要额外的保护（尽管 G2 已豁免）：

```bash
# 使用您的数据库工具导出 schema
# 例如：mysqldump -h localhost -u user -p --no-data pantheon > schema_backup_20260922.sql
```

### 步骤 3：执行迁移

```bash
cd D:\workspace\go\pantheon-platform\pantheon-base\backend
go run ./cmd/server migrate up
```

**预期输出**：
```
Applying migration 000014_tenant_settings...
✓ Migration 000014 applied successfully

Applying migration 000015_tenant_auth_logs...
✓ Migration 000015 applied successfully

Applying migration 000016_tenant_mfa_challenge...
✓ Migration 000016 applied successfully

All migrations applied successfully.
```

**如果失败**：
- 检查错误信息
- 确认数据库连接正常
- 确认用户权限足够（需要 ALTER TABLE 权限）
- 查看 `backend/logs/` 目录中的详细日志

### 步骤 4：验证迁移结果

```bash
# 1. 再次检查迁移状态
go run ./cmd/server migrate status

# 2. 验证列已添加（通过数据库客户端）
USE pantheon;

-- 检查 system_setting
DESCRIBE system_setting;
SHOW INDEX FROM system_setting;

-- 检查 system_log_login
DESCRIBE system_log_login;
SHOW INDEX FROM system_log_login;

-- 检查 system_auth_security_event
DESCRIBE system_auth_security_event;
SHOW INDEX FROM system_auth_security_event;

-- 检查 system_auth_mfa_challenge
DESCRIBE system_auth_mfa_challenge;
SHOW INDEX FROM system_auth_mfa_challenge;
```

**预期结果**：
- 每个表都有 `tenant_id BIGINT UNSIGNED NOT NULL DEFAULT 0` 列
- 每个表都有 `idx_<table>_tenant_id` 索引
- `system_setting` 有组合唯一键 `uk_system_setting_tenant_key (tenant_id, setting_key)`

### 步骤 5：运行冒烟测试

```bash
cd D:\workspace\go\pantheon-platform\pantheon-base\backend

# 运行单元测试
go test -short ./...

# 或者快速测试关键包
go test -short ./pkg/database
go test -short ./modules/system/config/setting
go test -short ./modules/auth/login
```

**预期结果**：所有测试通过，无失败。

### 步骤 6：启动应用并验证

```bash
# 启动后端
go run ./cmd/server

# 在另一个终端测试基本功能
# 1. 登录
# 2. 查看系统设置
# 3. 查看审计日志
# 4. 确认无错误日志
```

**验证要点**：
- ✅ 登录功能正常
- ✅ 系统设置读写正常
- ✅ 审计日志正常记录
- ✅ 无 SQL 错误或异常日志
- ✅ 性能无明显下降

## 🔄 回滚流程（如果需要）

### 选项 1：迁移回滚（推荐）

```bash
cd D:\workspace\go\pantheon-platform\pantheon-base\backend

# 回滚一个迁移
go run ./cmd/server migrate down 1

# 或者回滚到特定版本
go run ./cmd/server migrate down-to 000013
```

**效果**：
- 删除 `tenant_id` 列
- 删除相关索引
- 恢复到迁移前状态

### 选项 2：Feature Flag 回滚（即时）

如果迁移已成功但运行时有问题：

```bash
# 在配置文件或环境变量中
platform.tenant_mode=compat

# 清除租户相关缓存
redis-cli KEYS "t*:*" | xargs redis-cli DEL
```

**效果**：
- 应用行为恢复为单租户模式
- `tenant_id` 列存在但始终为 0
- 无数据丢失

### 选项 3：数据库恢复（G2 已豁免）

如果需要完全恢复到迁移前（接受数据丢失）：

```bash
# 恢复之前的快照（如果有）
mysql -h localhost -u user -p pantheon < schema_backup_20260922.sql

# 注意：这将丢失迁移后产生的所有数据
```

## 📊 迁移后监控

迁移完成后，建议监控以下指标 24-48 小时：

### 性能指标
- [ ] 登录延迟（< 500ms）
- [ ] 设置读写延迟（< 100ms）
- [ ] 审计日志写入延迟（< 50ms）
- [ ] 数据库连接池使用率（< 80%）

### 错误监控
- [ ] 应用错误日志（无 SQL 错误）
- [ ] 数据库慢查询日志（无新增慢查询）
- [ ] Redis 错误日志（无缓存错误）

### 数据完整性
- [ ] 审计日志连续性（无丢失记录）
- [ ] 系统设置一致性（无重复或丢失）
- [ ] 用户会话正常（无异常登出）

## 📈 预期迁移时间

基于 G3 评估（6,825 行数据）：

| 阶段 | 预计时间 |
|------|---------|
| 000014 system_setting (156 行) | 3-5 秒 |
| 000015 auth logs (~3,773 行) | 45-75 秒 |
| 000016 mfa challenge (234 行) | 5-8 秒 |
| **总计** | **~1-2 分钟** |

**生产环境**：如果数据量更大，时间会相应增加。建议在维护窗口执行。

## ✅ 迁移成功标志

迁移成功的判断标准：

1. ✅ `go run ./cmd/server migrate status` 显示 000014/000015/000016 已应用
2. ✅ 数据库表结构包含 `tenant_id` 列和相关索引
3. ✅ 单元测试全部通过
4. ✅ 应用启动无错误
5. ✅ 基本功能（登录、设置、审计）正常
6. ✅ 无性能明显下降

## 🚨 常见问题

### Q1: 迁移执行很慢怎么办？
**A**: 正常，特别是有大量审计日志时。`system_log_login` 表如果有几十万行，迁移可能需要几分钟。耐心等待，不要中断。

### Q2: 迁移失败说 "Duplicate entry"？
**A**: 说明 `system_setting` 表中有重复的 `setting_key`。迁移前需要清理重复数据：
```sql
-- 查找重复
SELECT setting_key, COUNT(*) FROM system_setting GROUP BY setting_key HAVING COUNT(*) > 1;

-- 手动删除重复（保留一个）
```

### Q3: 应用启动报 "column not found" 错误？
**A**: 代码和数据库不同步。确保：
- 代码已更新到最新（包含租户实现）
- 迁移已成功执行
- 应用已重启

### Q4: 想回滚但担心数据丢失？
**A**: 迁移回滚（`migrate down`）只删除列，不删除其他数据。但如果在 multi 模式下产生了租户数据，回滚会丢失这些数据。建议：
1. 先切回 `compat` 模式测试
2. 确认问题确实需要回滚
3. 再执行 `migrate down`

### Q5: 多久可以启用 multi 模式？
**A**: 建议步骤：
1. 迁移后保持 `compat` 模式 24-48 小时
2. 监控性能和错误日志
3. 在测试环境验证 multi 模式
4. 逐步为测试租户启用 multi
5. 全量启用前再监控 1-2 周

## 📞 获取帮助

如果遇到问题：

1. **检查日志**：`backend/logs/` 目录
2. **查看 Evidence**：`.harness/evidence/2026-09-10-tenant-*/`
3. **参考 Runbook**：`docs/runbooks/TENANT_MIGRATION_RUNBOOK.md`
4. **联系维护者**：duanxiaolong

## 📄 相关文档

- 租户合同：`docs/contracts/TENANT_CONTRACT_V1.md`
- 迁移 Runbook：`docs/runbooks/TENANT_MIGRATION_RUNBOOK.md`
- 生产就绪报告：`.harness/tasks/PRODUCTION_READY_REPORT_20260921.md`
- G2 豁免记录：`.harness/evidence/.../artifacts/g2-waiver-record-20260921.md`
- G3 窗口工作表：`.harness/evidence/.../artifacts/g3-window-worksheet-20260921-local.md`

---

**执行建议时间**：非高峰时段（如凌晨 2:00-4:00）  
**建议维护窗口**：2-3 小时（包含监控时间）  
**回滚窗口**：< 5 分钟（Feature Flag）或 < 1 小时（数据库回滚）

**准备好后，请按步骤执行。祝迁移顺利！** 🚀
