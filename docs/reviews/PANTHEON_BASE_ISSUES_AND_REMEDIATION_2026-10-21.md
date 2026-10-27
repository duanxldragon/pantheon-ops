---
title: Pantheon Base 企业级验收问题清单与整改记录（2026-10-21）
doc_type: Remediation
layer: platform
status: Active
linked_contracts:
  - docs/contracts/TENANT_CONTRACT_V1.md
  - docs/contracts/PLATFORM_CONTRACT.md
  - docs/contracts/SYSTEM_IAM_CONTRACT.md
  - docs/contracts/SYSTEM_AUTH_CONTRACT.md
updated_at: 2026-10-22
---

# Pantheon Base 企业级验收问题清单与整改记录（2026-10-21）

审查对象：`github.com/duanxldragon/pantheon-base`，`origin/main = ffb22988`，本地 main 含未推送 release-cut 提交 `ce82b94e`（PR #395，分支 `release/v0.17.0`）。
（收口更新：#395 已合并，网络修复后本地 `main` 已同步到 `origin/main = 73d36ecc`；`ce82b94e` 仍可从两个本地 acceptance 分支到达。见 §5 末条。）
严重度：P0 跨租户泄露/严重绕过；P1 隔离缺陷/核心闭环失效；P2 一般缺陷；P3 建议。
状态图例：**已修复（本地，待 CI 复验）** / **已修复（文档）** / **延期（有理由）** / **维护者动作**。

关联：总报告 [`PANTHEON_BASE_FINAL_ACCEPTANCE_REVIEW_2026-10-21.md`](./PANTHEON_BASE_FINAL_ACCEPTANCE_REVIEW_2026-10-21.md)、验收矩阵、测试报告同目录。

## 1. 阻塞级问题与整改映射（P0/P1/P2 已修复）

