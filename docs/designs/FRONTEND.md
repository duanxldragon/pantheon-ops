# 前端架构设计与 UI 规范

更新时间：2026-04-17

类型：Design
归属层：platform
状态：Active

> 本文偏”架构总览”。更细的页面骨架、导航、状态、表单、表格、响应式和权限态规范，见 `docs/designs/FRONTEND_UI_SPEC.md`；后台 UI 专项整改见 `docs/remediations/BACKOFFICE_UI_REMEDIATION_PLAN_20260423.md`。

## 1. 架构目标：模块化、声明式、解耦
前端底座是一个“壳”，业务模块通过配置化的形式向壳注册自己。

## 1.1 当前阶段原则

- **先补设计，再补实现**
- **先锁边界，再写页面**
- **先沉淀通用骨架，再做业务模块**
- **平台壳层统一治理左导航、右辅助栏和浮层容器，不再让系统页各自发明版式**

尤其是认证、安全、权限、配置类页面，必须先在文档中明确页面类型、交互状态和模块边界，再进入编码阶段。

## 2. 视觉规范 (The Indigo Identity)
- **色调**: 核心 Indigo Blue (#165DFF)，辅助色 Neutral Gray (#F2F3F5)。
- **风格参考**: 参考 `awesome-design-md` 的 Markdown 设计系统方法，审美方向偏 Figma：界面骨架克制、内容区可有少量多彩点缀、pill/circle 几何、精确 focus 反馈。
- **去 AI 味**: 禁止默认紫蓝渐变、通用三栏卡片、彩色圆形 icon 堆叠、无信息架构的卡片墙。
- **后台整改基线**: 登录页、应用壳层、工作台和系统页必须统一为“冷静、可信、工具化”的企业后台质感，不做营销页式 hero。
- **层级架构 (Layering Strategy)**:
  - **Base (底板)**: `#F7F8FA`，用于全局背景。
  - **Surface (表面)**: `#FFFFFF`，用于卡片、侧边栏。带有极细边框 `1px solid rgba(0,0,0,0.04)`。
  - **Overlay (悬浮)**: 纯白，带有弥散投影，用于下拉菜单、弹窗。
- **排版 (Typography)**:
  - 严格遵循 1.25 倍缩放比例 (12, 14, 18, 24, 32px)。
  - 字重分层：标题 (600), 重点正文 (500), 普通正文 (400)。
- **阴影与深度 (Shadow & Depth)**:
  - **Subtle**: `0 1px 2px rgba(0,0,0,0.05)` (静态卡片)。
  - **Elevated**: `0 12px 32px rgba(22, 93, 255, 0.08)` (悬浮/活动组件)。

补充平台壳层约束：

- 左侧侧边栏只承担导航定向，不承载说明卡、统计卡和帮助文案；
- 右侧辅助栏只承担次级上下文和风险提示，不复制主内容区摘要；
- Modal / Drawer 属于统一浮层系统，不再把页面压缩塞进浮层。

## 3. 核心交互细节 (Awesome Design Details)
- **去线化布局 (Lineless Layout)**: 减少物理分割线，通过 24px/32px 的 **负空间 (Negative Space)** 产生视觉切片。
- **状态表现 (Status Semantic)**:
  - **Success**: 背景 `#E8FFFB`, 文字 `#00B42A` (青色系)。
  - **Warning**: 背景 `#FFF7E8`, 文字 `#FF7D00` (琥珀色系)。
  - **Danger**: 背景 `#FFE8E8`, 文字 `#F53F3F` (红色系)。
- **交互回馈 (Micro-interactions)**:
  - **Buttons**: Hover 时背景加深 10%，Active 时轻微缩小 (98%)。
  - **Inputs**: 聚焦时边框色改为 Indigo，并增加 `2px` 的外发光扩散。
- **动效 (Motion)**: 统一使用短时长 `ease-out` 或 Arco 默认动效，不使用带明显回弹的娱乐化转场。


## 4. 模块解耦注册 (Module Registration)
业务模块存放在 `src/modules/business/`，系统模块存放在 `src/modules/system/`；页面模块通过 `index.ts` 导出 `ModuleConfig`，由 `src/core/router/modules.ts` **显式注册**。

### 4.1 注册配置示例
```typescript
export const OrderModule = {
  name: 'order',
  routes: [
    { path: 'order/list', titleKey: 'biz.order.menu.list', component: React.lazy(() => import('./pages/list')) }
  ]
};
```

> 注意：当前真实类型以 `frontend/src/core/router/types.ts` 为准，`ModuleConfig` 已升级为包含 `scope / menus / permissions / i18nNamespaces / pagePermission` 的模块 manifest。

## 5. 多语言方案 (Dynamic I18n)
- **i18next**: 核心引擎。
- **Backend Sync**: 应用启动时调用 `/api/v1/system/i18n/pack` 接口，拉取数据库中的全量翻译并注入资源池。
- **Fallback Resources**: 前端内置 `zh-CN`、`en-US`、`ja-JP`、`ko-KR`、`fr-FR` 最小语言包；其中新增语言可先回退到英文骨架，再由 `system/config -> i18n` 运行时资产覆盖。
- **模块语言包聚合**: 业务模块和系统模块可以在自身目录维护 `locales/{locale}.json`。平台层通过 `npm run i18n:generate-module` 扫描 `src/modules/**/locales/*.json`，生成 `src/i18n/resources/generated/{locale}.ts`，再由 `src/i18n/index.ts` 与基础 fallback、远端语言包合并。模块开发时只维护本模块 json，禁止再手工双写到全局 fallback。
- **UI 绑定**: 使用 `t('key')` 或 `<Trans />` 组件进行文本翻译。
- **运行时刷新**: `system/config -> i18n` 管理端修改翻译后，前端应支持重新拉取并刷新资源池，不要求用户手工刷新整个页面。
- **请求层兜底**: 请求失败、网络异常、超时等默认提示统一回退到 i18n key，如 `request.failed / network.error / network.timeout`，不再直接暴露硬编码英文 fallback。
- **导入导出本地化**: i18n 导入结果摘要、错误 CSV 列头、错误文件名、冲突阻断提示都应走翻译链路，而不是在前端拼接英文结果文本。
- **语言治理边界**: 团队内部可以继续保留中文注释、中文建模语义和中文治理术语；需要强制国际化的是运行时面向用户的菜单、按钮、页头、提示、弹窗和导入导出反馈。
- **生成器约束**: 代码生成器属于 `system/config` 国际化治理外延，生成的页面动作名、字段模板占位和审计标题应坚持 key-first，而不是把自然语言展示词直接写入源码。
- **扩语种策略**: 前端不预先无限制扩充 locale；只有在出现明确市场需求时，才新增对应 fallback locale，并要求与基准语言包保持完整 key 对齐。

## 6. 组件开发标准 (Arco Design)
- **严禁大量手写 CSS**: 优先使用 Arco Design 的属性（如 `Grid`, `Space`）进行布局。
- **Form 封装**: 统一使用表单校验，并配合后端返回的业务错误码显示 Tip。
- **Fetch 封装**: 统一处理 Token 注入、401/403 异常拦截及 RequestID 日志跟踪。
- **国际化防回归**: 前端构建前必须执行 `npm run i18n:generate-module`、`npm run check:i18n-missing-keys`、`npm run check:i18n-hardcode`。其中生成脚本保证模块语言包进入运行时 fallback，缺失 key 检查扫描静态 `t('...')` 调用，硬编码扫描阻断 `title / label / placeholder / content / defaultValue / Message / Notification` 等展示位写死自然语言。
- **Locale 完整性门禁**: 扩语种、补翻译、导入翻译资产或批量改 key 后，必须执行 `npm run audit:i18n-locales`。该脚本以 `zh-CN` 为基准、`en-US` 为参考，检查各 locale 的 `missing / extra / empty / sameAsEn`，用于阻断缺 key、空值和可疑未翻译值进入仓库。
- **语种扩展验收**: 新增 fallback locale 时，不要求预先支持所有语言，但一旦新增，就必须同步补齐本地资源、通过 `check:i18n-hardcode`、通过 `audit:i18n-locales`，并完成一次 `npm run build` 验证，确保语言切换、动态菜单、导入导出结果与错误反馈都不回退到硬编码。

## 7. 当前页面闭环
- **登录页**: `src/modules/auth/Login.tsx`，已从 `system/user` 迁出，完成 access/refresh token 持久化、用户信息写入与真实 TOTP MFA 登录链路；当 `login.mfa_enabled=true` 且用户未绑定因子时，页面进入现场绑定态，优先展示二维码，并保留手动密钥与 `otpauth://` URI 复制兜底。
- **认证 API**: `src/modules/auth/api.ts`，统一承接 login / refresh / logout / getMe / updatePassword。
- **会话活动与锁屏**: `src/core/layout/index.tsx` 已接入空闲计时、活动上报、锁屏按钮与解锁遮罩；锁屏属于 `platform` 壳层能力，不退出会话、不清空已打开标签，但仍受 `login.session_idle_minutes` 控制。
- **安全中心**: `src/modules/auth/SecurityCenter.tsx`，通过 `/api/v1/auth/security` 承接安全概览，并组合当前用户在线会话管理、登录日志与密码修改。
- **安全审计页**: `src/modules/auth/LoginLogList.tsx`、`src/modules/auth/SessionList.tsx`，已承接管理员登录日志与全局会话管理；登录日志页现已支持按筛选条件导出、按 `system/config -> audit` 设置动态下发的保留期清理，以及按选择集批量删除。
- **请求封装**: `src/api/request.ts` 自动注入 access token；业务码 `401` 时使用 `auth/refresh` 轮换并重放原请求。
- **登录错误反馈**: 登录页和 MFA 验证页对 `/auth/login`、`/auth/mfa/verify` 采用页面级错误处理，优先展示后端返回的认证错误 key 翻译，例如用户名/密码错误、动态码错误、challenge 过期，而不是统一模糊提示。
- **模块 Manifest**: `src/core/router/types.ts` 已将 `ModuleConfig` 升级为包含 `scope / menus / permissions / i18nNamespaces / pagePermission` 的模块契约。
- **权限钩子**: `src/hooks/usePermission.ts` 统一处理 `admin` 角色和权限标识判断。
- **页面权限**: `src/core/router/RoutePermissionGuard.tsx` 根据路由 `pagePermission` 做页面级拦截，并统一展示 403。
- **用户页**: `src/modules/system/user/UserList.tsx`，支持筛选、分页、排序、读取、新增、编辑、删除用户，并维护用户角色绑定；同时支持按部门/岗位筛选、CSV 模板下载、导出、导入摘要反馈与批量启用/禁用。
- **用户详情页**: `src/modules/system/user/UserDetail.tsx` 走独立 `system:user:view` 页面权限，展示用户基础资料、组织归属、角色摘要，并通过详情路由 `activeMenu` 保持菜单高亮。
- **用户密码重置**: `src/modules/system/user/UserList.tsx` 已把管理员重置密码从编辑弹窗中拆出，独立弹窗走 `system:user:reset` 权限点，并提示会话强制下线影响。
- **角色页**: `src/modules/system/role/RoleList.tsx`，支持筛选、分页、排序、读取、新增、编辑、删除角色，并把授权表单拆成“导航授权 / 页面授权 / 操作授权”三段，接口策略仍在权限页独立维护；列表页额外支持角色基础信息导出与批量启用/禁用。
- **菜单页**: `src/modules/system/menu/MenuList.tsx`，支持筛选、排序、读取菜单树、新增、编辑、删除菜单，并把 `pagePerm` 与动作 `perms` 分开维护；页面支持表格、列表、卡片三种浏览方式。
- **部门页**: `src/modules/system/dept/DeptList.tsx`，支持树形读取、新增、编辑、删除部门，并维护上下级组织结构；页面会显示真实组织根节点，普通部门默认挂载在根节点之下，并支持 CSV 模板下载、导出、导入与批量启用/禁用；同时新增“组织架构”视图，以部门树为主干展示岗位和成员归属。
- **组织健康总览**: `src/modules/system/dept/DeptList.tsx` 顶部已补组织治理摘要，直接展示部门总数、岗位总数、缺负责人部门、空部门和问题数，保持在 `system/org` 单域内演进；缺负责人部门、空部门卡片可直接回填筛选。
- **组织治理整改**: 部门页当前筛选结果可直接执行“批量补负责人”，用于对缺负责人部门做集中收口；单条部门仍可沿现有编辑动作进入整改。
- **组织治理闭环增强**: 部门维护表已补治理标签列，区分缺负责人、无岗位、空部门与治理正常；针对无岗位部门，管理员可在表格内直接发起“补岗位”整改动作，并与组织架构中的岗位创建弹窗复用同一链路。
- **治理结果导出增强**: 部门导出按钮继续沿用原入口，但导出的 CSV 现在会附带部门完整路径、子部门数、岗位数、成员数，并统一落到 `governanceScope / governanceTags / governanceProblemCount / governanceBlockedBy / governanceActions` 这套治理模板列；同时追加对应 `*Label` 字典列，方便 `system/org` 在站外做治理分发、整改指派与复盘。
- **治理任务工作台**: `src/modules/system/dept/DeptList.tsx` 已补轻量治理任务区，把部门问题、岗位问题和删除阻断统一收敛为任务清单，并提供“定位整改”入口，直接跳转到负责人补齐、岗位补齐或组织架构定位链路。
- **负责人候选人约束**: 部门编辑弹窗已补“负责人候选人”选择器，默认从当前部门内已启用且已挂岗位的成员中绑定负责人；新建部门仍允许先落组织节点、后补真实负责人，保留原文本负责人字段作为兼容兜底，不强制一次性清洗历史数据。
- **批量负责人治理收口**: “批量补负责人”已升级为任务式治理弹窗。入口仍支持多选部门，但每个部门都需要逐条选择本部门负责人候选人；没有候选人的部门会明确提示先补岗位与成员，避免批量入口继续绕过真实成员约束。
- **岗位页**: `src/modules/system/post/PostList.tsx`，支持分页读取、新增、编辑、删除岗位，并维护岗位所属部门；同时支持 CSV 模板下载、导出、导入与批量启用/禁用。
- **岗位治理导出增强**: 岗位导出按钮继续沿用原入口，但导出的 CSV 现在会附带在用成员数，并统一落到 `governanceScope / governanceTags / governanceProblemCount / governanceBlockedBy / governanceActions` 这套治理模板列；同时追加对应 `*Label` 字典列，方便 `system/org` 对岗位占用治理做线下分发与复核。
- **权限页**: `src/modules/system/permission/PermissionList.tsx`，已升级为“权限工作台 + Casbin 路由策略”双视图，统一展示角色的导航、页面/按钮权限与接口策略；接口策略页支持 CSV 模板下载、导出与导入。
- **个人中心**: `src/modules/system/profile/ProfileCenter.tsx`，支持查看当前账号信息、维护昵称/邮箱/手机号/头像；密码修改请求已切到 `system/auth` API。
- **头像上传**: 个人中心已新增基于统一上传接口的头像上传入口，上传行为受 `system/config` 的上传配置实时约束，不再只能手工输入头像 URL。
- **安全入口**: 顶部用户区和个人中心页均已提供“安全中心”入口，安全能力不再继续堆叠在 `ProfileCenter` 内。
- **审计入口**: 管理员可通过动态菜单进入“登录日志”“会话管理”页面。
- **模块治理兜底**: `src/modules/system/dynamicmodule/ModuleManager.tsx` 已补“注册表自检/修复”动作，用于在历史脏数据、手工删改源码或注册表漂移时重写 generated registries，并把缺失源码的接入记录自动校正为已卸载。
- **基础布局**: `src/core/layout/index.tsx`，支持动态菜单、语言切换、登出，并按当前用户权限渲染侧边导航。
- **页面骨架第一批组件**: 已新增 `src/components/` 下的 `PageContainer`、`PageHeader`、`FilterPanel`、`PageLoading`、`PageEmpty`、`PageError`、`PageForbidden`，并优先接入 `auth` 相关页面。
- **页面骨架第二批组件**: 已新增 `AppTable`、`PageActions`、`FormSection`、`SubmitBar`，并开始接入 `UserList`、`RoleList`、`PermissionList`、`ProfileCenter`。
- **第二批覆盖扩展**: `DeptList`、`MenuList`、`PostList` 已接入统一页面头部、筛选区、表格封装与提交栏。
- **异常态补强**: 已补 `PageServerError`、`PageNetworkError`，请求层也已能区分 `network / timeout / server / business` 基础错误类型。
- **仪表盘真实数据化**: `src/modules/dashboard/` 已接入平台层汇总接口，不再使用硬编码统计数字。
- **首页归属澄清**: dashboard 在模块 manifest 中按 `platform` scope 理解，语义上属于跨域聚合页；物理目录已从 `platform/dashboard` 扁平化到顶层 `dashboard`。
- **系统设置页**: 已新增 `src/modules/system/setting/SettingPage.tsx`，按 `basic/security/login/audit/upload/i18n/ui` 分组维护系统设置，并对敏感配置提供“已加密/留空不变”交互表达。
- **配置健康总览**: `src/modules/system/setting/SettingPage.tsx` 顶部已补配置治理摘要，展示公开/敏感配置数量、缺失必填项、运行时风险以及当前语言、主题、上传驱动状态。
- **平台公开设置消费**: `site.name / site.logo / i18n.default_language / ui.default_theme / ui.enable_tab_bar / login.session_idle_minutes` 已接入登录页与应用壳层；其中默认语言仅在“用户未显式切换语言”时生效，标签栏可由 `ui.enable_tab_bar` 控制显隐，空闲时长由 `login.session_idle_minutes` 控制自动退出。
- **平台能力开关消费**: `platform.app_mode / org.enabled / org.required_for_user` 已进入公开设置链路。`org.enabled=false` 时，壳层会隐藏 `system.org` 导航，用户页隐藏部门/岗位列和表单字段；`org.required_for_user=true` 且组织启用时，用户表单要求选择部门。
- **用户扩展档案契约**: 用户相关 API 类型已预留 `profileExt`，用于 C 端或混合模式下的扩展档案展示与编辑。后台管理页默认不渲染任意 JSON 字段，后续应由具体业务页面或受控表单定义字段语义，避免把未知 PII 直接散落到通用用户列表。
- **平台壳层偏好持久化**: 当前登录用户的 `theme / language / layoutMode / densityMode` 已通过 `GET/PUT /api/v1/auth/me/preferences` 收口到 `platform` 壳层偏好链路；`system/config` 的公开设置继续只负责默认值，不再覆盖用户已经显式保存的壳层选择。登录页语言下拉也视为显式壳层选择，进入系统后不得再被账号历史偏好或默认语言反向覆盖。
- **上传配置消费**: 个人中心与用户管理头像上传都已接入 `/system/upload`，会实时遵守 `upload.max_file_size / upload.allowed_types / upload.public_base_url / upload.s3_*`；本地驱动下返回平台文件 URL，S3 驱动下返回对象访问 URL。
- **设置审计详情**: 系统设置页底部已补最近配置变更审计表，支持查看操作人、操作 IP、变更字段、状态与操作时间，敏感字段只展示“已变更”而不回显明文。
- **设置缓存刷新**: 系统设置页已补“刷新设置缓存”入口，允许管理员按当前分组手动预热缓存。
- **字典管理页**: 已新增 `src/modules/system/dict/DictPage.tsx`，采用左侧字典类型 + 右侧字典项的主从布局，支持类型筛选、字典项排序、状态和颜色维护；类型与字典项都支持各自的 CSV 模板下载、导出与导入。
- **字典缓存刷新**: 字典页右侧卡片已补“刷新缓存”入口，管理员可按当前选中字典手动刷新 options 缓存。
- **菜单元数据页增强**: `src/modules/system/menu/MenuList.tsx` 已支持维护 `routeName / module / isCache / isExternal / activeMenu`，图标输入已收敛为平台层共享 icon registry 的枚举选择器；前后端菜单配置统一复用同一套 `MenuIconKey`，避免自由字符串和默认 fallback 导致重复图标。
- **操作日志页**: `src/modules/system/audit/OperationLogList.tsx` 已支持按筛选条件导出 CSV，并把危险动作收口为“按 `system/config -> audit` 设置动态下发的保留期清理”与“按选择集批量删除”；危险动作继续要求独立权限点和二次验证。

## 8. 交互补充
- **列表筛选**: 用户页支持用户名、昵称、部门、岗位、状态筛选，并与分页、排序联动；角色页支持角色名称、角色标识、状态筛选，并与分页、排序联动；菜单页支持标题键、路径、显示状态筛选，并与排序联动；岗位页支持所属部门、岗位编码、岗位名称、状态筛选，并与分页、排序联动；部门页支持部门名称、状态、治理视角（缺负责人 / 无岗位 / 空部门）筛选，并与树排序联动；字典页支持按 `dictCode / dictName / status` 筛选字典类型，并与右侧字典项主从联动。
- **按钮权限**: 增删改、批量状态更新与敏感动作按钮通过 `usePermission` 按细粒度权限点控制，例如 `system:user:create`、`system:user:reset`、`system:user:batch-update`、`system:dept:batch-update`、`system:role:update`、`system:permission:delete`、`system:dict:update`；`admin` 角色默认拥有全部操作能力。
- **表单校验**: 用户页对密码长度、邮箱格式、角色必选以及部门/岗位选择做前端约束；角色页对角色名称、角色标识必填做前端校验；菜单页对标题键必填做前端校验；部门/岗位页分别对名称、编码等关键字段做必填校验。
- **表格交互**: 用户、角色、部门、岗位在存在批量状态操作时均在表格最左侧展示选择框；用户、角色、岗位表格使用服务端分页与排序，切换页码、每页条数、列排序时统一回写 query 状态。
- **角色授权**: 角色页通过统一树形面板维护 `menuIds` 导航授权、`pagePerm` 页面权限和 `perms` 操作权限，三类授权均支持搜索、全展开/全收起和父级批量勾选，并保留未知历史权限键避免误删授权。
- **树表交互**: 菜单页树表使用服务端排序，列头排序会回写 `sortField/sortOrder` 并保留当前筛选条件。
- **菜单元数据行为**: 菜单导航已支持外链菜单新窗口打开，并支持基于 `activeMenu` 的菜单高亮兜底；图标渲染已统一收口到共享 icon 映射。
- **敏感配置交互**: 系统设置页已识别 `isEncrypted` 元数据；敏感项不回显明文，只显示“已配置/留空不变”提示，并使用密码型输入控件提交新值。
- **设置审计交互**: 切换设置分组时同步刷新该分组最近审计记录；保存成功后自动回到最新第一页，便于管理员确认配置变更已落库。
- **字典缓存交互**: 字典类型/字典项保存后，后端会自动失效对应字典缓存；页面额外提供手动刷新入口，方便联调业务下拉取值。
- **导入导出交互**: 系统域导入统一采用隐藏文件选择 + `ImportCsvButton` 模式，导出统一走 blob 下载；导入结果通过摘要弹窗展示 `created / updated / failed / row errors`，不直接依赖错误 toast 承载结构化详情。
- **危险日志动作交互**: `system/auth` 登录日志与 `system/audit` 操作日志统一采用“保留期清理 + 选择集删除”交互，不再默认暴露全表清空；触发时复用二次验证弹窗并走 `X-Operation-Token` 链路。
- **国际化导入导出交互**: i18n 资源导入时，如果出现重复 key 冲突、目标 key 已存在或与 canonical 记录冲突，前端应以阻断式提示和本地化错误清单反馈，不允许继续“部分成功 + 英文提示兜底”。
- **组织字段接入**: 用户页已支持选择部门和岗位，并在列表中直接展示 `deptName/postName`；用户表单中的部门选项会排除组织根节点，岗位下拉按所选部门过滤，避免用户岗位与部门不一致。
- **组织架构视图**: 部门页中的“组织架构”页签采用 `system/org` 语义，部门节点下直接展示岗位卡片与成员摘要，选中节点后右侧展示直属岗位、直属成员和当前组织规则说明；管理员可在选中部门下直接新增岗位，并从直属成员列表查看用户详情。
- **权限三轨**: 导航授权通过 `system_role_menu`，页面/按钮权限通过 `system_role_permission`，接口访问权限通过 `casbin_rule`；角色保存会同步系统已知权限点对应的 API 策略，权限页仍保留独立治理入口。
- **菜单作用域**: 侧边栏通过 `getMenuTree({ scope: 'nav' })` 只拉取当前用户可见导航；菜单页与角色页通过 `scope: 'manage'` 拉取完整授权树。
- **菜单权限边界**: 菜单元数据中的 `pagePerm` 用于页面进入权限，`perms` 用于按钮/动作权限，前端路由守卫与按钮显隐共享统一权限源但不再复用导航关系。
- **国际化约束**: 布局、按钮、页签等展示文本统一通过 `t()` 输出，避免系统页出现硬编码文案。
- **菜单与壳层联动**: 动态菜单标题、页头标题、导入导出反馈、重命名迁移报告等平台壳层与 `system/config` 交互文案，都必须参与语言切换验证，不能只覆盖普通表单字段。
- **个人入口**: 顶部用户区提供“个人中心”入口；该页面不依赖左侧菜单树，而通过模块路由注册进入。
- **状态基建起步**: 登录日志页、会话管理页、安全中心页已开始统一接入页面级 `empty / error / loading` 组件，作为后续系统页改造基线。
- **表单与列表收口**: 用户、角色、权限、个人中心页面已开始统一使用表格封装、操作区和提交栏，减少系统页继续各写各的风险。
- **基础异常页**: 全局兜底路由已切到 `PageNotFound`，不再直接渲染裸文本 404。
- **细分异常态起步**: dashboard 已作为首个页面接入 `network / timeout / server` 区分展示，为后续系统页收口异常体验打样。

## 9. 缺口与后续文档约束

当前 `FRONTEND.md` 只解决了“架构方向”和“已有能力概览”，还不够支撑后续大规模页面建设。

后续以前端实现为准时，必须同时参考：

- `docs/designs/FRONTEND_UI_SPEC.md`：UI 详细规范
- `docs/remediations/BACKOFFICE_UI_REMEDIATION_PLAN_20260423.md`：后台 UI 专项整改方案
- `docs/designs/AUTH_MODULE_DESIGN.md`：认证与安全中心边界
- `DESIGN.md`：顶层架构与能力域设计

其中 `docs/designs/FRONTEND_UI_SPEC.md` 已进一步把 `platform` 壳层的三类容器写死为统一规则：

- 左侧菜单栏：只做导航和域边界表达；
- 右侧辅助栏：只做上下文和风险提示；
- Modal / Drawer：只做短任务或连续编辑，不做压缩版页面。
