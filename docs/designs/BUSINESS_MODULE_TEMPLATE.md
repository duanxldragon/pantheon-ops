# 业务模块设计模板

更新时间：2026-04-17

类型：Design
归属层：platform
状态：Active

本文不是某个具体业务模块的实现说明，而是一份**标准模板**。

它解决的问题是：

- 后续新增 `business/*` 模块时，设计文档应该怎么写？
- 业务模块如何接入平台，而不把底座耦合坏？
- 一个“可上线、可维护、可扩展”的业务模块，最少要补哪些设计？

如果没有这份模板，项目后续很容易出现：

- 每个业务模块设计粒度不一致
- 有的模块只有页面，没有权限模型
- 有的模块只有接口，没有菜单和 i18n
- AI 和人工都按自己的理解各写各的

## 1. 适用范围

本文适用于所有未来新增的：

- `backend/modules/business/*`
- `frontend/src/modules/business/*`
- `database/*` 中与业务模块相关的 DDL
- 与业务模块相关的菜单、权限、字典、配置、审计设计

配套评审清单：

- `docs/designs/BUSINESS_MODELING_REVIEW_CHECKLIST.md`

它适合作为每个业务模块设计文档的母版，例如：

- `docs/business/ORDER_MODULE_DESIGN.md`
- `docs/business/PROJECT_MODULE_DESIGN.md`
- `docs/business/TICKET_MODULE_DESIGN.md`

## 2. 模块设计文档标准目录

每个业务模块设计文档建议至少包含以下章节。

### 2.1 模块概述

必须说明：

- 模块中文名 / 英文名
- 所属业务域
- 模块目标
- 本模块解决什么问题
- 本模块不负责什么

示例：

- `订单模块（order）`：负责订单全生命周期管理
- 不负责支付网关实现，不负责底层认证与权限实现

### 2.2 边界与依赖

必须先明确模块层级：

- 属于 `business/*`，不是 `system/*`

必须明确依赖边界：

- 依赖哪些平台公共能力
- 依赖哪些系统域能力
- 禁止直接依赖哪些实现

建议用表格写清楚：

| 类别 | 允许依赖 | 禁止依赖 |
| :--- | :--- | :--- |
| 平台层 | `pkg/common`、`pkg/contracts`、`gin.Context` | 直接耦合启动装配逻辑 |
| 系统域 | `auth` 上下文、`iam` 权限校验结果、`dict` 枚举能力 | 直接 import `system/*` 内部 service |
| 业务域 | 本模块内部能力、显式声明的跨模块接口 | 跨业务模块直接深度耦合 |

### 2.3 业务对象与术语

必须列出核心业务对象与统一术语，例如：

- 主实体
- 子实体
- 聚合根
- 状态机节点
- 关键编号规则

要避免同一个概念在前后端和文档里使用多个名字。

### 2.4 业务流程

至少补以下内容：

- 主流程
- 异常流程
- 状态流转
- 审批/确认/撤销等关键动作

建议用步骤或状态表描述，不要求一开始就画图，但必须清晰表达：

- 谁发起
- 谁审批
- 谁能撤销
- 哪些状态可以编辑
- 哪些状态只能查看

### 2.5 数据模型设计

必须说明：

- 主表
- 明细表
- 扩展表
- 关联关系
- 唯一约束
- 索引
- 软删 / 审计字段 / 租户字段（若预留）

并且必须补一段“租户就绪判断”：

- 当前模块是明确单租户，还是未来可能租户化
- 若未来可能租户化，第一版是否直接增加 `tenant_id`
- 当前唯一键是平台全局唯一，还是未来应当改成租户内唯一
- 列表 / 导出 / 统计接口未来是否需要统一注入 tenant 过滤

禁止默认跳过这一步，否则后续业务扩张时会把多租户改造成本直接堆高。

建议最少包含一张表：

| 表名 | 说明 | 关键字段 | 索引/约束 |
| :--- | :--- | :--- | :--- |
| `biz_order` | 订单主表 | `id`、`order_no`、`status` | `uk_order_no` |

并同步说明：

