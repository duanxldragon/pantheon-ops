# 租户全流程审查与验证报告（2026-10-14）

**范围**：pantheon-base 多租户完整链路 —— 租户初始化、租户内用户与角色、权限分配、数据权限隔离、生命周期、平台视角切换。
**方法**：代码走查（backend + frontend + 合同/设计文档） + 真栈 E2E（多租户 `multi` 模式） + 单测/集成测试（MySQL 真库） + 前端类型检查。
**结论**：租户主链路可用，E2E 62/62 通过。本轮共定位并修复 **3 个生产缺陷** 与 **1 类测试陈旧问题**，另有 1 项潜在风险记录在案（未改动）。

关联文档：

- 合同：[`docs/contracts/TENANT_CONTRACT_V1.md`](./contracts/TENANT_CONTRACT_V1.md)
- 设计：[`docs/designs/MULTI_TENANT_DESIGN.md`](./designs/MULTI_TENANT_DESIGN.md)
- 操作指南：[`docs/TENANT_INITIALIZATION_GUIDE.md`](./TENANT_INITIALIZATION_GUIDE.md)

---

## 一、租户全流程（说明）

### 1.1 角色与作用域模型

| 主体 | `tenant_id` | 能力来源 | 数据边界 |
| :--- | :--- | :--- | :--- |
| 平台主体 | `0` | 全局（无 domain）Casbin 策略，如 `admin` / `platform_ops` | 跨租户（平台管理 API 显式声明） |
| 租户管理员 | `>0` | 租户域角色 `role:tenant_admin@tenant:<id>` | 本租户（由 tenant scope 无条件收窄） |
| 租户成员 | `>0` | 租户域角色 `role:tenant_member@tenant:<id>`（仅 `GET /api/v1/*`） | 本租户、只读 |

租户上下文解析优先级（合同 §3.1）：`X-Tenant-Id`（平台能力）→ 主体 JWT claim → 平台作用域兜底 → compat 兜底。
数据隔离是**两层**：Casbin domain 授权 + `tenant.WithTenantScope(ctx)` 行级过滤。

### 1.2 生命周期八阶段

| 阶段 | 触发 | 关键实现 | 验收 |
| :--- | :--- | :--- | :--- |
| P1 初始化 | `POST /api/v1/system/tenant`（带 `admin_username`/`admin_password`） | `ProvisionTenant`：租户角色、管理员用户、角色绑定、`owner` membership、Casbin 域策略、设置副本，单事务 + 幂等 | ✅ |
| P2 租户管理员登录 | `POST /api/v1/auth/login` | 签发含 `tenant_id` claim 的会话；`/auth/tenants/current` 回显解析来源 | ✅ |
| P3 租户内建用户 | `POST /api/v1/system/user` | `system_user.tenant_id = <tenant>`，用户名 `(tenant_id, username)` 唯一 | ✅ |
| P4 角色与权限分配 | `POST /api/v1/system/role/members` 等 | `system_user_role(tenant_id, user_id, role_id)` + Casbin `role:<key>@tenant:<id>` | ✅ |
| P5 membership 与成员登录 | `POST /api/v1/system/tenant/:id/members` | `tenant_memberships` 决定可登录性；成员登录后按角色放行 | ✅ |
| P6 数据隔离 | 任意读 | Casbin 域 + tenant scope 双保险；跨租户按 id 读取返回空/404，不泄露存在性 | ✅ |
| P7 生命周期 | `PUT /api/v1/system/tenant/:id`（status）、`DELETE` | suspended/archived 拒绝登录与写入；有活跃成员时禁止删除 | ✅ |
| P8 平台视角与切换 | `X-Tenant-Id` 头、`POST /api/v1/auth/tenants/switch` | 平台主体可 pin 任意有效租户；非平台主体 403；同租户切换被拒 | ✅ |

### 1.3 关键 API 与错误码

| 场景 | HTTP | code |
| :--- | :--- | :--- |
| 主体无平台能力却带 `X-Tenant-Id` | 403 | `tenant.forbidden` |
| 无 membership / membership disabled | 403 | `tenant.forbidden` |
| 租户 suspended / archived | 403 | `tenant.suspended` / `tenant.archived` |
| 上下文缺失（非 compat） | 500 | `tenant.context.missing` |
| 租户控制台需要平台能力 | 403 | `tenant.platformOnly` |
| 切换到当前租户（no-op） | 500 | `tenant.switch.same` |
| 仍有活跃成员时删除租户 | 500 | `tenant.has_active_members` |

