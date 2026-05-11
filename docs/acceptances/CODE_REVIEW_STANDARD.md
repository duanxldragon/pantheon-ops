# Pantheon 代码评审标准流程

更新时间：2026-05-06

类型：Acceptance
归属层：platform
状态：Active

本文定义 Pantheon Base 的固定代码评审流程。它适用于人工评审、AI 评审和阶段性交付前自检。

目标不是做格式化意见，而是防止设计边界、权限、多语言、审计、动态菜单和生成器契约在迭代中漂移。

## 评审层级

每项检查标注所属层级：

- `[Auto]` — 自动化门禁，CI 不可绕过，提交前必须通过
- `[PR]` — PR 提交前必查，reviewer 逐项确认
- `[Phase]` — 阶段深度评审，每个路线图阶段结束后执行

## 1. 评审入口

每次评审开始前必须先声明本次改动归属层：

- `platform`
- `system/auth`
- `system/iam`
- `system/org`
- `system/config`
- `business/*`

如果改动跨层，必须同时说明：

- 主责层是谁
- 依赖层是谁
- 哪些改动是逻辑拆分，哪些改动是物理拆分
- 是否存在业务域反向侵入系统域的风险

## 2. 必读上下文

评审前必须至少读取：

1. `DESIGN.md`
2. `AGENTS.md`
3. 与本次归属层匹配的 `docs/contracts/*`
4. 与本次功能匹配的 `docs/designs/*`
5. `docs/acceptances/ACCEPTANCE_CHECKLIST.md`

低代码生成器相关评审必须额外读取：

- `docs/designs/GENERATOR_MODULE_DESIGN.md`
- `docs/designs/LOWCODE_GENERATOR_GUIDE.md`
- `docs/designs/MODULE_CONTRACT.md`
- `docs/designs/ERROR_CODE_AND_I18N.md`

## 3. 固定评审顺序

### 3.1 范围确认

- `git diff --stat`
- `git diff --name-only`
- 检查是否存在生成文件、注册表、schema、i18n 资源同时变更
- 检查是否存在未解释的跨层改动

### 3.2 架构边界

- `business/*` 不得直接 import `modules/system/*` 的 service / repository / handler
- `system/config` 不得吞并 `auth / iam / org` 职责
- `platform` 聚合页不得写入单一系统子域
- 根装配器只能组装模块，不承载模块内部业务逻辑

### 3.3 Schema 与数据库

- 表名必须使用 `system_` 或 `biz_` 前缀
- 新字段必须确认索引、唯一约束、审计字段和枚举来源
- 关系表默认不生成导航，除非设计文档明确说明
- 低代码 schema 必须保留 `displayNameEn`、字段英文文案、业务上下文、表角色、依赖、关系和数据权限模式

### 3.4 权限与菜单

- 菜单只承载导航，权限控制动作
- 页面权限、按钮权限、接口权限必须能独立演进
- 禁止继续用 `list` 权限代表 `create / update / delete`
- 生成器页面权限与高敏生成动作权限必须拆开：`system:generator:use` 与 `system:module:generate`
- 业务子模块必须确认父级菜单、页面菜单、按钮权限、组件 key 一致

### 3.5 多语言

#### 3.5.1 硬编码文本检测 `[Auto]`

- 执行 `npm run check:i18n-hardcode`，扫描中文/英文/日文/韩文硬编码字符串
- 覆盖范围：页面标题、按钮、提示文案、Modal 文案、错误文案、空态文案、placeholder、表格列名、面包屑、页签标题
- 排除范围：console.log、注释、测试文件、mock 数据、i18n 资源文件自身

#### 3.5.2 Key-first 原则 `[PR]`

- 前端展示文本必须走 `t()` 或等价能力
- 菜单和模块注册必须使用 `titleKey`，不得直接写自然语言
- 后端错误返回 key，不返回终端自然语言
- 新增/生成页面坚持 key-first，不允许硬编码展示文案
- i18n key 命名按域分层：`common.*` / `auth.*` / `system.user.*` / `system.role.*` / `biz.*`
- 低代码生成结果必须 key-first，并同步前端 fallback 与后端 i18n seed
- 生成链路允许只维护 `zh-CN / en-US` 初始翻译，其他 locale 必须有明确 fallback 策略

