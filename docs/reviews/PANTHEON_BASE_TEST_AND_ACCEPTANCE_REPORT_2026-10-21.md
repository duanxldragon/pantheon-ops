---
title: Pantheon Base 阶段性验收测试与验证记录（2026-10-21）
doc_type: Acceptance
layer: platform
status: Active
linked_contracts:
  - docs/contracts/TENANT_CONTRACT_V1.md
  - docs/contracts/PLATFORM_CONTRACT.md
updated_at: 2026-10-22
---

# Pantheon Base 阶段性验收测试与验证记录（2026-10-21）

原则：只记录真实执行过的命令与真实结果；失败与环境限制原样保留。时间为本地（Asia/Shanghai），2026-10-21。

## 1. 环境条件（非项目缺陷，但影响验证边界）

| 条件 | 实测 | 影响 |
|---|---|---|
| `git` 直连 github.com:443 超时 | 复现 | 所有 GitHub 状态核查改用 `gh` CLI（API 可达），结论不受影响 |
| 本地 MySQL 3306 存活；默认凭据 1045 | 复现 | 初始 DSN 核查失败；后从仓库根目录已入库的 `.env.test` 取得可用开发 DSN（root@127.0.0.1:3306）。`pkg/testmysql` 仅按测试名创建/删除 `pantheon_base_test_*` 隔离库，不触碰业务库 |
| Git Bash 下 `npm/npx` shim 损坏 | 复现 | 前端门禁以 `node ./node_modules/...` 直调执行 |
| Windows PATH 上的 `node` 是无扩展名 shim | `exec.LookPath("node")` 失败 ⇒ `dynamicmodule` 生成器套件 4 例报 `module.generate.server_export_failed`；设 `PANTHEON_NODE_BIN=D:\nodejs\node.exe` 后通过 | **本地环境怪癖，非代码缺陷**（CI Linux 正常）；已记入限制 |
| 无全栈运行环境（后端进程+浏览器） | 未启动 | Playwright 冒烟、HTTP 级对抗 curl、UI 视觉终验移交 CI 与维护者三触点③；本报告不声称浏览器级通过 |

## 2. 后端门禁（全部真实执行）

```
$ go build ./...            # 无输出，exit 0
$ go vet ./...              # 无输出，exit 0
$ gofmt -l .                # 无输出（含整改后全部文件）
```

### 2.1 无 DSN 全量（DB 套件内部 skip）

```
$ go test ./...             # exit 0；46 包 ok，0 FAIL
```
skip 面抽样（`go test ./modules/system/iam/tenant/... ./pkg/tenant/... ./pkg/database/... -v | grep '--- SKIP'`）：租户服务/ provisioning/成员/会话撤销等 30+ 例因 "mysql test dsn is not configured" 跳过——**此现象曾掩盖 R1-R4，见 §4**。

### 2.2 真 MySQL 全量（决定性证据）

`PANTHEON_TEST_DSN=root:***@tcp(127.0.0.1:3306)/pantheon_base...`、`PANTHEON_NODE_BIN=D:\nodejs\node.exe`，分四批前台执行（两次后台 `./...` 整批运行因包装器截断中断，不作为证据；批次输出如下）：

```
批1  ./cmd/... ./internal/... ./modules/auth/...
  ok cmd/server 0.052s | cmd/tenantmatrixdb | cmd/tenantsizing
  ok internal/middleware 6.787s | internal/scaffold
  ok modules/auth/login 104.034s | mfa | security | session 6.061s      → exit 0
批2  business 0.923s | platform 3.954s | audit 7.668s | config/dict 6.034s
     config/setting 10.645s | system/i18n 114.744s                       → exit 0
批3  iam/menu 4.564s | iam/permission 19.893s | iam/role 27.173s
     iam/tenant 46.992s | iam/user 38.665s | org/dept | org/post         → exit 0
批4  pkg/...（22 包全 ok，含 pkg/database 32.882s 迁移升级/回滚、pkg/tenant）
     modules/lowcode/dynamicmodule 14.361s | generator | modules/system 3.746s → exit 0
```

合计 46 个含测试包全部 ok，0 FAIL，0 SKIP-仅DB。关键新增/更新用例（真 MySQL 断言）：

