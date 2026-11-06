# 租户初始化流程实测报告

> ## ⚠️ 已修复 — 本文的结论已被后续整改推翻（2026-10-13 复验）
>
> 本文记录的是一份**问题清单**，不是当前行为。五项结论在整改后全部不成立，逐条订正如下：
>
> | # | 本文原结论 | 复验后的实际行为 |
> |---|-----------|-----------------|
> | 1 | 只有平台主体能建租户，无自动初始化路径 | ✅ 仍然只有平台主体能建租户（**这是正确的**：租户管理是平台能力）。但**建租户即完成初始化**——见第 2 条。 |
> | 2 | 建租户只产出一行 `tenants`，是空壳 | ❌ **已修复**。`CreateTenant` 现在调用 `ProvisionTenant`，一次事务内产出：租户内角色（`tenant_admin`/`tenant_member`）、租户管理员用户（`system_user.tenant_id`）、管理员角色绑定、`owner` membership、Casbin 策略 `role:<key>@tenant:<id>`，并把平台设置复制到租户。传 `admin_username`/`admin_password` 即一步得到可登录的租户。 |
> | 3 | 租户管理员登录后菜单 0 节点、接口全 403 | ❌ **已修复**。membership 角色已桥接到 Casbin domain 主体；`tenant_admin` 获得 `/api/v1/*` 全量控制台权限，且数据自动收窄到本租户。 |
> | 4 | `system_user` 没有 `tenant_id`，用户/角色/部门/岗位都是全局的 | ❌ **已修复**。迁移已给 18 张表补上 `tenant_id`，其中 IAM 全量（用户/角色/部门/岗位/权限）均按租户读写；`system_user` 的唯一键为 `(tenant_id, username)`。 |
> | 5 | admin 是超级管理员 | ✅ 仍然成立（**这是设计意图**）：`admin` 持有全局通配策略，可跨租户管理。租户管理员则是"本租户内的超级管理员"——全量控制台但自动限定在本租户，且被 `PlatformCapabilityMiddleware` 挡在租户管理控制台之外。 |
>
> **复验证据**：`.tmp/verify_tenant_isolation_phase_g.py` — 34/34 项通过（建租户即初始化 / 每租户同名 `admin` 可分别登录 / IAM 全量隔离 / 租户切换 / `X-Tenant-Id` 校验 / 跨租户写入不可见）。
>
> **本轮整改同时修掉 4 个真实缺陷**（本文写作时尚未暴露，因为当时功能还不存在）：
> 1. `copyPlatformSettingsToTenant` 漏写 NOT NULL 无默认值的 `group_key` 列 → MySQL 1364，**建租户整体失败**。
> 2. `activeTenantMemberPredicate` 未限定表名，`system_user` 加上 `tenant_id` 后 join 报 MySQL 1052 歧义 → 成员列表 500。
> 3. 登录按 `username` 全局查首个匹配行 → 租户 `admin` 命中平台 `admin`，报"密码错误"。现按 `(tenant_id, username)` 解析。
> 4. 数据范围中间件只认平台 `admin`，`tenant_admin` 无策略行 → 回退 custom+空部门集，**租户管理员看不到任何数据**。现同时认租户管理员，租户收窄由 tenant scope 无条件施加。
>
> 下文保留为**整改前的原始证据**，供对照。阅读时请以上表为准。

**日期**: 2026-10-13
**环境**: 本地 multi 模式（`platform.tenant_mode = multi`，后端 `:8080`，前端 `:5173`）
**方法**: 实机端到端验证（登录 / 建租户 / 建用户 / 绑成员 / 租户用户登录 / 越权探测）+ 数据库与源码交叉核对
**结论**: 多租户**尚未成型**。当前 multi 模式提供的是一套"租户声明（claim）"管道，而不是"租户隔离"。新建租户是一个空壳，租户管理员角色不产生任何权限，租户 claim 不约束任何控制台数据。

---

## 一、结论速览（直接回答五个问题）