---

## 二、缺陷与修复

### 缺陷 1 —— 租户模式标志解析不确定（P0）

**现象**：数据库里 `platform.tenant_mode = multi`，但后端按 `compat` 解析，E2E 直接掉到 compat 兜底，平台/租户主体行为全线错乱。

**根因**（两处叠加）：

1. `platform.tenant_mode` 被 `copyPlatformSettingsToTenant` **逐租户复制**，`system_setting` 中同一 `setting_key` 出现多行；auth 的 `tenantModeLoader` 又是**不带租户过滤**的裸查询 → 命中哪一行取决于返回顺序。
2. 值被写成 JSON 带引号的 `'"compat"'`，读取侧未归一。

**修复**：

- `pkg/tenant/authz.go`：`FeatureFlagSettingKeyReader` 固定 `tenant_id = 0`，新增 `NormalizeFlagValue`（剥一层包裹引号），空值回落 `compat`。
- `modules/auth/module.go`：`tenantModeLoader` 改为委托上述租户受限读取器。
- `modules/system/iam/tenant/tenant_provision.go`：新增 `platformScopedSettingKeys = {"platform.tenant_mode","platform.app_mode"}`，复制设置时 `NOT IN` 排除，部署级开关永不进租户。
- `pkg/capability/capability.go`：`Load()` 增加 `tenant_id = 0`，`platform.app_mode` / `org.enabled` 统一按平台全局解析。
- `modules/system/iam/tenant/tenant_bootstrap.go`：`InitializeTenantInfrastructure` 以 `tenant_id = 0` 写入**裸值** `compat`。

**验证**：`pkg/tenant/feature_flag_test.go`（新增）+ `pkg/tenant` 测试通过；E2E 全程 `mode=multi`。

### 缺陷 2 —— 租户角色绑定写进 tenant 0（P0）

**现象**：租户内用户/管理员能登录，但角色加载为空 → 403。

**根因**：`system_user_role` 的插入语句**未带 `tenant_id`**，行落在 `tenant_id = 0`（平台命名空间）；而登录时角色加载器按 `system_user_role.tenant_id = <登录租户>` 过滤 → 查不到任何角色。

**修复**：

- `pkg/rbacbind/rbacbind.go`：`InsertUserRole` / `InsertUserRoles` / `EnsureUserRole` 全部携带租户命名空间，`(tenant_id, user_id, role_id)` 作为身份键。
- 调用方 `role_helper.go`、`user_username.go` 传入 `tenant.PlatformGlobalTenantID`；`role_service.go` 的 `AddRoleMembers` 传 `tenantOwnerID()`，`RemoveRoleMembers` 用非作用域删除顺带清理历史遗留行。

**验证**：`pkg/rbacbind` 测试（含租户命名空间断言）通过；E2E 中租户管理员/成员登录后权限正常。

### 缺陷 3 —— 租户成员读接口返回 500（P0）

**现象**：`tenant_member` 登录成功，但 `GET /api/v1/system/user/list` 返回 `500 common.serverError`（不是 403）。

**根因**：`DataScopeMiddleware` 对「非管理员且无 `system_role_data_scope` 行」的角色 **fail-closed**。`tenant_admin` 被 `hasAdministrativeRole` 豁免，`tenant_member` 没有 → 落入 fail-closed 分支。但 Casbin 明确给成员授予 `GET /api/v1/*`，且租户角色的行级收窄本就由 tenant scope 承担、读写拆分由 Casbin 承担，**平台级 dept 策略不适用于租户内角色**。

**修复**：

- `pkg/tenant/authz.go`：新增 `TenantScopedRoleKeys` 与 `IsTenantScopedRoleKey`，作为「租户内角色」的单一权威清单。
- `internal/middleware/data_scope_middleware.go`：新增 `hasTenantLocalRole`，`DataScopeMiddleware` 对租户内角色跳过 dept 策略查找（`IsAdmin` 语义不变）。
- `modules/system/iam/tenant/tenant_role_scope_test.go`（新增）：断言 provisioning 角色集 ⊆ 豁免集，新增租户角色若漏登记直接 CI 失败，而不是运行期 500。

**验证**：新增/修正 3 个中间件单测；E2E 由 **60/62 → 62/62**。

