# 决策记录：多租户的冻结线（2026-10-15）

**状态**：已批准（2026-10-21 企业级验收审查确认冻结线有效；已知缺陷 A/B/C 及审查新发现 P0/P1 整改记录见 [`docs/reviews/PANTHEON_BASE_ISSUES_AND_REMEDIATION_2026-10-21.md`](./reviews/PANTHEON_BASE_ISSUES_AND_REMEDIATION_2026-10-21.md)）
**议题**：多租户（SaaS 化）能力的去留与边界——推倒、扩张、还是冻结
**触发**：架构评审提问「多租户是为将来 SaaS 准备的，早期不设计后期改造成本很高；但不能为不确定的未来做大的改造。多租户若已开发完成，也不能放弃，否则之前的租户改造就是无用功。所以建议是什么？」

关联文档：

- 合同：[`docs/contracts/TENANT_CONTRACT_V1.md`](./contracts/TENANT_CONTRACT_V1.md)
- 资源归属矩阵：[`docs/designs/TENANT_RESOURCE_SCOPE_MATRIX.md`](./designs/TENANT_RESOURCE_SCOPE_MATRIX.md)
- 登录/初始化评估：[`docs/TENANT_LOGIN_INIT_EVALUATION_2026-10-15.md`](./TENANT_LOGIN_INIT_EVALUATION_2026-10-15.md)
- 全流程审查：[`docs/TENANT_FULL_FLOW_REVIEW_2026-10-14.md`](./TENANT_FULL_FLOW_REVIEW_2026-10-14.md)

---

## 1. 决策

**不推倒、不扩张、只收尾。** 把多租户冻结在**当前这条线**上：

> 共享库 + 共享 schema + 行级 `tenant_id` 隔离 + Casbin domain 授权 + 部署级 `compat | multi` 开关

并同时做三件事：

1. **收尾**：修掉 3 个已知缺陷，让现有能力真正可用（天内）；
2. **冻结**：裁决合同里唯一一处「现在决定很便宜、以后很贵」的空白（角色/菜单/权限的租户归属）；
3. **挂触发条件**：明确列出"不做"的清单，并为每一项写明**什么时候才做**。

---

## 2. 关键事实（实测，非推断）

### 2.1 多租户在单租户（compat）模式下几乎零成本

这是推翻"为不确定的未来做改造"这一顾虑的决定性事实。

| 环节 | compat 模式下的行为 | 证据 |
| :--- | :--- | :--- |
| 模式读取 | **TTL 缓存**，请求路径不产生额外查询 | `backend/pkg/tenant/tenant.go:200-243` `ModeLoader.Load()`，注释原文："Loaded at most once per TTL to keep the request path free of extra queries" |
| 上下文解析 | **立即返回**，不查 membership | `backend/pkg/tenant/tenant.go:124-125`：`Mode != ModeMulti` → 直接返回 `{tenant_id:0, mode:compat}` |
| 中间件 | 用缓存 loader，不直查库 | `backend/internal/middleware/tenant_context_middleware.go:63-82` `modeLoader.Load()` |
| 签发门 | **立即返回 0** | `backend/pkg/tenant/authz.go:153-157` |
| 业务查询 | 全部 `tenant_id = 0`，与改造前一致 | 合同 §2.3：`tenant_id = 0` 保留给平台级/全局资源 |

> 唯一的例外是登录路径用 `FeatureFlagSettingKeyReader`（`authz.go:305-335`）做**一次**原始查询——不在热路径上，可接受。

**结论**：`compat` 模式下的多租户代码 = 一次 RWMutex 读 + 一次字符串比较。**它不产生运行成本，只是"存在"。** 因此"为了不确定的未来做了大改造"这个前提**不成立**——改造已经完成，而且它是**惰性**的。

### 2.2 同一套代码 = 单租户产品 + 多租户产品

合同 §6 明确承诺：

