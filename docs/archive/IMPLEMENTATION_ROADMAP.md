# 实施路线图

更新时间：2026-04-17

类型：Design
归属层：platform
状态：Superseded

本文把当前已经沉淀好的设计文档转成”可执行路线图”，用于后续按阶段推进代码实现。

目标是：

- 先把底座边界做对
- 再把 UI 基建做稳
- 再补系统能力闭环
- 最后再接业务模块

这份路线图默认遵循一个原则：

> **宁可前期设计重一些，也不要后期为了兼容历史包袱把系统拖成屎山。**

## 1. 当前阶段判断

Pantheon Base 目前处于：

- **项目早期**
- **允许大规模重构**
- **应优先文档和边界设计**
- **不应急于快速堆页面和功能**

因此当前的最优策略不是“把缺的页面先补齐”，而是：

1. 先固定模块边界
2. 先固定 UI 设计语言
3. 先固定模块注册与权限模型
4. 再进入代码实现

## 2. 路线图总览

| 阶段 | 目标 | 输出物 | 状态 |
| :--- | :--- | :--- | :--- |
| `P0-Design` | 补齐顶层设计与实施文档 | `DESIGN.md`、`AUTH_MODULE_DESIGN.md`、`FRONTEND_UI_SPEC.md`、本路线图 | 进行中 |
| `P0-Foundation` | 先做架构基建重构 | `auth` 模块边界、模块注册约定、页面骨架、统一状态组件 | 进行中 |
| `P0-System` | 完成企业后台底座闭环 | 安全中心、会话管理、权限工作台、403/500/空态体系 | 进行中 |
| `P1-Config` | 完成平台配置层 | 字典管理、系统设置、菜单元数据增强 | 进行中 |
| `P1-BusinessReady` | 让业务模块可落地 | 业务模块样例、模块脚手架、注册约定、文档模板 | 进行中 |
| `P2-Scale` | 面向中长期演进 | 数据权限、租户、SSO/OIDC、登录风控；MFA 已实现 | 设计锚点已补，待分项实现 |

## 3. 实施原则

### 3.1 先顺序，后速度

后续工作必须按以下依赖顺序推进：

1. 模块边界
2. 权限模型
3. 前端页面骨架
4. 安全与配置能力
5. 业务模块示例

### 3.2 先抽象公共骨架，再补具体页面

禁止后续出现：

- 每个列表页都各写一套布局
- 每个弹窗都各写一套 footer
- 每个页面都重新发明 loading / empty / error
- 权限校验分散在不同写法里

### 3.3 先做逻辑拆分，再做路径收口

例如 `auth`：

- 先把代码职责从 `user` 中抽出来
- 再收口接口路径
- 最后再清理兼容层

不要反过来。

### 3.4 先做文档闭环，再做开发排期

每进入一个新阶段前，必须确认：

- 目标是否写进文档
- 模块边界是否写清
- API 是否定义
- UI 是否有骨架规范
- 验收方式是否明确

## 4. P0-Design：文档设计阶段

### 4.1 目标

把后续开发必须依赖的文档补齐。

### 4.2 已完成

- 顶层设计：`DESIGN.md`
- Agent 规范：`AGENTS.md`、`agent.md`
- Auth 拆分设计：`docs/designs/AUTH_MODULE_DESIGN.md`
- 前端 UI 细则：`docs/designs/FRONTEND_UI_SPEC.md`
- 后台 UI 专项整改方案：`docs/remediations/BACKOFFICE_UI_REMEDIATION_PLAN_20260423.md`
- 平台仪表盘设计：`docs/designs/PLATFORM_DASHBOARD_DESIGN.md`
- 模块契约：`docs/designs/MODULE_CONTRACT.md`
- 权限模型：`docs/designs/PERMISSION_MODEL.md`
- 错误码与多语言：`docs/designs/ERROR_CODE_AND_I18N.md`
- 前端页面模板：`docs/designs/FRONTEND_PAGE_TEMPLATES.md`
- 前端组件规划：`docs/designs/FRONTEND_COMPONENT_PLAN.md`
- 安全中心设计：`docs/designs/SECURITY_CENTER_DESIGN.md`
- 字典与系统设置：`docs/designs/DICT_AND_SETTING_DESIGN.md`
- 业务模块设计模板：`docs/designs/BUSINESS_MODULE_TEMPLATE.md`
- 设计与实现验收清单：`docs/acceptances/ACCEPTANCE_CHECKLIST.md`

