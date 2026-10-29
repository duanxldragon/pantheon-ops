# Pantheon Base 全局代码精简与过度设计审计

**审计日期**：2026-10-27
**审计对象**：`pantheon-base`
**基线**：`main` @ `96ef1d66767cc418447422c6a9d56942935b2b73`（v0.17.6 已发布），工作区干净
**审计性质**：只读审计。未修改、删除、重命名任何业务源码、迁移、依赖或脚本。
**工具**：Ponytail 5.3.2（`/ponytail full` + `/ponytail-audit`）、CodeGraph（13,909 节点 / 32,513 边 / 863 文件）、`grep`/`find` 静态引用追踪；下游 `pantheon-ops` 实仓核对。

---

## 1. 审计范围与执行情况

### 1.1 覆盖范围

| 区域 | 覆盖方式 |
|---|---|
| `backend/modules/**`（149 文件 / 40,936 行） | 逐模块调用链、重复模式统计 |
| `backend/pkg/**`（56 文件 / 7,278 行） | 逐包外部引用计数 + 逐导出符号死码扫描 |
| `backend/internal/**`（21 文件 / 3,595 行） | 逐文件外部引用计数 |
| `frontend/src/**`（257 文件 / 68,169 行） | 逐文件 importer 计数 + 导出符号死码扫描 |
| `frontend/scripts/`（55 文件）、`scripts/`（65 文件） | 逐个文件名在 package.json / CI / 其他脚本中的引用追踪 |
| `.github/workflows/`（12 个） | 门禁与脚本的触发映射 |
| `docs/`（356 md / 83,449 行）、`.harness/`（1,081 tracked 文件） | 规模与重复面统计 |
| `pantheon-ops` | 真实依赖面核对（见 §6） |

### 1.2 执行情况说明（必须如实披露）

- Ponytail 已正确加载并执行：`/ponytail-help` → `/ponytail`（full）→ `/ponytail-audit` 全部成功，方法论已应用。
- 五个并行深审任务中，**两个完整返回**（后端模块重复/分层、下游依赖），**三个因 API 余额错误（402）中断**（后端 pkg/internal 死码、前端死码、脚本/CI 冗余）。这三个区域由主审计者**直接补做**，证据来自静态引用追踪（`grep -w` 全仓 + 逐文件 importer 计数），**未使用 Ponytail 子代理**。
- 因此：本报告的结论可复核，但“脚本/CI 冗余”一项**没有运行时验证**（未执行任何 CI 或脚本），仅做引用追踪。

### 1.3 未做的事

- 未运行 `go test`、`npm test`、`playwright` 或任何 CI；未连数据库。
- 未测量运行时内存/CPU/查询次数。
- 未对 `pantheon-ops` 执行构建或测试，仅做静态依赖核对。

---

## 2. 代码规模基线（客观口径）

### 2.1 分账后的规模

| 类别 | 文件数 | 行数 | 说明 |
|---|---:|---:|---|
| 后端生产代码 | 412（非测试 372） | 53,179 | `modules` 40,936 / `pkg` 7,278 / `internal` 3,595 / `cmd` 1,370 |
| 后端测试代码 | 40 | 39,067 | 测试/生产 ≈ 0.73，覆盖率信号良好 |
| 前端 `src/` | 257 | 68,169 | TS/TSX |
| `docs/` | 356 md | 83,449 | 其中 99 个 `.en.md` 双语副本 |
| `.harness/` | 1,081 | — | tasks 58 / evidence 63 / archive 659 |
| `frontend/scripts/` | 55 | — | 13 个 `check:*` 在 prebuild 链，10 个 `capture-*` |
| 根 `scripts/` | 65 | — | `harness/` 26、`foundation-release/` 8 |
| 全仓 tracked 文件 | 2,720 | — | 产品代码 982（backend 485 + frontend 497） |

### 2.2 结构性信号

- **治理面 ≈ 产品面**：`docs/`（441 tracked）+ `.harness/`（1,081）= 1,522 文件，占全仓 56%；产品代码 982 文件，占 36%。
- **门禁密度**：13 项前端 prebuild 契约门禁 + 26 个 `scripts/harness/check-*` + 12 个 CI workflow + 31 个 checker 测试文件。
- **重复率**：SonarCloud 报告整体 2.8%、新代码 1.9%（本次未复测，取自 `.harness/STATUS.md`）。
- **架构分层**：实际是 **2 层**（handler → service 内嵌 `*gorm.DB`），不存在 handler → service → repository 的三层转发。唯一 repository 是 `system/iam/user/auth_repository.go`（`pkg/contracts/authuser` 的端口适配器，有真实消费者）。

