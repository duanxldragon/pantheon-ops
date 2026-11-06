---
title: Pantheon Base 全局审计 2026-11-04
doc_type: Assessment
layer: platform
status: Active
updated_at: 2026-11-04
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
  - docs/contracts/TENANT_CONTRACT_V1.md
  - docs/contracts/SYSTEM_AUTH_CONTRACT.md
---

# Pantheon Base 全局审计（2026-11-04）

基线提交 `525506035b3a843f85b984beb46866c1329cd964`，审计时间 2026-11-04，裁决 **B**。

## 1. 裁决

**B：完成少量整改后进入维护冻结阶段。**

不是 A：`main` 上的产品缺陷还在，修复只存在于未提交工作区，不能当成已合并。不是 C：本轮没有确认的 P0；安全抽样没有打回运行时阻断项；剩余产品缺口是有限的、已经写了一半的收口，不是架构重做。

冻结对象只能是已合并的 `main`。工作区 49 项未提交改动（16 修改、1 删除、32 未跟踪）**不得**描述为已合并，也不得作为冻结签署的依据。

## 2. 基线（本轮实测，不沿用 2026-11-03 报告）

| 项 | 读数 | 证据 |
|---|---|---|
| HEAD | `525506035b3a843f85b984beb46866c1329cd964` | `git rev-parse HEAD`，2026-11-04 |
| 提交说明 | `52550603 chore(harness): land the 2026-10-31 audit remediation (#488)` | `git log -1 --oneline` |
| VERSION | `0.18.2` | 仓库根 `VERSION` |
| 工作区 | **不干净**。`git status --short` = **49** 行（16 `M`、1 `D`、32 `??`） | 本轮实测。2026-11-03 报告「工作区完全干净」**已失效** |
| 跟踪文件 diff | 17 files，+483 / −1118 | `git diff --stat HEAD` |
| 前端 package.json | `0.11.2` | 与 `VERSION` 0.18.2 是已知的双计数，不是笔误 |
| 发布标签 | `pantheon-base-v0.18.2` 指向 `a44ae4e0`，是 HEAD 的祖先 | ops lock 钉的是这个标签，不是 HEAD |
| ops | 兄弟仓库 `pantheon-ops` HEAD `1c057eed`，main，干净；`foundation-release.lock.json` 钉 `pantheon-base-v0.18.2` / `a44ae4e0` | 符号级核对，未编译 ops |

`.harness/STATUS.md` 最后更新 2026-11-01，仍写工作区 HEAD 为 `ef00738`、VERSION 0.18.1。这是过期快照，不是与代码矛盾的注册表。本轮**没有**改它（冻结宣告归维护者）。

2026-11-03 审计报告与其草稿仍是未跟踪文件，不是 main 上的事实来源。本报告重新核对代码，不沿用其计数或「工作区干净」的陈述。

## 3. 工作区相对 HEAD 改了什么（未合并）

未提交整改覆盖 programme `docs/harness/tasks/2026-11-03-audit-remediation-programme.task.md`（status: Archived，**未跟踪**）的范围内项。programme 自己写明不做：全部 P3、GOV-04/05、GOV-08、冻结签署、`000031` 生产执行。

| 审计 ID | HEAD（main） | 工作区 | 本轮判断 |
|---|---|---|---|
| FE-01 / FE-02 | 仍在。`useUserList.ts:329,355` `pageSize: 9999`；`DeptList.tsx:138` `orgChartPageSize = 1000` | 改为 `fetchAllPages`（pageSize 100，maxPages 200，到顶抛错而不是静默截断） | 原引用点在工作区闭合。**同族残留未修**，见 FE-01b |
| ARCH-01 | `system_modules.go` 14 处手写中间件链 | `SystemTokenGroup` / `SystemAuthGroup` / `SystemProtectedGroup` | 工作区链与 HEAD 等价（高置信）。源码棘轮不锁每组布尔值 |
| DB-01 | `data_scope_middleware.go:61-62` 仍 `AutoMigrate` | 去掉 AutoMigrate，改为 `CREATE TABLE IF NOT EXISTS` | 第二权威已去掉。bootstrap DDL 是模型列的子集，**不是**迁移链的字节等价 |
| GOV-01/02/03、DOC-01/02 | 仍按 11-03 描述存在于 HEAD | 工作区接线与文档修正 | 只在工作区闭合。GOV-05/08 **故意未改** |
| 权限角色下拉 | `PermissionList.tsx:227-236` 与 `permission/index.ts:41` 单页 `pageSize: 100` | **与 HEAD 无 diff**（`git diff` 行数 0） | 未修。见 FE-01b |