- `platform_capability_middleware_test.go`：compat 管理员 200 / tenant_admin 403（FIX-1）。
- `pkg/tenant` scope 钉租 SQL 断言 + `setting_tenant_mode_guard_test.go` 降级拒绝（FIX-2）。
- `TestRevokeTenantMemberSessions_KillsActiveMemberSessions`（FIX-3；该测试初版因缺 `.Error` 从未真正执行，修复后通过——见 §4-R3）。
- `TestCopyPlatformDictsToTenant*` 幂等 + `TestCreateTenant_ReusedCodeIsFullyProvisioned`（FIX-4）。
- `role_key_namespace_test.go`（双向互斥、改名排除自身）与 `TestPlatformEditOfTenantRoleKeepsGrantInRoleTenant`（FIX-5/R1/R2）。
- `TestLegacyTenantHandlerErrorEnvelopeIsUnifiedAndLeakFree` 12 子例（FIX-7）。
- `pkg/database` 迁移测试覆盖 `000029` up/down（FIX-8）。
- `TestAddMember_Duplicate`（幂等更新角色并复活）、`TestAddMember_OwnerIsProtected`（owner 拒降级）（FIX-11）。

### 2.3 MySQL + Redis 全量（修正 §2.2 的决定性主张）

§2.2 把「真 MySQL 全量」当作决定性证据，这个定性过高：`pkg/testredis.Open` 在未配置 Redis 地址时同样静默 skip，
所以 §2.2 的 46 包 ok 只覆盖 DB 维度，Redis 维度（会话撤销黑名单、token 失效）当时依然没跑。

```
$ PANTHEON_TEST_DSN=... PANTHEON_TEST_REDIS_ADDR=127.0.0.1:6379 PANTHEON_TEST_REDIS_PASSWORD=*** \
  go test -count=1 ./...          # exit 0；46 包 ok，0 FAIL，日志内 0 次 "redis addr is not configured"
```

同一轮配置下先暴露出 R5：`TestSessionAdminOps_CompatPinsToPlatformTenant` 在本地由绿转红，报错与 CI 完全一致
（`session_tenant_scope_test.go:225: nil-context view must still span tenants, got total=3`）。定位结论：

- **产品代码无错**。`applyAdminSessionFilters`（`backend/modules/auth/session/session_helpers.go:124`）在 `query == nil` 时原样返回 db，
  即内部维护视图按设计不加任何过滤，`Total` 天然含 revoked 行；`ListAllSessions` 另外用 `SUM(CASE …)` 分别给出 `ActiveCount` / `RevokedCount`。
- **测试期望写错**：把 `Total` 当成活跃会话数断言。已改为 `Total = 3 && ActiveCount = 2`，并要求 `platform-compat` /
  `tenant-a-compat` / `tenant-b-compat` 三个会话 ID 全部在场，跨租户覆盖强度不降反升。

方法论记录（比这条用例本身更重要）：本仓库的 skip 与 pass 在 CI 输出里长得一样，
「本地全绿」在未核对 skip 日志前不构成证据。CI 侧实况见 §5。


## 3. 前端门禁（全部真实执行）

```
$ node ./node_modules/typescript/bin/tsc -b                    # exit 0
$ node ./node_modules/vitest/vitest.mjs run                    # Test Files 26 passed (26)，exit 0
$ node ./node_modules/eslint/bin/eslint.js . --max-warnings 0  # exit 0
$ node ./node_modules/vite/bin/vite.js build                   # ✓ built in 716ms，exit 0
```
新增会话边界用例：`routeWarmDataSessionBoundary.test.ts`（4 例，含同页签 A→登出→B 不泄漏预热数据）、`requestOperationTokenSession.test.ts`（3 例）。

## 4. 对抗验证与整改质量记录

### 4.1 多租户对抗场景（单元/真 DB 层）

