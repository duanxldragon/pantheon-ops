# 租户实现功能完整性审查（2026-10-27）

**审查日期**：2026-10-27
**审查对象**：`pantheon-base` 租户域——后端 `modules/system/iam/tenant`、`modules/auth/{login,session}`、`pkg/tenant`、`internal/middleware`、`modules/system/{seed.go,system_modules.go}`；前端 `modules/system/iam/tenant`、`modules/auth/tenant`、`modules/system/iam/user`、`core/entitlements`、`core/layout`、`core/router`；文档与 CI。
**判据**：`docs/contracts/TENANT_CONTRACT_V1.md` §1 In/Out 与 §2 不变量；`docs/api/TENANT_API.md` v1.2。
**审查性质**：只读审查。未修改、删除、重命名任何源码、迁移、依赖或脚本，未执行任何数据库变更。
**基线**：工作区 `main`，审查期间工作区保持干净。

---

## 0. 结论

**单租户部署路径（平台运营控制台 + 每个租户自有账号）已经功能性完成，可以交付 MVP。**

**但「租户开发已完成」不成立。** 存在：

| 级别 | 数量 | 摘要 |
|---|---|---|
| P1（功能完整性） | 1 | 跨租户成员关系只有「能登录」，没有「能授权」：登录租户选择器与租户切换器都把用户送进一个空菜单、全 403 的控制台，且没有任何入口可以在目标租户授予角色 |
| P2（契约偏差） | 1 | `DeleteTenant` 允许删除仍处于 `active` 的租户（只要活跃成员数为 0），与契约 §2.1/§2.2「删除仅允许对 `archived` 执行」冲突 |
| P2（数据一致性） | 1 | `AddMember` 不校验用户是否存在：可写入不存在的 `user_id`，并可用它充当 `owner` 让真实 owner 被移除，实质破坏「每个非归档租户至少一个 owner」 |
| P3（已登记残余 / 漂移） | 6 | 前端套餐能力阶梯硬编码、API 文档登记未挂载路由、类型漂移、成员添加用裸 `user_id`、平台控制台无 UI 自动化、死代码与过期设计文档 |

一句话：**核心租户模型（生命周期、隔离、会话门禁、套餐阈值、运营视图）是完整且经过测试的；不完整的是「一个人跨多个租户」这条组合路径，以及删除/成员写入的两个边界。**

---

## 1. 判据：契约 In / Out

### 1.1 §1 In（本报告按此判定「完成」）

| In 项 | 判定 | 证据位置 |
|---|---|---|
| 租户实体与生命周期（active / suspended / archived / deleted 墓碑） | ✅ 完成 | `tenant_service.go:439-682`、`tenant_lifecycle_guard_test.go` |
| 租户上下文解析与请求隔离（`X-Tenant-Id`、`tenant_id` scope、fail-closed） | ✅ 完成 | `pkg/tenant/tenant.go`、`internal/middleware/tenant_context_middleware.go`、`tenant-protected-resources.spec.ts` |
| 租户创建 / provision（角色、菜单、根部门、管理员绑定、套餐） | ✅ 完成 | `tenant_provision.go:100-520` |
| 成员关系（owner/admin/member、owner 不变量、软禁用/物理删除按路径区分） | ✅ 基本完成，见 P2-2 | `tenant_service.go:858-975`、`user_membership.go` |
| 登录门禁、会话 claim、多成员选择器、租户切换 | ⚠️ 部分完成，见 P1-1 | `login_runtime.go:435-495`、`session_service.go:191-241`、`login_tenant_candidates.go` |
| 租户内角色与 Casbin domain（`role:<key>@tenant:<id>`） | ✅ 完成 | `tenant_provision.go:60-90,460-520`、`casbin_middleware.go` |
| 套餐定义与配额阈值（读侧可配、`-1` 不限、未知 code 回落 free） | ✅ 完成（阈值不拦截写入是契约 §2.4.1 的规定） | `tenant_plan_store.go`、`000030_tenant_plan_definitions.up.sql` |
| 运营视图（总览、跨租户成员、套餐编辑） | ✅ 完成 | `tenant_ops_view.go`、`TenantOverview.tsx` |
| 控制台 UI（租户 CRUD、成员、套餐、运营总览） | ✅ 完成（见 P3-5 测试缺口） | `TenantList.tsx`、`TenantOverview.tsx`、`TenantMemberList.tsx` |
| 权限码 / 菜单 / seed / 操作审计 | ✅ 完成 | `seed.go:474-527`、`permission_workbench.go:629-645`、`tenant_platform_handler.go` |

