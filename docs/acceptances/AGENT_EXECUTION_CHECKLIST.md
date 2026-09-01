---
title: Pantheon Base Agent 执行清单
doc_type: Acceptance
layer: platform
status: Active
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
  - docs/contracts/SYSTEM_AUTH_CONTRACT.md
  - docs/contracts/SYSTEM_IAM_CONTRACT.md
  - docs/contracts/SYSTEM_ORG_CONTRACT.md
  - docs/contracts/SYSTEM_CONFIG_CONTRACT.md
updated_at: 2026-06-10
---

# Pantheon Base Agent 执行清单

English version: [AGENT_EXECUTION_CHECKLIST.en.md](./AGENT_EXECUTION_CHECKLIST.en.md)

这份清单把 Harness Engineering 里最值得保留的几条实践压缩成 `pantheon-base` 的默认执行动作。

当前个人维护阶段的默认分级，先看 [pantheon-harness/architecture/methodology/solo-delivery-tiers.md](../../../pantheon-harness/architecture/methodology/solo-delivery-tiers.md)。本清单主要服务 `L1 / L2`，其中 `pantheon-base` 的共享合同、系统域、生成器、权限、审计、i18n 生命周期和继承边界任务默认按 `L2` 处理。

它不是新的模块合同，也不替代 [设计与实现验收清单](./ACCEPTANCE_CHECKLIST.md) 或 [代码评审标准](./CODE_REVIEW_STANDARD.md)。它解决的是更前置的问题：

- 一个任务开始前，先判断什么；
- 哪些改动必须有外部评估者视角；
- 哪些风险不能只靠“测试跑了”；
- 同类错误重复出现后，怎样升级成仓库规则或 checker。

## 1. 什么时候使用

- 所有 `non-trivial` 开发、修复、review、验收、drift 收敛任务都应使用。
- 对个人维护者，普通 `pantheon-base` 小中型任务可先按 `L1` 轻量闭环执行；一旦涉及共享底座高风险边界，直接升级 `L2`。
- 名义上是 trivial，但只要触碰 `schema`、权限、菜单、审计、i18n、路由、lowcode、动态模块、导入导出或继承边界，也应升级到这份清单。
- 建议与以下文档一起使用：
  - [../AGENTS.md](../../AGENTS.md)
  - [Pantheon Base Task Packet 模板](./TASK_PACKET_BASE_TEMPLATE.md)
  - [设计与实现验收清单](./ACCEPTANCE_CHECKLIST.md)
  - [代码评审标准](./CODE_REVIEW_STANDARD.md)

## 2. 默认判断顺序

开始动代码前，按这个顺序判断：

1. 这次改动属于 `platform`、`system/auth`、`system/iam`、`system/org`、`system/config`，还是 `business/*`。
2. 这是 `base-owned` 还是 `business-owned` 问题；如果在业务仓发现，也先判断是否应回 `pantheon-base` 修。
3. 任务类型包含哪些风险面：`UI`、`contract`、`schema`、`runtime-sensitive`、`inheritance-sync`、`generator`。
4. 谁负责实现，谁负责评审；不能默认由实现者自己宣布“已完成”。
5. 本轮最小证据是什么：测试、构建、smoke、浏览器证据、运行态证据，还是显式 gap 记录。

## 3. Base-Owned 优先判定

以下问题默认先视为 `pantheon-base` 归属：

- 平台壳层、导航、工作台、首页聚合、共享页面骨架。
- `auth / iam / org / config` 的通用行为、治理规则和共享接口。
- 通用权限、菜单、i18n、审计、共享上传、共享分页、共享表格。
- 低代码生成、动态模块注册、通用资源装配、共享 smoke helper。
- 对多个业务模块都有价值的通用 UX 修复或状态治理。

如果问题最初是在 `pantheon-ops` 或其他业务仓暴露，先问三件事：

- 它是否是共享能力缺口，而不是业务特有逻辑？
- 它是否会在第二个模块或第二个仓再次出现？
- 如果今天只在业务仓补丁修掉，是否会制造 drift？

只要其中任意一项答案偏向“是”，优先回 `pantheon-base` 设计或修复。

## 4. 约束增强，而不是自由扩张

Agent 任务的默认策略不是“给更多自由”，而是“给更窄但更稳定的边界”。

执行时默认遵守：

- 先应用最小复杂度阶梯：不做、复用、标准库、平台原生、已安装依赖、一条局部表达式，最后才写最小新增代码。
- 一次任务只收敛一个明确闭环，不顺手混入无关重构。
- 跨域改动优先按 gate 拆阶段，不要求一轮吃完。
- `task packet` 必须明确本轮 `In / Out`，并写清 `Do Not Touch`。
- 结构性定位先用 CodeGraph 缩小范围；字面量、日志、文案再用 `rg`。
- 如果需要解释某个规则为什么存在，优先把理由写进仓库工件，而不是留在聊天里。
- 不得用“简化”削弱鉴权、审计、i18n、可访问性、运行态证据或用户明确需求。