> 现有单租户部署 **零迁移** 继续运行：`tenant_id = 0` 语义，解析优先级直接落 §3.1 第 3 条
> **`multi` 是 `compat` 的超集**：同一部署内非租户（平台）主体与租户主体并存，`platform.tenant_mode` 表达的是"是否启用租户维度"，而不是"整站二选一的世界开关"

所以「单租户」与「多租户」**不是两条路，而是同一条路的两档**，由一个 feature flag 切换。

**推论（此前未被显式认识）**：`compat` 模式本身就是「单租户私有化交付」的方案——同一个制品，开关一关就是单租户部署。也就是说**私有化交付能力已经具备**，只是没有以这个名义被识别。

### 2.3 现有能力已完成且已有 CI 兜底

- 主链路 E2E **62/62 通过**；`ProvisionTenant` 6 步单事务且幂等（角色 / 管理员 / 绑定 / owner membership / Casbin 域策略 / 设置副本）。
- 已有租户专项 CI：`.github/workflows/smoke-core.yml` 的 `smoke-tenant` job（multi 模式下 provision 租户矩阵 + 跑租户 smoke 套件）。
- 已有专项测试：`data_scope_failclosed_test.go`（fail-closed）、`tenant_hostile_smoke_test.go`（对抗性）、`tenant_platform_scope_test.go`、`casbin_tenant_domain_test.go`。

**这不是一个半成品负债，而是一个已完成且有回归网的能力。**

### 2.4 待收尾的缺陷（详见评估文档）

| 缺陷 | 严重度 | 成本 |
| :--- | :--- | :--- |
| A：登录页选择器不可用（`tenantId` 双层语义冲突 → 401） | P0 | 1 个函数 |
| B：`/auth/tenants/current` 的 `role`/`isActive` 与 `switchable` 矛盾 | 低 | 小时级 |
| C：租户硬删不清理 Casbin 策略（无外键、无清理路径） | 运维 | runbook |

---

## 3. 为什么不建议放弃

**放弃 ≠ 回到原点。** 放弃意味着主动**移除**：

- `tenant_id` 列及其在全部业务表上的传播
- `tenant.WithTenantScope(ctx)` 行级过滤
- Casbin domain 维度（`role:<key>@tenant:<id>`）
- 上下文解析优先级链（`X-Tenant-Id` → claim → 平台兜底 → compat 兜底）
- 登录签发门

而这些恰好是**改造代价曲线最陡**的部分：

- `tenant_id` 是**横切所有业务查询**的——表越多，补列 + 补查询的成本越高；
- Casbin domain 是**授权模型的地基**——动它等于重做授权；
- 上下文解析是**所有请求的入口**。

删掉它们，等于把"未来做 SaaS 的成本"从**增量**推回**重写**。

> 沉没成本本身不是继续的理由。此处成立的判断是：**继续的边际成本 ≈ 3 件小事，而已经买到的能力恰好是最难回溯的那部分。**

---

## 4. 真正的风险不是"多租户"，是"纪律"

多租户的真实成本是**认知负担**：每个新表要在 scope 矩阵登记，每个新查询要带 tenant scope（合同 §3.3 是硬不变量）。这个负担**已经付过了**，但**未来新增业务表时"忘记加"的风险**是真实的。

**应对：把纪律变成门禁，而不是靠人记。** 这是本决策中投入产出比最高的动作：

- 扩展 `smoke-tenant` CI：新增业务表未在 scope 矩阵登记 → **CI 失败**；
- 扩展 `data_scope_failclosed_test.go` 的思路：带 `tenant_id` 的表若无 scope 过滤路径 → **CI 失败**；
- 合同 §2.3 已规定"新增表必须在登记表注册后才允许合入（见验收矩阵 T4）"——把它**自动化**，而不是停留在文档约定。

---

## 5. 明确"不做"清单（写进文档，避免重复讨论）