| ID | 级别 | 问题 | 证据（修复前） | 整改 | 修复后验证 |
|---|---|---|---|---|---|
| B1 | **P0** | 租户管理员可增删改全局 `system_menu` 并跨租户剥离权限键：`tenant_admin` 持有 `/api/v1/*` 租户域 Casbin 授权，菜单写路由与角色-菜单级联清理（`menu_service.go:634` 按 `tenant_id=0` 的 admin 角色清 `system_role_permission`）无平台能力门 | `backend/modules/system/system_modules.go`（原 190 附近菜单组无 capability 中间件）；对抗测试矩阵中"平台越权改全局资源"零覆盖 | **FIX-1**：`POST/PUT/DELETE /menu` 逐个挂 `PlatformCapabilityMiddleware()`（`system_modules.go:198-200`），i18n 系统写组挂中间件（`:460`）；`/menu/tree`、`/i18n/pack` 保持租户可读 | `internal/middleware/platform_capability_middleware_test.go`：compat 管理员放行、tenant_admin 403（真 MySQL 全绿） |
| B2 | **P1** | compat 模式 `WithTenantScope` 直接返回不过滤；`tenant_mode` 被误切/降级即全量并库 | `pkg/tenant/tenant.go`（原 `if !ctx.IsMulti() { return db }`） | **FIX-2**：ctx 非空即钉 `tenant_id`（compat⇒0，降级免疫）；`WithTenantWrite` 同步（`tenant.go:278-324`）；配置层新增 multi→compat 降级守卫（`modules/system/config/setting/setting_service.go:407,436-455`：存在非 0 未删租户时拒绝切换） | `pkg/tenant` 全绿 + `setting_tenant_mode_guard_test.go` |
| B3 | **P1** | i18n 全局表对租户管理员可写 | `system_modules.go` sysProtected 组（原无 capability 门） | **FIX-1**（同组整改，`:460`） | 同 B1 |
| B4 | **P1** | 新租户默认字典为空，租户内页面下拉无选项 | `tenant_provision.go` 无字典复制步骤 | **FIX-4**：`copyPlatformDictsToTenant`（`tenant_provision.go:169,695-752`），post-commit 第 7 步，INSERT…SELECT + NOT EXISTS 幂等；软删占用唯一键的行不重复插 | `tenant_provision_dicts_test.go`（真 MySQL 幂等复跑通过） |
| B5 | **P1** | 租户停用/成员移除后不吊销在发会话（≤15min 窗口内旧 Token 仍可用） | `tenant_service.go` UpdateTenant/RemoveMember 无 revoke 调用 | **FIX-3**：`revokeTenantMemberSessions`（`tenant_service.go:348-435`），停用与非 active 成员逐一 `tenant.RevokeUserSessionsInTenant`，best-effort + warn 日志 | `tenant_service_session_revocation_test.go`（真 MySQL：仅本租户 active 成员会话被杀，他租户不受影响） |
| B6 | **P1(latent)** | 角色键位互斥是单向时点检查：平台侧新建/改名可占用租户已持有的键，反向继承平台授权 | `role_helper.go`（原仅拒"租户取平台键"） | **FIX-5**：双向互斥 `roleKeyBlockingNamespace`（`role_helper.go:127-181`），软删行不阻塞；改名场景排除自身行 | 见 §3 R1/R2；`role_key_namespace_test.go` + `role_menu_tenant_test.go` 真 MySQL 全绿 |
| C1 | **P1** | OIDC schema 游离于权威迁移链（孤儿 `add_oidc_fields.sql`），multi/生产建库下 OIDC 必坏 | 迁移目录无对应编号脚本；孤儿 SQL 文件 | **FIX-8**：收编为受 guard 的 `000029_oidc_fields.up/down.sql`；`oidc/service.go` 查询与 provision 加租户谓词 | `pkg/database` migrate_test（真 MySQL 升级/回滚通过） |
| C4 | **P2** | `oidc_subject` 全局唯一索引跨租户互斥，违反"租户内唯一" | 旧 SQL 全局 unique | **FIX-8**：替换为 `uk_system_user_tenant_oidc_subject (tenant_id, oidc_provider, oidc_subject)`（`000029:62-73`） | 同上 |
| C2 | **P1** | `tenant_handler.go` 39 处裸 `{"error": err.Error()}` 绕过统一信封且透传内部错误 | 原 handler 全文 | **FIX-7**：错误路径统一 `common.Fail/FailWithError` + sentinel 映射 `failServiceError`；成功数据保持 legacy 裸 JSON 形状（该端点组未挂载、前端只消费信封，兼容性已论证） | `TestLegacyTenantHandlerErrorEnvelopeIsUnifiedAndLeakFree`（12 子用例） |
| E1 | **P1** | 路由预热缓存无会话维度：同页签换账号存在跨账号数据泄漏窗口 | `frontend/src/core/router/prefetch.ts`（warm 数据不清理） | **FIX-9**：`invalidateAllRouteWarmData()`（`prefetch.ts:215`）在 `performLogout` 与 `completeLogin` 调用；warmers 改懒加载避免循环依赖 | `routeWarmDataSessionBoundary.test.ts`（4 例，含同页签 logout→login 模拟） |
| F1 | **高** | Tenant Smoke 常年红且被 advisory 化（`continue-on-error: true`），release notes 宣称已修复=失实 | 运行 39533828246 trace：`POST /system/user`→id 5，`POST /system/tenant/205/members {"user_id":5,"role":"member"}`→`{"code":400,"message":"tenant.member_exists"}`；服务端建用户事务已 `ensureTenantMembership`（`user_service.go:417`），前端加入视角再 POST 撞唯一键 | **FIX-11**：`AddMember` 幂等（存在则更新角色/复活，owner 行拒降级）；同时消除每次"租户视角建用户"必然出现的假性失败 toast。文档修正见 F8 | `TestAddMember_Duplicate`（改为断言幂等成功）、`TestAddMember_OwnerIsProtected`（真 MySQL 通过）。**Tenant Smoke 判定修正**：同一 commit `ce82b94e` 上该作业先 FAIL（job 119005706470，`tenant-viewpoint-switcher.spec.ts:233`）后 PASS（job 119067519288，`10 passed`）⇒ 作业 flaky，FIX-11 只关闭了其中一个真实触发条件，不得表述成单根因收口；浏览器级断言仍待 CI 复验（本地无全栈，显式 gap） |
| D1 | **P2** | 生产未强制 Gin ReleaseMode（debug 日志含路由表与堆栈面） | `cmd/server/main.go` 原无 SetMode | **FIX-6**：`isProductionEnv` ⇒ `gin.SetMode(gin.ReleaseMode)`（`main.go:130-131,253`）；Dockerfile `ENV GIN_MODE=release` | build/vet 通过；cmd/server 测试绿 |
| E2 | **P2** | 二次验证 operation token 登出不清理（sessionStorage 残留） | `api/request.ts` teardown 路径 | **FIX-10**：`clearOperationToken()`（`request.ts:299`），在 `clearClientSession`/`performLogout` 调用（`:106,:347,:365`） | `requestOperationTokenSession.test.ts`（3 例） |

### 既有冻结决策文档所列缺陷（A/B/C 旧编号）的当前结论