- 表前缀必须为 `biz_`
- 字段命名规则
- 枚举是否来自字典
- 是否需要历史表 / 日志表 / 快照表

### 2.6 API 设计

必须列出：

- 模块 API 前缀
- 核心接口清单
- 请求参数
- 响应结构
- 错误 key
- 权限点

建议使用如下表格：

| 接口 | 方法 | 说明 | 权限点 |
| :--- | :--- | :--- | :--- |
| `/api/orders` | `GET` | 订单列表 | `biz:order:query` |
| `/api/orders` | `POST` | 创建订单 | `biz:order:create` |
| `/api/orders/:id/approve` | `POST` | 审批订单 | `biz:order:approve` |

必须额外说明：

- 哪些接口是列表类
- 哪些接口是状态动作类
- 哪些接口要记审计日志
- 哪些接口必须做幂等控制

### 2.7 权限模型

业务模块不能只写一个 `list` 权限。

至少要拆为：

- 导航权限
- 页面访问权限
- 按钮/动作权限
- 接口权限

建议最少覆盖：

- `query`
- `detail`
- `create`
- `update`
- `delete`
- `export`
- `approve` / `submit` / `close` 等业务动作

并明确：

- 哪些权限决定菜单可见
- 哪些权限决定页面按钮显示
- 哪些权限决定后端接口可访问

### 2.8 菜单与导航设计

必须说明：

- 一级导航归属
- 菜单层级
- 路由路径
- `titleKey`
- `routeName`
- `module`
- `component key`
- 页面入口
- 详情页是否进入菜单

建议表格：

| 菜单 key | 路径 | 标题 key | 路由名 | 组件键 | 类型 | 说明 |
| :--- | :--- | :--- | :--- | :--- | :--- | :--- |
| `business.order` | `/business/order/list` | `biz.order.menu.list` | `business-order-list` | `business/order/OrderList` | `C` | 订单列表页 |

约束：

- 菜单只表达导航，不承载业务逻辑
- 菜单标题必须使用 i18n key
- `module` 必须使用 `business.{module}` 命名
- `component key` 必须进入前端受控组件注册表和后端菜单组件白名单
- 详情页、创建页通常不作为侧边栏一级菜单

### 2.9 前端页面设计

必须说明该模块包含哪些页面类型：

- 列表页
- 详情页
- 新建/编辑页
- 审批页
- 配置页
- 关联弹窗 / 抽屉

每类页面至少要写清：

- 页面目标
- 页面骨架类型
- 筛选区字段
- 表格列
- 操作按钮
- 表单字段
- 状态要求

必须引用已有规范：

- `docs/designs/FRONTEND_UI_SPEC.md`
- `docs/designs/FRONTEND_PAGE_TEMPLATES.md`

### 2.10 多语言设计

必须声明本模块 i18n key 前缀，例如：

- `biz.order.*`

并至少补齐：

- 菜单标题
- 页面标题
- 筛选项
- 表格列名
- 表单字段
- 按钮
- 状态文案
- 错误提示
- 空状态文案

### 2.11 字典与配置依赖

如果模块依赖平台字典或平台设置，必须显式写明：

- 依赖哪些字典类型
- 依赖哪些平台配置
- 哪些配置是运行期可修改的
- 哪些配置只允许后端控制

例如：

- `order_status`
- `payment_channel`
- `order.autoCancelMinutes`

### 2.12 审计与安全要求

必须说明：

- 哪些动作要写操作日志
- 哪些动作要记录安全事件
- 哪些字段要脱敏
- 是否涉及敏感数据导出
- 是否需要防重复提交、状态防抖或审批防并发

### 2.13 Seed 与初始化要求

新增业务模块时，设计文档必须说明是否需要：

- 菜单 seed
- 权限 seed
- i18n seed
- 字典 seed
- 初始化演示数据

没有 seed 的模块，后续集成通常会缺一半。

### 2.14 测试与验收

至少要列出：

- 接口测试点
- 权限测试点
- UI 状态测试点
- 多语言测试点
- 审计测试点

并引用：

- `docs/acceptances/ACCEPTANCE_CHECKLIST.md`

## 3. 标准章节模板

下面是一份可直接复制的新模块文档骨架。

