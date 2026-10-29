---
title: 租户合同 V1（共享 schema MVP）
doc_type: Contract
layer: platform / system/auth / system/iam / system/org / system/config
status: Approved
related_designs:
  - docs/designs/TENANT_READY_SINGLE_TENANT_DESIGN.md
  - docs/designs/TENANT_RESOURCE_SCOPE_MATRIX.md
  - docs/designs/P2_SCALE_ROADMAP.md
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
  - docs/contracts/SYSTEM_AUTH_CONTRACT.md
  - docs/contracts/SYSTEM_IAM_CONTRACT.md
  - docs/contracts/SYSTEM_ORG_CONTRACT.md
  - docs/contracts/SYSTEM_CONFIG_CONTRACT.md
updated_at: 2026-10-27
---

# 租户合同 V1（共享 schema MVP）

English version: [TENANT_CONTRACT_V1.en.md](./TENANT_CONTRACT_V1.en.md)

本合同是 `2026-09-10-tenant-contract-design` 任务的冻结交付物，为 [P2 规模化路线图](../designs/P2_SCALE_ROADMAP.md) 的租户化阶段提供共享 schema MVP 的执行契约。设计依据见 [单租户先行、租户就绪设计](../designs/TENANT_READY_SINGLE_TENANT_DESIGN.md)。

冻结 = 冻结 V1 的 **模型、所有权边界、不变量**；具体表结构演进由 runbook（`2026-09-10-tenant-migration-runbook`）约束。

关联设计：
- `docs/designs/TENANT_READY_SINGLE_TENANT_DESIGN.md`
- `docs/designs/TENANT_RESOURCE_SCOPE_MATRIX.md`
- `docs/designs/P2_SCALE_ROADMAP.md`

---

## 1. 范围（In / Out）

### In

- 租户（Tenant）主数据模型与生命周期（active / suspended / archived）
- 租户成员（TenantMembership）：用户与租户的多对多归属及成员角色
- 租户上下文（Tenant Context）：请求级租户解析、注入、传播与校验
- 资源租户归属：业务/系统资源通过 `tenant_id` 归属租户，配 resource scope 矩阵约束
- Casbin domain 维度的权限隔离（policy 带 tenant domain）
- Compat 模式：单租户兼容开关，保证现有单租户部署零迁移继续运行
- 套餐定义与配额阈值的**存储与读展示**（V1.4 新增，§2.4）与平台侧租户运营的**跨租户聚合读路径**（§2.5）

### Out

- 独立 schema / 独立 database 隔离模型（V1 仅共享 schema + `tenant_id`）
- 租户计费与套餐的商业化售卖（下单、升降级、账期，归商业域，另行立项）。
  注意边界：套餐**定义**与配额**阈值**自 V1.4 起已落库并可由平台控制台编辑（§2.4），
  但配额只是度量与展示维度——**后端不因超出配额而拒绝任何写入**。
- 跨租户数据共享/联邦查询
- 租户级自定义域名、独立部署形态

---

## 2. 核心模型（冻结）

### 2.1 Tenant 主数据

```
tenants
  id            bigint PK          -- 雪花 ID，由 ID 生成器分配
  code          varchar(64)  UK    -- 租户编码，全局唯一，创建后不可变
  name          varchar(128)       -- 显示名
  status        varchar(16)        -- active | suspended | archived | deleted（墓碑，见不变量）
  plan          varchar(32)        -- 套餐标记：解析为 §2.4 的配额阈值（只用于度量展示）
  created_at / updated_at
  deleted_at                       -- 预留列，当前实现不写入（见不变量）
```

不变量：
- `code` 创建后不可变；匹配 `^[a-z][a-z0-9-]{1,62}$`
- `status` 流转仅允许：`active ↔ suspended`，`active|suspended → archived`；`archived` 终态不可逆
- `archived` 租户禁止登录、禁止发放新 token、禁止产生新业务写入
- 删除仅允许对 `archived` 租户执行，且为运维动作，不暴露给业务 API
- `deleted` 是**墓碑（tombstone）**，不是生命周期状态（V1.3 澄清）：只由删除动作在 `archived`
  之后写入，与释放 `code` 在同一事务内完成；它没有出边，**不得作为创建或更新的输入**，也不出现在
  默认列表里（仅显式状态过滤可见）。按 id 或 `code` 寻址已删除租户一律视为不存在（与 §5
  「缺失与他租户不可区分」一致），因此已删除租户无法被重新激活、无法新增成员、也无法重新初始化。
- 删除 `archived` 租户时，其剩余成员关系在同一事务内被置为停用：§2.2 的 owner 要求只约束
  **非归档**租户，归档后成员关系不再构成删除的阻碍（V1.3 澄清）。
- 删除标记是 `status = 'deleted'`，**不是** `deleted_at`：该列在表结构中预留，当前实现不写入，
  查询也不以它过滤。两套标记并存会让「是否已删除」出现两个可能不一致的答案。

### 2.2 TenantMembership

```
tenant_memberships
  id            bigint PK
  tenant_id     bigint       -- FK → tenants.id，复合唯一键前缀
  user_id       bigint       -- FK → iam users.id，复合唯一键前缀
  role          varchar(32)  -- owner | admin | member（租户内角色，非全局角色）
  status        varchar(16)  -- active | disabled
  created_at / updated_at
  UK(tenant_id, user_id)
```

不变量：
- 一个 user 可属于多个 tenant；同一 `(tenant_id, user_id)` 唯一
- 每个非 archived 租户至少一个 `owner`；**owner 转移没有单一端点**（V1.8 澄清），落地路径是
  两步显式 API 动作：先把新 owner 加入（`POST /system/tenant/:id/members`，`role=owner`，已有行
  按上表复活语义升级），再移除原 owner（`DELETE /system/tenant/:id/members/:userId`，
  `ensureNotLastOwner` 保证任意时刻至少剩一个 owner）。两步之间租户短暂持有两个 owner，属合法中间态
- membership 的**下架方式按路径区分**（V1.5 澄清，逐条对齐实现；原条款「一律软禁用、不物理删除」
  与实现不符）：

  | 路径 | 实际行为 | 代码 |
  |---|---|---|
  | 删除租户 | 软禁用：同一事务内把该租户剩余 `active` 成员关系置 `status = disabled`，与 `status = 'deleted'`、释放 `code` 一起提交（§2.1） | `tenant_service.go:608-615` |
  | 删除用户 | 软禁用：`status = disabled`；租户作用域只动本租户的行，平台作用域清理该用户全部成员关系（§2.3.1 第 2 条） | `user/user_membership.go:78-88`，回归 `user_membership_test.go:152-200` |
  | 显式移除成员 | **物理删除**（当前实现的唯一例外）：先 `ensureNotLastOwner`，按 `(tenant_id, user_id)` 删除（不限 `status`，模型无 `deleted_at` 列，`tenant_model.go:25-33`），`RowsAffected == 0` → `ErrMembershipNotFound`，随后尽力撤销该成员会话 | `tenant_service.go:914-940` |
  | 重新加入 / 重新创建用户 | **复活旧行**：命中 `(tenant_id, user_id)` 已有行则更新为请求角色 + `active`，不插第二行 | `tenant_service.go:869-888`、`user_membership.go:45-51` |

  默认方向是软禁用，物理删除是需要收敛的例外，理由保留在下一条。
