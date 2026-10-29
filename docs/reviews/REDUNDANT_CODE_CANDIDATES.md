# Pantheon Base 冗余代码候选清单

**审计日期**：2026-10-27
**基线**：`main` @ `96ef1d66767cc418447422c6a9d56942935b2b73`（v0.17.6），工作区干净
**配套报告**：[CODEBASE_SIMPLIFICATION_AUDIT.md](./CODEBASE_SIMPLIFICATION_AUDIT.md)（总体审计）、[CODE_SIMPLIFICATION_PLAN.md](./CODE_SIMPLIFICATION_PLAN.md)（分批计划）
**性质**：只读审计产出。本文件中的任何“删除/合并”均为**建议**，本次未执行。

---

## 0. 阅读约定

### 0.1 置信度定义

| 置信度 | 含义 |
|---|---|
| 高 | 已完成全仓静态引用追踪 + 定义处语义确认，且下游 `pantheon-ops` overlay 已核对 |
| 中 | 引用追踪完成，但存在动态调用/反射/模板字符串/公共 API 契约等无法完全排除的可能 |
| 低 | 仅基于命名与局部阅读判断，尚未完成调用链或运行时验证（一律标“待验证”） |

### 0.2 证据口径

- **“0 引用”指**：对 `pantheon-base` 全仓（含测试、fixture、脚本、文档、CI 配置）及 `pantheon-ops/backend/modules/business`（下游 overlay）做符号名检索，命中数仅来自定义文件自身。
- **“外部引用 N”指**：排除定义文件后的命中文件数。
- 下游核对口径：`pantheon-ops` 是 “base 全量拷贝 + `modules/business/**` overlay” 的 consumer 模型，**不存在 Go module 边界**。因此真正需要核对的是 overlay 对 base 符号的引用，而不是 `pantheon-ops/backend` 全目录（后者包含 base 副本，会产生 100% 假阳性）。
- 未执行任何构建/测试/CI/数据库操作。

### 0.3 汇总

| 类别 | 条目数 | 可删除/合并的规模（估算） |
|---|---:|---|
| A 确认死代码与死转发层 | 9 | ≈ 320 行 + 13 个脚本文件 |
| B 后端横向重复实现 | 19 | ≈ 1,600–2,200 行可收敛（新增约 300 行公共实现） |
| C 前端过度导出与投机设计 | 5 | ≈ 150–250 行 |
| D 治理面/文档冗余 | 4 | ≈ 150 个文件（需维护者决策） |
| E 证据不足（待验证） | 6 | 不计入收益 |
| F 已排除的误报 | 14 | 不处理 |

> 行数为**静态估算**，非实测 diff；实际收益以落地后的 `git diff --stat` 为准。

---

## A. 确认死代码与死转发层

### RC-001 `backend/pkg/common/token.go` 整个文件（129 行，34 个导出符号）
- **位置**：`backend/pkg/common/token.go:1-129`
- **类型**：死转发层（YAGNI / pass-through）
- **现状**：整文件是对 `pkg/authtoken` 与 `pkg/common/http` 的重导出 + 同名包装函数，包括 `TokenTypeAccess` 等常量、`TokenPair`/`TokenSessionData`/`OperationTokenData` 类型别名、`SetTokenTTL`、`TokenStoreSession`、`GenerateOperationToken`（已标 deprecated）、`ParseOperationToken` 等约 20 个函数。
- **证据**：
  - 全仓检索 `TokenStoreSession`、`DefaultAccessTokenTTL`、`TokenPair`、`common.TokenType*`、`common.SetTokenTTL`、`GenerateOperationToken`、`ParseOperationToken` → 命中仅限 `token.go` 自身。
  - 业务侧实际调用的是 `pkg/authtoken` 与 `pkg/common/http` 的原始符号（如 `authtoken.StoreSession`），未经过本文件。
  - `pantheon-ops/backend/modules/business` 无任何命中。
- **依赖关系**：`pkg/common/http/cookie.go:89 SetTokenTTL` 的唯一调用方就是本文件；`cookie.go:16-19` 的 `AccessTokenTTL`/`RefreshTokenTTL` 仅作为其内部默认值来源，删本文件后仍被 `cookie.go` 内部使用。
- **建议动作**：删除整文件；同步删除 `cookie.go:89` 的 `SetTokenTTL`（若确认无其他调用方）。
- **预计收益**：-129 行，-34 个公开符号，减少一条与新开发者认知冲突的“第二套 token API”。
- **风险**：低。若下游仓库在 `modules/business` 之外自写代码引用（不在 overlay 内则本审计不可见），会编译失败——属于“显式失败”，不是静默错误。
- **验证方法**：`go build ./...`、`go vet ./...`、`grep -rn "common\.Token" backend/`；在 `pantheon-ops` 执行 `go build ./...`。
- **置信度**：高

### RC-002 `backend/internal/middleware/logging_middleware.go`（39 行）
- **位置**：`backend/internal/middleware/logging_middleware.go:12-39`（`StructuredLoggingMiddleware`）
- **类型**：死代码
- **现状**：定义了一个 Gin 中间件，未被任何路由注册。
- **证据**：全仓检索 `StructuredLoggingMiddleware` → 仅命中注释与函数定义 2 处；`internal/server` 及所有 `*_router.go` 中无注册。
- **依赖关系**：无。日志能力由 `middleware/logger.go`（或等价实现）与 Zap 承担，本文件是并行方案。
- **建议动作**：删除。
- **预计收益**：-39 行；消除“有两个日志中间件，不知道启用了哪个”的歧义。
- **风险**：低。若下游按名字引用（overlay 已核对无命中），编译期即失败。
- **验证方法**：`go build ./...`；`grep -rn "StructuredLoggingMiddleware" .` 应只剩文档或 0 命中。
- **置信度**：高

