---
title: Pantheon Base 全局架构、代码质量、文档治理与工程成熟度审计（2026-10-31 合并终版）
doc_type: Assessment
layer: platform
status: Active
updated_at: 2026-10-31
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
  - docs/contracts/TENANT_CONTRACT_V1.md
  - docs/contracts/SYSTEM_AUTH_CONTRACT.md
---

# Pantheon Base 全局审计报告（合并终版）

> **版本说明**：本文件合并同日产出的两份审计（初版 `PANTHEON_BASE_GLOBAL_AUDIT_2026-10-31.md` 与复核稿
> `PANTHEON_BASE_AUDIT_FINAL_2026-10-31.md`），是**唯一权威版本**；两份前序文件已删除，不再保留副本。
> 合并方式不是取并集，而是对 12 处互相矛盾的事实逐条复跑取证后裁决（§0.3 / §0.4 / §0.5）。
>
> **性质**：只读审计。本轮审计阶段未修改任何业务代码、迁移、CI、规范或 Pantheon Ops；
> 报告与整改任务包是本轮唯一新增文件（§15 说明整改包只是计划，尚未执行）。

---

## 0. 版本合并与事实更正

### 0.1 为什么必须合并后再采信

两份前序报告结论方向一致（都是 **B：少量整改后冻结**），但支撑证据有 12 处直接冲突。其中两条冲突会**改变整改动作**，因此不能简单取并集：

| 冲突 | 若采信错误一方会造成的后果 |
| --- | --- |
| `pkg/security/ssrf` 是死代码还是被使用 | 会删掉下游 `pantheon-ops` 正在用的安全包（argocd / k8s / observability 5 处引用） |
| 低代码生成器是“缺编译门禁”还是“产物不可编译且不建表” | 只加 CI 门禁，而真正的功能仍然是坏的 |

### 0.2 基线口径（三快照，必须分清）

| 项 | 值 | 用途 |
| --- | --- | --- |
| **审计基线** | `origin/main` = `bb0498f5`，`VERSION` = `0.18.2`，tag `pantheon-base-v0.18.2` | **本报告所有结构性计数以此为准** |
| 取证快照 | 工作区 HEAD = `ef00738`（分支 `fix/release-gate-checks-poll-resilience`），落后 main 5 / 领先 3 | 构建、测试、门禁、生成器复现均在此快照执行 |
| 本地 main | `e17eaaa1`（落后 origin/main） | 不使用 |

**混用基线是两份前序报告大部分数字冲突的根因**：版本号读 main、文件清单读工作区，就会得出“发布说明缺失”这类假发现。本报告的统一规则是——**结构性计数（文件数、行数、迁移数）锁定 `origin/main`；可执行证据（构建/测试/门禁）标注为工作区 HEAD 快照**。

| 项 | 初版曾写 | 真值（main） | 差值原因 |
| --- | --- | --- | --- |
| `docs/*.md` | 404 | **408** | 初版读 HEAD；main 多 4 份 |
| 全仓 `.md` | 1,103 | **1,120** | 同上 |
| `.harness/` 文件 | 1,294 | **1,313** | 同上 |
| 工作区 `docs/*.md` | — | 409 | = 408 + 本报告 |

### 0.3 已更正的事实（8 项）

| # | 项 | 初版说法 | 真值 | 证据 |
| --- | --- | --- | --- | --- |
| C1 | `pkg/security/ssrf` | “被上传/外部数据源路径使用” | base 内 **0 引用**；`pantheon-ops` **5 处**引用：`business/cicd/argocd/argocd_client.go:16`、`business/k8s/pkg/client_factory.go:7`、`business/observability/datasource_service.go:19`、`business/observability/notification_service.go:19`、`argocd_client_test.go:12` | `grep -rn` 双向检索 |
| C2 | 前端规模 | 255 文件 / 68,115 行 | **256 文件 / 68,115 行**（行数正确，文件数少 1） | `find frontend/src -name '*.ts' -o -name '*.tsx' \| wc -l` |
| C3 | 前端单元测试 | 48 个 vitest 文件 | **34 个** | `find frontend/tests/unit -name '*.test.*' \| wc -l` |
| C4 | harness checker | “10 个” | **26 个** `scripts/harness/check-*.mjs`（目录内共 32 个 `.mjs`，另 6 个是辅助脚本） | `ls scripts/harness/check-*.mjs \| wc -l` |
| C5 | 未接线 checker | “未发现无价值 checker” | **11/26 未接入任何 npm script 或 workflow（42%）** | 逐文件比对 `package.json` / `frontend/package.json` / `.github/workflows/` |
| C6 | 任务包状态 | “82 份中 69 份 Active” | main 上 **82 份中 68 份 Active**；工作区 **86 份中 68 Active / 7 Archived / 4 Draft** | `grep -h '^status:' docs/harness/tasks/*.task.md \| sort \| uniq -c` |
| C7 | `docs/designs` 索引缺口 | “19 份未进索引” | **18 份** | 按 stem 逐个比对 `docs/designs/README.md` |
| C8 | `RELEASE_NOTES_v0.18.2.md` | 曾判“缺失，索引未更新” | **存在（98 行）**，`docs/release-notes/README.md` 已同步 | `git cat-file -e origin/main:docs/release-notes/RELEASE_NOTES_v0.18.2.md` |

> C8 是**唯一一条初版完全错误的文档结论**，成因是“版本读 main、文件读工作区”。复核确认 `git diff --name-only HEAD origin/main` 未触碰 `README.md` / `DESIGN.md` / `docs/README.md` / `docs/designs/README.md` / `docs/contracts/**` / `docs/operations/**`，因此其余文档结论对 main 依然成立。

### 0.4 新增发现（3 项，两份前序报告都没有）

| # | 发现 | 证据 | 归入 |
| --- | --- | --- | --- |
| N1 | **低代码生成器第三重缺陷：默认迁移模式下生成的表永远不会被创建** | 生成物不含任何 `.sql` 迁移；建表只靠 `service.Migrate()` → `AutoMigrate`；而 `pkg/contracts/module.go` 的 `RegisterBackendModules` 在 `PANTHEON_AUTO_MIGRATE != true` 时**只调 `Bootstrap`、不调 `Migrate`**；生成的 `contracts.FuncModule` 只设 `MigrateFunc`、不设 `BootstrapFunc`（`FuncModule.Bootstrap` 在 `BootstrapFunc == nil` 时返回 nil） | AUD-001 |
| N2 | **业务域整体不在租户模型内**（不是“按行跨租户读写”） | `ProtectedGroup`/`DataScopedGroup` 不挂租户中间件；`tenant.FromGin` 无值返回 nil；`WithTenantScope(nil)` 原样返回 db；`internal/scaffold/contract.go:74` 明确 `"tenant" is intentionally rejected: tenant isolation has no runtime implementation`；`grep -rn "tenant_id\|TenantID" pantheon-ops/backend/modules/business` = **0** | AUD-002 |
| N3 | **token 会话缓存存在数据竞争** | `lookupTokenSessionCache` 在 `RLock` 下返回 `*tokenCachedSession` 共享指针；`refreshTokenActivity` 在锁外改写 `cached.data.LastActivityAt`（`token_middleware.go:183-188`）；`resolveTokenSession` 也把该指针的 `data` 直接交给下游 | AUD-026 |

### 0.5 严重度调整（5 项）

| ID | 初版 | 本版 | 理由 |
| --- | --- | --- | --- |
| AUD-001 | P1 | **P1（加重）** | 从“不可编译”升级为“不可编译 + 不建表”，功能面完全不可用 |
| AUD-002 | P1 | **P1（重述）** | 机制成立但后果被夸大：业务表无 `tenant_id`，不存在按行越权；真实缺口是**业务域零租户建模 + 入口助手违约 + README 宣称“完整数据隔离”** |
| AUD-003 / AUD-004 | P1 | **P2** | 入口文档失实会误导维护者与 Agent，但不造成执行失败；与 AUD-016 分开定级 |
| AUD-005（OIDC） | P1 | **P2** | 587 行不可达代码无运行时风险（未注册路由），属“需维护者决策的清理”，不阻断冻结 |
| AUD-016（`docs/README` 建库/启动命令） | P2 | **P1** | 按文档操作**必然失败**（执行 `system_init.sql` 缺 13 张表、根目录 `go run ./backend/cmd/server` 无法执行），属可执行性缺陷 |

---

## 1. 总体结论

### 1.1 审计元信息

| 项 | 值 |
| --- | --- |
| 审计时间 | 2026-10-31（Asia/Shanghai） |
| 被审仓库 | `pantheon-base`（workspace::`pantheon-platform/pantheon-base`） |
| 审计基线 | `origin/main` = `bb0498f5`，`VERSION` = `0.18.2` |
| 取证快照 | 工作区 HEAD = `ef0073860ce03087d9cf05de6b5d7732687ef769`（分支 `fix/release-gate-checks-poll-resilience`，与 main 分叉 5/3） |
| 工作区状态 | **脏**：6 个未提交 i18n 文件 + 4 个审计任务包 + 本报告（见 §1.3） |
| 下游参照 | `pantheon-ops` @ `aee56bc6`（`fix/frontend-follow-redirects-audit`），锁 `pantheon-base-v0.18.2` / `a44ae4e0` |

### 1.2 审计范围与方法

- **范围**：`backend/`、`frontend/`、`backend/pkg/database/migrations/`、`.github/workflows/`、`docs/`（408 个 Markdown）、`.harness/`（1,313 个受管文件）、`scripts/`、`releases/`、`config/`、根级 11 份入口文档，以及 `pantheon-ops` 的 overlay 与 foundation lock。
- **方法**：Git 状态与历史取证 → 入口文档通读 → 结构/行数统计 → 关键调用链抽样（认证、租户上下文、数据权限、审计、字典、低代码生成器）→ 文档资产全量盘点与链接/引用分析 → 机械门禁实际执行（见 §14）→ **生成器实跑复现**（输出到系统临时目录，未落仓库）。
- **未采用**：不依据 README/历史报告下结论；不把目录命名当作实现证据；不对未执行的压力测试宣称性能结论。

### 1.3 工作区未提交修改（必须显式声明）

```
 M backend/modules/system/i18n/builtin_locale_resources.json      (+1059/-50)
 M frontend/src/i18n/resources/{en-US,fr-FR,ja-JP,ko-KR,zh-CN}.ts
?? .harness/tasks/2026-10-31-audit-*/                               （4 个整改任务包）
?? docs/harness/tasks/2026-10-31-audit-*.task.md                    （4 份任务文档）
?? docs/reviews/PANTHEON_BASE_GLOBAL_AUDIT_2026-10-31.md            （本报告）
```

i18n 改动内容为：新增 `system.tenant.*` 等租户控制台文案、删除 4 个无引用的 `auth.login.tenant.*` 死 key、改写登录页说明文案。已核对 `auth.login.tenant.*` 在 `frontend/src` 与 `backend/` 中**零引用**，删除安全。该改动**不属于已合并代码**，本报告的代码结论以 `origin/main` 提交为准。

### 1.4 规模基线（实测，`origin/main`）

| 区域 | 文件数 | 行数 |
| --- | --- | --- |
| 后端生产代码 | 231 `.go` | 52,342 |
| 后端测试代码 | 189 `_test.go` | 40,757（测试/生产 ≈ 0.78） |
| 前端 `src` | 256 `.ts/.tsx` | 68,115 |
| 前端单元测试 | 34 个 vitest 文件 | — |
| 前端 E2E | 51 个 Playwright spec | — |
| 数据库迁移 | 30 个版本（60 个 up/down 文件） | 结尾 `000030_tenant_plan_definitions` |
| 文档 | 1,120 个 `.md`（`docs/` 408、`.harness/` 1,313 受管文件、其余入口/发布/脚本） | — |
| harness checker | 26 个 `check-*.mjs`（其中 11 个未接线） | — |

模块体量：`system` 47,397 行、`auth` 14,288 行、`lowcode` 6,833 行、`platform` 1,184 行、`business` 440 行（仅注册骨架）。

### 1.5 总体成熟度判断

**结论：成熟度中上，具备“停止大规模功能开发”的基础，但不具备“零整改冻结”的条件。**

支撑结论的正面事实（均已验证）：

- `go build ./...`、`go vet ./...`、`go test -short ./...` **全部通过，0 失败**；`tsc -b` 通过；前端 4 条主门禁（menu-contract / i18n-hardcode / ui-contract / system-page-admission）通过；文档链接、清单、frontmatter、重复度检查通过。
- 安全主链路（Token 校验 + Redis 会话 + 黑名单 + 空闲超时 + 撤销、CSRF 双提交、HttpOnly/Secure/SameSite=Strict、Casbin domain 展开、租户上下文 deny-by-default、数据权限 fail-closed）实现完整且有针对性测试。
- 租户生命周期（创建/初始化/暂停/归档/删除/成员/套餐/审计）在**系统域**形成闭环，`ProvisionTenant` 在事务内完成，失败可回滚。
- 模块边界真实成立：`business` → `system` 生产代码 **0 引用**；无循环依赖；`auth` 通过 `pkg/contracts/authuser` 端口解耦，compose root 注入。
- 文档体系有骨架（frontmatter、链接检查、索引、合同/设计/验收分层）。

阻止“直接冻结”的问题集中在三处：**低代码生成器的后端产物不可编译且默认迁移模式下不建表**、**业务域整体不在租户模型内而入口文档宣称“完整数据隔离”**、**入口文档给出必然失败的操作步骤**。三者都属于“有限整改”范围，不构成架构性失败。

---

## 2. 整体架构分析

### 2.1 分层与边界（通过）

| 检查项 | 结论 | 证据 |
| --- | --- | --- |
| 组合根单一 | 通过 | `backend/cmd/server/main.go:157-176` 统一装配 platform/lowcode/system/auth/business |
| 业务域不得依赖系统域实现 | 通过 | `grep "modules/system" backend/modules/business`（生产代码 0 命中） |
| auth 与 user 解耦 | 通过 | `backend/modules/auth/module.go:18-33` 注释 + `pkg/contracts/authuser` 端口；`main.go:173` 注入 |
| 循环依赖 | 无（`go build` 通过即为充分条件） | — |
| 跨层反向依赖 | 1 处轻微 | `backend/modules/system/system_modules.go:7` 依赖 `modules/auth/session`（仅用于 `session.NewLifecycleService`，用户删除时吊销会话） |
| 平台层不做业务硬编码 | 通过 | `backend/modules/platform/*` 仅聚合读面，`dashboard_service.go` 使用表白名单 |

