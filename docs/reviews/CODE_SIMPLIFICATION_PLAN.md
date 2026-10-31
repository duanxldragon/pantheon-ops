# Pantheon Base 代码精简实施计划（建议稿）

**制定日期**：2026-10-27
**基线**：`main` @ `96ef1d66767cc418447422c6a9d56942935b2b73`（v0.17.6）
**配套**：[CODEBASE_SIMPLIFICATION_AUDIT.md](./CODEBASE_SIMPLIFICATION_AUDIT.md)、[REDUNDANT_CODE_CANDIDATES.md](./REDUNDANT_CODE_CANDIDATES.md)
**状态**：**已执行**（2026-10-29 收口）。13 个波次全部合并，实测与覆盖率见 §11；仍未关闭的两项维护者门也在 §11.6 列出。

---

## 0. 总则

### 0.1 本计划的边界

- 只做**精简**，不做功能变更、不做架构重写、不动数据库结构、不动迁移文件、不升级依赖。
- 每一个批次都必须**可独立合并、可独立回退**；批次内按模块再拆提交。
- 删除任何符号前，必须完成“业务源码 + 测试 + 脚本 + 文档 + 下游 overlay”五处检索；证据写进 task packet。
- 保留清单（审计报告 §7）中的内容一律不动：鉴权/会话/安全/IAM、租户隔离与兼容、权限边界、审计、i18n、菜单契约、迁移、现有测试与门禁、`frontend/src/index.ts` 公共入口。

### 0.2 角色与流程约束

- 依据 `CLAUDE.md`：Planner/Dispatcher **不得直接修改** `backend/` 与 `frontend/src/` 业务代码，实施须由 Generator 适配器执行，交接记录写入 task packet 与 `.harness/evidence/`。
- 依据 `AGENTS.md`：触碰共享合同、系统域、权限、i18n、继承边界时按 **L2** 走完整治理（task packet + evidence + review + human gate）。
- PR 必须带真实 `.harness/{tasks,evidence}/<task-id>/` 产物；不得使用 `solo-override`（维护者专属）。

### 0.3 批次划分原则

| 批次 | 判据 | 特征 |
|---|---|---|
| 第一批 | 删除后行为**零变化**，编译期即可证明 | 无运行时语义、无公共契约消费者 |
| 第二批 | 需要测试证明行为等价 | 收敛复制逻辑，接口内部重构 |
| 第三批 | 触碰租户/权限/协议/数据库/下游契约 | 需要专项回归与人工验收 |
| 暂不处理 | 收益低、风险高、或属维护者策略决策 | 显式记录，不做 |

### 0.4 度量方法（前后对比）

统一用同一组命令，结果写入 evidence：

```
git ls-files 'backend/**/*.go' | grep -v _test | xargs wc -l | tail -1
git ls-files 'frontend/src/**/*.ts' 'frontend/src/**/*.tsx' | xargs wc -l | tail -1
grep -rn "ParseUint(c.Param(\"id\")" backend | wc -l        # 目标：35 → 0
grep -rn "Offset((page\|Limit(pageSize)" backend | wc -l     # 目标：14 → 1
grep -rln "func.*tenantOwnerID" backend | wc -l              # 目标：11 → 1
grep -rln "var errParamInvalid" backend | wc -l              # 目标：11 → 1
```

外加 SonarCloud duplication（当前 overall 2.8% / new 1.9%）与 `go test ./backend/...` 的包通过数（当前 46 ok / 0 FAIL）。

---

## 第一批：低风险、收益明确（可立即执行）

### F1.1 目标

删除**确认无人使用且不属于下游契约**的实现，行为零变化。

### F1.2 条目

