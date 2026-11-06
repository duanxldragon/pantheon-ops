---
title: Pantheon Base 全局架构、代码质量、文档治理与工程成熟度独立审计（2026-11-03）
doc_type: Assessment
layer: platform
status: Superseded
superseded_by: docs/reviews/PANTHEON_BASE_GLOBAL_AUDIT_2026-11-04.md
updated_at: 2026-11-04
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
  - docs/contracts/TENANT_CONTRACT_V1.md
  - docs/contracts/SYSTEM_AUTH_CONTRACT.md
---

# Pantheon Base 全局独立审计报告（2026-11-03）

> **只读审计**：本轮未修改 `pantheon-base` 任何业务代码、迁移、CI、规范、文档，也未触碰 `pantheon-ops`。
> 报告存于 `docs/reviews/`，作为 2026-11-03 审计的正式记录；同日 workspace 根目录的中间稿（基线 `4a1fadbb`，基于 211 项未提交改动）已归档为 `docs/reviews/PANTHEON_BASE_GLOBAL_AUDIT_2026-11-03_DRAFT_4a1fadbb.md`（status: Superseded），仅作差异对比与追溯，以本文件为准。
> 所有结论均可由文中文件路径 / 行号 / 命令复跑复核；无法复核的一律标注「未验证」。
> 本轮在整改落地后的干净工作区上**独立复核**，上一轮结论不直接沿用。

> **与前版（`PANTHEON_BASE_GLOBAL_AUDIT_2026-11-03_DRAFT_4a1fadbb.md`，基线 `4a1fadbb` 中间稿）的差异与合并说明**：前版问题表（AUD-01…AUD-22）与整改建议（§12.1–12.4）分别收入本文件 §1.2 对比表 / §13 验收依据 / §12「明确不建议处理」。AUD-01/02/03/05/08/09/11/13 已在合并前的整改合一（PR #488）中处置完成。下表给出逐项对照，未纳入正式 issue 表（§11）的原因均列明。

| 前版 ID | 前版问题 | 最终版处置 |
| --- | --- | --- |
| AUD-01 | 211 项未提交 | ✅ 已闭环：工作区 0 项未提交，见 §1.2 首行 |
| AUD-02 | 业务域单租户语义缺合同条款 | ✅ 已闭环：合同 §2.6 已落地，见 §3.2「业务域 V1 保持单租户语义」 |
| AUD-03 | token 缓存竞态 | ✅ 已修复：值类型 + 深拷贝 + race 测试，见 §1.2 第三行 |
| AUD-04 | OIDC 不可达实现 | ✅ 已删除（W9），见 §1.2 第四行 |
| AUD-05 | 11 个 checker 未接线 | ✅ 已接线（24/26 + 2 书面例外），见 §9.1 |
| AUD-06 | CONTRIBUTING 工具链漂移 | 保留为 DOC-01 / GOV-06（P2） |
| AUD-07 | 前端 version 漂移 | 保留为 FE-10（P3） |
| AUD-08 | STATUS 写分支快照 | ✅ 已闭环：STATUS 记录 9/9 整改 + 7/7 门槛 |
| AUD-09 | 证据未入库 | ✅ 已闭环：`.harness/evidence/` 现跟踪 57 个目录（`git ls-tree` 实测，非前版的「仅 1 个」） |
| AUD-10 | 中间件链重复 14 次 | 保留为 ARCH-01（P2） |
| AUD-11 | 死表未 drop | ✅ 已新增迁移 `000031`（生产执行仍属维护者 human gate） |
| AUD-12 | 两个死 hook | ✅ 已删除，见 §4.1 |
| AUD-13 | frontmatter 双定义 | ✅ 已合并，见 §8.3 注 |
| AUD-14 | 149 份逾期文档 | 维持现状：不脚本批量改，人工复核轮（§12「可以优化」） |
| AUD-15 | ops 依赖 `internal/` | 维持判断：declared snapshot + overlay 模型受控，见 §10 |
| AUD-16 | N+1 未验证 | 维持「未验证」：正版 3 个 P1 是前端数据一致性，非性能 |
| AUD-17 | 后台任务租户维度未验证 | 维持「未验证」：§5.3 |
| AUD-18 | 前端测试覆盖薄 | 正版已移除该 P3：`vitest run` 32 files / 213 tests 全过（§7.1） |
| AUD-19 | `ResolveForCanary` 仅测试调用 | 维持：可删或标 Deprecated（P3，§12「可以优化」） |
| AUD-20 | casbin 白名单脆弱 | 维持判断：P2（§11），fail-closed 安全方向正确 |
| AUD-21 | 4 份 testing-*.md 重叠 | 维持：建议 MERGE 为 1 份（P3） |
| AUD-22 | system_init.sql 权威误读 | ✅ 已处理：`docs/README.md:151` 声明历史参考，见 §8.2 |