- 旧缺陷 A（登录租户选择器 401）：**已修复**（v0.17.0 切片，`platform.tenant_mode` 发布为 public + Bootstrap 提升 `is_public`；CI multi-mode smoke 的 picker 断言已通过，红点仅余 viewpoint membership，即本次 F1/B 组之外的第二条触发条件，由 FIX-11 消除；该 job 同 commit 上先红后绿，属 flaky，不能声称单一根因收口，见 §5）。
- 旧缺陷 B（current 端点）：**已修复**（租户控制台改用信封化 PlatformHandler 面，legacy 裸 JSON 组未挂载；FIX-7 统一其错误路径）。
- 旧缺陷 C（硬删 casbin 策略）：**已修复并收口**（runbook `docs/runbooks/TENANT_TROUBLESHOOTING.md` 存在；角色生命周期 delete/rename 已识别 `role:<key>@tenant:<id>` 形态，见 v0.17.0 Security 条目）。

## 2. 延期项（不阻塞冻结，全部有记录）

### 单租户维度（Task A）

- **A1（P2）验证码只有策略配置、无登录链路实现** — 状态：仅占位。安全文档如实声明未实现即不构成欺骗；建议实现或从策略面撤下开关。延期理由：非验收必需（无对外宣称）。
- **A2（P2）`InvalidateUserPermCache` 从未被调用** — 用户权限键缓存失效函数存在但无调用点；权限变更收敛依赖会话撤销路径。延期理由：与 A3 同族，属收敛时效优化；修复需梳理全部权限变更入口，超出最小整改面。
- **A3（P2）角色禁用/批量改状态不撤销在发会话** — 窗口 ≤ AccessTokenTTL 15 分钟。同上延期。
- **A4（P3）删除用户无"当前登录用户"守卫**（`user_service.go:683-695` 仅拦截内置 admin id=1）。
- **A5（P3）用户黑名单 Redis 读故障 fail-open**（`token_middleware.go:131-162` 两分支不对称；session 黑名单 fail-closed）。
- **A6（P3）刷新不换删旧 access token**（≤15min TTL 内旧 token 仍有效；撤销路径有黑名单兜底）。
- **A7（P3）审计队列满丢弃**（容量 1024，drop-on-full，有 `OperationLogDroppedTotal` 指标与排水；明示取舍）。
- **A8（P3）公开字典 options 端点可匿名枚举**（设计供登录页/表单，影响低）。
- **A9（P3）`permission.denied` 前端无译文**。
- **A10（P3）DataScope 空角色列表跳过即 fail-open 观察项**。

### 多租户维度（Task B）

- **B7（P2）** `ProvisionTenant` 事务边界不完整 + 并发/重跑理论可产生重复根部门与唯一键报错 — 部分实现保护（幂等步骤），延期需并发初始化专项。
- **B8（P2）** `security.password_min_length` 在 user 模块读取未钉平台行。
- **B9（P2）** 新角色默认无 Casbin 策略但控制台暴露角色管理+危险删除（空权限角色语义，非绕过）。
- **B10（P2）** 治理：scope 门禁与 CI 阻塞位缺失（`smoke-tenant` 仍 advisory；建议 FIX-11 经 CI 复验绿后转 blocking）。**已关闭（2026-10-22）**：main 连续 5 次 `smoke-core` 全绿后移除 `continue-on-error`，job 改名 `Tenant Smoke (multi-mode)`；其为 push-only mainline 阻塞信号，不是 PR required check。
- **B11（P3）** 其他散项（无租户上下文内部调用点清单等）。

### 架构/数据库（Task C）

- **C3（P2）** 错误码映射采用率低（统一机制存在）；**C5（P2）** 审计大表缺 (tenant_id, 时间) 复合索引；**C6（P2）** DB 语句无统一超时、多数 service 不透传 context；**C7（P2）** 租户开通 SQL 事务与 Casbin/settings 非原子、批量初始化时长；**C8-C12（P3）** dict 模型索引 tag 漂移、AutoMigrate 双轨残留（12 表）、进程内缓存跨实例不失效/i18n 缓存无租户维度（设计使然）、日志栈三套并存、OFFSET 深分页（已被页大小封顶压制）。

### 安全（Task D）

- **D2（P2，维护者动作）** `.harness/archive` 含真实开发/测试凭证入库且仓库公开（与 SECURITY.md 冲突）。需凭证轮换；历史重写属共享状态变更，不在本地整改范围。**冻结前必须完成轮换**——列入阻塞移交项。**已关闭（2026-10-22）**：MySQL/Redis 口令已轮换（旧口令被拒、新口令可用），跟踪文件清零并完成脱敏；v0.17.0 release 的 `repo.tar` 因构建于脱敏提交之前仍含已失效旧值，作为显式残留登记（建议 v0.17.1 或经授权替换资产）。
- **D3（P3）** 非生产初始管理员口令默认 `123456`；**D4（P3）** 租户指标标签把 ID 当码点转换；**D5（P3）** 两处 LIKE 通配符未转义（仅 LIKE 语义，无注入面）；**D6（P3）** `/metrics` 非生产无鉴权。
- **govulncheck（P3/维护）**：1 个被调用漏洞 GO-2026-6809（crypto/x509 @go1.26.9，1.26.10 修复，经 mfa 路径）——升级工具链即可。
- 其余核心攻击面（CORS/Cookie、CSRF 双提交、Token 存储、fail-closed 撤销、SSRF 防护、脱敏）经复核工程质量高，多处有对抗测试。

