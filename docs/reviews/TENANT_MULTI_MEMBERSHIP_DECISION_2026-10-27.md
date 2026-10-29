# 跨租户成员关系语义决策（2026-10-27）

**决策日期**：2026-10-27
**决策对象**：TC-01（P1）——跨租户 membership 授予登录与切换能力、但不授予任何授权（角色、菜单全空，受保护 API 全部 403）
**任务包**：`.harness/tasks/2026-10-27-tenant-multi-membership-decision/manifest.json`（W3，父程序 [`2026-10-27-tenant-completeness-remediation`](../harness/tasks/2026-10-27-tenant-completeness-remediation.task.md)）
**判据**：[租户合同 V1](../contracts/TENANT_CONTRACT_V1.md) §2.2 不变量 1（一个 user 可属于多个 tenant）与命名空间分离条款；[TENANT_API.md](../api/TENANT_API.md)；[租户功能完整性审查](TENANT_FUNCTIONAL_COMPLETENESS_REVIEW_2026-10-27.md)
**性质**：决策文档。本文件不改任何源码、不改合同文本（由 W4/W5 落地）、不清理任何存量行。

---

## 0. 结论（签名）

**选择 A：收窄 membership 语义**——跨租户成员关系不再出现在任何登录/切换入口，AddMember 收窄到目标租户自有用户。

- **B（跨租户角色绑定）登记为 V2 工作流**，本轮不立项不实现；**C（移除多租户成员关系）否决**（与合同 §2.2 不变量 1 直接冲突，且移除已交付能力）。
- **决策方式**：维护者授权委派（delegated decision），2026-10-27 由执行代理依授权记录，**对维护者审查开放**。若维护者改选 B/C，W4 包按其包规则必须**先以 abandoned 关闭再按新决策重写**，不得直接实现。
- **落地包**：
  - W4 `2026-10-27-tenant-multi-membership-authorization`（实现 A 的全部代码与回归，包清单见 `.harness/tasks/2026-10-27-tenant-multi-membership-authorization/manifest.json`）；
  - W5 `2026-10-27-tenant-contract-docs-alignment`（合同 §2.2 澄清 + API 文档 + 前端错误映射对齐，包清单见 `.harness/tasks/2026-10-27-tenant-contract-docs-alignment/manifest.json`）。
  - B 的 V2 包**不在本波创建**（W3 范围外），登记于 §3 的建议范围，立项时在 P2 规模化路线图下新开任务包。
- 本决策**不授权**自动清理任何存量跨租户 membership 行；处置见 §4。

## 1. 问题复述（读点已逐条核实，2026-10-27，W2 合并后）

授权是租户私有的，而成员关系不是：

| 读/写点 | 位置 | 行为 |
|---|---|---|
| 角色绑定写入 | `user_helper.go:83` `replaceUserRoles` | `system_user_role.tenant_id` 取自用户的**归属租户**（`lockUserOwningTenantIDTx`）；`:88` 的删除语句 `DELETE ... WHERE user_id = ?` **无租户谓词** |
| 角色写入校验 | `role_helper.go:301` `ensureUsersExist` | 按目标租户作用域校验 |
| 登录后角色解析 | `login_runtime.go:328` `GetUserRolesForTenant` | 按 claim 的租户过滤绑定 |

于是在租户 B 加入的用户（归属租户为 A）：登录选择器与 TenantSwitcher 都会提供 B，但解析出的角色集、菜单树为空，所有受保护 API 返回 403，且没有任何 UI 或 API 能在 B 里为其指派角色。

关键入口事实（核实于 `pkg/tenant/authz.go:159-180`）：`DiscoverDefaultTenant` 在存在**多于一个** active membership 时直接 `ErrTenantForbidden` 拒绝——多成员主体今天**只能通过登录选择器**进入第二租户。三个候选读点均无角色过滤：`authz.go:313` `ListActiveMembershipsForUser`、`login_tenant_switch.go:39` `ListSwitchableTenants`，以及发现路径本身。

既有 e2e（`frontend/tests/smoke-core/tenant-hostile-browser-matrix.spec.ts:82-92`）恰好构造了这一场景，但只断言选择 B 后应用壳渲染（`:138-140`），因此缺陷未被捕获。

W2（已合并）为 AddMember 补了**存在性**校验（`memberUserExists`），但其注释明确把「用户归属哪个租户」保留给本轮跨租户决策 wave——归属收窄**尚未实现**。

## 2. 三个选项

### 选项 A：收窄 membership 语义（✅ 选定，V1）

**改什么**：

