# 权限模型设计

更新时间：2026-04-29

类型：Design
归属层：system/iam
状态：Active

本文定义 Pantheon Base 的权限模型，目标是把”菜单能不能看、页面能不能进、按钮能不能点、接口能不能调”拆清楚。

如果这份模型不先定好，后续最容易变成：

- 一个 `list` 权限控制所有操作
- 菜单和按钮权限混在一起
- 前端隐藏按钮但后端接口仍可调用
- 后端 Casbin 有策略，前端用户却不知道为什么没权限
- 角色授权页越来越难理解

## 1. 设计目标

- **四层权限清晰**
- **前后端职责一致**
- **菜单、页面、按钮、接口解耦**
- **支持当前 Casbin 实现**
- **支持后续数据权限、多租户扩展**

## 2. 权限四层模型

Pantheon Base 权限分四层：

| 层级 | 名称 | 控制内容 | 示例 |
| :--- | :--- | :--- | :--- |
| L1 | 导航权限 | 是否能在侧边栏看到菜单 | 用户管理菜单 |
| L2 | 页面权限 | 是否能访问页面路由 | `/system/user` |
| L3 | 操作权限 | 是否能看到/点击按钮 | 新增、编辑、删除 |
| L4 | 接口权限 | 是否能请求后端接口 | `POST /api/v1/system/user` |

四层之间有关联，但不能混为一谈。

## 3. 核心原则

### 3.1 菜单不是权限

菜单是导航信息架构。

权限是操作授权能力。

菜单可以承载权限点字段，但不能把菜单本身当作权限模型。

### 3.2 页面权限不等于列表权限

页面权限用于判断用户能不能进入页面。

列表查询权限只是页面内的一种接口/操作能力。

### 3.3 按钮权限必须细粒度

禁止用：

- `system:user:list`

同时控制：

- 新增
- 编辑
- 删除

必须拆成：

- `system:user:view`
- `system:user:create`
- `system:user:update`
- `system:user:delete`
- `system:user:reset`
- `system:user:batch-update`
- `system:dept:batch-update`
- `system:post:batch-update`

### 3.4 前端隐藏不等于安全

前端按钮权限只是体验层。

真正安全必须靠后端接口权限。

## 4. 当前实现状态

当前系统已经具备：

- `system_menu` 菜单树
- `system_role_menu` 角色菜单关系
- `system_menu.perms` 按钮权限标识
- `casbin_rule` 接口权限策略
- 前端 `usePermission` 按 `perms` 控制按钮

当前采用的是“双轨模型”：

| 轨道 | 数据来源 | 用途 |
| :--- | :--- | :--- |
| 菜单/按钮权限 | `system_menu` + `system_role_menu` | 导航树、按钮显隐 |
| 接口权限 | `casbin_rule` | 后端 API 访问控制 |

这比“只有 Casbin”更接近产品化权限，但还需要统一认知和命名。

### 4.1 当前治理进展（2026-04-29）

`system/iam` 的权限工作台已经从“纯展示页”演进为“治理视图”，当前已支持：

- `integrity=unknown | clean`：识别角色上是否存在未知权限分配
- `coverage=page-gap | api-gap | complete`：识别授权链路是否缺页面权限或 API 策略
- 治理报表导出：支持把当前筛选结果导出为盘点文件，供线下整改与复核
- 受控整改动作：支持在工作台内按推荐映射一键补齐单角色缺失的 Casbin 策略

这三类问题分别代表：

- 未知权限：权限键漂移，角色仍挂着系统已无法解释的授权标识
- 页面缺口：角色已有导航，但缺少页面访问权限，形成“看得见进不去”的断链
- API 缺口：角色已有页面/动作权限，但后端 Casbin 策略未补齐，形成“前端可点、后端被拒”的断链

当前结论：

- Pantheon 已具备 `system/iam` 授权治理第二层闭环：发现、解释、导出、受控整改
- 当前整改仍限定为“推荐映射 + 单角色 + 后端重算”，不支持前端任意提交路径/方法写入 Casbin

## 5. 推荐目标模型

## 5.1 权限实体

建议长期演进为以下概念：

