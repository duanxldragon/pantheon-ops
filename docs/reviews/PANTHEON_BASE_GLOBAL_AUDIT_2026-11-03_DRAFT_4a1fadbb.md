---
title: Pantheon Base 全局审计报告（2026-11-03，基线 4a1fadbb 中间稿）
doc_type: Assessment
layer: platform
status: Superseded
superseded_by: docs/reviews/PANTHEON_BASE_GLOBAL_AUDIT_2026-11-03.md
updated_at: 2026-11-03
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
  - docs/contracts/TENANT_CONTRACT_V1.md
  - docs/contracts/SYSTEM_AUTH_CONTRACT.md
---

# Pantheon Base 全局审计报告（2026-11-03，基线 4a1fadbb 中间稿）

> **本文件为中间稿归档**：原写作于 workspace 根目录（仓库只读约束），基线 `4a1fadbb`、基于 211 项未提交改动。
> 整改合一（PR #488）后已在干净工作区复核，最终版本见 `docs/reviews/PANTHEON_BASE_GLOBAL_AUDIT_2026-11-03.md`（基线 `52550603`）。
> **本稿结论已被最终版取代**，保留仅作差异对比与追溯：AUD-01/02/03/05/08 等「未提交」类问题在最终版已闭环。
> 原文头两行声明（只读审计、写在仓库外）为写作时实况，保证追溯真实；所有结论均可由文中文件路径 / 行号 / 命令复跑复核。

---

## 1. 总体结论

| 项 | 值 |
| --- | --- |
| 审计时间 | 2026-11-03 |
| 审计 Commit SHA（基线） | `4a1fadbbda97e438b835a36c35126a3b090e0164`（`main`，2026-11-02，`chore(harness): archive the completed follow-redirects audit packet (#487)`） |
| 版本 / 发布锚 | `VERSION` = `0.18.2`；tag `pantheon-base-v0.18.2`（共 19 个 tag，400 次提交，首提交 2026-06-08） |
| 分支 | `main`（跟踪 `origin/main`，无本地领先/落后来源差异） |
| 工作区状态 | **不干净：211 项**（142 M / 51 ?? / 9 D / 2 A / 4 R / 3 RM；未暂存 +1594 −1489，暂存 +1295） |
| 审计范围 | 架构、功能完整性、冗余与过度设计、安全与多租户、数据库与性能、前端、文档资产、Harness/CI、下游 `pantheon-ops` 兼容性 |
| 审计方法 | 以 `HEAD` 提交为结构基线（`git show/grep HEAD`），工作区仅用于补充读数；静态检索 + 三个可执行基线（build/vet/test-short/tsc）+ 只读门禁脚本；文档逐项对代码交叉核对；未做运行时启动、烟测、压测、Sonar |
| 总体成熟度 | **较高（≈B+）**：核心域闭环、安全默认拒绝、工程纪律真实存在且产出过具体修复；但「已完成但未提交」的整改体量巨大，且 HEAD 上仍保留若干已确诊缺陷 |
| 是否具备维护冻结条件 | **暂不具备（B：完成少量整改后冻结）**。冻结对象是提交，而当前整改产物不在提交里 |

### 1.1 一句话结论

**底座本身已经够稳，不够稳的是「交付状态」。** 代码质量、安全模型与工程机制都在企业级基线的合理区间内；真正阻挡冻结的是 211 项未提交整改、11 个未接线 checker、3 处文档漂移和 1 处契约与代码相反。

### 1.2 必须同时读的两条基线

| 基线 | 值 | 用途 |
| --- | --- | --- |
| 结构性基线（本报告主口径） | `HEAD` = `4a1fadbb` | 文件数、行数、迁移数、路由数、checker 接线 |
| 可执行基线 | 当前工作区（含未提交整改） | `go build` / `go vet` / `go test -short` / `tsc -b` 实测 |

混用这两者是历史审计数字互相矛盾的根因。**未提交的整改不能当作 `main` 的现状**：HEAD 上 `oidc` 包仍存在、`000031` 迁移不存在、`tokenSessionCache` 仍是指针缓存、11 个 checker 仍未接线——这些在工作区里「看起来已经好了」。

---

## 2. 整体架构分析

### 2.1 分层与 Composition Root（通过）

`backend/cmd/server/main.go`（289 行）是干净的唯一装配点：

- 中间件顺序合理：SecurityHeaders → CSP Nonce → CSP → BodyLimit → CORS → otelgin → Prometheus → RequestContext+OperationLog → CSRF（`main.go:136-145`）。CSRF 在路由级鉴权之前，正确。
- 模块装配集中且显式（`main.go:159-171`）：`platform → lowcode → system → auth → business`。
- **auth 与 system 的解耦是真的**，不是文档口号：`auth.InitAuthModule(api, db, user.NewCredentialRepository(db))`，跨域通过 `pkg/contracts/authuser` 端口，两侧互不 import（`main.go:163-169` 注释与签名一致）。
- 生产相关硬化到位：Redis 必需时启动即 Fatal（`:96-101`）、迁移失败即退出（`:104-112`）、优雅停机后先 `Shutdown` 再 drain 审计队列（`:180-215`）、`/metrics` 在生产需 Bearer（`:196-215`）。

**判断：分层合理，没有为了「架构感」引入中间层。** `internal / pkg / modules` 三段职责清晰，`pkg` 中确有被多处复用的契约（`contracts`、`tenant`、`impexp`、`database`、`authtoken`）。

### 2.2 模块边界（通过，有 1 处结构性重复）

- 垂直切片成立：`system/{iam,org,config,audit,i18n}` 各自 handler/service/model，无「大 system 杂物间」。
- 平台层只做聚合：`modules/platform` 仅 dashboard/health/routes。
- 业务域不反向依赖系统域：`modules/business` 仅 `business.go` + 生成注册表 + 退役模块表。

**问题（P2）**：`system_modules.go` 中同一串 4 段中间件被**手写重复 14 次**：

```
backend/modules/system/system_modules.go:162,163,188,209,238,288,321,346,375,419,430,435,471,513
TokenAuthMiddleware(RDB).Use(PlatformScopeTenantContextMiddleware(...)).Use(CasbinMiddleware()).Use(RefreshSyncMiddleware(...))
```

风险不是行数，而是**漂移**：任何一处漏掉 `DataScopeMiddleware` 或 `PlatformCapabilityMiddleware` 都会静默产生权限差异，而这种差异不会被编译期发现。仓库已有 `pkg/contracts/route_groups.go` 作为路由组助手，系统域自身却没有走它。

### 2.3 依赖关系（通过）

- 无循环依赖（`go build ./...` 在包级通过即证伪了 Go 的循环 import）。
- 无跨模块直接访问内部实现：`auth` → `system/iam/user` 的唯一通道是 `contracts/authuser` 端口 + 构造函数注入。
- 无「无意义中间层」：没有为单实现强造的 Interface 群。仅有的两个接口（`user.SessionLifecycle`、`contracts.BackendModule`）都有真实多实现/多消费者（测试替身 + 各模块）。

### 2.4 过度设计迹象（有限，见 §4）

唯一的系统性过度工程在**治理层**，不在业务代码层。

---

## 3. 核心功能完整性分析

### 3.1 单租户模式（通过）

| 能力 | 结论 | 证据 |
| --- | --- | --- |
| 登录 / 刷新 / 登出 | 通过 | `auth/login/*`；双层接口（`/auth/*` 与 legacy `/system/*`） |
| Refresh Token / 会话 | 通过 | `auth/session` 生命周期服务 + Redis 会话 + 撤销闭环 |
| 用户 / 角色 / 菜单 / 权限 | 通过 | `system/iam/*`；125 条路由注册（`system_modules.go:200-525`） |
| 组织（部门 / 岗位） | 通过 | `system/org/{dept,post}`，含树展开与治理任务 |
| 数据权限 | 通过 | 角色策略 + `dept_and_children` + fail-closed（§5.4） |
| 审计日志 | 通过 | 操作日志中间件 + 登录日志 + 导出 + 保留期后台清理 |
| 字典 / 系统设置 | 通过 | `system/config/{dict,setting}`，含租户覆盖测试与敏感配置加密 |
| 国际化 | 通过 | `system/i18n` + 内置语言资源 + 前端 `check:i18n-hardcode` |
| 动态菜单 | 通过 | `iam/menu` + 组件注册表 + 前端 manifest + 契约检查 |

