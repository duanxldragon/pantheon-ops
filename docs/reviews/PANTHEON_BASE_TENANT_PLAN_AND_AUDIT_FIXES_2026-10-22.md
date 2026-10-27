---
title: 租户菜单边界、登录审计归属与套餐差异化修复验收（2026-10-22）
doc_type: Acceptance
layer: platform
depends_on_layers:
  - system/auth
  - system/iam
status: Active
linked_contracts:
  - docs/contracts/TENANT_CONTRACT_V1.md
  - docs/contracts/PLATFORM_CONTRACT.md
  - docs/contracts/SYSTEM_AUTH_CONTRACT.md
  - docs/contracts/SYSTEM_IAM_CONTRACT.md
updated_at: 2026-10-22
---

# 租户菜单边界、登录审计归属与套餐差异化修复验收（2026-10-22）

## 0. 结论

- 维护者人工验收提出的三类问题均已复现、修复并在真实 multi 模式环境中回归：
  1. **菜单权限**：租户会话不再收到「租户管理 / 菜单管理 / 国际化管理」；直连受限 URL 渲染 403 页，
     无租户数据加载；平台管理员能力不变。
  2. **功能不全（登录日志为空）**：租户登录成功/失败、MFA 失败与安全事件按会话归属租户落库，
     租户审计页可见自己的记录；compat 仍写平台 0。
  3. **套餐差异化**：免费/基础/专业/旗舰四个版本在前台呈现明确差异；差异由前台注册表驱动，
     模块只声明 feature，后端不按套餐初始化或写死控制台功能。
- **严重度结论**：无未解决 P0/P1/P2/P3；原 P3 标签栏残留已修复并经浏览器探针复验。
- **验证**：后端真实 MySQL + Redis `go test -race -p 1 ./...` exit 0（46 包）；前端 30 文件 /
  214 用例、tsc、ESLint、13 步 prebuild（Arco patch + 12 项 contract）、Vite build、i18n 五语言对齐全部通过；四套餐浏览器验收在登录重置与标签修复后重跑通过。
- **授权边界声明**：套餐门禁是 UX 呈现层，不是授权。后端角色权限与 `PlatformCapabilityMiddleware`
  仍是最终判定；绕过前端直接调用 API 不会因套餐被放行。这是本次设计的显式边界，不是缺口。

## 1. 验收对象与方法

- 仓库 `github.com/duanxldragon/pantheon-base`，基线 main `aef57e30`；本轮为工作区未提交改动。
- 环境：本机 multi 租户模式；真实 MySQL（`.env.test`）、Redis、`go run ./cmd/server`、
  Vite dev server `127.0.0.1:5173`、Chromium headless（Playwright 1440×900）。
- 方法：按维护者描述先做现象复现 → 沿调用链定位归属边界 → 修复 → 单测/真库回归 →
  用四个真实套餐租户做浏览器验收。所有结论以代码、测试输出、HTTP 响应与截图为准。
- 证据目录：`.harness/evidence/2026-10-22-tenant-plan-entitlements-and-audit/`
  （`summary.md`、`review.md`、`commands.json`、`acceptance-result.json`、`artifacts/`）。

## 2. 问题 1：平台专用菜单对租户可见（P1）

### A1 — 租户管理员收到平台管理面菜单并落到 403

- **编号/严重度**：A1 / P1（越权面暴露，非数据泄露）
- **文件:行号**：
  - `backend/pkg/tenant/platform_scope.go:31`（新增分类）
  - `backend/modules/system/iam/tenant/tenant_provision.go:272`、`:490`（初始化/复制时剔除）
  - `backend/modules/system/iam/menu/menu_service.go:73`、`:237`（租户作用域过滤）
  - `backend/modules/system/iam/role/role_helper.go:255`、`:288`（角色写入守卫）
  - `frontend/src/core/layout/useLayoutMenu.tsx:169`、`:202`（渲染兜底）
- **复现**：以平台管理员创建一个租户 → 用该租户 owner 登录 → 侧栏出现「租户管理」（部分环境
  还有「菜单管理」「国际化管理」）→ 点击或直连 `/system/tenant`，页面显示
  「403 你没有访问当前页面的权限」。