工作区跟踪改动还包括：`.github/workflows/quality.yml`、`CONTRIBUTING.md`、`docs/HARNESS_GOVERNANCE_GUIDE.md`、`docs/harness-pr-generator-guide.md`、`docs/harness/failure-registry.md`、`package.json`、两个 pattern checker、`scripts/harness/README.md`，以及删除根目录 `scripts/migrate-task-manifests.mjs`（迁到 `scripts/harness/`，未跟踪）。

## 4. 问题表

严重度：P0 阻断冻结且已确认；P1 冻结前必须闭合的产品缺陷；P2 应记录、不单独阻断冻结；P3 观察，不立项。

状态列区分 **HEAD** 与 **工作区**。工作区闭合 ≠ 已合并。

### 4.1 应该在冻结前处理（有限、已写了一半）

| ID | 类 | 级 | 路径:行 | 描述 | 证据 | 影响 | 修复建议 | 修复风险 | 置信 | 状态 |
|---|---|---|---|---|---|---|---|---|---|---|
| FE-01 | 前端 | P1 | `frontend/src/modules/system/iam/user/useUserList.ts:329,355` | 角色/岗位下拉取 `pageSize: 9999`，后端 `NormalizePagination` cap=100，超出部分静默丢掉 | HEAD 上 `git show` 仍是 9999。工作区改为 `fetchAllPages` | 用户表单角色/岗位选项在 >100 时缺项，操作者看不到缺了什么 | 提交已有的分页聚合，不要把 9999 改成 100 假装修好 | 低。不改 API 契约 | 高 | HEAD 开；工作区原引用点闭 |
| FE-02 | 前端 | P1 | `frontend/src/modules/system/org/dept/DeptList.tsx:138` | 组织架构图 `orgChartPageSize = 1000`，同样被 cap=100 截断，`.length` 当总数 | HEAD 上常量仍是 1000 | 岗位/成员计数在 >100 时偏小 | 同 FE-01，提交已有聚合 | 低 | 高 | HEAD 开；工作区原引用点闭 |
| FE-01b | 前端 | P1 | `PermissionList.tsx:227-236`；`permission/index.ts:41` | 权限工作台角色下拉只取一页 100，把 `result.items` 当全集。预热同调用 | 本轮读文件。`git diff HEAD` 这两文件为 0。`frontend/src` 已无 `pageSize: 9999\|1000\|500\|999\|-1` | 角色 >100 时，权限分配下拉静默缺角色。爆炸半径小于用户表单/组织计数，但是同一类失败 | 复用已有 `fetchAllPages`，两处一起改。不要新造 helper | 低 | 高 | HEAD 与工作区都开 |
| DB-01 | 数据库 | P2 | `backend/internal/middleware/data_scope_middleware.go:57-72`（工作区）；HEAD `:61-62` | HEAD 在 `ShouldAutoMigrate()` 下对单表 `AutoMigrate`，模型成为第二 schema 权威。工作区改为 bootstrap DDL | 本轮 `git show HEAD` 确认 AutoMigrate 仍在。工作区 DDL 见下节 | dev 模式模型改动会静默改表，与版本化迁移分叉 | 去掉 AutoMigrate（工作区已做）。注释不要写「与迁移链逐字相同」——它不是。若「未跑迁移的全新启动」是支持路径，把 `scope_defs`/`created_at`/`updated_at` 补进 bootstrap，或明确该路径不支持 | 低，若只改注释或补列且保持 `IF NOT EXISTS`。不要改 000001 | 高（缺陷）；中（全新库路径是否被支持） | HEAD 开；工作区半闭 |

DB-01 工作区 DDL（`data_scope_middleware.go:64-72`）只有 `id, tenant_id, role_key, mode, dept_ids` 和 `uk_role_data_scope_tenant_role`。迁移链不是这个形状：

