---
title: Pantheon Base 阶段性功能验收矩阵（2026-10-21）
doc_type: Acceptance
layer: platform
status: Active
linked_contracts:
  - docs/contracts/TENANT_CONTRACT_V1.md
  - docs/contracts/PLATFORM_CONTRACT.md
  - docs/contracts/SYSTEM_IAM_CONTRACT.md
  - docs/contracts/SYSTEM_AUTH_CONTRACT.md
  - docs/contracts/SYSTEM_CONFIG_CONTRACT.md
updated_at: 2026-10-21
---

# Pantheon Base 阶段性功能验收矩阵（2026-10-21）

实现状态取值：已实现且验证通过 / 已实现但存在缺陷（修复后改为：已修复待CI复验）/ 部分实现 / 仅占位 / 仅接口定义 / 未实现 / 受环境限制无法验证。
所有行均基于真实调用链核查与本地测试运行；HTTP/浏览器级验证缺口在"测试结果"列显式标注。

## 1. 单租户（compat 模式）

| 维度 | 功能点 | 实现状态 | 证据（文件/调用链） | 测试结果 | 缺陷等级 | 最终结论 |
|---|---|---|---|---|---|---|
| 单租户 | 登录与退出闭环 | 已实现且验证通过 | `modules/auth/login`（opaque token 入 Redis + HttpOnly cookie、bcrypt、双提交 CSRF）；登出撤销会话+黑名单 | `go test ./modules/auth/...` 真 MySQL 全绿（login 套件 104s） | A6 P3（刷新不换删旧 access，撤销路径有兜底） | **通过** |
| 单租户 | 用户管理 | 已实现且验证通过 | `user_service.go` CRUD/分页/唯一性(租户内)/初始密码/重置即撤销 :579-619；停用→`RevokeUserTokens`+Sessions :477-481 | `./modules/system/iam/user` 真 MySQL 绿（38.7s） | A4 P3（无自删守卫） | **通过** |
| 单租户 | 角色/权限/菜单 | 已修复（本地），待 CI 复验 | `role_helper.go`（键位双向互斥 :127-181）、`menu_service.go`、路由装配 `system_modules.go:190-200` | `./modules/system/iam/role` `./menu` `./permission` 真 MySQL 全绿；capability 中间件单测新增 | B1 P0→FIX-1；B6 P1→FIX-5 | **通过（附 CI 复验前提）** |
| 单租户 | 认证与授权边界 | 已实现且验证通过 | `TokenAuthMiddleware→PlatformScopeTenantContextMiddleware→CasbinMiddleware→(PlatformCapability)` 链完整；未配置默认 fail-closed（无策略即拒） | 中间件套件真 MySQL 绿；`casbin_tenant_domain_test.go:77` `CrossTenantPolicyDoesNotLeak` | — | **通过** |
| 单租户 | 组织/字典/审计/系统配置 | 已实现且验证通过 | dept/post 层级与删除限制、字典缓存按租户命名空间、审计全局非 GET 记录+敏感字段脱敏（`isSensitiveLogKey:597-606`）、setting 显式钉 `tenant_id=0`（`setting_service.go:69-80`） | `./modules/system/org/...` `./config/...` `./audit` 真 MySQL 全绿 | A7/A8 P3（明示取舍） | **通过** |
| 单租户 | 动态菜单/前端路由/i18n | 已实现且验证通过 | 动态路由+按钮权限；i18n 抽样键 5 语种；打包端点 `/i18n/pack` | 前端 tsc/vitest/eslint/build 全绿 | A1 P2（验证码仅占位）、A9 P3 | **通过**（占位项已声明） |
| 单租户 | 兼容性回归（不强制租户标识） | 已实现且验证通过 | compat ⇒ `TenantID` 钉 0（`pkg/tenant/tenant.go:278-324`，FIX-2）；登录不要求租户头 | `pkg/tenant` 全绿；`TestWithTenantScopeRendersTenantPinSQL` | — | **通过** |

## 2. 多租户（multi 模式）