| 概念 | 说明 |
| :--- | :--- |
| `Menu` | 导航结构 |
| `PagePermission` | 页面访问权限 |
| `ActionPermission` | 按钮/操作权限 |
| `ApiPermission` | 后端接口权限 |
| `DataPermission` | 数据范围权限，后续预留 |

## 5.2 数据关系

推荐关系：

```text
Role
  ├── Menus
  ├── PagePermissions
  ├── ActionPermissions
  └── ApiPermissions
```

当前阶段可以继续复用现有表，但设计上必须按这个模型理解。

## 6. 权限命名规范

## 6.1 命名格式

统一格式：

```text
{scope}:{resource}:{action}
```

示例：

```text
system:user:view
system:user:create
system:user:update
system:user:delete
system:user:reset
system:user:batch-update
system:dept:batch-update
system:post:batch-update
system:permission:manage
biz:order:approve
```

## 6.2 scope 规范

| scope | 说明 |
| :--- | :--- |
| `system` | 系统底座 |
| `auth` | 认证与安全中心 |
| `biz` | 业务模块 |

说明：

- `auth` 可以作为独立 scope，也可以短期归入 `system`，但文档和代码必须明确。
- 业务模块统一使用 `biz` 前缀，避免和底座冲突。

## 6.3 resource 规范

resource 使用模块或资源名：

- `user`
- `role`
- `menu`
- `permission`
- `dept`
- `post`
- `session`
- `dict`
- `setting`
- `i18n`
- `upload`
- `module`
- `generator`

## 6.4 action 规范

基础 action：

| action | 说明 |
| :--- | :--- |
| `view` | 查看页面或详情 |
| `list` | 查询列表 |
| `create` | 新增 |
| `update` | 编辑 |
| `delete` | 删除 |
| `reset` | 重置敏感凭据，如管理员重置密码 |
| `export` | 导出 |
| `import` | 导入 |
| `enable` | 启用 |
| `disable` | 禁用 |
| `assign` | 授权/分配 |
| `manage` | 管理型兜底权限，谨慎使用 |
| `refresh` | 刷新缓存或重载运行时资产 |
| `register` | 注册模块、挂接模块接入状态 |
| `unregister` | 卸载模块、解除模块接入状态 |
| `generate` | 触发受控代码生成 |

### 6.4.1 `view` 与 `list` 的区别

- `view`：能不能进入页面或查看详情
- `list`：能不能调用列表查询接口

简单模块可短期合并，但长期建议区分。

### 6.4.2 高敏治理动作约束

以下动作不应继续被宽泛权限兜底：

- 模块注册
- 模块卸载
- 触发代码生成
- 日志清理
- 批量删除高敏记录

原则：

- 页面可见权限不等于动作执行权限
- `use`、`manage` 这类宽泛动作只能作为短期兼容，不应长期替代高敏动作拆分

## 7. 导航权限设计

## 7.1 作用

导航权限决定用户侧边栏能看到什么。

当前通过：

- `system_role_menu`
- `GET /system/menu/tree?scope=nav`

实现。

## 7.2 规则

- 只返回当前用户授权菜单
- 只返回 `is_visible = 1`
- 不返回 `type = F` 的按钮节点
- 子菜单命中时自动补齐祖先节点

## 7.3 禁止行为

- 不允许前端拿全量菜单后自己裁剪
- 不允许未授权菜单只隐藏按钮不隐藏导航

## 8. 页面权限设计

## 8.1 作用

页面权限决定用户能不能进入某个路由。

当前前端主要只有 token 守卫，还缺统一页面级权限守卫。

## 8.2 建议目标

每个路由配置都允许声明：

```ts
requiredPerm?: string
```

例如：

```ts
{
  path: 'system/user',
  titleKey: 'system.menu.user',
  requiredPerm: 'system:user:view',
  component: lazy(() => import('./UserList'))
}
```

## 8.3 无权限体验

页面级无权限时：

- 渲染统一 403 页面
- 不应只依赖接口请求失败后弹 toast

## 9. 操作权限设计

## 9.1 作用

操作权限控制页面内按钮、菜单项、危险操作。

例如：

- 新增用户
- 编辑角色
- 删除菜单
- 下线会话

## 9.2 前端规则

默认规则：