#### 3.5.3 日期与货币格式 `[PR]`

- 日期时间展示必须通过 `i18n.language` 感知当前 locale 格式，不硬编码 `YYYY-MM-DD` 等固定格式
- 货币金额展示不得硬编码 `¥` / `$` 符号，使用 `Intl.NumberFormat` 或 i18n 库格式化
- 数字千分位、小数点符号随 locale 变化
- 时区转换展示使用用户偏好时区，不强制 UTC

#### 3.5.4 RTL（从右到左）适配 `[Phase]`

- CSS 是否使用逻辑属性（`margin-inline-start` 替代 `margin-left`，`padding-inline` 替代 `padding-left/right`）
- Arco 组件是否在 RTL 方向下正常（当前可作为 P2 预留）
- 文本对齐避免硬编码 `text-align: left`，使用 `text-align: start`
- 图标方向是否有 RTL 语义（如返回箭头、面包屑分隔符）

#### 3.5.5 翻译覆盖率 `[Auto]` + `[PR]`

- 执行 `npm run audit:i18n-locales` 检查各 locale key 集合一致性
- 新增 locale 必须补齐 fallback 资源 + 远端资产 + 构建验证
- 缺失 locale 不得 fallback 回退到英文硬编码，使用结构化 fallback 链：远端翻译 → 前端 fallback → 通用兜底 key → 原始 key
- 导入导出链路（CSV 列头、结果摘要、重命名报告）随语言切换
- 请求失败、网络异常、超时时不再泄漏 `Network Error`、`Request Failed` 等英文硬编码 fallback

#### 3.5.6 长文本兼容 `[PR]`

- 英文/德文通常比中文长 30%~50%，按钮区、表格列头、筛选项不得因翻译长度撑爆布局
- 动态菜单标题、页头标题允许文本溢出省略，但须有 tooltip 展示完整文本

### 3.6 前端页面与 UI

#### 3.6.1 设计源引用 `[Auto]`

- 所有 UI 规则以上游文档为准：`DESIGN.md` → `docs/designs/FRONTEND_UI_SPEC.md` → `docs/designs/BACKOFFICE_STYLE_CONSTRAINTS.md`
- 评审时不凭个人审美判断，只检查偏离文档的违规点

#### 3.6.2 统一组件封装 `[PR]`

| 场景 | 统一方案 | 禁止 |
| :--- | :--- | :--- |
| 浮层 | 平台封装的 Modal / Drawer | 原生 Arco Modal/Drawer |
| 确认弹窗 | 平台确认组件 | `Modal.confirm` 裸调 |
| 通知/消息 | 统一 Message/Notification 封装 | 每页自行调用方式不一致 |
| 右侧辅助栏 | `SummaryRail` / `RiskRail` / `PolicyRail` 三模板 | 旧类名、自建右栏 |
| 页面骨架 | 标准列表/详情/配置/工作台骨架 | 每页自行发明布局 |

- 封装层只做三件事：统一 token（圆角/阴影/间距）、统一交互（footer 按钮顺序、focus 回退、esc 关闭）、统一 i18n（关闭/确认文案）
- 封装层不做 props 截断、不改 Arco 默认行为

补充检查：

- Modal 用于短表单、确认动作、聚焦创建/编辑；Drawer 用于多分区详情、长表单、需要保留上下文的连续编辑
- 不允许把完整页面压缩塞进弹窗或抽屉
- Footer 按钮顺序统一为"次操作在左，主操作在右"；危险操作额外要求二次确认链路
- 不允许让关闭、取消、返回同时出现，造成退出路径混乱

#### 3.6.3 Token 一致性 `[Auto]` + `[PR]`

- 全局扫描检查是否存在 Arco 原始 token（`--color-text-1`、`--color-border-2`、`--color-fill-1` 等），所有颜色引用统一使用 Pantheon token
- 间距基于 4px 基准单位（4/8/12/16/24/32），页面 padding 24px，筛选区与表格区间距 16px
- 圆角来自四档 token：`--radius-xs` 4px / `--radius-sm` 6px / `--radius-md` 8px / `--radius-lg` 12px
- 阴影使用 token 级轻阴影，禁止主按钮彩色大阴影
- 字体：Source Sans 3（正文/UI）/ JetBrains Mono（代码），字重仅使用 400/500/600/700，禁止 650/620