| 序 | 条目 | 动作 | 预计 |
|---|---|---|---|
| 1 | RC-001 `pkg/common/token.go` 整文件 | 删除 + 删除 `pkg/common/http/cookie.go:89 SetTokenTTL` | -129 行 |
| 2 | RC-002 `internal/middleware/logging_middleware.go` | 删除整文件 | -39 行 |
| 3 | RC-003 `pkg/common/error.go:106-115 ErrorResponse` | 删除类型 | -10 行 |
| 4 | RC-005 `pkg/database/redis.go:76 SetEx` | 删除函数 | -6 行 |
| 5 | RC-009 `frontend/src/api/request.ts:241 isForbiddenRequestError` | 删除函数 | -8 行 |
| 6 | RC-029 `permission_handler.go:20-27 permissionServiceErrorCode` | 删除函数，调用点直连 `common.CodeForError` | -8 行 |
| 7 | RC-004 `common.IsI18nMessageKey` | 改为未导出 | API 面 |
| 8 | RC-007 `rbacbind.InsertRoleMenu` / `InsertUserRole` | 改为未导出 | API 面 |
| 9 | RC-010 10 个 `frontend/scripts/capture-*.mjs` | 删除 | -10 文件 |
| 10 | RC-011 3 个孤儿脚本 | 删除 | -3 文件 |

### F1.3 前置条件

- 每个符号在五处检索完毕（backend 源码、backend 测试、frontend、`scripts/`、`pantheon-ops/backend/modules/business`）。
- **RC-001 与 RC-004/007 的差异**：RC-001 是整层死转发；RC-004/007 只是导出面收窄，实现保留。
- RC-010 删除前先问维护者一句“这些截图脚本是否仍用于人工走查”——这是唯一需要人工确认的一项（若维护者要保留，本批其余项不受影响）。

### F1.4 验收标准

1. `go build ./...`、`go vet ./...` 通过。
2. `go test ./backend/...`：**包数与通过数与基线一致**（46 ok，0 FAIL，0 SKIP）。
3. `npm run build`（`frontend`）通过；`npm run typecheck`、`npm run lint` 无新增错误。
4. 五条度量命令中，第 3–5 条保持不变（本批不动重复逻辑）。
5. `git diff --stat` 与预计规模同量级（±30%）。
6. 无任何 `_test.go` 被修改（若测试引用了被删符号，说明该符号不是死码——**回到分析而不是改测试**）。

### F1.5 回归测试要求

- 后端：全量 `go test ./backend/...`。
- 前端：`npm run build` + 现有单元测试。
- **不需要**运行时/数据库验证（无运行时语义变化）。
- 需要一次 `pantheon-ops` 编译验证：`cd pantheon-ops && go build ./... && npm run build`（在 base 发布 foundation release 后）。

### F1.6 回滚

单提交回滚即可；无数据/无配置变更。

### F1.7 预期结果

- 代码：约 **-250 行、-13 个文件、-38 个公开符号**。
- 风险：低。
- 门禁：全部应保持绿色。

---

## 第二批：需测试与调用链验证（收敛内部重复）

### F2.1 目标

把**同一段基础设施逻辑的 N 份拷贝**收敛为 1 份共享实现，同时保持每个模块的错误码、i18n key、权限校验和租户过滤完全不变。

> 关键原则：**只收敛“怎么做”，不统一“做什么”**。列名、错误文案、上限值、业务特判属于各模块差异，通过参数传入，不拉平。

### F2.2 条目与顺序（按依赖排序）

**第 1 步：公共助手先落地（只新增，不改调用点）**

| 序 | 条目 | 新增内容 |
|---|---|---|
| 1 | RC-014 | `pkg/common`（或 handler 共享包）新增 `ParseUintParam`，并以 `user_handler.go:283-286` 为蓝本 |
| 2 | RC-013 | 新增 `NormalizePagination(page, pageSize, max)` |
| 3 | RC-015 | 新增 `NormalizeSort(input, allowed, def)` |
| 4 | RC-024 | 新增 `NotFoundIfMissing(err, key)` |
| 5 | RC-017 | 在公共包定义 `ErrParamInvalid` / `ErrRequestFailed`（先只定义） |

验收：新增代码有单元测试；无调用点改动；全仓测试仍全绿。

**第 2 步：按模块替换（每个模块一个提交）**