### 缺陷 4 —— 新增租户时管理员密码被前端/后端双重标准拒绝（2026-10-15 人工验证发现）

**现象**：控制台「新增租户」填写完整表单（含 6 位管理员密码）后提交，弹出 `请求失败，请稍后重试 (param.invalid)`，表单无任何字段级提示。

**根因**：同一个未发布改动集里，**同一份密码规则被写了两遍且不一致**：

| 位置 | 规则 |
| --- | --- |
| 后端 `CreateTenantDTO.AdminPassword` binding | `min=8`（硬编码） |
| 前端 `TenantList.tsx` 表单校验 | `< 6` 拒绝（硬编码） |
| 平台策略 `security.password_min_length` | 默认 **6**，可配置；用户创建/改密均以此为准 |

前端放行 6 位 → 后端 binding 拒绝 → 返回通用 `param.invalid`（无字段、无原因）。**与数据库迁移无关**：同一次探测中，密码 ≥ 8 位时租户可正常创建，且 provisioning 全量落库（角色 2、管理员用户 1、角色绑定 1、成员关系 1、设置复制 41）。

**修复**：让租户管理员密码遵守**同一个可配置策略**，而不是再写第三个数字。

- `tenant_model.go`：`AdminPassword` 去掉 binding 的 `min=8`（仅保留 `max=128` 上限）。
- `tenant_service.go`：新增 `validateAdminPassword` + `configuredPasswordMinLength`，在 `CreateTenant` 中按 `security.password_min_length` 校验；错误复用已翻译的 `user.update.error.password_too_short`。读取**固定 `tenant_id = 0`**（该设置逐租户复制，不过滤会命中随机副本，与 `platform.tenant_mode` 同类非确定性）。
- `tenant_platform_handler.go`：`failServiceError` 补 `common.ErrBadRequest → CodeParamInvalid` 分支。此前所有业务错误都被压成 `code:500`，导致「输入不合法」被报成服务端故障，且提示用户「稍后重试」——而重试永远不会成功。
- 前端 `TenantList.tsx`：改为从 `/auth/security` 读取**真实策略值**（`policy.passwordMinLength`）做校验与提示，前后端由构造上无法再分歧；i18n 文案改为带 `{{count}}` 的 5 语言版本。
- 新增单测 `TestCreateTenant_AdminPasswordHonoursConfiguredPolicy`：默认策略下 6 位通过、5 位拒绝；把策略调高到 10 后同样的 6 位被拒绝（证明是「读」而不是「写死」）；不填管理员时不约束密码。

**验证**：修复前 6 位 → `param.invalid`；修复后 6 位 → `code:200` 创建成功；4 位 → `code:400` + `user.update.error.password_too_short`。

### 缺陷 5 —— 软删除的租户永久占用其 `code`，控制台报裸 `tenant.conflict`（2026-10-15 人工验证发现）

**现象**（用户截图）：控制台「新增租户」填 `test-tenant`，弹出两个 toast —— `请求失败，请稍后重试 (tenant.conflict)` 与 `操作失败，请稍后重试`；同时「租户总数」显示 **0**，列表里看不到任何占用该编码的租户。

**定位**：`test-tenant` 是用户自己在 08:54 软删除的租户（id=30）。三条链路叠在一起：

1. `DeleteTenant` 只写 `status='deleted'`，**保留 `code`**；而 `tenants.code` 是 **UNIQUE** 列（合同 §2.1「全局唯一，创建后不可变」），因此该编码在存储层仍被占用；
2. `CreateTenant` 的唯一性检查**不区分 `status`**，命中后返回 `ErrTenantCodeExists`；
3. `tenant.conflict` 这个 key **在 5 个 locale 里都不存在**，前端 `translateMessage` 于是在 DEV 下回退成 `请求失败，请稍后重试 (tenant.conflict)` —— 既没说清原因，也没说该怎么做。

即「一个看不见的行占着一个看得见的编码」，且报错既不给原因也不给动作。

**这不是产品决策，是漏了一条已有约定。** `docs/designs/BACKEND.md` §7 早已规定：对「软删除 + 唯一键」实体（`system_user.username`、`system_role.role_key`、`system_post.post_code`、`system_dict_type.dict_code`…），删除时**统一先归档唯一标识再软删**，且**启动迁移自动修复历史遗留数据**。`tenants.code` 恰好漏在该清单之外。

**修复**（对齐既有约定，不新造机制）：

