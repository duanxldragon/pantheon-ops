# 租户初始化指南

**版本**: v1.0  
**适用范围**: pantheon-base Alpha阶段  
**最后更新**: 2026-09-29

---

## 概述

本指南说明如何在pantheon-base系统中初始化和管理租户。由于当前系统处于Alpha阶段，租户CRUD API尚未实现，本指南提供**临时的数据库操作方法**和**未来API的预期使用方式**。

### 当前状态

- ✅ 数据库schema已就绪（`tenants`、`tenant_memberships`表）
- ✅ 认证层已集成租户支持（登录、JWT、Session）
- ⚠️ 租户CRUD API未实现（计划中）
- ⚠️ 租户初始化逻辑不完整（需手动操作）

---

## 一、前置条件

### 1.1 检查租户模式

确认系统运行在正确的租户模式：

```bash
# 检查配置
grep "tenant_mode" config/application.yml

# 输出示例
platform:
  tenant_mode: compat  # 或 multi
```

**模式说明**：
- `compat`: 单租户兼容模式（所有数据`tenant_id=0`）
- `multi`: 多租户模式（需要显式租户标识）

### 1.2 验证数据库迁移

确保租户相关的数据库迁移已执行：

```sql
-- 检查租户表是否存在
SELECT table_name 
FROM information_schema.tables 
WHERE table_name IN ('tenants', 'tenant_memberships');

-- 检查全局占位租户
SELECT * FROM tenants WHERE id = 0;
-- 应返回: id=0, code='__global__', status='archived'
```

---

## 二、创建租户（当前方法）

### 2.1 通过SQL创建租户

由于租户CRUD API尚未实现，当前需要通过SQL手动创建：

```sql
-- 1. 创建租户
INSERT INTO tenants (code, name, status, plan, created_at, updated_at)
VALUES ('acme', 'Acme Corporation', 'active', 'standard', NOW(), NOW());

-- 获取新租户ID
SELECT id FROM tenants WHERE code = 'acme';
-- 假设返回 tenant_id = 101

-- 2. 创建初始管理员成员关系
-- 首先确保用户存在
SELECT id FROM system_user WHERE username = 'admin@acme.com';
-- 假设返回 user_id = 1001

-- 添加成员关系
INSERT INTO tenant_memberships (tenant_id, user_id, role, status, created_at, updated_at)
VALUES (101, 1001, 'owner', 'active', NOW(), NOW());
```

### 2.2 使用测试工具（仅测试环境）

系统提供了测试工具`tenantmatrixdb`用于快速创建测试租户：

```bash
# 创建测试租户（101和202）
go run cmd/tenantmatrixdb/main.go up

# 清理测试租户
go run cmd/tenantmatrixdb/main.go down
```

**警告**: 此工具仅用于开发/测试环境，不应在生产环境使用。

### 2.3 租户字段说明

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `code` | string | ✅ | 租户唯一标识符（英文小写，建议使用域名前缀如"acme"） |
| `name` | string | ✅ | 租户显示名称（如"Acme Corporation"） |
| `status` | string | ✅ | 租户状态：`active`（活跃）、`suspended`（暂停）、`archived`（归档） |
| `plan` | string | ❌ | 租户计划类型（预留字段，如"free"、"standard"、"enterprise"） |

---

## 三、管理租户成员

### 3.1 添加成员

```sql
-- 添加普通成员
INSERT INTO tenant_memberships (tenant_id, user_id, role, status, created_at, updated_at)
VALUES (101, 1002, 'member', 'active', NOW(), NOW());

-- 添加管理员
INSERT INTO tenant_memberships (tenant_id, user_id, role, status, created_at, updated_at)
VALUES (101, 1003, 'admin', 'active', NOW(), NOW());
```

### 3.2 成员角色说明

| 角色 | 权限级别 | 说明 |
|------|---------|------|
| `owner` | 最高 | 租户所有者，拥有所有权限（建议每租户1个） |
| `admin` | 高 | 租户管理员，可管理成员和设置 |
| `member` | 普通 | 普通成员，访问租户资源 |