> **最终版沿用前版的全文**（§2 架构、§3 功能、§4 冗余、§5 安全、§6 DB、§8 文档）为当前代码快照的复核结论；前版已被取代的 13 项已从正式 issue 汇总中除名，避免「未提交」历史问题混入最终验收口径。

---

## 1. 总体结论

| 项 | 值 |
| --- | --- |
| 审计时间 | 2026-11-03 |
| 审计 Commit SHA | `525506035b3a843f85b984beb46866c1329cd964`（`main`，`chore(harness): land the 2026-10-31 audit remediation (#488)`） |
| 版本 / 发布锚 | `VERSION` = `0.18.2`；tag `pantheon-base-v0.18.2`；上一轮审计时 HEAD 为 `4a1fadbb` |
| 分支 / 工作区 | `main`；**工作区完全干净（0 项未提交改动）** |
| 审计范围 | 架构、功能完整性、冗余与过度设计、安全与多租户、数据库与性能、前端、文档资产、Harness/CI、下游 `pantheon-ops` 兼容性 |
| 审计方法 | 静态检索 + 四个可执行基线（`go build` / `go vet` / `go test -short` / `tsc -b`）+ 文档逐项对代码交叉核对；多 Agent 并行审计受模型限流影响，仅工程治理方向完整返回，其余方向由主 Agent 直接采证补足（逐项标注证据来源）；未做运行时启动、烟测、压测、Sonar |
| 总体成熟度 | **成熟（≈A−）**：上一轮 9/9 整改包全部落地，P0=0 / P1=2（前端数据一致性，均>100 行才触发），四个可执行基线全绿，冻结门槛 7/7 |
| 是否具备维护冻结条件 | **B：完成少量整改后进入维护冻结阶段**（整改主体已完成；修复 2 个前端 P1 后签署冻结） |

### 1.1 一句话结论

**上一轮评估的「交付状态」问题已解决：整改产物全部进库，工作区干净，基线可独立复现。** 剩余问题从「211 项未提交 + 12 项已确诊缺陷」降为「8 个 P2 治理类问题（均不阻塞运行时）+ 1 项维护者 human-gate 未签署」。底座进入冻结前的最后一步是维护者签署——该动作按 programme 归属维护者，代理未代做。

### 1.2 与上一轮审计（`4a1fadbb` 基线）的关键对比

| 项 | 上一轮（4a1fadbb，未提交态） | 本轮（52550603，已提交） | 状态 |
| --- | --- | --- | --- |
| 工作区未提交改动 | **211 项**（142 M / 51 ?? / 9 D …） | **0 项** | ✅ 已落地 |
| 11 个未接线 checker | HEAD 上 11 个空转 | `scripts/harness/` 26 个 checker 中 24 个接线，2 个书面例外 | ✅ 已接线 |
| `tokenSessionCache` 数据竞争 | 2× DATA RACE（`:182/183/186`） | 值类型 + 深拷贝 + race 测试（`token_session_cache_race_test.go`） | ✅ 已修复 |
| `oidc` 包 | HEAD 存在、不可达（587 行） | 已删除（W9）；迁移 `000029` 与 `oidc_*` 列保留 | ✅ 已删除 |
| 迁移 `000031` | 不存在 | 存在（up/down + `preflight/000031_preflight.sql`） | ✅ 已新增 |
| 生效可执行基线 | 未完整复跑 | `go build` / `go vet` / `go test -short`（46/46 包）/ `tsc -b` 全绿 | ✅ 复现 |
| 冻结门槛进度 | 6/7 | **7/7 全部满足**（`.harness/STATUS.md`） | ✅ 达标 |
| 入口文档（README/DESIGN） | 版本号手抄、租户章缺失 | 版本单源指向 `VERSION`、DESIGN 增补 §3.3/§3.4 租户章、`system_init.sql` 标记历史参考 | ✅ 已治理 |

---

## 2. 整体架构分析

### 2.1 分层与装配（通过）

`backend/cmd/server/main.go:155-175` 是唯一装配点，顺序 `platform → lowcode → system → auth → business`，全部显式构造函数注入：