### 3.2 多租户模式（核心闭环成立，边界有明确未做项）

逐项核对，而非只看表/字段/页面：

| 项 | 结论 | 证据 |
| --- | --- | --- |
| 租户上下文解析 | **通过** | `pkg/tenant/tenant.go:112-152` 四级优先级；缺失即 `ErrTenantContextMissing`（默认拒绝） |
| 中间件落地 | **通过** | `PlatformScopeTenantContextMiddleware` / `TenantContextMiddleware` 两种口径，分别给平台组与租户组 |
| 租户 CRUD + 成员 | **通过** | `system_modules.go:290-297` 恰 8 条，与 DESIGN §3.4 声称一致 |
| 平台运营读面 + 套餐 | **通过** | `system_modules.go:302-306` 恰 5 条 `tenant-ops/*` |
| 租户隔离（行级） | **通过（结构层）** | `tenant.Scoper` 绑定进 Service；13 个迁移涉及 `tenant_id`，28 处索引含 `tenant_id` |
| Casbin 域隔离 | **通过** | 先全局、后 `role:<key>@tenant:<id>`，两命名空间不合并（`casbin_middleware.go:41-52`） |
| 租户停用 / 归档 | **通过** | `ErrTenantSuspended` / `ErrTenantArchived`，显式 header 覆盖时校验真实存在且 active |
| 初始化 / Bootstrap | **通过** | `tenant_provision.go` + 初始化指南 + 幂等测试（`tenant_provision_*_test.go`） |
| 配额 | **度量展示，不强制**（与合同一致，非缺陷） | 合同 §2.4；`QuotaFor` 只读展示 |
| 业务域租户隔离 | **明确不做（V1 单租户语义）** | 见下方 P1-2 |
| 跨租户关联查询 / 异步丢失上下文 | **未验证** | 未做运行时多租户矩阵压测；后台维护任务未见租户维度参数 |

**结论：多租户不是「只有表和页面」。** 上下文解析、拒绝策略、Casbin 域、路由分组、隔离测试（`tenant_hostile_smoke_test.go`、`tenant_platform_scope_test.go`、`data_scope_tenant_test.go`、`*_tenant_isolation_test.go`）都在。

### 3.3 真实调用链抽样（通过）

`前端 UserList → /system/user/list → systemDataScoped 组（Token→TenantContext→Casbin→DataScope→RefreshSync）→ UserHandler → UserService(tenant.Scoper) → GORM`：

- 写路径挂 `SecureActionMiddleware`（批量删、重置密码、删用户），读/导出走 `systemDataScoped`，**不是所有接口一律加 DataScope**，这个区分是对的（导出与列表才需要行级收窄）。
- 菜单写入额外要求平台能力（`PlatformCapabilityMiddleware`，`system_modules.go:200-205`），而树读取保持租户可达——这是有实际攻击面意识的设计，不是装饰。

### 3.4 功能完整性缺口（均为「已明确不做」，不是半成品）

- 配额强制、独立 schema/DB、跨租户共享、自助注册/计费、租户自定义域名、品牌配置——合同 §1 Out 已登记。
- SSO/OIDC、登录风控（设备指纹/异地）——设计与合同均为 Draft/未批准，UI 与路由不存在。
- 6 份运维中心设计（告警/通知/报表/调度/任务/审批）为 Draft，无实现。**不建议为实现而实现**（§12「明确不建议处理」）。

---

## 4. 冗余代码与过度设计分析

分类口径：A 确定无用 / B 重复实现 / C 过度抽象 / D 合理保留 / E 证据不足。

| # | 项 | 类 | 位置 | 实况证据 | 建议 / 风险 |
| --- | --- | --- | --- | --- | --- |
| R1 | OIDC 不可达实现 | **A** | `backend/modules/auth/oidc/`（4 文件 / 587 行，`git ls-tree HEAD` 确认存在） | HEAD 上唯一引用是自身 `handler.go:110` 的字面量 + `000029` 迁移注释；无路由注册、无 env 登记 | 删除（工作区已删，**未提交**）。风险低：包外 0 引用；迁移 000029 与 `oidc_*` 列保留不动 |
| R2 | `tokenSessionCache` 指针缓存 | **A→已确诊缺陷** | `HEAD:backend/internal/middleware/token_middleware.go:24,38,60` | HEAD 为 `map[string]*tokenCachedSession`，命中后交出的指针被并发读写（活跃时间戳） | 已在前次审计复现 2 处 DATA RACE；工作区改为值 + 深拷贝，**未提交** |
| R3 | 前端两个死 hook | **A** | `frontend/src/hooks/usePagination.ts`、`useRequest.ts`（HEAD 存在），`hooks/index.ts` 再导出 | `git grep HEAD` 在 `frontend/src` 非 hooks 目录 0 命中 | 删除。下游 ops 有自有副本，不受影响 |
| R4 | 死表 `permission_role_data_scope_policy` | **A** | 迁移 `000031`（仅工作区） | HEAD 无 `000031`；生产代码仅剩两处注释提到（`role_service.go:600`、`cmd/tenantsizing/main.go:89`） | drop 迁移需生产 human gate（已有预检/runbook 在工作区） |
| R5 | 中间件链重复 14 次 | **B** | `system_modules.go` 上列 14 行 | 逐行比对可见仅 1 处差异（DataScope / PlatformCapability / 无） | 抽 3 个路由组助手；风险：触碰全部系统域路由，需路由快照测试兜底 |
| R6 | `ResolveForCanary` 仅测试调用 | **C** | `pkg/tenant/tenant.go:155-162` | 生产 0 调用；调用点全在 `*_test.go` | 可删或标 `Deprecated`；收益小，**D 也可接受**（保留成本近零） |
| R7 | `pkg/common/re_export.go` | **D** | `re_export.go:1-60` | 为 `common.*` 名字向后兼容的重导出 | **保留**：被大量 handler 使用，删它是纯风险 |
| R8 | `pkg/common/builtin.go` 魔法值集中 | **D** | `builtin.go` | 内置 admin 用户/角色常量 | 保留，正确做法 |
| R9 | `.harness/` 与 `docs/harness/` 双目录 | **D** | 目录职责 | 机器态 vs 可审阅态，`docs/README.md §2.3` 已显式说明不合并 | 保留，**不要合并** |
| R10 | 11 个未接线 checker | **C（治理过度）** | 见 §9 | `scripts/harness/check-*.mjs` 共 26；HEAD 上 11 个在 `package.json`/`quality.yml`/`ci.yml` 全无引用 | 接线或按 `HARNESS_RETIREMENT_REVIEW` 退役；现状是「写了规则但没有任何一次执行」 |
| R11 | `lowcode/templates` | **E** | `frontend/src/modules/lowcode/templates`（4 文件 / 705 行） | 本轮未建立 import 关系证据 | 前次审计称 1,894 行死代码且被 tsconfig exclude；**需独立决策，本轮不认定** |
| R12 | 生成物 `feature-ledger.json` 派生快照 | **D** | `schema/generated/` | 由模块注册派生，检查器校验 | 保留；注意不要反向当输入 |

**没有发现的**：无重复 Service 实现、无重复 API 封装层、无「一个调用方的复杂 Factory/Registry」、无多套并发安全实现同一能力的机制。生成器/动态模块的 Registry 有真实消费者（后端注册表 + 前端组件注册表 + overlay），属 D。

---

## 5. 安全与多租户审计

### 5.1 认证（通过）