### 3.3 移除成员

```sql
-- 软删除（推荐）
UPDATE tenant_memberships 
SET status = 'inactive', updated_at = NOW()
WHERE tenant_id = 101 AND user_id = 1002;

-- 硬删除（慎用）
DELETE FROM tenant_memberships 
WHERE tenant_id = 101 AND user_id = 1002;
```

### 3.4 查询用户的租户

```sql
-- 查询用户所属的所有租户
SELECT t.id, t.code, t.name, tm.role, tm.status
FROM tenants t
INNER JOIN tenant_memberships tm ON t.id = tm.tenant_id
WHERE tm.user_id = 1001 AND tm.status = 'active';
```

---

## 四、租户生命周期管理

### 4.1 暂停租户

```sql
-- 暂停租户（用户无法登录此租户）
UPDATE tenants 
SET status = 'suspended', updated_at = NOW()
WHERE id = 101;
```

**影响**：
- 该租户的用户无法登录
- 已登录的会话在刷新时会被拒绝（通过`GateSessionRefresh()`）
- 需要手动清理活跃会话（见4.4）

### 4.2 恢复租户

```sql
-- 恢复租户为活跃状态
UPDATE tenants 
SET status = 'active', updated_at = NOW()
WHERE id = 101;
```

### 4.3 归档租户

```sql
-- 归档租户（永久禁用）
UPDATE tenants 
SET status = 'archived', updated_at = NOW()
WHERE id = 101;
```

**注意**：归档是终态，合同规定归档租户不可恢复。

### 4.4 清理租户会话

暂停或归档租户时，需要手动清理活跃会话：

```sql
-- 查找租户的活跃会话
SELECT id, user_id, token 
FROM system_user_session 
WHERE tenant_id = 101 AND status = 'active';

-- 撤销会话
UPDATE system_user_session 
SET status = 'revoked', updated_at = NOW()
WHERE tenant_id = 101;
```

**TODO**: 未来会集成`RevokeUserSessionsInTenant()`自动化此流程。

---

## 五、多租户模式切换

### 5.1 从Compat模式切换到Multi模式

**前置条件**：
- ✅ 已创建至少一个租户
- ✅ 已为现有用户分配租户成员资格
- ✅ 已验证数据隔离机制

**步骤**：

1. **检查现有数据**
   ```sql
   -- 确认所有现有用户都有tenant membership
   SELECT u.id, u.username, COUNT(tm.tenant_id) as tenant_count
   FROM system_user u
   LEFT JOIN tenant_memberships tm ON u.id = tm.user_id AND tm.status = 'active'
   GROUP BY u.id, u.username
   HAVING tenant_count = 0;
   -- 应返回空（所有用户都有租户）
   ```

2. **更新配置**
   ```yaml
   # config/application.yml
   platform:
     tenant_mode: multi  # 从 compat 改为 multi
   ```

3. **重启服务**
   ```bash
   # 重启应用服务器
   systemctl restart pantheon-base
   # 或
   ./start-dev.sh
   ```

4. **验证切换**
   ```bash
   # 登录时应要求提供tenant_id（如果用户有多个租户）
   curl -X POST http://localhost:8080/api/v1/auth/login \
     -H "Content-Type: application/json" \
     -d '{
       "username": "admin@acme.com",
       "password": "...",
       "tenantId": 101
     }'
   ```

### 5.2 回滚到Compat模式

如果切换后出现问题，可以回滚：

```yaml
# config/application.yml
platform:
  tenant_mode: compat  # 改回 compat
```

**注意**：回滚后所有请求的`tenant_id`会被强制为0。

---

## 六、登录行为说明

### 6.1 Compat模式下的登录

```bash
# 登录请求（无需提供tenantId）
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin",
    "password": "..."
  }'

# 响应中 tenantId 始终为 0
{
  "token": "...",
  "user": {
    "id": 1,
    "username": "admin",
    "tenantId": 0
  }
}
```

