---
title: 租户资源 scope 矩阵模板与全局唯一键登记
doc_type: Design
layer: platform
status: Active
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
  - docs/contracts/SYSTEM_CONFIG_CONTRACT.md
  - docs/contracts/TENANT_CONTRACT_V1.md
updated_at: 2026-10-22
---

# 租户资源 scope 矩阵模板与全局唯一键登记

English version: [TENANT_RESOURCE_SCOPE_MATRIX.en.md](./TENANT_RESOURCE_SCOPE_MATRIX.en.md)

本文是 `2026-09-10-tenant-ready-guardrails` 任务 0 的交付物，落实
[单租户先行、租户就绪设计](./TENANT_READY_SINGLE_TENANT_DESIGN.md) 第 5 节的规则，
为后续 [租户合同设计](../../.harness/archive/2026-09/tasks/2026-09-10-tenant-contract-design/task.md)
提供资源分类模板和唯一键登记入口。

> 运行态是 **compat（默认）与 multi 双模**：compat 下所有行落在 `tenant_id = 0`，multi 下按合同
> §6 启用租户维度。本文是给已交付的双模模型做资源分类与唯一键登记的治理模板，不是“未来才做多租户”的
> 承诺；`multi` 是 `compat` 的超集（合同 §6）。生成器的 `tenant` 数据范围选项仍下线。
> 分类义务与放行面以 [租户合同 V1](../contracts/TENANT_CONTRACT_V1.md) §2.3.1 为准。

---

## 1. 资源 scope 四分类

任何资源（表、配置、缓存、文件、异步任务）新增或评审时，必须归入以下四类之一：

| 分类 | 定义 | tenant 过滤 | 租户内唯一 | 典型例子 |
| :--- | :--- | :--- | :--- | :--- |
| `platform-global` | 平台级全局资源，永远不按租户切分 | 永不 | 不适用 | 平台菜单、平台字典、全局系统设置、平台管理员账号 |
| `tenant-owned` | 租户私有资源，行只属于一个租户 | 必须有 | 默认是 | 业务单据、租户配置覆盖、租户自定义字典 |
| `tenant-overridable` | 平台提供默认值，租户可覆盖 | 条件有 | 组合键 | 租户级系统设置覆盖、租户级角色模板 |
| `derived` | 派生/聚合数据，继承来源资源的 scope | 跟随来源 | 不适用 | 统计聚合、导出文件、报表缓存 |

判断规则：

1. 分类必须写入设计文档，不允许“默认全局”。
2. `tenant-owned` 与 `tenant-overridable` 资源的未来查询/导出/统计路径必须可叠加 tenant 过滤。
3. 分类变更视为共享合同变更，需要重新评审。

## 2. 初始资源盘点（按当前代码归类）

以下为当前底座主要资源的初步归类，作为合同任务冻结的输入，最终清单由租户合同任务维护：

| 资源 | 当前归类 | 备注 |
| :--- | :--- | :--- |
| `system_user` | `tenant-replicated` | 用户名**租户内唯一**（2026-10-13 维护者决策，迁移 000024 落地 `uk_system_user_tenant_username`） |
| `system_role` | `tenant-replicated` | **2026-10-15 维护者裁决：角色按租户复制**。每个租户持有自己的 `tenant_admin` / `tenant_member` 副本；角色**定义**是全局统一的，单一来源为 `pkg/tenant` 的角色常量与 `modules/system/iam/tenant` 的 `tenantRoleDefinitions`，由 `ProvisionTenant` 代码生成而非人工逐租户维护，因此不会产生漂移 |
| `system_menu` / `system_permission` | `platform-global` | **2026-10-15 维护者裁决：菜单与权限保持全局共享**。注意：迁移 000020 已为两表加了 `tenant_id` 列，但菜单/权限读写路径不按该列过滤，列当前未参与隔离语义（占位）。2026-10-21 验收整改：菜单写接口与 i18n 系统域写接口已加 `PlatformCapabilityMiddleware`，租户管理员不能再写全局菜单/权限表。租户差异只由 Casbin domain 表达（契约 §4），不做租户内菜单/权限定制 |
| `system_dept` / `system_post` | `platform-global`（倾向） | 组织是组织，租户是租户，不混用 |
| `tenants` / `tenant_plans` | `platform-global` | 2026-10-25 登记（合同 V1.4 §2.4）。两张表**都没有 `tenant_id` 列**：`tenants` 是租户主数据本身，`tenant_plans` 是它的属性定义表（套餐阈值）。本模块的隔离由各 service 显式 `WithTenantScope` 施加、不是全局插件，因此「不带该列」在这里是正确形状而非遗漏——给计划表加 `tenant_id` 会把平台控制台的读过滤成空集。唯一键 `tenant_plans.code` 全局唯一（`uk_tenant_plan_code`），没有租户内变体 |
| 系统设置（`system_setting`） | `tenant-overridable` 候选 | canary 首选资源域（`system/config`） |
| 字典（`system_dict_type` / `system_dict_item`） | `tenant-replicated`（自平台默认复制） | 表已带 `tenant_id`（000013 canary）。2026-10-21 验收整改：`ProvisionTenant` 初始化时将平台（tenant_id=0）字典类型与字典项幂等复制到新租户（INSERT…SELECT + NOT EXISTS），平台字典为单一来源、租户持有副本可覆盖 |
| 审计日志 | `derived`（跟随写操作来源） | 需保留可切分维度 |
| 导出文件 / 上传对象 | `derived`（跟随来源资源） | key 必须可携带租户维度，避免串租户命中 |
| 缓存 key | `derived`（跟随来源资源） | key 设计必须允许租户段 |
| 异步任务 | `derived`（跟随目标资源） | 上下文透传由合同冻结 |

