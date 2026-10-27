# 租户初始化指南

**版本**: v2.0  
**适用范围**: pantheon-base Alpha阶段  
**最后更新**: 2026-10-13

---

## 概述

本指南说明如何在 pantheon-base 中初始化和管理租户。

**自 2026-10-13 起，建租户即完成初始化。** 一次 `POST /api/v1/system/tenant` 调用（带 `admin_username`）会在一个事务内产出：租户内角色、租户管理员用户、管理员角色绑定、`owner` membership、Casbin 域策略，以及平台设置的租户副本。租户创建后立即可登录、可管理。

### 当前状态

- ✅ 租户 CRUD API 已实现（`/api/v1/system/tenant/*`，平台能力）
- ✅ 建租户自动完成初始化（`ProvisionTenant`，幂等，可重跑修复）
- ✅ 租户内 IAM 全量隔离（用户 / 角色 / 部门 / 岗位 / 权限，18 张表带 `tenant_id`）
- ✅ 用户名 `(tenant_id, username)` 唯一——每个租户都可以有自己的 `admin`
- ✅ 租户管理员：**全量控制台，自动限定在本租户**；租户管理控制台本身仍是平台能力
- ✅ 会话级租户切换（`/api/v1/auth/tenants/*`）
- ✅ `X-Tenant-Id` 显式切换经真实租户校验（不存在 / 已暂停 / 已归档一律 403）

> 复验证据：`.tmp/verify_tenant_isolation_phase_g.py`（34/34 通过，自带清理）。

### 术语

| 角色 | 说明 |
|------|------|
| **平台主体**（platform population） | `tenant_id = 0` 的用户（如 `admin`），不属于任何租户，可跨租户管理 |
| **平台能力**（platform capability） | 持有全局（无 domain）Casbin 策略的主体。租户管理控制台要求此项 |
| **租户管理员** | 持有租户内角色 `tenant_admin`，全量控制台但只能看到本租户数据 |
| **租户成员** | 持有租户内角色 `tenant_member`，只读 |

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

### 1.3 确认核心表已带租户维度

```sql
-- 期望 18 张表带 tenant_id（IAM 全量 + 字典 + 设置 + 日志 + 会话）
SELECT COUNT(*) FROM information_schema.columns
WHERE table_schema = DATABASE() AND column_name = 'tenant_id';

-- 用户名唯一键应为 (tenant_id, username)
SHOW INDEX FROM system_user WHERE Key_name = 'uk_system_user_tenant_username';
```

---

## 二、创建租户

### 2.1 通过 API 创建（推荐，一步到位）

租户管理是**平台能力**，必须用平台主体（如 `admin`）调用。

```bash
curl -X POST http://127.0.0.1:8080/api/v1/system/tenant \
  -H 'Content-Type: application/json' \
  -b cookies.txt -H "X-CSRF-Token: $CSRF" \
  -d '{
    "code": "acme",
    "name": "Acme 公司",
    "plan": "free",
    "admin_username": "admin",
    "admin_password": "AcmeAdmin123!",
    "admin_nickname": "Acme 管理员"
  }'
```

| 字段 | 必填 | 说明 |
|------|------|------|
| `code` | ✅ | 租户编码，全局唯一，2–64 字符 |
| `name` | ✅ | 租户名称，2–128 字符 |
| `plan` | — | `free` / `basic` / `professional` / `enterprise`，默认 `free` |
| `admin_username` | — | 租户管理员用户名。**省略则只初始化角色与设置，不创建登录** |
| `admin_password` | — | 管理员密码，≥8 字符 |
| `admin_nickname` | — | 管理员昵称，默认同用户名 |

> ⚠️ 字段名是 **snake_case**（`admin_username`，不是 `adminUsername`）。传 camelCase 会被静默忽略，结果是一个没有管理员的租户。

创建成功后，管理员可用 `admin_username` + `admin_password` **立即登录**（登录时选择该租户，或依赖自动发现）。

### 2.2 创建产出什么

`ProvisionTenant` 在一个事务内写入：

1. 租户内角色：`tenant_admin`（全量控制台）、`tenant_member`（只读），`system_role.tenant_id = <租户>`
2. 租户管理员用户：`system_user.tenant_id = <租户>`，用户名在租户内唯一
3. 管理员角色绑定：`system_user_role`（带租户维度）
4. `owner` membership：`tenant_memberships`，登录签发闸门要求
5. Casbin 策略：`role:tenant_admin@tenant:<id>` / `role:tenant_member@tenant:<id>` → `/api/v1/*`
6. 平台设置的租户副本（`system_setting`，只补缺失键）