#### 3.6.4 状态完整性 `[PR]`

- 每个页面必须处理六态：`loading` / `success` / `empty` / `error` / `forbidden` / `submitting`
- `loading`：页面级使用 skeleton 或 spin，表格级使用表格 loading，不允许整页闪烁
- `empty` 区分：首次使用空态 / 筛选后无结果 / 无权限空态
- `error` 至少覆盖：网络错误 / 服务器错误 / 请求超时 / 数据加载失败
- `forbidden`：页面级无权限 + 操作级无权限统一 403 表达
- 检查 Modal / Drawer 是否走统一平台封装

#### 3.6.5 禁止清单 `[PR]`

- `radial-gradient` 光晕装饰
- `linear-gradient` 大面积渐变（侧栏、内容区、卡片表面）
- `::before` 伪元素网格背景叠加
- 主按钮彩色大阴影（`0 8px 18px` 级投影）
- 非标准 `font-weight` 值（650/620）
- Inter 作为主字体（已替换为 Source Sans 3）
- 紫蓝渐变背景伪装高级感
- 彩色圆形图标卡片装饰
- 未实现真实行为的控件展示为可点击能力
- 页面内容全部居中

补充平台层混合态检查：

- `rg "system-page-side|system-page-summary-card|system-page-note|system-page-main-grid|system-page-main" frontend/src` 是否为 0 命中
- `rg "<Modal|<Drawer" frontend/src/modules frontend/src/components` 是否只剩平台封装组件
- `rg "Modal\\.confirm|Modal\\.(success|error|info|warning)" frontend/src` 是否只剩平台封装内部命中

#### 3.6.6 可访问性（A11y）`[PR]` + `[Phase]`

- 所有可交互元素必须有清晰 focus 态，使用 `2px dashed` 或 `2px solid`，全局一致
- 不可只靠颜色表达状态，状态/错误必须同时有图标 + 颜色 + 文本
- 表单必填字段标记 `aria-required`，错误提示关联 `aria-describedby`
- 图片必须有 `alt` 属性（纯装饰图 `alt=""`）
- Icon-only 按钮必须有 `aria-label` 或 tooltip
- 浮层（Modal/Drawer/Dropdown）关闭后焦点回退到触发元素
- 页面 `lang` 属性随语言切换正确更新 `<html lang="...">`
- 色彩对比度：正文文本 ≥ 4.5:1，大文本 ≥ 3:1 `[Phase]`

#### 3.6.7 响应式验收 `[PR]` + `[Phase]`

- 固定视口覆盖：PC 1440×900 / Pad 1024×768 / Phone 390×844
- 中小屏退化：筛选区折叠、表单单列化、表格允许横向滚动、侧边栏抽屉化
- 壳层改动必须执行竖版侧栏 + 横版顶栏双模式验收：
  - 竖版侧栏展开态 / 折叠态
  - 横版顶栏主导航态 / 弹出子菜单态
- 固定通过门槛：不允许只提交单一布局截图或单一布局结论
- 统一使用 `docs/acceptances/PLATFORM_SHELL_DUAL_MODE_ACCEPTANCE_TEMPLATE.md`

### 3.7 安全与审计

#### 3.7.1 OWASP Top 10 对照清单 `[PR]` + `[Phase]`