### RC-003 `backend/pkg/common/error.go:107 ErrorResponse`（导出但无任何引用）
- **位置**：`backend/pkg/common/error.go:106-115`
- **类型**：死代码 / 过度导出
- **现状**：定义 HTTP 错误响应信封结构体，但全仓无构造或使用。
- **证据**：全仓检索 `ErrorResponse` → 命中仅 `error.go` 定义行与注释行；`pkg/common/http/response.go` 内有实际在用的私有同构定义。
- **依赖关系**：无（`IsI18nMessageKey` 在 `error.go:129` 有内部使用，**不能删**，见 RC-004）。
- **建议动作**：删除 `ErrorResponse`。
- **预计收益**：-10 行；消除“公共包与 http 子包各有一套响应信封”的重复认知。
- **风险**：低。
- **验证方法**：`go build ./...`。
- **置信度**：高

### RC-004 `backend/pkg/common/error.go:117 IsI18nMessageKey` 导出但仅内部使用
- **位置**：`backend/pkg/common/error.go:117-127`
- **类型**：过度导出
- **现状**：函数在 `error.go:129` 被 `ResolveErrorMessageKey` 内部调用，外部 0 引用；`pkg/common/http/response.go:29` 另有一份同语义实现。
- **证据**：全仓检索 `IsI18nMessageKey` → 3 处：定义、内部调用、`http/response.go` 的独立实现。
- **依赖关系**：`ResolveErrorMessageKey`（外部有引用，保留）。
- **建议动作**：改为小写 `isI18nMessageKey`；或让 `common` 包直接复用 `http` 包的实现（需评估 import 方向，见 RC-021）。
- **预计收益**：-1 个公开符号，减少 API 面。
- **风险**：低。
- **验证方法**：`go build ./...`。
- **置信度**：高

### RC-005 `backend/pkg/database/redis.go:76 SetEx`（0 调用）
- **位置**：`backend/pkg/database/redis.go:76`
- **类型**：死代码（薄包装）
- **现状**：`SetEx` 包装 `rdb.Set(ctx, key, value, ttl)`，全仓 0 调用。
- **证据**：全仓检索 `\.SetEx\(` → 除定义外 0 命中；`pantheon-ops` overlay 0 命中。
- **依赖关系**：无。
- **建议动作**：删除。
- **预计收益**：-6 行。
- **风险**：极低。
- **验证方法**：`go build ./...`。
- **置信度**：高

### RC-006 `backend/pkg/database/migrate.go:444 RunAutoMigrate`（0 调用）
- **位置**：`backend/pkg/database/migrate.go:442-460`
- **类型**：死代码 / 公共 API 残留
- **现状**：GORM AutoMigrate 入口，项目实际迁移走 `golang-migrate` SQL 文件，此函数无人调用。
- **证据**：全仓与 ops overlay 检索 `RunAutoMigrate` → 仅定义处 + 注释。
- **依赖关系**：无。
- **建议动作**：**待确认后删除**。它位于 `pkg/`（公共能力层），名义上是给下游用的；但实测下游未用。
- **预计收益**：-19 行。
- **风险**：低—中。若维护者视其为“下游可能采用的迁移逃生口”，删除会改变公共 API 面。
- **验证方法**：`go build ./...`；确认 `docs/` 无引用该函数的 runbook。
- **置信度**：中（死码判定高；是否保留属策略问题）

### RC-007 `backend/pkg/rbacbind/rbacbind.go` 单数版本 `InsertRoleMenu` / `InsertUserRole` 导出但仅供包内复用
- **位置**：`backend/pkg/rbacbind/rbacbind.go:49`、`:55`
- **类型**：过度导出（非死码）
- **现状**：两者仅被同文件的 `EnsureRoleMenu`（`:100`）、`EnsureUserRole`（`:116`）调用，包外 0 引用。
- **证据**：全仓检索 → 命中 4 处，均在 `rbacbind.go` 内部；批量版 `InsertRoleMenus`/`InsertUserRoles` 才有外部消费者（`role_service.go:292`、`role_menu.go:67`）。
- **依赖关系**：包内。**不能删除**，否则 `Ensure*` 失效。
- **建议动作**：改名为小写 `insertRoleMenu` / `insertUserRole`（收敛 API 面），**不是删除**。
- **预计收益**：-2 个公开符号。
- **风险**：低。
- **验证方法**：`go build ./...`、`go test ./backend/pkg/rbacbind/...`。
- **置信度**：高

### RC-008 `backend/pkg/contracts/route_groups.go:16 ProtectedGroupWithRedis` 导出但仅包内使用
- **位置**：`backend/pkg/contracts/route_groups.go:13,16`
- **类型**：过度导出
- **现状**：仅被同文件 `ProtectedGroup`（`:13`）调用；overlay 未直接使用（overlay 用的是 `ProtectedGroup` 与 `DataScopedGroup`）。
- **证据**：全仓检索 4 处，均在 `route_groups.go`；ops overlay 0 命中。
- **依赖关系**：`ProtectedGroup` → 有外部消费者（含 ops overlay 1 处）。
- **建议动作**：**保留**（公共契约层，下游可能需要自定义 redis 客户端），或降为未导出。归类为“待维护者决策”，不建议本轮改动。
- **预计收益**：0 行，仅 API 面。
- **风险**：低。
- **验证方法**：`go build ./...`。
- **置信度**：中

### RC-009 前端 `isForbiddenRequestError` 导出但 0 消费者
- **位置**：`frontend/src/api/request.ts:241`
- **类型**：死代码 / 过度导出
- **现状**：导出以区分 403，但全仓（含 `src/index.ts` barrel、ops overlay）无消费者。
- **证据**：全仓检索 `isForbiddenRequestError` → 仅定义行 1 处；`frontend/src/index.ts` 未 re-export。
- **依赖关系**：无。
- **建议动作**：删除（若无人计划用于路由级降级）。
- **预计收益**：-8 行。
- **风险**：低。
- **验证方法**：`npm run build`（Vite 会在类型检查阶段暴露引用）。
- **置信度**：高