---

## 3. 是否存在明显的过度设计或冗余

**结论：存在“横向重复债”，不存在“纵向过度抽象”。**

- **纵向**：没有空转发层。每个 service 都承担校验、事务、树构建或租户过滤。抗辩成立。
- **横向**：同一段基础设施逻辑被复制 11–35 次（tenant-scope、分页、路径 ID 解析、排序白名单、导入导出样板、批量操作、错误映射）。这是 AI 辅助开发最典型的模式：每个模块独立生成一次“正确”的实现。
- **投机性设计**：少量。集中在 `src/core/router/types.ts` 的 dashboard widget 类型族、`internal/middleware/rate_limit_presets.go` 的下游预设、`pkg/common/token.go` 的整层转发。
- **治理膨胀**：显著。文档与 Harness 文件数超过产品代码文件数；双语 `.en.md` 99 份无机器同步校验。

---

## 4. 最重要的 10 项发现

> 完整证据见 `REDUNDANT_CODE_CANDIDATES.md`。严重程度定义见任务书 §8。

### AUDIT-001（P1，无效代码）`pkg/common/token.go` 整个文件无调用方

- **位置**：`backend/pkg/common/token.go:1-129`（129 行，34 个导出符号）
- **当前实现**：一层对 `pkg/authtoken` 与 `pkg/common/http` 的纯转发别名（`TokenStoreSession` → `authtoken.StoreSession`，等）。
- **问题**：全仓（含测试）对 `common.<任一符号>` 的限定引用为 **0**；16 个符号在“含本包裸引用”的口径下也为 0。业务代码一律直接使用 `authtoken.*`（如 `login_runtime.go:1101`、`session_service.go:168`）。
- **证据**：`grep -rn "\bcommon\.<sym>\b"` 全部为 0；`authtoken.AccessTokenTTL` 有引用，`common.AccessTokenTTL` 无。
- **建议**：删除整个文件；连带删除仅被它调用的 `pkg/common/http/cookie.go:89 SetTokenTTL`。
- **收益**：−129 行，−1 个重复令牌 API 面。
- **风险**：低。下游 overlay 未引用（见 §6）。
- **验证**：`go build ./... && go test ./...`。
- **置信度**：高。

### AUDIT-002（P1，重复代码）租户作用域包装三元组在 ~11 个 service 中复制

- **位置**：`role_service.go:27-50`、`post_service.go:29-50`、`dept_service.go:31-53`、`user_service.go:46-74`、`dict_service.go:58-82`、`setting_service.go:59-89`、`permission_service.go:67-83`、`dashboard_service.go:54`、`login_service.go:51-58`、`security_service.go:128`、`session_service.go:79`
- **当前实现**：每个 service 各自定义 `WithTenantContext` + `scoped` + `tenantOwnerID`，内部调用的 `tenant.FromGin`（`pkg/tenant/tenant.go:184`）与 `tenant.WithTenantScope`（`:278`）**已经存在**。
- **问题**：11 份近似逐字复制，注释都相同；`user_service.go:58/63` 还多一层冗余双包装。
- **建议**：在 `pkg/tenant` 提供一个可嵌入的 `TenantScoped` 基类型（或两个自由函数），11 处改为复用。
- **收益**：净减约 250 行，且租户隔离规则只有一处可改。
- **风险**：中。触及租户隔离，必须保持 `tenantOwnerID()` 的 `IsMulti()` 语义逐字不变。
- **验证**：现有租户/数据范围红用例（`-run 'Tenant|DataScope'`）+ 双租户冒烟。
- **置信度**：高。

### AUDIT-003（P1，重复代码）分页归一化 ~13 份、`Offset/Limit` ~14 处

