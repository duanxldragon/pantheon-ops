# Pantheon Base - 企业级后台总体设计

English version: [DESIGN.en.md](./DESIGN.en.md)

Pantheon Base 的目标不是再做一个“只有登录 + 菜单 + CRUD”的后台壳，而是沉淀一个**多语言、通用、业务与底座解耦、支持动态菜单与标准权限模型的企业级管理平台底座**。

项目核心原则是：

- **底座稳定**：认证、授权、组织、配置、审计、多语言等公共能力长期稳定演进。
- **业务解耦**：业务模块只能依赖公共契约，不能耦合到底座实现细节。
- **通用扩展**：所有页面、菜单、权限、字典、配置都以“可配置 + 可注册 + 可审计”为优先。
- **AI 友好**：目录、命名、边界、文档和约束足够明确，AI 可以稳定理解项目结构并生成一致代码。
- **审美克制**：前端借鉴 Figma 式工具感和 `DESIGN.md` 设计系统方法，避免通用 AI 生成 UI 的模板感。

## 1. 产品定位

Pantheon Base 应定位为“**模块化单体 + 企业后台底座**”，适用于：

- 多业务线共享一套系统管理与基础平台能力；
- 多语言交付；
- 未来支持 SaaS / 多租户 / 开放平台时仍可平滑演进；
- 前后端模块都可通过注册机制接入，而不是把业务写死在底座里。

## 2. 分层与模块边界

### 2.1 建议的逻辑分层

| 层级 | 说明 | 典型目录 |
| :--- | :--- | :--- |
| **平台壳层** | 启动、路由装配、中间件、公共库、基础设施 | `backend/cmd`、`backend/internal`、`backend/pkg`、`frontend/src/core` |
| **系统底座层** | 后台公共域能力，不承载具体业务流程 | `backend/modules/system/*`、`frontend/src/modules/system/*`，其中 `auth` 与 `dashboard` 可按独立顶层模块物理放置 |
| **业务领域层** | 订单、商品、客户、工单等业务域 | `backend/modules/business/*`、`frontend/src/modules/business/*` |

### 2.2 系统底座建议拆分的能力域

系统底座不应继续被理解为一个“大 system 杂物间”，建议按能力域理解：

- **auth**：登录、刷新、注销、会话、安全策略、登录日志、MFA/验证码/SSO
- **iam**：用户、角色、权限点、菜单、角色授权、按钮权限、资源权限
- **org**：部门、岗位、组织树、用户组织归属
- **i18n**：语言包、翻译资源管理、缺失 key 检测
- **audit**：操作日志、登录日志、安全审计
- **dict**：字典类型、字典项、枚举下发
- **setting**：系统设置、租户级配置、UI 偏好、平台参数

这几个域都属于“底座公共能力”，但在代码与职责上必须边界清晰。

### 2.3 业务与底座解耦原则

- 业务模块**禁止**直接依赖底座某个具体 Service/Repository。
- 业务模块只能依赖：
  - `gin.Context` 中注入的身份、租户、请求上下文；
  - `pkg/common`、`pkg/database`、`pkg/contracts` 这类公共契约；
  - 明确定义的跨模块接口，而不是直接 import 某个底座实现。
- 菜单、权限、字典、配置等底座能力应通过“注册 + 配置”接入业务，而不是在 system 模块里手写业务分支。

## 3. 标准企业级后台能力清单

### 3.1 P0 必须闭环

- 认证：登录、刷新、注销、密码修改、会话吊销
- 授权：角色、菜单、权限点、接口权限、按钮权限
- 组织：用户、部门、岗位
- 导航：动态菜单、面包屑、个人中心入口
- 多语言：页面文案、菜单标题、表单提示、错误消息
- 审计：操作日志、登录日志、关键安全事件

### 3.2 P1 应补齐

- 会话管理：登录设备、在线会话、强制下线
- 权限管理台：权限点 CRUD、策略可视化、按钮/资源授权
- 字典管理：字典类型、字典项、状态、排序、缓存刷新
- 系统设置：站点配置、安全策略、上传配置、通知配置

### 3.3 P2 演进能力

- 多租户
- SSO / OAuth2 / OIDC
- MFA / 登录风控
- 数据权限
- 主题与品牌配置
- 可视化工作台

## 4. 认证与授权设计

### 4.1 `auth` 是否需要从 `system` 拆分？

**结论：需要从当前 `system/user` 混合实现中拆出来，但现阶段不建议拆成独立服务。**

推荐策略：

1. **立即做逻辑拆分**：把登录、刷新、注销、令牌、会话、登录日志从 `user` 子域中拆出，形成独立 `auth` 子域。
2. **模块内保留单体部署**：短期保持模块化单体部署；`auth` 可以物理独立到 `modules/auth`，但不必为了“拆分”而提前微服务化。
3. **保留系统域协同**：`auth` 与 `iam/org` 同属底座安全域，但职责分离。