| # | 问题 | 实测结论 |
|---|------|----------|
| 1 | 多租户下，租户谁来初始化？ | **只有平台级主体（admin）** 能建租户，走 `POST /api/v1/system/tenant`。**没有任何自动初始化路径**：所有 bootstrap 函数零调用者。 |
| 2 | 用管理员初始化，产出什么？ | **只有一行 `tenants` 记录**。没有租户管理员账号、没有角色、没有菜单、没有设置、没有成员。**新建租户是空壳，没人能登进去。** |
| 3 | 租户管理登录进去的页面长什么样？ | 需要 admin **手工两步**（建全局用户 + 手工加 membership）才能造出"租户管理员"。登录**能成功**（session 正确写入 `tenant_id`），但**菜单 0 个节点、所有控制台接口 403**——因为 membership 角色从不映射到权限。除非给它一个**全局角色**，那就直接变成平台管理员。 |
| 4 | 是不是和单租户一样？字段要显示 tenant_id 吗？ | **不一样，而且"显示 tenant_id"这个前提不成立**：`system_user` 根本没有 `tenant_id` 列，全库只有 **9 张表**有租户维度，用户/角色/菜单/部门/岗位全是全局的。 |
| 5 | admin 是超级管理员吗？ | **是**。admin 持有全局通配策略 `p: admin /api/v1/* GET\|POST\|PUT\|PATCH\|DELETE`，能看所有租户、按 id 读任意租户、在任意租户加删成员。 |

---

## 二、实测方法与原始证据

新建的两支探针（可复跑，自带清理）：

- `.tmp/verify_tenant_init_flow.py` — 21 项检查，覆盖建租户 / 建用户 / 绑成员 / 租户登录 / 租户视图 / admin 跨租户
- `.tmp/verify_tenant_scope.py` — 深挖 session claim、菜单树、dashboard 载荷

关键原始输出：

```
Q1 admin can create a tenant via the console        status=200 code=200
   created tenant id = 12
   response payload = {"id":12,"code":"probe-init-...","name":"...","status":"active",
                       "plan":"free","created_at":"...","updated_at":"..."}
Q2 create-tenant response carries NO admin/owner credentials   keys=['id','code','name','status','plan','created_at','updated_at']
Q2 new tenant has ZERO members (no owner, no admin)            items=0
Q3 admin can create a user                                     status=200 code=200
Q3 membership can be added with role=admin                     status=200 code=200
Q3 tenant user can log in                                      status=200 code=200
Q3 tenant user CANNOT log into a tenant it is not a member of   code=403 tenant.forbidden
```

---

## 三、Q1 — 谁初始化租户

**唯一入口：平台级主体调 `POST /api/v1/system/tenant`。**

该路由在 `backend/modules/system/system_modules.go:263`，整组挂了 `PlatformScopeTenantContextMiddleware` + `CasbinMiddleware`，需要 `system:tenant:create`。只有持有全局通配策略的 `admin` 满足。

**所有"自动初始化"代码都是死代码**——实测零调用者：

| 函数 | 位置 | 调用者 |
|------|------|--------|
| `InitializeTenantDefaults` | `tenant_service.go:479` | 无（源码自注：Phase 2.2 未交付，显式 no-op） |
| `InitializeTenantInfrastructure` | `tenant_bootstrap.go:112` | 无 |
| `EnsureDefaultTenant` | `tenant_bootstrap.go:25` | 无 |
| `AssignAdminToDefaultTenant` | `tenant_bootstrap.go:52` | 无 |
| `MigrateExistingUsersToDefaultTenant` | `tenant_bootstrap.go:78` | 无 |

后果：启动时**不建默认租户**（库里没有 `default` 租户）、**不迁移存量用户**、**不为租户播种**。`docs/TENANT_INITIALIZATION_GUIDE.md` 描述的 SQL 手工流程，是当前**唯一**可用的租户初始化方式（而且该文档已过时——它说 CRUD API 未实现，实际已实现）。

---

## 四、Q2 — 建租户产出什么（空壳）

`CreateTenant` 的实现只有一次 `INSERT INTO tenants`：

```go
// tenant_service.go:62-89
tenant := &Tenant{Code: dto.Code, Name: dto.Name, Status: "active",
                  Plan: dto.Plan, Metadata: dto.Metadata}
if err := s.db.Create(tenant).Error; err != nil { ... }
return tenant, nil
```

**没有**：初始管理员用户、租户内角色、菜单、设置、字典、membership。