| 项 | 证据 |
| --- | --- |
| 密码存储 | `bcrypt.GenerateFromPassword(..., security.GetBcryptCost())`（`security_service.go:194`、`tenant_provision.go:415`）；校验用 `CompareHashAndPassword`（`login_service.go:90`） |
| Token | Redis 会话 + `ValidateSession`；用户级与会话级双黑名单；**Redis 异常 → 401**（不静默放行，`token_middleware.go:160-200` 区段注释与实现一致） |
| 空闲超时 | `LastActivityAt` + 可配置 idle minutes，超时删会话并 401 |
| 登出失效 | 会话删除 + 黑名单；`session_revocation_closure_test.go` 覆盖 |
| 暴力破解 | 源级 + 用户级 throttle 表与窗口（`login_service.go:99,515,574,588,613,691`） |
| Cookie | `HttpOnly: true`、`Secure: true`、`SameSite=Strict`（`pkg/common/http/cookie.go:31-33`） |
| MFA | TOTP 实现 + 测试（`mfa_totp_test.go`），`/auth/mfa/verify` 为 CSRF 豁免之一 |
| 默认账号 | 非生产 `admin/123456`；生产要求 `PANTHEON_INITIAL_ADMIN_PASSWORD` ≥12。**代码是否强制校验该 env 未在本轮验证（E）** |

### 5.2 授权（通过）

- Casbin 中间件在 `Enforcer == nil` 时 **fail-closed** 403（`casbin_middleware.go:28-31`）。
- 角色集合为空按 `guest` 处理，不会「无角色=全通」。
- `isSelfServiceRouteBySignature` 用 FullPath + Method + `scope` 精确匹配（`casbin_middleware.go:104-140`），路由改名后**默认拒绝**（安全方向正确，代价是脆弱性，P3）。
- 权限变更后的缓存失效：Token 缓存按 token 失效 + 策略同步（`role_policy_sync_test.go`、`role_concurrency_test.go`）。
- 超级管理员保护：`BuiltinAdminUserID/AdminRoleID/AdminRoleKey` 常量统一引用。

### 5.3 多租户安全（通过，边界明确）

- **Tenant Context 可信性**：只接受 3 类可信源 + compat 兜底；`X-Tenant-Id` 要求平台级授予；subject claim 不接受 `0`（防静默放大，`tenant.go:170-186`）。
- **改参数越权**：租户 ID 不从 query/body 取，只从 header（且需平台能力）/ session claim 取。
- **平台兜底收紧**：`PlatformScoped && PlatformCapable && 不属于任何租户` 三者同时成立才回退到 tenant 0（`tenant_context_middleware.go:74-129`）；`isPlatformPopulationSubject` 在 membership 不可用时 **fail-closed**。
- **平台控制台二次门禁**：`PlatformCapabilityMiddleware` 阻断「租户管理员借 `/api/v1/*` 域策略枚举全租户」（`system_modules.go:288`）。
- **缓存 Key 租户维度**：`userDeptCache` 用 `buildUserDeptCacheKey(db, userID)`，含连接命名空间（`connection_cache_namespace_test.go`）；策略读取按 `(tenant_id, role_key)` 唯一键，避免跨租户合并策略（`data_scope_middleware.go:36-52` 注释即为该缺陷的修复记录）。
- **未验证**：后台维护任务/异步链路是否携带租户上下文（未见运行时证据）；多实例并发下的隔离压测未做。

### 5.4 常见安全问题（通过，少量说明）

| 项 | 结论 | 证据 |
| --- | --- | --- |
| SQL 注入 | **未见可注入面** | `Sprintf` 拼 SELECT/WHERE/FROM：0 命中；46 处 `Raw/Exec` 均为常量 DDL 或参数化 |
| XSS | 通过 | CSP + Nonce（`csp_middleware.go`、`csp_nonce_middleware.go`）+ React 默认转义 |
| CSRF | 通过 | double-submit + **精确路径**豁免（`csrf_middleware.go:46-56`），注释明确拒绝前缀匹配以防新路由被静默豁免 |
| CORS | 通过 | 显式 allowlist（`PANTHEON_ALLOWED_ORIGINS`），凭据仅在命中 origin 时下发，`Vary: Origin`（`cors_middleware.go:46-60`） |
| IDOR/BOLA | 通过（抽样） | 用户详情/删除走 `:id` + Casbin + DataScope + SecureAction；租户成员删除需平台能力 |
| 请求体 / 上传 | 通过 | BodyLimit 默认上限；上传授权与 S3 下载分别有契约测试 |
| SSRF | 有独立防护包 | `pkg/security/ssrf`（**下游 ops 5 处引用，禁删**） |
| 敏感信息 | 通过 | 设置项加密（`setting_crypto.go`）；操作日志请求体守卫（`operation_log_request_body_guard_test.go`）；错误统一错误码 + i18n key |
| 错误信息暴露 | 通过 | `common.Fail` 统一结构，不返回内部 err |

**P0 结论：本轮未发现可直接利用的认证绕过、跨租户读取或注入漏洞。** 未验证项不得读作「已验证安全」。

---

## 6. 数据库与性能分析

### 6.1 Schema 与迁移（通过）

- 60 个 `.sql`（`HEAD`，`000001`–`000030` + `preflight/` 目录），每个版本 **up + down 成对**，无手改历史迁移。
- 索引覆盖与租户字段匹配：13 个迁移涉及 `tenant_id`，28 处索引声明包含 `tenant_id`；`system_role_data_scope` 使用 `(tenant_id, role_key)` 唯一键（`data_scope_middleware.go:44-48`）。
- 软删除/GORM 命名统一（`gorm.Model` 风格的 `created_at/updated_at/deleted_at`），`duplicate_key.go`、`notfound.go` 统一错误分类。
- 迁移安全性有测试：`migrate_test.go`、`migrate_down_test.go`、`migrate_down_shape_test.go`（工作区）、`health_migrations_test.go`、`audit_migrate_test.go`。
- **幂等性**：`tenant_provision_*` 有重复初始化用例；`MigrateDataScopePolicy` 明确「不在启动时重复回填，避免复活被删策略」，这是正确的取舍。

### 6.2 数据访问与性能（未见系统性缺陷；有限项未验证）

| 关注点 | 结论 |
| --- | --- |
| N+1 | **抽样未见**；未做全量静态扫描与查询日志验证（E） |
| 分页 | 会话列表有专门索引迁移 `000018_session_pagination_indexes`；导出统一 10,000 行上限（`impexp/export_limits.go`） |
| 重复查询 | 热点读有缓存：`userDeptCache`(5min)、`tableExistCache`、tenant mode loader(5s)、i18n cache、token session cache |
| 事务范围 | 角色替换/授权有并发用例（`role_replace_concurrency_test.go`、`role_concurrency_test.go`），未见「请求内长事务」 |
| 缓存失效 | token 按 token 失效；角色策略变更同步 + watcher（`casbin_watcher.go`） |
| 全表扫描风险 | 平台/租户读面限定 `tenant_id`；`1=0` 拒绝分支不走全表 |
| 性能结论 | **仅静态与单测层面**；无压测、无真实执行计划。`dashboard-benchmark.json` 存在但本轮未复跑 |

### 6.3 性能设计判断

**不建议**引入额外缓存层或 CQRS 拆分：当前读路径已通过后台维护器把重活移出请求（`pkg/maintenance`，`auditSvc.RegisterMaintenanceTasks`），这是对的复杂度。

---

## 7. 前端质量与 UI 分析

### 7.1 代码质量（通过）

- 规模：`frontend/src` 254 个 `.ts/.tsx`，67,966 行；108 个 `.tsx` 组件。
- 分层清晰：`core/{layout,router,theme,menu,entitlements}` 壳层，`modules/{auth,system,platform,lowcode,generated}` 业务，`components/{patterns,feedback,data-display,governance,operational}` 设计系统。
- 设计系统确实被用来消除重复：`AppTable`、`SearchToolbar`、`FilterPanel`、`PageContainer`、`PageSplitLayout`、`PageEmpty/PageError/PageForbidden/PageLoading/PageNetworkError/PageServerError/PageSkeleton`、`PermissionAction`、`FormModalFooter`、`TableBatchActionBar`。
- `tsc -b` **通过（exit 0）**，说明类型面完整。
- 13 条 `prebuild` 契约检查（菜单契约、i18n 硬编码、i18n 生成范围、时间展示、壳层视觉、UI 契约、密码可见切换、搜索工具栏、`!important` 预算、系统页准入、smoke base 覆盖、import 路径大小写）——这是**真实存在且会阻塞构建**的约束，不是装饰。

### 7.2 UI / 交互（通过，未做视觉走查）