### 1.2 §1 Out（明确不计入「未完成」）

以下项目在契约中写明不在 V1 范围，本报告不作为缺口：计费与售卖（下单、升降级、账期）、独立 schema/库、自定义域名与品牌、公开注册、套餐能力裁剪的商业化语义、配额**强制拦截**（契约 §2.4.1 明确「只作度量，不作写入门禁」）。

---

## 2. 已实现并验证的功能面

### 2.1 生命周期

- 状态机：`active ↔ suspended`、`→ archived`（终态）、`→ deleted`（墓碑）。`UpdateTenant` 用 `tenantStatusTransitionAllowed` 校验，墓碑不可复活（`TestDeletedTenantCannotBeRevivedByUpdate`）。
- 离开 `active`（archived/deleted）时同事务撤销成员会话。
- 删除释放 `code` 为 `__deleted_tenant_<id>`，`GetTenantByID` 把墓碑视为不存在，墓碑不可再被寻址、不可新增成员、不可重新 provision（三个回归测试齐备）。
- 前端状态筛选含 `deleted`，并对墓碑隐藏编辑/删除动作。

### 2.2 上下文与请求隔离

- 生产路由面全部挂在 `PlatformScopeTenantContextMiddleware + CasbinMiddleware + PlatformCapabilityMiddleware` 之后（`system_modules.go:258-306`）。
- `tenant.WithTenantScope` 统一注入 `tenant_id` 过滤；`qualifyTenantColumn` 处理 join 前缀。
- 浏览器冒烟 `tenant-protected-resources.spec.ts` 用两个真实租户验证：页面、字典读、CSV 导出互不可见，跨租户改单/批量改状态被拒绝。

### 2.3 登录与会话

- `resolveLoginTenantClaim`：compat 恒为 0；multi 下显式选择 > 单成员自动解析 > 多成员拒绝并要求选择；无成员的非平台主体直接拒绝（避免「登录成功但每页 500」）。
- `GateSessionIssuance` / `GateSessionRefresh` 每次签发/刷新都复核 membership 与租户状态。
- 平台主体在 multi 模式下通过 `platform-fallback` 落到 `tenant_id = 0`，并用 `X-Tenant-Id` 表达「租户视角」；平台能力由全局 Casbin 角色提供，视角只改变作用域、不提权。

### 2.4 成员与 owner

- `AddMember`：idempotent；`owner` 行不可降级；重复 owner 为 no-op。
- `RemoveMember`：先 `ensureNotLastOwner`，再物理删除，随后尽力撤销该成员会话。
- 删除用户时成员关系软禁用（`status=disabled`），保留审计链；租户作用域只动本租户行。

### 2.5 套餐

- `tenant_plans` 表 + `PlanStore`（5s TTL）+ `ValidateAssignment`；四个内置 code 由迁移 seed；`Update` 拒绝停用在用套餐。
- 运营总览报告每租户 `MaxUsers/MaxRoles/MaxDepts` 与用量；未知 code 以 `Known=false` 如实呈现。
- 前端 `planOptions.ts` 从 `/tenant-ops/plan/list` 派生选项，V1.5 已接通自定义 code 的指派。

### 2.6 前端控制台与权限装配

- 菜单契约、i18n 硬编码、系统页准入检查均通过（见 §5）。
- 平台主体隐藏非平台菜单（`useLayoutMenu` 按 `tenantMode` 过滤 `system.tenant`）。