| 场景（任务书 §5.9） | 覆盖 | 结果 |
|---|---|---|
| A 租户读/改/删 B 租户数据 | scoped 查询 + `CrossTenantPolicyDoesNotLeak`（`casbin_tenant_domain_test.go:77`）+ 各服务租户谓词测试 | 通过 |
| 为他租户用户分配他租户角色 | role/permission 套件（他租户角色不可见/不可授权） | 通过 |
| 篡改请求头/体租户标识 | 中间件测试 + `TENANT_FORBIDDEN`（v0.17.0 #393 路径，body tenantId 指向他租户被拒） | 通过 |
| 租户停用后旧 Token 调用 | FIX-3 会话撤销测试（即时性语义层） | 通过（HTTP 级见 §4.3 gap） |
| 初始化并发/重跑幂等 | `*Idempotently` 系列、`TestEnsureTenantRolePolicies_IsIdempotent` 等真 DB 通过；并发双写场景未注入（B7 延期项） | 部分通过 |
| 租户上下文缺失默认行为 | FIX-2 后 ctx==nil 不加过滤仅限无上下文内部调用；HTTP 链路必有上下文，缺失即 401/403 | 通过（静态+单测） |

### 4.2 CI 证据（gh 实测，2026-10-21）

- `smoke-tenant`（run 39533828246，advisory）红根因取证：trace network JSONL 显示 `POST /api/v1/system/user` 成功返回 id=5 后，`POST /api/v1/system/tenant/205/members {"user_id":5,"role":"member"}` 返回 `{"code":400,"message":"tenant.member_exists"}`；后端日志确认 `ensureTenantMembership` 已在建用户事务内插入 member 行。⇒ FIX-11。
- **同一 commit 上 Tenant Smoke 既红又绿**（job 119005706470 FAIL / 119067519288 PASS，`10 passed`）⇒ 该作业 flaky，上面这条取证只关闭了其中一个真实触发条件，不能表述成「根因已关闭、将恢复绿色」。
- SonarCloud：main 最新 analysis（revision ffb22988）Quality Gate **OK**、unresolved **0**（匿名 API 可达，已验证）。整改分支上则出现新问题，详见 §4.4。
- PR #395（release/v0.17.0 → main）：已合并（本机取证时点为 OPEN，结论已被 §5 的实测取代）。merge commit `762d94a545adb53dd606d8e093704ea8abfc02f4`，tree `f7135e30`；
  `statusCheckRollup` 除 `SonarCloud Code Analysis` FAILURE 外全绿，Docs Governance 的 PR-body 仪式字段红灯已随 #396/#397 转绿，`release/v0.17.0` 分支随合并删除。
  S3649 的二次处置因此改以 `main` 为 base，作为 PR #398 提交。
- main 分支保护：`gh api repos/…/branches/main/protection` → 404 `Branch not protected`（F2，维护者动作）。
- tag `pantheon-base-v0.17.0`：本地与远端均不存在；GitHub Releases 最新 v0.16.1（F5）。

### 4.4 PR #396 复核实况（2026-10-21，gh 实测）

整改 PR #396（`acceptance/remediation-2026-10-21` → `release/v0.17.0`）已合并为 `462216ec`，但合并发生在四项红灯之下，
且合并后 push 触发的 run 39553427164 / 39553427175 复现同样四项：

| 门禁 | 结果 | 取证 |
|---|---|---|
| Backend Tests | 失败 | job 119067894874 唯一失败用例 `TestSessionAdminOps_CompatPinsToPlatformTenant`（§2.3 的 R5） |
| Unit Tests | 失败 | job 119067544459 同一用例、同一断言行 |
| CI Summary | 失败 | job 119069015404 关键项判据只读 `UNIT_TESTS` / `COVERAGE_GATE`；前者 failure、后者 skipped（Coverage Gate 本身未失败） |
| Quality Gates | 失败 | job 119069653739 输入里仅 `Backend Tests: failure`，其余全绿 |
| SonarCloud Code Analysis | 失败 | check-run 119067749735，`Quality Gate failed`：Security Rating on New Code 未达 A；新代码问题 `gosecurity:S3649`（BLOCKER，`tenant_service.go:714`）、`go:S1192`、`go:S3776` |
| Core Smoke Tests / Security Gates / CodeQL / Go Lint / Secret Scan / Docs Governance / PR Governance Prereq / Frontend 全套 / Boundary / Duplication / Change Scope | 通过 | 同一 commit |