### 6.2 Multi模式下的登录

**场景1：用户只属于一个租户**
```bash
# 登录请求（可省略tenantId，系统自动发现）
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin@acme.com",
    "password": "..."
  }'

# 响应
{
  "token": "...",
  "user": {
    "id": 1001,
    "username": "admin@acme.com",
    "tenantId": 101
  }
}
```

**场景2：用户属于多个租户**
```bash
# 登录请求（必须提供tenantId）
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin@multi.com",
    "password": "...",
    "tenantId": 101
  }'

# 如果不提供tenantId，响应会包含租户选择器
{
  "requireTenantSelection": true,
  "availableTenants": [
    {"id": 101, "code": "acme", "name": "Acme Corp"},
    {"id": 202, "code": "beta", "name": "Beta Inc"}
  ]
}
```

---

## 七、数据隔离验证

### 7.1 验证租户上下文传播

```go
// 在业务代码中获取当前租户ID
import "github.com/duanxldragon/pantheon-base/backend/pkg/tenant"

func MyHandler(c *gin.Context) {
    tenantID := tenant.ResolveForCanary(c)
    // Multi模式下应返回实际租户ID（如101）
    // Compat模式下始终返回0
}
```

### 7.2 验证数据隔离查询

```go
// 使用WithTenantScope确保查询隔离
db := tenant.WithTenantScope(c, dbConn)

var users []SystemUser
db.Find(&users)
// Multi模式下只返回当前租户的用户
// Compat模式下返回所有用户（tenant_id=0）
```

### 7.3 验证跨租户访问拒绝

```sql
-- 尝试跨租户查询（应被拒绝）
-- 假设当前登录租户是101，尝试访问租户202的数据
SELECT * FROM system_user WHERE tenant_id = 202;
-- 应通过ORM层的WithTenantScope被过滤
```

---

## 八、故障排查

### 8.1 用户无法登录（Multi模式）

**症状**: 登录返回"no active tenant membership"

**排查步骤**:
```sql
-- 1. 检查用户是否存在
SELECT * FROM system_user WHERE username = 'admin@acme.com';

-- 2. 检查用户的租户成员资格
SELECT tm.*, t.code, t.name, t.status
FROM tenant_memberships tm
INNER JOIN tenants t ON tm.tenant_id = t.id
WHERE tm.user_id = 1001;

-- 3. 确认租户和成员资格都是active状态
```

**解决方案**:
- 如果用户无成员资格，添加membership（见3.1）
- 如果租户状态非active，恢复租户（见4.2）
- 如果成员资格状态非active，更新状态

### 8.2 会话刷新失败

**症状**: Token刷新返回"membership verification failed"

**原因**: 用户的租户成员资格在会话创建后被修改（移除或暂停）

**解决方案**:
```sql
-- 检查成员资格状态
SELECT * FROM tenant_memberships 
WHERE user_id = 1001 AND tenant_id = 101;

-- 如果需要，恢复成员资格
UPDATE tenant_memberships 
SET status = 'active', updated_at = NOW()
WHERE user_id = 1001 AND tenant_id = 101;
```

### 8.3 租户上下文缺失

**症状**: 业务代码中`tenant.ResolveForCanary(c)`返回0（Multi模式下）

**排查步骤**:
1. 检查token是否包含tenantId
   ```bash
   # 解码JWT token查看payload
   echo "YOUR_ACCESS_TOKEN" | base64 -d
   # 应包含 "tenantId": 101
   ```

2. 检查中间件是否正确注入上下文
   ```go
   // 在handler中打印上下文
   tenantID, exists := c.Get("tenantId")
   log.Printf("TenantID: %v, Exists: %v", tenantID, exists)
   ```

**解决方案**: 确保请求经过`TokenMiddleware`处理。

---

## 九、未来API预览（计划中）

以下是计划实现的租户管理API（当前不可用）：

### 9.1 创建租户