| 项 | 现状 | 处理 |
| :--- | :--- | :--- |
| 租户自助注册 / 邀请码入驻 | 无 | **不做**——需要独立安全设计（限流/人机校验/审批流/`status=pending`） |
| 计费 / `plan` 消费 | `plan` 字段注释为"预留：V1 不消费" | **不做**——保持不消费 |
| 租户自定义菜单 / 角色 | 菜单/权限全局、角色按租户复制 | **不做**——见 §6 冻结 |
| 分库 / 按租户路由 | 无 | **不做**——见 §7 触发条件 |
| 登录页决定单/多租户模式 | `compat` 已是部署级开关 | **不做**——违反合同 §6（模式是部署级、单向） |
| 未认证的登录页创建租户 | 无 | **不做**——违反合同 §7（tenants CRUD 归 `system/org`） |

---

## 6. 唯一必须现在拍板的空白：角色 / 菜单 / 权限的租户归属

scope 矩阵把这三张表登记为**「待合同冻结」**，而实现**已经自选了混合方案**：

| 表 | 矩阵登记 | 实现现状 |
| :--- | :--- | :--- |
| `system_role` | 待合同冻结 | **按租户复制**（`provisionTenantRoles` 写 `tenant_id`） |
| `system_menu`（102 行） | 同上 | **全局**（无 `tenant_id` 列） |
| `system_permission` | 同上 | **全局**（无 `tenant_id` 列） |

这是三个选项里**最缺防御性的组合**：角色复制 → 同一 `role_key` 有 N 份，改一次平台策略要改 N 次，必然漂移；菜单/权限全局 → 租户不可定制；两者混用 → 心智模型不一致。

**建议冻结为「全局共享」**：三张表全部 `tenant_id = 0`，租户差异只靠 Casbin domain 表达。
理由：与 `system_menu` / `system_permission` 的**现状一致**，只需把 `system_role` 改回全局（改动极小）；且与 `tenant_admin` 已有的设计取向一致——它刻意拿到与平台 `admin` 相同的 `/api/v1/*` 方法面，靠域 + scope 收窄：

> `tenant_admin` deliberately receives the same `/api/v1/*` breadth as the platform `admin` role: the chosen product model is "the full console, automatically confined to the caller's tenant".

**这个决定现在拍板 = 改一行或零行；等有 20 个业务模块后再拍 = 改 20 个模块。**

### 6.1 裁决（2026-10-15，维护者）

> **角色按租户复制（定义全局统一）；菜单 / 权限保持全局共享。**

| 表 | 冻结归属 | 唯一键 | 实现落点 |
| :--- | :--- | :--- | :--- |
| `system_role` | **`tenant-replicated`**（每个租户持有自己的副本） | `(tenant_id, role_key)` | 迁移 000024 `uk_system_role_tenant_key`；`ProvisionTenant` 按 `tenantRoleDefinitions` 生成 |
| `system_menu` | **`platform-global`**（无 `tenant_id` 列） | — | 102 行菜单全局共享 |
| `system_permission` | **`platform-global`**（无 `tenant_id` 列） | — | 权限键全局，租户差异只由 Casbin domain 表达（§4） |

**为什么“角色复制”不等于“必然漂移”**（这是 §6 原先担心的核心）：

- 角色**定义**不是逐租户人工维护的：它是单一来源的常量（`pkg/tenant` 的 `TenantRoleAdminKey` /
  `TenantRoleMemberKey` + `modules/system/iam/tenant` 的 `tenantRoleDefinitions`），由 `ProvisionTenant`
  在建租户时**代码生成**，而不是由运营人员在每个租户里手抄一遍。
- 因此“改一次平台策略要改 N 次”这个前提不成立——改的是那一份定义，新租户与重跑的修复入口
  （`InitializeTenantDefaults` 也委派 `ProvisionTenant`，幂等）会自然收敛。
- 漂移风险的真正来源是**手工在单个租户里改角色**，这属于正常业务操作（租户自定义角色），而不是治理缺陷。