### 4.2 推荐边界

| 子域 | 职责 |
| :--- | :--- |
| **auth** | 登录、刷新、注销、密码校验、会话、token、登录日志、安全策略 |
| **user** | 用户档案、用户列表、用户编辑、用户组织归属 |
| **role** | 角色 CRUD、角色授权 |
| **permission** | 权限点、按钮权限、资源权限 |
| **menu** | 菜单树、导航信息架构、菜单元数据 |

### 4.3 为什么要拆

- 当前 `backend/modules/system/user/user_service.go` 同时承担认证、会话、用户资料和后台用户管理，职责过重。
- 用户管理和认证安全不是同一种业务：
  - “管理别人”属于 IAM；
  - “验证我是谁 / 我能否登录”属于 Auth。
- 后续一旦增加验证码、MFA、单点登录、设备管理、登录风控，不拆会继续膨胀。

### 4.4 权限模型必须补强

当前系统已经有 Casbin 和菜单授权，但离企业级后台还差一个关键升级：

- **菜单 ≠ 权限点**
- **列表权限 ≠ 写权限**

建议目标模型：

- `menu`：导航结构、图标、路由、排序、显隐、缓存策略
- `permission_point`：按钮/接口/资源动作，例如 `system:user:create`
- `role_menu`：控制能看到什么导航
- `role_permission`：控制能执行什么动作

前端按钮权限不应再使用 `system:user:list` 同时兜底新增/编辑/删除。

## 5. 动态菜单设计

动态菜单要满足“通用”和“业务解耦”，建议明确以下约束：

- 菜单只承载导航元数据，不承载具体业务逻辑。
- 菜单标题统一存 `title_key`，由 i18n 解析，不存多语言文案本体。
- 菜单应至少包含：
  - `path`
  - `name`
  - `component`
  - `icon`
  - `type`
  - `sort`
  - `is_visible`
  - `is_cache`
  - `is_external`
  - `active_menu`
  - `perms`（如保留）
  - `module`（归属 system / business）
- `scope=nav` 只返回当前用户可见导航树；
- `scope=manage` 返回完整可管理树；
- 业务模块菜单优先通过模块注册 + 后端菜单数据挂接，不允许在 Layout 里写死。

## 6. 多语言设计

企业级后台的多语言不能只停留在“页面能切中英文”，还要满足可维护性：

- 所有展示文案必须使用 `t()` 或等价翻译函数；
- 菜单、按钮、表单、错误消息、空状态、确认弹窗都必须国际化；
- 后端错误码返回 key，不直接返回自然语言；
- i18n key 命名建议按域分层：
  - `common.*`
  - `auth.*`
  - `system.user.*`
  - `system.role.*`
  - `biz.order.*`
- 前端保留最小 fallback，后端语言包作为运行时增量覆盖；
- 必须补充“缺失翻译 key 检查规则”，防止新增页面漏翻译。

## 7. UI / UX 设计建议

### 7.1 信息架构

标准企业级后台建议形成如下一级导航：

- 工作台
- 访问控制
  - 用户
  - 角色
  - 权限
  - 菜单
- 组织架构
  - 部门
  - 岗位
- 平台配置
  - 字典
  - 系统设置
  - 多语言资源（可选）
- 安全审计
  - 登录日志
  - 会话管理
  - 操作日志

个人中心与当前用户安全中心建议放在右上角用户菜单，而不是左侧主导航。

### 7.2 页面交互规范

所有列表页应统一具备：

- 筛选区
- 表格区
- 分页
- 批量操作（如适用）
- 空状态
- 加载状态
- 无权限状态
- 删除确认
- 提交中状态

### 7.3 视觉一致性建议

- 延续 Arco Design，但需要把 icon、间距、弹窗宽度、表格筛选布局、表单栅格形成统一规范；
- 菜单图标不要继续用自由字符串输入，建议补“图标枚举/选择器”；
- Profile、安全中心、会话管理页应从“普通 CRUD 页”中独立出更强的账号安全表达。

### 7.4 响应式与导航态

- 折叠菜单、移动端侧栏、超长面包屑、长菜单滚动、高层级菜单收起策略都需要补设计；
- 页面标题、标签页、路由缓存策略建议形成统一规范。

### 7.5 后台 UI 专项整改方向

后台 UI 整改归属 `platform` 与 `system/*` 底座层，重点是登录页、应用壳层、平台工作台和系统管理页的一致性，不涉及 `business/*` 页面设计。

目标视觉语言统一为：**冷静、可信、工具化的企业后台**。

必须遵守：

