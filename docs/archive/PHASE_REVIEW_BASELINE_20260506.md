# Phase 基线审查报告

审查日期：2026-05-06
归属层：`platform` + `system/*` 底座层
审查类型：Phase 深度（按 `CODE_REVIEW_STANDARD.md` §1~§7 执行）
基线 commit：`9843da6`

## 自动化门禁通过项

| 检查项 | 命令 | 结果 |
| :--- | :--- | :--- |
| i18n 硬编码扫描 | `check:i18n-hardcode` | ✅ 129 文件通过 |
| TypeScript 类型检查 | `type-check` | ✅ tsc 零错误 |
| 菜单契约一致性 | `check:menu-contract` | ✅ 16 菜单/19 路由/79 权限一致 |
| ESLint | `lint` | ✅ 零 error |
| 构建 | `build` | ✅ 619ms 成功 |
| 后端测试 | `go test ./backend/...` | ✅ generator/dynamicmodule/i18n/scaffold 全通过 |
| 旧右侧栏类名 | `rg system-page-side\|system-page-summary-card\|...` | ✅ 零命中 |
| 原生 Modal/Drawer (业务层) | `rg "<Modal\|<Drawer" src/modules` | ✅ 零命中 |
| Modal.confirm 裸调用 | `rg "Modal\.(confirm\|success\|error\|info\|warning)"` | ✅ 仅在 `AppModalActions` 封装层内部使用 |
| Arco 原始 token | `rg --color-text\|--color-border\|--color-fill` | ✅ 零命中 |
| dangerouslySetInnerHTML | `rg` | ✅ 零使用 |
| radial-gradient / linear-gradient | `rg` | ✅ 零使用 |
| 非标准字重 (650/620) | `rg font-weight:.*(650\|620)` | ✅ 零使用 |
| console.log | `rg` | ✅ 零命中 |
| 架构边界 (business → system) | `rg` | ✅ 前后端均无跨层导入 |
| `any` 类型 | `rg ": any\|: any\[\|as any"` | ✅ 仅 1 处 (`exporter.ts`) |
| RTL 逻辑属性 | `rg padding-inline\|margin-inline\|text-align:.*start` | ✅ 已有 14 处使用 `padding-inline` |

## 发现项

### P0

**[P0-1] localStorage 存储 access_token / refresh_token**
- 严重程度：10/10
- 违反章节：§3.7.8 前端存储安全
- 影响：XSS 攻击可直接读取 token，绕过整个认证体系
- 位置：
  - `frontend/src/api/request.ts:48-58` — token 读写在 localStorage
  - `frontend/src/store/useAuthStore.ts:32-43` — Zustand 同步读写 localStorage
  - `frontend/src/api/file.ts:15` — 文件下载直接读 localStorage token
  - `frontend/src/core/refresh/refreshBus.ts:102,118` — 刷新总线读取 token
- 建议：迁移到 httpOnly cookie + CSRF token 模式，或短期记录为首批 P0 整改项

### P1

**[P1-1] `format:check` 脚本不存在**
- 严重程度：9/10
- 违反章节：§4.1 固定验证命令
- 影响：Prettier 格式化检查无法执行，`CODE_REVIEW_STANDARD.md` §4.1 引用不可用命令
- 修复：`package.json` 添加 `"format:check": "prettier --check \"src/**/*.{ts,tsx,css,json}\""`

**[P1-2] ja-JP/ko-KR/fr-FR 存在 4 个 extra key**
- 严重程度：8/10
- 违反章节：§3.5.5 翻译覆盖率
- 影响：`business.cmdb.host.permission.create/delete/update/view` 在三个 locale 中为 extra key，zh-CN/en-US 中不存在
- 修复：删除 extra key 或补全 zh-CN/en-US 对应项

**[P1-3] same-as-en 翻译未本地化**
- 严重程度：7/10
- 违反章节：§3.5.5 翻译覆盖率
- 影响：
  - ja-JP: 1 条 (`business.cmdb.vendor.type.idc` 显示英文)
  - ko-KR: 1 条 (同上)
  - fr-FR: 6 条 (vendor type 系列 + securityEvent + permissionWorkbench remediationAction)
- 修复：补充对应语言翻译或显式标记为 `sameAsEn` 允许项

**[P1-4] `npm audit` 不可用**
- 严重程度：6/10
- 违反章节：§3.7.9 依赖安全
- 影响：npmmirror 不支持 security advisories API，无法自动检查前端依赖漏洞
- 修复：审计时临时切回 `registry.npmjs.org` 或接入 `snyk`/`socket.dev`

### P2

**[P2-1] 6 个文件超过 300 行单文件阈值**
- 严重程度：8/10
- 违反章节：§3.10.3 组件拆分粒度
- 文件列表：

| 文件 | 行数 |
| :--- | :--- |
| `src/modules/system/i18n/I18nList.tsx` | 1813 |
| `src/core/layout/index.tsx` | 1642 |
| `src/modules/system/dept/DeptList.tsx` | 1577 |
| `src/modules/generator/pages/ModuleWizard.tsx` | 1572 |
| `src/modules/system/permission/PermissionList.tsx` | 1247 |
| `src/modules/system/dict/DictPage.tsx` | 1159 |

- 影响：历史遗留，非本次新增，职责集中导致评审和修改困难
- 建议：记录到技术债清单，逐步按职责拆分

**[P2-2] RTL 适配仍是早期阶段**
- 严重程度：5/10
- 违反章节：§3.5.4 RTL 适配
- 影响：`padding-inline` 已有 14 处使用，但整体 CSS 仍大量使用方向硬编码
- 状态：Phase 级别检查项，当前阶段符合预期，不阻塞

## 汇总

| 严重程度 | 数量 | 项目 |
| :--- | :--- | :--- |
| P0 | 1 | token localStorage 存储 |
| P1 | 4 | format:check 缺失、i18n extra key、same-as-en、npm audit 不可用 |
| P2 | 2 | 大文件拆分、RTL 渐进式 |

## 修复执行记录

| 发现项 | 状态 | 修复内容 |
| :--- | :--- | :--- |
| P0-1 | 技术债 | localStorage → httpOnly cookie 排期到独立分支 `feat(auth): migrate token to httpOnly cookie` |
| P1-1 | ✅ 已修复 | 安装 prettier + 添加 `format:check` / `format` 脚本 + `.prettierrc` + `.prettierignore` |
| P1-2 | ✅ 已修复 | 补全 zh-CN/en-US 中缺失的 `business.cmdb.host.permission.*` 四个 key |
| P1-3 | ✅ 已修复 | fr-FR `remediationAction` 改为 `Action corrective`；其余 5 条为法语同源词，无需修改 |
| P1-4 | ✅ 已修复 | 添加 `npm run audit` 脚本使用 npmjs.org registry `.npmrc` |
| P2-1 | 技术债 | 记录到技术债清单，后续逐步拆分 |
| P2-2 | 不阻塞 | Phase 级别，渐进推进 |