### RC-010 前端 10 个孤儿截图脚本 `capture-*.mjs`
- **位置**：`frontend/scripts/capture-*.mjs`（10 个文件）
- **类型**：死脚本
- **现状**：一次性 UI 走查截图脚本，未被 `package.json` 任何 script、未被 `.github/workflows/**`、未被其他脚本引用。
- **证据**：逐个文件名在根 `package.json`、`frontend/package.json`、`scripts/**`、`.github/workflows/**` 中检索 → 0 命中（`check-contrast.mjs` 等门禁脚本另有引用，见 RC-011 与 F 类）。
- **依赖关系**：无。渲染证据由 `.harness/evidence/**` 与 `check-visual-evidence.mjs` 承担。
- **建议动作**：删除，或按 `docs/README.md` “删除优先于归档”原则不保留。
- **预计收益**：-10 个文件。
- **风险**：低—中。若维护者仍用它们做人工走查，删除会丢工具；建议删除前询问一次或移到 `docs/` 指针。
- **验证方法**：`grep -rn "capture-" package.json frontend/package.json .github/ scripts/ frontend/scripts/` 应 0 命中后再删。
- **置信度**：高（引用状态）/ 中（是否仍有手工价值）

### RC-011 前端 3 个孤儿工具脚本
- **位置**：`frontend/scripts/merge-coverage-to-lcov.mjs`、`frontend/scripts/migrate-spacing-tokens.mjs`、`frontend/scripts/run-request-error-utils-test.mjs`
- **类型**：死脚本
- **现状**：均未出现在 `package.json` scripts、CI workflow 或其他脚本调用中。
- **证据**：同名检索 0 命中。注意 `migrate-spacing-tokens.mjs` 是一次性迁移工具，迁移已完成。
- **依赖关系**：无。
- **建议动作**：删除。
- **预计收益**：-3 个文件。
- **风险**：低。
- **验证方法**：同上；`npm run build` / `npm run test` 仍通过。
- **置信度**：高

---

## B. 后端横向重复实现

> 本组**不是死代码**，而是“同一段基础设施逻辑被子系统各自实现一次”。全部有真实调用方，删除会破坏功能；正确动作是**收敛到一处共享实现**，并保留所有校验、租户过滤与错误处理。

### RC-012 租户作用域三件套（`WithTenantContext` / `scoped` / `tenantOwnerID`）在约 11 个 service 中重复
- **位置**：
  - `modules/system/iam/role/role_service.go:27-50`
  - `modules/system/iam/post/post_service.go:29-50`
  - `modules/system/iam/dept/dept_service.go:31-53`
  - `modules/system/iam/user/user_service.go:46-74`（含 `:58`/`:63` 的双重包装）
  - `modules/system/config/dict/dict_service.go:58-82`
  - `modules/system/config/setting/setting_service.go:59-89`
  - `modules/system/iam/permission/permission_service.go:67-83`
  - `modules/system/platform/dashboard/dashboard_service.go:54,349-352`
  - `modules/auth/login/login_service.go:51-58`
  - `modules/auth/security/security_service.go:128`
  - `modules/auth/session/session_service.go:79`
- **类型**：重复实现（近拷贝，必须同步修改）
- **现状**：每个 service 自己实现“从 gin context 取租户 → 构造 `*gorm.DB` 作用域 → 取 owner 租户 ID”。
- **证据**：11 处结构几乎逐行相同，差异只在包名与方法接收者。
- **依赖关系**：`pkg/tenant`（`FromGin`、`WithTenantScope`、`PlatformGlobalTenantID`）为唯一底层实现，可安全作为收敛目标。
- **建议动作**：在 `pkg/tenant` 增加一个内嵌辅助（如 `tenant.ScopedDB`），各 service 改为调用；`user_service.go:58/63` 的冗余双重包装直接去掉。
- **预计收益**：-250～-350 行；未来租户语义变更只需改 1 处。
- **风险**：**高**。这是多租户隔离核心路径，改错会跨租户串数据。必须逐个 service 回归。
- **验证方法**：`go test ./backend/...`；租户隔离专项测试；重点跑 `role_menu_tenant_test.go`、tenant 相关 handler 测试；手工双租户冒烟（list 不越界、write 不串租户）。
- **置信度**：高（重复事实）/ 中（收敛方案需逐个评审）

### RC-013 分页归一化函数约 13 份 + `Offset((page-1)*pageSize).Limit(pageSize)` 约 14 处
- **位置（归一化）**：`role_helper.go:523-539` 与 `:541-557`（**同一模块内两份**）、`post_service.go:690-706`、`permission_service.go:963-979`、`user_helper.go:29-48`、`dict_service.go:1339`、`audit_service.go:378`、`security_service.go:722`、`login_service.go`、`session_service.go:460`、`tenant_ops_view.go:390`、`tenant_handler.go:276`、`tenant_platform_handler.go:187`、`setting_audit.go:162`
- **位置（Offset/Limit）**：`role_service.go:134,766`、`post_service.go:97`、`permission_service.go:226`、`user_service.go:266`、`dict_service.go:602`、`audit_service.go:96`、`security_service.go:229`、`session_service.go:504`、`setting_audit.go:180`、`login_service.go:286`、`tenant_ops_view.go:437`、`tenant_service.go:1005,1039`
- **类型**：重复实现
- **现状**：每处各自处理默认值、上限、非法值回退；默认值/上限常量不统一。
- **证据**：函数体 95% 相同；`role_helper.go` 内已存在两份并存。
- **依赖关系**：无外部契约；返回值直接进 GORM。
- **建议动作**：抽到 `pkg/common`（或 `pkg/pagination`）单一 `Normalize(page, pageSize, max)`；`role_helper.go` 的两份先合并。
- **预计收益**：-150～-220 行；分页语义（如最大 pageSize）统一，避免“同一 API 不同模块上限不同”。
- **风险**：中。若各模块上限本就该不同，抽公共函数需带参数化上限，不能硬编码一个值。
- **验证方法**：`go test ./backend/...`；补/跑分页边界测试（page=0、pageSize=0、超大值、负数）。
- **置信度**：高

