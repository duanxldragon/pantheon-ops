# Pantheon Base 开发环境数据库初始化指南

## 问题描述

当你删除数据库后重新启动后端服务，点击左侧菜单时出现 **404 错误**，这是因为数据库表结构与代码模型不匹配导致的。

## 根本原因

### 字段名不一致问题

项目存在两套菜单表字段命名规范：

#### 旧版字段（迁移文件使用）

- `menu_type` - 菜单类型
- `permission` - 权限标识
- `visible` - 是否可见
- `cache` - 是否缓存

#### 新版字段（代码模型使用）✅

- `type` - 菜单类型 (M目录/C菜单/F按钮)
- `perms` - 权限标识
- `is_visible` - 是否可见 (1:是, 0:否)
- `is_cache` - 是否缓存 (1:是, 0:否)
- `is_external` - 是否外链
- `active_menu` - 高亮菜单路径
- `hide_in_nav` - 隐藏导航

### 执行流程分析

1. **默认模式** (`PANTHEON_AUTO_MIGRATE=false`):

   ```
   启动后端
   → 执行 golang-migrate 版本化迁移
   → 应用 000001_init_schema.up.sql (创建旧版表结构)
   → 应用 000002_align_runtime_schema.up.sql (添加新字段并迁移数据)
   → 代码尝试查询新版字段 ✅
   ```

   **问题**: 如果只执行了第一个迁移文件，表结构不完整

2. **开发模式** (`PANTHEON_AUTO_MIGRATE=true`):
   ```
   启动后端
   → 跳过 golang-migrate
   → GORM AutoMigrate 根据代码模型创建表结构
   → 直接创建新版表结构 ✅
   → 代码查询正常 ✅
   ```

## 解决方案

### 方案一：启用开发模式 AutoMigrate（推荐开发环境）✅

这是最简单、最可靠的方式，适合开发环境快速迭代。

#### Windows PowerShell

```powershell
# 设置环境变量
$env:PANTHEON_AUTO_MIGRATE="true"
$env:PANTHEON_ENV="development"
$env:PANTHEON_DSN="root:dev_password_change_me@tcp(localhost:3306)/pantheon_base?charset=utf8mb4&parseTime=True&loc=Local"

# 启动后端
cd d:\workspace\go\pantheon-platform\pantheon-base
go run ./backend/cmd/server/main.go
```

#### Windows CMD

```cmd
set PANTHEON_AUTO_MIGRATE=true
set PANTHEON_ENV=development
set PANTHEON_DSN=root:dev_password_change_me@tcp(localhost:3306)/pantheon_base?charset=utf8mb4^&parseTime=True^&loc=Local
cd d:\workspace\go\pantheon-platform\pantheon-base
go run .\backend\cmd\server\main.go
```

#### Linux/Mac

```bash
export PANTHEON_AUTO_MIGRATE=true
export PANTHEON_ENV=development
export PANTHEON_DSN='root:dev_password_change_me@tcp(localhost:3306)/pantheon_base?charset=utf8mb4&parseTime=True&loc=Local'
cd /path/to/pantheon-base/pantheon-base
go run ./backend/cmd/server/main.go
```

#### 使用提供的启动脚本

**Windows:**

```cmd
cd d:\workspace\go\pantheon-platform\pantheon-base\backend
start-dev.bat
```

**Linux/Mac:**

```bash
cd /path/to/pantheon-base/pantheon-base/backend
chmod +x start-dev.sh
./start-dev.sh
```

### 方案二：确保完整执行所有迁移文件

如果你需要使用生产模式的版本化迁移，需要确保：

1. **检查迁移状态**:

   ```sql
   SELECT * FROM schema_migrations;
   ```

2. **手动修复缺失的迁移**:

   ```sql
   -- 如果只有 version 1，需要手动执行 version 2
   INSERT INTO schema_migrations (version, dirty) VALUES (1, false);
   -- 然后重新运行后端，会自动应用 version 2
   ```

3. **或者重置迁移状态**:
   ```sql
   DROP TABLE IF EXISTS schema_migrations;
   -- 重新运行后端，会从 version 1 开始完整执行
   ```

### 方案三：使用 Docker Compose（推荐团队开发）

修改 `docker-compose.yml` 添加环境变量：

```yaml
services:
  backend:
    build:
      context: .
      dockerfile: Dockerfile
    environment:
      - PANTHEON_AUTO_MIGRATE=true # 开发环境启用
      - PANTHEON_ENV=development
      - PANTHEON_DSN=root:dev_password_change_me@tcp(mysql:3306)/pantheon_base?charset=utf8mb4&parseTime=True&loc=Local
      - PANTHEON_REDIS_ADDR=redis:6379
    ports:
      - "8080:8080"
    depends_on:
      - mysql
      - redis
```