- `000001_init_schema.up.sql:367-376` 建表含 `scope_defs`、`created_at`、`updated_at`，唯一键只有 `role_key`。
- `000002` 加 `dept_ids` 并从 `scope_defs` 回填。
- `000024:201-221` 加 `tenant_id`，丢掉旧唯一键，加上 `(tenant_id, role_key)`。

`CREATE TABLE IF NOT EXISTS` 在表已存在时是空操作，**已迁移的库不受影响**。未跑迁移链、只靠这次 bootstrap 起的新库会得到更窄的表。Go 模型（`:45-51`）也不含 `scope_defs` 和时间戳，所以运行时读写不会立刻炸；后续若有迁移假设 000001 的列，会在这条捷径上失败。未跟踪测试 `data_scope_schema_test.go` 只断言模型字段名出现在链和 DDL 里，**不断言 DDL ≡ 链**。注释 `:59-62` 写「must stay identical」过了头。

### 4.2 工作区已闭合、HEAD 仍开（提交即消失，不另立项）

| ID | 级 | 说明 | 残留 |
|---|---|---|---|
| ARCH-01 | P2 | 14 组手写链换成 `System*Group`。对照（架构复核，高置信）：refresh-sync 只有 TokenAuth；user/menu/role/permission/dept/post 为 Token→PlatformScopeTenant→Casbin→可选 Refresh；user 数据域多 DataScope；tenant console 与 i18n console 多 Capability；dict/setting 组级无 Refresh，写路由仍单独挂；upload serve 为 Token→TenantContext（无 Casbin）。`main.go` 无 diff。装配仍是 platform→lowcode→system→auth→business | `system_modules_route_groups_test.go` 只锁调用形态（禁止 `.Group(routeGroupSystem)` 和 `.Use(middleware.`），**不锁**每组 Capability/DataScope/RefreshSync 布尔值。删掉一处 `WithCapability()` 仍能通过源码棘轮。`route_groups_system_test.go` 锁的是助手自身的 handler 顺序，不是每个注册点 |
| GOV-01 | P2 | FR-016/FR-017 标 implemented；quality.yml 对应三步无 `continue-on-error` | FR-017 备注若仍把 `check-inheritance-contract` 说成 base PR 阻断门，则过称。该检查是 npm-only，README 写明不是 base PR 门 |
| GOV-02 | P2 | `check:security-patterns`、`check:ui-tokens` 接入 `package.json` 与 `quality.yml:190-200`，无 `continue-on-error` | 无 |
| GOV-03 | P2 | `scripts/harness/check-checker-wiring.mjs` 接入 `quality.yml:183-188`。干跑：32 checkers / 31 wired / 1 例外（`check-doc-frontmatter.mjs` mirror-only）/ 0 unwired | 未由本协调者重跑干跑，沿用治理复核的命令输出 |
| DOC-01 / GOV-06 | P2 | CONTRIBUTING 不再写 Go 1.21，也不再写 `go run ./cmd/server --migrate` | 示例库名 `pantheon` 与 README `pantheon_base` 仍不一致 |
| DOC-02 / GOV-07 | P2 | 治理指南收成指针；PR 指南不再把已归档 `sonarcloud-s4666` 当模板 | 一处否定路径字面量残留，不构成可用的错误模板 |

`route_groups.go` 新增对 `pkg/tenant` 的导入。该文件在 HEAD 上已经导入 `internal/middleware`。`check-boundaries.mjs` 不扫描 `pkg/contracts`。这不是新的分层违规。`pkg/tenant` 不导入 contracts。auth 生产代码不导入 `modules/system`。

### 4.3 故意留下的治理项（维护者门禁策略，不是本轮代码缺陷）

| ID | 级 | 路径:行 | 描述 | 证据 | 处置 |
|---|---|---|---|---|---|
| GOV-05 | P2 | `.github/workflows/quality.yml:218,223,228,233,238,243,248` 及 `:260,:274` | 这些步 `continue-on-error: true`。`Enforce harness governance gate`（`:277-327`）在 `pull_request` / `merge_group` 上 **exit 0**，只在 push main/release 时看 outcome | 本轮读文件。`git diff` 没有改这些 `continue-on-error` | programme 明确不立项。不要为了形式完整把它们改成阻断。维护者若要收紧，是门禁策略决策，不是冻结前置代码 |
| GOV-08 | P2 | `ci.yml` 前端覆盖率默认阈值 `'0'`，下载步 `continue-on-error` | 治理复核。后端默认阈值 50 是真门 | 同上，维护者一句话。不单独把裁决打成 C |