实测：`GET /system/tenant/12/members` → `items=0`。

`docs/TENANT_INITIALIZATION_GUIDE.md` §9.1 把"自动初始化"列为**计划中**：

> **自动初始化**（计划）：
> - 创建租户记录
> - 创建初始管理员用户（如果不存在）
> - 添加管理员到租户（role=owner）
> - 初始化默认角色和权限
> - 复制租户级 settings 和字典

**这五项里只有第一项实现了。** 所以新建租户 = 一个没有任何人可以登录进去的空壳。

---

## 五、Q3 — 租户管理员：能存在，但形同虚设

### 5.1 造出一个"租户管理员"要三步手工操作

1. admin 在**全局**用户表建用户（`POST /api/v1/system/user`）——请求体 `UserCreateReq` **没有 `tenantId` 字段**（`user_dto.go:79-91`）
2. admin 手工加 membership：`POST /api/v1/system/tenant/{id}/members` `{"user_id":N,"role":"admin"}`
3. 该用户登录

### 5.2 登录是通的（claim 管道正常）

实测 session 表正确落库：

```
system_user_session: session_id=f533446d-... user_id=522 tenant_id=13
```

单 membership 用户不带 `tenantId` 登录会自动解析（`DiscoverDefaultTenant`）；带 `tenantId` 登录会被 `GateSessionIssuance` 校验；跨租户登录被拒（`403 tenant.forbidden`）。**这一层是健康的。**

### 5.3 但 membership 角色没有任何权限

`owner` / `admin` / `member` 三个角色值**从不映射到 Casbin**。全库搜索 `MembershipRole*` 的消费者，只有一处把它当显示值返回：

```go
// tenant_service.go:440
return membership.Role, true, nil     // 仅用于列表展示
```

`pkg/tenant/authz.go` 里**没有**任何 `membership.role → Casbin policy` 的桥接。

### 5.4 实测租户管理员看到的页面：空壳

role=owner 的租户用户登录后：

```
--- Casbin policies for this user's roles (should be none) ---
(no roles)

--- menu tree for the tenant user: 0 nodes ---
   tenant console (/system/tenant) present in tenant menu? False

tenant console list    status=200 code=403 msg=permission.denied
user list              status=200 code=403 msg=permission.denied
role list              status=200 code=403 msg=permission.denied
platform settings      status=200 code=403 msg=permission.denied
menu tree              status=200 code=200 msg=success   (但 0 节点)
dashboard summary      status=200 code=200 msg=success   ← 见 5.5
```

**菜单 0 个节点**，因为菜单树由角色权限驱动，而该用户没有角色。侧边栏是空的，页面是个空壳。

### 5.5 而且 dashboard 泄漏平台全局数字

`dashboard/summary` 对一个**零角色**的租户用户返回 **200**，且内容是**平台全局统计**：

```json
{"totalUsers":3,"enabledUsers":3,"totalRoles":2,"totalDepts":1,"totalPosts":1,
 "totalDictTypes":2,"totalI18nEntries":379,"visibleMenuCount":22,"activeSessionCount":2, ...}
```

两个原因叠加：

1. `backend/modules/platform/routes.go:29` —— dashboard 路由**只挂了** `TokenAuthMiddleware` + `PlatformScopeTenantContextMiddleware`，**没有 `CasbinMiddleware`**，所以任何已登录用户都能访问。
2. `backend/modules/platform/dashboard_service.go:320-326` —— `scopedTable` 只对 **4 张表**加租户过滤：

```go
switch tableName {
case "system_log_login", authSecurityEventTableName, operationLogTableName, "system_setting":
    q = q.Scopes(tenant.WithTenantScope(s.tenantCtx))
}
```

`system_user` / `system_role` / `system_dept` / `system_post` / 字典 / i18n 的计数**完全不按租户过滤**。

### 5.6 反证：给它一个全局角色，它就变成平台管理员

同一个租户用户（session `tenant_id=15`），只要在 `system_user_role` 里挂上全局 `admin` 角色：