### 4.3 还建议补

- `docs/business/ORDER_MODULE_DESIGN.md`（业务样例）
- `docs/FRONTEND_COMPONENT_CONTRACT.md`（如后续组件体系继续扩大）
- `docs/acceptance/business-module.md`（如后续验收清单拆分）

### 4.4 完成定义

当以下问题都能在文档里找到答案，说明 `P0-Design` 可以结束：

- auth 和 user 的边界是什么？
- 页面骨架如何统一？
- 公共前端组件先沉淀哪些？
- 权限如何分导航、页面、按钮、接口？
- 新增模块怎么注册？
- 新增业务模块设计文档怎么写？
- 错误信息如何国际化？
- 一个阶段完成后如何验收？

## 5. P0-Foundation：架构基建阶段

这是最关键的一阶段。

### 5.1 目标

把未来最容易演变成屎山的部分先重构好。

### 5.2 工作包

#### A. `auth` 模块拆分

- 从 `system/user` 中抽出：
  - login
  - refresh
  - logout
  - session
  - password
  - login log
- 新建 `backend/modules/auth/`
- 前端 `Login.tsx` 迁到 `frontend/src/modules/auth/`

#### B. 模块注册契约

后端统一约定模块装配能力：

- `RegisterRoutes`
- `SeedMenus`
- `SeedPerms`
- `Migrate`（如果采用）

前端统一约定模块 manifest：

- routes
- menu metadata
- permission keys
- i18n namespace
- feature flags（预留）

当前落地状态：

- 后端已新增 `pkg/contracts.BackendModule` 与 `RegisterBackendModules`
- 前端已升级 `ModuleConfig`，系统模块已开始声明 `scope / menus / permissions / i18nNamespaces / pagePermission`
- 页面级权限拦截已接入 `RoutePermissionGuard`

#### C. 前端页面骨架基建

先封装统一页面结构：

- `PageHeader`
- `PageActionBar`
- `FilterPanel`
- `DataTableState`
- `PageEmpty`
- `PageError`
- `PageForbidden`

当前落地状态：

- 第一批已完成：`PageContainer / PageHeader / FilterPanel / PageLoading / PageEmpty / PageError / PageForbidden`
- 第二批已完成首轮：`AppTable / PageActions / FormSection / SubmitBar`
- 已接入页面：`auth/*`、`system/user`、`system/role`、`system/permission`、`system/profile`、`system/dept`、`system/menu`、`system/post`
- 异常态起步：已补全局 `404` 页面兜底
- 细分异常态：已补 `PageServerError / PageNetworkError`，dashboard 已先行接入

#### D. 统一状态体系

建立统一的：

- loading
- empty
- error
- forbidden
- submitting

当前新增进展：

- 请求层已能区分 `network / timeout / server / business`
- dashboard 已作为首个页面接入细分异常态

#### E. 后台 UI 专项整改

归属 `platform` 与 `system/*` 底座层，不涉及 `business/*` 页面设计。

P0 必须完成：

- 登录页从营销式 hero 收敛为专业认证控制台；
- 应用壳层统一 side nav、top bar、tabs、content surface 和全局状态页；
- 平台工作台去卡片墙，按 `StatusStrip / PrimaryActions / AttentionPanel / DomainOverview / RecentActivity` 重排；
- 系统域列表页、树表页、配置页统一使用页面骨架与 Arco 组件；
- 移除或禁用未实现真实行为的装饰性控件；
- 登录页到系统内部共享同一套 theme token。

专项设计依据：

- `docs/remediations/BACKOFFICE_UI_REMEDIATION_PLAN_20260423.md`
- `docs/designs/FRONTEND_UI_SPEC.md`
- `docs/designs/PLATFORM_DASHBOARD_DESIGN.md`

### 5.3 依赖关系

- `auth` 拆分要先于安全中心页面
- 页面骨架要先于系统设置/字典页面
- theme token 与应用壳层整改要先于批量系统页视觉收敛
- 模块注册契约要先于业务模块示例

### 5.4 完成定义

满足以下条件才算完成：

- `user` 不再承载核心认证逻辑
- 前端至少有一套统一页面骨架被系统页复用
- 模块注册规则形成稳定文档和代码约束
- 登录页、应用壳层、工作台和系统页视觉语言统一，不再呈现明显 AI 模板感

## 6. P0-System：系统底座闭环阶段