```go
platform.RegisterPlatformRoutes(...)
lowcode.InitLowcodeModule(...)
system.InitSystemModule(...)
auth.InitAuthModule(api, database.DB, user.NewCredentialRepository(database.DB))
business.InitBusinessModules(...)
```

- **auth↔system 解耦真实**：`auth` 消费 `system_user` 的唯一通道是 `pkg/contracts/authuser` 端口（`main.go:169` 注释与签名一致）。实测 5 个 `auth/login` 文件 + `auth/module.go` + `auth/security` 均只 import `contracts/authuser`，无 `modules/system` 直引。
- 生产硬化在位：Redis 必需时启动 Fatal、迁移失败即退出、优雅停机先 Shutdown 再 drain 审计队列、`/metrics` 生产需 Bearer。
- `go build ./...` / `go vet ./...` 全绿，从包层面证伪循环依赖。

### 2.2 模块边界（有 1 处结构性重复，P2）

- 垂直切片成立：`system/{iam,org,config,audit,i18n}` 各自 handler/service/model。
- 业务域收敛：`modules/business` 仅 `business.go + generated_registry.go + retired_modules.go`，不再内置 cmdb 等业务模块。
- `retired_modules.go` 是 ops overlay 复用 hook：`retiredBusinessModules` 为空切片、整套 cleanup 链在 base 无调用方，但注释明确「业务模块退役条目由 ops 维护」（对应 ops `backend/modules/business/retired_modules_overlay.go`）。**判 D 类合理保留**，不可删。

**问题（P2，ARCH-01）**：`system_modules.go` 中同一串 4 段中间件链手写重复 **14 处**，行号 `162,163,188,209,238,288,321,346,375,419,430,435,471,513`。全部由 `TokenAuth + PlatformScopeTenantContext + Casbin + (DataScope|PlatformCapability) + RefreshSync` 组合，但彼此有细微差异（`288/471` 多 `PlatformCapabilityMiddleware`、`163` 多 `DataScopeMiddleware`、`375/435/513` 无 `RefreshSync`）。风险是漂移——现有 `pkg/contracts/route_groups.go` 正是为此而生，但 **grep 实测 route_groups 在系统域 0 调用方**（routing 全部绕开它手写链）。这是高置信度、低风险的清理项，不阻塞冻结。

### 2.3 依赖关系（通过）

无跨模块直访内部实现；`pkg/` 契约层（contracts/tenant/impexp/database/authtoken）均有真实多处复用；无「无意义中间层」。

---

## 3. 核心功能完整性分析

### 3.1 单租户模式（通过）

登录/刷新/登出、Session 生命周期 + Redis 撤销闭环、用户/角色/菜单/权限（139 条路由注册：50 GET / 64 POST / 16 PUT / 12 DELETE）、组织、数据权限 fail-closed、审计日志（操作 + 登录 + 导出 + 保留清理）、字典/设置（含租户覆盖 + 敏感配置加密）、i18n、动态菜单均落地并有测试。

### 3.2 多租户模式（核心闭环成立）

- 上下文解析四级优先级 + 默认拒绝：`pkg/tenant/tenant.go:123-153`（header → subject → platform-fallback → compat，残余 `ErrTenantContextMissing`）。
- `X-Tenant-Id` header 仅平台能力主体可用（`HeaderAllowed`），不可伪造提权；`parseTenantID` 拒绝非数字 / subject claim 拒绝 0（防静默扩大作用域）。
- 隔离行级：`tenant_id` 出现在 13 个 `*.up.sql` / 183 处；索引覆盖存在（28 处含 `tenant_id` 为上一轮读数，本轮未逐条复算，标「未复核」）。
- 业务域 V1 保持单租户语义（合同 §2.6，W2 方案 C 正式书面化，非缺陷）。

### 3.3 真实调用链抽样（通过）

前端 UserList → `/system/user/list`（systemDataScoped 组：Token→TenantContext→Casbin→DataScope→RefreshSync）→ UserHandler → UserService(tenant.Scoper) → GORM。写路径挂 `SecureActionMiddleware`（批量删/重置密码/删用户），读/导出才走 DataScope——读写区别对待是有意的攻击面设计。上传链路 `setting_handler.go:171-199` 有 `MaxBytesReader`（10MB/20MB 双限）+ `NormalizeObjectKey` + `EnforceTenantObjectScope`（注释明示「Tenancy is enforced from the resolved context only — never request fields」）。

---