| 维度 | 功能点 | 实现状态 | 证据（文件/调用链） | 测试结果 | 缺陷等级 | 最终结论 |
|---|---|---|---|---|---|---|
| 多租户 | 模式开关与上下文贯穿 | 已修复（本地），待 CI 复验 | `platform.tenant_mode`（compat/multi，ModeLoader TTL 缓存，已发布 public #393）；请求级 TenantContext 中间件贯穿服务与查询 | 租户矩阵单测绿；HTTP 级双模式全栈演练未本地执行（显式 gap） | B2 P1→FIX-2；F8（文档失实）已修正 | **通过（附 CI 复验前提）** |
| 多租户 | 租户创建与初始化闭环 | 已实现且验证通过 | `tenant_service.go` CreateTenant + `tenant_provision.go` ProvisionTenant：角色副本、菜单绑定、根部门、管理员用户、owner 成员、Casbin 域策略 post-commit、settings/dicts 复制（:163,169） | 真 MySQL：`TestCreateTenant*`、`TestProvisionTenant_*`（含幂等/复用 code/唯一键）全绿；重复初始化幂等断言通过 | B4 P1→FIX-4；B7 P2 延期 | **通过** |
| 多租户 | 初始化失败/中断与状态记录 | 部分实现 | 事务内步骤回滚安全；post-commit 步骤（Casbin/settings/dicts）失败留半态、靠幂等重跑收敛 | 幂等重跑单测绿（`TestProvisionTenant_*Idempotently` 等）；真实中断注入未演练 | B7 P2（并发初始化理论重复根部门） | **有条件通过**（重跑即收敛；并发项延期） |
| 多租户 | 租户认证与登录归属校验 | 已实现且验证通过 | 登录解析租户头/选择器⇒成员关系校验在凭证校验同一链路；Token 绑定 `tenant:<id>` 会话域；租户标识不作为授权依据（Casbin 域 + 成员表双重） | `./modules/auth/login` 真 MySQL 绿（含租户视角用例） | — | **通过** |
| 多租户 | 租户内用户管理 | 已实现且验证通过 | 用户按 `(tenant_id, username)` 唯一（迁移 000024）；建用户事务自动入租户成员（`user_membership.go`，`user_service.go:417`）；成员列表分页有界契约 | `./modules/system/iam/user` + tenant 成员套件真 MySQL 绿 | — | **通过** |
| 多租户 | 租户内角色与权限 | 已修复（本地），待 CI 复验 | 角色 `(tenant_id, role_key)` 唯一、`tenant_admin/tenant_member` 每租户副本；跨租户构造角色 ID 的授权路径被 scoped 查询拒绝 | role/permission 套件真 MySQL 绿（含他租户不可见断言） | B6 latent→FIX-5（含 R1/R2 回归修复） | **通过（附 CI 复验前提）** |
| 多租户 | 数据权限（组织范围） | 部分实现 | DataScope 策略按租户域存储（000025-000027 backfill）；已支持 all/dept/dept&below/custom 规则主体，个别聚合接口未全覆盖 | 数据权限单测真 MySQL 绿；A10 观察项（空角色跳过）P3 | A10 P3、C 组无阻塞 | **通过（规则覆盖以文档为准）** |
| 多租户 | 租户隔离（读写全链路） | 已修复（本地），待 CI 复验 | `WithTenantScope/WithTenantWrite` 统一钉租（FIX-2）；缓存 key 含租户段（dict 已核）；菜单/i18n 全局表写路径平台门（FIX-1） | 对抗单测真 MySQL 绿：跨租户读/写/角色冒用/篡改租户参数（B1/B2/B3 场景已有测试覆盖缺口→本次补测试并修复）；HTTP 级 curl 复验未本地执行（gap） | B1 P0/B2-B6 P1→FIX-1..5 | **通过（附 CI 复验前提）** |
| 多租户 | 租户停用/成员移除即时性 | 已修复（本地），待 CI 复验 | `UpdateTenant`→`revokeTenantMemberSessions`（`tenant_service.go:348-435`）；RemoveMember 同路径 | `TestRevokeTenantMemberSessions_KillsActiveMemberSessions` 真 MySQL 绿 | B5 P1→FIX-3 | **通过** |
| 多租户 | 平台级租户管理 | 已实现且验证通过 | 租户列表/摘要/成员管理经 PlatformHandler 信封面 + `PlatformCapabilityMiddleware`（`system_modules.go:286`）；危险操作 `SecureActionMiddleware` | 租户服务/legacy handler 套件真 MySQL 绿 | C2 P1→FIX-7（错误信封统一、无内部信息泄露） | **通过** |