- 登录页与系统内部共享同一套主题 token，不再做割裂的营销式 hero；
- 应用壳层使用中性 surface、弱边框、稳定导航态，减少大面积渐变、glass blur 和装饰光晕；
- 平台工作台只做跨系统域摘要与待关注事项，不硬编码业务模块卡片；
- 系统域页面统一使用页面骨架、Arco 组件、标准状态与 i18n key；
- 未实现真实行为的控件不展示为可点击能力，例如“忘记密码”“记住我”“通知”等；
### 7.6 字体体系

企业级后台字体选择原则：中性、可读优先、支持表格数字等宽。

| 角色 | 字体 | 说明 |
| :--- | :--- | :--- |
| **UI 正文 / 全局** | Source Sans 3 | Adobe 开源 UI 字体，支持 tabular-nums，与 CJK 搭配成熟 |
| **代码 / 数据** | JetBrains Mono | 等宽，连字支持，运维日志与代码编辑器场景 |
| **回退** | system-ui, -apple-system, 'Segoe UI', sans-serif | |

Source Sans 3 通过 Google Fonts 加载，`index.html` 中引入。

字重使用标准值：400（正文）、500（标签/次级文本）、600（标题/强调值）、700（Hero 大标题）。禁止使用非标准权重如 650/620。

### 7.7 色彩体系

**四主题制**：indigo（默认）、emerald、violet、slate。每个主题定义完整的品牌色阶和语义色。

**核心 Token 语义：**

| Token | 用途 |
| :--- | :--- |
| `--brand-primary` | 主按钮、选中态、链接、活跃指示器 |
| `--text-primary` | 标题、正文 |
| `--text-secondary` | 描述文本、辅助信息 |
| `--text-tertiary` | 占位符、禁用态、水印文本 |
| `--panel-bg-solid` | 卡片、弹窗、表单容器背景 |
| `--panel-border` | 卡片边框、分割线、输入框边框 |
| `--panel-muted` | 次级面板背景（如偶行条纹、嵌套卡片） |
| `--app-bg` | 页面最外层底色 |
| `--radius-*` | 统一圆角尺度 |

**Arco 变量禁用：** 全局禁止直接使用 Arco 原始 token（`--color-text-1`、`--color-border-2`、`--color-fill-1` 等）。所有颜色引用统一使用 Pantheon token。

### 7.8 间距与圆角

**间距基准：** 4px 基础单位。

| 级别 | 值 | 典型用途 |
| :--- | :--- | :--- |
| xs | 4px | 紧密图标间距、表内标签间距 |
| sm | 8px | 表单栅格间距、同类元素间距 |
| md | 16px | 卡片内边距、区段间距 |
| lg | 24px | 页面内容区 padding |
| xl | 32px | 大区块间隔 |

**圆角体系：** 工具型后台使用克制圆角。

| Token | 值 | 适用 |
| :--- | :--- | :--- |
| `--radius-xs` | 4px | 标签、徽章、小开关 |
| `--radius-sm` | 6px | 按钮、输入框、选择器、分页 |
| `--radius-md` | 8px | 卡片、面板、弹窗、抽屉 |
| `--radius-lg` | 12px | 大模态框 |
| `--radius-pill` | 999px | 圆角标签、状态指示点 |

### 7.9 视觉反模式（禁止清单）

以下模式已被清除，不得重新引入：

- `radial-gradient` 光晕装饰（Hero 卡片、仪表盘背景）
- `linear-gradient` 大面积渐变（侧栏、内容区、卡片表面）
- `::before` 伪元素网格背景叠加（登录页）
- 主按钮彩色大阴影（`0 8px 18px` 级投影）
- 非标准 `font-weight` 值（650、620 等）
- 营销式登录页 Assurance 区块（三点安全承诺列表）
- Inter 作为主字体（已替换为 Source Sans 3）

所有视觉表面使用纯色 Token。层次通过边框、弱阴影、间距来建立。

## 8. 当前实现状态判断

结合现有代码，当前底座已经具备以下基础闭环：

- 登录 / refresh / logout
- 用户、角色、部门、岗位、菜单、权限策略
- 动态菜单
- 多语言语言包拉取
- 个人中心资料维护
- 安全中心 / 当前用户会话管理 / 最近登录日志
- MFA / TOTP 与高敏操作二次验证
- 平台首页真实统计卡与最近登录活动概览
- Casbin 鉴权

但距离“标准企业级后台”仍有明显缺口：

- 后台 UI 的主要短板已经从“风格不统一”收敛到“持续验收和基线防回归”；
- 字典管理已完成基础闭环与缓存刷新，业务字典接入样例与引用保护已有指南，后续重点是按业务模块逐步落地；
- 系统设置已完成上传配置分组、敏感配置加密与配置变更审计基础能力，后续重点是安全策略消费和缓存策略制度化；
- 菜单元数据基础能力已补齐，IA、外链、iframe、页签缓存和导航高亮已有深化设计；
- MFA 已实现，来源级节流、安全事件、历史密码复用限制和密码过期提醒已补齐；后续安全策略缺口集中在新设备/异地识别、SSO/OIDC 和更细风控；
- UI 状态规范虽已起步，但还未全站制度化