---

## 3. 缺口

### P1-1 跨租户成员关系只有登录，没有授权

**现象**：`tenant_memberships` 的约束是「一个 user 可属于多个 tenant」（契约 §2.2），登录选择器与 `TenantSwitcher` 也按这条约束实现。但一个用户只在**拥有其 `system_user` 行的租户**里能拿到 IAM 角色；在第二个租户里永远拿到空角色集。

**代码证据**（三条互锁，缺一不可）：

1. `replaceUserRoles`（`user/user_helper.go:83-108`）把 `system_user_role.tenant_id` 从用户行自身解析，注释写明「a binding can never be written into a tenant the user does not belong to」。
2. `RoleService.ensureUsersExist`（`role/role_helper.go:301-307`）经 `s.scoped(...)`（`role_service.go:39-41`）按 `system_user.tenant_id` 过滤，因此在租户 B 的视角里看不到租户 A 的用户，无法把他们加进 B 的角色。
3. 登录/切换时 `GetUserRolesForTenant`（`login_runtime.go:328-338`）要求 `system_user_role.tenant_id = claim` 且 `system_role.tenant_id = claim`。B 的绑定不存在 ⇒ 返回 `[]`。

`AddMember` 只写 `tenant_memberships`，`role` 字段是租户内展示角色（owner/admin/member），契约 §2.2 明确「与全局 IAM 角色命名空间分离，不做隐式映射」。于是没有任何路径能把「B 的成员关系」翻译成「B 的权限」。

**失败场景（可复现）**：

1. 平台管理员在租户运营的跨租户成员页看到租户 A 的用户 `lisi`（`user_id=42`）。
2. 打开租户 B 的成员弹窗，输入 `42`、角色 `member`，提交成功（`POST /system/tenant/B/members`）。
3. `lisi` 登录：两个 active membership ⇒ `tenantSelectionRequired` + 两个候选（`auth-tenant-picker` 路径）。
4. 选择租户 B ⇒ `GateSessionIssuance` 通过（membership active + B 状态 active），会话 claim = B，角色集 = `[]`。
5. 进入应用：`loadAllowedNavigationMenuIDs` 对零角色直接返回空（`menu_service.go:256-265`），菜单树为空；所有受保护 API 走 Casbin 得到 403 `permission.denied`。
6. 控制台没有任何入口说明「你在 B 没有角色」，也没有入口可以授予——租户 B 的控制台看不到 A 的用户，平台视角下 B 的角色成员选择同样只列 B 的用户。

`TenantSwitcher.tsx:22-34` 的注释已经为**平台主体**推理过同一问题（「a platform operator owns none, so the result would be a console the operator cannot use」），但对**普通多租户成员**没有对应处理。

**现有测试为什么没拦住**：`tenant-hostile-browser-matrix.spec.ts:82-92` 正是给租户 A 的管理员加了 B 的成员关系，但选择 B 之后只断言 `.app-shell__header` 可见（`:138-140`），没有断言菜单/页面/接口可用。断言强度不足，把「能登录」当成了「能用」。

**影响**：契约 §2.2 的多租户成员不变量与已出货的登录选择器/切换器组合不可用。不影响单租户部署（每个租户自有账号）。

**修复方向（不实施，供决策）**，三选一：

- **A（最小）**：把 `AddMember` 收窄为「只允许添加该租户拥有的用户（`system_user.tenant_id = tenantID`）」，并在 UI 把成员添加改成租户内用户选择器。多租户成员仍保留为登录/审计语义，但契约明确「跨租户成员不授予权限」。
- **B（完整）**：新增「跨租户角色绑定」路径（例如成员添加时同时提交 `role_ids`，由服务端在目标租户写入 `system_user_role`），并放开 `ensureUsersExist` 的目标租户校验。需要重新定义「用户归属租户」与「绑定租户」的关系。
- **C（收缩）**：删除登录选择器与 `TenantSwitcher` 的非平台分支，契约改为「一个用户只属于一个租户」。这是产品降级，但不留半成品。