- **根因**：
  1. 租户 provisioning 将平台管理员的 `system_role_menu` / `system_role_permission` 逐字复制到
     租户命名空间，租户管理员因此持有 `system.tenant` 菜单与 `system:i18n:*` / `system:menu:create|update|delete`。
  2. 菜单树读取不区分平台/租户作用域，把上述菜单原样返回。
  3. 这些路由组由 `PlatformCapabilityMiddleware` 拒绝，所以「菜单可见、页面 403」。
  4. 既有租户不会被重新 provisioning 覆盖，旧授权会存活；每用户权限缓存也可能继续返回旧键。
- **影响**：租户用户看到无法使用的平台入口；菜单与页面契约不一致；若后续有页面缺少路由级能力门，
  该授权面可能成为实际越权路径。当前审查未发现该页面数据可被读取。
- **修复**：
  - 在 `pkg/tenant` 建立唯一分类：平台专用菜单 = module `system.tenant` 或 path
    `/system/tenant`、`/system/menu`、`/system/i18n`；平台专用权限键 = `system:tenant:*`、
    `system:i18n:*`、`system:menu:create|update|delete`。`system:menu:list` 保持可分配，
    因为租户角色管理需要读取管理树。
  - 租户作用域的导航树与管理树都过滤平台专用行。
  - provisioning 首次初始化与重跑均剔除平台绑定；`Bootstrap()` 启动时幂等修复存量租户，
    仅删除 `tenant_id <> 0` 的行，不触碰平台命名空间。
  - 租户角色写入 API 拒绝平台专用菜单/权限键，避免直连 API 重建入口。
  - 权限缓存命中时，若包含平台键则按用户归属租户过滤并回写；权限查询本身也对租户用户过滤。
  - 前端按租户会话再次过滤平台专用项，兜底网关/过期缓存返回的树。
- **验证**：
  - 浏览器四套餐（free/basic/professional/enterprise）：侧栏 `platformOnlyMenuVisible` 均为空；
    直连 `/system/tenant` 均渲染 403，且无租户列表数据（`acceptance-result.json` 的 `issue1`）。
  - 平台管理员侧栏仍包含「租户管理」「菜单管理」「国际化管理」，可正常进入。
  - 回归测试：`menu_service_test.go`、`tenant_provision_menus_test.go`、`role_menu_tenant_test.go`。

### A2 — 存量授权与权限缓存的残留（P2）

- **编号/严重度**：A2 / P2
- **文件:行号**：
  - `backend/modules/system/iam/tenant/tenant_provision.go:540`、`:549`
  - `backend/modules/system/iam/tenant/tenant_service.go:505`
  - `backend/modules/auth/login/login_runtime.go:644`、`:683`
- **复现**：在修复前已创建的租户上直接登录，仍能看到平台菜单；调用 `/auth/user/perms` 或读取
  权限缓存可见平台键。
- **根因**：provisioning 只补缺失行，不删除多余行；权限缓存 TTL 5 分钟内不会因授权修复而失效。
- **影响**：修复对新租户生效、对存量租户不生效；缓存继续向旧会话暴露旧键。
- **修复**：启动时 `RepairTenantPlatformGrants()` 扫描所有租户命名空间，删除平台专用菜单/权限绑定
  （含平台页面下的按钮行），幂等且按表存在性守卫；缓存命中时仅当列表含平台键才做归属查询，
  过滤后回写。
- **验证**：真库测试覆盖重跑 provisioning 的修复路径；全量 `-race` 通过；浏览器复验存量验收租户
  （创建于修复后）无平台菜单。

### A3 — 直连被禁 URL 后已打开标签残留（P3，已修复）

- **编号/严重度**：A3 / P3（外观）
- **文件:行号**：
  - `frontend/src/core/layout/AppLayout.tsx:1039`、`:1046`、`:1262`、`:1529`
  - `frontend/src/core/layout/layoutTabs.ts:77`
- **复现**：租户管理员在地址栏直接输入 `/system/tenant`，页面 403；顶部已打开标签栏仍出现
  「租户管理」标签，可再次点击（仍 403）。
- **根因**：标签栏在路由跳转时只按路径记录标签，既不检查路由的 `pagePermission`，也不检查套餐
  门禁；权限拒绝发生在页面层，标签层没有回滚依据。