- loading / empty / error / forbidden / submitting 五态有对应组件与路由级 `RouteContentFallback` + `RouteErrorBoundary`。
- 权限不足由 `RoutePermissionGuard` + `PermissionAction` 处理，避免「按钮可见但点了 403」。
- **本轮未启动浏览器做视觉验收**（按只读与最小验证原则），故不对像素级一致性下结论。

### 7.3 前端问题

- **P3**：`frontend/package.json` 的 `version` 仍是 `0.11.2`，而 `VERSION` 为 `0.18.2`（见 §8 冲突 C3）。
- **P3**：两个死 hook（R3）。
- **P3**：46 个 vitest 文件 / 254 源文件 ≈ 18% 文件级覆盖，UI 逻辑覆盖偏薄；但 smoke（platform/system/business）+ 13 条契约检查提供了补偿，**不建议为此大规模补测**。

---

## 8. 文档资产治理与代码一致性分析

### 8.1 文档资产总览（实测盘点，非估算）

| 口径 | 数量 | 命令 |
| --- | --- | --- |
| `docs/` 下 `.md`（工作区） | **426** | `find docs -name '*.md' \| wc -l` |
| `docs/` 下 `.md`（`HEAD` 提交） | **409** | `git ls-tree -r --name-only HEAD -- docs \| grep -c '\.md$'` |
| 全仓版本化 `.md`（`HEAD`） | **1,124** | `git ls-tree -r --name-only HEAD \| grep -c '\.md$'` |
| 工作区全部 `.md`（含未跟踪工具目录） | 1,931 | 差额 807 为 `.tmp/.omx/.codegraph/.qoder/dist` 等本地噪音，**不计入资产** |
| `.harness/` 跟踪文件 | 1,318 | `git ls-tree -r HEAD -- .harness \| wc -l` |

`docs/` 子目录分布（工作区）：

| 目录 | 文档数 | 定位 | 判断 |
| --- | --- | --- | --- |
| `docs/designs/` | 121 | 设计与规范（中英成对） | 主资产 |
| `docs/harness/` | 135 | 方法/Harness 合同 + 86 份任务包 | 主资产 + 过程记录 |
| `docs/history/` | 38 | 历史归档 | 保留 |
| `docs/reviews/` | 36 | 审计/评审凭证 | 保留（其中 2 份为本轮前新增、未提交） |
| `docs/acceptances/` | 22 | 验收清单/矩阵/模板 | 主资产 |
| `docs/contracts/` | 20 | 权威合同（中英） | **第一权威** |
| `docs/release-notes/` | 19 | 逐版本发布说明 | 保留 |
| `docs/runbooks/` | 5 | 运维规程 | 保留 |
| `docs/archive/` | 4 | 归档样例 | 保留 |
| `docs/frontend/` | 4 | 前端专题 | 保留 |
| `docs/case-studies/`、`docs/operations/` | 2 / 2 | 案例与运维 | 保留 |
| `docs/api/`、`docs/migrations/` | 1 / 1 | 租户 API、升级指南 | **偏薄**（见 8.9） |
| `docs/` 根 | 16 | 部署/版本/测试/治理入口 | 部分为一次性报告（见 8.8） |

### 8.2 元数据与有效性分类（只读门禁实测）

`node scripts/frontmatter-check.mjs --report-legacy` 输出（工作区 426 份）：

- 带 frontmatter：**310**；不含：**116（27%，informational，不强制）**
- 强制目录（4 个）：`docs/superpowers/specs/`、`docs/archive/{examples,baselines,upgrade}`，共 4 份必须带
- 超出 90 天复核窗口的 Active 文档：**149**（不阻断，见合同 §6.5）
- 未迁移 legacy 元数据：**1**（`docs/runbooks/DECOMMISSION_LEGACY_AUTH_ROUTES_2026-12-31.md`）

`check-doc-inventory` / `check-doc-links` 均为 **0 finding**（report-only）。这说明**链接与索引没有断**，但不能证明「内容与代码一致」——后者只能逐项比对，见 8.4。

按 8 类分类（依据：frontmatter 状态 + 目录定位 + 逐项核对；**116 份无 frontmatter 的文档无法凭元数据归类，统一记 UNVERIFIED**）：

| 类别 | 数量 | 代表 |
| --- | --- | --- |
| CURRENT | ≈ 250 | `contracts/*`（20）、`designs/*` 大部分、`acceptances/*`、`harness/*` 核心合同、`release-notes/*` |
| OUTDATED | 2（确认） | `CONTRIBUTING.md`（工具链）、`frontend/package.json` 版本字段 |
| PARTIALLY_OUTDATED | 3（确认） | `DESIGN.md`（§3.3 OIDC 行未反映 HEAD 上死代码存在）、`.harness/STATUS.md`（暂存版本自述 HEAD `ef00738`/`0.18.1`）、`docs/README.md` §4.1（指向的验收报告为历史结论） |
| DUPLICATED | 1（确认）+ 若干成对 | `docs/harness/DOCUMENT_FRONTMATTER_SPEC.md` 与 `docs/contracts/DOCUMENT_FRONTMATTER_SCHEMA.md` 字段定义重复；中英成对 121+20 属**必要重复**，不算问题 |
| CONFLICTED | 3（确认） | 见 8.5 C1/C2/C4 |
| HISTORICAL | 74 | `docs/history/` 38 + `docs/reviews/` 36 |
| ORPHANED | 0（确认） | 链接检查 0 断链；无「无引用且无场景」证据 |
| OBSOLETE | 1（确认） | `database/system_init.sql`（文档已多处声明仅历史参考，但文件仍在根，见 8.8） |
| UNVERIFIED | ≈ 116 | 无 frontmatter，未逐份读 |

### 8.3 过期文档清单（确认项）

| 文档 | 过期内容 | 真值 | 证据 |
| --- | --- | --- | --- |
| `CONTRIBUTING.md`（Prerequisites） | Go **1.21+**、Node **18.x+** | `backend/go.mod:3` `go 1.26.10`；`SHELL_VERSION.json` `minimumNode: 24`；`smoke-core.yml` `NODE_VERSION: "24"` | 同一仓库两份工具链基线 |
| `frontend/package.json:3` | `"version": "0.11.2"` | `VERSION` = `0.18.2`、tag `pantheon-base-v0.18.2` | 版本号双源 |
| `.harness/STATUS.md`（**暂存版本**） | 工作区 HEAD `ef00738`、`VERSION 0.18.1` | 实际 HEAD `4a1fadbb`、`VERSION 0.18.2` | 文档描述的是整改前的分支快照 |
| `DESIGN.md` §3.3（HEAD） | 「OIDC 未实现，除测试与迁移外无引用者」 | HEAD 上 `modules/auth/oidc/` 4 文件确实存在且 0 引用——**描述正确但未指出「与合同冲突的未批准实现」** | `git ls-tree HEAD -- backend/modules/auth/oidc` |

### 8.4 文档与代码一致性清单（核心项逐条核对）

| 文档声明 | 代码事实 | 一致性 |
| --- | --- | --- |
| 租户 CRUD 8 条路由 | `system_modules.go:290-297` 恰 8 条 | ✅ |
| 平台租户运营 5 条路由 | `system_modules.go:302-306` 恰 5 条 | ✅ |
| 迁移 `000030` 承载套餐阈值 | `000030_tenant_plan_definitions.{up,down}.sql` 存在 | ✅ |
| `pkg/tenant.Resolve` 为参考实现、缺失即拒绝 | `tenant.go:112-152` 与文档一致 | ✅ |
| 中间件顺序 / CSP / CSRF | `main.go:136-145` | ✅ |
| 低代码物理路径 `backend/modules/lowcode/{generator,dynamicmodule}` | 目录存在，README 路径正确 | ✅ |
| 生成能力真相源 = 模块注册 + `schema/generated/**.json` | `schema/generated/feature-ledger.json` 存在 | ✅ |
| 「Smoke 为 report-only，不阻塞合并」 | `smoke-core.yml` 头部注释一致 | ✅ |
| 「Harness 基线 1.4.0」 | `SHELL_VERSION.json` = 1.4.0 | ✅ |
| README 不硬编码 release 号 | 版本表指向 `VERSION` + release-notes 索引 | ✅（本轮前已修） |
| `000031` 已 drop 死表 | **HEAD 无该迁移**；仅在 HEAD 的两处注释里出现 | ❌（未提交） |
| 合同 §2.6「业务域不参与租户隔离」 | HEAD 无该章节（整改包 W2 引入，未提交） | ❌（未提交） |
| DESIGN §3.4 租户章 | HEAD 无该章节（整改包 W3 引入，未提交） | ❌（未提交） |
| `SYSTEM_AUTH_CONTRACT.md:168`「未明确身份源前不提前实现 OIDC」 | HEAD 存在 OIDC 包（0 引用、0 路由） | ❌ **代码违反合同** |