| 位置 | 改动 |
| :--- | :--- |
| `tenant_service.go` `DeleteTenant` | 单事务内先归档 `code` 为 `__deleted_tenant_<id>`，再写 `status='deleted'`。归档值前缀以下划线开头，而合同 §2.1 的编码文法 `^[a-z][a-z0-9-]{1,62}$` 不可能产生下划线，故归档值不会与任何可输入的编码冲突 |
| `tenant_service.go` `ReleaseDeletedTenantCodes` | 新增启动修复：把历史遗留的「已删除但仍持有原 code」的行归档。谓词是 `status='deleted' AND id <> 0`（该表软删是 status 制，`deleted_at` 不参与租户生命周期；显式排除 §2.3 预留占位行） |
| `tenant_service.go` `Bootstrap` + `system_modules.go` | tenant 模块此前**没有** `BootstrapFunc`，故修复无入口；补上并接入（与 role/user 模块同形） |
| `tenant_service.go` `CreateTenant` | 唯一性检查改为读取命中行的 `status`：活跃 → `ErrTenantCodeExists`；已删除 → 新增 `ErrTenantCodeReserved`。两种情况的处置不同（换编码 vs. 先清理），因此**必须分开报** |
| `tenant_platform_handler.go` | 拆开被合并的 key：`tenant.code_taken` / `tenant.code_reserved` / `tenant.member_exists`（原先三者共用一个 `tenant.conflict`） |
| 5 个 locale | 补齐 `tenant.code_taken`、`tenant.code_reserved`、`tenant.member_exists`、`tenant.not_found`、`tenant.has_active_members`、`tenant.delete.error.archive_code_conflict` |
| `TenantList.tsx` | 新增 `reportUnexpectedError`：拦截器已为 `RequestError` 弹过具体消息时不再补一个通用的「操作失败」，消除重复 toast（沿用 `Login.tsx` / `ModuleWizard.tsx` 既有模式） |

**真栈验证**（`.tmp/verify_tenant_code_release.py`，全部 PASS）：

| 场景 | 修复前 | 修复后 |
| :--- | :--- | :--- |
| 建 `test-tenant`（截图原样） | `tenant.conflict`（500 语义） | **200 创建成功** |
| 活跃租户占用同编码 | `tenant.conflict` | `400` + `tenant.code_taken` |
| 已删除租户占用同编码 | `tenant.conflict` | `400` + `tenant.code_reserved` |
| 释放后重建同编码 | — | **200 创建成功** |

**启动修复实测**：重启后 `test`(11) / `test-tenant`(30) 的 `code` 自动变为 `__deleted_tenant_11` / `__deleted_tenant_30`，`name` 保留，`__global__` 占位行未被触碰。E2E 与冒烟自身删除的租户（244/245/250/251）也留下归档值，说明**真实 DELETE 处理器**（含二次验证）走的就是新路径。

**相邻问题（仍开放）**：`pkg/common/error.go` 的信封码集没有 409 sentinel，因此冲突类错误统一映射为 `400 param.invalid`，靠 `message` 区分语义。若后续引入 409，本组映射需同步。

### 缺陷 6 —— 租户管理员「能登录，但登录后大面积报错」（2026-10-15 人工验证发现）

**现象**（用户口述）：`test-tenant` 这个用户**能登录**，但进去之后几乎每个页面都报错。后端日志给出的关键行：

```
POST /api/v1/auth/activity -> {"code":500,"data":null,"message":"tenant.context.missing"}
login -> code=200 user.id=684 roles=['tenant_admin']
```

即**登录成功但会话没有租户上下文**：`/auth/login` 返回 200，之后所有租户域接口 fail-closed 报 `tenant.context.missing`。用户看到的正是「登录成功 + 满屏报错」这种最难自证的组合。

**定位**：软删除租户**不会**删除它 provisioning 出来的 `system_user` 行，于是这些行成为孤儿。而全局凭据查找 `FindByUsername` 的排序是「平台主体优先、其次 id 升序」，**孤儿 id 更小**，就把活跃租户的同名用户遮蔽了：

| id | tenant_id | username | 该租户状态 |
| :--- | :--- | :--- | :--- |
| 684 | 252 | `test-tenant` | **deleted**（孤儿，id 更小 → 被选中） |
| 685 | 253 | `test-tenant` | active（真正的管理员，membership 170 = owner/active） |