`check-pr-governance` 在 `quality.yml:203` 与 `:214` 各跑一次，另有 `pr-automation.yml`。重复执行，不是死代码。保留（D）。

Actions 引用：74 处是 40 位 SHA；2 处是 tag：`ci.yml` 的 `actions/download-artifact@v8`（两处）。P3，不阻断。

### 4.4 安全、认证、租户（抽样，无 P0）

不写利用步骤。下列为已确认的控制存在，或已确认的缺口观察。

| ID | 级 | 结论 | 证据 | 分类 |
|---|---|---|---|---|
| SEC-01 | — | 会话缓存是值类型，`RoleKeys` 深拷贝。TokenAuth 先查缓存/Redis，再查黑名单（Redis 失败不放行），再查空闲超时。先前的并发修没有回退 | `token_middleware.go` clone 与校验顺序（安全复核，高） | 已确认闭合 |
| SEC-02 | — | Refresh 轮换重写 RefreshJTI 并重签；handler 在 Redis 校验后删除旧 refresh key。防重放靠删旧 key，不是第二次 JTI 比较（payload 只有 uid/sid） | `session_service.go:159-169`，`login_handler.go:307-323`，`token.go:186-217` | 已确认。设计如此，不是缺陷 |
| SEC-03 | — | 登出：`RevokeSession`（DB `revoked_at` + 黑名单 + 撤销 refresh，错误向上返回）并删当前 access、清进程缓存。其他进程靠黑名单，不靠缓存同步 | `login_handler.go:685-706`，`session_service.go:250-280` | 已确认 |
| SEC-04 | — | 登录限流不能靠空 IP 绕过：空 IP 变成 `ip:unknown`。账号失败在 bcrypt 之前计数。`SourceMaxFailedAttempts<=0` 会关掉来源限流，这是策略关闭，不是漏洞。默认 20。login/refresh/MFA 另有速率限制 | `login_handler.go:173,742-749`；`login_service.go`；`login_runtime.go:41`；`auth/module.go:34-51` | 已确认 |
| SEC-05 | — | 租户解析：compat 恒为 0。multi 下 `X-Tenant-Id` 仅在 HeaderAllowed（平台能力中间件设置）时接受，否则拒绝。主体 claim 拒绝 0。无 header/claim 时，只有平台范围 + 平台能力才回落到 0，否则缺上下文。请求体不参与 | `pkg/tenant/tenant.go:123-152`；`tenant_context_middleware.go:96-100` | 已确认。业务域 V1 故意不挂租户中间件（`ProtectedGroup`，合同 §2.6） |
| SEC-06 | — | 用户列表走 Scoped + WithDataScope；审计非平台主体带 tenant 过滤返回 403；设置列表缓存键在 multi 下带 `t{id}:` 前缀；上传 10MB 全局体限制，对象键拒绝 `..` 与空段，租户前缀只来自已解析的 gin 上下文 | `user_service.go:228`；`audit_handler.go:30-38`；`setting_service.go:245-254`；`main.go:139`；`setting_handler.go:213-221`；`upload/service.go:484-492,605-621` | 已确认（抽样） |
| SEC-07 | P3 | `POST /tenant` 创建与 `PUT /tenant/:id` 更新没有 SecureAction。删除、批量删除、重置密码、数据域修复、成员移除、菜单删除、设置组更新、日志清理、会话撤销等破坏性写有 SecureAction | `system_modules.go` 创建/更新约 308-309；破坏性写见安全复核清单 | 观察。产品是否要求这两处也做 step-up：**未验证**。不要发明需求 |
| SEC-08 | — | 全路由「无认证」普查、CSRF 豁免精确表、MFA TOTP 加密、seed/authz SQL 正文 | 未做完 | **证据不足**。不得从抽样写成「全站无洞」 |

