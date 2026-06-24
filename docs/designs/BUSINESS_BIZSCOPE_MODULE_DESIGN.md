# 业务域模块设计

English version: [BUSINESS_BIZSCOPE_MODULE_DESIGN.en.md](./BUSINESS_BIZSCOPE_MODULE_DESIGN.en.md)

更新时间：2026-06-18

类型：Design
归属层：business/bizscope
状态：Active

本文定义 `pantheon-ops` 中 `business/bizscope` 模块的业务边界、数据模型、接口、UI 和验收要求。该模块负责治理“业务域”本身，并作为 CMDB 主机分配和 Deploy 任务信任边界的唯一业务来源。

## 1. 模块概述

`business/bizscope` 是运维平台的业务域治理模块，与 `business/cmdb`、`business/deploy` 平级，不属于 CMDB 子模块。

本模块解决：

- 维护业务域台账，明确 `code / name / owner / environment / status`。
- 作为主机绑定入口，把 CMDB 主机纳入明确的业务责任边界。
- 为 Deploy 提供可信的 `business_scope_id / business_scope_name` 来源，避免绕过业务域直接对主机发起部署。
- 提供业务域详情和主机数量概览，帮助业务和运维确认域边界。

本模块不负责：

- 主机资源台账、分组、标签规范，属于 `business/cmdb`。
- 软件包、模板、部署任务编排和执行，属于 `business/deploy`。
- 用户、角色、菜单、组织、字典、审计底座实现，属于 `system/*`。
- 平台壳层和共享 UI 规范，属于 `pantheon-base`。

## 2. 边界与依赖

| 类别 | 允许依赖 | 禁止依赖 |
| :--- | :--- | :--- |
| platform | 统一响应、模块注册、菜单与 i18n seed 契约 | 本地重定义平台壳层规则 |
| system/auth | 登录主体、JWT 会话上下文 | 直接依赖 auth service |
| system/iam | 页面权限、按钮权限、Casbin 结果 | 在业务模块内重写权限策略 |
| system/org | 数据范围上下文 | 直接依赖 org repository |
| business/cmdb | 通过主机绑定字段维护 `business_scope_*` 快照 | 直接接管 CMDB 的主机台账、标签和分组逻辑 |
| business/deploy | 作为任务创建时的业务域选择来源 | 绕过业务域为空或无效的主机直接建任务 |

边界约束：

- `business/bizscope` 是业务域定义的唯一落点，CMDB 和 Deploy 只能消费它的快照结果。
- 主机绑定与解绑只允许更新 `biz_cmdb_host.business_scope_id / code / name` 和必要的状态回写，不允许修改 CMDB 其他业务字段。
- Deploy 必须使用业务域作为目标过滤和审计上下文，不能长期依赖自由输入的业务域名称。

## 3. 数据模型

当前数据表：`biz_business_scope`

| 字段 | 类型 | 说明 |
| :--- | :--- | :--- |
| `id` | bigint | 主键 |
| `code` | varchar(255) | 业务域编码，唯一 |
| `name` | varchar(255) | 业务域名称 |
| `owner` | varchar(255) | 负责人 |
| `environment` | varchar(50) | 环境，当前支持 `dev` / `test` / `prod` |
| `status` | varchar(50) | 状态，当前支持 `active` / `inactive` |
| `remark` | text | 备注 |
| `created_at / updated_at / deleted_at` | datetime | 审计字段 |

关联快照：

- `biz_cmdb_host.business_scope_id`
- `biz_cmdb_host.business_scope_code`
- `biz_cmdb_host.business_scope_name`

设计约束：

- `code` 必须全局唯一。
- 删除前必须检查是否仍有主机绑定该业务域。
- 解绑主机时，如果主机状态仍为 `assigned`，回退到 `pending`；已上线主机不在本模块内回退运行态。

## 4. 业务流程

### 4.1 业务域维护

```text
创建业务域
  -> 填写 code / name / owner / environment / status / remark
  -> 校验 code 唯一
  -> 落表 biz_business_scope
```

```text
更新业务域
  -> 按需修改展示和治理字段
  -> 若 code 变更，仍需保证唯一
  -> 不主动批量回写历史部署快照
```

### 4.2 主机绑定

```text
选择业务域
  -> 查询可绑定主机（当前 business_scope_id 为空）
  -> 批量绑定 hostIds
  -> 回写 biz_cmdb_host.business_scope_id/code/name
  -> 主机状态从 pending 进入 assigned
```

### 4.3 主机解绑

```text
选择某业务域下已绑定主机
  -> 清空 business_scope_id/code/name
  -> 若主机状态为 assigned，则回退为 pending
  -> 若主机已 online，则不在 bizscope 模块内强制降级
```

## 5. API 设计

接口前缀：`/api/v1/business/bizscope`

| 方法 | 路径 | 说明 | 权限点 |
| :--- | :--- | :--- | :--- |
| `GET` | `/list` | 业务域分页列表 | `business:bizscope:list` |
| `GET` | `/options` | 业务域下拉选项 | `business:bizscope:list` |
| `GET` | `/:id` | 业务域详情 | `business:bizscope:view` |
| `GET` | `/:id/hosts` | 已绑定主机列表 | `business:bizscope:view` |
| `GET` | `/:id/available-hosts` | 可绑定主机列表 | `business:bizscope:view` |
| `POST` | `/:id/hosts/bind` | 绑定主机到业务域 | `business:bizscope:update` |
| `DELETE` | `/:id/hosts/:hostId` | 解绑单台主机 | `business:bizscope:update` |
| `POST` | `` | 创建业务域 | `business:bizscope:create` |
| `PUT` | `/:id` | 更新业务域 | `business:bizscope:update` |
| `DELETE` | `/:id` | 删除业务域 | `business:bizscope:delete` |