### 2.2 抽象与设计合理性

**判断为“合理保留（D）”的抽象：**

| 抽象 | 使用面 | 判断 |
| --- | --- | --- |
| `pkg/contracts.BackendModule` / `FuncModule` | 全部 12+ 模块注册 | 必要：统一 Migrate/Bootstrap/SeedMenus/SeedPerms/SeedI18n/RegisterRoutes 生命周期 |
| `tenant.Scoper`（嵌入式租户上下文） | 11 个 service | 必要：把“读过滤 + 写归属 + nil 语义”收敛为一处，且有契约注释与测试 |
| `pkg/tenant.Resolve` + 两个租户中间件 | 全部受保护路由组 | 必要：把 deny-by-default 与平台兜底做成可测试函数 |
| `pkg/impexp` 导出统一上限 | 4 个导出入口 | 必要：10,000 行硬上限单点维护（`pkg/impexp/export_limits.go:5`） |
| `pkg/contracts/authuser` 端口 | auth ↔ system 唯一通道 | 必要：无它则 auth 直接 import system，边界即失守 |
| `pkg/security/ssrf` | base 内 0 消费者；**下游 5 处** | 必要（D 类）：为消费仓提供的公共安全能力，**禁止以“base 内无引用”为由删除** |

**判断为“不必要/失效（C/A）”的抽象：** 见 §4 的 `ProtectedGroup`、`scaffold.Generate`、`readRoleKeysFromContext`、`usePagination/useRequest`、OIDC 包。

### 2.3 发现：业务域整体不在租户模型内，且路由助手违反自家合同（AUD-002，P1）

这是本报告与初版结论差异最大的一条，**必须区分“机制”与“后果”**：

**机制（成立）**

- `backend/pkg/contracts/route_groups.go:15-28` 的 `ProtectedGroup` / `DataScopedGroup` 只有 `TokenAuthMiddleware` + `CasbinMiddleware`（+ 可选 `DataScopeMiddleware`），**没有** `TenantContextMiddleware` / `PlatformScopeTenantContextMiddleware`。
- 这两个函数是业务模块的官方入口，且**下游大量在用**：`pantheon-ops` 共 **14 处**调用（13 处 `DataScopedGroup` + 1 处 `ProtectedGroup`）（`business/cmdb/cmdb.go`、`k8s/k8s.go`、`deploy/deploy.go`、`filedist/module.go`、`bizscope/module.go`、`cicd/cicd.go` 等）；base 自身 **0 处**使用。
- `pkg/tenant/tenant.go` 的 `FromGin` 在无上下文时返回 `nil`；`WithTenantScope(nil)` **原样返回查询**（不过滤）。因此这些路由在 multi 模式下既不解析租户，也不会拒绝请求。
- 这与 `docs/contracts/TENANT_CONTRACT_V1.md:307-311`（“上下文缺失且非 compat 模式 → 请求拒绝”）直接冲突。

**后果（初版夸大，必须更正）**

- 业务域**根本没有租户建模**：`backend/internal/scaffold/contract.go:74` 明确注释 `"tenant" is intentionally rejected: tenant isolation has no runtime implementation`；生成器产出的表**不含 `tenant_id`**；`pantheon-ops/backend/modules/business` 全目录 `TenantID` 命中 **0**，`tenant_id` 仅 **1** 处（`retired_modules_test.go:110` 的 `CREATE TABLE` 测试夹具，非模型字段）。
- 因此**不存在“按行跨租户读写”**——业务表里没有可越权的租户列。真实缺口是：**业务域是单租户语义，但既没有在合同里写明，也没有在 README 里说清**，而 `README.md:31` 宣称“完整数据隔离”。
- **定性：设计/文档缺口 + 入口助手违约，不是可利用漏洞。** 置信度：高（静态检索 + 下游代码实测）。

**建议（二选一，须维护者拍板）**

- (a) **明确单租户语义**：在 `TENANT_CONTRACT_V1` 增加“业务域在 v1 内不参与租户隔离”的显式条款，修正 README，并把 `ProtectedGroup`/`DataScopedGroup` 的文档注释改成“不解析租户上下文”，同时给下游一条显式说明。
- (b) **补齐租户语义**：助手内置 `PlatformScopeTenantContextMiddleware`，生成器产出 `tenant_id` + 迁移，业务表加租户列与索引，与 `pantheon-ops` 同批升级。

**风险**：方案 (b) 需双仓同批发布，且要处理存量业务表的数据归属；方案 (a) 改动小但会冻结一项能力承诺。

---

## 3. 核心功能完整性分析

### 3.1 单租户（compat）能力闭环

| 能力 | 状态 | 证据 |
| --- | --- | --- |
| 登录 / 刷新 / 登出 | 完整 | `modules/auth/login/login_handler.go`；`/auth/login|refresh|logout` |
| 密码存储 | 完整 | bcrypt（`tenant_provision.go` 与 user service 同源） |
| 会话 / 在线设备 / 强制下线 | 完整 | `modules/auth/session/*`，含分页与撤销闭环测试 |
| 用户 / 角色 / 菜单 / 权限 | 完整 | `modules/system/iam/*`；菜单契约门禁实测通过（19 菜单/23 路由/102 权限/23 组件键） |
| 组织（部门/岗位） | 完整 | `modules/system/org/{dept,post}`，含导入导出与治理任务 |
| 数据权限 | 完整（含 UI） | `internal/middleware/data_scope_middleware.go` + `PermissionDataScopeTab.tsx` |
| 审计（登录/操作/安全事件） | 完整（租户过滤） | `audit_handler.go:30-40` 越权过滤；`audit_service.go:37` scope |
| 字典 / 设置 / i18n | 完整 | `config/dict`、`config/setting`（敏感项加密 + 变更审计）、`i18n`（5 locale） |
| MFA / TOTP | **完整，含首次绑定** | `mfa_service.go:83-146`（无 factor 时 `setupRequired` 下发 secret/URI），前端 `Login.tsx:138-155` 渲染密钥 |
| OIDC | **未接线（见 AUD-005）** | `modules/auth/oidc/*` 无任何路由注册 |
| 业务域（cmdb/k8s/deploy 等） | 仅在 `pantheon-ops`，base 只有 440 行注册骨架 | 符合“底座不装业务”的定位 |

### 3.2 多租户闭环（仅系统域）

| 能力 | 状态 | 证据 |
| --- | --- | --- |
| 租户 CRUD | 完整 | `tenant_platform_handler.go`；8 个 `/system/tenant*` 路由 |
| 初始化（角色/管理员/成员/策略/设置/字典副本） | 完整且事务化 | `tenant_provision.go` 单事务 + `lockTenantRow` 串行化；幂等，可作修复入口 |
| 生命周期（active/suspended/archived/deleted） | 完整 | `tenant_model.go:11`；`tenantStatusTransitionAllowed` 实现合同 §2.1 |
| 成员与配额 | 成员完整；**配额仅展示（AUD-003）** | `tenant_ops_view.go:159-176` 读 `QuotaFor`；`user/role` 写路径无 quota 校验 |
| 套餐 | 完整（可扩展、落库） | `tenant_plan_model.go` + 迁移 `000030` + 前端 `TenantPlanMatrix.tsx` |
| 数据隔离 | **系统域完整；业务域不适用（AUD-002）** | §2.3 |
| 租户上下文传递 | 中间件完整；业务入口缺失 | `tenant_context_middleware.go` 全文 |
| 停用/恢复拦截 | 完整 | `tenant.Resolve` 抛 `ErrTenantSuspended/Archived`，`tenantErrorMessageKey` 映射 i18n |
| 平台控制台越权 | 已封堵 | `PlatformCapabilityMiddleware` 阻断租户管理员读全量租户表 |
| 租户切换 | 完整（成员行为授权，非 Casbin） | `auth/module.go:126-133` 注释与实现 |

### 3.3 调用链抽样（前端 → API → 中间件 → Handler → Service → DB）

1. **用户列表**：`UserList` → `/system/user/list` → Token → Tenant → Casbin → DataScope → `ListUsers`。批量加载角色/部门/岗位（`user_service.go:240-290`），**无 N+1**。
2. **租户创建**：`TenantList` → `POST /system/tenant` → `PlatformCapabilityMiddleware` → `CreateTenant` → `ProvisionTenant`（事务）→ 审计。平台能力门禁在 `system_modules.go:288` 组级 + 路由级确认。
3. **操作日志导出**：`OperationLogList` → `POST /system/log/oper/export` → 租户 scope → `impexp` 10k 上限 → CSV。`audit_handler.go:26-40` 拒绝非平台主体携带 `tenantIdFilter`。
4. **低代码生成**：`ModuleWizard` → `POST /lowcode/generator/preview-files` → Go `scaffold` → **Node 子进程** → TS 生成器 → 返回文件列表。**该链路后端产物不可编译且不建表（AUD-001）**。

### 3.4 发现：低代码生成器后端产物四重缺陷（AUD-001，P1）

实测（`node frontend/scripts/export-generated-module.mjs <schema>`，输出到系统临时目录，未落仓库），生成 10 个文件 / 74,509 字节：

```
backend/modules/business/order/module.go:
  import (
      "pantheon-base/internal/middleware"     ← 缺陷 1
      "pantheon-base/pkg/contracts"
  )
  protected := r.Group("/business/order").Use(middleware.TokenAuthMiddleware()).Use(middleware.CasbinMiddleware())
                                          ↑ 缺陷 2
```

| # | 缺陷 | 证据 |
| --- | --- | --- |
| 1 | 导入路径 `pantheon-base/...` 与模块名 `github.com/duanxldragon/pantheon-base/backend`（`backend/go.mod:1`）不符（5 个 Go 文件里 3 个受影响） | `go list pantheon-base/internal/middleware` → `is not in std` |
| 2 | `middleware.TokenAuthMiddleware()` 少传 `rdb *redis.Client` | 真实签名见 `internal/middleware/token_middleware.go:214` |
| 3 | **不生成任何 `.sql` 迁移**，建表只靠 `service.Migrate()` → `AutoMigrate` | 生成文件清单无迁移；`backendGenerator.ts` 无 SQL 模板 |
| 4 | **生成的 `contracts.FuncModule` 只设 `MigrateFunc`、不设 `BootstrapFunc`** | `RegisterBackendModules`（`pkg/contracts/module.go`）在 `PANTHEON_AUTO_MIGRATE != true` 时**只调 `Bootstrap`、不调 `Migrate`**；`FuncModule.Bootstrap` 在 `BootstrapFunc == nil` 时返回 nil → **默认版本化迁移模式下表永远不会被创建** |

另有旁证说明“模板与注册器不同源”：同一生成器写出的注册表用的是正确模块路径（`backend/internal/scaffold/workspace.go:394`）。

穷尽了三条可能的补偿路径，均不存在改写：`exporter.ts`、`frontend/scripts/export-generated-module.mjs`、`scaffold.WriteGeneratedModuleSource`（`workspace.go:151-191`）都只按原文写盘；`dynamic_module_service.go:323-333` 的 `validateGeneratedModuleSource` 只校验目录存在，**不做编译校验**。

**影响**：生成后的“重启激活”必然以编译失败告终；即便编译通过，默认迁移模式下表也不存在。`dynamicmodule` 侧仍会把注册状态推进到 `pending_activation`/`active`，属**静默失败面**。**置信度：高**（静态 + 实跑复现）。

> **处置（2026-10-31）**：四处缺陷已全部闭合，执行记录见 §16.5。执行中另发现两项（AUD-001b `gofmt` 不干净 / AUD-001c `BootstrapFunc → AutoMigrate` 与部署指南冲突），均登记在任务包 `residualFindings`。

### 3.5 发现：OIDC 为不可达脚手架（AUD-005，P2）

- `backend/modules/auth/oidc/` 共 587 行（`config.go 77` / `handler.go 115` / `provider.go 92` / `service.go 303`），导出 `InitiateOIDCLogin`、`HandleOIDCCallback`、`GetOIDCConfig`。
- 全仓非测试代码对 `oidc.` 的引用：**0**。未注册任何路由；前端 `frontend/src` 对 oidc **0 命中**。
- 读取 `PANTHEON_OIDC_*` 环境变量（`oidc/config.go:24-61`），但 `.env.example` 与部署文档均未登记，说明不是对外承诺。
- 与 `docs/contracts/SYSTEM_AUTH_CONTRACT.md:168`（“未明确身份源前，不提前实现真实 SSO/OIDC”）直接冲突——**代码违反合同**，而不是文档过期。
- 迁移 `000029_oidc_fields` 已把 `oidc_subject/oidc_provider/oidc_last_sync/auth_type` 并入版本链（该迁移本身写法正确）。
- **建议**：三选一并记录决策——(a) 删除包与迁移列（不可逆，需合同评审）；(b) 保留但移入 `internal/experimental/` 并从可发布面移除；(c) 补齐路由 + 文档 + 测试。**定级 P2**：不可达代码无运行时风险，但需维护者拍板。

---

## 4. 冗余代码与过度设计分析

### 4.1 已确认无用（A 类）

| ID | 位置 | 事实 | 调用方 | 建议 | 风险 |
| --- | --- | --- | --- | --- | --- |
| AUD-012a | `frontend/src/hooks/usePagination.ts`（59 行）、`useRequest.ts`（105 行） | 仅被 `hooks/index.ts:15-18` 再导出，`frontend/src` 内**零 import** | 无 | 删除（含 barrel 两行） | 低；若下游按名引用会编译失败（**须先做 workspace 级检索**，见 FR-021） |
| AUD-012b | `backend/internal/scaffold/engine.go:28-44` `Generate()` | 反序列化后丢弃，返回 `nil`；`capitalize` 仅被它使用 | 无 | 删除 50 行整文件或将 `Go` 文档标记为示例 | 低 |
| AUD-020 | `backend/internal/middleware/casbin_middleware.go:128-130` | `readRoleKeysFromContext` 仅测试调用 | `casbin_middleware_test.go:61,71` | 删除并让测试直接用 `common.GetRoleKeys` | 低 |
| AUD-013 | 表 `permission_role_data_scope_policy` | 与 `system_role_data_scope` 功能重叠；**无功能读写**，仅有 `role_service.go:600-603` 删除角色时的防御性清理 + `dashboard_service.go:348` 表白名单占位 | 无 | 新迁移 drop（**不得改写 `000001`**），保留 down | 中：需数据归档确认 |
| AUD-006 | `docs/harness/tasks/*.task.md` | main 上 82 份中 **68 份仍 `status: Active`**（含 2026-08-05 的发布任务），而 `.harness/archive/` 中证据已归档 | 索引脚本不校验 | 批量置 `Archived` | 低 |