### RC-014 路径 ID 解析 `strconv.ParseUint(c.Param("id"), 10, 64)` + 400 响应约 35 处内联
- **位置**：`role_handler.go:157,201,222,245,268,292`；`tenant_handler.go:103,157,187,215,245,251,298`；`dict_handler.go:68,83,171,186,226`；`i18n_handler.go:250,295,326`；`dept_handler.go:73,111,161`；`menu_handler.go:79,97`；`permission_handler.go:182,198`；`post_handler.go:72,105`；`audit_handler.go:67,83`；`tenant_platform_handler.go:40,229`；`login_handler.go:406`
- **类型**：重复实现
- **现状**：`user_handler.go:283-286` 已抽出 `parseUintParam`，其余模块仍内联；每处都重复“解析失败 → 返回 400 + 错误码”的 6 行。
- **证据**：逐文件计数 35 处；形态一致。
- **依赖关系**：`pkg/common` 的错误响应构造（`Fail`/`FailWithCode`）。
- **建议动作**：把 `user_handler.go` 的 `parseUintParam` 提升到 `pkg/common`（或 handler 共享包），全量替换。
- **预计收益**：-250～-300 行。
- **风险**：低—中。需注意少数 handler 的错误文案/错误码不同（如租户相关），替换时要保持各自响应体不变。
- **验证方法**：`go test ./backend/...`；handler 层 400 用例；`check:*` 门禁。
- **置信度**：高

### RC-015 排序字段白名单助手 6–8 份
- **位置**：`role_helper.go:559+`、`post_service.go:708+`、`permission_service.go:984+`、`user_helper.go:50+`、`menu_service.go:597-630`，以及 dept / dict / audit / security 的对应变体
- **类型**：重复实现
- **现状**：每份都是 “map 白名单 → 若不在则回退默认列” 的拷贝，允许列与默认列不同。
- **证据**：结构同构，差异仅在列名集合。
- **依赖关系**：无。
- **建议动作**：抽 `NormalizeSort(input string, allowed map[string]string, def string)`，各模块只传自己的列集合。**不要**统一列名集合——列名是业务差异。
- **预计收益**：-80～-120 行。
- **风险**：低。
- **验证方法**：排序参数注入测试（非法列名必须被拒绝）。
- **置信度**：高

### RC-016 导入 / 导出 / 模板下载样板在约 8 个模块重复
- **位置（handler 样板）**：`dept_handler.go:229-257`、`post_handler.go:156-184`、`role_handler.go:117-145`、`user_handler.go:114-142`、`dict_handler.go:298-325,352-379`、`permission_handler.go:251-279`、`i18n_handler.go`、`audit_handler.go`、`login_handler.go:482-518`
- **位置（模板构造）**：`dept_export.go:89 BuildDeptImportTemplate`、`post_service.go:336 BuildPostImportTemplate`、`role_export.go:92 BuildRoleImportTemplate`、`user_export.go:71 BuildUserImportTemplate`、`dict_service.go:340 BuildDictTypeImportTemplate`、`dict_service.go:694 BuildDictItemImportTemplate`
- **类型**：重复实现
- **现状**：每个模块重复“接收文件 → 校验扩展名/大小 → 解析 → 逐行校验 → 汇总成功/失败 → 生成模板 CSV/XLSX”。
- **证据**：handler 段约 28 行 × 8 处；模板构造器各自拼表头。
- **依赖关系**：`pkg/upload`、导入导出公共工具（`frontend` 侧亦有对应 `importExport`）。
- **建议动作**：抽“导入管线”到 `pkg/impexp`（保留每模块的 row mapper 与错误文案）；模板构造抽为 `BuildTemplate(columns)`。
- **预计收益**：-350～-500 行。
- **风险**：**中—高**。导入导出涉及文件类型校验、CSV 注入防护、编码处理、行级错误聚合；收敛时任何一项丢失都是安全/数据问题。
- **验证方法**：`docs/acceptances/SYSTEM_IMPORT_EXPORT_SMOKE_GUIDE.md` 全流程；各模块导入导出测试；CSV 注入用例；`check:*`。
- **置信度**：高（重复事实）/ 中（收敛风险）

### RC-017 `errParamInvalid` / `errRequestFailed` 在 11 个 handler 文件中重复声明
- **位置**：`menu_handler.go:12-13`、`role_handler.go:16-17`、`dept_handler.go:13-14`、`post_handler.go:13-14`、`dict_handler.go:14-15`、`setting_handler.go:22-23`、`audit_handler.go:13-14`、`permission_handler.go:13-14`、`tenant_platform_handler.go:13-14`、`i18n_handler.go:16`、`user_handler.go:13`
- **类型**：重复实现
- **现状**：每文件两行 `var errX = common.NewBadRequest(...)` / `common.NewInternal(...)`。
- **证据**：变量名与构造函数完全一致。
- **依赖关系**：`pkg/common` 构造函数。
- **建议动作**：在 `pkg/common`（或 handler 共享包）导出 `ErrParamInvalid` / `ErrRequestFailed`，各文件删除本地声明。
- **预计收益**：-20 行，但**收益主要是消除错误码漂移**。
- **风险**：低。需确认 11 处构造参数（i18n key、code）确实一致；不一致的保留本地。
- **验证方法**：`go build ./...`；错误码断言测试。
- **置信度**：高（重复）/ 中（是否完全等价需逐个比对参数）