### 8.5 文档冲突清单（附权威裁决）

| # | 冲突方 | 具体冲突 | 以谁为准 | 处置 |
| --- | --- | --- | --- | --- |
| C1 | `CONTRIBUTING.md` vs `README.md:84` / `go.mod:3` / `SHELL_VERSION.json` | Go 1.21 vs 1.26.10；Node 18 vs 24 | **`go.mod` + `SHELL_VERSION.json`**（机器可执行） | UPDATE `CONTRIBUTING.md` 的 Prerequisites |
| C2 | `frontend/package.json:3` vs `VERSION` | 0.11.2 vs 0.18.2 | **`VERSION`** | UPDATE 或移除前端 `version`（若该包不单独发布） |
| C3 | `.harness/STATUS.md`（暂存） vs 实际 HEAD | 自述 `ef00738` / `0.18.1` | **实际 HEAD** | UPDATE 后再提交，勿把分支快照写进状态总览 |
| C4 | `SYSTEM_AUTH_CONTRACT.md:168` vs `backend/modules/auth/oidc/` | 合同禁止、代码已写 | **合同**（未经批准的实现不得成为事实） | DELETE 代码（工作区已删，待提交） |
| C5 | `docs/harness/DOCUMENT_FRONTMATTER_SPEC.md` vs `docs/contracts/DOCUMENT_FRONTMATTER_SCHEMA.md` | 两份都定义字段 | **SCHEMA** | MERGE→指针（工作区已改，待提交） |
| C6 | `.harness/tasks/*/manifest.json` vs `docs/harness/tasks/*.task.md` | 两份任务记录体系 | 两者职责不同（机器态 vs 可审阅），`docs/README.md §2.3` 已裁决 | KEEP，但需在 STATUS 里说清「哪个是执行真相」 |

### 8.6 重复文档清单

| 文档对 | 性质 | 处置 |
| --- | --- | --- |
| `DOCUMENT_FRONTMATTER_SPEC.md` ↔ `DOCUMENT_FRONTMATTER_SCHEMA.md` | 真重复（字段定义） | MERGE 到 SCHEMA，另一份改指针（工作区已做） |
| `DESIGN.md` ↔ `docs/designs/BACKEND.md`/`FRONTEND.md` | 摘要 vs 详述（非重复） | KEEP，`DESIGN.md` 只留边界与指向 |
| `README.md` ↔ `docs/README.md` | 入口 vs 索引（非重复） | KEEP |
| 中英成对（`*.md` + `*.en.md`） | 必要双语 | KEEP，但 149 份过期需人工复核轮 |
| 7 份 `TENANT_*` 专题记录 | 结论已被合同取代 | ARCHIVE 到 `docs/reviews/`（工作区已做） |

### 8.7 孤立 / 失效文档

- **池子偏大但无孤儿**：`docs/history/` 38 份、`docs/reviews/` 36 份属证据类，**不应删**。
- `database/system_init.sql`：所有入口文档已声明「仅历史参考，schema 以迁移 + seed 为准」，但文件仍在根目录且名字像权威。**建议 ARCHIVE 或加显式 header**，不建议直接删（下游 ops 快照可能引用）。
- `docs/` 根 16 份中，`testing-implementation-report.md` / `testing-strategy-optimization.md` / `testing-quick-start.md` / `testing-usage-guide.md` 四份测试文档内容重叠，属「一次过程稿 + 三份指南」；建议 MERGE 为一份测试指南（P3）。

### 8.8 文档与代码一致性矩阵（核心文档）

| 文档 | 对应代码/配置 | 状态 | 一致性 | 主要问题 | 建议 |
| --- | --- | --- | --- | --- | --- |
| `README.md` | `VERSION`、`backend/go.mod`、`.github/workflows/*` | CURRENT | 高 | 无（本轮前已把版本/门禁状态改为指向而非快照） | KEEP |
| `README.en.md` | 同上 | CURRENT | 高 | 与中文版同步 | KEEP |
| `DESIGN.md` | `modules/*`、`pkg/tenant` | PARTIALLY | 中 | §3.3 OIDC 行未点明「未批准实现存在」；§3.4 租户章仅在工作区 | UPDATE 并提交 |
| `DESIGN.en.md` | 同上 | PARTIALLY | 中 | 同步滞后于中文 | UPDATE |
| `AGENTS.md` | `docs/harness/*`、`.agents/skills/*` | CURRENT | 高 | 引用 `../pantheon-harness/*`（workspace sibling，存在） | KEEP |
| `CLAUDE.md` | `AGENTS.md` | CURRENT | 高 | 明确「AGENTS.md 为唯一真相源」，无第二套规则 | KEEP |
| `CONTRIBUTING.md` | `go.mod`、`SHELL_VERSION.json` | **OUTDATED** | 低 | 工具链版本错误 | **UPDATE** |
| `CHANGELOG.md` | `VERSION`、`docs/release-notes/` | CURRENT | 高 | 0.18.2 条目存在 | KEEP |
| `docs/README.md` | `docs/**`、`.harness/**` | CURRENT | 高 | §4.1 把历史验收列为一线入口 | UPDATE 轻量 |
| `docs/contracts/TENANT_CONTRACT_V1.md` | `pkg/tenant`、`system_modules.go` | CURRENT | 高（工作区）/ 中（HEAD） | HEAD 缺 §2.6 业务域条款 | 提交整改 |
| `docs/contracts/SYSTEM_AUTH_CONTRACT.md` | `modules/auth/*` | **CONFLICTED** | 低（HEAD） | 合同禁 OIDC、代码有 OIDC | 删代码并提交 |
| `docs/contracts/DOCUMENT_FRONTMATTER_SCHEMA.md` | `scripts/frontmatter-check.mjs` | CURRENT | 高 | 强制目录 4 个，覆盖 310/426 | KEEP |
| `docs/contracts/DOCUMENT_METADATA_AND_STATUS.md` | 同上 | DUPLICATED | 中 | 与 SCHEMA 字段定义重叠 | 改指针（工作区已做） |
| `docs/harness/DOCUMENT_FRONTMATTER_SPEC.md` | 同上 | DUPLICATED | 中 | 第二套定义 | ARCHIVE/指针 |
| `docs/api/TENANT_API.md` | `system_modules.go`、handler DTO | CURRENT | 高 | 仅 1 份 API 文档，其余 API 无文档 | KEEP + 视需要补（非冻结阻塞） |
| `docs/migrations/COMPAT_TO_MULTI_UPGRADE.md` | `tenant.go`、`000020`–`000027` | CURRENT | 中 | 未逐条核对 SQL 细节 | KEEP |
| `docs/DEPLOYMENT_GUIDE.md` | `Dockerfile`、`k8s/*`、`main.go` env | PARTIALLY | 中 | 生成模块 AutoMigrate 例外写在工作区 | UPDATE 并提交 |
| `docs/TENANT_INITIALIZATION_GUIDE.md` | `tenant_provision.go` | CURRENT | 高 | — | KEEP |
| `.harness/STATUS.md` | 实际提交历史 | **OUTDATED（暂存版）** | 低 | 自述 HEAD/版本错误 | UPDATE 后提交 |
| `docs/reviews/PANTHEON_BASE_GLOBAL_AUDIT_2026-10-31.md` | 本轮复核对象 | HISTORICAL（暂存） | 中 | 结论已执行但未提交 | KEEP（提交后转历史） |
| `docs/reviews/PANTHEON_BASE_MAINTENANCE_FREEZE_DECLARATION.md` | 同上 | HISTORICAL（Draft/未签） | — | 未签署，无约束力 | REVIEW（维护者签） |
| `scripts/harness/README.md` | 26 个 checker | CURRENT | 中 | 接线表仅在暂存改动中完整 | UPDATE 并提交 |
| `docs/designs/README.md` | 121 份 designs | CURRENT | 高 | 索引齐全 | KEEP |
| `docs/acceptances/*` | 验收流程 | CURRENT | 高 | — | KEEP |