| 序 | 条目 | 说明 |
|---|---|---|
| 6 | RC-014 路径 ID 解析（35 处） | 逐 handler 文件替换；**逐处核对原始错误码与文案** |
| 7 | RC-013 分页（13 + 14 处） | 先合并 `role_helper.go:523` 与 `:541` 两份；再替换其余 |
| 8 | RC-015 排序白名单 | 只替换骨架，白名单继续由各模块提供 |
| 9 | RC-024 `ErrRecordNotFound` | 逐处比对 i18n key；key 不同则传入 |
| 10 | RC-017 `errParamInvalid` 等 | 先 diff 11 处构造参数；不一致的保留本地 |
| 11 | RC-019 `common` 包内自重复 | 先决定唯一实现方向（建议顶层 `common` 为唯一实现），再替换 `menu_handler.go:110-124`、`user_handler.go:274-281` |
| 12 | RC-020 错误映射 | `permission` 纯 pass-through 已在一批删除；本步处理 `role`/`menu` 的特判，**保留特判并加注释** |
| 13 | RC-023 批量 handler | 抽 `handleBatchAction`，保留各模块权限与部分失败语义 |
| 14 | RC-028 `setting_audit` 内联分页 | 并入第 7 步 |
| 15 | RC-030 `user_service.go` 双重包装 | 先确认两层不是“租户 + 部门”叠加 |
| 16 | RC-016 导入/导出/模板 | **单独一个子批次**，见下 |
| 17 | RC-006 `RunAutoMigrate` | 确认非下游 API 后删除 |
| 18 | RC-025 租户 handler 裸 `c.JSON` | 并入第 2 步后单独做，需与前端同步 |

**第 3 步：导入导出收敛（RC-016，独立评审）**

- 抽“文件接收 → 类型/大小校验 → 解析 → 行级校验 → 汇总”管线。
- **必须逐项保留**：扩展名与 MIME 校验、大小上限、CSV 公式注入防护、编码处理、行级错误聚合、权限校验。
- 各模块只提供 row mapper 与错误文案。
- 建议单独一个 PR，避免与其它重构混在一起导致回归定位困难。

### F2.3 每步验收标准

1. `go test ./backend/...` 全绿，**包数不减少**。
2. 被改模块新增/更新对应单元测试；至少覆盖：非法输入、边界值、权限失败、部分失败。
3. 度量命令目标：
   - `ParseUint(c.Param("id")` 内联数 35 → **≤2**（保留确有特殊响应的）
   - 分页归一化实现 13 → **1**
   - `tenantOwnerID` 定义 11 → 不变（第三批才动）
   - `errParamInvalid` 声明 11 → **1**
4. SonarCloud：`new_duplication ≤ 3%`（当前 1.9%，不得劣化）。
5. 响应体契约不变：对每个改动过的接口，抓取改造前后一次真实响应做 diff（字段、错误码、i18n key）。

### F2.4 回归测试要求

- 全量 `go test ./backend/...`。
- 相关 handler 的 HTTP 层测试（400/404/403/500 分支）。
- 分页/排序边界：`page=0`、`page=999999`、`pageSize=0`、`pageSize=10000`、非法排序列。
- 导入导出：`docs/acceptances/SYSTEM_IMPORT_EXPORT_SMOKE_GUIDE.md` 全套。
- 前端：`npm run build`；若改动了响应体（RC-025），追加对应页面冒烟。

### F2.5 风险控制

- **一次只收敛一类模式**，不要在同一提交里同时改分页和错误映射。
- 每个模块替换后单独提交，便于 `git bisect`。
- 若某模块的特殊性导致公共助手需要“第三、第四个可选参数”，**停下**——说明它不该被抽，保留本地实现并记录原因。

### F2.6 预期结果

- 代码：约 **-1,200 ～ -1,700 行净减**。
- 重复债：6 类“必须同步修改”的模式降为 1 处。
- 风险：中。
- 预计拆分：公共助手 1 PR + 后端模块替换约 6–8 个 PR + 导入导出 1 PR。

---

## 第三批：架构 / 租户 / 权限 / 数据库 / 下游契约

> 本批**必须**有维护者在三个触点介入（需求澄清、门禁策略、最终验收）。默认不排期，等前两批稳定一个发布周期后再评估。

### F3.1 条目