### 前端（Task E）

- **E3（P2）** 列表加载普遍缺请求乱序守卫/取消机制；**E4（P3）** ja/ko/fr 各约 190 key 未翻译；**E5（P3）** 建用户/重置密码前端无强度提示；**E6（P3）** `useRequest` 死代码且自带竞态；**E7（P3）** `frontend/package.json version=0.11.2` 与 release 版本脱节。

### 工程化/治理（Task F）

- **F2（高）** main 无分支保护（`gh api branches/main/protection` → 404），红门禁不阻合并 — 维护者 GitHub 设置动作，需授权。**已关闭（2026-10-22）**：ruleset `17011510` active，覆盖 default branch，无 bypass actor；required checks 含 `Quality Gates`/`Security Gates`/`Unit Tests`/`CI Summary`/`SonarCloud Code Analysis`。
- **F3（中）** 12 个 workflow 无 sonar-scanner 步骤，分析外部触发；`release-gate.yml:174` 按 SHA 精确匹配可能误挡 — 维护者流程决策。
- **F4（中）** `check-security-patterns.mjs` 无任何引用（死门禁，手动跑必红 295 errors，无 baseline）。记录为发现；不静默删除/屏蔽。
- **F5（高）** 发布未闭环：v0.17.0 tag 不存在；PR #395 已于 2026-10-21 合入 `main`（`762d94a5`，仅 Sonar 红，见 §5 追加记录），Docs Governance 的 PR-body 仪式字段与聚合红灯已随 #396/#397 收口。剩余缺口只在 tag/release 发布与 Sonar 处置。**已关闭（2026-10-22）**：tag `pantheon-base-v0.17.0` → `728feb2b`，release 四个资产已发布，`Foundation Release Verify` run `39698850076` success。
- **F6（低）** 陈旧分支残留（10 个本地/远端分支内容全部已 squash 并入 main，经 `git cherry` 交叉验证，无未交付修复）；**F7（低）** 历史中 node_modules blob 致仓库膨胀（真实凭证抽查 AKIA/ghp_/RSA 无命中，仅压缩 JS 误影）。
- **F8（高）** RELEASE_NOTES/CHANGELOG 宣称"Tenant Smoke 红因已修复"— **失实**（当时 membership 触发条件未修）。**已修复（文档）**：两处加入 FIX-1…FIX-11 验收整改小节与验证边界声明；"两个独立根因均已关闭"的叙述同样证据不足（同 commit 上该 job 先红后绿，属 flaky），改为"两条真实触发条件，其中 membership 一条由 FIX-11 消除，job 稳定性与最终转绿仍需 CI 复验"。
- **F9（中）** `TENANT_SCOPE_FREEZE_DECISION_2026-10-15` 头部"待批准"与文内 §6.1 维护者裁决矛盾 — **已修复（文档）**：状态改为已批准并指向本清单。
- **F10（中）** `TENANT_INITIALIZATION_GUIDE` §2.2 "事务写 6 步"清单过期（另有 roleMenus/rootDept 及本次 dict 复制步骤）— 建议随 #395 文档收口一并更新。
- **F11（中）** `DEPLOYMENT_GUIDE` 全文无 tenant_mode/multi/compat — 缺口记录，文档维护项。
- **F12（中）** 租户审查快照多份并存结论互斥（09-29 vs 10-14），无 superseded 标记 — 建议加状态头；权威链应为 10-15 冻结决策（含 §6.1/§9.5）→ 本验收报告。
- **F13（低）** `docs/acceptances/` 实为模板+检查清单，真实 evidence 在 `.harness/evidence/`；**F14（低）** `ci.yml:394` `FE_COVERAGE_THRESHOLD=0` 前端覆盖率门禁形同虚设。

## 3. 整改过程自身引入、随后被真实数据库运行捕获并修复的问题（诚实记录）

本地默认无 DSN 时 DB 套件 skip，曾掩盖以下四点；补跑真 MySQL 后全部暴露并修复——这正是 D2/B10 类"skip 掩盖"风险的实证：