登录因此以**孤儿 684** 的身份成功：它的租户已经没了，`DiscoverDefaultTenant` 解析不出任何 claim，于是签发了一个 **claim = 0** 的会话 —— 而 claim-0 会话在 `multi` 模式下对每个租户域路由都是 deny-by-default。**登录成功，处处失败。**

**两条修复**（缺一不可）：

| 位置 | 改动 | 作用 |
| :--- | :--- | :--- |
| `auth_repository.go` `FindByUsername` | 排除所属租户已软删除的主体：`tenant_id = 0 OR tenant_id IN (SELECT id FROM tenants WHERE status <> 'deleted')`；并把 `CASE WHEN tenant_id = 0 THEN 0 ELSE 1 END, id ASC` 作为确定性排序 | 让**活跃租户的用户**赢得 tie-break。这是「谁能成为登录主体」的定义问题：一个所属租户已消失的主体**无法承载任何会话**，因此不是登录候选 |
| `login_runtime.go` `refuseClaimlessTenantSubject` | `multi` 模式下解析不出 claim 时：平台主体（`tenant_id = 0`）仍得 claim 0；**带归属租户的主体则拒绝**（`tenant.forbidden`，合同 §5） | 堵住 fail-open 的另一半。原先是「登录成功 + 会话不可用」，现在是「登录时就明确说明没有租户上下文」 |

**为什么必须两条都做**：只做排除，历史遗留孤儿仍会在某些排序下胜出；只做拒绝，则活跃租户的管理员会被自己租户里的孤儿**连坐拒绝**——用户仍然进不去，只是报错从「满屏 500」变成「登录被拒」。两条合起来才是「活着的用户能进去，死掉的用户进不去」。

**顺带修掉的同类 i18n 缺口**：`pkg/tenant` 的 `tenant.forbidden` / `tenant.suspended` / `tenant.archived` / `tenant.context.missing` / `tenant.id.invalid` 这 5 个 key **在 5 个 locale 里都不存在**，而 `Login.tsx` 的白名单又只放行 `user.login.error.` / `auth.login.error.` / `auth.mfa.` 前缀。于是租户被拒登录时，用户看到的是通用「登录失败」，而不是真正的原因。已补齐 5 个 key × 5 locale，并把白名单抽成 `LOGIN_PASSTHROUGH_MESSAGE_PREFIXES`（新增 `tenant.`），与缺陷 5 的处理同形。

**真栈验证**（`.tmp/verify_tenant_orphan_login.py`，16/16 PASS；脚本用**真实 API** 造出孤儿行，而不是手工改库）：

| 场景 | 断言 |
| :--- | :--- |
| 建租户 A（管理员 X）→ 移除成员 → 走**真实 DELETE**（含二次验证）软删 A | A 的 `system_user` 行留存为孤儿，且 `id` 小于后来者 |
| 用**同一管理员用户名 X** 建租户 B | 前置条件成立：`orphan=693 < live=694`（旧 tie-break 确实会选中孤儿） |
| 用 X 登录（不带租户选择） | 200；`/auth/me` → **live 用户 694**（不是孤儿 693）；`/auth/tenants/current` → 租户 B；`POST /auth/activity` → **200，不再是 `tenant.context.missing`** |
| `/auth/tenants/switchable` | 只列出 B，且 `role=owner isActive=true`（与 `/current` 一致，缺陷 B 未回归） |
| 平台主体（`admin`，无 membership） | 仍正常登录并得 claim 0（未被拒绝逻辑误伤） |
| 有归属租户但 membership 被移除 | 登录被拒 `403 tenant.forbidden`，且**不签发任何会话 cookie** |

**用户自有数据的直接验证**：对用户实际数据执行修复后的解析，username `test-tenant` 现在解析到 **685（活跃租户 253）**，不再是孤儿 684 —— 即用户报的那条路径已经修好。

**回归影响**：`FindByUsername` 现在会读 `tenants` 表，因此**凭据查找依赖租户主表存在**。这一点已写进 `pkg/contracts/authuser` 的接口注释，并同步到 3 个测试夹具（`modules/auth/login` 的 `login_runtime_test.go` / `login_handler_service_test.go` / `login_service_throttle_test.go`，以及 `modules/system/iam/user` 的 `setupUserTestDB`）。新增 2 个单测固定新语义：孤儿被跳过（活跃租户用户胜出）、只有孤儿的用户名查不到。