| 序 | 条目 | 为什么是第三批 |
|---|---|---|
| 1 | RC-012 租户作用域三件套（11 处） | 多租户隔离核心路径，串租户即数据事故 |
| 2 | RC-027 登录/会话/安全模块的租户作用域与分页 | 认证路径，且与 RC-012 同源 |
| 3 | RC-022 `role_data_scope_policy` 镜像模型 | 涉及表结构与权限范围 |
| 4 | RC-026 OIDC handler 裸 JSON | 第三方协议兼容，改了可能断登录 |
| 5 | RC-025 租户 handler 响应信封（若第二批未做完） | 前后端契约 |
| 6 | RC-043 `pkg/common/re_export.go` 收窄 | 公共再导出面，下游可能依赖 |
| 7 | RC-031 dashboard widget 类型族 | 需确认 roadmap |
| 8 | RC-036 `.en.md` 双语策略 | 维护者策略决策 |
| 9 | RC-037 LLM 评审稿去留 | 维护者策略决策 |

### F3.2 前置条件

- 前两批已合并并至少发布一个版本，无回滚。
- 至少有一份**双租户端到端冒烟**脚本可用（当前审计未发现现成的双租户隔离冒烟用例，需先补）。
- 对 RC-012：先写“收敛前行为快照”测试（对同一输入断言当前 SQL/结果），再重构。
- 对 RC-022：先用 `golang-migrate` 导出两张表的 schema 做 diff，确认无字段漂移；**本阶段不删表、不改迁移**。

### F3.3 验收标准

1. 全量 `go test ./backend/...` 全绿。
2. **租户隔离专项**：双租户（平台租户 + 业务租户）下，所有受影响模块的 list/get/update/delete 均不越界；平台全局数据仍可见。
3. **认证专项**：登录、刷新、登出、会话列表、MFA、OIDC 全流程通过。
4. **权限专项**：角色 → 菜单 → 权限点 → 接口权限端到端；越权请求返回 403 且不泄露资源存在性。
5. **下游**：`pantheon-ops` 编译 + 构建 + 其自有 smoke 通过；`check-shared-drift.mjs` 通过。
6. `golang-migrate` 版本与 schema **零变化**（除非另立迁移任务）。
7. SonarCloud 质量门禁绿。

### F3.4 回归测试要求

- 后端全量 + 租户隔离 + 认证 + 权限三组专项。
- 前端：登录/租户切换/权限可见性页面冒烟；`impeccable` 视觉门禁（若触及 UI）。
- 运行时证据：`GET /api/v1/health`、一次真实登录、一次跨租户拒绝，写入 evidence。
- 下游：ops 侧至少一条使用 `DataScopedGroup` 的接口能用。

### F3.5 拒绝条件（出现即停）

- 双租户冒烟无法构造。
- 任何模块的租户过滤依赖 service 内部状态而非请求上下文（说明收敛方案不成立）。
- 下游出现依赖 `re_export.go` 的符号。
- 迁移文件需要改动才能继续。

### F3.6 预期结果

- 代码：约 **-250 ～ -400 行**（主要在 RC-012/027）。
- 真正价值不是行数，而是**把租户隔离语义集中到一处**，降低未来事故面。
- 风险：高。
- 若评估后认为风险/收益不划算，**结论可以是“不执行”**——这是可接受的终局。

---

## 暂不处理（显式记录）

| 条目 | 位置 | 原因 |
|---|---|---|
| RC-018 handler `svc` 包装（7 份） | 各 handler | 泛型化收益 ≈ 0，可读性下降 |
| RC-021 `dept` / `post` 骨架合并 | `modules/system/iam/{dept,post}` | 业务语义不同（树 vs 平铺），合并风险 >> 收益；只抽叶子 |
| RC-032 / RC-033 / RC-034 | 前端类型与模块入口 | 类型导出成本极低；收窄反伤可测试性 |
| RC-035 `frontend/src/index.ts` | 前端公共入口 | **下游唯一稳定契约面，禁止缩减** |
| RC-038 `.harness/archive/` 659 文件 | `.harness/archive/**` | 版本库体积问题，非代码问题；有审计价值 |
| RC-039 `docs/harness/` vs `.harness/` | 两目录 | 已论证职责不同，**不是重复**；合并会破坏门禁 |
| RC-040 `rate_limit_presets.go` | `internal/middleware/` | 注释明确为下游扩展点 |
| RC-041 `pkg/capability` | `pkg/capability/` | 有真实消费者，且可能进入 API 响应 |
| RC-042 `pkg/security/ssrf` | `pkg/security/ssrf/` | **ops overlay 使用 5 次** |
| RC-044 `scaffold/types.go` 字段结构 | `internal/scaffold/` | JSON 反射消费 |
| RC-045 `scaffold/workspace.go` | `internal/scaffold/` | 代码生成模板内容 |
| 全部 Casbin 适配器方法 | `pkg/database/casbin_*.go` | 库接口反射调用 |

