# 设计与实现验收清单

更新时间：2026-04-30

类型：Acceptance
归属层：platform
状态：Active

本文不是路线图，也不是模块契约。

它解决的问题是：

- 一份设计文档什么时候算“能开始写代码”？
- 一个模块什么时候算“具备上线前的基本完成度”？
- 后续每个阶段完成后，团队和 AI 应该检查什么？

如果没有统一验收清单，项目很容易出现：

- 文档写了一半就开始写代码
- 功能做出来了，但权限、多语言、审计漏了
- 页面能打开，但 loading / empty / forbidden 都没做
- 后面只能不断返工

## 1. 使用原则

验收必须按阶段进行，不能只看“功能能不能跑”。

代码评审固定执行 `docs/acceptances/CODE_REVIEW_STANDARD.md`。该文档是提交前、阶段评审和 AI 自动评审的标准入口；本清单负责定义验收维度，代码评审标准负责定义执行顺序、发现项格式和固定验证命令。

建议分为五层：

1. 设计验收
2. 数据与接口验收
3. 前端页面验收
4. 系统集成验收
5. 发布前验收

## 2. 设计验收（开始编码前）

当一个模块还处于设计阶段时，至少检查以下内容。

### 2.0 合同前置条件

- 是否已先判断本次需求归属 `platform`、`system/auth`、`system/iam`、`system/org`、`system/config` 或 `business/*`
- 是否已有对应 `Contract`
- 若没有对应 `Contract`，是否先补合同或合同骨架，而不是直接进入设计或代码
- 当前设计文档是否已明确回链对应合同
- 当前文档是否已标明 `类型 / 归属层 / 状态`
- 若属于新的阶段评估稿或整改稿，是否已标明 `关联合同`

### 2.1 边界是否清晰

- 是否已明确模块属于 `platform`、`system/*` 还是 `business/*`
- 是否已说明与其他模块的边界
- 是否已说明允许依赖和禁止依赖
- 是否避免把 `auth / iam / org / config` 混为一个模块
- 若实现后台管理批量删除，是否按 `system/iam`、`system/org`、`system/config` 分域落点，而不是提供跨域通用删除器

### 2.2 文档是否完整

- 是否已有模块概述
- 是否已有业务流程或状态流转说明
- 是否已有数据模型设计
- 是否已有 API 清单
- 是否已有权限清单
- 批量删除是否声明独立权限点、二次验证要求、部分成功响应和失败原因结构
- 是否已有菜单与路由设计
- 是否已有 i18n 规划
- 是否已有审计与安全要求
- 若模块属于 `system/auth` 且声明预留 `MFA / SSO / CAPTCHA`，是否明确写清“当前只预留设计还是已经进入真实协议实现”
- 若模块属于 `system/config`，是否明确它归属 `dict / setting / i18n / upload / dynamicmodule / generator` 哪个子域
- 若模块属于高敏治理能力，是否说明权限、二次验证、环境限制与回滚策略

### 2.3 是否满足文档前置条件

- 是否与 `DESIGN.md` 保持一致
- 是否符合 `docs/contracts/DOCUMENT_GOVERNANCE_CONTRACT.md`
- 是否符合 `docs/contracts/DOCUMENT_METADATA_AND_STATUS.md`
- 是否符合 `docs/designs/MODULE_CONTRACT.md`
- 是否符合 `docs/designs/PERMISSION_MODEL.md`
- 是否符合 `docs/designs/ERROR_CODE_AND_I18N.md`
- 是否符合 `docs/designs/FRONTEND_PAGE_TEMPLATES.md`
- 主设计文档若进入主索引，是否满足 `Contract + Active` 或 `Design + Active` 的展示规则

## 3. 数据与接口验收

开始写后端前，至少检查以下内容。

### 3.1 数据模型

- 表前缀是否符合规范（`system_` / `biz_`）
- 主表、子表、关系表是否定义清楚
- 唯一约束是否明确
- 查询索引是否明确
- 审计字段是否完整
- 枚举字段是否说明来自字典或固定枚举

### 3.2 API 设计

- 是否列出了核心接口
- 是否区分查询类接口和动作类接口
- 是否定义了分页、排序、筛选参数
- 是否定义了成功响应结构
- 是否定义了错误 key
- 是否标注了接口权限点

### 3.3 安全与幂等