- **影响**：无数据加载、无权限提升、不跨登录边界（登录时清空 shell 状态）；仅是视觉不一致。
- **修复**：标签层统一用 `isPathAccessible` 判定——套餐 `isPathEntitled` 与路由 `pagePermission`
  同时满足才记录/展示；已存在但不满足的标签不进入标签栏与命令面板。套餐上下文解析前允许临时
  记录，解析后派生列表自动收敛。
- **验证**：免费租户登录后直连 `/system/tenant` 与 `/system/generator`，标签栏始终只有「工作台」，
  `tenantTabLeaked=false`、`generatorTabLeaked=false`、403/升级页正常；证据
  `tab-residual-result.json` 与两张截图。

## 3. 问题 2：租户登录日志为空（P1）

### B1 — 审计行写入平台 0，租户视图读不到

- **编号/严重度**：B1 / P1（安全审计不可用）
- **文件:行号**：
  - `backend/modules/auth/login/login_handler.go:180`、`:196`、`:256`
  - `backend/modules/auth/login/login_tenant_audit.go:46`、`:58`
  - `backend/modules/auth/login/login_service.go:89`、`:634`
  - `backend/modules/auth/login/login_runtime.go:223`
  - `backend/modules/auth/mfa/mfa_service.go:163`、`:173`
- **复现**：租户 owner 登录 → 打开「安全审计 / 登录日志」→ 列表为空；平台管理员在同一部署的
  全量视图中能看到该条记录。
- **根因**：
  1. 登录日志/安全事件按请求上下文租户写入；登录请求在解析/签发前上下文为平台 0。
  2. 租户管理员读取时使用 `WithTenantScope(tenant)`，`tenant_id = 0` 的行被过滤。
  3. MFA 在校验码阶段失败时，尚未把请求钉到 challenge 的租户，失败记录同样落到 0。
- **影响**：租户无法审计自己的登录成功/失败与安全事件；平台视图混入所有租户记录；
  安全事件响应缺少租户归属。
- **修复**：
  - 成功登录：按实际签发的 tenant claim 写登录日志。
  - 失败登录：当主体已知时，通过 `tenantAttributionError` 携带归属租户，错误文本与 `errors.Is`
    语义不变；handler 用该租户写日志。
  - MFA：加载 challenge 后把 `req.TenantId` 钉到 challenge 租户；调用方显式选择仍参与 gate 校验；
    challenge 失败也写入正确租户。
  - 安全事件：主体租户优先，无主体时才回退请求上下文。
  - compat：审计行保持平台 0，compat 管理员可读（有回归测试）。
  - 前端：`useTenantSessionStore` 统一 `/auth/tenants/current`，登录后拉取、登出重置；
    收尾审查发现同一标签页重新登录会复用上一主体的上下文，已在 `Login.tsx:248` 的登录边界补
    `resetTenantSession()`，并用 3 个 store 单测覆盖请求去重、重置重载与作用域判定。
- **验证**：
  - 浏览器四套餐登录日志页均显示自己的成功记录（free 7 行、basic 6 行、professional 8 行、
    enterprise 6 行；行数随验收登录次数增加，不作为固定断言）。
  - `login_log_tenant_attribution_test.go` 覆盖：隐式租户成功登录、失败登录 + 安全事件、compat 保持 0；
    每个用例同时以租户作用域读取列表断言可见。
  - 全量 `-race` 通过。

## 4. 问题 3：套餐版本没有功能差异（P1，本轮最重要）

### C1 — 套餐只是字符串，控制台完全一致

- **编号/严重度**：C1 / P1（商业版本不可区分）
- **文件:行号**：
  - `frontend/src/core/entitlements/plans.ts:37`、`:70`、`:90`
  - `frontend/src/core/entitlements/usePlanEntitlements.ts:34`
  - `frontend/src/core/entitlements/PlanFeatureGuard.tsx:59`
  - `frontend/src/core/layout/useLayoutMenu.tsx:212`
  - `frontend/src/App.tsx:171`
  - `frontend/src/modules/lowcode/dynamicmodule/index.ts:17`、`generator/index.ts:69`
  - `frontend/src/modules/platform/widgets.tsx:200`
  - `frontend/src/modules/system/iam/tenant/TenantList.tsx:164`、`:794`
  - `frontend/src/modules/system/iam/tenant/TenantPlanMatrix.tsx:21`