- **R1（P2，FIX-5 回归）** `ensureRoleKeyAllowed` 生成的条件串 `tenant_id <>` 缺 `?` 占位符，真 MySQL 下 SQL 1064 语法错。修复：`Where("tenant_id "+column+" ?", value)`（`role_helper.go:164`）。
- **R2（P2，FIX-5 回归）** 反向互斥在"平台管理员编辑租户角色且键不变"时命中被编辑行自身，误报 `role.key.reserved`（`TestPlatformEditOfTenantRoleKeepsGrantInRoleTenant` 捕获）。修复：`ensureRoleKeyAllowed(roleKey, excludeRoleID)`，更新路径传 `role.ID`、创建路径传 0（`role_helper.go:95,115,145-170`）。
- **R3（P3，FIX-3 测试缺陷）** `tenant_service_session_revocation_test.go:55` `db.Exec(...)` 漏 `.Error`，判空恒真——该测试在有 DB 环境必红、无 DB 环境 skip，从未真正验证过。修复占位符后该测试真 MySQL 通过。
- **R4（P3，FIX-4 fixture 缺口）** `tenantTestDDL` 未建 `system_dict_type/system_dict_item`，含 FIX-4 后所有走 provisioning 的租户测试在真 MySQL 下 1146。修复：按 000001/000002/000013 权威形状补两张表 DDL（`tenant_service_test.go:103-133`）。
- **R5（P2，Redis-skip 掩盖 + 断言写错）** `TestSessionAdminOps_CompatPinsToPlatformTenant` 在本地"通过"仅因 `pkg/testredis.Open` 未配地址时静默 skip（`redis.go:73`），CI 才暴露真红：`session_tenant_scope_test.go:225 got total=3`。
  定位：`applyAdminSessionFilters`（`session_helpers.go:124-154`）在 `query == nil` 时不加任何过滤，维护视图 `Total` 按设计含 revoked 行，`ActiveCount` 才是活跃数——错的是测试期望，不是产品代码。
  修复：断言改为 `Total=3 && ActiveCount=2`，并要求三个租户的会话 ID 全部在场（跨租户覆盖比原断言更强）。配 MySQL+Redis 复跑 46 包全绿、日志无 skip 提示。
  **这一条推翻了我此前"真 MySQL 全量=决定性证据"的定性**：只配 DSN 的全量对 Redis 维度等于没跑。
- **R6（P2，门禁掩盖）** 上一轮全量把 `PANTHEON_NODE_BIN` 之外的 scaffold 环境变量留空时，`modules/lowcode/dynamicmodule` 会以 `module.generate.server_export_failed` 硬失败（`internal/scaffold/workspace.go:238-245` 需要 node 与 exporter 脚本）——这是环境缺口而非回归；补齐 `PANTHEON_NODE_BIN`/`PANTHEON_WORKSPACE_ROOT` 后同包 16.0s ok。记录以免被误读为产品缺陷，也提示：本地全绿依赖一小组必须显式设置的环境变量，未设置时的失败/skip 形态各异。

## 4. 剩余风险

1. FIX-1…FIX-11 的最终浏览器级/HTTP 级复验依赖 CI（本地无全栈；见测试报告限制节）。`smoke-tenant` 转 blocking 应以本次整改进 CI 变绿为前提（B10）。
2. D2 凭证轮换未完成前，仓库暴露面持续存在（维护者动作，冻结放行条件之一）。
3. compat⇒multi 升级路径与历史数据归属由迁移与守卫保障，但真实升级演练（备份库回放）未在本次执行。
4. A2/A3/B7-B9 类时效与并发项为已知窗口（≤15min），无跨租户越界后果。

## 5. PR #396 合并实况与新增门禁项（2026-10-21 追加）

- **#396 在四项红灯下合并**（`462216ec` → `release/v0.17.0`）：`Backend Tests` / `Unit Tests` 红于 R5，`CI Summary`（关键项读 `UNIT_TESTS`/`COVERAGE_GATE`）与 `Quality Gates`（输入仅 `Backend Tests: failure`）是其下游聚合；合并后 push run 39553427164 / 39553427175 复现同样四项。其余门禁（Core Smoke Tests、Security Gates、CodeQL、Go Lint、Secret Scan、Docs/PR Governance、前端全套、Boundary、Duplication、Change Scope）全绿。
- **G1（P1，门禁红）SonarCloud Quality Gate**：`Security Rating on New Code ≥ A` 未达（check-run 119067749735）。新代码问题三条：
  `gosecurity:S3649` BLOCKER @ `tenant_service.go:714`（`Service.AddMember()`，污点跟踪误报——值只进绑定 `?` 占位符）、
  `go:S1192` @ `:704`（字面量重复 3 次）、`go:S3776` @ `tenant_scope_sql_test.go:62`（认知复杂度 25）。
  处置（不改运行语义）：`AddMember`/`RemoveMember` 改 GORM map 条件 `membershipScope(tenantID, userID)`（两键恒在，不像 struct 条件会丢零值租户 ID），
  表驱动断言拆出 `assertScopedSQL`。**PR #397 实测**：两条异味消失，`gosecurity:S3649` 仍在 `Service.AddMember` 报三条 BLOCKER
  （`:711` 读、`:722` 写两条）⇒ 该规则把「用请求传入的值构造查询」一律视作 sink，与是否绑定占位符无关。
  二次处置：写路径改为对上一步刚读到的行按主键更新（`Model(&existing)` + `Select("role","status")` + 类型化 struct），
  请求值不再出现在列名或条件位置；读路径按 id 查行是业务本质，无法在不做抑制的前提下消除该污点边。
  **因此 S3649 的最终处置是维护者在 Sonar 侧标记误报（或在门禁规则上让步），本轮不加 `NOSONAR`、不做规则排除、不贴 `solo-override`。**
  其余三项红灯（Backend/Unit/Summary/Quality Gates）已在 #397 实测转绿。