- 是否识别敏感接口
- 是否说明幂等要求
- 是否说明并发控制要求
- 是否说明审批/提交/撤销等动作的状态校验

## 4. 前端页面验收

开始写前端前，至少检查以下内容。

### 4.1 页面骨架

- 是否明确页面属于哪类模板
- 是否明确筛选区、表格区、详情区、表单区结构
- 是否使用统一页面骨架思路
- 是否定义页面标题、副标题、主操作区

### 4.2 状态完整性

- 是否考虑 `loading`
- 是否考虑 `empty`
- 是否考虑 `error`
- 是否考虑 `forbidden`
- 是否考虑 `submitting`

### 4.3 交互与反馈

- 是否定义删除确认、危险操作确认
- 是否定义提交成功 / 失败提示
- 是否定义长列表、长表单、抽屉/弹窗行为
- 是否定义批量操作反馈

### 4.4 多语言

- 是否为页面标题、按钮、字段、提示、空态定义 key
- 是否避免硬编码中文或英文
- 是否明确模块 i18n key 前缀

## 5. 模块接入验收

模块完成开发后，至少检查以下内容。

### 5.0 合同回链检查

- 本次开发是否仍在对应合同边界内
- 若实现已经改变边界、完成定义或验收口径，是否先更新合同，再更新实现
- 新增的 `Design / Assessment / Remediation / Acceptance` 文档是否都已回链对应合同
- 若历史文档已被新文档覆盖，是否已删除、降级或标记为 `Superseded / Archived`

### 5.1 后端接入

- 是否有清晰模块装配入口
- 是否已接入路由
- 是否已规划或实现 seed
- 是否没有把模块逻辑散落到装配器

### 5.2 前端接入

- 是否通过模块注册方式接入
- 是否没有把业务路由写死在 Layout
- 是否已接入菜单映射
- 是否已接入权限判断

### 5.3 平台能力接入

- 是否补齐菜单 seed
- 是否补齐权限 seed
- 若新增批量删除，权限 seed 是否使用独立 `*:batch-delete` 动作权限，且与前端模块声明、后端路由一致
- 是否补齐 i18n seed
- 是否补齐字典 / 配置依赖
- 是否补齐审计点
- 若属于 `business/*` 子模块切片，是否已说明它与父业务域入口的菜单关系、权限前缀和 namespace 边界

## 6. 功能验收

功能完成后，至少验证以下能力。

### 6.1 基础功能

- CRUD 或核心动作是否可正常执行
- 列表筛选、分页、排序是否正确
- 详情加载是否正确
- 表单校验是否完整
- 高频新增/编辑页是否已纳入“表单态矩阵”验收
- 治理动作页（如会话下线、日志清理、批量删除、模块卸载）是否已纳入独立“治理动作矩阵”验收，避免把无表单页面硬塞进 CRUD 表单用例

### 6.2 权限功能

- 无导航权限时是否不显示菜单
- 无页面权限时是否不能进入页面
- 无按钮权限时是否隐藏或禁用操作
- 无接口权限时后端是否返回 403
- 是否不再用 `list` 权限代理写操作
- 批量删除是否不复用单条 `delete` 或批量启停 `batch-update` 权限
- 批量删除是否经过二次验证，并复用单条删除保护逻辑返回部分成功结构
- 若 `system/iam` 权限工作台识别出推荐 API 缺口，是否支持在角色明细中触发受控补齐，而不是要求人工逐条录入 Casbin
- 权限工作台的一键补齐是否坚持“推荐映射 + 单角色 + 后端重算”约束，不接受前端任意提交 path / method 写策略
- 高敏整改动作是否经过二次验证，并在补齐后能立即收敛 `api-gap` 状态

### 6.2.1 `system/auth` 外部身份接入专项

- 若尚未确定身份源类型，是否坚持“不在登录页展示伪 SSO 能力”
- 若启用了 SSO provider，是否仍保留本地登录兜底，除非设计文档明确要求替代
- 是否存在 `external_identity -> local_user` 的明确绑定策略
- 是否明确“外部认证成功后仍签发本地 Pantheon 会话”，而不是直接透传第三方 token
- 是否补齐 provider 级审计、回调失败处理、注销语义与权限收敛说明

### 6.3 多语言功能

