# 登录页承载租户初始化 / 选择 —— 可行性评估（2026-10-15）

**评估对象**：提案「把租户初始化搬到登录页；登录页加选择按钮；单租户走单租户模式；admin 启动时注入菜单/角色/权限；选择租户走租户初始化逻辑」

**评估方法**：代码走查（backend + frontend + 合同 / scope 矩阵）+ 真栈运行态探针（`multi` 模式，MySQL 真库，临时夹具建删）。

**结论摘要**：提案的 **3 项已经实现**、**2 项与已冻结合同冲突**、**1 项是安全边界变更**；另有 **3 个缺陷**，其中 1 个 P0 **直接阻断提案的核心用法**——登录页的选择器今天根本选不进去。

关联文档：

- 合同：[`docs/contracts/TENANT_CONTRACT_V1.md`](./contracts/TENANT_CONTRACT_V1.md)
- 资源归属矩阵：[`docs/designs/TENANT_RESOURCE_SCOPE_MATRIX.md`](./designs/TENANT_RESOURCE_SCOPE_MATRIX.md)
- 前轮全流程审查：[`docs/TENANT_FULL_FLOW_REVIEW_2026-10-14.md`](./TENANT_FULL_FLOW_REVIEW_2026-10-14.md)

---

## 0. 结论速览

| # | 提案项 | 现状 | 评估 |
| :--- | :--- | :--- | :--- |
| 1 | 登录页加租户选择按钮 | **已实现**（服务端驱动，>1 membership 时出现） | ✅ 已建，但**当前不可用**（见缺陷 A，P0） |
| 2 | admin 启动时注入菜单 / 角色 / 权限 | **已实现**（`ProvisionTenant`，6 步、单事务、幂等） | ✅ 已建；注意**菜单是全局的，不按租户注入** |
| 3 | 单租户走单租户模式 | **已实现**（`compat` 模式） | ⚠️ 与合同 §6 冲突：模式是**部署级单向**开关，不是登录页选项 |
| 4 | 选择租户走租户初始化逻辑 | 选择器只解析上下文，不初始化 | ⚠️ 语义二义（见 §2.5），需你裁决 |
| 5 | 租户初始化放登录页 | 无 | ❌ 安全边界变更（合同 §7），需独立设计 |

**一句话**：你想在登录页做的事，架构上已经建好了，而且**建得是对的**（签发门 `GateSessionIssuance` 校验 membership + 租户状态，错误码也符合合同）。它只是被上游一个**字段语义冲突**打死了。**先修这个字段，比重新设计登录页价值高一个数量级。**

---

## 一、现状实测（运行态，非设计意图）

### 1.1 三种主体的登录解析

`platform.tenant_mode = multi`（实测库值）。用临时夹具建两个租户 + 一个双 membership 用户，观察 `/auth/login`：

| 主体 | membership | 登录响应 | `/auth/tenants/current` |
| :--- | :--- | :--- | :--- |
| 平台 `admin` | 0 | `200`，无选择器 | `{mode:multi, resolvedBy:platform-fallback, tenantId:0}` |
| 租户管理员 `usr-v1` | 1 | `200`，无选择器 | `{resolvedBy:subject, tenantId:82}` |
| 同上 + 加入第二个租户 | 2 | `200` + `tenantSelectionRequired:true` + `tenantCandidates:[82,83]` | 需显式选择 |

E2E 全流程 `verify_tenant_full_flow.py`：**62/62 通过**。也就是说——**当前设计按它自己的规格是通的**，缺陷 A 落在规格覆盖不到的组合上。

### 1.2 缺陷 A（P0）：登录页选择器**不可用**

**证据**（新夹具，租户 82 / 83，用户 `usr-v1` 归属 82、成员于 83）：

```
picker 提供给用户的候选：
  tenantCandidates = [{tenantId:82, role:"owner"}, {tenantId:83, role:"member"}]

用户选中 82（凭据所在租户）→ code=200 success
用户选中 83（仅 membership） → code=401 user.login.error.not_found   ← 选择器就是为这个场景存在的
```