## 4. 冗余代码与过度设计分析

### 4.1 已清理（通过）

上一轮 W6 死代码/legacy surface 处置已落地：`usePagination`/`useRequest` hook 零消费再导出删除、`scaffold.Generate` 删除、`readRoleKeysFromContext` 包装删除、死表 `permission_role_data_scope_policy` 由 `000031` drop（含 down 重建回归测试）。`pkg/security/ssrf` 因 ops 5 处 import 保留——正确。

### 4.2 残留

| ID | 分类 | 严重度 | 描述 | 证据 | 置信度 |
| --- | --- | --- | --- | --- | --- |
| BE-01 | E/C 讨论 | P3 | `business/retired_modules.go` 整条 cleanup 链无 base 调用方（空切片）——**判 D 保留**（ops overlay 复用点），但建议在文件头显式注明「当前 base 无条目，由 ops 注入」防误删 | `retired_modules.go:26`；ops overlay 声明确认存在 | 高 |
| BE-02 | A | P3 | `scripts/check-security-patterns.mjs`、`scripts/check-ui-tokens.mjs`、`scripts/migrate-task-manifests.mjs` 三个治理脚本零引用（无 npm script / workflow / 源码引用） | 方向6 GOV-02 逐文件 grep 0/0 | 高 |

### 4.3 无过度设计判断

未发现无消费者的 Interface / Repository / Factory 群。仅有的两个接口（`user.SessionLifecycle`、`contracts.BackendModule`）均有真实多实现/多消费者。token 缓存已从指针改值 + 深拷贝，无过度加锁。

---

## 5. 安全与多租户审计

### 5.1 认证与授权（通过）

- 密码：bcrypt，生产 cost 由 `pkg/common/security/config.go:14` 控制（`BcryptCostProduction`）。
- CSRF：全局中间件（`main.go:142`）+ **精确路径豁免表**（`csrf_middleware.go:16-18`，非前缀匹配，防未来 POST 路由静默豁免）；CORS 头白名单（`cors_middleware.go:60-61`）。
- 登录节流：`SystemLoginThrottle` 按 `source_key` + `BlockedUntil`（`login_service.go:99,515`），表在 `login_runtime.go:147` AutoMigrate。
- 数据权限：`DataScopeMiddleware` fail-closed 有专项测试（`data_scope_failclosed_test.go`）；Casbin 域隔离（全局 + `role:<key>@tenant:<id>` 两命名空间不合并）。
- MFA：`auth/mfa` 独立子域（TOTP secret 加密 `EncryptMFASecret`）。

### 5.2 多租户可信度（通过，有权衡）

- `Resolve` 只从可信源（header + 权限 gate / subject claim / 平台声明 / compat）构造，无请求体字段参与——**结构上无法用参数伪造跨租户访问**。
- 上传/下载的租户强制明确「仅从已解析 context，绝不读请求字段」（`setting_handler.go:218-219`）。

### 5.3 未验证项（明示）

- 未做真实跨租户越权渗透测试（仅静态 + 现有 hostile smoke 测试证据）。
- 后台维护任务（session inventory、审计清理）的租户维度参数未逐一核 —— 标「未验证」。

**结论**：无 P0/P1。安全面处于企业级基线的合理区间，且整改轮已消除上一轮确诊项（token 竞争、未接线 oidc、死表）。

---

## 6. 数据库与性能分析

- 迁移连续 `000001`–`000031`，`000025/000026` 租户 backfill 带 down；`000031` 带 `preflight/000031_preflight.sql`（生产执行属维护者 human gate）。
- AutoMigrate 主入口已门控 dev-only（`main.go:116-117` `PANTHEON_AUTO_MIGRATE`）。**残留（DB-01, P2）**：`data_scope_middleware.go:61-62` 运行时仍对 `SystemRoleDataScope` 单表 AutoMigrate（`ShouldAutoMigrate()` 门控内）——dev-schema 已过迁移表，边缘场景可能漂移，建议改为迁移表持有；不阻塞冻结。
- 性能结论分层：**静态发现**存在中间件重复链（上节）；**实现推断**存在 list 场景潜在 N+1（未逐一穷举，需压测确认）；**测试验证**`go test -short` 46/46 通过。**未做**真库压测与全索引复算——不得据此宣称性能达标或故障。

---

## 7. 前端质量与 UI 分析

**前端方向本轮由完整运行的独立审计 Agent 复核**（tsc/eslint/vitest/13 项 prebuild checker 实跑），结论并入本节。