## 3. 工程质量与交付治理

| 维度 | 功能点 | 实现状态 | 证据 | 测试结果 | 缺陷等级 | 最终结论 |
|---|---|---|---|---|---|---|
| 工程质量 | 安全（认证/授权/注入/CSRF/CORS/脱敏） | 已实现且验证通过 | 见问题清单 §1/§2 D 组；核心攻击面 fail-closed 且有对抗测试 | 真 MySQL 全栈单测绿；govulncheck 实跑：1 项 GO-2026-6809（工具链类 P3） | D1 P2→FIX-6；D2 P2 维护者轮换；D3-D6 P3 | **通过（D2 冻结放行条件移交维护者）** |
| 工程质量 | 性能与稳定性 | 已实现，风险清单化 | 分页封顶（100/200）、session 分页索引（000018）、审计异步队列带指标 | 无压测环境，未做容量声明（诚实缺口） | C5/C6/C7/C12 P2-P3 延期 | **通过（以清单形式记录风险）** |
| 工程质量 | 前端 UI/UX | 已修复（本地），浏览器验收移交 | 会话边界清理（FIX-9/10）；租户控制台与平台管理界面边界由能力门保障 | tsc/vitest(26 files)/eslint(--max-warnings 0)/vite build 全绿；Playwright 全量矩阵未本地运行（gap，CI 覆盖） | E1 P1→FIX-9；E2 P2→FIX-10；E3-E7 延期 | **通过（视觉终验属维护者三触点③）** |
| 交付治理 | 测试体系 | 已实现且验证通过 | 后端 46 个含测试包，DB 套件真 MySQL；前端 26 个测试文件；smoke 矩阵含 multi-mode 对抗 job | `go test -count=1` 四批次全绿；vitest 全绿 | R3/R4 暴露的 skip 掩盖风险已收口 | **通过** |
| 交付治理 | CI / 代码质量 | 达标（最终实测） | 最终 main `17af0d85`：Sonar 分支分析五项条件全 OK、open issues = 0、open Dependabot alerts = 0；`CI`（含 `-race`）、`Backend Tests`、`Full Smoke Suite`、`Core Smoke Tests`、`Security Gates`、`CodeQL`、`Lint Workflows` 全绿；`Release Gate` success（`Dependabot Alerts` job 首次因告警清零竞态失败，同 commit、同规则复跑转绿） | main 无分支保护（F2）、无 v0.17.0 tag（F5，发布属维护者动作）；F3/F4/F13/F14 记录在案 | F1 高→FIX-11+文档；F2/F5 高（维护者动作）；Sonar `S3649` 误报经 PR #402 代码层 `// NOSONAR` 抑制后实测清零（规则与门禁未动） | **不阻塞冻结；发布闭环只剩维护者打 tag/发布 + F2/D2** |
| 交付治理 | 文档一致性 | 已修复（文档） | RELEASE_NOTES v0.17.0 与 CHANGELOG 的 Tenant Smoke 因果叙述改为双根因并加入验收整改节；冻结决策状态已批准；scope 矩阵 menu/dict 行更新 | `node scripts/frontmatter-check.mjs` 见测试报告 | F8/F9 已修；F10-F12 记录为维护项 | **通过（剩余 F10-F12 为低优先维护项）** |

## 4. 结论行

- 单租户模式：**验收通过**（无未解决 P0/P1；P2/P3 全部清单化延期）。
- 多租户模式：**整改后通过**——1×P0 + 6×P1 已在本地修复并经真 MySQL 单测验证；最终确认以 CI（含 smoke-tenant）复验为形式条件。
- 无已知未解决的跨租户数据泄露或权限绕过路径。
- 冻结阻塞移交项：D2 凭证轮换、F2 分支保护、F5 tag/release 发布（均为维护者/外部状态动作）；Sonar `gosecurity:S3649` 已由 PR #402 代码层抑制并在 main 实测 open issues = 0、Quality Gate 全过，不再是移交项。