⇒ 四条测试类红灯是**一个根因**（R5），跟进提交已修正测试期望；`gosecurity:S3649` 属污点跟踪误报（值只进入绑定的 `?` 占位符），
但为让门禁可读，`AddMember` / `RemoveMember` 的原样 SQL 片段已改为 GORM map 条件（同一批占位符、零字符串拼接），
顺带消掉 `go:S1192` 的字面量重复，并把表驱动断言拆成 `assertScopedSQL` 清掉 `go:S3776`。三者均不改运行语义，
本地 `go build ./...` + MySQL/Redis 下 `pkg/tenant` `modules/system/iam/tenant` `modules/auth/session` 全 ok；
门禁是否转绿只能在下一轮 hosted CI 证明。

**跟进 PR #397 实测（head `f0f650a5`，2026-10-21）**：`CI` 工作流整体 success——`Unit Tests`、`Coverage Gate`、`CI Summary`、
`Fast Checks`、`Go Lint`、`Boundary Gate`、`Frontend Unit Tests`、`Bench Perf Smoke` 全绿；`Code Quality Gates` 的
`Backend Tests`、`Docs Governance`、`Duplication Gate`、`Change Scope`、`Frontend Contract` 全绿。
⇒ R5 的诊断与修复被 CI 证实，四项红灯收敛为一项。
唯 `SonarCloud Code Analysis` 仍红：异味两条（`go:S1192`、`go:S3776`）消失，但 `gosecurity:S3649` 在 `Service.AddMember`
报三条 BLOCKER，说明该规则不看绑定与否——只要请求传入的值参与构造查询就视作 sink。
写路径随后改为对上一步读到的行按主键 + `Select("role","status")` 类型化更新（请求值不再出现在列名或条件位置），
读路径按 id 取行是业务本质，无法在不做抑制的前提下消掉这条边 ⇒ 安全门禁的收口属维护者在 Sonar 侧的误报处置决定，
本轮不加 `NOSONAR`、不做规则排除、不贴 `solo-override`。

**PR #398（`fix/tenant-membership-update-by-primary-key` → `main`，head `2d035ef1`，合并为 `c17ef999`）**：承载上述写路径二次处置与 evidence/文档收口，
diff 为 8 个文件（1 个 Go 文件 + 治理与文档），不含任何 release 重复改动。合并前实测全部 GitHub 门禁 pass，PR 级 Sonar `project_status = OK`、新代码问题 0。
合并后 main 的 push run 实测：`CI` / `Full Smoke Suite` / `Core Smoke Tests` / `Security Gates` / `CodeQL` / `Lint Workflows` 全 success，
唯一红项是 `Release Gate`——分支级 Sonar 分析 `new_security_rating` = E（阈值 ≥ A），`gosecurity:S3649` 在 `Service.AddMember()` 剩两条 OPEN（`:711` 读、`:726` 写）。
⇒ 功能与安全测试维度已全部有据转绿，`Tenant Smoke` 所在的 Full Smoke Suite 这轮为 success（该作业此前在同 commit 上既红又绿，仍按 flaky 处理）；
剩余唯一阻塞是扫描器口径下的污点误报，需维护者在 Sonar 侧处置，处置前不打 `pantheon-base-v0.17.0` tag。

**S3649 处置落地（PR #402，`fix/sonar-s3649-nosonar` → `main`）**：维护者对上一条「需维护者在 Sonar 侧处置」给出了决定——
「既然是误报，那应该可以处理，我希望在代码层面屏蔽，sonar 上的规则我不想动」。
落地方式是 `Service.AddMember()` 两条 sink 各加一行行尾 `// NOSONAR gosecurity:S3649`（改写后行号 `:717` 读、`:732` 写），
Sonar 规则配置、Quality Gate 阈值、`Release Gate` 定义一律不动，未屏蔽失败测试、未贴 `solo-override`。
本地验证（真实 MySQL + Redis）：`gofmt -l` 干净、`go build ./...` 通过、`go vet ./modules/system/iam/tenant/` 通过、
`go test -count=1 ./...` 全绿；其中 `modules/lowcode/dynamicmodule` 首轮因未设 `PANTHEON_NODE_BIN` / `PANTHEON_WORKSPACE_ROOT` 报
`module.generate.server_export_failed`（R6 已知环境项），补齐两个变量后该包 `ok 16.738s`。
**hosted 实测收口（2026-10-21）**：PR #402 的 34 项检查 pass、2 项按设计 skip，PR 级 Sonar `new-code issues = 0`、门禁 `OK`、`new_security_rating = 1`；
合并为 `8c12d4cb` 后 main 分支分析（09:57:38Z）open issues = 0、五项条件全 OK。随后清理两条遗留高危依赖：
#400（`klauspost/compress` 1.18.6→1.18.7，alert #44）合并为 `1783543d`，#403（frontend `follow-redirects` 1.16.0→1.16.1，alerts #45/#46）合并为 `17af0d85`，
open Dependabot alerts 归零。最终 main `17af0d85`：Sonar 分支分析（10:25:38Z）五项条件全 OK、open issues = 0；`Release Gate` success
（首次执行时 `Dependabot Alerts` job 与告警清零存在竞态而失败，告警归零后在同一 commit、同一规则下 `gh run rerun --failed` 复跑转绿，不记为首次即绿）。
⇒ 维护者「Quality Gate 全过、Open issues 清零」的验收条件已由 hosted 实测满足；余下只有 tag/release 发布这一维护者动作。