- 菜单标题是否按 key 翻译
- 页面标题和按钮是否可切语言
- 错误提示是否使用 message key 翻译
- 缺失 key 是否有 fallback 策略
- 导入结果摘要、错误 CSV、重命名迁移报告是否跟随语言切换
- 请求失败、网络异常、超时时是否不再泄漏 `Network Error`、`Request Failed` 等英文硬编码 fallback
- 动态菜单、页头、弹窗、空态、导入导出结果是否在不刷新整页的情况下完成语言刷新
- 新增或生成页面是否坚持 key-first，没有把展示文案硬编码进源码或生成产物

### 6.4 审计功能

- 关键操作是否写操作日志
- 认证/安全类动作是否写安全日志
- 敏感字段是否脱敏
- 是否能通过 `requestId` 把接口响应、前端报错与统一审计串联起来

### 6.5 `system/config` 扩展能力专项

- `/system/dict`、`/system/setting` 是否继续正常加载
- `/system/i18n` 是否已纳入固定验收范围
- `/system/modules` 是否已纳入固定验收范围
- `/system/generator` 是否已纳入固定验收范围
- 以上三类高敏治理页是否按 `docs/acceptances/SYSTEM_CONFIG_GOVERNANCE_ACCEPTANCE.md` 留存权限、二次验证、审计和失败态证据
- `system/config` 是否已明确区分普通配置能力与高敏治理能力
- `i18n` 的新增、编辑、导入、导出、缓存刷新与生命周期治理链路是否可解释
- 上传配置是否已验证大小、类型、路径和访问地址约束
- 动态模块注册、卸载、生成等写操作是否受更高权限和二次验证保护
- `/system/modules` 是否已区分页可见权限与 `register / unregister` 动作权限
- `/system/generator` 是否已区分页可见权限与真正的 `generate` 动作权限
- 若当前仍保留 `system:generator:use`，是否已把它标注为短期兼容权限而非长期高敏模型
- 失败场景下是否能明确区分“权限拒绝 / 环境限制 / 运行失败 / 工具失败”

### 6.6 `system/org` 治理闭环专项

- 组织健康总览是否能正确返回缺负责人、无岗位、空部门等摘要
- 部门治理筛选是否能直接定位到缺负责人、无岗位、空部门
- 部门负责人是否已收敛为“组织内真实可选人”
- 岗位在用时，禁用和删除是否被正确阻断
- 部门删除前，子部门、岗位、成员阻断是否完整
- 部门与岗位导出是否都输出统一治理模板列
- 治理导出是否同时输出机器值与 `*Label` 字典列
- 治理任务清单是否能覆盖部门问题、岗位问题与删除阻断
- “定位整改”入口是否能回到部门编辑、补岗位或组织架构链路

## 7. UI 验收

这是避免“后台虽然能用，但很像临时拼出来”的关键部分。

- 页面层级是否清晰
- 视觉是否保持克制和一致
- 筛选区与表格区间距是否统一
- 表单布局是否稳定
- 弹窗 / 抽屉宽度是否统一
- 空状态、错误态、无权限态是否统一
- 是否避免明显 AI 味设计堆砌
- 是否符合 `docs/designs/FRONTEND_UI_SPEC.md`
- 是否存在“旧壳层样式 + 新公共骨架”双轨并存却未被标注的状态

### 7.1 后台 UI 专项验收

后台 UI 专项整改完成时，必须额外检查：

- 登录页是否是专业认证控制台，而不是营销页或宣传 hero
- 登录页与应用壳层是否共用 theme token、圆角、阴影、背景和主色策略
- 应用壳层的侧边栏、顶部栏、页签、内容区是否保持统一 surface 与边框语言
- 平台工作台是否仍归属 `platform`，没有硬编码 `business/*` 卡片
- 工作台是否避免卡片墙，是否优先呈现状态、待关注事项、快捷入口和最近活动
- 所有展示控件是否有真实行为，未实现能力是否隐藏、禁用或明确说明
- 系统列表页、树表页、配置页是否统一使用页面骨架和 Arco 组件
- 是否符合 `docs/remediations/BACKOFFICE_UI_REMEDIATION_PLAN_20260423.md`

补充平台层混合态验收：