### RC-018 `svc` / `boundService` handler 包装结构体重复 7 份
- **位置**：`role_handler.go:34-36`、`dept_handler.go:29-31`、`post_handler.go:29-31`、`dict_handler.go:28-30`、`setting_handler.go:42-44`、`user_handler.go:28-30`、`permission_handler.go:41-43`
- **类型**：重复实现（小）
- **现状**：每模块定义一个持 service 指针的 handler struct 及其构造。
- **证据**：7 处同构。
- **依赖关系**：无。
- **建议动作**：**保留**。泛型化收益小、可读性下降（Ponytail：不为消除 3 行而引入泛型）。仅记录，不列入执行批次。
- **预计收益**：≈0。
- **风险**：—
- **验证方法**：—
- **置信度**：高（判定为“可接受重复”）

### RC-019 `pkg/common` 包内自重复，且模块内还有第三份
- **位置**：
  - `common/error.go:139-151 GetUserID` vs `common/http/response.go:115-127 GetUserID`
  - `common/http/response.go:29,61-90`（`IsI18nMessageKey` + `ResolveErrorMessageKey`）vs `common/error.go:117-136`
  - `menu_handler.go:110-124 getRoleKeysFromContext` vs `common.GetRoleKeys`（`error.go:154-174`）
  - `user_handler.go:274-281 getUserIDFromContext` vs `common.GetUserID`
- **类型**：重复实现
- **现状**：同一语义最多存在 3 份实现。
- **证据**：函数体逐行比对一致。
- **依赖关系**：均被各自模块使用，**不能直接删**，需先替换调用点。
- **建议动作**：保留 `common` 顶层一份为唯一实现（或反向：保留 `http` 子包一份，`common` 委托）；模块内私有副本改调公共版。
- **预计收益**：-60～-80 行；消除“同一 context key 取法有细微差异”的隐患。
- **风险**：中。context key 与 fallback 行为必须逐一核对。
- **验证方法**：`go test ./backend/...`；登录/鉴权相关测试；handler 权限用例。
- **置信度**：高

### RC-020 错误 → HTTP 映射在各 handler 漂移
- **位置**：`permission_handler.go:20-27 permissionServiceErrorCode`（函数体就是 `return common.CodeForError(err)`）、`role_handler.go:46-55 roleErrorResponse`、`menu_handler.go:126-167`（`menuErrorCode` / `menuErrorMessage` / `isMenuValidationError`）
- **类型**：重复/漂移实现
- **现状**：`common.CodeForError` 已是统一映射，但至少三个模块各自又包了一层；其中 `permissionServiceErrorCode` 是纯 pass-through。
- **证据**：`permission_handler.go:20-27` 只有一行有效逻辑；`role`/`menu` 在其上叠加了模块特判。
- **依赖关系**：`pkg/common` 错误码体系。
- **建议动作**：删除纯 pass-through；保留确有模块特判的逻辑，并在特判处加一行注释说明为什么不能走统一映射。
- **预计收益**：-30～-50 行。
- **风险**：中。删错会导致前端收到不同错误码，破坏既有 UI 分支。
- **验证方法**：各模块错误码断言测试；前端 `request.ts` 错误分支冒烟。
- **置信度**：中—高（`permission` 明确；`role`/`menu` 需逐个用例确认）

### RC-021 `dept` 与 `post` 约 80% 同构
- **位置**：`modules/system/iam/dept/**`（`dept_service.go`、`dept_handler.go`、`dept_tree.go`、`dept_import.go`、`dept_export.go`）与 `modules/system/iam/post/**`（`post_service.go`、`post_handler.go`）
- **类型**：重复实现（骨架级）
- **现状**：组织结构体 + CRUD + 导入导出 + 排序 + 分页 + 错误映射几乎一一对应；dept 多一棵树。
- **证据**：函数名与函数体高度对应。
- **依赖关系**：各自的路由、权限码、菜单、i18n 已独立注册。
- **建议动作**：**本轮不合并模块**。只抽取上文的公共助手（分页/ID 解析/导入管线）。合并需重新设计树与组织语义，风险远大于收益。
- **预计收益**：间接收益已计入 RC-013/014/016。
- **风险**：高（若强行合并）。
- **验证方法**：—
- **置信度**：高（判定为“保留结构，抽取叶子”）

### RC-022 `role_data_scope_policy` 镜像模型
- **位置**：`role_service.go:65-78`（注释明确写 “mirrors … to avoid cross-directory import”），删除路径同时写两张表 `:614-628`
- **类型**：重复模型（刻意的）
- **现状**：为避开跨目录 import 复制了一份 GORM 模型。
- **证据**：字段一一对应；写入逻辑成对出现。
- **依赖关系**：两张表均在 schema 中，迁移已存在。
- **建议动作**：**待验证/暂不处理**。需先确认两模型是否已有字段漂移（若有，说明是隐患而非单纯重复）；修复方向是把模型上移到 `pkg/model` 或允许单向 import，而不是删表。
- **预计收益**：-20 行（若可行）。
- **风险**：**高**。涉及数据表与权限范围。
- **验证方法**：`golang-migrate` schema 对比；数据范围策略测试。
- **置信度**：中

### RC-023 批量状态 / 批量删除 handler 重复 6 处
- **位置**：`role_handler.go:171-198`、`dept_handler.go:125-184`、`post_handler.go:86-128`、`dict_handler.go:95-119,198-222`、`user_handler.go:229-256`、`permission_handler.go:211-224`
- **类型**：重复实现
- **现状**：解析 ID 数组 → 校验非空 → 调 service → 返回成功/失败计数，结构一致。
- **证据**：`common.BatchDeleteResp` / `BatchDeleteFailure` 已是共享返回类型，说明设计意图本就统一，但 handler 侧未收敛。
- **依赖关系**：`pkg/common/batch.go`。
- **建议动作**：抽 `handleBatchAction(c, fn)` 到 handler 共享包。
- **预计收益**：-100～-140 行。
- **风险**：中。各模块的权限校验与“部分失败”语义需逐一对齐。
- **验证方法**：批量接口测试（含部分失败、空数组、越权 ID）。
- **置信度**：高