### 6.1 目标

把“企业后台底座”真正补成可运营的系统管理台。

### 6.2 工作包

#### A. 安全中心

- 会话列表
- 当前设备标识
- 强制下线
- 密码修改
- 安全提示

#### B. 权限工作台

当前权限能力已存在，但还不够完整。

需要补：

- 角色 -> 菜单 -> 接口/按钮 的完整认知链路
- 更稳定的权限命名规范
- 页面级权限拦截
- 403 页面

#### C. 错误与异常态体系

- 403
- 404
- 500
- 网络错误
- 搜索空态
- 首次使用空态

#### D. 仪表盘升级

- 真实统计卡
- 快捷入口
- 最近操作
- 告警摘要

当前新增进展：

- 平台层已新增 `/api/v1/platform/dashboard/summary`
- 首页 dashboard 已升级为真实统计卡 + 最近登录活动概览

### 6.3 完成定义

满足以下条件：

- 后台不再只是 CRUD 拼装
- 权限态、异常态、空态有统一体验
- 安全中心成为独立能力，不再混在 profile

## 7. P1-Config：平台配置阶段

### 7.1 目标

把系统设置和字典补齐，形成真正可复用的配置底座。

### 7.2 工作包

#### A. 字典管理

- 字典类型
- 字典项
- 排序
- 状态
- 缓存刷新

#### B. 系统设置

- 基础站点信息
- 安全策略
- 上传配置
- 登录策略
- 国际化默认设置

当前新增进展：

- 后端已新增 `system/dict` 垂直切片，包含 `system_dict_type / system_dict_item` 模型与默认字典种子
- 已新增字典类型 CRUD、字典项 CRUD 与公共 `GET /api/v1/system/dict/options` 接口
- 已补字典 options 进程内缓存、自动失效机制与手动刷新入口
- 前端已新增字典管理页，按左侧类型、右侧字典项的主从布局维护通用枚举
- 后端已新增 `system/setting` 垂直切片
- 后端模块契约已补 `Migrate` 钩子，用于模块自有表结构初始化
- 已新增 `system_setting` 表、默认配置种子、公开配置读取接口
- 前端已新增系统设置页，支持按分组编辑基础信息、安全策略、登录策略、上传配置、国际化和 UI 偏好
- 敏感配置已补加密存储与管理端脱敏表达，上传配置分组已落地
- 已补配置变更审计详情：复用 `system_log_oper` 记录配置变更，并在设置页展示最近审计明细
- 已补系统设置进程内缓存、自动失效与手动刷新入口
- 当前 `system/config` 已完成 setting + dict 基础闭环；剩余重点收敛到业务字典接入样例

#### C. 菜单元数据增强

- 已完成图标选择器与图标映射收口
- 已完成组件路径、`routeName`、外链 URL 的基础校验
- 已完成 `routeName / module / isCache / isExternal / activeMenu` 元数据落库与前端维护
- 已完成 `pagePerm / perms` 元数据拆分，角色授权关系已拆为导航菜单与页面/动作权限两条链路
- 后续仅保留更深一层的 iframe、标签页缓存策略和更完整 IA 规划

### 7.3 完成定义

满足以下条件：

- 业务模块不再依赖硬编码枚举
- 平台配置不再散落在代码和环境变量中
- 字典与系统设置具备最小可运营后台能力，剩余重点为业务接入样例

## 8. P1-BusinessReady：业务接入就绪阶段

### 8.1 目标

让 `business/*` 不是一个空概念，而是真能按统一契约稳定接入。

### 8.2 当前判断

当前 `P1-BusinessReady` 已进入进行中状态，原因是：

- 第一个业务域样例 `business/cmdb` 已完成基础闭环；
- 动态菜单能力已验证可承载真实业务模块；
- 但页面组件装配仍停留在前端 manifest 显式注册阶段，尚未收口到平台级组件注册表。

### 8.3 当前产出

- `docs/designs/BUSINESS_CMDB_MODULE_DESIGN.md` 已形成 CMDB 业务域设计基线；
- `backend/modules/business/cmdb/` 已完成模块装配、菜单/权限 seed、导入导出、详情关系能力；
- `frontend/src/modules/business/cmdb/` 已完成类型页、实例页、详情页接入；
- `docs/assessments/DYNAMIC_MENU_MATURITY_20260422.md` 已明确动态菜单成熟度结论与下一阶段蓝图。