| 漏洞类型 | 检查项 | 层级 |
| :--- | :--- | :--- |
| **A01 访问控制失效** | 接口是否有 Casbin 权限守卫；按钮是否按权限隐藏；越权尝试返回 403 而非 200 | `[PR]` |
| **A02 加密失效** | 敏感配置是否加密存储；密码是否 bcrypt 哈希；token 是否设置合理过期 | `[PR]` |
| **A03 注入** | SQL 参数化查询（检查 GORM `Raw()`/`Exec()` 使用）；前端 URL 参数是否 encode；命令执行是否参数化 | `[PR]` |
| **A04 不安全设计** | 敏感接口是否有二次验证；导入/导出/生成/卸载等治理动作是否有完整失败路径 | `[PR]` |
| **A05 安全配置错误** | 生产环境 debug 是否关闭；CORS 是否限制允许域；错误响应不暴露堆栈/表结构 | `[Phase]` |
| **A06 脆弱/过时组件** | `npm audit` 无 high/critical；`go mod tidy` 无已知 CVE 依赖 | `[Auto]` |
| **A07 认证失效** | JWT 过期/刷新逻辑是否正确；MFA 二次验证是否在高敏操作前触发；登录失败是否有节流/锁定 | `[PR]` |
| **A08 软件与数据完整性** | 文件上传是否校验类型/大小；导入 CSV 是否有格式校验与行级错误报告 | `[PR]` |
| **A09 日志与监控** | 关键操作是否写审计日志（含 requestId）；敏感字段是否脱敏 | `[PR]` |
| **A10 SSRF** | 服务端请求（Webhook、文件下载等）是否校验目标 URL；是否限制内网地址 | `[Phase]` |

#### 3.7.2 XSS 专项 `[PR]`

- 用户输入渲染到 DOM 时是否经过 React 默认转义
- `dangerouslySetInnerHTML` 使用是否经过 sanitize（DOMPurify 或等价）
- URL 跳转是否校验 `javascript:` / `data:` 协议
- 富文本编辑器内容是否在服务端做 HTML 清洗

#### 3.7.3 CSRF 专项 `[PR]`

- 所有写操作（POST/PUT/DELETE/PATCH）是否携带 CSRF token 或走 SameSite Cookie
- 敏感操作（删除/批量操作/导出/配置变更）是否有额外确认或二次验证

#### 3.7.4 敏感数据泄露 `[PR]` + `[Phase]`

- 前端请求/响应中不得暴露：密码原文、完整 token、内部路径、未脱敏手机号/邮箱
- 审计日志中敏感字段是否脱敏（密码、token、身份证号）
- API 响应不得泄露数据库错误、表名、字段名、堆栈信息

#### 3.7.5 HTTP 安全头 `[Auto]` + `[PR]`

- 响应头是否包含：`X-Content-Type-Options: nosniff` / `X-Frame-Options: DENY` / `Referrer-Policy: strict-origin-when-cross-origin`
- CSP（Content-Security-Policy）是否存在，至少限制 `script-src 'self'` `[Phase]`
- HSTS（Strict-Transport-Security）是否在生产环境启用 `[Phase]`

#### 3.7.6 认证与会话安全 `[PR]`

- JWT 算法固定为 HS256/RS256，禁止 `alg: none`
- access_token 有效期 ≤ 15min，refresh_token 轮换后旧 token 立即失效
- 密码策略：最短 8 位 + 复杂度要求（大小写/数字/特殊字符至少 3 类）
- 历史密码复用限制（最近 5 次不可用）
- 会话管理：支持查看活跃会话、远程强制下线
- MFA / TOTP 在高敏操作前触发二次验证

#### 3.7.7 接口防滥用 `[PR]` + `[Phase]`

- 登录/验证码接口是否有频率限制（来源级节流）
- 批量操作接口是否有单次数量上限
- 导出接口是否有数据量限制，防止 OOM
- 文件上传：MIME 白名单、文件大小上限、文件名防路径穿越

#### 3.7.8 前端存储安全 `[PR]`

- token 不得存储在 `localStorage`（防 XSS 直接读取），优先 httpOnly cookie
- 敏感状态不得持久化在 `sessionStorage` 或 URL 参数中

#### 3.7.9 依赖安全 `[Auto]`

- `npm audit --audit-level=high` 无结果（CI 门禁）
- Go 依赖定期检查更新 `[Phase]`

### 3.8 测试副作用

低代码生成器和动态模块测试会重写 generated 注册表。评审时必须检查以下文件没有被误清空：

- `backend/modules/business/generated_registry.go`
- `frontend/src/modules/generated/business.ts`
- `frontend/src/core/router/generatedComponentRegistry.ts`
- `schema/generated/**`