- 保留行的意义：成员关系是「此人曾属于该租户」的审计记录，且登录门禁已把非 `active` 视为拒绝
  （`pkg/tenant.GateSessionIssuance`），所以禁用足以阻断访问。物理删除路径的审计事实则由
  全局操作日志承担（`operation_log_middleware.go:224-276` 记录全部非 GET 请求；平台侧成员增删另加
  `SetAuditMetadata(..., BusinessGrant/BusinessDelete)`，`tenant_platform_handler.go:201-239`；
  租户侧 `/tenants/:id/members/:user_id` 的 handler 不设标题、只依赖默认标题，属可改进项而非契约要求）。
- **把「移除成员」也改成软禁用是契约变更，不是澄清**，本轮未做（写侧行为不动）。落地前置条件已登记：
  ①要定义重复移除的错误语义——现状靠 `RowsAffected == 0` 区分「没有这行」，改成 `UPDATE ... status = disabled`
  后 MySQL 对无变化的行同样返回 `RowsAffected = 0`，存在性判断不能再复用这个信号；
  ②复活语义要覆盖所有插入路径（`AddMember`、`user` 侧建用户/导入、租户 provisioning），否则会撞上
  `UK(tenant_id, user_id)`；③成员列表、跨租户成员视图（§2.5）与「至少一个 owner」的统计口径要显式
  声明是否把 `disabled` 行计入。
- **跨租户 membership 不授予授权、也不在登录时提供**（V1.7，2026-10-27；W3 决策 A）。
  角色（`system_user_role`）按主体的 owning tenant 落库，因此 membership 存在于 owning tenant
  之外时，该主体在该租户**没有任何角色**——登录发现、登录候选列表、可切换租户列表三个读点
  只返回「存在至少一条 `system_user_role` 绑定」的租户（单一谓词 `pkg/tenant.HasRoleBindingInTenant`，
  三处共用；查询失败按无绑定处理，fail-closed）；`SwitchSessionTenant` 对解析出空角色集的目标拒绝；
  `AddMember` 只接受 `system_user.tenant_id` 等于目标租户的用户（错误码见 §5）。
  **存量跨租户 membership 行不清理**：它们继续满足数据层不变量「一个 user 可属于多个 tenant」，
  但从本澄清起不可达于登录，`/auth/current` 仍如实上报该 membership 行。选项 B（跨租户角色绑定）
  是独立工作流，落地前必须先修复 `user_helper.go` 跨租户 DELETE 的前置缺陷。
- 租户内角色（owner/admin/member）与全局 IAM 角色（`iam roles`）**命名空间分离**，不做隐式映射

### 2.3 资源租户归属（tenant_id 传播）

- 需要租户归属的表统一增加 `tenant_id bigint NOT NULL DEFAULT 0`：
  - `tenant_id = 0` 保留给 **平台级 / 全局资源**（compat 模式下的存量数据即全局资源）
  - `tenant_id > 0` 表示租户私有资源
- 哪些表带 `tenant_id`、哪些保持全局，由 [租户资源 scope 矩阵](../designs/TENANT_RESOURCE_SCOPE_MATRIX.md) 登记，新增表必须在登记表注册后才允许合入（见验收矩阵 T4）
- **禁止**：`tenant_id` 可空（NULL 语义歧义）；用字符串租户编码做外键；业务代码直接 `WHERE tenant_code = ?`

#### 2.3.1 读写义务（V1.2 补充，2026-10-19）

`tenant_id` 是**行的身份**，不是装饰字段。带 `tenant_id` 的表，任何读写代码必须同时满足下列三条；三条各自可被 reviewer 单独引用驳回。

1. **读必须带租户谓词。** 查询必须附加 `tenant_id = ctx.tenant_id`（或经 §3.3 的 scope helper 注入）。
   「租户内唯一」的业务键（`role_key`、`username`、`dept_code`、`post_code` 等）在库里的唯一约束是
   `(tenant_id, key)`，**不是全局唯一**。因此 `WHERE role_key = ?` 本身就是跨租户读取，
   即使加了 `LIMIT 1`、即使只用来回显一个字段。
2. **写必须由上下文赋值租户。** `tenant_id` 只能来自请求上下文（§3.2），不得来自请求体；
   upsert 的冲突键 / `ON CONFLICT` 必须包含 `tenant_id`。以裸键作为冲突键会把租户私有行**改写成
   `tenant_id = 0` 的平台行**，而平台行对所有租户生效——这是最隐蔽的一种放大，因为写入方看到的只是「保存成功」。
3. **按裸键做存在性或归属校验同样算读。** 「这个 `role_key` 存在吗」「这些 `dept_id` 是我的吗」
   这类判断必须带租户谓词。否则一个租户可以用猜到的键，给另一个租户的资源建策略，
   或把另一个租户的主数据 id 写进自己的配置（`dept_id`、`post_id` 是全局序列，同一数字在不同租户指不同对象）。

**唯一的两个放行面**（对应 §3.3 的 cross-tenant 两条路径）：

| 作用域 | 可以读写 | 不因此获得 |
|---|---|---|
| 平台作用域（`ctx.tenant_id = 0`，需 §3.4 声明 + 平台能力） | `tenant_id = 0` 的全局行 | 按裸键操作**租户私有行**的权力 |
| 租户作用域（`ctx.tenant_id > 0`） | 自己那一份 | 任何其他租户的行，包括「同名」行 |

上下文缺失且非 compat → 按 §3.2 拒绝，**不得退化为「读全部」**；策略/配置读不到时的默认行为是 fail-closed，
而不是回到最宽松的一档。

> 可引用条款：reviewer 驳回一个 `WHERE role_key = ?` 时引用「§2.3.1 第 1 条」；驳回一个
> `ON CONFLICT(role_key)` 时引用「§2.3.1 第 2 条」；驳回一个不带租户的 `Count()` 归属校验时引用「§2.3.1 第 3 条」。
> 本条款的回归测试：`backend/internal/middleware/data_scope_tenant_test.go`、
> `backend/modules/system/iam/permission/permission_data_scope_tenant_test.go`。