初始化是**幂等**的：重复执行只会补齐缺失部分，因此它同时是**修复入口**。

### 2.3 通过 SQL 创建（应急路径）

仅在 API 不可用时使用。注意：**只插一行 `tenants` 不会得到可用的租户**——必须自己补齐角色、管理员、membership 和 Casbin 策略，否则没人能登进去。

```sql
-- 1) 租户行
INSERT INTO tenants (code, name, status, plan, created_at, updated_at)
VALUES ('acme', 'Acme 公司', 'active', 'free', NOW(), NOW());

SET @tid = LAST_INSERT_ID();

-- 2) 租户内角色
INSERT INTO system_role (tenant_id, role_key, role_name, sort, status)
VALUES (@tid, 'tenant_admin', '租户管理员', 1, 1),
       (@tid, 'tenant_member', '租户成员', 2, 1);

SET @admin_role = (SELECT id FROM system_role WHERE tenant_id = @tid AND role_key = 'tenant_admin');

-- 3) 租户管理员（password 需为 bcrypt 哈希）
INSERT INTO system_user (tenant_id, username, password, nickname, status)
VALUES (@tid, 'admin', '<bcrypt-hash>', 'Acme 管理员', 1);

SET @uid = LAST_INSERT_ID();

-- 4) 角色绑定 + owner membership
INSERT INTO system_user_role (tenant_id, user_id, role_id) VALUES (@tid, @uid, @admin_role);
INSERT INTO tenant_memberships (tenant_id, user_id, role, status) VALUES (@tid, @uid, 'owner', 'active');
```

Casbin 策略无法用纯 SQL 可靠表达（走 synced enforcer），补完后重新调用一次 API 的初始化即可——`ProvisionTenant` 会补齐第 5、6 步。

### 2.4 使用测试工具（仅测试环境）

系统提供了测试工具`tenantmatrixdb`用于快速创建测试租户：

```bash
# 创建测试租户（101和202）
go run cmd/tenantmatrixdb/main.go up

# 清理测试租户
go run cmd/tenantmatrixdb/main.go down
```

**警告**: 此工具仅用于开发/测试环境，不应在生产环境使用。

### 2.5 租户字段说明

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| `code` | string | ✅ | 租户唯一标识符（英文小写，建议使用域名前缀如"acme"） |
| `name` | string | ✅ | 租户显示名称（如"Acme Corporation"） |
| `status` | string | ✅ | 租户状态：`active`（活跃）、`suspended`（暂停）、`archived`（归档） |
| `plan` | string | ❌ | 租户计划类型（`free` / `basic` / `professional` / `enterprise`） |

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

## 七、租户隔离模型与验证

### 7.1 隔离模型

**多租户 = 共享 schema + 行级 `tenant_id` 过滤。** `tenant_id = 0` 保留给平台/全局行（contract §2.3）。

隔离在两处同时成立，缺一不可：

| 层 | 机制 | 作用 |
|----|------|------|
| **授权** | Casbin 域主体 `role:<key>@tenant:<id>` | 决定"能不能调用这个接口" |
| **数据** | `tenant.WithTenantScope(ctx)` | 决定"能看见哪些行" |

租户管理员的授权面与平台 `admin` 一样宽（`/api/v1/*` 全方法），**收窄完全靠数据层**——这是刻意的设计：路径白名单会随路由新增而腐化。

### 7.2 服务端写法

每个 IAM 服务持有 `tenantCtx *tenant.Context`，并提供 `WithTenantContext` 克隆与 `scoped(db)` 助手：

```go
// 读：每次查询都从干净的句柄派生，避免 GORM 语句复用串味
func (s *UserService) scoped(db *gorm.DB) *gorm.DB {
    return db.Scopes(tenant.WithTenantScope(s.tenantCtx))
}

// 写：落到调用者的租户
func (s *UserService) tenantOwnerID() uint64 {
    if s.tenantCtx == nil || !s.tenantCtx.IsMulti() {
        return 0
    }
    return s.tenantCtx.TenantID
}
```

Handler 侧把 Gin 上下文里的租户交给服务：

```go
func (h *UserHandler) svc(c *gin.Context) *UserService {
    return h.service.WithTenantContext(tenant.FromGin(c))
}
```