## 3. 全局唯一键登记

未来可能租户化的资源，其唯一键必须登记，避免“先按全局唯一上线，后改租户内唯一”的返工：

| 表 | 唯一键 | 当前 scope | 目标 scope | 冲突策略 |
| :--- | :--- | :--- | :--- | :--- |
| `system_user` | username | 平台全局 | `tenant-unique`：`(tenant_id, username)` **已冻结**（2026-10-13） | 迁移 000024 已建 `uk_system_user_tenant_username` |
| `system_role` | role_key | 平台全局 | `tenant-unique`：`(tenant_id, role_key)` **已冻结**（2026-10-15） | 迁移 000024 已建 `uk_system_role_tenant_key`；角色按租户复制，复制体在租户内唯一 |
| `system_setting` | key（或 key+scope 组合） | 平台全局 | `tenant-overridable`：组合键候选 | 合同冻结 |
| 业务 `biz_*` 表 | 新增时逐表登记 | 按评审 | 按评审 | 按评审 |

**`system_user.username` 的推论（2026-10-15 补记）**：用户名既然只保证**租户内**唯一，那么「按用户名找登录主体」这个全局查找就必然要给出一个确定性赢家。实现约定为：平台主体（`tenant_id = 0`）优先，其次 id 升序；**所属租户已软删除的行不是登录候选**（该主体没有任何可承载会话的租户上下文，让它胜出会把活跃租户的同名用户遮蔽掉，登录成功但处处 `tenant.context.missing`）。软删除租户**不会**清理其 `system_user` 行，因此这条排除必须由查找侧保证，不能指望删除侧清干净。详见 `docs/TENANT_FULL_FLOW_REVIEW_2026-10-14.md` 缺陷 6 与 `docs/runbooks/TENANT_TROUBLESHOOTING.md` §5b。

登记规则：

1. 新 DDL 评审必须回答“平台全局唯一还是租户内唯一”（见业务模块验收矩阵 1.1 节四必答项）。
2. 登记表由租户合同任务接管维护；合同冻结前本表为唯一登记入口。
3. 目标 scope 为“待合同冻结”的键，禁止在合同冻结前提前改造成组合唯一键。
   > **回溯备案（2026-10-15）**：`system_user.username` 与 `system_role.role_key` 在迁移 000024
   > （2026-10-13「真隔离」任务）落地时已经改造成组合唯一键，当时本登记表仍是“待合同冻结”。
   > 该状态已由本轮裁决正式认可：`system_user` 随「用户名租户内唯一」决策冻结为租户内唯一，
   > `system_role` 随本轮「角色按租户复制」裁决冻结为租户内唯一。两条唯一键从现在起是**合同认可**的
   > 既成事实，不再算提前改造。

## 4. 租户识别方式（全部待决策，禁止按示例实现）

以下识别方式在合同冻结前一律不得实现，仅作为决策输入记录：

- Header（如 `X-Tenant-ID`）：可伪造，需网关级可信来源配合。
- 子域名：需域名规划与证书策略。
- 自定义域名：需租户域名管理能力。
- JWT claim：需先冻结 token 结构与刷新语义。

缺失、冲突或不可信的租户上下文对 `tenant-owned` / `tenant-overridable` 资源
必须默认拒绝（deny-by-default）；该语义由租户合同任务冻结。

---

## 5. 维护责任

| 事项 | 责任任务 |
| :--- | :--- |
| 资源分类最终冻结 | `2026-09-10-tenant-contract-design` |
| 唯一键冲突策略与迁移顺序 | `2026-09-10-tenant-migration-runbook` |
| 首个 canary 资源选择 | `2026-09-10-tenant-canary-slice`（合同 reviewer 批准） |