### 2.4 套餐定义与配额语义（V1.4 新增，2026-10-25）

```
tenant_plans
  id              bigint unsigned PK AUTO_INCREMENT
  code            varchar(32)  UK  -- 套餐编码，^[a-z][a-z0-9_-]{1,31}$，规范化为小写后比较
  name            varchar(64)      -- 显示名，缺省取 code
  max_users       int              -- 成员上限
  max_roles       int              -- 角色上限
  max_depts       int              -- 部门上限
  max_api_keys    int              -- API Key 上限
  max_storage_mb  int              -- 存储上限（MB）
  status          varchar(16)      -- active | disabled
  sort            int              -- 控制台排序
  created_at / updated_at
  UK(code)
```

DDL 与种子：编号迁移 `000030_tenant_plan_definitions`；AutoMigrate 部署走
`PlanStore.Migrate()`（`Service.Migrate()` 调用，同时是 system 模块的 `MigrateFunc`）。

不变量：

1. **平台全局表，没有 `tenant_id` 列**——与 `tenants` 本身同形。本模块的租户隔离由各 service
   显式 `WithTenantScope` 施加，不是全局 GORM 插件；给计划表加上 `tenant_id` 会让平台控制台的读
   被过滤成空集（§2.3 的归属规则约束的是"带该列的表"，不要求每张表都带）。
2. **`-1` 表示无上限（`tenant.UnlimitedQuota`），`0` 不是合法值**，写路径拒绝。理由：每个消费者
   都要区分"没有上限"与"上限已用尽"，允许 `0` 会让落库的定义把这两态压成同一个数字。
   校验发生在写入侧（`PlanQuotaReq.resolve`），不在读侧。
3. **空 code 的语义两侧不同**：定义侧（`tenant_plans.code`）空值非法，因为无法赋值也无法上报；
   租户侧（`tenants.plan`）空值合法，它是"未选套餐"的列默认值，也是本特性之前所有租户行存的东西。
4. **解析永不失败，未知值回落编译内置表**。`PlanStore.QuotaFor` 对空、未知、已停用 code 一律落到
   `GetDefaultQuota`，而后者把未知 code 映射到 free。读路径不能因为运营者不可能弄坏的数据而变成故障页。
   因此 `GetDefaultQuota` 不是遗留死代码：它是 seed 的值来源，也是表还不存在时（首次 AutoMigrate 之前、
   手工 schema 的单测）唯一的解析来源。
5. **退役是 `status = disabled`，删除行未建模**。仍有非删除租户携带该 code 时拒绝退役
   （`ErrPlanInUse`）；一行在租户还挂着它时消失，会让该租户的配额变成"没人选择过的上限"。
6. **种子幂等且只插不改**（`INSERT … WHERE NOT EXISTS` / 逐 code 计数后跳过）。一旦控制台能编辑阈值，
   重放时改写既有值的 seed 会静默撤销运营者；迁移重放在本仓库是真实状态（见 `000024` 的基线说明）。
7. **读穿透缓存有界**：TTL 5 秒（`planCacheTTL`），任何写入后立即 `Invalidate()`。阈值必须不重启就
   传播到每个实例，而这份数据的形态是"每请求读、极少写"——与 `pkg/tenant.ModeLoader` 对
   `tenant_mode` 的策略一致。
8. **能指派给租户的 code 由服务端判定**（V1.5 补齐，2026-10-26）。`PlanStore.ValidateAssignment`
   在租户创建与更新路径上调用，顺序即语义：空值合法（"未选套餐"，见不变量 3）→ 格式非法
   `tenant.plan.code_invalid` → 表不可读或为空时只接受编译内置四档（seed 还没到达不是运营者的错，
   provisioning 不能因此停摆，其余 code 报 `tenant.plan.unknown`）→ 已定义但停用
   `tenant.plan.disabled`。停用判定走 `tenant.Plan.IsEnabled` 的**失败开放**归一化：除 `disabled`
   以外的状态一律算启用。控制台的指派面镜像这一条规则，不另立第二套策略。

   控制台的租户表单因此从 `GET /system/tenant-ops/plan/list` 取定义，顺序沿用服务端的
   `sort ASC, id ASC`（内置阶梯正是靠这个顺序呈现，而不是前端再排一次）；写侧只列可指派 code，
   但**该租户当前携带的 code 即使已停用、甚至已不在定义里，也保持可选项**——否则打开一个退役套餐的
   租户会呈现一张一保存就静默改写其套餐的表单。读侧筛选列出全部已定义 code（含停用），因为仍挂着它的
   租户要能被筛出来如实报告（§2.5）。

**前端能力矩阵边界（W7 决策 A，2026-10-28）**：本节的配额权威与前端**能力矩阵**是两套面。
能力矩阵（`frontend/src/core/entitlements/plans.ts` 的内建四档 rank 与 `lowcode.*` feature key）
**只覆盖内建四档**；`normalizeTenantPlan` 对未知 code **fail-closed 归 `free`**——陈旧后端或
失败的 lookup 不得静默发放付费面。因此操作员自定义 code 的语义是「**配额桶 + `free` 能力档**」：
其阈值按本节表行照常生效，但 lowcode 门控面在租户视角不可见。该门控是 **UX-only**——隐藏
不授予也不撤销任何权限，授权仍由 §2.3/§3 的权限链决定；控制台的 `outsideLadder` 警告向操作员
告知这一后果（`system.tenant.plan.outsideLadder`，五语言）。让自定义 code 完整参与能力门控
（服务端 entitlements）属 V2 商业化工作流，不在 V1；选项裁剪与理由见
[套餐能力阶梯语义决策](../reviews/TENANT_PLAN_LADDER_DECISION_2026-10-28.md)。

#### 2.4.1 配额语义：度量，不是闸门（本节的边界条款）

- **后端在任何写入路径上都不因超出套餐而拒绝。** 配额的用途是让总览/成员页如实报告"用量 vs 阈值"，
  超限在界面上高亮而不是拦截。这是本轮显式的设计决策（只做真实统计展示），§1 Out 的商业化售卖
  仍然另行立项。
- 可引用条款：任何在写路径上加入 `用量 >= 上限 → 拒绝` 的改动都构成 **contract change**，
  必须走 §8 的版本号 + 变更记录 + runbook/matrix 同步流程，不能以"顺手补个校验"合入。
  反过来，任何把总览页的阈值改成写侧强制的"临时保护"同样适用本条。
- **`QuotaEnforcer`（`tenant_production.go`）与 `MetricsCollector`（`tenant_metrics.go`）已实现
  但在出货请求路径中未接线**：没有生产代码构造它们。记录这一事实是为了让"看起来已有强制能力"
  不被误读成"已经有强制"。保留的理由也记录在此：它们的用量口径与控制台同源，且经
  `quotaFor → PlanStore.QuotaFor` 读同一份定义，因此**将来若开启强制，不会与控制台已显示的
  阈值不一致**。未走 §2.4.1 上一条流程前，不得接线。