### RC-024 `gorm.ErrRecordNotFound` 处理 30+ 份
- **位置（示例）**：`dict_service.go:988`、`dept_service.go:417,530-533`、`dept_tree.go:153`、`dept_import.go:164-168`、`i18n_handler.go:257,310,332`、`i18n_seed.go:238,476`、`tenant_service.go:233,322,337,588,892,953,1069`、`session_service.go:359`、`login_service.go:124,163,524,564`、`mfa_service.go:101,225,274`、`dynamic_module_lifecycle.go:58,87`、`generator_datasource_service.go:81,248`、`user_helper.go:311`、`user_service.go:326`
- **类型**：重复实现
- **现状**：`if errors.Is(err, gorm.ErrRecordNotFound) { return nil, common.NewNotFound(...) }` 反复出现。
- **证据**：形态一致，差异仅在 i18n key。
- **依赖关系**：`pkg/common` 错误构造。
- **建议动作**：抽 `common.NotFoundIfMissing(err, key)`；保留各自 key。
- **预计收益**：-150～-200 行。
- **风险**：低—中。少数位置需要不同 HTTP 状态或不同日志行为，替换时逐个核对。
- **验证方法**：`go test ./backend/...`；404 用例。
- **置信度**：高（重复）/ 中（等价性）

### RC-025 租户 handler 直接写裸 `c.JSON`
- **位置**：`tenant_handler.go:90,115,142,174,198,232,262,310,347`
- **类型**：响应信封不一致
- **现状**：其余模块走 `common.Success/Fail`，租户模块直接 `c.JSON`，导致响应结构/错误码与其他模块不一致。
- **证据**：9 处直接 `c.JSON`；同文件其他分支又用 `common.*`。
- **依赖关系**：无。
- **建议动作**：改为 `common.Success` / `common.Fail`。**这是正确性问题优先于精简**（前端需分两条解析路径）。
- **预计收益**：-40～-60 行，且消除契约不一致。
- **风险**：中。改动响应体会影响现有前端调用方，需同步核对 `frontend/src/modules/**/tenant*`。
- **验证方法**：租户 API 契约测试；前端租户页冒烟。
- **置信度**：高

### RC-026 OIDC handler 裸 `c.JSON`
- **位置**：`modules/auth/oidc/handler.go:28,46,55,63,88,94`
- **类型**：同上
- **现状**：6 处直接写响应。
- **证据**：无 `common.*` 调用。
- **依赖关系**：OIDC 流程可能有第三方回调期望特定结构，**必须逐条核对**。
- **建议动作**：待验证后再统一；OIDC 协议响应（如 token endpoint）**不应**套业务信封。
- **预计收益**：0～-30 行。
- **风险**：**高**（协议兼容）。
- **验证方法**：OIDC 登录端到端；与 IdP 的联调日志。
- **置信度**：低（标“待验证”）

### RC-027 登录模块重复租户作用域与分页
- **位置**：`login_service.go:51-58,273,286`、`session_service.go:79,460,504`、`security_service.go:128,208-229,722`
- **类型**：重复实现
- **现状**：与 RC-012 / RC-013 同类，只是位于 `modules/auth`。
- **证据**：同一 trio 与分页逻辑第三、四份拷贝。
- **依赖关系**：同上。
- **建议动作**：纳入 RC-012 / RC-013 的统一收敛，**单独不处理**。
- **预计收益**：已计入。
- **风险**：高（认证路径）。
- **验证方法**：登录/刷新/登出/会话列表回归。
- **置信度**：高（重复）/ 中（收敛）

### RC-028 `setting_audit.go` 内联分页
- **位置**：`setting_audit.go:162,180`
- **类型**：重复实现
- **建议动作**：并入 RC-013。
- **预计收益**：已计入。
- **风险**：低。
- **验证方法**：设置审计列表分页用例。
- **置信度**：高

### RC-029 `permissionServiceErrorCode` 纯 pass-through
- **位置**：`permission_handler.go:20-27`
- **类型**：无价值包装
- **现状**：函数体只有 `return common.CodeForError(err)`。
- **证据**：逐行阅读；全仓唯一调用点在同类 handler。
- **依赖关系**：`common.CodeForError`。
- **建议动作**：删除函数，调用点直达 `common.CodeForError`。
- **预计收益**：-8 行。
- **风险**：低。
- **验证方法**：`go build ./...`；权限错误码测试。
- **置信度**：高

### RC-030 `user_service.go:58/63` 冗余双重包装
- **位置**：`user_service.go:46-74`（`:58` 与 `:63` 两层 `scoped` 包装）
- **类型**：无价值包装
- **现状**：同一查询被套了两层作用域构造。
- **证据**：局部阅读；无额外过滤语义。
- **依赖关系**：租户作用域核心（RC-012）。
- **建议动作**：去掉内层；并入 RC-012 收敛。
- **预计收益**：-5 行。
- **风险**：中（必须确认两层不是有意叠加的租户 + 部门范围）。
- **验证方法**：用户列表跨租户/跨部门用例。
- **置信度**：中

---

## C. 前端过度导出与投机设计

### RC-031 `core/router/types.ts` dashboard widget 类型族
- **位置**：`frontend/src/core/router/types.ts`
- **类型**：投机设计（类型面）
- **现状**：定义了一组 dashboard widget 相关类型，实际只在同文件的联合类型中被引用，无独立消费者。
- **证据**：逐符号 importer 计数 → 0 外部引用；仅在本文件 union 中出现。
- **依赖关系**：`index.ts` barrel 未导出该族。
- **建议动作**：**保留但收窄**（不导出/合并为内联联合），或确认是否有 roadmap 计划。当前不构成运行时代码负担。
- **预计收益**：-30～-60 行（纯类型）。
- **风险**：低。
- **验证方法**：`npm run typecheck` / `npm run build`。
- **置信度**：中