- 无权限按钮直接隐藏
- 危险或需要教育用户的场景，可以 disabled + tooltip

## 9.3 后端规则

按钮隐藏不能替代接口鉴权。

所有写接口必须由后端权限模型兜底。

### 9.4 `system/config` 高敏能力的权限拆分

`system/config` 中必须特别区分两类能力：

普通治理能力：

- `system:i18n:update`
- `system:i18n:import`
- `system:i18n:export`
- `system:i18n:refresh`

高敏治理能力：

- `system:module:register`
- `system:module:unregister`
- `system:module:generate`

当前现状：

- 动态模块页已拆出 `system:module:list / register / unregister`
- 生成器页当前仍主要使用 `system:generator:use`

推荐目标：

- 页面进入权限：`system:generator:view` 或保留短期兼容的 `system:generator:use`
- 真正触发生成：`system:module:generate`

这样可以避免：

- 任何能进入生成器页的人都能直接生成模块

## 10. 接口权限设计

## 10.1 当前实现

当前接口权限走 Casbin：

```text
p, roleKey, path, method
```

### 10.1.1 当前工作台治理视图

当前 `GET /api/v1/system/permission/workbench` 会统一返回：

- 角色导航授权计数
- 页面权限计数
- 动作权限计数
- API 策略计数
- 未知权限计数
- 页面缺口标记 `hasPageGap`
- API 缺口标记 `hasApiGap`

同时支持：

- `GET /api/v1/system/permission/workbench?integrity=unknown`
- `GET /api/v1/system/permission/workbench?coverage=page-gap`
- `GET /api/v1/system/permission/workbench?coverage=api-gap`
- `GET /api/v1/system/permission/workbench/export`

例如：

```text
p, admin, /api/v1/system/user, POST
```

## 10.2 目标规则

- 每个受保护接口都必须有策略或白名单解释
- 管理员可以有通配策略
- 自助接口需要明确白名单

## 10.3 自助接口白名单

以下接口建议不依赖角色授权，只要求登录态：

- 当前用户信息
- 当前用户 profile
- 修改当前用户密码
- 当前用户会话列表
- 注销当前会话

这些不是“公共接口”，只是“不按角色菜单授权”。

## 11. 角色授权设计

## 11.1 当前问题

当前角色页已从单一菜单授权升级为三段式授权表单：

- 导航授权：维护 `system_role_menu`，决定侧边栏可见菜单；
- 页面授权：维护 `system_role_permission` 中来自菜单 `pagePerm` 的页面权限；
- 操作授权：维护 `system_role_permission` 中来自按钮节点 `perms` 的操作权限。

当前代码已补一版“权限工作台”：

- 从角色维度统一查看导航授权、页面权限、操作权限、接口策略；
- 保留 Casbin 策略独立维护入口；
- 先解决“前端不知道为什么没权限”的认知断层。

当前仍需明确一个边界：

- 角色页负责导航、页面、操作三类前端体验与路由权限；
- 权限页继续提供 Casbin 接口策略独立维护入口和角色维度的权限工作台统一查看；
- 角色页保存时会按“已知权限点 → API 策略”映射自动同步可推导的 `casbin_rule`，避免出现“菜单可见、页面可进、接口被拒”的断链；
- 自动同步仅覆盖系统已声明映射的 API 策略，未知或手工维护的 Casbin 策略仍由权限页治理。

## 11.2 目标体验

角色授权应逐步演进为：

```text
角色
  ├── 导航菜单
  ├── 页面权限
  ├── 按钮权限
  └── 接口权限
```

## 11.3 短期策略

短期不强行大改表结构。

优先做：

- 权限命名统一；
- 页面上说明“三轨模型”；
- 角色页按导航、页面、操作拆分授权编辑；
- 后端保存角色时把已选导航节点展开到子导航节点，并补齐这些页面节点的 `pagePerm`，操作权限仍需显式授权；
- 后端保存角色和迁移历史角色时根据 `system_role_permission` 中的已知权限点同步对应 `casbin_rule`；
- 权限页维护接口策略并提供统一工作台视图，工作台复用同一份权限点到 API 策略映射检查缺口。

## 12. Admin 角色规则

`admin` 是内置超级管理员角色。

规则：