---

## 全局验收与回归标准

### A. 每批都必须满足

1. **编译**：`go build ./...`、`go vet ./...`、`npm run build`。
2. **测试**：`go test ./backend/...` 包数与通过数不低于基线（46 ok）。
3. **门禁**：现有 13 项前端 prebuild 契约门禁 + `scripts/harness/check-*` 全绿；SonarCloud 质量门禁绿；`new_duplication ≤ 3%`。
4. **契约**：接口路径、方法、请求/响应字段、错误码、权限码、菜单、i18n key **零变化**（除显式声明的通道）。
5. **证据**：每个批次有 `.harness/tasks/<task-id>/manifest.json` 与 `.harness/evidence/<task-id>/`（命令、输出、diff stat、必要时的响应快照）。
6. **可回退**：单批次回滚不影响其他批次。

### B. 明确不做的事

- 不删测试、不改测试断言来迁就重构。
- 不引入新依赖（含工具链）。
- 不改数据库 schema / 迁移文件 / seed。
- 不改 `frontend/src/index.ts` 的导出集合。
- 不修改下游 `pantheon-ops` 的业务代码（只做兼容性验证）。
- 不使用 `solo-override`。

### C. 建议的排期

| 批次 | 建议窗口 | 依赖 |
|---|---|---|
| 第一批 | 下一次常规迭代，1 个 PR | 无 |
| 第二批 | 第一批发布后，6–9 个 PR，分 2–3 个迭代 | 第一批已合并 |
| 第三批 | 前两批稳定一个发布周期后评估 | 双租户冒烟用例就绪 |

### D. 成功判据

- 净减少约 **1,500–2,100 行**代码与 **38 个公开符号**。
- 消除 6 类横向重复（租户作用域、分页、ID 解析、排序、导入导出、批量操作）。
- **零功能回归、零租户隔离回归、零下游破坏**。
- 若第三批最终判定“不执行”，整体仍算达成目标——精简的目的是降低维护成本，不是追求行数。

---

**声明勘误**：本文件的 §0–§F3 保持为**制定时的方案文本**（含当时的估算与风险判断），不随后续执行回写；
真正的执行结果在 **§11**。2026-10-27 写作时的"未修改任何业务源码"只描述当时状态，
2026-10-29 起该描述已失效——13 个波次均已合并，实际读数见 §11。

---

## 11. 执行记录（2026-10-29 收口）

本节是**执行事实**，与 §0–§F3 的方案文本并列保存：方案里的估算、风险等级和"预期结果"
不回写，实测值统一写在这里，便于对照"当初怎么估的 / 实际做到什么"。

- 任务包：`2026-10-27-codebase-simplification-program`（状态 `in-progress`，见 §11.6）
- 证据：`.harness/evidence/2026-10-27-codebase-simplification-program/`（程序级汇总）
  与 `.harness/archive/2026-10/evidence/2026-10-27-s{1,2,3,4}-*/`（13 个波次原始证据）
- 收口包：`2026-10-29-simplification-program-closeout`

### 11.1 结论

1. **13 个波次全部合并**（S1×2、S2×6、S3×3、S4×2）。最后一项 `pantheon-ops` 构建验证已于
   2026-10-29 关闭：`pantheon-base-v0.18.0`（`b4bb2392`）发布后消费仓前端红在 `TS2614`，
   `pantheon-base-v0.18.1`（`93505225`）恢复 `isForbiddenRequestError` 导出，消费仓
   `go build ./...` 与 `npm run build` 均通过（staging 树读数；live 替换属消费仓 L2 门禁）。