### 缺陷 7 —— 租户管理员登录成功后处处 `permission.denied`（2026-10-15 人工验证发现）

**现象**（用户口述）：用 `test-tenant` 登录成功，之后报 `permission.denied`。这是缺陷 6 修好之后的**下一个**故障：会话现在正确带上了租户 253 的 claim，于是请求进到了授权层，而授权层把它拒了。

**定位**：租户 253 的角色与绑定**完全正常**，唯独 Casbin 策略是空的：

| 检查项 | 实际值 | 结论 |
| :--- | :--- | :--- |
| `system_role`（tenant 253） | 576 `tenant_admin` / 577 `tenant_member` | 正常 |
| `system_user_role` | `(685, 576, 253)` | 正常 |
| `tenant_memberships` | `(170, 253, 685, owner, active)` | 正常 |
| `system_role_permission`（role 576） | **97 条**（与平台 admin 一致） | 正常 |
| `system_role_menu`（role 576） | 22 条 | 正常 |
| `casbin_rule`（`@tenant:253`） | **0 条** | **缺失** |

于是 `role:tenant_admin@tenant:253` 这条主体在 Casbin 里**不存在任何规则**，而 `internal/middleware/casbin_middleware.go` 在无匹配规则时返回 `403 permission.denied`。登录成功、角色正确、权限键齐全、菜单齐全，却每个接口都被拒 —— 从 UI 完全无法自证原因。

**为什么策略会丢**：`ProvisionTenant` 会写这些策略（`grantTenantRolePolicies`），但**没有任何地方重新推导它们**：

1. `casbin_rule` **没有** `tenants` 外键，数据库层面无法级联保护；
2. 合同把「硬删租户」定为**运维动作**，其步骤是**手工** `DELETE FROM casbin_rule WHERE v0 LIKE 'role:%@tenant:<id>'`（runbook §2b）—— 一次作用域写宽的清理就会把**活跃**租户的策略一起删掉；
3. 早于该 provisioning 步骤创建的租户本来就没有策略。

这三条叠加的结果是：租户可以长期处于「能登录、无权限」的状态，而没有任何机制会把它修回来。

**修复**：新增 `Service.EnsureTenantRolePolicies()`，挂进 tenant 模块的 `Bootstrap`（与缺陷 5 的 `ReleaseDeletedTenantCodes` 同形，是「启动迁移也会自动修复历史遗留数据」规则的另一半）。

| 设计点 | 取值 | 理由 |
| :--- | :--- | :--- |
| 作用域 | **仅 `status='active'` 的租户**，且排除 id=0 预留占位行 | 挂起（`suspended`）才是**受支持**的租户停用手段（登录门会拒绝挂起租户），修复**不能**和它对抗；因此挂起/归档/已删除租户一律不发策略 |
| 方向 | **只增不删**（`AddPolicy` 对已存在的规则是 no-op） | 修复恢复的是 `tenantRoleDefinitions` 定义的那一套；运维自己加的额外规则必须保留，已由单测固定 |
| 幂等 | 是 | Bootstrap 每次进程启动都跑 |
| Casbin 未初始化时 | 静默 no-op | 单测夹具或早退进程没有 enforcer 可修 |

**关键性质**：这是一次**纯增量**的启动修复，不改变任何请求路径的判定逻辑 —— 它只把「本该存在但被删掉的」策略补回去。

**真栈验证**（`.tmp/verify_tenant_policy_repair.py`，两阶段；脚本用**真实 API + 真实 DELETE 路径**造状态）：

| 阶段 | 断言 | 结果 |
| :--- | :--- | :--- |
| break | 建租户 P → provisioning 写入策略 | `policies=6` |
| break | P 的管理员登录并访问控制台路由（`/dashboard/summary`、`/system/user/list`、`/system/role/list`） | 全部 **200** |
| break | 按 runbook §2b 的方式清掉 P 的域策略 | `policies=0` |
| break | 清库后**正在运行的**进程仍放行（enforcer 有内存副本 + Redis watcher） | 说明策略丢失**只在重载后**才生效 —— 这正是「启动修复」是正确的修复位置的原因 |
| verify | 重启后端（含修复）后 P 的策略 | **恢复为 6** |
| verify | P 的管理员登录并访问同一组控制台路由 | 全部 **200** |
| verify | **用户自己的 `test-tenant`(253)** | **恢复为 6** |

**启动日志实证**：

