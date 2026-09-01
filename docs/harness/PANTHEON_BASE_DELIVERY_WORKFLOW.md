---
title: Pantheon Base 多 Agent 交付流程
doc_type: Acceptance
layer: platform
status: Active
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
  - docs/contracts/DOCUMENT_GOVERNANCE_CONTRACT.md
updated_at: 2026-06-23
---

# Pantheon Base 多 Agent 交付流程

English version: [PANTHEON_BASE_DELIVERY_WORKFLOW.en.md](./PANTHEON_BASE_DELIVERY_WORKFLOW.en.md)

本文定义 `pantheon-base` 如何使用 Claude、Codex、`acpx` 和 `omc` 执行 Harness Engineering 流程。

当前个人维护阶段，先按 `../../../pantheon-harness/patterns/method-playbook.md`、`../../../pantheon-harness/patterns/minimal-complexity-ladder.md` 和 `../../../pantheon-harness/patterns/execution-guardrails.md` 判断任务档位与治理强度。本文主要定义 `L2` 和需要多 agent / 多 gate 的 `L1` 任务；`L0` 与普通 `L1` 不必强行套完整多 agent 调度。

目标是让人只提出目标和关键决策，不需要手动调度每个 agent。调度由当前协调 agent 负责完成，并把结果沉淀到仓库 artifact。

## 1. 角色分工

| 角色             | 默认工具                       | 职责                                                                          | 禁止事项                       |
| ---------------- | ------------------------------ | ----------------------------------------------------------------------------- | ------------------------------ |
| Human Owner      | 人                             | 给目标、优先级、风险接受、human gate 决策                                     | 不手动搬运 Claude/Codex 上下文 |
| Dispatcher       | 当前协调 agent                 | 调用 planner、executor、reviewer，维护 task/evidence/review 链路              | 不把聊天记录当事实源           |
| Planner          | Claude                         | 需求澄清、设计边界、task packet、task manifest linkage、验收标准、stop points | 不直接改业务代码               |
| Explorer         | Codex                          | 代码结构、影响面、CodeGraph、现有测试和文档定位                               | 不扩大任务范围                 |
| Executor         | Codex                          | 实现、测试、证据、修复 review finding                                         | 不跳过 task packet 和 evidence |
| Reviewer         | Claude                         | findings-first review，检查范围、证据、安全、质量和剩余风险                   | 不直接修代码                   |
| Mechanical Gates | GitHub Actions / local scripts | 可重复验证、CI、security、smoke、docs governance                              | 不替代人工高风险决策           |

## 2. 人机交互原则

人只需要表达：

- 要解决什么问题。
- 哪些范围不能动。
- 是否接受高风险 gate。
- 是否允许进入下一阶段。

人不需要表达：

- “请用 Claude 规划”。
- “请把规划复制给 Codex”。
- “请再让 Claude review”。
- “请把 evidence 链接回 task packet”。

这些属于 dispatcher 的职责。dispatcher 必须自动完成工具路由，并在交付说明里写清楚调用了哪些角色、产生了哪些 artifact、还剩哪些 gap。

## 3. 标准调度流

```text
Human Goal
  -> Dispatcher intake
  -> Claude planner
  -> Dispatcher scope check
  -> Codex explorer
  -> Task packet and task manifest finalized
  -> Codex executor
  -> Local sensors and evidence
  -> Claude reviewer
  -> Codex fixer when needed
  -> GitHub / CI governance
  -> PR 合并与分支收口
  -> Ratchet or closeout
```

## 4. 阶段规则

### 4.1 Intake

Dispatcher 先判断任务类型：

- `trivial`：可直接执行，但仍要说明验证方式。
- `L1`：必须有轻量计划、明确 In / Out 和最小验证集合。
- `L2`：必须有 task packet 或明确继承父 task packet。
- `high-risk`：涉及 schema、权限、认证、审计、CI、secrets、删除、发布或外部服务时，必须停下来请求 human gate。

### 4.2 Claude Planning

Claude 只负责规划输出：

- 任务归属层：`platform / system/auth / system/iam / system/org / system/config / business/*`
- In / Out / Do Not Touch
- Assumptions and Open Questions
- Minimum Viable Approach
- Success Criteria
- 受影响合同、设计、验收文档
- 最小验证集合
- runtime evidence 或 visual evidence 预期
- 若任务触及 generated capability，还要说明 feature ledger 和 drift checker 的预期产物
- human gate 和 stop points
- task manifest / evidence / review linkage
- 是否需要分阶段执行

Planner 输出必须能落到 `docs/harness/tasks/*.task.md` 与 `.harness/tasks/<task-id>/manifest.json`，不得只留在对话里。

### 4.3 Codex Exploration

Codex 负责把计划映射到真实仓库：

- 结构性问题优先用 CodeGraph。
- 字面量、文案、日志、配置用 `rg`。
- 找到现有测试、fixture、smoke、contract checker。
- 识别 runtime-sensitive、UI-sensitive、security-sensitive 范围。
- 如果任务会改动 generated module 或 schema/generated，顺手确认 feature ledger 快照和 checker 影响面。