> ⚠️ **GORM 语句复用陷阱**：不要把 `s.scoped(db)` 的结果在多个 `Where`/`Table` 调用间共享——GORM 会复用 statement，把前一次的条件和表名泄漏到下一次查询（实测症状：`Unknown column 'parent_id' in 'where clause'`）。**每次查询都要重新派生一个句柄。**

### 7.3 验证租户上下文传播

```go
import "github.com/duanxldragon/pantheon-base/backend/pkg/tenant"

func MyHandler(c *gin.Context) {
    ctx := tenant.FromGin(c)   // 中间件解析后存入，只读
    // ctx.Mode      -> "multi" / "compat"
    // ctx.TenantID  -> 实际租户 ID（平台全局为 0）
    // ctx.ResolvedBy-> "header" / "subject" / "compat-fallback"
}
```

### 7.4 验证数据隔离（实测命令）

```bash
# 租户 A 的管理员只看得到本租户用户（应只有它自己）
curl -b a.txt .../api/v1/system/user/list?page=1&size=100

# 租户 A 读租户 B 的用户 id -> 拒绝
curl -b a.txt .../api/v1/system/user/531        # 531 属于租户 B

# 租户 A 枚举租户 -> 403（租户管理是平台能力）
curl -b a.txt .../api/v1/system/tenant/list

# 租户 A 的角色列表只有租户内角色
curl -b a.txt .../api/v1/system/role/list
# -> role_keys = ["tenant_admin", "tenant_member"]
```

一键复跑全部 34 项隔离断言：

```bash
python .tmp/verify_tenant_isolation_phase_g.py
```

### 7.5 隔离覆盖的表（18 张）

用户 / 角色 / 部门 / 岗位 / 权限（IAM 全量）、字典类型 / 字典项、设置、用户会话、登录日志、操作日志、安全事件、MFA 挑战、用户-角色、角色-菜单、角色-权限、角色数据范围、数据范围策略、租户 membership。

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

## 九、API 参考（已实现）

### 9.1 租户管理（平台能力）

所有路由在 `/api/v1/system/tenant` 下，整组挂 `PlatformCapabilityMiddleware`：**只有持有平台级授权的主体可调用**。租户管理员即使拥有 `/api/v1/*` 的租户域权限，也会被该中间件拒绝（403 `tenant.platformOnly`）——这是有意为之，租户管理控制台不对外开放。

```bash
# 列表 / 详情
GET    /api/v1/system/tenant/list?page=1&size=20&keyword=&status=&plan=
GET    /api/v1/system/tenant/:id

# 创建（建租户即初始化，见 §2）
POST   /api/v1/system/tenant
{
  "code": "acme",
  "name": "Acme 公司",
  "plan": "free",
  "admin_username": "admin",
  "admin_password": "AcmeAdmin123!",
  "admin_nickname": "Acme 管理员"
}

# 更新（status 只接受 active / suspended / archived）
PUT    /api/v1/system/tenant/:id

# 删除（软删除；要求安全操作令牌，且租户不能有活跃成员）
DELETE /api/v1/system/tenant/:id

# 成员
GET    /api/v1/system/tenant/:id/members
POST   /api/v1/system/tenant/:id/members          { "user_id": 1002, "role": "member" }
DELETE /api/v1/system/tenant/:id/members/:userId
```

> ⚠️ 字段名一律 **snake_case**（`user_id`、`admin_username`）。camelCase 会被静默忽略。

### 9.2 会话与租户切换

这组路由在 `/api/v1/auth/tenants` 下，**不挂 Casbin**——权限来源是 membership 表本身（`contract §2.2`），而不是角色策略。任何已登录主体都可调用，返回内容严格限于其**自己**的活跃 membership。

```bash
# 当前请求解析到的租户上下文（含解析来源，contract §3.1）
GET  /api/v1/auth/tenants/current
# → { "mode": "multi", "tenantId": 24, "resolvedBy": "subject", "tenant": {...} }

# 我可切换到的租户 = 我的活跃 membership（前端据此决定是否显示切换器）
GET  /api/v1/auth/tenants/switchable
# → { "items": [ { "id": 24, "code": "...", "name": "...", "role": "owner", "isActive": true } ] }

# 会话级切换：保留 sessionId，重签 access/refresh 对
POST /api/v1/auth/tenants/switch
{ "tenantId": 25 }
```

切换的约束（全部实测）：

