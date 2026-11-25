---
title: Pantheon Base 多 Agent 交付流程
doc_type: Acceptance
layer: platform
status: Active
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
  - docs/contracts/DOCUMENT_GOVERNANCE_CONTRACT.md
updated_at: 2026-11-23
---

# Pantheon Base 多 Agent 交付流程

English version: [PANTHEON_BASE_DELIVERY_WORKFLOW.en.md](./PANTHEON_BASE_DELIVERY_WORKFLOW.en.md)

本文定义 `pantheon-base` 如何消费 [能力适配交付规则](../../../pantheon-harness/patterns/capability-adaptive-delivery.md)。默认由当前主 agent 完成规划、探索、实现、验证和修复，不固定 Claude/Codex 分工；可用工具、权限和任务风险决定是否委派。

当前个人维护阶段，先按 `../../../pantheon-harness/patterns/method-playbook.md`、`../../../pantheon-harness/patterns/minimal-complexity-ladder.md` 和 `../../../pantheon-harness/patterns/execution-guardrails.md` 判断任务档位与治理强度。本文主要定义 `L2` 和需要多 agent / 多 gate 的 `L1` 任务；`L0` 与普通 `L1` 不必强行套完整多 agent 调度。

目标是让人只提出目标和关键决策，不需要手动调度每个 agent。调度由当前协调 agent 负责完成，并把结果沉淀到仓库 artifact。

## 1. 角色分工

| 角色                          | 默认执行者                                 | 职责                                         | 约束                                        |
| ----------------------------- | ------------------------------------------ | -------------------------------------------- | ------------------------------------------- |
| Human Owner                   | 维护者                                     | 目标、风险接受、三触点决策                   | 不手动搬运工具上下文                        |
| Planner / Explorer / Executor | 当前主 agent，按阶段承担职责               | 范围、计划、结构检索、实现、验证、修复、证据 | 只读任务保持只读；按 In/Out 和 sandbox 执行 |
| Dispatcher                    | 当前主 agent，仅在需要委派时               | 所有权、预算、证据、停止条件和集成           | 保留其他人的编辑；不把聊天当事实源          |
| Reviewer / Evaluator          | 普通 L1 主 agent 自检；高风险独立 reviewer | findings-first 范围与证据审查                | 独立性不足必须记录 gap 与验收 gate          |
| Mechanical Gates              | GitHub Actions / local scripts             | 可重复验证、security、smoke、docs            | 不替代风险接受或授权                        |

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
  -> 主 agent intake：档位、能力、In/Out、验收和停止条件
  -> 同一上下文：计划 / 聚焦探索 / 实现 / 验证 / 修复
  -> 普通 L1 findings-first 自检；L2 review artifact，高风险独立 reviewer
  -> 必需证据与门禁通过，剩余 gap 明示
  -> 最终验收；仅任务包含发布时进入 PR/release 与分支收口
  -> Ratchet 或 closeout