如果测试需要改写工作区，必须满足其一：

- 在临时 workspace 中执行
- 或测试后显式验证并恢复真实 generated registry

### 3.9 性能

#### 3.9.1 首屏加载 `[Auto]` + `[PR]`

- Lighthouse Performance 评分 ≥ 90（阶段评审时跑 `lighthouse-batch`）
- LCP ≤ 2.5s / FCP ≤ 1.8s / TBT ≤ 200ms
- 路由懒加载：非首屏页面使用 `React.lazy()` + `Suspense` 按需加载
- 公共库按需引入，禁止全量 import（如 `import Arco from '@arco-design/web-react'`）
- 首屏资源总大小 ≤ 500KB（gzip 后）

#### 3.9.2 API 响应时间 `[Phase]`

- 列表查询 < 1s（含分页）；详情查询 < 500ms
- 大数据量导出是否走异步 + 轮询，而非阻塞请求
- 高频接口是否有缓存策略（字典/语言包/系统设置等低频变更数据）
- 日志列表查询是否有时间范围默认限制，避免全表扫描

#### 3.9.3 虚拟滚动 `[PR]`

- 表格/列表数据量 > 200 条时，是否启用虚拟滚动
- 树形数据量大时，使用虚拟列表 + 懒加载子节点

#### 3.9.4 图片与资源懒加载 `[PR]`

- 图片是否使用 `loading="lazy"` 属性
- 非首屏图片/图表是否通过 `IntersectionObserver` 或骨架占位延迟渲染
- Icon 使用按需引入（SVG symbol 或 tree-shakable），不加载完整 icon 库

#### 3.9.5 内存泄漏排查 `[PR]`

- `useEffect` 中 `addEventListener` / `setInterval` / `setTimeout` / `IntersectionObserver` 是否有对应 cleanup
- `unmounted` 后是否不再调用 `setState`
- WebSocket / SSE 连接在组件卸载时是否关闭

#### 3.9.6 构建产物 `[Auto]` + `[PR]`

- 构建后 bundle 分析：无重复打包（如 lodash 被打入两处）、无未使用依赖
- Tree shaking 生效（dead code 占比 ≤ 5%）
- vendor chunk 与业务 chunk 分离合理

### 3.10 代码规范

#### 3.10.1 ESLint / Prettier `[Auto]`

- `eslint` 零 error（门禁不可绕过）
- `prettier --check` 通过
- 规则集与下游构建/测试命令统一

#### 3.10.2 命名约定 `[PR]`

- 组件文件/目录：PascalCase（`UserList.tsx`）
- 工具函数/API 模块：camelCase（`formatDate.ts`、`systemApi.ts`）
- 常量：UPPER_SNAKE_CASE
- 路由 path：kebab-case（`/system/login-log`）
- 数据库表名：`system_` / `biz_` 前缀 + snake_case
- CSS 类名：BEM 或 Pantheon 统一的 `p-` 前缀
- 禁止拼音命名、缩写不解释（如 `usr`、`cfg` 无可查定义）

#### 3.10.3 组件拆分粒度 `[PR]`

- 单文件 ≤ 300 行（超出应拆分）
- 组件只做一件事：展示 / 数据获取 / 状态管理 / 业务逻辑四个角色不混在一个组件中
- 列表页标配：`SearchForm` + `DataTable` + `Pagination` 独立组件，不写成一个巨型 JSX
- 重复 ≥ 3 遍的 JSX 片段抽为独立组件

#### 3.10.4 业务逻辑与 UI 解耦 `[PR]`

- 数据获取（API 调用）、状态管理、UI 渲染三层分离
- 业务逻辑和类型定义禁止直接依赖 UI 库的组件类型（如 `Arco.FormInstance` 不作为业务函数的参数类型）
- API 请求不在组件内裸写 `fetch/axios`，统一走 `apiClient` 或模块 API 封装
- 表单校验逻辑可独立于组件单独测试

#### 3.10.5 TypeScript `[PR]`

- 禁止 `any`（确有必要的场景需注释说明理由）
- 接口/类型定义优先于类型推断
- 组件 Props 必须有明确接口定义，不接受隐式 `any`

