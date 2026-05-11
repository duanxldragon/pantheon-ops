# 模块契约设计

更新时间：2026-04-17

类型：Design
归属层：platform
状态：Active

本文定义 Pantheon Base 的”模块契约”，目标是让后续所有模块都按统一方式接入，而不是各写各的。

它解决的问题是：

- 后端模块如何装配？
- 前端模块如何注册？
- 菜单、权限、i18n、seed 到底归谁管？
- 业务模块如何接入底座而不直接耦合实现？

如果这份契约不先定清楚，后续很容易变成：

- 每个模块写不同的初始化方式
- 前端路由和菜单各有一套来源
- 权限点、菜单、页面注册互相脱节
- AI 每次都按自己的理解生成新结构

## 1. 设计目标

- **统一接入方式**
- **明确职责边界**
- **支持 system 与 business 双层模块**
- **支持菜单 / 权限 / i18n / seed 同步注册**
- **支持后续逐步演进，不一次设计死**

## 2. 模块定义

Pantheon Base 中的“模块”指一个可独立注册、可独立演进、可独立拥有菜单/权限/i18n/seed 的功能单元。

模块分两类：

| 类型 | 说明 | 示例 |
| :--- | :--- | :--- |
| `system` | 底座公共模块 | `auth`、`user`、`role`、`menu`、`permission`、`dept`、`post` |
| `business` | 业务模块 | `order`、`project`、`ticket` |

## 3. 模块契约的核心原则

### 3.1 一个模块，一个边界

模块必须围绕单一能力域组织。

例如：

- `auth` 只负责认证与会话
- `user` 只负责用户管理
- `menu` 只负责菜单树和导航元数据

### 3.2 一个模块，一套注册信息

每个模块必须显式声明自己的：

- 路由
- 菜单
- 权限
- i18n
- seed

不允许这些信息分散在多个无关联文件里靠“记忆”维护。

### 3.3 契约先于实现

先定义模块输出什么，再写模块代码。

### 3.4 显式注册优先

当前阶段优先使用**显式注册**，不使用自动扫描。

原因：

- 可读性更高
- 更适合 AI 理解
- 更适合项目早期控制结构

## 4. 后端模块契约

## 4.1 后端模块职责

每个后端模块负责：

- 提供本模块路由
- 提供本模块 service / handler
- 提供本模块 seed 能力
- 提供本模块菜单和权限点种子
- 提供本模块 DTO 与数据模型

不负责：

- 直接初始化整个系统
- 修改其他模块内部实现
- 绕过契约直接耦合别的模块 service

## 4.2 推荐目录结构

```text
backend/modules/{system|business}/{module}/
  module.go
  *_handler.go
  *_service.go
  *_dto.go
  *_model.go
```

其中：

- `module.go` 用于导出模块装配入口
- 其他文件保持垂直切片

## 4.3 后端模块装配接口

建议每个模块最终统一导出一个模块定义对象，最少包含以下能力：

```go
type BackendModule interface {
    Name() string
    RegisterRoutes(r *gin.RouterGroup)
    SeedMenus(db *gorm.DB) error
    SeedPerms(db *gorm.DB) error
    SeedI18n(db *gorm.DB) error
}
```

如果后续需要数据库迁移，再扩展：

```go
Migrate(db *gorm.DB) error
```

### 4.3.1 当前阶段的最低要求

在真正引入统一接口前，文档层先约束模块必须具备以下“概念能力”：

- `RegisterRoutes`
- `SeedMenus`
- `SeedPerms`
- `SeedI18n`

即使短期仍在 `system.go` 手工装配，也必须按这个方向组织。

## 4.4 后端装配规则

### 4.4.1 根装配器职责

例如：

- `backend/cmd/server/main.go`
- `backend/modules/system/system.go`

根装配器只负责：

- 初始化基础设施
- 组装模块
- 挂接路由

根装配器不应承载：

- 模块内部业务逻辑
- 模块专属 SQL
- 模块专属校验逻辑

### 4.4.2 模块之间的调用规则

禁止：

- `business` 直接 import `system` 的 handler / service / repo
- `system` 之间互相深度调用内部实现

允许：

- 通过 `gin.Context` 获取认证主体
- 通过 `pkg/common`、`pkg/contracts` 获取稳定能力
- 通过显式定义的接口调用跨模块能力

## 4.5 后端 seed 契约

每个模块如果拥有平台元数据，必须同步提供 seed：

- 菜单 seed
- 权限 seed
- i18n seed

### 4.5.1 菜单 seed

用于：

- 侧边栏导航
- 角色授权树
- 面包屑基础数据

### 4.5.2 权限 seed

用于：

- 按钮权限
- 接口权限命名规范
- 初始角色授权

### 4.5.3 i18n seed

用于：

- 菜单标题
- 页面标题
- 按钮文案
- 错误 key 对应文案

## 5. 前端模块契约

## 5.1 前端模块职责

每个前端模块负责：

- 页面组件
- API 封装
- 路由注册
- 菜单元数据声明
- 权限点声明
- i18n namespace 声明（后续）

## 5.2 推荐目录结构

```text
frontend/src/modules/{system|business}/{module}/
  index.ts
  api.ts
  pages/
  components/
  locales/
```

当前阶段若还未拆 `pages/`、`components/`，允许过渡，但目标结构应向上面靠拢。

## 5.3 前端模块 manifest

当前 `ModuleConfig` 过于轻，只包含：

- `name`
- `routes`

这不足以支撑长期扩展。

建议目标演进为：

```ts
export interface ModuleManifest {
  name: string
  scope: 'system' | 'business'
  routes: ModuleRouteConfig[]
  menus?: ModuleMenuMeta[]
  permissions?: string[]
  i18nNamespaces?: string[]
  featureFlags?: string[]
}
```