### 8.9 AI Agent 指令一致性分析（重点）

读取的指令面：`AGENTS.md`、`AGENTS.en.md`、`CLAUDE.md`、根 `agents.md`（workspace 级）、`.agents/skills/*`、`docs/harness/{HARNESS_ENGINEERING_CONTRACT,AI_QUALITY_GOVERNANCE,TASK_PACKET_SPEC,VERIFICATION_EVIDENCE_SPEC}`、`.github/*` 门禁、`scripts/harness/*`。

| 检查 | 结论 |
| --- | --- |
| AGENTS vs CLAUDE | **无冲突**：`CLAUDE.md` 首句即「Follow `AGENTS.md` as the project source of truth」，只追加 Maintainer 三触点与 CodeGraph 用法 |
| Workspace `agents.md` vs 子仓 `AGENTS.md` | **无冲突**：前者声明「不重复子仓规则，只做跨仓协调」，继承方向单一 |
| 规则 vs 实际门禁 | **部分脱节**：`AGENTS.md` 要求「改接口/权限/菜单/i18n/DB 同步更新测试与门禁」，但 11 个 checker 未接线，意味着部分「规则」没有执行者（R10） |
| 是否存在多套冲突规范 | **未发现**。唯一「双份」是 frontmatter 定义（C5），已由 SCHEMA 裁决 |
| Agent 可执行的权威链 | `AGENTS.md` → `docs/README.md` → `contracts/*` → `designs/*`；`.harness/STATUS.md` 只是状态，不是规范 |

**判断：Agent 指令层是本仓库最健康的部分之一**——单一真相源、无第二套并行规则。真正的风险不是「指令互相矛盾」，而是「指令要求的检查有一半没跑」。

### 8.10 推荐的权威文档体系（冻结期最小集）

1. **规范权威**：`docs/contracts/*`（唯一合同层，含 `DOCUMENT_FRONTMATTER_SCHEMA`）
2. **架构权威**：`DESIGN.md` + `docs/designs/{BACKEND,FRONTEND,REPOSITORY_LAYOUT,PERMISSION_MODEL}.md`
3. **执行权威**：`AGENTS.md` → `CLAUDE.md`（指针）→ `.agents/skills/*`
4. **门禁权威**：`.github/workflows/quality.yml` + `scripts/harness/README.md`（必须与实际接线一致）
5. **状态权威**：`.harness/STATUS.md`（只写状态与残留，不写规范）
6. **证据/历史**：`.harness/evidence/*`、`docs/reviews/*`、`docs/history/*`（只追加，不改写）
7. **发布权威**：`VERSION` + `docs/release-notes/*` + `CHANGELOG.md`

**建议归档/合并**：`DOCUMENT_FRONTMATTER_SPEC.md` → 指针；4 份 `testing-*.md` → 1 份指南；`database/system_init.sql` → `docs/archive/`。

**建议删除**：无（除已确诊的死代码文档引用需同步）。

### 8.11 文档体系最终评估（对应 12 问）

1. 文档数量与类型：`docs/` 409（HEAD）/ 426（工作区）；全仓版本化 1,124 份；类型见 8.1。
2. 当前有效：≈ 250（合同/设计/验收/Harness 核心 + 发布说明）。
3. 已过期：**2 份确认**（`CONTRIBUTING.md`、`frontend/package.json`）。
4. 部分过期：**3 份确认** + `docs/DEPLOYMENT_GUIDE.md`（生成模块例外仅在工作区）。
5. 与代码不一致：**5 条确认**（§8.4 ❌ 行）+ **3 条待提交整改**。
6. 存在冲突：**4 份文档确认**（C1/C2/C4/C5），C3 为暂存态漂移。
7. 重复：**1 组真重复**（frontmatter 定义）+ 4 份测试文档内容重叠。
8. 建议归档：**9 份**（1 frontmatter 指针 + 4 测试文档合并 + 7 份 `TENANT_*` 已移 reviews + `system_init.sql`）。
9. 建议删除：**0 份**（历史审计与评审有追溯价值，不得按修改时间删除）。
10. 建议补充：API 文档仅 1 份（`TENANT_API.md`），系统域其余 API 无独立文档——**冻结期不阻塞**，但若对外开放需补。
11. 权威文档是否清晰：**清晰**（合同层单一、`docs/README.md` 索引可用、0 断链）。
12. Agent 指令是否冲突：**不冲突**（§8.9）。
13. 是否适合进入冻结：**文档层基本适合**，但需先提交整改、修 2 处版本漂移、并处理 149 份逾期文档的人工复核轮（不阻断，但要有归属）。

---

## 9. Harness Engineering 与工程治理

### 9.1 结构与机制（真实存在，非纸面）

- 合同层：`docs/harness/` 35 份（核心模型、覆盖模型、任务包规范、证据规范、失败棘轮、退役评审）。
- 执行层：`.harness/` 1,318 文件（任务包、证据、归档、状态）。
- 工具层：`scripts/harness/` 35 个脚本，其中 **26 个 `check-*.mjs`**。
- 门禁：12 个 workflow；`quality.yml` 在 HEAD 有 **51 个 `run:` 步骤**，工作区为 60（新增 9 个接线后被删减/合并）。
- 棘轮回路有实例：`failure-registry` 新登记 `FR-026`，`HARNESS_RETIREMENT_REVIEW` 存在，说明「规则会退役」已被制度化。

### 9.2 问题：11 个 checker 未接线（P2，治理过度）

`HEAD` 上以下 11 个 checker 在 `package.json` / `quality.yml` / `ci.yml` 中**零引用**：

```
check-audit-coverage, check-backend-dto-contract, check-backend-response-contract,
check-doc-frontmatter, check-evidence, check-feature-ledger, check-graph-review,
check-inheritance-contract, check-permission-contract, check-review, check-runtime-evidence
```

含义：这些「规则」从未在任何一次 CI 中执行。工作区整改把它们接进 `Docs Governance`（9 个）与 workspace 根（1 个），但**未提交**。

**判断**：这不是「流程越复杂越成熟」，而是**规则库存大于执行能力**。冻结前必须二选一：接线，或按退役评审标记为 retired。留着不接线的 checker 会让 Agent 与人都误以为约束生效。

### 9.3 重复与成本

| 现象 | 评价 |
| --- | --- |
| `go test -race` 只在 `quality.yml:416` 出现一次 | 合理（前次整改把 coverage 步骤的 `-race` 去掉，避免每 PR 跑两次 race） |
| smoke 为 report-only | 合理：smoke 依赖 MySQL/Redis，作为合并阻塞会制造脆弱门禁 |
| 文档门禁 5 层（frontmatter / links / inventory / metadata / task-packet） | **偏重但有分工**；`check-doc-frontmatter` 未接线是唯一冗余 |
| `.harness/evidence` 在 HEAD **仅 1 个目录被跟踪**，26 个未跟踪 | **证据不落盘**：证据存在本地而非提交里，冻结后不可追溯 → P2 |
| 12 个 workflow | 数量合理（CI/Quality/Security/Release×3/Smoke×2/PR/Branch/Lint） |

### 9.4 CI 实测状态

| 检查 | 结果 |
| --- | --- |
| `go build ./...` | **exit 0** |
| `go vet ./...` | **exit 0** |
| `go test -short ./...` | **exit 0**（全部包 ok，无 FAIL） |
| `tsc -b`（本地 typescript） | **exit 0** |
| `frontmatter-check` | exit 0（426 检查 / 310 带元数据 / 149 逾期） |
| `check-doc-inventory` / `check-doc-links` | 0 finding |
| 未运行 | Playwright smoke、vitest、`-race` 全量、SonarCloud、真实 MySQL/Redis 集成、压测、`npm audit`、`govulncheck` |

> `npx tsc -b` 首次失败是**本机 WSL shim 环境问题**（`execvpe(/bin/bash) failed`），不是项目问题；改用 `node frontend/node_modules/typescript/bin/tsc -b` 后 exit 0。

---

