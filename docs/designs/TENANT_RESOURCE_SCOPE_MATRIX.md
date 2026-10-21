---
title: 租户资源 scope 矩阵模板与全局唯一键登记
doc_type: Design
layer: platform
status: Active
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
  - docs/contracts/SYSTEM_CONFIG_CONTRACT.md
  - docs/contracts/TENANT_CONTRACT_V1.md
updated_at: 2026-09-10
---

# 租户资源 scope 矩阵模板与全局唯一键登记

English version: [TENANT_RESOURCE_SCOPE_MATRIX.en.md](./TENANT_RESOURCE_SCOPE_MATRIX.en.md)

本文是 `2026-09-10-tenant-ready-guardrails` 任务 0 的交付物，落实
[单租户先行、租户就绪设计](./TENANT_READY_SINGLE_TENANT_DESIGN.md) 第 5 节的规则，
为后续 [租户合同设计](../../.harness/archive/2026-09/tasks/2026-09-10-tenant-contract-design/task.md)
提供资源分类模板和唯一键登记入口。

> 当前运行态是单租户。本文是治理模板，不是多租户实现承诺；
> 生成器的 `tenant` 数据范围选项已下线。

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
| `system_user` | 待合同冻结 | 用户与租户关系（单/多 membership）未冻结前不归类 |
| `system_role` / `system_menu` / `system_permission` | 待合同冻结 | 平台治理对象是否租户内复制由合同决定 |
| `system_dept` / `system_post` | `platform-global`（倾向） | 组织是组织，租户是租户，不混用 |
| 系统设置（`system_setting`） | `tenant-overridable` 候选 | canary 首选资源域（`system/config`） |
| 字典（`system_dict`） | `tenant-overridable` 候选 | 平台默认 + 租户覆盖 |
| 审计日志 | `derived`（跟随写操作来源） | 需保留可切分维度 |
| 导出文件 / 上传对象 | `derived`（跟随来源资源） | key 必须可携带租户维度，避免串租户命中 |
| 缓存 key | `derived`（跟随来源资源） | key 设计必须允许租户段 |
| 异步任务 | `derived`（跟随目标资源） | 上下文透传由合同冻结 |

## 3. 全局唯一键登记

未来可能租户化的资源，其唯一键必须登记，避免“先按全局唯一上线，后改租户内唯一”的返工：

| 表 | 唯一键 | 当前 scope | 目标 scope | 冲突策略 |
| :--- | :--- | :--- | :--- | :--- |
| `system_user` | username | 平台全局 | 待合同冻结 | — |
| `system_role` | role_key | 平台全局 | 待合同冻结 | — |
| `system_setting` | key（或 key+scope 组合） | 平台全局 | `tenant-overridable`：组合键候选 | 合同冻结 |
| 业务 `biz_*` 表 | 新增时逐表登记 | 按评审 | 按评审 | 按评审 |

登记规则：

1. 新 DDL 评审必须回答“平台全局唯一还是租户内唯一”（见业务模块验收矩阵 1.1 节四必答项）。
2. 登记表由租户合同任务接管维护；合同冻结前本表为唯一登记入口。
3. 目标 scope 为“待合同冻结”的键，禁止在合同冻结前提前改造成组合唯一键。

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
