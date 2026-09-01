---
title: 代码质量与安全治理策略
doc_type: Design
layer: platform
status: Active
linked_contracts:
  - docs/contracts/PLATFORM_CONTRACT.md
  - docs/contracts/DOCUMENT_GOVERNANCE_CONTRACT.md
updated_at: 2026-06-23
---

# 代码质量与安全治理策略

English version: [QUALITY_AND_SECURITY_STRATEGY.en.md](./QUALITY_AND_SECURITY_STRATEGY.en.md)

本文定义 Pantheon 的代码质量、安全扫描和第三方工具使用策略。目标是建立可持续的质量体系，让 GitHub Actions、CodeQL 和 GitHub 原生 AI 评审各司其职，不让多套外部扫描器同时抢合并权。

代码重复率仍然是维护性信号，但不再以 Sonar 或任何单一工具分数作为治理目标。重复治理必须回到工程语义：消除重复业务决策、重复安全规则和重复协议转换；允许低风险语法重复，避免为了数字制造无 owner 的抽象。

## 1. 总原则

1. **只保留一个主门禁体系**：GitHub Actions + Branch Protection 是合并依据。
2. **只保留一个主安全信号**：CodeQL 是代码安全分析的主信号。
3. **架构与意图风险要有 AI 评审补位**：GitHub Copilot review 可作为架构边界、业务意图和可维护性补充信号，但不替代 required checks。
4. **不再依赖 Codacy**：当前流程不使用 Codacy 作为 required check 或合并前置条件。
5. **风险优先，不追求零噪音**：只阻塞会影响安全、正确性、共享底座稳定性或可回归性的事项。纯风格、低风险重复和无可达风险的建议进入 backlog。

当工具结论冲突时，优先级固定为：

```text
功能正确性：GitHub Actions 测试/构建/契约检查
安全可达性：CodeQL + secret/dependency/workflow posture
质量与设计兜底：Copilot review + PR 说明中的 residual risk
外部对比：无
```

## 2. 仓库分层

### 2.1 pantheon-base

`pantheon-base` 是后续业务仓库的底座，策略必须更严格。

必须阻塞合并的情况：

- GitHub required checks 未通过。
- CodeQL 命中可达的高危安全问题。
- `system/auth`、`system/iam`、`system/config`、权限、审计、共享 `pkg/*`、生成器、CI、凭据处理出现未解释的高风险变更。
目标阈值：

- New Code 覆盖率默认不低于 `80%`，例外必须在 PR 中说明补测计划。
- 高风险改动必须在 PR 中写明 residual risk、回滚方式和补充验证，而不是依赖额外人工审批才允许合并。

前端重复代码是当前主要维护性热点。治理顺序固定为：

1. 优先治理真实运行时页面、共享组件、布局、表单、表格和生成模板。
2. 其次治理 smoke、fixture、示例代码中的重复，避免它们被误当成生产代码或反复触发安全扫描。
3. vendored 方法层脚本如 `scripts/harness/` 不计入产品重复率；它们由 `pantheon-harness` 统一治理。
4. 不为了压低数字盲目抽象业务叶子逻辑；只有当重复影响维护、安全或生成模板时才抽公共层。

### 2.1.1 重复治理与抽象边界

重复治理遵循：

- **重复业务决策必须治理**：认证策略、权限判断、数据范围、审计规则、导入导出协议、生成器模板等不能多处各自实现。
- **重复安全规则必须治理**：登录失败节流、会话有效性、密码策略、token/cookie 处理、CSRF/CSP 等必须有唯一 owner。
- **重复页面模板应进入模板或生成器**：列表页、筛选区、批量操作、治理侧栏等稳定模式可以抽象。
- **重复 JSX/类名/数字不自动抽象**：除非它已经是设计系统 token、布局契约或可访问性交互规范。
- **公共层必须有 owner**：`pkg`、`components`、`hooks` 不能成为为了减少重复而产生的杂物层。

反例：

- 只为减少 `div className` 重复而新增组件。
- 只为减少列宽数字重复而新增业务常量中心。
- 把 auth/session 的领域动作放进 `pkg`。
- 新增无人真实调用的通用 hook。

正例：

- 同一认证策略只在 `system/auth` 一个 owner 内实现。
- 页面模板重复进入 generator 或 page template。
- 治理洞察、侧栏摘要等产品语义沉淀为共享组件。
- 平台契约通过 `pkg/contracts` 暴露，而不是暴露领域实现细节。