## 10. Pantheon Ops 下游兼容性

| 项 | 实测 | 判断 |
| --- | --- | --- |
| Ops 基线 | `HEAD 1c057eed`（2026-10-31） | — |
| 消费方式 | **完整快照 + overlay 重建**（`foundation-release.lock.json`：`consumerMode: foundation-release-consumer`，`baseCommit a44ae4e0`，`releaseVersion v0.18.2`） | 不是 Go module 依赖，而是同 module 名的源码快照 |
| Base 领先量 | `main` 4a1fadbb 相对 `a44ae4e0` 仅 2 个提交（依赖升级 + 归档） | **无 API 漂移风险** |
| 实际 import | Ops `backend/cmd/server/main.go` 直接 import base 的 `internal/middleware`、`modules/*`、`pkg/*` | **下游依赖 `internal/`**：Base 重构 internal 即破坏 Ops → P2 |
| 前端 | Ops `frontend/src` 有 27 处 `pantheon-base` 引用 | 契约面真实存在 |
| 权限/菜单/审计/i18n 复用 | 通过 system 域共享 + overlay 注入；Ops 自有 `business_overlay_registry.go`、`retired_modules_overlay.go` | 复用成立 |
| Ops 自有副本 | `business/shared/timeout.go`；工作区证据称 Ops 仍保留 `usePagination`/`useRequest` 副本 | 可接受的重复（AUD-012-ops 已登记） |
| 禁止删除项 | `pkg/security/ssrf`（Ops 5 处 import） | **冻结期不得删** |
| Base 为 Ops 加的专门逻辑 | 未发现。`business/retired_modules.go` 是「Base 侧业务模块退役表」，属通用机制 | 通过 |

**结论**：Base 的可复用性成立，**稳定性风险集中在 `internal/` 被下游直接引用**。冻结期建议把「`internal/*` 视为事实契约」写进 `MODULE_CONTRACT`，而不是重构它。

---

## 11. P0–P3 问题汇总

> 「已确认」= 本轮有直接证据；「高风险疑点」= 结构上可推断但未复现；「未验证」= 明确缺口。

| ID | 分类 | 级 | 文件 / 行 | 问题 | 证据 | 影响 | 建议 | 风险 | 置信度 |
| --- | --- | --- | --- | --- | --- | --- | --- | --- | --- |
| **AUD-01** | 工程/交付 | **P1** | 工作区 211 项；`.harness/STATUS.md`（暂存） | 完整的 2026-10-31 整改（13 任务包、OIDC 删除、token 缓存修复、`000031`、9 个 checker 接线）**全部未提交**，且所在快照（`ef00738`）已被 `main` 推进 2 个提交 | `git status` 211 项；STATUS 自述 HEAD `ef00738`/`0.18.1` vs 实际 `4a1fadbb`/`0.18.2` | 整改从未过 CI/评审；「冻结对象」不存在；文档与实际互相引用未提交内容 | 拆 PR 提交（代码/文档/迁移分开），每包跑齐门禁后再宣告冻结 | 中：211 项一次性提交不可审 | 高 |
| **AUD-02** | 安全/多租户 | **P1** | `HEAD:docs/contracts/TENANT_CONTRACT_V1.md`（无 §2.6）；`DESIGN.md` §3.4 仅工作区 | 业务域（`modules/business/*`）**单租户语义在 HEAD 上无合同条款**，而系统域已全面租户化 | 合同 §2.6、`route_groups.go` 注释、README:31 改写均只在工作区 | 运维/Agent 可能给业务路由挂租户中间件，`multi` 模式下直接把业务打到 `TENANT_CONTEXT_MISSING` | 提交 §2.6 与 README/DESIGN 同步改动；加结构断言测试 | 低（纯文档+断言） | 高 |
| **AUD-03** | 可靠性 | **P2** | `HEAD:backend/internal/middleware/token_middleware.go:24,38,60` | `tokenSessionCache` 用 `map[string]*tokenCachedSession`，命中后共享指针，活跃时间戳被并发写 | 前次审计已复现 2 处 DATA RACE；工作区改为值+深拷贝并新增 `token_session_cache_race_test.go` | 高并发下 map/slice 竞态，可能 panic 或读到撕裂的 `RoleKeys` | 提交工作区修复并保留 `-race` 门禁 | 低 | 高 |
| **AUD-04** | 冗余 | **P2** | `backend/modules/auth/oidc/`（4 文件/587 行） | 不可达 OIDC 实现与合同「未明确身份源前不实现」直接冲突 | `git ls-tree HEAD` 存在；`git grep 'auth/oidc' HEAD` 仅自身字面量 + 迁移注释；`SYSTEM_AUTH_CONTRACT.md:168` | 读者误判 SSO 已实现；未批准代码成为事实标准 | 删除包，保留 `000029` 与 `oidc_*` 列（历史迁移不改写） | 低 | 高 |
| **AUD-05** | 工程治理 | **P2** | `scripts/harness/check-*.mjs`；`package.json`；`quality.yml` | HEAD 上 **11 个 checker 未接线**（29 个中） | 逐文件比对 `HEAD:package.json` / `quality.yml` / `ci.yml` | 规则未执行，误以为有约束 | 接线或按 `HARNESS_RETIREMENT_REVIEW` 退役 | 低 | 高 |
| **AUD-06** | 文档 | **P2** | `CONTRIBUTING.md` Prerequisites | Go 1.21 / Node 18 vs 实际 Go 1.26.10 / Node 24 | `go.mod:3`、`SHELL_VERSION.json`、`smoke-core.yml` | 新人按错版本搭建，CI 与本地不一致 | UPDATE | 低 | 高 |
| **AUD-07** | 文档 | **P3** | `frontend/package.json:3` | `version 0.11.2` vs `VERSION 0.18.2` | 两处版本号 | 发布/追溯困惑 | 移除或与 `VERSION` 同步 | 低 | 高 |
| **AUD-08** | 文档 | **P2** | `.harness/STATUS.md`（暂存） | 状态总览写的是整改前分支快照（`ef00738` / `0.18.1`） | `git show HEAD:.harness/STATUS.md` vs 暂存版 | 状态文档不可作为冻结依据 | 提交前改写为实际 HEAD | 低 | 高 |
| **AUD-09** | 治理证据 | **P2** | `.harness/evidence/` | HEAD 仅 1 个证据目录被跟踪，26 个未跟踪 | `git ls-tree` / `git status` | 证据不随代码走，冻结后不可追溯 | 把证据与任务包一起入库 | 低（体积） | 高 |
| **AUD-10** | 架构 | **P2** | `backend/modules/system/system_modules.go:162,163,188,209,238,288,321,346,375,419,430,435,471,513` | 14 处手写同一中间件链 | 逐行比对 | 漏挂 `DataScope`/`PlatformCapability` 会静默产生权限差异 | 抽路由组助手（3 种组合）+ 路由快照测试 | 中：触碰全部系统域路由 | 高 |
| **AUD-11** | 冗余/DB | **P2** | `role_service.go:600`、`cmd/tenantsizing/main.go:89`；迁移 `000031`（仅工作区） | 死表 `permission_role_data_scope_policy` 在 HEAD 仍存在 | HEAD 无 `000031`；代码仅剩注释引用 | 死 schema 增加认知负担 | 提交 `000031`；生产执行走预检 + runbook（human gate） | 中：含 DROP | 高 |
| **AUD-12** | 冗余/前端 | **P3** | `frontend/src/hooks/usePagination.ts`、`useRequest.ts` | 0 消费者 | `git grep HEAD` 非 hooks 目录 0 命中 | 死代码 | 删除（Ops 有自有副本） | 低 | 高 |
| **AUD-13** | 文档 | **P2** | `docs/harness/DOCUMENT_FRONTMATTER_SPEC.md` vs `docs/contracts/DOCUMENT_FRONTMATTER_SCHEMA.md` | 字段定义两套 | 两文件同字段表 | Agent 可能按旧表写元数据 | 合并到 SCHEMA，另一份改指针 | 低 | 高 |
| **AUD-14** | 文档 | **P2** | 149 份 Active 文档 | 超出 90 天复核窗口，无人认领 | `frontmatter-check --report-legacy` | 缓慢腐化，非立即错误 | 安排一次人工复核轮（不脚本批量改） | 低 | 高 |
| **AUD-15** | 下游 | **P2** | Ops `backend/cmd/server/main.go` → base `internal/middleware` | 下游直接依赖 Base `internal/` | import 清单 | Base 重构 internal 即破坏 Ops | 在 `MODULE_CONTRACT` 声明 `internal/*` 为事实契约 | 低 | 高 |
| **AUD-16** | 性能 | **P3** | 全局 | N+1 / 查询计划未系统验证 | 仅抽样 | 未知 | 压测/慢查询轮（非冻结阻塞） | 低 | 低 |
| **AUD-17** | 多租户 | **P3** | `pkg/maintenance`、异步链路 | 后台任务/异步是否携带租户上下文未验证 | 未见租户维度参数 | 潜在跨租户写 | 加租户维度断言测试 | 中 | 低 |
| **AUD-18** | 前端 | **P3** | `frontend/tests` | 46 个 vitest / 254 源文件 | 文件计数 | UI 逻辑覆盖薄（smoke + 契约检查补偿） | 只对新改动补测 | 低 | 高 |
| **AUD-19** | 架构 | **P3** | `pkg/tenant/tenant.go:155-162` | `ResolveForCanary` 仅测试调用 | 生产 0 调用 | 轻微冗余 | 可删或标 Deprecated | 低 | 高 |
| **AUD-20** | 契约脆弱性 | **P3** | `casbin_middleware.go:104-140` | 自助路由白名单硬编码 FullPath+Method | 路由改名即失效（fail-closed） | 维护成本 | 改为路由注册时声明 `SelfService` 元数据 | 中 | 高 |
| **AUD-21** | 文档 | **P3** | `docs/testing-*.md` 4 份 | 内容重叠 | 目录清单 | 读者分流困难 | MERGE 为 1 份 | 低 | 中 |
| **AUD-22** | 文档 | **P3** | `database/system_init.sql` | 名字像权威，实为历史参考 | 入口文档已声明 | 误用风险 | ARCHIVE 或加 header | 低 | 高 |