**根因**：`tenantId` 这一个字段，在**两层**被赋予了**两种不同语义**，且错误的那个先执行。

| 层 | 位置 | 把 `tenantId` 当作 | 结果 |
| :--- | :--- | :--- | :--- |
| ① 认证 | `login_service.go:140-147` `findCredentialByUsername` → `FindByUsernameInTenant(tenantID, username)` | **凭据作用域**：去「哪个租户的用户表」找这个人 | 在 83 找不到 `usr-v1` 的行 → `ErrRecordNotFound` → **401** |
| ② 上下文 | `login_runtime.go:415-441` `resolveLoginTenantClaim` | **会话上下文**：给会话盖哪个租户的 claim | 走 `GateSessionIssuance` 校验 membership → 正确返回 `ErrTenantForbidden` → **403** |

② 的注释明确写着这就是选择器路径：

> `explicit tenantChoice > 0 wins and is validated through the issuance gate (active membership + active tenant master) — this is the picker path for multi-membership users (slice 2)`

**但 ② 对选择器场景是不可达的**——① 已经先抛 401 了。所以：

- 第 ② 层那段**正确、且符合合同**的逻辑，实际上是一段**死代码**；
- 顺带违反合同 §5：`主体无该租户 membership` 应为 `403 TENANT_FORBIDDEN`，实测是 `401 user.login.error.not_found`；
- 而 membership 语义下**本来就不存在**「83 租户里的 `usr-v1`」这行凭据——`tenant_memberships` 只是关系表，不建用户行。所以「按名租户找凭据」对 picker 场景**在数据模型上就是错的**。

> 前端 `Login.tsx:250-289` 的两步式提交（先无 `tenantId` 登录拿候选 → 用户选择 → 带 `tenantId` 重提）逻辑本身没错，它只是忠实地把 `tenantId` 交给了后端。

### 1.3 缺陷 B：同一租户、同一会话，两个接口自相矛盾

同一用户 `usr-v1`、同一会话、同一租户 82：

| 接口 | `role` | `isActive` |
| :--- | :--- | :--- |
| `GET /auth/tenants/current` | `""` | `false` |
| `GET /auth/tenants/switchable` | `"owner"` | `true` |

`current` 的 `role`/`isActive` 恒为空/假——它会误导任何「按当前租户角色渲染」的前端逻辑。属低危但确定的不一致。

### 1.4 缺陷 C（运维）：租户硬删不清理 Casbin 策略

`casbin_rule` **没有** `tenants` 外键，且代码里**没有任何**租户策略清理路径（`RemoveFilteredPolicy` / `PurgeTenant` 均不存在）。本次实测中，硬删租户后残留 `role:*@tenant:*` 策略 **48 条**（租户 76–83），已手工清除。

后果：合同 §2.1 规定硬删是**运维动作**，那么运维 runbook 必须显式包含 Casbin 清理；否则 `GrantTenantRolePolicies` 的「幂等追加」特性会让**回收 id** 的租户继承陈旧授权。

---

## 二、逐条评估提案

### 2.1 「登录页选租户」——已实现，但被缺陷 A 打死

**不需要重新设计。** 现有形态（服务端驱动候选 + 签发门校验）恰好是正确形态：

- 候选来自 `tenant_memberships`（active），前端不猜；
- 选择结果经 `GateSessionIssuance` 双重校验（membership active + 租户 `status=active`），合同 §5 的三类拒绝（无 membership / suspended / archived）都已有对应错误码；
- 只在 `len(candidates) > 1` 时出现，前端同步要求 `>= 2`，两侧一致。

**所以这一项的优先级不是「做」，是「修」。**

**修复建议（Option A，推荐）**：认证层先按名租户精确命中，未命中再回退全局，把「是否是合法目标租户」完全交给第 ② 层：

```
findCredentialByUsername(username, tenantChoice):
  1. tenantChoice > 0 且 FindByUsernameInTenant(tenantChoice, username) 命中 → 返回
  2. 未命中 → FindByUsername(username) 全局命中 → 返回
  3. 均未命中 → not_found
  （全局命中多条 → 保持 deny，不猜）
```