> **注意**：`pkg/security/ssrf`（268 行 + 221 行测试）**不属于 A 类**。base 内 0 引用，但 `pantheon-ops` 有 5 处引用，是 D 类“为下游保留的公共安全能力”。**禁止以“零引用”为由删除**（初版曾如此建议，已撤回）。

### 4.2 重复实现（B 类）

| ID | 重复面 | 证据 | 建议 |
| --- | --- | --- | --- |
| AUD-014 | 两套认证自助入口：`/system/login|refresh|logout|user/info` 与 `/auth/login|refresh|logout|me` | `backend/modules/auth/module.go:69-80,86-90,107-117`；`csrf_middleware.go:22-26` 为 legacy 路径开 CSRF 豁免 | base 前端只用 `/auth/*`（`api.ts:30`、`request.ts:98,149`），ops 亦然。建议设弃用期后下线 legacy（需对外公告） |
| AUD-015 | 同一条 Go 测试套件在 `ci.yml` 与 `quality.yml` 各跑一次 | `ci.yml:167`（`go test -race -short`）、`quality.yml:384`（`go test -race`），二者均在 `pull_request` 触发 | 保留 quality（合并门禁），ci 侧改为按路径过滤或删除 |
| AUD-008 | 三份文档元数据规范：`docs/harness/DOCUMENT_FRONTMATTER_SPEC.md`(102 行) / `docs/contracts/DOCUMENT_FRONTMATTER_SCHEMA.md`(321 行) / `docs/contracts/DOCUMENT_METADATA_AND_STATUS.md`(302 行) | 三者定义同一 frontmatter 约定；检查器 `scripts/frontmatter-check.mjs` 实际按 contracts 版实现 | 合并为 contracts 一份权威，harness 那份改为一句话指针或归档 |
| AUD-021 | `FilterPanel` 已“整体退役”，但生成器模板仍在产出 | `DESIGN.md:355` 与 `check-search-toolbar-contract.mjs:37` 明确退役；`templates/listTemplate.ts:36` 仍 import `FilterPanel` | 已有显式豁免（`EXEMPT_SEGMENTS`）+ `generator-quality-contract` 覆盖，属**知情保留**；若要根治，应把模板切到 `SearchToolbar` |

### 4.3 过度设计（C 类）

- **OIDC 脚手架**（AUD-005）：为未决策身份源提前实现 587 行 + 一套环境变量 + 一条迁移，且违反自家合同。典型“为未来需求预付复杂度”。
- **`pkg/contracts.ProtectedGroup` 双入口**（AUD-002）：一个不带租户的“保护组”与一套手写 `Token→Tenant→Casbin` 链并存；前者 base 内无人使用却被下游依赖，属**有害的便利抽象**。
- 未发现其他明显过度设计：`pkg/` 21 个子包均有明确消费者（`capability`、`platformprefs`、`rbacbind`、`maintenance` 均被生产代码引用）。

---

## 5. 安全与多租户审计

### 5.1 认证

| 检查项 | 结论 | 证据 |
| --- | --- | --- |
| Token 存储 | Redis opaque token（非 JWT 自包含） | `pkg/authtoken` |
| 会话校验 | 无效即 401，Redis 故障按无效处理（fail-closed） | `token_middleware.go:120-186` |
| 撤销 | 用户级 + 会话级黑名单，每请求校验；本地 60s 缓存不影响撤销 | `token_middleware.go:236-262`、`InvalidateTokenAuthCache` |
| 空闲超时 | 可配置（`login.session_idle_minutes`，默认 30） | `token_middleware.go:265-276` |
| 登录暴力破解 | 按 `FullPath+ClientIP` 限流（生产 5/min），并落 `system_login_throttle` | `auth/module.go:37-56`、`login_throttle.go` |
| Cookie | `HttpOnly` + `Secure` + `SameSite=Strict` | `pkg/common/http/cookie.go:31-37` |
| CSRF | 双提交 + 精确豁免路径（防前缀扩散） | `csrf_middleware.go:20-66` |
| 密码策略 | 长度下限来自配置 `security.password_min_length`，非硬编码 | `tenant_model.go:70-77` |
| MFA | 登录时首绑 + 验证，challenge 5 分钟过期、单次消费 | `mfa_service.go:83-260` |
| 默认账号 | 非生产 `admin/123456`；生产强制 `PANTHEON_INITIAL_ADMIN_PASSWORD` ≥12 | `user_username.go:93-108`、`admin.initial_password_required` |

未发现 P0 级认证缺陷。

### 5.2 授权

- 四层模型落地：导航（`system_role_menu`）、页面（`pagePerm`）、操作（`perms`）、接口（Casbin）——`menu-contract` 门禁实测 102 个权限、23 条路由。
- Casbin domain 展开正确：全局主体 + `role:<key>@tenant:<id>`，两者不隐式合并（`casbin_middleware.go:68-118`）。
- 引擎未初始化时 fail-closed（`casbin_middleware.go:27-30`）。
- 权限变更缓存：`RefreshSyncMiddleware` + Redis pub/sub 刷新。
- 自助服务白名单 13 条路由签名（`casbin_middleware.go:95-126`），其中 `/api/v1/system/profile`、`/api/v1/system/profile/password` 两条**已无对应路由**（随 legacy 端点下线遗留）。

### 5.3 多租户安全

| 检查项 | 结论 | 证据 |
| --- | --- | --- |
| 上下文可信来源 | header 需平台能力；session claim 由签发门禁盖章；compat 兜底 | `tenant_context_middleware.go:22-44` |
| header 伪造 | 需 `platform_ops` 或全局 Casbin policy，且租户必须存在且 active | `tenant_context_middleware.go:98-135` |
| 平台控制台越权 | `PlatformCapabilityMiddleware` 阻断租户管理员读全量租户表 | `tenant_context_middleware.go:180-200` |
| 数据过滤 | 系统域 11 个 service 全量 `Scoper`；审计/字典/设置/组织/用户/角色/会话均覆盖 | §2.2 |
| 策略读取失败 | **fail-closed**（非管理员拒绝而非 `all`） | `data_scope_middleware.go:103-118` |
| 缓存 key 含租户 | 用户部门缓存 key 含 db 命名空间；token 缓存按 token；模式 flag 读 `tenant_id=0` | `data_scope_middleware.go:140-160` |
| 异步任务 | 后台维护器无租户上下文（仅保留期清理，跨租户按设计） | `pkg/maintenance` |
| 已知缺口 | 业务域路由无租户上下文、业务表无 `tenant_id`（AUD-002） | §2.3 |

### 5.4 常见漏洞面

- **SQL 注入**：核心查询走 GORM 参数绑定；`role_service.go` 与 `menu` 过滤器使用包级常量拼 clause（正是 #477 的修复方向），未发现字符串拼接用户输入。
- **XSS**：CSP + nonce 中间件（`csp_nonce_middleware.go`），前端不使用 `dangerouslySetInnerHTML`（未检出）。
- **SSRF**：`pkg/security/ssrf` 有独立校验器与测试（`validator.go` 268 行），**base 内 0 引用，下游 `pantheon-ops` 5 处引用**（argocd / k8s / observability）。低代码数据源另有就地校验（`generator_datasource_service.go:334-372`，含私网开关 `PANTHEON_GENERATOR_DATASOURCE_ALLOW_PRIVATE`）。**两套并存属分工，不是重复实现**。
- **文件上传**：`pkg/upload` + `system_setting` 上传配置；未见路径穿越（`filepath.IsLocal` 校验见 `scaffold/workspace.go:195`）。
- **日志脱敏**：操作日志对请求体有 guard（`operation_log_request_body_guard_test.go`），敏感设置项加密存储（`setting_crypto.go`）。
- **错误信息**：统一 `common.Fail(code,key)`，后端不直出自然语言（与 `docs/designs/ERROR_CODE_AND_I18N.md` 一致）。
- **CORS**：默认仅 localhost:5173/5174，生产必须显式配置（`cors_middleware.go:15-38`）——默认安全，但**部署文档未强调**，记为文档缺口。

### 5.5 发现：token 会话缓存数据竞争（AUD-026，P2）

- `lookupTokenSessionCache(token)` 在 `RLock` 下返回 `*tokenCachedSession` **共享指针**，锁在返回时即释放。
- `refreshTokenActivity`（`token_middleware.go:181-190`）随后在**锁外**改写 `cached.data.LastActivityAt`；`resolveTokenSession` 也把 `cached.data` 直接交给下游读写。
- 同一 token 的并发请求会同时读写同一 `SessionData` 结构，`go test -race` 在并发同 token 用例下会报 `DATA RACE`。
- **影响**：不构成越权（被改写的只是 `LastActivityAt` 时间戳），最坏结果是空闲超时判定偏移（提前踢出或延迟踢出）。**定级 P2**，因为它是可复现的内存安全缺陷，且 `-race` 门禁因缺并发用例而未覆盖。
- **建议**：缓存存储改为**值拷贝**（返回 `authtoken.SessionData` 副本），或让 `refreshTokenActivity` 持锁改写并避免向调用方暴露内部指针。

---

## 6. 数据库与性能分析

### 6.1 设计

- 表结构统一 `system_*` / `permission_*` / `module_registration` / `tenants` 前缀；主键自增 `BIGINT UNSIGNED`；`created_at/updated_at` 一致（`DATETIME(3)`）。
- 租户列与索引成对出现：`system_user(tenant_id,username)` 唯一、`system_role(tenant_id,role_key)` 唯一、`system_dict_*`、`system_dept/post`、`system_log_login/oper`、`system_user_session` 均有 tenant 索引（迁移 `000013`–`000028` 逐条核对，共 34 处 `tenant_id` 索引）。
- 软删除仅用于用户/角色/部门/岗位/字典等业务实体（5 个模型），日志类表不做软删除——**一致且合理**。
- 迁移安全：全部 up/down 成对；多个迁移使用 `information_schema` + `PREPARE` 实现幂等（`000015/000020/000024/000029`），可重复执行。
- `database/system_init.sql` 顶部有明确 `DEPRECATED` 横幅并列出缺失表（`:1-17`）——处置正确，但 `docs/README.md` 仍指导使用它（AUD-016）。

### 6.2 数据访问

- 列表页批量装配，无 N+1（用户列表已核对；部门/岗位名同样批量）。
- 分页统一 `page/pageSize` 且有上限（`pkg/common/pagination`），导出硬上限 10,000（`pkg/impexp/export_limits.go:5`）。
- 会话列表已改为 SQL 分页 + 专用索引（迁移 `000018`）。
- 读路径不做保留期清理：已迁移到 `pkg/maintenance` 后台调度（`main.go:214-223`），方向正确。
- 潜在风险（未压测，仅静态）：`dashboard_service` 一次请求聚合多表白名单统计，租户数据量增长后可能变慢；目前无缓存，属**可接受且不应提前优化**。

### 6.3 性能结论边界

本轮**未执行任何压力测试或基准对比**（`backend.pid` 陈旧、无可用 DSN）。因此：不宣称性能达标，也不宣称存在性能缺陷。`ci.yml` 的 bench-perf-smoke 只证明“基准可运行”，README 亦已注明非容量基线。

---

## 7. 前端质量与 UI 分析

### 7.1 结构与一致性

- 页面集中在 `src/modules/{system,platform,auth,lowcode}`，通用件在 `src/components`（patterns/table/data-display/feedback/governance），壳层在 `src/core`。
- 组件出口 `components/index.ts` 72 个具名导出，**全部在 `src` 内被引用**（逐名核对，无死导出）。
- 门禁覆盖真实有效：`check:ui-contract`（0 findings / 276 文件）、`check:i18n-hardcode`（237 文件）、`check:system-page-admission`（16 项）、`check:password-toggle`、`check:search-toolbar-contract` 均在 `prebuild` 链上。
- i18n：5 语言 **3020 key 完全对齐，0 缺失 0 冗余**（`audit-i18n-locales.mjs` 实测）；ja/ko 各 188 条与英文相同，属可接受回退。

### 7.2 状态与交互

- 统一状态组件齐备：`PageLoading/PageSkeleton/PageEmpty/PageError/PageForbidden/PageNotFound/PageServerError/PageNetworkError`。
- 路由级权限：`RoutePermissionGuard.tsx` 区分 loading / 可重试错误 / forbidden 三态，行为正确。
- 表格密度、列宽、跨页选择、标准分页均收敛到 `components/table`，未见多套并存。

### 7.3 发现的体验/维护问题

| ID | 问题 | 证据 | 级别 |
| --- | --- | --- | --- |
| AUD-022 | 单文件过大：`I18nList.tsx` 2173 行、`AppLayout.tsx` 2001、`DeptList.tsx` 1813、`OperationLogList.tsx` 1588、`RoleList.tsx` 1326 | `wc -l` | P3（不建议为拆而拆） |
| AUD-021 | 生成页使用已退役的 `FilterPanel` | 见 §4.2 | P3（知情豁免） |
| — | 未发现重复 API 封装：`src/api/*` 6 个文件职责清晰 | 目录核对 | 通过 |

---

## 8. 文档资产治理与代码一致性分析

### 8.1 文档资产总览（全量盘点，非抽样）

| 类别 | 数量（`origin/main`） | 说明 |
| --- | --- | --- |
| 全仓 Markdown | **1,120** | `git ls-tree -r --name-only origin/main \| grep '\.md$'` |
| `docs/` 下 Markdown | **408** | 见下表分布 |
| `.harness/` 受管文件 | **1,313** | 归档为主，活跃任务与 evidence 占少数 |
| YAML/JSON 文档面 | 26 `.yml` + 457 `.json` | 含 `schema/generated`、`.harness` manifest、workflow |
| 根级入口文档 | 11 | README/DESIGN/AGENTS/CLAUDE/CONTRIBUTING/CHANGELOG/SECURITY 及 .en 版 |