**P0：0 项。** P1：2 项。P2：10 项。P3：10 项。合计 22 项（已确认 18，证据不足 4：AUD-16/17 与低置信度项）。

---

## 12. 整改建议与优先级

### 12.1 必须立即处理（冻结前）

1. **AUD-01 把整改变成提交**：拆 3 个 PR —— ①代码（token 缓存、OIDC 删除、`000031`、死 hook）②门禁（9 个 checker 接线 + `check-doc-frontmatter` 收编）③文档（合同 §2.6、DESIGN §3.4、STATUS、任务包状态）。每 PR 跑齐 `go test -short`、`go vet`、`tsc -b`、文档门禁。
2. **AUD-03 token 缓存竞态**：随 ① 落地，保留 `-race`。
3. **AUD-02 业务域租户语义**：随 ③ 落地，并补结构断言（防「给业务组挂租户中间件」回归）。

### 12.2 应该处理（冻结前或冻结后首个补丁）

4. AUD-04 OIDC 死代码（随 ①）。
5. AUD-05 11 个 checker：接线或退役，**不留中间态**。
6. AUD-06 / 07 / 08 三处版本/状态漂移。
7. AUD-09 证据入库。
8. AUD-13 frontmatter 双定义合并。
9. AUD-11 死表 DROP 的生产执行（维护者 human gate + 预检）。

### 12.3 可以优化（不阻塞冻结）

10. AUD-10 中间件链抽助手（需路由快照测试）。
11. AUD-12 死 hook 删除。
12. AUD-14 149 份逾期文档人工复核轮。
13. AUD-15 把 `internal/*` 写进 `MODULE_CONTRACT`。
14. AUD-19 / 20 / 21 / 22 小清理。

### 12.4 明确不建议处理

- **不要**做微服务拆分、CQRS、独立 schema 多租户。
- **不要**为实现 6 份 Draft 设计（告警/通知/报表/调度/任务/审批）而增加功能。
- **不要**把 `pkg/security/ssrf` 当死代码删（Ops 在用）。
- **不要**合并 `.harness/` 与 `docs/harness/`。
- **不要**重写/批量删除 `docs/history/` 与 `docs/reviews/`（证据价值）。
- **不要**为覆盖率数字大规模补测试。
- **不要**重构 `internal/`（下游直接依赖）。
- **不要**为「文档数量」而合并中英成对文档。

---

## 13. 最终验收意见

### **B：完成少量整改后进入维护冻结阶段**

**依据（按权重）：**

1. **核心能力可靠**：认证/授权/租户/审计/字典/i18n/组织全部闭环；多租户不是「表 + 页面」，而是上下文解析 + 默认拒绝 + Casbin 域 + 隔离测试俱全（§3、§5）。
2. **安全模型正确**：密码 bcrypt、Cookie Strict/HttpOnly/Secure、CSRF 精确豁免、CORS allowlist、租户解析 fail-closed、Casbin 引擎缺失即 403、数据权限 fail-closed。**未发现 P0 级可利用漏洞**。
3. **可执行基线全绿**：`go build` / `go vet` / `go test -short ./...` / `tsc -b` 均 exit 0（§9.4）。
4. **架构没有过度设计**：唯一的系统性过度工程在治理层（26 checker 中 11 个未接线），业务代码层没有为未来需求预埋的复杂机制。
5. **文档体系权威清晰**：合同单一、索引 0 断链、Agent 指令单源无冲突。

**但不足以直接宣告冻结的原因是三条硬事实：**

- **AUD-01**：一套完整、带证据的整改（211 项）**只存在于工作区**，从未过 CI；而「冻结」的对象是提交。当前 `main` 仍是**未整改**的状态（OIDC 死代码在、token 竞态在、11 checker 未接线、`000031` 不在）。
- **AUD-02**：HEAD 上「业务域不参与租户隔离」没有合同条款，这是**会造成错误加固**的语义空白。
- **AUD-03**：HEAD 上仍存在已复现过的并发竞态。

**冻结路径（建议）：**

1. 按 §12.1 拆 3 个 PR 提交整改，全部门禁跑绿；
2. 合并后由维护者在 `docs/reviews/PANTHEON_BASE_MAINTENANCE_FREEZE_DECLARATION.md` §7 填入**合并提交 SHA** 并签署；
3. 冻结后仅接受：安全修复、依赖升级、文档纠错、以及 §12.3 的纯清理。

**不建议**在整改提交前宣布冻结：那会把「已经被诊断、已经写好、已经验证过但没进 main」的修复留在冻结线之外，之后每次维护都要重新发现同一批问题——这正是 ratchet loop 想避免的结果。

---

### 附：本轮验证命令与结果

| 命令 | 结果 |
| --- | --- |
| `git rev-parse HEAD` / `git status --porcelain` | `4a1fadbb`；211 项（142M/51??/9D/2A/4R/3RM） |
| `git ls-tree -r HEAD -- docs \| grep -c '\.md$'` | 409 |
| `git ls-tree -r HEAD \| grep -c '\.md$'` | 1,124 |
| `git ls-tree -r HEAD -- backend/pkg/database/migrations \| grep -c '\.sql$'` | 60 |
| `go build ./...`（backend） | exit 0 |
| `go vet ./...`（backend） | exit 0 |
| `go test -short ./...`（backend） | exit 0，全包 ok，0 FAIL |
| `node frontend/node_modules/typescript/bin/tsc -b` | exit 0 |
| `node scripts/frontmatter-check.mjs --report-legacy` | exit 0；426 / 310 带元数据 / 149 逾期 / 1 legacy |
| `node scripts/harness/check-doc-inventory.mjs --root .` | 0 finding |
| `node scripts/harness/check-doc-links.mjs --root .` | 0 finding |
| 未运行 | Playwright smoke、vitest、`-race` 全量、SonarCloud、`govulncheck`、`npm audit`、真实 MySQL/Redis 集成、压测 |

**审计限制（显式声明）**：本轮未启动服务、未连接真实数据库/Redis、未做浏览器视觉走查、未跑多租户运行时矩阵与压力测试。凡涉及「运行时行为」的结论均标注为静态推断或证据不足；LINENO 以 `HEAD` 为准，工作区行号可能偏移。