`FailWithError` 对无 i18n key 的错误返回兜底文案，原始错误只进日志。它盖不住每一处 `Fail(c, …, err.Error())`。P3 观察，未逐调用点普查。

硬编码口令 `admin123` / `changeme` / `password123` 只出现在测试与 i18n JSON，未在生产路径确认。

### 4.5 数据库与性能（静态，无运行时故障声明）

| ID | 级 | 路径 | 描述 | 分类 |
|---|---|---|---|---|
| DB-02 | P2 | `menu_model.go:8-30`；`000020:126-132` | `SystemMenu` 无 `TenantID`、无 `DeletedAt`。SQL 有 `tenant_id` 与 `idx_system_menu_tenant_parent`（无 IF 守卫）。`menu_service.go:25-30` 的 `Migrate()` 仍对 `SystemMenu` 调 `AutoMigrate`——这是菜单模型自己的运行时迁移，与 DB-01 不是同一处，本轮未把它升成新的冻结阻断 | **已确认**模型/表漂移。运行时是否跨租户泄露菜单：**未验证**。`GetMenuTree`（`:34-80`）按 `tenant.IsTenantScoped` 在内存里滤平台专用菜单，SQL 查询本身不加 `tenant_id`。不要从静态分析写成已确认的跨租户泄露，也不要建议重构菜单租户模型 |
| DB-03 | P2 | `user_service.go` UpdateUser 提交后 `revokeUsersAccess`；BatchUpdateUserStatus 的状态更新与撤销循环不在同一事务；`CreateRole` 提交后 `reloadRolePolicies()` | 提交成功、副作用失败时，旧 token 仍有效，或 DB 已变而 Casbin/缓存未变 | **已确认**的静态缺口，不是已复现的事故。已在 main 上，不是这次整改引入的。不要把「收成一个大事务」写成冻结前置 |
| DB-04 | P2 | `000031` up.sql `DROP TABLE IF EXISTS permission_role_data_scope_policy`；down 只建表、注释写明不恢复行；preflight 只 SELECT | 旧表有行且替换表为空时，数据永久丢失 | 维护者 human gate。**不要执行**。本审计未连库 |
| DB-05 | P3 | `000005` 无守卫 `DROP INDEX` / `DROP COLUMN dept_code` | 历史不可逆。已在 main | 陈述即可，不是新的冻结阻断 |
| DB-06 | — | 菜单树、部门树、GetOverview 无 LIMIT | 需要压测才能谈性能 | **证据不足**，不声明快或慢 |
| DB-07 | P3 | 权限缓存键 `user_perms:%d` 只有 user id，注释写明一用户一家租户。TTL 5 分钟。`InvalidateUserPermCache` 本轮 grep 只在 `login_runtime.go` 一个文件 | 角色/权限变更后最多 5 分钟旧权限，除非有间接调用 | 中置信。不是跨租户键碰撞的已确认缺陷 |

`tenant_id` 的 ADD INDEX / ADD UNIQUE INDEX **语句数是 27**，不是 11-03 报告的 28。000013×2、000014×2、000015×2、000016×1、000017×2、000020×9、000024×7、000028×1、000029×1。000020 与 000024 在 IF 守卫下重复声明同名索引，落地的不同索引少于 27。`uk_permission_role_data_scope_policy_tenant` 随 000031 一起消失。

软删与唯一键：MySQL 唯一索引不含 `deleted_at`。删除时改名释放键（用户 `__deleted_user_`、角色 `__deleted_role_`、岗位 `__deleted_post_`、字典、租户 code）。这是已确认的释放策略，不是推断。

列表 N+1：抽到的用户/角色列表是批量加载，菜单树与部门树是一次 Find 加内存组树。`refreshChildAncestors` 在更新事务内按子节点递归 Update，是放大，不是列表 N+1。

事务内的写：CreateUser、DeleteDept、UpdateDept、BatchUpdateDeptLeader。BatchUpdateDeptStatus / BatchUpdateRoleStatus 是单条 UPDATE，不是多写缺口。

未验证：现场 `information_schema` / `schema_migrations`；ExportAdminSessions 是否无界；限流 Redis 键形状；RevokeUserSessions 是一条 SQL 还是按会话循环。

### 4.6 前端其余项（不升格）