- **位置**：`role_helper.go:523-539` 与 `:541-557`（同一模块两份）、`post_service.go:690-706`、`permission_service.go:963-979`、`user_helper.go:29-48`、`dict_service.go:1339`、`audit_service.go:378`、`security_service.go:722`、`login_service.go`、`session_service.go:460`、`tenant_ops_view.go:390`、`tenant_handler.go:276`、`tenant_platform_handler.go:187`、`setting_audit.go:162`
- **问题**：同样的 `page=1 / pageSize=10 / cap 100 / Offset((page-1)*pageSize)` 重新实现十余次，个别还内联未提取。
- **建议**：`pkg/common` 增加 `NormalizePage(page, pageSize int) (int, int)`，`pkg/database` 增加 `Page()` scope。
- **收益**：净减约 180 行；分页上限只改一处。
- **风险**：低。行为等价，但 `user_helper.go` 的 max-offset 变体需保留。
- **验证**：列表接口回归 + `pagination-contract` 冒烟。
- **置信度**：高。

### AUDIT-004（P1，重复代码）路径参数 ID 解析内联 ~35 处

- **位置**：`role_handler.go:157,201,222,245,268,292`、`tenant_handler.go:103,157,187,215,245,251,298`、`dict_handler.go:68,83,171,186,226`、`i18n_handler.go:250,295,326`、`dept_handler.go:73,111,161`、`menu_handler.go:79,97`、`permission_handler.go:182,198`、`post_handler.go:72,105`、`audit_handler.go:67,83`、`tenant_platform_handler.go:40,229`、`login_handler.go:406`
- **当前实现**：每处 `strconv.ParseUint(c.Param("id"), 10, 64)` + 相同的 `common.Fail(CodeParamInvalid, ...)`。
- **反证**：`user_handler.go:283-286` 已提取 `parseUintParam` 并在本模块复用 4 次 —— 修复模式已被证明可行。
- **建议**：`common.ParsePathID(c, key) (uint64, error)`。
- **收益**：净减约 190 行。
- **风险**：低。
- **验证**：各模块 400 响应断言 + handler 单测。
- **置信度**：高。

### AUDIT-005（P1，重复代码）导入/导出/模板 handler 样板在 ~8 个模块复制

- **位置**：`dept_handler.go:229-257`、`post_handler.go:156-184`、`role_handler.go:117-145`、`user_handler.go:114-142`、`dict_handler.go:298-325/352-379`、`permission_handler.go:251-279`、`i18n_handler.go`、`audit_handler.go`、`login_handler.go:482-518`
- **问题**：同样的 `ShouldBindJSON + impexp.WriteCSV`、`FormFile → Open → impexp.ReadCSV → service`，连中文注释都逐字相同；`pkg/impexp` 已拥有 `ReadCSV/WriteCSV`。
- **建议**：在 `pkg/impexp` 提供 CSV 导入/导出/模板的一对 helper，handler 缩到约 5 行。
- **收益**：净减约 200 行。
- **风险**：中。CSV 注入防护与大小上限必须保留在 helper 内。
- **验证**：`test:smoke:system:api`（import/export 与 batch-delete 两个 spec）。
- **置信度**：高。

### AUDIT-006（P1，重复代码）`errParamInvalid` / `errRequestFailed` 在 11 个 handler 文件重复声明

- **位置**：`menu_handler.go:12-13`、`role_handler.go:16-17`、`dept_handler.go:13-14`、`post_handler.go:13-14`、`dict_handler.go:14-15`、`setting_handler.go:22-23`、`audit_handler.go:13-14`、`permission_handler.go:13-14`、`tenant_platform_handler.go:13-14`、`i18n_handler.go:16`、`user_handler.go:13`
- **建议**：`common.MsgParamInvalid` / `common.MsgRequestFailed` 导出一次。
- **收益**：−22 行，文案改动一处生效。
- **风险**：低。
- **置信度**：高。

### AUDIT-007（P1，重复代码）`common` 包自我重复，且模块再抄第三遍

- **位置**：
  - `common/error.go:139 GetUserID` **与** `common/http/response.go:115 GetUserID` 两份；
  - `common/error.go:117 IsI18nMessageKey` / `:121 ResolveErrorMessageKey` **与** `common/http/response.go:29,61-90` 的私有实现两份；
  - `menu_handler.go:110-124 getRoleKeysFromContext` 重复 `common.GetRoleKeys`（`error.go:154`）；
  - `user_handler.go:274-281 getUserIDFromContext` 重复 `common.GetUserID`。
- **问题**：同一个语义有 2–3 个实现，行为已经可能出现漂移。
- **建议**：以 `pkg/common` 为唯一 canonical，`http` 包 import 它；删除模块本地副本。
- **收益**：−60 行，消除语义漂移源。
- **风险**：低。
- **置信度**：高。