`docs/` 分布：`designs` 121、`history` 38、`harness` 118、`reviews` 27、`acceptances` 22、`contracts` 20、`release-notes` 18、根级 25、`archive` 4、`frontend` 4、`runbooks` 3、`operations` 2、`case-studies` 2、`api`/`migrations` 各 1。

### 8.2 元数据与有效性分类（工作区实测）

| 统计项 | 数值 | 说明 |
| --- | --- | --- |
| `docs/` 中带 YAML frontmatter | **284 / 404（HEAD）** | `frontmatter-check.mjs` 实测；main 上为 408 份 docs |
| 全仓 `status: Active` | 258 | 另有 Draft 14、Archived 11、completed 13、Approved 2、无 status 11 |
| 无 frontmatter 的 `docs/` 文档 | **120** | 含 `docs/history` 119、`release-notes` 18、`runbooks` 3、根级 `TENANT_*` 8 等 |
| `updated_at` 超过 60 天的 Active 文档 | **102 / 171** | 按 frontmatter 字段统计（不代表已失效） |
| `docs/designs` 索引缺口 | **18** | 逐个 stem 比对 `docs/designs/README.md` |
| 零入链设计文档 | **1** | `TASK_CENTER_DESIGN`（`PERMISSION_MENU_DECOUPLING` 有 1 处引用、`MULTI_TENANT_DESIGN` 有 26 处） |

> **“未进索引”≠“孤立”**：初版把 18 份索引缺口直接计为 ORPHANED=25，属高估。正确表述是 **18 份索引缺口 + 1 份真孤立**。

**门禁覆盖的真相**：`frontmatter-check.mjs:154-172` 的 `expectedIndexGroup` / `expectedStatuses` 只对 `docs/superpowers/specs/` 与 `docs/archive/{examples,baselines,upgrade}/` 返回非空，因此**其他目录缺 frontmatter 不会报错**；`check-doc-inventory` 只校验 10 个固定条目。所以“检查通过”的实际含义是“已登记文档合规”，不是“全部文档有元数据”。（AUD-010）

### 8.3 过期与不一致清单（逐份核对，均给出代码/配置反证）

| # | 文档 | 声明/状态 | 当前事实 | 判定 | 置信度 |
| --- | --- | --- | --- | --- | --- |
| 1 | `README.md:11-15` | “当前 foundation release v0.17.8” | `origin/main:VERSION` = 0.18.2；tag `pantheon-base-v0.18.2`；`RELEASE_NOTES_v0.18.2.md` **存在（98 行）** | OUTDATED | 高 |
| 2 | `README.md:31` | “配额强制” + “完整数据隔离” | 无任何 quota 校验写路径（§3.2）；业务域无租户建模（§2.3） | CONFLICTED（与同文件 `:20` 自相矛盾） | 高 |
| 3 | `README.md:262` | “尚未提供 CONTRIBUTING 或 Issue Templates” | `CONTRIBUTING.md` 存在；`.github/ISSUE_TEMPLATE/` 3 个模板 | OUTDATED | 高 |
| 4 | `README.md:238-249` | “Quality Gates 最后验证 2026-10-03 … 0 个阻塞性失败” | 静态快照；无对应 run 链接 | PARTIALLY_OUTDATED | 中 |
| 5 | `DESIGN.md:22,87,375` | 多租户属“未来/P2 演进/不在租户模型未明确前提前实现” | 租户已在系统域完整交付（30 迁移、`TENANT_CONTRACT_V1`、控制台、套餐、决策记录 2026-10-15/27/28） | **CONFLICTED（核心架构文档）** | 高 |
| 6 | `DESIGN.md:75-86` | 4.3 以“当前 `system/iam/user/user_service.go` 同时承担认证”为拆分理由 | auth 已物理拆分（`modules/auth/`），理由已过时 | PARTIALLY_OUTDATED | 高 |
| 7 | `docs/README.md:134,151-157` | 手动执行 `database/system_init.sql` 建库；`go run ./backend/cmd/server` | 该 SQL 自述已废弃且缺 13 张表；`backend/` 才是 Go module 根，根目录无法 `go run` | **OUTDATED（可执行性错误）** | 高 |
| 8 | `docs/designs/README.md`（`updated_at: 2026-06-27`） | 声称“主题文档均已补齐 .en companion”；索引 45/62 | 5 份无 `.en.md`；索引**缺口 18 份**（含 `MULTI_TENANT_DESIGN`） | OUTDATED | 高 |
| 9 | `docs/designs/TENANT_RESOURCE_SCOPE_MATRIX.md:120-131` | “租户识别方式全部待决策，禁止按示例实现” | `X-Tenant-Id` + session claim 已实现并冻结 | OUTDATED | 高 |
| 10 | 同上 `:62` | `system_dept/post` 归类 `platform-global（倾向）` | 两者均带 `tenant_id` 并走 `tenant.Scoper`（`dept_service.go:28`、`post_service.go:26`） | CONFLICTED | 中高 |
| 11 | `docs/contracts/SYSTEM_AUTH_CONTRACT.md:168` | “未明确身份源前不提前实现 SSO/OIDC” | `modules/auth/oidc` 已实现 587 行（未接线） | CONFLICTED（代码违约，非文档过期） | 高 |
| 12 | `docs/harness/DOCUMENT_FRONTMATTER_SPEC.md` | 独立的 frontmatter 规范 | 与 contracts 两份并存 | DUPLICATED | 高 |
| 13 | `docs/designs/ALERT_MONITORING_CENTER_DESIGN.md` 等 6 份 | `status: Active` | 代码中无 alert/approval/notice/report/scheduler/task-center 任何实现 | CONFLICTED（状态语义错误） | 高 |
| 14 | `docs/harness/tasks/*.task.md` | 68 份 `status: Active` | 对应 `.harness/archive/**` 证据已归档 | OUTDATED | 高 |
| 15 | `docs/api/TENANT_API.md` | 13 个端点 + 错误码表 | 与 `system_modules.go:290-306` 逐条一致 | **CURRENT（通过）** | 高 |

### 8.4 重复与冲突清单

**跨仓库重复（base ↔ ops）**：以下 8 份文件在两仓**字节相同**，且属 base 架构/评审内容，违反 workspace 规则“Ops 只做 `business/*`”：

```
docs/TENANT_FULL_FLOW_REVIEW_2026-10-14.md
docs/TENANT_INITIALIZATION_FINDINGS_2026-10-13.md
docs/TENANT_INITIALIZATION_GUIDE.md
docs/TENANT_LOGIN_INIT_EVALUATION_2026-10-15.md
docs/TENANT_MATURITY_FINAL_2026-09-29.md
docs/TENANT_REVIEW_SUMMARY_2026-09-29.md
docs/TENANT_SCOPE_FREEZE_DECISION_2026-10-15.md
docs/TENANT_SYSTEM_REVIEW_2026-09-29.md
```

**仓内冲突对**：

| 冲突 | 双方 | 权威判定 |
| --- | --- | --- |
| 建库方式 | `README.md:104-108`（迁移 + seed） vs `docs/README.md:151`（`system_init.sql`） | 以 README + `system_init.sql:1` 的 DEPRECATED 横幅为准 |
| 租户现状 | `DESIGN.md:375` vs `TENANT_CONTRACT_V1` + 代码 | 以合同 + 代码为准，DESIGN 需补章 |
| 配额/隔离 | `README.md:31` vs `README.md:20` + 代码 | 以代码为准：配额不强制；隔离仅系统域 |
| 元数据规范 | harness 版 vs contracts 版 | 以 `frontmatter-check.mjs` 实际实现（contracts 版）为准 |
| OIDC | 合同“不提前实现” vs 已存在实现 | 以合同为准，需删除或补齐 |

**历史材料不判为无效**：`docs/history/2026-09/**`（38 份）、`docs/reviews/pantheon-base-review-*.md`（7 份模型交叉评审）、`docs/reviews/*_2026-09/10-21` 验收报告均具追溯价值，`docs/README.md:120-124` 也已显式标注“被 2026-10-21 验收继任确认”。**保留（HISTORICAL）**。

### 8.5 文档与代码一致性矩阵（核心文档逐项）

| 文档 | 对应代码/配置 | 状态 | 一致性 | 主要问题 | 建议 |
| --- | --- | --- | --- | --- | --- |
| `README.md` | `VERSION`、`backend/go.mod`、`frontend/package.json` | OUTDATED | 部分 | 版本表落后 4 个 release；配额/隔离/community files 失实 | UPDATE |
| `DESIGN.md` | 全后端 | CONFLICTED | 部分 | 缺租户章；§4.3 理由过时 | UPDATE |
| `AGENTS.md` / `CLAUDE.md` | `.agents/`、`docs/harness/` | CURRENT | 一致 | 无实质冲突 | KEEP |
| `CONTRIBUTING.md` | `.githooks/`、`.gitmessage` | CURRENT | 一致 | README 却称未提供 | KEEP（改 README） |
| `docs/README.md` | `docs/**` | PARTIALLY_OUTDATED | 部分 | §6 建库/启动命令失效；根级 `TENANT_*` 未入索引 | UPDATE |
| `docs/designs/README.md` | `docs/designs/**` | OUTDATED | 低 | 遗漏 18 份；`.en` 声明不成立 | UPDATE |
| `docs/designs/BACKEND.md` | `backend/**` | UNVERIFIED | 待核 | 2026-04-17 起未更新 | REVIEW |
| `docs/designs/DATABASE.md` | `migrations/**` | UNVERIFIED | 待核 | 同上 | REVIEW |
| `docs/designs/FRONTEND.md` / `FRONTEND_UI_SPEC.md` | `frontend/src/**` | PARTIALLY_OUTDATED | 部分 | `FilterPanel` 退役与生成器模板张力 | UPDATE（说明豁免） |
| `docs/designs/PERMISSION_MODEL.md` | `iam/{role,menu,permission}` | UNVERIFIED | 待核 | 2026-04-29 未更新 | REVIEW |
| `docs/designs/TENANT_RESOURCE_SCOPE_MATRIX.md` | `pkg/tenant`、`iam/*_service.go` | CONFLICTED | 低 | §4 与 dept/post 归类失效 | UPDATE |
| `docs/designs/MULTI_TENANT_DESIGN.md` | 租户实现 | HISTORICAL | 自述放弃权威 | 无 frontmatter，自述“非 V1 承诺” | ARCHIVE |
| `docs/contracts/TENANT_CONTRACT_V1.md` | `pkg/tenant`、中间件、租户模块 | CURRENT | 高 | 未约束业务域路由入口（AUD-002） | KEEP + 补条款 |
| `docs/contracts/SYSTEM_AUTH_CONTRACT.md` | `modules/auth/**` | PARTIALLY_OUTDATED | 部分 | OIDC 条款与代码冲突 | UPDATE |
| `docs/contracts/{IAM,ORG,CONFIG}_CONTRACT.md` | 对应模块 | UNVERIFIED | 待核 | 均为 2026-04/05 基线 | REVIEW |
| `docs/api/TENANT_API.md` | `system_modules.go:290-306` | CURRENT | 高 | 无 | KEEP |
| `docs/migrations/COMPAT_TO_MULTI_UPGRADE.md` | 迁移 000013–000030 | UNVERIFIED | 待核 | 需一次演练验证 | REVIEW |
| `docs/DEPLOYMENT_GUIDE.md` | `.env.example`、`docker-compose.yml`、`Dockerfile` | CURRENT | 中高 | CORS 生产配置未强调 | UPDATE（小） |
| `docs/VERSION_MANAGEMENT_GUIDE.md` | `scripts/foundation-release/**` | CURRENT | 高 | 与 release 脚本口径一致 | KEEP |
| `docs/harness/*.md`（38 份） | `.harness/`、`scripts/harness/**` | MIXED | 中 | 1 份重复规范 + 7 份 2026-06 未动 | MERGE / REVIEW |
| `docs/harness/tasks/*.task.md` | `.harness/archive/**` | OUTDATED | 低（68/82） | 完成态未回写 | UPDATE（批量归档） |
| `docs/acceptances/*.md` | 各域实现 | UNVERIFIED | 待核 | 多为模板与清单 | REVIEW |
| `docs/runbooks/*.md` | 运维脚本 | UNVERIFIED | 待核 | 未执行演练 | REVIEW |
| `docs/reviews/*.md`（27 份） | 历史决策 | HISTORICAL | — | 有效追溯材料 | KEEP |
| `docs/history/**`（38 md） | `.harness/archive/**` | HISTORICAL | — | 与归档互补，无重复 | KEEP |
| `SECURITY.md` | 安全策略 | CURRENT | 一致 | 无 | KEEP |
| `.harness/STATUS.md` | 发布事实 | PARTIALLY_OUTDATED | 中 | v0.18.1 及更早记录详实；**缺 v0.18.2 章节**（main 已发 0.18.2），已在该文件头登记为已知缺口 | UPDATE |

### 8.6 AI Agent 指令一致性

- **指令源**：workspace `agents.md` → `pantheon-base/AGENTS.md` → `CLAUDE.md` → `docs/README.md` → `docs/harness/{DELIVERY_WORKFLOW,AI_QUALITY_GOVERNANCE}.md` → `.agents/skills/**` → `docs/acceptances/AGENT_EXECUTION_CHECKLIST.md`。
- **结论：未发现互相矛盾的硬规则。** `CLAUDE.md` 是对 `AGENTS.md` 的英文摘要 + 维护者三触点契约，口径一致；`.agents/skills` 与 `docs/harness` 分工明确（前者操作、后者方法）。
- **存在的风险**：①`docs/harness/DOCUMENT_FRONTMATTER_SPEC.md` 与 contracts 版并存，Agent 可能按旧版字段写文档；②`docs/designs/README.md` 索引缺口会让 Agent 读不到 18 份设计；③`SYSTEM_AUTH_CONTRACT` 的 OIDC 禁令与既有代码并存，Agent 无法判断该删还是该补。三条都应在冻结前一次性收敛。

### 8.7 推荐的权威文档体系（冻结后）