- **G2（P1，方法论）skip 掩盖是复发型缺陷**：本轮两次（R1-R4 缺 DSN、R5 缺 Redis 地址）都是 helper 静默 skip 把红变成绿。
  建议补一条 checker/CI 步骤：当 task packet 声明 runtime-sensitive 且涉及会话/撤销面时，未配置 `PANTHEON_TEST_REDIS_ADDR` 应视为失败而非 skip（属 B10 同族，未在本次实施，显式延期）。
- **合并状态结论**：#397 实测 `Backend Tests` / `Unit Tests` / `CI Summary` / `Coverage Gate` / `Quality Gates` 全绿（四项红灯的根因确认为 R5 一条），
  唯 `SonarCloud Code Analysis` 仍因 `gosecurity:S3649` 误报为红（见 G1 二次处置）。在该项得到维护者处置前，
  `release/v0.17.0` 的测试门禁可称「已证」，安全门禁仍「未证」，不得据此打 `pantheon-base-v0.17.0` tag。
- **PR #395 合并实况（2026-10-21 追加，覆盖上文 F5 与旧状态）**：#395（`release/v0.17.0` → `main`）已被仓库自动化合并，
  merge commit `762d94a545adb53dd606d8e093704ea8abfc02f4`，`statusCheckRollup` 中除 `SonarCloud Code Analysis` FAILURE 外全部为绿
  （Draft PR Handler / PR Smoke Core Optional 为 SKIPPED，非可选门禁失败），且 `release/v0.17.0` 分支已随之删除。
  核对方式：`git/commits/762d94a5` 的 tree 为 `f7135e3066b39ed95024d0df0864b60aa9842bd4`，与本地 release-cut 提交 `597548e4` 的 tree 完全一致；
  `main` 上的 `tenant_service.go` 已含 `membershipScope`、`session_tenant_scope_test.go` 已含修正后的 `Total=3 / ActiveCount=2` 断言，
  证明 #396/#397 的整改内容确实进入 `main`（#395 为 squash 合并，故 `21de18ee` 不是 `main` 的祖先，object API 返回 404 属预期）。
  因此 S3649 二次处置提交（写路径按主键更新 + PR body）的 base 改为 `main`，分支以 Git Data API 重放为
  `fix/tenant-membership-update-by-primary-key`，经 PR #398 合并为 main 的 `c17ef999`（tree `90b4beeb`，与本地 `f02eda1e` tree 一致）。
  v0.17.0 的 tag/release 仍未发布——这不是遗漏：见下方「PR #398 与 main 门禁实况」，`Release Gate` 仍红，处置权在维护者。