### AUDIT-008（P2，过度设计）`internal/middleware` 两个零引用文件 + 三个未注册预设

- **位置**：`internal/middleware/logging_middleware.go:1-39`（`StructuredLoggingMiddleware`，全仓 0 引用）；`internal/middleware/rate_limit_presets.go:13,24,38`（三个预设，0 引用，注释自承“底座未注册本预设”）。
- **问题**：`cmd/server/main.go:159` 与 `modules/auth/module.go:34-46` 走的是通用 `RateLimiter`，预设从未接线。
- **建议**：删除 `logging_middleware.go`；`rate_limit_presets.go` 要么接线到认证路由，要么降级为文档示例后删除。
- **收益**：−约 70 行。
- **风险**：低。若下游未来要用，`ratelimit.NewRateLimitMiddleware` 仍是公开能力。
- **置信度**：高。

### AUDIT-009（P2，工程治理冗余）10 个 `capture-*.mjs` 为孤立一次性脚本

- **位置**：`frontend/scripts/capture-audit-round2.mjs`、`capture-screenshots.mjs`、`capture-search-toolbar-evidence.mjs`、`capture-security-audit-fixes.mjs`、`capture-security-audit-recheck.mjs`、`capture-toolbar-rollout-evidence.mjs`、`capture-ui-components.mjs`、`capture-ui-details.mjs`、`capture-ui-part2.mjs`、`capture-ui-part3.mjs`
- **证据**：在 `package.json`、`.github/workflows/**`、其他 `.mjs` 中**零引用**（全仓排除自身后无命中）。
- **问题**：历史某次审计的截图工具，已成孤儿；仍计入维护面。
- **建议**：删除，或移到 `.harness/archive/` 作为历史证据工具。
- **收益**：−10 文件。
- **风险**：低。
- **置信度**：高（静态引用层面）。

### AUDIT-010（P2，工程治理冗余）文档/治理面超过产品代码面，双语副本无同步校验

- **位置**：`docs/` 356 md / 83,449 行（99 个 `.en.md`）；`.harness/` 1,081 文件；`docs/reviews/` 内 7 份未被任何流程引用的多模型评审稿（`pantheon-base-review-{claude,chatgpt,deepseek,glm,grok,kimi}.md` + `-20260910`）。
- **证据**：`grep -rl "pantheon-base-review-(claude|chatgpt|...)"` 仅命中 `.harness/archive` 的一个 inventory；双语副本除 `docs/README.en.md` 与 task-packet 模板外，没有同步漂移检查器。
- **问题**：文件数 1,522 > 产品代码 982；`.en.md` 实际是人工维护的第二份真相，长期必然漂移。
- **建议**：① 删除 7 份一次性评审稿；② 明确双语策略：要么保留少数“对外合同”英译并由 checker 校验标题/章节数，要么停止新增 `.en.md`；③ `.harness/archive` 保留（审计链需要），但不再计入日常维护面。
- **收益**：文件数 −7 起；阻止治理面继续线性增长。
- **风险**：低（删除评审稿）；中（双语策略需维护者拍板）。
- **置信度**：高（规模与引用事实）/ 中（取舍建议）。

---

## 5. 按模块划分的精简潜力

| 模块 | 主要问题 | 可精简量级（估算） | 风险 |
|---|---|---:|---|
| `backend/modules/system/iam/role` | 重复 helper 最密集：2 份分页归一化、本地 error mapper、镜像 data-scope 模型、本地 sort 白名单 | 250–400 行 | 中 |
| `backend/modules/system/iam/user` | 重复 tenant-scope、分页变体、`getUserIDFromContext` | 120–200 行 | 中 |
| `backend/modules/system/org/{dept,post}` | 近 80% 同构骨架（handler + service + 导入导出） | 300–500 行 | 中 |
| `backend/modules/system/config/{dict,setting}` | 重复 pagination/sort/import-export/error const | 150–250 行 | 低 |
| `backend/modules/auth/{login,session,security}` | 重复 tenant-scope 与分页（跨 auth/system 边界） | 150–250 行 | 中 |
| `backend/modules/system/iam/tenant` + `tenant_ops` | 原始 `c.JSON` 绕过统一信封；分页重复 | 100–180 行 | 中 |
| `backend/pkg/common` | `token.go` 整文件、error/http 双份、batch 类型过度导出 | 200–250 行 | 低 |
| `backend/internal/middleware` | 零引用文件与未注册预设 | 70 行 | 低 |
| `frontend/src/core/router/types.ts` | dashboard widget 类型族为推测性设计，无实现消费 | 80–120 行（类型） | 低 |
| `frontend/scripts` | 10 个孤立 capture 脚本 + 2–3 个孤立工具 | 12–13 文件 | 低 |
| `docs/` + `.harness/` | 一次性评审稿、双语副本策略 | 文件数而非行数 | 低–中 |