**本裁决的范围与后果**：

- 与当前实现**完全一致**（本轮所有探针验证的正是这个组合），所以不需要代码改动，只需把登记表从
  “待合同冻结”变为“已冻结”。
- 重新开放该决策的触发条件：若未来需要**租户自定义角色**（租户自己新增角色、改权限面），需要引入
  “平台下发模板 + 租户覆写”的合并语义，那是一次独立的 contract-change。

---

## 7. 触发条件（不做 ≠ 永远不做）

| 触发条件 | 那时才做 | 为什么届时成本可控 |
| :--- | :--- | :--- |
| 出现第一个要求**数据物理隔离**的客户 | 按租户路由到独立库 | `tenant.Resolve` 已是**可插拔**的（`Resolve(in ResolveInput)` 接 `Mode` 参数）；这是**替换 scope helper 的实现**，不是重写业务 |
| 出现**自助注册**的真实需求 | 独立设计入驻流程 | 与现有认证链正交，不触碰既有上下文模型 |
| 租户数超出共享 schema 的舒适区 | 评估分片 | `tenant_id` 已贯穿，分片键现成 |
| 大客户要求**独立部署** | 直接用 `compat` 模式交付 | 合同 §6 已承诺零迁移；**现在就具备该能力** |

---

## 8. 立即行动项

| # | 动作 | 依赖 | 成本 |
| :--- | :--- | :--- | :--- |
| 1 | 修缺陷 A：`findCredentialByUsername` 改为「名租户优先 → 全局回退」，让签发门重新可达 | 无 | 小时级 |
| 2 | 修缺陷 B：`/auth/tenants/current` 的 `role`/`isActive` | 无 | 小时级 |
| 3 | 缺陷 C 写入运维 runbook：硬删租户必须清 `casbin_rule` 中 `v0 LIKE 'role:%@tenant:<id>'` | 无 | 分钟级 |
| 4 | **冻结 §6**：裁决角色/菜单/权限归属，建议「全局共享」 | **需决策** | 拍板 |
| 5 | 把 §4 的 scope 门禁自动化（新表未登记 / 无 scope 过滤 → CI 失败） | 无 | 天级 |
| 6 | 将 §5「不做」清单与 §7 触发条件写入合同或设计文档，作为后续评审基线 | 无 | 小时级 |

**行动项 1 的验收锚点**（可直接作为回归用例）：

- 双 membership 用户选中「仅成员」租户 → `200`，会话 claim 为该租户；
- 选中「非成员」租户 → `403 TENANT_FORBIDDEN`（不是 401）；
- 同名 `admin` 存在于两个租户 → 带 `tenantId` 登录命中对应租户那行；
- 单 / 零 membership 行为不变（E2E 62/62 不回退）。

---

## 9. 后果

**正面**

- 保留全部已建成的架构性能力（最难回溯的部分），不产生运行成本；
- 单租户与多租户共用一套制品，交付形态由 flag 决定，**同时具备私有化交付能力**；
- 决策边界清晰：未来任何"要不要加租户特性"的讨论，都有本记录作为基线，不必重新论证。

**负面 / 需接受**

- 多租户相关的**认知负担**长期存在（每张新表都要登记 + scope）；
- 若未来真要做分库路由或租户自定义角色，仍需一次**架构增量**（但成本远低于重写）；
- §6 的三张表冻结为「全局共享」后，若将来客户强烈要求自定义角色，需要重开该决策。

---

## 9.5 收尾进展（2026-10-15 晚，实测更新）

§8 的立即行动项已执行到「可交付但仍有未裁决项」的状态：

