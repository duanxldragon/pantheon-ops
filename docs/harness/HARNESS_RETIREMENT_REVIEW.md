---
title: Harness Retirement Review
doc_type: Contract
layer: platform
status: Active
updated_at: 2026-06-08
---

# Harness Retirement Review

English version: [HARNESS_RETIREMENT_REVIEW.en.md](./HARNESS_RETIREMENT_REVIEW.en.md)

本文定义 Harness 约束何时应该被降级、替换或删除，而不是无限累积。

## 1. 目标

Harness 的职责是约束当前的 probabilistic runtime，而不是永久保存每一条历史 workaround。

如果旧约束已经不再减少 failure，反而制造 friction，它就应该进入 retirement review。

## 2. 触发时机

以下任一情况都应触发 retirement review：

1. 模型、工具链或 repo shell 发生重大升级
2. 连续多个任务已经不再需要某条旧 workaround
3. 同一规则连续造成误报、误拦截或明显 adoption friction
4. 原本依赖人工提醒的约束，已经被更好的 sensor 或原生能力替代

最低 cadence：

- 每次 method release review
- 每次重大模型 / 工具升级后

## 3. 常见 retirement 候选

优先审查以下类型：

- 只为旧模型上下文限制而存在的繁琐步骤
- 已被新 checker 或新 contract 覆盖的重复文档规则
- 明显绑定某个工具实现细节、但已无共享价值的约束
- 已经连续多个任务没有实际帮助，却仍增加负担的手工 checklist

## 4. 评估问题

对每个候选规则，至少回答：

1. 它当初在防什么 failure pattern？
2. 最近一个阶段，这个 failure pattern 还真实存在吗？
3. 现在是否已有更轻、更准或更自动化的控制替代它？
4. 删除或降级后，最坏回归风险是什么？
5. 如果保留，它带来的 friction 是否已经超过收益？

## 5. 允许的结果

retirement review 的结论只能是以下之一：

- keep：保留原规则
- downgrade：从 blocking gate 降为 warning、指南或备注
- replace：用更轻或更自动化的控制替代
- remove：直接删除

任何不是 `keep` 的结论，都必须写明回滚条件。

## 6. 记录要求

retirement review 至少要记录：

- 被评估的规则或 artifact
- 结论是 keep / downgrade / replace / remove
- 原因
- 对应 change、review note、release note 或 open task

推荐更新位置：

- `HARNESS_OPEN_TASKS.md`
- release notes
- 相关 policy / contract / checker README

如果某条规则被 remove，但它对应的 failure pattern 仍然重要，应同步说明新的替代控制是什么。