### 2.2 pantheon-ops

`pantheon-ops` 是业务仓库，继承 base 的安全和架构底线，但不复制 base 的所有治理文档。

必须坚持：

- 通用后台底座问题回流到 `pantheon-base`。
- 业务模块只在 `business/*` 内演进，不能反向侵入系统域。
- 触达 auth、IAM、配置、权限、审计、CI、凭据的变更按 base 高风险标准执行。

建议阈值：

- 核心业务链路新增或修改时必须有测试、smoke 或验收记录。
- 业务叶子模块的低风险重复可以先记录 backlog，但不能扩散到共享包、平台壳层或系统域。

## 3. 工具职责

### 3.1 GitHub Actions

GitHub Actions 是主门禁，负责：

- 后端测试。
- 前端 lint、build、契约检查。
- 文档、菜单、i18n、generated module 等项目专项检查。
- PR 和 merge queue 的 required checks。

Branch Protection 只要求 GitHub-native checks。不要把 Codacy 或其他外部扫描器的 check 加入 required checks。

### 3.2 CodeQL

CodeQL 是主安全信号，负责代码级安全分析。

- `pantheon-base` 上 CodeQL finding 默认按阻塞项处理，除非确认误报并留下依据。
- `pantheon-ops` 上业务域 finding 可按风险分级处理，但触达高风险域时必须按 base 标准处理。
- CodeQL 发现的高危问题优先于任何 AI review 评论或非阻塞质量建议。
- CodeQL analysis job 成功不等于仓库没有 open alerts；`Security Gates` 必须记录 open alert 报告。既有 high/critical baseline 在 PR / merge queue 上先 report-only，在 protected-branch push、scheduled 或 manual 安全复核上执行。

### 3.3 Dependency、Secret、Workflow Posture

依赖、密钥和 GitHub Actions 工作流态势扫描是供应链和 CI 安全补充。

- 新增依赖、升级依赖、修改 workflow、触达密钥处理时必须看这些报告。
- 非可达、开发依赖或低风险噪音可进入 backlog，但必须说明为什么不阻塞。
- 真实 secret 泄露、可达高危依赖漏洞和危险 workflow 权限必须立即处理。

### 3.4 Copilot review

Copilot review 是 GitHub 原生的辅助信号，负责补充扫描器覆盖不到的事项：

- 架构边界是否明显漂移
- 业务意图是否可能被误实现
- 是否存在低成本即可修复的可维护性或安全提示

规则如下：

- Copilot review 默认自动请求，或在仓库/账号能力允许时启用自动评审
- Copilot review 只产生 comment，不作为 required approval
- Copilot 不可用时，PR 仍可依赖 `Quality Gates`、`Security Gates` 和 PR 留痕继续推进
- 自动打开 squash auto-merge 前，PR body 仍必须先通过仓库治理校验，避免 auto-merge 绕开留痕要求
- 高风险改动必须在 PR 中显式记录 residual risk、回滚方式和后续跟进

## 4. 例外处理

允许例外，但必须轻量留痕：

- PR 描述写明例外原因。
- 标明是否影响 `pantheon-base` 底座稳定性。
- 给出补测、重构或清理计划。
- 高风险例外必须有明确 residual risk、回滚方案和 follow-up。

禁止的例外：

- 已确认可达的高危漏洞。
- 未 review 的 Security Hotspot。
- 真实 secret 泄露。
- 破坏权限、认证、审计或配置安全边界的变更。

## 5. 快速开发保护

质量治理不能变成开发阻力。默认执行以下规则：

- 先修阻塞风险，再处理趋势问题。
- 前端重复代码优先通过共享组件、页面模板和生成模板治理，但共享组件必须有设计系统或产品语义。
- 后端重复只在共享包、系统域、高风险链路或真实维护成本出现时治理。
- 生成代码、smoke 代码和 fixture 代码必须能清理、隔离或明确排除，避免长期污染运行时代码质量。
- 对低风险质量建议使用 backlog，不在功能 PR 里无限扩张重构范围。

## 6. PR 留痕

PR 至少记录：

- GitHub required checks 结果。
- CodeQL 结果或链接。
- Copilot review 是否已请求、是否自动评审、是否存在关键评论。
- 是否触达高风险域。
- 如果触达高风险域，residual risk 和回滚方式是什么。