- **保留**「我是 X 租户的 `admin`」语义：多租户下同名 `admin` 仍能按名租户精确命中（合同 §2.2 允许用户名跨租户重复）；
- **修好** picker：membership 用户能在全局命中自己的凭据，随后由签发门给出正确的 403 / 200；
- 错误类自动回归合同 §5；
- 同一修法需覆盖 MFA 路径（`login_runtime.go:364` 同样用 `tenantChoice` 重解析）。

**备选**：Option B 拆成 `credentialTenantId` + `sessionTenantId` 两个显式字段——语义最干净，但要改 DTO / 前端 / 合同，且 picker 场景前端**本来就不掌握** credentialTenantId，收益不抵成本。Option C「认证层完全忽略 tenantId」不可取：两租户各有一个 `admin` 时会全局歧义，可能拿 A 租户的密码去验 B 租户。

### 2.2 「单租户走单租户模式」——与合同 §6 冲突

单租户模式**已经存在**，就是 `compat`（`platform.tenant_mode = compat | multi`）。但合同把它定为**部署级、单向**：

> 开关：`system/config` 的 feature flag（`tenant.mode = compat | multi`，默认 `compat`），运行时可读、变更走 config 变更流程
> **`multi` 是 `compat` 的超集**：因此**同一部署内非租户（平台）主体与租户主体并存**，`platform.tenant_mode` 表达的是"是否启用租户维度"，而不是"整站二选一的世界开关"
> 模式切换只允许 compat → multi 单向（per 部署）

所以「登录页上选择按钮决定单 / 多租户」= 把**部署级开关降级为会话级选项**，直接违反 §6；而「整站二选一」正是 §6 明确否掉的那个模型。

**如果你的真实意图是**「compat 部署不显示选择器、multi 部署才显示」——**这已经是现状**：compat 下无 membership → 无候选 → 无选择器。

**如果确实要改成会话级**：这是 contract-change（改版本号 + 变更记录 + runbook / matrix 同步），且要回答「同一部署内两套模式并存时，`tenant_id=0` 的平台数据由谁可见」。

### 2.3 「租户初始化放登录页」——安全边界变更，不建议

- 合同 §7 明确：`tenants` 主数据 CRUD 归 `system/org`，**禁止** auth 直写；`POST /api/v1/system/tenant` 今天要求平台级 Casbin 能力 + CSRF。
- 放到登录页 = 新增一个**未认证的 tenants 写入面**。
- 且与现状不一致：`DELETE /api/v1/system/tenant/:id` 实测都要求 **step-up 验证**（`403 auth.operation.verification_required`），而「创建租户」却要搬到未认证页面？
- 若你的意图是**自助注册 / 邀请码入驻**，那是一个独立产品特性（新端点 + 限流 + 人机校验 + 审批流 + `status=pending` + 滥用控制），不是「把 UI 挪个位置」。

**另一种读法**（更可能）：你指的是**首次部署引导**——第一个平台管理员如何产生。那属于安装 / 初始化步骤（种子数据），与登录页无关。

### 2.4 「admin 注入菜单 / 角色 / 权限」——已实现

`ProvisionTenant`（`tenant_provision.go:98`）在一个事务里做 6 件事，**且幂等、可重跑当修复入口**：

1. 租户域角色（`tenant_admin` / `tenant_member`）
2. 租户管理员用户（`system_user.tenant_id = <tenant>`）
3. 管理员角色绑定（租户命名空间）
4. `owner` membership（**登录签发门要求的那条**）
5. Casbin 策略 `role:<key>@tenant:<id>`
6. 平台设置副本（实测 41 条）

实测落库：角色 2、用户 1、绑定 1、membership 1、设置 41。

**一处需要澄清**：**菜单不按租户注入**——`system_menu`（102 行）与 `system_permission` 都**没有** `tenant_id` 列，是全局的。租户管理员之所以看到「全量控制台」，是因为 `tenantRoleDefinitions` 刻意给了 `tenant_admin` 与平台 `admin` 相同的 `/api/v1/*` 方法面，靠 **Casbin 域 + tenant scope** 收窄到本租户：