如果 exploration 发现 planner 假设错误，dispatcher 先回到 Claude 或 human gate，而不是让 Codex自行扩大实现范围。

### 4.4 Codex Execution

Codex 执行时必须遵守：

- 只改 task packet 范围内的文件。
- 先固定可验证行为，再实现。
- 保持 task packet、task manifest、evidence、review 的 linkage 一致。
- 更新对应文档、测试、fixture、smoke 或 checker。
- generated module 变更时要同步更新 generated registries 和 `schema/generated/feature-ledger.json`，不能只改一半。
- 保存或摘要 evidence。
- 明确 known gaps。

### 4.5 Claude Review

Claude review 只做评审：

- 是否满足 task packet。
- 是否有未声明的范围扩张。
- 是否破坏 `platform / system/* / business/*` 边界。
- 是否有安全、权限、审计、i18n、菜单、runtime 或 UI 状态缺口。
- evidence 是否足以支撑完成声明。

Review 必须 findings-first。没有 finding 时，也要写残余风险和未验证项。

### 4.6 Fix And Close

如果 review 有 P0/P1：

- Codex 只修 review finding。
- 修完后重跑受影响 sensors。
- 必要时再次 Claude review。

如果同类问题重复出现：

- 更新 `docs/harness/failure-registry.md`。
- 选择最小 ratchet：guide、sensor、gate、template、adapter 或 no-action。

## 5. 工具使用策略

### 5.1 `acpx`

`acpx` 是默认可控执行入口，用来稳定调用指定 agent、session 和权限策略。

Dispatcher 可使用：

```powershell
acpx --cwd D:\workspace\go\pantheon-platform\pantheon-base --approve-reads claude -s planner "<planning prompt>"
acpx --cwd D:\workspace\go\pantheon-platform\pantheon-base --approve-reads codex -s executor "<execution prompt>"
acpx --cwd D:\workspace\go\pantheon-platform\pantheon-base --approve-reads claude -s reviewer "<review prompt>"
```

这些命令是 dispatcher 内部操作，不要求 Human Owner 手动执行。

### 5.2 `omc`

`omc` 用作 agent 发现、能力路由和未来编排入口。

当前使用原则：

- `omc discover` 和 `omc agents` 可用于确认可用 agent。
- `omc -q` 只用于低风险只读分析或快速问题。
- 非 trivial 任务不要完全依赖自动路由，除非配置已明确把 planning 交给 Claude、execution 交给 Codex、review 交给 Claude。
- 如果 `omc` 的实际路由与本文角色分工冲突，以本文和 `docs/harness/AI_QUALITY_GOVERNANCE.md` 为准。

### 5.3 GitHub Actions

GitHub Actions 是 mechanical gates，不是 planner。

worktree 只负责本地实现和验证，不是长期保留的交付物。更完整的 worktree -> push -> PR -> merge -> cleanup 约定见 `docs/designs/WORKFLOW.md`。

PR required:

- `Quality Gates`
- `Security Gates`

辅助或周期性：

- `Full Smoke Suite`
- `Deep Dependency Audit`

CodeQL 在 `Security Gates` 中承担主安全信号；质量和设计风险则通过 Copilot review 状态与 PR residual-risk 留痕补齐。`Full Smoke Suite` 是深度信号，不应替代本轮 focused evidence。

## 6. 自动调度时的停点

Dispatcher 必须在以下情况停下来请求 Human Owner 决策：

- 修改数据库 schema、seed、权限模型、菜单模型、认证链路。
- 删除大量文件、批量迁移或影响上游/下游继承。
- 引入新外部服务、依赖、secrets 或发布流程。
- Planner 和 Explorer 对任务范围结论冲突。
- Review 暴露 P0/P1，但修复会扩大原 task 范围。
- CI gate 需要从 required 改成 advisory，或反过来。

## 7. 最小交付件

非 trivial 任务收口时至少要有：

- task packet 或父 task packet 链接。
- task manifest 路径。
- evidence 路径或命令摘要。
- reviewer 角色和 review 结论。
- GitHub signal 分类：required、advisory、scheduled、manual。
- 合并后的 PR URL 和 merge commit。
- 分支收口状态。
- known gaps。
- ratchet decision。
- generated capability 任务还要包含 feature ledger snapshot 路径或严格 checker 的 evidence 摘要。

如果缺少其中任意一项，不能把任务描述为完整闭环，只能描述为阶段性进展。

## 8. 当前推荐模式

短期采用 `acpx` 作为确定性调度入口，`omc` 作为发现和轻量路由入口。

当 `omc` 配置稳定满足以下条件后，再升级为主编排入口：

- planner 固定为 Claude。
- executor 固定为 Codex。
- reviewer 固定为 Claude。
- dispatch 结果能写回 task packet、evidence 和 review artifact。
- 不支持的能力会显式记录 gap，而不是静默跳过。

在此之前，dispatcher 可以调用 `omc` 辅助，但不能把 `omc` 自动路由结果当作完成证据。