```bash
POST /api/v1/tenants
Content-Type: application/json

{
  "code": "acme",
  "name": "Acme Corporation",
  "plan": "standard",
  "adminUsername": "admin@acme.com",
  "adminPassword": "secure_password",
  "adminEmail": "admin@acme.com"
}

# 响应
{
  "id": 101,
  "code": "acme",
  "name": "Acme Corporation",
  "status": "active",
  "createdAt": "2026-09-29T10:00:00Z"
}
```

**自动初始化**（计划）：
- 创建租户记录
- 创建初始管理员用户（如果不存在）
- 添加管理员到租户（role=owner）
- 初始化默认角色和权限
- 复制租户级settings和字典

### 9.2 管理成员

```bash
# 添加成员
POST /api/v1/tenants/101/memberships
{
  "userId": 1002,
  "role": "member"
}

# 移除成员
DELETE /api/v1/tenants/101/memberships/1002

# 修改角色
PATCH /api/v1/tenants/101/memberships/1002
{
  "role": "admin"
}

# 查询成员列表
GET /api/v1/tenants/101/memberships
```

### 9.3 查询当前租户

```bash
# 获取当前登录用户的租户信息
GET /api/v1/tenants/current

# 响应
{
  "id": 101,
  "code": "acme",
  "name": "Acme Corporation",
  "status": "active",
  "plan": "standard",
  "currentUserRole": "owner"
}
```

### 9.4 租户生命周期

```bash
# 暂停租户
POST /api/v1/tenants/101/suspend

# 恢复租户
POST /api/v1/tenants/101/resume

# 归档租户
POST /api/v1/tenants/101/archive
```

**自动化处理**（计划）：
- 暂停/归档时自动撤销活跃会话
- 归档前检查是否有活跃数据
- 级联清理租户相关token

---

## 十、最佳实践

### 10.1 租户命名规范

- **code**: 使用小写英文、数字、连字符，建议使用域名前缀
  - ✅ 好的例子：`acme`, `beta-corp`, `example-inc`
  - ❌ 不好的例子：`Acme`, `租户1`, `test@123`

- **name**: 使用正式的公司/组织全称
  - ✅ 好的例子：`Acme Corporation`, `Beta测试公司`

### 10.2 成员角色分配

- 每个租户至少保留1个`owner`角色
- 避免所有用户都是`owner`（权限过大）
- 根据职责分配合适的角色（owner/admin/member）

### 10.3 租户生命周期

- 测试环境使用测试工具创建租户
- 生产环境等待API实现后再创建租户
- 归档是终态，归档前确认数据已备份
- 定期清理inactive的memberships

### 10.4 数据迁移

当前系统处于Alpha阶段，如果需要从单租户迁移到多租户：

1. 保持compat模式运行
2. 等待Phase 1（数据隔离完善）完成
3. 参考`TENANT_MIGRATION_RUNBOOK.md`执行迁移
4. 迁移验证通过后切换到multi模式

---

## 十一、参考资料

- [TENANT_CONTRACT_V1.md](./contracts/TENANT_CONTRACT_V1.md) - 租户体系执行契约
- [TENANT_MIGRATION_RUNBOOK.md](./runbooks/TENANT_MIGRATION_RUNBOOK.md) - Schema迁移手册
- [TENANT_SYSTEM_REVIEW_2026-09-29.md](./TENANT_SYSTEM_REVIEW_2026-09-29.md) - 租户体系全局审查报告
- `backend/pkg/tenant/` - 租户运行时实现源码

---

## 十二、联系与支持

如果在租户初始化过程中遇到问题：

1. 查看故障排查章节（第八章）
2. 参考全局审查报告了解已知限制
3. 等待API实现后使用标准方法

**当前状态**: Alpha阶段，建议在测试环境使用  
**API实现进度**: 参见全局审查报告Phase 2路线图  
**预计可用时间**: Phase 2完成后（约2周）

---

**文档版本**: v1.0  
**最后更新**: 2026-09-29  
**下次更新**: API实现后（预计v2.0）