> `tenant_admin` deliberately receives the same `/api/v1/*` breadth as the platform `admin` role: the chosen product model is "the full console, automatically confined to the caller's tenant".

所以「给租户注入菜单」在现有模型里**不是缺失，而是设计选择**——但见 §三。

### 2.5 「选择租户走租户初始化逻辑」——二义，需裁决

| 读法 | 含义 | 评估 |
| :--- | :--- | :--- |
| (a) 选择**已有**租户 → 进入该租户上下文 | 就是选择器 | 已实现，被缺陷 A 阻断；修法见 §2.1 |
| (b) 选择 / 输入租户 → **创建并初始化**该租户 | 自助注册 | 安全边界变更，见 §2.3 |

---

## 三、动登录页之前必须先裁决的合同空白

scope 矩阵里这三行是**未冻结**状态，而实现**已经自行选定了方案**：

| 表 | 矩阵登记 | 实现现状 |
| :--- | :--- | :--- |
| `system_role` | 待合同冻结（「平台治理对象是否租户内复制由合同决定」） | **按租户复制**（`provisionTenantRoles` 写 `tenant_id`） |
| `system_menu` | 同上 | **全局**（无 `tenant_id`） |
| `system_permission` | 同上 | **全局**（无 `tenant_id`） |

即实现选了**混合方案**：角色复制、菜单/权限全局、Casbin 按租户域。这是三个选项里**最缺防御性的组合**：

- 角色按租户复制 → 同一 `role_key` 有 N 份，改一次平台策略要改 N 次，必然漂移；
- 菜单 / 权限全局 → 租户无法定制自己的控制台；
- 两者混用 → 「租户内角色」与「平台治理对象」的心智模型不一致，后续每次加表都要重新吵一遍。

**三个选项**：

1. **全局共享**：角色 / 菜单 / 权限全部 `tenant_id=0`，租户差异只靠 Casbin 域。最少复制、最易治理；代价是租户不能自定义角色。
2. **按租户复制**：三张表都带 `tenant_id`，租户可自定义。代价是漂移 + 治理成本 + 需要「平台下发 → 租户覆写」的继承机制。
3. **平台下发 + 租户覆写**（中间态）：平台行是模板，租户行只存差异。最灵活，但需要覆写语义与合并规则，实现最重。

**建议**：先冻结为 **1**（与当前 `system_menu` / `system_permission` 现状一致，只需把 `system_role` 改回全局），把「租户自定义角色」推后；否则每次租户相关变更都要连带回答一遍这个问题。

---

## 四、建议路径

| 阶段 | 动作 | 依赖 |
| :--- | :--- | :--- |
| **S1（立即，独立于任何提案）** | 修缺陷 A：`findCredentialByUsername` 改为「名租户优先 → 全局回退」，让签发门重新可达 | 无 |
| **S1'** | 顺带修缺陷 B（`current` 的 `role`/`isActive`） | 无 |
| **S1''** | 缺陷 C 写入运维 runbook：硬删租户必须清 Casbin | 无 |
| **S2** | 冻结 §三 的角色 / 菜单 / 权限归属（建议选项 1） | 需你裁决 |
| **S3** | 把缺陷 A 的修法补进合同 §3.1 / §5 的登录解析说明（`tenantId` 双层语义显式化） | 需你裁决 |
| **S4** | 若确要会话级模式或自助注册 → 走 contract-change | 需你裁决 |

**S1 的验收锚点**（可直接作为回归用例）：

- 双 membership 用户，选中「仅成员」租户 → `200`，会话 claim 为该租户；
- 选中「非成员」租户 → `403 TENANT_FORBIDDEN`（不是 401）；
- 同名 `admin` 存在于两个租户 → 带 `tenantId` 登录命中对应租户那行；
- 单 membership / 零 membership 行为不变（E2E 62/62 不回退）。

---

## 五、需要你裁决的问题

1. **「租户初始化放登录页」** 指的是 (a) 已有租户的选择，还是 (b) 自助创建租户？如果是 (b)，是否有入驻审批 / 邀请码的产品前提？
2. **模式语义**：接受合同 §6 的部署级单向语义（= 现状，登录页无需模式开关），还是正式提 contract-change 改成会话级？
3. **角色 / 菜单 / 权限归属**：全局共享（建议）、按租户复制、还是平台下发 + 租户覆写？