**可安全精简合计（估算）**：后端约 **1,000–1,600 行**（净减，已扣除新增共享 helper 的约 250 行），前端约 **80–120 行类型 + 13 个脚本文件**。

> 这是估算，不是已验证删除量。逐项证据与验证方式见另两份报告。

---

## 6. 可以安全精简的内容（含下游约束）

### 6.1 下游依赖的真实形态（关键前提）

`pantheon-ops` **不是** base 的外部依赖消费者，而是 **base 全量源码副本 + 业务 overlay**：

- `pantheon-ops/backend/go.mod` 自身声明 `module github.com/duanxldragon/pantheon-base/backend`，**没有 `require` base，也没有 `replace`**。
- `pantheon-ops/frontend/package.json` 自身就是 `@duanxldragon/pantheon-base-ui`，通过 `exports: {".": "./src/index.ts"}` 自引用。
- 唯一 ops 自有后端代码是 `backend/modules/business/**`；共享树由 `sharedPaths` + `check-shared-drift.mjs` 约束与 base HEAD 一致。

**这意味着**：不存在“外部 `require` 边界”会被删除打断；但“base 内零引用”不等于“下游零使用”，因为 ops 的 overlay 可能引用 base 的任何公开符号。

### 6.2 overlay 的真实契约面（必须保留）

后端（在 `pantheon-ops/backend/modules/business/**` 中的引用）：

| 包 | 引用点 | 典型符号 |
|---|---:|---|
| `pkg/common` | 59 | `Success/Fail/FailWithCode/FailWithError`、`DataScopeReq`、`SetAuditMetadata`、`CodeParamInvalid`、`NormalizeUint`、`BusinessInsert/Update/Delete` |
| `pkg/database` | 19 | `WithDataScope`、`WithCustomTimeout`、`RDB` |
| `pkg/testmysql` | 17（测试） | — |
| `pkg/impexp` | 14 | CSV 辅助 |
| `pkg/contracts` | 8 | `BackendModule`、`FuncModule`、`RegisterBackendModules`、`DataScopedGroup`（18）、`ProtectedGroup`（1） |
| `pkg/security/ssrf` | 5 | `Validator`（3）、`DefaultValidator` |
| `pkg/logging` | 4 | — |

前端 barrel：`defineModule`、`ModuleConfig`、`apiRequest`、`downloadFile`、`uploadImportFile`、`usePermission`、`FormSection`，以及 `@/components`、`@/hooks/usePermission`、`@/api/*`、`@/core/format/dateTime`。

**交叉核对结果**：本次标记的 base 死码候选中，仅 `DataScopedGroup`、`ProtectedGroup`、`ssrf.Validator` 被 ops overlay 使用 → 已从删除清单剔除。其余候选 overlay 引用均为 0。

### 6.3 可安全精简（低风险）

1. `pkg/common/token.go` 整文件 + `common/http.SetTokenTTL`。
2. `internal/middleware/logging_middleware.go` 整文件。
3. `pkg/database.RunAutoMigrate`、`pkg/database.SetEx`（全仓 0 引用）。
4. `pkg/common/error.go` 的 `ErrorResponse` 导出类型、`IsI18nMessageKey` 导出函数（均有私有副本或 0 引用）。
5. `pkg/contracts/route_groups.go` 的 `ProtectedGroupWithRedis`（0 引用；`ProtectedGroup` 保留）。
6. 10 个 `capture-*.mjs`、`migrate-spacing-tokens.mjs`、`merge-coverage-to-lcov.mjs`、`run-request-error-utils-test.mjs`（后者仅被注释提及）。
7. `docs/reviews/` 7 份一次性多模型评审稿。
8. 前端 `isForbiddenRequestError`（`src/api/request.ts:241`，全仓 0 引用）。

---

## 7. 必须保留的复杂度及原因

以下**不是**冗余，不得以“精简”为由删除：