```
INFO tenant/tenant_service.go:491 tenant policy repair restored missing Casbin domain policies
     {"tenants": 2, "policiesRestored": 12}
```

（2 个活跃租户 × 6 条 = 12，与 `tenantRoleDefinitions` 完全吻合。）

**单测**：新增 `tenant_policy_repair_test.go`（6 例）：为活跃租户恢复全部策略、幂等、**跳过非活跃租户**、**不碰预留占位行**、**不删除运维已有策略**、无 enforcer 时不报错。

**回归**：`modules/system/iam/tenant` 全包 ok（含新 6 例）；Go 全量包全 ok；E2E `verify_tenant_full_flow.py` **62/62**；build / gofmt / vet 干净。

### 相邻问题（已随本轮修复）

- `failServiceError` 原先把「未找到 / 冲突 / 仍有活跃成员」一律压成 `code:500`（既掩盖原因，又提示「稍后重试」——而重试永远不会成功）。现已按 `pkg/common/error.go` 的 `errors.Is` 约定分类：`ErrBadRequest→param.invalid`、`NotFound→not_found`、冲突类→各自 key、`HasActiveUsers→tenant.has_active_members`。

### 测试陈旧问题（随修复暴露）

复验过程中发现若干测试未跟上本轮的租户化改动（**非产品缺陷**），一并修正：

| 位置 | 问题 | 处理 |
| :--- | :--- | :--- |
| `internal/middleware/tenant_hostile_smoke_test.go`、`tenant_platform_scope_test.go` | `X-Tenant-Id` 新增「真实租户校验」后，用例仍传 `nil` 库并期望 pin 一个不存在的租户成功 | 注入带真实 active 租户的测试库 |
| `internal/middleware/data_scope_middleware_test.go` | 断言写成「HTTP 非 200 即拒绝」，而框架 fail-closed 是 **HTTP 200 + `code:500` 信封**；且误用 `tenant_member` 当作「普通平台角色」 | 改为断言 `code:500`/`common.serverError` 且 handler 未执行；普通角色改用 `viewer` |
| `modules/system/iam/role/role_service_test.go`、`modules/auth/login/login_preferences_contract_test.go` | `system_user_role` / `system_role` 测试 DDL 缺 `tenant_id` | 补列 |
| `modules/auth/login/login_tenant_gate_test.go`、`modules/system/iam/menu/menu_service_test.go` | `system_setting` 测试 DDL 缺 `tenant_id`（标志/能力读取已改为租户受限） | 补列并按 `tenant_id = 0` 播种 |
| `pkg/database/migrate_test.go` | 基线由 20 提到 24 后，`currentRuntimeSchemaVersion+1`(=25) 已无对应迁移文件，`Up()` 直接报错 | 改为播种基线版本并断言基线 |
| `.tmp/verify_tenant_init_flow.py` | 断言陈旧：API 路径 `/api/v1/tenants/current`、字段 `tenant_id`，以及「平台创建的 user 可用显式 tenantId 登入租户」的错误模型 | 修正路径/字段，并按设计语义改写 4 条断言 |

---

## 三、验证证据

### 3.1 E2E / 冒烟（真栈，`multi` 模式）

| 套件 | 覆盖 | 结果 |
| :--- | :--- | :--- |
| `.tmp/verify_tenant_full_flow.py` | P1–P8 全链路（含 P7 真实 DELETE + 二次验证） | **62/62** |
| `.tmp/verify_tenant_isolation_phase_g.py` | 租户隔离 | **34/34** |
| `.tmp/verify_tenant_remediation.py` | 控制台整改 | **30/30** |
| `.tmp/verify_tenant_menu_fix.py` | 菜单可见性（multi/compat） | **18/18** |
| `.tmp/verify_frontend_viewpoint_flow.py` | 前端视角切换链路 | **22/22** |
| `.tmp/verify_tenant_init_flow.py` | 初始化/登录/跨租户 | **21/21** |
| `.tmp/verify_tenant_code_release.py` | 编码归档/复用（缺陷 5） | **6/6** |
| `.tmp/verify_tenant_orphan_login.py` | 已删租户孤儿不再遮蔽活跃用户；无 membership 主体被拒（缺陷 6） | **16/16** |
| `.tmp/verify_tenant_policy_repair.py` | 域策略被清后启动修复恢复；控制台路由重新放行（缺陷 7） | **9/9**（break 5 + verify 4） |
| `npm run test:smoke:tenant` | 租户浏览器冒烟矩阵 | **10/10** |
| `npm run test:smoke:tenant:viewpoint` | 租户视角切换冒烟 | **5/5** |
| `smoke-core/auth-tenant-picker.spec.ts` | 登录页租户选择器 | **3/3** |