---

## 附：本轮验证与清理

**探针**（`.tmp/`，临时夹具，已删除）

- `probe_login_resolution.py` —— 三种主体登录解析（0 / 1 / 2 memberships）
- `verify_picker_defect.py` —— 缺陷 A / B 的可复现夹具

> 注：服务端 cookie 带 `Secure`，浏览器把 `http://127.0.0.1` 视为可信源会照常发送，而 Python `http.cookiejar` 不实现该例外。探针改为手工维护 Cookie 头。

**清理**（本次共创建临时租户 80–83，全部硬删 + 清除 Casbin 残留）

- 删除租户 80–83 及其 `system_role` / `system_user` / `system_user_role` / `tenant_memberships` / `system_setting` 行；
- 删除孤儿 Casbin 策略 48 条（租户域 76–83）；
- 复核：活跃租户仅剩预留行 `0 __global__`（`archived`）；`casbin_rule` 回到 40 条平台策略；租户域策略 0 条。

**未改动任何生产代码**——本轮为评估，缺陷 A/B/C 均已定位并给出修法，待你裁决后再实施。

---

# 收尾结果（2026-10-15 闭环）

用户要求「把租户功能闭环，完成 E2E 与冒烟测试，不要人工确认时仍是不可用状态」。以下是闭环记录。

## 1. 缺陷 A（P0）：已修复

**改动**：`backend/modules/auth/login/login_service.go` `findCredentialByUsername` 改为**两段式解析**：

1. 先按名租户精确查（保留「我是 X 租户的 admin」语义 —— 合同 §2.2 下用户名只在租户内唯一）；
2. 名租户查无此行时**回退全局查询**（picker 场景：membership 用户选中非归属租户）；
3. 目标租户是否合法**完全交给签发门** `resolveLoginTenantClaim` → `GateSessionIssuance`。

回退**不会**在名租户内选到另一个主体（该租户内根本没有匹配行），只在该租户「沉默」时放宽搜索范围；密码比对与签发门仍然生效。同步更新 `auth_repository.go` 中 `FindByUsername` 的文档注释。MFA 路径无需改动——它只走上下文层（`resolveLoginTenantClaim`），本就没有凭据查找。

**新增回归单测**（`login_tenant_gate_test.go`，5 个）：

| 测试 | 断言 |
| :--- | :--- |
| `TestLoginNamedTenantFallsBackToGlobalCredential` | picker 场景认证成功，且 claim 落在**所选**租户（不是归属租户） |
| `TestLoginNamedTenantFallbackStillVerifiesPassword` | 回退不放宽凭据校验（错密码 → `user.login.error.password_wrong`） |
| `TestLoginNamedTenantScopedCredentialWins` | 名租户自有同名凭据优先，归属租户的行不得抢占 |
| `TestLoginNamedTenantWithoutMembershipDeniedByGate` | 非成员租户 → 签发门 `ErrTenantForbidden` |
| `TestLoginNamedTenantIgnoredInCompatMode` | compat 模式忽略名租户（合同 §6 零变更） |

> 这 5 个测试补上了此前的覆盖盲区：原有 13–17 号测试只单独驱动 `resolveLoginTenantClaim`（传裸 user id），因此**看不见**认证层也消费了同一个字段——正是缺陷 A 得以长期潜伏的原因。

**真栈验证**（租户 232/233，用户归属 232、仅成员于 233）：

| 请求 | 修复前 | 修复后 |
| :--- | :--- | :--- |
| `tenantId=232`（归属租户） | `200` | `200` |
| `tenantId=233`（仅成员） | **`401 user.login.error.not_found`** | **`200 success`** |
| `tenantId=<非成员租户>` | `401 user.login.error.not_found` | **`403 tenant.forbidden`**（合同 §5） |

## 2. 缺陷 B：已修复

`/auth/tenants/current` 现在回填调用者在该租户的 membership 角色。真栈复核（同一用户、同一会话、同一租户 232）：