```

自包含研究或独立实现分支才按收益委派，主 agent 负责合并结果并验证组合。

## 4. 阶段规则

### 4.1 Intake

Dispatcher 先判断任务类型：

- `trivial`：可直接执行，但仍要说明验证方式。
- `L1`：必须有轻量计划、明确 In / Out 和最小验证集合。
- `L2`：必须有 task packet 或明确继承父 task packet。
- `high-risk`：涉及 schema、权限、认证、审计、CI、secrets、发布或外部服务时提高证据与独立评审要求。是否停止由 §6 的现有授权判断，不因敏感域名称重复请求批准。

### 4.2 Planning

当前主 agent 先明确：

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

L2 的 Planner 输出落到 task packet 和现有 task manifest；普通 L1 可保留简短计划，不新建完整 manifest。

### 4.3 Exploration

当前主 agent 把计划映射到真实仓库：

- 结构性问题优先用 CodeGraph。
- 字面量、文案、日志、配置用 `rg`。
- 找到现有测试、fixture、smoke、contract checker。
- 识别 runtime-sensitive、UI-sensitive、security-sensitive 范围。
- 如果任务会改动 generated module 或 schema/generated，顺手确认 feature ledger 快照和 checker 影响面。

如果探索推翻假设，主 agent 在既定 In/Out 内修正方案并记录依据；独立 reviewer 纠偏。只有实质扩大范围或缺少授权才触发 human gate。

### 4.4 Execution

主 agent 或有明确所有权的执行分支必须遵守：

- 只改 task packet 范围内的文件。
- 先固定可验证行为，再实现。
- 保持 task packet、task manifest、evidence、review 的 linkage 一致。
- 更新对应文档、测试、fixture、smoke 或 checker。
- generated module 变更时要同步更新 generated registries 和 `schema/generated/feature-ledger.json`，不能只改一半。
- 保存或摘要 evidence。
- 明确 known gaps。

### 4.5 Review

按档位执行 findings-first 自检或独立评审：

- 是否满足 task packet。
- 是否有未声明的范围扩张。
- 是否破坏 `platform / system/* / business/*` 边界。
- 是否有安全、权限、审计、i18n、菜单、runtime 或 UI 状态缺口。
- evidence 是否足以支撑完成声明。

Review 必须 findings-first。没有 finding 时，也要写残余风险和未验证项。

### 4.6 Fix And Close

如果 review 有 P0/P1：

- 执行者只修已确认 finding，保留范围。
- 修完后重跑受影响 sensors。
- 高风险修复再次由独立 reviewer 对照最新 diff/evidence 审查。

如果同类问题重复出现：

- 更新 `docs/harness/failure-registry.md`。
- 选择最小 ratchet：guide、sensor、gate、template、adapter 或 no-action。

## 5. 工具使用策略

### 5.1 当前 Runtime

先确认当前模型、effort、客户端、检索/执行工具、sandbox、委派和恢复能力。未知字段写 unknown；保留用户已有选择和设置，不从模型名字推断工具或授权。默认直接使用当前主会话，任务计划不要求固定 Plan Mode 或 slash command。

### 5.2 可选调度 Adapter

原生 subagent、`acpx`、`omc` 或 CI matrix 只在实际可用、获准且有自包含分支收益时使用，不作为先决依赖。委派记录所有权、上下文路径、证据、预算和停止条件；辅助程序遵守 workspace Windows 隐藏启动规则。调度结果不是验证证据，不以角色数量衡量质量。

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

维护者按 workspace `AGENTS.md` 的三触点介入：①开工时的需求/范围澄清；②质量门禁策略、豁免或规则变更；③最终视觉/功能验收。

已授权范围内的安全可逆整改默认自主推进，用门禁与 evidence 收口。不能仅因涉及 schema、seed、权限、菜单、auth、audit 或 CI 就重复中途请求批准。触碰敏感域仍升到适当档位，补足回归与下游风险证据。

只有以下尚未获得授权的动作需要维护者决策：

- 破坏性/不可逆操作、外部生产或需要凭证的操作、正式发布。
- 新增外部服务/依赖，或 review 指出修复将实质扩大已确认范围。
- required/advisory、阈值或豁免策略变更。

Planner/Explorer 的范围差异由 Reviewer 对照 In/Out 纠正；没有实质范围变化时持续执行。已授权事项不重新索取同一批准。

## 7. 最小交付件

### 归档原始字节保护

`.harness/archive/` 中的历史 task/evidence 和保全快照保持原始字节及记录的 SHA；
格式化不用于修复历史文件的空行或布局。根 `.prettierignore` 排除整个归档目录，
默认 Prettier 与 `lint-staged` 使用该保护；活跃文档继续正常格式化。
不得通过替代 ignore 文件或显式关闭 ignore 来批量重写归档。
需要补充事实时新增当前 evidence 或归档说明，并保留原件与原哈希。

回归入口为 `node --test tests/scripts/harness-archive-formatting.test.mjs`，包含关闭
保护时的改写见证和正常 `lint-staged` 的工作树/索引字节检查。该共享配置按
Base-first 正式发行后供 Ops 消费，不能以本地复制代替发行继承。

L2 本地任务收口时至少要有 task packet/现有 manifest、验证证据、review 结论、known gaps 和 ratchet decision。以下发布字段只在用户要求 PR/release 收口时适用；普通 L1 用短计划与验证说明即可：

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

缺少本任务所需的证据或 review 时明确 gap；未要求的 PR/发布不作为本地实现完成条件。

## 8. 当前推荐模式

默认单主 agent 闭环；独立 review 或有明确收益的自包含分支按当前 runtime 能力委派。开工和 closeout 验证 `.harness/method-consumption.json`：

```powershell
node ../pantheon-harness/scripts/harness/check-method-consumption.mjs --root . --source ../pantheon-harness
```

规范变动先审查再显式 `--update`，不在验证时静默刷新。方法版本与产品 VERSION 独立；共享产品代码/配置/脚本更新仍按 foundation release 供 Ops 消费。