```md
# XXX 模块设计

## 1. 模块概述
## 2. 边界与依赖
## 3. 核心业务对象
## 4. 业务流程与状态流转
## 5. 数据模型设计
## 6. API 设计
## 7. 权限模型
## 8. 菜单与路由设计
## 9. 前端页面设计
## 10. 多语言设计
## 11. 字典与配置依赖
## 12. 审计与安全要求
## 13. Seed 与初始化
## 14. 风险与边界外事项
## 15. 测试与验收
```

## 4. 后端落地模板

新增业务模块时，后端至少应预留：

```text
backend/modules/business/{module}/
  module.go
  {module}_handler.go
  {module}_service.go
  {module}_dto.go
  {module}_model.go
```

约束：

- 必须有模块装配入口
- DTO 与 Model 分离
- 不直接返回 GORM Model
- 不把菜单、权限、i18n 规划遗漏在文档外

## 5. 前端落地模板

新增业务模块时，前端至少应预留：

```text
frontend/src/modules/business/{module}/
  index.ts
  api.ts
  pages/
  components/
  locales/
```

约束：

- 必须由 `index.ts` 导出模块注册信息
- 页面路由必须使用 `componentKey`，不要继续直接在 manifest 中写 `component: lazy(...)`
- 菜单 manifest 必须声明 `path / titleKey / routeName / module`
- 路由 manifest 的 `pagePermission` 必须同步出现在 `permissions`
- 页面优先复用统一页面骨架
- 所有文案必须使用 `t()`
- 页面状态必须覆盖 `loading / empty / error / forbidden / submitting`

## 6. DDL 模板要求

业务模块 DDL 设计至少要补齐：

- 主表
- 唯一索引
- 查询索引
- 审计字段
- 状态字段
- 时间字段

推荐字段基线：

- `id`
- `created_at`
- `updated_at`
- `created_by`
- `updated_by`
- `deleted_at`（如采用软删）

如果是编号型业务对象，建议明确：

- 编号生成规则
- 是否要求业务唯一
- 是否允许人工修改

## 7. 模块接入检查清单

每新增一个业务模块，至少自查以下事项：

- 是否明确属于 `business/*`
- 是否写清与 `auth / iam / org / config` 的边界
- 是否有 DDL 与索引说明
- 是否列出 API 清单
- 是否列出权限点清单
- 是否列出菜单结构
- 是否列出 i18n key 前缀
- 是否说明依赖哪些字典和系统设置
- 是否说明审计点
- 是否说明测试与验收方式
- 是否已把页面组件键加入 `frontend/src/core/router/componentRegistry.ts`
- 是否已把页面组件键加入后端菜单组件白名单
- 是否已通过 `cd frontend && npm run check:menu-contract`

## 8. 与其他文档的边界

本文只定义“业务模块应该怎么设计”。

它与其他文档的关系如下：

| 文档 | 负责什么 | 不负责什么 |
| :--- | :--- | :--- |
| `DESIGN.md` | 顶层架构方向 | 某个业务模块的细节模板 |
| `docs/designs/MODULE_CONTRACT.md` | 模块如何注册与接入 | 某个业务模块内部的业务设计 |
| `docs/designs/PERMISSION_MODEL.md` | 平台权限分层与命名规范 | 某个业务模块自己的权限清单 |
| `docs/designs/ERROR_CODE_AND_I18N.md` | 错误 key 与 i18n 责任分工 | 某个业务模块的文案清单 |
| `docs/designs/FRONTEND_PAGE_TEMPLATES.md` | 页面骨架模板 | 某个业务模块具体字段和按钮 |
| `docs/acceptances/ACCEPTANCE_CHECKLIST.md` | 阶段验收标准 | 模块设计结构本身 |

## 9. 完成定义

如果一个业务模块满足以下条件，说明它的设计文档达标：

- 说清了模块边界
- 说清了数据模型
- 说清了 API
- 说清了权限
- 说清了菜单
- 说清了页面骨架
- 说清了 i18n
- 说清了字典/配置依赖
- 说清了审计与测试

只要以上任意一项缺失，后续实现都容易返工。