### 5.3.1 当前阶段最少约束

在未升级真实类型前，所有模块至少要在文档层同步声明：

- 模块名
- 路由
- 菜单归属
- 权限点
- i18n key 前缀

## 5.4 前端路由注册规则

当前阶段采用：

- `src/core/router/modules.ts` 显式聚合注册

规则：

- 所有模块通过 `index.ts` 导出模块配置
- 所有模块统一在 `modules.ts` 中显式导入
- 不在 `App.tsx` 里直接散写业务路由

## 5.5 页面与菜单关系

前端要明确区分：

| 概念 | 说明 |
| :--- | :--- |
| `route` | 页面访问路径 |
| `menu` | 导航与信息架构元数据 |
| `permission` | 操作授权能力 |

三者相关，但不是同一个东西。

禁止：

- 把 `route` 当菜单
- 把 `menu` 当权限点
- 把 `list` 权限同时当作 `create/update/delete`

## 6. 菜单契约

## 6.1 菜单的定义

菜单是导航元数据，不是业务逻辑容器。

菜单至少应具备：

- `titleKey`
- `path`
- `component`
- `icon`
- `type`
- `sort`
- `isVisible`
- `module`

当前已落地的扩展元数据：

- `routeName`
- `isCache`
- `isExternal`
- `activeMenu`

## 6.2 菜单归属

每个菜单必须明确归属哪个模块。

例如：

- `system.user`
- `system.permission`
- `business.order`

## 6.3 菜单来源规则

当前阶段允许：

- 后端为主数据源
- 前端静态路由表为页面组件来源

目标阶段：

- 菜单元数据来自后端
- 前端通过 manifest 和路由映射完成组件加载

## 7. 权限契约

## 7.1 权限来源

权限分为两层：

- 菜单/按钮权限
- 接口权限

后续要统一命名规范，但在契约层先要求：

- 模块必须定义自己的权限点命名空间
- 权限点必须以模块为前缀

例如：

- `system:user:create`
- `system:role:update`
- `biz:order:approve`

## 7.2 权限注册要求

每个模块必须明确：

- 页面访问需要什么权限
- 页面按钮需要什么权限
- 接口需要什么权限

## 8. i18n 契约

## 8.1 i18n 命名空间

每个模块必须有自己的 key 前缀。

例如：

- `auth.*`
- `system.user.*`
- `system.role.*`
- `biz.order.*`

## 8.2 i18n 责任分工

后端负责：

- 返回 message key
- 提供语言包接口

前端负责：

- `t(key)` 渲染
- fallback 资源
- 缺失 key 告警（后续）

## 8.3 模块新增时的 i18n 要求

新增模块时，必须同步补：

- 菜单 key
- 页面标题 key
- 表单字段 key
- 按钮 key
- 错误提示 key

## 9. Seed 与文档同步契约

模块新增时，不允许只加代码。

必须同步补：

- menu seed
- permission seed
- i18n seed
- 文档说明

## 10. 业务模块接入契约

业务模块必须遵守：

- 不直接耦合底座实现
- 通过模块契约接入菜单、权限、i18n
- 不把业务页面写死进 system 模块

## 11. 当前阶段的实施要求

在还没有真正升级代码结构前，从现在开始先遵守以下最低标准：

### 后端

- 新模块必须有清晰装配入口
- 新模块必须有 seed 规划
- 新模块不能把路由、seed、逻辑分散乱放

### 前端

- 新模块必须通过 `index.ts` 导出注册对象
- 新模块必须列出权限点和 i18n key 前缀
- 新页面必须符合统一页面骨架设计

## 12. 当前文档落地到代码的差距

目前代码现状是：

- 后端已新增 `pkg/contracts.BackendModule` 契约，并通过 `RegisterBackendModules` 统一执行模块 seed 与路由注册
- `system.go` 已从单纯手写路由，演进为显式模块列表装配；`auth / user / role / menu / dept / post / permission / i18n` 已进入同一装配模型
- 前端 `ModuleConfig` 已升级为包含 `scope / menus / permissions / i18nNamespaces / pagePermission` 的 manifest
- 页面级权限已通过 `RoutePermissionGuard` 接入，声明 `pagePermission` 的路由会统一进入 403 兜底

这很正常，因为当前阶段本来就是先补文档、再按文档重构代码。

当前仍需继续收口：

- 后端模块 seed 仍处于“部分按子域拆分”阶段，后续应逐步迁移到各模块自己的 `module.go`
- 前端 manifest、后端菜单 seed、组件注册表已接入基础一致性检查
- 后续应把一致性检查从源码正则解析继续演进为标准契约数据源
- 业务模块已用 `business/cmdb` 做端到端验证，后续还需要第二个 `business/*` 模块重复验证

## 13. 验收问题

当以下问题都能明确回答时，说明模块契约设计是完整的：

- 新模块由谁注册？
- 菜单从哪里来？
- 权限由谁定义？
- i18n key 谁负责补？
- 前后端模块如何一一对应？
- 业务模块如何避免耦合底座？
- `component key` 是否同时存在于前端注册表与后端白名单？
- `pagePermission` 与按钮权限是否全部声明到模块 manifest 的 `permissions`？

## 14. 下一份建议补的文档

在本文件之后，建议继续补：

- `docs/designs/BUSINESS_MODULE_TEMPLATE.md`
- `docs/acceptances/ACCEPTANCE_CHECKLIST.md`

因为模块注册规则定下来后，下一步要解决的就是：

- 新业务模块到底按什么模板接入；
- 每次设计或实现完成后，如何统一验收。