2. **六个横向重复族各收敛到一份实现**：路径 ID 解析、分页归一化、排序白名单、
   `ErrRecordNotFound` 转换、批量操作、导入导出管线。例外 3 处（`setting_audit` 分页回退、
   tenant member 分页、i18n 的两处 ID 解析），每处都在对应波次证据里写明"为什么不能合"。
3. **两次"假阳性"被记录而非强改**：RC-029（`permissionServiceErrorCode` 有测试直接调用）
   与 RC-030（`user_service` 双层包装不是重复作用域）。按 §0.1 的禁令，宁可保留也不改测试迁就重构。
4. **S3 的两项高风险改造都做了**：租户作用域三件套（RC-012/027）改走 `pkg/tenant.Scoper`，
   `tenantOwnerID` 定义数从 11 降到 **0**；角色数据范围镜像模型（RC-022）删除镜像结构。
5. **相对方案的最大偏差是"行数"**：净减少没有达到 −1,500～−2,100 的估算区间，
   原因是 S2 的共享助手与配套测试是**新增**代码（+507、+262 净增），删除集中在 S1。

### 11.2 波次实测

每行取该波证据里记录的 `git diff --stat`（不是合并提交的 `--shortstat`——
本仓的 squash 会把相邻波次的证据文件一并带入，提交级数字不可比）。

| 波次 | 合并 | 实测 diff | 处置的 RC |
|---|---|---|---|
| S1 dead-code | #438 `321a6384` | 19 files, +0 / −1,923 | RC-001/002/003/005/009/010/011 |
| S1 api-surface | #439 `b0a1952a` | 3 files, +11 / −9 | RC-004/007；RC-006/008/029 记录为保留 |
| S2 shared-helpers | #450 `958bc6ba` | 9 files, +641 / −134（纯新增，0 调用点改动） | RC-013/014/015/017/019/024 的共享落点 |
| S2 id-parse | #451 `2decc4b4` | 11 files, +66 / −134 | RC-014（35 → 2 处） |
| S2 pagination-sort | #452 `73126586` | 29 files, +685 / −423 | RC-013/015/028 |
| S2 error-mapping | #454 `9b22b36b` | 31 files, +208 / −290 | RC-017/019/020/024 |
| S2 batch-and-envelope | #455 `92520d75` | 0 代码（5 文件 +180 / −1，仅证据与状态） | RC-023/025/030（复核后属已消解或假阳性） |
| S2 import-export | #456 `547add28` | 19 files, +578 / −404（含 95 行清单文档） | RC-016 |
| S3 tenant-scope | #461 `cbd37694` | 29 files, +278 / −373（生产代码 25 files +267 / −364） | RC-012/027 |
| S3 role-data-scope | #461 `cbd37694` | 7 files, +54 / −75 | RC-022 |
| S3 oidc-envelope | #461 `cbd37694` | 1 file, +13 / −0（仅注释，零功能差异） | RC-026 |
| S4 governance | #441 `bf0711e9` | 0 代码（8 files +411 / −1，决策文档 + 索引） | RC-031/036/037/038/043 的决策与逐符号计量 |
| S4 alias-removal | #447 `54151d54` | `re_export.go` 112 → 52 行；6 files +202 / −45（含证据） | RC-043（10 个零消费别名） |

### 11.3 结构指标（§0.4 的度量命令，2026-10-29 复测）