### 2.5 平台侧租户运营读路径（V1.4 新增，2026-10-25）

| 路由 | 权限 key | 性质 |
|---|---|---|
| `GET /api/v1/system/tenant-ops/overview` | `system:tenant:overview` | 跨租户聚合读 |
| `GET /api/v1/system/tenant-ops/member` | `system:tenant:member-list` | 跨租户成员读 |
| `GET /api/v1/system/tenant-ops/plan/list` | `system:tenant:plan:list` | 平台全局读 |
| `POST /api/v1/system/tenant-ops/plan` | `system:tenant:plan:create` | 平台全局写，落审计 |
| `PUT /api/v1/system/tenant-ops/plan/:code` | `system:tenant:plan:update` | 平台全局写，落审计 |

规则（每条可被 reviewer 单独引用）：

1. **每一条都是 §3.3 / §3.4 意义上的显式声明平台作用域路由**：位于 `system/*` 保护组内，由租户控制台的
   `PlatformCapabilityMiddleware` 保护。跨租户读**只允许**这一种形态；任何租户侧 API 不得复用
   `Service.Overview` / `Service.ListMembersAcrossPages`——它们天生无视当前请求的租户。
   路径单独成组而非 `/system/tenant/overview` 的原因是形状而非审美：`GET /system/tenant/:id`
   已占用该组的通配子节点，静态兄弟路径是等待在 Gin 升级时 panic 的路由决策。
2. **权限 key 留在 `system:tenant:` 前缀下**，使 `pkg/tenant.IsPlatformOnlyPermissionKey` 不需要第二份
   清单即可分类。后端 `pkg/tenant/platform_scope.go` 与前端 `core/layout/useLayoutMenu.tsx` 的
   platform-only 判定必须继续对同一前缀生效；新增 key 若跳出前缀，两侧判定必须在同一 patch 内补齐。
3. **口径必须与租户管理页一致**。所有读共享 `opsTenantScope`（排除 `id = platformGlobalTenantID`；
   除非显式按状态过滤，否则排除 `status = deleted`，与 §2.1 的墓碑规则同源）。占用统计复用模块常量
   （`activeTenantMemberPredicate`、`tenantStatusDeleted`、`platformGlobalTenantID`）：成员只计
   `status = 'active'`，角色与部门尊重软删，`tenant_id = 0` 的平台行不计入任何租户的占用。
   **禁止**在读模型里另起一套状态字符串——成员状态只有 `active | disabled`（§2.2）。
4. **分页有界**：总览与成员页默认 20、上限 100。上限有两道，语义不同：请求绑定
   （`Filter.Size` / `MemberRowFilter.PageSize` 的 `max=100`）**拒绝**超限输入，实测 `size=5000` →
   `param.invalid`；服务层再钳一次（`maxTenantOverviewSize`、`maxTenantMemberPageSize`），保护的是不走
   绑定的内部调用方。合同只承诺「有界」，不承诺超限会被静默改小。理由：每次请求都对全库聚合，
   无界 size 会把一次控制台调用变成全表扫描。
5. **未知套餐如实呈现**：`OverviewPlanRow.Known = false` 且 `PlanName` 为空时，页面按原始值展示，
   不得静默归档到 free——一个"没人管理"的套餐和"free 套餐"是两个不同的运营事实。
6. **指标集合冻结为核心组**：租户、成员、角色、部门。最近登录、会话数、安全事件**不在**本合同的总览里
   （显式 Out），否则总览会慢慢长成第二个仪表盘。
7. **权限↔路由绑定受门禁保护**：每条路由登记在 `permission_workbench.go` 的
   `requiredAPIRoutesByPermission`，由 `TestRequiredAPIRoutesExistOnEngine` 对照
   `buildWorkbenchRouteProbeEngine()` 的注册镜像校验。新增或改变路由形状必须在同一个 patch 内更新
   probe，否则门禁失败——这是防止"权限表与实注册漂移"的唯一机械手段。

---

## 3. 租户上下文（冻结）

### 3.1 解析优先级（从高到低）

1. 显式 header `X-Tenant-Id`（平台运维/后台管理场景，要求平台级权限 + 审计）
2. 认证主体（JWT claims 中的 `tenant_id`，登录时根据"用户默认租户"或登录请求指定租户签发）
3. 平台作用域兜底：请求落在**显式声明为平台管理 API 的路由**上，且主体具平台级能力、且不属于任何租户 → `tenant_id = 0`（multi 模式，见 §3.4）
4. Compat 模式兜底：`tenant_id = 0`

> **V1.1 澄清（2026-10-13）**：V1 原文只有第 1/2/4 条，且把兜底限定在 compat 模式，于是 multi 模式下"非租户（平台）主体"被判为 §3.2 的"上下文缺失"（500 `TENANT_CONTEXT_MISSING`），平台控制台在 multi 下整体不可用。新增的第 3 条把 §3.3 早已承认的"平台管理 API（显式声明，要求平台权限，落审计）"落到解析层：**`multi` 是 `compat` 的超集**，`tenant_id = 0`（平台级/全局，§2.3/§4）在两种模式下都是一等作用域。第 1/2 条优先级不变，原 compat 兜底语义不变（仅编号后移为第 4 条）。

### 3.2 传播规则

- 租户上下文在 **认证中间件之后、业务 handler 之前** 确定，随后只读
- 业务代码 **只读** 上下文，不允许在 handler 内改写当前请求的租户上下文
- 后台/异步任务必须显式携带租户上下文（worker 消息带 `tenant_id`），禁止依赖进程级全局
- 上下文缺失且非 compat 模式 → 请求拒绝（500 分类为配置错误，401/403 分类为认证/授权错误，见 §5）。**适用范围**：未声明平台作用域的路由组；已声明为平台管理 API 的路由按 §3.1 第 3 条解析（见 §3.4）

### 3.4 平台作用域声明（V1.1 澄清）

§3.3 要求平台管理 API 必须"显式声明"。声明的载体是**路由组**，也是 §3.1 第 3 条兜底的启用条件：

- **未声明**的路由组保持 V1 语义：multi 模式下无 claim 即拒绝（deny-by-default），不享有平台作用域兜底。
- **已声明**的路由组额外获得一条兜底，但需同时满足两个条件，缺一不可：
  1. 主体具**平台级能力** —— 全局角色（如 `platform_ops`）或全局（无 domain）Casbin policy 允许该路由；租户域内角色（`role:<key>@tenant:<id>`）不构成平台能力（§4：两个命名空间不合并、不隐式提升）。
  2. 主体**不属于任何租户** —— 无 active membership。该主体正是 §3.1 第 2 条里"零成员 → 无 claim 签发"的平台人群，两侧判定必须一致。