## 5. 实现者与评审者分离

每个 `non-trivial` 任务都要显式声明至少一个评审视角。默认视角如下：

- Architect Reviewer：层级、边界、合同、继承、drift。
- Security Reviewer：认证、权限、审计、敏感数据、依赖或外部服务。
- UX / QA Reviewer：页面状态、i18n、菜单、交互流程、浏览器证据。
- Mechanical Gate：lint、test、build、smoke、schema 或脚本校验。

默认规则：

- 高风险任务不应由实现者单独 self-approve。
- 评审输出使用 findings-first 风格，先报问题，再给摘要。
- handoff 必须指向同一份 task packet 和 evidence，而不是只写口头说明。

## 6. 重复失败 Ratchet Loop

同类错误不要只修一次代码，要决定它应该升级成什么方法资产。

建议按这个梯度推进：

1. **第一次出现**
   - 在 closeout、review 或 evidence 里明确写出失败模式和影响面。
2. **第二次出现**
   - 把规则写入 `AGENTS.md`、task packet 模板、执行清单或相关合同入口。
3. **第三次出现，或跨任务/跨仓重复出现**
   - 升级成 deterministic sensor：checker、smoke case、fixture、review gate，或 failure registry 条目。

只有当重复失败已经被脚本、测试或 gate 接住后，才算真正“沉淀成方法”。

## 7. Runtime-Sensitive 任务的最小证据

以下改动默认视为 `runtime-sensitive`：

- 登录、会话、令牌、鉴权链路。
- 菜单装配、权限守卫、路由准入。
- 导入导出、文件上传、批量任务、审批流。
- `generator`、`dynamicmodule`、注册/卸载/激活流程。
- 异步任务、外部集成、重试、幂等、并发相关逻辑。
- 明显涉及性能、超时、内存、请求风暴或资源竞争的变更。

这类任务除了测试外，至少还应具备以下证据之一：

- 聚焦日志或错误日志摘要。
- 一条完整 smoke / 浏览器路径。
- 关键指标或耗时样本。
- 请求链路或 trace 说明。
- 如果当前环境拿不到这些证据，必须写明 `runtime gap`、原因和风险。

不能只写“本地测试通过，所以运行态应该没问题”。

## 8. UI 与状态证据

只要触碰页面、路由、弹窗、抽屉、列表、表单、工作台或响应式布局，默认检查：

- `loading`
- `empty`
- `error`
- `forbidden`
- `submitting`

同时要求：

- 记录最终 URL 或路由。
- 记录 console error 结果。
- 有截图或显式 `visual gap` 说明。
- 布局受影响时，同时考虑 desktop / mobile 或明确说明只验证了哪一端。

## 9. Task Packet 最少补齐的字段

在 [Pantheon Base Task Packet 模板](./TASK_PACKET_BASE_TEMPLATE.md) 基础上，建议每个 `non-trivial` 任务至少补齐：

- 实现者视角：这轮主要在实现什么。
- 评审视角：架构 / 安全 / UX / mechanical 至少选择一项。
- 最小复杂度 rung：不做 / 复用 / 标准库 / 平台原生 / 已安装依赖 / 一条局部表达式 / 最小新增代码。
- 最小运行态证据：`none`、日志、smoke、trace、metrics，或显式 gap。
- 停点：在 schema、合同、删除、外部依赖或高风险动作前是否需要人工确认。

## 10. 完成时必须交付什么

最终说明至少包含：

- 改了什么。
- 没改什么。
- 跑了哪些命令。
- 证据放在哪。
- 还剩哪些未验证项或 residual risk。
- 如果影响 `pantheon-ops` 继承，是否需要 `base -> ops` 同步。
- 合并后的 PR URL 和 merge commit。
- 分支收口状态：远端 head branch 是否已删除、本地 worktree 是否已清理。

建议使用这个最小 closeout 模板：

```text
层级：
归属：
实现范围：
未做范围：
验证命令：
证据位置：
运行态 / 视觉 gap：
剩余风险：
是否需要 base -> ops 同步：
PR / merge URL：
分支收口：
待人工确认：
```

## 11. 什么时候推动方法升级

如果某条约束只是为了弥补旧模型、旧工具或旧流程的短板，而最近连续几个任务已经不再依赖它，应把它作为 `pantheon-harness` 的“退役候选”提出，而不是永久堆在 `AGENTS.md` 里。

反过来，如果某个错误已经连续出现，却还只能靠人工提醒，说明它还没有真正进入 harness。