推荐 A 作为 V1 收口、B 作为 V2 立项；不推荐继续保留现状。

---

### P2-1 删除绕过 `archived`（合同偏差）

**契约**：§2.1「删除仅允许对 `archived` 租户执行」；§2.2 删除路径表也按此描述。

**实现**（`tenant_service.go:596-604`）：

```go
if memberCount > 0 && row.Status != tenant.TenantStatusArchived {
    return ErrTenantHasActiveUsers
}
```

判断是「有活跃成员 **且** 不是 archived」，而不是「不是 archived」。因此一个 `active` 且活跃成员为 0 的租户可以直接删除，跳过归档。

**失败场景**：平台管理员在租户视角删除该租户的全部用户（`DeleteUser` 把成员关系置 `disabled`），然后点删除。此时 `memberCount = 0`，删除通过：状态直接变 `deleted`、`code` 被释放、没有归档窗口、没有「先归档再删」的运维缓冲。这条路径在 UI 上只需两次合法操作，且删除按钮对任何状态都可用（只受权限码限制，`TenantList.tsx:511-522`）。

**附带**：`docs/api/TENANT_API.md` 与 `TENANT_TROUBLESHOOTING.md` 按宽松规则描述删除，但没有标注这是对契约的偏离。

**修复方向**：守卫改为 `if row.Status != tenant.TenantStatusArchived { return ErrTenantNotArchived }`。归档租户仍保留「有活跃成员也可删」的同事务停用逻辑——这正是现有注释里那个死锁的解法，不需要 `memberCount` 参与准入判断。前端应把删除动作在非 `archived` 行上禁用并提示先归档。

---

### P2-2 `AddMember` 不校验用户存在：幽灵 owner 破坏 owner 不变量

**实现**（`tenant_service.go:858-903`）：只调用 `GetTenantByID(tenantID)` 校验租户存在，`userID` 直接写库。`tenant_memberships` 没有到 `system_user` 的外键（`000013_tenant_canary.up.sql:48-59`）。现有测试 `TestAddMember` 用 `user_id = 100` 成功添加，而测试库中没有任何用户行，等于把这个行为固定成了预期。

**失败场景 A（幽灵 owner → 真实 owner 被移除）**：

1. 租户 B 的实际 owner 是 `user_id=42`。
2. 平台管理员在成员弹窗误输入 `429`（或 `AddMember` 的 API 调用方传错），角色选 `owner` ⇒ 写入一行不存在的 owner。
3. `ensureNotLastOwner` 只统计 `role='owner' AND status='active'` 的行（`tenant_service.go:948-975`），现在计数为 2。
4. 移除 `42` 被允许。此后租户 B 的唯一 owner 是一个永远无法登录的 id；租户自有管理员全部失去控制台。
5. 另一条路径同理：输入一个已删除用户的 id、或另一租户的 user id，都会产生「成员列表里有名字/空名字，但不可用」的行。

**影响**：owner 不变量（每个非归档租户至少一个 owner）在语义上被破坏，而平台的 `PlatformCapabilityMiddleware` 还能修复，所以不是不可恢复的锁死；但租户侧确实会失管。

**修复方向**：

- `AddMember` 校验用户存在于目标租户作用域（与 P1-1 的 A 方案一致），不存在返回 `user.not_found`。
- `ensureNotLastOwner` 的 owner 计数 JOIN `system_user`（未删除、启用），只把真实可登录用户算作 owner。
- 前端成员添加改为租户内用户选择器（见 P3-4），不再手输 id。

---

### P3-1 前端套餐能力阶梯仍是四档硬编码（已登记）

`core/entitlements/plans.ts` 固定 `free|basic|professional|enterprise`，未知 code 归一化为 `free`。运营者自定义套餐的配额阈值生效，但控制台功能入口按免费版展示；`TenantPlanMatrix` 用 `plan.outsideLadder` 提示。契约 V1.5 已把这个残余写成显式 gap（「按套餐裁剪菜单/能力仍是 §1 Out 的商业化语义，需要另行立项」）。**不是新发现，也不算违约**，但它是「租户运营」功能面上唯一的已知能力断点。