```
menu nodes = 22
  top-level: ['/dashboard','/system/access','/system/user','/system/role','/system/permission',
              '/system/tenant','/system/menu','/system/org','/system/dept','/system/post',
              '/system/security','/system/login-log','/system/session','/system/operation-log']
  /system/tenant in menu? True

tenant console   status=200 code=200 total=2      ← 看到全部 2 个租户
user list        status=200 code=200 total=3      ← 看到全部 3 个用户
role list        status=200 code=200 total=2      ← 看到全部 2 个角色
```

**关键结论**：租户 claim（15）对这几个接口**没有任何约束作用**。可见性 100% 由全局角色决定。所以现在的"租户管理员"只有两种形态——**要么零权限空壳，要么等于平台管理员**，中间没有"租户内管理员"这一档。

---

## 六、Q4 — 和单租户不一样；"字段显示 tenant_id" 前提不成立

### 6.1 全库只有 9 张表有租户维度

```
system_auth_mfa_challenge
system_auth_security_event
system_dict_item
system_dict_type
system_log_login
system_log_oper
system_setting
system_user_session
tenant_memberships
```

**`system_user` 没有 `tenant_id` 列。** 用户的租户归属只存在于 `tenant_memberships` 多对多表。角色（`system_role`）、菜单（`system_menu`）、权限（`system_permission`）、部门（`system_dept`）、岗位（`system_post`）**全部是全局的**。

所以不存在"租户视图"这个东西，只有"平台视图 + 一个租户 claim"。在 UI 上给字段加 `tenant_id` 列没有意义——**大部分实体根本没有租户维度可显示**。

### 6.2 用户唯一键仍是全局的

```
SHOW INDEX FROM system_user:
  PRIMARY                id
  idx_system_user_username  username   Non_unique=0   ← 仍是 username 单列唯一
```

即：**两个不同租户不能有同名用户**（`acme` 和 `beta` 都不能有 `admin`）。这是共享 schema 多租户必须改成 `(tenant_id, username)` 的硬约束，现在没有。

---

## 七、Q5 — admin 是超级管理员

### 7.1 全局通配策略

```
p  admin  /api/v1/*  GET
p  admin  /api/v1/*  POST
p  admin  /api/v1/*  PUT
p  admin  /api/v1/*  PATCH
p  admin  /api/v1/*  DELETE
```

`subjectHasPlatformCapability`（`tenant_context_middleware.go:132`）用 `Enforce(roleKey, path, method)` 匹配全局策略，admin 恒为 platform-capable；admin 无 membership → `isPlatformPopulationSubject` 为真 → 在平台作用域路由上隐式回退到 tenant 0。

实测 admin 能力：

| 能力 | 实测 |
|------|------|
| 列出**所有**租户 | ✅ `total=2`，包含新建租户 |
| 按 id 读**任意**租户 | ✅ `GET /system/tenant/12` → 200 |
| 在**任意**租户加/删成员 | ✅ |
| `X-Tenant-Id` 切换作用域 | ✅ 生效（见 7.2） |

**是。admin 就是超级管理员，具备管理所有租户的能力。**

### 7.2 `X-Tenant-Id` 覆盖真的生效，但不校验目标租户

以 `system_dict_type`（有租户维度）为观测点：

```
no header (platform-global)      status=200 code=200 rows=2
X-Tenant-Id: 0                   status=200 code=200 rows=2
X-Tenant-Id: 13 (empty tenant)   status=200 code=200 rows=0    ← 切换生效
X-Tenant-Id: 999999 (missing)    status=200 code=200 rows=0    ← 不存在的租户也接受
X-Tenant-Id: abc (invalid)       status=200 code=403 tenant.forbidden
```

**缺陷**：`X-Tenant-Id: 999999`（不存在的租户）返回 200。中间件只解析数字、不校验租户是否存在/是否 active。安全影响有限（只是读不到数据），但语义上是错的——应当拒绝或至少落审计。

### 7.3 租户用户不能提权（这层是安全的）

```
tenant user + X-Tenant-Id: 0 (escalate to platform)  code=403 tenant.forbidden
tenant user + X-Tenant-Id: 1 (another tenant)        code=403 tenant.forbidden
```

因为 `subjectHasPlatformCapability` 要求全局策略，租户用户没有 → `HeaderAllowed=false` → 拒绝。✅

### 7.4 但"租户切换 API"全是 404