- **PR #398 与 main 门禁实况（2026-10-21 追加，取代上文「待下一轮 hosted analysis」的悬置）**：
  #398 合并前全部门禁 pass（`Backend Tests`、`Unit Tests`×2、`Coverage Gate`×2、`CI Summary`、`Code Quality Gates`、`Fast Checks`、`Go Lint`×3、
  `Boundary Gate`、`Duplication Gate`、`Docs Governance`、`Change Scope`、`Frontend Contract`、`Frontend Unit Tests`、`Secret Scan`、`CodeQL`、
  `Security Gates`、`Workflow Security`、`Dependency Vulnerabilities`、`Bench Perf Smoke`），PR 级 Sonar 门禁 `status = OK`、新代码问题 `total = 0`。
  合并后 main（`c17ef999`）的 push run：`CI` / `Full Smoke Suite` / `Core Smoke Tests` / `Security Gates` / `Lint Workflows` 全 success，
  **`Release Gate` failure**——其 `Candidate Checks` 步骤读到 `SonarCloud Code Analysis` 在分支分析里 FAILURE。
  口径差别必须写清：PR 泄漏期只覆盖本次 diff，分支泄漏期覆盖落入 main 的全部新增行，因此「PR 级 0 问题」不等于「分支级门禁绿」。
  分支实测 `project_status`：`new_security_rating` 实际 E（阈值 ≥ A），其余条件（reliability、maintainability、duplication、hotspot reviewed）OK。
  规则存量：main 上 `gosecurity:S3649` 历史共 4 条，全部在 `Service.AddMember()`（2 OPEN @ `:711`/`:726`，2 CLOSED 随被替换的旧行关闭）。
  ⇒ 结论修正：两次保语义重构把 3 条 BLOCKER 压到 2 条，但该规则把「请求值作为参数进入 GORM 构造调用」一律视作 sink，
  按主键 + `Select` 类型化更新只解决了写侧的一条半边（`:722` 的两条并为一条 `:726`），读侧 `:711` 无法在不做抑制或不改变业务定位语义的前提下消除。
  **剩余动作是维护者的门禁处置（在 Sonar 侧标记误报 / 接受风险，或在 `Release Gate` 规则上让步），不是第三次代码 reshuffle。**
  在此之前不得打 `pantheon-base-v0.17.0` tag；本包未加 `NOSONAR`、未排除规则、未贴 `solo-override`。
  取证失误也记在这里：#398 内的记录提交 `f421d9dc`（远端 `3bb7f3d8`）在 PR 尚未跑完时就写下「承载 S3649 的二次处置」「CI 与 Sonar 结论以本轮 hosted run 为准」，
  读者容易把「已提交处置」当成「门禁已收口」；该提交已合并、历史不可改写，故以本条记录显式更正而不是回滚。
- **S3649 最终处置：维护者选择在代码层抑制（2026-10-21 追加，取代上文「处置权在维护者」的悬置）**：
  维护者决定「既然是误报，那应该可以处理，我希望在代码层面屏蔽，sonar 上的规则我不想动」⇒
  `Service.AddMember()` 的两条 sink（读侧 `memberRow`、写侧 `membershipUpdate`）各加一行行尾 `// NOSONAR gosecurity:S3649`，
  Sonar 规则配置、Quality Gate 阈值与 `Release Gate` 定义均未改动，也未贴 `solo-override`、未屏蔽任何失败测试。
  抑制的技术依据（逐条可核）：`tenantID`/`userID` 进入服务前已是 `strconv.ParseUint`/DTO required binding 产出的 `uint64`，
  只经 `membershipScope` 落成 GORM 的绑定 `?` 参数；`role` 是校验后的枚举值，同样只作为绑定参数，`Select("role","status")` 是常量列名。
  该规则把「请求值参与构造 GORM 调用」一律视作注入源，与是否参数绑定无关，因此这条污点边无法在不引入抑制或改变业务定位语义的前提下消除——
  这正是选择行内抑制而非第三次 reshuffle 的理由。
  为配合行尾注释，两处调用从链式换行改写为单语句（`memberRow := …Where(…)` + `err := memberRow.Limit(1).First(…)`；
  `membershipUpdate := …Updates(…)` + `if err := membershipUpdate.Error`），语义等价：查询条件、更新列与错误返回路径逐一对应，未增删功能。
  行号随该改写移动：Sonar 在 `c17ef999` 上报的 `:711`（读）与 `:726`（写）在改写后分别落在 `tenant_service.go:717` 与 `:732`，
  两条行尾注释就放在这两个 sink 行上。
  本地验证：`gofmt -l` 无输出、`go build ./...` 通过、`go vet ./modules/system/iam/tenant/` 通过、
  该包 `go test -count=1 -run 'TestAddMember|TestRemoveMember|TestMember'`（真实 MySQL + Redis）通过。
  **hosted 实测收口（2026-10-21）**：PR #402（`8dd7c960`）合并为 `8c12d4cb`；PR 级 Sonar 新代码问题 `total = 0`、门禁 `OK`、`new_security_rating = 1`，
  合并后 main 分支分析（09:57:38Z）open issues = 0、五项条件全 OK。随后 #400（`klauspost/compress` → 1.18.7）与 #403（frontend `follow-redirects` → 1.16.1）
  两条 Dependabot 高危清理合入（`1783543d`、`17af0d85`），open alerts 归零。
  最终 main `17af0d85`：Sonar 分支分析（10:25:38Z）五项条件全 OK、open issues = 0；`Release Gate` success
  （首次执行时 `Dependabot Alerts` job 与告警清零竞态而失败，告警归零后在同一 commit、同一规则下 `gh run rerun --failed` 复跑转绿，不按首次即绿记录）。
  ⇒ 维护者的验收条件「Quality Gate 全过、Open issues 清零」已在 Sonar 侧满足；剩余发布动作只有维护者打 tag/发布（F5）。