### P3-2 API 文档登记了未挂载的 `/members/page`

`docs/api/TENANT_API.md:229,255` 说 `/system/tenant/:id/members` 的响应信封「与 `/members/page` 兼容别名共享」。该路由只存在于 `tenant_routes.go:25`（`RegisterRoutes`），而生产装配只调用 `system_modules.go` 里的显式路由，`RegisterRoutes` / `NewHandler` 在非测试代码中零调用。实际挂载面没有 `/members/page`。文档会让调用方按一个 404 路由开发。

### P3-3 `MemberUsageQuery.status` 类型漂移

`frontend/src/modules/system/iam/tenant/api.ts:239` 声明 `'active' | 'removed'`，而 `TenantMemberList.tsx:230-236` 实际发送 `'active' | 'disabled'`，并通过 `as MemberUsageQuery['status']` 强制转换把错误压掉。后端接受自由字符串（默认 `active`），所以运行时正常，但类型检查已经不能保护这个字段。

### P3-4 成员添加 UI 是裸 `user_id` 文本输入

`TenantList.tsx:847-856`：成员弹窗只有一个数字输入框 + 角色下拉。操作者必须自己知道用户 id（可从租户运营的跨租户成员页抄），没有用户选择器、没有存在性反馈。这是 P2-2 的主要人为入口，也让「添加成员」在日常运营中不可用。

### P3-5 平台租户控制台没有自动化 UI 覆盖

现有冒烟覆盖：租户视角切换（`tenant-viewpoint-switcher.spec.ts`）、隔离（`tenant-protected-resources.spec.ts`）、登录选择器与 dashboard 隔离（`tenant-hostile-browser-matrix*.spec.ts`）。**没有**覆盖：租户列表 CRUD、归档/删除、成员增删、运营总览、套餐编辑器。`TenantList/TenantOverview/TenantPlanEditor` 也没有单测。而 `smoke-tenant` 作业只在 push 上跑且是 report-only，等于这些页面在任何门禁上都没有回归信号。

### P3-6 死代码与未接线组件

- `tenant_handler.go` + `tenant_routes.go`：旧 raw-JSON handler，生产未挂载，仅测试引用。
- `tenant_bootstrap.go`（`BootstrapService` 等 5 个入口）：全仓零引用。
- `tenant_production.go` / `tenant_metrics.go` 的 `QuotaEnforcer` / `AuditLogger` / `HealthChecker` / `MetricsCollector`：零调用；契约 §2.4.1 已说明是「只定义不启用」，属已登记范围。

这些不影响功能，但会持续误导读者以为存在第二套入口。

### P3-7 owner 转移没有单列 API

契约 §2.2 说「转移 owner 是显式 API 动作」。实现上可以「先 add 第二个 owner，再 remove 第一个」完成转移，但没有独立的 transfer 端点，也没有 UI 把这两步组合成一个动作。功能可达，措辞不完全对应；若运维文档按「转移 API」指引会找不到入口。

### P3-8 `MULTI_TENANT_DESIGN.md` 与已交付实现不一致

`docs/designs/MULTI_TENANT_DESIGN.md` 描述了子域名识别、自定义域名、品牌、公开注册与「Next Phase: Implementation」，这些都在契约 §1 Out。它没有标注「设计愿景 / 非 V1 承诺」，容易被当作现状说明。契约与 API 文档是准确的，这份设计文档不是。

---

## 4. 显式非缺口

以下项目曾被怀疑为缺失，核对后确认**是契约规定或已登记，不算未完成**：

