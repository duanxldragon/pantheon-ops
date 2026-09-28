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
updated_at: 2026-09-10
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

### Out

- 独立 schema / 独立 database 隔离模型（V1 仅共享 schema + `tenant_id`）
- 租户计费、套餐、配额（归商业域，另行立项）
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
  status        varchar(16)        -- active | suspended | archived
  plan          varchar(32)        -- 预留：套餐标记（V1 不消费）
  created_at / updated_at / deleted_at
```

不变量：
- `code` 创建后不可变；匹配 `^[a-z][a-z0-9-]{1,62}$`
- `status` 流转仅允许：`active ↔ suspended`，`active|suspended → archived`；`archived` 终态不可逆
- `archived` 租户禁止登录、禁止发放新 token、禁止产生新业务写入
- 删除（`deleted_at`）仅允许对 `archived` 租户执行，且为运维动作，不暴露给业务 API

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
- 每个非 archived 租户至少一个 `owner`；转移 owner 是显式 API 动作
- membership 删除 = `status=disabled`（软禁用），不物理删除，保证审计链完整
- 租户内角色（owner/admin/member）与全局 IAM 角色（`iam roles`）**命名空间分离**，不做隐式映射

### 2.3 资源租户归属（tenant_id 传播）

- 需要租户归属的表统一增加 `tenant_id bigint NOT NULL DEFAULT 0`：
  - `tenant_id = 0` 保留给 **平台级 / 全局资源**（compat 模式下的存量数据即全局资源）
  - `tenant_id > 0` 表示租户私有资源
- 哪些表带 `tenant_id`、哪些保持全局，由 [租户资源 scope 矩阵](../designs/TENANT_RESOURCE_SCOPE_MATRIX.md) 登记，新增表必须在登记表注册后才允许合入（见验收矩阵 T4）
- **禁止**：`tenant_id` 可空（NULL 语义歧义）；用字符串租户编码做外键；业务代码直接 `WHERE tenant_code = ?`

---

## 3. 租户上下文（冻结）

### 3.1 解析优先级（从高到低）

1. 显式 header `X-Tenant-Id`（平台运维/后台管理场景，要求平台级权限 + 审计）
2. 认证主体（JWT claims 中的 `tenant_id`，登录时根据"用户默认租户"或登录请求指定租户签发）
3. Compat 模式兜底：`tenant_id = 0`

### 3.2 传播规则

- 租户上下文在 **认证中间件之后、业务 handler 之前** 确定，随后只读
- 业务代码 **只读** 上下文，不允许在 handler 内改写当前请求的租户上下文
- 后台/异步任务必须显式携带租户上下文（worker 消息带 `tenant_id`），禁止依赖进程级全局
- 上下文缺失且非 compat 模式 → 请求拒绝（500 分类为配置错误，401/403 分类为认证/授权错误，见 §5）

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

---

## 6. Compat 模式（冻结）

- 开关：`system/config` 的 feature flag（`tenant.mode = compat | multi`，默认 `compat`），运行时可读、变更走 config 变更流程
- `compat` 模式行为承诺：
  - 现有单租户部署 **零迁移** 继续运行：`tenant_id = 0` 语义，解析优先级直接落 §3.1 第 3 条
  - 不校验 membership； Casbin 全部 `tenant:0`
  - 存量测试、存量 API 行为不变 —— 这是 canary 期（`2026-09-10-tenant-canary-slice`）的回归底线
- `multi` 模式行为：完整启用 §3/§4/§5
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