| 能力/复杂度 | 原因 |
|---|---|
| 多租户双模式（compat/multi）与租户谓词 | 合同 §2.3.1；历史 P0 缺陷簇的防线 |
| `pkg/tenant` 的 fail-closed 语义、tombstone 处理 | 已发布合同 V1.3，删除会造成越权/复活 |
| Casbin 适配器 `GormCasbinAdapter` 及其 `SavePolicy/RemovePolicy` | 虽无直接调用，但必须满足 `persist.Adapter` 接口；删除会编译失败或行为退化 |
| `SetUpdateCallback`（casbin watcher） | 实现 `watcher.Watcher` 接口契约 |
| `internal/scaffold` 的 `EnumOption/FieldValidation/MenuSeedConfig/...` | 是 `ModuleSchema` 的 JSON 字段类型，经反射/反序列化消费；静态 grep 的“0 引用”是假阳性 |
| `ComponentLoader` / `InitGenerated`（`scaffold/workspace.go`） | 位于**代码生成模板字符串**内，是生成产物的一部分 |
| `pkg/upload.ConfigReader`、`pkg/tenant.SettingFlagReader` | 包内接口类型，被 `Service`/`ModeLoader` 持有 |
| `RateLimitAllower` | 同文件内被 `RateLimiterConfig` 使用（静态扫描的假阳性） |
| 后端 39,067 行测试 | 租户/权限/竞态回归的唯一防线 |
| 13 项 prebuild 契约门禁 | 防回归的机械闸门；删除会立刻引入视觉/i18n 漂移 |
| `.harness/tasks` + `.harness/evidence` | 发布事实与审计链，删了无法回溯 |
| `.harness/archive`（659 文件） | 历史证据，不参与日常维护，但不应删除 |
| `pkg/security/ssrf` | 下游 ops 的 5 处真实调用 |

---

## 8. 证据不足、不能贸然删除的内容

| 候选 | 为什么不能删 | 下一步 |
|---|---|---|
| `pkg/rbacbind.InsertRoleMenu` / `InsertUserRole` | 全仓 0 直接调用，但与 `InsertRoleMenus/InsertUserRoles` 成对，可能是下游公开 API 的一部分；ops overlay 也未用 | 低价值，标记“待验证”；保留 |
| `pkg/common.BatchDeleteFailure/BatchDeleteResp` | 是 `BatchDelete` 的返回类型，包内使用；只是过度导出 | 仅去 `export` 语义无意义（Go），保留 |
| `internal/middleware/rate_limit_presets.go` | 注释明确“保留导出供下游仓库按用户限流” | 需要维护者决定：接线 / 删除 / 保留为扩展点 |
| `frontend/src/core/router/types.ts` 的 dashboard widget 类型族 | 是 `src/index.ts` 的公开导出（`export *`），可能被下游类型引用 | 先在 ops 全量核对；无命中再删除 |
| `.en.md` 双语副本 | 可能是对外交付承诺 | 需要维护者定双语契约 |
| `frontend/scripts/check-contrast.mjs` | 被 `config/ui-quality-gate.json:163` 的 UI 质量门引用 | 保留；但建议加入 prebuild 或 CI |
| `frontend/scripts/audit-i18n-locales.mjs` 等手工审计工具 | 仅供人工按需运行，不在 CI | 保留，但可合并 |

---

## 9. 推荐的精简顺序与预期收益

### 第一批（低风险、收益明确，可与下次发布一起走）

1. 删除 `pkg/common/token.go` + `common/http.SetTokenTTL`。
2. 删除 `internal/middleware/logging_middleware.go`。
3. 删除 `pkg/database.RunAutoMigrate`、`SetEx`；`ProtectedGroupWithRedis`；`common.ErrorResponse` / `IsI18nMessageKey` 导出。
4. 删除 13 个孤立脚本与 7 份一次性评审稿。
5. 删除前端 `isForbiddenRequestError`。
   **验收**：`go build ./... && go vet ./... && go test ./...`、`npm run build`、`npm run test:unit`。

### 第二批（需要测试与调用链验证）

6. 统一 `parse-id`、分页、排序白名单、错误常量、tenant-scope 三元组（AUDIT-002~004、006、007）。
7. 统一导入/导出样板（AUDIT-005）。
    **验收**：`go test -race ./...` + `test:smoke:system:api` + `pagination-contract` + 双租户红用例。