### 3.2 Go 单测/集成测试（MySQL 真库）

```
ok  backend/pkg/tenant
ok  backend/pkg/rbacbind
ok  backend/pkg/database
ok  backend/internal/middleware
ok  backend/modules/auth/login
ok  backend/modules/auth/mfa
ok  backend/modules/auth/security
ok  backend/modules/auth/session
ok  backend/modules/system/iam/tenant
ok  backend/modules/system/iam/user
ok  backend/modules/system/iam/role
ok  backend/modules/system/iam/menu
ok  backend/modules/system/iam/permission
ok  backend/modules/system/audit
```

`backend/modules/system/iam/tenant` 新增 `tenant_code_lifecycle_test.go`（11 例），覆盖：删除释放编码、活跃占用 vs 已删除占用的区分、启动修复、修复幂等、预留占位行不被触碰、归档值冲突时的兜底、复用后仍完整 provisioning、已释放编码不再被 `GetTenantByCode` 解析。

### 3.3 前端

- `tsc -b`（type-check）通过，无错误。
- `check:i18n-hardcode`、`check:i18n-generated-scope`、`check:menu-contract` 三关全绿。
- 租户相关前端面：`core/tenant/viewpoint.ts`（视角状态）、`api/request.ts`（`X-Tenant-Id` 注入 + 错误消息翻译回退）、`TenantSwitcher`（membership 切换）、`TenantViewpointSwitcher`（平台视角 pin）、`system/iam/tenant/TenantList.tsx`（租户控制台）。安全语义全部在服务端，前端不承担隔离职责。

---

## 四、遗留与风险

1. **`system_role_data_scope` 的租户维度未闭环（潜在，未改动）**
   该表已按 `uk(tenant_id, role_key)` 租户化，但 `internal/middleware/data_scope_middleware.go` 的读取与 `modules/system/iam/permission/permission_data_scope.go` 的读写**均未按租户过滤**，当前 144 行全部落在 `tenant_id = 0`、且无跨租户重名 `role_key`，因此行为正确。一旦写入租户级策略行，读取侧可能跨租户合并。建议后续将租户上下文引入该读取路径。

2. **CGO/sqlite 工具链缺口 —— 已于 2026-10-15 解决**
   `modules/system/iam/tenant` 曾是全仓**唯一**仍使用 `gorm.io/driver/sqlite` 的测试包：本机 mingw gcc 无法完成 CGO 链接，该包测试长期无法本地执行，历次记录为 `runtimeGap`。现已统一迁移到 `pkg/testmysql`（真 MySQL，与仓库其余所有包一致），并补齐 provisioning 真正会写入的跨模块表（`system_role` / `system_user` / `system_user_role` / `system_setting`）——此前 sqlite fixture 只建两张表，使该套件悄悄停止覆盖 provisioning 路径。
   该包现已 **23/23 本地通过**；`gorm.io/driver/sqlite` 与 `github.com/mattn/go-sqlite3` 已从 `backend/go.mod` 移除，全仓不再有 sqlite 测试依赖，CI 也不再需要为该包开启 CGO。
   一处 MySQL 语义差异已按生产写法处理：AUTO_INCREMENT 列中显式插入 `0` 会被改写成下一个计数值，故预留占位行改为「先插入、再 `UPDATE ... SET id = 0`」两步法（与 migration 000013 的 adopt 分支一致）。

3. **`platform.tenant_mode` 单向切换**
   `compat → multi` 单向（合同 §6），回退需数据回滚，禁止运行期直接改回。

---

## 五、结论

- 租户主链路（初始化 → 用户 → 角色权限 → 数据隔离 → 生命周期 → 平台视角）在 `multi` 模式下端到端可用，E2E 62/62、隔离 34/34。
- 本轮修复的 3 个缺陷均为**运行期可见的真实问题**：模式解析不确定、角色绑定错租户、成员读接口 500。修复后相关单测与 E2E 全绿。
- 剩余 1 项潜在风险（`system_role_data_scope` 租户维度未闭环）已记录，未改动，建议排入后续任务。