| ID | 级 | 说明 | 分类 |
|---|---|---|---|
| FE-03 | P2 | 删除成功等路径重复 toast | 前端复核，高。本协调者未逐行重读 |
| FE-04 | P2 | 删掉当前页最后一行不回退页码 | 同上 |
| FE-05 | P2 | prettier 漂移 | 同上。本轮未跑 prettier |
| FE-06…FE-14 | P3 | 交互打磨 | 不立项。programme 已排除 |

`DeptOrgTab` 仍用数组 `.length` 当岗位/成员数。在工作区里，输入数组来自聚合后的全集，所以 FE-02 的「静默算错」在那些调用点上不再成立。在 HEAD 上，源页面仍被 cap，计数仍错。

`fetchAllPages` 的已知边界：后端少报 `total` 且返回短页时，helper 提前返回，仍会静默偏短。`total` 虚高且每页都满时，到 maxPages 抛错（可见，不是请求风暴：200×100）。单测覆盖了恰好 100、202 跨页、150 边界、空集、maxPages 抛错。**单测本轮未执行。**

### 4.7 文档（计数是清点，不是估计）

范围合计 **1166** 个 markdown：根 11、`.github` 4、`.harness` 717、`docs/` 434。`docs/` 子目录：harness 141、designs 121、history 38、reviews 38、acceptances 22、contracts 20、release-notes 19、docs 根 16、runbooks 5、archive 4、frontend 4、case-studies 2、operations 2、api 1、migrations 1。`*.en.md` 100（docs 96 + 根 4）。

| 分类 | 对象 | 依据 |
|---|---|---|
| CURRENT | `TENANT_CONTRACT_V1`（含 §2.6 业务域单租户，中英）、`PERMISSION_MODEL.md`（Casbin 中间件在）、`SYSTEM_IAM_CONTRACT.md`（边界与目录一致）、README 不硬编码 release 号 | 与代码对照 |
| CONFLICTED | `docs/designs/TENANT_READY_SINGLE_TENANT_DESIGN.md` status Active、更新于 2026-05-04，仍写没有真实租户管理、没有租户隔离中间件 | 代码有 `pkg/tenant/tenant.go:123` Resolve 与租户路由。合同是当前权威，这份设计不是 |
| PARTIALLY_OUTDATED | `MULTI_TENANT_DESIGN.md`（愿景，合同优先）；`AUTH_MODULE_DESIGN.md`（仍把 `backend/modules/system/user/` 当当前问题）；`AUTH_PROVIDER_ABSTRACTION.md`（Active 但没有 provider 类型，OIDC 包已删）；`DATA_PERMISSION_HOOK.md`（hook 在，「所有 List 必须注入」是规范，覆盖未验证） | 文档复核 |
| UNVERIFIED | `PERMISSION_MENU_DECOUPLING.md` 自标 in progress（2026-06-22） | 未重核菜单/按钮列表回退 |
| HISTORICAL | `docs/history/README.md` 自标历史；`docs/archive` 4 份 status Archived | 不要删 |
| ORPHANED（索引） | `docs/README.md` 不链 `docs/history` 与 `docs/archive` | 索引缺口，不是文件该删 |
| PARTIALLY_OUTDATED | 四份测试文档数字过期：`testing-strategy-optimization.md` 等仍写 86 个 `*_test.go`、28 个 spec | 实测后端 `*_test.go` = 197；前端 smoke-core spec = 14 |
| PARTIALLY_OUTDATED | CHANGELOG 最新条 `[pantheon-base-v0.18.2]` 写 schema 仍是 30 | 对标签 `a44ae4e0` 为真；对 HEAD（已有 000031）为假。读的人必须看自己站在哪 |

DESIGN 模块切分与目录一致。`main.go:167-175` 装配顺序与 DESIGN 一致。DESIGN §1「未来多租户」与 §3.3「已交付」是同一文件里的语气冲突，不是两条事实。

根 `AGENTS.md` / `CLAUDE.md` 一致：规划者不得改 `backend/` 与 `frontend/src/`。`frontend/AGENTS.md` 开头是自主执行口吻（OMX 模板）。这是指令冲突，不是产品缺陷。本审计以根规则和用户只读约束为准。