### 8.4 工作包

#### A. 业务模块样例

当前已选择轻量级 `business/cmdb` 作为第一个样例，重点验证：

- 业务模块后端装配
- 业务模块前端 manifest 注册
- 菜单、页面权限、按钮权限 seed
- 业务表 `biz_` 前缀与垂直切片
- 字典、审计、系统组织字段的弱依赖接入
- 平台层仪表盘聚合业务域摘要
- CSV 导入导出与模板下载能力
- 资源详情页与轻量关系视图

设计文档见 `docs/designs/BUSINESS_CMDB_MODULE_DESIGN.md`。

#### B. 模块脚手架

补标准目录模板：

- backend module template
- frontend module template
- DDL template
- i18n key template
- seed template

#### C. 文档模板

新增业务模块时必须同步：

- 模块设计说明
- API 清单
- 权限点清单
- 菜单配置
- i18n key 清单

#### D. 动态菜单下一阶段

- 建立平台级 `component registry`
- 为 `business/*` 模块增加 manifest 与菜单 seed 一致性检查
- 以 CMDB 为首个样板，验证 `component key -> 页面组件` 受控解析
- 继续补齐业务模块脚手架与验收模板

当前新增进展：

- `frontend/src/core/router/componentRegistry.ts` 已成为平台级受控组件注册表；
- `system/*`、`auth`、`platform/dashboard` 与 `business/cmdb` 已逐步接入 `component key`；
- 已新增 `frontend/scripts/check-menu-contract.mjs`，用于检查 frontend manifest 与 backend menu seed 的一致性；
- 当前可通过 `cd frontend && npm run check:menu-contract` 做发布前契约校验。
- 后端菜单保存已补 `component key` 白名单校验，避免绕过前端直接写入未注册组件；
- `frontend` 已通过 `prebuild` 强制在构建前执行菜单契约检查；
- `.github/workflows/quality.yml` 已加入 PR / main 推送质量门禁。
- 契约检查已增强为模块能力声明校验，覆盖重复 path、重复 routeName、重复权限、模块边界、i18nNamespaces、页面权限声明和后端 seed 权限声明。

### 8.5 完成定义

满足以下条件：

- 至少一个真实 `business` 模块跑通
- 没有直接耦合到底座内部实现
- 模块注册和权限接入全流程被验证
- 新增 `business/*` 页面无需再修改全局模块装配清单
- 后端菜单 `component` 已收口为受控注册表 key

## 9. P2-Scale：中长期能力阶段

这部分不是现在马上做，但要在设计上留边界。

### 9.1 包含能力

- 数据权限
- 多租户
- SSO / OAuth2 / OIDC
- MFA（已实现，后续只保留回归与风控联动）
- 登录风控
- 安全事件中心

### 9.2 原则

- 现在不实现
- 现在必须留边界

## 10. 推荐执行顺序

接下来建议按下面的次序逐步推进：

1. `docs/designs/MODULE_CONTRACT.md`
2. `docs/designs/PERMISSION_MODEL.md`
3. `docs/designs/ERROR_CODE_AND_I18N.md`
4. `docs/designs/FRONTEND_COMPONENT_PLAN.md`
5. `docs/designs/BUSINESS_MODULE_TEMPLATE.md`
6. `docs/acceptances/ACCEPTANCE_CHECKLIST.md`
7. `auth` 后端拆分
8. `auth` 前端拆分
9. 页面骨架组件封装
10. 403/500/空态体系
11. 安全中心 / 会话管理
12. 字典管理
13. 系统设置
14. 业务模块样例

## 11. 每阶段的交付要求

每一阶段结束时，必须同时交付：

- 代码
- 文档更新
- 验收清单
- 后续影响说明

不允许只交代码，不补文档。

## 12. 当前文档完成后的下一批重点

如果继续坚持“文档先行”，建议下一批优先补：

1. `docs/business/ORDER_MODULE_DESIGN.md`
2. `docs/FRONTEND_COMPONENT_CONTRACT.md`
3. `docs/acceptance/business-module.md`

这三份会把“通用规范”推进到“可直接照着落具体模块”。

## 13. 关联文档

- `DESIGN.md`
- `AGENTS.md`
- `docs/designs/AUTH_MODULE_DESIGN.md`
- `docs/designs/FRONTEND.md`
- `docs/designs/FRONTEND_UI_SPEC.md`
- `docs/designs/WORKFLOW.md`
