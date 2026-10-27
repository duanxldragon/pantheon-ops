---
title: Triviality Classification Policy
doc_type: Contract
layer: platform
status: Active
updated_at: 2026-06-08
---

# Triviality Classification Policy

English version: [TRIVIALITY_CLASSIFICATION_POLICY.en.md](./TRIVIALITY_CLASSIFICATION_POLICY.en.md)

本文定义如何把任务归类为 `trivial` 或 `non-trivial`，避免仓库只靠个人判断决定是否需要 task packet、evidence 和 review artifact。

## 1. 默认规则

默认按 `non-trivial` 处理。

只有在本策略明确允许的情况下，任务才可以被标记为 `trivial`。

## 2. 判定树

按顺序回答以下问题，只要有一个问题回答为“是”，该任务就是 `non-trivial`：

1. 是否改动代码行为、接口、schema、权限、菜单、审计、i18n、路由、UI 状态或用户流程？
2. 是否跨层，或会影响 contracts / designs / acceptances / governed docs？
3. 是否需要新增、修改或删除运行时依赖、外部服务、CI gate、发布规则或安全边界？
4. 是否需要专门验证命令、浏览器证据、review artifact 或 residual-risk 说明才能证明完成？
5. 是否存在需要 human gate 的高影响动作？

如果以上全部为“否”，再看是否满足 `trivial` 白名单。

## 3. Trivial 白名单

只有下列任务可以标记为 `trivial`：

- 单个 typo 修复
- 注释或 README 的小幅澄清，且不改变规范语义
- 只读查询、日志查看、状态汇报
- 已有 formatter 覆盖、且不改变行为的纯格式化

## 4. 输出要求

### 4.1 Trivial

`trivial` 任务至少仍需：

- 说明本次为何属于 trivial
- 记录执行过的最小验证或说明无需验证的原因
- 不得绕过 human gate

### 4.2 Non-trivial

`non-trivial` 任务必须带：

- task packet，或显式关联已批准的父级 task packet
- verification evidence
- review artifact
- known gaps / residual risk

## 5. 优先级

若本策略与仓库合同、用户直接指令冲突，以用户直接指令和仓库合同为先。