- 因此兜底永不覆盖第 1/2 条：有 claim 的主体走第 2 条；有 membership 的主体只能通过第 1 条显式 header 落到平台作用域（需平台级能力）。
- 兜底只决定**上下文**，不放宽 §3.3 的读取不变量：解析到 `tenant_id = 0` 的请求只能读写平台/全局行。
- 参考实现：`internal/middleware.PlatformScopeTenantContextMiddleware`（声明侧）与 `internal/middleware.TenantContextMiddleware`（未声明侧的严格语义），解析逻辑在 `pkg/tenant.Resolve`。

### 3.3 数据访问不变量（最关键）

- 所有带 `tenant_id` 的表，业务查询 **必须** 附加 `tenant_id = ctx.tenant_id`（或经统一 scope helper / GORM scope 注入）
- 禁止业务代码手写裸查询绕过 scope helper；cross-tenant 读取仅允许两种路径：
  1. 平台管理 API（显式声明，要求平台权限，落审计）
  2. 统计/报表聚合（经 platform 层专用通道，只读副本或只读事务）
- 写路径必须经统一 helper 注入 `tenant_id`，禁止从请求体读取 `tenant_id` 决定写入归属（唯一例外：平台管理 API 显式指定）

---

## 4. Casbin domain 维度（冻结）

- Casbin policy 的 domain 字段取 `tenant:<tenant_id>`；全局（compat/平台级）用 `tenant:0`
- 角色命名空间：租户内角色映射为 `role:<name>@tenant:<tenant_id>`；全局角色不携带 domain
- 权限检查顺序：先全局角色/权限（平台能力），再 domain 内角色（租户能力）；二者不合并、不隐式提升
- 存量 policy 迁移：compat 模式下全部视为 `tenant:0`，迁移由 runbook 执行，不做隐式转换

---

## 5. 错误码（冻结）

| 场景 | HTTP | code |
|------|------|------|
| header 指定租户但主体无平台权限 | 403 | `TENANT_FORBIDDEN` |
| 主体无该租户 membership 或 membership disabled | 403 | `TENANT_FORBIDDEN` |
| 租户 suspended / archived 时登录或写入 | 403 | `TENANT_SUSPENDED` / `TENANT_ARCHIVED` |
| 上下文缺失（非 compat） | 500 | `TENANT_CONTEXT_MISSING`（配置错误，告警） |
| 资源不属于当前租户（查询返回空，不泄露存在性） | 200/404 | 不引入新码：返回空集或 404，不区分"不存在"与"他租户" |
| 删除未归档租户（成员数是否为 0 不影响判定，V1.6） | 400 | `tenant.not_archived` |
| 添加成员时目标租户不存在或已删除（墓碑，先于用户检查，W2） | 404 | `tenant.not_found` |
| 添加成员时 `user_id` 不指向存活的 `system_user` 行（W2，V1.8 登记） | 404 | `user.not_found` |
| 添加成员时用户已是 `owner` 而请求非 owner 角色（非 owner 行重加幂等，W2，V1.8 登记） | 400 | `tenant.member_exists` |
| 添加成员时用户 owning tenant 与目标租户不一致（V1.7） | 400 | `tenant.member.user_not_in_tenant` |

---

## 6. Compat 模式（冻结）

- 开关：`system/config` 的 feature flag（`tenant.mode = compat | multi`，默认 `compat`），运行时可读、变更走 config 变更流程
- `compat` 模式行为承诺：
  - 现有单租户部署 **零迁移** 继续运行：`tenant_id = 0` 语义，解析优先级直接落 §3.1 第 3 条
  - 不校验 membership； Casbin 全部 `tenant:0`
  - 存量测试、存量 API 行为不变 —— 这是 canary 期（`2026-09-10-tenant-canary-slice`）的回归底线
- `multi` 模式行为：完整启用 §3/§4/§5
- **`multi` 是 `compat` 的超集**（V1.1 澄清）：multi 下 §3.1 第 4 条（compat 兜底）不再生效，但平台级/全局作用域（`tenant_id = 0`）依然是一等作用域（§3.1 第 3 条）。因此**同一部署内非租户（平台）主体与租户主体并存**，`platform.tenant_mode` 表达的是"是否启用租户维度"，而不是"整站二选一的世界开关"
- 模式切换只允许 compat → multi 单向（per 部署），回退 = 数据回滚（runbook），禁止 multi 运行后直接改回 compat 继续写入

---

## 7. 所有权边界（冻结）

| 关注点 | 归属域 | 禁止 |
|--------|--------|------|
| tenants 主数据 CRUD、生命周期 | `system/org`（租户即组织实体） | auth/iam/business 直接写 tenants 表 |
| membership 管理 | `system/org` 提供能力，`system/iam` 消费（登录时读取） | iam 维护 memberships 表 |
| 登录时租户解析与 claims 签发 | `system/auth` | auth 直查 tenants 表（经 org 的读接口/gRPC/进程内接口） |
| Casbin domain 规则 | `system/iam` | auth 或业务模块自建 domain 规则 |
| feature flag `tenant.mode` | `system/config` | 各模块自造开关 |
| tenant_id scope helper | `platform` | 各域复制粘贴私有实现 |
| 业务资源租户归属 | `business/*`（按矩阵登记） | business 定义自己的隔离机制 |

---

## 8. 验收锚点

- [租户就绪四项检查](../acceptances/BUSINESS_MODULE_ACCEPTANCE_MATRIX.md)（矩阵 T1–T4）适用于所有触碰租户语义的变更
- canary 切片（任务 3）是本合同行为承诺的首个运行态验证，其隔离测试即合同验收测试
- 合同变更：V1 冻结后修改需走 contract-change 流程（改版本号 + 变更记录 + 对应 runbook/matrix 同步）

---

## 9. 变更记录