### RC-032 `usePlanEntitlements` 返回类型 `PlanEntitlements` 仅内部使用
- **位置**：`frontend/src/core/entitlements/usePlanEntitlements.ts`
- **类型**：过度导出
- **现状**：类型导出但无外部引用。
- **证据**：importer 计数 0。
- **建议动作**：改未导出或保留（类型导出成本极低）。
- **预计收益**：≈0。**建议不处理**。
- **风险**：—
- **验证方法**：—
- **置信度**：高

### RC-033 `core/shellState.ts` / `core/setting/publicSettings.ts` 的导出面大于消费面
- **位置**：`frontend/src/core/shellState.ts`、`frontend/src/core/setting/publicSettings.ts`
- **类型**：过度导出
- **现状**：多个导出仅被同文件内部使用（storage key、`clearShellLockState`、`applyPublicSettings`、`getPublicSettingsSnapshot`）。
- **证据**：全仓 importer 计数；`main.tsx` 只调用其中少数入口。
- **依赖关系**：`App` / `main` 启动链。
- **建议动作**：**保留**。这是模块入口的合理出口，收窄会降低可测试性。记录为“可接受”。
- **预计收益**：≈0。
- **风险**：—
- **验证方法**：—
- **置信度**：高

### RC-034 `components/data-display/tablePreferences.ts` 导出面
- **位置**：`frontend/src/components/data-display/tablePreferences.ts`
- **类型**：过度导出
- **现状**：仅被 `AppTable.tsx` 引用，全部导出在该文件内使用。
- **证据**：importer 计数 1。
- **建议动作**：**保留**（同目录拆分是合理组织）。
- **预计收益**：≈0。
- **风险**：—
- **验证方法**：—
- **置信度**：高

### RC-035 `frontend/src/index.ts` barrel 的导出面
- **位置**：`frontend/src/index.ts:1-40`
- **类型**：公共入口（慎改）
- **现状**：barrel 导出 router/store/modules/components/hooks/api/format/runtime/App。
- **证据**：ops overlay 实际消费：`defineModule`(16)、`usePermission`(7)、`apiRequest`(7)、`uploadImportFile`(2)、`downloadFile`(2)、`FormSection`，以及 `App`。其余为 base 内部或未来消费面。
- **依赖关系**：**这是下游唯一稳定契约面**。
- **建议动作**：**不缩减**。可考虑标注 `@public` 注释，避免后续误删。
- **预计收益**：0。
- **风险**：删任何一项都可能破坏 ops。
- **验证方法**：`pantheon-ops` 前端 `npm run build`。
- **置信度**：高

---

## D. 治理面 / 文档冗余

### RC-036 99 个 `.en.md` 双语副本，未纳入任何机械校验
- **位置**：`docs/**/*.en.md`（99 个）
- **类型**：治理冗余 / 漂移风险
- **现状**：中文主文档与英文副本并存，但无 checker 校验两者内容是否同步。
- **证据**：`docs/README.md` 只在部分行给出 English version 链接；`scripts/harness/check-*` 中无双语一致性检查。
- **依赖关系**：`check-shared-drift.mjs` 只比对 base↔ops，不比对 .md↔.en.md。
- **建议动作**：**维护者决策**。三选一：①删除英文副本；②保留并新增同步 checker；③声明英文副本为“尽力而为、不保证同步”。
- **预计收益**：删除可减 99 文件；保留 + checker 则增加治理成本。
- **风险**：中。若对外发布英文文档，删除会影响用户。
- **验证方法**：`grep -rn "\.en\.md" docs/ scripts/ .github/`；文档站构建（若有）。
- **置信度**：高（现状）/ 低（该删还是该补）

### RC-037 6–7 份 LLM 交叉评审原文未被索引引用
- **位置**：`docs/reviews/pantheon-base-review-{chatgpt,claude,deepseek,glm,grok,kimi}.md`、`docs/reviews/pantheon-base-review-20260910.md`
- **类型**：一次性过程稿
- **现状**：`docs/README.md` §4.1 只索引 3 份正式验收/评审报告，未索引这些 LLM 原文。
- **证据**：`docs/README.md:88-90` 与目录实际内容对比；全仓检索文件名引用 0。
- **依赖关系**：`PANTHEON_BASE_CROSS_REVIEW_REPORT.md` 可能汇总引用它们（需确认）。
- **建议动作**：按 `docs/README.md` §1.1“删除优先于归档”处理；若 `CROSS_REVIEW` 依赖其证据链则保留。
- **预计收益**：-7 文件。
- **风险**：低—中（丢失评审溯源）。
- **验证方法**：检索交叉报告中的相互引用后再决定。
- **置信度**：中

### RC-038 `.harness/archive/` 659 个文件
- **位置**：`.harness/archive/**`
- **类型**：历史归档
- **现状**：占 `.harness/` 1,081 tracked 文件的 61%。
- **证据**：文件计数。
- **依赖关系**：`check-*` 可能扫描 `.harness/tasks|evidence`，未必扫 archive；需确认。
- **建议动作**：**本轮不处理**。属版本库体积问题，非代码问题；且归档有审计价值。
- **预计收益**：仓库体积，非代码行数。
- **风险**：删除会丢证据链。
- **验证方法**：`grep -rn "archive" scripts/harness/*.mjs`。
- **置信度**：高（判定为暂不处理）

### RC-039 `docs/harness/` 与 `.harness/` 并存
- **位置**：`docs/harness/**`、`.harness/**`
- **类型**：疑似重复
- **现状**：**已论证不是重复**。`docs/README.md:78` 明确说明：前者保存可审阅、可版本化的方法/合同/任务包规范；后者只存机器可读状态、证据与运行产物；检查器与交付流程分别依赖。
- **证据**：`docs/README.md:78`；`scripts/harness/check-*` 对两者的不同读取路径。
- **建议动作**：**保留**。记录以阻止后续被误判。
- **预计收益**：0。
- **风险**：合并会破坏门禁。
- **验证方法**：—
- **置信度**：高