| 指标 | 方案目标 | 基线（2026-10-27） | 实测（2026-10-29） | 判读 |
|---|---|---|---|---|
| `ParseUint(c.Param("id")` 内联 | 35 → ≤2 | 35 | **2** | 达标；残留 = i18n 两处（文案是 `id.invalid`，与共享的 `param.invalid` 不同） |
| 分页归一化实现 | 13 → 1 | 13 | **1 + 字段适配器** | 达标；模块里保留的同名函数只做 nil 安全取字段，不含钳制/回退逻辑 |
| `pageSize > cap` 钳制行 | — | 11 | **1** | 残留 = `setting_audit`（oversize 回退 10，与共享 clamp 语义相反） |
| `Offset((page-1)*pageSize)` 内联 | 14 → 1 | 14 | **4** | 残留 = `setting_audit` ×1 + tenant ×3（随 W9 租户域重写收敛） |
| `func ... tenantOwnerID` 定义 | 11 → 1 | 11 | **0** | 超额达标：全部走 `pkg/tenant.Scoper` / `OwnerTenantID` |
| `var errParamInvalid` 声明 | 11 → 1 | 11 | **0** | 达标：唯一来源 `pkg/common.ErrParamInvalid` |
| 排序白名单 helper | 8 → 1 | 8 | **0 内联 + 8 适配器** | 达标：`common.NormalizeSort` + 各模块包级 var |
| 后端测试包 | ≥46 ok / 0 FAIL | 46 ok | **47 ok / 0 FAIL / 6 无测试文件** | 达标（多出的包是 `modules/system/iam` 根包，来自 S3 的双租户隔离冒烟） |

### 11.4 行数窗口

| 范围 | 2026-10-27 基线 `96ef1d66` | 2026-10-29 `HEAD` | 差 |
|---|---|---|---|
| `backend/**/*.go`（不含 `_test.go`） | 53,179 | 52,332 | **−847** |
| `frontend/src/**/*.{ts,tsx}` | 68,169 | 67,828 | **−341** |
| 合计 | 121,348 | 120,160 | **−1,188** |

**口径警告**：该窗口同时包含**净增**的非精简工作——租户 W6 控制台 UI、W7 计划阶梯、
W9 租户域重写、发布说明目录化、`TenantList` 的 S3776 修复。因此 −1,188 是**下界**，
既不能当作"精简收益"引用，也无法据此判定是否达到方案的 −1,500～−2,100 区间。
可比的只有波次级实测（§11.2）与结构指标（§11.3）。

### 11.5 RC 覆盖率

`REDUNDANT_CODE_CANDIDATES.md` 共 45 条（RC-001…RC-045）：

- **33 条有明确处置记录**（关闭、保留并说明、或定义为假阳性）：
  001–017、019、020、022–031、036、037、038、043。
- **12 条无波次处置，全部落在本文"暂不处理"表**，属显式遗留而非常规遗漏：
  RC-018、RC-021、RC-032、RC-033、RC-034、RC-035、RC-039、RC-040、RC-041、RC-042、RC-044、RC-045。
- RC-038（归档文件体积）同时在"暂不处理"表与 S4 决策里，结论是 **keep**：
  文件数从 659 涨到 1,193，涨幅来自当日新增证据，不是代码重复。

### 11.6 验证读数与仍未关闭的项

2026-10-29 在 `main` @ `3b73c7a4` 复测：

- `cd backend && go build ./... && go vet ./...` → 0 错误
- `cd backend && go test -count=1 ./...` → 47 ok / 0 FAIL / 6 个无测试文件的包
- `npm run check:task-packet` → 13 warning（0 error）；其中"程序包 review 缺失"一条由本轮收口消除
- `check:harness-docs`、`check:structure`、`check:harness-encoding`、`check:docs-frontmatter`、
  `check:harness-inventory`、`check:harness-adoption`、`check:harness-visual`、
  `check:ui-quality-gate`、`check:failure-registry`、`check:duplication` → 0 finding

**仍未关闭（维护者决策，不属于本程序遗漏）**：

1. ~~`pantheon-ops` 对同步后 foundation release 的 `go build ./... && npm run build`~~
   — **已关闭（2026-10-29）**：`pantheon-base-v0.18.0` 发布后消费仓前端红在 `TS2614`
   （`api/request` 缺 `isForbiddenRequestError`），由 `pantheon-base-v0.18.1` 以 patch 恢复；
   重建后后端 `go build ./...` 与前端 `npm run build` 均通过。修复类目登记为 `FR-021`，
   规则写入 `docs/harness/AI_QUALITY_GOVERNANCE.md` §5.4。live 树替换仍需消费仓自身评审。
2. 是否在 S2/S3 之后冻结功能进入维护期（方案 §C 的排期问题）。
3. RC-036/037/038 的最终去留：S4 已定 keep / keep-and-check（含
   `scripts/harness/check-doc-bilingual.mjs` 规格），若维护者改判需另立波次。