#### 3.10.6 Git 规范 `[PR]`

- Commit Message 遵循 Conventional Commits（`feat:` / `fix:` / `docs:` / `refactor:` / `chore:`），subject ≤ 72 字符，使用英文
- 分支命名：`feat/<name>` / `fix/<name>` / `docs/<name>`，使用短横线连接
- PR 单一目的：不混入无关重构或格式化变更
- 一个 commit 只做一件事，可独立回滚、独立 review
- 禁止提交：`console.log`、无归属 `TODO`、注释掉的代码块、`.env` 等敏感文件
- 合入前 squash 为有意义的历史节点，不保留 `fix typo`、`wip` 等中间态 commit

## 4. 固定验证命令

按改动范围选择执行，不能只跑单一 build。

### 4.1 前端

```bash
cd frontend
npm run check:menu-contract
npm run check:i18n-hardcode
npm run audit:i18n-locales
npm run type-check
npm run lint
npm run format:check
npm run audit
npm run build
```

### 4.2 后端

```bash
go test ./backend/internal/scaffold ./backend/modules/system/generator ./backend/modules/system/dynamicmodule ./backend/modules/system/i18n
go test ./backend/modules/business
```

### 4.3 性能审计（阶段评审）

```bash
cd frontend
npx lighthouse-batch --score=90
```

### 4.4 UI 冒烟

涉及后台 UI、导航、对话框、响应式布局时，必须补浏览器证据：

- 默认布局
- 折叠菜单
- 横版顶栏
- 移动端窄屏
- 关键弹窗或抽屉

## 5. 发现项格式

评审输出必须 findings first，按严重程度排序：

```text
[P0|P1|P2] (confidence: N/10) file:line - 问题描述
影响：用户或平台会看到什么错误
修复：已修复 / 建议修复方式
验证：对应命令或证据
```

严重程度定义：

- `P0`：会导致安全问题、数据损坏、构建失败、核心链路不可用
- `P1`：会导致模块不可达、权限/i18n/菜单契约失效、明显行为回归
- `P2`：治理、文档、测试覆盖、可维护性缺口

## 6. 评审完成定义

一次评审只有同时满足以下条件，才能标记为通过：

- 已声明归属层和跨层边界
- 已读对应设计和验收文档
- 已检查 diff 中所有生成物与注册物
- 已修复可自动修复的 P0 / P1
- 已记录剩余 P2 和后续处理建议
- 已运行或明确说明未运行的验证命令
- 若代码影响合同、接口、菜单、权限、i18n、数据库或验收口径，已同步更新文档

## 7. 双层执行矩阵

| 维度 | 章节 | Auto（CI 门禁） | PR（提交前必查） | Phase（阶段深度） |
| :--- | :--- | :--- | :--- | :--- |
| 架构边界 | §3.2 | — | 跨层 import、业务侵入系统域 | 模块边界漂移审计 |
| Schema | §3.3 | — | 表前缀/索引/审计字段 | 低代码 schema 完整性 |
| 权限菜单 | §3.4 | `check:menu-contract` | 菜单≠权限、按钮权限不代理、生成器权限分离 | 权限工作台缺口扫描 |
| 多语言 | §3.5 | `check:i18n-hardcode` `audit:i18n-locales` | key-first、日期/货币格式、长文本兼容 | RTL 适配、翻译覆盖率审计 |
| UI | §3.6 | Token 扫描 | 封装组件合规、状态完整、禁止清单、双模式验收 | A11y 对比度、全视口回归 |
| 安全 | §3.7 | `npm audit` | XSS/CSRF/越权/认证/敏感数据/安全头 | CSP/SSRF/CVE 依赖/接口防滥用策略 |
| 测试副作用 | §3.8 | — | generated registry 完整性 | — |
| 性能 | §3.9 | — | 虚拟滚动、内存泄漏、懒加载 | Lighthouse 评分/bundle 分析/API 响应时间 |
| 代码规范 | §3.10 | `lint` `format:check` | 命名、组件拆分、业务解耦、TS 禁 any、Git 规范 | 组件重构审计、技术债追踪 |
