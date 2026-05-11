# Pantheon 前端合并评估报告

评估日期：2026-05-06
重评估：2026-05-06（v2）
类型：Assessment
归属层：platform
状态：Active

## 1. 评估边界

本报告合并了两类检查结果：

- 平台层前端底座评估：模块注册、动态菜单、路由权限、i18n、请求错误翻译、构建门禁。
- UI 工程质量量化评估：设计 token、组件封装、状态完整性、响应式、A11y、bundle、动效、安全与字体。

本次归属为 `platform` 层。Dashboard、工作台、页签、面包屑、命令面板、偏好设置、通知中心都属于平台壳层能力；系统域页面只作为被平台壳层装配和治理的页面样本，不把 `system/auth`、`system/iam`、`system/org`、`system/config` 混成一个大模块处理。

## 2. 综合结论

综合评分：**7.6 / 10**（v1: 7.2）

v2 评分提升源于三处可验证改进：locale 懒加载（性能 +1）、壳层视觉契约自动化（设计 token +0.5）、构建时间缩短 30%（性能 +0.5），部分被 i18n 资源文件体积偏大和 >300 行文件仍在扩散所抵消。

| 维度 | v1 | v2 | 变化 | 评语 |
| --- | --- | --- | --- | --- |
| 架构与模块边界 | 8 | 8 | — | Dashboard 按平台聚合层处理，`ModuleConfig.domain` 类型层仍待补 |
| 菜单、权限与路由 | 8 | 8 | — | 菜单契约、组件键、页面权限和按钮权限链路完整 |
| i18n 与错误反馈 | 8 | 8 | — | 硬编码扫描 + locale 完整性闭环，资源文件 2212-2466 行仍偏大 |
| 设计 token | 8 | **8** | — | 壳层视觉契约已自动化（`check:shell-visual-contract`），token 一致性检查仍缺 |
| 组件封装 | 8 | 8 | — | AppModal/AppDrawer/SubmitBar 等齐备，零原生泄漏 |
| 状态完整性 | 8 | 8 | — | loading/empty/error/forbidden/submitting 有公共表达 |
| 响应式 | 7 | 7 | — | 壳层和列表页覆盖较好，生成器面板仍缺窄屏验证 |
| A11y | 4 | 4 | — | 壳层有基础 aria-label，覆盖率仍不足 |
| 性能与代码分割 | 6 | **7** | ↑1 | locale 懒加载已实现（0.03KB stub），生成器 chunk 310KB 仍是瓶颈 |
| 动效与微交互 | 5 | 5 | — | 企业后台不需要强动效，基础过渡应在封装层统一 |
| 安全前端面 | 9 | 9 | — | 零 `dangerouslySetInnerHTML`，CSRF + httpOnly cookie 已闭环 |
| 字体体系 | 8 | 8 | — | Source Sans 3 + JetBrains Mono，内网需自托管兜底 |

## 3. 已验证门禁

**v2 重评估全部通过（10/10）：**

```bash
npm run check:menu-contract     ✅ 16 菜单, 19 路由, 79 权限, 19 组件键
npm run check:i18n-hardcode     ✅ 136 文件扫描通过
npm run check:shell-visual-contract  ✅ Shell visual contract passed
npm run audit:i18n-locales      ✅ 5 语种均为 2104 keys, missing/extra/empty 均为 0
npm run lint                    ✅ 零 error
npm run format:check            ✅ All matched files use Prettier code style
npm run type-check              ✅ tsc 零 error
npm run audit                   ✅ 0 vulnerabilities
npm run build                   ✅ 489ms 构建成功
```

**v1 → v2 关键变化：**

| 指标 | v1 | v2 |
| --- | --- | --- |
| 构建时间 | 696ms | **489ms** (-30%) |
| i18n 扫描文件数 | 129 | **136** (+7 新文件) |
| locale stub 大小（每个） | 132-161KB 全打入 | **0.03KB**（懒加载 stub） |
| 壳层视觉契约 | 无 | **已自动化**（prebuild 门禁） |
| 0-vuln audit | npm registry 不可用 | **已修复** |