---

## E. 证据不足（待验证，不得当作已确认冗余）

| 编号 | 位置 | 现状 | 为什么不删 | 需要的验证 |
|---|---|---|---|---|
| RC-040 | `backend/internal/middleware/rate_limit_presets.go`（`LoginRateLimitMiddleware` / `RefreshTokenRateLimitMiddleware` / `GeneralAPIRateLimitMiddleware`） | base 内未注册，文件注释明确说是给下游 route-level 使用 | 是刻意的下游扩展点，不是死码 | 在 ops overlay 与生产部署配置中确认是否被引用；查 runbook |
| RC-041 | `backend/pkg/capability/capability.go` | 仅被 `modules/system/iam/menu/menu_service.go:5` 引用 | 有 1 个真实消费者；`PlatformCapabilities` 可能经 JSON 暴露给前端 | 追踪该结构是否进入 API 响应；若是则不能删 |
| RC-042 | `backend/pkg/security/ssrf/validator.go` | base 内 0 importer | **ops overlay 使用 5 次**（`argocd_client.go`、`client_factory.go`、`datasource_service.go`、`notification_service.go`） | 已核对，**保留** |
| RC-043 | `backend/pkg/common/re_export.go` | 112 行再导出，12 个符号 0 限定引用 | 可能存在包内裸名使用或下游使用 | 逐符号在 ops overlay 检索；确认后再谈收窄 |
| RC-044 | `backend/internal/scaffold/types.go` 中的 `EnumOption` / `FieldValidation` / `MenuSeedConfig` / `ModuleI18nTranslations` / `ModuleModel` / `PermissionConfig` | 看似 0 直接引用 | 它们是 `ModuleSchema` 字段，经 JSON 反射消费 | 读取 schema 生成链路，确认 JSON tag 使用；**当前判定：不可删** |
| RC-045 | `backend/internal/scaffold/workspace.go` 中 `ComponentLoader` / `InitGenerated` | 看似 0 引用 | 出现在 Go raw-string **代码生成模板**里，是生成产物的一部分 | 生成一次脚手架并编译产物；**当前判定：不可删** |

---

## F. 已排除的误报（不要删）

| 符号 / 文件 | 为什么看起来像死码 | 实际原因 |
|---|---|---|
| `pkg/database/casbin_adapter.go` 的 `SavePolicy` / `RemovePolicy` / `RemoveFilteredPolicy`、`casbin_watcher.go` 的 `SetUpdateCallback` | 包外无直接调用 | 实现 Casbin `persist.Adapter` / watcher 接口，由库反射调用 |
| `pkg/common/batch.go` 的 `BatchDeleteFailure` / `BatchDeleteResp` | 无构造调用 | 是 `BatchDelete` 的返回类型，被 handler 声明使用 |
| `pkg/upload/service.go` 的 `ConfigReader` | 搜索显示仅内部 | `Service`/`NewService`/`readSetting`/`applyMaxFileSize`/`applyAllowedTypes` 的构造参数 |
| `pkg/tenant/membership.go` 的 `SettingFlagReader` | 同上 | `NewSettingModeLoader` 的依赖注入参数 |
| `pkg/tenant/authz.go` 的 `ReadOnlyExportRoute` / `LoginDiscovery` | 外部引用少 | 前者是 `ReadOnlyExportRoutes` 的元素类型，后者是函数返回类型 |
| `pkg/contracts/route_groups.go` 的 `DataScopedGroup` | 疑似无用 | ops overlay 使用 18 次 |
| `pkg/contracts/route_groups.go` 的 `ProtectedGroup` | — | ops overlay 使用 1 次 |
| `frontend/src/core/layout/AppLayout.tsx` | 静态 import 检索 0 | 被 `App.tsx:28` 以 `React.lazy(() => import(...))` 动态导入 |
| `frontend/src/components/feedback/SecondaryVerifyModal.tsx` | 同上 | `App.tsx:34-36` 懒加载，`:184` 使用 |
| `frontend/src/hooks/index.ts` 等 barrel 文件 | importer 检索 0 | 通过目录 import（`from '../hooks'`）消费 |
| `frontend/scripts/check-*.mjs`（13 个） | 不在 npm scripts 中直接出现 | 由 `config/ui-quality-gate.json:160-166` 统一编排 |
| `frontend/scripts/start-smoke-vite.mjs` / `transpile-typescript-files.mjs` | 同上 | 被 smoke 流程调用 |
| `docs/harness/` 与 `.harness/` | 名称近似 | 职责不同，检查器分别依赖（见 RC-039） |
| `menu_handler.go:110-124 getRoleKeysFromContext` | — | **不是误报，是真重复**（见 RC-019） |

---

## G. 统计与后续

| 动作 | 条目 | 合计估算 |
|---|---|---:|
| 可直接删除 | RC-001, 002, 003, 005, 009, 010, 011 | ≈ -260 行，-13 文件 |
| 降级导出（不删） | RC-004, 007, 008 | -4 公开符号 |
| 确认后删除 | RC-006 | -19 行 |
| 合并/收敛 | RC-012～RC-017, 019, 020, 023, 024, 025, 027, 028, 029, 030 | ≈ -1,300～-1,800 行（净） |
| 保留（已论证） | RC-018, 021, 022, 033, 034, 035, 038, 039, 042, 045 | 0 |
| 维护者决策 | RC-036, 037, 040, 041, 043 | 视决策 |
| 待验证 | RC-026, RC-044 及 E 类其余 | 不计收益 |

**净精简估算：约 1,500–2,100 行（占后端生产代码 3–4%），并从根上消除 6 类“必然同步修改”的重复债。**
精确数字以实际落地后的 `git diff --stat` 为准——本文件不声称已实测。

下一步见 [CODE_SIMPLIFICATION_PLAN.md](./CODE_SIMPLIFICATION_PLAN.md)。