- 右侧栏是否明确属于 `Context` 或 `Alert`，而不是“历史模板复制”；
- 是否还在新增 `system-page-side / system-page-summary-card / system-page-note` 等旧右栏类名；
- `rg "system-page-side|system-page-summary-card|system-page-note|system-page-main-grid|system-page-main" frontend/src` 是否为 `0` 命中；
- `rg "<Modal|<Drawer" frontend/src/modules frontend/src/components` 是否只剩平台封装组件；
- `rg "Modal\\.confirm|Modal\\.(success|error|info|warning)" frontend/src` 是否只剩平台封装内部命中，而不是业务层直接调用；
- 是否仍有业务页面直接新增原生 `Modal` / `Drawer` 而绕过统一浮层模式；
- 左侧导航是否只承担导航，不混入说明卡、统计卡、帮助卡；
- 竖版侧栏和横版顶栏两种导航模式是否都完成验收，而不是只验证默认模式；
- 对仍未迁移的历史页面，是否已明确标记为“遗留待收口”，而不是默认视为通过。

### 7.2 `platform` 壳层双模式验收纪律

凡是改动以下任一对象，必须同时触发竖版侧栏与横版顶栏双模式验收：

- `frontend/src/core/layout/index.tsx`
- `frontend/src/core/layout/index.css`
- 动态菜单渲染、菜单图标映射、菜单选中态、菜单弹出层样式
- 影响导航区域的品牌区、页签区、顶部栏、偏好切换入口

固定验收输入：

- 竖版侧栏展开态
- 竖版侧栏折叠态
- 横版顶栏主导航态
- 横版顶栏弹出子菜单态

固定验收项：

- 当前菜单高亮是否一致
- hover / selected / open 三态是否一致
- icon badge 尺寸、间距、文字节奏是否一致
- 外链菜单、分组菜单、普通叶子菜单是否一致
- 面包屑、页签、头部品牌区是否未因布局切换而错位

固定通过门槛：

- 不允许只提交单一布局截图或单一布局结论
- 不允许出现“竖版已修复、横版待后续处理”的半通过状态
- 如有例外，必须在矩阵文档中显式记录为 `Pending`，不能口头保留

固定模板：

- 统一使用 `docs/acceptances/PLATFORM_SHELL_DUAL_MODE_ACCEPTANCE_TEMPLATE.md`
- 首个基准样例：`docs/archive/PLATFORM_SHELL_DUAL_MODE_ACCEPTANCE_20260430_LAYOUT_UNIFICATION.md`

提交流程要求：

- 后续壳层 PR 或阶段记录必须附双模式验收文档链接；
- 未附文档链接时，只能视为“待验收”，不能视为“已完成”；
- 若存在 `Pending`，必须同时附矩阵文档链接与挂账位置。
- PR 描述正文建议统一使用 `docs/acceptances/PLATFORM_SHELL_PR_TEMPLATE.md`。
- PR checklist 片段可直接复用 `docs/acceptances/PLATFORM_SHELL_PR_CHECKLIST_SNIPPET.md`。

## 8. 回归验收

新增模块或重构后，至少做以下回归检查：

- 是否影响登录态
- 是否影响动态菜单
- 是否影响权限判断
- 是否影响 i18n 资源加载
- 是否影响现有系统模块
- 是否影响通用页面骨架组件
- 前端 `prebuild` 是否通过 `check:i18n-hardcode`
- locale 审计是否通过 `audit:i18n-locales`
- 各 locale 是否与基准语言包保持 key 集合一致
- 是否重新扫描旧右栏类名、原生 `Modal` / `Drawer` 的新增扩散
- 若影响 `platform` 壳层导航，是否同步补齐竖版 / 横版双模式验收记录
- 若影响 `system/config`，是否重新验证 `/system/i18n`、`/system/modules`、`/system/generator` 的页面、权限和危险动作链路
- 若影响 `business/*`，是否重新验证业务入口页与其子模块页的菜单、页面权限、动作权限和审计链路
- 若影响 `business/*`，是否按 `docs/acceptances/BUSINESS_MODULE_ACCEPTANCE_MATRIX.md` 补齐九维验收证据

## 9. 发布前验收

准备进入联调或发布前，至少检查以下内容。

### 9.1 文档同步