然后启动：

```bash
docker-compose up -d
```

## 验证方法

启动后端后，检查以下内容确认数据库初始化成功：

### 1. 检查日志输出

**成功标志**:

```
INFO database migrations: all migrations applied successfully
# 或
INFO auto-migrate completed successfully
```

**失败标志**:

```
ERROR migration failed: ...
# 或
WARN module seed error module=system seed=menus error=...
```

### 2. 检查数据库表结构

```sql
-- 连接数据库
mysql -u root -p pantheon_base

-- 查看菜单表结构
DESCRIBE system_menu;

-- 应该包含以下字段：
-- id, parent_id, title_key, path, component, page_perm, perms, type,
-- icon, route_name, module, sort, is_visible, is_cache, is_external,
-- active_menu, hide_in_nav, created_at, updated_at
```

### 3. 测试 API 接口

```bash
# 获取当前用户菜单
curl http://localhost:8080/api/v1/auth/me \
  -H "Authorization: Bearer YOUR_TOKEN"

# 应该返回完整的菜单树结构，而不是 404
```

### 4. 前端测试

1. 登录系统（默认账号 admin / 密码 123456）
2. 点击左侧任意菜单项
3. 应该能正常访问，不再报 404 错误

## 常见问题排查

### Q1: 仍然报 404 错误

**可能原因**:

1. 数据库未正确初始化
2. 菜单数据未插入
3. 路由配置问题

**解决步骤**:

```sql
-- 1. 检查菜单表是否存在
SHOW TABLES LIKE 'system_menu';

-- 2. 检查是否有菜单数据
SELECT COUNT(*) FROM system_menu;

-- 3. 如果没有数据，重新运行后端（会执行 SeedMenus）
-- 4. 如果有数据但字段不对，删除表重新初始化
DROP TABLE system_menu;
-- 重启后端
```

### Q2: 菜单重复问题

**现象**: 左侧菜单显示重复项

**原因**: 多次执行种子数据导致重复插入

**解决**:

- 代码已内置 `CleanupObsoleteMenus` 和 `cleanupDuplicateMenus` 函数自动清理
- 如果仍有问题，清空菜单表重新初始化：

```sql
TRUNCATE TABLE system_menu;
TRUNCATE TABLE system_role_menu;
-- 重启后端
```

### Q3: Redis 连接失败

**现象**: 后端启动报错 Redis 连接失败

**解决**:

```bash
# 确保 Redis 正在运行
docker ps | grep redis

# 或者设置正确的 Redis 地址
export PANTHEON_REDIS_ADDR=localhost:6379
# 如果设置了密码
export PANTHEON_REDIS_PASSWORD=your_redis_password
```

### Q4: 数据库连接失败

**现象**: 后端启动报错数据库连接失败

**解决**:

```bash
# 1. 检查 MySQL 是否运行
docker ps | grep mysql

# 2. 检查 DSN 格式是否正确
echo $PANTHEON_DSN
# 应该是: user:password@tcp(host:port)/dbname?params

# 3. 测试连接
mysql -u root -p -h localhost pantheon_base
```

## 最佳实践

### 开发环境

1. ✅ 始终使用 `PANTHEON_AUTO_MIGRATE=true`
2. ✅ 使用提供的 `start-dev.bat` 或 `start-dev.sh` 脚本
3. ✅ 定期清理并重新初始化数据库测试完整性
4. ❌ 不要在生产环境使用 AutoMigrate

### 生产环境

1. ✅ 使用版本化迁移 (`PANTHEON_AUTO_MIGRATE=false`)
2. ✅ 确保所有迁移文件与代码模型一致
3. ✅ 在部署前测试完整的迁移流程
4. ✅ 备份数据库后再执行迁移

### 团队协作

1. ✅ 统一使用 Docker Compose 管理依赖
2. ✅ 在 `.env` 文件中管理环境变量
3. ✅ 提交迁移文件变更到版本控制
4. ✅ 在 README 中说明本地开发 setup 步骤

## 相关文档

- [主程序入口](../backend/cmd/server/main.go)
- [数据库迁移逻辑](../backend/pkg/database/migrate.go)
- [菜单模型定义](../backend/modules/system/iam/menu/menu_model.go)
- [初始化 SQL](../database/system_init.sql)
- [迁移文件目录](../backend/pkg/database/migrations/)
- [Docker Compose 配置](../docker-compose.yml)

## 技术支持

如果遇到问题，请检查：

1. 后端日志输出
2. 数据库表结构是否正确
3. 环境变量是否设置正确
4. 依赖服务（MySQL、Redis）是否正常运行

也可以参考项目根目录的 [README.md](../README.md) 获取更多帮助。