- **复现**：平台管理员分别创建 free/basic/professional/enterprise 租户 → 以各租户 owner 登录 →
  四个控制台的菜单、页面、快捷入口完全相同。
- **根因**：`tenants.plan` 仅作为数据存储（既有配额映射除外），没有任何呈现层定义把套餐映射到
  功能；后端 provisioning 对所有租户复制同一套菜单/权限，导致版本差异无处表达。
- **影响**：付费版本与免费版无可见差异；若靠后端 seed 区分，会产生不可维护的多套初始化数据，
  且与维护者「差异由前台完成」的明确要求冲突。
- **修复（前台单一注册表）**：
  - `PLAN_FEATURES` 定义 feature key → 最低套餐：
    `lowcode.dynamic-module` → basic、`lowcode.generator` → professional、
    `lowcode.generator.datasource` → enterprise。
  - 模块在自己的路由与 dashboard widget 上声明 `feature`；shell 统一过滤菜单、已打开路由与
    dashboard 入口，`PlanFeatureGuard` 为直连 URL 渲染升级页。
  - 平台作用域（tenant 0）与 compat 不启用套餐门禁；未知套餐归一为 free，未知 feature fail-open。
  - 生成器在低于旗舰版时只显示当前数据源，禁用管理按钮并显示「外部数据源需旗舰版套餐」；
    后端数据源权限仍必须满足。
  - 租户新建/编辑弹窗渲染只读对照矩阵（3 功能 × 4 套餐，当前列高亮）；套餐提交值仍只有
    `tenants.plan` 字符串。后端既有配额映射未改动，未新增任何套餐 seed。
- **验证**（浏览器真实租户）：

| 套餐 | 低代码入口 | `/system/modules` | `/system/generator` | 数据源管理 |
|---|---|---|---|---|
| free | 无 | 升级页 | 升级页 | 未暴露 |
| basic | 模块管理 | 正常 | 升级页 | 未暴露 |
| professional | 模块管理 + 生成器 | 正常 | 正常 | 锁定标签 + 按钮禁用 |
| enterprise | 模块管理 + 生成器 | 正常 | 正常 | 无锁定、按钮可用 |

  - 平台管理员新建租户弹窗：3 个功能行、4 个套餐列、当前套餐标签（截图
    `artifacts/tenant-create-matrix.png`）。
  - 前端单测：套餐归一/层级/包含关系/单调性/注册表完整性 6 项；菜单空分组剪枝 2 项；
    dashboard widget feature 门禁 1 项。

### C2 — 生成器数据源选择器未按套餐收敛（P2）

- **编号/严重度**：C2 / P2
- **文件:行号**：`frontend/src/modules/lowcode/generator/pages/ModuleWizard.tsx:760`、`:763`、`:1460`、`:1472`
- **复现**：专业版租户在生成器中选择「从数据库表导入」，可看到并选择非当前租户的数据源，
  管理按钮可用。
- **根因**：数据源列表只受角色权限控制，没有套餐条件。
- **影响**：低版本用户可看到更高版本能力入口；若后端权限配置宽松，会形成功能越级使用。
- **修复**：低于旗舰版时列表只保留 `isCurrent`，管理按钮禁用并显示锁定标签；企业版恢复完整列表。
  后端 `system:generator:datasource:manage` 权限仍是必要条件。
- **验证**：professional 截图显示锁定标签与禁用按钮；enterprise 截图显示可用按钮。

## 5. 验证矩阵