### 4.3 未执行/无法执行（显式 gap，不得视为通过）

1. **浏览器级 Tenant Smoke 复验**：需后端+MySQL+Redis+前端全栈；本地未运行。FIX-11 的 CI 复验在整改进入 #395/CI 后自动完成。当前"Tenant Smoke 将恢复绿色"是基于取证链的推断，非实测。
2. **HTTP 级跨租户 curl 攻击矩阵**（`cmd/tenantmatrixdb` 全栈模式）：同上未本地执行；单元层等价场景已通过。
3. **真实升级演练**（备份库回放 000025→000029 + 数据兼容性）：迁移测试覆盖 schema 层，数据回放未做。
4. **UI 视觉终验**（两种模式的界面边界/反馈）：代码级与构建级已验证；视觉验收属维护者三触点③。
5. 曾把未推送的整改描述为 CI 恢复——已在文档修正中禁止此类表述；本报告中所有"通过"均限定到实际执行层级。

## 5. 文档与门禁自证

```
$ node scripts/frontmatter-check.mjs
Frontmatter check passed. Checked 335 docs, 242 with frontmatter.   # exit 0
```
本次同步修正：`RELEASE_NOTES_v0.17.0.md`（双根因叙述+验收整改节+验证边界）、`CHANGELOG.md`（同）、`docs/TENANT_SCOPE_FREEZE_DECISION_2026-10-15.md`（状态→已批准+指向）、`docs/designs/TENANT_RESOURCE_SCOPE_MATRIX.md`（menu tenant_id 占位列澄清、平台能力门、字典改 tenant-replicated、updated_at 2026-10-21）。

## 6. 2026-10-22 浏览器级验收与 skip 审计（追加）

§4.3 的两条前端层 gap 已在本机全栈补齐：

- **环境**：原生 MySQL 8（3306）+ Redis 7（6379，均为轮换后口令）、`go run ./cmd/server`（8080）、Vite dev server（5173）、Playwright Chromium 1440×900。
- **路由巡检**：18 条（Dashboard、租户/用户/角色/权限/菜单、部门/岗位/字典/设置、i18n、会话/登录日志/安全事件/操作日志、模块/生成器、安全中心）全部渲染实际内容，`0` console error、`0` 个 `/api/` 响应 ≥ 400。
- **交互**：登录、租户搜索、租户成员抽屉、用户详情、角色编辑（导航 + 按钮授权树）、角色成员、i18n 搜索、会话筛选、当前会话详情均正常；未提交写操作。
- **skip 审计**：开启 `PANTHEON_TEST_DSN_REQUIRED=true` 与 `PANTHEON_TEST_REDIS_REQUIRED=true` 后运行 `go test ./... -count=1`（真实 MySQL + Redis），exit 0 且 `--- SKIP` 计数为 0；required + 无 DSN 的探针按预期 `FAIL`，未设置 required 时按预期 `SKIP`。
- **仍为 gap**：浏览器写路径、跨租户 HTTP 攻击矩阵、生产备份恢复演练。前者由 Go 集成测试/CI Tenant Smoke 覆盖，后两者保持原登记。

证据：[`.harness/evidence/2026-10-22-release-and-residual-closeout/`](../../.harness/evidence/2026-10-22-release-and-residual-closeout/summary.md)。