- **传输通道复测与本地主干同步（2026-10-21 追加）**：维护者修复网络后重跑，`git ls-remote --heads origin`、`git fetch origin --prune`、
  `git push --dry-run origin HEAD:refs/heads/transport-check` 对 `github.com` 全部成功（dry-run 只解析 pack，未写任何 ref；`curl https://github.com` 的 HTML 请求仍超时，
  因此只宣称 git-http 通道可用）。fetch 将本地 `ffb22988..73d36ecc` 同步并清理已合并分支（含 `release/v0.17.0` 的远端引用）。
  本地 `main` 原停在 `ce82b94e`（#395 为 squash 合并，故不是 `origin/main` 的祖先，不能直接 ff）：先确认 `ce82b94e` 仍可从
  `acceptance/remediation-2026-10-21` 与 `acceptance/remediation-2026-10-21-followup` 到达，且当前工作分支 tree `ed274ae5` 与 `origin/main` tree 相等，
  再以 `git branch -f main origin/main` 移动指针 ⇒ 无提交丢失、可回退。**当时的结论是「常规 git 通道可用」；2026-10-21 收口轮已修正为「间歇可用」**：
  #402 与本次记录的推送再次遇到 `git push` 连续四次连接超时/重置、`git fetch` 失败，而 `api.github.com` 正常，
  因此仍以 Git Data API 重放（本地 tree 与远端 parent tree 相等才写 ref），本地 `main` 无法 ff，收口提交以一个「tree 等于远端 `17af0d85`」的本地同步提交为祖先。
  ⇒ 后续推送必须逐次复测通道并在失败时回退到 API 重放，不能假设 git 一定可用。

## 6. 2026-10-22 发布与遗留项收口（追加）

证据目录：[`.harness/evidence/2026-10-22-release-and-residual-closeout/`](../../.harness/evidence/2026-10-22-release-and-residual-closeout/summary.md)。

- **F5 发布闭环**：annotated tag `pantheon-base-v0.17.0` → `728feb2b66e0ed5f915cafc785c287fc2b3eca2a`；GitHub release 于 `2026-10-22T00:17:53Z` 发布，含 `foundation-release-pantheon-base-v0.17.0.tgz`、`.tgz.sha256`、`repo.tar`、`repo.tar.sha256` 四个资产；本地复算 SHA-256 与 release digest 一致；`Foundation Release Verify` run `39698850076` success。发布后未替换资产。
- **F2 分支保护**：ruleset `17011510`（`solo dev merge rules`）active，覆盖 default branch，`bypass_actors=[]`；required checks 为 `Quality Gates`、`Security Gates`、`Unit Tests`、`CI Summary`、`SonarCloud Code Analysis`。经典 `branches/main/protection` API 仍为 404，但实际保护由 ruleset 提供。
- **B10 Tenant Smoke**：`smoke-core.yml` 的 `smoke-tenant` 移除 `continue-on-error` 并去掉 advisory 命名；main 最近 5 次运行全绿。该 workflow 为 push-only，故阻塞的是 mainline 运行状态，不是 PR required check。
- **G2 skip 掩盖防护**：`testmysql` 新增 `PANTHEON_TEST_DSN_REQUIRED`（required 时无 DSN 失败，否则 skip），`mysql_test.go` 锁定 fail/skip 决策；`ci.yml` 两个 Go 测试 job 与 `quality.yml` `backend-tests` 设置该标志。实测 required + 无 DSN 为 `FAIL`，未设置时为 `SKIP`；在真实 MySQL + Redis 且两个 required 标志开启下 `go test ./...` 全绿、0 `--- SKIP`。
- **D2 凭据轮换**：MySQL/Redis 旧口令已拒绝、新口令可用；HEAD 跟踪文件旧凭据 `git grep` 命中 0。发布工件 `repo.tar` 因构建自脱敏前提交 `728feb2b` 仍含两份已失效旧凭据（tgz 内嵌同名文件），历史 git 对象同样保留；未在未授权下替换资产或重写历史。建议以 v0.17.1 或维护者授权的资产替换清除。
- **浏览器级终验**：本机全栈（后端 + MySQL + Redis + Vite + Chromium）完成 18 条路由巡检与 8 组交互（租户搜索/成员抽屉、用户详情、角色授权树、角色成员、i18n 搜索、会话筛选、当前会话详情），0 console error、0 个 `/api/` 响应 ≥400；截图与原始结果在证据目录 `artifacts/`、`browser-sweep.json`、`browser-interactions.json`。写操作未在浏览器提交，`AddMember` 幂等与停用/移除成员撤销会话由真实 DB 测试与 CI Tenant Smoke 覆盖。
- **仍未关闭**：迁移前生产备份恢复演练（G2 runbook 的 G2 gate，与 D2 编号无关）、G3 生产量级 sizing、B7 并发初始化专项、跨租户 HTTP 攻击矩阵、v0.17.0 工件中的失效旧凭据清理。
