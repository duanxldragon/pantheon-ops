# 套餐能力阶梯语义决策（2026-10-28）

**决策日期**：2026-10-28
**决策对象**：TC-07（P3）——`tenants.plan` 已数据驱动（`tenant_plans` 表 + 控制台套餐编辑器），但前端能力矩阵把 code 联合收拢到四个内建档、其余 code 一律按 `free` 解析并隐藏 lowcode 门控面
**任务包**：`.harness/tasks/2026-10-27-tenant-plan-ladder-decision/manifest.json`（W7，父程序 [`2026-10-27-tenant-completeness-remediation`](../harness/tasks/2026-10-27-tenant-completeness-remediation.task.md)）
**判据**：[租户合同 V1](../contracts/TENANT_CONTRACT_V1.md) §2.4（plan 归 `tenant_plans` 表管）；[租户功能完整性审查](TENANT_FUNCTIONAL_COMPLETENESS_REVIEW_2026-10-27.md) TC-07 条目
**性质**：决策文档。本文件不改任何源码、不改合同文本；按 W3 先例，实现随小包另行落地。

---

## 0. 结论（签名）

**选择 A：套餐阶梯只管能力矩阵，自定义 plan code 就是配额桶——把后果写清，不改门控语义。**

- `tenant_plans` 表继续是**配额**的唯一权威（`tenant_plan_model.go`：quota 字段 + `UnlimitedQuota` 哨兵 + `AsQuota` 投影），自定义 code 的配额能力**今天就是完整的**，不因本决策改变。
- 前端能力矩阵（`core/entitlements/plans.ts` 的四档 rank + `lowcode.*` 三个 feature key）**只覆盖四个内建档**；操作员自定义 code 在矩阵中按 `free` 解析（fail-closed），其 lowcode 门控面不可见。这不是缺陷，是**刻意的呈现层语义**；本次要修的是**后果未告知**：`outsideLadder` 警告只说「不在阶梯内」，不说「代价是什么」。
- **B（服务端驱动 entitlements）登记为 V2 商业化工作流**，本轮不立项不实现——它是唯一让自定义 code 完整参与能力门控的方案，也是唯一需要改 schema 的方案（另立迁移包）。**C（移除套餐 UI 门控）否决**：等于把付费 lowcode 面开放给所有持有页面权限的租户，是产品决策层面的能力回退，不是实现细节。
- **决策方式**：维护者授权委派（delegated decision），2026-10-28 由执行代理依授权记录，**对维护者审查开放**。若维护者改选 B/C：B 按其规模新建迁移包，C 须先修订本决策与本文件 §2 选项 C 的否决理由。
- **落地**：A 的实现 = 两处文案/文档（① `outsideLadder` 警告补后果一句；② 设计文档 §2.4 相邻处写明「能力矩阵只覆盖内建四档，自定义 code = 配额桶 + free 能力档」），属 L1 小改，随独立小 PR 落地，**不占用 W7 本包**（W7 交付物是决策，不是 diff）。**本决策不计入 V1 完成度**（程序文档 §完成判据第 199 行口径）。

## 1. 问题复述（读点已逐条核实，2026-10-28，W5 合并后）

配额与能力门控分属两处，只有配额是数据驱动的：

| 读/写点 | 位置 | 行为 |
|---|---|---|
| 配额权威 | `backend/modules/system/iam/tenant/tenant_plan_model.go:11,70,93` | `tenant_plans` 行携带 quota 阈值，`AsQuota()` 投影给全部消费方；`UnlimitedQuota=-1` 哨兵 |
| 写路径校验 | `PlanStore.ValidateAssignment`（同一 store） | 按表内 `IsEnabled` 判定可分配；表不可读时 fail-open 回退内建四档（与前端同构） |
| 能力矩阵 | `frontend/src/core/entitlements/plans.ts:16-23` | 联合类型收拢为 `'free' \| 'basic' \| 'professional' \| 'enterprise'` |
| rank 表 | `plans.ts:58-63` | `free=0 … enterprise=3`，`isPlanAtLeast` 按 rank 比较 |
| 归一化 | `plans.ts:70-75` `normalizeTenantPlan` | 未知/缺失 code **fail-closed 归 `free`**——「陈旧后端或失败的 lookup 不得静默发放付费面」 |
| 编辑器警告 | `frontend/src/modules/system/iam/tenant/TenantList.tsx:272-273,890-893` | `planOutsideLadder` 触发 `system.tenant.plan.outsideLadder` 文案（五语言各一份） |
| 代码自述 | `planOptions.ts:4-20`、`TenantList.tsx:265-267`、`plans.ts:1-15` | 三处注释明说「operator-defined code resolves to free / 门控是 UX-only / 隐藏不等于授权」 |
| 门控消费方 | `usePlanEntitlements.ts` → `AppLayout.tsx`、`lowcode/generator/pages/ModuleWizard.tsx`、`platform/Dashboard.tsx` | 租户会话按 plan 隐藏 lowcode 路由/向导/挂件；平台视角与 compat 直通 |

**复核的真正贡献**（依包实现注记，不得把现状描述成本轮新发现的缺陷）：后果链没有被告知操作者——一个 `trial` code 的租户登录后，其配额按 `tenant_plans` 行生效，但生成器页面**不可见**，且警告文案不说明这一点；能力矩阵（`TenantPlanMatrix`）也不展示自定义 code 落在哪一档。这是**已文档化的限制**（三处代码自述），缺的是对操作者可见的后果表述与契约表述。

