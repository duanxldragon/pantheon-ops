---
title: Review Loop Spec
doc_type: Contract
layer: platform
status: Active
updated_at: 2026-06-23
---

# Review Loop Spec

English version: [REVIEW_LOOP_SPEC.en.md](./REVIEW_LOOP_SPEC.en.md)

本文定义 Pantheon 的工具无关 review loop。实现者和 reviewer 可以是不同工具或人工，但输出格式必须一致。

## 1. Review 目标

Review 不是总结变更，而是发现：

- 行为回归
- 安全风险
- 架构边界漂移
- 权限、菜单、i18n、审计缺口
- 测试和证据不足
- base/business 继承风险

## 2. Review 角色

至少支持以下角色：

- Architect Reviewer：层级、边界、合同、继承、drift。
- Security Reviewer：认证、权限、审计、敏感数据、依赖安全。
- UX / QA Reviewer：页面状态、i18n、菜单、浏览器证据、响应式。
- Mechanical Gate：CI、lint、test、smoke、静态检查。

同一工具可以扮演多个角色，但高风险任务建议实现者和 reviewer 分离。

### 2.1 默认分离规则

对 `non-trivial` 任务，review 默认不是作者自检，而是实现之后的外部评估环节。

允许 self-review 的情况：

- trivial 任务
- 低风险文档整理
- 当前环境只有单执行者，且已记录原因与剩余风险

以下情况默认需要独立 reviewer 或显式 review gate：

- 触发 human gate 的任务
- security / permission / audit / schema / inheritance 边界任务
- release、CI、secrets、删除或高影响运维流程变更

## 3. Review 输入

Reviewer 必须读取：

- task packet、task manifest 或用户请求
- implementer posture 与 reviewer posture 说明
- Structural Scope / Affected Subgraph 说明，如任务已提供
- diff
- 相关合同和验收文档
- verification evidence
- CI 或本地验证结果

### 3.1 CodeGraph 结构性审查最小要求

对 `non-trivial`、跨层、runtime-sensitive、权限/菜单/i18n/审计/生成器/动态模块相关任务，review 默认至少回答：

- 本轮实际修改的是哪条受影响子图，而不是泛泛而谈“改了某模块”。
- 是否引入新的循环依赖、扩大已有循环簇，或让关键依赖方向变得更模糊。
- 是否把单个 handler、service、registry 或 orchestrator 推成新的 hub 节点。
- 是否让关键调用链显著变深，导致验证点、失败点或回滚点更难定位。
- 是否存在未验证输入穿过关键边界后直达权限判断、外部 side effect、审计缺口或敏感操作。

这些结构性检查是 review gate 的一部分。它们通常以 P1/P2 findings 体现，而不是作为脱离上下文的全局 KPI。

## 4. Findings 格式

Findings 必须优先输出，按严重程度排序：

```text
[P0|P1|P2] (confidence: N/10) file:line - 问题描述
Impact:
Fix:
Verification:
```

严重程度：

- P0：安全、数据损坏、构建失败、核心链路不可用。
- P1：模块不可达、权限/i18n/菜单契约失效、明显行为回归。
- P2：治理、文档、测试覆盖、可维护性缺口。

## 5. 无发现格式

如果没有发现问题，必须写：

```text
No P0/P1/P2 findings found.
Residual risk:
Verification checked:
```

## 6. Approval 状态

Review 结论只能是：

- `approved`
- `changes requested`
- `blocked`
- `approved with documented P2 follow-up`

P0/P1 未解决时不能 approved。

## 7. 多工具协作

允许组合：

- Codex 实现，Claude Code review。
- Cursor 实现，Codex review。
- 人工实现，agent review。
- Agent 实现，人工 final approval。

但所有 review 都必须引用同一 task manifest 和 verification evidence；如仓库保留 task packet，也应引用同一 task id 的 task packet。

## 8. Review Artifact Linkage

如果 review 结果需要作为仓库内 artifact 留存，默认放在：

```text
.harness/evidence/<task-id>/review.md
```

推荐最小模板：

````md
# Review Summary: <task-id>

## Machine Readable

```json
{
  "taskId": "<task-id>",
  "verdict": "approved",
  "structuralReview": {
    "affectedSubgraph": [
      "entry -> core path -> exit/side effect"
    ],
    "checks": [
      "cycle",
      "hub",
      "call-depth",
      "sensitive-flow"
    ],
    "findings": [],
    "notes": "none"
  },
  "linkage": {
    "taskManifest": ".harness/tasks/<task-id>/manifest.json",
    "evidence": ".harness/evidence/<task-id>/commands.json",
    "reviewFile": ".harness/evidence/<task-id>/review.md",
    "changeRef": "openspec/changes/<name>/",
    "planRefs": [
      "docs/superpowers/plans/<file>.md"
    ]
  }
}
```

## Linkage

- Task Manifest: `.harness/tasks/<task-id>/manifest.json`
- Evidence: `.harness/evidence/<task-id>/commands.json`
- OpenSpec Change: `openspec/changes/<name>/` | none

## Verdict

approved | changes requested | blocked | approved with documented P2 follow-up

## Findings

No P0/P1/P2 findings found.

## Structural Notes

- Affected subgraph: `entry -> core path -> exit/side effect`
- Checks: `cycle`, `hub`, `call-depth`, `sensitive-flow`
- Findings: none

## Residual Risk

- none

## Verification Checked

- `command`
````

如仓库保留人类可读 task packet，可在 `## Linkage` 或 PR / review comment 中一起引用；但 machine-readable 闭环字段以 `taskManifest`、`evidence`、`reviewFile`、`changeRef`、`planRefs` 为准。