| 版本 | 日期 | 变更 | 性质 |
|------|------|------|------|
| V1.1 | 2026-10-13 | §3.1 新增第 3 条「平台作用域兜底」（原第 3 条顺延为第 4 条，语义不变）；新增 §3.4「平台作用域声明」；§3.2 补注拒绝规则的适用范围；§6 补注「multi 是 compat 的超集」 | 澄清（Clarification） |
| V1.2 | 2026-10-19 | 新增 §2.3.1「读写义务」：读必须带租户谓词、写必须由上下文赋值且冲突键含租户、按裸键的归属校验同样算读；给出唯一两个放行面与 fail-closed 默认 | 澄清（Clarification） |
| V1.3 | 2026-10-23 | §2.1 补入第四态：`deleted` 是墓碑而非生命周期状态、不可作为输入、按 id/code 寻址视为不存在；明确删除标记是 `status` 而非预留的 `deleted_at`；明确删除归档租户时其成员关系同事务停用（§2.2 的 owner 要求只约束非归档租户） | 澄清（Clarification） |
| V1.4 | 2026-10-25 | 新增 §2.4「套餐定义与配额语义」（`tenant_plans` 模型、平台全局、`-1` 无上限 / `0` 非法、未知 code 回落编译内置表、退役而非删除、幂等 seed、5 秒读穿透缓存）与 §2.4.1「配额是度量不是闸门」（写路径不因超限拒绝；`QuotaEnforcer`/`MetricsCollector` 未接线）；新增 §2.5「平台侧租户运营读路径」（`/system/tenant-ops/*` 五条路由的声明平台作用域、`system:tenant:` 前缀、统一口径、有界分页、未知套餐如实呈现、指标集合冻结、权限↔路由 probe 门禁）；§1 In/Out 改写套餐与配额的归属边界；§2.1 `plan` 列注释由「V1 不消费」改为「解析为 §2.4 阈值」 | 扩展（Extension） |
| V1.5 | 2026-10-26 | §2.2 的「membership 删除 = `status=disabled`，不物理删除」改写为**按路径区分的四行表**（删除租户 / 删除用户 = 软禁用，显式移除成员 = 物理删除，重新加入 = 复活旧行），并把「审计链完整」的承担者从「行保留」更正为「全局操作日志」；§2.4 新增不变量 8，把 V1.4 登记为残余 gap 的**套餐指派面**移进 In 并接通前端（后端 `AssignableCodes` 删除）；同步在 EN 文本补译 §2.3.1（V1.2 遗留缺口）。写侧强制行为未改动，「把移除成员也改成软禁用」显式登记为未来的契约变更候选及其前置条件 | 澄清（§2.2）+ 扩展（§2.4 不变量 8） |
| V1.6 | 2026-10-27 | §5 登记删除守卫错误码 `tenant.not_archived`（400）。实现此前按「成员数 > 0 且未归档」拒绝删除，与 §2.1「删除仅允许对 `archived` 执行」不符：删光用户即可把活跃租户的活跃成员数降为 0，绕过归档直接删除。现按 §2.1 原文执行（守卫是 `status` 而非成员数），平台与 legacy 两个 handler 均注册该码，控制台对非归档行禁用删除入口；§2.1 与 §2.2 条文不变 | 澄清（Clarification）+ 扩展（§5 错误码） |
| V1.7 | 2026-10-27 | §2.2 新增不变量「跨租户 membership 不授予授权、也不在登录时提供」（W3 决策 A / TC-01）：登录发现、候选列表、可切换列表共用单一角色存在性谓词，`SwitchSessionTenant` 拒绝空角色集目标，`AddMember` 收窄到 owning tenant 用户并登记 §5 错误码 `tenant.member.user_not_in_tenant`（400）；登录选择器与 TenantSwitcher 控件移除、端点保留为过滤后的兼容面；存量跨租户 membership 行保留不清理，`/auth/current` 仍如实上报。数据层「一个 user 可属于多个 tenant」不变量不变 | 澄清（Clarification）+ 扩展（§5 错误码） |
| V1.8 | 2026-10-27 | 纯追加修订（W5 对齐，不改变任何不变量）：① §2.2 owner 转移表述对齐实现——**无单一转移端点**，落地路径为两步显式动作（加入新 owner → 移除旧 owner，`ensureNotLastOwner` 兜底，中间双 owner 是合法中间态），原文「转移 owner 是显式 API 动作」未对应任何端点；② §5 补登记 W2 成员完整性守卫（已实现于 PR #434）的两个错误码：`user.not_found`（404——`tenant_memberships` 对 `system_user` 无外键，无此守卫会写入指名不存在用户的幽灵行）与 `tenant.member_exists`（400——owner 降级拒绝，非 owner 行重加幂等），并注明墓碑检查先于用户存在性检查。In/Out 边界、生命周期与授权不变量均不变 | 澄清（Clarification） |
| V1.9 | 2026-10-28 | §2.4 新增「前端能力矩阵边界」段（澄清，Clarification）：前端能力矩阵只覆盖内建四档、`normalizeTenantPlan` 对未知 code fail-closed 归 `free`、自定义 code = 配额桶 + `free` 能力档、门控 UX-only（隐藏不授予也不撤销权限）；`outsideLadder` 警告文案五语言同步补「lowcode 门控面不可见」后果句。服务端 entitlements（自定义 code 完整参与能力门控）登记为 V2 商业化工作流，不在 V1；决策与选项裁剪见 [套餐能力阶梯语义决策](../reviews/TENANT_PLAN_LADDER_DECISION_2026-10-28.md)（W7 决策 A，maintainer-delegated）。不变量、错误码与写侧行为均不变 | 澄清（Clarification） |

**V1.5 的触发与范围**

- 触发：V1.4 收口时把 §2.2 的文档↔实现漂移作为「顺带登记、本轮不裁定」的条目留在变更记录里
  （见下方 V1.4 块最后一条）。2026-10-26 单独一轮把它处置完。
- 裁定：**改合同承认实现**，不改写侧。理由是四条路径里已有三条按软禁用工作并被测试钉住
  （`DeleteTenant`、`deactivateTenantMemberships` + `user_membership_test.go:152-200`），
  只有 `RemoveMember` 走物理删除；而把它改成软禁用会同时改掉「重复移除」的错误语义
  （现依赖 `RowsAffected == 0`）与成员列表口径，属于 §2.4.1 那类需要单独评估的写侧变更。
- 性质：**两部分，分别为 Clarification 与 Extension**。§2.2 那部分是 Clarification（条款语义层面）——
  「至少一个 owner」「命名空间分离」原文不动，只把「怎么下架」从单句改成按路径的表。§2.4 新增的不变量 8
  是 **Extension**：V1.4 把指派面登记为残余 gap（Out），本轮把它移进 In 并写成规范条款。
  其余 §2.1、§2.3、§2.3.1、§2.5、§3、§4、§5、§6、§7 的语义未改动；写侧强制仍留在 §2.4.1 的 Out 与门槛之后。
- **来源与代价（显式声明，不粉饰）**：「软禁用、不物理删除」不是随手写的措辞，而是 2026-09-10 合同冻结时的
  一条 Frozen 决策，理由是「审计链完整（审计评审人要求）」
  （`.harness/archive/2026-09/evidence/2026-09-10-tenant-contract-design/summary.md:24`）。本轮把合同改成
  追认现状，等于在**一条路径上承认该冻结要求未被实现**；这不是把文档改对了就消失的事实。
  因此本轮同时保留两件事：①§2.2 明确写「默认方向是软禁用，物理删除是需要收敛的例外」；②收敛候选及其
  前置条件已登记（见 §2.2 最后一条）。是否要反过来改实现，留给维护者在验收触点裁定——本合同不自行宣告
  该 Frozen 决策已被作废。