## 2. 三个选项

### 选项 A：阶梯只管能力矩阵，自定义 code = 配额桶（✅ 选定，V1）

**改什么**：

1. `outsideLadder` 警告文案补一句后果：「该 code 的配额按其 `tenant_plans` 行生效；能力矩阵中按 `free` 解析，lowcode 门控面将不可见」（五语言同步）。
2. `MULTI_TENANT_DESIGN.md`（§2.4 相邻处）与合同 §2.4 如实写明：能力矩阵只覆盖内建四档；自定义 code 是配额桶 + `free` 能力档；门控是 UX-only、隐藏不授予也不撤销权限。
3. `TenantPlanMatrix` 在自定义 code 下如实渲染 `free` 档位（现状即如此渲染，无需改逻辑；仅确认文案与矩阵一致）。

**设计约束**（继承自包实现注记）：`normalizeTenantPlan` 的 **fail-closed 语义不动**——它是「不得静默发放付费面」的安全语义；任何让自定义 code 获得 rank 的方案都属于 B，不属于 A。

**修掉的失败案例**：TC-07 的操作者可感知部分——警告说了代价，文档说了边界，矩阵与真实门控一致。

**可能新引入的失败案例**：
- 操作者看到警告后可能误以为配额也回退 `free`（实际不回退）——文案必须把「配额照常 / 能力按 free」两句分开说；
- 「一个自定义 code 想要 professional 能力」的诉求在 V1 内无法满足（须走 B）。

**成本**：文案 + 文档，一个 L1 小 PR，无 schema、无逻辑改动。**最小。**

### 选项 B：服务端驱动 entitlements（V2 商业化工作流，本轮不立项）

**改什么**：把 feature 集从编译期联合改为数据——`tenant_plans` 行（或新表）携带 per-plan feature 键集合，后端下发，前端按会话解析 rank/门控；自定义 code 即可获得操作者勾选的任何能力面。

**前置必修**：schema 迁移（feature 键存储 + 校验 + 审计）、后端下发 API 与缓存失效、前端矩阵与警告重写为数据驱动、五语言文案重排。

**修掉的失败案例**：自定义 code 完整参与能力门控；「trial 看不到生成器」类问题按配置解决。

**可能新引入的失败案例**：
- 特性矩阵成为可配置数据后，「隐藏页面」与「授权页面」的边界更容易被配错——需要新的契约条款与测试矩阵；
- 五档语义被打破后，`TenantPlanMatrix`、过滤器、总览统计的既有展示全部要跟随数据化重构；
- 复核基线（230 vitest，**无 entitlements 覆盖**）目前接不住这个改动面。

**成本**：**大**——schema + 后端 + 前端 + 契约 + 测试，多 wave。作为 V2「套餐商业化」工作流立项，届时新建任务包（本包不预创建）。

### 选项 C：移除套餐 UI 门控（❌ 否决）

**改什么**：删除 `PLAN_FEATURES`/`usePlanEntitlements` 门控与 `TenantPlanMatrix`，所有 lowcode 面对持有页面权限的租户一律可见。

**修掉的失败案例**：TC-07 按构造消失（不再有阶梯可供自定义 code 落在外面）。

**可能新引入的失败案例**：把付费 lowcode 面开放给每一个持有页面权限的租户——这是**产品决策本身**，不是副作用；套餐从「能力分层」退化为「纯配额」，既有四档语义与已交付的套餐编辑器能力面失去意义。

**成本**：实现为负（删代码），但产品能力与已交付面双重回退。否决。

## 3. 对比

| 维度 | A 阶梯=能力矩阵 ✅ | B 服务端 entitlements | C 移除门控 |
|---|---|---|---|
| schema/迁移 | 无 | 需要 | 无 |
| 自定义 code 能力面 | free（写明后果） | 操作者可配 | 全量放开 |
| 配额能力 | 照常（表权威） | 照常 | 照常 |
| 修复 TC-07 | 后果告知 + 契约表述 | 完整修复 | 按构造消失 |
| 新增风险 | 文案歧义（可控） | 配错面扩大 + 测试缺口 | 产品能力回退 |
| 成本 | 一个 L1 小 PR | 多 wave，V2 | 删代码但损失分层 |

**推荐**：A for V1，B as V2 commercialization workstream（立项时新建包，范围以 §2 选项 B 的前置与风险清单为起点）。

## 4. 存量数据处置

- **无存量迁移**：`tenants.plan` 列与 `tenant_plans` 行全部原样保留；本决策不触碰任何行。
- 已存在的自定义 code 租户行为不变（配额照常、能力按 free）；唯一变化是警告文案多一句后果说明。

## 5. 明确不做

- 在 W7 本包内改任何源码/文案/合同（本包 doNotTouch：`backend/**`、`frontend/src/**`、合同、API 文档）。
- 为 B/C 预创建空任务包。
- 把 `normalizeTenantPlan` 改成 fail-open，或在前端为自定义 code 伪造 rank——那都是 B 的实现片段，且破坏「不静默发放付费面」的安全语义。
- 将本决策计入 V1 完成度（程序文档口径：维持现状或立项均不计入）。