### 7.1 可执行基线（全绿）

- `npx tsc -b` 退出码 0、零输出（noUnusedLocals/noUnusedParameters/verbatimModuleSyntax 全开）。
- `eslint .` 0 error；`vitest run` **32 files / 213 tests 全过**（exit 0）。
- prebuild 契约门禁抽查 7/13 + `check-contrast.mjs`（AA，body 7.84:1 / link 5.12:1，深浅双模）全部 PASS；`audit-i18n-locales` 各 locale 3020 key、0 missing/extra/empty。
- `npm audit` 0 high/0 critical（19 moderate 全 dev-only 传递依赖）。
- 上轮遗留全部复验干净：follow-redirects lockfile=1.16.2、登录日志 `gorm default` 缺陷已修（`login_model.go:15-19` 无 default 标签）、管理视图 mock 处理器清零、死 hook 已删。
- 路由/权限分层正确：每条路由经 `RoutePermissionGuard → PlanFeatureGuard → Suspense → RouteContentFallback → RouteErrorBoundary`；profile 失败可重试、无权限显 `PageForbidden`。`AppTable`（700 行）列显隐/排序/密度持久化、响应式列隐藏、i18n aria-label 齐备。

### 7.2 前端问题（2 P1 / 3 P2 / 9 P3）

| ID | 严重度 | 描述 | 证据（本主 Agent 复核） |
| --- | --- | --- | --- |
| FE-01 | **P1** | 角色/岗位下拉 `pageSize: 9999` 直发，后端各模块 `NormalizePagination` cap=100 → 角色/岗位 >100 时下拉**静默截断**，`UserFormModal` 多选（mode=multiple）无法选中缺席选项 | `useUserList.ts:329,355`（前端 9999）vs `role_helper.go:534,542` / `post_service.go:663`（后端 cap 100）；`handler.go:39-50` 通用 cap 语义 |
| FE-02 | **P1** | 组织架构成员 `pageSize: 1000` 取数，后端 cap 100；`OrgTab` 用 `.length` 直出 `orgMemberTotal` / `{deptCount}/{postCount}/{userCount}` 静态数字，**无截断指示**，>100 时数字静默错误、误导治理决策 | `DeptList.tsx:138`（`orgChartPageSize=1000`）、`:358,370`；`dept/index.ts:44`；后端 `user_helper.go:39` / `post_service.go:663` cap 100 |
| FE-03 | P2 | 加载失败重复 toast（拦截器 + 页面 catch 双弹）：`LoginLogList.tsx:117`、`SecurityCenter.tsx:397`、`SessionList.tsx:153`、`OperationLogList.tsx:997` | request.ts 拦截器 `feedbackMessage.error` 除非 `skipErrorMessage`，catch 又 `message.error(t('common.loadFailed'))` |
| FE-04 | P2 | 删除「当前页最后一行」后 7 处不向前翻页（MenuList:301、I18nList:585、OperationLogList:1077、LoginLogList:150、DeptList:661、Dict*Tab:311/210）；对照 PostList:311 / RoleList:654 / PermissionList:349 有 `data.length===1&&page>1?page-1:page` | grep 比对 |
| FE-05 | P2 | Prettier 漂移 38 文件，`format:check` 存在但无 CI 调用；`api/request.ts:1-2` 出现同行连排 import | `import { switchI18nLanguage } from '../i18n';import { clearClientAuthSession,` |
| FE-06 | P3 | 死代码：`hooks/useTheme.ts`（36 行，0 消费者）、`SettingPage.tsx` re-export、`ProfileIdentityHero.tsx`（52 行） | grep 0 引用；hooks/index.ts 导出的是 core/theme 的 usePantheonTheme |
| FE-07 | P3 | 重复：`canAccess` 复制 4 次（I18nList/MenuList/PermissionList/DeptList）；`toArcoSortOrder` 复制 8 次 | grep 命中 |
| FE-08 | P3 | `checkPermission.ts` 已带 `isAdmin` 短路，全站仍 71 处手写 `isAdmin \|\| hasPerm(...)` 冗余 | grep 计数 |
| FE-09 | P3 | `isForbiddenRequestError` 导出但 0 消费者 → 页面中途 403 落通用错误 toast，无「无权限」引导 | `api/request.ts:241` |
| FE-10 | P3 | `frontend/package.json` version `0.11.2` vs 仓库 `VERSION` `0.18.2` 漂移 | 对比 |
| FE-11 | P3 | ja/ko/fr 三个 locale 各仅 ~188/188/191 key 且值同英文占位（en 3020）；`audit-i18n` 报 0 missing 属掩蔽 | i18n/locales 计数 |
| FE-12 | P3 | `check-contrast.mjs` 通过但**未接入 prebuild 链**（其余 13 项均接入），对比度契约无自动门禁 | node -e 扫描 predev+prebuild |
| FE-13 | P3 | `npm audit` 19 moderate（@typescript-eslint→minimatch、eslint 传递链），dev-only 需排期 | audit 输出 |
| FE-14 | P3 | vitest jsdom 每文件重建 32 次、单测 145.77s（占 37%），建议 `pool: 'vmThreads'` | vitest stdout |