不要批量删除历史文档。冲突设计改 status 或加「已被合同取代」一句即可，那是文档治理，不是冻结前置的运行时修复。

## 5. 冗余分类

| 符号 / 文件 | 类 | 调用方 | 公开 API | ops 使用 | 删的收益 | 删的风险 | 处置 |
|---|---|---|---|---|---|---|---|
| `retired_modules.go` 空切片 `retiredBusinessModules` | D 保留 | ops 通过 `retired_modules_overlay.go` 的 `init()` 追加 | 是，overlay 钩子 | 是 | 无 | ops 覆盖点消失 | 不删 |
| `pkg/security/ssrf` | D 保留 | base 内 + ops 5 处 import | 是 | 是 | 小 | ops 编译断 | 不删 |
| `ProtectedGroup` / `DataScopedGroup` / `ProtectedGroupWithRedis` | D 保留 | base 业务域；ops 符号级使用前两个 | 是 | 是（前两个） | 无 | 合同 §2.6 与 ops 同时断 | 不删。ops 未使用新的 `System*Group`（0 命中），这不使旧助手变死 |
| `check-pr-governance` 跑两次 | D 保留 | CI | — | — | 一次 CI 分钟 | 无功能收益，不值得为去重改门禁 | 不删 |
| `scan-tree.mjs` | D 保留 | 被两个 pattern checker import | — | — | 无 | 检查器断 | 不是死代码 |
| `scripts/migrate-task-manifests.mjs` 根副本 | B 重复 | 工作区已删根文件，未跟踪副本在 `scripts/harness/` | — | — | 去重 | 若有外部脚本钉旧路径会断。仓库内引用需随提交一起改 | 工作区方向对。未合并前 HEAD 仍是根路径 |
| 四份 testing-*.md | B 重复 + 过期 | 文档入口 | — | — | 读者少走弯路 | 删历史会丢决策记录 | 不删。标过期或收成一份指针，另作文档治理 |
| `TENANT_READY_SINGLE_TENANT_DESIGN.md` | 不是死代码。是 CONFLICTED 文档 | — | — | — | — | 删了丢 2026-05 的决策背景 | 不删。改状态 |
| 路由助手「只有一个调用方」 | D 保留 | 系统域注册 | 是 | System* 组 ops 未用，旧组 ops 在用 | 无 | 把共享合同助手当死代码删掉会伤 ops | 单调用方不是死代码 |

未做全仓调用图。凡没有逐符号核对调用方的，一律 **E 证据不足**，本表不列「未使用」。

## 6. 架构与模块边界

装配点 `backend/cmd/server/main.go`：platform → lowcode → system → auth → business。auth 只通过 `backend/pkg/contracts/authuser` 消费系统用户。工作区没有改这条装配。

系统域工作区链的顺序是安全属性：Casbin 在 DataScope 之前，RefreshSync 最后（`route_groups.go:137-149`）。业务域 `ProtectedGroup` 故意没有租户中间件。不要「修」它。

低代码 `dynamicmodule` 在 HEAD 上就导入 `modules/system/i18n`。边界检查器不扫描 lowcode。不是这次 diff 引入的。角色服务导入同子域 `iam/permission`，允许。

## 7. 下游 pantheon-ops

ops 是物理快照分叉（同一 Go module path，无 replace），不是 module 依赖。lock 钉标签 `a44ae4e0`，不是 base HEAD `52550603`，更不是未提交工作区。

| 核对 | 结果 |
|---|---|
| 业务代码 import `backend/internal/` | 0 |
| import `modules/{system,auth,platform,lowcode}` | 0 |
| import `pkg/` | common 59、database 19、testmysql 18、impexp 14、contracts 8、security/ssrf 5、logging 4（文件命中） |
| contracts 实际用到的符号 | `BackendModule`、`FuncModule`、`RegisterBackendModules`、`ProtectedGroup`、`DataScopedGroup`。未用：`ProtectedGroupWithRedis`、`runtime_settings`、`authuser`、三个新 `System*Group` |
| `ProtectedGroup` / `DataScopedGroup` 签名 | ops 副本 vs base 标签 `diff -q` 退出 0；vs 已提交 HEAD 只有注释差 |
| `MigrateDataScopePolicy` 签名 | 未变。行为只在「用这份工作区重建且 `PANTHEON_AUTO_MIGRATE=true`」时变：不再用 GORM 改已有表 |