```
入口层（必须维护）
  README.md / README.en.md
  DESIGN.md（补齐租户章）/ DESIGN.en.md
  AGENTS.md / CLAUDE.md
  docs/README.md（唯一索引）

合同层（唯一权威，冲突时以此为准）
  docs/contracts/{PLATFORM,TENANT_V1,SYSTEM_AUTH,SYSTEM_IAM,SYSTEM_ORG,SYSTEM_CONFIG}_CONTRACT.md

实现层（按域，随代码更新）
  docs/designs/<DOMAIN>_DESIGN.md
  docs/api/TENANT_API.md

流程层
  docs/harness/{HARNESS_ENGINEERING_CONTRACT,AI_QUALITY_GOVERNANCE,FAILURE_REGISTRY}.md
  docs/acceptances/*（模板与清单）

运维层
  docs/DEPLOYMENT_GUIDE.md、docs/operations/RUNBOOK.md、docs/runbooks/*

历史层（只读，不再更新）
  docs/history/**、docs/reviews/*2026-*、docs/archive/**、.harness/archive/**
```

### 8.8 文档体系最终评估（13 问，计数均来自实测）

| # | 问题 | 回答 |
| --- | --- | --- |
| 1 | 数量与类型 | 1,120 md（`docs/` 408）；类型覆盖架构/设计/合同/验收/评审/运维/发布/Harness/历史 |
| 2 | 当前有效 | 258 份标记 `Active`；本轮抽核 15 份核心文档，8 份一致、7 份存在问题 |
| 3 | 已过期 | 6 类：README 版本表、`docs/README` §6、`designs/README`、`TENANT_RESOURCE_SCOPE_MATRIX`、68 份 task packet、DESIGN §4.3 |
| 4 | 部分过期 | 2 份（`SYSTEM_AUTH_CONTRACT`、`DESIGN`） |
| 5 | 与代码不一致 | 5 份（README 两处、DESIGN、`designs/README`、`TENANT_RESOURCE_SCOPE_MATRIX`） |
| 6 | 存在冲突 | 5 组（§8.4） |
| 7 | 重复 | 仓内 4 组 + 跨仓 8 份 + 元数据规范 3 合 1 |
| 8 | 建议归档 | 68 份 task packet + 6 份未实现 Active 设计 + 1 份 `MULTI_TENANT_DESIGN` |
| 9 | 建议删除 | **0 份**。合并后的 `DOCUMENT_FRONTMATTER_SPEC` 与零入链的 `PERMISSION_MENU_DECOUPLING` 均标 REVIEW，**不自动删** |
| 10 | 建议补充 | DESIGN 租户章、README 版本表、`docs/README` 启动命令、`designs` 索引 18 条、CORS 生产配置说明 |
| 11 | 权威文档是否清晰 | **否**（元数据规范 3 份并存；设计索引缺口 18 份） |
| 12 | Agent 指令是否冲突 | 硬规则不冲突；存在 3 条“解读歧义”（§8.6） |
| 13 | 是否适合冻结 | 经一轮文档治理后可冻结；当前不适合直接冻结 |

---

## 9. Harness Engineering 与工程治理

### 9.1 结构（通过，但覆盖率有缺口）

- 指令：`AGENTS.md`、`CLAUDE.md`、`.agents/{skills,adapters,prompts}`。
- 机器状态：`.harness/{STATUS,ARCHIVE}.md`、`tasks/`、`evidence/`、`state/`。
- 方法层：`docs/harness/` 36 份 + `scripts/harness/` **26 个 `check-*.mjs`**（目录内共 32 个 `.mjs`，另 6 个是 `generate-pr-body`、`scaffold-graph-review`、`sort-utils`、`triage-base-drift`、`upstream-root`、`build-graph-review-import` 等辅助脚本）。
- 发布：`scripts/foundation-release/**` + 3 个 workflow（cut/verify/backfill），`releases/*/manifest.json` + `verification-summary.json`。
- 分层刻意不合并：`docs/harness/` 是可评审合同，`.harness/` 是机器状态与证据——**判断为合理，不建议合并**。

### 9.2 流程合理性

| 判断 | 结论 | 依据 |
| --- | --- | --- |
| 检查重复 | 存在 1 处实质重复 | `go test -race` 在 `ci.yml:167` 与 `quality.yml:384` 各一次（AUD-015）；`golangci-lint` 两次但 ci 侧显式降级为 advisory（`ci.yml:298-303` 注释说明），属知情设计 |
| **checker 是否都在跑** | **否：11/26 未接线（42%）** | 未接入任何 npm script 或 workflow：`check-audit-coverage`、`check-backend-dto-contract`、`check-backend-response-contract`、`check-doc-frontmatter`、`check-evidence`、`check-feature-ledger`、`check-graph-review`、`check-inheritance-contract`、`check-permission-contract`、`check-review`、`check-runtime-evidence`（AUD-027） |
| 无价值 checker | 未发现空断言 | 抽查 `check-ui-quality-gate`、`check-structure-contract`、`check-generated` 均有实际断言与测试 |
| 流程过复杂 | 偏重但可控 | `prebuild` 链 12 条检查、`quality.yml` 995 行；对单人维护是负担，但每条都对应一次历史事故 |
| Evidence 是否过多 | 偏多 | `.harness` 1,313 文件；按月归档机制已存在（`archive/2026-07..10`），结构可接受 |
| 规则冲突 | 1 处 | frontmatter 规范 3 份（AUD-008） |
| 是否降低错误率 | 有正面证据 | 失败登记 `FR-016/FR-017/FR-021` 与 ratchet 规则（`docs/harness/AI_QUALITY_GOVERNANCE.md §5.4`）能追溯到具体事故 |

> **初版此处结论错误**：初版写“未发现无价值 checker”，只核了“checker 内部有没有断言”，没核“它是否被执行”。11 个未接线 checker 中包括 `check-permission-contract`、`check-feature-ledger`、`check-inheritance-contract` 这类与核心契约直接相关的检查——**写了但没跑，等于没有**。这正是 FR-016/FR-017 记录过的同一失败模式再次出现。

### 9.3 CI/门禁

- 12 个 workflow，职责清晰：CI（快检/单测/构建/边界）、Quality（文档/契约/测试/重复度/smoke 分流）、Security（依赖/secret/zizmor/CodeQL）、Release Gate（候选检查 + 三个外部 gate + 汇总）。
- 测试分流合理：`change-scope` → `governance_only` 时跳过重活；PR smoke 按改动模块选择。
- 门禁有效性：本地实测 4 条前端门禁 + 3 条文档门禁 + duplication 均绿，说明脚本可执行、无空转。
- 未验证：SonarCloud、CodeQL、gitleaks、zizmor 的实际通过状态（需 GitHub 凭证）；README 的“最后验证 2026-10-03”无法复核（AUD-024）。

---

## 10. Pantheon Ops 下游兼容性

| 检查项 | 结论 | 证据 |
| --- | --- | --- |
| 消费模型 | **源码复制，非 Go module 依赖**：`sharedPaths`（52 项）+ overlay | `pantheon-ops/backend/go.mod` 声明**相同** module 路径；`foundation-release.lock.json` |
| 锁定版本 | `pantheon-base-v0.18.2` / `a44ae4e0`，`lockedAt 2026-10-31T01:07:28Z` | 同上 |
| 依赖的 base 能力 | `pkg/common`(59)、`pkg/database`(19)、`pkg/impexp`(14)、`pkg/contracts`(8)、`pkg/security/ssrf`(5)、`pkg/logging`(4)、`pkg/testmysql`(18) | `grep` 统计 |
| 是否依赖内部实现 | **是，且依赖的是违约的入口** | ops **14 处**使用 `contracts.{DataScoped,Protected}Group` |
| 公共 API 稳定性 | 前端符号曾被误删（v0.18.1 恢复 `isForbiddenRequestError`），已建立 workspace 级检索规则 | `.harness/STATUS.md` v0.18.1 段 |
| 权限/菜单/审计/i18n 复用 | 可复用 | 均由 base `sharedPaths` 覆盖，ops 只加 business overlay |
| 升级破坏风险 | 中 | 删除/改签名的 `pkg/*` 符号会直接打断 ops 构建（无 Go module 边界，失败显式） |
| base 是否为 ops 引入特殊逻辑 | 未发现 | `backend/modules/business/*` 在 base 中只有注册骨架（440 行） |

**结论**：作为“可复用底座”的定位成立；**最大兼容性风险不是 API 破坏，而是 AUD-002**——base 提供给下游的路由助手本身不满足 base 自己的租户合同，且下游对它的依赖面是 14 处而非 8 处。

---

## 11. P0–P3 问题汇总（合并后，共 28 项）

> 编号沿用初版 `AUD-001…025` 以保持与既有任务包的可追溯性；本轮新增 `AUD-026/027/028`。
> 前序复核稿中的“14 个 P2 / 12 个 P3”是更粗的归并口径，**本表为权威粒度**。

### P0（严重，必须立即处理）

**无。**

### P1（重要，阻断冻结，3 项）

| ID | 分类 | 位置 | 问题 | 证据 | 影响 | 建议 | 风险 | 置信度 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- |
| AUD-001 | 功能完整性 | `frontend/src/modules/lowcode/generator/backendGenerator.ts:712-713,761`；`dynamic_module_service.go:323-333` | 生成的后端模块 **四重缺陷**：import 路径错、`TokenAuthMiddleware()` 缺 `rdb`、无 SQL 迁移、未设 `BootstrapFunc` 导致默认迁移模式不建表 | 实跑生成 + `go list` 报 `not in std`；`RegisterBackendModules` 只调 `Bootstrap`；`FuncModule.Bootstrap` 空实现返回 nil | 生成→重启激活必然失败，且注册状态仍推进（静默失败） | 修模板（路径 + 传参 + 生成迁移或设 `BootstrapFunc`）；注册前置加 `go build` 校验 | 低（仅影响新生成模块） | 高 |
| AUD-002 | 架构/安全/文档 | `backend/pkg/contracts/route_groups.go:15-28`；`internal/scaffold/contract.go:74`；`README.md:31` | 业务路由助手不挂租户上下文，业务域整体无租户建模，而入口文档宣称“完整数据隔离” | `tenant.FromGin` 无值返回 nil；`WithTenantScope(nil)` 不过滤；ops 14 处调用；业务域 `TenantID` 命中 0（`tenant_id` 仅 1 处测试夹具） | multi 模式下业务域不解析租户；文档承诺不可兑现 | **已裁定方案 C**：合同 §2.6 显式声明业务域不参与隔离 + README/MODULE_CONTRACT 对齐 + 测试断言（见 §16） | 中（需决策 + 双仓协同） | 高 |
| AUD-016 | 文档可执行性 | `docs/README.md:134,151-157` | 建库与启动命令均已失效 | `system_init.sql:1-17` 自述 DEPRECATED 且缺 13 张表；`backend/` 才是 module 根 | 新维护者按文档操作必然失败 | UPDATE 两段 | 低 | 高 |

### P2（一般，同一轮内关闭，18 项）

| ID | 分类 | 位置 | 问题 | 建议 | 置信度 |
| --- | --- | --- | --- | --- | --- |
| AUD-003 | 文档准确性 | `README.md:11-15,31,262` | 版本表落后 4 个 release；“配额强制/完整数据隔离”失实；“未提供 CONTRIBUTING/Issue 模板”失实 | UPDATE 三段 | 高 |
| AUD-004 | 文档/架构 | `DESIGN.md:22,87,375` | 核心架构文档仍把多租户写成“未来/不要提前实现”，且无租户章节 | 增补 `§2.5 多租户`，改写 §3.3/§9 | 高 |
| AUD-005 | 过度设计 | `backend/modules/auth/oidc/*`（587 行） | 不可达 OIDC 脚手架，违反 `SYSTEM_AUTH_CONTRACT:168` | 维护者决策：删 / 移入实验目录 / 补齐 | 高 |
| AUD-006 | 文档生命周期 | `docs/harness/tasks/*.task.md` | 82 份中 68 份仍 `Active`，对应证据已归档 | 批量置 `Archived` 并加 checker | 高 |
| AUD-007 | 文档 | `docs/designs/README.md` | 索引缺口 18 份；`.en` 声明不成立；`updated_at` 2026-06-27 | UPDATE | 高 |
| AUD-008 | 文档冗余 | `docs/harness/DOCUMENT_FRONTMATTER_SPEC.md` | 与 contracts 两份规范三足鼎立 | MERGE → contracts 权威 | 高 |
| AUD-009 | 文档 | `docs/designs/TENANT_RESOURCE_SCOPE_MATRIX.md:62,120-131` | §4 已实现却写“禁止实现”；dept/post 归类与代码相反 | UPDATE | 高 |
| AUD-010 | 工程治理 | `scripts/frontmatter-check.mjs:154-172` | 元数据门禁只覆盖 archive/superpowers；120 份 docs 无元数据 | 扩大覆盖或显式声明“不覆盖” | 高 |
| AUD-011 | 跨仓文档 | base/ops 8 份逐字节相同 | 违反“Ops 只做 `business/*`” | 从 ops 删除并指向 base | 高 |
| AUD-012 | 无效代码 | `hooks/usePagination.ts`、`useRequest.ts`、`scaffold/engine.go` | 零调用（合计 ~214 行） | 删除（**先做 workspace 级检索**） | 高 |
| AUD-013 | 无效 schema | `permission_role_data_scope_policy` | 与 `system_role_data_scope` 重复，无功能读写 | 新迁移 drop（保留 down） | 中高 |
| AUD-014 | 重复 API | `/system/login|refresh|logout|user/info` | 与 `/auth/*` 重复，仅 CSRF 豁免表引用 | 公告弃用后下线 | 中高 |
| AUD-015 | 工程成本 | `ci.yml:167` / `quality.yml:384` | 同一 `go test -race` 跑两遍 | 合并或条件化 | 高 |
| AUD-017 | 文档状态语义 | 6 份未实现设计标 `Active` | alert/approval/notice/report/scheduler/task-center | 改 `Draft`/`Archived` | 高 |
| AUD-018 | 孤立文档 | `docs/designs/TASK_CENTER_DESIGN.md` | 全仓零入链 | ARCHIVE 或并入索引 | 高 |
| AUD-019 | 索引 | `docs/TENANT_*.md`（8 份） | 位于 `docs/` 根，未进 `docs/README.md` | 移入 `docs/reviews/` 或补索引 | 高 |
| AUD-026 | 并发正确性 | `internal/middleware/token_middleware.go:181-190` | 缓存返回共享指针，锁外改写 `LastActivityAt`，存在数据竞争 | 改为值拷贝或持锁改写 | 高 |
| AUD-027 | 工程治理 | `scripts/harness/check-*.mjs` | 11/26 checker 未接入任何 script/workflow | 接线或删除并登记 | 高 |