- 文档是否已更新
- 是否已同步更新对应 `Contract`
- API 清单是否已同步
- 菜单、权限、i18n、DDL 是否已同步
- 是否补充了后续演进注意事项
- 是否检查新增或重写文档已补 `类型 / 归属层 / 状态`
- 是否检查新增的阶段评估稿、整改稿已补 `关联合同`
- 若有旧文档被替代，是否明确删除、归档或标记为 `Superseded`

### 9.2 测试准备

- 是否有最小测试用例清单
- 是否有角色权限测试账号规划
- 是否有多语言切换验证方式
- 是否有关键失败路径验证方式
- 是否覆盖菜单、页头、按钮、导入导出摘要、错误 CSV、重命名报告的语言切换验证
- 是否保留 `check:i18n-hardcode`、`audit:i18n-locales`、`npm run build` 三条国际化固定门禁的执行记录
- 是否验证 `GET /api/v1/health` 可被部署平台、探针或运维脚本直接调用
- 浏览器页面链路、UI 冒烟、截图证据是否默认使用 gstack browse / gstack Browser；Playwright 仅作为 CI/API smoke 或明确要求时的补充
- 是否保留一次前端全局扫描记录，至少包含旧右栏模式、原生浮层使用点和导航双模式验收结果

### 9.3 质量门槛

- 是否存在未解释的技术债
- 是否存在临时兼容逻辑未记录
- 是否存在未补权限 / i18n / 审计的缺口
- 是否存在新的用户可见硬编码展示文案未被 `check:i18n-hardcode` 覆盖

## 10. 推荐验收方式

建议后续把验收分成三种使用方式：

### 10.1 文档评审

用于判断“这份设计能不能开始开发”。

### 10.2 开发自检

用于模块开发者在提交前逐项自查。

### 10.3 阶段评审

用于每个路线图阶段结束后做统一收口。

阶段评审中的页面级证据默认来自 gstack：

- `platform` 层全局评估优先同步到 `docs/PLATFORM_GLOBAL_EVALUATION_*.md`
- 跨域验收结果优先沉淀到 `docs/PLATFORM_ACCEPTANCE_MATRIX_*.md`

- 使用 `browse chain` 或 gstack Browser 采集最终 URL、console error、页面快照与截图。
- 组织架构、访问控制、按钮权限、无权限态等交互链路优先用 gstack 做真实浏览器验证。
- Playwright 结果可以作为自动化补充，但不能替代 gstack 截图和交互证据；若 Playwright 因浏览器未安装或 Windows 权限失败，不直接判定业务功能失败。

## 11. 与其他文档的边界

| 文档 | 负责什么 | 不负责什么 |
| :--- | :--- | :--- |
| `docs/archive/IMPLEMENTATION_ROADMAP.md` | 规定阶段目标与顺序 | 不做逐项验收打勾 |
| `docs/contracts/DOCUMENT_GOVERNANCE_CONTRACT.md` | 规定文档治理主干与合同模型 | 不替代具体模块设计 |
| `docs/contracts/DOCUMENT_METADATA_AND_STATUS.md` | 规定文档类型、状态与主索引规则 | 不替代模块完成定义 |
| `docs/designs/MODULE_CONTRACT.md` | 规定模块接入方式 | 不做开发完成度验收 |
| `docs/designs/BUSINESS_MODULE_TEMPLATE.md` | 规定业务模块文档结构 | 不做阶段验收门槛 |
| `docs/designs/FRONTEND_UI_SPEC.md` | 规定 UI 设计细则 | 不做发布检查清单 |

## 12. 完成定义

如果一个模块在“设计、数据、接口、页面、接入、权限、多语言、审计、回归”九个维度都能回答清楚，才算具备进入下一阶段的资格。

否则就不算真正完成，只是“部分可运行”。

## 13. 2026-04-20 平台层验收对照结论

以下记录用于沉淀一次真实验收样例，范围属于 `platform` 与 `system/*` 底座层，不涉及 `business/*`。

### 13.1 本次验收范围

- `platform`：`/dashboard`
- `system/auth`：`/login`、`/auth/security`、`/system/login-log`、`/system/session`
- `system/iam`：`/system/profile`、`/system/user`、`/system/user/1`、`/system/role`、`/system/menu`、`/system/permission`、`/system/operation-log`
- `system/org`：`/system/dept`、`/system/post`
- `system/config`：`/system/dict`、`/system/setting`

### 13.2 对照结果