1. `AddMember` 拒绝 `system_user.tenant_id ≠ tenantID` 的用户，注册错误键（如 `tenant.member.user_not_in_tenant`）；保留 W2 的存在性校验与 last-owner 防护。
2. 三个登录读点增加**角色存在谓词**：membership 仅在该 `(user, tenant)` 存在 enabled 的 `system_user_role` 行时才可解析/被提供——`DiscoverDefaultTenant`、登录候选列表、`/auth/tenants/switchable` 共用一个谓词（同在 `pkg/tenant`，一处实现）。
3. `SwitchSessionTenant` 在 `loadRolesForTenant` 为空时拒绝切换。
4. 前端移除 AppLayout 的 TenantSwitcher 与登录表单的租户选择器；两个端点保留为**过滤后的兼容面**（至多返回一项）。
5. 合同 §2.2 澄清（非改写）：「归属租户之外的 membership 不授予任何授权、不在登录时提供」；owner/admin/member 与 IAM 角色命名空间分离、不做隐式映射的原条款**不动**。

**设计约束**（继承自包实现注记）：不过滤 `ListActiveMemberships`（`pkg/tenant/membership.go:70`）——`/auth/current` 用它如实报告调用者自己的 membership 角色，不可登录的成员关系仍须按其本相报告，不得隐藏为不存在。

**平台视角不受影响**：TenantSwitcher 对平台能力主体本就自隐藏（由 `TenantViewpointSwitcher` 经 `X-Tenant-Id` 服务，读的是租户表而非 membership 表）。

**修掉的失败案例**：TC-01 整体——选择器/切换器不再提供不可用上下文；自动解析不再落在无角色租户。

**可能新引入的失败案例**：
- 被跨租户加入的用户登录后只见归属租户（其第二租户从入口消失）——这正是语义修正的本意，但**对既有跨租户成员是可感知变化**，须在 W5 的 API 文档与发布说明中写明；
- 「一个用户在两个租户工作」的产品诉求在 V1 内无法满足（须走 B）；
- 兼容端点的消费方看到的列表变短。

**成本**：W4 一个 wave（后端谓词 + AddMember 收窄 + 前端移除两个控件 + e2e 重写），无 schema/迁移改动；W5 文档对齐。**最小。**

### 选项 B：跨租户角色绑定（V2 工作流，本轮不立项）

**改什么**：`AddMember` 接受 `role_ids` 并向 `system_user_role` 写入目标租户的绑定；`ensureUsersExist` 接受「目标租户的 live 成员」；合同重新定义「用户归属租户」与「绑定租户」两个概念。

**前置必修**：`replaceUserRoles` 的删除语句（`user_helper.go:88`）按 `user_id` 无租户谓词删除——B 落地后，在租户 A 编辑一个用户会**抹掉其在所有其他租户的绑定**。此修复是 B 的硬前置。

**修掉的失败案例**：跨租户 membership 完全可用。

**可能新引入的失败案例**：
- 所有权模型歧义：绑定在 B、用户归属在 A——谁管理这些绑定？删除用户时按谁的规则清理？
- 跨租户提权面扩大，权限策略审查（qualityProfile: permission-policy）必须升级；
- 「每租户至少一个 owner」等不变量须按新模型逐条重推导；
- 存量行需要迁移与去歧义。

**成本**：**大**——写路径重设计 + 合同 §2.2/§2.3 重写 + 多个 wave。作为 V2「多租户深化」工作流立项，届时新建任务包。

### 选项 C：移除多租户成员关系（❌ 否决）

**改什么**：下线登录选择器与 TenantSwitcher，membership 收敛为单租户，修订合同 §2.2 不变量 1，存量多成员行的数据迁移另立包。

**修掉的失败案例**：TC-01 按构造消失。

**可能新引入的失败案例**：移除**已交付**的用户能力（登录选择器与 TenantSwitcher 均为 §2.2 不变量 1 而建）；冻结合同的不变量需要修订而非澄清；存量行强制迁移。

**成本**：中（实现 + 合同修订 + 迁移），但代价是产品能力与合同稳定性的双重损失。否决。

## 3. 对比

| 维度 | A 收窄语义 ✅ | B 跨租户绑定 | C 移除多成员 |
|---|---|---|---|
| 合同改动 | 澄清（不改不变量） | 重写 §2.2 所有权模型 | 修订不变量 1 |
| schema/迁移 | 无 | 绑定语义迁移 | 存量行迁移 |
| 修复 TC-01 | 是 | 是 | 是 |
| 新增风险 | 兼容面行为变化（文档化） | 提权面 + 所有权歧义 | 能力回退 |
| 成本 | 一个 wave + 文档 | 多 wave，V2 | 中，且损失能力 |

**推荐**：A for V1，B as V2 workstream（立项时新建包，范围以 §2 选项 B 的前置与风险清单为起点）。

## 4. 存量行处置

- **保留所有存量跨租户 membership 行，不做自动清理**；W4 的角色存在谓词会把它们从登录发现、候选列表与切换端点中隐藏。
- `/auth/current` 继续如实报告这些行（不过滤 `ListActiveMemberships`）。
- 数据清理 runbook 如有需要为独立包，本决策不预授权。

## 5. 明确不做

- 任何「membership 角色 → IAM 角色」的隐式映射（合同 §2.2 命名空间分离条款禁止）。
- 在 W3 内改源码、改合同文本或删除存量行（本包 doNotTouch：`backend/**`、`frontend/src/**`、合同、API 文档）。
- 为 B/C 预创建空任务包。