- 证据（逐条核对的代码位置已在 §2.2 表内给出）：`tenant_service.go:608-615`、`:869-888`、`:914-940`，
  `user/user_membership.go:45-51`、`:78-88`，`tenant_model.go:25-33`（`Membership` 无 `DeletedAt`，
  故 `Delete` 是 SQL `DELETE`），`operation_log_middleware.go:224-276`（非 GET 全量落库），
  `tenant_platform_handler.go:201-239`（平台侧增删有显式 audit 标题）。
- 指派面接通的实现与回归：`frontend/src/modules/system/iam/tenant/planOptions.ts`（派生规则，§2.4
  不变量 8 的前端镜像）、`TenantList.tsx`（编辑表单 + 筛选器改用派生结果）、`api.ts` 的
  `getPlanList({ skipErrorMessage })`（这份列表对表单是咨询性的，读不到时回落内置四档，不应弹错），
  以及 5 个新 i18n 资源里的 `system.tenant.plan.outsideLadder` 提示。
  单测 `frontend/tests/unit/modules/system/iam/tenant/planOptions.test.ts`（5 项：表不可读/空时回落、
  沿用服务端顺序并纳入自定义 code、停用只从写侧隐藏而读侧保留、当前 code 被带住且不重复、
  来自表里的内置 code 仍按内置标注）。后端 `AssignableCodes` 已删除（见上方 §2.4 那条 gap 的闭合说明）。
- 同轮附带（不改变合同语义）：EN 文本 `TENANT_CONTRACT_V1.en.md` 补译 §2.3.1「读写义务」（V1.2），
  并在其 §9 补上缺失的 V1.2 / V1.3 行；原先声明「EN 落后 V1.2 与 V1.3」的翻译注记本身已过期——
  V1.3 的墓碑内容早已在 EN §2.1 内，本轮一并更正该注记。
- 顺带登记 → 本轮以证据裁定为「前提不成立，无需改动」：曾登记「租户侧 `DELETE /tenants/:id/members/:user_id`
  的 handler 不设 audit 标题，只吃到全局中间件的默认标题，与平台侧的 `SetAuditMetadata(..., BusinessDelete)`
  不对称」。复查发现该路径属于 legacy `Handler`（`tenant_handler.go:244-263`），而它唯一的挂载入口
  `RegisterRoutes`（`tenant_routes.go:9-28`）在出货代码里没有调用方——除包内定义外，`RegisterRoutes` 与
  `NewHandler(` 的调用只出现在同包的 `tenant_handler_test.go`（6 处，不带包名前缀）；`system_modules.go:57`
  声明的字段类型本身就是 `*tenantiam.PlatformHandler`，出货的租户写路径由 `system_modules.go:290-306`
  全部挂到 `PlatformHandler`，每条都显式设标题（`tenant_platform_handler.go:131/148/168/203/223`，
  `tenant_ops_handler.go:70/87`）。`docs/api/TENANT_API.md:17` 早已写明这一组「not mounted」，2026-10-22
  的交付审计在运行态实测 `/api/v1/tenants` 返回 404。因此运行态不存在这条不对称；给一个没有路由的 handler
  补标题属于死代码，不做。

**V1.4 的触发与范围**

- 触发：2026-10-25 的租户运营信息架构改造。租户管理此前孤零零挂在「访问控制」下，平台侧没有
  任何全局视图能回答「这套部署有多少租户、每个租户占了套餐的多少」；而 `tenants.plan` 指向的阈值
  是 `GetDefaultQuota` 里的 switch，运营者改一个数字需要发一次版。维护者拍板「选 B 方案，要做就做彻底」，
  并把范围锁定为：信息架构与**读侧**视图做彻底，**写侧与商业化语义不动**。
- 性质：这是 **Extension（扩展）而非 Clarification（澄清）**。V1 原文把「套餐、配额」整体放在 §1 Out，
  本节把其中的**定义存储与读展示**从 Out 移进 In，同时用 §2.4.1 明确把**写侧强制**留在 Out 并给它
  加上 contract-change 门槛。既有 §3（上下文）、§4（Casbin domain）、§5（错误码）、§6（模式）、
  §7（归属边界）**语义均未改动**；§2.1/§2.2 只改了 `plan` 列的注释，状态机与成员不变量原文不动。
- 归属：`tenant_plans` 与 `/system/tenant-ops/*` 全部落在 `system/iam` 的 tenant 模块内，
  §7 的「tenants 主数据归 system/org、Casbin 归 system/iam」边界未被触碰；计划表是租户主数据的
  属性表，不是新的资源归属维度，因此不需要在 scope 矩阵新增 `tenant_id` 条目（§2.4 不变量 1）。
- 配套实现与回归测试：`backend/modules/system/iam/tenant/tenant_plan_model.go`、
  `tenant_plan_store.go`、`tenant_ops_view.go`、`tenant_ops_handler.go`、迁移 `000030`；
  `tenant_plan_store_test.go`（8 项：seed 幂等且不改写运营者编辑、经表解析且回落、缓存有界并被写入失效、
  创建要求五个阈值、非法 code 与重复、部分更新保留未给字段、退役被在用的 code 拒绝、
  未 seed 时指派校验回落内置集合）、
  `tenant_ops_view_test.go`（7 项：作用域排除平台行与墓碑、按套餐阈值报告占用、未知套餐如实呈现、
  过滤、跨租户成员带各自上限、分页稳定有界、`ListPlans` 每套餐租户数）；
  路由形状由 `permission_workbench_routes_test.go` 的 probe 镜像守卫。
- 残余 gap（显式）：套餐的**售卖**（下单、升降级、账期、按套餐裁剪菜单/能力之外的商业化语义）仍在 §1 Out；
  `max_api_keys` / `max_storage_mb` 两个维度本轮只有定义与落库，总览未报告其用量（§2.5 规则 6 的核心组不含它们）。