- 设计边界：通过，页面归属已按 `platform` / `system/auth` / `system/iam` / `system/org` / `system/config` 重新确认。
- 数据与接口：通过，运行时数据库与后端测试已统一切换为 MySQL，`auth` 与 `audit` 子域迁移注册完整生效。
- 前端页面：通过，登录后系统域页面均可打开，面包屑告警已消除，React/Arco 版本已重新对齐。
- 系统集成：通过，菜单导航、页面访问、接口调用、登录态注入和截图采集均跑通。
- 回归验收：通过，重点复核了安全中心、会话管理、登录日志、操作日志等此前受影响页面。
- 日志治理：登录日志与操作日志均需复核“保留最近 `1 / 7 / 30` 天清理”“按选择集删除”“二次验证拦截”三条链路，且不应再出现默认全表清空入口。
- 发布前文档：通过，已同步补充 QA 归档报告与 Windows 下 gstack 操作说明。

### 13.3 验收证据

- 最终汇总：`docs/archive/QA_SMOKE_REPORT_20260420.md`
- 机器可读结果：`.gstack/qa-reports/summary-20260420-final.json`
- 原始输出：`.gstack/qa-reports/raw/20260420-final`
- 页面截图：`.gstack/qa-reports/screenshots/20260420-final`

### 13.4 验收结论

- 当前 `platform` 与 `system/*` 底座层已满足一轮发布前基础冒烟要求。
- 本轮未发现新的真实页面级阻断故障。
- Windows 环境下 gstack 仍存在偶发浏览器会话中断现象，但已确认属于工具运行特性，不是系统页面故障；执行时应优先使用单条 `browse chain` 并在需要时提权启动。

## 14. 2026-04-29 补充验收基线

本节用于修正此前对 `system/config` 和 `business/*` 覆盖不足的问题。

### 14.1 `system/config` 固定页面基线

后续默认固定覆盖：

- `/system/dict`
- `/system/setting`
- `/system/i18n`
- `/system/modules`
- `/system/generator`

每页至少采集：

- 最终 URL
- console error
- 页面截图
- 页面快照或等价交互证据

### 14.2 `system/config` 高敏动作基线

以下动作不得只验证“按钮能不能点”：

- i18n 导入、导出、缓存刷新、key 重命名、生命周期治理
- 动态模块注册、卸载
- 生成器触发代码生成

至少检查：

- 页面权限
- 动作权限
- Casbin 接口权限
- 二次验证
- 环境限制
- 审计记录

### 14.3 `business/*` 固定页面基线

当业务域已经出现子模块切片时，不能只验父入口页。

以当前 `CMDB` 为例，自动化烟测至少覆盖已实现子模块：

- `/business/cmdb/host`

若后续新增供应商、资源类型、资源实例等子模块，必须同步扩展 `test:smoke:full-system`，不能只保留主机管理页。

至少检查：

- 菜单是否正确挂接到业务域入口
- 页面权限是否与模块声明一致
- 子模块权限前缀是否清晰
- i18n namespace 是否已补齐
- 审计点是否覆盖新增、编辑、删除、导入导出等关键动作

### 14.4 全系统页面烟测脚本

固定命令：

- `cd frontend; npm run test:smoke:full-system`

固定覆盖：

- `platform`：`/dashboard`
- `system/auth`：`/login`、`/auth/security`、`/system/login-log`、`/system/session`
- `system/iam`：`/system/profile`、`/system/user`、`/system/user/1`、`/system/role`、`/system/menu`、`/system/permission`、`/system/operation-log`
- `system/org`：`/system/dept`、`/system/post`
- `system/config`：`/system/dict`、`/system/setting`、`/system/i18n`、`/system/modules`、`/system/generator`
- `business/cmdb`：`/business/cmdb/host`

固定视口：

- PC：`1440x900`
- Pad：`1024x768`
- Phone：`390x844`

### 14.5 业务模块专项文档要求

如果业务域从单入口演进为多子模块切片，至少满足以下之一：

- 在主业务域设计文档中明确纳入子模块边界、路由、权限和验收
- 或者单独补对应子模块设计文档
- CMDB 业务域和 `business/cmdb/vendor` 归属以 `docs/designs/BUSINESS_CMDB_MODULE_DESIGN.md` 为准

不能再接受“代码里已有模块，文档里没有定义”的状态。