**一句话：当前未提交的 base 改动不会在符号级打断已锁定的 ops 快照。** 未验证：ops 的 `go test`、overlay 重建、ops 副本上的前端分页行为、`.foundation/allowed-drift.json` 相对 HEAD-vs-tag 的漂移、GitHub 上的 weekly workflow 是否真在跑。

## 8. 本轮执行过的检查

| 命令 | 结果 | 范围 |
|---|---|---|
| `cd backend && go build ./... && go vet ./...` | 退出 0，无输出 | 含工作区 |
| `go test -short -count=1 ./internal/middleware/ ./pkg/contracts/ ./modules/system/ -timeout 180s` | 三包 ok（0.104s / 0.082s / 0.044s） | 含未跟踪的路由组与 data-scope 测试。**不是**全仓 `./...` |
| `frontend` 下 `./node_modules/.bin/tsc -b --pretty false` | 退出 0 | 未跑 eslint、prettier、vitest |

未跑：服务启动、Playwright smoke、压测、连库、Sonar、渗透、全仓 `go test -short`、`fetchAllPages` 单测、ops 编译。静态绿不等于运行时绿。

## 9. 冻结前的有限清单（做完这些就可以签，不要加功能）

1. 把已有整改提交到 main。不要在提交时把工作区说成已经冻结。
2. 补 FE-01b：`PermissionList.tsx:227-236` 与 `permission/index.ts:41` 复用 `fetchAllPages`。这是冻结前唯一还没写的产品改动。
3. DB-01：要么把注释改成「bootstrap 只保证模型读写的列，不是 000001 的全形」，并写明未跑迁移的全新启动不是支持路径；要么把缺的列补进 DDL。不要改 000001/000002/000024。不要恢复 AutoMigrate。
4. 可选、不阻断：给 `system_modules_route_groups_test.go` 补每组布尔值断言。现在删 `WithCapability()` 测不出来。
5. 维护者动作，本审计不代做：冻结宣告、GOV-08 一句话、`000031` 是否在生产执行。GOV-05 保持现状即可。

不做：新功能、大重构、批量删文档、删 `pkg/security/ssrf`、删空的 `retiredBusinessModules`、把 P3 立项、改 ops、打 release、改 CI 阈值。

## 10. 维度收口

| 维度 | 结论 |
|---|---|
| 1 架构与边界 | 装配与合同一致。ARCH-01 只在工作区闭合，链等价。无新的分层违规 |
| 2 后端质量与冗余 | 无确认的死共享抽象。保留项有 ops 调用方。未做全调用图的一律 E |
| 3 前端 | HEAD 上 FE-01/FE-02 仍在。工作区修了原引用点，漏了权限角色下拉。其余 P2/P3 不阻断 |
| 4 安全 / 租户 | 抽样无 P0。控制面（缓存、轮换、登出、限流、header 门、上传前缀）在。全路由普查未做 |
| 5 数据库 / 性能 | 迁移 000001–000031 连续。菜单模型缺 `tenant_id` 是已确认漂移，运行时影响未验证。三处提交后副作用是静态缺口。无性能裁决 |
| 6 治理 / CI / Harness | 工作区补了空转检查器。GOV-05/08 是维护者策略。STATUS.md 过期，未改 |
| 7 文档 | 1166 md 清点。一份 Active 租户设计与合同冲突。测试文档数字过期。不删历史 |
| 8 ops 兼容 | 未提交 diff 不打断 lock 钉住的快照。ops 未消费新 `System*Group` |

## 11. 未检查与风险

未启动服务，未跑 smoke / 压测 / 连库 / Sonar / 渗透，未跑全仓 `go test -short`、vitest、eslint、prettier、`fetchAllPages` 单测，未在 GitHub 上核对 ops weekly workflow，未做全路由认证普查，未读完 seed/authz SQL。安全与数据库结论是静态抽样，不能当成运行时证明。最大遗漏风险是 FE-01b：若只提交现有工作区就签冻结，权限角色下拉仍会在 >100 角色时静默缺项。