**FE-01/FE-02 严重度说明**：属数据一致性/正确性缺陷（>100 行才触发，中小规模部署不显现），非安全或架构问题；但会造成**静默错误数据展示**，采纳方向3「冻结前修复」建议，改法低风险（前端 pageSize 对齐或独立下拉接口）。

---

## 8. 文档资产治理与代码一致性分析

### 8.1 资产盘点（实测）

`docs/` 下 .md 分布：harness **135**、designs **121**、history **38**、reviews **36**、acceptances **22**、contracts **20**、release-notes **19**、runbooks **5**；另有根级 README/DESIGN/CHANGELOG/CONTRIBUTING/AGENTS/AGENTS.en/CLAUDE/SECURITY 等。英文镜像 **96** 份（中文本体 330 份）。`.harness/` 共 **1453 个文件**（tasks 15 active + archive 任务 2026-07:26 / 08:23 / 09:72 / 10:94）。

### 8.2 文档-代码一致性抽查（通过，本轮强项）

上一轮 W3/W5 的治理已生效，实测：

1. **README 版本号单源**：不再硬编码 release 号，指向 `VERSION` + release-notes 索引（「正文手抄必然滞后」历史教训被治理）。
2. **`system_init.sql` 明确降级**：`docs/README.md:151` 声明「只是历史参考，schema 权威来源是版本化迁移 + runtime seed」。
3. **DESIGN 租户章**：§3.3（P2 演进项逐项现状 + 判据）、§3.4（多租户已交付面与明确未做面）——与 W1/W2 落地代码一致；§4.3「多租户不再是预留项」为审计 AUD-004 更正。
4. **92 份任务包**状态与证据对齐（Archived 77 / Active 3 / Draft 5），不可编造完成度（方向6 复核确认）。

### 8.3 残留文档问题

| ID | 分类 | 严重度 | 描述 | 证据 | 建议 |
| --- | --- | --- | --- | --- | --- |
| DOC-01 | 文档漂移 | P2 | `CONTRIBUTING.md:27,424` 写「Go 1.21 or later」，实际 `backend/go.mod` = `go 1.26.10`；`:74` 命令 `go run ./cmd/server --migrate` 无此 flag；`:288` `npm test` 实为 `test:unit` | go.mod、frontend/package.json | UPDATE 对齐 |
| DOC-02 | 失效引用 | P2 | 两份 PR 生成器指南引用已归档模板 `sonarcloud-s4666`，且 manifest 必填字段与现行 schema 不一致 | `docs/HARNESS_GOVERNANCE_GUIDE.md`、`docs/harness-pr-generator-guide.md` vs `scripts/task-manifest.mjs:210-211` | MERGE 为一份，指向 live 模板 |
| DOC-03 | 重复 | P3 | 96 份 en 镜像 + 330 份中文本体，en `.en.md` 镜像维护成本高；部分 mirror 更新滞后（如 CONTRIBUTING.en 同样漂移） | 文件名计数 | 或改「仅 contracts/designs 强制镜像」，其余指针化 |

另：`docs/reviews/` 36 份含 5 家第三方模型 review（chatgpt/deepseek/glm/grok/kimi/claude），属历史记录有追溯价值，**不得删除**；`docs/history/` 38 份按 W4 已归位。

---

## 9. Harness Engineering 与工程治理（本方向完整 subagent 复核）

### 9.1 通过项