| 条件 | 结果 |
|------|------|
| 目标是活跃租户 + 我是活跃成员 | ✅ 200，后续请求解析到新租户 |
| 目标是当前租户 | ❌ 500 `tenant.switch.same` |
| 目标租户不存在 / 已暂停 / 已归档 | ❌ 403 `tenant.forbidden` / `tenant.suspended` / `tenant.archived` |
| 我不是目标租户的活跃成员 | ❌ 403 `tenant.forbidden` |
| compat 模式 | ❌ 403（只有一个作用域，无需切换） |

### 9.3 显式租户上下文：`X-Tenant-Id`

平台主体可用 `X-Tenant-Id` 显式指定作用域：

```bash
# 落到某个租户
curl -H "X-Tenant-Id: 24" .../api/v1/auth/tenants/current

# 落到平台全局作用域（tenant 0）
curl -H "X-Tenant-Id: 0" .../api/v1/auth/tenants/current
```

**该头部只对平台能力主体生效**（contract §3.1 来源 1），并且值会经过真实租户校验：

- 不存在 / 已暂停 / 已归档的租户 → **403**，不会静默回退
- 非平台主体发送该头部 → 忽略，仍解析到自己的租户 claim

### 9.4 登录与租户

```bash
# 未指定租户
POST /api/v1/system/login
{ "username": "admin", "password": "..." }

# 指定租户（多租户下的确定性方式）
POST /api/v1/system/login
{ "username": "admin", "password": "...", "tenantId": 24 }
```

解析规则：

| 情况 | 行为 |
|------|------|
| 指定了 `tenantId` | 凭证在该租户内查找：`(tenant_id, username)`。用户名在租户内唯一，因此每个租户都可以有自己的 `admin` |
| 未指定 `tenantId`，且我只有 1 个活跃 membership | 自动解析到该租户 |
| 未指定 `tenantId`，且有多个活跃 membership | 返回 `tenantSelectionRequired: true` + `tenantCandidates`，前端弹出租户选择器 |
| 未指定 `tenantId`，且我不属于任何租户 | 平台主体路径（`tenant_id = 0`） |
| 指定的租户我不是成员 | 403 `tenant.forbidden` |

> 未指定租户时的全局查找是**确定性**的：平台主体（`tenant_id = 0`）优先。这样同名用户不会因为行顺序而随机命中。


---

## 十、能力矩阵：平台主体 vs 租户管理员

| 能力 | 平台主体（`admin`，tenant 0） | 租户管理员（`tenant_admin`） | 租户成员（`tenant_member`） |
|------|------------------------------|------------------------------|------------------------------|
| 登录 | ✅ | ✅（需选择租户） | ✅ |
| 用户 / 角色 / 部门 / 岗位 控制台 | ✅ 全部租户 | ✅ **仅本租户** | ❌ 只读（`GET`） |
| 字典 / 设置 / 日志 / 会话 | ✅ 全部租户 | ✅ 仅本租户 | 只读 |
| 租户管理控制台（`/system/tenant/*`） | ✅ | ❌ **403 `tenant.platformOnly`** | ❌ |
| `X-Tenant-Id` 显式切换 | ✅ | ❌ 忽略 | ❌ 忽略 |
| 会话级租户切换 | ✅（按 membership） | ✅（按 membership） | ✅（按 membership） |
| 新建用户落在哪个租户 | `tenant_id = 0` | 本租户 | — |

**关键点**：租户管理员的"全量"是**授权面**全量（Casbin `/api/v1/*`），但**数据面**被 `WithTenantScope` 无条件收窄到本租户；租户管理控制台另由 `PlatformCapabilityMiddleware` 挡住。因此"全量控制台 + 自动限定本租户"是两套机制叠加的结果，不是靠路径白名单。

### 10.1 租户管理员不能做什么

- 枚举/创建/修改/删除租户
- 查看其他租户的任何 IAM 数据（用户/角色/部门/岗位/日志/会话）
- 通过 `X-Tenant-Id` 越权进入其他租户（头部只对平台能力主体生效）
- 通过 `/auth/tenants/switch` 切换到非成员租户

---

## 十一、最佳实践

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

## 十二、参考资料

- [TENANT_CONTRACT_V1.md](./contracts/TENANT_CONTRACT_V1.md) - 租户体系执行契约
- [TENANT_MIGRATION_RUNBOOK.md](./runbooks/TENANT_MIGRATION_RUNBOOK.md) - Schema迁移手册
- [TENANT_SYSTEM_REVIEW_2026-09-29.md](./TENANT_SYSTEM_REVIEW_2026-09-29.md) - 租户体系全局审查报告
- `backend/pkg/tenant/` - 租户运行时实现源码

---

## 十三、联系与支持

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
