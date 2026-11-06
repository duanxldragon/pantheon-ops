---
title: 审批流骨架设计
doc_type: Design
layer: platform / business/*
status: Draft
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
  - docs/contracts/SYSTEM_IAM_CONTRACT.md
  - docs/contracts/TENANT_CONTRACT_V1.md
updated_at: 2026-11-04
---

# 审批流骨架设计

> 未实现声明：本文为设计稿，仓库内尚无对应实现；在实现落地前状态保持 Draft。
> Not implemented: this is a design draft with no corresponding implementation in the repository; it stays Draft until implemented.

关联合同：
- `PLATFORM_CONTRACT.md`
- `SYSTEM_IAM_CONTRACT.md`
- `TENANT_CONTRACT_V1.md`

关联路线：
- `P2_SCALE_ROADMAP.md`
- `PANTHEON_BASE_ARCHITECTURE_OVERVIEW.md`

实现配套文档（同一设计包，落地前必须一起读）：

| 文档 | 职责 |
| :--- | :--- |
| 审批流数据库设计（planned/unavailable：`./APPROVAL_WORKFLOW_DB_DESIGN.md`；无当前交付承诺） | 表、列、索引、租户列、迁移编号与 down 形态 |
| 审批流接口契约（planned/unavailable：`./APPROVAL_WORKFLOW_API_CONTRACT.md`；无当前交付承诺） | REST 面、DTO、错误码、分页、权限键、`SecureAction` 范围、i18n 命名空间 |
| 审批流测试计划（planned/unavailable：`./APPROVAL_WORKFLOW_TEST_PLAN.md`；无当前交付承诺） | 验收标准 → 用例（含并发双审）、挂载的既有门禁、runtime gap |
| [工作流引擎选型](./WORKFLOW_ENGINE_SELECTION.md) | 本文沿用其结论：第一阶段自建轻量引擎，不引入 BPM/Temporal |

---

本文定义 Pantheon 标准后台产品位中的“审批流骨架”。

当前目标不是直接建设通用 BPM 平台，而是先收口“待审批 / 已处理 / 我的提交”的统一产品位、接入契约和审计语义。

## 1. 设计目标

审批流骨架要解决：

1. 平台层有统一审批入口，而不是每个业务模块各做一套待办
2. 不同业务域可以接入统一审批视图，但仍保留各自业务语义
3. 审批动作、状态流转和审计留痕具备一致表达

## 2. 非目标

当前阶段不做：

- 可视化流程设计器
- 通用规则编排引擎
- 任意节点脚本执行
- 跨组织复杂会签编排

## 3. 归属边界

### 3.1 `platform` 负责

- 审批中心入口
- 审批单统一列表模型
- 统一状态视图
- 与任务中心的聚合关系

### 3.2 `business/*` 负责

- 审批单业务语义
- 节点流转规则
- 业务详情页与处理动作
- 领域侧审计补充

一句话：

> `platform` 负责统一承载审批产品位，业务域负责真正的审批处理逻辑。

边界落地的两条硬约束（来自现有代码，不是本文新发明）：

- `modules/business/*` 不能直接依赖 `modules/system/*` 的 Service / Repository（仓库根 `DESIGN.md` §2.3）。业务域接入审批中心只能通过 `platform` 暴露的窄接口（§10 的注册契约）和自身的 `biz_*` 表，不能反手 import 平台审批 Service。
- 平台路由组的中间件链是有契约的：业务路由用 `contracts.ProtectedGroup` / `contracts.DataScopedGroup`（`backend/pkg/contracts/route_groups.go`，链路 `TokenAuth -> Casbin`（-> `DataScope`）），系统/平台控制台用 `contracts.SystemProtectedGroup`（`TokenAuth -> PlatformScopeTenantContext -> Casbin`，可选 Capability / DataScope / RefreshSync）。审批中心属于平台控制台，因此必须挂 `SystemProtectedGroup` 形态的链，不能自建裸 `r.Group(...)`。

## 4. 审批单模型

建议最小字段：

- `id`
- `workflowType`
- `sourceDomain`
- `bizType`
- `bizId`
- `title`
- `status`
- `currentStep`
- `applicantId`
- `currentApproverId`
- `submittedAt`
- `processedAt`
- `jumpTarget`

### 4.1 状态

最小状态：

- `draft`
- `submitted`
- `pending`
- `approved`
- `rejected`
- `cancelled`

这组值是对外口径（字典、i18n、筛选器都用它）。**权威的迁移规则以 §11 状态机为准**，其它文档不得各自再解释一遍迁移图。

### 4.2 默认视图

最小视图：

- `pending-for-me`
- `processed-by-me`
- `submitted-by-me`

视图是查询谓词而不是状态：`pending-for-me` = `status in (submitted, pending)` 且当前处理人集合命中我（含 §13 的代理人），`processed-by-me` = 我在 `system_approval_action_log` 里对单号执行过 `approve/reject/transfer`，`submitted-by-me` = `applicant_id = 我`。

### 4.3 模型补充字段（工程落地必需）

§4 的字段清单是产品视角；实现还需要下列列，完整定义见数据库设计文档：

| 字段 | 用途 | 来源约束 |
| :--- | :--- | :--- |
| `version` | 决策/转办/撤回的乐观锁基号（§15） | 本设计新增；仓库内已有列级版本先例 `backend/modules/system/refresh_sync.go:19` |
| `current_step_no` + `step_total` | 列表展示「第 2/3 步」而无需 join | §4 `currentStep` 的可查询化 |
| `due_at` | 超时与升级扫描依据（§16） | 本设计新增 |
| `escalated_count` | 升级次数上限守卫，防升级风暴（§16.3） | 本设计新增 |
| `terminal_reason` | 终态归因：`approved/rejected/cancelled/expired` | 与 `status` 分离，便于统计 |
| `tenant_id` | 租户归属（§21） | `TENANT_CONTRACT_V1.md` §2.3 |
| `deleted_at` | 软删除（GORM 默认） | `docs/designs/DATABASE.md` §1 |

## 5. 动作契约

第一阶段统一动作语义：

- `approve`
- `reject`
- `transfer`
- `cancel`

本文在该语义集上再补三条（§17 撤回、§13 委托、§15 并发都需要它们，但都不改变 §5 的四条对外口径）：

- `withdraw`（申请人撤回，§17）
- `escalate`（系统动作，非人工，§16）
- `delegate`（代理人处理，落库仍记 `approve/reject` + 代理标记，§13）

约束：

- 动作能力由源域决定是否暴露
- 平台层不直接承诺所有审批单都支持全部动作
- 不允许前端展示无后端语义支撑的伪按钮

### 5.1 动作-守卫矩阵

每个动作四要素：谁能触发、前置状态、守卫条件、副作用。缺任一要素即为未定义动作，不得进路由。

| 动作 | 触发者 | 前置状态 | 守卫 | 副作用 |
| :--- | :--- | :--- | :--- | :--- |
| `submit` | 申请人本人 | `draft` | 源域回调允许提交（§10.3）；bizId 存在且未删除 | `status: draft -> pending`；写 `submitted_at`、`due_at`；生成第 1 步实例 |
| `approve` | 当前步处理人或其代理人 | `pending` 且当前步 `running` | 非申请人（§14）；`version` 匹配；备注非空 | 记动作；末步 → `approved`，否则推进一步；刷新 `current_approver_id` |
| `reject` | 同上 | 同上 | 同上；备注必填（不允许无原因驳回） | 记动作；`status -> rejected`，整单终止 |
| `transfer` | 当前步处理人或其代理人 | `pending` | 目标人具该步候选角色且非申请人；不允许转给自己 | 改派 `assignee_id`；记 `transferred_from/to`；`version + 1` |
| `cancel` | 申请人本人 | `draft` 或 `pending` 且无人已决策 | 无 | `status -> cancelled`；终态 |
| `withdraw` | 申请人本人 | `pending` 且当前步 `running`（尚未有人决策） | 无历史动作记录，或源域显式允许「已审后撤回」 | `status -> cancelled`，`terminal_reason = withdrawn`；允许再提交（§17.3） |
| `escalate` | 系统（维护任务，§16） | `pending` 且 `due_at` 已过 | `escalated_count < 上限` | 记系统动作；改派到上级候选；不改变 `status` |

`approve` / `reject` 备注必填这条与现有实现口径一致：安全事件确认就强制备注，缺失返回 `auth.security_event.acknowledge.note_required`（`backend/modules/auth/security/security_service.go:366-389`）。

## 6. 与任务中心的关系

审批流和任务中心不是两套平行模型。

推荐关系：

- 审批单是业务对象
- 审批待办是任务中心中的一种任务
- 任务中心展示审批摘要
- 审批中心承载审批专属筛选和处理入口

实现口径（避免两份真相源）：任务中心不复制审批状态。[任务中心设计](./TASK_CENTER_DESIGN.md)落地时，审批类任务条目由审批单表**派生**（`derived` scope，见 `docs/designs/TENANT_RESOURCE_SCOPE_MATRIX.md` §1），只读投影 + `jumpTarget` 回审批中心；审批中心是唯一写入口。谁都不允许「在任务中心里把审批点掉」。

## 7. UI 骨架

第一阶段最小 UI：

- 审批中心列表页
- 三个视图页签：待我审批 / 我已处理 / 我的提交
- 状态筛选
- 行级跳转到源域详情或处理页

暂不要求：

- 拖拽流程图
- 节点级流程可视化编排
- 审批统计大屏

必须覆盖的页面状态（平台壳层契约，`docs/designs/EMPTY_LOADING_ERROR_STATES.md`）：loading / empty / error / forbidden / submitting。跨页选择集遵循 `PLATFORM_CONTRACT.md` §7.3.1：翻页不清空、改查询上下文清空。

批量动作（批量 approve / batch reject / batch transfer）是**独立权限点 + 独立路由 + 必须二次验证**，且必须返回部分成功结果（`successCount / failedCount / failures[]`）——这是 `SYSTEM_IAM_CONTRACT.md` §9.0 与 `SYSTEM_CONFIG_CONTRACT.md` §8.0 已确立的批量治理口径，审批不能另立一套。批量决策语义见 §15.4。

## 8. 审计与权限

至少需要：

- 审批中心访问权限
- 审批单查看权限
- 审批动作权限
- 审批动作审计记录

必须保留：

- 提交人
- 处理人
- 动作
- 时间
- 结果
- 备注摘要

权威形态在 §19（审计留痕）与 §20（权限键 / 菜单 / i18n）。

## 9. 最小样板要求

第一阶段必须至少选一个真实业务场景接入，避免空壳审批中心。

推荐样板：

- 业务资源上下线审批
- 高敏配置变更审批
- 资产录入审核

样板选择受租户契约约束（§21.2）：`modules/business/*` 在 V1 不参与租户隔离（`TENANT_CONTRACT_V1.md` §2.6），因此业务类样板在多租户 `multi` 模式下不具备租户语义；第一阶段样板必须以 `高敏配置变更审批`（`system/config` 域）为主样板，业务样板只承诺 compat 单租户部署可用。

## 10. 接入契约（源域注册）

审批中心不是「平台猜业务域想怎么审」。源域必须显式注册一个审批类型，注册信息是平台与业务之间唯一的契约面。

### 10.1 注册要素

| 要素 | 说明 | 缺失后果 |
| :--- | :--- | :--- |
| `workflow_type` | 稳定编码，`<domain>_<object>_<action>` 风格，如 `config_setting_change_approval` | 无法路由到处理器 |
| `source_domain` | `platform / system/* / business/*` 之一，直接落审计检索列 `source_domain`（`docs/designs/DATABASE.md` §3.2） | 审计无法按来源域筛选 |
| `biz_type` + `biz_id` 语义 | 源域自己的对象类型与主键；平台不解释其含义 | 跳转与回调无法定位 |
| `step_definitions` | 步骤序列（有序，第一步固定），每步一个候选集合（§12.1） | 提交即失败 |
| `actions` | 该类型支持的动作文案子集（§5） | 前端必须不显示该按钮 |
| `jump_target` | 详情页路由模板，如 `/system/setting/approval/:id` 或业务详情路径 | 列表行不可点 |
| `permission_prefix` | 该类型的动作权限键命名空间（§20.1） | Casbin 无法收敛 |

### 10.2 注册的存放形态

第一阶段不建「流程设计器」，注册以**代码内声明 + seed** 落库（`system_approval_workflow_type` 表，见数据库设计 §5），与仓库现有做法同构：菜单/权限点由 `backend/modules/system/seed.go` 的 `menuSeed` 结构体声明（`backend/modules/system/seed.go:203-250`），前端由 `frontend/src/modules/<域>/index.ts` 的 `defineModule(...)` 声明路由、菜单、权限与 i18n 命名空间（`frontend/src/modules/system/audit/index.ts`），注册总表在 `frontend/src/core/router/modules.ts`。

禁止：把「哪些类型接入」只写在文档或口头约定里。`workflow_type` 表是真相源，权限工作台式的缺口检查（§20.3）才能工作。

### 10.3 回调边界

平台在状态跃迁时通知源域，源域决定业务副作用（改业务表状态、发通知）。三条规则：

1. 回调是**同进程窄接口**（一个 Go 接口按 `workflow_type` 注册 handler），不是 HTTP 自调用；业务域不得 import 平台审批 Service 之外再做二次封装。
2. 回调与审批单跃迁在**同一事务**内执行：`submit` 的业务前置校验失败必须让整单不进入 `pending`。跨进程异步回调带来的「审批已通过但业务未生效」是本设计明确不接受的状态。
3. 回调失败 → 跃迁失败，返回业务错误 key；决策类动作的失败语义见 §15.2（不是 500 吞掉，而是 `approval.decision.side_effect_failed` 且单据保持 `pending`）。

## 11. 状态机

### 11.1 状态清单

| 状态 | 含义 | 是否终态 | 计入待办 |
| :--- | :--- | :--- | :--- |
| `draft` | 草稿，申请人可编辑，尚未占位审批人 | 否 | 否 |
| `submitted` | 已提交、正在解析首步处理人（瞬时态） | 否 | 是 |
| `pending` | 审批中，存在当前处理人 | 否 | 是 |
| `approved` | 通过，末步已完成 | 是 | 否 |
| `rejected` | 驳回，任一节点驳回即终止 | 是 | 否 |
| `cancelled` | 撤销：`cancel` 或 `withdraw`；`terminal_reason` 区分 | 是 | 否 |

`submitted` 是瞬时态：正常路径下同一事务内即转 `pending`；它存在只为了让「提交后处理人解析失败」有可解释的落点（该态由 §16 超时扫描兜底收敛为 `cancelled/failed`，不长期占待办）。

### 11.2 迁移表

| # | 起态 | 动作 | 止态 | 触发者 | 守卫（全部为真才允许） |
| :--- | :--- | :--- | :--- | :--- | :--- |
| T1 | （无） | `create` | `draft` | 申请人/源域 | `workflow_type` 已注册且 `enabled`；`biz_id` 存在 |
| T2 | `draft` | `update` | `draft` | 申请人 | 无人已决策；仅申请人 |
| T3 | `draft` | `submit` | `pending`（瞬时经 `submitted`） | 申请人 | §5.1；源域 `beforeSubmit` 回调通过；步骤定义非空 |
| T4 | `pending` | `approve`（非末步） | `pending` | 当前处理人/代理人 | 非申请人；`version` 匹配；当前步 `running`；备注非空 |
| T5 | `pending` | `approve`（末步） | `approved` | 同上 | 同上 + 当前步是末步 |
| T6 | `pending` | `reject` | `rejected` | 同上 | 同上 + 备注非空 |
| T7 | `pending` | `transfer` | `pending` | 当前处理人/代理人 | 非申请人；目标有效且非申请人；`version` 匹配 |
| T8 | `pending` | `cancel` | `cancelled` | 申请人 | 无任何 `approve/reject` 历史动作 |
| T9 | `pending` | `withdraw` | `cancelled` | 申请人 | 同 T8，或类型显式允许 `withdraw_after_decision` |
| T10 | `pending` | `escalate` | `pending` | 系统维护任务 | `due_at` 已过；`escalated_count < 上限` |
| T11 | `submitted` | `escalate`(超时) | `cancelled` | 系统维护任务 | 停留超过阈值；`terminal_reason = submit_stuck` |
| T12 | `approved/rejected/cancelled` | （无出边） | — | — | 终态不可逆；纠错走 §17.4 重新提交 |

规则：

- 迁移表是**唯一**权威。任何不在表内的组合必须被服务层拒绝，返回 `approval.transition_invalid`（§20.2 错误码表），不允许「前端不显示就行」。
- 终态不可逆。不存在 `approved -> pending` 的「重开」；重开等价于 T12 → 新单（§17.4），保留旧单作为审计事实。

## 12. 处理人解析与或签/会签

### 12.1 候选集合

一步的候选集合由三种来源之一确定（第一阶段只支持这三种，顺序即解析优先级）：

| 来源 | 说明 | 现有依赖 |
| :--- | :--- | :--- |
| `user` | 指定单个用户 | `system_user` |
| `role` | 指定角色键，候选 = 该角色下启用用户 | `system_role` / `system_user_role`（`system/iam`） |
| `dept_leader_chain` | 沿部门祖先链找负责人，逐级向上 | `system_dept.leader_user_id`、`ancestors`（`docs/designs/SYSTEM_ORG_DESIGN.md`；`system_dept` 是 `tenant-owned`，见 `docs/designs/TENANT_RESOURCE_SCOPE_MATRIX.md` §2） |

解析失败（候选为空）时：按 §16 的升级链回落到「上一级负责人 → 审批中心管理员角色」，仍为空则 T3 直接失败并返回 `approval.approver_unresolved`，单据留在 `draft`。不允许生成一条「没有审批人的 pending 单」，那会造出永远无法被处理的待办。

### 12.2 或签（第一阶段支持）

- 一步多候选、任一人决策即完成该步，称 `any`。
- 决策即竞争：先到的 `approve/reject` 生效，其余人在同一 `version` 上 CAS 失败，得到 `approval.decision_stale`（§15.2）。
- 已决策后，其他人的待办通过视图谓词自然消失（§4.2），无需额外通知补偿。

### 12.3 会签（第一阶段不做，语义先钉死）

`all`（全部候选必须决策）与按比例通过属于 §2「跨组织复杂会签编排」，第一阶段明确不做。为避免后续实现各写一套，先把规则写在这里：

- 引入时步内需要 `approval_step_vote` 记录，**每条投票独立幂等键**（`step_id + voter_id` 唯一），重新投票按 UPSERT 复活语义处理（同 `TENANT_CONTRACT_V1.md` §2.2 的成员复活语义）。
- 步完成判定必须是单行 CAS（`version`），不得靠「先查后改」的两步，否则并发投票会双满足阈值。

## 13. 委托与代理

### 13.1 模型

委托是「在时间窗内把某类待办的处理权转授给他人」，不是改单据的 `assignee`。

| 维度 | 规则 |
| :--- | :--- |
| 范围 | 按 `workflow_type` 粒度；不支持「全部类型无限期委托」（不可审计） |
| 时间窗 | `starts_at` / `ends_at` 必填，`ends_at` 必须晚于 `starts_at`，窗口上限由配置项控制（默认 90 天） |
| 层级 | **禁止链式委托**：A→B 生效期间，B 不得再委托给 C。落库守卫：委托人已处于「被委托」状态时拒绝，返回 `approval.delegate_chain_not_allowed` |
| 生效对象 | 只影响待办可见性（§4.2 `pending-for-me` 含代理窗口）与决策主体判定 |
| 撤销 | 委托人或具备治理权限的角色可随时置 `revoked_at`，历史动作不回滚 |

### 13.2 身份与留痕

- 代理人执行 `approve/reject` 时，`operator_id` = 代理人，`on_behalf_of_id` = 原委托人，两者都进 `system_approval_action_log`（§19.2）。审计里「谁点的按钮」和「权是谁的」必须是两个字段。
- 权限判定使用**代理人自身**的权限键与 Casbin 结果：委托不放大权限。若代理人在审批中心没有 `platform:approval:act`，或在其自身角色上没有该路由策略，则 `403 permission.denied`（`backend/internal/middleware/casbin_middleware.go` 按 `path + method` 判定，不因委托而绕过）。
- §14 的自审禁止同样作用于委托人：代理人不得是原申请人，委托人也不得通过「委托给自己」绕过。

### 13.3 待办归属的读取规则

`pending-for-me` 查询必须等价于：

```
当前步候选含我
  OR 当前步候选含某个「我是其代理人且窗口有效且未撤销」的委托人
```

窗口有效性由 DB 时间比较（`starts_at <= NOW() AND ends_at > NOW() AND revoked_at IS NULL`），不用应用层缓存时间，避免多实例时钟漂移导致待办忽隐忽现。

## 14. 自审禁止

一条硬规则：**申请人不能是自己的审批人**，且不能通过任何间接路径达成同一效果。

| 场景 | 判定 | 理由 |
| :--- | :--- | :--- |
| `operator_id == applicant_id` | 拒绝，`approval.self_approval_forbidden` | 直接自审 |
| 代理人 == 原申请人 | 拒绝，同上 | 委托绕道 |
| 委托人把「自己申请的单」委托给他人 | 允许 | 单据仍是别人审，合法；但待办来源仍要记录 |
| 角色候选集合里包含申请人本人（申请人恰好也是审批角色成员） | 该单在解析候选时**剔除申请人**；剔除后为空则走 §12.1 回落 | 不能因为「他真有这个角色」就允许自审 |
| 申请人同时是被委托人的上级，形成 `A -> B(申请人)` 的委托 | 拒绝建立委托 | 间接自审 |

实现要求：守卫必须写在**服务层**（写事务内），不能只在前端隐藏按钮。守卫输入是 `operator_id / applicant_id / on_behalf_of_id` 三元组，缺 `on_behalf_of` 检查即视为未实现。

与现有机制的关系：`SecureActionMiddleware` 只证明「这个会话刚做过二次验证」，它把 `claims.UserID` 与当前会话主体绑死（`backend/internal/middleware/secure_action_middleware.go:37-49`），所以二次验证不能替代表格里的自审判定——两者互补。

## 15. 并发与竞态

### 15.1 乐观锁基线

`system_approval_instance.version`（整单）与 `system_approval_step.version`（步内）是并发裁决位。所有跃迁走「带条件的 UPDATE + RowsAffected 判定」：

```sql
UPDATE system_approval_instance
   SET status = 'approved', current_step_no = 3, version = version + 1, updated_at = NOW(3)
 WHERE id = ? AND tenant_id = ? AND status = 'pending' AND version = ?;
```

`RowsAffected == 0` → 要么单据不存在/跨租户，要么已被他人决策 → 服务层返回 `common.NewConflict("approval.decision_stale")`。**响应形状必须是信封码而不是 HTTP 409**：`CodeForError` 把 `ErrConflict` 映射为 `code=400`，且本仓没有 409 通道（`backend/pkg/common/error.go:85-104`）；HTTP 状态仍是 200，因为业务失败统一走 `FailWithError → FailWithData`（`backend/pkg/common/http/response.go:47-59`、`:98-111`）。这个模式是仓库既有口径：安全事件确认就用 `RowsAffected == 0 → gorm.ErrRecordNotFound`（`backend/modules/auth/security/security_service.go:374-388`），而「唯一索引是并发下的最终裁决；预检保留为快速路径，输家统一走 `IsDuplicateKey`」是本仓已登记的治理结论（`docs/harness/tasks/2026-10-23-iam-residual-closeout.task.md:133`）。

### 15.2 双审同时决策（核心场景）

| 时点 | 会话 A | 会话 B | 结果 |
| :--- | :--- | :--- | :--- |
| t0 | 读单，`version = 7` | 读单，`version = 7` | 两者都看到可决策 |
| t1 | `approve`，CAS 命中 `version = 7` → 写成功，`version = 8` | — | A 成为赢家 |
| t2 | — | `reject`，CAS 条件 `version = 7` 不匹配 → `RowsAffected = 0` | B 得到信封 `code=400` + `approval.decision_stale`，**不写任何动作记录** |

要求：

- 失败方不得留下 `system_approval_action_log` 行（决策记录与状态跃迁同一事务，CAS 未命中就整体回滚）。审计里出现「没人采纳但已记录」的决策是缺陷。
- 失败方前端必须重新拉单并显示终态，不得自动重放为「那我用另一个决定」。
- 步内并发（或签）同理：`system_approval_step(version)` + 整单 `version` 两级都要更新，缺一会允许两个候选各自推进一步。

### 15.3 其它竞态

| 竞态 | 裁决手段 |
| :--- | :--- |
| `transfer` vs `approve` 同时发生 | 同一 `version` CAS；输家得 `approval.decision_stale`（信封 400） |
| 同一 `biz_type + biz_id` 并发重复提交 | 部分唯一索引 `uk(tenant_id, workflow_type, biz_type, biz_id, status_in_flight)`（数据库设计 §6），赢家插入成功，输家 `IsDuplicateKey → approval.duplicate_in_flight` |
| 超时扫描 vs 人工决策 | 扫描任务本身走同一 CAS 语句；扫到旧 `version` 就跳过本轮，不重试覆盖人工决定 |
| 维护任务多实例并发 | 复用 `backend/pkg/maintenance` 的重叠保护语义：同任务不重叠执行（`maintenance.Result{Outcome: OutcomeOverlap}`，`backend/pkg/maintenance/maintenance.go:155-190`） |

### 15.4 批量决策

批量动作在服务层**逐单**执行同一套 CAS，返回 `successCount / failedCount / failures[]`（`failures[].reason` 用错误 key）。批量不引入「一次 UPDATE 多行」的写法：那会让每条单据的 `version` 前置条件与回调（§10.3）无法逐单成立，也无法给出可信的部分失败。批量输入上限 500（与安全事件批量口径一致，`security_service.go:417-421`）。

## 16. 超时与升级

### 16.1 定义

- `due_at` = `submitted_at + 类型级 sla_hours`，或按类型配置的「工作日小时」近似（第一阶段只做自然小时，工作日历法归入开放决策 §23）。
- 过期未决策 = `status = 'pending' AND due_at < NOW()`。

### 16.2 扫描实现

第一阶段用平台既有维护任务注册器，而不是等未实现的调度中心：

```go
reg.Register(maintenance.Task{
    Name:     ApprovalTimeoutTaskName,
    Interval: maintenance.DefaultInterval,
    Run:      func(context.Context) error { return s.RunApprovalTimeoutEscalation() },
})
```

这是仓库现成模式（`backend/modules/system/audit/audit_service.go:317-323`，`backend/pkg/maintenance/maintenance.go`，注册点 `backend/modules/system/system_modules.go:103`，进程入口 `backend/cmd/server/main.go:205-212`）。默认 15 分钟一轮（`backend/pkg/maintenance/defaults.go:10`），因此 §16.1 的 SLA 精度承诺只能是「小时级」，不得宣称分钟级。[调度中心设计](./SCHEDULER_CENTER_DESIGN.md)落地后，本任务可迁移为其受管任务，迁移属实现变更，需回写本节。

### 16.3 升级动作

命中过期后按序执行，任一成功即止：

1. `escalate` 到部门祖先链上一级负责人（§12.1 第三类来源）。
2. 无上一级 → 改派到类型配置的兜底角色。
3. 兜底也为空 → 不再改派，写一条系统动作记录 + 通知扇出（§19.4），`escalated_count` 不再增加。

守卫：`escalated_count >= approval.max_escalations`（默认 2）时永久停止升级，防止每 15 分钟一轮的升级风暴把待办和通知刷爆。

### 16.4 提醒与自动通过

- 超时**不自动通过**，也不自动驳回。自动决策会让 SLA 变成「不审就过」的漏洞。此项如要改，属产品决策，见 §23。
- 到期前的提醒（如剩余 25%）只发通知，不改状态；通知扇出走 [通知中心设计](./NOTICE_CENTER_DESIGN.md)的契约，本设计不复制其模型。

## 17. 撤回与重新提交

### 17.1 撤回（withdraw）

- 只有申请人可发起（T9）。
- 允许条件：`status = 'pending'` 且**尚无任何 `approve/reject` 动作记录**。已有人决策的单不允许撤回后「当没事发生」——那会让决策记录与单据终态互相打脸。类型可用 `withdraw_after_decision` 显式放开，默认关。
- 语义是 `cancelled + terminal_reason = withdrawn`，不是物理删除，也不是软删（`deleted_at`）。撤回单必须仍可在「我的提交」里查到。

### 17.2 取消（cancel）

`cancel` 与 `withdraw` 的结果态相同、语义不同：`cancel` 是「作废草稿/停止流程」的通用动作（含源域侧作废），`withdraw` 明确是申请人主动撤回。审计与统计口径要能分开，因此 `terminal_reason` 必填。

### 17.3 重新提交

- 撤回/取消后允许申请人**基于同一 `biz_type + biz_id`** 重新提交：新建一条实例（新 `id`、`version = 0`、`source_instance_id` 指回上一单）。
- 不做「把 `cancelled` 单复活」：终态不可逆（§11.2），复活会破坏审计。
- 重复在途由 §15.3 的部分唯一索引挡住：同一 bizId 只能存在一条非终态实例，返回 `approval.duplicate_in_flight`。

### 17.4 通过后的纠错

`approved` 之后发现错误：走源域自己的「撤销已生效变更」流程，并在审批中心新建一条反向审批单（新 `workflow_type`），不在平台层加「反审批」伪状态。

## 18. 决策动作幂等

平台今天**没有**请求级幂等键基础设施（全仓 `idempot` 命中只在注释与迁移语义里，无 `Idempotency-Key` 中间件）。因此第一阶段幂等由三条服务端不变量提供，而不是靠客户端约定：

1. **状态守卫**：同一 `(instance_id, step_no)` 只允许一次生效决策。终态单再次 `approve` → `approval.transition_invalid`，且**不新增动作记录**。
2. **CAS**：`version` 不匹配 → `ErrConflict` 信封 `code=400` + `approval.decision_stale`（§15）。
3. **唯一键**：`(tenant_id, instance_id, step_no, operator_id, action)` 唯一索引裁决「同一人对同步的重复提交」；重放命中 `IsDuplicateKey` 时返回**首次结果**（成功 + 首次动作记录 id），而不是冲突码 —— 同一人重复点「通过」在业务上是同一件事。

响应重放所需的信息来自动作记录本身（`system_approval_action_log.id` + `instance.version`），不引入新表。

关于 `Idempotency-Key` 请求头：是否引入为通用平台能力（会影响其它域的写接口），登记为开放决策 §23-O3，由维护者拍板；本设计不预设。

网络重试的用户体验兜底：前端决策按钮必须 `submitting` 态 + 禁用，但这只是减震，不是幂等保证；服务端三条不变量才是。

## 19. 审计留痕形态

### 19.1 两层，不复算

| 层 | 载体 | 谁写 | 用途 |
| :--- | :--- | :--- | :--- |
| 统一操作日志 | `system_log_oper` | `middleware.OperationLogMiddleware`（`backend/internal/middleware/operation_log_middleware.go`），记录所有非 GET 请求 | 治理与合规检索、导出、按保留期清理 |
| 审批动作流水 | `system_approval_action_log`（新表） | 审批服务，与状态跃迁同事务 | 单据时间线、责任认定、`processed-by-me` 视图派生 |

两层不是重复：前者是「接口被调了」，后者是「业务上发生了什么」。时间线只读后者，不解析前者 payload。

### 19.2 动作流水字段

| 字段 | 含义 |
| :--- | :--- |
| `instance_id` / `step_no` | 定位 |
| `action` | `submit/approve/reject/transfer/cancel/withdraw/escalate` |
| `operator_id` / `operator_username` | 实际操作人（代理人场景是代理人） |
| `on_behalf_of_id` | 委托人；非代理场景为 0 |
| `result_status` | 该动作后的单据状态 |
| `from_assignee_id` / `to_assignee_id` | 仅 `transfer/escalate` 使用 |
| `note` | 备注摘要，决策类必填 |
| `version_before` / `version_after` | 并发裁决证据；事后重建时间线的关键 |
| `request_id` | 关联 `system_log_oper.request_id` 与 `X-Request-ID` 响应头（`backend/pkg/common/http/request_context.go:10-13`） |
| `tenant_id` | 租户归属（§21） |
| `created_at` | 动作时间 |

`request_id` 让「谁在哪个请求里点的通过」与统一审计双向可追（`docs/designs/DATABASE.md` §3.2 已把 `request_id` 定为串联接口响应、前端报错与统一审计的字段）。

### 19.3 写审计元数据

处理人必须给审批路由设置结构化标题与业务类型，而不是沿用默认路径标题：

```go
common.SetAuditMetadata(c, "audit.title.approval.decide", common.BusinessUpdate)
```

来源：`backend/pkg/common/audit.go:31-34`（`BusinessInsert/Update/Delete/Grant/Export/Import/Force/Clean` 枚举）。`source_domain / source_page / failure_category` 是统一审计的派生检索列，启动迁移回填、运行期不再补偿（`docs/designs/DATABASE.md` §3.2）。

### 19.4 与通知中心的关系

审批事件（新待办、即将超时、已升级、已决策）作为 `category = task` 的消息扇出到通知中心（`docs/designs/NOTICE_CENTER_DESIGN.md` §4 的消息模型与 §5 的 `jumpType/jumpTarget` 契约）。审批中心不自建消息投递、不自建已读状态；跳转目标必须回到审批详情或源域详情，`jumpType = none` 时不得伪造跳转按钮（通知中心 §5.1）。

## 20. 权限键、菜单与 i18n 影响

### 20.1 权限键清单

命名必须落在治理校验器的文法内：`(?:system|business|platform):[a-z0-9:-]+`（`scripts/harness/check-permission-contract.mjs:24`）。审批中心属 `platform`，动作权限用 `platform:approval:*`；业务域样板的动作权限归业务域自己的 `business:<module>:*`，不借用平台键。

| 权限键 | 语义 | 绑定路由（详见接口契约 §4） |
| :--- | :--- | :--- |
| `platform:approval:view` | 审批中心页面准入（页面权限） | `GET /api/v1/platform/approval/list` |
| `platform:approval:detail` | 读单号详情与时间线 | `GET /api/v1/platform/approval/:id` |
| `platform:approval:submit` | 发起/提交 | `POST /api/v1/platform/approval`、`POST /:id/submit` |
| `platform:approval:act` | 执行 `approve/reject` | `POST /:id/approve`、`POST /:id/reject` |
| `platform:approval:transfer` | 转办 | `POST /:id/transfer` |
| `platform:approval:withdraw` | 撤回 | `POST /:id/withdraw` |
| `platform:approval:batch-act` | 批量决策（独立键，不复用 `act`） | `POST /api/v1/platform/approval/batch-decide` |
| `platform:approval:delegate-manage` | 建立/撤销自己的委托 | `POST/DELETE /api/v1/platform/approval/delegations...` |
| `platform:approval:type-manage` | 维护 `workflow_type` 定义（高敏，写操作需二次验证） | `PUT /api/v1/platform/approval/types/:code` |
| `platform:approval:export` | 导出（`POST /export` 形态） | `POST /api/v1/platform/approval/export` |
| `business:<module>:approval-act` | 业务域自有动作（样板各自登记） | 业务域路由 |

四条硬规则：

1. **页面权限 ≠ 列表权限**，`:view` 用于页面准入、`:detail`/`:act` 用于动作；不得用一个 `:list` 兜住新增/审批/删除（`SYSTEM_IAM_CONTRACT.md` §7.2；`scripts/harness/check-permission-contract.mjs:122-145` 会直接报 `action menu Type F must not use a list permission`）。
2. **批量动作独立权限点 + 二次验证 + 部分成功返回**（`SYSTEM_IAM_CONTRACT.md` §9.0）。
3. 权限键在 `menuSeed` 的 `page_perm` / `perms` 里声明，`Type: "F"` 的动作节点用动作键（`backend/modules/system/seed.go:203-250`；页面权限用 `platform:dashboard:view` 是同类先例）。
4. 菜单只承载导航，权限靠 `system_role_permission` + Casbin 路由策略，两者解耦（`docs/designs/PERMISSION_MENU_DECOUPLING.md`、`backend/modules/system/iam/permission/`）。

### 20.2 错误码

服务端只返回稳定 key，不返回自然语言（`docs/designs/ERROR_CODE_AND_I18N.md` §3.1）。新增 `approval.*` 域码（完整表在接口契约 §5）：`approval.not_found`、`approval.transition_invalid`、`approval.decision_stale`、`approval.self_approval_forbidden`、`approval.duplicate_in_flight`、`approval.approver_unresolved`、`approval.note_required`、`approval.delegate_chain_not_allowed`、`approval.no_permission_on_item`。

### 20.3 权限工作台的登记义务

新键与路由一旦进入权限模型，必须同步加进 `requiredAPIRoutesByPermission` 映射表（`backend/modules/system/iam/permission/permission_workbench.go:622-644`）。该表有漂移守卫：`TestRequiredAPIRoutesExistOnEngine`（`backend/modules/system/iam/permission/permission_workbench_routes_test.go`）会用真实路由表逐条核对，写了不存在的路由就红。漏登记 = 权限工作台把审批中心当「未知权限」报出来。

### 20.4 菜单与路由

- 一级入口：审批中心挂在平台导航（`/platform/approval`），`Component` 用平台模块路径，`RouteName` 全局唯一（`check-menu-contract.mjs` 检查 routeName 重复）。
- 详情路由 `/platform/approval/:id` 是二级路由，需同时满足冒烟契约（`frontend/tests/smoke/system/system-routes.spec.ts` 一类）。
- 前端注册形态与 audit 模块一致：`defineModule({ routes, menus, dashboardWidgets, permissions, i18nNamespaces })`（`frontend/src/modules/system/audit/index.ts`），并加进 `frontend/src/core/router/modules.ts` 的 `systemModules` 数组。
- 页面准入：审批中心是**任务优先页**还是**治理型页**必须先定（`PLATFORM_CONTRACT.md` §7.4 只有治理型/审计型页面允许治理抽屉），本设计默认按任务优先页处理——不引入治理抽屉与 Hero 卡；工作台摘要卡由 dashboard 侧以 widget 形式消费（`cleanupPolicy: 'hide_when_forbidden'` 语义，同 audit 模块）。

### 20.5 i18n 影响

| 命名空间 | 内容 | 落点 |
| :--- | :--- | :--- |
| `system.menu.approval*` | 菜单标题 | `frontend/src/i18n/resources/zh-CN.ts` + `en-US.ts`（扁平点键，`system.menu.*` 现有约定） |
| `platform.approval.*` | 页面文案、列头、表单提示、视图页签 | 前端模块 `i18nNamespaces: ['platform.approval']` |
| `approval.status.*` / `approval.action.*` | 状态与动作文案（与 §11 状态、§5 动作一一对应） | 同上，配 `system_dict_*` 字典下发 |
| `approval.*`（错误 key） | §20.2 全部码 | 后端内置语言包 `backend/modules/system/i18n/builtin_locale_resources.json`（`auth.operation.verification_required` 等键即在此，zh-CN/en-US/ja-JP/ko-KR/fr-FR 五语齐备） |

约束：所有展示文本走 i18n，禁止硬编码（`frontend` 构建前置检查 `npm run check:i18n-hardcode`，见 `frontend/package.json` 的 `prebuild` 链）。新增状态/动作时，字典、后端 key、前端文案三处必须同批改齐，缺一即视为未完成（`SYSTEM_IAM_CONTRACT.md` §7.3「菜单、路由、权限点、i18n、seed 同步接入」）。

## 21. 数据与租户边界

### 21.1 scope 分类（必须登记）

按 `docs/designs/TENANT_RESOURCE_SCOPE_MATRIX.md` §1 的四分类：

| 表 | 分类 | 说明 |
| :--- | :--- | :--- |
| `system_approval_workflow_type` | `platform-global` | 类型定义是平台配置，无 `tenant_id`；同 `tenant_plans` 的「平台全局表不带列」理由（`docs/designs/TENANT_RESOURCE_SCOPE_MATRIX.md` §2） |
| `system_approval_instance` | `tenant-owned` | 带 `tenant_id`，读路径显式 `tenant.WithTenantScope` |
| `system_approval_step` | `derived` | 跟随实例，靠 `instance_id` 收敛，不独立承载租户判定 |
| `system_approval_action_log` | `derived` | 同上；审计事实 |
| `system_approval_delegation` | `tenant-owned` | 委托是「某租户内某人把待办授给某人」，必须按租户唯一 |

唯一键登记（矩阵 §3）：`system_approval_instance` 的在途唯一键 `(tenant_id, workflow_type, biz_type, biz_id)`（部分索引条件为非终态，见数据库设计 §6）必须在实现前登记进矩阵 §3。

### 21.2 与契约的一致性

- 解析优先级、`tenant_id = 0` 的平台语义、`multi` 是 `compat` 超集：一律按 `TENANT_CONTRACT_V1.md` §3.1/§6 与 §3.4 执行，本设计不另立规则。审批中心是平台控制台路由，链路必须带 `PlatformScopeTenantContextMiddleware`（`contracts.SystemProtectedGroup` / `SystemAuthGroup`，`backend/pkg/contracts/route_groups.go`）。
- 跨租户运营读（平台运维在 `multi` 下查全平台审批单）只允许走显式 `X-Tenant-Id` + 平台能力 + 落审计的路径（合同 §3.1 第 1 条、§3.3）；第一版**不实现**跨租户聚合视图，登记为 §23-O4。
- 业务域样板：`modules/business/*` 在 V1 不参与租户隔离（合同 §2.6），业务表不带 `tenant_id`。因此「租户 A 提交的业务审批单」实际指向全局业务对象——这是已知语义空洞，处理方式见 §9 与 §23-O1，禁止在文档里宣称业务审批已按租户隔离（合同 §2.6「禁止的宣称」）。
- 配额：审批相关配置（例如「每租户最多在途审批单数」）**只作度量与展示**，后端不得因超出套餐配额拒绝写入（合同 §1 Out：套餐配额只是度量展示维度）。若未来要做硬限流，属合同变更。

### 21.3 数据权限钩子

列表读路径是否叠加 IAM 行级数据范围（`self / dept_and_children / all`）是可选项：`contracts.SystemProtectedGroup` 的 `WithDataScope()` 决定是否挂 `middleware.DataScopeMiddleware`（`backend/pkg/contracts/route_groups.go:137-150`）。本设计规定：

- `pending-for-me` / `processed-by-me` / `submitted-by-me` 三个视图**不叠加** DataScope（视图谓词本身就是归属边界）。
- 「全部审批单」治理视图**必须**叠加 DataScope，否则一个只该看本部门的角色会看到全平台单据。该视图需要独立页面权限键，缺失时返回 `403 permission.denied`。

## 22. 完成定义

第一阶段完成标准：

- 有统一审批单模型
- 有审批中心入口
- 有最小动作契约
- 有至少一个真实业务样板接入
- 有权限与审计留痕

补充（配套文档给出的工程验收面，逐项可查）：

- §11 状态机迁移表 12 条全部有服务层守卫与错误 key，不存在「未列组合被默认允许」
- §14 自审禁止在服务层实现，覆盖委托间接路径
- §15 并发：双审同时决策有确定的失败语义（`ErrConflict` → 信封 `code=400` + `approval.decision_stale`；本仓无 409 通道），失败方不留动作记录；测试计划有对应用例
- §16 超时扫描注册进 `maintenance.Registry`，带重叠保护与升级次数上限
- §17 撤回产生新实例而非复活终态
- §18 幂等三不变量成立（重复同动作返回首次结果且不重复记录）
- §19 动作流水与 `system_log_oper` 通过 `request_id` 可串联
- §20 权限键、菜单、i18n、`requiredAPIRoutesByPermission` 四处同步登记，`Docs Governance` / 前端 `prebuild` 契约链全绿

不是以「已经做出完整 BPM 平台」作为完成标准。

## 23. 开放决策

以下问题本设计不替维护者决定；每条给出选项、影响面和建议。决策人统一为**维护者**（产品/架构最终裁定），实现方不得自行选默认值。

| 编号 | 议题 | 选项 | 影响面 | 建议 |
| :--- | :--- | :--- | :--- | :--- |
| O1 | `multi` 模式下业务域样板的租户语义 | A. 第一阶段样板只承诺 compat；B. 审批实例带 `tenant_id` 且业务详情跳转按 `tenant_id = 0` 全局对象处理；C. 等合同 §2.6 V2 前置条件满足再开业务样板 | §9、§21.2、验收范围 | B（不撒谎、不阻塞），并显式写明限制 |
| O2 | SLA 是否按工作日历法计算 | A. 自然小时；B. 工作日 + 节假日历（需要新配置与数据源） | §16.1、数据库列语义 | 第一阶段 A |
| O3 | 是否引入平台级 `Idempotency-Key` 中间件 | A. 只用 §18 三不变量；B. 平台新增幂等键能力（跨域共享合同变更） | 所有写接口、`route_groups.go` 链路 | A 先行；B 需要单独设计文档与合同评审 |
| O4 | 平台运维的跨租户审批聚合视图 | A. 不做；B. 按合同 §3.1 第 1 条 + `PlatformCapabilityMiddleware` 提供只读聚合 | §21.2、权限键、审计 | 第一阶段 A |
| O5 | 会签（`all` / 按比例）进入的时机 | A. 保持 §12.3 语义冻结、不实现；B. 第一阶段即实现 | §12、数据库表数量、测试面 | A |
| O6 | 超时是否自动决策 | A. 永不自动决策；B. 允许类型级「超时自动通过」 | §16.4、安全边界 | A；B 等同引入审批绕过面 |
| O7 | 审批中心页面准入归类 | A. 任务优先页（无治理抽屉）；B. 治理型页 | §20.4、`PLATFORM_CONTRACT.md` §7.4 白名单 | A，若做「全部审批单」治理视图则该视图另评 |