错误语义：

| Key | 说明 |
| :--- | :--- |
| `bizscope.code_exists` | 业务域编码重复 |
| `bizscope.in_use` | 业务域仍绑定主机，不能删除 |
| `bizscope.not_found` | 业务域不存在 |
| `param.invalid` | 请求参数无效 |

仓库级 canonical 错误语义仍以 `BUSINESS_ERROR_SEMANTICS_APPENDIX.md` 为准；本节只保留本模块局部摘要。

## 6. 菜单、路由与权限

后端菜单 seed 与前端模块注册当前都收敛到以下入口：

| 项 | 值 |
| :--- | :--- |
| 菜单标题 key | `operations.bizscope.menu` |
| 列表路由 | `/operations/business-scope` |
| 详情路由 | `/operations/business-scope/:id` |
| 列表组件键 | `business/bizscope/BizScopeList` |
| 详情组件键 | `business/bizscope/BizScopeDetail` |
| 模块名 | `business.bizscope` |

权限点：

- `business:bizscope:list`
- `business:bizscope:view`
- `business:bizscope:create`
- `business:bizscope:update`
- `business:bizscope:delete`

约束：

- 列表页进入与详情页进入权限分离：当前列表路由使用 `list`，详情路由使用 `view`。
- 绑定主机、解绑主机和业务域字段更新统一归于 `update`。
- 删除权限不兜底查看和编辑权限。

## 7. 前端页面设计

当前前端模块路径：`frontend/src/modules/business/bizscope`

### 7.1 列表页 `BizScopeList`

- 骨架：hero + 筛选区 + 表格 + 分页。
- hero 指标：业务域总数、启用中数量、生产域数量。
- 筛选字段：`code`、`name`、`owner`、`environment`、`status`。
- 表格列：编码、名称、负责人、环境、状态、备注、操作。
- 操作：详情、编辑、删除；批量删除仅在有删除权限时出现。
- 空态文案：`business.bizscope.empty`。

### 7.2 表单 `BizScopeForm`

- 用于创建和编辑业务域。
- 字段：编码、名称、负责人、环境、状态、备注。
- 环境枚举：`dev / test / prod`。
- 状态枚举：`active / inactive`。

### 7.3 详情页 `BizScopeDetail`

- 顶部使用业务详情页 hero 指标区，至少展示 `hostCount`、状态、环境、更新时间。
- 展示编码、名称、负责人、环境、状态、备注、更新时间。
- 展示当前已绑定主机列表，字段至少包含 `hostname / ip / os / status`。
- 当拥有 `business:bizscope:update` 时，页面内提供“绑定主机”入口，并通过抽屉或等价的二级表面展示可绑定主机列表。
- 绑定主机只展示当前 `business_scope_id` 为空的主机；绑定后刷新详情与已绑定主机列表。
- 已绑定主机支持单台解绑；解绑后刷新详情与已绑定主机列表。
- 当前接口已经返回 `hostCount`，概览字段必须优先复用现有详情接口而不是新增接口。

### 7.4 UI 约束

- 沿用 base 的系统页模板、状态页、主题 token 与响应式规则。
- 不在 ops 本地定义新的视觉规范。
- 触碰该模块 UI 时，仍先遵守 `pantheon-base` 的 `FRONTEND_UI_SPEC.md`、`BACKOFFICE_STYLE_CONSTRAINTS.md`、`MOBILE_RESPONSIVE_BREAKPOINTS.md`。

## 8. i18n 与审计

当前 i18n 命名空间：`business.bizscope`

已存在的关键 key：

- `operations.bizscope.menu`
- `operations.bizscope.detail`
- `business.bizscope.hero.*`
- `business.bizscope.field.*`
- `business.bizscope.environment.*`
- `business.bizscope.status.*`
- `business.bizscope.permission.*`
- `business.bizscope.audit.*`

审计动作：

- `business.bizscope.audit.create`
- `business.bizscope.audit.update`
- `business.bizscope.audit.delete`

说明：

- 主机绑定/解绑当前复用 `business.bizscope.audit.update`。
- 如果后续需要把“字段更新”和“主机绑定变更”拆分成更细粒度审计动作，应优先补 key 和验收，而不是继续复用模糊 action。

## 9. 与 CMDB / Deploy 的协同

- CMDB 主机在业务域为空时视为未分配资源，不应直接进入正式部署目标。
- Deploy 任务创建必须选择有效业务域，并以其作为目标过滤和审计上下文。
- `business/bizscope` 是 `business/cmdb` 和 `business/deploy` 之间的业务边界锚点，不应由任一侧私自复制一套业务域定义。

## 10. 验收与后续清理要求

最小验收：

- 列表、详情、新增、编辑、删除链路可用。
- 重复编码返回明确错误。
- 仍绑定主机的业务域不能删除。
- 主机绑定与解绑能够正确回写 `biz_cmdb_host.business_scope_*` 字段。
- 菜单、权限、i18n 与页面入口一致。

文档治理要求：

- `docs/README.md` 与仓库 README 必须把 `business/bizscope` 作为正式业务模块入口列出。
- 若后续该模块继续扩展主机绑定 UI 或 Deploy 约束，应优先更新本设计文档，而不是只改 `CMDB` 或 `Deploy` 文档中的侧面描述。