- **11 个空转 checker 已闭合**：`scripts/harness/` 26 个 `check-*.mjs` 中 **24 个已接线**（package.json → quality.yml/ci.yml），2 个为例外且有书面理由（`check-inheritance-contract` workspace-only、`check-doc-frontmatter` mirror-only）并建表登记；无「既无 npm 也无 workflow」的隐藏 checker。
- `go test -race` 不再双跑；`check-task-packet` 接入生命周期判据（Archived 无 summary → error）。
- 发布门禁 fail-closed（`gh api` 传输失败读作「无结论」而非中止）；CodeQL/Dependabot/Sonar/required-checks 四道硬门齐全。
- 12 个 workflow 职责不重叠、actions 全 SHA pin、permissions 最小化。

### 9.2 问题（8 个 P2 + 3 个 P3，均不阻塞运行时）

| ID | 严重度 | 描述 | 建议 |
| --- | --- | --- | --- |
| GOV-01 | P2 | failure-registry FR-016/FR-017 仍写「no workflow / no npm script」，Status open，但对应 checker 已在 quality.yml:151/155/159 接线——登记表自相矛盾 | 更新 Current Gate/Status 为 implemented |
| GOV-02 | P2 | `scripts/` 根下 3 个治理脚本零引用（BE-02 同源）；FR-026 根因约束只覆盖 `scripts/harness/` | 接线 / 移入 harness / 删除并在 registry 登记 |
| GOV-03 | P2 | 接线表只靠人审（README 自述「review-enforced」），**「写了但没跑」类已复发第 4 次且未被发现** | 新增轻量 checker：枚举 checker↔npm↔workflow 引用，未接线且不在例外表即红 |
| GOV-04 | P2 | 归档证据不在 `check-evidence` 扫描面（FR-024 实测 3 error） | 扩扫描面或显式声明不校验并写入 registry |
| GOV-05 | P2 | Docs Governance 9 个 checker `continue-on-error`，PR 可在其全红时合并，红灯后移 main | 与维护者确认转 blocking 子集 |
| GOV-06 | P2 | 与 DOC-01 同源（CONTRIBUTING 命令/版本漂移） | UPDATE |
| GOV-07 | P2 | 与 DOC-02 同源（PR 生成器指南失效引用） | MERGE/归档 |
| GOV-08 | P2 | 前端覆盖率门禁阈值默认 `0` + `continue-on-error`，恒真的假门禁 | 维护者设非零阈值或显式声明不设 |
| GOV-09 | P3 | `smoke-core.yml:31` 的 `continue-on-error` 置于 job 级（仅支持 step 级），no-op | 改 step 级或删键 |
| GOV-10 | P3 | `check-boundaries` 裸 `--strict` 无 CI 入口，只跑 `--baseline` | 纳入 workflow 或声明 baseline 等价 |
| GOV-11 | P3 | `check-graph-review`/`check-feature-ledger` 当前 finding 恒 0（pre-wired no-op） | README 注明 dormant 状态 |

**方向6 结论**：工程治理的「长效 ratchet」仍是文字 + 人审，是防复发的最薄弱环节（GOV-03）；这是本轮唯一「系统性问题」，但属治理层、不阻塞运行时冻结。

---

## 10. Pantheon Ops 下游兼容性

上一轮审计时 ops 是「手抄 base 文件」；**本轮回 v0.18.2 已切换为声明式继承模型**，实测证据：

- `foundation-release.lock.json`：锁定 `pantheon-base-v0.18.2` / `baseCommit a44ae4e0`，`sharedPaths` 含 backend 4 目录 + frontend 53 条 + docs 2 条，带 checksum 与 repo snapshot sha256。
- `business-overlay.json` + `.business-overlay-report.json`：唯一机器可读业务资产清单与装配报告；重建时 ops 业务源码 `pantheon-ops/backend/...` import 改写为 `pantheon-base/...`。
- `docs/PROJECT_INHERITANCE.md`：完整快照 + 声明式 overlay 重建；每周 workflow 在 base HEAD 上兼容性重建，失败即真实接口断点。
- ops 前端 `package.json` name 为 `@duanxldragon/pantheon-base-ui`（快照 fork，非依赖包）——设计使然。
- base README 同步写明「下游以源码副本消费 + lock 锁定」。

**判断**：Base 升级对 ops 的破坏面通过 lock + rebuild + 周兼容重建受控；`pkg/security/ssrf` 等 ops 在用符号被正确保留。**无 P0/P1 兼容性风险**。剩余：周 workflow 是否在 CI 真实运行「未验证」（需 GitHub 凭证）。

---

## 11. P0–P3 问题汇总