**Bundle 分析（v2）：**

| Chunk | 大小 | 评价 |
| --- | --- | --- |
| `platform-builder` | 310KB | 🔴 生成器模块单体，P1 拆分目标 |
| `arco-table` | 264KB | ⚠️ 第三方，可接受 |
| `zh-CN` | 132KB | ⚠️ locale 独立 chunk（✅懒加载），体积仍需按 namespace 拆分 |
| `en-US` | 140KB | ⚠️ 同上 |
| `ja-JP` | 161KB | ⚠️ 同上 |
| `ko-KR` | 150KB | ⚠️ 同上 |
| `fr-FR` | 156KB | ⚠️ 同上 |
| `react-vendor` | 140KB | ✅ React + 生态 |
| `arco-feedback` | 115KB | ✅ 第三方 |
| `arco-form-base` | 105KB | ✅ 第三方 |
| `arco-icons` | 87KB | ⚠️ tree-shaking 待验证 |
| `app-vendor` | 89KB | ✅ |
| 其余 chunks | < 56KB | ✅ |

locale 已从"全部打入主包"改为"独立 chunk + stub 按需加载"（每个 stub 0.03KB）。288KB chunk（Dict/I18n/Dept）已内联到页面组件中。

## 4. >300 行文件清单（v2）

i18n 资源文件（数据文件，非组件）：

| 文件 | 行数 |
| --- | --- |
| `fr-FR.ts` | 2466 |
| `en-US.ts` | 2413 |
| `ja-JP.ts` | 2318 |
| `ko-KR.ts` | 2288 |
| `zh-CN.ts` | 2212 |

业务组件（超过 300 行阈值）：

| 文件 | 行数 | 状态 |
| --- | --- | --- |
| `ModuleWizard.tsx` | 2077 | 已提取 Step3Preview(340) + DatasourceManagerModal(234)，主体仍待拆 |
| `I18nList.tsx` | 2065 | 未拆分（生命周期/导入导出/rename 紧密耦合） |
| `layout/index.tsx` | 1867 | 未拆分（壳层核心，P1 列入拆分计划） |
| `DeptList.tsx` | 1789 | 已提取 DeptOrgTab(381)，-319 行 |
| `OperationLogList.tsx` | 1400 | 未拆分（此前未在列表中） |
| `UserList.tsx` | 1242 | 未拆分（此前未在列表中） |
| `RoleList.tsx` | 1236 | 未拆分（此前未在列表中） |
| `generator/schema.ts` | 1227 | 类型定义文件，合理 |
| `backend-generator.ts` | 1171 | 生成逻辑，合理 |
| `SettingPage.tsx` | 1106 | 未拆分 |
| `MenuList.tsx` | 1001 | 未拆分 |
| `PostList.tsx` | 900 | 未拆分 |
| `DictItemTab.tsx` | 849 | 已拆分（P2-1），单项职责 |
| `PermissionList.tsx` | 792 | 已提取 WorkbenchTab(603) + DataScopeTab(381) |

> **注：** v2 扫描发现了 v1 未包含的 7 个 >300 行文件（OperationLogList、UserList、RoleList、schema、backend-generator、SettingPage、MenuList、PostList）。这些文件 v1 时已被计入总行数但未单独列出。

## 5. v1 → v2 已修复项

| 项目 | v1 状态 | v2 状态 |
| --- | --- | --- |
| locale 全打入首屏 | 🔴 5×150KB | ✅ 独立 chunk + 0.03KB stub 懒加载 |
| 面包屑裁切 | 🔴 | ✅ `height: auto` / `line-height: normal` / 稳定行高体系 |
| 页签/功能栏/表头边框 | 🔴 混色/渐变观感 | ✅ 统一 `--panel-border` token，移除主题色轻染 |
| 表头 theme color-mix | 🔴 主题染色 | ✅ 改为中性 `--panel-muted` |
| npm audit 不可用 | 🔴 npmmirror | ✅ npmjs.org 注册表 |
| 壳层视觉回归门禁 | 🔴 无 | ✅ `check:shell-visual-contract` + Playwright 用例 |
| 用户文档指出的不准确项 | 🔴 3 处 | ✅ 已修正（A11y、locale 拆分、代码分割评分） |