| 范围 | 命令 | 结果 |
|---|---|---|
| 后端全量（真实 MySQL/Redis） | `cd backend && PATH="/d/msys64/mingw64/bin:/d/nodejs:$PATH" CGO_ENABLED=1 CC='D:\msys64\mingw64\bin\gcc.exe' go test -race -p 1 -count=1 ./...` | exit 0；46 包 ok；无 FAIL、无 data race |
| 后端 MySQL 8 严格模式 | `cd backend && go test -count=1 ./modules/system/iam/menu/`（DSN 追加 `sql_mode=ONLY_FULL_GROUP_BY`） | 整包通过；修复 hosted 首轮 Unit Tests 暴露的 Error 3065 |
| 前端单测 | `cd frontend && node ./node_modules/vitest/vitest.mjs run` | 30 文件 / 214 用例通过 |
| 前端类型 | `cd frontend && node ./node_modules/typescript/bin/tsc -b` | exit 0 |
| 前端 lint | `cd frontend && node ./node_modules/eslint/bin/eslint.js .` | exit 0 |
| 前端构建 | `cd frontend && npm run build` | 13 步 prebuild（Arco patch + 12 项 contract）+ Vite build 通过 |
| i18n | 审计脚本 | 2929 key；zh-CN/en-US/ja-JP/ko-KR/fr-FR 缺失 0 |
| 文档门禁 | `node scripts/frontmatter-check.mjs`；`node scripts/harness/check-task-packet.mjs --root .` | 通过 |
| 浏览器验收 | `acceptance.mjs`（四套餐） | 见第 2–4 节；JSON + 截图归档 |

## 6. 环境限制与显式 gap

1. **race 的 CGO 依赖**：Windows 本机 `go test -race` 需要 CGO 与 mingw GCC；未显式设置 PATH/CC 时
   Go runtime 的 `cc1.exe` 会静默失败。上表命令是实际执行的命令，不能假设默认 shell 可直接复现。
2. **并行 race 的假失败**：全量并行运行曾出现 Redis 状态竞争（`TestSessionAdminOps_CompatPinsToPlatformTenant`）
   和 generator 环境变量导致的失败；本轮用 `-p 1` 串行取得确定性结果。hosted CI 的并行 leg 仍是
   最终信号，本报告不把本地串行结果外推为 CI 结论。
3. **npm 递归调用依赖 PATH**：prebuild 脚本内部递归调用 `npm`，本机 cmd.exe 默认找不到 npm；
   实际构建命令为 `PATH="/d/nodejs:$PATH" npm run build`，并另以 `node` 逐条执行同一组 prebuild 步骤复核。
4. **MySQL 8 严格模式**：hosted 首轮 Unit Tests 在新增租户导航用例上返回 Error 3065
   （`DISTINCT system_menu.id ... ORDER BY system_menu.sort` 与 `ONLY_FULL_GROUP_BY` 不兼容）。
   已移除该未被消费的排序，并以相同 `sql_mode` 在本地复跑 menu 整包通过；hosted 复跑结论为准。
5. **浏览器工具限制**：本会话 browser-use MCP 的快照工具被策略拦截，浏览器验收使用 Playwright
   （Node + Chromium）完成；截图与 JSON 是实际运行产物。
4. **浏览器复验范围**：四套餐验收与标签栏探针均针对本机 multi 部署；hosted CI 的并行 `-race`
   腿仍需在 PR 上确认，本地串行结果不作 CI 结论外推。
5. **套餐门禁非授权**：套餐只影响前台呈现；后端权限仍是边界。若未来需要商业级强制（例如按套餐
   拒绝 API），必须新增后端 entitlement 校验，属于新需求，不在本轮范围。
6. **未做生产数据修改**：存量租户授权修复在服务启动时幂等执行；本报告未执行任何生产库写操作。

## 7. 变更文件与证据

- 后端：`pkg/tenant/platform_scope.go`、`modules/system/iam/{menu,tenant,role}/*`、
  `modules/auth/{login,mfa}/*` 及对应测试。
- 前端：`core/entitlements/*`、`core/layout/*`（含标签可达性过滤）、`core/router/types.ts`、`App.tsx`、
  `store/useTenantSessionStore.ts`、`modules/auth/login/components/Login.tsx`、
  `modules/auth/tenant/usePlatformScope.ts`、`modules/lowcode/*`、`modules/platform/*`、
  `modules/system/iam/tenant/*`、五语言资源、单测。
- 任务包：`docs/harness/tasks/2026-10-22-tenant-plan-entitlements-and-audit.task.md`；
  清单 `.harness/tasks/2026-10-22-tenant-plan-entitlements-and-audit/manifest.json`。
- 证据：`.harness/evidence/2026-10-22-tenant-plan-entitlements-and-audit/`。