### P3（优化建议，7 项，不立项）

| ID | 问题 | 建议 |
| --- | --- | --- |
| AUD-020 | `readRoleKeysFromContext` 仅测试使用 | 删除包装 |
| AUD-021 | 生成页使用已退役 `FilterPanel`（有显式豁免） | 模板切 `SearchToolbar`，或保留并写明设计意图 |
| AUD-022 | 4 个 1300–2200 行的页面组件 | **不建议**为拆而拆；下次功能改动时顺手分层 |
| AUD-023 | `lowcode/dynamicmodule` 直接 import `system/i18n` | 可接受；若出现第三个跨域依赖再抽契约 |
| AUD-024 | README 引用 2026-10-03 的门禁快照 | 改为链接到 run 或删除时间点 |
| AUD-025 | 102 份 Active 文档 >60 天未更新 | 建立“Active 需 90 天内复核”规则，不逐份处理 |
| AUD-028 | `main.go:70` 的 `_, cancel := context.WithTimeout(...)` 未使用 ctx 也未在函数返回前调用 cancel | 删除或改为真正传入下游 |

### 明确判定为“通过（D）”的能力（不建议改动）

1. 认证会话链路（token/黑名单/空闲超时/撤销/限流/审计）——设计克制、fail-closed 明确。
2. MFA 全链路，含首绑下发 secret 与前端渲染。
3. 租户初始化事务 + 生命周期守卫 + 平台能力门禁。
4. 数据权限中间件 fail-closed 语义与 `system_role_data_scope` 单表模型。
5. 审计的租户 scope 与 `tenantIdFilter` 越权拦截。
6. 用户列表批量装配（无 N+1）与 10k 导出硬上限。
7. 5 语言 3020 key 的对齐与 12 条 `prebuild` 门禁。
8. `docs/api/TENANT_API.md` 与真实路由逐条一致。
9. `business` 与 `system` 的编译期隔离。
10. 迁移幂等写法（`information_schema` + `PREPARE`）。
11. `pkg/security/ssrf` 为下游保留（D 类，禁止删除）。

---

## 12. 整改建议与优先级

### 12.1 必须立即处理（阻断冻结）

1. **AUD-001**：修生成器模板（import 路径、`TokenAuthMiddleware(database.RDB)`、生成迁移或补 `BootstrapFunc`），并在 `validateGeneratedModuleSource` 加编译校验。需一次真实“生成 → `go build` → 建表”往返。
2. **AUD-002**：决定“业务域是否纳入租户模型”并落到合同与 README；若纳入，与 ops 同批升级。
3. **AUD-016**：修正 `docs/README.md` 的建库与启动命令。

### 12.2 应该处理（同一轮内关闭）

4. **AUD-003 / AUD-004**：README 三处失实 + DESIGN 租户章。
5. **AUD-006 / AUD-017 / AUD-018 / AUD-019**：一次文档归档动作（68 task packet + 6 设计状态 + 1 孤儿 + 8 根级索引）。
6. **AUD-007 / AUD-008 / AUD-010**：`designs/README` 索引 18 条、元数据规范三合一、扩大门禁覆盖或声明边界。
7. **AUD-009**：scope 矩阵 §4 与 dept/post 归类。
8. **AUD-011**：清除 ops 侧 8 份重复 base 文档。
9. **AUD-012 / AUD-013 / AUD-014**：死代码、死表、legacy 端点（死表需一条新迁移；legacy 端点需公告期）。
10. **AUD-005**：OIDC 三选一，写入决策记录。
11. **AUD-026**：token 缓存数据竞争。
12. **AUD-015 / AUD-027**：CI 去重 + 11 个 checker 接线。

### 12.3 可以优化（冻结后按需）

- AUD-020、AUD-021、AUD-022、AUD-023、AUD-024、AUD-025、AUD-028。

### 12.4 明确不建议处理

| 项 | 理由 |
| --- | --- |
| **删除 `pkg/security/ssrf`** | base 内 0 引用但下游 5 处引用；删了会打断 `pantheon-ops` 构建（**初版曾建议删除，此处明确否决**） |
| 拆分 `I18nList.tsx` / `AppLayout.tsx` 等大组件 | 无功能收益，回归风险高；按“下次触碰时顺手拆”处理 |
| 为 dashboard 加缓存 | 无实测瓶颈，属提前优化 |
| 把 `system` 对 `auth/session` 的单一依赖改成端口 | 收益远小于改动面；记录为已知耦合即可 |
| 合并 `docs/harness` 与 `.harness` | 两者职责不同（可审阅文档 vs 机器产物），文档已说明 |
| 删除历史评审/验收报告 | 有追溯价值，属 HISTORICAL |
| 补齐 `docs/designs` 全部 `.en.md` | 双语维护成本高于收益；改索引声明即可 |
| 把 P3 观察项包装成整改任务 | 会以“整改”为名继续增加维护面 |

---

## 13. 最终验收意见

### 结论：**B — 完成少量整改后进入维护冻结阶段**

> **执行状态（2026-11-01）**：整改已完成——9/9 子包 `completed`，7 项冻结门槛全部达成（§16.4）。
> 结论仍为 B（这是对本项目成熟度的判断，不因整改完成而变成 A：A 的含义是"零整改即可冻结"）。
> 「进入维护冻结」这一动作待维护者宣告，见 §16.6 末尾。

依据：

1. **核心能力可靠**：认证、授权、租户（系统域）、数据权限、审计、i18n、菜单契约均有实现 + 测试 + 门禁三重覆盖；构建、vet、单测、类型检查、前端 4 条门禁、文档 3 条门禁、重复度门禁在本次审计中**实际执行并全部通过**（§14）。
2. **不存在 P0**：未发现可直接利用的越权、注入、凭证泄漏或数据破坏路径；安全链路的关键失败点均 fail-closed。
3. **P1 共 3 项，全部“有限且可定位”**：一项功能面完全不可用（生成器）、一项能力承诺与实现不一致（业务域租户语义）、一项文档步骤必然失败（`docs/README`）。没有一项需要架构重写或大规模重构。
4. **不具备 C 的条件**：C 的前提是“重要架构/安全/功能完整性/工程质量问题需要继续整改”。本项目架构边界真实成立（business↛system、无循环依赖、组合根单一），安全主链路完整，测试密度 0.78，门禁执行力在同类项目中属良好；问题集中在**收口与同步**，不集中在**设计与正确性**。
5. **也不具备 A 的条件**：生成器产出不可编译且不建表、业务域租户语义与 README 承诺相反、入口文档步骤失效——三条都会直接误导下一位维护者或下游消费者，因此不能“零整改冻结”。

### 冻结门槛（7 项，逐项对应任务包）

| # | 门槛 | 对应任务包 |
| --- | --- | --- |
| 1 | 生成器“生成 → `go build` → 建表”往返证据（AUD-001） | `2026-10-31-audit-p1-lowcode-generator-build` |
| 2 | 业务域租户语义有明确结论并落到合同 + README（AUD-002） | `2026-10-31-audit-p1-business-route-tenant-context` |
| 3 | 入口文档可执行性与失实修正（AUD-003/004/007/016/024） | `2026-10-31-audit-p1-entry-doc-truthfulness` |
| 4 | 文档归档轮完成（AUD-006/011/017/018/019） | `2026-10-31-audit-p2-doc-lifecycle-archive` |
| 5 | 元数据规范三合一 + scope 矩阵 + 门禁边界（AUD-008/009/010） | `2026-10-31-audit-p2-doc-metadata-and-index` |
| 6 | 死代码/死表/legacy 端点处理完毕（AUD-012/013/014） | `2026-10-31-audit-p2-dead-code-legacy-surface` |
| 7 | token 缓存竞争修复 + CI 去重 + checker 接线（AUD-026/015/027） | `2026-10-31-audit-p2-token-cache-race`、`2026-10-31-audit-p2-ci-checker-wiring` |

> OIDC（AUD-005）**不列入冻结门槛**：它是不可达代码，无运行时风险，允许作为“冻结后接受的 gap”显式登记；但必须有一条决策记录，避免下一位维护者反复猜测。

完成上述 7 项后，可进入维护冻结：只接受安全修复、依赖升级与 foundation release 补丁，不再新增功能面。

---

## 14. 验证记录与局限

### 14.1 本轮实际执行的验证（只读）

| 命令 | 结果 |
| --- | --- |
| `git status` / `git log` / `git show origin/main:VERSION` | 见 §1.1、§1.3 |
| `cd backend && go build ./...` | **通过** |
| `cd backend && go vet ./...` | **通过**（无输出） |
| `cd backend && go test -short ./...` | **通过，0 FAIL**（43 个包） |
| `cd backend && go list pantheon-base/internal/middleware` | 失败（AUD-001 证据） |
| `cd frontend && npx tsc -b` | **通过** |
| `npm run check:menu-contract` | 通过（19 菜单/23 路由/102 权限/23 组件键） |
| `npm run check:i18n-hardcode` | 通过（237 文件） |
| `npm run check:ui-contract` | 通过（0 findings / 276 文件） |
| `npm run check:system-page-admission` | 通过（16 项） |
| `node scripts/harness/check-doc-links.mjs --strict` | 通过（0 findings） |
| `node scripts/harness/check-doc-inventory.mjs --strict` | 通过（0 findings） |
| `node scripts/frontmatter-check.mjs` | 通过（404 docs / 284 with frontmatter） |
| `node scripts/check-duplication.mjs` | 通过（PASS） |
| `node frontend/scripts/audit-i18n-locales.mjs` | 5 locale × 3020 key，0 缺失 |
| `node frontend/scripts/export-generated-module.mjs <临时 schema>` | 成功生成 10 文件 / 74,509 字节；暴露 AUD-001 四重缺陷 |
| 逐文件比对 26 个 `check-*.mjs` 与 script/workflow 引用 | 11 个未接线（AUD-027） |
| `git cat-file -e origin/main:docs/release-notes/RELEASE_NOTES_v0.18.2.md` | 存在（98 行），撤回初版“缺失”结论 |

### 14.2 未执行及原因（必须作为显式 gap）

| 未执行项 | 原因 |
| --- | --- |
| 真实 API 端到端验证（登录/租户/审计） | 本机 MySQL 与 Redis 端口可达但无可用凭据（`PANTHEON_DSN` 未配置），未启动后端 |
| `go test -race ./...`（完整集） | 需要 MySQL/Redis 测试库；本轮只跑 `-short`（**这也是 AUD-026 未被门禁捕获的原因**） |
| Playwright smoke（system/platform/business） | 需要运行中的前后端与已初始化数据库 |
| `npm run build` 全链路 | `prebuild` 会执行 `patch:arco-react19` 改写 `node_modules`，为避免改动本地依赖状态而只跑 `tsc -b` + 4 条检查 |
| SonarCloud / CodeQL / gitleaks / zizmor 结果 | 需 GitHub 凭证与远端 run |
| 数据库真实结构核对 | 无 DB 访问权；表结构结论均来自 `migrations/*.sql` 与 GORM 模型，**未声称已验证线上库** |
| 压力测试/性能基线 | 无代表性数据与压测环境 |
| 生成器产物在仓库内落盘编译 | 为避免污染工作树，生成到系统临时目录；`go build` 仅在临时目录执行 |

### 14.3 证据口径说明

- “0 引用”指对 `pantheon-base` 全仓（含测试、脚本、fixture、文档）**与** `pantheon-ops/backend/modules/business` 做符号检索后的结果；只搜 base 会得出 AUD-012 与 `ssrf` 相反的结论。
- “通过”仅表示本轮执行的检查通过，不代表对应 CI run 的远端状态。
- 所有 P1/P2 均附文件与行号；未附行号的结论（如“无循环依赖”）属编译期可判定事实。

---

## 15. 整改任务分解（映射到 `.harness`）

任务包已按“**一次可评审的 diff**”为单位拆分，不按“一个发现一个包”。programme 与 9 个子包的登记如下（明细见各 `manifest.json`）：

| 顺序 | 任务包 | 覆盖 | 依赖 | 状态（2026-11-01 收口） |
| --- | --- | --- | --- | --- |
| — | `2026-10-31-audit-remediation-programme` | 冻结门槛、依赖图、不处理清单 | — | **completed（9/9 子包）** |
| W1 | `2026-10-31-audit-p1-lowcode-generator-build` | AUD-001 | 无 | **completed（四项缺陷闭合）** |
| W2 | `2026-10-31-audit-p1-business-route-tenant-context` | AUD-002 | 无（决策点） | **completed（方案 C）** |
| W3 | `2026-10-31-audit-p1-entry-doc-truthfulness` | AUD-003/004/007/016/024 | W2 | **completed** |
| W4 | `2026-10-31-audit-p2-doc-lifecycle-archive` | AUD-006/011/017/018/019 | W3 | **completed** |
| W5 | `2026-10-31-audit-p2-doc-metadata-and-index` | AUD-008/009/010 | W4 | **completed** |
| W6 | `2026-10-31-audit-p2-dead-code-legacy-surface` | AUD-012/013/014/020 | W1 | **completed（含收口复核修复的 down 迁移缺陷）** |
| W7 | `2026-10-31-audit-p2-token-cache-race` | AUD-026 | 无 | **completed（2 × DATA RACE → 0）** |
| W8 | `2026-10-31-audit-p2-ci-checker-wiring` | AUD-015/027 | W4/W5 | **completed** |
| W9 | `2026-10-31-audit-p2-oidc-scaffold-decision` | AUD-005 | 无（维护者决策） | **completed（方案 a 第一步）** |