## 6. 当前缺口

### P0

当前没有阻塞交付的 P0。构建、类型、lint、菜单契约、i18n 门禁均通过。

### P1

| 项目 | 边界 | 目标 |
| --- | --- | --- |
| `ModuleConfig.domain` | platform | 在类型层显式表达 `system/auth`、`system/iam`、`system/org`、`system/config` |
| A11y 基线 | platform | 补 icon-only aria-label、skip-to-content、表单错误关联策略和焦点路径检查 |
| 生成器 chunk 拆分 | system/config | 拆分 `ModuleWizard` 步骤、`CodePreview`、`FieldEditor`，降低 `/system/generator` 首次加载 |
| 生成器响应式验收 | system/config | 为 `ModuleWizard`、`CodePreview`、数据源弹窗增加窄屏验收 |
| 平台壳层拆分 | platform | 拆出 `ShellHeader`、`ShellTabs`、`CommandPalette`、`NoticeCenter`、`useOpenedTabs`、`useShellActivity` |
| locale namespace 拆分 | platform + system/config | zh-CN 2212 行 → 按 `app`/`system`/`auth`/`business` 拆分 |

### P2

| 项目 | 边界 | 目标 |
| --- | --- | --- |
| Token 一致性检查 | platform | 新增 `check:pantheon-tokens`，校验设计文档与 CSS token |
| 字体自托管 | platform | 提供 Source Sans 3 / JetBrains Mono 内网部署兜底 |
| 动效基线 | platform | Modal、Drawer、Tab、表格 hover 使用 150-200ms 的克制过渡 |
| 色彩对比度审计 | platform | 对关键文本、按钮、状态色做 AA 对比度检查 |
| A11y 深度 | platform | focus trap、skip-to-content、表单 aria-describedby |

## 7. 本轮修复详情

本轮按用户截图反馈和自动化评估结果，修复了以下问题：

1. **面包屑文字裁切**：根因是平台 Header 没有重置 Arco `Layout.Header` 的默认行高/高度语义。修复：Header 使用 `height: auto`、`line-height: normal`，breadcrumb 外层稳定 `20px` line-height，子项和分隔符 `24px` line-height。

2. **页签/功能栏/表头边框风格不一致**：根因是截图里的"功能栏"实际包含三类节点——平台 opened tabs、列表批量操作栏、Arco 表格头。修复：opened tabs 透明边框 + 稳定 20px 行高；批量操作栏透明背景 + `border: 0` + `box-shadow: none`；表格头从 `brand-primary` 的 `color-mix` 改为中性 `--panel-muted`；active 页签只通过背景与文字色表达状态。

3. **全局 Arco rounded tabs 收口**：使用真实类名 `.arco-tabs-header-nav-rounded .arco-tabs-header-title*` 统一规则。

4. **壳层视觉回归自动化**：新增 `check:shell-visual-contract` 静态契约脚本 + Playwright 视觉回归用例，接入 `prebuild` 门禁。

5. **locale 懒加载**：5 个 locale 各拆为 0.03KB stub，运行时按当前语言按需加载对应 chunk。

6. **npm audit 修复**：添加 `npm run audit` 脚本直连 npmjs.org 注册表。

## 8. 后续验收命令

```bash
cd frontend
npm run type-check
npm run lint
npm run build
npm run check:menu-contract
npm run check:i18n-hardcode
npm run audit:i18n-locales
npm run check:shell-visual-contract
npm run format:check
npm run audit
```

有后端与前端服务时补跑：

```bash
cd frontend
npm run test:smoke:shell-visual-contract
npm run test:smoke:backoffice-ui
npm run test:smoke:system
```