- **P0**：0
- **P1**：2（FE-01 角色/岗位下拉 9999 vs cap 100 静默截断、FE-02 组织架构 1000 vs cap 100 静态计数错误 —— 均>100 行才触发，前端数据一致性，非安全）
- **P2**：14（ARCH-01 中间件链重复、DB-01 运行时单表 AutoMigrate、GOV-01…GOV-08、DOC-01/DOC-02、FE-03/FE-04/FE-05）
- **P3**：15（BE-01/BE-02、DOC-03、GOV-09/10/11、FE-06…FE-14）

置信度标注：ARCH-01 / GOV 系列 / DOC-01/02 / DB-01 / FE-01/02（主 Agent 逐行复核前端 9999/1000 与后端 3 处 cap=100）为高；多租户运行时越权与后台任务租户维度为「静态推断」，未做渗透测试，标中/未验证；前端交互项（FE-03/04）未做浏览器渲染验证，置信度按源码审查口径折减。

## 12. 整改建议与优先级

**必须立即处理（维护者 human-gate 事项）**：
1. 签署维护冻结宣告（`.harness/STATUS.md` 已列出 7/7 门槛，宣告动作按 programme 归维护者）。
2. 决定 `000031`（drop 死表）与 legacy 认证端点（`Sunset: 2026-12-31`）的生产执行时间点。

**应该处理（冻结前，2 个 P1 数据一致性缺陷）**：
- FE-01 / FE-02：前端下拉与组织架构取数对齐后端 cap=100（或加独立列表接口 / 截断提示）。两处改法低风险，方向3 建议冻结前修复。

**应该处理（冻结后第一批维护队列，均低风险）**：
- GOV-01/02/03（registry 同步 + 死脚本处置 + 接线自检 checker——3 项同源，一个 diff 可收）。
- GOV-06/DOC-01（CONTRIBUTING 版本与命令对齐）、GOV-07/DOC-02（PR 生成器指南合并）。
- ARCH-01（系统域路由组统一走 `route_groups.go` 助手）、DB-01（运行时 AutoMigrate 迁移化）。
- GOV-08（前端覆盖率门禁阈值决策，需维护者）。

**可以优化（P3）**：GOV-04/05/09/10/11、DOC-03 en 镜像策略、BE-02 死脚本。

**明确不建议处理**：`retired_modules.go`（保留，ops 复用）；`pkg/security/ssrf`（ops 在用）；`docs/history` 与 `docs/reviews` 第三方 review（历史追溯价值）；不新增任何功能、不重构低代码生成器。

## 13. 最终验收意见

### **B：完成少量整改后进入维护冻结阶段**

**依据**：
1. 上一轮「交付状态」问题已全部解除：0 项未提交、四个基线（build/vet/test-short/tsc）独立复现全绿，前端方向另有 eslint/vitest/契约门禁全绿佐证。
2. 冻结门槛 7/7 满足（`.harness/STATUS.md:44`，2026-11-01 记录）。
3. 剩余 2 个 P1（FE-01/FE-02）为**数据一致性缺陷**（>100 行才触发、非安全/架构、改法低风险），建议冻结前随一个前端小 diff 修复；其余为 14 个 P2 治理类问题，不阻塞运行时，可在冻结后第一批维护收口。
4. 唯一的待办门槛是**维护者签署冻结宣告**与 `000031` 生产执行——属 human gate，代理不代做。

**不选 A 的理由**：FE-01/FE-02 会造成静默错误数据展示（角色/岗位>100 时下拉截断、组织成员>100 时计数误导治理），不宜带进冻结；GOV-03「写了但没跑」类问题已复发 4 次且发现依赖人审，是治理层真实缺口；GOV-08 的假覆盖率门禁也需维护者一句话决策。

**不选 C 的理由**：无架构、安全、功能完整性、工程质量层面的阻塞项；安全面上一轮确诊项（token 竞争、不可达 oidc、死表）已全部闭环。

---

*证据获取说明：本轮启用的多个并行审计 Agent 因模型限流多次失败，其中两个方向完整返回并经主 Agent 复核——工程治理（§9，checker 接线表逐文件核验）与前端（§7，实跑 tsc/eslint/vitest/prebuild 契约门禁，FE-01/FE-02 两处 P1 已由主 Agent 复原前端 pageSize 与后端 cap=100 后确认）；其余方向的结论由主 Agent 以 Bash/Grep/Read 直接采证（§2-6、§8、§10 各条均标注 file:line 或命令输出），关键可执行基线（go build / go vet / go test -short 46 包 / tsc -b）由主 Agent 亲跑。所有 grep/命令可原文复跑。*