`tenant_switch.go:206` 定义了 `RegisterSwitchRoutes`，注册 `/tenants/switch`、`/tenants/switchable`、`/tenants/current`——**但从未被调用**：

```
/api/v1/tenants/current    → 404
/api/v1/tenants/switchable → 404
```

所以**没有任何租户切换 API**，`X-Tenant-Id` 是唯一途径，而前端不可能自己发这个 header。`docs/TENANT_MATURITY_FINAL_2026-09-29.md` 声称 "Tenant switching API with role validation" 已交付——**与实测不符**。

---

## 八、根因：迁移 000020 是"幽灵迁移"

`docs/TENANT_MATURITY_FINAL_2026-09-29.md` 声称成熟度 **96/100、Production-ready**，并列出：

> - ✅ All 23 core tables have tenant_id column
> - ✅ Composite unique indexes (tenant_id, *) on all tables
> - ✅ Migration 000020 with full rollback support
> - 16 tables upgraded: users, roles, menus, permissions, depts, posts, settings, user_role, role_menu, role_permission, data_scope, i18n_locale

**实测全部不成立。**

迁移文件存在、**且被记录为已应用**（`schema_migrations.version = 23, dirty = 0`，23 ≥ 20），但它的 DDL 没有落地：

| 000020 的目标表 | 现在有 `tenant_id`？ |
|---|---|
| `system_user` | ❌ |
| `system_role` | ❌ |
| `system_permission` | ❌ |
| `system_dept` | ❌ |
| `system_dept_ancestors` | ❌ |
| `system_dept_closures` | ❌ |
| `system_post` | ❌ |
| `system_role_menu` | ❌ |
| `system_role_permission` | ❌ |
| `system_role_data_scope` | ❌ |
| `system_user_role` | ❌ |
| `system_user_dept` | ❌ |
| `permission_role_data_scope_policy` | ❌ |
| `system_setting` | ✅（但来自 000014，非 000020） |

**14 个目标里 13 个没有 `tenant_id`。** 索引同样没变（`idx_system_user_username` 仍是单列唯一，不是 `uk_system_user_tenant_username`）。

**机制**：000020 的每条 DDL 都用 `information_schema` 守卫，写法是 `IF 表存在 AND 列不存在 THEN ALTER ELSE SELECT 1`。迁移头部注释自己警告过这个陷阱：