**P3（7 项）不立项**，只在 programme 的 `out` 中登记，避免以"整改"为名继续扩大维护面。
其中 AUD-020 并入 W6、AUD-024 并入 W3；AUD-022/023 按审计原判断"不建议处理"。
AUD-021/025/028 三条在 programme 收口时仍为 open，随后由 **W10**
（`2026-10-31-audit-p3-residual-closeout`）逐条收口，三条现均 closed，各附负向读数。
**28 条不是 28/28**：着落表见 programme 的 `review.md` §2，并区分"programme 收口时"与"W10 之后"两个时点。

---

## 16. 决策落定与执行记录（2026-10-31 起，2026-11-01 收口）

维护者对决策点拍板后，9 个包已全部执行完毕。本节记录决策、实际改动、验证读数与**新发现的数字更正**，供复核；W3–W8 与 programme 收口见 §16.6。

### 16.1 决策一：业务域租户语义 → 方案 C（W2）

**决策**：显式声明"业务域在 V1 内不参与租户隔离"，运行行为零变化。

| 改动 | 位置 |
|---|---|
| 新增 §2.6 业务域租户语义（结论 / 四条事实 / §3.2 不适用的理由 / 禁止宣称 / V2 四项前置条件 / 回归锚点） | `docs/contracts/TENANT_CONTRACT_V1.md:296` |
| §1 Out 登记该边界；§7 表格改「V1 内不适用」；§9 追加 V1.10（Clarification） | 同上 |
| 英文版四处同步 + 一条翻译注记（V1.7–V1.9 变更记录仅存于中文版） | `docs/contracts/TENANT_CONTRACT_V1.en.md` |
| 三个导出函数补文档注释：**不挂租户中间件是合同要求**，并写明"错误修法"的后果 | `backend/pkg/contracts/route_groups.go` |
| 新增结构断言：链路长度 + 顺序 + **不含** `TenantContextMiddleware` / `PlatformScopeTenantContextMiddleware` | `backend/pkg/contracts/route_groups_test.go`（新增） |
| 业务模块接入契约新增两条（不参与租户隔离 / 不得宣称隔离） | `docs/designs/MODULE_CONTRACT.md §10` |
| 「完整数据隔离」→ 系统域隔离 + 业务域单租户语义；同句「配额强制」→ 配额度量与展示 | `README.md:31` |

**为什么 README 放在 W2 而不是 W3**：`README.md:31` 同一句里既有"完整数据隔离"（AUD-002）又有"配额强制"（AUD-003b）。分两次改会让中间态变成"合同说单租户、README 说完整隔离"，比不改更糟。因此 W2 整行改完，W3 任务文档已把该行标记为"已由 W2 处置"。

**负向验证（门禁是否真会红）**：给 `ProtectedGroupWithRedis` 注入 `TenantContextMiddleware(nil, nil)` 后，测试如期失败——
`ProtectedGroup chain length = 3 (...TenantContextMiddleware.tenantContextMiddleware.func3), want 2`；还原后立即回到 `ok`。这条测试是 §2.6 唯一的机械保障。

### 16.2 决策二：OIDC 脚手架 → 方案 (a) 第一步（W9）

**决策**：删除不可达实现，保留迁移与列；drop 列登记为延后事项。

**决定性证据（此前两份报告都没读到的材料）**：
- `docs/designs/SSO_OIDC_DESIGN.md`（799 行）`status: Draft`；
- `.harness/archive/2026-09/evidence/2026-09-08-p1-sso-oidc-design/completion.md` 原文写着
  "This task is **design-first**. Implementation (Phase 2, ~9 hours) is deferred for future execution after design approval."；
- `.harness/archive/2026-09/tasks/2026-09-08-p1-sso-oidc-implementation/` **是空目录**（无 task.md、无 completion.md）。

即：**设计任务自己声明实现被推迟，归档里没有任何"实现已批准"的记录，而 587 行代码已经存在**。这不是"预留能力"，是未获批准的施工面——这是选 (a) 而不是 (b) 的关键。

| 改动 | 结果 |
|---|---|
| `backend/modules/auth/oidc/` | **删除**（4 文件 / 587 行；删前备份 `/tmp/pantheon-oidc-backup-2026-10-31/`，HEAD `ef00738` 可恢复） |
| `000029_oidc_fields.{up,down}.sql` | **未改动**（历史迁移不可改写） |
| `system_user` 四个 `oidc_*` 列 + `uk_system_user_tenant_oidc_subject` | **保留**（可空、无消费者、`auth_type` 默认 `local`；`migrate_test.go:30,102-106` 仍断言其存在） |
| `SYSTEM_AUTH_CONTRACT.md §7.3` | 追加处置说明（删了什么/保留什么/什么延后/证据在哪），EN 版同步 |
| `docs/designs/SSO_OIDC_DESIGN.md` | 未改动（保持 `Draft`，作为身份源确定后的设计输入） |

**验证**：`go build ./...` exit 0；`go vet ./...` exit 0；`go test -short ./modules/auth/... ./pkg/contracts/...` 全 ok；删除后全仓引用面 0。
**下游**：ops 仍持有自己的一份副本（源码复制式消费），其包外引用为 0，下次 `rebuild:from-base` 随 `sharedPaths` 一并移除。

### 16.3 本轮新发现的数字更正（必须记住，勿再引用旧值）

| 项 | 旧值（本报告早期 / 前两份报告） | 实测值 | 影响 |
|---|---|---|---|
| `pantheon-ops` 对 `ProtectedGroup`/`DataScopedGroup` 的调用点 | 25 | **14**（13 × `DataScopedGroup` + 1 × `ProtectedGroup`） | 25 把 `pkg/contracts/route_groups.go` 副本里的 3 行定义与文档提及一并计入；结论不变 |
| 下游业务域 `tenant_id` 命中 | 0 | **1**（`retired_modules_test.go:110` 的 `CREATE TABLE` 测试夹具），`TenantID` 命中 **0** | "零租户建模"仍成立，但准确表述是"0 处模型/查询命中" |
| W9 任务包的 `.env.example` 路径 | `backend/.env.example` | 仓库根 `.env.example`（前者不存在） | 该文件从未登记 `PANTHEON_OIDC_*`，故无需删除配置行 |

### 16.4 冻结门槛的当前进度

| # | 门槛 | 状态 |
| --- | --- | --- |
| 1 | 生成器"生成 → `go build` → 建表"往返证据（AUD-001） | **已满足**（§16.5） |
| 2 | 业务域租户语义有明确结论并落到合同 + README（AUD-002） | **已满足**（§16.1） |
| 3 | 入口文档可执行性与失实修正（AUD-003/004/007/016/024） | **已满足**（§16.6） |
| 4 | 文档归档轮完成（AUD-006/011/017/018/019） | **已满足**（§16.6） |
| 5 | 元数据规范三合一 + scope 矩阵 + 门禁边界（AUD-008/009/010） | **已满足**（§16.6） |
| 6 | 死代码/死表/legacy 端点处理完毕（AUD-012/013/014） | **已满足**（§16.6；下线与生产执行属维护者 human gate） |
| 7 | token 缓存竞争修复 + CI 去重 + checker 接线（AUD-026/015/027） | **已满足**（§16.6） |

> OIDC（AUD-005）不列入冻结门槛，但决策记录已落盘（§16.2），不再需要下一位维护者重新猜测。

**7/7 达成。** 但「进入维护冻结」是维护者的宣告动作，本报告不代替它：
programme 的 `Human Gates` 已写明这一点，manifest 里也以 `FREEZE-DECLARATION`（P2）显式登记。

### 16.5 W1 执行记录：AUD-001 是四处断裂，不是一处 bug

维护者指示"先做 W1"后执行。结论先说：**这条链路此前从未成立过一次**——四处缺陷互相掩盖，因为第一步就编译失败，后面三处永远不会暴露。

| # | 断裂 | 事实（修复前） | 修法 |
| --- | --- | --- | --- |
| 1 | 导入路径 | 产物 3 个文件写 `"pantheon-base/internal/middleware"`，而 `backend/go.mod` 的模块名是 `github.com/duanxldragon/pantheon-base/backend`；`go list` 报 `is not in std` | 模板全部 Go 导入从单一常量 `GO_MODULE_PATH` 派生（`goImportPath()` / `buildGoImportBlock()`） |
| 2 | 中间件签名 | `middleware.TokenAuthMiddleware()` 漏 `rdb *redis.Client` | 不再自拼链，改调 `contracts.{Protected,DataScoped}Group`（`rdb` 由助手内部传入） |
| 3 | 链的形状 | 模板自己拼 `r.Group().Use(...)`，把"业务域该挂哪些中间件"复制进生成器 | 链归 `pkg/contracts` 一处定义，生成器只声明"要不要 DataScope"；顺带让 W2 的 §2.6 结论自动对生成物生效 |
| 4 | 建表 | 全仓**没有任何模块实现 `Bootstrap`**（`grep` 命中 0），生成物只设 `MigrateFunc`；而默认迁移模式下 `RegisterBackendModules` 只跑 `Bootstrap` → `FuncModule.Bootstrap` 在 nil 时返回 nil → **生成的表永远不会被创建** | `FuncModule` 同时设 `MigrateFunc` 与 `BootstrapFunc` |

第 4 重最隐蔽：它只在"重启后"显现，重启前页面一切正常——与 AUD-001 被归类为 P1 的理由一致。

**运行态证据**（在仓库副本 `/tmp/w1-ws` 内完成，产物不入库）：

- 产物 `go build ./modules/business/order/` exit 0、`go vet` exit 0（修复前同一 schema 无法编译）。
- 默认迁移模式（`PANTHEON_AUTO_MIGRATE` 未设）下调用 `InitOrderModule`，`HasTable("biz_order")` = true；日志为 `versioned migrations mode: skipping module AutoMigrate and running runtime bootstraps`。
- **负向**：摘掉 `BootstrapFunc` → 同一测试 `biz_order was not created`；摘掉闸门 → `TestRegisterManagedModuleRejectsUncompilableGeneratedSource` FAIL。两次都恢复后复绿。
- 注册前置新增 `go build ./modules/<scope>/<name>` 闸门，失败返回 `module.register.source_uncompilable` 且 `registration` 不落库（`count = 0`）。

**执行中纠正了任务包原文两处**（原文有误，已写进任务包与 manifest）：

1. 原文要求生成的路由组"与 `system_modules.go:162` 同形（含租户上下文中间件）"。生成物是 `business/*`，按 §2.6 不得挂租户中间件——照原文执行会让 `multi` 模式下**所有**业务请求变成 `TENANT_CONTEXT_MISSING`，即用更严重的故障替换原故障。
2. 原文把 i18n 列为禁区。闸门引入了新的用户可见错误键 `module.register.source_uncompilable`，不登记前端就显示裸键，故放开该一条。

**顺带修掉**：`dynamic_module_handler.go` 三处 `switch err.Error()` 与裸键比较——`common.NewBadRequest(key)` 的 `Error()` 带 `bad_request: ` 前缀，这三个 switch **从未命中**，新闸门的错误会被当成 500 类错误返回。改用 `common.ErrMessage(err)`。

**新增两条未闭合项**（已进任务包 `residualFindings`，不阻断冻结）：

| ID | 级别 | 内容 | 建议 |
| --- | --- | --- | --- |
| AUD-001c | P2 | `BootstrapFunc → AutoMigrate` 与 `docs/DEPLOYMENT_GUIDE.md:44`"生产环境不得启用 GORM AutoMigrate"冲突。当前是任务包预先批准的选项之一，但文档未豁免 | 冻结前择一：文档加例外条款，或改为 Bootstrap 执行显式 `CREATE TABLE IF NOT EXISTS`（先例 `internal/middleware/data_scope_middleware.go:68`）。**已由 W11 选前者收口，见 §16.8** |
| AUD-001b | P3 | 产物 `gofmt -l` 列出全部 5 个 Go 文件（模板写死字段对齐空格） | 提交时 lint-staged 的 `gofmt -w` 会归位，CI gofmt 对 PR 仅报告；不混进本包 |

---

### 16.6 W3–W8 执行记录与 programme 收口（2026-11-01）

W1、W2、W9 之后的六个包与 programme 本体已在 2026-11-01 全部收口。**9/9 子包 `completed`，
7/7 冻结门槛达成，全量门禁 0 error。**

| 包 | 做了什么 | 关键读数 |
| --- | --- | --- |
| W3 入口文档 | README 版本表改为指向 `VERSION` + release-notes 索引（不再手抄 release 号）、配额改"度量展示（超限不拒绝写入）"、community files 按实际列举（只缺 `CODE_OF_CONDUCT.md`）、门禁段删掉 2026-10-03 快照改"以最近一次 required checks 为准"；DESIGN 新增 §3.4 租户章；`docs/README` 启动命令改 `cd backend && go run ./cmd/server`；designs 索引补齐 **62/62** 并如实声明 5 份缺 `.en.md` | `frontmatter-check` 415 docs 通过；`check-doc-links`/`check-doc-inventory` 0 findings |
| W4 文档生命周期 | 判据先行（证据 + `summary.md` + manifest `completed` 才允许改状态），新增 `scripts/harness/task-evidence-state.mjs`；92 份任务包中 **69 份 `Active`/`Draft` → `Archived`**；6 份未实现设计降 `Draft` + 未实现声明；7 份 `docs/TENANT_*` 移入 `docs/reviews/` 并重算链接 | 任务包状态由证据而非作者自述决定 |
| W5 元数据与索引 | 字段定义收敛到 `DOCUMENT_FRONTMATTER_SCHEMA` 一处（其余改指针）；scope 矩阵 `dept/post` 改 `tenant-owned` 附行号；`frontmatter-check` 新增覆盖范围自报 | 输出"415 扫描 / 302 带元数据 / 113 不强制 + 强制目录清单" |
| W6 死代码/死表/legacy | 删两个零消费 hook、`scaffold.Generate`、`readRoleKeysFromContext`；`000031` drop 死表；legacy 认证端点保留 + `Deprecation`/`Sunset: 2026-12-31` | `pkg/security/ssrf` **未删**（ops 5 处 import）；每项删除都有 base + ops 双侧检索 |
| W7 token 缓存竞争 | 缓存与 `SessionData` 由指针改值、`RoleKeys` 深拷贝、活跃时间同步进写锁（锁内无 Redis I/O） | 修复前 **2 × DATA RACE**（`:182/:183/:186`）→ 修复后 **0**；`-race ./internal/middleware/` ok |
| W8 checker 接线 | 9 个进 `Docs Governance`、1 个仅 workspace 根、1 个 mirror-only；`ci.yml` coverage 步骤去掉 `-race` | 每个接线都造违规验证过会红；race 每 PR 2→1 |