### 第三批（架构边界/权限/下游契约）

8. 租户 handler 统一响应信封（AUDIT-016/017）。
9. `role_data_scope_policy` 镜像模型上提为共享模型。
10. `dept/post` 同构骨架提取（或代码生成）。
11. 文档双语策略与治理面收敛。
    **验收**：合同变更流程 + 全量 46 包真库测试 + `Tenant Smoke (multi)` + `check-inheritance-contract`。

### 预期收益（估算）

- 后端净减 **约 1,000–1,600 行**（占生产代码 2–3%），但**修改点集中度**显著改善：租户规则、分页上限、CSV 防护各只剩一处。
- 前端净减约 80–120 行类型 + 13 个脚本文件。
- 文档/治理文件数：先 −7，后续取决于双语策略。
- **维护成本**：新增一个列表页时，不再需要复制 5 段样板；新增租户隔离规则时只改一处。

---

## 10. 对发布状态与后续维护的影响

- **当前发布状态健康**：v0.17.6 已发布，Sonar 未解决 issue 0、Duplication Gate 通过、门禁全绿。精简不是“救火”，是“降本”。
- **不建议在发布窗口内做大改**：第三批涉及租户/权限/信封，应单独排期。
- **第一批可以随时做**：纯删除，不改变任何对外行为。
- **风险主要来自“假阳性”**：本审计已用 CodeGraph + 下游 overlay 交叉核对剔除接口实现、JSON 反射结构体和模板字符串内的符号。剩余删除项置信度高。

---

## 11. 最终必须回答的问题

1. **是否存在明显代码冗余？** 是。证据：约 35 处内联 ID 解析、约 13 份分页归一化、约 11 份租户作用域三元组、约 8 份导入导出样板、129 行零调用 `token.go`。
2. **哪些模块过度设计最严重？** `system/iam/role`（重复 helper 最密集）；其次是 `org/dept` 与 `org/post` 的同构骨架。
3. **哪些抽象没有实际价值？** `pkg/common/token.go` 整层转发；`internal/middleware/logging_middleware.go`；`permission_handler.go:20-27` 的纯转发 error code；`user_service.go:63` 的冗余双包装。
4. **哪些可以安全删除？** 见 §6.3 八项，合计约 13 个脚本 + 约 300 行 Go。
5. **哪些可以通过复用合并？** 租户作用域、分页、ID 解析、排序白名单、导入导出、错误常量、common 内部重复（AUDIT-002~007）。
6. **哪些复杂度不可避免？** 租户双模式与隔离、Casbin 适配器接口、生成器 JSON schema、审计与发布证据、权限/i18n/视觉门禁。
7. **哪些冗余来自 AI 辅助开发常见模式？** 恰恰是横向复制：每个模块被单独生成一次“正确”的 helper；以及“为未来下游预留”的预设/类型（`rate_limit_presets`、dashboard widget 类型）。
8. **只做高收益低风险的精简能减多少？** 第一批 + 第二批：后端约 800–1,200 行、13 个脚本文件、7 份文档；维护点（规则副本）从 35/13/11 降到各 1。
9. **哪些改动影响 `pantheon-ops`？** 因 ops 是全量副本，任何 base 内部删除都会随下次 `sharedPaths` 同步进入 ops。已核对 overlay 真实使用面；除 `ssrf`、`contracts.DataScopedGroup/ProtectedGroup`、`pkg/common` 常用符号外，无外部依赖。改动需走 foundation release + `check-shared-drift`。
10. **是否适合冻结功能、进入维护阶段？** **适合，但建议先完成第一、二批再冻结**。当前架构没有系统性过度设计，主要债务是横向重复；趁功能稳定时消除，比在新增功能中继续复制更便宜。

---

## 12. 结论

**LEAN-WITH-DEBT**

- **置信度**：高（规模、重复计数、死码引用、下游依赖均有多条独立证据）；中（行数收益为估算；脚本/CI 未做运行时验证）。
- **主要限制**：三个子审计区域由主审计者静态补做；未运行测试与 CI；行数为静态估算。
- **一句话**：architecturally 健康（两层、边界清晰、测试充足），但基础设施逻辑被复制了 11–35 次，治理文档面已超过产品代码面。不需要重写，需要一次“去重 + 删孤儿”的收敛。

---

*本报告为只读审计产物，未修改任何业务代码。*