> Add tenant_id to system_user (guarded: the marker-driven bootstrap replay window models a minimal current schema that may not include this table — see 000012's header comment for the same replay hazard)

**这个 hazard 真的发生了**：迁移在"表还不存在"的重放窗口里执行，每条守卫都退化成 `SELECT 1`，但版本号照常盖章。之后真正的建表 DDL 创建了 `system_user` 等表，**不带 `tenant_id`**。于是迁移永久显示"已应用"，效果永久缺失。

这是整个问题的**根因**：租户隔离的地基（users/roles/permissions/depts 的租户维度）从未建成，但被记为已完成，并据此宣称 96/100 成熟度。

---

## 九、缺陷清单（按严重度）

### P0

| ID | 缺陷 | 证据 |
|----|------|------|
| P0-1 | **迁移 000020 是幽灵迁移**：记为已应用，14 个目标表里 13 个没有 `tenant_id`。租户隔离地基不存在。 | §8 |
| P0-2 | **新建租户是空壳**：`CreateTenant` 只插 `tenants` 一行，`InitializeTenantDefaults` 是显式 no-op。无管理员、无角色、无菜单、无成员。 | §4 |
| P0-3 | **所有 bootstrap 函数零调用者**：不建默认租户、不迁移存量用户、不播种。启动后无租户可用。 | §3 |
| P0-4 | **membership 角色无权限映射**：`owner`/`admin`/`member` 从不桥接到 Casbin。租户管理员菜单 0 节点、全接口 403。 | §5.3/5.4 |

### P1

| ID | 缺陷 | 证据 |
|----|------|------|
| P1-1 | **`system_user` 无 `tenant_id`，用户名全局唯一**：两个租户不能有同名用户。 | §6.1/6.2 |
| P1-2 | **dashboard 无权限门**：路由缺 `CasbinMiddleware`，零角色用户也能读，且泄漏平台全局统计。 | §5.5 |
| P1-3 | **dashboard 租户过滤只覆盖 4 张表**：用户/角色/部门/岗位计数全局化。 | §5.5 |
| P1-4 | **租户切换 API 全是 404**：`RegisterSwitchRoutes` 从未调用；前端无法切换租户。 | §7.4 |
| P1-5 | **`X-Tenant-Id` 不校验目标租户**：不存在的租户 id 返回 200。 | §7.2 |
| P1-6 | **`docs/TENANT_MATURITY_FINAL_2026-09-29.md` 与实测严重不符**（96/100 / 23 表 / 切换 API）。文档会误导决策。 | §8 |
| P1-7 | **`docs/TENANT_INITIALIZATION_GUIDE.md` 已过时**：称 CRUD API 未实现，实际已实现。 | §3 |

### P2

| ID | 缺陷 |
|----|------|
| P2-1 | `tenant_switch.go` 整个文件是死代码（`SwitchHandler` 无调用者）。 |
| P2-2 | `Tenant.Plan` 的注释允许 `standard`，但 DTO 的 `oneof` 只允许 `free basic professional enterprise`；初始化指南的示例用 `standard`。 |

---

## 十、设计建议：先决定"multi 到底意味着什么"

当前的 multi 模式实际上只做了"**租户 claim 管道**"，没做"**租户隔离**"。要回答"租户管理员页面该长什么样""字段要不要显示 tenant_id"，必须先选一个目标形态：

### 方案 A — 平台运营型（当前事实形态，收口即可）

承认**不存在租户内管理**。租户只是平台用来分组/计费/归属的标签，所有管理动作都由平台管理员完成。

- 删掉或明确标注"租户管理员"概念：`tenant_memberships.role` 降级为纯归属标记，不再暗示权限
- 补齐：建租户时自动建管理员用户 + owner membership（把 §9.1 的五项补齐）
- 修 P1-2/P1-3（dashboard 门禁与租户过滤）、P1-5（header 校验）
- **UI**：租户控制台只在平台侧出现；租户用户不显示控制台，也不显示任何 tenant_id 列
- 成本：低。能立刻消除"空壳租户"和"全局统计泄漏"

### 方案 B — 真隔离型（契约 §3 的原意）

让 multi 成为真正的隔离边界。**必须先把 P0-1 补上**，这是不可绕过的前置：

1. **修复幽灵迁移**：新增迁移，对 13 张缺失表补 `tenant_id` + 复合唯一键（`(tenant_id, username)` 等），并加一条 CI 断言——**迁移标记为 applied 时，其目标列必须存在**（这是本次事故的通用防线）
2. **桥接 membership 角色 → Casbin**：`owner`/`admin`/`member` 映射到 `role:<key>@tenant:<id>` 域内策略
3. **租户创建即初始化**：建租户 → 建管理员 → 加 owner → 播种租户内角色/菜单/设置
4. **租户切换**：把 `RegisterSwitchRoutes` 接上，给前端一个合法切换入口
5. **补齐租户作用域**：`WithTenantScope` 覆盖用户/角色/部门等，dashboard 全表过滤
6. **UI**：租户管理员看到的是**租户内的**用户/角色/设置，**不显示**全局 tenant_id 列（隔离边界内不需要）；平台管理员才需要"租户"这一列来区分归属

- 成本：高。涉及数据迁移、权限模型、UI 三块

### 我的建议

**先做方案 A 收口，把 P0-1 单独立项为技术债。** 理由：方案 A 能立刻消除当前所有"看起来坏了"的现象（空壳租户、空菜单、统计泄漏），且不阻塞；而方案 B 的第一步（修幽灵迁移）会改 13 张核心表的 schema，风险高、需要独立的数据迁移与回归计划，不适合和其他改动混在一起。

无论选哪个，**P1-6/P1-7 两份文档必须先修正或标注**——它们现在宣称的成熟度会直接误导后续决策。

---

## 十一、本次实测对数据库的改动

所有探针数据已清理：

```
tenants:                0 __global__ archived | 11 test active
tenant_memberships:     0 行
system_user:            无 probe_* 用户
schema_migrations:      23, dirty=0
```

> `id=11 code=test` 不是本次探针创建的，保留原样。