| # | 动作 | 结果 |
| :--- | :--- | :--- |
| 1 | 修缺陷 A（picker 可达签发门） | **已闭环**。实测：选中「仅成员」租户 → `200 success`（此前 `401 user.login.error.not_found`） |
| 2 | 修缺陷 B（`/auth/tenants/current` 的 `role`/`isActive`） | **已闭环**。`GetCurrentTenant` 现按调用者自身 membership 回填 role 并置 `isActive=true`；实测 `current` 与 `switchable` 对同一租户一致（`AGREE=True`）。只读，不授予新权限 |
| 3 | 缺陷 C 写入运维 runbook | **已闭环**。`docs/runbooks/TENANT_TROUBLESHOOTING.md` §2b：硬删前必须清 `casbin_rule` 中 `role:%@tenant:<id>`，含安全理由（无外键 + 无清理路径 + id 回收会继承旧授权）、18 张表全量清理语句与校验语句 |
| 4 | 冻结 §6（角色/菜单/权限归属） | **未裁决**——见下 |
| 5 | scope 门禁自动化 | **未做**（天级，未在本轮范围） |
| 6 | 「不做」清单入合同/设计文档 | 本条已由本决策记录承担，不再另开文档 |

**验收态（人工确认时的状态）**：`platform.tenant_mode=multi`，活跃 membership 0、夹具租户 0、租户域 Casbin 策略 0、平台策略 40；平台管理员 `admin` 登录直接进入平台控制台（`resolvedBy=platform-fallback`，**不弹选择器**）。此前实测库中残留的探针夹具与平台管理员身上多余的 membership 已清理——正是这些残留会把选择器推到操作者面前。

**实测证据（真栈：后端 8080 + MySQL 3306 + Redis 6379 + 真实 Chromium）**：租户全生命周期 E2E **62/62**；隔离 E2E **34/34**；核心冒烟 **26 passed / 3 skipped / 0 failed**；租户视角浏览器验收 **5/5**。证据目录：`.harness/evidence/2026-10-15-tenant-closeout/`。

**仍未闭**（需要维护者裁决或单独任务）：

1. **§6 归属裁决**（角色 / 菜单 / 权限租户内还是全局）。当前实现 = 角色按租户复制 + 菜单/权限全局，本轮全部探针验证的正是这个组合；本记录 §6 建议把角色改回全局。**只有维护者能拍板。**
2. **敌对抗性租户冒烟套件（hostile matrix）当前是红的，且是夹具陈旧而非产品缺陷**：夹具工具与三个 spec 以显式 `tenantId` 登录**平台** `admin`；在隔离模型下，带租户 claim 的会话其角色在租户内解析，平台管理员没有租户角色 → 角色集为空 → 每次调用 `403 permission.denied`。修复方向是改用「建租户 → 用该租户自己的管理员登录」（即通过验收的 viewpoint spec 的做法），但还需决定夹具写入的策略如何让运行中的 Casbin enforcer 感知（重新加载或经 API 预置）。CI 中该 job 为 `continue-on-error`，故长期未被门禁暴露。
3. **平台主体 + 租户 claim = 空角色集**：它指向一个产品问题——平台能力主体选择租户登录后是否应保留平台角色。

## 10. 附录：本轮实测证据

- E2E：`verify_tenant_full_flow.py` **62/62**。
- 登录解析三态：0 membership → `{resolvedBy:platform-fallback, tenantId:0}`；1 membership → `{resolvedBy:subject, tenantId:82}`；2 memberships → `tenantSelectionRequired` + 候选。
- 缺陷 A 复现：租户 82（归属）/ 83（仅成员），选 82 → `200`；选 83 → `401 user.login.error.not_found`。
- 缺陷 B 复现：同租户同会话，`current` 给 `role:""/isActive:false`，`switchable` 给 `role:"owner"/isActive:true`。
- 缺陷 C：硬删租户后残留 `role:*@tenant:*` 策略 48 条（已手工清除）；代码中无 `RemoveFilteredPolicy` / `PurgeTenant`。
- 清理复核：活跃租户仅 `0 __global__`(archived)；`casbin_rule` 回到 40 条平台策略，租户域策略 0 条。
- 本轮**未改动任何生产代码**。