| 项 | 说明 |
|---|---|
| 配额不阻断写入 | 契约 §2.4.1：只作度量与运营视图，`QuotaEnforcer` 未接线是刻意选择 |
| `max_api_keys` / `max_storage_mb` 无用量报告 | 契约 V1.5 已列为显式残余 |
| 租户内角色（owner/admin/member）与 IAM 角色不映射 | 契约 §2.2 明文「不做隐式映射」 |
| 计费、升降级、账期 | §1 Out |
| 独立 schema / 库、自定义域名、品牌、公开注册 | §1 Out |
| 租户套餐功能入口按免费版展示 | 前端四档阶梯的已知残余（P3-1） |
| `tenant_production.go` / `tenant_metrics.go` 未启用 | 契约 §2.4.1 登记 |

---

## 5. 验证证据

本次审查执行过的只读检查（均通过；审查期间未改动任何文件，因此结果保持有效）：

**后端单元/集成测试**

```
cd backend && go test -count=1 ./modules/system/iam/tenant/...
cd backend && go test -count=1 ./pkg/tenant/...
cd backend && go test -count=1 ./modules/auth/session/...
cd backend && go test -count=1 ./modules/auth/login/...
cd backend && go test -count=1 ./internal/middleware/...
```

结果：全部 `ok` / 0 FAIL。租户包共有 16 个测试文件、92 个 `Test` 函数；关键回归包括 `TestDeletedTenantCannotBeRevivedByUpdate`、`TestDeletedTenantRefusesNewMembers`、`TestDeletedTenantCannotBeReprovisioned`、`TestAddMember*`、`tenant_concurrency_test.go`、`tenant_ops_view_test.go`。

> **2026-10-27 复测更正（重要）**：上面这轮 `ok` 是在**没有设置 `PANTHEON_TEST_DSN`** 的情况下跑的。`pkg/testmysql` 在没有 DSN 时会把依赖 MySQL 的用例 `t.Skip`，租户包因此只跑了 34 个用例、**85 个静默 skip**，整包 0.04s 就报 `ok`——"全绿"其实是"基本没跑"。带 DSN 复测后的实测基线：
>
> - 租户包：**119 pass / 0 skip / 0 fail**（约 65s，真实 MySQL）
> - 后端全量：46 个包 `ok`，exit 0
> - 前端：`tsc -b`、`lint`、`build`（含 14 项 prebuild 契约检查）通过；`vitest` 34 文件 / 230 测试通过
> - 复测命令：`export PANTHEON_TEST_DSN='<dsn>' PANTHEON_TEST_DSN_REQUIRED=true`（后者把 skip 变成 fail，防止再次被静默跳过）
>
> 本节之前的结论（P1/P2/P3 的代码路径判断）不依赖测试执行结果，仍然成立；但如果有人引用"测试全绿"来证明租户域健康，必须使用上面带 DSN 的数字。

**前端静态门禁**

```
cd frontend && npm run check:menu-contract      # 19 menus / 23 routes / 102 permissions / 23 component keys
cd frontend && npm run check:i18n-hardcode      # 238 files
cd frontend && npm run check:i18n-generated-scope
cd frontend && npm run check:system-page-admission  # 16 entries
cd frontend && npx tsc -b
cd frontend && npx eslint <tenant dirs>
```

结果：全部通过。

**路由与权限装配**

- 生产挂载面 13 条（`system_modules.go:258-306`），与 `permission_workbench.go:629-645` 的权限→路由绑定表一一对应，且由 `TestRequiredAPIRoutesExistOnEngine` 在 CI 强制。
- `seed.go:474-527` 的 3 个菜单 + 11 个权限码全部能在挂载面找到对应路由。

---

## 6. 未检查 / 待验证