```
current   -> role='owner' isActive=True
switchable-> role='owner' isActive=True
AGREE? True
```

## 3. 缺陷 C：已写入运维 runbook

`docs/runbooks/TENANT_TROUBLESHOOTING.md` §2b「Hard-deleting a tenant (ops action): purge Casbin policies first」给出完整步骤与 SQL（含 `v0 LIKE CONCAT('role:%@tenant:', <id>)` 的核对与删除）。与合同 §2.1「硬删是运维动作、不暴露给业务 API」一致，因此不新增应用层 purge 接口。

## 4. 合同空白：已冻结（非本轮，由并行工作流完成）

`docs/designs/TENANT_RESOURCE_SCOPE_MATRIX.md` 于 2026-10-15 给出维护者裁决：

- `system_role` → `tenant-replicated`（角色按租户复制；定义单一来源为 `pkg/tenant` 常量与 `tenantRoleDefinitions`，由 `ProvisionTenant` 代码生成，故不产生漂移）
- `system_menu` / `system_permission` → `platform-global`（租户差异只由 Casbin domain 表达）

即维持实现现状并正式认可，与「全局共享」的备选方案相比同样可辩护（避免了复制体人工维护的漂移风险）。

## 5. 验证结果（全绿）

| 项目 | 结果 |
| :--- | :--- |
| 租户全流程 E2E（`verify_tenant_full_flow.py`） | **62/62** |
| 租户冒烟 `test:smoke:tenant` | **10 passed** |
| 租户视角冒烟 `test:smoke:tenant:viewpoint` | **5 passed** |
| 登录选择器冒烟 `auth-tenant-picker.spec.ts` | **3 passed** |
| Go 单测 `modules/auth/login` | ok |
| Go 单测 `modules/system/iam/user`、`pkg/tenant`、`modules/system/iam/tenant` | ok |
| `go build ./...` / `gofmt` | exit 0 / 干净 |
| 前端 `tsc -b` | exit 0 |
| `check:i18n-hardcode` / `check:i18n-generated-scope` / `check:menu-contract` | 全绿 |
| 架构边界 `check-boundaries.mjs` | 0 新增（1 条既有 platform→system 前端 finding，与本轮无关） |

### 冒烟过程中修掉的两个非产品缺陷

1. **本地后端脚本与 CI 环境不一致**：`.tmp/run-tenant-e2e-backend.sh` 设了
   `PANTHEON_ENABLE_DYNAMIC_MODULES=false`，导致 `GET /lowcode/dynamic-modules` 返回
   `403 module.dynamic.disabled`，冒烟用例 `module registry list is readable per tenant` 失败。
   CI 的 `smoke-tenant` job 不设该变量（非生产环境默认启用）。已改为 `true`。
2. **冒烟夹具自造歧义**：`tenant-protected-resources.spec.ts` 的 `createDictType` 把
   `dictName` 设为与 `dictCode` 相同，而字典表的「名称」列在无翻译时回退渲染原始值
   （`t(value, value)`），于是同一个字符串出现在**编码列（Button）**与**名称列（Text）**两个单元格里，
   `getByText(code, { exact: true })` 解析到 2 个元素 → strict mode violation（且与渲染时序竞争，表现为偶发）。
   已让夹具使用可区分的名称，跨租户隔离断言（`toHaveCount(0)`）不变。

## 6. 环境清理

删除测试夹具租户 101、202、204–235（含全部 `system_role` / `system_user` / `system_user_role` /
`tenant_memberships` / `system_setting` 行）与全部 `role:*@tenant:*` Casbin 策略。
保留用户自有行：`0 __global__`（archived）、`11 test`、`30 test-tenant`（均为 `deleted`）。

复核：租户域 Casbin 策略 **0** 条、`casbin_rule` 总数回到 **40**（纯平台策略）、
`platform.tenant_mode` 仍为 **`multi`**。

> 注：**未**执行 `cmd/tenantmatrixdb down` —— 该子命令会把 `platform.tenant_mode` 强制回 `compat`
> 并回滚迁移 16..13，会破坏当前 multi 运行态。夹具改用 SQL 定点清理。