**收口复核推翻并修复的一处（W6）**：`000031.down` 初版以"up/down/up 三步 exit 0"为判据，
但把 000024 **已删除**的 `idx_permission_role_data_scope_policy_role_key (role_key)` 重建了回来
——回滚到 v30 后两个租户不能再用同一个 `role_key`。改用 `SHOW CREATE TABLE` 逐字节对比
（"空库直接迁到 30" vs "迁到 31 再回滚到 30"）暴露，已修复并新增
`TestDropPolicyTableDownReconstructsVersion30Shape`，该用例在缺陷版本上验证过会红。
**教训：命令能跑通不等于结果是那个结果。**

**28 条的着落不是 28/28**：23 条落到子包；AUD-022/023 按审计原判断"不建议处理"；
AUD-021/025/028 在 programme 收口时留开，随后由 **W10** 收口（见 §16.7）。
逐条见 `.harness/evidence/2026-10-31-audit-remediation-programme/review.md` §2。

**未执行的动作：进入维护冻结的宣告。** 按 programme `Human Gates`，该动作归维护者；
programme 只交付 9/9 与 7/7 的读数，并在 manifest 中以 `FREEZE-DECLARATION`（P2）登记。

---

### 16.7 W10：三条 P3 残留收口（2026-11-01）

programme 留开的三条 P3 另立一包（`2026-10-31-audit-p3-residual-closeout`）收口。
三条都小，但都没有停在"把审计描述的动作照做一遍"。

| 项 | 审计的描述 | 真正做了什么 | 读数 |
| --- | --- | --- | --- |
| AUD-021 | 模板里仍 import 已退役的 `FilterPanel`，建议切 `SearchToolbar` 或保留并写明意图 | **发现成因不是那行 import**：`lowcode/templates` 自 #120 起被 `tsconfig` exclude（当次提交说明就是"修 TypeScript 与构建错误"）、全仓零 import，不参与构建；却被 `check-search-toolbar-contract.mjs` 用 `EXEMPT_SEGMENTS` **整目录豁免**。删死 import，并**撤销豁免**，另补生成器侧第 14 条 + 门禁侧第 10 条两条锁 | 被检文件 **102 → 106**；塞回 `FilterPanel` → `1 finding` / exit 1；恢复豁免 → 第 10 条红；拼进生成器 → 第 14 条红 |
| AUD-025 | 102 份 `Active` 文档 >60 天未更新，建议建立"90 天内复核"规则 | `frontmatter-check` 增加 90 天窗口的**计数**与 `--report-overdue` 明细；规则写入 `DOCUMENT_METADATA_AND_STATUS` §6.5 中英双版。**刻意不做成 error** | 实测 **144 份**逾期（71 份 `.en.md`、最老 198 天）；门禁 exit 0；把逾期计入 `ok` → docs 第 14 条红 |
| AUD-028 | `main.go` 的 `_, cancel := context.WithTimeout(...)` 未使用 ctx 也未提前 cancel | 删除，并**原地写明为什么不设 timeout**（`InitTracer` 不收 context），避免被好心加回 | `go build` / `go vet` / `gofmt -l` 三清；`context` 仍被 shutdown 段使用，import 未动 |

**两处口径更正**：

1. AUD-025 审计写"102 份 / 60 天"，本次实测是"144 份 / 90 天"。窗口不同、统计对象可能不同，
   **未复现审计口径**，已在 §6.5 写明以实测为准，避免被读成回归。
2. AUD-021 的成因与审计的描述不一致（见上表），因此"模板切 `SearchToolbar`"这条建议本身
   对死模板没有意义——有意义的是撤掉豁免，让模板目录回到被检范围。

**W10 自己又留了两条 P3 残留**（收口不是把问题清成零）：

| ID | 内容 | 为什么没做 |
| --- | --- | --- |
| AUD-021b | `lowcode/templates/` 目录本身 1,894 行 / 5 文件死代码仍在 | 删 1,894 行与删一行 import 不是一个量级的决定；已核对 `pantheon-ops` 仅 `.foundation/releases/*/bundle/` 历史快照内有同名文件、活动代码不引用，将来删是安全的 |
| AUD-025-review-round | 144 份逾期 `Active` 文档需一次人工复核轮 | 按 §6.5 只能"复核后重写 `updated_at`"或"降 `Draft`/`Superseded`"，两者都要求真读一遍，不能脚本批量完成 |

另有一处文案漂移**就地修掉、未计为残留**：该 checker 的文档指针与错误文案原本都写
`DESIGN.md §7.2`，而仓库里已无此文件（现为 `docs/frontend/DESIGN_ENGINEERING_GUIDE.md §8.1`
+ `UI_PATTERN_LIBRARY.md`）。既然本来就要改这个脚本，留一个"去找一个不存在的文档"的指针
没有道理——这正是本次审计要治的那类漂移。

**三条共同说明的一件事**：`Active`/豁免/未使用变量这类"小项"，gap 往往不在被指出的那一行，
而在它赖以存在的那条规则里。三条都不是靠"把测试改绿"能收口的，所以每条都留了一次人为制造的红。

---

### 16.8 W11：生产 AutoMigrate 边界（2026-11-01，AUD-001c）

W1 留下的 P2，审计要求"冻结前择一"。选**文档加例外**，并把例外写窄到可逐表核对。

**先确认冲突是否真实可达。** 第一反应是 `PANTHEON_ENABLE_DYNAMIC_MODULES` 生产固定 `false`，
冲突应该不存在。查下来不成立：`DynamicModuleEnvGuard` 只挂在 `/lowcode/dynamic-modules/*` 这组
**HTTP 路由**上，而生成模块是通过 `backend/modules/business/generated_registry.go`
**编译进二进制**的，由 `InitGeneratedBusinessModules` 静态注册，不经过那组路由——
启动时照样走 `Bootstrap → service.Migrate() → AutoMigrate`。冲突是真的。

| 项 | 做了什么 | 读数 |
| --- | --- | --- |
| `DEPLOYMENT_GUIDE §3` | 禁令限定为**平台 schema**（`system_*` / `platform_*` 只允许版本化迁移）；新增边界表（表类型 / 由谁创建 / 受哪个开关控制）；写明两点代价：应用账号需 DDL 权限、生成表列**只增不减**；区分 `PANTHEON_ENABLE_DYNAMIC_MODULES` 与本例外无关 | 边界可逐表核对 |
| `MODULE_CONTRACT §4.3` | 接口补 `Migrate`；加"何时跑 `Migrate` / `Bootstrap`"判定表；说明 `BootstrapFunc` 是运行时才有 schema 的模块的出口 | 契约不再写"后续再扩展" |
| `contracts/module.go` | 启动日志 `skipping module AutoMigrate` → `module Migrate() skipped, runtime bootstraps run instead` + 指向文档的 note | 原句与 bootstrap 阶段行为相反，会误导排障 |
| `contracts/module_test.go` | 新增 `DefaultModeRunsBootstrapForGeneratedModules`，钉住 `FuncModule` 形状（只有 `BootstrapFunc` 仍 bootstrap；只有 `MigrateFunc` 不 migrate） | 负向 A 只打红它；负向 B 与 `PhasedExecution` 同时红 |

**被拒绝的两条，以及为什么**：

1. **显式 `CREATE TABLE IF NOT EXISTS`**（审计建议的另一半）：表已存在时它什么都不做。
   低代码 schema 加字段后重新生成模块，表**不会补列**，运行时报"列不存在"；`AutoMigrate` 会补。
   这是用"更弱的建表能力"换"更严格的文档措辞"，把文档问题转成运行时问题。
2. **生成器产出迁移文件**：原理上更好（可审查、可回滚），生成器也已有 `TYPE_MAPPING[].sql` /
   `getSQLType()`。但要让手写 DDL 与 GORM 模型逐列一致——`time.Time` 在 MySQL 上是 `datetime(3)`
   而非映射表里写的 `DATETIME`，索引命名、m2m 关联表主键与列顺序也要对——**一旦不一致，
   查询直接失败，而那时已经没有 `AutoMigrate` 兜底**。冻结前不引入这一类风险。
   登记为 `AUD-001c-ddl`（P3）待后续版本重估，不写成"不需要做"。

**一处自我更正**：新用例的初版注释写"既有用例在删掉 bootstrap 阶段时仍会通过"，负向 B 实测推翻
（`PhasedExecution` 也会红）。真实缺口是另一件事：既有用例用同时实现两个方法的自定义类型，
无法区分"模块完整所以 bootstrap 跑了"与"只有 `BootstrapFunc` 也会 bootstrap"——而后者正是
`FuncModule` 与生成模块的形状。注释已按实测改写。**与 §16.6 的 `000031.down` 同源：先写结论再验，
验完才发现结论不成立。**

### 16.9 W12：三条 P2 的可验证前置条件（2026-11-01）

programme 收口后剩三条 P2，全部登记为"归维护者"。W12 不代做其中任何一个动作，
只补它们缺的东西——在此之前，"归维护者"等于"到那天现场发明流程"。

| ID | 原来缺什么 | W12 补了什么 | 仍缺 |
| --- | --- | --- | --- |
| `AUD-013-prod` | 原话要求"确认表内数据可丢弃"，但**没有任何只读手段能读出表内有多少行** | 6 条 SELECT 的预检脚本 + 逐条断言的用例 + 生产执行规程（预检 → 导出 → 还原证明 → 变更记录 → 部署 → 事后核对 → 回滚） | 维护者选放行路径（A 先做 G2 演练 / B 接受表级导出还原证明）并执行 |
| `AUD-014-followup` | 只有一句日期 | 下线清单精确到文件与行号，含"删完怎么证明删干净了" | 维护者书面确认无外部集成方；日期到 2026-12-31 |
| `FREEZE-DECLARATION` | 只有一行"该动作归维护者" | 宣告文本：7 项门槛逐项对应、今日实测读数、12 条残留归属与不阻断理由、签署块 | 整改产物进 `main`；维护者填 SHA 并签字 |

**W12 最关键的一条发现：DROP 与部署不可分离。**

`PANTHEON_AUTO_MIGRATE=false` 是生产配置（`k8s/configmap.yaml`），而这个值恰恰是
"走版本化迁移"的那条分支：`main.go:101 → runDatabaseMigrations → RunMigrations → m.Up()`。
镜像只构建 `./cmd/server`（`Dockerfile:41`）。所以：

- 没有"先迁移、后切流量"的窗口——服务启动那一刻 DROP 就发生；
- 三副本滚动更新时，由最先取到 `GET_LOCK`（`mysql.go:298`，等待 10s）的 Pod 执行，
  其余等待超时后启动失败并被 k8s 重启（表现为部署抖动，不是数据损坏）；
- 一次部署会应用**所有** pending 迁移，不只是 `000031`；
- `k8s/README.md:52` 的 "run migrations manually" 是一个**镜像内没有入口**的指示，
  可用的 `cmd/tenantmigration` 只在仓库内，且名字（tenantmigration）与职责（全量平台迁移）不符。

最后一条登记为 `AUD-013-ops-migrate-entry`（P3）：当前不造成故障（自动路径有效），
但会让"按文档手工迁移"的操作者找不到入口。

**顺带纠正一个我自己记错的事实**：此前记录"没有迁移 CLI"。实际 `cmd/tenantmigration` 存在，
`status` 子查询只读、不触发 `RunMigrations`（`main.go:23` 的 `up` 分支才触发）。
执行规程据此给出 `go run ./cmd/tenantmigration status` 作为部署前的版本检查。

**预检为什么值得一个测试**：`rows_to_be_dropped` 是执行者批准一次不可逆 DDL 的**唯一读数**。
如果它低估行数，人就在错误信息上签字。所以第 3 条 `COUNT(*)` 不加任何 `WHERE`（加租户过滤
就会低估），并且负向验证必须红在这一条上，而不是红在别处。

**一处自我更正**：解析器初版按字面 `"\n;\n"` 切分语句。负向验证时用脚本改写文件、换行被写成
CRLF，解析器把整个文件当成一条语句，报 `parsed 1`。这不是负向验证生效，是解析器**对行尾敏感**
——Windows 检出或任何编辑器保存一次都会触发。改成按行锚定的正则 `(?m)^;[ \t\r]*$` 后，
才真正红在 `rows_to_be_dropped=0, want 2` 上。若把 `parsed 1` 当成"负向验证通过了"，
就会留下一个在 Windows 上必然失败的测试，并且以为它已经被验证过。

**冻结宣告为什么写着"未签署"**：HEAD `ef00738` 是整改前快照（`VERSION 0.18.1`，
150 modified / 9 deleted / 4 renamed / 3 renamed+modified / 49 untracked），9 个子包的整改
全部未进 `main`。冻结的对象是提交，而包含整改的提交还不存在——在它进 `main` 之前签署，
等于宣告冻结一份不包含整改的代码。这不是流程问题。

**顺带核对的读数漂移**：`DOCUMENT_METADATA_AND_STATUS` §6.5 记的逾期 Active 文档从 144 走到 143，
差的一份是 `docs/designs/MODULE_CONTRACT.md`——W11 因 §4.3 真实内容变更把 `updated_at` 推到
2026-11-01，退出逾期集合。这正是 §6.5 期望的处置方式（复核内容后重写日期），不是批量改日期。
已在中英双版补写这一变化，而不是把旧数字留着。

---

*§1–§14 为只读审计产出（基线 `origin/main bb0498f` / 工作树 `ef00738`）。§16 记录维护者拍板后的执行结果：
W1–W9 九个包全部落地并附证据，7 项冻结门槛全部达成；programme 留开的三条 P3 由 W10 于同日收口
（§16.7），W1 留开的 P2（AUD-001c）由 W11 收口（§16.8），三条 human-gated P2 的前置条件由 W12 补齐
（§16.9），各自留下新的 P3 残留。**三条 P2 的动作仍未执行——它们归维护者，且冻结宣告的签署前提
（整改产物进 `main`）当前不成立。***