1. **浏览器 e2e 只跑了既有 4 个 tenant spec**：2026-10-27 12:40 在后端 :8080 + MySQL + Redis 的实栈上跑通 `tenant-hostile-browser-matrix`、`-phase2`、`tenant-protected-resources`、`tenant-viewpoint-switcher`，**15 passed / 0 failed**（证据：`.harness/evidence/2026-10-27-tenant-completeness-remediation/smoke-tenant-baseline.md`）。这覆盖隔离与视角切换，但**不覆盖**租户控制台 CRUD/归档/成员/套餐（W6 缺口），且登录选择器用例未断言"选中所属租户后控制台可用"；因此 **P1-1 的失败场景仍未在真实环境复现**，仍只是按代码路径推导。审查期间未运行 e2e 的部分，其结论来自代码与单测，UI 结论来自静态装配检查。
2. **未连接真实租户数据库**，未验证 `tenant_plans` 的 seed 与既有部署的兼容性。
3. **未审查 `pantheon-ops` 对租户域的业务扩展**（其 `backend/modules/system/iam/tenant` 与 base 字节一致，前端有本地副本）；ops 侧是否有自己的成员/角色组合逻辑未逐文件核对。
4. **未评估性能**：运营总览的跨租户查询在租户数达到数百时的表现没有压测数据（当前实现有 20/100 条上界与聚合查询）。
5. **未验证 OIDC 登录 + 多租户选择器组合**：OIDC 回调不经过密码登录的 tenant picker 路径，多成员 OIDC 用户在无法选择租户时会落到哪个分支，本次未逐行追踪。

---

## 7. 建议整改顺序（只给方案，不实施）

| 顺序 | 项 | 理由 |
|---|---|---|
| 1 | P2-2 用户存在性 + owner 计数 JOIN | 最小改动（一个校验 + 一个 JOIN），消除数据一致性隐患，且是 P1-1 修法的前置 |
| 2 | P2-1 删除守卫改为「非 archived 一律拒绝」+ 前端禁用删除 | 一行后端守卫 + 一个 `disabled`，直接回到契约 |
| 3 | P1-1 方案 A（收窄 AddMember + 用户选择器） | 让多租户成员关系回到「登录/审计」语义，删掉半成品路径 |
| 4 | P3-2 / P3-3 / P3-8 文档与类型对齐 | 低成本，避免调用方按错误契约开发 |
| 5 | P3-5 补平台控制台的 UI 自动化 | 把已经手工验证过的 CRUD/成员/套餐路径纳入回归 |
| 6 | P3-1 套餐能力阶梯 | 已登记，按商业化立项处理，不在本轮 |
| 7 | P3-6 死代码清理 | 合并到既有的代码精简程序（`2026-10-27-s1-dead-code-removal`） |

若第 3 项选择方案 C（收缩多租户成员），则第 1 项的「用户归属校验」简化为「只允许本租户用户」，改动更小。

---

## 附录 A：当前生产路由面（13 条）

```
GET    /api/v1/system/tenant/list
GET    /api/v1/system/tenant/:id
POST   /api/v1/system/tenant
PUT    /api/v1/system/tenant/:id
DELETE /api/v1/system/tenant/:id                    (+ SecureAction)
GET    /api/v1/system/tenant/:id/members
POST   /api/v1/system/tenant/:id/members
DELETE /api/v1/system/tenant/:id/members/:userId    (+ SecureAction)
GET    /api/v1/system/tenant-ops/overview
GET    /api/v1/system/tenant-ops/member
GET    /api/v1/system/tenant-ops/plan/list
POST   /api/v1/system/tenant-ops/plan
PUT    /api/v1/system/tenant-ops/plan/:code
```

全部位于 `PlatformScopeTenantContextMiddleware + CasbinMiddleware + PlatformCapabilityMiddleware` 之后。

## 附录 B：审查方法

- 以真实文件读取为准。**CodeGraph 索引对租户包滞后**：它列出了一个不存在的 `backend/modules/system/iam/tenant/tenant_switch.go`，且缺失真实的 `backend/modules/auth/login/login_tenant_switch.go`。本报告未采用该索引的租户符号位置，全部引用均来自工作区真实文件。
- 静态引用追踪（全仓 `rg`，排除 `.tmp` 备份与测试）用于判断死代码与调用面。
- 契约与 API 文档逐条对照，偏差在正文中标出。
- 本报告不修改代码；所有「修复方向」均为待决策方案。