- 残余 gap（显式，指派面）：`PlanStore.AssignableCodes`（`tenant_plan_store.go:175`）已实现并有单测，
  但**出货路径没有调用方**——租户表单的套餐下拉仍是四个内置字面值（`frontend/.../tenant/TenantList.tsx:696-705`），
  所以新定义的自定义 code 能在控制台落库、能被读侧如实显示（`Known=false`），却暂时无法指派给租户。
  原因与 §2.4.1 同源：本轮锁定「读侧与阈值可配、写侧不动」，而指派面还与前端自有套餐层级
  （`core/entitlements/plans.ts` 的四档排序，未知 code 失败闭合为 free）耦合，需要单独一轮对齐。
  **→ 指派面已由 V1.5 接通**：`frontend/src/modules/system/iam/tenant/planOptions.ts` 从
  `/system/tenant-ops/plan/list` 派生选项（规则镜像 §2.4 不变量 8），`TenantList.tsx` 的编辑表单与筛选器都改用它，
  保存仍由服务端 `ValidateAssignment` 兜底。`AssignableCodes` 本身已删除：它按字母排序（阶梯顺序错），
  且从未有出货调用方，留着只会诱导第二套策略。
  **仍未闭合**的是前端能力阶梯：`core/entitlements/plans.ts` 依旧是四档硬编码、未知 code 失败闭合为 free，
  所以运营者自定义套餐的控制台**功能入口**按免费版展示（配额上限仍按定义生效）。表单用一行提示明示这一点，
  把它变成真正的按套餐裁剪菜单/能力仍是 §1 Out 的商业化语义，需要另行立项。
- 顺带登记（不在本轮 In 内，未改动 §2.2 原文）：§2.2 写着「membership 删除 = `status=disabled`（软禁用），
  不物理删除，保证审计链完整」，但 `Service.RemoveMember` 走的是物理 `Delete`（`Membership` 无 `DeletedAt`）。
  这是一个真实的合同/实现漂移，本轮不自行裁定。
  **→ 已由 V1.5 闭合**：裁定为「改合同承认按路径区分」，§2.2 原文已改写，写侧行为未动。

**V1.3 的触发与范围**

- 触发：2026-10-22 的交付验收把「`deleted` 未进合同」登记为 P3 残余；补写时实测发现该缺口背后
  有一个真实缺陷：`tenantStatusTransitionAllowed` 对未知的当前状态走 `default: return true`，
  而 `UpdateTenant` 读取租户行时不过滤墓碑，于是一个 `PUT {"status":"active"}` 就能把已删除的租户
  **复活**——它会带着删除时写入的合成 code（`__deleted_tenant_<id>`，违反本节的 `code` 正则）
  回到活跃列表，成员关系却仍是停用状态。同一条路径还允许给墓碑新增成员、对墓碑重新初始化。
- 缺口定位：§2.1 只列了三个状态与「删除仅允许对 archived 执行」，没有说清删除在存储层留下什么、
  留下之后是否还能被寻址。实现选择了 `status = 'deleted'` 的墓碑语义，合同没有承认它，于是
  「墓碑不是生命周期状态」这条约束既没被写下来、也没被状态机执行。
- 澄清范围：只在 §2.1 增加状态取值说明与三条不变量，**不新增概念、不改任何既有条款的语义**。
  §2.2 的 owner 要求原文不动（V1.3 只是明确它约束的是非归档租户，这正是原文「every non-archived
  tenant」的字面含义）；§3（上下文）、§4（Casbin domain）、§5（错误码）、§6（模式）、§7（归属边界）
  均未改动。删除仍只允许对 `archived` 执行，仍是运维动作。
- 配套实现与回归测试：`tenantStatusTransitionAllowed` 增加 `deleted` 分支、`GetTenantByID` 把墓碑
  视为不存在；`backend/modules/system/iam/tenant/tenant_lifecycle_guard_test.go`
  （`TestDeletedTenantCannotBeRevivedByUpdate`、`TestDeletedTenantRefusesNewMembers`、
  `TestDeletedTenantCannotBeReprovisioned`）。

**V1.2 的触发与范围**

- 触发：任务 `2026-10-19-tenant-permission-isolation-remediation` 的 W0。2026-10-19 的租户交付验收在实机测出
  6 个同源缺陷（F7/F8/F9/F10/F11/F13）：`system_role_data_scope` 早已带 `tenant_id` 与 `(tenant_id, role_key)` 唯一键，
  但读、写、校验三条路径里有一半按裸 `role_key` 工作，实测一个租户能列出 53 个角色（自己只有 3 个）、
  能给另一租户独有的角色写数据范围、且改写后让另一租户的普通用户可见行数从 1 变成 3。
- 缺口定位：§3.3 已经要求「所有带 `tenant_id` 的表必须附加租户谓词」，但它写在**上下文**一章，
  描述的是「查询」；§2.3 讲归属时只说了「哪些表带 `tenant_id`」，没有说「带了之后读写各有什么义务」。
  于是「表里有没有这一列」被当成合规判据，而「路径上有没有用这一列」没人检查。
- 范围：只在 §2.3 增加一个子节，**不新增概念、不改任何既有条款的语义**；§3.3 的 cross-tenant 两条路径、
  §4 的 Casbin domain、§5 的错误码、§7 的归属边界均未改动。本条把 §3.3 的义务**前移**到归属模型里，
  使「列存在但路径忽略」在模型层就可被判定为违规。
- 配套实现与回归测试：`2026-10-19-w1-data-scope-tenant-dimension`（W1），
  `backend/internal/middleware/data_scope_tenant_test.go` 与
  `backend/modules/system/iam/permission/permission_data_scope_tenant_test.go`。

**本次澄清的触发与范围**

- 触发：任务 `2026-10-13-tenant-console-remediation`。实测 multi 模式下 `dashboard` / `system/dict/*` / `system/setting/*` / `system/operation-log/*` / `auth/operation-verify` 全部 `500 tenant.context.missing`，`X-Tenant-Id` 对非 `platform_ops` 角色无效（403），平台控制台整体不可用；且因系统设置页同样 500，UI 内无法切回 compat（§6 亦禁止回切）。
- 缺口定位：§3.3 要求平台管理 API「显式声明」，但 §3.1 没有给出**声明之后如何解析上下文**——实现把兜底只留给了 compat，于是 multi 下的平台级主体落入 §3.2 的「上下文缺失」。
- 澄清范围：只补齐 §3 的解析语义与 §6 的模式语义。**§2（模型）、§4（Casbin domain）、§5（错误码）、§7（归属边界）均未改动**；§3.1 的优先级次序与第 1/2 条判定条件未改动。
- 配套实现：`pkg/tenant.Resolve`（`platform-fallback` 解析来源）、`internal/middleware.PlatformScopeTenantContextMiddleware`、`pkg/tenant.IsPlatformPopulationUser`；路由组声明清单见任务包 `2026-10-13-tenant-console-remediation`。

> 治理提示：本记录使 runbook `TENANT_MIGRATION_RUNBOOK.md` §8 的 G4（「无未闭合 contract-change」）不再自动成立。G4 的放行判定权在维护者，本澄清不自行宣告 G4 通过；生产 DDL/数据变更禁令不受本文件影响。