- 默认拥有全部导航、页面、操作、接口权限
- 不允许被停用
- 不允许被删除
- 不允许移除内置管理员用户的 admin 角色

## 13. 权限与菜单节点类型

建议菜单类型：

| 类型 | 说明 |
| :--- | :--- |
| `M` | 目录 |
| `C` | 页面菜单 |
| `F` | 按钮/功能点 |

规则：

- `M/C` 可参与导航
- `F` 不参与导航
- `F` 可参与按钮权限
- `F` 必须挂在对应页面菜单下

## 14. 权限 seed 规范

每个模块必须声明自己的权限点。

示例：

```text
system:user:view
system:user:list
system:user:create
system:user:update
system:user:delete
system:user:reset
system:user:batch-update
system:dept:batch-update
system:post:batch-update
```

模块 seed 必须包含：

- 菜单节点
- 页面权限
- 按钮权限
- 默认 admin 绑定
- i18n key

## 15. 前端权限消费规范

前端只能通过统一 hooks 使用权限：

```ts
hasPerm('system:user:create')
hasAnyPerm(['system:user:create', 'system:user:update'])
```

禁止：

- 直接读取原始 token 自己解析
- 不同页面写不同权限判断
- 用角色名硬编码业务权限，除了 `admin` 兜底

## 16. 后端权限消费规范

后端受保护路由必须经过：

- JWT 身份校验
- Casbin 接口校验
- 必要的业务级校验

业务级校验示例：

- 不能删除自己
- 不能停用 admin
- 不能删除已被使用的角色

## 17. 数据权限预留

当前不实现数据权限，但设计上预留。

未来可能包含：

- 本人数据
- 本部门数据
- 本部门及下级
- 指定部门
- 全部数据

不要现在把数据权限硬编码进业务模块。

## 18. 验收清单

新增模块时必须回答：

- 这个模块有哪些页面？
- 每个页面的 `view` 权限是什么？
- 每个按钮的权限点是什么？
- 每个接口对应什么 Casbin 策略？
- admin 是否自动拥有权限？
- 菜单树是否只返回授权导航？
- 无权限页面是否显示 403？

### 18.2 `system/config -> dynamicmodule / generator` 额外验收项

- `/system/modules` 的页面权限是否独立于注册/卸载动作权限
- `/system/generator` 的页面权限是否独立于真正的生成动作权限
- 是否已明确 `system:module:generate` 的接口策略归属
- 若当前仍保留 `system:generator:use`，是否在文档中明确它只是短期兼容权限，而不是高敏动作的长期模型
- 注册、卸载、生成动作是否同时受页面权限、动作权限、Casbin、二次验证和环境限制保护

### 18.1 `system/config -> setting` 额外验收项

系统设置页是当前权限模型里容易误配的典型页面，验收时必须额外确认：

- `system:setting:list` 是否已授予对应角色；
- `GET /api/v1/system/setting/list` 是否已授予对应 Casbin 策略；
- `GET /api/v1/system/menu/tree` 是否已授予对应 Casbin 策略；
- 若角色需要保存，是否同时具备：
  - `system:setting:update`
  - `PUT /api/v1/system/setting/group/:groupKey`
- 若角色需要手动刷新缓存，是否同时具备：
  - `system:setting:refresh`
  - `POST /api/v1/system/setting/cache/refresh`
- 若角色不具备统一审计查看能力，系统设置页是否仍能正常加载主体，而不是因为审计区块失败直接展示错误态。

## 19. 当前落地差距

当前已有：

- 按钮细粒度权限
- Casbin 策略页面
- 动态菜单裁剪
- 页面级权限守卫
- 统一 403 页面
- 角色页中的导航、页面、操作分层授权表单
- 权限工作台中的角色维度综合授权视图

当前仍缺：

- 权限命名统一文档落地
- 数据权限、多租户等 L5 能力

## 20. 下一份建议补的文档

下一份建议补：

- `docs/designs/ERROR_CODE_AND_I18N.md`
- `docs/acceptances/ACCEPTANCE_CHECKLIST.md`

因为权限和模块契约定下来后，前后端最容易继续混乱的是：

- 错误码与多语言边界；
- 权限设计完成后如何统一验收。