## 9. 近期建议路线图

### P0

1. 已完成：`auth` 从 `user` 中逻辑拆分，并落地 MFA / TOTP。
2. 已完成：后台 UI 专项整改进入基线防回归阶段。
3. 已完成：补齐 `system/config` 高敏页面验收基线。
4. 已完成：补齐 `business/cmdb` 设计锚点和 `business/*` 业务模块验收矩阵。

### P1

5. 已完成设计锚点：`system/org` 独立组织域设计。
6. 已完成设计锚点：权限工作台治理深化，从“发现 + 导出 + 受控补齐”推进到“整改追踪”。
7. 已完成设计锚点：业务字典接入、菜单 IA 深化、安全策略路线图。

### P2

8. 已完成：数据权限中间件、角色数据范围策略、`dept_and_children` 部门树展开和 CMDB Host 数据范围回归样板。
9. 已完成设计锚点：SSO / OAuth2 / OIDC 与登录风控后续边界。
10. 后续演进：更多业务模块 smoke 覆盖有权限/无权限数据集；真实多租户、SSO provider 和登录风控自动化继续按边界预留，不在身份源或租户模型未明确前提前实现。

## 10. 文档使用顺序

开始任何设计或开发前，建议按以下顺序理解项目：

1. `DESIGN.md`
2. `AGENTS.md` / `agent.md`
3. `docs/README.md`
4. `docs/contracts/DOCUMENT_GOVERNANCE_CONTRACT.md`
5. `docs/contracts/DOCUMENT_METADATA_AND_STATUS.md`
6. `docs/contracts/PLATFORM_CONTRACT.md`
7. `docs/contracts/SYSTEM_AUTH_CONTRACT.md`
8. `docs/contracts/SYSTEM_IAM_CONTRACT.md`
9. `docs/contracts/SYSTEM_ORG_CONTRACT.md`
10. `docs/contracts/SYSTEM_CONFIG_CONTRACT.md`
11. `docs/designs/BACKEND.md`
12. `docs/designs/FRONTEND.md`
13. `docs/designs/FRONTEND_UI_SPEC.md`
14. `docs/designs/PLATFORM_DASHBOARD_DESIGN.md`
15. `docs/designs/AUTH_MODULE_DESIGN.md`
16. `docs/designs/MODULE_CONTRACT.md`
17. `docs/designs/BUSINESS_MODULE_TEMPLATE.md`
18. `docs/designs/PERMISSION_MODEL.md`
19. `docs/designs/ERROR_CODE_AND_I18N.md`
20. `docs/designs/FRONTEND_PAGE_TEMPLATES.md`
21. `docs/designs/FRONTEND_COMPONENT_PLAN.md`
22. `docs/designs/SECURITY_CENTER_DESIGN.md`
23. `docs/designs/DICT_AND_SETTING_DESIGN.md`
24. `docs/designs/TENANT_READY_SINGLE_TENANT_DESIGN.md`
25. `docs/designs/BUSINESS_MODELING_REVIEW_CHECKLIST.md`
26. `docs/designs/LOWCODE_GENERATOR_GUIDE.md`
27. `docs/designs/DYNAMIC_MODULE_GOVERNANCE_DESIGN.md`
28. `docs/designs/GENERATOR_MODULE_DESIGN.md`
29. `docs/designs/I18N_MODULE_DESIGN.md`
30. `docs/designs/UPLOAD_AND_STORAGE_DESIGN.md`
31. `docs/designs/SYSTEM_ORG_DESIGN.md`
32. `docs/designs/BUSINESS_CMDB_MODULE_DESIGN.md`
33. `docs/designs/BUSINESS_DICT_INTEGRATION_GUIDE.md`
34. `docs/designs/NAVIGATION_IA_STRATEGY.md`
35. `docs/designs/PERMISSION_WORKBENCH_GOVERNANCE_DESIGN.md`
36. `docs/designs/SECURITY_POLICY_ROADMAP.md`
37. `docs/designs/SSO_OIDC_DESIGN.md`
38. `docs/designs/P2_SCALE_ROADMAP.md`
39. `docs/designs/DATABASE.md`
40. `docs/designs/WORKFLOW.md`
41. `docs/acceptances/ACCEPTANCE_CHECKLIST.md`
42. `docs/acceptances/SYSTEM_CONFIG_GOVERNANCE_ACCEPTANCE.md`
43. `docs/acceptances/BUSINESS_MODULE_ACCEPTANCE_MATRIX.md`
44. `docs/archive/IMPLEMENTATION_ROADMAP.md`
45. `docs/assessments/SYSTEM_MODULE_AUDIT.md`
